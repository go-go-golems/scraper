---
Title: SQLite store and engine view service decomposition
Ticket: SCRAPER-CLEANUP-STORE-VIEW
Status: closed
Topics:
    - scraper
    - backend
    - architecture
    - cleanup
    - sqlite
    - api
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Cleanup plan for decomposing the oversized SQLite store and engine view read service into smaller domain files."
LastUpdated: 2026-04-07T16:05:00-04:00
WhatFor: "Plan the decomposition of the storage and read-model layers without changing behavior."
WhenToUse: "Use when implementing or reviewing the store and engine view cleanup plan."
---

# SQLite store and engine view service decomposition

## Overview

This ticket covers two backend cleanup targets that have grown together over time:

- [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go)
- [service.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/services/engineview/service.go)

The goal is to split them by responsibility while keeping package boundaries and runtime behavior stable.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Design guide**: [design/01-store-and-engineview-decomposition-plan.md](./design/01-store-and-engineview-decomposition-plan.md)
- **Investigation diary**: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)

## Status

Current status: **closed**

This ticket is complete. The planned decomposition shipped, validation stayed green, and the ticket now remains as implementation history and review context.

## Topics

- scraper
- backend
- architecture
- cleanup
- sqlite
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
