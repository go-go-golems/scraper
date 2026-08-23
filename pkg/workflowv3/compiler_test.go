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
			Policy: SetInputPolicy{MaxItems: 2000},
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

func TestCompileReductionPinsBoundedHomogeneousTemplate(t *testing.T) {
	catalog, err := NewCatalog(
		TaskSpec{
			Identity: ImplementationIdentity{TaskKey: TaskKey{Kind: "count", Version: "v1"}, BundleDigest: testBundleDigest, Entrypoint: "tasks.cjs#count", ABI: TaskABI},
			Inputs:   map[string]string{"item": "document/v1"}, Outputs: map[string]string{"count": "word-count/v1"},
		},
		TaskSpec{
			Identity: ImplementationIdentity{TaskKey: TaskKey{Kind: "merge", Version: "v1"}, BundleDigest: testBundleDigest, Entrypoint: "tasks.cjs#merge", ABI: TaskABI},
			Inputs:   map[string]string{"partition": ReductionPartitionSchemaV1}, Outputs: map[string]string{"count": "word-count/v1"},
			ResourceClass: "cpu.reduce",
		},
		TaskSpec{
			Identity: ImplementationIdentity{TaskKey: TaskKey{Kind: "finalize", Version: "v1"}, BundleDigest: testBundleDigest, Entrypoint: "tasks.cjs#finalize", ABI: TaskABI},
			Inputs:   map[string]string{"count": "word-count/v1"}, Outputs: map[string]string{"receipt": "receipt/v1"},
		},
	)
	require.NoError(t, err)
	ir := WorkflowIR{
		Schema: IRSchema, Name: "word-count", Inputs: []IRInput{}, Nodes: []IRNode{{
			Key: "finalize", Task: TaskKey{Kind: "finalize", Version: "v1"},
			Bindings: map[string]ValueRef{"count": {Source: "reduction-output", ReduceKey: "merge-counts", Schema: "word-count/v1"}},
		}},
		SetInputs: []IRSetInput{{Name: "documents", ItemSchema: "document/v1", ManifestSchema: ItemManifestSchemaV1, Policy: SetInputPolicy{MaxItems: 100}}},
		Maps: []IRMap{{
			Key: "count-documents", Source: SetRef{Source: "set-input", Name: "documents", ItemSchema: "document/v1", ManifestSchema: ItemManifestSchemaV1},
			ItemTask: TaskKey{Kind: "count", Version: "v1"},
			Bindings: map[string]ValueRef{"item": {Source: "map-item", MapKey: "count-documents", Schema: "document/v1"}},
			Policy:   MapPolicy{PageSize: 16, MaxItems: 100, MaxMaterializedAhead: 32},
		}},
		Reductions: []IRReduce{{
			Key: "merge-counts", Source: SetRef{Source: "map-output", MapKey: "count-documents", ItemSchema: "word-count/v1", ManifestSchema: ItemManifestSchemaV1},
			PartitionTask: TaskKey{Kind: "merge", Version: "v1"},
			Bindings:      map[string]ValueRef{"partition": {Source: "reduction-partition", ReduceKey: "merge-counts", Schema: ReductionPartitionSchemaV1}},
			Policy:        ReducePolicy{FanIn: 8, MaxLevels: 4},
		}},
		Outputs: []IROutput{{Name: "count", Value: ValueRef{Source: "reduction-output", ReduceKey: "merge-counts", Schema: "word-count/v1"}}},
	}
	plan, err := Compile(ir, catalog)
	require.NoError(t, err)
	require.Len(t, plan.Reductions, 1)
	require.Equal(t, "cpu.reduce", plan.Reductions[0].ResourceClass)
	require.Equal(t, ReducePolicy{FanIn: 8, MaxLevels: 4}, plan.Reductions[0].Policy)
	require.Equal(t, "word-count/v1", plan.Outputs[0].Value.Schema)
	require.Equal(t, "reduction-output", plan.Nodes[0].Bindings["count"].Source)

	invalid := ir
	invalid.Reductions = append([]IRReduce(nil), ir.Reductions...)
	invalid.Reductions[0].Policy.FanIn = 1
	require.ErrorContains(t, ValidateIR(invalid, catalog), "invalid reduction policy")

	tooLarge := ir
	tooLarge.SetInputs = append([]IRSetInput(nil), ir.SetInputs...)
	tooLarge.SetInputs[0].Policy.MaxItems = 4097
	tooLarge.Maps = append([]IRMap(nil), ir.Maps...)
	tooLarge.Maps[0].Policy.MaxItems = 4097
	require.ErrorContains(t, ValidateIR(tooLarge, catalog), "capacity 4096 is smaller than source contract 4097")
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
		SetInputs: []IRSetInput{{Name: "items", ItemSchema: "item/v1", ManifestSchema: ItemManifestSchemaV1, Policy: SetInputPolicy{MaxItems: 100}}},
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

	missingInputBound := base
	missingInputBound.SetInputs = append([]IRSetInput(nil), base.SetInputs...)
	missingInputBound.SetInputs[0].Policy = SetInputPolicy{}
	require.ErrorContains(t, ValidateIR(missingInputBound, catalog), "max items must be positive")

	consumerTooSmall := base
	consumerTooSmall.SetInputs = append([]IRSetInput(nil), base.SetInputs...)
	consumerTooSmall.SetInputs[0].Policy.MaxItems = 101
	require.ErrorContains(t, ValidateIR(consumerTooSmall, catalog), "smaller than source contract")
}

func TestCompileAcceptsBoundedSetInputPassThrough(t *testing.T) {
	ir := WorkflowIR{
		Schema: IRSchema, Name: "set-pass-through",
		SetInputs: []IRSetInput{{
			Name: "items", ItemSchema: "item/v1", ManifestSchema: ItemManifestSchemaV1,
			Policy: SetInputPolicy{MaxItems: 10},
		}},
		SetOutputs: []IRSetOutput{{
			Name: "items", Value: SetRef{Source: "set-input", Name: "items", ItemSchema: "item/v1", ManifestSchema: ItemManifestSchemaV1},
		}},
	}
	plan, err := Compile(ir, testCatalog(t))
	require.NoError(t, err)
	require.Equal(t, 10, plan.SetInputs[0].Policy.MaxItems)
	require.Empty(t, plan.Nodes)
	require.Empty(t, plan.Maps)
	require.Empty(t, plan.Reductions)
}
