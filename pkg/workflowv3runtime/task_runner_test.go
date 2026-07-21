package workflowv3runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/go-go-golems/scraper/pkg/testfixtures/workflowv3linear"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/stretchr/testify/require"
)

func sealedLinearRegistry(t *testing.T) *workflowv3.SealedRegistry {
	t.Helper()
	registry, err := workflowv3linear.Registry()
	require.NoError(t, err)
	return registry
}

func taskByKind(t *testing.T, registry *workflowv3.SealedRegistry, kind string) workflowv3.RegisteredTask {
	t.Helper()
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	spec, ok := catalog.Lookup(workflowv3.TaskKey{Kind: kind, Version: "v1"})
	require.True(t, ok)
	registered, err := registry.Resolve(spec.Identity)
	require.NoError(t, err)
	return registered
}

func TestRunTaskExecutesRealFileTransformInFreshRuntimes(t *testing.T) {
	ctx := context.Background()
	artifacts, err := workflowv3.NewFileArtifactStore(t.TempDir(), 1<<20)
	require.NoError(t, err)
	registry := sealedLinearRegistry(t)

	source, err := artifacts.Put(
		ctx,
		"customer-jsonl-ref/v1",
		"application/x-ndjson",
		[]byte("{\"id\":\" 2 \",\"email\":\" B@EXAMPLE.COM \"}\n"+
			"{\"id\":\"1\",\"email\":\"A@Example.com\"}\n"),
	)
	require.NoError(t, err)

	normalized, err := RunTask(ctx, TaskRequest{
		RunID: "run-1", NodeKey: "normalize", Attempt: 1,
		Task:   taskByKind(t, registry, "cookbook.linear.normalize-customers"),
		Inputs: map[string]workflowv3.ArtifactRef{"source": source}, Artifacts: artifacts,
	})
	require.NoError(t, err)

	validated, err := RunTask(ctx, TaskRequest{
		RunID: "run-1", NodeKey: "validate", Attempt: 1,
		Task:      taskByKind(t, registry, "cookbook.linear.validate-dataset"),
		Inputs:    map[string]workflowv3.ArtifactRef{"dataset": normalized.Outputs["dataset"]},
		Artifacts: artifacts,
	})
	require.NoError(t, err)
	body, err := workflowv3.ReadArtifact(ctx, artifacts, validated.Outputs["validatedDataset"])
	require.NoError(t, err)
	require.JSONEq(t, `{
	  "rows":[
	    {"id":"2","email":"b@example.com"},
	    {"id":"1","email":"a@example.com"}
	  ],
	  "count":2
	}`, string(body))
}

func TestRunTaskPreservesTypedFailure(t *testing.T) {
	ctx := context.Background()
	artifacts, err := workflowv3.NewFileArtifactStore(t.TempDir(), 1<<20)
	require.NoError(t, err)
	registry := sealedLinearRegistry(t)
	source, err := artifacts.Put(
		ctx,
		"customer-jsonl-ref/v1",
		"application/x-ndjson",
		[]byte("{\"id\":\"1\",\"email\":\"a@example.com\"}\n"+
			"{\"id\":\"1\",\"email\":\"b@example.com\"}\n"),
	)
	require.NoError(t, err)
	normalized, err := RunTask(ctx, TaskRequest{
		RunID: "run", NodeKey: "normalize", Attempt: 1,
		Task:   taskByKind(t, registry, "cookbook.linear.normalize-customers"),
		Inputs: map[string]workflowv3.ArtifactRef{"source": source}, Artifacts: artifacts,
	})
	require.NoError(t, err)
	_, err = RunTask(ctx, TaskRequest{
		RunID: "run", NodeKey: "validate", Attempt: 1,
		Task:      taskByKind(t, registry, "cookbook.linear.validate-dataset"),
		Inputs:    map[string]workflowv3.ArtifactRef{"dataset": normalized.Outputs["dataset"]},
		Artifacts: artifacts,
	})
	var failure *TaskFailureError
	require.True(t, errors.As(err, &failure))
	require.Equal(t, "validation", failure.Failure.Class)
	require.Equal(t, "CUSTOMER_DUPLICATE_ID", failure.Failure.Code)
	require.False(t, failure.Failure.Retryable)
}

func TestRunTaskRejectsWrongOutputSchema(t *testing.T) {
	ctx := context.Background()
	base, err := workflowv3linear.Bundle()
	require.NoError(t, err)
	badSource := []byte(`
const task = require("workflow/task");
exports.normalizeCustomers = task.implementation(async ctx => {
  const dataset = await ctx.outputs.putJSON("dataset", {
    schema: "wrong/v1",
    value: [],
  });
  return task.success({dataset});
});
exports.validateDataset = task.implementation(async ctx => {
  return task.success({});
});`)
	bundle, err := workflowv3.NewBundle(
		base.Manifest(),
		map[string][]byte{"execution/tasks.cjs": badSource},
	)
	require.NoError(t, err)
	builder := workflowv3.NewRegistryBuilder()
	require.NoError(t, builder.AddBundle(bundle))
	registry, err := builder.Seal()
	require.NoError(t, err)
	artifacts, err := workflowv3.NewFileArtifactStore(t.TempDir(), 1024)
	require.NoError(t, err)
	ref, err := artifacts.Put(
		ctx,
		"customer-jsonl-ref/v1",
		"application/x-ndjson",
		[]byte("{\"id\":\"1\",\"email\":\"a@example.com\"}\n"),
	)
	require.NoError(t, err)
	_, err = RunTask(ctx, TaskRequest{
		RunID: "run", NodeKey: "normalize", Attempt: 1,
		Task:   taskByKind(t, registry, "cookbook.linear.normalize-customers"),
		Inputs: map[string]workflowv3.ArtifactRef{"source": ref}, Artifacts: artifacts,
	})
	require.ErrorContains(t, err, "output dataset schema wrong/v1 does not match")
}

func TestRunTaskRejectsUnsupportedModuleProfile(t *testing.T) {
	ctx := context.Background()
	base, err := workflowv3linear.Bundle()
	require.NoError(t, err)
	manifest := base.Manifest()
	manifest.Tasks[0].Modules = []string{"db:ambient"}
	source, ok := base.File("execution/tasks.cjs")
	require.True(t, ok)
	bundle, err := workflowv3.NewBundle(
		manifest,
		map[string][]byte{"execution/tasks.cjs": source},
	)
	require.NoError(t, err)
	builder := workflowv3.NewRegistryBuilder()
	require.NoError(t, builder.AddBundle(bundle))
	registry, err := builder.Seal()
	require.NoError(t, err)
	artifacts, err := workflowv3.NewFileArtifactStore(t.TempDir(), 1024)
	require.NoError(t, err)
	ref, err := artifacts.Put(
		ctx,
		"customer-jsonl-ref/v1",
		"application/x-ndjson",
		[]byte("{\"id\":\"1\",\"email\":\"a@example.com\"}\n"),
	)
	require.NoError(t, err)
	_, err = RunTask(ctx, TaskRequest{
		RunID: "run", NodeKey: "normalize", Attempt: 1,
		Task:   taskByKind(t, registry, "cookbook.linear.normalize-customers"),
		Inputs: map[string]workflowv3.ArtifactRef{"source": ref}, Artifacts: artifacts,
	})
	require.ErrorContains(t, err, `unsupported module "db:ambient"`)
}

func TestRunTaskRejectsWrongInputSchema(t *testing.T) {
	ctx := context.Background()
	artifacts, err := workflowv3.NewFileArtifactStore(t.TempDir(), 1024)
	require.NoError(t, err)
	ref, err := artifacts.Put(ctx, "wrong/v1", "text/plain", []byte("x"))
	require.NoError(t, err)
	_, err = RunTask(ctx, TaskRequest{
		RunID: "run", NodeKey: "normalize", Attempt: 1,
		Task:   taskByKind(t, sealedLinearRegistry(t), "cookbook.linear.normalize-customers"),
		Inputs: map[string]workflowv3.ArtifactRef{"source": ref}, Artifacts: artifacts,
	})
	require.ErrorContains(t, err, "does not match")
}
