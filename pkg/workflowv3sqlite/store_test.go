package workflowv3sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/stretchr/testify/require"
)

func storeFixture(t *testing.T, source string) (*workflowv3.SealedRegistry, workflowv3.WorkflowPlan) {
	t.Helper()
	bundle, err := workflowv3.NewBundle(workflowv3.BundleManifest{
		Name: "fixture", Version: "1", ABI: workflowv3.TaskABI,
		Tasks: []workflowv3.BundleTask{
			{
				TaskKey:    workflowv3.TaskKey{Kind: "linear.normalize", Version: "v1"},
				Entrypoint: "tasks.cjs#normalize",
				Inputs:     map[string]string{"source": "source/v1"},
				Outputs:    map[string]string{"dataset": "dataset/v1"},
				Retry: workflowv3.RetryPolicy{
					MaxAttempts: 3, BackoffMillis: 500,
				},
			},
			{
				TaskKey:    workflowv3.TaskKey{Kind: "linear.validate", Version: "v1"},
				Entrypoint: "tasks.cjs#validate",
				Inputs:     map[string]string{"dataset": "dataset/v1"},
				Outputs:    map[string]string{"validated": "validated/v1"},
			},
		},
	}, map[string][]byte{"tasks.cjs": []byte(source)})
	require.NoError(t, err)
	builder := workflowv3.NewRegistryBuilder()
	require.NoError(t, builder.AddBundle(bundle))
	registry, err := builder.Seal()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	plan, err := workflowv3.Compile(workflowv3.WorkflowIR{
		Schema: workflowv3.IRSchema,
		Name:   "linear",
		Inputs: []workflowv3.IRInput{{Name: "source", Schema: "source/v1"}},
		Nodes: []workflowv3.IRNode{
			{
				Key:  "normalize",
				Task: workflowv3.TaskKey{Kind: "linear.normalize", Version: "v1"},
				Bindings: map[string]workflowv3.ValueRef{
					"source": {Source: "input", Name: "source", Schema: "source/v1"},
				},
			},
			{
				Key:  "validate",
				Task: workflowv3.TaskKey{Kind: "linear.validate", Version: "v1"},
				Bindings: map[string]workflowv3.ValueRef{
					"dataset": {
						Source: "node-output", NodeKey: "normalize", Port: "dataset",
						Schema: "dataset/v1",
					},
				},
				DependsOn: []workflowv3.NodeKey{"normalize"},
			},
		},
		Outputs: []workflowv3.IROutput{{
			Name: "result",
			Value: workflowv3.ValueRef{
				Source: "node-output", NodeKey: "validate", Port: "validated",
				Schema: "validated/v1",
			},
		}},
	}, catalog)
	require.NoError(t, err)
	return registry, plan
}

func artifactRef(schema, seed string) workflowv3.ArtifactRef {
	return workflowv3.ArtifactRef{
		Schema:    schema,
		Digest:    "sha256:" + seed + strings.Repeat("a", 64-len(seed)),
		MediaType: "application/json",
		Size:      1,
		Locator:   "objects/" + seed,
	}
}

func TestOpenMigratesCompletedMinimalSliceDatabase(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "workflow.db")
	database, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	_, err = database.Exec(`
CREATE TABLE v3_runs (
  run_id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  plan_digest TEXT NOT NULL,
  plan_json BLOB NOT NULL,
  status TEXT NOT NULL,
  cancel_epoch INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE v3_nodes (
  run_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  task_kind TEXT NOT NULL,
  task_version TEXT NOT NULL,
  bundle_digest TEXT NOT NULL,
  entrypoint TEXT NOT NULL,
  task_abi TEXT NOT NULL,
  bindings_json BLOB NOT NULL,
  input_schemas_json BLOB NOT NULL,
  output_schemas_json BLOB NOT NULL,
  modules_json BLOB NOT NULL,
  status TEXT NOT NULL,
  attempt_count INTEGER NOT NULL DEFAULT 0,
  lease_token TEXT,
  lease_cancel_epoch INTEGER,
  lease_expires_at TEXT,
  PRIMARY KEY (run_id, node_key)
);
CREATE TABLE v3_attempts (
  run_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  attempt_no INTEGER NOT NULL,
  status TEXT NOT NULL,
  lease_token TEXT NOT NULL,
  cancel_epoch INTEGER NOT NULL,
  registry_generation TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT,
  failure_class TEXT,
  failure_code TEXT,
  failure_retryable INTEGER,
  failure_message TEXT,
  PRIMARY KEY (run_id, node_key, attempt_no)
);`)
	require.NoError(t, err)
	require.NoError(t, database.Close())

	store, err := Open(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	for table, columns := range map[string][]string{
		"v3_runs":     {"dispatch_count"},
		"v3_nodes":    {"resource_class", "max_attempts", "retry_backoff_ms", "ready_at"},
		"v3_attempts": {"resource_class"},
	} {
		for _, column := range columns {
			exists, columnErr := columnExists(ctx, store.db, table, column)
			require.NoError(t, columnErr)
			require.True(t, exists, "%s.%s", table, column)
		}
	}
}

func TestStorePersistsAppendOnlyAttemptsAndReopens(t *testing.T) {
	ctx := context.Background()
	registry, plan := storeFixture(t, "first")
	path := filepath.Join(t.TempDir(), "workflow.db")
	store, err := Open(ctx, path)
	require.NoError(t, err)
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	require.NoError(t, store.CreateRun(ctx, "run-1", plan, map[string]workflowv3.ArtifactRef{
		"source": artifactRef("source/v1", "1"),
	}, now))

	first, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	require.Equal(t, workflowv3.NodeKey("normalize"), first.NodeKey)
	inputs, err := store.ResolveInputs(ctx, *first)
	require.NoError(t, err)
	require.Equal(t, "source/v1", inputs["source"].Schema)
	require.NoError(t, store.Complete(ctx, *first, map[string]workflowv3.ArtifactRef{
		"dataset": artifactRef("dataset/v1", "2"),
	}, now.Add(time.Second)))

	second, err := store.LeaseNext(ctx, registry, now.Add(2*time.Second), time.Minute)
	require.NoError(t, err)
	require.Equal(t, workflowv3.NodeKey("validate"), second.NodeKey)
	require.NoError(t, store.Complete(ctx, *second, map[string]workflowv3.ArtifactRef{
		"validated": artifactRef("validated/v1", "3"),
	}, now.Add(3*time.Second)))
	require.NoError(t, store.Close())

	reopened, err := Open(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, reopened.Close()) })
	snapshot, err := reopened.Snapshot(ctx, "run-1")
	require.NoError(t, err)
	require.Equal(t, "succeeded", snapshot.Status)
	require.Equal(t, artifactRef("validated/v1", "3"), snapshot.Outputs["result"])
	require.Len(t, snapshot.Attempts, 2)
	require.Equal(t, "succeeded", snapshot.Attempts[0].Status)
	require.Equal(t, registry.Generation(), snapshot.Attempts[0].RegistryGeneration)
}

func TestStoreRejectsStaleCompletionAfterCancel(t *testing.T) {
	ctx := context.Background()
	registry, plan := storeFixture(t, "first")
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	now := time.Now().UTC()
	require.NoError(t, store.CreateRun(ctx, "run", plan, map[string]workflowv3.ArtifactRef{
		"source": artifactRef("source/v1", "1"),
	}, now))
	lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	require.NoError(t, store.Cancel(ctx, "run", now.Add(time.Second)))
	err = store.Complete(ctx, *lease, map[string]workflowv3.ArtifactRef{
		"dataset": artifactRef("dataset/v1", "2"),
	}, now.Add(2*time.Second))
	require.True(t, errors.Is(err, ErrStaleCompletion))
}

func TestStoreReclaimsExpiredLeaseAsNewAttempt(t *testing.T) {
	ctx := context.Background()
	registry, plan := storeFixture(t, "first")
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	now := time.Now().UTC()
	require.NoError(t, store.CreateRun(ctx, "run", plan, map[string]workflowv3.ArtifactRef{
		"source": artifactRef("source/v1", "1"),
	}, now))
	first, err := store.LeaseNext(ctx, registry, now, time.Second)
	require.NoError(t, err)
	second, err := store.LeaseNext(ctx, registry, now.Add(2*time.Second), time.Second)
	require.NoError(t, err)
	require.Equal(t, 2, second.Attempt)
	require.NotEqual(t, first.Token, second.Token)
	err = store.Complete(ctx, *first, map[string]workflowv3.ArtifactRef{
		"dataset": artifactRef("dataset/v1", "2"),
	}, now.Add(3*time.Second))
	require.True(t, errors.Is(err, ErrStaleCompletion))
	snapshot, err := store.Snapshot(ctx, "run")
	require.NoError(t, err)
	require.Equal(t, "lease_lost", snapshot.Attempts[0].Status)
	require.Equal(t, "running", snapshot.Attempts[1].Status)
}

func TestStoreRetryBackoffSurvivesReopen(t *testing.T) {
	ctx := context.Background()
	registry, plan := storeFixture(t, "first")
	path := filepath.Join(t.TempDir(), "workflow.db")
	store, err := Open(ctx, path)
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, store.CreateRun(ctx, "retry", plan, map[string]workflowv3.ArtifactRef{
		"source": artifactRef("source/v1", "retry"),
	}, now))
	lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	require.NoError(t, store.Fail(ctx, *lease, workflowv3.Failure{
		Class: "transport", Code: "TRANSIENT", Retryable: true,
		Message: "redacted transient failure",
	}, now.Add(time.Millisecond)))
	require.NoError(t, store.Close())

	reopened, err := Open(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, reopened.Close()) })
	queue, err := reopened.QueueSnapshot(
		ctx, registry, map[string]int{workflowv3.ResourceCPUDefault: 1},
		now.Add(500*time.Millisecond),
	)
	require.NoError(t, err)
	require.Equal(t, 1, queue.BlockedByReason["retry-backoff"])
	blocked, err := reopened.LeaseNext(
		ctx, registry, now.Add(500*time.Millisecond), time.Minute,
	)
	require.NoError(t, err)
	require.Nil(t, blocked)
	retried, err := reopened.LeaseNext(
		ctx, registry, now.Add(2*time.Second), time.Minute,
	)
	require.NoError(t, err)
	require.Equal(t, 2, retried.Attempt)
	snapshot, err := reopened.Snapshot(ctx, "retry")
	require.NoError(t, err)
	require.Equal(t, "failed", snapshot.Attempts[0].Status)
	require.True(t, snapshot.Attempts[0].Failure.Retryable)
}

func TestStoreResourceCapacityIsDatabaseScopedAcrossConnections(t *testing.T) {
	ctx := context.Background()
	registry, plan := storeFixture(t, "first")
	path := filepath.Join(t.TempDir(), "workflow.db")
	firstStore, err := Open(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, firstStore.Close()) })
	secondStore, err := Open(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, secondStore.Close()) })
	now := time.Now().UTC()
	for _, runID := range []workflowv3.RunID{"connection-a", "connection-b"} {
		require.NoError(t, firstStore.CreateRun(ctx, runID, plan, map[string]workflowv3.ArtifactRef{
			"source": artifactRef("source/v1", string(runID)),
		}, now))
		now = now.Add(time.Millisecond)
	}
	capacities := map[string]int{workflowv3.ResourceCPUDefault: 1}
	stores := []*Store{firstStore, secondStore}
	leases := make(chan *workflowv3.Lease, len(stores))
	errorsFound := make(chan error, len(stores))
	var wait sync.WaitGroup
	for _, store := range stores {
		store := store
		wait.Add(1)
		go func() {
			defer wait.Done()
			lease, leaseErr := store.LeaseNextWithResources(
				ctx, registry, capacities, now, time.Minute,
			)
			if leaseErr != nil {
				errorsFound <- leaseErr
				return
			}
			leases <- lease
		}()
	}
	wait.Wait()
	close(leases)
	close(errorsFound)
	for leaseErr := range errorsFound {
		require.NoError(t, leaseErr)
	}
	winners := 0
	for lease := range leases {
		if lease != nil {
			winners++
		}
	}
	require.Equal(t, 1, winners)
	queue, err := secondStore.QueueSnapshot(ctx, registry, capacities, now)
	require.NoError(t, err)
	require.Equal(t, 1, queue.ActiveByResource[workflowv3.ResourceCPUDefault])
	require.Equal(t, 1, queue.BlockedByReason["resource-capacity"])
}

func TestStoreConcurrentLeaseRaceHasSingleWinner(t *testing.T) {
	ctx := context.Background()
	registry, plan := storeFixture(t, "first")
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	now := time.Now().UTC()
	require.NoError(t, store.CreateRun(ctx, "run", plan, map[string]workflowv3.ArtifactRef{
		"source": artifactRef("source/v1", "1"),
	}, now))

	const contenders = 8
	var wait sync.WaitGroup
	leases := make(chan *workflowv3.Lease, contenders)
	errorsFound := make(chan error, contenders)
	for i := 0; i < contenders; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
			if err != nil {
				errorsFound <- err
				return
			}
			leases <- lease
		}()
	}
	wait.Wait()
	close(leases)
	close(errorsFound)
	for err := range errorsFound {
		require.NoError(t, err)
	}
	winners := 0
	for lease := range leases {
		if lease != nil {
			winners++
		}
	}
	require.Equal(t, 1, winners)
}

func TestStoreResourceCapacityAndFairnessAcrossRuns(t *testing.T) {
	ctx := context.Background()
	registry, plan := storeFixture(t, "first")
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	now := time.Now().UTC()
	for _, runID := range []workflowv3.RunID{"run-a", "run-b"} {
		require.NoError(t, store.CreateRun(ctx, runID, plan, map[string]workflowv3.ArtifactRef{
			"source": artifactRef("source/v1", string(runID)),
		}, now))
		now = now.Add(time.Millisecond)
	}
	capacities := map[string]int{workflowv3.ResourceCPUDefault: 1}
	first, err := store.LeaseNextWithResources(ctx, registry, capacities, now, time.Minute)
	require.NoError(t, err)
	require.Equal(t, workflowv3.RunID("run-a"), first.RunID)

	blocked, err := store.LeaseNextWithResources(ctx, registry, capacities, now, time.Minute)
	require.NoError(t, err)
	require.Nil(t, blocked)
	queue, err := store.QueueSnapshot(ctx, registry, capacities, now)
	require.NoError(t, err)
	require.Equal(t, 1, queue.ActiveByResource[workflowv3.ResourceCPUDefault])
	require.Equal(t, 2, queue.BlockedByReason["dependency"])
	require.Equal(t, 1, queue.BlockedByReason["resource-capacity"])

	require.NoError(t, store.Complete(ctx, *first, map[string]workflowv3.ArtifactRef{
		"dataset": artifactRef("dataset/v1", "fair-a"),
	}, now.Add(time.Second)))
	second, err := store.LeaseNextWithResources(
		ctx, registry, capacities, now.Add(2*time.Second), time.Minute,
	)
	require.NoError(t, err)
	require.Equal(t, workflowv3.RunID("run-b"), second.RunID)
	require.Equal(t, workflowv3.NodeKey("normalize"), second.NodeKey)
}

func TestStoreDoesNotLeaseImplementationFromDifferentBundle(t *testing.T) {
	ctx := context.Background()
	_, plan := storeFixture(t, "first")
	wrongRegistry, _ := storeFixture(t, "different source")
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	now := time.Now().UTC()
	require.NoError(t, store.CreateRun(ctx, "run", plan, map[string]workflowv3.ArtifactRef{
		"source": artifactRef("source/v1", "1"),
	}, now))
	lease, err := store.LeaseNext(ctx, wrongRegistry, now, time.Minute)
	require.NoError(t, err)
	require.Nil(t, lease)
}
