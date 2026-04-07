---
Title: Dashboard Architecture and Component Design
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
      Note: Backend service for engine status, workflows, ops
    - Path: pkg/services/catalog/service.go
      Note: Backend service for site/verb discovery
    - Path: pkg/services/submission/service.go
      Note: Backend service for workflow submission
    - Path: pkg/engine/model/types.go
      Note: Core domain types (WorkflowRun, OpSpec, OpResult, etc.)
    - Path: pkg/engine/store/sqlite/status.go
      Note: EngineStatus inspection type
    - Path: pkg/engine/scheduler/scheduler.go
      Note: Scheduler events and CycleResult types
ExternalSources: []
Summary: "Full dashboard design with ASCII mockups, YAML widget skeletons, Redux state, RTK Query API, and Storybook plan"
LastUpdated: 2026-03-23T21:53:18.656992062-04:00
WhatFor: "Guide implementation of the scraper dashboard frontend"
WhenToUse: "When building or extending the dashboard UI"
---

# Dashboard Architecture and Component Design

## Executive Summary

A React + Material UI dashboard for the scraper engine providing four main capabilities: workflow monitoring, engine health, queue/rate-limiting visibility, and workflow submission. The frontend uses Redux Toolkit with RTK Query for server state, communicates with the existing Go backend via a JSON REST API (endpoints already designed in SCRAPER-HTTP-API), and is decomposed into reusable widgets with full Storybook coverage.

## Problem Statement

Operators currently interact with the scraper engine exclusively through CLI commands (`scraper engine status`, `scraper worker run`, etc.). This works for single-workflow debugging but fails for:

- Monitoring multiple concurrent workflows across sites
- Understanding queue backpressure and rate-limiting behavior in real time
- Quickly submitting test workflows without remembering verb flags
- Correlating op failures with their dependency chains

A visual dashboard makes these workflows accessible without memorizing CLI flags.

## Technology Stack

| Layer | Choice | Rationale |
|-------|--------|-----------|
| UI framework | React 18+ | Team standard |
| Component library | Material UI v5 | Rich data-display components (DataGrid, Chip, LinearProgress) |
| State management | Redux Toolkit | Predictable state, good DevTools |
| Server state | RTK Query | Cache invalidation, polling, optimistic updates |
| Routing | React Router v6 | Standard, lazy-loadable routes |
| Charts | Recharts | Lightweight, composable, MUI-compatible |
| Stories | Storybook 8 | Visual testing, component catalog |
| Build | Vite | Fast HMR, Go embed-friendly output |

---

## Screen Layouts (ASCII Mockups)

### 1. Engine Overview (default landing page)

```
+------------------------------------------------------------------+
| SCRAPER ENGINE                  [Overview] [Workflows] [Queues]  |
|                                 [Submit]                         |
+------------------------------------------------------------------+
|                                                                  |
|  +------------+  +------------+  +------------+  +------------+  |
|  | Workflows  |  | Operations |  | Leases     |  | Artifacts  |  |
|  |     12     |  |    847     |  | Active: 4  |  |   1,234    |  |
|  | running: 3 |  | ready: 23  |  | Expired: 0 |  |            |  |
|  | ok: 8      |  | run: 4     |  |            |  |            |  |
|  | fail: 1    |  | ok: 808    |  |            |  |            |  |
|  +------------+  | fail: 12   |  +------------+  +------------+  |
|                  +------------+                                   |
|                                                                  |
|  +-------------------------------+  +---------------------------+|
|  | OP STATUS BREAKDOWN           |  | MIGRATIONS                ||
|  |                               |  |                           ||
|  | pending  [====          ] 23  |  | 001_core      applied     ||
|  | ready    [==            ] 12  |  | 002_runtime   applied     ||
|  | running  [=             ]  4  |  |                           ||
|  | succeed  [==============]808  |  | Engine DB: up to date     ||
|  | failed   [==            ] 12  |  +---------------------------+|
|  +-------------------------------+                               |
|                                                                  |
|  +-------------------------------+  +---------------------------+|
|  | RECENT EVENTS                 |  | QUEUE HEALTH              ||
|  |                               |  |                           ||
|  | 14:32:05 op_succeeded         |  | site:hn:http   2/4  ████ ||
|  |   hn-001:frontpage-extract    |  | site:hn:js     1/4  ██   ||
|  | 14:32:04 op_leased            |  | site:sd:http   0/4  ░░   ||
|  |   hn-001:page2-fetch          |  | site:nv:js     0/1  ░░   ||
|  | 14:31:58 workflow_created     |  |                           ||
|  |   hn-001                      |  | [View All Queues ->]      ||
|  | 14:31:45 op_failed            |  +---------------------------+|
|  |   sd-003:extract (retryable)  |                               |
|  | 14:31:30 queue_rate_limited   |                               |
|  |   site:hn:http                |                               |
|  +-------------------------------+                               |
+------------------------------------------------------------------+
```

### 2. Workflows List

```
+------------------------------------------------------------------+
| SCRAPER ENGINE                  [Overview] [Workflows] [Queues]  |
|                                 [Submit]                         |
+------------------------------------------------------------------+
|                                                                  |
|  Workflows  [Site: All v] [Status: All v]  [Search: ________]   |
|                                                                  |
|  +--------------------------------------------------------------+|
|  | ID          | Site       | Name          | Status  | Ops     ||
|  |-------------|------------|---------------|---------|---------|+|
|  | hn-001      | hackernews | seed workflow | RUNNING | 12/47   ||
|  | sd-003      | slashdot   | seed workflow | RUNNING |  8/24   ||
|  | demo-1      | js-demo    | seed workflow | OK      |  7/7    ||
|  | nv-fixture  | nereval    | scrape wf     | FAILED  | 31/45   ||
|  | hn-prev     | hackernews | seed workflow | OK      | 23/23   ||
|  +--------------------------------------------------------------+|
|                                                                  |
|  Showing 5 of 12 workflows                    [< 1 2 3 >]       |
+------------------------------------------------------------------+
```

### 3. Workflow Detail

```
+------------------------------------------------------------------+
| < Back to Workflows    Workflow: hn-001                          |
+------------------------------------------------------------------+
|                                                                  |
|  +-------------------+  +--------------------------------------+ |
|  | Status: RUNNING   |  | Op Progress                          | |
|  | Site: hackernews  |  | [████████████░░░░░░░░] 12/47         | |
|  | Created: 14:31:58 |  |                                      | |
|  | Updated: 14:32:05 |  | pending:  3  ready: 8  running: 2    | |
|  +-------------------+  | ok: 12  failed: 0                    | |
|                          +--------------------------------------+ |
|                                                                  |
|  OP GRAPH (DAG)                                                  |
|  +--------------------------------------------------------------+|
|  |                                                              ||
|  |  [seed]-->[fetch-p1]-->[extract-p1]-->[fetch-p2]            ||
|  |    ok        ok            ok      \     running            ||
|  |                                     \                       ||
|  |                                      +->[detail-1] ok      ||
|  |                                      +->[detail-2] ok      ||
|  |                                      +->[detail-3] running ||
|  |                                                              ||
|  +--------------------------------------------------------------+|
|                                                                  |
|  OPS TABLE                                                       |
|  +--------------------------------------------------------------+|
|  | ID                  | Kind      | Queue         | Status     ||
|  |---------------------|-----------|---------------|------------|+|
|  | hn-001:seed         | js        | site:hn:js    | succeeded  ||
|  | hn-001:fetch-p1     | http/fetch| site:hn:http  | succeeded  ||
|  | hn-001:extract-p1   | js        | site:hn:js    | succeeded  ||
|  | hn-001:fetch-p2     | http/fetch| site:hn:http  | running    ||
|  | hn-001:detail-1     | js        | site:hn:js    | succeeded  ||
|  +--------------------------------------------------------------+|
|                                                                  |
|  [Click any op to see detail drawer ->]                          |
+------------------------------------------------------------------+
```

### 4. Op Detail Drawer (slides in from right)

```
                                    +------------------------------+
                                    | Op: hn-001:extract-p1        |
                                    | Kind: js  Script: extract.js |
                                    | Queue: site:hn:js            |
                                    | Status: succeeded            |
                                    |------------------------------|
                                    | INPUT                        |
                                    | {                            |
                                    |   "baseURL": "https://...",  |
                                    |   "fetchedOpID": "...:fetch",|
                                    |   "pageNumber": 1,           |
                                    |   "maxPages": 2              |
                                    | }                            |
                                    |------------------------------|
                                    | DEPENDENCIES                 |
                                    | hn-001:fetch-p1 (required) ok|
                                    |------------------------------|
                                    | RESULT                       |
                                    | data: { stories: 30 }       |
                                    | artifacts: 0                 |
                                    | emitted: 4 child ops        |
                                    |------------------------------|
                                    | RETRY                        |
                                    | attempts: 0/3                |
                                    | backoff: exponential 1s-30s  |
                                    |------------------------------|
                                    | LEASE                        |
                                    | worker: hn-cli               |
                                    | acquired: 14:32:01           |
                                    | expires: 14:32:31            |
                                    +------------------------------+
```

### 5. Queue Monitor

```
+------------------------------------------------------------------+
| SCRAPER ENGINE                  [Overview] [Workflows] [Queues]  |
|                                 [Submit]                         |
+------------------------------------------------------------------+
|                                                                  |
|  Queue Monitor                                                   |
|                                                                  |
|  +--------------------------------------------------------------+|
|  | Queue Key       | In-Flight | Max | Tokens | Rate     | Burst||
|  |-----------------|-----------|-----|--------|----------|------||
|  | site:hn:http    |    2      |  4  |  1.8   | 2/sec    |   4  ||
|  | site:hn:js      |    1      |  4  |  4.0   | (none)   |   -  ||
|  | site:sd:http    |    0      |  4  |  4.0   | 2/sec    |   4  ||
|  | site:sd:js      |    0      |  4  |  4.0   | (none)   |   -  ||
|  | site:nv:js      |    0      |  1  |  1.0   | (none)   |   -  ||
|  | site:nv:http    |    3      |  4  |  0.2   | 1/sec    |   2  ||
|  +--------------------------------------------------------------+|
|                                                                  |
|  +--------------------------------------------------------------+|
|  | THROUGHPUT (ops/min)              last 15 minutes             ||
|  |                                                              ||
|  |  8 |         *                                               ||
|  |  6 |    *   * *      *                                       ||
|  |  4 |   * * *   *    * *    *                                 ||
|  |  2 |  *   *     *  *   *  * *                                ||
|  |  0 +--+----+----+----+----+----+                             ||
|  |    14:18 14:21 14:24 14:27 14:30 14:33                       ||
|  |                                                              ||
|  |  --- site:hn:http  --- site:hn:js  --- site:nv:http          ||
|  +--------------------------------------------------------------+|
|                                                                  |
|  +--------------------------------------------------------------+|
|  | RATE LIMIT EVENTS (recent)                                   ||
|  |                                                              ||
|  | 14:31:30  site:hn:http   tokens exhausted (0.2 remaining)   ||
|  | 14:29:15  site:nv:http   tokens exhausted (0.0 remaining)   ||
|  | 14:27:42  site:hn:http   tokens exhausted (0.1 remaining)   ||
|  +--------------------------------------------------------------+|
+------------------------------------------------------------------+
```

### 6. Submit Workflow

```
+------------------------------------------------------------------+
| SCRAPER ENGINE                  [Overview] [Workflows] [Queues]  |
|                                 [Submit]                         |
+------------------------------------------------------------------+
|                                                                  |
|  Submit Workflow                                                 |
|                                                                  |
|  +---------------------------+  +-------------------------------+|
|  | SITE                      |  | VERB                          ||
|  |                           |  |                               ||
|  | ( ) js-demo               |  | ( ) seed                      ||
|  | (*) hackernews            |  | (*) seed                      ||
|  | ( ) slashdot              |  | ( ) extract-frontpage          ||
|  | ( ) nereval               |  |                               ||
|  +---------------------------+  +-------------------------------+|
|                                                                  |
|  +--------------------------------------------------------------+|
|  | PARAMETERS                 hackernews > seed                  ||
|  |                                                              ||
|  |  Workflow ID:  [hn-manual-001_____________]                  ||
|  |  base-url:     [https://news.ycombinator.com/_]              ||
|  |  max-pages:    [2__]                                         ||
|  |                                                              ||
|  |  "Submit the Hacker News seed workflow starting at seed.js.  ||
|  |   This command only submits the initial durable work."       ||
|  |                                                              ||
|  |           [Cancel]  [Submit Workflow]                         ||
|  +--------------------------------------------------------------+|
|                                                                  |
|  +--------------------------------------------------------------+|
|  | RECENT SUBMISSIONS                                           ||
|  |                                                              ||
|  | 14:31:58  hackernews/seed  hn-001     -> RUNNING             ||
|  | 14:25:12  js-demo/seed     demo-1     -> SUCCEEDED           ||
|  | 14:20:03  slashdot/seed    sd-003     -> RUNNING             ||
|  +--------------------------------------------------------------+|
+------------------------------------------------------------------+
```

---

## Widget & Component Skeleton (YAML)

```yaml
# ============================================================
# LAYOUT SHELL
# ============================================================

AppShell:
  description: "Top-level layout: AppBar + sidebar/tabs + content area"
  library: "@mui/material"
  components:
    - AppBar
    - Tabs (Overview | Workflows | Queues | Submit)
    - Box (content area with React Router Outlet)
  props:
    currentTab: string
    onTabChange: (tab: string) => void
  stories:
    - default: "All four tabs visible, Overview selected"
    - workflows-active: "Workflows tab selected with breadcrumb"

# ============================================================
# PAGE: ENGINE OVERVIEW
# ============================================================

EngineOverviewPage:
  description: "Landing page with health summary cards, event feed, queue preview"
  layout: "CSS Grid: 4-col stat row, 2-col bottom (events + queues)"
  children:
    - StatCardRow
    - OpStatusBreakdown
    - MigrationStatusCard
    - RecentEventsTimeline
    - QueueHealthPreview

StatCard:
  description: "Single KPI card with title, big number, optional breakdown chips"
  library: "@mui/material Card"
  props:
    title: string
    value: number
    breakdown: { label: string; value: number; color: string }[]
  variants:
    - workflows: "title=Workflows, breakdown by status"
    - operations: "title=Operations, breakdown by OpStatus"
    - leases: "title=Leases, active + expired"
    - artifacts: "title=Artifacts, single count"
  stories:
    - default: "Workflows card with running/succeeded/failed"
    - zero-state: "All zeros, empty engine"
    - high-counts: "Thousands of ops"

StatCardRow:
  description: "Horizontal row of 4 StatCards"
  layout: "Grid row, responsive: 4col -> 2col -> 1col"
  children: [StatCard, StatCard, StatCard, StatCard]
  stories:
    - default: "4 cards with realistic data"
    - loading: "All cards in skeleton state"

OpStatusBreakdown:
  description: "Horizontal stacked bar or segmented progress showing op status distribution"
  library: "@mui/material LinearProgress + custom segments"
  props:
    counts: Record<OpStatus, number>
  stories:
    - default: "Mix of all statuses"
    - all-succeeded: "100% green"
    - mostly-pending: "Large pending, small running"
    - has-failures: "Red segment visible"

MigrationStatusCard:
  description: "Shows engine DB migration version and status"
  library: "@mui/material Card + List"
  props:
    migrations: MigrationStatus[]
    upToDate: boolean
  stories:
    - up-to-date: "All migrations applied, green check"
    - pending: "One migration not yet applied, amber warning"

RecentEventsTimeline:
  description: "Scrollable timeline of scheduler events"
  library: "@mui/material List + ListItem"
  props:
    events: SchedulerEvent[]
    maxItems: number (default 20)
  features:
    - Color-coded icons per event kind
    - Clickable workflow/op IDs navigate to detail
    - Auto-refresh via polling
  stories:
    - default: "Mix of event types"
    - empty: "No events yet"
    - rate-limited: "Several queue_rate_limited events"
    - failures: "op_failed events with retryable flag"

QueueHealthPreview:
  description: "Compact queue status showing in-flight vs max for top queues"
  library: "@mui/material Card + LinearProgress"
  props:
    queues: QueueSnapshot[]
    maxVisible: number (default 6)
  stories:
    - default: "4 queues, mixed utilization"
    - saturated: "All queues at max in-flight"
    - idle: "All queues at zero"

# ============================================================
# PAGE: WORKFLOWS
# ============================================================

WorkflowsPage:
  description: "Filterable workflow list with navigation to detail"
  children:
    - WorkflowFilters
    - WorkflowTable

WorkflowFilters:
  description: "Filter bar: site selector, status selector, search"
  library: "@mui/material Select + TextField"
  props:
    sites: SiteName[]
    selectedSite: string | null
    selectedStatus: WorkflowStatus | null
    searchQuery: string
    onChange: (filters) => void
  stories:
    - default: "All filters empty"
    - filtered: "Site=hackernews, Status=running"

WorkflowTable:
  description: "Sortable paginated table of workflows"
  library: "@mui/material DataGrid or Table"
  columns:
    - id: "Workflow ID (link to detail)"
    - site: "Site name (Chip)"
    - name: "Workflow name"
    - status: "Status (colored Chip)"
    - progress: "Op progress bar (succeeded/total)"
    - createdAt: "Relative timestamp"
  props:
    workflows: WorkflowSummary[]
    loading: boolean
    page: number
    pageSize: number
  stories:
    - default: "10 workflows, mixed statuses"
    - empty: "No workflows found"
    - loading: "Skeleton rows"
    - single-site: "All hackernews workflows"

WorkflowStatusChip:
  description: "Colored chip for workflow status"
  library: "@mui/material Chip"
  props:
    status: WorkflowStatus
  color-map:
    pending: "default"
    running: "info"
    succeeded: "success"
    failed: "error"
    canceled: "warning"
  stories:
    - all-statuses: "One chip per status in a row"

# ============================================================
# PAGE: WORKFLOW DETAIL
# ============================================================

WorkflowDetailPage:
  description: "Single workflow view with header, op graph, op table, op drawer"
  children:
    - WorkflowHeader
    - WorkflowProgressBar
    - OpDagVisualization
    - OpTable
    - OpDetailDrawer

WorkflowHeader:
  description: "Workflow ID, site, status, timestamps"
  library: "@mui/material Card + Typography"
  props:
    workflow: WorkflowRun
  stories:
    - running: "Running workflow"
    - succeeded: "Completed workflow"
    - failed: "Failed workflow with error"

WorkflowProgressBar:
  description: "Multi-segment progress bar showing op status counts"
  library: "@mui/material Box + custom segments"
  props:
    stats: WorkflowStats
  stories:
    - in-progress: "Mixed pending/ready/running/succeeded"
    - complete: "All succeeded"
    - has-failures: "Some failed ops"

OpDagVisualization:
  description: "Directed acyclic graph of ops showing parent-child and dependency edges"
  library: "Custom SVG or @dagrejs/dagre for layout"
  props:
    ops: WorkflowOp[]
  features:
    - Nodes colored by status
    - Solid edges for parent-child, dashed for dependency
    - Click node to select op (opens drawer)
    - Zoom/pan for large graphs
  stories:
    - simple: "3-node linear chain (seed -> fetch -> extract)"
    - fanout: "seed -> 5 parallel fetches -> 5 extracts"
    - complex: "nereval-style: seed -> list pages -> detail fan-out"
    - single-op: "One seed op only"

OpTable:
  description: "Sortable table of all ops in a workflow"
  library: "@mui/material Table"
  columns:
    - id: "Op ID"
    - kind: "js or http/fetch (icon)"
    - queue: "Queue key"
    - status: "Status chip"
    - retryAttempt: "Attempt N/max"
    - createdAt: "Timestamp"
  props:
    ops: WorkflowOp[]
    selectedOpId: OpID | null
    onSelectOp: (opId: OpID) => void
  stories:
    - default: "12 ops, mixed statuses"
    - with-retries: "Some ops showing attempt 2/3"
    - all-succeeded: "Everything green"

OpDetailDrawer:
  description: "Right-side drawer showing full op detail"
  library: "@mui/material Drawer"
  sections:
    - header: "Op ID, kind, script, queue, status"
    - input: "JSON viewer for op input"
    - dependencies: "List of dependency op IDs with status"
    - result: "data, artifacts count, emitted child count"
    - error: "Error code, message, retryable flag (if failed)"
    - retry: "RetryPolicy + RetryState"
    - lease: "Worker ID, acquired/expires timestamps"
  props:
    op: WorkflowOp
    result: OpResult | null
    open: boolean
    onClose: () => void
  stories:
    - js-op-succeeded: "JS op with result data and emitted children"
    - http-op-succeeded: "HTTP fetch op with artifact"
    - op-failed-retryable: "Failed op with retry state"
    - op-running: "Running op with active lease"
    - op-pending: "Pending op, no lease, no result"

JsonViewer:
  description: "Collapsible JSON tree viewer"
  library: "Custom or react-json-view"
  props:
    data: any
    collapsed: number (default 1)
  stories:
    - simple-object: "Flat key-value object"
    - nested: "Deeply nested object"
    - large-array: "Array of 50 items"
    - null-value: "null input"

# ============================================================
# PAGE: QUEUE MONITOR
# ============================================================

QueueMonitorPage:
  description: "Queue status table, throughput chart, rate limit events"
  children:
    - QueueStatusTable
    - ThroughputChart
    - RateLimitEventsLog

QueueStatusTable:
  description: "Table of all queues with current in-flight, policy, token state"
  library: "@mui/material Table"
  columns:
    - queueKey: "Queue key"
    - inFlight: "Current in-flight count"
    - maxInFlight: "MaxInFlight from policy"
    - utilization: "Progress bar (inFlight/max)"
    - tokens: "Current token count (if rate-limited)"
    - ratePerSecond: "Configured rate"
    - burst: "Configured burst"
  props:
    queues: QueueSnapshot[]
  stories:
    - default: "6 queues, mixed utilization"
    - all-idle: "No in-flight ops"
    - saturated: "Multiple queues at max"
    - no-rate-limit: "Queues without rate limiting configured"

TokenBucketGauge:
  description: "Visual gauge for a single queue's token bucket"
  library: "@mui/material CircularProgress or custom SVG"
  props:
    tokens: number
    burst: number
    ratePerSecond: number
  stories:
    - full: "tokens == burst"
    - depleted: "tokens near 0"
    - refilling: "tokens at 50%"

ThroughputChart:
  description: "Line chart showing ops completed per minute over time, per queue"
  library: "Recharts LineChart"
  props:
    series: { queueKey: string; dataPoints: { time: Date; opsPerMinute: number }[] }[]
    timeRange: "5m" | "15m" | "1h"
  stories:
    - default: "3 queues over 15 minutes"
    - single-queue: "One queue, 1 hour"
    - bursty: "Spiky traffic pattern"
    - idle: "Flat zero lines"

RateLimitEventsLog:
  description: "Reverse-chronological list of queue_rate_limited events"
  library: "@mui/material List"
  props:
    events: SchedulerEvent[] (filtered to queue_rate_limited)
  stories:
    - default: "10 recent rate limit events"
    - empty: "No rate limit events"

# ============================================================
# PAGE: SUBMIT WORKFLOW
# ============================================================

SubmitWorkflowPage:
  description: "Site/verb picker, dynamic form, submission result"
  children:
    - SitePicker
    - VerbPicker
    - VerbParameterForm
    - SubmitButton
    - RecentSubmissionsTable

SitePicker:
  description: "Radio group or card list of available sites"
  library: "@mui/material RadioGroup or ToggleButtonGroup"
  props:
    sites: SiteSummary[]
    selected: SiteName | null
    onSelect: (site: SiteName) => void
  stories:
    - default: "4 sites, none selected"
    - selected: "hackernews selected"

VerbPicker:
  description: "Radio group of verbs for the selected site"
  library: "@mui/material RadioGroup"
  props:
    verbs: VerbSummary[]
    selected: string | null
    onSelect: (verb: string) => void
  stories:
    - default: "2 verbs for hackernews"
    - single-verb: "js-demo with only seed"
    - loading: "Skeleton while fetching"

VerbParameterForm:
  description: "Dynamic form generated from verb field metadata"
  library: "@mui/material TextField, Select, Switch"
  props:
    verb: VerbSummary
    values: Record<string, any>
    onChange: (field: string, value: any) => void
  features:
    - Auto-generates fields from VerbSummary.sections[].fields[]
    - Respects field types (string -> TextField, int -> NumberField, bool -> Switch)
    - Shows defaults, help text, required markers
  stories:
    - hackernews-seed: "base-url and max-pages fields"
    - jsdemo-seed: "count, multiplier, prefix fields"
    - nereval-seed: "town, base_url, max_pages fields"
    - empty: "No fields (verb with no parameters)"

SubmitButton:
  description: "Submit button with loading state and result display"
  library: "@mui/material LoadingButton"
  props:
    loading: boolean
    disabled: boolean
    onSubmit: () => void
  stories:
    - default: "Ready to submit"
    - loading: "Submission in progress"
    - disabled: "No verb selected"

RecentSubmissionsTable:
  description: "Table of recently submitted workflows from this session"
  library: "@mui/material Table"
  columns:
    - timestamp: "When submitted"
    - site: "Site name"
    - verb: "Verb name"
    - workflowId: "Workflow ID (link to detail)"
    - status: "Current status (auto-refreshes)"
  props:
    submissions: RecentSubmission[]
  stories:
    - default: "3 recent submissions"
    - empty: "No submissions yet"
```

---

## Redux State Shape

```yaml
ReduxState:
  engine:
    description: "RTK Query cache for engine status (auto-polled)"
    source: "engineApi.useGetEngineStatusQuery()"

  workflows:
    description: "RTK Query cache for workflow list and detail"
    source: "workflowApi endpoints"

  queues:
    description: "RTK Query cache for queue snapshots"
    source: "queueApi endpoints"

  catalog:
    description: "RTK Query cache for sites and verbs"
    source: "catalogApi endpoints"

  ui:
    description: "Local UI state not tied to server"
    shape:
      currentTab: "'overview' | 'workflows' | 'queues' | 'submit'"
      workflowFilters:
        site: "string | null"
        status: "WorkflowStatus | null"
        search: "string"
      selectedOpId: "OpID | null"
      opDrawerOpen: "boolean"
      submitForm:
        selectedSite: "SiteName | null"
        selectedVerb: "string | null"
        fieldValues: "Record<string, any>"
      recentSubmissions: "RecentSubmission[]"
```

## RTK Query API Definition

```yaml
engineApi:
  baseUrl: "/api/v1"
  tagTypes: ["EngineStatus"]
  endpoints:
    getEngineStatus:
      method: GET
      url: "/engine/status"
      returns: EngineStatus
      pollingInterval: 5000
      providesTags: ["EngineStatus"]

    getEngineEvents:
      method: GET
      url: "/engine/events"
      params: { limit: number; since?: string }
      returns: SchedulerEvent[]
      pollingInterval: 3000

workflowApi:
  baseUrl: "/api/v1"
  tagTypes: ["Workflow", "WorkflowList", "WorkflowOps"]
  endpoints:
    listWorkflows:
      method: GET
      url: "/workflows"
      params: { site?: string; status?: string; page?: number; pageSize?: number }
      returns: { workflows: WorkflowSummary[]; total: number }
      pollingInterval: 5000
      providesTags: ["WorkflowList"]

    getWorkflow:
      method: GET
      url: "/workflows/{workflowId}"
      returns: WorkflowSummary
      pollingInterval: 3000
      providesTags: (result) => [{ type: "Workflow", id: result.workflow.id }]

    getWorkflowOps:
      method: GET
      url: "/workflows/{workflowId}/ops"
      returns: WorkflowOp[]
      pollingInterval: 3000
      providesTags: (_, __, arg) => [{ type: "WorkflowOps", id: arg.workflowId }]

    getOpResult:
      method: GET
      url: "/workflows/{workflowId}/ops/{opId}/result"
      returns: OpResult | null

queueApi:
  baseUrl: "/api/v1"
  tagTypes: ["QueueStatus"]
  endpoints:
    listQueues:
      method: GET
      url: "/queues"
      returns: QueueSnapshot[]
      pollingInterval: 5000
      providesTags: ["QueueStatus"]

    getQueueThroughput:
      method: GET
      url: "/queues/throughput"
      params: { timeRange: "5m" | "15m" | "1h" }
      returns: ThroughputSeries[]
      pollingInterval: 10000

catalogApi:
  baseUrl: "/api/v1"
  tagTypes: ["Sites", "Verbs"]
  endpoints:
    listSites:
      method: GET
      url: "/sites"
      returns: SiteSummary[]
      providesTags: ["Sites"]

    listVerbs:
      method: GET
      url: "/sites/{site}/verbs"
      returns: VerbSummary[]
      providesTags: (_, __, arg) => [{ type: "Verbs", id: arg.site }]

submissionApi:
  baseUrl: "/api/v1"
  endpoints:
    submitWorkflow:
      method: POST
      url: "/sites/{site}/verbs/{verb}:submit"
      body: { workflowId?: string; values: Record<string, any> }
      returns: SubmissionResult
      invalidatesTags: ["WorkflowList", "EngineStatus"]
```

---

## Storybook Strategy

### Organization

```
stories/
  layout/
    AppShell.stories.tsx
  overview/
    StatCard.stories.tsx
    StatCardRow.stories.tsx
    OpStatusBreakdown.stories.tsx
    MigrationStatusCard.stories.tsx
    RecentEventsTimeline.stories.tsx
    QueueHealthPreview.stories.tsx
    EngineOverviewPage.stories.tsx
  workflows/
    WorkflowFilters.stories.tsx
    WorkflowTable.stories.tsx
    WorkflowStatusChip.stories.tsx
    WorkflowsPage.stories.tsx
    WorkflowHeader.stories.tsx
    WorkflowProgressBar.stories.tsx
    OpDagVisualization.stories.tsx
    OpTable.stories.tsx
    OpDetailDrawer.stories.tsx
    JsonViewer.stories.tsx
    WorkflowDetailPage.stories.tsx
  queues/
    QueueStatusTable.stories.tsx
    TokenBucketGauge.stories.tsx
    ThroughputChart.stories.tsx
    RateLimitEventsLog.stories.tsx
    QueueMonitorPage.stories.tsx
  submit/
    SitePicker.stories.tsx
    VerbPicker.stories.tsx
    VerbParameterForm.stories.tsx
    SubmitButton.stories.tsx
    RecentSubmissionsTable.stories.tsx
    SubmitWorkflowPage.stories.tsx
```

### Mock Data Factory

```yaml
MockDataFactory:
  description: "Shared factory for generating realistic test data"
  file: "stories/__fixtures__/factories.ts"
  factories:
    createEngineStatus:
      params: { workflowCount?: number; opDistribution?: Partial<Record<OpStatus, number>> }
      returns: EngineStatus

    createWorkflow:
      params: { site?: SiteName; status?: WorkflowStatus; opCount?: number }
      returns: WorkflowSummary

    createWorkflowOp:
      params: { kind?: string; status?: OpStatus; hasLease?: boolean }
      returns: WorkflowOp

    createVerbSummary:
      params: { site?: SiteName; fieldCount?: number }
      returns: VerbSummary

    createQueueSnapshot:
      params: { inFlight?: number; maxInFlight?: number; hasRateLimit?: boolean }
      returns: QueueSnapshot

    createSchedulerEvent:
      params: { kind?: EventKind }
      returns: SchedulerEvent
```

### Decorator Pattern

```yaml
Decorators:
  withReduxProvider:
    description: "Wraps story in Redux Provider with configurable initial state"
    usage: "All page-level stories"

  withRouterProvider:
    description: "Wraps story in MemoryRouter for link navigation"
    usage: "Stories with clickable navigation"

  withMuiTheme:
    description: "Wraps story in MUI ThemeProvider"
    usage: "Global decorator in .storybook/preview.tsx"

  withMockApi:
    description: "Configures MSW handlers for RTK Query endpoints"
    usage: "Page-level integration stories"
```

---

## Design Decisions

### Polling vs WebSocket

**Decision: Polling with RTK Query.**

RTK Query's `pollingInterval` provides cache-aware polling out of the box. The scraper engine is SQLite-backed with no built-in pub/sub, so a WebSocket would require a separate event bus in the Go server. Polling at 3-5 second intervals is sufficient for workflow monitoring and matches the worker's own poll interval.

### DAG Visualization

**Decision: dagre layout + custom SVG rendering.**

Using `@dagrejs/dagre` for layout computation and rendering with plain SVG/React. This avoids heavy graph libraries (cytoscape, vis.js) while producing clean DAG layouts. Most scraper workflows have 5-50 ops, well within dagre's performance envelope.

### Dynamic Form Generation

**Decision: Generate from VerbSummary.sections[].fields[].**

The catalog service already provides full field metadata (type, help, default, choices, required). The `VerbParameterForm` widget maps these to MUI form controls at render time, requiring no hardcoded knowledge of any site's verbs.

---

## Open Questions

1. **Queue token state endpoint**: The current engine store doesn't expose a public method to read `queue_limit_state` rows. We need to add a `ListQueueStates()` method or SQL query for the queue monitor page.

2. **Event persistence**: Scheduler events are currently ephemeral (emitted to an observer callback). For the dashboard to show historical events, we need either: (a) an in-memory ring buffer in the HTTP server, or (b) a new `events` table in the engine DB.

3. **Throughput aggregation**: Ops-per-minute requires counting completions over time windows. This could be computed client-side from event history or server-side with a dedicated query.

## References

- SCRAPER-HTTP-API ticket: HTTP API architecture and endpoint design
- `pkg/services/` — pre-extracted backend services
- `pkg/engine/model/types.go` — all domain types
