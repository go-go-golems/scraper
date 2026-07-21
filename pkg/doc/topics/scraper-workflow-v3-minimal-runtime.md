---
Title: Workflow V3 Runtime Slices 1–5
Slug: scraper-workflow-v3-minimal-runtime
Short: "Explains executable workflow-v3 file, HTTP, dispatcher, and database slices with exact capabilities and durable privacy boundaries."
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

Workflow v3 has five executable vertical slices for trusted first-party
JavaScript tasks: linear file processing, typed authoring, allowlisted HTTP,
work-conserving resource dispatch, and idempotent database synchronization. It
remains intentionally separate from the existing v2 site and submission
runtime while later map/reduction/budget capabilities are added.

## What runs today

A CommonJS authoring script can import `workflow` and a descriptor-only task
module, then use:

- `workflow.define(name, callback)`;
- `plan.input(name, {schema})`;
- `plan.task(nodeKey, taskDescriptor, callback?)`;
- `job.after(otherJob)`;
- `job.output(port)`;
- `plan.output(name, value)`;
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
  credential sources disabled, and redirect policy enforcement;
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

## HTTP and database behavior

The HTTP fixture snapshots at most eight explicitly supplied article URLs.
Transport, rate-limit, server-status, and validation failures become stable
redacted codes. Origin policy applies to the initial request and every redirect;
response headers, URL credentials, and raw failure text never enter workflow
rows or events.

The database fixture uses a stable SHA-256 operation key derived from
`(run_id,node_key)`. Side effects and the operation marker commit in one target
transaction. A crash after that commit creates a retry attempt, but the fresh
runtime observes the same operation key and cannot apply the logical write a
second time. Domain rows remain in the target database; workflow SQLite stores
only artifact refs and redacted attempt evidence.

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
```

The focused tests cover the 12,000-row file workflow plus real local HTTP and
SQLite target servers. They prove typed retry, allowlist and redirect denial,
response limits, in-flight cancellation, independent resource refill,
per-resource fairness, blocked projections, database reconfiguration denial,
post-commit crash recovery, failure isolation, reopen, and SQLite/WAL/SHM
canary privacy. The 500-row database fixture persisted 90,112 workflow bytes
for 499,554 source bytes (18.04%).

## Source map

- `pkg/workflowv3` — canonical contracts, compiler, artifacts, bundles, and
  registry.
- `pkg/gojamodules/workflow` — safe authoring module and TypeScript declaration.
- `pkg/workflowv3runtime` — fresh-runtime task runner, exact host modules,
  deterministic engine hook, and long-lived dispatcher.
- `pkg/workflowv3sqlite` — compact durable store, resource admission,
  projections, retries, and fencing.
- `pkg/testfixtures/workflowv3linear` — linear file workflow and bundle.
- `pkg/testfixtures/workflowv3http` — bounded public HTTP snapshot workflow.
- `pkg/testfixtures/workflowv3database` — idempotent database synchronization
  workflow.

Later slices add lazy maps, bounded reductions, budgets, gates, rolling
registry generations, and stronger process isolation. V3 does not translate or
silently accept v2 raw operations.
