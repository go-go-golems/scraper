---
Title: Queue Policies and Rate Limiting
Slug: scraper-queue-policies-and-rate-limiting
Short: "Explains queue keys, max-in-flight control, and the durable token-bucket limiter used by the scheduler."
Topics:
- scraper
- scheduler
- rate-limiting
- token-bucket
- queues
Commands:
- worker
- engine
Flags:
- max-workers
- poll-interval
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

Queue policy is the mechanism that controls how fast work starts. The scheduler already uses queue keys to group related work, but the current implementation now goes further: it can enforce both a maximum number of active ops for a queue and a durable token-bucket rate limit. This matters because scraping behavior should be controlled by explicit queue policy, not by accidental timing of fast or slow ops.

The durable rate limiter was implemented in the engine rather than in site code or runner-local memory. That means multiple workers, restarts, and repeated smoke tests all see the same token-bucket state.

## The Three Concepts

New contributors should separate these ideas clearly:

- queue identity: which ops share a pacing domain
- max in flight: how many ops from that queue may be active at once
- rate limit: how often new ops from that queue may start over time

In this codebase, the queue key lives on each op. The policy lives in site registration or a provider function. The mutable token state lives in the engine SQLite store.

## Where The Code Lives

The shared queue-policy types are in [pkg/engine/model/types.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/model/types.go). The scheduler resolves queue policy in [pkg/engine/scheduler/scheduler.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go). The SQLite store enforces it transactionally in [pkg/engine/store/sqlite/store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go). Site declarations can attach queue policies through [pkg/sites/registry/registry.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/registry/registry.go).

The durable table for token state is created in [002_engine_runtime.sql](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/migrations/002_engine_runtime.sql).

## How The Scheduler Uses Policy

The scheduler does not do token math itself. It asks for queue candidates, resolves policy for each `site + queue`, and passes that policy into the store lease call. The store then decides whether a lease may be created.

That boundary matters because it keeps the scheduling loop readable:

- scheduler decides “which queues should I try?”
- store decides “may this queue start another op right now?”

The scheduler can also attempt more than one lease from the same queue in one cycle when `MaxInFlight > 1`.

## How The Store Enforces Policy

The store uses one transaction to:

1. count active leases for the queue
2. block if `active >= MaxInFlight`
3. load current token state
4. refill tokens from elapsed time
5. block if `tokens < 1`
6. select one ready op
7. consume one token
8. persist token state
9. write the lease and mark the op running

This is the key race-safety invariant. If token checks and lease creation happened in separate steps, two workers could both believe they were allowed to start work.

## Site Integration

Queue policies are available at the site definition layer, but built-in sites do not yet ship aggressive non-default policies by default. That was an intentional rollout choice so fixture and smoke paths would stay deterministic while the limiter was being proven.

The worker and local runner paths already honor any site policy present in the registry. This means a test or future rollout can enable queue pacing without changing scheduler code again.

## Operational Reality

Today the rate limiter is real and durable, but operator visibility is still lightweight. `scraper engine status` shows workflow, op, lease, result, and artifact counts. It does not yet expose the current token count in `queue_limit_state`. If you need to debug limiter state deeply, you currently inspect the SQLite tables directly or extend the admin commands.

The main validation points already in the repo are:

- model tests for queue-policy normalization
- store tests for burst-1, burst-2, and reopen durability
- scheduler tests for token-bucket blocking and `MaxInFlight > 1`
- command-path tests that exercise `js-demo` and a rate-limited Hacker News fixture path

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| A queue never starts a second op | `MaxInFlight` is still the default `1` | Check the site definition or queue policy provider |
| A queue starts one op and then appears idle | Tokens are exhausted and refill has not happened yet | Increase wait time between worker cycles or inspect the configured rate policy |
| Rate limiting seems inconsistent across restarts | The engine DB changed or was reset | Remember that limiter state is durable in the engine DB, not in memory |
| A test became flaky after adding queue policy | Timing-sensitive policy was enabled in a fixture path | Prefer custom registry policy in tests instead of changing built-in defaults immediately |

## See Also

- [scraper-runtime-model](scraper help scraper-runtime-model) — How queue policy fits into the worker/runtime split
- [scraper-architecture-overview](scraper help scraper-architecture-overview) — Broader engine and site model
- [scraper-new-developer-onboarding](scraper help scraper-new-developer-onboarding) — Suggested first validation path for a new contributor
