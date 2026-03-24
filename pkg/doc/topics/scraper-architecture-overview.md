---
Title: Scraper Architecture Overview
Slug: scraper-architecture-overview
Short: "High-level map of the durable engine, JS site layer, and built-in site packages."
Topics:
- scraper
- architecture
- engine
- javascript
- sites
Commands:
- scraper
- worker
- site
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `scraper` repository is a durable workflow engine for scraping tasks. Go owns persistence, scheduling, HTTP execution, leases, retries, queue policy, and CLI ergonomics. JavaScript owns most site-specific behavior: parsing HTML, deciding what work to emit next, and writing site-specific projections into each site database. That split is the main thing a new contributor needs to understand before reading code.

The current system is built around a small set of stable primitives. A workflow contains ops. Ops are persisted in the engine SQLite database. Workers poll for ready ops, lease them, execute them through a runner such as `js` or `http/fetch`, and persist results plus artifacts. Site packages such as `js-demo`, `hackernews`, `slashdot`, and `nereval` provide the JS scripts, submit verbs, fixtures, and per-site schema that sit on top of that engine.

## Core Layers

The engine layer is the durable runtime. It lives mainly in [pkg/engine/model/types.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/model/types.go), [pkg/engine/scheduler/scheduler.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go), and [pkg/engine/store/sqlite/store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go). It is responsible for turning “a graph of durable work” into repeatable execution with leases, retries, dependency tracking, queue policies, artifacts, and workflow state.

The site layer is the programmable behavior layer. Each site definition in [pkg/sites](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites) contributes:

- a site name and embedded filesystem
- optional submit verbs under `verbs/`
- op execution scripts under `scripts/`
- site DB migrations under `migrations/`
- optional fixtures used by tests

The CLI layer is the operator shell. It wires logging, help, migrations, engine inspection, site submission commands, and the background worker loop. The main entrypoints are [pkg/cmd/root.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/root.go), [pkg/cmd/site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site.go), [pkg/cmd/worker.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/worker.go), and [pkg/cmd/engine.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/engine.go).

## Runtime Model

The most important runtime distinction is between submission-time JS and execution-time JS.

- Submission-time JS lives in `verbs/` and is discovered from `__verb__` metadata.
- Execution-time JS lives in `scripts/` and runs as durable `js` ops through the worker.

This means a typical workflow looks like this:

```text
scraper site <site> run <verb>
  -> Go host loads JS submit verb
  -> JS submit verb inserts initial durable ops
  -> CLI exits

scraper worker run
  -> polls engine DB
  -> leases ready ops
  -> runs http/fetch or js runners
  -> persists results, artifacts, emitted child ops
```

The submit verb does not run the whole scrape inline. It only seeds the first durable work. The worker is the process that actually executes queued ops.

## Databases

The engine database stores cross-site runtime state. It contains workflows, ops, leases, dependencies, results, artifacts, and queue limiter state. The schema is managed in [pkg/engine/store/sqlite/migrations](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/migrations).

Each site gets its own SQLite database under the sites directory. A site DB stores query-oriented read models and projections that are specific to one site. For example, `nereval.db` contains normalized property assessment tables, while `js-demo.db` contains demo rows used to prove the JS runtime and durable worker path.

This split matters because it keeps engine correctness and site-specific schema evolution separate. If a site needs new projection tables, it changes its own migrations without forcing a top-level engine redesign.

## Built-In Sites

The current built-in sites are intentionally progressive:

- `js-demo` proves pure `js -> js -> site-db` execution without HTTP.
- `hackernews` proves `js -> http/fetch -> js -> site-db`.
- `slashdot` proves the same path on a different HTML shape and multipage fan-out.
- `nereval` is the first complex site. It adds ASP.NET form-state pagination, detail-page fan-out, and normalized site projections.

The default site registry lives in [pkg/sites/defaults/defaults.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/defaults/defaults.go).

## Commands You Will Use First

The fastest way to get oriented is to use the CLI against fixture-backed paths and the engine visibility commands.

- `scraper engine status`
- `scraper engine migrations status`
- `scraper site migrate js-demo`
- `scraper site js-demo run seed --workflow-id demo-1`
- `scraper worker run --max-cycles 16 --poll-interval 5ms`
- `scraper site hackernews run seed --fixture --max-pages 2`
- `scraper site slashdot run seed --fixture --max-pages 2`
- `scraper site nereval run seed --workflow-id nereval-fixture --base-url <fixture-server> --max-pages 2`

Use `scraper help <slug>` for the detailed pages added in this help set.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| `site <name> run <verb>` submits work but nothing happens | The worker is not polling the engine DB | Run `scraper worker run` against the same `--engine-db` and `--sites-dir` |
| JS script cannot see a database | The runtime was not given `site-db` or `scraper-db` | Start by reviewing [pkg/js/runtime/databases.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/js/runtime/databases.go) and the worker setup in [pkg/cmd/worker.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/worker.go) |
| Workflow looks stuck | Ready ops are not being leased or a dependency failed | Check `scraper engine status`, then inspect the scheduler/store path in [pkg/engine/scheduler/scheduler.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go) |
| A site parser seems wrong | The HTML fixture does not match the parser assumptions or the live site changed | Start with the fixture-backed tests for that site before changing runtime code |

## See Also

- [scraper-runtime-model](scraper help scraper-runtime-model) — Deeper explanation of submit verbs, workers, op JS, and durable execution
- [scraper-queue-policies-and-rate-limiting](scraper help scraper-queue-policies-and-rate-limiting) — How queue policies and durable token-bucket pacing work
- [scraper-new-developer-onboarding](scraper help scraper-new-developer-onboarding) — Step-by-step onboarding path for a new contributor
- [scraper-adding-a-site](scraper help scraper-adding-a-site) — How to add a new site package using the current patterns
