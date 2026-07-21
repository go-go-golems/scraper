---
Title: Investigation diary
Ticket: SCRAPER-WORKFLOW-V3
Status: active
Topics:
    - architecture
    - scheduler
    - goja
    - javascript
    - scraper
    - workflows
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/workflowv3
      Note: Initial canonical IR catalog compiler and focused tests (commit ff286a1)
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/design-doc/01-durable-dataflow-workflow-v3-and-modern-scripting-architecture.md
      Note: Primary design produced by the investigation
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/design-doc/02-reproducible-javascript-task-bundles-and-worker-registries.md
      Note: Step 5 custom JavaScript task-bundle design
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/reference/02-source-catalogue-and-evidence-map.md
      Note: Evidence map and continuation entry point
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/reference/03-workflow-v3-javascript-cookbook-and-execution-atlas.md
      Note: Step 4 cookbook deliverable
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/scripts/02-inventory-current-scripting.py
      Note: Reproducible scripting inventory described in Step 1
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/scripts/03-workflow-dsl-grammar-probe.mjs
      Note: Reproducible DSL grammar experiment described in Step 1
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/scripts/04-check-cookbook-js.py
      Note: Step 4 reproducible JavaScript syntax validator
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/scripts/05-js-task-bundle-registration-probe.mjs
      Note: Step 5 registration and matching experiment
ExternalSources:
    - https://parc.yolo.scapegoat.dev/note/research/kb/projects/scraper
    - https://parc.yolo.scapegoat.dev/note/research/kb/projects/go-go-goja
    - https://parc.yolo.scapegoat.dev/note/research/kb/projects/widget-dsl
Summary: Chronological evidence, experiments, failures, and design decisions behind scraper workflow v3 and its modern Goja authoring module.
LastUpdated: 2026-07-21T16:35:00Z
WhatFor: Resume or review the workflow-v3 investigation without losing the commands, failures, evidence boundaries, and implementation risks that shaped the design.
WhenToUse: Read before continuing implementation, reviewing the architecture, or validating the ticket deliverables.
---





# Diary

## Goal

Capture the evidence-first investigation and architecture work for a compact, work-conserving scraper workflow engine and reusable `require("workflow")` module. The diary records what was inspected, what experiments established, which failures mattered, and how to review or continue the work.

## Step 1: Preserve sources and map the current scripting and engine boundaries

The investigation first established what scraper already does well and where its current representation and scripting APIs become unsafe at TTC scale. Historical PARC material was copied into the ticket so the design can be reviewed without depending on a live site, and repository evidence was indexed before recommendations were written.

Two reproducible probes turned broad impressions into concrete facts. The inventory measured the duplicated execution/submission `ctx` surfaces and raw-operation exposure; the grammar probe tested a pure fluent workflow authoring shape with typed symbolic references, resources, map/reduce jobs, task descriptors, and deterministic serialization.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Resume the compacted workflow-v3 research session, finish the evidence-backed architecture guide and ticket bookkeeping, validate it, and prepare the reMarkable deliverable.

**Inferred user intent:** Produce an implementation-ready design that fixes the TTC storage/privacy and scheduling failures while making modern Goja/xgoja scripting reusable across scraper, researchctl, and RAG.

### What I did

- Created and ran `scripts/01-fetch-research-sources.sh` to preserve ten historical/reference extracts under `sources/`.
- Created and ran `scripts/02-inventory-current-scripting.py`.
- Stored its validated output at `scripts/output/current-scripting-inventory.json`.
- Created and ran `scripts/03-workflow-dsl-grammar-probe.mjs`.
- Stored its deterministic output at `scripts/output/workflow-dsl-grammar-probe.json`.
- Wrote `reference/02-source-catalogue-and-evidence-map.md`.
- Inspected scraper model, scheduler, SQLite migrations/store transactions, public `pkg/workflow` façade, events, metrics, execution scripts, submission verbs, site manifests, and documentation.
- Compared researchctl builders, the RAG v2 DSL/native module and TypeScript declarations, widget descriptor patterns, and xgoja/v2 provider/host-service APIs.
- Re-ran focused `rg -n` queries to capture exact implementation anchors for the final design.

### Why

- Recommendations about scheduler, persistence, and scripting must be traceable to current code rather than inferred from the TTC symptoms alone.
- Historical design notes explain why existing APIs took their current shape and identify reusable local patterns.
- Reproducible probes give future implementers a stable baseline and prevent the design from depending on undocumented manual inspection.

### What worked

- The inventory found 15 execution scripts, 8 submission verbs, and two independently assembled 11-member context surfaces.
- It confirmed that current scripts expose raw engine storage fields through `ctx.emit` and that site roots currently use CommonJS JavaScript rather than TypeScript.
- The grammar probe produced a pure plan with resources, map/reduce tasks, validation/explanation, and digest `sha256:8a0f2909936ed292e511cc2c2cf4cb20dd59890f42487eed8b75e374d43ecab8`.
- Scheduler inspection directly located the fixed-batch barrier at `pkg/engine/scheduler/scheduler.go:223-321`.
- SQLite inspection confirmed globally keyed operation IDs, arbitrary inline input JSON, inline artifact BLOBs, transactional leases, and atomic completion/emission.
- Modern local modules demonstrated that immediate callbacks, hidden typed handles, pure compile terminals, TypeScript parity, and xgoja providers are already established project patterns.

### What didn't work

- Initial resumed commands were run from the workspace parent rather than the scraper repository. The Git command failed exactly with:

  `fatal: not a git repository (or any of the parent directories): .git`

- Initial file reads also omitted the `scraper/` prefix and failed with errors such as:

  `ENOENT: no such file or directory, access '/home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/design-doc/01-durable-dataflow-workflow-v3-and-modern-scripting-architecture.md'`

- The correction was to run repository commands with `cd scraper && ...` and use `scraper/...` paths from the workspace root for file tools.

### What I learned

- The existing durability core is worth preserving: lease acquisition and completion already have strong transactional boundaries.
- The principal storage defect is an API/type defect, not a SQLite tuning defect. Arbitrary `json.RawMessage` made complete-plan and source duplication legal.
- The scheduler behavior is not ordinary polling latency; `WaitGroup.Wait()` creates a strict sibling barrier.
- Queue `MaxInFlight` is useful but is not a complete resource model for independent generation, embedding, CPU, and publication capacities.
- Private Goja symbols are useful within one module, but cross-module RAG/workflow composition needs a portable versioned task descriptor boundary rather than shared symbol identity.

### What was tricky to build

- The design has to preserve scraper as generic infrastructure while still being concrete enough to support the RAG TTC workload. The solution was to define generic task/reference/catalog contracts and place corpus materialization/provider semantics in RAG adapters.
- Safe authoring and privileged execution are easy to blur because current scripts receive one powerful mutable `ctx`. The solution was a module matrix: pure `workflow`, opt-in `workflow/submit`, lease-scoped `workflow/task`, and explicit `workflow/operator`.
- A fluent map callback looks as if it could become durable JavaScript. The grammar probe and target design instead invoke it once with a symbolic item handle and persist only normalized expression IR.

### What warrants a second pair of eyes

- Review whether the proposed package boundaries avoid dependency cycles and keep scraper free of RAG/researchctl imports.
- Review cross-module `WorkflowTask<I,O>` descriptor interchange; runtime validation must not rely on TypeScript brands.
- Verify that every source-bearing value is excluded from node/event/report schemas, not merely omitted in examples.
- Check whether existing public `pkg/workflow` additions such as external artifact/projection hooks should be reused directly or moved behind v3 contracts.

### What should be done in the future

- Keep the probes as regression inputs and convert their desired v3 properties into native Go/Goja golden tests.
- Add synthetic storage-amplification and scheduler-timeline experiments before implementation.

### Code review instructions

- Start with `reference/02-source-catalogue-and-evidence-map.md`.
- Re-run:
  - `python3 scripts/02-inventory-current-scripting.py --repo /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper --out scripts/output/current-scripting-inventory.json`
  - `node scripts/03-workflow-dsl-grammar-probe.mjs > scripts/output/workflow-dsl-grammar-probe.json`
- Compare the JSON outputs and inspect the scheduler's `RunOnce`, SQLite migrations, `StepContext.Emit`, JS executor context, and submission verb context.

### Technical details

- Source fetch script: `scripts/01-fetch-research-sources.sh`.
- Inventory schema: `scraper-scripting-inventory/v1`.
- Probe plan schema: `scraper-workflow-plan/v3`.
- Evidence path: `reference/02-source-catalogue-and-evidence-map.md`.
- The probe is deliberately not production code; canonical IR, validation, compilation, and digesting belong in Go.

## Step 2: Write the workflow-v3 and modern scripting architecture

The primary design was expanded from a placeholder into an intern-oriented implementation guide. It starts from current transactional strengths, explains the measured storage and scheduler failures, then defines a versioned target across representations, persistence, continuous dispatch, attempts, resources, projections, reductions, budgets, scripting, xgoja packaging, security, migration, tests, and acceptance criteria.

The document makes a deliberate separation between pure workflow authoring and runtime authority. It also specifies how researchctl and RAG can both `require("workflow")` without turning either the script or research ledger into a scheduler or giving them ambient scraper-store access.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Synthesize the repository evidence and experiments into the complete architecture/design/implementation guide requested for SCRAPER-WORKFLOW-V3.

**Inferred user intent:** Give an intern enough exact contracts, ordering constraints, file guidance, pseudocode, test criteria, and architectural rationale to implement workflow v3 safely.

### What I did

- Replaced the placeholder `design-doc/01-durable-dataflow-workflow-v3-and-modern-scripting-architecture.md` with the full design.
- Documented current code evidence with file and line anchors.
- Defined immutable builder → IR → compiled plan → run-state layers.
- Proposed compact `ArtifactRef`, `ChunkRef`, `BatchRef`, task descriptor, failure, attempt, resource, budget, and schema contracts.
- Proposed composite run/node identities and v3 logical SQLite tables.
- Specified lease, success, failure, cancellation, resource, and budget transaction boundaries.
- Designed a completion-driven continuous dispatcher with independent resource classes and fairness.
- Defined lazy deterministic map expansion and bounded reduction trees.
- Added full `require("workflow")` JavaScript/TypeScript API sketches and callback semantics.
- Defined pure/privileged module separation and researchctl/RAG/scraper-site integration paths.
- Added xgoja/v2 packaging and generated-host guidance.
- Added phased implementation, tests, privacy scans, scale benchmark, exact-profile preflight, acceptance criteria, eight decision records, alternatives, risks, and open questions.

### Why

- A narrow scheduler patch would leave the source-bearing representation, missing attempts, raw scripting API, and publication/failure risks intact.
- Implementers need explicit transaction and identity invariants; prose such as "make it durable" is insufficient.
- The new DSL needs to follow established local patterns and be independently importable, not become another hand-assembled scraper context.

### What worked

- The final design ties each major recommendation to current code, TTC evidence, a local reusable pattern, or an executable probe.
- It preserves the strong existing lease/completion semantics while replacing global IDs, arbitrary node JSON, mutable result history, and cycle barriers.
- It clearly answers the cross-host scripting requirement: `workflow` is a pure selected module; submission/task/operator capabilities are separate and host-service gated.
- The implementation phases put compact source-free persistence first, making the privacy repair an explicit prerequisite for any fresh real-provider TTC run.
- Acceptance criteria are measurable, including canary scans, row limits, SQLite/WAL bounds, timeline behavior, cardinality, ordering, restart, and reopen checks.

### What didn't work

- N/A during document generation. The implementation has not begun, so none of the proposed APIs or thresholds should be reported as validated production behavior.

### What I learned

- A resource class should be the initial scheduling abstraction; an unrestricted multidimensional resource solver would overcomplicate the first implementation.
- Progress events and progress truth should be separate: events stream changes, while snapshots are authoritative store-derived projections.
- Requested versus effective scheduling policy must both be recorded. Silent host clamping would make TTC throughput and cost evidence ambiguous.
- A compatibility wrapper around raw `ctx.emit` would preserve precisely the unsafe escape hatch v3 is intended to remove.

### What was tricky to build

- The design needed enough schema detail to guide migrations without pretending table names are already final. It therefore specifies logical tables, keys, foreign-key relationships, and transaction invariants while leaving naming review open.
- Cross-repository TypeScript generic types can create a false sense of runtime safety. The design explicitly uses TypeScript brands only for author ergonomics and requires strict descriptor decoding plus task-catalog validation in Go.
- Continuous dispatch has two layers of capacity: local process semaphores and database-scoped grants. The guide makes the store transaction authoritative so multiple processes cannot oversubscribe a resource.
- Cancellation interacts with publication and stale workers. The proposed cancellation epoch is captured in the lease and checked again at commit, preventing late authoritative publication.

### What warrants a second pair of eyes

- Validate the provisional v3 SQLite schema and whether event/attempt volume needs partitioning or retention from the first release.
- Validate canonical JSON rules and choose an implementation before any digest becomes public.
- Review the proposed 64 KiB row and 512 MiB scale limits against representative non-RAG site workloads.
- Review budget semantics for provider requests that time out locally but may have consumed remote tokens.
- Review whether TypeScript declarations should be fully generated or combine descriptor generation with reviewed generic raw DTS.
- Confirm the RAG workflow-adapter ownership and dependency direction before adding cross-repository imports.

### What should be done in the future

- Run docmgr validation and address vocabulary/frontmatter issues.
- Relate focused source files and ticket artifacts.
- Check all ticket tasks and update the changelog after validation.
- Commit and push the documentation bundle.
- Dry-run and upload the design and diary to reMarkable.
- Begin Phase 0/1 only after the design review, starting with compact references and privacy acceptance tests.

### Code review instructions

- Read the design in order through Executive summary, Evidence, Target architecture, Durable store, Dispatcher, DSL, Capability separation, Implementation plan, and Acceptance criteria.
- Cross-check cited code:
  - `pkg/engine/scheduler/scheduler.go:206-321`
  - `pkg/engine/store/sqlite/migrations/001_engine_core.sql`
  - `pkg/engine/store/sqlite/migrations/002_engine_runtime.sql`
  - `pkg/engine/store/sqlite/lease_store.go`
  - `pkg/engine/store/sqlite/result_store.go`
  - `pkg/workflow/package.go`
  - `pkg/workflow/context.go`
  - `pkg/js/runtime/executor.go`
  - `pkg/sites/submitverbs/runtime.go`
- Review the module capability matrix and ensure `require("workflow")` itself has no execution/store authority.
- Validate proposed test cases against every acceptance criterion before implementation is approved.

### Technical details

- Primary design: `design-doc/01-durable-dataflow-workflow-v3-and-modern-scripting-architecture.md`.
- Design size after synthesis: approximately 86 KiB.
- Core pipeline: JavaScript intent → Go normalized IR → host-policy compilation → immutable plan → durable compact run.
- Required implementation validation mode: `GOWORK=off` for scraper and affected consumers.
- Real-provider TTC remains blocked until compact source-free persistence, typed malformed-output retries, and continuous scheduling pass preflight.

## Step 3: Validate and publish the design bundle

The completed ticket documents were related to their concrete source and experiment files, validated through docmgr, and rendered as one reMarkable PDF. The upload contains the primary architecture, evidence catalogue, and chronological diary with a level-two table of contents.

Validation completed before publication. The remaining ticket task was closed only after the upload command returned an explicit success path.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Finish the ticket deliverable by validating its documentation and publishing the review bundle.

**Inferred user intent:** Make the workflow-v3 design easy to review offline and leave the ticket in a clean, continuation-ready state.

### What I did

- Related the primary design to seven focused implementation/reference files with `docmgr doc relate`.
- Related the evidence catalogue and diary to their ticket scripts and documents.
- Checked tasks `dex6`, `1ebk`, and `hdre` after their evidence was present.
- Updated the ticket changelog for source preservation, architecture mapping, and the design guide.
- Ran frontmatter validation for all three ticket documents.
- Ran `docmgr doctor --ticket SCRAPER-WORKFLOW-V3 --stale-after 30`.
- Ran a required dry-run of the reMarkable bundle upload.
- Uploaded `SCRAPER WORKFLOW V3 Architecture.pdf` to `/ai/2026/07/21/SCRAPER-WORKFLOW-V3`.

### Why

- File relations make design claims discoverable from the code that shaped them.
- Doctor/frontmatter validation prevents malformed ticket metadata from becoming the long-term record.
- A dry-run catches bundle path and rendering mistakes before remote publication.

### What worked

- All three frontmatter validations returned `Frontmatter OK`.
- Docmgr doctor reported `✅ All checks passed`.
- JSON outputs remained valid and `git diff --check` found no whitespace errors.
- The dry-run listed the expected three Markdown documents and destination.
- The actual command returned:

  `OK: uploaded SCRAPER WORKFLOW V3 Architecture.pdf -> /ai/2026/07/21/SCRAPER-WORKFLOW-V3`

### What didn't work

- The first staged `git diff --cached --check` failed because terminal-formatted output captured by `xgoja help` contained trailing padding. The first reported errors included:

  `ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/sources/10-xgoja-provider-runtime-config-and-host-services.txt:412: trailing whitespace.`

  `ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/sources/10-xgoja-provider-runtime-config-and-host-services.txt:616: new blank line at EOF.`

- I added deterministic trailing-padding/final-newline normalization to `scripts/01-fetch-research-sources.sh`, normalized the existing ticket text artifacts, restaged them, and reran `git diff --cached --check`; the second run passed.
- No authentication, Pandoc, XeLaTeX, filename, or upload failure occurred.

### What I learned

- Terminal-oriented help output must be normalized before it becomes a tracked research artifact; otherwise a reproducible fetch script can recreate whitespace failures.
- The approximately 1,900-line primary design renders successfully as part of the standard bundle workflow.
- The focused three-document bundle is preferable to embedding all ten source extracts; the catalogue keeps the PDF navigable while the repository preserves full source material.

### What was tricky to build

- The primary design has many nested code blocks, diagrams, and TypeScript/SQL examples, which can expose Markdown-to-PDF rendering problems. The dry-run verified assembly metadata, and the actual renderer/upload completed without an alias or nested-fence error.
- Doc relations needed to remain focused despite many evidence files. The design relates seven decisive implementation files, while the evidence catalogue relates the reproducible scripts and inventories the wider source set.

### What warrants a second pair of eyes

- Review the rendered PDF's diagrams, long code lines, and table widths on the device when convenient.
- Confirm that future implementation commits update design decision statuses rather than treating every proposed detail as already accepted.

### What should be done in the future

- Begin Phase 0 only after architecture review.
- Keep the TTC execution blocked until Phase 1 compact/privacy criteria and the later scheduler/retry preflight criteria pass.

### Code review instructions

- Run `docmgr doctor --ticket SCRAPER-WORKFLOW-V3 --stale-after 30` from the scraper repository.
- Inspect `docmgr task list --ticket SCRAPER-WORKFLOW-V3` and the ticket changelog.
- Review the remote bundle at `/ai/2026/07/21/SCRAPER-WORKFLOW-V3/SCRAPER WORKFLOW V3 Architecture.pdf`.

### Technical details

- Dry-run command used `remarquee upload bundle ... --dry-run --non-interactive --toc-depth 2`.
- Upload command used the same three inputs without `--dry-run`.
- No routine post-upload listing was performed because the upload returned explicit success.

## Step 4: Add the JavaScript workflow cookbook and execution atlas

A new long-form cookbook now pressure-tests workflow v3 against fifteen workflows outside the RAG domain. The examples cover linear transforms, bounded web snapshots, partner API synchronization, ETL joins and quality gates, media conversion, map/reduce analytics, security policy, image classification, notifications, verified backups, inventory repair, build/release pipelines, human approvals, probe matrices, and document conversion.

Each example follows authored JavaScript through normalized IR, compiled jobs, durable node materialization, leases, attempts, runner execution, artifact references, reduction, and final projections. A deep website-snapshot atlas traces one workflow across every layer, while a terminology section distinguishes definitions, plans, jobs, nodes, tasks, runners, and attempts.

### Prompt Context

**User prompt (verbatim):** "make a document with a whole set of example JS scripts for different workflows (not just RAG) and explain how each will be transformed ande xecuted and ultimately mapped to jobs and how these jobs get executed. Store in the ticket as well and upload to remarkable"

**Assistant interpretation:** Add a comprehensive ticket document containing varied workflow-v3 JavaScript examples and explain their complete compilation and durable execution lifecycle, then publish it to reMarkable.

**Inferred user intent:** Make the proposed DSL concrete enough to teach, review, and pressure-test against real orchestration patterns rather than designing only around the RAG workload.

**Commit (code/docs):** a17ec75 — "docs: add workflow v3 JavaScript cookbook"

### What I did

- Added `reference/03-workflow-v3-javascript-cookbook-and-execution-atlas.md`.
- Added fifteen standalone workflow examples and seventeen JavaScript code fences in total.
- Added common terminology, a universal transformation pipeline, task-module/capability models, job/node/attempt mapping, task granularity guidance, anti-patterns, DSL pressure-test findings, test matrices, and review checklists.
- Added `scripts/04-check-cookbook-js.py` to extract every JavaScript fence and invoke `node --check`.
- Stored deterministic validation output at `scripts/output/workflow-cookbook-js-check.json`.
- Updated the ticket index to link the cookbook and describe the ticket artifacts.
- Related the cookbook to the architecture, scheduler/workflow baselines, grammar probe, and checker.
- Validated frontmatter and ran docmgr doctor.
- Dry-ran and uploaded a separate reMarkable PDF.

### Why

- A generic workflow engine should explain web, API, ETL, media, analytics, ML, security, operational, and human-in-the-loop workloads without importing their semantics into scraper core.
- Concrete examples reveal missing DSL contracts earlier than abstract type sketches.
- Explaining job-to-node-to-attempt expansion prevents implementers from treating a compiled job as one goroutine or one database row.

### What worked

- All 17 JavaScript fences passed Node syntax validation.
- The cookbook contains 15 domain-diverse workflow definitions and a layer-by-layer website-snapshot execution atlas.
- The examples exposed actionable DSL decisions around gates, finite resource routing, optional dependencies, typed ports, set manifests, schedules, expansion backpressure, and task-catalog metadata.
- Frontmatter validation returned `Frontmatter OK`.
- `docmgr doctor --ticket SCRAPER-WORKFLOW-V3 --stale-after 30` returned `✅ All checks passed`.
- The reMarkable command returned:

  `OK: uploaded SCRAPER WORKFLOW V3 JavaScript Cookbook.pdf -> /ai/2026/07/21/SCRAPER-WORKFLOW-V3`

### What didn't work

- N/A. The syntax checker, Markdown validation, dry-run, PDF rendering, and upload all succeeded on their first run.

### What I learned

- The core task/map/reduce model covers most workflows when domain modules return typed task descriptors and set manifests are first-class references.
- Human approval cannot be modeled as a sleeping task; it justifies a first-class durable gate that releases its lease while waiting.
- Item-dependent resource routing must be finite and compiler-visible. Arbitrary JavaScript routing at dispatch time would undermine capability validation.
- Side-effecting workflows make node identity and idempotency keys as important as retry policy.

### What was tricky to build

- The examples needed to be varied without claiming nonexistent modules are already implemented. The cookbook labels every domain module and script as a proposed target contract and distinguishes syntax validation from runtime validation.
- Some workflows naturally want dynamic routing or waiting. Rather than hiding those gaps, the examples identify design pressure and constrain safe implementation options.
- Long examples can obscure the execution model, so each one includes a graph, mapping table, walkthrough, or correctness notes, and the website example receives a complete multi-stage atlas.

### What warrants a second pair of eyes

- Review `p.gate` and finite resource-routing proposals before freezing the builder API.
- Review whether joins and multi-output map items need additional first-class types or are sufficiently represented by task descriptors and typed ports.
- Confirm that every example's side-effecting task has adequate idempotency semantics.
- Verify the proposed task granularity against measured store/dispatcher overhead once implementation exists.

### What should be done in the future

- Convert all fifteen examples into Goja fixtures with normalized-IR and compiled-plan golden files.
- Add fake-runner execution tests for the job/node/attempt mappings and privacy assertions.
- Keep the cookbook synchronized with any accepted changes to the DSL and task catalog contracts.

### Code review instructions

- Start with the cookbook's terminology and universal pipeline sections.
- Review Examples 2, 6, 9, and 13 for map streaming, reduction, finite resource routing, and lease-free gates.
- Run:

  `ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/scripts/04-check-cookbook-js.py --doc ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/reference/03-workflow-v3-javascript-cookbook-and-execution-atlas.md --out /tmp/workflow-cookbook-js-check.json`

- Compare the test matrix to the primary architecture acceptance criteria.

### Technical details

- Cookbook size: approximately 1,680 lines and 65 KiB.
- Validation result: 17/17 JavaScript fences parsed successfully.
- reMarkable path: `/ai/2026/07/21/SCRAPER-WORKFLOW-V3/SCRAPER WORKFLOW V3 JavaScript Cookbook.pdf`.
- The examples remain target API documentation until native modules and compiler/runtime tests exist.

## Step 5: Design reproducible domain-authored JavaScript task bundles

The workflow design now supports domain developers supplying custom JavaScript task catalogs and implementations without adding every task to scraper core. The proposal keeps the ergonomic idea that a worker “loads JavaScript and registers tasks,” but confines registration to an explicit, atomic worker boot/reload phase rather than ambient process-global `require()` side effects.

A task bundle pins schemas, entrypoints, dependencies, capabilities, provenance, and digest. Workers verify the exact artifact, evaluate its catalog in a registration-only runtime, seal a registry generation, advertise exact implementation identities, and run each leased attempt in a fresh capability-limited Goja runtime. Plans bind task kind/version plus bundle digest, entrypoint, and ABI, so a worker cannot silently substitute changed code.

### Prompt Context

**User prompt (verbatim):** "one thing, I see that there are many \"custom\" tasks in data.tasks, and I think it would be useful for a certain domain developer to be able to submit custom JS tasks so that we can have flexibility to process those. Can we make some setup where like, loading some JS registers a set of tasks, and that way each worker could populate and provide these tasks in a reproducible and robust manner?"

**Assistant interpretation:** Extend workflow v3 with a safe, reproducible mechanism for domain-owned JavaScript bundles to register multiple task implementations on workers and make those exact implementations available to compiled workflows.

**Inferred user intent:** Avoid hard-coding every cookbook domain operation in scraper while preserving durable execution, schemas, worker consistency, reproducibility, and operational safety.

**Commit (code/docs):** 0e2e1d7 — "docs: design reproducible JavaScript task bundles"

### What I did

- Added `design-doc/02-reproducible-javascript-task-bundles-and-worker-registries.md`.
- Documented source/built bundle layouts, strict manifests, explicit JavaScript catalog registration, task implementation context, authoring descriptor modules, deterministic builds, dependency locks, signatures, and provenance.
- Designed immutable worker registry generations, exact capability advertisements, compiler binding, lease matching, fresh attempt runtimes, atomic reload, version coexistence, quarantine, and process isolation for untrusted bundles.
- Added seven focused decision records, phased implementation, store sketches, tests, and acceptance criteria.
- Updated the primary architecture, cookbook, ticket index, and evidence catalogue to treat custom domain tasks as bundle-provided rather than scraper built-ins.
- Added `scripts/05-js-task-bundle-registration-probe.mjs` and a two-task fixture bundle.
- Generated `scripts/output/js-task-bundle-registration-probe.json`.

### Why

- A generic engine cannot require a scraper release for every customer-specific transform or domain rule.
- Persisting arbitrary callbacks in workflow plans would sacrifice exact code identity, schema validation, capability planning, and secure worker matching.
- Task kind/version alone is insufficient for restart and rolling deployment because different code can claim the same key.

### What worked

- The fixture loads a JavaScript catalog that registers `acme.customer.normalize@v1` and `acme.customer.validate@v1`.
- It validates that each bundle-local entrypoint export exists before sealing the registry.
- Repeated probe execution produced the same bundle and registry digests.
- Exact task/version/bundle/entrypoint/ABI requirements matched successfully.
- Wrong bundle, task version, and entrypoint requirements were all rejected.
- Frontmatter validation and docmgr doctor passed.

### What didn't work

- N/A. The design validation, JavaScript syntax checks, deterministic probe comparison, and exact-matching assertions passed on the first run.

### What I learned

- The existing per-operation Goja runtime is a useful execution baseline, but current script-path metadata must become an immutable bundle/entrypoint identity.
- The current string-keyed runner registry needs task version, implementation digest, registry generation, and worker advertisement before it can support reproducible custom tasks.
- Registration, authoring, and execution require distinct runtime module sets.
- Goja module allowlists are not sufficient for hostile code; mutually untrusted bundles need process/container isolation.

### What was tricky to build

- “Loading JS registers tasks” is convenient but ambient registration is order-dependent and unsafe. The design preserves the phrase only for an explicit loader transaction: evaluate catalog → validate candidate → self-test → seal → advertise.
- Rolling upgrades must not change code beneath active runs. Plans therefore pin exact implementation digests, and workers may advertise old/new registry generations while runs drain.
- Domain flexibility could reopen arbitrary payloads. The host still validates input/config/output schemas and compact references outside JavaScript.

### What warrants a second pair of eyes

- Review whether task contract versions and bundle implementation versions are modeled at the correct separate layers.
- Review trust/signature policy and the boundary between trusted in-process and sandboxed subprocess workers.
- Benchmark fresh Goja runtime creation before considering immutable compiled-program caches or runtime pooling.
- Review worker capability storage/indexing for multi-process SQLite and a future PostgreSQL store.

### What should be done in the future

- Implement Phase 1 catalog/bundle types and upgrade the v3 runner registry identity.
- Build one production-quality pure transform bundle before adding network/database host capabilities.
- Add a sandbox worker class before accepting untrusted bundle publishers.
- Convert cookbook placeholders to namespaced bundle-provided authoring modules.

### Code review instructions

- Start with the executive answer and security statement in the new design.
- Review catalog/entrypoint, worker boot, compiler binding, registry generation, and lease execution sections in order.
- Run:

  `node ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/scripts/05-js-task-bundle-registration-probe.mjs`

- Compare `pkg/engine/runner/runner.go` and `pkg/js/runtime/executor.go` to the documented v3 gaps.

### Technical details

- Probe bundle digest: `sha256:b349df43ce6f637dde813d2f611dff9fcef43063840709333c7d513c7688eb28`.
- Probe registry digest: `sha256:cc2cff85c297bd3f90b3d8ac87ab45ad0534b298caef24a299dacba5f957aedd`.
- Custom task bundles remain proposed architecture; no production runner/store behavior changed.

## Step 6: Align the workflow cookbook with custom task bundles

The cookbook now consistently treats every hypothetical domain module as a descriptor-only authoring module generated from a domain-owned JavaScript task bundle. All fifteen workflow scripts import namespaced bundle modules such as `acme-data-tasks`, `acme-web-tasks`, `studio-media-tasks`, and `company-security-tasks` rather than implying that generic `data`, `web`, or `security` modules are built into scraper.

A new cookbook chapter follows one custom data task from bundle source through catalog registration, execution entrypoint, generated authoring module, worker bundle lock, deterministic build, compiler binding, registry advertisement, exact dispatch matching, lease-scoped execution, and host-side output validation.

### Prompt Context

**User prompt (verbatim):** "can you update the cookbook accordingly, in case you haven't already?"

**Assistant interpretation:** Update the existing JavaScript examples cookbook comprehensively so it reflects the newly designed custom task-bundle and exact worker registration model.

**Inferred user intent:** Ensure the teaching material no longer suggests that cookbook domain tasks are scraper built-ins and demonstrates how domain developers actually package, register, author with, and execute custom JavaScript tasks.

**Commit (code/docs):** e971ff3 — "docs: align cookbook with JavaScript task bundles"

### What I did

- Replaced generic cookbook imports with namespaced bundle authoring modules across all fifteen examples.
- Reworked the common task-module table into a bundle/import/namespace/worker-authority matrix.
- Added catalog, execution entrypoint, descriptor-only authoring module, bundle lock, and complete binding-chain examples.
- Added a scraper-core versus domain-bundle responsibility table.
- Updated normalized IR and compiled-plan examples to use namespaced task keys and exact bundle/entrypoint/ABI implementation identity.
- Expanded pressure-test findings, test matrix, fixture layout, author checklist, and operator checklist for bundle registration, sealed worker registries, wrong-digest rejection, and rolling versions.
- Re-ran the cookbook JavaScript fence validator.
- Dry-ran and uploaded a new, non-destructive reMarkable PDF name rather than overwriting the previously published cookbook.

### Why

- The earlier cookbook had only a short note about bundles while its scripts still imported generic names such as `require("data")`.
- Examples must model the intended extension mechanism or implementers may accidentally recreate a hard-coded domain-module monolith.
- Separate authoring and worker runtime phases need to be visible in the primary teaching document.

### What worked

- All fifteen workflow scripts now use namespaced descriptor-only bundle imports.
- All 20 JavaScript fences passed syntax validation.
- The deep website atlas pins `acme.web.fetch@v1` to an explicit JavaScript bundle digest, entrypoint, and task ABI.
- Frontmatter validation and docmgr doctor passed.
- The updated PDF uploaded successfully:

  `OK: uploaded SCRAPER WORKFLOW V3 JavaScript Cookbook Task Bundles.pdf -> /ai/2026/07/21/SCRAPER-WORKFLOW-V3`

### What didn't work

- N/A. Documentation edits, JavaScript syntax validation, dry-run, rendering, and upload succeeded.

### What I learned

- A single short caveat was insufficient; module names in executable examples strongly imply ownership and packaging.
- The cookbook needs two fixture/runtime layers for every domain task: descriptor-only authoring modules and exact worker execution bundles.
- Bundle/registry identity belongs in fixture goldens alongside workflow IR and compiled plans.

### What was tricky to build

- The examples need readable local variables such as `data` and `web` while exposing real ownership. The scripts keep those concise variables but import explicit namespaced bundles.
- Adding task-bundle code blocks changed validation line mappings, so the deterministic syntax output had to be regenerated after docmgr updated frontmatter relations.
- The existing reMarkable PDF may contain annotations, so the updated cookbook used a new document name instead of `--force` overwrite.

### What warrants a second pair of eyes

- Review the example bundle/module names and namespace conventions before they become public API guidance.
- Verify whether the generated authoring module should always come from the execution bundle or may be published as a separately signed artifact sharing one catalog digest.
- Review the fixture layout for enough coverage of old/new implementation generations and capability-blocked nodes.

### What should be done in the future

- Turn the cookbook bundle snippets into actual Goja/xgoja integration fixtures.
- Generate real descriptor modules and DTS from one catalog source.
- Add sealed-registry and exact-dispatch goldens to every example family.

### Code review instructions

- Review the cookbook sections `Custom task bundles used below` and `How a custom bundle supplies a cookbook task` first.
- Confirm every workflow example imports an explicitly namespaced `*-tasks` module.
- Run the cookbook checker and confirm `20/20` blocks pass.
- Compare the website snapshot's IR/compiled-plan walkthrough to the focused task-bundle design.

### Technical details

- Updated cookbook size: approximately 1,900 lines and 77 KiB.
- Validation output: `scripts/output/workflow-cookbook-js-check.json`.
- Updated reMarkable path: `/ai/2026/07/21/SCRAPER-WORKFLOW-V3/SCRAPER WORKFLOW V3 JavaScript Cookbook Task Bundles.pdf`.

## Step 7: Add a companion task bundle to every cookbook workflow

Every one of the fifteen workflows now imports one dedicated companion bundle and links directly to its complete catalog later in the cookbook. The companion catalogs enumerate all task factories used by their workflow, namespaced task identities, exact bundle-local entrypoints, input/output contracts, primary resources, requested modules/capabilities, side-effect/idempotency semantics, and generated authoring export groups.

The cookbook also defines a shared build-only bundle helper, required source layout, and execution-entrypoint obligations. This keeps the document readable while making each workflow self-contained at the workflow/task contract level.

### Prompt Context

**User prompt (verbatim):** "for each example, can you also add the repsective task bundle, to have it kind of \"self-contained\""

**Assistant interpretation:** Pair every cookbook workflow with a concrete custom JavaScript task-bundle catalog so readers can see both authored workflow intent and all domain task implementations/contracts it requires.

**Inferred user intent:** Make each example independently understandable and prevent task factories from appearing as unexplained external magic.

**Commit (code/docs):** e216481 — "docs: add companion bundles to cookbook examples"

### What I did

- Changed all fifteen workflow imports to unique `cookbook-*-tasks` companion bundles.
- Added an explicit `Companion task bundle` subsection to every example.
- Added a shared build-only catalog helper and required bundle source layout.
- Added fifteen full companion `catalog.js` definitions.
- Enumerated every task key, entrypoint export, output port, resource, module/capability request, and relevant side-effect/idempotency policy.
- Updated the first bundle's detailed catalog, descriptor-only authoring module, worker lock, and binding chain to match its workflow exactly.
- Updated the website atlas and fixture layout to use cookbook bundle namespaces.
- Added consistency validation proving 15 workflows, 15 bundle headings, and exact import-to-bundle-name equality.
- Regenerated JavaScript syntax-check output.

### Why

- A workflow script alone does not explain where domain task descriptors and worker implementations come from.
- One companion bundle per example makes ownership, registration, execution, resource, and capability boundaries visible without assuming scraper built-ins.
- Exact catalog entries provide a direct implementation checklist for future fixture bundles.

### What worked

- The consistency check reported `examples=15 companionBundles=15 importsMatch=true`.
- All 36 JavaScript fences passed Node syntax validation.
- The existing registration probe remained deterministic.
- Frontmatter validation and docmgr doctor passed.
- The cookbook grew to approximately 2,350 lines and 99 KiB while retaining a navigable two-level structure.

### What didn't work

- N/A. Catalog insertion, namespace alignment, consistency assertions, syntax checks, and document validation passed.

### What I learned

- Pairing examples one-to-one with bundles is clearer than sharing broad hypothetical modules across unrelated examples.
- A build-only catalog helper can remove repetitive registration boilerplate without entering authoring or execution runtimes.
- “Self-contained” should mean complete task identities/contracts and entrypoint obligations, while privileged capability implementations remain host-owned and policy-constrained.

### What was tricky to build

- Catalogs had to match every method called by each workflow, including grouped exports such as `{web, data}` and `{ops, notify}`.
- Task bundle code must remain concise enough for a cookbook while still documenting exact resources, outputs, capabilities, and side effects.
- The approval example required distinguishing a bundle-provided gate initializer from engine-owned lease-free waiting state.

### What warrants a second pair of eyes

- Verify every catalog output port against the corresponding workflow's `output(...)` references.
- Review whether the shared helper's `authoring` metadata is the preferred source for generated module groups/factory names.
- Review default semantics versus explicit side-effect declarations before converting catalogs into executable fixtures.

### What should be done in the future

- Materialize each appendix catalog as a real fixture directory with schemas and `execution/tasks.cjs`.
- Generate descriptor modules and DTS from the catalogs.
- Add runtime tests that build all fifteen bundles and compile their paired workflows.

### Code review instructions

- Confirm each example's `Companion task bundle` link resolves to the matching appendix heading.
- Compare each workflow's `tasks.*` calls to its catalog factories.
- Run the syntax checker and the 15-example/15-bundle import consistency assertion.

### Technical details

- Cookbook validation: 36/36 JavaScript blocks passed.
- Companion coverage: 15/15 examples.
- Bundle catalogs are proposed source contracts; host capabilities and task runtime modules remain future implementation.

## Step 8: Publish the self-contained cookbook as v2

The self-contained cookbook was rendered and uploaded under an explicit v2 name. The previous cookbook PDFs were not overwritten, preserving any existing annotations.

### Prompt Context

**User prompt (verbatim):** "upload as v2 to remarkable when done"

**Assistant interpretation:** Publish the completed companion-bundle cookbook as a separately named v2 PDF on reMarkable.

**Inferred user intent:** Make the revised self-contained edition easy to distinguish from previous cookbook uploads while preserving older copies.

### What I did

- Ran the required dry-run with the final cookbook.
- Rendered a level-two table-of-contents bundle.
- Uploaded `SCRAPER WORKFLOW V3 JavaScript Cookbook V2.pdf` to the existing ticket directory.

### Why

- A new versioned name avoids destructive `--force` replacement and protects annotations on earlier PDFs.

### What worked

- Dry-run identified the expected source, output name, and destination.
- Upload returned:

  `OK: uploaded SCRAPER WORKFLOW V3 JavaScript Cookbook V2.pdf -> /ai/2026/07/21/SCRAPER-WORKFLOW-V3`

### What didn't work

- N/A. Rendering and upload succeeded.

### What I learned

- The expanded 99 KiB cookbook, including thirty-six JavaScript blocks and fifteen bundle catalogs, renders successfully with the standard layout.

### What was tricky to build

- Versioning the remote document rather than overwriting was important because reMarkable annotations are attached to the existing document identity.

### What warrants a second pair of eyes

- Inspect the long bundle-catalog code blocks and table widths on-device when convenient.

### What should be done in the future

- Publish a v3 only after executable fixture bundles or accepted DSL contract changes materially alter the cookbook.

### Code review instructions

- Review `/ai/2026/07/21/SCRAPER-WORKFLOW-V3/SCRAPER WORKFLOW V3 JavaScript Cookbook V2.pdf`.

### Technical details

- Remote directory: `/ai/2026/07/21/SCRAPER-WORKFLOW-V3`.
- Remote document: `SCRAPER WORKFLOW V3 JavaScript Cookbook V2.pdf`.

## Step 9: Show readable trusted JavaScript task implementations

The cookbook now shows the complete illustrative `execution/tasks.cjs` source
for every companion bundle rather than stopping at catalog declarations. All
JavaScript fences—not only the new implementations—are deterministically
formatted to an 80-column ceiling so code remains readable in the reMarkable
PDF layout.

The implementations use current go-go-goja host modules where appropriate:
HTTP clients, preconfigured databases, attempt-scoped filesystems, portable
paths, hashing, YAML, monotonic timing, and allowlisted process execution.
Catalog module profiles and implementation imports are checked for exact
agreement.

### Prompt Context

**User prompt (verbatim):** "can you format the example bundle so they are 80 char wide, i can't actually see their implementation in JS. Also use some of the more \"advanced\" goja modules since we are now in \"safe code\" territory, for example to do HTTP requests or quiery the db or load from the fs."

**Assistant interpretation:** Make all cookbook JavaScript readable at 80
columns and include visible implementations for every task bundle, using
trusted execution-phase go-go-goja modules for realistic I/O.

**Inferred user intent:** See how domain task behavior is actually implemented,
not just registered, and understand how trusted worker JavaScript performs
HTTP, database, filesystem, tool, and structured-data operations.

**Commit (code/docs):** 89d38be — "docs: show guarded JavaScript task implementations"

### What I did

- Added complete illustrative `execution/tasks.cjs` blocks for all fifteen
  bundles and every catalog task.
- Used current `fetch`, `database`, `fs`, `exec`, `crypto`, `path`, `yaml`, and
  `time` APIs through worker-profile aliases.
- Kept model and signer behavior behind dedicated host modules.
- Added explicit trusted-code security boundaries for credentials, DSNs,
  filesystems, process allowlists, and process/container isolation.
- Added current go-go-goja module API provenance to the evidence catalogue.
- Added `scripts/06-format-cookbook-js.py` to format and enforce an 80-column
  JavaScript-fence ceiling.
- Added `scripts/07-check-cookbook-bundles.py` to prove catalog tasks match
  implementation handlers and catalog module profiles match implementation
  imports.
- Reformatted all fifty-two JavaScript fences.
- Regenerated the syntax-check artifact.
- Published the result non-destructively as cookbook V3 on reMarkable.

### Why

- Catalogs prove identity and registration but do not teach task-domain
  implementation.
- Wide code fences are clipped or unreadable in a portrait PDF.
- Trusted lease-scoped worker code can use explicitly selected host modules;
  pretending all domain behavior is a Go helper would hide the intended
  JavaScript extensibility model.

### What worked

- The width checker reported `formatted=52 maxWidth=80 violations=0`.
- All 52/52 JavaScript blocks passed syntax validation.
- Bundle validation reported
  `bundles=15 catalogsMatchHandlers=true modulesMatchImports=true`.
- Frontmatter validation and docmgr doctor passed.
- reMarkable upload and remote listing both showed
  `SCRAPER WORKFLOW V3 JavaScript Cookbook V3`.

### What didn't work

- The first bulk catalog edit was rejected because
  `namespace: "cookbook.linear"` appeared in both the detailed opening example
  and appendix. The edit was retried with bundle-name context so each target
  was unique.
- A second edit was rejected because two identical `modules` lines were not
  unique. The retry included adjacent task identity context.

### What I learned

- Bundle-level module profiles are the correct concise representation when one
  CommonJS entrypoint file imports modules at top level: every exported handler
  needs those imports available during module evaluation.
- Current xgoja host providers support aliases, so examples can distinguish
  `db:warehouse`, `db:restore-sandbox`, `fetch:partner`, and `fetch:probe`
  without inventing different JavaScript APIs.
- `database` calls are synchronous while `fetch` and async `fs` calls return
  promises; the examples now reflect those actual contracts.

### What was tricky to build

- Each catalog had to enumerate exactly the module aliases imported by its
  implementation. The new checker prevents either undeclared imports or stale
  unused declarations.
- Tool execution had to remain visibly safe: command names are finite and
  profile-allowlisted, while worker process/container isolation constrains the
  host filesystem and resources.
- Dynamic SQL table identifiers cannot use placeholders, so the backup example
  includes a strict identifier validator before interpolation.
- The approval initializer had to remain a short JavaScript execution while
  the engine owns long-lived lease-free waiting.

### What warrants a second pair of eyes

- Review whether module grants should remain bundle-level or be split into one
  entrypoint module per least-privilege task before production implementation.
- Review the proposed `ctx.outputs.createGate(...)` and lease-local resolved-ref
  `path` representation before freezing the task ABI.
- Inspect code-block readability on the physical reMarkable device.
- Treat `exec:*` as trusted-code convenience, not security isolation; validate
  the subprocess/container design before enabling third-party publishers.

### What should be done in the future

- Convert these illustrative sources into actual immutable fixture bundles.
- Run them through xgoja-generated worker profiles with preconfigured aliases.
- Add runtime integration tests for HTTP, database, filesystem, and allowlisted
  command examples using local fake services and temporary stores.

### Code review instructions

- Start at `Companion task bundles for all examples` in the cookbook.
- For each bundle, compare catalog task names to the following
  `execution/tasks.cjs` handler names.
- Run scripts 04, 06, and 07; expect 52/52 syntax, zero width violations, and
  exact catalog/handler/module agreement.
- Review the module contracts cited in the evidence catalogue against the
  current go-go-goja source.
- Open the V3 PDF and inspect long implementations such as partner sync,
  database backup, release build, and deployment.

### Technical details

- Commit: `89d38be`.
- JavaScript fences: 52.
- Width ceiling: 80 characters.
- Companion implementations: 15 bundles, all catalog tasks covered.
- Remote document:
  `/ai/2026/07/21/SCRAPER-WORKFLOW-V3/SCRAPER WORKFLOW V3 JavaScript Cookbook V3.pdf`.

## Step 10: Start the implementation tranche with frozen contracts

The architecture now has an executable delivery order that reaches real work
immediately: a durable linear file transform first, then the minimal
`require("workflow")` DSL over the same canonical Go plan. The complete design
still leads through HTTP, work-conserving dispatch, database side effects,
maps, reductions, rolling registries, budgets, gates, isolation, and finally
the TTC preflight.

The first production package freezes canonical workflow IR, compact artifact
references, exact implementation identity, strict decoding, deterministic
digests, task catalogs, graph validation, and compilation. This is deliberately
below the JavaScript surface so both direct Go plans and JavaScript-authored
plans must converge on one representation.

### Prompt Context

**User prompt (verbatim):** "User task:
ok, so update the design doc for the ticket to incorporate the new JS stuff and this sequence of work (in tasks), then get to work and implement it and get to the minimal workflow DSL running

Turn the user task into exactly one durable pi-codex-goal objective, then call the goal creation tool with that objective.

This prompt invocation is an explicit user request to set a new goal. When the goal creation tool exposes `replace_existing`, pass `replace_existing: true` so an existing active, paused, or budget-limited goal is replaced instead of requiring `/goal clear` first.

Do not set a token budget limit unless the user explicitly provides a budget/limit in the task. If no explicit budget is provided, omit the token budget field entirely.

The goal must be a completion contract, not a task summary. Preserve the user's full intent. Do not weaken broad acceptance criteria such as \"all\", \"any\", \"complete\", \"no tech debt\", \"do it right\", \"fully\", or \"hard acceptance criteria\".

The goal must require:

1. Outcome
   - State what must be true when complete.
   - Preserve the full requested end state.
   - Do not narrow scope after the fact unless the original user task explicitly defined that scope.

2. Verification evidence
   - Name the concrete evidence required before completion.
   - Include relevant tests, lint, type checks, builds, smoke checks, diffs, docs, generated outputs, rendered UI inspection, or artifact checks when applicable.
   - If the repo has an existing local CI/validation command, require it unless clearly irrelevant.

3. Constraints
   - Preserve existing behavior unless the task explicitly changes it.
   - Do not discard user changes.
   - Do not leave unapproved shortcuts, compatibility shims, TODO placeholders, dead code, duplicated logic, hidden assumptions, or undocumented behavior changes.

4. Iteration policy
   - After each attempt, inspect evidence, update the plan, and keep taking the next low-risk useful step.
   - Do not stop at a plan when implementation or verification remains.
   - If validation fails, triage and fix the cause rather than reporting partial completion.

5. Completion audit
   - Before marking the goal complete, map every explicit requirement in the goal to fresh evidence from files, commands, diffs, tests, screenshots, artifacts, or logs.
   - The goal is not complete if any requirement is unverified, narrowed, deferred, or only probably satisfied.
   - Phrases like \"for the scope this is complete\", \"good enough\", \"out of scope\", or \"remaining tech debt\" are not valid completion evidence unless the original user task explicitly allowed that limitation.

6. Blocked stop condition
   - If completion is impossible with current access, tools, budget, or missing decisions, stop without marking complete.
   - Report attempted paths, evidence gathered, exact blockers, remaining unmet requirements, and what input would unblock progress.

Use concise imperative language in the goal. If the task is blank or only whitespace, infer the goal based on the conversation context or ask the user to clarify."

**Assistant interpretation:** Replace the previous long-running goal with a
strict completion contract, revise ticket design/tasks for trusted JavaScript
execution and vertical slices, and continue implementation until the minimal
DSL executes a real durable file workflow with all foundational invariants.

**Inferred user intent:** Move from architecture into evidence-backed working
software without postponing real workload tests until the entire v3 platform
is built.

**Commit (code/docs):** ff286a1 — "workflowv3: freeze core IR and vertical slices"

### What I did

- Replaced the previous active goal with the requested durable completion
  contract and no token budget.
- Added seven implementation tasks to the ticket.
- Updated the primary design with trusted execution-phase Goja module aliases
  and twelve workload-driven vertical slices.
- Updated the bundle design to permit exact profile-selected host modules while
  retaining phase separation and isolation requirements.
- Added `pkg/workflowv3` with strict core representations and validation.
- Added exact task kind/version/bundle/entrypoint/ABI catalog identities.
- Added deterministic IR/catalog/plan digests.
- Added compiler validation for port schemas, dependencies, cycles, duplicate
  names, output refs, and unsupported task identities.
- Added focused unit tests and ran them with workspace isolation.

### Why

- Persisted run identity and reference shapes cannot be safely retrofitted after
  real work begins.
- A canonical Go model lets direct Go construction and the upcoming JavaScript
  DSL share one validation and compilation path.
- Workload slices make every infrastructure addition answer to a concrete
  restartable workflow.

### What worked

- `GOWORK=off go test ./pkg/workflowv3 -count=1` passed.
- Deterministic compilation produced exact pinned implementation identities.
- Negative tests rejected schema drift, cycles, unknown JSON fields, and an
  unsupported task ABI.
- Commit `ff286a1` captured the first focused implementation interval.

### What didn't work

- N/A in the core implementation. The first compile/test pass succeeded.

### What I learned

- Bundle-level module profiles fit a CommonJS implementation file because all
  top-level imports must be available before any exported handler can run.
- The minimal compiler can stay small if task catalogs own all port schemas and
  exact implementation identity.

### What was tricky to build

- The plan digest cannot include itself; compilation hashes a copy with an empty
  digest field and then sets the resulting digest.
- Schema checks must validate both the resolved source schema and the schema
  carried by the symbolic reference, otherwise a stale or forged handle can
  hide drift.
- Dependency validation and binding validation are separate: dataflow refs do
  not implicitly replace explicit readiness dependencies in the durable plan.

### What warrants a second pair of eyes

- Review whether explicit `DependsOn` should be normalized from data bindings
  during compilation or remain mandatory author intent.
- Review canonical JSON assumptions before adding values with durations,
  timestamps, or semantically unordered arrays.

### What should be done in the future

- Implement the minimal Goja authoring module over these exact Go types.
- Add golden IR and plan files before changing the model further.
- Implement sealed bundle registration, task execution, and compact SQLite
  persistence next.

### Code review instructions

- Start with `pkg/workflowv3/types.go`, then `catalog.go` and `compiler.go`.
- Compare the implementation sequence in the primary design to ticket tasks.
- Run `GOWORK=off go test ./pkg/workflowv3 -count=1`.

### Technical details

- New schemas: `scraper-workflow-ir/v3` and `scraper-workflow-plan/v3`.
- Task ABI: `scraper-js-task/v1`.
- Completed ticket tasks: `foy2`, `0jm0`.

## Step 11: Make diary maintenance part of the completion contract

Diary maintenance is now an explicit goal invariant rather than optional end
bookkeeping. Work will continue in focused code commits followed immediately by
diary, changelog, task, and file-relation updates carrying the code commit hash.

### Prompt Context

**User prompt (verbatim):** "keep a detailed diary as you work, and commit at regular intervals."

**Assistant interpretation:** Record implementation decisions, commands,
failures, fixes, and review evidence continuously, and split work into regular
focused commits.

**Inferred user intent:** Preserve a trustworthy continuation and review trail
through a long implementation rather than reconstructing an incomplete story
at the end.

### What I did

- Committed the first architecture/core implementation interval as `ff286a1`.
- Checked the first two implementation tasks and updated the ticket changelog.
- Began this detailed diary entry immediately after the code commit.

### Why

- The runtime/store/Goja work crosses several sharp trust and durability
  boundaries; contemporaneous evidence is necessary for review and recovery.

### What worked

- The focused commit contains only architecture, task planning, core model,
  compiler, and tests.

### What didn't work

- N/A.

### What I learned

- Recording task completion at each commit keeps docmgr state aligned with the
  actual implementation rather than the intended plan.

### What was tricky to build

- The goal was already active, so the explicit follow-up required replacing it
  again while preserving every original acceptance condition and adding diary
  requirements rather than narrowing the objective.

### What warrants a second pair of eyes

- Confirm future commit intervals remain focused enough to review independently.

### What should be done in the future

- Continue the code-commit then diary-commit loop for authoring, registry,
  persistence, and end-to-end execution.

### Code review instructions

- Compare commit `ff286a1` with this step and the ticket changelog entry.

### Technical details

- Goal was updated after the follow-up prompt below.

## Step 12: Update the durable goal with strict diary evidence

The active goal now requires detailed append-only diary updates after every
meaningful attempt and at regular focused commit intervals. Completion cannot
be claimed without diary entries and linked evidence for all requirements.

### Prompt Context

**User prompt (verbatim):** "update the goal accordingly to make sure we always properly write things up in our diary"

**Assistant interpretation:** Amend the active completion contract so ongoing
diary quality and timing are mandatory acceptance requirements.

**Inferred user intent:** Prevent implementation momentum or future context
compaction from causing diary updates to be skipped.

### What I did

- Replaced the active goal with the same implementation contract plus explicit
  continuous diary, changelog, task, relation, command, error, decision, risk,
  validation, and commit-hash requirements.

### Why

- A conversation-level reminder is weaker than a durable completion condition.

### What worked

- Goal `a7904295-8b3c-4259-851f-e6a69a9522cb` is active without a token budget.

### What didn't work

- N/A.

### What I learned

- Goal replacement is appropriate here because the user explicitly requested a
  stronger contract, not a second concurrent goal.

### What was tricky to build

- The updated objective had to retain the complete implementation and
  verification scope while adding diary obligations without accidentally
  resetting the work outcome.

### What warrants a second pair of eyes

- Verify every subsequent completion audit includes diary evidence as well as
  code and test evidence.

### What should be done in the future

- N/A; this requirement remains active throughout the goal.

### Code review instructions

- Inspect the active goal and compare it with Steps 10–12.

### Technical details

- Active goal ID: `a7904295-8b3c-4259-851f-e6a69a9522cb`.

## Step 13: Run the minimal `require("workflow")` authoring DSL

A real CommonJS script can now import `workflow` and a generated-style
`cookbook-linear-transform-tasks` descriptor module, define inputs and two
ordered tasks, publish a named output, validate the graph, and compile it into
the canonical exact-identity Go plan. The authoring runtime contains no store,
filesystem, network, task-execution, or submission authority.

The builder uses Go-owned opaque object identity rather than trusting mutable
JavaScript properties. Direct Go construction and JavaScript authoring produce
the same `WorkflowIR`, and deterministic JSON goldens freeze both normalized IR
and compiled plan contracts.

### Prompt Context

**User prompt (verbatim):** (same as Step 10)

**Assistant interpretation:** Implement the smallest safe authoring surface
that can describe and compile the real linear-transform plan.

**Inferred user intent:** Exercise the public JavaScript API early while keeping
canonical validation and compilation authoritative in Go.

**Commit (code):** b4d54ba — "workflowv3: add minimal Goja authoring DSL"

### What I did

- Added `pkg/gojamodules/workflow`.
- Implemented `workflow.define`, `input`, descriptor-backed `task`, `after`,
  `output`, `validate`, `digest`, `toIR`, and `compile`.
- Added descriptor-module generation from explicit factory-to-task mappings.
- Kept handles opaque by associating Goja object identity with Go refs,
  invocations, jobs, workflows, and plans.
- Rejected unknown task options and descriptors created outside
  `workflow.define`.
- Added a minimal semantic TypeScript declaration without `any`.
- Added a real CommonJS linear-transform script.
- Added normalized IR and exact compiled-plan goldens.
- Added direct Go versus JavaScript IR equality testing.

### Why

- The DSL must prove ergonomics without introducing a second model or compiler.
- Opaque handles prevent scripts from forging a task descriptor or cross-plan
  output by mutating public object fields.
- Goldens expose representation drift before durable persisted plans depend on
  it.

### What worked

- `GOWORK=off go test ./pkg/workflowv3 ./pkg/gojamodules/workflow -count=1`
  passed.
- The plan golden pins kind, version, bundle digest, entrypoint, ABI, schemas,
  modules, IR digest, catalog digest, and plan digest.
- Negative authoring tests reject undeclared task input fields.
- TypeScript tests verify the minimal surface and reject semantic `any`.

### What didn't work

- N/A. The first authoring implementation and golden generation passed.

### What I learned

- A native module can provide private typed handles without exposing Go fields:
  map each returned `*goja.Object` to its Go representation and ignore mutable
  JavaScript properties.
- CommonJS `module.exports = workflow.compile(...)` can retain plan identity by
  mapping the exact exported object to the Go plan.
- Immediate configurator callbacks make explicit dependencies available in IR
  without persisting functions.

### What was tricky to build

- Descriptor factories execute inside the active plan build and must resolve
  options through the same runtime's opaque ref map. This simultaneously
  rejects plain-object forgeries and cross-runtime handles.
- The script wrapper must pass both `module` and the original `exports`; the
  final result must be read back from `module.exports` because scripts replace
  it during compilation.
- Go structs cannot be exported directly if lower-camel JSON shape matters, so
  terminal inspection values are converted through canonical JSON to plain
  JavaScript objects.

### What warrants a second pair of eyes

- Review whether the first public API should expose `formatDiagnostics` now or
  wait until diagnostic codes are introduced.
- Review whether dataflow bindings should automatically normalize explicit
  `after` dependencies in a later compiler revision.
- Review TypeScript generation strategy before expanding beyond the minimal
  hand-declared surface.

### What should be done in the future

- Bind the catalog to a real computed bundle digest instead of the authoring
  test fixture digest.
- Execute the compiled plan through the sealed registry and SQLite store.
- Add cross-plan and post-build mutation rejection tests as the API expands.

### Code review instructions

- Start with `pkg/gojamodules/workflow/authoring_test.go` and its JavaScript
  fixture, then inspect object-identity maps in `authoring.go`.
- Diff the IR and plan goldens against `pkg/workflowv3/types.go`.
- Run the focused `GOWORK=off` test command above.

### Technical details

- JavaScript fixture: `testdata/linear-transform.js`.
- Plan digest: `sha256:5ad8d9c58ea4f769653b15d069f415a0f014f7a43e55b35100c3c86adf3b0305`.
- Completed ticket task: `as4j`.

## Step 14: Execute and reopen a real durable JavaScript file workflow

The JavaScript-authored linear plan now runs end to end through immutable task
bundle bytes, an exact sealed registry generation, fresh Goja runtimes, the
current guarded `fs` module, an external content-addressed artifact store, and
new compact SQLite v3 tables. The integration test transforms and validates
12,000 real JSONL rows, deliberately restarts after the first node, reopens the
final output, and scans SQLite main/WAL/SHM bytes for private source canaries.

The store creates one immutable attempt row per lease, fences completion by
lease token and cancellation epoch, marks expired attempts `lease_lost`, and
starts a new monotonically numbered attempt. Plans pinned to different bundle
bytes are not leased to the worker even when task kind and version match.

### Prompt Context

**User prompt (verbatim):** (same as Step 10)

**Assistant interpretation:** Connect the minimal DSL to real immutable bundle
execution and durable compact persistence without waiting for later maps,
budgets, or work-conserving dispatch.

**Inferred user intent:** Reach a meaningful restartable workload quickly while
proving the foundational privacy and reproducibility invariants now.

**Commit (code):** 756dbf5 — "workflowv3: execute durable JavaScript file workflow"

### What I did

- Added deterministic bundle manifests and source-derived bundle digests.
- Added exact sealed registry generations and full-identity resolution.
- Added a shared embedded linear-transform workflow/task-bundle fixture.
- Added a content-addressed file artifact store with size and digest checks.
- Added fresh per-attempt Goja runtimes exposing only `workflow/task` and a
  read-only attempt-scoped `fs:input` module.
- Added lease-local input materialization and host-side output-port/schema
  validation.
- Added compact SQLite v3 run, input, node, dependency, attempt, output, and
  redacted event tables.
- Added transactional leasing, expiry reclamation, completion/failure,
  cancellation fencing, snapshots, and reopen.
- Added an engine that submits compiled plans and executes ready nodes through
  the exact sealed registry.
- Added exact bundle/entrypoint/ABI mismatch tests, stale completion tests,
  lease-loss tests, artifact-integrity tests, and real task-runtime tests.
- Added a 12,000-row restart/reopen/privacy/storage-amplification integration
  test driven by the JavaScript-authored plan.

### Why

- Source bytes belong in verified external artifacts, not durable control rows.
- Exact implementation identity must participate in lease admission, not only
  runner lookup after a worker has already claimed work.
- Fresh runtimes make mutable CommonJS globals attempt-local and permit module
  authority to match the compiled task profile.
- A restart between the two real tasks is stronger evidence than a unit-only
  compiler demonstration.

### What worked

- Focused tests passed for core, authoring, fixtures, task runtime, and SQLite.
- The real task source imports `workflow/task` and current go-go-goja
  `fs:input`; both async file reads completed through runtime promise handling.
- A global load counter in the task bundle proves a new runtime/module cache is
  created for each attempt.
- The integration test reopened after node one and again after final completion.
- The final output reports 12,000 validated rows.
- The private source canary and secret token remain present in the external
  source artifact but absent from SQLite main, WAL, SHM, and final output.
- SQLite control-plane bytes remain below half the source payload size.
- A worker registry built from different task bytes receives no lease for the
  pinned plan.

### What didn't work

- The first SQLite test build failed with:

  `pkg/workflowv3sqlite/store.go:7:2: "encoding/json" imported and not used`

  Command:

  `GOWORK=off go test ./pkg/workflowv3sqlite -count=1`

  I removed the stale import and reran the test.

- The next stale-cancellation assertion failed:

  `--- FAIL: TestStoreRejectsStaleCompletionAfterCancel`

  `Error: Should be true`

  The cancellation transaction correctly cleared `lease_token`, but
  `checkFence` scanned nullable lease columns into plain strings/integers and
  returned a SQL scan error instead of `ErrStaleCompletion`. I changed the
  fence read to `sql.NullString`/`sql.NullInt64`, explicitly classified missing
  lease values as stale, and reran the suite successfully.

### What I learned

- Exact registry matching naturally belongs inside `LeaseNext`; scanning ready
  candidates against the sealed registry leaves incompatible work pending for
  a worker that actually advertises it.
- Bundle digest construction is simplest and deterministic when the manifest is
  canonicalized with sorted file path/digest/size entries rather than raw map
  iteration.
- Async current-module `fs.readFile` works cleanly when the task runtime uses the
  existing owned-runtime promise waiter.
- External artifacts may be written before node completion because they are
  content-addressed and unreferenced until the fenced completion transaction.

### What was tricky to build

- A bundle digest cannot be embedded in its own manifest task identities.
  Manifest tasks omit it; `TaskSpecs()` injects the computed digest into exact
  implementation identities after hashing.
- Input refs must become usable paths without granting access to the whole
  artifact store. Each attempt gets a temporary read-only filesystem containing
  only its bound artifacts.
- Completion updates output refs, attempt status, node status, run terminal
  status, and redacted event evidence in one transaction.
- SQLite canceled/expired leases deliberately null fence columns, so all fence
  readers must treat nullability as a security state rather than a scan detail.
- Common fixtures initially duplicated bundle/workflow source across authoring
  and runtime tests. I moved both into `pkg/testfixtures/workflowv3linear` and
  switched authoring tests to an external package to avoid an import cycle.

### What warrants a second pair of eyes

- Review the SQLite transaction and ready-node query for multi-process leasing
  under heavier contention; the DSN currently uses immediate transactions and
  a busy timeout.
- Review whether the initial failure path should support bounded retry now or
  remain terminal until typed retry policy enters the plan.
- Review temporary input copies versus hardlinks or read-only bind mounts for
  larger production artifacts.
- Review whether event payload output digest maps meet the desired metadata
  allowlist.

### What should be done in the future

- Add concurrent lease-race coverage and run store tests under `-race`.
- Preserve typed `task.failure(...)` codes through the Goja boundary rather than
  classifying all execution exceptions as internal.
- Run repository-wide tests/lint and complete final docs/API validation.
- Add a small runnable command or example if operators need a manual smoke path
  outside integration tests.

### Code review instructions

- Start with `pkg/workflowv3runtime/engine_integration_test.go` for the complete
  path, then follow into `engine.go`, `task_runner.go`, and SQLite `store.go`.
- Inspect `schema.sql` for reference-only columns and composite run/node keys.
- Run:

  `GOWORK=off go test ./pkg/workflowv3 ./pkg/gojamodules/workflow ./pkg/testfixtures/workflowv3linear ./pkg/workflowv3runtime ./pkg/workflowv3sqlite -count=1`

### Technical details

- Code commit: `756dbf5`.
- Real input rows: 12,000.
- Runtime modules exposed per attempt: `workflow/task`, `fs:input`.
- Completed ticket tasks: `9pis`, `eyzy`, `8cs8`.

## Step 15: Harden typed failures, module admission, and lease races

The first vertical slice now preserves stable typed JavaScript task failures
through asynchronous Promise rejection and stores only a bounded host-generated
message. Task runtimes expose exactly the module aliases declared by the pinned
task spec; a linked provider or another bundle's privileges do not make an
ambient module available.

Concurrent SQLite lease contenders now have an explicit single-winner test,
and store/runtime suites pass Go race detection. Bundle internals are private
and returned manifests/files are cloned so registry generations cannot be
mutated after sealing.

### Prompt Context

**User prompt (verbatim):** (same as Step 10)

**Assistant interpretation:** Close correctness and trust-boundary gaps exposed
by the first end-to-end implementation before repository-wide validation.

**Inferred user intent:** Ensure the minimal working slice already embodies the
final typed-failure, exact-capability, immutability, and concurrency rules.

**Commit (code):** f25f558 — "workflowv3: harden failure and lease boundaries"

### What I did

- Added a stable failure-class vocabulary and uppercase code validation.
- Changed `task.failure(...)` to create a marked JavaScript failure object.
- Added task-specific Promise waiting that recovers typed failures before Goja
  exports away their identity.
- Persisted task class/code/retryability with a bounded host-generated message,
  never the arbitrary JavaScript message.
- Added durable typed-failure integration coverage.
- Added wrong-output-schema and unsupported-module-profile tests.
- Changed task runtime construction to register only declared module aliases.
- Made bundle manifest/files private and exposed defensive clones.
- Added an eight-contender SQLite lease test proving one winner.
- Ran store/runtime tests under `-race`.
- Added explicit source/persistence byte ratio logging to the privacy test.

### Why

- Async JavaScript exceptions cross both Goja and Promise boundaries; losing
  class/code there would force brittle string matching.
- Bundle catalog module lists are security inputs and must control actual
  runtime composition.
- A sealed registry is not immutable if callers can mutate public manifest
  slices or file maps through retained references.
- Lease correctness must survive concurrent transactions, not only sequential
  tests.

### What worked

- Focused core/runtime/store tests passed after the fixes.
- `GOWORK=off go test -race ./pkg/workflowv3sqlite ./pkg/workflowv3runtime -count=1`
  passed.
- Typed duplicate-ID failure persists as class `validation`, code
  `CUSTOMER_DUPLICATE_ID`, retryable `false`, with host message
  `task reported CUSTOMER_DUPLICATE_ID`.
- A runtime requesting undeclared `db:ambient` is rejected before execution.
- Eight lease contenders produce exactly one non-nil lease.

### What didn't work

- Initial typed-failure tests showed that async rejections lost the Go-backed
  failure object:

  `expected: "validation"`

  `actual  : "internal"`

  and:

  `Error: Should be true`

  The generic promise waiter exported the rejection before classification. I
  replaced it in the task runtime with a waiter that inspects the rejected
  Goja value on the runtime owner thread.

- The initial wrong-output test returned:

  `await task: promise rejected: map[]`

  The generic waiter also erased useful `TypeError` text. The task-specific
  waiter now retains rejected-value string diagnostics while typed failures use
  their marked object.

- The first custom waiter then panicked on a rejected object without the marker:

  `runtimeowner workflowv3.task.promise: runtime call panicked: runtime error: invalid memory address or nil pointer dereference`

  `object.Get("__workflowTaskFailure")` can return nil. I added an explicit nil
  guard before `ToBoolean()` and reran both focused and race suites.

### What I learned

- Go-backed object identity is not reliable after an async JavaScript throw;
  an explicit private marker on a plain rejection object survives Promise state
  transitions predictably.
- Promise rejection inspection must happen on the owned runtime thread, just
  like all other Goja value access.
- Exact module policy needs enforcement both in the compiled task spec and in
  runtime factory composition.

### What was tricky to build

- The task-specific waiter must distinguish pending, fulfilled, generic
  rejected, and typed rejected states while honoring context cancellation and
  never touching Goja values outside `runtime.Owner.Call`.
- Durable messages must help operators without allowing arbitrary task/source
  text into SQLite; class and code carry stable meaning while the stored message
  is generated by Go.
- Defensive bundle immutability required changing test helpers that previously
  read public manifest fields.

### What warrants a second pair of eyes

- Review whether the failure class vocabulary is complete enough for the next
  HTTP slice without being overly broad.
- Review the five-millisecond Promise polling strategy inherited from the
  existing runtime before high-throughput execution.
- Stress multi-process rather than only multi-goroutine SQLite lease races in a
  later dispatcher slice.

### What should be done in the future

- Run repository-wide tests, lint, build, generated checks, and JavaScript
  syntax checks.
- Document the implemented package/API status in the design and public docs.
- Perform the explicit completion audit against every active-goal requirement.

### Code review instructions

- Review `taskFailureFromValue`, `waitForTaskPromise`, and Engine failure
  persistence together.
- Review runtime module construction against each task's `Modules` field.
- Run focused tests and the race command above.

### Technical details

- Code commit: `f25f558`.
- Supported first-slice task module profile: `fs:input`.
- Concurrent lease contenders tested: 8.
