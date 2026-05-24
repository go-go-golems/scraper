package workflow

import (
	"context"
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

func TestRuntimeStartRunRejectsUnknownPackage(t *testing.T) {
	rt, err := NewRuntime(context.Background(), Config{Store: SQLiteStore(t.TempDir() + "/engine.db")})
	require.NoError(t, err)
	defer func() { require.NoError(t, rt.Close()) }()
	_, err = rt.StartRun(context.Background(), "missing", map[string]any{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not registered")
}
