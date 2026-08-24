package workflowv3isolation

import (
	_ "embed"

	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

const ResourceClass = "cpu.isolated"

//go:embed tasks.cjs
var taskSource []byte

//go:embed workflow.js
var workflowSource string

func Policy() workflowv3.IsolationPolicy {
	return workflowv3.IsolationPolicy{
		Class:          workflowv3.IsolationSubprocessRestricted,
		WallTimeMillis: 10_000, CPUTimeMillis: 5_000, MemoryBytes: 8 << 30,
		MaxProcesses: 64, MaxOutputBytes: 1 << 20,
		MaxOutputFiles: 8, MaxProtocolBytes: 1 << 20,
	}
}

func Bundle() (*workflowv3.Bundle, error) {
	policy := Policy()
	return workflowv3.NewBundle(workflowv3.BundleManifest{
		Name: "isolation-fixture", Version: "1.0.0", ABI: workflowv3.TaskABI,
		Tasks: []workflowv3.BundleTask{{
			TaskKey:    workflowv3.TaskKey{Kind: "fixture.isolation.transform", Version: "v1"},
			Entrypoint: "tasks.cjs#transform", Inputs: map[string]string{"source": "isolation-source/v1"},
			Outputs: map[string]string{"output": "isolation-output/v1"}, Modules: []string{"fs:input"},
			ResourceClass: ResourceClass, IsolationMaximum: &policy,
		}, {
			TaskKey:    workflowv3.TaskKey{Kind: "fixture.isolation.spin", Version: "v1"},
			Entrypoint: "tasks.cjs#spin", Inputs: map[string]string{"source": "isolation-source/v1"},
			Outputs: map[string]string{"output": "isolation-output/v1"}, Modules: []string{"fs:input"},
			ResourceClass: ResourceClass, IsolationMaximum: &policy,
		}, {
			TaskKey:    workflowv3.TaskKey{Kind: "fixture.isolation.crash-retry", Version: "v1"},
			Entrypoint: "tasks.cjs#crashRetry", Inputs: map[string]string{"source": "isolation-source/v1"},
			Outputs: map[string]string{"output": "isolation-output/v1"}, Modules: []string{"exec:allowlisted", "fs:input"},
			ResourceClass: ResourceClass, Retry: workflowv3.RetryPolicy{MaxAttempts: 2}, IsolationMaximum: &policy,
		}, {
			TaskKey:    workflowv3.TaskKey{Kind: "fixture.isolation.tool", Version: "v1"},
			Entrypoint: "tasks.cjs#tool", Inputs: map[string]string{"source": "isolation-source/v1"},
			Outputs: map[string]string{"output": "isolation-output/v1"}, Modules: []string{"exec:allowlisted", "fs:input"},
			ResourceClass: ResourceClass, IsolationMaximum: &policy,
		}},
	}, map[string][]byte{"tasks.cjs": taskSource})
}

func Registry() (*workflowv3.SealedRegistry, error) {
	return RegistryWithExecutor("sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
}

func RegistryWithExecutor(executorDigest string) (*workflowv3.SealedRegistry, error) {
	bundle, err := Bundle()
	if err != nil {
		return nil, err
	}
	builder := workflowv3.NewRegistryBuilder()
	if err := builder.AdvertiseModules("exec:allowlisted", "fs:input"); err != nil {
		return nil, err
	}
	if err := builder.AdvertiseIsolationExecutor(workflowv3.IsolationSubprocessRestricted, executorDigest); err != nil {
		return nil, err
	}
	if err := builder.AddBundle(bundle); err != nil {
		return nil, err
	}
	return builder.Seal()
}

func WorkflowSource() string { return workflowSource }

func DescriptorModule() workflowmodule.DescriptorModule {
	return workflowmodule.DescriptorModule{Name: "isolation-fixture-tasks", Factories: map[string]workflowv3.TaskKey{
		"transform":  {Kind: "fixture.isolation.transform", Version: "v1"},
		"spin":       {Kind: "fixture.isolation.spin", Version: "v1"},
		"crashRetry": {Kind: "fixture.isolation.crash-retry", Version: "v1"},
		"tool":       {Kind: "fixture.isolation.tool", Version: "v1"},
	}}
}
