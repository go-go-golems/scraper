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

## 4. Design Exploration

### Design A: RTK Query `onCacheEntryAdded` (✅ RECOMMENDED)

RTK Query has a **built-in streaming lifecycle** via `onCacheEntryAdded` designed exactly for SSE/WebSocket use cases. This is the official Redux Toolkit pattern and requires **no new files** — just rewriting the existing `runtimeEventsApi.ts` endpoint.

See: https://redux-toolkit.js.org/rtk-query/usage/streaming-updates

#### How It Works

The SSE connection lifecycle is managed inside the endpoint definition, not in a hook. RTK Query handles:
- Opening the SSE connection after the initial REST fetch resolves
- Pushing SSE messages into the cache via Immer `updateCachedData`
- Auto-closing the SSE connection when no subscribers remain (`cacheEntryRemoved`)
- Deduplication — same endpoint+params = one shared SSE connection

#### Implementation

```typescript
// web/src/api/runtimeEventsApi.ts
import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { fromJson } from '@bufbuild/protobuf';
import type { JsonValue } from '@bufbuild/protobuf';
import {
  RuntimeEventV1Schema,
  type RuntimeEventV1,
} from '../pb/proto/scraper/runtime/v1/events_pb';

interface RuntimeEventsResponse {
  events: JsonValue[];
}

export interface RuntimeEventsParams {
  workflowId?: string;
  opId?: string;
  site?: string;
  workerId?: string;
  limit?: number;
  since?: string;
  until?: string;
  offset?: number;
}

const MAX_CACHED_EVENTS = 500;

function decodeRuntimeEvent(json: JsonValue): RuntimeEventV1 {
  return fromJson(RuntimeEventV1Schema, json);
}

function eventMillis(event: RuntimeEventV1): number {
  return Number(event.occurredAt?.seconds ?? 0n) * 1000
    + Math.floor((event.occurredAt?.nanos ?? 0) / 1_000_000);
}

function buildRuntimeEventQuery(params: RuntimeEventsParams): string {
  const sp = new URLSearchParams();
  if (params.workflowId) sp.set('workflowId', params.workflowId);
  if (params.opId)      sp.set('opId', params.opId);
  if (params.site)      sp.set('site', params.site);
  if (params.workerId)  sp.set('workerId', params.workerId);
  if (params.limit)     sp.set('limit', String(params.limit));
  if (params.since)     sp.set('since', params.since);
  if (params.until)     sp.set('until', params.until);
  if (params.offset)    sp.set('offset', String(params.offset));
  return `/runtime-events?${sp.toString()}`;
}

function buildSSEUrl(params: RuntimeEventsParams): string {
  const sp = new URLSearchParams();
  if (params.workflowId) sp.set('workflowId', params.workflowId);
  if (params.opId)      sp.set('opId', params.opId);
  if (params.site)      sp.set('site', params.site);
  if (params.workerId)  sp.set('workerId', params.workerId);
  const search = sp.toString();
  return search
    ? `/api/v1/runtime-events/stream?${search}`
    : '/api/v1/runtime-events/stream';
}

export const runtimeEventsApi = createApi({
  reducerPath: 'runtimeEventsApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api/v1' }),
  tagTypes: ['RuntimeEvents'],
  endpoints: (builder) => ({
    getRecentRuntimeEvents: builder.query<RuntimeEventV1[], RuntimeEventsParams>({
      query: (params) => buildRuntimeEventQuery(params),
      transformResponse: (response: RuntimeEventsResponse) =>
        response.events.map(decodeRuntimeEvent),
      keepUnusedDataFor: 30, // keep cache for 30s after last subscriber unmounts

      async onCacheEntryAdded(
        arg,
        { updateCachedData, cacheDataLoaded, cacheEntryRemoved },
      ) {
        // --- SSE lifecycle managed by RTK Query ---

        // 1. Wait for the initial REST fetch to resolve before opening SSE
        try {
          await cacheDataLoaded;
        } catch {
          // cacheEntryRemoved resolved before cacheDataLoaded (no backend)
          return;
        }

        // 2. Open SSE stream
        const sseUrl = buildSSEUrl(arg);
        const eventSource = new EventSource(sseUrl);

        const onMessage = (event: MessageEvent<string>) => {
          try {
            const decoded = decodeRuntimeEvent(JSON.parse(event.data));
            updateCachedData((draft) => {
              // Dedupe
              const exists = draft.some((e) => e.id === decoded.id);
              if (!exists) draft.unshift(decoded);
              // Sort newest-first
              draft.sort((a, b) => eventMillis(b) - eventMillis(a));
              // Trim to max
              if (draft.length > MAX_CACHED_EVENTS) draft.length = MAX_CACHED_EVENTS;
            });
          } catch {
            // ignore malformed event payloads
          }
        };

        eventSource.addEventListener('runtime-event', onMessage as EventListener);

        // 3. Auto-cleanup when no subscribers remain
        await cacheEntryRemoved;
        eventSource.close();
      },
    }),
  }),
});

export const { useGetRecentRuntimeEventsQuery } = runtimeEventsApi;
```

#### Consumer API (Before vs After)

```typescript
// ── BEFORE (custom hook with useState + useEffect chain) ──
const { events, connectionState, pause, resume, clearEvents } = useRuntimeEventFeed({
  serverFilters: { workflowId, limit: 100 },
  clientFilters: { severity, source },
});

// ── AFTER (RTK Query — no custom hook) ──
const { data: events = [], isLoading, isError, isSuccess } =
  useGetRecentRuntimeEventsQuery({
    workflowId: workflowId || undefined,
    limit: 100,
    since: serverSince,
    until: serverUntil,
  });

// Connection state derived from query status:
const connectionState =
  isLoading ? 'connecting' :
  isError  ? 'error' :
  isSuccess ? 'live' : 'closed';

// Client-side filtering stays the same (useMemo)
const filteredEvents = useMemo(
  () => events.filter(e => /* severity/source checks */),
  [events, selectedSeverities, selectedSources]
);

// Pause/resume: skip the query to close SSE, re-enable to reopen
const [paused, setPaused] = useState(false);
useGetRecentRuntimeEventsQuery(params, { skip: paused });
```

#### Why This Fixes Everything

| Problem | How `onCacheEntryAdded` solves it |
|---------|-----------------------------------|
| Infinite re-render loop | No `useState` + `useEffect` chain. Cache is managed by RTK Query. `updateCachedData` is Immer-powered, no setState. |
| SSE reconnect loop | `cacheEntryRemoved` resolves only on unsubscribe — clean lifecycle. No auto-reconnect. |
| Duplicate SSE connections | RTK Query deduplicates by endpoint+params. Same `workflowId` → one connection, shared cache. |
| Storybook | Pre-seed cache with `store.dispatch(api.util.updateQueryData(...))` — `onCacheEntryAdded` only fires on real fetches. |
| Memory leak | Cache auto-evicts after `keepUnusedDataFor: 30` seconds with no subscribers. |
| Pause/resume | Use `{ skip: paused }` option. Skipping unsubscribes → `cacheEntryRemoved` fires → SSE closes. |
| No backend | `cacheDataLoaded` rejects → catch block returns early. No SSE opened, no retry loop. |

#### Storybook Fix

```typescript
// In story: pre-seed the cache, never fetch
function createStoryStore() {
  const store = configureStore({
    reducer: {
      [runtimeEventsApi.reducerPath]: runtimeEventsApi.reducer,
    },
    middleware: (getDefault) => getDefault().concat(runtimeEventsApi.middleware),
  });

  // Pre-seed cache with mock events — no network call, no SSE
  store.dispatch(
    runtimeEventsApi.util.updateQueryData(
      'getRecentRuntimeEvents',
      { limit: 100 },
      (draft) => {
        draft.push(...generateMockEvents(20));
      },
    )
  );

  return store;
}
```

#### Files Changed

```
MODIFY  web/src/api/runtimeEventsApi.ts     ← add onCacheEntryAdded + SSE logic
DELETE  web/src/features/runtime-events/runtimeEventFeed.ts
DELETE  web/src/features/runtime-events/runtimeEventFeed.test.ts
MODIFY  web/src/pages/RuntimeEventsPage.tsx  ← use useGetRecentRuntimeEventsQuery directly
MODIFY  web/src/pages/WorkflowDetailPage.tsx ← same
MODIFY  web/src/components/workflows/OpDetailDrawer.tsx ← same
```

---

### Design B: Redux Slice + Listener Middleware (alternative)

> This was the original proposal before discovering `onCacheEntryAdded`.
> It would work but adds more boilerplate. Kept for reference.

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

1. **Rewrite `runtimeEventsApi.ts`** — add `onCacheEntryAdded` with SSE lifecycle (see Design A above)
2. **Update consumers** — replace `useRuntimeEventFeed()` calls with `useGetRecentRuntimeEventsQuery()`
3. **Delete `runtimeEventFeed.ts`** and its test file
4. **Fix Storybook** — pre-seed RTK Query cache in story decorators
5. **Add pause/resume** — use `{ skip: paused }` query option
6. **Add tests** — test `onCacheEntryAdded` with mock EventSource
7. Then continue with **Phase 3** (WorkflowDetailPage tabs + OpTable upgrade)

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
