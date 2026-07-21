package workflowmodule_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/testfixtures/workflowv3linear"
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
