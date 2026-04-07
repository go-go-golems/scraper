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
    - Path: web/src/components/artifacts/ActiveFilterChips.tsx
      Note: ActiveFilterChips (commit 633bbb1)
    - Path: web/src/components/artifacts/ArtifactsPanel.tsx
      Note: Created ArtifactsPanel skeleton (commit e0410e4)
    - Path: web/src/components/artifacts/FilterBar.tsx
      Note: FilterBar + debounced search (commit 633bbb1)
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

## Step 2: ArtifactsPanel skeleton  [committed: e0410e4]

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Create `ArtifactsPanel` as the root component, wire it to `useGetWorkflowArtifactsQuery`, and add it to `WorkflowDetailPage`.

**Inferred user intent:** Get the skeleton rendering on screen before layering on complexity.

**Commit (code):** e0410e4 — "feat(ArtifactsPanel): add ArtifactsPanel skeleton wired to useGetWorkflowArtifactsQuery"

### What I did

- Created `web/src/components/artifacts/ArtifactsPanel.tsx` — owns `page`/`limit` state, calls `useGetWorkflowArtifactsQuery`, renders loading/error/empty/full states. Comment stubs mark where Steps 3-5 will land.
- Created `web/src/components/artifacts/ArtifactsPanel.stories.tsx` — minimal skeleton story + empty state story.
- Added `ArtifactsPanel` to `WorkflowDetailPage` as a new "Artifacts" card below Runtime Events. Added a `Divider` to visually separate the future filter bar slot.

### Key decisions

- **Tab structure deferred**: The design doc specifies `Overview | Ops | Runtime | Artifacts | Site Info` tabs. Adding them now would require migrating the Ops and Runtime Events sections into tab panels, which is a larger change. Added the Artifacts card at the bottom for now; the tab structure will be added as part of Step 3 (filter bar) since both touch the same area.
- **Preview panel deferral**: The preview panel (right half of the split pane) is intentionally skipped in the skeleton. It requires per-artifact body fetching and is a larger lift. Step 5 adds it.
- **Page state in component**: `page` and `limit` are local `useState` for now. When pagination is wired in Step 4, these will be lifted or managed via URL params.

### What worked

- `npx tsc --noEmit` clean.
- `ArtifactsPanel` correctly uses `skip: !workflowId` so it won't fire until the page has a workflow ID.

### What was tricky to build

The trade-off between adding tabs now vs. keeping the change minimal. Chose minimal: a new card at the bottom is safe and lets the query fire and return real data without restructuring the page. The tab shell will come back when the filter bar (Step 3) is added.

### What warrants a second pair of eyes

The tab structure decision — if tabs are required for the final design, we should add them in Step 3 rather than later.

### What should be done in the future

- Add tab bar structure to `WorkflowDetailPage` when Step 3 (filter bar) is implemented.

### Code review instructions

Start at `ArtifactsPanel.tsx`. Verify the query params (`limit=20, offset=0`) match the design doc. Verify the `skip` condition (`!workflowId`) is correct. Validate: `cd web && npx tsc --noEmit`.

---

## Step 3: FilterBar + ActiveFilterChips  [committed: 633bbb1]

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Add the filter bar with Op dropdown, Kind, Content-Type, and debounced search.

**Commit (code):** 633bbb1 — "feat(FilterBar): add FilterBar + ActiveFilterChips with debounced search"

### What I did

- Created `FilterBar.tsx`: Op dropdown (from `useGetWorkflowOpsQuery`), Kind dropdown (static list), Content-Type dropdown (static list), debounced search via separate `onSearchChange` + `searchInputValue` props.
- Created `ActiveFilterChips.tsx`: dismissible chips for each active filter + "Clear all" button.
- Wired both into `ArtifactsPanel` — query params now include `opId`, `kind`, `contentType`, `search`.
- Created stories for `FilterBar`, `ActiveFilterChips`, and updated `ArtifactsPanel` story.

### Key decisions

- **Debounce split**: `searchInputValue` (live, tracks what the user is typing) and `filters.search` (debounced, applied after 300ms). This avoids a controlled-input freeze — without it, the TextField would freeze at the last debounced value while the user is still typing.
- **Op name map**: Built a `Record<opId, "Kind:shortId">` map so the Op filter shows readable names like `js:frontpage-extract` instead of raw IDs.
- **Static dropdown lists**: Kind and Content-Type are hard-coded based on known artifact kinds in the codebase (`http-response-body`, `json-output`, etc.). A future ticket could make these dynamic from the API.

### What was tricky to build

The controlled input + debounce conflict. `FilterBar` takes `filters.search` as the `value` of the TextField (controlled). If `onSearchChange` fires immediately and updates `filters.search`, the TextField value would update on every keystroke — defeating the purpose of debouncing. The fix: `searchInputValue` tracks live typing separately, and only `filters.search` (updated after 300ms debounce) goes to the query.

### What warrants a second pair of eyes

The debounce lives in `ArtifactsPanel` but the search input is in `FilterBar`. This split means `FilterBar` needs two props (`searchInputValue` + `onSearchChange`) which slightly increases coupling. An alternative would be to move debounce logic into `FilterBar` itself. Fine for now but worth revisiting if the pattern grows.

### What should be done in the future

- N/A

### Code review instructions

Start at `FilterBar.tsx`. Verify the `searchInputValue` / `onSearchChange` split matches the comment. Verify `ArtifactsPanel` passes all four filter fields to `useGetWorkflowArtifactsQuery`. Validate: `cd web && npx tsc --noEmit`.

---

## Step 4: ArtifactTable + pagination  [committed: 56b28bf]

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Replace the placeholder list in `ArtifactsPanel` with a proper `ArtifactTable` (MUI Table) and add pagination controls.

**Commit (code):** 56b28bf — "feat(ArtifactTable): add ArtifactTable with pagination controls"

### What I did

- Created `ArtifactTable.tsx`: MUI `Table` with Name/Op/Kind/Size/Actions columns. Row click selects artifact (for Step 5 preview). Action buttons: preview (`OpenInBrowser`) + download (`CloudDownload` as `<a>`).
- Updated `ArtifactsPanel`: now owns `selectedArtifactId` state (stub for Step 5), uses `ArtifactTable`, adds prev/next pagination controls wired to the `page` state → `offset` param.
- `Pagination` math: `startItem/endItem` from `offset+1` to `min(offset+limit, total)`, page count from `Math.ceil(total/limit)`.
- Created `ArtifactTable.stories.tsx`: Default, WithSelection, ManyRows, Empty.

### Key decisions

- **Icons**: `OpenInBrowser` (not `OpenInNew`) and `CloudDownload` (not `Download`) — confirmed available in `@mui/icons-material`.
- **`selectedArtifactId` lives in `ArtifactsPanel`**: `ArtifactTable` is a pure presentational table — it takes `selectedId` + `onSelectArtifact`. State management stays in the parent so Step 5 can wire it to the preview panel without changing the table contract.
- **Download as native `<a>`**: `component="a" href={...} download` on the IconButton. This is the simplest reliable download mechanism without needing programmatic fetch+blob.

### What was tricky to build

The `open in new tab` vs `download` distinction. For download, `target="_blank"` + `download` attribute is needed on the `<a>` tag. Using MUI's `IconButton component="a"` makes the whole button a link, which works cleanly.

### What warrants a second pair of eyes

The `component="a"` approach for download — verify that the `href` URL (`/api/v1/artifacts/{id}`) triggers a browser download rather than navigation. The backend should set `Content-Disposition: attachment` headers. If not, the browser might try to display HTML responses inline.

### What should be done in the future

- N/A

### Code review instructions

Start at `ArtifactTable.tsx`. Verify columns match the design doc. Check that `ArtifactsPanel` passes the right `offset = page * 20` to the query. Validate: `cd web && npx tsc --noEmit`.

---

## Step 5: ArtifactPreviewPanel

[TODO]

---

## Step 6: Bridge links

[TODO]

---

## Step 7: Storybook stories

[TODO]
