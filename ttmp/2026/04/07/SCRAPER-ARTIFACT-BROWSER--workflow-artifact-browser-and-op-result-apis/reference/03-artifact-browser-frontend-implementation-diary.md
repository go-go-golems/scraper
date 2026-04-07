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
LastUpdated: 2026-04-07T22:00:00-04:00
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

## Step 5: ArtifactPreviewPanel  [committed: de02d69]

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Add the right-half preview panel with metadata header, action buttons, and content dispatch by MIME type.

**Commit (code):** de02d69 — "feat(ArtifactPreviewPanel): add preview panel with body fetch + dispatch by content type"

### What I did

- Created `ArtifactPreviewPanel.tsx`: metadata header (name, op link, kind, contentType, size, created), action bar (Open Raw, Download), body fetch from `/api/v1/artifacts/{id}` on `artifact` prop change, dispatches content to `ArtifactPreview` (HTML/JSON/text) or `<img>` (images) or `BinaryFallbackView` (binary). Close button wired.
- Created `BinaryFallbackView.tsx`: `InsertDriveFile` icon + size + contentType + download button for non-previewable artifacts.
- Created `ArtifactPreviewPanel.stories.tsx`: Empty, JsonArtifact, HtmlArtifact, BinaryArtifact.
- Updated `ArtifactsPanel`: owns `previewVisible` state, renders `ArtifactPreviewPanel` below the table (bordered card), toggle via `ViewAgenda` icon button in the pagination bar.

### Key decisions

- **Fetch on `artifact` change**: `useEffect` watches `artifact.id` and fetches `/api/v1/artifacts/{id}` as text. `loading` state shown during fetch.
- **Content dispatch**: `isPreviewable` check uses `artifact.previewable` + `contentType` check. Images use `<img src>` (browser handles fetch). Binary uses `BinaryFallbackView`.
- **Preview panel layout**: rendered below the table in a bordered `Box`. Toggle hides/shows the entire panel. `selectedArtifactId` state in `ArtifactsPanel` — `ArtifactPreviewPanel` is purely presentational.
- **Open Raw**: `component="a" href={...} target="_blank"` — opens raw artifact in a new tab.

### What warrants a second pair of eyes

The `fetch` in `ArtifactPreviewPanel` fetches raw text. For large binary files, this would fail. The `artifact.previewable` field gates this, but worth verifying that very large HTML responses (e.g., 10MB) don't cause memory issues — `ArtifactPreview` renders them as a `<pre>` which could be slow.

### What should be done in the future

- N/A

### Code review instructions

Start at `ArtifactPreviewPanel.tsx`. Verify the `useEffect` dependency on `artifact?.id` is correct (fetch re-fires when artifact changes). Check that `BinaryFallbackView` is shown for non-previewable artifacts. Validate: `cd web && npx tsc --noEmit`.

---

## Step 6: Bridge links  [committed: ad44040]

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Add the bidirectional links: "See all artifacts from this op" in OpResultTab, and "→ Op detail" in ArtifactPreviewPanel.

**Commit (code):** ad44040 — "feat(ArtifactsPanel): add bridge links + tab structure to WorkflowDetailPage"

### What I did

- `WorkflowDetailPage`: added `activeTab` state (`'ops'|'artifacts'`), `artifactFilterOpId` state for bridge navigation. Replaced the separate Artifacts + Ops cards with a single tabbed card (`Tabs` + `Tab` for Ops/Artifacts). `OpDetailDrawer` is always shown when an op is selected, regardless of which tab is active.
- `ArtifactsPanel`: added `initialOpIdFilter?: string` prop. Initializes `filters.opId` to this value so the Op filter is pre-filled when navigating from OpResultTab.
- `OpResultTab`: added `onBrowseArtifacts?: (opId: string) => void` prop. Renders "See all N artifacts from this op" link when the op has artifacts.
- `OpDetailDrawer`: accepts and passes `onBrowseArtifacts` to `OpResultTab`.
- `ArtifactsPanel`: added `onNavigateToOp` prop to `ArtifactPreviewPanel` (forward link from artifact → op detail). Currently wired to navigate to Ops tab when clicked — `selectedOpId` + `drawerOpen` are in `WorkflowDetailPage`, not in `ArtifactsPanel`, so the forward bridge needs more state lifting. Backward bridge (OpResultTab → Artifacts) is fully wired.

### Key decisions

- **Backward bridge (OpResultTab → Artifacts)**: Fully working. Clicking "See all artifacts from this op" in the op drawer Result tab switches to Artifacts tab + pre-fills the Op filter.
- **Forward bridge (artifact → op detail)**: Partial. `ArtifactPreviewPanel` has the `onNavigateToOp` callback, but `ArtifactsPanel` needs to signal `WorkflowDetailPage` to open the drawer for that op. Needs `handleSelectOp` to be passed in. Left as a follow-up.
- **Tabs**: Simple `Tabs` component with Ops + Artifacts. Runtime Events stays as a separate card above the tabs (as per the design doc).

### What was tricky to build

The bidirectional nature of the bridge. The backward bridge (OpResultTab → Artifacts) is straightforward: `WorkflowDetailPage` owns all the state and passes callbacks down. The forward bridge (artifact → op detail) requires `ArtifactsPanel` to signal upward to `WorkflowDetailPage` to open the drawer for a different op than the currently selected one. Since `ArtifactsPanel` doesn't own `selectedOpId`, this requires passing `handleSelectOp` into it — worth doing as a follow-up.

### What warrants a second pair of eyes

The forward bridge (`ArtifactPreviewPanel` "→ Op detail" → opens drawer for that artifact's op) is not wired. `ArtifactPreviewPanel.onNavigateToOp` is defined but not connected. Verify this is intentional and track as follow-up.

### What should be done in the future

- Wire the forward bridge: pass `handleSelectOp` into `ArtifactsPanel` → `ArtifactPreviewPanel.onNavigateToOp`.

### Code review instructions

Start at `WorkflowDetailPage.tsx` — verify `activeTab` state and the `handleBrowseArtifacts` callback. Verify `ArtifactsPanel.initialOpIdFilter` seeds the filter correctly. Verify `OpResultTab.onBrowseArtifacts` renders only when artifacts exist. Validate: `cd web && npx tsc --noEmit`.

---

## Step 7: Storybook stories

## Bug fix: Storybook crashes — missing RTK Query store + MSW handlers  [committed: 8320152]

### What happened

Storybook stories for `ArtifactsPanel` crashed on load with three errors:

1. `"Middleware for RTK-Query API at reducerPath 'workflowApi' has not been added to the store"` — `preview.tsx` mock store only had `runtimeEventsApi`
2. `"searchInputValue is not defined"` — misleading error triggered by RTK Query init failure
3. `GET /api/v1/artifacts/art-1 404` — no MSW handlers for artifact endpoints

### Root cause

- `preview.tsx` mock store was missing `workflowApi`, `catalogApi`, `engineApi`, `queueApi`, `submissionApi` reducers and middleware
- Stories had no `parameters.msw.handlers`, so real HTTP calls fired and failed

### Fix

- `preview.tsx`: added all 6 RTK Query APIs to the mock store
- `web/src/stories/msw/handlers.ts` (new): shared MSW handlers with fixture data — `defaultArtifactHandlers` (4 artifacts + 3 ops) and `emptyArtifactHandlers`
- `ArtifactsPanel.stories.tsx`: uses shared handlers
- `ArtifactPreviewPanel.stories.tsx`: uses shared handlers, consistent artifact IDs

### Also done

- Removed forward bridge link (`onNavigateToOp`) from `ArtifactPreviewPanel` — the Op column in the artifact table already opens the op drawer, so the shortcut adds no value

Full bug report: `reference/04-bug-report-storybook-msw-and-redux-store.md`

See the existing story files:
- `web/src/components/artifacts/ArtifactsPanel.stories.tsx`
- `web/src/components/artifacts/ArtifactTable.stories.tsx`
- `web/src/components/artifacts/ArtifactPreviewPanel.stories.tsx`
- `web/src/components/artifacts/FilterBar.stories.tsx`
- `web/src/components/artifacts/ActiveFilterChips.stories.tsx`
