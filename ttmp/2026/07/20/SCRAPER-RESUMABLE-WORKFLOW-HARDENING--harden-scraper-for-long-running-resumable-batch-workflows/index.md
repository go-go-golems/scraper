---
Title: Harden scraper for long-running resumable batch workflows
Ticket: SCRAPER-RESUMABLE-WORKFLOW-HARDENING
Status: complete
Topics:
    - architecture
    - scraper
    - scheduler
    - worker
    - sqlite
    - workflows
    - onboarding
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: 'Engine hardening roadmap for safe long-running resumable workflows: leases, blocked dependencies, true concurrency, immutable identity, observers, snapshots, and migration safety.'
LastUpdated: 2026-07-20T15:13:48.612282791-04:00
WhatFor: Implement scraper-level guarantees needed by expensive resumable batch consumers without duplicating workflow orchestration downstream.
WhenToUse: Use before changing scheduler/store state transitions, embedded workflow APIs, worker concurrency, operator retry, or runtime event observation.
---


# Harden scraper for long-running resumable batch workflows

## Overview

This ticket hardens scraper v0.0.4 for long-running, resumable, failure-isolated workflows. It records concrete defects in lease ownership, timestamp comparison, dependency recovery, single-process concurrency, workflow attachment, observer exposure, and cycle accounting, then gives a phased implementation and release plan.

The ticket preserves scraper's generic boundary: it improves durable engine guarantees and generic observer/snapshot APIs. RAG/provider semantics, researchctl evidence, Goja adapters, browser delivery, and Redis deployment remain consumer-layer concerns.

## Key Links

- [Long-running resumable workflow hardening architecture and implementation guide](./design-doc/01-long-running-resumable-workflow-hardening-architecture-and-implementation-guide.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Status

Current status: **active**

## Topics

- architecture
- scraper
- scheduler
- worker
- sqlite
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
