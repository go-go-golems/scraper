# Changelog

## 2026-03-23

Added an explicit `js-demo` queue-domain test proving that the scheduler’s queue throttling applies to JavaScript ops too, not just HTTP ops.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site_test.go — Added a scheduler integration test showing two `js` ops in the same queue are processed one per cycle
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc/topics/scraper-architecture-overview.md — Clarified that queue-domain control is kind-agnostic and therefore applies to `js` ops as well

Added named operator entrypoints for the built-in Hacker News and Slashdot sites so both HTTP-backed exercise paths can be run locally as `seed` or `extract-frontpage`, including a `--fixture` mode that serves embedded HTML without touching the live sites.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/cliutil/http_runner.go — Added the shared HTTP-backed site runner helper used by built-in site commands
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/cli.go — Added `scraper site hackernews run seed|extract-frontpage`
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/workflow.go — Added durable workflow builders for Hacker News operator runs
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/slashdot/cli.go — Added `scraper site slashdot run seed|extract-frontpage`
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/slashdot/workflow.go — Added durable workflow builders for Slashdot operator runs
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/root.go — Switched site command construction to fail fast on site CLI registration errors
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site.go — Propagated site-specific CLI registration errors during command construction
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site_test.go — Added CLI coverage for the new Hacker News and Slashdot entrypoints
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc/topics/scraper-architecture-overview.md — Documented the new site run smoke-test commands

Generalized `scraper site js-demo run` into named entrypoints so the demo can run `seed`, `item`, or `summary` directly from the CLI.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/cli.go — Reworked the js-demo CLI into `run <entrypoint>` subcommands
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/workflow.go — Added dedicated builders for seed, item, and summary workflows
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site_test.go — Added direct scheduler coverage for the item entrypoint
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site_test.go — Added CLI coverage for `run seed`, `run item`, and `run summary`
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc/topics/scraper-architecture-overview.md — Updated operator docs to reference the new `run <entrypoint>` shape

Added a pure-JS `js-demo` site and `scraper site js-demo run` so the engine can exercise fan-out, dependency joins, `site-db`, records, artifacts, and async JS op execution without any `http/fetch` dependency.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site.go — Added the new built-in `js-demo` site definition and module wiring
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/cli.go — Added the operator-facing `scraper site js-demo run` command
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/workflow.go — Added deterministic workflow construction for the pure-JS demo pipeline
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site_test.go — Added end-to-end scheduler coverage for the pure-JS workflow
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site_test.go — Added CLI smoke coverage for `scraper site js-demo run`
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/js/runtime/executor.go — Extended the JS executor so sites can opt into extra go-go-goja module sets
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc/topics/scraper-architecture-overview.md — Updated the architecture help topic with the new pure-JS exercise path

Added two smaller built-in exercise sites, Hacker News and Slashdot, so the generic scheduler/runtime path can be validated end to end before starting the NEREVAL port.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/site.go — Added the Hacker News site definition, embedded scripts, migrations, and fixtures
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/site_test.go — Added the end-to-end Hacker News workflow test using the real scheduler, HTTP runner, JS runner, and site DB
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/slashdot/site.go — Added the Slashdot site definition, embedded scripts, migrations, and fixtures
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/slashdot/site_test.go — Added the end-to-end Slashdot workflow test using the real scheduler, HTTP runner, JS runner, and site DB
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/defaults/defaults.go — Added the built-in site registry wiring used by the CLI
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/root.go — Switched the production root command to the built-in site registry
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go — Added scheduler-side injection of preconfigured site DB handles into runner execution
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/reference/01-investigation-diary.md — Recorded the exercise-site milestone and debugging notes

## 2026-03-23

Added the first generic `http/fetch` runner with templated request rendering, form/body support, response metadata capture, optional body artifact persistence, and retry classification tests.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/runner/http.go — Added the Go-backed HTTP fetch runner
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/runner/http_test.go — Added fixture-style tests for success, retryable server errors, and non-retryable client errors
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc/topics/scraper-architecture-overview.md — Updated the help topic with the current `http/fetch` input contract
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/reference/01-investigation-diary.md — Recorded the HTTP milestone

## 2026-03-23

Added the first real scheduler loop with workflow submission, dependency promotion/cancellation, retry and backoff handling, expired-lease recovery, queue-domain leasing, and integration tests against the SQLite store.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go — Added the worker loop, retry logic, workflow-state updates, and scheduler events
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler_test.go — Added integration tests for fan-out, dependency completion, retry behavior, lease recovery, and queue-domain control
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go — Added runnable-op refresh, queue-candidate listing, workflow stats, retry-state persistence, and failure-result persistence
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/store.go — Extended the engine store contract for scheduler operation
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/model/types.go — Extended `OpSpec` with durable retry state
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc/topics/scraper-architecture-overview.md — Updated the operator help topic with current scheduler semantics
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/reference/01-investigation-diary.md — Recorded the scheduler milestone

## 2026-03-23

Added the first executable JS op runtime: site-script loading, a `ctx` contract for dependency reads and durable writes, a concrete `js` runner, and runtime tests covering emitted ops, artifacts, records, DB access, and teardown.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/js/runtime/executor.go — Added the generic scraper JS executor and result marshalling
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/js/runtime/promises.go — Added shared promise waiting for async JS ops
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/js/runtime/executor_test.go — Added end-to-end JS runtime tests for dependencies, emitted ops, records, artifacts, DB access, and teardown
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/runner/js.go — Added the engine-facing `js` runner
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/runner/js_test.go — Added runner integration coverage for site scripts
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc/topics/scraper-architecture-overview.md — Updated the operator help topic with the new JS runner contract
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/reference/01-investigation-diary.md — Recorded the JS runtime milestone

## 2026-03-23

- Initial workspace created
- Imported `/tmp/scraper.md` into the ticket sources as the primary design input
- Added the main design guide describing how the imported op/result architecture maps onto the current NEREVAL prototype and how to implement the Go/goja port in `scraper/`
- Added the investigation diary capturing the research path, commands, and design decisions

## 2026-03-23

Added the initial design guide and diary mapping the imported scraper architecture onto the current NEREVAL prototype and the local go-go-goja runtime.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/design-doc/01-generic-go-scraper-engine-and-nereval-port-design-guide.md — Primary design deliverable for the ticket
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/reference/01-investigation-diary.md — Chronological research log for the ticket


## 2026-03-23

Validated the ticket with docmgr doctor, seeded the local vocabulary, and uploaded the final document bundle to reMarkable at /ai/2026/03/23/SCRAPER-DESIGN.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/.docmgrignore — Ignored the raw imported source from doc validation so doctor only checks authored docs
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/changelog.md — Ticket changelog records validation and delivery
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/vocabulary.yaml — Seeded vocabulary needed for this repo's first ticket


## 2026-03-23

Revised the design so engine state lives in the engine DB while each site owns its own DB and can apply ordered SQL and JS migrations; prepared a v2 bundle for reMarkable delivery.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/design-doc/01-generic-go-scraper-engine-and-nereval-port-design-guide.md — Updated storage and migration architecture
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/reference/01-investigation-diary.md — Recorded the v2 architecture revision


## 2026-03-23

Expanded the ticket backlog into phased implementation work, bootstrapped the real `scraper` Go module and Glazed CLI, added embedded help docs, clarified `Lease` and `RetryPolicy` in the design guide, and added a repo-local `go.work` file so the new module builds in the local mono-workspace.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/go.work — Added a repo-local workspace linking `scraper`, `../glazed`, and `../go-go-goja`
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/go.mod — Created the first real module definition for the scraper repo
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/cmd/scraper/main.go — Added the CLI entrypoint
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/root.go — Added the Glazed root command with logging and help wiring
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc/topics/scraper-architecture-overview.md — Added the first embedded help entry
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/tasks.md — Replaced the one-shot checklist with phased build tasks
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/reference/01-investigation-diary.md — Recorded the bootstrap implementation step


## 2026-03-23

Added the phase-2 engine contracts: durable workflow/op/result types, store interfaces, runner interfaces, scheduler/config validation, and a site registry contract with package-level tests.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/model/types.go — Defined the durable engine data model
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/store.go — Added the first store interfaces
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/runner/runner.go — Added runner contracts and duplicate-safe runner registration
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go — Added scheduler config validation
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/registry/registry.go — Added the site registration contract
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/reference/01-investigation-diary.md — Recorded the contract milestone


## 2026-03-23

Added the first engine SQLite implementation with embedded SQL migrations, schema version tracking, and a concrete store for workflows, ops, leases, results, and artifacts.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/migrations/001_engine_core.sql — Initial core engine schema
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/migrations/002_engine_runtime.sql — Runtime tables for dependencies, leases, results, and artifacts
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/migrations.go — Embedded migration loader and applier
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go — First concrete SQLite-backed engine store
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store_test.go — Migration and round-trip store tests
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/reference/01-investigation-diary.md — Recorded the engine DB milestone


## 2026-03-23

Added `scraper engine status` and `scraper engine migrations status` so engine DB existence, schema state, migration state, and runtime counts can be inspected during smoke tests and debugging.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/engine.go — Added engine visibility commands
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/engine_test.go — Added CLI tests for the visibility commands
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/status.go — Added non-destructive engine DB inspection helpers
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/status_test.go — Added status inspection tests
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc/topics/scraper-architecture-overview.md — Added command discovery notes for the new engine status commands


## 2026-03-23

Added per-site SQLite DB management, combined SQL/JS site migrations, and the `scraper site migrate <site>` operator command.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/registry/registry.go — Extended the site definition with DB filename and runtime-module registrar support
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/migrate/manager.go — Added the site migration manager and JS migration runtime
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/migrate/manager_test.go — Added mixed SQL/JS migration and idempotency tests
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site.go — Added the explicit site migration CLI command
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site_test.go — Added CLI tests for site migration execution
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc/topics/scraper-architecture-overview.md — Added discovery notes for the site migration command


## 2026-03-23

Added preconfigured JS database exposure so runtimes can inject `scraper-db` and `site-db` instead of requiring JS to open SQLite files by path.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/js/runtime/databases.go — Added the scraper-side runtime registrar for preconfigured DB modules
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/js/runtime/databases_test.go — Added runtime tests covering `scraper-db` and `site-db`
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/migrate/manager.go — Wired the site migration runtime to expose `site-db`
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/migrate/manager_test.go — Updated migration tests to use the injected `site-db` module
