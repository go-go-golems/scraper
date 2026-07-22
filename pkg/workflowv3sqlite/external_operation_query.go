package workflowv3sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

// ExternalOperations reconstructs compact, ordered operation evidence for one
// run. It deliberately does not expose completion capabilities or event bodies.
func (s *Store) ExternalOperations(ctx context.Context, runID workflowv3.RunID) ([]workflowv3.ExternalOperation, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	operations, err := externalOperationsTx(ctx, tx, runID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return operations, nil
}

func externalOperationsTx(ctx context.Context, tx *sql.Tx, runID workflowv3.RunID) ([]workflowv3.ExternalOperation, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT o.operation_id,o.node_key,o.attempt_no,o.ordinal,o.kind,o.kind_version,
  o.descriptor_digest,o.authority_digest,COALESCE(o.correlation_digest,''),o.admitted_at,
  c.provider_started_at,c.elapsed_micros,c.outcome,c.failure_class,c.failure_code,
  c.accounting_mode,c.completed_at
FROM v3_external_operations o
LEFT JOIN v3_external_operation_completions c ON c.operation_id=o.operation_id
WHERE o.run_id=?
ORDER BY o.node_key,o.attempt_no,o.ordinal,o.operation_id`, runID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var operations []workflowv3.ExternalOperation
	for rows.Next() {
		var operation workflowv3.ExternalOperation
		operation.RunID = runID
		var admitted string
		var providerStarted, outcome, failureClass, failureCode, accountingMode, completed sql.NullString
		var elapsed sql.NullInt64
		if err := rows.Scan(&operation.OperationID, &operation.NodeKey, &operation.Attempt, &operation.Ordinal,
			&operation.Kind.Name, &operation.Kind.Version, &operation.DescriptorDigest, &operation.AuthorityDigest,
			&operation.CorrelationDigest, &admitted, &providerStarted, &elapsed, &outcome, &failureClass,
			&failureCode, &accountingMode, &completed); err != nil {
			return nil, err
		}
		operation.AdmittedAt, err = time.Parse(time.RFC3339Nano, admitted)
		if err != nil {
			return nil, err
		}
		operation.Reservation, err = operationCountersTx(ctx, tx, "v3_external_operation_allocations", "dimension", operation.OperationID)
		if err != nil {
			return nil, err
		}
		operation.Measures, err = operationCountersTx(ctx, tx, "v3_external_operation_measures", "name", operation.OperationID)
		if err != nil {
			return nil, err
		}
		if providerStarted.Valid {
			started, parseErr := time.Parse(time.RFC3339Nano, providerStarted.String)
			if parseErr != nil {
				return nil, parseErr
			}
			completion := &workflowv3.ExternalOperationCompletion{ProviderStartedAt: started, ElapsedMicros: elapsed.Int64, Outcome: outcome.String, AccountingMode: accountingMode.String}
			if failureClass.Valid {
				completion.Failure = &workflowv3.ExternalOperationFailure{Class: failureClass.String, Code: failureCode.String}
			}
			completion.CompletedAt, parseErr = time.Parse(time.RFC3339Nano, completed.String)
			if parseErr != nil {
				return nil, parseErr
			}
			completion.Counters, err = operationCountersTx(ctx, tx, "v3_external_operation_counters", "name", operation.OperationID)
			if err != nil {
				return nil, err
			}
			operation.Completion = completion
		}
		operations = append(operations, operation)
	}
	return operations, rows.Err()
}

func operationCountersTx(ctx context.Context, tx *sql.Tx, table, nameColumn, operationID string) ([]workflowv3.ExternalOperationCounter, error) {
	if table != "v3_external_operation_allocations" && table != "v3_external_operation_measures" && table != "v3_external_operation_counters" {
		return nil, fmt.Errorf("unsupported external operation counter table")
	}
	rows, err := tx.QueryContext(ctx, "SELECT "+nameColumn+",units FROM "+table+" WHERE operation_id=? ORDER BY "+nameColumn, operationID) // #nosec G201 -- table/nameColumn are closed constants above
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var counters []workflowv3.ExternalOperationCounter
	for rows.Next() {
		var counter workflowv3.ExternalOperationCounter
		if err := rows.Scan(&counter.Name, &counter.Units); err != nil {
			return nil, err
		}
		counters = append(counters, counter)
	}
	return counters, rows.Err()
}

func (s *Store) ExternalOperationProgress(ctx context.Context, runID workflowv3.RunID) (workflowv3.ExternalOperationProgress, error) {
	operations, err := s.ExternalOperations(ctx, runID)
	if err != nil {
		return workflowv3.ExternalOperationProgress{}, err
	}
	progress := workflowv3.ExternalOperationProgress{ActiveByKind: map[string]int{}, Outcomes: map[string]int{}}
	for _, operation := range operations {
		progress.Admitted++
		if operation.Completion == nil {
			progress.Incomplete++
			continue
		}
		progress.Completed++
		progress.Outcomes[operation.Completion.Outcome]++
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT operation.kind,COUNT(*)
FROM v3_external_operations operation
JOIN v3_attempts attempt ON attempt.run_id=operation.run_id
  AND attempt.node_key=operation.node_key AND attempt.attempt_no=operation.attempt_no
LEFT JOIN v3_external_operation_completions completion ON completion.operation_id=operation.operation_id
WHERE operation.run_id=? AND completion.operation_id IS NULL AND attempt.status='running'
GROUP BY operation.kind`, runID)
	if err != nil {
		return workflowv3.ExternalOperationProgress{}, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var kind string
		var count int
		if err := rows.Scan(&kind, &count); err != nil {
			return workflowv3.ExternalOperationProgress{}, err
		}
		progress.ActiveByKind[kind] = count
	}
	if err := rows.Err(); err != nil {
		return workflowv3.ExternalOperationProgress{}, err
	}
	return progress, nil
}

// ExportExternalOperations atomically publishes canonical JSONL and its compact
// manifest. The caller may safely remove a transient Workflow database only
// after both returned files exist and their digests are verified.
func (s *Store) ExportExternalOperations(ctx context.Context, runID workflowv3.RunID, jsonlPath, manifestPath string) (workflowv3.ExternalOperationExportManifest, error) {
	if jsonlPath == "" || manifestPath == "" {
		return workflowv3.ExternalOperationExportManifest{}, fmt.Errorf("external operation export paths are required")
	}
	operations, err := s.ExternalOperations(ctx, runID)
	if err != nil {
		return workflowv3.ExternalOperationExportManifest{}, err
	}
	var planDigest string
	var sequence int64
	if err := s.db.QueryRowContext(ctx, `SELECT plan_digest FROM v3_runs WHERE run_id=?`, runID).Scan(&planDigest); err != nil {
		return workflowv3.ExternalOperationExportManifest{}, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence),0) FROM v3_events WHERE run_id=?`, runID).Scan(&sequence); err != nil {
		return workflowv3.ExternalOperationExportManifest{}, err
	}
	digest, size, err := writeOperationJSONLAtomically(jsonlPath, operations)
	if err != nil {
		return workflowv3.ExternalOperationExportManifest{}, err
	}
	manifest := workflowv3.ExternalOperationExportManifest{SchemaVersion: workflowv3.ExternalOperationExportSchema, RunID: runID, PlanDigest: planDigest, EventSequence: sequence, RecordCount: len(operations), JSONLDigest: digest, JSONLSizeBytes: size, PrivacyClass: "bounded-identifiers-digests-integers"}
	seen := map[string]bool{}
	for _, operation := range operations {
		seen[operation.DescriptorDigest] = true
		if operation.Completion == nil {
			manifest.IncompleteCount++
		} else {
			manifest.CompletedCount++
		}
	}
	for descriptor := range seen {
		manifest.DescriptorDigests = append(manifest.DescriptorDigests, descriptor)
	}
	sort.Strings(manifest.DescriptorDigests)
	body, err := workflowv3.CanonicalJSON(manifest)
	if err != nil {
		return workflowv3.ExternalOperationExportManifest{}, err
	}
	if err := writeAtomically(manifestPath, append(body, '\n')); err != nil {
		return workflowv3.ExternalOperationExportManifest{}, err
	}
	return manifest, nil
}

func writeOperationJSONLAtomically(path string, operations []workflowv3.ExternalOperation) (string, int64, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, err
	}
	tmp, err := os.CreateTemp(dir, ".external-operations-*")
	if err != nil {
		return "", 0, err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	hash := sha256.New()
	writer := io.MultiWriter(tmp, hash)
	var size int64
	for _, operation := range operations {
		body, marshalErr := workflowv3.CanonicalJSON(workflowv3.ExternalOperationExportRecord{SchemaVersion: workflowv3.ExternalOperationExportSchema, Operation: operation})
		if marshalErr != nil {
			_ = tmp.Close()
			return "", 0, marshalErr
		}
		body = append(body, '\n')
		written, writeErr := writer.Write(body)
		if writeErr != nil || written != len(body) {
			_ = tmp.Close()
			if writeErr != nil {
				return "", 0, writeErr
			}
			return "", 0, io.ErrShortWrite
		}
		size += int64(written)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", 0, err
	}
	if err := tmp.Close(); err != nil {
		return "", 0, err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return "", 0, err
	}
	if err := syncDirectory(dir); err != nil {
		return "", 0, err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), size, nil
}

func writeAtomically(path string, body []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".external-operation-manifest-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	return syncDirectory(dir)
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = directory.Close() }()
	return directory.Sync()
}
