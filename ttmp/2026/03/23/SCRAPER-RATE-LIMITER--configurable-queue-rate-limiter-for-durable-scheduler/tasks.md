# Tasks

## Analysis and planning

- [x] Create the dedicated `SCRAPER-RATE-LIMITER` ticket workspace
- [x] Inspect the current scheduler, store contract, SQLite store, and queue tests
- [x] Confirm the gap between current queue serialization and the intended token-bucket design
- [x] Update the design to assume fresh-schema bootstrap during prototyping instead of migration work
- [x] Write a detailed design and implementation guide for a new intern
- [x] Record a chronological investigation diary
- [x] Upload the ticket bundle to reMarkable

## Detailed implementation plan

### Phase 1. Model and policy contracts

- [x] Add `RateLimitKind` to [types.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/model/types.go)
- [x] Add `RateLimitPolicy` with:
  - [x] `Kind`
  - [x] `RatePerSecond`
  - [x] `Burst`
- [x] Add `QueuePolicy` with:
  - [x] `MaxInFlight`
  - [x] `RateLimit`
- [x] Add helper normalization rules so zero values fall back cleanly to current behavior
- [x] Keep `QueuePolicy` in `pkg/engine/model` for the prototype slice
- [x] Add unit tests for the policy normalization rules

### Phase 2. Scheduler wiring

- [x] Add a queue policy provider hook to [scheduler.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go)
- [x] Add a default provider that returns:
  - [x] `MaxInFlight = 1`
  - [x] `RateLimit = nil`
- [x] Thread resolved queue policy into lease requests
- [x] Keep the scheduler responsible only for:
  - [x] getting candidates
  - [x] resolving queue policy
  - [x] passing policy into the store
  - [x] emitting events
- [x] Add scheduler tests that confirm no-policy queues still behave exactly like current `one active op per queue`

### Phase 3. Store contract changes

- [x] Extend `LeaseRequest` in [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/store.go) to carry queue policy
- [x] Decide whether queue-status read methods should be added now or deferred
- [x] Keep the `Store` interface small enough that existing callers are easy to update
- [x] Add compile-time coverage for all scheduler/store call sites after the contract change

### Phase 4. Fresh-schema engine update

- [x] Update the baseline engine schema in [migrations](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/migrations)
- [x] Add `queue_limit_state` table directly to the baseline schema for prototype use
- [x] Add supporting index or indexes if still needed after implementation review
- [x] Document that local prototype databases may be deleted and recreated after this schema change
- [x] Verify a brand-new engine DB boots successfully with the updated schema

### Phase 5. Queue limiter math and state handling

- [x] Add an internal store-side queue limiter state type
- [x] Implement lazy creation of queue limiter state when a queue is first used
- [x] Implement refill logic based on:
  - [x] `last_refill_at`
  - [x] elapsed seconds
  - [x] `RatePerSecond`
  - [x] `Burst`
- [x] Implement token consumption on successful lease start
- [x] Keep `MaxInFlight` checks separate from token-bucket checks
- [x] Derive availability from `tokens` plus `last_refill_at` instead of persisting `available_at`
- [x] Add tests covering:
  - [x] refill up to burst cap
  - [x] fractional token refill
  - [x] zero/disabled rate limit
  - [x] reopen durability of limiter state

### Phase 6. Atomic lease enforcement in SQLite

- [x] Update `LeaseReadyOp` in [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go) to:
  - [x] load queue limiter state in the same transaction
  - [x] count active leases for the queue
  - [x] reject when `active >= MaxInFlight`
  - [x] refill tokens
  - [x] reject when `tokens < 1`
  - [x] select one ready op
  - [x] consume one token
  - [x] persist limiter state
  - [x] create the lease
  - [x] mark the op `running`
- [x] Ensure no token is consumed when no ready op is ultimately leased
- [ ] Add a dedicated concurrent multi-worker lease-attempt test
- [x] Add SQLite integration tests for:
  - [x] `RateLimit=nil` fallback behavior
  - [x] burst-1 pacing
  - [x] burst-2 pacing
  - [x] restart/reopen durability
  - [ ] concurrent lease attempts

### Phase 7. Scheduler behavior tests

- [x] Add scheduler tests for:
  - [x] queue blocked by rate limit produces `Processed == 0`
  - [x] later cycle succeeds after refill
  - [ ] two different queues are paced independently
  - [x] `MaxInFlight > 1` works independently of token-bucket pacing if enabled
- [x] Keep the existing queue-serialization test in [scheduler_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler_test.go)
- [x] Keep the existing `js-demo` queue test in [site_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site_test.go) as the baseline default-path proof

### Phase 8. Site integration

- [x] Extend site definition or a related config layer so built-in sites can declare queue policies
- [ ] Ship non-default queue policies in the built-in site definitions
- [x] Add at least one `js-demo` workflow/test that demonstrates non-trivial token-bucket behavior
- [x] Add one HTTP-backed exercise-site test proving the same limiter path applies there too

### Phase 9. Observability and debugging

- [x] Add a scheduler event such as `queue_rate_limited`
- [x] Decide whether to add store helpers for queue limiter status in the same slice
- [ ] Add a lightweight operator command for inspecting queue tokens and next eligibility
- [ ] Make log messages clearly distinguish:
  - [ ] blocked by active lease
  - [ ] blocked by token bucket
  - [ ] no ready work

### Phase 10. Prototype rollout and cleanup

- [x] Delete and recreate local engine DBs used for manual testing after schema changes
- [x] Run full `go test ./...`
- [x] Run focused command-path smoke tests for:
  - [x] `js-demo`
  - [x] one HTTP-backed site
- [ ] Update `SCRAPER-DESIGN` if the final implementation shape diverges from the broader architecture notes
- [ ] Re-upload the updated rate-limiter ticket bundle to reMarkable after implementation begins
