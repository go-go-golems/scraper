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

## Step 3: Validate, commit, and stop before deletion

The inventory document, detailed task state, related-file evidence, and cross-ticket navigation were validated and committed. This is the requested stopping point: disposition is complete, but no removal tranche has been executed.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Stop after classification and leave destructive tasks open for review.

**Inferred user intent:** Make deletion a deliberate reviewed follow-up rather than an uninterrupted audit-and-delete operation.

**Commit (code):** `7543603ef3eec1c24a1401805a763100b1d31696` — cleanup inventory and classification documentation.

### What I did
- Checked the first seven inventory/classification tasks.
- Left review, immediate deletion, and deferred hard-cut tasks open.
- Ran docmgr doctor successfully.
- Cross-linked the cleanup report to key source files, the umbrella, sibling cleanup tickets, and replacement tickets.

### Why
- The remove-now list needs review before destructive edits.

### What worked
- Docmgr validation passed.
- Baseline test result: all packages passed.

### What didn't work
- No additional failure beyond those recorded in Step 2.

### What I learned
- Repository-local classifications can share one common umbrella without hiding different readiness levels.

### What was tricky to build
- The stopping point had to preserve enough evidence for a later agent to execute cleanup without rerunning the entire investigation. File references, deletion gates, exact commands, and open tasks provide that continuation state.

### What warrants a second pair of eyes
- Approve the remove-now table and any stated assumption about external users or disposable state.

### What should be done in the future
- After review, check the review task, execute only the immediate tranche, run validation, and commit it separately. Deferred paths remain until their named replacement tickets pass.

### Code review instructions
- Read the report's executive summary, immediate removal section, deferred removal section, and review checklist.
- Confirm tasks 8–10 remain open.

### Technical details
- Ticket: `SCRAPER-LEGACY-CLEANUP`.
- Stop condition reached: classification complete; deletion not started.

## Step 4: Execute the approved Scraper no-delete tranche

The user approved the remove-now tables. Scraper's table contained no active production deletion, so execution consisted of reconfirming the dependency boundary and validating the repository without manufacturing a deletion.

### What I did
- Preserved all old-engine product code because no V3 replacement product exists yet.
- Ran the full test suite and built all three Go binaries.
- Ran lint in module mode.
- Reverted unrelated `go mod tidy` direct/indirect movement for `golang.org/x/sys`.

### Validation and failure trace
- `GOWORK=off go test ./... -count=1` passed.
- `make build-go` passed.
- The first `make lint` ran in workspace mode and failed while type-checking a local `go-go-goja` dependency because `goja_nodejs` expected `goja.IsNumber`, `goja.IsBigInt`, and `goja.IsString`. This did not involve changed Scraper code.
- `GOWORK=off .bin/golangci-lint run -v ./cmd/... ./pkg/...` passed with zero issues, matching the repository's test/build module isolation.

### Review guidance
Confirm the repository has no implementation diff beyond this ticket update. The next destructive operation remains the separately gated V3 product hard cut.

## Step 5: Execute the deferred whole-product hard cut

### Prompt Context

**User prompt (verbatim):** "Ok, continue working through the tickets until completion. Use a goal."

**Assistant interpretation:** Continue through accepted replacement tickets and remove the old Scraper product only after every local and downstream owner has cut over.

**Inferred user intent:** End with Workflow V3 as the sole engine, without aliases, compatibility services, or duplicate persistence and telemetry.

### What I did and why

The prior inventory found 49 production importers and correctly deferred deletion. Since then, the Workflow V3 product, observations, Researchctl runner, RAG execution, RAG intake, external-operation custody, reproducible analysis, and scripted TTC gates all passed. A fresh RAG search found zero imports of old Scraper packages.

I deleted the old engine/store/scheduler/runners, convenience workflow package, site registry/manifests/scripts, JS operation runtime, API/services/types, metrics/runtime events/protobufs, frontend/Storybook output, and legacy dev stack in one tranche. Partial retention would preserve a second lifecycle. I simplified root bootstrap to a Workflow V3-only command tree and added a negative root test for `legacy`, `engine`, `api`, `site`, and `sites-manifest-dir`.

I updated the product guard from “legacy must still exist” to the final invariant: zero local and downstream importers and absent old commands. Active README/help now describe only Workflow V3. `go mod tidy` removed Watermill, Redis, Sessionstream, Prometheus, WebSocket, protobuf, and other dependencies owned solely by the deleted product.

### Failures and fixes

1. Compile initially failed with `pattern tutorials/*: no matching files found` after obsolete tutorials were deleted. I narrowed the embed directive to retained topics.
2. The first lint attempt used workspace mode and failed in the workspace's incompatible `goja_nodejs`/`goja` pair (`undefined: goja.IsNumber`). Go tests/build already passed in the repository's supported `GOWORK=off` mode. I fixed the Makefile lint/lintmax targets to enforce that same module boundary; lint then passed with zero issues.
3. The old product guard intentionally failed conceptually because it required a legacy worker and nonzero callers. I converted it to a deletion guard and added RAG downstream verification.

### Validation and review guidance

Full tests passed; Workflow V3 runtime's real isolation suite completed in 82.820s. All binaries built. The built-binary tmux smoke completed a submitted run through a separately started worker, stable follow, read API, rejected unauthorized cancellation, and accepted bearer-authorized cancellation. The final guard reports zero local and downstream importers. Review the deleted package clusters, `pkg/cmd/root.go`, `Makefile`, module dependency reduction, and `analysis/02-deferred-hard-cut-closure-audit.md`.

### Future guard

Scraping functionality may return only as a closed versioned task package with Workflow V3 custody. Do not restore old row projections, manual operation retry, dynamic root site commands, old event truth, or a second frontend/API lifecycle.
