package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/scraper/pkg/testfixtures"
	"github.com/stretchr/testify/require"
)

func TestParseBootstrapArgsRepeatedSitesManifestDir(t *testing.T) {
	first := filepath.Join(t.TempDir(), "sites-a")
	second := filepath.Join(t.TempDir(), "sites-b")

	options, err := ParseBootstrapArgs([]string{
		"--sites-manifest-dir", first,
		"worker", "run",
		"--sites-manifest-dir", second,
	})
	require.NoError(t, err)
	require.Equal(t, []string{first, second}, options.SitesManifestDirs)
}

func TestParseBootstrapArgsIgnoresUnknownFlags(t *testing.T) {
	sitesDir := filepath.Join(t.TempDir(), "sites")

	options, err := ParseBootstrapArgs([]string{
		"--sites-manifest-dir", sitesDir,
		"site", "js-demo", "run", "seed",
		"--engine-db", "state/engine.db",
		"--count", "3",
	})
	require.NoError(t, err)
	require.Equal(t, []string{sitesDir}, options.SitesManifestDirs)
}

func TestNewRootCommandFromBootstrapBuildsSiteVerbCommandsFromFlag(t *testing.T) {
	sitesDir := testfixtures.SitesDir(t)

	rootCmd, err := NewRootCommandFromBootstrap("test-version", []string{
		"--sites-manifest-dir", sitesDir,
		"site", "js-demo", "run", "seed",
	})
	require.NoError(t, err)

	cmd, _, err := rootCmd.Find([]string{"site", "js-demo", "run", "seed"})
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "seed", cmd.Name())
}

func TestCollectSitesManifestDirsMergesConfigEnvAndBootstrapFlag(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "config-sites")
	envDir := filepath.Join(t.TempDir(), "env-sites")
	bootstrapDir := filepath.Join(t.TempDir(), "bootstrap-sites")

	homeDir := t.TempDir()
	appConfigDir := filepath.Join(homeDir, ".scraper")
	err := os.MkdirAll(appConfigDir, 0o755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(appConfigDir, "config.yaml"), []byte("sitesManifestDirs:\n  - "+configDir+"\n"), 0o644)
	require.NoError(t, err)

	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	t.Setenv(scraperSitesManifestDirsEnvVar, envDir)

	dirs, err := CollectSitesManifestDirs("scraper", []string{
		"--sites-manifest-dir", bootstrapDir,
		"site", "js-demo", "run", "seed",
	})
	require.NoError(t, err)
	require.Equal(t, []string{configDir, envDir, bootstrapDir}, dirs)
}
