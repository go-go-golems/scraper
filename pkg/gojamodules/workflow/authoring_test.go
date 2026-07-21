package workflowmodule

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/stretchr/testify/require"
)

const bundleDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func linearCatalog(t *testing.T) *workflowv3.Catalog {
	t.Helper()
	catalog, err := workflowv3.NewCatalog(
		workflowv3.TaskSpec{
			Identity: workflowv3.ImplementationIdentity{
				TaskKey: workflowv3.TaskKey{
					Kind: "cookbook.linear.normalize-customers", Version: "v1",
				},
				BundleDigest: bundleDigest,
				Entrypoint:   "./execution/tasks.cjs#normalizeCustomers",
				ABI:          workflowv3.TaskABI,
			},
			Inputs:  map[string]string{"source": "customer-jsonl-ref/v1"},
			Outputs: map[string]string{"dataset": "normalized-customers-ref/v1"},
			Modules: []string{"fs:input"},
		},
		workflowv3.TaskSpec{
			Identity: workflowv3.ImplementationIdentity{
				TaskKey: workflowv3.TaskKey{
					Kind: "cookbook.linear.validate-dataset", Version: "v1",
				},
				BundleDigest: bundleDigest,
				Entrypoint:   "./execution/tasks.cjs#validateDataset",
				ABI:          workflowv3.TaskABI,
			},
			Inputs:  map[string]string{"dataset": "normalized-customers-ref/v1"},
			Outputs: map[string]string{"validatedDataset": "validated-customers-ref/v1"},
			Modules: []string{"fs:input"},
		},
	)
	require.NoError(t, err)
	return catalog
}

func linearDescriptorModule() DescriptorModule {
	return DescriptorModule{
		Name: "cookbook-linear-transform-tasks",
		Factories: map[string]workflowv3.TaskKey{
			"normalizeCustomers": {
				Kind: "cookbook.linear.normalize-customers", Version: "v1",
			},
			"validateDataset": {
				Kind: "cookbook.linear.validate-dataset", Version: "v1",
			},
		},
	}
}

func TestAuthorCompilesMinimalWorkflowToGoldens(t *testing.T) {
	source, err := os.ReadFile("testdata/linear-transform.js")
	require.NoError(t, err)

	result, err := Author(
		context.Background(), string(source), linearCatalog(t), linearDescriptorModule(),
	)
	require.NoError(t, err)

	require.Equal(t, directLinearIR(), result.IR)
	assertGolden(t, "linear-transform.ir.json", result.IR)
	assertGolden(t, "linear-transform.plan.json", result.Plan)
	require.Len(t, result.Plan.Nodes, 2)
	require.Equal(t, bundleDigest, result.Plan.Nodes[0].Implementation.BundleDigest)
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

func TestAuthorRejectsUnknownTaskInput(t *testing.T) {
	source := `
const workflow = require("workflow");
const tasks = require("cookbook-linear-transform-tasks");
module.exports = workflow.compile(workflow.define("bad", p => {
  const source = p.input("source", {schema: "customer-jsonl-ref/v1"});
  const node = p.task("normalize", tasks.normalizeCustomers({source, extra: source}));
  p.output("dataset", node.output("dataset"));
}));`
	_, err := Author(context.Background(), source, linearCatalog(t), linearDescriptorModule())
	require.ErrorContains(t, err, "unknown input extra")
}

func TestTypeScriptDeclaresMinimalSurface(t *testing.T) {
	declaration := TypeScript()
	for _, expected := range []string{
		"declare module \"workflow\"", "function define", "function compile",
		"input(name: string", "task(", "output(name: string",
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
