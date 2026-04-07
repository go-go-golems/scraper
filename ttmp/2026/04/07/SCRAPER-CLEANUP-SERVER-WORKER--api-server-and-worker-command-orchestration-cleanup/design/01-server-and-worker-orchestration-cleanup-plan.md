---
Title: Server and worker orchestration cleanup plan
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
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Detailed plan for splitting API server wiring and worker orchestration into smaller files."
LastUpdated: 2026-04-07T16:05:00-04:00
WhatFor: "Guide a low-risk split of the server and worker orchestration code."
WhenToUse: "Use when implementing or reviewing the server/worker cleanup."
---

# Server and worker orchestration cleanup plan

## API server target layout

```text
pkg/api/server/
  server.go
  routes_catalog.go
  routes_engine.go
  routes_runtime_events.go
  middleware_request.go
  runtime_event_router.go
```

## Worker target layout

```text
pkg/cmd/
  worker.go
  worker_runtime.go
  worker_metrics.go
  worker_observers.go
```

## Principles

- keep entrypoints obvious
- move code before redesigning code
- keep routes, flags, and behavior stable during the first pass

## Validation

```bash
go test ./pkg/api/server ./pkg/cmd -count=1
go test ./... -count=1
```
