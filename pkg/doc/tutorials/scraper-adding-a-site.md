---
Title: Adding a New Site
Slug: scraper-adding-a-site
Short: "Step-by-step guide for adding a new built-in site using the current registry, JS runtime, migrations, and fixture-backed tests."
Topics:
- scraper
- tutorial
- sites
- javascript
- migrations
Commands:
- site
- worker
Flags:
- fixture
- sites-dir
- engine-db
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

Adding a site means fitting your scraper into the existing engine shape rather than inventing a new execution path. The durable engine is already responsible for scheduling, retries, HTTP execution, DB lifecycle, and queue policy. Your job is to describe the site’s behavior in a site package with submit verbs, scripts, migrations, fixtures, and tests.

This tutorial shows the path that the current built-in sites follow. Use it as a checklist when creating a new site package.

## Prerequisites

Before starting, make sure you understand:

- [scraper-architecture-overview](scraper help scraper-architecture-overview)
- [scraper-runtime-model](scraper help scraper-runtime-model)

It also helps to read one simple site and one complex site:

- [pkg/sites/jsdemo](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo)
- [pkg/sites/nereval](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/nereval)

## Step 1 — Create The Site Package Skeleton

Create a new directory under `pkg/sites/<site>/` and add:

- `site.go`
- `migrations/`
- `scripts/`
- `verbs/`
- optional `fixtures/`

Your `site.go` should embed those directories and return a `registry.Definition`. Follow the pattern in [pkg/sites/jsdemo/site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site.go) or [pkg/sites/nereval/site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/nereval/site.go).

The important parts are:

- `Name`
- `DatabaseFileName`
- `ScriptsFS` / `ScriptsRoot`
- `VerbsFS` / `VerbsRoot`
- `SQLMigrationsFS` / `SQLMigrationsRoot`

## Step 2 — Decide The First Submit Verb

Every site should have at least one operator entrypoint that seeds the first durable work. This lives in `verbs/` and uses `__verb__`.

Use `js-demo` as the smallest pattern:

- [pkg/sites/jsdemo/verbs/seed.js](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/verbs/seed.js)

Use `nereval` when the site needs more real input:

- [pkg/sites/nereval/verbs/seed.js](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/nereval/verbs/seed.js)

The submit verb should:

- read CLI values from `ctx.values`
- set a workflow name if useful
- emit one or more initial ops
- optionally set a target op ID for the command result

It should not:

- crawl pages directly
- sleep or poll
- act like a worker

## Step 3 — Write The Durable Scripts

Your durable site behavior lives in `scripts/`. Start with the smallest graph that proves the site can run in the engine.

Typical shapes:

- simple site:
  - `seed.js`
  - `extract_frontpage.js`
- complex site:
  - `seed.js`
  - `extract_list.js`
  - `extract_detail.js`
  - helper files under `scripts/lib/`

Use the helper modules already exposed by the runtime:

- `require("site-db")`
- `require("scraper-db")`

And use the runtime context carefully:

- `ctx.input`
- `ctx.dep(...)`
- `ctx.emit(...)`

For examples:

- [pkg/sites/hackernews/scripts/extract_frontpage.js](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/scripts/extract_frontpage.js)
- [pkg/sites/slashdot/scripts/extract_frontpage.js](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/slashdot/scripts/extract_frontpage.js)
- [pkg/sites/nereval/scripts/extract_list.js](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/nereval/scripts/extract_list.js)
- [pkg/sites/nereval/scripts/extract_detail.js](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/nereval/scripts/extract_detail.js)

## Step 4 — Add Site DB Migrations

If the site needs queryable output, define it in `migrations/`. The site DB should contain projection tables, not engine workflow state.

Examples:

- [pkg/sites/jsdemo/migrations/001_init.sql](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/migrations/001_init.sql)
- [pkg/sites/nereval/migrations/001_init.sql](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/nereval/migrations/001_init.sql)

Keep the first migration small and query-oriented. Add only the tables that the first end-to-end workflow actually writes.

## Step 5 — Register The Site

Add the site to the default registry so the root command can discover it.

Update:

- [pkg/sites/defaults/defaults.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/defaults/defaults.go)

If you skip this step, your site package will compile but the CLI will not expose it.

## Step 6 — Add Fixtures Before Live Traffic

Use fixtures first. They make parser tests fast, deterministic, and reviewable.

Good fixture sets usually include:

- one first page
- one later page if pagination exists
- one or more detail pages

Current examples:

- [pkg/sites/hackernews/fixtures/frontpage.html](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/fixtures/frontpage.html)
- [pkg/sites/slashdot/fixtures/frontpage.html](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/slashdot/fixtures/frontpage.html)
- [pkg/sites/nereval/fixtures](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/nereval/fixtures)

## Step 7 — Add A Command-Path Test

Do not stop at parser unit tests. Add at least one test that exercises:

1. `site <site> run <verb>`
2. `engine status`
3. `worker run`
4. site DB assertions

The strongest current examples are in [pkg/cmd/site_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site_test.go).

That test proves the site works in the actual engine path rather than only in a hand-rolled helper.

## Step 8 — Run The Validation Loop

Before committing:

```bash
gofmt -w pkg/sites/<site> pkg/cmd/site_test.go pkg/sites/defaults/defaults.go
go test ./... -count=1
```

If the site adds or changes embedded help pages, also verify:

```bash
go run ./cmd/scraper help <your-slug>
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| The CLI does not show the site | The site was not registered in the default registry | Update [pkg/sites/defaults/defaults.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/defaults/defaults.go) |
| `site <site> run <verb>` exists but does nothing useful | The submit verb emitted no initial ops | Review the `verbs/` file first |
| The worker runs but the site DB stays empty | The durable scripts never write projections or they errored out early | Add assertions around `ctx.dep(...)`, artifact extraction, and `site-db` writes |
| Pagination or detail fan-out duplicates work | The script emits duplicate child ops | Add workflow-local dedup checks or explicit dedup keys before changing the engine |

## See Also

- [scraper-new-developer-onboarding](scraper help scraper-new-developer-onboarding) — Suggested first-day path through the existing sites
- [scraper-runtime-model](scraper help scraper-runtime-model) — Why submit verbs and durable scripts are separate
- [scraper-architecture-overview](scraper help scraper-architecture-overview) — Broader system map
