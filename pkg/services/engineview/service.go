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
