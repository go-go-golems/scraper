---
Title: Investigation diary
Ticket: SCRAPER-ARTIFACT-BROWSER
Status: active
Topics:
    - scraper
    - backend
    - frontend
    - http-api
    - artifacts
    - workflows
    - onboarding
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Records the evidence and implementation progress for workflow artifact browsing and op-result APIs."
LastUpdated: 2026-04-07T15:36:00-04:00
WhatFor: "Resume backend implementation without re-discovering the current artifact/result seams."
WhenToUse: "Use when continuing the ticket or reviewing why the ticket is scoped to artifacts/results but not JS replay."
---

# Investigation diary

## Goal

Split the earlier broad debugger design into a smaller, shippable ticket that gives immediate value:

- workflow-level artifact browsing,
- artifact download and preview support,
- and a real op-result backend endpoint.

The explicit decision is to defer JS replay/debug execution to a later ticket.

## What I inspected

I reviewed the current seams in:

- `pkg/api/handlers/engine.go`
- `pkg/api/server/server.go`
- `pkg/services/engineview/service.go`
- `pkg/api/types/types.go`
- `pkg/engine/store/sqlite/store.go`
- `web/src/api/workflowApi.ts`
- `web/src/components/workflows/OpDetailDrawer.tsx`

## What exists already

### Per-op artifacts

The backend already supports:

- `GET /api/v1/workflows/{workflowID}/ops/{opID}/artifacts`
- `GET /api/v1/artifacts/{artifactID}`

This is enough for the existing op detail drawer but not for a workflow artifact browser.

### Artifact storage

Artifacts are already durably stored in the engine SQLite DB and loaded through the engine store. The current service layer already exposes artifact summaries and artifact download detail.

### Frontend expectation for op result

The frontend already defines:

`GET /api/v1/workflows/{workflowID}/ops/{opID}/result`

in `web/src/api/workflowApi.ts`.

That makes the missing backend route more than a nice-to-have. It is a contract gap.

## Main findings

### 1. The new browser should be workflow-scoped, not op-scoped

Operators and site authors first ask:

- what artifacts exist for this workflow?
- which op produced them?
- which one is the HTML body, JSON summary, or execution log?

That workflow-wide view does not exist yet.

### 2. The op-result route should be implemented before new UI work

The op detail drawer already wants result data. A real backend endpoint gives us:

- a stable contract,
- one place to normalize empty/missing-result behavior,
- and a better foundation for later browser and debugger work.

### 3. This ticket should not take on JS replay

Replay is a different risk profile:

- execution semantics,
- sandbox/read-only rules,
- output capture,
- and possible divergence from live runtime behavior.

That would slow down the artifact browser work. It is better to land artifact/result access first.

## Immediate implementation plan

1. Add workflow-level artifact listing in the engine view service.
2. Add a handler and route for workflow-level artifact listing.
3. Add a service-backed op-result retrieval method.
4. Add a handler and route for op-result retrieval.
5. Add tests at the service and HTTP/server levels.

## Notes for later

If a later replay ticket is created, it should build on top of the artifact/result contracts from this ticket rather than bypass them.

## Implementation log

### Backend slice completed

I implemented the first backend slice in these files:

- `pkg/services/engineview/service.go`
- `pkg/services/engineview/service_test.go`
- `pkg/api/types/types.go`
- `pkg/api/handlers/engine.go`
- `pkg/api/server/server.go`
- `pkg/api/server/server_test.go`
- `web/src/api/workflowApi.ts`

### What changed

#### Workflow artifact listing

Added a workflow-level service method that:

- checks whether the workflow exists,
- queries all artifacts for that workflow,
- orders them by `created_at`, `op_id`, and `id`,
- returns a workflow-scoped result envelope.

#### Op result retrieval

Added a service-backed op-result method that:

- checks whether the op exists for the workflow,
- reuses the engine store’s `GetResult(...)` logic,
- returns `exists=false` when the workflow/op pair is missing,
- returns `result=nil` when the op exists but no result has been written yet.

#### Handler and route additions

Added:

- `GET /api/v1/workflows/{workflowID}/artifacts`
- `GET /api/v1/workflows/{workflowID}/ops/{opID}/result`

#### Frontend compatibility fix

Because the new op-result endpoint returns `{ "result": ... }`, I updated the existing RTK Query endpoint to unwrap that response with `transformResponse`.

### Validation commands

```bash
go test ./pkg/services/engineview ./pkg/api/server -count=1
go test ./... -count=1
docmgr doctor --ticket SCRAPER-ARTIFACT-BROWSER --stale-after 30
```

### Validation notes

- Both targeted backend packages passed.
- Full Go test suite passed.
- The ticket validates after adding the new `artifacts` and `workflows` vocabulary slugs.
- `npm run build` still fails for pre-existing Storybook/type issues elsewhere in `web/`. The errors are not caused by this backend slice.
