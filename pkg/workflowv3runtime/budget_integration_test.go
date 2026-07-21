package workflowv3runtime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/testfixtures/workflowv3budget"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3sqlite"
	"github.com/stretchr/testify/require"
)

func budgetEngineFixture(t *testing.T) (*workflowv3.SealedRegistry, workflowv3.WorkflowPlan, *TaskModuleRegistry) {
	t.Helper()
	registry, err := workflowv3budget.Registry()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	authored, err := workflowmodule.Author(
		context.Background(), workflowv3budget.WorkflowSource(), catalog,
		workflowv3budget.DescriptorModule(),
	)
	require.NoError(t, err)
	modules, err := NewTaskModuleRegistry(FSInputModule())
	require.NoError(t, err)
	return registry, authored.Plan, modules
}

func TestBudgetedJavaScriptSettlesUsageRejectsOverageAndReopens(t *testing.T) {
	ctx := context.Background()
	registry, plan, modules := budgetEngineFixture(t)
	root := t.TempDir()
	databasePath := filepath.Join(root, "workflow.db")
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(root, "artifacts"), 1<<20)
	require.NoError(t, err)
	store, err := workflowv3sqlite.Open(ctx, databasePath)
	require.NoError(t, err)
	engine := &Engine{Store: store, Registry: registry, Artifacts: artifacts, Modules: modules}

	const privateCanary = "PRIVATE-BUDGET-REQUEST-CANARY"
	successInput, err := artifacts.Put(ctx, "budget-request/v1", "application/json", []byte(`{"requestId":"`+privateCanary+`","outputTokens":3}`))
	require.NoError(t, err)
	require.NoError(t, engine.Submit(ctx, "budget-success", plan, map[string]workflowv3.ArtifactRef{"request": successInput}))
	ran, err := engine.RunOne(ctx)
	require.True(t, ran)
	require.NoError(t, err)
	runID := workflowv3.RunID("budget-success")
	progress, err := store.BudgetSnapshot(ctx, &runID)
	require.NoError(t, err)
	require.Len(t, progress, 2)
	require.Equal(t, "output_tokens", progress[0].Dimension)
	require.Equal(t, int64(3), progress[0].Used)
	require.Zero(t, progress[0].Reserved)
	require.Equal(t, int64(1), progress[1].Used)
	successSnapshot, err := engine.Snapshot(ctx, "budget-success")
	require.NoError(t, err)
	successOutput, err := workflowv3.ReadArtifact(ctx, artifacts, successSnapshot.Outputs["response"])
	require.NoError(t, err)
	require.NotContains(t, string(successOutput), privateCanary)

	failureInput, err := artifacts.Put(ctx, "budget-request/v1", "application/json", []byte(`{"requestId":"failed","outputTokens":2,"fail":true}`))
	require.NoError(t, err)
	require.NoError(t, engine.Submit(ctx, "budget-failed", plan, map[string]workflowv3.ArtifactRef{"request": failureInput}))
	ran, err = engine.RunOne(ctx)
	require.True(t, ran)
	var attemptErr *AttemptExecutionError
	require.ErrorAs(t, err, &attemptErr)
	failureSnapshot, err := engine.Snapshot(ctx, "budget-failed")
	require.NoError(t, err)
	require.Equal(t, "BUDGET_FIXTURE_PROVIDER_REJECTED", failureSnapshot.Attempts[0].Failure.Code)
	failureRunID := workflowv3.RunID("budget-failed")
	failureProgress, err := store.BudgetSnapshot(ctx, &failureRunID)
	require.NoError(t, err)
	require.Equal(t, int64(2), failureProgress[0].Used)
	require.Equal(t, int64(1), failureProgress[1].Used)

	overInput, err := artifacts.Put(ctx, "budget-request/v1", "application/json", []byte(`{"requestId":"over","outputTokens":6}`))
	require.NoError(t, err)
	require.NoError(t, engine.Submit(ctx, "budget-over", plan, map[string]workflowv3.ArtifactRef{"request": overInput}))
	ran, err = engine.RunOne(ctx)
	require.True(t, ran)
	require.ErrorAs(t, err, &attemptErr)
	overSnapshot, err := engine.Snapshot(ctx, "budget-over")
	require.NoError(t, err)
	require.Equal(t, "failed", overSnapshot.Status)
	require.Equal(t, "BUDGET_USAGE_EXCEEDS_RESERVATION", overSnapshot.Attempts[0].Failure.Code)
	overRunID := workflowv3.RunID("budget-over")
	overProgress, err := store.BudgetSnapshot(ctx, &overRunID)
	require.NoError(t, err)
	require.Equal(t, int64(5), overProgress[0].Used)
	require.Equal(t, int64(1), overProgress[1].Used)

	missing := workflowv3.ArtifactRef{
		Schema: "budget-request/v1", Digest: "sha256:" + strings.Repeat("f", 64),
		MediaType: "application/json", Size: 2, Locator: "cas://missing",
	}
	require.NoError(t, engine.Submit(ctx, "budget-no-contact", plan, map[string]workflowv3.ArtifactRef{"request": missing}))
	ran, err = engine.RunOne(ctx)
	require.True(t, ran)
	require.ErrorAs(t, err, &attemptErr)
	missingRunID := workflowv3.RunID("budget-no-contact")
	missingProgress, err := store.BudgetSnapshot(ctx, &missingRunID)
	require.NoError(t, err)
	for _, item := range missingProgress {
		require.Zero(t, item.Used)
		require.Zero(t, item.Reserved)
	}

	require.NoError(t, store.Close())
	store, err = workflowv3sqlite.Open(ctx, databasePath)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	progress, err = store.BudgetSnapshot(ctx, &runID)
	require.NoError(t, err)
	require.Equal(t, int64(3), progress[0].Used)
	operational, err := store.OperationalSnapshot(
		ctx, nil, registry, map[string]int{workflowv3budget.ResourceClass: 1}, time.Now().UTC(),
	)
	require.NoError(t, err)
	projectionJSON, err := json.Marshal(operational)
	require.NoError(t, err)
	require.NotContains(t, string(projectionJSON), privateCanary)
	body, err := os.ReadFile(databasePath)
	require.NoError(t, err)
	require.False(t, strings.Contains(string(body), privateCanary))
	wal, walErr := os.ReadFile(databasePath + "-wal")
	if walErr == nil {
		require.False(t, strings.Contains(string(wal), privateCanary))
	} else {
		require.True(t, errors.Is(walErr, os.ErrNotExist))
	}
}

func TestBudgetCancellationChargesRunningReservationConservatively(t *testing.T) {
	ctx := context.Background()
	registry, plan, _ := budgetEngineFixture(t)
	store, err := workflowv3sqlite.Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	now := time.Now().UTC()
	input := workflowv3.ArtifactRef{Schema: "budget-request/v1", Digest: "sha256:" + strings.Repeat("e", 64), MediaType: "application/json", Size: 2, Locator: "cas://request"}
	require.NoError(t, store.CreateRun(ctx, "cancel", plan, map[string]workflowv3.ArtifactRef{"request": input}, now))
	lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NoError(t, store.Cancel(ctx, "cancel", now.Add(time.Second)))
	runID := workflowv3.RunID("cancel")
	progress, err := store.BudgetSnapshot(ctx, &runID)
	require.NoError(t, err)
	require.Equal(t, int64(5), progress[0].Used)
	require.Equal(t, int64(1), progress[1].Used)
	for _, item := range progress {
		require.Zero(t, item.Reserved)
	}
}
