package registry

import (
	"testing"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/stretchr/testify/require"
)

func TestRegisterRejectsEmptyName(t *testing.T) {
	registry := New()

	err := registry.Register(Definition{})
	require.Error(t, err)
}

func TestListReturnsSortedDefinitions(t *testing.T) {
	registry := New()

	require.NoError(t, registry.Register(Definition{Name: model.SiteName("nereval")}))
	require.NoError(t, registry.Register(Definition{Name: model.SiteName("zillow")}))

	definitions := registry.List()
	require.Len(t, definitions, 2)
	require.Equal(t, model.SiteName("nereval"), definitions[0].Name)
	require.Equal(t, model.SiteName("zillow"), definitions[1].Name)
}
