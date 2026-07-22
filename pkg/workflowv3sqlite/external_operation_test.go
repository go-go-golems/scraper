package workflowv3sqlite

import (
	"context"
	"path/filepath"
	"strings"
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
	spec := workflowv3.ExternalOperationSpec{DescriptorDigest: descriptor.Digest, Reservation: []workflowv3.ExternalOperationCounter{{Name: "requests", Units: 1}}}
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

	var operations, completions, counters int
	require.NoError(t, store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM v3_external_operations`).Scan(&operations))
	require.NoError(t, store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM v3_external_operation_completions`).Scan(&completions))
	require.NoError(t, store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM v3_external_operation_counters`).Scan(&counters))
	require.Equal(t, 1, operations)
	require.Equal(t, 1, completions)
	require.Equal(t, 1, counters)
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
