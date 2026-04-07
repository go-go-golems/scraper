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

## Third cleanup slice: runtime-event router startup

The last API-server concern still mixed into `server.go` was runtime-event router startup.

File added:

- `pkg/api/server/runtime_event_router.go`

What moved:

- `startRuntimeEventRouter(...)`

What this achieved:

- `server.go` now mostly reads as a composition root:
  - open event resources
  - build services and handlers
  - register routes
  - wrap middleware
  - wire shutdown cleanup
- route registration, request middleware, and event-router startup are all out in dedicated files

Validation for this slice:

```bash
gofmt -w pkg/api/server/server.go pkg/api/server/runtime_event_router.go
go test ./pkg/api/server ./pkg/cmd -count=1
```

The focused packages passed again, so the API half of this cleanup ticket is structurally complete.

## Fourth cleanup slice: worker observer composition

With the API-side splits done, the first worker-side extraction was the smallest orchestration concern: observer composition.

File added:

- `pkg/cmd/worker_observers.go`

What moved:

- `composeSchedulerObservers(...)`

What was added:

- `newWorkerObserver(...)` as the worker-specific composition helper that wires:
  - runtime-event scheduler observation
  - Prometheus scheduler observation

Why this helped:

- it removes one policy/composition detail from the `RunE` body in `worker.go`
- it makes the later `worker_runtime.go` extraction easier because observer wiring already lives elsewhere

Validation for this slice:

```bash
gofmt -w pkg/cmd/worker.go pkg/cmd/worker_observers.go
go test ./pkg/api/server ./pkg/cmd -count=1
```

The focused packages stayed green after the move.

## Fifth cleanup slice: worker metrics listener boot

The next worker-specific concern was the Prometheus listener helper.

File added:

- `pkg/cmd/worker_metrics.go`

What moved:

- `maybeStartWorkerMetricsServer(...)`

Why this split mattered:

- the helper is operational infrastructure, not worker command definition
- it includes listener boot, graceful shutdown on context cancellation, and warning logs on unexpected server exit
- moving it out leaves `worker.go` closer to “flags plus runtime flow”

Validation for this slice:

```bash
gofmt -w pkg/cmd/worker.go pkg/cmd/worker_metrics.go
go test ./pkg/api/server ./pkg/cmd -count=1
```

The focused packages passed again after the move.

## Sixth cleanup slice: worker runtime setup

The final worker-side extraction moved the large operational `RunE` body out of `worker.go`.

File added:

- `pkg/cmd/worker_runtime.go`

What moved:

- the full worker runtime flow into `runWorkerCommand(...)`

That runtime flow still does the same work as before:

- build and validate config
- open engine and scraper databases
- open runtime-event publisher resources
- create the metrics registry
- build runner registry and site DB provider
- construct the scheduler
- boot optional worker metrics HTTP listener
- execute scheduler cycles
- print the final loop summary

What `worker.go` became after this move:

- worker option struct
- Cobra command construction
- flag registration
- a one-line `RunE` that delegates to `runWorkerCommand(...)`

This satisfies the intended layout from the design:

- `worker.go` for Cobra and flags
- `worker_runtime.go` for the operational flow
- `worker_metrics.go` for metrics listener boot
- `worker_observers.go` for observer composition

Validation for this final slice:

```bash
gofmt -w pkg/cmd/worker.go pkg/cmd/worker_runtime.go
go test ./pkg/api/server ./pkg/cmd -count=1
go test ./... -count=1
```

The focused packages and the full repository both passed after the move.
