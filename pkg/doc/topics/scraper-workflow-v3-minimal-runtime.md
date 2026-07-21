---
Title: Workflow V3 Runtime Slices 1–7
Slug: scraper-workflow-v3-minimal-runtime
Short: "Explains executable workflow-v3 file, HTTP, dispatcher, database, lazy-map, and reduction slices with exact capabilities and durable privacy boundaries."
Topics:
- scraper
- runtime
- workflows
- javascript
- workers
Commands:
- worker
- engine
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

Workflow v3 has seven executable vertical slices for trusted first-party
JavaScript tasks: linear file processing, typed authoring, allowlisted HTTP,
work-conserving resource dispatch, idempotent database synchronization,
deterministic lazy maps, and bounded reduction trees. It remains intentionally
separate from the existing v2 site and submission runtime while later
upgrade/budget capabilities are added.

## What runs today

A CommonJS authoring script can import `workflow` and a descriptor-only task
module, then use:

- `workflow.define(name, callback)`;
- `plan.input(name, {schema})`;
- `plan.inputSet(name, {itemSchema, manifestSchema})`;
- `plan.task(nodeKey, taskDescriptor, callback?)`;
- `plan.map(name, set, itemCallback, mapPolicyCallback?)`;
- `plan.reduce(name, set, partitionCallback, reducePolicyCallback?)`;
- `job.after(otherJob)`;
- `job.output(port)`;
- `plan.output(name, value)` and `plan.outputSet(name, set)`;
- `workflow.validate`, `workflow.digest`, `workflow.toIR`, and
  `workflow.compile`.

The JavaScript builder produces the same canonical Go IR as direct Go
construction. Compilation pins task kind, task version, bundle digest,
entrypoint, task ABI, schemas, declared modules, resource class, retry policy,
catalog digest, IR digest, and plan digest.

## Execution boundary

Workers seal exact module aliases into the same generation advertisement as
task implementations. The implemented profiles are:

- `fs:input`, a read-only mount containing only bound artifacts;
- `fetch:public`, an origin-allowlisted HTTP client with bounded timeout/body,
  credential sources disabled, URL/header credentials rejected, wildcard or
  empty allowlists rejected at worker boot, and redirect policy enforcement;
- `db:sync`, a Go-preconfigured database handle that rejects JavaScript
  `configure()`.

Each attempt receives:

- a fresh Goja runtime and CommonJS module cache;
- only `workflow/task` and modules declared by the pinned task;
- a read-only temporary filesystem containing only bound input artifacts;
- a context with immutable input refs, identity, cancellation checkpoint, and
  validating output writers.

Source bytes live in an external content-addressed artifact store. SQLite stores
only schemas, digests, media types, sizes, bounded locators, plan identities,
lease state, attempt outcomes, and redacted events.

## Durable behavior

The v3 SQLite store uses `(run_id, node_key)` identities and append-only attempt
numbers. Lease expiry creates a `lease_lost` outcome and a later attempt.
Completion requires the current lease token and cancellation epoch. A stale,
expired, or canceled attempt cannot publish output refs.

Workers match the complete implementation identity, declared aliases,
resource class, and retry policy before leasing. A worker with the same task
kind/version but different bundle bytes, entrypoint, ABI, or module profile
cannot claim the node.

## Work-conserving dispatch

`workflowv3runtime.Dispatcher` leases one eligible node at a time until every
configured resource class is full. Each completion wakes immediate refill; a
released HTTP slot does not wait for an unrelated slow database or network
attempt. Capacity is checked transactionally from running nodes in SQLite, and
fairness counters are scoped by `(run_id, resource_class)`.

Queue projections derive ready count, active counts by resource, and bounded
blocked reasons (`dependency`, `retry-backoff`, `resource-capacity`, and
`implementation-unavailable`) from authoritative rows. Retryable failures leave
an immutable failed attempt and return the node to pending with a durable
`ready_at` deadline.

## Lazy map behavior

A typed set input is one immutable ordered manifest artifact. Each manifest
entry has a canonical unique item key and compact artifact ref. JavaScript map
callbacks execute once during authoring with an opaque symbolic item; callback
code is never persisted or replayed during expansion.

The store records one `v3_expansions` row per declared map. This is first-class
Slice 6 control state, not a backwards-compatibility table. It owns the source
ref, bounded expansion policy, cursor, materialized/terminal counts, status,
and published output ref. Older static runs simply have no expansion rows.

Expansion inserts at most one configured page in an atomic transaction. Child
keys hash the map key, source-manifest digest, and canonical item key. Dynamic
children then use the ordinary node, lease, attempt, resource, retry, fencing,
and output path. Materialized-ahead backpressure prevents the graph from being
expanded without bound. Queue projections expose source, materialized,
terminal, and execution backlog plus `map-backpressure`.

After every child succeeds, the engine constructs one ordered immutable output
manifest from validated child refs. The run cannot succeed until that manifest
artifact reference reaches durable `published` state. Empty maps publish an
empty manifest without acquiring a task lease.

## Bounded reduction behavior

A reduction consumes a typed set through immutable partition artifacts. The
compiled plan pins one exact homogeneous reducer, maximum fan-in, maximum
levels, resource class, modules, and retry policy. Partition identity covers
the original source digest, reduction key, level, ordinal, and ordered member
keys/digests; completion timing does not affect membership or root identity.

The engine materializes each member artifact only inside the reducer's private
lease workspace and exposes read-only member paths through typed
`workflow/task` input metadata. Workflow SQLite stores partition and output
refs, counts, identities, and state—not member payloads.

Completed partitions survive restart. When one level finishes, its outputs are
ordered by partition ordinal and become the next bounded level. The run cannot
succeed until one validated root ref is published. A single source item is its
identity root; an empty source fails without a worker lease.

The real word-count fixture reduces 257 map outputs through partition counts
33 → 5 → 1 with fan-in eight, closes/reopens after a level-zero partition
succeeds, and finishes with 296 total attempts. Capacity 1 and capacity 4
produce the same root digest. Malformed-shard failure remains isolated from an
unrelated successful run.

## HTTP and database behavior

The HTTP fixture snapshots at most eight explicitly supplied article URLs.
Transport, rate-limit, server-status, and validation failures become stable
redacted codes. Origin policy applies to the initial request and every redirect;
response headers, URL credentials, and raw failure text never enter workflow
rows or events. Snapshot outputs use stable list indexes rather than echoing
request URLs, so query credentials are not copied into the output artifact.

The database fixture uses a stable SHA-256 operation key derived from
`(run_id,node_key)`. Side effects and the operation marker commit in one target
transaction. The crash test discards the task result after commit, closes the
workflow store with attempt one still running, waits for lease expiry, and then
reopens. Attempt one becomes `lease_lost`; the fresh second runtime observes the
same operation key and cannot apply the logical write again. Domain rows remain
in the target database; workflow SQLite stores only artifact refs and redacted
attempt evidence.

## Validation

Run the focused implementation suites with:

```text
GOWORK=off go test ./pkg/workflowv3 \
  ./pkg/gojamodules/workflow \
  ./pkg/workflowv3runtime \
  ./pkg/workflowv3sqlite -count=1

GOWORK=off go test -race \
  ./pkg/workflowv3runtime \
  ./pkg/workflowv3sqlite -count=1

cd web && pnpm exec tsc --noEmit --skipLibCheck \
  ../pkg/gojamodules/workflow/testdata/workflow.d.ts \
  ../pkg/workflowv3runtime/testdata/workflow-task.d.ts
```

The focused tests cover the 12,000-row file workflow, real local HTTP and
SQLite target servers, a 1,807-item JavaScript lazy map, and a 257-item
multi-level JavaScript reduction. They prove typed
retry, allowlist and redirect denial,
response limits, in-flight cancellation, independent resource refill,
per-resource fairness, blocked projections, database reconfiguration denial,
post-commit crash recovery, deterministic paged expansion, backpressure,
ordered map publication, failure isolation, reopen, and SQLite/WAL/SHM canary
privacy. The 500-row database fixture persisted 90,112 workflow bytes for
499,554 source bytes (18.04%). The normal lazy-map evidence processes 1,807
items across restart and records 7,561,185 source bytes versus approximately
5,353,472 workflow SQLite bytes (70.80%); source-private fields are absent from
control persistence and the published output manifest.

## Source map

- `pkg/workflowv3` — canonical contracts, compiler, artifacts, bundles, and
  registry.
- `pkg/gojamodules/workflow` — safe authoring module and TypeScript declaration.
- `pkg/workflowv3runtime` — fresh-runtime task runner, exact host modules,
  deterministic engine hook, long-lived dispatcher, and exact task-ABI DTS.
- `pkg/workflowv3sqlite` — compact durable store, resource admission,
  projections, retries, and fencing.
- `pkg/testfixtures/workflowv3linear` — linear file workflow and bundle.
- `pkg/testfixtures/workflowv3http` — bounded public HTTP snapshot workflow.
- `pkg/testfixtures/workflowv3database` — idempotent database synchronization
  workflow.
- `pkg/testfixtures/workflowv3map` — real 1,807-item lazy-map workflow and
  trusted JavaScript bundle.
- `pkg/testfixtures/workflowv3reduce` — real bounded word-count map/reduction
  workflow and trusted JavaScript bundle.

Later slices add rolling registry generations, budgets, gates, and stronger
process isolation. V3 does not translate or silently accept v2 raw operations.
