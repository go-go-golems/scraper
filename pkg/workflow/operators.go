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

// RetryStep moves a failed step back to ready so workers can execute it again.
func (rt *Runtime) RetryStep(ctx context.Context, runID model.WorkflowID, stepID model.OpID) error {
	if rt == nil || rt.operators == nil {
		return fmt.Errorf("workflow runtime operator service is not configured")
	}
	return rt.operators.RetryOp(ctx, runID, stepID)
}

// CancelRun cancels pending, ready, and running steps for a run. The current
// SQLite implementation marks running steps canceled and removes leases; future
// phases should add cooperative executor cancellation for in-flight subprocesses.
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
