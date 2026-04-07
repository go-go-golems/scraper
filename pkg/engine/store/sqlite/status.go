package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	_ "github.com/mattn/go-sqlite3"
)

type MigrationStatus struct {
	Version   int
	Name      string
	Applied   bool
	AppliedAt *time.Time
}

type EngineStatus struct {
	Path                 string
	Exists               bool
	Initialized          bool
	CurrentVersion       int
	LatestKnownMigration int
	MigrationsUpToDate   bool
	WorkflowCount        int
	WorkflowCounts       map[model.WorkflowStatus]int
	OpCounts             map[model.OpStatus]int
	ActiveLeases         int
	ExpiredLeases        int
	ResultCount          int
	ArtifactCount        int
	Migrations           []MigrationStatus
}

func LatestMigrationVersion() (int, error) {
	migrations, err := loadMigrations()
	if err != nil {
		return 0, err
	}
	if len(migrations) == 0 {
		return 0, nil
	}
	return migrations[len(migrations)-1].version, nil
}

func Inspect(ctx context.Context, dsn string) (*EngineStatus, error) {
	latest, err := LatestMigrationVersion()
	if err != nil {
		return nil, err
	}

	status := &EngineStatus{
		Path:                 dsn,
		LatestKnownMigration: latest,
		WorkflowCounts:       map[model.WorkflowStatus]int{},
		OpCounts:             map[model.OpStatus]int{},
	}

	if _, err := os.Stat(dsn); err != nil {
		if os.IsNotExist(err) {
			migrations, err := loadKnownMigrationStatuses()
			if err != nil {
				return nil, err
			}
			status.Migrations = migrations
			return status, nil
		}
		return nil, fmt.Errorf("stat engine db: %w", err)
	}

	status.Exists = true

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db for inspection: %w", err)
	}
	defer func() { _ = db.Close() }()

	initialized, err := hasTable(ctx, db, "schema_migrations")
	if err != nil {
		return nil, err
	}
	status.Initialized = initialized
	if !initialized {
		migrations, err := loadKnownMigrationStatuses()
		if err != nil {
			return nil, err
		}
		status.Migrations = migrations
		return status, nil
	}

	status.CurrentVersion, err = currentVersion(ctx, db)
	if err != nil {
		return nil, err
	}
	status.MigrationsUpToDate = status.CurrentVersion == status.LatestKnownMigration

	status.Migrations, err = loadAppliedMigrationStatuses(ctx, db)
	if err != nil {
		return nil, err
	}

	if status.WorkflowCount, err = countTable(ctx, db, "workflows"); err != nil {
		return nil, err
	}
	if status.WorkflowCounts, err = countWorkflowsByStatus(ctx, db); err != nil {
		return nil, err
	}
	if status.ResultCount, err = countTable(ctx, db, "results"); err != nil {
		return nil, err
	}
	if status.ArtifactCount, err = countTable(ctx, db, "artifacts"); err != nil {
		return nil, err
	}
	if status.ActiveLeases, err = countWhere(
		ctx,
		db,
		"leases",
		"expires_at > ?",
		time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		return nil, err
	}
	if status.ExpiredLeases, err = countWhere(
		ctx,
		db,
		"leases",
		"expires_at <= ?",
		time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		return nil, err
	}

	opCounts, err := countOpsByStatus(ctx, db)
	if err != nil {
		return nil, err
	}
	status.OpCounts = opCounts

	return status, nil
}

func hasTable(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var count int
	if err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`,
		name,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("check table %s: %w", name, err)
	}
	return count > 0, nil
}

func countTable(ctx context.Context, db *sql.DB, table string) (int, error) {
	return countWhere(ctx, db, table, "1 = 1")
}

func countWhere(ctx context.Context, db *sql.DB, table, predicate string, args ...any) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(1) FROM %s WHERE %s", table, predicate)
	var count int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count rows for %s: %w", table, err)
	}
	return count, nil
}

func countOpsByStatus(ctx context.Context, db *sql.DB) (map[model.OpStatus]int, error) {
	rows, err := db.QueryContext(ctx, `SELECT status, COUNT(1) FROM ops GROUP BY status ORDER BY status`)
	if err != nil {
		return nil, fmt.Errorf("count ops by status: %w", err)
	}
	defer rows.Close()

	ret := map[model.OpStatus]int{}
	for rows.Next() {
		var status model.OpStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan op status count: %w", err)
		}
		ret[status] = count
	}

	return ret, rows.Err()
}

func countWorkflowsByStatus(ctx context.Context, db *sql.DB) (map[model.WorkflowStatus]int, error) {
	rows, err := db.QueryContext(ctx, `SELECT status, COUNT(1) FROM workflows GROUP BY status ORDER BY status`)
	if err != nil {
		return nil, fmt.Errorf("count workflows by status: %w", err)
	}
	defer rows.Close()

	ret := map[model.WorkflowStatus]int{}
	for rows.Next() {
		var status model.WorkflowStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan workflow status count: %w", err)
		}
		ret[status] = count
	}

	return ret, rows.Err()
}

func loadKnownMigrationStatuses() ([]MigrationStatus, error) {
	migrations, err := loadMigrations()
	if err != nil {
		return nil, err
	}

	ret := make([]MigrationStatus, 0, len(migrations))
	for _, migration := range migrations {
		ret = append(ret, MigrationStatus{
			Version: migration.version,
			Name:    migration.name,
			Applied: false,
		})
	}

	return ret, nil
}

func loadAppliedMigrationStatuses(ctx context.Context, db *sql.DB) ([]MigrationStatus, error) {
	migrations, err := loadKnownMigrationStatuses()
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `SELECT version, applied_at FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	appliedAtByVersion := map[int]time.Time{}
	for rows.Next() {
		var version int
		var appliedAtText string
		if err := rows.Scan(&version, &appliedAtText); err != nil {
			return nil, fmt.Errorf("scan applied migration: %w", err)
		}
		appliedAt, err := time.Parse(time.RFC3339Nano, appliedAtText)
		if err != nil {
			return nil, fmt.Errorf("parse migration applied_at: %w", err)
		}
		appliedAtByVersion[version] = appliedAt
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range migrations {
		appliedAt, ok := appliedAtByVersion[migrations[i].Version]
		if !ok {
			continue
		}
		migrations[i].Applied = true
		appliedAtCopy := appliedAt
		migrations[i].AppliedAt = &appliedAtCopy
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}
