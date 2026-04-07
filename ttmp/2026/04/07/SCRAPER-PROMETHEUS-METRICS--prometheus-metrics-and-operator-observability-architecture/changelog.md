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
