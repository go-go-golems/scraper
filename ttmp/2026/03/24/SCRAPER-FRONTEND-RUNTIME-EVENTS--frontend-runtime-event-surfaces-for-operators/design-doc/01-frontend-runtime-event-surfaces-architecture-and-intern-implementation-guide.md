---
Title: Frontend runtime event surfaces architecture and intern implementation guide
Ticket: SCRAPER-FRONTEND-RUNTIME-EVENTS
Status: active
Topics:
    - scraper
    - frontend
    - react
    - api
    - events
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/api/handlers/runtime_events.go
      Note: |-
        History and SSE contract for frontend consumers
        history and SSE API contract
    - Path: pkg/api/server/server.go
      Note: |-
        Backend endpoints and event hub bootstrap that feed the frontend
        backend route registration and event hub bootstrap
    - Path: proto/scraper/runtime/v1/events.proto
      Note: |-
        Shared protobuf contract that defines every frontend runtime event field
        shared event schema used by the frontend
    - Path: web/src/App.tsx
      Note: |-
        Top-level route table where a global runtime events page would be added
        top-level route table for new event screens
    - Path: web/src/api/runtimeEventsApi.ts
      Note: |-
        Current RTK Query runtime event history client and protobuf JSON decode seam
        current history fetch and protobuf decode seam
    - Path: web/src/components/layout/AppShell.tsx
      Note: |-
        Top navigation tabs that must expose new operator-facing runtime event screens
        top navigation entrypoint for an events page
    - Path: web/src/components/workflows/OpDetailDrawer.tsx
      Note: |-
        Natural place for op-scoped live runtime event views
        natural op-scoped runtime event surface
    - Path: web/src/components/workflows/RuntimeEventList.tsx
      Note: |-
        Existing runtime event list renderer that can be generalized into shared operator surfaces
        existing shared-ish event renderer
    - Path: web/src/pages/EngineOverviewPage.tsx
      Note: |-
        Dashboard surface that still relies on polling and can add live event widgets
        overview widget integration point
    - Path: web/src/pages/QueueMonitorPage.tsx
      Note: |-
        Queue monitoring surface that can consume rate-limit and failure events
        queue-local event widget integration point
    - Path: web/src/pages/SubmitWorkflowPage.tsx
      Note: |-
        Natural place for immediate post-submit live progress UX
        post-submit live progress integration point
    - Path: web/src/pages/WorkflowDetailPage.tsx
      Note: |-
        Current workflow-local runtime event timeline implementation
        current workflow-local runtime event consumer
ExternalSources: []
Summary: Detailed current-state analysis and phased implementation guide for expanding scraper's frontend runtime event UX from a single workflow timeline into operator-grade event surfaces.
LastUpdated: 2026-03-24T21:20:07-04:00
WhatFor: Give a new engineer enough architectural context and implementation detail to extend scraper's frontend runtime event experience safely and consistently.
WhenToUse: Use before adding any new runtime event page, panel, hook, store slice, or dashboard widget in the web frontend.
---


# Frontend runtime event surfaces architecture and intern implementation guide

## Executive Summary

The runtime event backend pipeline now exists. The worker, submission path, and HTTP server all publish `RuntimeEventV1` protobuf messages, the API server stores a bounded recent history in memory, and the frontend already renders one workflow-local timeline. That means the hard platform work is done. The remaining job is product work: turn the raw event stream into operator-facing frontend surfaces that are easy to understand, safe to extend, and consistent with the rest of the application.

Today the web app has only one runtime event surface: the workflow detail page fetches recent events for a workflow and opens an `EventSource` stream for live updates in `web/src/pages/WorkflowDetailPage.tsx:74-123`. That is a good first integration, but it is intentionally narrow. There is no global event console, no op-scoped runtime event tab, no reusable streaming hook, no connection-status UI, and no dashboard widgets that turn live events into operational signals.

This document recommends a phased frontend expansion that keeps the current backend contract unchanged. The frontend should add:

1. a shared runtime event data layer and streaming hook,
2. a global operator-facing runtime events page,
3. an op-scoped runtime event tab in the workflow drawer,
4. a live submission progress panel,
5. overview and queue widgets that summarize live events,
6. tests and UX guardrails around reconnects, deduplication, and filtering.

The goal is not to redesign the transport. The goal is to use the transport that already exists and build a frontend architecture that a new engineer can extend without duplicating connection logic on every page.

## Problem Statement and Scope

### Problem

The scraper runtime event system is now technically functional, but from the frontend perspective it is still underexposed. Operators can see runtime events only while looking at a single workflow detail page. The application still lacks the screens and abstractions that would make runtime events useful as an everyday operator tool.

### What this ticket covers

This ticket covers frontend-facing design and implementation planning for:

- global event monitoring,
- workflow and op-scoped runtime event views,
- post-submission live progress,
- dashboard and queue event widgets,
- shared client-side data-flow patterns,
- testing and validation guidance.

### What this ticket does not cover

This ticket does not propose changes to:

- the protobuf event schema itself,
- Watermill transport internals,
- Redis retention policy,
- backend event kinds or backend event storage model,
- a local merged `server+worker` mode.

Those concerns were already handled or intentionally deferred in the earlier runtime event ticket.

## Definitions and System Orientation

Before implementing UI, a new engineer needs a precise mental model of what the backend already does.

### Runtime event

A runtime event is one protobuf message defined in `proto/scraper/runtime/v1/events.proto:10-27`. It has:

- stable identity via `id`,
- source via `RuntimeEventSource`,
- semantic type via `RuntimeEventKind`,
- severity via `RuntimeEventSeverity`,
- timestamp via `occurred_at`,
- optional workflow, op, site, queue, worker, request, and artifact identifiers,
- free-form structured metadata via `payload`.

### History endpoint

The API server exposes recent events at `GET /api/v1/runtime-events` in `pkg/api/server/server.go:70` and `pkg/api/handlers/runtime_events.go:21-33`. This endpoint returns JSON shaped like:

```json
{
  "events": [
    { "...protojson RuntimeEventV1..." : "..." }
  ]
}
```

Filtering is supported through query parameters parsed in `pkg/api/handlers/runtime_events.go:77-89`:

- `workflowId`
- `opId`
- `site`
- `workerId`
- `limit`

### Live stream endpoint

The API server exposes an SSE stream at `GET /api/v1/runtime-events/stream` in `pkg/api/server/server.go:71` and `pkg/api/handlers/runtime_events.go:35-75`.

The stream:

- keeps a subscriber channel open,
- emits a heartbeat comment every 15 seconds,
- sends events with `event: runtime-event`,
- sends JSON payloads as one `data:` line,
- sets `id:` when the runtime event has an ID.

### In-memory recent-event hub

The API server does not query Redis directly on every frontend request. Instead it keeps an in-memory bounded hub in `pkg/runtimeevents/hub.go:19-118`. The hub:

- stores a rolling window of recent events,
- clones events for safety,
- filters by workflow/op/site/worker,
- supports live subscriptions,
- drops messages on slow subscribers rather than blocking the server.

This matters for the frontend because the replay model is "recent API-server memory", not "full durable historical query." The UI should treat the history endpoint as a short-horizon replay cache.

### Current frontend runtime event integration

The frontend data path currently looks like this:

```text
GET /api/v1/runtime-events?workflowId=...
  -> RTK Query in web/src/api/runtimeEventsApi.ts
  -> fromJson(RuntimeEventV1Schema, ...)
  -> WorkflowDetailPage local state

GET /api/v1/runtime-events/stream?workflowId=...
  -> EventSource in WorkflowDetailPage
  -> decodeRuntimeEvent(JSON.parse(event.data))
  -> mergeRuntimeEvents(...)
  -> RuntimeEventList
```

That flow works, but all stream orchestration currently lives inside one page component.

## Current-State Analysis

This section is intentionally evidence-first. It describes what is in the repository today, not what we wish existed.

### 1. The route tree has no global runtime event page

The app routes are defined in `web/src/App.tsx:13-24`. Existing top-level pages are:

- `/`
- `/workflows`
- `/workflows/:workflowId`
- `/queues`
- `/sites`
- `/sites/:siteName`
- `/submit`

There is no `/events` or similar route. A global runtime event console does not exist.

The top navigation is defined in `web/src/components/layout/AppShell.tsx:5-18` and `:38-47`. The tab bar also has no event-monitoring entry. Any operator-facing page must be wired in two places:

- route registration in `App.tsx`,
- tab exposure or alternate entrypoint in `AppShell.tsx`.

### 2. Workflow detail already mixes polling and live event streaming

`web/src/pages/WorkflowDetailPage.tsx` is the only page that consumes runtime events. It currently does all of the following:

- polls workflow summary via `useGetWorkflowQuery` at `:46-49`,
- polls workflow ops via `useGetWorkflowOpsQuery` at `:51-54`,
- fetches recent runtime events via `useGetRecentRuntimeEventsQuery` at `:74-77`,
- copies fetched history into local React state at `:101-103`,
- opens an `EventSource` directly in a `useEffect` at `:105-123`,
- decodes each message with `decodeRuntimeEvent(JSON.parse(...))` at `:109-113`,
- merges events locally with `mergeRuntimeEvents()` at `:23-36`.

This proves the backend is usable from the browser, but it also shows the current architectural weakness: transport management is page-local and not reusable.

### 3. Runtime event history fetching is already centralized, but live streaming is not

`web/src/api/runtimeEventsApi.ts:22-44` defines one RTK Query API slice for runtime event history. It handles:

- building query parameters,
- calling `/api/v1/runtime-events`,
- decoding generated protobuf JSON using `fromJson`.

This is good. It means history fetches already have a single source of truth.

However, there is no corresponding shared abstraction for live updates:

- no `openRuntimeEventsStream(...)`,
- no `useRuntimeEventsStream(...)`,
- no common reconnect logic,
- no shared dedupe/merge policy outside the one workflow page.

### 4. The current runtime event renderer is intentionally minimal

`web/src/components/workflows/RuntimeEventList.tsx:46-101` renders:

- source chip,
- severity chip,
- timestamp,
- message,
- optional op ID,
- optional worker ID,
- one short payload summary for a small set of keys.

What it does not render:

- grouped sections by op or source,
- expandable structured payload detail,
- connection state,
- filter controls,
- severity toggles,
- virtualized long lists,
- event-detail drawers.

This is fine for a first timeline, but not enough for a global operator console.

### 5. The op drawer has a natural extension point

`web/src/components/workflows/OpDetailDrawer.tsx:145-168` already uses tabbed UI for:

- input,
- dependencies,
- result,
- artifacts,
- script,
- logs.

This component is the natural place to add an op-scoped runtime event tab. It already receives:

- selected op,
- result,
- artifacts,
- script source,
- retry actions.

Adding op-scoped runtime events here is low conceptual overhead because the drawer is already the "everything about this op" surface.

### 6. Submission and dashboard pages still rely on standard UI state and polling

`web/src/pages/SubmitWorkflowPage.tsx:64-95` submits a workflow and then:

- stores a small record in `uiSlice`,
- shows a snackbar,
- stops there.

It does not transition into a live progress view even though the backend now emits submission and workflow events.

`web/src/pages/EngineOverviewPage.tsx:8-27` polls:

- engine status every 5 seconds,
- queues every 5 seconds.

`web/src/pages/QueueMonitorPage.tsx:27-87` also polls queues every 5 seconds and still uses placeholder throughput data at `:9-25`.

This means the two operator-style pages that would benefit most from live events still use only snapshot-style data.

### 7. Shared UI state is still sparse and does not know about streaming state

The only custom Redux UI state today is `web/src/store/uiSlice.ts:11-55`. It stores:

- workflow filters,
- submit form selections,
- recent submissions.

There is no store model for:

- stream connection state,
- live event filters,
- unread counts,
- last-seen event IDs,
- active operator event console preferences.

That does not automatically mean Redux is required for runtime events, but it does show that the current store shape has no prepared place for live-event UX.

### 8. The backend contract is already sufficient for multiple frontend surfaces

The protobuf contract in `proto/scraper/runtime/v1/events.proto:10-64` already provides enough fields for the proposed frontend work. Specifically:

- source can distinguish scheduler, runner, submission, request,
- kind can distinguish workflow lifecycle, op lifecycle, queue issues, logs,
- identifiers can scope UI to workflow, op, site, queue, worker,
- payload can carry summaries such as retry details or artifact summaries.

The frontend does not need a schema redesign before the next wave of UI work.

### 9. There is already an end-to-end test that proves the frontend contract is not hypothetical

`pkg/api/server/server_test.go:219-321` proves:

- the API history endpoint returns runtime events,
- the SSE stream opens successfully,
- a submitted workflow emits events,
- worker execution emits `OP_SUCCEEDED` and `LOG_LINE`,
- the client can distinguish event kinds.

That test is crucial because it means frontend work can proceed against a validated backend contract.

## Gap Analysis

The gap between "what exists" and "what operators need" is now mostly frontend architecture and UX.

### Gap 1: No reusable streaming abstraction

Today each new screen would need to repeat:

- `EventSource` setup,
- cleanup,
- JSON parse,
- protobuf decode,
- merge logic,
- error and reconnect handling.

That is too much duplication for a growing feature.

### Gap 2: No global event-oriented operator view

Operators currently need to know a workflow ID and navigate to that workflow before they can see runtime events. That is too narrow for tasks like:

- watching all failures,
- monitoring queue rate limits,
- seeing what workers are active,
- understanding recent submission traffic.

### Gap 3: No op-scoped runtime event view

The workflow timeline shows everything for the workflow, but it is not the best place to inspect a single noisy or failing op. The op drawer is missing a runtime event tab.

### Gap 4: No live post-submit experience

The submit page stops at "workflow submitted". It does not take advantage of the fact that the server can now stream progress immediately after submission.

### Gap 5: No dashboard widgets derived from events

Overview and queue pages still show snapshot state only. They do not highlight:

- recent failures,
- recent retries,
- recent rate limits,
- active workers,
- last event times.

### Gap 6: No frontend opinion on connection state

The current workflow detail page does not show whether the stream is:

- connected,
- connecting,
- stale,
- closed after an error.

That creates ambiguous UX when live updates stop.

## Design Goals

The proposed frontend work should satisfy these goals.

### Primary goals

1. Reuse one event decode and stream lifecycle model across pages.
2. Add operator-facing surfaces without changing the backend contract.
3. Preserve the current workflow timeline while making it a consumer of shared primitives.
4. Keep implementation approachable for a new engineer.
5. Make it obvious where page-level UX belongs versus where shared runtime-event infrastructure belongs.

### Secondary goals

1. Minimize duplicated list rendering and filter logic.
2. Keep historical fetch and live stream semantics explicit.
3. Avoid prematurely centralizing every detail into Redux if component-local state is enough.
4. Leave room for future features such as unread counts, persisted filters, and global notifications.

## Proposed Solution

The recommended frontend architecture is:

```text
generated protobuf schema
  -> decodeRuntimeEvent(json)

runtime event data layer
  -> buildRuntimeEventQuery(params)
  -> useGetRecentRuntimeEventsQuery(...)
  -> useRuntimeEventStream(params)
  -> mergeRuntimeEventLists(history, live)
  -> derive connection state

shared UI components
  -> RuntimeEventFilters
  -> RuntimeEventList
  -> RuntimeEventConnectionBadge
  -> RuntimeEventPayloadPanel

pages
  -> RuntimeEventsPage (global console)
  -> WorkflowDetailPage (reuse shared hook/components)
  -> OpDetailDrawer runtime tab
  -> SubmitWorkflowPage live progress panel
  -> EngineOverviewPage recent event widgets
  -> QueueMonitorPage recent queue event widgets
```

The key design principle is to split the problem into three layers:

1. transport and decode,
2. event-specific shared UI,
3. page-specific operator workflows.

### Proposed file layout

The repository currently stores event-related frontend code across generic folders. For the next wave of work, create a dedicated feature area:

```text
web/src/features/runtime-events/
  api.ts
  stream.ts
  merge.ts
  filters.ts
  types.ts
  hooks/
    useRuntimeEventFeed.ts
    useRuntimeEventStream.ts
  components/
    RuntimeEventFilters.tsx
    RuntimeEventConnectionBadge.tsx
    RuntimeEventPayloadPanel.tsx
    RuntimeEventConsole.tsx
```

This is a recommendation, not a strict rule. If the team prefers flatter placement under `api/` and `components/`, keep that style. The important part is to stop embedding stream logic directly inside page components.

## Detailed Architecture

## 1. Shared data and streaming layer

### Recommendation

Keep `web/src/api/runtimeEventsApi.ts` as the history-fetch layer, but add a reusable streaming companion.

Suggested API:

```ts
type RuntimeEventFilter = {
  workflowId?: string;
  opId?: string;
  site?: string;
  workerId?: string;
  limit?: number;
};

type RuntimeEventConnectionState =
  | "idle"
  | "connecting"
  | "live"
  | "reconnecting"
  | "closed"
  | "error";

function buildRuntimeEventStreamUrl(filter: RuntimeEventFilter): string;

function useRuntimeEventStream(
  filter: RuntimeEventFilter,
  options?: { enabled?: boolean }
): {
  events: RuntimeEventV1[];
  connectionState: RuntimeEventConnectionState;
  lastEventAt?: Date;
  error?: string;
};
```

### Why

Without this layer, every page will duplicate:

- filter-to-query-string conversion,
- `EventSource` setup,
- cleanup,
- message parsing,
- decode errors,
- connection-state bookkeeping.

### Pseudocode

```ts
function useRuntimeEventStream(filter, options) {
  state = {
    events: [],
    connectionState: options.enabled ? "connecting" : "idle",
    error: undefined,
  }

  useEffect(() => {
    if (!options.enabled) return

    url = buildRuntimeEventStreamUrl(filter)
    es = new EventSource(url)
    setConnectionState("connecting")

    es.onopen = () => setConnectionState("live")

    es.addEventListener("runtime-event", (raw) => {
      parsed = JSON.parse(raw.data)
      decoded = decodeRuntimeEvent(parsed)
      setEvents((prev) => mergeRuntimeEventLists(prev, [decoded]))
      setConnectionState("live")
      setLastEventAt(now())
    })

    es.onerror = () => {
      setConnectionState("reconnecting")
    }

    return () => {
      es.close()
      setConnectionState("closed")
    }
  }, [stableFilterKey(filter), options.enabled])

  return state
}
```

### Important implementation detail

Use a stable serialized filter key instead of raw object identity in hook dependencies. Otherwise `EventSource` will reconnect too often.

## 2. Shared event rendering layer

The current `RuntimeEventList` component is a useful seed, but it should be expanded into a small component family.

### Recommended component split

1. `RuntimeEventList`
   - owns list layout,
   - delegates item rows,
   - optionally supports virtualization later.

2. `RuntimeEventRow`
   - renders one event summary.

3. `RuntimeEventPayloadPanel`
   - renders structured payload details,
   - expands keys like `artifactSummaries`, `error`, `retryable`, `submittedCount`.

4. `RuntimeEventFilters`
   - renders source, severity, workflow/op/site filters.

5. `RuntimeEventConnectionBadge`
   - shows `Live`, `Connecting`, `Reconnecting`, `Stale`, `Error`.

### Why this split matters

Global console UX and workflow-local UX are not identical, but they should still render the same event rows and payload conventions.

## 3. Global runtime events page

### Why this page should exist

This is the highest-value next step because it exposes the full system to operators without requiring them to start from one workflow ID.

### Route and navigation changes

Add:

- route in `web/src/App.tsx`,
- tab entry in `web/src/components/layout/AppShell.tsx`.

Recommended route:

- `/events`

Recommended navigation label:

- `Events`

### Page behavior

The page should:

- fetch recent events with a default limit, for example 100,
- stream live updates with no initial workflow filter,
- allow filters for source, severity, site, worker, workflow ID,
- support quick chips such as:
  - `Failures`
  - `Retries`
  - `Queue Limits`
  - `Submissions`
  - `Requests`
- allow clicking through to the workflow detail page when `workflowId` exists,
- optionally allow clicking through to an op-focused drawer when `opId` exists.

### Pseudocode

```ts
function RuntimeEventsPage() {
  filter = useRuntimeEventFilterState()
  history = useGetRecentRuntimeEventsQuery({ ...filter, limit: 100 })
  live = useRuntimeEventStream(filter, { enabled: true })
  events = mergeRuntimeEventLists(history.data ?? [], live.events)

  return (
    <Page>
      <RuntimeEventFilters value={filter} onChange={setFilter} />
      <RuntimeEventConnectionBadge state={live.connectionState} />
      <RuntimeEventList events={applyClientFilters(events, filter)} />
    </Page>
  )
}
```

## 4. Op-scoped runtime events in the drawer

### Why here

The op drawer is already the inspection surface for one op. Adding a runtime tab is more coherent than forcing operators to mentally filter the workflow timeline.

### Current seam

`web/src/components/workflows/OpDetailDrawer.tsx:145-168` defines the current tab strip. Add a `runtime` tab here.

### Data contract

The page-level owner should pass:

- `workflowId`,
- `opId`,
- optionally prefiltered event list,
- optionally shared connection state.

Do not make the drawer open its own independent global stream if the page already has the data. That would create duplicate `EventSource` connections.

### Recommendation

In phase 1, pass a filtered array from `WorkflowDetailPage`. In phase 2, if needed, lift the feed logic into a shared page-level hook and reuse it across the workflow timeline and drawer tab.

## 5. Live submission panel

### Current behavior

`web/src/pages/SubmitWorkflowPage.tsx:64-95` submits a workflow and shows a snackbar, but does nothing with the runtime event stream afterward.

### Proposed behavior

After successful submission:

1. keep showing the snackbar,
2. create a compact live panel for the new workflow,
3. show recent runtime events for that workflow,
4. highlight terminal state,
5. provide a button linking to `/workflows/:workflowId`.

### Why this matters

This is the smoothest way to make runtime events feel useful to users. They submit a workflow and immediately see the system doing work.

### Suggested UI

```text
+----------------------------------------------+
| Workflow event-js-demo submitted             |
| Live status: Running                         |
| [Open workflow]                              |
|----------------------------------------------|
| 21:14:03 SUBMISSION accepted                 |
| 21:14:03 WORKFLOW created                    |
| 21:14:04 OP leased event-js-demo:seed        |
| 21:14:04 LOG runner started                  |
| 21:14:05 OP succeeded event-js-demo:seed     |
+----------------------------------------------+
```

## 6. Overview and queue widgets

### Current behavior

`web/src/pages/EngineOverviewPage.tsx:8-27` and `web/src/pages/QueueMonitorPage.tsx:27-87` use polling only. Queue throughput is still placeholder data in `QueueMonitorPage.tsx:9-25`.

### Proposed event-driven widgets

On overview page:

- recent failures panel,
- recent retries panel,
- active workers panel,
- last event timestamp badge.

On queue page:

- recent `QUEUE_RATE_LIMITED` events for selected queues,
- recent failed ops grouped by queue,
- recent processing activity for the expanded queue.

These widgets should supplement polling, not replace it. Polling remains the baseline state model for durable totals and counts. Events provide "what just happened" context.

## API References

## Runtime event history

Handler:

- `pkg/api/handlers/runtime_events.go:21-33`

Request:

```http
GET /api/v1/runtime-events?workflowId=wf-123&opId=op-2&limit=50
```

Response:

```json
{
  "events": [
    {
      "schemaVersion": 1,
      "id": "6ad7...",
      "source": "RUNTIME_EVENT_SOURCE_RUNNER",
      "kind": "RUNTIME_EVENT_KIND_LOG_LINE",
      "severity": "RUNTIME_EVENT_SEVERITY_INFO",
      "occurredAt": "2026-03-24T21:16:15.67Z",
      "message": "runner completed",
      "workflowId": "wf-123",
      "opId": "op-2",
      "payload": {
        "durationMillis": 123,
        "artifactCount": 1
      }
    }
  ]
}
```

## Runtime event SSE

Handler:

- `pkg/api/handlers/runtime_events.go:35-75`

Wire format:

```text
id: <event-id>
event: runtime-event
data: {"schemaVersion":1,"id":"...","kind":"RUNTIME_EVENT_KIND_OP_SUCCEEDED",...}
```

## Protobuf contract

Source:

- `proto/scraper/runtime/v1/events.proto:10-64`

Most important frontend fields:

- `source`
- `kind`
- `severity`
- `occurred_at`
- `message`
- `workflow_id`
- `op_id`
- `site`
- `queue`
- `worker_id`
- `payload`

## File-by-File Implementation Plan

This section is intentionally explicit for a new engineer.

## Phase 0: Refactor the current workflow page into reusable primitives

### Files to change

- `web/src/api/runtimeEventsApi.ts`
- new `web/src/features/runtime-events/*` or equivalent
- `web/src/pages/WorkflowDetailPage.tsx`

### Tasks

1. Extract `mergeRuntimeEvents()` from `WorkflowDetailPage.tsx:23-36` into shared code.
2. Extract `EventSource` lifecycle from `WorkflowDetailPage.tsx:105-123` into a shared hook.
3. Keep `WorkflowDetailPage` behavior unchanged after extraction.
4. Add connection-state return values from the hook.

### Exit criteria

- workflow detail page still works,
- one shared hook now owns live stream management,
- page code gets smaller and clearer.

## Phase 1: Add global runtime events page

### Files to add

- `web/src/pages/RuntimeEventsPage.tsx`
- `web/src/components/runtime-events/RuntimeEventFilters.tsx`
- `web/src/components/runtime-events/RuntimeEventConnectionBadge.tsx`

### Files to change

- `web/src/App.tsx`
- `web/src/components/layout/AppShell.tsx`

### Tasks

1. Add `/events` route.
2. Add `Events` tab or other clear navigation entry.
3. Build a page that composes:
   - history query,
   - stream hook,
   - filters,
   - connection badge,
   - event list.
4. Add click-through navigation to workflows.

### Exit criteria

- operators can monitor runtime events without first knowing a workflow ID.

## Phase 2: Add runtime tab to op drawer

### Files to change

- `web/src/components/workflows/OpDetailDrawer.tsx`
- `web/src/pages/WorkflowDetailPage.tsx`

### Tasks

1. Add `runtime` to the drawer tab enum.
2. Derive op-filtered events from page-level workflow events.
3. Render a filtered `RuntimeEventList` in the new tab.
4. Prefer page-owned feed data to avoid duplicate streams.

### Exit criteria

- selecting an op gives an event history focused on that op.

## Phase 3: Add post-submit live progress

### Files to change

- `web/src/pages/SubmitWorkflowPage.tsx`
- `web/src/store/uiSlice.ts` if you choose to persist draft/live workflow info

### Tasks

1. After successful submit, create a live panel for the new workflow ID.
2. Fetch recent events and open the shared stream hook.
3. Show a compact event list and high-level workflow status.
4. Add an "Open workflow" action.

### Exit criteria

- submit page no longer ends with only a snackbar.

## Phase 4: Add overview and queue widgets

### Files to change

- `web/src/pages/EngineOverviewPage.tsx`
- `web/src/pages/QueueMonitorPage.tsx`
- possibly add new small overview/queue components

### Tasks

1. Add recent-failures and recent-rate-limit summaries.
2. Bind queue widgets to filtered event feeds.
3. Keep polling-based durable snapshots alongside event-based recency widgets.

### Exit criteria

- overview and queue pages show what just happened, not only current totals.

## State Management Guidance

This is the most likely place for an intern to overcomplicate the design.

### Recommendation

Use this rule:

- use RTK Query for server-backed history fetches,
- use a shared custom hook for SSE state,
- use local component state for page-specific live lists and connection badges,
- add Redux state only when filters or unread counts must be shared across routes.

### Why not put every stream event into Redux immediately

Because that would:

- complicate dedupe and cleanup,
- make transient stream state global by default,
- add reducer noise for a still-evolving UX.

The current app already uses Redux sparingly for UI state in `web/src/store/uiSlice.ts:11-55`. Follow that style unless shared state is clearly needed.

## Testing Strategy

The implementation should be testable in layers.

## 1. Pure utility tests

Add tests for:

- merge and dedupe by event ID,
- sorting by `occurredAt`,
- client-side filter predicates.

## 2. Component tests

Add tests for:

- `RuntimeEventList` rendering severity/source/payload summaries,
- filter controls,
- connection badge state display.

## 3. Stream-hook tests

Mock `EventSource` and verify:

- open state,
- message decode,
- cleanup,
- reconnect/error state transitions.

## 4. Page-level tests

For `RuntimeEventsPage`, `WorkflowDetailPage`, and `SubmitWorkflowPage`, verify:

- history appears first,
- live events append after stream messages,
- duplicate event IDs do not create duplicate rows.

## 5. End-to-end confidence

The backend already has `pkg/api/server/server_test.go:219-321`. Frontend work should complement that with browser-level tests later, not replace it.

## Risks and Failure Modes

### Risk 1: duplicate events in UI

Cause:

- history endpoint returns recent events,
- SSE sends some of the same events after page load.

Mitigation:

- dedupe by stable event ID before render.

### Risk 2: too many open streams

Cause:

- each panel opens its own `EventSource`.

Mitigation:

- prefer page-owned streams and pass filtered events downward.
- only open additional streams where the page cannot reasonably own the feed.

### Risk 3: ambiguous "not updating" UX

Cause:

- stream disconnects silently,
- user thinks no new events exist.

Mitigation:

- show connection state badges,
- optionally show last event timestamp.

### Risk 4: noisy event lists

Cause:

- request-level and debug events may overwhelm operator views.

Mitigation:

- default-hide `DEBUG`,
- add quick severity/source filters,
- use different defaults for workflow-local versus global console views.

### Risk 5: payload rendering becomes inconsistent

Cause:

- every page invents its own `payload` interpretation.

Mitigation:

- centralize structured payload rendering conventions in one shared component.

## Alternatives Considered

## Alternative 1: Keep extending `WorkflowDetailPage` only

Rejected because:

- it does not create global operator visibility,
- it duplicates logic if submit/overview/queue pages need the same feed,
- it keeps stream orchestration trapped in one page.

## Alternative 2: Put all live event state into Redux immediately

Rejected for now because:

- the product requirements are still evolving,
- the app does not yet need cross-route persistent event state,
- a shared hook plus RTK Query is simpler and more consistent with current code.

## Alternative 3: Build a global console first and ignore workflow/op contexts

Rejected because:

- operators still need contextual workflow/op inspection,
- the current workflow page already demonstrates demand for local context.

The correct approach is both:

- a global console,
- contextual local views.

## Intern Checklist

If you are the engineer implementing this, follow this order:

1. Read `proto/scraper/runtime/v1/events.proto`.
2. Read `pkg/api/handlers/runtime_events.go`.
3. Read `web/src/api/runtimeEventsApi.ts`.
4. Read `web/src/pages/WorkflowDetailPage.tsx`.
5. Extract shared streaming logic before adding new pages.
6. Add the global `/events` page.
7. Add the op drawer runtime tab.
8. Add submit-page live progress.
9. Add overview/queue widgets last.

Do not start by editing six pages at once. First make the runtime-event feed reusable.

## Open Questions

1. Should the global event console appear as a top nav tab or as a secondary operator route linked from overview?
2. Should workflow-local event lists default to showing `DEBUG` events, or should debug be hidden by default everywhere?
3. Does the product want unread counts or toast notifications for failures, or should that wait until after the console ships?
4. Should queue page widgets remain lightweight summaries, or do we eventually want queue-specific event history panels?

## References

Key current-state files:

- `web/src/App.tsx`
- `web/src/components/layout/AppShell.tsx`
- `web/src/pages/WorkflowDetailPage.tsx`
- `web/src/api/runtimeEventsApi.ts`
- `web/src/components/workflows/RuntimeEventList.tsx`
- `web/src/components/workflows/OpDetailDrawer.tsx`
- `web/src/pages/SubmitWorkflowPage.tsx`
- `web/src/pages/EngineOverviewPage.tsx`
- `web/src/pages/QueueMonitorPage.tsx`
- `pkg/api/server/server.go`
- `pkg/api/handlers/runtime_events.go`
- `pkg/runtimeevents/hub.go`
- `proto/scraper/runtime/v1/events.proto`
- `pkg/api/server/server_test.go`
