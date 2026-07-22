---
Title: Workflow V3 External Operation Ledger Completion Audit
Ticket: SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS
Status: active
Topics:
    - workflow-v3
    - durability
    - observability
    - privacy
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system/cmd/rag-ttc-v3-sweep/main.go
      Note: Per-cell custody and researchctl export
    - Path: repo://pkg/workflowv3/external_operation.go
      Note: Privacy-safe external operation contract
    - Path: repo://pkg/workflowv3sqlite/external_operation.go
      Note: Durable admission/completion and ticket fencing
ExternalSources: []
Summary: Evidence-backed audit of the durable external-operation ledger; blocked only on explicit paid real-provider authority.
LastUpdated: 2026-07-22T17:20:35.33864348-04:00
WhatFor: Prevent premature completion and provide the exact evidence/recheck plan for real qualification.
WhenToUse: Before requesting authority, running a paid qualification, or closing SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS.
---


# Completion Audit — Active, Not Complete

## Verified implementation requirements

| Requirement | Evidence |
| --- | --- |
| Closed privacy-safe operation contracts | `pkg/workflowv3/external_operation.go`, contract tests, commit `b637095` |
| Durable admission/completion and ticket/lease fencing | SQLite schema/store and tests; commits `1542075`, `e061769` |
| Trusted Go-only recorder injection | `pkg/workflowv3runtime/{engine,modules,task_runner}.go`, commits `80ef254`, `16860ef` |
| Bounded projections and canonical atomic JSONL/manifest export | `pkg/workflowv3sqlite/{external_operation_query,operational}.go`, commit `27efa9e` |
| Attempt budget reconciliation | `pkg/workflowv3sqlite/budget.go`, commit `b8857b1` |
| RAG generation/embedding operation spans | RAG `internal/workflowv3ttc/provider.go`, commits `b728e0a`, `3bde483`, `0147ea2` |
| Per-cell success/failure custody and reductions | RAG sweep commits `9603d6f`, `4b0f2bf` |
| Researchctl verified artifact and scalar-metric custody | RAG commits `88d846d`, `4ac4ca4`; fresh import staged 37 artifacts and 4 metrics |
| Cancellation, lease-loss, reopen, idempotency, concurrency, privacy tests | scraper commits `c9e69c4`, `1d31270`, plus focused and race suites |

## Fresh validation evidence

- `GOWORK=off go test ./... -count=1` passed in scraper (Step 22).
- `GOWORK=off golangci-lint run ./...` passed in scraper (Step 22).
- Full normal and race suites passed for `pkg/workflowv3sqlite` and `pkg/workflowv3runtime` (Step 21).
- RAG pre-push hooks passed after operation-custody wiring: lint, core tests, TypeScript typecheck, and GoReleaser snapshot.
- Twelve-cell fixture sweep produced 282 operation rows, per-cell JSONL/manifests, and a verified researchctl import.
- Byte-level fixture custody canary found none of source text, sensitive provider sentinel, auth/bearer material, URLs, `content_text`, or provider config names.
- `docmgr doctor` and `git diff --check` passed at each focused publication point.

## Unmet requirement and blocker

The paid real-provider smoke/qualification is not verified. It must not run without explicit authority. The exact requested cumulative envelope is:

- 129 generation requests;
- 1,373,850 microunits;
- 2,113,536 input tokens;
- 1,056,768 output tokens;
- 128 embedding requests;
- 3,932,160 embedding tokens; and
- at most four concurrent Umans calls.

This includes the 61 previously admitted generation requests and explicit retry headroom. Required input to unblock: an affirmative authorization for exactly that envelope, plus the operator-owned real provider configuration and canonical researchctl specification already required by the sweep.

## Required post-authorization evidence

1. Run one bounded real qualification only; do not auto-repeat.
2. Verify every successful/failed cell's operation JSONL and manifest before runtime cleanup.
3. Run privacy canaries over evidence, exports, retained SQLite/WAL, logs, reports, and graphs.
4. Import the generated custody export into a fresh researchctl lab and verify checks/metrics.
5. Produce and visually inspect derived graphs/report, update the diary/changelog/tasks, then repeat this audit.

Until then, this ticket and the active goal remain **not complete**.
