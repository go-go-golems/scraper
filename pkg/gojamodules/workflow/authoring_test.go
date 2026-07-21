package workflowmodule_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/testfixtures/workflowv3database"
	"github.com/go-go-golems/scraper/pkg/testfixtures/workflowv3http"
	"github.com/go-go-golems/scraper/pkg/testfixtures/workflowv3linear"
	"github.com/go-go-golems/scraper/pkg/testfixtures/workflowv3map"
	"github.com/go-go-golems/scraper/pkg/testfixtures/workflowv3reduce"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/stretchr/testify/require"
)

func linearCatalog(t *testing.T) *workflowv3.Catalog {
	t.Helper()
	registry, err := workflowv3linear.Registry()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	return catalog
}

func TestAuthorCompilesMinimalWorkflowToGoldens(t *testing.T) {
	result, err := workflowmodule.Author(
		context.Background(),
		workflowv3linear.WorkflowSource(),
		linearCatalog(t),
		workflowv3linear.DescriptorModule(),
	)
	require.NoError(t, err)

	bundle, err := workflowv3linear.Bundle()
	require.NoError(t, err)
	require.Equal(t, directLinearIR(), result.IR)
	assertGolden(t, "linear-transform.ir.json", result.IR)
	assertGolden(t, "linear-transform.plan.json", result.Plan)
	require.Len(t, result.Plan.Nodes, 2)
	require.Equal(t, bundle.Digest(), result.Plan.Nodes[0].Implementation.BundleDigest)
	require.Equal(t, []workflowv3.NodeKey{"normalize"}, result.Plan.Nodes[1].DependsOn)
}

func directLinearIR() workflowv3.WorkflowIR {
	return workflowv3.WorkflowIR{
		Schema: workflowv3.IRSchema,
		Name:   "linear-transform",
		Inputs: []workflowv3.IRInput{{
			Name: "source", Schema: "customer-jsonl-ref/v1",
		}},
		Nodes: []workflowv3.IRNode{
			{
				Key: "normalize",
				Task: workflowv3.TaskKey{
					Kind: "cookbook.linear.normalize-customers", Version: "v1",
				},
				Bindings: map[string]workflowv3.ValueRef{
					"source": {
						Source: "input", Name: "source", Schema: "customer-jsonl-ref/v1",
					},
				},
			},
			{
				Key: "validate",
				Task: workflowv3.TaskKey{
					Kind: "cookbook.linear.validate-dataset", Version: "v1",
				},
				Bindings: map[string]workflowv3.ValueRef{
					"dataset": {
						Source: "node-output", NodeKey: "normalize", Port: "dataset",
						Schema: "normalized-customers-ref/v1",
					},
				},
				DependsOn: []workflowv3.NodeKey{"normalize"},
			},
		},
		Outputs: []workflowv3.IROutput{{
			Name: "dataset",
			Value: workflowv3.ValueRef{
				Source: "node-output", NodeKey: "validate", Port: "validatedDataset",
				Schema: "validated-customers-ref/v1",
			},
		}},
	}
}

func TestAuthorCompilesHTTPAndDatabaseSlicesToGoldens(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		module      workflowmodule.DescriptorModule
		registry    func() (*workflowv3.SealedRegistry, error)
		planGolden  string
		irGolden    string
		resource    string
		modules     []string
		maxAttempts int
	}{
		{
			name: "http", source: workflowv3http.WorkflowSource(),
			module: workflowv3http.DescriptorModule(), registry: workflowv3http.Registry,
			planGolden: "http-snapshot.plan.json", irGolden: "http-snapshot.ir.json",
			resource: workflowv3http.ResourceClass,
			modules:  []string{"fetch:public", "fs:input"}, maxAttempts: 3,
		},
		{
			name: "database", source: workflowv3database.WorkflowSource(),
			module: workflowv3database.DescriptorModule(), registry: workflowv3database.Registry,
			planGolden: "database-sync.plan.json", irGolden: "database-sync.ir.json",
			resource: workflowv3database.ResourceClass,
			modules:  []string{"db:sync", "fs:input"}, maxAttempts: 3,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry, err := test.registry()
			require.NoError(t, err)
			catalog, err := registry.Catalog()
			require.NoError(t, err)
			result, err := workflowmodule.Author(
				context.Background(), test.source, catalog, test.module,
			)
			require.NoError(t, err)
			assertGolden(t, test.irGolden, result.IR)
			assertGolden(t, test.planGolden, result.Plan)
			require.Len(t, result.Plan.Nodes, 1)
			node := result.Plan.Nodes[0]
			require.Equal(t, test.resource, node.ResourceClass)
			require.Equal(t, test.modules, node.Modules)
			require.Equal(t, test.maxAttempts, node.Retry.MaxAttempts)
		})
	}
}

func TestAuthorCompilesLazyMapToCanonicalPlan(t *testing.T) {
	catalog, err := workflowv3.NewCatalog(workflowv3.TaskSpec{
		Identity: workflowv3.ImplementationIdentity{
			TaskKey:      workflowv3.TaskKey{Kind: "cookbook.map.normalize-customer", Version: "v1"},
			BundleDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Entrypoint:   "tasks.cjs#normalizeCustomer", ABI: workflowv3.TaskABI,
		},
		Inputs:        map[string]string{"customer": "customer/v1"},
		Outputs:       map[string]string{"normalized": "normalized-customer/v1"},
		ResourceClass: "cpu.normalize", Retry: workflowv3.RetryPolicy{MaxAttempts: 2, BackoffMillis: 10},
	})
	require.NoError(t, err)
	source := `
const workflow = require("workflow");
const tasks = require("customer-map-tasks");
let callbacks = 0;
const definition = workflow.define("mapped-customers", plan => {
  const customers = plan.inputSet("customers", {
    itemSchema: "customer/v1",
    manifestSchema: "scraper-workflow-item-manifest/v1",
  });
  const normalized = plan.map(
    "normalize",
    customers,
    customer => {
      callbacks += 1;
      return tasks.normalizeCustomer({customer});
    },
    map => map.pageSize(64).maxItems(2000).maxMaterializedAhead(128),
  );
  plan.outputSet("customers", normalized);
});
if (callbacks !== 1) throw new Error("map callback must execute exactly once");
module.exports = workflow.compile(definition);`
	result, err := workflowmodule.Author(
		context.Background(), source, catalog,
		workflowmodule.DescriptorModule{
			Name: "customer-map-tasks",
			Factories: map[string]workflowv3.TaskKey{
				"normalizeCustomer": {Kind: "cookbook.map.normalize-customer", Version: "v1"},
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, result.IR.Maps, 1)
	require.Equal(t, "map-item", result.IR.Maps[0].Bindings["customer"].Source)
	require.Equal(t, workflowv3.MapPolicy{
		PageSize: 64, MaxItems: 2000, MaxMaterializedAhead: 128,
	}, result.IR.Maps[0].Policy)
	assertGolden(t, "lazy-map.ir.json", result.IR)
	assertGolden(t, "lazy-map.plan.json", result.Plan)
}

func TestAuthorCompilesRealLazyMapFixtureToGoldens(t *testing.T) {
	registry, err := workflowv3map.Registry()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	result, err := workflowmodule.Author(
		context.Background(), workflowv3map.WorkflowSource(), catalog,
		workflowv3map.DescriptorModule(),
	)
	require.NoError(t, err)
	assertGolden(t, "lazy-map-transform.ir.json", result.IR)
	assertGolden(t, "lazy-map-transform.plan.json", result.Plan)
	require.Len(t, result.Plan.Maps, 1)
	require.Equal(t, workflowv3map.ResourceClass, result.Plan.Maps[0].ResourceClass)
}

func TestAuthorCompilesBoundedReductionToGoldens(t *testing.T) {
	const bundleDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	catalog, err := workflowv3.NewCatalog(
		workflowv3.TaskSpec{
			Identity: workflowv3.ImplementationIdentity{TaskKey: workflowv3.TaskKey{Kind: "count", Version: "v1"}, BundleDigest: bundleDigest, Entrypoint: "tasks.cjs#count", ABI: workflowv3.TaskABI},
			Inputs:   map[string]string{"document": "document/v1"}, Outputs: map[string]string{"count": "word-count/v1"},
		},
		workflowv3.TaskSpec{
			Identity: workflowv3.ImplementationIdentity{TaskKey: workflowv3.TaskKey{Kind: "merge", Version: "v1"}, BundleDigest: bundleDigest, Entrypoint: "tasks.cjs#merge", ABI: workflowv3.TaskABI},
			Inputs:   map[string]string{"partition": workflowv3.ReductionPartitionSchemaV1}, Outputs: map[string]string{"count": "word-count/v1"}, ResourceClass: "cpu.reduce",
		},
	)
	require.NoError(t, err)
	source := `
const workflow = require("workflow");
const tasks = require("word-count-tasks");
let reduceCallbacks = 0;
module.exports = workflow.compile(workflow.define("word-count", plan => {
  const documents = plan.inputSet("documents", {
    itemSchema: "document/v1",
    manifestSchema: "scraper-workflow-item-manifest/v1",
  });
  const counts = plan.map("count-documents", documents,
    document => tasks.count({document}),
    map => map.pageSize(16).maxItems(100).maxMaterializedAhead(32));
  const total = plan.reduce("merge-counts", counts, partition => {
    reduceCallbacks += 1;
    return tasks.merge({partition});
  }, reduce => reduce.fanIn(8).maxLevels(4));
  plan.output("count", total);
}));
if (reduceCallbacks !== 1) throw new Error("reduce callback must run once");`
	result, err := workflowmodule.Author(
		context.Background(), source, catalog,
		workflowmodule.DescriptorModule{Name: "word-count-tasks", Factories: map[string]workflowv3.TaskKey{
			"count": {Kind: "count", Version: "v1"}, "merge": {Kind: "merge", Version: "v1"},
		}},
	)
	require.NoError(t, err)
	require.Len(t, result.IR.Reductions, 1)
	require.Equal(t, workflowv3.ReducePolicy{FanIn: 8, MaxLevels: 4}, result.IR.Reductions[0].Policy)
	assertGolden(t, "bounded-reduction.ir.json", result.IR)
	assertGolden(t, "bounded-reduction.plan.json", result.Plan)
}

func TestAuthorCompilesRealBoundedReductionFixtureToGoldens(t *testing.T) {
	registry, err := workflowv3reduce.Registry()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	result, err := workflowmodule.Author(
		context.Background(), workflowv3reduce.WorkflowSource(), catalog,
		workflowv3reduce.DescriptorModule(),
	)
	require.NoError(t, err)
	assertGolden(t, "bounded-word-count.ir.json", result.IR)
	assertGolden(t, "bounded-word-count.plan.json", result.Plan)
	require.Len(t, result.Plan.Reductions, 1)
	require.Equal(t, workflowv3reduce.ReduceResource, result.Plan.Reductions[0].ResourceClass)
}

func TestAuthorRejectsInvalidLazyMapHandles(t *testing.T) {
	catalog, err := workflowv3.NewCatalog(workflowv3.TaskSpec{
		Identity: workflowv3.ImplementationIdentity{
			TaskKey:      workflowv3.TaskKey{Kind: "map-item", Version: "v1"},
			BundleDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Entrypoint:   "tasks.cjs#run", ABI: workflowv3.TaskABI,
		},
		Inputs: map[string]string{"item": "item/v1"}, Outputs: map[string]string{"output": "result/v1"},
	})
	require.NoError(t, err)
	module := workflowmodule.DescriptorModule{
		Name: "map-tasks", Factories: map[string]workflowv3.TaskKey{
			"run": {Kind: "map-item", Version: "v1"},
		},
	}
	_, err = workflowmodule.Author(context.Background(), `
const workflow = require("workflow");
const tasks = require("map-tasks");
module.exports = workflow.compile(workflow.define("bad", p => {
  const item = p.input("item", {schema: "item/v1"});
  const mapped = p.map("mapped", item, value => tasks.run({item: value}));
  p.outputSet("results", mapped);
}));`, catalog, module)
	require.ErrorContains(t, err, "map requires a key, set, and item task callback")

	_, err = workflowmodule.Author(context.Background(), `
const workflow = require("workflow");
const tasks = require("map-tasks");
module.exports = workflow.compile(workflow.define("bad", p => {
  const items = p.inputSet("items", {
    itemSchema: "item/v1",
    manifestSchema: "scraper-workflow-item-manifest/v1",
  });
  const mapped = p.map("mapped", items, value => ({value}));
  p.outputSet("results", mapped);
}));`, catalog, module)
	require.ErrorContains(t, err, "map item callback must return a task descriptor")
}

func TestAuthorRejectsUnknownTaskInput(t *testing.T) {
	source := `
const workflow = require("workflow");
const tasks = require("cookbook-linear-transform-tasks");
module.exports = workflow.compile(workflow.define("bad", p => {
  const source = p.input("source", {schema: "customer-jsonl-ref/v1"});
  const node = p.task("normalize", tasks.normalizeCustomers({source, extra: source}));
  p.output("dataset", node.output("dataset"));
}));`
	_, err := workflowmodule.Author(
		context.Background(), source, linearCatalog(t), workflowv3linear.DescriptorModule(),
	)
	require.ErrorContains(t, err, "unknown input extra")
}

func TestTypeScriptDeclaresMinimalSurface(t *testing.T) {
	declaration := workflowmodule.TypeScript()
	expectedDeclaration, err := os.ReadFile("testdata/workflow.d.ts")
	require.NoError(t, err)
	require.Equal(t, string(expectedDeclaration), declaration)
	for _, expected := range []string{
		"declare module \"workflow\"", "function define", "function compile",
		"input<T = unknown>", "inputSet<T = unknown>", "map<I, O>(", "reduce<I, O>(",
		"outputSet(name: string", "task(", "output(name: string",
	} {
		require.Contains(t, declaration, expected)
	}
	require.NotContains(t, declaration, ": any")
}

func assertGolden(t *testing.T, name string, value any) {
	t.Helper()
	body, err := workflowv3.CanonicalJSON(value)
	require.NoError(t, err)
	body = append(body, '\n')
	path := filepath.Join("testdata", name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		require.NoError(t, os.WriteFile(path, body, 0o644))
	}
	expected, err := os.ReadFile(path)
	require.NoError(t, err, "run UPDATE_GOLDEN=1 go test ./pkg/gojamodules/workflow")
	require.Equal(t, string(expected), string(body))
}
