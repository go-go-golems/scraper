---
Title: Investigation diary
Ticket: SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER
Status: active
Topics:
    - scraper
    - workflow-v3
    - architecture
    - cleanup
    - cli
    - onboarding
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/api/server/server_test.go
      Note: Legacy API acceptance follows explicit worker namespace (commit d9bcb68)
    - Path: repo://pkg/cmd/workflow_v3.go
      Note: Public Workflow V3 commands and worker (commit 3746f37)
    - Path: repo://pkg/cmd/workflow_v3_test.go
      Note: End-to-end command lifecycle proofs (commit 3746f37)
    - Path: repo://pkg/doc/topics/scraper-workflow-v3-product.md
      Note: Embedded operator guide (commit 3746f37)
    - Path: repo://pkg/taskpackages/cookbooklinear/package.go
      Note: Representative production JavaScript package (commit 3746f37)
    - Path: repo://pkg/workflowv3product/application.go
      Note: Production composition root and deterministic recovery (commit 3746f37)
    - Path: repo://pkg/workflowv3product/application_test.go
      Note: Restart failure cancellation and purity proofs (commit 3746f37)
    - Path: repo://pkg/workflowv3product/packages.go
      Note: Versioned task-package registry (commit 3746f37)
    - Path: repo://pkg/workflowv3product/service.go
      Note: Submission staging execution and read models (commit 3746f37)
    - Path: repo://pkg/workflowv3runtime/map_integration_test.go
      Note: Shared-host timeout stabilization without weakened assertion (commit d9bcb68)
    - Path: repo://ttmp/2026/07/22/SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER--make-workflow-v3-the-sole-public-scraper-workflow-product/analysis/01-product-cutover-acceptance-and-deletion-gates.md
      Note: Acceptance and deletion inventory
    - Path: repo://ttmp/2026/07/22/SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER--make-workflow-v3-the-sole-public-scraper-workflow-product/scripts/02-smoke-workflow-v3-product.sh
      Note: Reproducible cross-process smoke
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-22T19:25:19.11458645-04:00
WhatFor: ""
WhenToUse: ""
---



# Investigation diary

## Goal

Record the evidence and decisions used to create the intern-facing design for **SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER** and place it in the wider experiment-platform convergence program.

## Step 1: Establish the ticket and implementation guide

This step created the ticket as a focused child of the multi-repository convergence program. The guide explains the existing systems, proposed contracts, pseudocode, diagrams, implementation phases, tests, deletion rules, and neighboring tickets so an intern can begin without reconstructing the architecture from Git history.

No production code was changed. The architecture is proposed work; current files are referenced as evidence and implementation entry points.

### Prompt Context

**User prompt (verbatim):** "Create all the tickets with docmgr, then for each ticket, Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


That way we will have everything. Have all the tickets reference each other so that if someone stumbles upon one ticket, they'll know what this is all about."

**Assistant interpretation:** Create the umbrella and all directed tickets, write self-contained technical guides, cross-link them, validate docmgr state, and publish the complete set to reMarkable.

**Inferred user intent:** Preserve a coherent program plan before more ad-hoc implementation occurs, while making every workstream independently discoverable and approachable by a new engineer.

### What I did
- Created `SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER` with docmgr in the scraper repository.
- Added a long-term design/implementation guide and this diary.
- Mapped the ticket to all nine sibling/program tickets.
- Grounded the guide in current repository packages and public contracts.

### Why
- The convergence crosses repository boundaries and needs explicit ownership and sequencing.
- Standalone guides reduce the chance that an implementer recreates generic functionality in a workload package.

### What worked
- Existing greenfield contracts provided concrete API and file anchors.
- The ticket scope could be expressed as one independently testable capability: Workflow V3 public CLI, worker, extension model, and legacy deletion.

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
- Validate documentation with `docmgr doctor --ticket SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER --stale-after 30`.

### Technical details
- Program umbrella: `EXPERIMENT-PLATFORM-CONVERGENCE`.
- Ticket responsibility: Workflow V3 public CLI, worker, extension model, and legacy deletion.

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

## Step 4: Build the first production Workflow V3 vertical slice

This step turned Workflow V3 from a strong internal runtime into the primary command-level product. It added production dependency construction, task-package selection, pure authoring commands, durable submission, a long-running worker, operator inspection and cancellation, a read-only-by-default HTTP surface, and a representative two-task JavaScript package. The vertical slice deliberately reuses the existing compiler, SQLite store, dispatcher, artifact store, registry manager, and task runner rather than introducing a second lifecycle implementation.

The retained site engine remains available for named downstream migrations, but its worker moved under `scraper legacy worker run`. New `scraper worker run` invocations now mean Workflow V3. This is an intentional command cut, not a compatibility adapter.

### Prompt Context

**User prompt (verbatim):** "phase 2"

**Assistant interpretation:** Implement the second approved convergence phase, `SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER`, completely rather than stopping at another design pass.

**Inferred user intent:** Make Workflow V3 usable as Scraper's public workflow product, prove its durable lifecycle from the main binary, preserve evidence and downstream migration boundaries, and keep a detailed implementation record.

**Commit (code):** `3746f37bf3bc3cbc6d9809c6bb93c97e509dbc66` — `feat: productize Workflow V3 commands and workers`

### What I did
- Moved the linear JavaScript fixture into the production `pkg/taskpackages/cookbooklinear` package and updated its existing runtime/compiler tests.
- Added `pkg/workflowv3product` with validated configuration, deterministic package-set construction, exact registry/module wiring, authoring, input staging, submission, waiting, worker dispatch, list/show/cancel service methods, and stable HTTP read models.
- Added bounded SQLite run listing without exposing database rows directly as the product API.
- Added `workflow validate`, `explain`, `compile`, `submit`, `run`, `runs list/show/follow/cancel`, `workflow serve`, `worker run`, and `task-packages list`.
- Moved the old site worker to `legacy worker run` and updated devctl, tests, README, and embedded legacy documentation accordingly.
- Added bearer authorization for mutating HTTP cancellation; an absent token makes the operator API read-only.
- Added a runnable cookbook example and tests for process-restart recovery, typed failure, cancellation, strict inputs, deterministic package generations, pure authoring, HTTP views, and the full CLI lifecycle.

### Why
- The V3 runtime already owned the difficult correctness mechanisms; the missing layer was one production composition root and public operator surface.
- A versioned production task package is necessary to prove that authoring descriptors, pinned bundle bytes, runtime modules, and worker restart all agree outside test-only packages.
- The old worker command could not remain the default without leaving two claims on the canonical worker product name.

### What worked
- A submitted run survived closing the first SQLite connection, reopened in another application instance, executed two JavaScript tasks, and produced the expected immutable output.
- CLI tests validated, compiled, executed, listed, showed, followed, and canceled runs through newly constructed root commands.
- A separate `worker run` command recovered a previously submitted run and shut down cleanly after context cancellation.
- Focused command, product, compiler, SQLite, runtime restart, lease, cancellation, and lint checks passed after fixes.

### What didn't work
- The first focused compile failed with an import cycle because the new production task package imported `workflowv3runtime` to return module factories while same-package runtime tests imported that task package: `import cycle not allowed in test`. I changed the package contract to declare required module aliases and made the product composition root resolve host factories.
- The same attempt used nonexistent `IROutput.Schema` and `IRSetOutput.ManifestSchema` fields. The real schema is on `output.Value`; the explanation projection now reads those typed references.
- Moving every old command under one `legacy` group caused broad command tests to fail with `unknown command "site" for "scraper"`. That was wider than the phase's safe cut. I restored direct site/API/engine commands and moved only the conflicting worker command.
- Existing site tests then invoked the new worker with old flags and failed with `unknown flag: --sites-dir`. I updated only legacy-worker invocations to the explicit namespace.
- Workspace-mode `make lint` reproduced the known dependency skew: `undefined: goja.IsNumber`, `undefined: goja.IsBigInt`, and `undefined: goja.IsString`. `GOWORK=off make lint` used the module's pinned graph, then exposed seven real issues: unchecked closes/removal and one unused append. I fixed all seven; module-mode lint passed with zero issues.

### What I learned
- Task packages should declare required authority by alias; the application composition root, not a task package, selects concrete host module factories.
- The least disruptive hard command cut is the ambiguous worker name. Site, API, and engine removal must wait for their already named downstream replacements, but no new work should target them.
- Product read models can compose existing immutable snapshots and bounded operational projections without freezing SQLite schema as a public contract.

### What was tricky to build
- The worker dispatcher is intentionally endless, while `workflow run` must stop at one terminal run. The service starts the authoritative dispatcher under a child context, waits through product snapshots, then cancels and joins it; it does not duplicate leasing or retry loops.
- Package identity has to remain deterministic across processes. The package set seals exact bundles and module aliases, and the process-restart test reconstructs the same registry generation before claiming pending work.
- HTTP cancellation is host authority. The API therefore defaults to read-only and uses constant-time bearer comparison when an operator token is configured.

### What warrants a second pair of eyes
- Review `BuildPackageSet`, especially alias sharing and descriptor-name collision rejection.
- Review `RunUntilTerminal` cancellation/join behavior and the long-running dispatcher's completion-channel bounds.
- Review whether future task packages need additional capacity classes exposed by defaults rather than only operator-supplied `--capacity` entries.

### What should be done in the future
- Add explicit acceptance/deletion guard artifacts and reconcile the ticket guide with the implemented cut boundary.
- Run the complete repository, race, web, build, generated-code, and real multi-process smoke validation before closing the ticket.
- Delete the remaining old site/API/engine paths only after their named successor tickets pass; do not wrap them in V3.

### Code review instructions
- Start at `pkg/workflowv3product/application.go`, then `packages.go`, `service.go`, and `pkg/cmd/workflow_v3.go`.
- Trace the cookbook package from `examples/workflowv3/cookbook-linear/workflow.js` to `pkg/taskpackages/cookbooklinear/tasks.cjs`.
- Run `GOWORK=off go test ./pkg/workflowv3product ./pkg/cmd ./pkg/gojamodules/workflow ./pkg/workflowv3sqlite -count=1` and `GOWORK=off make lint`.

### Technical details
- Product commands use structured JSON; `runs follow` emits changed snapshots as NDJSON.
- Input paths resolve relative to their strict JSON manifest and are staged into the existing CAS.
- Default state is `state/workflow-v3.db` plus `state/workflow-v3-artifacts`; exact paths and capacities remain host flags.

## Step 5: Add executable cutover guards and complete full validation

This step converted the product/legacy boundary from prose into an executable acceptance contract. It added a local and optional cross-repository import guard, documented every retained cluster and successor gate, made terminal CLI failures return a nonzero status after printing evidence, stabilized full-suite validation under shared-host load, and ran the real binary across separate submit, worker, inspection, API, and cancellation processes.

The final evidence distinguishes “Workflow V3 is the primary generic product” from “all historical consumers have migrated.” New product code is clean of legacy imports; current old site and RAG imports are counted and block premature deletion until their owning tickets pass.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** Finish phase 2 with hard evidence, full repository validation, cross-repository safety, and an auditable deletion boundary.

**Inferred user intent:** Deliver a trustworthy product cutover that is immediately usable without breaking consumers scheduled for later phases.

**Commit (code):** `e54caa7d58654adbb26d2955b2a7cb1ae100c9b9` — `fix: fail command on terminal workflow failure`

**Commit (code):** `d9bcb68c0640c7f3288e5bbf8ba359803d9d96fb` — `test: stabilize Workflow V3 product validation`

### What I did
- Added a terminal failure test and changed `workflow run` to print the complete failed snapshot and then return an error.
- Added `scripts/01-check-product-cutover.sh`, which rejects legacy imports from the new product, verifies command ownership, runs focused acceptance, and counts local/downstream deletion blockers.
- Added the acceptance/deletion report with current local and RAG caller inventories and successor gates.
- Added a reproducible tmux-backed smoke script for separate submit/worker restart, terminal follow, operator API reads, unauthorized cancellation, and authorized cancellation.
- Ran RAG's current `internal/workflow` and `internal/preparationworkflow` tests against the workspace Scraper tree.
- Ran module tidiness, full Go tests, focused race tests, web tests/type-check/build, all binary builds, lint, logcopter generation checks, `make validate`, and the real smoke.

### Why
- A command that exits successfully for a failed workflow is ambiguous in automation even if its JSON says `failed`.
- “Do not delete yet” needs exact callers and named gates; otherwise it becomes undocumented coexistence.
- Full repository execution exposed timing assumptions that focused tests did not.

### What worked
- The cutover guard passed with 52 local legacy-importing files and 15 RAG files, while finding zero forbidden imports in the Workflow V3 product surface.
- RAG's focused downstream tests both passed.
- The real smoke produced plan digest `sha256:14bf6794a6b40e363e6d1a7a7d2ff680544f7ac71c1fbbd7a25e14ef5ecbb676`, recovered a submitted run in another worker process, completed two attempts, followed one terminal snapshot, rejected an unauthenticated HTTP cancel with 403, and accepted an authorized cancel.
- `make validate` passed the full Go suite, four web unit tests, TypeScript build, Vite production build, generated Go build, and all three binaries.
- Focused race tests, module-mode lint, and logcopter checks passed.

### What didn't work
- `go mod tidy` moved `golang.org/x/sys` from indirect to direct because the new signal-aware command imports `syscall` support through that module. The initial no-diff assertion correctly failed; I inspected and committed the explained classification change.
- The first full suite found one old API-server test still calling `worker run --sites-dir`; it failed with `unknown flag: --sites-dir`. I moved that invocation to `legacy worker run`.
- Under full package concurrency, the product typed-failure test's 100 ms lease expired while a Goja task was executing, adding a valid `lease_lost` attempt before the final typed failure. I changed the product fixture lease to five seconds; dedicated lease-expiry tests remain responsible for short-lease behavior.
- A subsequent full suite hit the existing lazy-map shared-host timeout at exactly 40 seconds: `Condition never satisfied`. The same test passed alone in 14.28 seconds. I retained the terminal assertion and increased only its wall-time allowance to 90 seconds (180 under race), with an explanatory comment. The next full suite and `make validate` passed.
- Plain workspace-mode `make lint` previously failed in Step 4 because workspace goja and goja-nodejs versions disagree. `GOWORK=off make lint` is the repository's module-authoritative validation and passed after real lint findings were fixed.
- A wildcard used before the ticket's `sources/` directory existed was not expanded and created a literal `SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER--*` directory. That also made three `docmgr doc relate` calls fail with `doc not found`. I moved the smoke summary into the real ticket directory, removed the literal directory, and reran every relation with exact paths successfully.

### What I learned
- Command-name cutovers must update tests outside `pkg/cmd`, including API integration tests that construct the root binary.
- Very short leases test lease loss, not ordinary product execution; fixture timeouts and workflow lease semantics should not be conflated.
- Cross-repository acceptance can prove both compatibility and incompleteness: green RAG tests preserve current behavior, while import counts prove deletion is not yet allowed.

### What was tricky to build
- The guard must reject exact `pkg/workflow` imports without falsely matching `pkg/workflowv3*`. Its regular expression requires either a slash or closing quote after the legacy package name.
- The smoke needed true process boundaries. It submits first, starts a worker in tmux against persisted paths, kills it after terminal state, then starts a separate API process and tests both authorization outcomes.

### What warrants a second pair of eyes
- Confirm the 52/15 deletion inventory matches the intended successor-ticket ownership.
- Review whether the V3 operator API should eventually share an authentication middleware with the broader Scraper API rather than its minimal bearer boundary.
- Review the lazy-map timeout increase as validation-host accommodation; the state assertion and fixture cardinality are unchanged.

### What should be done in the future
- `EXPERIMENT-PLATFORM-SCRAPER-RUNNER` should consume the V3 product service or command contract without importing SQLite internals.
- Site and RAG successor tickets should reduce the guard counts to zero, then perform focused package deletion rather than extending the legacy namespace.

### Code review instructions
- Run both ticket scripts with `RAG_EVAL_REPO` set, then inspect the acceptance report.
- Review commits `3746f37`, `e54caa7`, and `d9bcb68` in order.
- Run `make validate`, `GOWORK=off make lint`, and `make logcopter-check`.

### Technical details
- Smoke evidence: `sources/product-smoke/01-summary.json`.
- Deletion evidence: `analysis/01-product-cutover-acceptance-and-deletion-gates.md`.
- No runtime database or artifact payload was committed; the smoke script recreates all state in a temporary directory and cleans tmux sessions on exit.
