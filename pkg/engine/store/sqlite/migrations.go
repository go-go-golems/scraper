package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type migration struct {
	version int
	name    string
	sql     string
}

func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	for _, m := range migrations {
		applied, err := isMigrationApplied(ctx, db, m.version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", m.version, err)
		}

		if _, err := tx.ExecContext(ctx, m.sql); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", m.version, err)
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO schema_migrations(version, name, applied_at) VALUES(?, ?, ?)`,
			m.version,
			m.name,
			time.Now().UTC().Format(time.RFC3339Nano),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", m.version, err)
		}
	}

	return nil
}

func currentVersion(ctx context.Context, db *sql.DB) (int, error) {
	var version sql.NullInt64
	if err := db.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		return 0, fmt.Errorf("query current schema version: %w", err)
	}
	if !version.Valid {
		return 0, nil
	}
	return int(version.Int64), nil
}

func isMigrationApplied(ctx context.Context, db *sql.DB, version int) (bool, error) {
	var count int
	if err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(1) FROM schema_migrations WHERE version = ?`,
		version,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("query migration %d: %w", version, err)
	}

	return count > 0, nil
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}

	ret := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		versionText, _, ok := strings.Cut(entry.Name(), "_")
		if !ok {
			return nil, fmt.Errorf("migration filename must start with numeric prefix: %s", entry.Name())
		}

		version, err := strconv.Atoi(versionText)
		if err != nil {
			return nil, fmt.Errorf("parse migration version %q: %w", versionText, err)
		}

		body, err := fs.ReadFile(migrationsFS, "migrations/"+entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		ret = append(ret, migration{
			version: version,
			name:    entry.Name(),
			sql:     string(body),
		})
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].version < ret[j].version
	})

	return ret, nil
}
