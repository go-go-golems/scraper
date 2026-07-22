---
Title: Durable External Operation Evidence Ledger
Ticket: SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS
Status: active
Topics:
    - workflow-v3
    - durability
    - observability
    - privacy
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design ticket for generic, failure-durable external-call admission, measurement, accounting, export, and research custody in Workflow V3."
LastUpdated: 2026-07-22T19:55:00-04:00
WhatFor: "Coordinate implementation of a reusable external-operation evidence ledger across scraper Workflow V3, RAG evaluation, and researchctl custody."
WhenToUse: "Use as the entry point for architecture, implementation tasks, decisions, and investigation history for external-operation evidence."
---

# Durable External Operation Evidence Ledger

## Overview

Workflow V3 already persists attempts, leases, retries, budgets, output references, and operational events. This ticket designs the missing nested-effect ledger: one durable pre-call admission and one immutable post-call completion for each provider, HTTP, browser, database, or tool operation performed by trusted host code.

The ledger is intended to preserve precise provider-wall latency, outcome, usage, concurrency, and overlap evidence even when a task fails, is canceled, loses its lease, retries, or the process restarts. It uses a closed privacy-safe schema and does not persist payloads, URLs, headers, credentials, source text, provider bodies, vectors, arbitrary metadata, or error messages.

## Architectural boundary

- **scraper / Workflow V3:** generic operation authority, persistence, fencing, budget linkage, projections, and canonical export.
- **RAG evaluation:** generation/embedding descriptors, safe failure classification, provider measurements, and study reductions.
- **researchctl:** verified immutable artifact custody and reviewed derived metrics; never the live workflow recorder.

## Primary documents

- [Durable External Operation Evidence Ledger Design and Implementation Guide](design-doc/01-durable-external-operation-evidence-ledger-design-and-implementation-guide.md)
- [Investigation Diary](reference/01-investigation-diary.md)
- [Implementation Tasks](tasks.md)
- [Changelog](changelog.md)

## Proposed implementation order

1. Domain contracts and validation.
2. Additive SQLite schema and durable admission/completion APIs.
3. Host-only runtime injection and operation-ticket fencing.
4. Budget reservation reconciliation.
5. Projections and canonical export.
6. RAG provider integration and failed-cell custody.
7. researchctl verified artifact/metric mapping.
8. Failure, restart, race, privacy, fixture, and bounded real qualification.

## Current status

**Design complete; implementation not started.** Eleven implementation tasks are open. No paid execution should depend on this mechanism until generic and RAG-specific fixture failure tests pass.
