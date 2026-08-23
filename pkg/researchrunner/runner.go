package researchrunner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3observations"
	"github.com/go-go-golems/scraper/pkg/workflowv3product"
)

const defaultMaxRequestBytes int64 = 32 << 20

var artifactNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
var domainObservationNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._:-][a-z0-9]+){0,15}$`)

type Config struct {
	StateRoot             string
	ArtifactRoot          string
	TaskPackages          []string
	Capacities            map[string]int
	LeaseDuration         time.Duration
	PollInterval          time.Duration
	CancellationTimeout   time.Duration
	MaxRequestBytes       int64
	MaxResolvedInputBytes int64
	MaxExportBytes        int64
	AvailableTaskPackages []workflowv3product.TaskPackage
	DomainProjector       DomainProjector
}

func DefaultConfig() Config {
	return Config{
		StateRoot: "state/research-runner", ArtifactRoot: "state/research-runner-artifacts",
		TaskPackages:  []string{"research-runner-fixture"},
		Capacities:    map[string]int{workflowv3.ResourceCPUDefault: 2},
		LeaseDuration: 30 * time.Second, PollInterval: 25 * time.Millisecond,
		CancellationTimeout: 5 * time.Second, MaxRequestBytes: defaultMaxRequestBytes,
		MaxResolvedInputBytes: 32 << 20, MaxExportBytes: 16 << 20,
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.StateRoot) == "" || strings.TrimSpace(c.ArtifactRoot) == "" {
		return fmt.Errorf("runner state and artifact roots are required")
	}
	if len(c.TaskPackages) == 0 || len(c.Capacities) == 0 {
		return fmt.Errorf("runner task packages and capacities are required")
	}
	if c.LeaseDuration <= 0 || c.PollInterval <= 0 || c.CancellationTimeout <= 0 {
		return fmt.Errorf("runner lease, poll, and cancellation durations must be positive")
	}
	if c.MaxRequestBytes <= 0 || c.MaxResolvedInputBytes <= 0 || c.MaxExportBytes <= 0 {
		return fmt.Errorf("runner request, resolved input, and export limits must be positive")
	}
	return nil
}

type emitter struct{ encoder *json.Encoder }

func (e emitter) frame(frame Frame) error { return e.encoder.Encode(frame) }

func Run(ctx context.Context, input io.Reader, output io.Writer, config Config) error {
	if err := config.Validate(); err != nil {
		return err
	}
	request, err := decodeRequest(input, config.MaxRequestBytes)
	if err != nil {
		return err
	}
	emit := emitter{encoder: json.NewEncoder(output)}
	if err := emit.frame(Frame{Type: "hello", Hello: &Hello{
		ProtocolVersion: ProtocolVersion,
		Runner:          RunnerRecord{Name: RunnerName, ResolvedVersion: RunnerVersion},
		Domains:         []DomainVersion{{Domain: Domain, SchemaVersion: DomainSchemaVersion}},
	}}); err != nil {
		return err
	}
	if err := execute(ctx, request, config, emit); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		closed := classifyError(err)
		if emitErr := emit.frame(Frame{Type: "error", Error: &closed}); emitErr != nil {
			return emitErr
		}
		return nil
	}
	return nil
}

func decodeRequest(input io.Reader, limit int64) (Request, error) {
	decoder := json.NewDecoder(io.LimitReader(input, limit+1))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var request Request
	if err := decoder.Decode(&request); err != nil {
		return Request{}, fmt.Errorf("decode runner request: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return Request{}, fmt.Errorf("decode runner request: multiple JSON values")
		}
		return Request{}, fmt.Errorf("decode runner request trailing content: %w", err)
	}
	return request, nil
}

func execute(ctx context.Context, request Request, config Config, emit emitter) error {
	if request.ProtocolVersion != ProtocolVersion {
		return contractError("RUNNER_PROTOCOL")
	}
	identity := request.Attempt.Specification.CanonicalIdentity
	if identity.Domain != Domain || identity.DomainSchemaVersion != DomainSchemaVersion {
		return contractError("RUNNER_DOMAIN")
	}
	var execution WorkflowExecution
	if err := decodeStrict(identity.DomainConfig, &execution); err != nil {
		return contractError("RUNNER_DOMAIN_CONFIG")
	}
	if err := validateExecution(execution); err != nil {
		return err
	}
	stateKey := opaqueWorkflowID(request.Attempt.Run.ID, request.Attempt.AttemptID)
	productConfig := workflowv3product.DefaultConfig()
	productConfig.DatabasePath = filepath.Join(config.StateRoot, stateKey+".db")
	productConfig.ArtifactRoot = filepath.Join(config.ArtifactRoot, stateKey)
	productConfig.TaskPackages = append([]string(nil), config.TaskPackages...)
	productConfig.Capacities = cloneCapacities(config.Capacities)
	productConfig.LeaseDuration = config.LeaseDuration
	productConfig.PollInterval = config.PollInterval
	productConfig.MaxArtifactBytes = max(config.MaxExportBytes, config.MaxResolvedInputBytes)
	app, err := workflowv3product.Open(ctx, productConfig, config.AvailableTaskPackages...)
	if err != nil {
		return infrastructureError("RUNNER_OPEN")
	}
	defer func() { _ = app.Close() }()
	if err := validateCatalog(execution, app.Authoring.Packages); err != nil {
		return err
	}
	inputs, err := resolveInputs(ctx, execution, request.Inputs, app, config.MaxResolvedInputBytes)
	if err != nil {
		return err
	}
	workflowRunID := workflowv3.RunID(stateKey)
	submission, err := app.SubmitArtifacts(ctx, execution.Plan, inputs, workflowRunID)
	if err != nil {
		return infrastructureError("RUNNER_SUBMIT")
	}
	sequence := int64(1)
	lineage := mustJSON(map[string]any{
		"schemaVersion":      "scraper-workflow-lineage/v1",
		"specificationId":    request.Attempt.Specification.ID,
		"researchRunId":      request.Attempt.Run.ID,
		"researchAttemptId":  request.Attempt.AttemptID,
		"workflowRunId":      submission.RunID,
		"planDigest":         submission.PlanDigest,
		"registryGeneration": app.Registry.Snapshot().Active,
	})
	if err := emit.frame(Frame{Type: "event", Event: &Event{Type: "workflow.submitted", ProducerSequence: &sequence, ProducerOccurredAt: time.Now().UTC().Format(time.RFC3339Nano), Payload: lineage}}); err != nil {
		return err
	}

	workerCtx, cancelWorker := context.WithCancel(context.Background())
	workerDone := make(chan error, 1)
	go func() { workerDone <- app.RunWorker(workerCtx) }()
	view, err := waitForTerminal(ctx, app, workflowRunID, config.PollInterval)
	if err != nil {
		cancelCtx, cancel := context.WithTimeout(context.Background(), config.CancellationTimeout)
		_, _ = app.Cancel(cancelCtx, workflowRunID)
		cancel()
		cancelWorker()
		<-workerDone
		return err
	}
	cancelWorker()
	if workerErr := <-workerDone; workerErr != nil {
		return infrastructureError("RUNNER_WORKER")
	}
	return exportTerminal(ctx, app, execution, view, workflowRunID, sequence, emit, config)
}

func waitForTerminal(ctx context.Context, app *workflowv3product.Application, runID workflowv3.RunID, poll time.Duration) (workflowv3product.RunView, error) {
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	for {
		view, err := app.Show(ctx, runID)
		if err != nil {
			return workflowv3product.RunView{}, err
		}
		switch view.Snapshot.Status {
		case "succeeded", "failed", "canceled":
			return view, nil
		}
		select {
		case <-ctx.Done():
			return workflowv3product.RunView{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func BuildExecution(plan workflowv3.WorkflowPlan, packages *workflowv3product.PackageSet, bindings map[string]InputBinding, observation ObservationPolicy) (WorkflowExecution, error) {
	if packages == nil {
		return WorkflowExecution{}, fmt.Errorf("task package set is required")
	}
	catalogDigest, err := packages.Catalog().Digest()
	if err != nil {
		return WorkflowExecution{}, err
	}
	info := packages.Info()
	identities := make([]PackageIdentity, len(info))
	for index, current := range info {
		identities[index] = PackageIdentity{Name: current.Name, Version: current.Version, BundleDigest: current.BundleDigest}
	}
	execution := WorkflowExecution{
		SchemaVersion: DomainSchemaVersion, Plan: plan,
		InputBindings: bindings,
		TaskCatalog:   TaskCatalog{Digest: catalogDigest, Packages: identities},
		Observation:   observation,
	}
	if err := validateExecution(execution); err != nil {
		return WorkflowExecution{}, err
	}
	if err := validateCatalog(execution, packages); err != nil {
		return WorkflowExecution{}, err
	}
	return execution, nil
}

func validateExecution(execution WorkflowExecution) error {
	if execution.SchemaVersion != DomainSchemaVersion {
		return contractError("RUNNER_EXECUTION_SCHEMA")
	}
	planWithoutDigest := execution.Plan
	planWithoutDigest.Digest = ""
	digest, err := workflowv3.Digest(planWithoutDigest)
	if err != nil || digest != execution.Plan.Digest {
		return contractError("RUNNER_PLAN_DIGEST")
	}
	if execution.TaskCatalog.Digest != execution.Plan.CatalogDigest {
		return contractError("RUNNER_CATALOG_DIGEST")
	}
	if len(execution.InputBindings) != len(execution.Plan.Inputs)+len(execution.Plan.SetInputs) {
		return contractError("RUNNER_INPUT_BINDINGS")
	}
	inputNames := make([]string, 0, len(execution.Plan.Inputs)+len(execution.Plan.SetInputs))
	for _, input := range execution.Plan.Inputs {
		inputNames = append(inputNames, input.Name)
	}
	for _, input := range execution.Plan.SetInputs {
		inputNames = append(inputNames, input.Name)
	}
	for _, name := range inputNames {
		binding, ok := execution.InputBindings[name]
		if !ok || strings.TrimSpace(binding.Role) == "" || strings.TrimSpace(binding.Kind) == "" || strings.TrimSpace(binding.ID) == "" {
			return contractError("RUNNER_INPUT_BINDINGS")
		}
	}
	if !execution.Observation.ExportOutputs {
		return contractError("RUNNER_OUTPUT_EXPORT_REQUIRED")
	}
	if !execution.Observation.ExportCanonicalObservations {
		return contractError("RUNNER_OBSERVATIONS_REQUIRED")
	}
	return nil
}

func validateCatalog(execution WorkflowExecution, packages *workflowv3product.PackageSet) error {
	catalogDigest, err := packages.Catalog().Digest()
	if err != nil || catalogDigest != execution.TaskCatalog.Digest {
		return contractError("RUNNER_CATALOG_MISMATCH")
	}
	actual := packages.Info()
	if len(actual) != len(execution.TaskCatalog.Packages) {
		return contractError("RUNNER_PACKAGE_MISMATCH")
	}
	for index, expected := range execution.TaskCatalog.Packages {
		if index > 0 && expected.Name <= execution.TaskCatalog.Packages[index-1].Name {
			return contractError("RUNNER_PACKAGE_ORDER")
		}
		current := actual[index]
		if expected.Name != current.Name || expected.Version != current.Version || expected.BundleDigest != current.BundleDigest {
			return contractError("RUNNER_PACKAGE_MISMATCH")
		}
	}
	return nil
}

func resolveInputs(ctx context.Context, execution WorkflowExecution, resolved []ResolvedInput, app *workflowv3product.Application, maxBytes int64) (map[string]workflowv3.ArtifactRef, error) {
	bySelector := map[string]ResolvedInput{}
	for _, input := range resolved {
		key := selectorKey(input.Reference.Role, input.Reference.Kind, input.Reference.ID)
		if _, exists := bySelector[key]; exists {
			return nil, contractError("RUNNER_INPUT_DUPLICATE")
		}
		bySelector[key] = input
	}
	if len(bySelector) != len(execution.InputBindings) {
		return nil, contractError("RUNNER_INPUT_BINDINGS")
	}
	ret := make(map[string]workflowv3.ArtifactRef, len(execution.InputBindings))
	for name, binding := range execution.InputBindings {
		resolvedInput, ok := bySelector[selectorKey(binding.Role, binding.Kind, binding.ID)]
		if !ok || strings.TrimSpace(resolvedInput.Path) == "" {
			return nil, contractError("RUNNER_INPUT_MISSING")
		}
		body, err := readVerifiedInput(resolvedInput, maxBytes)
		if err != nil {
			return nil, err
		}
		if expected, ok := scalarInputSchema(execution.Plan, name); ok {
			if resolvedInput.Reference.SchemaVersion != expected {
				return nil, contractError("RUNNER_INPUT_SCHEMA")
			}
			mediaType := strings.TrimSpace(resolvedInput.Reference.MediaType)
			if mediaType == "" {
				mediaType = "application/octet-stream"
			}
			ref, err := app.Artifacts.Put(ctx, expected, mediaType, body)
			if err != nil {
				return nil, infrastructureError("RUNNER_INPUT_STAGE")
			}
			if ref.Digest != resolvedInput.Reference.Digest || ref.Size != *resolvedInput.Reference.SizeBytes {
				return nil, contractError("RUNNER_INPUT_CUSTODY")
			}
			ret[name] = ref
			continue
		}
		setInput, ok := setInputSchema(execution.Plan, name)
		if !ok || resolvedInput.Reference.SchemaVersion != SetInputArchiveSchema {
			return nil, contractError("RUNNER_INPUT_SCHEMA")
		}
		ref, err := stageSetInput(ctx, app, setInput, body)
		if err != nil {
			return nil, err
		}
		ret[name] = ref
	}
	return ret, nil
}

func scalarInputSchema(plan workflowv3.WorkflowPlan, name string) (string, bool) {
	for _, input := range plan.Inputs {
		if input.Name == name {
			return input.Schema, true
		}
	}
	return "", false
}

func setInputSchema(plan workflowv3.WorkflowPlan, name string) (workflowv3.IRSetInput, bool) {
	for _, input := range plan.SetInputs {
		if input.Name == name {
			return input, true
		}
	}
	return workflowv3.IRSetInput{}, false
}

func readVerifiedInput(input ResolvedInput, maxBytes int64) ([]byte, error) {
	if input.Reference.SizeBytes == nil || *input.Reference.SizeBytes < 0 {
		return nil, contractError("RUNNER_INPUT_SIZE")
	}
	if maxBytes <= 0 || *input.Reference.SizeBytes > maxBytes {
		return nil, contractError("RUNNER_INPUT_LIMIT")
	}
	file, err := os.Open(input.Path)
	if err != nil {
		return nil, contractError("RUNNER_INPUT_READ")
	}
	defer func() { _ = file.Close() }()
	body, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, contractError("RUNNER_INPUT_READ")
	}
	if int64(len(body)) > maxBytes {
		return nil, contractError("RUNNER_INPUT_LIMIT")
	}
	if int64(len(body)) != *input.Reference.SizeBytes {
		return nil, contractError("RUNNER_INPUT_SIZE")
	}
	sum := sha256.Sum256(body)
	if input.Reference.Digest != "sha256:"+hex.EncodeToString(sum[:]) {
		return nil, contractError("RUNNER_INPUT_DIGEST")
	}
	return body, nil
}

func stageSetInput(ctx context.Context, app *workflowv3product.Application, input workflowv3.IRSetInput, body []byte) (workflowv3.ArtifactRef, error) {
	var archive SetInputArchive
	if err := decodeStrict(body, &archive); err != nil || archive.SchemaVersion != SetInputArchiveSchema || archive.ItemSchema != input.ItemSchema || archive.ManifestSchema != input.ManifestSchema {
		return workflowv3.ArtifactRef{}, contractError("RUNNER_SET_INPUT_ARCHIVE")
	}
	if input.Policy.MaxItems < 1 || len(archive.Items) > input.Policy.MaxItems {
		return workflowv3.ArtifactRef{}, contractError("RUNNER_SET_INPUT_LIMIT")
	}
	items := make([]workflowv3.ManifestItem, 0, len(archive.Items))
	for _, item := range archive.Items {
		if strings.TrimSpace(item.Key) == "" || strings.TrimSpace(item.MediaType) == "" {
			return workflowv3.ArtifactRef{}, contractError("RUNNER_SET_INPUT_ARCHIVE")
		}
		ref, err := app.Artifacts.Put(ctx, archive.ItemSchema, item.MediaType, item.Data)
		if err != nil {
			return workflowv3.ArtifactRef{}, infrastructureError("RUNNER_SET_INPUT_STAGE")
		}
		items = append(items, workflowv3.ManifestItem{Key: item.Key, Value: ref})
	}
	manifest, err := workflowv3.NewItemManifest(archive.ItemSchema, items)
	if err != nil {
		return workflowv3.ArtifactRef{}, contractError("RUNNER_SET_INPUT_ARCHIVE")
	}
	encoded, err := workflowv3.EncodeItemManifest(manifest)
	if err != nil {
		return workflowv3.ArtifactRef{}, infrastructureError("RUNNER_SET_INPUT_STAGE")
	}
	ref, err := app.Artifacts.Put(ctx, archive.ManifestSchema, "application/json", encoded)
	if err != nil {
		return workflowv3.ArtifactRef{}, infrastructureError("RUNNER_SET_INPUT_STAGE")
	}
	return ref, nil
}

func exportTerminal(ctx context.Context, app *workflowv3product.Application, execution WorkflowExecution, view workflowv3product.RunView, runID workflowv3.RunID, sequence int64, emit emitter, config Config) error {
	sequence++
	terminalPayload := mustJSON(map[string]any{"schemaVersion": "scraper-workflow-terminal/v1", "workflowRunId": runID, "status": view.Snapshot.Status, "planDigest": view.Snapshot.PlanDigest})
	if err := emit.frame(Frame{Type: "event", Event: &Event{Type: "workflow.terminal", ProducerSequence: &sequence, ProducerOccurredAt: time.Now().UTC().Format(time.RFC3339Nano), Payload: terminalPayload}}); err != nil {
		return err
	}
	observations, err := app.Observations(ctx, runID)
	if err != nil {
		return infrastructureError("RUNNER_OBSERVATIONS")
	}
	if err := emitObservations(observations, emit, config); err != nil {
		return err
	}
	domainOutputs := make(map[string]DomainOutput, len(view.Snapshot.Outputs))
	var domainOutputBytes int64
	if execution.Observation.ExportOutputs {
		names := make([]string, 0, len(view.Snapshot.Outputs))
		for name := range view.Snapshot.Outputs {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if !artifactNamePattern.MatchString(name) {
				return contractError("RUNNER_OUTPUT_NAME")
			}
			ref := view.Snapshot.Outputs[name]
			body, err := workflowv3.ReadArtifact(ctx, app.Artifacts, ref)
			if err != nil || int64(len(body)) > config.MaxExportBytes {
				return infrastructureError("RUNNER_OUTPUT_READ")
			}
			domainOutputBytes += int64(len(body))
			if domainOutputBytes > config.MaxExportBytes {
				return infrastructureError("RUNNER_DOMAIN_OUTPUT_LIMIT")
			}
			domainOutputs[name] = DomainOutput{Name: name, SchemaVersion: ref.Schema, MediaType: ref.MediaType, Digest: ref.Digest, Data: append([]byte(nil), body...)}
			metadata := mustJSON(map[string]any{"digest": ref.Digest, "sizeBytes": ref.Size, "workflowRunId": runID, "planDigest": view.Snapshot.PlanDigest})
			if err := emit.frame(Frame{Type: "artifact", Artifact: &Artifact{Role: "workflow-output", Kind: "scraper-workflow-output", ID: name, Name: "workflow-output-" + name + ".json", SchemaVersion: ref.Schema, MediaType: ref.MediaType, Metadata: metadata, Data: body}}); err != nil {
				return err
			}
		}
	}
	if config.DomainProjector != nil && view.Snapshot.Status == "succeeded" {
		projection, err := config.DomainProjector.Project(ctx, DomainProjectionInput{WorkflowRunID: string(runID), PlanDigest: view.Snapshot.PlanDigest, Outputs: domainOutputs})
		if err != nil {
			return infrastructureError("RUNNER_DOMAIN_PROJECTION")
		}
		if err := emitDomainProjection(projection, emit, config.MaxExportBytes); err != nil {
			return err
		}
	}
	if execution.Observation.ExportExternalOperations {
		if err := exportOperations(ctx, app, runID, view.Snapshot.PlanDigest, emit, config); err != nil {
			return err
		}
	}
	finished := time.Now().UTC()
	payload := mustJSON(map[string]any{"schemaVersion": "scraper-workflow-result/v1", "workflowRunId": runID, "planDigest": view.Snapshot.PlanDigest, "status": view.Snapshot.Status, "attempts": len(view.Snapshot.Attempts), "retries": view.Operations.RetryAttempts})
	return emit.frame(Frame{Type: "complete", Complete: &Complete{Status: view.Snapshot.Status, ProducerFinishedAt: finished.Format(time.RFC3339Nano), Payload: payload}})
}

func emitDomainProjection(projection DomainProjection, emit emitter, maxBytes int64) error {
	if len(projection.Metrics) > 4096 || len(projection.Traces) > 4096 {
		return contractError("RUNNER_DOMAIN_PROJECTION_LIMIT")
	}
	var projectedBytes int64
	for _, metric := range projection.Metrics {
		projectedBytes += int64(len(metric.Value) + len(metric.Metadata))
		if projectedBytes > maxBytes {
			return contractError("RUNNER_DOMAIN_PROJECTION_LIMIT")
		}
		if !domainObservationNamePattern.MatchString(metric.Name) || strings.HasPrefix(metric.Name, "workflow.") || len(metric.Scope) > 128 || len(metric.Unit) > 64 || len(metric.Value) == 0 || int64(len(metric.Value)+len(metric.Metadata)) > maxBytes || !json.Valid(metric.Value) || (len(metric.Metadata) > 0 && !json.Valid(metric.Metadata)) || (metric.NumericProjection != nil && (math.IsNaN(*metric.NumericProjection) || math.IsInf(*metric.NumericProjection, 0))) {
			return contractError("RUNNER_DOMAIN_PROJECTION_INVALID")
		}
		current := metric
		if err := emit.frame(Frame{Type: "metric", Metric: &current}); err != nil {
			return err
		}
	}
	for _, trace := range projection.Traces {
		projectedBytes += int64(len(trace.Value))
		if projectedBytes > maxBytes {
			return contractError("RUNNER_DOMAIN_PROJECTION_LIMIT")
		}
		if !domainObservationNamePattern.MatchString(trace.Kind) || strings.HasPrefix(trace.Kind, "workflow.") || len(trace.Value) == 0 || int64(len(trace.Value)) > maxBytes || !json.Valid(trace.Value) {
			return contractError("RUNNER_DOMAIN_PROJECTION_INVALID")
		}
		current := trace
		if err := emit.frame(Frame{Type: "trace", Trace: &current}); err != nil {
			return err
		}
	}
	return nil
}

func emitObservations(observations workflowv3observations.ObservationSet, emit emitter, config Config) error {
	for _, current := range observations.Metrics {
		var numeric *float64
		var text string
		switch current.ValueKind {
		case "integer":
			var value int64
			if err := json.Unmarshal(current.Value, &value); err != nil {
				return infrastructureError("RUNNER_OBSERVATIONS")
			}
			projection := float64(value)
			numeric = &projection
		case "ratio":
			var value workflowv3observations.Ratio
			if err := json.Unmarshal(current.Value, &value); err != nil {
				return infrastructureError("RUNNER_OBSERVATIONS")
			}
			if value.Denominator > 0 {
				projection := float64(value.Numerator) / float64(value.Denominator)
				numeric = &projection
			}
		case "string":
			if err := json.Unmarshal(current.Value, &text); err != nil {
				return infrastructureError("RUNNER_OBSERVATIONS")
			}
		default:
			return infrastructureError("RUNNER_OBSERVATIONS")
		}
		metadata := mustJSON(map[string]any{
			"schemaVersion": observations.SchemaVersion, "derivationVersion": observations.DerivationVersion,
			"sourceDigest": observations.SourceDigest, "observationDigest": observations.Digest,
			"valueKind": current.ValueKind, "boundary": current.Boundary, "coverage": observations.Coverage,
		})
		if err := emit.frame(Frame{Type: "metric", Metric: &Metric{Name: current.Name, Scope: current.Scope, Value: current.Value, NumericProjection: numeric, TextProjection: text, Unit: current.Unit, Metadata: metadata}}); err != nil {
			return err
		}
	}
	for _, current := range observations.Traces {
		value := mustJSON(map[string]any{
			"schemaVersion": current.SchemaVersion, "derivationVersion": observations.DerivationVersion,
			"sourceDigest": observations.SourceDigest, "observationDigest": observations.Digest,
			"truncated": current.Truncated, "value": json.RawMessage(current.Value),
		})
		if err := emit.frame(Frame{Type: "trace", Trace: &Trace{Kind: current.Kind, Value: value}}); err != nil {
			return err
		}
	}
	body, err := workflowv3.CanonicalJSON(observations)
	if err != nil || int64(len(body)) > config.MaxExportBytes {
		return infrastructureError("RUNNER_OBSERVATIONS")
	}
	metadata := mustJSON(map[string]any{"digest": observations.Digest, "sourceDigest": observations.SourceDigest, "workflowRunId": observations.RunID, "planDigest": observations.PlanDigest})
	return emit.frame(Frame{Type: "artifact", Artifact: &Artifact{
		Role: "workflow-evidence", Kind: "scraper-workflow-observations", Name: "workflow-observations.json",
		SchemaVersion: observations.SchemaVersion, MediaType: "application/json", Metadata: metadata, Data: append(body, '\n'),
	}})
}

func exportOperations(ctx context.Context, app *workflowv3product.Application, runID workflowv3.RunID, planDigest string, emit emitter, config Config) error {
	directory, err := os.MkdirTemp("", "research-runner-operations-*")
	if err != nil {
		return infrastructureError("RUNNER_OPERATION_EXPORT")
	}
	defer func() { _ = os.RemoveAll(directory) }()
	jsonlPath, manifestPath := filepath.Join(directory, "operations.jsonl"), filepath.Join(directory, "manifest.json")
	manifest, err := app.Store.ExportExternalOperations(ctx, runID, jsonlPath, manifestPath)
	if err != nil {
		return infrastructureError("RUNNER_OPERATION_EXPORT")
	}
	for _, current := range []struct{ name, role, kind, schema, media string }{
		{"workflow-external-operations.jsonl", "workflow-evidence", "scraper-workflow-external-operations", workflowv3.ExternalOperationExportSchema, "application/x-ndjson"},
		{"workflow-external-operations-manifest.json", "workflow-evidence", "scraper-workflow-external-operation-manifest", workflowv3.ExternalOperationExportSchema, "application/json"},
	} {
		path := jsonlPath
		if strings.Contains(current.name, "manifest") {
			path = manifestPath
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil || int64(len(body)) > config.MaxExportBytes {
			return infrastructureError("RUNNER_OPERATION_EXPORT")
		}
		metadata := mustJSON(map[string]any{"workflowRunId": runID, "planDigest": planDigest, "recordCount": manifest.RecordCount, "digest": manifest.JSONLDigest})
		if err := emit.frame(Frame{Type: "artifact", Artifact: &Artifact{Role: current.role, Kind: current.kind, Name: current.name, SchemaVersion: current.schema, MediaType: current.media, Metadata: metadata, Data: body}}); err != nil {
			return err
		}
	}
	return nil
}

func decodeStrict(body []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("trailing JSON content")
	}
	return nil
}

func selectorKey(role, kind, id string) string { return role + "\x00" + kind + "\x00" + id }
func opaqueWorkflowID(runID, attemptID string) string {
	sum := sha256.Sum256([]byte(runID + "\x00" + attemptID))
	return "rwf-" + hex.EncodeToString(sum[:12])
}
func cloneCapacities(input map[string]int) map[string]int {
	output := make(map[string]int, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
func mustJSON(value any) json.RawMessage {
	body, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return body
}

type classifiedError struct {
	code      string
	retryable bool
}

func (e classifiedError) Error() string     { return e.code }
func contractError(code string) error       { return classifiedError{code: code} }
func infrastructureError(code string) error { return classifiedError{code: code, retryable: true} }
func classifyError(err error) RunnerError {
	var classified classifiedError
	if errors.As(err, &classified) {
		return RunnerError{Code: classified.code, Message: "runner rejected execution", Retryable: classified.retryable}
	}
	return RunnerError{Code: "RUNNER_INTERNAL", Message: "runner execution failed", Retryable: true}
}
