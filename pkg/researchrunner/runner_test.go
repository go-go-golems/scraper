package researchrunner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/taskpackages/researchfixture"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3observations"
	"github.com/go-go-golems/scraper/pkg/workflowv3product"
	"github.com/go-go-golems/scraper/pkg/workflowv3sqlite"
	"github.com/stretchr/testify/require"
)

func runnerRequest(t *testing.T, root string, input []byte) (Request, Config) {
	t.Helper()
	ctx := context.Background()
	environment, err := workflowv3product.NewAuthoringEnvironment([]string{researchfixture.Name})
	require.NoError(t, err)
	authored, err := environment.Author(ctx, researchfixture.WorkflowSource())
	require.NoError(t, err)
	catalogDigest, err := environment.Packages.Catalog().Digest()
	require.NoError(t, err)
	info := environment.Packages.Info()
	require.Len(t, info, 1)
	inputPath := filepath.Join(root, "input.json")
	require.NoError(t, os.WriteFile(inputPath, input, 0o600))
	sum := sha256.Sum256(input)
	size := int64(len(input))
	reference := ArtifactRef{
		Role: "workflow-input", Kind: "fixture-source", ID: "source",
		Digest: "sha256:" + hex.EncodeToString(sum[:]), SizeBytes: &size,
		SchemaVersion: "fixture-source/v1", MediaType: "application/json", URI: inputPath,
	}
	execution := WorkflowExecution{
		SchemaVersion: DomainSchemaVersion, Plan: authored.Plan,
		InputBindings: map[string]InputBinding{"source": {Role: reference.Role, Kind: reference.Kind, ID: reference.ID}},
		TaskCatalog:   TaskCatalog{Digest: catalogDigest, Packages: []PackageIdentity{{Name: info[0].Name, Version: info[0].Version, BundleDigest: info[0].BundleDigest}}},
		Observation:   ObservationPolicy{ExportOutputs: true, ExportExternalOperations: true, ExportCanonicalObservations: true},
	}
	domainConfig, err := json.Marshal(execution)
	require.NoError(t, err)
	request := Request{
		ProtocolVersion: ProtocolVersion,
		Attempt: Attempt{
			Specification: Specification{ID: "spec-1", IdentityScheme: "researchctl-execution-identity/v1", CanonicalIdentity: ExecutionIdentity{
				SchemaVersion: "researchctl-execution-spec/v1", IdentityScheme: "researchctl-execution-identity/v1",
				Domain: Domain, DomainSchemaVersion: DomainSchemaVersion, Inputs: []ArtifactRef{reference},
				DomainConfig: domainConfig, RequestedMeasures: []json.RawMessage{},
			}, DisplayName: "fixture"},
			Run:       ResearchRun{ID: "research-run-1", ReplicateIndex: 1, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)},
			AttemptID: "research-attempt-1", AttemptIndex: 1,
		},
		Inputs: []ResolvedInput{{Reference: reference, Path: inputPath}},
	}
	config := DefaultConfig()
	config.StateRoot = filepath.Join(root, "state")
	config.ArtifactRoot = filepath.Join(root, "artifacts")
	config.PollInterval = time.Millisecond
	config.LeaseDuration = 5 * time.Second
	return request, config
}

func runRequest(t *testing.T, ctx context.Context, request Request, config Config) ([]Frame, error) {
	t.Helper()
	payload, err := json.Marshal(request)
	require.NoError(t, err)
	var output bytes.Buffer
	runErr := Run(ctx, bytes.NewReader(payload), &output, config)
	decoder := json.NewDecoder(&output)
	frames := []Frame{}
	for {
		var frame Frame
		err := decoder.Decode(&frame)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		frames = append(frames, frame)
	}
	return frames, runErr
}

func TestRunnerExportsRetryFailedOperationAndVerifiedOutput(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	request, config := runnerRequest(t, root, []byte(`{"value":"cross repository"}`))
	frames, err := runRequest(t, context.Background(), request, config)
	require.NoError(t, err)
	require.NotEmpty(t, frames)
	require.Equal(t, "hello", frames[0].Type)
	require.Equal(t, RunnerName, frames[0].Hello.Runner.Name)
	require.Equal(t, "complete", frames[len(frames)-1].Type)
	require.Equal(t, "succeeded", frames[len(frames)-1].Complete.Status)

	metrics := map[string]float64{}
	metricMetadata := map[string]json.RawMessage{}
	artifacts := map[string]Artifact{}
	var failureTrace Trace
	for _, frame := range frames {
		if frame.Metric != nil {
			metricMetadata[frame.Metric.Name] = frame.Metric.Metadata
			if frame.Metric.NumericProjection != nil {
				metrics[frame.Metric.Name] = *frame.Metric.NumericProjection
			}
		}
		if frame.Artifact != nil {
			artifacts[frame.Artifact.Name] = *frame.Artifact
		}
		if frame.Trace != nil && frame.Trace.Kind == "workflow.failures" {
			failureTrace = *frame.Trace
		}
	}
	require.Equal(t, float64(1), metrics["workflow.retries"])
	require.Equal(t, float64(2), metrics["workflow.external_operations.admitted"])
	require.Equal(t, float64(1), metrics["workflow.external_operations.failed"])
	require.Equal(t, float64(1), metrics["workflow.external_operations.succeeded"])
	require.JSONEq(t, `{"published":true,"value":"CROSS REPOSITORY"}`, string(artifacts["workflow-output-result.json"].Data))
	require.Len(t, strings.Split(strings.TrimSpace(string(artifacts["workflow-external-operations.jsonl"].Data)), "\n"), 2)
	require.NotContains(t, string(failureTrace.Value), "fixture operation requested retry")
	require.Contains(t, string(failureTrace.Value), "FIXTURE_OPERATION_TRANSIENT")
	require.Equal(t, "scraper-workflow-observations/v1", artifacts["workflow-observations.json"].SchemaVersion)
	observations, err := workflowv3observations.Decode(artifacts["workflow-observations.json"].Data)
	require.NoError(t, err)
	var retryMetadata map[string]any
	require.NoError(t, json.Unmarshal(metricMetadata["workflow.retries"], &retryMetadata))
	require.Equal(t, observations.SourceDigest, retryMetadata["sourceDigest"])
	require.Equal(t, observations.Digest, retryMetadata["observationDigest"])
}

func TestRunnerMapsPermanentTaskFailureWithoutLeakingInput(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	secret := "SECRET-CANARY-IN-BAD-JSON"
	request, config := runnerRequest(t, root, []byte(`{"value":"`+secret+`"`))
	frames, err := runRequest(t, context.Background(), request, config)
	require.NoError(t, err)
	require.Equal(t, "complete", frames[len(frames)-1].Type)
	require.Equal(t, "failed", frames[len(frames)-1].Complete.Status)
	encoded, err := json.Marshal(frames)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), secret)
}

func TestRunnerRejectsCatalogAndInputDigestMismatchWithClosedErrors(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	request, config := runnerRequest(t, root, []byte(`{"value":"fixture"}`))
	var execution WorkflowExecution
	require.NoError(t, json.Unmarshal(request.Attempt.Specification.CanonicalIdentity.DomainConfig, &execution))
	execution.TaskCatalog.Packages[0].BundleDigest = "sha256:" + strings.Repeat("0", 64)
	request.Attempt.Specification.CanonicalIdentity.DomainConfig = mustJSON(execution)
	frames, err := runRequest(t, context.Background(), request, config)
	require.NoError(t, err)
	require.Equal(t, "error", frames[len(frames)-1].Type)
	require.Equal(t, "RUNNER_PACKAGE_MISMATCH", frames[len(frames)-1].Error.Code)
	require.NotContains(t, frames[len(frames)-1].Error.Message, root)

	request, config = runnerRequest(t, t.TempDir(), []byte(`{"value":"fixture"}`))
	request.Inputs[0].Reference.Digest = "sha256:" + strings.Repeat("f", 64)
	frames, err = runRequest(t, context.Background(), request, config)
	require.NoError(t, err)
	require.Equal(t, "RUNNER_INPUT_DIGEST", frames[len(frames)-1].Error.Code)
}

func TestResolveInputsStagesExactlyTheVerifiedScalarBytes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	original := []byte(`{"value":"verified"}`)
	request, config := runnerRequest(t, root, original)
	var execution WorkflowExecution
	require.NoError(t, json.Unmarshal(request.Attempt.Specification.CanonicalIdentity.DomainConfig, &execution))

	productConfig := workflowv3product.DefaultConfig()
	productConfig.DatabasePath = filepath.Join(root, "custody.db")
	productConfig.ArtifactRoot = filepath.Join(root, "custody-artifacts")
	productConfig.TaskPackages = []string{researchfixture.Name}
	productConfig.MaxArtifactBytes = config.MaxResolvedInputBytes
	app, err := workflowv3product.Open(ctx, productConfig)
	require.NoError(t, err)
	defer func() { require.NoError(t, app.Close()) }()

	refs, err := resolveInputs(ctx, execution, request.Inputs, app, config.MaxResolvedInputBytes)
	require.NoError(t, err)
	ref := refs["source"]
	require.Equal(t, request.Inputs[0].Reference.Digest, ref.Digest)

	require.NoError(t, os.WriteFile(request.Inputs[0].Path, []byte(`{"value":"replaced"}`), 0o600))
	staged, err := workflowv3.ReadArtifact(ctx, app.Artifacts, ref)
	require.NoError(t, err)
	require.Equal(t, original, staged)
}

func TestReadVerifiedInputEnforcesResolvedInputLimit(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	request, _ := runnerRequest(t, root, []byte(`{"value":"too-large"}`))
	_, err := readVerifiedInput(request.Inputs[0], *request.Inputs[0].Reference.SizeBytes-1)
	require.ErrorContains(t, err, "RUNNER_INPUT_LIMIT")
}

func TestRunnerCompletesPassThroughWithoutScheduledWork(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	input := []byte(`{"value":"pass-through"}`)
	request, config := runnerRequest(t, root, input)
	var execution WorkflowExecution
	require.NoError(t, json.Unmarshal(request.Attempt.Specification.CanonicalIdentity.DomainConfig, &execution))
	environment, err := workflowv3product.NewAuthoringEnvironment([]string{researchfixture.Name})
	require.NoError(t, err)
	execution.Plan, err = workflowv3.Compile(workflowv3.WorkflowIR{
		Schema: workflowv3.IRSchema, Name: "runner-pass-through",
		Inputs: []workflowv3.IRInput{{Name: "source", Schema: "fixture-source/v1"}},
		Outputs: []workflowv3.IROutput{{
			Name: "source", Value: workflowv3.ValueRef{Source: "input", Name: "source", Schema: "fixture-source/v1"},
		}},
	}, environment.Packages.Catalog())
	require.NoError(t, err)
	execution.TaskCatalog.Digest = execution.Plan.CatalogDigest
	request.Attempt.Specification.CanonicalIdentity.DomainConfig = mustJSON(execution)

	frames, err := runRequest(t, context.Background(), request, config)
	require.NoError(t, err)
	require.Equal(t, "complete", frames[len(frames)-1].Type)
	require.Equal(t, "succeeded", frames[len(frames)-1].Complete.Status)
	var exported []byte
	for _, frame := range frames {
		if frame.Artifact != nil && frame.Artifact.Name == "workflow-output-source.json" {
			exported = frame.Artifact.Data
		}
	}
	require.Equal(t, input, exported)
}

func TestRunnerRequiresCanonicalObservationProjection(t *testing.T) {
	t.Parallel()
	request, config := runnerRequest(t, t.TempDir(), []byte(`{"value":"fixture"}`))
	var execution WorkflowExecution
	require.NoError(t, json.Unmarshal(request.Attempt.Specification.CanonicalIdentity.DomainConfig, &execution))
	execution.Observation.ExportCanonicalObservations = false
	request.Attempt.Specification.CanonicalIdentity.DomainConfig = mustJSON(execution)
	frames, err := runRequest(t, context.Background(), request, config)
	require.NoError(t, err)
	require.Equal(t, "RUNNER_OBSERVATIONS_REQUIRED", frames[len(frames)-1].Error.Code)
}

func TestDomainProjectionRejectsWorkflowNamesAndNonfiniteValues(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	emit := emitter{encoder: json.NewEncoder(&output)}
	value := 1.0
	require.NoError(t, emitDomainProjection(DomainProjection{Metrics: []Metric{{Name: "domain.score", Scope: "domain.item.q1", Value: json.RawMessage(`1`), NumericProjection: &value, Unit: "ratio", Metadata: json.RawMessage(`{}`)}}}, emit, 1024))
	require.Contains(t, output.String(), `"name":"domain.score"`)
	require.Error(t, emitDomainProjection(DomainProjection{Metrics: []Metric{{Name: "workflow.elapsed", Value: json.RawMessage(`1`)}}}, emit, 1024))
	notFinite := math.Inf(1)
	require.Error(t, emitDomainProjection(DomainProjection{Metrics: []Metric{{Name: "domain.score", Value: json.RawMessage(`1`), NumericProjection: &notFinite}}}, emit, 1024))
	require.ErrorContains(t, emitDomainProjection(DomainProjection{Traces: []Trace{{Kind: "domain.item", Value: json.RawMessage(`{"value":"1234"}`)}, {Kind: "domain.item", Value: json.RawMessage(`{"value":"5678"}`)}}}, emit, 20), "RUNNER_DOMAIN_PROJECTION_LIMIT")
}

func TestRunnerStagesSetInputFromExplicitPolicy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	config := workflowv3product.DefaultConfig()
	config.DatabasePath = filepath.Join(root, "workflow.db")
	config.ArtifactRoot = filepath.Join(root, "artifacts")
	config.TaskPackages = []string{researchfixture.Name}
	app, err := workflowv3product.Open(ctx, config)
	require.NoError(t, err)
	defer func() { _ = app.Close() }()
	archive := SetInputArchive{SchemaVersion: SetInputArchiveSchema, ItemSchema: "query/v1", ManifestSchema: workflowv3.ItemManifestSchemaV1, Items: []SetInputArchiveItem{{Key: "q1", MediaType: "application/json", Data: []byte(`{"id":"q1"}`)}, {Key: "q2", MediaType: "application/json", Data: []byte(`{"id":"q2"}`)}}}
	body := mustJSON(archive)
	plan := workflowv3.WorkflowPlan{SetInputs: []workflowv3.IRSetInput{{Name: "queries", ItemSchema: "query/v1", ManifestSchema: workflowv3.ItemManifestSchemaV1, Policy: workflowv3.SetInputPolicy{MaxItems: 2}}}}
	ref, err := stageSetInput(ctx, app, plan.SetInputs[0], body)
	require.NoError(t, err)
	manifestBody, err := workflowv3.ReadArtifact(ctx, app.Artifacts, ref)
	require.NoError(t, err)
	manifest, err := workflowv3.DecodeItemManifest(manifestBody)
	require.NoError(t, err)
	require.Len(t, manifest.Items, 2)
	itemBody, err := workflowv3.ReadArtifact(ctx, app.Artifacts, manifest.Items[0].Value)
	require.NoError(t, err)
	require.JSONEq(t, `{"id":"q1"}`, string(itemBody))
	archive.Items = append(archive.Items, SetInputArchiveItem{Key: "q3", MediaType: "application/json", Data: []byte(`{}`)})
	_, err = stageSetInput(ctx, app, plan.SetInputs[0], mustJSON(archive))
	require.ErrorContains(t, err, "RUNNER_SET_INPUT_LIMIT")
}

func TestRunnerCancellationFencesDurableWorkflow(t *testing.T) {
	root := t.TempDir()
	request, config := runnerRequest(t, root, []byte(`{"value":"cancel"}`))
	payload, err := json.Marshal(request)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	reader, writer := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, bytes.NewReader(payload), writer, config)
		_ = writer.Close()
	}()
	decoder := json.NewDecoder(reader)
	var hello, submitted Frame
	require.NoError(t, decoder.Decode(&hello))
	require.NoError(t, decoder.Decode(&submitted))
	require.Equal(t, "workflow.submitted", submitted.Event.Type)
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	require.NoError(t, reader.Close())

	stateKey := opaqueWorkflowID(request.Attempt.Run.ID, request.Attempt.AttemptID)
	store, err := workflowv3sqlite.Open(context.Background(), filepath.Join(config.StateRoot, stateKey+".db"))
	require.NoError(t, err)
	defer func() { _ = store.Close() }()
	snapshot, err := store.Snapshot(context.Background(), workflowv3.RunID(stateKey))
	require.NoError(t, err)
	require.Equal(t, "canceled", snapshot.Status)
}

func TestRunnerStrictlyRejectsUnknownRequestFieldBeforeHello(t *testing.T) {
	t.Parallel()
	config := DefaultConfig()
	var output bytes.Buffer
	err := Run(context.Background(), strings.NewReader(`{"protocolVersion":"researchctl-runner-stdio/v1","attempt":{},"inputs":[],"unknown":true}`), &output, config)
	require.ErrorContains(t, err, "unknown field")
	require.Empty(t, output.String())
}
