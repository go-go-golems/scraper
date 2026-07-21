<!-- Source: https://parc.yolo.scapegoat.dev/note/projects/2026/05/25/article-scraper-workflow-api-building-a-public-reusable-durable-workflow-runtime -->
<!-- Retrieved: 2026-07-21 -->

[Terminology and agent guide](https://parc.yolo.scapegoat.dev/AGENTS.md)

modifiedJul 20, 2026

tags

aliasesScraper Workflow API Report, Embeddable Workflow Runtime, Public Reusable Workflow API

createdMay 25, 2026

prhttps://github.com/go-go-golems/scraper/pull/3

repo/home/manuel/workspaces/2026-05-20/book-ocr/scraper

statusactive

typearticle

## Scraper Workflow API: Building a Public Reusable Durable Workflow Runtime

This is the workflow-runtime foundation in the [Scraper — Durable Workflows, Research Extraction, and Evidence Pipelines](https://parc.yolo.scapegoat.dev/note/research/kb/projects/scraper) project map.

This report explains the public reusable workflow API added to the scraper repository in PR #3, "Feat: Introduce an embeddable workflow engine." The API lives under `pkg/workflow` and turns scraper's lower-level engine, scheduler, runner registry, store, artifacts, projections, and operator controls into a small Go-facing package for embedding durable workflows in another program.

≡ Summary

- The new API exposes a durable workflow runtime as Go concepts: `Runtime`, `Package`, `Entrypoint`, `Executor`, `RunBuilder`, `StepContext`, `ArtifactStore`, and `ProjectionStore`.
- It was built to let Go applications define and run multi-step durable workflows without going through JavaScript site manifests or direct scheduler/store plumbing.
- The design keeps the hard operational parts in the existing engine: leases, queue policies, dependency refresh, result persistence, retries, cancellation, and workflow status transitions.
- The API is now documented both as Glazed CLI help (`scraper help scraper-workflow-api-getting-started`, `scraper help scraper-workflow-api-reference`) and as this longer technical narrative.

The implementation is not a separate workflow engine. It is a public façade over the engine that already powers scraper. That choice matters. A new façade can give application authors clean typed APIs without duplicating scheduling logic or creating a second persistence model. The lower-level engine remains responsible for correctness. The public package is responsible for making the engine usable from ordinary Go code.

## Why this was built

The scraper project started with durable execution concepts aimed at scraping sites: workflows, ops, queues, runners, results, records, artifacts, and JavaScript execution. That model works well for manifest-driven sites, but it is too low-level for a Go package author who wants to embed scraper's engine in a service or tool.

Before `pkg/workflow`, using the engine directly meant understanding several internal layers at once:

- the store contract in `pkg/engine/store`,
- the scheduler in `pkg/engine/scheduler`,
- the runner interface in `pkg/engine/runner`,
- the engine model types in `pkg/engine/model`,
- result rows, dependency rows, leases, queue policies, artifacts, and workflow status transitions.

Those are legitimate internal concepts, but they are not the right first API for a user who wants to say: define this workflow, register these step functions, start a run, and let workers process it. The new package creates that surface.

The concrete trigger was the need to pull OCR-style workflows out of scraper-specific code and make the workflow machinery reusable. The branch history shows this evolution clearly: first the workflow executor façade was added, then the runtime skeleton, then operator controls, artifact storage, projection storage, runtime tests, and finally public help documentation. The result is a reusable Go API that can support book OCR, scraper sites, and future durable task systems without forcing each of them to talk directly to the scheduler.

## The public shape of the API

The API is centered on a small set of exported types. Each type has a specific responsibility, and those responsibilities line up with the lifecycle of a workflow run.

| Public concept | Main file | Responsibility |
| --- | --- | --- |
| `Runtime` | `pkg/workflow/runtime.go` | Owns the store, scheduler, executor registry, package registry, queue policies, artifacts, projections, and operator services. |
| `Config` | `pkg/workflow/runtime.go` | Configures the runtime store, worker identity, queue behavior, artifact store, and projection store. |
| `Package` | `pkg/workflow/package.go` | Defines a named workflow domain that callers can start. |
| `Entrypoint` / `EntrypointFunc[I]` | `pkg/workflow/package.go` | Creates the initial durable step graph for a run. |
| `RunBuilder` | `pkg/workflow/package.go` | Lets entrypoints set workflow metadata and create initial steps. |
| `Executor` / `NewTypedExecutor[I]` | `pkg/workflow/executor.go` | Implements one durable step kind. |
| `StepContext` | `pkg/workflow/context.go` | Gives executors access to input, dependency results, result writers, artifacts, projections, and child-step emission. |
| `ArtifactStore` | `pkg/workflow/artifact_store.go` | Stores large bytes outside the engine result row. |
| `ProjectionStore` | `pkg/workflow/projection_store.go` | Provides workflow-owned query databases. |
| `RetryStep` / `CancelRun` | `pkg/workflow/operators.go` | Exposes operator mutations through the existing engineview service. |
| `Retryable` / `Permanent` | `pkg/workflow/errors.go` | Gives failed steps stable operator-facing error metadata. |

The API is small because the runtime does not try to expose every engine detail. It exposes the points where application code legitimately participates: describing work, executing work, writing results, storing artifacts, querying projections, and controlling failed or unwanted work.

## The core execution path

A workflow run starts as typed Go input and becomes durable engine state. Later, workers lease ready steps, call registered executors, and persist the resulting state changes. This is the main path through the system.

```
flowchart TD
    A[Application code] --> B[workflow.NewRuntime]
    B --> C[RegisterExecutor]
    B --> D[RegisterPackage]
    D --> E[Runtime.StartRun]
    E --> F[EntrypointFunc decodes typed input]
    F --> G[RunBuilder creates initial OpSpec graph]
    G --> H[Scheduler.CreateWorkflow persists workflow and ops]
    H --> I[RunOnce or StartWorkers]
    I --> J[Scheduler leases ready op]
    J --> K[Executor receives StepContext]
    K --> L[Executor writes result, records, artifacts, emitted ops]
    L --> M[Scheduler persists completion]
    M --> N[Workflow and result reads]

    style H fill:#d8f3dc,stroke:#2d6a4f
    style M fill:#d8f3dc,stroke:#2d6a4f
    style K fill:#dbeafe,stroke:#1d4ed8
```

There are two important boundaries in that sequence.

First, `StartRun` is not a worker. It serializes input, invokes the package entrypoint, and persists the workflow plus initial steps. It should stay fast and deterministic. Long-running work belongs in executors so the scheduler can lease it, retry it, cancel it, record failures, and recover from process restarts.

Second, an executor does not update the engine store directly. It writes its intended outcome into `StepContext`: result data, records, artifacts, emitted child steps, or a structured error. The scheduler owns the store write that turns those intentions into durable completion state. That keeps executor code focused on domain work and keeps persistence semantics centralized.

## A minimal workflow in Go

The smallest useful workflow has four parts: runtime construction, executor registration, package registration, and run execution.

```
type HelloInput struct {
    Message string \`json:"message"\`
}

type HelloResult struct {
    Echoed string \`json:"echoed"\`
}

rt, err := workflow.NewRuntime(ctx, workflow.Config{
    Store: workflow.SQLiteStore("./var/hello-engine.db"),
})
if err != nil {
    return err
}
defer rt.Close()

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
```

This example contains the whole design in a compact form. The package describes the initial graph. The executor implements the step kind. The runtime persists and executes the graph. The caller reads the result by workflow ID and step ID.

## Runtime construction and durable ownership

`NewRuntime` is where the public API meets the internal engine. The runtime requires a `StoreConfig`, currently provided by `SQLiteStore(path)`. Opening the store creates the durable backend used by the scheduler. The same store configuration also provides an operator service when possible, which is how `RetryStep` and `CancelRun` are implemented for SQLite.

The runtime constructor does several things in one place:

1. It normalizes unset configuration values.
2. It opens the durable store.
3. It creates a runner registry.
4. It creates a scheduler over the store and registry.
5. It installs the runtime's queue policy provider into the scheduler.
6. It stores optional artifact and projection backends for executor use.

The simplified structure is:

```
func NewRuntime(ctx context.Context, cfg Config) (*Runtime, error) {
    cfg = normalizeConfig(cfg)
    store, closeStore, err := cfg.Store.Open(ctx)
    if err != nil { return nil, err }

    runners := runner.NewRegistry()
    scheduler, err := scheduler.New(store, runners, scheduler.Config{...}, cfg.WorkerID, nil)
    if err != nil {
        _ = closeStore()
        return nil, err
    }

    rt := &Runtime{
        store:       store,
        closeStore:  closeStore,
        runners:     runners,
        scheduler:   scheduler,
        operators:   cfg.Store.OperatorService(),
        artifacts:   cfg.ArtifactStore,
        projections: cfg.ProjectionStore,
        packages:    map[string]*Package{},
        queues:      cfg.Queues,
    }
    scheduler.SetQueuePolicyProvider(rt.queuePolicy)
    return rt, nil
}
```

The runtime is deliberately stateful. It contains the registered packages and executors that make durable step specs executable. The database stores that a step has kind `book/ocr-page`; the runtime's registry stores the Go function that can execute `book/ocr-page` in the current process.

## Packages, entrypoints, and the initial graph

A `Package` is the public unit of workflow creation. It has a stable name, an optional display name, and an entrypoint. `StartRun(ctx, packageName, input, opts...)` looks up the package and asks its entrypoint to build the initial graph.

The package name is more than a display string. It becomes the workflow's `Site` in the underlying engine model. That keeps workflows grouped by the domain that created them and lets steps default to the same site/domain unless they intentionally override it.

`RunBuilder` is the entrypoint's write surface. It can:

- set the persisted workflow name,
- write workflow metadata,
- append initial steps,
- return `StepHandle` values that can be converted into dependencies.

The initial graph path is:

```
flowchart LR
    A[StartRun input] --> B[marshalJSON]
    B --> C[model.WorkflowRun]
    C --> D[RunBuilder]
    D --> E[Entrypoint.Start]
    E --> F[builder.workflow mutations]
    E --> G[builder.steps]
    F --> H[CreateWorkflowParams.Workflow]
    G --> I[CreateWorkflowParams.Initial]
    H --> J[Scheduler.CreateWorkflow]
    I --> J

    style J fill:#d8f3dc,stroke:#2d6a4f
```

A subtle bug found during review came from this exact path. The entrypoint could mutate the workflow through `RunBuilder`, for example by calling `run.Name(...)`, but `StartRun` originally persisted the pre-builder `workflow` variable instead of `builder.workflow`. That meant entrypoint mutations were silently dropped. The fix was one line:

```
workflow = builder.workflow
```

A regression test now verifies that an entrypoint-mutated run name and metadata are both returned and persisted. This is an important API correctness detail because `RunBuilder` is part of the documented public surface. If it accepts workflow mutations, `StartRun` must persist those mutations.

## Executors and typed step functions

Executors are the public replacement for direct use of the lower-level `runner.Runner` interface. An executor has a kind and an execution function. The kind is a durable string stored in `model.OpSpec`; the function is runtime process state registered before execution.

The typed helper is the most common path:

```
workflow.NewTypedExecutor("book/ocr-page",
    func(ctx context.Context, step *workflow.StepContext, input OCRPageInput) error {
        // domain work here
        return step.Result(OCRPageResult{Text: text})
    },
)
```

`NewTypedExecutor[I]` decodes `step.Input` into `I` before calling the user function. This keeps persisted input JSON-compatible while giving application code typed structs. If an executor needs full control over decoding, `NewExecutor` exposes `StepContext` without the typed adapter.

Internally, the public executor is adapted back to the existing runner interface. That adapter is intentionally thin:

```
func (r executorRunner) Run(ctx context.Context, runCtx runner.RunContext) (*model.OpResult, error) {
    step := newStepContext(ctx, runCtx, nil, nil)
    if err := r.executor.Execute(ctx, step); err != nil {
        return nil, err
    }
    return step.opResult(), nil
}
```

When artifact or projection stores are configured, the runtime registers a variant that passes those stores into `StepContext`. The scheduler still sees a runner. The application author sees a workflow executor.

## StepContext as the executor boundary

`StepContext` is the most important executor-facing type. It provides access to the current workflow, current step, lease metadata, scheduler timestamp, input, dependency results, result writers, artifacts, projections, and emitted child steps.

The public methods are designed around the kinds of durable state an executor can legitimately produce:

| Executor need | StepContext method |
| --- | --- |
| Decode current input | `Input(out)` or `RawInput()` |
| Read an earlier step result | `DependencyResult(opID)` or `DependencyData(opID, out)` |
| Store structured output | `Result(data)` |
| Add queryable records to the result | `Record(collection, key, data)` |
| Add inline artifacts | `Artifact(name, contentType, body, opts...)` |
| Store large artifacts externally | `StoreArtifact(name, contentType, body, opts...)` |
| Update a domain projection | `Projection(name)` |
| Add child work | `Emit(id, input, opts)` |

The key implementation detail is that `StepContext` accumulates changes in memory while the executor runs. It does not write the store directly. When the executor returns nil, the adapter converts the accumulated state into `model.OpResult`, and the existing scheduler completion path persists it.

```
func (s *StepContext) opResult() *model.OpResult {
    result := &model.OpResult{
        OpID:        s.run.Op.ID,
        Data:        append(json.RawMessage(nil), s.data...),
        Records:     append([]model.RecordWrite(nil), s.records...),
        Artifacts:   append([]model.ArtifactWrite(nil), s.artifacts...),
        Emitted:     append([]model.OpSpec(nil), s.emitted...),
        CompletedAt: s.run.Now,
    }
    for _, emitted := range result.Emitted {
        result.EmittedIDs = append(result.EmittedIDs, emitted.ID)
    }
    return result
}
```

This design gives executors a clean API while preserving the scheduler's central role in state transitions.

## Dynamic graph expansion

Workflows often cannot know all steps at the beginning. A page fetch discovers pagination. A PDF conversion discovers page images. An OCR pass discovers figure crops that need separate processing. `StepContext.Emit` handles that case by letting a successful executor append child steps.

```
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

Emitted steps are persisted only when the current step completes successfully. That rule prevents failed steps from partially expanding the graph. It also means the result row records both the step output and the emitted child IDs, giving operator tools a traceable parent-to-child relationship.

Initial steps and emitted steps use the same `StepOpts` type. That keeps the graph language consistent across entrypoints and executors:

| Field | Meaning |
| --- | --- |
| `Kind` | Required executor kind. |
| `Queue` | Scheduler queue key. |
| `DedupKey` | Optional deduplication key. |
| `DependsOn` | Required dependencies. |
| `Retry` | Retry policy from the engine model. |
| `Metadata` | Step metadata stored with the op. |
| `Site` | Optional domain/site override. |
| `ParentID` | Optional parent override for emitted steps. |

## Artifacts and projections

The workflow API makes two important distinctions about output data.

The first distinction is between structured result data and large binary data. `step.Result` is for compact JSON data that belongs in the engine result row. `step.StoreArtifact` is for bytes that should live outside the engine DB. The included `FileArtifactStore` writes the bytes under a local root and records a compact JSON reference artifact in the result.

```
flowchart LR
    A[Executor] --> B[step.Result JSON]
    A --> C[step.StoreArtifact bytes]
    B --> D[Engine DB result row]
    C --> E[FileArtifactStore root]
    C --> F[external-artifact-ref in Engine DB]

    style D fill:#d8f3dc,stroke:#2d6a4f
    style E fill:#fde68a,stroke:#92400e
```

The second distinction is between engine state and query-facing domain state. The engine store tracks workflows, ops, dependencies, leases, results, artifacts, and queue limiter state. A projection stores application-specific read models. The included `SQLiteProjectionStore` creates one SQLite database per projection name.

```
projection, err := step.Projection("book-ocr")
if err != nil {
    return err
}

_, err = projection.Exec(ctx, \`
    INSERT OR REPLACE INTO pages(page, status, text)
    VALUES(?, ?, ?)
\`, input.Page, "done", text)
```

This separation prevents domain tables from becoming scheduler tables and prevents scheduler tables from becoming application read models. It also makes it possible to reset, inspect, or migrate projections independently from the engine DB.

## Operator controls and structured errors

Durable execution needs operator controls because long-running systems fail in recoverable and non-recoverable ways. The workflow API exposes two runtime methods:

```
err := rt.RetryStep(ctx, runID, stepID)
err := rt.CancelRun(ctx, runID)
```

These methods delegate to `OperatorService`, which the SQLite store config provides through the existing engineview service. The workflow package does not reimplement retry or cancellation. It exposes the existing mutation surface in workflow vocabulary.

Executors can improve operator visibility by returning structured errors:

```
if temporaryHTTPFailure(err) {
    return workflow.Retryable("fetch_failed", err)
}
if invalidInput(err) {
    return workflow.Permanent("invalid_input", err)
}
return err
```

`Retryable` and `Permanent` return a `workflow.Error` that carries a stable code, message, retryability flag, optional details, and an underlying cause. The scheduler recognizes the embedded `model.OpError` metadata and persists it. That gives dashboards and operators something better than an arbitrary error string.

## Queue policy and worker modes

The runtime can execute work in two modes.

`RunOnce` runs one scheduler cycle. It is best for tests and bounded command-style execution.

```
cycle, err := rt.RunOnce(ctx)
```

`StartWorkers` loops until its context is canceled, an error occurs, or an optional max-cycle limit is reached.

```
err := rt.StartWorkers(ctx,
    workflow.WithWorkerPollInterval(time.Second),
    workflow.WithWorkerMaxCycles(100),
)
```

Queue policies come from `Config.Queues`. The runtime registers `rt.queuePolicy` as the scheduler's queue policy provider. Unknown queues fall back to `model.DefaultQueuePolicy()`. This keeps the public API compact while preserving the engine's existing queue behavior.

## Documentation added for users

The public API now has Glazed help entries in the scraper CLI. The help system was already wired in `pkg/cmd/root.go` through:

```
helpSystem := help.NewHelpSystem()
if err := scraperdoc.AddDocToHelpSystem(helpSystem); err != nil {
    return nil, err
}
helpcmd.SetupCobraRootCommand(helpSystem, rootCmd)
```

The docs are embedded by `pkg/doc/doc.go`:

```
//go:embed topics/* tutorials/*
var docFS embed.FS

func AddDocToHelpSystem(helpSystem *help.HelpSystem) error {
    return helpSystem.LoadSectionsFromFS(docFS, ".")
}
```

Two new help entries were added locally for this API:

- `pkg/doc/tutorials/scraper-workflow-api-getting-started.md` renders as `scraper help scraper-workflow-api-getting-started` and walks through a complete minimal embedded workflow.
- `pkg/doc/topics/scraper-workflow-api-reference.md` renders as `scraper help scraper-workflow-api-reference` and provides the longer user guide and reference.

Those help pages are written for command-line discovery. This Obsidian article has a different purpose: it preserves the project narrative and the design rationale so a future reader can understand why the public API has this shape.

## Tests as executable documentation

The strongest examples are in `pkg/workflow/runtime_test.go`. They exercise the workflow API end to end:

| Test | What it demonstrates |
| --- | --- |
| `TestRuntimeStartRunAndRunOnce` | Registering executors, registering a package, starting a run, executing a root step, emitting a child step, reading results, and reaching succeeded workflow status. |
| `TestRuntimeStartRunPersistsEntrypointWorkflowMutations` | Entrypoint changes to workflow name and metadata survive `StartRun`. |
| `TestRuntimeExternalFileArtifactStore` | External artifacts are stored on disk while compact references remain in the result. |
| `TestRuntimeSQLiteProjectionStore` | Executors can update a package/domain SQLite projection. |
| `TestRuntimeRetryStep` | Operator retry moves a failed step back through execution. |
| `TestRuntimeCancelRun` | Operator cancellation prevents pending work from running. |
| `TestRuntimeStartRunRejectsUnknownPackage` | `StartRun` validates package registration. |

These tests are important because they do not only verify implementation details. They define the public behavior expected from the reusable API.

## Implementation sequence

The branch history shows a clean build-up of the public package:

1. `Add workflow executor facade` introduced executor abstractions over lower-level runners.
2. `Add workflow runtime skeleton` introduced runtime construction, package registration, run start, and scheduler integration.
3. `Add workflow operator controls` exposed retry and cancellation in workflow terms.
4. `Add workflow artifact store` added external artifact storage and references.
5. `Add workflow projection store` added per-domain SQLite projections.
6. Runtime tests exercised the full public path.
7. A review fix persisted `RunBuilder` workflow mutations correctly.
8. GoReleaser placeholders were replaced with the actual `scraper` binary name so release hooks could build the project.
9. Glazed help entries were added for getting started and reference usage.

This order was effective because each step wrapped an existing engine capability rather than inventing a parallel subsystem. The public API grew around proven engine behavior.

## Design rules that make the API reusable

Several rules are worth preserving as the package evolves.

- Entrypoints should describe initial durable work. They should not do long-running domain work because the scheduler cannot lease, retry, or recover work done inside `StartRun`.
- Executor kind names are durable API. They are stored in op specs, so renaming them can strand existing persisted work.
- Executors should write effects through `StepContext`. Direct store writes bypass the scheduler's completion semantics and make failures harder to reason about.
- Large binary output should go through an artifact store. The engine DB should remain a state and metadata store, not a blob store.
- Query-oriented domain state should go into projections. Engine tables should stay focused on scheduling and execution state.
- Runtime lifecycle should be context-driven. Embedded applications should decide when workers start and stop.
- Public builder APIs must persist what they allow callers to mutate. The `RunBuilder` mutation bug is the concrete example of this rule.

## Common failure modes

The API removes a large amount of scheduler/store complexity, but it still has predictable failure modes.

| Problem | Cause | Correction |
| --- | --- | --- |
| `workflow runtime store is required` | `Config.Store` is nil. | Pass `workflow.SQLiteStore(path)` or implement `StoreConfig`. |
| A run cannot start because the package is missing | `StartRun` references a name that was not registered. | Call `RegisterPackage` before `StartRun` and use the exact package name. |
| A step cannot execute | The step's `Kind` does not match any registered executor. | Keep kind names stable and register every executor before starting workers. |
| A step is created but waits forever | Its dependencies did not complete successfully or refer to the wrong op IDs. | Inspect initial `RunBuilder.Step` handles, `workflow.Require`, and emitted dependency specs. |
| A result row has no useful data | The executor returned nil without calling `step.Result`. | Make successful executors write explicit result data, even if compact. |
| `artifact store is not configured` | The executor called `StoreArtifact` but the runtime has no `ArtifactStore`. | Configure `workflow.NewFileArtifactStore` or another store implementation. |
| `projection store is not configured` | The executor called `Projection` but the runtime has no `ProjectionStore`. | Configure `workflow.NewSQLiteProjectionStore` or another projection backend. |
| Entrypoint customization disappears | The runtime persisted the wrong workflow value. | This was fixed by persisting `builder.workflow`; keep regression coverage for future builder fields. |

## What this enables next

The public workflow API makes scraper's engine usable outside the original manifest-driven scraper path. That opens several directions:

- OCR workflows can live as Go packages that embed durable execution directly.
- Services can start runs from HTTP handlers and supervise workers inside their own process manager.
- Tests can run durable workflows deterministically with `RunOnce`.
- Future packages can bring their own artifact and projection backends without changing the scheduler.
- CLI and API layers can share operator actions because retry and cancellation stay in the engineview/operator surface.

The immediate near-term work is documentation and stabilization. The Glazed help entries make the API discoverable from the CLI. The runtime tests define behavior. The remaining design work is to keep the public surface small while adding only the extension points that real embeddings need.

## Files to read

Read these files in order for the fastest understanding of the implementation:

1. `/home/manuel/workspaces/2026-05-20/book-ocr/scraper/pkg/workflow/runtime.go` — runtime creation, package registration, run start, worker modes, workflow/result reads, queue policy provider.
2. `/home/manuel/workspaces/2026-05-20/book-ocr/scraper/pkg/workflow/package.go` — packages, entrypoints, run builder, initial steps, dependencies.
3. `/home/manuel/workspaces/2026-05-20/book-ocr/scraper/pkg/workflow/executor.go` — executor interfaces and runner adapters.
4. `/home/manuel/workspaces/2026-05-20/book-ocr/scraper/pkg/workflow/context.go` — executor context, result accumulation, artifacts, projections, emitted steps.
5. `/home/manuel/workspaces/2026-05-20/book-ocr/scraper/pkg/workflow/artifact_store.go` — external artifact storage.
6. `/home/manuel/workspaces/2026-05-20/book-ocr/scraper/pkg/workflow/projection_store.go` — projection database store.
7. `/home/manuel/workspaces/2026-05-20/book-ocr/scraper/pkg/workflow/operators.go` — retry and cancellation controls.
8. `/home/manuel/workspaces/2026-05-20/book-ocr/scraper/pkg/workflow/errors.go` — structured workflow errors.
9. `/home/manuel/workspaces/2026-05-20/book-ocr/scraper/pkg/workflow/runtime_test.go` — executable public API examples.

## Related notes

- [VLM Separation Benchmark for Book OCR - Prompt Block Layouts and Turn Persistence](https://parc.yolo.scapegoat.dev/note/projects/2026/05/25/article-vlm-separation-benchmark-for-book-ocr-prompt-block-layouts-and-turn-persistence) — related work on durable OCR-oriented workflows and output persistence.
- `scraper help scraper-workflow-api-getting-started` — command-line tutorial for the API.
- `scraper help scraper-workflow-api-reference` — command-line reference for the API.
- PR #3: [https://github.com/go-go-golems/scraper/pull/3](https://github.com/go-go-golems/scraper/pull/3)
