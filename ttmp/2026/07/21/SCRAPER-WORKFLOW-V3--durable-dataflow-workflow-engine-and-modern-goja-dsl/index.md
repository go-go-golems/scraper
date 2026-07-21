---
Title: Durable dataflow workflow engine and modern Goja DSL
Ticket: SCRAPER-WORKFLOW-V3
Status: active
Topics:
    - architecture
    - scheduler
    - goja
    - javascript
    - scraper
    - workflows
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Ticket hub for the workflow-v3 architecture, JavaScript cookbook, evidence catalogue, experiments, and implementation diary.
LastUpdated: 2026-07-21T17:50:00Z
WhatFor: Navigate the complete design and examples for scraper's compact durable dataflow engine and modern Goja workflow DSL.
WhenToUse: Start here when reviewing or implementing workflow v3.
---

# Durable dataflow workflow engine and modern Goja DSL

## Overview

This ticket designs scraper workflow v3 as a generic durable dataflow engine with compact references, continuous resource-aware dispatch, immutable attempts, projections, lazy expansion, bounded reduction, transactional budgets, and a typed `require("workflow")` authoring module. It also preserves the evidence and executable probes that motivated the design.

The architecture is proposed and documented; production implementation and the blocked real-provider TTC rerun remain future work.

## Key Links

- [Primary architecture and implementation guide](design-doc/01-durable-dataflow-workflow-v3-and-modern-scripting-architecture.md)
- [Reproducible JavaScript task bundles and worker registries](design-doc/02-reproducible-javascript-task-bundles-and-worker-registries.md)
- [JavaScript cookbook and execution atlas](reference/03-workflow-v3-javascript-cookbook-and-execution-atlas.md)
- [Source catalogue and evidence map](reference/02-source-catalogue-and-evidence-map.md)
- [Investigation diary](reference/01-investigation-diary.md)
- [Tasks](tasks.md)
- [Changelog](changelog.md)

## Status

Current status: **active**

## Topics

- architecture
- scheduler
- goja
- javascript
- scraper
- workflows

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- `design-doc/` — architecture and implementation guide
- `reference/` — diary, evidence map, and JavaScript cookbook
- `scripts/` — reproducible evidence and syntax-validation tools
- `scripts/output/` — deterministic probe/check results
- `sources/` — preserved historical and xgoja reference material
