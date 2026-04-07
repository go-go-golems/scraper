package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, dsn string) (*Store, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	if err := migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) CurrentVersion(ctx context.Context) (int, error) {
	return currentVersion(ctx, s.db)
}

func (s *Store) Enqueue(ctx context.Context, ops []model.OpSpec) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin enqueue: %w", err)
	}

	if err := insertOps(ctx, tx, ops); err != nil {
		_ = tx.Rollback()
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
		`SELECT id, workflow_id, parent_id, site, kind, queue_key, dedup_key, input_json, retry_json, retry_state_json, metadata_json, next_attempt_at, created_at, updated_at
		 FROM ops WHERE id = ?`,
		id,
	)

	var op model.OpSpec
	var parentID sql.NullString
	var inputText string
	var retryText string
	var retryStateText string
	var metadataText string
	var nextAttemptText sql.NullString
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&op.ID,
		&op.WorkflowID,
		&parentID,
		&op.Site,
		&op.Kind,
		&op.Queue,
		&op.DedupKey,
		&inputText,
		&retryText,
		&retryStateText,
		&metadataText,
		&nextAttemptText,
		&createdAt,
		&updatedAt,
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
	op.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	op.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	if nextAttemptText.Valid && nextAttemptText.String != "" {
		nextAttemptAt, err := time.Parse(time.RFC3339Nano, nextAttemptText.String)
		if err != nil {
			return nil, fmt.Errorf("parse next attempt time: %w", err)
		}
		op.NextReadyAt = &nextAttemptAt
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

	totalChanged := 0

	recovered, err := execRowsAffected(
		ctx,
		tx,
		`UPDATE ops
		 SET status = ?, updated_at = ?
		 WHERE status = ?
		   AND EXISTS (
		     SELECT 1 FROM leases
		     WHERE leases.op_id = ops.id
		       AND leases.expires_at <= ?
		   )`,
		model.OpStatusReady,
		now.UTC().Format(time.RFC3339Nano),
		model.OpStatusRunning,
		now.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("recover expired leases: %w", err)
	}
	totalChanged += recovered

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM leases WHERE expires_at <= ?`,
		now.UTC().Format(time.RFC3339Nano),
	); err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("delete expired leases: %w", err)
	}

	for {
		canceled, err := execRowsAffected(
			ctx,
			tx,
			`UPDATE ops
			 SET status = ?, updated_at = ?
			 WHERE status = ?
			   AND EXISTS (
			     SELECT 1
			     FROM op_dependencies d
			     JOIN ops dep ON dep.id = d.depends_on_op_id
			     WHERE d.op_id = ops.id
			       AND d.required = 1
			       AND dep.status IN (?, ?)
			   )`,
			model.OpStatusCanceled,
			now.UTC().Format(time.RFC3339Nano),
			model.OpStatusPending,
			model.OpStatusFailed,
			model.OpStatusCanceled,
		)
		if err != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("cancel blocked ops: %w", err)
		}
		totalChanged += canceled
		if canceled == 0 {
			break
		}
	}

	ready, err := execRowsAffected(
		ctx,
		tx,
		`UPDATE ops
		 SET status = ?, updated_at = ?
		 WHERE status = ?
		   AND NOT EXISTS (
		     SELECT 1
		     FROM op_dependencies d
		     JOIN ops dep ON dep.id = d.depends_on_op_id
		     WHERE d.op_id = ops.id
		       AND (
		         (d.required = 1 AND dep.status != ?)
		         OR
		         (d.required = 0 AND dep.status NOT IN (?, ?, ?))
		       )
		   )`,
		model.OpStatusReady,
		now.UTC().Format(time.RFC3339Nano),
		model.OpStatusPending,
		model.OpStatusSucceeded,
		model.OpStatusSucceeded,
		model.OpStatusFailed,
		model.OpStatusCanceled,
	)
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("promote pending ops: %w", err)
	}
	totalChanged += ready

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit refresh runnable ops: %w", err)
	}

	return totalChanged, nil
}

func (s *Store) ListQueueCandidates(ctx context.Context, now time.Time) ([]storecontract.QueueCandidate, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT DISTINCT ops.site, ops.queue_key
		 FROM ops
		 WHERE ops.status = ?
		   AND (ops.next_attempt_at IS NULL OR ops.next_attempt_at <= ?)
		 ORDER BY ops.site, ops.queue_key`,
		model.OpStatusReady,
		now.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, fmt.Errorf("list queue candidates: %w", err)
	}
	defer rows.Close()

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

func (s *Store) LeaseReadyOp(
	ctx context.Context,
	req storecontract.LeaseRequest,
) (*model.OpSpec, *model.Lease, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("begin lease op: %w", err)
	}

	policy := req.Policy.Normalize()

	activeCount, err := countActiveLeasesForQueue(ctx, tx, req.Site, req.Queue, req.Now)
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, err
	}
	if activeCount >= policy.MaxInFlight {
		_ = tx.Rollback()
		return nil, nil, nil
	}

	limiterState, err := loadQueueLimiterState(ctx, tx, req.Site, req.Queue)
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, err
	}
	if policy.RateLimit != nil {
		limiterState = refillQueueLimiterState(limiterState, *policy.RateLimit, req.Now)
		if limiterState.Tokens < 1 {
			_ = tx.Rollback()
			return nil, nil, nil
		}
	}

	row := tx.QueryRowContext(
		ctx,
		`SELECT id, workflow_id, parent_id, site, kind, queue_key, dedup_key, input_json, retry_json, retry_state_json, metadata_json, next_attempt_at, created_at, updated_at
		 FROM ops
		 WHERE status = ?
		   AND queue_key = ?
		   AND site = ?
		   AND (next_attempt_at IS NULL OR next_attempt_at <= ?)
		   AND id NOT IN (
		     SELECT op_id FROM leases WHERE expires_at > ?
		   )
		 ORDER BY created_at ASC
		 LIMIT 1`,
		model.OpStatusReady,
		req.Queue,
		req.Site,
		req.Now.UTC().Format(time.RFC3339Nano),
		req.Now.UTC().Format(time.RFC3339Nano),
	)

	var op model.OpSpec
	var parentID sql.NullString
	var inputText string
	var retryText string
	var retryStateText string
	var metadataText string
	var nextAttemptText sql.NullString
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&op.ID,
		&op.WorkflowID,
		&parentID,
		&op.Site,
		&op.Kind,
		&op.Queue,
		&op.DedupKey,
		&inputText,
		&retryText,
		&retryStateText,
		&metadataText,
		&nextAttemptText,
		&createdAt,
		&updatedAt,
	); err != nil {
		_ = tx.Rollback()
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("select ready op: %w", err)
	}

	if policy.RateLimit != nil {
		limiterState.Tokens -= 1
		if err := upsertQueueLimiterState(ctx, tx, req.Site, req.Queue, limiterState); err != nil {
			_ = tx.Rollback()
			return nil, nil, err
		}
	}

	if parentID.Valid {
		parent := model.OpID(parentID.String)
		op.ParentID = &parent
	}
	op.Input = json.RawMessage(inputText)
	if err := unmarshalJSON(retryText, &op.Retry); err != nil {
		_ = tx.Rollback()
		return nil, nil, fmt.Errorf("decode retry policy: %w", err)
	}
	if err := unmarshalJSON(retryStateText, &op.RetryState); err != nil {
		_ = tx.Rollback()
		return nil, nil, fmt.Errorf("decode retry state: %w", err)
	}
	if err := unmarshalJSON(metadataText, &op.Metadata); err != nil {
		_ = tx.Rollback()
		return nil, nil, fmt.Errorf("decode op metadata: %w", err)
	}
	op.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	op.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	if nextAttemptText.Valid && nextAttemptText.String != "" {
		nextAttemptAt, err := time.Parse(time.RFC3339Nano, nextAttemptText.String)
		if err != nil {
			_ = tx.Rollback()
			return nil, nil, fmt.Errorf("parse next attempt time: %w", err)
		}
		op.NextReadyAt = &nextAttemptAt
	}

	dependencies, err := loadDependenciesTx(ctx, tx, op.ID)
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, err
	}
	op.DependsOn = dependencies

	lease := model.Lease{
		WorkerID:   req.WorkerID,
		Token:      fmt.Sprintf("%s:%d", req.WorkerID, req.Now.UTC().UnixNano()),
		AcquiredAt: req.Now.UTC(),
		ExpiresAt:  req.Now.UTC().Add(req.LeaseDuration),
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO leases(op_id, worker_id, token, acquired_at, expires_at)
		 VALUES(?, ?, ?, ?, ?)
		 ON CONFLICT(op_id) DO UPDATE SET
		   worker_id = excluded.worker_id,
		   token = excluded.token,
		   acquired_at = excluded.acquired_at,
		   expires_at = excluded.expires_at`,
		op.ID,
		lease.WorkerID,
		lease.Token,
		lease.AcquiredAt.Format(time.RFC3339Nano),
		lease.ExpiresAt.Format(time.RFC3339Nano),
	); err != nil {
		_ = tx.Rollback()
		return nil, nil, fmt.Errorf("upsert lease: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE ops SET status = ?, updated_at = ? WHERE id = ?`,
		model.OpStatusRunning,
		req.Now.UTC().Format(time.RFC3339Nano),
		op.ID,
	); err != nil {
		_ = tx.Rollback()
		return nil, nil, fmt.Errorf("mark op running: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit lease op: %w", err)
	}

	return &op, &lease, nil
}

func (s *Store) HeartbeatLease(
	ctx context.Context,
	opID model.OpID,
	lease model.Lease,
	extendBy time.Duration,
) error {
	newExpiry := lease.ExpiresAt.UTC().Add(extendBy)
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE leases SET expires_at = ? WHERE op_id = ? AND token = ?`,
		newExpiry.Format(time.RFC3339Nano),
		opID,
		lease.Token,
	)
	if err != nil {
		return fmt.Errorf("heartbeat lease: %w", err)
	}
	return nil
}

func (s *Store) loadDependencies(ctx context.Context, opID model.OpID) ([]model.Dependency, error) {
	return loadDependenciesTx(ctx, s.db, opID)
}
