package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	scraperjsruntime "github.com/go-go-golems/scraper/pkg/js/runtime"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

type MigrationKind string

const (
	MigrationKindSQL MigrationKind = "sql"
	MigrationKindJS  MigrationKind = "js"
)

type Migration struct {
	Version int
	Name    string
	Kind    MigrationKind
	Path    string
}

type Report struct {
	Site           model.SiteName
	DatabasePath   string
	AppliedCount   int
	CurrentVersion int
	Migrations     []Migration
}

type Manager struct {
	registry *siteregistry.Registry
}

func NewManager(registry *siteregistry.Registry) *Manager {
	if registry == nil {
		registry = siteregistry.New()
	}

	return &Manager{
		registry: registry,
	}
}

func (m *Manager) DatabasePath(sitesDir string, def siteregistry.Definition) string {
	filename := def.DatabaseFileName
	if strings.TrimSpace(filename) == "" {
		filename = fmt.Sprintf("%s.db", def.Name)
	}
	return filepath.Join(sitesDir, filename)
}

func (m *Manager) Migrate(ctx context.Context, site model.SiteName, sitesDir string) (*Report, error) {
	def, ok := m.registry.Get(site)
	if !ok {
		return nil, fmt.Errorf("site %q is not registered", site)
	}

	dbPath := m.DatabasePath(sitesDir, def)
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create site db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open site db: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := ensureMigrationTable(ctx, db); err != nil {
		return nil, err
	}

	migrations, err := loadMigrations(def)
	if err != nil {
		return nil, err
	}

	appliedCount := 0
	for _, migration := range migrations {
		applied, err := isMigrationApplied(ctx, db, migration.Version)
		if err != nil {
			return nil, err
		}
		if applied {
			continue
		}

		if err := applyMigration(ctx, db, def, migration); err != nil {
			return nil, fmt.Errorf("apply %s migration %s: %w", migration.Kind, migration.Name, err)
		}
		appliedCount++
	}

	currentVersion, err := currentVersion(ctx, db)
	if err != nil {
		return nil, err
	}

	return &Report{
		Site:           site,
		DatabasePath:   dbPath,
		AppliedCount:   appliedCount,
		CurrentVersion: currentVersion,
		Migrations:     migrations,
	}, nil
}

func ensureMigrationTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("ensure site schema_migrations table: %w", err)
	}

	return nil
}

func isMigrationApplied(ctx context.Context, db *sql.DB, version int) (bool, error) {
	var count int
	if err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(1) FROM schema_migrations WHERE version = ?`,
		version,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("query site migration %d: %w", version, err)
	}

	return count > 0, nil
}

func currentVersion(ctx context.Context, db *sql.DB) (int, error) {
	var version sql.NullInt64
	if err := db.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		return 0, fmt.Errorf("query current site schema version: %w", err)
	}
	if !version.Valid {
		return 0, nil
	}
	return int(version.Int64), nil
}

func applyMigration(
	ctx context.Context,
	db *sql.DB,
	def siteregistry.Definition,
	migration Migration,
) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin site migration transaction: %w", err)
	}

	switch migration.Kind {
	case MigrationKindSQL:
		body, err := readMigrationBody(def.SQLMigrationsFS, def.SQLMigrationsRoot, migration.Path)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := tx.ExecContext(ctx, body); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("execute sql migration: %w", err)
		}
	case MigrationKindJS:
		if err := runJSMigration(ctx, tx, def, migration); err != nil {
			_ = tx.Rollback()
			return err
		}
	default:
		_ = tx.Rollback()
		return fmt.Errorf("unsupported migration kind %q", migration.Kind)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO schema_migrations(version, name, applied_at) VALUES(?, ?, ?)`,
		migration.Version,
		migration.Name,
		time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record site migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit site migration: %w", err)
	}

	return nil
}

func loadMigrations(def siteregistry.Definition) ([]Migration, error) {
	ret := []Migration{}
	seenVersions := map[int]string{}

	sqlMigrations, err := loadMigrationsFromFS(def.SQLMigrationsFS, def.SQLMigrationsRoot, MigrationKindSQL)
	if err != nil {
		return nil, err
	}
	jsMigrations, err := loadMigrationsFromFS(def.JSMigrationsFS, def.JSMigrationsRoot, MigrationKindJS)
	if err != nil {
		return nil, err
	}

	for _, migration := range append(sqlMigrations, jsMigrations...) {
		if existing, ok := seenVersions[migration.Version]; ok {
			return nil, fmt.Errorf(
				"duplicate site migration version %03d across %s and %s",
				migration.Version,
				existing,
				migration.Name,
			)
		}
		seenVersions[migration.Version] = migration.Name
		ret = append(ret, migration)
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Version < ret[j].Version
	})

	return ret, nil
}

func loadMigrationsFromFS(fsys fs.FS, root string, kind MigrationKind) ([]Migration, error) {
	if fsys == nil {
		return nil, nil
	}

	entries, err := fs.ReadDir(subFS(fsys, root), ".")
	if err != nil {
		if errorsIsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s migrations: %w", kind, err)
	}

	ext := ".sql"
	if kind == MigrationKindJS {
		ext = ".js"
	}

	ret := []Migration{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ext) {
			continue
		}
		version, err := parseMigrationVersion(entry.Name())
		if err != nil {
			return nil, err
		}
		ret = append(ret, Migration{
			Version: version,
			Name:    entry.Name(),
			Kind:    kind,
			Path:    entry.Name(),
		})
	}

	return ret, nil
}

func parseMigrationVersion(name string) (int, error) {
	versionText, _, ok := strings.Cut(name, "_")
	if !ok {
		return 0, fmt.Errorf("migration filename must start with numeric prefix: %s", name)
	}

	version, err := strconv.Atoi(versionText)
	if err != nil {
		return 0, fmt.Errorf("parse migration version %q: %w", versionText, err)
	}

	return version, nil
}

func readMigrationBody(fsys fs.FS, root, path string) (string, error) {
	body, err := fs.ReadFile(subFS(fsys, root), path)
	if err != nil {
		return "", fmt.Errorf("read migration body %s: %w", path, err)
	}
	return string(body), nil
}

func runJSMigration(
	ctx context.Context,
	tx *sql.Tx,
	def siteregistry.Definition,
	migration Migration,
) error {
	loader := migrationLoader(def.JSMigrationsFS, def.JSMigrationsRoot)

	builder := gggengine.NewBuilder().
		WithRequireOptions(require.WithLoader(loader)).
		WithRuntimeModuleRegistrars(scraperjsruntime.NewDatabaseRegistrar(scraperjsruntime.DatabaseRegistrarConfig{
			SiteDB: tx,
		}))
	builder = builder.WithRuntimeModuleRegistrars(def.RuntimeModuleRegistrars...)

	factory, err := builder.Build()
	if err != nil {
		return fmt.Errorf("build js migration runtime: %w", err)
	}

	runtime, err := factory.NewRuntime(ctx)
	if err != nil {
		return fmt.Errorf("create js migration runtime: %w", err)
	}
	defer func() {
		_ = runtime.Close(context.Background())
	}()

	ret, err := runtime.Owner.Call(ctx, "site-migration.run", func(_ context.Context, vm *goja.Runtime) (any, error) {
		migrationValue, err := runtime.Require.Require("./" + migration.Path)
		if err != nil {
			return nil, fmt.Errorf("require migration %s: %w", migration.Path, err)
		}

		fn, ok := goja.AssertFunction(migrationValue)
		if !ok {
			return nil, fmt.Errorf("migration %s must export a function", migration.Name)
		}

		result, err := fn(goja.Undefined(), buildMigrationAPI(vm, tx, def.Name))
		if err != nil {
			return nil, err
		}
		if result == nil || goja.IsUndefined(result) || goja.IsNull(result) {
			return nil, nil
		}
		if promise, ok := result.Export().(*goja.Promise); ok {
			return promise, nil
		}
		return result.Export(), nil
	})
	if err != nil {
		return fmt.Errorf("execute js migration %s: %w", migration.Name, err)
	}

	if promise, ok := ret.(*goja.Promise); ok {
		if _, err := waitForPromise(ctx, runtime, promise); err != nil {
			return fmt.Errorf("await js migration promise: %w", err)
		}
	}

	return nil
}

func migrationLoader(fsys fs.FS, root string) func(modulePath string) ([]byte, error) {
	migrationFS := subFS(fsys, root)
	return func(modulePath string) ([]byte, error) {
		candidates := []string{
			cleanModulePath(modulePath),
			cleanModulePath(modulePath) + ".js",
		}
		for _, candidate := range candidates {
			body, err := fs.ReadFile(migrationFS, candidate)
			if err == nil {
				return body, nil
			}
			if errorsIsNotExist(err) {
				continue
			}
			return nil, err
		}
		return nil, require.ModuleFileDoesNotExistError
	}
}

func buildMigrationAPI(vm *goja.Runtime, tx *sql.Tx, site model.SiteName) goja.Value {
	api := vm.NewObject()

	_ = api.Set("exec", func(call goja.FunctionCall) goja.Value {
		sqlText := call.Argument(0).String()
		args := exportArgs(call.Arguments[1:])
		result, err := tx.ExecContext(context.Background(), sqlText, args...)
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("exec %q: %w", sqlText, err)))
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("rows affected for %q: %w", sqlText, err)))
		}
		return vm.ToValue(rowsAffected)
	})

	_ = api.Set("query", func(call goja.FunctionCall) goja.Value {
		sqlText := call.Argument(0).String()
		args := exportArgs(call.Arguments[1:])
		rows, err := tx.QueryContext(context.Background(), sqlText, args...)
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("query %q: %w", sqlText, err)))
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("read query columns: %w", err)))
		}

		ret := make([]map[string]any, 0)
		for rows.Next() {
			values := make([]any, len(columns))
			scanTargets := make([]any, len(columns))
			for i := range values {
				scanTargets[i] = &values[i]
			}
			if err := rows.Scan(scanTargets...); err != nil {
				panic(vm.NewGoError(fmt.Errorf("scan query row: %w", err)))
			}

			row := map[string]any{}
			for i, column := range columns {
				row[column] = normalizeDBValue(values[i])
			}
			ret = append(ret, row)
		}
		if err := rows.Err(); err != nil {
			panic(vm.NewGoError(fmt.Errorf("iterate query rows: %w", err)))
		}

		return vm.ToValue(ret)
	})

	_ = api.Set("hasTable", func(name string) bool {
		var count int
		if err := tx.QueryRowContext(
			context.Background(),
			`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`,
			name,
		).Scan(&count); err != nil {
			panic(vm.NewGoError(fmt.Errorf("check table %q: %w", name, err)))
		}
		return count > 0
	})

	_ = api.Set("hasColumn", func(table string, column string) bool {
		var count int
		if err := tx.QueryRowContext(
			context.Background(),
			`SELECT COUNT(1) FROM pragma_table_info(?) WHERE name = ?`,
			table,
			column,
		).Scan(&count); err != nil {
			panic(vm.NewGoError(fmt.Errorf("check column %q on %q: %w", column, table, err)))
		}
		return count > 0
	})

	_ = api.Set("log", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, 0, len(call.Arguments))
		for _, arg := range call.Arguments {
			parts = append(parts, fmt.Sprint(arg.Export()))
		}
		log.Info().
			Str("site", string(site)).
			Str("component", "site-migrate").
			Msg(strings.Join(parts, " "))
		return goja.Undefined()
	})

	return api
}

func waitForPromise(ctx context.Context, runtime *gggengine.Runtime, promise *goja.Promise) (any, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		ret, err := runtime.Owner.Call(ctx, "site-migration.promise", func(_ context.Context, vm *goja.Runtime) (any, error) {
			return promiseSnapshot{
				State:  promise.State(),
				Result: promise.Result(),
			}, nil
		})
		if err != nil {
			return nil, err
		}

		snapshot := ret.(promiseSnapshot)
		switch snapshot.State {
		case goja.PromiseStatePending:
			time.Sleep(5 * time.Millisecond)
			continue
		case goja.PromiseStateFulfilled:
			if snapshot.Result == nil || goja.IsUndefined(snapshot.Result) || goja.IsNull(snapshot.Result) {
				return nil, nil
			}
			return snapshot.Result.Export(), nil
		case goja.PromiseStateRejected:
			if snapshot.Result == nil || goja.IsUndefined(snapshot.Result) || goja.IsNull(snapshot.Result) {
				return nil, fmt.Errorf("promise rejected")
			}
			return nil, fmt.Errorf("promise rejected: %v", snapshot.Result.Export())
		default:
			return nil, fmt.Errorf("unknown promise state %v", snapshot.State)
		}
	}
}

type promiseSnapshot struct {
	State  goja.PromiseState
	Result goja.Value
}

func cleanModulePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "/")
	return filepath.Clean(path)
}

func subFS(fsys fs.FS, root string) fs.FS {
	if fsys == nil {
		return os.DirFS(".")
	}
	root = strings.TrimSpace(root)
	if root == "" || root == "." {
		return fsys
	}
	sub, err := fs.Sub(fsys, root)
	if err != nil {
		return fsys
	}
	return sub
}

func errorsIsNotExist(err error) bool {
	return err != nil && (os.IsNotExist(err) || strings.Contains(err.Error(), "file does not exist"))
}

func exportArgs(values []goja.Value) []any {
	ret := make([]any, 0, len(values))
	for _, value := range values {
		ret = append(ret, value.Export())
	}
	return ret
}

func normalizeDBValue(v any) any {
	switch typed := v.(type) {
	case []byte:
		return string(typed)
	default:
		return typed
	}
}
