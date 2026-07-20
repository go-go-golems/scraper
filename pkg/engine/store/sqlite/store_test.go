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
	require.Equal(t, 3, version)
}

func TestMigrateUpgradeFromVersionOne(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "engine.db")
	db, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()

	migrations, err := loadMigrations()
	require.NoError(t, err)
	require.Len(t, migrations, 3)

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
	require.Equal(t, 3, version)

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

func TestLeaseReadyOpTokenBucketBurstOne(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { require.NoError(t, store.Close()) }()

	now := time.Date(2026, 3, 23, 18, 30, 0, 0, time.UTC)
	workflow := model.WorkflowRun{
		ID:   "wf-rate-one",
		Site: "js-demo",
		Name: "rate limiter burst one",
	}
	require.NoError(t, store.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: workflow,
		Initial: []model.OpSpec{
			{ID: "op-1", WorkflowID: workflow.ID, Site: workflow.Site, Kind: "js", Queue: "site:js-demo:js", DedupKey: "op-1"},
			{ID: "op-2", WorkflowID: workflow.ID, Site: workflow.Site, Kind: "js", Queue: "site:js-demo:js", DedupKey: "op-2"},
		},
	}))

	policy := model.QueuePolicy{
		MaxInFlight: 1,
		RateLimit: &model.RateLimitPolicy{
			Kind:          model.RateLimitKindTokenBucket,
			RatePerSecond: 1,
			Burst:         1,
		},
	}

	firstOp, firstLease, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID:      "worker-1",
		Queue:         "site:js-demo:js",
		Site:          "js-demo",
		Policy:        policy,
		LeaseDuration: 30 * time.Second,
		Now:           now,
	})
	require.NoError(t, err)
	require.NotNil(t, firstOp)
	require.NotNil(t, firstLease)
	require.NoError(t, store.CompleteOp(ctx, firstOp.ID, storecontract.Completion{
		Lease: *firstLease,
		Result: model.OpResult{
			OpID:        firstOp.ID,
			CompletedAt: now,
		},
	}))

	secondOp, secondLease, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID:      "worker-1",
		Queue:         "site:js-demo:js",
		Site:          "js-demo",
		Policy:        policy,
		LeaseDuration: 30 * time.Second,
		Now:           now,
	})
	require.NoError(t, err)
	require.Nil(t, secondOp)
	require.Nil(t, secondLease)

	secondOp, secondLease, err = store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID:      "worker-1",
		Queue:         "site:js-demo:js",
		Site:          "js-demo",
		Policy:        policy,
		LeaseDuration: 30 * time.Second,
		Now:           now.Add(1 * time.Second),
	})
	require.NoError(t, err)
	require.NotNil(t, secondOp)
	require.NotNil(t, secondLease)
	require.Equal(t, model.OpID("op-2"), secondOp.ID)
}

func TestLeaseReadyOpTokenBucketBurstTwo(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { require.NoError(t, store.Close()) }()

	now := time.Date(2026, 3, 23, 18, 45, 0, 0, time.UTC)
	workflow := model.WorkflowRun{
		ID:   "wf-rate-two",
		Site: "js-demo",
		Name: "rate limiter burst two",
	}
	require.NoError(t, store.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: workflow,
		Initial: []model.OpSpec{
			{ID: "op-a", WorkflowID: workflow.ID, Site: workflow.Site, Kind: "js", Queue: "site:js-demo:js", DedupKey: "op-a"},
			{ID: "op-b", WorkflowID: workflow.ID, Site: workflow.Site, Kind: "js", Queue: "site:js-demo:js", DedupKey: "op-b"},
			{ID: "op-c", WorkflowID: workflow.ID, Site: workflow.Site, Kind: "js", Queue: "site:js-demo:js", DedupKey: "op-c"},
		},
	}))

	policy := model.QueuePolicy{
		MaxInFlight: 1,
		RateLimit: &model.RateLimitPolicy{
			Kind:          model.RateLimitKindTokenBucket,
			RatePerSecond: 1,
			Burst:         2,
		},
	}

	leaseAndComplete := func(current time.Time) model.OpID {
		op, lease, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
			WorkerID:      "worker-1",
			Queue:         "site:js-demo:js",
			Site:          "js-demo",
			Policy:        policy,
			LeaseDuration: 30 * time.Second,
			Now:           current,
		})
		require.NoError(t, err)
		require.NotNil(t, op)
		require.NotNil(t, lease)
		require.NoError(t, store.CompleteOp(ctx, op.ID, storecontract.Completion{
			Lease: *lease,
			Result: model.OpResult{
				OpID:        op.ID,
				CompletedAt: current,
			},
		}))
		return op.ID
	}

	require.Equal(t, model.OpID("op-a"), leaseAndComplete(now))
	require.Equal(t, model.OpID("op-b"), leaseAndComplete(now))

	blockedOp, blockedLease, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID:      "worker-1",
		Queue:         "site:js-demo:js",
		Site:          "js-demo",
		Policy:        policy,
		LeaseDuration: 30 * time.Second,
		Now:           now,
	})
	require.NoError(t, err)
	require.Nil(t, blockedOp)
	require.Nil(t, blockedLease)

	nextOp, nextLease, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID:      "worker-1",
		Queue:         "site:js-demo:js",
		Site:          "js-demo",
		Policy:        policy,
		LeaseDuration: 30 * time.Second,
		Now:           now.Add(1 * time.Second),
	})
	require.NoError(t, err)
	require.NotNil(t, nextOp)
	require.NotNil(t, nextLease)
	require.Equal(t, model.OpID("op-c"), nextOp.ID)
}

func TestLeaseReadyOpTokenBucketStatePersistsAcrossReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "engine.db")

	store, err := Open(ctx, path)
	require.NoError(t, err)

	now := time.Date(2026, 3, 23, 19, 15, 0, 0, time.UTC)
	workflow := model.WorkflowRun{
		ID:   "wf-rate-reopen",
		Site: "js-demo",
		Name: "rate limiter reopen",
	}
	require.NoError(t, store.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: workflow,
		Initial: []model.OpSpec{
			{ID: "op-1", WorkflowID: workflow.ID, Site: workflow.Site, Kind: "js", Queue: "site:js-demo:js", DedupKey: "op-1"},
			{ID: "op-2", WorkflowID: workflow.ID, Site: workflow.Site, Kind: "js", Queue: "site:js-demo:js", DedupKey: "op-2"},
		},
	}))

	policy := model.QueuePolicy{
		MaxInFlight: 1,
		RateLimit: &model.RateLimitPolicy{
			Kind:          model.RateLimitKindTokenBucket,
			RatePerSecond: 1,
			Burst:         1,
		},
	}

	firstOp, firstLease, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID:      "worker-1",
		Queue:         "site:js-demo:js",
		Site:          "js-demo",
		Policy:        policy,
		LeaseDuration: 30 * time.Second,
		Now:           now,
	})
	require.NoError(t, err)
	require.NotNil(t, firstOp)
	require.NotNil(t, firstLease)
	require.NoError(t, store.CompleteOp(ctx, firstOp.ID, storecontract.Completion{
		Lease: *firstLease,
		Result: model.OpResult{
			OpID:        firstOp.ID,
			CompletedAt: now,
		},
	}))
	require.NoError(t, store.Close())

	reopened, err := Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, reopened.Close()) }()

	blockedOp, blockedLease, err := reopened.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID:      "worker-2",
		Queue:         "site:js-demo:js",
		Site:          "js-demo",
		Policy:        policy,
		LeaseDuration: 30 * time.Second,
		Now:           now,
	})
	require.NoError(t, err)
	require.Nil(t, blockedOp)
	require.Nil(t, blockedLease)

	nextOp, nextLease, err := reopened.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID:      "worker-2",
		Queue:         "site:js-demo:js",
		Site:          "js-demo",
		Policy:        policy,
		LeaseDuration: 30 * time.Second,
		Now:           now.Add(1 * time.Second),
	})
	require.NoError(t, err)
	require.NotNil(t, nextOp)
	require.NotNil(t, nextLease)
	require.Equal(t, model.OpID("op-2"), nextOp.ID)
}

func TestMigrateBackfillsSortableTimestamps(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "legacy-engine.db")
	db, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()

	migrations, err := loadMigrations()
	require.NoError(t, err)
	require.Len(t, migrations, 3)
	_, err = db.ExecContext(ctx, migrations[0].sql)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, migrations[1].sql)
	require.NoError(t, err)
	for _, migration := range migrations[:2] {
		_, err = db.ExecContext(ctx, `INSERT INTO schema_migrations(version, name, applied_at) VALUES(?, ?, ?)`, migration.version, migration.name, "2026-07-20T18:00:00Z")
		require.NoError(t, err)
	}

	const created = "2026-07-20T18:00:01Z"
	const updated = "2026-07-20T18:00:01.5Z"
	_, err = db.ExecContext(ctx, `INSERT INTO workflows(id, site, name, status, input_json, metadata_json, created_at, updated_at) VALUES('wf', 'site', 'legacy', 'running', 'null', '{}', ?, ?)`, created, updated)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO ops(id, workflow_id, site, kind, queue_key, dedup_key, input_json, retry_json, metadata_json, status, retry_state_json, next_attempt_at, created_at, updated_at) VALUES('op', 'wf', 'site', 'kind', 'queue', 'dedup', 'null', '{}', '{}', 'running', '{}', ?, ?, ?)`, updated, created, updated)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO leases(op_id, worker_id, token, acquired_at, expires_at) VALUES('op', 'worker', 'token', ?, ?)`, created, updated)
	require.NoError(t, err)

	require.NoError(t, migrate(ctx, db))
	var expiresAt int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT expires_at_us FROM leases WHERE op_id = 'op'`).Scan(&expiresAt))
	expected, err := time.Parse(time.RFC3339Nano, updated)
	require.NoError(t, err)
	require.Equal(t, epochMicros(expected), expiresAt)

	store := &Store{db: db}
	changed, err := store.RefreshRunnableOps(ctx, expected)
	require.NoError(t, err)
	require.Equal(t, 1, changed)
	op, err := store.GetOp(ctx, "op")
	require.NoError(t, err)
	require.Equal(t, model.OpStatusReady, opStatus(t, db, "op"))
	require.Equal(t, expected, op.UpdatedAt)
}

func TestHeartbeatExtendsFromCurrentTimeAndRejectsLostLease(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { require.NoError(t, store.Close()) }()
	now := time.Date(2026, 7, 20, 18, 0, 1, 0, time.UTC)
	workflow, op, lease := createAndLeaseTestOp(t, ctx, store, now, time.Second)
	_ = workflow

	first, err := store.HeartbeatLease(ctx, op.ID, lease, now.Add(500*time.Millisecond), time.Second)
	require.NoError(t, err)
	require.Equal(t, now.Add(1500*time.Millisecond), first.ExpiresAt)
	second, err := store.HeartbeatLease(ctx, op.ID, *first, now.Add(time.Second), time.Second)
	require.NoError(t, err)
	require.Equal(t, now.Add(2*time.Second), second.ExpiresAt)

	_, err = store.HeartbeatLease(ctx, op.ID, lease, now.Add(3*time.Second), time.Second)
	require.ErrorIs(t, err, storecontract.ErrLeaseLost)
}

func TestStaleLeaseCannotCompleteOrFail(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() { require.NoError(t, store.Close()) }()
	now := time.Date(2026, 7, 20, 18, 0, 1, 0, time.UTC)
	workflow, op, oldLease := createAndLeaseTestOp(t, ctx, store, now, time.Second)
	_, err := store.RefreshRunnableOps(ctx, now.Add(2*time.Second))
	require.NoError(t, err)
	currentOp, currentLease, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{WorkerID: "worker-current", Site: workflow.Site, Queue: op.Queue, LeaseDuration: time.Second, Now: now.Add(2 * time.Second)})
	require.NoError(t, err)
	require.NotNil(t, currentOp)
	require.NotNil(t, currentLease)

	err = store.CompleteOp(ctx, op.ID, storecontract.Completion{Lease: oldLease, Result: model.OpResult{CompletedAt: now.Add(2 * time.Second), Data: []byte(`{"stale":true}`)}})
	require.ErrorIs(t, err, storecontract.ErrLeaseLost)
	err = store.FailOp(ctx, op.ID, storecontract.Failure{Lease: oldLease, Error: model.OpError{OccurredAt: now.Add(2 * time.Second)}})
	require.ErrorIs(t, err, storecontract.ErrLeaseLost)
	require.Equal(t, model.OpStatusRunning, opStatus(t, store.db, op.ID))
	require.NoError(t, store.CompleteOp(ctx, op.ID, storecontract.Completion{Lease: *currentLease, Result: model.OpResult{CompletedAt: now.Add(2 * time.Second), Data: []byte(`{"current":true}`)}}))
	result, err := store.GetResult(ctx, workflow.ID, op.ID)
	require.NoError(t, err)
	require.JSONEq(t, `{"current":true}`, string(result.Data))
}

func createAndLeaseTestOp(t *testing.T, ctx context.Context, store *Store, now time.Time, leaseDuration time.Duration) (model.WorkflowRun, *model.OpSpec, model.Lease) {
	t.Helper()
	workflow := model.WorkflowRun{ID: model.WorkflowID("wf-" + now.Format("150405.000000000")), Site: "site", Name: "test"}
	opSpec := model.OpSpec{ID: model.OpID("op-" + now.Format("150405.000000000")), WorkflowID: workflow.ID, Site: workflow.Site, Kind: "kind", Queue: "queue"}
	require.NoError(t, store.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{Workflow: workflow, Initial: []model.OpSpec{opSpec}}))
	op, lease, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{WorkerID: "worker-old", Site: workflow.Site, Queue: opSpec.Queue, LeaseDuration: leaseDuration, Now: now})
	require.NoError(t, err)
	require.NotNil(t, op)
	require.NotNil(t, lease)
	return workflow, op, *lease
}

func opStatus(t *testing.T, db *sql.DB, opID model.OpID) model.OpStatus {
	t.Helper()
	var status model.OpStatus
	require.NoError(t, db.QueryRow(`SELECT status FROM ops WHERE id = ?`, opID).Scan(&status))
	return status
}

func openTestStore(t *testing.T) *Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "engine.db")
	store, err := Open(context.Background(), path)
	require.NoError(t, err)

	return store
}
