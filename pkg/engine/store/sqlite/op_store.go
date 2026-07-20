package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
)

func (s *Store) Enqueue(ctx context.Context, ops []model.OpSpec) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin enqueue: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := insertOps(ctx, tx, ops); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit enqueue: %w", err)
	}
	return nil
}

func (s *Store) GetOp(ctx context.Context, id model.OpID) (*model.OpSpec, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, workflow_id, parent_id, site, kind, queue_key, dedup_key, input_json, retry_json, retry_state_json, metadata_json, next_attempt_at_us, created_at_us, updated_at_us
		 FROM ops WHERE id = ?`,
		id,
	)

	var op model.OpSpec
	var parentID sql.NullString
	var inputText, retryText, retryStateText, metadataText string
	var nextAttemptAt sql.NullInt64
	var createdAt, updatedAt int64
	if err := row.Scan(
		&op.ID, &op.WorkflowID, &parentID, &op.Site, &op.Kind, &op.Queue, &op.DedupKey,
		&inputText, &retryText, &retryStateText, &metadataText, &nextAttemptAt, &createdAt, &updatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query op %s: %w", id, err)
	}

	if parentID.Valid {
		parent := model.OpID(parentID.String)
		op.ParentID = &parent
	}
	op.Input = json.RawMessage(inputText)
	if err := unmarshalJSON(retryText, &op.Retry); err != nil {
		return nil, fmt.Errorf("decode retry policy: %w", err)
	}
	if err := unmarshalJSON(retryStateText, &op.RetryState); err != nil {
		return nil, fmt.Errorf("decode retry state: %w", err)
	}
	if err := unmarshalJSON(metadataText, &op.Metadata); err != nil {
		return nil, fmt.Errorf("decode op metadata: %w", err)
	}
	op.CreatedAt = timeFromEpochMicros(createdAt)
	op.UpdatedAt = timeFromEpochMicros(updatedAt)
	if nextAttemptAt.Valid {
		readyAt := timeFromEpochMicros(nextAttemptAt.Int64)
		op.NextReadyAt = &readyAt
	}

	dependencies, err := s.loadDependencies(ctx, id)
	if err != nil {
		return nil, err
	}
	op.DependsOn = dependencies
	return &op, nil
}

func (s *Store) RefreshRunnableOps(ctx context.Context, now time.Time) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin refresh runnable ops: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now = now.UTC()
	nowUS := epochMicros(now)
	totalChanged := 0

	recovered, err := execRowsAffected(ctx, tx, `UPDATE ops
		SET status = ?, updated_at = ?, updated_at_us = ?
		WHERE status = ? AND EXISTS (
			SELECT 1 FROM leases
			WHERE leases.op_id = ops.id AND leases.expires_at_us <= ?
		)`, model.OpStatusReady, now.Format(time.RFC3339Nano), nowUS, model.OpStatusRunning, nowUS)
	if err != nil {
		return 0, fmt.Errorf("recover expired leases: %w", err)
	}
	totalChanged += recovered

	if _, err := tx.ExecContext(ctx, `DELETE FROM leases WHERE expires_at_us <= ?`, nowUS); err != nil {
		return 0, fmt.Errorf("delete expired leases: %w", err)
	}

	for {
		canceled, err := execRowsAffected(ctx, tx, `UPDATE ops
			SET status = ?, updated_at = ?, updated_at_us = ?
			WHERE status = ? AND EXISTS (
				SELECT 1 FROM op_dependencies d
				JOIN ops dep ON dep.id = d.depends_on_op_id
				WHERE d.op_id = ops.id AND d.required = 1 AND dep.status IN (?, ?)
			)`, model.OpStatusCanceled, now.Format(time.RFC3339Nano), nowUS,
			model.OpStatusPending, model.OpStatusFailed, model.OpStatusCanceled)
		if err != nil {
			return 0, fmt.Errorf("cancel blocked ops: %w", err)
		}
		totalChanged += canceled
		if canceled == 0 {
			break
		}
	}

	ready, err := execRowsAffected(ctx, tx, `UPDATE ops
		SET status = ?, updated_at = ?, updated_at_us = ?
		WHERE status = ? AND NOT EXISTS (
			SELECT 1 FROM op_dependencies d
			JOIN ops dep ON dep.id = d.depends_on_op_id
			WHERE d.op_id = ops.id AND (
				(d.required = 1 AND dep.status != ?) OR
				(d.required = 0 AND dep.status NOT IN (?, ?, ?))
			)
		)`, model.OpStatusReady, now.Format(time.RFC3339Nano), nowUS, model.OpStatusPending,
		model.OpStatusSucceeded, model.OpStatusSucceeded, model.OpStatusFailed, model.OpStatusCanceled)
	if err != nil {
		return 0, fmt.Errorf("promote pending ops: %w", err)
	}
	totalChanged += ready

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit refresh runnable ops: %w", err)
	}
	return totalChanged, nil
}

func (s *Store) ListQueueCandidates(ctx context.Context, now time.Time) ([]storecontract.QueueCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT ops.site, ops.queue_key
		FROM ops
		WHERE ops.status = ?
		  AND (ops.next_attempt_at_us IS NULL OR ops.next_attempt_at_us <= ?)
		ORDER BY ops.site, ops.queue_key`, model.OpStatusReady, epochMicros(now))
	if err != nil {
		return nil, fmt.Errorf("list queue candidates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	ret := []storecontract.QueueCandidate{}
	for rows.Next() {
		var candidate storecontract.QueueCandidate
		if err := rows.Scan(&candidate.Site, &candidate.Queue); err != nil {
			return nil, fmt.Errorf("scan queue candidate: %w", err)
		}
		ret = append(ret, candidate)
	}
	return ret, rows.Err()
}

func (s *Store) loadDependencies(ctx context.Context, opID model.OpID) ([]model.Dependency, error) {
	return loadDependenciesTx(ctx, s.db, opID)
}
