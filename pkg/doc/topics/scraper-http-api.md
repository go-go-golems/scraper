---
Title: Scraper HTTP API
Slug: scraper-http-api
Short: "Explains the local HTTP API server, how it maps to submit verbs and workers, and how to smoke test it."
Topics:
- scraper
- http-api
- api
- server
- workflows
Commands:
- api
- worker
- site
- engine
Flags:
- address
- engine-db
- sites-dir
- read-timeout
- write-timeout
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The HTTP API is the networked counterpart to the existing CLI submit and inspection commands. It is intentionally narrow. The API server accepts requests, resolves JS submit verbs, writes durable workflow state into the engine DB, and exposes read-only inspection endpoints. It does not replace the worker process.

The main mental model is:

```text
HTTP API -> submit initial durable work
worker run -> execute queued ops
```

That split is the same one used by `scraper site <site> run <verb>` and `scraper worker run`. The HTTP layer is a thin host around the same services and submit-verb runtime.

## Command

Start the server with `scraper api serve`. In practice, most local runs also need bootstrap site manifests loaded first, so the full command usually looks like:

```bash
scraper \
  --sites-manifest-dir ./sites \
  api serve \
  --address 127.0.0.1:8080 \
  --engine-db state/engine.db \
  --sites-dir state/sites
```

Important flags:

- `--sites-manifest-dir`: directory or directories containing site manifests used to build the site/verb catalog before the command tree is executed
- `--address`: loopback bind address for the local server
- `--engine-db`: durable engine SQLite database
- `--sites-dir`: directory holding per-site DBs
- `--read-timeout`: HTTP read timeout
- `--write-timeout`: HTTP write timeout

## Runtime Boundary

The API does not execute full workflows inline. A submit request runs exactly one JS submit verb under `sites/<site>/verbs/*.js`. That JS function can seed the workflow by emitting initial ops. Those ops are then picked up later by the worker process.

This is the same split described in `scraper help scraper-runtime-model`:

- submit-verb runtime: workflow-building JS
- op runtime: durable worker-side JS or `http/fetch`

## Endpoints

The first server slice exposes:

- `GET /healthz`
- `GET /api/v1/info`
- `GET /api/v1/sites`
- `GET /api/v1/sites/{site}`
- `GET /api/v1/sites/{site}/verbs`
- `GET /api/v1/sites/{site}/verbs/{verb}`
- `POST /api/v1/sites/{site}/verbs/{verb}:submit`
- `GET /api/v1/engine/status`
- `GET /api/v1/engine/migrations`
- `GET /api/v1/workflows/{workflowID}`
- `GET /api/v1/workflows/{workflowID}/ops`

Catalog endpoints expose the same JS/Glazed metadata the CLI uses for `site <site> run <verb>`. That makes the API suitable for future form generation or a small dashboard without inventing a second schema language.

## Smoke Test

Start the server:

```bash
scraper \
  --sites-manifest-dir ./sites \
  api serve \
  --address 127.0.0.1:8080 \
  --engine-db /tmp/scraper-http-api/engine.db \
  --sites-dir /tmp/scraper-http-api/sites
```

Submit a `js-demo` workflow:

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

In another terminal, process the queued work:

```bash
scraper \
  --sites-manifest-dir ./sites \
  worker run \
  --engine-db /tmp/scraper-http-api/engine.db \
  --sites-dir /tmp/scraper-http-api/sites \
  --max-cycles 16 \
  --poll-interval 5ms
```

Then inspect the workflow:

```bash
curl http://127.0.0.1:8080/api/v1/workflows/demo-http-001
curl http://127.0.0.1:8080/api/v1/workflows/demo-http-001/ops
curl http://127.0.0.1:8080/api/v1/engine/status
```

## Code Map

Start here if you need to change the implementation:

1. `pkg/cmd/api.go`
2. `pkg/api/server/server.go`
3. `pkg/api/handlers/`
4. `pkg/services/catalog/service.go`
5. `pkg/services/submission/service.go`
6. `pkg/services/engineview/service.go`
7. `pkg/sites/submitverbs/host.go`

## Common Mistakes

- Expecting the API server to run the scheduler itself by default. It does not. Start `scraper worker run` separately.
- Forgetting to provide site manifest directories during bootstrap. If the API starts with no manifests loaded, the site/verb catalog will be empty.
- Treating submit verbs like worker-side op scripts. Submit verbs seed workflows; scripts do the durable scraping later.
- Posting arbitrary JSON fields that are not declared in the JS verb metadata. The API validates values against the generated Glazed schema and returns `400` on unknown or incompatible fields.

## See Also

- `scraper help scraper-runtime-model`
- `scraper help scraper-architecture-overview`
- `scraper help scraper-new-developer-onboarding`
- `scraper help scraper-bootstrap-config-and-site-manifest-loading`
