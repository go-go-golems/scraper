package engineview

import (
	"context"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
)

func (s *Service) RetryOp(ctx context.Context, workflowID model.WorkflowID, opID model.OpID) error {
	db, err := s.openReadDB()
	if err != nil {
		return err
	}
	if db == nil {
		return fmt.Errorf("engine db not found")
	}
	defer func() { _ = db.Close() }()

	result, err := db.ExecContext(ctx,
		`UPDATE ops SET status = 'ready', retry_state_json = '{}', next_attempt_at = NULL, updated_at = ?
		 WHERE id = ? AND workflow_id = ? AND status = 'failed'`,
		time.Now().UTC().Format(time.RFC3339Nano), opID, workflowID,
	)
	if err != nil {
		return fmt.Errorf("retry op: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("op %s is not in failed status", opID)
	}
	_, _ = db.ExecContext(ctx,
		`UPDATE workflows SET status = 'running', updated_at = ? WHERE id = ? AND status IN ('failed', 'canceled')`,
		time.Now().UTC().Format(time.RFC3339Nano), workflowID,
	)
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
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC().Format(time.RFC3339Nano)

	if _, err := tx.ExecContext(ctx,
		`UPDATE ops SET status = 'canceled', updated_at = ? WHERE workflow_id = ? AND status IN ('pending', 'ready', 'running')`,
		now, workflowID,
	); err != nil {
		return fmt.Errorf("cancel ops: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM leases WHERE op_id IN (SELECT id FROM ops WHERE workflow_id = ? AND status = 'canceled')`,
		workflowID,
	); err != nil {
		return fmt.Errorf("delete leases: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE workflows SET status = 'canceled', updated_at = ? WHERE id = ? AND status NOT IN ('succeeded')`,
		now, workflowID,
	); err != nil {
		return fmt.Errorf("cancel workflow: %w", err)
	}

	return tx.Commit()
}
