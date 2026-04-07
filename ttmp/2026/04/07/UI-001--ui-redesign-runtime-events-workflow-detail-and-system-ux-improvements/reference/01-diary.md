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
