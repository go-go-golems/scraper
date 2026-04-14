---
Title: Investigation diary
Ticket: SCRAPER-ARTIFACT-BROWSER
Status: done
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
LastUpdated: 2026-04-07T15:47:00-04:00
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

## Second backend slice

After the foundation endpoints were in place, I expanded the workflow artifact endpoint so it is actually suitable for a browser view instead of being a raw dump.

### Added contract improvements

- `total` count in the workflow artifact list response
- server-side filtering by:
  - `opId`
  - `kind`
  - `contentType`
  - `search`
- pagination via:
  - `limit`
  - `offset`
- preview hints on each artifact summary:
  - `previewable`
  - `previewKind`

### Why this matters

Without those fields, the future browser UI would need to:

- fetch everything up front,
- re-derive preview behavior client-side,
- and handle large workflows awkwardly.

With the current contract, the browser can render:

- filtered workflow-local artifact tables,
- smarter previews for JSON, HTML, and text,
- and cheap server-side paging for larger workflows.

### Additional validation

I extended both service and server tests to verify:

- `total` reflects filtered and unfiltered result sets
- preview hints are populated for HTML and JSON artifacts
- query filtering by `opId` and `search` works as expected

## Phase 2: Frontend UI design

### What happened

After the backend slice landed, I wrote the full frontend UI design as `design/02-artifact-browser-frontend-ui-design.md`.

### What the design doc covers

- 5 annotated ASCII screen screenshots (full browser view, HTML preview, image preview, binary fallback, empty state)
- YAML DSL component hierarchy (`ArtifactsPanel` → `FilterBar` → `ArtifactTable` → `ArtifactPreviewPanel`)
- Complete API mapping (endpoint → component → query param → state)
- New RTK Query endpoint definition (`getWorkflowArtifacts`)
- Implementation order: wire query → skeleton → filter bar → pagination → preview panel → bridge → stories
- Component inventory: 7 new components, 4 existing components to reuse

### Backend is done, frontend is not

Confirmed by grep:
- `GET /api/v1/workflows/{workflowID}/artifacts` — registered in `routes_engine.go`, handler in `engine.go`, service method in `service.go` ✅
- `GET /api/v1/workflows/{workflowID}/ops/{opID}/result` — same ✅
- `useGetWorkflowArtifactsQuery` in `workflowApi.ts` — **not present** ❌

Frontend work is the remaining Phase 2. See `design/02-artifact-browser-frontend-ui-design.md` for the full spec.

## Bug: nil slice crash in OpResultTab

### What happened

While testing the workflow detail page (`/workflows/:id`), opening an op drawer and clicking the "Result" tab crashed the app with:

```
TypeError: can't access property "length", result.EmittedIDs is null
    OpResultTab OpResultTab.tsx:63
```

The affected op was `hackernews-extract-frontpage-1775586649974859668:frontpage-extract:page-2-fetch`.

### Root cause

Go's `encoding/json` serializes **nil slices as `null`**, not `[]`. In `model.OpResult`, fields `Records`, `Artifacts`, `Emitted`, and `EmittedIDs` were left nil when:

1. The scheduler creates a minimal fallback `OpResult` (only `OpID` + `CompletedAt`).
2. The DB columns are NULL and `unmarshalJSON` leaves Go slices as nil.

The TypeScript type declared them as non-nullable `string[]`, so `.length` threw.

### Fix

**`pkg/engine/store/sqlite/result_store.go`**: after loading from DB, normalize nil slices to empty slices:

```go
if result.Records == nil { result.Records = []model.RecordWrite{} }
if result.Emitted == nil { result.Emitted = []model.OpSpec{} }
if result.EmittedIDs == nil { result.EmittedIDs = []model.OpID{} }
```

**`web/src/components/workflows/op-detail/OpResultTab.tsx`**: defensive optional chaining:

```ts
result.EmittedIDs?.length ?? 0
result.Artifacts?.length ?? 0
```

### Files changed

- `pkg/engine/store/sqlite/result_store.go` — nil-guard for Records, Emitted, EmittedIDs after DB load
- `web/src/components/workflows/op-detail/OpResultTab.tsx` — `?.length ?? 0` safety net

### Validation

```bash
go test ./... -count=1   # all pass
npx tsc --noEmit         # no type errors
```

### Also created

- `reference/02-bug-report-nil-slice-crash.md` — full bug report for long-term reference.
