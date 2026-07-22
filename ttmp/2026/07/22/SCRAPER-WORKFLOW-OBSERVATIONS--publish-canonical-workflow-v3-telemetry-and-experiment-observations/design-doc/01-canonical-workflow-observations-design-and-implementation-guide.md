---
Title: Canonical workflow observations design and implementation guide
Ticket: SCRAPER-WORKFLOW-OBSERVATIONS
Status: active
Topics:
    - scraper
    - workflow-v3
    - observability
    - durability
    - privacy
    - api
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/workflowv3/external_operation.go
      Note: Canonical operation evidence records
    - Path: repo://pkg/workflowv3/types.go
      Note: Run and attempt snapshots
    - Path: repo://pkg/workflowv3runtime/dispatcher.go
      Note: Lease and dispatch timing semantics
    - Path: repo://pkg/workflowv3sqlite/external_operation.go
      Note: Durable operation storage and queries
ExternalSources: []
Summary: Design for deterministic retry-aware metrics and traces derived from Workflow V3 durable state.
LastUpdated: 2026-07-22T23:15:00-04:00
WhatFor: Give every downstream experiment one canonical definition of elapsed time, retries, operations, concurrency, and failure coverage.
WhenToUse: Use when adding Workflow V3 metrics, exports, or Researchctl observation mappings.
---


# Canonical workflow observations design and implementation guide

## Program context

This is the telemetry child of **EXPERIMENT-PLATFORM-CONVERGENCE**. Related tickets are `RESEARCHCTL-EXPERIMENT-PLANS`, `SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER`, `EXPERIMENT-PLATFORM-SCRAPER-RUNNER`, `RAG-V2-WORKFLOW-LOWERING`, `RAG-GEPPETTO-WORKFLOW-OPERATIONS`, `RESEARCHCTL-EXPERIMENT-ANALYSIS`, `RAG-V2-EXECUTION-CUTOVER`, and `TTC-SCRIPTED-EXPERIMENT-ACCEPTANCE`. The umbrella and all child guides live under the three repositories' `ttmp/2026/07/22/` trees.

## Executive summary

Workflow V3 records detailed jobs, attempts, artifacts, and external operations. Consumers should not have to interpret raw rows to define elapsed time or retries. The previous TTC report excluded failed provider calls because its published boundary differed from the operation ledger. This ticket creates a versioned, deterministic observation projector over durable state.

The projector emits bounded metrics and structured traces. It does not store a second authoritative timeline. Every value includes its boundary, unit, coverage, and derivation version.

## Evidence sources

- `pkg/workflowv3/types.go`: run/node snapshot structures.
- `pkg/workflowv3/external_operation.go`: joined admitted-operation records.
- `pkg/workflowv3sqlite/`: durable attempts, leases, gates, budgets, and operations.
- `pkg/workflowv3runtime/dispatcher.go`: dispatch semantics.
- Existing external-operation export manifests provide atomic JSONL custody.

## Observation schema

```go
type ObservationSet struct {
    SchemaVersion     string
    RunID             workflowv3.RunID
    DerivationVersion string
    SourceDigest      string
    Metrics           []Metric
    Traces            []Trace
    Coverage          Coverage
}

type Metric struct {
    Name       string
    Scope      string
    ValueKind  string
    Value      json.RawMessage
    Unit       string
    Boundary   string
    Metadata   json.RawMessage
}
```

Initial stable names:

```text
workflow.elapsed                         microseconds
workflow.job_attempts                    count
workflow.failed_job_attempts             count
workflow.retries                         count
workflow.queue_wait                      microseconds
workflow.external_operations.admitted   count
workflow.external_operations.succeeded  count
workflow.external_operations.failed     count
workflow.external_operations.elapsed     microseconds
workflow.external_operations.coverage   ratio
workflow.attempt_peak_active             count
workflow.operation_peak_active           count
workflow.critical_path                   structured trace
workflow.accounting.coverage             ratio
```

## Boundary definitions

- **Run elapsed**: earliest durable run admission to terminal run recording.
- **Operation-inclusive elapsed**: earliest admitted external operation start to latest completion, intersected with the run lineage but not restricted to successful operations.
- **Queue wait**: sum or distribution of runnable-to-lease intervals, explicitly labeled; never inferred from provider gaps.
- **Retry count**: attempts beyond the first for the same logical node/item/reduction partition.
- **Provider coverage**: union duration of admitted external-operation intervals divided by selected run elapsed.
- **Peak active**: maximum interval overlap using half-open intervals `[start, end)`.

Unknown end times make interval-derived values incomplete. The projector reports coverage and does not silently substitute run end unless a specifically named conservative metric requests it.

## Projection pseudocode

```pseudo
snapshot = store.readTerminalSnapshot(runID)
attempts = store.listAttempts(runID)
operations = store.listExternalOperations(runID)
assert stable read watermark

metrics.add(elapsed(snapshot.createdAt, snapshot.terminalAt))
metrics.add(count(attempts))
metrics.add(countRetries(groupLogicalAttempts(attempts)))
metrics.add(countByOutcome(operations))

closed = operations where completion exists
intervals = closed.map(providerStartedAt, providerStartedAt + elapsedMicros)
metrics.add(unionDuration(intervals))
metrics.add(peakOverlap(intervals))
metrics.add(coverage(closed, operations))

criticalPath = computeDependencyCriticalPath(snapshot.plan, attempts)
traces.add(criticalPath)
sourceDigest = digest(canonical(snapshot, attempts, operations))
return canonicalObservationSet(metrics, traces, sourceDigest)
```

## Privacy model

The standard projector excludes:

- provider request or response bodies;
- arbitrary provider error text;
- workflow input payloads;
- completion keys;
- credentials and headers;
- generated source text.

Allowed failure fields are bounded class and code values. Domain packages may emit separate artifacts under their own explicit policy.

## Decisions

### Decision: derived views, not duplicated counters

- **Context:** Persisted aggregate counters drift when crashes occur between writes.
- **Decision:** Standard observations derive from durable source records at a stable watermark.
- **Consequences:** Projection performance needs indexing and may later use verified caches keyed by source digest.
- **Status:** accepted.

### Decision: all percentages name their denominator

- **Decision:** Occupancy, operation coverage, accounting coverage, and retry incidence remain separate metrics.
- **Rationale:** A single “utilization” percentage is ambiguous.
- **Status:** accepted.

### Decision: failed operations are first-class

- **Decision:** Inclusive timing and operation counts include failed, canceled, timed-out, and unknown admitted operations unless the metric name explicitly selects success.
- **Status:** accepted.

## Implementation phases

1. Inventory durable source tables and define a stable read transaction.
2. Implement canonical attempt grouping and retry identity.
3. Implement interval union and peak overlap with table-driven tests.
4. Implement run, attempt, queue, and operation metrics.
5. Implement critical-path structured trace.
6. Add coverage and missing-data metadata.
7. Add JSON export and Researchctl frame mapping.
8. Add CLI inspection: `scraper workflow observations <run-id>`.
9. Add verified cache only if profiling proves necessary.

## Test strategy

- zero-operation workflows;
- overlapping and adjacent intervals;
- failed operation before successful retry;
- operation extending beyond a successful-attempt-only boundary;
- unknown completion;
- task retry and whole-run retry distinction;
- stale lease completion;
- map and reduce logical attempt grouping;
- truncated operation export;
- secret canary exclusion;
- deterministic output under shuffled query order;
- TTC regression fixture with a 21.472-second omitted retry interval.

## Intern guidance

Implement interval and retry logic as pure functions before touching SQLite. Use integer microseconds and UTC timestamps. Document whether every duration is wall-clock union, sum of operation durations, or critical-path duration. They answer different questions. Never call a single metric `utilization` without a denominator and interval boundary.

## Completion criteria

For any terminal Workflow V3 run, one command returns deterministic, versioned, privacy-safe observations whose retry-aware timing agrees with the underlying operation ledger. Researchctl can record those values without domain-specific code.

## Technology primer: events, intervals, counters, and projections

Workflow state answers what is currently authoritative. An event records a transition. An interval records time between two boundaries. A counter records a bounded quantity. A projection derives a useful view from those records. These are different data forms and should not be forced into one metric table prematurely.

Elapsed wall time is the difference between two timestamps. Operation work may be measured as the sum of durations, which double-counts concurrent operations, or as the union of intervals, which measures covered wall time. Peak concurrency is computed from interval endpoints. A critical path follows dependency and timing constraints through attempts. Each metric answers a different question.

```text
operation A: |-------------------|
operation B:       |--------|
operation C:                    |----|

sum durations: A + B + C
union duration: from A start through C end, excluding uncovered gaps
peak active: 2
```

The API must name which quantity it emits. `workflow.external_operations.elapsed_sum` and `workflow.external_operations.elapsed_union` are preferable to one ambiguous `provider_time`.

## Stable read and source digest

A projection must read a consistent state. If it lists attempts, then lists operations while another transaction commits completion, the output can combine two moments. Use one read transaction or an explicit store watermark. Canonicalize source records in stable identity order and hash them. The observation set can then state exactly which durable state produced it.

```pseudo
begin read transaction
snapshot = read run terminal state
attempts = read all attempts ordered by logical key, attempt
operations = read all operations ordered by operation id
budgets = read budget records
commit read transaction
sourceDigest = digest(canonical(snapshot, attempts, operations, budgets))
```

A cached projection is valid only when its derivation version and source digest match.

## Interval algorithms

For union duration, sort intervals by start and merge overlaps. Treat intervals as half-open so an operation ending at `t` and another starting at `t` do not overlap.

```pseudo
sort intervals by (start, end)
current = first
for next in intervals[1:]:
    if next.start <= current.end:
        current.end = max(current.end, next.end)
    else:
        total += current.end - current.start
        current = next
total += current.end - current.start
```

For peak concurrency, emit `(start,+1)` and `(end,-1)` endpoints. Sort ends before starts at equal timestamps to preserve half-open semantics. Scan cumulative activity and retain the maximum.

These pure functions deserve property tests: union duration never exceeds elapsed span, never exceeds summed duration, and is invariant under input permutation.

## Retry identity

A retry is not “every attempt after the first in the run.” Attempts must be grouped by logical work identity:

```text
simple node: run + node key
map item: run + map key + item identity
reduction: run + reduce key + level + partition identity
```

If ten map items each execute once, retry count is zero even though there are ten attempts. If one item executes three times, retry count is two. The grouping implementation should live close to Workflow V3 semantics rather than in Researchctl analysis.

## Worked TTC boundary regression

The historical cell published elapsed time from a successful execution boundary while an earlier failed provider operation lay outside it. The operation ledger showed:

```text
failed op begins ---------------- fails
                         successful retry begins -------- succeeds
published boundary:      [--------------------------------]
inclusive boundary: [-------------------------------------]
```

The canonical projector does not choose a successful-attempt-only boundary for `workflow.elapsed`. It derives run elapsed from run admission to terminal recording and separately publishes successful-operation summaries. The report can compare them, but cannot silently substitute one for the other.

## Observation metadata example

```json
{
  "name":"workflow.external_operations.elapsed_union",
  "valueKind":"integer",
  "value":178522000,
  "unit":"microseconds",
  "scope":"run",
  "metadata":{
    "boundary":"all-admitted-closed-operations/v1",
    "outcomes":["succeeded","failed","canceled","timed-out","unknown"],
    "coverage":{"closed":10,"admitted":10},
    "derivationVersion":"workflow-observations/v1"
  }
}
```

Metadata should be bounded and schema-controlled. Avoid embedding a prose methodology in every row; the boundary identifier points to documentation.

## File-level implementation route

Create a package such as `pkg/workflowv3observations` with pure `intervals.go`, `attempts.go`, and `project.go`. Define source-reader interfaces in that package and implement them in `pkg/workflowv3sqlite`. Keep Researchctl mapping in the integration runner, not in the projector. Add `cmd` wiring only after the data contract and pure tests are accepted.

## Debugging and validation

When a projected value looks wrong, first export the source attempt and operation records. Recalculate the smallest metric by hand. Check UTC normalization and microsecond conversion. Then check grouping identity. Do not patch the final aggregate until source records and boundary semantics are understood.

Useful focused commands:

```bash
cd /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper
go test ./pkg/workflowv3/... ./pkg/workflowv3sqlite/... -count=1
go test ./pkg/workflowv3runtime/... -run 'External|Retry|Lease' -count=1
```

## Common mistakes

- Summing overlapping operations and calling the result wall time.
- Counting map items as retries because grouping identity is incomplete.
- Closing unknown operations at run end without labeling the value conservative.
- Mixing provider-reported occurrence time with host-recorded time without preserving both.
- Omitting failures from elapsed metrics.
- Emitting percentages without denominator counts.
- Persisting aggregate counters as a second mutable authority.

## API reference for downstream consumers

The projector should expose one service method and keep storage details behind a source interface:

```go
type Source interface {
    Snapshot(context.Context, workflowv3.RunID) (SourceSnapshot, error)
}

type Projector interface {
    Project(context.Context, workflowv3.RunID, ProjectOptions) (ObservationSet, error)
}
```

`ProjectOptions` may select standard detail levels, but it must not redefine metric formulas. Formula changes require a new derivation version. The Researchctl runner maps `Metric` and `Trace` values to `lab.MetricInput` and `lab.TraceInput`. A CLI encoder renders the same `ObservationSet` as JSON. Both consumers therefore share one implementation.

A structured critical-path trace can include node key, logical item identity, attempt number, start/end timestamps, and predecessor edge. It should not include task payloads. Queue-distribution traces can retain bounded quantiles rather than every scheduling event when a complete event artifact would be too large.

## Review exercise

Take a synthetic run with two parallel nodes, one retry, and three operations. Draw the intervals, calculate union/sum/peak manually, and write the expected observation JSON before running code. Then permute database result order and confirm the same digest. This exercise catches most ordering and boundary mistakes before integration.

## Intern onboarding checklist

The engineer should hand-compute union duration and peak overlap, explain half-open intervals, group a map retry correctly, reproduce the historical omitted-failure fixture, identify privacy-safe operation fields, and regenerate the same observation digest from shuffled source rows.

## References

- Program: `EXPERIMENT-PLATFORM-CONVERGENCE` in Researchctl.
- `pkg/workflowv3/external_operation.go`
- `pkg/workflowv3/types.go`
- `pkg/workflowv3sqlite/`
- Depends on `SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER`.
- Consumed by `EXPERIMENT-PLATFORM-SCRAPER-RUNNER`, `RESEARCHCTL-EXPERIMENT-ANALYSIS`, and all RAG/TTC tickets.
