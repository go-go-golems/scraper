---
Title: Durable External Operation Evidence Ledger Design and Implementation Guide
Ticket: SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS
Status: active
Topics:
    - workflow-v3
    - durability
    - observability
    - privacy
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system/cmd/rag-ttc-v3-sweep/main.go
      Note: Current successful-cell and failed-cell custody behavior
    - Path: abs:///home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system/internal/workflowv3ttc/provider.go
      Note: Current provider-wall timer and failure measurement loss
    - Path: repo://pkg/workflowv3runtime/engine.go
      Note: Parent lease owner and scoped recorder injection point
    - Path: repo://pkg/workflowv3runtime/modules.go
      Note: Trusted host module descriptor and recorder seam
    - Path: repo://pkg/workflowv3sqlite/schema.sql
      Note: Current attempts and budget schema plus proposed additive operation tables
    - Path: repo://pkg/workflowv3sqlite/store.go
      Note: Lease admission completion fencing migration and transaction reference
    - Path: ws://researchctl/pkg/lab/runtime.go
      Note: Researchctl downstream immutable observation boundary
ExternalSources: []
Summary: Implementation-ready design for recording external calls durably before and after execution, preserving precise safe measurements across task failure, cancellation, retry, and restart.
LastUpdated: 2026-07-22T19:55:00-04:00
WhatFor: Guide implementation of a generic Workflow V3 external-operation ledger and its RAG and researchctl integration without leaking payloads or weakening lease and budget authority.
WhenToUse: Read before changing Workflow V3 persistence, task-module APIs, provider instrumentation, operation evidence exports, or researchctl custody for network, model, browser, database, or subprocess calls.
---


# Durable External Operation Evidence Ledger Design and Implementation Guide

## Executive summary

Workflow V3 durably records runs, nodes, attempts, leases, budget reservations, output references, and bounded operational events. It does not yet durably record the smaller external calls made *inside* an attempt. An attempt that calls an LLM, embedding service, HTTP origin, browser, database, or fixed tool can measure that call in memory and include the measurement in a successful output. If the call or task fails, the output is never published and the most useful data-plane evidence—provider start time, provider wall duration, outcome, usage, and safe failure code—disappears.

This design adds a generic **external-operation evidence ledger** to scraper's Workflow V3 packages. The ledger records an immutable admission row before an external effect begins and an immutable completion row after the effect returns. It is attached to the active Workflow attempt, but it has its own completion ticket so a call that began under a valid lease can still record its outcome after cancellation or lease expiry. The ledger stores only bounded identities, timestamps, integer counters, digests, and closed outcome codes. It cannot store URLs, headers, credentials, prompts, source text, request or response bodies, vectors, arbitrary metadata, or error messages.

The ownership boundary is deliberate:

- **scraper / Workflow V3** owns generic operation identity, admission, fencing, persistence, queries, projections, and canonical export;
- **RAG evaluation** declares generation and embedding operation descriptors, maps provider outcomes to closed codes, and derives latency, concurrency, overlap, token, cost, and throughput statistics;
- **researchctl** imports the canonical operation export as a verified immutable artifact and stores reviewed derived metrics; it is not a live recorder or a second scheduler.

The first implementation should support trusted in-process Go host modules. It must not expose a general-purpose recorder to JavaScript and must not grant restricted child processes database access. Parent-mediated isolated-operation streaming is a future protocol extension, not part of the initial change.

The intended result is a reusable building block: failed or canceled workflows remain scientifically useful, every admitted external call is auditable, provider-wall timing remains distinct from attempt timing, and privacy is enforced structurally rather than by convention.

## 1. Audience and reading path

This guide is written for an engineer who is new to scraper, Workflow V3, RAG evaluation, and researchctl. Read it in this order:

1. Read the glossary and system map below.
2. Inspect the current persistence schema and lease transaction.
3. Inspect how the runtime constructs a lease-scoped task and its host modules.
4. Inspect the current RAG batch provider to see why failed timing is lost.
5. Review the proposed contracts and SQLite schema.
6. Follow the implementation phases in order; do not begin with the RAG adapter.
7. Run the failure, privacy, migration, race, and restart tests before any real-provider validation.

The critical conceptual distinction is:

> A Workflow attempt is control-plane execution evidence. An external operation is data-plane effect evidence. They are related, but they are not interchangeable.

## 2. Glossary

### Attempt

One leased execution of one Workflow node. Attempts are append-only historical records. A retry creates another attempt; it does not overwrite the previous attempt.

### External operation

One bounded effect initiated by trusted host code while an attempt is active. Examples include one provider generation request, one embedding request, one HTTP round trip, one browser command, or one allowlisted tool invocation.

An operation is narrower than an attempt. One attempt may contain zero, one, or several operations if its declared descriptor and budget allow that cardinality.

### Admission

The durable pre-call fact that Workflow authorized an operation to begin. Admission must commit before the external request is submitted. If the process crashes after admission, the operation remains conservatively observable as admitted with unknown completion.

### Completion

The durable post-call fact that the external operation returned, failed, or was canceled. Completion includes precise provider-wall timing captured by trusted Go code, a closed outcome, safe failure class/code, accounting mode, and bounded integer counters.

### Operation descriptor

A host-owned immutable policy describing an allowed operation kind and version, its authority identity, counter schema, reservation dimensions, and per-attempt cardinality. Its canonical digest is stored with each admission.

### Operation ticket

An opaque capability returned by admission. The ticket authorizes exactly one completion for that operation. It is not exposed to JavaScript or persisted in clear text.

### Provider-wall time

Time spent in the provider call itself, measured with Go's monotonic clock. It excludes Workflow queueing, input materialization, Goja runtime creation, output publication, and SQLite checkpointing.

### Attempt-wall time

The interval from Workflow attempt start to attempt finish. It includes control-plane overhead and must not be used as a substitute for provider latency.

### Custody

The process of converting mutable runtime evidence into canonical, digest-addressed, immutable evidence suitable for researchctl import and publication.

## 3. System boundaries

### 3.1 Component map

```text
JavaScript workflow author
        |
        | composes typed task nodes; cannot open arbitrary operations
        v
Workflow V3 compiler and exact bundle registry
        |
        | pins task/bundle/module/resource/retry/budget/isolation identity
        v
Workflow V3 SQLite store <------------------------------+
  runs, nodes, leases, attempts, budgets, gates          |
  NEW: operation admissions + completions + counters     |
        ^                                                 |
        | lease-scoped recorder                           |
Workflow V3 runtime                                      |
        | constructs fresh Goja runtime                  |
        | injects recorder only into trusted Go module   |
        v                                                 |
RAG / HTTP / browser / fixed-tool host module            |
        | Begin -> commit -> external call -> Finish -----+
        |
        v
Canonical operation JSONL + manifest
        |
        | digest/size verification
        v
researchctl immutable laboratory artifact
        |
        v
reviewed metrics, reports, graphs, and conclusions
```

### 3.2 What scraper owns

Scraper is domain-neutral. It may know that an operation has a kind, descriptor digest, start time, elapsed microseconds, outcome, and integer counters. It must not know what a prompt, embedding, browser page, or scientific hypothesis means.

Scraper owns:

- validation of operation and counter names;
- operation admission under an active lease fence;
- operation completion under an operation-ticket fence;
- append-only SQLite persistence;
- safe bounded projections and events;
- deterministic query ordering and canonical export;
- budget-reservation linkage;
- restart and cancellation semantics;
- generic privacy constraints.

### 3.3 What RAG evaluation owns

RAG evaluation owns provider semantics:

- `provider.generate` and `provider.embed` descriptors;
- provider/profile/model digests;
- safe error classification;
- definitions of `input_tokens`, `output_tokens`, `embedding_tokens`, `cost_microunits`, `requests`, `chunk_count`, and `representation_count`;
- aggregation into latency distributions, concurrency timelines, overlap, throughput, retry rates, and cost efficiency;
- source reconstruction and evidence publication policy.

RAG must never ask scraper to persist prompts, chunks, generated records, provider bodies, vectors, or credentials.

### 3.4 What researchctl owns

Researchctl starts at the custody boundary. Its existing laboratory contracts already accept events, verified artifacts, metrics, traces, terminal summaries, and canonical run exports. It verifies artifact digest/size and imports complete run exports transactionally.

Researchctl owns:

- immutable run and attempt identity;
- verified artifact custody;
- import idempotency and conflict detection;
- scientific metric and trace records;
- reviewed interpretation and reporting.

Researchctl must not become a synchronous dependency of Workflow task execution. A provider call must not block on a researchctl daemon or mutate a research project graph.

## 4. Current-state architecture

### 4.1 SQLite durability and migration

`pkg/workflowv3sqlite/store.go:28-57` opens SQLite in WAL mode, enables foreign keys, configures immediate write transactions, executes the embedded schema, applies additive column migrations, and reconciles budget invariants. New tables added with `CREATE TABLE IF NOT EXISTS` are therefore additive for existing databases.

The current DSN does not explicitly request `synchronous=FULL`. Operation admission is a spend/effect boundary: returning from `BeginExternalOperation` must mean that SQLite has durably committed the admission before the caller submits the request. The implementation must explicitly validate SQLite's synchronous behavior and should use `FULL` unless a fault-injection test proves an equally durable alternative.

### 4.2 Lease, attempt, and budget admission are already transactional

`pkg/workflowv3sqlite/store.go:410-476` performs the core lease transaction:

- selects a ready node;
- allocates an attempt number and random lease token;
- marks the node running;
- inserts the running attempt;
- reserves the node budget;
- increments resource dispatch evidence;
- writes a bounded `attempt.started` event;
- commits before returning the lease.

This is the model for operation admission. The new operation must not be represented only as an event because events are notification records, not the source of truth. It needs normalized tables and invariants.

### 4.3 Completion is fenced by the active lease

`pkg/workflowv3sqlite/store.go:567-662` checks the lease fence, settles budget usage, publishes output references, marks the attempt and node successful, updates map/reduction/run projections, writes an event, and commits in one transaction.

`pkg/workflowv3sqlite/store.go:1113-1131` defines the current fence: the run must still be running, the node token must match, and the cancellation epochs must match. This fence is correct for task output publication. It is *too strict* for operation completion: a provider call may return after cancellation, and its safe outcome is still valuable evidence even though its task output must be rejected.

The design therefore uses two fences:

- **admission fence:** the current Workflow lease must be valid;
- **completion fence:** the opaque operation ticket must match an admitted, not-yet-completed operation.

Operation completion never grants node completion authority.

### 4.4 Attempts and budgets do not represent nested effects

The existing schema stores budget accounts/reservations and attempt timing (`pkg/workflowv3sqlite/schema.sql:231-303`). `workflowv3.Attempt` exposes attempt start/finish and failure data, while `BudgetProgress` exposes aggregate limits, used, reserved, and remaining units (`pkg/workflowv3/types.go:282-319` and `352-362`).

These records answer control-plane questions:

- Was a task leased?
- Did it retry?
- How long did the task occupy a Workflow resource slot?
- What budget was reserved and settled?

They do not answer data-plane questions:

- Was an external request actually admitted?
- When did provider execution begin?
- How long did the provider call take?
- Did a failed attempt include a completed provider call?
- Which usage counters were reported for that call?
- How many provider calls overlapped?

### 4.5 The runtime has the right injection seam

`pkg/workflowv3runtime/engine.go:272-320` owns the lease and constructs `TaskRequest`. It watches the lease and cancels the task context when the lease becomes invalid. This is the correct place to create a lease-scoped operation recorder.

`pkg/workflowv3runtime/task_runner.go:21-29` currently carries run/node/attempt identity, task, inputs, artifacts, and module registry. `RunTask` builds requested host modules with a `TaskModuleContext` (`task_runner.go:69-107`).

`pkg/workflowv3runtime/modules.go:22-63` defines that host-module context and immutable alias registry. This is the narrowest safe injection point. Add the recorder to `TaskModuleContext`; do not add `beginOperation()` to the public JavaScript `workflow/task` object at `task_runner.go:214-252`.

The host module is trusted Go authority. JavaScript expresses task intent but cannot invent operation descriptors, counters, authority digests, or cardinality.

### 4.6 Restricted children must remain database-free

The isolated-worker protocol currently carries one canonical request and one canonical response. The child receives immutable bundle/input/tool authority and returns outputs, usage, or a typed failure. It does not receive the parent SQLite path or lease token.

This boundary must remain intact. A restricted child cannot write the operation ledger directly. A terminal response containing child-collected observations is also insufficient because child death would lose the observations. Initial implementation therefore supports parent-hosted external operations only. A future isolated operation mechanism must use a parent-mediated streaming/broker protocol.

### 4.7 The RAG failure demonstrates the gap

`internal/workflowv3ttc/provider.go:80-134` admits a generation request, captures `time.Now()`, invokes the provider, and computes monotonic elapsed microseconds. On success it places the measurement inside `GeneratedBatch`. On provider or malformed-output failure it returns an error before publishing that output.

The sweep reads measurements only from the successful `measured` output manifest (`cmd/rag-ttc-v3-sweep/main.go:377-433`). Failed-cell custody currently contains attempt counts, failure-code counts, and budgets, but no provider spans (`main.go:436-455`).

This means the timer itself is correct, yet failure destroys the data. The missing abstraction is a durable side channel associated with the attempt but independent of task outputs.

### 4.8 Researchctl already has the custody primitives

`pkg/lab/runtime.go:70-83` defines an `ObservationSink` for events, artifacts, metrics, traces, and attempt completion. `pkg/lab/types.go:91-157` includes verified artifacts and canonical attempt/run export records. `internal/labsqlite/import.go:13-58` validates the export digest and artifact checks before planning import, and lines 125-258 import all records transactionally with conflict checks.

No researchctl schema extension is required for the first integration. The operation JSONL and manifest can be verified artifacts; selected aggregate values can be ordinary metrics. `pkg/lab/artifacts.go:18-84` already verifies every referenced artifact by URI, digest, and size, and `RunExportDigest` canonicalizes the complete export.

## 5. Problem statement

### 5.1 Functional problem

Workflow tasks can make external calls whose result determines success or failure. Successful output artifacts are not an adequate measurement channel because output publication is conditional on task success. The system therefore loses precise latency and usage evidence on exactly the cases where diagnosis is most important: timeout, malformed output, transport failure, cancellation, retry, provider death, and budget exhaustion.

### 5.2 Durability problem

A matrix or multi-cell experiment may delete source-bearing transient databases after reading successful outputs. If aggregate evidence exists only in process memory until the entire matrix completes, a later cell failure can lose earlier measurements. Per-cell checkpointing fixes successful cells, but it does not recover failed provider spans unless those spans were persisted independently at call completion.

### 5.3 Authority problem

An external request is an effect and potentially a spend event. The system needs a durable pre-call record. Recording only after the call leaves a crash window in which the provider may have received a request but the local system has no evidence that it was admitted.

### 5.4 Fencing problem

Task output must obey the live lease fence. Measurement completion has different semantics. If cancellation occurs while a request is active, the eventual cancellation/provider outcome belongs in evidence even though node completion is stale. Reusing the node completion fence would discard that evidence.

### 5.5 Privacy problem

A generic metadata map would become a path for prompts, URLs, provider bodies, source text, and credentials to enter SQLite and research reports. The ledger needs a deliberately impoverished schema and descriptor-controlled integer counters.

### 5.6 Scope

In scope:

- generic operation contracts and validation;
- additive Workflow V3 SQLite tables;
- pre-call admission and post-call completion APIs;
- lease and operation-ticket fencing;
- descriptor-controlled integer counters;
- budget-reservation linkage and reconciliation;
- bounded events and operational projections;
- canonical partial and terminal export;
- trusted host-module injection;
- RAG generation/embedding integration;
- researchctl artifact/metric mapping;
- migration, restart, race, failure, and privacy tests.

Out of scope for the first implementation:

- distributed tracing protocols such as OpenTelemetry export;
- arbitrary tags, logs, stack traces, URLs, payload capture, or error strings;
- unrestricted JavaScript operation creation;
- child access to the parent database;
- a researchctl runtime service;
- cross-database atomicity for cumulative multi-run authority;
- automatic interpretation of measurements as scientific conclusions.

## 6. Required invariants

The implementation is acceptable only if all of these invariants hold.

### 6.1 Admission invariants

1. `BeginExternalOperation` validates the active run/node/attempt lease fence.
2. Admission is committed before the method returns.
3. The caller submits no external request until admission returns successfully.
4. Admission consumes only descriptor-declared reservation units.
5. Aggregate operation admissions cannot exceed the attempt's reservation.
6. A stale or canceled lease cannot admit a new operation.
7. Each admission receives a store-generated operation ID and ordinal.

### 6.2 Completion invariants

1. Completion requires an opaque operation ticket.
2. Completion may succeed after run cancellation or lease expiry if admission happened under a valid lease.
3. Completion cannot publish task outputs or mutate node status.
4. Exactly one completion exists per operation.
5. Repeating byte-equivalent completion is idempotent.
6. A conflicting second completion fails closed.
7. Precise elapsed time is nonnegative and captured using a monotonic clock.
8. Unknown completion after process death remains explicitly unknown; it is never represented as zero latency or success.

### 6.3 Privacy invariants

The ledger never persists:

- source records, chunks, documents, or prompts;
- request or response bodies;
- generated text or provider error bodies;
- vectors or database results;
- credentials, authorization/cookie headers, environment values, or secret URLs;
- arbitrary host paths;
- arbitrary SQL or configuration;
- free-form labels, metadata, or error messages.

Allowed values are:

- bounded identifiers matching closed regular expressions;
- SHA-256 digests;
- RFC3339Nano UTC timestamps;
- nonnegative checked integers;
- closed outcome/accounting/failure codes;
- run/node/attempt identity already owned by Workflow.

### 6.4 Compatibility invariants

- Existing Workflow V3 databases open through additive migration.
- Existing v2 and V3 workflows behave identically when no operation descriptors are configured.
- Task bundles and JavaScript APIs do not gain ambient authority.
- Restricted worker protocol remains compatible in Phase 1.
- Existing attempt and budget semantics remain authoritative.

## 7. Proposed domain model

### 7.1 Public types in `pkg/workflowv3`

Create `pkg/workflowv3/external_operation.go`.

```go
package workflowv3

type ExternalOperationKind struct {
    Name    string `json:"name"`    // e.g. provider.generate
    Version string `json:"version"` // e.g. v1
}

type OperationCounterDescriptor struct {
    Name string `json:"name"`
    Unit string `json:"unit"`
    Role string `json:"role"` // reservation | usage | measure
}

type ExternalOperationDescriptor struct {
    Kind            ExternalOperationKind       `json:"kind"`
    AuthorityDigest string                      `json:"authorityDigest"`
    Counters        []OperationCounterDescriptor `json:"counters"`
    MaxPerAttempt   int                         `json:"maxPerAttempt"`
    Digest          string                      `json:"digest"`
}

type ExternalOperationSpec struct {
    DescriptorDigest string         `json:"descriptorDigest"`
    CorrelationDigest string        `json:"correlationDigest,omitempty"`
    Reservation      []BudgetAmount `json:"reservation,omitempty"`
    Measures         []OperationCounter `json:"measures,omitempty"`
}

type OperationCounter struct {
    Name  string `json:"name"`
    Units int64  `json:"units"`
}

type ExternalOperationTicket struct {
    OperationID    string
    CompletionKey string // secret capability; never serialize or log
}

type ExternalOperationCompletion struct {
    ProviderStartedAt time.Time          `json:"providerStartedAt"`
    ElapsedMicros     int64              `json:"elapsedMicros"`
    Outcome           string             `json:"outcome"`
    FailureClass      string             `json:"failureClass,omitempty"`
    FailureCode       string             `json:"failureCode,omitempty"`
    AccountingMode    string             `json:"accountingMode"`
    Counters          []OperationCounter `json:"counters,omitempty"`
}

type ExternalOperationRecorder interface {
    Begin(context.Context, ExternalOperationSpec) (ExternalOperationTicket, error)
    Finish(context.Context, ExternalOperationTicket, ExternalOperationCompletion) error
}
```

The exact names may change during implementation, but these concepts must remain separate. Do not combine admission and completion into one `Record()` method.

### 7.2 Closed values

Recommended outcomes:

- `succeeded`
- `failed`
- `canceled`
- `timed-out`
- `unknown`

Recommended accounting modes:

- `actual`: all declared usage is known;
- `conservative`: charge the operation reservation because usage is incomplete;
- `none`: descriptor declares no budget usage.

Failure class and code use the same bounded validation style as Workflow failures, but completion does not store `Failure.Message`.

### 7.3 Descriptor identity

A descriptor digest is computed over the canonical descriptor with `Digest` cleared. Sort counters by name before digesting. Reject duplicate counters, unsupported roles, invalid units, unknown digest algorithms, zero/negative cardinality, and more than a small fixed counter count such as 32.

Descriptors are host configuration, not dynamic JavaScript data. A task module factory declares the descriptors it can use. The runtime creates a recorder scoped to the intersection of:

- descriptors declared by requested module aliases;
- current run/node/attempt identity;
- active lease and budget reservation.

### 7.4 No arbitrary metadata

Do not add `map[string]any`, `json.RawMessage`, `attributes`, `labels`, or `message` fields. If a new scalar is scientifically necessary, add a reviewed counter descriptor with a unit and privacy analysis.

`chunk_count`, `input_runes`, and `representation_count` are integer measures. They reveal magnitude but not content. Their use must be descriptor-declared and covered by privacy tests.

## 8. Proposed persistence schema

Use separate immutable admission and completion tables. This avoids mutating an admission from `running` to terminal and makes the crash boundary explicit.

```sql
CREATE TABLE IF NOT EXISTS v3_external_operations (
  operation_id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  attempt_no INTEGER NOT NULL,
  ordinal INTEGER NOT NULL CHECK (ordinal > 0),
  kind TEXT NOT NULL,
  kind_version TEXT NOT NULL,
  descriptor_digest TEXT NOT NULL,
  authority_digest TEXT NOT NULL,
  correlation_digest TEXT,
  completion_key_digest TEXT NOT NULL,
  admitted_at TEXT NOT NULL,
  UNIQUE (run_id, node_key, attempt_no, ordinal),
  FOREIGN KEY (run_id, node_key, attempt_no)
    REFERENCES v3_attempts(run_id, node_key, attempt_no) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_external_operation_allocations (
  operation_id TEXT NOT NULL,
  dimension TEXT NOT NULL,
  units INTEGER NOT NULL CHECK (units > 0),
  PRIMARY KEY (operation_id, dimension),
  FOREIGN KEY (operation_id)
    REFERENCES v3_external_operations(operation_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_external_operation_measures (
  operation_id TEXT NOT NULL,
  name TEXT NOT NULL,
  units INTEGER NOT NULL CHECK (units >= 0),
  PRIMARY KEY (operation_id, name),
  FOREIGN KEY (operation_id)
    REFERENCES v3_external_operations(operation_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_external_operation_completions (
  operation_id TEXT PRIMARY KEY,
  provider_started_at TEXT NOT NULL,
  elapsed_micros INTEGER NOT NULL CHECK (elapsed_micros >= 0),
  outcome TEXT NOT NULL CHECK (outcome IN
    ('succeeded','failed','canceled','timed-out','unknown')),
  failure_class TEXT,
  failure_code TEXT,
  accounting_mode TEXT NOT NULL CHECK (accounting_mode IN
    ('actual','conservative','none')),
  completed_at TEXT NOT NULL,
  completion_digest TEXT NOT NULL,
  FOREIGN KEY (operation_id)
    REFERENCES v3_external_operations(operation_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_external_operation_counters (
  operation_id TEXT NOT NULL,
  name TEXT NOT NULL,
  units INTEGER NOT NULL CHECK (units >= 0),
  PRIMARY KEY (operation_id, name),
  FOREIGN KEY (operation_id)
    REFERENCES v3_external_operation_completions(operation_id)
      ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_v3_external_operations_attempt
  ON v3_external_operations(run_id, node_key, attempt_no, ordinal);

CREATE INDEX IF NOT EXISTS idx_v3_external_operations_kind
  ON v3_external_operations(run_id, kind, admitted_at, operation_id);
```

### 8.1 Why admissions and completions are separate

A single mutable row is simpler but hides the distinction between pre-call authority and post-call evidence. Separate tables make these states representable without inference:

- admission exists, completion absent: call may be running or completion is unknown;
- admission and completion exist: observed terminal outcome;
- no admission: no authorized call should have been submitted.

The completion table's primary key enforces exactly-once terminal evidence.

### 8.2 Completion capability storage

Generate at least 256 random bits for `CompletionKey`. Return the raw key only in the in-memory ticket. Persist `sha256(CompletionKey)` in the admission row. `Finish` hashes the presented key and compares it in constant time.

Do not reuse the lease token as the completion key. Lease tokens authorize Workflow state transitions; operation completion has intentionally different late-arrival semantics.

### 8.3 Durability mode

Add `_synchronous=FULL` to the Workflow SQLite DSN or an equivalent verified configuration. Add a startup assertion:

```sql
PRAGMA journal_mode;  -- must be wal
PRAGMA synchronous;   -- must be FULL for authority-bearing stores
PRAGMA foreign_keys;  -- must be 1
```

Benchmark admission overhead with fixture calls, but do not weaken durability to improve benchmark numbers. If global `FULL` is unacceptable, design a separate authority-bearing SQLite connection/store and prove ordering under crash injection; do not silently rely on WAL `NORMAL`.

## 9. Transaction and fencing semantics

### 9.1 Begin transaction

`Store.BeginExternalOperation` receives the live `workflowv3.Lease`, a validated descriptor, a spec, and the current time.

```text
BEGIN IMMEDIATE
  1. checkFence(run, node, lease token, cancel epoch)
  2. verify referenced attempt is running
  3. validate descriptor digest and requested counters
  4. count prior admissions for descriptor in this attempt
  5. reject if count == MaxPerAttempt
  6. validate reservation allocations against attempt reservations
  7. ensure sum(previous allocations + new allocation) <= reserved units
  8. allocate operation ID, ordinal, and completion-key digest
  9. insert admission, allocation, and measure rows
 10. insert bounded external_operation.admitted wake-up event
COMMIT DURABLY
return opaque ticket
```

The external call occurs only after step 10 returns.

### 9.2 Provider wrapper

Trusted host code should use a helper that makes omission difficult:

```go
ticket, err := recorder.Begin(ctx, spec)
if err != nil {
    return admissionFailure(err)
}

started := time.Now() // includes a monotonic component
result, callErr := provider.Call(ctx, request)
elapsed := time.Since(started)

completion := classifyCompletion(started, elapsed, result, callErr)
finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
defer cancel()
if err := recorder.Finish(finishCtx, ticket, completion); err != nil {
    // Measurement durability is part of task correctness. Do not report
    // provider success if its completion evidence could not be committed.
    return evidencePersistenceFailure(err)
}

return resultOrTypedFailure(result, callErr)
```

`context.WithoutCancel` is intentional for the bounded persistence step. Cancellation should stop provider work, not erase the resulting cancellation evidence. The finish timeout prevents an unbounded shutdown stall.

### 9.3 Finish transaction

```text
BEGIN IMMEDIATE
  1. load admission by operation ID
  2. constant-time compare completion-key digest
  3. validate completion against descriptor
  4. canonicalize completion and calculate completion digest
  5. if identical completion exists: return success (idempotent)
  6. if different completion exists: fail closed
  7. insert immutable completion row
  8. insert sorted counter rows
  9. insert bounded external_operation.completed wake-up event
COMMIT
```

Do **not** call the current lease `checkFence()` in `Finish`. The ticket proves this operation was admitted while the lease was valid. `Finish` cannot update node outputs, attempt status, budgets directly, or admit another call.

### 9.4 Crash states

| Crash point | Durable evidence | Required interpretation |
|---|---|---|
| Before `Begin` commit | No operation | No call may have been submitted |
| After `Begin` commit, before provider call | Admitted, incomplete | Conservatively authorized/unknown |
| During provider call | Admitted, incomplete | Possibly active or unknown after owner loss |
| After provider return, before `Finish` | Admitted, incomplete | Outcome lost; charge conservatively |
| After `Finish` commit | Complete operation | Exact safe span/outcome/counters available |
| After operation completion, before task terminal commit | Complete operation + running attempt | Retry/recovery may occur; operation remains evidence |

A maintenance projection may label an incomplete operation `orphaned` after its attempt is terminal or its lease is lost. Do not fabricate a completion timestamp or elapsed duration. Export it as `completion: null` with an explicit derived state.

## 10. Budget integration

### 10.1 Why observation and accounting must meet

The ledger must not become a second, disagreeing budget system. Existing `v3_budget_reservations` remains authoritative for run limits. Operation allocations explain which external calls consumed portions of an attempt reservation.

### 10.2 Allocation rules

An operation descriptor may bind counters to the attempt's budget dimensions. For example:

```text
provider.generate/v1
  reservation:
    requests          1
    input_tokens      16384
    output_tokens     8192
    cost_microunits   10650
```

At admission, the store verifies:

```text
sum(all operation allocations for attempt/dimension)
    + requested operation allocation
    <= attempt reservation for dimension
```

This guarantees that a task cannot submit more calls than its Workflow reservation authorizes.

### 10.3 Settlement rules

For each admitted operation:

- `actual`: use completed counters, each no greater than its allocation;
- `conservative`: use the full allocation;
- no completion: use the full allocation when the attempt settles;
- `none`: no budget-bound counters are allowed.

The effective usage for operation-bound dimensions is derived by SQLite. Task-reported usage must not silently overwrite it.

Recommended transition:

1. Add operation allocations and query-only reconciliation first.
2. Add a test-only assertion comparing current task-reported usage with derived operation usage.
3. Once all relevant host modules report consistently, make mismatch terminal with `BUDGET_OPERATION_USAGE_MISMATCH`.
4. Then derive bound dimensions directly during `CompleteWithUsage`, `FailWithUsage`, and conservative failure settlement.

This staged implementation avoids changing budget settlement and instrumentation in one unreviewable patch.

### 10.4 Cross-run cumulative authority

A Workflow store controls one database's run budgets. The TTC sweep also has cumulative authority across independently deleted per-cell databases. Cross-database atomicity is not solved by this ledger.

The safe ordering is:

```text
1. Persist cumulative authority admission.
2. Begin Workflow external operation and persist its authority-admission digest.
3. Submit provider request.
```

A crash between steps 1 and 2 overcounts authority, which is conservative. Reversing the order could undercount spend and is forbidden. A future generic `AuthorityLedger` may replace the sweep file, but it is a separate component and decision.

## 11. Runtime and module API wiring

### 11.1 Engine construction

Add a store method that returns a lease-scoped recorder:

```go
recorder := e.Store.ExternalOperations(lease, e.Modules.OperationDescriptors(lease.PlanNode.Modules))

taskRequest := TaskRequest{
    RunID: lease.RunID,
    NodeKey: lease.NodeKey,
    Attempt: lease.Attempt,
    Task: registered,
    Inputs: inputs,
    Artifacts: e.Artifacts,
    Modules: e.Modules,
    ExternalOperations: recorder,
}
```

The recorder retains the lease only as an admission fence. It does not expose lease tokens through public fields or logs.

### 11.2 Task module descriptors

Extend `TaskModuleFactory`:

```go
type TaskModuleFactory struct {
    Alias      string
    Validate   func() error
    Operations []workflowv3.ExternalOperationDescriptor
    Build      func(TaskModuleContext) (RuntimeModuleRegistrar, error)
}
```

Registry construction validates and clones descriptors. Duplicate `(kind, version)` declarations with different digests are rejected. `TaskModuleContext` gains:

```go
type TaskModuleContext struct {
    Context            context.Context
    Request            TaskRequest
    Workspace          string
    ExternalOperations workflowv3.ExternalOperationRecorder
}
```

### 11.3 JavaScript authority remains unchanged

Do not expose the recorder on `workflow/task`, ordinary `require()`, or a new public module. The JavaScript task calls a domain operation such as `rag.generateBatch()`. Trusted Go code inside that exact registered module performs admission and completion.

This preserves the existing principle:

> JavaScript expresses intent; Go owns effects, authority, validation, and persistence.

### 11.4 RAG provider adaptation

The current provider object is shared by module factories, while recorders are attempt-scoped. Avoid storing the recorder in a shared provider field. Use one of these equivalent shapes:

```go
type InvocationContext struct {
    Context    context.Context
    Operations workflowv3.ExternalOperationRecorder
}

type BatchProvider interface {
    GenerateBatch(InvocationContext, ChunkBatch) (Result[GeneratedBatch], error)
    EmbedBatch(InvocationContext, GeneratedBatch) (Result[MeasuredBatch], error)
}
```

or:

```go
type OperationAwareProvider interface {
    WithExternalOperations(workflowv3.ExternalOperationRecorder) Provider
}
```

Prefer the explicit `InvocationContext`: it prevents accidental recorder retention and makes tests straightforward.

The provider wrapper begins the operation immediately before the provider invocation, captures monotonic elapsed time, finishes the operation on every return path, and then returns success or a typed RAG failure. Measurements no longer need to travel inside source-bearing `GeneratedBatch` merely to survive; they may remain there temporarily for compatibility but the ledger becomes authoritative.

## 12. Query, projection, and export APIs

### 12.1 Query model

Add a joined public view:

```go
type ExternalOperation struct {
    OperationID       string
    RunID             RunID
    NodeKey           NodeKey
    Attempt           int
    Ordinal           int
    Kind              ExternalOperationKind
    DescriptorDigest  string
    AuthorityDigest   string
    CorrelationDigest string
    AdmittedAt        time.Time
    Completion        *ExternalOperationCompletion
    Reservation       []BudgetAmount
    Measures          []OperationCounter
}
```

Query API:

```go
func (s *Store) ExternalOperations(
    ctx context.Context,
    runID RunID,
    afterOrdinal int64,
    limit int,
) ([]workflowv3.ExternalOperation, error)
```

Limits must be bounded, for example `1 <= limit <= 1000`. Order by run, node, attempt, ordinal, and operation ID. Return empty slices rather than `null` in canonical exports.

### 12.2 Operational projection

Extend `OperationalSnapshot` with bounded aggregate fields, not complete operation rows:

```go
type ExternalOperationProgress struct {
    ActiveByKind   map[string]int `json:"activeByKind"`
    Outcomes       map[string]int `json:"outcomes"`
    Orphaned       int            `json:"orphaned"`
    OldestActiveMS int64          `json:"oldestActiveMs"`
}
```

Events such as `external_operation.admitted` and `external_operation.completed` wake consumers. As with existing operational snapshots, normalized tables remain truth.

### 12.3 Canonical export

Add a versioned export schema:

```text
scraper-workflow-v3-external-operations/v1
```

Export files:

```text
external-operations.jsonl
external-operations-manifest.json
```

Each JSONL line is canonical JSON containing one admission and optional completion. The manifest contains:

- schema version;
- run ID and plan digest;
- as-of event sequence;
- row count;
- complete/incomplete counts;
- JSONL SHA-256 and size;
- descriptor digests;
- privacy classification identifier.

The exporter must:

1. open one coherent read transaction;
2. query all rows in deterministic order;
3. encode canonical JSON one record per line;
4. hash while writing a temporary file;
5. `fsync` the file;
6. atomically rename it;
7. `fsync` the containing directory;
8. write the manifest by the same process.

Export may occur for succeeded, failed, canceled, or running runs. Partial exports explicitly include incomplete admissions. A terminal export does not imply every operation has a completion.

### 12.4 Derived performance calculations

RAG derives metrics from completed operation spans:

```text
provider_end = provider_started_at + elapsed_micros
latency distribution = completed operation elapsed values
peak concurrency = sweep line over provider intervals
provider overlap = intersection of generation and embedding intervals
request throughput = completed/admitted requests per chosen wall interval
retry rate = attempts or calls beyond first logical operation
cost efficiency = useful chunks or representations / cost microunits
```

Rules:

- process end events before start events at identical timestamps;
- never treat missing completion, usage, token, or cost data as zero;
- report denominator and missing-data count with every rate;
- keep attempt-wall and provider-wall statistics in separate fields and charts;
- mark incomplete calls as censored/unknown rather than inventing latency.

## 13. Researchctl integration

### 13.1 First integration: verified artifacts, no schema migration

A Workflow-to-researchctl adapter should register:

```text
role: operation-evidence
kind: measurement
schemaVersion: scraper-workflow-v3-external-operations/v1
mediaType: application/x-ndjson
uri: artifacts/.../external-operations.jsonl
digest: sha256:...
sizeBytes: ...
```

The manifest is a second verified artifact or the JSONL artifact metadata references its digest. Prefer two explicit artifacts when both must survive independently.

Researchctl already verifies artifact path, size, and digest and imports them transactionally. This makes the operation ledger self-contained without copying it into mutable research graph YAML.

### 13.2 Metrics

Record selected scalar projections as researchctl metrics:

- `provider_latency_p50_us`
- `provider_latency_p95_us`
- `provider_peak_concurrency`
- `provider_request_count`
- `provider_incomplete_count`
- `provider_retry_rate`
- `provider_cost_microunits`
- `provider_input_tokens`
- `provider_output_tokens`
- `generation_embedding_overlap_us`

Each metric metadata object should contain only schema/digest/scope identifiers, never source or provider content.

### 13.3 Why not import every operation into researchctl tables

The laboratory is optimized for immutable experiment attempts, artifacts, metrics, and traces. Duplicating every Workflow operation into a new researchctl table would:

- couple researchctl migrations to scraper runtime details;
- duplicate canonical data;
- complicate import idempotency;
- invite inconsistent projections.

The canonical JSONL remains primary evidence. Researchctl stores its verified identity and selected query-friendly metrics.

## 14. Privacy and threat model

### 14.1 Trusted and untrusted parties

- Workflow store/runtime code is trusted authority.
- Registered Go host module factories are trusted but must use bounded APIs.
- JavaScript task code is not trusted with arbitrary process-global or database authority.
- Restricted child processes are not trusted with parent database or credentials.
- Provider responses and errors are untrusted data.

### 14.2 Main threats

1. Provider error text contains response bodies or secret URLs.
2. A module uses an arbitrary metadata map to persist prompts.
3. JavaScript invents counter names or huge values.
4. A stale worker admits another call after cancellation.
5. A stale worker completes a different operation.
6. Concurrent finishes create conflicting terminal evidence.
7. Integer overflow corrupts cost/token settlement.
8. An export writes partial bytes and is mistaken for canonical evidence.
9. A restricted child gains the SQLite path.

### 14.3 Structural mitigations

- closed schema with no string payload fields;
- host-declared descriptors and counter allowlists;
- safe-name and SHA-256 validation;
- checked nonnegative `int64` arithmetic;
- lease fence on admission;
- secret operation ticket on completion;
- unique constraints and completion digest idempotency;
- atomic canonical export and manifest digest;
- no recorder in JavaScript context;
- no DB path or ticket in isolated protocol;
- canary scans across SQLite, WAL, JSONL, manifests, logs, reports, and graphs.

## 15. Failure semantics

| Condition | Operation evidence | Workflow failure | Budget behavior |
|---|---|---|---|
| Provider succeeds | `succeeded`, actual span/counters | task may succeed | actual |
| Provider returns safe error | `failed`, safe class/code | domain/provider typed failure | actual if complete, otherwise conservative |
| Provider output malformed after response | `failed`, malformed-output code, provider span | retryable domain failure | actual if known |
| Context deadline | `timed-out` | retryable policy decision | conservative unless provider usage known |
| Cancellation | `canceled` | canceled attempt/run | conservative unless usage known |
| Admission denied | no operation row or rejected begin event | budget/authority failure | no new spend |
| Process dies after admission | incomplete admission | lease loss/infrastructure evidence | conservative allocation |
| Lease expires during call | completion may still append by ticket | stale task completion rejected | actual/conservative by completion |
| Duplicate identical finish | one completion | none | unchanged |
| Conflicting finish | first completion retained | internal evidence conflict | unchanged; alert operator |
| SQLite finish fails | admission remains incomplete | task must not report clean success | conservative |

## 16. Decision records

### Decision: Put the live ledger in Workflow V3 SQLite

- **Context:** External operations occur inside Workflow attempts and must share lease and budget authority.
- **Options considered:** sweep-specific JSON sidecar; researchctl live sink; OpenTelemetry-only traces; Workflow SQLite tables.
- **Decision:** Add normalized tables and APIs to scraper Workflow V3 SQLite.
- **Rationale:** This is the only location that can transactionally validate leases and attempt reservations without introducing a distributed dependency.
- **Consequences:** Scraper gains a generic persistence feature and migration; RAG becomes an adapter; researchctl remains downstream custody.
- **Status:** proposed.

### Decision: Separate immutable admission and completion records

- **Context:** Admission must survive a crash before completion, and evidence must show what was known at each boundary.
- **Options considered:** one mutable status row; event-only journal; separate admission/completion tables.
- **Decision:** Use separate normalized tables with a one-to-zero-or-one relationship.
- **Rationale:** It makes the pre-call authority fact explicit and preserves incomplete operations without invented terminal data.
- **Consequences:** Queries require a join; append-only semantics and crash interpretation become simple.
- **Status:** proposed.

### Decision: Use lease fencing for Begin and ticket fencing for Finish

- **Context:** Cancellation must stop new effects but must not erase the outcome of an already-started effect.
- **Options considered:** active lease required for both; no fence on finish; operation-specific capability.
- **Decision:** Begin requires the current lease; Finish requires a secret operation ticket and may occur after lease invalidation.
- **Rationale:** It preserves late evidence without granting stale output authority.
- **Consequences:** Completion keys require secure generation, digest storage, constant-time comparison, and idempotency tests.
- **Status:** proposed.

### Decision: Do not expose generic recording to JavaScript

- **Context:** JavaScript is ergonomic intent, not host authority, and arbitrary records create a privacy channel.
- **Options considered:** `workflow/task.operations`; public `workflow/operations` module; host-module-only recorder.
- **Decision:** Inject the recorder only into trusted Go `TaskModuleContext`.
- **Rationale:** Host descriptors can enforce kinds, counters, cardinality, and privacy.
- **Consequences:** Domain module changes are required; task scripts remain stable.
- **Status:** proposed.

### Decision: Keep researchctl downstream and artifact-oriented

- **Context:** Researchctl provides immutable scientific custody but should not participate in every provider call.
- **Options considered:** direct live writes to laboratory SQLite; custom researchctl operation tables; verified JSONL artifacts plus metrics.
- **Decision:** Import canonical operation JSONL/manifests as verified artifacts and store selected derived metrics.
- **Rationale:** Existing run-export verification and import are sufficient and avoid runtime coupling.
- **Consequences:** Detailed operation queries read the artifact; high-level research queries use metrics.
- **Status:** proposed.

### Decision: Defer restricted-child operation streaming

- **Context:** Restricted children have one terminal response and no parent database authority; child death loses terminal-only observations.
- **Options considered:** pass DB path; include observations only in final response; parent-mediated streaming protocol; defer.
- **Decision:** Implement parent-hosted operations first and design an NDJSON/broker extension separately if a restricted workload needs external effects.
- **Rationale:** Passing DB authority violates isolation, while terminal-only response does not satisfy failure durability.
- **Consequences:** Phase 1 supports trusted host modules; isolated tools remain measured at the parent invocation boundary only.
- **Status:** proposed.

### Decision: Require durable SQLite admission semantics

- **Context:** An admitted provider request may create cost even if the process crashes immediately.
- **Options considered:** WAL default synchronous mode; explicit `FULL`; periodic checkpoint; file sidecar.
- **Decision:** Require and test a configuration whose commit is durably synced before `Begin` returns, with `synchronous=FULL` as the default design.
- **Rationale:** Checkpointing later does not close the pre-call crash window.
- **Consequences:** Admission has additional I/O latency; benchmark and document it rather than weakening correctness.
- **Status:** proposed.

## 17. Alternatives rejected

### Measurements only in task outputs

Rejected because failures and cancellation prevent output publication.

### Measurements only in `v3_events`

Rejected because events are bounded wake-up notifications with JSON payloads, not normalized source-of-truth records. An event payload also invites arbitrary metadata.

### Use attempt timing as provider timing

Rejected because attempt timing includes runtime construction, queueing, artifact I/O, and persistence. It cannot support valid provider concurrency or latency claims.

### Sweep-specific fsynced JSON journal

Useful as an emergency containment mechanism, but rejected as the long-term architecture because every HTTP/browser/provider workload would reimplement admission, redaction, fencing, and export.

### OpenTelemetry spans only

Rejected as primary evidence because exporter loss, sampling, collector availability, and backend retention are not Workflow durability guarantees. An OpenTelemetry adapter may mirror committed operations later.

### Direct researchctl observation sink from tasks

Rejected because it creates cross-store ordering, availability, and authority problems. Researchctl is immutable custody after Workflow evidence exists.

### Arbitrary JSON attributes

Rejected because the privacy boundary would be unenforceable. Closed integer counters and digests are sufficient for performance evidence.

## 18. Implementation plan

### Phase 1: Contracts and validation

Files:

- new `pkg/workflowv3/external_operation.go`;
- new `pkg/workflowv3/external_operation_test.go`.

Work:

1. Define descriptors, specs, tickets, completions, counters, query records, and constants.
2. Add safe-name, version, digest, outcome, accounting-mode, timestamp, cardinality, ordering, and overflow validation.
3. Add canonical descriptor digest calculation.
4. Add fuzz tests for validators and strict decode.

Exit criteria:

- invalid/duplicate/unsorted counters fail closed;
- canonical descriptor identity is deterministic;
- no generic payload field exists.

### Phase 2: Additive SQLite schema and durability

Files:

- `pkg/workflowv3sqlite/schema.sql`;
- `pkg/workflowv3sqlite/store.go`;
- `pkg/workflowv3sqlite/store_test.go`.

Work:

1. Add operation tables, constraints, foreign keys, and indexes.
2. Configure and assert durable synchronous mode.
3. Extend legacy-minimal-database migration tests to require new tables.
4. Add invariant checks for orphan counters/completions and over-allocation.

Exit criteria:

- existing database fixtures reopen unchanged;
- new tables exist and foreign keys reject invalid rows;
- startup fails on reconciliation corruption.

### Phase 3: Begin/Finish store APIs

Files:

- new `pkg/workflowv3sqlite/external_operation.go`;
- new `pkg/workflowv3sqlite/external_operation_test.go`.

Work:

1. Implement lease-fenced Begin transaction.
2. Implement completion-key generation and digest comparison.
3. Implement late, exactly-once, idempotent Finish transaction.
4. Add bounded admitted/completed events.
5. Add paginated coherent queries.

Exit criteria:

- stale lease cannot begin;
- canceled lease's existing ticket can finish;
- ticket cannot finish another operation;
- concurrent finishes create one completion;
- restart between Begin and Finish preserves admission.

### Phase 4: Runtime injection

Files:

- `pkg/workflowv3runtime/engine.go`;
- `pkg/workflowv3runtime/task_runner.go`;
- `pkg/workflowv3runtime/modules.go`;
- corresponding unit/integration tests.

Work:

1. Add descriptor declarations to module factories.
2. Clone and validate descriptor registries.
3. Create the scoped recorder in `Engine.ExecuteLease`.
4. Pass it through `TaskRequest` and `TaskModuleContext`.
5. Confirm JavaScript task context has no new operation API.
6. Make operation persistence failure a typed task/infrastructure failure.

Exit criteria:

- only requested trusted modules receive matching descriptors;
- fresh runtimes receive fresh scoped recorders;
- no process-global recorder exists;
- race tests pass.

### Phase 5: Budget reconciliation

Files:

- `pkg/workflowv3sqlite/budget.go`;
- operation store file;
- budget and operation integration tests.

Work:

1. Enforce admission allocations against attempt reservations.
2. Calculate derived actual/conservative operation usage.
3. Compare derived usage with task-reported usage.
4. Integrate operation-bound dimensions into settlement.
5. Reconcile incomplete operation allocations on lease loss/cancellation.

Exit criteria:

- no admitted call exceeds requests/tokens/cost reservation;
- unknown calls charge conservatively;
- actual usage cannot exceed allocation;
- checked arithmetic rejects overflow;
- retries spend new reservations.

### Phase 6: Projections and canonical export

Files:

- `pkg/workflowv3/types.go`;
- `pkg/workflowv3sqlite/operational.go`;
- new `pkg/workflowv3sqlite/external_operation_export.go`;
- tests.

Work:

1. Add bounded active/outcome/orphan projections.
2. Add coherent partial and terminal export.
3. Add canonical JSONL and manifest digest.
4. Add atomic file publication helper or writer contract.
5. Prove byte-identical repeated export from unchanged state.

Exit criteria:

- partial failed runs export completed and incomplete operations;
- pagination cannot reorder or duplicate rows;
- output is canonical, fsynced, and privacy-safe.

### Phase 7: RAG adapter

Files:

- `internal/workflowv3ttc/module.go`;
- `internal/workflowv3ttc/provider.go`;
- `internal/workflowv3ttc/contracts.go`;
- provider/module tests.

Work:

1. Declare generation and embedding descriptors.
2. Pass the scoped recorder through explicit invocation context.
3. Wrap each provider call in Begin/Finish.
4. Normalize every return path to closed outcome/failure codes.
5. Capture safe measures and checked usage counters.
6. Treat ledger spans as authoritative; remove duplicated timing only after compatibility review.

Exit criteria:

- success, malformed output, provider failure, timeout, and cancellation all produce operation evidence;
- provider bodies and source values never enter the ledger;
- one provider request maps to one admitted operation.

### Phase 8: Sweep custody and analytics

Files:

- `cmd/rag-ttc-v3-sweep/main.go`;
- graph renderer and tests.

Work:

1. Export operation evidence before deleting each cell runtime.
2. Build successful and failed cell checkpoints from the same operation source.
3. Make JSON, JSONL, and CSV publication atomic.
4. Reconstruct aggregate evidence deterministically from cell checkpoints.
5. Derive latency/concurrency/overlap with missing-data semantics.

Exit criteria:

- a later cell failure cannot destroy prior cell evidence;
- failed-cell completed provider spans remain queryable;
- incomplete calls are explicit and excluded from latency denominators.

### Phase 9: Researchctl custody

Files in researchctl:

- adapter package chosen by the implementation owner;
- `pkg/lab` tests only if a generic helper is needed;
- import/export integration tests.

Work:

1. Register operation JSONL and manifest as verified artifacts.
2. Record selected scalar metrics.
3. Export, import into a fresh laboratory, and re-export.
4. Verify artifact bytes/digests and run-export identity.

Exit criteria:

- imported custody survives deletion of the source bundle;
- repeated import is idempotent;
- conflicting digest/source identity fails closed;
- no live researchctl dependency exists in Workflow.

### Phase 10: Qualification

1. Run fixture providers with injected delays and failures.
2. Run cancellation, lease loss, process death, restart, stale completion, and concurrency tests.
3. Run privacy canary scans of SQLite, WAL, JSONL, CSV, logs, reports, and graphs.
4. Run focused tests, all tests, race tests, lint, release builds, repository hooks, and `git diff --check` with `GOWORK=off` where required.
5. Only then authorize a small real-provider smoke.

## 19. Test strategy

### 19.1 Unit tests

- Descriptor digest stable under clone and decode.
- Unsorted, duplicate, unknown, oversized, negative, or overflowing counters rejected.
- Completion outcome/failure combinations validated:
  - success cannot include failure code;
  - failure must include class/code;
  - incomplete export has no fabricated completion;
  - accounting mode matches declared counters.
- Completion key never appears in JSON, events, snapshots, errors, or logs.

### 19.2 SQLite transaction tests

- Begin commits before a test provider callback runs.
- Admission survives close/reopen.
- Stale lease Begin returns `ErrStaleCompletion`.
- Finish after cancellation succeeds with valid ticket.
- Finish cannot modify attempt/node/run state.
- Two goroutines finishing the same ticket produce one row.
- Identical finish is idempotent; conflicting finish returns a typed error.
- Corrupt foreign keys or allocation totals fail reconciliation.

### 19.3 Runtime tests

- Module factory receives only descriptors it declared.
- JavaScript cannot require or discover an operation API.
- New runtime/attempt gets a fresh recorder.
- Provider error still commits completion before task failure.
- Failure to persist completion prevents clean task success.
- Cancellation uses a bounded non-canceled finish context.

### 19.4 Failure matrix

Inject these boundaries independently:

```text
before Begin
inside Begin before commit
after Begin commit before call
provider returns success
provider returns typed error
provider returns malformed output
context timeout
context cancellation
Finish before insert
Finish after insert before commit
attempt failure after Finish
lease expiry during provider call
parent restart with incomplete operation
```

For each case assert:

- operation rows;
- attempt rows;
- budget reservation/settlement;
- retry debt;
- event sequence;
- canonical export;
- privacy canaries.

### 19.5 Performance tests

Measure ledger overhead separately from provider latency:

- Begin transaction p50/p95/p99;
- Finish transaction p50/p95/p99;
- operations/second under 1 and 4 concurrent writers;
- WAL growth per operation;
- export time and size for 10, 1,000, and 100,000 operations;
- impact of `synchronous=FULL`.

The benchmark is descriptive. Correctness and authority durability are not relaxed to meet a target.

### 19.6 Privacy canaries

Plant unique values representing:

- source text;
- prompt text;
- response text;
- authorization token;
- secret URL;
- vector bytes;
- provider error body;
- local database path.

After every failure mode, scan:

```text
workflow SQLite
workflow SQLite-WAL and SHM while open
canonical operation JSONL and manifest
cell JSON/JSONL/CSV
logs and operational events
researchctl export/import artifacts
rendered report and graph labels
```

Only digests and approved integer measures may survive.

## 20. Implementation review checklist

### Contracts

- [ ] No arbitrary payload or metadata field exists.
- [ ] Descriptor and completion validation is centralized.
- [ ] Integer arithmetic is checked and nonnegative.
- [ ] JavaScript cannot create descriptors or tickets.

### Persistence

- [ ] Admission is durably committed before call submission.
- [ ] Completion is immutable and exactly once.
- [ ] Existing databases migrate additively.
- [ ] Foreign keys and reconciliation detect corruption.

### Fencing

- [ ] Begin uses active lease token and cancel epoch.
- [ ] Finish uses operation ticket, not active lease.
- [ ] Finish cannot mutate Workflow completion state.
- [ ] Late stale node output remains rejected.

### Budget

- [ ] Allocation cannot exceed attempt reservation.
- [ ] Incomplete/unknown operations charge conservatively.
- [ ] Actual counters cannot exceed allocation.
- [ ] Retry creates a new attempt reservation and operation.

### Custody

- [ ] Partial and terminal exports are canonical.
- [ ] JSONL/manifest publication is atomic and fsynced.
- [ ] Failed runs preserve completed provider spans.
- [ ] researchctl verifies digest and size before import.

### Privacy

- [ ] No provider body/error text is persisted.
- [ ] No URL/header/prompt/source/vector field exists.
- [ ] Completion key is never serialized.
- [ ] Canary scans include WAL and rendered artifacts.

## 21. Intern-oriented first implementation walkthrough

A new implementer should use this sequence.

### Step A: Make the domain compile without SQLite

Create the types and validators. Write table-driven tests first. Use canonical digest helpers already present in `pkg/workflowv3/canonical.go`. Do not add database code until malformed descriptors and completions are rejected deterministically.

### Step B: Add schema and migration tests

Add all tables to `schema.sql`. Extend `TestOpenMigratesCompletedMinimalSliceDatabase` so opening the historical minimal database creates the new tables. Add direct SQL tests for foreign keys and unique constraints.

### Step C: Implement Begin alone

Implement admission with a generated ticket and query it back after reopen. Add a fake provider function that flips an atomic boolean; assert the admission query succeeds before invoking the fake provider. Test stale lease and reservation over-allocation.

### Step D: Implement Finish alone

Start with success, then failure, then late cancellation, then concurrent idempotency. Ensure event payloads contain only operation ID, kind, attempt number, outcome, and completion digest.

### Step E: Wire one test host module

Create a fixture module descriptor and a Go function that begins, sleeps, and finishes. Verify its JavaScript caller sees only the domain function. Kill/cancel the task at controlled boundaries.

### Step F: Integrate budget reconciliation

Do not guess. Build fixtures with known request/token/cost allocations and compare every SQL balance before and after actual, conservative, retry, and cancellation paths.

### Step G: Export and reopen

Export a mixed run containing success, failed completion, and incomplete admission. Close and reopen SQLite, export again, and compare bytes. Then import the files into a temporary researchctl laboratory and verify the run-export digest.

### Step H: Adapt RAG

Only after the generic machinery passes race and privacy tests, replace the RAG in-output timing path with authoritative operation evidence. Re-run fixture matrices and forced malformed-output retries before any real provider.

## 22. Open questions

These questions require review before implementation merges:

1. Should Workflow V3 globally switch to SQLite `synchronous=FULL`, or should operation authority use a dedicated durable connection/store? The design defaults to global `FULL` pending benchmark evidence.
2. Should descriptor identities become compiler-pinned plan fields in the first release, or is module-registry generation plus recorded descriptor digest sufficient for the first additive slice? Exact reproducibility favors compiler pinning.
3. How should operation-bound usage coexist temporarily with `task.usage.report`? The recommended path is compare-first, derive-second.
4. Should incomplete operations ever receive an explicit immutable `unknown` completion during operator-directed abandonment, or remain completion-less forever? Completion-less is safer unless a separate abandonment command records who/why without fabricating provider timing.
5. Which package should own the Workflow-to-researchctl adapter? It should not create a scraper-to-researchctl dependency if a higher-level integration repository can own the bridge.
6. Is `input_runes` acceptable in general operation evidence, or should RAG retain it only in a domain artifact? Privacy review should decide; source length can be sensitive in some deployments.
7. When a provider reports usage after timeout, can it be trusted as actual, or should policy require conservative charging? The descriptor/accounting policy should decide explicitly.

## 23. Acceptance criteria

The ticket is complete when all of the following are evidenced:

1. A provider call that returns an error has a precise provider-wall span after task failure.
2. A canceled call can finish operation evidence while stale task output remains rejected.
3. A crash after admission leaves an explicit incomplete operation and conservative accounting.
4. One provider request corresponds to one Workflow operation admission.
5. Provider concurrency and overlap are computed from operation spans, not attempts.
6. Failed and partial cells produce canonical operation JSONL and a digest manifest.
7. Existing Workflow databases migrate and existing tests remain unchanged in behavior.
8. JavaScript and restricted children gain no database or arbitrary recording authority.
9. Privacy canaries are absent from SQLite, WAL, exports, reports, and graphs.
10. researchctl export/import retains byte-identical artifact custody and idempotent identity.
11. Focused/full/race tests, lint, builds, hooks, doctor, and diff checks pass.
12. A bounded fixture study demonstrates exact latency, retry, concurrency, overlap, and missing-data semantics before any paid run.

## 24. File-level reference map

### Scraper

- `pkg/workflowv3/types.go:282-319,352-395` — attempt, lease, budget, event, and projection contracts to extend.
- `pkg/workflowv3/canonical.go` — canonical JSON and digest helpers.
- `pkg/workflowv3sqlite/schema.sql:231-303` — existing budget reservations and attempts; add normalized operation tables nearby.
- `pkg/workflowv3sqlite/store.go:28-57` — WAL/migration/open behavior and durability configuration.
- `pkg/workflowv3sqlite/store.go:410-476` — transaction model for lease and budget admission.
- `pkg/workflowv3sqlite/store.go:567-662` — transaction model for fenced task completion.
- `pkg/workflowv3sqlite/store.go:1113-1131` — active lease fence used by Begin but intentionally not Finish.
- `pkg/workflowv3sqlite/operational.go:13-154` — coherent read-transaction projections.
- `pkg/workflowv3runtime/engine.go:272-320` — lease owner and recorder construction point.
- `pkg/workflowv3runtime/task_runner.go:21-29,69-107` — request and module construction seam.
- `pkg/workflowv3runtime/task_runner.go:214-252` — JavaScript task context that must not expose generic recording.
- `pkg/workflowv3runtime/modules.go:22-63` — host factory/registry descriptor seam.
- `pkg/workflowv3runtime/isolation_protocol.go` — restricted child boundary; no parent DB authority.

### RAG evaluation

- `internal/workflowv3ttc/provider.go:80-134` — current in-memory provider timing and failure-loss path.
- `internal/workflowv3ttc/module.go:15-82` — task-scoped host-module invocation seam.
- `internal/workflowv3ttc/contracts.go:43-67` — current success-output measurement contracts.
- `cmd/rag-ttc-v3-sweep/main.go:270-329` — current successful/failed cell custody flow.
- `cmd/rag-ttc-v3-sweep/main.go:377-455` — successful output reduction versus compact failed-cell evidence.
- `cmd/rag-ttc-v3-sweep/admission.go` — cumulative cross-cell authority that remains a separate boundary.

### Researchctl

- `pkg/lab/runtime.go:70-83` — downstream observation sink.
- `pkg/lab/types.go:91-157` — verified artifact, metric, trace, attempt, and run-export contracts.
- `pkg/lab/artifacts.go:18-84,497-499` — artifact verification and canonical run-export digest.
- `internal/labsqlite/import.go:13-58,125-258` — validated, idempotent, transactional import.
- `internal/labsqlite/migrations.go:12-170` — immutable laboratory table model.

## 25. Final recommendation

Implement this as a generic Workflow V3 subsystem, not as another sweep sidecar. Start with a narrow host-only API, append-only admission/completion tables, strict closed schemas, and precise failure tests. Keep task output, operation evidence, budget accounting, and research custody as distinct layers. Integrate RAG only after the generic store proves cancellation, late completion, restart, and privacy behavior.

The design deliberately chooses correctness over convenience: an external call cannot begin without durable authority, a failed call cannot erase its timing, a stale task cannot publish output, and no component can smuggle provider or source payloads into the evidence ledger.
