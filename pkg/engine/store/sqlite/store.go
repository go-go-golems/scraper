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

func (s *Store) CreateWorkflow(ctx context.Context, params storecontract.CreateWorkflowParams) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create workflow: %w", err)
	}

	workflow := params.Workflow
	if workflow.CreatedAt.IsZero() {
		workflow.CreatedAt = time.Now().UTC()
	}
	if workflow.UpdatedAt.IsZero() {
		workflow.UpdatedAt = workflow.CreatedAt
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO workflows(id, site, name, status, input_json, metadata_json, created_at, updated_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
		workflow.ID,
		workflow.Site,
		workflow.Name,
		workflow.Status,
		jsonText(workflow.Input, `null`),
		mustJSON(workflow.Metadata),
		workflow.CreatedAt.Format(time.RFC3339Nano),
		workflow.UpdatedAt.Format(time.RFC3339Nano),
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("insert workflow: %w", err)
	}

	if err := insertOps(ctx, tx, params.Initial); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create workflow: %w", err)
	}

	return nil
}

func (s *Store) GetWorkflow(ctx context.Context, id model.WorkflowID) (*model.WorkflowRun, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, site, name, status, input_json, metadata_json, created_at, updated_at
		 FROM workflows WHERE id = ?`,
		id,
	)

	var workflow model.WorkflowRun
	var inputText string
	var metadataText string
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&workflow.ID,
		&workflow.Site,
		&workflow.Name,
		&workflow.Status,
		&inputText,
		&metadataText,
		&createdAt,
		&updatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query workflow %s: %w", id, err)
	}

	workflow.Input = json.RawMessage(inputText)
	if err := unmarshalJSON(metadataText, &workflow.Metadata); err != nil {
		return nil, fmt.Errorf("decode workflow metadata: %w", err)
	}
	workflow.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	workflow.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)

	return &workflow, nil
}

func (s *Store) UpdateWorkflowStatus(ctx context.Context, id model.WorkflowID, status model.WorkflowStatus) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE workflows SET status = ?, updated_at = ? WHERE id = ?`,
		status,
		time.Now().UTC().Format(time.RFC3339Nano),
		id,
	)
	if err != nil {
		return fmt.Errorf("update workflow status: %w", err)
	}
	return nil
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
		`SELECT id, workflow_id, parent_id, site, kind, queue_key, dedup_key, input_json, retry_json, retry_state_json, metadata_json
		 FROM ops WHERE id = ?`,
		id,
	)

	var op model.OpSpec
	var parentID sql.NullString
	var inputText string
	var retryText string
	var retryStateText string
	var metadataText string
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

func (s *Store) GetWorkflowStats(ctx context.Context, workflowID model.WorkflowID) (*storecontract.WorkflowStats, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT
		 COUNT(1),
		 COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		 COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		 COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		 COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		 COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		 COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0)
		 FROM ops
		 WHERE workflow_id = ?`,
		model.OpStatusPending,
		model.OpStatusReady,
		model.OpStatusRunning,
		model.OpStatusSucceeded,
		model.OpStatusFailed,
		model.OpStatusCanceled,
		workflowID,
	)

	stats := &storecontract.WorkflowStats{WorkflowID: workflowID}
	if err := row.Scan(
		&stats.Total,
		&stats.Pending,
		&stats.Ready,
		&stats.Running,
		&stats.Succeeded,
		&stats.Failed,
		&stats.Canceled,
	); err != nil {
		return nil, fmt.Errorf("query workflow stats: %w", err)
	}

	return stats, nil
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
		`SELECT id, workflow_id, parent_id, site, kind, queue_key, dedup_key, input_json, retry_json, retry_state_json, metadata_json
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

func (s *Store) CompleteOp(
	ctx context.Context,
	opID model.OpID,
	completion storecontract.Completion,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin complete op: %w", err)
	}

	result := completion.Result
	if result.CompletedAt.IsZero() {
		result.CompletedAt = time.Now().UTC()
	}
	workflowID, site, err := lookupOpContext(ctx, tx, opID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	normalizeEmittedOps(result.Emitted, workflowID, site, opID)

	if _, err := tx.ExecContext(
		ctx,
		`INSERT OR REPLACE INTO results(op_id, workflow_id, data_json, records_json, emitted_json, emitted_ids_json, error_json, completed_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
		opID,
		workflowID,
		jsonText(result.Data, `null`),
		mustJSON(result.Records),
		mustJSON(result.Emitted),
		mustJSON(result.EmittedIDs),
		nullableJSON(result.Error),
		result.CompletedAt.Format(time.RFC3339Nano),
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("insert result: %w", err)
	}

	for _, artifact := range result.Artifacts {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT OR REPLACE INTO artifacts(id, workflow_id, op_id, name, kind, content_type, metadata_json, body, created_at)
			 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			artifact.ID,
			workflowID,
			opID,
			artifact.Name,
			artifact.Kind,
			artifact.ContentType,
			mustJSON(artifact.Metadata),
			artifact.Body,
			result.CompletedAt.Format(time.RFC3339Nano),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert artifact %s: %w", artifact.ID, err)
		}
	}

	if err := insertOps(ctx, tx, result.Emitted); err != nil {
		_ = tx.Rollback()
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM leases WHERE op_id = ? AND token = ?`,
		opID,
		completion.Lease.Token,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete lease: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE ops SET status = ?, updated_at = ? WHERE id = ?`,
		model.OpStatusSucceeded,
		result.CompletedAt.Format(time.RFC3339Nano),
		opID,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("mark op succeeded: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit complete op: %w", err)
	}
	return nil
}

func (s *Store) FailOp(ctx context.Context, opID model.OpID, failure storecontract.Failure) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin fail op: %w", err)
	}

	workflowID, _, err := lookupOpContext(ctx, tx, opID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	failedAt := failure.Error.OccurredAt
	if failedAt.IsZero() {
		failedAt = time.Now().UTC()
	}

	nextAttempt := nullableTime(failure.RetryState.NextAttemptAt)
	status := model.OpStatusFailed
	if failure.RetryState.NextAttemptAt != nil {
		status = model.OpStatusReady
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT OR REPLACE INTO results(op_id, workflow_id, data_json, records_json, emitted_json, emitted_ids_json, error_json, completed_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
		opID,
		workflowID,
		`null`,
		`[]`,
		`[]`,
		`[]`,
		mustJSON(failure.Error),
		failedAt.UTC().Format(time.RFC3339Nano),
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("insert failed result: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE ops
		 SET status = ?, retry_state_json = ?, next_attempt_at = ?, updated_at = ?
		 WHERE id = ?`,
		status,
		mustJSON(failure.RetryState),
		nextAttempt,
		failedAt.UTC().Format(time.RFC3339Nano),
		opID,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("update failed op: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM leases WHERE op_id = ? AND token = ?`,
		opID,
		failure.Lease.Token,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete failed lease: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit fail op: %w", err)
	}

	return nil
}

func (s *Store) GetResult(
	ctx context.Context,
	workflowID model.WorkflowID,
	opID model.OpID,
) (*model.OpResult, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT data_json, records_json, emitted_json, emitted_ids_json, error_json, completed_at
		 FROM results WHERE workflow_id = ? AND op_id = ?`,
		workflowID,
		opID,
	)

	var result model.OpResult
	var dataText string
	var recordsText string
	var emittedText string
	var emittedIDsText string
	var errorText sql.NullString
	var completedAt string
	if err := row.Scan(
		&dataText,
		&recordsText,
		&emittedText,
		&emittedIDsText,
		&errorText,
		&completedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query result: %w", err)
	}

	result.OpID = opID
	result.Data = json.RawMessage(dataText)
	if err := unmarshalJSON(recordsText, &result.Records); err != nil {
		return nil, fmt.Errorf("decode result records: %w", err)
	}
	if err := unmarshalJSON(emittedText, &result.Emitted); err != nil {
		return nil, fmt.Errorf("decode emitted ops: %w", err)
	}
	if err := unmarshalJSON(emittedIDsText, &result.EmittedIDs); err != nil {
		return nil, fmt.Errorf("decode emitted ids: %w", err)
	}
	if errorText.Valid {
		var opErr model.OpError
		if err := unmarshalJSON(errorText.String, &opErr); err != nil {
			return nil, fmt.Errorf("decode result error: %w", err)
		}
		result.Error = &opErr
	}
	result.CompletedAt, _ = time.Parse(time.RFC3339Nano, completedAt)
	artifacts, err := s.loadArtifacts(ctx, workflowID, opID)
	if err != nil {
		return nil, err
	}
	result.Artifacts = artifacts

	return &result, nil
}

func (s *Store) loadDependencies(ctx context.Context, opID model.OpID) ([]model.Dependency, error) {
	return loadDependenciesTx(ctx, s.db, opID)
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type queueLimiterState struct {
	Tokens       float64
	LastRefillAt time.Time
}

func loadDependenciesTx(ctx context.Context, db queryer, opID model.OpID) ([]model.Dependency, error) {
	rows, err := db.QueryContext(
		ctx,
		`SELECT depends_on_op_id, required FROM op_dependencies WHERE op_id = ? ORDER BY depends_on_op_id`,
		opID,
	)
	if err != nil {
		return nil, fmt.Errorf("query dependencies for %s: %w", opID, err)
	}
	defer rows.Close()

	ret := []model.Dependency{}
	for rows.Next() {
		var dep model.Dependency
		var required int
		if err := rows.Scan(&dep.OpID, &required); err != nil {
			return nil, fmt.Errorf("scan dependency: %w", err)
		}
		dep.Required = required != 0
		ret = append(ret, dep)
	}

	return ret, rows.Err()
}

func countActiveLeasesForQueue(
	ctx context.Context,
	tx *sql.Tx,
	site model.SiteName,
	queue model.QueueKey,
	now time.Time,
) (int, error) {
	row := tx.QueryRowContext(
		ctx,
		`SELECT COUNT(1)
		 FROM leases l
		 JOIN ops active ON active.id = l.op_id
		 WHERE l.expires_at > ?
		   AND active.site = ?
		   AND active.queue_key = ?`,
		now.UTC().Format(time.RFC3339Nano),
		site,
		queue,
	)

	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count active leases for %s/%s: %w", site, queue, err)
	}
	return count, nil
}

func loadQueueLimiterState(
	ctx context.Context,
	tx *sql.Tx,
	site model.SiteName,
	queue model.QueueKey,
) (queueLimiterState, error) {
	row := tx.QueryRowContext(
		ctx,
		`SELECT tokens, last_refill_at
		 FROM queue_limit_state
		 WHERE site = ? AND queue_key = ?`,
		site,
		queue,
	)

	var tokens float64
	var lastRefillAt string
	if err := row.Scan(&tokens, &lastRefillAt); err != nil {
		if err == sql.ErrNoRows {
			return queueLimiterState{}, nil
		}
		return queueLimiterState{}, fmt.Errorf("load queue limiter state for %s/%s: %w", site, queue, err)
	}

	parsed, err := time.Parse(time.RFC3339Nano, lastRefillAt)
	if err != nil {
		return queueLimiterState{}, fmt.Errorf("parse last_refill_at for %s/%s: %w", site, queue, err)
	}
	return queueLimiterState{
		Tokens:       tokens,
		LastRefillAt: parsed,
	}, nil
}

func refillQueueLimiterState(
	state queueLimiterState,
	policy model.RateLimitPolicy,
	now time.Time,
) queueLimiterState {
	if policy.Burst <= 0 || policy.RatePerSecond <= 0 {
		return state
	}
	if state.LastRefillAt.IsZero() {
		return queueLimiterState{
			Tokens:       float64(policy.Burst),
			LastRefillAt: now.UTC(),
		}
	}

	elapsed := now.UTC().Sub(state.LastRefillAt).Seconds()
	if elapsed < 0 {
		elapsed = 0
	}
	tokens := state.Tokens + elapsed*policy.RatePerSecond
	burst := float64(policy.Burst)
	if tokens > burst {
		tokens = burst
	}
	return queueLimiterState{
		Tokens:       tokens,
		LastRefillAt: now.UTC(),
	}
}

func upsertQueueLimiterState(
	ctx context.Context,
	tx *sql.Tx,
	site model.SiteName,
	queue model.QueueKey,
	state queueLimiterState,
) error {
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO queue_limit_state(site, queue_key, tokens, last_refill_at)
		 VALUES(?, ?, ?, ?)
		 ON CONFLICT(site, queue_key) DO UPDATE SET
		   tokens = excluded.tokens,
		   last_refill_at = excluded.last_refill_at`,
		site,
		queue,
		state.Tokens,
		state.LastRefillAt.UTC().Format(time.RFC3339Nano),
	); err != nil {
		return fmt.Errorf("upsert queue limiter state for %s/%s: %w", site, queue, err)
	}
	return nil
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func execRowsAffected(ctx context.Context, db execer, query string, args ...any) (int, error) {
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(rowsAffected), nil
}

func insertOps(ctx context.Context, db execer, ops []model.OpSpec) error {
	for _, op := range ops {
		now := time.Now().UTC()
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO ops(
				id, workflow_id, parent_id, site, kind, queue_key, dedup_key,
				input_json, retry_json, metadata_json, status, retry_state_json,
				next_attempt_at, created_at, updated_at
			) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			op.ID,
			op.WorkflowID,
			nullableParentID(op.ParentID),
			op.Site,
			op.Kind,
			op.Queue,
			op.DedupKey,
			jsonText(op.Input, `null`),
			mustJSON(op.Retry),
			mustJSON(op.Metadata),
			initialStatus(op),
			mustJSON(op.RetryState),
			nil,
			now.Format(time.RFC3339Nano),
			now.Format(time.RFC3339Nano),
		); err != nil {
			return fmt.Errorf("insert op %s: %w", op.ID, err)
		}

		for _, dep := range op.DependsOn {
			if _, err := db.ExecContext(
				ctx,
				`INSERT INTO op_dependencies(workflow_id, op_id, depends_on_op_id, required)
				 VALUES(?, ?, ?, ?)`,
				op.WorkflowID,
				op.ID,
				dep.OpID,
				boolToInt(dep.Required),
			); err != nil {
				return fmt.Errorf("insert dependency %s -> %s: %w", dep.OpID, op.ID, err)
			}
		}
	}

	return nil
}

func lookupOpContext(ctx context.Context, db *sql.Tx, opID model.OpID) (model.WorkflowID, model.SiteName, error) {
	row := db.QueryRowContext(
		ctx,
		`SELECT workflow_id, site FROM ops WHERE id = ?`,
		opID,
	)

	var workflowID model.WorkflowID
	var site model.SiteName
	if err := row.Scan(&workflowID, &site); err != nil {
		return "", "", fmt.Errorf("query op context for %s: %w", opID, err)
	}

	return workflowID, site, nil
}

func normalizeEmittedOps(ops []model.OpSpec, workflowID model.WorkflowID, site model.SiteName, parentID model.OpID) {
	for i := range ops {
		if ops[i].WorkflowID == "" {
			ops[i].WorkflowID = workflowID
		}
		if ops[i].Site == "" {
			ops[i].Site = site
		}
		if ops[i].ParentID == nil {
			parent := parentID
			ops[i].ParentID = &parent
		}
	}
}

func initialStatus(op model.OpSpec) model.OpStatus {
	if len(op.DependsOn) == 0 {
		return model.OpStatusReady
	}
	return model.OpStatusPending
}

func nullableParentID(parentID *model.OpID) any {
	if parentID == nil {
		return nil
	}
	return string(*parentID)
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func jsonText(raw json.RawMessage, fallback string) string {
	if len(raw) == 0 {
		return fallback
	}
	return string(raw)
}

func mustJSON(v any) string {
	if v == nil {
		return "null"
	}
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func nullableJSON(v any) any {
	if v == nil {
		return nil
	}
	return mustJSON(v)
}

func unmarshalJSON(s string, target any) error {
	if s == "" || s == "null" {
		return nil
	}
	return json.Unmarshal([]byte(s), target)
}

func (s *Store) loadArtifacts(
	ctx context.Context,
	workflowID model.WorkflowID,
	opID model.OpID,
) ([]model.ArtifactWrite, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, name, kind, content_type, metadata_json, body
		 FROM artifacts
		 WHERE workflow_id = ? AND op_id = ?
		 ORDER BY id`,
		workflowID,
		opID,
	)
	if err != nil {
		return nil, fmt.Errorf("query artifacts: %w", err)
	}
	defer rows.Close()

	ret := []model.ArtifactWrite{}
	for rows.Next() {
		var artifact model.ArtifactWrite
		var metadataText string
		if err := rows.Scan(
			&artifact.ID,
			&artifact.Name,
			&artifact.Kind,
			&artifact.ContentType,
			&metadataText,
			&artifact.Body,
		); err != nil {
			return nil, fmt.Errorf("scan artifact: %w", err)
		}
		if err := unmarshalJSON(metadataText, &artifact.Metadata); err != nil {
			return nil, fmt.Errorf("decode artifact metadata: %w", err)
		}
		ret = append(ret, artifact)
	}

	return ret, rows.Err()
}
