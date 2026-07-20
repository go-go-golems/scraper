---
Title: Investigation diary
Ticket: SCRAPER-RESUMABLE-WORKFLOW-HARDENING
Status: active
Topics:
    - architecture
    - scraper
    - scheduler
    - worker
    - sqlite
    - workflows
    - onboarding
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/05/22/SCRAPER-SESSIONSTREAM-EVENTS--use-sessionstream-as-the-scraper-runtime-event-distribution-mechanism/design-doc/01-intern-guide-to-sessionstream-backed-scraper-runtime-events.md
      Note: Existing event-distribution boundary reviewed
    - Path: repo://ttmp/2026/07/20/SCRAPER-RESUMABLE-WORKFLOW-HARDENING--harden-scraper-for-long-running-resumable-batch-workflows/scripts/01-probe-retry-descendants.go
      Note: Reproduces finalizer cancellation after manual repair
    - Path: repo://ttmp/2026/07/20/SCRAPER-RESUMABLE-WORKFLOW-HARDENING--harden-scraper-for-long-running-resumable-batch-workflows/scripts/02-probe-single-process-concurrency.go
      Note: Measures lack of in-process concurrency
    - Path: repo://ttmp/2026/07/20/SCRAPER-RESUMABLE-WORKFLOW-HARDENING--harden-scraper-for-long-running-resumable-batch-workflows/scripts/03-probe-stale-lease-completion.go
      Note: Proves stale worker can commit after re-lease
    - Path: repo://ttmp/2026/07/20/SCRAPER-RESUMABLE-WORKFLOW-HARDENING--harden-scraper-for-long-running-resumable-batch-workflows/scripts/04-probe-rfc3339nano-text-ordering.go
      Note: Proves mixed-precision timestamp TEXT ordering defect
ExternalSources: []
Summary: Chronological evidence, probes, and design reasoning for scraper durable-workflow hardening.
LastUpdated: 2026-07-20T20:15:00Z
WhatFor: Resume implementation of lease safety, recovery, concurrency, identity, and observer hardening with exact observed behavior.
WhenToUse: Read before changing scraper scheduler/store state transitions or implementing ticket tasks.
---


# Diary

## Goal

This diary records the investigation and design work for hardening scraper as a reusable engine for long-running, resumable, failure-isolated batch workflows. It includes exact probe output, observed state-machine gaps, implementation boundaries, and validation instructions.

## Step 1: Create the Engine Hardening Ticket and Map Existing Boundaries

I created `SCRAPER-RESUMABLE-WORKFLOW-HARDENING` inside the scraper repository rather than treating the work as a downstream RAG-only workaround. The ticket is intentionally engine-focused: it addresses leases, operation transitions, concurrency, idempotent attachment, observer APIs, inspection, migrations, and release compatibility, while leaving provider policy and application payloads to consumers.

I also reviewed the prior scraper runtime-event and sessionstream tickets. Scraper already has a protobuf `RuntimeEventV1`, Watermill/Redis Streams options, sessionstream projections, and a WebSocket endpoint. Those systems should consume generic post-commit scheduler transitions; this ticket does not replace their transport or frontend design.

### Prompt Context

**User prompt (verbatim):** "Create a ticket in ./scraper (which i've now added to the workspace) and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.
 and tasks."

**Assistant interpretation:** Create a scraper-local docmgr ticket with implementation tasks and a thorough intern-oriented guide explaining the required engine hardening, then validate and publish the documentation bundle to reMarkable.

**Inferred user intent:** Turn the prior downstream assessment into an actionable upstream scraper roadmap that a new engineer can implement safely.

### What I did

- Created ticket `SCRAPER-RESUMABLE-WORKFLOW-HARDENING` under `ttmp/2026/07/20/`.
- Created this diary, the primary design guide, and implementation/release/validation/upload tasks.
- Read core models, store interfaces, SQLite schema/migrations, scheduler, result/lease/read/mutation services, embedded workflow facade, CLI worker composition, runtime events, sessionstream runtime, and prior ticket designs.
- Defined an explicit boundary: generic scraper transitions and snapshots belong here; RAG, researchctl, Goja, dashboard-specific payloads, and provider recovery policies do not.

### Why

Long workloads require engine guarantees that downstream consumers cannot safely reconstruct: only scraper can make lease ownership atomic, distinguish blocked from canceled, prevent stale commits, enforce concurrent queue admission, and expose a stable embedding API.

### What worked

- Scraper already has strong durable primitives and a clean package split.
- Existing workflow API is a credible embedding seam.
- Existing runtime event/sessionstream work removes the need to invent a second event transport.
- SQLite store contracts concentrate state changes enough to harden them centrally.

### What didn't work

- No implementation failures occurred in this documentation setup step.

### What I learned

- The codebase already has generic runtime event distribution, so the important remaining observer work is post-commit safety and public workflow-facade exposure.
- The engine's existing `MaxWorkers` API/documentation overpromises relative to v0.0.4 execution behavior.

### What was tricky to build

The difficult design boundary was separating durable engine correctness from live event transport. A scheduler observer is useful for runtime events, but it cannot become the source of truth because observer events disappear after process restart. The guide therefore requires store-derived snapshots and treats sessionstream/WebSocket as delivery infrastructure.

### What warrants a second pair of eyes

- Whether `blocked` should be a state versus a separate reason/transition table.
- Whether the optional transition history table should be part of the first release.
- Whether epoch microseconds provide enough time precision for all current users.

### What should be done in the future

- Complete the implementation tasks in phase order; do not change transport/UI behavior before state-machine invariants pass.

### Code review instructions

- Start with the guide’s Executive Summary and Sections 2-4.
- Read `scheduler.go`, `lease_store.go`, `result_store.go`, and `op_store.go` before editing public APIs.

### Technical details

- Baseline analyzed: scraper `v0.0.4`, commit `17f6f6528eaef77d5dcf847e39b4a8cdada9d4a1`.
- Existing related tickets: `SCRAPER-RUNTIME-EVENTS`, `SCRAPER-SESSIONSTREAM-EVENTS`, and `SCRAPER-DASHBOARD`.

## Step 2: Execute Regression Probes for Recovery, Concurrency, and Lease Safety

I copied the two previously established scheduler probes into the scraper ticket and added two store-level lease/time probes. The tests confirmed the known retry-descendant and sequential-concurrency behavior, then found two additional release-blocking lease correctness bugs: a stale worker can commit after re-leasing, and mixed-precision RFC3339Nano strings are not chronologically sortable in SQLite TEXT predicates.

The guide now treats time representation and token-verified atomic result/failure writes as Phase 1 requirements, rather than assuming the existing heartbeat method can be wired into the scheduler unchanged.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Gather concrete scraper evidence before writing implementation recommendations.

**Inferred user intent:** Ensure the proposed engine work addresses real observable behavior rather than speculative architecture concerns.

### What I did

- Added and ran `scripts/01-probe-retry-descendants.go`.
- Added and ran `scripts/02-probe-single-process-concurrency.go`.
- Added and ran `scripts/03-probe-stale-lease-completion.go`.
- Added and ran `scripts/04-probe-rfc3339nano-text-ordering.go`.
- Formatted all ticket probes with `gofmt`.

### Why

Unit tests had coverage for happy-path leasing and retry but did not prove the failure/restart properties that expensive batches need. Small executable probes make the defects reproducible for reviewers and form templates for the permanent regression suite.

### What worked

- Independent sibling operations continued after one permanent failure.
- Durable queue admission allowed the probes to create and inspect leases as expected.
- The probes were self-contained: temporary SQLite files, no network/provider calls, and cleanup through `defer os.RemoveAll`.

Observed output:

```text
first cycle: processed=2 succeeded=0 failed=0
workflow=failed batch-a=failed batch-b=succeeded finalize=canceled
after manual retry: processed=1 succeeded=0 failed=0
workflow=failed batch-a=succeeded batch-b=succeeded finalize=canceled
```

```text
processed=3 max_active=1 elapsed_ms=317
```

```text
after two heartbeats: status=running lease=worker=worker-1 expires=2026-07-20T18:00:02.123456789Z
re-leased: old_token=worker-1:... new_token=worker-2:...
stale completion error=<nil>
after stale completion: status=succeeded lease=worker=worker-2 expires=...
current completion error=<nil>
```

```text
expires=2026-07-20T18:00:01Z refresh_at=2026-07-20T18:00:01.5Z chronological_expired=true lexical_expired=false
refresh_changed=0 status=running lease_present=true
```

### What didn't work

- The first stale-lease probe used an integral fixed time and could not re-lease after the intended expiry because RFC3339Nano TEXT ordering caused the expiry comparison to fail. The exact error was:

```text
panic: second lease not acquired
```

  I changed the probe’s main stale-commit case to fixed nanosecond precision so it could isolate stale completion, then added the separate mixed-precision probe to record the timestamp ordering defect explicitly.

### What I learned

- Current heartbeat extensions are not cumulative when callers reuse the acquired lease object.
- `CompleteOp` and `FailOp` require transactional token validation before any result/status side effects.
- RFC3339Nano text serialization is unsafe for SQL ordering because fractional precision is variable-width.
- Current cycle counts are not usable as progress facts.

### What was tricky to build

The stale-lease test required separating two defects. If the probe used `...01Z` and `...01.5Z`, SQLite failed to recover the expired lease due to lexicographic comparison, masking the stale-commit defect. Using a fixed nine-digit fractional timestamp allowed re-leasing and proved stale commit independently. The final ticket keeps both probes so future changes cannot accidentally fix only one layer.

### What warrants a second pair of eyes

- Review the proposed epoch-microsecond migration versus a fixed-width textual format.
- Review whether all existing query predicates that compare timestamps have been inventoried before migration work begins.
- Review whether stale result artifacts could already have escaped to an external store before engine commit; external executors need idempotent artifact identity.

### What should be done in the future

- Convert probes into package tests before refactoring implementation.
- Add stale failure and heartbeat-supervisor cancellation probes.

### Code review instructions

Run:

```bash
cd /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper
for f in ttmp/2026/07/20/SCRAPER-RESUMABLE-WORKFLOW-HARDENING--harden-scraper-for-long-running-resumable-batch-workflows/scripts/*.go; do
  GOWORK=off go run "$f"
done
```

Then compare the results with `pkg/engine/store/sqlite/lease_store.go`, `result_store.go`, and `op_store.go`.

### Technical details

- The stale token is deleted conditionally today, but results/status are mutated unconditionally before that deletion is checked.
- `HeartbeatLease` currently computes expiry from the caller’s original `Lease.ExpiresAt`.
- Lease/retry SQL stores/computes timestamps as RFC3339Nano TEXT.

## Step 3: Validate the Documentation Package and Baseline Test Mode

I validated the new ticket documents and ran the focused scraper engine/workflow/runtime-event suites. The ticket structure passed cleanly. The source test suite first failed in the newly added workspace because its local `goja` checkout is incompatible with scraper's pinned `goja_nodejs`; rerunning in scraper's normal module mode with `GOWORK=off` passed every selected package.

This finding is recorded because future implementation validation must use `GOWORK=off` until the workspace dependency versions are intentionally reconciled. It is unrelated to the ticket documentation and does not change the engine findings.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Validate the ticket deliverable and leave an intern with a reproducible test command.

**Inferred user intent:** The guide should be usable rather than merely descriptive, including a reliable baseline command.

### What I did

- Validated frontmatter for index, guide, and diary.
- Ran `docmgr doctor --ticket SCRAPER-RESUMABLE-WORKFLOW-HARDENING --stale-after 30`.
- Ran `git diff --check` for the ticket.
- Attempted focused tests in workspace mode:

```text
go test ./pkg/workflow ./pkg/engine/... ./pkg/services/engineview ./pkg/runtimeevents/... -count=1
```

- Reran successfully in module mode:

```text
GOWORK=off go test ./pkg/workflow ./pkg/engine/... ./pkg/services/engineview ./pkg/runtimeevents/... -count=1
```

### Why

The workspace now includes a local `goja` checkout. Scraper's pinned dependency versions need to be tested in the mode that resolves their declared module graph, otherwise unrelated API incompatibilities obscure the scheduler baseline.

### What worked

- All ticket frontmatter checks passed.
- `docmgr doctor` reported all checks passed.
- `GOWORK=off` focused suites passed for workflow, engine, engineview, runtimeevents, and sessionstream packages.
- Ticket markdown passed `git diff --check`.

### What didn't work

Workspace-mode test compilation failed with the local Goja/nodejs mismatch:

```text
../../../../go/pkg/mod/github.com/dop251/goja_nodejs@v0.0.0-20260212111938-1f56ff5bcf14/goutil/argtypes.go:14:10: undefined: goja.IsNumber
../../../../go/pkg/mod/github.com/dop251/goja_nodejs@v0.0.0-20260212111938-1f56ff5bcf14/goutil/argtypes.go:81:11: undefined: goja.IsBigInt
```

`GOWORK=off` resolved scraper's declared `github.com/dop251/goja v0.0.0-20260311135729-065cd970411c` and passed.

### What I learned

- The workspace contains `./goja`, which overrides scraper’s pinned version in workspace mode.
- `GOWORK=off` is the correct current validation command for scraper until a deliberate workspace dependency alignment is performed.

### What was tricky to build

The test failure superficially looked like a scraper regression because it happened while compiling scheduler-adjacent packages. The undefined symbols were in `goja_nodejs` against a different local `goja` API, so isolating the module graph was necessary before drawing any conclusion about the engine.

### What warrants a second pair of eyes

- Review whether the workspace should ultimately align its local `goja` and `goja_nodejs` versions. This is not part of the scraper hardening scope and should not be changed opportunistically here.

### What should be done in the future

- Use `GOWORK=off` for scraper CI-style engine validation until the workspace is reconciled.
- Add the focused regression tests described in the guide, then extend validation to race and upgrade suites.

### Code review instructions

```bash
cd /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper
GOWORK=off go test ./pkg/workflow ./pkg/engine/... ./pkg/services/engineview ./pkg/runtimeevents/... -count=1
```

### Technical details

- Workspace `go.work` uses local `./goja` and `./go-go-goja` modules.
- Scraper `go.mod` pins `goja v0.0.0-20260311135729-065cd970411c`, `goja_nodejs v0.0.0-20260212111938-1f56ff5bcf14`, and `go-go-goja v0.8.3`.
