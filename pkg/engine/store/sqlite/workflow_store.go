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

func (s *Store) GetWorkflowStats(ctx context.Context, workflowID model.WorkflowID) (*storecontract.WorkflowStats, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(1),
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0)
		FROM ops WHERE workflow_id = ?`, model.OpStatusPending, model.OpStatusReady, model.OpStatusRunning, model.OpStatusSucceeded, model.OpStatusFailed, model.OpStatusCanceled, workflowID)
	stats := &storecontract.WorkflowStats{WorkflowID: workflowID}
	if err := row.Scan(&stats.Total, &stats.Pending, &stats.Ready, &stats.Running, &stats.Succeeded, &stats.Failed, &stats.Canceled); err != nil {
		return nil, fmt.Errorf("query workflow stats: %w", err)
	}
	return stats, nil
}
