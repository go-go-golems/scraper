---
Title: Adding a New Site
Slug: scraper-adding-a-site
Short: "Step-by-step guide for adding a new Go-native site when declarative manifests are not enough."
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
- sites-dir
- engine-db
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

Adding a site means fitting your scraper into the existing engine shape rather than inventing a new execution path. The durable engine is already responsible for scheduling, retries, HTTP execution, DB lifecycle, and queue policy. Your job is to describe the site's behavior in a site definition with submit verbs, scripts, migrations, fixtures, and tests.

This tutorial now covers the fallback Go-native path only. Use it when the site truly needs custom Go-owned behavior beyond what `site.yaml` plus JavaScript can express.

If your site can be expressed with `site.yaml` plus JavaScript, start with:

- `scraper help scraper-adding-a-declarative-site`

## Prerequisites

Before starting, make sure you understand:

- `scraper help scraper-architecture-overview`
- `scraper help scraper-runtime-model`
- `scraper help scraper-js-api-reference`

It also helps to read one simple site and one complex site under the repo-level `sites/` directory:

- `sites/jsdemo/`
- `sites/nereval/`

## Step 1 — Decide If You Really Need Go

Before writing any Go, start from a declarative site under `sites/<site>/` and ask what is still missing.

You only need this Go-native path when at least one of these is true:

- the site requires a custom native module not available through the manifest module IDs
- the site needs custom runtime registrars or other non-standard runtime wiring
- the site needs custom CLI behavior that cannot be expressed through JS verbs alone
- the site needs another Go-only integration point that the manifest loader cannot describe

If none of those are true, stop and use `scraper help scraper-adding-a-declarative-site` instead.

If you do need Go, create a normal Go package under `pkg/sites/<site>/` that contributes the extra runtime behavior, then point it at the declarative content under `sites/<site>/`.

Typical additions in a Go-native site are:

- extra module specs
- runtime registrars
- custom CLI wiring
- special test helpers
- other site-specific Go seams that cannot be shared

## Step 2 — Decide The First Submit Verb

Every site should have at least one operator entrypoint that seeds the first durable work. This lives in `verbs/` and uses `__verb__`.

Use `js-demo` as the smallest pattern:

- `sites/jsdemo/verbs/seed.js`

Use `hackernews` for an HTTP-based site:

- `sites/hackernews/verbs/seed.js`

Use `nereval` when the site needs more complex input:

- `sites/nereval/verbs/seed.js`

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
- `ctx.writeRecord(...)` and `ctx.writeArtifact(...)`

See `scraper help scraper-js-api-reference` for the complete API.

For examples:

- `sites/hackernews/scripts/extract_frontpage.js`
- `sites/slashdot/scripts/extract_frontpage.js`
- `sites/nereval/scripts/extract_list.js`
- `sites/nereval/scripts/extract_detail.js`

## Step 4 — Add Site DB Migrations

If the site needs queryable output, define it in `migrations/`. The site DB should contain projection tables, not engine workflow state.

Migration files are numbered SQL scripts (e.g. `001_init.sql`). The migration manager (`pkg/sites/migrate/`) discovers and applies them in order when `scraper site migrate <site>` runs, or automatically when the worker opens the site DB. Each migration runs in a transaction and is tracked so it only applies once.

Examples:

- `sites/jsdemo/migrations/001_init.sql`
- `sites/nereval/migrations/001_init.sql`

Keep the first migration small and query-oriented. Add only the tables that the first end-to-end workflow actually writes.

## Step 5 — Register Or Bootstrap The Site

The site still needs to be reachable during bootstrap so the root command can build dynamic site verbs.

There are two normal ways to do that:

1. put the site's declarative content in a directory that scraper loads during bootstrap (for example via `--sites-manifest-dir`, `SCRAPER_SITES_MANIFEST_DIRS`, or `~/.scraper/config.yaml`)
2. if the site truly needs Go-native extensions, make sure the constructor path that builds the root command is given the correct manifest directory and any required Go-native registration hooks

If you skip this step, your code may compile but the CLI will not expose the site verbs.

## Step 6 — Add Fixtures Before Live Traffic

Use fixtures first. They make parser tests fast, deterministic, and reviewable.

Good fixture sets usually include:

- one first page
- one later page if pagination exists
- one or more detail pages

Current examples:

- `sites/hackernews/fixtures/frontpage.html`
- `sites/slashdot/fixtures/frontpage.html`
- `sites/nereval/fixtures/`

## Step 7 — Add A Command-Path Test

Do not stop at parser unit tests. Add at least one test that exercises:

1. `site <site> run <verb>`
2. `engine status`
3. `worker run`
4. site DB assertions

The strongest current examples are in `pkg/cmd/site_test.go`.

That test proves the site works in the actual engine path rather than only in a hand-rolled helper.

## Step 8 — Run The Validation Loop

Before committing:

```bash
gofmt -w pkg/sites/<site> pkg/cmd/site_test.go
go test ./... -count=1
```

If the site adds or changes help pages, also verify:

```bash
go run ./cmd/scraper --sites-manifest-dir ./sites help <your-slug>
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| The CLI does not show the site | The site manifests were not available during bootstrap, or Go-native registration is incomplete | Check the bootstrap manifest dirs first, then review the root-command construction path |
| `site <site> run <verb>` exists but does nothing useful | The submit verb emitted no initial ops | Review the `verbs/` file first |
| The worker runs but the site DB stays empty | The durable scripts never write projections or they errored out early | Add assertions around `ctx.dep(...)`, artifact extraction, and `site-db` writes |
| Pagination or detail fan-out duplicates work | The script emits duplicate child ops | Add workflow-local dedup checks or explicit dedup keys before changing the engine |

## See Also

- `scraper help scraper-new-developer-onboarding` — Suggested first-day path through the existing sites
- `scraper help scraper-bootstrap-config-and-site-manifest-loading` — How manifest directories are discovered before dynamic site commands exist
- `scraper help scraper-runtime-model` — Why submit verbs and durable scripts are separate
- `scraper help scraper-js-api-reference` — Complete JavaScript API reference
- `scraper help scraper-architecture-overview` — Broader system map
