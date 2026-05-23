---
Title: Intern guide to sessionstream-backed scraper runtime events
Ticket: SCRAPER-SESSIONSTREAM-EVENTS
Status: active
Topics:
    - scraper
    - events
    - websocket
    - architecture
    - onboarding
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/pinocchio/pkg/chatapp/chat.go
      Note: Pinocchio schema registration and hub installation pattern for sessionstream apps
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/pinocchio/pkg/chatapp/projections.go
      Note: Pinocchio backend-event to UI/timeline projection pattern
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/pinocchio/pkg/chatapp/runner.go
      Note: Pinocchio reusable sessionstream runner wiring
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/pinocchio/pkg/chatapp/runtime_inference.go
      Note: Pinocchio command handler publishing typed backend sessionstream events
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/pinocchio/pkg/chatapp/service.go
      Note: Pinocchio app-facing service wrapping Hub.Submit
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/pinocchio/proto/pinocchio/chatapp/v1/chat.proto
      Note: Real-world sessionstream consumer defining app-specific protobuf command/event/entity messages
    - Path: scraper/pkg/api/handlers/runtime_events.go
      Note: Current REST and SSE runtime-event handler to delete during the sessionstream migration
    - Path: scraper/pkg/runtimeevents/watermill.go
      Note: Current scraper Watermill runtime event codec and metadata path being replaced/bridged
    - Path: scraper/proto/scraper/runtime/v1/events.proto
      Note: Defines RuntimeEventV1
    - Path: scraper/web/src/api/runtimeEventsApi.ts
      Note: Current frontend REST plus EventSource consumer to migrate to sessionstream websocket
    - Path: sessionstream/pkg/sessionstream/bus.go
      Note: Sessionstream Watermill integration used to connect worker producers to API consumers
    - Path: sessionstream/pkg/sessionstream/hub.go
      Note: Sessionstream hub command/event/projection/fanout pipeline that becomes the core mechanism
    - Path: sessionstream/pkg/sessionstream/transport/ws/server.go
      Note: Sessionstream websocket snapshot and fanout adapter for browser delivery
ExternalSources: []
Summary: Design and implementation guide for replacing scraper runtime event fanout with sessionstream-backed sessions, projections, snapshots, and websocket delivery.
LastUpdated: 2026-05-22T21:55:00-04:00
WhatFor: Use when implementing or reviewing the sessionstream-backed runtime event distribution mechanism for scraper.
WhenToUse: Before changing scraper runtimeevents, API streaming routes, worker event publication, or the runtime-events frontend feed.
---



# Intern guide to sessionstream-backed scraper runtime events

## Executive summary

The `scraper` repository already emits rich runtime events from API requests, submissions, scheduler decisions, workers, and runners. Today those events flow through a scraper-owned `pkg/runtimeevents` package, a Watermill topic, an API-local in-memory hub, a REST endpoint, and a Server-Sent Events stream. That works, but it duplicates infrastructure that now exists in `./sessionstream`: typed events, per-session routing, Watermill-backed publish/consume, snapshot hydration, UI projection, websocket fanout, and ordered ordinals.

This design proposes making `./sessionstream` the only runtime-event distribution mechanism for live scraper progress. The old scraper-owned in-memory runtime event hub, REST recent-event endpoint, SSE endpoint, and scraper-specific Watermill protobuf-byte codec should be removed rather than preserved for backwards compatibility. Scraper should define protobuf messages for the sessionstream application layer: explicit commands, backend events, UI events, and timeline entities that carry or refine scraper runtime-event data. Runtime events are published into one or more session ids such as `runtime:global` and `workflow:<workflow-id>`. The API process runs the sessionstream consumer, projects backend runtime events into UI events and timeline entities, and exposes a websocket endpoint that clients subscribe to.

The goal for a new intern is not merely to wire a websocket. The important idea is to treat runtime events as a sessionstream application:

- **Scraper owns the domain schema**: scraper-specific protobuf commands/events/entities, including `RuntimeEventV1` or narrower event payloads for sources, severities, workflow ids, op ids, worker ids, and payload shape.
- **Sessionstream owns distribution mechanics**: session ids, event ordinals, snapshots, projection state, fanout, reconnect behavior, and transport frames.
- **The API process owns the live view**: it consumes bus events, persists a bounded/hydratable timeline, and serves snapshots plus live UI events to clients.
- **Workers and runners remain dumb producers**: they publish typed events with stable routing ids and do not know how many browsers are connected.

## Problem statement and scope

### What the user asked for

Create a new docmgr ticket for using `./sessionstream` as the core mechanism to distribute events coming from running scrapers, progress, and related runtime activity for `./scraper`. Produce a detailed analysis, design, and implementation guide aimed at a new intern, and upload the result to reMarkable.

### In scope

This guide covers:

- the current scraper runtime-event pipeline;
- the relevant sessionstream primitives;
- the proposed scraper-to-sessionstream mapping;
- new packages, APIs, routes, and frontend wiring;
- implementation phases;
- tests and validation steps;
- removal plan for the old REST/SSE/in-memory runtime-event stack.

### Out of scope

This guide does not implement the code. It also does not redesign the scraper engine, scheduler, JS runner, or site manifests. Those systems remain sources of runtime events. The design only changes how events are routed, persisted for live views, hydrated on reconnect, and distributed to browser clients.

## Vocabulary

| Term | Meaning in this design |
| --- | --- |
| Backend runtime event | A domain event emitted by scraper code, represented by `scraper.runtime.v1.RuntimeEventV1`. |
| Session | A routing and hydration scope managed by sessionstream. Examples: `runtime:global`, `workflow:wf-123`. |
| Backend event | A sessionstream `Event` containing a protobuf payload and a session id. |
| UI event | A sessionstream `UIEvent` sent to websocket clients after projection. |
| Timeline entity | A sessionstream persisted view entity used to build snapshots. For runtime events, each entity can represent one recent event. |
| Hydration | The subscribe-time snapshot flow: client receives current entities, then live events. |
| Fanout | Delivery of projected UI events to every connection subscribed to the session id. |

## Current-state architecture in scraper

### Current runtime event schema

Scraper runtime events are already protobuf messages. The source schema is `scraper/proto/scraper/runtime/v1/events.proto`. The main message starts at line 10 and includes fields for schema version, id, source, component, kind, severity, timestamp, message, workflow id, op id, site, queue, worker id, request id, artifact id, labels, and arbitrary structured payload.

Important file reference:

- `scraper/proto/scraper/runtime/v1/events.proto:10` defines `message RuntimeEventV1`.
- `scraper/proto/scraper/runtime/v1/events.proto:30` defines `RuntimeEventSource`.
- `scraper/proto/scraper/runtime/v1/events.proto:40` defines `RuntimeEventSeverity`.
- `scraper/proto/scraper/runtime/v1/events.proto:48` defines `RuntimeEventKind`.

The existing schema is good enough to reuse. Do not replace it with a generic JSON blob. Sessionstream expects protobuf payloads, and `RuntimeEventV1` is already exactly that.

### Current event producers

Runtime events are produced in several places:

- Scheduler observer events are mapped from `scheduler.Event` in `scraper/pkg/runtimeevents/scheduler.go` and emitted by `scraper/pkg/runtimeevents/scheduler_observer.go`.
- Runner start/complete/fail events are emitted by `scraper/pkg/runtimeevents/runner.go`.
- API request events are emitted by request middleware in `scraper/pkg/api/server/middleware_request.go`.
- Submission events are emitted by `scraper/pkg/services/submission/service.go` and `scraper/pkg/api/handlers/submission.go`.
- Worker command setup opens event resources and passes the publisher into scheduler/runner wiring in `scraper/pkg/cmd/worker_runtime.go:54`.

Observed facts:

- `scraper/pkg/runtimeevents/scheduler_observer.go:14` returns a scheduler observer that publishes mapped runtime events.
- `scraper/pkg/runtimeevents/runner.go:17` defines `ObservedRunner`, and its `Run` method emits `runner started` and completion/failure events.
- `scraper/pkg/cmd/worker_runtime.go:54` opens runtime event publisher resources before building runner and scheduler observers.

### Current transport and buffering

The current event backend is scraper-owned. `scraper/pkg/runtimeevents/backend.go:14` defines `Backend` with values `off`, `gochannel`, and `redis`. The backend opens Watermill publishers/subscribers around either an in-process gochannel or Redis Streams.

The current message codec lives in `scraper/pkg/runtimeevents/watermill.go`:

- `TopicRuntimeEventsV1` is `scraper.runtime.v1.events`.
- `MessageFromEvent` starts at `scraper/pkg/runtimeevents/watermill.go:78` and encodes a `RuntimeEventV1` as protobuf bytes plus Watermill metadata.
- `EventFromMessage` decodes the protobuf payload back into `RuntimeEventV1`.

The API process currently consumes the Watermill topic and pushes decoded events into an in-memory hub:

- `scraper/pkg/api/server/runtime_event_router.go:12` starts the runtime event router.
- The router handler decodes `runtimeevents.EventFromMessage(msg)` and calls `hub.Add(event)`.
- `scraper/pkg/runtimeevents/hub.go:19` defines an in-memory hub with a recent-events buffer and subscriber channels.

### Current HTTP/SSE API

The current API exposes:

- `GET /api/v1/runtime-events` for recent buffered events.
- `GET /api/v1/runtime-events/stream` for live Server-Sent Events.

File references:

- `scraper/pkg/api/server/routes_runtime_events.go:9` registers the runtime event routes.
- `scraper/pkg/api/handlers/runtime_events.go:19` lists recent events.
- `scraper/pkg/api/handlers/runtime_events.go:35` streams events via SSE.

The SSE handler subscribes to the in-memory hub, writes heartbeat comments every 15 seconds, and emits `event: runtime-event` frames with JSON data. This is simple, but it has several limitations:

- SSE is one-way; there is no shared command/subscription frame model.
- The in-memory hub has only recent buffered events, not a persisted hydratable timeline.
- Filtering is implemented as query params over a single process-local buffer.
- Reconnect semantics depend on browser/EventSource behavior rather than explicit snapshot ordinals.
- It duplicates sessionstream's websocket, fanout, hydration, observer, and projection machinery.

### Current frontend consumption

The React frontend currently fetches recent events and then opens an SSE stream:

- `scraper/web/src/api/runtimeEventsApi.ts:48` builds the REST query URL.
- `scraper/web/src/api/runtimeEventsApi.ts:59` builds the SSE URL.
- `scraper/web/src/api/runtimeEventsApi.ts:107` creates `new EventSource(sseUrl)`.

The cache update path deduplicates by event id and keeps at most 500 events. That frontend behavior should survive the migration, but its transport should change from EventSource to the sessionstream websocket protocol.

## Relevant sessionstream architecture

Sessionstream is a generic substrate for session-routed protobuf events. Its package doc says the design goals are one canonical routing key (`SessionId`), typed commands in, typed backend events out, sibling UI and timeline projections, and storage/transport behind small public interfaces.

### Core types

Important file references:

- `sessionstream/pkg/sessionstream/types.go:5` defines `SessionId` as the universal routing key.
- `sessionstream/pkg/sessionstream/types.go:21` defines `Event` with `Name`, `Payload`, `SessionId`, and `Ordinal`.
- `sessionstream/pkg/sessionstream/handler.go:5` defines `CommandHandler`.
- `sessionstream/pkg/sessionstream/handler.go:8` defines `EventPublisher`.

Conceptual model:

```text
Command{Name, Payload, SessionId}
  -> CommandHandler
  -> EventPublisher.Publish(Event{Name, Payload, SessionId})
  -> Hub assigns/receives ordinal
  -> UIProjection and TimelineProjection
  -> HydrationStore.Apply(...)
  -> UIFanout.PublishUI(...)
  -> Websocket clients
```

### Hub and event pipeline

The `Hub` is the central entrypoint. It owns schema registry, hydration store, sessions, command registry, projections, fanout, optional bus config, and observers.

Important file references:

- `sessionstream/pkg/sessionstream/hub.go:159` constructs a hub.
- `sessionstream/pkg/sessionstream/hub.go:215` submits a typed command into the hub.
- `sessionstream/pkg/sessionstream/hub.go:274` runs the configured bus consumer.
- `sessionstream/pkg/sessionstream/hub.go:463` applies one backend event through event append, projections, store apply, cursor advance, and fanout.

### Watermill bus integration

Sessionstream already speaks Watermill. This is the key bridge for scraper because scraper already uses Watermill and Redis Streams.

Important file references:

- `sessionstream/pkg/sessionstream/bus.go:15` defines `DefaultEventBusTopic`.
- `sessionstream/pkg/sessionstream/bus.go:102` defines `WithEventBus(pub, sub, opts...)`.
- `sessionstream/pkg/sessionstream/bus.go:128` defines the JSON event envelope with event name, session id, and payload.
- `sessionstream/pkg/sessionstream/consumer.go:54` subscribes to the bus topic.
- `sessionstream/pkg/sessionstream/consumer.go:71` decodes each message, assigns an ordinal, and projects/applies it.

A producer hub configured with `WithEventBus` publishes events to the bus. A consumer hub configured with the same schema and bus calls `Run(ctx)` to consume those messages and drive projection/fanout.

### Hydration store and snapshots

Sessionstream separates storage behind interfaces:

- `sessionstream/pkg/sessionstream/hydration.go:6` defines `HydrationStore` with `Apply`, `Snapshot`, `View`, and `Cursor`.
- `sessionstream/pkg/sessionstream/hydration.go:14` defines `EventStore` for append/replay.
- `sessionstream/pkg/sessionstream/hydration/sqlite/store.go:21` implements `HydrationStore`, `EventStore`, `ProjectionCursorStore`, `TimelineResetStore`, and error stores.

This matters for scraper because browser clients should not only receive future progress. They should subscribe to a workflow and immediately receive the latest known progress snapshot.

### Websocket transport

Sessionstream's websocket transport is already an HTTP handler and fanout adapter:

- `sessionstream/pkg/sessionstream/transport/ws/server.go:56` documents the websocket server as a snapshot/fanout adapter.
- `sessionstream/pkg/sessionstream/transport/ws/server.go:181` fans projected UI events to subscribed clients.
- `sessionstream/pkg/sessionstream/transport/ws/server.go:257` handles a subscribe frame, loads a snapshot, sends it, flushes hydration buffers, and marks the subscription live.
- `sessionstream/pkg/sessionstream/transport/ws/observer.go:13` defines transport observation stages.
- `sessionstream/proto/sessionstream/v1/transport.proto:8` defines `ClientFrame`.
- `sessionstream/proto/sessionstream/v1/transport.proto:18` defines `ServerFrame`.

The browser protocol is not plain SSE. It uses protobuf JSON frames shaped by `sessionstream.v1.ClientFrame` and `sessionstream.v1.ServerFrame`.

## Real-world reference: Pinocchio chatapp

`/home/manuel/code/wesen/go-go-golems/pinocchio` is a useful example because it is a third-party application that uses sessionstream at its core while defining its own domain commands, backend events, UI events, and timeline entities.

Key files:

- `/home/manuel/code/wesen/go-go-golems/pinocchio/proto/pinocchio/chatapp/v1/chat.proto` defines app-specific protobuf messages such as `StartInferenceCommand`, `StopInferenceCommand`, `ChatRunStarted`, `ChatTextPatch`, `ChatTextSegmentFinished`, and `ChatMessageEntity`.
- `/home/manuel/code/wesen/go-go-golems/pinocchio/pkg/chatapp/chat.go` defines logical names, registers schemas, and installs command handlers/projections on a sessionstream hub.
- `/home/manuel/code/wesen/go-go-golems/pinocchio/pkg/chatapp/runtime_inference.go` handles commands and publishes typed backend events through `sessionstream.EventPublisher`.
- `/home/manuel/code/wesen/go-go-golems/pinocchio/pkg/chatapp/projections.go` maps backend events into UI events and timeline entities.
- `/home/manuel/code/wesen/go-go-golems/pinocchio/pkg/chatapp/runner.go` shows reusable runner wiring: schema registry, hydration store, hub, engine, fanout, and service.
- `/home/manuel/code/wesen/go-go-golems/pinocchio/pkg/chatapp/service.go` wraps raw `Hub.Submit` calls behind app-facing methods such as `SubmitPrompt` and `Stop`.

The scraper implementation should copy this architectural shape rather than treating sessionstream as a generic pipe for one opaque message type:

```text
Pinocchio pattern
  proto domain messages
  -> RegisterSchemas(reg)
  -> Install(hub, engine)
  -> app Service methods call hub.Submit(...)
  -> command handlers publish typed backend events
  -> UI projection + timeline projection
  -> UIFanout / websocket / JSONL / TUI adapters

Scraper equivalent
  proto runtime-stream messages
  -> RegisterSchemas(reg)
  -> Install(hub, runtime engine/publisher)
  -> worker/API/submission code calls publisher.Publish(ctx, event)
  -> command handler publishes RuntimeEventObserved
  -> UI projection emits RuntimeEventAppended
  -> timeline projection persists RuntimeEventEntity
  -> sessionstream websocket adapter serves browser clients
```

Two details from Pinocchio should be applied directly:

1. **Use protobuf contracts for app-specific commands/events/entities.** Pinocchio does not register anonymous maps or plain JSON; it registers concrete protobuf messages in `RegisterSchemas`.
2. **Hide raw sessionstream calls behind app-facing services/adapters.** Pinocchio callers use `Service.SubmitPrompt`, while scraper producers should use a runtime-event publisher/service instead of importing command names everywhere.

## Gap analysis

### What scraper already has

Scraper already has:

- a useful `RuntimeEventV1` protobuf schema;
- event producers in scheduler, worker, runner, API, and submission code;
- Watermill/Redis resource configuration;
- an HTTP API and frontend feed;
- tests around runtime event API behavior.

### What sessionstream adds

Sessionstream adds:

- explicit session routing with `SessionId`;
- typed command/event registration through `SchemaRegistry`;
- hub-level event processing and observer hooks;
- event ordinals per session;
- timeline projection and snapshot hydration;
- websocket fanout with subscribe/unsubscribe/ping/pong frames;
- hydration buffering to avoid losing live events during snapshot load;
- SQLite-backed event and projection storage.

### Main mismatch

The main mismatch is that scraper's current runtime-event API is filter-oriented, while sessionstream is session-oriented.

Current scraper filtering:

```text
GET /api/v1/runtime-events?workflowId=wf-1&opId=op-2&workerId=worker-a
```

Sessionstream subscription:

```text
ClientFrame{subscribe:{session_id:"workflow:wf-1", since_snapshot_ordinal:123}}
```

The implementation should not try to hide this mismatch with complicated server-side ad-hoc filters inside the websocket path. Instead, define a small routing vocabulary and let each UI surface subscribe to the session that matches its scope.

## Proposed architecture

### Design principle

Use sessionstream as the canonical live distribution layer. Keep scraper-specific runtime event semantics in scraper. That means:

- Runtime event payloads stay as `RuntimeEventV1`.
- Existing event producers call a scraper-facing publisher interface.
- The publisher routes one runtime event to one or more sessionstream sessions.
- Sessionstream assigns per-session ordinals, persists/replays, projects, and fans out.
- Frontend code subscribes to sessionstream websocket sessions instead of opening SSE.

### High-level diagram

```text
+----------------------+        +---------------------------+
| scraper worker       |        | scraper API process       |
|                      |        |                           |
| scheduler observer   |        | sessionstream Hub.Run     |
| runner observer      |        | consumes Watermill topic  |
| request/submission   |        |                           |
| publisher adapter    |        | UI projection             |
+----------+-----------+        | timeline projection       |
           |                    | SQLite hydration store    |
           | Watermill/Redis    | ws.Server as UIFanout     |
           v                    +-------------+-------------+
+----------------------+                      |
| sessionstream topic  |                      | websocket frames
| scraper.runtime...   |                      v
+----------------------+        +---------------------------+
                                | React runtime event feed  |
                                | subscribes to sessions    |
                                +---------------------------+
```

### Routing model

Use deterministic session ids. Session ids are strings, but they need a stable grammar.

Recommended initial session ids:

| Session id | Receives | Used by |
| --- | --- | --- |
| `runtime:global` | Every runtime event | Global Runtime Events page |
| `workflow:<workflowId>` | Events with a workflow id | Workflow detail page |
| `op:<workflowId>:<opId>` | Events with both workflow id and op id | Future op detail live panel |
| `worker:<workerId>` | Events with a worker id | Future worker diagnostics page |
| `site:<site>` | Events with a site id | Future site-focused diagnostics page |

Phase 1 should publish to `runtime:global` and `workflow:<workflowId>` only. That minimizes duplicate writes while supporting the two current UI surfaces: global runtime feed and workflow details.

Pseudo-routing:

```go
func RuntimeEventSessionIDs(event *runtimev1.RuntimeEventV1) []sessionstream.SessionId {
    ids := []sessionstream.SessionId{"runtime:global"}

    if event.GetWorkflowId() != "" {
        ids = append(ids, sessionstream.SessionId("workflow:"+event.GetWorkflowId()))
    }

    // Defer these until the UI needs them.
    // if event.GetWorkflowId() != "" && event.GetOpId() != "" { ... }
    // if event.GetWorkerId() != "" { ... }
    // if event.GetSite() != "" { ... }

    return dedupeSessionIDs(ids)
}
```

### Schema registration

Create scraper-specific protobuf messages for the sessionstream application layer, then register those messages with `sessionstream.SchemaRegistry`. This follows the real-world Pinocchio pattern: Pinocchio defines domain commands/events/entities in `proto/pinocchio/chatapp/v1/chat.proto`, registers each logical command/event/UI-event/entity in `pkg/chatapp/chat.go`, and installs command handlers plus projections on a sessionstream hub.

Recommended proto file:

```text
scraper/proto/scraper/runtime/sessionstream/v1/runtime_stream.proto
```

Recommended generated Go package:

```text
scraper/gen/proto/scraper/runtime/sessionstream/v1
```

Recommended adapter package:

```text
scraper/pkg/runtimeevents/sessionstream/
  names.go
  schema.go
  publisher.go
  projections.go
  hub.go
```

Initial protobuf sketch:

```proto
syntax = "proto3";

package scraper.runtime.sessionstream.v1;

import "scraper/runtime/v1/events.proto";

option go_package = "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/sessionstream/v1;scraperruntimestreamv1";

// Command submitted by scraper producers into the sessionstream hub.
message PublishRuntimeEventCommand {
  scraper.runtime.v1.RuntimeEventV1 event = 1;
}

// Backend event emitted by the command handler and consumed by projections.
message RuntimeEventObserved {
  scraper.runtime.v1.RuntimeEventV1 event = 1;
}

// UI event sent to browser clients through sessionstream websocket fanout.
message RuntimeEventAppended {
  scraper.runtime.v1.RuntimeEventV1 event = 1;
}

// Timeline entity persisted in the sessionstream hydration store.
message RuntimeEventEntity {
  scraper.runtime.v1.RuntimeEventV1 event = 1;
}
```

Names:

```go
const (
    CommandPublishRuntimeEvent = "scraper.runtime.PublishRuntimeEvent"
    EventRuntimeEventObserved  = "scraper.runtime.RuntimeEventObserved"
    UIEventRuntimeEventAppended = "scraper.runtime.RuntimeEventAppended"
    EntityRuntimeEvent          = "scraper.runtime.RuntimeEvent"
)
```

Schema setup:

```go
func RegisterSchemas(reg *sessionstream.SchemaRegistry) error {
    for _, err := range []error{
        reg.RegisterCommand(CommandPublishRuntimeEvent, &streamv1.PublishRuntimeEventCommand{}),
        reg.RegisterEvent(EventRuntimeEventObserved, &streamv1.RuntimeEventObserved{}),
        reg.RegisterUIEvent(UIEventRuntimeEventAppended, &streamv1.RuntimeEventAppended{}),
        reg.RegisterTimelineEntity(EntityRuntimeEvent, &streamv1.RuntimeEventEntity{}),
    } {
        if err != nil {
            return err
        }
    }
    return nil
}
```

Why not reuse bare `RuntimeEventV1` for every sessionstream slot?

- A command is not the same thing as a backend event, and a backend event is not the same thing as a projected UI event or persisted entity.
- Explicit protobuf messages make contracts discoverable in generated Go/TypeScript APIs.
- The wrapper messages leave room for sessionstream-specific fields later, such as routing reason, producer process id, retention class, display hints, or ingest metadata.
- This matches Pinocchio's proven pattern: distinct protobuf command/event/entity messages, all registered explicitly with sessionstream.

`RuntimeEventV1` can still remain the inner payload at first. If the scraper runtime grows beyond the generic envelope, the same proto package can add narrower event messages such as `WorkflowCreated`, `OpLeased`, `OpSucceeded`, `RunnerLogLine`, and `QueueRateLimited` and register them as separate backend/UI event types.

### Command handler

The command handler adapts scraper publication commands into backend sessionstream events. The input payload is a scraper-specific command message, and the emitted backend event is a scraper-specific event message.

```go
func RegisterRuntimeEventCommand(hub *sessionstream.Hub) error {
    return hub.RegisterCommand(CommandPublishRuntimeEvent,
        func(ctx context.Context, cmd sessionstream.Command, sess *sessionstream.Session, pub sessionstream.EventPublisher) error {
            command, ok := cmd.Payload.(*streamv1.PublishRuntimeEventCommand)
            if !ok || command.GetEvent() == nil {
                return fmt.Errorf("publish runtime event payload must be %T, got %T", &streamv1.PublishRuntimeEventCommand{}, cmd.Payload)
            }

            normalized := proto.Clone(command.GetEvent()).(*runtimev1.RuntimeEventV1)
            if normalized.Id == "" {
                normalized.Id = stableOrRandomEventID(normalized)
            }
            if normalized.SchemaVersion == 0 {
                normalized.SchemaVersion = runtimeevents.SchemaVersionV1
            }
            if normalized.OccurredAt == nil {
                normalized.OccurredAt = timestamppb.Now()
            }

            return pub.Publish(ctx, sessionstream.Event{
                Name:      EventRuntimeEventObserved,
                SessionId: cmd.SessionId,
                Payload:   &streamv1.RuntimeEventObserved{Event: normalized},
            })
        })
}
```

A producer process calls `Hub.Submit(ctx, sid, CommandPublishRuntimeEvent, &streamv1.PublishRuntimeEventCommand{Event: event})`. If the hub has `WithEventBus`, the handler publishes through sessionstream's Watermill envelope. If the hub is local-only, the event is projected in process. That makes tests easy.

### Publisher adapter

Keep a scraper-facing publisher so existing producer code does not import sessionstream everywhere.

```go
type Publisher struct {
    hub *sessionstream.Hub
}

func (p *Publisher) Publish(ctx context.Context, event *runtimev1.RuntimeEventV1) error {
    if p == nil || p.hub == nil || event == nil {
        return nil
    }
    normalized := normalizeRuntimeEvent(event)
    ids := RuntimeEventSessionIDs(normalized)

    var errs []error
    for _, sid := range ids {
        payload := &streamv1.PublishRuntimeEventCommand{
            Event: proto.Clone(normalized).(*runtimev1.RuntimeEventV1),
        }
        if err := p.hub.Submit(ctx, sid, CommandPublishRuntimeEvent, payload); err != nil {
            errs = append(errs, fmt.Errorf("session %s: %w", sid, err))
        }
    }
    return errors.Join(errs...)
}
```

No backwards-compatibility shim is required. Update producer call sites to pass `context.Context` and delete the old context-free `runtimeevents.Publisher.Publish(event)` shape when the sessionstream publisher lands.

### Projection design

Use two projections:

1. **UI projection**: every backend runtime event becomes one `RuntimeEventAppended` UI event.
2. **Timeline projection**: every backend runtime event becomes one `RuntimeEventEntity` timeline entity keyed by event id, with bounded retention enforced by the projection.

UI projection pseudocode:

```go
type RuntimeEventUIProjection struct{}

func (RuntimeEventUIProjection) Project(ctx context.Context, ev sessionstream.Event, sess *sessionstream.Session, view sessionstream.TimelineView) ([]sessionstream.UIEvent, error) {
    if ev.Name != EventRuntimeEventObserved {
        return nil, nil
    }
    observed, ok := ev.Payload.(*streamv1.RuntimeEventObserved)
    if !ok || observed.GetEvent() == nil {
        return nil, fmt.Errorf("unexpected payload %T", ev.Payload)
    }
    return []sessionstream.UIEvent{{
        Name: UIEventRuntimeEventAppended,
        Payload: &streamv1.RuntimeEventAppended{
            Event: proto.Clone(observed.GetEvent()).(*runtimev1.RuntimeEventV1),
        },
    }}, nil
}
```

Timeline projection pseudocode:

```go
type RuntimeEventTimelineProjection struct {
    MaxEntitiesPerSession int
}

func (p RuntimeEventTimelineProjection) Project(ctx context.Context, ev sessionstream.Event, sess *sessionstream.Session, view sessionstream.TimelineView) ([]sessionstream.TimelineEntity, error) {
    if ev.Name != EventRuntimeEventObserved {
        return nil, nil
    }
    observed := ev.Payload.(*streamv1.RuntimeEventObserved)
    event := observed.GetEvent()
    id := event.GetId()
    if id == "" {
        id = fmt.Sprintf("ordinal-%020d", ev.Ordinal)
    }

    entities := []sessionstream.TimelineEntity{{
        Kind:             EntityRuntimeEvent,
        Id:               id,
        CreatedOrdinal:   ev.Ordinal,
        LastEventOrdinal: ev.Ordinal,
        Payload: &streamv1.RuntimeEventEntity{
            Event: proto.Clone(event).(*runtimev1.RuntimeEventV1),
        },
    }}

    entities = append(entities, p.tombstonesForRetention(view, ev.Ordinal)...)
    return entities, nil
}
```

Retention warning: sessionstream's generic store can hold all timeline entities unless you tombstone old ones. The current scraper hub keeps a bounded recent buffer (`DefaultRecentEventLimit` is 256 in `scraper/pkg/runtimeevents/backend.go:25`). Preserve that operational expectation by adding retention in Phase 2:

- list existing `runtime-event` entities from the view;
- keep the newest N by `LastEventOrdinal`;
- emit tombstone entities for older ids.

### Hub construction in scraper

Create one helper for producer-only hubs and one helper for API/server hubs.

Producer hub:

```go
func NewProducer(runtimeBus RuntimeEventBusConfig) (*Publisher, io.Closer, error) {
    reg, err := NewSchemaRegistry()
    if err != nil { return nil, nil, err }

    resources, err := OpenSessionstreamWatermillResources(runtimeBus, producerOnly)
    if err != nil { return nil, nil, err }

    hub, err := sessionstream.NewHub(
        sessionstream.WithSchemaRegistry(reg),
        sessionstream.WithEventBus(resources.Publisher, resources.SubscriberOrNoop, sessionstream.WithBusTopic(topic)),
    )
    if err != nil { resources.Close(); return nil, nil, err }

    if err := RegisterRuntimeEventCommand(hub); err != nil { ... }
    return &Publisher{hub: hub}, resources, nil
}
```

API hub:

```go
func NewServerHub(ctx context.Context, cfg Config) (*RuntimeEventSessionstream, error) {
    reg, err := NewSchemaRegistry()
    store, err := sqlite.New(fileDSN(cfg.SessionstreamDB), reg)
    wsServer, err := ws.NewServer(snapshotProviderFromHubOrStore)

    hub, err := sessionstream.NewHub(
        sessionstream.WithSchemaRegistry(reg),
        sessionstream.WithHydrationStore(store),
        sessionstream.WithUIFanout(wsServer),
        sessionstream.WithEventBus(resources.Publisher, resources.Subscriber, sessionstream.WithBusTopic(cfg.Topic)),
        sessionstream.WithProjectionErrorPolicy(sessionstream.ProjectionErrorPolicyAdvance),
    )
    register command, UI projection, timeline projection
    go hub.Run(ctx)
    return composite{Hub: hub, WSServer: wsServer, Store: store, Resources: resources}, nil
}
```

Important implementation detail: `ws.NewServer` needs a `SnapshotProvider`. You can adapt the hub:

```go
type snapshotProvider struct { hub *sessionstream.Hub }
func (p snapshotProvider) Snapshot(ctx context.Context, sid sessionstream.SessionId) (sessionstream.Snapshot, error) {
    return p.hub.Snapshot(ctx, sid)
}
```

### API route design

Add a websocket endpoint:

```text
GET /api/v1/runtime-events/sessionstream
```

or a shorter websocket path:

```text
GET /api/v1/runtime-events/ws
```

Recommended route:

```go
mux.Handle("GET /api/v1/runtime-events/ws", runtimeEventStream.WSServer)
```

Remove the old runtime-event routes instead of keeping compatibility endpoints:

- delete `GET /api/v1/runtime-events`;
- delete `GET /api/v1/runtime-events/stream`;
- delete the API-local `runtimeevents.Hub` and `startRuntimeEventRouter` path;
- make the sessionstream websocket endpoint the only live runtime-event distribution API.

If a non-streaming debug view is still useful later, implement it as a new sessionstream snapshot inspection endpoint rather than preserving the old REST contract.

### Frontend design

Generate or import the sessionstream transport protobuf TypeScript code. The repo already uses Buf-generated TypeScript for scraper runtime events under `scraper/web/src/pb/...`. Add generated code for `sessionstream.v1.transport.proto` or a minimal client-side frame codec if codegen is already available in the workspace.

New frontend module:

```text
scraper/web/src/api/sessionstreamRuntimeEventsClient.ts
```

Responsibilities:

- open `new WebSocket(wsUrl('/api/v1/runtime-events/ws'))`;
- send `ClientFrame.subscribe` for a session id;
- parse `ServerFrame.hello`, `snapshot`, `uiEvent`, `error`, `pong`;
- decode `Any` payloads containing `scraper.runtime.sessionstream.v1.RuntimeEventAppended` and `RuntimeEventEntity` wrappers, then read their inner `RuntimeEventV1`;
- merge snapshot entities and live UI events into the RTK Query cache;
- track `snapshotOrdinal` and last live `eventOrdinal` for reconnect.

Pseudo-flow:

```ts
connect(sessionId): RuntimeEventStream {
  socket = new WebSocket(url)

  socket.onopen = () => {
    socket.send(toJson(ClientFrameSchema, {
      subscribe: { sessionId, sinceSnapshotOrdinal: lastSnapshotOrdinal ?? 0n },
    }))
  }

  socket.onmessage = (message) => {
    frame = fromJson(ServerFrameSchema, JSON.parse(message.data))
    switch (frame.frame.case) {
      case 'snapshot':
        events = frame.frame.value.entities
          .filter(e => e.kind === 'scraper.runtime.RuntimeEvent')
          .map(e => unpackRuntimeEventEntity(e.payload).event)
        replaceCacheWithSnapshot(events)
        lastSnapshotOrdinal = frame.frame.value.snapshotOrdinal
        break
      case 'uiEvent':
        if (frame.frame.value.name === 'scraper.runtime.RuntimeEventAppended') {
          upsertRuntimeEvent(unpackRuntimeEventAppended(frame.frame.value.payload).event)
          lastEventOrdinal = frame.frame.value.eventOrdinal
        }
        break
      case 'error':
        reportStreamError(frame.frame.value)
        break
    }
  }
}
```

Session selection in frontend:

```ts
function runtimeEventSession(params: RuntimeEventsParams): string {
  if (params.workflowId) return `workflow:${params.workflowId}`;
  return 'runtime:global';
}
```

This intentionally does not support arbitrary filter combinations in Phase 1. The current pages primarily need global and workflow-scoped feeds. Add op/worker/site sessions when corresponding pages require live subscriptions.

## Implementation phases

### Phase 0: Add dependency and prove compile path

Files likely touched:

- `scraper/go.mod`
- `scraper/go.sum`
- `go.work` only if workspace setup changes, but it already includes `./sessionstream`.

Tasks:

1. Add `github.com/go-go-golems/sessionstream` to `scraper/go.mod`.
2. Run `go mod tidy` in `scraper`.
3. Confirm `go test ./pkg/runtimeevents/...` still passes.

Validation:

```bash
cd scraper
go test ./pkg/runtimeevents/... -count=1
```

### Phase 1: Add scraper sessionstream protobufs and adapter package

Files to add:

```text
scraper/proto/scraper/runtime/sessionstream/v1/runtime_stream.proto
scraper/pkg/runtimeevents/sessionstream/names.go
scraper/pkg/runtimeevents/sessionstream/schema.go
scraper/pkg/runtimeevents/sessionstream/routing.go
scraper/pkg/runtimeevents/sessionstream/projections.go
scraper/pkg/runtimeevents/sessionstream/publisher.go
scraper/pkg/runtimeevents/sessionstream/hub.go
```

Key tests:

```text
scraper/pkg/runtimeevents/sessionstream/routing_test.go
scraper/pkg/runtimeevents/sessionstream/projections_test.go
scraper/pkg/runtimeevents/sessionstream/publisher_test.go
```

Test cases:

- routes every event to `runtime:global`;
- routes workflow events to `workflow:<id>`;
- does not produce duplicate session ids;
- normalizes missing id/schema/timestamp;
- schema registration uses concrete scraper sessionstream protobuf messages;
- UI projection emits exactly one `RuntimeEventAppended` UI event;
- timeline projection creates stable `RuntimeEventEntity` ids;
- local in-memory hub path can submit event and produce snapshot.

### Phase 2: Replace API event hub with sessionstream hub

Files likely touched:

- `scraper/pkg/api/server/server.go`
- `scraper/pkg/api/server/routes_runtime_events.go`
- `scraper/pkg/api/server/runtime_event_router.go` removed.
- `scraper/pkg/api/handlers/runtime_events.go` removed or reduced to non-runtime-event helpers only.
- `scraper/pkg/cmd/api.go` and `scraper/pkg/cmd/runtime_events.go` for new flags.

New API flags:

```text
--events-backend off|gochannel|redis
--events-redis-address ...
--events-sessionstream-db state/runtime-events-sessionstream.db
--events-sessionstream-topic scraper.runtime.v1.sessionstream
--events-recent-limit 500
```

Design note: keep existing event backend names if possible. Operators should not need to learn two separate Redis flag groups.

Server setup pseudocode:

```go
runtimeStream, err := runtimeeventssessionstream.NewServerRuntime(ctx, cfg.RuntimeEvents)
if err != nil { return nil, err }

submissionService := submission.NewService(siteRegistry, runtimeStream.Publisher, metricsRegistry)
submissionHandler := handlers.NewSubmissionHandler(submissionService, cfg.EngineDB, cfg.SitesDir, runtimeStream.Publisher)

mux := http.NewServeMux()
registerRuntimeEventRoutes(mux, runtimeStream.WSServer)
```

### Phase 3: Move worker publication to sessionstream publisher

Files likely touched:

- `scraper/pkg/cmd/worker_runtime.go`
- `scraper/pkg/cmd/worker_observers.go`
- `scraper/pkg/runtimeevents/runner.go`
- `scraper/pkg/runtimeevents/scheduler_observer.go`

Keep the producer-facing type small. Existing producer code should still call something conceptually named `Publish(event)`. The implementation can now submit into sessionstream.

Important test:

- Start API hub and worker publisher against the same gochannel/Redis test backend.
- Publish scheduler and runner events from the worker side.
- Assert API websocket subscriber receives snapshot plus live UI events.

### Phase 4: Add websocket frontend client

Files likely touched:

- `scraper/web/src/api/runtimeEventsApi.ts`
- `scraper/web/src/store/index.ts` if new API slice is added.
- `scraper/web/src/pages/RuntimeEventsPage.tsx`
- `scraper/web/src/pages/WorkflowDetailPage.tsx`
- `scraper/web/src/components/workflows/RuntimeEventTable.tsx` likely unchanged.
- generated protobuf files for sessionstream transport.

Implementation approach:

1. Keep `RuntimeEventV1` decoding helpers.
2. Add a websocket stream helper that returns decoded `RuntimeEventV1[]` updates.
3. Make RTK Query use websocket when available.
4. Remove the REST initial fetch and rely on sessionstream snapshots for initial state.
5. Remove EventSource/SSE code entirely.

### Phase 5: Retention, observability, and cleanup

Add:

- retention/tombstone behavior in timeline projection;
- metrics for websocket connections, subscriptions, events fanned out, and projection errors;
- logs/diagnostics via sessionstream observers;
- migration docs for operators;
- delete any remaining old `runtimeevents.Hub`, REST, SSE, and protobuf-byte Watermill codec code not used by the sessionstream path.

## API reference summary

### Go: scraper-facing publisher

```go
type RuntimeEventPublisher interface {
    Publish(ctx context.Context, event *runtimev1.RuntimeEventV1) error
}
```

### Go: session routing

```go
const SessionRuntimeGlobal = sessionstream.SessionId("runtime:global")

func WorkflowSessionID(workflowID string) sessionstream.SessionId
func RuntimeEventSessionIDs(event *runtimev1.RuntimeEventV1) []sessionstream.SessionId
```

### Go: hub setup

```go
type ServerRuntime struct {
    Hub       *sessionstream.Hub
    WSServer  http.Handler
    Publisher *Publisher
    Close     func() error
}

func NewServerRuntime(ctx context.Context, cfg Config) (*ServerRuntime, error)
func NewProducerRuntime(cfg Config) (*Publisher, io.Closer, error)
```

### HTTP/websocket

```text
GET /api/v1/runtime-events/ws
```

Client sends:

```json
{"subscribe":{"sessionId":"workflow:wf-123","sinceSnapshotOrdinal":"0"}}
```

Server sends, in order:

```json
{"hello":{"connectionId":"conn-1"}}
{"snapshot":{"sessionId":"workflow:wf-123","snapshotOrdinal":"42","entities":[...]}}
{"subscribed":{"sessionId":"workflow:wf-123","sinceSnapshotOrdinal":"0"}}
{"uiEvent":{"sessionId":"workflow:wf-123","eventOrdinal":"43","name":"scraper.runtime.RuntimeEventAppended","payload":{...}}}
```

Exact JSON shape depends on the protobuf JSON codec used by the generated TypeScript and Go code. Use generated schemas rather than hand-written string manipulation.

## Testing strategy

### Unit tests

Run from `scraper`:

```bash
go test ./pkg/runtimeevents/... -count=1
```

Add tests for:

- routing;
- schema registration;
- command handler type checking;
- projection output;
- timeline entity ids;
- publisher duplicates and error joining.

### Integration tests

Add an API/server test that:

1. starts the API server with gochannel sessionstream backend;
2. opens a websocket client;
3. subscribes to `runtime:global`;
4. submits a workflow or directly publishes a `RuntimeEventV1`;
5. asserts the client receives a snapshot and a live runtime UI event;
6. reconnects with `sinceSnapshotOrdinal` and asserts snapshot hydration works.

### End-to-end smoke test

Manual command flow:

```bash
cd scraper
SCRAPER_SITES_MANIFEST_DIRS=./sites go run ./cmd/scraper api serve \
  --address 127.0.0.1:8080 \
  --events-backend redis \
  --events-redis-address 127.0.0.1:6379 \
  --events-sessionstream-db state/runtime-events-sessionstream.db
```

In a second terminal:

```bash
cd scraper
SCRAPER_SITES_MANIFEST_DIRS=./sites go run ./cmd/scraper worker run \
  --events-backend redis \
  --events-redis-address 127.0.0.1:6379 \
  --max-cycles 16 \
  --poll-interval 10ms
```

In the browser:

- open Runtime Events page;
- confirm `runtime:global` snapshot loads;
- submit a site workflow;
- confirm workflow page subscribes to `workflow:<id>` and receives scheduler/runner progress.

### Frontend tests

Run from `scraper/web`:

```bash
pnpm test
pnpm storybook
```

Add tests for:

- frame decoding;
- snapshot-to-event-list conversion;
- live UI event insertion and deduplication by runtime event id;
- reconnect state preserving `snapshotOrdinal`;
- websocket close/error handling without falling back to REST/SSE.

## Risks and mitigations

### Risk: event duplication across sessions

Publishing the same runtime event to `runtime:global` and `workflow:<id>` means it receives different session ordinals. This is acceptable if the event id remains stable and the frontend deduplicates by `RuntimeEventV1.id` within one feed. Do not compare ordinals across sessions.

### Risk: unbounded timeline growth

The current in-memory scraper hub has bounded retention. A sessionstream SQLite timeline can grow indefinitely unless the projection tombstones old entities or a cleanup job prunes old event rows. Implement retention before enabling long-running production use.

### Risk: Redis ordering semantics

Sessionstream expects per-session ordering. Its bus metadata includes a partition key helper (`sessionstream.PartitionKeyForSession`). Verify Redis Streams preserves enough order for each session, especially when publishing duplicates to multiple sessions. If necessary, use one stream topic and a consumer group per API process, but keep only one active consumer for a given deployment until parallel consumption semantics are deliberately tested.

### Risk: frontend payload unpacking

Sessionstream frames use `google.protobuf.Any`. The frontend must correctly unpack `RuntimeEventV1` payloads. Avoid manual `typeUrl` string hacks where possible; use generated protobuf support or a small typed unpacking helper with tests.

### Risk: larger breaking change surface

Removing REST/SSE immediately means tests, frontend code, and operator habits must move in the same implementation series. Mitigate this with a focused branch, explicit deletion commits, and end-to-end websocket tests that prove snapshot hydration and live fanout before merging.

## Alternatives considered

### Alternative 1: Keep existing SSE hub

This is the smallest change but does not satisfy the goal of using sessionstream as the core distribution mechanism. It leaves duplicate buffering, no session snapshots, and ad-hoc reconnect semantics.

### Alternative 2: Replace only frontend SSE with a custom scraper websocket

This would solve bidirectional transport but duplicate sessionstream's websocket and hydration logic. It is not recommended.

### Alternative 3: Publish raw `RuntimeEventV1` directly to sessionstream bus without commands

The unexported sessionstream publisher path is intentionally behind command handlers and `EventPublisher`. Using `Hub.Submit` with a small command handler stays inside the public API and exercises schema validation.

### Alternative 4: One session per arbitrary filter combination

Creating session ids for every query filter combination would be hard to reason about and likely wasteful. Prefer a small, stable session vocabulary and add new sessions only when a UI surface needs them.

## File-by-file implementation map

| File | Change |
| --- | --- |
| `scraper/go.mod` | Add `github.com/go-go-golems/sessionstream`. |
| `scraper/proto/scraper/runtime/sessionstream/v1/runtime_stream.proto` | New scraper sessionstream command/event/UI/entity protobuf contracts. |
| `scraper/pkg/runtimeevents/sessionstream/*.go` | New adapter package for schema, routing, projections, publisher, and hub setup. |
| `scraper/pkg/runtimeevents/runner.go` | Keep event construction; call the new context-aware publisher interface. |
| `scraper/pkg/runtimeevents/scheduler_observer.go` | Keep scheduler mapping; publish through sessionstream adapter. |
| `scraper/pkg/runtimeevents/hub.go` | Delete the old in-memory recent-event/subscriber hub. |
| `scraper/pkg/runtimeevents/watermill.go` | Delete or replace the old protobuf-byte Watermill codec with sessionstream bus publication. |
| `scraper/pkg/cmd/runtime_events.go` | Keep only flags needed for sessionstream bus/store configuration. |
| `scraper/pkg/cmd/worker_runtime.go` | Build producer-side sessionstream publisher. |
| `scraper/pkg/api/server/server.go` | Build API-side sessionstream hub, websocket server, and publisher. |
| `scraper/pkg/api/server/runtime_event_router.go` | Delete old router that copied Watermill messages into `runtimeevents.Hub`. |
| `scraper/pkg/api/server/routes_runtime_events.go` | Register only the sessionstream websocket runtime-events route. |
| `scraper/pkg/api/handlers/runtime_events.go` | Delete the old REST/SSE runtime event handler. |
| `scraper/web/src/api/runtimeEventsApi.ts` | Remove EventSource and REST initial fetch; use sessionstream websocket snapshots and UI events. |
| `scraper/web/src/pb/...` | Add generated sessionstream transport and scraper runtime-stream protobuf TypeScript files. |

## Reading path for a new intern

Read in this order:

1. `scraper/proto/scraper/runtime/v1/events.proto` to understand the event payload.
2. `scraper/pkg/runtimeevents/runner.go` and `scheduler_observer.go` to see producers.
3. `scraper/pkg/api/server/server.go` to see how runtime event infrastructure is currently assembled.
4. `scraper/pkg/api/handlers/runtime_events.go` and `scraper/web/src/api/runtimeEventsApi.ts` to see current REST/SSE behavior.
5. `sessionstream/pkg/sessionstream/types.go`, `handler.go`, and `hub.go` to understand command/event flow.
6. `sessionstream/pkg/sessionstream/bus.go` and `consumer.go` to understand Watermill integration.
7. `sessionstream/pkg/sessionstream/transport/ws/server.go` and `sessionstream/proto/sessionstream/v1/transport.proto` to understand websocket frames and hydration.
8. Implement the adapter package with tests before touching API/server or frontend code.

## Implementation notes from completed work

The implementation completed the breaking migration in three code commits:

1. `0ea7c29071279544366f5878edf34ac79c63d0db` — added scraper sessionstream protobuf contracts, generated Go/TypeScript bindings, and the `pkg/runtimeevents/sessionstream` adapter package.
2. `ee5f4ba936ee0f5ce49d7d9f7d988855518ae567` — replaced the backend REST/SSE runtime-event infrastructure with the sessionstream runtime and websocket endpoint.
3. `d00312f93f504427fd381e5a9d4dc5f50bdd102d` — moved the frontend runtime-event RTK Query endpoint from REST/EventSource to sessionstream websocket snapshots and live UI events.

The implemented backend endpoint is now:

```text
GET /api/v1/runtime-events/ws
```

The old backend routes are gone:

```text
GET /api/v1/runtime-events
GET /api/v1/runtime-events/stream
```

The implemented scraper-specific sessionstream protobuf file is:

```text
proto/scraper/runtime/sessionstream/v1/runtime_stream.proto
```

It defines wrapper messages around `RuntimeEventV1` for the first implementation. This keeps scraper's current runtime event schema intact while giving sessionstream separate typed contracts for commands, backend events, UI events, and timeline entities.

Important implementation files added or changed:

- `pkg/runtimeevents/sessionstream/names.go`
- `pkg/runtimeevents/sessionstream/schema.go`
- `pkg/runtimeevents/sessionstream/routing.go`
- `pkg/runtimeevents/sessionstream/publisher.go`
- `pkg/runtimeevents/sessionstream/projections.go`
- `pkg/runtimeevents/sessionstream/runtime.go`
- `pkg/runtimeevents/publisher.go`
- `pkg/api/server/server.go`
- `pkg/api/server/routes_runtime_events.go`
- `pkg/api/server/middleware_request.go`
- `pkg/api/server/server_test.go`
- `web/src/api/runtimeEventsApi.ts`

Implementation caveats:

- `pkg/api/server/middleware_request.go` bypasses the normal status-recorder wrapper for websocket upgrade requests because Gorilla websocket needs the underlying response writer's hijacking support.
- `pkg/cmd/app_config.go` now performs local app config path discovery instead of calling the missing `glazed/pkg/config.ResolveAppConfigPath` symbol in the current workspace state.
- The frontend currently hand-parses the small subset of `sessionstream.v1.ServerFrame` JSON it needs, while using generated protobuf TypeScript bindings for scraper-specific `RuntimeEventAppended` and `RuntimeEventEntity` payloads.
- Full `pnpm build` still fails on pre-existing TypeScript/story issues unrelated to the runtime-event websocket migration; `pnpm test:unit -- --runInBand` passes.

## Open questions

1. What retention policy is correct for `runtime:global`: fixed count, fixed age, or both?
2. Should op-, worker-, and site-scoped sessions be enabled in Phase 1 or deferred until there are UI pages that need them?
3. Should the sessionstream SQLite DB live beside the engine DB (`state/runtime-events-sessionstream.db`) or inside the engine DB as additional tables? The standalone DB is lower-risk initially.
4. How should authentication/authorization affect websocket subscriptions once the API is exposed beyond local development? The sessionstream websocket default origin policy is intentionally permissive for examples.
5. Should scraper split `RuntimeEventV1` into narrower protobuf event messages in the first implementation, or start with wrapper messages around `RuntimeEventV1` and split after the websocket path is stable?

## References

### Primary source files

- `sessionstream/pkg/sessionstream/types.go`
- `sessionstream/pkg/sessionstream/hub.go`
- `sessionstream/pkg/sessionstream/bus.go`
- `sessionstream/pkg/sessionstream/hydration.go`
- `sessionstream/pkg/sessionstream/hydration/sqlite/store.go`
- `sessionstream/pkg/sessionstream/transport/ws/server.go`
- `sessionstream/pkg/sessionstream/transport/ws/observer.go`
- `sessionstream/proto/sessionstream/v1/transport.proto`
- `scraper/proto/scraper/runtime/v1/events.proto`
- `scraper/pkg/runtimeevents/backend.go`
- `scraper/pkg/runtimeevents/hub.go`
- `scraper/pkg/runtimeevents/watermill.go`
- `scraper/pkg/runtimeevents/runner.go`
- `scraper/pkg/runtimeevents/scheduler_observer.go`
- `scraper/pkg/api/server/server.go`
- `scraper/pkg/api/server/runtime_event_router.go`
- `scraper/pkg/api/handlers/runtime_events.go`
- `scraper/pkg/cmd/runtime_events.go`
- `scraper/pkg/cmd/worker_runtime.go`
- `scraper/web/src/api/runtimeEventsApi.ts`
- `/home/manuel/code/wesen/go-go-golems/pinocchio/proto/pinocchio/chatapp/v1/chat.proto`
- `/home/manuel/code/wesen/go-go-golems/pinocchio/pkg/chatapp/chat.go`
- `/home/manuel/code/wesen/go-go-golems/pinocchio/pkg/chatapp/runtime_inference.go`
- `/home/manuel/code/wesen/go-go-golems/pinocchio/pkg/chatapp/projections.go`
- `/home/manuel/code/wesen/go-go-golems/pinocchio/pkg/chatapp/runner.go`
- `/home/manuel/code/wesen/go-go-golems/pinocchio/pkg/chatapp/service.go`

### Investigation artifacts in this ticket

- `sources/01-file-inventory.txt` records the scanned file inventory.
- `sources/02-key-symbol-search.txt` records symbol search results for runtime events, websocket, observers, and sessionstream.
- `sources/03-line-referenced-excerpts.txt` contains line-numbered excerpts of the files used for this design.
