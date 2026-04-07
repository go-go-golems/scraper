---
Title: 'UI Redesign: Runtime Events, Workflow Detail, and System UX Improvements'
Ticket: UI-001
Status: active
Topics:
    - frontend
    - ux-design
    - ui-rework
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: web/src/components/layout/AppShell.tsx
      Note: Root layout — needs breadcrumbs and error boundary
    - Path: web/src/components/workflows/RuntimeEventList.tsx
      Note: Component to be replaced by RuntimeEventTable — uses list layout instead of table
    - Path: web/src/features/runtime-events/runtimeEventFeed.ts
      Note: Core SSE hook that needs time range and pause/resume support
    - Path: web/src/pages/QueueMonitorPage.tsx
      Note: Queue page with fake throughput data and poor expansion UX
    - Path: web/src/pages/RuntimeEventsPage.tsx
      Note: Primary page with broken single-select filters and bloated event list
    - Path: web/src/pages/WorkflowDetailPage.tsx
      Note: Workflow detail with unfiltered embedded events and no tab layout
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-07T11:33:00.503681243-04:00
WhatFor: ""
WhenToUse: ""
---


# UI Redesign: Runtime Events, Workflow Detail, and System UX Improvements

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- frontend
- ux-design
- ui-rework

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
