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
