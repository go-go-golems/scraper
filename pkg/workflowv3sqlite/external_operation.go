package workflowv3sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

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
	checks := []struct {
		name  string
		query string
	}{
		{
			name: "allocations without admissions",
			query: `SELECT COUNT(*) FROM v3_external_operation_allocations allocation
WHERE NOT EXISTS (
  SELECT 1 FROM v3_external_operations operation
  WHERE operation.operation_id = allocation.operation_id
)`,
		},
		{
			name: "measures without admissions",
			query: `SELECT COUNT(*) FROM v3_external_operation_measures measure
WHERE NOT EXISTS (
  SELECT 1 FROM v3_external_operations operation
  WHERE operation.operation_id = measure.operation_id
)`,
		},
		{
			name: "completions without admissions",
			query: `SELECT COUNT(*) FROM v3_external_operation_completions completion
WHERE NOT EXISTS (
  SELECT 1 FROM v3_external_operations operation
  WHERE operation.operation_id = completion.operation_id
)`,
		},
		{
			name: "counters without completions",
			query: `SELECT COUNT(*) FROM v3_external_operation_counters counter
WHERE NOT EXISTS (
  SELECT 1 FROM v3_external_operation_completions completion
  WHERE completion.operation_id = counter.operation_id
)`,
		},
		{
			name: "admissions without attempts",
			query: `SELECT COUNT(*) FROM v3_external_operations operation
WHERE NOT EXISTS (
  SELECT 1 FROM v3_attempts attempt
  WHERE attempt.run_id = operation.run_id
    AND attempt.node_key = operation.node_key
    AND attempt.attempt_no = operation.attempt_no
)`,
		},
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
