# Tasks

## Review And Documentation

- [x] Create the `SCRAPER-PROMETHEUS-METRICS` ticket workspace.
- [x] Review current server, worker, engine, runtime-event, and frontend observability seams.
- [x] Write a detailed design and implementation guide for adding Prometheus and proper operator metrics.
- [x] Record the investigation and conclusions in the diary.

## Architecture Decisions To Confirm

- [x] Confirm that Prometheus will be used for numeric time-series and alerting rather than building a custom in-app metrics store.
- [x] Confirm that scraper remains the source of truth for workflows, ops, dependencies, artifacts, and runtime-event detail.
- [x] Confirm that the worker process should expose its own `/metrics` endpoint through a configurable metrics HTTP listener.
- [x] Confirm whether Grafana is the first-class historical operator UI, with scraper frontend staying focused on current state and debugging detail.

## Implementation Backlog

### Phase 1: Shared Metrics Package

- [x] Add Prometheus client dependencies to the top-level Go module.
- [x] Create `pkg/metrics/` with a central registry/bootstrap API.
- [x] Define shared metric names, label names, and helper functions for status-class mapping.
- [ ] Split metrics into logical groups:
  - [x] API server metrics
  - [x] submission metrics
  - [x] scheduler metrics
  - [x] runner metrics
  - [x] snapshot/export gauges
- [x] Decide whether to use the default Prometheus registry or an explicit custom registry.
- [x] Add unit tests for registry creation and duplicate registration safety.

### Phase 2: API Server `/metrics`

- [x] Add a Prometheus handler to the server mux in `pkg/api/server/server.go`.
- [x] Add HTTP request counter instrumentation in the existing request wrapper.
- [x] Add HTTP request duration histograms in the existing request wrapper.
- [x] Use route-pattern labels rather than raw URL paths.
- [x] Confirm `/healthz` and `/metrics` are both accessible without interfering with current API routes.
- [ ] Add server tests covering:
  - [x] `/metrics` returns `200`
  - [x] scraper metric families are present
  - [x] request metrics increment after serving API requests

### Phase 3: Snapshot Collectors

- [x] Implement a collector that exports engine snapshot gauges from `engineview.Service` and/or SQLite inspection helpers.
- [ ] Export workflow counts by status.
- [ ] Export queue gauges:
  - [x] pending
  - [x] ready
  - [x] running
  - [x] in-flight
  - [x] tokens
- [x] Export artifact/result/lease gauges where useful.
- [x] Decide whether snapshot gauges live only on the API server or also on workers.
- [ ] Add tests for scrape-time collection behavior.

### Phase 4: Worker Metrics Listener

- [x] Add worker flags for metrics exposure:
  - [x] `--metrics-address`
  - [x] optional `--metrics-path`
- [x] Start a small HTTP server from the worker process when metrics are enabled.
- [x] Ensure worker shutdown also closes the metrics listener cleanly.
- [x] Export worker liveness/process metrics.
- [x] Add integration or smoke tests showing the worker exposes `/metrics`.

### Phase 5: Submission Metrics

- [x] Instrument successful workflow submissions in `pkg/services/submission/service.go`.
- [x] Instrument submission failures by stable error code/category.
- [ ] Add optional submission duration histograms if the path is expensive enough to justify them.
- [ ] Add tests proving submission counters move after accepted and rejected submissions.

### Phase 6: Scheduler Metrics

- [x] Instrument scheduler cycle counters.
- [x] Instrument scheduler cycle duration histograms.
- [x] Instrument leased-op counters by site, queue, and runner kind where available.
- [x] Instrument retry counters.
- [ ] Instrument failure counters by stable error code/category.
- [x] Instrument queue-rate-limited counters.
- [ ] Decide how queue wait time should be measured, then add a queue-wait histogram.
- [x] Keep idle/no-work polling out of high-volume metrics noise.
- [ ] Add tests or focused package-level assertions for scheduler metric updates.

### Phase 7: Runner Metrics

- [x] Instrument generic op execution duration by site, queue, runner, and terminal status.
- [x] Instrument HTTP runner request counts in `pkg/engine/runner/http.go`.
- [x] Instrument HTTP runner duration histograms in `pkg/engine/runner/http.go`.
- [x] Classify HTTP statuses by stable status classes rather than raw URLs or messages.
- [x] Classify transport errors separately from HTTP response errors.
- [ ] Add tests for HTTP runner metric emission on:
  - [ ] success
  - [ ] transport error
  - [ ] retryable HTTP failure
  - [ ] non-retryable HTTP failure

### Phase 8: Local Prometheus And Grafana Stack

- [x] Extend `docker-compose.yml` with Prometheus.
- [x] Extend `docker-compose.yml` with Grafana.
- [x] Add a Prometheus scrape config covering API server and worker targets.
- [x] Add a starter Grafana dashboard JSON or provisioning bundle.
- [x] Add local docs/runbook for:
  - [x] starting API + worker + Prometheus + Grafana
  - [x] checking target health
  - [x] submitting demo workflows
  - [x] verifying dashboard movement

### Phase 9: Frontend Operator Surfaces

- [ ] Remove or clearly gate the placeholder throughput data in `web/src/pages/QueueMonitorPage.tsx`.
- [ ] Decide whether the scraper frontend should:
  - [ ] link out to Grafana only
  - [ ] embed Grafana panels
  - [ ] proxy a curated set of Prometheus queries through the scraper API
- [ ] Update queue monitor copy so it distinguishes current queue state from historical metrics.
- [ ] Update overview copy so operators understand where snapshot data ends and metrics history begins.
- [ ] Add any required frontend cards or links for “View historical metrics”.

### Phase 10: Alerting And Recording Rules

- [ ] Add recording rules for common operator aggregations.
- [ ] Add alerts for:
  - [ ] worker down
  - [ ] API target down
  - [ ] sustained queue throttling
  - [ ] high failure rate by site/queue
  - [ ] excessive queue wait time
  - [ ] retry spikes
- [ ] Document the expected operator response for each alert.

## Cross-Cutting Review Checks

- [x] Verify that no metric uses high-cardinality labels such as `workflow_id`, `op_id`, or `request_id`.
- [x] Verify that queue, site, verb, runner, and status labels are bounded and intentional.
- [ ] Verify that runtime events and metrics are complementary rather than duplicative.
- [x] Verify that sensitive proxy details are not accidentally exported as metrics or labels.
- [x] Verify that the scraper frontend still uses scraper APIs for object-level debugging rather than trying to use Prometheus as a debugging database.

## Validation Plan

- [x] `go test ./... -count=1`
- [x] Add focused package tests under `pkg/metrics/...`
- [x] Add API/server metrics tests
- [x] Add worker metrics smoke test
- [x] Add local compose smoke test for Prometheus scrape health
- [x] Add local Grafana smoke test for dashboard visibility

## Follow-On Tickets To Consider

- [ ] Separate ticket for implementing the shared Go metrics package and server/worker exposure.
- [ ] Separate ticket for local Prometheus/Grafana compose and runbooks.
- [ ] Separate ticket for frontend historical metrics surfaces and Grafana integration.
- [ ] Separate ticket for alerts, recording rules, and operations playbooks.

## Validation And Publishing

- [x] Run `docmgr doctor --ticket SCRAPER-PROMETHEUS-METRICS --stale-after 30`.
- [x] Upload the bundled ticket docs to reMarkable.
