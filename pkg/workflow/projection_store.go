package workflow

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// ProjectionStore resolves domain/package projection databases. Projections are
// query-oriented read models owned by workflow packages, separate from engine
// scheduling state.
type ProjectionStore interface {
	Projection(ctx context.Context, name string) (Projection, error)
}

// Projection is a small database-like interface exposed to executors.
type Projection interface {
	Exec(ctx context.Context, query string, args ...any) (int64, error)
	Query(ctx context.Context, query string, args ...any) ([]map[string]any, error)
}

// SQLiteProjectionStore stores one SQLite projection database per projection
// name under a local directory.
type SQLiteProjectionStore struct {
	root string
	mu   sync.Mutex
	dbs  map[string]*sql.DB
}

func NewSQLiteProjectionStore(root string) *SQLiteProjectionStore {
	return &SQLiteProjectionStore{root: root, dbs: map[string]*sql.DB{}}
}

func (s *SQLiteProjectionStore) Projection(ctx context.Context, name string) (Projection, error) {
	if s == nil || strings.TrimSpace(s.root) == "" {
		return nil, fmt.Errorf("sqlite projection store root is required")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("projection name is required")
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if db := s.dbs[name]; db != nil {
		return sqliteProjection{db: db}, nil
	}
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(s.root, safeProjectionName(name)+".db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	s.dbs[name] = db
	return sqliteProjection{db: db}, nil
}

func (s *SQLiteProjectionStore) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var firstErr error
	for name, db := range s.dbs {
		if err := db.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close projection %s: %w", name, err)
		}
		delete(s.dbs, name)
	}
	return firstErr
}

type sqliteProjection struct {
	db *sql.DB
}

func (p sqliteProjection) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	result, err := p.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return rows, nil
}

func (p sqliteProjection) Query(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	ret := []map[string]any{}
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		m := map[string]any{}
		for i, col := range cols {
			switch v := values[i].(type) {
			case []byte:
				m[col] = string(v)
			default:
				m[col] = v
			}
		}
		ret = append(ret, m)
	}
	return ret, rows.Err()
}

func safeProjectionName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "projection"
	}
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, name)
}
