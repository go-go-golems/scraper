---
Title: Workflow V3 product hard-cut design and implementation guide
Ticket: SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER
Status: active
Topics:
    - scraper
    - workflow-v3
    - architecture
    - cleanup
    - cli
    - onboarding
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/engine/model/types.go
      Note: Superseded engine surface to inventory
    - Path: repo://pkg/gojamodules/workflow/authoring.go
      Note: Pure JavaScript workflow authoring
    - Path: repo://pkg/workflowv3/types.go
      Note: Canonical Workflow V3 values
    - Path: repo://pkg/workflowv3runtime/engine.go
      Note: Workflow V3 runtime
ExternalSources: []
Summary: Intern guide for productizing Workflow V3 as Scraper's sole public workflow engine and deleting superseded engines.
LastUpdated: 2026-07-22T23:15:00-04:00
WhatFor: Turn a strong Workflow V3 library into the canonical Scraper CLI, worker, storage, and extension product.
WhenToUse: Read before adding workflow features to Scraper or integrating Scraper with Researchctl.
---


# Workflow V3 product hard-cut design and implementation guide

## Program context

This ticket is part of **EXPERIMENT-PLATFORM-CONVERGENCE** in the Researchctl repository. Siblings are `RESEARCHCTL-EXPERIMENT-PLANS`, `EXPERIMENT-PLATFORM-SCRAPER-RUNNER`, `SCRAPER-WORKFLOW-OBSERVATIONS`, `RAG-V2-WORKFLOW-LOWERING`, `RAG-GEPPETTO-WORKFLOW-OPERATIONS`, `RESEARCHCTL-EXPERIMENT-ANALYSIS`, `RAG-V2-EXECUTION-CUTOVER`, and `TTC-SCRIPTED-EXPERIMENT-ACCEPTANCE`. All have guides under their repository's `ttmp/2026/07/22/` directory.

## Executive summary

Scraper currently contains an older site-oriented engine in `pkg/engine`, historical workflow code in `pkg/workflow`, and the newer Workflow V3 stack. Workflow V3 already has the right general-purpose concepts: canonical plans, durable runs, leases, retries, map/reduce, gates, budgets, isolation, content-addressed artifacts, external operations, and pure JS authoring. The missing work is to make it the product rather than another internal subsystem.

This ticket creates first-class CLI and worker surfaces around Workflow V3, defines task-package registration, adds operator inspection, updates documentation, and removes the superseded engine paths. Compatibility is explicitly not required.

## Architecture orientation

Read in this order:

1. `pkg/workflowv3/types.go` — canonical values and run snapshots.
2. `pkg/workflowv3/compiler.go` — IR to executable plan.
3. `pkg/gojamodules/workflow/authoring.go` — `require("workflow")` authoring.
4. `pkg/workflowv3runtime/engine.go` — submission and task execution.
5. `pkg/workflowv3runtime/dispatcher.go` — work-conserving dispatch.
6. `pkg/workflowv3sqlite/` — durable control state.
7. `pkg/workflowv3/artifacts.go` — CAS artifacts.
8. `pkg/workflowv3/external_operation.go` — side-effect custody.
9. `pkg/workflowv3runtime/isolation.go` — restricted subprocess execution.

Then compare the older public path in `pkg/engine/`, `pkg/js/runtime/`, `pkg/sites/`, and `pkg/cmd/`. The goal is extraction followed by deletion, not coexistence.

## Target product surface

```text
scraper workflow validate workflow.js
scraper workflow explain workflow.js
scraper workflow compile workflow.js --out plan.json
scraper workflow run workflow.js --inputs inputs.json
scraper workflow runs list
scraper workflow runs show <run-id>
scraper workflow runs cancel <run-id>
scraper worker run
scraper task-packages list
```

A single configuration selects:

- Workflow V3 SQLite database;
- artifact root;
- selected task packages;
- worker capacities;
- isolation launcher;
- polling and lease policy.

## Authoring and execution separation

```javascript
const workflow = require("workflow");
const demo = require("demo.tasks");

const definition = workflow.define("map-and-sum", p => {
  const source = p.inputSet("numbers", {
    itemSchema: "number/v1",
    manifestSchema: "number-set/v1",
  });
  const doubled = p.map("double", source, item => demo.double({ value: item }),
    m => m.pageSize(32).maxItems(10000));
  const total = p.reduce("sum", doubled, partition => demo.sum({ partition }),
    r => r.fanIn(16).maxLevels(8));
  p.output("total", total);
});

module.exports = workflow.compile(definition);
```

The authoring runtime exposes descriptors only. Task functions are not serialized JavaScript callbacks. A descriptor selects a host-registered task implementation and binds typed workflow values.

## Task-package API

```go
type TaskPackage interface {
    Identity() PackageIdentity       // name, version, digest
    Catalog() []workflowv3.TaskSpec
    DescriptorModule() workflowmodule.DescriptorModule
    RuntimeModules() map[string]workflowv3runtime.TaskModule
}
```

A sealed registry generation pins exact task implementation identities for each run. Hot reload creates a new generation; existing runs retain their original generation.

## Runtime flow

```text
JS source -> pure authoring -> WorkflowIR -> validation -> WorkflowPlan
                                                     |
inputs -> CAS -> create run -> scheduler -> lease -> task runtime
                                            |          |
                                            |      artifacts
                                            |      external operations
                                            +---- completion/retry
                                                     |
                                              terminal snapshot
```

## Hard-cut plan

Create a feature inventory before deleting anything. Each surviving capability must be classified:

- generic workflow capability to reimplement in V3;
- domain/site package to port as tasks;
- frontend/operator feature to point at V3 read models;
- obsolete capability to delete.

Do not wrap the old engine behind a V3 interface. After operator parity is demonstrated, remove:

- old scheduler/store/runner packages;
- old JS execution semantics that duplicate task packages;
- site bootstrap assumptions from the generic root command;
- old database migrations and API types no longer referenced;
- tests that assert superseded behavior.

## Decisions

### Decision: Workflow V3 is the only workflow model

- **Context:** Multiple engines make every integration choose a generation.
- **Decision:** Product commands, workers, API, and observability use Workflow V3 exclusively.
- **Rationale:** Workflow V3 has stronger identities, durability, isolation, budgeting, and effect custody.
- **Consequences:** Existing sites must be ported or deleted; no compatibility adapter is created.
- **Status:** accepted.

### Decision: task packages, not site manifests, are the extension unit

- **Decision:** A package supplies canonical task specs, JS descriptors, and implementations.
- **Consequences:** Scraping becomes one possible domain package alongside RAG, simulation, or robotics.
- **Status:** proposed.

## Implementation phases

1. Add stable application configuration and dependency construction for V3.
2. Add validate/explain/compile commands with pure JS loading.
3. Add run submission and input staging.
4. Add long-running worker command and graceful shutdown.
5. Add run list/show/cancel operator commands.
6. Add task-package registry and one fixture package.
7. Point API/read models and runtime events at V3.
8. Port only explicitly retained site functionality.
9. Delete old engines, commands, migrations, and documentation.
10. Run dependency scans proving no production import of superseded packages.

## Testing strategy

- Compile golden JS plans.
- Restart between submission and dispatch.
- Lease expiry and stale completion.
- Map/reduce with bounded materialization.
- Gate approval and expiration.
- Budget exhaustion policies.
- Restricted task isolation.
- Artifact integrity and oversize rejection.
- Registry generation pinning.
- CLI smoke tests using temporary DB/artifact roots.
- Negative compile test proving removed packages are not imported.

## Intern guidance

Do not start by deleting `pkg/engine`. First produce a command-level V3 fixture that survives restart. Use the existing Workflow V3 integration tests as the behavioral source, not the old CLI. Keep Cobra/Glazed wiring thin: service constructors should accept the store, artifact store, registry, and capacities explicitly. Once the new path supports submission, execution, and inspection, perform deletion in one focused change and let compilation reveal residual dependencies.

## Completion criteria

A new user can author, validate, run, resume, inspect, and cancel a Workflow V3 JS workflow using the main Scraper binary. No production command imports the old engine or old workflow packages, and the README teaches V3 first.

## Technology primer: plan, run, node, lease, and artifact

A Workflow V3 plan is a portable description of work. It contains nodes, dependencies, typed input/output ports, maps, reductions, gates, budgets, and isolation policy. It does not contain a live database connection, provider client, or JavaScript callback. The plan can therefore be validated and hashed before any work begins.

A run binds that plan to input artifact references. A node is one logical operation in the plan. An attempt is one effort to execute a node. A lease gives one worker temporary authority to complete an attempt. An artifact reference identifies immutable data by schema, digest, size, media type, and locator. These distinctions are what let a process crash without making the workflow unknowable.

```text
WorkflowPlan + input ArtifactRefs
             |
          CreateRun
             |
       runnable node rows
             |
       worker acquires lease
             |
       task reads artifacts
             |
       task writes artifacts
             |
       completion commits or lease expires
```

If a worker dies while holding a lease, the logical node remains. A later attempt can execute it. If a task output was published to the content-addressed store before the completion transaction, the artifact may exist without being authoritative; only committed control state links it as the node output.

## Reading the compiler

`pkg/gojamodules/workflow/authoring.go` creates an in-memory `WorkflowIR`. Calls such as `plan.task`, `plan.map`, and `plan.reduce` build references rather than running work. The state maps Goja objects to typed Go values through runtime-private identity. When `workflow.compile` runs, Workflow V3 validates the IR against a task catalog and emits a plan.

Follow one linear fixture from its JS source through `Author`, `ValidateIR`, `Compile`, and `Digest`. Then compare a map fixture. A map does not eagerly create one database node for every item at authoring time; runtime expansion uses manifests and materialization bounds. That distinction matters for large corpora.

## Reading the runtime

`workflowv3runtime.Engine` coordinates the store, registry, artifact store, and task modules. `Dispatcher.DispatchOnce` asks the store for runnable work and acquires leases. A task receives explicit inputs and an artifact store. Completion is accepted only when lease and attempt identity remain current.

The registry manager pins task implementations. A plan refers to task identity; a run records the registry generation that supplied it. Updating a task package must not silently change a run already in progress.

## Worked restart trace

```text
t=0  submit run-1; nodes prepare -> publish
t=1  worker leases prepare attempt 1
t=2  prepare writes artifact sha256:A
t=3  process crashes before completion transaction
t=15 lease expires
t=16 worker leases prepare attempt 2
t=17 prepare writes identical artifact sha256:A
t=18 completion commits output A
t=19 publish consumes A and run succeeds
```

The content-addressed store deduplicates the bytes, but task semantics must still tolerate repeated execution. Workflow durability does not make arbitrary side effects idempotent. External effects use the operation ledger and task-specific idempotency keys.

## Public configuration model

```yaml
workflow:
  database: /var/lib/scraper/workflow-v3.sqlite
  artifactRoot: /var/lib/scraper/artifacts
  leaseDuration: 30s
  workerConcurrency: 8
  capacities:
    cpu: 8
    provider.generate: 4
  taskPackages:
    - name: fixture
      version: v1
    - name: rag
      version: v2
  isolationLauncher: /usr/bin/workflowv3-isolation-launcher
```

CLI flags may override host locations, but a plan cannot. Secret-bearing configuration remains host-side.

## Operator workflow

An operator should be able to answer four questions without SQL:

1. What plan and inputs does this run use?
2. Which nodes are pending, running, blocked, failed, or complete?
3. Which attempts and external operations occurred?
4. Which output artifacts are authoritative?

`runs show` should present a bounded summary by default and offer JSON for exact structure. A separate command can export complete operation evidence atomically.

## Cutover inventory method

Build the old-to-new inventory as a table:

| Existing capability | Current file/package | V3 replacement | Proof | Action |
|---|---|---|---|---|
| durable scheduling | `pkg/engine/scheduler` | `pkg/workflowv3runtime` | restart test | delete old |
| JS site operation | `pkg/js/runtime` | task package | fixture parity | port/delete |
| workflow status API | old engine view | V3 snapshot projection | API test | rewrite |

Every row ends in retain, port, or delete. “Keep temporarily” requires an explicit blocking ticket and should be rare because compatibility is not required.

## First-week route for an intern

Run `go test ./pkg/workflowv3/... ./pkg/workflowv3runtime/... ./pkg/workflowv3sqlite/... -count=1`. Read the linear engine integration test and reproduce it with a temporary CLI prototype. Next, compile a JS fixture and print its canonical plan. Then kill the worker between lease and completion and observe recovery. Only after this should command wiring begin.

The first production-quality vertical slice is intentionally small: validate, compile, submit, run one worker, and show terminal output. Map/reduce, gates, budgets, and isolation already have package tests; expose them after the core lifecycle is coherent.

## Common mistakes

- Executing JavaScript callbacks as durable tasks makes plans non-portable.
- Loading task implementations without pinning a registry generation breaks replay.
- Returning raw SQLite rows as the public API freezes storage internals.
- Assuming CAS writes imply task completion confuses bytes with authority.
- Preserving old commands after the cutover gives users two incompatible engines.
- Putting domain names such as RAG into generic scheduler types weakens reuse.

## Intern onboarding checklist

The engineer should compile a plan, explain every identity in a run snapshot, recover a stale lease, inspect an artifact by digest, register a fixture task package, and identify all remaining production imports of `pkg/engine` before proposing deletion.

## References

- Program: Researchctl ticket `EXPERIMENT-PLATFORM-CONVERGENCE`.
- `pkg/workflowv3/`
- `pkg/workflowv3runtime/`
- `pkg/workflowv3sqlite/`
- `pkg/gojamodules/workflow/authoring.go`
- `cmd/workflowv3-task-worker/`
- Superseded candidates: `pkg/engine/`, `pkg/workflow/`, `pkg/sites/`, `pkg/js/runtime/`.
- Enables `EXPERIMENT-PLATFORM-SCRAPER-RUNNER` and `RAG-V2-WORKFLOW-LOWERING`.
