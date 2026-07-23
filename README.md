# scraper

`scraper` is a durable, JavaScript-authored workflow engine. Workflow V3 is the
primary product surface: canonical plans, SQLite-backed runs and leases,
append-only attempts, deterministic retries, cancellation fencing,
content-addressed artifacts, typed task packages, bounded dispatch, and pure
JavaScript authoring.

The older site-oriented engine remains available only while downstream site and
RAG cutovers are completed. Its worker is explicitly namespaced as
`scraper legacy worker run`; new generic workflow code must use Workflow V3.

## Workflow V3 quickstart

Build the binaries:

```bash
make build-go
```

Create an isolated state directory and use the bundled cookbook task package:

```bash
root=$(mktemp -d)
cp examples/workflowv3/cookbook-linear/* "$root/"

./dist/scraper workflow validate "$root/workflow.js"
./dist/scraper workflow explain "$root/workflow.js"
./dist/scraper workflow compile "$root/workflow.js" --out "$root/plan.json"

./dist/scraper workflow \
  --workflow-db "$root/workflow.db" \
  --artifact-root "$root/artifacts" \
  run "$root/workflow.js" \
  --inputs "$root/inputs.json" \
  --run-id cookbook-1
```

The example executes two versioned JavaScript tasks: normalize newline-delimited
customer JSON and validate unique IDs. Input paths are resolved relative to the
inputs manifest, staged into the content-addressed store, and represented in
SQLite only by typed immutable references.

## Separate submit and worker processes

Submission does not require a running worker:

```bash
./dist/scraper workflow \
  --workflow-db "$root/workflow.db" \
  --artifact-root "$root/artifacts" \
  submit "$root/workflow.js" \
  --inputs "$root/inputs.json" \
  --run-id durable-1
```

A worker may start in a later process or after a machine restart:

```bash
./dist/scraper worker \
  --workflow-db "$root/workflow.db" \
  --artifact-root "$root/artifacts" \
  --capacity cpu.default=4 \
  run
```

Stop workers with SIGINT or SIGTERM. Pending work, retry deadlines, leases,
attempt history, and cancellation epochs remain durable. Restarting a worker is
the recovery operation; do not resubmit the same scientific execution under a
new identity merely because a worker stopped.

## Inspect and control runs

```bash
./dist/scraper workflow --workflow-db "$root/workflow.db" \
  --artifact-root "$root/artifacts" runs list
./dist/scraper workflow --workflow-db "$root/workflow.db" \
  --artifact-root "$root/artifacts" runs show durable-1
./dist/scraper workflow --workflow-db "$root/workflow.db" \
  --artifact-root "$root/artifacts" runs follow durable-1
./dist/scraper workflow --workflow-db "$root/workflow.db" \
  --artifact-root "$root/artifacts" runs cancel durable-1
./dist/scraper task-packages list
```

All command output is structured JSON. `runs follow` emits NDJSON only when the
snapshot changes and exits at a terminal state.

## Operator HTTP API

Serve the same stable product read models and cancellation operation:

```bash
export SCRAPER_WORKFLOW_OPERATOR_TOKEN="$(openssl rand -hex 32)"
./dist/scraper workflow \
  --workflow-db "$root/workflow.db" \
  --artifact-root "$root/artifacts" \
  serve --address 127.0.0.1:8081

curl http://127.0.0.1:8081/api/v3/workflow/health
curl http://127.0.0.1:8081/api/v3/workflow/runs
curl http://127.0.0.1:8081/api/v3/workflow/runs/durable-1
curl http://127.0.0.1:8081/api/v3/workflow/runs/durable-1/observations
curl -X POST -H "Authorization: Bearer $SCRAPER_WORKFLOW_OPERATOR_TOKEN" \
  http://127.0.0.1:8081/api/v3/workflow/runs/durable-1/cancel
curl http://127.0.0.1:8081/api/v3/workflow/task-packages
```

## Canonical Workflow observations

Derive deterministic retry-aware metrics, traces, coverage, and output lineage from any terminal run:

```bash
./dist/scraper workflow \
  --workflow-db "$root/workflow.db" \
  --artifact-root "$root/artifacts" \
  observations durable-1 > "$root/observations.json"
./dist/scraper help scraper-workflow-v3-observations
```

Observations include failed external operations, separate elapsed sum and interval union, exact rational coverage values, task retry grouping, peak activity, bounded failure evidence, and an explicit critical-path coverage boundary. They are re-derived from one stable read transaction and never persisted as a second mutable authority.

## Researchctl integration runner

The `scraper-workflow-runner` binary executes one Workflow V3 run as one Researchctl laboratory attempt through the existing NDJSON process protocol. Generate a strict portable domain config from a workflow and artifact bindings:

```bash
./dist/scraper workflow \
  --task-package research-runner-fixture \
  researchctl-config examples/research-runner/workflow.js \
  --bindings examples/research-runner/bindings.json \
  --out /tmp/scraper-workflow-execution.json
```

The v2 runner contract preserves Researchctl run/attempt IDs and an opaque Scraper run ID, keeps task retries inside Scraper, copies verified final outputs, canonical observations, and external-operation evidence into Researchctl, and propagates cancellation before forced process termination. See:

```bash
./dist/scraper help scraper-researchctl-runner
```

## Repository layout

- `cmd/scraper/` — main product binary;
- `cmd/scraper-workflow-runner/` — Researchctl NDJSON integration executable;
- `pkg/workflowv3/` — canonical plans, identities, registries, artifacts, and policies;
- `pkg/gojamodules/workflow/` — pure descriptor-only JavaScript authoring;
- `pkg/workflowv3sqlite/` — durable control state and stable source snapshots;
- `pkg/workflowv3observations/` — canonical retry-aware observation contract and pure projector;
- `pkg/workflowv3runtime/` — dispatcher, task execution, modules, and isolation;
- `pkg/workflowv3product/` — production configuration, dependency construction, service/read models, and HTTP handler;
- `pkg/researchrunner/` — strict execution contract, lineage, observation projection, and cancellation bridge;
- `pkg/taskpackages/` — versioned production task packages;
- `examples/workflowv3/` — runnable authoring and input examples;
- `pkg/doc/` — embedded operator and architecture help;
- `pkg/engine/`, `pkg/workflow/`, `pkg/sites/`, `pkg/js/runtime/` — retained legacy system awaiting explicit downstream deletion gates;
- `web/` — current frontend.

## Legacy site workflows

Existing site commands and API remain available during the migration:

```bash
./dist/scraper --sites-manifest-dir ./sites site js-demo run seed --help
./dist/scraper api serve --help
./dist/scraper legacy worker run --help
./dist/scraper engine status --help
```

They are not the extension path for new generic workflows. Deletion requires
the named site and RAG replacement tickets to pass their acceptance fixtures;
no Workflow V3 compatibility adapter wraps the old engine.

## Development and validation

```bash
make test
make build
make lint
make logcopter-check
```

The complete local stack remains available through `devctl up`; it currently
hosts legacy API/frontend consumers until their separate cutover gates pass.

Useful embedded documentation:

```bash
./dist/scraper help scraper-workflow-v3-product
./dist/scraper help scraper-workflow-v3-minimal-runtime
./dist/scraper help scraper-architecture-overview
./dist/scraper help scraper-new-developer-onboarding
```
