---
Title: Slice 8 Registry Generations - Safe Durable Upgrades
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
    - Path: repo://pkg/workflowv3/bundle.go
      Note: Content-addressed bundle identity and source bytes
    - Path: repo://pkg/workflowv3/registry.go
      Note: |-
        Current immutable sealed registry and exact node resolution
        Current sealed exact registry is the rolling-generation baseline
    - Path: repo://pkg/workflowv3runtime/dispatcher.go
      Note: |-
        Generation-aware exact admission and acquisition
        Generation acquisition must be coherent with dispatch admission
    - Path: repo://pkg/workflowv3runtime/engine.go
      Note: Attempt execution currently bound to one registry pointer
    - Path: repo://pkg/workflowv3sqlite/schema.sql
      Note: Attempt generation evidence and optional worker advertisements
ExternalSources: []
Summary: Implementation contract for atomically activating immutable worker registry generations while old attempts and exact pinned plans continue safely and broken generations quarantine without consuming domain retries.
LastUpdated: 2026-07-21T20:40:00-04:00
WhatFor: Freeze candidate validation activation acquisition draining quarantine and recovery semantics before adding hot reload.
WhenToUse: Read before changing registry ownership worker advertisements reload behavior or attempt-to-generation binding.
---


# Slice 8 Registry Generations - Safe Durable Upgrades

## Executive summary

Slice 8 replaces the engine's single registry pointer with a manager for
immutable generations. A candidate is completely built, validated, self-tested,
and sealed before one atomic activation. Active attempts retain acquired old
generations. Plans continue to require exact bundle/entrypoint/ABI/module/policy
identity; “newest” is never a substitute.

A failed candidate leaves the active generation unchanged. Repeated runtime
construction failure quarantines the affected generation/implementation before
it burns task retry budgets.

## Current baseline and gap

`SealedRegistry` already clones state, validates module advertisements,
computes a deterministic generation digest, and resolves exact `PlanNode`
identity and policy. `Engine` currently holds exactly one `*SealedRegistry`.
Replacing that pointer directly would make lifetime, active attempt, and old
plan behavior implicit.

## Scope

This slice includes candidate construction, validation hooks, atomic activation,
exact resolution across retained generations, reference acquisition/release,
draining, quarantine, operator snapshots/events, and restart from immutable
bundle lock configuration.

Artifact signatures and remote distribution may remain host policy inputs if
not needed by the local fixture, but digest verification is mandatory. Process
isolation is Slice 11.

## Generation model

```go
type GenerationState string
const (
    GenerationActive GenerationState = "active"
    GenerationDraining GenerationState = "draining"
    GenerationQuarantined GenerationState = "quarantined"
)

type GenerationHandle interface {
    Generation() string
    Registry() *SealedRegistry
    Release()
}

type RegistryManager interface {
    AcquireNode(PlanNode) (RegisteredTask, GenerationHandle, error)
    BuildCandidate(context.Context, BundleLock) (*Candidate, error)
    Activate(context.Context, *Candidate) error
    Snapshot() RegistrySnapshot
}
```

Handles are idempotently releasable and cannot expose mutable generation
internals. Manager state changes under one lock or equivalent atomic snapshot;
task execution does not hold that lock.

## Candidate transaction

Candidate construction is outside active state:

1. read an immutable lock/config generation;
2. fetch local/remote bundle artifacts;
3. verify size and every file digest;
4. strictly decode manifests/catalogs;
5. validate unique exact identities and aliases;
6. resolve every entrypoint/export;
7. build only approved module profiles;
8. run deterministic registration self-tests in a registration-only runtime;
9. seal registry and compute generation digest;
10. compare digest with configured expectation when supplied.

No ordinary authoring/task `require()` can invoke this path.

`Activate` rechecks candidate validity and atomically marks the old active
generation draining and the candidate active. Failure before the swap changes
nothing. Activation emits bounded generation IDs and status, never source,
credentials, or stack traces.

## Exact acquisition and lease ordering

A plan pins exact implementation identity and policy. The manager searches
non-quarantined retained generations for an exact `ResolveNode` match.
Preference order is deterministic: active generation first, then retained
compatible generations by activation sequence.

Generation acquisition and durable leasing must be coherent. The selected
generation is acquired before a lease is exposed for execution, and the exact
generation digest is written to the attempt. If the store transaction fails,
release the handle. If runtime construction or execution begins, hold the
handle through terminal attempt persistence.

One acceptable API combines store candidate selection with a resolver callback
that returns an acquired generation token only for the final chosen row. Do not
acquire handles for an unbounded candidate scan and leak them.

## Pinning semantics

- Plan compiled with bundle A always requires A.
- Activating B does not rewrite A plans or pending nodes.
- A and B may register the same task kind/version only because their bundle
  identities differ.
- New plan compilation uses an explicitly selected catalog/generation; it does
  not read a changing global implicitly mid-compile.
- Attempt rows preserve the actual generation digest used.

## Draining and cleanup

The old active generation becomes draining immediately after activation. It may
serve exact A-pinned work while retention policy allows it. It cannot be freed
while any handle exists.

Cleanup prerequisites:

```text
reference count == 0
and no worker-local acquired lease waits to start
and configured plan-retention policy permits removal
and advertisement withdrawal is committed/published
```

Cleanup removes runtime/bundle caches but never deletes durable attempt
history. If an A-pinned run remains and A is removed by operator policy, its
projection becomes `implementation-unavailable`; B does not run it.

## Quarantine

Quarantine handles failures indicating worker implementation health rather than
domain input:

- runtime cannot be constructed;
- required advertised module factory is unavailable;
- bundle file disappears or fails digest verification;
- deterministic self-check fails after activation;
- repeated invariant-level internal failure reaches configured threshold.

Domain validation, provider 429, task timeout, and ordinary JavaScript typed
failures do not quarantine a generation.

Quarantine withdraws new admission for the affected generation or exact
implementation. Already running attempts are canceled only if the failure
means execution cannot remain trusted; otherwise they retain their handle.
Quarantine records stable code, count, threshold, and time. Reset requires an
authenticated operator action or activation of a verified replacement.

Pre-lease runtime construction should occur before consuming a domain attempt
when feasible. If construction fails after attempt creation, classify it as an
infrastructure admission outcome and ensure it does not exhaust the task's
semantic retry allowance.

## Durable worker advertisement

A single-process SQLite fixture can keep manager state in memory and preserve
generation digest in attempts. Multi-worker operation should add leased worker
advertisements:

```text
worker_id, generation_digest, state, heartbeat_expires_at
implementation identity rows
module aliases and isolation/resource tags
```

Advertisements are immutable per generation and expire with worker heartbeat.
They contain no module configuration secrets.

The scheduler still validates exact local manager resolution before leasing;
advertisements support routing/projection, not trust by themselves.

## Restart behavior

On restart the worker reloads the configured immutable bundle lock and rebuilds
generations deterministically. A running attempt from the dead process becomes
`lease_lost` through existing store logic. Pending A work is executable only if
A is present in the lock/retention set. The worker does not deserialize
executable registry objects from SQLite.

## Projection and events

Expose:

```text
active generation digest and activated time
draining generations and reference counts
quarantined generations/implementations and stable reason codes
exact implementation availability counts
candidate reload last success/failure time
```

Events: `registry.activated`, `registry.draining`, `registry.quarantined`, and
`registry.removed`, all with bounded identity metadata.

## Migration

Current attempt rows already store `registry_generation`. Add worker generation
or quarantine tables only if needed for cross-process routing. Migration is
additive. Existing attempts remain valid historical evidence even when the
corresponding in-memory generation is no longer configured.

Engine and dispatcher APIs may accept a small resolver interface so tests and
single-generation callers remain straightforward, but do not add a silent
compatibility adapter that substitutes the current registry.

## Test matrix

- deterministic candidate digest from identical lock and bytes;
- wrong file digest, entrypoint, module, duplicate identity, or self-test fails
  without changing active generation;
- attempt A starts, B activates, A finishes using A;
- new B plan executes B bytes;
- pending A plan executes from retained draining A;
- removing A makes A pending work implementation-unavailable, never B-run;
- activation races acquisition and each attempt records one coherent digest;
- concurrent handles prevent cleanup until final release;
- double release is safe or rejected deterministically;
- repeated construction failure quarantines without consuming domain retries;
- domain task failures do not quarantine;
- restart rebuilds exact generations from lock;
- source/module configuration secrets absent from SQLite/events;
- race detector covers manager acquire/activate/release/snapshot.

## Implementation sequence

1. Add immutable `RegistryManager` and single-generation adapter only as an
   explicit implementation of the interface, not identity substitution.
2. Add candidate/lock validation and self-test hooks.
3. Add atomic activation and handle reference counting.
4. Integrate generation-aware lease acquisition and attempt evidence.
5. Add draining/removal and exact unavailable projections.
6. Add quarantine classification and threshold behavior.
7. Add optional durable advertisements required by multi-worker fixture.
8. Update help, DTS if exposed, diary, changelog, and generated artifacts.
9. Run race, focused, migration, full validation, privacy, and diff checks.

## Acceptance criteria

Slice 8 is complete only when old and new executable bytes coexist safely,
active attempts cannot change generation, exact old plans are never
substituted, failed reload leaves prior service intact, quarantine protects
domain retries, cleanup waits for references, restart is deterministic, and
all focused/race/full/documentation/privacy checks pass while Slices 1–7 remain
unchanged.

## Alternatives rejected

- **Replace one global pointer and discard old code:** active and pending old
  work become nondeterministic or unavailable without controlled draining.
- **Resolve kind/version from latest:** violates exact durable identity.
- **Mutate a registry in place:** readers can observe partial reload.
- **Treat construction failure as ordinary task retry:** burns domain retry
  budgets for worker configuration defects.
- **Persist executable runtime state:** unsafe and unnecessary; rebuild from
  verified immutable artifacts.

## References

- [Bundle and registry design](02-reproducible-javascript-task-bundles-and-worker-registries.md)
- [All-slice intern guide](08-workflow-v3-slices-1-through-12-intern-architecture-and-analysis-guide.md)
- [Diary](../reference/01-investigation-diary.md)
