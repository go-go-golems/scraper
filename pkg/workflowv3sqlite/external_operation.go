package workflowv3sqlite

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/google/uuid"
)

var (
	ErrExternalOperationCompletionConflict    = errors.New("workflow v3 external operation completion conflict")
	ErrExternalOperationDescriptorUnavailable = errors.New("workflow v3 external operation descriptor unavailable")
)

type externalOperationRecorder struct {
	store       *Store
	lease       workflowv3.Lease
	descriptors map[string]workflowv3.ExternalOperationDescriptor
}

// ExternalOperationRecorder returns the host-only recorder permitted for one
// leased attempt. Each descriptor must have been selected by an exact task
// module alias; JavaScript never receives this recorder directly.
func (s *Store) ExternalOperationRecorder(lease workflowv3.Lease, descriptors []workflowv3.ExternalOperationDescriptor) (workflowv3.ExternalOperationRecorder, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("workflow v3 store is required")
	}
	if lease.RunID == "" || lease.NodeKey == "" || lease.Attempt < 1 || lease.Token == "" {
		return nil, fmt.Errorf("valid workflow lease is required")
	}
	allowed := make(map[string]workflowv3.ExternalOperationDescriptor, len(descriptors))
	for _, descriptor := range descriptors {
		if err := workflowv3.ValidateExternalOperationDescriptor(descriptor); err != nil {
			return nil, err
		}
		if _, duplicate := allowed[descriptor.Digest]; duplicate {
			continue
		}
		allowed[descriptor.Digest] = descriptor
	}
	return &externalOperationRecorder{store: s, lease: lease, descriptors: allowed}, nil
}

func (r *externalOperationRecorder) BeginExternalOperation(ctx context.Context, spec workflowv3.ExternalOperationSpec) (workflowv3.ExternalOperationTicket, error) {
	if r == nil {
		return workflowv3.ExternalOperationTicket{}, fmt.Errorf("external operation recorder is required")
	}
	descriptor, ok := r.descriptors[spec.DescriptorDigest]
	if !ok {
		return workflowv3.ExternalOperationTicket{}, ErrExternalOperationDescriptorUnavailable
	}
	return r.store.BeginExternalOperation(ctx, r.lease, descriptor, spec, time.Now().UTC())
}

func (r *externalOperationRecorder) FinishExternalOperation(ctx context.Context, ticket workflowv3.ExternalOperationTicket, completion workflowv3.ExternalOperationCompletion) error {
	if r == nil {
		return fmt.Errorf("external operation recorder is required")
	}
	var descriptorDigest string
	if err := r.store.db.QueryRowContext(ctx, `SELECT descriptor_digest FROM v3_external_operations WHERE operation_id=?`, ticket.OperationID).Scan(&descriptorDigest); err != nil {
		return err
	}
	descriptor, ok := r.descriptors[descriptorDigest]
	if !ok {
		return ErrExternalOperationDescriptorUnavailable
	}
	return r.store.FinishExternalOperation(ctx, ticket, descriptor, completion, time.Now().UTC())
}

// checkSQLiteDurability verifies the authority-bearing SQLite settings that the
// DSN requests for every connection. External operation admission must not
// return until an SQLite FULL synchronous commit has completed.
func checkSQLiteDurability(ctx context.Context, db *sql.DB) error {
	var journalMode string
	if err := db.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		return err
	}
	if journalMode != "wal" {
		return fmt.Errorf("journal mode is %q, want wal", journalMode)
	}
	var synchronous, foreignKeys int
	if err := db.QueryRowContext(ctx, `PRAGMA synchronous`).Scan(&synchronous); err != nil {
		return err
	}
	if synchronous != 2 {
		return fmt.Errorf("synchronous mode is %d, want FULL (2)", synchronous)
	}
	if err := db.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		return err
	}
	if foreignKeys != 1 {
		return fmt.Errorf("foreign keys are disabled")
	}
	return nil
}

func checkExternalOperationInvariants(ctx context.Context, db *sql.DB) error {
	checks := []struct{ name, query string }{
		{"allocations without admissions", `SELECT COUNT(*) FROM v3_external_operation_allocations allocation WHERE NOT EXISTS (SELECT 1 FROM v3_external_operations operation WHERE operation.operation_id = allocation.operation_id)`},
		{"measures without admissions", `SELECT COUNT(*) FROM v3_external_operation_measures measure WHERE NOT EXISTS (SELECT 1 FROM v3_external_operations operation WHERE operation.operation_id = measure.operation_id)`},
		{"completions without admissions", `SELECT COUNT(*) FROM v3_external_operation_completions completion WHERE NOT EXISTS (SELECT 1 FROM v3_external_operations operation WHERE operation.operation_id = completion.operation_id)`},
		{"counters without completions", `SELECT COUNT(*) FROM v3_external_operation_counters counter WHERE NOT EXISTS (SELECT 1 FROM v3_external_operation_completions completion WHERE completion.operation_id = counter.operation_id)`},
		{"admissions without attempts", `SELECT COUNT(*) FROM v3_external_operations operation WHERE NOT EXISTS (SELECT 1 FROM v3_attempts attempt WHERE attempt.run_id = operation.run_id AND attempt.node_key = operation.node_key AND attempt.attempt_no = operation.attempt_no)`},
	}
	for _, check := range checks {
		var count int
		if err := db.QueryRowContext(ctx, check.query).Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return fmt.Errorf("EXTERNAL_OPERATION_INVARIANT: %d %s", count, check.name)
		}
	}
	return nil
}

// BeginExternalOperation durably admits one host-owned external effect before
// its provider/tool call begins. The active Workflow lease is the admission
// authority; callers must not invoke the effect until this method returns.
func (s *Store) BeginExternalOperation(ctx context.Context, lease workflowv3.Lease, descriptor workflowv3.ExternalOperationDescriptor, spec workflowv3.ExternalOperationSpec, now time.Time) (workflowv3.ExternalOperationTicket, error) {
	if err := workflowv3.ValidateExternalOperationSpec(descriptor, spec); err != nil {
		return workflowv3.ExternalOperationTicket{}, err
	}
	completionKey, err := newCompletionKey()
	if err != nil {
		return workflowv3.ExternalOperationTicket{}, err
	}
	keyDigest := digestCompletionKey(completionKey)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return workflowv3.ExternalOperationTicket{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := checkFence(ctx, tx, lease); err != nil {
		return workflowv3.ExternalOperationTicket{}, err
	}
	var admitted int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM v3_external_operations WHERE run_id=? AND node_key=? AND attempt_no=? AND descriptor_digest=?`, lease.RunID, lease.NodeKey, lease.Attempt, descriptor.Digest).Scan(&admitted); err != nil {
		return workflowv3.ExternalOperationTicket{}, err
	}
	if admitted >= descriptor.MaxPerAttempt {
		return workflowv3.ExternalOperationTicket{}, fmt.Errorf("external operation %s@%s exceeds max per attempt", descriptor.Kind.Name, descriptor.Kind.Version)
	}
	var ordinal int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(ordinal), 0) + 1 FROM v3_external_operations WHERE run_id=? AND node_key=? AND attempt_no=?`, lease.RunID, lease.NodeKey, lease.Attempt).Scan(&ordinal); err != nil {
		return workflowv3.ExternalOperationTicket{}, err
	}
	operationID := uuid.NewString()
	var correlation any
	if spec.CorrelationDigest != "" {
		correlation = spec.CorrelationDigest
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO v3_external_operations(operation_id,run_id,node_key,attempt_no,ordinal,kind,kind_version,descriptor_digest,authority_digest,correlation_digest,completion_key_digest,admitted_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, operationID, lease.RunID, lease.NodeKey, lease.Attempt, ordinal, descriptor.Kind.Name, descriptor.Kind.Version, descriptor.Digest, descriptor.AuthorityDigest, correlation, keyDigest, formatTime(now)); err != nil {
		return workflowv3.ExternalOperationTicket{}, err
	}
	for _, allocation := range spec.Reservation {
		if _, err := tx.ExecContext(ctx, `INSERT INTO v3_external_operation_allocations(operation_id,dimension,units) VALUES(?,?,?)`, operationID, allocation.Name, allocation.Units); err != nil {
			return workflowv3.ExternalOperationTicket{}, err
		}
	}
	for _, measure := range spec.Measures {
		if _, err := tx.ExecContext(ctx, `INSERT INTO v3_external_operation_measures(operation_id,name,units) VALUES(?,?,?)`, operationID, measure.Name, measure.Units); err != nil {
			return workflowv3.ExternalOperationTicket{}, err
		}
	}
	if err := insertEvent(ctx, tx, lease.RunID, lease.NodeKey, "external_operation.admitted", map[string]any{"attempt": lease.Attempt, "operationId": operationID, "ordinal": ordinal, "kind": descriptor.Kind.Name, "descriptorDigest": descriptor.Digest}, now); err != nil {
		return workflowv3.ExternalOperationTicket{}, err
	}
	if err := tx.Commit(); err != nil {
		return workflowv3.ExternalOperationTicket{}, err
	}
	return workflowv3.ExternalOperationTicket{OperationID: operationID, CompletionKey: completionKey}, nil
}

// FinishExternalOperation appends one immutable safe completion. It validates
// the ticket rather than the live lease, so an operation admitted before
// cancellation can preserve its outcome without gaining task-output authority.
func (s *Store) FinishExternalOperation(ctx context.Context, ticket workflowv3.ExternalOperationTicket, descriptor workflowv3.ExternalOperationDescriptor, completion workflowv3.ExternalOperationCompletion, now time.Time) error {
	if ticket.OperationID == "" || ticket.CompletionKey == "" {
		return fmt.Errorf("external operation ticket is required")
	}
	if err := workflowv3.ValidateExternalOperationCompletion(descriptor, completion); err != nil {
		return err
	}
	completionDigest, err := workflowv3.Digest(completion)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var descriptorDigest, completionKeyDigest string
	if err := tx.QueryRowContext(ctx, `SELECT descriptor_digest,completion_key_digest FROM v3_external_operations WHERE operation_id=?`, ticket.OperationID).Scan(&descriptorDigest, &completionKeyDigest); err != nil {
		return err
	}
	if descriptorDigest != descriptor.Digest {
		return fmt.Errorf("external operation descriptor digest mismatch")
	}
	if subtle.ConstantTimeCompare([]byte(completionKeyDigest), []byte(digestCompletionKey(ticket.CompletionKey))) != 1 {
		return fmt.Errorf("external operation completion ticket is invalid")
	}
	var existingDigest string
	err = tx.QueryRowContext(ctx, `SELECT completion_digest FROM v3_external_operation_completions WHERE operation_id=?`, ticket.OperationID).Scan(&existingDigest)
	switch {
	case err == nil:
		if existingDigest == completionDigest {
			return tx.Commit()
		}
		return ErrExternalOperationCompletionConflict
	case !errors.Is(err, sql.ErrNoRows):
		return err
	}
	var failureClass, failureCode any
	if completion.Failure != nil {
		failureClass, failureCode = completion.Failure.Class, completion.Failure.Code
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO v3_external_operation_completions(operation_id,provider_started_at,elapsed_micros,outcome,failure_class,failure_code,accounting_mode,completed_at,completion_digest)
VALUES(?,?,?,?,?,?,?,?,?)`, ticket.OperationID, formatTime(completion.ProviderStartedAt), completion.ElapsedMicros, completion.Outcome, failureClass, failureCode, completion.AccountingMode, formatTime(now), completionDigest); err != nil {
		return err
	}
	for _, counter := range completion.Counters {
		if _, err := tx.ExecContext(ctx, `INSERT INTO v3_external_operation_counters(operation_id,name,units) VALUES(?,?,?)`, ticket.OperationID, counter.Name, counter.Units); err != nil {
			return err
		}
	}
	var runID workflowv3.RunID
	var nodeKey workflowv3.NodeKey
	var attempt int
	if err := tx.QueryRowContext(ctx, `SELECT run_id,node_key,attempt_no FROM v3_external_operations WHERE operation_id=?`, ticket.OperationID).Scan(&runID, &nodeKey, &attempt); err != nil {
		return err
	}
	if err := insertEvent(ctx, tx, runID, nodeKey, "external_operation.completed", map[string]any{"attempt": attempt, "operationId": ticket.OperationID, "outcome": completion.Outcome, "completionDigest": completionDigest}, now); err != nil {
		return err
	}
	return tx.Commit()
}

func newCompletionKey() (string, error) {
	body := make([]byte, 32)
	if _, err := rand.Read(body); err != nil {
		return "", err
	}
	return hex.EncodeToString(body), nil
}

func digestCompletionKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}
