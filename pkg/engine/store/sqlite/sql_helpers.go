package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
)

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
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
