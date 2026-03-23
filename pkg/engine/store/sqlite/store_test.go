package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestOpenAppliesLatestMigrations(t *testing.T) {
	store := openTestStore(t)
	defer func() { require.NoError(t, store.Close()) }()

	version, err := store.CurrentVersion(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, version)
}

func TestMigrateUpgradeFromVersionOne(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "engine.db")
	db, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()

	migrations, err := loadMigrations()
	require.NoError(t, err)
	require.Len(t, migrations, 2)

	_, err = db.ExecContext(ctx, `
		CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		)
	`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, migrations[0].sql)
	require.NoError(t, err)
	_, err = db.ExecContext(
		ctx,
		`INSERT INTO schema_migrations(version, name, applied_at) VALUES(?, ?, ?)`,
		migrations[0].version,
		migrations[0].name,
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	require.NoError(t, err)

	require.NoError(t, migrate(ctx, db))

	version, err := currentVersion(ctx, db)
	require.NoError(t, err)
	require.Equal(t, 2, version)

	var tableName string
	err = db.QueryRowContext(
		ctx,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'results'`,
	).Scan(&tableName)
	require.NoError(t, err)
	require.Equal(t, "results", tableName)
}

func TestStoreWorkflowLeaseCompleteRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { require.NoError(t, store.Close()) }()

	workflow := model.WorkflowRun{
		ID:        model.WorkflowID("wf-1"),
		Site:      model.SiteName("nereval"),
		Name:      "NEREVAL bootstrap",
		Status:    model.WorkflowStatusPending,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	initial := []model.OpSpec{
		{
			ID:         model.OpID("op-list-fetch"),
			WorkflowID: workflow.ID,
			Site:       workflow.Site,
			Kind:       "http/fetch",
			Queue:      model.QueueKey("site:nereval:http"),
			Input:      []byte(`{"page":1}`),
			Retry: model.RetryPolicy{
				MaxAttempts:    3,
				BackoffKind:    model.BackoffKindExponential,
				InitialBackoff: 1 * time.Second,
				MaxBackoff:     10 * time.Second,
				Multiplier:     2,
			},
		},
	}

	err := store.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: workflow,
		Initial:  initial,
	})
	require.NoError(t, err)

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
	require.Equal(t, initial[0].ID, leasedOp.ID)

	err = store.CompleteOp(ctx, leasedOp.ID, storecontract.Completion{
		Lease: *lease,
		Result: model.OpResult{
			OpID:        leasedOp.ID,
			Data:        []byte(`{"status":"ok"}`),
			EmittedIDs:  []model.OpID{model.OpID("op-list-extract")},
			CompletedAt: time.Now().UTC(),
		},
	})
	require.NoError(t, err)

	result, err := store.GetResult(ctx, workflow.ID, leasedOp.ID)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.JSONEq(t, `{"status":"ok"}`, string(result.Data))
}

func openTestStore(t *testing.T) *Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "engine.db")
	store, err := Open(context.Background(), path)
	require.NoError(t, err)

	return store
}
