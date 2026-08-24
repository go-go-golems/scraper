package workflowv3sqlite

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/stretchr/testify/require"
)

func budgetStoreFixture(t *testing.T, nodes, limit int) (*workflowv3.SealedRegistry, workflowv3.WorkflowPlan) {
	return budgetStoreFixtureWithPolicy(t, nodes, limit, workflowv3.BudgetExhaustBlock)
}

func budgetStoreFixtureWithPolicy(t *testing.T, nodes, limit int, policy string) (*workflowv3.SealedRegistry, workflowv3.WorkflowPlan) {
	t.Helper()
	maximum := &workflowv3.BudgetClaim{
		Account: "provider", OnExhausted: workflowv3.BudgetExhaustBlock,
		Reserve: []workflowv3.BudgetAmount{{Dimension: "requests", Units: 1}},
	}
	bundle, err := workflowv3.NewBundle(workflowv3.BundleManifest{
		Name: "budget", Version: "1", ABI: workflowv3.TaskABI,
		Tasks: []workflowv3.BundleTask{{
			TaskKey:    workflowv3.TaskKey{Kind: "budget.call", Version: "v1"},
			Entrypoint: "tasks.cjs#run", Inputs: map[string]string{"source": "source/v1"},
			Outputs: map[string]string{"output": "output/v1"}, ResourceClass: "network.provider",
			Retry: workflowv3.RetryPolicy{MaxAttempts: 3}, BudgetMaximum: maximum,
		}},
	}, map[string][]byte{"tasks.cjs": []byte(`exports.run = () => ({})`)})
	require.NoError(t, err)
	builder := workflowv3.NewRegistryBuilder()
	require.NoError(t, builder.AddBundle(bundle))
	registry, err := builder.Seal()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	claim := &workflowv3.BudgetClaim{
		Account: "provider", OnExhausted: policy,
		Reserve: []workflowv3.BudgetAmount{{Dimension: "requests", Units: 1}},
	}
	if policy == workflowv3.BudgetExhaustRequireApproval {
		claim.ApprovalGate = "budget-approval"
	}
	ir := workflowv3.WorkflowIR{
		Schema: workflowv3.IRSchema, Name: "budget",
		Inputs: []workflowv3.IRInput{{Name: "source", Schema: "source/v1"}},
		Budgets: []workflowv3.BudgetAccount{{
			Account: "provider", PolicyDigest: "sha256:" + strings.Repeat("a", 64),
			Limits: []workflowv3.BudgetAmount{{Dimension: "requests", Units: int64(limit)}},
		}},
	}
	if policy == workflowv3.BudgetExhaustRequireApproval {
		ir.Gates = []workflowv3.IRGate{{
			Key:    "budget-approval",
			Policy: workflowv3.GatePolicy{DecisionSchema: "budget-approval/v1", RequiredRole: "budget.operator", OnReject: workflowv3.GateFailRun, OnExpire: workflowv3.GateFailRun},
		}}
	}
	for index := range nodes {
		key := workflowv3.NodeKey("call-" + string(rune('a'+index)))
		ir.Nodes = append(ir.Nodes, workflowv3.IRNode{
			Key: key, Task: workflowv3.TaskKey{Kind: "budget.call", Version: "v1"},
			Bindings: map[string]workflowv3.ValueRef{"source": {Source: "input", Name: "source", Schema: "source/v1"}},
			Budget:   claim,
		})
	}
	ir.Outputs = []workflowv3.IROutput{{
		Name: "output", Value: workflowv3.ValueRef{Source: "node-output", NodeKey: ir.Nodes[0].Key, Port: "output", Schema: "output/v1"},
	}}
	plan, err := workflowv3.Compile(ir, catalog)
	require.NoError(t, err)
	return registry, plan
}

func createBudgetRun(t *testing.T, store *Store, plan workflowv3.WorkflowPlan, runID workflowv3.RunID, now time.Time) {
	t.Helper()
	input := workflowv3.ArtifactRef{
		Schema: "source/v1", Digest: "sha256:" + strings.Repeat("b", 64),
		MediaType: "application/json", Size: 2, Locator: "cas://source",
	}
	require.NoError(t, store.CreateRun(context.Background(), runID, plan, map[string]workflowv3.ArtifactRef{"source": input}, now))
}

func TestBudgetReservationIsAtomicAcrossConnectionsAndIncreaseUnblocks(t *testing.T) {
	ctx := context.Background()
	registry, plan := budgetStoreFixture(t, 2, 1)
	path := filepath.Join(t.TempDir(), "workflow.db")
	storeA, err := Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, storeA.Close()) }()
	storeB, err := Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, storeB.Close()) }()
	now := time.Now().UTC()
	createBudgetRun(t, storeA, plan, "race", now)

	leases := make(chan *workflowv3.Lease, 2)
	errors := make(chan error, 2)
	start := make(chan struct{})
	var wait sync.WaitGroup
	for _, store := range []*Store{storeA, storeB} {
		wait.Add(1)
		go func(candidate *Store) {
			defer wait.Done()
			<-start
			lease, leaseErr := candidate.LeaseNext(ctx, registry, now, time.Minute)
			leases <- lease
			errors <- leaseErr
		}(store)
	}
	close(start)
	wait.Wait()
	close(leases)
	close(errors)
	for leaseErr := range errors {
		require.NoError(t, leaseErr)
	}
	var winner *workflowv3.Lease
	leased := 0
	for lease := range leases {
		if lease != nil {
			winner = lease
			leased++
		}
	}
	require.Equal(t, 1, leased)
	progress, err := storeA.BudgetSnapshot(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), progress[0].Reserved)
	require.Zero(t, progress[0].Remaining)

	require.NoError(t, storeA.CompleteWithUsage(ctx, *winner, map[string]workflowv3.ArtifactRef{
		"output": {Schema: "output/v1", Digest: "sha256:" + strings.Repeat("c", 64), MediaType: "application/json", Size: 2, Locator: "cas://output"},
	}, []workflowv3.BudgetAmount{{Dimension: "requests", Units: 1}}, now.Add(time.Second)))
	blocked, err := storeB.LeaseNext(ctx, registry, now.Add(2*time.Second), time.Minute)
	require.NoError(t, err)
	require.Nil(t, blocked)
	queue, err := storeB.QueueSnapshot(ctx, registry, nil, now.Add(2*time.Second))
	require.NoError(t, err)
	require.Equal(t, 1, queue.BlockedByReason["budget:provider:requests"])
	require.NoError(t, storeA.IncreaseBudget(ctx, "race", "provider", "requests", 1, 1, "operator@example", now.Add(3*time.Second)))
	require.ErrorContains(t, storeA.IncreaseBudget(ctx, "race", "provider", "requests", 1, 1, "operator@example", now.Add(4*time.Second)), "version conflict")
	second, err := storeB.LeaseNext(ctx, registry, now.Add(5*time.Second), time.Minute)
	require.NoError(t, err)
	require.NotNil(t, second)
	require.NotEqual(t, winner.NodeKey, second.NodeKey)
	runID := workflowv3.RunID("race")
	operational, err := storeA.OperationalSnapshot(
		ctx, &runID, registry, map[string]int{"network.provider": 2}, now.Add(5*time.Second),
	)
	require.NoError(t, err)
	require.Equal(t, 1, operational.NodeStatuses["succeeded"])
	require.Equal(t, 1, operational.NodeStatuses["running"])
	require.Equal(t, 1, operational.AttemptStatuses["succeeded"])
	require.Equal(t, 1, operational.AttemptStatuses["running"])
	require.Len(t, operational.Budgets, 1)
	require.Equal(t, int64(1), operational.Budgets[0].Used)
	require.Equal(t, int64(1), operational.Budgets[0].Reserved)
	require.Equal(t, 1, operational.Queue.ActiveByResource["network.provider"])
	require.NoError(t, storeB.CompleteWithUsage(ctx, *second, map[string]workflowv3.ArtifactRef{
		"output": {Schema: "output/v1", Digest: "sha256:" + strings.Repeat("8", 64), MediaType: "application/json", Size: 2, Locator: "cas://second"},
	}, []workflowv3.BudgetAmount{{Dimension: "requests", Units: 1}}, now.Add(6*time.Second)))
	events, err := storeA.EventsAfter(ctx, operational.EventSequence, &runID, 100)
	require.NoError(t, err)
	require.NotEmpty(t, events)
	for _, event := range events {
		require.Greater(t, event.Sequence, operational.EventSequence)
	}
	require.Contains(t, eventTypes(events), "budget.settled")
}

func eventTypes(events []workflowv3.OperationalEvent) []string {
	ret := make([]string, 0, len(events))
	for _, event := range events {
		ret = append(ret, event.Type)
	}
	return ret
}

func TestBudgetReopenRejectsInconsistentReservationTotals(t *testing.T) {
	ctx := context.Background()
	registry, plan := budgetStoreFixture(t, 1, 1)
	path := filepath.Join(t.TempDir(), "workflow.db")
	store, err := Open(ctx, path)
	require.NoError(t, err)
	now := time.Now().UTC()
	createBudgetRun(t, store, plan, "corrupt", now)
	lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	_, err = store.db.ExecContext(ctx, `
UPDATE v3_budget_accounts SET reserved_units = 0 WHERE run_id = 'corrupt'`)
	require.NoError(t, err)
	require.NoError(t, store.Close())
	_, err = Open(ctx, path)
	require.ErrorContains(t, err, "BUDGET_ACCOUNT_INVARIANT")
}

func TestBudgetLeaseLossChargesConservativelyAndRetryReservesFresh(t *testing.T) {
	ctx := context.Background()
	registry, plan := budgetStoreFixture(t, 1, 2)
	path := filepath.Join(t.TempDir(), "workflow.db")
	store, err := Open(ctx, path)
	require.NoError(t, err)
	now := time.Now().UTC()
	createBudgetRun(t, store, plan, "lease-loss", now)
	first, err := store.LeaseNext(ctx, registry, now, time.Second)
	require.NoError(t, err)
	require.NoError(t, store.Close())

	store, err = Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	second, err := store.LeaseNext(ctx, registry, now.Add(2*time.Second), time.Minute)
	require.NoError(t, err)
	require.Equal(t, first.NodeKey, second.NodeKey)
	require.Equal(t, 2, second.Attempt)
	progress, err := store.BudgetSnapshot(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), progress[0].Used)
	require.Equal(t, int64(1), progress[0].Reserved)
	var conservative int
	require.NoError(t, store.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM v3_budget_reservations
WHERE run_id = 'lease-loss' AND status = 'conservative'`).Scan(&conservative))
	require.Equal(t, 1, conservative)
}

func TestOperationalSnapshotRateBoundaryUsesParsedTime(t *testing.T) {
	ctx := context.Background()
	registry, plan := budgetStoreFixture(t, 1, 1)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	now := time.Date(2026, 7, 21, 12, 0, 0, 123456789, time.UTC)
	createBudgetRun(t, store, plan, "rate", now)
	lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	require.NoError(t, store.CompleteWithUsage(ctx, *lease, map[string]workflowv3.ArtifactRef{
		"output": {Schema: "output/v1", Digest: "sha256:" + strings.Repeat("7", 64), MediaType: "application/json", Size: 2, Locator: "cas://output"},
	}, []workflowv3.BudgetAmount{{Dimension: "requests", Units: 1}}, now))
	atBoundary, err := store.OperationalSnapshot(ctx, nil, registry, nil, now.Add(60*time.Second))
	require.NoError(t, err)
	require.Equal(t, 1, atBoundary.Rates[0].Terminal)
	afterBoundary, err := store.OperationalSnapshot(ctx, nil, registry, nil, now.Add(61*time.Second))
	require.NoError(t, err)
	require.Zero(t, afterBoundary.Rates[0].Terminal)
}

func TestBudgetCompletionCancellationRaceSettlesExactlyOnce(t *testing.T) {
	ctx := context.Background()
	registry, plan := budgetStoreFixture(t, 1, 1)
	path := filepath.Join(t.TempDir(), "workflow.db")
	storeA, err := Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, storeA.Close()) }()
	storeB, err := Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, storeB.Close()) }()
	now := time.Now().UTC()
	createBudgetRun(t, storeA, plan, "settle-race", now)
	lease, err := storeA.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		<-start
		results <- storeA.CompleteWithUsage(ctx, *lease, map[string]workflowv3.ArtifactRef{
			"output": {Schema: "output/v1", Digest: "sha256:" + strings.Repeat("9", 64), MediaType: "application/json", Size: 2, Locator: "cas://output"},
		}, []workflowv3.BudgetAmount{{Dimension: "requests", Units: 1}}, now.Add(time.Second))
	}()
	go func() {
		<-start
		results <- storeB.Cancel(ctx, "settle-race", now.Add(time.Second))
	}()
	close(start)
	firstErr, secondErr := <-results, <-results
	require.True(t, (firstErr == nil) != (secondErr == nil), "exactly one terminal transaction must win: %v / %v", firstErr, secondErr)
	progress, err := storeA.BudgetSnapshot(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), progress[0].Used)
	require.Zero(t, progress[0].Reserved)
	var terminalReservations int
	require.NoError(t, storeA.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM v3_budget_reservations
WHERE run_id = 'settle-race' AND status != 'reserved'`).Scan(&terminalReservations))
	require.Equal(t, 1, terminalReservations)
}

func TestBudgetRetryCreatesFreshImmutableReservation(t *testing.T) {
	ctx := context.Background()
	registry, plan := budgetStoreFixture(t, 1, 2)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	now := time.Now().UTC()
	createBudgetRun(t, store, plan, "retry", now)
	first, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	require.NoError(t, store.Fail(ctx, *first, workflowv3.Failure{
		Class: "rate-limit", Code: "BUDGET_FIXTURE_RETRY", Retryable: true,
		Message: "retry fixture",
	}, now.Add(time.Second)))
	second, err := store.LeaseNext(ctx, registry, now.Add(2*time.Second), time.Minute)
	require.NoError(t, err)
	require.Equal(t, 2, second.Attempt)
	var reservations int
	require.NoError(t, store.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM v3_budget_reservations WHERE run_id = 'retry'`).Scan(&reservations))
	require.Equal(t, 2, reservations)
	progress, err := store.BudgetSnapshot(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), progress[0].Used)
	require.Equal(t, int64(1), progress[0].Reserved)
}

func TestBudgetExhaustionFailRunAndRequireApprovalAreLeaseFree(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		name, policy, status, reason string
	}{
		{name: "fail", policy: workflowv3.BudgetExhaustFailRun, status: "failed"},
		{name: "approval", policy: workflowv3.BudgetExhaustRequireApproval, status: "running", reason: "budget-approval:provider:requests"},
	} {
		t.Run(test.name, func(t *testing.T) {
			registry, plan := budgetStoreFixtureWithPolicy(t, 1, 0, test.policy)
			store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
			require.NoError(t, err)
			defer func() { require.NoError(t, store.Close()) }()
			now := time.Now().UTC()
			createBudgetRun(t, store, plan, workflowv3.RunID(test.name), now)
			lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
			require.NoError(t, err)
			require.Nil(t, lease)
			snapshot, err := store.Snapshot(ctx, workflowv3.RunID(test.name))
			require.NoError(t, err)
			require.Equal(t, test.status, snapshot.Status)
			require.Empty(t, snapshot.Attempts)
			if test.policy == workflowv3.BudgetExhaustRequireApproval {
				advanced, err := store.AdvanceOneGate(ctx, now)
				require.NoError(t, err)
				require.True(t, advanced)
				operational, err := store.OperationalSnapshot(ctx, nil, registry, nil, now)
				require.NoError(t, err)
				require.Equal(t, 1, operational.GateStatuses["waiting"])
				require.Empty(t, operational.AttemptStatuses)
			}
			if test.reason != "" {
				queue, err := store.QueueSnapshot(ctx, registry, nil, now)
				require.NoError(t, err)
				require.Equal(t, 1, queue.BlockedByReason[test.reason])
			}
		})
	}
}

func TestUnusedBudgetApprovalGateDoesNotBlockSuccessfulRun(t *testing.T) {
	ctx := context.Background()
	registry, plan := budgetStoreFixtureWithPolicy(t, 1, 1, workflowv3.BudgetExhaustRequireApproval)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	now := time.Now().UTC()
	createBudgetRun(t, store, plan, "budget-gate-unused", now)
	advanced, err := store.AdvanceOneGate(ctx, now)
	require.NoError(t, err)
	require.False(t, advanced)
	lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NoError(t, store.CompleteWithUsage(ctx, *lease, map[string]workflowv3.ArtifactRef{
		"output": {Schema: "output/v1", Digest: "sha256:" + strings.Repeat("5", 64), MediaType: "application/json", Size: 2, Locator: "cas://output"},
	}, []workflowv3.BudgetAmount{{Dimension: "requests", Units: 1}}, now.Add(time.Second)))
	snapshot, err := store.Snapshot(ctx, "budget-gate-unused")
	require.NoError(t, err)
	require.Equal(t, "succeeded", snapshot.Status)
	operational, err := store.OperationalSnapshot(ctx, nil, registry, nil, now.Add(time.Second))
	require.NoError(t, err)
	require.Equal(t, "pending", operational.Gates[0].Status)
	require.True(t, operational.Gates[0].BudgetActivation)
}

func TestBudgetApprovalGateAndAccountIncreaseContinueExactlyOnce(t *testing.T) {
	ctx := context.Background()
	registry, plan := budgetStoreFixtureWithPolicy(t, 1, 0, workflowv3.BudgetExhaustRequireApproval)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	now := time.Now().UTC()
	createBudgetRun(t, store, plan, "budget-gate", now)
	lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	require.Nil(t, lease)
	advanced, err := store.AdvanceOneGate(ctx, now)
	require.NoError(t, err)
	require.True(t, advanced)
	require.NoError(t, store.IncreaseBudget(ctx, "budget-gate", "provider", "requests", 1, 1, "budget-operator", now.Add(time.Second)))
	decision := workflowv3.ArtifactRef{Schema: "budget-approval/v1", Digest: "sha256:" + strings.Repeat("6", 64), MediaType: "application/json", Size: 2, Locator: "cas://budget-approval"}
	command := workflowv3.GateDecisionCommand{RunID: "budget-gate", GateKey: "budget-approval", ExpectedVersion: 1, Decision: "approve", DecisionCode: "BUDGET_INCREASE_APPROVED", ActorID: "budget-operator", AuthorizedRole: "budget.operator", DecisionRef: &decision}
	require.NoError(t, store.DecideGate(ctx, command, now.Add(2*time.Second)))
	lease, err = store.LeaseNext(ctx, registry, now.Add(3*time.Second), time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	second, err := store.LeaseNext(ctx, registry, now.Add(3*time.Second), time.Minute)
	require.NoError(t, err)
	require.Nil(t, second)
	var attempts int
	require.NoError(t, store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM v3_attempts WHERE run_id = 'budget-gate'`).Scan(&attempts))
	require.Equal(t, 1, attempts)
}

func TestBudgetUsageAboveReservationRollsBackThenChargesConservatively(t *testing.T) {
	ctx := context.Background()
	registry, plan := budgetStoreFixture(t, 1, 2)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	now := time.Now().UTC()
	createBudgetRun(t, store, plan, "over", now)
	lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	err = store.CompleteWithUsage(ctx, *lease, map[string]workflowv3.ArtifactRef{
		"output": {Schema: "output/v1", Digest: "sha256:" + strings.Repeat("d", 64), MediaType: "application/json", Size: 2, Locator: "cas://output"},
	}, []workflowv3.BudgetAmount{{Dimension: "requests", Units: 2}}, now.Add(time.Second))
	require.ErrorIs(t, err, ErrBudgetUsageExceedsReservation)
	progress, err := store.BudgetSnapshot(ctx, nil)
	require.NoError(t, err)
	require.Zero(t, progress[0].Used)
	require.Equal(t, int64(1), progress[0].Reserved)
	require.NoError(t, store.Fail(ctx, *lease, workflowv3.Failure{
		Class: "budget", Code: "BUDGET_USAGE_EXCEEDS_RESERVATION", Retryable: false,
		Message: "usage exceeded reservation",
	}, now.Add(2*time.Second)))
	progress, err = store.BudgetSnapshot(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), progress[0].Used)
	require.Zero(t, progress[0].Reserved)
}
