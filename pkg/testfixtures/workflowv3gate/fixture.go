package workflowv3gate

import (
	_ "embed"

	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

const ResourceClass = "cpu.gate-fixture"

//go:embed tasks.cjs
var taskSource []byte

//go:embed workflow.js
var workflowSource string

//go:embed independent.js
var independentSource string

func Bundle() (*workflowv3.Bundle, error) {
	return workflowv3.NewBundle(workflowv3.BundleManifest{
		Name: "gate-fixture-tasks", Version: "1.0.0", ABI: workflowv3.TaskABI,
		Tasks: []workflowv3.BundleTask{{
			TaskKey:    workflowv3.TaskKey{Kind: "fixture.gate.prepare", Version: "v1"},
			Entrypoint: "tasks.cjs#prepare", Inputs: map[string]string{"source": "gate-source/v1"},
			Outputs: map[string]string{"prepared": "gate-prepared/v1"}, Modules: []string{"fs:input"}, ResourceClass: ResourceClass,
		}, {
			TaskKey:    workflowv3.TaskKey{Kind: "fixture.gate.publish", Version: "v1"},
			Entrypoint: "tasks.cjs#publish", Inputs: map[string]string{"decision": "gate-decision/v1"},
			Outputs: map[string]string{"published": "gate-published/v1"}, Modules: []string{"fs:input"}, ResourceClass: ResourceClass,
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
		Name: "gate-fixture-tasks",
		Factories: map[string]workflowv3.TaskKey{
			"prepare": {Kind: "fixture.gate.prepare", Version: "v1"},
			"publish": {Kind: "fixture.gate.publish", Version: "v1"},
		},
	}
}

func WorkflowSource() string    { return workflowSource }
func IndependentSource() string { return independentSource }
