---
Title: Queue rate limiter analysis and implementation guide
Ticket: SCRAPER-RATE-LIMITER
Status: active
Topics:
    - scraper
    - scheduler
    - rate-limiting
    - token-bucket
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/engine/model/types.go
      Note: Queue identity and durable execution types that the future policy should extend
    - Path: pkg/engine/scheduler/scheduler.go
      Note: Current scheduler loop and candidate leasing behavior analyzed for future rate limiter integration
    - Path: pkg/engine/store/sqlite/store.go
      Note: SQLite queue candidate and lease logic are the main implementation sites for atomic token consumption
    - Path: pkg/engine/store/store.go
      Note: Store contract currently lacks queue policy and durable limiter state hooks
ExternalSources: []
Summary: Detailed design guide for adding durable per-queue token-bucket rate limiting to the scraper scheduler during the current fresh-schema prototyping phase, with current-state analysis, API sketches, phased implementation guidance, and testing strategy.
LastUpdated: 2026-03-23T22:15:00-04:00
WhatFor: Explains how to evolve the current one-active-op-per-queue scheduler into a configurable durable rate limiter that supports token-bucket semantics for HTTP and JS queues.
WhenToUse: Use when implementing or reviewing queue-level rate limiting, token-bucket state, queue policy configuration, or scheduler/store changes related to pacing work.
---


# Queue Rate Limiter Analysis and Implementation Guide

## Executive Summary

The scraper engine already treats each `site + queue` pair as a queue domain, but the current behavior is only queue serialization, not a real rate limiter. In the current implementation, the scheduler finds leaseable queues, then leases at most one ready op from each queue when no non-expired lease already exists for that queue. That is enough for milestone-one safety, but it does not satisfy the broader architecture goal described in the imported design notes, where rate limiting is supposed to be a first-class scheduling concern rather than an incidental byproduct of queue naming.

This ticket proposes the next step: keep queue keys as the routing primitive, but add durable, configurable token-bucket rate limiting on top of them. The recommended shape is:

1. Keep `QueueKey` on each op as the stable routing key.
2. Add queue policy configuration that says how a queue behaves.
3. Persist queue limiter state in SQLite so multiple workers and restarts behave consistently.
4. Make lease acquisition consume limiter capacity atomically in the same transaction that leases an op.
5. Keep concurrency control and rate control separate, because they solve different problems.

For a new intern, the most important mental model is this:

- queue identity answers "which work shares a pacing domain?"
- concurrency answers "how many ops may be active at once?"
- rate limiting answers "how fast may new work start over time?"

The current code only implements the first part plus a fixed concurrency limit of one active lease per queue. This guide explains how to add the missing third part without breaking the current scheduler model.

## Implementation Status

This ticket is no longer design-only. The core queue limiter described here was implemented in commit `263b2dc` (`Add durable queue rate limiter`) and validated with the full Go test suite plus command-path integration tests.

What now exists in the codebase:

- `pkg/engine/model/types.go` defines `RateLimitKind`, `RateLimitPolicy`, and `QueuePolicy`, plus normalization rules.
- `pkg/engine/store/store.go` threads resolved queue policy into `LeaseRequest`.
- `pkg/engine/store/sqlite/migrations/002_engine_runtime.sql` creates durable `queue_limit_state`.
- `pkg/engine/store/sqlite/store.go` enforces `MaxInFlight` and token-bucket consumption atomically inside `LeaseReadyOp`.
- `pkg/engine/scheduler/scheduler.go` resolves queue policy per `site + queue`, emits `queue_rate_limited`, and can process multiple ops from the same queue in one cycle when the policy allows it.
- `pkg/sites/registry/registry.go` lets sites declare queue policies, and the worker plus local runner paths now install that provider automatically.
- `pkg/cmd/site_test.go` proves the durable path end to end with `site js-demo run seed`, `engine status`, and `worker run` under a custom `js-demo` token-bucket policy.
- `pkg/cmd/site_test.go` also proves the same policy plumbing works for an HTTP-backed site through a rate-limited Hacker News fixture run.

Deliberate non-goals of this slice:

- No built-in site ships with a non-default queue policy yet. The hook is implemented, but the default site definitions remain conservative so existing fixture and smoke flows do not become time-sensitive unexpectedly.
- There is no dedicated admin command yet for inspecting token counts in `queue_limit_state`.
- The scheduler currently emits one generic `queue_rate_limited` event when a queue candidate cannot lease under policy, but it does not yet distinguish `max_in_flight` blocking from token exhaustion in operator-facing output.

Validation completed for this implemented slice:

- `go test ./... -count=1`
- `go test ./pkg/cmd -count=1`
- `go test ./pkg/engine/store/sqlite ./pkg/engine/scheduler -count=1`

The most important practical outcome is that rate limiting is now a durable scheduler concern rather than an accidental side effect of queue serialization.

## Problem Statement

### What the design intended

The imported source document for the main scraper design explicitly treats rate limiting as part of scheduling, not as site-specific logic hidden in JS or HTTP runners. It says that jobs belong to queues keyed by domain, site, account, proxy pool, or job class, and that queue-based rate limiting is part of the architecture rather than a side concern in the scraper runtime. See [sources/local/scraper.md](../sources/local/scraper.md), especially the notes around queue-keyed rate limiting and Go-owned rate limits.

### What the code currently does

The current scheduler config in [scheduler.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go) only has:

- `MaxWorkers`
- `PollInterval`
- `DefaultLeaseDuration`

There is no rate-limit configuration in `scheduler.Config`. See [scheduler.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go#L16).

The scheduler loop currently:

1. refreshes runnable ops,
2. lists queue candidates from the store,
3. iterates candidates,
4. tries to lease one op per queue,
5. executes the leased op immediately.

See [scheduler.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go#L175).

The store contract only exposes queue candidates by `site` and `queue`. It does not expose any rate-limit state or policy. See [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/store.go#L15).

The SQLite store’s queue candidate query simply filters for ready ops whose queue has no active lease. See [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go#L320). The lease query repeats that same "no active lease in this queue" guard before selecting one ready op. See [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go#L395).

### What the tests currently prove

The scheduler test [scheduler_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler_test.go#L304) proves one-op-at-a-time queue serialization for a synthetic runner. The `js-demo` test [site_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site_test.go#L192) proves that real JS ops participate in the same queue-domain serialization.

Those tests are useful, but they do not prove any of the following:

- 2 requests per second
- burst capacity of 5 then refill
- "wait 500ms before another op in this queue may start"
- durable limiter behavior across process restarts
- fair or consistent rate pacing across multiple workers

### Why the gap matters

Without a real rate limiter:

- a queue can still start one op every scheduler cycle if the op finishes fast enough,
- the effective request rate depends on `PollInterval` and op runtime duration,
- there is no way to model burst capacity cleanly,
- there is no operator-visible place to say "this queue runs at 0.5 ops/second",
- future HTTP and JS queues cannot share a durable pacing model.

For a scraping engine, that is an important missing piece because production scrape behavior usually depends on explicit pacing guarantees rather than on incidental queue serialization.

## Scope

### In scope

- durable per-queue rate-limiter state
- configurable queue policies
- token-bucket style rate limiting
- maintaining the current queue identity model
- making the solution work for both HTTP and JS ops
- scheduler/store changes required to enforce rate limits across multiple workers
- tests for timing-independent limiter semantics

### Out of scope

- browser-specific throttling
- proxy-pool orchestration
- global multi-site fairness policies
- human-facing dashboard work
- fully user-editable runtime config files
- changing how JS op code itself performs extraction

## Current-State Architecture

### Core types already present

The durable engine data model already has the right identity primitives:

- `SiteName`
- `QueueKey`
- `OpSpec.Queue`
- `Lease`
- `RetryPolicy`
- `RetryState`

See [types.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/model/types.go#L8).

That means the engine does not need a new concept for "which pacing domain does this op belong to?" The queue key already answers that.

### Current queue flow

The present flow is:

```text
ready ops in DB
    |
    v
ListQueueCandidates(now)
    |
    v
for each site+queue candidate
    |
    v
LeaseReadyOp(site, queue, now)
    |
    +-- if active lease exists in same queue: skip
    +-- else pick oldest ready op
    +-- mark it running
    +-- write lease row
```

This behavior is simple and safe, but it implements only one queue invariant:

- at most one active lease per queue

It does not implement a pacing invariant like:

- at most `N` starts per second
- at most `B` burst tokens

### Evidence from SQLite queries

The queue-candidate query:

```sql
SELECT DISTINCT ops.site, ops.queue_key
FROM ops
WHERE ops.status = 'ready'
  AND ...
  AND NOT EXISTS (
    SELECT 1
    FROM leases l
    JOIN ops active ON active.id = l.op_id
    WHERE l.expires_at > ?
      AND active.site = ops.site
      AND active.queue_key = ops.queue_key
  )
```

Source: [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go#L320)

This query filters on active leases only. There is no limiter table, no token count, and no next-eligible-at field.

The lease query repeats the same constraint:

```sql
AND NOT EXISTS (
  SELECT 1
  FROM leases l
  JOIN ops active ON active.id = l.op_id
  WHERE l.expires_at > ?
    AND active.site = ops.site
    AND active.queue_key = ops.queue_key
)
```

Source: [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go#L415)

That is strong evidence that the current store knows only how to serialize queue execution, not how to rate-limit it.

### Existing design notes already point at the next step

The main design guide already says queue semantics should model shared site rate limits and related pacing domains, but it explicitly records the current implementation note as "one active rate domain per `site + queue` pair." See [01-generic-go-scraper-engine-and-nereval-port-design-guide.md](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/design-doc/01-generic-go-scraper-engine-and-nereval-port-design-guide.md#L486).

This ticket is therefore not inventing a new direction. It is closing a known phase gap between the broader architecture and the current milestone-one implementation.

## Actual Implementation Shape

The final implementation differed in a few useful ways from the earlier draft.

### Queue policy lives in the engine model

`QueuePolicy` and `RateLimitPolicy` now live in [types.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/model/types.go), not in a dedicated policy package. That keeps the store, scheduler, and site registry on one shared contract without introducing another package boundary during prototyping.

### Site registration owns static policy declaration

Site definitions can now declare:

```go
QueuePolicies map[model.QueueKey]model.QueuePolicy
```

in [registry.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/registry/registry.go). The registry exposes a scheduler-ready provider function, so worker setup and local runners do not need to know policy details beyond "ask the site registry."

### Scheduler resolves policy, store enforces it

The intended boundary held up in implementation:

- the scheduler resolves queue policy and decides how many times to try leasing from a queue in a cycle
- the store performs active-lease checks, token refill, token consumption, and lease creation atomically

That split keeps the rate limiter testable.

### No persisted `available_at`

The implementation stores:

- `tokens`
- `last_refill_at`

and derives availability from them. That kept the schema smaller and avoided needing to keep a second persisted time field consistent with the token math.

### Queue candidate listing is intentionally broader now

`ListQueueCandidates` no longer filters out queues just because they already have an active lease. That check moved into `LeaseReadyOp`, because once `MaxInFlight` becomes configurable the old query-level "no active lease at all" rule is too restrictive.

That means the candidate list is now:

- simpler
- compatible with `MaxInFlight > 1`
- still race-safe because lease enforcement stays transactional

## Current Flow

The implemented flow is:

```text
ready ops in DB
    |
    v
ListQueueCandidates(now)
    |
    v
for each queue candidate
    |
    +-- scheduler resolves QueuePolicy(site, queue)
    |
    +-- repeat up to min(MaxInFlight, remaining worker slots)
           |
           v
      LeaseReadyOp(site, queue, policy, now)
           |
           +-- count active leases for queue
           +-- reject if active >= MaxInFlight
           +-- load queue_limit_state
           +-- refill tokens from elapsed time
           +-- reject if tokens < 1
           +-- pick oldest ready op
           +-- consume one token
           +-- persist queue_limit_state
           +-- create lease
           +-- mark op running
```

This is the core invariant to preserve if the implementation is extended later.

## Gap Analysis

### Gap 1: queue identity exists, queue policy does not

The op model says which queue an op belongs to, but there is no durable or configured policy for how that queue should behave.

Missing concepts:

- `max_in_flight`
- `rate_per_second`
- `burst`
- limiter type

### Gap 2: no durable limiter state

If a token bucket is implemented only in memory:

- workers disagree after restart,
- multiple workers cannot coordinate,
- smoke tests may pass while production behavior is wrong.

The scheduler/store architecture is already durable-first, so limiter state should follow that same pattern.

### Gap 3: list-then-lease must remain race-safe

The current scheduler first lists queue candidates and then leases an op. That is acceptable for queue serialization because the lease transaction repeats the no-active-lease check. For token-bucket semantics, the same pattern must also prevent double token consumption across workers.

That means rate-limit gating must happen in the same transaction as lease acquisition.

### Gap 4: concurrency and rate are currently conflated

Today, "one active op per queue" acts as both:

- queue-domain concurrency control
- accidental pacing

Those are not the same thing. A queue may legitimately need:

- `max_in_flight = 1`, `rate = 0.5/sec`
- `max_in_flight = 4`, `rate = 2/sec`
- `max_in_flight = 1`, `rate = unlimited`

The design needs separate knobs for each.

## Design Goals

1. Preserve the existing durable scheduler/store architecture.
2. Keep queue identity as the central routing primitive.
3. Make rate limiting configurable and durable.
4. Keep behavior deterministic enough to test without flaky wall-clock assertions.
5. Support both HTTP and JS queues.
6. Avoid putting rate-limit logic inside runners.
7. Keep the first implementation understandable to a new intern.

## Proposed Architecture

### High-level recommendation

Use three layers:

1. `QueueKey` on ops for identity
2. `QueuePolicy` in Go config/registry for static policy
3. durable `queue_limit_state` rows in SQLite for mutable runtime state

### Recommended mental model

```text
OpSpec.Queue
    |
    v
QueuePolicyProvider
    |
    v
LeaseReadyOp(..., policy, now)
    |
    +-- enforce max_in_flight
    +-- refill token bucket
    +-- consume token if allowed
    +-- lease one op atomically
```

### Recommended API surface

The smallest coherent addition is a queue policy type. One possible shape is:

```go
type QueuePolicy struct {
    MaxInFlight int
    RateLimit   *RateLimitPolicy
}

type RateLimitPolicy struct {
    Kind          RateLimitKind
    RatePerSecond float64
    Burst         int
}

type RateLimitKind string

const (
    RateLimitKindTokenBucket RateLimitKind = "token_bucket"
)
```

Recommended defaults:

- `MaxInFlight = 1` to preserve current behavior unless overridden
- `RateLimit = nil` means no token-bucket pacing

This preserves backward compatibility:

- current queues still serialize if no custom policy is defined
- new queues can opt into stronger pacing

### Where queue policy should live

Recommended first version:

- put the durable runtime state in the store
- put static policy in Go configuration, not in SQLite

Why:

- the limiter state changes continuously and belongs in durable runtime storage
- the policy is deployment/site intent and can start as code-configured data
- changing the rate from code or site registration is simpler than building a full admin config UX first

Possible home for the policy type:

- `pkg/engine/model`
- or a new `pkg/engine/policy`

I recommend `pkg/engine/model` for the first version because queue policy is conceptually part of the durable execution model, and keeping the type close to `QueueKey` makes the code easier to navigate.

### How sites should supply policy

A clean operator-facing shape is for sites to declare policies for their known queues.

Example:

```go
type Definition struct {
    ...
    QueuePolicies map[model.QueueKey]model.QueuePolicy
}
```

Examples:

```go
"site:hackernews:http": {
    MaxInFlight: 1,
    RateLimit: &model.RateLimitPolicy{
        Kind:          model.RateLimitKindTokenBucket,
        RatePerSecond: 0.5,
        Burst:         1,
    },
},
"site:js-demo:js": {
    MaxInFlight: 1,
    RateLimit: &model.RateLimitPolicy{
        Kind:          model.RateLimitKindTokenBucket,
        RatePerSecond: 10,
        Burst:         2,
    },
},
```

The scheduler should also allow a default policy provider so queues without explicit site policies fall back to the current behavior.

### Durable limiter state

Recommended table shape for the current prototyping phase:

```sql
CREATE TABLE queue_limit_state (
  site TEXT NOT NULL,
  queue_key TEXT NOT NULL,
  tokens REAL NOT NULL,
  last_refill_at TEXT NOT NULL,
  available_at TEXT,
  updated_at TEXT NOT NULL,
  PRIMARY KEY(site, queue_key)
);
```

What each field is for:

- `tokens`: current bucket balance after last persisted update
- `last_refill_at`: time used to compute refills on next lease attempt
- `available_at`: optional derived field for visibility and cheap filtering
- `updated_at`: bookkeeping and debugging

Why `REAL` for tokens:

- token buckets often use fractional refill math
- for example 0.5 req/sec is easier to represent with fractional token accumulation

### Why state must be durable

The store already owns:

- leases
- retries
- ready/running state
- results and artifacts

Rate-limit state belongs with those because it participates in distributed correctness. If the limiter state resets on process restart, the engine can burst more aggressively than intended after each deploy or crash.

## Detailed Lease Flow

### Before

```text
1. scheduler asks store for queue candidates
2. scheduler asks store to lease one op for a queue
3. store checks for active lease in queue
4. store leases oldest ready op
```

### After

```text
1. scheduler asks store for queue candidates
2. scheduler resolves queue policy for each candidate
3. scheduler asks store to lease one op for a queue under that policy
4. store transaction:
   - load or initialize queue limiter state
   - count active leases in queue
   - reject if active leases >= max_in_flight
   - refill token bucket from last_refill_at to now
   - reject if token balance < 1
   - select oldest ready op
   - consume one token
   - write limiter state
   - write lease row
   - mark op running
5. scheduler executes leased op
```

### Pseudocode for atomic lease-and-consume

```go
func LeaseReadyOp(ctx context.Context, req LeaseRequest) (*OpSpec, *Lease, error) {
    tx := beginTx()

    active := countActiveLeases(tx, req.Site, req.Queue, req.Now)
    policy := req.QueuePolicy.Normalize()
    if active >= policy.MaxInFlight {
        rollback()
        return nil, nil, nil
    }

    state := loadOrInitQueueState(tx, req.Site, req.Queue, req.Now, policy)
    state = refill(state, req.Now, policy)

    if policy.RateLimit != nil && state.Tokens < 1.0 {
        persistState(tx, state)
        commit()
        return nil, nil, nil
    }

    op := selectOldestReadyOp(tx, req.Site, req.Queue, req.Now)
    if op == nil {
        rollback()
        return nil, nil, nil
    }

    if policy.RateLimit != nil {
        state.Tokens -= 1.0
        state.AvailableAt = computeAvailableAt(state, req.Now, policy)
        persistState(tx, state)
    }

    lease := insertLease(tx, op, req)
    markRunning(tx, op)
    commit()
    return op, lease, nil
}
```

### Refill algorithm

```go
func refill(state QueueLimitState, now time.Time, policy QueuePolicy) QueueLimitState {
    if policy.RateLimit == nil {
        return state
    }

    elapsed := now.Sub(state.LastRefillAt).Seconds()
    state.Tokens = min(
        float64(policy.RateLimit.Burst),
        state.Tokens + elapsed*policy.RateLimit.RatePerSecond,
    )
    state.LastRefillAt = now
    return state
}
```

### Why consume on start, not on finish

Rate limiting should govern when new work starts. Waiting until finish would make long-running ops distort pacing incorrectly and would not match normal token-bucket semantics.

## Store Contract Changes

### Recommended contract extension

Current `LeaseRequest` only carries queue identity and lease timing. It should carry queue policy too.

Proposed sketch:

```go
type LeaseRequest struct {
    WorkerID      string
    Queue         model.QueueKey
    Site          model.SiteName
    LeaseDuration time.Duration
    Now           time.Time
    QueuePolicy   model.QueuePolicy
}
```

This keeps the store in charge of atomic enforcement but lets the scheduler remain the place where policy is resolved.

### Optional visibility methods

For operator tooling later, consider adding:

```go
type QueueLimitStatus struct {
    Site         model.SiteName
    Queue        model.QueueKey
    Tokens       float64
    LastRefillAt time.Time
    AvailableAt  *time.Time
    ActiveLeases int
}

GetQueueLimitStatus(ctx, site, queue) (*QueueLimitStatus, error)
ListQueueLimitStatus(ctx) ([]QueueLimitStatus, error)
```

This is not required for the first implementation but it will make debugging much easier.

## Scheduler Changes

### Scheduler responsibilities should remain small

The scheduler should not implement token math itself. It should:

1. get queue candidates,
2. resolve policy,
3. pass policy into the store,
4. emit events.

This keeps the store responsible for atomic correctness and the scheduler responsible for orchestration.

### Suggested scheduler additions

```go
type QueuePolicyProvider func(site model.SiteName, queue model.QueueKey) model.QueuePolicy

type Scheduler struct {
    ...
    queuePolicyProvider QueuePolicyProvider
}

func (s *Scheduler) SetQueuePolicyProvider(provider QueuePolicyProvider) {
    s.queuePolicyProvider = provider
}
```

Default behavior:

- if no provider is set, use `{MaxInFlight: 1, RateLimit: nil}`

This preserves current semantics without requiring every caller to understand the new feature immediately.

### Event model

The current scheduler events are enough for workflow lifecycle, but rate limiting would benefit from one extra event:

```go
const EventQueueRateLimited EventKind = "queue_rate_limited"
```

That event would be emitted when:

- a queue has ready work,
- the queue policy allows more concurrency,
- but token-bucket state prevents a lease at `now`

This makes operator logs much easier to read than repeated "no leaseable queues" messages.

## Schema Strategy During Prototyping

The team is still in a prototyping phase and is comfortable wiping state and starting fresh. That changes the implementation recommendation substantially:

- do not spend time designing or testing upgrade migrations for this limiter slice
- update the engine schema directly to the new desired shape
- clear existing prototype databases during rollout if needed

That means the first limiter implementation can assume a fresh database bootstrap path instead of a production upgrade path.

Recommended approach:

1. update the engine schema files directly to include `queue_limit_state`
2. treat the resulting schema as the new baseline
3. delete and recreate local prototype databases as needed
4. postpone migration-upgrade compatibility work until the engine exits prototyping

Suggested schema addition:

```sql
CREATE TABLE queue_limit_state (
  site TEXT NOT NULL,
  queue_key TEXT NOT NULL,
  tokens REAL NOT NULL,
  last_refill_at TEXT NOT NULL,
  available_at TEXT,
  updated_at TEXT NOT NULL,
  PRIMARY KEY(site, queue_key)
);

CREATE INDEX idx_queue_limit_available
  ON queue_limit_state(available_at);
```

This is intentionally a prototype-era recommendation. Once the engine becomes less disposable, the team should revisit a proper migration story.

## Testing Strategy

### Unit tests for pure refill math

Add tests for:

- exact refill to burst cap
- fractional refill
- no refill past burst
- consume one token
- compute next available time

These should be pure Go tests with a fake clock, not database tests.

### Store tests

Add SQLite integration tests for:

1. queue with `RateLimit=nil` behaves exactly like today
2. queue with `Burst=1`, `RatePerSecond=1` allows one lease and rejects another until refill
3. queue with `Burst=2` allows two starts if `MaxInFlight` also permits it
4. limiter state survives store reopen
5. two lease attempts at the same logical time do not both consume the same token

### Scheduler tests

Add scheduler tests that prove:

- rate-limited queue emits `Processed == 0` before refill time
- later cycle processes one op after refill
- different queues remain independent
- JS and HTTP queues both use the same limiter path

### Site-level tests

Re-use:

- `js-demo` for JS queues
- `hackernews` or `slashdot` for HTTP queues

The new `js-demo` queue serialization test is a good baseline. It should stay even after the token bucket is added, because it still documents the `MaxInFlight=1` default path.

## Recommended Phased Implementation

### Phase 1. Analysis and contract preparation

1. Add `QueuePolicy`, `RateLimitPolicy`, and related types to the engine model.
2. Add scheduler support for a queue policy provider with backward-compatible defaults.
3. Add store contract fields so lease requests can carry queue policy.

### Phase 2. Durable state and store enforcement

1. Update the baseline engine schema to include `queue_limit_state`.
2. Implement lazy load/init of queue limiter state.
3. Implement refill and consume logic inside `LeaseReadyOp`.
4. Preserve current semantics when `RateLimit=nil`.
5. Keep the prototype workflow simple: clear and recreate local DBs if the schema changes during development.

### Phase 3. Scheduler observability

1. Add queue rate-limited events.
2. Add status helpers or debug commands if useful.
3. Ensure logs make it obvious when a queue is blocked by limiter state rather than by active leases.

### Phase 4. Site integration

1. Let built-in sites define queue policies.
2. Start with conservative HTTP queue settings.
3. Add an explicit `js-demo` token-bucket example to prove JS queues use the same mechanism.

### Phase 5. Validation

1. Add store-level and scheduler-level tests.
2. Add site-level smoke tests where appropriate.
3. Verify restart behavior by reopening SQLite between lease attempts.
4. Verify clean bootstrap on a freshly created engine DB, since that is the intended prototype rollout path.

## Intern Walkthrough: How To Read The Code Before Implementing

Read these files in order:

1. [types.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/model/types.go)
   - Learn the durable model vocabulary.
2. [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/store.go)
   - Learn what the scheduler expects from persistence.
3. [scheduler.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go)
   - Learn the current worker loop.
4. [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go)
   - Learn where queue selection and lease acquisition happen.
5. [scheduler_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler_test.go)
   - Learn what behavior is already guaranteed.
6. [site_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site_test.go)
   - Learn how a real JS queue currently behaves under the scheduler.
7. [01-generic-go-scraper-engine-and-nereval-port-design-guide.md](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/design-doc/01-generic-go-scraper-engine-and-nereval-port-design-guide.md)
   - Learn the broader architectural intent.

## Alternatives Considered

### Alternative 1: keep queue serialization only

Pros:

- simplest
- already implemented
- easy to reason about

Cons:

- pacing depends on runtime duration and poll interval
- no burst behavior
- cannot express low fixed RPS cleanly
- does not satisfy the intended architecture well

Recommendation: reject as the final design, keep as the default fallback behavior.

### Alternative 2: in-memory token buckets only

Pros:

- easy to implement
- low code churn

Cons:

- wrong for multi-worker use
- wrong after restart
- inconsistent with the durable engine model

Recommendation: reject, even in prototyping, because the rest of the engine is already durable-first and this state affects correctness across restarts.

### Alternative 3: store only `next_allowed_at` and no token balance

Pros:

- simpler than full token bucket
- easy for strict "one every N milliseconds" pacing

Cons:

- cannot express burst capacity naturally
- less aligned with the requested token-bucket direction

Recommendation: possible fallback if implementation complexity becomes a problem, but token bucket is the better target.

### Alternative 4: put rate limiting inside HTTP runner only

Pros:

- could solve the immediate HTTP problem faster

Cons:

- does not work for JS queues
- duplicates scheduling logic in runners
- violates the architecture where Go scheduling owns rate limits

Recommendation: reject.

## Risks

### Risk 1: race conditions around token consumption

Mitigation:

- do rate-limit checks and token consumption inside the same transaction as lease acquisition

### Risk 2: flaky tests if wall-clock time is used badly

Mitigation:

- use scheduler/store clocks that can be controlled in tests
- favor logical time jumps over `time.Sleep`

### Risk 3: overcomplicating site configuration too early

Mitigation:

- start with code-defined queue policies
- postpone admin UX until core behavior is correct

### Risk 4: conflating concurrency with rate

Mitigation:

- keep `MaxInFlight` and `RateLimit` as separate fields

## Open Questions

1. Should the first implementation allow `MaxInFlight > 1`, or should it keep concurrency fixed at `1` and add only token-bucket pacing first?
2. Should queue policies live directly in site registration definitions, or should they come from a higher-level config layer later?
3. Do we want `queue_limit_state` visibility commands in the same implementation slice, or only after limiter correctness is in place?
4. Is `available_at` worth persisting in the first prototype schema, or should it remain a derived debug field only?

## Suggested Task Breakdown

```text
Phase 1: contracts
  - add model types
  - add scheduler provider hook
  - extend store lease request

Phase 2: durable store
  - update baseline schema
  - add refill/consume logic
  - preserve no-policy behavior

Phase 3: tests
  - pure math tests
  - SQLite store tests
  - scheduler tests
  - site-level tests

Phase 4: observability
  - add rate-limited event
  - add optional status helpers
```

## References

- [scheduler.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go)
- [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/store.go)
- [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go)
- [types.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/model/types.go)
- [scheduler_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler_test.go)
- [site_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site_test.go)
- [01-generic-go-scraper-engine-and-nereval-port-design-guide.md](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/design-doc/01-generic-go-scraper-engine-and-nereval-port-design-guide.md)
- [scraper.md](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/sources/local/scraper.md)
