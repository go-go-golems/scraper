# Changelog

## 2026-04-07

- Created the `SCRAPER-PROMETHEUS-METRICS` ticket and scaffold documents.
- Reviewed current observability-related code in the API server, worker, engine snapshot service, runtime-events pipeline, HTTP runner, and frontend queue/overview pages.
- Confirmed that scraper currently has durable snapshots and live runtime events, but no Prometheus metrics, no `/metrics` endpoint, and no trustworthy throughput history in the UI.
- Identified the main architectural split: scraper should keep owning durable workflow/op state, while Prometheus should own numeric time-series, rates, latency distributions, and alerting.
- Wrote the main design and implementation guide, including metric taxonomy, label rules, endpoint topology, alternatives, and phased rollout.
- Related the design doc to the main server, worker, engine, runner, and queue UI files.
- Ran `docmgr doctor --ticket SCRAPER-PROMETHEUS-METRICS --stale-after 30` successfully.
- Uploaded the bundled report to reMarkable at `/ai/2026/04/07/SCRAPER-PROMETHEUS-METRICS`.
- Implemented the first backend metrics slice:
  - added `pkg/metrics` with an explicit Prometheus registry, request/submission/scheduler/runner metrics, and snapshot collectors,
  - exposed `/metrics` on the API server,
  - added worker `--metrics-address` and `--metrics-path` flags plus a scrapeable worker metrics listener,
  - instrumented submission, scheduler, and runner paths,
  - added focused metrics tests and `/metrics` handler coverage.
- Deferred queue wait histograms, failure counters by stable error code, and compose/Grafana wiring to the next slice.
- Implemented the local monitoring stack:
  - extended `docker-compose.yml` with Prometheus and Grafana,
  - added Prometheus scrape config targeting local API and worker metrics,
  - provisioned a Grafana Prometheus datasource and starter `Scraper Overview` dashboard,
  - added a ticket playbook for the local smoke procedure.
- Verified the full local smoke flow with tmux-managed API and worker processes, Prometheus targets `up`, Grafana datasource provisioning, and the starter dashboard discoverable through Grafana’s API.
- Ran `go test ./... -count=1` successfully after the metrics and compose changes.
- Implemented the next backend metrics slice:
  - added `scraper_op_failures_total` for stable scheduler failure classification by site, queue, runner, and error code,
  - added focused tests for runner metrics classification, scheduler failure metrics, and the worker metrics endpoint,
  - added Prometheus recording rules and a first alert set for worker down, API down, sustained throttling, and elevated failure rate,
  - mounted Prometheus rule files through `docker-compose.yml`.
- Verified the new rule bundle by starting Prometheus locally and confirming it loads the config and starts the rule manager without errors.
