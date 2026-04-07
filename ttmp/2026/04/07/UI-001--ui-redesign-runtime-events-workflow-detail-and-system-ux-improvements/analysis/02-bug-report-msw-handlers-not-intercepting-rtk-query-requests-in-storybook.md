---
Title: "Bug Report: MSW Handlers Not Intercepting RTK Query Requests in Storybook"
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
Summary: "MSW handlers defined in stories are not intercepting RTK Query fetch requests in Storybook. All XHR requests return 404. This report documents the setup, what was tried, and what still fails."
LastUpdated: 2026-04-07
WhatFor: "Handover document for in-house MSW/Storybook expert to diagnose and fix."
WhenToUse: "Read this before modifying MSW handler setup or Storybook preview configuration."
---

# Bug Report: MSW Handlers Not Intercepting RTK Query Requests in Storybook

**Date:** 2026-04-07
**Severity:** 🟡 Non-blocking — Storybook renders but with error states instead of mock data
**Status:** Open — needs MSW/Storybook expertise
**Discovered in:** Phase 2B of UI-001, after adding `msw` + `msw-storybook-addon`
**Related bug:** `analysis/01-bug-analysis-runtimeeventfeed-infinite-loop.md` (the SSE loop that prompted this work)

---

## 1. Bug Report

### Symptom

In Storybook, the `RuntimeEventsPage` story fires an XHR request to `GET http://localhost:6006/api/v1/runtime-events?limit=100` which returns a **404**. The MSW handlers defined in the story's `parameters.msw.handlers` are not intercepting the request.

Console shows MSW is initialized:
```
[MSW] Mocking enabled.
Worker script URL: http://localhost:6006/mockServiceWorker.js
Worker scope: http://localhost:6006/
```

But the XHR still returns 404:
```
XHR GET http://localhost:6006/api/v1/runtime-events?limit=100 [HTTP/1.1 404 Not Found 0ms]
```

### Reproduction

1. `cd web && pnpm storybook`
2. Navigate to **Pages > RuntimeEventsPage > Default**
3. Observe: page shows "Stream: error" state, no events
4. DevTools Network: `GET /api/v1/runtime-events?limit=100` returns 404
5. Console: MSW says "Mocking enabled" but doesn't log intercepting any requests

### Expected Behavior

MSW should intercept `GET /api/v1/runtime-events?limit=100` and return the mock JSON from `runtimeEventsHandlers`. The page should show 20 mock events with "Stream: live" (after initial fetch) then "Stream: error" (SSE fails, which is expected).

---

## 2. Environment

| Tool | Version |
|------|---------|
| msw | 2.13.0 |
| msw-storybook-addon | 2.0.6 |
| storybook | 10.3.4 (framework `@storybook/react-vite`) |
| vitest | 4.1.3 |
| @storybook/addon-vitest | 10.3.4 |
| @vitest/browser-playwright | 4.1.1 |
| Vite | (whatever storybook 10.3 uses) |
| React | 19.2.4 |
| @reduxjs/toolkit | 2.11.2 |
| Browser | Firefox (also tested: same behavior expected in Chrome) |

### Storybook Config

```typescript
// .storybook/main.ts
import type { StorybookConfig } from '@storybook/react-vite';
const config: StorybookConfig = {
  staticDirs: ['../public'],           // serves mockServiceWorker.js
  stories: ["../src/**/*.mdx", "../src/**/*.stories.@(js|jsx|mjs|ts|tsx)"],
  addons: [
    "@chromatic-com/storybook",
    "@storybook/addon-vitest",
    "@storybook/addon-a11y",
    "@storybook/addon-docs",
    "@storybook/addon-onboarding"
  ],
  framework: "@storybook/react-vite"
};
```

### MSW Worker Script

Generated via `npx msw init public/ --save`:
```
web/public/mockServiceWorker.js   ← exists, served correctly at http://localhost:6006/mockServiceWorker.js
```

---

## 3. Current Setup (What We Have)

### 3.1 Preview Configuration (`.storybook/preview.tsx`)

**Current state** (after several iterations — see Section 5 for what was tried):

```typescript
import type { Preview } from '@storybook/react-vite';
import React from 'react';
import { ThemeProvider, CssBaseline } from '@mui/material';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import { theme } from '../src/theme';
import { uiSlice } from '../src/store/uiSlice';
import { runtimeEventsApi } from '../src/api/runtimeEventsApi';

function createMockStore() {
  return configureStore({
    reducer: {
      ui: uiSlice.reducer,
      [runtimeEventsApi.reducerPath]: runtimeEventsApi.reducer,
    },
    middleware: (getDefault) =>
      getDefault().concat(runtimeEventsApi.middleware),
  });
}

let mswActive = false;
try {
  const { initialize, mswLoader } = require('msw-storybook-addon');
  initialize({ onUnhandledRequest: 'bypass' });
  mswActive = true;
  module.exports = { mswLoader };
} catch {}

const preview: Preview = {
  decorators: [
    (Story) => (
      <Provider store={createMockStore()}>
        <ThemeProvider theme={theme}>
          <CssBaseline />
          <Story />
        </ThemeProvider>
      </Provider>
    ),
  ],
  loaders: mswActive
    ? [require('msw-storybook-addon').mswLoader]
    : [],
};

export default preview;
```

### 3.2 MSW Handlers (`src/mocks/runtimeEventsHandlers.ts`)

```typescript
import { http, HttpResponse } from 'msw';
import { generateMockEvents } from '../test-utils/mockRuntimeEvents';

let cachedEvents = generateMockEvents(20);

export const runtimeEventsHandlers = [
  http.get('*/api/v1/runtime-events', ({ request }) => {
    const url = new URL(request.url);
    const limit = Number(url.searchParams.get('limit') ?? 100);
    // ... filtering logic ...
    return HttpResponse.json({
      events: filtered.slice(0, limit).map((event) => ({
        id: event.id,
        // ... serialized protobuf fields ...
      })),
    });
  }),
];
```

### 3.3 Story (`src/pages/RuntimeEventsPage.stories.tsx`)

```typescript
import { runtimeEventsHandlers } from '../mocks/runtimeEventsHandlers';

const meta: Meta<typeof RuntimeEventsPage> = {
  title: 'Pages/RuntimeEventsPage',
  component: RuntimeEventsPage,
  parameters: {
    msw: {
      handlers: runtimeEventsHandlers,    // <-- handlers registered here
    },
  },
  decorators: [
    (Story) => (
      <Provider store={createStoryStore()}>
        <MemoryRouter initialEntries={['/events']}>
          <Routes><Route path="/events" element={<Story />} /></Routes>
        </MemoryRouter>
      </Provider>
    ),
  ],
};
```

### 3.4 RTK Query API (`src/api/runtimeEventsApi.ts`)

```typescript
export const runtimeEventsApi = createApi({
  reducerPath: 'runtimeEventsApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api/v1' }),
  endpoints: (builder) => ({
    getRecentRuntimeEvents: builder.query<RuntimeEventV1[], RuntimeEventsParams>({
      query: (params) => buildRuntimeEventQuery(params),  // returns "/runtime-events?..."
      // ... onCacheEntryAdded for SSE ...
    }),
  }),
});
```

The actual URL RTK Query constructs:
```
GET http://localhost:6006/api/v1/runtime-events?limit=100
```

---

## 4. Hypotheses (Why Handlers Don't Intercept)

### Hypothesis A: Handler URL Pattern Mismatch

The handler uses `http.get('*/api/v1/runtime-events', ...)`. MSW v2 URL matching:
- `*/api/v1/runtime-events` — wildcard origin, should match any host
- But maybe the `?limit=100` query string isn't matched?

**Test:** Try `http.get('http://localhost:6006/api/v1/runtime-events', ...)` with explicit origin.

**Test:** Try `http.get('*/api/v1/runtime-events*', ...)` with trailing wildcard.

### Hypothesis B: Service Worker Not Active When Request Fires

MSW's service worker activation is async. The `initialize()` call starts the worker, but RTK Query might fire the fetch before the worker is fully active.

The `mswLoader` is supposed to wait for the worker, but:
- If `initialize()` is called in a `try/require` block, the worker might not be ready when `mswLoader` runs
- The `mswLoader` should await `waitForMswReady()` internally — but does it?

**Evidence for:** MSW logs "Mocking enabled" in console, which means the worker started. But the XHR still 404s.

### Hypothesis C: RTK Query Uses `fetch` But MSW Only Intercepts `XMLHttpRequest`

MSW v2's service worker intercepts `fetch` requests. But RTK Query's `fetchBaseQuery` uses the `fetch` API.

Actually, the DevTools show the request as **XHR** not **Fetch**. This is suspicious. Storybook might be wrapping fetch or using XHR internally?

**Test:** Check if `fetchBaseQuery` actually uses `fetch` or XHR. In the browser, `fetch` requests show up in DevTools as "Fetch/XHR" type, so this might just be a DevTools display issue.

### Hypothesis D: Vite Dev Server Proxying or CORS

Storybook's Vite dev server might be intercepting `/api/v1/*` requests before they reach the service worker. The service worker only sees requests that actually hit the network, but if Vite's HMR or proxy middleware handles them first, MSW never sees them.

**Test:** Check if Storybook's Vite config has any proxy settings for `/api`.

### Hypothesis E: `require()` vs `import` Issue

The current preview.tsx uses `require('msw-storybook-addon')` to conditionally load. But `msw-storybook-addon` v2 is ESM-only. `require()` might not properly resolve the module, or might get a different instance than what the story expects.

**Evidence:** The `try { require() }` block succeeds (MSW logs show "Mocking enabled"), but maybe the handler registration path is broken because the addon instance is different.

### Hypothesis F: Story-level `parameters.msw.handlers` Not Merged

The `mswLoader` from the addon is supposed to read `parameters.msw.handlers` from the story and register them with the worker. But if the loader isn't running correctly (due to the `require()` hack), the handlers might never be registered.

**Evidence:** MSW shows "Mocking enabled" (worker is active) but doesn't log `[MSW] XX:XX GET /api/v1/runtime-events 200` (no interception).

---

## 5. What Was Tried

| Attempt | What | Result |
|---------|------|--------|
| 1 | `import { initialize, mswLoader } from 'msw-storybook-addon'` at top level, `initialize()`, `loaders: [mswLoader]` | ✅ Interactive Storybook shows "Mocking enabled" but ❌ XHR still 404. ❌ Vitest crashes: `TypeError: Cannot set properties of undefined (setting 'activationPromise')` because `setupWorker().context` is undefined in Playwright. |
| 2 | Lazy `import('msw-storybook-addon')` in loader function | ❌ Same XHR 404 in interactive Storybook. ❌ Vitest passes (no MSW = no crash) but all stories with `useGetRecentRuntimeEventsQuery` show error state. |
| 3 | `require('msw-storybook-addon')` in try/catch at top level | ❌ Same XHR 404 in interactive Storybook. ❌ Vitest crashes (same reason as attempt 1). |
| 4 | Handler URL `'/api/v1/runtime-events'` (relative) | ❌ MSW doesn't match — same 404 |
| 5 | Handler URL `'*/api/v1/runtime-events'` (wildcard origin) | ❌ Same 404 |
| 6 | `staticDirs: ['../public']` in `.storybook/main.ts` | ✅ `mockServiceWorker.js` served correctly (confirmed via curl). ❌ No change in interception behavior. |
| 7 | `upsertQueryData` to pre-seed cache instead of MSW | ⚠️ Pre-seeds cache but `onCacheEntryAdded` still fires and opens SSE, which 404s. Not a complete solution. |

### What Worked

- `mockServiceWorker.js` is correctly served at `http://localhost:6006/mockServiceWorker.js`
- MSW initializes and logs "Mocking enabled" in the console
- The service worker registers successfully (visible in DevTools > Application > Service Workers)
- Vitest passes when MSW is NOT initialized (all 155 tests pass)

### What Didn't Work

- No combination of `initialize()` + `mswLoader` + handler patterns successfully intercepts the RTK Query fetch

---

## 6. Key Files

```
web/.storybook/main.ts                          ← Storybook config
web/.storybook/preview.tsx                      ← global decorators + MSW init
web/src/mocks/runtimeEventsHandlers.ts          ← MSW handler definitions
web/src/pages/RuntimeEventsPage.stories.tsx     ← story with parameters.msw.handlers
web/src/api/runtimeEventsApi.ts                 ← RTK Query API (fetchBaseQuery)
web/src/test-utils/mockRuntimeEvents.ts         ← mock data factory
web/public/mockServiceWorker.js                 ← MSW service worker script
```

---

## 7. Constraints for the Fix

1. **Interactive Storybook must work** — MSW must intercept REST API calls and return mock data
2. **Vitest must not crash** — `npx vitest run` must pass (42 test files, 155+ tests)
3. **SSE failure must be graceful** — `EventSource` to `/api/v1/runtime-events/stream` will always 404 in Storybook; `onCacheEntryAdded` should handle this without infinite loops (already fixed in `e647cc3`)
4. **No MSW in production** — MSW is dev-only, must not leak into production builds
5. **Prefer minimal changes** — if there's a simpler alternative to MSW (e.g., RTK Query `queryFn` override), that's acceptable

---

## 8. Possible Alternative Approaches

If MSW proves too finicky with this stack, consider:

### Alternative A: RTK Query `queryFn` Override in Stories

Override the endpoint's `queryFn` in the story's store to return mock data without fetching:

```typescript
// In story decorator:
const store = configureStore({
  reducer: { /* ... */ },
  middleware: (getDefault) =>
    getDefault().concat(
      runtimeEventsApi.middleware,
    ),
});

// Override the endpoint to return mock data
// This is done by manipulating the cache directly after store creation
store.dispatch(
  runtimeEventsApi.endpoints.getRecentRuntimeEvents.initiate(
    { limit: 100 },
    { subscribe: false, forceRefetch: true },
  ),
);
```

### Alternative B: Custom `fetchBaseQuery` Wrapper for Stories

```typescript
const mockBaseQuery = () =>
  async () => ({
    data: generateMockEvents(20),
  });

const mockApi = createApi({
  baseQuery: mockBaseQuery,
  // ... same endpoints ...
});
```

### Alternative C: MSW `setupServer` for Vitest + `setupWorker` for Interactive Storybook

The `msw-storybook-addon` internally uses `setupWorker` for browser and there should be a way to use `setupServer` for the vitest Playwright environment. This might require separate preview files or a conditional import.

---

## 9. Recommended Investigation Steps

1. **Verify handler URL matching**: Add `console.log` inside the handler to see if it's ever called. Try explicit `http://localhost:6006/api/v1/runtime-events` as the URL pattern.
2. **Check MSW worker readiness**: In the browser console, run `navigator.serviceWorker.controller` — should return the active worker.
3. **Check request type**: In DevTools Network, check if the request is `fetch` or `xhr`. MSW v2 service worker intercepts `fetch` only.
4. **Try the official Storybook MSW docs exactly**: https://storybook.js.org/docs/writing-stories/mocking-data-and-modules/mocking-network-requests — follow step by step.
5. **Check if `@storybook/addon-vitest` conflicts with MSW**: The vitest addon may interfere with MSW's service worker registration.
6. **Try a minimal reproduction**: Create a fresh story that just does `fetch('/api/v1/test')` with a simple MSW handler — does that work? If yes, the issue is specific to RTK Query's fetch timing.
