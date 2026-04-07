---
Title: Workflow artifact browsing and per-op JavaScript replay debugging
Ticket: SCRAPER-OP-DEBUGGER
Status: active
Topics:
    - scraper
    - architecture
    - frontend
    - backend
    - http-api
    - javascript
    - jsverbs
    - onboarding
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-07T15:12:00-04:00
WhatFor: ""
WhenToUse: ""
---

# Workflow artifact browsing and per-op JavaScript replay debugging

## Overview

This ticket studies how to make workflow debugging practical for three closely related use cases:

- an operator who wants to quickly browse workflow artifacts and logs,
- a site author who wants to understand why a specific JS op produced the wrong output,
- a new engineer who needs a reliable way to replay a single script with real intermediate data.

The current system already has useful pieces:

- durable workflows and ops in SQLite,
- artifact storage and download endpoints,
- op-level script metadata,
- runtime-event history,
- a JS execution context that already knows how to resolve dependency results through `ctx.dep(...)`.

What is missing is a coherent “debug surface” that ties those primitives together. Today a user can inspect some workflow information in the UI, but cannot yet:

- quickly browse all workflow artifacts in one place,
- pull together an op’s full debugging context as one bundle,
- replay one JS script with the exact workflow input, op input, dependency results, and artifacts that existed during the real run,
- iteratively debug a script without resubmitting or rerunning the entire workflow.

This ticket provides the design and implementation guide for that missing layer.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Main design guide**: [design/01-workflow-artifact-browser-and-js-op-replay-debugger-guide.md](./design/01-workflow-artifact-browser-and-js-op-replay-debugger-guide.md)
- **Investigation diary**: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)

## Status

Current status: **active**

This ticket is currently a design/research ticket. It does not implement the debugger yet. The documentation has been validated with `docmgr doctor` and published to reMarkable for review.

## Topics

- scraper
- architecture
- frontend
- backend
- http-api
- javascript
- jsverbs
- onboarding

## Tasks

See [tasks.md](./tasks.md) for the current task list.

High-level workstreams:

- document the current artifact/result/debugging seams,
- define the recommended backend API and CLI contracts,
- define the frontend artifact browser and replay-launcher surfaces,
- spell out an implementation order that is useful for an intern.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
