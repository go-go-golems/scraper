package defaults

import (
	"path/filepath"
	"testing"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/stretchr/testify/require"
)

func TestNewRegistryFromDirsLoadsSites(t *testing.T) {
	// Resolve the repo's sites/ directory relative to this test file.
	// This test lives in pkg/sites/defaults/, so ../../../sites reaches scraper/sites/.
	sitesDir := filepath.Join("..", "..", "..", "sites")

	registry, err := NewRegistryFromDirs(sitesDir)
	require.NoError(t, err)

	_, ok := registry.Get(model.SiteName("hackernews"))
	require.True(t, ok, "hackernews should be in registry")

	_, ok = registry.Get(model.SiteName("slashdot"))
	require.True(t, ok, "slashdot should be in registry")

	_, ok = registry.Get(model.SiteName("js-demo"))
	require.True(t, ok, "js-demo should be in registry")

	_, ok = registry.Get(model.SiteName("nereval"))
	require.True(t, ok, "nereval should be in registry")
}

func TestNewRegistryFromDirsEmpty(t *testing.T) {
	registry, err := NewRegistryFromDirs()
	require.NoError(t, err)
	require.Empty(t, registry.List())
}
