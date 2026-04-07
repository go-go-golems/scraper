---
Title: Investigation diary
Ticket: SCRAPER-CLEANUP-STORE-VIEW
Status: active
Topics:
    - scraper
    - backend
    - architecture
    - cleanup
    - sqlite
    - api
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Records why store.go and engineview/service.go are ready for decomposition."
LastUpdated: 2026-04-07T16:15:00-04:00
WhatFor: "Resume the cleanup plan with the original reasoning intact."
WhenToUse: "Use when implementing or reviewing the store/view split."
---

# Investigation diary

## Main observation

Two files have become broad aggregation points:

- `pkg/engine/store/sqlite/store.go`
- `pkg/services/engineview/service.go`

Both are still functional and internally understandable, but both now mix multiple subdomains that can be separated without changing package boundaries.

## Store findings

`store.go` currently owns workflow operations, op lifecycle, queue limiting, result persistence, artifact persistence, and helper functions. That makes every review and merge broader than it needs to be.

## Engine view findings

`service.go` currently owns workflow reads, queue reads, artifact reads, op-result reads, retry/cancel helpers, DB bootstrap helpers, and artifact preview classification. That is too much for one service file.

## Recommendation

Split by responsibility inside the existing packages first. Do not redesign interfaces during the first pass.

## Implementation log

### First cleanup slice: engineview artifact reads and DB helpers

I started with `pkg/services/engineview` because it is a lower-risk structural cleanup than the SQLite store and it sits behind existing tests.

Files added:

- `pkg/services/engineview/artifact_read_service.go`
- `pkg/services/engineview/db_helpers.go`

Files updated:

- `pkg/services/engineview/service.go`

What moved into `artifact_read_service.go`:

- `ArtifactSummary`
- `ArtifactDetail`
- `WorkflowArtifactsResult`
- `ListWorkflowArtifactsOptions`
- `ListArtifacts(...)`
- `ListWorkflowArtifacts(...)`
- `GetOpResult(...)`
- `GetArtifact(...)`

What moved into `db_helpers.go`:

- `openStore(...)`
- `openReadDB(...)`
- `workflowExists(...)`
- `workflowOpExists(...)`
- `enrichArtifactSummary(...)`
- `classifyArtifactPreview(...)`
- `loadDependencies(...)`
- `loadLease(...)`

What stayed in `service.go`:

- service type
- workflow reads
- queue reads
- retry/cancel mutations

Validation for this slice:

```bash
go test ./pkg/services/engineview ./pkg/api/server -count=1
```

Both packages passed after the move.
