---
Title: HTTP API for durable scraper engine
Ticket: SCRAPER-HTTP-API
Status: active
Topics:
    - scraper
    - http-api
    - api
    - server
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Ticket workspace for designing an HTTP API around the durable scraper system, with emphasis on reusing JS submit verbs for workflow submission and keeping worker execution as a separate polling process.
LastUpdated: 2026-03-23T21:20:00-04:00
WhatFor: Tracks the design and future implementation of a Go-hosted HTTP API for the scraper engine.
WhenToUse: Use when implementing, reviewing, or onboarding contributors to the scraper HTTP server workstream.
---

# HTTP API for durable scraper engine

## Overview

This ticket defines the HTTP API workstream for the scraper system. The goal is to add a Go-hosted HTTP server that:

- discovers site JS submit verbs
- exposes them through stable HTTP endpoints
- submits durable workflows into the existing engine DB
- exposes read-side endpoints for engine and workflow inspection
- keeps actual work execution in the existing worker polling process

The intended architecture is not "run the whole scraper over one HTTP request." It is:

- HTTP API for submission and inspection
- worker for durable execution
- JS submit verbs for workflow planning
- JS op scripts for later workflow execution

## Key Links

- Design guide: [01-http-api-architecture-and-implementation-guide.md](./design-doc/01-http-api-architecture-and-implementation-guide.md)
- Diary: [01-investigation-diary.md](./reference/01-investigation-diary.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **active**

Current ticket state:

- design guide written
- implementation tasks detailed
- HTTP API bootstrap and core endpoints implemented
- `js-demo` submission and worker follow-up tested end to end
- embedded help page added
- ticket refresh upload pending

## Topics

- scraper
- http-api
- api
- server

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
