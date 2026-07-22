---
Title: Investigation Diary
Ticket: SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS
Status: active
Topics:
    - workflow-v3
    - durability
    - observability
    - privacy
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system/internal/workflowv3ttc/provider.go
      Note: Real failure path that motivated the reusable ledger
    - Path: repo://pkg/workflowv3sqlite/store.go
      Note: Primary lease and transaction evidence inspected during Step 1
    - Path: ws://researchctl/internal/labsqlite/import.go
      Note: Verified transactional custody evidence inspected during Step 1
ExternalSources: []
Summary: Chronological evidence and decisions behind the Workflow V3 external-operation ledger design.
LastUpdated: 2026-07-22T19:55:00-04:00
WhatFor: Preserve the investigation path, evidence, failures, design decisions, validation, and continuation instructions for the external-operation ledger ticket.
WhenToUse: Read before resuming design or implementation work on SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS.
---


# Investigation Diary

## Goal

Record the evidence-first design of a generic, privacy-safe external-operation ledger for Workflow V3 and explain how it should integrate with RAG evaluation and researchctl custody.

## Step 1: Establish the failure boundary and design the reusable ledger

This step traced a real RAG sweep measurement loss through the Workflow V3 runtime, SQLite store, RAG provider adapter, sweep exporter, and researchctl laboratory. The investigation separated two concerns that had previously looked like one problem: provider timing was measured correctly, but it lived in successful task output and therefore disappeared when a task or later matrix cell failed.

The resulting design moves external-call admission and completion into generic Workflow V3 persistence. It retains scraper as the domain-neutral authority, keeps RAG responsible for provider semantics, and uses researchctl only after canonical evidence has been produced and verified.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to implement that, that is a good building block in the workflow/scraper package.

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new scraper docmgr ticket and produce an implementation-ready, intern-oriented architecture guide for a generic durable external-operation ledger, including cross-repository integration boundaries, then validate and upload the ticket documents to reMarkable.

**Inferred user intent:** Turn the TTC instrumentation lesson into reusable Workflow V3 infrastructure rather than adding another workload-specific sidecar, while preserving the architecture, privacy, and evidence-custody boundaries.

### What I did

- Created ticket `SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS` in the scraper repository.
- Added a design document, investigation diary, and eleven implementation tasks.
- Mapped Workflow V3 persistence, leasing, attempts, budgets, events, runtime construction, host module injection, and isolated-child protocol.
- Traced the RAG batch provider's provider-wall timer and the sweep's success-only output reduction.
- Inspected researchctl's observation sink, verified artifact contracts, canonical run export, and transactional import.
- Captured line-anchored references for all major claims.
- Designed separate immutable operation admission and completion tables, lease-fenced Begin, operation-ticket-fenced Finish, descriptor-controlled counters, budget reconciliation, canonical JSONL custody, and phased implementation/testing.
- Explicitly kept cumulative cross-run authority separate from per-Workflow operation authority.

Commands included:

```text
docmgr ticket create-ticket --ticket SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS --title 'Durable External Operation Evidence Ledger' --topics workflow-v3,durability,observability,privacy
docmgr doc add --ticket SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS --doc-type design-doc --title 'Durable External Operation Evidence Ledger Design and Implementation Guide'
docmgr doc add --ticket SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS --doc-type reference --title 'Investigation Diary'
rg --files pkg/workflowv3 pkg/workflowv3runtime pkg/workflowv3sqlite
rg -n 'ProviderMeasurement|AdmitGeneration|GenerateBatch|readCell|writeFailedCellCheckpoint' cmd/rag-ttc-v3-sweep internal/workflowv3ttc -S
```

### Why

Provider calls are common to scraper workloads, and their evidence has the same durability and authority requirements regardless of domain. Workflow already owns leases, cancellation, retries, resources, budgets, and attempts. Recording nested external effects anywhere else would duplicate authority and create cross-store races.

Researchctl is valuable after the operation ledger becomes immutable, but making it the live sink would couple provider execution to a separate laboratory runtime. The design therefore uses Workflow SQLite for live evidence and researchctl verified artifacts for scientific custody.

### What worked

- The existing lease transaction provides a strong template for pre-call operation admission.
- `TaskModuleContext` is a narrow host-only injection seam that does not require a new JavaScript capability.
- Existing researchctl artifacts and metrics can hold operation evidence without an immediate researchctl schema migration.
- The current RAG timer uses Go monotonic elapsed time, so the missing feature is durable persistence rather than a new timing algorithm.
- Additive `CREATE TABLE IF NOT EXISTS` migration matches the current Workflow V3 database strategy.

### What didn't work

I initially attempted to read a researchctl file that does not exist:

```text
ENOENT: no such file or directory, access '/home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/researchctl/pkg/lab/export.go'
```

The export contracts are in `pkg/lab/types.go`, validation is in `pkg/lab/validation.go`, canonical digest support is in `pkg/lab/artifacts.go`, and SQLite reconstruction is in `internal/labsqlite/query.go`. I located them with:

```text
rg -n 'type RunExport|func .*RunExport|RunExportDigest|ValidateRunExport' pkg/lab internal/labsqlite
```

No implementation code was changed in this step; this is a design ticket.

### What I learned

- Attempt completion fencing cannot be reused unchanged for operation completion. A call admitted under a valid lease may return after cancellation, and its evidence should persist even though stale task output must remain rejected.
- Separate immutable admission and completion rows model crash uncertainty more accurately than a mutable operation-status row.
- A generic metadata map would undermine privacy. Descriptor-controlled integer counters and digests are sufficient for performance work.
- WAL mode alone does not state the pre-call fsync guarantee. SQLite synchronous policy must be explicit and fault-tested.
- Restricted child processes cannot satisfy failure-durable observations through a terminal-only response. Parent-mediated streaming is a separate future protocol.

### What was tricky to build

The hardest design point was reconciling late evidence with stale-work fencing. Requiring the live lease during Finish would discard cancellation and timeout results. Ignoring fencing would let stale workers mutate arbitrary evidence. The solution is to require the live lease only for Begin and return a secret, operation-specific completion capability. Finish can append one bounded completion after lease invalidation, but it cannot publish task output, change node status, or admit another call.

Budget integration was also subtle. Operation accounting must explain existing attempt reservations, not become another budget ledger. The guide therefore allocates descriptor-declared reservation units at Begin and derives actual or conservative usage from operation completions, with a staged compare-first migration from current task-reported usage.

### What warrants a second pair of eyes

- Whether the entire Workflow V3 connection should use `synchronous=FULL` or operation authority should use a dedicated durable store.
- Whether operation descriptor digests must become compiler-pinned plan fields in the first implementation.
- The exact transition from `task.usage.report` to operation-derived budget dimensions.
- Whether input-size measures such as `input_runes` are acceptable under every deployment's privacy model.
- The operation ticket's idempotency and late-completion behavior under cancellation/lease-reclaim races.

### What should be done in the future

Implement the ticket phases in order. Do not run another paid RAG matrix until generic operation admission/completion, RAG integration, failed-cell export, race tests, privacy canaries, and fixture failure injection all pass.

### Code review instructions

Start with the design's Sections 6 through 12, then inspect:

1. `pkg/workflowv3sqlite/store.go` lease admission and `checkFence`;
2. `pkg/workflowv3runtime/engine.go` and `modules.go` injection seam;
3. RAG `internal/workflowv3ttc/provider.go` success-only measurement path;
4. researchctl `pkg/lab/runtime.go`, `types.go`, and `internal/labsqlite/import.go` custody boundary.

Validate the ticket with:

```text
docmgr doctor --ticket SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS --stale-after 30
docmgr validate frontmatter --doc ttmp/2026/07/22/SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS--durable-external-operation-evidence-ledger/design-doc/01-durable-external-operation-evidence-ledger-design-and-implementation-guide.md --suggest-fixes
git diff --check
```

### Technical details

The proposed persistence state machine is:

```text
no row
  -> lease-fenced durable admission
  -> admitted with no completion (running or unknown)
  -> ticket-fenced immutable completion
```

The proposed custody path is:

```text
Workflow SQLite
  -> canonical external-operations.jsonl + manifest
  -> verified researchctl artifact
  -> selected derived metrics
  -> report and graphs
```

## Step 2: Validate and publish the design bundle

This step validated the ticket's frontmatter, vocabulary, internal structure, and docmgr health, then published one bundled PDF to reMarkable. The bundle contains the ticket overview, full design guide, investigation diary, implementation tasks, and changelog so an intern can read the architecture and execution plan offline in one document.

The initial doctor run identified four ticket topics that were not yet in the scraper vocabulary. I added precise definitions, reran doctor cleanly, performed the required upload dry-run, and then completed the non-interactive upload.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Validate the completed ticket and deliver its full document bundle to reMarkable.

**Inferred user intent:** Make the design immediately reviewable and usable as an implementation handoff.

### What I did

- Validated design and diary frontmatter.
- Verified the design has 9,246 words, 1,579 lines, 134 headings, balanced code fences, and no placeholder comments.
- Added `workflow-v3`, `durability`, `observability`, and `privacy` to the scraper docmgr vocabulary.
- Ran `docmgr doctor` until it passed cleanly.
- Ran the reMarkable bundle upload in dry-run mode.
- Uploaded `SCRAPER WORKFLOW V3 EXTERNAL OPERATIONS DESIGN.pdf`.

### Why

A design document is only useful when its metadata is searchable, its ticket bookkeeping is valid, and the intended reader can access the full bundle. The dry-run protects against path and bundle mistakes before rendering and upload.

### What worked

- Both document frontmatter validations passed.
- `docmgr doctor` passed after vocabulary additions.
- The dry-run included exactly five intended ticket documents.
- The real upload returned:

```text
OK: uploaded SCRAPER WORKFLOW V3 EXTERNAL OPERATIONS DESIGN.pdf -> /ai/2026/07/22/SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS
```

### What didn't work

The first doctor run reported unknown topic warnings for `durability`, `observability`, `privacy`, and `workflow-v3`. These were vocabulary gaps rather than document errors. I added all four entries with non-placeholder descriptions and reran doctor successfully.

### What I learned

The scraper vocabulary predated Workflow V3's newer documentation themes. Adding the topics makes this ticket and later related work discoverable without weakening doctor validation.

### What was tricky to build

The upload bundle had to remain navigable despite a long implementation guide. A ToC depth of two preserves the major architecture and implementation sections without flooding the PDF outline with every subsection.

### What warrants a second pair of eyes

Review the rendered code blocks and ASCII diagrams on the reMarkable page size, especially the proposed SQL schema and API sketches. The upload succeeded, but visual layout remains a human review concern.

### What should be done in the future

Begin implementation with task `boyf` only after the proposed decisions and open questions have been reviewed.

### Code review instructions

Open the reMarkable bundle and inspect the executive summary, system map, schema, transaction pseudocode, implementation phases, and acceptance criteria. In the repository, run:

```text
docmgr doctor --ticket SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS --stale-after 30
git diff --check
```

### Technical details

- Remote directory: `/ai/2026/07/22/SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS`
- PDF: `SCRAPER WORKFLOW V3 EXTERNAL OPERATIONS DESIGN.pdf`
- Bundle inputs: index, design guide, diary, tasks, changelog.
- ToC depth: 2.
