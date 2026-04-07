---
Title: 'Dashboard Phase 2: Artifact Browser, Rate Limiter, Op Logs, Retry-Cancel, Script Viewer'
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
    - Path: pkg/engine/model/types.go
      Note: ArtifactWrite, OpError, RetryPolicy types
    - Path: pkg/engine/store/sqlite/store.go
      Note: CompleteOp artifact storage, loadArtifacts, queue limiter state
    - Path: pkg/js/runtime/executor.go
      Note: ctx.log implementation, ctx.writeArtifact, artifact/record writes
    - Path: pkg/sites/registry/registry.go
      Note: Definition struct with ScriptsFS for script viewer
    - Path: pkg/engine/store/sqlite/migrations/002_engine_runtime.sql
      Note: artifacts table, queue_limit_state table, leases table
ExternalSources: []
Summary: "Design for 5 new dashboard features: artifact browser, rate limiter detail, op execution logs, retry/cancel operations, and script source viewer"
LastUpdated: 2026-03-23T22:39:37.738357782-04:00
WhatFor: "Guide Phase 2 dashboard implementation"
WhenToUse: "When building the next batch of dashboard features"
---

# Dashboard Phase 2: Five New Features

## Executive Summary

Five features that transform the dashboard from a monitoring tool into a debugging and operations tool. Ordered by implementation dependency:

1. **Artifact Browser** — view fetched HTML, JSON results, and other artifacts inline per op
2. **Rate Limiter Detail View** — expanded queue monitor with token bucket visualization and policy config
3. **Op Execution Log** — per-op log capture from `ctx.log()` calls, stored as artifacts
4. **Retry/Cancel Operations** — force-retry failed ops, cancel running workflows from the UI
5. **Script Source Viewer** — read-only view of the JS that ran for a given op

## Backend Status

| Feature | Backend Ready? | What's Missing |
|---------|---------------|----------------|
| Artifact Browser | 85% | Need public `GetArtifact` + `ListArtifacts` store methods, API endpoint for artifact body |
| Rate Limiter Detail | 100% | ListQueues already returns token state. Need to surface QueuePolicy from registry |
| Op Execution Log | 20% | `ctx.log` writes to zerolog, not persisted. Store logs as a special artifact during execution |
| Retry/Cancel | 60% | `UpdateWorkflowStatus` exists. Need `RetryOp` and `CancelWorkflow` mutations + API endpoints |
| Script Source Viewer | 0% | Need to read from embedded FS via registry. Add `ListScripts` + `ReadScript` service methods + API |

---

## Feature 1: Artifact Browser

### Context

Ops produce artifacts via `ctx.writeArtifact()` (JS scripts) and the HTTP runner (response bodies). These are stored as BLOBs in the `artifacts` table with metadata (name, kind, content_type). Currently only accessible via direct SQLite queries.

### ASCII Mockup — Artifact Tab in Op Drawer

```
+------------------------------------------+
| Op: hn-001:frontpage-fetch               |
| Kind: http/fetch   Status: succeeded     |
|------------------------------------------|
| [Input] [Dependencies] [Result]          |
| [Artifacts] [Script] [Logs]              |
|------------------------------------------|
| ARTIFACTS (2)                            |
|                                          |
| +--------------------------------------+ |
| | frontpage.html                       | |
| | text/html  12.4 KB                  | |
| | +----------------------------------+ | |
| | | <!DOCTYPE html>                  | | |
| | | <html>                           | | |
| | |   <head>                         | | |
| | |     <title>Hacker News</title>   | | |
| | |   </head>                        | | |
| | |   <body>                         | | |
| | |     <table>                      | | |
| | |       ...                        | | |
| | +----------------------------------+ | |
| | [Download] [Open in New Tab]         | |
| +--------------------------------------+ |
|                                          |
| +--------------------------------------+ |
| | response-headers.json                | |
| | application/json  842 B              | |
| | +----------------------------------+ | |
| | | {                                | | |
| | |   "Content-Type": "text/html",  | | |
| | |   "Server": "nginx"             | | |
| | | }                                | | |
| | +----------------------------------+ | |
| +--------------------------------------+ |
+------------------------------------------+
```

### Widget Skeleton

```yaml
ArtifactList:
  description: "List of artifacts for an op with inline preview"
  props:
    artifacts: ArtifactSummary[]
    onSelect: (id: string) => void
    selectedId: string | null
  stories:
    - default: "2 artifacts (HTML + JSON)"
    - single: "One HTML artifact"
    - empty: "No artifacts"
    - many: "8 artifacts"

ArtifactPreview:
  description: "Inline artifact content viewer"
  props:
    artifact: ArtifactDetail
  features:
    - text/html: syntax-highlighted HTML in a scrollable pre block
    - application/json: JsonViewer (reuse existing)
    - text/plain: plain pre block
    - other: show metadata only + download button
  stories:
    - html: "HTML content with syntax color"
    - json: "JSON content via JsonViewer"
    - plain-text: "Plain text content"
    - binary: "Unknown type, download only"
    - large: "Content > 100KB, show truncated + full download"

ArtifactDownloadButton:
  description: "Button that fetches artifact body and triggers download"
  props:
    workflowId: string
    opId: string
    artifactId: string
    filename: string
  stories:
    - default: "Download button"
    - downloading: "Loading state"
```

### Backend Changes

```
NEW store method:  GetArtifacts(ctx, workflowID, opID) ([]ArtifactDetail, error)
NEW API type:      ArtifactDetail { ID, Name, Kind, ContentType, Metadata, Body string, Size int, CreatedAt }
NEW endpoint:      GET /api/v1/workflows/{wfID}/ops/{opID}/artifacts
NEW endpoint:      GET /api/v1/workflows/{wfID}/ops/{opID}/artifacts/{artifactID}
                   (returns raw body with correct Content-Type header for download)
NEW RTK Query:     workflowApi.getOpArtifacts(wfID, opID)
```

---

## Feature 2: Rate Limiter Detail View

### Context

The queue monitor already shows in-flight counts and token state from `ListQueues`. This feature adds a detailed view per queue showing the full QueuePolicy config, a token bucket gauge, and policy explanation.

### ASCII Mockup — Expanded Queue Row

```
+------------------------------------------------------------------+
| QUEUE MONITOR                                                    |
+------------------------------------------------------------------+
| Queue Key       | In-Flight | Max | Tokens | Rate   | Burst     |
|-----------------|-----------|-----|--------|--------|-----------|
| site:hn:http    |  2/4 ████ |  4  |  1.8   | 2/sec  |   4       |
|  [v Expand]                                                      |
|  +------------------------------------------------------------+  |
|  | RATE LIMITER DETAIL                                        |  |
|  |                                                            |  |
|  | Policy: token_bucket                                       |  |
|  | Max In-Flight: 4                                           |  |
|  | Rate: 2.0 ops/sec                                          |  |
|  | Burst: 4 tokens                                            |  |
|  |                                                            |  |
|  | Token Gauge:                                               |  |
|  |   [##-------] 1.8 / 4.0 tokens                            |  |
|  |   Refills at 2.0 tokens/sec                                |  |
|  |   Next full refill in ~1.1s                                |  |
|  |                                                            |  |
|  | Queue Stats:                                               |  |
|  |   Pending: 3   Ready: 5   Running: 2                      |  |
|  |   Succeeded: 120   Failed: 1                               |  |
|  +------------------------------------------------------------+  |
|                                                                  |
| site:hn:js      |  1/4 ██   |  4  |   -    | (none) |   -       |
| site:nv:http    |  3/4 ████ |  4  |  0.2   | 1/sec  |   2       |
+------------------------------------------------------------------+
```

### Widget Skeleton

```yaml
QueueDetailPanel:
  description: "Expandable detail panel for a single queue"
  props:
    queue: QueueStatus
    policy: QueuePolicyConfig | null
  stories:
    - with-rate-limit: "Token bucket policy with partial tokens"
    - without-rate-limit: "MaxInFlight only, no token bucket"
    - saturated: "All tokens consumed, in-flight at max"
    - idle: "Full tokens, zero in-flight"

TokenBucketGauge:
  description: "Visual gauge showing current tokens vs burst capacity"
  library: "Custom SVG or MUI LinearProgress"
  props:
    tokens: number
    burst: number
    ratePerSecond: number
  computed:
    timeToFull: "(burst - tokens) / ratePerSecond"
  stories:
    - full: "tokens == burst"
    - depleted: "tokens near 0"
    - half: "tokens at 50%"
    - no-rate-limit: "null tokens (no rate limiting)"

QueuePolicySummary:
  description: "Readable text summary of a queue's policy config"
  props:
    maxInFlight: number
    rateLimit: { kind: string; ratePerSecond: number; burst: number } | null
  stories:
    - token-bucket: "Full token bucket config"
    - max-in-flight-only: "No rate limiting"
```

### Backend Changes

```
NEW endpoint:      GET /api/v1/queues/{site}/{queue}/policy
                   Returns QueuePolicy from registry (MaxInFlight + RateLimit config)
NEW service:       catalogApi.GetQueuePolicy(site, queue) -> reads from registry
NEW RTK Query:     queueApi.getQueuePolicy(site, queue)

NOTE: The dynamic token state already comes from ListQueues.
      The policy config is static (from registry) - needs a new read path.
```

---

## Feature 3: Op Execution Log

### Context

JS scripts call `ctx.log(...)` which writes to zerolog (application stderr). The output is not persisted or retrievable per-op. For debugging, operators need to see what a script logged during execution.

### Design Decision: Logs as Artifacts

Store log output as a special artifact with `kind: "execution-log"` instead of creating a new table. This reuses existing artifact storage and the new artifact browser. During execution, the runtime captures log calls into a buffer and writes it as an artifact when the op completes.

### ASCII Mockup — Logs Tab in Op Drawer

```
+------------------------------------------+
| Op: hn-001:extract-p1                    |
| Kind: js   Script: extract_frontpage.js  |
|------------------------------------------|
| [Input] [Dependencies] [Result]          |
| [Artifacts] [Script] [Logs]              |
|------------------------------------------|
| EXECUTION LOG                            |
|                                          |
| 14:32:01.234  Parsing frontpage HTML     |
| 14:32:01.236  Found 30 stories           |
| 14:32:01.237  Extracting story: Show HN  |
| 14:32:01.238  Extracting story: Ask HN   |
| 14:32:01.240  Writing 30 rows to site DB |
| 14:32:01.245  Emitting page-2 fetch op   |
| 14:32:01.246  Done. 30 stories, 1 child  |
|                                          |
| [Download Full Log]                      |
+------------------------------------------+
```

### Widget Skeleton

```yaml
OpExecutionLog:
  description: "Scrollable log viewer for per-op execution output"
  props:
    entries: LogEntry[]
    loading: boolean
  features:
    - Monospace font, alternating row shading
    - Timestamp + message per line
    - Auto-scroll to bottom option
    - Search/filter within log
  stories:
    - default: "15 log entries from a JS extraction"
    - empty: "No log output"
    - long: "200+ entries with scroll"
    - loading: "Skeleton state"

LogEntry:
  type:
    timestamp: string
    message: string
```

### Backend Changes

```
MODIFY executor.go:
  - Add logBuffer []LogEntry to executionState
  - ctx.log() appends to logBuffer AND writes to zerolog
  - buildOpResult() creates an artifact { kind: "execution-log", body: JSON(logBuffer) }

MODIFY submitverbs/runtime.go:
  - Same pattern for submit verb logging

NOTE: This requires NO new tables, endpoints, or store methods.
      Logs appear as artifacts and are viewable through the artifact browser.
      The OpExecutionLog widget is just a specialized renderer for "execution-log" artifacts.
```

---

## Feature 4: Retry/Cancel Operations

### Context

Failed ops can only be retried automatically via the scheduler's retry policy. Operators have no way to manually retry a failed op or cancel a stuck workflow from the dashboard.

### ASCII Mockup — Action Buttons in Op Drawer + Workflow Header

```
Op Drawer (failed op):
+------------------------------------------+
| Op: sd-003:extract       Status: FAILED  |
|                          [Retry Op]      |
|------------------------------------------|
| ERROR                                    |
| Code: PARSE_ERROR                        |
| Message: unexpected <div> nesting        |
| Retryable: true                          |
|                                          |
| RETRY STATE                              |
| Attempt: 3/3 (exhausted)                |
| Last error: unexpected <div> nesting     |
+------------------------------------------+

Workflow Header (running workflow):
+------------------------------------------+
| Workflow: sd-003       Status: RUNNING   |
| Site: slashdot         [Cancel Workflow] |
| Created: 2 min ago                       |
+------------------------------------------+

Confirmation Dialog:
+----------------------------------+
| Cancel Workflow?                 |
|                                  |
| This will cancel workflow sd-003 |
| and all its pending/running ops. |
|                                  |
|      [Keep Running]  [Cancel]    |
+----------------------------------+
```

### Widget Skeleton

```yaml
RetryOpButton:
  description: "Button to retry a failed op, with confirmation"
  props:
    workflowId: string
    opId: string
    disabled: boolean
    onRetry: () => void
  states:
    - idle: "Retry Op button enabled"
    - confirming: "Confirm dialog open"
    - loading: "Retry in progress"
    - done: "Success snackbar"
  stories:
    - default: "Enabled retry button"
    - disabled: "Op is not failed"
    - confirming: "Dialog open"
    - loading: "Submitting retry"

CancelWorkflowButton:
  description: "Button to cancel a workflow, with confirmation dialog"
  props:
    workflowId: string
    status: WorkflowStatus
    onCancel: () => void
  states:
    - idle: "Cancel Workflow button (only shown for running/pending)"
    - confirming: "Confirmation dialog"
    - loading: "Cancel in progress"
  stories:
    - running-workflow: "Cancel button visible"
    - succeeded-workflow: "Button hidden (terminal state)"
    - confirming: "Dialog open"

ConfirmDialog:
  description: "Reusable confirmation dialog"
  props:
    open: boolean
    title: string
    message: string
    confirmLabel: string
    confirmColor: 'error' | 'primary'
    loading: boolean
    onConfirm: () => void
    onCancel: () => void
  stories:
    - cancel-workflow: "Cancel workflow confirmation"
    - retry-op: "Retry op confirmation"
```

### Backend Changes

```
NEW store method:  RetryOp(ctx, opID) error
                   - Validates op is in 'failed' status
                   - Resets status to 'ready', clears retry state
                   - Increments attempt counter

NEW store method:  CancelWorkflow(ctx, workflowID) error
                   - Sets workflow status to 'canceled'
                   - Cancels all non-terminal ops (pending, ready, running)
                   - Deletes leases for running ops

NEW endpoint:      POST /api/v1/workflows/{wfID}/ops/{opID}:retry
NEW endpoint:      POST /api/v1/workflows/{wfID}:cancel

NEW RTK Query:     workflowApi.retryOp (mutation, invalidates WorkflowOps)
                   workflowApi.cancelWorkflow (mutation, invalidates Workflow + WorkflowList)
```

---

## Feature 5: Script Source Viewer

### Context

Each op references a script via `metadata.script` (e.g., `"seed.js"`, `"extract_frontpage.js"`). The scripts are embedded in site packages via `go:embed`. Showing the source inline helps operators understand what an op does without switching to their editor.

### ASCII Mockup — Script Tab in Op Drawer

```
+------------------------------------------+
| Op: hn-001:extract-p1                    |
| Kind: js   Script: extract_frontpage.js  |
|------------------------------------------|
| [Input] [Dependencies] [Result]          |
| [Artifacts] [Script] [Logs]              |
|------------------------------------------|
| SCRIPT: extract_frontpage.js             |
| Site: hackernews                         |
|                                          |
| +--------------------------------------+ |
| |  1 | const helpers = require("./lib/ | |
| |  2 |   frontpage");                  | |
| |  3 |                                 | |
| |  4 | module.exports = function(ctx) { | |
| |  5 |   const dep = ctx.dep(          | |
| |  6 |     ctx.input.fetchedOpID);     | |
| |  7 |   if (!dep) return {            | |
| |  8 |     error: {                    | |
| |  9 |       code: "MISSING_DEP",     | |
| | 10 |       message: "no fetch data" | |
| | 11 |     }                           | |
| | 12 |   };                            | |
| | 13 |                                 | |
| | 14 |   const html = dep.artifacts    | |
| | 15 |     .find(a => a.name ===       | |
| | 16 |       "frontpage.html")         | |
| | 17 |     .bodyText;                  | |
| | 18 |                                 | |
| | 19 |   // ... parse and emit ...     | |
| +--------------------------------------+ |
|                                          |
| Also requires:                           |
|   scripts/lib/frontpage.js               |
+------------------------------------------+
```

### Widget Skeleton

```yaml
ScriptViewer:
  description: "Read-only source code viewer with line numbers"
  props:
    source: string
    language: 'javascript'
    filename: string
  features:
    - Line numbers
    - Monospace font
    - Horizontal scroll for long lines
    - Syntax highlighting (basic: keywords, strings, comments)
  stories:
    - short-script: "20-line seed.js"
    - long-script: "100-line extract with helpers"
    - empty: "Script not found"

ScriptTab:
  description: "Tab content that loads and displays a script"
  props:
    site: string
    scriptPath: string
  features:
    - Loads script via API
    - Shows loading skeleton
    - Shows error if script not found
    - Links to required modules
  stories:
    - loaded: "Script content displayed"
    - loading: "Skeleton"
    - not-found: "Error state"
    - with-requires: "Shows linked helper scripts"

ScriptListPanel:
  description: "Sidebar or dropdown listing all scripts for a site"
  props:
    site: string
    scripts: string[]
    selected: string
    onSelect: (path: string) => void
  stories:
    - hackernews: "3 scripts"
    - nereval: "5 scripts + lib/"
```

### Backend Changes

```
NEW service method: catalog.ListScripts(site) -> []string
                    Walks ScriptsFS to list all .js files under ScriptsRoot

NEW service method: catalog.ReadScript(site, path) -> string
                    Reads file content from ScriptsFS

NEW endpoint:       GET /api/v1/sites/{site}/scripts
                    Returns { scripts: string[] }

NEW endpoint:       GET /api/v1/sites/{site}/scripts/{path...}
                    Returns { source: string, path: string }
                    Path validation to prevent traversal

NEW RTK Query:      catalogApi.listScripts(site)
                    catalogApi.getScript(site, path)
```

---

## Unified Op Detail Drawer — Tab Layout

All five features integrate into an enhanced OpDetailDrawer with tabs:

```
+------------------------------------------+
| Op: hn-001:extract-p1                    |
| Kind: js   Script: extract_frontpage.js  |
| Status: succeeded        [Retry Op]     |
|------------------------------------------|
| [Input] [Deps] [Result] [Artifacts]     |
| [Script] [Logs]                          |
|------------------------------------------|
|                                          |
|  (tab content here)                      |
|                                          |
+------------------------------------------+
```

```yaml
OpDetailDrawerTabs:
  tabs:
    - id: input
      label: Input
      content: JsonViewer (existing)
    - id: dependencies
      label: Deps
      content: Dependency list (existing)
    - id: result
      label: Result
      content: Result data + error + retry (existing, restructured)
    - id: artifacts
      label: Artifacts
      badge: artifact count
      content: ArtifactList + ArtifactPreview (NEW)
    - id: script
      label: Script
      content: ScriptTab (NEW)
    - id: logs
      label: Logs
      content: OpExecutionLog (NEW)
```

---

## Implementation Plan

### Phase 2A: Artifact Browser (backend + frontend)

| Task | Est. |
|------|------|
| Add `GetArtifacts` store method + SQL query | 30m |
| Add artifacts API endpoint (list + raw download) | 45m |
| Add RTK Query endpoint | 15m |
| Build ArtifactList widget + stories | 45m |
| Build ArtifactPreview widget + stories (HTML/JSON/text renderers) | 1h |
| Integrate into OpDetailDrawer as Artifacts tab | 30m |
| **Subtotal** | **~3.5h** |

### Phase 2B: Rate Limiter Detail (mostly frontend)

| Task | Est. |
|------|------|
| Add queue policy endpoint (read from registry) | 30m |
| Build TokenBucketGauge widget + stories | 45m |
| Build QueueDetailPanel (expandable row) + stories | 45m |
| Build QueuePolicySummary widget + stories | 30m |
| Integrate into QueueStatusTable as expandable rows | 30m |
| **Subtotal** | **~2.5h** |

### Phase 2C: Op Execution Log (backend + frontend)

| Task | Est. |
|------|------|
| Modify executor.go: capture ctx.log into buffer | 30m |
| Write log buffer as "execution-log" artifact in buildOpResult | 30m |
| Same for submitverbs/runtime.go | 20m |
| Build OpExecutionLog widget + stories | 45m |
| Integrate into OpDetailDrawer as Logs tab | 20m |
| **Subtotal** | **~2.5h** |

### Phase 2D: Retry/Cancel (backend + frontend)

| Task | Est. |
|------|------|
| Add RetryOp store method | 30m |
| Add CancelWorkflow store method | 30m |
| Add API endpoints (retry + cancel) | 30m |
| Add RTK Query mutations | 15m |
| Build ConfirmDialog widget + stories | 30m |
| Build RetryOpButton + CancelWorkflowButton + stories | 45m |
| Integrate into OpDetailDrawer and WorkflowHeader | 30m |
| **Subtotal** | **~3.5h** |

### Phase 2E: Script Source Viewer (backend + frontend)

| Task | Est. |
|------|------|
| Add ListScripts + ReadScript service methods | 45m |
| Add API endpoints (list + read) | 30m |
| Add RTK Query endpoints | 15m |
| Build ScriptViewer widget + stories (with line numbers) | 1h |
| Build ScriptTab wrapper + stories | 30m |
| Integrate into OpDetailDrawer as Script tab | 20m |
| **Subtotal** | **~3h** |

### Phase 2F: Unified Op Drawer Tabs

| Task | Est. |
|------|------|
| Refactor OpDetailDrawer into tabbed layout | 45m |
| Wire all new tabs (artifacts, script, logs) | 30m |
| Add action buttons (retry/cancel) to drawer header | 20m |
| Page-level stories for the complete drawer | 30m |
| **Subtotal** | **~2h** |

### Total: ~17 hours

### Recommended Order

1. **2A Artifacts** — unblocks 2C (logs stored as artifacts)
2. **2C Logs** — depends on artifact storage
3. **2F Tabbed Drawer** — refactor before adding more tabs
4. **2D Retry/Cancel** — integrates into refactored drawer
5. **2E Scripts** — independent, integrates into drawer
6. **2B Rate Limiter** — independent of drawer, enhances queue page

---

## New Component Summary

| Category | New Components | New Stories |
|----------|---------------|-------------|
| Artifacts | ArtifactList, ArtifactPreview, ArtifactDownloadButton | 10 |
| Rate Limiter | QueueDetailPanel, TokenBucketGauge, QueuePolicySummary | 8 |
| Logs | OpExecutionLog | 4 |
| Retry/Cancel | RetryOpButton, CancelWorkflowButton, ConfirmDialog | 8 |
| Script | ScriptViewer, ScriptTab, ScriptListPanel | 7 |
| Integration | OpDetailDrawerTabs (refactor) | 6 |
| **Total** | **14** | **~43** |

## New API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/workflows/{wfID}/ops/{opID}/artifacts` | List op artifacts with metadata |
| GET | `/api/v1/workflows/{wfID}/ops/{opID}/artifacts/{aID}` | Download artifact body |
| GET | `/api/v1/queues/{site}/{queue}/policy` | Queue policy from registry |
| POST | `/api/v1/workflows/{wfID}/ops/{opID}:retry` | Retry failed op |
| POST | `/api/v1/workflows/{wfID}:cancel` | Cancel workflow |
| GET | `/api/v1/sites/{site}/scripts` | List site scripts |
| GET | `/api/v1/sites/{site}/scripts/{path}` | Read script source |
