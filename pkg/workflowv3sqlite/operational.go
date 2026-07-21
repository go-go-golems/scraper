package workflowv3sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

// OperationalSnapshot reconstructs one coherent operator view from a single
// read transaction. Events wake consumers; this snapshot remains the truth.
func (s *Store) OperationalSnapshot(
	ctx context.Context,
	runID *workflowv3.RunID,
	registry workflowv3.RegistryResolver,
	capacities map[string]int,
	now time.Time,
) (workflowv3.OperationalSnapshot, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return workflowv3.OperationalSnapshot{}, err
	}
	defer func() { _ = tx.Rollback() }()
	snapshot := workflowv3.OperationalSnapshot{
		AsOf: now.UTC(), RunStatuses: map[string]int{}, NodeStatuses: map[string]int{},
		AttemptStatuses: map[string]int{},
	}
	eventQuery := "SELECT COALESCE(MAX(sequence), 0) FROM v3_events"
	var eventArgs []any
	if runID != nil {
		eventQuery += " WHERE run_id = ?"
		eventArgs = append(eventArgs, *runID)
	}
	if err := tx.QueryRowContext(ctx, eventQuery, eventArgs...).Scan(&snapshot.EventSequence); err != nil {
		return snapshot, err
	}
	if err := groupedStatus(ctx, tx, "v3_runs", "", runID, snapshot.RunStatuses); err != nil {
		return snapshot, err
	}
	if err := groupedStatus(ctx, tx, "v3_nodes", "run_id", runID, snapshot.NodeStatuses); err != nil {
		return snapshot, err
	}
	if err := groupedStatus(ctx, tx, "v3_attempts", "run_id", runID, snapshot.AttemptStatuses); err != nil {
		return snapshot, err
	}
	where, args := runWhere(runID, "run_id")
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM v3_attempts`+where+andWhere(where)+`attempt_no > 1`, args...).Scan(&snapshot.RetryAttempts); err != nil {
		return snapshot, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM v3_attempts`+where+andWhere(where)+`status = 'lease_lost'`, args...).Scan(&snapshot.LeaseLosses); err != nil {
		return snapshot, err
	}
	var oldest sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT MIN(created_at) FROM v3_runs`+where+andWhere(where)+`status = 'running'`, args...).Scan(&oldest); err != nil {
		return snapshot, err
	}
	if oldest.Valid {
		created, err := time.Parse(time.RFC3339Nano, oldest.String)
		if err != nil {
			return snapshot, err
		}
		if now.After(created) {
			snapshot.OldestRunningAgeMS = now.Sub(created).Milliseconds()
		}
	}
	for _, seconds := range []int64{60, 300} {
		boundary := formatTime(now.Add(-time.Duration(seconds) * time.Second))
		query := `
SELECT COUNT(*),
  COALESCE(SUM(CASE WHEN status = 'succeeded' THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0)
FROM v3_attempts WHERE finished_at IS NOT NULL
  AND julianday(finished_at) >= julianday(?)`
		rateArgs := []any{boundary}
		if runID != nil {
			query += " AND run_id = ?"
			rateArgs = append(rateArgs, *runID)
		}
		rate := workflowv3.TerminalRate{WindowSeconds: seconds}
		if err := tx.QueryRowContext(ctx, query, rateArgs...).Scan(&rate.Terminal, &rate.Succeeded, &rate.Failed); err != nil {
			return snapshot, err
		}
		snapshot.Rates = append(snapshot.Rates, rate)
	}
	snapshot.Queue, err = queueSnapshot(ctx, tx, registry, capacities, runID, now)
	if err != nil {
		return snapshot, err
	}
	budgetQuery := `
SELECT run_id, account, dimension, limit_units, used_units, reserved_units,
  limit_units - used_units - reserved_units, version, policy_digest
FROM v3_budget_accounts`
	budgetArgs := []any{}
	if runID != nil {
		budgetQuery += " WHERE run_id = ?"
		budgetArgs = append(budgetArgs, *runID)
	}
	budgetQuery += " ORDER BY run_id, account, dimension"
	rows, err := tx.QueryContext(ctx, budgetQuery, budgetArgs...)
	if err != nil {
		return snapshot, err
	}
	for rows.Next() {
		var item workflowv3.BudgetProgress
		if err := rows.Scan(
			&item.RunID, &item.Account, &item.Dimension, &item.Limit,
			&item.Used, &item.Reserved, &item.Remaining, &item.Version,
			&item.PolicyDigest,
		); err != nil {
			_ = rows.Close()
			return snapshot, err
		}
		snapshot.Budgets = append(snapshot.Budgets, item)
	}
	if err := rows.Close(); err != nil {
		return snapshot, err
	}
	if err := tx.Commit(); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

func (s *Store) EventsAfter(
	ctx context.Context,
	after int64,
	runID *workflowv3.RunID,
	limit int,
) ([]workflowv3.OperationalEvent, error) {
	if after < 0 || limit < 1 || limit > 1000 {
		return nil, fmt.Errorf("event cursor and limit are invalid")
	}
	query := `
SELECT sequence, run_id, COALESCE(node_key, ''), event_type, payload_json,
  created_at
FROM v3_events WHERE sequence > ?`
	args := []any{after}
	if runID != nil {
		query += " AND run_id = ?"
		args = append(args, *runID)
	}
	query += " ORDER BY sequence LIMIT ?"
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var events []workflowv3.OperationalEvent
	for rows.Next() {
		var event workflowv3.OperationalEvent
		var body []byte
		var created string
		if err := rows.Scan(
			&event.Sequence, &event.RunID, &event.NodeKey, &event.Type, &body,
			&created,
		); err != nil {
			return nil, err
		}
		event.DataJSON = string(body)
		event.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func groupedStatus(
	ctx context.Context,
	tx *sql.Tx,
	table, runColumn string,
	runID *workflowv3.RunID,
	destination map[string]int,
) error {
	query := fmt.Sprintf("SELECT status, COUNT(*) FROM %s", table)
	var args []any
	if runID != nil {
		if runColumn == "" {
			runColumn = "run_id"
		}
		query += " WHERE " + runColumn + " = ?"
		args = append(args, *runID)
	}
	query += " GROUP BY status"
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return err
		}
		destination[status] = count
	}
	return rows.Err()
}

func runWhere(runID *workflowv3.RunID, column string) (string, []any) {
	if runID == nil {
		return "", nil
	}
	return " WHERE " + column + " = ?", []any{*runID}
}

func andWhere(where string) string {
	if where == "" {
		return " WHERE "
	}
	return " AND "
}
