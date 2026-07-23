package workflowv3sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/stretchr/testify/require"
)

func TestRenewLeaseExtendsOnlyCurrentUnexpiredAuthority(t *testing.T) {
	ctx := context.Background()
	registry, plan := storeFixture(t, "lease-renewal")
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	require.NoError(t, store.CreateRun(ctx, "lease-renewal", plan, map[string]workflowv3.ArtifactRef{"source": artifactRef("source/v1", "lease-renewal")}, now))
	lease, err := store.LeaseNext(ctx, registry, now, 100*time.Millisecond)
	require.NoError(t, err)
	renewed, err := store.RenewLease(ctx, *lease, now.Add(50*time.Millisecond), now.Add(250*time.Millisecond))
	require.NoError(t, err)
	require.True(t, renewed)
	valid, err := store.LeaseValid(ctx, *lease, now.Add(150*time.Millisecond))
	require.NoError(t, err)
	require.True(t, valid)
	renewed, err = store.RenewLease(ctx, *lease, now.Add(300*time.Millisecond), now.Add(400*time.Millisecond))
	require.NoError(t, err)
	require.False(t, renewed)
}

func TestRenewLeaseRejectsCanceledRun(t *testing.T) {
	ctx := context.Background()
	registry, plan := storeFixture(t, "lease-canceled")
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	require.NoError(t, store.CreateRun(ctx, "lease-canceled", plan, map[string]workflowv3.ArtifactRef{"source": artifactRef("source/v1", "lease-canceled")}, now))
	lease, err := store.LeaseNext(ctx, registry, now, time.Second)
	require.NoError(t, err)
	require.NoError(t, store.Cancel(ctx, "lease-canceled", now.Add(time.Millisecond)))
	renewed, err := store.RenewLease(ctx, *lease, now.Add(2*time.Millisecond), now.Add(time.Second))
	require.NoError(t, err)
	require.False(t, renewed)
}
