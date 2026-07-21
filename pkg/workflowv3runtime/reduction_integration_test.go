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
	"github.com/go-go-golems/scraper/pkg/testfixtures/workflowv3reduce"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3sqlite"
	"github.com/stretchr/testify/require"
)

func TestReductionRootDigestIsIndependentOfConcurrency(t *testing.T) {
	serial := runReductionDigest(t, 17, 1, 1)
	concurrent := runReductionDigest(t, 17, 4, 4)
	require.Equal(t, serial, concurrent)
	require.NotEmpty(t, runReductionDigest(t, 1, 1, 1))
}

func runReductionDigest(t *testing.T, itemCount, mapCapacity, reduceCapacity int) string {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(root, "artifacts"), 1<<20)
	require.NoError(t, err)
	registry, err := workflowv3reduce.Registry()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	authored, err := workflowmodule.Author(ctx, workflowv3reduce.WorkflowSource(), catalog, workflowv3reduce.DescriptorModule())
	require.NoError(t, err)
	items := make([]workflowv3.ManifestItem, 0, itemCount)
	for index := 0; index < itemCount; index++ {
		body, err := json.Marshal(map[string]any{"text": fmt.Sprintf("common item-%02d", index)})
		require.NoError(t, err)
		ref, err := artifacts.Put(ctx, "word-document/v1", "application/json", body)
		require.NoError(t, err)
		items = append(items, workflowv3.ManifestItem{Key: fmt.Sprintf("document-%04d", index), Value: ref})
	}
	manifest, err := workflowv3.NewItemManifest("word-document/v1", items)
	require.NoError(t, err)
	body, err := workflowv3.EncodeItemManifest(manifest)
	require.NoError(t, err)
	manifestRef, err := artifacts.Put(ctx, workflowv3.ItemManifestSchemaV1, "application/json", body)
	require.NoError(t, err)
	store, err := workflowv3sqlite.Open(ctx, filepath.Join(root, "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	modules, err := NewTaskModuleRegistry(FSInputModule())
	require.NoError(t, err)
	engine := &Engine{Store: store, Registry: registry, Artifacts: artifacts, Modules: modules}
	require.NoError(t, engine.Submit(ctx, "digest-reduction", authored.Plan, map[string]workflowv3.ArtifactRef{"documents": manifestRef}))
	dispatcher := &Dispatcher{Engine: engine, Capacities: map[string]int{
		workflowv3reduce.MapResource: mapCapacity, workflowv3reduce.ReduceResource: reduceCapacity,
	}, PollInterval: time.Millisecond}
	dispatchCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- dispatcher.Run(dispatchCtx) }()
	require.Eventually(t, func() bool {
		snapshot, snapshotErr := engine.Snapshot(ctx, "digest-reduction")
		return snapshotErr == nil && snapshot.Status == "succeeded"
	}, 30*time.Second, 5*time.Millisecond)
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	snapshot, err := engine.Snapshot(ctx, "digest-reduction")
	require.NoError(t, err)
	return snapshot.Outputs["count"].Digest
}

func TestEmptyReductionFailsWithoutWorkerLease(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(root, "artifacts"), 1<<20)
	require.NoError(t, err)
	registry, err := workflowv3reduce.Registry()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	authored, err := workflowmodule.Author(ctx, workflowv3reduce.WorkflowSource(), catalog, workflowv3reduce.DescriptorModule())
	require.NoError(t, err)
	manifest, err := workflowv3.NewItemManifest("word-document/v1", []workflowv3.ManifestItem{})
	require.NoError(t, err)
	body, err := workflowv3.EncodeItemManifest(manifest)
	require.NoError(t, err)
	manifestRef, err := artifacts.Put(ctx, workflowv3.ItemManifestSchemaV1, "application/json", body)
	require.NoError(t, err)
	store, err := workflowv3sqlite.Open(ctx, filepath.Join(root, "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	modules, err := NewTaskModuleRegistry(FSInputModule())
	require.NoError(t, err)
	engine := &Engine{Store: store, Registry: registry, Artifacts: artifacts, Modules: modules}
	require.NoError(t, engine.Submit(ctx, "empty-reduction", authored.Plan, map[string]workflowv3.ArtifactRef{"documents": manifestRef}))
	require.NoError(t, engine.RunUntilIdle(ctx))
	snapshot, err := engine.Snapshot(ctx, "empty-reduction")
	require.NoError(t, err)
	require.Equal(t, "failed", snapshot.Status)
	require.Empty(t, snapshot.Attempts)
}

func TestReductionLevelMaterializationIsIdempotentAcrossConnections(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	path := filepath.Join(root, "workflow.db")
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(root, "artifacts"), 1<<20)
	require.NoError(t, err)
	registry, err := workflowv3reduce.Registry()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	plan, err := workflowv3.Compile(workflowv3.WorkflowIR{
		Schema: workflowv3.IRSchema, Name: "direct-reduction",
		Inputs: []workflowv3.IRInput{}, Nodes: []workflowv3.IRNode{},
		SetInputs: []workflowv3.IRSetInput{{Name: "counts", ItemSchema: "word-count/v1", ManifestSchema: workflowv3.ItemManifestSchemaV1}},
		Reductions: []workflowv3.IRReduce{{
			Key: "merge-counts", Source: workflowv3.SetRef{Source: "set-input", Name: "counts", ItemSchema: "word-count/v1", ManifestSchema: workflowv3.ItemManifestSchemaV1},
			PartitionTask: workflowv3.TaskKey{Kind: "cookbook.word-count.merge", Version: "v1"},
			Bindings:      map[string]workflowv3.ValueRef{"partition": {Source: "reduction-partition", ReduceKey: "merge-counts", Schema: workflowv3.ReductionPartitionSchemaV1}},
			Policy:        workflowv3.ReducePolicy{FanIn: 8, MaxLevels: 4},
		}},
		Outputs: []workflowv3.IROutput{{Name: "count", Value: workflowv3.ValueRef{Source: "reduction-output", ReduceKey: "merge-counts", Schema: "word-count/v1"}}},
	}, catalog)
	require.NoError(t, err)
	items := make([]workflowv3.ManifestItem, 0, 9)
	for index := 0; index < 9; index++ {
		body, err := json.Marshal(map[string]any{"counts": map[string]int{fmt.Sprintf("word-%02d", index): 1}})
		require.NoError(t, err)
		ref, err := artifacts.Put(ctx, "word-count/v1", "application/json", body)
		require.NoError(t, err)
		items = append(items, workflowv3.ManifestItem{Key: fmt.Sprintf("count-%04d", index), Value: ref})
	}
	manifest, err := workflowv3.NewItemManifest("word-count/v1", items)
	require.NoError(t, err)
	body, err := workflowv3.EncodeItemManifest(manifest)
	require.NoError(t, err)
	manifestRef, err := artifacts.Put(ctx, workflowv3.ItemManifestSchemaV1, "application/json", body)
	require.NoError(t, err)
	firstStore, err := workflowv3sqlite.Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, firstStore.Close()) }()
	modules, err := NewTaskModuleRegistry(FSInputModule())
	require.NoError(t, err)
	firstEngine := &Engine{Store: firstStore, Registry: registry, Artifacts: artifacts, Modules: modules}
	require.NoError(t, firstEngine.Submit(ctx, "reduction-race", plan, map[string]workflowv3.ArtifactRef{"counts": manifestRef}))
	secondStore, err := workflowv3sqlite.Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, secondStore.Close()) }()
	secondEngine := &Engine{Store: secondStore, Registry: registry, Artifacts: artifacts, Modules: modules}
	start := make(chan struct{})
	errors := make(chan error, 2)
	for _, engine := range []*Engine{firstEngine, secondEngine} {
		engine := engine
		go func() {
			<-start
			_, reduceErr := engine.ReduceOne(ctx)
			errors <- reduceErr
		}()
	}
	close(start)
	require.NoError(t, <-errors)
	require.NoError(t, <-errors)
	queue, err := firstStore.QueueSnapshot(ctx, registry, map[string]int{
		workflowv3reduce.ReduceResource: 2,
	}, time.Now().UTC())
	require.NoError(t, err)
	require.Len(t, queue.Reductions, 1)
	require.Equal(t, 2, queue.Reductions[0].PartitionsTotal)
}

func TestReductionFailureIsIsolatedFromAnotherRun(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(root, "artifacts"), 1<<20)
	require.NoError(t, err)
	registry, err := workflowv3reduce.Registry()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	authored, err := workflowmodule.Author(ctx, workflowv3reduce.WorkflowSource(), catalog, workflowv3reduce.DescriptorModule())
	require.NoError(t, err)
	makeManifest := func(fail bool) workflowv3.ArtifactRef {
		items := make([]workflowv3.ManifestItem, 0, 9)
		for index := 0; index < 9; index++ {
			text := fmt.Sprintf("ok item-%02d", index)
			if fail && index == 4 {
				text = "ok __fail__"
			}
			body, marshalErr := json.Marshal(map[string]any{"text": text})
			require.NoError(t, marshalErr)
			ref, putErr := artifacts.Put(ctx, "word-document/v1", "application/json", body)
			require.NoError(t, putErr)
			items = append(items, workflowv3.ManifestItem{Key: fmt.Sprintf("document-%04d", index), Value: ref})
		}
		manifest, manifestErr := workflowv3.NewItemManifest("word-document/v1", items)
		require.NoError(t, manifestErr)
		body, encodeErr := workflowv3.EncodeItemManifest(manifest)
		require.NoError(t, encodeErr)
		ref, putErr := artifacts.Put(ctx, workflowv3.ItemManifestSchemaV1, "application/json", body)
		require.NoError(t, putErr)
		return ref
	}
	store, err := workflowv3sqlite.Open(ctx, filepath.Join(root, "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	modules, err := NewTaskModuleRegistry(FSInputModule())
	require.NoError(t, err)
	engine := &Engine{Store: store, Registry: registry, Artifacts: artifacts, Modules: modules}
	require.NoError(t, engine.Submit(ctx, "bad-reduction", authored.Plan, map[string]workflowv3.ArtifactRef{"documents": makeManifest(true)}))
	require.NoError(t, engine.Submit(ctx, "good-reduction", authored.Plan, map[string]workflowv3.ArtifactRef{"documents": makeManifest(false)}))
	dispatcher := &Dispatcher{Engine: engine, Capacities: map[string]int{
		workflowv3reduce.MapResource: 4, workflowv3reduce.ReduceResource: 2,
	}, PollInterval: time.Millisecond}
	dispatchCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- dispatcher.Run(dispatchCtx) }()
	require.Eventually(t, func() bool {
		bad, badErr := engine.Snapshot(ctx, "bad-reduction")
		good, goodErr := engine.Snapshot(ctx, "good-reduction")
		return badErr == nil && goodErr == nil && bad.Status == "failed" && good.Status == "succeeded"
	}, 30*time.Second, 5*time.Millisecond)
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	bad, err := engine.Snapshot(ctx, "bad-reduction")
	require.NoError(t, err)
	found := false
	for _, attempt := range bad.Attempts {
		if attempt.Failure != nil && attempt.Failure.Code == "REDUCTION_SHARD_INVALID" {
			found = true
		}
	}
	require.True(t, found)
}

func TestBoundedReductionRunsMultipleLevelsAcrossRestart(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	workflowPath := filepath.Join(root, "workflow.db")
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(root, "artifacts"), 8<<20)
	require.NoError(t, err)
	registry, err := workflowv3reduce.Registry()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	authored, err := workflowmodule.Author(
		ctx, workflowv3reduce.WorkflowSource(), catalog,
		workflowv3reduce.DescriptorModule(),
	)
	require.NoError(t, err)
	modules, err := NewTaskModuleRegistry(FSInputModule())
	require.NoError(t, err)

	itemCount := 257
	if raceDetectorEnabled {
		itemCount = 65
	}
	const privateCanary = "PRIVATE-REDUCTION-SOURCE-CANARY"
	items := make([]workflowv3.ManifestItem, 0, itemCount)
	for index := 0; index < itemCount; index++ {
		body, err := json.Marshal(map[string]any{
			"text":    fmt.Sprintf("common group-%02d item-%03d common", index%10, index),
			"private": privateCanary + strings.Repeat("x", 128),
		})
		require.NoError(t, err)
		ref, err := artifacts.Put(ctx, "word-document/v1", "application/json", body)
		require.NoError(t, err)
		items = append(items, workflowv3.ManifestItem{
			Key: fmt.Sprintf("document-%04d", index), Value: ref,
		})
	}
	manifest, err := workflowv3.NewItemManifest("word-document/v1", items)
	require.NoError(t, err)
	manifestBody, err := workflowv3.EncodeItemManifest(manifest)
	require.NoError(t, err)
	manifestRef, err := artifacts.Put(ctx, workflowv3.ItemManifestSchemaV1, "application/json", manifestBody)
	require.NoError(t, err)

	store, err := workflowv3sqlite.Open(ctx, workflowPath)
	require.NoError(t, err)
	engine := &Engine{Store: store, Registry: registry, Artifacts: artifacts, Modules: modules}
	require.NoError(t, engine.Submit(ctx, "bounded-reduction", authored.Plan, map[string]workflowv3.ArtifactRef{
		"documents": manifestRef,
	}))
	initialQueue, err := store.QueueSnapshot(ctx, registry, map[string]int{
		workflowv3reduce.MapResource: 8, workflowv3reduce.ReduceResource: 4,
	}, time.Now().UTC())
	require.NoError(t, err)
	require.Len(t, initialQueue.Reductions, 1)
	require.Equal(t, 1, initialQueue.BlockedByReason["reduction-source"])
	restartedDuringReduction := false
	for range itemCount + 100 {
		ran, err := engine.RunOne(ctx)
		require.NoError(t, err)
		require.True(t, ran)
		queue, err := store.QueueSnapshot(ctx, registry, map[string]int{
			workflowv3reduce.MapResource: 8, workflowv3reduce.ReduceResource: 4,
		}, time.Now().UTC())
		require.NoError(t, err)
		if len(queue.Reductions) == 1 && queue.Reductions[0].CurrentLevel == 0 &&
			queue.Reductions[0].PartitionsSucceeded > 0 {
			restartedDuringReduction = true
			break
		}
	}
	require.True(t, restartedDuringReduction)
	require.NoError(t, store.Close())

	store, err = workflowv3sqlite.Open(ctx, workflowPath)
	require.NoError(t, err)
	engine = &Engine{Store: store, Registry: registry, Artifacts: artifacts, Modules: modules}
	dispatcher := &Dispatcher{
		Engine: engine,
		Capacities: map[string]int{
			workflowv3reduce.MapResource: 8, workflowv3reduce.ReduceResource: 4,
		},
		PollInterval: time.Millisecond,
	}
	dispatchCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- dispatcher.Run(dispatchCtx) }()
	timeout := 60 * time.Second
	if raceDetectorEnabled {
		timeout = 90 * time.Second
	}
	require.Eventually(t, func() bool {
		snapshot, snapshotErr := engine.Snapshot(ctx, "bounded-reduction")
		return snapshotErr == nil && snapshot.Status == "succeeded"
	}, timeout, 10*time.Millisecond)
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)

	snapshot, err := engine.Snapshot(ctx, "bounded-reduction")
	require.NoError(t, err)
	expectedPartitions := 0
	remaining := itemCount
	for remaining > 1 {
		remaining = (remaining + 7) / 8
		expectedPartitions += remaining
	}
	require.Len(t, snapshot.Attempts, itemCount+expectedPartitions)
	rootRef := snapshot.Outputs["count"]
	rootBody, err := workflowv3.ReadArtifact(ctx, artifacts, rootRef)
	require.NoError(t, err)
	var rootCount struct {
		Counts map[string]int `json:"counts"`
	}
	require.NoError(t, json.Unmarshal(rootBody, &rootCount))
	require.Equal(t, itemCount*2, rootCount.Counts["common"])
	require.Equal(t, 1, rootCount.Counts["item-000"])
	require.NotContains(t, string(rootBody), privateCanary)
	require.NoError(t, store.Checkpoint(ctx))
	require.NoError(t, store.Close())
	persisted, _ := readSQLiteFiles(t, workflowPath)
	require.NotContains(t, string(persisted), privateCanary)
}
