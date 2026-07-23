---
Title: Inventory and classify legacy Scraper engines and product paths
Ticket: SCRAPER-LEGACY-CLEANUP
Status: active
Topics:
    - scraper
    - workflow-v3
    - cleanup
    - architecture
    - onboarding
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-22T21:37:41.693056977-04:00
WhatFor: ""
WhenToUse: ""
---

# Inventory and classify legacy Scraper engines and product paths

## Overview

This ticket inventories Scraper legacy and canonical paths before destructive cleanup. It stops after classifying immediate removals and deferred removals with explicit replacement gates.

## Program navigation

- **Umbrella:** `EXPERIMENT-PLATFORM-CONVERGENCE`
- **Cleanup workstreams:** `RESEARCHCTL-LEGACY-CLEANUP`, `SCRAPER-LEGACY-CLEANUP`, `RAG-EVAL-LEGACY-CLEANUP`
- **Primary replacement ticket:** `SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER`
- **Cleanup report:** see the `analysis/` or `design-doc/` directory
- **Chronological evidence:** `reference/01-investigation-diary.md`

The umbrella and Researchctl cleanup live under `/home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/researchctl/ttmp/2026/07/22/`. Scraper cleanup lives under the sibling `scraper/ttmp/2026/07/22/`. RAG cleanup lives under `/home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system/ttmp/2026/07/22/`.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- scraper
- workflow-v3
- cleanup
- architecture
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
