# Tasks

## Phase 0: Research and Direction

- [x] Create the ticket workspace and investigation diary
- [x] Inspect the current worker, scheduler, submission, and HTTP server seams
- [x] Compare option families: SQLite polling, merged worker+server, Redis transport, and hybrid abstraction
- [x] Write a recommendation before the implementation guide
- [x] Decide to use Watermill as the standard event transport layer
- [x] Decide to use protobuf as the shared Go/TS event contract

## Phase 1: Schema and Codegen Foundation

- [x] Define an initial `RuntimeEventV1` protobuf envelope and source taxonomy
- [x] Add `proto/`, `buf.yaml`, and `buf.gen.yaml` for Go + TS generation
- [x] Define stable protobuf package names, `go_package`, and schema versioning rules
- [x] Generate Go code under `gen/proto/...`
- [x] Generate web artifacts under `web/src/pb/...`
- [x] Add TS runtime/codegen dependency for protobuf decoding in `web/`
- [x] Add Go helpers for binary and `protojson` encode/decode of `RuntimeEventV1`
- [x] Add a small Go validation test around the generated event envelope
- [x] Decide Watermill wire format: protobuf binary internally, protojson at the HTTP/SSE boundary
- [ ] Add a small TS decode example or test using `fromJson`

## Phase 2: Watermill Contract and Topics

- [x] Define Watermill topic names and message metadata conventions
- [ ] Decide the delivery contract for replayable events vs best-effort transient events
- [x] Add a thin scraper-owned Watermill wrapper usable by both worker and server codepaths
- [x] Add a GoChannel backend for tests and optional local single-process mode
- [ ] Add Watermill Redis-backed transport for cross-process delivery
- [ ] Add Docker Compose for local Redis development

## Phase 3: Worker and Submission Emission

- [ ] Add worker-side event emission from the scheduler observer
- [ ] Map scheduler events into generated `RuntimeEventV1` messages
- [ ] Add runner/log event emission surfaces where worthwhile
- [ ] Add server-side event emission for submission lifecycle visibility
- [ ] Stop dropping submission-time workflow events on the `observer == nil` path

## Phase 4: Server Consumption and Frontend Delivery

- [ ] Add a server-side Watermill subscriber/router for runtime events
- [ ] Add a bounded recent-event buffer in the API server
- [ ] Add server-side event history endpoint
- [ ] Add server-side SSE endpoint
- [ ] Decide how log payloads are stored: inline event payloads, artifacts, filesystem logs, or a mix
- [ ] Add frontend decoding and state wiring for generated event messages

## Phase 5: Validation and Ops

- [ ] Add integration tests that prove end-to-end event flow from submission through worker execution to frontend consumption
- [ ] Add a local single-process `server+worker` mode only if still useful after Redis-backed mode exists
- [ ] Document operator workflows, message topics, and failure modes
