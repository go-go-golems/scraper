---
Title: "Workflow V3 Canonical Observations"
Slug: scraper-workflow-v3-observations
Short: "Derive deterministic retry-aware metrics and traces from one terminal Workflow V3 run."
Topics:
- scraper
- workflow-v3
- observations
- researchctl
Commands:
- workflow observations
- workflow researchctl-config
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

Workflow V3 observations are a deterministic projection over authoritative
Scraper records. They are not mutable counters and do not replace runs, nodes,
attempts, external operations, or artifacts. The projector reads one stable
SQLite transaction and identifies the exact source with an event sequence and
SHA-256 source digest.

## Inspect a terminal run

```bash
scraper workflow \
  --workflow-db state/workflow-v3.db \
  --artifact-root state/workflow-v3-artifacts \
  observations <run-id>
```

The command requires a terminal `succeeded`, `failed`, or `canceled` run. It
returns `scraper-workflow-observations/v1`, derivation
`workflow-observations/v1`. Running it again against unchanged records returns
the same source digest, observation digest, metrics, traces, coverage, and
artifact lineage.

The same read model is available at:

```text
GET /api/v3/workflow/runs/{runID}/observations
```

This endpoint is read-only and needs no operator bearer token. A nonterminal
run returns conflict because a terminal observation must not silently change
while work is still being admitted.

## Timing and retry meanings

- `workflow.elapsed` spans durable run admission through terminal recording.
- `workflow.retries` counts attempts beyond the first for each persisted logical node key. Independent map items and reduction partitions are distinct logical work, not retries.
- `workflow.external_operations.elapsed_sum` sums every closed admitted operation, including failed, canceled, timed-out, and unknown outcomes.
- `workflow.external_operations.elapsed_union` merges overlapping provider intervals as half-open intervals.
- `workflow.external_operations.coverage` divides operation interval union intersected with run wall time by run elapsed time.
- `workflow.external_operations.completion_coverage` reports completed versus admitted operations.
- `workflow.accounting.coverage` reports operations with actual or conservative accounting versus all admissions.
- `workflow.attempt_peak_active` and `workflow.operation_peak_active` are interval-overlap maxima.

Ratios retain exact integer numerator and denominator in `value`. Consumers may
use `numericProjection` from the Researchctl bridge for display, but exact
analysis should retain the rational value.

## Coverage instead of guesses

`coverage.queueWaits` states how many attempt eligibility boundaries were
reconstructable. Initial unconstrained static nodes and retries with durable
backoff semantics are covered. First attempts of dynamically materialized map
items, gate-controlled nodes, and budget-controlled nodes remain uncovered
rather than receiving an invented timestamp.

The critical-path trace uses explicit dependencies and closed attempt execution
times for static plan nodes. Dynamic map and reduction orchestration is reported
as uncovered until the durable schema contains the dependency edge needed for
an exact derivation.

## Privacy and custody

The standard observation set includes closed identifiers, digests, integer
counters, timestamps used only in bounded structured traces, failure class and
code, output schemas and digests, and coverage counts. It excludes:

- task inputs and provider request or response bodies;
- arbitrary failure messages;
- completion capabilities and lease tokens;
- credentials and environment values;
- artifact locators and generated output bodies;
- raw event payloads.

The Researchctl runner contract `scraper-workflow-execution/v2` requires
`exportCanonicalObservations: true`. It emits all canonical metrics and traces
plus a verified `workflow-observations.json` artifact. Version 1 is not accepted
through a compatibility path; regenerate the domain config:

```bash
scraper workflow --task-package <package> \
  researchctl-config workflow.js --bindings bindings.json \
  --out execution.json
```

## Review checks

For a retrying run, compare these sources:

```bash
scraper workflow observations <run-id> > observations.json
scraper workflow runs show <run-id> > run.json
```

Then verify that attempt and operation counts match, failed operations are
included in elapsed sum and union, named output digests match artifact lineage,
and a fresh process produces byte-equivalent JSON. The ticket acceptance smoke
also checks Researchctl custody, crash retry, resume, timeout cancellation, and
fresh-process reprojection.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `workflow observations` rejects a run as nonterminal | Work is still active and the source digest may change | Wait for `succeeded`, `failed`, or `canceled`; do not cache a running projection |
| Runner reports `RUNNER_DOMAIN` | The Researchctl specification still embeds execution contract v1 | Regenerate the domain config and plan fixture with `researchctl-config` |
| Queue or critical-path coverage is less than total | The durable source does not contain an exact eligibility or dynamic dependency boundary | Preserve the reported coverage; do not fill the gap from provider timing or wall-clock guesses |
| Fresh-process digest differs | Authoritative records changed, or a projector ordering regression exists | Compare event sequences and source digests, then run the permutation and restart tests |
| Observation artifact verification fails | Bytes changed after the runner emitted the declared artifact | Treat the attempt as failed; never accept metadata-only digest equality |

## See Also

- `scraper help scraper-workflow-v3-product`
- `scraper help scraper-researchctl-runner`
- `scraper workflow runs show --help`
- `scraper workflow researchctl-config --help`
