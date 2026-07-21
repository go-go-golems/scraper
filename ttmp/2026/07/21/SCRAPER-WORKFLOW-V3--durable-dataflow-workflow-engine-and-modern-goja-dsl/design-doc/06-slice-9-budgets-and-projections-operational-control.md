---
Title: Slice 9 Budgets and Projections - Operational Control
Ticket: SCRAPER-WORKFLOW-V3
Status: complete
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
    - Path: repo://pkg/workflowv3/compiler.go
      Note: Requested versus effective budget validation
    - Path: repo://pkg/workflowv3/types.go
      Note: Canonical integer budget claims policies usage and projection types
    - Path: repo://pkg/workflowv3runtime/task_runner.go
      Note: Bounded task usage evidence returned for settlement
    - Path: repo://pkg/workflowv3sqlite/projection.go
      Note: |-
        Authoritative operational snapshots
        Current queue projection is the operational snapshot baseline
    - Path: repo://pkg/workflowv3sqlite/store.go
      Note: |-
        Atomic lease reservation settlement retry and recovery
        Budget reservation and settlement must join lease transactions
ExternalSources: []
Summary: Implementation contract for integer transactional budget admission and comprehensive store-derived operational projections across resources maps reductions registries retries and reopen.
LastUpdated: 2026-07-21T20:45:00-04:00
WhatFor: Freeze Slice 9 accounting transaction exhaustion recovery and projection semantics before paid provider work.
WhenToUse: Read before adding plan budget claims usage settlement cost controls progress APIs or operator dashboards.
---


# Slice 9 Budgets and Projections - Operational Control

## Executive summary

Slice 9 makes request, token, byte, and monetary limits transactional with
leasing. Every budgeted attempt reserves a conservative integer amount in the
same transaction that creates its lease. Completion settles validated actual
usage and releases the remainder. Ambiguous lease loss charges conservatively.
Independent connections cannot overspend the same account.

The slice also expands projections into a complete authoritative operational
view. Events signal changes; snapshots reconstructed from committed rows remain
truth after reconnect or restart.

## Problem statement

Concurrency and retry bounds do not bound total provider cost. Checking a
counter before lease in one query and incrementing it later races across
workers. Floating-point currency creates non-reproducible arithmetic. A mutable
“progress” cache can disagree with attempts after crash.

## Scope

Included: canonical budget dimensions/claims, host ceilings, account creation,
reservation at lease, settlement, retry, cancellation, lease-loss policy,
exhaustion policies, usage validation, budget events, and comprehensive
projections.

Excluded: invoice reconciliation, live exchange rates, arbitrary user-defined
floating dimensions, and provider-specific credentials. Price/profile data is
versioned host/compiler input.

## Canonical model

All quantities are nonnegative signed 64-bit integers with checked arithmetic:

```go
type BudgetAmount struct {
    Dimension string `json:"dimension"`
    Units int64 `json:"units"`
}

type BudgetClaim struct {
    Account string `json:"account"`
    Reserve []BudgetAmount `json:"reserve"`
    OnExhausted string `json:"onExhausted"` // fail-run | block | require-approval
}

type BudgetUsage struct {
    Dimension string `json:"dimension"`
    Units int64 `json:"units"`
}
```

Initial closed dimensions:

```text
requests
input_tokens
output_tokens
embedding_tokens
input_bytes
output_bytes
cost_microunits
```

Unknown dimensions fail compilation. Currency uses integer microunits. Token
price tables have identity/digest and effective model/provider scope.

Task/catalog policy provides a maximum reservation. Authoring may request a
smaller run ceiling but cannot raise host/task ceilings. The compiled plan
records requested and effective values so reports explain admission.

## Durable schema

```text
v3_budget_accounts
  run_id, account, dimension, limit_units, used_units, reserved_units,
  policy_digest, updated_at

v3_budget_reservations
  run_id, node_key, attempt_no, account, dimension,
  reserved_units, settled_units, status, created_at, settled_at
```

Primary keys include attempt identity and dimension. Checks enforce nonnegative
values, `used + reserved <= limit`, and reservation status vocabulary.
Reservation rows are immutable in identity and retain final settlement evidence.

If accounts span runs, add a host/project scope with the same transactional
rules and stable account identity. The first fixture should use run-scoped
accounts before generalizing.

## Lease admission transaction

Extend `LeaseNextWithResources`:

1. select a ready, exact-implementation-compatible candidate;
2. verify resource capacity;
3. read its compiled claims;
4. for every dimension, check overflow and available units;
5. apply exhaustion policy if unavailable;
6. increment account `reserved_units`;
7. insert attempt reservation rows;
8. create lease/running attempt and fairness update;
9. commit.

All dimensions reserve atomically. Partial reservation is prohibited. Use
conditional updates or immediate transaction serialization so two SQLite
connections racing for the final unit produce one winner.

A node blocked on budget remains pending and receives derived reason
`budget:<account>:<dimension>`. Sensitive price configuration is not copied
into that reason.

## Usage evidence and settlement

`workflow/task` adds a narrow usage reporter or task success envelope. The
runtime accepts only declared dimensions, nonnegative integers, and bounded
values. Where possible, trusted host modules report usage directly rather than
trusting JavaScript. Provider adapters normalize response metadata.

Successful completion transaction:

```text
validate usage and output refs before transaction
check current lease/cancel epoch
for each reservation:
  reserved_units -= reservation
  used_units += actual/conservative charge
  mark reservation settled with settled_units
finish attempt/node and commit outputs/event
```

A task reporting above reservation fails closed unless policy explicitly
supports a bounded supplemental atomic charge. The first implementation should
reject it as `budget/BUDGET_USAGE_EXCEEDS_RESERVATION`.

A failed attempt still settles any actual usage. Failure classification and
budget charge are separate facts.

## Lease loss, timeout, and cancellation

A remote request may have completed after local timeout. Without authoritative
provider evidence, settle the full conservative reservation on lease loss or
ambiguous transport timeout. This may overcount but cannot overspend.

If failure proves no provider contact—for example policy denied before
transport—settle zero and release reservation. The failure adapter supplies a
host-trusted usage disposition, not arbitrary error-string matching.

Cancellation before execution releases unused reservation. Cancellation during
ambiguous external work follows conservative charge. Every branch is tested.

## Exhaustion policies

- `fail-run`: persist typed `budget/BUDGET_EXHAUSTED` and fail according to
  compiled policy.
- `block`: keep node pending until an authenticated budget increase; no lease.
- `require-approval`: create or activate a Slice 10 gate; no lease while
  waiting.

Account increases are operator actions with actor identity and expected
version. They do not rewrite historical reservations.

## Projection model

Define a new `OperationalSnapshot` rather than overloading a small queue type.
It contains bounded maps/lists and a snapshot timestamp:

```text
run statuses and ages
node/attempt counts and retry/lease-loss counts
ready and blocked-by-stable-reason
active/capacity by resource
completion/failure rates over explicit windows
map total/materialized/terminal/backlog
reduction levels/partition states/root readiness
registry active/draining/quarantined/availability
budget limit/reserved/used/remaining by account/dimension
gate waiting/decision/age counts
```

Rates are derived from indexed terminal timestamps over named windows and state
the denominator. Ages parse timestamps; do not compare RFC3339Nano strings
lexically.

Snapshots support global and run-filtered queries. They never include artifact
payloads, request URLs, prompts, headers, SQL, provider bodies, or arbitrary
failure messages.

## Snapshot consistency

One snapshot should use one read transaction so counts correspond to a coherent
SQLite state. Expensive sections may be separately paged only if the API
reports consistency boundaries. Index every status/time/join path used at
scale.

Events are compact notifications such as `budget.reserved`, `budget.settled`,
and `budget.exhausted`. Consumers always reattach by reading a snapshot then
following sequence-numbered events.

## Recovery and reconciliation

On lease reclamation, settle or transfer every running reservation in the same
transaction that marks the attempt `lease_lost`. The first implementation
settles conservatively and creates a fresh reservation for retry; it does not
move an old reservation silently.

A startup invariant query detects running/terminal attempts with inconsistent
reservation state. Repair is deterministic where possible; otherwise stop
admission and report a configuration/internal failure rather than invent usage.

## Failure vocabulary

- `budget/BUDGET_EXHAUSTED`;
- `budget/BUDGET_USAGE_INVALID`;
- `budget/BUDGET_USAGE_EXCEEDS_RESERVATION`;
- `configuration/BUDGET_POLICY_INVALID`;
- `configuration/BUDGET_PRICE_PROFILE_UNAVAILABLE`;
- `internal/BUDGET_ACCOUNT_INVARIANT`.

Messages contain dimension/account identifiers only when those identifiers are
validated non-sensitive plan values.

## Migration

Add budget tables, plan fields, indexes, and usage columns additively. Existing
nodes have no claims and lease exactly as before. Existing attempts require no
synthetic reservation. Canonical schema evolution must preserve old plan
interpretation or use a new explicit version.

Projection APIs should retain `QueueSnapshot` as a focused compatibility view
implemented from the same authoritative query components, not duplicate state.

## Test matrix

### Arithmetic and canonical

- reject negative, duplicate, unknown, overflow, and unsorted dimensions;
- requested/effective claims in exact plan goldens;
- integer cost calculation against versioned price profile;
- TypeScript exact declaration and compilation.

### Transactions

- two connections race for one request unit; one lease wins;
- multi-dimension reservation is all-or-nothing;
- settle below reservation releases exact remainder;
- settle at reservation reaches exact limit;
- over-reservation usage rejected without corrupting account;
- completion/cancel race settles once;
- retry creates a new immutable reservation;
- lease loss charges conservatively and reopens correctly;
- no-contact policy denial charges zero;
- migration from Slice 8 database.

### Exhaustion and projections

- fail, block, and approval policies;
- authenticated account increase unblocks exact nodes;
- snapshots equal direct SQL oracle after success/failure/retry/cancel/reopen;
- map/reduction/registry/resource/budget sections agree at one read boundary;
- attach snapshot plus event continuation misses no committed transition;
- timestamp age/rate boundary tests;
- privacy canary scan of SQLite/events/snapshot JSON.

## Implementation sequence

1. Freeze dimensions, claims, usage, exhaustion, canonical plan, and DTS.
2. Add account/reservation schema and invariant tests.
3. Integrate atomic reservation into lease transaction.
4. Add runtime/host usage evidence and completion settlement.
5. Add lease-loss/cancel/failure dispositions.
6. Add exhaustion policies, leaving approval integration behind Slice 10 API.
7. Build coherent operational snapshot and indexes.
8. Exercise all prior fixtures plus map/reduction under budgets.
9. Update help, diary, changelog, relations, generated artifacts.
10. Run race, full validation, privacy, docmgr, and diff checks.

## Implementation outcome

Commit `f07cbb7` implements closed integer dimensions, run accounts, task
maximums, requested/effective claims, JavaScript and TypeScript authoring, and
checked microunit arithmetic. Optional canonical fields preserve old plan
bytes; budgeted plans intentionally pin account policy and bundle/catalog
identity.

SQLite now stores compact account totals, per-node claims, and immutable
attempt reservations. Immediate lease transactions reserve all dimensions or
none. Completion settles actual usage, failure may settle trusted reported
usage, ambiguous execution/lease loss/cancellation charges conservatively, and
preparation failure releases reservations. Startup reconciliation fails closed
on account/reservation disagreement. Versioned operator increases use actor and
expected-version evidence.

The real JavaScript fixture reports request/output-token usage, proves
below-reservation release, failed-task actual settlement, overage failure,
zero-charge preparation denial, conservative cancellation, reopen, and privacy.
Independent SQLite connections prove final-unit admission and terminal races.
Map children and reduction partitions inherit claims through ordinary dynamic
node materialization.

`OperationalSnapshot` uses one read transaction and supports global/run
filters. It includes status counts, retries, lease losses, parsed ages, explicit
60/300-second terminal windows, resources, blocked reasons, maps, reductions,
budgets, and an event high-water mark. Bounded `EventsAfter` continuation and
runtime registry augmentation complete the local operator view. Slice 10 adds
the gate section and converts the existing lease-free `require-approval` bridge
into durable decisions.

No cross-run account scope, invoice reconciliation, exchange rates, or remote
worker heartbeat was added; those remain outside the documented Slice 9 scope.

## Acceptance criteria

Slice 9 is complete only when database-scoped admission cannot overspend,
settlement is exact and idempotent, ambiguous work charges conservatively,
retries preserve history, all exhaustion policies work, projections reconstruct
truth after reopen, no sensitive payload enters accounting/observability, and
all prior slices plus focused/race/full validation pass.

## Alternatives rejected

- **Check budget outside lease transaction:** races across workers.
- **Use floating-point currency:** arithmetic and reports become unstable.
- **Trust arbitrary JavaScript usage:** permits undercharging or invalid units.
- **Release all reservations on timeout:** may exceed paid provider ceiling.
- **Maintain mutable progress counters as truth:** crash can separate them from
  authoritative attempt/node state.

## References

- [All-slice intern guide](08-workflow-v3-slices-1-through-12-intern-architecture-and-analysis-guide.md)
- [Slice 10 gates](07-slice-10-approval-gates-durable-lease-free-waiting.md)
- [Primary architecture](01-durable-dataflow-workflow-v3-and-modern-scripting-architecture.md)
- [Diary](../reference/01-investigation-diary.md)
