---
Title: Workflow API User Guide and Reference
Slug: scraper-workflow-api-reference
Short: "Explains the embedded Go workflow API, its runtime model, public types, extension points, and failure modes."
Topics:
- scraper
- workflow-api
- go
- embedded-runtime
- reference
Commands:
- help
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The workflow API is the Go-facing surface for embedding scraper's durable execution engine in another program. It wraps the lower-level store, scheduler, runner registry, and operator services behind workflow-native concepts: runtimes, packages, entrypoints, executors, steps, artifacts, projections, and operator actions. Use it when your application wants scraper's durable graph execution but wants to define workflows directly in Go rather than through JavaScript site manifests.

The central idea is simple and strict: an entrypoint creates the initial durable graph, executors complete individual durable steps, and the runtime persists the state transition between those points. The API does not ask application code to manage leases, dependency refresh, queue limits, result rows, or workflow status transitions directly. Those concerns stay in the engine.

## When To Use The Workflow API

Use the workflow API when a Go application needs durable work orchestration inside its own process. The API is useful for services that already have their own lifecycle manager, tests that need deterministic scheduler cycles, and packages that want typed Go executors with access to scraper's engine store.

Use the CLI/site-manifest path instead when operators should define site behavior from files under `sites/`, use JavaScript submit verbs, or run the standard `scraper worker run` process. Both paths share the same engine concepts, but they optimize for different authors.

| Use case | Better fit | Reason |
|----------|------------|--------|
| A Go service wants to start workflows from HTTP handlers | Workflow API | The service can call `StartRun` directly and supervise `StartWorkers` with its own context |
| A scraper site should be editable as manifests and scripts | CLI/site manifest runtime | Site authors can work in `site.yaml`, `verbs/`, and `scripts/` without recompiling Go |
| A unit test needs to seed a run and execute one scheduler cycle | Workflow API | `RunOnce` gives deterministic control over execution |
| An operator wants to run an existing site workflow | CLI | Dynamic site commands are already exposed through Cobra/Glazed |

## Runtime Lifecycle

`workflow.Runtime` is the main object. It owns the store connection, scheduler, executor registry, registered packages, and optional stores for artifacts and projections.

```go
rt, err := workflow.NewRuntime(ctx, workflow.Config{
    Store:           workflow.SQLiteStore("./var/engine.db"),
    ArtifactStore:   workflow.NewFileArtifactStore("./var/artifacts"),
    ProjectionStore: workflow.NewSQLiteProjectionStore("./var/projections"),
    WorkerID:        "api-worker-1",
    MaxWorkers:      4,
    PollInterval:    250 * time.Millisecond,
    LeaseDuration:   30 * time.Second,
    Queues: map[model.QueueKey]workflow.QueueConfig{
        "fetch": {MaxWorkers: 8},
        "parse": {MaxWorkers: 2},
    },
})
if err != nil {
    return err
}
defer rt.Close()
```

`NewRuntime` normalizes unset options. The defaults are conservative: worker ID `workflow-runtime`, one worker, a 250ms poll interval, a 30s lease duration, and an empty queue-policy map. `Store` has no default because durability needs an explicit backend.

The runtime should be closed when the embedding application is done with it. `Close` closes projection stores that implement `Close() error` and then closes the underlying engine store.

## Configuration Reference

`workflow.Config` defines the runtime's durable backend and scheduling behavior.

| Field | Required | Purpose |
|-------|----------|---------|
| `Store` | Yes | Opens the durable engine store and provides operator services when available |
| `ArtifactStore` | No | Stores large artifact bytes outside the engine result row |
| `ProjectionStore` | No | Resolves query-oriented projection databases for executors |
| `WorkerID` | No | Identifies this runtime in leases and scheduler activity |
| `MaxWorkers` | No | Sets scheduler worker concurrency |
| `PollInterval` | No | Sets the default worker-loop sleep interval used by scheduler config |
| `LeaseDuration` | No | Sets the default lease duration for running steps |
| `Queues` | No | Maps queue names to per-queue policy overrides |

`workflow.SQLiteStore(path)` is the built-in store configuration. It creates the parent directory when needed, opens the existing SQLite engine store, and exposes an operator service backed by `engineview.NewService(path)`.

`workflow.QueueConfig` controls a named queue:

| Field | Purpose |
|-------|---------|
| `MaxWorkers` | Maximum in-flight work for the queue after normalization |
| `RateLimit` | Optional token-bucket policy from `pkg/engine/model` |

If a queue is not listed in `Config.Queues`, the scheduler uses `model.DefaultQueuePolicy()`.

## Packages and Entrypoints

A package is the workflow domain that callers start. It has a stable name, an optional display name, and an entrypoint. The package name becomes the workflow site name in the underlying engine model, which keeps workflow records grouped by domain.

```go
pkg := workflow.NewPackage("book-ocr").
    DisplayName("Book OCR").
    Entrypoint(workflow.EntrypointFunc[StartInput](startBookOCR)).
    Build()

if err := rt.RegisterPackage(pkg); err != nil {
    return err
}
```

The entrypoint creates initial durable steps with `RunBuilder`. It may also set run metadata or replace the run's display name.

```go
func startBookOCR(ctx context.Context, run *workflow.RunBuilder, input StartInput) error {
    run.Name("OCR " + input.BookID)
    run.Metadata("bookID", input.BookID)

    convert, err := run.Step("convert", input, workflow.StepOpts{
        Kind:  "book/convert-pdf",
        Queue: "cpu",
    })
    if err != nil {
        return err
    }

    _, err = run.Step("index", input, workflow.StepOpts{
        Kind:      "book/index-pages",
        Queue:     "sqlite",
        DependsOn: workflow.Require(convert),
    })
    return err
}
```

Entrypoints do not perform durable work themselves. They describe the first graph that should be persisted. That design keeps `StartRun` fast and makes the actual work recoverable by the scheduler.

## RunBuilder Reference

`RunBuilder` is available only during package entrypoint execution. It constructs the initial workflow graph.

| Method | Purpose |
|--------|---------|
| `Name(name string)` | Sets the persisted workflow display name |
| `Metadata(key, value string)` | Adds or updates workflow metadata |
| `Step(id string, input any, opts StepOpts) (StepHandle, error)` | Appends an initial durable step and returns a handle for dependencies |

`Step` generates a stable ID when `id` is empty, but explicit IDs make dependency graphs and tests easier to read. `StepOpts.Kind` is required because it selects the executor. If `StepOpts.Site` is empty, the step uses the package name as its site.

`workflow.Require(handles...)` converts step handles into required dependencies. Use it when a step cannot run until earlier steps have completed successfully.

## Executors

Executors implement step behavior. Register each executor before workers can execute steps of that kind.

```go
err := rt.RegisterExecutor(workflow.NewTypedExecutor(
    "book/index-pages",
    func(ctx context.Context, step *workflow.StepContext, input IndexInput) error {
        var converted ConvertResult
        if err := step.DependencyData("convert", &converted); err != nil {
            return err
        }

        return step.Result(IndexResult{Pages: len(converted.Pages)})
    },
))
```

`workflow.NewTypedExecutor[I]` decodes step input into `I` before calling your function. `workflow.NewExecutor` gives you raw access through `StepContext` when an executor needs custom decoding.

The executor kind is part of durable data. Changing kind names after runs have been created can strand existing ready steps because the scheduler will look for the old kind in the runner registry.

## StepContext Reference

`StepContext` is the executor-facing view of the current durable step. It exposes input, dependency results, result writers, artifact writers, projection access, and child-step emission.

| Method | Purpose |
|--------|---------|
| `Workflow() model.WorkflowRun` | Returns the current workflow record |
| `Step() model.OpSpec` | Returns the current step/op spec |
| `Lease() model.Lease` | Returns lease metadata for this execution |
| `Now() time.Time` | Returns the scheduler-provided execution timestamp |
| `Input(out any) error` | Decodes step input JSON into `out` |
| `RawInput() json.RawMessage` | Returns a copy of the raw input JSON |
| `DependencyResult(opID model.OpID)` | Loads a dependency result by op ID |
| `DependencyData(opID model.OpID, out any)` | Decodes dependency result data into `out` |
| `Result(data any) error` | Sets structured JSON result data for this step |
| `Record(collection, key string, data any) error` | Adds a record write to the result |
| `Artifact(name, contentType string, body []byte, opts ...ArtifactOption)` | Adds an inline artifact to the result row |
| `StoreArtifact(name, contentType string, body []byte, opts ...ArtifactOption)` | Writes bytes to the configured external artifact store and records a reference artifact |
| `Projection(name string)` | Opens a named projection from the configured projection store |
| `Emit(id string, input any, opts StepOpts)` | Appends a child step to be persisted when the current step succeeds |

The scheduler persists `Result`, `Record`, `Artifact`, and `Emit` output when the executor returns nil. If the executor returns an error, the step follows the failure path instead.

## Emitting Child Steps

`StepContext.Emit` lets an executor expand the graph after it has inspected inputs, fetched a page, read a file, or queried a dependency result. Emitted steps become durable only when the current step completes successfully.

```go
_, err := step.Emit("page-2", FetchInput{URL: nextURL}, workflow.StepOpts{
    Kind:     "book/fetch-page",
    Queue:    "fetch",
    Metadata: map[string]string{"source": "pagination"},
})
if err != nil {
    return err
}

return step.Result(map[string]any{"next": nextURL})
```

By default, emitted steps use the current step as their parent and the current step's site. Set `StepOpts.ParentID` or `StepOpts.Site` only when the workflow deliberately needs a different structure.

## Artifacts

Artifacts represent files or blobs produced by a step. Small artifacts can be written inline with `Artifact`; larger artifacts should use an external `ArtifactStore` so the engine result row stays small.

```go
ref, err := step.StoreArtifact(
    "page-001.md",
    "text/markdown",
    []byte(markdown),
    workflow.ArtifactKind("markdown"),
    workflow.ArtifactMetadata(map[string]string{"page": "1"}),
)
if err != nil {
    return err
}

return step.Result(map[string]any{"artifactID": ref.ID})
```

`workflow.NewFileArtifactStore(root)` stores artifact bytes under a local filesystem root and writes a JSON metadata sidecar. `StoreArtifact` records a compact `external-artifact-ref` artifact in the engine result so existing result/artifact APIs can still point operators to the external object.

Artifact options:

| Option | Purpose |
|--------|---------|
| `ArtifactID(id string)` | Sets a stable artifact ID instead of the generated one |
| `ArtifactKind(kind string)` | Sets the artifact kind shown to operators |
| `ArtifactMetadata(map[string]string)` | Attaches metadata to the artifact write or external reference |

## Projections

A projection is a query-oriented read model owned by a workflow package or domain. It is separate from the engine store. The engine store tracks scheduling state; projections hold application data that operators or downstream systems query.

```go
projection, err := step.Projection("book-ocr")
if err != nil {
    return err
}

if _, err := projection.Exec(ctx, `
    CREATE TABLE IF NOT EXISTS pages(
        page INTEGER PRIMARY KEY,
        status TEXT,
        text TEXT
    )
`); err != nil {
    return err
}

_, err = projection.Exec(ctx,
    `INSERT OR REPLACE INTO pages(page, status, text) VALUES(?, ?, ?)`,
    input.Page,
    "done",
    extractedText,
)
return err
```

`workflow.NewSQLiteProjectionStore(root)` stores one SQLite database per projection name. The runtime closes opened projection databases when `Runtime.Close` runs.

## Errors, Retry, and Cancellation

Executors signal failure by returning errors. Use `workflow.Retryable(code, err)` when retrying makes sense, and `workflow.Permanent(code, err)` when the failure should be treated as non-retryable unless an operator deliberately intervenes.

```go
if temporaryHTTPFailure(err) {
    return workflow.Retryable("fetch_failed", err)
}
if invalidInput(err) {
    return workflow.Permanent("invalid_input", err)
}
return err
```

The stable code is stored in `model.OpError` and is useful for dashboards, metrics, and filtering. A plain error still fails the step, but it carries less operator-facing structure.

The runtime exposes two operator actions when the store configuration provides an operator service:

| Method | Purpose |
|--------|---------|
| `RetryStep(ctx, runID, stepID)` | Moves a failed step back to ready so workers can execute it again |
| `CancelRun(ctx, runID)` | Cancels pending, ready, and running steps for a workflow |

The built-in SQLite store provides these actions through the engineview service. Current cancellation marks running steps canceled and removes leases; executors should still use contexts for cooperative cancellation where possible.

## Running Workers

`RunOnce` and `StartWorkers` are the two execution modes.

```go
cycle, err := rt.RunOnce(ctx)
```

`RunOnce` executes a scheduler cycle and returns a `scheduler.CycleResult`. It is the best choice for tests and command handlers that want bounded work.

```go
err := rt.StartWorkers(ctx,
    workflow.WithWorkerPollInterval(time.Second),
    workflow.WithWorkerMaxCycles(100),
)
```

`StartWorkers` loops until the context is canceled, an error occurs, or `WithWorkerMaxCycles` is reached. It is intentionally context-driven so the embedding service can own shutdown behavior.

## Reading Runtime State

The runtime exposes convenience reads for the most common embedded use cases:

| Method | Purpose |
|--------|---------|
| `Workflow(ctx, runID)` | Reads the durable workflow record |
| `Result(ctx, runID, stepID)` | Reads one step result |
| `Projection(ctx, name)` | Opens a named projection outside executor code |

These methods do not replace the lower-level engine store APIs. They provide the narrow reads most embedding applications need after starting and executing runs.

## Design Rules That Keep Workflows Durable

The API is small because most correctness comes from preserving a few boundaries.

- Entrypoints should describe initial work, not perform long-running work. If an entrypoint fetches a remote page or processes a large file, that work cannot be leased, retried, or recovered by the scheduler.
- Executors should make durable effects through `StepContext`. Results, records, artifacts, emitted steps, and structured errors are persisted by the scheduler completion path.
- Step kind names should be stable. They are stored in durable op specs and used later to find executors.
- Large blobs should use an `ArtifactStore`. Keeping large bodies out of result rows keeps the engine DB responsive.
- Query-facing application state should go into projections, not engine scheduling tables.
- Context cancellation should be honored by long-running executors. The runtime controls scheduler lifecycle through contexts, but executor code must cooperate with cancellation while doing external work.

## Public API Summary

This summary lists the primary exported constructors and methods in `pkg/workflow`.

| API | Role |
|-----|------|
| `NewRuntime(ctx, Config)` | Creates an embedded workflow runtime |
| `SQLiteStore(path)` | Configures the runtime to use the SQLite engine store |
| `NewPackage(name)` | Starts a package builder |
| `EntrypointFunc[I]` | Adapts a typed Go function into an entrypoint |
| `RunBuilder.Step` | Adds an initial durable step |
| `Require(handles...)` | Builds required dependencies between initial steps |
| `NewExecutor(kind, fn)` | Registers an untyped executor function |
| `NewTypedExecutor[I](kind, fn)` | Registers a typed executor function |
| `StepContext.Result` | Writes structured step result data |
| `StepContext.Emit` | Emits child steps dynamically |
| `StepContext.DependencyData` | Reads typed dependency result data |
| `NewFileArtifactStore(root)` | Stores artifacts on the local filesystem |
| `NewSQLiteProjectionStore(root)` | Stores per-projection SQLite databases |
| `Retryable(code, err)` | Returns a structured retryable step error |
| `Permanent(code, err)` | Returns a structured non-retryable step error |
| `Runtime.StartRun` | Persists a new workflow run and its initial steps |
| `Runtime.RunOnce` | Runs one scheduler cycle |
| `Runtime.StartWorkers` | Runs scheduler cycles until stopped |
| `Runtime.RetryStep` | Retries a failed step through operator services |
| `Runtime.CancelRun` | Cancels a run through operator services |

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| `workflow runtime store is required` | `Config.Store` is nil | Pass `workflow.SQLiteStore(path)` or implement `StoreConfig` |
| `sqlite workflow store path is required` | `SQLiteStore("")` was used | Pass a non-empty DB path |
| `workflow package name is required` | A package was built with an empty name | Use `workflow.NewPackage("stable-name")` |
| `workflow package "..." entrypoint is required` | `Entrypoint(...)` was not configured | Add an entrypoint before `Build` or before registration |
| `workflow package "..." is not registered` | `StartRun` references an unknown package | Call `RegisterPackage` and verify the package name |
| `workflow step kind is required` or `emitted step kind is required` | `StepOpts.Kind` is empty | Set a stable kind on every initial and emitted step |
| A step remains failed after fixing code | The persisted op failed earlier and has not been retried | Call `RetryStep` or create a new run, depending on the operator model |
| `artifact store is not configured` | An executor called `StoreArtifact` without `Config.ArtifactStore` | Configure `workflow.NewFileArtifactStore` or another `ArtifactStore` |
| `projection store is not configured` | An executor called `Projection` without `Config.ProjectionStore` | Configure `workflow.NewSQLiteProjectionStore` or another `ProjectionStore` |
| Dependency data cannot be decoded | The dependency result shape does not match the target struct | Inspect `Result(ctx, runID, opID).Data` and adjust the result type or producing executor |

## See Also

- `scraper help scraper-workflow-api-getting-started` — Step-by-step introduction to the embedded workflow API
- `scraper help scraper-runtime-model` — Runtime concepts shared by the CLI and embedded workflow API
- `scraper help scraper-queue-policies-and-rate-limiting` — Queue policy and rate limit behavior
- `scraper help scraper-http-api` — HTTP API for observing and mutating engine state from outside a Go embedding
- `pkg/workflow/runtime.go` — Runtime construction, package registration, run start, workers, and reads
- `pkg/workflow/package.go` — Packages, entrypoints, run builder, steps, and dependencies
- `pkg/workflow/context.go` — Executor step context, results, artifacts, projections, and emitted child steps
- `pkg/workflow/runtime_test.go` — End-to-end examples of the public API
