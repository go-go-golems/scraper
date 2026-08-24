---
Title: Systemic design review of PR 10 workflow correctness findings
Ticket: SCRAPER-PR10-SYSTEMIC-REVIEW
Status: complete
Topics:
    - scraper
    - workflow-v3
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - https://github.com/go-go-golems/scraper/pull/10
Summary: Intern-oriented architecture and implementation review of four PR 10 correctness findings across input custody, set bounds, dependency readiness, and terminal lifecycle.
LastUpdated: 2026-08-23T16:55:18.32433526-04:00
WhatFor: Decide and implement a systemic correction for PR 10 rather than four drifting local patches.
WhenToUse: Start here when reviewing or implementing the PR 10 follow-up.
---


# Systemic design review of PR 10 workflow correctness findings

## Overview

This ticket analyzes all four inline review comments on `go-go-golems/scraper#10` and maps them through Researchctl input verification, Workflow V3 authoring and compilation, SQLite readiness, dynamic map/reduction lowering, runtime execution, and terminal state.

The verdict is that every comment is correct and has a local repair, but the defects share a systemic cause: each correctness fact currently has more than one owner. The proposed corrective slice gives immutable byte custody, set cardinality, producer readiness, and successful terminalization one authoritative boundary each.

## Primary documents

1. [PR 10 systemic correctness analysis and implementation design](design-doc/01-pr-10-systemic-correctness-analysis-and-implementation-design.md) — architecture primer, finding analysis, decision records, API sketches, diagrams, pseudocode, phased implementation, tests, risks, and file references.
2. [Investigation diary](reference/01-investigation-diary.md) — chronological GitHub, source, history, ticket, validation, and delivery evidence.
3. [Tasks](tasks.md) — completion checklist.
4. [Changelog](changelog.md) — ticket-level decisions and delivery status.

## Headline recommendation

Implement four focused phases rather than four isolated conditionals:

- stage verified bytes directly into the content-addressed artifact store;
- make set-input cardinality an explicit compiled ingress policy;
- derive one typed dependency graph from data refs and control-only `after` edges;
- centralize successful run completion in one transaction-local reconciler.

Preserve existing specialized SQLite readiness tables during this slice, but populate them from shared compiler-owned semantics. Add a topology matrix that proves compile, admission, scheduling, output, terminal, and reopen behavior agree.

## Review scope

- PR: https://github.com/go-go-golems/scraper/pull/10
- Reviewed commit: `202229464629e2b6d0e193ff7798b16770b3a270`
- Current analyzed merge: `e4578b8bcb17317c3fcccbde854c177c24993bdf`
- Core reviewed files are unchanged between those commits.

## Status

Current status: **complete**. The systemic design was implemented in focused custody, set-policy, dependency-graph, and lifecycle slices; all tests, race checks, build, lint, and ticket validation pass.

## Topics

- scraper
- workflow-v3
- architecture
