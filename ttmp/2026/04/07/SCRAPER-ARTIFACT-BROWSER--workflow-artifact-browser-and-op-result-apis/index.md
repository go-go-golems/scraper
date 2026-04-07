---
Title: Workflow artifact browser and op result APIs
Ticket: SCRAPER-ARTIFACT-BROWSER
Status: active
Topics:
    - scraper
    - backend
    - frontend
    - http-api
    - artifacts
    - workflows
    - onboarding
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-07T15:24:00-04:00
WhatFor: "Plan and implement workflow-level artifact browsing, artifact download, and op-result retrieval without taking on JS replay yet."
WhenToUse: "Use when implementing or reviewing the backend and UI surfaces for browsing workflow artifacts and retrieving op results."
---

# Workflow artifact browser and op result APIs

## Overview

This ticket narrows the broader debugging problem down to the first useful slice:

- browse artifacts across an entire workflow,
- download and preview them quickly,
- retrieve real op results through a stable backend endpoint,
- prepare the UI for an artifact browser without taking on JS replay yet.

The codebase already has pieces of this:

- workflow and op inspection endpoints,
- per-op artifact listing,
- artifact download by ID,
- an op detail drawer that already renders artifacts and result-shaped data.

What is missing is the backend/API layer that turns those op-local pieces into a workflow-level browser. This ticket focuses on that missing layer and leaves JS replay/debug execution for a later, separate ticket.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Design guide**: [design/01-artifact-browser-and-op-result-implementation-guide.md](./design/01-artifact-browser-and-op-result-implementation-guide.md)
- **Investigation diary**: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)

## Status

Current status: **active**

This ticket is in active implementation. The first backend tasks are:

- workflow-level artifact listing,
- real op-result retrieval,
- tests for both.

## Topics

- scraper
- backend
- frontend
- http-api
- artifacts
- workflows
- onboarding

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
