package store

import (
	"context"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
)

type CreateWorkflowParams struct {
	Workflow model.WorkflowRun
	Initial  []model.OpSpec
}

type LeaseRequest struct {
	WorkerID      string
	Queue         model.QueueKey
	Site          model.SiteName
	LeaseDuration time.Duration
	Now           time.Time
}

type Completion struct {
	Lease  model.Lease
	Result model.OpResult
}

type Failure struct {
	Lease      model.Lease
	Error      model.OpError
	RetryState model.RetryState
}

type WorkflowStore interface {
	CreateWorkflow(ctx context.Context, params CreateWorkflowParams) error
	GetWorkflow(ctx context.Context, id model.WorkflowID) (*model.WorkflowRun, error)
	UpdateWorkflowStatus(ctx context.Context, id model.WorkflowID, status model.WorkflowStatus) error
}

type OpStore interface {
	Enqueue(ctx context.Context, ops []model.OpSpec) error
	GetOp(ctx context.Context, id model.OpID) (*model.OpSpec, error)
	LeaseReadyOp(ctx context.Context, req LeaseRequest) (*model.OpSpec, *model.Lease, error)
	HeartbeatLease(ctx context.Context, opID model.OpID, lease model.Lease, extendBy time.Duration) error
	CompleteOp(ctx context.Context, opID model.OpID, completion Completion) error
	FailOp(ctx context.Context, opID model.OpID, failure Failure) error
}

type ResultStore interface {
	GetResult(ctx context.Context, workflowID model.WorkflowID, opID model.OpID) (*model.OpResult, error)
}

type Store interface {
	WorkflowStore
	OpStore
	ResultStore
}
