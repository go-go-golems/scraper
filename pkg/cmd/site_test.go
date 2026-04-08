package cmd

import (
	"bytes"
	"database/sql"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"testing/fstest"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"os"
	sitemanifest "github.com/go-go-golems/scraper/pkg/sites/manifest"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	"github.com/go-go-golems/scraper/pkg/testfixtures"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
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
	rootCmd, err := NewRootCommand("test-version", testfixtures.SitesDir(t))
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
	rootCmd, err := NewRootCommand("test-version", testfixtures.SitesDir(t))
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
	rootCmd, err := NewRootCommand("test-version", testfixtures.SitesDir(t))
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
	rootCmd, err := NewRootCommand("test-version", testfixtures.SitesDir(t))
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
	rootCmd, err := NewRootCommand("test-version", testfixtures.SitesDir(t))
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
		rootCmd, err := NewRootCommand("test-version", testfixtures.SitesDir(t))
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
	def, err := sitemanifest.LoadDefinition(os.DirFS(filepath.Join(testfixtures.SitesDir(t), "jsdemo")), "")
	require.NoError(t, err)
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
	sitesDir := t.TempDir()
	engineDB := filepath.Join(t.TempDir(), "engine.db")
	fixture := testfixtures.ReadFixture(t, "hackernews", "frontpage.html")
	server := newStaticFixtureServer(t, fixture)

	submitOutput := runRootCommand(t, nil,
		"site", "hackernews", "run", "seed",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--workflow-id", "cmd-hackernews-seed",
		"--base-url", server.URL+"/",
		"--max-pages", "1",
	)
	require.Contains(t, submitOutput, "Site: hackernews")
	require.Contains(t, submitOutput, "Command: site hackernews run seed")
	require.Contains(t, submitOutput, "Submitted ops: 1")
	require.Contains(t, submitOutput, "Target op: cmd-hackernews-seed:seed:frontpage-extract")
	require.Contains(t, submitOutput, `"submittedEntrypoint": "seed"`)

	statusBefore := runRootCommand(t, nil, "engine", "status", "--engine-db", engineDB)
	require.Contains(t, statusBefore, "ready: 1")
	require.Contains(t, statusBefore, "succeeded: 0")

	workerOutput := runRootCommand(t, nil,
		"worker", "run",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--max-cycles", "16",
		"--poll-interval", "5ms",
	)
	require.Contains(t, workerOutput, "Succeeded:")

	statusAfter := runRootCommand(t, nil, "engine", "status", "--engine-db", engineDB)
	require.Contains(t, statusAfter, "ready: 0")
	require.Contains(t, statusAfter, "succeeded: 3")
	require.Contains(t, statusAfter, "Results: 3")

	siteDB, err := sql.Open("sqlite3", filepath.Join(sitesDir, "hackernews.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, siteDB.Close()) }()

	var stories int
	require.NoError(t, siteDB.QueryRow("SELECT COUNT(*) FROM stories").Scan(&stories))
	require.Equal(t, 2, stories)

	var topStoryID string
	require.NoError(t, siteDB.QueryRow("SELECT story_id FROM stories ORDER BY rank LIMIT 1").Scan(&topStoryID))
	require.Equal(t, "47490070", topStoryID)
}

func TestHackerNewsRunSeedCommandWithQueueRateLimit(t *testing.T) {
	pageOne := hackerNewsFixturePage(
		[]hackerNewsFixtureStory{
			{ID: "47490070", Rank: 1, Title: "First & Strong Story", StoryURL: "https://example.com/first-story", SiteName: "example.com", Score: 236, Author: "anemll", AgeText: "3 hours ago", CommentsText: "138 comments"},
			{ID: "47490080", Rank: 2, Title: "Ask HN: Building Smaller Scrapers?", StoryURL: "item?id=47490080", SiteName: "", Score: 87, Author: "somebody", AgeText: "2 hours ago", CommentsText: "discuss"},
		},
		"?p=2",
	)
	pageTwo := hackerNewsFixturePage(
		[]hackerNewsFixtureStory{
			{ID: "47490100", Rank: 31, Title: "A Deeper Page Story", StoryURL: "https://example.com/deeper-story", SiteName: "example.com", Score: 55, Author: "deeper", AgeText: "1 hour ago", CommentsText: "12 comments"},
			{ID: "47490110", Rank: 32, Title: "Ask HN: Page Two", StoryURL: "item?id=47490110", SiteName: "", Score: 21, Author: "pager", AgeText: "50 minutes ago", CommentsText: "5 comments"},
		},
		"",
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.RawQuery {
		case "":
			_, _ = w.Write([]byte(pageOne))
		case "p=2":
			_, _ = w.Write([]byte(pageTwo))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	sitesDir := t.TempDir()
	engineDB := filepath.Join(t.TempDir(), "engine.db")
	registry := siteregistry.New()
	def, err := sitemanifest.LoadDefinition(os.DirFS(filepath.Join(testfixtures.SitesDir(t), "hackernews")), "")
	require.NoError(t, err)
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

	submitOutput := runRootCommand(t, registry,
		"site", "hackernews", "run", "seed",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--workflow-id", "cmd-hackernews-rate",
		"--base-url", server.URL+"/",
		"--max-pages", "2",
	)
	require.Contains(t, submitOutput, "Submitted ops: 1")

	firstWorkerRun := runRootCommand(t, registry,
		"worker", "run",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--max-workers", "4",
		"--max-cycles", "3",
		"--poll-interval", "120ms",
	)
	require.Contains(t, firstWorkerRun, "Succeeded:")

	statusMid := runRootCommand(t, registry, "engine", "status", "--engine-db", engineDB)
	require.Contains(t, statusMid, "Workflows: 1")
	require.Contains(t, statusMid, "ready: 1")

	secondWorkerRun := runRootCommand(t, registry,
		"worker", "run",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--max-workers", "4",
		"--max-cycles", "8",
		"--poll-interval", "120ms",
	)
	require.Contains(t, secondWorkerRun, "Succeeded:")

	statusAfter := runRootCommand(t, registry, "engine", "status", "--engine-db", engineDB)
	require.Contains(t, statusAfter, "ready: 0")
	require.Contains(t, statusAfter, "succeeded: 5")
}

func TestHackerNewsRunSeedHelpIncludesMaxPages(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version", testfixtures.SitesDir(t))
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
	sitesDir := t.TempDir()
	engineDB := filepath.Join(t.TempDir(), "engine.db")
	fixture := testfixtures.ReadFixture(t, "hackernews", "frontpage.html")
	server := newStaticFixtureServer(t, fixture)

	submitOutput := runRootCommand(t, nil,
		"site", "hackernews", "run", "extract-frontpage",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--workflow-id", "cmd-hackernews-extract",
		"--base-url", server.URL+"/",
	)
	require.Contains(t, submitOutput, "Command: site hackernews run extract-frontpage")
	require.Contains(t, submitOutput, "Submitted ops: 2")
	require.Contains(t, submitOutput, "Target op: cmd-hackernews-extract:frontpage-extract")
	require.Contains(t, submitOutput, `"submittedEntrypoint": "extract-frontpage"`)

	workerOutput := runRootCommand(t, nil,
		"worker", "run",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--max-cycles", "12",
		"--poll-interval", "5ms",
	)
	require.Contains(t, workerOutput, "Succeeded:")

	statusAfter := runRootCommand(t, nil, "engine", "status", "--engine-db", engineDB)
	require.Contains(t, statusAfter, "succeeded: 2")

	siteDB, err := sql.Open("sqlite3", filepath.Join(sitesDir, "hackernews.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, siteDB.Close()) }()

	var stories int
	require.NoError(t, siteDB.QueryRow("SELECT COUNT(*) FROM stories").Scan(&stories))
	require.Equal(t, 2, stories)

	var storyID string
	require.NoError(t, siteDB.QueryRow("SELECT story_id FROM stories WHERE rank = 2").Scan(&storyID))
	require.Equal(t, "47490080", storyID)
}

func TestSlashdotRunSeedCommand(t *testing.T) {
	sitesDir := t.TempDir()
	engineDB := filepath.Join(t.TempDir(), "engine.db")
	fixture := testfixtures.ReadFixture(t, "slashdot", "frontpage.html")
	server := newStaticFixtureServer(t, fixture)

	submitOutput := runRootCommand(t, nil,
		"site", "slashdot", "run", "seed",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--workflow-id", "cmd-slashdot-seed",
		"--base-url", server.URL+"/",
		"--max-pages", "1",
	)
	require.Contains(t, submitOutput, "Site: slashdot")
	require.Contains(t, submitOutput, "Command: site slashdot run seed")
	require.Contains(t, submitOutput, "Submitted ops: 1")
	require.Contains(t, submitOutput, "Target op: cmd-slashdot-seed:seed:frontpage-extract")
	require.Contains(t, submitOutput, `"submittedEntrypoint": "seed"`)

	workerOutput := runRootCommand(t, nil,
		"worker", "run",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--max-cycles", "16",
		"--poll-interval", "5ms",
	)
	require.Contains(t, workerOutput, "Succeeded:")

	statusAfter := runRootCommand(t, nil, "engine", "status", "--engine-db", engineDB)
	require.Contains(t, statusAfter, "succeeded: 3")

	siteDB, err := sql.Open("sqlite3", filepath.Join(sitesDir, "slashdot.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, siteDB.Close()) }()

	var stories int
	require.NoError(t, siteDB.QueryRow("SELECT COUNT(*) FROM stories").Scan(&stories))
	require.Equal(t, 2, stories)

	var topStoryID string
	require.NoError(t, siteDB.QueryRow("SELECT story_id FROM stories ORDER BY position LIMIT 1").Scan(&topStoryID))
	require.Equal(t, "181087690", topStoryID)
}

func TestSlashdotRunSeedHelpIncludesMaxPages(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version", testfixtures.SitesDir(t))
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
	sitesDir := t.TempDir()
	engineDB := filepath.Join(t.TempDir(), "engine.db")
	fixture := testfixtures.ReadFixture(t, "slashdot", "frontpage.html")
	server := newStaticFixtureServer(t, fixture)

	submitOutput := runRootCommand(t, nil,
		"site", "slashdot", "run", "extract-frontpage",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--workflow-id", "cmd-slashdot-extract",
		"--base-url", server.URL+"/",
	)
	require.Contains(t, submitOutput, "Command: site slashdot run extract-frontpage")
	require.Contains(t, submitOutput, "Submitted ops: 2")
	require.Contains(t, submitOutput, "Target op: cmd-slashdot-extract:frontpage-extract")
	require.Contains(t, submitOutput, `"submittedEntrypoint": "extract-frontpage"`)

	workerOutput := runRootCommand(t, nil,
		"worker", "run",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--max-cycles", "12",
		"--poll-interval", "5ms",
	)
	require.Contains(t, workerOutput, "Succeeded:")

	statusAfter := runRootCommand(t, nil, "engine", "status", "--engine-db", engineDB)
	require.Contains(t, statusAfter, "succeeded: 2")

	siteDB, err := sql.Open("sqlite3", filepath.Join(sitesDir, "slashdot.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, siteDB.Close()) }()

	var stories int
	require.NoError(t, siteDB.QueryRow("SELECT COUNT(*) FROM stories").Scan(&stories))
	require.Equal(t, 2, stories)

	var storyID string
	require.NoError(t, siteDB.QueryRow("SELECT story_id FROM stories WHERE position = 2").Scan(&storyID))
	require.Equal(t, "181087016", storyID)
}

func TestNerevalRunSeedHelpIncludesFlags(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version", testfixtures.SitesDir(t))
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
		rootCmd, err := NewRootCommand("test-version", testfixtures.SitesDir(t))
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

	listPage1 := testfixtures.ReadFixture(t, "nereval", "list-page-1.html")
	listPage2 := testfixtures.ReadFixture(t, "nereval", "list-page-2.html")
	detail24038 := testfixtures.ReadFixture(t, "nereval", "detail-24038.html")
	detail24058 := testfixtures.ReadFixture(t, "nereval", "detail-24058.html")
	detail24100 := testfixtures.ReadFixture(t, "nereval", "detail-24100.html")

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

func runRootCommand(t *testing.T, registry *siteregistry.Registry, args ...string) string {
	t.Helper()

	var (
		rootCmd *cobra.Command
		err     error
	)
	if registry == nil {
		rootCmd, err = NewRootCommand("test-version", testfixtures.SitesDir(t))
	} else {
		rootCmd, err = newRootCommand("test-version", registry)
	}
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs(args)

	err = rootCmd.Execute()
	require.NoError(t, err)
	return stdout.String()
}

func newStaticFixtureServer(t *testing.T, body []byte) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	return server
}

type hackerNewsFixtureStory struct {
	ID           string
	Rank         int
	Title        string
	StoryURL     string
	SiteName     string
	Score        int
	Author       string
	AgeText      string
	CommentsText string
}

func hackerNewsFixturePage(stories []hackerNewsFixtureStory, nextHref string) string {
	page := "<html><body><table class=\"itemlist\">"
	for _, story := range stories {
		siteMarkup := ""
		if story.SiteName != "" {
			siteMarkup = "<span class=\"sitebit comhead\"> (<a href=\"from?site=" + story.SiteName + "\"><span class=\"sitestr\">" + story.SiteName + "</span></a>) </span>"
		}
		page += "<tr class=\"athing submission\" id=\"" + story.ID + "\">" +
			"<td class=\"title\"><span class=\"rank\">" + strconv.Itoa(story.Rank) + ".</span></td>" +
			"<td class=\"title\"><span class=\"titleline\"><a href=\"" + story.StoryURL + "\">" + story.Title + "</a>" + siteMarkup + "</span></td>" +
			"</tr>" +
			"<tr><td colspan=\"2\"></td><td class=\"subtext\">" +
			"<span class=\"score\" id=\"score_" + story.ID + "\">" + strconv.Itoa(story.Score) + " points</span> by " +
			"<a href=\"user?id=" + story.Author + "\" class=\"hnuser\">" + story.Author + "</a> " +
			"<span class=\"age\" title=\"2026-03-23T10:00:00 1742724000\"><a href=\"item?id=" + story.ID + "\">" + story.AgeText + "</a></span> " +
			"<a href=\"item?id=" + story.ID + "\">" + story.CommentsText + "</a>" +
			"</td></tr>"
	}
	page += "</table>"
	if nextHref != "" {
		page += "<a class=\"morelink\" href=\"" + nextHref + "\">More</a>"
	}
	page += "</body></html>"
	return page
}

var _ fs.FS = fstest.MapFS{}
