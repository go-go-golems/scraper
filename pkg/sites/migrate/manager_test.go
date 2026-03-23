package migrate

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"testing/fstest"
)

func TestManagerAppliesMixedSQLAndJSMigrations(t *testing.T) {
	registry := siteregistry.New()
	require.NoError(t, registry.Register(siteregistry.Definition{
		Name: model.SiteName("demo"),
		SQLMigrationsFS: fstest.MapFS{
			"001_init.sql": &fstest.MapFile{Data: []byte(`
				CREATE TABLE widgets (
					id INTEGER PRIMARY KEY,
					name TEXT NOT NULL
				);
			`)},
		},
		JSMigrationsFS: fstest.MapFS{
			"002_seed.js": &fstest.MapFile{Data: []byte(`
				module.exports = async function(m) {
					m.exec("ALTER TABLE widgets ADD COLUMN notes TEXT");
					m.exec("INSERT INTO widgets(name, notes) VALUES(?, ?)", "alpha", "seeded");
				};
			`)},
		},
	}))

	manager := NewManager(registry)
	report, err := manager.Migrate(context.Background(), model.SiteName("demo"), t.TempDir())
	require.NoError(t, err)
	require.Equal(t, 2, report.AppliedCount)
	require.Equal(t, 2, report.CurrentVersion)
	require.Equal(t, filepath.Base(report.DatabasePath), "demo.db")

	rows := openRows(t, report.DatabasePath, `SELECT name, notes FROM widgets`)
	defer rows.Close()

	require.True(t, rows.Next())
	var name string
	var notes string
	require.NoError(t, rows.Scan(&name, &notes))
	require.Equal(t, "alpha", name)
	require.Equal(t, "seeded", notes)
}

func TestManagerSupportsRelativeRequireInJSMigrations(t *testing.T) {
	registry := siteregistry.New()
	require.NoError(t, registry.Register(siteregistry.Definition{
		Name: model.SiteName("demo"),
		SQLMigrationsFS: fstest.MapFS{
			"001_init.sql": &fstest.MapFile{Data: []byte(`
				CREATE TABLE widgets (
					id INTEGER PRIMARY KEY,
					name TEXT NOT NULL
				);
			`)},
		},
		JSMigrationsFS: fstest.MapFS{
			"lib/helper.js": &fstest.MapFile{Data: []byte(`
				module.exports = {
					name: function() { return "from-helper"; }
				};
			`)},
			"002_seed.js": &fstest.MapFile{Data: []byte(`
				const helper = require("./lib/helper.js");
				const siteDB = require("site-db");
				module.exports = function(m) {
					siteDB.exec("INSERT INTO widgets(name) VALUES(?)", helper.name());
				};
			`)},
		},
	}))

	manager := NewManager(registry)
	report, err := manager.Migrate(context.Background(), model.SiteName("demo"), t.TempDir())
	require.NoError(t, err)

	rows := openRows(t, report.DatabasePath, `SELECT name FROM widgets`)
	defer rows.Close()

	require.True(t, rows.Next())
	var name string
	require.NoError(t, rows.Scan(&name))
	require.Equal(t, "from-helper", name)
}

func TestManagerIsIdempotentOnRerun(t *testing.T) {
	registry := siteregistry.New()
	require.NoError(t, registry.Register(siteregistry.Definition{
		Name: model.SiteName("demo"),
		SQLMigrationsFS: fstest.MapFS{
			"001_init.sql": &fstest.MapFile{Data: []byte(`CREATE TABLE widgets(id INTEGER PRIMARY KEY);`)},
		},
	}))

	manager := NewManager(registry)
	sitesDir := t.TempDir()

	first, err := manager.Migrate(context.Background(), model.SiteName("demo"), sitesDir)
	require.NoError(t, err)
	require.Equal(t, 1, first.AppliedCount)

	second, err := manager.Migrate(context.Background(), model.SiteName("demo"), sitesDir)
	require.NoError(t, err)
	require.Equal(t, 0, second.AppliedCount)
	require.Equal(t, 1, second.CurrentVersion)
}

func TestManagerRejectsDuplicateVersionsAcrossSQLAndJS(t *testing.T) {
	registry := siteregistry.New()
	require.NoError(t, registry.Register(siteregistry.Definition{
		Name: model.SiteName("demo"),
		SQLMigrationsFS: fstest.MapFS{
			"001_init.sql": &fstest.MapFile{Data: []byte(`CREATE TABLE widgets(id INTEGER PRIMARY KEY);`)},
		},
		JSMigrationsFS: fstest.MapFS{
			"001_seed.js": &fstest.MapFile{Data: []byte(`module.exports = function(m) {};`)},
		},
	}))

	manager := NewManager(registry)
	_, err := manager.Migrate(context.Background(), model.SiteName("demo"), t.TempDir())
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate site migration version")
}

func openRows(t *testing.T, path string, query string) *sql.Rows {
	t.Helper()

	db, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	rows, err := db.Query(query)
	require.NoError(t, err)
	return rows
}

var _ fs.FS = fstest.MapFS{}
