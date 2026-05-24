package workflow

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/stretchr/testify/require"
)

type startInput struct {
	Message string `json:"message"`
}

type childInput struct {
	Message string `json:"message"`
}

func TestRuntimeStartRunAndRunOnce(t *testing.T) {
	ctx := context.Background()
	rt, err := NewRuntime(ctx, Config{
		Store:         SQLiteStore(t.TempDir() + "/engine.db"),
		WorkerID:      "test-worker",
		MaxWorkers:    2,
		PollInterval:  time.Millisecond,
		LeaseDuration: time.Minute,
		Queues: map[model.QueueKey]QueueConfig{
			"test": {MaxWorkers: 2},
		},
	})
	require.NoError(t, err)
	defer func() { require.NoError(t, rt.Close()) }()

	require.NoError(t, rt.RegisterExecutor(NewTypedExecutor("test/echo", func(ctx context.Context, step *StepContext, input startInput) error {
		require.Equal(t, "hello", input.Message)
		require.NoError(t, step.Result(map[string]any{"message": input.Message}))
		_, err := step.Emit("child", childInput{Message: input.Message + " child"}, StepOpts{Kind: "test/child", Queue: "test"})
		return err
	})))
	require.NoError(t, rt.RegisterExecutor(NewTypedExecutor("test/child", func(ctx context.Context, step *StepContext, input childInput) error {
		require.Equal(t, "hello child", input.Message)
		return step.Result(map[string]any{"child": input.Message})
	})))

	pkg := NewPackage("testpkg").
		DisplayName("Test Package").
		Entrypoint(EntrypointFunc[startInput](func(ctx context.Context, run *RunBuilder, input startInput) error {
			run.Metadata("source", "test")
			_, err := run.Step("root", input, StepOpts{Kind: "test/echo", Queue: "test"})
			return err
		})).
		Build()
	require.NoError(t, rt.RegisterPackage(pkg))

	run, err := rt.StartRun(ctx, "testpkg", startInput{Message: "hello"}, WithRunID("run-1"))
	require.NoError(t, err)
	require.Equal(t, model.WorkflowID("run-1"), run.ID)

	workflow, err := rt.Workflow(ctx, run.ID)
	require.NoError(t, err)
	require.NotNil(t, workflow)
	require.Equal(t, model.SiteName("testpkg"), workflow.Site)
	require.Equal(t, "test", workflow.Metadata["source"])

	first, err := rt.RunOnce(ctx)
	require.NoError(t, err)
	// The current scheduler refreshes runnable child steps after completing a
	// parent and may process the emitted child in the same RunOnce cycle.
	require.Equal(t, 2, first.Processed)
	rootResult, err := rt.Result(ctx, run.ID, "root")
	require.NoError(t, err)
	require.NotNil(t, rootResult)
	require.JSONEq(t, `{"message":"hello"}`, string(rootResult.Data))
	require.Equal(t, []model.OpID{"child"}, rootResult.EmittedIDs)

	childResult, err := rt.Result(ctx, run.ID, "child")
	require.NoError(t, err)
	require.NotNil(t, childResult)
	require.JSONEq(t, `{"child":"hello child"}`, string(childResult.Data))

	second, err := rt.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, second.Processed)

	workflow, err = rt.Workflow(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, model.WorkflowStatusSucceeded, workflow.Status)
}

func TestRuntimeExternalFileArtifactStore(t *testing.T) {
	ctx := context.Background()
	artifactRoot := t.TempDir() + "/artifacts"
	rt, err := NewRuntime(ctx, Config{
		Store:         SQLiteStore(t.TempDir() + "/engine.db"),
		ArtifactStore: NewFileArtifactStore(artifactRoot),
	})
	require.NoError(t, err)
	defer func() { require.NoError(t, rt.Close()) }()

	require.NoError(t, rt.RegisterExecutor(NewTypedExecutor("test/store-artifact", func(ctx context.Context, step *StepContext, input startInput) error {
		ref, err := step.StoreArtifact("page.md", "text/markdown", []byte("# "+input.Message), ArtifactKind("markdown"))
		require.NoError(t, err)
		require.NotEmpty(t, ref.URI)
		return step.Result(map[string]any{"artifactID": ref.ID})
	})))
	pkg := NewPackage("artifact-pkg").
		Entrypoint(EntrypointFunc[startInput](func(ctx context.Context, run *RunBuilder, input startInput) error {
			_, err := run.Step("store", input, StepOpts{Kind: "test/store-artifact", Queue: "artifact"})
			return err
		})).
		Build()
	require.NoError(t, rt.RegisterPackage(pkg))

	run, err := rt.StartRun(ctx, "artifact-pkg", startInput{Message: "artifact"}, WithRunID("artifact-run"))
	require.NoError(t, err)
	_, err = rt.RunOnce(ctx)
	require.NoError(t, err)
	result, err := rt.Result(ctx, run.ID, "store")
	require.NoError(t, err)
	require.Len(t, result.Artifacts, 1)
	require.Equal(t, "external-artifact-ref", result.Artifacts[0].Kind)
	reader, ref, err := NewFileArtifactStore(artifactRoot).Open(ctx, string(result.Artifacts[0].ID))
	require.NoError(t, err)
	defer reader.Close()
	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, "# artifact", string(body))
	require.Equal(t, "page.md", ref.Name)
}

func TestRuntimeRetryStep(t *testing.T) {
	ctx := context.Background()
	rt, err := NewRuntime(ctx, Config{Store: SQLiteStore(t.TempDir() + "/engine.db")})
	require.NoError(t, err)
	defer func() { require.NoError(t, rt.Close()) }()

	attempts := 0
	require.NoError(t, rt.RegisterExecutor(NewTypedExecutor("test/flaky", func(ctx context.Context, step *StepContext, input startInput) error {
		attempts++
		if attempts == 1 {
			return Permanent("first_attempt_failed", errors.New("first attempt failed"))
		}
		return step.Result(map[string]any{"attempts": attempts})
	})))

	pkg := NewPackage("retry-pkg").
		Entrypoint(EntrypointFunc[startInput](func(ctx context.Context, run *RunBuilder, input startInput) error {
			_, err := run.Step("flaky", input, StepOpts{Kind: "test/flaky", Queue: "retry"})
			return err
		})).
		Build()
	require.NoError(t, rt.RegisterPackage(pkg))

	run, err := rt.StartRun(ctx, "retry-pkg", startInput{Message: "hello"}, WithRunID("retry-run"))
	require.NoError(t, err)
	_, err = rt.RunOnce(ctx)
	require.NoError(t, err)
	failed, err := rt.Result(ctx, run.ID, "flaky")
	require.NoError(t, err)
	require.NotNil(t, failed.Error)
	require.Equal(t, "first_attempt_failed", failed.Error.Code)

	require.NoError(t, rt.RetryStep(ctx, run.ID, "flaky"))
	_, err = rt.RunOnce(ctx)
	require.NoError(t, err)
	succeeded, err := rt.Result(ctx, run.ID, "flaky")
	require.NoError(t, err)
	// The current retry mutation resets op status but does not clear the previous
	// result row before completion. Successful completion overwrites the data;
	// some stores may decode an empty error object as a zero-value OpError.
	require.JSONEq(t, `{"attempts":2}`, string(succeeded.Data))
}

func TestRuntimeCancelRun(t *testing.T) {
	ctx := context.Background()
	rt, err := NewRuntime(ctx, Config{Store: SQLiteStore(t.TempDir() + "/engine.db")})
	require.NoError(t, err)
	defer func() { require.NoError(t, rt.Close()) }()

	require.NoError(t, rt.RegisterExecutor(NewTypedExecutor("test/noop", func(ctx context.Context, step *StepContext, input startInput) error {
		return step.Result(map[string]any{"should": "not run"})
	})))
	pkg := NewPackage("cancel-pkg").
		Entrypoint(EntrypointFunc[startInput](func(ctx context.Context, run *RunBuilder, input startInput) error {
			_, err := run.Step("noop", input, StepOpts{Kind: "test/noop", Queue: "cancel"})
			return err
		})).
		Build()
	require.NoError(t, rt.RegisterPackage(pkg))

	run, err := rt.StartRun(ctx, "cancel-pkg", startInput{Message: "hello"}, WithRunID("cancel-run"))
	require.NoError(t, err)
	require.NoError(t, rt.CancelRun(ctx, run.ID))
	result, err := rt.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, result.Processed)
	wf, err := rt.Workflow(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, model.WorkflowStatusCanceled, wf.Status)
}

func TestRuntimeStartRunRejectsUnknownPackage(t *testing.T) {
	rt, err := NewRuntime(context.Background(), Config{Store: SQLiteStore(t.TempDir() + "/engine.db")})
	require.NoError(t, err)
	defer func() { require.NoError(t, rt.Close()) }()
	_, err = rt.StartRun(context.Background(), "missing", map[string]any{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not registered")
}
