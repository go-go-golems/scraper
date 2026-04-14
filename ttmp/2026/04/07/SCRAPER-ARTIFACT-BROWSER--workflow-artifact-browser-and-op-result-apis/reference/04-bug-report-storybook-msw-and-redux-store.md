---
Title: "Bug: Storybook stories crash — missing RTK Query store + MSW handlers"
Ticket: SCRAPER-ARTIFACT-BROWSER
Status: done
Topics:
    - bug
    - frontend
    - storybook
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/.storybook/preview.tsx"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/stories/msw/handlers.ts"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/artifacts/ArtifactsPanel.stories.tsx"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/artifacts/ArtifactPreviewPanel.stories.tsx"
Summary: "ArtifactsPanel stories crashed on load: (1) workflowApi missing from mock Redux store in preview.tsx, (2) no MSW handlers for workflow artifacts + artifact body endpoints."
LastUpdated: 2026-04-07T22:30:00-04:00
WhatFor: "Track the storybook crash root cause and fix so stories render correctly."
WhenToUse: "Reference when writing new artifact browser stories or adding MSW handlers."
---

# Bug: Storybook stories crash — missing RTK Query store + MSW handlers

## Summary

`ArtifactsPanel` stories crashed on load with two errors:

1. **"Middleware for RTK-Query API at reducerPath 'workflowApi' has not been added to the store"**
   — `preview.tsx` only included `runtimeEventsApi` in the mock Redux store. All other RTK Query APIs were missing.

2. **`searchInputValue is not defined`**
   — Triggered as a consequence of #1. Because `workflowApi` middleware was missing, RTK Query threw during initialization, which caused React to render an error boundary. The error message in the error boundary was misleading — the real issue was the missing middleware.

3. **`GET /api/v1/artifacts/art-1 404`** (MSW)
   — Stories had no MSW handlers for artifact body endpoints, so every preview attempt resulted in a 404.

## Root cause

### 1. Incomplete mock Redux store

`preview.tsx` created a mock store with only `runtimeEventsApi`:

```typescript
function createMockStore() {
  return configureStore({
    reducer: {
      ui: uiSlice.reducer,
      [runtimeEventsApi.reducerPath]: runtimeEventsApi.reducer,
      // ❌ workflowApi, catalogApi, engineApi, queueApi, submissionApi missing
    },
    middleware: (getDefault) =>
      getDefault({ serializableCheck: false }).concat(runtimeEventsApi.middleware),
      // ❌ workflowApi.middleware missing
  });
}
```

Every component that uses `workflowApi`, `catalogApi`, etc. would fail at runtime.

### 2. No MSW handlers

Stories for `ArtifactsPanel`, `ArtifactPreviewPanel`, etc. had no `parameters.msw.handlers`, so:
- `useGetWorkflowArtifactsQuery` → real HTTP call to non-existent dev server → fails
- `useGetWorkflowArtifactsQuery` → XHR 404 for artifact bodies → no preview

## Fix

### 1. `web/.storybook/preview.tsx`

Added all RTK Query APIs to the mock store:

```typescript
function createMockStore() {
  return configureStore({
    reducer: {
      ui: uiSlice.reducer,
      [runtimeEventsApi.reducerPath]: runtimeEventsApi.reducer,
      [workflowApi.reducerPath]: workflowApi.reducer,
      [catalogApi.reducerPath]: catalogApi.reducer,
      [engineApi.reducerPath]: engineApi.reducer,
      [queueApi.reducerPath]: queueApi.reducer,
      [submissionApi.reducerPath]: submissionApi.reducer,
    },
    middleware: (getDefault) =>
      getDefault({ serializableCheck: false })
        .concat(runtimeEventsApi.middleware)
        .concat(workflowApi.middleware)
        .concat(catalogApi.middleware)
        .concat(engineApi.middleware)
        .concat(queueApi.middleware)
        .concat(submissionApi.middleware),
  });
}
```

### 2. `web/src/stories/msw/handlers.ts` (new file)

Created a shared MSW handlers file with fixture data and default handler sets:

- `defaultArtifactHandlers` — mocks `GET /workflows/{id}/artifacts`, `GET /workflows/{id}/ops`, and artifact body endpoints (`story-art-json`, `story-art-html`, `story-art-log`)
- `emptyArtifactHandlers` — mocks empty artifact list + empty ops list

Stories import from this file and extend as needed.

### 3. `ArtifactsPanel.stories.tsx`

Now uses `defaultArtifactHandlers` / `emptyArtifactHandlers` via `parameters.msw.handlers`.

### 4. `ArtifactPreviewPanel.stories.tsx`

Updated to use consistent artifact IDs matching `handlers.ts` (e.g., `story-art-json` instead of `art-1`). Added `parameters.msw.handlers: defaultArtifactHandlers`.

## Files changed

| File | Change |
|---|---|
| `web/.storybook/preview.tsx` | Added all 6 RTK Query APIs to mock store |
| `web/src/stories/msw/handlers.ts` | New — shared MSW handlers + fixture data |
| `web/src/components/artifacts/ArtifactsPanel.stories.tsx` | Uses shared handlers |
| `web/src/components/artifacts/ArtifactPreviewPanel.stories.tsx` | Uses shared handlers, consistent IDs |

## Validation

```bash
cd web && npx tsc --noEmit   # clean
# Open Storybook: yarn storybook
# Check Artifacts/ArtifactsPanel stories render without errors
# Check Artifacts/ArtifactPreviewPanel stories render without 404s
```

## Lesson learned

When adding a new RTK Query API, update `preview.tsx` mock store to include it. Otherwise stories using that API will crash silently or with misleading error messages.
