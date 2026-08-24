package workflowv3

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const testIsolationExecutorDigest = "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"

func restrictedIsolation() IsolationPolicy {
	return IsolationPolicy{
		Class:          IsolationSubprocessRestricted,
		WallTimeMillis: 30_000, CPUTimeMillis: 10_000, MemoryBytes: 256 << 20,
		MaxProcesses: 8, MaxOutputBytes: 16 << 20,
		MaxOutputFiles: 16, MaxProtocolBytes: 1 << 20,
	}
}

func TestIsolationPolicyValidationAndCompilation(t *testing.T) {
	require.NoError(t, ValidateIsolationPolicy(TrustedIsolationPolicy()))
	require.NoError(t, ValidateIsolationPolicy(restrictedIsolation()))

	invalid := restrictedIsolation()
	invalid.Class = "container.networked"
	require.ErrorContains(t, ValidateIsolationPolicy(invalid), "not supported")
	invalid = TrustedIsolationPolicy()
	invalid.MemoryBytes = 1
	require.ErrorContains(t, ValidateIsolationPolicy(invalid), "cannot declare")
	invalid = restrictedIsolation()
	invalid.MaxProtocolBytes = 0
	require.ErrorContains(t, ValidateIsolationPolicy(invalid), "protocol")

	maximum := restrictedIsolation()
	requested := maximum
	requested.WallTimeMillis = 5_000
	requested.MemoryBytes = 512 << 20
	compiled, err := CompileIsolation(&requested, maximum, testIsolationExecutorDigest)
	require.NoError(t, err)
	require.Equal(t, requested, compiled.Requested)
	require.Equal(t, int64(5_000), compiled.Effective.WallTimeMillis)
	require.Equal(t, int64(256<<20), compiled.Effective.MemoryBytes)
	require.NoError(t, validateSHA256Digest(compiled.PolicyDigest))

	trusted := TrustedIsolationPolicy()
	_, err = CompileIsolation(&trusted, maximum, testIsolationExecutorDigest)
	require.ErrorContains(t, err, "does not match task-required class")
}

func TestCompilerRequiresRestrictedIsolationForBroadModules(t *testing.T) {
	catalog, err := NewCatalog(TaskSpec{
		Identity: ImplementationIdentity{TaskKey: TaskKey{Kind: "broad.task", Version: "v1"}, BundleDigest: "sha256:" + strings.Repeat("a", 64), Entrypoint: "tasks.cjs#run", ABI: TaskABI},
		Inputs:   map[string]string{"input": "input/v1"}, Outputs: map[string]string{"output": "output/v1"},
		Modules: []string{"exec:allowlisted"}, ResourceClass: ResourceCPUDefault, Retry: RetryPolicy{MaxAttempts: 1},
	})
	require.NoError(t, err)
	_, err = Compile(WorkflowIR{
		Schema: IRSchema, Name: "broad",
		Inputs:  []IRInput{{Name: "input", Schema: "input/v1"}},
		Nodes:   []IRNode{{Key: "task", Task: TaskKey{Kind: "broad.task", Version: "v1"}, Bindings: map[string]ValueRef{"input": {Source: "input", Name: "input", Schema: "input/v1"}}}},
		Outputs: []IROutput{{Name: "output", Value: ValueRef{Source: "node-output", NodeKey: "task", Port: "output", Schema: "output/v1"}}},
	}, catalog)
	require.ErrorContains(t, err, "modules require restricted isolation")
}

func TestCatalogDefaultsIsolationAndBundleDigestPinsIt(t *testing.T) {
	manifest := BundleManifest{Name: "isolation", Version: "1", ABI: TaskABI, Tasks: []BundleTask{{
		TaskKey:    TaskKey{Kind: "isolation.task", Version: "v1"},
		Entrypoint: "tasks.cjs#run", Inputs: map[string]string{"input": "input/v1"}, Outputs: map[string]string{"output": "output/v1"},
	}}}
	trustedBundle, err := NewBundle(manifest, map[string][]byte{"tasks.cjs": []byte(`exports.run = () => ({})`)})
	require.NoError(t, err)
	trustedSpec := trustedBundle.TaskSpecs()[0]
	require.Equal(t, IsolationInProcessTrusted, trustedSpec.IsolationMaximum.Class)

	restricted := restrictedIsolation()
	manifest.Tasks[0].IsolationMaximum = &restricted
	restrictedBundle, err := NewBundle(manifest, map[string][]byte{"tasks.cjs": []byte(`exports.run = () => ({})`)})
	require.NoError(t, err)
	require.NotEqual(t, trustedBundle.Digest(), restrictedBundle.Digest())
	require.Equal(t, IsolationSubprocessRestricted, restrictedBundle.TaskSpecs()[0].IsolationMaximum.Class)

	missing := NewRegistryBuilder()
	require.NoError(t, missing.AddBundle(restrictedBundle))
	_, err = missing.Seal()
	require.ErrorContains(t, err, "unadvertised restricted isolation executor")

	first := NewRegistryBuilder()
	require.NoError(t, first.AdvertiseIsolationExecutor(IsolationSubprocessRestricted, testIsolationExecutorDigest))
	require.NoError(t, first.AddBundle(restrictedBundle))
	firstRegistry, err := first.Seal()
	require.NoError(t, err)
	firstCatalog, err := firstRegistry.Catalog()
	require.NoError(t, err)
	firstSpec, ok := firstCatalog.Lookup(TaskKey{Kind: "isolation.task", Version: "v1"})
	require.True(t, ok)
	require.Equal(t, testIsolationExecutorDigest, firstSpec.IsolationExecutorDigest)
	compiledIsolation, err := CompileIsolation(nil, firstSpec.IsolationMaximum, firstSpec.IsolationExecutorDigest)
	require.NoError(t, err)
	node := PlanNode{
		Key: "task", Implementation: firstSpec.Identity,
		InputSchemas: firstSpec.Inputs, OutputSchemas: firstSpec.Outputs,
		Modules: firstSpec.Modules, ResourceClass: firstSpec.ResourceClass,
		Retry: firstSpec.Retry, Isolation: &compiledIsolation,
	}
	_, err = firstRegistry.ResolveNode(node)
	require.NoError(t, err)
	wrongIsolation := compiledIsolation
	wrongIsolation.ExecutorDigest = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	node.Isolation = &wrongIsolation
	_, err = firstRegistry.ResolveNode(node)
	require.ErrorContains(t, err, "isolation")

	secondDigest := "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	second := NewRegistryBuilder()
	require.NoError(t, second.AdvertiseIsolationExecutor(IsolationSubprocessRestricted, secondDigest))
	require.NoError(t, second.AddBundle(restrictedBundle))
	secondRegistry, err := second.Seal()
	require.NoError(t, err)
	require.NotEqual(t, firstRegistry.Generation(), secondRegistry.Generation())
}
