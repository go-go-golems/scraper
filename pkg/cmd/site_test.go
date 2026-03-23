package cmd

import (
	"bytes"
	"database/sql"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	hackernews "github.com/go-go-golems/scraper/pkg/sites/hackernews"
	"github.com/go-go-golems/scraper/pkg/sites/jsdemo"
	"github.com/go-go-golems/scraper/pkg/sites/nereval"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestSiteMigrateCommand(t *testing.T) {
	registry := siteregistry.New()
	require.NoError(t, registry.Register(siteregistry.Definition{
		Name: model.SiteName("demo"),
		SQLMigrationsFS: fstest.MapFS{
			"001_init.sql": &fstest.MapFile{Data: []byte(`CREATE TABLE widgets(id INTEGER PRIMARY KEY);`)},
		},
	}))

	rootCmd, err := newRootCommand("test-version", registry)
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"site", "migrate", "demo", "--sites-dir", t.TempDir()})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Site: demo")
	require.Contains(t, stdout.String(), "Applied migrations: 1")
	require.Contains(t, stdout.String(), "Current schema version: 1")
}

func TestSiteMigrateUnknownSite(t *testing.T) {
	rootCmd, err := newRootCommand("test-version", siteregistry.New())
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"site", "migrate", "missing", "--sites-dir", t.TempDir()})

	err = rootCmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), `site "missing" is not registered`)
}

func TestRootCommandIncludesBuiltinSites(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"site", "migrate", "hackernews", "--sites-dir", t.TempDir()})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Site: hackernews")
	require.Contains(t, stdout.String(), "Current schema version: 1")
}

func TestJSDemoRunSeedCommand(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{
		"site", "js-demo", "run", "seed",
		"--sites-dir", t.TempDir(),
		"--engine-db", filepath.Join(t.TempDir(), "engine.db"),
		"--workflow-id", "cmd-js-demo",
		"--count", "3",
		"--multiplier", "4",
		"--prefix", "cmd",
	})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Site: js-demo")
	require.Contains(t, stdout.String(), "Command: site js-demo run seed")
	require.Contains(t, stdout.String(), "Workflow: cmd-js-demo")
	require.Contains(t, stdout.String(), "Submitted ops: 1")
	require.Contains(t, stdout.String(), "Target op: cmd-js-demo:seed:summary")
	require.Contains(t, stdout.String(), `"submittedEntrypoint": "seed"`)
}

func TestJSDemoRunItemCommand(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{
		"site", "js-demo", "run", "item",
		"--sites-dir", t.TempDir(),
		"--engine-db", filepath.Join(t.TempDir(), "engine.db"),
		"--workflow-id", "cmd-js-demo-item",
		"--index", "2",
		"--multiplier", "4",
		"--prefix", "cmd",
	})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Command: site js-demo run item")
	require.Contains(t, stdout.String(), "Submitted ops: 1")
	require.Contains(t, stdout.String(), "Target op: cmd-js-demo-item:item:03")
	require.Contains(t, stdout.String(), `"submittedEntrypoint": "item"`)
}

func TestJSDemoRunSummaryCommand(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{
		"site", "js-demo", "run", "summary",
		"--sites-dir", t.TempDir(),
		"--engine-db", filepath.Join(t.TempDir(), "engine.db"),
		"--workflow-id", "cmd-js-demo-summary",
		"--count", "3",
		"--multiplier", "5",
		"--prefix", "sum",
	})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Command: site js-demo run summary")
	require.Contains(t, stdout.String(), "Submitted ops: 4")
	require.Contains(t, stdout.String(), "Target op: cmd-js-demo-summary:summary")
	require.Contains(t, stdout.String(), `"submittedEntrypoint": "summary"`)
}

func TestJSDemoRunSeedHelpIncludesJSFlags(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"site", "js-demo", "run", "seed", "--help"})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "--count")
	require.Contains(t, stdout.String(), "--multiplier")
	require.Contains(t, stdout.String(), "--prefix")
}

func TestJSDemoSubmitThenWorkerRun(t *testing.T) {
	sitesDir := t.TempDir()
	engineDB := filepath.Join(t.TempDir(), "engine.db")

	runCommand := func(args ...string) string {
		rootCmd, err := NewRootCommand("test-version")
		require.NoError(t, err)

		var stdout bytes.Buffer
		rootCmd.SetOut(&stdout)
		rootCmd.SetErr(&stdout)
		rootCmd.SetArgs(args)

		err = rootCmd.Execute()
		require.NoError(t, err)
		return stdout.String()
	}

	submitOutput := runCommand(
		"site", "js-demo", "run", "seed",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--workflow-id", "cmd-js-demo-worker",
		"--count", "3",
		"--multiplier", "4",
		"--prefix", "cmd",
	)
	require.Contains(t, submitOutput, "Submitted ops: 1")
	require.Contains(t, submitOutput, "Target op: cmd-js-demo-worker:seed:summary")

	statusBefore := runCommand("engine", "status", "--engine-db", engineDB)
	require.Contains(t, statusBefore, "Workflows: 1")
	require.Contains(t, statusBefore, "ready: 1")
	require.Contains(t, statusBefore, "succeeded: 0")

	workerOutput := runCommand(
		"worker", "run",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--max-cycles", "16",
		"--poll-interval", "5ms",
	)
	require.Contains(t, workerOutput, "Cycles:")
	require.Contains(t, workerOutput, "Succeeded:")

	statusAfter := runCommand("engine", "status", "--engine-db", engineDB)
	require.Contains(t, statusAfter, "Workflows: 1")
	require.Contains(t, statusAfter, "ready: 0")
	require.Contains(t, statusAfter, "succeeded: 5")
	require.Contains(t, statusAfter, "Results: 5")
}

func TestJSDemoSubmitThenWorkerRunWithQueueRateLimit(t *testing.T) {
	sitesDir := t.TempDir()
	engineDB := filepath.Join(t.TempDir(), "engine.db")

	registry := siteregistry.New()
	def := jsdemo.Definition()
	def.QueuePolicies = map[model.QueueKey]model.QueuePolicy{
		model.QueueKey("site:js-demo:js"): {
			MaxInFlight: 4,
			RateLimit: &model.RateLimitPolicy{
				Kind:          model.RateLimitKindTokenBucket,
				RatePerSecond: 10,
				Burst:         1,
			},
		},
	}
	require.NoError(t, registry.Register(def))

	runCommand := func(args ...string) string {
		rootCmd, err := newRootCommand("test-version", registry)
		require.NoError(t, err)

		var stdout bytes.Buffer
		rootCmd.SetOut(&stdout)
		rootCmd.SetErr(&stdout)
		rootCmd.SetArgs(args)

		err = rootCmd.Execute()
		require.NoError(t, err)
		return stdout.String()
	}

	submitOutput := runCommand(
		"site", "js-demo", "run", "seed",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--workflow-id", "cmd-js-demo-rate",
		"--count", "3",
		"--multiplier", "4",
		"--prefix", "rate",
	)
	require.Contains(t, submitOutput, "Submitted ops: 1")
	require.Contains(t, submitOutput, "Target op: cmd-js-demo-rate:seed:summary")

	statusBefore := runCommand("engine", "status", "--engine-db", engineDB)
	require.Contains(t, statusBefore, "ready: 1")
	require.Contains(t, statusBefore, "succeeded: 0")

	firstWorkerRun := runCommand(
		"worker", "run",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--max-workers", "4",
		"--max-cycles", "2",
		"--poll-interval", "120ms",
	)
	require.Contains(t, firstWorkerRun, "Processed:")
	require.Contains(t, firstWorkerRun, "Succeeded:")

	statusMid := runCommand("engine", "status", "--engine-db", engineDB)
	require.Contains(t, statusMid, "Workflows: 1")
	require.Contains(t, statusMid, "succeeded: 2")
	require.Contains(t, statusMid, "ready: 2")

	secondWorkerRun := runCommand(
		"worker", "run",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--max-workers", "4",
		"--max-cycles", "6",
		"--poll-interval", "120ms",
	)
	require.Contains(t, secondWorkerRun, "Succeeded:")

	statusAfter := runCommand("engine", "status", "--engine-db", engineDB)
	require.Contains(t, statusAfter, "ready: 0")
	require.Contains(t, statusAfter, "succeeded: 5")
	require.Contains(t, statusAfter, "Results: 5")
}

func TestHackerNewsRunSeedCommand(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{
		"site", "hackernews", "run", "seed",
		"--fixture",
		"--sites-dir", t.TempDir(),
		"--engine-db", filepath.Join(t.TempDir(), "engine.db"),
		"--workflow-id", "cmd-hackernews-seed",
		"--max-pages", "2",
	})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Site: hackernews")
	require.Contains(t, stdout.String(), "Entrypoint: seed")
	require.Contains(t, stdout.String(), "Status: succeeded")
	require.Contains(t, stdout.String(), "Fixture: true")
	require.Contains(t, stdout.String(), `"storyCount": 2`)
	require.Contains(t, stdout.String(), `"47490070"`)
}

func TestHackerNewsRunSeedCommandWithQueueRateLimit(t *testing.T) {
	registry := siteregistry.New()
	def := hackernews.Definition()
	def.QueuePolicies = map[model.QueueKey]model.QueuePolicy{
		model.QueueKey("site:hackernews:http"): {
			MaxInFlight: 4,
			RateLimit: &model.RateLimitPolicy{
				Kind:          model.RateLimitKindTokenBucket,
				RatePerSecond: 10,
				Burst:         1,
			},
		},
	}
	require.NoError(t, registry.Register(def))

	rootCmd, err := newRootCommand("test-version", registry)
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{
		"site", "hackernews", "run", "seed",
		"--fixture",
		"--sites-dir", t.TempDir(),
		"--engine-db", filepath.Join(t.TempDir(), "engine.db"),
		"--workflow-id", "cmd-hackernews-rate",
		"--max-pages", "2",
	})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Site: hackernews")
	require.Contains(t, stdout.String(), "Entrypoint: seed")
	require.Contains(t, stdout.String(), "Status: succeeded")
	require.Contains(t, stdout.String(), `"storyCount": 2`)
}

func TestHackerNewsRunSeedHelpIncludesMaxPages(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"site", "hackernews", "run", "seed", "--help"})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "--max-pages")
}

func TestHackerNewsRunExtractFrontpageCommand(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{
		"site", "hackernews", "run", "extract-frontpage",
		"--fixture",
		"--sites-dir", t.TempDir(),
		"--engine-db", filepath.Join(t.TempDir(), "engine.db"),
		"--workflow-id", "cmd-hackernews-extract",
	})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Entrypoint: extract-frontpage")
	require.Contains(t, stdout.String(), `"storyCount": 2`)
	require.Contains(t, stdout.String(), `"47490080"`)
}

func TestSlashdotRunSeedCommand(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{
		"site", "slashdot", "run", "seed",
		"--fixture",
		"--sites-dir", t.TempDir(),
		"--engine-db", filepath.Join(t.TempDir(), "engine.db"),
		"--workflow-id", "cmd-slashdot-seed",
		"--max-pages", "2",
	})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Site: slashdot")
	require.Contains(t, stdout.String(), "Entrypoint: seed")
	require.Contains(t, stdout.String(), "Status: succeeded")
	require.Contains(t, stdout.String(), "Fixture: true")
	require.Contains(t, stdout.String(), `"storyCount": 2`)
	require.Contains(t, stdout.String(), `"181087690"`)
}

func TestSlashdotRunSeedHelpIncludesMaxPages(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"site", "slashdot", "run", "seed", "--help"})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "--max-pages")
}

func TestSlashdotRunExtractFrontpageCommand(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{
		"site", "slashdot", "run", "extract-frontpage",
		"--fixture",
		"--sites-dir", t.TempDir(),
		"--engine-db", filepath.Join(t.TempDir(), "engine.db"),
		"--workflow-id", "cmd-slashdot-extract",
	})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Entrypoint: extract-frontpage")
	require.Contains(t, stdout.String(), `"storyCount": 2`)
	require.Contains(t, stdout.String(), `"181087016"`)
}

func TestNerevalRunSeedHelpIncludesFlags(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"site", "nereval", "run", "seed", "--help"})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "--town")
	require.Contains(t, stdout.String(), "--base-url")
	require.Contains(t, stdout.String(), "--max-pages")
}

func TestNerevalSubmitThenWorkerRun(t *testing.T) {
	sitesDir := t.TempDir()
	engineDB := filepath.Join(t.TempDir(), "engine.db")
	server := newNerevalFixtureServer(t)

	runCommand := func(args ...string) string {
		rootCmd, err := NewRootCommand("test-version")
		require.NoError(t, err)

		var stdout bytes.Buffer
		rootCmd.SetOut(&stdout)
		rootCmd.SetErr(&stdout)
		rootCmd.SetArgs(args)

		err = rootCmd.Execute()
		require.NoError(t, err)
		return stdout.String()
	}

	submitOutput := runCommand(
		"site", "nereval", "run", "seed",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--workflow-id", "cmd-nereval",
		"--town", "Providence",
		"--base-url", server.URL,
		"--max-pages", "2",
	)
	require.Contains(t, submitOutput, "Site: nereval")
	require.Contains(t, submitOutput, "Command: site nereval run seed")
	require.Contains(t, submitOutput, "Submitted ops: 1")
	require.Contains(t, submitOutput, `"submittedEntrypoint": "seed"`)

	statusBefore := runCommand("engine", "status", "--engine-db", engineDB)
	require.Contains(t, statusBefore, "ready: 1")
	require.Contains(t, statusBefore, "succeeded: 0")

	workerOutput := runCommand(
		"worker", "run",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--max-cycles", "24",
		"--poll-interval", "5ms",
	)
	require.Contains(t, workerOutput, "Processed:")
	require.Contains(t, workerOutput, "Succeeded:")

	statusAfter := runCommand("engine", "status", "--engine-db", engineDB)
	require.Contains(t, statusAfter, "ready: 0")
	require.Contains(t, statusAfter, "succeeded: 11")
	require.Contains(t, statusAfter, "Results: 11")
	require.Contains(t, statusAfter, "Artifacts: 5")

	siteDB, err := sql.Open("sqlite3", filepath.Join(sitesDir, "nereval.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, siteDB.Close()) }()

	var properties int
	require.NoError(t, siteDB.QueryRow("SELECT COUNT(*) FROM properties").Scan(&properties))
	require.Equal(t, 3, properties)

	var owners int
	require.NoError(t, siteDB.QueryRow("SELECT COUNT(*) FROM owners").Scan(&owners))
	require.Equal(t, 4, owners)

	var assessments int
	require.NoError(t, siteDB.QueryRow("SELECT COUNT(*) FROM assessments").Scan(&assessments))
	require.Equal(t, 3, assessments)

	var sales int
	require.NoError(t, siteDB.QueryRow("SELECT COUNT(*) FROM sales").Scan(&sales))
	require.Equal(t, 3, sales)

	var parcelTotal string
	require.NoError(t, siteDB.QueryRow("SELECT parcel_total FROM assessments WHERE account_number = '24038'").Scan(&parcelTotal))
	require.Equal(t, "$650,000", parcelTotal)
}

func newNerevalFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()

	listPage1, err := nereval.ReadFixture("list-page-1.html")
	require.NoError(t, err)
	listPage2, err := nereval.ReadFixture("list-page-2.html")
	require.NoError(t, err)
	detail24038, err := nereval.ReadFixture("detail-24038.html")
	require.NoError(t, err)
	detail24058, err := nereval.ReadFixture("detail-24058.html")
	require.NoError(t, err)
	detail24100, err := nereval.ReadFixture("detail-24100.html")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/PropertyList.aspx":
			_, _ = w.Write(listPage1)
		case r.Method == http.MethodPost && r.URL.Path == "/PropertyList.aspx":
			require.NoError(t, r.ParseForm())
			if r.Form.Get("__VIEWSTATE") == "vs-page-1" && r.Form.Get("__EVENTVALIDATION") == "ev-page-1" {
				_, _ = w.Write(listPage2)
				return
			}
			http.Error(w, "unexpected form state", http.StatusBadRequest)
		case r.Method == http.MethodGet && r.URL.Path == "/PropertyDetail.aspx":
			account := r.URL.Query().Get("accountnumber")
			switch account {
			case "24038":
				_, _ = w.Write(detail24038)
			case "24058":
				_, _ = w.Write(detail24058)
			case "24100":
				_, _ = w.Write(detail24100)
			default:
				http.Error(w, "missing detail fixture", http.StatusNotFound)
			}
		default:
			http.Error(w, "unexpected request "+r.Method+" "+r.URL.String(), http.StatusNotFound)
		}
	}))

	t.Cleanup(server.Close)
	return server
}

var _ fs.FS = fstest.MapFS{}
