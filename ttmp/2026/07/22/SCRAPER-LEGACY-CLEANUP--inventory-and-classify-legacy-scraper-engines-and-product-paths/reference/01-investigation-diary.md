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
