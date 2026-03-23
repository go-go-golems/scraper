package runtime

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestDatabaseRegistrarExposesScraperAndSiteDBs(t *testing.T) {
	scraperDB := openDB(t, "scraper.db")
	siteDB := openDB(t, "site.db")

	_, err := scraperDB.Exec(`CREATE TABLE engine_rows (name TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = scraperDB.Exec(`INSERT INTO engine_rows(name) VALUES (?)`, "from-scraper-db")
	require.NoError(t, err)

	_, err = siteDB.Exec(`CREATE TABLE site_rows (name TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = siteDB.Exec(`INSERT INTO site_rows(name) VALUES (?)`, "from-site-db")
	require.NoError(t, err)

	factory, err := gggengine.NewBuilder().
		WithRuntimeModuleRegistrars(NewDatabaseRegistrar(DatabaseRegistrarConfig{
			ScraperDB: scraperDB,
			SiteDB:    siteDB,
		})).
		Build()
	require.NoError(t, err)

	rt, err := factory.NewRuntime(context.Background())
	require.NoError(t, err)
	defer func() { require.NoError(t, rt.Close(context.Background())) }()

	ret, err := rt.Owner.Call(context.Background(), "runtime.preconfigured-dbs", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, err := rt.VM.RunString(`
			const scraperDB = require("scraper-db");
			const siteDB = require("site-db");
			JSON.stringify({
				scraper: scraperDB.query("SELECT name FROM engine_rows ORDER BY name"),
				site: siteDB.query("SELECT name FROM site_rows ORDER BY name")
			});
		`)
		if err != nil {
			return nil, err
		}
		return value.Export(), nil
	})
	require.NoError(t, err)
	require.Equal(t, `{"scraper":[{"name":"from-scraper-db"}],"site":[{"name":"from-site-db"}]}`, ret)
}

func openDB(t *testing.T, name string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), name))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	return db
}
