package workflowv3sqlite

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/stretchr/testify/require"
)

func mapStoreFixture(t *testing.T, policy workflowv3.MapPolicy) (*workflowv3.SealedRegistry, workflowv3.WorkflowPlan) {
	t.Helper()
	bundle, err := workflowv3.NewBundle(workflowv3.BundleManifest{
		Name: "map-fixture", Version: "1", ABI: workflowv3.TaskABI,
		Tasks: []workflowv3.BundleTask{{
			TaskKey:    workflowv3.TaskKey{Kind: "map.normalize", Version: "v1"},
			Entrypoint: "tasks.cjs#normalize", Inputs: map[string]string{"item": "item/v1"},
			Outputs: map[string]string{"output": "result/v1"}, ResourceClass: "cpu.map",
		}},
	}, map[string][]byte{"tasks.cjs": []byte(`exports.normalize = item => item`)})
	require.NoError(t, err)
	builder := workflowv3.NewRegistryBuilder()
	require.NoError(t, builder.AddBundle(bundle))
	registry, err := builder.Seal()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	plan, err := workflowv3.Compile(workflowv3.WorkflowIR{
		Schema: workflowv3.IRSchema, Name: "map",
		Inputs: []workflowv3.IRInput{}, Nodes: []workflowv3.IRNode{}, Outputs: []workflowv3.IROutput{},
		SetInputs: []workflowv3.IRSetInput{{
			Name: "items", ItemSchema: "item/v1", ManifestSchema: workflowv3.ItemManifestSchemaV1,
		}},
		Maps: []workflowv3.IRMap{{
			Key:      "normalize",
			Source:   workflowv3.SetRef{Source: "set-input", Name: "items", ItemSchema: "item/v1", ManifestSchema: workflowv3.ItemManifestSchemaV1},
			ItemTask: workflowv3.TaskKey{Kind: "map.normalize", Version: "v1"},
			Bindings: map[string]workflowv3.ValueRef{
				"item": {Source: "map-item", MapKey: "normalize", Schema: "item/v1"},
			},
			Policy: policy,
		}},
		SetOutputs: []workflowv3.IRSetOutput{{
			Name: "results", Value: workflowv3.SetRef{Source: "map-output", MapKey: "normalize", ItemSchema: "result/v1", ManifestSchema: workflowv3.ItemManifestSchemaV1},
		}},
	}, catalog)
	require.NoError(t, err)
	return registry, plan
}

func mapManifest(t *testing.T, count int) (workflowv3.ItemManifest, workflowv3.ArtifactRef) {
	t.Helper()
	items := make([]workflowv3.ManifestItem, 0, count)
	for index := 0; index < count; index++ {
		key := fmtItemKey(index)
		digest := sha256.Sum256([]byte(key))
		items = append(items, workflowv3.ManifestItem{
			Key: key,
			Value: workflowv3.ArtifactRef{
				Schema: "item/v1", Digest: "sha256:" + hex.EncodeToString(digest[:]),
				MediaType: "application/json", Size: int64(index + 1), Locator: "cas://" + key,
			},
		})
	}
	manifest, err := workflowv3.NewItemManifest("item/v1", items)
	require.NoError(t, err)
	body, err := workflowv3.EncodeItemManifest(manifest)
	require.NoError(t, err)
	digest, err := workflowv3.Digest(manifest)
	require.NoError(t, err)
	return manifest, workflowv3.ArtifactRef{
		Schema: workflowv3.ItemManifestSchemaV1, Digest: digest,
		MediaType: "application/json", Size: int64(len(body)), Locator: "cas://manifest",
	}
}

func fmtItemKey(index int) string {
	return "item-" + string(rune('a'+index))
}

func TestExpansionPagesBackpressureReopenAndResolveItems(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "workflow.db")
	registry, plan := mapStoreFixture(t, workflowv3.MapPolicy{
		PageSize: 2, MaxItems: 10, MaxMaterializedAhead: 2,
	})
	manifest, manifestRef := mapManifest(t, 5)
	now := time.Date(2026, 7, 21, 20, 0, 0, 0, time.UTC)

	store, err := Open(ctx, path)
	require.NoError(t, err)
	require.NoError(t, store.CreateRun(ctx, "map-run", plan, map[string]workflowv3.ArtifactRef{"items": manifestRef}, now))
	page, err := store.ExpandNextPage(ctx, "map-run", "normalize", manifestRef, manifest, now)
	require.NoError(t, err)
	require.Equal(t, 2, page.ItemCount)
	require.Equal(t, 2, page.NextIndex)

	blocked, err := store.ExpandNextPage(ctx, "map-run", "normalize", manifestRef, manifest, now.Add(time.Millisecond))
	require.NoError(t, err)
	require.Nil(t, blocked)
	queue, err := store.QueueSnapshot(ctx, registry, map[string]int{"cpu.map": 2}, now)
	require.NoError(t, err)
	require.Len(t, queue.Maps, 1)
	require.Equal(t, 5, queue.Maps[0].TotalItems)
	require.Equal(t, 3, queue.Maps[0].BacklogToMaterialize)
	require.Equal(t, 2, queue.Maps[0].BacklogToExecute)
	require.Equal(t, 1, queue.BlockedByReason["map-backpressure"])

	for index := 0; index < 2; index++ {
		lease, err := store.LeaseNext(ctx, registry, now.Add(time.Duration(index+1)*time.Second), time.Minute)
		require.NoError(t, err)
		require.NotNil(t, lease)
		inputs, err := store.ResolveInputs(ctx, *lease)
		require.NoError(t, err)
		require.Equal(t, manifest.Items[index].Value, inputs["item"])
		output := artifactRef("result/v1", "map-output-"+fmtItemKey(index))
		require.NoError(t, store.Complete(ctx, *lease, map[string]workflowv3.ArtifactRef{"output": output}, now.Add(time.Duration(index+2)*time.Second)))
	}
	require.NoError(t, store.Close())

	store, err = Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	page, err = store.ExpandNextPage(ctx, "map-run", "normalize", manifestRef, manifest, now.Add(4*time.Second))
	require.NoError(t, err)
	require.Equal(t, 2, page.ItemCount)
	require.Equal(t, 4, page.NextIndex)

	var pages, items, nodes int
	require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM v3_expansion_pages WHERE run_id = 'map-run'`).Scan(&pages))
	require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM v3_map_items WHERE run_id = 'map-run'`).Scan(&items))
	require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM v3_nodes WHERE run_id = 'map-run'`).Scan(&nodes))
	require.Equal(t, 2, pages)
	require.Equal(t, 4, items)
	require.Equal(t, items, nodes)
}

func TestExpansionIsAtomicAcrossIndependentConnections(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "workflow.db")
	first, err := Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, first.Close()) }()
	second, err := Open(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, second.Close()) }()
	_, plan := mapStoreFixture(t, workflowv3.MapPolicy{
		PageSize: 2, MaxItems: 4, MaxMaterializedAhead: 4,
	})
	manifest, ref := mapManifest(t, 4)
	now := time.Date(2026, 7, 21, 20, 30, 0, 0, time.UTC)
	require.NoError(t, first.CreateRun(ctx, "map-run", plan, map[string]workflowv3.ArtifactRef{"items": ref}, now))

	start := make(chan struct{})
	results := make(chan *ExpansionPage, 2)
	errors := make(chan error, 2)
	for _, store := range []*Store{first, second} {
		store := store
		go func() {
			<-start
			page, expandErr := store.ExpandNextPage(ctx, "map-run", "normalize", ref, manifest, now)
			results <- page
			errors <- expandErr
		}()
	}
	close(start)
	for range 2 {
		require.NoError(t, <-errors)
		require.NotNil(t, <-results)
	}

	var pages, items, distinctItems int
	require.NoError(t, first.db.QueryRow(`SELECT COUNT(*) FROM v3_expansion_pages WHERE run_id = 'map-run'`).Scan(&pages))
	require.NoError(t, first.db.QueryRow(`SELECT COUNT(*), COUNT(DISTINCT item_key) FROM v3_map_items WHERE run_id = 'map-run'`).Scan(&items, &distinctItems))
	require.Equal(t, 2, pages)
	require.Equal(t, 4, items)
	require.Equal(t, items, distinctItems)
}

func TestEmptyExpansionCompletesWithoutLease(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	registry, plan := mapStoreFixture(t, workflowv3.MapPolicy{PageSize: 2, MaxItems: 2, MaxMaterializedAhead: 2})
	manifest, ref := mapManifest(t, 0)
	now := time.Now().UTC()
	require.NoError(t, store.CreateRun(ctx, "empty-map", plan, map[string]workflowv3.ArtifactRef{"items": ref}, now))
	page, err := store.ExpandNextPage(ctx, "empty-map", "normalize", ref, manifest, now)
	require.NoError(t, err)
	require.NotNil(t, page)
	require.True(t, page.Expanded)
	require.Zero(t, page.ItemCount)
	lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	require.Nil(t, lease)
	outputManifest, err := store.MapOutputManifest(ctx, "empty-map", "normalize")
	require.NoError(t, err)
	body, err := workflowv3.EncodeItemManifest(outputManifest)
	require.NoError(t, err)
	digest, err := workflowv3.Digest(outputManifest)
	require.NoError(t, err)
	outputRef := workflowv3.ArtifactRef{
		Schema: workflowv3.ItemManifestSchemaV1, Digest: digest,
		MediaType: "application/json", Size: int64(len(body)), Locator: "cas://empty-output",
	}
	require.NoError(t, store.PublishMapOutput(ctx, "empty-map", "normalize", outputRef, now))
	require.NoError(t, store.PublishMapOutput(ctx, "empty-map", "normalize", outputRef, now))
	snapshot, err := store.Snapshot(ctx, "empty-map")
	require.NoError(t, err)
	require.Equal(t, "succeeded", snapshot.Status)
	require.Equal(t, outputRef, snapshot.Outputs["results"])
}

func TestExpansionCancellationAndTerminalFailureStopScaleOut(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		name string
		stop func(*Store, *workflowv3.SealedRegistry, time.Time) error
		want string
	}{
		{
			name: "cancel",
			stop: func(store *Store, _ *workflowv3.SealedRegistry, now time.Time) error {
				return store.Cancel(ctx, "map-run", now)
			},
			want: "canceled",
		},
		{
			name: "terminal child failure",
			stop: func(store *Store, registry *workflowv3.SealedRegistry, now time.Time) error {
				lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
				if err != nil {
					return err
				}
				return store.Fail(ctx, *lease, workflowv3.Failure{
					Class: "validation", Code: "MAP_ITEM_INVALID", Retryable: false,
					Message: "task reported MAP_ITEM_INVALID",
				}, now)
			},
			want: "failed",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
			require.NoError(t, err)
			defer func() { require.NoError(t, store.Close()) }()
			registry, plan := mapStoreFixture(t, workflowv3.MapPolicy{PageSize: 2, MaxItems: 4, MaxMaterializedAhead: 2})
			manifest, ref := mapManifest(t, 4)
			now := time.Now().UTC()
			require.NoError(t, store.CreateRun(ctx, "map-run", plan, map[string]workflowv3.ArtifactRef{"items": ref}, now))
			page, err := store.ExpandNextPage(ctx, "map-run", "normalize", ref, manifest, now)
			require.NoError(t, err)
			require.NotNil(t, page)
			require.NoError(t, test.stop(store, registry, now.Add(time.Second)))
			candidates, err := store.ExpansionCandidates(ctx)
			require.NoError(t, err)
			require.Empty(t, candidates)
			var status string
			require.NoError(t, store.db.QueryRow(`SELECT status FROM v3_expansions WHERE run_id = 'map-run'`).Scan(&status))
			require.Equal(t, test.want, status)
		})
	}
}

func TestExpansionRejectsManifestIdentityAndCardinalityDrift(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	_, plan := mapStoreFixture(t, workflowv3.MapPolicy{PageSize: 2, MaxItems: 2, MaxMaterializedAhead: 2})
	manifest, ref := mapManifest(t, 3)
	now := time.Now().UTC()
	require.NoError(t, store.CreateRun(ctx, "map-run", plan, map[string]workflowv3.ArtifactRef{"items": ref}, now))
	_, err = store.ExpandNextPage(ctx, "map-run", "normalize", ref, manifest, now)
	require.ErrorContains(t, err, "cardinality")

	manifest, ref = mapManifest(t, 2)
	wrong := ref
	wrong.Digest = artifactRef("x", "wrong-manifest").Digest
	_, err = store.ExpandNextPage(ctx, "map-run", "normalize", wrong, manifest, now)
	require.ErrorContains(t, err, "does not match canonical manifest")
}
