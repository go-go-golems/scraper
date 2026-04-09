---
Title: New Developer Onboarding
Slug: scraper-new-developer-onboarding
Short: "Step-by-step onboarding path for a new contributor using the current filesystem-loaded sites and engine commands."
Topics:
- scraper
- onboarding
- tutorial
- sites
- testing
Commands:
- scraper
- engine
- worker
- site
Flags:
- engine-db
- sites-dir
- max-pages
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial gets a new contributor from zero context to a working mental model and a set of successful smoke tests. It deliberately starts with fixture-backed or pure-JS paths so the reader can validate the engine without depending on live websites. By the end, the reader will have seen the submit-verb flow, the worker loop, the engine visibility commands, and one complex site package.

The goal is not to memorize every file. The goal is to establish a safe first-day path through the codebase and the CLI.

## Prerequisites

Before starting, the reader should have:

- a working Go toolchain
- the repo checked out with local workspace dependencies available
- the ability to run `go test ./...`
- basic familiarity with Go, Cobra, and JavaScript

All commands below assume the current working directory is the `scraper/` repository root.

## Step 1 — Read The Architecture Pages

Start with the shortest conceptual pages before reading implementation code. This gives names to the moving parts and reduces confusion when you see submit verbs, scheduler code, and JS op scripts later.

Read these pages in order:

1. `scraper help scraper-architecture-overview`
2. `scraper help scraper-runtime-model`
3. `scraper help scraper-queue-policies-and-rate-limiting`

Then skim these code files:

1. `pkg/cmd/root.go`
2. `pkg/cmd/site.go`
3. `pkg/cmd/worker.go`

## Step 2 — Run The Full Test Suite

The fastest sanity check is the whole Go test suite. This confirms that the engine, site manifests, embedded help pages, and fixture-backed workflows all load correctly in the current environment.

```bash
go test ./... -count=1
```

If this fails, stop and debug the environment before moving on.

## Step 3 — Smoke-Test The Pure-JS Path

`js-demo` is the smallest useful site. It proves the split between submit verbs and the worker without involving any HTTP.

First submit work:

```bash
tmpdir=$(mktemp -d)

go run ./cmd/scraper \
  --sites-manifest-dir ./sites \
  site js-demo run seed \
  --sites-dir "$tmpdir/sites" \
  --engine-db "$tmpdir/engine.db" \
  --workflow-id demo-1 \
  --count 3 \
  --multiplier 4 \
  --prefix smoke
```

The flags `--count`, `--multiplier`, and `--prefix` are defined in `sites/jsdemo/verbs/seed.js` using `__verb__` metadata. The submit-verb host discovers these JS declarations and wires them into Cobra CLI flags automatically. This pattern is used by all default sites.

Then inspect the engine DB:

```bash
go run ./cmd/scraper engine status --engine-db "$tmpdir/engine.db"
```

You should see one workflow and ready work, but not a completed workflow yet. Now run the worker:

```bash
go run ./cmd/scraper \
  --sites-manifest-dir ./sites \
  worker run \
  --sites-dir "$tmpdir/sites" \
  --engine-db "$tmpdir/engine.db" \
  --max-cycles 16 \
  --poll-interval 5ms
```

Re-run engine status after that. The workflow should now be succeeded and the result/artifact counts should be non-zero.

## Step 4 — Smoke-Test An HTTP Site

Now move to a site that uses the full `js -> http/fetch -> js -> site-db` path. Hacker News is the simplest HTTP site.

All sites use the same two-step pattern: submit work with a verb, then run the worker. The hackernews verb defines `--base-url` and `--max-pages` flags in `sites/hackernews/verbs/seed.js`.

```bash
tmpdir=$(mktemp -d)

go run ./cmd/scraper \
  --sites-manifest-dir ./sites \
  site hackernews run seed \
  --sites-dir "$tmpdir/sites" \
  --engine-db "$tmpdir/engine.db" \
  --workflow-id hn-test \
  --base-url "https://news.ycombinator.com/" \
  --max-pages 1
```

Then run the worker to execute the queued ops:

```bash
go run ./cmd/scraper \
  --sites-manifest-dir ./sites \
  worker run \
  --sites-dir "$tmpdir/sites" \
  --engine-db "$tmpdir/engine.db" \
  --max-cycles 32 \
  --poll-interval 25ms
```

This path proves that JS emits HTTP work, the HTTP runner persists artifacts, and the follow-up JS extractor writes rows into the site DB.

For fully offline testing, the `go test ./...` suite uses fixture-backed tests that serve embedded HTML from local HTTP test servers.

## Step 5 — Inspect A Complex Site Without Going Live

The first complex site is `nereval`. Its value is not just parsing HTML. It proves:

- submit-verb driven workflow creation
- ASP.NET list-page pagination with explicit form state
- detail-page fan-out
- normalized site DB writes

Do not run it live as part of onboarding. Instead, study these files:

- `sites/nereval/site.yaml`
- `sites/nereval/verbs/seed.js`
- `sites/nereval/scripts/seed.js`
- `sites/nereval/scripts/extract_list.js`
- `sites/nereval/scripts/extract_detail.js`
- `sites/nereval/migrations/001_init.sql`

Then read the fixture-backed test:

- `pkg/cmd/site_test.go`

## Step 6 — Learn The Debugging Commands

The minimum useful operator debugging set is:

```bash
scraper engine status
scraper engine migrations status
scraper site migrate <site>
scraper worker run --max-cycles 1
scraper help <slug>
```

These commands are enough to answer:

- was the engine DB created?
- are migrations applied?
- did a site DB get created?
- is the worker leasing anything?
- where is the missing conceptual documentation?

## Step 7 — Read The Tickets Only After The Embedded Docs

The ticket docs are still valuable, but they should now be second-pass reading rather than the only onboarding path.

Read these if you need deeper implementation history (search for these ticket IDs in the `ttmp/` directory):

- `SCRAPER-DESIGN` — initial design guide and investigation diary
- `SCRAPER-RATE-LIMITER` — queue rate limiter analysis and implementation guide

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| `go test ./...` fails immediately | Workspace dependencies or generated docs are not loading | Fix the environment before debugging scraper logic |
| `site js-demo run seed` works but nothing completes | The worker was never run | Use `worker run` against the same temp DBs |
| `site js-demo run seed` is missing entirely | scraper did not load the site manifests during bootstrap | Pass `--sites-manifest-dir ./sites`, set `SCRAPER_SITES_MANIFEST_DIRS`, or configure `~/.scraper/config.yaml` |
| You do not know whether a bug is engine or site specific | Too many layers are being changed at once | Reproduce first on `js-demo`, then on `hackernews` or `slashdot`, then on `nereval` |
| `nereval` feels too big to start with | You skipped the simpler sites | Go back to `js-demo` and one HTTP site first |

## See Also

- `scraper help scraper-architecture-overview` — High-level map of the repository
- `scraper help scraper-runtime-model` — Submit verbs, workers, and op execution explained in more detail
- `scraper help scraper-js-api-reference` — Complete JavaScript API reference
- `scraper help scraper-adding-a-site` — Step-by-step Go-native site-authoring path once onboarding is complete
- `scraper help scraper-bootstrap-config-and-site-manifest-loading` — How scraper finds site manifests before building dynamic site commands
