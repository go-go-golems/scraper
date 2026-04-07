# Tasks

## Review And Documentation

- [x] Create the `SCRAPER-PROMETHEUS-API` ticket workspace.
- [x] Review the current Prometheus, Grafana, API-server, and frontend seams relevant to historical graph data.
- [x] Write a detailed design and implementation guide for scraper-owned Prometheus-backed graph endpoints.
- [x] Add a follow-on design doc mapping current UI components to the first graphs we can ship.
- [x] Record the investigation process and findings in the diary.
- [x] Publish and validate the completed ticket bundle.

## Architecture Decisions To Confirm

- [x] Confirm that the browser should not query Prometheus directly.
- [x] Confirm that Grafana remains useful for operators, but scraper needs its own graph API for integrated product UX.
- [x] Confirm that scraper should expose only a curated metrics surface, not arbitrary PromQL passthrough.
- [x] Confirm that the graph API should return normalized JSON shapes tailored to web UI needs.
- [x] Confirm that current object state continues to come from the existing engine/catalog/runtime-event APIs.

## Proposed Backend Implementation Backlog

### Phase 1: Prometheus Client Service

- [ ] Add a small Prometheus HTTP client package under `pkg/services/metricsapi/` or similar.
- [ ] Load Prometheus base URL and timeout from explicit server config, not ad hoc environment reads.
- [ ] Add a thin query executor for:
  - [ ] instant queries
  - [ ] range queries
  - [ ] result decoding
  - [ ] timeout and HTTP error handling
- [ ] Normalize Prometheus API failures into stable scraper error categories.
- [ ] Add unit tests with canned Prometheus API responses.

### Phase 2: Query Registry And Safety Model

- [ ] Define a fixed query registry of supported graph kinds.
- [ ] Map user-facing graph IDs to internal PromQL templates.
- [ ] Keep query parameters bounded to:
  - [ ] time range
  - [ ] resolution step
  - [ ] site
  - [ ] queue
  - [ ] runner
- [ ] Reject unsupported labels and arbitrary PromQL input.
- [ ] Add tests proving unsupported graph IDs and label sets are rejected.

### Phase 3: API DTOs And Handlers

- [ ] Add dedicated handler(s) under `pkg/api/handlers/` for metrics graph endpoints.
- [ ] Define response DTOs for:
  - [ ] overview cards
  - [ ] queue history
  - [ ] site history
  - [ ] worker health history
- [ ] Add clear endpoint-level validation for time range and downsampling step.
- [ ] Wire routes in `pkg/api/server/server.go`.
- [ ] Add handler tests for success, validation failures, and Prometheus-upstream failures.

### Phase 4: Suggested Initial Endpoints

- [ ] `GET /api/v1/metrics/overview`
- [ ] `GET /api/v1/metrics/queues/{site}/{queue}/history`
- [ ] `GET /api/v1/metrics/sites/{site}/history`
- [ ] `GET /api/v1/metrics/workers/history`
- [ ] `GET /api/v1/metrics/metadata`

### Phase 5: Integration Validation

- [ ] Add an integration-style test using a fake Prometheus server.
- [ ] Add a local smoke test against the existing compose Prometheus service.
- [ ] Verify that queue-monitor placeholder throughput can be replaced by real endpoint output in a follow-on frontend ticket.
- [ ] Verify that the API surface does not leak internal PromQL details into the frontend contract.

## Open Product Follow-Ons

- [ ] Separate implementation ticket for backend Prometheus graph APIs.
- [ ] Separate frontend ticket for consuming the new graph endpoints in overview and queue pages.
- [ ] Separate dashboard alignment ticket so scraper cards and Grafana panels use the same underlying aggregations.

## Current UI Graph Rollout Tasks

### Phase A: Queue Monitor First

- [ ] Replace `placeholderThroughput` in `web/src/pages/QueueMonitorPage.tsx` with real backend data.
- [ ] Feed real queue completion-rate series into `web/src/components/queues/ThroughputChart.tsx`.
- [ ] Add ready-depth history to the queue page.
- [ ] Add queue wait p95 history to the queue page.
- [ ] Decide whether the first queue page uses one chart with multiple lines or separate cards per graph family.

### Phase B: Overview Cards And Trends

- [ ] Extend `web/src/components/overview/StatCardRow.tsx` to show metrics-backed cards for:
  - [ ] workers up
  - [ ] active workflows
  - [ ] worst queue wait p95
- [ ] Add one compact overview trend card for workflow submit rate.
- [ ] Add one compact overview trend card for aggregate op completion rate.

### Phase C: Deeper Queue Diagnostics

- [ ] Add retry-rate trend graph to queue detail surfaces.
- [ ] Add failure-rate trend graph to queue detail surfaces.
- [ ] Add throttle-rate trend graph to queue detail surfaces.
- [ ] Add running vs in-flight depth trends if operators still need more capacity diagnostics after Phase A.

## Validation And Publishing

- [x] Run `docmgr doctor --ticket SCRAPER-PROMETHEUS-API --stale-after 30`.
- [x] Upload the bundled ticket docs to reMarkable.
- [ ] Commit the doc slice.
