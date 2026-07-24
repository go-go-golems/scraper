---
Title: Slice 12 RAG TTC - Safe Real-Provider Production Workload
Ticket: SCRAPER-WORKFLOW-V3
Status: active
Topics:
    - architecture
    - artifacts
    - worker
    - scraper
    - workflows
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/researchctl/pkg/lab
      Note: Generic experiment custody and immutable evidence boundary
    - Path: abs:///home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system/pkg/ragproviders/provider_set.go
      Note: Provider and model profile authority
    - Path: repo://pkg/workflowv3/types.go
      Note: Generic exact plans refs resources retries budgets gates and isolation
    - Path: repo://pkg/workflowv3runtime/dispatcher.go
      Note: Work-conserving heterogeneous execution
ExternalSources: []
Summary: Implementation and operating contract for migrating the TTC real-provider study onto compact Workflow V3 execution, proving preflight, then publishing and evaluating a fresh admissible run.
LastUpdated: 2026-07-22T01:15:00-04:00
WhatFor: Freeze the final integrated acceptance workload before modifying RAG execution or spending provider budget.
WhenToUse: Read before implementing the Workflow V3 RAG adapter, running TTC preflight, launching provider work, publishing prepared artifacts, evaluating quality, or writing the final report.
---


# Slice 12 RAG TTC - Safe Real-Provider Production Workload

## Executive summary

Slice 12 is complete only when a fresh, exact-profile TTC study runs through
Workflow V3 with real providers, publishes and reopens immutable prepared
artifacts, evaluates all frozen queries, records metrics/costs/citations in
researchctl, and produces a redacted evidence-backed report. It is the
integrated acceptance test for Slices 1–11, not permission to bypass them.

The diagnostic v9 databases and files under `/tmp/rag-ttc-full-v9*` are
non-publishable. They contain source-bearing durable inputs and cannot be
resumed, imported, copied into artifacts, summarized from raw database content,
or cited as final measurements.

## Authority boundaries

### Scraper Workflow V3

Owns only generic execution concerns:

- exact plans, bundles, registry generation, and isolation identity;
- compact artifact refs, maps, reductions, resources, attempts, retry, fencing;
- transactional budgets, gates, cancellation, events, and projections;
- validated refs and generic artifact publication mechanics.

It does not understand TTC, prompts, chunks, embeddings, relevance, providers,
models, citations, or quality metrics.

### RAG repository

Owns:

- TTC source/evaluation materialization and lineage;
- chunk/unit identity and preparation semantics;
- model/prompt/schema manifests and provider construction;
- generation, embedding, index, retrieval, rerank, answer, citation, and quality
validation;
- domain task bundles and narrow trusted host modules;
- prepared corpus/index publication and fresh-process reopen.

The RAG integration binary composes scraper's public Workflow V3 runtime with
RAG-owned bundles/modules. Scraper must not import the RAG repository.

### Researchctl

Owns immutable study/specification/run identity, generic artifact custody,
observations, metrics, usage/cost, citations, attachments, export/import, and
report evidence. It is not a second task scheduler and does not persist provider
credentials or source bodies.

## Frozen study identity

Before execution, produce a signed/redacted run manifest that freezes:

- source corpus snapshot ID and complete file digest/size;
- evaluation dataset ID, adjudication revision, query count, and digest;
- chunker/unitizer/representation/index/retrieval/rerank/answer specifications;
- model, prompt, schema, and provider-profile manifest digests;
- exact task bundle digests, entrypoints, ABI, module aliases, resource classes,
retry policies, isolation policies, registry generation, and plan digest;
- budget accounts/limits, approval gate policy, concurrency, and run ID;
- researchctl project/study/specification IDs and repository commits;
- machine/runtime versions and redacted endpoint identities.

A changed identity creates a new run. “Latest,” mutable tags without resolved
manifest evidence, ambient provider config, and undocumented host clamping are
invalid.

## Workflow graph

The target graph is generic Workflow V3 composition of RAG-owned tasks:

```text
verified corpus envelope ref
  → materialize ordered 1,807-item chunk manifest
  → lazy generation map
       resource generation.remote
       requests/input_tokens/output_tokens/cost budget
       typed malformed-output retry before publication
  → lazy embedding map
       resource embedding.local
       embedding_tokens/input_bytes budget
  → bounded prepared-shard reduction
       exact key order and bounded fan-in
  → validate complete prepared corpus/index
  → optional cost/publication approval gate
  → idempotent atomic publication
  → fresh-process reopen
  → ordered query evaluation map (all frozen queries)
       retrieval, rerank, answer, citation validation
  → bounded metrics/cost/citation reductions
  → researchctl immutable evidence attachment
```

Generation and embedding capacities refill independently. Dynamic identity is
based on canonical source keys and immutable source digest, never completion
order. Provider task payloads are rehydrated from local immutable artifacts
inside the attempt and are never copied into Workflow SQLite.

## RAG task contract

Each RAG bundle task declares strict input/output schemas, complete exact
identity, narrow modules, resource class, retry maximum, budget maximum, and
isolation profile. Representative kinds are:

- `rag.ttc.materialize-chunks/v1`;
- `rag.ttc.generate-representations/v1`;
- `rag.ttc.embed-representations/v1`;
- `rag.ttc.reduce-prepared-shard/v1`;
- `rag.ttc.publish-prepared/v1`;
- `rag.ttc.reopen-prepared/v1`;
- `rag.ttc.evaluate-query/v1`;
- `rag.ttc.reduce-study-evidence/v1`.

Provider aliases are trusted Go factories configured at worker boot. JavaScript
selects manifest IDs but receives no endpoint, credential, raw HTTP, arbitrary
filesystem, SQL, or process authority. RAG adapters convert provider errors to
closed failures before returning to Workflow V3.

`RAG_GENERATOR_COMBINED_JSON` and other provider-originated generated-response
shape failures are `malformed-output` and retryable only when validation fails
before cache/artifact publication. Configuration/schema/programmer failures are
terminal. Retries retain immutable attempt evidence and spend newly reserved
budget.

## Compact data and privacy

Workflow SQLite, WAL, events, logs, projections, researchctl observations, and
reports may contain only bounded identities, counts, timings, failure codes,
usage/cost integers, metrics, citation IDs, and compact refs. They may not
contain:

- credentials, authorization/cookie headers, provider config, or secret URLs;
- source records/chunks, prompts, rendered requests, provider response bodies;
- vectors, prepared shard bodies, arbitrary SQL, or database contents;
- raw errors/stderr likely to echo any of the above.

Preflight plants unique canaries in source, prompt, vector fixture, provider
body, credential, and endpoint fields and scans SQLite main/WAL/SHM, event and
projection JSON, logs, researchctl export, and report sources. Artifact bodies
may contain domain data only in the explicitly classified immutable artifact
root; the evidence manifest records classification and digest, never copies the
body.

## Preparation validation and publication

Every generated item is validated for exact source key, cardinality, schema,
and lineage before entering cache/artifact storage. Every embedding validates
model/dimension/count/finite values before publication. Reduction validates no
missing/duplicate keys and canonical ordering.

Publication follows an idempotent two-phase side-effect contract:

1. write immutable content-addressed component files;
2. validate complete cardinality, schema, lineage, and digest closure;
3. atomically create one release manifest/pointer under a stable operation key;
4. if the pointer already exists, verify byte-identical identity and return it;
5. reopen from a fresh process with no in-memory preparation state;
6. only then permit evaluation.

Crash before pointer publication leaves harmless unreferenced immutable files.
Crash after publication and before workflow completion retries against the same
operation key and cannot create a second logical release.

## Budgets, approvals, and cost

All dimensions use checked nonnegative integers. Provider calls reserve worst
case requests/tokens/cost in the lease transaction. Validated usage settles
actual units; ambiguous timeout/lease loss charges conservatively. No full run
starts without explicit account limits covering generation, embedding,
reranking, and answer stages.

Preflight uses fixture providers and zero external cost. A bounded real-provider
sample has a separate low limit. Its reviewed actual cost/latency/error/output
evidence is required before a versioned approval gate authorizes the full
account increase/publication policy. Approval does not fabricate budget.

## Required preflight ladder

### P0 — static identity

- clean/pinned repository commits and generated bundle/catalog/plan digests;
- exact source/evaluation/model/prompt/schema/provider identities;
- no mutable/unresolved aliases or missing operator policy;
- compile direct Go and JavaScript plans to equal goldens.

### P1 — deterministic fixture providers

- all 1,807 source items, no duplicate/missing keys;
- capacities 1 and production concurrency yield equal manifests/results;
- forced malformed generation returns typed retry and publishes only valid
second-attempt output;
- independent generation/embedding timeline proves immediate refill;
- deterministic reduction/publication digest;
- storage amplification and all privacy canary scans pass.

### P2 — durability/failure

Restart at four boundaries: generation, embedding, reduction, and after
publication pointer commit. Prove exact attempt history, no duplicated provider
logical operation where idempotency applies, conservative budget settlement,
continued cardinality/order, fresh publication reopen, and stale completion
rejection. Also prove cancellation, lease loss, child/provider death, unrelated
run failure isolation, and registry generation pinning.

### P3 — bounded real-provider sample

Use the exact full profile but a frozen small sample. Record provider/model
health, output validation, citations, usage/cost, retries, latency distributions,
redaction, publication/reopen, and researchctl export/import. Review evidence
and approve the full account increase through the durable gate.

### P4 — full real-provider study

Launch under a fresh run/database/artifact identity. Monitor authoritative
projections and bounded events without querying/copying payload bodies. Do not
alter policy in place. Any identity/policy change creates a new run. Completion
requires terminal success, complete publication/reopen, and all frozen queries.

## Evaluation and evidence

For every query, persist bounded observation identity and validated citation
IDs plus retrieval/rerank/answer status, latency, usage, and cost. Compute and
report at least precision@k, recall@k, MRR, nDCG@k, hit rate, citation coverage
and validity, provider retry/error rate, latency distributions, throughput,
resource utilization, and total/per-stage cost. Aggregations must be
reproducible from immutable researchctl observations.

The report includes:

- exact identity manifest and repository commits;
- preflight and sample evidence;
- preparation and evaluation timelines;
- quality/citation/cost tables and graphs;
- retry/failure/restart/accounting evidence;
- publication manifest and fresh reopen proof;
- explicit limitations without weakening acceptance claims;
- commands/scripts that regenerate redacted metrics and figures.

Researchctl export/import must reconstruct evidence byte-identically or by its
canonical identity contract. Published documents and reMarkable bundles contain
no diagnostic v9 payload/database material.

## Migration and compatibility

The existing RAG durable preparation remains read-only historical evidence while
the Workflow V3 adapter is proven. Do not translate v2 raw operations or import
v9 state. New runs use new task/plan/storage identities. Existing scraper v2,
Workflow V3 Slices 1–11, researchctl projects, and RAG public contracts remain
compatible unless an explicit versioned migration is documented and tested.

## Test matrix

- canonical RAG bundle/catalog/plan/registry identity and direct JS parity;
- 1,807-item fixture-provider execution, concurrency equality, storage/privacy;
- typed malformed-output retry, provider timeout, cancellation, lease loss,
stale fencing, and failure isolation;
- independent resource refill timeline;
- dynamic budget claims and conservative/actual settlement;
- publication crash-before/crash-after idempotency and fresh-process reopen;
- complete citation and relevance-lineage validation;
- researchctl attachment/export/import identity;
- bounded real-provider sample and full-run operational checks;
- repository tests/lint/race/type/build/help/migration/generated/doc validation
across every modified repository.

## Acceptance criteria

Slice 12 is complete only when every preflight stage passes, a reviewed bounded
sample authorizes a fresh full run, the exact 1,807-item preparation and all
frozen queries finish under real providers, immutable publication reopens in a
fresh process, researchctl contains reproducible complete quality/citation/cost
evidence, privacy and storage bounds hold, the report and graphs are validated
and published, diagnostic v9 material is excluded, all affected repositories
pass fresh validation, and every requirement maps to concrete files, commands,
artifacts, logs, and digests. Missing credentials, endpoints, provider capacity,
or an approval decision is a blocker to report—not permission to claim partial
completion.
