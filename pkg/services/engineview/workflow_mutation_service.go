package engineview

import (
	"context"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
)

// RetryOp explicitly repairs a terminal operation and transactionally reopens
// descendants whose required dependencies are no longer terminal blockers.
func (s *Service) RetryOp(ctx context.Context, workflowID model.WorkflowID, opID model.OpID) error {
	db, err := s.openReadDB()
	if err != nil {
		return err
	}
	if db == nil {
		return fmt.Errorf("engine db not found")
	}
	defer func() { _ = db.Close() }()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin retry op: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := time.Now().UTC()

	result, err := tx.ExecContext(ctx, `UPDATE ops
		SET status = 'ready', retry_state_json = '{}', next_attempt_at = NULL, next_attempt_at_us = NULL, updated_at = ?, updated_at_us = ?
		WHERE id = ? AND workflow_id = ? AND status = 'failed'`, now.Format(time.RFC3339Nano), now.UnixMicro(), opID, workflowID)
	if err != nil {
		return fmt.Errorf("retry op: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count retry operation: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("op %s is not in failed status", opID)
	}

	for {
		changed, err := tx.ExecContext(ctx, `UPDATE ops
			SET status = 'pending', updated_at = ?, updated_at_us = ?
			WHERE workflow_id = ? AND status = 'blocked' AND NOT EXISTS (
				SELECT 1 FROM op_dependencies d
				JOIN ops dep ON dep.id = d.depends_on_op_id
				WHERE d.op_id = ops.id AND d.required = 1 AND dep.status IN ('failed', 'blocked', 'canceled')
			)`, now.Format(time.RFC3339Nano), now.UnixMicro(), workflowID)
		if err != nil {
			return fmt.Errorf("reopen blocked descendants: %w", err)
		}
		count, err := changed.RowsAffected()
		if err != nil {
			return fmt.Errorf("count reopened descendants: %w", err)
		}
		if count == 0 {
			break
		}
	}

	if _, err := tx.ExecContext(ctx, `UPDATE workflows SET status = 'running', updated_at = ?, updated_at_us = ?
		WHERE id = ? AND status IN ('failed', 'canceled')`, now.Format(time.RFC3339Nano), now.UnixMicro(), workflowID); err != nil {
		return fmt.Errorf("reopen workflow: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit retry op: %w", err)
	}
	return nil
}

func (s *Service) CancelWorkflow(ctx context.Context, workflowID model.WorkflowID) error {
	db, err := s.openReadDB()
	if err != nil {
		return err
	}
	if db == nil {
		return fmt.Errorf("engine db not found")
	}
	defer func() { _ = db.Close() }()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin cancel workflow: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `UPDATE ops SET status = 'canceled', updated_at = ?, updated_at_us = ?
		WHERE workflow_id = ? AND status IN ('pending', 'ready', 'running', 'blocked')`, now.Format(time.RFC3339Nano), now.UnixMicro(), workflowID); err != nil {
		return fmt.Errorf("cancel ops: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM leases WHERE op_id IN (SELECT id FROM ops WHERE workflow_id = ? AND status = 'canceled')`, workflowID); err != nil {
		return fmt.Errorf("delete leases: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE workflows SET status = 'canceled', updated_at = ?, updated_at_us = ?
		WHERE id = ? AND status NOT IN ('succeeded')`, now.Format(time.RFC3339Nano), now.UnixMicro(), workflowID); err != nil {
		return fmt.Errorf("cancel workflow: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit cancel workflow: %w", err)
	}
	return nil
}
