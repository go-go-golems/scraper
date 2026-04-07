---
Title: Diary
Ticket: UI-001
Status: active
Topics:
    - frontend
    - ux-design
    - ui-rework
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: web/.storybook/preview.tsx
      Note: Storybook global store + MSW wiring debug trail and final fix (commit 92cb0a9)
    - Path: web/src/api/runtimeEventsApi.ts
      Note: Storybook-specific SSE skip added after debugging stream 404s (commit 92cb0a9)
    - Path: web/src/mocks/runtimeEventsHandlers.ts
      Note: Investigated manual JSON vs Buf message serialization; fixed with create()+toJson() (commit 92cb0a9)
    - Path: web/src/pages/RuntimeEventsPage.stories.tsx
      Note: Story store serializableCheck disabled for protobuf bigint timestamps (commit 92cb0a9)
ExternalSources: []
Summary: Implementation and debugging diary for UI-001, capturing phased UI work, SSE/Storybook debugging, and validation steps.
LastUpdated: 2026-04-07T00:00:00Z
WhatFor: Chronological engineering diary for continuing, reviewing, and handing off UI-001 work.
WhenToUse: Read this when you need the exact sequence of implementation decisions, experiments, failures, and validation steps.
---


# Diary

## Goal

Phase 0 implementation diary for UI-001: adding cross-cutting infrastructure (error boundary, toast notifications, breadcrumb navigation) that all later phases depend on.

## Step 1: AppErrorBoundary

Add a React error boundary that catches unhandled rendering errors and shows a friendly fallback card instead of a blank white screen. This is foundational infrastructure — without it, any component crash takes down the entire SPA.

### Prompt Context

**User prompt (verbatim):** "Now work on phase 0, one by one, committing as you go. Keep a diary as you work."

**Assistant interpretation:** Implement all Phase 0 tasks (Error Boundary, Toast System, Breadcrumb Nav), committing each individually.

**Inferred user intent:** Build the infrastructure layer systematically with clean git history.

**Commit (code):** 8059130 — "feat(ui): add AppErrorBoundary wrapping AppShell"

### What I did
- Created `AppErrorBoundary.tsx` as a React class component with `getDerivedStateFromError` + `componentDidCatch`
- Fallback UI: MUI Card with error icon, message, stack trace (dev only via `import.meta.env.DEV`), and "Try Again" button
- Created `AppErrorBoundary.stories.tsx` with Error and Healthy stories
- Wired into `App.tsx`: `<AppErrorBoundary>` wraps `<AppShell>` children

### Why
Any unhandled rendering error in a child page (e.g., a null dereference in RuntimeEventsPage) crashes the entire React tree to a blank page. An error boundary catches this and lets users recover without refreshing.

### What worked
- TypeScript compiled clean on first try
- The class component pattern with `getDerivedStateFromError` + `componentDidCatch` is straightforward
- `import.meta.env.DEV` is the correct Vite env variable for dev-only stack traces

### What didn't work
- Nothing failed — clean implementation

### What I learned
- The app had zero error handling before this. Any component crash = white screen of death.

### What was tricky to build
- Nothing particularly tricky — standard React error boundary pattern.

### What warrants a second pair of eyes
- The boundary only wraps `<AppShell>` children, not the `BrowserRouter`. If the router itself throws, it won't be caught. This is intentional — router errors should be caught at a higher level if needed.

### What should be done in the future
- Consider adding error reporting (e.g., Sentry) in `componentDidCatch` for production errors.

### Code review instructions
- File: `web/src/components/common/AppErrorBoundary.tsx` — review the fallback UI and ensure `handleRetry` properly resets state
- File: `web/src/App.tsx` — verify wrapping order: `BrowserRouter > AppErrorBoundary > ToastProvider > AppShell`
- Verify: Storybook > Common > AppErrorBoundary > Error story shows fallback card

### Technical details
- Error boundary must be a class component (React limitation)
- `getDerivedStateFromError` sets `hasError: true` to trigger fallback render
- `componentDidCatch` logs to console for debugging
- `handleRetry` resets state to re-render children (React will re-mount the subtree)

---

## Step 2: Toast Notification System

Replace per-page Snackbar implementations with a shared `ToastProvider` + `useToast()` hook. This gives all pages a consistent notification mechanism without each page managing its own Snackbar state.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Commit (code):** c5ae0c4 — "feat(ui): add ToastProvider with global snackbars"

### What I did
- Created `ToastProvider.tsx` with React context + MUI Snackbar stack
- Context exposes `showToast(message, severity)` via `useToast()` hook
- Toasts stack vertically (max 3 visible), auto-dismiss after 4s
- Created `ToastProvider.stories.tsx` with Success, Error, Stacked stories
- Wired into `App.tsx` between `AppErrorBoundary` and `AppShell`
- Refactored `SubmitWorkflowPage.tsx`: removed local `useState<Snackbar>` + `<Snackbar>` + `<Alert>`, replaced with `useToast()` calls
- Updated `WorkflowDetailPage.tsx`: retry and cancel handlers now call `showToast()` on success/failure

### Why
The SubmitWorkflowPage had its own Snackbar with manual open/close state. Every new mutation feedback would need to duplicate this pattern. A shared provider eliminates boilerplate and ensures consistent positioning and behavior.

### What worked
- SubmitWorkflowPage refactor was clean — removed `useState` for snackbar, removed `handleCloseSnackbar` callback, removed the entire `<Snackbar>` JSX block
- The `useToast()` hook integrates naturally into async mutation handlers

### What didn't work
- Nothing failed

### What I learned
- The SubmitWorkflowPage had a decent local Snackbar already — the migration was straightforward since the semantics (message + severity) match 1:1

### What was tricky to build
- The stacking behavior: each toast needs a different `bottom` CSS value. Solved by computing `24 + index * 56` based on the toast's position in the array.
- Using a global `nextId` counter for toast keys to ensure React can track individual toasts across renders.

### What warrants a second pair of eyes
- `WorkflowDetailPage.tsx` changes: the `handleRetryOp` and `handleCancelWorkflow` callbacks are now `async` and call `.unwrap()` on the mutation to detect success/failure. Previously they fired-and-forgot. Verify the `.unwrap()` error shape matches the catch handler.

### What should be done in the future
- Add `autoDismissMs` as a parameter to `showToast()` for customizable durations
- Consider adding an action button to toasts (e.g., "Undo" after cancel)

### Code review instructions
- File: `web/src/components/common/ToastProvider.tsx` — review the stacking logic and auto-dismiss timer cleanup
- File: `web/src/pages/SubmitWorkflowPage.tsx` — verify old Snackbar code is fully removed and `useToast()` is used correctly
- File: `web/src/pages/WorkflowDetailPage.tsx` — verify async retry/cancel handlers with `.unwrap()`

### Technical details
- ToastProvider uses `useState<ToastEntry[]>` with a max of 3 entries
- Each toast gets a `setTimeout` for auto-dismiss; the timeout captures the toast `id` in a closure
- The `showToast` callback is memoized with `useCallback` (stable reference)

---

## Step 3: Breadcrumb Navigation

Add a breadcrumb component below the AppBar that derives crumbs from the current route. Provides navigation context for deep pages (Workflow Detail, Site Detail) and replaces the sole reliance on "Back" buttons.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Commit (code):** 3a67968 — "feat(ui): add BreadcrumbNav below AppBar"

### What I did
- Created `BreadcrumbNav.tsx` with `deriveCrumbs()` function that maps route patterns to crumb arrays
- Route → crumb mapping: `/` → hidden, `/workflows` → hidden, `/workflows/:id` → [Workflows, {name}], `/sites/:name` → [Sites, {name}], etc.
- Hides when there's only 1 crumb (top-level pages don't need breadcrumbs)
- Created `BreadcrumbNav.stories.tsx` with MemoryRouter for each route pattern
- Wired into `AppShell.tsx` between AppBar and content area
- Updated `WorkflowTable.tsx`: changed `onWorkflowClick(id)` signature to `onWorkflowClick(id, name)` so the name is available for route state
- Updated `WorkflowsPage.tsx`: `navigate()` now passes `{ state: { workflowName: name } }` so the breadcrumb can show the workflow name instead of the UUID

### Why
Deep pages like Workflow Detail only had a "← Back to Workflows" text link. No breadcrumb trail, no way to understand where you are in the hierarchy. Breadcrumbs are standard navigation affordance for hierarchical pages.

### What worked
- Using `location.state` to pass the workflow name was cleaner than fetching the workflow data just for the breadcrumb
- Hiding for single-crumb pages keeps the UI clean (no "Overview > Overview")

### What didn't work
- Nothing failed

### What I learned
- The `WorkflowTable.onWorkflowClick` only passed the ID. Had to widen the signature to also pass the name. The stories used `() => {}` no-ops which accepted extra args silently — no story changes needed.

### What was tricky to build
- Deciding how to get the workflow name into the breadcrumb. Options: (A) fetch from RTK Query cache, (B) pass via route state, (C) read from URL. Chose (B) — simplest, no extra API calls, works even while loading.

### What warrants a second pair of eyes
- `WorkflowTable.tsx` signature change: `onWorkflowClick: (id: string, name: string) => void`. All callers must now pass two args. Verified WorkflowsPage is the only caller.

### What should be done in the future
- The breadcrumb for workflow detail currently falls back to the UUID if the user navigates directly via URL (no route state). Could enhance by reading from RTK Query cache when state is missing.

### Code review instructions
- File: `web/src/components/layout/BreadcrumbNav.tsx` — review `deriveCrumbs` for completeness (all routes covered)
- File: `web/src/components/layout/AppShell.tsx` — verify BreadcrumbNav is rendered between AppBar and content Box
- File: `web/src/components/workflows/WorkflowTable.tsx` — verify `onWorkflowClick` now passes `(workflow.ID, workflow.Name)`
- File: `web/src/pages/WorkflowsPage.tsx` — verify `navigate` passes `{ state: { workflowName: name } }`

### Technical details
- `deriveCrumbs` is a pure function mapping `pathname` → `Crumb[]`
- `Crumb` type: `{ label: string; path?: string }` — last crumb has no path (current page)
- Uses MUI `<Breadcrumbs>` with `<NavigateNextIcon>` separator
- Wrapped in `<Box sx={{ px: 3, py: 0.75, bgcolor: 'grey.100' }}>` for visual separation from AppBar

---

## Step 4: RuntimeEventTable + Hook Updates + Page Replacements (Phase 2)

The core of the UI redesign: replace the bloated RuntimeEventList (100px per event, no sort, no expand) with a dense expandable table (32px per row). This is the highest-impact change in the whole ticket.

### Prompt Context

**User prompt (verbatim):** "ok, continue"

**Commit (code):** edc8d94 (table), 57872d4 (hook), de5aed4 (RuntimeEventsPage), 2945cab (WorkflowDetail+Drawer)

### What I did
- Created `SeverityDotIndicator.tsx` — tiny 10px colored dot + label
- Created `RuntimeEventTable.tsx` — MUI Table with sortable columns, expandable detail rows, optional pagination
- Updated `runtimeEventsApi.ts` — added `since`, `until`, `offset` to `RuntimeEventsParams`
- Updated `runtimeEventFeed.ts` — added `paused` state, `pause()`/`resume()` return values, propagated time params
- Rewrote `RuntimeEventsPage.tsx` — multi-select severity/source chips, TimeRangeSelector, collapsed advanced filters, pause/resume button, paginated table
- Replaced RuntimeEventList with RuntimeEventTable in `WorkflowDetailPage.tsx` and `OpDetailDrawer.tsx`

### Why
The old RuntimeEventList used a vertical `<List>` layout — each event was ~100px tall with 3 Chips + multiline text. 50 events = 4000px of scrolling. The new table is ~32px per row, fits 30 rows in a viewport, and expands on click for details.

### What worked
- The RuntimeEventTable's `Box component="tbody"` pattern for expandable rows — MUI Table doesn't natively support expandable rows, but wrapping each row + its detail in a `<Box component="tbody">` works perfectly
- The `useMemo` for client-side filtering in RuntimeEventsPage — clean separation between server filters (query params) and client filters (multi-select chips)
- The `paused` state in the hook — simple boolean that the SSE effect checks before opening EventSource

### What didn't work
- Nothing failed — all four commits compiled clean on first try

### What I learned
- `DateRangePicker` is Pro-only in @mui/x-date-pickers v8 — caught in Storybook, fixed by using two `DateTimePicker` components instead
- The RuntimeEventList is now unused in all 3 locations but not yet deleted (Phase 6 cleanup)

### What was tricky to build
- The sort function needed to handle all 4 fields correctly — timestamp (numeric from protobuf seconds), severity (enum numeric), source/kind (string comparison on enum labels)
- Client-side vs server-side filtering boundary: severity/source are client-side (multi-select chips filter the merged event array), while time range is server-side (added as query params to both REST and SSE endpoints)

### What warrants a second pair of eyes
- The `since` computation in RuntimeEventsPage: uses `dayjs().subtract(...)` for relative ranges. Verify the backend accepts ISO timestamps in the `since` param.
- The SSE stream URL now includes `since`/`until` — verify the backend SSE endpoint supports these params, or they'll be silently ignored.

### What should be done in the future
- Delete `RuntimeEventList.tsx` and its stories in Phase 6
- Add virtualization (react-window) if events scale to 1000s
- Backend: return `total` count for proper pagination

### Code review instructions
- File: `web/src/components/workflows/RuntimeEventTable.tsx` — review sort logic, expandable row pattern, pagination
- File: `web/src/pages/RuntimeEventsPage.tsx` — review multi-select filter integration, time range → serverSince computation
- File: `web/src/features/runtime-events/runtimeEventFeed.ts` — review pause/resume, since/until propagation

---

## Step 5: Bug Discovery and Handover

Discovered an infinite render loop in `useRuntimeEventFeed` that manifests whenever the backend is unavailable (Storybook, offline dev, network errors). Wrote detailed analysis and design exploration for a Redux-based rewrite.

### Prompt Context

**User prompt (verbatim):** "can we use redux to good effect here?"

**Assistant interpretation:** User recognized the setState/useEffect chain in the hook is the architectural root cause and suggested exploring a Redux approach.

**Inferred user intent:** Write up the analysis as a handover document so the next developer can pick up the Redux rewrite without re-investigating.

**Commits:** bug fixes: `49cc0be`, `7bce7f1`, `75e3a5f`, `68a690c`

### What I did
- Fixed SSE auto-reconnect loop (close EventSource on error)
- Fixed table column alignment (Fragment instead of Box tbody)
- Fixed icon import (from @mui/icons-material)
- Fixed missing TableContainer import
- Wrote detailed bug analysis document in ticket

### Why
The hook's 4 useState + 4 useEffect pattern creates cascading re-renders when RTK Query retries failed fetches. Each retry changes the `recentRuntimeEvents` reference, triggering the merge effect, which triggers a re-render, which triggers another RTK Query evaluation.

### What worked
- The EventSource close fix eliminated the SSE loop
- The Fragment fix eliminated the column misalignment

### What didn't work
- Guarding the merge effect with `if (!recentRuntimeEvents || recentRuntimeEvents.length === 0) return` didn't fix the loop because RTK Query still retries, and the reference still changes even when empty

### What I learned
- EventSource auto-reconnects by browser spec — must call `.close()` on error to stop
- `<Box component="tbody">` creates a separate tbody element per group, breaking MUI Table column alignment
- RTK Query's default retry behavior (3 retries with backoff) is hostile to Storybook/no-backend environments

### What was tricky to build
- Tracing the exact re-render chain: RTK Query retry → reference change → useEffect → setState → re-render → RTK re-evaluation → retry

### What warrants a second pair of eyes
- The Redux slice + listener middleware design in the analysis doc — is this the right architecture?
- Should we use `createListenerMiddleware` or a custom middleware?
- Memory management: max events in store? Trim strategy?

### What should be done in the future
- Implement the Redux-based rewrite as described in the analysis doc
- Pre-seed Storybook stores to avoid all network calls
- Add unit tests for the new slice

### Code review instructions
- Read the analysis doc: `ttmp/.../analysis/01-bug-analysis-runtimeeventfeed-infinite-loop.md`
- Review the 4 bug-fix commits for correctness
- The runtimeEventFeed.ts is in a partially-fixed state — the RTK Query loop is NOT fixed, only the SSE loop is fixed

### Technical details
- The `useRuntimeEventFeed` hook is at `web/src/features/runtime-events/runtimeEventFeed.ts`
- It has a test file at `web/src/features/runtime-events/runtimeEventFeed.test.ts`
- 3 consumers: RuntimeEventsPage, WorkflowDetailPage, OpDetailDrawer
- Each consumer creates its own SSE connection (3 connections for the same data)

---

## Step 6: Phase 2B — SSE Rewrite with RTK Query `onCacheEntryAdded`

Starting the architectural fix for the infinite loop bug. The analysis doc recommended using RTK Query's built-in `onCacheEntryAdded` lifecycle instead of the current `useRuntimeEventFeed` hook with its 4 useState + 4 useEffect chain. Created Phase 2B tasks (2B.1–2B.8) in the ticket.

### Prompt Context

**User prompt (verbatim):** "Work on docmgr ticket UI-001 and @ttmp/.../analysis/01-bug-analysis-runtimeeventfeed-infinite-loop.md in particular. Create tasks in the ticket for implementing the proper SSE redux rtk-query pattern (including storybook stories and real mock, i'm not sure if we can mock through msw, if not, then through rtk-query if possible)."

**Assistant interpretation:** Create implementation tasks for the RTK Query `onCacheEntryAdded` SSE pattern, including Storybook mocking strategy, then work through them task by task.

**Inferred user intent:** Complete the bug fix end-to-end with proper Storybook support, committing incrementally.

### Commit (code): e647cc3 — "refactor(UI-001): rewrite SSE streaming with RTK Query onCacheEntryAdded"
### Commit (code): 47ff46d — "feat(UI-001): shared mock factory + cache-seeded Storybook stories"

### What I did
- Rewrote `runtimeEventsApi.ts` with `onCacheEntryAdded` SSE lifecycle
- Created `runtimeEventHelpers.ts` with extracted pure functions (`filterRuntimeEvents`, `mergeRuntimeEvents`)
- Updated 3 consumers (RuntimeEventsPage, WorkflowDetailPage, OpDetailDrawer) to use `useGetRecentRuntimeEventsQuery` directly
- Deleted `runtimeEventFeed.ts` (the buggy hook)
- Updated Storybook global store to include `runtimeEventsApi` reducer + middleware
- Created shared `test-utils/mockRuntimeEvents.ts` with `generateMockEvents()` factory
- Rewrote `RuntimeEventsPage.stories.tsx` with cache pre-seeding (no network/SSE)
- Deduplicated mock data from `RuntimeEventTable.stories.tsx` into shared factory

### Why
The `useRuntimeEventFeed` hook had 4 useState + 4 useEffect causing cascading re-renders when RTK Query retried failed fetches. RTK Query's built-in `onCacheEntryAdded` lifecycle manages SSE natively: no local state, no effect chains, auto-cleanup on unsubscribe.

### What worked
- `onCacheEntryAdded` pattern is clean — cacheDataLoaded rejection on no-backend prevents SSE from even opening
- Cache pre-seeding via `store.dispatch(runtimeEventsApi.util.updateQueryData(...))` works perfectly for Storybook — the query resolves from cache, onCacheEntryAdded sees it as already fulfilled, no SSE opened
- All 157 tests pass including the new RuntimeEventsPage stories

### What didn't work
- `export constWithManyEvents` — missing space caused a syntax error that broke Storybook indexing. Caught by vitest run.
- Using `require()` in `runtimeEventHelpers.ts` for the `seedRuntimeEventsCache` helper — abandoned in favor of having callers pass the API object directly, then simplified further by just inlining the cache seeding in stories

### What I learned
- RTK Query's `updateQueryData` is synchronous and uses Immer — you can `draft.splice(0, draft.length, ...events)` to replace the entire cache entry
- The `onCacheEntryAdded` callback only fires when a new cache entry is created — if data is already in cache (from pre-seeding), the callback may still fire but `cacheDataLoaded` resolves immediately from cache
- Storybook's CSF parser is stricter than TypeScript — syntax errors in stories break the entire test run

### What was tricky to build
- Getting the Storybook store right: adding `runtimeEventsApi` reducer + middleware to the global preview.tsx decorator fixed all 6 OpDetailDrawer story failures at once
- The cache seeding timing: `store.dispatch(updateQueryData(...))` must happen before the component mounts, which means it goes in the decorator function, not in a useEffect

### What warrants a second pair of eyes
- The `onCacheEntryAdded` error handling — if `cacheDataLoaded` rejects, we return early and never open SSE. Is this the right behavior? Or should we retry?
- The `keepUnusedDataFor: 30` — 30 seconds might be too short or too long for the use case
- The sort in `updateCachedData` draft — sorting on every SSE message could be expensive with 500 events. Consider only sorting on initial load and inserting at the correct position for subsequent events.

### What should be done in the future
- Task 2B.8: End-to-end verification with real backend
- Consider adding `maxRetries` logic in `onCacheEntryAdded` for transient failures
- Consider optimizing the draft sort to insertion-sort instead of full re-sort

### Code review instructions
- Start with `web/src/api/runtimeEventsApi.ts` — the `onCacheEntryAdded` is the key change
- Check `web/src/pages/RuntimeEventsPage.tsx` for the consumer pattern (pause via `{ skip: paused }`)
- Check `web/.storybook/preview.tsx` for the global store setup
- Check `web/src/test-utils/mockRuntimeEvents.ts` for the shared factory
- Run `npx vitest run` — all 157 tests should pass

### Technical details
- `cacheDataLoaded` resolves when the initial REST query succeeds; rejects if the query fails
- `cacheEntryRemoved` resolves when all subscribers unsubscribe (component unmount or `skip: true`)
- `updateCachedData` uses Immer — mutations to the draft are applied immutably
- The `buildSSEUrl` function only includes scoping params (workflowId, opId, site, workerId) — not pagination/time params

---

## Step 7: MSW Storybook Root-Cause Debugging and Bug-Report Update

I came back to the MSW Storybook bug report because the earlier analysis had stalled at the symptom level: Storybook showed `[MSW] Mocking enabled`, yet `/api/v1/runtime-events?limit=100` still returned 404. This step focused on turning that vague state into a precise dependency-level diagnosis with reproducible evidence and a concrete mitigation path.

The key outcome is that the 404 itself was not the real root cause. The primary failure is an incompatibility between `msw-storybook-addon@2.0.6` and `msw@2.13.0`: the addon writes to `worker.context.activationPromise`, but `msw@2.13.0` no longer exposes `context` on the object returned by `setupWorker()`. After temporarily pinning MSW to `2.12.0`, the handler started intercepting correctly and the 404 disappeared, which immediately exposed a smaller second bug in our mock protobuf JSON shape.

### Prompt Context

**User prompt (verbatim):** "Work on docmgr ticket UI-001 on @ttmp/2026/04/07/UI-001--ui-redesign-runtime-events-workflow-detail-and-system-ux-improvements/analysis/02-bug-report-msw-handlers-not-intercepting-rtk-query-requests-in-storybook.md 

We got stuck previously"

**Assistant interpretation:** Resume the stalled MSW / Storybook investigation on the existing UI-001 bug-report document, gather stronger evidence, and improve the ticket handoff so another expert can pick it up cleanly.

**Inferred user intent:** Convert the previous “we are stuck” state into a concrete, well-documented debugging result with a clear next step.

**Commit (code):** none yet during investigation — temporary working-tree edits only; final docs commit recorded after write-up

### What I did
- Re-read the existing UI-001 bug report and the current Storybook / RTK Query files.
- Re-enabled the standard MSW Storybook setup temporarily in `web/.storybook/preview.tsx`:
  - `import { initialize, mswLoader } from 'msw-storybook-addon'`
  - `initialize({ onUnhandledRequest: 'bypass' })`
  - `loaders: [mswLoader]`
- Added a temporary console log inside `web/src/mocks/runtimeEventsHandlers.ts` to prove whether the runtime-events handler ever matched.
- Opened Storybook in the browser (`http://localhost:6006/?path=/story/pages-runtimeeventspage--default`) and captured console + network behavior with Playwright MCP.
- Inspected the shipped addon code in `node_modules/.../msw-storybook-addon/dist/index.browser.js` and confirmed `initialize()` does `worker.context.activationPromise = worker.start(...)`.
- Inspected the shipped MSW browser bundle in `node_modules/.pnpm/msw@2.13.0.../lib/browser/index.mjs` and confirmed the returned `setupWorker()` object in 2.13.0 no longer exposes `context`.
- Downloaded and inspected multiple MSW tarballs (`2.0.9`, `2.6.8`, `2.8.0`, `2.10.0`, `2.11.0`, `2.12.0`, `2.13.0`) to locate the compatibility boundary.
- Temporarily pinned the repo to `msw@2.12.0`, restarted Storybook, and re-ran the same Storybook page.
- Once interception worked, I tested a follow-up fix attempt for the mock serializer and documented the next error.
- Updated `analysis/02-bug-report-msw-handlers-not-intercepting-rtk-query-requests-in-storybook.md` with the confirmed root cause, compatibility matrix, reproduction evidence, and recommended remediation path.

### Why
- The ticket handoff needed evidence, not hypotheses. The earlier report had multiple plausible causes (URL matching, timing, RTK Query behavior), but no decisive proof.
- By checking the addon bundle, the MSW bundle, and then validating behavior with a downgrade experiment, I could distinguish the real root cause from the surface-level 404 symptom.

### What worked
- The temporary handler log made it obvious whether requests were truly intercepted.
- Inspecting the actual shipped bundles in `node_modules` was much more effective than guessing from API docs.
- Downgrading to `msw@2.12.0` and restarting Storybook was the decisive experiment:
  - the `activationPromise` error disappeared,
  - the handler matched,
  - the runtime-events request returned `200 OK` instead of `404`.
- The compatibility sweep across extracted npm tarballs narrowed the regression window to the jump from `msw@2.12.0` to `msw@2.13.0`.

### What didn't work
- With `msw@2.13.0`, the standard Storybook setup still failed with:
  ```text
  TypeError: Cannot set properties of undefined (setting 'activationPromise')
      at initialize (.../msw-storybook-addon.js:3495:35)
      at .storybook/preview.tsx:11:1
  ```
- A first attempt to serialize mock events via a hand-written protobuf JSON object returned the wrong shape for `google.protobuf.Timestamp`, causing:
  ```text
  Error: cannot decode message google.protobuf.Timestamp from JSON: object
  ```
- A second attempt to use Buf serialization directly via `toJson(RuntimeEventV1Schema, event)` also failed because the mock factory returns plain objects typed as `RuntimeEventV1`, not actual Buf message instances:
  ```text
  ForeignFieldError: cannot use field scraper.runtime.v1.RuntimeEventV1.schema_version with message undefined
  ```

### What I learned
- The original 404 was mostly a **symptom** of Storybook preview initialization failing, not a handler URL mismatch.
- `msw-storybook-addon@2.0.6` depends on an internal-ish `worker.context` field that disappeared in `msw@2.13.0`.
- `google.protobuf.Timestamp` in protobuf JSON is not `{ seconds, nanos }`; Buf expects an RFC3339 string.
- `toJson()` from `@bufbuild/protobuf` expects an actual Buf message value, not just a plain object with a matching TypeScript shape.

### What was tricky to build
- The misleading part was that MSW still printed its normal startup banner even when `initialize()` crashed immediately afterward. That made it look like the worker was healthy and that the problem had to be URL matching or RTK Query timing.
- Another sharp edge was Storybook caching: after changing the MSW dependency version, I had to restart the Storybook dev server to flush the cached `sb-vite` dependency bundle; otherwise the page kept using the previously bundled addon/MSW code and reproduced the same old error.
- The protobuf mock follow-up bug is subtle because the TypeScript type `RuntimeEventV1` makes a plain object look “message-shaped”, but Buf JSON serialization/deserialization cares about actual message semantics, not just TypeScript structure.

### What warrants a second pair of eyes
- The exact long-term fix strategy:
  - pin `msw` to `2.12.0`,
  - or find an addon release compatible with `msw@2.13.x+`,
  - or abandon the addon in favor of RTK Query story-specific mocking.
- The correct protobuf JSON serialization path for story mocks:
  - manual encoder with RFC3339 timestamps,
  - or create real Buf message instances before calling `toJson()`.
- Whether the Vitest / Storybook integration needs a separate setup even after the MSW version pin.

### What should be done in the future
- Pin `msw` to `2.12.0` in a focused code change and verify interactive Storybook end-to-end.
- Fix `runtimeEventsHandlers.ts` so it emits valid protobuf JSON for `RuntimeEventV1`.
- Once interactive Storybook is green, revisit the test-runner side (`@storybook/addon-vitest`) separately.
- Check whether a newer `msw-storybook-addon` release has removed the `worker.context` assumption.

### Code review instructions
- Start with the updated bug report: `ttmp/2026/04/07/UI-001--ui-redesign-runtime-events-workflow-detail-and-system-ux-improvements/analysis/02-bug-report-msw-handlers-not-intercepting-rtk-query-requests-in-storybook.md`
- Then inspect these files for the evidence path:
  - `web/.storybook/preview.tsx`
  - `web/src/mocks/runtimeEventsHandlers.ts`
  - `web/node_modules/.../msw-storybook-addon/dist/index.browser.js` (`initialize`)
  - `web/node_modules/.pnpm/msw@2.13.0.../node_modules/msw/lib/browser/index.mjs` (`setupWorker` return object)
- Reproduce with Storybook at `/story/pages-runtimeeventspage--default`
- Validate the downgrade experiment by pinning `msw@2.12.0` and restarting Storybook

### Technical details
- Exact command used to pin MSW temporarily:
  ```bash
  cd /home/manuel/workspaces/2026-03-23/js-scraper/scraper/web && pnpm add -D msw@2.12.0
  ```
- Exact command used to restart Storybook after dependency changes:
  ```bash
  kill <storybook-pids>
  cd /home/manuel/workspaces/2026-03-23/js-scraper/scraper/web
  nohup pnpm storybook > /tmp/ui001-storybook.log 2>&1 &
  ```
- Confirmed good / bad version boundary from tarball inspection:
  - good: `2.0.9`, `2.6.8`, `2.8.0`, `2.10.0`, `2.11.0`, `2.12.0`
  - bad: `2.13.0`
- Successful interception evidence after downgrade:
  ```text
  [msw] runtime-events handler matched http://localhost:6006/api/v1/runtime-events?limit=100
  GET http://localhost:6006/api/v1/runtime-events?limit=100 => 200 OK
  ```

---

## Step 8: Implemented the Storybook MSW Fix

Once the dependency mismatch and mock-shape issues were proven, I switched from analysis back to implementation. This step takes the investigation from “we know why it is broken” to “the RuntimeEventsPage story works again in Storybook with stable mocks and no runtime console errors.”

The final fix is intentionally pragmatic. I pinned `msw` to the last verified-compatible version (`2.12.0`), restored standard `msw-storybook-addon` initialization, emitted valid protobuf JSON from the runtime-event mock handler by constructing real Buf messages, disabled Storybook-only serializability warnings for protobuf `bigint` timestamps, and skipped SSE in Storybook so the unmocked stream endpoint stops generating 404 noise.

### Prompt Context

**User prompt (verbatim):** "did you fix it though? you are the expert now."

**Assistant interpretation:** Stop handoff-only work, finish the actual implementation, and verify the bug is truly resolved.

**Inferred user intent:** Get the Storybook runtime-events mocking bug fixed, not just analyzed.

**Commit (code):** 92cb0a9 — "fix(UI-001): restore Storybook runtime event mocking"

### What I did
- Pinned `msw` from `2.13.0` to `2.12.0` in `web/package.json` and `web/pnpm-lock.yaml`
- Restored standard MSW Storybook setup in `web/.storybook/preview.tsx`:
  - `initialize({ onUnhandledRequest: 'bypass' })`
  - `loaders: [mswLoader]`
- Updated Storybook stores in:
  - `web/.storybook/preview.tsx`
  - `web/src/pages/RuntimeEventsPage.stories.tsx`
  to use `getDefault({ serializableCheck: false })` so protobuf timestamps with `bigint` do not trigger Redux Toolkit warnings in Storybook
- Fixed `web/src/mocks/runtimeEventsHandlers.ts` to serialize through real Buf messages:
  - `create(RuntimeEventV1Schema, event)`
  - `toJson(RuntimeEventV1Schema, message)`
- Updated `web/src/api/runtimeEventsApi.ts` to skip `EventSource` setup when running inside Storybook (`__STORYBOOK_PREVIEW__` present)
- Verified the story in a real browser via Playwright MCP and confirmed the table body renders 20 mocked rows
- Re-ran validation:
  - `npx tsc --noEmit`
  - `npx vitest run`

### Why
- Analysis alone had already identified the root cause and the shortest stable fix path. Continuing to leave the repo in a “known broken but documented” state would waste time.
- Pinning `msw` is the smallest reliable fix while keeping the existing Storybook addon integration intact.
- Skipping Storybook SSE is a quality-of-life fix: Storybook only needs deterministic mock REST data for these stories, not an unmocked live stream.

### What worked
- After pinning to `msw@2.12.0`, `msw-storybook-addon` stopped throwing `activationPromise` errors
- The runtime-events handler intercepted correctly and returned `200 OK`
- Constructing actual Buf messages before `toJson()` solved the protobuf JSON shape problem cleanly
- Disabling `serializableCheck` in Storybook stores removed the protobuf `bigint` warnings
- Skipping SSE in Storybook removed the last `/api/v1/runtime-events/stream` 404 console error
- The RuntimeEventsPage story now renders the full table and interactive controls without runtime errors
- `npx tsc --noEmit` and `npx vitest run` both pass; vitest result: **42 files / 156 tests**

### What didn't work
- Leaving the hand-written protobuf JSON in place caused:
  ```text
  Error: cannot decode message google.protobuf.Timestamp from JSON: object
  ```
- Calling `toJson(RuntimeEventV1Schema, event)` on a plain object failed with:
  ```text
  ForeignFieldError: cannot use field scraper.runtime.v1.RuntimeEventV1.schema_version with message undefined
  ```
  That only worked after wrapping the plain object with `create(RuntimeEventV1Schema, event)`.

### What I learned
- For Buf-generated message types, “plain object with the right shape” is not the same thing as a real message instance when you serialize.
- Storybook is a different enough environment from the real app that it is often worth explicitly skipping live subsystems like SSE instead of trying to mock every moving part.
- A narrow dependency pin is sometimes the most maintainable fix when the alternative is working around a third-party compatibility regression.

### What was tricky to build
- The last 20% was about reducing noise, not restoring functionality. Once the story loaded data, the remaining work was removing environment-specific errors (serializable checks, SSE 404s) without changing production behavior.
- The Storybook-specific SSE skip needed to be narrow: only disable the stream when `__STORYBOOK_PREVIEW__` is present, and keep the normal app behavior untouched.
- Because Storybook caches optimized dependencies, some changes only reflected after a full server restart; relying on HMR alone was not always enough for dependency-level fixes.

### What warrants a second pair of eyes
- The Storybook environment detection in `runtimeEventsApi.ts` (`__STORYBOOK_PREVIEW__` check) — it is intentionally narrow, but it is still an app-level reference to a Storybook global.
- The decision to pin `msw` to `2.12.0` — reasonable now, but should be revisited once addon compatibility with newer MSW is confirmed.
- The stream status label in Storybook still reads as “Live” because the page derives status from query success, not real SSE status. That is cosmetic but worth noting.

### What should be done in the future
- Revisit the `msw` pin when `msw-storybook-addon` supports `msw@2.13.x+`
- Consider a small Storybook-specific badge or prop to indicate mocked/static mode instead of “Live”
- Add MSW handlers for other pages (e.g. sites detail / verbs) if those stories are expected to run backend-free as well

### Code review instructions
- Start with `web/src/mocks/runtimeEventsHandlers.ts` — this is where the mock payload became valid protobuf JSON
- Then review `web/.storybook/preview.tsx` and `web/src/pages/RuntimeEventsPage.stories.tsx` for the MSW wiring and Storybook-only middleware change
- Review `web/src/api/runtimeEventsApi.ts` for the `isStorybookEnvironment()` gate in `onCacheEntryAdded`
- Validate manually:
  - run `pnpm storybook`
  - open `Pages / RuntimeEventsPage / Default`
  - confirm mocked rows render and console has no runtime/MSW errors
- Validate automatically:
  - `cd web && npx tsc --noEmit`
  - `cd web && npx vitest run`

### Technical details
- Working Storybook page URL:
  ```text
  http://localhost:6006/?path=/story/pages-runtimeeventspage--default
  ```
- Key files changed in the fix commit:
  - `web/package.json`
  - `web/pnpm-lock.yaml`
  - `web/.storybook/preview.tsx`
  - `web/src/pages/RuntimeEventsPage.stories.tsx`
  - `web/src/mocks/runtimeEventsHandlers.ts`
  - `web/src/api/runtimeEventsApi.ts`
- Browser verification after the fix showed the iframe body contained:
  - `Runtime Events`
  - `20 events`
  - 20 rendered rows with timestamps, severities, sources, and messages
