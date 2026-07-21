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

## Step 16: Validate, document, and audit the minimal runtime

The implementation now has public help documentation, exact DTS evidence,
generated logcopter package files, and current implementation status in both
design documents. Repository-wide Go tests, the project validation target,
Go lint under the required workspace isolation, race suites, JavaScript syntax,
web tests/build, generated binary build, help rendering, frontmatter, and
docmgr validation have been exercised.

A validation pass exposed two repository-environment issues rather than hiding
them: Go lint must run with `GOWORK=off`, and the web build lacked direct
TypeScript dependencies that its source already imports. The dependencies were
added explicitly and `make validate` now passes end to end.

### Prompt Context

**User prompt (verbatim):** (same as Step 10)

**Assistant interpretation:** Finish the implementation with fresh repository
validation, public documentation, generated artifacts, and a requirement-by-
requirement evidence audit.

**Inferred user intent:** Leave a minimal workflow DSL that is demonstrably
running and maintainable, not merely passing isolated package tests.

**Commits:**

- dfbd061 — "web: declare direct TypeScript build dependencies"
- 67d4776 — "workflowv3: validate and document minimal runtime"

### What I did

- Added exact `workflow.d.ts` golden comparison.
- Added package documentation for core, authoring, runtime, and SQLite.
- Added a public Glazed help topic for the implemented minimal runtime.
- Updated both design documents with implementation status and source mapping.
- Ran `go generate`, producing required logcopter files for every new package.
- Fixed all new Go lint findings rather than suppressing them.
- Added the two missing direct web build dependencies already imported by
  existing source: `@codemirror/state` and `@storybook/react`.
- Ran repository-wide Go tests, race tests, Go lint, Go/web builds, web unit
  tests, JavaScript syntax checks, help rendering, and the verbose privacy test.
- Checked the final implementation/validation task and updated changelog and
  file relations.

### Why

- Generated sources and public docs are part of the repository contract.
- Exact DTS comparison prevents runtime/type surface drift.
- A successful focused suite is insufficient if existing packages or the
  actual release build regress.
- pnpm's strict dependency model requires source imports to be direct
  dependencies rather than accidental transitive availability.

### What worked

- `GOWORK=off go test ./... -count=1` passed.
- `GOWORK=off .bin/golangci-lint run ./cmd/... ./pkg/...` passed with zero
  issues after fixes.
- `GOWORK=off go test -race ./pkg/workflowv3sqlite
  ./pkg/workflowv3runtime -count=1` passed.
- `make validate` passed, including Go tests/generation/build, web unit tests,
  TypeScript, and Vite production build.
- `node --check` passed for both workflow and task-bundle fixture sources.
- `scraper help scraper-workflow-v3-minimal-runtime` rendered the embedded page.
- Verbose privacy evidence reported:

  `source=1656000 persistedSQLite=73728 ratio=0.0445`

- The Vite build completed with only its existing large-chunk advisory.

### What didn't work

- Running `make lint` without workspace isolation failed in type checking due to
  the parent workspace's local go-go-goja checkout and module-version mismatch:

  `undefined: goja.IsNumber`

  `undefined: goja.IsBigInt`

  The design and goal already require `GOWORK=off`; rerunning the pinned linter
  as

  `GOWORK=off .bin/golangci-lint run ./cmd/... ./pkg/...`

  reached the actual new-code findings.

- The first isolated lint pass reported 16 visible issues: unchecked cleanup,
  close, rollback, and row-close errors plus an incomplete Promise-state switch.
  I replaced deferred cleanup calls with explicit checked/ignored closures,
  checked test cleanup with `require.NoError`, and added the pending Promise
  case. A final isolated lint pass reported `0 issues`.

- The first web validation attempt failed before tests because `node_modules`
  was absent:

  `sh: 1: vitest: not found`

  I ran `pnpm install --frozen-lockfile`; unit tests then passed.

- The first web build then failed because existing source directly imports
  packages absent from `package.json`:

  `Cannot find module '@storybook/react'`

  `Cannot find module '@codemirror/state'`

  I added both as direct dependencies and reran web tests/build successfully.

- A separate `pnpm lint` still reports broad pre-existing frontend lint debt:
  old Storybook renderer imports, React effect/style rules, unused variables,
  and generated `storybook-static` files. This command is not part of the
  repository `make lint` or `make validate` targets, and none of the reported
  source files were changed by workflow v3. The relevant repository targets,
  TypeScript build, Vite build, and unit tests pass. The exact output is
  preserved in this diary rather than misreported as a workflow-v3 failure.

### What I learned

- The repository's documented `GOWORK=off` requirement applies to lint as well
  as tests and builds, even though the current Makefile `lint` recipe omits it.
- `go generate ./...` correctly discovers new packages and emits logcopter
  files, so generated-file cleanliness must be checked before final diff review.
- The help-system smoke test is a useful validation of YAML frontmatter,
  embedding, and actual CLI discoverability in one command.

### What was tricky to build

- Errcheck requires a deliberate policy for deferred cleanup. Production paths
  use explicit best-effort cleanup closures where no return path exists, while
  tests register cleanup functions that assert close success.
- The inner Promise-state switch and outer waiter switch have different jobs;
  both still need exhaustive enum handling.
- Validation changed the web dependency lock despite no frontend feature work.
  Keeping that in a separate focused commit makes the reason reviewable.

### What warrants a second pair of eyes

- Review whether the Makefile `lint` target should permanently set
  `GOWORK=off`; this session used the correct explicit command without changing
  unrelated build policy.
- Review the direct `@storybook/react` dependency versus migrating all stories
  to `@storybook/react-vite` in a dedicated frontend cleanup.
- Inspect the 2 MB Vite chunk warning separately from workflow-v3 work.

### What should be done in the future

- Start Slice 3 with an allowlisted real HTTP snapshot after this minimal slice
  is reviewed.
- Address existing frontend lint debt in a dedicated ticket rather than mixing
  it into durable workflow implementation.

### Code review instructions

- Run `make validate` and the isolated Go lint/race commands above.
- Run the verbose privacy test and compare its byte ratio.
- Open the new public help topic through `dist/scraper help`.
- Review commits `dfbd061` and `67d4776` separately.

### Technical details

- Full validation target: passed.
- Go lint: zero issues with `GOWORK=off`.
- Web unit tests: 4/4 passed.
- Minimal runtime help slug: `scraper-workflow-v3-minimal-runtime`.
- Completed ticket task: `awrp`.

## Step 17: Make the minimal input handle generically typed

The completion audit caught that runtime inputs carried exact schemas but the
TypeScript declaration returned an unparameterized `ValueRef`. The public
minimal DSL now exposes `input<T = unknown>(...) -> ValueRef<T>`, preserving the
author's value type without weakening Go-owned runtime schema validation.

### Prompt Context

**User prompt (verbatim):** (same as Step 10)

**Assistant interpretation:** Verify that “typed input” is true in both runtime
contracts and the published TypeScript surface.

**Inferred user intent:** Ensure the minimal DSL is genuinely usable as a typed
JavaScript/TypeScript authoring API, not merely schema-tagged internally.

**Commit (code):** 3136cd6 — "workflowv3: type minimal DSL inputs"

### What I did

- Changed the DTS `PlanBuilder.input` signature to generic `input<T = unknown>`.
- Updated the exact DTS golden and semantic declaration assertion.
- Reran authoring/runtime tests and focused isolated lint.

### Why

- The active completion contract explicitly requires typed input.
- Schema IDs enforce runtime compatibility, while `ValueRef<T>` preserves
  compile-time author ergonomics; both layers are necessary.

### What worked

- Authoring and end-to-end runtime tests passed.
- Exact DTS comparison passed.
- Focused isolated lint reported zero issues.

### What didn't work

- N/A. This was an audit-found contract improvement, not a failing test.

### What I learned

- A requirement audit should inspect declarations semantically rather than only
  treating the existence of a DTS golden as sufficient evidence.

### What was tricky to build

- The generic type remains phantom authoring information; the schema string is
  still the authoritative runtime contract and must never be inferred from a
  TypeScript-only type.

### What warrants a second pair of eyes

- Review task descriptor generics when the generated domain-module DTS is
  expanded beyond this minimal slice.

### What should be done in the future

- Add typed task input/output descriptor generation in the next DSL expansion.

### Code review instructions

- Compare `TypeScript()` to `testdata/workflow.d.ts` and run the authoring test.

### Technical details

- Generic signature: `input<T = unknown>(..., {schema}): ValueRef<T>`.

## Step 18: Complete the requirement-by-requirement audit

A fresh final audit maps every active-goal requirement to committed code,
deterministic artifacts, commands, and logs. No required implementation item
is deferred: the requested end state was the fully running minimal DSL vertical
slice, while later HTTP/map/budget/gate slices remain the already documented
next architecture sequence.

### Prompt Context

**User prompt (verbatim):** (same as Step 10)

**Assistant interpretation:** Do not mark completion until every explicit
minimal-DSL, durability, privacy, documentation, and validation condition has
fresh evidence.

**Inferred user intent:** Receive a trustworthy implementation handoff whose
completion claim can be independently reproduced.

### Outcome audit

| Requirement | Fresh evidence |
|---|---|
| Design incorporates trusted JavaScript modules | Primary and bundle design docs describe exact `fetch:*`, `db:*`, `fs:*`, `exec:*`, utility aliases, and trust/isolation boundaries |
| Work sequence exists in ticket tasks | Twelve vertical slices are documented; implementation tasks `foy2` through `awrp` are checked |
| Minimal DSL runs | `pkg/testfixtures/workflowv3linear/workflow.js` executes `define`, generic typed input, task, `after`, output, validate, and compile |
| Go owns canonical IR and compilation | `pkg/workflowv3` plus direct-Go/JavaScript equality and exact IR/plan goldens |
| Exact implementation identity | Plan golden and registry tests cover kind, version, bundle digest, entrypoint, ABI, catalog digest, and wrong-identity rejection |
| Sealed registry and fresh runtimes | Registry-generation tests and bundle global-load assertion in both real attempts |
| Compact reference-only persistence | `schema.sql`, artifact store, and byte scan: 1,656,000 source bytes versus 73,728 SQLite bytes |
| Append-only attempts and fencing | Attempt history, expired lease, cancel epoch, stale completion, and eight-contender single-winner tests |
| Restart and reopen | End-to-end test closes/reopens after node one and again after success, preserving root output and plan digest |
| Source/secret privacy | Exact source and token canaries absent from main/WAL/SHM/events/final output but present in external source artifact |
| Typed failures and schema validation | Duplicate-ID failure class/code persistence plus wrong input/output schema tests |
| No silent capability/implementation substitution | Lease admission requires exact registry identity; unsupported `db:ambient` profile is rejected |
| Detailed diary and regular commits | Steps 10–18 and paired focused code/docs commits with hashes |

### Verification audit

- `make validate` passed after the final typed-input change.
- `GOWORK=off .bin/golangci-lint run ./cmd/... ./pkg/...` reported `0 issues`.
- `GOWORK=off go test -race ./pkg/workflowv3sqlite
  ./pkg/workflowv3runtime -count=1` passed.
- Both real JavaScript files pass `node --check`.
- Exact IR, plan, and DTS goldens pass.
- `dist/scraper help scraper-workflow-v3-minimal-runtime` passes.
- `docmgr doctor --ticket SCRAPER-WORKFLOW-V3 --stale-after 30` passes.
- `git diff --check` passes and the working tree was clean before this diary
  update.

### Constraints audit

- Existing v2 workflow/site behavior remains unchanged and the full repository
  Go suite passes.
- No v2 compatibility shim or raw-operation translation was added.
- No unrelated user changes were discarded.
- `rg` found no TODO/FIXME/HACK placeholders in new workflow-v3 packages.
- Bundle/catalog/runtime logic has one shared linear fixture rather than
  duplicated source.
- All behavior changes are documented in design, public help, changelog, tasks,
  and diary.

### What I did

- Repeated all completion-critical commands after the final generic typed-input
  change.
- Rechecked privacy/storage evidence and help rendering.
- Rechecked task completion, clean docmgr validation, commit history, and tree
  status.
- Updated the final changelog audit entry.

### What worked

- Every required command and artifact check passed.
- Privacy ratio remained `0.0445` across the fresh run.
- All seven implementation tasks are checked.

### What didn't work

- N/A in the final rerun. Earlier failures and their fixes are preserved in
  Steps 14–16.

### What I learned

- The strongest completion evidence is the intersection of representation
  goldens, negative trust-boundary tests, crash/reopen behavior, byte-level
  privacy scans, and repository-wide validation; no single category is enough.

### What was tricky to build

- The audit had to distinguish the requested complete minimal vertical slice
  from later architecture slices without weakening any requirement. The
  minimal slice includes all foundational identity, durability, fencing,
  privacy, and fresh-runtime invariants; only new workload capabilities remain
  for subsequent slices.

### What warrants a second pair of eyes

- Transaction ordering in `Store.Complete` and `Store.Fail`.
- Promise rejection classification in `waitForTaskPromise`.
- Module-profile admission and read-only input materialization.
- Plan/registry exact identity and digest normalization.

### What should be done in the future

- Begin Slice 3 only after review: add an allowlisted real HTTP snapshot without
  changing the proven minimal contracts.

### Code review instructions

- Read Steps 10–18 in order.
- Start code review at the end-to-end test and follow each boundary inward.
- Reproduce the commands in the verification audit.

### Technical details

- Final implementation commit range: `ff286a1..3136cd6`.
- Final diary/bookkeeping commits continue through this step.
- Active goal: `a7904295-8b3c-4259-851f-e6a69a9522cb`.

## Step 19: Start Slices 3–5 from the proven minimal runtime

### Prompt Context

**User prompt (verbatim):**

> slices 3-5

**Assistant interpretation:** Implement the next three documented executable
vertical slices completely: a policy-bound real HTTP snapshot, a
completion-driven work-conserving dispatcher with independent resources, and
an idempotent real database synchronization through host-preconfigured handles.

**Inferred user intent:** Continue directly from the completed minimal DSL
without revisiting or weakening its exact identity, compact persistence,
fencing, privacy, fresh-runtime, and reopen contracts.

**Active goal:** `dafed540-d78f-4507-968d-e24ef5fc2a20`.

### What I did

- Inspected the active goal and confirmed the branch is clean at `8dd8a34`.
- Re-read the architecture's Slice 3–5 exit criteria and current workflow-v3
  compiler, registry, runtime, SQLite store, schema, fixtures, and integration
  tests.
- Re-read the installed go-go-goja fetch policy/runtime and preconfigured
  database module APIs rather than inventing incompatible wrappers.
- Added four docmgr tasks:
  - `by9j` — Slice 3 HTTP snapshot;
  - `bjbi` — Slice 4 dispatcher;
  - `7gmp` — Slice 5 database synchronization;
  - `v8h8` — validation, documentation, and completion audit.
- Updated the ticket changelog immediately.

### Why

- The three slices share module-profile and resource metadata, so their durable
  contracts must be designed together even though implementation will proceed
  in focused commits.
- Existing go-go-goja already provides origin policy, response limits,
  cancellation-aware async fetch, abstract transaction wrappers, and explicit
  denial of `configure()` for preconfigured databases. Reusing those trusted
  APIs avoids duplicated host logic.

### What worked

- The completed minimal implementation exposes clean extension seams:
  `BundleTask`, `TaskSpec`, `PlanNode`, `Lease`, `RunTask`, and `LeaseNext`.
- Existing tests already cover exact implementation mismatch, basic module
  rejection, lease races, stale completion, and compact source privacy.

### What didn't work

- N/A. This step was evidence-backed scope setup and architecture inspection.

### What I learned

- Current `Engine.RunUntilIdle` is deliberately serial; Slice 4 needs a new
  dispatcher rather than making `RunOne` internally batch.
- Current retryable task failures terminate the run. Slice 3 requires retry
  policy and ready-time persistence before HTTP transient failures can be
  proven correctly.
- Resource identity is absent from current task/plan/lease/attempt records.
  Slice 4 must add it canonically rather than relying on module aliases.
- Database side effects need a stable host-owned idempotency key derived from
  run/node, not an attempt token, so lease loss cannot duplicate a logical
  write.

### What was tricky to build

- N/A yet. The main design pressure is keeping policy configuration out of the
  canonical plan while still making module/resource identities exact and
  worker-advertised.

### What warrants a second pair of eyes

- Retry-count semantics and whether lease loss consumes the task retry budget.
- Fairness ordering across runs within each resource class.
- SQL policy boundaries for the first write alias and transaction wrapper.

### What should be done next

1. Freeze resource and retry metadata in canonical IR/plan/bundle goldens.
2. Add a runtime profile registry for exact `fetch:public` and `db:sync`
   aliases, with no ambient default module fallback.
3. Build Slice 3 fixture and negative policy/redaction tests.
4. Introduce the store-backed dispatcher/resource admission contract.
5. Build the idempotent database synchronization fixture and crash test.

### Code review instructions

- Compare new metadata against the existing canonical goldens before reviewing
  runtime behavior.
- Verify policy values and secrets never enter bundle manifests or plans.

### Technical details

- Starting commit: `8dd8a34`.
- Goja dependency: `github.com/go-go-golems/go-go-goja v0.8.3`.
- Current runtime aliases: only `workflow/task` and `fs:input`.

## Step 20: Implement the HTTP, dispatcher, and database vertical paths

### Prompt Context

**User prompt (verbatim):** (same as Step 19)

**Assistant interpretation:** Move immediately from contract setup to executable
Slices 3–5 and preserve failure evidence while iterating.

**Inferred user intent:** See useful real HTTP and database work running through
the durable engine, with the scheduler barrier actually removed rather than
merely redesigned.

**Commit:** `b05e5a0` — "workflowv3: implement HTTP dispatch and database slices"

### What I did

- Added canonical task policy to bundle/catalog/plan identity:
  `resourceClass` and typed `retry {maxAttempts, backoffMillis}`.
- Kept completed Slice 1–2 plans compatible through documented canonical
  defaults (`cpu.default`, one attempt), and regenerated the exact plan golden.
- Extended sealed worker registries with explicit module-alias advertisement;
  registry generation now digests both implementations and aliases.
- Added exact node admission checks for implementation, modules, resource, and
  retry policy.
- Added an immutable lease-scoped `TaskModuleRegistry` with factories for:
  - read-only `fs:input`;
  - policy-selected `fetch:*`;
  - Go-preconfigured `db:*`.
- Upgraded go-go-goja from v0.8.3 to v0.10.6 because that is the first available
  tagged dependency in this repository line containing the reviewed fetch
  module.
- Wrapped HTTP redirect following with the same origin policy as the initial
  URL; disabled credential environment/file sources in the public test profile.
- Added durable retry deadlines, redacted failed attempts, resource class facts,
  per-resource fair dispatch counters, database-scoped active-capacity checks,
  and derived queue projections.
- Added `Dispatcher.Run` and deterministic `DispatchOnce`; a completion wakes
  refill without waiting for unrelated attempts.
- Added a lease watcher so durable cancellation/expiry cancels active Goja and
  HTTP contexts.
- Added a real authored HTTP snapshot fixture using `fs:input` and
  `fetch:public`, with typed status/transport failures and bounded cardinality.
- Added a real authored database synchronization fixture using `fs:input` and
  `db:sync`, with script-side `configure()` denial, transactions, stable
  run/node operation keys, and post-commit retry idempotency.
- Added exact IR/plan goldens for HTTP and database workflows.
- Added integration evidence for allowlist denial, redirect denial, response
  limits, cancellation, typed 503 retry, reopen, mixed-resource refill,
  capacity, fairness, blocked reasons, retry backoff across reopen, cardinality,
  failure isolation, and crash-after-side-effect recovery.

### Why

- Resource and retry policy affect lease eligibility and must be canonical plan
  facts, not runtime guesses.
- Module aliases carry authority; registering code must not silently advertise
  host services.
- A stable idempotency key must exclude attempt number so a lease-loss retry
  observes the same logical side effect.
- Fairness must be per resource class; a run's database work must not consume
  its HTTP scheduling share.

### What worked

- The HTTP workflow retries one 503 and succeeds on attempt two; URL query and
  response canaries are absent from SQLite/WAL/SHM.
- Denied origins and denied cross-origin redirects are never contacted; all
  three bounded retry attempts persist only stable failure codes and redacted
  messages.
- A blocked HTTP response is canceled after durable run cancellation.
- The dispatcher test holds one HTTP task and one unrelated slow task active,
  releases only the HTTP slot, and observes the next HTTP task begin before the
  slow task finishes. Peak HTTP concurrency remains exactly one.
- Queue projection reports both active resource classes and the capacity block.
- Store fairness selects run B after one run-A dispatch even though run A has
  another ready node in the same class.
- Retry backoff remains blocked after store close/reopen and leases attempt two
  only after its durable deadline.
- The database task commits 500 customer writes and returns a typed retryable
  post-commit failure, but the test kills the worker path without persisting
  that outcome. After workflow-store restart and lease expiry, attempt one is
  durably `lease_lost`; attempt two succeeds without a second audit or
  operation row.
- The database receipt proves `configureDenied=true`, second-attempt
  `applied=false`, and exact cardinality 500.
- Database privacy/storage evidence:

  `source=499554 persistedSQLite=90112 ratio=0.1804`

- Focused tests, race suites, isolated lint, JavaScript syntax, and deterministic
  goldens pass after fixes.

### What didn't work

- Initial compilation failed because pinned go-go-goja v0.8.3 did not ship the
  reviewed fetch package:

  `no required module provides package github.com/go-go-golems/go-go-goja/modules/fetch`

  I inspected published versions, confirmed v0.10.6 contains `modules/fetch`
  and supports the repository's Go 1.26.5, then upgraded exactly to v0.10.6.

- The first authoring test failed with the expected stale plan golden after
  resource/retry and normalized bundle identity became canonical. I reviewed
  the diff and regenerated it with:

  `UPDATE_GOLDEN=1 GOWORK=off go test ./pkg/gojamodules/workflow`

- The first fairness projection assertion expected one dependency block but
  observed two. Both runs' validation nodes were correctly dependency-blocked;
  I corrected the assertion rather than changing the projection.

- The first combined validation command stopped at:

  `changelog.md:158: new blank line at EOF.`

  I normalized ticket text files before rerunning `git diff --check`.

- The first race-suite cancellation run returned:

  `sql: transaction has already been committed or rolled back`

  instead of `context canceled`. The dispatcher could enter a lease transaction
  concurrently with context cancellation. I now return `ctx.Err()` when a
  dispatch transaction fails after cancellation. The full focused race suite
  then passed.

### What I learned

- The released go-go-goja module surface, not the local checkout, is the actual
  build contract; trusted-module design must verify the tagged dependency.
- Redirect policy must be enforced separately from initial URL policy when an
  injected `http.Client` follows redirects.
- Per-run dispatch counts are insufficient for independent resources; fairness
  accounting must key `(run_id, resource_class)`.
- A task failure can be durably handled while still being returned to
  deterministic `RunOne` callers; the long-lived dispatcher needs a typed
  handled-attempt wrapper so unrelated runs continue.

### What was tricky to build

- Completion channels must be sized to the sum of local resource capacities so
  canceled dispatcher callers cannot strand task goroutines.
- Retry state must leave the attempt immutable and failed while returning the
  node to pending with a separate `ready_at` deadline.
- The database transaction must commit the operation marker atomically with
  side effects, while the simulated crash must occur strictly after commit.
- The workflow database and target database have intentionally different
  privacy roles: the target stores domain rows; workflow SQLite stores only
  compact refs and redacted attempt facts.

### What warrants a second pair of eyes

- Cancellation polling interval and behavior during transient store read
  errors.
- The go-go-goja fetch module's future redirect behavior relative to this host
  wrapper.
- SQL statement policy if less-trusted bundles are introduced before process
  isolation Slice 11.
- Whether retry backoff should add capped exponential policy in a later slice;
  this slice pins deterministic fixed backoff.

### What should be done next

1. Add public documentation and update both design implementation-status
   sections.
2. Add migration and dispatcher edge tests where needed.
3. Run repository-wide validation under `GOWORK=off` and inspect dependency
   upgrade impact.
4. Check tasks only after docs and fresh complete evidence pass.

### Code review instructions

- Start with the HTTP, dispatcher, and database integration tests.
- Trace exact resource/retry/module metadata from bundle through plan, SQLite,
  lease, attempt, and registry resolution.
- Verify HTTP origins and database handles exist only in host configuration,
  never in plans.
- Inspect target audit count before and after workflow restart.

### Technical details

- HTTP resource: `network.http.public`.
- Database resource: `database.sync.primary`.
- Retry bounds: three attempts, 10 ms deterministic fixture backoff.
- Dispatcher wake fallback: 50 ms production default, configurable in tests.
- Operation key: canonical SHA-256 of `{runId,nodeKey}`.

## Step 21: Validate the dependency upgrade and existing repository behavior

### Prompt Context

**User prompt (verbatim):** (same as Step 19)

**Assistant interpretation:** Expand validation from the new packages to every
scraper package and release build, then fix compatibility rather than accepting
a narrowed workflow-only pass.

**Inferred user intent:** Slices 3–5 must preserve the existing application, not
just work in isolated tests.

**Commit:** `7df9f59` — "workflowv3: validate dependency upgrade and migration"

### What I did

- Ran `GOWORK=off go mod tidy` and the full repository test suite.
- Adapted two existing submit-verb scanners to the current jsverbs API after
  verifying that v0.10.6 removed implicit public-function discovery entirely.
- Added an old-schema migration test that creates the completed Slice 1–2
  `v3_runs`, `v3_nodes`, and `v3_attempts` tables, opens them through the new
  store, and verifies every Slice 3–5 column was added.
- Ran `make validate`, full isolated Go lint, JavaScript syntax checks, and the
  rendered public help smoke test.
- Committed `go mod tidy` cleanup and generated logcopter files for both new
  fixture packages.

### Why

- Upgrading a shared scripting dependency can alter v2 scanner APIs even when
  workflow-v3 focused tests pass.
- Existing minimal-v3 databases must reopen safely after the resource/retry
  schema expansion.
- Generated package artifacts are part of this repository's build contract.

### What worked

- Focused site/submission/cmd tests passed after the scanner adaptation.
- `GOWORK=off go test ./... -count=1` passed.
- `make validate` passed, including all Go tests, web unit tests, code
  generation, Go binary build, TypeScript build, and Vite production build.
- `GOWORK=off .bin/golangci-lint run ./cmd/... ./pkg/...` reported `0 issues`.
- All four new JavaScript files pass `node --check`.
- The rendered help title is `Workflow V3 Runtime Slices 1–5`.
- Old minimal-v3 table layouts acquire resource, retry, ready-time, attempt
  resource, and dispatch columns without data-destructive recreation.

### What didn't work

- The first repository-wide test after upgrading go-go-goja failed in two
  existing files:

  `unknown field IncludePublicFunctions in struct literal of type jsverbs.ScanOptions`

  v0.10.6 deliberately removed both the field and implicit public-function
  verb discovery. The old code set it to `false`; removing the obsolete field
  preserves exactly that behavior. Full tests then passed.

### What I learned

- An upstream API deletion can preserve semantics when it removes the behavior
  an explicit old flag had disabled; this still requires source and test
  evidence, not blind field removal.
- `CREATE TABLE IF NOT EXISTS` does not evolve existing SQLite tables, so the
  additive column migration must remain explicit and tested.

### What was tricky to build

- The old-schema migration fixture had to define enough foreign-key targets for
  the current embedded schema to create all newly introduced tables and
  indexes, while still omitting the columns under test.

### What warrants a second pair of eyes

- Review go-go-goja v0.8.3→v0.10.6 release changes beyond the compile-visible
  jsverbs field; the repository-wide suites and builds currently pass.
- Review whether `make lint` itself should set `GOWORK=off` in a separate build
  policy change.

### What should be done next

- Add explicit 429/404 classification and cross-connection resource-admission
  evidence, then perform the final requirement audit.

### Code review instructions

- Review the two one-line jsverbs changes against v0.10.6 `ScanOptions` and its
  removal design note.
- Run the old-schema migration test against a temporary SQLite database.

### Technical details

- Shared dependency: go-go-goja v0.10.6.
- Full validation warning only: existing Vite chunk exceeds 500 kB.

## Step 22: Harden credentials, crash semantics, and database-scoped admission

### Prompt Context

**User prompt (verbatim):** (same as Step 19)

**Assistant interpretation:** Treat the first green repository validation as an
opportunity for adversarial requirement review, not automatic completion.

**Inferred user intent:** Prove the hard boundaries named in the goal under the
strongest realistic failure mode.

**Commit:** `c1e0023` — "workflowv3: harden resource and capability boundaries"

### What I did

- Added explicit 429 rate-limit and terminal 404 status classification tests.
- Made public fetch profiles fail worker boot when origins are empty/wildcard,
  limits are invalid, or environment/file credential sources are enabled.
- Added a guarded RoundTripper that reapplies origin policy and rejects URL
  userinfo, `Authorization`, `Cookie`, and `Proxy-Authorization` headers on
  every request, including redirects.
- Added a URL-password canary test proving zero network contact and zero
  workflow-SQLite leakage.
- Strengthened the database crash test: it now runs the task side effect, drops
  the typed task result without calling `Complete` or `Fail`, closes the store
  with a running attempt, waits for lease expiry, and restarts. Attempt one is
  `lease_lost`; attempt two observes the same operation key and does not write
  again.
- Asserted the target operation key equals the canonical SHA-256 of exact
  run/node identity.
- Added a two-connection SQLite race proving resource capacity is
  database-scoped, not merely process-local.
- Replaced textual RFC3339 deadline comparisons with parsed Go times in
  projections and SQLite `julianday` comparisons in lease eligibility/expiry.
- Removed the ready-candidate `LIMIT 100`, which could starve a compatible node
  behind one hundred capacity- or implementation-blocked rows.

### Why

- HTTP URL userinfo can create Basic Authorization implicitly even when explicit
  credential APIs are disabled.
- A task-reported post-commit error is weaker than a process death before any
  workflow outcome is recorded; lease-loss recovery is the required proof.
- RFC3339Nano strings are not lexicographically ordered when one timestamp omits
  fractional seconds.
- Resource capacity must hold across independent store connections to be a
  durable admission invariant.

### What worked

- 429 creates three `rate-limit/HTTP_FETCH_RATE_LIMIT` attempts; 404 creates one
  terminal `validation/HTTP_FETCH_STATUS` attempt.
- URL credentials never reach the test server and their canary is absent from
  SQLite/WAL/SHM.
- Unsafe public profile construction fails before a runtime can advertise it.
- Two concurrent SQLite connections produce exactly one resource-capacity
  winner.
- The stronger crash test still reports the 18.04% workflow storage ratio and
  exactly one target operation/audit row for 500 customer writes.
- Focused tests, race tests, and lint pass after hardening.

### What didn't work

- No command failed in this step. The deadline and candidate-limit defects were
  found by manual invariant review before they manifested as flaky tests.

### What I learned

- Security review must follow the behavior of `net/http`, not only exposed
  JavaScript APIs; URL credentials and cookie jars reach the transport layer.
- Durable idempotency evidence is strongest when the workflow store has no
  knowledge of the task's returned failure at all.
- Bounded candidate scans need pagination with continuation; a bare SQL limit
  silently changes scheduling correctness. The current slice prefers complete
  scanning over starvation and can add measured pagination later without a
  correctness cap.

### What was tricky to build

- A custom transport must wrap, not replace, an injected client's transport so
  test/proxy/TLS behavior remains intact while policy executes first.
- The crash test must avoid the engine's correct failure persistence path, so it
  resolves/materializes/runs the leased task directly and then abandons the
  running lease.

### What warrants a second pair of eyes

- Public fetch currently forbids all explicit authorization/cookie headers by
  design. Future authenticated aliases must use different names and policy,
  never relax `fetch:public`.
- Full candidate scanning is correct but should be benchmarked before very
  large ready queues; any optimization must preserve continuation fairness.

### What should be done next

- Rerun all validation after documentation, commit the final evidence, and map
  every goal clause to fresh artifacts before completion.

### Code review instructions

- Start with `publicFetchTransport.RoundTrip` and unsafe-profile tests.
- Confirm the database crash path never calls `Store.Fail` for attempt one.
- Run the cross-connection capacity race under `-race`.

### Technical details

- HTTP credential guards apply on every RoundTrip.
- Lease/retry timestamps remain RFC3339Nano at rest but compare through parsed
  time or SQLite `julianday`.

## Step 23: Finalize typed task and output-privacy contracts

### Prompt Context

**User prompt (verbatim):** (same as Step 19)

**Assistant interpretation:** Check type surfaces and outbound artifacts, not
only database persistence, before the final audit.

**Inferred user intent:** Ensure trusted task authors can see the real ABI and
that an HTTP credential cannot be echoed into a durable result accidentally.

**Commit:** `c4c670b` — "workflowv3: finalize typed privacy contracts"

### What I did

- Added the exact `workflow/task` TypeScript declaration and golden, including
  artifact metadata, fresh-attempt identity, stable operation key, checkpoint,
  typed JSON outputs, success, failure, and implementation wrappers.
- Compiled both authoring and task declarations with TypeScript in addition to
  exact Go golden comparison.
- Completed the failure-class vocabulary with the already-designed `budget`,
  `configuration`, and `identity` classes; this makes the database
  reconfiguration guard a valid typed failure path instead of a latent
  TypeError if the invariant ever regresses.
- Changed HTTP snapshots to store stable list indexes rather than final request
  URLs and asserted the query credential canary is absent from the output
  artifact as well as SQLite/WAL/SHM.
- Added explicit `implementation-unavailable` blocked-projection evidence for a
  bundle mismatch.

### Why

- Runtime type declarations must include the operation key introduced by Slice
  5 or JavaScript authors will rely on undocumented dynamic properties.
- A compact external artifact is outside workflow SQLite, but it is still a
  durable result; copying a query credential there would violate the intended
  public snapshot boundary.
- Failure vocabulary and task implementations must agree on every class even
  when a negative branch is expected never to execute.

### What worked

- Exact task DTS comparison passes.
- TypeScript compiles both declaration files with no errors.
- Focused tests and race suites pass.
- Focused lint reports zero issues.
- The HTTP plan digest changed deterministically with the bundle source and its
  reviewed golden was regenerated.

### What didn't work

- N/A. Manual API/output review found the two contract gaps before a failing
  test.

### What I learned

- Privacy review must inspect referenced outputs as well as the control-plane
  database, even when the formal storage-amplification boundary excludes
  external artifacts.
- An unreachable negative branch can still hide vocabulary drift; type/failure
  contracts need direct unit coverage.

### What was tricky to build

- `workflow/task` declarations describe a trusted runtime module rather than the
  safe authoring module, so they are generated and tested separately to avoid
  capability confusion.

### What warrants a second pair of eyes

- The task DTS currently uses schema IDs plus generic `unknown` values; future
  descriptor-specific DTS should refine domain payloads without weakening Go
  schema authority.

### What should be done next

- Perform one final repository-wide validation and requirement mapping after
  committing documentation/bookkeeping.

### Code review instructions

- Compare `TaskTypeScript()` byte-for-byte with its golden and run the explicit
  TypeScript command in the public help page.
- Inspect the HTTP artifact for absence of its URL query canary.

### Technical details

- Task DTS: `pkg/workflowv3runtime/testdata/workflow-task.d.ts`.
- HTTP snapshot identity field: zero-based stable `index`.

## Step 24: Audit Slices 3–5 against the active completion contract

### Prompt Context

**User prompt (verbatim):** (same as Step 19)

**Assistant interpretation:** Mark the three slices complete only after every
outcome, constraint, and verification clause maps to fresh evidence.

**Inferred user intent:** Receive a reproducible handoff with no narrowed or
implicitly deferred requirement.

### Outcome audit

| Goal requirement | Fresh evidence |
|---|---|
| Real allowlisted HTTP snapshot | Authored `workflow.js`, exact bundle, IR/plan goldens, and successful local HTTP integration in `pkg/testfixtures/workflowv3http` |
| Exact `fetch:*` without ambient network | Explicit sealed `fetch:public`; empty/wildcard profile and missing alias rejected; denied origins, redirects, URL credentials, credential sources, and sensitive headers blocked |
| Typed HTTP failures/retry/limits/cancellation/redaction | 503 retry then success, 429 bounded retries, terminal 404, transport denial, 64-byte response bound, lease-watched cancellation, stable code/message assertions, SQLite/output canary scans |
| HTTP resource class | Canonical plan and attempts pin `network.http.public`; dispatcher capacity is one in integration |
| Work-conserving immediate refill | Timeline holds HTTP-1 and unrelated slow task, releases only HTTP-1, and observes HTTP-2 start before slow completion |
| Independent durable capacities | Mixed resource projection and two independent SQLite connections prove one database-scoped winner per class |
| Dependency/fairness/determinism | Existing dependency tests plus per-resource run fairness, stable tie ordering, exact plan goldens, and deterministic single-action `DispatchOnce` |
| Append-only attempts/retries/fencing | Retry/backoff/reopen, lease-loss, concurrent lease, stale completion, cancellation epoch, and immutable attempt snapshots |
| Blocked projections | Tests cover dependency, retry-backoff, resource-capacity, and implementation-unavailable reasons |
| Real preconfigured database synchronization | Authored fixture and exact `db:sync` alias execute 500 target writes through a Go-provided `*sql.DB` |
| Script reconfiguration denied | Every database attempt calls `configure()` and receipt proves `configureDenied=true`; nil host handle fails registry construction |
| Transactions and idempotency | Side effects, operation marker, and audit commit in one target transaction; operation key equals SHA-256 of exact run/node identity |
| Crash after side effect | Attempt one commits, its result is discarded, store closes while lease is running, restart reclaims it as `lease_lost`, and attempt two adds zero writes |
| Schema/cardinality/failure isolation | Ref schema/output tests, duplicate/cardinality failures, one bad database run terminally fails while another succeeds |
| Compact privacy and reopen | 499,554 source bytes versus 90,112 workflow SQLite bytes (18.04%); source/SQL/URL/body canaries absent; final refs and plan digest reopen identically |
| Go remains authoritative | Go owns normalization, resource/retry policy, digest, compile, module admission, leases, capacity, attempts, schemas, outputs, and persistence |
| Existing behavior preserved | Minimal-v3 schema migration passes; full v2/v3 repository tests and build pass after the reviewed jsverbs API adaptation |
| Documentation and continuous diary | Public help, two design docs, tasks, changelog, relations, and Steps 19–24 updated in focused commits |

### Privacy audit

- Workflow SQLite/WAL/SHM contains no HTTP query/password canary, response-body
  canary, database source canary, source SQL text, or target rows.
- HTTP output does not copy request URLs; stable indexes prevent query-token
  echo. Public response bytes exist only in the external content-addressed
  snapshot artifact.
- Target database intentionally contains domain rows and the compact operation
  key; workflow SQLite contains only its receipt ref.
- HTTP/database task errors are converted to stable typed failures before they
  cross the runtime boundary. Persisted messages contain only
  `task reported <CODE>`.
- Source review confirms the fetch module does not log request/response bodies;
  database debug logging emits fixed trusted query text but not parameter values.
  No workflow report includes source, credentials, headers, bodies, or rows.

### Verification audit

Fresh final commands and results:

- `make validate` — passed all Go tests, web unit tests, generation, Go binary
  build, TypeScript application build, and Vite production build.
- `GOWORK=off .bin/golangci-lint run ./cmd/... ./pkg/...` — `0 issues`.
- `GOWORK=off go test -race ./pkg/workflowv3runtime
  ./pkg/workflowv3sqlite -count=1` — passed.
- Four new JavaScript sources pass `node --check`.
- Authoring and task DTS goldens pass and both declarations compile with `tsc`.
- HTTP/database/linear IR and plan goldens pass.
- Verbose HTTP, dispatcher, database crash/privacy, old-schema migration,
  retry/reopen, cross-connection resource, and fairness suites pass.
- Public help smoke renders `Workflow V3 Runtime Slices 1–5`.
- `rg` finds no TODO/FIXME/HACK in new packages/fixtures.
- `git diff --check` passes; generated outputs introduced no uncommitted files.
- `docmgr doctor` and relevant frontmatter validation pass.

The only build advisory is the repository's previously documented Vite chunk
size warning; it is not a failed check or workflow-v3 requirement and no source
from that frontend bundle changed in this tranche.

### What I did

- Repeated every completion-critical command after the final privacy/type
  changes.
- Ran key evidence tests verbosely and preserved byte ratios in output.
- Checked final tasks, placeholder scan, generated tree, help rendering, and
  working-tree state.
- Mapped each active-goal clause to specific code and test evidence above.

### What worked

- Every required command and negative/positive artifact check passed.
- All four Slice 3–5 docmgr tasks are checked.
- No required capability is deferred to a later slice.

### What didn't work

- N/A in the final audit rerun. All earlier failures and fixes are preserved in
  Steps 20–21.

### What I learned

- Slices 3–5 form one coherent contract: resource policy belongs in canonical
  task identity; sensitive host policy stays outside plans; exact aliases join
  them at worker admission.

### What was tricky to build

- The audit had to distinguish future capabilities (lazy maps, reductions,
  budgets, gates, rolling generations, untrusted isolation) from requirements
  of these three slices. None is used as an excuse to weaken current HTTP,
  dispatch, or database invariants.

### What warrants a second pair of eyes

- Transaction ordering in resource lease/fairness updates.
- Lease watcher polling and cancellation during SQLite contention.
- Public fetch transport composition with custom clients.
- Target transaction/idempotency marker behavior on non-SQLite databases.

### What should be done in the future

- Proceed to Slice 6 lazy maps only after review of this checkpoint.
- Benchmark full ready-queue scanning before adding starvation-safe pagination.

### Code review instructions

- Review commits `b05e5a0`, `c1e0023`, and `c4c670b` first, then their paired
  documentation commits.
- Reproduce the verification commands and verbose tests listed above.
- Inspect workflow SQLite bytes separately from the intentional target database
  and external artifact store.

### Technical details

- Active goal: `dafed540-d78f-4507-968d-e24ef5fc2a20`.
- Final implementation commits: `b05e5a0`, `7df9f59`, `c1e0023`, `c4c670b`.
- Final documentation commits through the preceding step: `d35709e`,
  `3115402`.

## Step 25: Audit and document the complete twelve-slice architecture

The next implementation tranche began with the required documentation audit.
Slices 6–10 had short sections in the primary architecture and relevant
material in the cookbook, but they did not have dedicated implementation-ready
design documents. Five dedicated design documents now exist, and a separate
intern-oriented guide explains the architectural progression and acceptance
boundary for every slice from the linear foundation through paid RAG/TTC work.

The guide deliberately distinguishes implemented evidence for Slices 1–5 from
target contracts for Slices 6–12. It does not report planned maps, reductions,
rolling generations, budgets, gates, process isolation, or RAG integration as
already running.

### Prompt Context

**User prompts (verbatim):**

> slices 6. Lazy maps: scale-out is real 7. Reductions: scale-in is real 8. Registry generations: upgrades are real 9. Budgets/projections: operations are real 10. Approval gates: waiting is real Before you start: are the slices documented in their own right in a design doc?

> Ok, write a detailed design / analysis guide about all the slices, with a section for each, for an intern, from the architectural point of view.
>
> 3. HTTP snapshot: external work is real
>    4. Dispatcher: concurrency is real
>    5. Database sync: side effects are real
>    6. Lazy maps: scale-out is real
>    7. Reductions: scale-in is real
>    8. Registry generations: upgrades are real
>    9. Budgets/projections: operations are real
>    10. Approval gates: waiting is real
>    11. Process isolation: broader trust is viable
>    12. RAG/TTC: expensive production workload is safe
>
> for all these

> and slices 1 2 as well. Go over your diary and all to refresh your memory if needed.
>
> once done, upload to remarkable as a bundle

**Assistant interpretation:** Audit the prior documentation, refresh the full
implementation history, and write one detailed intern guide covering the
architectural purpose, state, failures, and evidence contract for all twelve
vertical slices before implementing Slices 6–10.

**Inferred user intent:** Give a new engineer one coherent explanation of why
the slices are ordered this way and what each one must prove, while also making
the immediate Slice 6–10 designs explicit enough to implement safely.

**Active goal:** `f47fcc94-5a32-47e1-8a60-d6b19cd8a417`.

**Commit (docs):** `84933ec` — "docs: design workflow v3 slices six
through ten"

### What I did

- Re-read the complete investigation diary, evidence catalogue, existing slice
  sequence, bundle-generation design, current canonical types, registry,
  dispatcher, schema, and projections.
- Confirmed the audit answer: Slices 6–10 were broadly documented but did not
  each have their own implementation-ready design document.
- Added six docmgr tasks for Slices 6–10 and their final audit.
- Created dedicated design documents 03–07 for lazy maps, reductions, registry
  generations, budgets/projections, and approval gates.
- Added `design-doc/08-workflow-v3-slices-1-through-12-intern-architecture-and-analysis-guide.md`.
- Documented shared representation, privacy, identity, lease, attempt,
  capability, state, migration, and evidence invariants.
- Added one detailed section per slice, including implemented paths and measured
  evidence for Slices 1–5 and explicit state/schema/transaction/test contracts
  for Slices 6–12.
- Added a cross-slice state model, dependency sequence, intern workflow, status
  table, review paths, and source links.
- Replaced all five generated templates with implementation-ready contracts for
  canonical APIs, durable schemas, transaction ordering, state machines,
  failure vocabularies, migration, projections, test matrices, alternatives,
  and hard acceptance criteria.
- Validated all six new design documents, checked local links and Markdown
  fences, ran docmgr doctor, and checked whitespace/placeholders.
- Uploaded the six-document bundle as `SCRAPER WORKFLOW V3 Slices 1 Through 12
  Architecture Guide.pdf`.

### Why

- Short roadmap bullets are insufficient for dynamic graph, upgrade, budget,
  and gate implementation because they do not freeze state-machine and crash
  semantics.
- Slices 11 and 12 must remain visible while implementing 6–10 so in-process
  trust assumptions and final RAG acceptance do not get lost.
- The previous TTC failure shows that storage, scheduling, privacy, and paid
  provider behavior must be explained as one sequence rather than isolated
  features.

### What worked

- The existing diary provided exact implementation failures and evidence for
  Slices 1–5 rather than requiring reconstruction from commit messages.
- The new guide covers all twelve requested slices and explicitly labels future
  behavior as target architecture.
- Current package and fixture paths are linked directly in frontmatter.
- Frontmatter validation returned `Frontmatter OK` for all six documents;
  docmgr doctor returned `✅ All checks passed`.
- Link/fence validation passed. The six documents contain 14,648 words total
  after docmgr relations were added.
- reMarkable returned:

  `OK: uploaded SCRAPER WORKFLOW V3 Slices 1 Through 12 Architecture Guide.pdf -> /ai/2026/07/21/SCRAPER-WORKFLOW-V3`

### What didn't work

- The initial answer to the audit correctly said the slices were not documented
  in their own right; the five generated docmgr documents initially contained
  only templates. They were all replaced before implementation began.
- The first `git diff --check` found trailing whitespace copied verbatim from
  the prompt's aligned slice list and one extra changelog newline. I normalized
  trailing padding and final newlines, then reran the check successfully.

### What I learned

- The twelve-slice sequence is best understood as progressive introduction of
  failure boundaries: identity, authoring, remote I/O, concurrency, side
  effects, cardinality, aggregation, upgrades, operations, waiting, trust, and
  paid production integration.
- The dedicated Slice 6–10 documents need more precision than the cross-slice
  guide around exact APIs, tables, transaction ordering, migrations, and test
  matrices.

### What was tricky to build

- The guide had to teach future contracts without implying they already exist.
  Every section therefore states implementation status and separates current
  evidence from required evidence.
- Dynamic maps and reductions must be described without putting item payloads
  or huge dependency sets back into SQLite. The guide uses immutable manifests,
  deterministic keys, bounded pages, and bounded fan-in throughout.

### What warrants a second pair of eyes

- Proposed map child-key encoding and reduction partition identity before they
  become persisted public contracts.
- Generation acquisition ordering relative to durable lease creation.
- Conservative settlement semantics for ambiguous provider timeout/lease loss.
- Gate compare-and-swap behavior during approval/cancellation races.

### What should be done in the future

- Commit the validated documentation tranche, then implement Slice 6 from its
  frozen design and evidence matrix.

### Code review instructions

- Read design document 08 in order, then compare each Slice 6–10 section with
  its dedicated design document.
- Cross-check Slices 1–5 against the real fixtures and diary Steps 10–24.
- Run frontmatter validation, docmgr doctor, link checks, and
  `git diff --check` before committing.

### Technical details

- Intern guide covers Slices 1–12 in approximately 45 KiB and 6,238 words.
- Dedicated slice design paths are `design-doc/03-*.md` through
  `design-doc/07-*.md`.
- Bundle destination:
  `/ai/2026/07/21/SCRAPER-WORKFLOW-V3/SCRAPER WORKFLOW V3 Slices 1 Through 12 Architecture Guide.pdf`.
- Documentation commit: `84933ec`.

## Step 26: Freeze the canonical lazy-map contract

Slice 6 implementation started below the JavaScript surface by extending the
canonical Go representation. Workflow IR and plans can now represent typed set
inputs, deterministic map declarations, one symbolic item binding, bounded
expansion policy, and named set outputs while leaving existing static workflow
JSON unchanged when these fields are absent.

Validation resolves set schemas and prior map outputs, pins the exact item-task
implementation/policy in `PlanMap`, requires one item binding and one output,
and rejects invalid limits, key collisions, wrong item ownership, forward map
references, and output schema drift.

### Prompt Context

**User prompt (verbatim):** (same as Step 25)

**Assistant interpretation:** Begin Slice 6 from the validated dedicated design
by freezing canonical identities and validation before persistence or runtime
work.

**Inferred user intent:** Build lazy maps on one deterministic Go-owned contract
instead of adding ad hoc dynamic rows or JavaScript runtime callbacks.

**Commit (code):** `cfc8f6d` — "workflowv3: freeze lazy map core contracts"

### What I did

- Added `SetRef`, `IRSetInput`, `IRSetOutput`, `MapPolicy`, `IRMap`, and
  `PlanMap`.
- Added `mapKey` to symbolic value refs for the authoring-time item binding.
- Added the versioned `scraper-workflow-item-manifest/v1` identity.
- Extended workflow IR/plan with omitted-when-empty set/map fields so completed
  static workflow goldens retain their serialized shape.
- Added strict map source, policy, task, binding, schema, key, and set-output
  validation.
- Compiled map templates to exact implementation, module, resource, retry,
  schema, binding, and expansion policy facts.
- Added deterministic compilation and negative contract tests.

### Why

- Expansion keys and persisted rows cannot be safe until map intent has one
  canonical representation independent of runtime callbacks.
- Omitting empty map fields preserves existing Slice 1–5 canonical bytes rather
  than silently changing all historical plan digests.

### What worked

- `GOWORK=off go test ./pkg/workflowv3 -count=1` passed.
- `GOWORK=off .bin/golangci-lint run ./pkg/workflowv3` reported `0 issues`.
- `git diff --check` passed.
- Existing compiler tests and the new deterministic map tests pass together.

### What didn't work

- N/A. The first formatted core implementation passed focused tests and lint.

### What I learned

- Map chaining can remain deterministic by permitting only references to prior
  map outputs in canonical declaration order; a forward output is rejected.
- A separate omitted `SetOutputs` collection preserves the existing value
  output JSON contract while allowing typed dynamic results.

### What was tricky to build

- A mapped task may have additional static bindings, but exactly one binding
  must be the symbolic item owned by that map. Validation therefore resolves
  item schema from the set source and all other bindings through existing
  value-ref rules.
- The map output item schema comes from the task's sole declared output. The
  first slice deliberately rejects multi-output item tasks until a port
  selection contract is designed explicitly.

### What warrants a second pair of eyes

- Whether first-class map output port selection should be added now or retain
  the simpler one-output contract through Slice 6.
- Whether declaration-order map chaining is preferable to a separate map DAG
  topological sort in the first release.
- Exact child-key encoding before persistence code freezes it.

### What should be done in the future

- Add opaque `inputSet`, `map`, item, and set-output handles to the authoring
  module with direct-Go/JavaScript equality and exact DTS goldens.
- Implement manifest codec and deterministic child-key goldens before schema
  migration.

### Code review instructions

- Start at `TestCompileMapPinsTemplateAndSetIdentity`, then review
  `ValidateIR`, `setRefSchema`, and the new types.
- Confirm existing static plan goldens remain unchanged.
- Reproduce the focused test and lint commands above.

### Technical details

- Code commit: `cfc8f6d`.
- New manifest schema: `scraper-workflow-item-manifest/v1`.
- First map contract requires one `map-item` binding and one task output.

## Step 27: Author typed lazy maps from JavaScript

The safe `require("workflow")` module now exposes opaque typed set handles and a
map builder. JavaScript declares one set input, invokes the map callback once
with a symbolic item value, configures bounded expansion policy, and publishes
a named set output. The resulting IR and plan are exact reviewed goldens over
the canonical Go model from Step 26.

The authoring runtime still has no execution or store authority. Descriptor
factories receive the symbolic item through the same hidden object-identity
map used by ordinary value refs, so plain objects and value refs cannot be
substituted for set handles.

### Prompt Context

**User prompt (verbatim):** (same as Step 25)

**Assistant interpretation:** Extend the minimal DSL only after the core map
contract is frozen, preserving direct-Go/JavaScript convergence and type parity.

**Inferred user intent:** Make scale-out authorable without turning JavaScript
callbacks into durable/runtime expansion logic.

**Commit (code):** `c898571` — "workflowv3: author typed lazy maps from
JavaScript"

### What I did

- Added process-local opaque `SetRef` handles to authoring state.
- Added `inputSet<T>`, `map<I,O>`, fluent `MapBuilder`, and `outputSet`.
- Made the map callback execute once with an opaque `map-item` value.
- Added default expansion limits and fluent overrides.
- Rejected value-as-set handles and callbacks that do not return a registered
  task descriptor.
- Extended exact TypeScript declarations without `any`.
- Added exact map IR/plan goldens and initialized required non-map arrays as
  empty arrays rather than JSON `null`.

### What worked

- `GOWORK=off go test ./pkg/workflowv3 ./pkg/gojamodules/workflow -count=1`
  passed.
- Focused isolated lint reported `0 issues`.
- The exact DTS golden passed and `pnpm exec tsc --noEmit --skipLibCheck` compiled
  it successfully.
- `git diff --check` passed.
- The fixture asserts the authoring callback runs exactly once.

### What didn't work

- The first generated map goldens exposed `inputs`, `nodes`, and `outputs` as
  JSON `null` in a map-only workflow. These fields are required collections,
  so authoring and compilation now initialize them as empty arrays. Goldens
  were regenerated and reviewed after that correction.

### What I learned

- Omitted optional set/map collections and explicit empty required value/node
  collections preserve both prior static-plan bytes and clear map-only schema
  shape.
- Descriptor factories require no separate map-specific path: the symbolic
  item is a normal opaque value ref with a constrained `map-item` source.

### What was tricky to build

- The map callback must run while `activeBuild` is set so descriptor-only
  modules accept the invocation, but its returned descriptor must be inspected
  immediately and only normalized data retained.
- Output schema is inferred from the exact one-output task catalog entry; the
  runtime cannot inspect arbitrary JavaScript properties for it.

### What warrants a second pair of eyes

- Fluent map callback argument order and naming before wider public adoption.
- Whether the default `pageSize=64`, `maxItems=10000`, and
  `maxMaterializedAhead=128` should come from an explicit compiler profile
  rather than authoring defaults in the next policy refinement.

### What should be done in the future

- Add strict immutable manifest encoding/validation and deterministic child-key
  derivation with exact goldens.
- Then add additive SQLite expansion tables and one-page transaction tests.

### Code review instructions

- Start with `TestAuthorCompilesLazyMapToCanonicalPlan` and the two new goldens.
- Follow `inputSet`, `map`, and `outputSet` in `authoring.go`.
- Compare `TypeScript()` byte-for-byte with `testdata/workflow.d.ts`.

### Technical details

- Code commit: `c898571`.
- Exact plan digest: `sha256:f452bb74fe9ba4c5afbb16fb83776c174d709377ab43f20a608e3606b41f7676`.

## Step 28: Freeze map manifest and child identity

Slice 6 now has a strict immutable data-plane manifest and exact child-key
derivation. Manifests contain canonical sorted unique item keys and compact
artifact references whose schema matches the declared item schema. Child node
keys hash the map key, source manifest digest, and item key, keeping source
identity stable without exposing item keys in durable node identifiers.

### Prompt Context

**User prompt (verbatim):** (same as Step 25)

**Assistant interpretation:** Define and test the immutable source contract and
identity envelope before writing expansion rows.

**Inferred user intent:** Ensure restart/concurrency can replay map expansion
without duplicate children, payload persistence, or completion-order identity.

**Commit (code):** `e88f95d` — "workflowv3: define canonical map manifests and
child keys"

### What I did

- Added `ItemManifest`, `ManifestItem`, strict construction, encoding, decoding,
  and validation.
- Required an explicit array, exact schema, nonempty item schema, strictly
  increasing unique keys, valid bounded UTF-8 keys without controls, valid
  artifact refs, and exact per-item schema.
- Added `MapChildNodeKey` over a canonical digest envelope and hid raw item keys
  from node IDs.
- Hardened all artifact/source SHA-256 validation to reject correctly sized
  non-hex strings.
- Added canonical round-trip, malformed manifest, duplicate/order/schema/key,
  unknown-field, null-array, invalid-digest, and exact-key tests.

### Why

- The expansion transaction needs fully validated deterministic inputs before
  any child rows are inserted.
- Length/prefix-only digest checks allowed invalid identities to cross the
  durable boundary.

### What worked

- `GOWORK=off go test ./pkg/workflowv3 -count=1` passed.
- Focused isolated lint reported `0 issues`.
- `git diff --check` passed.
- Exact child key is
  `map:762fdd8fbf74c3fab9d50ead448fcdb7ba311e5cbfe061f2677ff94deee84552`
  for the frozen test envelope.

### What didn't work

- N/A. Focused implementation and tests passed after formatting.

### What I learned

- Strict increasing order simultaneously proves canonical ordering and
  uniqueness without a second durable sort policy.
- Validating digest hex centrally improves every existing artifact boundary,
  not only lazy maps.

### What was tricky to build

- Empty manifests are valid, but JSON `null` items are not. The validator must
  distinguish an explicit empty array from an absent/null collection.
- Item keys are allowed to be domain-readable Unicode but must be bounded,
  trimmed, valid UTF-8, and control-free before hashing or diagnostics.

### What warrants a second pair of eyes

- Whether 256 bytes is the correct public item-key ceiling.
- Whether production manifests require a separately indexed/paged artifact
  format before the 1,807-item acceptance workload.

### What should be done in the future

- Add expansion/page/item tables and the first atomic page materialization API.
- Test two independent SQLite connections against one expansion cursor.

### Code review instructions

- Review `manifest_test.go` first, then `ValidateItemManifest` and
  `MapChildNodeKey`.
- Confirm no item payload is represented in either type.

### Technical details

- Code commit: `e88f95d`.
- Item manifest schema: `scraper-workflow-item-manifest/v1`.

## Step 29: Materialize durable lazy-map pages

The v3 store now persists map control state separately from ordinary static
plans and materializes deterministic child nodes in bounded atomic pages.
Each child uses the exact compiled task template and one compact item artifact
reference. Existing node leasing, attempts, resource admission, retries,
fencing, and output validation execute dynamic children without a second task
path.

Expansion respects materialized-ahead backpressure, survives store close/reopen,
and serializes correctly across independent SQLite connections. Child
completion advances authoritative terminal counts in the same fenced
completion transaction.

### Prompt Context

**User prompt (verbatim):** (same as Step 25)

**Assistant interpretation:** Implement the first durable expansion transaction
and prove restart, backpressure, item resolution, and multi-connection safety.

**Inferred user intent:** Make map scale-out a real store-backed state machine,
not only a canonical DSL feature.

**Commit (code):** `5ae1f14` — "workflowv3: materialize durable lazy map pages"

### What I did

- Added additive `v3_expansions`, `v3_expansion_pages`, and `v3_map_items`
  tables and indexes.
- Extended run submission to validate/store set input refs and initialize map
  state without payloads.
- Added `ExpandNextPage` with manifest/ref verification, cardinality ceiling,
  source identity checks, deterministic page digest, child key/ordinal,
  dynamic exact `v3_nodes`, item refs, dependencies, cursor CAS, event, and
  bounded backpressure.
- Added `map-item` lease input resolution through compact item rows.
- Made fenced child completion advance map terminal state and prevented run
  success while any expansion is nonterminal.
- Added zero-cardinality completion without a worker lease.
- Hardened legacy store test artifact helpers after strict hex validation.
- Added reopen/backpressure/resolve, independent-connection concurrency,
  empty-map, cardinality, and identity-drift tests.

### Why

- Dynamic children should reuse normal durable execution instead of introducing
  map-specific attempts or leases.
- Cursor, child rows, and page evidence must commit together so every crash is
  either before or after one complete page.

### What worked

- `GOWORK=off go test ./pkg/workflowv3sqlite -count=1` passed.
- `GOWORK=off go test -race ./pkg/workflowv3sqlite -count=1` passed.
- Focused isolated lint reported `0 issues`.
- Core, authoring, runtime, and SQLite focused package suites pass together.
- Two independent connections materialize two exact pages with four unique
  items and no duplicate node.
- Closing after page one and reopening resumes at index two.

### What didn't work

- Strict SHA-256 validation exposed old test helpers that constructed digests
  from non-hex run-name seeds. The store tests failed with errors such as:

  `run input "source": artifact digest: must be sha256 hex: encoding/hex: invalid byte`

  The helper now hashes seeds with SHA-256 and emits real hex; all suites pass.

### What I learned

- Backpressure must refuse an entire page when available ahead capacity is
  smaller than the next deterministic page; partial pages would make page
  boundaries dependent on completion timing.
- Dynamic nodes can keep symbolic `map-item` bindings because the node key joins
  to exactly one compact item ref at lease input resolution.

### What was tricky to build

- The source artifact must match every submitted ref field, not only digest,
  while canonical manifest bytes must independently reproduce digest and size.
- Dynamic ordinals need deterministic nonoverlapping ranges per map while node
  identity remains independent of ordinal.
- Empty input has no child completion to advance state, so the expansion
  transaction must terminalize the map/run directly without creating an
  attempt.

### What warrants a second pair of eyes

- Dynamic ordinal range arithmetic based on preceding maps' `MaxItems`.
- Page planning currently decodes the complete manifest before the transaction;
  the 1,807-item fixture is bounded, but larger production manifests may need a
  verified page index.
- Map result-manifest publication is still the next required boundary before
  Slice 6 can complete.

### What should be done in the future

- Add dispatcher maintenance integration so expansion and execution interleave
  automatically.
- Build/publish the ordered map output manifest and expose authoritative map
  projections.
- Add cancellation state updates and the 1,807-item real fixture.

### Code review instructions

- Start with `expansion_test.go`, then read `ExpandNextPage` transaction order.
- Verify dynamic nodes enter the existing lease/attempt/complete path.
- Review `Complete` expansion update and run-terminal predicate together.

### Technical details

- Code commit: `5ae1f14`.
- First page test policy: page size 2, materialized-ahead 2.

## Step 30: Execute, publish, and project a real 1,807-item lazy map

Slice 6 now runs its full vertical path. The engine reads and verifies the
submitted manifest artifact, interleaves bounded expansion with ordinary
resource-aware leasing, executes each dynamic child in a fresh Goja runtime,
and publishes one ordered immutable output manifest after every child succeeds.
The run does not become successful until that manifest reference is committed.

A real JavaScript fixture processes 1,807 separate source artifacts across a
store close/reopen and eight-way work-conserving dispatch. It produces exactly
1,807 attempts and an ordered output manifest, while source-private fields stay
outside final outputs and workflow SQLite.

### Prompt Context

**User prompt (verbatim):** (same as Step 25)

**User follow-up (verbatim):** "is v3_expansions a table for backwards compatibility?"

**Assistant interpretation:** Finish Slice 6 runtime/publication/projection and
clarify the persistence purpose of the new expansion table.

**Inferred user intent:** Confirm that new schema is principled first-class map
state rather than a legacy shim, and obtain real scale/restart evidence.

**Commit (code):** `0481f33` — "workflowv3: execute and publish real lazy maps"

### What I did

- Added engine discovery and artifact rehydration for expandable maps.
- Interleaved one expansion action, one finalization action, and ordinary lease
  dispatch under materialized-ahead backpressure.
- Added canonical ordered output-manifest construction from successful child
  refs and idempotent artifact publication.
- Added `published` expansion state and prevented run success before map output
  publication.
- Exposed named set outputs through the existing run snapshot output map.
- Added authoritative map progress/backlog fields and derived
  `map-backpressure` reason to queue projections.
- Added terminal child failure and run cancellation transitions for expansion
  state.
- Added a real authored JavaScript map bundle and exact fixture IR/plan goldens.
- Executed 1,807 items across restart, verified all attempts/cardinality/order,
  inspected a real normalized output, and scanned persistence/output manifests
  for a private canary.
- Added a smaller 257-item execution of the same state machine under Go race
  instrumentation while retaining the full 1,807-item normal acceptance test.

### Why

- Expansion without root publication cannot provide a durable typed set to
  downstream reductions.
- A map cursor is not sufficient operational truth; materialized, terminal,
  source backlog, execution backlog, and backpressure must be derived visibly.
- `v3_expansions` is first-class Slice 6 state, not a backward-compatibility
  table. Compatibility comes from additive creation and the absence of rows for
  old static runs.

### What worked

- Full focused workflow-v3 packages pass.
- Runtime/store race suites pass after using the 257-item race profile.
- Focused isolated lint reports `0 issues`.
- Both new JavaScript files pass `node --check`.
- Logcopter generated-file check passes.
- Normal scale evidence:

  `source=7561185 persistedSQLite=5353472 ratio=0.7080`

- Race profile evidence:

  `source=1075178 persistedSQLite=872448 ratio=0.8114`

- The normal run completed in approximately 27 seconds and produced exact first
  and last keys `record-0000` and `record-1806`.

### What didn't work

- Running the 1,807 fresh-Goja attempts under the race detector exceeded first
  a 45-second and then a 120-second test deadline. The exact failures were:

  `Condition never satisfied`

  at `map_integration_test.go:92`, after approximately 54 and 131 seconds.
  This was instrumentation overhead, not a failed durable invariant: the same
  full workload passed normally. I added build-tagged race-test constants so
  race mode runs 257 real items through the identical restart/concurrency path,
  while normal mode retains the hard 1,807-item acceptance cardinality. The
  complete race package suite then passed in about eight seconds.

### What I learned

- Map publication needs an explicit `published` state after child success;
  otherwise a run can report success before its named output reference exists.
- Backpressure belongs in expansion state/projections, while dynamic task
  resource capacity remains in the existing node dispatcher.
- The output manifest can be rebuilt deterministically from item-index order and
  child output refs after a crash; artifact publication is content-addressed
  and idempotent.

### What was tricky to build

- Final child completion, expansion success, artifact creation, output-ref
  publication, and run success span SQLite and the external artifact store.
  Immutable artifact bytes are written first, then the compact reference and
  run state commit transactionally; a crash may orphan bytes but cannot publish
  partial state.
- The dispatcher must avoid expanding all items before execution. It alternates
  control actions and leasing, and the persisted ahead ceiling stops further
  pages until terminal children release backlog.
- Empty maps need a zero-item canonical output manifest and no task lease.

### What warrants a second pair of eyes

- Whether `RunSnapshot.Outputs` should remain a unified map for value and set
  manifest refs or gain a typed set-output field in a later API revision.
- Full-manifest decode cost before each page; current 1,807-item evidence is
  acceptable, but indexed page artifacts may be needed at much larger scale.
- Fairness among multiple simultaneously expandable maps; current candidate
  ordering is deterministic but has not yet added persistent expansion dispatch
  counts.

### What should be done in the future

- Add deterministic completion-order/concurrency comparison and explicit
  expansion fairness if review requires it.
- Update public help and Slice 6 implementation status after final focused/full
  validation.
- Begin Slice 7 bounded reduction only after the Slice 6 completion audit.

### Code review instructions

- Start at `map_integration_test.go`, then follow `Engine.ExpandOne`,
  `ExpandNextPage`, normal node execution, `MapOutputManifest`, and
  `PublishMapOutput`.
- Inspect `workflowv3map/workflow.js` and `tasks.cjs` for the exact authored and
  executed paths.
- Reproduce normal and race-focused commands separately.

### Technical details

- Code commit: `0481f33`.
- Normal item count: 1,807; race-instrumented item count: 257.
- Resource capacity: `cpu.map=8`.
- Page size: 64; max materialized ahead: 128.

## Step 31: Complete the Slice 6 validation audit

Slice 6 now has fresh repository-wide validation, concurrency-independent output
evidence, additive migration coverage, public help, updated architecture status,
and a requirement-by-requirement audit. The task is ready to close without
deferring map behavior to Slice 7.

### Prompt Context

**User prompt (verbatim):** (same as Step 25)

**Assistant interpretation:** Harden the final scale test, prove output identity
across concurrency levels, update public/design documentation, and close Slice 6
only after full validation.

**Inferred user intent:** Receive a reviewable finished scale-out slice before
bounded reduction work begins.

**Commits (code):**

- `c055c1a` — "workflowv3: stabilize map scale and migration evidence"
- `f05ba65` — "workflowv3: prove map digest across concurrency levels"

### Outcome audit

| Slice 6 requirement | Fresh evidence |
|---|---|
| Typed set/map authoring | `inputSet`, opaque item callback, `map`, fluent bounded policy, `outputSet`, exact DTS |
| Go-owned canonical truth | Direct/core compiler tests plus authored real fixture IR/plan goldens |
| Deterministic item identity | Strict ordered manifest and exact opaque child-key golden |
| Bounded lazy expansion | Page size and materialized-ahead persisted; one atomic page per action |
| Restart-safe cardinality/order | 1,807-item fixture closes after 70 actions, reopens, and finishes with exact first/last keys and 1,807 attempts |
| Multi-connection safety | Two SQLite connections materialize four unique items in two pages |
| Ordinary execution invariants | Dynamic children use exact registry, resource admission, append-only attempts, retries, cancellation epoch, schemas, and fenced completion |
| Backpressure/projections | Map totals, cursors, materialized/terminal backlogs, and `map-backpressure` assertions |
| Failure/cancellation isolation | Terminal child failure marks map/run failed; cancel stops candidates; global stale completion fencing remains covered |
| Deterministic scale-in handoff | Ordered output manifest publication is idempotent; 65-item normal runs at capacities 1 and 8 have identical digest |
| Empty map | Publishes a zero-item manifest with no lease |
| Privacy/storage | 7,561,185 source bytes versus about 5.35 MB workflow SQLite; source-private canary absent from SQLite and output manifest |
| Compatibility | Existing static goldens unchanged; old completed minimal database opens with all additive expansion tables/columns |

### What I did

- Added migration assertions for expansion, page, and item tables/columns.
- Added a capacity-1 versus capacity-8 output-digest comparison.
- Increased the full scale test deadline for loaded repository runs and added
  timeout snapshot/queue diagnostics.
- Updated public help to Slices 1–6 and documented `v3_expansions` as new
  first-class map state rather than compatibility storage.
- Updated the primary architecture, dedicated Slice 6 design, and all-slice
  intern guide implementation status.
- Ran local CI, full isolated lint, race suites, JS syntax, DTS compilation,
  help rendering, generated checks, placeholder scan, docmgr validation, and
  whitespace checks.

### What worked

- Final `make validate` passed, including all Go tests, web unit tests, code
  generation, Go binary build, TypeScript build, and Vite production build.
- `GOWORK=off .bin/golangci-lint run ./cmd/... ./pkg/...` reported `0 issues`.
- Runtime/store race suites passed.
- Both authoring/task DTS files compile with TypeScript.
- Map JavaScript workflow/task files pass `node --check`.
- Public help renders title `Workflow V3 Runtime Slices 1–6`.
- Relevant design frontmatter and docmgr doctor pass.
- Logcopter generated check, placeholder scan, and `git diff --check` pass.

### What didn't work

- The first repository `make validate` run reached the new 1,807-item test under
  full-suite load and exceeded its original 45-second deadline after 47.40
  seconds. The focused test had passed in about 27–35 seconds. I changed the
  deadline to 120 seconds and added diagnostic snapshot/queue logging. The next
  full validation passed; the runtime package completed in 76.515 seconds.
- The first full lint command found only formatting in the newly edited
  migration assertion map:

  `pkg/workflowv3sqlite/store_test.go:153:1: File is not properly formatted (gofmt)`

  I ran `gofmt`; lint then reported zero issues.
- I incorrectly ran docmgr ticket-frontmatter validation against the public
  Glazed help page. It reported missing `Ticket` and `DocType`, which are not
  fields in the help-page schema. The help page was instead validated by the
  successful rendered CLI smoke and repository build; docmgr validation was
  rerun only on ticket documents and passed.
- The first 65-item serial/concurrent digest test under race instrumentation
  exceeded 20 seconds and then 90 seconds in its serial fresh-runtime branch.
  Race mode now uses nine items for this ordering/concurrency comparison, while
  normal mode retains 65 items across two pages; the separate race scale path
  remains 257 items and normal hard scale remains 1,807. Both complete race
  suites pass.

### What I learned

- Full-suite load is materially different from a focused 1,807-fresh-runtime
  run, so a scale acceptance test needs a generous deadline plus state
  diagnostics rather than a tight performance assertion.
- Public help frontmatter and docmgr ticket frontmatter are separate schemas;
  rendered help/build is the correct smoke for the former.
- Deterministic output identity can be proven with a modest multi-page fixture,
  while cardinality and race instrumentation remain separate tests with clear
  purposes.

### What was tricky to build

- Completion-order independence needed evidence without doubling the expensive
  1,807-item run. The dedicated 65-item two-page test compares capacity 1 and 8
  artifacts, while the large run proves scale/restart/privacy.
- Migration evidence must prove not only new columns on old v3 tables but that
  entirely new Slice 6 tables appear when an old Slice 1–2 database opens.

### What warrants a second pair of eyes

- Map expansion fairness among many runs remains deterministic through run
  creation/map key ordering but is not yet persisted with separate expansion
  dispatch counters.
- Full manifest decode per expansion page is acceptable at 1,807 items but
  should be benchmarked before materially larger production sets.
- Unified `RunSnapshot.Outputs` representation for value and set refs.

### What should be done in the future

- Begin Slice 7 from its dedicated design using the published map manifest as
  the immutable reduction source.

### Code review instructions

- Review commits `cfc8f6d..f05ba65` in order with diary Steps 26–31.
- Reproduce `make validate`, isolated lint/race, TypeScript, JavaScript, help,
  generated, docmgr, and privacy commands.
- Inspect the 1,807-item and concurrency-digest tests separately.

### Technical details

- Slice 6 code commits: `cfc8f6d`, `c898571`, `e88f95d`, `5ae1f14`,
  `0481f33`, `c055c1a`, `f05ba65`.
- Existing Vite large-chunk advisory remains non-failing and unrelated.

## Step 32: Freeze bounded reduction IR and authoring

Slice 7 implementation started with one homogeneous bounded reduction contract.
A reduction consumes a typed set, invokes its JavaScript authoring callback once
with an opaque symbolic partition value, pins one exact reducer task, and
publishes one typed root value. Fan-in and maximum levels are canonical policy.

The first contract requires reducer output schema to equal source item schema.
This makes every intermediate level type-stable and avoids hidden per-level task
selection while the durable tree is introduced.

### Prompt Context

**User prompt (verbatim):** (same as Step 25)

**Assistant interpretation:** Start Slice 7 from canonical Go/JavaScript parity
before adding reduction state or dynamic partition nodes.

**Inferred user intent:** Make scale-in deterministic and bounded rather than
encoding aggregation behavior in runtime callbacks.

**Commit (code):** `e760069` — "workflowv3: freeze bounded reduction contracts"

### What I did

- Added reduction partition schema identity, `ReducePolicy`, `IRReduce`, and
  exact `PlanReduce`.
- Added `reduceKey` to symbolic values and reduction-output workflow refs.
- Added compiler validation for key conflicts, source sets, fan-in/level policy,
  exact task identity, one partition binding, one task output, homogeneous
  output schema, and named root output.
- Added safe `plan.reduce<I,O>` authoring, opaque partition handles, fluent
  `fanIn`/`maxLevels`, and exact TypeScript declarations.
- Added direct core validation plus exact authored IR/plan/DTS goldens and a
  callback-once assertion.

### Why

- Partition identity and tree materialization need a stable normalized template
  before tables or transactions are introduced.
- Homogeneous intermediate values let one exact reducer implementation operate
  at every level and keep root schema validation direct.

### What worked

- Core and authoring package tests pass.
- Focused isolated lint reports `0 issues`.
- `git diff --check` passes.
- The authored reduction plan pins `cpu.reduce`, fan-in 8, and four maximum
  levels with no retained JavaScript function.

### What didn't work

- N/A. The first normalized contract passed after formatting and golden review.

### What I learned

- Existing `IROutput` can represent a reduction root by adding an explicit
  `reduction-output` source; a separate output collection is unnecessary.
- A partition is a value ref, not a set ref: each dynamic reducer receives one
  bounded immutable partition artifact.

### What was tricky to build

- Reduction output validation must resolve the reducer root separately from
  static node outputs while preserving the existing output JSON shape.
- A reduction key shares identity space with static nodes and maps even though
  its dynamic partition nodes receive derived keys later.

### What warrants a second pair of eyes

- The initial homogeneous schema rule versus future explicitly typed
  heterogeneous final reducers.
- Default fan-in 16 and max-level 8 policy ownership before production profile
  binding.

### What should be done in the future

- Define strict compact partition manifests and exact level/ordinal/member
  identity.
- Add additive reduction/partition tables and materialize level zero from a
  published map manifest.

### Code review instructions

- Start with the bounded-reduction authoring test/goldens, then review compiler
  validation and the new types.
- Confirm the reducer callback executes once and only normalized bindings remain.

### Technical details

- Code commit: `e760069`.
- Partition schema: `scraper-workflow-reduction-partition/v1`.

## Step 33: Execute bounded deterministic reduction trees

Slice 7 now has a real multi-level data path. The engine consumes a published
map or direct set manifest, creates immutable partitions of at most the compiled
fan-in, materializes exact reducer nodes, rehydrates partition members only in
the lease workspace, and repeats until one root remains. The root ref is
published idempotently and gates run success.

A real JavaScript word-count workflow maps and reduces 257 documents through
three reducer levels with fan-in eight. It deliberately closes and reopens the
store after a level-zero partition succeeds, then reuses completed partitions
and finishes with exact attempt cardinality and a validated root.

### Prompt Context

**User prompt (verbatim):** (same as Step 25)

**Assistant interpretation:** Turn the frozen reduction contract into real
bounded dynamic nodes, member rehydration, recovery, publication, projections,
and adversarial integration evidence.

**Inferred user intent:** Prove scale-in is deterministic, restartable, bounded,
and isolated before worker-upgrade work begins.

**Commit (code):** `e2c48f2` — "workflowv3: execute bounded reduction trees"

### What I did

- Added strict `ReductionPartition` artifacts and exact node identity over
  reduction key, source digest, level, ordinal, ordered member keys/digests.
- Added additive reduction and partition tables with source, policy, level,
  partition, status, and root-ref state.
- Added direct-set and published-map source discovery.
- Added atomic idempotent level materialization, exact reducer nodes, bounded
  partition refs, level output reconstruction, max-level enforcement, and root
  publication.
- Added lease-time resolution of `reduction-partition` bindings.
- Extended task workspaces to materialize partition member refs under read-only
  local paths and exposed exact typed `members` metadata in `workflow/task` DTS.
- Added reduction cancellation, terminal failure propagation, empty-source
  typed failure, run-success gating, and snapshot root resolution.
- Added authoritative reduction-source/level projections.
- Added a real word-count bundle with map/reducer resources, exact authored
  goldens, and guarded malformed-shard failure.
- Added tests for a one-item identity root, capacity-1/capacity-4 root digest
  equality, empty input, two-connection level races, malformed-shard isolation,
  three levels, restart during reduction, exact attempts, and privacy.

### Why

- Reducers need member payloads immediately before invocation but control rows
  must remain ref-only. Lease-local rehydration preserves both constraints.
- A reduction level may be planned concurrently by multiple workers; exact
  existing partition digests make the second materialization idempotent rather
  than fatal.
- Root publication must be a separate durable boundary so a successful final
  reducer cannot make the run successful before the named output exists.

### What worked

- The 257-item normal fixture reaches 33 level-zero partitions, five level-one
  partitions, and one level-two root partition (296 total attempts including
  map children).
- The race profile uses 65 items through the same restart path.
- Capacity 1 and capacity 4 produce identical root digests.
- A malformed `__fail__` shard records
  `validation/REDUCTION_SHARD_INVALID` while an unrelated run succeeds.
- Two independent SQLite connections can materialize the same first level and
  converge on exactly two partitions.
- Focused core/authoring/runtime/store tests pass; runtime/store race suites
  pass; isolated lint reports zero issues; JS syntax and both DTS compilations
  pass.

### What didn't work

- The first focused lint found one style-only issue in root publication:

  `pkg/workflowv3sqlite/reduction.go:418:2: QF1003: could use tagged switch on status (staticcheck)`

  I replaced the `if/else if` chain with a status switch. Lint then reported
  zero issues.

### What I learned

- The same reducer can safely operate at every level when source and output item
  schemas are homogeneous.
- A single source item needs no reducer attempt; its existing typed ref is the
  deterministic identity root. Empty reduction has no general identity and
  fails with `REDUCTION_SOURCE_EMPTY` without a worker lease.
- Partition member paths belong in attempt-local metadata, not immutable
  partition artifacts or durable rows.

### What was tricky to build

- Higher-level partition identity must retain the original source-manifest
  digest while members become prior-level output refs.
- Level materialization writes external immutable partition artifacts before
  one SQLite transaction inserts all exact nodes/partition refs and advances
  the level cursor. Concurrent replay compares every existing digest.
- Reduction and map control actions share the dispatcher but neither consumes a
  worker lease until an actual JavaScript task node is ready.

### What warrants a second pair of eyes

- One transaction currently inserts all partitions for a level. At 1,807 items
  and fan-in eight this is bounded to 226 first-level rows, but much larger
  future sources may need paged level planning.
- Reduction dynamic ordinal ranges are deterministic but use reserved numeric
  ranges; node identity does not depend on ordinal.
- Direct single-item root publication trusts the engine-validated source
  manifest member while the store validates source and root schemas/identity
  state.

### What should be done in the future

- Run full repository validation and update public/design Slice 7 status.
- Add final migration/help/privacy audit and check the Slice 7 task.

### Code review instructions

- Start with `reduction_integration_test.go`, then follow `Engine.ReduceOne`,
  `MaterializeReductionLevel`, task member materialization, and
  `PublishReductionRoot`.
- Compare partition identity tests and exact fixture plan golden.
- Inspect terminal failure/cancellation updates with run-success predicates.

### Technical details

- Code commit: `e2c48f2`.
- Normal source items: 257; fan-in: 8; levels: 3; reducer partitions: 39.
- Race source items: 65.

## Step 34: Complete the Slice 7 validation audit

The bounded reduction slice now passes repository-wide validation and every
explicit dedicated-design acceptance boundary has concrete evidence. Public
help and architecture documents describe Slices 1–7 as implemented; Slice 8
remains the next target.

### Prompt Context

**User prompt (verbatim):** (same as Step 25)

**Assistant interpretation:** Audit Slice 7 against its design, run the complete
validation matrix, update public/design status, and close its task only if no
reduction behavior is deferred.

**Inferred user intent:** Preserve the strict slice-by-slice completion standard
before introducing rolling code upgrades.

### Outcome audit

| Slice 7 requirement | Fresh evidence |
|---|---|
| Typed reduction authoring | `reduce<I,O>`, opaque partition ref, fan-in/max-level builder, exact DTS and authored goldens |
| Bounded partition identity | Exact partition schema/key tests cover source digest, level, ordinal, ordered keys/digests, fan-in, order, and schema |
| Bounded runtime input | Reducer rejects zero or more than eight members; lease workspace exposes only validated read-only member paths |
| Deterministic tree | 257 items produce 33 → 5 → 1 partitions; capacity 1 and 4 produce identical root digest |
| Restart recovery | Store closes after a level-zero partition succeeds; reopen reuses completed outputs and reaches exact 296 attempts |
| Concurrent planning | Two independent SQLite connections materialize the same direct-set level idempotently with exactly two partitions |
| Partial failure/retry isolation | Malformed shard records `REDUCTION_SHARD_INVALID`; unrelated run succeeds; successful siblings remain immutable |
| Edge cardinality | One item publishes identity root without reducer; empty source fails without attempt |
| Root publication | Run success requires published root ref; snapshot reopens root and verifies direct word-count oracle |
| Cancellation/fencing | Cancel marks reduction/partitions; reducer nodes use existing lease token/cancel epoch and stale completion tests |
| Projections | Source-wait and level-wait reasons plus source/level/partition/root fields derive from authoritative rows |
| Privacy | Private document canary absent from root and SQLite/WAL/SHM; partition/member payloads stay in external artifacts/workspaces |
| Migration | Old minimal database opens with additive reduction and partition tables/columns |

### Verification audit

- `make validate` passed all Go tests, web unit tests, generation, Go binary
  build, TypeScript application build, and Vite production build.
- Full isolated Go lint reported `0 issues`.
- Runtime/store race suites passed.
- Both JavaScript fixture files pass `node --check`.
- Authoring/task DTS files compile with TypeScript and exact goldens pass.
- Core partition identities and real fixture IR/plan goldens pass.
- Public help renders `Workflow V3 Runtime Slices 1–7`.
- Logcopter generated check, placeholder scan, docmgr frontmatter/doctor, and
  `git diff --check` pass.

### What I did

- Updated public help, primary architecture, dedicated Slice 7 design, and the
  all-slice guide with implementation/evidence status.
- Repeated repository CI and every focused validation after the final
  single-item, empty-source, concurrency, and failure-isolation changes.
- Mapped the dedicated design and active goal clauses to the evidence above.

### What worked

- Final `make validate` passed; the runtime package completed in 37.401 seconds.
- Race suites completed in about ten seconds.
- No new build warning exists beyond the previously documented non-failing Vite
  chunk advisory.

### What didn't work

- N/A during the final audit. The earlier lint style finding and its fix are
  preserved in Step 33.

### What I learned

- Separate tests for large multi-level recovery, cross-connection planning,
  completion-order identity, edge cardinality, and malformed input provide
  clearer evidence than one oversized scenario attempting every fault.

### What was tricky to build

- The completion audit had to prove reducer input fan-in at both construction
  and execution. Core validation enforces partition size, and trusted
  JavaScript rejects an oversized lease-local member list.
- Root publication for one item and many levels uses one final durable ref path
  despite different execution histories.

### What warrants a second pair of eyes

- Level insertion is one bounded transaction for current 1,807-scale goals;
  introduce paged level insertion before materially larger sources.
- Reduction progress is currently part of queue snapshots for running runs;
  Slice 9 will provide richer historical/attach projections.

### What should be done in the future

- Begin Slice 8 immutable rolling registry generations from its dedicated
  candidate/activation/acquisition/draining/quarantine design.

### Code review instructions

- Review `e760069` and `e2c48f2`, then diary Steps 32–34.
- Reproduce the verification matrix above.
- Inspect three-level attempt cardinality and root oracle in the real integration
  test.

### Technical details

- Slice 7 implementation commits: `e760069`, `e2c48f2`.
- Normal tree: 257 map children + 39 reducer partitions = 296 attempts.
