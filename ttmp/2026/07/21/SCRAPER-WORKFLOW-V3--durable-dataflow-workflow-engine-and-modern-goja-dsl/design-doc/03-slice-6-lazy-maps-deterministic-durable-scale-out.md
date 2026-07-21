---
Title: Slice 6 Lazy Maps - Deterministic Durable Scale-Out
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
      Note: Go-backed map and set-ref authoring handles
    - Path: repo://pkg/workflowv3/compiler.go
      Note: Authoritative map template and policy validation
    - Path: repo://pkg/workflowv3/types.go
      Note: |-
        Canonical IR and plan types extended by map declarations and typed sets
        Canonical map and typed set contracts will extend this model
    - Path: repo://pkg/workflowv3runtime/dispatcher.go
      Note: Interleaving expansion and task dispatch under backpressure
    - Path: repo://pkg/workflowv3sqlite/schema.sql
      Note: Expansion cursors pages and dynamic child persistence
    - Path: repo://pkg/workflowv3sqlite/store.go
      Note: |-
        Atomic page materialization cancellation and restart transitions
        Atomic paged expansion and restart transitions belong in this store
ExternalSources: []
Summary: Implementation contract for deterministic paged map expansion that materializes large dynamic node sets without persisting source payloads or requiring one large transaction.
LastUpdated: 2026-07-21T20:30:00-04:00
WhatFor: Freeze Slice 6 APIs identities state transitions transaction boundaries restart behavior and acceptance evidence before implementation.
WhenToUse: Read before adding set references map authoring dynamic nodes expansion cursors or map progress to Workflow V3.
---


# Slice 6 Lazy Maps - Deterministic Durable Scale-Out

## Executive summary

Slice 6 adds dynamic cardinality to Workflow V3. A compiled plan declares one
map over an immutable typed set, while the durable control plane materializes
child nodes in bounded pages. Child identity is derived from plan identity,
source manifest identity, and canonical item key. Expansion state is committed
atomically with each page, so restart cannot omit or duplicate children.

The slice is complete when at least 1,807 real items expand and execute across
multiple crashes with an identical ordered item-key set and output digest,
while workflow SQLite remains compact and source-free.

## Problem statement

Current `WorkflowPlan.Nodes` is static. Constructing thousands of nodes before
submission creates three problems:

1. submission and plan JSON grow with item count;
2. a single transaction must insert the entire graph;
3. source values are likely to be copied into per-node bindings.

A dynamic callback at runtime is not acceptable either. Re-executing arbitrary
JavaScript after restart can produce different children and gives control-plane
authority to task code.

## Scope

Slice 6 includes typed set refs, canonical map declarations, immutable item
manifests, deterministic child keys, bounded expansion pages, durable cursors,
backpressure, cancellation, projections, restart, and privacy evidence.

It does not include reduction trees, rolling registry upgrades, budgets, gates,
or untrusted execution. Those later slices consume the map contract.

## Canonical model

Add a `SetRef` that identifies an ordered immutable manifest, not an in-memory
JavaScript array:

```go
type SetRef struct {
    Source string  `json:"source"` // input | node
    Name string    `json:"name,omitempty"`
    NodeKey NodeKey `json:"nodeKey,omitempty"`
    Port string    `json:"port,omitempty"`
    ItemSchema string `json:"itemSchema"`
    ManifestSchema string `json:"manifestSchema"`
}

type MapPolicy struct {
    PageSize int `json:"pageSize"`
    MaxItems int `json:"maxItems"`
    MaxMaterializedAhead int `json:"maxMaterializedAhead"`
}

type IRMap struct {
    Key string `json:"key"`
    Source SetRef `json:"source"`
    ItemTask IRNodeTemplate `json:"itemTask"`
    Policy MapPolicy `json:"policy"`
}
```

The plan records requested and effective policy if host compilation clamps a
request downward. The item task becomes a `PlanNodeTemplate` with exact bundle,
entrypoint, ABI, modules, resource, retry, input/output schemas, and one
symbolic item binding. No callback survives compilation.

Map keys share the workflow namespace with static nodes, reductions, and gates.
Duplicate keys are rejected.

## JavaScript authoring contract

The target API is:

```typescript
interface PlanBuilder {
  map<I, O>(
    name: string,
    source: SetRef<I>,
    task: (item: ItemRef<I>) => WorkflowTask<I, O>,
    configure?: (map: MapBuilder) => void,
  ): SetRef<O>;
}

interface MapBuilder {
  pageSize(value: number): this;
  maxItems(value: number): this;
  maxMaterializedAhead(value: number): this;
}
```

The callback runs exactly once during `workflow.define`. The Go authoring
module creates an opaque symbolic `ItemRef`, invokes the callback, resolves the
returned descriptor, and records a template. Calling the callback for every
runtime item is prohibited.

## Manifest contract

Use a versioned artifact schema such as
`scraper-workflow-item-manifest/v1`. Each entry contains:

```json
{
  "key": "customer-000123",
  "value": {
    "schema": "customer-record/v1",
    "digest": "sha256:...",
    "mediaType": "application/json",
    "size": 271,
    "locator": "cas://sha256/..."
  }
}
```

The manifest has canonical item-key order and unique keys. Its artifact digest
binds cardinality, order, and refs. The expander validates schema, maximum
count, duplicate keys, ordering, and each bounded ref before inserting any
children. Item payloads remain external.

The first implementation may require the manifest to expose indexed pages so
expansion does not read the complete body repeatedly. Any index is itself an
immutable verified artifact.

## Deterministic identity

Freeze child identity as a canonical digest envelope:

```json
{
  "mapKey": "embed-documents",
  "sourceDigest": "sha256:...",
  "itemKey": "customer-000123"
}
```

The visible node key uses a bounded readable prefix plus the full digest, or a
fully encoded digest. Encoding rules, collision handling, and maximum length
must have exact goldens. Never derive identity from page number alone,
expansion time, worker, random UUID, database row ID, or completion order.

The stable operation key for a child remains derived from its final
`(runId,nodeKey)`, so side-effecting mapped tasks inherit Slice 5 idempotency.

## Durable schema

Add tables conceptually equivalent to:

```sql
CREATE TABLE v3_expansions (
  run_id TEXT NOT NULL,
  map_key TEXT NOT NULL,
  source_schema TEXT NOT NULL,
  source_digest TEXT NOT NULL,
  source_locator TEXT NOT NULL,
  total_items INTEGER NOT NULL,
  next_index INTEGER NOT NULL DEFAULT 0,
  materialized_items INTEGER NOT NULL DEFAULT 0,
  terminal_items INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL CHECK (status IN
    ('pending','expanding','expanded','succeeded','failed','canceled')),
  updated_at TEXT NOT NULL,
  PRIMARY KEY (run_id,map_key),
  FOREIGN KEY (run_id) REFERENCES v3_runs(run_id) ON DELETE CASCADE
);

CREATE TABLE v3_expansion_pages (
  run_id TEXT NOT NULL,
  map_key TEXT NOT NULL,
  page_no INTEGER NOT NULL,
  first_index INTEGER NOT NULL,
  item_count INTEGER NOT NULL,
  page_digest TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (run_id,map_key,page_no),
  FOREIGN KEY (run_id,map_key)
    REFERENCES v3_expansions(run_id,map_key) ON DELETE CASCADE
);

CREATE TABLE v3_map_items (
  run_id TEXT NOT NULL,
  map_key TEXT NOT NULL,
  item_key TEXT NOT NULL,
  item_index INTEGER NOT NULL,
  node_key TEXT NOT NULL,
  input_schema TEXT NOT NULL,
  input_digest TEXT NOT NULL,
  input_media_type TEXT NOT NULL,
  input_size_bytes INTEGER NOT NULL,
  input_locator TEXT NOT NULL,
  PRIMARY KEY (run_id,map_key,item_key),
  UNIQUE (run_id,map_key,item_index),
  UNIQUE (run_id,node_key)
);
```

`v3_map_items` contains compact refs only. Dynamic children continue to use
`v3_nodes`, `v3_dependencies`, `v3_attempts`, and `v3_node_outputs` so there is
one execution path.

## Expansion transaction

`ExpandNextPage` performs:

1. begin immediate transaction;
2. read run and expansion state;
3. reject canceled/terminal run;
4. verify source digest still matches the compiled/run input;
5. enforce `materialized - terminal < maxMaterializedAhead`;
6. read the next bounded manifest page outside the transaction if needed, then
   recheck cursor/version after beginning the write transaction;
7. validate all entries and precompute deterministic node rows;
8. insert page record, map-item rows, dynamic nodes, and dependencies;
9. advance `next_index` and `materialized_items`;
10. set `expanded` when cursor equals total;
11. append one bounded `map.page-materialized` event;
12. commit.

The implementation must not hold the SQLite write transaction while fetching a
remote artifact. Use an optimistic expected cursor/source digest and retry the
small transaction if another expander won.

Unique keys make replay safe. A conflicting row with different identity is an
`identity/MAP_ITEM_CONFLICT` failure, not silently ignored.

## Scheduling and backpressure

Expansion is control-plane work and consumes no task lease. The dispatcher
alternates bounded expansion actions with normal leasing. It should execute
ready children before materializing an unbounded backlog.

Use configurable limits:

- pages per maintenance cycle;
- page size;
- maximum materialized-ahead count;
- maximum total items;
- maximum manifest/page bytes.

Multiple runs remain fair. One million-item map cannot monopolize every
maintenance cycle while another run has a ten-item map.

## Completion accounting

A child terminal transition updates map terminal counters in the same
transaction, or counters are derived with indexed queries if that proves fast
enough. Avoid eventually consistent counters that can disagree after crash.

Map state becomes `succeeded` only when:

```text
next_index == total_items
materialized_items == total_items
terminal_items == total_items
all child outputs satisfy the declared output schema
no terminal disallowed child failure exists
```

The map output is a canonical output-manifest artifact sorted by item key. It
contains child output refs, not payloads. Publishing that manifest must be
idempotent and fenced.

## Failure and cancellation

Stable failures include:

- `validation/MAP_MANIFEST_SCHEMA`;
- `validation/MAP_CARDINALITY`;
- `validation/MAP_DUPLICATE_ITEM_KEY`;
- `identity/MAP_ITEM_CONFLICT`;
- `configuration/MAP_POLICY_INVALID`;
- `internal/MAP_PAGE_COMMIT` only for genuinely unclassified store failures.

A failed child follows its own retry policy. A terminal child failure fails the
map/run unless a future explicitly compiled partial-result policy exists.
There is no implicit skip.

Cancellation increments the existing run epoch, prevents new pages, cancels
running children, marks pending children canceled, and marks expansion
canceled. A page racing cancellation commits either before the cancellation
transaction and is then canceled, or loses its expected running-state check.
No page may appear after cancellation commit.

## Projection contract

Expose per map:

```text
sourceTotal
nextIndex
materialized
active
succeeded
failed
canceled
backlogToMaterialize
backlogToExecute
status
oldestReadyAge
```

Queue blocked reason adds `map-backpressure` only when the map has more source
items but is intentionally not expanding because materialized-ahead is full.
This reason is derived from authoritative rows.

## Migration and compatibility

Migration is additive. Existing Slice 1–5 plans decode unchanged because map
arrays are absent/empty under a schema-compatible canonical extension only if
the canonical version rules explicitly permit it. If adding fields changes
public canonical bytes, introduce a reviewed plan schema revision rather than
silently changing old digests.

Existing `v3_nodes` rows remain valid. Dynamic metadata is optional and joined
through `v3_map_items`; do not overload static bindings with arbitrary item
JSON.

## Test matrix

### Canonical and authoring

- direct Go and JavaScript map IR equality;
- exact IR/plan/DTS goldens;
- callback executes once;
- forged/cross-runtime `SetRef` and `ItemRef` rejected;
- invalid page/max/backpressure policies rejected;
- item/output schema mismatch rejected.

### Store and concurrency

- page transaction inserts exact bounded rows;
- crash before commit changes no cursor;
- reopen after commit starts at next cursor;
- two connections expanding same cursor create one page;
- conflicting deterministic identity fails closed;
- cancellation/page race has no post-cancel materialization;
- migration from completed Slice 5 database succeeds.

### End to end

- 1,807-item real manifest and task bundle;
- restart before, during, and after multiple pages;
- compare item-key and output-digest sets across concurrency 1 and higher;
- retry selected children without rematerialization;
- no missing or duplicate item keys;
- source/privacy canaries absent from SQLite/WAL/events;
- persisted control bytes remain bounded relative to source;
- projection counts equal direct durable facts.

## Implementation sequence

1. Freeze `SetRef`, map IR, plan template, canonical encoding, and DTS.
2. Add manifest validator and child-key golden tests.
3. Add additive tables and migration tests.
4. Implement one-page optimistic transaction and concurrency tests.
5. Add dispatcher maintenance/backpressure integration.
6. Add map completion manifest and projection.
7. Add real 1,807-item fixture with restart/cancel/privacy evidence.
8. Update help, diary, source map, changelog, and generated files.
9. Run focused/race/full repository validation.

## Acceptance criteria

Slice 6 is complete only when all of the following are fresh and reproducible:

- JavaScript and direct Go compile to identical canonical map plans;
- expansion is paged, bounded, deterministic, and source-free;
- 1,807 real items survive repeated restart with exact cardinality/order;
- concurrent expanders do not duplicate or omit children;
- cancellation fences future pages and stale child completion;
- output manifest digest is concurrency/completion-order independent;
- projections report authoritative map progress and backpressure;
- old databases and Slices 1–5 behavior remain valid;
- privacy scans, race tests, lint, TypeScript, generated checks,
  `make validate`, docmgr validation, and `git diff --check` pass.

## Alternatives rejected

### Expand the entire graph at submission

Rejected because graph size and transaction size scale directly with source
cardinality and encourage source duplication in plan/node rows.

### Execute JavaScript callback once per item after restart

Rejected because code execution would determine durable graph identity and
could produce nondeterministic children.

### Use random child IDs with a uniqueness table

Rejected because replay and cross-run comparison become dependent on the first
worker that expanded the page.

### Persist source items inline in child bindings

Rejected because it recreates the TTC storage/privacy failure.

## References

- [All-slice intern guide](08-workflow-v3-slices-1-through-12-intern-architecture-and-analysis-guide.md)
- [Primary architecture](01-durable-dataflow-workflow-v3-and-modern-scripting-architecture.md)
- [Cookbook](../reference/03-workflow-v3-javascript-cookbook-and-execution-atlas.md)
- [Diary](../reference/01-investigation-diary.md)
