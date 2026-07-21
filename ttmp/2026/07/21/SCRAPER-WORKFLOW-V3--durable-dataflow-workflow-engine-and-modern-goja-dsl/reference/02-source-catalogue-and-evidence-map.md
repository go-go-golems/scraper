---
Title: Source catalogue and evidence map
Ticket: SCRAPER-WORKFLOW-V3
Status: active
Topics:
    - architecture
    - scheduler
    - goja
    - javascript
    - scraper
    - workflows
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/scripts/01-fetch-research-sources.sh
      Note: Reproducibly preserves historical and xgoja reference material
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/scripts/02-inventory-current-scripting.py
      Note: Measures the current JavaScript and raw-operation API surface
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/scripts/03-workflow-dsl-grammar-probe.mjs
      Note: Executable target grammar and deterministic plan probe
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/scripts/05-js-task-bundle-registration-probe.mjs
      Note: Reproducible custom JavaScript task registration experiment
ExternalSources:
    - https://parc.yolo.scapegoat.dev/note/research/kb/projects/scraper
    - https://parc.yolo.scapegoat.dev/note/research/kb/projects/go-go-goja
    - https://parc.yolo.scapegoat.dev/note/research/kb/projects/widget-dsl
Summary: Catalogue of repository evidence, current go-go-goja host-module contracts, historical PARC notes, xgoja references, and reproducible probes used by the workflow v3 design and cookbook.
LastUpdated: 2026-07-21T21:00:00Z
WhatFor: Trace workflow-engine and JavaScript DSL recommendations back to source code, executable probes, and historical architecture notes.
WhenToUse: Read when reviewing the primary design, validating a claim, or resuming implementation after a gap.
---



# Source catalogue and evidence map

## Goal

This document identifies the evidence used to design scraper workflow v3 and its modern `require("workflow")` authoring module. Full external extracts are checked into this ticket's `sources/` directory so an intern can study the historical rationale even if the live site changes.

## Current scraper implementation

| Source | Evidence |
|---|---|
| `pkg/engine/model/types.go` | Workflow, operation, retry, lease, result, and artifact domain records are JSON-shaped engine types. |
| `pkg/engine/scheduler/scheduler.go` | `RunOnce` leases a fixed group and waits on a `sync.WaitGroup`, creating a cycle barrier. |
| `pkg/engine/store/store.go` | Store contracts combine workflow, operation, lease, result, dependency refresh, and queue candidate concerns. |
| `pkg/engine/store/sqlite/migrations/*.sql` | Operation IDs are globally keyed; payload JSON and artifacts live inline in SQLite. |
| `pkg/workflow/runtime.go` | Public Go façade owns runtime, scheduler, package registry, executor registry, and worker lifecycle. |
| `pkg/workflow/package.go` | Entrypoints serialize run input and each initial step input directly. |
| `pkg/workflow/context.go` | Executors accumulate result data, records, artifacts, and emitted steps for atomic completion. |
| `pkg/js/runtime/executor.go` | Execution scripts are CommonJS functions receiving a hand-built mutable `ctx` object. |
| `pkg/sites/submitverbs/runtime.go` | Submission verbs receive a second independently assembled `ctx` surface and emit raw engine operation specs. |
| `pkg/engine/runner/js.go` | Each JS operation constructs a site-scoped Goja executor from manifest modules and database handles. |
| `pkg/runtimeevents` | Scheduler observer events already provide a base for engine-owned projections and telemetry. |
| `pkg/doc/topics/scraper-*.md` | Current public behavior and intended runtime boundaries. |

## Modern local comparison systems

| Source | Reusable pattern |
|---|---|
| `researchctl/pkg/gojamodules/researchctl` | Go-owned fluent builders, immediate configurator callbacks, pure `toSpec()`, validation before execution. |
| `rag-evaluation-system/pkg/gojamodules/rag` | Hidden symbol-backed typed handles, pure Go domain model/compiler, fragments, `validate`, `explain`, compile terminals, precise TypeScript declarations. |
| `rag-evaluation-system/pkg/xgoja/providers/rag` | xgoja/v2 provider registration and generated-host selection. |
| `rag-evaluation-system/pkg/widgetdsl/v3_descriptors.go` | One descriptor inventory drives runtime/help/type parity and prevents surface drift. |
| `rag-evaluation-system/pkg/widgetdsl/testdata/v3/examples` | Executable authoring examples and golden serialized IR. |
| `go-go-goja/pkg/xgoja/providerapi` | Provider packages, module setup context, host services, module factories, TypeScript descriptors, and help sources. |
| `go-go-goja/pkg/xgoja/app` | RuntimePlan, selected module aliases, TypeScript execution/bundling, jsverbs sources, and generated host lifecycle. |

## Current go-go-goja execution-module contracts

| Source | JavaScript contract used in cookbook implementations |
|---|---|
| `go-go-goja/modules/fetch/typescript.go` | Direct `fetch` plus fluent clients, request builders, JSON/text responses, timeouts, and host-owned authentication. |
| `go-go-goja/modules/database/database.go` | Synchronous `query`, `exec`, and explicit transactions; xgoja can disable script-side `configure()`. |
| `go-go-goja/modules/fs/fs.go` | Async/sync filesystem calls, backend capability reporting, host filesystems, and read-only embedded mounts. |
| `go-go-goja/modules/exec/exec.go` | Trusted-runtime `run(command, args)` primitive. |
| `go-go-goja/pkg/xgoja/providers/host/host.go` | Explicit host-module registration, aliases, allow gates, command allowlists, and preconfigured database instances. |
| `go-go-goja/modules/crypto/crypto.go` | UUID/random helpers and incremental SHA-family hashing. |
| `go-go-goja/modules/path/path.go` | Portable path join, resolve, dirname, basename, extension, and relative-path helpers. |
| `go-go-goja/modules/yaml/yaml.go` | YAML parse, stringify, and validation. |
| `go-go-goja/modules/time/time.go` | Runtime-local monotonic `now()` and `since()` measurements. |

These modules are appropriate only in the trusted lease-scoped execution phase.
Descriptor-only authoring runtimes continue to receive none of them.

## Historical extracts saved under `sources/`

1. `01-scraper-project-map.md` — durable workflow and evidence-pipeline project map.
2. `02-go-go-goja-project-map.md` — runtime ownership, modules, xgoja, and generated-host map.
3. `03-widget-dsl-project-map.md` — intent → normalized IR → target architecture.
4. `04-scraper-workflow-api.md` — rationale and implementation history of `pkg/workflow`.
5. `05-goja-fluent-builder-dsls.md` — typed handles, builders, fragments, validation, and DTS parity patterns.
6. `06-designing-dsls-with-go-go-goja.md` — Go/JavaScript ownership and API-shape selection.
7. `07-data-only-vs-host-access-module-split.md` — safe authoring runtime versus explicitly privileged execution runtime.
8. `08-dsl-normalized-config-compiled-plan.md` — raw DSL → normalize → compile → execute rule.
9. `09-xgoja-v2-reference.txt` — current installed xgoja/v2 configuration reference.
10. `10-xgoja-provider-runtime-config-and-host-services.txt` — provider config and host-service lifecycle reference.

Regenerate these with:

```bash
scripts/01-fetch-research-sources.sh
```

## Reproducible experiments

### Current scripting inventory

```bash
python3 scripts/02-inventory-current-scripting.py \
  --repo /absolute/path/to/scraper \
  --out scripts/output/current-scripting-inventory.json
```

Observed baseline:

- scraper pins standalone `go-go-goja v0.8.3` while the local upstream has v0.10.x releases;
- 15 execution scripts and 8 submission verb files;
- execution and submission contexts each install 11 properties/methods independently;
- current site scripts are CommonJS JavaScript, with no TypeScript source under site roots;
- scripts expose raw engine operation fields through `ctx.emit`.

### Workflow DSL grammar probe

```bash
node scripts/03-workflow-dsl-grammar-probe.mjs \
  > scripts/output/workflow-dsl-grammar-probe.json
```

The probe demonstrates the target authoring shape: a pure `workflow.plan(...)` builder, typed hidden references, immediate callbacks, resource classes, map/reduce jobs, RAG task descriptors, validation, explanation, immutable serialization, and no retained JavaScript functions. Its output digest is `sha256:8a0f2909936ed292e511cc2c2cf4cb20dd59890f42487eed8b75e374d43ecab8`.

The probe is not the production implementation. Production must keep canonical types and validation in Go and expose them through `modules.NativeModule` plus an xgoja/v2 provider.

### JavaScript task-bundle registration probe

```bash
node scripts/05-js-task-bundle-registration-probe.mjs \
  > scripts/output/js-task-bundle-registration-probe.json
```

The fixture under `scripts/fixtures/customer-task-bundle/` loads a JavaScript catalog that explicitly registers two namespaced task implementations. The probe hashes the exact bundle files, validates bundle-local entrypoint exports, seals an immutable worker registry generation, and proves that exact task/version/bundle/entrypoint/ABI requirements match while wrong bundle, version, or entrypoint values are rejected.

This demonstrates the proposed extension grammar and exact matching. Production still requires Go-owned bundle/catalog types, trust verification, xgoja phase separation, worker advertisement, fresh lease-scoped runtimes, and store-backed capability/resource admission.

## Key synthesis

The historical and current evidence converges on one design:

```text
JavaScript authoring intent
        ↓
Go-owned normalized workflow IR
        ↓
capability and host-policy compilation
        ↓
immutable executable plan
        ↓
scraper durable dataflow runtime
```

The current `ctx.emit(rawOpSpec)` API skips the normalized IR and compiler layers. Workflow v3 should add those layers rather than extending the raw object surface.

## Related

- [Primary design](../design-doc/01-durable-dataflow-workflow-v3-and-modern-scripting-architecture.md)
- [JavaScript task bundles and worker registries](../design-doc/02-reproducible-javascript-task-bundles-and-worker-registries.md)
- [JavaScript cookbook and execution atlas](03-workflow-v3-javascript-cookbook-and-execution-atlas.md)
- [Investigation diary](01-investigation-diary.md)
