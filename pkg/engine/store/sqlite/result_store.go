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
