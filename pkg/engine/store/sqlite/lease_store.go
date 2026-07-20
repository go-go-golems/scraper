package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
)

func (s *Store) LeaseReadyOp(ctx context.Context, req storecontract.LeaseRequest) (*model.OpSpec, *model.Lease, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("begin lease op: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	req.Now = req.Now.UTC()
	if req.LeaseDuration <= 0 {
		return nil, nil, fmt.Errorf("lease duration must be > 0")
	}
	policy := req.Policy.Normalize()
	activeCount, err := countActiveLeasesForQueue(ctx, tx, req.Site, req.Queue, req.Now)
	if err != nil {
		return nil, nil, err
	}
	if activeCount >= policy.MaxInFlight {
		return nil, nil, nil
	}

	limiterState, err := loadQueueLimiterState(ctx, tx, req.Site, req.Queue)
	if err != nil {
		return nil, nil, err
	}
	if policy.RateLimit != nil {
		limiterState = refillQueueLimiterState(limiterState, *policy.RateLimit, req.Now)
		if limiterState.Tokens < 1 {
			return nil, nil, nil
		}
	}

	row := tx.QueryRowContext(ctx, `SELECT id, workflow_id, parent_id, site, kind, queue_key, dedup_key, input_json, retry_json, retry_state_json, metadata_json, next_attempt_at_us, created_at_us, updated_at_us
		FROM ops
		WHERE status = ? AND queue_key = ? AND site = ?
		  AND (next_attempt_at_us IS NULL OR next_attempt_at_us <= ?)
		  AND id NOT IN (SELECT op_id FROM leases WHERE expires_at_us > ?)
		ORDER BY created_at_us ASC
		LIMIT 1`, model.OpStatusReady, req.Queue, req.Site, epochMicros(req.Now), epochMicros(req.Now))

	var op model.OpSpec
	var parentID sql.NullString
	var inputText, retryText, retryStateText, metadataText string
	var nextAttemptAt sql.NullInt64
	var createdAt, updatedAt int64
	if err := row.Scan(&op.ID, &op.WorkflowID, &parentID, &op.Site, &op.Kind, &op.Queue, &op.DedupKey,
		&inputText, &retryText, &retryStateText, &metadataText, &nextAttemptAt, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("select ready op: %w", err)
	}
	if parentID.Valid {
		parent := model.OpID(parentID.String)
		op.ParentID = &parent
	}
	op.Input = json.RawMessage(inputText)
	if err := unmarshalJSON(retryText, &op.Retry); err != nil {
		return nil, nil, fmt.Errorf("decode retry policy: %w", err)
	}
	if err := unmarshalJSON(retryStateText, &op.RetryState); err != nil {
		return nil, nil, fmt.Errorf("decode retry state: %w", err)
	}
	if err := unmarshalJSON(metadataText, &op.Metadata); err != nil {
		return nil, nil, fmt.Errorf("decode op metadata: %w", err)
	}
	op.CreatedAt = timeFromEpochMicros(createdAt)
	op.UpdatedAt = timeFromEpochMicros(updatedAt)
	if nextAttemptAt.Valid {
		readyAt := timeFromEpochMicros(nextAttemptAt.Int64)
		op.NextReadyAt = &readyAt
	}
	dependencies, err := loadDependenciesTx(ctx, tx, op.ID)
	if err != nil {
		return nil, nil, err
	}
	op.DependsOn = dependencies

	if policy.RateLimit != nil {
		limiterState.Tokens--
		if err := upsertQueueLimiterState(ctx, tx, req.Site, req.Queue, limiterState); err != nil {
			return nil, nil, err
		}
	}

	lease := model.Lease{
		WorkerID:   req.WorkerID,
		Token:      fmt.Sprintf("%s:%d", req.WorkerID, req.Now.UnixNano()),
		AcquiredAt: req.Now,
		ExpiresAt:  req.Now.Add(req.LeaseDuration),
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO leases(op_id, worker_id, token, acquired_at, expires_at, acquired_at_us, expires_at_us)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(op_id) DO UPDATE SET worker_id = excluded.worker_id, token = excluded.token,
			acquired_at = excluded.acquired_at, expires_at = excluded.expires_at,
			acquired_at_us = excluded.acquired_at_us, expires_at_us = excluded.expires_at_us`,
		op.ID, lease.WorkerID, lease.Token, lease.AcquiredAt.Format(time.RFC3339Nano), lease.ExpiresAt.Format(time.RFC3339Nano), epochMicros(lease.AcquiredAt), epochMicros(lease.ExpiresAt)); err != nil {
		return nil, nil, fmt.Errorf("upsert lease: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE ops SET status = ?, updated_at = ?, updated_at_us = ? WHERE id = ?`,
		model.OpStatusRunning, req.Now.Format(time.RFC3339Nano), epochMicros(req.Now), op.ID); err != nil {
		return nil, nil, fmt.Errorf("mark op running: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit lease op: %w", err)
	}
	return &op, &lease, nil
}

func (s *Store) HeartbeatLease(ctx context.Context, opID model.OpID, lease model.Lease, now time.Time, leaseDuration time.Duration) (*model.Lease, error) {
	if leaseDuration <= 0 {
		return nil, fmt.Errorf("lease duration must be > 0")
	}
	now = now.UTC()
	newExpiry := now.Add(leaseDuration)
	result, err := s.db.ExecContext(ctx, `UPDATE leases
		SET expires_at = ?, expires_at_us = ?
		WHERE op_id = ? AND token = ? AND expires_at_us > ?
		  AND EXISTS (SELECT 1 FROM ops WHERE ops.id = leases.op_id AND ops.status = ?)`,
		newExpiry.Format(time.RFC3339Nano), epochMicros(newExpiry), opID, lease.Token, epochMicros(now), model.OpStatusRunning)
	if err != nil {
		return nil, fmt.Errorf("heartbeat lease: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("heartbeat lease rows affected: %w", err)
	}
	if changed != 1 {
		return nil, storecontract.ErrLeaseLost
	}
	refreshed := lease
	refreshed.ExpiresAt = newExpiry
	return &refreshed, nil
}

func requireCurrentLease(ctx context.Context, tx *sql.Tx, opID model.OpID, token string, now time.Time) error {
	row := tx.QueryRowContext(ctx, `SELECT l.expires_at_us, o.status
		FROM leases l JOIN ops o ON o.id = l.op_id
		WHERE l.op_id = ? AND l.token = ?`, opID, token)
	var expiresAt int64
	var status model.OpStatus
	if err := row.Scan(&expiresAt, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storecontract.ErrLeaseLost
		}
		return fmt.Errorf("query current lease: %w", err)
	}
	if status != model.OpStatusRunning || expiresAt <= epochMicros(now) {
		return storecontract.ErrLeaseLost
	}
	return nil
}
