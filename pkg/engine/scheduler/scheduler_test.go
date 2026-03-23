package scheduler

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	"github.com/stretchr/testify/require"
)

func TestConfigValidateRejectsZeroValues(t *testing.T) {
	cfg := Config{}

	err := cfg.Validate()
	require.Error(t, err)
}

func TestConfigValidateAcceptsPositiveValues(t *testing.T) {
	cfg := Config{
		MaxWorkers:           4,
		PollInterval:         1 * time.Second,
		DefaultLeaseDuration: 30 * time.Second,
	}

	require.NoError(t, cfg.Validate())
}

type runnerFunc struct {
	kind string
	run  func(ctx context.Context, runCtx runner.RunContext) (*model.OpResult, error)
}

func (f runnerFunc) Kind() string {
	return f.kind
}

func (f runnerFunc) Run(ctx context.Context, runCtx runner.RunContext) (*model.OpResult, error) {
	return f.run(ctx, runCtx)
}

type recordingObserver struct {
	events []Event
}

func (o *recordingObserver) OnSchedulerEvent(ctx context.Context, event Event) {
	o.events = append(o.events, event)
}

func TestSchedulerFanOutAndDependencyCompletion(t *testing.T) {
	ctx := context.Background()
	store := openSchedulerStore(t)
	registry := runner.NewRegistry()

	require.NoError(t, registry.Register(runnerFunc{
		kind: "seed",
		run: func(ctx context.Context, runCtx runner.RunContext) (*model.OpResult, error) {
			return &model.OpResult{
				OpID:        runCtx.Op.ID,
				Data:        []byte(`{"phase":"seed"}`),
				CompletedAt: runCtx.Now,
				Emitted: []model.OpSpec{
					{
						ID:         "child-op",
						WorkflowID: runCtx.Workflow.ID,
						Site:       runCtx.Workflow.Site,
						Kind:       "child",
						Queue:      "site:nereval:js",
						DedupKey:   "child-op",
						Input:      []byte(`{"step":"child"}`),
						DependsOn: []model.Dependency{
							{OpID: runCtx.Op.ID, Required: true},
						},
					},
				},
			}, nil
		},
	}))
	require.NoError(t, registry.Register(runnerFunc{
		kind: "child",
		run: func(ctx context.Context, runCtx runner.RunContext) (*model.OpResult, error) {
			dep, err := runCtx.Dependencies.Result(ctx, runCtx.Workflow.ID, "seed-op")
			require.NoError(t, err)
			require.NotNil(t, dep)
			return &model.OpResult{
				OpID:        runCtx.Op.ID,
				Data:        []byte(`{"phase":"child","seed":"ok"}`),
				CompletedAt: runCtx.Now,
			}, nil
		},
	}))

	observer := &recordingObserver{}
	s, err := New(store, registry, testSchedulerConfig(), "worker-1", observer)
	require.NoError(t, err)

	current := time.Date(2026, 3, 23, 14, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return current }

	err = s.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{
			ID:   "wf-fanout",
			Site: "nereval",
			Name: "fanout test",
		},
		Initial: []model.OpSpec{
			{
				ID:         "seed-op",
				WorkflowID: "wf-fanout",
				Site:       "nereval",
				Kind:       "seed",
				Queue:      "site:nereval:js",
				DedupKey:   "seed-op",
			},
		},
	})
	require.NoError(t, err)

	first, err := s.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, first.Processed)

	workflow, err := store.GetWorkflow(ctx, "wf-fanout")
	require.NoError(t, err)
	require.Equal(t, model.WorkflowStatusRunning, workflow.Status)

	childOp, err := store.GetOp(ctx, "child-op")
	require.NoError(t, err)
	require.NotNil(t, childOp)
	require.Len(t, childOp.DependsOn, 1)

	current = current.Add(1 * time.Second)
	second, err := s.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, second.Processed)

	childResult, err := store.GetResult(ctx, "wf-fanout", "child-op")
	require.NoError(t, err)
	require.NotNil(t, childResult)
	require.JSONEq(t, `{"phase":"child","seed":"ok"}`, string(childResult.Data))

	workflow, err = store.GetWorkflow(ctx, "wf-fanout")
	require.NoError(t, err)
	require.Equal(t, model.WorkflowStatusSucceeded, workflow.Status)
	requireEventKinds(t, observer.events, EventWorkflowCreated, EventOpLeased, EventOpSucceeded, EventWorkflowUpdated)
}

func TestSchedulerRetriesRetryableFailures(t *testing.T) {
	ctx := context.Background()
	store := openSchedulerStore(t)
	registry := runner.NewRegistry()

	attempts := 0
	require.NoError(t, registry.Register(runnerFunc{
		kind: "flaky",
		run: func(ctx context.Context, runCtx runner.RunContext) (*model.OpResult, error) {
			attempts++
			if attempts == 1 {
				return &model.OpResult{
					OpID: runCtx.Op.ID,
					Error: &model.OpError{
						Code:      "temporary",
						Message:   "transient issue",
						Retryable: true,
					},
				}, nil
			}
			return &model.OpResult{
				OpID:        runCtx.Op.ID,
				Data:        []byte(`{"attempts":2}`),
				CompletedAt: runCtx.Now,
			}, nil
		},
	}))

	observer := &recordingObserver{}
	s, err := New(store, registry, testSchedulerConfig(), "worker-1", observer)
	require.NoError(t, err)

	current := time.Date(2026, 3, 23, 15, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return current }

	err = s.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{
			ID:   "wf-retry",
			Site: "nereval",
			Name: "retry test",
		},
		Initial: []model.OpSpec{
			{
				ID:         "retry-op",
				WorkflowID: "wf-retry",
				Site:       "nereval",
				Kind:       "flaky",
				Queue:      "site:nereval:js",
				DedupKey:   "retry-op",
				Retry: model.RetryPolicy{
					MaxAttempts:    3,
					BackoffKind:    model.BackoffKindFixed,
					InitialBackoff: 1 * time.Second,
					MaxBackoff:     5 * time.Second,
				},
			},
		},
	})
	require.NoError(t, err)

	first, err := s.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, first.Processed)

	op, err := store.GetOp(ctx, "retry-op")
	require.NoError(t, err)
	require.Equal(t, 1, op.RetryState.Attempt)
	require.NotNil(t, op.RetryState.NextAttemptAt)

	current = current.Add(500 * time.Millisecond)
	second, err := s.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, second.Processed)

	current = current.Add(2 * time.Second)
	third, err := s.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, third.Processed)

	result, err := store.GetResult(ctx, "wf-retry", "retry-op")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.JSONEq(t, `{"attempts":2}`, string(result.Data))

	workflow, err := store.GetWorkflow(ctx, "wf-retry")
	require.NoError(t, err)
	require.Equal(t, model.WorkflowStatusSucceeded, workflow.Status)
	requireEventKinds(t, observer.events, EventOpRetried, EventOpSucceeded)
}

func TestSchedulerRecoversExpiredLeases(t *testing.T) {
	ctx := context.Background()
	store := openSchedulerStore(t)
	registry := runner.NewRegistry()

	require.NoError(t, registry.Register(runnerFunc{
		kind: "resume",
		run: func(ctx context.Context, runCtx runner.RunContext) (*model.OpResult, error) {
			return &model.OpResult{
				OpID:        runCtx.Op.ID,
				Data:        []byte(`{"resumed":true}`),
				CompletedAt: runCtx.Now,
			}, nil
		},
	}))

	s, err := New(store, registry, testSchedulerConfig(), "worker-2", nil)
	require.NoError(t, err)

	start := time.Date(2026, 3, 23, 16, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return start.Add(2 * time.Minute) }

	err = s.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{
			ID:   "wf-resume",
			Site: "nereval",
			Name: "resume test",
		},
		Initial: []model.OpSpec{
			{
				ID:         "resume-op",
				WorkflowID: "wf-resume",
				Site:       "nereval",
				Kind:       "resume",
				Queue:      "site:nereval:http",
				DedupKey:   "resume-op",
			},
		},
	})
	require.NoError(t, err)

	leasedOp, lease, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID:      "crashed-worker",
		Queue:         "site:nereval:http",
		Site:          "nereval",
		LeaseDuration: 30 * time.Second,
		Now:           start,
	})
	require.NoError(t, err)
	require.NotNil(t, leasedOp)
	require.NotNil(t, lease)

	cycle, err := s.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cycle.Processed)

	result, err := store.GetResult(ctx, "wf-resume", "resume-op")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.JSONEq(t, `{"resumed":true}`, string(result.Data))
}

func TestSchedulerHonorsQueueRateDomainOnePerQueue(t *testing.T) {
	ctx := context.Background()
	store := openSchedulerStore(t)
	registry := runner.NewRegistry()

	require.NoError(t, registry.Register(runnerFunc{
		kind: "queue-test",
		run: func(ctx context.Context, runCtx runner.RunContext) (*model.OpResult, error) {
			return &model.OpResult{
				OpID:        runCtx.Op.ID,
				Data:        []byte(`{"ok":true}`),
				CompletedAt: runCtx.Now,
			}, nil
		},
	}))

	s, err := New(store, registry, testSchedulerConfig(), "worker-3", nil)
	require.NoError(t, err)

	current := time.Date(2026, 3, 23, 17, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return current }

	err = s.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{
			ID:   "wf-queue",
			Site: "nereval",
			Name: "queue test",
		},
		Initial: []model.OpSpec{
			{
				ID:         "queue-op-1",
				WorkflowID: "wf-queue",
				Site:       "nereval",
				Kind:       "queue-test",
				Queue:      "site:nereval:http",
				DedupKey:   "queue-op-1",
			},
			{
				ID:         "queue-op-2",
				WorkflowID: "wf-queue",
				Site:       "nereval",
				Kind:       "queue-test",
				Queue:      "site:nereval:http",
				DedupKey:   "queue-op-2",
			},
		},
	})
	require.NoError(t, err)

	first, err := s.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, first.Processed)

	current = current.Add(1 * time.Second)
	second, err := s.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, second.Processed)

	stats, err := store.GetWorkflowStats(ctx, "wf-queue")
	require.NoError(t, err)
	require.Equal(t, 2, stats.Succeeded)
}

func openSchedulerStore(t *testing.T) *sqlitestore.Store {
	t.Helper()

	store, err := sqlitestore.Open(context.Background(), filepath.Join(t.TempDir(), "engine.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	return store
}

func testSchedulerConfig() Config {
	return Config{
		MaxWorkers:           4,
		PollInterval:         10 * time.Millisecond,
		DefaultLeaseDuration: 30 * time.Second,
	}
}

func requireEventKinds(t *testing.T, events []Event, kinds ...EventKind) {
	t.Helper()

	seen := map[EventKind]bool{}
	for _, event := range events {
		seen[event.Kind] = true
	}
	for _, kind := range kinds {
		require.True(t, seen[kind], "missing event kind %s", kind)
	}
}
