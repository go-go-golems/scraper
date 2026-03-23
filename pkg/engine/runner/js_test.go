package runner

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	"github.com/stretchr/testify/require"
	"testing/fstest"
)

type runnerDependencyResolver struct{}

func (runnerDependencyResolver) Result(ctx context.Context, workflowID model.WorkflowID, opID model.OpID) (*model.OpResult, error) {
	return &model.OpResult{
		OpID: opID,
		Data: json.RawMessage(`{"value":"from-dependency"}`),
	}, nil
}

func TestJSRunnerExecutesSiteScript(t *testing.T) {
	registry := siteregistry.New()
	require.NoError(t, registry.Register(siteregistry.Definition{
		Name: "nereval",
		ScriptsFS: fstest.MapFS{
			"scripts/extract.js": &fstest.MapFile{
				Data: []byte(`
					const siteDB = require("site-db");
					const scraperDB = require("scraper-db");

					module.exports = function (ctx) {
						const dep = ctx.dep("dep-1");
						return {
							data: {
								inputTown: ctx.input.town,
								depValue: dep.data.value,
								siteRows: siteDB.query("SELECT name FROM site_rows ORDER BY name"),
								scraperRows: scraperDB.query("SELECT name FROM scraper_rows ORDER BY name")
							}
						};
					};
				`),
			},
		},
		ScriptsRoot: "scripts",
	}))

	runner := NewJSRunner(registry)
	now := time.Date(2026, 3, 23, 13, 30, 0, 0, time.UTC)

	scraperDB := openRunnerDB(t, "scraper.db")
	siteDB := openRunnerDB(t, "site.db")
	insertRunnerRow(t, scraperDB, "scraper_rows", "engine-row")
	insertRunnerRow(t, siteDB, "site_rows", "site-row")

	result, err := runner.Run(context.Background(), RunContext{
		Workflow: model.WorkflowRun{
			ID:   "wf-1",
			Site: "nereval",
		},
		Op: model.OpSpec{
			ID:         "op-1",
			WorkflowID: "wf-1",
			Site:       "nereval",
			Kind:       "js",
			Input:      json.RawMessage(`{"town":"Milford"}`),
			Metadata:   map[string]string{"script": "extract.js"},
		},
		Lease: model.Lease{
			WorkerID:   "worker-1",
			Token:      "lease-1",
			AcquiredAt: now,
			ExpiresAt:  now.Add(30 * time.Second),
		},
		Now:          now,
		Dependencies: runnerDependencyResolver{},
		ScraperDB:    scraperDB,
		SiteDB:       siteDB,
	})
	require.NoError(t, err)
	require.JSONEq(t, `{
		"inputTown": "Milford",
		"depValue": "from-dependency",
		"siteRows": [{"name":"site-row"}],
		"scraperRows": [{"name":"engine-row"}]
	}`, string(result.Data))
}

func openRunnerDB(t *testing.T, name string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), name))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	return db
}

func insertRunnerRow(t *testing.T, db *sql.DB, table, value string) {
	t.Helper()

	_, err := db.Exec(`CREATE TABLE ` + table + ` (name TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO `+table+`(name) VALUES (?)`, value)
	require.NoError(t, err)
}
