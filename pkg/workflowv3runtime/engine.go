package workflowv3runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3sqlite"
)

type Engine struct {
	Store         *workflowv3sqlite.Store
	Registry      *workflowv3.SealedRegistry
	Artifacts     workflowv3.ArtifactStore
	Modules       *TaskModuleRegistry
	LeaseDuration time.Duration
	Now           func() time.Time
}

func (e *Engine) Submit(ctx context.Context, runID workflowv3.RunID, plan workflowv3.WorkflowPlan, inputs map[string]workflowv3.ArtifactRef) error {
	if err := e.validate(); err != nil {
		return err
	}
	return e.Store.CreateRun(ctx, runID, plan, inputs, e.now())
}

func (e *Engine) RunOne(ctx context.Context) (bool, error) {
	if err := e.validate(); err != nil {
		return false, err
	}
	lease, err := e.Store.LeaseNext(ctx, e.Registry, e.now(), e.leaseDuration())
	if err != nil || lease == nil {
		return false, err
	}
	return true, e.ExecuteLease(ctx, *lease)
}

func (e *Engine) ExecuteLease(ctx context.Context, lease workflowv3.Lease) error {
	if err := e.validate(); err != nil {
		return err
	}
	registered, err := e.Registry.ResolveNode(lease.PlanNode)
	if err != nil {
		return fmt.Errorf("resolve leased implementation: %w", err)
	}
	inputs, err := e.Store.ResolveInputs(ctx, lease)
	if err != nil {
		failure := workflowv3.Failure{
			Class: "internal", Code: "WORKFLOW_INPUT_RESOLUTION",
			Retryable: false, Message: "task input resolution failed",
		}
		if persistErr := e.Store.Fail(ctx, lease, failure, e.now()); persistErr != nil {
			return fmt.Errorf("resolve inputs: %v; persist failure: %w", err, persistErr)
		}
		return &AttemptExecutionError{Err: fmt.Errorf("resolve inputs: %w", err)}
	}
	attemptCtx, cancelAttempt := context.WithCancel(ctx)
	watchDone := make(chan struct{})
	go e.watchLease(ctx, lease, cancelAttempt, watchDone)
	defer func() {
		close(watchDone)
		cancelAttempt()
	}()
	result, err := RunTask(attemptCtx, TaskRequest{
		RunID: lease.RunID, NodeKey: lease.NodeKey, Attempt: lease.Attempt,
		Task: registered, Inputs: inputs, Artifacts: e.Artifacts,
		Modules: e.Modules,
	})
	if err != nil {
		failure := workflowv3.Failure{
			Class: "internal", Code: "WORKFLOW_TASK_EXECUTION",
			Retryable: false, Message: "task execution failed",
		}
		var taskFailure *TaskFailureError
		if errors.As(err, &taskFailure) {
			failure = taskFailure.Failure
			failure.Message = "task reported " + failure.Code
		}
		if persistErr := e.Store.Fail(ctx, lease, failure, e.now()); persistErr != nil {
			return fmt.Errorf("execute task: %v; persist failure: %w", err, persistErr)
		}
		return &AttemptExecutionError{Err: fmt.Errorf("execute task: %w", err)}
	}
	if err := e.Store.Complete(ctx, lease, result.Outputs, e.now()); err != nil {
		return fmt.Errorf("complete task: %w", err)
	}
	return nil
}

func (e *Engine) RunUntilIdle(ctx context.Context) error {
	for {
		ran, err := e.RunOne(ctx)
		if err != nil {
			return err
		}
		if !ran {
			return nil
		}
	}
}

func (e *Engine) watchLease(
	ctx context.Context,
	lease workflowv3.Lease,
	cancel context.CancelFunc,
	done <-chan struct{},
) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			cancel()
			return
		case <-done:
			return
		case <-ticker.C:
			valid, err := e.Store.LeaseValid(ctx, lease, e.now())
			if err == nil && !valid {
				cancel()
				return
			}
		}
	}
}

func (e *Engine) Snapshot(ctx context.Context, runID workflowv3.RunID) (workflowv3.RunSnapshot, error) {
	if err := e.validate(); err != nil {
		return workflowv3.RunSnapshot{}, err
	}
	return e.Store.Snapshot(ctx, runID)
}

func (e *Engine) validate() error {
	if e == nil || e.Store == nil || e.Registry == nil || e.Artifacts == nil || e.Modules == nil {
		return fmt.Errorf("workflow v3 engine requires store, sealed registry, artifacts, and task modules")
	}
	advertised := e.Registry.ModuleAliases()
	configured := e.Modules.Aliases()
	if len(advertised) != len(configured) {
		return fmt.Errorf("registry advertises modules %v but runtime configures %v", advertised, configured)
	}
	for i := range advertised {
		if advertised[i] != configured[i] {
			return fmt.Errorf("registry advertises modules %v but runtime configures %v", advertised, configured)
		}
	}
	return nil
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}

func (e *Engine) leaseDuration() time.Duration {
	if e.LeaseDuration > 0 {
		return e.LeaseDuration
	}
	return 30 * time.Second
}
