---
Title: Investigation diary
Ticket: SCRAPER-CLEANUP-SERVER-WORKER
Status: active
Topics:
    - scraper
    - backend
    - architecture
    - cleanup
    - api
    - server
    - worker
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Records why server.go and worker.go should be split by orchestration concern."
LastUpdated: 2026-04-07T16:05:00-04:00
WhatFor: "Resume the orchestration cleanup with the original problem framing intact."
WhenToUse: "Use when implementing or reviewing the server/worker split."
---

# Investigation diary

## Server observations

`server.go` mixes dependency assembly, route registration, middleware, request metrics, runtime-event emission, and runtime-event router startup.

## Worker observations

`worker.go` mixes Cobra command setup, scheduler/runtime setup, observer composition, and metrics listener boot.

## Recommendation

Split by orchestration role, not by tiny utility function. Keep one obvious entry file for the server and one for the worker command.

## First cleanup slice: API route registration

The first move-only slice targeted the easiest high-value split in `pkg/api/server/server.go`: route registration.

Files added:

- `pkg/api/server/routes_catalog.go`
- `pkg/api/server/routes_engine.go`
- `pkg/api/server/routes_runtime_events.go`

What moved:

- catalog and submission route wiring into `registerCatalogRoutes(...)`
- engine and workflow route wiring into `registerEngineRoutes(...)`
- runtime event HTTP routes into `registerRuntimeEventRoutes(...)`

What intentionally stayed in `server.go`:

- dependency assembly in `New(...)`
- request logging and metrics middleware
- runtime-event router startup
- shutdown wiring

Why this slice first:

- it reduces the biggest visual bulk in `server.go` without changing any runtime behavior
- it preserves `server.New(...)` as the obvious composition root
- it creates the file structure that later middleware and router-startup slices can plug into cleanly

Validation for this slice:

```bash
gofmt -w pkg/api/server/server.go pkg/api/server/routes_catalog.go pkg/api/server/routes_engine.go pkg/api/server/routes_runtime_events.go
go test ./pkg/api/server ./pkg/cmd -count=1
```

Both focused packages stayed green after the move.

## Second cleanup slice: request middleware

After route registration moved out, the next most obvious non-composition concern in `server.go` was the request middleware.

File added:

- `pkg/api/server/middleware_request.go`

What moved:

- `requestLogger(...)`
- `statusRecorder`
- `WriteHeader(...)`
- `Flush(...)`

Why this split was worth doing early:

- it removes request-level mechanics from the server assembly path
- it keeps logging, runtime-event request emission, and Prometheus HTTP observation together
- it leaves `server.go` focused on dependency assembly and lifecycle wiring

One small follow-up fix was needed after the move:

- `server.go` still needed the `watermill` import because `startRuntimeEventRouter(...)` still constructs a router with `watermill.NopLogger{}`

Validation for this slice:

```bash
gofmt -w pkg/api/server/server.go pkg/api/server/middleware_request.go
go test ./pkg/api/server ./pkg/cmd -count=1
```

Both focused packages passed again after the import cleanup.
