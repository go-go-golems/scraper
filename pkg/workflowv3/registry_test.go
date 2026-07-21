package workflowv3

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func testBundle(t *testing.T) *Bundle {
	t.Helper()
	bundle, err := NewBundle(BundleManifest{
		Name: "linear", Version: "1.0.0", ABI: TaskABI,
		Tasks: []BundleTask{{
			TaskKey:    TaskKey{Kind: "cookbook.linear.normalize", Version: "v1"},
			Entrypoint: "execution/tasks.cjs#normalize",
			Inputs:     map[string]string{"source": "source/v1"},
			Outputs:    map[string]string{"dataset": "dataset/v1"},
			Modules:    []string{"fs:input"},
		}},
	}, map[string][]byte{"execution/tasks.cjs": []byte(`exports.normalize = () => ({});`)})
	require.NoError(t, err)
	return bundle
}

func TestBundleDigestChangesWithSource(t *testing.T) {
	first := testBundle(t)
	second, err := NewBundle(first.Manifest, map[string][]byte{
		"execution/tasks.cjs": []byte(`exports.normalize = () => ({changed: true});`),
	})
	require.NoError(t, err)
	require.NotEqual(t, first.Digest(), second.Digest())
}

func TestSealedRegistryRequiresExactIdentity(t *testing.T) {
	bundle := testBundle(t)
	builder := NewRegistryBuilder()
	require.NoError(t, builder.AddBundle(bundle))
	registry, err := builder.Seal()
	require.NoError(t, err)

	identity := bundle.TaskSpecs()[0].Identity
	_, err = registry.Resolve(identity)
	require.NoError(t, err)

	wrongDigest := identity
	wrongDigest.BundleDigest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	_, err = registry.Resolve(wrongDigest)
	require.ErrorContains(t, err, "does not advertise exact implementation")

	wrongEntrypoint := identity
	wrongEntrypoint.Entrypoint = "execution/tasks.cjs#other"
	_, err = registry.Resolve(wrongEntrypoint)
	require.ErrorContains(t, err, "does not advertise exact implementation")

	wrongABI := identity
	wrongABI.ABI = "scraper-js-task/v2"
	_, err = registry.Resolve(wrongABI)
	require.ErrorContains(t, err, "does not advertise exact implementation")
}

func TestBundleRejectsMissingEntrypointFile(t *testing.T) {
	_, err := NewBundle(BundleManifest{
		Name: "bad", Version: "1", ABI: TaskABI,
		Tasks: []BundleTask{{
			TaskKey:    TaskKey{Kind: "bad", Version: "v1"},
			Entrypoint: "execution/missing.cjs#run",
			Inputs:     map[string]string{"input": "x/v1"},
			Outputs:    map[string]string{"output": "y/v1"},
		}},
	}, map[string][]byte{"execution/tasks.cjs": []byte("exports.run = () => {}")})
	require.ErrorContains(t, err, "is missing")
}
