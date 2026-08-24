package workflowv3product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3observations"
	"github.com/google/uuid"
)

type PlanExplanation struct {
	Name          string            `json:"name"`
	IRDigest      string            `json:"irDigest"`
	CatalogDigest string            `json:"catalogDigest"`
	PlanDigest    string            `json:"planDigest"`
	Inputs        map[string]string `json:"inputs"`
	Nodes         []NodeExplanation `json:"nodes"`
	Outputs       map[string]string `json:"outputs"`
}

type NodeExplanation struct {
	Key            workflowv3.NodeKey   `json:"key"`
	Task           string               `json:"task"`
	ResourceClass  string               `json:"resourceClass"`
	Dependencies   []workflowv3.NodeKey `json:"dependencies"`
	MaxAttempts    int                  `json:"maxAttempts"`
	IsolationClass string               `json:"isolationClass"`
}

type StagedInput struct {
	Path      string                  `json:"path,omitempty"`
	Schema    string                  `json:"schema"`
	MediaType string                  `json:"mediaType"`
	Reference *workflowv3.ArtifactRef `json:"-"`
}

type Submission struct {
	RunID      workflowv3.RunID `json:"runId"`
	PlanDigest string           `json:"planDigest"`
	Status     string           `json:"status"`
}

type RunSummary struct {
	RunID      workflowv3.RunID `json:"runId"`
	Name       string           `json:"name"`
	PlanDigest string           `json:"planDigest"`
	Status     string           `json:"status"`
	CreatedAt  time.Time        `json:"createdAt"`
	UpdatedAt  time.Time        `json:"updatedAt"`
}

type RunView struct {
	Snapshot   workflowv3.RunSnapshot         `json:"snapshot"`
	Operations workflowv3.OperationalSnapshot `json:"operations"`
}

func (a *Application) Explain(ctx context.Context, source string) (PlanExplanation, error) {
	authored, err := a.Authoring.Author(ctx, source)
	if err != nil {
		return PlanExplanation{}, err
	}
	inputs := make(map[string]string, len(authored.Plan.Inputs)+len(authored.Plan.SetInputs))
	for _, input := range authored.Plan.Inputs {
		inputs[input.Name] = input.Schema
	}
	for _, input := range authored.Plan.SetInputs {
		inputs[input.Name] = input.ManifestSchema
	}
	outputs := make(map[string]string, len(authored.Plan.Outputs)+len(authored.Plan.SetOutputs))
	for _, output := range authored.Plan.Outputs {
		outputs[output.Name] = output.Value.Schema
	}
	for _, output := range authored.Plan.SetOutputs {
		outputs[output.Name] = output.Value.ManifestSchema
	}
	nodes := make([]NodeExplanation, 0, len(authored.Plan.Nodes))
	for _, node := range authored.Plan.Nodes {
		nodes = append(nodes, NodeExplanation{
			Key: node.Key, Task: node.Implementation.Kind + "@" + node.Implementation.Version,
			ResourceClass:  node.ResourceClass,
			Dependencies:   append([]workflowv3.NodeKey(nil), node.DependsOn...),
			MaxAttempts:    node.Retry.MaxAttempts,
			IsolationClass: workflowv3.EffectivePlanIsolation(node.Isolation).Effective.Class,
		})
	}
	return PlanExplanation{
		Name: authored.Plan.Name, IRDigest: authored.Plan.IRDigest,
		CatalogDigest: authored.Plan.CatalogDigest, PlanDigest: authored.Plan.Digest,
		Inputs: inputs, Nodes: nodes, Outputs: outputs,
	}, nil
}

func (a *Application) Submit(
	ctx context.Context,
	plan workflowv3.WorkflowPlan,
	inputs map[string]StagedInput,
	baseDir string,
	runID workflowv3.RunID,
) (Submission, error) {
	if a == nil || a.Engine == nil {
		return Submission{}, fmt.Errorf("workflow application is required")
	}
	refs, err := a.stageInputs(ctx, inputs, baseDir)
	if err != nil {
		return Submission{}, err
	}
	return a.SubmitArtifacts(ctx, plan, refs, runID)
}

// SubmitArtifacts submits inputs that already crossed an immutable artifact
// custody boundary. Callers that verify external bytes must use this method so
// a mutable path cannot be read again after verification.
func (a *Application) SubmitArtifacts(
	ctx context.Context,
	plan workflowv3.WorkflowPlan,
	inputs map[string]workflowv3.ArtifactRef,
	runID workflowv3.RunID,
) (Submission, error) {
	if a == nil || a.Engine == nil {
		return Submission{}, fmt.Errorf("workflow application is required")
	}
	if strings.TrimSpace(string(runID)) == "" {
		runID = workflowv3.RunID(uuid.NewString())
	}
	if err := a.Engine.Submit(ctx, runID, plan, inputs); err != nil {
		return Submission{}, err
	}
	snapshot, err := a.Engine.Snapshot(ctx, runID)
	if err != nil {
		return Submission{}, err
	}
	return Submission{RunID: runID, PlanDigest: plan.Digest, Status: snapshot.Status}, nil
}

func (a *Application) stageInputs(
	ctx context.Context,
	inputs map[string]StagedInput,
	baseDir string,
) (map[string]workflowv3.ArtifactRef, error) {
	refs := make(map[string]workflowv3.ArtifactRef, len(inputs))
	for name, input := range inputs {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(input.Schema) == "" {
			return nil, fmt.Errorf("staged input name and schema are required")
		}
		if input.Reference != nil {
			if strings.TrimSpace(input.Path) != "" || input.Reference.Schema != input.Schema {
				return nil, fmt.Errorf("staged input %q reference is inconsistent", name)
			}
			if err := workflowv3.ValidateArtifactRef(*input.Reference); err != nil {
				return nil, fmt.Errorf("staged input %q reference: %w", name, err)
			}
			refs[name] = *input.Reference
			continue
		}
		if strings.TrimSpace(input.Path) == "" {
			return nil, fmt.Errorf("staged input %q path is required", name)
		}
		mediaType := strings.TrimSpace(input.MediaType)
		if mediaType == "" {
			mediaType = "application/octet-stream"
		}
		path := input.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read staged input %q: %w", name, err)
		}
		ref, err := a.Artifacts.Put(ctx, input.Schema, mediaType, body)
		if err != nil {
			return nil, fmt.Errorf("stage input %q: %w", name, err)
		}
		refs[name] = ref
	}
	return refs, nil
}

func DecodeInputs(path string) (map[string]StagedInput, string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read workflow inputs: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	var inputs map[string]StagedInput
	if err := decoder.Decode(&inputs); err != nil {
		return nil, "", fmt.Errorf("decode workflow inputs: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, "", fmt.Errorf("decode workflow inputs: multiple JSON values")
		}
		return nil, "", fmt.Errorf("decode workflow inputs: %w", err)
	}
	return inputs, filepath.Dir(path), nil
}

func (a *Application) ListRuns(ctx context.Context, status string, limit int) ([]RunSummary, error) {
	rows, err := a.Store.ListRuns(ctx, status, limit)
	if err != nil {
		return nil, err
	}
	ret := make([]RunSummary, 0, len(rows))
	for _, row := range rows {
		ret = append(ret, RunSummary{
			RunID: row.RunID, Name: row.Name, PlanDigest: row.PlanDigest,
			Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
	}
	return ret, nil
}

func (a *Application) Observations(ctx context.Context, runID workflowv3.RunID) (workflowv3observations.ObservationSet, error) {
	if a == nil || a.Store == nil {
		return workflowv3observations.ObservationSet{}, fmt.Errorf("workflow application is required")
	}
	return workflowv3observations.Project(ctx, a.Store, runID, workflowv3observations.DefaultProjectOptions())
}

func (a *Application) Show(ctx context.Context, runID workflowv3.RunID) (RunView, error) {
	snapshot, err := a.Engine.Snapshot(ctx, runID)
	if err != nil {
		return RunView{}, err
	}
	operations, err := a.Dispatcher.OperationalSnapshot(ctx, &runID)
	if err != nil {
		return RunView{}, err
	}
	return RunView{Snapshot: snapshot, Operations: operations}, nil
}

func (a *Application) Cancel(ctx context.Context, runID workflowv3.RunID) (RunView, error) {
	if err := a.Store.Cancel(ctx, runID, time.Now().UTC()); err != nil {
		return RunView{}, err
	}
	return a.Show(ctx, runID)
}

func (a *Application) RunWorker(ctx context.Context) error {
	if a == nil || a.Dispatcher == nil {
		return fmt.Errorf("workflow application is required")
	}
	err := a.Dispatcher.Run(ctx)
	if errors.Is(err, context.Canceled) && ctx.Err() != nil {
		return nil
	}
	return err
}

func (a *Application) RunUntilTerminal(ctx context.Context, runID workflowv3.RunID) (RunView, error) {
	workerCtx, cancelWorker := context.WithCancel(ctx)
	defer cancelWorker()
	workerDone := make(chan error, 1)
	go func() { workerDone <- a.RunWorker(workerCtx) }()

	ticker := time.NewTicker(a.Config.PollInterval)
	defer ticker.Stop()
	for {
		view, err := a.Show(ctx, runID)
		if err != nil {
			return RunView{}, err
		}
		if terminalRunStatus(view.Snapshot.Status) {
			cancelWorker()
			if workerErr := <-workerDone; workerErr != nil {
				return RunView{}, workerErr
			}
			return view, nil
		}
		select {
		case <-ctx.Done():
			return RunView{}, ctx.Err()
		case workerErr := <-workerDone:
			if workerErr == nil {
				return RunView{}, fmt.Errorf("workflow worker stopped before run %q became terminal", runID)
			}
			return RunView{}, workerErr
		case <-ticker.C:
		}
	}
}

func (a *Application) Wait(ctx context.Context, runID workflowv3.RunID) (RunView, error) {
	ticker := time.NewTicker(a.Config.PollInterval)
	defer ticker.Stop()
	for {
		view, err := a.Show(ctx, runID)
		if err != nil {
			return RunView{}, err
		}
		if terminalRunStatus(view.Snapshot.Status) {
			return view, nil
		}
		select {
		case <-ctx.Done():
			return RunView{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func terminalRunStatus(status string) bool {
	return status == "succeeded" || status == "failed" || status == "canceled"
}
