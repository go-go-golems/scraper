package workflowv3runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	fetchmod "github.com/go-go-golems/go-go-goja/modules/fetch"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3sqlite"
	"github.com/stretchr/testify/require"
)

const dispatcherTaskSource = `
const task = require("workflow/task");
const fs = require("fs:input");
const {fetch} = require("fetch:dispatch-test");
exports.snapshot = task.implementation(async ctx => {
  const input = ctx.input().request;
  const request = JSON.parse(await fs.readFile(input.path, "utf8"));
  const response = await fetch(request.url);
  if (!response.ok) {
    throw task.failure({class: "transport", code: "DISPATCH_HTTP",
      retryable: false, message: "dispatch fixture request failed"});
  }
  const body = await response.text();
  const output = await ctx.outputs.putJSON("output", {
    schema: "dispatch-output/v1", value: {body},
  });
  return task.success({output});
});
`

func TestDrainDispatchCompletionsEmptiesReadyBufferBeforeRefill(t *testing.T) {
	t.Parallel()
	completions := make(chan dispatchCompletion, 2)
	completions <- dispatchCompletion{lease: workflowv3.Lease{RunID: "run", NodeKey: "one"}}
	completions <- dispatchCompletion{lease: workflowv3.Lease{RunID: "run", NodeKey: "two"}, err: &AttemptExecutionError{Err: context.Canceled}}
	require.NoError(t, drainDispatchCompletions(completions))
	require.Empty(t, completions)
}

func TestDispatcherRefillsReleasedResourceWhileUnrelatedTaskRuns(t *testing.T) {
	var httpActive atomic.Int32
	var httpMax atomic.Int32
	httpOneStarted := make(chan struct{})
	httpTwoStarted := make(chan struct{})
	slowStarted := make(chan struct{})
	slowFinished := make(chan struct{})
	releaseHTTP := make(chan struct{})
	releaseSlow := make(chan struct{})
	var onceHTTPOne sync.Once
	var onceHTTPTwo sync.Once
	var onceSlow sync.Once

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/http-1":
			current := httpActive.Add(1)
			updateMaximum(&httpMax, current)
			onceHTTPOne.Do(func() { close(httpOneStarted) })
			<-releaseHTTP
			httpActive.Add(-1)
		case "/http-2":
			current := httpActive.Add(1)
			updateMaximum(&httpMax, current)
			onceHTTPTwo.Do(func() { close(httpTwoStarted) })
			httpActive.Add(-1)
		case "/slow":
			onceSlow.Do(func() { close(slowStarted) })
			<-releaseSlow
			close(slowFinished)
		default:
			http.NotFound(response, request)
			return
		}
		_, _ = response.Write([]byte(request.URL.Path))
	}))
	defer server.Close()

	bundle, registry, plan := dispatcherFixture(t)
	_ = bundle
	root := t.TempDir()
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(root, "artifacts"), 1<<20)
	require.NoError(t, err)
	inputs := map[string]workflowv3.ArtifactRef{
		"http1": putJSONArtifact(t, artifacts, "dispatch-request/v1", map[string]any{"url": server.URL + "/http-1"}),
		"http2": putJSONArtifact(t, artifacts, "dispatch-request/v1", map[string]any{"url": server.URL + "/http-2"}),
		"slow":  putJSONArtifact(t, artifacts, "dispatch-request/v1", map[string]any{"url": server.URL + "/slow"}),
	}
	store, err := workflowv3sqlite.Open(context.Background(), filepath.Join(root, "workflow.db"))
	require.NoError(t, err)
	modules, err := NewTaskModuleRegistry(
		FSInputModule(),
		FetchModule("fetch:dispatch-test", fetchmod.Policy{
			AllowedOrigins: []string{server.URL}, Timeout: time.Second,
			MaxResponseBytes: 1024,
			Credentials:      fetchmod.CredentialPolicy{AllowEnv: false, AllowFiles: false},
		}, nil),
	)
	require.NoError(t, err)
	engine := &Engine{
		Store: store, Registry: registry, Artifacts: artifacts,
		Modules: modules, LeaseDuration: 2 * time.Second,
	}
	dispatcher := &Dispatcher{
		Engine: engine,
		Capacities: map[string]int{
			"network.http.test": 1,
			"network.slow.test": 1,
		},
		PollInterval: 2 * time.Millisecond,
	}
	require.NoError(t, engine.Submit(context.Background(), "work-conserving", plan, inputs))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- dispatcher.Run(ctx) }()
	awaitClosed(t, httpOneStarted)
	awaitClosed(t, slowStarted)

	queue, err := dispatcher.QueueSnapshot(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, queue.ActiveByResource["network.http.test"])
	require.Equal(t, 1, queue.ActiveByResource["network.slow.test"])
	require.Equal(t, 1, queue.BlockedByReason["resource-capacity"])

	close(releaseHTTP)
	awaitClosed(t, httpTwoStarted)
	select {
	case <-slowFinished:
		t.Fatal("HTTP slot was not refilled until the unrelated slow task finished")
	default:
	}
	close(releaseSlow)

	require.Eventually(t, func() bool {
		snapshot, snapshotErr := engine.Snapshot(context.Background(), "work-conserving")
		return snapshotErr == nil && snapshot.Status == "succeeded"
	}, 3*time.Second, 5*time.Millisecond)
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	require.Equal(t, int32(1), httpMax.Load())
	require.NoError(t, store.Close())
}

func dispatcherFixture(t *testing.T) (*workflowv3.Bundle, *workflowv3.SealedRegistry, workflowv3.WorkflowPlan) {
	t.Helper()
	bundle, err := workflowv3.NewBundle(workflowv3.BundleManifest{
		Name: "dispatcher-fixture", Version: "1", ABI: workflowv3.TaskABI,
		Tasks: []workflowv3.BundleTask{
			{
				TaskKey:       workflowv3.TaskKey{Kind: "dispatch.http", Version: "v1"},
				Entrypoint:    "tasks.cjs#snapshot",
				Inputs:        map[string]string{"request": "dispatch-request/v1"},
				Outputs:       map[string]string{"output": "dispatch-output/v1"},
				Modules:       []string{"fetch:dispatch-test", "fs:input"},
				ResourceClass: "network.http.test",
				Retry:         workflowv3.RetryPolicy{MaxAttempts: 1},
			},
			{
				TaskKey:       workflowv3.TaskKey{Kind: "dispatch.slow", Version: "v1"},
				Entrypoint:    "tasks.cjs#snapshot",
				Inputs:        map[string]string{"request": "dispatch-request/v1"},
				Outputs:       map[string]string{"output": "dispatch-output/v1"},
				Modules:       []string{"fetch:dispatch-test", "fs:input"},
				ResourceClass: "network.slow.test",
				Retry:         workflowv3.RetryPolicy{MaxAttempts: 1},
			},
		},
	}, map[string][]byte{"tasks.cjs": []byte(dispatcherTaskSource)})
	require.NoError(t, err)
	builder := workflowv3.NewRegistryBuilder()
	require.NoError(t, builder.AdvertiseModules("fetch:dispatch-test", "fs:input"))
	require.NoError(t, builder.AddBundle(bundle))
	registry, err := builder.Seal()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	plan, err := workflowv3.Compile(workflowv3.WorkflowIR{
		Schema: workflowv3.IRSchema,
		Name:   "work-conserving",
		Inputs: []workflowv3.IRInput{
			{Name: "http1", Schema: "dispatch-request/v1"},
			{Name: "http2", Schema: "dispatch-request/v1"},
			{Name: "slow", Schema: "dispatch-request/v1"},
		},
		Nodes: []workflowv3.IRNode{
			dispatchNode("http-1", "dispatch.http", "http1"),
			dispatchNode("http-2", "dispatch.http", "http2"),
			dispatchNode("slow", "dispatch.slow", "slow"),
		},
		Outputs: []workflowv3.IROutput{{
			Name: "result",
			Value: workflowv3.ValueRef{
				Source: "node-output", NodeKey: "http-2", Port: "output",
				Schema: "dispatch-output/v1",
			},
		}},
	}, catalog)
	require.NoError(t, err)
	return bundle, registry, plan
}

func dispatchNode(key, kind, input string) workflowv3.IRNode {
	return workflowv3.IRNode{
		Key:  workflowv3.NodeKey(key),
		Task: workflowv3.TaskKey{Kind: kind, Version: "v1"},
		Bindings: map[string]workflowv3.ValueRef{
			"request": {Source: "input", Name: input, Schema: "dispatch-request/v1"},
		},
	}
}

func awaitClosed(t *testing.T, channel <-chan struct{}) {
	t.Helper()
	select {
	case <-channel:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for dispatch event")
	}
}

func updateMaximum(maximum *atomic.Int32, value int32) {
	for {
		current := maximum.Load()
		if value <= current || maximum.CompareAndSwap(current, value) {
			return
		}
	}
}
