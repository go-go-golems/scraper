package workflowv3sqlite

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/stretchr/testify/require"
)

func operationDescriptor(t *testing.T) workflowv3.ExternalOperationDescriptor {
	t.Helper()
	descriptor, err := workflowv3.NewExternalOperationDescriptor(workflowv3.ExternalOperationDescriptor{
		Kind:            workflowv3.ExternalOperationKind{Name: "provider.generate", Version: "v1"},
		AuthorityDigest: "sha256:" + strings.Repeat("e", 64), MaxPerAttempt: 1,
		Counters: []workflowv3.ExternalOperationCounterDescriptor{
			{Name: "requests", Unit: "requests", Roles: []workflowv3.ExternalOperationCounterRole{workflowv3.ExternalOperationCounterReservation, workflowv3.ExternalOperationCounterUsage}},
		},
	})
	require.NoError(t, err)
	return descriptor
}

func TestExternalOperationAdmissionAndLateTicketCompletion(t *testing.T) {
	ctx := context.Background()
	registry, plan := storeFixture(t, "first")
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	require.NoError(t, store.CreateRun(ctx, "operation", plan, map[string]workflowv3.ArtifactRef{"source": artifactRef("source/v1", "operation")}, now))
	lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	descriptor := operationDescriptor(t)
	spec := workflowv3.ExternalOperationSpec{DescriptorDigest: descriptor.Digest}
	ticket, err := store.BeginExternalOperation(ctx, *lease, descriptor, spec, now.Add(time.Millisecond))
	require.NoError(t, err)
	require.NotEmpty(t, ticket.OperationID)

	require.NoError(t, store.Cancel(ctx, "operation", now.Add(2*time.Millisecond)))
	completion := workflowv3.ExternalOperationCompletion{ProviderStartedAt: now.UTC(), ElapsedMicros: 123, Outcome: workflowv3.ExternalOperationOutcomeSucceeded, AccountingMode: workflowv3.ExternalOperationAccountingActual, Counters: []workflowv3.ExternalOperationCounter{{Name: "requests", Units: 1}}}
	require.NoError(t, store.FinishExternalOperation(ctx, ticket, descriptor, completion, now.Add(3*time.Millisecond)))
	require.NoError(t, store.FinishExternalOperation(ctx, ticket, descriptor, completion, now.Add(4*time.Millisecond)))
	conflict := completion
	conflict.ElapsedMicros++
	require.ErrorIs(t, store.FinishExternalOperation(ctx, ticket, descriptor, conflict, now.Add(5*time.Millisecond)), ErrExternalOperationCompletionConflict)
	_, err = store.BeginExternalOperation(ctx, *lease, descriptor, spec, now.Add(6*time.Millisecond))
	require.ErrorIs(t, err, ErrStaleCompletion)

	var operationRows, completions, counters int
	require.NoError(t, store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM v3_external_operations`).Scan(&operationRows))
	require.NoError(t, store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM v3_external_operation_completions`).Scan(&completions))
	require.NoError(t, store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM v3_external_operation_counters`).Scan(&counters))
	require.Equal(t, 1, operationRows)
	require.Equal(t, 1, completions)
	require.Equal(t, 1, counters)

	operations, err := store.ExternalOperations(ctx, "operation")
	require.NoError(t, err)
	require.Len(t, operations, 1)
	require.NotNil(t, operations[0].Completion)
	require.Equal(t, int64(123), operations[0].Completion.ElapsedMicros)
	require.Equal(t, []workflowv3.ExternalOperationCounter{{Name: "requests", Units: 1}}, operations[0].Completion.Counters)
	progress, err := store.ExternalOperationProgress(ctx, "operation")
	require.NoError(t, err)
	require.Equal(t, 1, progress.Admitted)
	require.Equal(t, 1, progress.Completed)
	require.Zero(t, progress.Incomplete)
	require.Equal(t, map[string]int{workflowv3.ExternalOperationOutcomeSucceeded: 1}, progress.Outcomes)

	exportDir := t.TempDir()
	jsonl := filepath.Join(exportDir, "external-operations.jsonl")
	manifestPath := filepath.Join(exportDir, "external-operations-manifest.json")
	manifest, err := store.ExportExternalOperations(ctx, "operation", jsonl, manifestPath)
	require.NoError(t, err)
	require.Equal(t, workflowv3.ExternalOperationExportSchema, manifest.SchemaVersion)
	require.Equal(t, 1, manifest.RecordCount)
	require.Equal(t, 1, manifest.CompletedCount)
	firstJSONL, err := os.ReadFile(jsonl)
	require.NoError(t, err)
	require.NotContains(t, string(firstJSONL), ticket.CompletionKey)
	firstManifest, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	_, err = store.ExportExternalOperations(ctx, "operation", jsonl, manifestPath)
	require.NoError(t, err)
	secondJSONL, err := os.ReadFile(jsonl)
	require.NoError(t, err)
	secondManifest, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	require.Equal(t, firstJSONL, secondJSONL)
	require.Equal(t, firstManifest, secondManifest)
}

func TestExternalOperationAllocationSettlesAttemptBudget(t *testing.T) {
	ctx := context.Background()
	registry, plan := budgetStoreFixture(t, 1, 1)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	now := time.Now().UTC()
	createBudgetRun(t, store, plan, "operation-budget", now)
	lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	descriptor := operationDescriptor(t)
	spec := workflowv3.ExternalOperationSpec{DescriptorDigest: descriptor.Digest, Reservation: []workflowv3.ExternalOperationCounter{{Name: "requests", Units: 1}}}
	ticket, err := store.BeginExternalOperation(ctx, *lease, descriptor, spec, now)
	require.NoError(t, err)
	_, err = store.BeginExternalOperation(ctx, *lease, descriptor, spec, now)
	require.Error(t, err)
	completion := workflowv3.ExternalOperationCompletion{ProviderStartedAt: now.UTC(), Outcome: workflowv3.ExternalOperationOutcomeSucceeded, AccountingMode: workflowv3.ExternalOperationAccountingActual, Counters: []workflowv3.ExternalOperationCounter{{Name: "requests", Units: 1}}}
	require.NoError(t, store.FinishExternalOperation(ctx, ticket, descriptor, completion, now))
	require.NoError(t, store.CompleteWithUsage(ctx, *lease, map[string]workflowv3.ArtifactRef{"output": artifactRef("output/v1", "operation-budget")}, nil, now))
	budget, err := store.BudgetSnapshot(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), budget[0].Used)
	require.Zero(t, budget[0].Reserved)
}

func TestExternalOperationCompletionSurvivesStoreReopen(t *testing.T) {
	ctx := context.Background()
	registry, plan := storeFixture(t, "first")
	path := filepath.Join(t.TempDir(), "workflow.db")
	store, err := Open(ctx, path)
	require.NoError(t, err)
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	require.NoError(t, store.CreateRun(ctx, "operation-reopen", plan, map[string]workflowv3.ArtifactRef{"source": artifactRef("source/v1", "operation-reopen")}, now))
	lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	descriptor := operationDescriptor(t)
	ticket, err := store.BeginExternalOperation(ctx, *lease, descriptor, workflowv3.ExternalOperationSpec{DescriptorDigest: descriptor.Digest}, now.Add(time.Millisecond))
	require.NoError(t, err)
	require.NoError(t, store.Close())

	store, err = Open(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	completion := workflowv3.ExternalOperationCompletion{ProviderStartedAt: now, ElapsedMicros: 11, Outcome: workflowv3.ExternalOperationOutcomeSucceeded, AccountingMode: workflowv3.ExternalOperationAccountingActual, Counters: []workflowv3.ExternalOperationCounter{{Name: "requests", Units: 1}}}
	require.NoError(t, store.FinishExternalOperation(ctx, ticket, descriptor, completion, now.Add(2*time.Millisecond)))
	require.NoError(t, store.FinishExternalOperation(ctx, ticket, descriptor, completion, now.Add(3*time.Millisecond)))
	progress, err := store.ExternalOperationProgress(ctx, "operation-reopen")
	require.NoError(t, err)
	require.Equal(t, 1, progress.Admitted)
	require.Equal(t, 1, progress.Completed)
	require.Zero(t, progress.Incomplete)
}

func TestExternalOperationConcurrentAdmissionAndCompletion(t *testing.T) {
	ctx := context.Background()
	registry, plan := storeFixture(t, "first")
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	require.NoError(t, store.CreateRun(ctx, "operation-concurrent", plan, map[string]workflowv3.ArtifactRef{"source": artifactRef("source/v1", "operation-concurrent")}, now))
	lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	descriptor, err := workflowv3.NewExternalOperationDescriptor(workflowv3.ExternalOperationDescriptor{Kind: workflowv3.ExternalOperationKind{Name: "provider.generate", Version: "v1"}, AuthorityDigest: "sha256:" + strings.Repeat("f", 64), MaxPerAttempt: 16, Counters: []workflowv3.ExternalOperationCounterDescriptor{{Name: "requests", Unit: "requests", Roles: []workflowv3.ExternalOperationCounterRole{workflowv3.ExternalOperationCounterReservation, workflowv3.ExternalOperationCounterUsage}}}})
	require.NoError(t, err)
	tickets := make([]workflowv3.ExternalOperationTicket, 16)
	errors := make(chan error, len(tickets))
	var admissions sync.WaitGroup
	for index := range tickets {
		admissions.Add(1)
		go func(index int) {
			defer admissions.Done()
			ticket, beginErr := store.BeginExternalOperation(ctx, *lease, descriptor, workflowv3.ExternalOperationSpec{DescriptorDigest: descriptor.Digest}, now.Add(time.Duration(index)*time.Microsecond))
			if beginErr == nil {
				tickets[index] = ticket
			}
			errors <- beginErr
		}(index)
	}
	admissions.Wait()
	close(errors)
	for beginErr := range errors {
		require.NoError(t, beginErr)
	}
	var completions sync.WaitGroup
	completionErrors := make(chan error, len(tickets))
	for index, ticket := range tickets {
		completions.Add(1)
		go func(index int, ticket workflowv3.ExternalOperationTicket) {
			defer completions.Done()
			completionErrors <- store.FinishExternalOperation(ctx, ticket, descriptor, workflowv3.ExternalOperationCompletion{ProviderStartedAt: now, ElapsedMicros: int64(index), Outcome: workflowv3.ExternalOperationOutcomeSucceeded, AccountingMode: workflowv3.ExternalOperationAccountingActual, Counters: []workflowv3.ExternalOperationCounter{{Name: "requests", Units: 1}}}, now.Add(time.Second))
		}(index, ticket)
	}
	completions.Wait()
	close(completionErrors)
	for completionErr := range completionErrors {
		require.NoError(t, completionErr)
	}
	operations, err := store.ExternalOperations(ctx, "operation-concurrent")
	require.NoError(t, err)
	require.Len(t, operations, len(tickets))
	progress, err := store.ExternalOperationProgress(ctx, "operation-concurrent")
	require.NoError(t, err)
	require.Equal(t, len(tickets), progress.Admitted)
	require.Equal(t, len(tickets), progress.Completed)
	require.Zero(t, progress.Incomplete)
}

func TestExternalOperationRejectsWrongCompletionTicket(t *testing.T) {
	ctx := context.Background()
	registry, plan := storeFixture(t, "first")
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	now := time.Now().UTC()
	require.NoError(t, store.CreateRun(ctx, "operation-ticket", plan, map[string]workflowv3.ArtifactRef{"source": artifactRef("source/v1", "ticket")}, now))
	lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	descriptor := operationDescriptor(t)
	ticket, err := store.BeginExternalOperation(ctx, *lease, descriptor, workflowv3.ExternalOperationSpec{DescriptorDigest: descriptor.Digest}, now)
	require.NoError(t, err)
	ticket.CompletionKey = "wrong"
	completion := workflowv3.ExternalOperationCompletion{ProviderStartedAt: now.UTC(), Outcome: workflowv3.ExternalOperationOutcomeUnknown, AccountingMode: workflowv3.ExternalOperationAccountingConservative}
	require.Error(t, store.FinishExternalOperation(ctx, ticket, descriptor, completion, now))
}
