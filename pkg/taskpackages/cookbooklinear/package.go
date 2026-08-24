package cookbooklinear

import (
	_ "embed"

	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

const (
	Name    = "cookbook-linear"
	Version = "1.0.0"
)

type Package struct{}

func New() Package { return Package{} }

func (Package) Name() string    { return Name }
func (Package) Version() string { return Version }

//go:embed tasks.cjs
var taskSource []byte

//go:embed workflow.js
var workflowSource string

func Bundle() (*workflowv3.Bundle, error) {
	return workflowv3.NewBundle(workflowv3.BundleManifest{
		Name: "cookbook-linear-transform-tasks", Version: "1.0.0", ABI: workflowv3.TaskABI,
		Tasks: []workflowv3.BundleTask{
			{
				TaskKey: workflowv3.TaskKey{
					Kind: "cookbook.linear.normalize-customers", Version: "v1",
				},
				Entrypoint: "execution/tasks.cjs#normalizeCustomers",
				Inputs:     map[string]string{"source": "customer-jsonl-ref/v1"},
				Outputs:    map[string]string{"dataset": "normalized-customers-ref/v1"},
				Modules:    []string{"fs:input"},
			},
			{
				TaskKey: workflowv3.TaskKey{
					Kind: "cookbook.linear.validate-dataset", Version: "v1",
				},
				Entrypoint: "execution/tasks.cjs#validateDataset",
				Inputs:     map[string]string{"dataset": "normalized-customers-ref/v1"},
				Outputs:    map[string]string{"validatedDataset": "validated-customers-ref/v1"},
				Modules:    []string{"fs:input"},
			},
		},
	}, map[string][]byte{"execution/tasks.cjs": taskSource})
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

func WorkflowSource() string {
	return workflowSource
}

func (Package) Bundle() (*workflowv3.Bundle, error) { return Bundle() }

func (Package) DescriptorModules() []workflowmodule.DescriptorModule {
	return []workflowmodule.DescriptorModule{DescriptorModule()}
}

func (Package) RequiredModules() []string { return []string{"fs:input"} }
