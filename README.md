# scraper

`scraper` is a durable workflow-driven scraping engine.

Go owns:
- workflow persistence
- scheduling and leases
- retries and queue policies
- worker runners (`js`, `http/fetch`)
- CLI and HTTP API hosts

JavaScript owns most site-specific behavior:
- submit verbs under `sites/<site>/verbs/`
- durable op scripts under `sites/<site>/scripts/`
- site-specific SQL projections under `sites/<site>/migrations/`

Site definitions are loaded from filesystem manifest directories during **bootstrap**, before the Cobra command tree is built. That is why commands such as `site js-demo run seed` only exist when scraper knows where the site manifests live.

## Repository layout

- `cmd/scraper/` — main CLI entrypoint
- `pkg/cmd/` — root command, bootstrap config, worker/api/site commands
- `pkg/engine/` — durable engine model, scheduler, runner registry, SQLite store
- `pkg/js/runtime/` — goja runtime and JS host APIs
- `pkg/sites/manifest/` — `site.yaml` loading and validation
- `sites/` — default site manifests, JS verbs/scripts, migrations, fixtures
- `pkg/doc/` — embedded help pages
- `web/` — frontend

## Bootstrap site loading

Scraper discovers site manifests from three sources, in this order:

1. app config file (`~/.scraper/config.yaml`)
2. environment variable (`SCRAPER_SITES_MANIFEST_DIRS`)
3. bootstrap CLI flags (`--sites-manifest-dir`)

Example config:

```yaml
sitesManifestDirs:
  - /absolute/path/to/sites
  - /another/path/to/sites
```

Example environment variable:

```bash
export SCRAPER_SITES_MANIFEST_DIRS="/path/to/sites-a:/path/to/sites-b"
```

Example CLI usage:

```bash
go run ./cmd/scraper --sites-manifest-dir ./sites site js-demo run seed --help
```

## Quickstart

### 1. Run the test suite

```bash
go test ./... -count=1
```

### 2. Submit a simple workflow

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

### 3. Run the worker

```bash
go run ./cmd/scraper \
  --sites-manifest-dir ./sites \
  worker run \
  --sites-dir "$tmpdir/sites" \
  --engine-db "$tmpdir/engine.db" \
  --max-cycles 16 \
  --poll-interval 5ms
```

### 4. Inspect engine state

```bash
go run ./cmd/scraper engine status --engine-db "$tmpdir/engine.db"
```

## HTTP API quickstart

Start the API server:

```bash
go run ./cmd/scraper \
  --sites-manifest-dir ./sites \
  api serve \
  --address 127.0.0.1:8080 \
  --engine-db /tmp/scraper-http-api/engine.db \
  --sites-dir /tmp/scraper-http-api/sites
```

Submit a workflow:

```bash
curl -X POST http://127.0.0.1:8080/api/v1/sites/js-demo/verbs/seed:submit \
  -H 'Content-Type: application/json' \
  -d '{
    "workflowID": "demo-http-001",
    "values": {
      "count": 3,
      "multiplier": 4,
      "prefix": "http"
    }
  }'
```

Then run the worker against the same engine/site DBs:

```bash
go run ./cmd/scraper \
  --sites-manifest-dir ./sites \
  worker run \
  --engine-db /tmp/scraper-http-api/engine.db \
  --sites-dir /tmp/scraper-http-api/sites \
  --max-cycles 16 \
  --poll-interval 5ms
```

## Help topics

Useful embedded help pages:

```bash
go run ./cmd/scraper --sites-manifest-dir ./sites help scraper-architecture-overview
go run ./cmd/scraper --sites-manifest-dir ./sites help scraper-runtime-model
go run ./cmd/scraper --sites-manifest-dir ./sites help scraper-bootstrap-config-and-site-manifest-loading
go run ./cmd/scraper --sites-manifest-dir ./sites help scraper-new-developer-onboarding
go run ./cmd/scraper --sites-manifest-dir ./sites help scraper-adding-a-declarative-site
```

## Current default site set

The repo currently ships a small progressive default set under `sites/`:

- `js-demo` — pure JS workflow path
- `hackernews` — JS + HTTP + JS
- `slashdot` — alternate HTML shape and pagination
- `nereval` — more complex fan-out and normalized projections

## Notes

- `--sites-dir` is the runtime directory for per-site SQLite databases.
- `--sites-manifest-dir` is the bootstrap directory for site definitions.
- If `site <name> run <verb>` is missing, scraper probably did not load the right manifest directories during bootstrap.
