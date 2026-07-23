---
Title: Investigation diary
Ticket: SCRAPER-WORKFLOW-OBSERVATIONS
Status: active
Topics:
    - scraper
    - workflow-v3
    - observability
    - durability
    - privacy
    - api
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/researchrunner/runner.go
      Note: Researchctl canonical projection (commits e818676 and 8a47500)
    - Path: repo://pkg/workflowv3observations/project.go
      Note: Canonical pure projector and metric formulas (commit e818676)
    - Path: repo://pkg/workflowv3sqlite/observations.go
      Note: Stable privacy-safe source transaction (commit e818676)
    - Path: repo://ttmp/2026/07/22/SCRAPER-WORKFLOW-OBSERVATIONS--publish-canonical-workflow-v3-telemetry-and-experiment-observations/scripts/01-smoke-canonical-observations.sh
      Note: Cross-repository acceptance smoke
    - Path: ws://researchctl/examples/lab/scraper-workflow-plan.js
      Note: Multi-case replicate contract v2 fixture (commit fbc9be9)
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-22T19:25:19.621967166-04:00
WhatFor: ""
WhenToUse: ""
---


# Investigation diary

## Goal

Record the evidence and decisions used to create the intern-facing design for **SCRAPER-WORKFLOW-OBSERVATIONS** and place it in the wider experiment-platform convergence program.

## Step 1: Establish the ticket and implementation guide

This step created the ticket as a focused child of the multi-repository convergence program. The guide explains the existing systems, proposed contracts, pseudocode, diagrams, implementation phases, tests, deletion rules, and neighboring tickets so an intern can begin without reconstructing the architecture from Git history.

No production code was changed. The architecture is proposed work; current files are referenced as evidence and implementation entry points.

### Prompt Context

**User prompt (verbatim):** "Create all the tickets with docmgr, then for each ticket, Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


That way we will have everything. Have all the tickets reference each other so that if someone stumbles upon one ticket, they'll know what this is all about."

**Assistant interpretation:** Create the umbrella and all directed tickets, write self-contained technical guides, cross-link them, validate docmgr state, and publish the complete set to reMarkable.

**Inferred user intent:** Preserve a coherent program plan before more ad-hoc implementation occurs, while making every workstream independently discoverable and approachable by a new engineer.

### What I did
- Created `SCRAPER-WORKFLOW-OBSERVATIONS` with docmgr in the scraper repository.
- Added a long-term design/implementation guide and this diary.
- Mapped the ticket to all nine sibling/program tickets.
- Grounded the guide in current repository packages and public contracts.

### Why
- The convergence crosses repository boundaries and needs explicit ownership and sequencing.
- Standalone guides reduce the chance that an implementer recreates generic functionality in a workload package.

### What worked
- Existing greenfield contracts provided concrete API and file anchors.
- The ticket scope could be expressed as one independently testable capability: Canonical retry-aware workflow metrics and traces.

### What didn't work
- N/A. No implementation or runtime validation was attempted in this documentation-only step.

### What I learned
- The repositories are closer to the target internally than the prior TTC path suggested; the largest deficit is the canonical integration route.

### What was tricky to build
- The guides must be self-contained without duplicating ownership. This was handled by giving each ticket one primary capability, linking the complete program map, and making predecessor/successor relationships explicit.

### What warrants a second pair of eyes
- Public schema names, package deletion lists, and the exact boundary between RAG factor expansion and Researchctl scheduling should be reviewed before implementation.

### What should be done in the future
- Review and accept the design, then implement phases in dependency order with the ticket's acceptance tests.

### Code review instructions
- Begin with the ticket's `design-doc/` guide and its “Program context” section.
- Validate documentation with `docmgr doctor --ticket SCRAPER-WORKFLOW-OBSERVATIONS --stale-after 30`.

### Technical details
- Program umbrella: `EXPERIMENT-PLATFORM-CONVERGENCE`.
- Ticket responsibility: Canonical retry-aware workflow metrics and traces.

## Step 2: Rewrite the briefs as textbook-style intern guides

The first drafts were structurally correct but too compressed. They named responsibilities and implementation phases without spending enough time teaching the foundational concepts, tracing real code, or showing how an intern would validate each claim. This revision expanded every guide into a teaching document rather than leaving it as an architecture summary.

The revised guides now introduce the domain vocabulary before APIs, walk through current source packages, show concrete identity and failure traces, provide pseudocode and data structures, explain common mistakes, and end with onboarding exercises. The ticket-research workflow remains responsible for evidence, bookkeeping, validation, and reMarkable publication; the textbook-authoring style governs the prose.

### Prompt Context

**User prompt (verbatim):** "these guides are much too concise / terse. Read the skill on writing guides for interns, and update them."

**User prompt (verbatim):** "write in a textbook authoring style as well"

**User prompt (verbatim):** "it's ticket-research-docmgr-remarkable skill"

**Assistant interpretation:** Replace concise RFC-style briefs with long-form, foundational, intern-oriented teaching guides while continuing to follow the ticket research, docmgr, diary, validation, and publication workflow.

**Inferred user intent:** Ensure the ticket set can genuinely onboard implementers without relying on the prior conversation or requiring them to rediscover the systems.

### What I did
- Loaded the full-blown intern research/design and textbook-authoring guidance, while retaining the pinned ticket-research workflow as the delivery contract.
- Expanded conceptual foundations, code tours, worked traces, API references, implementation routes, debugging guidance, common mistakes, and onboarding checklists.
- Preserved program navigation in every ticket.

### Why
- A list of recommendations does not teach the invariants that an implementer must preserve.
- The program spans three repositories and several identity/lifecycle layers; foundational explanation is required before implementation advice.

### What worked
- Existing canonical contracts and integration tests provide concrete teaching anchors.
- Textbook-style worked flows make responsibility boundaries easier to evaluate than responsibility tables alone.

### What didn't work
- The initial drafts were approximately architecture-brief length and did not meet the requested exhaustive intern-guide standard.

### What I learned
- Each guide needs both a local implementation path and the complete cross-program context; either one alone is insufficient.

### What was tricky to build
- Expansion had to add depth without duplicating ownership or introducing analogies. The solution was to teach using exact identities, state transitions, source files, schemas, and failure windows.

### What warrants a second pair of eyes
- Review whether each guide provides enough real source-level orientation for an intern and whether proposed package names should be accepted before implementation.

### What should be done in the future
- Review the expanded guides, then implement in dependency order. Add experiments to each ticket as its implementation starts.

### Code review instructions
- Read the technology-primer and guided-source-tour sections before reviewing proposed APIs.
- Check that all bullet points are complete technical statements and diagrams preserve actual system boundaries.

### Technical details
- Style: foundational prose, concrete examples, no analogies, precise diagrams, worked traces, and executable validation routes.

## Step 3: Validate and publish the program guide bundle

The ticket passed docmgr validation and its guide was included in the ordered ten-document program bundle. The dry run confirmed document ordering and destination; the real upload completed successfully.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Complete ticket bookkeeping and deliver all guides as one navigable PDF.

**Inferred user intent:** Make the complete program available as a durable reading package rather than scattered repository files.

### What I did
- Ran `docmgr doctor --ticket <ticket> --stale-after 30`.
- Performed a reMarkable bundle dry run with all ten guides in dependency order.
- Uploaded `Scriptable Experiment Platform Intern Guides` to `/ai/2026/07/22/EXPERIMENT-PLATFORM-CONVERGENCE`.

### Why
- A single bundle provides a table of contents across repository boundaries.

### What worked
- All ticket doctor checks passed and the upload reported success.

### What didn't work
- N/A.

### What I learned
- Cross-repository tickets remain locally owned while one ordered publication can present the program coherently.

### What was tricky to build
- Document order needed to follow dependency order rather than repository or alphabetical order.

### What warrants a second pair of eyes
- Review the PDF at normal reMarkable size for code-block and diagram readability.

### What should be done in the future
- Re-upload with `--force` only after accepted content revisions, because overwrite removes annotations.

### Code review instructions
- Verify ticket frontmatter, program links, design guide, tasks, and changelog.

### Technical details
- Remote destination: `/ai/2026/07/22/EXPERIMENT-PLATFORM-CONVERGENCE`.

## Step 4: Implement the canonical projector and hard-cut runner v2

This step implemented the first complete production slice. The observation package now owns strict source and output contracts, pure interval and retry algorithms, exact coverage semantics, critical-path and failure traces, source/observation digests, and bounded output lineage. SQLite reads all sources in one read transaction. Product, CLI, HTTP, and Researchctl runner surfaces call the same projector.

The runner domain contract moved from `scraper-workflow-execution/v1` to v2 because requiring canonical observations is a breaking contract change. No v1 compatibility decoder remains. Researchctl's canonical JavaScript fixture was regenerated to v2.

### Prompt Context

**User prompt (verbatim):** "SCRAPER-WORKFLOW-OBSERVATIONS"

**Continuation prompt (verbatim excerpt):** "Continue working toward the active thread goal. ... Avoid repeating work that is already done. Choose the next concrete action toward the objective."

**Assistant interpretation:** Implement and fully validate the accepted observations ticket rather than stopping at its existing design guide.

**Inferred user intent:** Establish one retry-aware source of truth before RAG lowering and experiment analysis consume telemetry.

**Scraper commits:**
- `e8186762316220caa353894f01c46e36800f86db` — `feat: publish canonical Workflow V3 observations`
- `8a47500` — `feat: require observations in runner contract v2`

**Researchctl commits:**
- `6108609` — `test: require canonical Scraper observations`
- `fbc9be9` — `test: adopt Scraper execution contract v2`

### What I did
- Added `pkg/workflowv3observations` with a closed v1 schema, 22 metrics, 3 traces, exact ratios, source/observation digests, strict decode, and record bounds.
- Added pure half-open interval sum/union/peak/clipping and logical-node retry accounting.
- Added static dependency-weighted critical path, closed failure trace, artifact lineage, explicit queue/critical/operation/accounting coverage, cancellation and lease-loss evidence, and terminal classification.
- Added `Store.ObservationSnapshot` over one read-only SQLite transaction without payloads, capabilities, arbitrary messages, or artifact locators.
- Added Application, CLI, read-only HTTP, and Researchctl runner projection surfaces.
- Replaced the runner's ad-hoc attempt metrics/trace with canonical frames and a verified observation artifact.
- Added deterministic, permutation, historical 21.472-second failed-operation, zero-operation, map/reduce identity, lease-loss, restart, failure, cancellation, HTTP, CLI, strict-contract, privacy, and process projection tests.
- Built a ticket-local cross-repository smoke and committed its stable summary.

### Why
- Persisted aggregate counters would create a second authority and drift across crashes.
- Failed operations must participate in inclusive timing, while success-specific counts remain separately named.
- Missing scheduling and dynamic dependency boundaries must lower coverage rather than become guessed timestamps.
- Requiring a new field in v1 would violate versioning; a hard v2 cut keeps the contract honest.

### What worked
- Focused tests passed across observations, SQLite, product, runner, and CLI.
- Fresh-process reprojection matched all four Researchctl-custodied observation artifacts.
- The timeout database produced a deterministic canceled observation after restart.
- The smoke retained one internal retry and one failed external operation inside each selected Researchctl attempt.

### What didn't work
- The first runner test dereferenced `NumericProjection` for every metric. `workflow.terminal_status` is intentionally textual and has no numeric projection, causing a nil-pointer panic at `runner_test.go:110`. The test now records numeric values only when the pointer is present and selects the failure trace by kind rather than assuming trace order.
- The first hand-calculated queue-wait assertion expected 600,000 microseconds. It omitted the root node's 100,000-microsecond admission-to-first-start interval. Recalculation showed 100,000 initial + 100,000 retry + 500,000 dependency wait = 700,000; the fixture expectation was corrected.
- The first lint run failed with `nonamedreturns` for the interval helper and staticcheck `S1016` for a same-shape artifact conversion. The helper now uses local result variables and the lineage code uses a direct type conversion.
- The first adapted cross-repository smoke failed at the old three-artifact assertion because canonical observations correctly add a fourth artifact. After updating that custody expectation, it passed.
- The first reproducibility diff still varied because `researchAttemptCounts` reflected concurrent result ordering. Assertions already sorted the list, but the summary did not. The summary now writes the sorted list, making ticket evidence stable.

### What I learned
- Completion coverage, accounting coverage, and operation wall-time coverage answer different questions and require separate names.
- Queue eligibility is only exact for a subset of current durable records; explicit observed/total coverage is safer than deriving from provider gaps.
- Dynamic map and reduction node keys are sufficient for retry identity but not for dependency critical-path edges.
- Exact rational values should remain canonical even when Researchctl receives a floating projection for analysis.

### What was tricky to build
- The same projection must remain pure enough for deterministic tests while reading a relational source atomically and mapping into a process protocol that supports both raw values and numeric projections.
- Cancellation occurs before the runner can publish terminal frames, so acceptance reopens the durably canceled subordinate database and projects it in a fresh Scraper process.

### What warrants a second pair of eyes
- Review the v1 queue eligibility boundaries and static-only critical path against future gate/map/reduction telemetry requirements.
- Review whether the 100,000 source-record limits are appropriate for the first large RAG workload; they are explicit contract bounds, not silent truncation.
- Review rational metric handling in the next Researchctl analysis ticket so it does not discard exact numerator/denominator custody.

### What should be done in the future
- RAG packages should emit domain measurements as separate artifacts and operations; they must not add RAG fields to the canonical Workflow observation contract.
- If profiling justifies a cache, key it by derivation version and source digest rather than persisting mutable counters.

### Code review instructions
- Read `analysis/01-observation-contract-acceptance-and-coverage-audit.md` before reviewing formulas.
- Review pure interval/retry tests, then SQLite source custody, then runner mapping.
- Run the focused commands and ticket-local smoke listed in the acceptance report.

### Technical details
- Observation schema: `scraper-workflow-observations/v1`.
- Derivation: `workflow-observations/v1`.
- Runner domain: `scraper-workflow-execution/v2`.
- Privacy class: `bounded-identifiers-digests-integers`.
- Acceptance evidence: `sources/smoke/01-summary.json`.
