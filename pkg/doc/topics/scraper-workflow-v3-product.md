---
Title: Workflow V3 Product and Operator Guide
Slug: scraper-workflow-v3-product
Short: "Author, validate, submit, execute, recover, inspect, follow, and cancel durable Workflow V3 runs with the main scraper binary."
Topics:
- scraper
- workflow-v3
- javascript
- workers
- operations
Commands:
- workflow
- worker
- task-packages
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

Workflow V3 is Scraper's primary workflow product. A JavaScript authoring file
builds a pure descriptor graph; compilation pins exact task, bundle, module,
resource, retry, isolation, catalog, and plan identities. The SQLite control
store owns runs, nodes, leases, attempts, retry deadlines, cancellation epochs,
and bounded operational events. Artifact payloads remain in the configured
content-addressed artifact root.

The retained older site engine is available only for downstream migrations. Its
worker moved to `scraper legacy worker run`. New generic workflow integrations
must use the commands documented here and must not import `pkg/engine`.

## Runnable example

The repository contains a versioned `cookbook-linear` task package. Its two
JavaScript tasks normalize newline-delimited customer JSON and validate unique
IDs. Prepare an inputs manifest whose paths are resolved relative to the
manifest itself:

```json
{
  "source": {
    "path": "customers.jsonl",
    "schema": "customer-jsonl-ref/v1",
    "mediaType": "application/x-ndjson"
  }
}
```

Run the complete lifecycle:

```bash
root=$(mktemp -d)
cp examples/workflowv3/cookbook-linear/* "$root/"

scraper workflow validate "$root/workflow.js"
scraper workflow explain "$root/workflow.js"
scraper workflow compile "$root/workflow.js" --out "$root/plan.json"

scraper workflow \
  --workflow-db "$root/workflow.db" \
  --artifact-root "$root/artifacts" \
  submit "$root/workflow.js" \
  --inputs "$root/inputs.json" \
  --run-id cookbook-1

scraper worker \
  --workflow-db "$root/workflow.db" \
  --artifact-root "$root/artifacts" \
  --poll-interval 25ms \
  run
```

The worker runs until SIGINT or SIGTERM. It is safe to stop after submission and
start another process later with the same database, artifact root, task package
set, and exact package bytes. Pending work and retry deadlines remain durable.
A stale lease cannot publish output after cancellation or after another attempt
has taken authority.

For one-process development, `workflow run` submits and dispatches until that
run reaches `succeeded`, `failed`, or `canceled`:

```bash
scraper workflow \
  --workflow-db "$root/workflow.db" \
  --artifact-root "$root/artifacts" \
  run "$root/workflow.js" \
  --inputs "$root/inputs.json" \
  --run-id cookbook-local
```

## Inspection and control

```bash
scraper workflow --workflow-db "$root/workflow.db" \
  --artifact-root "$root/artifacts" runs list
scraper workflow --workflow-db "$root/workflow.db" \
  --artifact-root "$root/artifacts" runs show cookbook-1
scraper workflow --workflow-db "$root/workflow.db" \
  --artifact-root "$root/artifacts" runs follow cookbook-1
scraper workflow --workflow-db "$root/workflow.db" \
  --artifact-root "$root/artifacts" runs cancel cookbook-1
scraper task-packages list
```

`runs show` combines the immutable run snapshot with a bounded operational
projection. `runs follow` emits changed snapshots as newline-delimited JSON and
stops at a terminal state. Cancellation increments durable fencing state;
workers cannot revive the run.

The optional operator API exposes the same service read models:

```bash
export SCRAPER_WORKFLOW_OPERATOR_TOKEN="$(openssl rand -hex 32)"
scraper workflow --workflow-db "$root/workflow.db" \
  --artifact-root "$root/artifacts" serve --address 127.0.0.1:8081

curl http://127.0.0.1:8081/api/v3/workflow/runs
curl http://127.0.0.1:8081/api/v3/workflow/runs/cookbook-1
curl -X POST -H "Authorization: Bearer $SCRAPER_WORKFLOW_OPERATOR_TOKEN" \
  http://127.0.0.1:8081/api/v3/workflow/runs/cookbook-1/cancel
curl http://127.0.0.1:8081/api/v3/workflow/task-packages
```

## Configuration and authority

The product flags are host authority, never plan authority:

- `--workflow-db` selects the durable SQLite control store;
- `--artifact-root` selects immutable payload custody;
- repeatable `--task-package` selects exact packages;
- repeatable `--capacity resource.class=N` bounds concurrent leases;
- `--lease-duration` controls temporary attempt authority;
- `--poll-interval` controls cross-process/retry wakeups;
- `--max-artifact-bytes` bounds staged and produced artifacts;
- `--operator-token-env` names the bearer-token environment variable required by mutating HTTP requests. An unset or empty token leaves the API read-only.

A plan cannot choose database paths, task implementation bytes, host modules,
worker capacity, or secrets. Unknown packages, duplicate package selections,
duplicate descriptor modules, unavailable runtime modules, and mismatched plan
identities fail before execution.

## Recovery and failures

Do not resubmit a run merely because a worker stopped. Restart a worker against
the same durable state. Lease expiry records an immutable lease-loss attempt and
a later lease receives a new attempt number. Retryable task failures use the
compiled retry policy and durable backoff. Non-retryable failures terminate the
run with typed class/code evidence. Use a new run ID only for a scientifically
new execution.

If a run appears stuck, inspect `runs show`. Common blocked reasons include a
dependency, retry deadline, capacity exhaustion, budget policy, gate decision,
or unavailable exact implementation. Do not edit SQLite rows manually.
