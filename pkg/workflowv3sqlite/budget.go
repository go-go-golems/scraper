package workflowv3sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

var budgetOperatorPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._@-]{0,127}$`)

var (
	ErrBudgetUsageInvalid            = errors.New("budget usage invalid")
	ErrBudgetUsageExceedsReservation = errors.New("budget usage exceeds reservation")
)

func checkBudgetInvariants(ctx context.Context, db *sql.DB) error {
	var inconsistent int
	if err := db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM v3_budget_accounts account
WHERE account.reserved_units != COALESCE((
  SELECT SUM(reservation.reserved_units)
  FROM v3_budget_reservations reservation
  WHERE reservation.run_id = account.run_id
    AND reservation.account = account.account
    AND reservation.dimension = account.dimension
    AND reservation.status = 'reserved'
), 0)`).Scan(&inconsistent); err != nil {
		return err
	}
	if inconsistent != 0 {
		return fmt.Errorf("BUDGET_ACCOUNT_INVARIANT: %d account totals disagree", inconsistent)
	}
	if err := db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM v3_budget_reservations reservation
JOIN v3_attempts attempt
  ON attempt.run_id = reservation.run_id
 AND attempt.node_key = reservation.node_key
 AND attempt.attempt_no = reservation.attempt_no
JOIN v3_nodes node
  ON node.run_id = reservation.run_id AND node.node_key = reservation.node_key
WHERE reservation.status = 'reserved'
  AND (attempt.status != 'running' OR node.status != 'running')`).Scan(&inconsistent); err != nil {
		return err
	}
	if inconsistent != 0 {
		return fmt.Errorf("BUDGET_ACCOUNT_INVARIANT: %d reservations lack running attempts", inconsistent)
	}
	return nil
}

func insertNodeBudget(
	ctx context.Context,
	tx *sql.Tx,
	runID workflowv3.RunID,
	nodeKey workflowv3.NodeKey,
	claim *workflowv3.PlanBudgetClaim,
) error {
	if claim == nil {
		return nil
	}
	for _, amount := range claim.Effective {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO v3_node_budget_claims(run_id, node_key, dimension, reserve_units)
VALUES (?, ?, ?, ?)`, runID, nodeKey, amount.Dimension, amount.Units); err != nil {
			return fmt.Errorf("insert node budget %s/%s: %w", nodeKey, amount.Dimension, err)
		}
	}
	return nil
}

func loadNodeBudget(
	ctx context.Context,
	tx *sql.Tx,
	runID workflowv3.RunID,
	nodeKey workflowv3.NodeKey,
	account, onExhausted sql.NullString,
) (*workflowv3.PlanBudgetClaim, error) {
	if !account.Valid {
		return nil, nil
	}
	rows, err := tx.QueryContext(ctx, `
SELECT dimension, reserve_units FROM v3_node_budget_claims
WHERE run_id = ? AND node_key = ? ORDER BY dimension`, runID, nodeKey)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	claim := &workflowv3.PlanBudgetClaim{Account: account.String, OnExhausted: onExhausted.String}
	for rows.Next() {
		var amount workflowv3.BudgetAmount
		if err := rows.Scan(&amount.Dimension, &amount.Units); err != nil {
			return nil, err
		}
		claim.Effective = append(claim.Effective, amount)
		claim.Requested = append(claim.Requested, amount)
	}
	return claim, rows.Err()
}

func reserveNodeBudget(
	ctx context.Context,
	tx *sql.Tx,
	runID workflowv3.RunID,
	nodeKey workflowv3.NodeKey,
	attempt int,
	claim *workflowv3.PlanBudgetClaim,
	now time.Time,
) error {
	if claim == nil {
		return nil
	}
	stamp := formatTime(now)
	for _, amount := range claim.Effective {
		result, err := tx.ExecContext(ctx, `
UPDATE v3_budget_accounts
SET reserved_units = reserved_units + ?, updated_at = ?
WHERE run_id = ? AND account = ? AND dimension = ?
  AND used_units + reserved_units + ? <= limit_units`,
			amount.Units, stamp, runID, claim.Account, amount.Dimension, amount.Units)
		if err != nil {
			return err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return fmt.Errorf("%w: %s/%s", ErrBudgetUsageInvalid, claim.Account, amount.Dimension)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO v3_budget_reservations(
  run_id, node_key, attempt_no, account, dimension, reserved_units,
  status, created_at
) VALUES (?, ?, ?, ?, ?, ?, 'reserved', ?)`,
			runID, nodeKey, attempt, claim.Account, amount.Dimension,
			amount.Units, stamp); err != nil {
			return err
		}
	}
	if err := insertEvent(ctx, tx, runID, nodeKey, "budget.reserved", map[string]any{
		"attempt": attempt, "account": claim.Account,
		"dimensions": len(claim.Effective),
	}, now); err != nil {
		return err
	}
	return nil
}

func settleAttemptBudget(
	ctx context.Context,
	tx *sql.Tx,
	lease workflowv3.Lease,
	usage []workflowv3.BudgetAmount,
	mode string,
	now time.Time,
) error {
	rows, err := tx.QueryContext(ctx, `
SELECT account, dimension, reserved_units
FROM v3_budget_reservations
WHERE run_id = ? AND node_key = ? AND attempt_no = ? AND status = 'reserved'
ORDER BY dimension`, lease.RunID, lease.NodeKey, lease.Attempt)
	if err != nil {
		return err
	}
	type reservation struct {
		account, dimension string
		reserved           int64
	}
	var reservations []reservation
	for rows.Next() {
		var item reservation
		if err := rows.Scan(&item.account, &item.dimension, &item.reserved); err != nil {
			_ = rows.Close()
			return err
		}
		reservations = append(reservations, item)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if len(reservations) == 0 {
		if len(usage) != 0 {
			return fmt.Errorf("%w: unbudgeted attempt reported usage", ErrBudgetUsageInvalid)
		}
		return nil
	}
	actual := make(map[string]int64, len(usage))
	previous := ""
	for _, amount := range usage {
		if amount.Dimension <= previous || amount.Units < 0 {
			return fmt.Errorf("%w: dimensions must be sorted, unique, and nonnegative", ErrBudgetUsageInvalid)
		}
		actual[amount.Dimension] = amount.Units
		previous = amount.Dimension
	}
	stamp := formatTime(now)
	for _, reservation := range reservations {
		settled := int64(0)
		status := "released"
		switch mode {
		case "actual":
			var ok bool
			settled, ok = actual[reservation.dimension]
			if !ok {
				return fmt.Errorf("%w: missing %s", ErrBudgetUsageInvalid, reservation.dimension)
			}
			if settled > reservation.reserved {
				return fmt.Errorf("%w: %s", ErrBudgetUsageExceedsReservation, reservation.dimension)
			}
			status = "settled"
		case "conservative":
			settled = reservation.reserved
			status = "conservative"
		case "release":
		default:
			return fmt.Errorf("unknown budget settlement mode %q", mode)
		}
		delete(actual, reservation.dimension)
		result, err := tx.ExecContext(ctx, `
UPDATE v3_budget_accounts
SET reserved_units = reserved_units - ?, used_units = used_units + ?, updated_at = ?
WHERE run_id = ? AND account = ? AND dimension = ?
  AND reserved_units >= ?`, reservation.reserved, settled, stamp,
			lease.RunID, reservation.account, reservation.dimension, reservation.reserved)
		if err != nil {
			return err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return fmt.Errorf("budget account invariant for %s/%s", reservation.account, reservation.dimension)
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE v3_budget_reservations
SET settled_units = ?, status = ?, settled_at = ?
WHERE run_id = ? AND node_key = ? AND attempt_no = ? AND dimension = ?
  AND status = 'reserved'`, settled, status, stamp, lease.RunID, lease.NodeKey,
			lease.Attempt, reservation.dimension); err != nil {
			return err
		}
	}
	if len(actual) != 0 {
		keys := make([]string, 0, len(actual))
		for dimension := range actual {
			keys = append(keys, dimension)
		}
		sort.Strings(keys)
		return fmt.Errorf("%w: undeclared %s", ErrBudgetUsageInvalid, keys[0])
	}
	if err := insertEvent(ctx, tx, lease.RunID, lease.NodeKey, "budget.settled", map[string]any{
		"attempt": lease.Attempt, "mode": mode,
	}, now); err != nil {
		return err
	}
	return nil
}

func settleConservativeWhere(
	ctx context.Context,
	tx *sql.Tx,
	whereSQL string,
	args []any,
	now time.Time,
) error {
	query := `
SELECT run_id, node_key, attempt_no FROM v3_budget_reservations
WHERE status = 'reserved' AND ` + whereSQL + `
GROUP BY run_id, node_key, attempt_no`
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	var leases []workflowv3.Lease
	for rows.Next() {
		var lease workflowv3.Lease
		if err := rows.Scan(&lease.RunID, &lease.NodeKey, &lease.Attempt); err != nil {
			_ = rows.Close()
			return err
		}
		leases = append(leases, lease)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, lease := range leases {
		if err := settleAttemptBudget(ctx, tx, lease, nil, "conservative", now); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) BudgetSnapshot(
	ctx context.Context,
	runID *workflowv3.RunID,
) ([]workflowv3.BudgetProgress, error) {
	query := `
SELECT run_id, account, dimension, limit_units, used_units, reserved_units,
  limit_units - used_units - reserved_units, version, policy_digest
FROM v3_budget_accounts`
	var args []any
	if runID != nil {
		query += " WHERE run_id = ?"
		args = append(args, *runID)
	}
	query += " ORDER BY run_id, account, dimension"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var progress []workflowv3.BudgetProgress
	for rows.Next() {
		var item workflowv3.BudgetProgress
		if err := rows.Scan(
			&item.RunID, &item.Account, &item.Dimension, &item.Limit,
			&item.Used, &item.Reserved, &item.Remaining, &item.Version,
			&item.PolicyDigest,
		); err != nil {
			return nil, err
		}
		progress = append(progress, item)
	}
	return progress, rows.Err()
}

func (s *Store) IncreaseBudget(
	ctx context.Context,
	runID workflowv3.RunID,
	account, dimension string,
	delta, expectedVersion int64,
	actor string,
	now time.Time,
) error {
	if delta <= 0 || expectedVersion < 1 {
		return fmt.Errorf("budget increase and expected version must be positive")
	}
	if !budgetOperatorPattern.MatchString(actor) {
		return fmt.Errorf("budget operator identity is invalid")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var minimum, maximum int64
	if err := tx.QueryRowContext(ctx, `
SELECT MIN(version), MAX(version) FROM v3_budget_accounts
WHERE run_id = ? AND account = ?`, runID, account).Scan(&minimum, &maximum); err != nil {
		return err
	}
	if minimum != expectedVersion || maximum != expectedVersion {
		return fmt.Errorf("budget account version conflict")
	}
	result, err := tx.ExecContext(ctx, `
UPDATE v3_budget_accounts
SET limit_units = limit_units + ?, updated_at = ?
WHERE run_id = ? AND account = ? AND dimension = ?
  AND limit_units <= 9223372036854775807 - ?`,
		delta, formatTime(now), runID, account, dimension, delta)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return fmt.Errorf("budget account dimension is missing or increase overflows")
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_budget_accounts SET version = version + 1, updated_at = ?
WHERE run_id = ? AND account = ?`, formatTime(now), runID, account); err != nil {
		return err
	}
	if err := insertEvent(ctx, tx, runID, "", "budget.increased", map[string]any{
		"account": account, "dimension": dimension, "delta": delta,
		"version": expectedVersion + 1, "actor": actor,
	}, now); err != nil {
		return err
	}
	return tx.Commit()
}

func failExhaustedBudgetNodes(ctx context.Context, tx *sql.Tx, now time.Time) error {
	stamp := formatTime(now)
	rows, err := tx.QueryContext(ctx, `
SELECT n.run_id, n.node_key, n.budget_account, MIN(claim.dimension)
FROM v3_nodes n
JOIN v3_node_budget_claims claim
  ON claim.run_id = n.run_id AND claim.node_key = n.node_key
JOIN v3_budget_accounts account
  ON account.run_id = claim.run_id
 AND account.account = n.budget_account
 AND account.dimension = claim.dimension
WHERE n.status = 'pending' AND n.budget_on_exhausted = 'fail-run'
  AND account.used_units + account.reserved_units + claim.reserve_units
      > account.limit_units
GROUP BY n.run_id, n.node_key, n.budget_account`)
	if err != nil {
		return err
	}
	type exhausted struct {
		runID              workflowv3.RunID
		nodeKey            workflowv3.NodeKey
		account, dimension string
	}
	var exhaustedNodes []exhausted
	for rows.Next() {
		var item exhausted
		if err := rows.Scan(&item.runID, &item.nodeKey, &item.account, &item.dimension); err != nil {
			_ = rows.Close()
			return err
		}
		exhaustedNodes = append(exhaustedNodes, item)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, item := range exhaustedNodes {
		if err := insertEvent(ctx, tx, item.runID, item.nodeKey, "budget.exhausted", map[string]any{
			"account": item.account, "dimension": item.dimension,
			"policy": workflowv3.BudgetExhaustFailRun,
			"class":  "budget", "code": "BUDGET_EXHAUSTED",
		}, now); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_nodes SET status = 'failed'
WHERE status = 'pending' AND budget_on_exhausted = 'fail-run'
  AND EXISTS (
    SELECT 1 FROM v3_node_budget_claims claim
    JOIN v3_budget_accounts account
      ON account.run_id = claim.run_id
     AND account.account = v3_nodes.budget_account
     AND account.dimension = claim.dimension
    WHERE claim.run_id = v3_nodes.run_id
      AND claim.node_key = v3_nodes.node_key
      AND account.used_units + account.reserved_units + claim.reserve_units
          > account.limit_units
  )`); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
UPDATE v3_runs SET status = 'failed', updated_at = ?
WHERE status = 'running' AND EXISTS (
  SELECT 1 FROM v3_nodes n
  WHERE n.run_id = v3_runs.run_id AND n.status = 'failed'
    AND n.budget_on_exhausted = 'fail-run'
)`, stamp)
	return err
}
