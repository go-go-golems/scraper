package workflowv3reduce

import (
	_ "embed"

	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

const (
	MapResource    = "cpu.word-count"
	ReduceResource = "cpu.reduce"
)

//go:embed tasks.cjs
var taskSource []byte

//go:embed workflow.js
var workflowSource string

func Bundle() (*workflowv3.Bundle, error) {
	return workflowv3.NewBundle(workflowv3.BundleManifest{
		Name: "cookbook-word-count-tasks", Version: "1.0.0", ABI: workflowv3.TaskABI,
		Tasks: []workflowv3.BundleTask{
			{
				TaskKey:    workflowv3.TaskKey{Kind: "cookbook.word-count.count", Version: "v1"},
				Entrypoint: "execution/tasks.cjs#countWords",
				Inputs:     map[string]string{"document": "word-document/v1"},
				Outputs:    map[string]string{"count": "word-count/v1"},
				Modules:    []string{"fs:input"}, ResourceClass: MapResource,
			},
			{
				TaskKey:    workflowv3.TaskKey{Kind: "cookbook.word-count.merge", Version: "v1"},
				Entrypoint: "execution/tasks.cjs#mergeCounts",
				Inputs:     map[string]string{"partition": workflowv3.ReductionPartitionSchemaV1},
				Outputs:    map[string]string{"count": "word-count/v1"},
				Modules:    []string{"fs:input"}, ResourceClass: ReduceResource,
				Retry: workflowv3.RetryPolicy{MaxAttempts: 2, BackoffMillis: 5},
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
		Name: "cookbook-word-count-tasks",
		Factories: map[string]workflowv3.TaskKey{
			"countWords":  {Kind: "cookbook.word-count.count", Version: "v1"},
			"mergeCounts": {Kind: "cookbook.word-count.merge", Version: "v1"},
		},
	}
}

func WorkflowSource() string { return workflowSource }
