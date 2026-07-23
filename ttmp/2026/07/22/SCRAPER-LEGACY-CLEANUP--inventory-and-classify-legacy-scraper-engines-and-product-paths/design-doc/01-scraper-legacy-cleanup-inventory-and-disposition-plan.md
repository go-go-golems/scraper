---
Title: Scraper legacy cleanup inventory and disposition plan
Ticket: SCRAPER-LEGACY-CLEANUP
Status: active
Topics:
    - scraper
    - workflow-v3
    - cleanup
    - architecture
    - onboarding
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/cmd/root.go
      Note: |-
        Current old-engine product command tree
        Current old-engine product command tree retained
    - Path: repo://pkg/cmd/runtime_helpers.go
      Note: |-
        Current old store scheduler and runner construction
        Current old store scheduler and runner construction retained
    - Path: repo://pkg/engine/scheduler/scheduler.go
      Note: Active legacy scheduler
    - Path: repo://pkg/gojamodules/workflow/authoring.go
      Note: Canonical Workflow V3 JavaScript authoring
    - Path: repo://pkg/workflowv3runtime/engine.go
      Note: |-
        Canonical replacement runtime
        Canonical replacement runtime awaiting product cutover
ExternalSources: []
Summary: Evidence-backed disposition of Scraper's active legacy engine, site runtime, APIs, and canonical Workflow V3 packages.
LastUpdated: 2026-07-22T23:45:00-04:00
WhatFor: Prevent blind deletion while planning a hard cut from the active site engine to Workflow V3.
WhenToUse: Read before SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER and before deleting any old engine package.
---



# Scraper legacy cleanup inventory and disposition plan

## Executive summary

Scraper contains two complete architectural generations. The active product path is the older site-oriented engine: the main CLI, worker, API, site registry, JavaScript runtime, services, metrics, and runtime events all import `pkg/engine`. Workflow V3 is a stronger durable engine, but today it is exposed mainly as libraries, tests, an isolation launcher, and a task-worker binary. It has no main product CLI for ordinary submission or inspection.

Consequently, no large old-engine package is safe to delete immediately. The audit found 49 non-test Go files outside `pkg/engine` importing old-engine packages. Deleting `pkg/engine`, `pkg/workflow`, `pkg/js/runtime`, or site infrastructure now would remove the current Scraper product before Workflow V3 has a replacement surface.

The correct cleanup is therefore staged but decisive. Immediate work should remove no active engine code. Freeze it, inventory behaviors, and build the smallest Workflow V3 product slice in `SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER`. Once submission, worker execution, inspection, API projections, and retained task packages exist, delete the old cluster atomically. No compatibility adapter should remain.

The approved no-delete tranche has been completed and freshly validated. No production code was deleted because no active path met the approved safety criteria.

## Program navigation

- Umbrella: `EXPERIMENT-PLATFORM-CONVERGENCE`.
- Cleanup siblings: `RESEARCHCTL-LEGACY-CLEANUP`, `RAG-EVAL-LEGACY-CLEANUP`.
- Replacement: `SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER`.
- Telemetry replacement: `SCRAPER-WORKFLOW-OBSERVATIONS`.
- Integration consumer: `EXPERIMENT-PLATFORM-SCRAPER-RUNNER`.

## Method and baseline

The audit mapped root command registration, old/new package imports, package size, runtime construction, site assets, product entry points, and tests.

```bash
cd /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper
GOWORK=off go test ./... -count=1
```

All packages passed. Workflow V3 runtime tests took roughly 113 seconds and Workflow V3 SQLite tests roughly eight seconds, demonstrating substantial existing coverage.

Approximate Go source sizes:

| Area | Lines |
|---|---:|
| `pkg/engine` | 5,096 |
| `pkg/workflow` | 1,963 |
| `pkg/js/runtime` | 1,250 |
| `pkg/sites` | 2,738 |
| `pkg/workflowv3` | 4,127 |
| `pkg/workflowv3runtime` | 6,275 |
| `pkg/workflowv3sqlite` | 6,856 |

The old system is not a small compatibility shim; it remains a complete product.

## Current active path

`pkg/cmd/root.go:58-66` registers old-engine commands:

```text
scraper engine
scraper worker
scraper api
scraper site
```

`pkg/cmd/runtime_helpers.go:35-72` opens the old SQLite store and registers old HTTP and JS runners. Site manifests are loaded before command construction. The worker, API handlers, submission service, engine views, metrics, runtime events, and site execution all depend on old model/store/scheduler contracts.

```text
site.yaml + JS scripts
      -> site registry
      -> submit verb
      -> old engine workflow/ops
      -> old scheduler leases
      -> HTTP or JS runner
      -> old store/API projections
```

Workflow V3 currently has only `workflowv3-task-worker` and `workflowv3-isolation-launcher` binaries. No main command builds a general V3 engine, submits authored plans, or lists runs.

## Retain as canonical

Retain:

- `pkg/workflowv3`: IR, compiler, artifacts, manifests, gates, budgets, external operations.
- `pkg/workflowv3runtime`: engine, dispatcher, task modules, registry generations, isolation.
- `pkg/workflowv3sqlite`: durable store and projections.
- `pkg/gojamodules/workflow`: pure JS plan authoring.
- `pkg/testfixtures/workflowv3*`: behavioral specifications for linear, HTTP, database, map, reduce, budget, gate, and isolation features.
- `cmd/workflowv3-task-worker` and `cmd/workflowv3-isolation-launcher`.

Retain generated protobuf/runtime event infrastructure only where the V3 product design explicitly reuses it; do not assume old event schemas are canonical merely because they are generated.

## Immediate cleanup completed

### Production Go code: none

This is a deliberate result, not an incomplete audit. Every major legacy cluster participates in the current CLI/API or has an external consumer. Deleting it before a replacement would leave the repository with a tested library but no ordinary product.

### Repository hygiene outside this cleanup's scope

The repository tracks 65 `web/storybook-static` files. These are generated output candidates, but their release/deployment role was not established by this workflow audit. Do not mix frontend artifact cleanup into the engine cutover without a separate evidence check.

## Remove later after Workflow V3 replacement

### 1. Old engine core

Remove `pkg/engine/{model,runner,scheduler,store}` after V3 supports production submission, worker operation, retry, cancellation, and inspection.

**Deletion gate:** main CLI smoke test performs compile -> submit -> worker -> terminal show across restart.

### 2. `pkg/workflow` old convenience runtime

This package wraps the old engine and is also consumed by RAG-eval's current preparation workflow. Remove only after RAG lowering/tasks no longer import it and site workloads have moved to task packages.

**Replacement:** versioned Workflow V3 task packages and plan authoring.

### 3. Site manifest execution stack

`pkg/sites`, `sites/`, submit verbs, per-site SQLite migrations, and dynamic site commands currently define Scraper as a scraping product. Classify each shipped site:

- retain as a domain task package and example;
- preserve only fixtures/test data;
- delete as obsolete.

Do not encode `site` into the generic V3 engine. Scraping becomes one domain package.

### 4. Old JS runtime

`pkg/js/runtime` executes old engine operations and exposes database modules. Replace site operation scripts with task package implementations or deliberately sandboxed task runtime modules. Delete after retained site scenarios pass V3 fixtures.

### 5. APIs and services

`pkg/api`, `pkg/services/engineview`, `pkg/services/submission`, and old API types project the old store. Rebuild public endpoints over Workflow V3 service/read models, then delete old services. Avoid adapters that make V3 imitate old row structures.

### 6. Metrics and runtime events

`pkg/metrics` and `pkg/runtimeevents` wrap old runners and scheduler. `SCRAPER-WORKFLOW-OBSERVATIONS` provides canonical V3 execution metrics. Port useful operator signals, then remove old wrappers and any old-only protobuf events.

### 7. Main command tree

Replace old `engine`, `worker`, `api`, and dynamic `site` command ownership with the V3 product commands defined in `SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER`. Domain task packages may add commands through a generic registration mechanism, but root construction must no longer require site manifests.

## Replacement matrix

| Legacy cluster | Replacement owner | Required proof before deletion |
|---|---|---|
| `pkg/engine` | V3 product cutover | durable CLI lifecycle and restart tests |
| `pkg/workflow` | V3 task packages | no cross-repo imports; fixture parity |
| `pkg/sites` | scraping task package or deletion | explicit disposition per shipped site |
| `pkg/js/runtime` | task module runtime | retained JS behavior tests |
| API/services | V3 read/submission services | endpoint smoke tests |
| metrics/runtimeevents | workflow observations | metric/event mapping tests |
| root commands | V3 CLI | help and operator runbook tests |

## Proposed cutover sequence

1. Freeze features on the old engine.
2. Inventory all 38 files under `sites/` and assign retain/delete disposition.
3. Build V3 dependency construction and `workflow validate/compile/run` commands.
4. Add a long-running V3 worker and run inspection/cancellation.
5. Port one representative HTTP/JS/database site as task packages.
6. Add V3 API/read models and observation export.
7. Update frontend/operator consumers.
8. Remove old root commands and runtime construction.
9. Delete old engine, workflow, JS runtime, site stack, services, metrics, and old-only events.
10. Add CI guards against production imports of `scraper/pkg/engine` and `scraper/pkg/workflow`.

## Why an adapter is rejected

A V3-to-old-engine adapter would preserve old model, store, and API assumptions. An old-to-V3 adapter would force V3 to reproduce operation shapes that predate maps, reductions, gates, budgets, registry generations, and external-operation custody. Since compatibility is not required, both approaches create cost without protecting a supported contract.

The transition uses characterization tests and selected domain ports, not runtime adapters.

## Validation plan for later deletion

```bash
GOWORK=off go test ./... -count=1
go run ./cmd/scraper workflow validate testdata/workflow.js
go run ./cmd/scraper workflow run testdata/workflow.js --inputs testdata/inputs.json
go run ./cmd/scraper workflow runs show <run-id>
rg -n 'scraper/pkg/(engine|workflow)(/|")' --glob '*.go' --glob '!ttmp/**' .
```

The final search must return no production imports. Historical ticket docs can remain.

## Implementation result: approved no-delete tranche completed

The approved remove-now result for Scraper is complete: no active production code was deleted. Fresh source inspection still shows the old engine is the root CLI, worker, API, site, service, metric, and event implementation, while Workflow V3 remains a library/worker foundation without an equivalent primary product surface. Deleting the old cluster would violate behavior preservation.

Fresh verification:

- the working tree remained unchanged by the cleanup implementation;
- `GOWORK=off go test ./... -count=1` passed;
- `make build-go` passed for `scraper`, `workflowv3-task-worker`, and `workflowv3-isolation-launcher`;
- lint passed with zero issues when run with `GOWORK=off`;
- `go mod tidy` proposed only an unrelated direct/indirect classification change for `golang.org/x/sys`, which was discarded so this no-delete cleanup introduced no module churn.

The initially attempted workspace-mode lint exposed an existing local-workspace dependency mismatch between `goja_nodejs` and `goja`. Rerunning the repository validation in its tested module mode (`GOWORK=off`, matching the test/build convention) passed. The old-engine deletion remains gated by `SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER`.

## Risks

The main risk is underestimating site/API behavior embedded outside the engine package. Forty-nine non-test importers show that deletion is a product rewrite, not a package removal. A second risk is accidentally retaining old semantics through compatibility read models. Review replacement APIs on their own V3 concepts.

## Review checklist

- Accept that there is no immediate active-engine deletion tranche.
- Accept Workflow V3 packages as canonical.
- Decide which shipped sites remain valuable.
- Approve product and observation tickets as deletion gates.
- Freeze old engine feature work.
- Do not start deletion until the V3 product slice passes.

## References

- `pkg/cmd/root.go:15-68`
- `pkg/cmd/runtime_helpers.go:24-180`
- `pkg/engine/`
- `pkg/workflow/`
- `pkg/js/runtime/`
- `pkg/sites/`
- `pkg/workflowv3/`
- `pkg/workflowv3runtime/`
- `pkg/workflowv3sqlite/`
- `pkg/gojamodules/workflow/authoring.go`
- `SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER`
- `SCRAPER-WORKFLOW-OBSERVATIONS`
- `EXPERIMENT-PLATFORM-CONVERGENCE`
