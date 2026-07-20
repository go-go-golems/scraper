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

func (s *Store) CreateWorkflow(ctx context.Context, params storecontract.CreateWorkflowParams) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create workflow: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	workflow := params.Workflow
	if workflow.CreatedAt.IsZero() {
		workflow.CreatedAt = time.Now().UTC()
	}
	if workflow.UpdatedAt.IsZero() {
		workflow.UpdatedAt = workflow.CreatedAt
	}
	workflow.CreatedAt = workflow.CreatedAt.UTC()
	workflow.UpdatedAt = workflow.UpdatedAt.UTC()
	if _, err := tx.ExecContext(ctx, `INSERT INTO workflows(id, site, name, status, input_json, metadata_json, created_at, created_at_us, updated_at, updated_at_us)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, workflow.ID, workflow.Site, workflow.Name, workflow.Status, jsonText(workflow.Input, `null`), mustJSON(workflow.Metadata), workflow.CreatedAt.Format(time.RFC3339Nano), epochMicros(workflow.CreatedAt), workflow.UpdatedAt.Format(time.RFC3339Nano), epochMicros(workflow.UpdatedAt)); err != nil {
		return fmt.Errorf("insert workflow: %w", err)
	}
	if err := insertOps(ctx, tx, params.Initial); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create workflow: %w", err)
	}
	return nil
}

func (s *Store) GetWorkflow(ctx context.Context, id model.WorkflowID) (*model.WorkflowRun, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, site, name, status, input_json, metadata_json, created_at_us, updated_at_us FROM workflows WHERE id = ?`, id)
	var workflow model.WorkflowRun
	var inputText, metadataText string
	var createdAt, updatedAt int64
	if err := row.Scan(&workflow.ID, &workflow.Site, &workflow.Name, &workflow.Status, &inputText, &metadataText, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query workflow %s: %w", id, err)
	}
	workflow.Input = json.RawMessage(inputText)
	if err := unmarshalJSON(metadataText, &workflow.Metadata); err != nil {
		return nil, fmt.Errorf("decode workflow metadata: %w", err)
	}
	workflow.CreatedAt = timeFromEpochMicros(createdAt)
	workflow.UpdatedAt = timeFromEpochMicros(updatedAt)
	return &workflow, nil
}

func (s *Store) UpdateWorkflowStatus(ctx context.Context, id model.WorkflowID, status model.WorkflowStatus) error {
	now := time.Now().UTC()
	if _, err := s.db.ExecContext(ctx, `UPDATE workflows SET status = ?, updated_at = ?, updated_at_us = ? WHERE id = ?`, status, now.Format(time.RFC3339Nano), epochMicros(now), id); err != nil {
		return fmt.Errorf("update workflow status: %w", err)
	}
	return nil
}

func (s *Store) GetWorkflowSnapshot(ctx context.Context, id model.WorkflowID) (*storecontract.WorkflowSnapshot, error) {
	workflow, err := s.GetWorkflow(ctx, id)
	if err != nil || workflow == nil {
		return nil, err
	}
	stats, err := s.GetWorkflowStats(ctx, id)
	if err != nil {
		return nil, err
	}
	return &storecontract.WorkflowSnapshot{Workflow: workflow, Stats: stats}, nil
}

// ListWorkflowSnapshots returns an authoritative store-derived inspection page.
// Pass the last observed UpdatedAt to resume polling without relying on
// process-local events. The tie-breaking ID order makes page output stable.
func (s *Store) ListWorkflowSnapshots(ctx context.Context, updatedAfter time.Time, limit int) ([]storecontract.WorkflowSnapshot, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM workflows
		WHERE updated_at_us > ?
		ORDER BY updated_at_us ASC, id ASC
		LIMIT ?`, epochMicros(updatedAfter), limit)
	if err != nil {
		return nil, fmt.Errorf("list workflow snapshot ids: %w", err)
	}
	defer func() { _ = rows.Close() }()
	ids := make([]model.WorkflowID, 0, limit)
	for rows.Next() {
		var id model.WorkflowID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan workflow snapshot id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close workflow snapshot ids: %w", err)
	}
	result := make([]storecontract.WorkflowSnapshot, 0, len(ids))
	for _, id := range ids {
		snapshot, err := s.GetWorkflowSnapshot(ctx, id)
		if err != nil {
			return nil, err
		}
		if snapshot != nil {
			result = append(result, *snapshot)
		}
	}
	return result, nil
}

func (s *Store) GetWorkflowStats(ctx context.Context, workflowID model.WorkflowID) (*storecontract.WorkflowStats, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(1),
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0)
		FROM ops WHERE workflow_id = ?`, model.OpStatusPending, model.OpStatusReady, model.OpStatusRunning, model.OpStatusSucceeded, model.OpStatusFailed, model.OpStatusBlocked, model.OpStatusCanceled, workflowID)
	stats := &storecontract.WorkflowStats{WorkflowID: workflowID}
	if err := row.Scan(&stats.Total, &stats.Pending, &stats.Ready, &stats.Running, &stats.Succeeded, &stats.Failed, &stats.Blocked, &stats.Canceled); err != nil {
		return nil, fmt.Errorf("query workflow stats: %w", err)
	}
	return stats, nil
}
