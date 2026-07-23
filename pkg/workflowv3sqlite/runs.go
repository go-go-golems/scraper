package workflowv3sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

type RunRecord struct {
	RunID      workflowv3.RunID
	Name       string
	PlanDigest string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (s *Store) ListRuns(ctx context.Context, status string, limit int) ([]RunRecord, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("workflow v3 store is required")
	}
	status = strings.TrimSpace(status)
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		return nil, fmt.Errorf("run list limit must not exceed 1000")
	}
	query := `
SELECT run_id, name, plan_digest, status, created_at, updated_at
FROM v3_runs`
	args := []any{}
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC, run_id ASC LIMIT ?"
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list workflow v3 runs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	ret := make([]RunRecord, 0)
	for rows.Next() {
		var row RunRecord
		var runID, createdAt, updatedAt string
		if err := rows.Scan(&runID, &row.Name, &row.PlanDigest, &row.Status, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan workflow v3 run: %w", err)
		}
		row.RunID = workflowv3.RunID(runID)
		row.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse run %s created time: %w", runID, err)
		}
		row.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse run %s updated time: %w", runID, err)
		}
		ret = append(ret, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workflow v3 runs: %w", err)
	}
	return ret, nil
}
