package workflowv3

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const testBundleDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func testCatalog(t *testing.T) *Catalog {
	t.Helper()
	catalog, err := NewCatalog(
		TaskSpec{
			Identity: ImplementationIdentity{
				TaskKey:      TaskKey{Kind: "cookbook.linear.normalize-customers", Version: "v1"},
				BundleDigest: testBundleDigest,
				Entrypoint:   "./execution/tasks.cjs#normalizeCustomers",
				ABI:          TaskABI,
			},
			Inputs:  map[string]string{"source": "customer-jsonl-ref/v1"},
			Outputs: map[string]string{"dataset": "normalized-customers-ref/v1"},
			Modules: []string{"fs:input"},
		},
		TaskSpec{
			Identity: ImplementationIdentity{
				TaskKey:      TaskKey{Kind: "cookbook.linear.validate-dataset", Version: "v1"},
				BundleDigest: testBundleDigest,
				Entrypoint:   "./execution/tasks.cjs#validateDataset",
				ABI:          TaskABI,
			},
			Inputs:  map[string]string{"dataset": "normalized-customers-ref/v1"},
			Outputs: map[string]string{"validatedDataset": "validated-customers-ref/v1"},
			Modules: []string{"fs:input"},
		},
	)
	require.NoError(t, err)
	return catalog
}

func testIR() WorkflowIR {
	return WorkflowIR{
		Schema: IRSchema,
		Name:   "linear-transform",
		Inputs: []IRInput{{Name: "source", Schema: "customer-jsonl-ref/v1"}},
		Nodes: []IRNode{
			{
				Key:  "normalize",
				Task: TaskKey{Kind: "cookbook.linear.normalize-customers", Version: "v1"},
				Bindings: map[string]ValueRef{
					"source": {Source: "input", Name: "source", Schema: "customer-jsonl-ref/v1"},
				},
			},
			{
				Key:       "validate",
				Task:      TaskKey{Kind: "cookbook.linear.validate-dataset", Version: "v1"},
				DependsOn: []NodeKey{"normalize"},
				Bindings: map[string]ValueRef{
					"dataset": {
						Source: "node-output", NodeKey: "normalize", Port: "dataset",
						Schema: "normalized-customers-ref/v1",
					},
				},
			},
		},
		Outputs: []IROutput{{
			Name: "dataset",
			Value: ValueRef{
				Source: "node-output", NodeKey: "validate", Port: "validatedDataset",
				Schema: "validated-customers-ref/v1",
			},
		}},
	}
}

func TestCompilePinsExactIdentityAndIsDeterministic(t *testing.T) {
	catalog := testCatalog(t)
	first, err := Compile(testIR(), catalog)
	require.NoError(t, err)
	second, err := Compile(testIR(), catalog)
	require.NoError(t, err)

	require.Equal(t, first, second)
	require.NotEmpty(t, first.Digest)
	require.Equal(t, testBundleDigest, first.Nodes[0].Implementation.BundleDigest)
	require.Equal(t, "./execution/tasks.cjs#normalizeCustomers", first.Nodes[0].Implementation.Entrypoint)
	require.Equal(t, TaskABI, first.Nodes[0].Implementation.ABI)
	require.Equal(t, ResourceCPUDefault, first.Nodes[0].ResourceClass)
	require.Equal(t, RetryPolicy{MaxAttempts: 1}, first.Nodes[0].Retry)
}

func TestValidateIRRejectsSchemaMismatch(t *testing.T) {
	ir := testIR()
	ir.Nodes[1].Bindings["dataset"] = ValueRef{
		Source: "node-output", NodeKey: "normalize", Port: "dataset", Schema: "wrong/v1",
	}
	err := ValidateIR(ir, testCatalog(t))
	require.ErrorContains(t, err, "does not match")
}

func TestValidateIRRejectsDependencyCycle(t *testing.T) {
	ir := testIR()
	ir.Nodes[0].DependsOn = []NodeKey{"validate"}
	err := ValidateIR(ir, testCatalog(t))
	require.ErrorContains(t, err, "cycle")
}

func TestCatalogRejectsUnsupportedABI(t *testing.T) {
	_, err := NewCatalog(TaskSpec{
		Identity: ImplementationIdentity{
			TaskKey: TaskKey{Kind: "x", Version: "v1"}, BundleDigest: testBundleDigest,
			Entrypoint: "./tasks.cjs#run", ABI: "unknown/v1",
		},
		Inputs: map[string]string{"input": "x/v1"}, Outputs: map[string]string{"output": "y/v1"},
	})
	require.ErrorContains(t, err, "not supported")
}

func TestCatalogRejectsInvalidResourceAndRetryPolicy(t *testing.T) {
	base := TaskSpec{
		Identity: ImplementationIdentity{
			TaskKey: TaskKey{Kind: "x", Version: "v1"}, BundleDigest: testBundleDigest,
			Entrypoint: "tasks.cjs#run", ABI: TaskABI,
		},
		Inputs: map[string]string{"input": "x/v1"}, Outputs: map[string]string{"output": "y/v1"},
	}
	invalidResource := base
	invalidResource.ResourceClass = "NOT VALID"
	_, err := NewCatalog(invalidResource)
	require.ErrorContains(t, err, "invalid resource class")
	invalidRetry := base
	invalidRetry.Retry = RetryPolicy{MaxAttempts: -1}
	_, err = NewCatalog(invalidRetry)
	require.ErrorContains(t, err, "max attempts")
}

func TestStrictDecodeRejectsUnknownFields(t *testing.T) {
	var ref ArtifactRef
	err := StrictDecode([]byte(`{"schema":"x","unknown":true}`), &ref)
	require.ErrorContains(t, err, "unknown field")
}

func TestCompileMapPinsTemplateAndSetIdentity(t *testing.T) {
	catalog, err := NewCatalog(TaskSpec{
		Identity: ImplementationIdentity{
			TaskKey:      TaskKey{Kind: "cookbook.map.normalize-customer", Version: "v1"},
			BundleDigest: testBundleDigest, Entrypoint: "tasks.cjs#normalizeCustomer", ABI: TaskABI,
		},
		Inputs:  map[string]string{"customer": "customer/v1"},
		Outputs: map[string]string{"normalized": "normalized-customer/v1"},
		Modules: []string{"fs:input"}, ResourceClass: "cpu.normalize",
		Retry: RetryPolicy{MaxAttempts: 2, BackoffMillis: 10},
	})
	require.NoError(t, err)
	ir := WorkflowIR{
		Schema: IRSchema, Name: "mapped-customers",
		SetInputs: []IRSetInput{{
			Name: "customers", ItemSchema: "customer/v1", ManifestSchema: ItemManifestSchemaV1,
		}},
		Maps: []IRMap{{
			Key: "normalize", Source: SetRef{
				Source: "set-input", Name: "customers", ItemSchema: "customer/v1",
				ManifestSchema: ItemManifestSchemaV1,
			},
			ItemTask: TaskKey{Kind: "cookbook.map.normalize-customer", Version: "v1"},
			Bindings: map[string]ValueRef{
				"customer": {Source: "map-item", MapKey: "normalize", Schema: "customer/v1"},
			},
			Policy: MapPolicy{PageSize: 64, MaxItems: 2000, MaxMaterializedAhead: 128},
		}},
		SetOutputs: []IRSetOutput{{Name: "customers", Value: SetRef{
			Source: "map-output", MapKey: "normalize", ItemSchema: "normalized-customer/v1",
			ManifestSchema: ItemManifestSchemaV1,
		}}},
	}
	first, err := Compile(ir, catalog)
	require.NoError(t, err)
	second, err := Compile(ir, catalog)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Len(t, first.Maps, 1)
	require.Equal(t, "cpu.normalize", first.Maps[0].ResourceClass)
	require.Equal(t, testBundleDigest, first.Maps[0].Implementation.BundleDigest)
	require.Equal(t, MapPolicy{PageSize: 64, MaxItems: 2000, MaxMaterializedAhead: 128}, first.Maps[0].Policy)
}

func TestValidateIRRejectsInvalidMapContracts(t *testing.T) {
	catalog, err := NewCatalog(TaskSpec{
		Identity: ImplementationIdentity{
			TaskKey: TaskKey{Kind: "map-item", Version: "v1"}, BundleDigest: testBundleDigest,
			Entrypoint: "tasks.cjs#run", ABI: TaskABI,
		},
		Inputs: map[string]string{"item": "item/v1"}, Outputs: map[string]string{"output": "result/v1"},
	})
	require.NoError(t, err)
	base := WorkflowIR{
		Schema: IRSchema, Name: "map",
		SetInputs: []IRSetInput{{Name: "items", ItemSchema: "item/v1", ManifestSchema: ItemManifestSchemaV1}},
		Maps: []IRMap{{
			Key: "mapped", Source: SetRef{Source: "set-input", Name: "items", ItemSchema: "item/v1", ManifestSchema: ItemManifestSchemaV1},
			ItemTask: TaskKey{Kind: "map-item", Version: "v1"},
			Bindings: map[string]ValueRef{"item": {Source: "map-item", MapKey: "mapped", Schema: "item/v1"}},
			Policy:   MapPolicy{PageSize: 10, MaxItems: 100, MaxMaterializedAhead: 20},
		}},
		SetOutputs: []IRSetOutput{{Name: "results", Value: SetRef{Source: "map-output", MapKey: "mapped", ItemSchema: "result/v1", ManifestSchema: ItemManifestSchemaV1}}},
	}
	invalidPolicy := base
	invalidPolicy.Maps = append([]IRMap(nil), base.Maps...)
	invalidPolicy.Maps[0].Policy.MaxMaterializedAhead = 1
	require.ErrorContains(t, ValidateIR(invalidPolicy, catalog), "invalid expansion policy")

	wrongOwner := base
	wrongOwner.Maps = append([]IRMap(nil), base.Maps...)
	wrongOwner.Maps[0].Bindings = cloneBindings(base.Maps[0].Bindings)
	wrongOwner.Maps[0].Bindings["item"] = ValueRef{Source: "map-item", MapKey: "other", Schema: "item/v1"}
	require.ErrorContains(t, ValidateIR(wrongOwner, catalog), "wrong item owner")

	wrongOutput := base
	wrongOutput.SetOutputs = []IRSetOutput{{Name: "results", Value: SetRef{Source: "map-output", MapKey: "mapped", ItemSchema: "wrong/v1", ManifestSchema: ItemManifestSchemaV1}}}
	require.ErrorContains(t, ValidateIR(wrongOutput, catalog), "schema mismatch")
}
