package workflowv3product_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/taskpackages/cookbooklinear"
	"github.com/go-go-golems/scraper/pkg/taskpackages/researchfixture"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3observations"
	"github.com/go-go-golems/scraper/pkg/workflowv3product"
	"github.com/stretchr/testify/require"
)

func productConfig(root string) workflowv3product.Config {
	config := workflowv3product.DefaultConfig()
	config.DatabasePath = filepath.Join(root, "control", "workflow.db")
	config.ArtifactRoot = filepath.Join(root, "artifacts")
	config.PollInterval = 5 * time.Millisecond
	config.LeaseDuration = 5 * time.Second
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

func TestRunUntilTerminalReturnsWorkerFailureBeforeContextDeadline(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	app, err := workflowv3product.Open(ctx, productConfig(root))
	require.NoError(t, err)
	defer func() { require.NoError(t, app.Close()) }()
	authored, err := app.Authoring.Author(ctx, cookbooklinear.WorkflowSource())
	require.NoError(t, err)
	inputPath := filepath.Join(root, "customers.jsonl")
	require.NoError(t, os.WriteFile(inputPath, []byte("{\"id\":\"1\",\"email\":\"a@example.com\"}\n"), 0o600))
	_, err = app.Submit(ctx, authored.Plan, map[string]workflowv3product.StagedInput{
		"source": {Path: inputPath, Schema: "customer-jsonl-ref/v1"},
	}, root, "worker-failure")
	require.NoError(t, err)
	app.Dispatcher.Engine = nil
	waitCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	_, err = app.RunUntilTerminal(waitCtx, "worker-failure")
	require.ErrorContains(t, err, "dispatcher requires an engine")
	require.NotErrorIs(t, err, context.DeadlineExceeded)
}

func TestProductImmediatelyCompletesScalarPassThrough(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	app, err := workflowv3product.Open(ctx, productConfig(t.TempDir()))
	require.NoError(t, err)
	defer func() { require.NoError(t, app.Close()) }()
	catalog := app.Authoring.Packages.Catalog()
	plan, err := workflowv3.Compile(workflowv3.WorkflowIR{
		Schema: workflowv3.IRSchema, Name: "scalar-pass-through",
		Inputs: []workflowv3.IRInput{{Name: "source", Schema: "source/v1"}},
		Outputs: []workflowv3.IROutput{{
			Name: "source", Value: workflowv3.ValueRef{Source: "input", Name: "source", Schema: "source/v1"},
		}},
	}, catalog)
	require.NoError(t, err)
	ref, err := app.Artifacts.Put(ctx, "source/v1", "application/json", []byte(`{"value":"pass-through"}`))
	require.NoError(t, err)
	submission, err := app.SubmitArtifacts(ctx, plan, map[string]workflowv3.ArtifactRef{"source": ref}, "pass-through")
	require.NoError(t, err)
	require.Equal(t, "succeeded", submission.Status)
	view, err := app.Show(ctx, "pass-through")
	require.NoError(t, err)
	require.Equal(t, "succeeded", view.Snapshot.Status)
	require.Equal(t, ref, view.Snapshot.Outputs["source"])
	require.Empty(t, view.Snapshot.Attempts)
}

func TestProductResearchFixtureRetainsRetryAndFailedOperation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	config := productConfig(root)
	config.TaskPackages = []string{researchfixture.Name}
	app, err := workflowv3product.Open(ctx, config)
	require.NoError(t, err)
	defer func() { _ = app.Close() }()
	authored, err := app.Authoring.Author(ctx, researchfixture.WorkflowSource())
	require.NoError(t, err)
	inputPath := filepath.Join(root, "source.json")
	require.NoError(t, os.WriteFile(inputPath, []byte(`{"value":"runner fixture"}`), 0o600))
	_, err = app.Submit(ctx, authored.Plan, map[string]workflowv3product.StagedInput{
		"source": {Path: inputPath, Schema: "fixture-source/v1", MediaType: "application/json"},
	}, root, "research-fixture")
	require.NoError(t, err)
	view, err := app.RunUntilTerminal(ctx, "research-fixture")
	require.NoError(t, err)
	require.Equal(t, "succeeded", view.Snapshot.Status)
	require.Len(t, view.Snapshot.Attempts, 3)
	var transientFailures int
	for _, attempt := range view.Snapshot.Attempts {
		if attempt.Failure != nil && attempt.Failure.Code == "FIXTURE_OPERATION_TRANSIENT" {
			transientFailures++
		}
	}
	require.Equal(t, 1, transientFailures)
	require.Equal(t, 1, view.Operations.RetryAttempts)
	require.NotNil(t, view.Operations.ExternalOperations)
	require.Equal(t, 2, view.Operations.ExternalOperations.Admitted)
	require.Equal(t, 1, view.Operations.ExternalOperations.Outcomes[workflowv3.ExternalOperationOutcomeFailed])
	require.Equal(t, 1, view.Operations.ExternalOperations.Outcomes[workflowv3.ExternalOperationOutcomeSucceeded])
	body, err := workflowv3.ReadArtifact(ctx, app.Artifacts, view.Snapshot.Outputs["result"])
	require.NoError(t, err)
	require.JSONEq(t, `{"published":true,"value":"RUNNER FIXTURE"}`, string(body))
	observations, err := app.Observations(ctx, "research-fixture")
	require.NoError(t, err)
	require.Equal(t, "scraper-workflow-observations/v1", observations.SchemaVersion)
	require.Equal(t, 1.0, observationNumeric(t, observations.Metrics, "workflow.retries"))
	require.Equal(t, 1.0, observationNumeric(t, observations.Metrics, "workflow.external_operations.failed"))
	require.Len(t, observations.ArtifactLineage, 1)
	require.NoError(t, app.Close())
	reopened, err := workflowv3product.Open(ctx, config)
	require.NoError(t, err)
	defer func() { _ = reopened.Close() }()
	afterRestart, err := reopened.Observations(ctx, "research-fixture")
	require.NoError(t, err)
	require.Equal(t, observations, afterRestart)
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
	observations, err := app.Observations(ctx, "failed-run")
	require.NoError(t, err)
	require.Equal(t, "failed", observations.RunStatus)
	require.Equal(t, 1.0, observationNumeric(t, observations.Metrics, "workflow.failed_job_attempts"))
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
	observations, err := app.Observations(ctx, "cancel-run")
	require.NoError(t, err)
	require.Equal(t, "canceled", observations.RunStatus)
	require.Equal(t, 0.0, observationNumeric(t, observations.Metrics, "workflow.job_attempts"))
}

func observationNumeric(t *testing.T, metrics []workflowv3observations.Metric, name string) float64 {
	t.Helper()
	for _, metric := range metrics {
		if metric.Name != name {
			continue
		}
		var value float64
		require.NoError(t, json.Unmarshal(metric.Value, &value))
		return value
	}
	t.Fatalf("observation metric %s not found", name)
	return 0
}

func TestProductAuthoringIsPureAndPackageGenerationIsDeterministic(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	first, err := workflowv3product.NewAuthoringEnvironment(nil)
	require.NoError(t, err)
	second, err := workflowv3product.NewAuthoringEnvironment([]string{"cookbook-linear", "research-runner-fixture"})
	require.NoError(t, err)
	require.Equal(t, first.Packages.Registry().Generation(), second.Packages.Registry().Generation())
	require.Len(t, first.Packages.Info(), 2)
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
