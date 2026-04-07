---
Title: "UI Design Review & Redesign Guide"
Ticket: UI-001
Status: active
Topics:
    - frontend
    - ux-design
    - ui-rework
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Comprehensive UI audit and redesign guide for the Scraper Engine dashboard, covering all 8 pages and 20+ components. Documents every broken, disconnected, or poor-UX element and proposes a detailed component-by-component redesign with ASCII wireframes, YAML layout DSL, Storybook stories, and an implementation plan suitable for a new intern."
LastUpdated: 2026-04-07T11:43:19.190170312-04:00
WhatFor: "Reference document for anyone implementing UI improvements to the Scraper Engine dashboard."
WhenToUse: "Read this before touching any frontend code in web/src."
---

# UI Design Review & Redesign Guide

**Ticket:** UI-001  
**Date:** 2026-04-07  
**Audience:** New intern joining the project — this doc assumes zero prior knowledge  
**Codebase:** `web/src/` (React 18 + MUI 6 + Redux Toolkit + RTK Query)

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [System Architecture Overview](#system-architecture-overview)
3. [Page-by-Page Audit](#page-by-page-audit)
4. [Cross-Cutting Issues](#cross-cutting-issues)
5. [Proposed Redesign](#proposed-redesign)
6. [Component Design DSL](#component-design-dsl)
7. [Storybook Stories](#storybook-stories)
8. [Implementation Plan](#implementation-plan)
9. [Open Questions](#open-questions)
10. [References](#references)

---

## 1. Executive Summary

The Scraper Engine dashboard is a single-page React application built with Material UI (MUI) that lets operators monitor and control a web scraping engine. The engine runs **workflows** (collections of **ops**) against configured **sites**, managed by a **scheduler** that dispatches work to **workers** via named **queues** with rate limiting.

The UI was built incrementally as a prototype. It works for basic monitoring, but has significant UX problems:

- **Runtime Events page** renders events as a bloated vertical list instead of a dense table; severity and source filters are single-select dropdowns instead of multi-select chips; there is no way to view historical time ranges.
- **Workflow Detail page** embeds a RuntimeEventList that has no filters, no time range, and wastes enormous vertical space.
- **Queue Monitor** has placeholder (fake) throughput data and an awkward expand/collapse pattern for queue details.
- **Sites** page makes N+1 API calls (fetches detail for every site card on the list page).
- **Overview** page is static — no navigation links, no clickable elements to drill into problems.
- **Submit Workflow** page has no feedback loop — you submit but can't see the result.
- **No error boundaries** — any component crash takes down the whole app.
- **No empty states** with guidance — just "No X found" text.
- **Inconsistent patterns** — some pages use Cards, some don't; some use skeletons, some don't.

This document audits every page and component, proposes concrete redesigns with ASCII wireframes and a YAML layout DSL, specifies Storybook stories for each new component, and provides a step-by-step implementation plan.

---

## 2. System Architecture Overview

### 2.1 What Is This System?

The Scraper Engine is a Go backend that orchestrates web scraping workflows. Think of it like a job queue system (similar to Celery or Bull) but specialized for web scraping.

**Key domain concepts:**

- **Site** — A configured scrape target (e.g., "hackernews", "slashdot"). Each site has a database of scripts and verbs (actions).
- **Verb** — A named action a site can perform (e.g., "scrape", "parse", "extract"). Verbs have parameters.
- **Script** — A JavaScript file that implements one or more verbs. Scripts live in a site's database.
- **Workflow** — A collection of ops submitted together. A workflow has a status (pending → running → succeeded/failed/canceled).
- **Op (Operation)** — A single unit of work within a workflow. An op has a kind ("js" or "http"), a queue assignment, optional dependencies on other ops, and retry configuration.
- **Queue** — A named work queue with concurrency limits (max in-flight) and optional token-bucket rate limiting. Ops are dispatched from queues to workers.
- **Worker** — A process that picks up ops from queues and executes them.
- **Runtime Event** — A structured log event emitted by the engine during execution. Events have a severity (debug/info/warn/error), a source (scheduler/worker/runner/server/submission/request), and a kind (workflow_started, op_completed, etc.).
- **Artifact** — Output produced by an op (e.g., scraped HTML, parsed JSON, execution logs). Stored server-side and fetchable by ID.

**Data flow diagram:**

```
┌─────────────┐    submit     ┌──────────────┐    dispatch    ┌────────────┐
│  Dashboard  │──────────────>│   Engine     │──────────────>│   Worker   │
│  (React)    │               │   (Go)       │               │  (Go/JS)   │
│             │<─────────────│              │<──────────────│            │
└─────────────┘   REST/SSE    └──────────────┘   result       └────────────┘
      │                        │      │                           │
      │                        │      │                           │
      │         ┌──────────────┘      │              ┌────────────┘
      │         │                     │              │
      ▼         ▼                     ▼              ▼
 ┌─────────┐  ┌──────────┐   ┌──────────────┐  ┌────────────┐
 │Runtime  │  │Workflow  │   │  Queue       │  │ Artifact   │
 │Events   │  │State     │   │  Manager     │  │ Store      │
 │(SSE)    │  │(SQLite)  │   │  (in-mem)    │  │ (disk/S3)  │
 └─────────┘  └──────────┘   └──────────────┘  └────────────┘
```

### 2.2 Frontend Architecture

**File: `web/src/components/layout/AppShell.tsx`** — The root layout. Contains a top AppBar with navigation tabs.

The app has 6 top-level routes:

```
/            → EngineOverviewPage      (engine stats + op breakdown + queue health)
/workflows   → WorkflowsPage            (list of workflows with filters)
/workflows/:id → WorkflowDetailPage     (single workflow detail + ops + events)
/queues      → QueueMonitorPage         (queue status table + throughput chart)
/events      → RuntimeEventsPage        (global runtime event viewer)
/sites       → SitesListPage            (grid of site cards)
/sites/:name → SiteDetailPage           (verbs, scripts, queue policies for a site)
/submit      → SubmitWorkflowPage       (form to submit new workflows)
```

**Technology stack:**

- **React 18** with TypeScript
- **MUI 6** (Material UI) for components
- **Redux Toolkit** for state management (`web/src/store/`)
- **RTK Query** for API data fetching (`web/src/api/`)
- **Protobuf** for runtime event types (`web/src/pb/`)
- **EventSource (SSE)** for real-time runtime event streaming

**API layer (`web/src/api/`):**

| File | Purpose | Key Endpoints |
|------|---------|---------------|
| `engineApi.ts` | Engine status | `GET /api/v1/engine/status` |
| `workflowApi.ts` | Workflow CRUD | `GET /api/v1/workflows`, `GET /api/v1/workflows/:id`, `POST /api/v1/workflows/:id/ops/:opId/retry` |
| `queueApi.ts` | Queue status | `GET /api/v1/queues` |
| `catalogApi.ts` | Site catalog | `GET /api/v1/catalog/sites`, `GET /api/v1/catalog/sites/:name` |
| `runtimeEventsApi.ts` | Event history | `GET /api/v1/runtime-events` |
| `submissionApi.ts` | Submit workflows | `POST /api/v1/submissions` |

**Real-time layer (`web/src/features/runtime-events/runtimeEventFeed.ts`):**

This is a custom hook `useRuntimeEventFeed` that:
1. Fetches recent historical events via RTK Query (`useGetRecentRuntimeEventsQuery`)
2. Opens an SSE connection to `/api/v1/runtime-events/stream`
3. Merges incoming events with history, deduplicating by event ID
4. Applies client-side filters (severity, source)
5. Returns `{ events, connectionState, lastEventAt, clearEvents }`

### 2.3 Component Tree

```
AppShell
├── EngineOverviewPage
│   ├── StatCardRow (3 stat cards: workflows, ops, queues)
│   ├── OpStatusBreakdown (pie/bar chart of op statuses)
│   └── QueueHealthPreview (mini table of queue health)
├── WorkflowsPage
│   ├── WorkflowFilters (site select, status select, search text)
│   └── WorkflowTable (sortable table of workflows)
├── WorkflowDetailPage
│   ├── WorkflowHeader (name, site, status, created)
│   ├── WorkflowProgressBar (ops progress bar)
│   ├── RuntimeEventList (embedded, NO filters)
│   ├── OpTable (table of ops in this workflow)
│   └── OpDetailDrawer (right drawer with tabs)
│       ├── Input tab (JsonViewer)
│       ├── Deps tab (list of dependency ops)
│       ├── Result tab (data, error, artifacts count)
│       ├── Artifacts tab (ArtifactList + ArtifactPreview)
│       ├── Runtime tab (RuntimeEventList for this op)
│       ├── Script tab (source code viewer)
│       └── Logs tab (OpExecutionLog)
├── QueueMonitorPage
│   ├── QueueStatusTable (queue metrics table)
│   ├── QueueDetailPanel (expanded detail per queue)
│   │   └── TokenBucketGauge
│   └── ThroughputChart (PLACEHOLDER DATA)
├── RuntimeEventsPage
│   └── RuntimeEventList (global events with filters)
├── SitesListPage
│   └── SiteCard[] (grid of cards, each triggers a detail API call)
├── SiteDetailPage
│   ├── SiteVerbList (table of verbs)
│   └── SiteScriptBrowser (file tree + source viewer)
└── SubmitWorkflowPage
    ├── SitePicker
    ├── VerbPicker
    ├── VerbParameterForm
    └── RecentSubmissionsTable
```


## 3. Page-by-Page Audit

### 3.1 Runtime Events Page (`web/src/pages/RuntimeEventsPage.tsx`)

**Current ASCII screenshot:**

```
┌─────────────────────────────────────────────────────────────────────────┐
│ ▣ Runtime Events                                                       │
│ Global operator console for recent runtime history and live streaming.  │
│                                                                         │
│ [Stream: live ✓] [47 events] [Last event 11:42:03 AM] [Clear]         │
│                                                                         │
│ ┌─────────────┐ ┌─────────┐ ┌──────────┐                               │
│ │ Workflow ID  │ │ Op ID   │ │ Site     │                               │
│ └─────────────┘ └─────────┘ └──────────┘                               │
│ ┌─────────────┐ ┌─────────┐                                           │
│ │Worker ID    │ │Severity▾│   ← SINGLE SELECT! Should be multi-select │
│ └─────────────┘ └─────────┘                                           │
│ ┌─────────────┐                                                        │
│ │Source      ▾│   ← SINGLE SELECT! Should be multi-select              │
│ └─────────────┘                                                        │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│  ┌──────────┐ ┌─────┐ ┌────────────────┐  11:42:03 AM                 │
│  │ Worker   │ │WARN │ │ Op Failed      │                               │
│  └──────────┘ └─────┘ └────────────────┘                              │
│  Connection timeout exceeded                                            │
│  Op: op-abc123  Workflow: wf-xyz789  Site: hackernews                  │
│  ─────────────────────────────────────────────────────────────────      │
│  ┌──────────┐ ┌──────┐ ┌──────────────┐  11:41:58 AM                  │
│  │ Runner   │ │INFO  │ │ Op Completed │                               │
│  └──────────┘ └──────┘ └──────────────┘                              │
│  Operation completed successfully                                      │
│  342 ms  Op: op-def456                                                 │
│  ─────────────────────────────────────────────────────────────────      │
│  ...47 events, each taking ~80px of vertical height...                 │
│  TOTAL: ~3760px of scrolling for 47 events                             │
└─────────────────────────────────────────────────────────────────────────┘
```

**Issues found:**

| # | Severity | Issue | File:Line |
|---|----------|-------|-----------|
| E1 | 🔴 Broken | **Events as list, not table** — `RuntimeEventList` renders each event as a `<ListItem>` with 3 Chips + multiline text. A single event takes ~80-100px of vertical space. 50 events = 4000px of scrolling. Should be a dense table (~32px per row). | `RuntimeEventList.tsx:107-172` |
| E2 | 🔴 Broken | **Severity is single-select** — `<TextField select>` with one value. Cannot filter to "show WARN + ERROR" simultaneously. Should be a multi-select with chip deletion. | `RuntimeEventsPage.tsx:75-85` |
| E3 | 🔴 Broken | **Source is single-select** — Same problem. Cannot select "Worker + Runner" at once. | `RuntimeEventsPage.tsx:86-96` |
| E4 | 🟡 Poor UX | **No historical time range** — Only fetches last N events (`limit: 100`). No date/time picker, no "last 1h/6h/24h/7d" selector, no custom range. Cannot investigate what happened yesterday. | `RuntimeEventsPage.tsx:62-68` |
| E5 | 🟡 Poor UX | **No pagination** — Loads up to 100 events, then live stream adds more. No way to page through older events. | `runtimeEventFeed.ts:27` |
| E6 | 🟡 Poor UX | **No auto-scroll toggle** — When live streaming, new events push the list down. No toggle to pin to top, no "new events" badge when scrolled away. | `RuntimeEventList.tsx` (missing) |
| E7 | 🟡 Poor UX | **No column sorting** — Events are always sorted by timestamp desc. No way to sort by severity, source, or kind. | `RuntimeEventList.tsx` (missing) |
| E8 | 🟠 Missing | **No event detail expansion** — Clicking an event does nothing. Should expand to show full payload, all metadata fields, and a "View Workflow" / "View Op" link. | `RuntimeEventList.tsx:120` |
| E9 | 🟠 Missing | **No event export** — No way to copy/download filtered events as JSON or CSV for debugging. | Missing feature |
| E10 | 🟠 Missing | **No pause/resume** — "Clear" button removes events. No way to pause the live stream without clearing. | `RuntimeEventsPage.tsx:71` |

### 3.2 Workflow Detail Page (`web/src/pages/WorkflowDetailPage.tsx`)

**Current ASCII screenshot:**

```
┌─────────────────────────────────────────────────────────────────────────┐
│ ← Back to Workflows                                                    │
│                                                                         │
│ ┌────────────────────────────────────────────────────┐ ┌──────────────┐│
│ │ scrape-hackernews                                   │ │ [Cancel]     ││
│ │ Site: hackernews  Status: running  Created: 5m ago  │ └──────────────┘│
│ └────────────────────────────────────────────────────┘                  │
│                                                                         │
│ ┌─────────────────────────────────────────────────────────────────────┐ │
│ │ ████████████████████░░░░░░░░  12/20 ops complete                   │ │
│ └─────────────────────────────────────────────────────────────────────┘ │
│                                                                         │
│ ┌─────────────────────────────────────────────────────────────────────┐ │
│ │ Runtime Events                        ← NO FILTERS! NO TIME RANGE! │ │
│ │ ┌──────────┐ ┌─────┐ ┌──────────────┐  11:42:03 AM                 │ │
│ │ │ Scheduler│ │INFO │ │ WorkflowStart │                               │ │
│ │ └──────────┘ └─────┘ └──────────────┘                              │ │
│ │ Workflow started                                                     │ │
│ │ 5 ops submitted                                                      │ │
│ │ ─────────────────────────────────────────────────────                │ │
│ │ ... bloated list, same as RuntimeEventsPage ...                     │ │
│ └─────────────────────────────────────────────────────────────────────┘ │
│                                                                         │
│ ┌─────────────────────────────────────────────────────────────────────┐ │
│ │ ID          │ Kind │ Queue     │ Status   │ Retry │ Created        │ │
│ │─────────────│──────│───────────│──────────│───────│────────────────│ │
│ │ op-abc123   │ JS   │ site:hn:js│ running  │ 1/3   │ 5m ago         │ │
│ │ op-def456   │ HTTP │ site:hn   │ succeeded│ -     │ 5m ago         │ │
│ │ ... more ops ...                                                    │ │
│ └─────────────────────────────────────────────────────────────────────┘ │
│                                                        [Drawer →]       │
└─────────────────────────────────────────────────────────────────────────┘
```

**Issues found:**

| # | Severity | Issue | File:Line |
|---|----------|-------|-----------|
| W1 | 🔴 Broken | **RuntimeEventList embedded with no filters** — The workflow detail page embeds `RuntimeEventList` directly but provides zero filter controls (no severity, no source, no time range). For a workflow with 200+ events, this is a wall of text. | `WorkflowDetailPage.tsx:93-96` |
| W2 | 🔴 Broken | **Runtime events between progress bar and ops table** — Events take up the most vertical space on the page, pushing the ops table (the most important element) below the fold. Events should be a collapsible panel or a tab. | `WorkflowDetailPage.tsx:88-97` |
| W3 | 🟡 Poor UX | **No workflow metadata** — The WorkflowHeader shows name, site, status, created. Missing: workflow ID (for copy), duration, error message (if failed), who submitted it. | `WorkflowHeader.tsx` |
| W4 | 🟡 Poor UX | **OpTable has no search/filter** — When a workflow has 100+ ops, there's no way to filter by status, search by ID, or sort columns. | `OpTable.tsx` (missing) |
| W5 | 🟡 Poor UX | **No visual DAG** — Ops have dependencies (`DependsOn`) but there's no visual representation. A mini DAG or dependency tree would help understand execution order. | Missing feature |
| W6 | 🟠 Missing | **No timeline view** — No way to see when each op started/completed relative to each other. A Gantt-like timeline would be very useful. | Missing feature |
| W7 | 🟠 Missing | **OpTable missing duration column** — Shows "Created" but not how long each op took. | `OpTable.tsx:56-61` |

### 3.3 Workflows List Page (`web/src/pages/WorkflowsPage.tsx`)

**Current ASCII screenshot:**

```
┌─────────────────────────────────────────────────────────────────────────┐
│ Workflows                                                               │
│                                                                         │
│ [Site ▾]  [Status ▾]  [Search______________]                           │
│                                                                         │
│ ┌─────────────────────────────────────────────────────────────────────┐ │
│ │ ID          │ Site │ Name               │ Status │ Progress│Created │ │
│ │─────────────│──────│────────────────────│────────│─────────│────────│ │
│ │ wf-abc      │ hn   │ scrape-hn          │running │███░ 60% │ 5m ago │ │
│ │ wf-def      │ sd   │ scrape-sd          │succ.   │████100% │ 1h ago │ │
│ │ wf-ghi      │ hn   │ scrape-hn-comments │failed  │██░░ 40% │ 2h ago │ │
│ └─────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

**Issues found:**

| # | Severity | Issue | File:Line |
|---|----------|-------|-----------|
| WL1 | 🟡 Poor UX | **Hardcoded site list** — `const sites = ['hackernews', 'slashdot', 'js-demo', 'nereval']` is hardcoded. Should be fetched from the catalog API. | `WorkflowsPage.tsx:12` |
| WL2 | 🟡 Poor UX | **No pagination** — Always loads `limit: 50`. No way to see older workflows. | `WorkflowsPage.tsx:24` |
| WL3 | 🟡 Poor UX | **No date range filter** — Cannot filter by "created in last hour" or "created today". | `WorkflowFilters.tsx` (missing) |
| WL4 | 🟠 Missing | **No bulk actions** — Cannot cancel multiple failed workflows at once. | Missing feature |
| WL5 | 🟠 Missing | **No column sorting** — Table columns are not sortable. | `WorkflowTable.tsx` (missing) |

### 3.4 Queue Monitor Page (`web/src/pages/QueueMonitorPage.tsx`)

**Current ASCII screenshot:**

```
┌─────────────────────────────────────────────────────────────────────────┐
│ Queue Monitor                                                           │
│                                                                         │
│ ┌─────────────────────────────────────────────────────────────────────┐ │
│ │ Queue Status                                                        │ │
│ │ Queue Key     │ Site │ In-Flight  │ Max │ Tokens │ Rate │ Burst    │ │
│ │───────────────│──────│────────────│─────│────────│──────│─────────│ │
│ │ site:hn:http  │ hn   │ ████░ 4/5 │  5  │  120   │ 10   │ 200     │ │
│ │ site:hn:js    │ hn   │ ██░░░ 2/3 │  3  │  none  │ none │ none    │ │
│ │ site:sd:http  │ sd   │ █░░░░ 1/2 │  2  │  50    │  5   │ 100     │ │
│ │                                                         │ │
│ │ ▸ site:hn:http  ← click to expand (awkward UX)        │ │
│ │ ▸ site:hn:js                                         │ │
│ │ ▾ site:sd:http  ← expanded                            │ │
│ │   ┌──────────────────────────────────────────────┐     │ │
│ │   │ site:sd:http                                  │     │ │
│ │   │ Max 2 in-flight                              │     │ │
│ │   │ [Token Bucket Gauge]                          │     │ │
│ │   │ Pending: 3  Ready: 1  Running: 1  Succ: 12  │     │ │
│ │   └──────────────────────────────────────────────┘     │ │
│ └─────────────────────────────────────────────────────────────────────┘ │
│                                                                         │
│ ┌─────────────────────────────────────────────────────────────────────┐ │
│ │ Throughput Chart (15m window)                                       │ │
│ │ ⚠️ USING PLACEHOLDER DATA — random numbers, not real metrics!      │ │
│ │ ┌──────────────────────────────────────────────────────────┐       │ │
│ │ │     ╱╲    ╱╲                                             │       │ │
│ │ │   ╱    ╲╱    ╲    ╱╲                                     │       │ │
│ │ │  ╱            ╲╱    ╲                                    │       │ │
│ │ └──────────────────────────────────────────────────────────┘       │ │
│ └─────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

**Issues found:**

| # | Severity | Issue | File:Line |
|---|----------|-------|-----------|
| Q1 | 🔴 Broken | **Throughput chart uses fake data** — `placeholderThroughput` generates `Math.random()` data. The chart is rendered but shows nothing real. No backend endpoint for throughput metrics exists. | `QueueMonitorPage.tsx:7-25` |
| Q2 | 🟡 Poor UX | **Expand/collapse is a plain text click target** — Queue detail expansion is a tiny `<Typography variant="caption">` with a ▸/▾ prefix. Should be a proper accordion or row expansion in the table itself. | `QueueMonitorPage.tsx:52-68` |
| Q3 | 🟡 Poor UX | **Detail panel below the table, not inline** — When you expand a queue, the detail appears below the table row. This pushes other rows down and is disorienting. Should use inline row expansion (like MUI `Table` with collapse). | `QueueMonitorPage.tsx:52-68` |
| Q4 | 🟠 Missing | **No time range selector for throughput** — Chart is hardcoded to "15m". No way to see 1h, 6h, 24h. | `ThroughputChart.tsx` |
| Q5 | 🟠 Missing | **No queue-level filtering** — Cannot filter to show only queues for a specific site. | Missing feature |

### 3.5 Overview Page (`web/src/pages/EngineOverviewPage.tsx`)

**Current ASCII screenshot:**

```
┌─────────────────────────────────────────────────────────────────────────┐
│ ┌──────────┐  ┌──────────┐  ┌──────────┐                              │
│ │Workflows │  │   Ops    │  │  Queues  │                              │
│ │    12    │  │   347    │  │    8     │                              │
│ │▶ 3 run   │  │▶ 200 ok  │  │▶ 6 healthy│                             │
│ │▶ 9 done  │  │▶ 147 pend│  │▶ 2 warning│                             │
│ └──────────┘  └──────────┘  └──────────┘                              │
│                                                                         │
│ ┌───────────────────────────┐  ┌──────────────────────────────────┐   │
│ │ Op Status Breakdown       │  │ Queue Health Preview             │   │
│ │ ████████ succeeded 200    │  │ site:hn:http  ████░ 4/5  ⚠️     │   │
│ │ ██████ pending    120     │  │ site:hn:js    ██░░░ 2/3  ✓      │   │
│ │ ████ running      20      │  │ site:sd:http  █░░░░ 1/2  ✓      │   │
│ │ ██ failed          7      │  │                                  │   │
│ └───────────────────────────┘  └──────────────────────────────────┘   │
│                                                                         │
│ ⚠️ Nothing is clickable! No drill-down links!                          │
└─────────────────────────────────────────────────────────────────────────┘
```

**Issues found:**

| # | Severity | Issue | File:Line |
|---|----------|-------|-----------|
| O1 | 🟡 Poor UX | **Nothing is clickable** — StatCards, OpStatusBreakdown, and QueueHealthPreview are all display-only. Clicking "347 ops" or "7 failed" should navigate to filtered views. | `StatCard.tsx`, `OpStatusBreakdown.tsx`, `QueueHealthPreview.tsx` |
| O2 | 🟡 Poor UX | **No refresh indicator** — Auto-polls every 5s but no visual feedback when data updates. | `EngineOverviewPage.tsx:13` |
| O3 | 🟠 Missing | **No recent activity feed** — The overview page is all aggregates. No "last 5 events" or "recently completed workflows" to give operators situational awareness. | Missing feature |
| O4 | 🟠 Missing | **No alert summary** — When there are failed ops or error-level events, the overview should prominently surface an alert banner. | Missing feature |

### 3.6 Sites Pages (`SitesListPage.tsx`, `SiteDetailPage.tsx`)

**Issues found:**

| # | Severity | Issue | File:Line |
|---|----------|-------|-----------|
| S1 | 🔴 Broken | **N+1 API calls on list page** — `SitesListPage` renders `SiteCardWithDetail` which calls `useGetSiteDetailQuery` for every site. With 10 sites, that's 10 parallel detail fetches. Should use a list endpoint that returns enough data for cards. | `SitesListPage.tsx:10-23` |
| S2 | 🟡 Poor UX | **Site detail page: Overview tab is sparse** — Just "Queue Policies" table and 3 stat lines. Should show recent workflows, health status, and link to submit. | `SiteDetailPage.tsx:65-99` |
| S3 | 🟠 Missing | **No "Submit Workflow from this site" action** — On a site detail page, there's no quick link to submit a workflow for this site. | Missing feature |

### 3.7 Submit Workflow Page (`SubmitWorkflowPage.tsx`)

**Issues found:**

| # | Severity | Issue | File:Line |
|---|----------|-------|-----------|
| SUB1 | 🟡 Poor UX | **No feedback after submission** — After clicking submit, the form resets but there's no success toast, no link to the created workflow, no "view workflow" button. User has to manually navigate to Workflows to find it. | `SubmitWorkflowPage.tsx` |
| SUB2 | 🟡 Poor UX | **Recent submissions table has no status polling** — Shows recently submitted workflows but doesn't update their status. Stale data. | `RecentSubmissionsTable.tsx` |

### 3.8 Cross-Component Issues

| # | Severity | Issue | Files Affected |
|---|----------|-------|----------------|
| X1 | 🔴 Broken | **No error boundary** — Any unhandled React error crashes the entire app to a white screen. | `App.tsx` or root layout |
| X2 | 🟡 Poor UX | **Inconsistent loading states** — Some pages show skeletons, some show "Loading..." text, some show nothing. | Multiple pages |
| X3 | 🟡 Poor UX | **No breadcrumbs** — Navigation relies solely on "Back" buttons. No breadcrumb trail for deep pages (e.g., Workflows → wf-abc → op-xyz). | `AppShell.tsx` |
| X4 | 🟡 Poor UX | **No keyboard shortcuts** — No global shortcuts for common actions (refresh, navigate to events, toggle auto-scroll). | Missing feature |
| X5 | 🟠 Missing | **No dark mode** — No theme toggle. App is always light mode. | `App.tsx` theme provider |
| X6 | 🟠 Missing | **No toast/notification system** — No global way to show success/error/info messages. Success/failure of mutations is invisible. | Missing feature |

---

## 4. Cross-Cutting Issues

### 4.1 Component Reuse Problems

The `RuntimeEventList` component is used in three places:
1. `RuntimeEventsPage` (global events with filters)
2. `WorkflowDetailPage` (workflow-scoped events, no filters)
3. `OpDetailDrawer` → Runtime tab (op-scoped events, no filters)

All three use the same bloated list layout. The fix must create a single `RuntimeEventTable` component that can be configured for all three contexts.

### 4.2 Data Fetching Anti-Patterns

- **Polling everywhere** — Most pages use `pollingInterval: 3000` or `5000`. This works but creates unnecessary network traffic. Consider using SSE/WebSocket for state changes and only polling when SSE is disconnected.
- **No optimistic updates** — Retry/cancel mutations wait for server response before updating UI. Should optimistically update the status chip.
- **No cache invalidation coordination** — Mutating a workflow doesn't automatically invalidate related queries (ops, events).

### 4.3 Accessibility Issues

- No `aria-label` on interactive table rows
- No `role="button"` on clickable `Typography` elements
- Color-only status indicators (StatusChip colors) without text alternatives
- No focus management when drawers open/close

---

## 5. Proposed Redesign

### 5.1 Runtime Events Page — Redesigned

**Proposed ASCII wireframe:**

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│ ▣ Runtime Events                                                [⚙️] [📎 Export] │
│                                                                                 │
│ ┌─ Time Range ─────────────────────────────────────────────────────────────┐    │
│ │ [● Live] [Last 1h] [Last 6h] [Last 24h] [Last 7d]  [Custom: ___-___]   │    │
│ └──────────────────────────────────────────────────────────────────────────┘    │
│                                                                                 │
│ ┌─ Filters ────────────────────────────────────────────────────────────────┐    │
│ │ Severity: [ERROR ×] [WARN ×] [+ Add]  ← MULTI-SELECT CHIPS             │    │
│ │ Source:   [Worker ×] [Runner ×] [+ Add]  ← MULTI-SELECT CHIPS          │    │
│ │ Workflow: [____________]  Op: [________]  Site: [________]              │    │
│ │ Worker:   [____________]                                                │    │
│ └──────────────────────────────────────────────────────────────────────────┘    │
│                                                                                 │
│ [Stream: live ●] [247 events] [Last event 11:42:03 AM] [⏸ Pause] [🗑 Clear]  │
│                                                                                 │
│ ┌──────────────────────────────────────────────────────────────────────────┐    │
│ │ Timestamp     │ Severity │ Source     │ Kind          │ Message          │    │
│ │───────────────│──────────│────────────│───────────────│──────────────────│    │
│ │ 11:42:03 AM   │ ● WARN   │ Worker     │ OpFailed      │ Connection ti... │    │
│ │ 11:41:58 AM   │ ● INFO   │ Runner     │ OpCompleted   │ Success (342ms)  │    │
│ │ 11:41:55 AM   │ ● ERROR  │ Runner     │ OpError       │ HTTP 503 from... │    │
│ │ 11:41:52 AM   │ ● INFO   │ Scheduler  │ OpDispatched  │ Dispatched to... │    │
│ │ ...          │          │            │               │                  │    │
│ │ ← 32px per row, ~30 rows visible in viewport without scrolling         │    │
│ └──────────────────────────────────────────────────────────────────────────┘    │
│                                                                                 │
│ ▸ Clicking a row expands to:                                                   │
│ ┌──────────────────────────────────────────────────────────────────────────┐    │
│ │ 11:42:03 AM  │ ● WARN  │ Worker  │ OpFailed                            │    │
│ │ Message: Connection timeout exceeded                                    │    │
│ │ Op: op-abc123  [→ View Op]   Workflow: wf-xyz789  [→ View Workflow]    │    │
│ │ Site: hackernews   Worker: worker-03                                    │    │
│ │ Payload: { attempt: 2, errorCode: "TIMEOUT", retryable: true }          │    │
│ └──────────────────────────────────────────────────────────────────────────┘    │
│                                                                                 │
│ [Showing 1-50 of 247]  [← Prev]  [1] [2] [3] [4] [5]  [Next →]             │
└─────────────────────────────────────────────────────────────────────────────────┘
```

**Component DSL:**

```yaml
# RuntimeEventsPage v2 layout
page:
  id: RuntimeEventsPage
  title: Runtime Events
  layout: vertical-stack
  
  children:
    - id: RuntimeEventsToolbar
      layout: horizontal-stack
      children:
        - id: ConnectionIndicator
          type: chip
          props: { color: dynamic }
        - id: EventCountBadge
          type: chip
          variant: outlined
        - id: LastEventTimestamp
          type: chip
          variant: outlined
        - id: PauseButton
          type: icon-button
          props: { icon: Pause|PlayArrow }
        - id: ClearButton
          type: button
          variant: outlined
        - id: ExportButton
          type: button
          variant: outlined
          props: { icon: FileDownload }

    - id: TimeRangeSelector
      type: button-group
      options: [Live, Last 1h, Last 6h, Last 24h, Last 7d, Custom]
      props: { exclusive: false }  # Custom opens a DateRangePicker

    - id: RuntimeEventFilters
      layout: vertical-stack
      children:
        - id: SeverityFilter
          type: multi-select-chips
          # Uses MUI Autocomplete with multiple + renderTags
          options: [DEBUG, INFO, WARN, ERROR]
          props: { chipColor: severity-mapped }
        - id: SourceFilter
          type: multi-select-chips
          options: [Scheduler, Worker, Runner, Server, Submission, Request]
        - id: TextFilters
          layout: grid-3-col
          children:
            - id: WorkflowIdInput
              type: text-field
              props: { size: small, label: "Workflow ID" }
            - id: OpIdInput
              type: text-field
              props: { size: small, label: "Op ID" }
            - id: SiteInput
              type: text-field
              props: { size: small, label: "Site" }

    - id: RuntimeEventTable
      type: data-table
      props:
        dense: true
        expandableRows: true
        columns:
          - { field: timestamp, label: Time, width: 100, sortable: true }
          - { field: severity, label: Severity, width: 80, sortable: true, render: severityDot }
          - { field: source, label: Source, width: 100, sortable: true }
          - { field: kind, label: Kind, width: 140, sortable: true }
          - { field: message, label: Message, flex: 1, truncate: 60 }
        expandRow:
          component: RuntimeEventDetail
          props:
            - message
            - opId
            - workflowId
            - site
            - workerId
            - payload
            - links: [ViewOp, ViewWorkflow]

    - id: Pagination
      type: table-pagination
      props: { rowsPerPageOptions: [25, 50, 100] }
```

### 5.2 Workflow Detail Page — Redesigned

**Proposed ASCII wireframe:**

```
┌─────────────────────────────────────────────────────────────────────────┐
│ Workflows > scrape-hackernews > wf-abc123     [Copy ID] [Cancel]       │
│                                                                         │
│ ┌──────────────────────────────────────────────────────────────────┐    │
│ │ scrape-hackernews          Site: hackernews  Status: ● running   │    │
│ │ Duration: 5m 23s  Started: 11:36 AM  Submitted by: dashboard    │    │
│ │ ████████████████████░░░░░░░░  12/20 ops complete                 │    │
│ └──────────────────────────────────────────────────────────────────┘    │
│                                                                         │
│ ┌─ Tabs ─────────────────────────────────────────────────────────────┐  │
│ │ [Operations (20)] [Runtime Events (47)] [Artifacts (8)] [JSON]    │  │
│ └────────────────────────────────────────────────────────────────────┘  │
│                                                                         │
│ ══ Operations Tab (default) ═════════════════════════════════════════   │
│                                                                         │
│ [Status filter: ● All ▾]  [Search___________]  [Sort: Created ▾]      │
│                                                                         │
│ ┌─────────────────────────────────────────────────────────────────────┐ │
│ │ ID          │ Kind │ Queue     │ Status   │ Retry │ Duration│Created│ │
│ │─────────────│──────│───────────│──────────│───────│─────────│───────│ │
│ │ op-abc123   │ JS   │ site:hn:js│ ● running│ 1/3   │ 2m 14s  │ 5m ago│ │
│ │ op-def456   │ HTTP │ site:hn   │ ✓ succ.  │   -   │  342ms  │ 5m ago│ │
│ │ op-ghi789   │ JS   │ site:hn:js│ ✗ failed │ 3/3   │  1m 02s │ 4m ago│ │
│ └─────────────────────────────────────────────────────────────────────┘ │
│                                                                         │
│ ══ Runtime Events Tab ═══════════════════════════════════════════════   │
│                                                                         │
│ [Same RuntimeEventTable as global page, but pre-filtered to workflow]  │
│ [Severity: multi-select]  [Source: multi-select]  [Time range selector]│
│                                                                         │
│ ══ Artifacts Tab ════════════════════════════════════════════════════   │
│                                                                         │
│ ┌──────────────────────────────────────────────────────────────────┐    │
│ │ Name              │ Op        │ Type       │ Size   │ Actions    │    │
│ │───────────────────│───────────│────────────│────────│────────────│    │
│ │ page-1.html       │ op-def456 │ text/html  │ 45 KB  │ [⬇] [👁]  │    │
│ │ parsed-items.json │ op-abc123 │ applicati… │ 12 KB  │ [⬇] [👁]  │    │
│ └──────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

**Component DSL:**

```yaml
# WorkflowDetailPage v2 layout
page:
  id: WorkflowDetailPage
  layout: vertical-stack
  
  children:
    - id: BreadcrumbNav
      type: breadcrumb
      crumbs:
        - { label: Workflows, path: /workflows }
        - { label: "${workflow.name}", path: /workflows/${id} }
      actions:
        - id: CopyIdButton
          type: icon-button
          props: { icon: ContentCopy, tooltip: "Copy workflow ID" }
        - id: CancelWorkflowButton
          type: button
          variant: outlined
          color: warning

    - id: WorkflowSummaryCard
      layout: vertical-stack
      children:
        - id: WorkflowHeader
          type: header
          fields: [name, site, status, duration, startedAt, submittedBy]
        - id: WorkflowProgressBar
          type: progress-bar
          props: { showFraction: true }

    - id: WorkflowTabs
      type: tabs
      tabs:
        - label: "Operations (${opCount})"
          id: operations
          children:
            - id: OpFilters
              layout: horizontal-stack
              children:
                - id: OpStatusFilter
                  type: select
                  options: [All, Pending, Ready, Running, Succeeded, Failed, Canceled]
                - id: OpSearch
                  type: text-field
                  props: { placeholder: "Search by op ID..." }
            - id: OpTable
              type: data-table
              props:
                columns:
                  - { field: id, label: ID, width: 120, monospace: true }
                  - { field: kind, label: Kind, width: 60, render: kindIcon }
                  - { field: queue, label: Queue, width: 120 }
                  - { field: status, label: Status, width: 90, render: statusChip }
                  - { field: retry, label: Retry, width: 60 }
                  - { field: duration, label: Duration, width: 80, sortable: true }
                  - { field: created, label: Created, width: 80, sortable: true }
                onRowClick: openOpDetailDrawer

        - label: "Runtime Events (${eventCount})"
          id: runtime-events
          children:
            - id: WorkflowRuntimeEventTable
              type: RuntimeEventTable  # REUSED from global page
              props:
                serverFilters: { workflowId: "${workflowId}" }
                showFilters: true

        - label: "Artifacts (${artifactCount})"
          id: artifacts
          children:
            - id: WorkflowArtifactTable
              type: data-table
              columns:
                - { field: name, label: Name }
                - { field: opId, label: Op }
                - { field: contentType, label: Type }
                - { field: size, label: Size }
                - { field: actions, label: Actions, render: [download, preview] }

        - label: JSON
          id: json
          children:
            - id: JsonViewer
              type: json-viewer
              props: { data: workflow }
```

### 5.3 Queue Monitor Page — Redesigned

**Proposed ASCII wireframe:**

```
┌─────────────────────────────────────────────────────────────────────────┐
│ Queue Monitor                                                           │
│                                                                         │
│ ┌─────────────────────────────────────────────────────────────────────┐ │
│ │ Queue Key     │ Site │ Utilization │ Tokens │ Rate │ Ops (P/R/S/F) │ │
│ │───────────────│──────│─────────────│────────│──────│───────────────│ │
│ │ ▸ site:hn:http│ hn   │ ████░ 4/5  │  120   │ 10/s │ 3 1 4 2      │ │
│ │ ▸ site:hn:js  │ hn   │ ██░░░ 2/3  │  none  │  -   │ 5 2 8 1      │ │
│ │ ▾ site:sd:http│ sd   │ █░░░░ 1/2  │  50    │  5/s │ 1 0 3 0      │ │
│ │  ┌──────────────────────────────────────────────────────────────┐  │ │
│ │  │ [Token Bucket Gauge]  Tokens: 50/100  Rate: 5/sec           │  │ │
│ │  │ Policy: max 2 in-flight, token bucket burst 100             │  │ │
│ │  └──────────────────────────────────────────────────────────────┘  │ │
│ └─────────────────────────────────────────────────────────────────────┘ │
│                                                                         │
│ ┌─ Throughput ──────────────────────────────────────────────────────┐   │
│ │ [15m] [1h] [6h] [24h]                  ← TIME RANGE SELECTOR     │   │
│ │ ┌──────────────────────────────────────────────────────────────┐  │   │
│ │ │  REAL DATA from /api/v1/queues/:key/metrics                 │  │   │
│ │ │  (requires backend endpoint)                                  │  │   │
│ │ └──────────────────────────────────────────────────────────────┘  │   │
│ └─────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

**Component DSL:**

```yaml
# QueueMonitorPage v2 layout
page:
  id: QueueMonitorPage
  layout: vertical-stack
  
  children:
    - id: QueueStatusTable
      type: data-table
      props:
        expandableRows: true  # inline expansion, not separate accordion
        columns:
          - { field: queueKey, label: "Queue Key", monospace: true }
          - { field: site, label: Site, render: chip }
          - { field: utilization, label: Utilization, render: progressBar }
          - { field: tokens, label: Tokens, align: right }
          - { field: ratePerSecond, label: Rate, align: right }
          - { field: opBreakdown, label: "Ops (P/R/S/F)", render: compactStat }
        expandRow:
          component: QueueDetailPanel
          children:
            - TokenBucketGauge
            - PolicySummary

    - id: ThroughputSection
      layout: card
      children:
        - id: TimeRangeSelector
          type: button-group
          options: [15m, 1h, 6h, 24h]
        - id: ThroughputChart
          type: line-chart
          # NOTE: Requires backend endpoint GET /api/v1/queues/:key/metrics
          props: { realData: true }
```

### 5.4 Overview Page — Redesigned

**Proposed ASCII wireframe:**

```
┌─────────────────────────────────────────────────────────────────────────┐
│ ┌──────────┐  ┌──────────┐  ┌──────────┐                              │
│ │Workflows │  │   Ops    │  │  Queues  │                              │
│ │    12    │  │   347    │  │    8     │                              │
│ │▶ 3 run   │  │▶ 200 ok  │  │▶ 6 healthy│                             │
│ │▶ 9 done  │  │▶ 147 pend│  │▶ 2 warning│                             │
│ │  [→ View]│  │  [→ View]│  │  [→ View]│    ← CLICKABLE              │
│ └──────────┘  └──────────┘  └──────────┘                              │
│                                                                         │
│ ┌─ Alert Banner (only shown when errors exist) ──────────────────────┐  │
│ │ ⚠️ 7 ops failed in the last hour  [View Failed Ops →]             │  │
│ └─────────────────────────────────────────────────────────────────────┘  │
│                                                                         │
│ ┌───────────────────────────┐  ┌──────────────────────────────────┐   │
│ │ Op Status Breakdown       │  │ Queue Health                     │   │
│ │ ████████ succeeded 200    │  │ site:hn:http  ████░ 4/5  ⚠️     │   │
│ │ ██████ pending    120     │  │   [→ View Queue Detail]          │   │
│ │ ████ running      20     │  │ site:hn:js    ██░░░ 2/3  ✓      │   │
│ │ ██ failed          7     │  │ site:sd:http  █░░░░ 1/2  ✓      │   │
│ │     [→ View Failed]      │  │                                  │   │
│ └───────────────────────────┘  └──────────────────────────────────┘   │
│                                                                         │
│ ┌─ Recent Activity ──────────────────────────────────────────────────┐  │
│ │ 11:42 AM  ▸ Op op-abc123 failed (worker-03, site:hn)              │  │
│ │ 11:41 AM  ▸ Workflow scrape-hn completed (15/20 ops, 3 failed)    │  │
│ │ 11:38 AM  ▸ Workflow scrape-sd started (8 ops)                    │  │
│ │            [→ View All Events]                                     │  │
│ └─────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

### 5.5 Global Improvements

```yaml
# Cross-cutting component additions
global:
  - id: ErrorBoundary
    type: react-error-boundary
    props:
      fallback: ErrorFallback  # friendly error page with retry button
    wrap: AppShell children

  - id: ToastNotifications
    type: snackbar-provider
    position: bottom-right
    useCases:
      - workflow submitted successfully
      - op retry initiated
      - workflow canceled
      - export complete

  - id: BreadcrumbNav
    type: breadcrumb
    location: below AppShell AppBar
    examples:
      - "Workflows > scrape-hn > wf-abc"
      - "Sites > hackernews > Verbs"
      - "Events (filtered: workflow wf-abc)"

  - id: ThemeToggle
    type: icon-button
    location: AppBar far-right
    props: { icon: Brightness4 }

  - id: KeyboardShortcuts
    bindings:
      "r": refresh current page data
      "e": navigate to /events
      "w": navigate to /workflows
      "/": focus search field (if available)
      "Escape": close any open drawer
```

---

## 6. Component Design DSL

This section provides a concise YAML specification for every new or redesigned component. Use this as a blueprint — each YAML block describes props, layout, and behavior.

### 6.1 RuntimeEventTable (replaces RuntimeEventList)

```yaml
component:
  name: RuntimeEventTable
  file: web/src/components/workflows/RuntimeEventTable.tsx
  stories: web/src/components/workflows/RuntimeEventTable.stories.tsx
  replaces: RuntimeEventList.tsx
  
  props:
    events: "RuntimeEventV1[]"
    loading: "boolean"
    dense: "boolean  # default true = 32px rows, false = 48px rows"
    expandable: "boolean  # default true"
    showFilters: "boolean  # default false — show inline severity/source chips"
    serverFilters: "{ workflowId?, opId?, site?, workerId?, limit?, since?, until? }"
    onWorkflowClick: "(workflowId: string) => void"
    onOpClick: "(opId: string) => void"
    emptyMessage: "string"

  subcomponents:
    - RuntimeEventDetailRow
    - SeverityDotIndicator
    - TimeRangeSelector
    - MultiSelectChipFilter

  state:
    - expandedRowId: "string | null"
    - sortField: "'timestamp' | 'severity' | 'source' | 'kind'"
    - sortDir: "'asc' | 'desc'"
    - page: "number"
    - rowsPerPage: "number (25|50|100)"
```

### 6.2 MultiSelectChipFilter

```yaml
component:
  name: MultiSelectChipFilter
  file: web/src/components/common/MultiSelectChipFilter.tsx
  stories: web/src/components/common/MultiSelectChipFilter.stories.tsx
  
  props:
    label: "string"
    options: "{ value: string, label: string, color?: string }[]"
    selected: "string[]"
    onChange: "(selected: string[]) => void"
  
  behavior:
    - Renders an MUI Autocomplete with `multiple` prop
    - Selected items render as deletable Chips with the specified color
    - Unselected items available in dropdown
    - "All" is not an option — empty selection = show all

  pseudocode: |
    function MultiSelectChipFilter({ label, options, selected, onChange }) {
      return (
        <Autocomplete
          multiple
          options={options}
          value={options.filter(o => selected.includes(o.value))}
          onChange={(_, newValue) => onChange(newValue.map(v => v.value))}
          renderTags={(value, getTagProps) =>
            value.map((option, index) => (
              <Chip
                label={option.label}
                color={option.color}
                {...getTagProps({ index })}
              />
            ))
          }
          renderInput={(params) => (
            <TextField {...params} label={label} size="small" />
          )}
        />
      )
    }
```

### 6.3 TimeRangeSelector

```yaml
component:
  name: TimeRangeSelector
  file: web/src/components/common/TimeRangeSelector.tsx
  stories: web/src/components/common/TimeRangeSelector.stories.tsx
  
  props:
    value: "{ mode: 'live' | 'relative' | 'absolute', range?: string, from?: Date, to?: Date }"
    onChange: "(value) => void"
    options: "string[]  # default: ['live', '1h', '6h', '24h', '7d', 'custom']"
  
  behavior:
    - "Live" mode = no time filter, streams new events
    - "Relative" modes (1h, 6h, 24h, 7d) = compute `since` from now
    - "Custom" mode = opens MUI DateRangePicker

  pseudocode: |
    function TimeRangeSelector({ value, onChange, options }) {
      return (
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
          {options.map(opt => (
            <Chip
              key={opt}
              label={opt === 'live' ? '● Live' : opt === 'custom' ? 'Custom' : `Last ${opt}`}
              variant={isActive(opt) ? 'filled' : 'outlined'}
              onClick={() => handleSelect(opt)}
            />
          ))}
          {value.mode === 'custom' && (
            <DateRangePicker
              value={[value.from, value.to]}
              onChange={([from, to]) => onChange({ mode: 'absolute', from, to })}
            />
          )}
        </Box>
      )
    }
```

### 6.4 OpTable v2

```yaml
component:
  name: OpTable
  file: web/src/components/workflows/OpTable.tsx  # rewrite in place
  stories: web/src/components/workflows/OpTable.stories.tsx
  
  props:
    ops: "WorkflowOp[]"
    selectedOpId: "string | null"
    onSelectOp: "(id: string) => void"
    statusFilter: "string  # for client-side filtering"
    searchText: "string  # for client-side search by ID"
    sortBy: "string"
    sortDir: "'asc' | 'desc'"
  
  new_columns:
    - id: { label: ID, width: 120, monospace: true, sortable: false }
    - kind: { label: Kind, width: 60, render: kindIcon, sortable: false }
    - queue: { label: Queue, width: 120, sortable: false }
    - status: { label: Status, width: 90, render: statusChip, sortable: true }
    - retry: { label: Retry, width: 60, sortable: false }
    - duration: { label: Duration, width: 80, sortable: true, NEW }
    - created: { label: Created, width: 80, sortable: true }
  
  new_behavior:
    - Column headers are clickable for sorting
    - Status column filter via dropdown above table
    - Search field filters by op ID (client-side substring match)
    - Duration column computes from op timing data
```

### 6.5 WorkflowSummaryCard v2

```yaml
component:
  name: WorkflowSummaryCard
  file: web/src/components/workflows/WorkflowSummaryCard.tsx
  replaces: WorkflowHeader.tsx + WorkflowProgressBar inline usage
  
  props:
    workflow: "WorkflowDetail"
    stats: "WorkflowStats"
    onCancel: "() => void"
    cancelLoading: "boolean"
  
  layout:
    - row1: [name (h5), site chip, status chip, duration, copy-id-button, cancel-button]
    - row2: [started-at, submitted-by, metadata key-values]
    - row3: [progress bar with fraction]
```

### 6.6 AlertBanner

```yaml
component:
  name: AlertBanner
  file: web/src/components/common/AlertBanner.tsx
  stories: web/src/components/common/AlertBanner.stories.tsx
  
  props:
    severity: "'error' | 'warning' | 'info'"
    message: "string"
    action: "{ label: string, onClick: () => void }"
    dismissible: "boolean  # default true"
  
  behavior:
    - Renders MUI Alert with severity color
    - Optional action button on the right
    - Dismissible with close icon
    - Auto-dismiss after 30s for 'info' severity
  
  useCases:
    - "7 ops failed in the last hour" → [View Failed Ops]
    - "Queue site:hn:http at 95% capacity" → [View Queue]
    - "Throughput data unavailable" → [Dismiss]
```

---

## 7. Storybook Stories

Every new or redesigned component must have Storybook stories. Here are the specifications.

### 7.1 RuntimeEventTable.stories.tsx

```typescript
// Pseudocode for stories — do NOT implement as code, just document the stories

// Story: Default (empty)
// - 0 events, show empty message "No runtime events matched the current filters."

// Story: WithEvents
// - 20 mock events with mixed severities (5 debug, 8 info, 4 warn, 3 error)
// - Mixed sources (scheduler, worker, runner, server)
// - Show the table in default dense mode
// - Events should have realistic timestamps within the last hour

// Story: ExpandedRow
// - Same as WithEvents but with the 3rd row expanded
// - Expanded row shows full payload, op/workflow links

// Story: WithFilters
// - Show the inline filter bar (severity multi-select + source multi-select)
// - Pre-select ERROR and WARN in severity filter

// Story: Loading
// - Show skeleton rows (5 skeleton rows)

// Story: StreamLive
// - Show connection indicator as "live" (green)
// - Simulate a new event being added every 2 seconds

// Story: StreamError
// - Show connection indicator as "error" (red)
// - Show last event timestamp from 5 minutes ago
```

### 7.2 MultiSelectChipFilter.stories.tsx

```typescript
// Story: Empty (nothing selected)
// - Severity options: DEBUG, INFO, WARN, ERROR
// - No chips selected

// Story: MultipleSelected
// - WARN and ERROR selected
// - Show as colored chips with delete icons

// Story: AllSelected
// - All options selected

// Story: WithColors
// - Each option has a mapped color (DEBUG=grey, INFO=blue, WARN=orange, ERROR=red)
```

### 7.3 TimeRangeSelector.stories.tsx

```typescript
// Story: LiveMode
// - "Live" chip is filled/highlighted, all others outlined

// Story: RelativeMode
// - "Last 6h" is highlighted

// Story: CustomMode
// - "Custom" is highlighted, date range picker is visible
// - Pre-fill with today 00:00 to now
```

### 7.4 OpTable v2.stories.tsx

```typescript
// Story: Default
// - 10 ops with mixed statuses
// - No row selected

// Story: WithSelectedOp
// - 3rd row highlighted as selected

// Story: FailedOpsOnly
// - Client-side filter set to "failed"
// - Only show 3 failed ops

// Story: WithDuration
// - Show the new Duration column
// - Running op shows "2m 14s (running...)"
// - Succeeded op shows "342ms"
// - Failed op shows "1m 02s"
// - Pending op shows "—"
```

### 7.5 AlertBanner.stories.tsx

```typescript
// Story: ErrorAlert
// - "7 ops failed in the last hour" with [View Failed Ops] action button

// Story: WarningAlert
// - "Queue site:hn:http at 95% capacity" with [View Queue] action

// Story: InfoAlert
// - "Engine restarted successfully" with dismiss

// Story: Dismissed
// - Show empty space where banner was (test dismissal animation)
```

### 7.6 WorkflowSummaryCard.stories.tsx

```typescript
// Story: Running
// - Name, site, status=running, duration ticking, progress 60%

// Story: Failed
// - Status=failed, error message visible, progress 40%

// Story: Succeeded
// - Status=succeeded, full progress bar, duration finalized

// Story: Canceled
// - Status=canceled, partial progress
```

### 7.7 Existing stories that need updates

| Component | Current Story | What to Add |
|-----------|--------------|-------------|
| `StatCard.stories.tsx` | Basic loading/loaded | Add `onClick` story to show clickable behavior |
| `OpStatusBreakdown.stories.tsx` | Static chart | Add `onSegmentClick` story |
| `QueueHealthPreview.stories.tsx` | Static table | Add `onRowClick` story |
| `QueueStatusTable.stories.tsx` | Basic table | Add inline expansion story |
| `RuntimeEventList.stories.tsx` | List view | **DELETE** — replaced by RuntimeEventTable stories |

---

## 8. Implementation Plan

This plan is ordered by dependency — each phase builds on the previous one. Estimated time assumes a new intern working full-time.

### Phase 0: Foundation (Day 1-2)

**Goal:** Set up cross-cutting infrastructure that all other phases depend on.

#### Step 0.1: Error Boundary

- **File:** `web/src/components/common/AppErrorBoundary.tsx`
- **What:** Wrap `AppShell` children in a React error boundary
- **How:**
  ```
  pseudocode:
  1. Create AppErrorBoundary component using react-error-boundary or class component
  2. Fallback UI: "Something went wrong" card with error message + stack trace (dev only) + retry button
  3. In App.tsx, wrap <AppShell>{children}</AppShell> with <AppErrorBoundary>
  4. Test: throw an error in EngineOverviewPage, verify fallback shows
  ```
- **Files to touch:** `App.tsx`, new `AppErrorBoundary.tsx`

#### Step 0.2: Toast Notification System

- **File:** `web/src/components/common/ToastProvider.tsx`
- **What:** Global snackbar provider using MUI Snackbar + a React context
- **How:**
  ```
  pseudocode:
  1. Create ToastContext with { showToast(message, severity) }
  2. Create ToastProvider that wraps children + renders MUI Snackbar
  3. Create useToast() hook for easy consumption
  4. Add ToastProvider to App.tsx above AppShell
  5. Use in SubmitWorkflowPage after successful submission
  6. Use in CancelWorkflowButton after successful cancel
  ```
- **Files to touch:** new `ToastProvider.tsx`, `App.tsx`, `SubmitWorkflowPage.tsx`, `CancelWorkflowButton.tsx`

#### Step 0.3: Breadcrumb Navigation

- **File:** `web/src/components/layout/BreadcrumbNav.tsx`
- **What:** A breadcrumb component below the AppBar that derives crumbs from the current route
- **How:**
  ```
  pseudocode:
  1. Use useLocation() and useNavigate() to derive crumbs
  2. Map route patterns to labels:
     / → Home
     /workflows → Workflows
     /workflows/:id → Workflows > {workflowName}
     /events → Events
     /queues → Queues
     /sites → Sites
     /sites/:name → Sites > {siteName}
     /submit → Submit
  3. Use MUI Breadcrumbs component with Link elements
  4. Add to AppShell below the AppBar
  ```
- **Files to touch:** new `BreadcrumbNav.tsx`, `AppShell.tsx`

### Phase 1: Shared Components (Day 3-5)

**Goal:** Build the reusable components that all page redesigns will use.

#### Step 1.1: MultiSelectChipFilter

- **File:** `web/src/components/common/MultiSelectChipFilter.tsx`
- **What:** Reusable multi-select filter with colored chips
- **Dependencies:** None
- **How:** See component DSL in section 6.2
- **Stories:** See section 7.2

#### Step 1.2: TimeRangeSelector

- **File:** `web/src/components/common/TimeRangeSelector.tsx`
- **What:** Reusable time range picker with live/relative/absolute modes
- **Dependencies:** MUI DateRangePicker (from `@mui/x-date-pickers`)
- **How:** See component DSL in section 6.3
- **Note:** This requires adding `@mui/x-date-pickers` and a date library (`dayjs`) as dependencies
- **Stories:** See section 7.3

#### Step 1.3: AlertBanner

- **File:** `web/src/components/common/AlertBanner.tsx`
- **What:** Dismissible alert banner with optional action button
- **Dependencies:** MUI Alert
- **How:** See component DSL in section 6.6
- **Stories:** See section 7.5

### Phase 2: RuntimeEventTable (Day 5-8)

**Goal:** Build the core event table that replaces RuntimeEventList in all three locations.

#### Step 2.1: Create RuntimeEventTable

- **File:** `web/src/components/workflows/RuntimeEventTable.tsx`
- **What:** Dense expandable data table for runtime events
- **Dependencies:** Phase 1 components (MultiSelectChipFilter, TimeRangeSelector)
- **How:**
  ```
  pseudocode:
  1. Use MUI Table with size="small"
  2. Columns: timestamp, severity (colored dot), source, kind, message (truncated)
  3. Each row is expandable — click toggles RuntimeEventDetailRow
  4. RuntimeEventDetailRow shows: full message, op ID (clickable), workflow ID (clickable),
     site, worker ID, full payload as JSON viewer
  5. Accept optional showFilters prop — if true, render severity + source MultiSelectChipFilters
  6. Accept optional showTimeRange prop — if true, render TimeRangeSelector
  7. Accept optional showPagination prop — if true, render TablePagination
  8. Sorting: store sortField + sortDir in local state, sort events client-side
  ```
- **Stories:** See section 7.1

#### Step 2.2: Update runtimeEventFeed hook

- **File:** `web/src/features/runtime-events/runtimeEventFeed.ts`
- **What:** Add time range support, pagination support
- **Changes:**
  ```
  pseudocode:
  1. Add to RuntimeEventsParams:
     - since?: string (ISO timestamp)
     - until?: string (ISO timestamp)
     - offset?: number
  2. buildRuntimeEventSearchParams: include since, until, offset in URL params
  3. Add pause/resume support:
     - Return pause() and resume() functions
     - pause() closes the EventSource without clearing events
     - resume() reopens it
  4. Add total count from API response header or field
  ```

#### Step 2.3: Replace RuntimeEventList usages

- **File:** Three files to update
- **Changes:**
  ```
  1. RuntimeEventsPage.tsx:
     - Remove old filter TextFields
     - Add TimeRangeSelector + MultiSelectChipFilter for severity/source
     - Replace <RuntimeEventList> with <RuntimeEventTable showFilters showTimeRange showPagination>
  
  2. WorkflowDetailPage.tsx:
     - Move events into a Tab (not standalone card)
     - Replace <RuntimeEventList> with <RuntimeEventTable showFilters showTimeRange>
  
  3. OpDetailDrawer.tsx:
     - In "Runtime" tab, replace <RuntimeEventList> with <RuntimeEventTable> (no filters, dense)
  ```

#### Step 2.4: Backend API changes needed

- **What:** The runtime events API needs time range and pagination support
- **Changes:**
  ```
  GET /api/v1/runtime-events?since=2026-04-07T10:00:00Z&until=2026-04-07T11:00:00Z&limit=50&offset=0
  Response should include:
  {
    "events": [...],
    "total": 247
  }
  
  The SSE stream endpoint should also support since parameter to backfill:
  GET /api/v1/runtime-events/stream?since=2026-04-07T10:00:00Z
  ```

### Phase 3: Workflow Detail Page Redesign (Day 8-11)

**Goal:** Transform the workflow detail page from a vertical scroll into a tabbed, dense layout.

#### Step 3.1: Rewrite WorkflowDetailPage with tabs

- **File:** `web/src/pages/WorkflowDetailPage.tsx`
- **What:** Convert from vertical stack of Cards to tabbed layout
- **How:**
  ```
  pseudocode:
  1. Replace vertical Card stack with MUI Tabs
  2. Tab 1: "Operations (N)" — OpTable with filters + sort
  3. Tab 2: "Runtime Events (N)" — RuntimeEventTable (reused from Phase 2)
  4. Tab 3: "Artifacts (N)" — new WorkflowArtifactTable
  5. Tab 4: "JSON" — raw workflow JSON viewer
  6. Move OpDetailDrawer to be triggered from any tab (it's already a drawer)
  7. Add breadcrumb at top
  8. Add copy-ID button to header
  ```

#### Step 3.2: Upgrade OpTable

- **File:** `web/src/components/workflows/OpTable.tsx`
- **What:** Add duration column, sort, filter, search
- **How:**
  ```
  pseudocode:
  1. Add duration column: compute from op timing data
     - Running: elapsed time from createdAt to now, with pulsing indicator
     - Completed: completedAt - createdAt
     - Pending: "—"
  2. Make column headers clickable for sorting
  3. Add status filter dropdown above table
  4. Add search text field above table
  5. Client-side filter + sort (ops array is small enough)
  ```

#### Step 3.3: Create WorkflowArtifactTable

- **File:** `web/src/components/workflows/WorkflowArtifactTable.tsx`
- **What:** Table of all artifacts across all ops in a workflow
- **How:**
  ```
  pseudocode:
  1. Fetch artifacts for each op (or add a batch endpoint)
  2. Columns: Name, Op ID, Content Type, Size, Actions (download, preview)
  3. Click "preview" opens ArtifactPreview inline or in drawer
  4. Click "download" triggers browser download via /api/v1/artifacts/:id
  ```

### Phase 4: Overview & Queue Pages (Day 11-14)

**Goal:** Make overview page interactive, fix queue page data.

#### Step 4.1: Make StatCard clickable

- **File:** `web/src/components/overview/StatCard.tsx`
- **What:** Add onClick handler to navigate to filtered views
- **How:**
  ```
  pseudocode:
  1. Add optional onClick and href props to StatCard
  2. Wrap Card in Link or add onClick to CardActionArea
  3. In EngineOverviewPage:
     - "12 Workflows" card → navigate to /workflows
     - "347 Ops" card → navigate to /events?kind=op (or /workflows with ops visible)
     - "8 Queues" card → navigate to /queues
  ```

#### Step 4.2: Add AlertBanner to Overview

- **File:** `web/src/pages/EngineOverviewPage.tsx`
- **What:** Show alert banner when there are failed ops or error events
- **How:**
  ```
  pseudocode:
  1. Read OpCounts.failed from engine status
  2. If > 0, show AlertBanner:
     severity: error
     message: "{N} ops failed"
     action: { label: "View Failed", onClick: () => navigate('/workflows?status=failed') }
  ```

#### Step 4.3: Add Recent Activity feed to Overview

- **File:** `web/src/components/overview/RecentActivityFeed.tsx` (new)
- **What:** Show last 5 runtime events with severity and timestamp
- **How:**
  ```
  pseudocode:
  1. Use useRuntimeEventFeed with limit: 5
  2. Render as a compact list (one line per event)
  3. Each line: timestamp + severity dot + message (truncated to 60 chars)
  4. Clicking a line navigates to /events with that event's filters
  5. "View All Events" link at bottom navigates to /events
  ```

#### Step 4.4: Fix Queue Monitor Page

- **File:** `web/src/pages/QueueMonitorPage.tsx`
- **What:** Fix expansion UX, add time range to throughput, note fake data
- **How:**
  ```
  pseudocode:
  1. Replace Typography click targets with proper TableRow expansion
     - Use MUI Table with Collapse pattern (see MUI docs "Collapsible table")
     - Detail panel appears inline below the row, not at the bottom
  2. Add time range buttons above throughput chart: [15m] [1h] [6h] [24h]
  3. Add a prominent notice if throughput data is placeholder:
     <Alert severity="info">Throughput data is simulated. Awaiting backend metrics endpoint.</Alert>
  4. Create TODO ticket for backend: GET /api/v1/queues/:key/metrics?range=15m
  ```

### Phase 5: Sites & Submit Pages (Day 14-16)

#### Step 5.1: Fix Sites N+1 API problem

- **File:** `web/src/pages/SitesListPage.tsx`
- **What:** Eliminate per-card detail API calls
- **How:**
  ```
  pseudocode:
  Option A (preferred): Add a list endpoint that returns summary data
    GET /api/v1/catalog/sites returns:
    [{ name, verbCount, scriptCount, hasSubmitVerbs, databaseFileName }]
    
  Option B: Use the existing list data and render cards without the extra detail.
    The SiteCard currently needs detail data for queue policies.
    Move queue policies to the detail page only.
  ```

#### Step 5.2: Add submission feedback

- **File:** `web/src/pages/SubmitWorkflowPage.tsx`
- **What:** Show toast + link to created workflow after submission
- **How:**
  ```
  pseudocode:
  1. After successful mutation:
     const result = await submitMutation(params).unwrap()
     showToast(`Workflow submitted: ${result.workflowId}`, 'success')
     navigate(`/workflows/${result.workflowId}`)
  ```

### Phase 6: Polish (Day 16-18)

- Add dark mode toggle to AppBar
- Add keyboard shortcuts (r=refresh, e=events, w=workflows, Escape=close drawer)
- Accessibility audit: aria-labels on all interactive elements, focus management
- Remove `RuntimeEventList.tsx` (fully replaced by `RuntimeEventTable.tsx`)
- Update all Storybook stories to reflect new components

---

## 9. Open Questions

| # | Question | Impact | Decision Needed By |
|---|----------|--------|-------------------|
| 1 | Does the backend support time-range filtering for runtime events? | Phase 2 depends on this. If not, implement client-side time filtering as a fallback. | Before Phase 2 |
| 2 | Is there a backend endpoint for queue throughput metrics? | QueueMonitorPage throughput chart is currently fake data. If no endpoint exists, we need to either build one or remove the chart. | Before Phase 4 |
| 3 | Should the API return `total` count for pagination? | Runtime events pagination needs to know total count. If the API doesn't return it, we can use "load more" pattern instead. | Before Phase 2 |
| 4 | What is the expected scale of events? 100s? 1000s? 100,000s? | Determines whether we need virtualization (react-window) for the events table. | Before Phase 2 |
| 5 | Is there a way to get all artifacts for a workflow without fetching per-op? | WorkflowArtifactTable needs all artifacts. If no batch endpoint, we need N API calls. | Before Phase 3 |
| 6 | Should the site list be hardcoded or fetched from API? | WorkflowsPage hardcodes `['hackernews', 'slashdot', 'js-demo', 'nereval']`. Should be dynamic. | Before Phase 5 |

---

## 10. References

### Source Files (key files to read)

| File | Purpose | Lines |
|------|---------|-------|
| `web/src/components/layout/AppShell.tsx` | Root layout + navigation | 45 |
| `web/src/pages/RuntimeEventsPage.tsx` | Global events page | 110 |
| `web/src/pages/WorkflowDetailPage.tsx` | Single workflow detail | 145 |
| `web/src/pages/WorkflowsPage.tsx` | Workflow list | 55 |
| `web/src/pages/QueueMonitorPage.tsx` | Queue status + throughput | 85 |
| `web/src/pages/EngineOverviewPage.tsx` | Dashboard overview | 35 |
| `web/src/pages/SitesListPage.tsx` | Site listing | 55 |
| `web/src/pages/SiteDetailPage.tsx` | Single site detail | 155 |
| `web/src/pages/SubmitWorkflowPage.tsx` | Submit workflow form | 180 |
| `web/src/components/workflows/RuntimeEventList.tsx` | Event list (TO BE REPLACED) | 175 |
| `web/src/components/workflows/OpTable.tsx` | Ops table | 90 |
| `web/src/components/workflows/OpDetailDrawer.tsx` | Op detail drawer with 7 tabs | 300 |
| `web/src/components/workflows/WorkflowTable.tsx` | Workflow list table | 100 |
| `web/src/components/workflows/WorkflowFilters.tsx` | Site/status/search filters | 65 |
| `web/src/components/queues/QueueStatusTable.tsx` | Queue metrics table | 110 |
| `web/src/components/queues/QueueDetailPanel.tsx` | Expanded queue detail | 55 |
| `web/src/components/overview/StatCard.tsx` | Stat card widget | 50 |
| `web/src/features/runtime-events/runtimeEventFeed.ts` | SSE + history merge hook | 140 |
| `web/src/api/runtimeEventsApi.ts` | RTK Query API for events | 50 |
| `web/src/api/workflowApi.ts` | RTK Query API for workflows | 120 |
| `web/src/api/types.ts` | TypeScript type definitions | 120 |

### External References

- **MUI Data Table pattern:** https://mui.com/material-ui/react-table/#collapsible-table
- **MUI Autocomplete (multi-select):** https://mui.com/material-ui/react-autocomplete/#multiple-values
- **MUI Date Range Picker:** https://mui.com/x/react-date-pickers/date-range-picker/
- **React Error Boundaries:** https://react.dev/reference/react/Component#catching-rendering-errors-with-an-error-boundary
- **RTK Query:** https://redux-toolkit.js.org/rtk-query/overview
- **EventSource (SSE) API:** https://developer.mozilla.org/en-US/docs/Web/API/EventSource

### API Endpoints Summary

```
GET  /api/v1/engine/status              → Engine status with OpCounts
GET  /api/v1/workflows?site=&status=&limit=50  → List workflows
GET  /api/v1/workflows/:id              → Single workflow detail
GET  /api/v1/workflows/:id/ops          → Ops for a workflow
GET  /api/v1/workflows/:id/ops/:opId/result    → Op result
GET  /api/v1/workflows/:id/ops/:opId/artifacts  → Op artifacts
POST /api/v1/workflows/:id/ops/:opId/retry      → Retry an op
POST /api/v1/workflows/:id/cancel               → Cancel a workflow
GET  /api/v1/queues                      → List all queues with status
GET  /api/v1/catalog/sites               → List site summaries
GET  /api/v1/catalog/sites/:name         → Site detail (verbs, scripts, policies)
GET  /api/v1/catalog/sites/:name/verbs   → List verbs for a site
GET  /api/v1/catalog/sites/:name/scripts?path=  → Get script source
POST /api/v1/submissions                 → Submit a new workflow
GET  /api/v1/runtime-events?workflowId=&opId=&site=&workerId=&limit=100  → Event history
GET  /api/v1/runtime-events/stream?...   → SSE stream of runtime events
GET  /api/v1/artifacts/:id               → Download artifact body
```
