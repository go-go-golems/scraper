package workflowv3product_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/taskpackages/cookbooklinear"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3product"
	"github.com/stretchr/testify/require"
)

func productConfig(root string) workflowv3product.Config {
	config := workflowv3product.DefaultConfig()
	config.DatabasePath = filepath.Join(root, "control", "workflow.db")
	config.ArtifactRoot = filepath.Join(root, "artifacts")
	config.PollInterval = 5 * time.Millisecond
	config.LeaseDuration = 100 * time.Millisecond
	config.Capacities = map[string]int{workflowv3.ResourceCPUDefault: 2}
	return config
}

func TestProductExecutesAuthoredWorkflowAcrossProcessRestart(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	config := productConfig(root)
	inputPath := filepath.Join(root, "customers.jsonl")
	require.NoError(t, os.WriteFile(inputPath, []byte("{\"id\":\" 2 \",\"email\":\" B@EXAMPLE.COM \"}\n{\"id\":\"1\",\"email\":\"A@example.com\"}\n"), 0o600))

	first, err := workflowv3product.Open(ctx, config)
	require.NoError(t, err)
	authored, err := first.Authoring.Author(ctx, cookbooklinear.WorkflowSource())
	require.NoError(t, err)
	explanation, err := first.Explain(ctx, cookbooklinear.WorkflowSource())
	require.NoError(t, err)
	require.Equal(t, authored.Plan.Digest, explanation.PlanDigest)
	require.Len(t, explanation.Nodes, 2)
	submission, err := first.Submit(ctx, authored.Plan, map[string]workflowv3product.StagedInput{
		"source": {Path: inputPath, Schema: "customer-jsonl-ref/v1", MediaType: "application/x-ndjson"},
	}, root, "restart-run")
	require.NoError(t, err)
	require.Equal(t, workflowv3.RunID("restart-run"), submission.RunID)
	require.NoError(t, first.Close())

	second, err := workflowv3product.Open(ctx, config)
	require.NoError(t, err)
	defer func() { _ = second.Close() }()
	view, err := second.RunUntilTerminal(ctx, "restart-run")
	require.NoError(t, err)
	require.Equal(t, "succeeded", view.Snapshot.Status)
	require.Len(t, view.Snapshot.Attempts, 2)
	output, err := workflowv3.ReadArtifact(ctx, second.Artifacts, view.Snapshot.Outputs["dataset"])
	require.NoError(t, err)
	var decoded struct {
		Rows  []map[string]string `json:"rows"`
		Count int                 `json:"count"`
	}
	require.NoError(t, json.Unmarshal(output, &decoded))
	require.Equal(t, 2, decoded.Count)
	require.Equal(t, "b@example.com", decoded.Rows[0]["email"])

	runs, err := second.ListRuns(ctx, "succeeded", 10)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, workflowv3.RunID("restart-run"), runs[0].RunID)
}

func TestProductPersistsTypedTaskFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	app, err := workflowv3product.Open(ctx, productConfig(root))
	require.NoError(t, err)
	defer func() { _ = app.Close() }()
	authored, err := app.Authoring.Author(ctx, cookbooklinear.WorkflowSource())
	require.NoError(t, err)
	inputPath := filepath.Join(root, "duplicates.jsonl")
	require.NoError(t, os.WriteFile(inputPath, []byte("{\"id\":\"1\",\"email\":\"a@example.com\"}\n{\"id\":\"1\",\"email\":\"b@example.com\"}\n"), 0o600))
	_, err = app.Submit(ctx, authored.Plan, map[string]workflowv3product.StagedInput{
		"source": {Path: inputPath, Schema: "customer-jsonl-ref/v1"},
	}, root, "failed-run")
	require.NoError(t, err)
	view, err := app.RunUntilTerminal(ctx, "failed-run")
	require.NoError(t, err)
	require.Equal(t, "failed", view.Snapshot.Status)
	require.Len(t, view.Snapshot.Attempts, 2)
	require.Equal(t, "CUSTOMER_DUPLICATE_ID", view.Snapshot.Attempts[1].Failure.Code)
	require.False(t, view.Snapshot.Attempts[1].Failure.Retryable)
}

func TestProductCancelFencesRunBeforeWorkerStart(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	app, err := workflowv3product.Open(ctx, productConfig(root))
	require.NoError(t, err)
	defer func() { _ = app.Close() }()
	authored, err := app.Authoring.Author(ctx, cookbooklinear.WorkflowSource())
	require.NoError(t, err)
	inputPath := filepath.Join(root, "customers.jsonl")
	require.NoError(t, os.WriteFile(inputPath, []byte("{\"id\":\"1\",\"email\":\"a@example.com\"}\n"), 0o600))
	_, err = app.Submit(ctx, authored.Plan, map[string]workflowv3product.StagedInput{
		"source": {Path: inputPath, Schema: "customer-jsonl-ref/v1"},
	}, root, "cancel-run")
	require.NoError(t, err)
	view, err := app.Cancel(ctx, "cancel-run")
	require.NoError(t, err)
	require.Equal(t, "canceled", view.Snapshot.Status)
	require.Empty(t, view.Snapshot.Attempts)
}

func TestProductAuthoringIsPureAndPackageGenerationIsDeterministic(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	first, err := workflowv3product.NewAuthoringEnvironment(nil)
	require.NoError(t, err)
	second, err := workflowv3product.NewAuthoringEnvironment([]string{"cookbook-linear"})
	require.NoError(t, err)
	require.Equal(t, first.Packages.Registry().Generation(), second.Packages.Registry().Generation())
	require.Len(t, first.Packages.Info(), 1)
	_, err = first.Author(ctx, `if (typeof process !== "undefined") throw new Error("process leaked"); module.exports = process;`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "process is not defined")
}

func TestProductRejectsInvalidConfigurationAndPackageSelection(t *testing.T) {
	t.Parallel()
	config := workflowv3product.DefaultConfig()
	config.Capacities = map[string]int{workflowv3.ResourceCPUDefault: 0}
	require.ErrorContains(t, config.Validate(), "capacity")
	_, err := workflowv3product.NewAuthoringEnvironment([]string{"missing"})
	require.ErrorContains(t, err, "unknown task package")
	_, err = workflowv3product.NewAuthoringEnvironment([]string{"cookbook-linear", "cookbook-linear"})
	require.ErrorContains(t, err, "selected more than once")
}

func TestDecodeInputsIsStrictAndResolvesBaseDirectory(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "inputs.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"source":{"path":"customers.jsonl","schema":"customer-jsonl-ref/v1","mediaType":"application/json"}}`), 0o600))
	inputs, base, err := workflowv3product.DecodeInputs(path)
	require.NoError(t, err)
	require.Equal(t, root, base)
	require.Equal(t, "customers.jsonl", inputs["source"].Path)

	require.NoError(t, os.WriteFile(path, []byte(`{"source":{"path":"x","schema":"s","typo":true}}`), 0o600))
	_, _, err = workflowv3product.DecodeInputs(path)
	require.ErrorContains(t, err, "unknown field")
}
