package workflowv3runtime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/testfixtures/workflowv3gate"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGateWaitsAcrossDispatcherRestartWhileOtherRunCompletes(t *testing.T) {
	ctx := context.Background()
	registry, err := workflowv3gate.Registry()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	gatePlan, err := workflowmodule.Author(ctx, workflowv3gate.WorkflowSource(), catalog, workflowv3gate.DescriptorModule())
	require.NoError(t, err)
	independentPlan, err := workflowmodule.Author(ctx, workflowv3gate.IndependentSource(), catalog, workflowv3gate.DescriptorModule())
	require.NoError(t, err)
	modules, err := NewTaskModuleRegistry(FSInputModule())
	require.NoError(t, err)
	root := t.TempDir()
	databasePath := filepath.Join(root, "workflow.db")
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(root, "artifacts"), 1<<20)
	require.NoError(t, err)
	const privateCanary = "PRIVATE-GATE-SOURCE-CANARY"
	const decisionCanary = "PRIVATE-GATE-DECISION-CANARY"
	source, err := artifacts.Put(ctx, "gate-source/v1", "application/json", []byte(`{"records":["`+privateCanary+`"]}`))
	require.NoError(t, err)
	store, err := workflowv3sqlite.Open(ctx, databasePath)
	require.NoError(t, err)
	engine := &Engine{Store: store, Registry: registry, Artifacts: artifacts, Modules: modules, LeaseDuration: 10 * time.Second}
	require.NoError(t, engine.Submit(ctx, "waiting", gatePlan.Plan, map[string]workflowv3.ArtifactRef{"source": source}))
	require.NoError(t, engine.Submit(ctx, "independent", independentPlan.Plan, map[string]workflowv3.ArtifactRef{"source": source}))
	dispatcher := &Dispatcher{Engine: engine, Capacities: map[string]int{workflowv3gate.ResourceClass: 1}, PollInterval: 2 * time.Millisecond}
	dispatchCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- dispatcher.Run(dispatchCtx) }()
	require.True(t, assert.Eventually(t, func() bool {
		waiting, waitingErr := engine.Snapshot(ctx, "waiting")
		independent, independentErr := engine.Snapshot(ctx, "independent")
		operational, operationalErr := dispatcher.OperationalSnapshot(ctx, nil)
		return waitingErr == nil && independentErr == nil && operationalErr == nil &&
			waiting.Status == "running" && len(waiting.Attempts) == 1 &&
			independent.Status == "succeeded" &&
			operational.GateStatuses["waiting"] == 1
	}, 10*time.Second, 10*time.Millisecond))
	operational, err := dispatcher.OperationalSnapshot(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, operational.Queue.ActiveByResource)
	require.Equal(t, 1, operational.Queue.BlockedByReason["gate-dependency"])
	require.Empty(t, operational.Budgets)
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	require.NoError(t, store.Close())

	store, err = workflowv3sqlite.Open(ctx, databasePath)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	engine = &Engine{Store: store, Registry: registry, Artifacts: artifacts, Modules: modules, LeaseDuration: 10 * time.Second}
	decision, err := artifacts.Put(ctx, "gate-decision/v1", "application/json", []byte(`{"approved":true,"private":"`+decisionCanary+`"}`))
	require.NoError(t, err)
	require.NoError(t, store.DecideGate(ctx, workflowv3.GateDecisionCommand{
		RunID: "waiting", GateKey: "review", ExpectedVersion: 1,
		Decision: "approve", DecisionCode: "REVIEW_APPROVED",
		ActorID: "reviewer@example", AuthorizedRole: "reviewer.primary",
		DecisionRef: &decision,
	}, time.Now().UTC()))
	dispatcher = &Dispatcher{Engine: engine, Capacities: map[string]int{workflowv3gate.ResourceClass: 1}, PollInterval: 2 * time.Millisecond}
	dispatchCtx, cancel = context.WithCancel(ctx)
	done = make(chan error, 1)
	go func() { done <- dispatcher.Run(dispatchCtx) }()
	require.True(t, assert.Eventually(t, func() bool {
		snapshot, snapshotErr := engine.Snapshot(ctx, "waiting")
		return snapshotErr == nil && snapshot.Status == "succeeded"
	}, 10*time.Second, 10*time.Millisecond))
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	snapshot, err := engine.Snapshot(ctx, "waiting")
	require.NoError(t, err)
	require.Len(t, snapshot.Attempts, 2)
	output, err := workflowv3.ReadArtifact(ctx, artifacts, snapshot.Outputs["published"])
	require.NoError(t, err)
	require.JSONEq(t, `{"approved":true}`, string(output))
	require.NotContains(t, string(output), privateCanary)
	require.NotContains(t, string(output), decisionCanary)
	operational, err = dispatcher.OperationalSnapshot(ctx, nil)
	require.NoError(t, err)
	projection, err := json.Marshal(operational)
	require.NoError(t, err)
	require.NotContains(t, string(projection), privateCanary)
	require.NotContains(t, string(projection), decisionCanary)
	events, err := store.EventsAfter(ctx, 0, nil, 1000)
	require.NoError(t, err)
	eventJSON, err := json.Marshal(events)
	require.NoError(t, err)
	require.NotContains(t, string(eventJSON), privateCanary)
	require.NotContains(t, string(eventJSON), decisionCanary)
	require.NoError(t, store.Checkpoint(ctx))
	databaseBytes, err := os.ReadFile(databasePath)
	require.NoError(t, err)
	require.NotContains(t, string(databaseBytes), privateCanary)
	require.NotContains(t, string(databaseBytes), decisionCanary)
	wal, walErr := os.ReadFile(databasePath + "-wal")
	if walErr == nil {
		require.NotContains(t, string(wal), privateCanary)
		require.NotContains(t, string(wal), decisionCanary)
	} else {
		require.True(t, errors.Is(walErr, os.ErrNotExist))
	}
}

func TestTaskRuntimeCannotImportGateOperatorAuthority(t *testing.T) {
	ctx := context.Background()
	base, err := workflowv3gate.Bundle()
	require.NoError(t, err)
	bundle, err := workflowv3.NewBundle(base.Manifest(), map[string][]byte{"tasks.cjs": []byte(`
const operator = require("workflow/operator");
exports.prepare = () => operator.approve();
exports.publish = () => operator.approve();`)})
	require.NoError(t, err)
	builder := workflowv3.NewRegistryBuilder()
	require.NoError(t, builder.AdvertiseModules("fs:input"))
	require.NoError(t, builder.AddBundle(bundle))
	registry, err := builder.Seal()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	spec, ok := catalog.Lookup(workflowv3.TaskKey{Kind: "fixture.gate.prepare", Version: "v1"})
	require.True(t, ok)
	registered, err := registry.Resolve(spec.Identity)
	require.NoError(t, err)
	artifacts, err := workflowv3.NewFileArtifactStore(t.TempDir(), 1024)
	require.NoError(t, err)
	input, err := artifacts.Put(ctx, "gate-source/v1", "application/json", []byte(`{"records":[]}`))
	require.NoError(t, err)
	modules, err := NewTaskModuleRegistry(FSInputModule())
	require.NoError(t, err)
	require.NotContains(t, registry.ModuleAliases(), "workflow/operator")
	_, err = RunTask(ctx, TaskRequest{RunID: "operator-denied", NodeKey: "prepare", Attempt: 1, Task: registered, Inputs: map[string]workflowv3.ArtifactRef{"source": input}, Artifacts: artifacts, Modules: modules})
	require.ErrorContains(t, err, "Invalid module")
}
