package engineview

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
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
	OpTotal  int               `json:"opTotal"`
	OpDone   int               `json:"opDone"`
}

type WorkflowListResult struct {
	Workflows []WorkflowListItem `json:"workflows"`
	Total     int                `json:"total"`
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
	defer func() { _ = rows.Close() }()

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
	defer func() { _ = rows.Close() }()

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
