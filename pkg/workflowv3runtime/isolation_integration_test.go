package workflowv3runtime

import (
	"context"
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/testfixtures/workflowv3isolation"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3sqlite"
	"github.com/stretchr/testify/require"
)

func buildIsolatedWorker(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	directory := t.TempDir()
	worker := filepath.Join(directory, "workflowv3-task-worker")
	launcher := filepath.Join(directory, "workflowv3-isolation-launcher")
	for target, pkg := range map[string]string{
		worker: "./cmd/workflowv3-task-worker", launcher: "./cmd/workflowv3-isolation-launcher",
	} {
		command := exec.Command("go", "build", "-trimpath", "-o", target, pkg)
		command.Dir = root
		command.Env = append(os.Environ(), "CGO_ENABLED=0", "GOWORK=off")
		output, buildErr := command.CombinedOutput()
		require.NoError(t, buildErr, string(output))
	}
	for _, path := range []string{worker, launcher} {
		binary, openErr := elf.Open(path)
		require.NoError(t, openErr)
		for _, program := range binary.Progs {
			require.NotEqual(t, elf.PT_INTERP, program.Type, "isolation executables must not depend on host dynamic libraries")
		}
		require.NoError(t, binary.Close())
	}
	return worker
}

func buildFixtureTool(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "main.go")
	require.NoError(t, os.WriteFile(source, []byte(`package main
import ("fmt"; "net"; "os"; "time")
func main() {
  if len(os.Args) != 2 || os.Getenv("HOME") != "" || os.Getenv("PATH") != "/nonexistent" || os.Getenv("PRIVATE_ISOLATION_ENV_CANARY") != "" { os.Exit(2) }
  if _, err := os.Stat("/etc/passwd"); !os.IsNotExist(err) { os.Exit(3) }
  connection, err := net.DialTimeout("tcp", "1.1.1.1:53", 50*time.Millisecond)
  if err == nil { connection.Close(); os.Exit(4) }
  fmt.Print("tool-ok:" + os.Args[1])
}
`), 0o600))
	tool := filepath.Join(root, "fixture-tool")
	command := exec.Command("go", "build", "-trimpath", "-o", tool, source)
	command.Env = append(os.Environ(), "CGO_ENABLED=0", "GOWORK=off")
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	return tool
}

func buildForkingTool(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "forking-tool.c")
	require.NoError(t, os.WriteFile(source, []byte(`#include <errno.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>
int main(void) {
  pid_t children[32]; int count = 0;
  for (int index = 0; index < 32; index++) {
    pid_t child = fork();
    if (child < 0) {
      for (int i = 0; i < count; i++) kill(children[i], SIGKILL);
      for (int i = 0; i < count; i++) waitpid(children[i], NULL, 0);
      if (errno == EAGAIN) { fputs("process-limit-enforced", stdout); return 0; }
      return 6;
    }
    if (child == 0) { sleep(5); _exit(0); }
    children[count++] = child;
  }
  for (int i = 0; i < count; i++) kill(children[i], SIGKILL);
  for (int i = 0; i < count; i++) waitpid(children[i], NULL, 0);
  return 5;
}
`), 0o600))
	tool := filepath.Join(root, "forking-tool")
	command := exec.Command("cc", "-static", "-O2", "-o", tool, source)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	return tool
}

func buildMemoryTool(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "memory-tool.c")
	require.NoError(t, os.WriteFile(source, []byte(`#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
int main(void) {
  size_t size = 512UL * 1024UL * 1024UL;
  char *memory = malloc(size);
  if (!memory) return 7;
  for (size_t offset = 0; offset < size; offset += 4096) memory[offset] = 1;
  fputs("memory-limit-failed", stdout);
  return 0;
}
`), 0o600))
	tool := filepath.Join(root, "memory-tool")
	command := exec.Command("cc", "-static", "-O2", "-o", tool, source)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	return tool
}

func buildProtocolFaultWorker(t *testing.T, oversized bool) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "fault-worker.c")
	program := `#include <stdio.h>
int main(void) { fputs("not-json\n", stdout); return 0; }
`
	if oversized {
		program = `#include <stdio.h>
int main(void) { for (int i = 0; i < 2000000; i++) fputc('x', stdout); fputc('\n', stdout); return 0; }
`
	}
	require.NoError(t, os.WriteFile(source, []byte(program), 0o600))
	worker := filepath.Join(root, "fault-worker")
	command := exec.Command("cc", "-static", "-O2", "-o", worker, source)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	if oversized {
		probe, probeErr := exec.Command(worker).Output()
		require.NoError(t, probeErr)
		require.Greater(t, len(probe), 1<<20)
	}
	return worker
}

func TestRestrictedProtocolRejectsMalformedAndOversizedWorkerFrames(t *testing.T) {
	ctx := context.Background()
	goodWorker := buildIsolatedWorker(t)
	launcher := filepath.Join(filepath.Dir(goodWorker), "workflowv3-isolation-launcher")
	registry, err := workflowv3isolation.Registry()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	spec, ok := catalog.Lookup(workflowv3.TaskKey{Kind: "fixture.isolation.transform", Version: "v1"})
	require.True(t, ok)
	registered, err := registry.Resolve(spec.Identity)
	require.NoError(t, err)
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(t.TempDir(), "artifacts"), 1<<20)
	require.NoError(t, err)
	input, err := artifacts.Put(ctx, "isolation-source/v1", "application/json", []byte(`{"values":[]}`))
	require.NoError(t, err)
	modules, err := NewTaskModuleRegistry(FSInputModule())
	require.NoError(t, err)
	baseRequest := TaskRequest{RunID: "protocol", NodeKey: "transform", Attempt: 1, Task: registered, Inputs: map[string]workflowv3.ArtifactRef{"source": input}, Artifacts: artifacts, Modules: modules}
	for _, test := range []struct {
		name      string
		oversized bool
		code      string
	}{
		{name: "malformed", code: "ISOLATION_FRAME_INVALID"},
		{name: "oversized", oversized: true, code: "ISOLATION_FRAME_INVALID"},
	} {
		t.Run(test.name, func(t *testing.T) {
			executor := &BubblewrapExecutor{WorkerExecutable: buildProtocolFaultWorker(t, test.oversized), LauncherExecutable: launcher, BubblewrapExecutable: "/usr/bin/bwrap"}
			digest, err := executor.Identity()
			require.NoError(t, err)
			request := baseRequest
			request.Task.Spec.IsolationExecutorDigest = digest
			isolation, err := workflowv3.CompileIsolation(nil, spec.IsolationMaximum, digest)
			require.NoError(t, err)
			require.Equal(t, int64(1<<20), isolation.Effective.MaxProtocolBytes)
			_, err = executor.Execute(ctx, request, isolation)
			var failure *TaskFailureError
			require.ErrorAs(t, err, &failure)
			require.Equal(t, "protocol", failure.Failure.Class)
			require.Equal(t, test.code, failure.Failure.Code)
		})
	}
}

func TestRestrictedAllowlistedExecUsesFixedToolWithoutShellAuthority(t *testing.T) {
	t.Setenv("PRIVATE_ISOLATION_ENV_CANARY", "must-not-enter-child")
	ctx := context.Background()
	worker := buildIsolatedWorker(t)
	tool := buildFixtureTool(t)
	executor := &BubblewrapExecutor{
		WorkerExecutable: worker, BubblewrapExecutable: "/usr/bin/bwrap",
		Tools: map[string]string{"fixture.echo": tool},
	}
	executorDigest, err := executor.Identity()
	require.NoError(t, err)
	registry, err := workflowv3isolation.RegistryWithExecutor(executorDigest)
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	spec, ok := catalog.Lookup(workflowv3.TaskKey{Kind: "fixture.isolation.tool", Version: "v1"})
	require.True(t, ok)
	registered, err := registry.Resolve(spec.Identity)
	require.NoError(t, err)
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(t.TempDir(), "artifacts"), 1<<20)
	require.NoError(t, err)
	input, err := artifacts.Put(ctx, "isolation-source/v1", "application/json", []byte(`{"values":[]}`))
	require.NoError(t, err)
	modules, err := NewTaskModuleRegistry(FSInputModule(), AllowlistedExecModule(map[string]string{"fixture.echo": tool}))
	require.NoError(t, err)
	request := TaskRequest{RunID: "tool", NodeKey: "tool", Attempt: 1, Task: registered, Inputs: map[string]workflowv3.ArtifactRef{"source": input}, Artifacts: artifacts, Modules: modules}
	isolation, err := workflowv3.CompileIsolation(nil, spec.IsolationMaximum, spec.IsolationExecutorDigest)
	require.NoError(t, err)
	result, err := executor.Execute(ctx, request, isolation)
	require.NoError(t, err)
	body, err := workflowv3.ReadArtifact(ctx, artifacts, result.Outputs["output"])
	require.NoError(t, err)
	require.JSONEq(t, `{"stdout":"tool-ok:hello"}`, string(body))

	forking := &BubblewrapExecutor{
		WorkerExecutable: worker, BubblewrapExecutable: "/usr/bin/bwrap",
		Tools: map[string]string{"fixture.echo": buildForkingTool(t)},
	}
	forkDigest, err := forking.Identity()
	require.NoError(t, err)
	forkRequest := request
	forkRequest.Task.Spec.IsolationExecutorDigest = forkDigest
	forkPolicy := workflowv3isolation.Policy()
	forkPolicy.MaxProcesses = 16
	forkIsolation, err := workflowv3.CompileIsolation(&forkPolicy, spec.IsolationMaximum, forkDigest)
	require.NoError(t, err)
	result, err = forking.Execute(ctx, forkRequest, forkIsolation)
	require.NoError(t, err)
	body, err = workflowv3.ReadArtifact(ctx, artifacts, result.Outputs["output"])
	require.NoError(t, err)
	require.JSONEq(t, `{"stdout":"process-limit-enforced"}`, string(body))

	memoryExecutor := &BubblewrapExecutor{
		WorkerExecutable: worker, BubblewrapExecutable: "/usr/bin/bwrap",
		Tools: map[string]string{"fixture.echo": buildMemoryTool(t)},
	}
	memoryDigest, err := memoryExecutor.Identity()
	require.NoError(t, err)
	memoryRequest := request
	memoryRequest.Task.Spec.IsolationExecutorDigest = memoryDigest
	memoryPolicy := workflowv3isolation.Policy()
	memoryPolicy.MemoryBytes = 128 << 20
	memoryIsolation, err := workflowv3.CompileIsolation(&memoryPolicy, spec.IsolationMaximum, memoryDigest)
	require.NoError(t, err)
	_, err = memoryExecutor.Execute(ctx, memoryRequest, memoryIsolation)
	var resourceFailure *TaskFailureError
	require.ErrorAs(t, err, &resourceFailure)
	require.Equal(t, "resource", resourceFailure.Failure.Class)
	require.Equal(t, "ISOLATION_MEMORY_LIMIT", resourceFailure.Failure.Code)

	withoutTool := &BubblewrapExecutor{WorkerExecutable: worker, BubblewrapExecutable: "/usr/bin/bwrap"}
	withoutToolDigest, err := withoutTool.Identity()
	require.NoError(t, err)
	withoutToolRequest := request
	withoutToolRequest.Task.Spec.IsolationExecutorDigest = withoutToolDigest
	withoutToolIsolation, err := workflowv3.CompileIsolation(nil, spec.IsolationMaximum, withoutToolDigest)
	require.NoError(t, err)
	_, err = withoutTool.Execute(ctx, withoutToolRequest, withoutToolIsolation)
	var failure *TaskFailureError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, "TASK_RUNTIME_CONSTRUCTION", failure.Failure.Code)
}

func buildCrashOnceTool(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "crash-once.c")
	require.NoError(t, os.WriteFile(source, []byte(`#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
int main(int argc, char **argv) {
  if (argc != 2) return 2;
  if (strcmp(argv[1], "1") == 0) { kill(getppid(), SIGKILL); usleep(50000); return 3; }
  fputs("recovered", stdout); return 0;
}
`), 0o600))
	tool := filepath.Join(root, "crash-once")
	command := exec.Command("cc", "-static", "-O2", "-o", tool, source)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	return tool
}

func TestIsolatedChildDeathRetriesWithFreshProcessAndImmutableAttempts(t *testing.T) {
	ctx := context.Background()
	worker := buildIsolatedWorker(t)
	tool := buildCrashOnceTool(t)
	executor := &BubblewrapExecutor{WorkerExecutable: worker, BubblewrapExecutable: "/usr/bin/bwrap", Tools: map[string]string{"fixture.echo": tool}}
	executorDigest, err := executor.Identity()
	require.NoError(t, err)
	registry, err := workflowv3isolation.RegistryWithExecutor(executorDigest)
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	plan, err := workflowv3.Compile(workflowv3.WorkflowIR{
		Schema: workflowv3.IRSchema, Name: "crash-retry",
		Inputs:  []workflowv3.IRInput{{Name: "source", Schema: "isolation-source/v1"}},
		Nodes:   []workflowv3.IRNode{{Key: "crash", Task: workflowv3.TaskKey{Kind: "fixture.isolation.crash-retry", Version: "v1"}, Bindings: map[string]workflowv3.ValueRef{"source": {Source: "input", Name: "source", Schema: "isolation-source/v1"}}}},
		Outputs: []workflowv3.IROutput{{Name: "output", Value: workflowv3.ValueRef{Source: "node-output", NodeKey: "crash", Port: "output", Schema: "isolation-output/v1"}}},
	}, catalog)
	require.NoError(t, err)
	root := t.TempDir()
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(root, "artifacts"), 1<<20)
	require.NoError(t, err)
	input, err := artifacts.Put(ctx, "isolation-source/v1", "application/json", []byte(`{"values":[]}`))
	require.NoError(t, err)
	store, err := workflowv3sqlite.Open(ctx, filepath.Join(root, "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	modules, err := NewTaskModuleRegistry(FSInputModule(), AllowlistedExecModule(map[string]string{"fixture.echo": tool}))
	require.NoError(t, err)
	engine := &Engine{Store: store, Registry: registry, Artifacts: artifacts, Modules: modules, Isolation: executor, LeaseDuration: 10 * time.Second}
	require.NoError(t, engine.Submit(ctx, "crash-retry", plan, map[string]workflowv3.ArtifactRef{"source": input}))
	ran, err := engine.RunOne(ctx)
	require.True(t, ran)
	require.Error(t, err)
	ran, err = engine.RunOne(ctx)
	require.NoError(t, err)
	require.True(t, ran)
	snapshot, err := engine.Snapshot(ctx, "crash-retry")
	require.NoError(t, err)
	require.Equal(t, "succeeded", snapshot.Status)
	require.Len(t, snapshot.Attempts, 2)
	require.Equal(t, "failed", snapshot.Attempts[0].Status)
	require.Equal(t, "ISOLATION_CHILD_EXIT", snapshot.Attempts[0].Failure.Code)
	require.Equal(t, "succeeded", snapshot.Attempts[1].Status)
	body, err := workflowv3.ReadArtifact(ctx, artifacts, snapshot.Outputs["output"])
	require.NoError(t, err)
	require.JSONEq(t, `{"stdout":"recovered"}`, string(body))
}

func TestCancelKillsIsolatedChildAndFencesStaleCompletion(t *testing.T) {
	ctx := context.Background()
	worker := buildIsolatedWorker(t)
	tool := buildFixtureTool(t)
	executor := &BubblewrapExecutor{WorkerExecutable: worker, BubblewrapExecutable: "/usr/bin/bwrap", Tools: map[string]string{"fixture.echo": tool}}
	digest, err := executor.Identity()
	require.NoError(t, err)
	registry, err := workflowv3isolation.RegistryWithExecutor(digest)
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	plan, err := workflowv3.Compile(workflowv3.WorkflowIR{
		Schema: workflowv3.IRSchema, Name: "cancel-isolated",
		Inputs:  []workflowv3.IRInput{{Name: "source", Schema: "isolation-source/v1"}},
		Nodes:   []workflowv3.IRNode{{Key: "spin", Task: workflowv3.TaskKey{Kind: "fixture.isolation.spin", Version: "v1"}, Bindings: map[string]workflowv3.ValueRef{"source": {Source: "input", Name: "source", Schema: "isolation-source/v1"}}}},
		Outputs: []workflowv3.IROutput{{Name: "output", Value: workflowv3.ValueRef{Source: "node-output", NodeKey: "spin", Port: "output", Schema: "isolation-output/v1"}}},
	}, catalog)
	require.NoError(t, err)
	root := t.TempDir()
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(root, "artifacts"), 1<<20)
	require.NoError(t, err)
	input, err := artifacts.Put(ctx, "isolation-source/v1", "application/json", []byte(`{"values":[]}`))
	require.NoError(t, err)
	store, err := workflowv3sqlite.Open(ctx, filepath.Join(root, "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	modules, err := NewTaskModuleRegistry(FSInputModule(), AllowlistedExecModule(map[string]string{"fixture.echo": tool}))
	require.NoError(t, err)
	engine := &Engine{Store: store, Registry: registry, Artifacts: artifacts, Modules: modules, Isolation: executor, LeaseDuration: 10 * time.Second}
	require.NoError(t, engine.Submit(ctx, "cancel-isolated", plan, map[string]workflowv3.ArtifactRef{"source": input}))
	lease, err := store.LeaseNext(ctx, registry, time.Now().UTC(), 10*time.Second)
	require.NoError(t, err)
	require.NotNil(t, lease)
	executeCtx, cancelExecution := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- engine.ExecuteLease(executeCtx, *lease) }()
	time.Sleep(200 * time.Millisecond)
	queue, err := store.QueueSnapshot(ctx, registry, map[string]int{workflowv3isolation.ResourceClass: 1}, time.Now().UTC())
	require.NoError(t, err)
	require.Equal(t, 1, queue.ActiveByIsolation[workflowv3.IsolationSubprocessRestricted])
	require.NoError(t, store.Cancel(ctx, "cancel-isolated", time.Now().UTC()))
	cancelExecution()
	require.Error(t, <-done)
	snapshot, err := store.Snapshot(ctx, "cancel-isolated")
	require.NoError(t, err)
	require.Equal(t, "canceled", snapshot.Status)
	require.Len(t, snapshot.Attempts, 1)
	require.Equal(t, "canceled", snapshot.Attempts[0].Status)
	require.Empty(t, snapshot.Outputs)
}

func TestRollingRegistryRequiresEveryRetainedIsolationExecutorDigest(t *testing.T) {
	worker := buildIsolatedWorker(t)
	first := &BubblewrapExecutor{WorkerExecutable: worker, BubblewrapExecutable: "/usr/bin/bwrap", Tools: map[string]string{"fixture.echo": buildFixtureTool(t)}}
	second := &BubblewrapExecutor{WorkerExecutable: worker, BubblewrapExecutable: "/usr/bin/bwrap", Tools: map[string]string{"fixture.echo": buildForkingTool(t)}}
	firstDigest, err := first.Identity()
	require.NoError(t, err)
	secondDigest, err := second.Identity()
	require.NoError(t, err)
	firstRegistry, err := workflowv3isolation.RegistryWithExecutor(firstDigest)
	require.NoError(t, err)
	secondRegistry, err := workflowv3isolation.RegistryWithExecutor(secondDigest)
	require.NoError(t, err)
	manager, err := NewRegistryManager(firstRegistry)
	require.NoError(t, err)
	require.NoError(t, manager.Activate(secondRegistry, nil))
	set, err := NewIsolationExecutorSet(first, second)
	require.NoError(t, err)
	store, err := workflowv3sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	artifacts, err := workflowv3.NewFileArtifactStore(t.TempDir(), 1024)
	require.NoError(t, err)
	modules, err := NewTaskModuleRegistry(FSInputModule(), AllowlistedExecModule(map[string]string{"fixture.echo": buildFixtureTool(t)}))
	require.NoError(t, err)
	engine := &Engine{Store: store, Registry: manager, Artifacts: artifacts, Modules: modules, Isolation: set}
	require.NoError(t, engine.validate())
	onlySecond, err := NewIsolationExecutorSet(second)
	require.NoError(t, err)
	engine.Isolation = onlySecond
	require.ErrorContains(t, engine.validate(), "unavailable")
	require.NoError(t, manager.RemoveDrained(firstRegistry.Generation()))
	require.NoError(t, engine.validate())
}

func TestEngineRunsAuthoredRestrictedTaskWithDurableIsolationEvidence(t *testing.T) {
	ctx := context.Background()
	worker := buildIsolatedWorker(t)
	tool := buildFixtureTool(t)
	executor := &BubblewrapExecutor{WorkerExecutable: worker, BubblewrapExecutable: "/usr/bin/bwrap", Tools: map[string]string{"fixture.echo": tool}}
	executorDigest, err := executor.Identity()
	require.NoError(t, err)
	registry, err := workflowv3isolation.RegistryWithExecutor(executorDigest)
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	authored, err := workflowmodule.Author(ctx, workflowv3isolation.WorkflowSource(), catalog, workflowv3isolation.DescriptorModule())
	require.NoError(t, err)
	root := t.TempDir()
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(root, "artifacts"), 1<<20)
	require.NoError(t, err)
	const canary = "PRIVATE-ISOLATION-SOURCE-CANARY"
	input, err := artifacts.Put(ctx, "isolation-source/v1", "application/json", []byte(`{"values":["`+canary+`"]}`))
	require.NoError(t, err)
	store, err := workflowv3sqlite.Open(ctx, filepath.Join(root, "workflow.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	modules, err := NewTaskModuleRegistry(FSInputModule(), AllowlistedExecModule(map[string]string{"fixture.echo": tool}))
	require.NoError(t, err)
	engine := &Engine{Store: store, Registry: registry, Artifacts: artifacts, Modules: modules, Isolation: executor, LeaseDuration: 20 * time.Second}
	require.NoError(t, engine.Submit(ctx, "isolated-run", authored.Plan, map[string]workflowv3.ArtifactRef{"source": input}))
	queue, err := store.QueueSnapshot(ctx, registry, map[string]int{workflowv3isolation.ResourceClass: 1}, time.Now().UTC())
	require.NoError(t, err)
	require.Equal(t, 1, queue.ReadyByIsolation[workflowv3.IsolationSubprocessRestricted])
	ran, err := engine.RunOne(ctx)
	require.NoError(t, err)
	require.True(t, ran)
	snapshot, err := engine.Snapshot(ctx, "isolated-run")
	require.NoError(t, err)
	require.Equal(t, "succeeded", snapshot.Status)
	require.Len(t, snapshot.Attempts, 1)
	require.Equal(t, workflowv3.IsolationSubprocessRestricted, snapshot.Attempts[0].IsolationClass)
	require.NotEmpty(t, snapshot.Attempts[0].IsolationPolicyDigest)
	require.Equal(t, executorDigest, snapshot.Attempts[0].IsolationExecutorDigest)
	body, err := workflowv3.ReadArtifact(ctx, artifacts, snapshot.Outputs["output"])
	require.NoError(t, err)
	require.NotContains(t, string(body), canary)
	databaseBody, err := os.ReadFile(filepath.Join(root, "workflow.db"))
	require.NoError(t, err)
	require.NotContains(t, string(databaseBody), canary)
	require.NotContains(t, strings.Join(registry.ModuleAliases(), ","), "workflow/operator")
}

func TestRestrictedSubprocessMatchesTrustedExecutionDigest(t *testing.T) {
	ctx := context.Background()
	worker := buildIsolatedWorker(t)
	tool := buildFixtureTool(t)
	executor := &BubblewrapExecutor{WorkerExecutable: worker, BubblewrapExecutable: "/usr/bin/bwrap", Tools: map[string]string{"fixture.echo": tool}}
	require.NoError(t, executor.Validate())
	executorDigest, err := executor.Identity()
	require.NoError(t, err)
	registry, err := workflowv3isolation.RegistryWithExecutor(executorDigest)
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	spec, ok := catalog.Lookup(workflowv3.TaskKey{Kind: "fixture.isolation.transform", Version: "v1"})
	require.True(t, ok)
	registered, err := registry.Resolve(spec.Identity)
	require.NoError(t, err)
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(t.TempDir(), "artifacts"), 1<<20)
	require.NoError(t, err)
	input, err := artifacts.Put(ctx, "isolation-source/v1", "application/json", []byte(`{"values":["alpha","beta","gamma"]}`))
	require.NoError(t, err)
	modules, err := NewTaskModuleRegistry(FSInputModule(), AllowlistedExecModule(map[string]string{"fixture.echo": tool}))
	require.NoError(t, err)
	request := TaskRequest{RunID: "isolation", NodeKey: "transform", Attempt: 1, Task: registered, Inputs: map[string]workflowv3.ArtifactRef{"source": input}, Artifacts: artifacts, Modules: modules}
	trusted, err := RunTask(ctx, request)
	require.NoError(t, err)
	isolation, err := workflowv3.CompileIsolation(nil, spec.IsolationMaximum, spec.IsolationExecutorDigest)
	require.NoError(t, err)
	isolated, err := executor.Execute(ctx, request, isolation)
	require.NoError(t, err)
	require.Equal(t, trusted.Outputs["output"].Digest, isolated.Outputs["output"].Digest)
	body, err := workflowv3.ReadArtifact(ctx, artifacts, isolated.Outputs["output"])
	require.NoError(t, err)
	require.JSONEq(t, `{"checksum":16,"count":3}`, string(body))
}
