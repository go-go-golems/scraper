---
Title: Getting Started with the Workflow API
Slug: scraper-workflow-api-getting-started
Short: "Build a small embedded workflow with the Go workflow API, run it locally, and inspect its result."
Topics:
- scraper
- workflow-api
- go
- embedded-runtime
Commands:
- help
Flags: []
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

The workflow API lets a Go program embed the scraper engine without going through site manifests or JavaScript submit verbs. You create a `Runtime`, register step executors, register a package entrypoint, start a run, and then let workers execute the durable steps. The API is intentionally small: packages describe the initial graph, executors do the work, and the runtime persists state in the same engine store used by the CLI worker.

By the end of this tutorial you should be able to read and write the smallest useful embedded workflow. The example runs in one process, uses SQLite for durability, and stores a structured result that can be read back after the scheduler finishes.

## What You Will Build

This tutorial builds a workflow package named `hello`. Starting a run creates one durable step named `root`. The registered executor receives typed input, writes a result, and finishes successfully. The runtime then marks the workflow succeeded.

The complete path is:

```text
NewRuntime
  -> RegisterExecutor("hello/echo")
  -> RegisterPackage("hello")
  -> StartRun("hello", input)
  -> RunOnce()
  -> Result(run.ID, "root")
```

That sequence is the foundation for larger workflows. More complex workflows add dependency edges, emitted child steps, artifact stores, projection databases, and long-running worker loops, but they still follow the same division of responsibilities.

## Prerequisites

You need this repository as a Go module and a program or test file that can import `github.com/go-go-golems/scraper/pkg/workflow`. The examples below omit package declarations and imports so the workflow code stays focused, but a real file needs at least:

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/go-go-golems/scraper/pkg/engine/model"
    "github.com/go-go-golems/scraper/pkg/workflow"
)
```

Use `context.Context` throughout. The runtime, entrypoints, executors, artifact stores, and projection stores all accept contexts because embedded applications should own cancellation and lifecycle.

## Step 1 — Create a Runtime

The runtime owns the durable store, scheduler, executor registry, package registry, and optional artifact/projection stores. A SQLite store is the normal local choice because it gives you the same durable engine behavior as the CLI.

```go
ctx := context.Background()

rt, err := workflow.NewRuntime(ctx, workflow.Config{
    Store:         workflow.SQLiteStore("./var/hello-engine.db"),
    WorkerID:      "hello-worker",
    MaxWorkers:    2,
    PollInterval:  250 * time.Millisecond,
    LeaseDuration: 30 * time.Second,
    Queues: map[model.QueueKey]workflow.QueueConfig{
        "default": {MaxWorkers: 2},
    },
})
if err != nil {
    return err
}
defer rt.Close()
```

`Store` is required. The other fields have defaults, but setting them explicitly is useful when learning the API. `WorkerID` appears in leases, `MaxWorkers` controls scheduler concurrency, `PollInterval` controls worker-loop sleep time, and `LeaseDuration` determines how long a leased step is considered owned before recovery logic can revisit it.

## Step 2 — Define Input and Result Types

Typed executors decode JSON step input into Go structs. This keeps the durable store JSON-compatible while letting executor code use ordinary Go values.

```go
type HelloInput struct {
    Message string `json:"message"`
}

type HelloResult struct {
    Echoed string `json:"echoed"`
}
```

The workflow package entrypoint receives the same typed input when you use `workflow.EntrypointFunc[I]`. The runtime serializes the value passed to `StartRun`, passes the raw JSON through the entrypoint adapter, and persists the original run input on the workflow record.

## Step 3 — Register an Executor

An executor handles one step kind. The kind string is the durable link between a step spec and the Go function that executes it. If a step says `Kind: "hello/echo"`, the runtime must have an executor registered with exactly that kind.

```go
err = rt.RegisterExecutor(workflow.NewTypedExecutor(
    "hello/echo",
    func(ctx context.Context, step *workflow.StepContext, input HelloInput) error {
        if input.Message == "" {
            return workflow.Permanent("empty_message", fmt.Errorf("message is required"))
        }

        return step.Result(HelloResult{Echoed: input.Message})
    },
))
if err != nil {
    return err
}
```

The executor writes its result through `step.Result`. Returning a normal error marks the step failed. Returning `workflow.Permanent` or `workflow.Retryable` gives the persisted error a stable code and retryability flag that operator tools can use later.

## Step 4 — Register a Package Entrypoint

A package is the unit that callers start. Its entrypoint creates the initial durable step graph for a run. In the smallest case, it creates one step.

```go
pkg := workflow.NewPackage("hello").
    DisplayName("Hello Workflow").
    Entrypoint(workflow.EntrypointFunc[HelloInput](
        func(ctx context.Context, run *workflow.RunBuilder, input HelloInput) error {
            run.Metadata("source", "getting-started")

            _, err := run.Step("root", input, workflow.StepOpts{
                Kind:  "hello/echo",
                Queue: "default",
            })
            return err
        },
    )).
    Build()

if err := rt.RegisterPackage(pkg); err != nil {
    return err
}
```

The entrypoint does not execute the step. It describes work that should become durable. This distinction is important: `StartRun` creates a workflow and initial ops in the store, while `RunOnce` or `StartWorkers` leases and executes ready ops later.

## Step 5 — Start a Run

Starting a run serializes the input, calls the package entrypoint, and persists the workflow plus initial steps. The returned handle gives you the workflow ID that all later reads use.

```go
run, err := rt.StartRun(
    ctx,
    "hello",
    HelloInput{Message: "hello workflow"},
    workflow.WithRunID("hello-run-1"),
    workflow.WithRunMetadata(map[string]string{"requestedBy": "example"}),
)
if err != nil {
    return err
}

fmt.Println(run.ID)
```

If you omit `WithRunID`, the runtime creates an ID from the package name and a UUID. Explicit IDs are useful in tests and in applications that need stable correlation with an external request.

## Step 6 — Execute Work

For a local program or test, `RunOnce` is the most direct way to execute ready work. It runs one scheduler cycle and returns a cycle result.

```go
cycle, err := rt.RunOnce(ctx)
if err != nil {
    return err
}

fmt.Printf("processed %d step(s)\n", cycle.Processed)
```

Long-running applications usually call `StartWorkers` in a goroutine controlled by a cancelable context:

```go
workerCtx, cancel := context.WithCancel(ctx)
defer cancel()

go func() {
    if err := rt.StartWorkers(workerCtx, workflow.WithWorkerPollInterval(time.Second)); err != nil {
        // In production, send this to your application's logger or supervisor.
        fmt.Printf("workflow workers stopped: %v\n", err)
    }
}()
```

Use `RunOnce` when a test wants deterministic control. Use `StartWorkers` when an embedded service should keep processing until the service shuts down.

## Step 7 — Read the Workflow and Result

The runtime exposes convenience read methods for the workflow record and individual step results.

```go
wf, err := rt.Workflow(ctx, run.ID)
if err != nil {
    return err
}
fmt.Println(wf.Status)

result, err := rt.Result(ctx, run.ID, "root")
if err != nil {
    return err
}
fmt.Println(string(result.Data))
// Output: {"echoed":"hello workflow"}
```

The result data is JSON because results are persisted in the engine store. Decode it into a Go struct in application code when you need typed access.

## Complete Example

This example keeps the whole workflow in one function. It is suitable for a small smoke test or for experimenting with the API before splitting code into packages.

```go
func runHelloWorkflow(ctx context.Context) error {
    type HelloInput struct {
        Message string `json:"message"`
    }
    type HelloResult struct {
        Echoed string `json:"echoed"`
    }

    rt, err := workflow.NewRuntime(ctx, workflow.Config{
        Store: workflow.SQLiteStore("./var/hello-engine.db"),
        Queues: map[model.QueueKey]workflow.QueueConfig{
            "default": {MaxWorkers: 1},
        },
    })
    if err != nil {
        return err
    }
    defer rt.Close()

    if err := rt.RegisterExecutor(workflow.NewTypedExecutor(
        "hello/echo",
        func(ctx context.Context, step *workflow.StepContext, input HelloInput) error {
            if input.Message == "" {
                return workflow.Permanent("empty_message", fmt.Errorf("message is required"))
            }
            return step.Result(HelloResult{Echoed: input.Message})
        },
    )); err != nil {
        return err
    }

    pkg := workflow.NewPackage("hello").
        DisplayName("Hello Workflow").
        Entrypoint(workflow.EntrypointFunc[HelloInput](
            func(ctx context.Context, run *workflow.RunBuilder, input HelloInput) error {
                _, err := run.Step("root", input, workflow.StepOpts{
                    Kind:  "hello/echo",
                    Queue: "default",
                })
                return err
            },
        )).
        Build()

    if err := rt.RegisterPackage(pkg); err != nil {
        return err
    }

    handle, err := rt.StartRun(ctx, "hello", HelloInput{Message: "hello workflow"})
    if err != nil {
        return err
    }

    if _, err := rt.RunOnce(ctx); err != nil {
        return err
    }

    result, err := rt.Result(ctx, handle.ID, "root")
    if err != nil {
        return err
    }
    fmt.Println(string(result.Data))
    return nil
}
```

## What To Change Next

Once the minimal workflow works, extend it in one direction at a time:

- Add a child step with `step.Emit` when an executor discovers more work.
- Add dependencies with `workflow.Require` when a later step must wait for earlier results.
- Add `workflow.NewFileArtifactStore` when steps produce large files that should not live inside result rows.
- Add `workflow.NewSQLiteProjectionStore` when steps should update a query-oriented read model.
- Replace `RunOnce` with `StartWorkers` when the runtime lives inside a service.

Each extension keeps the same durable boundary: entrypoints and executors describe work and results; the runtime stores the graph and scheduler state.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| `workflow runtime store is required` | `workflow.Config.Store` was not set | Pass `workflow.SQLiteStore(path)` or another `StoreConfig` implementation |
| `workflow package "..." is not registered` | `StartRun` was called before `RegisterPackage`, or the package name differs | Register the package and use the exact package name in `StartRun` |
| `runner not found` or a step never executes successfully | The step kind does not match any registered executor kind | Check the `Kind` in `RunBuilder.Step` or `StepContext.Emit` and the kind passed to `NewTypedExecutor` |
| `workflow step kind is required` | `StepOpts.Kind` was left empty | Set `Kind` on every initial and emitted step |
| A result is empty | The executor returned without calling `step.Result`, or it failed before doing so | Inspect the workflow/op status and return a structured result before successful completion |

## See Also

- `scraper help scraper-workflow-api-reference` — Full guide and reference for the embedded Go workflow API
- `scraper help scraper-runtime-model` — How the workflow engine, scheduler, workers, and JavaScript runtime fit together
- `scraper help scraper-queue-policies-and-rate-limiting` — Queue policy behavior used by both CLI and embedded runtimes
- `pkg/workflow/runtime_test.go` — Runnable tests that exercise packages, executors, artifacts, projections, retry, and cancel behavior
