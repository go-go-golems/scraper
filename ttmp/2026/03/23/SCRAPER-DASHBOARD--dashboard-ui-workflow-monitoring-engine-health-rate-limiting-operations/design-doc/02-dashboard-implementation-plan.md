---
Title: Dashboard Implementation Plan
Ticket: SCRAPER-DASHBOARD
Status: active
Topics:
    - dashboard
    - react
    - scraper
    - frontend
    - material-ui
    - redux
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/services/engineview/service.go
      Note: Backend service wired in Phase 1
    - Path: pkg/services/catalog/service.go
      Note: Backend service wired in Phase 1
    - Path: pkg/services/submission/service.go
      Note: Backend service wired in Phase 3
ExternalSources: []
Summary: "Phased implementation plan for the scraper dashboard with dependencies, deliverables, and validation criteria"
LastUpdated: 2026-03-23T21:53:18.8115997-04:00
WhatFor: "Track implementation progress and dependencies"
WhenToUse: "When planning sprints or picking up dashboard work"
---

# Dashboard Implementation Plan

## Executive Summary

The dashboard is implemented in 5 phases, each delivering a working vertical slice. Phase 0 sets up the project skeleton. Phases 1-4 each add one major screen (Overview, Workflows, Queues, Submit). Each phase produces working components with Storybook stories before wiring to the live API, ensuring the frontend is reviewable and testable at every step.

## Dependency Graph

```
Phase 0: Skeleton
    |
    +---> Phase 1: Engine Overview
    |         |
    |         +---> Phase 2: Workflows
    |         |         |
    |         |         +---> Phase 4: Submit (uses workflow navigation)
    |         |
    |         +---> Phase 3: Queues (uses event types from Phase 1)
```

Phases 2, 3, and 4 can proceed in parallel once Phase 1 is complete.

---

## Phase 0: Project Skeleton

**Goal**: Vite + React + MUI + Redux + Storybook + Go embed wiring. No features yet, just the development loop.

### Tasks

#### 0.1 — Scaffold frontend project

- Create `frontend/` directory at repo root
- Initialize with Vite + React + TypeScript template
- Install core dependencies:
  ```
  @mui/material @mui/icons-material @emotion/react @emotion/styled
  @reduxjs/toolkit react-redux react-router-dom
  recharts
  ```
- Configure `tsconfig.json` with strict mode
- Configure path aliases (`@/components`, `@/store`, `@/api`, `@/stories`)

#### 0.2 — Set up Storybook

- Install Storybook 8 with Vite builder
- Configure global decorators in `.storybook/preview.tsx`:
  - `withMuiTheme` — wrap all stories in `ThemeProvider`
  - `withReduxProvider` — wrap all stories in `Provider` with empty store
- Create `stories/__fixtures__/factories.ts` with initial mock data factory stubs
- Write a single smoke story (`AppShell.stories.tsx`) to validate the pipeline

#### 0.3 — Set up Redux store

- Create `store/index.ts` with `configureStore`
- Create `store/uiSlice.ts` with initial UI state (currentTab, filters)
- Create API service stubs (empty RTK Query APIs):
  - `api/engineApi.ts`
  - `api/workflowApi.ts`
  - `api/queueApi.ts`
  - `api/catalogApi.ts`
  - `api/submissionApi.ts`

#### 0.4 — Set up Go embed and dev proxy

- Create `cmd/scraper-dashboard/` or add `dashboard` subcommand to existing CLI
- Wire `go:embed frontend/dist` for production builds
- Configure Vite dev server to proxy `/api/v1/*` to Go backend (port 8080)
- Add `Makefile` targets: `frontend-dev`, `frontend-build`, `frontend-storybook`

#### 0.5 — Wire HTTP API handlers (backend)

- Create `pkg/cmd/api.go` with `scraper api serve` command
- Wire the existing services into HTTP handlers:
  - `GET /api/v1/engine/status` -> `engineview.Service.EngineStatus()`
  - `GET /api/v1/sites` -> `catalog.Service.ListSites()`
  - `GET /api/v1/sites/{site}/verbs` -> `catalog.Service.ListVerbs()`
  - `GET /api/v1/workflows` -> new list endpoint (needs store query)
  - `GET /api/v1/workflows/{id}` -> `engineview.Service.Workflow()`
  - `GET /api/v1/workflows/{id}/ops` -> `engineview.Service.WorkflowOps()`
  - `POST /api/v1/sites/{site}/verbs/{verb}:submit` -> `submission.Service.Submit()`
- Use `net/http` ServeMux (matches SCRAPER-HTTP-API design)
- Add JSON response helpers and error envelope

### Deliverables

- [ ] `npm run dev` starts Vite dev server with HMR
- [ ] `npm run storybook` opens Storybook with AppShell story
- [ ] `go run ./cmd/scraper api serve` starts HTTP server with health endpoint
- [ ] `make frontend-build && go run ./cmd/scraper api serve` serves embedded SPA

### Validation

- Storybook launches with one story rendered
- `curl localhost:8080/api/v1/engine/status` returns JSON
- Vite dev server proxies API calls to Go backend

---

## Phase 1: Engine Overview Page

**Goal**: Landing page with health KPIs, op breakdown, migration status, event feed, queue preview. All widgets built with Storybook stories first, then wired to RTK Query.

### Tasks

#### 1.1 — Build atomic widgets (Storybook-first)

Build each widget with stories covering all variants before any API wiring:

| Widget | Stories | Est. |
|--------|---------|------|
| `StatCard` | default, zero-state, high-counts, loading | Small |
| `StatCardRow` | default, loading | Small |
| `WorkflowStatusChip` | all-statuses | Tiny |
| `OpStatusBreakdown` | default, all-succeeded, mostly-pending, has-failures | Small |
| `MigrationStatusCard` | up-to-date, pending | Small |
| `RecentEventsTimeline` | default, empty, rate-limited, failures | Medium |
| `QueueHealthPreview` | default, saturated, idle | Small |

#### 1.2 — Build mock data factories

- Implement `createEngineStatus()` with configurable op distribution
- Implement `createSchedulerEvent()` with all event kinds
- Implement `createQueueSnapshot()` with optional rate limiting

#### 1.3 — Wire RTK Query endpoints

- Implement `engineApi.getEngineStatus` with 5-second polling
- Implement `engineApi.getEngineEvents` with 3-second polling
- Add backend handler for events (in-memory ring buffer, 200 events max)

#### 1.4 — Compose EngineOverviewPage

- Build `EngineOverviewPage` composing all widgets
- Create page-level story with MSW mocking all API calls
- Wire to live API in the router

#### 1.5 — Add AppShell navigation

- Implement tab navigation (Overview | Workflows | Queues | Submit)
- Wire React Router with lazy-loaded route stubs

### Deliverables

- [ ] 7 widget components with 20+ Storybook stories
- [ ] Engine overview page renders with live data
- [ ] Auto-refreshes every 5 seconds
- [ ] Tab navigation works (other pages show placeholder)

### Validation

- All stories render without errors in Storybook
- Overview page shows correct counts from a running engine
- Page handles empty engine (no workflows) gracefully

---

## Phase 2: Workflows Page + Detail

**Goal**: Workflow list with filtering, workflow detail with op graph and op drawer.

### Tasks

#### 2.1 — Build workflow list widgets (Storybook-first)

| Widget | Stories | Est. |
|--------|---------|------|
| `WorkflowFilters` | default, filtered | Small |
| `WorkflowTable` | default, empty, loading, single-site | Medium |
| `WorkflowsPage` | full page with filters + table | Medium |

#### 2.2 — Build workflow detail widgets (Storybook-first)

| Widget | Stories | Est. |
|--------|---------|------|
| `WorkflowHeader` | running, succeeded, failed | Small |
| `WorkflowProgressBar` | in-progress, complete, has-failures | Small |
| `OpTable` | default, with-retries, all-succeeded | Medium |
| `OpDetailDrawer` | js-succeeded, http-succeeded, failed-retryable, running, pending | Large |
| `JsonViewer` | simple, nested, large-array, null | Small |
| `OpDagVisualization` | simple, fanout, complex, single-op | Large |
| `WorkflowDetailPage` | full page composition | Large |

#### 2.3 — Build mock data factories

- Implement `createWorkflow()` with configurable status and op count
- Implement `createWorkflowOp()` with configurable kind, status, lease
- Implement `createOpResult()` with artifacts and emitted children

#### 2.4 — Wire RTK Query endpoints

- Implement `workflowApi.listWorkflows` with filtering params and 5-second polling
- Implement `workflowApi.getWorkflow` with 3-second polling
- Implement `workflowApi.getWorkflowOps` with 3-second polling
- Implement `workflowApi.getOpResult` (on-demand, no polling)
- Add backend list endpoint with site/status filtering and pagination

#### 2.5 — Wire UI state

- Connect `WorkflowFilters` to `uiSlice.workflowFilters`
- Connect `OpTable` row click to `uiSlice.selectedOpId` + drawer open
- Connect `OpDagVisualization` node click to same

#### 2.6 — Add navigation integration

- Workflow table rows navigate to `/workflows/{id}`
- Back button returns to list preserving filters
- Op IDs in events timeline (Phase 1) link to workflow detail

### Deliverables

- [ ] 11 widget components with 25+ Storybook stories
- [ ] Workflow list with working filters and pagination
- [ ] Workflow detail with op graph, op table, and op drawer
- [ ] Op drawer shows full detail including JSON input/result

### Validation

- Filter by site shows only matching workflows
- Clicking an op in the DAG opens the drawer with correct data
- Op drawer shows dependency chain, retry state, and lease info
- Page handles workflows with 50+ ops without performance issues

---

## Phase 3: Queue Monitor Page

**Goal**: Queue status table, throughput chart, rate-limit event log.

### Tasks

#### 3.1 — Backend: Queue state query

- Add `ListQueueStates(ctx) ([]QueueState, error)` to the store
- Query joins `queue_limit_state` with live lease counts
- Wire into HTTP handler `GET /api/v1/queues`
- Add throughput aggregation endpoint `GET /api/v1/queues/throughput`

#### 3.2 — Build queue widgets (Storybook-first)

| Widget | Stories | Est. |
|--------|---------|------|
| `QueueStatusTable` | default, all-idle, saturated, no-rate-limit | Medium |
| `TokenBucketGauge` | full, depleted, refilling | Small |
| `ThroughputChart` | default, single-queue, bursty, idle | Medium |
| `RateLimitEventsLog` | default, empty | Small |
| `QueueMonitorPage` | full page composition | Medium |

#### 3.3 — Wire RTK Query endpoints

- Implement `queueApi.listQueues` with 5-second polling
- Implement `queueApi.getQueueThroughput` with 10-second polling
- Reuse `engineApi.getEngineEvents` filtered to `queue_rate_limited`

### Deliverables

- [ ] 5 widget components with 12+ Storybook stories
- [ ] Queue table shows live in-flight and token counts
- [ ] Throughput chart renders time series per queue
- [ ] Rate limit events log shows recent throttling

### Validation

- Queue table updates when worker leases new ops
- Throughput chart shows meaningful data after 5 minutes of worker activity
- Token gauge reflects actual token state from engine DB

---

## Phase 4: Submit Workflow Page

**Goal**: Site/verb picker, dynamic form from verb metadata, submission with feedback.

### Tasks

#### 4.1 — Build submit widgets (Storybook-first)

| Widget | Stories | Est. |
|--------|---------|------|
| `SitePicker` | default, selected | Small |
| `VerbPicker` | default, single-verb, loading | Small |
| `VerbParameterForm` | hackernews-seed, jsdemo-seed, nereval-seed, empty | Large |
| `SubmitButton` | default, loading, disabled | Tiny |
| `RecentSubmissionsTable` | default, empty | Small |
| `SubmitWorkflowPage` | full page composition | Medium |

#### 4.2 — Wire RTK Query endpoints

- Implement `catalogApi.listSites` (cacheable, rarely changes)
- Implement `catalogApi.listVerbs` (per-site, cacheable)
- Implement `submissionApi.submitWorkflow` (mutation)

#### 4.3 — Wire UI state

- Connect site/verb selection to `uiSlice.submitForm`
- Track recent submissions in `uiSlice.recentSubmissions` (session-local)
- After successful submission, auto-navigate to workflow detail or show link

#### 4.4 — Dynamic form generation

- Build `fieldToFormControl()` mapper:
  - `string` -> `TextField`
  - `int` / `float` -> `TextField` with `type="number"`
  - `bool` -> `Switch`
  - `stringList` -> `TextField` with chip display
  - Field with `choices` -> `Select`
- Populate defaults from verb metadata
- Show `help` text as helper text below each field
- Mark `required` fields

### Deliverables

- [ ] 6 widget components with 12+ Storybook stories
- [ ] Dynamic form renders correct fields for any site/verb
- [ ] Submission creates workflow and shows result
- [ ] Recent submissions table with links to workflow detail

### Validation

- Selecting hackernews -> seed shows base-url and max-pages fields
- Submitting creates a real workflow visible in the Workflows page
- Form respects defaults from verb metadata
- Required fields prevent submission when empty

---

## File Structure

```
frontend/
  src/
    main.tsx                          # App entry point
    App.tsx                           # Router + AppShell
    theme.ts                          # MUI theme customization

    store/
      index.ts                        # configureStore
      uiSlice.ts                      # Local UI state

    api/
      engineApi.ts                    # RTK Query: engine status, events
      workflowApi.ts                  # RTK Query: workflows, ops, results
      queueApi.ts                     # RTK Query: queue status, throughput
      catalogApi.ts                   # RTK Query: sites, verbs
      submissionApi.ts                # RTK Query: submit mutation
      types.ts                        # Shared API response types

    components/
      layout/
        AppShell.tsx
      common/
        JsonViewer.tsx
        StatusChip.tsx
      overview/
        StatCard.tsx
        StatCardRow.tsx
        OpStatusBreakdown.tsx
        MigrationStatusCard.tsx
        RecentEventsTimeline.tsx
        QueueHealthPreview.tsx
      workflows/
        WorkflowFilters.tsx
        WorkflowTable.tsx
        WorkflowHeader.tsx
        WorkflowProgressBar.tsx
        OpDagVisualization.tsx
        OpTable.tsx
        OpDetailDrawer.tsx
      queues/
        QueueStatusTable.tsx
        TokenBucketGauge.tsx
        ThroughputChart.tsx
        RateLimitEventsLog.tsx
      submit/
        SitePicker.tsx
        VerbPicker.tsx
        VerbParameterForm.tsx
        SubmitButton.tsx
        RecentSubmissionsTable.tsx

    pages/
      EngineOverviewPage.tsx
      WorkflowsPage.tsx
      WorkflowDetailPage.tsx
      QueueMonitorPage.tsx
      SubmitWorkflowPage.tsx

  stories/
    __fixtures__/
      factories.ts                    # Mock data factories
      samples.ts                      # Pre-built sample data sets
    layout/
      AppShell.stories.tsx
    overview/
      StatCard.stories.tsx
      ...
    workflows/
      WorkflowTable.stories.tsx
      OpDagVisualization.stories.tsx
      OpDetailDrawer.stories.tsx
      ...
    queues/
      QueueStatusTable.stories.tsx
      ThroughputChart.stories.tsx
      ...
    submit/
      VerbParameterForm.stories.tsx
      ...

  .storybook/
    main.ts
    preview.tsx                       # Global decorators
```

---

## Summary: Component Count

| Phase | Components | Stories (est.) | Backend Changes |
|-------|-----------|----------------|-----------------|
| 0 | 1 (AppShell) | 2 | HTTP server + 6 endpoints |
| 1 | 7 | 20 | Events ring buffer |
| 2 | 11 | 25 | Workflow list query + pagination |
| 3 | 5 | 12 | Queue state query + throughput |
| 4 | 6 | 12 | (uses existing submission service) |
| **Total** | **30** | **~71** | |

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| DAG visualization performance with large workflows | Limit to 100 nodes; collapse sub-trees beyond depth 4 |
| Stale polling data confusing operators | Show "last updated" timestamp on every polled widget |
| Dynamic form edge cases | Test all 4 built-in sites' verbs as Storybook stories |
| Go embed + Vite dev loop friction | Provide separate `make frontend-dev` and `make frontend-build` targets; dev loop never needs `go build` |
| SQLite concurrent reads during polling | Use WAL mode (already enabled); read-only connections for status queries |
