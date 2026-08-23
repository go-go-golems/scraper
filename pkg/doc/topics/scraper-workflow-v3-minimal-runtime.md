---
Title: Workflow V3 Runtime Slices 1–11
Slug: scraper-workflow-v3-minimal-runtime
Short: "Explains executable workflow-v3 file, HTTP, dispatcher, database, map, reduction, registry, budget, gate, and restricted-process slices with durable privacy boundaries."
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

Workflow V3 is the primary public workflow product and has eleven executable
vertical slices for trusted and restricted JavaScript tasks: linear file
processing, typed authoring, allowlisted HTTP, work-conserving resource
dispatch, idempotent database synchronization, deterministic lazy maps, bounded
reduction trees, immutable rolling registry generations, transactional budgets
with authoritative operational projections, durable lease-free approval gates,
and exact bounded subprocess isolation. Use `scraper workflow`, `scraper worker`,
and `scraper task-packages`; the retained site engine is explicitly legacy.

## What runs today

A CommonJS authoring script can import `workflow` and a descriptor-only task
module, then use:

- `workflow.define(name, callback)`;
- `plan.input(name, {schema})`;
- `plan.inputSet(name, {itemSchema, manifestSchema, maxItems})`, where `maxItems` is the explicit external cardinality contract;
- `plan.task(nodeKey, taskDescriptor, callback?)`;
- `plan.map(name, set, itemCallback, mapPolicyCallback?)`;
- `plan.reduce(name, set, partitionCallback, reducePolicyCallback?)`;
- `job.after(otherJob)`;
- `job.output(port)`;
- `plan.output(name, value)` and `plan.outputSet(name, set)`;
- `workflow.validate`, `workflow.digest`, `workflow.toIR`, and
  `workflow.compile`.

The JavaScript builder produces the same canonical Go IR as direct Go
construction. Data bindings automatically create producer readiness edges;
`job.after(otherJob)` is only needed for control ordering when no output is
consumed. Compilation pins task kind, task version, bundle digest,
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

## Rolling registry behavior

A worker may atomically activate a fully validated sealed registry generation.
The prior generation becomes draining, and each lease retains the exact
registered task and generation acquired before its attempt is persisted. Old
A bytes can therefore finish after B activates while newly admitted B-pinned
plans execute B bytes. Attempt records expose the exact generation.

Candidate self-test failure leaves the active generation unchanged. Repeated
module/factory/runtime construction failure creates append-only infrastructure
attempts without consuming semantic task retry debt, quarantines the broken
generation, and projects affected work as `implementation-unavailable`.
Dispatcher projections include active, draining, and quarantined generations.
Removal is denied while a generation has acquired references.

## Transactional budget and projection behavior

Plans declare sorted run accounts with integer limits and policy digests. Exact
task, map, and reduction claims record requested and effective reservations;
task bundle catalogs cap each dimension. Closed dimensions cover requests,
tokens, bytes, and integer cost microunits. Unknown, negative, duplicate,
unsorted, floating, unsafe JavaScript integers, and overflow are rejected.

Lease admission reserves every dimension in the same immediate SQLite
transaction that creates the attempt. Two independent connections racing for
the final request unit produce one winner. Completion settles validated actual
usage and releases the remainder. Failed tasks may settle reported actual
usage; ambiguous failure, lease loss, or in-flight cancellation charges the
full conservative reservation. Preparation failure before task execution
releases it. Retries create new immutable reservation rows.

`block` leaves work pending with `budget:<account>:<dimension>`;
`require-approval` leaves it lease-free with a budget-approval reason for Slice
10; `fail-run` records `budget/BUDGET_EXHAUSTED` evidence without creating an
attempt. Versioned operator increases use expected-version CAS and unblock
exact work. Startup reconciliation rejects inconsistent account/reservation
state instead of inventing usage.

An operational snapshot is reconstructed in one read transaction. It includes
run/node/attempt states, retries, lease loss, resource queue state, maps,
reductions, budget limit/used/reserved/remaining, explicit terminal-rate
windows, ages, and an event high-water sequence. Runtime augmentation adds
active/draining/quarantined registry generations. Consumers read a snapshot,
then continue through bounded sequence-ordered events.

## Durable approval-gate behavior

`plan.gate(...)` declares a typed decision point with exact dependencies,
decision schema, bounded timeout, required role identifier, and explicit
reject/expiry policy. Once dependencies succeed, SQLite transitions the gate
from pending to waiting exactly once. Waiting creates no task attempt, lease,
Goja runtime, resource grant, or budget reservation, so the dispatcher remains
work-conserving for unrelated runs across arbitrarily long waits and restart.

A trusted operator service—not JavaScript authoring or task code—submits a
bounded versioned approve/reject command. The store revalidates role, current
version, run state, deadline, decision code, and typed immutable artifact ref in
one immediate transaction. Identical retries are idempotent; conflicting,
stale, unauthorized, expired, canceled, or already-decided commands cannot
revive or overwrite a terminal gate. Rejection and expiry fail the run because
branch cancellation is deliberately rejected until explicit branch boundaries
exist.

Approval publishes the compact decision ref and enables ordinary downstream
nodes exactly once. Maintenance expires durable deadlines without worker-local
timers. Run cancellation atomically cancels pending/waiting gates. Budget
`require-approval` claims activate their compiled gate only on exhaustion and
still require a separately authorized account increase before admission.
Operational projections expose bounded/paginated status, version, waiting age,
deadline remaining, role, decision code/time, and artifact presence; raw
approval bodies remain external.

## Restricted subprocess behavior

Tasks compiled as `subprocess.restricted` run in a fresh static worker under
Bubblewrap and delegated cgroup v2 controls. Plans and registry generations pin
requested/effective limits plus the exact digest of worker, pre-exec launcher,
Bubblewrap, protocol, and fixed allowlisted tool bytes. Retained rolling
registry generations retain matching executors; latest-profile substitution is
rejected.

A static launcher joins the configured memory/process/CPU cgroup before
Bubblewrap forks. The sandbox unshares user, PID, IPC, UTS, cgroup, and network
namespaces, clears environment, exposes only read-only worker/bundle/input/tool
mounts, and has writable attempt output and tmpfs. Parent cancellation or
wall-time uses `cgroup.kill`. Broad `exec:allowlisted` tasks are compiler-denied
outside this class and receive only fixed tool IDs—never shell strings,
executable paths, environment, arbitrary cwd, or redirection.

The child receives one bounded canonical request and returns one bounded
canonical response. It has no workflow SQLite or artifact-store authority.
Candidate output files are rechecked by the parent for exact ports/schemas,
file/byte bounds, regular-file confinement, size, and digest before parent-side
artifact publication and ordinary lease/cancellation fencing. Attempts and
queue projections expose isolation class, policy/executor digests, and
ready/active isolation counts.

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

```

The focused tests cover the 12,000-row file workflow, real local HTTP and
SQLite target servers, a 1,807-item JavaScript lazy map, a 257-item multi-level
JavaScript reduction, exact A/B registry generations, a real budget-reporting
JavaScript task, a real JavaScript approval workflow that waits across a store/dispatcher
restart while an unrelated run completes, and real static Bubblewrap workers
under cgroup limits. They prove typed
retry, atomic activation,
draining, quarantine without domain retry debt, database-scoped reservation,
actual/conservative/zero settlement, exhaustion and CAS increase, lease-free
gate waiting, typed continuation, version/idempotency enforcement, approval vs
reject/expiry/cancel races across independent connections, durable deadline
expiry, operator-module denial, allowlist and redirect denial,
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
- `pkg/testfixtures/workflowv3budget` — real integer usage reporting, settlement,
  overage, reopen, and privacy workflow.
- `pkg/testfixtures/workflowv3gate` — real wait, decision-artifact continuation,
  unrelated-run progress, dispatcher restart, and privacy workflow.
- `pkg/testfixtures/workflowv3isolation` — restricted transform, spinning child,
  allowlisted tool, worker-death retry, limits, cancellation, and privacy.

The final slice adds the integrated RAG/TTC production workload. V3 does not
translate or silently accept v2 raw operations.
