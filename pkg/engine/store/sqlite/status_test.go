package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	"github.com/stretchr/testify/require"
)

func TestInspectMissingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")

	status, err := Inspect(context.Background(), path)
	require.NoError(t, err)
	require.False(t, status.Exists)
	require.False(t, status.Initialized)
	require.Equal(t, 2, status.LatestKnownMigration)
	require.Len(t, status.Migrations, 2)
	require.False(t, status.Migrations[0].Applied)
}

func TestInspectPopulatedDatabase(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "engine.db")
	store, err := Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()

	workflow := model.WorkflowRun{
		ID:        model.WorkflowID("wf-1"),
		Site:      model.SiteName("nereval"),
		Name:      "Smoke test",
		Status:    model.WorkflowStatusRunning,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	initial := []model.OpSpec{
		{
			ID:         model.OpID("op-ready"),
			WorkflowID: workflow.ID,
			Site:       workflow.Site,
			Kind:       "http/fetch",
			Queue:      model.QueueKey("site:nereval:http"),
			Retry: model.RetryPolicy{
				MaxAttempts:    3,
				BackoffKind:    model.BackoffKindFixed,
				InitialBackoff: time.Second,
				MaxBackoff:     5 * time.Second,
				Multiplier:     1,
			},
		},
	}
	require.NoError(t, store.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: workflow,
		Initial:  initial,
	}))

	leasedOp, lease, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID:      "worker-1",
		Queue:         model.QueueKey("site:nereval:http"),
		Site:          workflow.Site,
		LeaseDuration: 30 * time.Second,
		Now:           time.Now().UTC(),
	})
	require.NoError(t, err)
	require.NotNil(t, leasedOp)
	require.NotNil(t, lease)

	require.NoError(t, store.CompleteOp(ctx, leasedOp.ID, storecontract.Completion{
		Lease: *lease,
		Result: model.OpResult{
			OpID: leasedOp.ID,
			Data: []byte(`{"ok":true}`),
			Artifacts: []model.ArtifactWrite{
				{
					ID:          model.ArtifactID("artifact-1"),
					Name:        "response.html",
					Kind:        "http-response-body",
					ContentType: "text/html",
					Body:        []byte("<html></html>"),
				},
			},
			CompletedAt: time.Now().UTC(),
		},
	}))

	status, err := Inspect(ctx, path)
	require.NoError(t, err)
	require.True(t, status.Exists)
	require.True(t, status.Initialized)
	require.True(t, status.MigrationsUpToDate)
	require.Equal(t, 1, status.WorkflowCount)
	require.Equal(t, 1, status.OpCounts[model.OpStatusSucceeded])
	require.Equal(t, 0, status.ActiveLeases)
	require.Equal(t, 1, status.ResultCount)
	require.Equal(t, 1, status.ArtifactCount)
	require.Len(t, status.Migrations, 2)
	require.True(t, status.Migrations[0].Applied)
}
