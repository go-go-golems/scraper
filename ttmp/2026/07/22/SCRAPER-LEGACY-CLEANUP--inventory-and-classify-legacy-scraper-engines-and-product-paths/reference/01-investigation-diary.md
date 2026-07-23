---
Title: Investigation diary
Ticket: SCRAPER-LEGACY-CLEANUP
Status: active
Topics:
    - scraper
    - workflow-v3
    - cleanup
    - architecture
    - onboarding
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-22T21:37:42.18373643-04:00
WhatFor: ""
WhenToUse: ""
---

# Investigation diary

## Goal

Inventory and classify legacy paths in Scraper under the `EXPERIMENT-PLATFORM-CONVERGENCE` umbrella, without deleting code during this investigation tranche.

## Step 1: Create the cleanup workstream

This step created a repository-local cleanup ticket with a ten-item task list. The three cleanup tickets separate repository ownership while sharing one classification vocabulary and one umbrella.

The stopping condition is explicit: identify canonical code, immediate safe removals, and deferred removals with named replacement gates; document and commit the result; do not execute deletion yet.

### Prompt Context

**User prompt (verbatim):** "Ok, create a separate ticket for the cleanup, and a detailed list of tasks. Then, start working on the cleanup, keep a detaile ddiary as you work.

Once you have identified what can immediately be removed, what should bre placed later on and how, stop. Make sure to keep a detailed document about the cleanup in each repositories respective ticket, and cross link all the files and tickets so that we don't forget we are working under a common umbrella down the road.

commit at appropriat eintervals."

**Assistant interpretation:** Create one cleanup ticket per repository, audit active and legacy paths, cross-link everything, commit at useful milestones, and stop after disposition planning.

**Inferred user intent:** Reduce architecture work before adding capabilities, while avoiding blind deletion and preserving a durable shared plan.

### What I did
- Created `SCRAPER-LEGACY-CLEANUP` and a detailed task list.
- Linked the workstream conceptually to `EXPERIMENT-PLATFORM-CONVERGENCE` and the two sibling cleanup tickets.
- Defined retain, remove-now, and remove-after-replacement classifications.

### Why
- Cleanup decisions need repository-local evidence and ownership.
- A shared umbrella prevents three independent cleanup efforts from producing incompatible boundaries.

### What worked
- Docmgr created the ticket, primary cleanup document, diary, tasks, index, and changelog.

### What didn't work
- N/A in ticket setup.

### What I learned
- The cleanup stage needs an explicit stop before deletion so classification can be reviewed.

### What was tricky to build
- “Legacy” cannot mean merely old. The ticket therefore requires active-reference and replacement evidence for every disposition.

### What warrants a second pair of eyes
- Review the eventual remove-now list before destructive work begins.

### What should be done in the future
- Complete the source inventory and disposition report, then pause for review.

### Code review instructions
- Start with `tasks.md`, then read the cleanup inventory document and later diary steps.

### Technical details
- Umbrella: `EXPERIMENT-PLATFORM-CONVERGENCE`.
- Siblings: `RESEARCHCTL-LEGACY-CLEANUP`, `SCRAPER-LEGACY-CLEANUP`, `RAG-EVAL-LEGACY-CLEANUP`.

## Step 2: Map both Scraper workflow generations

This step mapped the main CLI and all old-engine production importers, then compared them with Workflow V3's library and binary surfaces. The result is intentionally conservative: there is no safe immediate production deletion because the old engine remains the entire public product.

Workflow V3 is canonical for the future, but cleanup must first provide a product path. The report therefore freezes the old engine and assigns each cluster a specific replacement and deletion gate.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Identify safe and deferred Scraper cleanup without deleting the active product prematurely.

**Inferred user intent:** Reduce duplicate engines decisively while avoiding a broken repository between deletion and replacement.

### What I did
- Traced `pkg/cmd/root.go` and `runtime_helpers.go` through old store, scheduler, runners, sites, metrics, events, API, and services.
- Searched all non-test imports of old and V3 packages.
- Counted about 49 non-test old-engine importers.
- Measured approximately 11,047 Go lines in old engine/workflow/JS runtime and 17,258 in V3 core/runtime/SQLite.
- Verified the only V3 binaries are isolation/task-worker infrastructure, not a public workflow product.
- Ran `GOWORK=off go test ./... -count=1`; all packages passed.

### Why
- Deleting `pkg/engine` now would also delete root commands, worker, API, site execution, services, metrics, and external RAG consumers.

### What worked
- Import analysis made the active dependency direction unambiguous.
- Workflow V3's extensive integration tests provide a strong replacement foundation.

### What didn't work
- The first staging check failed on trailing whitespace in the verbatim prompt and an extra blank line in `changelog.md`. Markdown normalization fixed it, and the second commit passed.

### What I learned
- Scraper cleanup is a product cutover, not an immediate package deletion.
- “No immediate deletion” is the correct evidence-backed classification for active Go code.

### What was tricky to build
- Site functionality mixes generic scheduling, domain task behavior, migrations, and dynamic commands. Each shipped site needs an explicit port/delete decision; a blanket package decision is too coarse.

### What warrants a second pair of eyes
- Decide which of the 38 shipped site files represent product value worth porting.
- Confirm whether tracked Storybook output is release input before a separate hygiene cleanup.

### What should be done in the future
- Freeze old-engine features and begin `SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER`; do not delete the old cluster until the minimal V3 CLI/worker/API slice passes.

### Code review instructions
- Start with `pkg/cmd/root.go:15-68` and `runtime_helpers.go:35-180`.
- Review the replacement matrix against `SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER`.

### Technical details
- Baseline tests: all passing; V3 runtime tests took about 113 seconds.
- Setup commit: `5bcfd05861d9eae1bcc23d76cead4c93c938c298`.
