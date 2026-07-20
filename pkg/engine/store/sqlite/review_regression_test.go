package sqlite

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	"github.com/stretchr/testify/require"
)

func TestRefreshTreatsBlockedOptionalDependencyAsTerminal(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { require.NoError(t, store.Close()) }()
	now := time.Date(2026, 7, 20, 20, 0, 0, 0, time.UTC)
	workflow := model.WorkflowRun{ID: "wf-optional-blocked", Site: "site", Name: "optional"}
	initial := []model.OpSpec{
		{ID: "failed", WorkflowID: workflow.ID, Site: workflow.Site, Kind: "kind", Queue: "queue"},
		{ID: "blocked", WorkflowID: workflow.ID, Site: workflow.Site, Kind: "kind", Queue: "queue", DependsOn: []model.Dependency{{OpID: "failed", Required: true}}},
		{ID: "optional-child", WorkflowID: workflow.ID, Site: workflow.Site, Kind: "kind", Queue: "queue", DependsOn: []model.Dependency{{OpID: "blocked", Required: false}}},
	}
	require.NoError(t, store.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{Workflow: workflow, Initial: initial}))
	op, lease, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{WorkerID: "worker", Site: workflow.Site, Queue: "queue", LeaseDuration: time.Minute, Now: now})
	require.NoError(t, err)
	require.Equal(t, model.OpID("failed"), op.ID)
	require.NoError(t, store.FailOp(ctx, op.ID, storecontract.Failure{Lease: *lease, Error: model.OpError{OccurredAt: now}}))
	_, err = store.RefreshRunnableOps(ctx, now)
	require.NoError(t, err)
	require.Equal(t, model.OpStatusBlocked, opStatus(t, store.db, "blocked"))
	require.Equal(t, model.OpStatusReady, opStatus(t, store.db, "optional-child"))
}

func TestListWorkflowSnapshotsIncludesBoundaryTimestampTies(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { require.NoError(t, store.Close()) }()
	stamp := time.Date(2026, 7, 20, 20, 0, 1, 0, time.UTC)
	for i := range 3 {
		workflow := model.WorkflowRun{ID: model.WorkflowID(fmt.Sprintf("wf-tie-%d", i)), Site: "site", Name: "tie"}
		require.NoError(t, store.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{Workflow: workflow, Initial: []model.OpSpec{{ID: model.OpID(fmt.Sprintf("op-tie-%d", i)), WorkflowID: workflow.ID, Site: workflow.Site, Kind: "kind", Queue: "queue"}}}))
		_, err := store.db.ExecContext(ctx, `UPDATE workflows SET updated_at = ?, updated_at_us = ? WHERE id = ?`, stamp.Format(time.RFC3339Nano), stamp.UnixMicro(), workflow.ID)
		require.NoError(t, err)
	}
	items, err := store.ListWorkflowSnapshots(ctx, stamp.Add(-time.Microsecond), 2)
	require.NoError(t, err)
	require.Len(t, items, 3)
}
