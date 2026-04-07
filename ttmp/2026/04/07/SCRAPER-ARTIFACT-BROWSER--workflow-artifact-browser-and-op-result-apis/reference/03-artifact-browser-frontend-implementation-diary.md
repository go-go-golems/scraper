---
Title: Artifact Browser Frontend — Implementation Diary
Ticket: SCRAPER-ARTIFACT-BROWSER
Status: active
Topics:
    - scraper
    - frontend
    - artifacts
    - workflows
    - http-api
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: web/src/api/types.ts
      Note: Added WorkflowArtifactListResponse type (commit 7834370)
    - Path: web/src/api/workflowApi.ts
      Note: Added getWorkflowArtifacts endpoint (commit 7834370)
    - Path: web/src/pages/WorkflowDetailPage.tsx
ExternalSources: []
Summary: Step-by-step implementation diary for the artifact browser frontend (Phase 2 of SCRAPER-ARTIFACT-BROWSER).
LastUpdated: 2026-04-07T21:10:00-04:00
WhatFor: Track frontend implementation progress, decisions, failures, and review notes.
WhenToUse: Resume frontend work without rediscovery. Reference when reviewing or handing off.
---



# Artifact Browser Frontend — Implementation Diary

## Goal

Implement the artifact browser frontend (Phase 2 of SCRAPER-ARTIFACT-BROWSER) step by step, following the design doc `design/02-artifact-browser-frontend-ui-design.md`.

## Step 1: Wire RTK Query endpoint `getWorkflowArtifacts`  [committed: 7834370]

### Prompt Context

**User prompt (verbatim):** "ok, Let's work on this task by task, committing at appropriate intervals, keeping a diary as we work."

**Assistant interpretation:** Implement the artifact browser frontend Phase 2 step by step, following the design doc. Start with wiring the RTK Query endpoint.

**Inferred user intent:** Get the frontend implementation done methodically with a clear audit trail via commits and diary.

**Commit (code):** 7834370 — "feat(workflowApi): add getWorkflowArtifacts RTK Query endpoint"

### What I did

- Added `getWorkflowArtifacts` as a new RTK Query endpoint to `workflowApi.ts`.
- Added `WorkflowArtifactListResponse` to `types.ts` to match the Go backend response shape.
- Added `'WorkflowArtifacts'` to `tagTypes` so future operations (e.g., replay) can invalidate the cache.

### Key decisions

- **Return type**: `WorkflowArtifactListResponse` — not `ArtifactSummary[]`. Reason: `total` is needed for pagination. RTK Query will cache the full response. Callers use `selectFromResult: (r) => r.data?.artifacts ?? []` for just the artifacts array, or access `r.data?.total` directly.
- **Query params**: `limit` defaults to 20, `offset` to 0 — matches design doc. All others optional.
- **Tag**: `{ type: 'WorkflowArtifacts', id: workflowId }` — same pattern as existing `getWorkflowOps`.

### What worked

- Build clean: `npx tsc --noEmit` passes with no errors.
- Go backend response confirmed via `routes_engine.go` and `artifact_read_service.go`: `WorkflowArtifactsResult` has `{ workflowID, artifacts[], total }` — exactly what the TS type describes.

### What didn't work

- Initial attempt used `transformResponse` to return just `ArtifactSummary[]` — this discards `total`, which the pagination UI needs. Fixed by returning the full `WorkflowArtifactListResponse` and using `selectFromResult` in components.

### What was tricky to build

The `transformResponse` vs. full response tradeoff. With `transformResponse` the return type is `ArtifactSummary[]` (simple), but `total` is lost. Without it, callers need to use `selectFromResult` or access `.data?.total` directly. Chose the latter since pagination is a first-class requirement.

### What warrants a second pair of eyes

Verify that all callers of `useGetWorkflowArtifactsQuery` consistently use `selectFromResult` or access `.data` correctly. A mismatch would silently give `undefined`.

### What should be done in the future

- N/A for this step.

### Code review instructions

Start at `workflowApi.ts` — `getWorkflowArtifacts` and `useGetWorkflowArtifactsQuery`. Verify `types.ts` `WorkflowArtifactListResponse` matches the Go `WorkflowArtifactsResult` in `artifact_read_service.go`. Validate: `cd web && npx tsc --noEmit`.

### Technical details

**Endpoint**: `GET /api/v1/workflows/{workflowId}/artifacts`
**Response**: `{ workflowID: string; total: number; artifacts: ArtifactSummary[] }`

```typescript
// Usage in a component:
const { data } = useGetWorkflowArtifactsQuery(
  { workflowId, opId, kind, contentType, search, limit: 20, offset: 0 },
  { skip: !workflowId }
);
// data?.artifacts  → ArtifactSummary[]
// data?.total      → number for pagination
```

---

## Step 2: ArtifactsPanel skeleton

[TODO]

---

## Step 3: FilterBar

[TODO]

---

## Step 4: ArtifactTable + pagination

[TODO]

---

## Step 5: ArtifactPreviewPanel

[TODO]

---

## Step 6: Bridge links

[TODO]

---

## Step 7: Storybook stories

[TODO]
