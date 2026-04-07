---
Title: Prometheus-backed API endpoints for native web UI graphs
Ticket: SCRAPER-PROMETHEUS-API
Status: active
Topics:
    - scraper
    - architecture
    - prometheus
    - grafana
    - frontend
    - http-api
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-07T13:44:34.658742413-04:00
WhatFor: ""
WhenToUse: ""
---

# Prometheus-backed API endpoints for native web UI graphs

## Overview

This ticket defines the backend design for exposing Prometheus-backed graph data through scraper-owned HTTP APIs so the normal web UI can render historical charts without talking to Prometheus directly.

The current codebase already has a solid metrics foundation: the API server exposes `/metrics` in `pkg/api/server/server.go`, Prometheus scrapes the API and worker via `ops/monitoring/prometheus/prometheus.yml`, and Grafana dashboards exist under `ops/monitoring/grafana/dashboards/`. What is still missing is the application-facing bridge between those time-series backends and the React pages that currently rely on snapshots and placeholders.

The main recommendation in this ticket is:

- keep Prometheus as the time-series database
- keep Grafana as the rich operations dashboard
- add scraper-owned REST endpoints that expose a curated, normalized, low-cardinality subset of Prometheus data for the web UI
- do not let the browser query Prometheus directly
- do not expose arbitrary PromQL passthrough in the scraper API

This is an intern-facing design ticket. It explains the current state, the architectural tradeoffs, the recommended API shape, the implementation plan, and the validation steps needed to build the feature cleanly.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Main design doc**: [design-doc/01-prometheus-backed-api-endpoints-and-web-ui-graph-integration-guide.md](./design-doc/01-prometheus-backed-api-endpoints-and-web-ui-graph-integration-guide.md)
- **Investigation diary**: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)

## Status

Current status: **active**

This ticket is in design/research state. No runtime code is changed by this ticket yet.

## Topics

- scraper
- architecture
- prometheus
- grafana
- frontend
- http-api

## Tasks

See [tasks.md](./tasks.md) for the current task list.

High-level workstreams:

- document the current metrics and frontend gaps
- define the backend-only API design for Prometheus-backed graphs
- spell out route, service, DTO, query-registry, and validation details
- prepare the implementation backlog for a follow-on coding pass

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
