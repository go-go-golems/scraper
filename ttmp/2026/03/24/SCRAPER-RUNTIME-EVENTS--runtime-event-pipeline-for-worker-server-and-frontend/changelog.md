# Changelog

## 2026-03-24

- Initial workspace created
- Inspected the existing event and process boundaries in `pkg/engine/scheduler/scheduler.go`, `pkg/cmd/worker.go`, `pkg/api/server/server.go`, `pkg/services/submission/service.go`, and `pkg/sites/submitverbs/host.go`
- Documented the tradeoffs between four approaches:
  - keep polling and persist events locally
  - merge worker and server temporarily with in-process pub/sub
  - keep the split and bridge over Redis
  - use a hybrid abstraction with an optional single-process mode
- Recommended keeping worker and server as separate deployment roles, introducing a shared event envelope plus sink/source abstraction, and using Redis as the main cross-process transport instead of permanently collapsing the process model
- Follow-up decision: use Watermill as the standard eventing layer, with Redis-backed transport for cross-process delivery and in-process Watermill transport for tests and optional local mode
- Follow-up decision: define the event contract in protobuf and generate both Go and TypeScript types, using Buf-managed codegen and `protojson` at the web boundary
- Implemented the Phase 1 scaffold in commit `448d450050ae6ea9e0880b44b6f3cf1a176d0db1`:
  - added `proto/scraper/runtime/v1/events.proto`
  - added `buf.yaml` and `buf.gen.yaml`
  - generated Go and web protobuf artifacts
  - added `pkg/runtimeevents` codec helpers and round-trip tests
  - added the web protobuf runtime dependency
- Implemented the first Watermill integration slice in commit `33a9073faa7856fabe93ba55489d171f22609b53`:
  - chose protobuf binary as the internal Watermill payload format
  - standardized topic and message metadata conventions
  - added `pkg/runtimeevents/watermill.go`
  - added a GoChannel-backed integration test for publish/subscribe round-trips
- Implemented the scheduler-to-runtime-event adapter in commit `a217cbf47847e30f634fa2a142e414ed82f417ef`:
  - added `pkg/runtimeevents/scheduler.go`
  - mapped scheduler event kinds and severity into `RuntimeEventV1`
  - added scheduler payload extraction for attempt, workflow status, and error details
  - added tests covering failure and idle event mapping
- Implemented the end-to-end backend runtime event pipeline in commit `5e2808cf36c5cb302f6ecfa25d1d0c3278d4da8b`:
  - added configurable runtime event backends with `off`, `gochannel`, and Redis-backed Watermill transports
  - added a recent-event hub, scheduler observer publisher, and runner log emission wrapper
  - wired worker, submission, and HTTP request paths to publish runtime events
  - added API-side history and SSE delivery over a Watermill subscriber/router
  - added a local Redis `docker-compose.yml`
  - added a Go integration test that proves submission -> worker -> API event flow with a shared GoChannel backend
- Implemented the frontend runtime event view in commit `47c2c55d6ed2aa16b0971f4c438490c24a903790`:
  - added frontend protobuf JSON decoding with `fromJson`
  - added runtime event history fetching and SSE stream consumption
  - added a runtime event timeline card to the workflow detail page
  - fixed the `OpDetailDrawer` script-tab prop mismatch so `npm run build` passes again

- Ticket administratively closed on 2026-04-07 and retained as historical context; follow-on work should use newer focused tickets where they exist.
