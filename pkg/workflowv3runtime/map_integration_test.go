package workflowv3runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/testfixtures/workflowv3map"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLazyMapScaleAcrossRestartWithDeterministicOutput(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.db")
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(root, "artifacts"), 16<<20)
	require.NoError(t, err)
	registry, err := workflowv3map.Registry()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	authored, err := workflowmodule.Author(
		ctx, workflowv3map.WorkflowSource(), catalog, workflowv3map.DescriptorModule(),
	)
	require.NoError(t, err)
	modules, err := NewTaskModuleRegistry(FSInputModule())
	require.NoError(t, err)

	itemCount := 1807
	completionTimeout := 120 * time.Second
	if raceDetectorEnabled {
		itemCount = 257
		completionTimeout = 90 * time.Second
	}
	const privateCanary = "PRIVATE-MAP-SOURCE-CANARY-DO-NOT-PERSIST"
	items := make([]workflowv3.ManifestItem, 0, itemCount)
	var sourceBytes int64
	for index := 0; index < itemCount; index++ {
		body, err := json.Marshal(map[string]any{
			"index":   index,
			"value":   fmt.Sprintf("record-%04d", index),
			"private": privateCanary + strings.Repeat("x", 4096),
		})
		require.NoError(t, err)
		ref, err := artifacts.Put(ctx, "map-record/v1", "application/json", body)
		require.NoError(t, err)
		sourceBytes += int64(len(body))
		items = append(items, workflowv3.ManifestItem{
			Key: fmt.Sprintf("record-%04d", index), Value: ref,
		})
	}
	manifest, err := workflowv3.NewItemManifest("map-record/v1", items)
	require.NoError(t, err)
	manifestBody, err := workflowv3.EncodeItemManifest(manifest)
	require.NoError(t, err)
	manifestRef, err := artifacts.Put(
		ctx, workflowv3.ItemManifestSchemaV1, "application/json", manifestBody,
	)
	require.NoError(t, err)

	store, err := workflowv3sqlite.Open(ctx, workflowPath)
	require.NoError(t, err)
	engine := &Engine{
		Store: store, Registry: registry, Artifacts: artifacts, Modules: modules,
		LeaseDuration: 2 * time.Second,
	}
	require.NoError(t, engine.Submit(ctx, "lazy-map-1807", authored.Plan, map[string]workflowv3.ArtifactRef{
		"records": manifestRef,
	}))
	for range 70 {
		ran, err := engine.RunOne(ctx)
		require.NoError(t, err)
		require.True(t, ran)
	}
	require.NoError(t, store.Close())

	store, err = workflowv3sqlite.Open(ctx, workflowPath)
	require.NoError(t, err)
	engine = &Engine{
		Store: store, Registry: registry, Artifacts: artifacts, Modules: modules,
		LeaseDuration: 2 * time.Second,
	}
	dispatcher := &Dispatcher{
		Engine: engine, Capacities: map[string]int{workflowv3map.ResourceClass: 8},
		PollInterval: 2 * time.Millisecond,
	}
	dispatchCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- dispatcher.Run(dispatchCtx) }()
	completed := assert.Eventually(t, func() bool {
		snapshot, snapshotErr := engine.Snapshot(ctx, "lazy-map-1807")
		return snapshotErr == nil && snapshot.Status == "succeeded"
	}, completionTimeout, 10*time.Millisecond)
	if !completed {
		snapshot, snapshotErr := engine.Snapshot(ctx, "lazy-map-1807")
		queue, queueErr := dispatcher.QueueSnapshot(ctx)
		t.Logf("lazy map timeout snapshot=%+v err=%v queue=%+v queueErr=%v", snapshot, snapshotErr, queue, queueErr)
	}
	require.True(t, completed)
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)

	snapshot, err := engine.Snapshot(ctx, "lazy-map-1807")
	require.NoError(t, err)
	require.Len(t, snapshot.Attempts, itemCount)
	outputRef, ok := snapshot.Outputs["records"]
	require.True(t, ok)
	outputBody, err := workflowv3.ReadArtifact(ctx, artifacts, outputRef)
	require.NoError(t, err)
	outputManifest, err := workflowv3.DecodeItemManifest(outputBody)
	require.NoError(t, err)
	require.Len(t, outputManifest.Items, itemCount)
	require.Equal(t, "record-0000", outputManifest.Items[0].Key)
	require.Equal(t, fmt.Sprintf("record-%04d", itemCount-1), outputManifest.Items[itemCount-1].Key)

	firstBody, err := workflowv3.ReadArtifact(ctx, artifacts, outputManifest.Items[0].Value)
	require.NoError(t, err)
	require.JSONEq(t, `{"index":0,"value":"RECORD-0000"}`, string(firstBody))
	require.NotContains(t, string(outputBody), privateCanary)
	require.NoError(t, store.Checkpoint(ctx))
	require.NoError(t, store.Close())
	persisted, persistedBytes := readSQLiteFiles(t, workflowPath)
	require.NotContains(t, string(persisted), privateCanary)
	t.Logf(
		"lazy map privacy/storage evidence: source=%d persistedSQLite=%d ratio=%.4f",
		sourceBytes, persistedBytes, float64(persistedBytes)/float64(sourceBytes),
	)
}
