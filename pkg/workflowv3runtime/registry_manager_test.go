package workflowv3runtime

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	gggengine "github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3sqlite"
	"github.com/stretchr/testify/require"
)

func generationFixture(t *testing.T, version string) (*workflowv3.SealedRegistry, workflowv3.WorkflowPlan) {
	t.Helper()
	bundle, err := workflowv3.NewBundle(workflowv3.BundleManifest{
		Name: "generation-fixture", Version: version, ABI: workflowv3.TaskABI,
		Tasks: []workflowv3.BundleTask{{
			TaskKey:       workflowv3.TaskKey{Kind: "generation.transform", Version: "v1"},
			Entrypoint:    "tasks.cjs#transform",
			Inputs:        map[string]string{"input": "input/v1"},
			Outputs:       map[string]string{"output": "output/v1"},
			ResourceClass: "cpu.generation",
		}},
	}, map[string][]byte{"tasks.cjs": []byte(fmt.Sprintf(`
const task = require("workflow/task");
exports.transform = task.implementation(async ctx => {
  const output = await ctx.outputs.putJSON("output", {
    schema: "output/v1", value: {version: %q},
  });
  return task.success({output});
});`, version))})
	require.NoError(t, err)
	builder := workflowv3.NewRegistryBuilder()
	require.NoError(t, builder.AddBundle(bundle))
	registry, err := builder.Seal()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	plan, err := workflowv3.Compile(workflowv3.WorkflowIR{
		Schema: workflowv3.IRSchema, Name: "generation-" + version,
		Inputs: []workflowv3.IRInput{{Name: "input", Schema: "input/v1"}},
		Nodes: []workflowv3.IRNode{{
			Key: "transform", Task: workflowv3.TaskKey{Kind: "generation.transform", Version: "v1"},
			Bindings: map[string]workflowv3.ValueRef{"input": {Source: "input", Name: "input", Schema: "input/v1"}},
		}},
		Outputs: []workflowv3.IROutput{{Name: "output", Value: workflowv3.ValueRef{Source: "node-output", NodeKey: "transform", Port: "output", Schema: "output/v1"}}},
	}, catalog)
	require.NoError(t, err)
	return registry, plan
}

func TestRegistryManagerActivatesCoexistsDrainsAndRemoves(t *testing.T) {
	registryA, planA := generationFixture(t, "A")
	registryB, planB := generationFixture(t, "B")
	manager, err := NewRegistryManager(registryA)
	require.NoError(t, err)

	taskA, generationA, releaseA, err := manager.AcquireNode(planA.Nodes[0])
	require.NoError(t, err)
	require.Equal(t, registryA.Generation(), generationA)
	require.Equal(t, planA.Nodes[0].Implementation.BundleDigest, taskA.Spec.Identity.BundleDigest)

	require.NoError(t, manager.Activate(registryB, func(candidate *workflowv3.SealedRegistry) error {
		_, err := candidate.ResolveNode(planB.Nodes[0])
		return err
	}))
	snapshot := manager.Snapshot()
	require.Equal(t, registryB.Generation(), snapshot.Active)
	require.Equal(t, "draining", generationState(t, snapshot, registryA.Generation()).State)
	require.Equal(t, 1, generationState(t, snapshot, registryA.Generation()).References)
	require.Equal(t, "active", generationState(t, snapshot, registryB.Generation()).State)

	_, err = manager.ResolveNode(planA.Nodes[0])
	require.NoError(t, err)
	_, generationB, releaseB, err := manager.AcquireNode(planB.Nodes[0])
	require.NoError(t, err)
	require.Equal(t, registryB.Generation(), generationB)
	require.Error(t, manager.RemoveDrained(registryA.Generation()))
	releaseA()
	releaseA()
	require.NoError(t, manager.RemoveDrained(registryA.Generation()))
	_, err = manager.ResolveNode(planA.Nodes[0])
	require.Error(t, err)
	releaseB()
}

func TestRegistryManagerPinsDurableAttemptsAcrossActivation(t *testing.T) {
	ctx := context.Background()
	registryA, planA := generationFixture(t, "A")
	registryB, planB := generationFixture(t, "B")
	manager, err := NewRegistryManager(registryA)
	require.NoError(t, err)
	root := t.TempDir()
	store, err := workflowv3sqlite.Open(ctx, filepath.Join(root, "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(root, "artifacts"), 1<<20)
	require.NoError(t, err)
	modules, err := NewTaskModuleRegistry()
	require.NoError(t, err)
	engine := &Engine{Store: store, Registry: manager, Artifacts: artifacts, Modules: modules}
	input, err := artifacts.Put(ctx, "input/v1", "application/json", []byte(`{}`))
	require.NoError(t, err)
	now := time.Now().UTC()
	require.NoError(t, engine.Submit(ctx, "run-a", planA, map[string]workflowv3.ArtifactRef{"input": input}))
	leaseA, err := store.LeaseNext(ctx, manager, now, time.Minute)
	require.NoError(t, err)
	require.Equal(t, registryA.Generation(), leaseA.RegistryGeneration)
	require.NoError(t, manager.Activate(registryB, nil))
	require.NoError(t, engine.Submit(ctx, "run-a-pending", planA, map[string]workflowv3.ArtifactRef{"input": input}))
	pendingA, err := store.LeaseNext(ctx, manager, now.Add(time.Second), time.Minute)
	require.NoError(t, err)
	require.Equal(t, registryA.Generation(), pendingA.RegistryGeneration)
	require.NoError(t, engine.ExecuteLease(ctx, *pendingA))
	require.NoError(t, engine.ExecuteLease(ctx, *leaseA))
	require.NoError(t, engine.Submit(ctx, "run-b", planB, map[string]workflowv3.ArtifactRef{"input": input}))
	leaseB, err := store.LeaseNext(ctx, manager, now.Add(time.Second), time.Minute)
	require.NoError(t, err)
	require.Equal(t, registryB.Generation(), leaseB.RegistryGeneration)
	require.NoError(t, engine.ExecuteLease(ctx, *leaseB))
	snapshotA, err := store.Snapshot(ctx, "run-a")
	require.NoError(t, err)
	snapshotB, err := store.Snapshot(ctx, "run-b")
	require.NoError(t, err)
	require.Equal(t, registryA.Generation(), snapshotA.Attempts[0].RegistryGeneration)
	require.Equal(t, registryB.Generation(), snapshotB.Attempts[0].RegistryGeneration)
	bodyA, err := workflowv3.ReadArtifact(ctx, artifacts, snapshotA.Outputs["output"])
	require.NoError(t, err)
	bodyB, err := workflowv3.ReadArtifact(ctx, artifacts, snapshotB.Outputs["output"])
	require.NoError(t, err)
	require.JSONEq(t, `{"version":"A"}`, string(bodyA))
	require.JSONEq(t, `{"version":"B"}`, string(bodyB))
	require.NoError(t, manager.RemoveDrained(registryA.Generation()))
	require.NoError(t, store.CreateRun(
		ctx, "run-a-unavailable", planA,
		map[string]workflowv3.ArtifactRef{"input": input}, now.Add(2*time.Second),
	))
	queue, err := store.QueueSnapshot(ctx, manager, nil, now.Add(2*time.Second))
	require.NoError(t, err)
	require.Equal(t, 1, queue.BlockedByReason["implementation-unavailable"])
	restarted, err := NewRegistryManager(registryB)
	require.NoError(t, err)
	_, err = restarted.ResolveNode(planB.Nodes[0])
	require.NoError(t, err)
	_, err = restarted.ResolveNode(planA.Nodes[0])
	require.Error(t, err)
	require.Equal(t, registryB.Generation(), restarted.Snapshot().Active)
}

func TestRegistryManagerFailedCandidateIsAtomicAndQuarantineWithdrawsAdmission(t *testing.T) {
	registryA, _ := generationFixture(t, "A")
	registryB, planB := generationFixture(t, "B")
	manager, err := NewRegistryManager(registryA)
	require.NoError(t, err)
	err = manager.Activate(registryB, func(*workflowv3.SealedRegistry) error {
		return errors.New("self-test failed")
	})
	require.ErrorContains(t, err, "self-test failed")
	require.Equal(t, registryA.Generation(), manager.Snapshot().Active)

	require.NoError(t, manager.Activate(registryB, nil))
	quarantined, err := manager.RecordConstructionFailure(registryB.Generation(), "TASK_RUNTIME_CONSTRUCTION", 2)
	require.NoError(t, err)
	require.False(t, quarantined)
	quarantined, err = manager.RecordConstructionFailure(registryB.Generation(), "TASK_RUNTIME_CONSTRUCTION", 2)
	require.NoError(t, err)
	require.True(t, quarantined)
	_, err = manager.ResolveNode(planB.Nodes[0])
	require.Error(t, err)
	state := generationState(t, manager.Snapshot(), registryB.Generation())
	require.Equal(t, "quarantined", state.State)
	require.Equal(t, 2, state.Failures)
	require.Equal(t, "TASK_RUNTIME_CONSTRUCTION", state.QuarantineCode)
}

func TestRegistryManagerQuarantinesConstructionFailureWithoutDomainRetryDebt(t *testing.T) {
	ctx := context.Background()
	bundle, err := workflowv3.NewBundle(workflowv3.BundleManifest{
		Name: "broken-generation", Version: "1", ABI: workflowv3.TaskABI,
		Tasks: []workflowv3.BundleTask{{
			TaskKey:    workflowv3.TaskKey{Kind: "broken.task", Version: "v1"},
			Entrypoint: "tasks.cjs#run", Inputs: map[string]string{"input": "input/v1"},
			Outputs: map[string]string{"output": "output/v1"}, Modules: []string{"bad:module"},
			Retry: workflowv3.RetryPolicy{MaxAttempts: 1},
		}},
	}, map[string][]byte{"tasks.cjs": []byte(`exports.run = () => ({})`)})
	require.NoError(t, err)
	builder := workflowv3.NewRegistryBuilder()
	require.NoError(t, builder.AdvertiseModules("bad:module"))
	require.NoError(t, builder.AddBundle(bundle))
	registry, err := builder.Seal()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	plan, err := workflowv3.Compile(workflowv3.WorkflowIR{
		Schema: workflowv3.IRSchema, Name: "broken", Inputs: []workflowv3.IRInput{{Name: "input", Schema: "input/v1"}},
		Nodes:   []workflowv3.IRNode{{Key: "broken", Task: workflowv3.TaskKey{Kind: "broken.task", Version: "v1"}, Bindings: map[string]workflowv3.ValueRef{"input": {Source: "input", Name: "input", Schema: "input/v1"}}}},
		Outputs: []workflowv3.IROutput{{Name: "output", Value: workflowv3.ValueRef{Source: "node-output", NodeKey: "broken", Port: "output", Schema: "output/v1"}}},
	}, catalog)
	require.NoError(t, err)
	manager, err := NewRegistryManager(registry)
	require.NoError(t, err)
	modules, err := NewTaskModuleRegistry(TaskModuleFactory{
		Alias: "bad:module",
		Build: func(TaskModuleContext) (gggengine.RuntimeModuleRegistrar, error) {
			return nil, errors.New("module construction failed")
		},
	})
	require.NoError(t, err)
	root := t.TempDir()
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(root, "artifacts"), 1<<20)
	require.NoError(t, err)
	input, err := artifacts.Put(ctx, "input/v1", "application/json", []byte(`{}`))
	require.NoError(t, err)
	store, err := workflowv3sqlite.Open(ctx, filepath.Join(root, "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	engine := &Engine{
		Store: store, Registry: manager, Artifacts: artifacts, Modules: modules,
		RegistryQuarantineThreshold: 2,
	}
	require.NoError(t, engine.Submit(ctx, "broken", plan, map[string]workflowv3.ArtifactRef{"input": input}))
	for range 2 {
		ran, runErr := engine.RunOne(ctx)
		require.True(t, ran)
		var recorded *AttemptExecutionError
		require.ErrorAs(t, runErr, &recorded)
	}
	ran, err := engine.RunOne(ctx)
	require.NoError(t, err)
	require.False(t, ran)
	snapshot, err := engine.Snapshot(ctx, "broken")
	require.NoError(t, err)
	require.Equal(t, "running", snapshot.Status)
	require.Len(t, snapshot.Attempts, 2)
	for _, attempt := range snapshot.Attempts {
		require.NotNil(t, attempt.Failure)
		require.Equal(t, "TASK_RUNTIME_CONSTRUCTION", attempt.Failure.Code)
	}
	state := generationState(t, manager.Snapshot(), registry.Generation())
	require.Equal(t, "quarantined", state.State)
	dispatcher := &Dispatcher{
		Engine: engine, Capacities: map[string]int{workflowv3.ResourceCPUDefault: 1},
	}
	queue, err := dispatcher.QueueSnapshot(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, queue.BlockedByReason["implementation-unavailable"])
	require.Len(t, queue.RegistryGenerations, 1)
	require.Equal(t, "quarantined", queue.RegistryGenerations[0].State)
	operational, err := dispatcher.OperationalSnapshot(ctx, nil)
	require.NoError(t, err)
	require.Len(t, operational.RegistryGenerations, 1)
	require.Equal(t, "quarantined", operational.RegistryGenerations[0].State)
	require.Equal(t, operational.RegistryGenerations, operational.Queue.RegistryGenerations)
}

func TestRegistryManagerAcquireAndActivateAreRaceSafe(t *testing.T) {
	registryA, planA := generationFixture(t, "A")
	registryB, _ := generationFixture(t, "B")
	manager, err := NewRegistryManager(registryA)
	require.NoError(t, err)
	start := make(chan struct{})
	activationErrors := make(chan error, 1)
	var wait sync.WaitGroup
	for range 32 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, _, release, acquireErr := manager.AcquireNode(planA.Nodes[0])
			if acquireErr == nil {
				release()
			}
		}()
	}
	wait.Add(1)
	go func() {
		defer wait.Done()
		<-start
		activationErrors <- manager.Activate(registryB, nil)
	}()
	close(start)
	wait.Wait()
	require.NoError(t, <-activationErrors)
	snapshot := manager.Snapshot()
	require.Equal(t, registryB.Generation(), snapshot.Active)
	require.Equal(t, 0, generationState(t, snapshot, registryA.Generation()).References)
}

func generationState(t *testing.T, snapshot RegistryManagerSnapshot, generation string) RegistryGenerationSnapshot {
	t.Helper()
	for _, state := range snapshot.Generations {
		if state.Generation == generation {
			return state
		}
	}
	t.Fatalf("generation %s not found", generation)
	return RegistryGenerationSnapshot{}
}
