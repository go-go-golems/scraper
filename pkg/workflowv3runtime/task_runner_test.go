package workflowv3runtime

import (
	"context"
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
