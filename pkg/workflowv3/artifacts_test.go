package workflowv3

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileArtifactStoreRoundTripAndIntegrity(t *testing.T) {
	ctx := context.Background()
	store, err := NewFileArtifactStore(t.TempDir(), 1024)
	require.NoError(t, err)

	ref, err := store.Put(ctx, "customer-jsonl-ref/v1", "application/x-ndjson", []byte("secret-canary\n"))
	require.NoError(t, err)
	require.NoError(t, ValidateArtifactRef(ref))
	body, err := ReadArtifact(ctx, store, ref)
	require.NoError(t, err)
	require.Equal(t, "secret-canary\n", string(body))

	objectPath := filepath.Join(store.root, filepath.FromSlash(ref.Locator))
	require.NoError(t, os.WriteFile(objectPath, []byte("tampered"), 0o644))
	_, err = store.Open(ctx, ref)
	require.ErrorContains(t, err, "size mismatch")
}

func TestFileArtifactStoreRejectsOversize(t *testing.T) {
	store, err := NewFileArtifactStore(t.TempDir(), 4)
	require.NoError(t, err)
	_, err = store.Put(context.Background(), "x/v1", "text/plain", []byte("12345"))
	require.ErrorContains(t, err, "exceeds")
}
