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
