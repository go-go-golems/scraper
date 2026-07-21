package workflowv3budget

import (
	_ "embed"

	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

const ResourceClass = "network.budget-fixture"

//go:embed tasks.cjs
var taskSource []byte

//go:embed workflow.js
var workflowSource string

func Bundle() (*workflowv3.Bundle, error) {
	return workflowv3.NewBundle(workflowv3.BundleManifest{
		Name: "budget-fixture-tasks", Version: "1.0.0", ABI: workflowv3.TaskABI,
		Tasks: []workflowv3.BundleTask{{
			TaskKey:    workflowv3.TaskKey{Kind: "fixture.budget.invoke", Version: "v1"},
			Entrypoint: "tasks.cjs#invoke", Inputs: map[string]string{"request": "budget-request/v1"},
			Outputs: map[string]string{"response": "budget-response/v1"}, Modules: []string{"fs:input"},
			ResourceClass: ResourceClass, Retry: workflowv3.RetryPolicy{MaxAttempts: 2},
			BudgetMaximum: &workflowv3.BudgetClaim{
				Account: "provider", OnExhausted: workflowv3.BudgetExhaustBlock,
				Reserve: []workflowv3.BudgetAmount{{Dimension: "output_tokens", Units: 5}, {Dimension: "requests", Units: 1}},
			},
		}},
	}, map[string][]byte{"tasks.cjs": taskSource})
}

func Registry() (*workflowv3.SealedRegistry, error) {
	bundle, err := Bundle()
	if err != nil {
		return nil, err
	}
	builder := workflowv3.NewRegistryBuilder()
	if err := builder.AdvertiseModules("fs:input"); err != nil {
		return nil, err
	}
	if err := builder.AddBundle(bundle); err != nil {
		return nil, err
	}
	return builder.Seal()
}

func DescriptorModule() workflowmodule.DescriptorModule {
	return workflowmodule.DescriptorModule{
		Name: "budget-fixture-tasks",
		Factories: map[string]workflowv3.TaskKey{
			"invoke": {Kind: "fixture.budget.invoke", Version: "v1"},
		},
	}
}

func WorkflowSource() string { return workflowSource }
