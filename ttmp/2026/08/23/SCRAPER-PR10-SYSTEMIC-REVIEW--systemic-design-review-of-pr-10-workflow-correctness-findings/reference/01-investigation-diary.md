---
Title: Investigation diary
Ticket: SCRAPER-PR10-SYSTEMIC-REVIEW
Status: complete
Topics:
    - scraper
    - workflow-v3
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://.github/workflows/push.yml
      Note: Bubblewrap installation for isolation integration CI (commit dc24fb4)
    - Path: repo://cmd/scraper-workflow-runner/main.go
      Note: Host-level resolved-input byte limit flag (commit 5486f2e)
    - Path: repo://pkg/gojamodules/workflow/authoring_test.go
      Note: DSL option validation and canonical golden coverage (commit 2dfdee1)
    - Path: repo://pkg/researchrunner/runner_test.go
      Note: Custody replacement and resolved-input bound regression tests (commit 5486f2e)
    - Path: repo://pkg/workflowv3/compiler_test.go
      Note: Set policy, consumer capacity, reduction bound, and pass-through tests (commit 2dfdee1)
    - Path: repo://pkg/workflowv3product/application_test.go
      Note: Immediate persisted submission status and scalar pass-through test (commit 7f6e728)
    - Path: repo://pkg/workflowv3sqlite/readiness.go
      Note: Shared static and dynamic readiness lowering from second review (commit e0a3439)
    - Path: repo://pkg/workflowv3sqlite/store_test.go
      Note: Direct set-input output projection regression test (commit 2dfdee1)
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/design-doc/01-durable-dataflow-workflow-v3-and-modern-scripting-architecture.md
      Note: Original immutable-reference, boundedness, dependency, and durability requirements
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/design-doc/08-workflow-v3-slices-1-through-12-intern-architecture-and-analysis-guide.md
      Note: Implemented slice behavior and intern-oriented execution model
    - Path: repo://ttmp/2026/07/22/SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER--make-workflow-v3-the-sole-public-scraper-workflow-product/reference/02-researchctl-runner-implementation-diary.md
      Note: Original Researchctl runner intent, verification evidence, and boundary decisions
ExternalSources:
    - https://github.com/go-go-golems/scraper/pull/10
Summary: Chronological evidence record for the PR 10 systemic workflow correctness review.
LastUpdated: 2026-08-23T19:45:00-04:00
WhatFor: Resume or review the evidence collection, architectural diagnosis, ticket writing, validation, and reMarkable delivery.
WhenToUse: Read before implementing or reviewing the fixes proposed by SCRAPER-PR10-SYSTEMIC-REVIEW.
---






# Diary

## Goal

Record how the four PR 10 review comments were collected, validated against source and history, classified as local defects versus systemic invariant gaps, and turned into an intern-oriented implementation design.

## Step 1: Capture the PR review and establish scope

This step created a dedicated ticket before drawing conclusions and captured every GitHub review thread directly through the API. The PR contains one automated review with four inline comments: three P1 findings and one P2 finding, all on the reviewed commit `202229464629e2b6d0e193ff7798b16770b3a270`.

The work was deliberately scoped as an analysis and design review rather than an immediate patch. The ticket tasks require architecture mapping, local-versus-systemic classification, API design, testing strategy, validation, and reMarkable delivery.

### Prompt Context

**User prompt (verbatim):**

```text
Look at code review comments on https://github.com/go-go-golems/scraper/pull/10 and take a step back and see how this can be addressed, is it just localized fixes, or could a more systemic approach help out / better design ? 

Create a new docmgr ticket, and Create  a detailed analysis / design / code review / implementation review that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.

[REMINDER] Output a <summary>...</summary> block at the VERY END of your response. This is mandatory.
```

**Assistant interpretation:** Read all PR 10 review threads, investigate the relevant architecture and history, decide whether narrow fixes or a systemic design is appropriate, then create a complete docmgr research package and deliver it to reMarkable.

**Inferred user intent:** Avoid mechanically fixing four comments in ways that preserve deeper design inconsistencies; leave a durable, technically precise implementation guide that a new intern can follow safely.

### What I did

- Queried PR metadata with `gh pr view 10 --repo go-go-golems/scraper --json ...`.
- Queried inline comments with `gh api --paginate repos/go-go-golems/scraper/pulls/10/comments`.
- Queried review bodies with `gh api repos/go-go-golems/scraper/pulls/10/reviews`.
- Confirmed four inline comments and retained their discussion URLs.
- Created ticket `SCRAPER-PR10-SYSTEMIC-REVIEW` with topics `scraper`, `workflow-v3`, and `architecture`.
- Added the primary design document, this diary, and nine explicit tasks.

### Why

- Inline comments are not included in the ordinary PR issue-comment list, so the pull review-comment endpoint is the authoritative source.
- A separate ticket prevents review rationale from being lost in a transient PR thread.
- Explicit tasks make validation and delivery auditable.

### What worked

- GitHub returned all four comments, file locations, reviewed SHA, severity, body, and stable URLs.
- `docmgr` created a complete ticket workspace with index, tasks, changelog, design-doc, reference, scripts, and supporting directories.

### What didn't work

- `gh pr view` reported one review but zero top-level comments; this was not an error, but it did not include the four inline threads. The exact command was:

```bash
gh pr view 10 --repo go-go-golems/scraper \
  --json number,title,state,author,baseRefName,headRefName,url,reviewDecision,comments,reviews,commits,files
```

- The resolution was to use `gh api repos/go-go-golems/scraper/pulls/10/comments` for inline review comments.

### What I learned

- The review is focused and high-signal: byte custody, cardinality admission, dependency readiness, and terminal lifecycle.
- All findings concern boundaries between otherwise well-developed subsystems rather than missing core features.

### What was tricky to build

- The PR has two kinds of comment surfaces. Review metadata alone proves a review occurred but does not expose inline findings; the REST pull-comments endpoint was necessary.
- The review SHA is the fork's closure commit, while the local branch now has a merge commit. Source equivalence had to be checked rather than assumed.

### What warrants a second pair of eyes

- Verify that no later hidden/dismissed review thread exists before implementation starts.
- Confirm compatibility expectations for the still-open PR before changing canonical IR/plan schemas.

### What should be done in the future

- Re-fetch review threads immediately before implementation in case new comments are added.

### Code review instructions

- Start with the four URLs in the design document's References section.
- Confirm the reviewed SHA with `git show -s 2022294`.
- Re-run the two `gh api` commands above to refresh evidence.

### Technical details

- PR: `go-go-golems/scraper#10`, “Introduce Workflow V3 and remove legacy engine.”
- Base/head: `main` <- `task/benchmark-cpu-inference`.
- Reviewed commit: `202229464629e2b6d0e193ff7798b16770b3a270`.
- Review: `pullrequestreview-4772468539`.
- Findings: `discussion_r3644764557`, `r3644764562`, `r3644764564`, `r3644764567`.

## Step 2: Trace each failure across runner, compiler, store, and runtime

This step mapped each review comment through the entire execution path rather than stopping at the commented line. The source confirms all four findings. It also reveals adjacent mismatches: direct set-input outputs compile but cannot be resolved by `Snapshot`; run completion SQL is duplicated in four files; dynamic map/reduction nodes infer node-output dependencies while static nodes do not; and request-size configuration does not bound resolved input files.

History and prior ticket documents establish the intended contracts. Workflow V3 was explicitly designed around immutable artifact references, hard limits at every boundary, acyclic compiled dependencies, and durable readiness. The review findings are therefore incomplete implementation of accepted principles, not a request to change those principles.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Determine root causes and produce an implementation design that is safe across all supported workflow topologies.

**Inferred user intent:** Give an intern enough system context to implement the fix without regressing durability, determinism, concurrency, privacy, or compatibility.

### What I did

- Read `pkg/researchrunner/runner.go`, `types.go`, and runner tests.
- Traced `StagedInput` into `pkg/workflowv3product/service.go` and the content-addressed store in `pkg/workflowv3/artifacts.go`.
- Read IR/plan types, compiler validation, plan compilation, and cycle detection.
- Read JavaScript authoring APIs and TypeScript declarations for task, map, reduce, gate, `after`, and outputs.
- Traced run creation, dependency persistence, lease admission, input resolution, dynamic map/reduction materialization, task execution, and terminal updates.
- Compared reviewed files at `2022294` with local `HEAD`; the core review files are unchanged.
- Read commit history/blame and existing Workflow V3 architecture, intern, cutover, and Researchctl runner diary documents.
- Inspected current tests to identify covered and missing topology cases.

### Why

- A local code line can be correct only in context of the next consumer. The scalar-input defect spans runner verification and product staging; the dependency defect spans authoring, compiler, persistence, scheduler, and failure classification.
- Existing history distinguishes deliberate contracts from accidental behavior.

### What worked

- `git diff --quiet 2022294 HEAD -- pkg/researchrunner/runner.go pkg/workflowv3/compiler.go pkg/workflowv3sqlite/store.go` returned exit code `0`; the review applies unchanged.
- Blame tied the runner scalar/set behavior to `842303c`, core dependency validation to `601c0a1`, and run creation status to `02dc422`.
- Existing dynamic map/reduction code provided a useful partial pattern: it already derives node-output dependencies while materializing children.
- Existing architecture docs explicitly state that large values cross persistence as bounded `ArtifactRef`s and that compiled dependencies must be acyclic.

### What didn't work

- A first attempt to validate pre-merge build behavior in a temporary worktree failed because that path was outside the parent `go.work` module list:

```text
pattern ./...: directory prefix . does not contain modules listed in go.work or their selected dependencies
```

- This did not affect the review analysis. Earlier tree-object equality already proved the merge result and pre-merge content were byte-identical, and the current task is source/design analysis rather than merge validation.

### What I learned

- The four findings share one design smell: correctness facts have multiple owners.
- The best systemic response is not one mega-abstraction. It is four authoritative boundaries guided by one rule: declare each invariant at the earliest layer with enough information, then lower it once.
- The compiler accepts more topology than runtime/output layers support; a table-driven topology matrix is needed across compile, submit, terminal, snapshot, and reopen.

### What was tricky to build

- Data dependencies have heterogeneous producers. Static node dependencies use `v3_dependencies`; reduction and gate readiness use separate consumer tables; map-source readiness lives in expansion state; dynamic children are created later. A systemic graph must be typed without forcing an immediate storage-schema rewrite.
- Set bounds have two dimensions that must not be conflated: semantic item cardinality and host byte limits.
- Run success is derived but persisted. The reconciler must be idempotent and transaction-local so concurrent final transitions emit one terminal change.

### What warrants a second pair of eyes

- Review the proposal to amend V3 versus bump plan/IR schema; this depends on external consumption of the still-open PR.
- Review reduction capacity `FanIn^MaxLevels` against exact runtime level semantics and use saturating arithmetic.
- Review whether impossible post-lease input resolution should fail or quarantine a run; it must not be classified as ordinary task failure.
- Review event-schema implications of adding a durable `run.succeeded` event.

### What should be done in the future

- Before coding, accept or revise the four decision records in the primary design doc.
- Implement in custody, set-policy, graph, and reconciliation phases; do not mix all changes into one commit.

### Code review instructions

- Follow the reading order in Section 13 of the design doc.
- Compare compiler-supported shapes with `Store.Snapshot` output source cases.
- Search `UPDATE v3_runs SET status = 'succeeded'` to find all current terminal predicate copies.
- Search `binding.Source == "node-output"` in static and dynamic persistence paths.

### Technical details

Key commands:

```bash
rg -n "readVerifiedInput|stageInputs|RUNNER_SET_INPUT_LIMIT" pkg/researchrunner pkg/workflowv3product
rg -n "node-output|DependsOn|validateAcyclic" pkg/workflowv3 pkg/gojamodules/workflow
rg -n "UPDATE v3_runs SET status = 'succeeded'" pkg/workflowv3sqlite
rg -n "unsupported set output source" pkg/workflowv3sqlite
```

Primary conclusion:

```text
verified bytes -> immutable ArtifactRef
set input      -> explicit ingress cardinality policy
bindings       -> compiler-derived typed dependency graph
work state     -> one transaction-local terminal reconciler
```

## Step 3: Validate and deliver the review package

This step completed ticket hygiene and delivered the analysis as one reMarkable PDF with a depth-two table of contents. Validation covered docmgr metadata, absolute related-file links, Markdown fence balance, and Git whitespace checks before rendering.

The bundle includes the ticket index, 1,145-line primary design, this chronological diary, tasks, and changelog. The ticket remains in `review` because it proposes implementation decisions; it does not claim that the code findings have already been fixed.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Store a validated, navigable research package in the ticket and deliver the same package to reMarkable.

**Inferred user intent:** Make the review usable away from the terminal and durable enough for intern onboarding and implementation review.

### What I did

- Ran `docmgr doctor --ticket SCRAPER-PR10-SYSTEMIC-REVIEW --stale-after 30`.
- Ran `git diff --check` for the ticket workspace.
- Checked all Markdown files for balanced code fences.
- Related seven primary implementation files to the design and three historical architecture documents to the diary.
- Updated tasks and changelog.
- Ran a reMarkable bundle dry-run with all five ticket documents.
- Uploaded `SCRAPER PR10 Systemic Correctness Review.pdf` to `/ai/2026/08/23/SCRAPER-PR10-SYSTEMIC-REVIEW`.

### Why

- A technically correct document is still difficult to maintain if metadata, links, or task state are inconsistent.
- Bundle upload preserves one reading order and table of contents instead of scattering separate PDFs.

### What worked

- `docmgr doctor` reported `✅ All checks passed`.
- Markdown and whitespace checks produced no errors.
- Dry-run listed every intended source file and destination.
- Upload returned:

```text
OK: uploaded SCRAPER PR10 Systemic Correctness Review.pdf -> /ai/2026/08/23/SCRAPER-PR10-SYSTEMIC-REVIEW
```

### What didn't work

- N/A. Rendering and upload succeeded without reauthentication.

### What I learned

- The primary design is long enough to serve as an intern implementation guide but remains navigable through stable sections, a topology matrix, decisions, phases, tests, and references.
- Ticket status should remain `review` until compatibility decisions are accepted and implementation begins.

### What was tricky to build

- Delivery evidence is generated only after upload, so the diary must be updated and the just-created PDF refreshed once. This is safe before anyone can reasonably annotate the first upload, but future workflows should prepare a delivery placeholder before the initial render to avoid replacement.

### What warrants a second pair of eyes

- Confirm the four proposed decision records before implementation.
- Confirm whether the open PR permits amending Workflow V3 schema contracts in place.

### What should be done in the future

- When implementation starts, append new diary steps rather than rewriting this analysis history.
- Re-upload only a newly named revision if annotations may already exist on the current PDF.

### Code review instructions

- Start with the ticket index and Executive summary.
- Review Sections 4, 6, 9, 11, and 12 of the primary design.
- Run `docmgr doctor --ticket SCRAPER-PR10-SYSTEMIC-REVIEW --stale-after 30` after future edits.

### Technical details

- Ticket: `SCRAPER-PR10-SYSTEMIC-REVIEW`.
- Ticket path: `ttmp/2026/08/23/SCRAPER-PR10-SYSTEMIC-REVIEW--systemic-design-review-of-pr-10-workflow-correctness-findings`.
- reMarkable directory: `/ai/2026/08/23/SCRAPER-PR10-SYSTEMIC-REVIEW`.
- Bundle name: `SCRAPER PR10 Systemic Correctness Review.pdf`.

## Step 4: Accept the design and prepare implementation slices

This step transitions the ticket from review into active implementation. The accepted direction is to preserve current package boundaries while giving custody, cardinality, readiness, and successful terminalization one authoritative owner each; code will be committed in focused slices with diary and changelog updates between them.

No production behavior changed in this step. It records the implementation request, creates explicit phase tasks, and establishes a clean documentation baseline before source edits begin.

### Prompt Context

**User prompt (verbatim):**

```text
implement. commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill)

[REMINDER] Output a <summary>...</summary> block at the VERY END of your response. This is mandatory.
```

**Assistant interpretation:** Implement the accepted systemic design now, use focused commits at natural phase boundaries, and append strict diary entries containing commands, failures, decisions, review guidance, and follow-ups.

**Inferred user intent:** Turn the review into production-quality fixes without losing the reasoning trail or combining unrelated correctness changes into one opaque commit.

### What I did

- Re-read the complete ticket diary before resuming.
- Changed ticket documents from `review` to `active`.
- Added six implementation tasks for custody, set policy, dependency graph, lifecycle reconciliation, validation, and final documentation audit.
- Prepared to commit the completed research/design package as the implementation baseline.

### Why

- The research package was previously untracked; committing it first separates accepted design evidence from subsequent code changes.
- Focused tasks align implementation commits with the design's four phases.

### What worked

- The worktree contained only the new ticket directory beyond the completed merge commit.
- Docmgr accepted the new tasks with stable IDs.

### What didn't work

- N/A in implementation setup.

### What I learned

- The user explicitly accepted implementation, so the ticket should no longer remain in review-only status.
- The existing analysis gives enough file-level guidance to proceed without another planning-only phase.

### What was tricky to build

- The ticket's original nine tasks were complete because they represented research delivery, not code delivery. New implementation tasks were added rather than reopening or repurposing historical tasks, preserving accurate chronology.

### What warrants a second pair of eyes

- Confirm each code commit remains limited to one invariant boundary and its tests.
- Review schema/digest changes carefully in the set-policy and graph phases.

### What should be done in the future

- Append one diary step after each code slice and include its exact commit hash.

### Code review instructions

- Review the baseline ticket commit before code commits.
- Follow subsequent commits in custody, set-policy, dependency, lifecycle, and validation order.

### Technical details

Implementation task IDs:

```text
dlgs  verified-input custody
hc7q  set-input policy
zf9m  dependency graph
a09k  lifecycle reconciliation
e9or  focused/full/race validation
nwdm  final diary and ticket audit
```

## Step 5: Preserve verified input custody through immutable artifact references

This slice closes the Researchctl scalar-input TOCTOU window. Resolved files are now read through a byte ceiling, verified once, immediately staged into the content-addressed artifact store, and submitted as immutable `ArtifactRef`s; the runner no longer passes a verified pathname to product staging for a second read.

The product facade now exposes `SubmitArtifacts` for callers that already own immutable refs, while existing CLI path staging remains explicit through `Submit`. A dedicated `--max-resolved-input-bytes` host flag separates external artifact size from protocol-frame and export limits.

### Prompt Context

**User prompt (verbatim):** (see Step 4)

**Assistant interpretation:** Implement the first accepted invariant boundary as a focused, tested commit and record exact evidence.

**Inferred user intent:** Ensure the workflow executes exactly the bytes committed by Researchctl identity, even if a source pathname is replaced concurrently.

**Commit (code):** `5486f2e678abda499b1887cab0a2706b5a0b3259` — `fix: preserve verified workflow input custody`

### What I did

- Added `Config.MaxResolvedInputBytes` with a 32 MiB default and CLI flag.
- Changed `readVerifiedInput` to reject invalid/oversized declared sizes before reading and to use `io.LimitReader(max+1)`.
- Changed runner input resolution to return `map[string]workflowv3.ArtifactRef`.
- Staged scalar verified bodies directly with `ArtifactStore.Put` and checked staged digest/size against the Researchctl reference.
- Added `Application.SubmitArtifacts` and routed the runner through it.
- Added tests proving a replaced source path cannot change staged bytes and proving resolved-input limit enforcement.

### Why

- A digest check is meaningful only if the checked bytes remain the bytes executed.
- Protocol request bytes, resolved artifact bytes, and exported artifact bytes are separate operational boundaries.

### What worked

- Focused tests passed:

```text
ok github.com/go-go-golems/scraper/pkg/researchrunner
ok github.com/go-go-golems/scraper/pkg/workflowv3product
ok github.com/go-go-golems/scraper/cmd/scraper-workflow-runner
```

- `go test ./pkg/cmd ./cmd/scraper -count=1` passed.
- The pre-commit hook ran `GOWORK=off go test ./... -count=1` and golangci-lint with `0 issues`; the full suite passed, including the 74.528-second runtime package.

### What didn't work

- The first `edit` call attempted several small replacements in `resolveInputs`; one old text fragment appeared twice and the tool rejected the complete call with:

```text
Found 2 occurrences of edits[10] in pkg/researchrunner/runner.go. Each oldText must be unique.
```

- No partial edit was applied. I replaced the entire `resolveInputs` and `readVerifiedInput` blocks in one exact edit, then formatted and tested.

### What I learned

- The artifact store already provides the required atomic content-addressed custody boundary; no new storage mechanism was necessary.
- The runner's set-input path already used verified bytes, so scalar and set handling now converge on the same immutable-ref submission contract.

### What was tricky to build

- A declared size check alone does not prevent a file from growing before `os.ReadFile`; bounded streaming with `LimitReader(max+1)` is required to cap allocation while still detecting replacement/truncation through final size and digest checks.
- Product artifact capacity must cover both resolved input and export ceilings, so runner setup uses the larger limit.

### What warrants a second pair of eyes

- Review the 32 MiB resolved-input default against realistic Researchctl workloads.
- Review whether `SubmitArtifacts` should eventually replace the overloaded staged-input API entirely; this slice keeps explicit CLI path convenience without changing callers unnecessarily.

### What should be done in the future

- Consider a streaming `ArtifactStore.PutReader` only if measured input sizes justify it; it must preserve bounded hashing and atomic publication.

### Code review instructions

- Start at `pkg/researchrunner/runner.go:36-67`, then follow `resolveInputs` and `readVerifiedInput`.
- Review `pkg/workflowv3product/service.go:101-145` for the immutable-ref boundary.
- Run:

```bash
go test ./pkg/researchrunner ./pkg/workflowv3product ./cmd/scraper-workflow-runner -count=1
```

### Technical details

Custody invariant after this commit:

```text
Researchctl digest
  == sha256(single bounded read)
  == ArtifactRef digest returned by Put
  == v3_run_inputs digest
```

## Step 6: Compile explicit bounded set-input contracts

This slice moves set cardinality from runner inference into the canonical workflow contract. Every `inputSet` now declares positive `maxItems`; the compiler verifies map and reduction consumers can support that ingress contract, and the runner enforces the input policy without inspecting incidental map topology.

The slice also closes the adjacent runtime mismatch: a direct set-input pass-through output can now be resolved from `v3_run_inputs`. Existing V3 IR/plan fixtures were amended in place because Workflow V3 is still introduced by the open PR and has not crossed a released compatibility boundary.

### Prompt Context

**User prompt (verbatim):** (see Step 4)

**Assistant interpretation:** Implement the second accepted invariant boundary, including DSL/API propagation, compiler compatibility checks, runner admission, output resolution, and canonical fixtures.

**Inferred user intent:** Make reduction-only and pass-through workflows valid without weakening boundedness or hiding limits in consumer-specific logic.

**Commit (code):** `2dfdee1dbc47718d44d5b627791a1448a170979a` — `feat: compile bounded set input contracts`

### What I did

- Added `SetInputPolicy{MaxItems}` to `IRSetInput`; plans inherit the policy canonically.
- Required positive `maxItems` in JavaScript `inputSet` and TypeScript declarations.
- Added compiler checks that map capacity covers source contracts.
- Added saturating `ReductionCapacity(FanIn, MaxLevels)` and rejected incompatible reduction sources.
- Changed runner set admission to enforce `input.Policy.MaxItems` directly.
- Added direct `set-input` output resolution to `Store.Snapshot`.
- Updated executable fixtures, TypeScript output, eight canonical IR/plan JSON files, and embedded help.
- Added tests for missing bounds, undersized map consumers, reduction capacity, bounded pass-through compilation, direct set output snapshots, and runner policy admission without maps.

### Why

- External admission limits belong to the input contract, not whichever consumer happens to be scanned first.
- Compiler and runtime must support the same topology matrix.

### What worked

- Focused workflow, authoring, runner, SQLite, and runtime tests passed.
- The pre-commit hook passed full `GOWORK=off go test ./... -count=1` and lint with zero issues.
- The compiler correctly reports `capacity 4096 is smaller than source contract 4097` for fan-in 8 and four levels.

### What didn't work

- The first focused run failed as expected because canonical goldens did not yet contain the new policy. The failure showed expected JSON without `policy` and actual JSON with `"policy":{"maxItems":...}` for all four set-based fixtures. I regenerated them with:

```bash
UPDATE_GOLDEN=1 go test ./pkg/gojamodules/workflow -count=1
```

- A new missing-`maxItems` authoring test initially triggered a Go nil-pointer panic because `options.Get("maxItems")` can return `nil`; calling `ToInteger()` directly was unsafe. The stack pointed to `authoring.go:249`. I changed decoding to check `nil`, `undefined`, and `null` before conversion, producing the intended closed TypeError.

### What I learned

- Goja option access must not assume a missing property is represented by a non-nil undefined value.
- Reduction geometry gives a deterministic input-capacity contract: `FanIn^MaxLevels`, computed with saturation to avoid integer overflow.
- Direct set pass-through was already accepted by the compiler; output projection was the missing half.

### What was tricky to build

- Map limits are execution limits while set limits are ingress guarantees. The compiler must require each consumer to cover the source guarantee, including chained map outputs.
- Canonical policy fields change IR/plan digests. Amending V3 in place is appropriate only because the product is still under the open introduction PR; a released contract would require a schema bump.

### What warrants a second pair of eyes

- Confirm the choice to amend unreleased V3 instead of introducing V4.
- Review whether author-provided `maxItems` also needs a host-owned global ceiling in a later hardening slice.
- Review the semantic choice that empty reductions remain invalid while empty pass-through sets are valid.

### What should be done in the future

- Add a host-level maximum set cardinality if deployments need to constrain author contracts independently of byte limits.

### Code review instructions

- Start with `pkg/workflowv3/types.go` and `compiler.go` policy validation.
- Review `pkg/gojamodules/workflow/authoring.go` missing-option handling and declaration parity.
- Review `pkg/workflowv3sqlite/store.go` direct set-output branch.
- Run:

```bash
go test ./pkg/workflowv3 ./pkg/gojamodules/workflow ./pkg/researchrunner ./pkg/workflowv3sqlite ./pkg/workflowv3runtime -count=1
```

### Technical details

Compatibility rules:

```text
set-input.maxItems <= direct-map.maxItems
map-output.maxItems <= chained-map.maxItems
set-or-map maxItems <= reduction FanIn^MaxLevels
runner archive item count <= set-input.maxItems
```

## Step 7: Derive readiness from data bindings and validate the complete graph

This slice removes the requirement that authors repeat a `node-output` binding with `.after(...)`. Compilation now emits the canonical union of data-derived node producers and explicit control-only dependencies, while one typed acyclicity analysis covers nodes, maps, reductions, gates, source chains, and budget gates.

The same dependency helper now lowers static and dynamically materialized work. `CreateRun` validates a decoded plan's dependency structure in addition to its digest, preventing a cross-process caller from supplying a digest-valid plan that omits a producer edge.

### Prompt Context

**User prompt (verbatim):** (see Step 4)

**Assistant interpretation:** Implement the third accepted invariant boundary so dataflow and scheduling cannot disagree, including externally supplied plans and dynamic workers.

**Inferred user intent:** Prevent premature leasing and deadlocks without forcing workflow authors to manually synchronize schedule annotations with value references.

**Commit (code):** `b0cdd1b7f69bceefb6afd9acede2be9437d22b08` — `fix: derive workflow readiness from data bindings`

### What I did

- Added `pkg/workflowv3/dependencies.go` with canonical `EffectiveNodeDependencies`.
- Replaced separate node-only and node/gate cycle checks with one deterministic typed graph.
- Included map source, reduction source, value-binding, gate, control, and budget-gate edges.
- Compiled effective static `PlanNode.DependsOn` from bindings plus control edges.
- Added `ValidatePlanDependencies` for digest-valid cross-process plans.
- Reused the same helper when materializing map children and reduction partition nodes.
- Updated store dependency persistence defensively to derive the union again.
- Clarified in embedded documentation that `.after` is control-only.
- Added tests for inferred blocking, cross-kind cycles, and a digest-valid plan missing its derived edge.

### Why

- A data binding already identifies its producer and must be authoritative for readiness.
- Plan digest validation proves bytes are unchanged; it does not prove those bytes encode a valid scheduling contract.
- Dynamic and static work should not have separate dependency semantics.

### What worked

- The existing store fixture now omits explicit `DependsOn`; a second lease attempt returns nil until the producer completes.
- A node/reduction cycle is rejected before persistence with `dependency cycle`.
- A recomputed, digest-valid plan with the inferred dependency removed is rejected as `dependencies are not canonical`.
- Ten repeated focused runs of dependency compiler/store tests passed.
- Full pre-commit tests and lint passed; runtime integration took 80.424 seconds.

### What didn't work

- N/A. The first implementation compiled and passed focused tests after formatting.

### What I learned

- Existing SQLite tables are adequate lowering targets; semantic unification did not require a storage migration.
- Cross-process plan validation is necessary even when plans usually originate from the local compiler.
- Sorting and deduplication in the shared helper preserves deterministic plan digests when a data edge is also explicitly listed as a control dependency.

### What was tricky to build

- The graph spans static and template work kinds. A map/reduction template may depend on static node, gate, reduction, or upstream map state even though its concrete children do not exist at compile time.
- Budget claims use different IR and plan types; dependency validation projects only the approval-gate identity needed for graph analysis.
- The store still uses specialized readiness tables, so shared semantic analysis must lower into node, gate, and reduction projections without pretending the SQL schema is a single graph table.

### What warrants a second pair of eyes

- Review cycle diagnostics: they are deterministic and identify typed keys, but could later be shortened to the minimal repeated segment for operator readability.
- Review whether duplicate work keys in hand-constructed plans need an additional explicit validator beyond existing compile guarantees and database constraints.
- Review defense-in-depth classification for an impossible unresolved input after a valid lease; this slice prevents the known path but does not yet introduce run quarantine.

### What should be done in the future

- If blocked-reason explanations need richer UI detail, expose dependency edge reason/path from the pure analysis rather than reconstructing it from SQL.

### Code review instructions

- Start at `pkg/workflowv3/dependencies.go`.
- Follow `Compile` into `Store.CreateRun`, then dynamic lowering in `expansion.go` and `reduction.go`.
- Run:

```bash
go test ./pkg/workflowv3 -run 'Dependency|CrossKind' -count=10
go test ./pkg/workflowv3sqlite -run 'PersistsAppendOnly|DigestValidPlan' -count=10
```

### Technical details

Dependency rule:

```text
effective node dependencies
  = explicit control-only DependsOn
  union every node-output producer in Bindings
```

Typed cycle vertices use `node:`, `map:`, `reduction:`, and `gate:` prefixes.

## Step 8: Reconcile successful terminal state in one transaction-local owner

This slice replaces four copies of run-success SQL with one transaction-local reconciler. It runs after creation, node completion, map publication, reduction publication, and gate approval; before changing status it proves every work category is complete and every declared scalar or set output is resolvable.

Zero-work scalar and set pass-through workflows now become `succeeded` in the creation transaction, emit `run.created` followed by exactly one `run.succeeded`, and return their persisted status from the product submission API. A full Researchctl runner test proves the process no longer polls forever.

### Prompt Context

**User prompt (verbatim):** (see Step 4)

**Assistant interpretation:** Implement the fourth accepted invariant boundary and prove immediate pass-through completion through store, product, and Researchctl layers.

**Inferred user intent:** Eliminate lifecycle drift and ensure future work kinds have one place to participate in terminalization.

**Commit (code):** `7f6e728c993675699dfc49b304256e99dbf83aef` — `fix: reconcile workflow terminal state centrally`

### What I did

- Added `pkg/workflowv3sqlite/reconcile.go`.
- Centralized the successful completion predicate and output-resolvability checks.
- Called reconciliation after `run.created` and every current successful work transition.
- Removed duplicated success SQL from store, expansion, reduction, and gate files.
- Added compare-and-set `run.succeeded` event emission.
- Changed `Application.SubmitArtifacts` to return persisted snapshot status.
- Added store tests for immediate set pass-through and exactly one success event.
- Added product and full Researchctl tests for immediate scalar pass-through.
- Repeated zero-work and normal completion tests twenty times before commit.

### Why

- Successful terminal state is derived from durable facts; it should not depend on which transition happened to run last.
- A status transition is incomplete if a declared output cannot be resolved.
- Submission responses must not claim `running` after creation already committed `succeeded`.

### What worked

- `TestRunnerCompletesPassThroughWithoutScheduledWork` completes, exports the original input, and produces terminal status `succeeded` without attempts.
- Set pass-through snapshots return the submitted manifest and event order `run.created,run.succeeded`.
- Existing node/map/reduction/gate suites passed against the central predicate.
- Pre-commit full tests and lint passed; runtime integration took 84.907 seconds.

### What didn't work

- N/A. Focused and repeated lifecycle tests passed on the first run.

### What I learned

- The gate subsystem already had a private completion helper, confirming the need for centralization; other transition files had independently copied the same SQL.
- Output readiness is a useful final invariant check and naturally covers pass-through refs.
- Emitting success only after a compare-and-set winner makes terminal evidence idempotent.

### What was tricky to build

- Reconciliation must use the caller's transaction. Querying through `s.db` would escape the uncommitted transition and could observe stale state or deadlock under SQLite's immediate transaction mode.
- Gate outputs may be approved without an artifact for control-only use; if such a gate is a declared data output, reconciliation correctly requires a decision ref.
- The completion predicate preserves special budget-gate semantics: inactive pending budget gates do not block, but waiting/rejected/expired/canceled activation gates do.

### What warrants a second pair of eyes

- Review whether `run.succeeded` should become part of any externally versioned observation vocabulary; current tests accept it and canonical evidence naturally includes it.
- Review output-readiness error handling: a missing output rolls back the transition that would have completed the run, surfacing an invariant defect rather than committing an unusable success.
- Consider whether terminal timestamps should become an explicit run column in a future schema revision.

### What should be done in the future

- Every future durable work kind must extend `reconcileRunStateTx` and the topology matrix in the same change.

### Code review instructions

- Start at `pkg/workflowv3sqlite/reconcile.go`.
- Verify each call occurs after its domain event and before transaction commit.
- Review pass-through tests in `store_test.go`, `application_test.go`, and `runner_test.go`.
- Run:

```bash
go test ./pkg/workflowv3sqlite -run 'PassThrough|PersistsAppendOnly' -count=20
go test ./pkg/workflowv3product -run 'ImmediatelyCompletes' -count=20
go test ./pkg/researchrunner -run 'PassThroughWithoutScheduledWork' -count=10
```

### Technical details

Reconciliation order:

```text
persist domain transition
append domain event
prove all work complete
prove all outputs resolvable
CAS run running -> succeeded
append run.succeeded if CAS won
commit once
```

## Step 9: Audit all review requirements and close implementation

This step validates the complete corrective series as one system rather than relying only on per-commit tests. The final tree passes workspace-mode tests/build, module-mode lint and pre-commit suites, focused race detection, docmgr validation, and repeated invariant tests.

The audit maps each explicit PR review comment to code and regression evidence. No required work remains for the four findings; policy questions retained in the design are follow-ups rather than correctness blockers.

### Prompt Context

**User prompt (verbatim):** (see Step 4)

**Assistant interpretation:** Finish implementation only after an evidence-backed audit, keep the diary complete, and leave focused commits suitable for PR review.

**Inferred user intent:** Receive a genuinely complete correction—not partial progress reported as done—with enough evidence for another engineer to verify it independently.

### What I did

- Ran full workspace tests with `go test ./... -count=1`.
- Ran `go build ./...`.
- Ran race tests for SQLite, runtime, and Researchctl packages.
- Ran module-mode golangci-lint over `./cmd/... ./pkg/...`.
- Ran docmgr doctor, Git whitespace checks, and clean-tree checks.
- Repeated the highest-risk custody, dependency, pass-through, and terminal event tests.
- Added an implementation outcome section to the primary design with commit mapping and accepted decisions.
- Audited every review comment against concrete tests and commits.

### Why

- The fixes cross compiler/runtime/persistence boundaries and must be validated together.
- Focused tests prove regressions precisely; full and race suites prove integration and concurrency safety.

### What worked

Full suite:

```text
ok github.com/go-go-golems/scraper/pkg/researchrunner        1.833s
ok github.com/go-go-golems/scraper/pkg/workflowv3runtime     67.358s
ok github.com/go-go-golems/scraper/pkg/workflowv3sqlite      2.537s
```

Race suite:

```text
ok github.com/go-go-golems/scraper/pkg/workflowv3sqlite      3.256s
ok github.com/go-go-golems/scraper/pkg/workflowv3runtime     24.604s
ok github.com/go-go-golems/scraper/pkg/researchrunner        1.861s
```

Additional evidence:

```text
go build ./...                              PASS
GOWORK=off ./.bin/golangci-lint ...         0 issues
docmgr doctor                               All checks passed
git diff --check                            PASS
```

### What didn't work

- No failures occurred in final validation. Earlier implementation failures are recorded at the step where they occurred (duplicate edit match, stale goldens, and nil Goja option access).

### What I learned

- The systemic approach stayed bounded: four code commits, each aligned with one invariant, with no SQLite schema migration.
- Running both workspace and `GOWORK=off` modes matters because local workspace replacements and module-resolved dependencies exercise different integration environments.

### What was tricky to build

- Completion confidence required mapping review prose to artifacts, not merely observing a green full test. Each finding now has a direct regression test and an integration path.
- The implementation changed canonical V3 plan digests through set policy and inferred dependencies; because V3 remains unreleased within PR 10, fixtures were amended consistently rather than supporting two semantic shapes.

### What warrants a second pair of eyes

- Review the new canonical contract fields and dependency ordering in generated plans.
- Review the transaction-local output readiness checks and `run.succeeded` event ordering.
- Review default resolved input size and future host maximum set cardinality as deployment policy.

### What should be done in the future

- Optionally add a host-wide set-cardinality ceiling.
- Version any future external observation use of `run.succeeded`.
- Consider minimal-cycle diagnostic formatting and streaming artifact ingestion only when justified by operator evidence.

### Code review instructions

Review commits in this order:

```text
5486f2e  fix: preserve verified workflow input custody
2dfdee1  feat: compile bounded set input contracts
b0cdd1b  fix: derive workflow readiness from data bindings
7f6e728  fix: reconcile workflow terminal state centrally
```

Then run:

```bash
go test ./... -count=1
go build ./...
go test -race ./pkg/workflowv3sqlite ./pkg/workflowv3runtime ./pkg/researchrunner -count=1
GOWORK=off ./.bin/golangci-lint run ./cmd/... ./pkg/...
```

### Technical details

Requirement audit:

| Review finding | Implementation evidence | Regression evidence |
|---|---|---|
| Stage verified bytes | `resolveInputs` stages verified body; `SubmitArtifacts` accepts refs | `TestResolveInputsStagesExactlyTheVerifiedScalarBytes` |
| Set input without direct map | `SetInputPolicy.MaxItems`; reduction capacity; direct set output | `TestRunnerStagesSetInputFromExplicitPolicy`, pass-through/compiler tests |
| Producer dependencies | `EffectiveNodeDependencies`, typed graph, plan validation | inferred lease blocking, cross-kind cycle, malformed-plan tests |
| Zero-work workflow | `reconcileRunStateTx` at creation and transitions | store/product/runner pass-through and single-event tests |

## Step 10: Publish the completed implementation record

This step publishes a new implementation bundle without overwriting the earlier review-only PDF. The bundle captures the completed code mapping, all diary steps through final validation, closed tasks, and the ticket changelog.

The separate name preserves any annotations on the original review document while making the final implementation evidence available in the same reMarkable ticket directory.

### Prompt Context

**User prompt (verbatim):** (see Step 4)

**Assistant interpretation:** Finish the ticket workflow with a durable implementation record after code, tests, commits, and audit are complete.

**Inferred user intent:** Keep the off-device review package synchronized with the completed implementation without destroying prior annotations.

### What I did

- Ran a five-document bundle dry-run.
- Uploaded `SCRAPER PR10 Systemic Correctness Implementation.pdf` with a depth-two table of contents.
- Used a distinct name instead of `--force` against the earlier review bundle.

### Why

- The original ticket request included reMarkable delivery.
- Non-overwriting delivery preserves the review artifact and any annotations.

### What worked

```text
OK: uploaded SCRAPER PR10 Systemic Correctness Implementation.pdf -> /ai/2026/08/23/SCRAPER-PR10-SYSTEMIC-REVIEW
```

### What didn't work

- N/A. Dry-run, rendering, and upload succeeded without reauthentication.

### What I learned

- Revision-specific bundle names are safer than forced replacement once a document may have been annotated.

### What was tricky to build

- Delivery occurs after the implementation diary is otherwise complete; this final local step records the upload evidence without replacing the just-uploaded bundle solely to include its own receipt.

### What warrants a second pair of eyes

- N/A for code correctness; review the uploaded PDF only for rendering/layout quality if desired.

### What should be done in the future

- Use a new revision name for any later follow-up rather than overwriting either current PDF.

### Code review instructions

- Read the implementation bundle after the original review bundle, or start directly at the implementation outcome and Steps 5–9.

### Technical details

- Remote directory: `/ai/2026/08/23/SCRAPER-PR10-SYSTEMIC-REVIEW`.
- New bundle: `SCRAPER PR10 Systemic Correctness Implementation.pdf`.
- Original preserved bundle: `SCRAPER PR10 Systemic Correctness Review.pdf`.

## Step 11: Resolve second-round dynamic readiness, identity, and CI findings

The second Codex review exposed two remaining dynamic-materialization gaps: generated map/reduction nodes did not share one lowering path for reduction consumers, and chained-map child identity incorporated a timing-dependent prefix-manifest digest. This slice centralizes all durable node readiness registration and derives chained identities from the stable upstream map source plus item key.

The same round also exposed CI-environment and security drift. Pull-request tests now install Bubblewrap, the Go and Excelize versions include published vulnerability fixes, and all six gosec findings were either removed structurally or documented at closed, validated construction points.

### Prompt Context

**User prompt (verbatim):** "Address next round of issues: https://github.com/go-go-golems/scraper/pull/10#issuecomment-5389120843"

**Assistant interpretation:** Resolve the two new inline review findings and both linked CI failures, then validate and publish another reviewable correction.

**Inferred user intent:** Bring PR 10 back to deterministic Workflow V3 semantics and a genuinely green CI state before requesting another review.

**Commit (code):** `e0a3439cc1b59a4eafe15df02568037b9a73759e` — `fix: stabilize dynamic workflow readiness identities`

**Commit (CI/security):** `dc24fb42f89a647d081128420453f05cd8a211be` — `ci: restore isolation and security checks`

### What I did

- Added `insertNodeReadiness` as the shared lowering path for static nodes, map children, and reduction partitions.
- Registered node, gate, and reduction readiness edges through that helper.
- Derived chained-map child/page identity from stable source-map identity instead of the current completed-prefix manifest digest.
- Added timing-order and generated reduction-consumer regression tests.
- Installed `bubblewrap` in the pull-request test job.
- Updated the Go directive to 1.26.6 and Excelize to 2.11.0.
- Removed unsafe integer conversion and dynamic status-table formatting findings.
- Tightened temporary plan permissions to `0600`.
- Added narrow gosec annotations where directory execute permission and closed SQL predicates are intentional.

### Why

- Dynamic nodes must obey the same binding-owned readiness contract as static nodes.
- Durable node and operation identities cannot depend on dispatcher polling timing.
- Isolation integration tests require the actual Linux sandbox executable in CI.
- Security checks should fail on real regressions, not known stale toolchains or unexplained false positives.

### What worked

- `TestChainedMapChildIdentityDoesNotDependOnPollingTiming` proves early and late polling produce identical node and page identities.
- `TestGeneratedMapNodeRegistersReductionConsumer` proves a generated child remains unleaseable until its reduction is published.
- `GOWORK=off go test ./... -count=1` and lint passed in both commits.
- Local gosec completed with `Issues: 0`.
- Local govulncheck reported `No vulnerabilities found`.
- The full runtime suite, including installed local Bubblewrap integration, passed.

### What didn't work

- The first focused test command failed because the parent workspace still declared Go 1.26.5:

```text
go: module . listed in go.work file requires go >= 1.26.6, but go.work lists go 1.26.5; to update it:
	go work use
```

  Running `go work use` refreshed the parent workspace directive.
- The first new fixtures failed compilation with `workflow requires at least one output`; explicit set outputs were added.
- The first code commit attempt failed lint:

```text
pkg/workflowv3sqlite/expansion_test.go:408:13: ineffectual assignment to summariesRef (ineffassign)
```

  The fixture now discards the obsolete ref and declares only the rebuilt summary-manifest ref.

### What I learned

- Compiler-owned semantics are insufficient unless every persistence materializer calls one shared lowering function.
- Prefix manifests are valid availability snapshots but invalid identity roots.
- The security workflow was correctly identifying both stale dependencies and several narrow code hygiene issues; fixing the causes was preferable to broad scanner exclusions.

### What was tricky to build

- A chained map legitimately observes a growing prefix. Source locator/digest columns may track that prefix for availability, while child identity must instead use stable source-map identity and item key; conflating those purposes caused the timing bug.
- Reduction consumers use specialized readiness tables rather than ordinary node dependencies, so generated nodes need both binding classes lowered transactionally before becoming lease-visible.
- Bubblewrap tests intentionally exercise real namespace/resource behavior and therefore should install the executable rather than silently skip in CI.

### What warrants a second pair of eyes

- Review whether stable chained identity should eventually incorporate an explicit upstream child key field instead of the canonical source-map digest plus item key.
- Review the three narrow `#nosec` annotations and their closed-input invariants.
- Confirm GitHub-hosted Ubuntu permits all Bubblewrap namespace/resource tests after package installation.

### What should be done in the future

- Add a generated reduction-partition test that consumes another independent reduction, complementing the shared-helper map-child regression.
- Keep the Go patch version current because several August 2026 standard-library advisories were fixed in 1.26.6.

### Code review instructions

- Start with `pkg/workflowv3sqlite/readiness.go`, then inspect its three call sites.
- Review chained source identity in `expandNextPage` and the timing-order regression.
- Review `.github/workflows/push.yml`, `go.mod`, and each gosec remediation.
- Run:

```bash
GOWORK=off go test ./... -count=1
GOWORK=off gosec -exclude=G101,G304,G301,G306,G204 -exclude-dir=.history -exclude-dir=gen ./...
GOWORK=off govulncheck ./...
```

### Technical details

```text
static node ───────┐
map child ─────────┼─> insertNodeReadiness ─> node/gate/reduction readiness tables
reduction partition┘

chained child identity = digest(consumer map, stable upstream map identity, item key)
not digest(consumer map, currently completed prefix manifest, item key)
```
