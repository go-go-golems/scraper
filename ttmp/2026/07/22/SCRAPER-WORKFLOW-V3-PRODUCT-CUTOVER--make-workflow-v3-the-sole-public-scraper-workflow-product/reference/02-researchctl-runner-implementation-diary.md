---
Title: Researchctl runner implementation diary
Ticket: SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER
Status: complete
Topics:
    - scraper
    - workflow-v3
    - durability
    - artifacts
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/scraper-workflow-runner/main.go
      Note: Integration executable (commit 917e5b6)
    - Path: repo://examples/research-runner/execution.json
      Note: Canonical portable contract fixture (commit 917e5b6)
    - Path: repo://pkg/researchrunner/runner.go
      Note: Strict Researchctl process bridge (commit 917e5b6)
    - Path: repo://pkg/taskpackages/researchfixture/package.go
      Note: Deterministic retry and operation fixture (commit 917e5b6)
    - Path: repo://pkg/workflowv3product/fixture_module.go
      Note: Host-owned fixture operation authority (commit 917e5b6)
ExternalSources: []
Summary: Scraper-side implementation record for the EXPERIMENT-PLATFORM-SCRAPER-RUNNER cross-repository phase.
LastUpdated: 2026-07-23T12:30:00-04:00
WhatFor: Review the strict runner contract, fixture package, observation export, and cancellation implementation in Scraper.
WhenToUse: Read with the owning Researchctl ticket's primary diary.
---


# Researchctl runner implementation diary

## Goal

Record the Scraper-side changes for the Researchctl-to-Workflow V3 runner. The owning program ticket and complete cross-repository diary remain in Researchctl under `EXPERIMENT-PLATFORM-SCRAPER-RUNNER`.

## Step 1: Add the strict process runner and deterministic evidence fixture

Scraper gained a dedicated executable that consumes one Researchctl attempt request and delegates all subordinate lifecycle work to the completed Workflow V3 product. The implementation does not import Researchctl Go code, mirror Researchctl custody, or add a second Workflow scheduler. Data-only wire structs and a cross-repository canonical fixture keep the executable release boundary explicit.

The acceptance task package runs two JavaScript nodes. Its transform node admits one failed external operation on attempt one and succeeds on attempt two; its publish node emits the final result. The runner exports only lineage, closed failure fields, integer metrics, canonical operation evidence, and selected verified outputs.

### Prompt Context

**User prompt (verbatim):** "next phase"

**Assistant interpretation:** Implement the Scraper half of the accepted `EXPERIMENT-PLATFORM-SCRAPER-RUNNER` phase and keep repository-local implementation evidence.

**Inferred user intent:** Make Scraper independently runnable as Researchctl's durable workflow data plane without introducing RAG-specific or bidirectionally coupled code.

**Commit (code):** `917e5b661d562a7b76d27b75f03f059b4ae2f10e` — `feat: add Researchctl Workflow V3 runner`

### What I did
- Added `cmd/scraper-workflow-runner` and `pkg/researchrunner`.
- Added exact contract decoding, plan/catalog/package/input verification, opaque IDs, cancellation, observation projection, output copying, and external-operation export.
- Added `research-runner-fixture` and its least-authority host module.
- Added `workflow researchctl-config` and runnable examples.
- Added success, retry, operation, failure, privacy, mismatch, cancellation, strictness, and CLI tests.

### Why
- Scraper owns Workflow mechanics and must expose portable evidence rather than its SQLite schema.
- A deterministic operation fixture proves retry-aware semantics without network or provider variability.

### What worked
- Focused product/runtime/SQLite/CLI tests and lint passed.
- The cross-repository matrix executed four scientific runs, retained internal retries, verified three artifacts per attempt, resumed all four, and fenced timeout cancellation.

### What didn't work
- Supplying store-owned external-operation completion time caused a closed internal task failure; the caller field was removed.
- Sharing a CommonJS file leaked the transform task's operation-module import into the publish entrypoint and caused `GoError: Invalid module`; separate files restored least authority.

### What I learned
- Bundle-file boundaries are authority boundaries.
- Cancellation must use a fresh bounded cleanup context after the process signal cancels ordinary work.

### What was tricky to build
- Exporting enough evidence for audit without leaking task messages required explicit projections rather than serializing snapshots directly.
- The catalog contract verifies package name, version, bundle digest, and aggregate catalog digest before submission.

### What warrants a second pair of eyes
- Review protocol drift handling and artifact-size limits.
- Review future package-signing needs.

### What should be done in the future
- Keep new observation semantics in the Workflow observations ticket; do not expand this bridge into an analysis layer.

### Code review instructions
- Start at `pkg/researchrunner/runner.go`.
- Run `GOWORK=off go test ./pkg/researchrunner ./pkg/workflowv3product -count=1`.
- Run the owning Researchctl ticket's cross-repository smoke script.

### Technical details
- Protocol: `researchctl-runner-stdio/v1`.
- Domain config: `scraper-workflow-execution/v1`.
- Runner identity: `scraper-workflow-runner@v1`.
