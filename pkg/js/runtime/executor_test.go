package runtime

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	noderequire "github.com/dop251/goja_nodejs/require"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/stretchr/testify/require"
	"testing/fstest"
)

type fakeDependencyResolver struct {
	results map[model.OpID]*model.OpResult
}

func (r fakeDependencyResolver) Result(ctx context.Context, workflowID model.WorkflowID, opID model.OpID) (*model.OpResult, error) {
	return r.results[opID], nil
}

func TestExecutorRunsScriptAndCapturesSideEffects(t *testing.T) {
	scraperDB := openExecutorDB(t, "scraper.db")
	siteDB := openExecutorDB(t, "site.db")
	createExecutorTable(t, scraperDB, "scraper_rows", "from-scraper")
	createExecutorTable(t, siteDB, "site_rows", "from-site")

	executor := NewExecutor(ExecutorConfig{
		ScriptsFS: fstest.MapFS{
			"scripts/main.js": &fstest.MapFile{
				Data: []byte(`
					const helper = require("./lib/helper.js");
					const scraperDB = require("scraper-db");
					const siteDB = require("site-db");

					module.exports = async function (ctx) {
						const dep = ctx.dep("fetch-1");
						const scraperRows = scraperDB.query("SELECT name FROM scraper_rows ORDER BY name");
						const siteRows = siteDB.query("SELECT name FROM site_rows ORDER BY name");

						const emittedID = ctx.emit({
							kind: "js",
							queue: "site:test:js",
							dedupKey: "child:1",
							metadata: { script: "child.js" },
							input: {
								greeting: helper.upper(dep.data.message),
								scraperName: scraperRows[0].name,
								siteName: siteRows[0].name
							},
							retry: {
								maxAttempts: 3,
								backoffKind: "fixed",
								initialBackoff: "1s",
								maxBackoff: "5s",
								multiplier: 1
							},
							dependsOn: [{ opID: "fetch-1" }]
						});

						ctx.writeRecord("example-records", "acct:A-1", {
							message: dep.data.message,
							helperName: helper.upper(ctx.input.name)
						});

						const artifactID = ctx.writeArtifact({
							name: "payload.txt",
							kind: "text",
							contentType: "text/plain",
							metadata: { source: "unit-test" },
							body: helper.upper(dep.data.message)
						});

						return {
							data: {
								emittedID,
								artifactID,
								depMessage: dep.data.message,
								depArtifact: dep.artifacts[0].bodyText,
								scraperName: scraperRows[0].name,
								siteName: siteRows[0].name,
								helperName: helper.upper(ctx.input.name)
							}
						};
					};
				`),
			},
			"scripts/lib/helper.js": &fstest.MapFile{
				Data: []byte(`
					module.exports = {
						upper(value) {
							return String(value).toUpperCase();
						}
					};
				`),
			},
		},
		ScriptsRoot: "scripts",
		ScraperDB:   scraperDB,
		SiteDB:      siteDB,
	})

	now := time.Date(2026, 3, 23, 12, 30, 0, 0, time.UTC)
	result, err := executor.Execute(context.Background(), ExecutionRequest{
		Workflow: model.WorkflowRun{
			ID:       "wf-1",
			Site:     "test-site",
			Name:     "test workflow",
			Status:   model.WorkflowStatusRunning,
			Input:    json.RawMessage(`{"town":"Nereval"}`),
			Metadata: map[string]string{"source": "unit"},
		},
		Op: model.OpSpec{
			ID:         "root-op",
			WorkflowID: "wf-1",
			Site:       "test-site",
			Kind:       "js",
			Queue:      "site:test:js",
			Input:      json.RawMessage(`{"name":"manuel"}`),
			Metadata:   map[string]string{MetadataScript: "main.js"},
		},
		Lease: model.Lease{
			WorkerID:   "worker-1",
			Token:      "lease-1",
			AcquiredAt: now,
			ExpiresAt:  now.Add(30 * time.Second),
		},
		Now: now,
		Dependencies: fakeDependencyResolver{
			results: map[model.OpID]*model.OpResult{
				"fetch-1": {
					OpID: "fetch-1",
					Data: json.RawMessage(`{"message":"hello"}`),
					Artifacts: []model.ArtifactWrite{
						{
							ID:          "artifact-fetch-1",
							Name:        "fetch.html",
							Kind:        "html",
							ContentType: "text/html",
							Body:        []byte("<html>hello</html>"),
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	require.JSONEq(t, `{
		"emittedID": "root-op:emit:001",
		"artifactID": "root-op:artifact:001",
		"depMessage": "hello",
		"depArtifact": "<html>hello</html>",
		"scraperName": "from-scraper",
		"siteName": "from-site",
		"helperName": "MANUEL"
	}`, string(result.Data))
	require.Len(t, result.Emitted, 1)
	require.Equal(t, []model.OpID{"root-op:emit:001"}, result.EmittedIDs)
	require.Equal(t, "js", result.Emitted[0].Kind)
	require.NotNil(t, result.Emitted[0].ParentID)
	require.Equal(t, model.OpID("root-op"), *result.Emitted[0].ParentID)
	require.Equal(t, model.WorkflowID("wf-1"), result.Emitted[0].WorkflowID)
	require.Equal(t, model.SiteName("test-site"), result.Emitted[0].Site)
	require.JSONEq(t, `{
		"greeting": "HELLO",
		"scraperName": "from-scraper",
		"siteName": "from-site"
	}`, string(result.Emitted[0].Input))
	require.Len(t, result.Records, 1)
	require.Equal(t, "example-records", result.Records[0].Collection)
	require.Equal(t, "acct:A-1", result.Records[0].Key)
	require.JSONEq(t, `{"message":"hello","helperName":"MANUEL"}`, string(result.Records[0].Data))
	require.Len(t, result.Artifacts, 1)
	require.Equal(t, model.ArtifactID("root-op:artifact:001"), result.Artifacts[0].ID)
	require.Equal(t, "payload.txt", result.Artifacts[0].Name)
	require.Equal(t, "text/plain", result.Artifacts[0].ContentType)
	require.Equal(t, "HELLO", string(result.Artifacts[0].Body))
	require.Equal(t, now, result.CompletedAt)
}

type closingRegistrar struct {
	closed *bool
}

func (r closingRegistrar) ID() string {
	return "closing-registrar"
}

func (r closingRegistrar) RegisterRuntimeModules(ctx *gggengine.RuntimeModuleContext, reg *noderequire.Registry) error {
	if ctx == nil {
		return nil
	}
	return ctx.AddCloser(func(context.Context) error {
		*r.closed = true
		return nil
	})
}

func TestExecutorClosesRuntimeClosers(t *testing.T) {
	closed := false
	executor := NewExecutor(ExecutorConfig{
		ScriptsFS: fstest.MapFS{
			"scripts/main.js": &fstest.MapFile{
				Data: []byte(`module.exports = function () { return { data: { ok: true } }; };`),
			},
		},
		ScriptsRoot: "scripts",
		RuntimeModuleRegistrars: []gggengine.RuntimeModuleRegistrar{
			closingRegistrar{closed: &closed},
		},
	})

	result, err := executor.Execute(context.Background(), ExecutionRequest{
		Workflow: model.WorkflowRun{ID: "wf-1", Site: "test-site"},
		Op: model.OpSpec{
			ID:         "op-1",
			WorkflowID: "wf-1",
			Site:       "test-site",
			Kind:       "js",
			Metadata:   map[string]string{MetadataScript: "main.js"},
		},
		Now: time.Date(2026, 3, 23, 13, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"ok":true}`, string(result.Data))
	require.True(t, closed)
}

func openExecutorDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	return openDB(t, name)
}

func createExecutorTable(t *testing.T, db *sql.DB, table, value string) {
	t.Helper()

	_, err := db.Exec(`CREATE TABLE ` + table + ` (name TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO `+table+`(name) VALUES (?)`, value)
	require.NoError(t, err)
}
