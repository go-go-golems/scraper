---
Title: Slice 10 Approval Gates - Durable Lease-Free Waiting
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
      Note: Safe gate declaration without operator authority
    - Path: repo://pkg/workflowv3/compiler.go
      Note: Gate dependency output and expiry validation
    - Path: repo://pkg/workflowv3/types.go
      Note: Canonical gate declarations policies and decisions
    - Path: repo://pkg/workflowv3runtime/dispatcher.go
      Note: Gates must wait without dispatcher lease or resource use
    - Path: repo://pkg/workflowv3sqlite/projection.go
      Note: Waiting age and decision projections
    - Path: repo://pkg/workflowv3sqlite/schema.sql
      Note: Durable waiting and decision records
    - Path: repo://pkg/workflowv3sqlite/store.go
      Note: |-
        Compare-and-swap approve reject expire and cancel transitions
        Gate compare-and-swap transitions extend durable control state
ExternalSources: []
Summary: Implementation contract for durable approval and external-event gates that wait across restart without consuming attempts leases resources runtimes or budgets.
LastUpdated: 2026-07-21T20:50:00-04:00
WhatFor: Freeze Slice 10 authoring state operator transaction race privacy and continuation behavior before implementation.
WhenToUse: Read before adding human approval budget escalation external callbacks or any workflow step that may wait longer than an execution lease.
---


# Slice 10 Approval Gates - Durable Lease-Free Waiting

## Executive summary

Slice 10 introduces a first-class gate node. Once dependencies are satisfied,
the gate enters durable `waiting` state without creating an attempt, lease,
Goja runtime, resource grant, or budget reservation. An authenticated operator
service may approve or reject it through a versioned compare-and-swap
transaction. Durable deadlines support expiry. Run cancellation races safely
with decisions.

A task is never allowed to sleep while waiting for a human and never receives
operator authority merely because it created data used by a gate.

## Problem statement

Approvals, budget escalation, deployment windows, and external callbacks may
take hours or days. A sleeping task wastes capacity, loses state on worker
restart, requires repeated lease extension, and allows stale completion to
conflict with operator decisions. Polling tasks create repeated attempts and
provider traffic.

## Scope

Included: canonical gate declarations, gate node mode, waiting state, expiry,
approve/reject/cancel CAS transitions, authenticated operator boundary,
optional signed decision artifact refs, continuation, events, projections,
restart, and concurrency tests.

Excluded: identity-provider implementation, UI, notification delivery, general
message queues, and arbitrary comments in control rows. Those systems call the
operator API but do not change store semantics.

## Canonical model

```go
type GatePolicy struct {
    DecisionSchema string `json:"decisionSchema"`
    OnReject string `json:"onReject"` // fail-run | cancel-branch
    OnExpire string `json:"onExpire"` // fail-run | cancel-branch
    TimeoutMillis int64 `json:"timeoutMillis,omitempty"`
    RequiredRole string `json:"requiredRole"`
}

type IRGate struct {
    Key NodeKey `json:"key"`
    DependsOn []NodeKey `json:"dependsOn"`
    Policy GatePolicy `json:"policy"`
}
```

Required role is a non-secret policy identifier resolved by the operator host.
Absolute expiry is computed at the durable transition to waiting from compiled
timeout policy, not at authoring time.

Gate output is a typed compact decision reference or engine-owned decision
value. Downstream bindings can reference it through the same schema validation
as task outputs.

## JavaScript authoring contract

```typescript
interface PlanBuilder {
  gate<TDecision>(
    name: string,
    options: {
      schema: string;
      timeoutMs?: number;
      requiredRole: string;
      onReject?: "fail-run" | "cancel-branch";
      onExpire?: "fail-run" | "cancel-branch";
    },
    configure?: (gate: GateBuilder) => void,
  ): ValueRef<TDecision>;
}
```

`GateBuilder.after(...)` adds dependencies. The safe `workflow` module can
declare gates but cannot decide them. `workflow/task` may emit data that a gate
depends on; it cannot approve the gate. Operator APIs are a separate trusted
surface and are never installed in authoring/task runtimes.

## Durable schema

```sql
CREATE TABLE v3_gates (
  run_id TEXT NOT NULL,
  gate_key TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN
    ('pending','waiting','approved','rejected','expired','canceled')),
  version INTEGER NOT NULL DEFAULT 0,
  policy_digest TEXT NOT NULL,
  decision_schema TEXT NOT NULL,
  required_role TEXT NOT NULL,
  requested_at TEXT,
  expires_at TEXT,
  decided_at TEXT,
  decision_code TEXT,
  actor_id TEXT,
  decision_ref_schema TEXT,
  decision_ref_digest TEXT,
  decision_ref_media_type TEXT,
  decision_ref_size_bytes INTEGER,
  decision_ref_locator TEXT,
  PRIMARY KEY (run_id,gate_key),
  FOREIGN KEY (run_id) REFERENCES v3_runs(run_id) ON DELETE CASCADE
);
```

The gate may also have a row in `v3_nodes` with mode/status support so existing
dependency queries remain unified. It never has a `v3_attempts` row. If node
status vocabulary is extended with `waiting`, migration and all status queries
must be updated explicitly.

No auth token, session cookie, raw request, arbitrary comment, or external
payload is stored. Long/signed evidence is an immutable validated artifact ref.

## State machine

```text
pending
  -- dependencies succeeded --> waiting
waiting
  -- authorized approve CAS --> approved
  -- authorized reject CAS --> rejected
  -- deadline CAS --> expired
  -- run cancel CAS --> canceled
```

Terminal decisions do not transition again. Repeating the identical operator
request may return the committed decision idempotently; a conflicting request
returns `ErrGateAlreadyDecided` with no mutation.

Approval marks the gate node succeeded, commits the typed decision output, and
makes downstream nodes ready in one transaction. Rejection/expiry applies the
compiled policy and updates run/branch state atomically.

## Enter-waiting transition

A maintenance/store action scans pending gates whose dependencies succeeded.
One transaction:

1. verifies run is running and gate is pending;
2. verifies all dependencies succeeded;
3. computes `requested_at` and optional `expires_at` from trusted store time;
4. sets waiting and increments version;
5. appends bounded `gate.waiting` event;
6. commits.

It does not allocate a lease/resource/budget or create an attempt. Two workers
racing this transition produce one result.

## Operator boundary

The operator service authenticates the caller and authorizes `requiredRole`
outside the workflow store. It passes:

```go
type GateDecisionCommand struct {
    RunID RunID
    GateKey NodeKey
    ExpectedVersion int64
    Decision string // approve | reject
    DecisionCode string
    ActorID string
    DecisionRef *ArtifactRef
}
```

Actor and decision codes use strict bounded safe syntax. The store revalidates
command shape, current waiting state, expected version, deadline, run status,
and decision artifact schema. Authorization credentials never reach this
struct or SQLite.

For high-assurance environments, the operator service can create a signed
decision artifact. Signature verification occurs before the store transaction;
the store persists only the verified ref and identity facts.

## Expiry

Any maintenance worker can call `ExpireDueGates(now)`. SQLite compares parsed
or `julianday` timestamps, not lexical RFC3339Nano. The conditional update
requires waiting state and matching version/deadline. Restart does not reset the
deadline.

An approval at the same boundary as expiry has one winner under the immediate
transaction. The loser observes terminal state and cannot overwrite it.

## Cancellation and branch policy

Run cancellation increments the existing cancellation epoch and marks waiting
or pending gates canceled in the same transaction as nodes/attempts. Approval
racing cancel commits either before cancellation (after which the run may still
be canceled) or loses the running/waiting check. It cannot revive a canceled
run.

`cancel-branch` requires an explicitly represented branch/subgraph boundary.
If branch semantics are not implemented in this slice, compiler must reject
that option and support only `fail-run`; it must not silently interpret it.

## Budget integration

Slice 9 `require-approval` exhaustion activates a compiled gate. No budgeted
node lease exists while waiting. Approval must be paired with an authenticated
account increase or explicit one-time allowance in the same operator workflow;
otherwise the node becomes ready and immediately budget-blocked again. The
gate decision itself does not fabricate budget.

## Events and projections

Events:

```text
gate.waiting
gate.approved
gate.rejected
gate.expired
gate.canceled
```

Payloads contain run/gate key, version, decision code, actor ID if approved by
privacy policy, timestamps, and decision ref digest. No free-form text.

Projection exposes counts and per-run bounded details: waiting age, expiry
remaining, required role identifier, status, decision time/code, and whether a
decision artifact exists. Operator listings are paginated.

## Failure vocabulary

- `validation/GATE_DECISION_SCHEMA`;
- `configuration/GATE_POLICY_INVALID`;
- `identity/GATE_VERSION_CONFLICT`;
- `authorization/GATE_DECISION_DENIED` belongs at operator service boundary and
  should not leak credential detail;
- `timeout/GATE_EXPIRED` or the repository's chosen stable timeout class;
- `identity/GATE_ALREADY_DECIDED`.

Add any new failure class deliberately to the closed vocabulary and tests.

## Migration

Add gate tables and indexes additively. Existing plans have no gates. Update
all node/run terminal-state queries if `waiting` enters node status. Old
attempt invariants remain unchanged because gates never create attempts.

Canonical gate fields require exact plan goldens and may require a reviewed
plan schema revision. Public help must distinguish gate waiting from a blocked
executable task.

## Test matrix

### Authoring/compiler

- direct Go/JavaScript equality and exact IR/plan/DTS goldens;
- duplicate gate keys, unknown dependency, invalid timeout/role/policy rejected;
- task runtimes cannot import operator module;
- unsupported branch policy rejected explicitly.

### Store/concurrency

- dependencies transition pending to waiting once;
- waiting creates no attempt, lease, resource use, or reservation;
- close/reopen preserves deadline/version;
- approve starts downstream work and publishes typed decision;
- reject applies exact compiled policy;
- maintenance expiry survives restart;
- approve/reject/expire/cancel races across connections have one winner;
- stale expected version rejected;
- repeated identical decision is idempotent;
- cancel prevents later approval/revival;
- migration from Slice 9 database.

### Security/privacy

- unauthorized operator request causes no mutation (operator integration test);
- credentials, headers, comments, and raw payload canaries absent from
  SQLite/WAL/events/projections;
- malformed/oversized actor, code, and decision ref rejected;
- signed decision ref verified when profile requires it.

### End to end

- approval workflow waits across restart with dispatcher serving other runs;
- capacity remains fully available while gate waits;
- budget exhaustion gate plus account increase continues exactly once;
- rejected and expired workflows terminate deterministically;
- attach/reopen projection reports exact waiting/decision state.

## Implementation sequence

1. Freeze gate IR/plan/DTS, state vocabulary, and policy.
2. Add schema/migration and pending-to-waiting transaction.
3. Add operator command types and approve/reject CAS.
4. Add expiry and cancellation races.
5. Integrate dependencies/output refs/downstream readiness.
6. Integrate Slice 9 approval exhaustion.
7. Add events/projections/help.
8. Add real wait/reopen/concurrency/privacy fixture.
9. Update diary, changelog, relations, and generated artifacts.
10. Run focused/race/full/TypeScript/docmgr/privacy/diff validation.

## Acceptance criteria

Slice 10 is complete only when gates wait indefinitely across process restart
without attempts, leases, resources, runtimes, or budgets; authenticated
versioned decisions and expiry/cancel races commit one terminal outcome;
approval continues downstream work exactly once; privacy boundaries hold; and
all prior slices plus focused/race/full validation pass.

## Alternatives rejected

- **Sleeping task:** holds scarce capacity and relies on lease renewal.
- **Polling task retries:** creates attempt noise, traffic, and delayed
  cancellation.
- **In-memory promise/channel:** disappears on restart.
- **Let task approve its own gate:** collapses authoring/execution/operator
  authority.
- **Persist arbitrary approval comments:** violates bounded privacy-safe control
  state; use a typed external artifact when evidence is required.

## References

- [All-slice intern guide](08-workflow-v3-slices-1-through-12-intern-architecture-and-analysis-guide.md)
- [Slice 9 budgets and projections](06-slice-9-budgets-and-projections-operational-control.md)
- [Primary architecture](01-durable-dataflow-workflow-v3-and-modern-scripting-architecture.md)
- [Diary](../reference/01-investigation-diary.md)
