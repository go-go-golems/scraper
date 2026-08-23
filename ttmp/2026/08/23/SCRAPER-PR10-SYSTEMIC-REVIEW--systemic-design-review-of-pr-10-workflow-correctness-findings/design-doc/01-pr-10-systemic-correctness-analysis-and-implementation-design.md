---
Title: PR 10 systemic correctness analysis and implementation design
Ticket: SCRAPER-PR10-SYSTEMIC-REVIEW
Status: complete
Topics:
    - scraper
    - workflow-v3
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/gojamodules/workflow/authoring.go
      Note: JavaScript task/map/reduce/after API and TypeScript declarations
    - Path: repo://pkg/researchrunner/runner.go
      Note: Researchctl digest verification, scalar/set staging, and the two input findings
    - Path: repo://pkg/workflowv3/compiler.go
      Note: IR validation, dependency checks, topology support, and plan compilation
    - Path: repo://pkg/workflowv3/dependencies.go
      Note: Canonical data/control dependency derivation and typed cross-kind cycle validation (commit b0cdd1b)
    - Path: repo://pkg/workflowv3/types.go
      Note: Canonical set-input policy and plan contract (commit 2dfdee1)
    - Path: repo://pkg/workflowv3product/service.go
      Note: Product path/reference staging boundary and submission status
    - Path: repo://pkg/workflowv3sqlite/expansion.go
      Note: Dynamic map dependency lowering and map terminal transition
    - Path: repo://pkg/workflowv3sqlite/reconcile.go
      Note: Single transaction-local successful terminal state and output readiness owner (commit 7f6e728)
    - Path: repo://pkg/workflowv3sqlite/reduction.go
      Note: Dynamic reduction dependency lowering, capacity semantics, and reduction terminal transition
    - Path: repo://pkg/workflowv3sqlite/store.go
      Note: Run creation, static dependency persistence, readiness, input resolution, terminalization, and output projection
ExternalSources:
    - https://github.com/go-go-golems/scraper/pull/10
    - https://github.com/go-go-golems/scraper/pull/10#pullrequestreview-4772468539
Summary: Evidence-backed review of four PR 10 correctness findings and a systemic design that gives byte custody, cardinality, readiness, and terminalization one authoritative owner each.
LastUpdated: 2026-08-23T19:45:00-04:00
WhatFor: Understand, review, and implement the PR 10 correctness fixes without adding more duplicated workflow invariants.
WhenToUse: Read before changing Researchctl input staging, set-input limits, Workflow V3 dependency compilation, scheduling readiness, or run terminal transitions.
---





# PR 10 systemic correctness analysis and implementation design

## Executive summary

PR 10 introduces Workflow V3 as Scraper's sole workflow product and removes the legacy engine. The automated review at commit `202229464629e2b6d0e193ff7798b16770b3a270` found four real correctness defects:

1. a verified scalar input is later reread by pathname, creating a time-of-check/time-of-use window;
2. set-input cardinality is inferred only from direct map consumers, so reduction-only and pass-through plans are rejected;
3. a node-output binding does not automatically make its producer a scheduling dependency, so a consumer may be leased before its input exists;
4. a valid workflow with outputs but no scheduled work remains `running` forever.

At analysis time, the reviewed source files were unchanged between the reviewed commit and local merge commit `e4578b8bcb17317c3fcccbde854c177c24993bdf`, so every comment applied to the implementation baseline. Section 18 records the subsequent corrective commits and validation evidence.

These defects can each be patched locally, but **localized patches alone would preserve the design condition that produced them**: the same correctness fact has multiple owners in different layers. The systemic pattern is:

- Researchctl verifies bytes, but product submission later trusts a path.
- Set-input bounds belong to the input contract, but the runner guesses them from map consumers.
- A binding declares a data edge, while `DependsOn` separately declares a scheduling edge.
- Run completion is a derived predicate, but four transition methods each carry their own copy of the SQL.

The recommended solution is a focused corrective slice, not a rewrite:

- convert verified bytes to an immutable `ArtifactRef` before crossing the runner/product boundary;
- add an explicit cardinality policy to every set input and validate consumers against it;
- derive one typed dependency graph from bindings plus control-only `after(...)` edges, then use that graph for cycle checks and persistence;
- centralize successful terminalization in one transaction-local run reconciler and call it after run creation and every work transition.

A small emergency patch could unblock the PR, but it should use shared helpers and tests shaped toward this design. Avoid a fallback such as “if no map exists, accept anything,” avoid requiring authors to repeat every data edge with `.after(...)`, and avoid adding a fifth copy of the completion predicate to `CreateRun`.

## 1. Scope and review method

This document analyzes the four inline comments on [PR 10](https://github.com/go-go-golems/scraper/pull/10), the reviewed commit, current source, relevant history, tests, and Workflow V3 design tickets.

Evidence gathered:

- PR metadata and review body through `gh pr view` and `gh api`;
- all four inline comments through `repos/go-go-golems/scraper/pulls/10/comments`;
- source and tests under `pkg/researchrunner`, `pkg/workflowv3`, `pkg/workflowv3product`, `pkg/workflowv3runtime`, `pkg/workflowv3sqlite`, and `pkg/gojamodules/workflow`;
- commit history and blame for the relevant lines;
- the main Workflow V3 architecture guide, intern guide, product-cutover ticket, and Researchctl runner diary.

The findings are code-review findings, not observed production incidents. Their failure paths are nevertheless deterministic consequences of the current code. The review links are retained in Section 4 and the References section.

## 2. Intern primer: what system is being reviewed?

### 2.1 The layers

Workflow V3 has five relevant layers. A new contributor should understand the ownership boundary before editing any individual function.

```text
Researchctl process protocol
  pkg/researchrunner
  - verifies external experiment identity, plan/catalog identity, and input digests
  - translates external resolved files into Workflow V3 inputs
  - submits one workflow and exports bounded evidence
             |
             v
Workflow product facade
  pkg/workflowv3product
  - opens the configured artifact store, SQLite store, engine, dispatcher, registry
  - stages path-based CLI inputs
  - submits plans and exposes run views
             |
             v
Pure model/compiler + JavaScript authoring
  pkg/workflowv3
  pkg/gojamodules/workflow
  - IR and plan types
  - schema and policy validation
  - canonical plan digest
  - JavaScript builders for input/task/map/reduce/gate/output
             |
             v
Durable state and admission
  pkg/workflowv3sqlite
  - persists runs, inputs, nodes, dependencies, attempts, maps, reductions, gates
  - chooses ready work transactionally
  - resolves input refs and commits terminal transitions
             |
             v
Runtime and workers
  pkg/workflowv3runtime
  - interleaves gate/map/reduction control work
  - fills resource slots continuously
  - resolves leased inputs, runs tasks, renews leases, commits results
```

The architectural rule is: **descriptive layers may author intent, but authoritative execution facts must be compiled and persisted before workers depend on them**.

### 2.2 The three representations

Workflow V3 intentionally separates three representations:

1. **JavaScript builder objects** are ergonomic, in-memory authoring handles.
2. **`WorkflowIR`** is pure declarative intent: symbolic task keys, bindings, policies, and outputs.
3. **`WorkflowPlan`** is the immutable host-resolved execution contract: pinned bundle identity, resource class, retry, isolation, and digest.

Relevant types are in `pkg/workflowv3/types.go:115-277`. `Compile` validates the IR and pins task identities in `pkg/workflowv3/compiler.go:468-566`.

A `ValueRef` identifies scalar data:

```go
type ValueRef struct {
    Source    string  // input | node-output | gate-output | reduction-output | ...
    Name      string
    NodeKey   NodeKey
    ReduceKey string
    GateKey   NodeKey
    Port      string
    Schema    string
}
```

A `SetRef` identifies a manifest of ordered keyed items:

```go
type SetRef struct {
    Source         string // set-input | map-output
    Name           string
    MapKey         string
    ItemSchema     string
    ManifestSchema string
}
```

The key observation for this review is that these references already contain producer identity. A `node-output` ref is not merely a schema annotation; it states that the consumer cannot execute before that node publishes that port.

### 2.3 Authoring API and the duplicate dependency surface

The JavaScript API exposes `.after(job)` only on ordinary task and gate builders (`pkg/gojamodules/workflow/authoring.go:305-350`; declarations at `authoring.go:726-762`). Map and reduce builders expose boundedness, budget, and isolation, but not `after`.

```ts
const normalized = p.task("normalize", tasks.normalize(source));
const validated = p.task(
  "validate",
  tasks.validate(normalized.output("dataset")),
  job => job.after(normalized),
);
```

Today, `normalized.output("dataset")` and `job.after(normalized)` encode the same producer-consumer relationship twice. If the author forgets `.after`, schema validation succeeds but readiness is wrong. For map/reduce auxiliary bindings, the author cannot express `.after` at all.

### 2.4 Submission and execution flow

```text
ResolvedInput(path, expected digest)
        |
        | readVerifiedInput(path) -> body, verify digest
        | current scalar branch discards body
        v
StagedInput(path) ------------------------------.
        |                                        |
        v                                        | second filesystem read
Application.Submit -> stageInputs -> os.ReadFile(path)
        |
        v
ArtifactStore.Put -> immutable ArtifactRef
        |
        v
Engine.Submit -> Store.CreateRun
        |
        +-> run row status='running'
        +-> input refs
        +-> static node rows and explicit dependencies
        +-> map/reduction/gate rows
        v
Dispatcher
        +-> maintenance: gates, map expansion, map publish, reduction
        +-> LeaseNextWithResources
        v
Engine.ExecuteLease
        +-> Store.ResolveInputs
        +-> task execution
        +-> Store.CompleteWithUsage
```

`FileArtifactStore.Put` is already a strong immutable boundary: it enforces size, hashes the supplied bytes, writes through a temporary file, syncs, renames into a content-addressed locator, and returns an `ArtifactRef` (`pkg/workflowv3/artifacts.go:43-94`). The runner should cross into the product through this boundary immediately after verification.

### 2.5 Readiness and failure

`LeaseNextWithResources` excludes nodes blocked by rows in `v3_dependencies`, `v3_reduction_consumers`, and `v3_gate_consumers` (`pkg/workflowv3sqlite/store.go:300-361`). Once leased, `Engine.ExecuteLease` calls `ResolveInputs`. Missing producer output becomes a durable, non-retryable `WORKFLOW_INPUT_RESOLUTION` failure (`pkg/workflowv3runtime/engine.go:281-298`).

That classification assumes readiness is correct. If the scheduler leased the consumer early because the dependency was never persisted, the engine misclassifies an orchestration invariant failure as a task failure. The scheduler—not the task—is at fault.

### 2.6 Completion

A run is complete when every required work category is complete:

- all nodes succeeded;
- every map expansion is published;
- every reduction root is published;
- every ordinary gate is approved;
- no active budget gate blocks completion;
- every declared output can be resolved.

The current implementation repeats most of that SQL in four places:

- node completion: `pkg/workflowv3sqlite/store.go:671-687`;
- map publication: `pkg/workflowv3sqlite/expansion.go:598-614`;
- reduction publication: `pkg/workflowv3sqlite/reduction.go:484-500`;
- gate approval: `pkg/workflowv3sqlite/gate.go:458-474`.

`CreateRun` always inserts `status='running'` (`pkg/workflowv3sqlite/store.go:96-100`) and does not run the predicate. Therefore a run with no transition-producing work can never be reconciled.

## 3. Review verdict matrix

| # | Severity | Review finding | Correct? | Local repair exists? | Systemic issue exposed |
|---|---|---|---|---|---|
| 1 | P1 | Stage the exact bytes that passed digest verification | Yes | Yes: stage `body` immediately | Verified identity decays into mutable locator identity |
| 2 | P1 | Permit reduction-only and pass-through set inputs | Yes | Yes: derive/fallback bound | Cardinality ownership is absent from set-input contract |
| 3 | P1 | Enforce producer dependency for output bindings | Yes | Yes: infer/reject missing dependency | Dataflow and scheduling graph are separate truths |
| 4 | P2 | Complete workflows with no scheduled work | Yes | Yes: create as succeeded | Run completion predicate is duplicated and event-driven rather than reconciled |

**Overall recommendation:** address all four before merging. Use a systemic shape, but split implementation into reviewable phases. The fixes touch different packages; forcing them into one giant abstraction would be counterproductive. The common design rule is “one owner per invariant.”

## 4. Finding-by-finding analysis

### 4.1 Finding 1 — verified bytes are replaced by an unverified path read

Review: [discussion_r3644764557](https://github.com/go-go-golems/scraper/pull/10#discussion_r3644764557)

#### Current behavior

`resolveInputs` reads every resolved file and verifies size and SHA-256 (`pkg/researchrunner/runner.go:303-375`). The set-input branch passes the verified `body` to `stageSetInput`, which writes item bytes and a manifest into the artifact store. The scalar branch instead creates `StagedInput{Path: resolvedInput.Path, ...}` (`runner.go:325-330`).

`Application.Submit` calls `stageInputs`, which calls `os.ReadFile(path)` a second time (`pkg/workflowv3product/service.go:101-155`). No comparison with the Researchctl reference occurs on that second read.

#### Failure timeline

```text
T0 Researchctl says path P has digest D
T1 runner reads P -> bytes A
T2 runner verifies sha256(A) == D
T3 another process replaces P -> bytes B
T4 product stageInputs reads P -> bytes B
T5 ArtifactStore records sha256(B), workflow executes B
T6 experiment identity still says D == sha256(A)
```

The result is not merely a filesystem race. It violates experiment reproducibility: the canonical Researchctl identity commits to A while the workflow executes B.

#### Local fix

Stage the verified scalar `body` before returning from `resolveInputs`:

```go
ref, err := app.Artifacts.Put(ctx, expectedSchema, mediaType, body)
if err != nil { ... }
if ref.Digest != resolvedInput.Reference.Digest ||
   ref.Size != *resolvedInput.Reference.SizeBytes {
    return contractError("RUNNER_INPUT_CUSTODY")
}
ret[name] = StagedInput{Schema: expectedSchema, Reference: &ref}
```

This is safe and small. It uses the same custody shape already used for set inputs.

#### Systemic fix

Do not make `Application.Submit` an ambiguous “sometimes stage a path, sometimes accept a reference” operation. Split ingestion from submission:

```go
// Product boundary: submission accepts immutable refs only.
func (a *Application) SubmitArtifacts(
    ctx context.Context,
    plan workflowv3.WorkflowPlan,
    inputs map[string]workflowv3.ArtifactRef,
    runID workflowv3.RunID,
) (Submission, error)

// CLI convenience boundary: explicitly converts paths to refs first.
func (a *Application) StagePathInputs(
    ctx context.Context,
    inputs map[string]PathInput,
    baseDir string,
) (map[string]workflowv3.ArtifactRef, error)
```

The Researchctl runner calls `ArtifactStore.Put` on verified bytes and then `SubmitArtifacts`. CLI commands may still use `StagePathInputs`, but no caller can accidentally claim verified custody while passing a mutable path.

#### Adjacent gap

`Config.MaxRequestBytes` bounds the JSON request framing, not the content of resolved path files. `readVerifiedInput` uses `os.ReadFile` and can allocate the entire external file. Add a distinct host limit such as `MaxResolvedInputBytes`; check the declared size and `os.Stat` before reading, and use a bounded reader when practical. Do not overload “request bytes” to mean both protocol frame and resolved artifact.

### 4.2 Finding 2 — set cardinality is guessed from direct maps

Review: [discussion_r3644764562](https://github.com/go-go-golems/scraper/pull/10#discussion_r3644764562)

#### Current behavior

`stageSetInput` decodes an archive, then initializes `maxItems := 0`. It scans only `plan.Maps` whose source is the named set input and chooses the minimum `MapPolicy.MaxItems` (`pkg/researchrunner/runner.go:378-390`). If no direct map consumes the set, `maxItems` remains zero and every archive is rejected.

The compiler supports more topologies than the runner admits:

- `IRReduce.Source` may be a direct `set-input` (`compiler.go:307-385`);
- `IRSetOutput.Value` may be a direct `set-input` because `setRefSchema` accepts it (`compiler.go:431-448`, `599-618`).

The runner therefore rejects plans that the compiler declares valid.

#### Local fix options

1. **Derive from every consumer.** Include direct maps and direct reductions. A reduction with `FanIn=f` and `MaxLevels=L` can reduce at most `f^L` source items without hitting `REDUCTION_LEVEL_LIMIT`; calculate this with overflow-safe saturation.
2. **Byte-only fallback.** If there is no processing consumer, allow pass-through while enforcing resolved-input and export byte limits.

This addresses the comment but leaves input semantics dependent on incidental consumers.

#### Recommended systemic fix: explicit ingress policy

A set input is an external admission boundary and must carry its own limit:

```go
type SetInputPolicy struct {
    MaxItems int `json:"maxItems"`
}

type IRSetInput struct {
    Name           string         `json:"name"`
    ItemSchema     string         `json:"itemSchema"`
    ManifestSchema string         `json:"manifestSchema"`
    Policy         SetInputPolicy `json:"policy"`
}
```

JavaScript:

```ts
const queries = p.inputSet<Query>("queries", {
  itemSchema: "query/v1",
  manifestSchema: "scraper-workflow-item-manifest/v1",
  maxItems: 10_000,
});
```

Rules:

- `MaxItems` is mandatory and positive.
- Runner admission checks only `IRSetInput.Policy.MaxItems` plus host byte limits.
- A direct map must have `MapPolicy.MaxItems >= input.MaxItems`.
- A chained map must support the upstream map's maximum cardinality.
- A direct reduction must have `reductionCapacity(FanIn, MaxLevels) >= source.MaxItems`.
- A pass-through set output remains bounded by the input policy and host export bytes.
- Multiple consumers do not silently redefine the external contract.

This makes bounds visible in IR, plan JSON, explanation output, TypeScript declarations, and plan digest.

#### Adjacent gap: pass-through set output cannot be read

The compiler accepts `SetOutput{Value: SetRef{Source:"set-input"}}`, but `Store.Snapshot` returns `unsupported set output source` for anything except `map-output` (`pkg/workflowv3sqlite/store.go:995-1010`). Once the runner limit is fixed, this hidden mismatch becomes visible. Add direct `set-input` resolution from `v3_run_inputs` and include it in the topology test matrix.

### 4.3 Finding 3 — data references do not reliably become readiness edges

Review: [discussion_r3644764564](https://github.com/go-go-golems/scraper/pull/10#discussion_r3644764564)

#### Current behavior

For an ordinary node, `ValidateIR` validates the `node-output` producer and schema, then separately validates `node.DependsOn` (`pkg/workflowv3/compiler.go:176-215`). It does not require or infer that the producer appears in `DependsOn`.

`Compile` copies `IRNode.DependsOn` unchanged into `PlanNode.DependsOn` (`compiler.go:481-493`). `Store.CreateRun` persists only that list into `v3_dependencies` (`pkg/workflowv3sqlite/store.go:174-188`). With capacity greater than one, the producer and consumer may both be leased. `ResolveInputs` then cannot find the output row (`store.go:540-587`), and the engine durably fails the consumer as `WORKFLOW_INPUT_RESOLUTION`.

#### Existing partial mitigation

Dynamic map and reduction nodes already inspect their template bindings during materialization and persist node-output dependencies:

- map children: `pkg/workflowv3sqlite/expansion.go:393-427`;
- reduction partition nodes: `pkg/workflowv3sqlite/reduction.go:292-319`.

They also persist gate consumers. This is good runtime behavior, but it is implemented independently of compiler cycle analysis and is absent for static nodes. The inconsistency itself is the systemic issue.

#### Why “reject unless `.after` is present” is not enough

Requiring explicit `.after` for every data ref would preserve duplicate authoring and cannot uniformly solve maps/reductions because their builders do not expose `after`. It also lets the plan say contradictory things: the binding says “consume producer P,” while the explicit schedule can omit P.

The better authoring rule is:

- bindings create **data dependencies automatically**;
- `.after(job)` creates **control-only ordering** when no value is consumed.

#### Recommended systemic fix: one compiler-owned typed graph

Build one graph over work kinds:

```go
type WorkRef struct {
    Kind string // node | map | reduction | gate
    Key  string
}

type DependencyEdge struct {
    Consumer WorkRef
    Producer WorkRef
    Reason   string // data | control | source | gate | budget
    Port     string // optional diagnostic path
}

type PlanAnalysis struct {
    Edges []DependencyEdge
}
```

Graph derivation:

```text
IRNode.DependsOn              -> node <- node       reason=control
ValueRef node-output          -> consumer <- node   reason=data
ValueRef gate-output          -> consumer <- gate   reason=data
ValueRef reduction-output     -> consumer <- reduce reason=data
IRMap.Source map-output       -> map <- map          reason=source
IRReduce.Source map-output    -> reduce <- map       reason=source
IRMap/IRReduce auxiliary refs -> template <- producer
Gate.DependsOn                -> gate <- node        reason=control
Budget approval gate          -> consumer <- gate   reason=budget
```

Then:

1. validate that every producer exists and every ref schema matches;
2. sort/deduplicate edges canonically;
3. reject cycles across **all** work kinds, not only node-only and node/gate subsets;
4. compile effective static node dependencies as the union of data and control node edges;
5. use the same analysis helper when materializing dynamic map/reduction nodes;
6. populate gate/reduction readiness tables from graph edges rather than rescanning ad hoc;
7. expose edge reasons in `Explain` and blocked-reason diagnostics.

A cross-kind cycle currently passes validation. For example:

```text
node publish consumes reduction R
reduction R partition task consumes node publish output

publish -> waits for R
R       -> waits for publish
```

`validateAcyclic` sees only explicit node dependencies; `validateGateAcyclic` sees nodes and gates but not reductions/maps (`compiler.go:620-707`). A typed graph rejects this before persistence.

#### Defense in depth

A leased node with an unresolved producer-backed input indicates store corruption or a scheduler invariant defect. The runtime should not permanently blame the task. Options:

- assert producer readiness transactionally during lease selection;
- classify impossible unresolved inputs as an internal retryable/infrastructure fault;
- emit a bounded invariant code and stop leasing the affected run until reconciled.

The primary fix remains correct graph persistence; the runtime check is defense, not the scheduling mechanism.

### 4.4 Finding 4 — no-work runs never receive a transition trigger

Review: [discussion_r3644764567](https://github.com/go-go-golems/scraper/pull/10#discussion_r3644764567)

#### Current behavior

The compiler requires at least one output but does not require work, so scalar pass-through is valid:

```text
input("source") -> output("source")
no nodes, maps, reductions, or gates
```

`CreateRun` inserts the run as `running`. Success transitions happen only after node completion, map publication, reduction publication, or gate approval. A pass-through workflow has none of those events, so `waitForTerminal` polls forever.

#### Local fix

At creation, set status to `succeeded` when there are no work items. This fixes the reported shape but duplicates the completion rules and can drift as new work kinds are added.

#### Recommended systemic fix: transaction-local terminal reconciler

Create one store helper:

```go
func reconcileRunState(
    ctx context.Context,
    tx *sql.Tx,
    runID workflowv3.RunID,
    now time.Time,
) (changed bool, err error)
```

Pseudocode:

```text
if run is already terminal:
    return unchanged

if any required node is not succeeded:
    return running
if any map is not published:
    return running
if any reduction is not published:
    return running
if any ordinary gate is not approved:
    return running
if any active budget gate blocks:
    return running
if any declared output cannot resolve:
    return invariant error

compare-and-set run running -> succeeded
if transition won:
    append run.succeeded exactly once
return
```

Call it:

- at the end of `CreateRun`, after inputs/work rows and `run.created` exist;
- after node completion;
- after map publication;
- after reduction publication;
- after gate approval or any future work-kind completion.

This removes four SQL copies and automatically handles zero-work plans. It also makes adding a future work kind safer: one predicate must be extended, not every transition method.

#### Product facade consequence

`Application.Submit` currently always returns `Status: "running"` (`pkg/workflowv3product/service.go:101-121`). If creation can immediately succeed, submission must return the persisted status. Either make `CreateRun` return status or read a snapshot before constructing `Submission`.

## 5. Cross-cutting diagnosis: one owner per invariant

The four findings share a design smell but should not be collapsed into one oversized subsystem.

| Invariant | Current competing owners | Authoritative owner after change |
|---|---|---|
| Executed input bytes equal experiment identity | Researchctl digest check; mutable path; artifact store digest | Artifact store ref produced directly from verified bytes |
| Set input cardinality is bounded | Direct map policy; runner fallback; reduction geometry; export bytes | Explicit set-input policy, validated against consumers |
| Consumer cannot run before producer | Value binding; explicit `.after`; dynamic persistence scans | Compiler-derived typed graph, lowered by store |
| Run is successfully terminal | Four transition-specific SQL copies; creation default | One transaction-local reconciler |

This is the systemic opportunity: **move each fact to the earliest layer that has enough information to declare it, and lower it exactly once into durable state**.

## 6. Proposed architecture

### 6.1 Ingress custody boundary

```text
external path
   |
   | bounded read + reference size/digest verification
   v
verified bytes (short-lived memory)
   |
   | ArtifactStore.Put
   v
immutable content-addressed ArtifactRef
   |
   | only representation accepted by workflow submission
   v
v3_run_inputs
```

Required invariant:

```text
ResearchctlReference.digest
  == sha256(bytes read once)
  == staged ArtifactRef.digest
  == v3_run_inputs.digest
```

Never pass a pathname after the first equality is established.

### 6.2 Explicit set contract

```text
SetInputPolicy.MaxItems
      | compile-time compatibility checks
      +----------------------+----------------------+
      v                      v                      v
MapPolicy.MaxItems     Reduce capacity f^L    pass-through output
      |                      |                      |
      +----------------------+----------------------+
                             v
                    runner admission check
```

Host byte limits remain separate from domain cardinality. Item count bounds graph/state growth; byte limits bound memory and artifact/export size.

### 6.3 Compiled dependency analysis

```text
IR bindings + control .after + sources + gates + budget gates
                             |
                             v
                  BuildPlanAnalysis (pure Go)
                    - validate producers
                    - derive typed edges
                    - canonicalize
                    - detect all cycles
                    - validate topology capabilities
                             |
                 +-----------+-----------+
                 v                       v
          compiled plan             store lowering
       effective dependencies   dependency/consumer rows
                                         |
                                         v
                                 lease readiness SQL
```

The graph is a compiler concept. SQLite may keep specialized tables for efficient readiness; those tables are projections of the graph, not independent semantics.

### 6.4 Central lifecycle reconciliation

```text
CreateRun ───────────────┐
CompleteNode ────────────┤
PublishMap ──────────────┤
PublishReduction ────────┼──> reconcileRunState(tx) ──> running | succeeded
ApproveGate ─────────────┤
future work transition ──┘
```

Failure and cancellation remain explicit authoritative transitions. Successful completion is derived from the durable work graph and output resolvability.

## 7. Topology capability matrix

The compiler and store currently disagree about some accepted shapes. Add one explicit matrix and test it at compile, submit, terminal, snapshot, and reopen layers.

| Shape | Compile | Admission | Scheduling | Snapshot/output | Expected terminal |
|---|---|---|---|---|---|
| scalar input -> scalar output | accept | stage ref | no lease | resolve `v3_run_inputs` | immediate succeeded |
| set input -> set output | accept with maxItems | stage manifest ref | no lease | resolve `v3_run_inputs` | immediate succeeded |
| scalar input -> node -> output | accept | stage ref | node ready | node output | succeeded |
| node A output -> node B input, no `.after` | infer data edge | n/a | B waits for A | B output | succeeded |
| set input -> map -> set output | validate max compatibility | enforce input max | map children | published expansion | succeeded |
| set input -> reduction -> scalar output | validate `maxItems <= f^L` | enforce input max | reduction partitions | published root | succeeded |
| map -> reduction -> node | infer typed edges | n/a | staged readiness | node output | succeeded |
| auxiliary node output bound into map/reduce task | infer template edge | n/a | dynamic children wait | normal | succeeded |
| cross-kind dependency cycle | reject | n/a | never persisted | n/a | compile error |
| empty map input | accept | max permits zero | no child leases | empty published manifest | succeeded |
| empty direct reduction input | define explicitly | enforce | currently validation failure | n/a | documented failure or identity policy |

The final row is an open product decision: reductions currently reject empty manifests with `REDUCTION_SOURCE_EMPTY`. Keep that explicit; do not accidentally change it while fixing pass-through workflows.

## 8. API sketches

### 8.1 Product submission

Preferred internal API after cleanup:

```go
type PathInput struct {
    Path      string
    Schema    string
    MediaType string
}

func (a *Application) StagePathInputs(
    ctx context.Context,
    inputs map[string]PathInput,
    baseDir string,
) (map[string]workflowv3.ArtifactRef, error)

func (a *Application) SubmitArtifacts(
    ctx context.Context,
    plan workflowv3.WorkflowPlan,
    refs map[string]workflowv3.ArtifactRef,
    runID workflowv3.RunID,
) (Submission, error)
```

Do not serialize raw bytes into plan or run rows. The artifact store remains the large-value boundary.

### 8.2 Set input policy

```go
type SetInputPolicy struct {
    MaxItems int `json:"maxItems"`
}

func ValidateSetInputPolicy(p SetInputPolicy) error
func ReductionCapacity(p ReducePolicy) (int, bool /* saturated */)
```

Use overflow-safe multiplication:

```text
capacity = 1
repeat MaxLevels times:
    if capacity > MaxInt / FanIn:
        return MaxInt, saturated
    capacity *= FanIn
```

### 8.3 Plan analysis

```go
type WorkKind string
const (
    WorkNode      WorkKind = "node"
    WorkMap       WorkKind = "map"
    WorkReduction WorkKind = "reduction"
    WorkGate      WorkKind = "gate"
)

type WorkRef struct { Kind WorkKind; Key string }
type DependencyReason string

type DependencyEdge struct {
    Consumer WorkRef
    Producer WorkRef
    Reason   DependencyReason
    Path     string // e.g. nodes.validate.bindings.dataset
}

type PlanAnalysis struct {
    Edges []DependencyEdge
}

func AnalyzeIR(ir WorkflowIR, catalog *Catalog) (PlanAnalysis, error)
func AnalyzePlan(plan WorkflowPlan) (PlanAnalysis, error)
```

`AnalyzePlan` is important because `Store.CreateRun` accepts a plan value and currently checks only digest and run inputs. It should validate that a hand-decoded or cross-process plan still satisfies structural invariants.

### 8.4 Terminal reconcile

```go
type RunReconcileResult struct {
    StatusChanged bool
    Status        string
}

func reconcileRunState(
    ctx context.Context,
    tx *sql.Tx,
    runID workflowv3.RunID,
    now time.Time,
) (RunReconcileResult, error)
```

Keep it transaction-local; do not query through `s.db` while a transition transaction is open.

## 9. Decision records

### Decision: stage verified bytes before submission

- **Context:** Researchctl identity is digest-based, but scalar staging rereads a mutable path.
- **Options considered:** lock the file; restat before staging; reread and rehash; stage the already verified bytes.
- **Decision:** stage the already verified bytes into the content-addressed artifact store and submit only the resulting ref.
- **Rationale:** it closes the TOCTOU window and matches the existing artifact boundary without platform-specific file locking.
- **Consequences:** scalar and set inputs use the same custody model; resolved-input byte limits must be explicit.
- **Status:** proposed.

### Decision: make set-input cardinality explicit

- **Context:** maps, reductions, and pass-through outputs admit different valid topologies; consumer inference rejects some and hides bounds for others.
- **Options considered:** infer from maps only; infer from every consumer; rely only on bytes; declare an ingress policy.
- **Decision:** add mandatory `SetInputPolicy.MaxItems` and validate every consumer against it.
- **Rationale:** external admission semantics belong to the input contract and must be visible in the canonical plan.
- **Consequences:** IR/plan JSON and TypeScript fixtures change; schema-version handling must be chosen before merge.
- **Status:** proposed.

### Decision: derive data dependencies; reserve `after` for control ordering

- **Context:** a binding and an explicit dependency describe the same data relationship twice and can disagree.
- **Options considered:** reject missing `.after`; infer only node-to-node dependencies; derive a typed graph over all work kinds.
- **Decision:** derive typed data/source/gate dependencies from refs, preserve `.after` only as a control edge, and validate one combined graph.
- **Rationale:** bindings already identify producers; the compiler is the first layer with full topology information.
- **Consequences:** plans may gain effective dependencies and new digests; explanations become more accurate; cross-kind cycles become compile errors.
- **Status:** proposed.

### Decision: centralize successful terminalization

- **Context:** completion SQL is copied across node, map, reduction, and gate transitions and absent at creation.
- **Options considered:** reject no-work plans; special-case creation; use a central reconciler.
- **Decision:** support pass-through plans and invoke one idempotent transaction-local reconciler after creation and every successful work transition.
- **Rationale:** successful status is a derived property of durable work and output availability.
- **Consequences:** duplicated SQL is removed; submission must return persisted status; success-event semantics become explicit.
- **Status:** proposed.

### Decision: do not block the PR on a storage schema unification

- **Context:** readiness currently uses specialized tables for node, reduction, and gate dependencies.
- **Options considered:** replace them immediately with one polymorphic dependency table; retain tables forever; centralize semantic derivation while keeping storage projections.
- **Decision:** keep efficient specialized tables in this corrective slice, but populate them from shared plan analysis. Consider physical unification only with measured query/migration value.
- **Rationale:** the defect is semantic duplication, not necessarily table layout.
- **Consequences:** the fix remains bounded while leaving a clean future migration path.
- **Status:** proposed.

## 10. Localized repair versus systemic implementation

### 10.1 Minimum safe PR repair

If review latency requires the smallest patch, do all of the following together:

1. stage verified scalar `body` directly to `ArtifactStore`;
2. add an explicit set-input max or, temporarily, a shared `SetInputLimit` helper covering direct map, direct reduction, and pass-through byte bounds;
3. infer node-output dependencies into compiled `PlanNode.DependsOn`;
4. extract the existing completion SQL to `reconcileRunState` and call it from `CreateRun` plus the four current transitions;
5. add direct set-input output resolution in `Snapshot`;
6. add regression tests for every review comment.

Even the minimum repair should not add more ad hoc SQL or runner-only topology assumptions.

### 10.2 Recommended corrective slice

Implement the four phases in Section 11. This is somewhat larger than four line edits but still localized to existing package boundaries. It pays down the exact duplication that caused the comments and prevents adjacent topology failures.

## 11. Phased implementation plan

### Phase 1 — close input custody and size boundaries

Files:

- `pkg/researchrunner/runner.go`
- `pkg/researchrunner/runner_test.go`
- `pkg/workflowv3product/service.go`
- `pkg/workflowv3product/application_test.go`

Steps:

1. Add a resolved-input byte limit distinct from protocol request size.
2. Read, size-check, hash, and stage scalar bytes exactly once.
3. Submit immutable refs to the engine.
4. Split path convenience staging from ref submission, or at minimum prevent verified callers from returning paths.
5. Test file replacement after verification and before submission.

Exit criterion: replacing the source pathname cannot change the workflow input digest or output.

### Phase 2 — freeze set-input ingress policy and topology support

Files:

- `pkg/workflowv3/types.go`
- `pkg/workflowv3/compiler.go`
- `pkg/workflowv3/compiler_test.go`
- `pkg/gojamodules/workflow/authoring.go`
- `pkg/gojamodules/workflow/authoring_test.go`
- `pkg/researchrunner/runner.go`
- `pkg/researchrunner/runner_test.go`
- generated/golden IR, plan, and TypeScript fixtures
- `pkg/workflowv3sqlite/store.go`

Steps:

1. Add `SetInputPolicy.MaxItems` to IR and plan.
2. Add `maxItems` to `inputSet` TypeScript/runtime API.
3. Validate direct/chained map and reduction capacity compatibility.
4. Enforce the input policy in runner archive staging.
5. Support direct set-input outputs in `Snapshot`.
6. Decide whether unreleased V3 fixtures are amended in place or schema constants are bumped.

Exit criterion: map-only, reduction-only, and pass-through set plans agree across compile, submit, terminal, snapshot, and reopen.

### Phase 3 — compile one effective dependency graph

Files:

- `pkg/workflowv3/compiler.go` (prefer extracting `analysis.go`)
- `pkg/workflowv3/types.go`
- `pkg/workflowv3/compiler_test.go`
- `pkg/gojamodules/workflow/authoring.go` and help/declarations
- `pkg/workflowv3sqlite/store.go`
- `pkg/workflowv3sqlite/expansion.go`
- `pkg/workflowv3sqlite/reduction.go`
- readiness/dispatcher integration tests

Steps:

1. Introduce pure graph derivation with diagnostic paths and edge reasons.
2. Infer data dependencies from all value/set refs.
3. Merge explicit control edges canonically.
4. Validate cycles across node/map/reduction/gate work kinds.
5. Use graph analysis to populate static and dynamic readiness projections.
6. Clarify in docs and TypeScript that `.after` is control-only.
7. Reclassify impossible post-lease missing producer output as an invariant failure.

Exit criterion: a data-binding-only workflow succeeds under multi-slot dispatch, and every constructed cross-kind cycle is rejected before persistence.

### Phase 4 — centralize lifecycle reconciliation

Files:

- `pkg/workflowv3sqlite/store.go`
- `pkg/workflowv3sqlite/expansion.go`
- `pkg/workflowv3sqlite/reduction.go`
- `pkg/workflowv3sqlite/gate.go`
- `pkg/workflowv3product/service.go`
- corresponding store/product/runtime tests

Steps:

1. Extract one transaction-local completion predicate.
2. Include output resolvability, including pass-through inputs.
3. Call it at creation and each successful transition.
4. Emit one durable `run.succeeded` event when the compare-and-set wins.
5. Return actual persisted status from product submission.
6. Add concurrent last-transition tests and reopen assertions.

Exit criterion: zero-work pass-through succeeds immediately; existing node/map/reduction/gate workflows retain terminal behavior; exactly one success transition/event is visible.

### Phase 5 — full regression and downstream validation

Commands:

```bash
go test ./pkg/workflowv3 ./pkg/gojamodules/workflow -count=1
go test ./pkg/workflowv3sqlite ./pkg/workflowv3runtime -count=1
go test ./pkg/workflowv3product ./pkg/researchrunner -count=1
go test ./... -count=1
go test -race ./pkg/workflowv3sqlite ./pkg/workflowv3runtime ./pkg/researchrunner -count=1
```

Also run the Researchctl cross-repository runner matrix from the owning ticket because custody and domain execution schema are cross-repository contracts.

## 12. Regression test design

### 12.1 Custody tests

- Verify valid scalar input is staged and executed.
- Replace the source file after verification; prove output and staged digest still reflect original bytes.
- Reject missing size, wrong size, and wrong digest with closed runner codes.
- Reject declared/actual input larger than `MaxResolvedInputBytes` before unbounded allocation.
- Ensure runner errors do not leak paths or source bytes.

### 12.2 Set-input tests

- Exact `MaxItems` succeeds; `MaxItems+1` fails.
- Direct map consumer accepts declared bound.
- Direct reduction accepts up to `FanIn^MaxLevels` and rejects incompatible plans at compile time.
- Direct set pass-through stages, immediately succeeds, snapshots, exports, and reopens.
- Multiple consumers must all support the ingress contract.
- Empty set pass-through succeeds; empty reduction retains its explicitly documented failure.

### 12.3 Dependency tests

- Node B binding A output without `.after` compiles to an effective edge and succeeds with capacity 2.
- `.after` without a data binding still orders tasks.
- Map item task auxiliary node-output binding waits for producer.
- Reduction partition task auxiliary node-output binding waits for producer.
- Gate and reduction outputs lower to readiness edges.
- Node-only, node-gate, node-reduction, map-reduction, and mixed cycles fail with paths naming both ends.
- Canonical edge ordering produces stable plan digests across map iteration order.

### 12.4 Lifecycle tests

- Scalar pass-through is `succeeded` immediately and output ref equals input ref.
- Set pass-through is `succeeded` immediately and output ref equals input manifest ref.
- `Submission.Status` reflects persisted status.
- Existing empty-map flow remains successful.
- Last node/map/reduction/gate transition calls the same reconciler.
- Concurrent final transitions produce one status change and one `run.succeeded` event.
- Reopen preserves status and outputs.

### 12.5 Model-oriented topology tests

Table-drive the matrix in Section 7. For each supported topology, assert:

1. compile result;
2. canonical plan digest stability;
3. input validation/admission;
4. expected first readiness state;
5. terminal status;
6. output resolution;
7. reopen equivalence.

This matrix is the strongest systemic prevention mechanism. It tests contracts across layers instead of proving each package only against its favorite topology.

## 13. Code review and implementation guidance for an intern

### 13.1 Reading order

1. `pkg/workflowv3/types.go:115-277` — understand refs, IR, and plan.
2. `pkg/gojamodules/workflow/authoring.go:230-575` — see how JavaScript becomes IR.
3. `pkg/workflowv3/compiler.go:1-707` — follow validation, compilation, and current cycle checks.
4. `pkg/workflowv3product/service.go:38-156` — see path/ref staging and submission.
5. `pkg/workflowv3sqlite/store.go:76-278` — see run lowering.
6. `pkg/workflowv3sqlite/store.go:280-510` — see admission/readiness.
7. `pkg/workflowv3sqlite/store.go:540-687` — see resolution and completion.
8. `pkg/workflowv3sqlite/expansion.go:180-430` and `reduction.go:170-325` — see dynamic lowering.
9. `pkg/workflowv3runtime/dispatcher.go:40-130` and `engine.go:255-430` — see concurrent dispatch and failures.
10. Tests listed in Section 12.

### 13.2 Invariants to preserve while editing

- Plan digest must cover every execution-semantic field.
- Map/reduction child identities remain deterministic.
- No source bytes enter workflow SQLite; only bounded refs and canonical metadata do.
- Lease admission remains transactionally authoritative.
- A late/stale worker cannot publish after losing authority.
- Existing budget, gate, isolation, and external-operation semantics remain unchanged.
- Error messages exposed across the runner remain closed and do not contain sensitive paths/content.

### 13.3 What not to do

- Do not sleep/retry input resolution to hide missing dependencies.
- Do not require authors to mirror every binding with `.after`.
- Do not let `maxItems == 0` mean unbounded.
- Do not use map consumer policy as the only external set-input contract.
- Do not insert a creation-only completion predicate separate from transition predicates.
- Do not put raw verified bytes into the plan JSON or SQLite run rows.
- Do not rewrite dependency storage and lifecycle storage in one unreviewable migration.

## 14. Risks and mitigations

### Plan digest and schema churn

Adding set policy or compiled dependencies changes canonical digests. If Workflow V3 has not shipped outside this PR, amend V3 fixtures and reject old artifacts. If plans are already durable/public, bump IR/plan schema and document migration. Do not silently decode two semantic shapes under one schema without a deliberate compatibility decision.

### Reduction-capacity overflow

`FanIn^MaxLevels` can overflow `int`. Use saturating arithmetic and compare before multiplication. The host still needs policy ceilings so “mathematically huge” does not become an accepted operational bound.

### Reconciler races

Multiple final transitions may concurrently conclude that a run is complete. Use `UPDATE ... WHERE status='running'` and rows-affected to select one winner; only the winner appends `run.succeeded`.

### Artifact staging memory

Staging verified bytes directly closes custody but retains a whole input in memory. It does not worsen the current `os.ReadFile` behavior, but a later streaming artifact API may be justified. Streaming must hash and publish atomically; do not reintroduce path trust.

### Cross-kind graph complexity

Typed graph analysis can grow opaque if mixed with SQL. Keep it pure, deterministic, and table-driven in `pkg/workflowv3`; keep SQL as a lowering target.

## 15. Alternatives considered

### Reject pass-through plans

This avoids zero-work terminalization but removes a valid composition primitive and does not solve duplicated completion SQL. Rejected.

### Require explicit `.after` everywhere

This is simple for static tasks but duplicates dataflow intent and does not naturally cover map/reduction templates. Rejected.

### Infer set limits only from consumers

This can unblock direct reductions but still makes external admission depend on topology and gives pass-through no semantic cardinality contract. Acceptable only as a temporary bridge, not the final design.

### Replace every dependency table now

A polymorphic dependency table may eventually simplify queries, but it adds migration and performance risk unrelated to the immediate correctness defects. Defer physical unification; unify semantic derivation now.

### Mark no-work runs succeeded directly in `CreateRun`

Correct for one shape, but it creates another copy of completion semantics. Use the shared reconciler instead.

## 16. Open questions

1. Has Plan/IR V3 been consumed outside this PR in a way that requires a schema bump rather than amending the unreleased contract?
2. What host ceiling should constrain `SetInputPolicy.MaxItems`, independent of author-supplied values?
3. Should `MaxResolvedInputBytes` be one global limit, separate scalar/set limits, or derived from the artifact store ceiling?
4. Should `run.succeeded` become a new durable event, and do downstream observation digests need a coordinated schema update?
5. Should impossible post-lease input resolution fail the run or quarantine it for operator repair? The scheduler must not attribute it to user task execution.
6. Is empty reduction input intentionally invalid for every domain, or should reduction policy eventually support an identity value?

None of these questions blocked fixing the four reviewed defects. The unreleased V3 contract was amended in place and `run.succeeded` remains internal evidence unless a future versioned observation change promotes it.

## 17. Final recommendation

Treat the comments as **four manifestations of missing invariant ownership**, not as four unrelated conditionals. Implement a focused corrective slice with four authoritative boundaries:

1. verified bytes become immutable refs immediately;
2. set inputs declare their own cardinality;
3. the compiler derives one effective dependency graph;
4. the store reconciles successful terminal state in one place.

This approach is systemic enough to prevent adjacent failures but bounded enough for PR 10. The implementation should be split into custody, set-policy, graph, and reconciliation commits, each with focused tests, followed by the topology matrix and full cross-repository validation.

## 18. Implementation outcome

The four corrective slices described by this design were implemented and committed:

| Invariant | Code commit | Result |
|---|---|---|
| Verified byte custody | `5486f2e` | Bounded single read, immediate content-addressed staging, immutable-ref submission |
| Set-input cardinality | `2dfdee1` | Explicit `maxItems`, compiler consumer/capacity checks, runner enforcement, pass-through set projection |
| Dependency readiness | `b0cdd1b` | Data-derived canonical dependencies, typed cross-kind cycle validation, plan validation, shared dynamic lowering |
| Successful terminalization | `7f6e728` | One transaction-local reconciler, output readiness, immediate pass-through success, accurate submission status |

All five decision records in Section 9 are accepted for the current unreleased Workflow V3 contract. The implementation retained specialized SQLite readiness tables while centralizing semantic derivation, as proposed.

Final evidence:

- `go test ./... -count=1` passed;
- `go build ./...` passed;
- `go test -race ./pkg/workflowv3sqlite ./pkg/workflowv3runtime ./pkg/researchrunner -count=1` passed;
- golangci-lint reported `0 issues`;
- every code commit's pre-commit hook also ran `GOWORK=off go test ./... -count=1` and lint successfully;
- `docmgr doctor --ticket SCRAPER-PR10-SYSTEMIC-REVIEW --stale-after 30` passed.

The remaining questions in Section 16 are future policy/design topics, not blockers for the four PR findings. In particular, empty reduction remains explicitly invalid, a future host cardinality ceiling remains optional hardening, and the added `run.succeeded` event may be promoted into external observation vocabulary only through a versioned follow-up.

## References

### GitHub review

- PR: https://github.com/go-go-golems/scraper/pull/10
- Review: https://github.com/go-go-golems/scraper/pull/10#pullrequestreview-4772468539
- Verified bytes: https://github.com/go-go-golems/scraper/pull/10#discussion_r3644764557
- Set-input consumers: https://github.com/go-go-golems/scraper/pull/10#discussion_r3644764562
- Producer dependencies: https://github.com/go-go-golems/scraper/pull/10#discussion_r3644764564
- Zero-work completion: https://github.com/go-go-golems/scraper/pull/10#discussion_r3644764567

### Primary source files

- `pkg/researchrunner/runner.go:303-403`
- `pkg/researchrunner/types.go:1-110`
- `pkg/workflowv3/artifacts.go:43-126`
- `pkg/workflowv3/types.go:115-277`
- `pkg/workflowv3/compiler.go:176-448`
- `pkg/workflowv3/compiler.go:468-707`
- `pkg/gojamodules/workflow/authoring.go:230-575`
- `pkg/gojamodules/workflow/authoring.go:710-780`
- `pkg/workflowv3product/service.go:38-156`
- `pkg/workflowv3sqlite/store.go:76-278`
- `pkg/workflowv3sqlite/store.go:280-510`
- `pkg/workflowv3sqlite/store.go:540-687`
- `pkg/workflowv3sqlite/store.go:950-1010`
- `pkg/workflowv3sqlite/expansion.go:180-430`
- `pkg/workflowv3sqlite/reduction.go:1-325`
- `pkg/workflowv3runtime/dispatcher.go:40-130`
- `pkg/workflowv3runtime/engine.go:100-250`
- `pkg/workflowv3runtime/engine.go:255-430`

### Existing tests

- `pkg/researchrunner/runner_test.go:1-290`
- `pkg/workflowv3/compiler_test.go:1-290`
- `pkg/workflowv3sqlite/store_test.go:1-330`
- `pkg/workflowv3sqlite/expansion_test.go:180-270`
- `pkg/workflowv3runtime/dispatcher_test.go`
- `pkg/workflowv3runtime/reduction_integration_test.go`

### Existing architecture and history

- `ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/design-doc/01-durable-dataflow-workflow-v3-and-modern-scripting-architecture.md`
- `ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/design-doc/08-workflow-v3-slices-1-through-12-intern-architecture-and-analysis-guide.md`
- `ttmp/2026/07/22/SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER--make-workflow-v3-the-sole-public-scraper-workflow-product/reference/02-researchctl-runner-implementation-diary.md`
- `a8126d9` / `917e5b6` — add Researchctl Workflow V3 runner
- `842303c` / `ebf9a85` — support domain task packages and set inputs
- `601c0a1` / `ff286a1` — freeze core IR and vertical slices
- `d0afd47` / `9d714e6` — compose tasks after reductions
