---
Title: Workflow V3 Slices 1 Through 12 - Intern Architecture and Analysis Guide
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
    - Path: repo://pkg/workflowv3
      Note: Canonical workflow IR plans bundles registries failures and artifact references
    - Path: repo://pkg/gojamodules/workflow
      Note: Safe authoring module and exact TypeScript contract
    - Path: repo://pkg/workflowv3runtime
      Note: Lease-scoped task execution modules engine and dispatcher
    - Path: repo://pkg/workflowv3sqlite
      Note: Durable control-plane schema transactions and projections
    - Path: repo://pkg/testfixtures/workflowv3linear
      Note: Executable Slice 1 and 2 file workflow
    - Path: repo://pkg/testfixtures/workflowv3http
      Note: Executable Slice 3 HTTP workflow
    - Path: repo://pkg/testfixtures/workflowv3database
      Note: Executable Slice 5 database workflow
ExternalSources: []
Summary: Intern-oriented architectural guide to all twelve Workflow V3 vertical slices, including implemented evidence for Slices 1-9 and implementation contracts for Slices 10-12.
LastUpdated: 2026-07-21T20:15:00-04:00
WhatFor: Understand why each Workflow V3 slice exists, which architectural boundary it proves, how it changes durable state, and what evidence is required before advancing.
WhenToUse: Read before implementing, reviewing, testing, or operating any Workflow V3 slice, and before changing canonical IR, SQLite state, worker registries, task capabilities, or RAG integration.
---

# Workflow V3 Slices 1 Through 12 - Intern Architecture and Analysis Guide

## Executive summary

Workflow V3 is being delivered as twelve vertical slices. Each slice introduces
one new source of production complexity and proves it through an executable
workflow. This order is deliberate. The implementation does not begin with a
large abstract scheduler and then add workloads later. It starts with the
smallest durable path and adds external I/O, concurrency, side effects, graph
scale, aggregation, upgrades, operational controls, waiting, isolation, and
finally an expensive RAG workload.

The sequence is:

1. **Linear file transform: durable data and identity are real.**
2. **Minimal authoring DSL: JavaScript composition is real.**
3. **HTTP snapshot: external work is real.**
4. **Dispatcher: concurrency is real.**
5. **Database sync: side effects are real.**
6. **Lazy maps: scale-out is real.**
7. **Reductions: scale-in is real.**
8. **Registry generations: upgrades are real.**
9. **Budgets and projections: operations are real.**
10. **Approval gates: waiting is real.**
11. **Process isolation: broader trust is viable.**
12. **RAG/TTC: an expensive production workload is safe.**

Slices 1–10 are implemented and validated. Slices 11–12 are target contracts.
This distinction is important: a design section explains what must become true;
it is not evidence that the behavior already exists.

## 1. How to reason about the slices

### 1.1 Every slice crosses one new boundary

The slices are ordered by the kind of failure they introduce.

| Slice | New boundary | Principal failure question |
|---|---|---|
| 1 | Durable identity and artifact references | Can work restart without duplicating source data or changing code identity? |
| 2 | JavaScript authoring | Can scripts express intent without becoming execution authority? |
| 3 | External HTTP | Can remote failures, credentials, redirects, and cancellation remain bounded? |
| 4 | Concurrent dispatch | Can capacity remain full without oversubscription or starvation? |
| 5 | Database side effects | Can a crash after commit avoid duplicating the logical write? |
| 6 | Dynamic cardinality | Can thousands of children be materialized deterministically and incrementally? |
| 7 | Aggregation | Can thousands of results become one result with bounded fan-in and stable order? |
| 8 | Worker upgrades | Can code change while old attempts and plans retain exact implementation identity? |
| 9 | Operational limits | Can admission, usage, progress, and blocked state remain transactionally truthful? |
| 10 | Long-lived waiting | Can a workflow pause for a decision without occupying a lease or resource slot? |
| 11 | Broader publishers | Can less-trusted code execute outside the scraper process under stronger limits? |
| 12 | Expensive production work | Can all contracts hold together under real provider cost, latency, restart, and publication? |

A slice is complete only when the new boundary is represented in canonical
plans, durable state, runtime behavior, failure handling, projections,
documentation, and tests. Adding a method to the JavaScript DSL is not enough.
Adding a table without restart tests is not enough.

### 1.2 The shared execution pipeline

All slices extend one pipeline:

```text
JavaScript or direct Go intent
        ↓
Go-owned canonical WorkflowIR
        ↓
strict validation against a Task Catalog
        ↓
immutable WorkflowPlan with exact implementation identity
        ↓
durable run, nodes, dependencies, attempts, leases, refs, and events
        ↓
registry and resource admission
        ↓
fresh lease-scoped execution
        ↓
validated external ArtifactRef outputs
        ↓
authoritative store-derived projections
```

JavaScript never becomes the source of truth for durable state. SQLite never
becomes the payload store. A worker never chooses a semantically similar task
when the exact implementation is absent.

### 1.3 The four data categories

An intern should classify every new field before adding it.

1. **Canonical intent** belongs in IR or the compiled plan. Examples are task
   kind, schema, dependency, retry policy, resource class, map page size, and
   reduction fan-in.
2. **Mutable durable control state** belongs in SQLite. Examples are node
   status, expansion cursor, attempt number, lease token, reservation, and gate
   decision state.
3. **Bulk or sensitive data** belongs in an external artifact/domain store and
   enters the workflow database only as a bounded `ArtifactRef`.
4. **Host policy and secrets** remain in trusted worker/operator
   configuration. Allowed origins, database handles, credentials, signing
   keys, and container policy do not enter plans or events.

If a source record, prompt, response body, vector, SQL parameter, credential,
or database row is about to enter `v3_*` control tables, stop and redesign the
boundary.

### 1.4 The shared correctness invariants

Every slice preserves these invariants:

- node identity is scoped by `(run_id, node_key)`;
- attempt identity is `(run_id, node_key, attempt_no)`;
- every attempt is immutable after it reaches a terminal state;
- completion requires the current lease token and cancellation epoch;
- retries create new attempts and durable `ready_at` deadlines;
- plans pin task kind, version, bundle digest, entrypoint, ABI, modules,
  resource class, and retry policy;
- outputs are accepted only after schema, digest, size, and port validation;
- events contain compact allowlisted facts, not arbitrary exceptions or data;
- restart reads committed state and does not replay authoring callbacks;
- concurrent completion order never determines published logical order;
- v2 behavior and existing databases remain intact through additive migration.

## 2. Slice 1 — linear file transform: durable data and identity are real

### 2.1 Purpose

Slice 1 establishes the minimum complete durable path. It intentionally avoids
network calls, database writes, dynamic maps, and human waiting. This makes
identity, persistence, leasing, artifact handling, and restart behavior visible
without unrelated variables.

The real fixture is `pkg/testfixtures/workflowv3linear`. It accepts a JSONL
artifact, normalizes customer records, validates the resulting dataset, and
publishes a typed output reference.

### 2.2 Architectural contribution

Slice 1 introduces:

- `WorkflowIR`, `WorkflowPlan`, `PlanNode`, `ValueRef`, and `ArtifactRef`;
- strict canonical decoding and deterministic digests;
- exact task implementation identity;
- immutable JavaScript bundles and sealed registries;
- compact v3 SQLite tables;
- append-only attempts and transactional leases;
- cancellation epochs and stale-completion fencing;
- a content-addressed external artifact store;
- fresh Goja runtimes and read-only attempt workspaces;
- host-side input and output schema validation.

The crucial storage choice is that source bytes are not stored in the workflow
database. `v3_run_inputs` records schema, digest, media type, size, and locator.
The runtime verifies and materializes the artifact only for the current
attempt.

### 2.3 State transition

```text
run submitted
  → normalize pending
  → normalize leased / attempt 1 running
  → normalize succeeded / normalized ArtifactRef committed
  → validate becomes ready
  → process restart
  → validate leased / attempt 1 running
  → validate succeeded / final ArtifactRef committed
  → run succeeded
```

A lease expiry instead creates a terminal `lease_lost` attempt and returns the
node to pending. A stale worker cannot commit after cancellation or after a
newer lease replaces its token.

### 2.4 What the evidence proves

The integration test transforms 12,000 real rows, closes and reopens the store
between tasks, then reopens the final result. It scans SQLite main/WAL/SHM for
source and secret canaries. The recorded evidence is 1,656,000 source bytes
versus 73,728 workflow SQLite bytes, a ratio of 4.45%.

Exact negative tests reject wrong bundle bytes, entrypoint, ABI, schema,
module profile, stale lease, and forged output. An eight-contender lease race
produces one winner.

### 2.5 Review guidance

Start at `engine_integration_test.go`, then follow `CreateRun`, `LeaseNext`,
`RunTask`, and `Complete`. Confirm that artifact writes can occur before
completion only because they are immutable and unreferenced until the fenced
transaction commits.

## 3. Slice 2 — minimal authoring DSL: JavaScript composition is real

### 3.1 Purpose

Slice 2 proves that a modern JavaScript API can author the same plan as direct
Go construction without creating a second compiler or a privileged scripting
context.

The author imports `require("workflow")` and a descriptor-only task module.
`define`, `input<T>`, `task`, `after`, `output`, `validate`, `digest`, `toIR`,
and `compile` are backed by Go-owned builders.

### 3.2 Architectural contribution

The DSL uses opaque handles associated with the current Goja runtime. A
`ValueRef` visible to JavaScript is not trusted because of its public object
properties; the Go module resolves the object's hidden process-local identity.
A plain object cannot forge a reference. A handle from another workflow or
runtime cannot be reused.

Callbacks execute immediately during authoring. They configure Go builders and
are discarded. No function, closure, Goja value, module cache, or symbol is
serialized into IR.

Descriptor modules express task intent only. They do not load execution source
and cannot mutate the worker registry. The authoring runtime receives no
filesystem, network, database, store, submission, or operator capability.

### 3.3 Canonical equality requirement

The definitive Slice 2 test is not “JavaScript compilation returned no error.”
It requires:

```text
JavaScript-authored WorkflowIR == direct-Go WorkflowIR
canonical JavaScript IR bytes == reviewed IR golden
compiled JavaScript plan bytes == reviewed plan golden
```

This ensures JavaScript is a typed frontend to Go truth.

### 3.4 TypeScript role

TypeScript declarations improve author ergonomics, but schema strings remain
the runtime contract. `input<T>() -> ValueRef<T>` carries a phantom authoring
type. Go still validates the declared schema against the task catalog.
Declaration files are exact goldens and are compiled by TypeScript tests.

### 3.5 Review guidance

Read `pkg/gojamodules/workflow/authoring_test.go` before `authoring.go`.
Verify duplicate names, unknown tasks/options, wrong bindings, cycles,
cross-runtime handles, and schema drift fail before submission.

## 4. Slice 3 — HTTP snapshot: external work is real

### 4.1 Purpose

Slice 3 introduces work whose latency, availability, content, and charging are
controlled by another system. The workflow must survive transport errors,
status errors, redirects, large responses, cancellation, and restart without
persisting credentials or response bodies in control state.

The fixture in `pkg/testfixtures/workflowv3http` reads a bounded URL-list
artifact and writes a snapshot artifact through exact alias `fetch:public`.

### 4.2 Capability design

A module alias is a policy-selected capability, not just an import name. The
plan records `fetch:public`; host configuration provides:

- a nonempty, non-wildcard origin allowlist;
- timeout and maximum response bytes;
- an injected `http.Client` and guarded transport;
- disabled environment/file credential sources;
- redirect checks applying the same policy at every hop.

The public profile rejects URL userinfo and `Authorization`, `Cookie`, and
`Proxy-Authorization` headers. A future authenticated fetch must use a distinct
alias. Relaxing `fetch:public` would change the meaning of existing plans.

### 4.3 Typed failure contract

Remote failures are normalized before persistence:

| Condition | Durable class/code | Retry |
|---|---|---|
| transport/policy/size | `transport/HTTP_FETCH_TRANSPORT` | bounded |
| 429 | `rate-limit/HTTP_FETCH_RATE_LIMIT` | bounded |
| 5xx | `provider-5xx/HTTP_FETCH_SERVER` | bounded |
| other non-2xx | `validation/HTTP_FETCH_STATUS` | terminal |
| invalid cardinality | `validation/HTTP_SNAPSHOT_CARDINALITY` | terminal |

The persisted message is host-generated from the stable code. It does not copy
the URL, headers, body, or JavaScript stack.

### 4.4 Cancellation and privacy

The engine watches durable lease validity. Cancellation invalidates the epoch,
the watcher cancels the task context, and Goja/fetch observes that context.
The output records stable input indexes rather than response URLs so a query
credential cannot be copied into the artifact.

### 4.5 Evidence

Tests cover 503 then success, three 429 attempts, terminal 404, denied origin,
denied redirect, URL password, response limit, in-flight cancellation, reopen,
and canary scans. A denied request must prove zero network contact, not merely
return an error after contact.

## 5. Slice 4 — dispatcher: concurrency is real

### 5.1 Purpose

Slice 4 removes the v2 fixed-cycle barrier. Capacity must be replenished when
one compatible attempt completes, without waiting for unrelated long attempts.
It also makes resource limits durable across independent worker/store
connections.

### 5.2 Resource model

Each plan node has one primary symbolic resource class, such as:

```text
cpu.default
network.http.public
database.sync.primary
generation.primary
embedding.local
```

The store counts running nodes by class inside the immediate lease transaction.
A local semaphore may reduce contention, but it is not authoritative. Two
processes using separate SQLite connections must not both consume the final
slot.

### 5.3 Dispatch loop

The dispatcher fills all compatible capacity, then waits for a completion,
context cancellation, or a maintenance poll. A completion immediately returns
to the fill loop. Polling remains necessary for cross-process changes, retry
deadlines, and lease expiry; it is not the primary refill signal.

Fairness is tracked by `(run_id, resource_class)`. Dispatching database work
must not reduce a run's priority for HTTP work. Candidate scans cannot use an
unsafe fixed prefix that hides admissible work behind blocked rows.

### 5.4 Failure isolation

A typed attempt failure that the engine successfully records is not a
dispatcher infrastructure failure. The dispatcher continues unrelated runs.
An unrecorded store/transaction error remains fatal because the authoritative
outcome is unknown.

### 5.5 Projection contract

`QueueSnapshot` derives ready count, active-by-resource, and blocked reasons
from current rows. Current reasons include dependency, retry backoff, resource
capacity, and implementation unavailable. The snapshot is a query, not a
second mutable scheduler state.

### 5.6 Evidence

The timeline test holds an HTTP task and a slow unrelated task. When only the
HTTP task completes, the next HTTP task starts before the slow task finishes.
A two-connection race proves one capacity winner. Fairness, retry deadlines
across reopen, failure isolation, and all blocked reasons have focused tests.

## 6. Slice 5 — database synchronization: side effects are real

### 6.1 Purpose

Slice 5 introduces an externally committed side effect that cannot be rolled
back by the workflow database. The critical crash window is:

```text
target database commit succeeds
        ↓
worker dies before workflow Complete commits
```

Retrying without domain idempotency would duplicate the logical operation.

### 6.2 Host-preconfigured authority

The exact alias `db:sync` provides a Go-configured database handle. JavaScript
cannot select a driver, DSN, or credential and cannot call `configure()`.
Arbitrary SQL remains trusted-bundle authority; less-trusted database work
requires Slice 11 isolation or a narrower domain module.

### 6.3 Stable operation identity

`ctx.identity().operationKey` is the canonical SHA-256 digest of
`{runId,nodeKey}`. Attempt number is excluded. Every retry of one logical node
therefore presents the same key to the target system.

The target transaction atomically applies domain rows, inserts the operation
marker, and inserts one audit row. A retry first checks the marker and returns
a receipt without writing again.

### 6.4 Evidence

The strongest crash test runs attempt one and commits 500 customer rows, then
discards the returned result without calling workflow `Complete` or `Fail`.
The store closes with a running lease. After expiry and reopen, attempt one is
`lease_lost`; attempt two sees the same target marker and adds no rows or audit.

A bad-cardinality run fails independently while a valid run succeeds. Source
records and SQL values are absent from workflow SQLite. Evidence reports
499,554 source bytes versus 90,112 workflow-control bytes.

### 6.5 Review guidance

Inspect target transaction ordering and verify the marker is not written before
or after domain writes in a separate commit. Confirm workflow idempotency is
not inferred from attempt status; it is enforced by the target domain.

## 7. Slice 6 — lazy maps: scale-out is real

### 7.1 Status and purpose

Slice 6 is implemented. It adds dynamic graph cardinality while keeping
submission, restart, and scheduler memory bounded. The executable fixture
expands and runs 1,807 source items across reopen, then publishes one ordered
output manifest. Source items do not become source-bearing static plan nodes or
one transaction containing the entire expanded graph.

### 7.2 Canonical authoring contract

The DSL adds typed set references and a map declaration. The map callback runs
once during authoring with a symbolic item reference. It produces task
descriptor IR; it is never stored as JavaScript.

A compiled map records at least:

```go
type MapSpec struct {
    Key          string
    Source       SetRef
    ItemTask     PlanNodeTemplate
    PageSize     int
    MaxItems     int
    KeyPolicy    string
    OutputSchema string
}
```

Host policy validates effective page and item ceilings. A script may request
smaller limits but cannot raise them.

### 7.3 Source manifest contract

The map source is an immutable artifact whose entries have canonical unique
item keys and compact per-item references. The workflow database stores the
manifest `ArtifactRef`, digest, cursor, counts, and child identities—not item
payloads.

Item keys must come from stable domain identity or canonical index plus source
digest. They cannot depend on expansion time, random UUIDs, worker identity, or
completion order.

A child node key is derived deterministically from:

```text
map declaration key + source manifest digest + canonical item key
```

The exact encoding and digest algorithm must be frozen in golden tests.

### 7.4 Durable expansion state

Add logical tables similar to:

```text
v3_expansions
  (run_id, map_key, source_digest, next_index, total_items,
   materialized_items, terminal_items, status, updated_at)

v3_expansion_pages
  (run_id, map_key, page_no, first_index, item_count, page_digest)
```

One immediate transaction reads the current cursor, inserts at most `pageSize`
child nodes and dependencies with conflict-safe deterministic keys, records the
page, advances the cursor, and commits. A crash before commit adds nothing. A
crash after commit resumes at the next cursor. Replaying a page cannot create a
second logical child.

Expansion work must be bounded and interleaved with execution. The dispatcher
should not materialize every page before allowing existing children to run.
Backpressure limits the difference between materialized and terminal children.

### 7.5 Completion and failure semantics

A map is expansion-complete only when the source cursor reaches validated
cardinality. It is execution-complete only when every materialized child has a
terminal acceptable outcome and output cardinality matches policy.

Terminal child failure propagates according to compiled map failure policy.
Cancellation stops new pages and cancels pending/running children. Retry of one
child does not rematerialize the item. A source digest change is an identity
change and cannot be substituted into an existing run.

### 7.6 Required evidence

- expand hundreds and then at least 1,807 real items;
- kill/reopen before page commit, after page commit, and during child work;
- prove exact item-key set, no duplicates, no omissions, and stable order;
- prove bounded transaction size and materialized backlog;
- run two independent expanders against one SQLite database;
- cancel during expansion and prove no post-cancel children appear;
- scan SQLite/WAL/events for source-record canaries;
- expose total, materialized, running, succeeded, failed, and backlog counts.

## 8. Slice 7 — bounded reductions: scale-in is real

### 8.1 Status and purpose

Slice 7 is implemented. It turns a large typed set into one validated root
without creating a node with thousands of dependencies or loading all payloads
into workflow SQLite or one runtime. The real fixture reduces 257 map outputs
through three levels and resumes after a level-zero partition succeeds.

The reference workload is word count because its mathematical result is easy
to verify across different concurrency and partition schedules.

### 8.2 Canonical reduction contract

A reduction records:

```go
type ReduceSpec struct {
    Key          string
    Source       SetRef
    Task         PlanNodeTemplate
    FanIn        int
    OutputSchema string
    RootPolicy   string
}
```

`FanIn` is compiled and bounded by host policy. Every reducer consumes a compact
ordered partition manifest of at most `FanIn` references and emits one typed
immutable result reference.

### 8.3 Deterministic tree construction

Partition membership is based on canonical source item-key order, not task
completion time. A partition identity includes:

```text
reduction key + level + partition ordinal + digest(sorted member identities)
```

When level 0 outputs are complete, they become the ordered input set for level
1. The process repeats until the root policy is satisfied. Every dependency
fan-in remains bounded.

The reducer's output artifact should carry enough metadata to validate member
count, range, schema, and digest without embedding source payloads in control
rows.

### 8.4 Restart and partial failure

Completed partitions are immutable and reused after restart. A failed
partition retries independently. The engine must never rebuild successful
partitions merely because another partition failed or because completion order
changed.

The reduction coordinator materializes a level only from a complete validated
preceding level unless the specific algorithm declares safe incremental
combination semantics. The first implementation should prefer this clear
boundary.

### 8.5 Required evidence

- reduce thousands of word-count outputs with fan-in limits such as 16 or 128;
- compare root bytes/digest at concurrency 1, 2, and higher;
- randomize child completion order and preserve the root digest;
- crash during multiple levels and reuse completed partitions;
- prove no reducer exceeds compiled fan-in;
- inject one malformed shard and isolate its failure;
- verify root member cardinality equals map cardinality;
- publish exactly one named root after complete validation;
- expose levels, partitions ready/running/terminal, and root readiness.

## 9. Slice 8 — registry generations: upgrades are real

### 9.1 Status and purpose

Slice 8 is implemented. Sealed registries remain immutable while a registry
manager atomically activates validated candidates, retains draining generations
for exact pinned work, and quarantines broken runtime construction without
spending domain retry limits.

### 9.2 Candidate-to-active lifecycle

A registry manager owns immutable generations:

```text
candidate discovered
  → artifacts fetched and digests verified
  → catalogs strictly decoded
  → entrypoints and module aliases validated
  → self-tests executed in registration-only runtime
  → candidate sealed and generation digest computed
  → atomic activation for new admission
  → previous generation drains
  → cleanup only after references reach zero
```

Any candidate failure leaves the active generation unchanged and emits one
redacted operator event. Partial registration is prohibited.

### 9.3 Pinning semantics

Plans pin exact task implementations, not “latest generation.” An attempt
records the generation that supplied its implementation. New plan A may still
resolve from an older retained generation while new plan B resolves from the
active generation if both are advertised.

A dispatcher asks the manager for an exact `PlanNode` resolution and acquires a
generation reference before leasing or before runtime creation in one coherent
admission protocol. The reference remains held through attempt completion.
Activation cannot invalidate that reference.

### 9.4 Coexistence and draining

Generation A and B may coexist. New leases prefer the active generation when it
contains the exact identity, but there is no substitution: a B bundle cannot
run an A-pinned node. Generation A can be removed only when:

- no active attempt holds it;
- no worker-local queued lease requires it;
- retention policy permits removal;
- advertisements have been updated atomically.

### 9.5 Quarantine

Repeated runtime-construction or self-test failures indicate a broken
implementation or worker generation, not ordinary domain input failure.
Quarantine withdraws the affected advertisement and prevents burning every
node's task retry budget. Quarantine state has bounded reason codes, threshold,
time, generation identity, and operator reset/reload behavior. It never stores
source or stack dumps in durable events.

### 9.6 Implemented evidence

- lease attempt A, activate B, and prove A finishes with A executable bytes;
- compile/submit new work on B and prove exact B bytes execute;
- keep an A-pinned pending run executable while A is retained;
- fail candidate validation and prove active A is unchanged;
- race activation with lease acquisition and prove a coherent generation;
- restart the manager from configured immutable bundle locks;
- quarantine repeated construction failure without consuming domain retries;
- prove old generation cleanup waits for reference count zero;
- expose active, draining, quarantined, and implementation availability state.

## 10. Slice 9 — budgets and projections: operations are real

### 10.1 Status and purpose

Slice 9 is implemented. Transactional integer usage limits now share the lease
transaction, while one-read-transaction operational snapshots reconstruct
committed truth. This is required before paid provider work: retry and
concurrency limits alone do not bound requests, tokens, bytes, or money.

### 10.2 Integer budget model

Budget dimensions use integers, never floating-point accounting. Initial
dimensions may include:

```text
requests
input_tokens
output_tokens
embedding_tokens
input_bytes
output_bytes
cost_microunits
```

The compiled plan records requested claims and effective host-approved
ceilings. Price tables and profiles are versioned compiler inputs with digests.
Credentials remain outside the plan.

### 10.3 Reservation transaction

Leasing and budget reservation are one transaction:

1. select a compatible ready node;
2. calculate its conservative maximum reservation;
3. verify account limit minus used minus reserved;
4. insert an attempt-keyed reservation;
5. create lease and running attempt;
6. commit.

No lease may exist without its required reservation. Independent SQLite
connections racing for the final request unit must produce one winner.

### 10.4 Settlement

Completion validates host/task usage evidence, charges actual usage, releases
unused reservation, and finishes the attempt in one transaction. A task cannot
report negative usage or usage above policy without a typed failure.

Lease loss and timeout are ambiguous when a provider may have accepted the
request. The default policy charges the conservative reservation. A provider
adapter may safely reduce it only with authoritative idempotency/usage evidence.

Exhaustion policy is compiled as one of:

- fail the run with typed `budget` failure;
- keep the node blocked until an account increase;
- create/activate an approval gate under Slice 10.

### 10.5 Authoritative projections

Operational snapshots extend beyond queue counts. They should include:

- run/node/attempt status counts;
- active and capacity by resource;
- ready and blocked counts by stable reason;
- oldest ready/running/backoff/waiting ages;
- retry and lease-loss counts;
- completion rates over bounded windows;
- map source/materialized/terminal/backlog counts;
- reduction levels/partitions/root state;
- budget limit/reserved/used/remaining by dimension;
- gate waiting/approved/rejected/expired counts;
- registry active/draining/quarantined generations.

Events stream committed changes, but snapshots remain the source of truth after
missed events or reconnect.

### 10.6 Implemented evidence

- race two connections for one budget unit;
- reserve, settle below estimate, and release the remainder;
- retry with a new reservation while preserving prior usage;
- reopen with active reservations and recover lease loss conservatively;
- prove no overspend under cancellation/completion races;
- exercise block, fail, and approval exhaustion policies;
- compare projections with direct SQL invariants after restart;
- verify attach/reopen returns the same authoritative snapshot;
- scan events and projections for provider body, prompt, credential, and SQL
  canaries.

## 11. Slice 10 — approval gates: waiting is real

### 11.1 Status and purpose

Slice 10 is implemented. A human or external decision may take minutes
or days. Modeling this as a sleeping JavaScript task would hold a lease,
resource slot, runtime, and possibly budget reservation for the entire wait.
A gate is therefore first-class durable control state, not a long task.

### 11.2 Gate lifecycle

A gate control record has the exact states:

```text
pending dependencies
  → waiting
  → approved | rejected | expired | canceled
```

Entering `waiting` creates no attempt and holds no execution lease or resource
capacity. Approval publishes a typed decision ref and releases downstream dependencies.
Rejection follows compiled fail-run policy; unsupported branch cancellation is
rejected by the compiler. Expiry is evaluated
from a durable deadline, not an in-memory timer.

### 11.3 Durable schema

A logical gate record contains:

```text
run_id, gate_key, status, version, policy_digest, requested_at, expires_at,
decided_at, decision_code, actor_id, typed decision ref, budget_activation
```

It must not contain authentication tokens, free-form sensitive comments, or
arbitrary operator payloads. If a signed approval artifact is required, store
an external `ArtifactRef` and its schema/digest.

### 11.4 Operator boundary

Authentication and authorization occur in a separate trusted operator service.
The store receives an authenticated actor identity, allowed decision, expected
current version, and bounded reason code. A compare-and-swap transaction
prevents duplicate or stale decisions.

`require("workflow")` may declare a gate. `workflow/task` may consume a typed
gate output if the plan allows it. Neither
surface receives general operator authority. Ordinary task code cannot approve
its own gate.

### 11.5 Cancellation, expiry, and restart

Canceling the run atomically cancels waiting gates. An approval racing with
cancel must yield exactly one committed result. Expiry can be advanced by any
maintenance worker through a conditional transaction and survives process
restart. Reopening a run reconstructs waiting state from the store.

### 11.6 Implemented evidence

- enter waiting exactly once across connections with zero attempt, lease,
  runtime, resource usage, or budget reservation;
- close/reopen and preserve exact gate identity/deadline;
- approve once and start downstream work;
- reject and enforce compiled terminal policy;
- expire after durable deadline without an in-memory timer dependency;
- cancel while waiting;
- race approve/reject/expire/cancel across connections and commit one outcome;
- deny unauthorized/stale operator commands before store mutation;
- keep credentials and comments out of SQLite/events;
- project waiting age and decision state accurately through bounded pages;
- run a real JavaScript gate across dispatcher/store restart while an unrelated
  run completes, then publish the typed decision without persisting source or
  decision-body canaries;
- activate a budget exhaustion gate, apply an authenticated account increase,
  and admit exactly one attempt; leave an unused budget gate pending without
  blocking successful completion.

## 12. Slice 11 — process isolation: broader trust is viable

### 12.1 Status and purpose

Slice 11 is not implemented yet. Fresh Goja runtimes and module allowlists are
appropriate for trusted first-party JavaScript. They are not a security
sandbox for mutually untrusted publishers or tasks with broad filesystem,
network, database, or process capabilities.

Slice 11 moves selected attempts across a process/container boundary.

### 12.2 Isolation profile

The compiled plan and worker advertisement include an isolation class, for
example:

```text
in-process.trusted
subprocess.restricted
container.networked
container.exec
```

Host policy chooses the effective class. Scripts cannot downgrade it.

A restricted worker receives only:

- exact bundle artifact and entrypoint;
- immutable attempt identity and input refs;
- read-only input mounts and attempt-scoped writable output;
- explicit network policy or no network;
- pre-opened/narrow service handles where possible;
- CPU, memory, wall-time, process-count, and output-byte limits;
- a narrow framed protocol for checkpoints, typed failure, usage, and outputs.

It does not receive scraper SQLite credentials or direct store access.

### 12.3 Protocol and fencing

The parent owns the workflow lease. The child reports candidate outputs; the
parent validates refs and performs fenced completion. Child death, malformed
frames, limit violation, or timeout becomes a typed attempt failure. Killing a
child cannot bypass the current cancellation epoch.

The protocol must have bounded frame size, strict schemas, deterministic
version negotiation, and no logs interpreted as control frames. Output files
are published to the artifact store only through the parent or a constrained
artifact service.

### 12.4 Required evidence

- run the same pure fixture in-process and isolated with equal output digest;
- deny undeclared filesystem and network access;
- enforce CPU/memory/time/output/process limits;
- kill the child and recover through retry/lease semantics;
- cancel the run and prove child termination plus stale-output rejection;
- reject wrong bundle/protocol/isolation identity;
- fuzz malformed and oversized protocol frames;
- prove secrets and host environment are absent;
- demonstrate an allowlisted `exec:*` or broad tool task only in the isolated
  class.

## 13. Slice 12 — RAG/TTC: an expensive production workload is safe

### 13.1 Status and purpose

Slice 12 is not implemented yet. It is the integrated acceptance workload for
all prior contracts. The goal is not merely to finish a benchmark. It is to
prove that real-provider preparation, publication, reopen, evaluation, cost,
and evidence remain correct under heterogeneous latency and restart.

The previous diagnostic v9 run is explicitly non-publishable. It duplicated a
source-bearing plan into operation inputs, produced approximately 14.67 GB of
operation JSON, a 20.8 GB SQLite file, a 20.7 GB WAL, and 94% filesystem usage.
Its fixed-cycle scheduler also left local embedding capacity idle behind remote
generation. Those artifacts must never be resumed or uploaded.

### 13.2 Ownership boundaries

- The RAG package owns provider task semantics, schemas, prompt construction,
  generation/embedding validation, prepared-corpus rules, and publication.
- Workflow V3 owns refs, dynamic graph materialization, resources, attempts,
  retries, budgets, cancellation, gates, and projections.
- Researchctl owns immutable experiment identity, attachments, metrics, costs,
  citations, and reports. It is not a second scheduler.

### 13.3 Intended graph

A representative plan contains:

```text
source corpus ArtifactRef
  → deterministic item manifest
  → lazy generation map [generation resource + request/token budget]
  → validation and retry of malformed provider output
  → lazy embedding map [independent embedding resource]
  → bounded prepared-shard reductions
  → complete cardinality/order validation
  → optional approval gate for cost/publication
  → atomic publication task
  → reopen and evaluate published corpus/index
  → researchctl evidence attachment
```

Generation and embedding capacities remain independently saturated. Reduction
order follows canonical source item keys, not provider completion order.

### 13.4 Publication contract

Publication is a side effect and uses Slice 5 principles. It writes immutable
content-addressed components, validates complete cardinality and digests, then
atomically publishes one manifest/pointer using a stable operation key. A crash
before pointer publication leaves unreferenced immutable artifacts. A crash
after publication but before workflow completion reopens the same publication
and does not create a second logical release.

A successful run must reopen the published corpus and index in a fresh process
before evaluation begins.

### 13.5 Paid-run preflight

Before spending provider budget, a local/fake-provider preflight must prove:

- exact corpus, model, provider profile, bundle, registry, and plan identity;
- 1,807-item cardinality with no duplicate/missing keys;
- source/prompt/vector/provider-body canaries absent from control SQLite;
- bounded SQLite main/WAL growth;
- malformed generation output produces a typed bounded retry;
- generation and embedding resources refill independently;
- restart during generation, embedding, reduction, and publication;
- conservative budget settlement on ambiguous provider timeout;
- cancellation and stale completion fencing;
- deterministic reduction and publication digests across concurrency levels;
- atomic publication and fresh-process reopen;
- redacted researchctl events, costs, metrics, and citations.

Only after this preflight passes should a small paid sample run. The full TTC
run begins only after sample cost and output evidence are reviewed.

## 14. Cross-slice state model

Later slices should extend the current model rather than create parallel
engines.

```text
WorkflowPlan
  static task nodes
  map declarations
  reduction declarations
  gate declarations
  budget claims
  exact implementation and isolation identities

Run
  static nodes
  dynamically expanded item nodes
  dynamically materialized reduction nodes
  gate state
  budget accounts/reservations
  registry generation used per attempt

Attempt
  immutable lease/execution outcome
  resource class
  registry generation
  budget reservation and settlement
  typed failure
  validated output refs
```

The dispatcher remains responsible for executable node admission. Expanders,
reduction coordinators, gate expiry, registry reload, and budget reconciliation
are control-plane transitions. They should use small idempotent transactions
and must not perform provider/database calls while holding SQLite transactions.

## 15. Implementation order and dependency map

```text
Slices 1–2: representation, durability, authoring
      ↓
Slice 3: typed external failure and policy aliases
      ↓
Slice 4: durable concurrent resource admission
      ↓
Slice 5: stable side-effect idempotency
      ↓
Slice 6: deterministic dynamic children
      ↓
Slice 7: deterministic bounded aggregation
      ↓
Slice 8: implementation lifecycle during upgrades
      ↓
Slice 9: transactional cost and operational truth
      ↓
Slice 10: lease-free external decisions
      ↓
Slice 11: stronger execution trust boundary
      ↓
Slice 12: integrated paid production workload
```

Slices may share internal preparation, but their acceptance evidence should
remain separate. For example, budgets can be represented while implementing
maps, but Slice 9 is not complete until reservation races, settlement,
recovery, exhaustion policy, and projections all pass.

## 16. Intern implementation workflow

For each slice:

1. Read this guide, the primary architecture, bundle design, cookbook, and the
   dedicated slice design.
2. Start from the real fixture and write the failure/restart timeline before
   changing types.
3. Extend canonical Go types and strict validation.
4. Freeze IR/plan/schema/DTS goldens.
5. Add additive SQLite migration and migration-from-prior-slice tests.
6. Implement one idempotent store transition at a time.
7. Connect dispatcher/runtime behavior without adding ambient authority.
8. Add positive, negative, concurrent, restart, cancellation, and privacy
   tests.
9. Update projections and public help.
10. Run focused tests continuously, then full `make validate`, isolated lint,
    race tests, JavaScript syntax, TypeScript declarations, generated checks,
    help smoke, docmgr validation, and `git diff --check`.
11. Record commands, failures, fixes, decisions, risks, and commit hashes in the
    append-only diary.
12. Map every acceptance criterion to fresh evidence before checking the task.

Do not use a raw-operation shim to make an old API look like V3. Do not add a
second mutable projection table when state can be derived. Do not persist bulk
payloads to simplify restart. Do not treat a Goja module allowlist as process
isolation.

## 17. Current status and next action

| Slice | Current status | Next architectural action |
|---|---|---|
| 1 | implemented and validated | preserve invariants |
| 2 | implemented and validated | extend descriptors/types without a second compiler |
| 3 | implemented and validated | reuse typed external-failure boundary |
| 4 | implemented and validated | preserve database-scoped admission |
| 5 | implemented and validated | reuse stable operation keys for publication |
| 6 | implemented and validated | preserve deterministic paged expansion, publication, and privacy invariants |
| 7 | implemented and validated | preserve bounded fan-in, level recovery, deterministic root, and privacy invariants |
| 8 | implemented and validated | preserve exact generation acquisition, draining, quarantine, and retry-debt separation |
| 9 | implemented and validated | preserve atomic reservation, exact settlement/recovery, and authoritative projections |
| 10 | design target | freeze gate CAS/operator/state contract |
| 11 | design target | define worker protocol and isolation profiles |
| 12 | design target | implement only after Slices 6–11 pass their preflights |

The immediate implementation tranche is Slices 6–10. Each now requires its own
design document in addition to this cross-slice guide. Slice 11 and 12 remain
fully described here and in the primary architecture until their implementation
tranche begins.

## References

- [Primary Workflow V3 architecture](01-durable-dataflow-workflow-v3-and-modern-scripting-architecture.md)
- [JavaScript task bundles and worker registries](02-reproducible-javascript-task-bundles-and-worker-registries.md)
- [Slice 6 dedicated design](03-slice-6-lazy-maps-deterministic-durable-scale-out.md)
- [Slice 7 dedicated design](04-slice-7-bounded-reductions-deterministic-durable-scale-in.md)
- [Slice 8 dedicated design](05-slice-8-registry-generations-safe-durable-upgrades.md)
- [Slice 9 dedicated design](06-slice-9-budgets-and-projections-operational-control.md)
- [Slice 10 dedicated design](07-slice-10-approval-gates-durable-lease-free-waiting.md)
- [Investigation diary](../reference/01-investigation-diary.md)
- [Evidence map](../reference/02-source-catalogue-and-evidence-map.md)
- [JavaScript cookbook](../reference/03-workflow-v3-javascript-cookbook-and-execution-atlas.md)
