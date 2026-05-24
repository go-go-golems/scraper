---
Title: Shared Goja runtime helper consolidation for op and submit executors
Ticket: SCRAPER-CLEANUP-GOJA-RUNTIME
Status: active
Topics:
    - scraper
    - backend
    - architecture
    - cleanup
    - javascript
    - goja
    - jsverbs
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Cleanup plan for consolidating duplicated Goja runtime helpers while preserving separate op and submit executors."
LastUpdated: 2026-04-07T16:05:00-04:00
WhatFor: "Plan shared runtime helper extraction without collapsing two distinct executors into one."
WhenToUse: "Use when implementing or reviewing Goja runtime cleanup."
---

# Shared Goja runtime helper consolidation for op and submit executors

## Overview

This ticket addresses duplication between the durable op executor and the submit-verb executor. The goal is to share helper plumbing, not to merge the two execution phases into one universal executor.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Design guide**: [design/01-goja-runtime-helper-consolidation-plan.md](./design/01-goja-runtime-helper-consolidation-plan.md)
- **Investigation diary**: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)

## Status

Current status: **active**

This is a planning ticket. No helper extraction has been applied yet.

## Topics

- scraper
- backend
- architecture
- cleanup
- javascript
- goja
- jsverbs

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
