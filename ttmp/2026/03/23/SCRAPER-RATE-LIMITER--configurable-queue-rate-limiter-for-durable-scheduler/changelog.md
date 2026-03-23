# Changelog

## 2026-03-23

Created the dedicated `SCRAPER-RATE-LIMITER` planning ticket to close the gap between the current queue-serialization scheduler and the intended queue-based token-bucket rate-limiter design.

Later the same day, the design was simplified for the current prototyping phase: instead of planning upgrade migrations, the ticket now assumes fresh-schema bootstrap and explicit local DB resets when the engine schema changes.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go — Current scheduler loop and queue leasing behavior analyzed for the ticket
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/store.go — Current store contract analyzed for missing rate-limit policy/state hooks
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go — Current SQLite queue-candidate and lease logic analyzed for the design guide
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler_test.go — Existing queue-domain behavior referenced as current proof of serialization semantics
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site_test.go — Existing JS queue serialization test referenced as current proof for JS ops
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-RATE-LIMITER--configurable-queue-rate-limiter-for-durable-scheduler/design-doc/01-queue-rate-limiter-analysis-and-implementation-guide.md — Added the detailed analysis, design, API sketches, and implementation plan
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-RATE-LIMITER--configurable-queue-rate-limiter-for-durable-scheduler/reference/01-investigation-diary.md — Added the chronological research diary and validation record

Implemented the durable queue rate limiter in commit `263b2dc`, including queue policy types, SQLite token-bucket enforcement, scheduler policy wiring, site-level policy declaration, and command-path tests that exercise `js-demo` submit plus worker execution and an HTTP-backed Hacker News fixture run.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/model/types.go — Added `RateLimitPolicy`, `QueuePolicy`, and normalization logic
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/model/types_test.go — Added policy normalization coverage
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/store.go — Extended lease requests to carry resolved queue policy
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/migrations/002_engine_runtime.sql — Added durable `queue_limit_state`
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go — Implemented transactional active-lease checks, token refill, token consumption, and leasing
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store_test.go — Added burst and reopen durability tests
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go — Added queue policy provider support, multi-op same-queue cycles, and `queue_rate_limited` events
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler_test.go — Added scheduler coverage for rate-limited queues and `MaxInFlight > 1`
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/registry/registry.go — Added site-owned queue policy declaration and provider generation
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/runtime_helpers.go — Added shared scheduler runtime setup for queue policies
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/worker.go — Wired site queue policies into the durable worker path
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/cliutil/http_runner.go — Wired site queue policies into local HTTP-backed workflow runs
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site_test.go — Added end-to-end command tests for rate-limited `js-demo` and Hacker News runs
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-RATE-LIMITER--configurable-queue-rate-limiter-for-durable-scheduler/design-doc/01-queue-rate-limiter-analysis-and-implementation-guide.md — Updated the design guide to describe the implemented shape and remaining gaps
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-RATE-LIMITER--configurable-queue-rate-limiter-for-durable-scheduler/reference/01-investigation-diary.md — Recorded the implementation, validation, and review notes
