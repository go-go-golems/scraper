---
Title: Workflow observation contract acceptance and coverage audit
Ticket: SCRAPER-WORKFLOW-OBSERVATIONS
Status: active
Topics:
    - scraper
    - workflow-v3
    - observability
    - privacy
    - workflows
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/researchrunner/runner_test.go
      Note: Process projection privacy and custody
    - Path: repo://pkg/workflowv3observations/project_test.go
      Note: Permutation timing coverage and historical retry regression
    - Path: repo://pkg/workflowv3observations/retry_test.go
      Note: Map reduce and lease-loss identity proof
    - Path: repo://pkg/workflowv3product/application_test.go
      Note: Restart failure cancellation integration
    - Path: repo://ttmp/2026/07/22/SCRAPER-WORKFLOW-OBSERVATIONS--publish-canonical-workflow-v3-telemetry-and-experiment-observations/sources/smoke/01-summary.json
      Note: Machine-readable cross-repository result
ExternalSources: []
Summary: Acceptance evidence and exact coverage boundaries for canonical retry-aware Workflow V3 observations.
LastUpdated: 2026-07-23T14:05:00-04:00
WhatFor: Review metric formulas, source custody, privacy, and Researchctl projection before changing the observation contract.
WhenToUse: Use before analysis tooling, RAG lowering, or observation schema changes.
---


# Workflow observation contract acceptance and coverage audit

## Verdict

`scraper-workflow-observations/v1` is the canonical observation contract for terminal Workflow V3 runs. It derives from one stable read transaction and stores no aggregate counters. Its strict decoder accepts exactly 22 metrics and 3 trace kinds, verifies the complete observation digest, and rejects unknown JSON fields, unknown names, missing names, unsorted names, malformed identities, nonterminal sources, duplicate lineage, and source records beyond closed limits.

The Researchctl bridge hard-cuts its domain contract to `scraper-workflow-execution/v2`. Version 2 requires `exportCanonicalObservations: true`; the runner does not decode v1 through a compatibility branch.

## Authoritative source map

| Observation source | Authoritative record | Projection treatment |
|---|---|---|
| Run identity and elapsed | `v3_runs` exact plan, digest, status, creation, update | Admission through terminal recording |
| Logical work and retries | `v3_nodes`, `v3_attempts` | Attempts beyond first per persisted node key |
| Static dependencies | `v3_dependencies` and exact compiled plan | Dependency-weighted critical-path execution trace |
| Dynamic work identity | `v3_map_items`, `v3_reduction_partitions` | Distinct logical work; never counted as retries merely because several items exist |
| External effects | admissions and optional immutable completions | All outcomes included; incomplete admissions remain in denominators |
| Outputs | terminal named references resolved from plan | Name, schema, digest, media type, and size only |
| Stable watermark | maximum `v3_events.sequence` in the read transaction | Identifies the durable source moment |

The snapshot excludes event payloads, artifact locators, attempt failure messages, lease tokens, completion keys, task inputs, provider bodies, and credentials before the projector sees them.

## Metric formula audit

### Time

- `workflow.elapsed`: terminal recording minus run admission.
- `workflow.external_operations.elapsed_sum`: sum of all closed provider-reported intervals, including failures.
- `workflow.external_operations.elapsed_union`: half-open union of those same intervals; concurrency is not double-counted.
- `workflow.external_operations.coverage`: union after intersection with the run wall-time interval divided by run elapsed.
- `workflow.queue_wait`: sum of only reconstructable eligibility-to-start intervals.

All durations use signed 64-bit integer microseconds and UTC source timestamps. Unknown external-operation completions are not closed at run end.

### Counts and exact ratios

Retry, attempt, cancellation, lease-loss, operation outcome, and peak-active values are integers. Coverage and throughput values retain an exact `{numerator, denominator}` object. The runner adds a floating-point `numericProjection` only for Researchctl display and statistics; canonical custody retains the exact rational value.

`workflow.external_operations.completion_coverage` and `workflow.accounting.coverage` are distinct. Completion coverage answers whether every admission has terminal evidence. Accounting coverage answers whether actual or conservative usage accounting exists. Neither is mislabeled utilization.

### Critical path

The trace weights each static plan node by the sum of its closed attempt execution durations, including retries, and follows explicit persisted dependency edges. Dynamically materialized map items and reduction partitions are not assigned invented orchestration edges. `coverage.criticalPath` exposes the resulting observed/total count.

## Coverage policy

The projector never turns missing evidence into a plausible timestamp:

- an unconstrained static first attempt is eligible at run admission;
- a static dependent first attempt is eligible at the latest successful predecessor completion;
- a retry is eligible after the previous completion plus the compiled retry backoff;
- first attempts controlled by gates, budgets, maps, or reductions remain uncovered;
- incomplete operations remain in admitted and coverage denominators but not interval metrics;
- dynamic nodes remain outside the v1 critical path unless exact dependency edges become authoritative.

A future durable source can increase coverage without changing formulas. A formula or semantic boundary change requires a new derivation version.

## Determinism and restart evidence

Pure tests shuffle nodes, attempts, operations, dependencies, and interval inputs and produce identical source and observation digests. The product integration closes and reopens SQLite, then reproduces the complete observation set. The cross-repository smoke reopens each selected subordinate database through a fresh built Scraper process and compares the resulting JSON structurally with the artifact already verified by Researchctl.

The source digest covers the exact compiled plan, safe attempt records, safe operation ledger, output references, terminal run record, and event watermark in canonical order. The observation digest covers the complete observation set with only its own digest blanked.

## Privacy evidence

The public projection contains only:

- closed run, node, operation, schema, and digest identities;
- UTC-derived integer timing values and counts;
- closed failure class, code, and retryable flag;
- exact rational values and coverage counts;
- bounded artifact lineage without locator or bytes.

The fixture's free-form retry error remains absent. Input canaries remain absent from every frame. The Researchctl process host still performs its independent byte-level canary scan and artifact verification.

## Cross-repository acceptance

`scripts/01-smoke-canonical-observations.sh` builds Researchctl, Scraper, and the process runner, regenerates the execution v2 contract, and proves:

- two cases × two replicates execute four independent scientific runs;
- one runner crash creates a second Researchctl attempt and a fifth subordinate database;
- one task retry and one failed external operation remain within each selected Workflow attempt;
- every selected attempt publishes 22 canonical metrics, 3 traces, and 4 verified artifacts;
- failed-operation timing appears in the 2,000-microsecond operation sum and union;
- every observation and source digest agrees across metric metadata and the canonical artifact;
- fresh-process reprojection equals the Researchctl-custodied observation;
- complete resume executes zero and resumes all four identities;
- Researchctl timeout durably cancels the subordinate Workflow;
- a fresh process derives a terminal canceled observation from that database.

The committed machine-readable summary is `sources/smoke/01-summary.json`.

## Ownership and neutrality guards

The observation package imports Workflow V3 contracts but no Researchctl or workload package. Researchctl imports no Scraper Go package. The runner maps values across NDJSON and does not query Workflow SQLite directly. Searches for RAG, embedding, reranking, Geppetto, and TTC concepts in the observation and runner packages must remain empty.

## Review commands

```bash
GOWORK=off go test ./pkg/workflowv3observations ./pkg/workflowv3sqlite \
  ./pkg/workflowv3product ./pkg/researchrunner ./pkg/cmd -count=1
GOWORK=off go test -race ./pkg/workflowv3observations ./pkg/workflowv3sqlite \
  ./pkg/workflowv3product ./pkg/researchrunner -count=1
bash ttmp/2026/07/22/SCRAPER-WORKFLOW-OBSERVATIONS--*/scripts/01-smoke-canonical-observations.sh
```
