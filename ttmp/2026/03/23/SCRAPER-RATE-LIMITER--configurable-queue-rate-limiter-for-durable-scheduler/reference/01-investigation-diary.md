---
Title: Investigation diary
Ticket: SCRAPER-RATE-LIMITER
Status: active
Topics:
    - scraper
    - scheduler
    - rate-limiting
    - token-bucket
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/engine/scheduler/scheduler_test.go
      Note: Existing queue-domain test referenced as current serialization proof
    - Path: pkg/sites/jsdemo/site_test.go
      Note: Existing js-demo queue test referenced as proof that JS ops share the same queue behavior
ExternalSources: []
Summary: Chronological diary for the queue rate limiter analysis ticket, including evidence gathered, design conclusions, validation, and delivery notes.
LastUpdated: 2026-03-23T22:20:00-04:00
WhatFor: Records how the queue rate limiter analysis was assembled and what concrete code evidence informed the design.
WhenToUse: Use when reviewing how the ticket was researched, what commands were run, and which implementation conclusions were evidence-backed rather than speculative.
---


# Investigation diary

## Goal

Create a dedicated ticket for adding a real configurable queue rate limiter to the scraper scheduler, explain the current implementation gap versus the architecture spec, and produce a detailed implementation guide suitable for a new intern. Deliver the ticket bundle to reMarkable.

## Context

The immediate trigger for this ticket was a clarification question during the `js-demo` queue-throttling work. The code now proves that JS ops participate in queue-domain serialization, but that work also made the remaining gap explicit: the engine does not yet implement a real token-bucket or RPS limiter even though the broader architecture and imported design notes assume rate limiting is a first-class scheduler concern.

This ticket is therefore a planning and design ticket, not an implementation ticket.

One later adjustment was made after the initial write-up: the team clarified that this limiter work is still firmly in prototype territory, so the ticket no longer recommends spending time on upgrade migrations. The design now assumes a fresh-schema bootstrap path and explicit local DB resets where needed.

## Chronology

## Step 1: Create the dedicated ticket workspace

I created a new ticket rooted in `scraper/ttmp` with the ID `SCRAPER-RATE-LIMITER`, then added:

- one primary design doc
- one investigation diary

Commands run:

```bash
docmgr ticket create-ticket \
  --ticket SCRAPER-RATE-LIMITER \
  --title "Configurable queue rate limiter for durable scheduler" \
  --topics scraper,scheduler,rate-limiting,token-bucket

docmgr doc add --ticket SCRAPER-RATE-LIMITER \
  --doc-type design-doc \
  --title "Queue rate limiter analysis and implementation guide"

docmgr doc add --ticket SCRAPER-RATE-LIMITER \
  --doc-type reference \
  --title "Investigation diary"
```

What worked:

- The workspace initialized cleanly.
- The default ticket structure already gave the expected files:
  - `index.md`
  - `tasks.md`
  - `changelog.md`
  - the design doc
  - this diary

What did not work:

- N/A

## Step 2: Gather code evidence about the current scheduler

I inspected the current scheduler, store contract, SQLite store, existing queue tests, `js-demo` queue test, and the imported design source.

Key commands:

```bash
nl -ba scraper/pkg/engine/scheduler/scheduler.go | sed -n '1,260p'
nl -ba scraper/pkg/engine/store/store.go | sed -n '1,220p'
nl -ba scraper/pkg/engine/store/sqlite/store.go | sed -n '320,520p'
nl -ba scraper/pkg/engine/scheduler/scheduler_test.go | sed -n '300,370p'
nl -ba scraper/pkg/sites/jsdemo/site_test.go | sed -n '180,290p'
nl -ba scraper/ttmp/.../sources/local/scraper.md | sed -n '80,180p'
nl -ba scraper/ttmp/.../design-doc/01-generic-go-scraper-engine-and-nereval-port-design-guide.md | sed -n '470,530p'
```

Main findings:

1. The current scheduler config has no rate-limit policy fields.
2. The current store contract exposes queue identity but no queue policy or limiter state.
3. The current SQLite store enforces only "no active lease already exists for this queue."
4. The scheduler test proves one-op-per-queue behavior.
5. The `js-demo` test proves that real JS ops use the same queue behavior.
6. The imported design notes clearly intend queue-based rate limiting as a first-class feature of scheduling.

## Step 3: Resolve the central design question

The main design question was whether the future limiter should be:

- in memory only,
- embedded inside runners,
- or owned durably by the store/scheduler layer.

I chose the durable store/scheduler route.

Reasoning:

- the engine is already durable-first for leases, retries, workflow state, and results
- rate-limit state affects correctness across workers and restarts
- runner-local or process-local token buckets would break multi-worker coordination

This led to the main recommendation:

- queue identity stays on the op
- queue policy stays in Go configuration / site registration
- mutable limiter state lives in SQLite

## Step 4: Write the design guide

The design guide was written as a complete onboarding document for a new intern. It includes:

- current-state architecture
- problem statement
- gap analysis
- recommended token-bucket design
- API sketches
- SQLite schema sketch
- pseudocode for atomic lease-and-consume
- phased implementation plan
- testing strategy
- risks and alternatives

I also made the key distinction explicit:

- current behavior = queue serialization
- desired later behavior = configurable durable rate limiting

## Step 5: Update ticket bookkeeping

I replaced the default task list with a phased implementation and planning checklist, updated the changelog, and tightened the index overview so the ticket reads like a real working design ticket rather than an empty scaffold.

## Step 6: Validate the ticket

Validation command:

```bash
docmgr doctor --ticket SCRAPER-RATE-LIMITER --stale-after 30
```

Doctor passed cleanly after the ticket docs were updated.

## What worked

- The current codebase has a clean enough separation between scheduler and store that the missing rate-limiter layer is easy to describe precisely.
- The imported design notes and the current scheduler tests line up well enough that the gap is easy to explain without hand-waving.
- The `js-demo` queue test made the current "serialization, not token bucket" distinction much easier to explain concretely.

## What didn't work

- There was no existing dedicated ticket or explicit backlog item just for token-bucket / RPS limiting, so this ticket had to be created from scratch rather than extending an existing focused workstream.

## What I learned

- The current queue-domain model is a useful foundation because queue identity is already the right primitive. The main missing piece is policy and durable state, not a new scheduling abstraction.
- The most important technical requirement is atomicity: rate-limit checks and lease acquisition must happen in the same transaction or the limiter will be wrong under multiple workers.
- The right beginner framing is to separate:
  - queue identity
  - concurrency limit
  - rate limit

## What was tricky to build

- The hardest part of the analysis was not the token-bucket math itself. It was drawing the boundary cleanly between:
  - scheduler policy resolution
  - store-side atomic enforcement
  - runner execution

If those are mixed together, the design becomes harder to test and easier to race.

## What warrants a second pair of eyes

- The proposed placement of `QueuePolicy` and related types in `pkg/engine/model` is pragmatic, but someone may prefer a dedicated policy package for longer-term layering reasons.
- The decision to persist both `tokens` and `available_at` is helpful for debugging, but `available_at` may be redundant if the team wants a smaller first schema.

## What should be done in the future

- Use this ticket as the implementation ticket when the team is ready to add real rate limiting.
- Keep the existing queue-serialization tests even after the token bucket is added.
- Add later operator visibility commands for limiter state if the first implementation succeeds cleanly.

## Code review instructions

Read in this order:

1. [scheduler.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go)
2. [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/store.go)
3. [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go)
4. [scheduler_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler_test.go)
5. [site_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site_test.go)
6. [01-queue-rate-limiter-analysis-and-implementation-guide.md](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-RATE-LIMITER--configurable-queue-rate-limiter-for-durable-scheduler/design-doc/01-queue-rate-limiter-analysis-and-implementation-guide.md)

## Verification

- `docmgr doctor --ticket SCRAPER-RATE-LIMITER --stale-after 30`

Result:

- all checks passed after adding the missing topic vocabulary entries

## Delivery

Planned reMarkable destination:

- `/ai/2026/03/23/SCRAPER-RATE-LIMITER`

Completed delivery:

- uploaded bundle name: `SCRAPER-RATE-LIMITER queue rate limiter analysis`
- verified with:

```bash
remarquee cloud ls /ai/2026/03/23/SCRAPER-RATE-LIMITER --long --non-interactive
```

## Step 7: Implement the durable queue limiter and validate it through command-path tests

This step turned the planning ticket into an implementation ticket. The core change was to keep queue identity where it already lived, add queue policy to the shared engine model, then enforce both `MaxInFlight` and token-bucket pacing inside the SQLite lease transaction. That preserved the scheduler/store separation from the design doc instead of smearing limiter logic across runners.

The second half of the step was proving the new path with real command flows rather than only unit tests. I kept the built-in sites conservative by default, then used custom registry definitions in tests to exercise rate limiting through `site js-demo run seed`, `engine status`, `worker run`, and an HTTP-backed Hacker News fixture run.

### Prompt Context

**User prompt (verbatim):** "I'm oging for a walk, so feel free to continue with the RPS ticket SCRAPER-RATE-LIMITER and then continue with testing it and porting over the nereval stuff (but don't run it!). 

I hope to see everything done when I come back"

**Assistant interpretation:** Finish the queue rate limiter implementation, validate it thoroughly, and then move on to the NEREVAL port without running the NEREVAL scraper live.

**Inferred user intent:** Make the rate-limiter ticket real, not just planned, and leave the codebase in a state where durable queue pacing is proven before the more complex NEREVAL port starts.

**Commit (code):** `263b2dc` — `Add durable queue rate limiter`

### What I did

- Added `RateLimitKind`, `RateLimitPolicy`, `QueuePolicy`, and normalization tests in:
  - [types.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/model/types.go)
  - [types_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/model/types_test.go)
- Extended `LeaseRequest` to carry queue policy in [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/store.go).
- Added durable `queue_limit_state` to [002_engine_runtime.sql](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/migrations/002_engine_runtime.sql).
- Reworked [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go) so `LeaseReadyOp` now:
  - counts active leases by queue
  - refills token-bucket state
  - blocks when `tokens < 1`
  - persists updated token state only when an op is actually leased
- Added store tests for burst-1, burst-2, and reopen durability in [store_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store_test.go).
- Added scheduler queue policy support and new tests for:
  - multi-op same-queue leasing when `MaxInFlight > 1`
  - token-bucket blocking plus later success
  - `queue_rate_limited` event emission
  in [scheduler.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go) and [scheduler_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler_test.go).
- Extended the site registry with queue policy declarations in [registry.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/registry/registry.go).
- Wired policy resolution into the worker and local HTTP runner paths in:
  - [runtime_helpers.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/runtime_helpers.go)
  - [worker.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/worker.go)
  - [http_runner.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/cliutil/http_runner.go)
- Added command-path tests in [site_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site_test.go) for:
  - `js-demo` submit plus worker plus `engine status` with a custom JS queue token bucket
  - rate-limited Hacker News fixture execution
- Ran:
  - `go test ./pkg/engine/model ./pkg/engine/store/sqlite ./pkg/engine/scheduler ./pkg/cmd -count=1`
  - `go test ./pkg/engine/store/sqlite ./pkg/cmd -count=1`
  - `go test ./... -count=1`

### Why

- Queue pacing must be durable and worker-safe, so the store transaction is the right enforcement point.
- The scheduler should remain responsible for selecting queues and attaching policy, not for doing token accounting itself.
- Command-path tests were necessary because the user explicitly wanted proof that submitted tasks still execute through the durable worker flow.

### What worked

- The scheduler/store boundary from the design doc held up well in practice.
- `queue_limit_state` with `tokens` plus `last_refill_at` was enough; a persisted `available_at` field was not needed.
- The custom-registry command tests made it possible to prove queue limiting without forcing default rate policies onto every built-in site.
- The full suite stayed green after the limiter work.

### What didn't work

- I initially considered pushing default non-zero queue policies directly into the built-in site definitions, but that would have made several existing fixture tests timing-sensitive in a way that was unnecessary for this slice.
- There is still no dedicated concurrent multi-worker lease-attempt test. The transactional design is in place, but that exact test coverage remains future work.

### What I learned

- The candidate-list query must be broader once `MaxInFlight` is configurable; queue gating belongs in `LeaseReadyOp`, not in the candidate SQL.
- For this engine, concurrency control and rate control really are separate levers. The implementation got cleaner once the code reflected that distinction directly.
- Site-owned queue policy declaration is enough for now; shipping default policies is a separate rollout choice.

### What was tricky to build

- The main sharp edge was preserving race safety while letting a queue lease more than one op per cycle. The old "no active lease in this queue" query shape was safe for serialization, but too restrictive for `MaxInFlight > 1`. The fix was to move the active-lease check fully into the transactional lease path and let the scheduler attempt multiple leases against the same queue when policy allows it.

### What warrants a second pair of eyes

- The scheduler currently emits a generic `queue_rate_limited` event when a queue cannot lease under policy, but it does not distinguish token exhaustion from `MaxInFlight` blocking.
- I have not yet added a dedicated cross-process or dual-store concurrent token-consumption test.
- Built-in sites still use default queue behavior unless a test or future rollout sets explicit policies.

### What should be done in the future

- Add an operator/admin command to inspect `queue_limit_state`.
- Add one explicit concurrent multi-worker lease-attempt test.
- Decide, site by site, which default queue policies should ship once the team wants production-like pacing by default.

### Code review instructions

- Start with [types.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/model/types.go) and [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/store.go) to see the shared policy contract.
- Then read [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go) for the actual token-bucket enforcement.
- Then read [scheduler.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go) to see how queue policies are consumed.
- Finish with:
  - [scheduler_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler_test.go)
  - [site_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site_test.go)
- Validate with:
  - `go test ./... -count=1`
  - `docmgr doctor --ticket SCRAPER-RATE-LIMITER --stale-after 30`

### Technical details

Queue policy lookup now flows like this:

```text
site registry definition
  -> QueuePolicies map[QueueKey]QueuePolicy
  -> registry.QueuePolicyProvider()
  -> scheduler.SetQueuePolicyProvider(...)
  -> store.LeaseReadyOp(... Policy: resolvedPolicy ...)
```

The durable limiter table is:

```sql
CREATE TABLE queue_limit_state (
  site TEXT NOT NULL,
  queue_key TEXT NOT NULL,
  tokens REAL NOT NULL,
  last_refill_at TEXT NOT NULL,
  PRIMARY KEY(site, queue_key)
);
```
