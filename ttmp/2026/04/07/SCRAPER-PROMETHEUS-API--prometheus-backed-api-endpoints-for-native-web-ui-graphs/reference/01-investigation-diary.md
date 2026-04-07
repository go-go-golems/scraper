---
Title: Investigation diary
Ticket: SCRAPER-PROMETHEUS-API
Status: active
Topics:
    - scraper
    - architecture
    - prometheus
    - grafana
    - frontend
    - http-api
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-07T13:44:34.804157499-04:00
WhatFor: ""
WhenToUse: ""
---

# Investigation diary

## Goal

Capture the reasoning and concrete repository evidence behind a new backend API layer that exposes Prometheus-backed graph data to the scraper web UI.

## Context

The codebase already has Prometheus and Grafana integration from the `SCRAPER-PROMETHEUS-METRICS` work. That solved instrumentation, scrape configuration, alerts, and dashboards. It did not yet solve the product integration problem inside the scraper web application.

The web UI still shows current-state snapshots from scraper-owned APIs, while historical charts are either absent or placeholder-only. The clearest example is `web/src/pages/QueueMonitorPage.tsx`, which still uses hard-coded placeholder throughput data.

This ticket answers the next question: how should scraper expose historical graph data to its own frontend without pushing Prometheus or Grafana concerns into the browser?

## Quick Reference

### Key findings

- `pkg/api/server/server.go` already exposes `GET /metrics`, but there are no scraper-owned historical metrics endpoints.
- `web/src/pages/EngineOverviewPage.tsx` consumes only snapshot APIs: engine status and queue list.
- `web/src/pages/QueueMonitorPage.tsx` still renders static placeholder throughput series.
- `web/src/components/queues/ThroughputChart.tsx` is already a generic-enough multi-line Recharts widget even though its naming is throughput-specific.
- `web/src/components/overview/StatCardRow.tsx` is already a viable home for metrics-backed latest-value cards.
- `ops/monitoring/prometheus/prometheus.yml` already scrapes `scraper-api` and `scraper-worker`.
- `ops/monitoring/grafana/dashboards/scraper-overview.json` already contains time-series panels proving the raw data is available.

### Architectural recommendation

- frontend talks only to scraper API
- scraper API talks to Prometheus
- Grafana remains available for rich operator dashboards
- graph endpoints expose a small, curated, typed contract
- no arbitrary PromQL passthrough

### Candidate endpoints

- `GET /api/v1/metrics/overview`
- `GET /api/v1/metrics/queues/{site}/{queue}/history`
- `GET /api/v1/metrics/sites/{site}/history`
- `GET /api/v1/metrics/workers/history`
- `GET /api/v1/metrics/metadata`

### Current UI-compatible first graphs

- queue completion rate
- queue ready depth
- queue wait p95
- retry/failure rate
- workers up
- active workflows
- worst queue wait p95
- workflow submit rate

## Usage Examples

### Example backend flow

1. User opens the queue monitor page.
2. Frontend requests current queue state from existing queue APIs.
3. Frontend also requests queue history from `GET /api/v1/metrics/queues/{site}/{queue}/history`.
4. Scraper API maps that request to a fixed PromQL template.
5. Scraper API queries Prometheus HTTP API.
6. Scraper API normalizes the result into graph-friendly JSON.
7. Frontend renders the series without knowing any PromQL.

### Example graph-service pseudocode

```text
func QueueHistory(site, queue, range, step):
    validate site and queue parameters
    validate range and step are within allowed bounds
    query = registry.Render("queue-history", {
        site: site,
        queue: queue,
        range: range,
        step: step,
    })
    result = prometheusClient.QueryRange(query)
    return normalizeQueueHistory(result)
```

### Example normalized response shape

```json
{
  "graph": "queue-history",
  "site": "js-demo",
  "queue": "site:js-demo:js",
  "range": "6h",
  "stepSeconds": 60,
  "series": [
    {
      "id": "ready-depth",
      "label": "Ready",
      "unit": "ops",
      "points": [
        { "ts": "2026-04-07T16:00:00Z", "value": 3 },
        { "ts": "2026-04-07T16:01:00Z", "value": 5 }
      ]
    }
  ]
}
```

### Commands used during investigation

```bash
sed -n '1,260p' scraper/pkg/api/server/server.go
sed -n '1,260p' scraper/web/src/pages/QueueMonitorPage.tsx
sed -n '1,260p' scraper/web/src/pages/EngineOverviewPage.tsx
sed -n '1,260p' scraper/web/src/components/queues/ThroughputChart.tsx
sed -n '1,260p' scraper/web/src/components/overview/StatCardRow.tsx
sed -n '1,260p' scraper/web/src/components/overview/QueueHealthPreview.tsx
sed -n '1,260p' scraper/web/src/components/overview/OpStatusBreakdown.tsx
sed -n '1,260p' scraper/pkg/services/engineview/service.go
sed -n '1,260p' scraper/ops/monitoring/prometheus/prometheus.yml
sed -n '1,260p' scraper/ops/monitoring/grafana/dashboards/scraper-overview.json
docmgr vocab add --category topics --slug prometheus --description 'Prometheus metrics and query integration'
docmgr vocab add --category topics --slug grafana --description 'Grafana dashboards and graph integration'
docmgr doctor --ticket SCRAPER-PROMETHEUS-API --stale-after 30
remarquee upload bundle ttmp/2026/04/07/SCRAPER-PROMETHEUS-API--prometheus-backed-api-endpoints-for-native-web-ui-graphs --remote-dir /ai/2026/04/07/SCRAPER-PROMETHEUS-API --name 'Scraper Prometheus API endpoints for native web UI graphs' --non-interactive --force
remarquee cloud ls '/ai/2026/04/07/SCRAPER-PROMETHEUS-API/' --long --non-interactive
```

### Validation results

- `docmgr doctor --ticket SCRAPER-PROMETHEUS-API --stale-after 30` passed.
- The reMarkable bundle upload succeeded.
- Remote listing confirmed the uploaded document exists under `/ai/2026/04/07/SCRAPER-PROMETHEUS-API`.

## Related

- [../design-doc/01-prometheus-backed-api-endpoints-and-web-ui-graph-integration-guide.md](../design-doc/01-prometheus-backed-api-endpoints-and-web-ui-graph-integration-guide.md)
- [../design-doc/02-current-ui-graph-plan-and-integration-sequencing.md](../design-doc/02-current-ui-graph-plan-and-integration-sequencing.md)
- [../../SCRAPER-PROMETHEUS-METRICS--prometheus-metrics-and-operator-observability-architecture/index.md](../../SCRAPER-PROMETHEUS-METRICS--prometheus-metrics-and-operator-observability-architecture/index.md)
- `pkg/api/server/server.go`
- `web/src/pages/EngineOverviewPage.tsx`
- `web/src/pages/QueueMonitorPage.tsx`
- `web/src/components/queues/ThroughputChart.tsx`
- `web/src/components/overview/StatCardRow.tsx`
- `ops/monitoring/prometheus/prometheus.yml`
