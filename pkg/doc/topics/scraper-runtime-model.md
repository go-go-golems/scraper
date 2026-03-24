---
Title: Scraper Runtime Model
Slug: scraper-runtime-model
Short: "Explains submit verbs, durable ops, workers, runners, and how JS fits into the execution model."
Topics:
- scraper
- runtime
- workflows
- javascript
- workers
Commands:
- site
- worker
- engine
Flags:
- engine-db
- sites-dir
- max-cycles
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The scraper runtime is intentionally split into two JavaScript environments that do different jobs. Submit verbs create initial durable work. Op scripts execute durable work later. New contributors often blur those together at first, and that confusion makes the code harder to reason about. This page is the short version of the real execution model.

At the highest level, the CLI submits workflows into the engine DB and the worker process executes them later. The worker uses runners such as `js` and `http/fetch`, reads site definitions from the registry, opens the right site DB, and writes back results plus any child ops emitted during execution.

## Submission-Time JS

Submission-time JS lives under `pkg/sites/<site>/verbs/`. These files expose top-level functions annotated with `__verb__`. The submit-verb host scans those files, builds Glazed/Cobra commands, and runs the selected function exactly once when the operator invokes a command such as `scraper site js-demo run seed`.

CLI flags are defined in the `__verb__` metadata, not in Go code. For example, `hackernews/verbs/seed.js` declares `--base-url` and `--max-pages` as fields, and the host wires them into Cobra automatically. The parsed values are available in the verb function as `ctx.values`.

The important constraint is that a submit verb is not a worker. It does not crawl pages for minutes and it does not keep an in-process scheduler alive by default. Its job is to describe or emit the initial durable work graph.

The submit-verb host is implemented in:

- `pkg/sites/submitverbs/host.go`
- `pkg/sites/submitverbs/runtime.go`

## Execution-Time JS

Execution-time JS lives under `pkg/sites/<site>/scripts/`. These files run as durable ops through the `js` runner. Each op is persisted in the engine DB and references the script to run through op metadata, usually `metadata.script`.

Scripts can be synchronous or async. The runtime supports `async function` exports and will await the returned Promise before persisting results.

The execution context is intentionally narrow. A script can:

- read `ctx.input`
- inspect workflow metadata via `ctx.workflow` and op metadata via `ctx.op`
- read dependency results with `ctx.dep(opID)`
- emit child ops with `ctx.emit(spec)`
- write records with `ctx.writeRecord(collection, key, data)` and artifacts with `ctx.writeArtifact(spec)`
- use `require("site-db")` and `require("scraper-db")` for direct SQL access

For the complete API with type signatures, see `scraper help scraper-js-api-reference`.

The execution path is implemented in:

- `pkg/engine/runner/js.go`
- `pkg/js/runtime/executor.go`
- `pkg/js/runtime/databases.go`

## Workers, Runners, and Scheduling

The worker process runs `scraper worker run`. It opens the engine store, builds the runner registry, opens site DBs on demand, and loops over ready queues. The scheduler is responsible for dependency refresh, lease recovery, queue policy resolution, and calling the appropriate runner.

The runner registry (`pkg/engine/runner/runner.go`) maps op kinds to runner implementations. The two built-in runners are registered at startup:

- `js` runner — loads site scripts, builds a goja runtime per execution, and runs the script function
- `http/fetch` runner — performs HTTP requests using `pkg/engine/config/` settings (user agent, timeout)

The main files are:

- `pkg/cmd/worker.go`
- `pkg/engine/scheduler/scheduler.go`
- `pkg/engine/runner/http.go`
- `pkg/engine/store/sqlite/store.go`

The scheduler does not know site-specific parsing logic. It only knows how to:

- find ready queues
- resolve queue policy
- lease work safely
- invoke a runner
- write back results, emitted ops, artifacts, and errors

That separation is what lets `js-demo`, `hackernews`, `slashdot`, and `nereval` all use the same engine.

## Data Flow

The runtime model is easiest to understand as a durable graph:

```text
submit verb
  -> create workflow
  -> emit initial op(s)

worker
  -> lease op
  -> run js or http/fetch
  -> persist result/artifacts
  -> persist emitted child ops
  -> later child ops become ready
```

For example, the NEREVAL workflow looks like this:

```text
site nereval run seed
  -> verb emits js seed op

worker run
  -> js seed emits list fetch + list extract
  -> list extract emits detail fetches and page-2 fetch
  -> detail extractors write normalized property tables into nereval.db
```

## Engine DB vs Site DB

The engine DB is the durable workflow runtime database. It stores workflows, ops, dependencies, leases, results, artifacts, and queue limiter state.

The site DB is the query-facing projection database for one site. It stores the records an operator or downstream tool actually wants to inspect. Keeping those separate matters because engine state is generic runtime infrastructure, while site schema is allowed to be specific and evolve differently.

If a contributor is unsure where a new table should go, the rule of thumb is:

- runtime/scheduling/lease/result table: engine DB
- query-oriented site projection: site DB

## What To Read In Code

Read these in order if you want the shortest path through the real runtime:

1. `pkg/cmd/root.go`
2. `pkg/cmd/site.go`
3. `pkg/sites/submitverbs/host.go`
4. `pkg/sites/submitverbs/runtime.go`
5. `pkg/cmd/worker.go`
6. `pkg/engine/scheduler/scheduler.go`
7. `pkg/engine/store/sqlite/store.go`

Then read one site package end to end.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| You expected `site <site> run <verb>` to do the whole scrape | Submit verbs only seed work | Run `scraper worker run` against the same DBs |
| JS script cannot find dependency output | The dependency op ID is wrong or the dependency was never emitted | Check the workflow graph in the emitting script and the dependency read in the consumer script |
| A site writes nothing to its DB | The worker never opened that site DB or the script returned an error early | Start with the command-path tests in `pkg/cmd/site_test.go` |
| It is unclear whether logic belongs in a verb or a script | Submission and execution responsibilities are mixed | Keep workflow seeding in `verbs/` and durable parsing/fan-out in `scripts/` |

## See Also

- `scraper help scraper-architecture-overview` — Broader system map and command overview
- `scraper help scraper-js-api-reference` — Complete JavaScript API reference for both verb and script contexts
- `scraper help scraper-queue-policies-and-rate-limiting` — Queue policy and token-bucket behavior in the worker
- `scraper help scraper-new-developer-onboarding` — First-day path through the repo and smoke tests
