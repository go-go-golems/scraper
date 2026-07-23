---
Title: Researchctl Workflow V3 Runner
Slug: scraper-researchctl-runner
Short: "Execute one durable Workflow V3 run as one Researchctl laboratory attempt through researchctl-runner-stdio/v1."
Topics:
- scraper
- workflow-v3
- durability
- artifacts
- worker
Commands:
- workflow
- scraper-workflow-runner
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

`scraper-workflow-runner` is the canonical process boundary between Researchctl
experiment custody and Scraper Workflow V3 execution. Researchctl sends one
`researchctl-runner-stdio/v1` request. The runner validates the strict
`scraper-workflow-execution/v1` domain config, verifies input digests and the
exact task-package catalog, creates one Workflow V3 run, dispatches it to a
terminal state, and emits bounded events, metrics, traces, and copied artifacts.

Researchctl and Scraper retain different truths:

```text
Researchctl specification -> run -> attempt
                                  |
                                  | lineage event
                                  v
Scraper workflow run -> nodes -> leases -> task attempts -> operations
```

One Researchctl attempt always creates one opaque Scraper run ID. Workflow task
retries remain inside that Scraper run. If the runner process crashes,
Researchctl may create a new attempt and therefore a new Workflow run. The old
Workflow database remains evidence; the runner never guesses that its result
was accepted.

## Build the contract

Use the production task package and pure Workflow V3 authoring surface to
produce a canonical domain config:

```bash
scraper workflow \
  --task-package research-runner-fixture \
  researchctl-config examples/research-runner/workflow.js \
  --bindings examples/research-runner/bindings.json \
  --out /tmp/scraper-workflow-execution.json
```

Bindings map plan input names to exact Researchctl artifact selectors:

```json
{
  "source": {
    "role": "workflow-input",
    "kind": "fixture-source",
    "id": "source"
  }
}
```

The generated config pins the plan digest, catalog digest, package name,
package version, bundle digest, input bindings, and observation policy. It does
not contain a database path, artifact root, executable path, secret, or worker
capacity. Those are host authority supplied to the runner process.

## Run from Researchctl

Researchctl's example plan contains two cases and two replicates per case. The
laboratory artifact root must contain the input URIs declared by the plan.

```bash
researchctl experiment run-plan examples/lab/scraper-workflow-plan.js \
  --project examples/lab/scraper-workflow-project.js \
  --runner-command /path/to/scraper-workflow-runner \
  --runner-name scraper-workflow-runner \
  --runner-version v1 \
  --runner-arg=--state-root \
  --runner-arg=/var/lib/scraper/research-runner \
  --runner-arg=--artifact-root \
  --runner-arg=/var/lib/scraper/research-artifacts \
  --max-attempts 2 \
  --timeout 30s
```

Host flags include:

- `--state-root`: durable per-attempt Workflow SQLite databases;
- `--artifact-root`: subordinate Workflow content-addressed artifacts;
- repeatable `--task-package`: exact enabled package set;
- repeatable `--capacity name=count`: host scheduling capacities;
- `--lease-duration` and `--poll-interval`: Workflow execution policy;
- `--cancellation-timeout`: bounded Scraper cancellation acknowledgement;
- `--max-request-bytes` and `--max-export-bytes`: protocol limits.

## Evidence projection

A successful fixture attempt records:

- `workflow.submitted` lineage before dispatch;
- `workflow.terminal` with plan and run identities;
- attempt and retry counts;
- admitted, failed, and succeeded external-operation counts;
- a sanitized attempt trace containing closed failure class/code fields;
- each final Workflow output as a Researchctl-verified artifact;
- canonical external-operation JSONL and manifest artifacts;
- a terminal `scraper-workflow-result/v1` payload.

Task inputs, output payloads other than explicitly exported artifacts, arbitrary
task error messages, completion keys, provider bodies, and credentials never
enter event, metric, trace, or completion frames. Researchctl's process runner
also rejects any frame containing configured secret canaries.

## Cancellation and recovery

Researchctl sends an interrupt to the runner before forced termination. The
runner cancels the subordinate Workflow through its durable cancellation epoch,
waits within `--cancellation-timeout`, and exits nonzero so Researchctl records
the attempt as canceled or timeout-classified. `--runner-cancel-grace` on the
Researchctl command must exceed the runner cancellation timeout.

Do not reconnect a later Researchctl attempt to a prior Workflow run. Initial
semantics intentionally create another Workflow run and preserve both lineage
records. Adoption of an earlier terminal Workflow would require a separately
versioned exactly-once protocol.

## Validation

The cross-repository smoke builds both binaries, verifies the embedded contract
against freshly compiled Scraper output, kills one runner after
`workflow.submitted`, proves the Researchctl retry creates a fifth Workflow run,
executes four desired runs, verifies retries/failed operations/artifacts,
resumes all four without execution, and confirms timeout propagation cancels
the subordinate Workflow.

```bash
cd researchctl
ttmp/2026/07/22/EXPERIMENT-PLATFORM-SCRAPER-RUNNER--*/scripts/01-smoke-scraper-workflow-runner.sh
```
