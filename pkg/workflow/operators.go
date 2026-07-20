package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
)

// OperatorService is the minimal mutation surface needed by embedded operator
// controls. SQLiteStore provides this through the existing engineview service;
// future backends can provide their own implementation without changing the
// public Runtime methods.
type OperatorService interface {
	RetryOp(ctx context.Context, workflowID model.WorkflowID, opID model.OpID) error
	CancelWorkflow(ctx context.Context, workflowID model.WorkflowID) error
}

// RetryStep repairs a failed step and reopens dependency-blocked descendants
// whose required dependencies are no longer terminal blockers.
func (rt *Runtime) RetryStep(ctx context.Context, runID model.WorkflowID, stepID model.OpID) error {
	if rt == nil || rt.operators == nil {
		return fmt.Errorf("workflow runtime operator service is not configured")
	}
	return rt.operators.RetryOp(ctx, runID, stepID)
}

// CancelRun cancels pending, ready, running, and blocked steps for a run. The
// SQLite implementation removes their leases; active workers detect lease loss
// through their heartbeat and receive a canceled execution context.
func (rt *Runtime) CancelRun(ctx context.Context, runID model.WorkflowID) error {
	if rt == nil || rt.operators == nil {
		return fmt.Errorf("workflow runtime operator service is not configured")
	}
	return rt.operators.CancelWorkflow(ctx, runID)
}

type workerOptions struct {
	PollInterval time.Duration
	MaxCycles    int
	cycles       int
}

type WorkerOption func(*workerOptions)

func WithWorkerPollInterval(interval time.Duration) WorkerOption {
	return func(o *workerOptions) { o.PollInterval = interval }
}

func WithWorkerMaxCycles(maxCycles int) WorkerOption {
	return func(o *workerOptions) { o.MaxCycles = maxCycles }
}
