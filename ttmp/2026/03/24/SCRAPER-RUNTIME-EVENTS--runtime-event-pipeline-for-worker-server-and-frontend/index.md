---
Title: Runtime event pipeline for worker, server, and frontend
Ticket: SCRAPER-RUNTIME-EVENTS
Status: closed
Topics:
    - architecture
    - scraper
    - worker
    - server
    - http-api
    - scheduler
    - api
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/api/server/server.go
      Note: Current server role and future event API surface
    - Path: pkg/cmd/worker.go
      Note: Current worker role and lifecycle
    - Path: pkg/engine/scheduler/scheduler.go
      Note: Current event model that motivates the ticket
ExternalSources: []
Summary: Track the chosen Watermill-based architecture for delivering real-time worker and server events to the frontend, now with a protobuf-generated Go and TypeScript event contract.
LastUpdated: 2026-03-24T20:16:15-04:00
WhatFor: Track the decision and implementation plan for a runtime event pipeline that can carry scheduler events, logs, and other live operational data from scraper processes to operators and frontend clients.
WhenToUse: Use when implementing the Watermill-based runtime event pipeline, when wiring Redis-backed delivery between worker and server, or when extending dashboard-facing runtime visibility.
---


# Runtime event pipeline for worker, server, and frontend

## Overview

This ticket covers the missing event pipeline between the durable worker, the HTTP server, and the frontend.

The current codebase already emits scheduler events inside the worker process, but those events are ephemeral and stop at the local observer callback. The HTTP server reads engine state directly from SQLite and has no live connection to worker activity. That makes it easy to build polling-based dashboards for workflow state, but not enough to support real-time logs, event streaming, or cross-process operator telemetry.

The design comparison is now done. The ticket treats Watermill as the standard eventing layer, keeps the current worker/server split as the default topology, and uses a protobuf-generated event contract shared between Go and TypeScript. Redis-backed transport remains the cross-process path, with in-process transport for tests and optional local mode.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **closed**

Current ticket state:

- architecture inspected
- option matrix written
- Watermill decision recorded
- protobuf contract decision recorded
- implementation still pending

## Topics

- architecture
- scraper
- worker
- server
- http-api
- scheduler
- api

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
