---
Title: 'Dashboard UI: Workflow Monitoring, Engine Health, Rate Limiting, Operations'
Ticket: SCRAPER-DASHBOARD
Status: closed
Topics:
    - dashboard
    - react
    - scraper
    - frontend
    - material-ui
    - redux
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/engine/model/types.go
      Note: Core domain types
    - Path: pkg/engine/scheduler/scheduler.go
      Note: Scheduler events and CycleResult
    - Path: pkg/engine/store/sqlite/status.go
      Note: EngineStatus inspection
    - Path: pkg/services/catalog/service.go
      Note: Backend service for site/verb discovery
    - Path: pkg/services/engineview/service.go
      Note: Backend service for engine status and workflows
    - Path: pkg/services/submission/service.go
      Note: Backend service for workflow submission
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-23T21:53:13.228762422-04:00
WhatFor: ""
WhenToUse: ""
---


# Dashboard UI: Workflow Monitoring, Engine Health, Rate Limiting, Operations

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **closed**

## Topics

- dashboard
- react
- scraper
- frontend
- material-ui
- redux

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
