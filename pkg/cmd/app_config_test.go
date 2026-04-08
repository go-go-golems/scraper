package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadAppConfigFromPathEmptyPath(t *testing.T) {
	cfg, err := loadAppConfigFromPath("")
	require.NoError(t, err)
	require.Empty(t, cfg.SitesManifestDirs)
}

func TestLoadAppConfigFromPathYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte(`sitesManifestDirs:
  - /tmp/sites-a
  - " /tmp/sites-b "
  - /tmp/sites-a
  - "$HOME/sites-c"
  - ""
`), 0o644)
	require.NoError(t, err)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	cfg, err := loadAppConfigFromPath(configPath)
	require.NoError(t, err)
	require.Equal(t, []string{"/tmp/sites-a", "/tmp/sites-b", filepath.Join(homeDir, "sites-c")}, cfg.SitesManifestDirs)
}

func TestSitesManifestDirsFromEnv(t *testing.T) {
	first := filepath.Join(t.TempDir(), "sites-a")
	second := filepath.Join(t.TempDir(), "sites-b")

	t.Setenv(scraperSitesManifestDirsEnvVar, first+string(os.PathListSeparator)+second+string(os.PathListSeparator)+first)

	require.Equal(t, []string{first, second}, sitesManifestDirsFromEnv())
}

func TestCollectSitesManifestDirsMergesConfigEnvAndBootstrap(t *testing.T) {
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

	dirs, err := collectSitesManifestDirs("scraper", []string{bootstrapDir, envDir})
	require.NoError(t, err)
	require.Equal(t, []string{configDir, envDir, bootstrapDir}, dirs)
}
