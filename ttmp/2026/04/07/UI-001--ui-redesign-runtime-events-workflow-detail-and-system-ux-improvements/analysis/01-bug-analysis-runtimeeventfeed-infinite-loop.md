---
Title: "Bug Analysis: RuntimeEventFeed Infinite Loop"
Ticket: UI-001
Status: active
Topics:
    - frontend
    - ux-design
    - ui-rework
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Analysis of the infinite render loop in useRuntimeEventFeed, root cause, and design exploration for a Redux-based rewrite."
LastUpdated: 2026-04-07
WhatFor: "Handover document for the next developer picking up Phase 2."
WhenToUse: "Read this before modifying runtimeEventFeed.ts or adding event-consuming components."
---

# Bug Analysis: RuntimeEventFeed Infinite Loop

**Date:** 2026-04-07
**Severity:** 🔴 Blocking — Storybook and any environment without a backend spins forever
**Status:** Open — workaround in place, proper fix requires architectural change
**Affected file:** `web/src/features/runtime-events/runtimeEventFeed.ts`
**Discovered in:** Phase 2 of UI-001, when loading RuntimeEventsPage in Storybook

---

## 1. Bug Report

### Symptom

When `RuntimeEventsPage` renders in Storybook (or any environment where the backend is unavailable), React throws:

```
Maximum update depth exceeded. This can happen when a component calls
setState inside useEffect, but useEffect either doesn't have a dependency
array, or one of the dependencies changes on every render.
```

The page never finishes rendering. In the browser, the SSE connection also loops: `EventSource` auto-reconnects on error, creating an infinite connect → error → reconnect cycle.

### Reproduction

1. Run `pnpm storybook`
2. Navigate to **Pages > RuntimeEventsPage > Default**
3. Observe: Storybook spinner never resolves, console fills with the error above

### Current Workaround (partial)

- Commit `49cc0be`: Added `eventSource.close()` inside the SSE `onError` handler to prevent EventSource auto-reconnect.
- This fixed the SSE loop but did **not** fix the RTK Query loop.

---

## 2. Root Cause Analysis

### 2.1 The Hook's Architecture

`useRuntimeEventFeed` manages **4 pieces of local state** via `useState`, coordinated by **4 `useEffect` hooks**:

```
State:
  allEvents: RuntimeEventV1[]          ← merged history + live events
  connectionState: ConnectionState     ← 'connecting' | 'live' | 'error' | 'closed' | 'paused'
  lastEventAt: number | null           ← timestamp of most recent event
  paused: boolean                      ← user paused the stream

Effects:
  [search, stream]    → reset allEvents + connectionState when filters change
  [recentRuntimeEvents] → merge RTK Query history into allEvents
  [allEvents]         → update lastEventAt when events change
  [search, stream]    → open/close SSE EventSource for live streaming
```

### 2.2 The Infinite Loop

Here is the exact chain that causes the infinite loop:

```
1. Component mounts
2. useGetRecentRuntimeEventsQuery fires → RTK Query fetches /api/v1/runtime-events
3. No backend → fetch fails → RTK Query retries automatically (built-in retry behavior)
4. Retry "succeeds" with an error response → recentRuntimeEvents reference changes
5. useEffect([recentRuntimeEvents]) fires → setAllEvents(mergeRuntimeEvents(current, []))
6. setAllEvents triggers re-render
7. RTK Query sees the re-render, may re-evaluate cache → triggers another refetch
8. Goto step 3 → infinite loop
```

The key insight: **`recentRuntimeEvents` is an array reference that changes on every RTK Query cache update, even when empty.** The effect that merges it into `allEvents` calls `setAllEvents` on every change, which triggers a re-render, which can cause RTK Query to re-evaluate its cache state.

### 2.3 Why It Works With a Real Backend

With a real backend:
- Step 3 succeeds, returns actual events
- Step 5 merges them, `allEvents` stabilizes
- Step 7 doesn't re-trigger because the query is fulfilled, not retrying

The bug is masked in production because the query succeeds on the first try.

### 2.4 Contributing Factors

| Factor | Why It Matters |
|--------|----------------|
| RTK Query built-in retry | Failed queries retry up to 3 times with exponential backoff, each retry changes the query state |
| `recentRuntimeEvents` defaults to `[]` | Even a failed query returns `[]` as the default, which is a new array reference each render |
| `setAllEvents` called on empty arrays | `mergeRuntimeEvents(current, [])` returns the same array but the setter still triggers a re-render |
| SSE EventSource auto-reconnect | Browser spec: EventSource reconnects on error unless explicitly closed |

---

## 3. What We've Fixed So Far

| Commit | Fix | Status |
|--------|-----|--------|
| `49cc0be` | Close EventSource on error | ✅ Fixes SSE loop |
| `7bce7f1` | Fragment instead of Box tbody | ✅ Fixes table column alignment |
| `75e3a5f` | Icons from @mui/icons-material | ✅ Fixes Storybook import error |
| `68a690c` | Missing TableContainer import | ✅ Fixes render crash |

**Still broken:** The RTK Query retry → setState → re-render loop.

---

## 4. Design Exploration: Redux-Based Rewrite

### 4.1 Current Architecture (Problems)

```
┌─ useRuntimeEventFeed hook ─────────────────────────────┐
│                                                         │
│  useState(allEvents)        ← 4 local state slices      │
│  useState(connectionState)  ← each triggers re-render   │
│  useState(lastEventAt)      ← cascading useEffects      │
│  useState(paused)           ← hard to test              │
│                                                         │
│  useEffect([search])        → reset state               │
│  useEffect([recentEvents])  → merge into allEvents      │
│  useEffect([allEvents])     → update lastEventAt        │
│  useEffect([search,stream]) → open/close SSE            │
│                                                         │
│  Returns: { events, connectionState, lastEventAt,       │
│             clearEvents, pause, resume }                │
└─────────────────────────────────────────────────────────┘
         ↑ consumed by 3 components:
         - RuntimeEventsPage
         - WorkflowDetailPage
         - OpDetailDrawer
```

**Problems with this architecture:**

1. **3 separate event arrays in memory** — each consumer calls `useRuntimeEventFeed` independently, creating separate SSE connections and duplicate event arrays
2. **useEffect chains** — 4 effects that trigger each other, hard to reason about lifecycle
3. **No shared state** — Overview page's "Recent Activity" feed can't reuse events already fetched by RuntimeEventsPage
4. **Untestable** — hook depends on EventSource browser API, no way to unit test without DOM
5. **Storybook-hostile** — requires real backend or complex mocking

### 4.2 Proposed Architecture: Redux Slice + Listener Middleware

```
┌─ store/runtimeEventsSlice.ts ──────────────────────────┐
│                                                         │
│  State:                                                 │
│    events: RuntimeEventV1[]                             │
│    connectionState: ConnectionState                     │
│    lastEventAt: number | null                           │
│    paused: boolean                                      │
│    filters: { workflowId?, opId?, site?, ... }          │
│                                                         │
│  Reducers:                                              │
│    addEvent(event)         → merge into events[]        │
│    addEvents(events[])     → batch merge                │
│    setConnectionState(s)   → update state               │
│    clearEvents()           → empty events[]             │
│    setPaused(bool)         → toggle pause               │
│    setFilters(filters)     → update server filters      │
│                                                         │
│  Selectors:                                             │
│    selectAllEvents                                     │
│    selectFilteredEvents(severity[], source[])           │
│    selectConnectionState                               │
│    selectLastEventAt                                   │
│    selectIsPaused                                      │
└─────────────────────────────────────────────────────────┘

┌─ store/runtimeEventsListener.ts ───────────────────────┐
│                                                         │
│  listenerMiddleware.startListening({                    │
│    actionCreator: setFilters,                           │
│    effect: (action, api) => {                           │
│      // 1. Fetch history via RTK Query                 │
│      // 2. Open SSE stream                              │
│      // 3. Dispatch addEvent on each SSE message        │
│      // 4. Close old SSE when filters change            │
│    }                                                    │
│  })                                                     │
│                                                         │
│  listenerMiddleware.startListening({                    │
│    actionCreator: setPaused,                            │
│    effect: (action, api) => {                           │
│      // Close/reopen SSE based on paused state          │
│    }                                                    │
│  })                                                     │
└─────────────────────────────────────────────────────────┘
```

### 4.3 Why This Fixes the Bug

| Problem | Redux Fix |
|---------|-----------|
| RTK Query retry loop | Listener middleware controls when to fetch; can check `isError` and stop |
| setState chains | Dispatch actions instead of setState; reducers are synchronous, no cascading |
| SSE auto-reconnect | Listener manages SSE lifecycle; closes on error, doesn't recreate |
| Duplicate connections | Single SSE connection shared via Redux; all consumers read from same store |
| Storybook unfriendly | Pre-seed `store.events = [...]` in story; listener never opens SSE |
| Untestable | Test reducers + selectors as pure functions; test listener with mock dispatch |

### 4.4 Consumer API (Before vs After)

**Before (current hook):**
```typescript
// Each consumer creates its own SSE connection and event array
const { events, connectionState, pause, resume } = useRuntimeEventFeed({
  serverFilters: { workflowId, limit: 50 },
  stream: true,
});
```

**After (Redux):**
```typescript
// Dispatch filters, read from store
dispatch(setFilters({ workflowId, limit: 50 }));
const events = useSelector(selectFilteredEvents(selectedSeverities, selectedSources));
const connectionState = useSelector(selectConnectionState);
const isPaused = useSelector(selectIsPaused);
// Pause/resume
dispatch(setPaused(true));
```

### 4.5 Implementation Sketch

```
Files to create:
  web/src/store/runtimeEventsSlice.ts       ← state + reducers + selectors
  web/src/store/runtimeEventsListener.ts    ← SSE + fetch lifecycle

Files to modify:
  web/src/store/index.ts                    ← register new slice + listener
  web/src/features/runtime-events/runtimeEventFeed.ts  ← DELETE or thin wrapper
  web/src/pages/RuntimeEventsPage.tsx       ← use selectors + dispatch instead of hook
  web/src/pages/WorkflowDetailPage.tsx      ← same
  web/src/components/workflows/OpDetailDrawer.tsx      ← same
```

### 4.6 Storybook With Redux

```typescript
// In story:
const storyStore = configureStore({
  reducer: {
    runtimeEvents: runtimeEventsSlice.reducer,
    // ...
  },
  preloadedState: {
    runtimeEvents: {
      events: generateMockEvents(20),    // ← pre-seeded!
      connectionState: 'live',
      lastEventAt: Date.now(),
      paused: false,
      filters: {},
    },
  },
});
// No SSE, no fetching — just renders from pre-seeded state
```

### 4.7 Risks / Tradeoffs

| Concern | Mitigation |
|---------|------------|
| More boilerplate | Yes, but the current hook has 170 lines of useEffect spaghetti. A slice + listener is clearer. |
| Memory pressure | Events array grows unbounded. Add a `maxEvents` config (default 500) and trim in the reducer. |
| Listener middleware is less familiar | It's official RTK API: https://redux-toolkit.js.org/api/createListenerMiddleware |
| Migration path | Can do incrementally: create slice, wire consumers one at a time, delete hook last. |

---

## 5. Recommended Next Steps

1. **Create `runtimeEventsSlice.ts`** — state, reducers, selectors (no SSE yet)
2. **Wire consumers** — update RuntimeEventsPage to use selectors/dispatch instead of hook
3. **Create `runtimeEventsListener.ts`** — move SSE + fetch logic out of the hook into listener
4. **Delete `runtimeEventFeed.ts`** — or keep as a thin compat wrapper if needed
5. **Fix Storybook** — pre-seed state, no network calls needed
6. **Add tests** — test reducers (pure functions), test selectors, test listener with mock SSE

---

## 6. Reference

### Current Hook Source

`web/src/features/runtime-events/runtimeEventFeed.ts` (~170 lines)

Key functions that would move to the Redux slice:
- `mergeRuntimeEvents()` → becomes the `addEvent` reducer logic
- `filterRuntimeEvents()` → becomes a memoized selector
- `buildRuntimeEventSearchParams()` → used inside the listener middleware
- `runtimeEventOccurredAtMillis()` → helper, stays as-is

### Consumers (files that call `useRuntimeEventFeed`)

| File | Filters | Stream | Notes |
|------|---------|--------|-------|
| `pages/RuntimeEventsPage.tsx` | user-configurable | true | Global events, all filters |
| `pages/WorkflowDetailPage.tsx` | `{ workflowId, limit: 50 }` | true | Workflow-scoped |
| `components/workflows/OpDetailDrawer.tsx` | `{ workflowId, opId, limit: 40 }` | conditional | Op-scoped, only when tab active |

### RTK Listener Middleware Docs

- https://redux-toolkit.js.org/api/createListenerMiddleware
- https://redux-toolkit.js.org/api/createListenerMiddleware#examples
