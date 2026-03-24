package engineview

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

type WorkflowSummary struct {
	Workflow *model.WorkflowRun           `json:"workflow"`
	Stats    *storecontract.WorkflowStats `json:"stats"`
}

type WorkflowOp struct {
	Op            model.OpSpec   `json:"op"`
	Status        model.OpStatus `json:"status"`
	NextAttemptAt *time.Time     `json:"nextAttemptAt,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	Lease         *model.Lease   `json:"lease,omitempty"`
}

type ListWorkflowsOptions struct {
	Site   model.SiteName
	Status model.WorkflowStatus
	Limit  int
	Offset int
}

type WorkflowListItem struct {
	Workflow model.WorkflowRun `json:"workflow"`
	OpTotal int               `json:"opTotal"`
	OpDone  int               `json:"opDone"`
}

type WorkflowListResult struct {
	Workflows []WorkflowListItem `json:"workflows"`
	Total     int                `json:"total"`
}

type QueueStatus struct {
	Site        model.SiteName `json:"site"`
	Queue       model.QueueKey `json:"queue"`
	Pending     int            `json:"pending"`
	Ready       int            `json:"ready"`
	Running     int            `json:"running"`
	Succeeded   int            `json:"succeeded"`
	Failed      int            `json:"failed"`
	InFlight    int            `json:"inFlight"`
	MaxInFlight int            `json:"maxInFlight"`
	Tokens      *float64       `json:"tokens,omitempty"`
	Burst       *int           `json:"burst,omitempty"`
	RatePerSec  *float64       `json:"ratePerSecond,omitempty"`
}

type Service struct {
	engineDB string
}

func NewService(engineDB string) *Service {
	return &Service{engineDB: engineDB}
}

func (s *Service) EngineStatus(ctx context.Context) (*sqlitestore.EngineStatus, error) {
	return sqlitestore.Inspect(ctx, s.engineDB)
}

func (s *Service) Workflow(ctx context.Context, workflowID model.WorkflowID) (*WorkflowSummary, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, nil
	}
	defer func() { _ = store.Close() }()

	workflow, err := store.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if workflow == nil {
		return nil, nil
	}
	stats, err := store.GetWorkflowStats(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	return &WorkflowSummary{Workflow: workflow, Stats: stats}, nil
}

func (s *Service) WorkflowOps(ctx context.Context, workflowID model.WorkflowID) ([]WorkflowOp, error) {
	db, err := s.openReadDB()
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, nil
	}
	defer func() { _ = db.Close() }()

	rows, err := db.QueryContext(
		ctx,
		`SELECT id, workflow_id, parent_id, site, kind, queue_key, dedup_key, input_json, retry_json, retry_state_json, metadata_json, status, next_attempt_at, created_at, updated_at
		 FROM ops
		 WHERE workflow_id = ?
		 ORDER BY created_at, id`,
		workflowID,
	)
	if err != nil {
		return nil, fmt.Errorf("query workflow ops: %w", err)
	}
	defer rows.Close()

	ret := []WorkflowOp{}
	for rows.Next() {
		var op WorkflowOp
		var parentID sql.NullString
		var inputText string
		var retryText string
		var retryStateText string
		var metadataText string
		var nextAttemptText sql.NullString
		var createdAt string
		var updatedAt string
		if err := rows.Scan(
			&op.Op.ID,
			&op.Op.WorkflowID,
			&parentID,
			&op.Op.Site,
			&op.Op.Kind,
			&op.Op.Queue,
			&op.Op.DedupKey,
			&inputText,
			&retryText,
			&retryStateText,
			&metadataText,
			&op.Status,
			&nextAttemptText,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan workflow op: %w", err)
		}
		if parentID.Valid {
			parent := model.OpID(parentID.String)
			op.Op.ParentID = &parent
		}
		op.Op.Input = json.RawMessage(inputText)
		if err := json.Unmarshal([]byte(retryText), &op.Op.Retry); err != nil {
			return nil, fmt.Errorf("decode op retry policy: %w", err)
		}
		if err := json.Unmarshal([]byte(retryStateText), &op.Op.RetryState); err != nil {
			return nil, fmt.Errorf("decode op retry state: %w", err)
		}
		if err := json.Unmarshal([]byte(metadataText), &op.Op.Metadata); err != nil {
			return nil, fmt.Errorf("decode op metadata: %w", err)
		}
		if nextAttemptText.Valid && nextAttemptText.String != "" {
			nextAttemptAt, err := time.Parse(time.RFC3339Nano, nextAttemptText.String)
			if err != nil {
				return nil, fmt.Errorf("parse next attempt time: %w", err)
			}
			op.NextAttemptAt = &nextAttemptAt
		}
		op.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		op.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)

		op.Op.DependsOn, err = loadDependencies(ctx, db, op.Op.ID)
		if err != nil {
			return nil, err
		}
		op.Lease, err = loadLease(ctx, db, op.Op.ID)
		if err != nil {
			return nil, err
		}

		ret = append(ret, op)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *Service) ListWorkflows(ctx context.Context, opts ListWorkflowsOptions) (*WorkflowListResult, error) {
	db, err := s.openReadDB()
	if err != nil {
		return nil, err
	}
	if db == nil {
		return &WorkflowListResult{}, nil
	}
	defer func() { _ = db.Close() }()

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	where := "1=1"
	args := []any{}
	if opts.Site != "" {
		where += " AND w.site = ?"
		args = append(args, string(opts.Site))
	}
	if opts.Status != "" {
		where += " AND w.status = ?"
		args = append(args, string(opts.Status))
	}

	countQuery := "SELECT COUNT(1) FROM workflows w WHERE " + where
	var total int
	if err := db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count workflows: %w", err)
	}

	query := `SELECT w.id, w.site, w.name, w.status, w.input_json, w.metadata_json, w.created_at, w.updated_at,
		COALESCE((SELECT COUNT(1) FROM ops o WHERE o.workflow_id = w.id), 0),
		COALESCE((SELECT COUNT(1) FROM ops o WHERE o.workflow_id = w.id AND o.status IN ('succeeded','failed','canceled')), 0)
		FROM workflows w
		WHERE ` + where + `
		ORDER BY w.created_at DESC
		LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}
	defer rows.Close()

	result := &WorkflowListResult{Total: total, Workflows: []WorkflowListItem{}}
	for rows.Next() {
		var item WorkflowListItem
		var inputText, metadataText, createdAt, updatedAt string
		if err := rows.Scan(
			&item.Workflow.ID, &item.Workflow.Site, &item.Workflow.Name, &item.Workflow.Status,
			&inputText, &metadataText, &createdAt, &updatedAt,
			&item.OpTotal, &item.OpDone,
		); err != nil {
			return nil, fmt.Errorf("scan workflow: %w", err)
		}
		item.Workflow.Input = json.RawMessage(inputText)
		_ = json.Unmarshal([]byte(metadataText), &item.Workflow.Metadata)
		item.Workflow.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		item.Workflow.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		result.Workflows = append(result.Workflows, item)
	}
	return result, rows.Err()
}

func (s *Service) ListQueues(ctx context.Context) ([]QueueStatus, error) {
	db, err := s.openReadDB()
	if err != nil {
		return nil, err
	}
	if db == nil {
		return []QueueStatus{}, nil
	}
	defer func() { _ = db.Close() }()

	query := `SELECT o.site, o.queue_key, o.status, COUNT(1)
		FROM ops o
		GROUP BY o.site, o.queue_key, o.status
		ORDER BY o.site, o.queue_key, o.status`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list queue op counts: %w", err)
	}
	defer rows.Close()

	type queueKey struct {
		site  model.SiteName
		queue model.QueueKey
	}
	queueMap := map[queueKey]*QueueStatus{}
	var order []queueKey
	for rows.Next() {
		var site model.SiteName
		var queue model.QueueKey
		var status model.OpStatus
		var count int
		if err := rows.Scan(&site, &queue, &status, &count); err != nil {
			return nil, fmt.Errorf("scan queue status: %w", err)
		}
		key := queueKey{site, queue}
		qs, ok := queueMap[key]
		if !ok {
			qs = &QueueStatus{Site: site, Queue: queue, MaxInFlight: 1}
			queueMap[key] = qs
			order = append(order, key)
		}
		switch status {
		case model.OpStatusPending:
			qs.Pending = count
		case model.OpStatusReady:
			qs.Ready = count
		case model.OpStatusRunning:
			qs.Running = count
			qs.InFlight = count
		case model.OpStatusSucceeded:
			qs.Succeeded = count
		case model.OpStatusFailed:
			qs.Failed = count
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load token bucket state
	tokenRows, err := db.QueryContext(ctx, `SELECT site, queue_key, tokens FROM queue_limit_state`)
	if err == nil {
		defer tokenRows.Close()
		for tokenRows.Next() {
			var site model.SiteName
			var queue model.QueueKey
			var tokens float64
			if err := tokenRows.Scan(&site, &queue, &tokens); err == nil {
				key := queueKey{site, queue}
				if qs, ok := queueMap[key]; ok {
					qs.Tokens = &tokens
				}
			}
		}
	}

	result := make([]QueueStatus, 0, len(order))
	for _, key := range order {
		result = append(result, *queueMap[key])
	}
	return result, nil
}

func (s *Service) openStore(ctx context.Context) (*sqlitestore.Store, error) {
	if _, err := os.Stat(s.engineDB); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat engine db: %w", err)
	}
	return sqlitestore.Open(ctx, s.engineDB)
}

func (s *Service) openReadDB() (*sql.DB, error) {
	if _, err := os.Stat(s.engineDB); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat engine db: %w", err)
	}
	db, err := sql.Open("sqlite3", s.engineDB)
	if err != nil {
		return nil, fmt.Errorf("open engine db: %w", err)
	}
	return db, nil
}

func loadDependencies(ctx context.Context, db *sql.DB, opID model.OpID) ([]model.Dependency, error) {
	rows, err := db.QueryContext(
		ctx,
		`SELECT depends_on_op_id, required
		 FROM op_dependencies
		 WHERE op_id = ?
		 ORDER BY depends_on_op_id`,
		opID,
	)
	if err != nil {
		return nil, fmt.Errorf("query op dependencies: %w", err)
	}
	defer rows.Close()

	ret := []model.Dependency{}
	for rows.Next() {
		var dependency model.Dependency
		var required int
		if err := rows.Scan(&dependency.OpID, &required); err != nil {
			return nil, fmt.Errorf("scan op dependency: %w", err)
		}
		dependency.Required = required == 1
		ret = append(ret, dependency)
	}
	return ret, rows.Err()
}

func loadLease(ctx context.Context, db *sql.DB, opID model.OpID) (*model.Lease, error) {
	row := db.QueryRowContext(
		ctx,
		`SELECT worker_id, token, acquired_at, expires_at
		 FROM leases
		 WHERE op_id = ?`,
		opID,
	)
	var lease model.Lease
	var acquiredAt string
	var expiresAt string
	if err := row.Scan(&lease.WorkerID, &lease.Token, &acquiredAt, &expiresAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query op lease: %w", err)
	}
	lease.AcquiredAt, _ = time.Parse(time.RFC3339Nano, acquiredAt)
	lease.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expiresAt)
	return &lease, nil
}
