package runner

import (
	"context"
	"testing"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/stretchr/testify/require"
)

type fakeRunner struct {
	kind string
}

func (f fakeRunner) Kind() string {
	return f.kind
}

func (f fakeRunner) Run(ctx context.Context, runCtx RunContext) (*model.OpResult, error) {
	return &model.OpResult{OpID: runCtx.Op.ID}, nil
}

func TestRegistryRejectsDuplicateKinds(t *testing.T) {
	registry := NewRegistry()

	require.NoError(t, registry.Register(fakeRunner{kind: "http/fetch"}))
	err := registry.Register(fakeRunner{kind: "http/fetch"})
	require.Error(t, err)
}

func TestRegistryReturnsSortedKinds(t *testing.T) {
	registry := NewRegistry()

	require.NoError(t, registry.Register(fakeRunner{kind: "js/analyze"}))
	require.NoError(t, registry.Register(fakeRunner{kind: "http/fetch"}))

	require.Equal(t, []string{"http/fetch", "js/analyze"}, registry.Kinds())
}
