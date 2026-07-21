package workflowv3runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3sqlite"
)

type Engine struct {
	Store         *workflowv3sqlite.Store
	Registry      *workflowv3.SealedRegistry
	Artifacts     workflowv3.ArtifactStore
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
	registered, err := e.Registry.Resolve(lease.PlanNode.Implementation)
	if err != nil {
		return true, fmt.Errorf("resolve leased implementation: %w", err)
	}
	inputs, err := e.Store.ResolveInputs(ctx, *lease)
	if err != nil {
		failure := workflowv3.Failure{
			Class: "internal", Code: "WORKFLOW_INPUT_RESOLUTION",
			Retryable: false, Message: "task input resolution failed",
		}
		if persistErr := e.Store.Fail(ctx, *lease, failure, e.now()); persistErr != nil {
			return true, fmt.Errorf("resolve inputs: %v; persist failure: %w", err, persistErr)
		}
		return true, fmt.Errorf("resolve inputs: %w", err)
	}
	result, err := RunTask(ctx, TaskRequest{
		RunID: lease.RunID, NodeKey: lease.NodeKey, Attempt: lease.Attempt,
		Task: registered, Inputs: inputs, Artifacts: e.Artifacts,
	})
	if err != nil {
		failure := workflowv3.Failure{
			Class: "internal", Code: "WORKFLOW_TASK_EXECUTION",
			Retryable: false, Message: "task execution failed",
		}
		if persistErr := e.Store.Fail(ctx, *lease, failure, e.now()); persistErr != nil {
			return true, fmt.Errorf("execute task: %v; persist failure: %w", err, persistErr)
		}
		return true, fmt.Errorf("execute task: %w", err)
	}
	if err := e.Store.Complete(ctx, *lease, result.Outputs, e.now()); err != nil {
		return true, fmt.Errorf("complete task: %w", err)
	}
	return true, nil
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

func (e *Engine) Snapshot(ctx context.Context, runID workflowv3.RunID) (workflowv3.RunSnapshot, error) {
	if err := e.validate(); err != nil {
		return workflowv3.RunSnapshot{}, err
	}
	return e.Store.Snapshot(ctx, runID)
}

func (e *Engine) validate() error {
	if e == nil || e.Store == nil || e.Registry == nil || e.Artifacts == nil {
		return fmt.Errorf("workflow v3 engine requires store, sealed registry, and artifacts")
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
