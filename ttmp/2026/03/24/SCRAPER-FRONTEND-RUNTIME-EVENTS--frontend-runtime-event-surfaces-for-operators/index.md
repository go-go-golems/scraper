---
Title: Frontend runtime event surfaces for operators
Ticket: SCRAPER-FRONTEND-RUNTIME-EVENTS
Status: active
Topics:
    - scraper
    - frontend
    - react
    - api
    - events
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/api/handlers/runtime_events.go
      Note: |-
        Backend API contract consumed by the proposed frontend surfaces
        backend API contract that the ticket expands in the frontend
    - Path: web/src/api/runtimeEventsApi.ts
      Note: |-
        Existing frontend runtime event history client and decode seam
        current runtime event client entrypoint
    - Path: web/src/pages/WorkflowDetailPage.tsx
      Note: |-
        Existing workflow-local runtime event UI that motivates the follow-up work
        existing runtime event timeline motivating this follow-up ticket
ExternalSources: []
Summary: Follow-up ticket that plans the next wave of frontend runtime event UX on top of the newly landed backend event pipeline.
LastUpdated: 2026-03-24T21:20:07-04:00
WhatFor: Track and document the next phase of frontend work that turns runtime events into operator-facing pages, panels, and widgets.
WhenToUse: Use when planning or implementing new runtime event pages, hooks, filters, panels, or dashboard widgets in the web app.
---


# Frontend runtime event surfaces for operators

## Overview

This ticket is the frontend follow-up to the runtime event backend work. The backend pipeline and the first workflow-local UI timeline are already in place. The purpose of this ticket is to document and plan the next frontend wave: reusable runtime event client abstractions, a global event console, op-scoped event inspection, live submission progress, and overview/queue event widgets.

The primary document is the intern-facing design and implementation guide in:

- `design-doc/01-frontend-runtime-event-surfaces-architecture-and-intern-implementation-guide.md`

The ticket is intentionally documentation-first. It exists to give a new engineer a safe map of the system before more UI code is written.

## Key Links

- Design doc:
  `design-doc/01-frontend-runtime-event-surfaces-architecture-and-intern-implementation-guide.md`
- Diary:
  `reference/01-investigation-diary.md`
- Tasks:
  `tasks.md`
- Changelog:
  `changelog.md`

## Status

Current status: **active**

This ticket currently contains:

- architecture analysis,
- an intern-oriented implementation guide,
- a phased task breakdown,
- a research diary.

## Scope

The ticket covers:

- frontend runtime event pages and panels,
- reusable stream and filter abstractions,
- operator UX built on the new runtime event API.

The ticket does not change:

- backend event transport,
- protobuf schema design,
- Redis backend behavior.

## Structure

- `design-doc/` contains the primary implementation guide
- `reference/` contains chronological research notes
- `tasks.md` tracks implementation phases
- `changelog.md` records ticket-level decisions and progress
