package workflowv3database

import (
	_ "embed"

	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

//go:embed tasks.cjs
var taskSource []byte

//go:embed workflow.js
var workflowSource string

const (
	ResourceClass = "database.sync.primary"
	DatabaseAlias = "db:sync"
)

func Bundle() (*workflowv3.Bundle, error) {
	return workflowv3.NewBundle(workflowv3.BundleManifest{
		Name: "cookbook-database-sync-tasks", Version: "1.0.0", ABI: workflowv3.TaskABI,
		Tasks: []workflowv3.BundleTask{{
			TaskKey: workflowv3.TaskKey{
				Kind: "cookbook.database.synchronize-customers", Version: "v1",
			},
			Entrypoint: "tasks.cjs#synchronizeCustomers",
			Inputs: map[string]string{
				"dataset": "database-sync-dataset-ref/v1",
			},
			Outputs: map[string]string{
				"receipt": "database-sync-receipt-ref/v1",
			},
			Modules:       []string{"fs:input", DatabaseAlias},
			ResourceClass: ResourceClass,
			Retry: workflowv3.RetryPolicy{
				MaxAttempts: 3, BackoffMillis: 10,
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
	if err := builder.AdvertiseModules("fs:input", DatabaseAlias); err != nil {
		return nil, err
	}
	if err := builder.AddBundle(bundle); err != nil {
		return nil, err
	}
	return builder.Seal()
}

func DescriptorModule() workflowmodule.DescriptorModule {
	return workflowmodule.DescriptorModule{
		Name: "cookbook-database-sync-tasks",
		Factories: map[string]workflowv3.TaskKey{
			"synchronizeCustomers": {
				Kind: "cookbook.database.synchronize-customers", Version: "v1",
			},
		},
	}
}

func WorkflowSource() string { return workflowSource }
