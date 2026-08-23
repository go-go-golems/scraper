package workflowv3sqlite

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/stretchr/testify/require"
)

func budgetedMapFixture(t *testing.T) (*workflowv3.SealedRegistry, workflowv3.WorkflowPlan) {
	t.Helper()
	maximum := &workflowv3.BudgetClaim{
		Account: "provider", OnExhausted: workflowv3.BudgetExhaustBlock,
		Reserve: []workflowv3.BudgetAmount{{Dimension: "requests", Units: 1}},
	}
	bundle, err := workflowv3.NewBundle(workflowv3.BundleManifest{
		Name: "budget-map", Version: "1", ABI: workflowv3.TaskABI,
		Tasks: []workflowv3.BundleTask{{
			TaskKey:    workflowv3.TaskKey{Kind: "budget.map", Version: "v1"},
			Entrypoint: "tasks.cjs#run", Inputs: map[string]string{"item": "item/v1"},
			Outputs: map[string]string{"output": "result/v1"}, ResourceClass: "network.map",
			BudgetMaximum: maximum,
		}},
	}, map[string][]byte{"tasks.cjs": []byte(`exports.run = value => value`)})
	require.NoError(t, err)
	builder := workflowv3.NewRegistryBuilder()
	require.NoError(t, builder.AddBundle(bundle))
	registry, err := builder.Seal()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	claim := &workflowv3.BudgetClaim{
		Account: "provider", OnExhausted: workflowv3.BudgetExhaustBlock,
		Reserve: []workflowv3.BudgetAmount{{Dimension: "requests", Units: 1}},
	}
	plan, err := workflowv3.Compile(workflowv3.WorkflowIR{
		Schema: workflowv3.IRSchema, Name: "budget-map",
		SetInputs: []workflowv3.IRSetInput{{Name: "items", ItemSchema: "item/v1", ManifestSchema: workflowv3.ItemManifestSchemaV1, Policy: workflowv3.SetInputPolicy{MaxItems: 2}}},
		Budgets:   []workflowv3.BudgetAccount{{Account: "provider", PolicyDigest: "sha256:" + strings.Repeat("1", 64), Limits: []workflowv3.BudgetAmount{{Dimension: "requests", Units: 1}}}},
		Maps: []workflowv3.IRMap{{
			Key: "mapped", Source: workflowv3.SetRef{Source: "set-input", Name: "items", ItemSchema: "item/v1", ManifestSchema: workflowv3.ItemManifestSchemaV1},
			ItemTask: workflowv3.TaskKey{Kind: "budget.map", Version: "v1"},
			Bindings: map[string]workflowv3.ValueRef{"item": {Source: "map-item", MapKey: "mapped", Schema: "item/v1"}},
			Policy:   workflowv3.MapPolicy{PageSize: 2, MaxItems: 2, MaxMaterializedAhead: 2}, Budget: claim,
		}},
		SetOutputs: []workflowv3.IRSetOutput{{Name: "results", Value: workflowv3.SetRef{Source: "map-output", MapKey: "mapped", ItemSchema: "result/v1", ManifestSchema: workflowv3.ItemManifestSchemaV1}}},
	}, catalog)
	require.NoError(t, err)
	return registry, plan
}

func budgetedReductionFixture(t *testing.T) (*workflowv3.SealedRegistry, workflowv3.WorkflowPlan) {
	t.Helper()
	maximum := &workflowv3.BudgetClaim{Account: "provider", OnExhausted: workflowv3.BudgetExhaustBlock, Reserve: []workflowv3.BudgetAmount{{Dimension: "requests", Units: 1}}}
	bundle, err := workflowv3.NewBundle(workflowv3.BundleManifest{
		Name: "budget-reduce", Version: "1", ABI: workflowv3.TaskABI,
		Tasks: []workflowv3.BundleTask{{
			TaskKey: workflowv3.TaskKey{Kind: "budget.reduce", Version: "v1"}, Entrypoint: "tasks.cjs#run",
			Inputs: map[string]string{"partition": workflowv3.ReductionPartitionSchemaV1}, Outputs: map[string]string{"output": "item/v1"},
			ResourceClass: "network.reduce", BudgetMaximum: maximum,
		}},
	}, map[string][]byte{"tasks.cjs": []byte(`exports.run = value => value`)})
	require.NoError(t, err)
	builder := workflowv3.NewRegistryBuilder()
	require.NoError(t, builder.AddBundle(bundle))
	registry, err := builder.Seal()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	claim := &workflowv3.BudgetClaim{Account: "provider", OnExhausted: workflowv3.BudgetExhaustBlock, Reserve: []workflowv3.BudgetAmount{{Dimension: "requests", Units: 1}}}
	plan, err := workflowv3.Compile(workflowv3.WorkflowIR{
		Schema: workflowv3.IRSchema, Name: "budget-reduce",
		SetInputs: []workflowv3.IRSetInput{{Name: "items", ItemSchema: "item/v1", ManifestSchema: workflowv3.ItemManifestSchemaV1, Policy: workflowv3.SetInputPolicy{MaxItems: 8}}},
		Budgets:   []workflowv3.BudgetAccount{{Account: "provider", PolicyDigest: "sha256:" + strings.Repeat("2", 64), Limits: []workflowv3.BudgetAmount{{Dimension: "requests", Units: 1}}}},
		Reductions: []workflowv3.IRReduce{{
			Key: "reduced", Source: workflowv3.SetRef{Source: "set-input", Name: "items", ItemSchema: "item/v1", ManifestSchema: workflowv3.ItemManifestSchemaV1},
			PartitionTask: workflowv3.TaskKey{Kind: "budget.reduce", Version: "v1"},
			Bindings:      map[string]workflowv3.ValueRef{"partition": {Source: "reduction-partition", ReduceKey: "reduced", Schema: workflowv3.ReductionPartitionSchemaV1}},
			Policy:        workflowv3.ReducePolicy{FanIn: 2, MaxLevels: 3}, Budget: claim,
		}},
		Outputs: []workflowv3.IROutput{{Name: "root", Value: workflowv3.ValueRef{Source: "reduction-output", ReduceKey: "reduced", Schema: "item/v1"}}},
	}, catalog)
	require.NoError(t, err)
	return registry, plan
}

func TestBudgetClaimsPropagateToDynamicMapChildren(t *testing.T) {
	ctx := context.Background()
	registry, plan := budgetedMapFixture(t)
	manifest, manifestRef := mapManifest(t, 2)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	now := time.Now().UTC()
	require.NoError(t, store.CreateRun(ctx, "budget-map", plan, map[string]workflowv3.ArtifactRef{"items": manifestRef}, now))
	page, err := store.ExpandNextPage(ctx, "budget-map", "mapped", manifestRef, manifest, now)
	require.NoError(t, err)
	require.Equal(t, 2, page.ItemCount)
	first, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, first)
	second, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	require.Nil(t, second)
	queue, err := store.QueueSnapshot(ctx, registry, nil, now)
	require.NoError(t, err)
	require.Equal(t, 1, queue.BlockedByReason["budget:provider:requests"])
}

func TestBudgetClaimsPropagateToReductionPartitions(t *testing.T) {
	ctx := context.Background()
	registry, plan := budgetedReductionFixture(t)
	manifest, manifestRef := mapManifest(t, 4)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	now := time.Now().UTC()
	require.NoError(t, store.CreateRun(ctx, "budget-reduce", plan, map[string]workflowv3.ArtifactRef{"items": manifestRef}, now))
	partitions := make([]ReductionPartitionInput, 0, 2)
	for ordinal := range 2 {
		partition, err := workflowv3.NewReductionPartition(
			"reduced", manifestRef.Digest, "item/v1", 0, ordinal, 2,
			manifest.Items[ordinal*2:ordinal*2+2],
		)
		require.NoError(t, err)
		body, err := workflowv3.EncodeReductionPartition(partition, 2)
		require.NoError(t, err)
		digest, err := workflowv3.Digest(partition)
		require.NoError(t, err)
		partitions = append(partitions, ReductionPartitionInput{
			Partition: partition,
			Ref:       workflowv3.ArtifactRef{Schema: workflowv3.ReductionPartitionSchemaV1, Digest: digest, MediaType: "application/json", Size: int64(len(body)), Locator: "cas://partition"},
		})
	}
	require.NoError(t, store.MaterializeReductionLevel(ctx, "budget-reduce", "reduced", manifestRef, 4, 0, partitions, now))
	first, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, first)
	second, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	require.Nil(t, second)
	queue, err := store.QueueSnapshot(ctx, registry, nil, now)
	require.NoError(t, err)
	require.Equal(t, 1, queue.BlockedByReason["budget:provider:requests"])
}
