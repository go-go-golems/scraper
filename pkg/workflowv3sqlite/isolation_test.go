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

func TestRestrictedIsolationPersistsOnNodeLeaseAndAttempt(t *testing.T) {
	ctx := context.Background()
	maximum := workflowv3.IsolationPolicy{
		Class:          workflowv3.IsolationSubprocessRestricted,
		WallTimeMillis: 30_000, CPUTimeMillis: 10_000, MemoryBytes: 256 << 20,
		MaxProcesses: 8, MaxOutputBytes: 16 << 20,
		MaxOutputFiles: 16, MaxProtocolBytes: 1 << 20,
	}
	bundle, err := workflowv3.NewBundle(workflowv3.BundleManifest{
		Name: "restricted", Version: "1", ABI: workflowv3.TaskABI,
		Tasks: []workflowv3.BundleTask{{
			TaskKey: workflowv3.TaskKey{Kind: "restricted.task", Version: "v1"}, Entrypoint: "tasks.cjs#run",
			Inputs: map[string]string{"input": "input/v1"}, Outputs: map[string]string{"output": "output/v1"},
			ResourceClass: "cpu.restricted", IsolationMaximum: &maximum,
		}},
	}, map[string][]byte{"tasks.cjs": []byte(`exports.run = () => ({})`)})
	require.NoError(t, err)
	builder := workflowv3.NewRegistryBuilder()
	require.NoError(t, builder.AddBundle(bundle))
	require.NoError(t, builder.AdvertiseIsolationExecutor(workflowv3.IsolationSubprocessRestricted, "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"))
	registry, err := builder.Seal()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	requested := maximum
	requested.WallTimeMillis = 5_000
	requested.MemoryBytes = 512 << 20
	plan, err := workflowv3.Compile(workflowv3.WorkflowIR{
		Schema: workflowv3.IRSchema, Name: "restricted",
		Inputs: []workflowv3.IRInput{{Name: "input", Schema: "input/v1"}},
		Nodes: []workflowv3.IRNode{{
			Key: "task", Task: workflowv3.TaskKey{Kind: "restricted.task", Version: "v1"},
			Bindings:  map[string]workflowv3.ValueRef{"input": {Source: "input", Name: "input", Schema: "input/v1"}},
			Isolation: &requested,
		}},
		Outputs: []workflowv3.IROutput{{Name: "output", Value: workflowv3.ValueRef{Source: "node-output", NodeKey: "task", Port: "output", Schema: "output/v1"}}},
	}, catalog)
	require.NoError(t, err)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	now := time.Now().UTC()
	input := workflowv3.ArtifactRef{Schema: "input/v1", Digest: "sha256:" + strings.Repeat("1", 64), MediaType: "application/json", Size: 2, Locator: "cas://input"}
	require.NoError(t, store.CreateRun(ctx, "restricted", plan, map[string]workflowv3.ArtifactRef{"input": input}, now))
	lease, err := store.LeaseNext(ctx, registry, now, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.PlanNode.Isolation)
	require.Equal(t, workflowv3.IsolationSubprocessRestricted, lease.PlanNode.Isolation.Effective.Class)
	require.Equal(t, int64(5_000), lease.PlanNode.Isolation.Effective.WallTimeMillis)
	require.Equal(t, int64(256<<20), lease.PlanNode.Isolation.Effective.MemoryBytes)

	snapshot, err := store.Snapshot(ctx, "restricted")
	require.NoError(t, err)
	require.Len(t, snapshot.Attempts, 1)
	require.Equal(t, workflowv3.IsolationSubprocessRestricted, snapshot.Attempts[0].IsolationClass)
	require.Equal(t, lease.PlanNode.Isolation.PolicyDigest, snapshot.Attempts[0].IsolationPolicyDigest)
	require.Equal(t, lease.PlanNode.Isolation.ExecutorDigest, snapshot.Attempts[0].IsolationExecutorDigest)
}
