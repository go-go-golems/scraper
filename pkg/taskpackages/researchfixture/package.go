package researchfixture

import (
	_ "embed"

	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

const (
	Name                 = "research-runner-fixture"
	Version              = "1.0.0"
	OperationModuleAlias = "fixture:operation"
)

//go:embed transform.cjs
var transformSource []byte

//go:embed publish.cjs
var publishSource []byte

//go:embed workflow.js
var workflowSource string

type Package struct{}

func New() Package                        { return Package{} }
func (Package) Name() string              { return Name }
func (Package) Version() string           { return Version }
func (Package) RequiredModules() []string { return []string{"fs:input", OperationModuleAlias} }

func Bundle() (*workflowv3.Bundle, error) {
	return workflowv3.NewBundle(workflowv3.BundleManifest{
		Name: Name, Version: Version, ABI: workflowv3.TaskABI,
		Tasks: []workflowv3.BundleTask{
			{
				TaskKey:    workflowv3.TaskKey{Kind: "fixture.research.transform", Version: "v1"},
				Entrypoint: "transform.cjs#transform", Inputs: map[string]string{"source": "fixture-source/v1"},
				Outputs: map[string]string{"transformed": "fixture-transformed/v1"},
				Modules: []string{"fs:input", OperationModuleAlias}, ResourceClass: workflowv3.ResourceCPUDefault,
				Retry: workflowv3.RetryPolicy{MaxAttempts: 2, BackoffMillis: 5},
			},
			{
				TaskKey:    workflowv3.TaskKey{Kind: "fixture.research.publish", Version: "v1"},
				Entrypoint: "publish.cjs#publish", Inputs: map[string]string{"transformed": "fixture-transformed/v1"},
				Outputs: map[string]string{"result": "fixture-result/v1"},
				Modules: []string{"fs:input"}, ResourceClass: workflowv3.ResourceCPUDefault,
			},
		},
	}, map[string][]byte{"transform.cjs": transformSource, "publish.cjs": publishSource})
}

func (Package) Bundle() (*workflowv3.Bundle, error) { return Bundle() }

func DescriptorModule() workflowmodule.DescriptorModule {
	return workflowmodule.DescriptorModule{
		Name: "research-runner-fixture-tasks",
		Factories: map[string]workflowv3.TaskKey{
			"transform": {Kind: "fixture.research.transform", Version: "v1"},
			"publish":   {Kind: "fixture.research.publish", Version: "v1"},
		},
	}
}

func (Package) DescriptorModules() []workflowmodule.DescriptorModule {
	return []workflowmodule.DescriptorModule{DescriptorModule()}
}

func WorkflowSource() string { return workflowSource }
