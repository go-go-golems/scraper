---
Title: Prometheus metrics and operator observability architecture
Ticket: SCRAPER-PROMETHEUS-METRICS
Status: active
Topics:
    - scraper
    - architecture
    - frontend
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Ticket index for the Prometheus and operator metrics review and implementation, covering the observability model, Prometheus integration, metric taxonomy, endpoint design, dashboard strategy, rule bundle, playbooks, and phased rollout results.
LastUpdated: 2026-04-07T13:10:00-04:00
WhatFor: Track the design work required to add proper operator-facing metrics and Prometheus integration to scraper without confusing metrics with durable workflow state.
WhenToUse: Use when planning metrics, dashboards, alerting, worker or server instrumentation, or deciding how the frontend should surface current state versus historical time-series.
---

# Prometheus metrics and operator observability architecture

## Overview

This ticket analyzes how scraper should expose operator-useful metrics and whether Prometheus should be used instead of building a custom metrics/history system inside the application.

The core conclusion is that scraper should keep owning durable object state such as workflows, ops, dependencies, artifacts, and runtime events, while Prometheus should own numeric time-series such as throughput, queue wait time, HTTP latency, retries, and saturation. The ticket explains why that split is low-regret, where instrumentation should be added, what labels are safe, and how Grafana and the frontend should consume the results.

## Key Links

- Main design doc: [design-doc/01-prometheus-metrics-architecture-and-implementation-guide-for-operator-observability.md](./design-doc/01-prometheus-metrics-architecture-and-implementation-guide-for-operator-observability.md)
- Investigation diary: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)
- Local smoke playbook: [playbook/01-local-prometheus-and-grafana-smoke-test.md](./playbook/01-local-prometheus-and-grafana-smoke-test.md)
- Metrics/operator guide: [playbook/02-metrics-and-grafana-operator-guide.md](./playbook/02-metrics-and-grafana-operator-guide.md)

## Status

Current status: **active**

Current review status:

- current observability seams inspected across server, worker, engine view, runtime events, and frontend queue/overview pages,
- metrics gaps identified and alternatives compared,
- detailed intern-facing design guide drafted,
- backend metrics implementation completed through dashboard, rules, and playbook slices,
- local Prometheus and Grafana workflow validated.

## Topics

- scraper
- architecture
- frontend

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- `design-doc/` - main Prometheus and operator metrics architecture guide
- `reference/` - investigation diary and supporting context
- `playbook/` - local validation and operator-use playbooks
- `tasks.md` - phased implementation backlog derived from the review
- `changelog.md` - summary of what was learned and published
