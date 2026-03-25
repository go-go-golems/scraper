---
Title: Event transport options and implementation plan for worker, server, and frontend
Ticket: SCRAPER-RUNTIME-EVENTS
Status: active
Topics:
    - architecture
    - scraper
    - worker
    - server
    - http-api
    - scheduler
    - api
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: buf.gen.yaml
      Note: Go and web code generation config
    - Path: pkg/api/server/server.go
      Note: HTTP server boundary and future event delivery surface
    - Path: pkg/cmd/worker.go
      Note: Worker process boundary and current observer wiring
    - Path: pkg/engine/scheduler/scheduler.go
      Note: Existing scheduler event seam and observer contract
    - Path: pkg/runtimeevents/codec.go
      Note: Binary and protojson helpers built on the generated event type
    - Path: pkg/runtimeevents/watermill.go
      Note: Topic
    - Path: pkg/runtimeevents/watermill_test.go
      Note: GoChannel-backed validation of the first Watermill integration slice
    - Path: pkg/services/submission/service.go
      Note: Server-side workflow submission seam that should emit events
    - Path: pkg/sites/submitverbs/host.go
      Note: Submission-time workflow creation path that currently uses a nil observer
    - Path: proto/scraper/runtime/v1/events.proto
      Note: Canonical protobuf event contract shared between Go and TypeScript
    - Path: web/package.json
      Note: Frontend protobuf runtime dependency for generated event decoding
ExternalSources: []
Summary: Compares the available transport and topology choices for runtime events in scraper, then records the decision to standardize on Watermill plus a protobuf-generated Go and TypeScript event contract.
LastUpdated: 2026-03-24T20:16:15-04:00
WhatFor: Guide the decision and implementation of a runtime event pipeline that can carry scheduler events, logs, and other live operational data from workers and servers to the frontend using Watermill and protobuf-generated Go/TS types.
WhenToUse: Use when implementing the Watermill-based runtime event pipeline, when wiring protobuf schema generation for Go and TS, or when extending dashboard-facing real-time event handling.
---




# Event transport options and implementation plan for worker, server, and frontend

## Executive Summary

The scraper codebase already has a useful event seam inside the scheduler, but it stops inside the worker process. The worker emits `scheduler.Event` values through an optional observer callback, while the HTTP server is a separate process that only reads persisted engine state from SQLite. That means the current architecture is good at durable workflow execution, but it has no built-in path for real-time logs, event streaming, or live frontend telemetry.

There were four realistic choices:

1. stay with SQLite-only persistence and polling
2. merge worker and server and use an in-process pub/sub
3. keep worker and server separate and bridge over Redis
4. define a transport abstraction and support both local in-process and Redis-backed delivery

The recommendation is now fixed. Keep the current worker/server split as the default architecture, use Watermill as the standard eventing layer, and define the canonical event contract in protobuf so both Go and TypeScript are generated from the same schema. Use a Redis-backed Watermill transport for cross-process delivery, use an in-process Watermill transport for tests and optional local `server+worker` mode, and expose events to the frontend with `protojson`.

## Problem Statement

We want scraper to emit more than durable workflow state. Operators want live scheduler events, log lines, and other runtime signals that can be collected, inspected, and shown in the frontend.

The current code has part of that story already:

- the scheduler exposes `Event` and `Observer` in `pkg/engine/scheduler/scheduler.go`
- the worker creates the scheduler with `observer == nil` in `pkg/cmd/worker.go`
- workflow submission in `pkg/sites/submitverbs/host.go` also creates a scheduler with `observer == nil`
- the HTTP server in `pkg/api/server/server.go` has no event transport or streaming endpoint
- the earlier dashboard design explicitly assumed polling and called out event persistence as unresolved

That leaves several gaps:

- worker events are ephemeral and local to one process
- submission-time events are not exposed to the HTTP server
- request logs and worker logs have no shared event contract
- the frontend cannot subscribe to or replay recent runtime activity
- there is no development environment for a Redis-backed transport

The design question is not just "how do we send events" but "what process topology do we want to preserve while doing it."

## Proposed Solution

Keep the worker and HTTP server as separate roles, but standardize runtime events behind a protobuf-defined scraper-owned event model carried over Watermill so both single-process and Redis-backed deployments use the same contract.

### Recommended architecture

```text
worker process
  -> scheduler observer
  -> runner/log adapters
  -> RuntimeEventV1 proto
  -> Watermill message
  -> Watermill Publisher
  -> Redis-backed pub/sub

HTTP server
  -> submission/request events
  -> RuntimeEventV1 proto
  -> Watermill message
  -> Watermill Publisher
  -> Redis-backed pub/sub
  -> Watermill Subscriber/Router
  -> recent-event buffer + optional log/artifact persistence
  -> protojson
  -> /api/v1/events
  -> /api/v1/events/stream (SSE)
  -> frontend
  -> fromJson(RuntimeEventV1Schema, ...)
```

### Core design elements

1. Define a canonical protobuf event envelope.
   - `RuntimeEventV1` message with `schema_version`
   - stable protobuf package name
   - stable ID
   - source type (`worker`, `server`, `submission`, `scheduler`, `runner`, `request`)
   - event kind (`op_leased`, `op_succeeded`, `op_failed`, `workflow_created`, `request_served`, `log_line`, and so on)
   - timestamp fields
   - workflow/op/site/queue identifiers when applicable
   - severity level
   - structured payload
   - optional artifact/log reference for large bodies

2. Generate both Go and TypeScript from the same schema.
   - add `proto/` at repo root
   - use Buf-managed generation
   - generate Go into the Go module
   - generate TS into `web/src/pb`
   - use `js+dts` output for the web because this repo enables `erasableSyntaxOnly`
   - emit JSON with `protojson`
   - decode JSON in TS with `fromJson`

3. Introduce a Watermill-based transport boundary.
   - scraper owns `RuntimeEventV1`
   - adapter layer maps `RuntimeEventV1 <-> message.Message`
   - use protobuf binary inside Watermill messages
   - worker and server code talk to a thin local wrapper, not directly to backend-specific plumbing everywhere
   - implementations:
     - Watermill GoChannel for tests and optional single-process dev mode
     - Watermill Redis-backed transport for real cross-process delivery

4. Emit events from both worker and server paths.
   - worker: scheduler observer, runner lifecycle, optional log adapters
   - server: submission lifecycle, request lifecycle, future admin actions

5. Let the HTTP server own frontend-facing delivery.
   - keep recent events in a bounded in-memory buffer for quick history
   - expose history by HTTP
   - expose real-time updates by SSE first
   - serialize with `protojson` at the HTTP boundary

6. Treat large logs separately from small event envelopes.
   - event stream carries metadata and short payloads
   - larger blobs become artifacts or log files referenced by event ID

### Why this fits the current codebase

This preserves the architecture already documented in scraper:

- workers execute durable ops and own scheduling
- the API server remains an operator/read-write façade
- SQLite continues to store durable workflow state
- the new transport solves only the missing real-time cross-process channel

It also matches the likely future shape better than a permanent merged process. The existing `worker run` and `api serve` commands already represent separate operational roles.

## Design Decisions

### Decision 1: Keep the worker/server split as the primary architecture

The current code intentionally separates submission/inspection from execution. `pkg/cmd/worker.go` owns the scheduler loop, while `pkg/api/server/server.go` constructs an HTTP server over service objects. Keeping that split means the new event system adds transport, not a new runtime model.

### Decision 2: Define the event contract in protobuf and generate Go + TS types

The event envelope should not live only as an internal Go struct. The user wants event generation on both the Go and TS sides, and the protobuf skill fits that directly:

- one schema defines the wire contract
- Go code gets generated message types
- the frontend gets generated TS schema/types
- the HTTP/SSE boundary can use `protojson`
- TS can decode with `fromJson`

This also avoids hand-maintained mirror types between backend and frontend.

Recommended protobuf conventions:

- add `schema_version` to the top-level event message
- use a stable package such as `scraper.runtime.v1`
- prefer string identifiers for workflow/op/site/queue IDs
- prefer protobuf timestamp types over ad hoc integer timestamp fields where possible
- use open-shape fields sparingly, with clear intent

### Decision 3: Use Watermill as the standard messaging layer

The user explicitly chose Watermill as the easiest standardized way to handle event transport. That is a reasonable fit here because scraper needs the same publish/subscribe flow in multiple places:

- worker to transport
- server to transport
- transport to server-side consumers
- tests and local single-process mode

Watermill gives one messaging model across those cases. Scraper should still keep a small wrapper around it so domain code deals in generated event types, not raw transport concerns.

### Decision 4: Use protobuf binary inside Watermill and `protojson` at the HTTP boundary

The transport boundary and the web boundary have different needs:

- inside Watermill, protobuf binary keeps the payload exact, compact, and strongly typed
- at the HTTP/SSE boundary, `protojson` gives the frontend and operators readable payloads

This avoids paying the JSON cost on every internal publish/subscribe hop while still keeping the browser-facing interface ergonomic.

### Decision 5: Use Redis-backed Watermill transport for cross-process delivery

This is an inference from the use case. Pure pub/sub is good for transient fan-out, but the request includes log collection and other real-time data that operators may want to inspect after the fact. Streams are a better fit for:

- replay from a recent offset
- multiple consumers
- server restarts without losing the entire recent event window
- bounded history with trimming

Raw pub/sub can still be added later for ultra-low-latency fan-out if needed.

### Decision 6: Use GoChannel for tests and optional local single-process mode

Watermill's in-process backend is a good fit for:

- unit and integration tests
- a local `server+worker` mode if we want one
- exercising the same message handlers without requiring Redis

This is better than inventing a second non-Watermill local bus if Watermill is now the standard.

### Decision 7: Use SSE before WebSocket

The dashboard planning docs already leaned toward polling because there was no event bus. Once an event bus exists, the lowest-complexity real-time HTTP delivery is SSE:

- server to browser is one-way, which matches the use case
- browser support is straightforward
- backend implementation is smaller than WebSocket session management

### Decision 8: Keep the scraper-owned protobuf envelope even though Watermill is chosen

Watermill should not become the domain model. Scraper still needs its own stable event envelope so that:

- frontend contracts stay scraper-shaped
- logging/artifact decisions stay local to scraper
- backend changes do not force wide refactors
- message metadata can be normalized in one place
- Buf and protobuf generation stay aligned across Go and TS

## Alternatives Considered

### Option A: SQLite persistence plus frontend polling only

Description:
Persist events in SQLite or keep them in a server-local ring buffer and let the frontend poll `/api/v1/events`.

Pros:

- smallest implementation delta
- no new infrastructure
- matches the current dashboard ticket assumptions
- easy to debug locally

Cons:

- does not solve cross-process real-time transport cleanly
- poor fit for live log tails
- server-local ring buffers miss worker events unless the worker and server are merged
- polling cadence becomes the de facto latency budget

Verdict:
Useful as a fallback or for historical views, but not enough as the primary architecture for live events and log collection.

### Option B: Merge worker and server and use in-process pub/sub

Description:
Run the scheduler loop inside the same binary or process as the API server and fan out events over Watermill GoChannel.

Pros:

- fastest path to a working demo
- no Redis required on day one
- direct access to scheduler observer events
- simplest path to a server-local recent-event buffer

Cons:

- changes the operational model of the system
- one process crash now removes both API and execution
- scaling multiple workers later becomes a second redesign
- still does not solve the real multi-process transport problem by itself

Verdict:
Reasonable as an optional development mode, not a good permanent default.

### Option C: Keep the split and bridge over Redis immediately

Description:
Workers and servers stay separate. Both publish runtime events through Watermill and the server subscribes for frontend delivery over a Redis-backed Watermill transport.

Pros:

- matches the current architecture
- works with multiple workers
- makes frontend streaming and external consumers straightforward
- creates a real transport boundary early

Cons:

- adds infrastructure and local setup cost
- requires clearer semantics for replay, retention, and backpressure
- log volume can become expensive if payload design is careless

Verdict:
Strong production-oriented choice. Best if paired with a transport abstraction and a disciplined envelope.

### Option D: Hybrid abstraction with optional single-process mode

Description:
Use Watermill as the standard layer everywhere, with GoChannel for tests/local mode and Redis-backed transport for cross-process delivery, while still keeping an optional `server+worker` mode instead of making it the default deployment model.

Pros:

- lowest-regret path
- lets development start without blocking on infrastructure
- preserves the current architecture while still allowing a simplified local mode
- gives one standardized message flow across local and distributed modes

Cons:

- still requires discipline to keep the scraper-owned event wrapper thin and stable

Verdict:
Best fit for the current codebase and the user’s decision to standardize on Watermill without collapsing the worker/server split.

## Implementation Plan

### Phase 1: Stabilize the event model

1. Add `proto/` plus Buf config and define the canonical event schema.
2. Generate Go and TS outputs from the same schema.
3. Add a new package for Watermill adapters around the generated event types.
4. Map existing `scheduler.Event` values into the canonical envelope.
5. Decide which event classes are:
   - replayable
   - best-effort transient
   - large-payload references only

### Phase 2: Add local emission points

1. Worker:
   - pass a real observer into `scheduler.New(...)` in `pkg/cmd/worker.go`
   - add adapters for runner and worker log emission where worthwhile
2. Server:
   - emit submission events around `submission.Service.Submit(...)`
   - emit request lifecycle events in the HTTP middleware
3. Submission host:
   - stop hardcoding `observer == nil` around workflow creation when submission should emit events

### Phase 3: Add Watermill transport implementations

1. GoChannel transport for tests and optional single-process mode
2. Redis-backed transport for cross-process mode
3. Configuration surface for Redis URL, topic names, trimming policy, and consumer identity
4. Docker Compose file for local Redis development

### Phase 4: Add server-side frontend delivery

1. Add a recent-event buffer in the API server
2. Add `GET /api/v1/events` for bounded history
3. Add `GET /api/v1/events/stream` as SSE
4. Serialize generated proto messages with `protojson`
5. Add filtering by site, workflow, event kind, and severity

### Phase 5: Add log and artifact handling

1. Decide which logs stay inline versus spill to files/artifacts
2. Add references from events to stored blobs
3. Expose artifact/log lookup endpoints if needed

### Phase 6: Add end-to-end validation

1. single-process test with Watermill GoChannel
2. multi-process test with Redis
3. frontend consumption smoke test
4. worker crash/server restart behavior tests

### Suggested initial scope

If the goal is to move quickly without painting the architecture into a corner, the first implementation slice should be:

- canonical event envelope
- protobuf schema and Buf generation
- Watermill wrapper plus GoChannel transport
- worker scheduler observer emission
- server recent-event buffer
- SSE endpoint
- optional `server+worker` mode for local demos only

Then add Redis-backed Watermill transport once the envelope and endpoint contract feel stable.

## Open Questions

1. Should log lines be first-class runtime events, or should events only reference external log blobs?
2. Should recent history live only in Redis Streams, or should the API server also maintain its own bounded in-memory cache for fast reads?
3. Do we want a combined `daemon` command or a flag on `api serve` for optional local `server+worker` mode?
4. Which submission-time events matter most: workflow accepted, workflow created, site migration opened, or command validation outcome?
5. Do we want tenant-like stream partitioning by site, by environment, or by event class?
6. Where should generated Go code live inside this repo so imports stay clean with `paths=source_relative`?

## References

- Existing dashboard design notes on event history and polling in `SCRAPER-DASHBOARD`
- `pkg/engine/scheduler/scheduler.go`
- `pkg/cmd/worker.go`
- `pkg/api/server/server.go`
- `pkg/services/submission/service.go`
- `pkg/sites/submitverbs/host.go`
