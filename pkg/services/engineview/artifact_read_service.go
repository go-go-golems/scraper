package engineview

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
)

type ArtifactSummary struct {
	ID          model.ArtifactID  `json:"id"`
	OpID        model.OpID        `json:"opID"`
	WorkflowID  model.WorkflowID  `json:"workflowID"`
	Name        string            `json:"name"`
	Kind        string            `json:"kind"`
	ContentType string            `json:"contentType"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Size        int               `json:"size"`
	CreatedAt   time.Time         `json:"createdAt"`
	Previewable bool              `json:"previewable"`
	PreviewKind string            `json:"previewKind,omitempty"`
}

type ArtifactDetail struct {
	ArtifactSummary
	Body []byte `json:"-"`
}

type WorkflowArtifactsResult struct {
	WorkflowID model.WorkflowID  `json:"workflowID"`
	Artifacts  []ArtifactSummary `json:"artifacts"`
	Total      int               `json:"total"`
}

type ListWorkflowArtifactsOptions struct {
	OpID        model.OpID
	Kind        string
	ContentType string
	Search      string
	Limit       int
	Offset      int
}

type ResultSummary struct {
	OpID          model.OpID     `json:"opID"`
	Kind          string         `json:"kind"`
	Status        string         `json:"status"`
	RecordCount   int            `json:"recordCount"`
	ArtifactCount int            `json:"artifactCount"`
	DataSize      int            `json:"dataSize"`
	Error         *model.OpError `json:"error,omitempty"`
	CompletedAt   time.Time      `json:"completedAt"`
}

type WorkflowResultsResult struct {
	WorkflowID model.WorkflowID `json:"workflowID"`
	Total      int              `json:"total"`
	Results    []ResultSummary  `json:"results"`
}

type ListWorkflowResultsOptions struct {
	OpID   model.OpID
	Kind   string
	Status string
	Search string
	Limit  int
	Offset int
}

func (s *Service) ListArtifacts(ctx context.Context, workflowID model.WorkflowID, opID model.OpID) ([]ArtifactSummary, error) {
	db, err := s.openReadDB()
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, nil
	}
	defer func() { _ = db.Close() }()

	rows, err := db.QueryContext(ctx,
		`SELECT id, workflow_id, op_id, name, kind, content_type, metadata_json, length(body), created_at
		 FROM artifacts
		 WHERE workflow_id = ? AND op_id = ?
		 ORDER BY created_at, id`,
		workflowID, opID,
	)
	if err != nil {
		return nil, fmt.Errorf("list artifacts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ret []ArtifactSummary
	for rows.Next() {
		var a ArtifactSummary
		var metadataText string
		var createdAt string
		if err := rows.Scan(&a.ID, &a.WorkflowID, &a.OpID, &a.Name, &a.Kind, &a.ContentType, &metadataText, &a.Size, &createdAt); err != nil {
			return nil, fmt.Errorf("scan artifact: %w", err)
		}
		_ = json.Unmarshal([]byte(metadataText), &a.Metadata)
		a.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		enrichArtifactSummary(&a)
		ret = append(ret, a)
	}
	return ret, rows.Err()
}

func (s *Service) ListWorkflowArtifacts(ctx context.Context, workflowID model.WorkflowID, opts ListWorkflowArtifactsOptions) (*WorkflowArtifactsResult, error) {
	db, err := s.openReadDB()
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, nil
	}
	defer func() { _ = db.Close() }()

	exists, err := workflowExists(ctx, db, workflowID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	where := []string{"workflow_id = ?"}
	args := []any{workflowID}
	if opts.OpID != "" {
		where = append(where, "op_id = ?")
		args = append(args, opts.OpID)
	}
	if opts.Kind != "" {
		where = append(where, "kind = ?")
		args = append(args, opts.Kind)
	}
	if opts.ContentType != "" {
		where = append(where, "content_type = ?")
		args = append(args, opts.ContentType)
	}
	if strings.TrimSpace(opts.Search) != "" {
		where = append(where, "(id LIKE ? OR op_id LIKE ? OR name LIKE ? OR kind LIKE ? OR content_type LIKE ?)")
		needle := "%" + strings.TrimSpace(opts.Search) + "%"
		args = append(args, needle, needle, needle, needle, needle)
	}
	whereClause := strings.Join(where, " AND ")

	limit := opts.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	var total int
	// #nosec G202 -- whereClause is assembled only from fixed predicates above; user values are bound parameters.
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM artifacts WHERE `+whereClause, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count workflow artifacts: %w", err)
	}

	queryArgs := append(append([]any{}, args...), limit, offset)
	// #nosec G202 -- whereClause is assembled only from fixed predicates above; user values are bound parameters.
	rows, err := db.QueryContext(ctx,
		`SELECT id, workflow_id, op_id, name, kind, content_type, metadata_json, length(body), created_at
		 FROM artifacts
		 WHERE `+whereClause+`
		 ORDER BY created_at, op_id, id
		 LIMIT ? OFFSET ?`,
		queryArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("list workflow artifacts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	ret := &WorkflowArtifactsResult{
		WorkflowID: workflowID,
		Artifacts:  []ArtifactSummary{},
		Total:      total,
	}
	for rows.Next() {
		var a ArtifactSummary
		var metadataText string
		var createdAt string
		if err := rows.Scan(&a.ID, &a.WorkflowID, &a.OpID, &a.Name, &a.Kind, &a.ContentType, &metadataText, &a.Size, &createdAt); err != nil {
			return nil, fmt.Errorf("scan workflow artifact: %w", err)
		}
		_ = json.Unmarshal([]byte(metadataText), &a.Metadata)
		a.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		enrichArtifactSummary(&a)
		ret.Artifacts = append(ret.Artifacts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *Service) GetOpResult(ctx context.Context, workflowID model.WorkflowID, opID model.OpID) (*model.OpResult, bool, error) {
	db, err := s.openReadDB()
	if err != nil {
		return nil, false, err
	}
	if db == nil {
		return nil, false, nil
	}
	defer func() { _ = db.Close() }()

	exists, err := workflowOpExists(ctx, db, workflowID, opID)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}

	store, err := s.openStore(ctx)
	if err != nil {
		return nil, false, err
	}
	if store == nil {
		return nil, false, nil
	}
	defer func() { _ = store.Close() }()

	result, err := store.GetResult(ctx, workflowID, opID)
	if err != nil {
		return nil, false, err
	}
	return result, true, nil
}

func (s *Service) ListWorkflowResults(ctx context.Context, workflowID model.WorkflowID, opts ListWorkflowResultsOptions) (*WorkflowResultsResult, error) {
	db, err := s.openReadDB()
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, nil
	}
	defer func() { _ = db.Close() }()

	exists, err := workflowExists(ctx, db, workflowID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	// Build the WHERE clause from the read DB (ops table) — results live in the
	// durable store (openStore), but we need op metadata (kind, status) from ops.
	where := []string{"results.workflow_id = ?"}
	args := []any{workflowID}
	if opts.OpID != "" {
		where = append(where, "results.op_id = ?")
		args = append(args, opts.OpID)
	}
	if opts.Kind != "" {
		where = append(where, "ops.kind = ?")
		args = append(args, opts.Kind)
	}
	if opts.Status != "" {
		where = append(where, "ops.status = ?")
		args = append(args, opts.Status)
	}
	if strings.TrimSpace(opts.Search) != "" {
		where = append(where, "results.op_id LIKE ?")
		args = append(args, "%"+strings.TrimSpace(opts.Search)+"%")
	}
	whereClause := strings.Join(where, " AND ")

	limit := opts.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	var total int
	// #nosec G202 -- whereClause is assembled only from fixed predicates above; user values are bound parameters.
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM results JOIN ops ON results.op_id = ops.id AND results.workflow_id = ops.workflow_id WHERE `+whereClause,
		args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count workflow results: %w", err)
	}

	queryArgs := append(append([]any{}, args...), limit, offset)
	// #nosec G202 -- whereClause is assembled only from fixed predicates above; user values are bound parameters.
	rows, err := db.QueryContext(ctx,
		`SELECT results.op_id, ops.kind, ops.status,
		        length(results.data_json),
		        results.records_json,
		        results.error_json,
		        results.completed_at,
		        (
		          SELECT COUNT(1) FROM artifacts
		          WHERE artifacts.op_id = results.op_id
		            AND artifacts.workflow_id = results.workflow_id
		        ) AS artifact_count
		 FROM results
		 JOIN ops ON results.op_id = ops.id AND results.workflow_id = ops.workflow_id
		 WHERE `+whereClause+`
		 ORDER BY results.completed_at DESC, results.op_id
		 LIMIT ? OFFSET ?`,
		queryArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("list workflow results: %w", err)
	}
	defer func() { _ = rows.Close() }()

	ret := &WorkflowResultsResult{WorkflowID: workflowID, Total: total, Results: []ResultSummary{}}
	for rows.Next() {
		var rs ResultSummary
		var recordsJSON string
		var errorJSON sql.NullString
		var completedAt string
		if err := rows.Scan(&rs.OpID, &rs.Kind, &rs.Status, &rs.DataSize, &recordsJSON, &errorJSON, &completedAt, &rs.ArtifactCount); err != nil {
			return nil, fmt.Errorf("scan result summary: %w", err)
		}
		rs.CompletedAt, _ = time.Parse(time.RFC3339Nano, completedAt)

		var records []model.RecordWrite
		if err := json.Unmarshal([]byte(recordsJSON), &records); err != nil {
			records = nil
		}
		rs.RecordCount = len(records)

		if errorJSON.Valid {
			var err2 model.OpError
			if err := json.Unmarshal([]byte(errorJSON.String), &err2); err == nil {
				rs.Error = &err2
			}
		}

		ret.Results = append(ret.Results, rs)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *Service) GetArtifact(ctx context.Context, artifactID model.ArtifactID) (*ArtifactDetail, error) {
	db, err := s.openReadDB()
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, nil
	}
	defer func() { _ = db.Close() }()

	row := db.QueryRowContext(ctx,
		`SELECT id, workflow_id, op_id, name, kind, content_type, metadata_json, body, length(body), created_at
		 FROM artifacts
		 WHERE id = ?`,
		artifactID,
	)
	var a ArtifactDetail
	var metadataText string
	var createdAt string
	if err := row.Scan(&a.ID, &a.WorkflowID, &a.OpID, &a.Name, &a.Kind, &a.ContentType, &metadataText, &a.Body, &a.Size, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get artifact: %w", err)
	}
	_ = json.Unmarshal([]byte(metadataText), &a.Metadata)
	a.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	enrichArtifactSummary(&a.ArtifactSummary)
	return &a, nil
}
