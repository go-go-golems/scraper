---
Title: Local Prometheus and Grafana smoke test
Ticket: SCRAPER-PROMETHEUS-METRICS
Status: active
Topics:
    - scraper
    - architecture
    - frontend
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Repeatable local procedure for running scraper API and worker with Prometheus metrics enabled and verifying Prometheus and Grafana scrape them correctly.
LastUpdated: 2026-04-07T13:26:00-04:00
WhatFor: Give developers a concrete local workflow for validating the Prometheus integration without guessing ports, flags, or Grafana setup.
WhenToUse: Use after backend Prometheus instrumentation lands or when debugging local scrape failures in the development stack.
---

# Local Prometheus and Grafana smoke test

## Purpose

Start scraper locally with Prometheus metrics enabled, run Prometheus and Grafana via `docker compose`, submit a demo workflow, and verify that:

- the API exposes `/metrics`,
- the worker exposes `/metrics`,
- Prometheus sees both targets as healthy,
- Grafana has a provisioned Prometheus datasource and starter dashboard.

## Environment Assumptions

- Working directory is `/home/manuel/workspaces/2026-03-23/js-scraper/scraper`
- Docker is running
- `tmux` is available
- No conflicting services are already bound to ports `3000`, `8080`, `9090`, or `9091`

If needed, free the common local ports first:

```bash
lsof-who -p 3000 -k || true
lsof-who -p 8080 -k || true
lsof-who -p 9090 -k || true
lsof-who -p 9091 -k || true
```

## Commands

Create a shared temp workspace:

```bash
cd /home/manuel/workspaces/2026-03-23/js-scraper/scraper
export SCRAPER_TMPDIR=$(mktemp -d)
echo "$SCRAPER_TMPDIR"
```

Start monitoring services:

```bash
docker compose up -d redis prometheus grafana
docker compose ps
```

Start the API server in tmux so Prometheus can scrape it from Docker:

```bash
tmux new-session -d -s scraper-api \
  "cd /home/manuel/workspaces/2026-03-23/js-scraper/scraper && \
   go run ./cmd/scraper api serve \
     --address 0.0.0.0:8080 \
     --engine-db \"$SCRAPER_TMPDIR/engine.db\" \
     --sites-dir \"$SCRAPER_TMPDIR/sites\""
tmux capture-pane -pt scraper-api
```

Start the worker with its own metrics listener:

```bash
tmux new-session -d -s scraper-worker \
  "cd /home/manuel/workspaces/2026-03-23/js-scraper/scraper && \
   go run ./cmd/scraper worker run \
     --engine-db \"$SCRAPER_TMPDIR/engine.db\" \
     --sites-dir \"$SCRAPER_TMPDIR/sites\" \
     --poll-interval 50ms \
     --metrics-address 0.0.0.0:9091"
tmux capture-pane -pt scraper-worker
```

Verify the raw metrics endpoints:

```bash
curl -s http://127.0.0.1:8080/metrics | rg 'scraper_http_requests_total|scraper_engine_workflows_total|scraper_engine_workflow_status_total'
curl -s http://127.0.0.1:9091/metrics | rg 'scraper_scheduler_cycles_total|scraper_workers_up|scraper_queue_wait_seconds'
```

Submit a demo workflow:

```bash
curl -X POST http://127.0.0.1:8080/api/v1/sites/js-demo/verbs/seed:submit \
  -H 'Content-Type: application/json' \
  -d '{
    "workflowID": "prom-smoke-001",
    "values": {
      "count": 3,
      "multiplier": 4,
      "prefix": "prom"
    }
  }'
```

Check Prometheus targets:

```bash
curl -s http://127.0.0.1:9090/api/v1/targets | rg 'scraper-api|scraper-worker|health'
```

Open Grafana:

```bash
open http://127.0.0.1:3000
```

Default local credentials from compose:

- username: `admin`
- password: `admin`

Cleanup:

```bash
tmux kill-session -t scraper-api || true
tmux kill-session -t scraper-worker || true
docker compose down
rm -rf "$SCRAPER_TMPDIR"
```

## Exit Criteria

- `curl http://127.0.0.1:8080/metrics` returns Prometheus text format with `scraper_*` metrics
- `curl http://127.0.0.1:9091/metrics` returns worker `scraper_*` metrics
- Prometheus target API shows both `scraper-api` and `scraper-worker` as `up`
- Grafana loads with a provisioned Prometheus datasource
- The starter dashboard `Scraper Overview` appears under the `Scraper` folder

## Alert Response Guidance

- `ScraperWorkerDown`
  - Check `tmux capture-pane -pt scraper-worker`
  - Verify the worker process is running and `--metrics-address` is set
  - Restart the worker if the process is gone or the listener failed to bind
- `ScraperAPIUnavailable`
  - Check `tmux capture-pane -pt scraper-api`
  - Verify the API is bound to `0.0.0.0:8080`
  - Confirm `curl http://127.0.0.1:8080/metrics` works on the host
- `ScraperQueueRateLimitedHot`
  - Inspect queue policy and token bucket configuration for the affected `site/queue`
  - Check whether throughput is intentionally capped or whether the queue is under-provisioned
  - Correlate with runtime events to see whether jobs are backing up
- `ScraperOpFailureRateHigh`
  - Inspect runtime events and workflow details for the affected `site/queue/runner`
  - Separate transient upstream failures from permanent verb or input errors
  - If failures are retryable, check whether they are also causing retry spikes
- `ScraperQueueWaitHigh`
  - Check whether the queue is starved by rate limits or worker capacity
  - Compare `scraper_queue_state_total` ready/running counts with `scraper_queue_wait_seconds`
  - If queue wait is isolated to one queue, inspect that queue’s policies and runner type
- `ScraperRetrySpike`
  - Look for upstream flakiness, bad proxy behavior, or unstable verb logic
  - Correlate retries with failure-rate and queue-wait alerts to decide whether the issue is transient or systemic

## Notes

- If Prometheus cannot scrape the host targets, confirm the API and worker were started with `0.0.0.0`, not `127.0.0.1`
- If the worker metrics endpoint is empty, confirm `--metrics-address` was passed
- If Grafana has no datasource, inspect `ops/monitoring/grafana/provisioning/`
