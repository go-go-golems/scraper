---
Title: API server and worker command orchestration cleanup
Ticket: SCRAPER-CLEANUP-SERVER-WORKER
Status: active
Topics:
    - scraper
    - backend
    - architecture
    - cleanup
    - api
    - server
    - worker
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Cleanup plan for splitting API server wiring and worker command orchestration into smaller files."
LastUpdated: 2026-04-07T16:05:00-04:00
WhatFor: "Plan the cleanup of server and worker composition roots."
WhenToUse: "Use when implementing or reviewing the server/worker orchestration split."
---

# API server and worker command orchestration cleanup

## Overview

This ticket groups the orchestration-level cleanup work for:

- [server.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/api/server/server.go)
- [worker.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/worker.go)

The goal is to split route registration, middleware, runtime-event router wiring, worker runtime setup, worker metrics boot, and worker observer composition into clearer files.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Design guide**: [design/01-server-and-worker-orchestration-cleanup-plan.md](./design/01-server-and-worker-orchestration-cleanup-plan.md)
- **Investigation diary**: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)

## Status

Current status: **active**

This is a planning ticket. No orchestration cleanup has been applied yet.

## Topics

- scraper
- backend
- architecture
- cleanup
- api
- server
- worker

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
