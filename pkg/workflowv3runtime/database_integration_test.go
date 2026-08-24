package workflowv3runtime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/testfixtures/workflowv3database"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3sqlite"
	"github.com/stretchr/testify/require"
)

func TestDatabaseSyncCrashAfterSideEffectIsIdempotentAcrossRestart(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.db")
	target := openSyncTarget(t, filepath.Join(root, "target.db"))
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(root, "artifacts"), 1<<20)
	require.NoError(t, err)
	privateCanary := "PRIVATE-DB-ROW-CANARY-5a7e"
	// Keep the source substantially larger than SQLite's fixed durable schema
	// footprint. The assertion below still rejects persistence of the source
	// payload; the added operation-ledger tables make a 500-row fixture too
	// small to distinguish that fixed overhead from a privacy regression.
	const rowCount = 750
	input := putJSONArtifact(t, artifacts, "database-sync-dataset-ref/v1", map[string]any{
		"expectedCount":    rowCount,
		"crashAfterCommit": true,
		"rows":             databaseRows(privateCanary, rowCount),
	})
	registry, err := workflowv3database.Registry()
	require.NoError(t, err)
	plan := authoredDatabasePlan(t, registry).Plan
	store, err := workflowv3sqlite.Open(ctx, workflowPath)
	require.NoError(t, err)
	engine, dispatcher := databaseEngine(t, store, registry, artifacts, target)
	engine.LeaseDuration = 20 * time.Millisecond
	require.NoError(t, engine.Submit(ctx, "database-restart", plan, map[string]workflowv3.ArtifactRef{
		"dataset": input,
	}))

	lease, err := dispatcher.DispatchOnce(ctx)
	require.NoError(t, err)
	require.NotNil(t, lease)
	registered, err := registry.ResolveNode(lease.PlanNode)
	require.NoError(t, err)
	inputs, err := store.ResolveInputs(ctx, *lease)
	require.NoError(t, err)
	_, err = RunTask(ctx, TaskRequest{
		RunID: lease.RunID, NodeKey: lease.NodeKey, Attempt: lease.Attempt,
		Task: registered, Inputs: inputs, Artifacts: artifacts,
		Modules: engine.Modules,
	})
	var taskFailure *TaskFailureError
	require.True(t, errors.As(err, &taskFailure))
	require.Equal(t, "DB_SYNC_POST_COMMIT", taskFailure.Failure.Code)
	require.True(t, taskFailure.Failure.Retryable)
	require.Equal(t, 1, queryInt(t, target, "SELECT COUNT(*) FROM workflow_sync_audit"))
	require.Equal(t, rowCount, queryInt(t, target, "SELECT COUNT(*) FROM workflow_sync_customers"))
	require.NoError(t, store.Close())

	time.Sleep(30 * time.Millisecond)
	reopened, err := workflowv3sqlite.Open(ctx, workflowPath)
	require.NoError(t, err)
	restarted, restartedDispatcher := databaseEngine(t, reopened, registry, artifacts, target)
	snapshot := runDispatcherUntilStatus(
		t, restartedDispatcher, restarted, "database-restart", "succeeded",
	)
	require.Len(t, snapshot.Attempts, 2)
	require.Equal(t, "lease_lost", snapshot.Attempts[0].Status)
	require.Nil(t, snapshot.Attempts[0].Failure)
	require.Equal(t, "succeeded", snapshot.Attempts[1].Status)
	require.Equal(t, workflowv3database.ResourceClass, snapshot.Attempts[1].ResourceClass)
	require.Equal(t, 1, queryInt(t, target, "SELECT COUNT(*) FROM workflow_sync_audit"))
	require.Equal(t, 1, queryInt(t, target, "SELECT COUNT(*) FROM workflow_sync_operations"))
	require.Equal(t, rowCount, queryInt(t, target, "SELECT COUNT(*) FROM workflow_sync_customers"))
	expectedOperationKey, err := workflowv3.Digest(struct {
		RunID   workflowv3.RunID   `json:"runId"`
		NodeKey workflowv3.NodeKey `json:"nodeKey"`
	}{RunID: "database-restart", NodeKey: "synchronize"})
	require.NoError(t, err)
	require.Equal(t, expectedOperationKey, queryString(
		t, target, "SELECT operation_key FROM workflow_sync_operations",
	))

	receipt, err := workflowv3.ReadArtifact(ctx, artifacts, snapshot.Outputs["receipt"])
	require.NoError(t, err)
	require.Contains(t, string(receipt), `"configureDenied":true`)
	require.Contains(t, string(receipt), `"applied":false`)
	require.Contains(t, string(receipt), fmt.Sprintf(`"count":%d`, rowCount))
	require.NoError(t, reopened.Close())

	finalStore, err := workflowv3sqlite.Open(ctx, workflowPath)
	require.NoError(t, err)
	finalSnapshot, err := finalStore.Snapshot(ctx, "database-restart")
	require.NoError(t, err)
	require.Equal(t, snapshot.Outputs, finalSnapshot.Outputs)
	require.NoError(t, finalStore.Close())
	persisted, persistedBytes := readSQLiteFiles(t, workflowPath)
	require.NotContains(t, string(persisted), privateCanary)
	require.NotContains(t, string(persisted), "INSERT INTO workflow_sync_customers")
	require.Less(t, persistedBytes, input.Size/2)
	t.Logf(
		"database sync privacy/storage evidence: source=%d persistedSQLite=%d ratio=%.4f",
		input.Size,
		persistedBytes,
		float64(persistedBytes)/float64(input.Size),
	)
}

func TestDatabaseSyncCardinalityFailureDoesNotBlockAnotherRun(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	target := openSyncTarget(t, filepath.Join(root, "target.db"))
	registry, err := workflowv3database.Registry()
	require.NoError(t, err)
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(root, "artifacts"), 1<<20)
	require.NoError(t, err)
	store, err := workflowv3sqlite.Open(ctx, filepath.Join(root, "workflow.db"))
	require.NoError(t, err)
	engine, dispatcher := databaseEngine(t, store, registry, artifacts, target)
	plan := authoredDatabasePlan(t, registry).Plan
	bad := putJSONArtifact(t, artifacts, "database-sync-dataset-ref/v1", map[string]any{
		"expectedCount": 2,
		"rows":          []map[string]any{{"id": "bad", "email": "bad@example.test"}},
	})
	good := putJSONArtifact(t, artifacts, "database-sync-dataset-ref/v1", map[string]any{
		"expectedCount": 1,
		"rows":          []map[string]any{{"id": "good", "email": "good@example.test"}},
	})
	require.NoError(t, engine.Submit(ctx, "database-bad", plan, map[string]workflowv3.ArtifactRef{"dataset": bad}))
	require.NoError(t, engine.Submit(ctx, "database-good", plan, map[string]workflowv3.ArtifactRef{"dataset": good}))

	runDispatcherUntilStatus(t, dispatcher, engine, "database-good", "succeeded")
	failed, err := engine.Snapshot(ctx, "database-bad")
	require.NoError(t, err)
	require.Equal(t, "failed", failed.Status)
	require.Equal(t, "DB_SYNC_CARDINALITY", failed.Attempts[0].Failure.Code)
	require.False(t, failed.Attempts[0].Failure.Retryable)
	require.Equal(t, 1, queryInt(t, target, "SELECT COUNT(*) FROM workflow_sync_audit"))
	require.NoError(t, store.Close())
}

func TestDatabaseWorkerRejectsMissingExactModuleAlias(t *testing.T) {
	registry, err := workflowv3database.Registry()
	require.NoError(t, err)
	modules, err := NewTaskModuleRegistry(FSInputModule())
	require.NoError(t, err)
	engine := &Engine{
		Store: &workflowv3sqlite.Store{}, Registry: registry,
		Artifacts: &stubArtifactStore{}, Modules: modules,
	}
	err = engine.validate()
	require.ErrorContains(t, err, "registry advertises modules")
	require.ErrorContains(t, err, workflowv3database.DatabaseAlias)
}

func databaseEngine(
	t *testing.T,
	store *workflowv3sqlite.Store,
	registry *workflowv3.SealedRegistry,
	artifacts workflowv3.ArtifactStore,
	target *sql.DB,
) (*Engine, *Dispatcher) {
	t.Helper()
	modules, err := NewTaskModuleRegistry(
		FSInputModule(),
		DatabaseModule(workflowv3database.DatabaseAlias, target),
	)
	require.NoError(t, err)
	engine := &Engine{
		Store: store, Registry: registry, Artifacts: artifacts,
		Modules: modules, LeaseDuration: 2 * time.Second,
	}
	return engine, &Dispatcher{
		Engine:       engine,
		Capacities:   map[string]int{workflowv3database.ResourceClass: 1},
		PollInterval: 2 * time.Millisecond,
	}
}

func authoredDatabasePlan(t *testing.T, registry *workflowv3.SealedRegistry) workflowmodule.AuthoringResult {
	t.Helper()
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	result, err := workflowmodule.Author(
		context.Background(),
		workflowv3database.WorkflowSource(),
		catalog,
		workflowv3database.DescriptorModule(),
	)
	require.NoError(t, err)
	return result
}

func openSyncTarget(t *testing.T, path string) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	database.SetMaxOpenConns(4)
	_, err = database.Exec(`
PRAGMA journal_mode = WAL;
CREATE TABLE workflow_sync_customers (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL
);
CREATE TABLE workflow_sync_operations (
  operation_key TEXT PRIMARY KEY,
  row_count INTEGER NOT NULL
);
CREATE TABLE workflow_sync_audit (
  sequence INTEGER PRIMARY KEY AUTOINCREMENT,
  operation_key TEXT NOT NULL UNIQUE
);`)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	return database
}

func databaseRows(canary string, count int) []map[string]any {
	rows := make([]map[string]any, 0, count)
	padding := strings.Repeat("p", 900)
	for index := 0; index < count; index++ {
		rows = append(rows, map[string]any{
			"id":      fmt.Sprintf("customer-%04d", index),
			"email":   fmt.Sprintf("customer-%04d@example.test", index),
			"private": canary + padding,
		})
	}
	return rows
}

func queryString(t *testing.T, database *sql.DB, query string) string {
	t.Helper()
	var value string
	require.NoError(t, database.QueryRow(query).Scan(&value))
	return value
}

func queryInt(t *testing.T, database *sql.DB, query string) int {
	t.Helper()
	var value int
	require.NoError(t, database.QueryRow(query).Scan(&value))
	return value
}

type stubArtifactStore struct{}

func (*stubArtifactStore) Put(context.Context, string, string, []byte) (workflowv3.ArtifactRef, error) {
	return workflowv3.ArtifactRef{}, nil
}

func (*stubArtifactStore) Open(context.Context, workflowv3.ArtifactRef) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func TestWatchLeaseRenewsLongRunningAttemptAuthority(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(root, "artifacts"), 1<<20)
	require.NoError(t, err)
	target := openSyncTarget(t, filepath.Join(root, "target.db"))
	registry, err := workflowv3database.Registry()
	require.NoError(t, err)
	store, err := workflowv3sqlite.Open(ctx, filepath.Join(root, "workflow.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	engine, dispatcher := databaseEngine(t, store, registry, artifacts, target)
	engine.LeaseDuration = 40 * time.Millisecond
	input := putJSONArtifact(t, artifacts, "database-sync-dataset-ref/v1", map[string]any{"expectedCount": 1, "rows": databaseRows("heartbeat", 1)})
	require.NoError(t, engine.Submit(ctx, "lease-heartbeat", authoredDatabasePlan(t, registry).Plan, map[string]workflowv3.ArtifactRef{"dataset": input}))
	lease, err := dispatcher.DispatchOnce(ctx)
	require.NoError(t, err)
	require.NotNil(t, lease)
	attemptCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go engine.watchLease(ctx, *lease, cancel, done)
	time.Sleep(130 * time.Millisecond)
	require.NoError(t, attemptCtx.Err())
	valid, err := store.LeaseValid(ctx, *lease, time.Now().UTC())
	require.NoError(t, err)
	require.True(t, valid)
	close(done)
	cancel()
}
