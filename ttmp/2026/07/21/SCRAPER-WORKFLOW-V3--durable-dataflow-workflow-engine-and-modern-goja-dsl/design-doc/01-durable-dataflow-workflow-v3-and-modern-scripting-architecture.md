---
Title: Durable dataflow workflow v3 and modern scripting architecture
Ticket: SCRAPER-WORKFLOW-V3
Status: active
Topics:
    - architecture
    - scheduler
    - goja
    - javascript
    - scraper
    - workflows
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system/pkg/gojamodules/rag/module.go
      Note: Modern Go-backed DSL handles validation explanation and compilation pattern
    - Path: repo://pkg/engine/scheduler/scheduler.go
      Note: Current fixed-cycle scheduler and WaitGroup barrier that workflow v3 replaces
    - Path: repo://pkg/engine/store/sqlite/migrations/001_engine_core.sql
      Note: Current globally keyed operations and inline workflow/operation JSON schema
    - Path: repo://pkg/engine/store/sqlite/migrations/002_engine_runtime.sql
      Note: Current dependencies leases mutable results and inline artifact BLOB schema
    - Path: repo://pkg/gojamodules/workflow
      Note: Implemented safe minimal require(workflow) authoring module and DTS
    - Path: repo://pkg/js/runtime/executor.go
      Note: Current execution-time mutable ctx surface and raw operation decoder
    - Path: repo://pkg/sites/submitverbs/runtime.go
      Note: Current independent submission ctx and raw operation decoder
    - Path: repo://pkg/testfixtures/workflowv3database
      Note: Reproducible Slice 5 authored workflow and bundle
    - Path: repo://pkg/testfixtures/workflowv3http
      Note: Reproducible Slice 3 authored workflow and bundle
    - Path: repo://pkg/workflow/context.go
      Note: Current typed Go step context and raw input/emission persistence boundary
    - Path: repo://pkg/workflowv3
      Note: Implemented canonical model compiler bundles registry failures and artifacts
    - Path: repo://pkg/workflowv3runtime
      Note: Implemented fresh Goja runtime engine and end-to-end privacy/restart evidence
    - Path: repo://pkg/workflowv3runtime/dispatcher.go
      Note: Implemented Slice 4 continuous work-conserving dispatch
    - Path: repo://pkg/workflowv3sqlite
      Note: Implemented compact append-only fenced SQLite persistence
    - Path: repo://pkg/workflowv3sqlite/projection.go
      Note: Derived active ready and blocked scheduler projection
ExternalSources:
    - https://parc.yolo.scapegoat.dev/note/research/kb/projects/scraper
    - https://parc.yolo.scapegoat.dev/note/research/kb/projects/go-go-goja
    - https://parc.yolo.scapegoat.dev/note/research/kb/projects/widget-dsl
Summary: Evidence-backed architecture, executable vertical-slice sequence, and implementation guide for a compact durable dataflow engine and typed Goja workflow DSL; the minimal file-transform slice is now running end to end.
LastUpdated: 2026-07-21T22:30:00Z
WhatFor: Implement scraper workflow v3 without repeating the source-bearing payload, fixed-cycle scheduling, untyped scripting, and observability defects found during the real-provider TTC preparation run.
WhenToUse: Read before changing scraper persistence, scheduling, workflow APIs, Goja/xgoja integration, or adapting researchctl and RAG to durable workflow execution.
---




# Durable dataflow workflow v3 and modern scripting architecture

## Executive summary

Scraper already has the difficult nucleus of a durable workflow engine: transactional workflow creation, dependency-aware operations, leases and heartbeats, retries, cancellation states, atomic result/emission persistence, queue limits, artifacts, runtime events, metrics, and restart tests. Workflow v3 should preserve and harden that nucleus. It should not replace scraper with a new orchestrator, and it should not embed RAG semantics in generic infrastructure.

The present engine nevertheless has two architectural defects that make large heterogeneous workloads unsafe and slow:

1. **The durable data plane accepts arbitrary JSON and inline bytes.** A caller can place a complete source-bearing plan in every operation. The TTC preparation incident did exactly that: roughly 1,807 batch descriptions were duplicated into each operation, producing about 14.67 GB of operation input JSON and SQLite/WAL files around 20 GB each. This is not merely inefficient; it violates the intended redaction boundary because source text entered the workflow database.
2. **The scheduler is fixed-cycle rather than work-conserving.** `RunOnce` leases up to `MaxWorkers`, starts that fixed group, and waits for every member before it considers new work. A 35-second generation request therefore prevents an already-free slot from accepting a short embedding request. Resource types also share one undifferentiated worker ceiling.

Workflow v3 addresses both defects and makes scripting a first-order capability:

- durable rows contain immutable identities, digests, small typed references, policy decisions, and redacted outcomes—not whole plans, prompts, source text, vectors, credentials, or provider payloads;
- a continuous dispatcher replenishes a released slot immediately and admits work through independent resource classes;
- every execution creates an immutable, redacted attempt record;
- progress is a store-derived projection rather than an inference from scheduler cycles;
- fan-out is expanded lazily and reductions are sharded with bounded fan-in;
- budgets are reserved and settled transactionally with leases;
- failures use structured classes and stable codes rather than message matching;
- JavaScript authors intent through a typed `require("workflow")` module, while Go owns normalized IR, validation, compilation, durability, permissions, and execution;
- the same pure authoring module can be selected by scraper, researchctl, and RAG xgoja hosts; privileged submission, task-runtime, and operator capabilities remain separate.

The target pipeline is:

```text
TypeScript / JavaScript authoring
              │
              ▼
Go-backed fluent builders and typed symbolic handles
              │  immediate callbacks only
              ▼
Normalized immutable Workflow IR v3
              │  structural + domain validation
              ▼
Host-policy compiler
              │  task catalog + capability manifest + ceilings
              ▼
Immutable executable plan + digest
              │
              ▼
Durable run: compact references, nodes, attempts, leases, events
              │
              ▼
Domain runners: scraper HTTP, RAG providers, CPU transforms, publication
              │
              ▼
External artifacts + store-derived projections + evidence ledger links
```

This is a versioned architecture, not a compatibility shim. A workflow-v3 host must reject a v2 raw operation graph unless an explicit offline migration tool can prove that every input is compact and source-free. Existing v2 runs remain inspectable through existing code; they are not silently decoded as v3.

## Scope and non-goals

### In scope

- generic durable dataflow execution in scraper;
- immutable workflow definitions and compiled plans;
- compact artifact and domain references;
- continuous scheduling, fairness, resource classes, retries, leases, cancellation, and budgets;
- append-only attempts and redacted events;
- lazy map expansion and sharded reduction;
- workflow progress projections and metrics;
- a Go-owned fluent JavaScript/TypeScript authoring DSL;
- xgoja/v2 provider packaging, TypeScript declarations, help, and generated-host selection;
- safe use from scraper, researchctl, and RAG scripts;
- migration, test, benchmark, and operational acceptance criteria.

### Explicit non-goals

- putting RAG provider logic, prepared-corpus schemas, or researchctl ledger semantics into scraper core;
- allowing workflow scripts to open scraper's SQLite database or arbitrary artifact stores;
- persisting provider credentials, authorization headers, prompts, source text, raw provider responses, or secret-bearing errors;
- making JavaScript callbacks durable or replaying JavaScript to recover a compiled run;
- promising bit-for-bit execution order across workers; deterministic identity and output order are required, not deterministic wall-clock interleaving;
- silently converting historical source-bearing operation inputs;
- retaining the current raw `ctx.emit({...engine fields...})` contract as the v3 DSL.

## Design principles and invariants

The implementation is acceptable only if these invariants hold.

### Ownership boundaries

- **Scraper owns generic durability.** It owns run/node state, dependencies, leasing, attempts, resource admission, budgets, artifacts, cancellation, and projections.
- **Domain packages own task semantics.** RAG owns generation, embedding, validation, prepared-corpus publication, and provider error translation. Scraper sites own fetch/extract semantics.
- **Researchctl owns immutable research evidence.** It records identities, manifests, execution references, and reports; it does not become a second scheduler.
- **JavaScript describes intent.** Go normalizes, validates, compiles, and executes that intent.

### Persistence boundaries

- Operation and event rows contain no credentials, headers, prompts, source text, vectors, or raw provider bodies.
- Every durable value has a schema version and, when identity-bearing, a canonical digest.
- Large or sensitive values live behind a typed `ArtifactRef` or are rehydrated from an already-authoritative immutable corpus.
- References are verified before use: expected digest, schema, media type, and domain identity must match.
- Provider output is validated before a result, cache entry, artifact reference, or publication manifest is committed.

### Execution boundaries

- One provider operation corresponds to one bounded provider request.
- Leases, resource reservations, budget reservations, and attempt start are committed atomically.
- Completion requires the current lease token and settles attempt, node state, resource use, and budget in one transaction.
- A late worker cannot commit after lease loss, cancellation, or supersession.
- Retry decisions consume a typed failure, not raw string matching.
- A free resource slot can be replenished without waiting for unrelated running tasks.

### Determinism boundaries

- Definition, normalized IR, compiled plan, input identity, and output manifest all have separate canonical digests.
- Node identity is namespaced by run and deterministic logical key.
- Lazy expansion produces the same logical child keys regardless of restart or page size.
- Concurrent completion order does not determine published order; reducers and manifests use canonical item keys.
- Reopening a published artifact verifies identity and schema before it is considered successful.

## Evidence: current implementation

This section distinguishes code-backed facts from recommendations.

### Durable engine strengths to preserve

The current SQLite schema already separates workflows, operations, dependencies, leases, results, queue limiter state, and artifacts (`pkg/engine/store/sqlite/migrations/001_engine_core.sql:7-38`, `002_engine_runtime.sql:1-52`). Leasing is transactional: it checks active queue leases and token-bucket state, selects one ready operation, writes the lease, and changes the operation to `running` before commit (`pkg/engine/store/sqlite/lease_store.go:15-121`). Heartbeats require the current token and a still-running operation (`lease_store.go:123-163`).

Completion is also transactionally valuable. It checks the lease, writes result/artifact/emitted-child records, deletes the lease, and marks the operation successful before one commit (`pkg/engine/store/sqlite/result_store.go:15-64`). Failure stores the typed `OpError`, updates retry state and next-ready time, and removes the lease in one transaction (`result_store.go:66-112`). These are the semantics to evolve, not discard.

The public Go façade has useful higher-level concepts:

- packages and typed entrypoints (`pkg/workflow/package.go:12-60`);
- deterministic immutable run attachment through `EnsureRun` (`pkg/workflow/runtime.go:278-351`);
- step result, record, artifact, dependency, and child-emission methods (`pkg/workflow/context.go:18-295`);
- external `ArtifactStore` and projection hooks (`pkg/workflow/artifact_store.go`, `pkg/workflow/projection.go`);
- restart-safe aggregate snapshots (`pkg/workflow/runtime.go:353-370`).

The runtime event and metrics layers also provide a useful base. Scheduler events carry workflow, operation, queue, runner, queue wait, attempt, status, and typed error fields (`pkg/engine/scheduler/scheduler.go:42-68`). Prometheus already tracks cycle, lease, completion, failure, retry, queue wait, operation duration, and worker metrics (`pkg/metrics/metrics.go:55-72`), while store snapshots expose status counts (`pkg/engine/store/store.go:49-64`).

### Current durable-data defects

The v2 schema gives `ops.id` a database-global primary key (`001_engine_core.sql:18-35`). Although an operation also has `workflow_id`, two workflows cannot reuse a logical operation ID. This caused the observed `UNIQUE constraint failed: ops.id` collision when a prepared graph was reused.

`workflows.input_json` and `ops.input_json` accept arbitrary inline JSON (`001_engine_core.sql:7-35`). `RunBuilder.Step` and `StepContext.Emit` marshal the caller's entire input value directly into `model.OpSpec.Input` (`pkg/workflow/package.go:84-116`, `pkg/workflow/context.go:247-288`). There is no size ceiling, schema allowlist, secret scanner, or requirement to use references. The store therefore behaved as designed when the TTC adapter accidentally placed a complete source-bearing plan in every input; the design itself supplied no guardrail.

Artifacts are stored as inline SQLite BLOBs by default (`002_engine_runtime.sql:40-52`), and completion inserts those bytes into the main transaction (`result_store.go:40-48`). `StoreArtifact` can use an external store, but this is optional and leaves a JSON reference embedded in a v2 artifact body. Workflow v3 must reverse the default: external content-addressed storage for bytes, compact metadata rows in SQLite, and an explicitly tiny-inline class for exceptional values.

The `results` table has one mutable row per operation (`002_engine_runtime.sql:29-38`). Retried failures overwrite that row through `INSERT OR REPLACE` (`result_store.go:94-96`), so the final row cannot answer: How many attempts ran? How long did each queue wait? Which worker held each lease? Which error class triggered each retry? An append-only attempt ledger is required.

### Current scheduling defect

`RunOnce` refreshes state, obtains a candidate list, and leases up to `MaxWorkers` in round-robin passes (`pkg/engine/scheduler/scheduler.go:223-282`). It then starts one goroutine per leased operation and calls `workers.Wait()` before aggregating outcomes or leasing again (`scheduler.go:283-321`). `Run` invokes another complete `RunOnce` only on the next loop/ticker (`scheduler.go:206-221`).

That code creates a cycle barrier:

```text
lease [G1, G2, E1]
start [G1, G2, E1]
E1 finishes ───── free capacity is idle
G1 finishes ───── still no refill
G2 finishes ───── WaitGroup releases
next RunOnce can lease [G3, G4, E2]
```

The real-provider trace matched this exact shape: most progress gaps were around 34 seconds, 843 scheduler cycles completed exactly three operations, and 865 cycles mixed generation and embedding. A local six-item embedding probe completed in about 1.64 seconds, so embedding was commonly idle behind remote generation rather than being the cycle's bottleneck.

A single `MaxWorkers` limit also conflates incompatible capacities. An OpenAI-compatible generation endpoint, a serial local Ollama embedding model, local CPU reducers, and publication I/O should not compete under one semaphore. Current queue policies provide per-site/per-queue `MaxInFlight` and token buckets, which are useful, but they do not model cross-queue resource pools or reserve budget.

### Current JavaScript surface

The reproducible inventory in `scripts/02-inventory-current-scripting.py` found:

- 15 execution scripts and 8 submission verb files;
- 11 independently assembled members in each execution and submission context;
- seven literal CommonJS require targets;
- no TypeScript site sources;
- raw engine operation fields exposed through `ctx.emit`;
- scraper pinned to standalone `go-go-goja v0.8.3` while the local upstream has newer v0.10.x/xgoja-v2 infrastructure.

Execution scripts are loaded as CommonJS functions and receive a mutable object assembled property-by-property (`pkg/js/runtime/executor.go:54-122`, `162-253`). Submission verbs build a second mutable object with overlapping but different behavior (`pkg/sites/submitverbs/runtime.go:70-145`, `179-238`). Both contain separate raw-op decoders (`executor.go:327-386`, `submitverbs/runtime.go:274-333`). This duplication makes semantic drift likely and asks script authors to understand storage fields such as ID, parent, queue, dedup key, retry policy, dependencies, and metadata.

The right response is not to add more keys to both `ctx` objects. The right response is one Go-owned workflow model exposed through a native module, plus small host-specific capability modules.

### Modern patterns available locally

The repositories already contain the patterns needed for v3:

- researchctl uses a `modules.NativeModule`, Go-backed fluent builders, immediate configurator callbacks, pure `toSpec()`, and validation (`researchctl/pkg/gojamodules/researchctl/module.go:17-80`, `builders.go:13-14`);
- RAG uses private symbol-backed handles, pure Go model/compiler code, `validate`, `explain`, and compile terminals (`rag-evaluation-system/pkg/gojamodules/rag/module.go:23-65`, `91-105`, `555-607`);
- RAG's TypeScript surface declares the same fluent/compile model and has parity tests (`pkg/gojamodules/rag/typescript.go`, `module_test.go:216-304`);
- the RAG xgoja provider couples a module factory with its TypeScript descriptor (`pkg/xgoja/providers/rag/provider.go:13-23`);
- xgoja/v2 gives factories `ModuleSetupContext`, selected aliases, config, assets, and typed host services (`go-go-goja/pkg/xgoja/providerapi/module.go:13-50`);
- xgoja source graphs and commands already support TypeScript sources and generated declaration bundles.

The ticket's `scripts/03-workflow-dsl-grammar-probe.mjs` proves an ergonomic grammar for resource classes, map jobs, reduce jobs, typed references, validation, explanation, and deterministic serialization. The probe deliberately uses JavaScript internals; production must reimplement the model and validation in Go.

## Target architecture

### Component map

```text
┌──────────────────────────────── Authoring plane ────────────────────────────────┐
│                                                                                │
│  researchctl.ts / rag-study.ts / site-workflow.ts                              │
│          │ require("workflow"), require("rag")                                │
│          ▼                                                                     │
│  Goja NativeModules: builders, typed handles, validation, explain              │
│          ▼                                                                     │
│  workflowir.DefinitionV3 ──canonicalize── definition digest                    │
│                                                                                │
└──────────────────────────────────────┬─────────────────────────────────────────┘
                                       │ explicit compile
┌──────────────────────────────────────▼─────────────────────────────────────────┐
│                                Compiler plane                                  │
│  task catalog + schema registry + public capability manifest + policy ceilings │
│          ▼                                                                     │
│  validate → bind inputs → expand static graph → resolve effective policy       │
│          ▼                                                                     │
│  workflowplan.PlanV3 + plan digest + diagnostics + requested/effective diff     │
└──────────────────────────────────────┬─────────────────────────────────────────┘
                                       │ explicit submit by trusted host
┌──────────────────────────────────────▼─────────────────────────────────────────┐
│                                 Control plane                                   │
│  run identity │ nodes │ dependencies │ attempts │ leases │ budgets │ events    │
│          │                                                                     │
│  continuous dispatcher ↔ resource admission ↔ worker registry                  │
│          │                                                                     │
│  lazy expander │ reducers │ cancellation │ projections                         │
└──────────────────────────────────────┬─────────────────────────────────────────┘
                                       │ compact refs only
┌──────────────────────────────────────▼─────────────────────────────────────────┐
│                                  Data plane                                     │
│  corpus/artifact registry │ content-addressed store │ provider/domain runners  │
│          │ rehydrate in process                 │ validate before commit       │
│          ▼                                      ▼                              │
│  ChunkRef / BatchRef / ArtifactRef       output refs / publication manifests    │
└────────────────────────────────────────────────────────────────────────────────┘
                                       │ evidence references
┌──────────────────────────────────────▼─────────────────────────────────────────┐
│ researchctl run ledger: immutable identities, scraper run/plan refs, metrics,   │
│ evaluation products, reports; never a duplicate scheduler or payload store      │
└────────────────────────────────────────────────────────────────────────────────┘
```

### Package boundaries

Use explicit packages rather than growing `pkg/workflow` into one cyclic package:

```text
pkg/workflowir/                 immutable authoring IR and canonical digests
pkg/workflowplan/               compiled plan and policy-resolution diagnostics
pkg/workflowcompile/            IR → plan compiler
pkg/workflowtask/               task descriptors, schemas, catalog interfaces
pkg/workflowref/                ArtifactRef, ChunkRef, BatchRef, ValueRef
pkg/engine/v3/model/            run/node/attempt/lease/failure/budget records
pkg/engine/v3/store/            narrow store interfaces
pkg/engine/v3/store/sqlite/     SQLite implementation and v3 migrations
pkg/engine/v3/dispatcher/       continuous dispatch and local resource admission
pkg/engine/v3/projection/       snapshots, rates, progress, event cursors
pkg/gojamodules/workflow/       pure authoring NativeModule
pkg/gojamodules/workflowsubmit/ privileged submission module, opt-in
pkg/gojamodules/workflowtask/   worker-scoped task context module, opt-in
pkg/xgoja/providers/workflow/   xgoja/v2 provider and descriptors
```

The `v3` directory is intentional during migration. It prevents accidental use of a v2 `OpSpec` in v3 code. Once v2 is removed, packages can be flattened in a separate mechanical change.

## Immutable representations

### Representation pipeline

The system has four different values and must not blur them:

1. **Builder state**: mutable Go-owned authoring objects used only while a script runs.
2. **Workflow IR**: normalized, portable, data-only intent. It contains requested policies and symbolic references, not host credentials or live stores.
3. **Compiled plan**: immutable host-resolved execution contract. It records effective resource classes, task versions, policy ceilings, input bindings, and digests.
4. **Run state**: mutable durable execution facts such as readiness, attempts, leases, results, and cancellation.

A builder is never serialized. JavaScript functions are never retained. Run state is never fed back into the definition digest.

### Workflow IR sketch

```go
// Package workflowir
const SchemaV3 = "scraper-workflow-ir/v3"

type Definition struct {
    SchemaVersion string
    Name          string
    Description   string
    Inputs        []Input
    Resources     []ResourceRequest
    Jobs          []Job
    Outputs       []Output
    Labels        map[string]string // normalized/sorted; never secrets
}

type Job struct {
    Key          string
    Mode         JobMode // task | map | reduce | gate
    From         []PortRef
    Task         workflowtask.Descriptor
    Resource     string
    Queue        QueueRequest
    Retry        RetryRequest
    Timeout      time.Duration
    Budget       BudgetRequest
    Expansion    *ExpansionRequest
    Reduction    *ReductionRequest
}
```

Validation rejects unknown fields after decoding. Maps are normalized into sorted key/value lists before digesting where order matters. The canonical encoder must define treatment of nil versus empty, durations, timestamps, floating point, and Unicode. Do not use ad hoc `json.Marshal` output as the identity contract without tests.

### Compiled plan sketch

```go
// Package workflowplan
const SchemaV3 = "scraper-workflow-plan/v3"

type Plan struct {
    SchemaVersion        string
    DefinitionDigest     Digest
    CompilerVersion      string
    CapabilityDigest     Digest
    InputSchemaDigest    Digest
    Jobs                 []CompiledJob
    Outputs              []CompiledOutput
    RequestedPolicy      PolicySummary
    EffectivePolicy      PolicySummary
    Diagnostics          []Diagnostic
    Digest               Digest // computed with this field absent
}
```

The compiler records both requested and effective policies. A script may request ten concurrent generation calls; a production profile may cap that resource at three. The plan must say both. Silent clamping makes performance evidence uninterpretable.

### Reference types

The base reference is compact and content-verifiable:

```go
type ArtifactRef struct {
    SchemaVersion string            `json:"schemaVersion"`
    Store         string            `json:"store"`
    Locator       string            `json:"locator"`
    Digest        string            `json:"digest"`
    SizeBytes     int64             `json:"sizeBytes"`
    MediaType     string            `json:"mediaType"`
    LogicalType   string            `json:"logicalType"`
    Metadata      map[string]string `json:"metadata,omitempty"`
}
```

`Locator` is an opaque store key, not a filesystem path accepted from an untrusted script. The host resolves it through a named store. Metadata is allowlisted and size-limited.

RAG may define domain references without teaching scraper their semantics:

```go
type ChunkRef struct {
    SchemaVersion  string `json:"schemaVersion"` // rag-chunk-ref/v1
    CorpusIdentity string `json:"corpusIdentity"`
    ChunkID        string `json:"chunkId"`
    TextDigest     string `json:"textDigest"`
}

type BatchRef struct {
    SchemaVersion  string     `json:"schemaVersion"`
    CorpusIdentity string     `json:"corpusIdentity"`
    BatchID        string     `json:"batchId"`
    Items          []ChunkRef `json:"items"`
    Digest         string     `json:"digest"`
}
```

These values contain no text. Immediately before a provider call, a RAG runner asks its authorized corpus resolver for `ChunkID`, verifies `CorpusIdentity` and `TextDigest`, builds the prompt in memory, invokes the provider, validates the response, and drops the materialized source. Scraper sees only the reference and redacted task result.

### Size and redaction enforcement

V3 imposes hard limits at every boundary:

- definition and compiled-plan byte ceilings;
- per-node input reference ceiling (proposed default: 64 KiB, expected median below 4 KiB);
- metadata key count, key length, and value length ceilings;
- tiny-inline artifact class (proposed maximum: 16 KiB) disabled for sensitive logical types;
- artifact reference verification before node creation;
- a secret/source scanner in conformance tests and optional production admission;
- structured errors that discard raw response bodies and headers.

The store accepts `workflowref.ValueRef`, not arbitrary `json.RawMessage`, for v3 node inputs. Domain-specific inline values must be registered schemas whose codecs prove compactness.

## Durable store v3

### Logical schema

A concrete SQLite migration should use composite identities and append-only attempts. Names may change after implementation review, but the relationships must not.

```sql
CREATE TABLE workflow_v3_definitions (
  definition_digest TEXT PRIMARY KEY,
  schema_version TEXT NOT NULL,
  ir_ref_json TEXT NOT NULL,
  created_at_us INTEGER NOT NULL
);

CREATE TABLE workflow_v3_runs (
  run_id TEXT PRIMARY KEY,
  definition_digest TEXT NOT NULL,
  plan_digest TEXT NOT NULL,
  identity_digest TEXT NOT NULL,
  input_ref_json TEXT NOT NULL,
  profile_digest TEXT NOT NULL,
  status TEXT NOT NULL,
  cancel_epoch INTEGER NOT NULL DEFAULT 0,
  created_at_us INTEGER NOT NULL,
  updated_at_us INTEGER NOT NULL,
  UNIQUE(definition_digest, identity_digest)
);

CREATE TABLE workflow_v3_nodes (
  run_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  job_key TEXT NOT NULL,
  item_key TEXT,
  task_kind TEXT NOT NULL,
  task_version TEXT NOT NULL,
  resource_class TEXT NOT NULL,
  queue_key TEXT NOT NULL,
  input_ref_json TEXT NOT NULL,
  status TEXT NOT NULL,
  next_attempt_at_us INTEGER,
  created_at_us INTEGER NOT NULL,
  updated_at_us INTEGER NOT NULL,
  PRIMARY KEY(run_id, node_key)
);

CREATE TABLE workflow_v3_dependencies (
  run_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  depends_on_key TEXT NOT NULL,
  requirement TEXT NOT NULL,
  PRIMARY KEY(run_id, node_key, depends_on_key),
  FOREIGN KEY(run_id, node_key)
    REFERENCES workflow_v3_nodes(run_id, node_key) ON DELETE CASCADE
);

CREATE TABLE workflow_v3_attempts (
  run_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  attempt_no INTEGER NOT NULL,
  worker_id TEXT NOT NULL,
  resource_class TEXT NOT NULL,
  queued_at_us INTEGER NOT NULL,
  started_at_us INTEGER NOT NULL,
  completed_at_us INTEGER,
  status TEXT NOT NULL,
  failure_json TEXT,
  usage_json TEXT,
  PRIMARY KEY(run_id, node_key, attempt_no)
);

CREATE TABLE workflow_v3_leases (
  run_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  attempt_no INTEGER NOT NULL,
  token_hash TEXT NOT NULL,
  worker_id TEXT NOT NULL,
  cancel_epoch INTEGER NOT NULL,
  acquired_at_us INTEGER NOT NULL,
  expires_at_us INTEGER NOT NULL,
  PRIMARY KEY(run_id, node_key)
);

CREATE TABLE workflow_v3_outputs (
  run_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  output_name TEXT NOT NULL,
  ref_json TEXT NOT NULL,
  committed_at_us INTEGER NOT NULL,
  PRIMARY KEY(run_id, node_key, output_name)
);

CREATE TABLE workflow_v3_events (
  event_id INTEGER PRIMARY KEY AUTOINCREMENT,
  run_id TEXT NOT NULL,
  node_key TEXT,
  attempt_no INTEGER,
  kind TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  occurred_at_us INTEGER NOT NULL
);
```

Additional tables hold resource counters, token buckets, budget accounts/reservations, expansion cursors, and reduction partitions. All have run-scoped/composite foreign keys.

### Identity rules

- `run_id` is externally useful and unique.
- `node_key` is logical and unique only within a run.
- the internal identity is `(run_id, node_key)`; never concatenate unescaped strings as the database key;
- map child keys are derived from `job_key` plus canonical `item_key`, not array position;
- reduction partition keys include level and a digest of sorted members;
- attempt number increments transactionally under the node lease transaction;
- artifact identity is a content digest plus logical output name; artifact-store locator is not identity.

This directly removes the v2 global `ops.id` collision.

### Transaction boundaries

**Submit transaction**:

1. verify definition, plan, profile, and input digests;
2. insert or attach to the immutable run identity;
3. create only static root nodes and lazy expansion cursors;
4. append `run.created`;
5. commit.

**Lease transaction**:

1. select one eligible node under fairness/resource filters;
2. verify dependencies and cancellation epoch;
3. atomically reserve resource and budget;
4. increment attempt number and insert `attempt.running`;
5. insert lease containing the same attempt and cancellation epoch;
6. mark node running and append `node.leased`;
7. commit.

**Success transaction**:

1. require current, unexpired lease token and cancellation epoch;
2. validate output references and expected cardinality;
3. insert outputs;
4. finish attempt with redacted usage;
5. settle budget and release resource;
6. advance expansion/reduction state;
7. mark node succeeded, remove lease, append events;
8. commit.

**Failure transaction**:

1. require current lease;
2. finish attempt with typed redacted failure;
3. settle known usage and release resource;
4. calculate retry from failure class and compiled retry policy;
5. mark node ready at next time or terminally failed;
6. remove lease, append events;
7. commit.

No provider call occurs inside a database transaction.

## Continuous work-conserving dispatcher

### Runtime model

Replace `RunOnce` as the production scheduling primitive with a long-lived dispatcher. Keep a deterministic `DispatchOnce` test hook that leases at most one item or performs one maintenance action; do not retain fixed worker batches.

```text
store notification ─┐
worker completion ──┤
lease expiry tick ──┼──► wake dispatcher ─► compute available capacities
retry deadline ─────┤                         │
cancellation ───────┘                         ▼
                                      lease one eligible node
                                               │
                                capacity remains? ── yes ──► lease again
                                               │ no
                                               ▼
                                         wait for wake
```

Illustrative Go pseudocode:

```go
func (d *Dispatcher) Run(ctx context.Context) error {
    completions := make(chan Completion, d.localLimit)
    wake := d.notifier.Subscribe()

    for {
        if err := d.drainCompletions(ctx, completions); err != nil {
            return err
        }

        started := false
        for d.localResources.AnyAvailable() {
            grant, err := d.store.LeaseEligible(ctx, d.admissionSnapshot())
            if err != nil { return err }
            if grant == nil { break }

            if !d.localResources.Acquire(grant.Claims) {
                _ = d.store.ReleaseUnstartedLease(ctx, grant)
                break
            }
            started = true
            go d.execute(ctx, grant, completions)
        }

        if started { continue }
        select {
        case <-ctx.Done(): return ctx.Err()
        case <-wake:       // new node, retry, cancellation, or policy update
        case c := <-completions:
            if err := d.commitCompletion(ctx, c); err != nil { return err }
        case <-d.maintenanceTicker.C:
            if err := d.maintain(ctx); err != nil { return err }
        }
    }
}
```

The actual implementation must ensure a completion cannot block while its channel is full, and store notifications may be coalesced. Correctness comes from transactional leasing, not from notification delivery.

### Resource classes

A resource class is a named capacity domain with optional rate and budget dimensions:

```go
type ResourceClass struct {
    Name          string
    MaxInFlight   int
    RateLimit     *TokenBucket
    Fairness      FairnessPolicy
    Scope         Scope // process | database | tenant | provider
    RequiredTags  map[string]string
}
```

Example profile:

```yaml
resources:
  llm.generate.primary:
    maxInFlight: 2
    rate: { requestsPerMinute: 30, burst: 2 }
  embedding.ollama.local:
    maxInFlight: 1
  cpu.reduce:
    maxInFlight: 4
  publication.prepared-corpus:
    maxInFlight: 1
```

A task requests a symbolic class such as `generation.primary`; the compiler binds it to an available effective class. The worker advertises supported task kinds and resource tags. A worker never accepts a task merely because its global goroutine pool has space.

For the TTC workload, two generation slots and one embedding slot can remain busy independently. When embedding completes after two seconds, the dispatcher can start the next embedding even if both generation calls are still running.

### Fairness and queue policy

Recommended ordering:

1. reject nodes whose resource/budget/capability cannot be admitted;
2. group by tenant or trust domain if configured;
3. deficit-round-robin across workflows within a resource class;
4. apply queue priority inside the workflow;
5. choose oldest ready time and stable `(run_id, node_key)` as ties.

Rate limits remain transactional and database-scoped. Process-local semaphores are only a fast local ceiling. Multi-process correctness must use store-backed active grants/leases.

### Cancellation

Cancellation increments `run.cancel_epoch`, marks not-started nodes canceled, revokes or flags active leases, and emits a durable event. Dispatchers stop leasing that run immediately. Workers receive context cancellation through their lease watcher. A late success with the old epoch is rejected; it may not publish an artifact reference or overwrite cancellation.

A publication task receives an additional rule: if cancellation was requested before its commit transaction, publication cannot become authoritative.

## Attempts, failures, retries, and progress

### Immutable attempts

Each execution, including lease-loss recovery, is an attempt. Required fields are:

- run/node/attempt identity;
- worker identity and resource class;
- ready/queued/start/end timestamps;
- queue wait and run duration (derived, not trusted input);
- terminal outcome: succeeded, retryable-failed, terminal-failed, canceled, or lease-lost;
- typed failure code/class and safe message;
- redacted usage: request count, token counts if provider supplies them, output bytes, and cost estimate/actual;
- lease-loss and heartbeat facts;
- compiler/task versions.

Raw errors remain process logs under the host's secure logging policy, not workflow events.

### Typed failure contract

```go
type FailureClass string
const (
    FailureTransport       FailureClass = "transport"
    FailureRateLimit       FailureClass = "rate-limit"
    FailureProvider5xx     FailureClass = "provider-5xx"
    FailureMalformedOutput FailureClass = "malformed-output"
    FailureValidation      FailureClass = "validation"
    FailureIdentity        FailureClass = "identity"
    FailureConfiguration   FailureClass = "configuration"
    FailureBudget          FailureClass = "budget"
    FailureCanceled        FailureClass = "canceled"
    FailureLeaseLost       FailureClass = "lease-lost"
    FailureInternal        FailureClass = "internal"
)

type Failure struct {
    Class       FailureClass
    Code        string
    Retryable   bool
    SafeMessage string
    Provider    string // public provider identifier only
    Phase       string
    HTTPStatus  int    // optional, no headers/body
    CauseDigest string // optional correlation digest, not raw text
}
```

The domain adapter translates provider-specific errors at its boundary. For example, the RAG generator should classify a syntactically malformed combined response as `malformed-output` with code `RAG_GENERATOR_COMBINED_JSON`. The compiled policy can retry that class with bounded attempts and jitter. `strings.Contains(err.Error(), ...)` is prohibited in v3 retry code.

### Progress as a projection

A workflow snapshot should answer, from committed state:

```json
{
  "runId": "...",
  "status": "running",
  "nodes": {"total": 5431, "ready": 91, "running": 3, "succeeded": 5325, "failed": 12},
  "attempts": {"total": 5384, "retried": 12, "leaseLost": 0},
  "activeByResource": {"llm.generate.primary": 2, "embedding.ollama.local": 1},
  "oldestRunningAgeSeconds": 42.1,
  "rates": {"nodesPerMinute1m": 5.2, "nodesPerMinute15m": 4.8},
  "budgets": {"requestsUsed": 5384, "requestsRemaining": 616},
  "expansion": {"knownItems": 1807, "materializedItems": 1807},
  "eventCursor": 91823
}
```

Events are useful for streaming but not authoritative by themselves. Snapshots are calculated from durable tables or maintained transactionally as rebuildable projections. A projection rebuild must reproduce the same counts from base rows.

Metrics should add:

- queue wait and run duration by resource/task/outcome;
- active and configured capacity by resource class;
- dispatcher idle-with-ready-work duration;
- lease loss and heartbeat failures;
- attempts and retries by typed failure class/code;
- expansion backlog and reduction partitions;
- budget reserved/used/remaining;
- reference-resolution and validation failures;
- durable input and output-reference byte histograms.

Never label Prometheus series with run ID, node key, chunk ID, model URL, or unbounded provider errors.

## Lazy expansion and sharded reduction

### Why expansion must be lazy

A dataflow definition may describe 1,807 chunks, but submission need not create every downstream node and must never copy the 1,807-item plan into each node. A map job stores:

- the source artifact-set reference;
- a deterministic item-key extractor;
- a task template containing a symbolic item reference;
- an expansion cursor and page digest.

An expander materializes bounded pages as capacity/backlog requires. Restarting reads the committed cursor. Re-expanding the same page uses deterministic keys and is idempotent.

```go
func ExpandPage(run Run, job MapJob, cursor Cursor, limit int) (Page, error) {
    items := source.ListAfter(cursor, limit)
    nodes := make([]Node, 0, len(items))
    for _, item := range items {
        key := job.Key + "/" + CanonicalItemKey(item)
        nodes = append(nodes, job.Instantiate(key, Ref(item)))
    }
    return Page{Nodes: nodes, Next: items.NextCursor, Digest: Digest(nodes)}, nil
}
```

The expander transaction verifies the prior cursor, inserts missing nodes, records the page digest, advances the cursor, and appends one summary event. A crash cannot skip or duplicate logical children.

### Bounded reduction

A single root node should not depend directly on thousands of full outputs. Use a reduction tree with a compiled maximum fan-in:

```text
1,807 map outputs
    ├── shard reduce level 0: groups of 128 → 15 compact shard manifests
    ├── validation reduce level 1: groups of 16 → 1 validated root manifest
    └── publication: atomic PreparedCorpusStore.Put(rootRef)
```

Reducers consume sorted references, verify expected item identity/cardinality, and produce compact immutable manifests. The root publication transaction happens only after every required partition succeeds and reopen validation passes. A malformed partition is isolated and retryable without rerunning unrelated successful partitions.

### Ordering

Scheduling order is intentionally nondeterministic. Logical output order is deterministic:

1. source item has a canonical item key;
2. map output includes that key;
3. shard membership is based on sorted keys, not completion order;
4. reducer output is sorted and digested;
5. root manifest records count and ordered child digests;
6. reopening verifies the same order and count.

## Transactional budgets

Budgets are correctness constraints, not dashboard annotations.

Supported dimensions should include:

- provider request count;
- estimated and actual input/output tokens;
- monetary micro-units in a declared currency;
- durable artifact bytes;
- attempts;
- optional wall-clock deadline.

At lease time the store reserves a conservative estimate. If no reservation is possible, the node remains budget-blocked or the run fails according to compiled policy. Completion settles actual usage and releases the remainder. Lease expiry transfers or releases the reservation according to whether the provider request may have occurred; ambiguous external side effects are charged conservatively.

```go
type BudgetPolicy struct {
    Limit       Usage
    OnExhausted ExhaustionPolicy // fail-run | block | require-approval
}

type Usage struct {
    Requests     int64
    InputTokens  int64
    OutputTokens int64
    CostMicros   int64
    ArtifactBytes int64
}
```

Scripts may request a smaller budget, but they cannot raise the host ceiling. Currency/model price tables are compiler inputs with their own digest; credentials and provider headers are not.

## Task contracts and domain integration

### Task descriptor

A task descriptor is portable data, not executable JavaScript:

```go
const TaskDescriptorSchema = "scraper-task-descriptor/v1"

type Descriptor struct {
    SchemaVersion string
    Kind          string
    Version       string
    InputSchema   string
    OutputSchema  string
    Config        json.RawMessage
    ConfigDigest  Digest
}
```

The compiler resolves `(kind, version)` in a host task catalog. The catalog supplies config schema validation, input/output schemas, required capabilities, cost estimator, default retry classification, and runner factory. Unknown task kinds or versions fail compilation.

### Cross-module interoperability

Private Goja symbols are excellent for protecting a builder inside one module, but RAG's private symbols cannot be directly read by an independently implemented workflow module. Do not couple modules through shared JavaScript symbol identity.

Use this boundary:

- a domain module such as `rag` returns a frozen object implementing the TypeScript `WorkflowTask<I,O>` shape;
- `toTaskDescriptor()` produces a plain, versioned descriptor value;
- `workflow` strictly decodes and canonicalizes that value;
- the compile host validates the descriptor against its task catalog;
- TypeScript brands improve editor safety, but runtime validation never trusts a brand.

This preserves module independence and allows a descriptor to cross process/repository boundaries as canonical JSON.

### Domain-authored JavaScript task bundles

Task catalogs must not imply that every domain task is compiled into scraper. A domain developer can publish an immutable JavaScript task bundle containing:

- a registration catalog with namespaced task keys and versions;
- input/config/output schemas and semantics;
- bundle-local execution entrypoints;
- a safe authoring-side descriptor module and TypeScript declarations;
- declared modules/capabilities/resources;
- deterministic dependency locks, self-tests, provenance, digest, and optional signature.

Workers explicitly load an approved bundle lock during boot/reload, evaluate each catalog in a registration-only Goja runtime, build and self-test a candidate registry, then atomically seal and advertise a registry generation. Compiled plans pin the exact task kind/version, bundle digest, entrypoint, and task ABI. The dispatcher leases only to workers advertising that exact implementation and compatible resource/isolation capabilities.

Ordinary `require()` calls do not mutate a process-global registry. Task attempts run their pinned entrypoint in a fresh lease-scoped Goja runtime with `workflow/task` and only allowlisted services. Go validates inputs and outputs outside JavaScript. Mutually untrusted bundles require process/container isolation because Goja is not a hostile-code sandbox.

See [Reproducible JavaScript task bundles and worker registries](02-reproducible-javascript-task-bundles-and-worker-registries.md) for the bundle layout, registration API, worker lifecycle, exact implementation matching, rolling upgrades, security model, and acceptance criteria.

### RAG runner sequence

For a combined-generation node:

1. receive a `ChunkRef` and immutable provider-profile reference;
2. resolve the verified corpus through the RAG host capability;
3. fetch the chunk by ID and verify text digest;
4. render the prompt in process memory;
5. make one bounded provider request under context timeout;
6. decode and validate schema, chunk identity, uniqueness, and cardinality;
7. write validated domain output to a content-addressed store;
8. return only its reference and redacted usage;
9. let scraper commit attempt/output state.

Embedding follows the same sequence and never places vectors in operation JSON. Publication consumes shard manifests, verifies all counts/digests, atomically writes the prepared-corpus root, and reopens it before reporting success.

## Modern `require("workflow")` DSL

### Goals

The module should feel as modern as the researchctl and RAG DSLs while staying generic:

- fluent builders with immediate configurator callbacks;
- typed symbolic input/item/output handles;
- task descriptors supplied by domain modules;
- resource, queue, retry, timeout, budget, map, reduce, dependency, and output APIs;
- pure `toIR`, `validate`, `explain`, and compile terminals;
- deterministic serialization and digest;
- precise TypeScript declarations;
- runtime/help/type definitions generated from one descriptor inventory;
- no store, filesystem, network, clock, random, credentials, or provider access in the authoring module.

### Proposed authoring grammar

```ts
import workflow = require("workflow");
import rag = require("rag");

const plan = workflow.define("ttc-preparation", p => {
  const chunks = p.inputSet<rag.ChunkRef>("chunks", {
    schema: "rag-chunk-ref-set/v1",
    role: "corpus-chunks",
  });

  p.resource("generation", r => r
    .class("llm.generate.primary")
    .maxInFlight(2)
    .rate({ requestsPerMinute: 30, burst: 2 })
    .fairness("workflow-round-robin"));

  p.resource("embedding", r => r
    .class("embedding.ollama.local")
    .maxInFlight(1));

  p.resource("publication", r => r
    .class("publication.prepared-corpus")
    .maxInFlight(1));

  const generated = p.map("generate", chunks, chunk =>
    rag.tasks.combinedPrepare({
      chunk,
      profile: p.inputRef("generationProfile"),
      questionsPerChunk: 4,
    }),
    j => j
      .resource("generation")
      .timeout("3m")
      .retry({
        maxAttempts: 3,
        classes: ["transport", "rate-limit", "provider-5xx", "malformed-output"],
        backoff: { kind: "exponential-jitter", min: "2s", max: "45s" },
      })
      .budget({ requests: 1 }));

  const embedded = p.map("embed", generated, representation =>
    rag.tasks.embedRepresentation({
      representation,
      profile: p.inputRef("embeddingProfile"),
    }),
    j => j.resource("embedding").timeout("3m"));

  const shards = p.reduce("reduce-shards", embedded,
    partition => rag.tasks.reducePreparedShard({ partition }),
    j => j
      .resource("publication")
      .fanIn(128)
      .orderedBy("itemKey"));

  const published = p.task("publish",
    rag.tasks.publishPreparedCorpus({ shards }),
    j => j
      .after(shards)
      .resource("publication")
      .timeout("10m")
      .retry({ maxAttempts: 1 }));

  p.output("preparedCorpus", published.output("manifest"));
});

const validation = workflow.validate(plan);
if (!validation.ok) throw new Error(workflow.formatDiagnostics(validation));

module.exports = workflow.compile(plan);
```

Every callback above is invoked immediately while building. The map callback receives a typed symbolic `ItemRef<ChunkRef>` once, not once per durable item. The resulting task template contains an expression reference such as `{$ref: "map-item", source: "chunks"}`. No callback/function enters the IR or plan.

### Builder state machine

Builder methods should reject invalid order immediately:

- a plan cannot define two inputs/jobs/resources/outputs with the same key;
- a job can select one task and one primary resource;
- `fanIn` is valid only for reduction;
- output ports must exist on the task descriptor;
- handles belong to one plan; cross-plan handles fail;
- compile closes the builder; later mutation fails;
- `toIR()` returns a deep immutable copy.

Use hidden per-runtime symbols or private Go wrappers for handles exactly as the RAG DSL does. Exposed object keys should be intentional API, not mutable internal state.

### Pure module exports

Recommended public surface:

```ts
export function define(name: string, build: (p: PlanBuilder) => void): Workflow;
export function fragment(name: string, build: (p: PlanBuilder) => void): WorkflowFragment;
export function validate(value: Workflow | WorkflowIRV3): ValidationReport;
export function explain(value: Workflow | WorkflowIRV3): WorkflowExplanation;
export function toIR(value: Workflow): WorkflowIRV3;
export function digest(value: Workflow | WorkflowIRV3 | WorkflowPlanV3): string;
export function compile(value: Workflow | WorkflowIRV3, options?: CompileOptions): WorkflowPlanV3;
export function formatDiagnostics(report: ValidationReport): string;
export const schemas: Readonly<Record<string, string>>;
```

`compile` is pure with respect to execution. Its module setup receives only a public `CompilationProfile`/task catalog—not a scraper store. If no compile service was selected, `compile` returns a precise capability error while `toIR`, `validate`, and `explain` continue to work.

### TypeScript core sketch

```ts
export interface WorkflowTask<I, O> {
  readonly kind: string;
  readonly version: string;
  readonly inputSchema: string;
  readonly outputSchema: string;
  toTaskDescriptor(): TaskDescriptorV1;
  readonly __workflowTask?: { input: I; output: O };
}

export interface ValueRef<T> {
  readonly schema: string;
  readonly __workflowValue?: T;
}

export interface SetRef<T> extends ValueRef<readonly T[]> {
  readonly itemSchema: string;
}

export interface JobRef<T> extends ValueRef<T> {
  output<N extends keyof T & string>(name: N): ValueRef<T[N]>;
}

export interface PlanBuilder {
  input<T>(name: string, options: InputOptions): ValueRef<T>;
  inputSet<T>(name: string, options: SetInputOptions): SetRef<T>;
  inputRef<T = unknown>(name: string): ValueRef<T>;
  resource(name: string, build: (r: ResourceBuilder) => void): this;
  task<I, O>(name: string, task: WorkflowTask<I, O>, build?: (j: JobBuilder) => void): JobRef<O>;
  map<I, O>(name: string, source: SetRef<I>, task: (item: ValueRef<I>) => WorkflowTask<I, O>, build?: (j: MapJobBuilder) => void): SetRef<O>;
  reduce<I, O>(name: string, source: SetRef<I>, task: (partition: SetRef<I>) => WorkflowTask<readonly I[], O>, build?: (j: ReduceJobBuilder) => void): SetRef<O>;
  output<T>(name: string, value: ValueRef<T>): this;
}
```

Production declarations should avoid semantic `any`; `unknown` is acceptable at truly dynamic boundaries. Golden tests must assert runtime export/declaration/help parity.

### Descriptor-driven surface

Define resource/job/retry/budget method descriptors once in Go and derive:

- runtime export registration;
- method help and examples;
- TypeScript declarations or declaration fragments;
- validation vocabulary;
- parity tests.

This follows the widget DSL descriptor inventory and avoids three hand-maintained surfaces drifting apart.

## Capability separation

### Module matrix

| Module | Runtime | Authority | Default availability |
|---|---|---|---|
| `workflow` | authoring/compiler | build, normalize, validate, explain, optionally compile against public profile | safe; scraper/researchctl/RAG hosts |
| `rag` | authoring/compiler | build RAG definitions and task descriptors | safe when selected |
| `workflow/submit` | trusted control host | submit/attach/cancel/snapshot through narrow service | opt-in; never generic study eval |
| `workflow/task` | one leased worker task | read bound refs, write validated refs/progress through lease-scoped service | worker-only |
| `fetch:*`, `db:*`, `fs:*` | trusted leased worker task | profile-configured HTTP, database, and attempt filesystem access | worker-only; explicit bundle grant |
| `exec:*` | trusted isolated worker task | allowlisted subprocess execution | worker-only; process/container isolation required |
| `crypto`, `path`, `yaml`, `time` | leased worker task | current go-go-goja utility contracts | worker-only when declared |
| `workflow/operator` | administrative host | inspect/retry/cancel/repair under authenticated policy | explicit operator command only |

Do not expose a generic `store` object to any module. Host services are narrow Go interfaces:

```go
type SubmitService interface {
    Submit(context.Context, workflowplan.Plan, InputBindings) (RunRef, error)
    Ensure(context.Context, workflowplan.Plan, Identity, InputBindings) (RunRef, error)
    Snapshot(context.Context, RunID) (Snapshot, error)
    Cancel(context.Context, RunID, ReasonCode) error
}

type TaskService interface {
    InputRef(name string) (workflowref.ValueRef, error)
    Resolve(ref workflowref.ValueRef, expectedSchema string) (io.ReadCloser, error)
    PutValidated(name string, object ValidatedObject) (workflowref.ArtifactRef, error)
    Progress(ProgressUpdate) error
}
```

A task service is bound to one current lease. It cannot read another run, choose arbitrary store locators, alter dependencies, or submit a workflow.

Trusted first-party bundle code may additionally import profile-selected current
go-go-goja modules. Aliases carry authority: `fetch:partner` owns its base URL,
authentication, redirect policy, timeout, and response limits;
`db:destination` owns a preconfigured handle and rejects JavaScript
`configure()`; `fs:input` is a read-only attempt mount; and `exec:media` exposes
only an allowlisted command set inside process/container isolation. The catalog,
compiled job, registry generation, and worker advertisement all record the exact
module aliases. No alias is available merely because its provider was linked
into the worker binary.

This is a trusted-code boundary, not a claim that Goja is a hostile-code
sandbox. Third-party or broadly privileged bundles require subprocess or
container isolation before admission.

### Researchctl integration

Researchctl scripts should be able to import the pure module:

```js
const research = require("researchctl");
const rag = require("rag");
const workflow = require("workflow");

const study = rag.study(/* immutable experimental intent */);
const preparation = workflow.define(/* durable execution intent */);

module.exports = research.project("rag-ttc", p => p
  .experiment("real-provider-v2", e => e
    .spec(study.compileStudy(inputBindings))
    .execution({
      engine: "scraper-workflow/v3",
      plan: workflow.compile(preparation),
    }))
).toSpec();
```

The script can describe both evidence and execution, but execution happens only when trusted Go command code validates the research spec, stores its immutable identity, submits the compiled plan through a configured adapter, and records the returned run reference. The authoring runtime itself has no submission service.

Researchctl records:

- research spec digest;
- workflow definition and plan digests;
- immutable input manifest/profile digests;
- scraper run ID and event/projection cursors;
- validated output manifest digests;
- evaluation/report references.

It does not copy source-bearing workflow rows or provider payloads into the ledger.

### RAG integration

The RAG module should add `rag.tasks.*` factories that return generic workflow task descriptors. RAG compilation remains responsible for product/study semantics; the workflow adapter converts approved RAG execution intent into dataflow jobs.

Recommended dependency direction:

```text
scraper workflow contracts  ←  RAG workflow adapter
          ↑                         ↑
          └──── researchctl execution binding
```

Scraper must not import RAG. Researchctl core should not import scraper internals; an integration package or generated host selects both providers.

### Scraper site migration

A site can split authoring from task code:

```text
sites/nereval/
  workflows/seed.ts          # pure workflow definition
  tasks/fetch-list.ts        # leased task entrypoint
  tasks/extract-detail.ts
  lib/extract.ts
  site.yaml
```

A submission verb loads/compiles `workflows/seed.ts` and trusted Go submits the resulting plan. Task scripts receive a lease-scoped typed context through `workflow/task`, not a mutable raw operation object.

```ts
import task = require("workflow/task");
import db = require("site-db");

module.exports = task.define("nereval.extract-detail/v1", async ctx => {
  const input = ctx.input<NerevalDetailRef>("detail");
  const html = await ctx.resolveText(input.html, { schema: "scraper-html-ref/v1" });
  const record = extractDetail(html);
  await ctx.writeRecord("items", input.itemKey, record);
  return ctx.output("record", record);
});
```

Even here, `resolveText` is schema-checked and lease-scoped. For sensitive corpora, a task-specific native runner may be safer than JavaScript materialization.

## xgoja/v2 packaging plan

### Native module

Implement `pkg/gojamodules/workflow` with:

- a `modules.NativeModule` loader;
- `modules.TypeScriptDeclarer` or a `spec.Module` factory;
- per-Goja-runtime hidden symbols/handle registry;
- Go model codecs and canonical compiler adapter;
- no package-global mutable builder state;
- explicit module descriptor inventory;
- tests that create an engine runtime and call `require("workflow")`.

The standalone registration path remains useful for unit tests and scraper's current runtime while xgoja adoption proceeds.

### Provider package

Implement `pkg/xgoja/providers/workflow/provider.go` following the RAG provider pattern:

```go
const PackageID = "github.com/go-go-golems/scraper"

func Register(reg *providerapi.ProviderRegistry) error {
    return reg.Register(providerapi.Provider{
        ID: PackageID,
        Modules: []providerapi.Module{
            {
                Name: "workflow",
                TypeScript: workflowmodule.TypeScriptModule(),
                NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
                    profile, err := compilationProfileFrom(ctx.Config, ctx.Host)
                    if err != nil { return nil, err }
                    return workflowmodule.New(profile).Loader, nil
                },
            },
        },
        HelpSources: workflowhelp.Sources(),
    })
}
```

Privileged modules belong in separate provider modules/capabilities so selecting `workflow` never accidentally selects submission authority. Host service lookup validates exact Go types and fails closed.

### Generated host configuration

A research/RAG authoring host can select aliases declaratively:

```yaml
runtime:
  modules:
    - provider: github.com/go-go-golems/researchctl
      name: researchctl
      as: researchctl
    - provider: github.com/go-go-golems/rag-evaluation-system
      name: rag
      as: rag
    - provider: github.com/go-go-golems/scraper
      name: workflow
      as: workflow
sources:
  - id: studies
    kind: jsverbs
    path: experiments
    typescript:
      enabled: true
```

Use the current xgoja/v2 provider APIs rather than inventing scraper-specific module discovery. Pin an intentional go-go-goja version and run all affected tests with `GOWORK=off` to prove published module compatibility.

### TypeScript execution

TypeScript is an authoring convenience, not durable state:

1. xgoja discovers `.ts` source and generated declarations;
2. the host compiles/bundles it deterministically;
3. Goja executes the bundle to produce Workflow IR or a compiled plan;
4. the normalized value and digests are stored;
5. workers execute the plan without rerunning authoring TypeScript.

Record source/bundle/compiler digests for provenance. Do not store unredacted source code inside every node.

## Validation and compilation

### Validation stages

1. **Builder validation**: local method order, names, handle ownership, duplicate keys.
2. **IR structural validation**: schema versions, unknown fields, cycles, missing ports/resources, size limits.
3. **Task validation**: kind/version exists, config schema and config digest valid, input/output schemas compatible.
4. **Capability validation**: target workers/providers/resource classes exist; required host services selected.
5. **Policy validation**: requested retry, timeout, queue, budget, and concurrency fit host ceilings.
6. **Binding validation**: every input reference has expected schema/digest and no unexpected input is supplied.
7. **Privacy validation**: compactness and forbidden-field/content checks.
8. **Plan validation**: canonical digest, compiler/capability/profile identities, acyclic compiled dependencies.
9. **Submission validation**: immutable identity attach-or-conflict semantics.

Validation occurs before persistence. Diagnostics include stable code, severity, JSON pointer/job key, safe message, and suggested fix.

### Example diagnostics

```json
{
  "ok": false,
  "diagnostics": [
    {
      "code": "WORKFLOW_RESOURCE_UNKNOWN",
      "severity": "error",
      "path": "/jobs/generate/resource",
      "message": "resource request generation cannot be bound by profile local-real-v2"
    },
    {
      "code": "WORKFLOW_INPUT_TOO_LARGE",
      "severity": "error",
      "path": "/bindings/chunks",
      "message": "inline input is 2839412 bytes; expected rag-chunk-ref-set/v1 reference"
    }
  ]
}
```

Diagnostics must not echo the rejected value.

## Security and privacy model

### Threats addressed

- accidental source/prompt duplication into operation JSON;
- secret-bearing headers or errors entering events;
- untrusted authoring scripts obtaining store/network/database access;
- a task resolving arbitrary artifact locators;
- late workers publishing after cancellation or lease expiry;
- denial of service through giant graphs, metadata, fan-in, retries, or artifact bytes;
- cross-run operation ID collision;
- policy escalation by script-authored concurrency/budget settings.

### Controls

- data-only authoring runtime and explicit privileged modules;
- strict schema decoding with unknown-field rejection;
- compact reference types and row size limits;
- content-addressed artifact verification;
- host policy ceilings and capability catalog;
- lease-scoped task services;
- hashed lease tokens at rest if raw token recovery is unnecessary;
- typed redacted failures and allowlisted metadata;
- graph/job/item/fan-in/attempt limits;
- transactional cancellation epoch and budget admission;
- privacy conformance tests that inspect SQLite, WAL, events, reports, and logs.

A production test must seed recognizable canary strings as credential, header, prompt, and source text, run the workflow, checkpoint SQLite/WAL, and prove none appears in durable workflow tables/events/reports. Artifact stores that are intentionally authorized to hold corpus content are tested separately under their own policy.

## API compatibility and migration

### Versioning position

Workflow v3 is a new contract:

- schema names include `/v3`;
- store tables/types are distinct during migration;
- commands choose engine version explicitly;
- v3 rejects v2 `OpSpec` and raw `ctx.emit` objects;
- no automatic compatibility decoder is added.

A site or integration migrates by compiling a v3 definition and updating tests/docs. If a v2 durable run is in progress, finish or cancel it with the v2 engine. Source-bearing v2 TTC databases are diagnostic-only and must never be imported or published.

### Deprecation path for `ctx.emit`

1. mark the raw API as v2 in docs and runtime warnings;
2. provide v3 examples and conversion guide;
3. migrate one simple site end to end;
4. migrate remaining site definitions and task contexts;
5. stop enabling v2 submission verbs by default;
6. remove v2 after no supported manifests select it.

Do not implement `ctx.emit` as a thin adapter to v3. It cannot prove compact references, task schemas, or capability intent and would preserve the architectural ambiguity.

## Implementation plan

### Delivery strategy: executable vertical slices

Implementation proceeds as walking vertical slices rather than building every
store, compiler, dispatcher, and module layer before exercising real work. Each
slice must produce a reopenable artifact and add only the machinery needed by
its workload. The original phases below remain the complete architecture
horizon; this sequence changes delivery order, not the end-state invariants.

#### Slice 1 — real linear file transform

Run a real JSONL customer normalization and validation bundle from a directly
constructed canonical Go plan. Implement the minimum v3 model/store, compact
artifact refs, exact task identity, static sealed registry, fresh Goja runtime,
`workflow/task`, guarded `fs:input`, output validation, append-only attempts,
lease/cancellation fencing, reopen, and SQLite/WAL privacy scan.

Exit evidence: kill/restart succeeds, output digest is stable, source canaries
are absent from SQLite/WAL/events, and wrong bundle/entrypoint/ABI workers
cannot lease the node.

#### Slice 2 — minimal `require("workflow")` authoring DSL

Implement `define`, typed `input`, descriptor-backed `task`, `after`, named
`output`, `toIR`, `validate`, `digest`, and `compile`. Compile the cookbook
linear transform and require byte-identical normalized IR and plan digests to
the Slice 1 Go golden. No callback/function may survive serialization.

Exit evidence: the JavaScript-authored plan executes through the same durable
path as the direct Go plan and reopens the same validated output shape.

#### Slice 3 — real allowlisted HTTP snapshot

Add `fetch:public`, typed transport/status failures, response limits,
cancellation, retry, redaction, and an HTTP resource class. Start with a bounded
static article list before introducing lazy maps.

#### Slice 4 — work-conserving dispatch

Replace fixed scheduler cycles with completion-driven refill, independent
resource capacity, deterministic single-action test hooks, fairness, and
blocked-reason projections. Prove a released HTTP slot starts ready work while
unrelated slow tasks remain active.

#### Slice 5 — real database synchronization

Add Go-preconfigured query/write aliases, disabled script-side `configure()`,
transactions, stable node idempotency keys, typed transient failures, and a
crash-after-side-effect test that cannot apply the logical write twice.

#### Slice 6 — lazy map expansion

Add typed set refs, deterministic item keys, expansion cursors, bounded pages,
and restart-safe cardinality accounting. Exercise hundreds or thousands of
real items before optimizing for larger plans.

#### Slice 7 — bounded reduction

Use the word-count workflow to add deterministic bounded fan-in, intermediate
manifests, completion-order-independent root digests, and restart tests across
multiple concurrency levels.

#### Slice 8 — rolling registry generations

Add candidate validation/self-tests, atomic sealing, immutable advertisements,
generation reference counting, coexistence/draining, and quarantine. Prove old
runs stay pinned to digest A while new runs use digest B.

#### Slice 9 — budgets and projections

Add transactional reservation/settlement and authoritative progress, resource,
rate, blocked-reason, expansion, reduction, and attach/reopen projections using
the already-running file, HTTP, database, and reduction workloads.

#### Slice 10 — lease-free gates

Implement durable waiting, authenticated operator events, expiry/cancellation,
and continuation without holding a worker lease or resource slot.

#### Slice 11 — stronger process isolation

Move attempts that use writable filesystems, broad network policy, or
`exec:*` into constrained subprocess/container workers before accepting
untrusted publishers.

#### Slice 12 — RAG preflight and TTC

Run source-canary, storage-amplification, malformed-output retry, independent
resource, cardinality, cancellation/restart, publication, reopen, and redaction
preflights. Only then run a paid sample and the full TTC study.

### Immediate implementation tranche

The current ticket implements Slices 1 and 2 completely. It must not defer the
identity, compact-reference, attempt, fencing, registry, privacy, or reopen
invariants merely because later scheduler/map/budget features are not yet
needed. The first implementation may use a deterministic one-node-at-a-time
executor, but its store and plan identities must be compatible with the final
work-conserving dispatcher.

### Phase 0 — freeze contracts and reproduce defects

Files:

- ticket experiments under `scripts/`;
- new architecture decision records under `pkg/doc` or ticket docs;
- benchmark fixtures with synthetic canary source.

Work:

1. retain the current inventory and grammar probe as baselines;
2. add a storage-amplification reproducer that attempts the 1,807-item pattern with synthetic text;
3. add a scheduler timeline test that demonstrates the `WaitGroup` barrier;
4. capture v3 invariants as failing acceptance tests;
5. freeze reference/failure/IR/plan schemas.

Exit criteria: defects reproduce without real provider calls or sensitive source.

### Phase 1 — compact references and storage guardrails

Files/packages:

- `pkg/workflowref/`;
- `pkg/workflowtask/`;
- `pkg/engine/v3/model/`;
- `pkg/engine/v3/store/sqlite/`;
- external artifact store adapters.

Work:

1. implement canonical digests and strict codecs;
2. implement `ArtifactRef`, compact value refs, size limits, and metadata allowlists;
3. create v3 run/node/dependency/output tables with composite keys;
4. require compact refs for node input;
5. make external content-addressed artifacts the normal path;
6. implement attach-or-conflict immutable run identity;
7. add SQLite/WAL privacy scans.

Exit criteria: the synthetic 1,807-item workload creates source-free bounded rows and operation IDs can repeat across runs.

**This phase is the minimum prerequisite for another real-provider TTC preparation.**

### Phase 2 — append-only attempts, typed failures, and projections

Work:

1. add attempts/events and typed failure contract;
2. translate existing runner failures at domain boundaries;
3. persist lease/attempt completion atomically;
4. add rebuildable snapshots, event cursor, active-by-resource, rates, and oldest age;
5. add bounded-cardinality metrics;
6. verify malformed provider JSON can be retried under explicit policy.

Exit criteria: every retry and lease loss is inspectable without raw provider content; projections agree with base rows after rebuild.

### Phase 3 — continuous dispatcher and resource classes

Work:

1. implement transactional `LeaseEligible` with resource grants;
2. build long-lived dispatcher and completion-driven wakeup;
3. separate local semaphores from global store-backed admission;
4. implement fairness and resource-class token buckets;
5. implement cancellation epoch and late-completion rejection;
6. keep deterministic single-action hooks for tests.

Exit criteria: a short operation starts in a released slot before unrelated long operations finish; generation and embedding capacities remain independently saturated.

### Phase 4 — lazy expansion, sharded reduction, and budgets

Work:

1. implement deterministic paged map expansion;
2. implement bounded reduction partitions and root manifests;
3. add transactional budget reservations/settlement;
4. add expansion/reduction/budget projections;
5. add restart and crash-point tests around every transaction.

Exit criteria: a 1,807-item workflow resumes after crashes without missing/duplicated item keys, and publication occurs once after validated complete reduction.

### Phase 5 — Go model, compiler, and pure Goja module

Files:

- `pkg/workflowir/`;
- `pkg/workflowplan/`;
- `pkg/workflowcompile/`;
- `pkg/gojamodules/workflow/`.

Work:

1. implement immutable Go IR and canonical encoding;
2. implement task catalog, schema compatibility, capability/policy compiler;
3. implement fluent builders and typed handles;
4. implement immediate callbacks, map item expressions, reduce partition expressions;
5. implement `validate`, `explain`, `toIR`, `compile`, and digest terminals;
6. generate TypeScript/help from descriptors;
7. port grammar probe to native-module integration/golden tests.

Exit criteria: equivalent JS and Go authoring produce byte-identical normalized IR and plan digests; no functions survive serialization.

### Phase 6 — xgoja/v2 provider and TypeScript hosts

Work:

1. upgrade/pin go-go-goja intentionally;
2. add workflow provider registration and TypeScript descriptor;
3. add safe authoring host profile;
4. add separate submission/task/operator capabilities;
5. add generated-host and TypeScript source tests;
6. document aliases/config/services.

Validation uses `GOWORK=off` for scraper and every affected consumer.

Exit criteria: generated scraper, researchctl, and RAG test hosts can `require("workflow")`; safe hosts cannot import privileged modules.

### Phase 7 — domain migrations

Order:

1. simple scraper demo site;
2. one real scraper site;
3. RAG task descriptors and compact materializers;
4. researchctl execution binding;
5. complete TTC preparation plan;
6. remaining site scripts.

Each migration updates docs and removes its raw-operation path rather than retaining dual behavior inside one script.

Exit criteria: exact-profile TTC preflight proves compact persistence, malformed-output retries, continuous scheduling, independent resources, restart, publication/reopen, and redaction before a full run begins.

### Phase 8 — retire v2

- disable new v2 submissions;
- retain read-only inspection for a declared period;
- remove duplicated JS context emit decoders;
- remove v2 scheduler after all supported runs finish;
- publish migration and operational recovery documentation.

## Implemented vertical slices 1–8

Slices 1–8 are now executable rather than design-only. The exact identity,
compact-reference, attempt, fencing, privacy, and reopen contracts from the
minimal tranche remain unchanged while HTTP, resources, retries, and database
side effects use the same durable path.

| Boundary | Implementation evidence |
|---|---|
| Canonical model/compiler | `pkg/workflowv3` strict IR, catalog, exact identity, validation, digest, bundle, registry, failure, and artifact contracts |
| Minimal authoring DSL | `pkg/gojamodules/workflow` plus IR, plan, and authoring DTS goldens |
| Real bundle | `pkg/testfixtures/workflowv3linear` paired workflow and `execution/tasks.cjs` source |
| Fresh task runtime | `pkg/workflowv3runtime/task_runner.go` with `workflow/task`, exact task-ABI DTS, and only declared aliases |
| Durable execution | `pkg/workflowv3runtime/engine.go` and `pkg/workflowv3sqlite` compact schema/store |
| Restart/privacy proof | `TestEngineRunsAuthoredWorkflowAcrossRestartWithoutPersistingSource` over 12,000 JSONL rows |
| Fencing/identity proof | cancellation, expired lease, concurrent lease, and wrong bundle/entrypoint/ABI tests |
| Allowlisted HTTP | `pkg/testfixtures/workflowv3http`, exact `fetch:public`, typed retry, response limit, redirect denial, cancellation, redaction, and reopen tests |
| Work-conserving dispatch | `pkg/workflowv3runtime/dispatcher.go`, transactional resource admission, per-resource fairness, blocked projections, and mixed-resource timeline test |
| Database synchronization | `pkg/testfixtures/workflowv3database`, exact preconfigured `db:sync`, denied `configure()`, transaction marker, post-commit crash/restart, and failure-isolation tests |
| Lazy maps | `pkg/testfixtures/workflowv3map`, typed set/map authoring, strict manifests, deterministic child keys, paged expansion, backpressure, ordinary dynamic attempts, ordered publication, and 1,807-item restart/privacy evidence |
| Bounded reductions | `pkg/testfixtures/workflowv3reduce`, typed reduce authoring, immutable bounded partitions, lease-local member rehydration, multi-level recovery, deterministic root publication, failure isolation, and 257-item evidence |
| Rolling registries | `pkg/workflowv3runtime/registry_manager.go`, atomic activation, exact generation acquisition before lease persistence, draining, reference-safe removal, quarantine, restart reconstruction, and A/B executable-byte evidence |

The implemented DSL surface includes `define`, typed `input` and `inputSet`,
descriptor-backed `task`, `map`, `reduce`, `after`, `output` and `outputSet`, `toIR`,
`validate`, `digest`, and `compile`. Goja objects are opaque handles backed by
Go-owned runtime maps; JavaScript properties do not constitute identity.

The store persists plan and ref metadata but never artifact bodies. Task inputs
are copied into a temporary read-only mount containing only bound refs. Output
bytes enter the content-addressed artifact store before their refs are
transactionally committed, so a crash may leave an unreferenced immutable
object but cannot publish an invalid partial output.

`Engine.RunOne` remains the deterministic bounded-action hook. It can advance
one map page or publication and one task lease. Production-style v3 execution
uses a long-lived completion-driven `Dispatcher`: it interleaves bounded map
and reduction control actions, leases until
all compatible resource capacities are full and refills immediately after each
completion. SQLite admission counts active nodes by resource class, while
fairness counters are keyed by `(run_id, resource_class)`. No fixed worker batch
or `WaitGroup` barrier remains in the v3 dispatch path.

Task bundles canonically pin resource class and fixed bounded retry policy.
Retryable failures finish one immutable attempt, set a durable `ready_at`, and
later lease a new attempt. Queue projections derive active resources, ready
count, and bounded blocked reasons without creating a second mutable queue.

Host policy values remain outside plans. `fetch:public` receives origins,
timeout, response bound, redirect checking, and disabled credential sources
from Go. Worker boot rejects empty/wildcard public allowlists or enabled
credential sources; transport rejects URL userinfo and authorization/cookie
headers. `db:sync` receives a Go-preconfigured target handle and rejects
JavaScript `configure()`. The database task's logical operation key derives
from run/node identity, so a post-commit crash can retry without duplicating the
write.

Focused validation:

```text
GOWORK=off go test ./pkg/workflowv3 \
  ./pkg/gojamodules/workflow \
  ./pkg/workflowv3runtime \
  ./pkg/workflowv3sqlite -count=1

GOWORK=off go test -race \
  ./pkg/workflowv3runtime \
  ./pkg/workflowv3sqlite -count=1
```

The public implementation overview is
`pkg/doc/topics/scraper-workflow-v3-minimal-runtime.md`. Its historical slug is
stable, while its title and content now cover Slices 1–8.

## Testing strategy

### Unit tests

- canonical encoding and digest golden vectors;
- nil/empty/duration/Unicode/map-order normalization;
- strict unknown-field rejection;
- reference schema/digest/media/size validation;
- task catalog lookup and schema compatibility;
- builder state and cross-plan handle rejection;
- retry policy by every failure class;
- budget reservation and settlement arithmetic;
- deterministic map/reduction keys;
- cancellation epoch and lease-token checks.

### Store and concurrency tests

Run against SQLite with real transactions and fake clock:

- two processes race to lease the same node; exactly one wins;
- global resource capacity is never exceeded;
- attempt number remains monotonic across expiry/restart;
- stale completion cannot commit;
- completion atomically settles output, attempt, lease, resource, budget, and event;
- crash after each statement/transaction boundary recovers safely;
- two runs reuse the same `node_key` without collision;
- snapshot rebuild matches incremental projection;
- WAL checkpoint/reopen preserves all identities.

Run `go test -race` on dispatcher/store suites.

### Scheduler timeline tests

Use controllable runners:

```text
capacity generation=2, embedding=1
G1=10s, G2=10s, E1=1s, E2=1s
```

Required assertion: E2 starts shortly after E1 finishes and before G1/G2 finish. A second test verifies no generation task consumes the embedding grant. A fairness test keeps one large workflow from starving a small workflow.

### JavaScript/TypeScript tests

- `require("workflow")` through `engine.New()`;
- native module fluent example to golden IR;
- `validate`, `explain`, `compile`, and digest parity;
- immediate callbacks called exactly once;
- no callback retained in IR;
- task descriptor interchange with `require("rag")`;
- DTS contains every runtime export and no semantic `any`;
- xgoja generated host compiles and executes `.ts` source;
- safe host cannot resolve `workflow/submit`, `workflow/task`, database, process, filesystem, or network modules;
- privileged module rejects absent/wrong host service types.

### Privacy tests

Use canaries:

```text
SECRET_CANARY_...
AUTH_HEADER_CANARY_...
PROMPT_CANARY_...
SOURCE_TEXT_CANARY_...
PROVIDER_BODY_CANARY_...
```

After success, retry, cancellation, lease loss, restart, and report generation, scan:

- SQLite main database;
- WAL and journal;
- event payloads;
- snapshots and metrics exposition;
- researchctl ledger and reports;
- structured logs under test capture.

Only the explicitly authorized corpus/artifact fixture may contain source canaries. All other hits fail the test.

### Scale benchmark

Synthetic exact-shape benchmark:

- 1,807 source items;
- one generation map, one embedding map, reduction shards of 128, and publication;
- at least twelve injected malformed-output retries;
- randomized restarts and lease losses;
- no real provider or source.

Proposed acceptance bounds, to be confirmed on the target machine:

- median node input row below 4 KiB;
- no node input above 64 KiB;
- SQLite main + WAL below 512 MiB at peak, excluding external artifacts;
- zero canary source/prompt/credential hits;
- exactly one logical successful output per item key;
- output root cardinality exactly 1,807;
- projection counts match base rows;
- no resource limit overshoot;
- work-conserving timeline passes.

The storage bound is deliberately generous relative to compact references and more than an order of magnitude below the observed tens-of-gigabytes failure.

### End-to-end TTC preflight

Before a full real-provider run:

1. freeze corpus, dataset, model/provider profiles, source snapshot, workflow definition, compiled plan, and compiler capability digests;
2. use a small representative corpus through real providers;
3. inspect durable rows and privacy scans;
4. inject malformed generation output and verify typed retry;
5. restart worker during generation and embedding;
6. verify attach-to-same-identity and mismatch failure;
7. verify continuous refill and independent resource occupancy;
8. verify cardinality, ordering, citation/identity propagation, publication, and reopen;
9. verify researchctl records only immutable references and redacted metrics;
10. approve full run only if every check passes.

## Acceptance criteria

### Durability and identity

- [ ] Same immutable identity attaches to one run; any identity mismatch fails explicitly.
- [ ] Node keys are run-scoped and reusable across runs.
- [ ] Every attempt, lease, retry, result, artifact, and event is attributable to run/node/attempt.
- [ ] Restart and lease expiry do not duplicate logical outputs.
- [ ] Late or canceled workers cannot commit.

### Compactness and privacy

- [ ] V3 nodes accept only registered compact values/references.
- [ ] Whole workflow plans, source text, prompts, vectors, credentials, headers, and provider bodies never enter workflow rows/events/reports.
- [ ] Large bytes use content-addressed external storage; inline bytes are explicitly tiny and policy-approved.
- [ ] The 1,807-item benchmark meets row/database bounds and canary scans.

### Scheduling and resources

- [ ] Dispatcher replenishes a free slot without waiting for unrelated work.
- [ ] Resource classes have independent capacities and database-scoped admission.
- [ ] Fairness prevents starvation.
- [ ] Rate and budget reservations are transactional with leasing.
- [ ] Cancellation stops new admission and rejects stale completion.

### Attempts, failures, and observability

- [ ] Every execution has an append-only attempt row.
- [ ] Retry classification uses stable typed codes/classes only.
- [ ] Malformed provider JSON can be policy-retried and is visible as a redacted attempt.
- [ ] Snapshots expose counts, active-by-resource, ages, rates, retries, expansion, and budgets.
- [ ] Projection rebuild matches base state.
- [ ] Metrics use bounded labels.

### Dataflow semantics

- [ ] Map expansion is deterministic, paged, idempotent, and restart-safe.
- [ ] Reduction fan-in is bounded and output ordering is canonical.
- [ ] Provider responses validate before persistence.
- [ ] Root publication occurs atomically only after complete validation and successful reopen.

### Scripting

- [ ] Scraper, researchctl, and RAG authoring hosts can `require("workflow")`.
- [ ] JavaScript callbacks execute immediately and never enter IR.
- [ ] Go owns IR, codecs, validation, compilation, and digesting.
- [ ] Runtime exports, help, examples, and TypeScript declarations have parity tests.
- [ ] Safe authoring hosts have no ambient store/network/filesystem/process authority.
- [ ] Submission/task/operator modules are explicit, narrow, and host-service gated.
- [ ] V3 does not include a raw-op compatibility shim.

### Operational release

- [ ] `GOWORK=off go test ./...` passes in scraper and affected consumers.
- [ ] Race, scale, restart, privacy, and generated-host suites pass.
- [ ] Migration and operator docs are published.
- [ ] Exact-profile real-provider preflight passes before a full TTC run starts.

## Design decisions

### ADR-1 — Evolve scraper rather than replace it

- **Status:** proposed
- **Context:** scraper already has transactional leases, dependencies, retries, atomic completion, events, metrics, and store abstractions.
- **Options:** rewrite in a new orchestrator; graft RAG behavior onto v2; add a versioned generic v3.
- **Decision:** add workflow v3 around the proven durability nucleus.
- **Rationale:** preserves tested semantics while fixing representation and scheduling at their real boundaries.
- **Consequences:** temporary v2/v3 code coexistence; explicit migration required.

### ADR-2 — Reference-only durable data plane

- **Status:** proposed
- **Decision:** v3 node inputs/outputs are compact registered values and content-verifiable references.
- **Rationale:** prevents the demonstrated multi-gigabyte duplication and privacy breach.
- **Consequences:** domains must implement resolvers/materializers and external artifact stores.

### ADR-3 — Continuous dispatcher with resource classes

- **Status:** proposed
- **Decision:** replace fixed-cycle production scheduling with completion-driven refill and independent resource admission.
- **Rationale:** eliminates the measured `WaitGroup` barrier and models heterogeneous capacity honestly.
- **Consequences:** dispatcher lifecycle and concurrency tests become more complex; transactional leasing remains the correctness anchor.

### ADR-4 — Append-only attempts

- **Status:** proposed
- **Decision:** preserve every attempt separately and derive final node/run state.
- **Rationale:** retries, lease loss, latency, cost, and failure evidence cannot be reconstructed from one overwritten result row.
- **Consequences:** more rows, retention policy needed; rows remain compact/redacted.

### ADR-5 — JavaScript authors; Go normalizes and compiles

- **Status:** proposed
- **Decision:** expose fluent Go-backed builders and pure terminals, but keep canonical IR/compiler code in Go.
- **Rationale:** combines authoring ergonomics with deterministic validation and multi-host reuse.
- **Consequences:** module API needs precise codecs, hidden handles, DTS, and parity tests.

### ADR-6 — Data-only and privileged modules are separate

- **Status:** proposed
- **Decision:** `workflow` is safe authoring; submission, task, and operator authority use separate opt-in modules/services.
- **Rationale:** importing a DSL must not grant a database/store/runtime capability.
- **Consequences:** generated hosts must explicitly select capabilities; some current `ctx` convenience moves to typed task APIs.

### ADR-7 — Lazy expansion and sharded reduction

- **Status:** proposed
- **Decision:** materialize maps in deterministic pages and reduce through bounded immutable manifests.
- **Rationale:** controls graph size, restart cost, fan-in, and failure blast radius.
- **Consequences:** expander/reducer state and projections become first-class engine concepts.

### ADR-8 — No v3 raw-operation compatibility shim

- **Status:** proposed
- **Decision:** reject v2 raw graphs in v3.
- **Rationale:** a shim would preserve arbitrary JSON, global ID assumptions, and absent task/capability schemas—the exact defects v3 removes.
- **Consequences:** migration is deliberate and visible; old runs use v2 read/finish paths.

## Alternatives considered

### Increase `MaxWorkers`

Rejected. More workers enlarge each fixed batch but do not remove the barrier, isolate embedding capacity, or fix payload duplication.

### Poll `RunOnce` more frequently

Rejected. `RunOnce` is blocked inside `WaitGroup.Wait`; another poll in the same scheduler cannot refill it. Concurrent `RunOnce` calls would complicate local limits without fixing the model.

### Keep arbitrary JSON but document reference discipline

Rejected. The TTC incident proves caller discipline is insufficient. The v3 store API must make the unsafe representation difficult or impossible.

### Put source in one workflow-level JSON row

Rejected as the primary design. It reduces duplication but still puts source/prompt material in the workflow database and makes authorization/redaction coarse. Use an authoritative corpus/artifact store plus compact verified references.

### Store all artifacts as SQLite BLOBs

Rejected for large values. It amplifies WAL, backup, checkpoint, and lock pressure. Small allowlisted inline artifacts remain possible.

### Use JavaScript as the durable workflow format

Rejected. Replaying scripts can depend on runtime version, hidden state, clock, random behavior, or changed modules. Persist canonical IR/plan and source/bundle digests instead.

### Expose scraper `Runtime` or `Store` directly to scripts

Rejected. It collapses authoring, execution, and operator authority and makes least privilege impossible.

### Build a RAG-specific scheduler in researchctl

Rejected. It duplicates scraper durability and pollutes researchctl's evidence-ledger role. RAG supplies tasks; scraper executes; researchctl records immutable evidence links.

### Adopt an external workflow system immediately

Not selected. Systems such as Temporal, Argo, or a cloud queue may be valid at another scale, but migration would not automatically solve compact reference contracts, RAG validation, script capabilities, or researchctl identity. Current scraper primitives are sufficient to implement and test the needed semantics locally.

## Risks and mitigations

### Schema and migration complexity

Risk: v2/v3 coexistence can create confusion. Mitigation: distinct packages/tables/schema names and explicit CLI engine selection; no implicit decode.

### SQLite writer contention

Risk: attempts/events/projections add writes. Mitigation: short transactions, indexed eligibility queries, batched/coalesced noncritical event summaries, external artifact bytes, WAL benchmarks, and no provider calls in transactions. If measured limits are exceeded, preserve store contracts and add PostgreSQL later rather than weakening semantics.

### Canonicalization mistakes

Risk: a digest changes across versions/platforms. Mitigation: explicit canonical spec, golden vectors shared by Go and JS-facing tests, compiler version in plan, and never deriving identity from ordinary debug JSON.

### Capability leakage through module selection

Risk: generated host accidentally includes privileged module/default registry entries. Mitigation: allowlisted runtime plans, safe-host negative tests, separate provider modules, and exact host-service type checks.

### Overgeneralized resource model

Risk: arbitrary multi-dimensional claims become hard to schedule. Mitigation: start with one primary resource class plus rate/budget dimensions; add secondary claims only from measured needs and with deadlock-free atomic admission.

### Event growth

Risk: per-attempt events grow indefinitely. Mitigation: compact payloads, append-only attempt facts as authority, event retention/checkpoint policy, projection snapshots, and no per-token/per-chunk-progress spam.

### Provider side effects after lease loss

Risk: a timed-out request may still consume provider budget. Mitigation: bounded calls, idempotency keys where providers support them, conservative budget settlement, attempt evidence, and no blind unlimited retry.

## Open questions

These should be resolved with small experiments, not assumptions:

1. Which canonical JSON implementation and normalization rules should become the public digest contract?
2. Should the first v3 store ship only on SQLite, or should store interfaces be shaped for immediate PostgreSQL parity?
3. What exact inline-value/artifact byte ceilings fit current site records without reopening the arbitrary-payload hole?
4. Which event retention/checkpoint policy balances auditability and database size?
5. Does a map expander use fixed page size, target ready-backlog depth, or both?
6. How should ambiguous provider usage after timeout/lease loss settle monetary budgets?
7. Should task descriptors carry JSON Schema references, protobuf schema identities, or a registry-specific schema digest? The base contract should permit both but one first implementation is needed.
8. Which host owns the RAG-to-workflow adapter package so scraper remains domain-free and researchctl remains scheduler-free?
9. Should TypeScript declarations be entirely descriptor-generated or use generated structured declarations plus reviewed raw DTS for generic types?
10. What measured SQLite size/throughput thresholds should replace the provisional 512 MiB scale bound on CI and the production workstation?

## Intern implementation checklist

Before coding:

- read this document and `reference/02-source-catalogue-and-evidence-map.md`;
- run the existing inventory and grammar probe;
- reproduce the scheduler barrier and synthetic storage amplification;
- write the failing acceptance test for the phase being implemented;
- confirm package direction avoids importing RAG/researchctl into scraper core.

For every change:

- keep provider calls outside transactions;
- use run-scoped composite identities;
- store only typed compact refs;
- validate before persistence;
- record typed redacted failure/attempt facts;
- test restart and stale lease behavior;
- run formatting, unit tests, race tests where relevant, and `GOWORK=off` consumer tests;
- update the diary with exact commands/failures and relate changed files through docmgr.

Do not:

- add a `map[string]any` escape hatch to v3 inputs;
- match errors with string fragments for retry;
- put credentials/config secrets in plan or profile values;
- give a safe Goja module ambient host services;
- serialize callbacks;
- use completion order as output order;
- publish before cardinality, identity, schema, and reopen validation;
- resume or import the diagnostic source-bearing TTC v9 database.

## References

### Ticket artifacts

- `../reference/02-source-catalogue-and-evidence-map.md`
- `../scripts/output/current-scripting-inventory.json`
- `../scripts/output/workflow-dsl-grammar-probe.json`
- `../sources/01-scraper-project-map.md`
- `../sources/02-go-go-goja-project-map.md`
- `../sources/03-widget-dsl-project-map.md`
- `../sources/04-scraper-workflow-api.md`
- `../sources/05-goja-fluent-builder-dsls.md`
- `../sources/06-designing-dsls-with-go-go-goja.md`
- `../sources/07-data-only-vs-host-access-module-split.md`
- `../sources/08-dsl-normalized-config-compiled-plan.md`
- `../sources/09-xgoja-v2-reference.txt`
- `../sources/10-xgoja-provider-runtime-config-and-host-services.txt`

### Scraper code

- `pkg/engine/model/types.go`
- `pkg/engine/scheduler/scheduler.go`
- `pkg/engine/store/store.go`
- `pkg/engine/store/sqlite/migrations/001_engine_core.sql`
- `pkg/engine/store/sqlite/migrations/002_engine_runtime.sql`
- `pkg/engine/store/sqlite/lease_store.go`
- `pkg/engine/store/sqlite/result_store.go`
- `pkg/workflow/package.go`
- `pkg/workflow/context.go`
- `pkg/workflow/runtime.go`
- `pkg/js/runtime/executor.go`
- `pkg/sites/submitverbs/runtime.go`
- `pkg/runtimeevents/`
- `pkg/metrics/`

### Local comparison code

- `researchctl/pkg/gojamodules/researchctl/`
- `rag-evaluation-system/pkg/gojamodules/rag/`
- `rag-evaluation-system/pkg/xgoja/providers/rag/provider.go`
- `rag-evaluation-system/pkg/widgetdsl/v3_descriptors.go`
- `go-go-goja/pkg/xgoja/providerapi/`
- `go-go-goja/pkg/xgoja/app/`
