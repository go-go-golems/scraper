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
    - Path: abs:///home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system/cmd/rag-ttc-v3-sweep/main.go
      Note: Step 11 canonical per-cell operation JSONL and manifest custody
    - Path: abs:///home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system/internal/workflowv3ttc/module.go
      Note: Host-only RAG operation dispatch (commit b728e0a)
    - Path: abs:///home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system/internal/workflowv3ttc/provider.go
      Note: |-
        Real failure path that motivated the reusable ledger
        Step 9 RAG provider operation adapter (commits b728e0a 3bde483 0147ea2)
    - Path: repo://pkg/workflowv3/external_operation.go
      Note: Step 3 implementation and policy boundary (commit b637095)
    - Path: repo://pkg/workflowv3/external_operation_test.go
      Note: Step 3 focused regression evidence (commit b637095)
    - Path: repo://pkg/workflowv3sqlite/budget.go
      Note: Step 8 operation budget accounting implementation (commit b8857b1)
    - Path: repo://pkg/workflowv3sqlite/external_operation.go
      Note: |-
        Step 4 durability evidence (commit 1542075)
        Step 5 authoritative persistence implementation (commit e061769)
    - Path: repo://pkg/workflowv3sqlite/external_operation_query.go
      Note: Step 7 custody/export implementation (commit 27efa9e)
    - Path: repo://pkg/workflowv3sqlite/schema.sql
      Note: Step 4 schema implementation (commit 1542075)
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

**Commit (docs):** `2cd6536` — "docs: design durable external operation ledger"

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

## Step 3: Define closed external-operation contracts

This step implements the first reusable slice: the Workflow V3 domain model for durable external effects. The new contracts distinguish immutable host policy from one admission specification, one opaque completion ticket, and one terminal completion. This establishes the privacy and validation boundary before any SQLite persistence or runtime authority is added.

The descriptor supports counters with multiple explicitly sorted roles. A provider `requests` counter, for example, is both reserved before the call and reported as actual usage after it returns. That corrected an ambiguity in the initial guide, where a single counter role could not express this common accounting relationship without duplicating counter names.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Begin implementation of the documented generic external-operation ledger in small, reviewable slices while maintaining the ticket diary and committing at appropriate intervals.

**Inferred user intent:** Deliver production-quality reusable Workflow V3 machinery, not a workload-specific workaround, with an auditable implementation history.

**Commit (code):** `b637095` — "workflowv3: define external operation contracts"

### What I did

- Added `pkg/workflowv3/external_operation.go`.
- Added closed contracts for operation kind, version, counter roles, descriptor, admission spec, redacted failure, completion, opaque ticket, joined read model, and host-only recorder interface.
- Added canonical descriptor identity creation and validation.
- Enforced SHA-256 authority/correlation identities, bounded cardinality, sorted unique counters/roles, checked nonnegative integers, closed outcomes, closed accounting modes, and no arbitrary metadata payload.
- Added cloning helpers so policy registries can keep immutable copies.
- Added `pkg/workflowv3/external_operation_test.go` for descriptor identity, mutation isolation, malformed policies, descriptor-authorized specs, completion semantics, and ticket JSON/log redaction.
- Corrected the design's counter API sketch from one `Role` to sorted `Roles`.

### Why

Persistence needs a stable domain contract before it can safely enforce fencing or accounting. Defining policy, data, and capability types now lets the SQLite and runtime slices accept only validated, closed values and prevents an arbitrary JSON metadata channel from appearing later.

### What worked

The focused checks passed:

```text
GOWORK=off go test ./pkg/workflowv3 -count=1
ok   github.com/go-go-golems/scraper/pkg/workflowv3

GOWORK=off go test -race ./pkg/workflowv3 -count=1
ok   github.com/go-go-golems/scraper/pkg/workflowv3

GOWORK=off golangci-lint run ./pkg/workflowv3/...
0 issues.
```

The tests prove the completion key is omitted from JSON and the ticket's string representation, while descriptor digest and counter policy are deterministic.

### What didn't work

The first lint invocation failed because an unused provisional normalization helper remained after the validator was simplified:

```text
pkg/workflowv3/external_operation.go:378:6: func normalizeExternalOperationName is unused (unused)
func normalizeExternalOperationName(value string) string {
     ^
1 issues:
* unused: 1
```

I removed the unused helper and an unused import, then reran formatting, focused tests, race tests, lint, and `git diff --check` successfully.

### What I learned

- Counter roles need to be a sorted set, not a single enum: `requests`, token quantities, and cost are commonly both reservation and usage values.
- A Go struct cannot be an absolute secret container, but omitting the completion key from JSON and `String()` prevents ordinary evidence and log paths from exposing it. Later store APIs must never return it in snapshots, events, errors, or exports.
- Completion validation can reuse the existing closed Workflow failure class/code vocabulary while excluding its free-form message field.

### What was tricky to build

The important distinction is between descriptor counters and operation values. Descriptors define the only names and roles a trusted module may use. An admission spec can provide only descriptor-authorized reservation/measure values; a completion can provide only descriptor-authorized usage values. This keeps the eventual SQLite schema generic without turning into a generic arbitrary-key telemetry store.

The validation also needs to distinguish `unknown` completion from zero duration or zero cost. Unknown is allowed only with conservative accounting and no failure payload, preventing an aborted request from being misrepresented as a successful zero-cost call.

### What warrants a second pair of eyes

- Review whether provider timing must require a UTC timestamp at API validation, or whether normalization should occur at persistence boundaries while accepting offset timestamps from trusted host modules.
- Review the 100,000 per-attempt cardinality and 32-counter bounds for realistic browser/batch workloads.
- Review whether `ExternalOperationTicket.CompletionKey` should become a private capability wrapper after the SQLite package API is designed; current JSON and string redaction is tested, but Go callers can still access an exported field.
- Review future budget mappings: the domain slice intentionally validates counter roles without yet binding descriptor reservation names to `BudgetAmount` dimensions.

### What should be done in the future

Implement task `k5rh` next: additive SQLite tables, full-durability configuration, and migration/invariant tests. Do not wire the recorder into runtime or RAG until durable admission/completion storage exists.

### Code review instructions

Start with `pkg/workflowv3/external_operation.go`:

1. inspect the descriptor digest envelope and sorted-role rules;
2. verify no free-form data field or failure message exists;
3. inspect completion outcome/accounting combinations;
4. inspect ticket redaction semantics.

Then run:

```text
GOWORK=off go test ./pkg/workflowv3 -count=1
GOWORK=off go test -race ./pkg/workflowv3 -count=1
GOWORK=off golangci-lint run ./pkg/workflowv3/...
git diff --check
```

### Technical details

The implemented state contracts are:

```text
ExternalOperationDescriptor
  -> validated ExternalOperationSpec
  -> future lease-fenced BeginExternalOperation
  -> ExternalOperationTicket (operation ID + non-serialized completion key)
  -> future ticket-fenced FinishExternalOperation
  -> ExternalOperationCompletion
```

The descriptor digest is SHA-256 over canonical JSON with `Digest` cleared. Counter names and roles are required to be strictly sorted and unique, so equal semantic policies always produce the same digest.

## Step 4: Add the additive SQLite evidence schema and durability checks

This step adds the persistent foundation for the operation ledger without yet granting any runtime code authority to write it. The schema now has separate admission, admission allocation/measure, immutable completion, and completion-counter tables. The existing SQLite open path now requests WAL, foreign keys, and `synchronous=FULL`, then verifies those settings before allowing the store to be used.

The phase deliberately stops before Begin/Finish APIs. That keeps migration and durable-open behavior reviewable independently from lease fencing, operation tickets, and budget settlement changes that will follow in the next slice.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue the planned generic ledger implementation with a durable SQLite foundation and retain exact validation evidence in the ticket diary.

**Inferred user intent:** Ensure external-call authority and evidence can survive failure without weakening existing Workflow V3 database compatibility or privacy guarantees.

**Commit (code):** `1542075` — "workflowv3: add external operation ledger schema"

### What I did

- Added five additive tables to `pkg/workflowv3sqlite/schema.sql`:
  - `v3_external_operations`
  - `v3_external_operation_allocations`
  - `v3_external_operation_measures`
  - `v3_external_operation_completions`
  - `v3_external_operation_counters`
- Added foreign keys, uniqueness constraints, nonnegative amount checks, immutable-completion shape, and attempt/kind indexes.
- Added `pkg/workflowv3sqlite/external_operation.go` with startup checks for WAL, `synchronous=FULL`, foreign keys, and referential operation-ledger consistency.
- Requested `_synchronous=FULL` in the SQLite DSN and invoke the durability and operation-invariant checks at store open.
- Extended the legacy minimal-database migration test to require the five new tables and assert `journal_mode=wal`, `synchronous=2`, and `foreign_keys=1`.

### Why

Operation admission is a pre-effect authority boundary. A later checkpoint cannot repair a crash window between returning admission and submitting a provider call. The store must request and verify durable commit configuration before the future Begin API exists.

Separate admission and completion tables make incomplete calls explicit and keep terminal evidence append-only. Foreign keys attach every operation to a real Workflow attempt and prevent orphaned counters from becoming canonical evidence.

### What worked

The targeted validation passed:

```text
GOWORK=off go test ./pkg/workflowv3sqlite -count=1
ok   github.com/go-go-golems/scraper/pkg/workflowv3sqlite

GOWORK=off go test -race ./pkg/workflowv3sqlite -count=1
ok   github.com/go-go-golems/scraper/pkg/workflowv3sqlite

GOWORK=off go test ./pkg/workflowv3runtime -run '^(TestEngineCompletesWorkflow|TestRunTaskPreservesTypedFailure)$' -count=1
ok   github.com/go-go-golems/scraper/pkg/workflowv3runtime

GOWORK=off golangci-lint run ./pkg/workflowv3sqlite/...
0 issues.
```

The historical minimal-schema fixture opens successfully and the test proves all external-operation tables and required PRAGMA settings exist afterward.

### What didn't work

The full runtime package has an existing storage-size assertion failure unrelated to this schema addition:

```text
GOWORK=off go test ./pkg/workflowv3runtime -run '^TestDatabaseSyncCrashAfterSideEffectIsIdempotentAcrossRestart$' -count=3 -v

Error: "299008" is not less than "249777"
Test: TestDatabaseSyncCrashAfterSideEffectIsIdempotentAcrossRestart
```

I temporarily changed the new DSN setting from `FULL` to `NORMAL` and adjusted only the temporary startup assertion to reproduce the baseline. The same test failed with exactly the same `299008` versus `249777` result. I restored `FULL` and the required assertion; no unrelated threshold was changed or masked.

### What I learned

- The failing storage assertion includes the SQLite database, WAL, and SHM files together. Its fixed ratio is not a valid signal for this change because it fails under both the prior-normal and new-FULL settings in the same environment.
- Driver DSN support explicitly recognizes `_synchronous=FULL`, and the runtime query returns SQLite value `2`; testing this at store-open prevents a silent configuration regression.
- Schema-level foreign keys make operation orphan checks defensive reconciliation rather than the primary integrity mechanism.

### What was tricky to build

SQLite configuration is per connection, while `database/sql` manages a pool. Setting `_synchronous=FULL`, `_journal_mode=WAL`, and `_foreign_keys=on` in the DSN ensures the driver applies them to each opened connection; checking PRAGMAs at startup catches an unsupported or unexpectedly altered connection configuration. The initial implementation retains the existing four-connection pool rather than introducing a hidden single-connection performance change.

The operation tables are intentionally not a mutable `status` table. An admission may exist without completion after process death. Completion is a separate one-to-zero-or-one relation with a unique operation ID, which will make exactly-once Finish semantics enforceable in the next phase.

### What warrants a second pair of eyes

- Confirm that globally requiring `synchronous=FULL` is acceptable for all Workflow V3 workloads, or decide whether operation admission should move to a dedicated authority-bearing connection after performance measurement.
- Review whether a future schema migration should persist descriptor bodies in addition to their digests; this slice records only immutable identifiers, matching the privacy constraint.
- Investigate the pre-existing `TestDatabaseSyncCrashAfterSideEffectIsIdempotentAcrossRestart` storage-ratio failure separately; it should not be normalized as part of this ticket.

### What should be done in the future

Implement task `u57a`: lease-fenced admission and operation-ticket-fenced completion APIs, with reopen, cancellation, idempotency, and stale-lease tests. Keep the full-runtime storage-ratio failure visible in subsequent validation reports until it is separately resolved.

### Code review instructions

Review `schema.sql` next to the existing attempts and budget reservation tables, then inspect `Store.Open` and `checkSQLiteDurability`. Verify that no new source/provider payload field is present and that all dependent rows cascade from the admission/completion identities.

Run:

```text
GOWORK=off go test ./pkg/workflowv3sqlite -count=1
GOWORK=off go test -race ./pkg/workflowv3sqlite -count=1
GOWORK=off golangci-lint run ./pkg/workflowv3sqlite/...
git diff --check
```

### Technical details

The durable relation is now:

```text
v3_attempts
  └─ v3_external_operations (one admission per ordinal)
       ├─ allocations and measures
       └─ v3_external_operation_completions (zero or one)
            └─ counters
```

The next API must write the admission transaction before submitting an effect and must never use a completion to mutate task/node/run state.

## Step 5: Implement lease-fenced admission and ticket-fenced completion

This step turned the schema into working durable behavior. `BeginExternalOperation` validates a host descriptor/spec, checks the active Workflow lease inside an immediate SQLite transaction, allocates an immutable operation ordinal and a random completion capability, persists admission/allocations/measures/events, and commits before returning. `FinishExternalOperation` uses only that capability—not the live lease—to append exactly one safe completion after cancellation or lease loss.

This is the key semantic split required by the design: cancellation forbids a new external call, but it must not erase the safe completion evidence for a request that was admitted earlier. Completion remains incapable of changing task, node, run, output, or budget-terminal state.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue implementation with the durable ledger's central Begin/Finish authority and evidence semantics.

**Inferred user intent:** Make failed/canceled external calls measurably auditable without allowing stale task execution to regain Workflow authority.

**Commit (code):** `e061769` — "workflowv3: persist external operation evidence"

### What I did

- Implemented `Store.BeginExternalOperation` and `Store.FinishExternalOperation` in `pkg/workflowv3sqlite/external_operation.go`.
- Generated 256-bit random completion keys, persisted only SHA-256 digests, and omitted raw keys from events/JSON.
- Enforced current lease/cancel-epoch fencing only at Begin.
- Enforced descriptor digest and constant-time completion-ticket checks at Finish.
- Added exactly-once immutable completion with byte-identical completion digest idempotency and `ErrExternalOperationCompletionConflict` on conflict.
- Persisted bounded admitted/completed events.
- Added focused tests for admission, cancellation followed by successful late completion, idempotent finish, conflicting finish, stale Begin rejection, and wrong-ticket rejection.

### Why

The ledger cannot preserve provider-wall evidence unless it creates a durable pre-call record and can append the outcome even after the enclosing attempt becomes stale. Requiring the normal `checkFence` at Finish would incorrectly discard cancellation evidence; omitting it at Begin would permit unauthorized new effects.

### What worked

```text
GOWORK=off go test ./pkg/workflowv3sqlite -run '^TestExternalOperation' -count=1
ok   github.com/go-go-golems/scraper/pkg/workflowv3sqlite

GOWORK=off go test -race ./pkg/workflowv3sqlite -run '^TestExternalOperation' -count=1
ok   github.com/go-go-golems/scraper/pkg/workflowv3sqlite

GOWORK=off golangci-lint run ./pkg/workflowv3sqlite/...
0 issues.
```

The tests prove one operation/completion/counter row after a late completion and prove a stale canceled lease cannot admit a second operation.

### What didn't work

The initial focused test compile failed because of an unused import:

```text
pkg/workflowv3sqlite/external_operation_test.go:5:2: "errors" imported and not used
FAIL github.com/go-go-golems/scraper/pkg/workflowv3sqlite [build failed]
```

I removed it and reran focused tests, race tests, lint, and diff checks successfully.

### What I learned

- Completion idempotency must compare the canonical completion digest, not merely operation ID; otherwise conflicting terminal facts could be silently accepted.
- The completion capability is sufficient to associate a late return with a prior valid admission without reopening node/output authority.
- Allocation-to-budget reconciliation is still intentionally deferred: this slice persists descriptor-authorized allocations but does not yet alter existing attempt settlement.

### What was tricky to build

SQLite's active lease fence correctly protects Begin but is deliberately inappropriate for Finish. The completion code instead reads only the admitted operation's descriptor/key digest, inserts the immutable terminal row, and emits safe evidence. This is the minimum authority required for a late observation.

### What warrants a second pair of eyes

- Review whether explicit operation IDs should use UUIDs or a digest-derived identifier; UUIDs avoid correlation disclosure but make deterministic exports rely on ordinal sorting.
- Review error typing for an invalid completion key versus an operation that does not exist; both currently fail closed without revealing sensitive state.
- Review the future finish context behavior in the runtime: it must be bounded and detached from cancellation only for evidence persistence.

### What should be done in the future

Implement task `ujto`: inject a recorder scoped to trusted host modules, without exposing it to JavaScript. Then add operation query/export APIs and budget reconciliation.

### Code review instructions

Review Begin's call ordering: validation, active lease fence, admission persistence, event, commit, then returned ticket. Review Finish's ticket-only authority and idempotent completion digest branch. Run the focused test/race/lint commands above.

### Technical details

```text
active Workflow lease --Begin--> durable operation admission --ticket-->
provider/tool call --Finish--> immutable completion

canceled/stale lease --Begin--> rejected
canceled/stale lease + prior ticket --Finish--> accepted safe evidence only
```

## Step 6: Scope recorders to trusted module factories

This step connects the ledger to the Workflow runtime without making it a JavaScript capability. Exact host-module factories now declare immutable operation descriptors. The engine collects descriptors only for the leased task's requested module aliases and builds a recorder bound to that lease. The recorder checks the lease during admission and resolves only those descriptor digests.

`TaskModuleContext` receives the recorder for trusted Go module implementation. `workflow/task` remains unchanged, and the isolated-worker request protocol still has no recorder or database path.

**Commit (code):** `80ef254` — "workflowv3: scope external operation recorders"

### What I did
- Added descriptor ownership and cloned lookup to `TaskModuleRegistry`.
- Added the lease-scoped SQLite recorder adapter.
- Constructed and injected the recorder in `Engine.ExecuteLease`.
- Passed it only through `TaskRequest` and `TaskModuleContext`.
- Added registry clone/scope regression coverage.

### Why
The host module that performs a provider/tool call needs the recorder, but JavaScript and restricted workers must not gain arbitrary external-operation or database authority.

### What worked
```text
GOWORK=off go test ./pkg/workflowv3runtime -run '^(TestTaskModuleRegistry|TestEngineCompletesWorkflow|TestRunTaskPreservesTypedFailure)$' -count=1
ok github.com/go-go-golems/scraper/pkg/workflowv3runtime
GOWORK=off golangci-lint run ./pkg/workflowv3runtime/... ./pkg/workflowv3sqlite/...
0 issues.
```

### What didn't work
The initial compile omitted the `workflowv3` import in `modules.go`; Go reported undefined references. Adding the explicit import fixed the build. No behavior was masked.

### What I learned
Descriptor selection belongs to exact module aliases, not JavaScript task code. A recorder with no matching descriptor rejects admission; completing an operation still uses the admission's descriptor digest and ticket.

### What was tricky to build
The engine must treat descriptor/recorder construction failures as typed Workflow failures before task execution. This preserves parent lease authority while avoiding a fresh side channel into Goja.

### What warrants a second pair of eyes
Review the recorder's completion lookup path and the planned bounded finish context before provider modules begin using it.

### What should be done in the future
Implement queries/projections/export next, then wire the RAG provider using the scoped recorder.

### Code review instructions
Review `modules.go`, `engine.go`, and the SQLite adapter. Verify no JavaScript object or isolated-worker protocol field exposes the recorder.

## Step 7: Add coherent operation queries, projection, and atomic custody export

This step makes the persisted ledger usable after task or run failure. Workflow V3 can now reconstruct ordered compact operation records, derive bounded counts/outcomes/active-by-kind status, and atomically publish a canonical JSONL ledger plus a digest/size manifest. The export deliberately includes incomplete admissions rather than inventing a completion.

The export is not tied to successful task outputs. It therefore remains available for succeeded, failed, canceled, and partially observed runs and is safe to checkpoint before a source-bearing transient database is removed.

**Commit (code):** `27efa9e` — "workflowv3: export external operation evidence"

### What I did

- Added canonical JSONL export record and manifest contracts.
- Added ordered `Store.ExternalOperations`, bounded progress, and per-run operational projection.
- Added coherent read transactions for detailed rows and safe scalar counters.
- Added atomic JSONL and manifest file writers with file and directory fsync.
- Added plan digest, event sequence, row/completion/incomplete counts, descriptor digests, JSONL SHA-256, byte count, and privacy classification to the manifest.
- Added regression coverage for successful late completion, query reconstruction, progress, ticket non-disclosure, deterministic repeated export bytes, and manifest output.
- Made `CompletedAt` store-owned so callers cannot claim their own completion persistence timestamp.

### Why

A durable recorder is only useful if its evidence can leave a transient workflow database safely. JSONL is the primary row-level evidence; the manifest identifies an exact immutable export and allows downstream researchctl artifact verification.

### What worked

```text
GOWORK=off go test ./pkg/workflowv3 ./pkg/workflowv3sqlite -count=1
ok github.com/go-go-golems/scraper/pkg/workflowv3
ok github.com/go-go-golems/scraper/pkg/workflowv3sqlite

GOWORK=off go test -race ./pkg/workflowv3 ./pkg/workflowv3sqlite -count=1
ok github.com/go-go-golems/scraper/pkg/workflowv3
ok github.com/go-go-golems/scraper/pkg/workflowv3sqlite

GOWORK=off golangci-lint run ./pkg/workflowv3/... ./pkg/workflowv3sqlite/...
0 issues.
```

### What didn't work

The first export test reused `operations` as both a SQL count variable and the operation slice:

```text
no new variables on left side of :=
cannot use store.ExternalOperations(...) as int value in assignment
```

I renamed the SQL count to `operationRows`, then reran focused, race, lint, and diff validation successfully.

### What I learned

- The manifest must avoid a generated-at timestamp if byte-identical repeated export is required. Event sequence plus digest/size supplies a stable as-of identity.
- Incomplete does not imply active. The projection computes incomplete from every admission without completion, while active-by-kind includes only incomplete rows whose attempt remains `running`.
- Completion timestamps belong to persistence, not to a provider caller; provider start plus monotonic elapsed remains the data-plane timing evidence.

### What was tricky to build

The export has two crash-safe files. JSONL is written and fsynced first, renamed, and directory-synced; the manifest is then canonically written by the same sequence. A caller may only remove a runtime after both files are present. The manifest's JSONL digest protects against a partial or mismatched downstream file.

### What warrants a second pair of eyes

- The current detailed query performs bounded child-counter queries per operation. It is simple and correct, but a large-scale workload may warrant a batched internal loader after a benchmark.
- Review whether export paths should be constrained to an explicit evidence root by a future higher-level custody API rather than accepting operator-provided output paths.
- Verify the researchctl bridge uses both JSONL and manifest artifacts and checks the manifest digest before treating derived metrics as evidence.

### What should be done in the future

Implement budget reconciliation next, then instrument RAG provider calls and use this export for failed/successful sweep-cell custody.

### Code review instructions

Start with `external_operation_query.go`. Verify stable order, no secret completion key, atomic writer ordering, and explicit incomplete count. Then inspect `operational.go` for coherent progress projection and run the validation commands above.

## Step 8: Reconcile operation reservations with Workflow attempt budgets

This step closes the accounting gap between observed effects and existing Workflow budgets. An external operation may now reserve only dimensions already reserved for its active attempt, and cumulative operation allocations cannot exceed that attempt reservation. At settlement, actual operation counters become authoritative for allocated dimensions; incomplete or conservative operations charge their allocation.

This keeps one budget ledger: `v3_budget_reservations` remains authoritative for run limits, while operation allocations explain how an attempt spent its already-reserved capacity.

**Commit (code):** `b8857b1` — "workflowv3: reconcile operation budget usage"

### What I did

- Checked every operation reservation against the active attempt's durable reservation.
- Rejected unbudgeted, over-allocated, and overflowed operation admissions.
- Validated actual completion counters against admitted allocations.
- Derived actual/conservative operation usage during attempt settlement.
- Rejected disagreement between task-reported and operation-derived usage.
- Allowed operation-derived dimensions to satisfy actual settlement without duplicate task reporting.
- Added a budgeted operation regression proving one request allocation, rejected second admission, actual completion, and final budget settlement.

### Why

Provider requests, tokens, and cost are effects. A task cannot claim those values in a separate unverified channel after the operation ledger exists. The admission allocation needs to prevent excess calls before provider submission, while settlement needs conservative charging when a call's usage remains unknown.

### What worked

```text
GOWORK=off go test ./pkg/workflowv3sqlite -count=1
ok github.com/go-go-golems/scraper/pkg/workflowv3sqlite
GOWORK=off go test -race ./pkg/workflowv3sqlite -count=1
ok github.com/go-go-golems/scraper/pkg/workflowv3sqlite
GOWORK=off golangci-lint run ./pkg/workflowv3sqlite/...
0 issues.
```

### What didn't work

The first build failed because `budget.go` needed the `math` import for checked integer summation:

```text
pkg/workflowv3sqlite/budget.go:309:50: undefined: math
```

Adding the import fixed the build; no accounting behavior was bypassed.

### What I learned

- Actual operation accounting must validate against admission allocation at Finish, not only during final task settlement; otherwise a malformed completion could remain durable until a distant terminal path.
- An incomplete admission is naturally conservative: it has a durable allocation but no terminal completion row.
- A task that reports an operation-bound dimension must agree exactly with the ledger. A task may omit that dimension because the ledger supplies it authoritatively.

### What was tricky to build

There are three distinct settlement states: actual counters, conservative allocation, and no allocation. The implementation preserves existing behavior for tasks with no external allocations, derives only allocated dimensions, and retains existing reserved/used invariants. This avoids inventing a parallel accounting system.

### What warrants a second pair of eyes

- Verify the intended policy for a provider that reports partial usage after timeout; current descriptors choose actual versus conservative at completion.
- Review future multi-request batch descriptors to ensure their allocation and counter names remain aligned with task budget claims.
- Stress-test the per-attempt allocation aggregation under higher concurrent Begin calls; SQLite immediate transactions serialize it, but a benchmark should quantify overhead.

### What should be done in the future

Wire the RAG generation and embedding providers through the host-only recorder, export each sweep cell before transient deletion, and add forced malformed/timeout/cancellation fixture tests.

### Code review instructions

Review `checkExternalOperationAllocation`, `validateCompletionAllocation`, and `operationActualUsage`, then run the focused SQLite tests. Check that no path can submit more allocated request units than the active attempt reservation.

## Step 9: Adapt TTC generation and embedding providers to the generic ledger

This step connected the generic host-only recorder to RAG TTC provider calls. `OperatorProvider` now declares exact generation and embedding operation descriptors tied to non-secret provider/embedding profile digests. Generation is admitted after cumulative sweep authority and immediately before the provider call. Embedding is admitted once per underlying embedding provider request, not once per aggregate batch.

The implementation preserves the existing domain contracts: RAG still owns provider error classification, representation validation, and domain usage. Workflow owns admission, span custody, and budget authority. The provider uses a short `context.WithoutCancel` persistence window only after a call returns, so cancellation stops the call but does not erase safe completion evidence.

**Commits (code):** `b728e0a` — "workflowv3: record TTC generation operations"; `3bde483` — "workflowv3: record TTC embedding operations"; `0147ea2` — "workflowv3: preserve TTC malformed provider spans"

### What I did

- Updated RAG's scraper dependency to the pushed ledger implementation (`b8857b1`); private-module retrieval required `GOPRIVATE`/`GONOSUMDB`.
- Added operation descriptor/provider capability interfaces.
- Added generation and embedding descriptors to `OperatorProvider`.
- Wrapped generation and per-request embedding calls in recorder Begin/Finish.
- Recorded actual counters where known and conservative completion on provider error.
- Recorded a failed, conservative generation span when returned representations are malformed.
- Added a recorder regression test for generation success.
- Ran focused TTC tests, lint, then repository hooks including full core tests, TypeScript typecheck, and snapshot release builds.

### What worked

The RAG repository pre-commit and pre-push hooks passed. The pushed pre-push evidence includes lint, `go vet`, core package tests, site typecheck, GoReleaser snapshot build, and a second core test pass.

### What didn't work

The first `go get` of the newly pushed scraper commit failed through the public checksum service:

```text
verifying module: reading https://sum.golang.org/lookup/github.com/go-go-golems/scraper@...: 500 Internal Server Error
```

Rerunning with the repository's private-module configuration succeeded:

```text
GOWORK=off GOPRIVATE=github.com/go-go-golems/* GONOSUMDB=github.com/go-go-golems/* go get github.com/go-go-golems/scraper@b8857b1
```

### What I learned

- A batch abstraction can conceal multiple provider calls. Embedding instrumentation belongs around each `Embed` request so operation concurrency and request count remain true.
- Cumulative sweep authority is intentionally ordered before generic Workflow operation admission; a crash can overcount conservatively but cannot undercount a submitted generation request.
- Provider response validation is a domain failure after a successful transport response; it still needs a durable span. The current implementation records it as a failed conservative operation to keep the measurement available.

### What was tricky to build

The `OperatorProvider` instance is shared while recorders are lease-scoped. The adapter therefore takes a recorder as a method argument rather than storing it in provider configuration, avoiding cross-attempt leakage and races.

### What warrants a second pair of eyes

- Review the remaining early accounting-error paths after a provider success: they currently preserve the admission and will conservatively settle, but should be normalized into explicit completion outcomes before the real fixture qualification.
- Review whether malformed response completion should be `failed/actual` rather than `failed/conservative` once usage extraction is moved earlier in the generation path.
- Add an end-to-end engine test using `OperatorProvider` rather than only a recorder unit test before authorizing a paid run.

### What should be done in the future

Update the sweep to export per-cell operation JSONL/manifests before runtime deletion, add failure-cell reductions from those records, then implement researchctl verified artifact import and fixture failure qualification.

### Code review instructions

Review RAG `internal/workflowv3ttc/provider.go` around provider admission/call/finish ordering and `module.go` for host-only dispatch. Confirm no prompt, source text, provider body, URL, header, or credential crosses into an operation spec or completion.

## Step 10: Contain fixture-control latency regression

I attempted a 16-chunk fixture sweep after enabling the new operation recorder through `OperatorProvider`. The first cell completed all sixteen generation and sixteen embedding attempts but remained running until the fixed 30-second fixture deadline. Its compact failed-cell custody exported zero-secret operation files. This indicates that the additional FULL-synchronous admission/completion transactions materially change fixture-control timing or expose a downstream completion delay.

I did not increase the fixture timeout or publish the sweep custody wiring. I restored the uncommitted sweep changes so the regression is contained, leaving the committed provider instrumentation available for focused investigation.

**Exact command:** `GOWORK=off go run ./cmd/rag-ttc-v3-sweep --profile fixtures --chunks 16 --concurrency 1,2,4 --maximum-requests 90 --output /tmp/rag-ttc-operation-control`

**Exact failure:** `cell {ChunksPerRequest:1 Concurrency:1 Replicate:1} timed out after 30s with status running (dispatcher: context canceled)`.

### What should be done in the future

Profile Begin/Finish transaction latency and inspect the run/map/reduction state before deciding whether fixture controls need a deliberately instrumented profile or whether the recorder has a completion-path defect. Do not raise the timeout blindly.

## Step 11: Verify per-cell operation custody through the TTC sweep

The prior timeout proved transient: a repeat with the ordinary 30-second cell bound completed all twelve fixture cells. I then wired the generic export into the RAG sweep. Before a successful cell's SQLite runtime is deleted, the sweep writes its canonical operation JSONL and manifest and places only their relative paths plus canonical manifest data in the cell checkpoint. Timeout and terminal paths do the same before their failed-cell checkpoint. Export/checkpoint errors now fail loudly rather than silently producing incomplete custody.

A fixture run produced 282 operation records across twelve cells; every referenced JSONL and manifest existed. This is a generic-ledger integration validation, not paid-provider authority.

**RAG validation command:** `GOWORK=off go run ./cmd/rag-ttc-v3-sweep --profile fixtures --chunks 16 --concurrency 1,2,4 --maximum-requests 90 --output /tmp/rag-ledger-default-check`.

### What was tricky to build

The sweep deletes per-cell SQLite files to avoid retaining transient runtime state. Export therefore must happen after terminal snapshot/budget resolution but before `Store.Close` and runtime deletion. The error paths need exactly the same ordering; otherwise failed cells would be the ones least likely to retain call-time evidence.

## Step 12: Close per-cell evidence and failed reductions

The RAG sweep now derives a closed, deterministic reduction directly from operation ledger rows for failed or timed-out cells: admission/completion/incomplete counts, bounded outcomes, elapsed time, peak active spans, generation/embedding overlap, and operation counts. A forced one-nanosecond fixture deadline emitted a privacy-safe failed checkpoint, JSONL, and manifest with a zero-operation reduction; the normal fixture run exercises populated per-cell exports. No provider text or arbitrary metadata is admitted into this reduction.

## Step 13: Concurrent ledger admission/completion regression

Added a store-level concurrency regression that admits sixteen operations concurrently under one active Workflow lease, then completes those exact tickets concurrently. It asserts all admissions and completions are persisted, the joined query returns every row, and the bounded progress projection is `16/16/0` (admitted/completed/incomplete). The same test passed under Go's race detector.

**Validation:**

```text
GOWORK=off go test ./pkg/workflowv3sqlite -run 'TestExternalOperation(ConcurrentAdmissionAndCompletion|AdmissionAndLateTicketCompletion|RejectsWrongCompletionTicket)' -count=1
GOWORK=off go test -race ./pkg/workflowv3sqlite -run TestExternalOperationConcurrentAdmissionAndCompletion -count=1
```

This advances, but does not close, the lifecycle validation task: restart/process-death and broader cancellation/lease-loss privacy-corpus checks remain required.

## Step 14: Researchctl custody seam reconnaissance

I traced researchctl's existing `import-run` path before adding a domain adapter. The correct boundary is a canonical `lab.RunExport` bundle authored by RAG: `researchctl` verifies and stages referenced relative artifacts through `lab.StageImportedArtifacts`, validates checks, then atomically imports the generic export. Researchctl must not import RAG or Workflow packages.

The RAG-side adapter must therefore create one completed external-run export whose artifacts include the sweep evidence, per-cell operation JSONL, and manifests, and whose scalar metrics are derived only from the compact evidence. It must receive a canonical researchctl specification, explicit operator-recorded timestamp, project/experiment import target, and stable external-run identity; inventing timestamps or using host paths would make repeat import identity non-deterministic or leak privacy-sensitive information.

The next implementation slice is an RAG-owned exporter plus an import integration test that verifies artifact staging and scalar metrics via researchctl's public contracts. This reconnaissance made no code change and left task `a77h` open.

## Step 15: RAG-owned researchctl custody builder

Implemented the RAG-side `BuildOperationCustodyRunExport` adapter. It turns prepublished compact operation artifacts and scalar reductions into researchctl's existing generic run-export format, while keeping source paths out of persisted values. The builder is strict about relative artifact URIs, identity, and timestamps; researchctl's own verifier validates the completed bundle. CLI wiring/import remains the next `a77h` substep.

## Step 16: End-to-end researchctl operation-custody import

Wired the RAG sweep to emit an optional compact `researchctl-run-export.json` only with explicit operator-owned custody identity fields. The generated bundle references aggregate/cell evidence plus every JSONL/manifest artifact using relative URIs and derives four scalar-only metrics. A fresh researchctl laboratory successfully staged and atomically imported 37 verified artifacts and four metrics from a twelve-cell fixture sweep. This completes task `a77h` without creating a researchctl-to-RAG dependency.

## Step 17: Reopen-safe operation completion regression

Added a process-boundary persistence regression: admit an operation, close the SQLite store, reopen it, then finish using the original opaque ticket. The reopened store accepts the first completion, accepts an identical retry idempotently, and reports `1/1/0` admitted/completed/incomplete. Focused normal and race runs plus package lint passed.

This validates the key crash window after durable admission and before post-call completion. It does not yet model an actual SIGKILL between SQLite writes, but exercises the durable reopen semantics through the same public store boundary.
