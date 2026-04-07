---
Title: Investigation diary
Ticket: SCRAPER-PROMETHEUS-METRICS
Status: active
Topics:
    - scraper
    - architecture
    - frontend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Chronological investigation notes for the Prometheus and operator metrics review, including the current observability inventory, architectural conclusions, mistakes, and proposed next tickets.
LastUpdated: 2026-04-07T12:18:00-04:00
WhatFor: Preserve the evidence and reasoning behind the Prometheus recommendation so a future engineer can understand what scraper already exposes, what it does not, and why metrics should not replace durable state.
WhenToUse: Use when revisiting observability architecture, implementing metrics instrumentation, or checking how this ticket’s recommendations were derived.
---

# Investigation diary

## 2026-04-07

### Prompt context

The request was to create a new ticket and produce a detailed design/implementation guide for adding Prometheus and proper operator metrics. The guide needed to be detailed enough for a new intern and clearly explain what parts of scraper already provide observability, what Prometheus would add, and how the system should be instrumented without confusing metrics with durable product state.

### Interpretation

The key architectural question was not “can we add `/metrics`?” It was “what belongs in Prometheus, and what must stay in scraper’s own DB/API model?” That meant reviewing current snapshots, runtime events, worker/server topology, and frontend operator pages before recommending Prometheus.

### What I inspected

- `pkg/api/server/server.go`
- `pkg/api/handlers/catalog.go`
- `pkg/api/handlers/engine.go`
- `pkg/api/types/types.go`
- `pkg/cmd/worker.go`
- `pkg/services/engineview/service.go`
- `pkg/engine/store/sqlite/status.go`
- `pkg/engine/scheduler/scheduler.go`
- `pkg/engine/runner/http.go`
- `pkg/runtimeevents/backend.go`
- `pkg/runtimeevents/scheduler.go`
- `pkg/services/submission/service.go`
- `web/src/pages/EngineOverviewPage.tsx`
- `web/src/pages/QueueMonitorPage.tsx`
- `web/src/components/queues/QueueStatusTable.tsx`
- `web/src/api/engineApi.ts`
- `web/src/api/queueApi.ts`
- `docker-compose.yml`

### What worked

- The existing state and inspection seams are clear. The engine snapshot service already returns current counts and queue state, so it was easy to distinguish “current object state” from “historical metrics”.
- The runtime-event system already provides a second observability channel, which made it easier to explain why Prometheus should complement runtime events rather than replace them.
- The queue monitor page explicitly documents its placeholder throughput data, which made the current product gap concrete instead of speculative.

### What did not work

- A broad search for `metrics`, `counter`, `histogram`, and `prometheus` turned up almost no application code because scraper does not currently have a real metrics subsystem. That is informative, but it means the ticket has to define the basic package and endpoint structure from scratch.
- The worker is currently just a CLI loop, not an HTTP service, so Prometheus integration is not as simple as “add a handler.” The worker needs its own small metrics listener if it is to be scraped directly.

### Key findings

- Scraper currently has two observability layers:
  - durable state snapshots via the engine DB and JSON API,
  - live runtime events via Watermill and SSE.
- Scraper does not currently have:
  - a Prometheus registry,
  - `/metrics` endpoints,
  - real throughput history,
  - alerting-oriented time-series,
  - Grafana/dashboard integration.
- The frontend queue page still uses static random throughput data, so the product currently fakes part of the operator story.
- Prometheus is a good fit for:
  - numeric rates,
  - latency distributions,
  - saturation,
  - historical charts,
  - alerts.
- Prometheus is a bad fit for:
  - workflow graphs,
  - dependency explanations,
  - artifact navigation,
  - exact emitted ops,
  - high-cardinality per-workflow/per-op debugging.
- Therefore the right model is hybrid:
  - scraper owns durable object state and debugging detail,
  - Prometheus owns time-series metrics and alerting.

### Why the design guide recommends Prometheus

Prometheus reduces the amount of custom history and aggregation logic scraper would otherwise have to implement. The application does not need to build its own retention, rollups, or alerting semantics for counters, gauges, and histograms. But Prometheus does not remove all state from scraper. It only removes the need for scraper to manage time-series state for operational metrics. The domain model still needs to live in scraper.

### What warrants a second pair of eyes

- Whether the first implementation should let Grafana be the only historical metrics UI, or whether scraper should also proxy a small set of Prometheus queries back into its own frontend.
- Whether the worker metrics endpoint should be mandatory or optional behind a flag.
- Whether queue wait time should be measured directly at lease time or inferred from op timestamps in a less precise but simpler way.
- Whether request and HTTP runner metrics should include the site and queue labels in all cases, or only when that context is available.

### Reproduction notes

Useful commands for re-running the investigation:

```bash
cd /home/manuel/workspaces/2026-03-23/js-scraper/scraper
rg -n "prometheus|metrics|runtime-events|http-proxy|QueueMonitorPage|LeaseReadyOp|EngineStatus" pkg web cmd docker-compose.yml
sed -n '1,220p' pkg/api/server/server.go
sed -n '1,220p' pkg/cmd/worker.go
sed -n '260,420p' pkg/services/engineview/service.go
sed -n '1,220p' web/src/pages/QueueMonitorPage.tsx
docmgr doctor --ticket SCRAPER-PROMETHEUS-METRICS --stale-after 30
remarquee cloud ls '/ai/2026/04/07/SCRAPER-PROMETHEUS-METRICS/' --long --non-interactive
```

### Related

- Main design guide: [../design-doc/01-prometheus-metrics-architecture-and-implementation-guide-for-operator-observability.md](../design-doc/01-prometheus-metrics-architecture-and-implementation-guide-for-operator-observability.md)

## 2026-04-07 Implementation Slice 1

### Goal

Land the first usable Prometheus implementation slice without touching the frontend: create a reusable metrics package, expose API-server metrics, expose worker metrics, and instrument the main request/submission/scheduler/runner paths.

### What I changed

I added a new `pkg/metrics` package with:

- an explicit custom Prometheus registry,
- request counters and duration histograms,
- submission counters,
- scheduler counters and duration histograms,
- generic op-duration metrics,
- HTTP-runner-specific counters and duration histograms,
- worker liveness gauge,
- a scrape-time snapshot collector that exports engine and queue gauges from `engineview.Service`.

I then wired that into:

- the API server in `pkg/api/server/server.go`,
- the submission service in `pkg/services/submission/service.go`,
- the runner registry setup in `pkg/cmd/runtime_helpers.go`,
- the worker command in `pkg/cmd/worker.go`,
- the scheduler event model in `pkg/engine/scheduler/scheduler.go`.

### Why I chose this shape

I used an explicit custom registry instead of the default global Prometheus registry because:

- it keeps tests cleaner,
- it avoids accidental duplicate registrations across repeated server construction in tests,
- it makes it easier to reason about which collectors belong to which process.

I used a scrape-time snapshot collector for engine and queue gauges because the application already has trustworthy read paths for that data. That is simpler and less drift-prone than trying to manually increment and decrement many gauges throughout the code.

### Tricky parts

- The worker is not already an HTTP service, so metrics exposure needed a small sidecar HTTP server inside the worker command rather than just adding a handler.
- Scheduler event metrics needed runner kind for leased/retried ops, so I extended `scheduler.Event` with `RunnerKind` and set it from the leased op’s kind.
- HTTP runner metrics could not be added directly in the runner package via a metrics dependency because that would create an import cycle. I solved that by instrumenting through the generic metrics runner wrapper and extracting HTTP status-class information from the op result envelope.
- Queue wait histograms are not cleanly available from the current store interface, so I explicitly left that for a follow-up slice instead of guessing.

### Validation performed

Commands run:

```bash
cd /home/manuel/workspaces/2026-03-23/js-scraper/scraper
gofmt -w pkg/metrics/*.go pkg/api/server/server.go pkg/services/submission/service.go pkg/cmd/runtime_helpers.go pkg/cmd/worker.go pkg/engine/scheduler/scheduler.go pkg/api/server/server_test.go pkg/cmd/root_test.go
go test ./pkg/metrics ./pkg/api/server ./pkg/services/submission ./pkg/cmd ./pkg/engine/scheduler ./pkg/engine/runner -count=1
go test ./pkg/metrics ./pkg/api/server ./pkg/cmd -count=1
```

Results:

- all targeted package tests passed,
- `/metrics` endpoint coverage was added in `pkg/api/server/server_test.go`,
- worker help coverage was updated in `pkg/cmd/root_test.go`.

### What is still pending after this slice

- queue wait histogram design and implementation,
- stable failure counters by error code/category in scheduler-level metrics,
- worker metrics smoke test,
- full `go test ./... -count=1`,
- Prometheus and Grafana compose wiring,
- local scrape-health runbook,
- any frontend integration.

## 2026-04-07 Implementation Slice 2

### Goal

Make the Prometheus work locally testable by adding Prometheus and Grafana to Compose, provisioning a starter dashboard stack, writing an operator/developer smoke playbook, and validating the end-to-end scrape path with real processes.

### What I changed

I extended `docker-compose.yml` with:

- `prometheus`
- `grafana`
- the existing `redis` service left intact

I added monitoring config under:

- `ops/monitoring/prometheus/prometheus.yml`
- `ops/monitoring/grafana/provisioning/datasources/prometheus.yml`
- `ops/monitoring/grafana/provisioning/dashboards/dashboards.yml`
- `ops/monitoring/grafana/dashboards/scraper-overview.json`

I also added a ticket playbook:

- `playbook/01-local-prometheus-and-grafana-smoke-test.md`

### How the local topology works

The current local dev topology assumes:

- scraper API runs on the host at `0.0.0.0:8080`
- scraper worker runs on the host with `--metrics-address 0.0.0.0:9091`
- Prometheus and Grafana run in Docker
- Prometheus scrapes `host.docker.internal:8080` and `host.docker.internal:9091`

That is why the playbook explicitly uses `0.0.0.0` rather than `127.0.0.1`.

### Manual smoke procedure I ran

I used `tmux` for the long-running scraper processes as required by the repo guidance.

Commands used:

```bash
cd /home/manuel/workspaces/2026-03-23/js-scraper/scraper
lsof-who -p 3000 -k || true
lsof-who -p 8080 -k || true
lsof-who -p 9090 -k || true
lsof-who -p 9091 -k || true

docker compose up -d redis prometheus grafana

tmux new-session -d -s scraper-api \
  "cd /home/manuel/workspaces/2026-03-23/js-scraper/scraper && \
   go run ./cmd/scraper api serve \
     --address 0.0.0.0:8080 \
     --engine-db /tmp/scraper-prom-metrics-smoke/engine.db \
     --sites-dir /tmp/scraper-prom-metrics-smoke/sites"

tmux new-session -d -s scraper-worker \
  "cd /home/manuel/workspaces/2026-03-23/js-scraper/scraper && \
   go run ./cmd/scraper worker run \
     --engine-db /tmp/scraper-prom-metrics-smoke/engine.db \
     --sites-dir /tmp/scraper-prom-metrics-smoke/sites \
     --poll-interval 50ms \
     --metrics-address 0.0.0.0:9091"

curl -sf http://127.0.0.1:8080/metrics | rg 'scraper_http_requests_total|scraper_engine_workflows_total'
curl -sf http://127.0.0.1:9091/metrics | rg 'scraper_scheduler_cycles_total|scraper_workers_up'

curl -s -X POST http://127.0.0.1:8080/api/v1/sites/js-demo/verbs/seed:submit \
  -H 'Content-Type: application/json' \
  -d '{"workflowID":"prom-smoke-001","values":{"count":3,"multiplier":4,"prefix":"prom"}}'

curl -s http://127.0.0.1:9090/api/v1/targets
curl -s -u admin:admin http://127.0.0.1:3000/api/datasources
curl -s -u admin:admin 'http://127.0.0.1:3000/api/search?query=Scraper%20Overview'
```

### What worked

- `docker compose config` validated successfully.
- API `/metrics` returned scraper metrics.
- Worker `/metrics` returned scheduler-cycle and worker-up metrics.
- Prometheus saw both `scraper-api` and `scraper-worker` targets as `health: up`.
- Grafana had the provisioned `Prometheus` datasource.
- Grafana search returned the starter `Scraper Overview` dashboard.
- `go test ./... -count=1` passed after these changes.

### What went wrong

- I accidentally ran `gofmt` against the Markdown playbook, which of course failed with `expected 'package', found '--'`. That was harmless, but it belongs in the diary because it is exactly the kind of sloppy command mistake that wastes time if hidden.
- I also forgot to quote the Grafana search URL the first time, and zsh treated the `?query=...` segment as globbing input. Quoting the URL fixed it immediately.

### Remaining work after this slice

- add scheduler failure counters by stable error code/category,
- design and implement queue wait histograms,
- add more direct metrics-specific tests for runner and snapshot collectors,
- optionally add alert rules and recording rules,
- keep frontend work out of this ticket unless explicitly requested later.
