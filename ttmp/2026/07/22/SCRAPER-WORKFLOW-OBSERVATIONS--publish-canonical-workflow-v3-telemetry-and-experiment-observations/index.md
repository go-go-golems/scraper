---
Title: Publish canonical Workflow V3 telemetry and experiment observations
Ticket: SCRAPER-WORKFLOW-OBSERVATIONS
Status: complete
Topics:
    - scraper
    - workflow-v3
    - observability
    - durability
    - privacy
    - api
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-23T14:13:06.672273113-04:00
WhatFor: ""
WhenToUse: ""
---


# Publish canonical Workflow V3 telemetry and experiment observations

## Overview

This ticket delivers **Canonical retry-aware workflow metrics and traces** as part of the cross-repository `EXPERIMENT-PLATFORM-CONVERGENCE` program. Read the design guide before implementation; no legacy compatibility is required.

## Program navigation

The umbrella is **EXPERIMENT-PLATFORM-CONVERGENCE**. Every program ticket is listed here so this ticket remains discoverable in isolation:

- **EXPERIMENT-PLATFORM-CONVERGENCE** (researchctl) — Program architecture and cross-repository milestone map
- **RESEARCHCTL-EXPERIMENT-PLANS** (researchctl) — Generic cases, factors, replicates, ordering, scheduling, and resume
- **SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER** (scraper) — Workflow V3 public CLI, worker, extension model, and legacy deletion
- **EXPERIMENT-PLATFORM-SCRAPER-RUNNER** (researchctl) — Generic Researchctl-to-Scraper runner vertical slice
- **SCRAPER-WORKFLOW-OBSERVATIONS** (scraper) — Canonical retry-aware workflow metrics and traces
- **RAG-V2-WORKFLOW-LOWERING** (rag-eval) — RAG v2 task catalog and compiler to Workflow V3
- **RAG-GEPPETTO-WORKFLOW-OPERATIONS** (rag-eval) — Geppetto generation, embedding, and reranking operation adapters
- **RESEARCHCTL-EXPERIMENT-ANALYSIS** (researchctl) — Reproducible JS analysis, statistics, charts, and reports
- **RAG-V2-EXECUTION-CUTOVER** (rag-eval) — Hard-cut to the sole canonical RAG execution path
- **TTC-SCRIPTED-EXPERIMENT-ACCEPTANCE** (rag-eval) — Thin scripted TTC final acceptance workload

## Key links

- **Design and implementation guide:** see `design-doc/`
- **Investigation diary:** see `reference/01-investigation-diary.md`
- **Tasks:** see `tasks.md`
- **Related files:** see frontmatter `RelatedFiles`

## Status

Current status: **active**

## Topics

- scraper
- workflow-v3
- observability
- durability
- privacy
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
