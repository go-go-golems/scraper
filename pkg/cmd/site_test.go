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

func TestJSDemoRunCommand(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{
		"site", "js-demo", "run",
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
	require.Contains(t, stdout.String(), "Status: succeeded")
	require.Contains(t, stdout.String(), `"itemCount": 3`)
	require.Contains(t, stdout.String(), `"totalBase": 24`)
}

var _ fs.FS = fstest.MapFS{}
