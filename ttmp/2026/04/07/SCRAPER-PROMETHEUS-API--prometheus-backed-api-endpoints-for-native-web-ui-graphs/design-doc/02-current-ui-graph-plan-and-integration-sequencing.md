---
Title: Current UI graph plan and integration sequencing
Ticket: SCRAPER-PROMETHEUS-API
Status: active
Topics:
    - scraper
    - architecture
    - prometheus
    - grafana
    - frontend
    - http-api
    - react
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Maps current frontend components to the first set of Prometheus-backed graphs we can realistically ship without redesigning the UI layer."
LastUpdated: 2026-04-07T15:10:00-04:00
WhatFor: "Choose the first graph surfaces to implement with the existing React/MUI/Recharts setup."
WhenToUse: "Use when turning the Prometheus API design into concrete overview and queue-monitor graph work."
---

# Current UI graph plan and integration sequencing

## Executive Summary

The current frontend is already capable of showing a meaningful first wave of Prometheus-backed operator graphs. The limiting factor is not charting infrastructure; it is missing backend graph endpoints and page-level integration.

Today the UI already has:

- stat-card components for latest values
- progress/health preview components for queue utilization snapshots
- one reusable Recharts line-chart component in `web/src/components/queues/ThroughputChart.tsx`

That means we can ship a useful v1 without introducing a new chart library or redesigning the frontend information architecture.

The best first graphs are:

- queue completion rate
- queue depth over time
- queue wait p95
- retry/failure rate
- overview cards for workers up, active workflows, and worst queue wait

## Why This Matters

The backend Prometheus work already produced real data, and Grafana already proves that the data is useful. The missing piece is a product-facing choice about what the existing UI should show first.

Without this plan, implementation tends to drift into one of two bad patterns:

- trying to mirror full Grafana dashboards inside the app, or
- shipping one-off charts without a page-by-page operator story

This document avoids both problems by grounding the graph plan in the current UI components that already exist.

## Current UI Component Inventory

### `web/src/components/queues/ThroughputChart.tsx`

This is the only explicit chart component currently in the app. It uses Recharts and renders multiple line series against a shared time axis.

Important current properties:

- supports multiple lines
- expects a simple time-series shape
- already works inside MUI cards
- is named around throughput, but structurally can render any numeric time-series

Implication:

- we do not need a new charting package for the first graph rollout
- we may eventually want to rename or generalize this component, but that is optional for v1

### `web/src/components/overview/StatCardRow.tsx`

This component already renders four overview cards from engine snapshot data. It is suitable for the latest-value metrics that operators want to see at a glance.

Immediate fit:

- workers up
- active workflows
- worst queue wait p95
- total ready queue depth

### `web/src/components/overview/QueueHealthPreview.tsx`

This component is a snapshot health preview, not a historical graph. It should remain snapshot-oriented. It complements historical metrics rather than replacing them.

Implication:

- do not force historical charts into this component
- let it continue to show current in-flight utilization

### `web/src/components/overview/OpStatusBreakdown.tsx`

This is a snapshot distribution component showing current op counts by status. It should remain a current-state widget, not a time-series graph.

Implication:

- keep it as-is
- add historical metrics around it rather than inside it

## What We Can Show Right Now

## Queue page graphs

These fit the current queue page and existing line-chart component well.

### 1. Queue completion rate over time

Backing metrics:

- `scraper_ops_completed_total`

Query shape:

- range query
- grouped by `site`, `queue`, and optionally `status`

Why it is useful:

- tells operators whether the queue is making forward progress
- helps distinguish “backlogged but healthy” from “stuck”

Recommended v1 rendering:

- one line per queue on the global queue page, or
- one line per completion status on a queue detail view

### 2. Queue ready depth over time

Backing metrics:

- `scraper_queue_state_total{state="ready"}`

Why it is useful:

- shows whether work is accumulating
- pairs naturally with completion rate

Recommended v1 rendering:

- line chart

### 3. Queue running and in-flight depth over time

Backing metrics:

- `scraper_queue_state_total{state="running"}`
- `scraper_queue_state_total{state="in_flight"}`

Why it is useful:

- shows whether workers are actually consuming queue capacity
- makes rate-limit bottlenecks visible when combined with ready depth

Recommended v1 rendering:

- two or three lines on the same chart

### 4. Queue wait p95 over time

Backing metrics:

- `scraper:queue_wait_p95_15m`

Why it is useful:

- gives a robust operator-facing latency signal
- easier to reason about than raw histogram buckets in the UI

Recommended v1 rendering:

- single-line chart

### 5. Retry and failure rate over time

Backing metrics:

- `scraper_op_retries_total`
- `scraper_op_failures_total`

Why it is useful:

- exposes unhealthy verbs or degraded sites quickly
- helps builders and operators distinguish slow systems from failing systems

Recommended v1 rendering:

- one chart with separate retry/failure lines

### 6. Rate-limit/throttle rate over time

Backing metrics:

- `scraper_queue_rate_limited_total`

Why it is useful:

- shows whether queue policy is the bottleneck
- useful for tuning and capacity planning

Recommended v1 rendering:

- single-line chart on queue detail surfaces, not necessarily on the main overview

## Overview page graphs and cards

### 1. Workers up

Backing metrics:

- `scraper_workers_up`

Best rendering:

- stat card for current value
- optional small trend series later

### 2. Active workflows

Backing metrics:

- `scraper_engine_workflow_status_total{status=~"pending|running"}`

Best rendering:

- stat card for current value
- optional trend chart below

### 3. Worst queue wait p95

Backing metrics:

- `max(scraper:queue_wait_p95_15m)`

Best rendering:

- stat card

### 4. Workflow submit rate

Backing metrics:

- `scraper_workflows_submitted_total`

Best rendering:

- line chart on the overview page

### 5. Aggregate op completion rate

Backing metrics:

- `scraper_ops_completed_total`

Best rendering:

- line chart on the overview page

## What We Cannot Show Cleanly Yet

The current UI setup is fine for line charts and summary cards. It is not yet well-shaped for richer metric visualizations.

Not a good fit yet:

- histogram visualizations
- percentile bands
- stacked areas
- heatmaps
- annotation overlays from runtime events
- dual-axis charts
- dense per-queue small multiples

These are all possible later, but they would require either new reusable graph components or a stronger chart abstraction than the current queue throughput widget.

## Recommended Page-by-Page Plan

### Phase A: queue monitor replacement of placeholder data

Target page:

- `web/src/pages/QueueMonitorPage.tsx`

Use current UI pieces:

- keep `QueueStatusTable`
- keep `QueueDetailPanel`
- replace placeholder throughput with real Prometheus-backed line charts

Graphs to add first:

- completion rate
- ready depth
- queue wait p95

Rationale:

- this page already expects historical chart content
- it currently has obvious placeholder data
- replacing fake data with real data produces immediate product value

### Phase B: engine overview metric cards and top-level trends

Target page:

- `web/src/pages/EngineOverviewPage.tsx`

Use current UI pieces:

- extend `StatCardRow`
- add one or two compact line-chart cards below the existing snapshot widgets

Cards and graphs to add first:

- workers up
- active workflows
- worst queue wait p95
- workflow submit rate
- aggregate completion rate

Rationale:

- this gives operators a top-level historical pulse without rebuilding the page

### Phase C: queue-detail specific graphs

Target area:

- likely a richer queue detail panel or expanded queue card

Graphs to add:

- retry rate
- failure rate
- throttling rate
- running vs in-flight depth

Rationale:

- these are more diagnostic than the first-wave graphs
- they belong in deeper operator views, not the top-level overview

## API Contract Implications

The backend graph API should prioritize the graph families that the current UI can actually use.

Recommended first endpoint payloads:

- overview cards endpoint
- overview timeseries endpoint
- queue history endpoint

This means the first backend implementation does not need to solve every possible metric surface. It only needs to serve the data that the current UI can render well.

## Suggested DTO Shapes For Current UI

### Overview cards

```json
{
  "cards": [
    { "id": "workers-up", "label": "Workers Up", "value": 2, "unit": "count" },
    { "id": "active-workflows", "label": "Active Workflows", "value": 18, "unit": "count" },
    { "id": "worst-queue-wait-p95", "label": "Worst Queue Wait P95", "value": 42.5, "unit": "seconds" }
  ]
}
```

### Generic line-series shape

```json
{
  "series": [
    {
      "id": "ready-depth",
      "label": "Ready Depth",
      "unit": "ops",
      "points": [
        { "ts": "2026-04-07T19:00:00Z", "value": 4 },
        { "ts": "2026-04-07T19:01:00Z", "value": 6 }
      ]
    }
  ]
}
```

If the frontend keeps using the existing `ThroughputChart` shape directly, the backend or RTK Query adapter can translate this generic DTO into:

```json
[
  {
    "queueKey": "ready-depth",
    "points": [
      { "time": "19:00", "opsPerMin": 4 },
      { "time": "19:01", "opsPerMin": 6 }
    ]
  }
]
```

That translation layer is cheap and means we do not need to refactor all chart components before shipping real graphs.

## Implementation Sequencing For An Intern

1. Replace queue placeholder throughput first.
2. Add overview metric cards next.
3. Add overview trend charts after the cards.
4. Add deeper queue-detail charts only after the first two pages are stable.

This order matters because it aligns implementation effort with the highest visible product payoff.

## Task Matrix

| Priority | Page | UI surface | Graph/data | Why first |
| --- | --- | --- | --- | --- |
| P0 | Queue Monitor | Existing throughput card | Completion rate | Replaces fake data immediately |
| P0 | Queue Monitor | Existing throughput card | Ready depth | Makes backlog visible |
| P0 | Queue Monitor | Existing throughput card or second chart | Queue wait p95 | Best latency signal |
| P1 | Engine Overview | Stat cards | Workers up | Strong top-level signal |
| P1 | Engine Overview | Stat cards | Active workflows | Strong top-level signal |
| P1 | Engine Overview | Stat cards | Worst queue wait p95 | Strong top-level signal |
| P1 | Engine Overview | New chart card | Workflow submit rate | Useful historical context |
| P1 | Engine Overview | New chart card | Aggregate completion rate | Useful historical context |
| P2 | Queue Detail | Expanded diagnostics | Retry/failure/throttle trends | More diagnostic, less core |

## Recommendation

Do not over-engineer the frontend graph layer yet. Use the current line-chart and card components to ship a focused first set of operator graphs. The product already has enough UI structure to make these metrics useful.

The right move is:

- backend first: add the curated Prometheus-backed API endpoints
- frontend second: replace fake throughput and extend overview cards
- chart-system redesign later, only if the product genuinely outgrows the current line-chart/card model

## References

- `web/src/components/queues/ThroughputChart.tsx`
- `web/src/components/overview/StatCardRow.tsx`
- `web/src/components/overview/QueueHealthPreview.tsx`
- `web/src/components/overview/OpStatusBreakdown.tsx`
- `web/src/pages/QueueMonitorPage.tsx`
- `web/src/pages/EngineOverviewPage.tsx`
- `ops/monitoring/grafana/dashboards/scraper-overview.json`
