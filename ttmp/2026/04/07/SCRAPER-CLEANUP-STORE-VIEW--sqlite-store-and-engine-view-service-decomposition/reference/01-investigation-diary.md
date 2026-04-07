---
Title: Investigation diary
Ticket: SCRAPER-CLEANUP-STORE-VIEW
Status: active
Topics:
    - scraper
    - backend
    - architecture
    - cleanup
    - sqlite
    - api
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Records why store.go and engineview/service.go are ready for decomposition."
LastUpdated: 2026-04-07T16:28:00-04:00
WhatFor: "Resume the cleanup plan with the original reasoning intact."
WhenToUse: "Use when implementing or reviewing the store/view split."
---

# Investigation diary

## Main observation

Two files have become broad aggregation points:

- `pkg/engine/store/sqlite/store.go`
- `pkg/services/engineview/service.go`

Both are still functional and internally understandable, but both now mix multiple subdomains that can be separated without changing package boundaries.

## Store findings

`store.go` currently owns workflow operations, op lifecycle, queue limiting, result persistence, artifact persistence, and helper functions. That makes every review and merge broader than it needs to be.

## Engine view findings

`service.go` currently owns workflow reads, queue reads, artifact reads, op-result reads, retry/cancel helpers, DB bootstrap helpers, and artifact preview classification. That is too much for one service file.

## Recommendation

Split by responsibility inside the existing packages first. Do not redesign interfaces during the first pass.

## Implementation log

### First cleanup slice: engineview artifact reads and DB helpers

I started with `pkg/services/engineview` because it is a lower-risk structural cleanup than the SQLite store and it sits behind existing tests.

Files added:

- `pkg/services/engineview/artifact_read_service.go`
- `pkg/services/engineview/db_helpers.go`

Files updated:

- `pkg/services/engineview/service.go`

What moved into `artifact_read_service.go`:

- `ArtifactSummary`
- `ArtifactDetail`
- `WorkflowArtifactsResult`
- `ListWorkflowArtifactsOptions`
- `ListArtifacts(...)`
- `ListWorkflowArtifacts(...)`
- `GetOpResult(...)`
- `GetArtifact(...)`

What moved into `db_helpers.go`:

- `openStore(...)`
- `openReadDB(...)`
- `workflowExists(...)`
- `workflowOpExists(...)`
- `enrichArtifactSummary(...)`
- `classifyArtifactPreview(...)`
- `loadDependencies(...)`
- `loadLease(...)`

What stayed in `service.go`:

- service type
- workflow reads
- queue reads
- retry/cancel mutations

Validation for this slice:

```bash
go test ./pkg/services/engineview ./pkg/api/server -count=1
```

Both packages passed after the move.

### Second cleanup slice: remaining engineview reads and mutations

After the artifact/helper split, I moved the rest of `engineview/service.go` into dedicated files:

Files added:

- `pkg/services/engineview/workflow_read_service.go`
- `pkg/services/engineview/queue_read_service.go`
- `pkg/services/engineview/workflow_mutation_service.go`

What moved into `workflow_read_service.go`:

- `WorkflowSummary`
- `WorkflowOp`
- `ListWorkflowsOptions`
- `WorkflowListItem`
- `WorkflowListResult`
- `Workflow(...)`
- `WorkflowOps(...)`
- `ListWorkflows(...)`

What moved into `queue_read_service.go`:

- `QueueStatus`
- `ListQueues(...)`

What moved into `workflow_mutation_service.go`:

- `RetryOp(...)`
- `CancelWorkflow(...)`

Result:

- `service.go` is now a thin file with the service type, constructor, and `EngineStatus(...)`.

Validation for this slice:

```bash
go test ./pkg/services/engineview ./pkg/api/server -count=1
```

Both packages passed again after the move.

### Third cleanup slice: SQLite queue limiter and shared helper extraction

After the `engineview` split, I moved the helper-heavy tail of `pkg/engine/store/sqlite/store.go` into two dedicated files:

Files added:

- `pkg/engine/store/sqlite/queue_limiter.go`
- `pkg/engine/store/sqlite/sql_helpers.go`

What moved into `queue_limiter.go`:

- `queueLimiterState`
- `countActiveLeasesForQueue(...)`
- `loadQueueLimiterState(...)`
- `refillQueueLimiterState(...)`
- `upsertQueueLimiterState(...)`

What moved into `sql_helpers.go`:

- `queryer`
- `loadDependenciesTx(...)`
- `execer`
- `execRowsAffected(...)`
- `insertOps(...)`
- `lookupOpContext(...)`
- `normalizeEmittedOps(...)`
- `initialStatus(...)`
- `nullableParentID(...)`
- `nullableTime(...)`
- `boolToInt(...)`
- `jsonText(...)`
- `mustJSON(...)`
- `nullableJSON(...)`
- `unmarshalJSON(...)`

What intentionally stayed in `store.go` for the next slice:

- `loadArtifacts(...)`
- core workflow/op/lease/result methods

Validation for this slice:

```bash
go test ./pkg/engine/store/sqlite -count=1
```

The SQLite package stayed green after the move.

### Fourth cleanup slice: SQLite result and artifact storage methods

With the helper and limiter tail moved out of `store.go`, the next clean chunk was the result/artifact section.

File added:

- `pkg/engine/store/sqlite/result_store.go`

What moved into `result_store.go`:

- `CompleteOp(...)`
- `FailOp(...)`
- `GetResult(...)`
- `loadArtifacts(...)`

Why this was a good next split:

- these methods all operate on the `results` and `artifacts` tables
- they share the same completion/failure lifecycle concerns
- they still use shared store helpers from `sql_helpers.go`, so the extraction stayed move-only

What changed in `store.go`:

- removed the duplicated result/artifact methods after adding `result_store.go`
- left lease handling, workflow creation, and op enqueue/read methods in place for later slices

Validation for this slice:

```bash
gofmt -w pkg/engine/store/sqlite/store.go pkg/engine/store/sqlite/result_store.go
go test ./pkg/engine/store/sqlite -count=1
```

The package passed after the move, so the public `Store` surface remained intact while reducing `store.go` further.

### Fifth cleanup slice: SQLite workflow methods

After moving result/artifact logic out, the workflow-oriented methods were a clean, self-contained group.

File added:

- `pkg/engine/store/sqlite/workflow_store.go`

What moved into `workflow_store.go`:

- `CreateWorkflow(...)`
- `GetWorkflow(...)`
- `UpdateWorkflowStatus(...)`
- `GetWorkflowStats(...)`

Why this split works well:

- these methods all center on `workflows` plus workflow-scoped status aggregation
- they reuse existing shared helpers instead of adding new behavior
- removing them cuts a large chunk of non-lease, non-op logic out of `store.go`

What stayed in `store.go` after this slice:

- `Open(...)`, `Close(...)`, and `CurrentVersion(...)`
- op enqueue/read/runnable logic
- queue candidate listing
- lease acquisition and heartbeat
- dependency loading wrapper

Validation for this slice:

```bash
gofmt -w pkg/engine/store/sqlite/store.go pkg/engine/store/sqlite/workflow_store.go
go test ./pkg/engine/store/sqlite -count=1
```

The package stayed green again, so the move remained strictly structural.

### Sixth cleanup slice: SQLite op methods

Once the workflow methods were out, the remaining non-lease body of `store.go` was almost entirely op-centric.

File added:

- `pkg/engine/store/sqlite/op_store.go`

What moved into `op_store.go`:

- `Enqueue(...)`
- `GetOp(...)`
- `RefreshRunnableOps(...)`
- `ListQueueCandidates(...)`
- `loadDependencies(...)`

Why this grouping made sense:

- these methods all work from the `ops` table or `op_dependencies`
- `RefreshRunnableOps(...)` and `ListQueueCandidates(...)` are scheduler-facing op lifecycle helpers
- `loadDependencies(...)` belongs with op materialization, not with leasing

What remained in `store.go` after this slice:

- store shell: `Open(...)`, `Close(...)`, `CurrentVersion(...)`
- lease acquisition and heartbeat only

Validation for this slice:

```bash
gofmt -w pkg/engine/store/sqlite/store.go pkg/engine/store/sqlite/op_store.go
go test ./pkg/engine/store/sqlite -count=1
```

The SQLite package passed again after the move, which left the final lease-only slice as the remaining store decomposition step.
