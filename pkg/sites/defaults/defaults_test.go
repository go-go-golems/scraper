package defaults

import (
	"testing"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/stretchr/testify/require"
)

func TestNewRegistryIncludesBuiltinSites(t *testing.T) {
	registry, err := NewRegistry()
	require.NoError(t, err)

	_, ok := registry.Get(model.SiteName("hackernews"))
	require.True(t, ok)

	_, ok = registry.Get(model.SiteName("slashdot"))
	require.True(t, ok)
}
