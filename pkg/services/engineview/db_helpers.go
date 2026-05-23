package engineview

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
)

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

func workflowExists(ctx context.Context, db *sql.DB, workflowID model.WorkflowID) (bool, error) {
	row := db.QueryRowContext(ctx, `SELECT 1 FROM workflows WHERE id = ?`, workflowID)
	var found int
	if err := row.Scan(&found); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("query workflow existence: %w", err)
	}
	return true, nil
}

func workflowOpExists(ctx context.Context, db *sql.DB, workflowID model.WorkflowID, opID model.OpID) (bool, error) {
	row := db.QueryRowContext(ctx, `SELECT 1 FROM ops WHERE workflow_id = ? AND id = ?`, workflowID, opID)
	var found int
	if err := row.Scan(&found); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("query op existence: %w", err)
	}
	return true, nil
}

func enrichArtifactSummary(a *ArtifactSummary) {
	a.Previewable, a.PreviewKind = classifyArtifactPreview(a.ContentType)
}

func classifyArtifactPreview(contentType string) (bool, string) {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.Contains(ct, "application/json"):
		return true, "json"
	case strings.Contains(ct, "text/html"):
		return true, "html"
	case strings.HasPrefix(ct, "text/"):
		return true, "text"
	case strings.Contains(ct, "javascript"):
		return true, "text"
	case strings.Contains(ct, "xml"):
		return true, "text"
	default:
		return false, "binary"
	}
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
	defer func() { _ = rows.Close() }()

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
