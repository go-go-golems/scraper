---
Title: Configurable queue rate limiter for durable scheduler
Ticket: SCRAPER-RATE-LIMITER
Status: closed
Topics:
    - scraper
    - scheduler
    - rate-limiting
    - token-bucket
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Dedicated planning ticket for evolving the scraper scheduler from one-active-op-per-queue serialization to a durable configurable token-bucket rate limiter."
LastUpdated: 2026-03-23T16:43:06.49798567-04:00
WhatFor: "Collects the design, task breakdown, and investigation record for future implementation of queue-level token-bucket rate limiting."
WhenToUse: "Use when planning or implementing durable queue pacing, queue policy configuration, and scheduler/store changes for HTTP and JS queues."
---

# Configurable queue rate limiter for durable scheduler

## Overview

This ticket exists to address a known gap in the current scraper engine. The engine already has queue-domain serialization, but it does not yet have a real configurable queue rate limiter. The deliverables in this ticket explain the gap, propose a durable token-bucket design, and break future implementation into phases.

## Key Links

- [Design guide](./design-doc/01-queue-rate-limiter-analysis-and-implementation-guide.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Status

Current status: **closed**

This is currently a planning and design ticket. No engine code changes are proposed in this ticket itself.

## Topics

- scraper
- scheduler
- rate-limiting
- token-bucket

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
