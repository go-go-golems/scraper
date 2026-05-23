# Tasks

## Completed setup/documentation

- [x] Create ticket workspace and initial docs
- [x] Gather file inventory and line-referenced evidence from scraper and sessionstream
- [x] Write intern-oriented design and implementation guide
- [x] Write investigation diary
- [x] Run docmgr validation
- [x] Upload bundled docs to reMarkable

## Implementation phases

### Phase 1 — Define scraper sessionstream protobuf contracts and adapter core

- [x] Add `proto/scraper/runtime/sessionstream/v1/runtime_stream.proto` with command/event/UI/entity wrapper messages around `RuntimeEventV1`.
- [x] Regenerate Go and TypeScript protobuf bindings with `buf generate`.
- [x] Add `github.com/go-go-golems/sessionstream` to scraper dependencies.
- [x] Add `pkg/runtimeevents/sessionstream` adapter package with names, schema registration, session routing, publisher, projections, and server/producer hub wiring.
- [x] Add unit tests for schema registration, routing, projection output, local hub snapshots, and Watermill/gochannel fanout.
- [x] Validate Phase 1 with `go test ./pkg/runtimeevents/... -count=1`.
- [x] Commit Phase 1.

### Phase 2 — Replace backend runtime-event infrastructure with sessionstream

- [x] Update API server construction to create the sessionstream runtime, register only websocket runtime-event routes, and close hub/resources on shutdown.
- [x] Update worker, submit-verb, submission, request middleware, and runner/scheduler producers to use the context-aware sessionstream publisher.
- [x] Delete old in-memory runtime event hub, old runtime event REST/SSE handler, and old runtime event router.
- [x] Replace or remove old Watermill protobuf-byte runtime event codec tests.
- [x] Add/adjust API integration tests for websocket snapshot and live runtime-event delivery.
- [x] Validate Phase 2 with focused Go tests.
- [x] Commit Phase 2.

### Phase 3 — Move frontend runtime-event feed to sessionstream websocket only

- [x] Replace `EventSource`/REST runtime event API usage with a sessionstream websocket client.
- [x] Decode `sessionstream.v1.ServerFrame` and scraper `RuntimeEventAppended`/`RuntimeEventEntity` protobuf `Any` payloads in TypeScript.
- [x] Update global runtime events, workflow detail, and op runtime tab consumers to use websocket snapshots/live UI events.
- [x] Remove MSW/runtime-events REST mocks or rewrite them around websocket-independent component stories/tests.
- [x] Validate Phase 3 with frontend tests/build.
- [x] Commit Phase 3.

### Phase 4 — Final validation, docs, diary, and reMarkable refresh

- [x] Run `go test ./... -count=1` in `scraper` or document any pre-existing failures.
- [x] Run frontend validation (`pnpm test`/build) or document any pre-existing failures.
- [x] Update the design guide with implementation notes and final file references.
- [x] Update the diary with each implementation step, commands, failures, commits, and review instructions.
- [x] Run `docmgr doctor --ticket SCRAPER-SESSIONSTREAM-EVENTS --stale-after 30`.
- [x] Upload the final bundle to reMarkable.
- [x] Commit final docs.
