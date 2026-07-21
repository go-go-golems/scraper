---
Title: Slice 7 Bounded Reductions - Deterministic Durable Scale-In
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
    - Path: repo://pkg/gojamodules/workflow/authoring.go
      Note: Go-backed reduce authoring handles
    - Path: repo://pkg/workflowv3/compiler.go
      Note: Reduction schema policy and graph validation
    - Path: repo://pkg/workflowv3/types.go
      Note: |-
        Reduction declarations typed set refs and bounded fan-in policy
        Canonical bounded reduction contract will extend this model
    - Path: repo://pkg/workflowv3runtime/dispatcher.go
      Note: Interleaved reducer admission under normal resource limits
    - Path: repo://pkg/workflowv3sqlite/schema.sql
      Note: Durable reduction level and partition state
    - Path: repo://pkg/workflowv3sqlite/store.go
      Note: |-
        Deterministic partition materialization and completion
        Partition materialization and root transitions belong in this store
ExternalSources: []
Summary: Implementation contract for deterministic restart-safe reduction trees whose partition fan-in and payload size remain bounded regardless of source cardinality.
LastUpdated: 2026-07-21T20:35:00-04:00
WhatFor: Freeze Slice 7 partition identity ordering state transitions recovery and root publication before implementation.
WhenToUse: Read before adding reduce authoring partition manifests dynamic reducer nodes or aggregate progress to Workflow V3.
---


# Slice 7 Bounded Reductions - Deterministic Durable Scale-In

## Executive summary

Slice 7 converts a large typed set into a smaller set and ultimately one
validated root through an immutable reduction tree. Every reducer consumes at
most the compiled `fanIn`. Partition membership is determined by canonical
item-key order, never task completion order. Successful partitions survive
restart and are not recomputed because siblings failed.

The reference fixture is word count. The same input must produce byte-identical
root output and digest at different concurrency levels and under randomized
completion schedules.

## Problem statement

One root node depending directly on thousands of map children creates an
unbounded dependency row set, an unbounded input manifest, and a runtime memory
spike. Reducing results in completion order is bounded but nondeterministic.
Rebuilding every aggregate after one failure wastes completed work.

## Scope and non-goals

This slice includes reduce IR and DSL, deterministic partition trees, bounded
partition manifests, dynamic reducer nodes, restart recovery, root
publication, progress, and malformed-shard isolation.

It does not attempt arbitrary streaming algebra, speculative duplicate
reducers, incremental mutable aggregates, or partial-result success. The first
contract requires a complete validated source level before materializing the
next level.

## Canonical model

```go
type ReducePolicy struct {
    FanIn int `json:"fanIn"`
    MaxLevels int `json:"maxLevels"`
}

type IRReduce struct {
    Key string `json:"key"`
    Source SetRef `json:"source"`
    PartitionTask IRNodeTemplate `json:"partitionTask"`
    Policy ReducePolicy `json:"policy"`
    RootSchema string `json:"rootSchema"`
}
```

The compiled partition template pins exact implementation, input partition
manifest schema, output schema, modules, resource class, retry, and effective
fan-in. Fan-in less than two is rejected unless a separately defined identity
operation exists. Host policy caps the request.

The authoring callback runs once with a symbolic `PartitionRef<I>`:

```typescript
reduce<I, O>(
  name: string,
  source: SetRef<I>,
  task: (partition: PartitionRef<I>) => WorkflowTask<readonly I[], O>,
  configure?: (r: ReduceBuilder) => void,
): ValueRef<O>;
```

No runtime reducer function is persisted.

## Deterministic ordering and identity

Level 0 membership comes from the source output-manifest sorted by canonical
item key. Higher levels use the prior level's partition ordinal order.
Completion timestamps are irrelevant.

For each partition compute an identity envelope:

```json
{
  "reduceKey": "sum-word-counts",
  "sourceDigest": "sha256:...",
  "level": 0,
  "ordinal": 3,
  "memberDigests": ["sha256:...", "sha256:..."]
}
```

Member identities are already in canonical partition order. Digest this exact
canonical representation. The reducer node key and partition-manifest digest
must have golden tests. A conflicting row with the same logical key but
different membership is an identity failure.

## Partition manifest

A versioned immutable artifact contains:

```json
{
  "schema": "scraper-workflow-reduction-partition/v1",
  "reduceKey": "sum-word-counts",
  "level": 0,
  "ordinal": 3,
  "members": [
    {"itemKey": "doc-0048", "value": {"schema": "word-count/v1", "digest": "sha256:...", "mediaType": "application/json", "size": 91, "locator": "cas://..."}}
  ]
}
```

The manifest contains at most `fanIn` compact refs. Reducer tasks resolve
payloads through their lease-scoped workspace; control rows never contain word
maps or source text.

## Durable schema

```text
v3_reductions
  run_id, reduce_key, source_digest, fan_in, current_level,
  source_items, current_items, status, root_schema, root_ref columns,
  updated_at

v3_reduction_partitions
  run_id, reduce_key, level, ordinal, partition_digest,
  first_member_key, member_count, node_key, status, output ref columns
```

Primary key is `(run_id,reduce_key,level,ordinal)`. A unique
`(run_id,node_key)` ties each partition to the normal node execution path.
Partition rows carry compact manifest/output refs only.

Do not add one dependency from a final root to every original map child. Each
partition node depends only on the bounded producers represented by its
manifest or on an engine-owned readiness condition that is atomically proven
when the manifest is created.

## Coordinator state machine

```text
pending-source
  → materializing-level-0
  → executing-level-0
  → materializing-level-N
  → executing-level-N
  → publishing-root
  → succeeded | failed | canceled
```

A level is ready to materialize when its source set is complete, validated, and
has an immutable canonical manifest. The coordinator divides that manifest into
contiguous partitions of at most `fanIn` and inserts partition rows/nodes in
bounded transactions. For very large levels, page the partition insertion with
a durable cursor using the same principles as Slice 6.

When every partition in a level succeeds, create the next level's canonical
manifest from outputs sorted by partition ordinal. If the level has one output,
validate it against `rootSchema` and publish it as the reduction root.

## Transaction boundaries

Partition planning reads immutable manifest bytes outside the SQLite write
transaction. The insertion transaction checks expected source digest, level,
and cursor, inserts deterministic rows/nodes/dependencies, updates coordinator
state, emits one bounded event, and commits.

Partition completion is the normal fenced task completion plus an atomic
partition status update. The transition that declares a level complete must
observe exact expected partition count and all succeeded statuses in one
transaction.

Root publication writes an immutable artifact before its fenced pointer/ref
commit. Repeating publication after a crash produces the same digest.

## Failure semantics

A reducer task can retry independently. Completed sibling partitions remain
succeeded. A malformed output fails with a typed schema/validation code and
cannot become an input to the next level.

Stable engine failures include:

- `validation/REDUCTION_SOURCE_INCOMPLETE`;
- `validation/REDUCTION_PARTITION_SCHEMA`;
- `validation/REDUCTION_CARDINALITY`;
- `identity/REDUCTION_PARTITION_CONFLICT`;
- `configuration/REDUCTION_FANIN_INVALID`;
- `configuration/REDUCTION_LEVEL_LIMIT`.

There is no implicit “best effort” reduction. A future partial policy must be a
new explicit canonical contract.

Cancellation prevents future partitions/levels, cancels pending/running
partition nodes through the run epoch, and leaves successful immutable outputs
unpublished except as unreferenced artifacts.

## Projection contract

Expose:

```text
sourceItems
fanIn
currentLevel
levelsMaterialized
partitionsTotal
partitionsReady
partitionsRunning
partitionsSucceeded
partitionsFailed
rootReady
status
```

Blocked reasons include `reduction-source`, `reduction-level`, and ordinary
resource/retry/implementation reasons for executable partition nodes.

## Migration and compatibility

Add tables and indexes without rewriting completed Slice 1–6 data. Canonical
plan schema evolution must preserve old digest interpretation; use an explicit
new schema version if adding reduction fields changes decoding semantics.
Existing static workflows have zero reductions.

## Reference fixture

Generate a deterministic document set and lazily map one word-count task per
document. Each task emits a sorted map of token to count. Reducers merge at
most `fanIn` sorted maps and emit a sorted canonical result. A direct trusted Go
count provides the oracle.

The fixture must use enough documents to create at least three levels under a
small test fan-in and enough real items to exercise production fan-in.

## Test matrix

### Canonical and authoring

- direct Go/JavaScript equality and exact goldens;
- callback executes once with an opaque partition ref;
- wrong source/item/root schemas rejected;
- fan-in and level ceilings validated;
- reduction key collisions rejected.

### Identity and store

- exact partitions for boundary counts: 0, 1, `fanIn`, `fanIn+1`;
- stable identities across repeated planning;
- crash before/after partition-page commit;
- two coordinators create one level;
- conflict fails closed;
- migration from Slice 6 database.

### Runtime and recovery

- root equals direct oracle;
- root bytes/digest equal at concurrency 1, 2, and higher;
- randomized child completion order changes no membership or digest;
- restart during at least two levels;
- one retrying partition does not rerun successful siblings;
- malformed partition blocks next level and fails with stable code;
- no runtime receives more than `fanIn` refs;
- cancellation publishes no authoritative root;
- source text absent from SQLite/WAL/events.

## Implementation sequence

1. Freeze reduction IR/plan/DTS and fan-in policy.
2. Implement canonical partition/root manifest codecs and identity goldens.
3. Add schema/migrations and coordinator transaction tests.
4. Add dynamic partition node materialization through the existing store.
5. Add level completion and root publication.
6. Add projections and blocked reasons.
7. Implement real word-count fixture and crash/concurrency matrix.
8. Update help, diary, changelog, relations, and generated artifacts.
9. Run race, lint, TypeScript, full validation, privacy, and diff checks.

## Acceptance criteria

Slice 7 is complete only when:

- all reducer inputs are bounded by compiled fan-in;
- partition membership and root digest are completion-order independent;
- successful partitions survive restart and sibling failure;
- exact source cardinality reaches the root with no missing/duplicate members;
- malformed shards cannot be published;
- cancellation and stale completion are fenced;
- projections match authoritative rows after reopen;
- control persistence contains refs rather than source/aggregate payloads;
- Slices 1–6 and existing databases remain valid;
- all focused, race, lint, TypeScript, generated, repository, docmgr, privacy,
  and whitespace checks pass with fresh evidence.

## Alternatives rejected

### One reducer with every child as dependency

Rejected because dependency, manifest, runtime memory, and retry cost are
unbounded.

### Partition by completion order

Rejected because concurrency changes tree membership and root digest.

### Mutable aggregate row updated by every child

Rejected because update order, lock contention, crash replay, and schema/privacy
become difficult to prove.

### Recompute the entire tree after restart

Rejected because immutable successful partition outputs already provide safe
reusable checkpoints.

## References

- [All-slice intern guide](08-workflow-v3-slices-1-through-12-intern-architecture-and-analysis-guide.md)
- [Slice 6 lazy maps](03-slice-6-lazy-maps-deterministic-durable-scale-out.md)
- [Primary architecture](01-durable-dataflow-workflow-v3-and-modern-scripting-architecture.md)
- [Diary](../reference/01-investigation-diary.md)
