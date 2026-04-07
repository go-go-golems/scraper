---
Title: Artifact browser and op result implementation guide
Ticket: SCRAPER-ARTIFACT-BROWSER
Status: active
Topics:
    - scraper
    - backend
    - frontend
    - http-api
    - artifacts
    - workflows
    - onboarding
DocType: design
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Detailed intern-facing implementation guide for workflow artifact browsing, downloading, and op-result retrieval."
LastUpdated: 2026-04-07T15:24:00-04:00
WhatFor: "Implement workflow artifact browser APIs and UI surfaces without taking on JS replay yet."
WhenToUse: "Use when implementing the artifact browser backend or wiring the workflow UI to browse artifacts and inspect op results."
---

# Artifact browser and op result implementation guide

## Why this ticket exists

The scraper already stores durable workflow results and artifacts, but the product experience is still too op-local.

Today, a user can inspect one op at a time. That is not enough when a workflow produces multiple HTML bodies, summaries, logs, or derivative JSON artifacts across many ops. A practical operator or site author needs to answer three questions quickly:

1. what artifacts exist for this workflow?
2. which op produced each artifact?
3. what was the result payload of that op?

This ticket builds the backend and API layer for those questions. It intentionally does not implement JS replay. Replay is a later ticket because it adds execution semantics and safety constraints that are not necessary to ship artifact browsing.

## Audience

This document is written for a new intern who needs to understand:

- how artifacts are stored,
- how the engine view service exposes workflow state,
- how the HTTP API is assembled,
- how the current React workflow pages consume those APIs,
- and what exactly must change to build a workflow artifact browser.

## Current system map

```mermaid
flowchart LR
    Runner[Runner executes op]
    Store[SQLite engine store]
    EngineView[engineview service]
    Handler[engine handler]
    API[HTTP API]
    UI[Workflow UI]

    Runner --> Store
    Store --> EngineView
    EngineView --> Handler
    Handler --> API
    API --> UI
```

The important thing to understand is that artifacts are already durable. The missing part is not persistence. The missing part is the read model and read API shape for workflow-level browsing.

## Current backend components

### 1. Engine store

The engine SQLite store already persists:

- workflow rows,
- op rows,
- result rows,
- artifact rows.

Relevant code:

- [store.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/store.go)

The existing `GetResult(...)` logic already reconstructs an `OpResult` and attaches artifacts for a single op.

### 2. Engine view service

The engine view service is the read-model layer used by the HTTP API.

Relevant code:

- [service.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/services/engineview/service.go)

It already provides:

- workflow summary
- workflow ops
- queue list
- per-op artifact listing
- artifact body download

That makes it the right place to add:

- workflow artifact listing
- op result retrieval

### 3. Engine HTTP handler

Relevant code:

- [engine.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/api/handlers/engine.go)

This handler currently exposes:

- workflow endpoints
- workflow op list
- per-op artifact list
- artifact download

It does not yet expose:

- workflow artifact list
- op result retrieval

### 4. Server route registration

Relevant code:

- [server.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/api/server/server.go)

Any new engine handler method needs a corresponding route registered here.

### 5. Frontend seams

Relevant code:

- [workflowApi.ts](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/api/workflowApi.ts)
- [OpDetailDrawer.tsx](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/workflows/OpDetailDrawer.tsx)

The frontend already has:

- a `getOpResult` query pointing to `/workflows/{workflowId}/ops/{opId}/result`
- a `getOpArtifacts` query for per-op artifacts
- an op drawer that can display result and artifacts

So the backend work in this ticket directly unblocks and stabilizes existing UI behavior.

## API design

### Endpoint 1: workflow artifact list

`GET /api/v1/workflows/{workflowID}/artifacts`

Purpose:

- list artifacts across the entire workflow
- allow grouping by op
- support future filtering by kind/content type/name

Suggested v1 query parameters:

- `opId`
- `kind`
- `contentType`
- `search`
- `limit`
- `offset`

Recommended v1 response:

```json
{
  "workflowID": "wf-123",
  "total": 1,
  "artifacts": [
    {
      "id": "wf-123:frontpage-fetch:response-body",
      "opID": "wf-123:frontpage-fetch",
      "workflowID": "wf-123",
      "name": "frontpage.html",
      "kind": "http-response-body",
      "contentType": "text/html",
      "metadata": {
        "source": "fetch"
      },
      "size": 48123,
      "createdAt": "2026-04-07T14:20:01Z",
      "previewable": true,
      "previewKind": "html"
    }
  ]
}
```

### Endpoint 2: op result

`GET /api/v1/workflows/{workflowID}/ops/{opID}/result`

Purpose:

- provide the exact stored result for a single op
- normalize missing-result behavior
- support both current op detail UI and later debugger/replay work

Recommended behavior:

- `200` with a `null` result body if the workflow/op exists but the op has no result yet
- `404` only if the workflow or op truly does not exist and the service can determine that

Recommended v1 response:

```json
{
  "result": {
    "OpID": "wf-123:frontpage-extract",
    "Data": { "stories": 30 },
    "Records": [],
    "Artifacts": [
      {
        "ID": "wf-123:frontpage-extract:summary-json",
        "Name": "summary.json",
        "Kind": "json-output",
        "ContentType": "application/json"
      }
    ],
    "EmittedIDs": [
      "wf-123:page-2-fetch"
    ],
    "CompletedAt": "2026-04-07T14:20:09Z"
  }
}
```

## DTO recommendations

### Workflow artifact list response

Use a dedicated response type rather than reusing the per-op artifact response verbatim. That keeps the API self-describing.

Suggested Go DTO:

```go
type WorkflowArtifactListResponse struct {
    WorkflowID model.WorkflowID              `json:"workflowID"`
    Total      int                           `json:"total"`
    Artifacts  []engineview.ArtifactSummary  `json:"artifacts"`
}
```

### Op result response

Suggested Go DTO:

```go
type OpResultResponse struct {
    Result *model.OpResult `json:"result"`
}
```

This keeps the frontend contract explicit and avoids returning a bare object at the route root.

## Service-layer design

### New engine view methods

Add:

- `ListWorkflowArtifacts(ctx, workflowID)`
- `GetOpResult(ctx, workflowID, opID)`

Pseudocode:

```text
ListWorkflowArtifacts(ctx, workflowID):
    open read db
    verify workflow exists or return nil/empty according to chosen behavior
    query artifacts where workflow_id = ?
    order by created_at, op_id, id
    map rows -> ArtifactSummary
    return list

GetOpResult(ctx, workflowID, opID):
    open store
    optionally verify workflow/op exists
    call store.GetResult(ctx, workflowID, opID)
    return result
```

## Handler and route design

### Handler methods to add

In [engine.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/api/handlers/engine.go):

- `WorkflowArtifacts`
- `OpResult`

### Routes to add

In [server.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/api/server/server.go):

- `GET /api/v1/workflows/{workflowID}/artifacts`
- `GET /api/v1/workflows/{workflowID}/ops/{opID}/result`

## Screen designs

These are v1 screen sketches for the workflow artifact browser. They are deliberately simple and fit the current UI architecture.

### Screen 1: workflow page with artifact tab

```text
+--------------------------------------------------------------------------------+
| Workflow: hackernews-extract-frontpage-...                                     |
| Status: running    Site: hackernews    Total Ops: 10                           |
+--------------------------------------------------------------------------------+
| Tabs: Overview | Ops | Runtime | Artifacts | Site Info                         |
+--------------------------------------------------------------------------------+
| Filters: [All Ops v] [All Kinds v] [All Types v] [ search................ ]    |
+--------------------------------------------------------------------------------+
| Artifact Name           | Op ID                    | Kind        | Size | Open  |
| frontpage.html          | ...:frontpage-fetch      | http-body   | 48k  | View  |
| execution-log.json      | ...:frontpage-extract    | exec-log    | 12k  | View  |
| summary.json            | ...:frontpage-extract    | json-output | 2k   | View  |
+--------------------------------------------------------------------------------+
| Right pane / modal preview when row selected                                   |
| - text preview for json/log/html                                               |
| - download/open raw link                                                       |
| - link to owning op                                                            |
+--------------------------------------------------------------------------------+
```

### Screen 2: artifact preview drawer

```text
+---------------------------------------------------------------+
| Artifact: summary.json                                        |
| Op: ...:frontpage-extract                                     |
| Kind: json-output    Content-Type: application/json           |
| Size: 2 KB            Created: 2026-04-07 14:20:09           |
+---------------------------------------------------------------+
| Actions: [Open Raw] [Download] [Go To Op]                     |
+---------------------------------------------------------------+
| Preview                                                        |
| {                                                              |
|   "stories": 30,                                               |
|   "nextPage": "..."                                            |
| }                                                              |
+---------------------------------------------------------------+
```

### Screen 3: op drawer result + artifacts coherence

```text
+---------------------------------------------------------------+
| Op Detail: ...:frontpage-extract                              |
| Tabs: Input | Deps | Result | Artifacts | Runtime | Script    |
+---------------------------------------------------------------+
| Result tab                                                    |
| - Data JSON                                                   |
| - Error block if present                                      |
| - Artifact count and emitted op count                         |
| - Link: "Open workflow artifact browser filtered to this op"  |
+---------------------------------------------------------------+
```

## UX principles

- The artifact browser should be workflow-first.
- Artifact body retrieval should stay separate from list retrieval.
- The browser should be useful before filtering and pagination are perfect.
- The owning op must always be visible because artifacts alone are not enough context.

## Implementation order

### Phase 1

1. Add workflow artifact service method.
2. Add workflow artifact handler and route.
3. Add op result service method.
4. Add op result handler and route.
5. Add tests.

### Phase 2

1. Add frontend query for workflow artifacts.
2. Add the workflow artifact browser tab.
3. Reuse existing artifact preview/download logic.

## Tests to write

### Service tests

- workflow artifact listing returns artifacts from multiple ops in one workflow
- workflow artifact listing excludes artifacts from other workflows
- op result retrieval returns stored result
- op result retrieval returns nil when no result exists

### HTTP/server tests

- `GET /api/v1/workflows/{workflowID}/artifacts` returns the expected list
- `GET /api/v1/workflows/{workflowID}/ops/{opID}/result` returns the expected result envelope
- artifact download still works unchanged

## Non-goals

This ticket does not:

- implement JS replay
- add mutable debug execution
- change artifact persistence format
- add a second storage layer

## Final recommendation

Ship the backend in this order:

1. workflow artifact list
2. op result route
3. tests
4. UI integration

That gets the useful browsing and inspection surface into the product quickly and creates clean contracts for the later replay/debug ticket.
