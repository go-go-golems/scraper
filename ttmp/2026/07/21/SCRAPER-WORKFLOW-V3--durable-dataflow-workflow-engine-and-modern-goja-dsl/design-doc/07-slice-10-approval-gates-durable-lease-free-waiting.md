---
Title: Slice 10 Approval Gates - Durable Lease-Free Waiting
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
    - Path: repo://pkg/gojamodules/workflow/authoring.go
      Note: Safe gate declaration without operator authority
    - Path: repo://pkg/workflowv3/compiler.go
      Note: Gate dependency output budget relation cycle and expiry validation
    - Path: repo://pkg/workflowv3/gate.go
      Note: Canonical policy command validation errors and progress contracts
    - Path: repo://pkg/workflowv3/types.go
      Note: Canonical gate declarations policies and decisions
    - Path: repo://pkg/workflowv3runtime/dispatcher.go
      Note: Gates must wait without dispatcher lease or resource use
    - Path: repo://pkg/workflowv3sqlite/gate.go
      Note: Waiting decision expiry pagination and terminal transactions
    - Path: repo://pkg/workflowv3sqlite/projection.go
      Note: Run and attempt projections integrated with gate state
    - Path: repo://pkg/workflowv3sqlite/schema.sql
      Note: Durable waiting and decision records
    - Path: repo://pkg/workflowv3sqlite/store.go
      Note: Gate submission cancellation compact outputs and ordinary node admission
    - Path: repo://pkg/testfixtures/workflowv3gate/workflow.js
      Note: Real typed wait restart and downstream continuation workflow
    - Path: repo://pkg/workflowv3runtime/gate_integration_test.go
      Note: Dispatcher capacity authority restart and privacy evidence
ExternalSources: []
Summary: Implementation contract for durable approval and external-event gates that wait across restart without consuming attempts leases resources runtimes or budgets.
LastUpdated: 2026-07-22T00:35:00-04:00
WhatFor: Define and record the implemented Slice 10 authoring state operator transaction race privacy and continuation contracts and evidence.
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

Included: canonical gate declarations, independent durable gate control rows,
waiting state, expiry, approve/reject/cancel CAS transitions, authenticated
operator boundary, externally verified immutable decision artifact refs,
continuation, events, projections, restart, and concurrency tests.

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
  budget_activation INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (run_id,gate_key),
  FOREIGN KEY (run_id) REFERENCES v3_runs(run_id) ON DELETE CASCADE
);
```

The implementation deliberately keeps gates out of `v3_nodes`: immutable
`v3_gate_dependencies` and `v3_gate_consumers` edges connect them to ordinary
nodes without contaminating executable-node status or attempt invariants. A
gate never has a `v3_attempts` row.

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
    AuthorizedRole string
    DecisionRef *ArtifactRef
}
```

Actor and decision codes use strict bounded safe syntax. The store revalidates
command shape, current waiting state, expected version, deadline, run status,
and decision artifact schema. Authorization credentials never reach this
struct or SQLite.

For high-assurance environments, the operator service can create and verify a
signed decision artifact before calling the store. Signature-profile selection
and identity-provider integration remain operator-service concerns; the store
persists only the already-verified compact ref and bounded identity facts.

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

Slice 9 `require-approval` exhaustion activates a compiled gate. Each compiled
claim owns a dedicated activation gate; sharing one terminal gate across claims
is rejected because it could not be re-armed for later exhaustion. A gate also
cannot depend on its blocked node or expose its activation decision as ordinary
data. No budgeted node lease exists while waiting. Approval must be paired with
an authenticated account increase or explicit one-time allowance in the same
operator workflow; otherwise the node remains budget-blocked. The gate decision
itself does not fabricate budget.

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
decision artifact exists. Global snapshots cap details and mark truncation;
`GatePage` provides deterministic per-run keyset pagination.

## Failure vocabulary

- `validation/GATE_DECISION_SCHEMA`;
- `configuration/GATE_POLICY_INVALID`;
- `identity/GATE_VERSION_CONFLICT`;
- `authorization/GATE_DECISION_DENIED` belongs at operator service boundary and
  should not leak credential detail;
- `timeout/GATE_EXPIRED` or the repository's chosen stable timeout class;
- `identity/GATE_ALREADY_DECIDED`.

Gate commands are trusted control-plane calls rather than task failures, so the
implementation returns stable typed sentinel errors (`ErrGateVersionConflict`,
`ErrGateAlreadyDecided`, `ErrGateExpired`, `ErrGateUnauthorized`) instead of
persisting synthetic task failures. Terminal reject/expiry codes are bounded
engine-owned gate events and run/node failure evidence.

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

## Implementation evidence

The implementation is in `pkg/workflowv3/gate.go`, canonical compiler/types,
`pkg/gojamodules/workflow`, `pkg/workflowv3sqlite/gate.go`, gate-aware store and
dynamic map/reduction insertion, runtime dispatcher maintenance, and the real
`pkg/testfixtures/workflowv3gate` bundle. Exact JavaScript IR/plan/DTS goldens
freeze authoring output.

Focused tests prove pending-to-waiting CAS across independent connections,
zero attempts while waiting, close/reopen approval, restart-preserved expiry,
role/schema/version/idempotency checks, approve/reject/expire/cancel races,
typed downstream artifact publication, budget approval plus account increase,
unused budget-gate nonblocking behavior, bounded gate pagination, task denial of
operator modules, SQLite/WAL/event/projection canaries, and a dispatcher restart
that continues unrelated work while the gate waits. Branch cancellation is
explicitly compiler-rejected. Cancellation-context normalization prevents
SQLite transaction shutdown races from masking `context.Canceled`.

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
