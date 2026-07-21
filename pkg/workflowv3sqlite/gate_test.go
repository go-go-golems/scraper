package workflowv3sqlite

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/stretchr/testify/require"
)

func gateStoreFixture(t *testing.T, timeoutMillis int64, withTasks bool) (*workflowv3.SealedRegistry, workflowv3.WorkflowPlan) {
	t.Helper()
	tasks := []workflowv3.BundleTask{{
		TaskKey: workflowv3.TaskKey{Kind: "gate.prepare", Version: "v1"}, Entrypoint: "tasks.cjs#prepare",
		Inputs: map[string]string{"source": "source/v1"}, Outputs: map[string]string{"prepared": "prepared/v1"}, ResourceClass: "cpu.gate",
	}, {
		TaskKey: workflowv3.TaskKey{Kind: "gate.publish", Version: "v1"}, Entrypoint: "tasks.cjs#publish",
		Inputs: map[string]string{"decision": "approval-decision/v1"}, Outputs: map[string]string{"published": "published/v1"}, ResourceClass: "cpu.gate",
	}}
	bundle, err := workflowv3.NewBundle(workflowv3.BundleManifest{
		Name: "gate", Version: "1", ABI: workflowv3.TaskABI, Tasks: tasks,
	}, map[string][]byte{"tasks.cjs": []byte(`exports.prepare = value => value; exports.publish = value => value`)})
	require.NoError(t, err)
	builder := workflowv3.NewRegistryBuilder()
	require.NoError(t, builder.AddBundle(bundle))
	registry, err := builder.Seal()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	ir := workflowv3.WorkflowIR{Schema: workflowv3.IRSchema, Name: "gate"}
	var dependencies []workflowv3.NodeKey
	if withTasks {
		ir.Inputs = []workflowv3.IRInput{{Name: "source", Schema: "source/v1"}}
		ir.Nodes = append(ir.Nodes, workflowv3.IRNode{
			Key: "prepare", Task: workflowv3.TaskKey{Kind: "gate.prepare", Version: "v1"},
			Bindings: map[string]workflowv3.ValueRef{"source": {Source: "input", Name: "source", Schema: "source/v1"}},
		})
		dependencies = []workflowv3.NodeKey{"prepare"}
	}
	ir.Gates = []workflowv3.IRGate{{
		Key: "review", DependsOn: dependencies,
		Policy: workflowv3.GatePolicy{DecisionSchema: "approval-decision/v1", RequiredRole: "reviewer.primary", OnReject: workflowv3.GateFailRun, OnExpire: workflowv3.GateFailRun, TimeoutMillis: timeoutMillis},
	}}
	if withTasks {
		ir.Nodes = append(ir.Nodes, workflowv3.IRNode{
			Key: "publish", Task: workflowv3.TaskKey{Kind: "gate.publish", Version: "v1"},
			Bindings: map[string]workflowv3.ValueRef{"decision": {Source: "gate-output", GateKey: "review", Schema: "approval-decision/v1"}},
		})
		ir.Outputs = []workflowv3.IROutput{{Name: "published", Value: workflowv3.ValueRef{Source: "node-output", NodeKey: "publish", Port: "published", Schema: "published/v1"}}}
	} else {
		ir.Outputs = []workflowv3.IROutput{{Name: "decision", Value: workflowv3.ValueRef{Source: "gate-output", GateKey: "review", Schema: "approval-decision/v1"}}}
	}
	plan, err := workflowv3.Compile(ir, catalog)
	require.NoError(t, err)
	return registry, plan
}

func gateInput() workflowv3.ArtifactRef {
	return workflowv3.ArtifactRef{Schema: "source/v1", Digest: "sha256:" + strings.Repeat("1", 64), MediaType: "application/json", Size: 2, Locator: "cas://source"}
}

func gateDecisionRef(seed string) workflowv3.ArtifactRef {
	return workflowv3.ArtifactRef{Schema: "approval-decision/v1", Digest: "sha256:" + strings.Repeat(seed, 64), MediaType: "application/json", Size: 20, Locator: "cas://decision-" + seed}
}

func TestGateOperationalDetailsAreBoundedAndMarkedTruncated(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	now := time.Now().UTC()
	tx, err := store.db.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `
INSERT INTO v3_runs(run_id, name, plan_digest, plan_json, status, created_at, updated_at)
VALUES ('many-gates', 'many-gates', ?, '{}', 'running', ?, ?)`,
		"sha256:"+strings.Repeat("9", 64), formatTime(now), formatTime(now))
	require.NoError(t, err)
	statement, err := tx.PrepareContext(ctx, `
INSERT INTO v3_gates(
  run_id, gate_key, status, version, policy_digest, decision_schema,
  required_role, on_reject, on_expire, timeout_ms, budget_activation
) VALUES ('many-gates', ?, 'pending', 0, ?, 'decision/v1',
  'reviewer.primary', 'fail-run', 'fail-run', 0, 0)`)
	require.NoError(t, err)
	for index := 0; index < 1001; index++ {
		_, err = statement.ExecContext(ctx, fmt.Sprintf("gate-%04d", index), "sha256:"+strings.Repeat("8", 64))
		require.NoError(t, err)
	}
	require.NoError(t, statement.Close())
	require.NoError(t, tx.Commit())
	snapshot, err := store.OperationalSnapshot(ctx, nil, nil, nil, now)
	require.NoError(t, err)
	require.Len(t, snapshot.Gates, 1000)
	require.True(t, snapshot.GatesTruncated)
	page, err := store.GatePage(ctx, "many-gates", "gate-0998", 10, now)
	require.NoError(t, err)
	require.Len(t, page, 2)
}

func TestGatePendingToWaitingRaceTransitionsOnce(t *testing.T) {
	ctx := context.Background()
	_, plan := gateStoreFixture(t, 60_000, false)
	path := filepath.Join(t.TempDir(), "workflow.db")
	storeA, err := Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, storeA.Close()) }()
	storeB, err := Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, storeB.Close()) }()
	now := time.Now().UTC()
	require.NoError(t, storeA.CreateRun(ctx, "waiting-race", plan, nil, now))
	start := make(chan struct{})
	results := make(chan bool, 2)
	errors := make(chan error, 2)
	for _, store := range []*Store{storeA, storeB} {
		go func(store *Store) {
			<-start
			advanced, advanceErr := store.AdvanceOneGate(ctx, now)
			results <- advanced
			errors <- advanceErr
		}(store)
	}
	close(start)
	advanced := 0
	for range 2 {
		if <-results {
			advanced++
		}
		require.NoError(t, <-errors)
	}
	require.Equal(t, 1, advanced)
	var attempts int
	require.NoError(t, storeA.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM v3_attempts WHERE run_id = 'waiting-race'`).Scan(&attempts))
	require.Zero(t, attempts)
}

func TestGateWaitsWithoutLeaseAcrossReopenAndApprovalContinues(t *testing.T) {
	ctx := context.Background()
	registry, plan := gateStoreFixture(t, 60_000, true)
	path := filepath.Join(t.TempDir(), "workflow.db")
	store, err := Open(ctx, path)
	require.NoError(t, err)
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	require.NoError(t, store.CreateRun(ctx, "approval", plan, map[string]workflowv3.ArtifactRef{"source": gateInput()}, now))
	advanced, err := store.AdvanceOneGate(ctx, now)
	require.NoError(t, err)
	require.False(t, advanced)
	prepare, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	require.Equal(t, workflowv3.NodeKey("prepare"), prepare.NodeKey)
	require.NoError(t, store.Complete(ctx, *prepare, map[string]workflowv3.ArtifactRef{"prepared": artifactRef("prepared/v1", "prepared")}, now.Add(time.Second)))
	advanced, err = store.AdvanceOneGate(ctx, now.Add(2*time.Second))
	require.NoError(t, err)
	require.True(t, advanced)
	lease, err := store.LeaseNext(ctx, registry, now.Add(3*time.Second), time.Minute)
	require.NoError(t, err)
	require.Nil(t, lease)
	snapshot, err := store.Snapshot(ctx, "approval")
	require.NoError(t, err)
	require.Equal(t, "running", snapshot.Status)
	require.Len(t, snapshot.Attempts, 1)
	operational, err := store.OperationalSnapshot(ctx, nil, registry, map[string]int{"cpu.gate": 1}, now.Add(3*time.Second))
	require.NoError(t, err)
	require.Equal(t, 1, operational.GateStatuses["waiting"])
	require.Len(t, operational.Gates, 1)
	require.Equal(t, int64(1), operational.Gates[0].Version)
	require.Empty(t, operational.Queue.ActiveByResource)
	require.Equal(t, 1, operational.Queue.BlockedByReason["gate-dependency"])
	page, err := store.GatePage(ctx, "approval", "", 1, now.Add(3*time.Second))
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, workflowv3.NodeKey("review"), page[0].GateKey)
	nextPage, err := store.GatePage(ctx, "approval", "review", 1, now.Add(3*time.Second))
	require.NoError(t, err)
	require.Empty(t, nextPage)
	require.Empty(t, operational.Budgets)
	require.NoError(t, store.Close())

	store, err = Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	operational, err = store.OperationalSnapshot(ctx, nil, registry, nil, now.Add(4*time.Second))
	require.NoError(t, err)
	require.Equal(t, int64(1), operational.Gates[0].Version)
	require.NotNil(t, operational.Gates[0].ExpiresInMS)
	decision := gateDecisionRef("2")
	wrongSchema := decision
	wrongSchema.Schema = "wrong-decision/v1"
	wrongCommand := workflowv3.GateDecisionCommand{RunID: "approval", GateKey: "review", ExpectedVersion: 1, Decision: "approve", DecisionCode: "APPROVED_FOR_TEST", ActorID: "reviewer@example", AuthorizedRole: "reviewer.primary", DecisionRef: &wrongSchema}
	require.ErrorContains(t, store.DecideGate(ctx, wrongCommand, now.Add(5*time.Second)), "schema mismatch")
	staleCommand := wrongCommand
	staleCommand.ExpectedVersion = 2
	staleCommand.DecisionRef = &decision
	require.ErrorIs(t, store.DecideGate(ctx, staleCommand, now.Add(5*time.Second)), ErrGateVersionConflict)
	command := workflowv3.GateDecisionCommand{RunID: "approval", GateKey: "review", ExpectedVersion: 1, Decision: "approve", DecisionCode: "APPROVED_FOR_TEST", ActorID: "reviewer@example", AuthorizedRole: "wrong.role", DecisionRef: &decision}
	require.ErrorIs(t, store.DecideGate(ctx, command, now.Add(5*time.Second)), ErrGateAuthorization)
	command.AuthorizedRole = "reviewer.primary"
	require.NoError(t, store.DecideGate(ctx, command, now.Add(5*time.Second)))
	require.NoError(t, store.DecideGate(ctx, command, now.Add(6*time.Second)), "identical decision is idempotent")
	unauthorizedRetry := command
	unauthorizedRetry.AuthorizedRole = "reviewer.secondary"
	require.ErrorIs(t, store.DecideGate(ctx, unauthorizedRetry, now.Add(6*time.Second)), ErrGateAuthorization)
	conflict := command
	conflict.Decision = "reject"
	conflict.DecisionCode = "REJECTED_FOR_TEST"
	conflict.DecisionRef = nil
	require.ErrorIs(t, store.DecideGate(ctx, conflict, now.Add(6*time.Second)), ErrGateAlreadyDecided)

	publish, err := store.LeaseNext(ctx, registry, now.Add(7*time.Second), time.Minute)
	require.NoError(t, err)
	require.Equal(t, workflowv3.NodeKey("publish"), publish.NodeKey)
	inputs, err := store.ResolveInputs(ctx, *publish)
	require.NoError(t, err)
	require.Equal(t, decision, inputs["decision"])
	require.NoError(t, store.Complete(ctx, *publish, map[string]workflowv3.ArtifactRef{"published": artifactRef("published/v1", "published")}, now.Add(8*time.Second)))
	snapshot, err = store.Snapshot(ctx, "approval")
	require.NoError(t, err)
	require.Equal(t, "succeeded", snapshot.Status)
	require.Len(t, snapshot.Attempts, 2)
}

func TestGateExpiryDeadlineSurvivesReopen(t *testing.T) {
	ctx := context.Background()
	_, plan := gateStoreFixture(t, 1_000, false)
	path := filepath.Join(t.TempDir(), "workflow.db")
	store, err := Open(ctx, path)
	require.NoError(t, err)
	now := time.Now().UTC()
	require.NoError(t, store.CreateRun(ctx, "expiry-reopen", plan, nil, now))
	advanced, err := store.AdvanceOneGate(ctx, now)
	require.NoError(t, err)
	require.True(t, advanced)
	require.NoError(t, store.Close())
	store, err = Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	expired, err := store.ExpireDueGates(ctx, now.Add(2*time.Second))
	require.NoError(t, err)
	require.Equal(t, 1, expired)
	snapshot, err := store.Snapshot(ctx, "expiry-reopen")
	require.NoError(t, err)
	require.Equal(t, "failed", snapshot.Status)
	operational, err := store.OperationalSnapshot(ctx, nil, nil, nil, now.Add(2*time.Second))
	require.NoError(t, err)
	require.Equal(t, "expired", operational.Gates[0].Status)
	require.Equal(t, int64(2), operational.Gates[0].Version)
}

func TestGateRejectExpireAndCancelAreTerminalWithoutAttempts(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		name       string
		action     func(*Store, time.Time) error
		gateStatus string
		runStatus  string
	}{
		{name: "reject", gateStatus: "rejected", runStatus: "failed", action: func(store *Store, now time.Time) error {
			return store.DecideGate(ctx, workflowv3.GateDecisionCommand{RunID: "reject", GateKey: "review", ExpectedVersion: 1, Decision: "reject", DecisionCode: "REJECTED_BY_OPERATOR", ActorID: "reviewer@example", AuthorizedRole: "reviewer.primary"}, now)
		}},
		{name: "expire", gateStatus: "expired", runStatus: "failed", action: func(store *Store, now time.Time) error {
			_, err := store.ExpireDueGates(ctx, now.Add(2*time.Second))
			return err
		}},
		{name: "cancel", gateStatus: "canceled", runStatus: "canceled", action: func(store *Store, now time.Time) error {
			return store.Cancel(ctx, "cancel", now)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			registry, plan := gateStoreFixture(t, 1_000, false)
			store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
			require.NoError(t, err)
			defer func() { require.NoError(t, store.Close()) }()
			now := time.Now().UTC()
			require.NoError(t, store.CreateRun(ctx, workflowv3.RunID(test.name), plan, nil, now))
			advanced, err := store.AdvanceOneGate(ctx, now)
			require.NoError(t, err)
			require.True(t, advanced)
			require.NoError(t, test.action(store, now))
			snapshot, err := store.Snapshot(ctx, workflowv3.RunID(test.name))
			require.NoError(t, err)
			require.Equal(t, test.runStatus, snapshot.Status)
			require.Empty(t, snapshot.Attempts)
			operational, err := store.OperationalSnapshot(ctx, nil, registry, nil, now.Add(3*time.Second))
			require.NoError(t, err)
			require.Equal(t, 1, operational.GateStatuses[test.gateStatus])
			decision := gateDecisionRef("3")
			err = store.DecideGate(ctx, workflowv3.GateDecisionCommand{RunID: workflowv3.RunID(test.name), GateKey: "review", ExpectedVersion: 1, Decision: "approve", DecisionCode: "LATE_APPROVAL", ActorID: "reviewer@example", AuthorizedRole: "reviewer.primary", DecisionRef: &decision}, now.Add(4*time.Second))
			require.Error(t, err)
		})
	}
}

func TestGateApprovalRacesExpiryAndCancellationWithoutRevival(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		name    string
		timeout int64
		compete func(*Store, time.Time) error
	}{
		{name: "expiry", timeout: 1, compete: func(store *Store, now time.Time) error {
			_, err := store.ExpireDueGates(ctx, now)
			return err
		}},
		{name: "cancellation", timeout: 60_000, compete: func(store *Store, now time.Time) error {
			return store.Cancel(ctx, "decision-race", now)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, plan := gateStoreFixture(t, test.timeout, false)
			path := filepath.Join(t.TempDir(), "workflow.db")
			storeA, err := Open(ctx, path)
			require.NoError(t, err)
			defer func() { require.NoError(t, storeA.Close()) }()
			storeB, err := Open(ctx, path)
			require.NoError(t, err)
			defer func() { require.NoError(t, storeB.Close()) }()
			now := time.Now().UTC()
			require.NoError(t, storeA.CreateRun(ctx, "decision-race", plan, nil, now))
			advanced, err := storeA.AdvanceOneGate(ctx, now)
			require.NoError(t, err)
			require.True(t, advanced)
			boundary := now.Add(2 * time.Millisecond)
			if test.name == "cancellation" {
				boundary = now.Add(time.Second)
			}
			decision := gateDecisionRef("5")
			command := workflowv3.GateDecisionCommand{RunID: "decision-race", GateKey: "review", ExpectedVersion: 1, Decision: "approve", DecisionCode: "RACE_APPROVAL", ActorID: "reviewer", AuthorizedRole: "reviewer.primary", DecisionRef: &decision}
			start := make(chan struct{})
			results := make(chan error, 2)
			go func() { <-start; results <- storeA.DecideGate(ctx, command, boundary) }()
			go func() { <-start; results <- test.compete(storeB, boundary) }()
			close(start)
			<-results
			<-results
			operational, err := storeA.OperationalSnapshot(ctx, nil, nil, nil, boundary.Add(time.Second))
			require.NoError(t, err)
			require.Equal(t, int64(2), operational.Gates[0].Version)
			require.Contains(t, []string{"approved", "expired", "canceled"}, operational.Gates[0].Status)
			late := storeA.DecideGate(ctx, command, boundary.Add(2*time.Second))
			if operational.Gates[0].Status == "approved" {
				require.NoError(t, late)
			} else {
				require.Error(t, late)
			}
			var terminalEvents int
			require.NoError(t, storeA.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM v3_events
WHERE run_id = 'decision-race' AND event_type IN
  ('gate.approved','gate.expired','gate.canceled')`).Scan(&terminalEvents))
			require.Equal(t, 1, terminalEvents)
		})
	}
}

func TestGateApproveRejectRaceAcrossConnectionsHasOneDecision(t *testing.T) {
	ctx := context.Background()
	_, plan := gateStoreFixture(t, 60_000, false)
	path := filepath.Join(t.TempDir(), "workflow.db")
	storeA, err := Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, storeA.Close()) }()
	storeB, err := Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, storeB.Close()) }()
	now := time.Now().UTC()
	require.NoError(t, storeA.CreateRun(ctx, "race", plan, nil, now))
	advanced, err := storeA.AdvanceOneGate(ctx, now)
	require.NoError(t, err)
	require.True(t, advanced)
	decision := gateDecisionRef("4")
	approve := workflowv3.GateDecisionCommand{RunID: "race", GateKey: "review", ExpectedVersion: 1, Decision: "approve", DecisionCode: "RACE_APPROVE", ActorID: "reviewer-a", AuthorizedRole: "reviewer.primary", DecisionRef: &decision}
	reject := workflowv3.GateDecisionCommand{RunID: "race", GateKey: "review", ExpectedVersion: 1, Decision: "reject", DecisionCode: "RACE_REJECT", ActorID: "reviewer-b", AuthorizedRole: "reviewer.primary"}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wait sync.WaitGroup
	for index, command := range []workflowv3.GateDecisionCommand{approve, reject} {
		wait.Add(1)
		go func(store *Store, command workflowv3.GateDecisionCommand) {
			defer wait.Done()
			<-start
			results <- store.DecideGate(ctx, command, now.Add(time.Second))
		}([]*Store{storeA, storeB}[index], command)
	}
	close(start)
	wait.Wait()
	close(results)
	succeeded := 0
	for decisionErr := range results {
		if decisionErr == nil {
			succeeded++
		} else {
			require.True(t, errors.Is(decisionErr, ErrGateAlreadyDecided) || errors.Is(decisionErr, ErrGateVersionConflict))
		}
	}
	require.Equal(t, 1, succeeded)
	operational, err := storeA.OperationalSnapshot(ctx, nil, nil, nil, now.Add(2*time.Second))
	require.NoError(t, err)
	require.Equal(t, int64(2), operational.Gates[0].Version)
	require.Contains(t, []string{"approved", "rejected"}, operational.Gates[0].Status)
}
