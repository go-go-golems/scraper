package cmd

import (
	"bytes"
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
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
	require.Contains(t, stdout.String(), "Entrypoint: seed")
	require.Contains(t, stdout.String(), "Status: succeeded")
	require.Contains(t, stdout.String(), `"itemCount": 3`)
	require.Contains(t, stdout.String(), `"totalBase": 24`)
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
	require.Contains(t, stdout.String(), "Entrypoint: item")
	require.Contains(t, stdout.String(), `"itemKey": "cmd-03"`)
	require.Contains(t, stdout.String(), `"baseValue": 12`)
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
	require.Contains(t, stdout.String(), "Entrypoint: summary")
	require.Contains(t, stdout.String(), `"itemCount": 3`)
	require.Contains(t, stdout.String(), `"totalSquared": 350`)
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

var _ fs.FS = fstest.MapFS{}
