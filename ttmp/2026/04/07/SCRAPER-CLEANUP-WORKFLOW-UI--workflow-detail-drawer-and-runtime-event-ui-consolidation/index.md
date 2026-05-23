---
Title: Workflow detail drawer and runtime event UI consolidation
Ticket: SCRAPER-CLEANUP-WORKFLOW-UI
Status: active
Topics:
    - scraper
    - frontend
    - architecture
    - cleanup
    - react
    - events
    - workflows
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Cleanup plan for decomposing OpDetailDrawer and consolidating runtime-event rendering on RuntimeEventTable."
LastUpdated: 2026-04-07T16:05:00-04:00
WhatFor: "Plan the workflow UI cleanup and the removal of RuntimeEventList."
WhenToUse: "Use when implementing or reviewing the workflow UI consolidation."
---

# Workflow detail drawer and runtime event UI consolidation

## Overview

This ticket groups the main workflow-area frontend cleanup work:

- split [OpDetailDrawer.tsx](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/workflows/OpDetailDrawer.tsx) into tab-focused subcomponents
- converge on [RuntimeEventTable.tsx](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/workflows/RuntimeEventTable.tsx)
- remove [RuntimeEventList.tsx](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/workflows/RuntimeEventList.tsx) once callers are migrated

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Design guide**: [design/01-workflow-ui-cleanup-and-runtime-event-consolidation-plan.md](./design/01-workflow-ui-cleanup-and-runtime-event-consolidation-plan.md)
- **Investigation diary**: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)

## Status

Current status: **active**

This is a planning ticket. No workflow UI cleanup has been applied yet.

## Topics

- scraper
- frontend
- architecture
- cleanup
- react
- events
- workflows

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
