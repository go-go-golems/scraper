---
Title: Workflow v3 JavaScript cookbook and execution atlas
Ticket: SCRAPER-WORKFLOW-V3
Status: active
Topics:
    - architecture
    - scheduler
    - goja
    - javascript
    - scraper
    - workflows
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/engine/scheduler/scheduler.go
      Note: Current scheduler baseline contrasted with continuous per-node dispatch walkthroughs
    - Path: repo://pkg/workflow/context.go
      Note: Current durable task/result/emission boundary contrasted with v3 typed tasks
    - Path: repo://pkg/workflow/package.go
      Note: Current Go package and initial-step builder baseline for job mapping
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/design-doc/01-durable-dataflow-workflow-v3-and-modern-scripting-architecture.md
      Note: Defines the proposed DSL contracts and durable execution invariants used by every example
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/scripts/03-workflow-dsl-grammar-probe.mjs
      Note: Original executable grammar probe that informed cookbook syntax
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/scripts/04-check-cookbook-js.py
      Note: Extracts and syntax-checks every JavaScript fence
ExternalSources: []
Summary: A broad, non-RAG cookbook of proposed workflow-v3 JavaScript definitions with exact transformation, compilation, durable job/node mapping, leasing, attempt, and execution walkthroughs.
LastUpdated: 2026-07-21T17:45:00Z
WhatFor: Pressure-test and teach the workflow-v3 DSL by showing how varied JavaScript workflows become normalized IR, compiled jobs, durable nodes, attempts, artifacts, and projections.
WhenToUse: Read when implementing the workflow module/compiler, writing workflow definitions, adding task providers, or reviewing how authored scripts become durable execution.
---


# Workflow v3 JavaScript cookbook and execution atlas

## Goal

This document gives workflow-v3 implementers and authors a substantial set of JavaScript examples outside the RAG domain. Each example shows four distinct things:

1. the JavaScript an author writes;
2. the normalized workflow jobs that Go derives from it;
3. the durable nodes and attempts scraper materializes;
4. the worker/runtime behavior that executes those nodes and commits outputs.

The examples cover linear pipelines, web scraping, paginated API synchronization, joins, quality gates, media processing, map/reduce analytics, security scans, batch machine learning, notification fan-out, backups, inventory reconciliation, build pipelines, approvals, and monitoring matrices.

## Important status: target API, not current production API

These scripts are **executable design examples for the proposed workflow-v3 API**. They are not expected to run against scraper v2 today. In particular:

- `require("workflow")` and the example domain modules are proposed native modules;
- task factories such as `web.tasks.fetch(...)` return portable task descriptors; they do not perform network access while the authoring script runs;
- `workflow.compile(...)` is pure with respect to execution;
- trusted Go host code, not the authoring script, submits the compiled plan;
- some examples deliberately exercise proposed gate/condition APIs so implementation gaps are visible before the DSL is frozen.

The authoritative architecture and invariants remain in [the primary design](../design-doc/01-durable-dataflow-workflow-v3-and-modern-scripting-architecture.md). If an example conflicts with those invariants, the invariant wins and the example must be corrected.

## Reproducible syntax validation

All JavaScript fences are extracted and checked with Node's parser:

```bash
ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/scripts/04-check-cookbook-js.py \
  --doc ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/reference/03-workflow-v3-javascript-cookbook-and-execution-atlas.md \
  --out ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/scripts/output/workflow-cookbook-js-check.json
```

The current result is 17 of 17 JavaScript blocks passing syntax validation. This proves syntax only. Runtime/IR/plan behavior remains a proposed contract until the native modules, task catalogs, compiler, and golden tests are implemented.

## The vocabulary: definition, job, node, attempt, and task

These terms are easy to conflate. They describe different layers.

| Term | Cardinality | Created when | Meaning |
|---|---:|---|---|
| Authoring script | one source file | developer time | JavaScript that composes workflow intent |
| Workflow definition | one per normalized authored workflow | authoring runtime | Portable immutable IR before host policy binding |
| Compiled plan | one per definition + target capability profile | compile time | Validated jobs with effective resources, task versions, schemas, retry/budget ceilings, and digest |
| Job | one template in definition/plan | compile time | A static task, map template, reduction template, or gate |
| Node | zero, one, or many per job per run | submit/expansion time | One durable unit with bound compact input references |
| Attempt | one or more per node | lease time | One worker execution under one lease and resource/budget reservation |
| Task descriptor | one per job | authoring/compile time | Portable `kind@version`, schemas, and compact configuration |
| Runner | host implementation | worker startup | Go or lease-scoped JS implementation registered for a task kind/version |

A map job is the clearest illustration:

```text
JavaScript: p.map("thumbnail", images, image => media.tasks.thumbnail({image}))
                              │
                              ▼
Compiled plan: one job template, key="thumbnail", mode="map"
                              │
                 input set contains 10,000 ImageRef values
                              ▼
Durable run: up to 10,000 nodes, one per canonical item key
                              │
                 two nodes each retry once
                              ▼
Attempt ledger: 10,002 attempts
```

Jobs are not worker goroutines. Nodes are not attempts. A plan with six jobs may create millions of nodes lazily, while each node can create several immutable attempts.

## Universal transformation and execution pipeline

Every example follows the same pipeline regardless of domain.

```text
1. Load authoring modules
   require("workflow"), require("web"), require("data"), ...
                           │
2. Invoke workflow.define callback immediately
   Go-backed builders create hidden typed handles
                           │
3. Normalize to Workflow IR v3
   functions disappear; symbolic expressions and task descriptors remain
                           │
4. Validate IR
   names, cycles, schemas, ports, compactness, task config, resource requests
                           │
5. Compile against target profile
   bind task versions/runners/resources; clamp policy; record requested/effective
                           │
6. Trusted host submits compiled plan + immutable input bindings
   attach to same identity or fail on mismatch
                           │
7. Store creates run and static/lazy-expansion state
   compact refs only
                           │
8. Dependencies become ready
   continuous dispatcher admits a node under resource/rate/budget policy
                           │
9. Lease transaction creates attempt
   node running + lease + attempt + resource/budget reservation, atomically
                           │
10. Worker runs registered task implementation
    resolve allowed refs, perform bounded work, validate output, write artifact refs
                           │
11. Completion transaction
    check lease/cancel epoch; commit outputs/attempt/node/resource/budget/events
                           │
12. Expansion/reduction/projection advances
    downstream nodes become ready; snapshots and metrics reflect committed truth
```

### What happens to JavaScript callbacks

Consider:

```js
const pages = p.map("fetch", urls, url =>
  web.tasks.fetch({ url }),
  j => j.resource("internet").timeout("30s"));
```

The `workflow` native module invokes both callbacks during authoring:

- the map callback is called **once** with a symbolic `ValueRef<URLRef>`;
- `web.tasks.fetch` returns a versioned task descriptor whose input contains that symbolic reference;
- the job configurator is called **once** against a Go-backed `MapJobBuilder`;
- the builder emits normalized data;
- neither callback is retained, serialized, or replayed by workers.

A simplified normalized job is:

```json
{
  "key": "fetch",
  "mode": "map",
  "from": [{"job": "$input", "port": "urls"}],
  "item": {"name": "$item", "schema": "web-url-ref/v1"},
  "task": {
    "schemaVersion": "scraper-task-descriptor/v1",
    "kind": "web.fetch",
    "version": "v1",
    "inputSchema": "web-fetch-input/v1",
    "outputSchema": "web-fetch-output/v1",
    "config": {"url": {"$ref": "map-item", "source": "urls"}}
  },
  "requestedPolicy": {
    "resource": "internet",
    "timeout": "30s"
  }
}
```

At compile time the host may bind `internet` to `http.public.egress`, select runner `web.fetch/v1`, cap requested concurrency, and add a host-required retry ceiling. The compiled plan records both requested and effective policy.

### What the authoring modules may and may not do

A domain authoring module may:

- validate task configuration;
- return task descriptors;
- expose typed reference/schema helpers;
- explain requirements and outputs.

It may not:

- fetch a URL;
- open a database;
- read an artifact;
- use provider credentials;
- submit a run;
- access scraper's store;
- retain the live Goja runtime in durable state.

Actual work belongs to a registered runner invoked under a lease.

## Common example capability profile

The examples refer to symbolic resources. A compile host might expose this public profile:

```yaml
profile: workstation-v3
resources:
  control.local:
    maxInFlight: 2
  http.public.egress:
    maxInFlight: 8
    rate: {requestsPerMinute: 240, burst: 8}
  http.partner-api:
    maxInFlight: 3
    rate: {requestsPerMinute: 60, burst: 3}
  cpu.transform:
    maxInFlight: 6
  cpu.security-scan:
    maxInFlight: 2
  media.ffmpeg:
    maxInFlight: 2
  gpu.inference:
    maxInFlight: 1
  db.source.read:
    maxInFlight: 2
  db.destination.write:
    maxInFlight: 1
  storage.object:
    maxInFlight: 4
  notification.email:
    maxInFlight: 2
    rate: {requestsPerMinute: 30, burst: 2}
  notification.chat:
    maxInFlight: 4
  operator.approval:
    maxInFlight: 100
```

The workflow can request lower limits but cannot raise these ceilings. Credentials, endpoint secrets, filesystem paths, and database DSNs are not part of the public profile or compiled plan.

## Common task modules used below

These are proposed examples of domain-owned native modules. Each `tasks.*` function returns a task descriptor.

| Module | Example task kinds | Worker authority |
|---|---|---|
| `web` | `web.fetch`, `web.extract-links`, `web.parse-records` | bounded HTTP and HTML artifact resolution |
| `api` | `api.enumerate-pages`, `api.fetch-page`, `api.apply-mutation` | named partner APIs through host clients |
| `data` | `data.normalize`, `data.join`, `data.aggregate`, `data.validate` | compact refs and content-addressed datasets |
| `files` | `files.enumerate`, `files.convert`, `files.bundle` | named artifact stores, not arbitrary host paths |
| `media` | `media.probe`, `media.transcode`, `media.thumbnail` | sandboxed media tools and object storage |
| `analytics` | `analytics.token-count`, `analytics.reduce-counts` | CPU and compact shard manifests |
| `security` | `security.scan`, `security.evaluate-policy`, `security.sign-report` | sandboxed scanners and signing service |
| `ml` | `ml.preprocess-image`, `ml.classify-image`, `ml.aggregate-predictions` | named model profile and GPU resource |
| `notify` | `notify.render`, `notify.email`, `notify.chat`, `notify.receipt` | configured channel clients |
| `database` | `database.snapshot-shard`, `database.restore-verify` | named source/destination handles |
| `build` | `build.checkout`, `build.test`, `build.package`, `build.sign` | sandboxed command runner and artifact store |
| `ops` | `ops.await-approval`, `ops.probe`, `ops.evaluate-slo` | operator signal or bounded network probe |

The table is an API design aid, not a claim that these modules already exist.

# Example 1 — Minimal linear transform

## Intent

Normalize one input dataset, validate it, and publish a compact output manifest. This is the smallest useful durable pipeline.

## JavaScript

```js
const workflow = require("workflow");
const data = require("data");

const definition = workflow.define("normalize-customer-export", p => {
  const source = p.input("source", {
    schema: "customer-export-ref/v1",
    role: "source-export",
  });

  p.resource("cpu", r => r
    .class("cpu.transform")
    .maxInFlight(2));

  const normalized = p.task("normalize",
    data.tasks.normalizeCustomers({source}),
    j => j
      .resource("cpu")
      .timeout("2m")
      .retry({maxAttempts: 2, classes: ["internal"]}));

  const validated = p.task("validate",
    data.tasks.validateDataset({dataset: normalized.output("dataset")}),
    j => j.after(normalized).resource("cpu").timeout("1m"));

  p.output("dataset", validated.output("validatedDataset"));
});

const report = workflow.validate(definition);
if (!report.ok) throw new Error(workflow.formatDiagnostics(report));
module.exports = workflow.compile(definition);
```

## Job graph

```text
input:source → [normalize] → [validate] → output:dataset
```

## Transformation and mapping

| Layer | Result |
|---|---|
| IR | two `task` jobs with one data dependency |
| Compiled jobs | `data.normalize-customers/v1` and `data.validate-dataset/v1`, both bound to `cpu.transform` |
| Initial durable nodes | `normalize` only is ready; `validate` is pending |
| After normalize success | output `dataset` ref commits; `validate` becomes ready with that ref bound |
| Attempts | normally two; more only if configured retry class occurs |
| Final run output | one validated dataset reference, not dataset bytes |

## Execution walkthrough

1. Submission stores one input `ArtifactRef`; it does not copy the customer export.
2. Dispatcher leases `normalize` under `cpu.transform` and inserts attempt 1.
3. Runner resolves the source through its named artifact capability, normalizes records, writes a content-addressed dataset, and returns its reference.
4. Completion commits the reference and makes `validate` ready.
5. Validation runner checks schema, uniqueness, and cardinality and produces a validated-manifest reference.
6. Run projection becomes succeeded when the declared output is committed.

## Failure and privacy notes

A malformed source is a non-retryable `validation` failure. A transient artifact-store write may be retryable `transport`. Records never appear in node input, attempt failure, or event payloads.

# Example 2 — Bounded website snapshot

## Intent

Fetch a seed page, extract same-origin article links, fetch each article with rate limits, parse records, and publish one ordered snapshot manifest.

## JavaScript

```js
const workflow = require("workflow");
const web = require("web");
const data = require("data");

const definition = workflow.define("news-site-snapshot", p => {
  const seed = p.input("seed", {schema: "web-url-ref/v1", role: "seed-url"});

  p.resource("http", r => r
    .class("http.public.egress")
    .maxInFlight(4)
    .rate({requestsPerMinute: 60, burst: 4})
    .fairness("workflow-round-robin"));
  p.resource("cpu", r => r.class("cpu.transform").maxInFlight(4));

  const frontPage = p.task("fetch-front-page",
    web.tasks.fetch({url: seed, response: "html-ref"}),
    j => j.resource("http").timeout("30s")
      .retry({maxAttempts: 3, classes: ["transport", "rate-limit", "provider-5xx"]}));

  const links = p.task("extract-links",
    web.tasks.extractLinks({
      html: frontPage.output("html"),
      sameOrigin: true,
      selector: "article a",
      limit: 500,
    }),
    j => j.after(frontPage).resource("cpu"));

  const pages = p.map("fetch-articles", links.output("urls"), url =>
    web.tasks.fetch({url, response: "html-ref"}),
    j => j.resource("http").timeout("30s")
      .retry({maxAttempts: 3, classes: ["transport", "rate-limit", "provider-5xx"]}));

  const records = p.map("parse-articles", pages, page =>
    web.tasks.parseArticle({html: page.output("html")}),
    j => j.resource("cpu").timeout("30s"));

  const snapshot = p.reduce("build-snapshot", records, partition =>
    data.tasks.reduceManifest({partition, orderedBy: "canonicalUrl"}),
    j => j.resource("cpu").fanIn(128).orderedBy("itemKey"));

  p.output("snapshot", snapshot.output("manifest"));
});

module.exports = workflow.compile(definition);
```

## Job graph

```text
seed
  ↓
fetch-front-page
  ↓
extract-links ── URLRef set
  ↓ map
fetch-articles[N] ── HTML ArtifactRef set
  ↓ map
parse-articles[N] ── ArticleRecordRef set
  ↓ reduce tree (fan-in 128)
build-snapshot[partitions] → root manifest
```

## Transformation and mapping

Assume 347 unique links:

| Plan job | Mode | Durable nodes | Resource |
|---|---|---:|---|
| `fetch-front-page` | task | 1 | `http.public.egress` |
| `extract-links` | task | 1 | `cpu.transform` |
| `fetch-articles` | map | 347, lazily materialized | `http.public.egress` |
| `parse-articles` | map | 347 | `cpu.transform` |
| `build-snapshot` | reduce | 3 level-0 partitions + 1 root | `cpu.transform` |

The plan contains five jobs, while the run contains 700 nodes. Node keys use canonical URL/item keys, not completion order.

## Execution walkthrough

- Fetch and CPU work overlap because resources are independent.
- When one HTTP slot frees, another article fetch starts immediately; no batch-wide barrier exists.
- Parsing can start for article A while article B is still downloading because each map output is independently committed.
- Extracted URL values are compact `URLRef` records. HTML bytes live in the artifact store.
- Reducers sort by canonical URL and build bounded shard manifests. A final reducer creates one root manifest.

## Failure and ordering notes

A required failed article fetch normally fails its dependent parse node and eventually the snapshot. A domain option could permit a declared partial snapshot, but partial semantics must be explicit in task/schema and root manifest. Completion timing never determines manifest order.

# Example 3 — Paginated partner API synchronization

## Intent

Discover immutable page cursors, fetch pages with a partner-specific rate limit, normalize records, and apply idempotent destination upserts.

## JavaScript

```js
const workflow = require("workflow");
const api = require("api");
const data = require("data");

const definition = workflow.define("partner-catalog-sync", p => {
  const account = p.input("account", {schema: "partner-account-ref/v1"});
  const since = p.input("checkpoint", {schema: "sync-checkpoint-ref/v1"});

  p.resource("partner-api", r => r
    .class("http.partner-api")
    .maxInFlight(3)
    .rate({requestsPerMinute: 60, burst: 3}));
  p.resource("cpu", r => r.class("cpu.transform").maxInFlight(4));
  p.resource("destination", r => r.class("db.destination.write").maxInFlight(1));

  const pageRefs = p.task("enumerate-pages",
    api.tasks.enumeratePages({account, since, maxPages: 10000}),
    j => j.resource("partner-api").timeout("2m"));

  const pages = p.map("fetch-pages", pageRefs.output("pages"), page =>
    api.tasks.fetchPage({account, page}),
    j => j.resource("partner-api").timeout("1m")
      .budget({requests: 1})
      .retry({maxAttempts: 5, classes: ["transport", "rate-limit", "provider-5xx"]}));

  const normalized = p.map("normalize-pages", pages, page =>
    data.tasks.normalizeCatalogPage({page}),
    j => j.resource("cpu"));

  const applied = p.map("apply-pages", normalized, page =>
    api.tasks.applyCatalogPage({account, page, idempotency: "item-key"}),
    j => j.resource("destination").timeout("2m")
      .retry({maxAttempts: 3, classes: ["transport"]}));

  const checkpoint = p.reduce("commit-checkpoint", applied, partition =>
    api.tasks.reduceSyncCheckpoint({partition}),
    j => j.resource("destination").fanIn(256).orderedBy("pageCursor"));

  p.output("checkpoint", checkpoint.output("checkpoint"));
});

module.exports = workflow.compile(definition);
```

## Mapping and execution

- `enumerate-pages` executes once and outputs a compact set of cursor refs.
- `fetch-pages` creates one node per cursor. Credentials are selected by the runner from `account`; they are not in the descriptor or node input.
- Rate tokens and request budget are reserved in the lease transaction.
- A page's normalization can run as soon as that page succeeds.
- `apply-pages` is serialized by `db.destination.write=1`, even while fetch and normalization remain parallel.
- Idempotency keys derive from `(run identity, job key, page cursor)` so a retry cannot apply the page twice.
- The checkpoint reducer commits only after all required page applications succeed.

## Why page enumeration is a task

The authoring script cannot call the partner API to discover pages: that would make compilation network-dependent and leak authority into the authoring runtime. Enumeration is a durable task under the same lease/rate/budget rules as fetching.

# Example 4 — Two-source ETL join with a quality gate

## Intent

Normalize customer and order exports independently, join them, evaluate quality, and publish only an accepted dataset.

## JavaScript

```js
const workflow = require("workflow");
const data = require("data");

const definition = workflow.define("customer-order-mart", p => {
  const customers = p.inputSet("customers", {schema: "customer-shard-ref-set/v1"});
  const orders = p.inputSet("orders", {schema: "order-shard-ref-set/v1"});

  p.resource("cpu", r => r.class("cpu.transform").maxInFlight(6));
  p.resource("publish", r => r.class("storage.object").maxInFlight(1));

  const cleanCustomers = p.map("normalize-customers", customers, shard =>
    data.tasks.normalizeCustomerShard({shard}),
    j => j.resource("cpu"));

  const cleanOrders = p.map("normalize-orders", orders, shard =>
    data.tasks.normalizeOrderShard({shard}),
    j => j.resource("cpu"));

  const joined = p.task("join",
    data.tasks.joinDatasets({
      left: cleanCustomers,
      right: cleanOrders,
      on: ["customerId"],
      strategy: "partitioned-hash",
    }),
    j => j.after(cleanCustomers, cleanOrders).resource("cpu").timeout("15m"));

  const quality = p.task("quality",
    data.tasks.evaluateQuality({
      dataset: joined.output("dataset"),
      rules: {
        minimumRows: 1000,
        maximumOrphanRate: 0.001,
        requiredColumns: ["customerId", "orderId", "total"],
      },
    }),
    j => j.after(joined).resource("cpu"));

  const published = p.task("publish",
    data.tasks.publishDataset({
      dataset: joined.output("dataset"),
      quality: quality.output("acceptedReport"),
    }),
    j => j.after(quality).resource("publish").retry({maxAttempts: 1}));

  p.output("mart", published.output("manifest"));
});

module.exports = workflow.compile(definition);
```

## Job and node mapping

With 20 customer shards and 80 order shards:

```text
normalize-customers: 20 map nodes ─┐
                                    ├─ join: 1 task node
normalize-orders:    80 map nodes ─┘         │
                                              ├─ quality: 1
                                              └─ publish: 1
```

The join runner receives two set-manifest references, not 100 inlined shard bodies. It may internally create partition artifacts, but if partitions need independent retry/visibility the workflow should model them as map/reduce jobs rather than hiding them inside one long task.

## Gate semantics

This example treats `acceptedReport` as a typed output that exists only when quality passes. A quality rejection is a non-retryable `validation` failure, so `publish` never becomes ready. This avoids introducing a conditional API for a simple fail-closed gate.

# Example 5 — Media transcode matrix

## Intent

Probe uploaded videos, transcode each video into three renditions, generate thumbnails, and publish one media-package manifest.

## JavaScript

```js
const workflow = require("workflow");
const media = require("media");
const files = require("files");

const definition = workflow.define("video-rendition-package", p => {
  const videos = p.inputSet("videos", {schema: "video-ref-set/v1"});

  p.resource("probe", r => r.class("cpu.transform").maxInFlight(4));
  p.resource("ffmpeg", r => r.class("media.ffmpeg").maxInFlight(2));
  p.resource("storage", r => r.class("storage.object").maxInFlight(4));

  const metadata = p.map("probe", videos, video =>
    media.tasks.probe({video}),
    j => j.resource("probe").timeout("2m"));

  const renditions = p.map("transcode", metadata, item =>
    media.tasks.transcodeMatrix({
      video: item.output("video"),
      metadata: item.output("metadata"),
      renditions: [
        {name: "360p", height: 360, videoBitrate: "800k"},
        {name: "720p", height: 720, videoBitrate: "2500k"},
        {name: "1080p", height: 1080, videoBitrate: "5000k"},
      ],
    }),
    j => j.resource("ffmpeg").timeout("45m")
      .budget({artifactBytes: 20_000_000_000}));

  const thumbnails = p.map("thumbnails", metadata, item =>
    media.tasks.thumbnailSheet({video: item.output("video"), columns: 5, rows: 4}),
    j => j.resource("ffmpeg").timeout("10m"));

  const bundle = p.task("bundle",
    files.tasks.bundleMediaPackage({renditions, thumbnails}),
    j => j.after(renditions, thumbnails).resource("storage"));

  p.output("package", bundle.output("manifest"));
});

module.exports = workflow.compile(definition);
```

## Execution behavior

- Probe nodes can run four at a time; transcode and thumbnail nodes share only two ffmpeg slots.
- The dispatcher can choose fairness between the two ffmpeg jobs so thumbnails are not starved behind all transcodes.
- `transcodeMatrix` is one bounded node per source video in this example. If each rendition requires independent retry, change it to an expansion task that outputs rendition requests followed by a rendition map job.
- Output video bytes are external artifacts. SQLite stores digest, size, media type, logical type, and locator.
- Artifact-byte budget is reserved conservatively and settled from actual output sizes.

## Alternative finer-grained graph

```text
probe video
   ↓
expand rendition requests (3 refs)
   ↓ map
transcode one rendition (independent attempts)
   ↓ reduce
rendition manifest
```

Choose granularity based on retry boundary, provider/tool invocation boundary, output validation, and observability—not merely on function decomposition.

# Example 6 — Word-count map/reduce

## Intent

Count normalized tokens across a large document collection using bounded map tasks and a deterministic reduction tree.

## JavaScript

```js
const workflow = require("workflow");
const analytics = require("analytics");

const definition = workflow.define("document-word-count", p => {
  const documents = p.inputSet("documents", {schema: "text-document-ref-set/v1"});

  p.resource("cpu", r => r.class("cpu.transform").maxInFlight(6));

  const partials = p.map("count-document", documents, document =>
    analytics.tasks.tokenCount({
      document,
      normalization: {caseFold: true, unicode: "NFKC", punctuation: "drop"},
    }),
    j => j.resource("cpu").timeout("5m"));

  const totals = p.reduce("sum-counts", partials, partition =>
    analytics.tasks.reduceTokenCounts({partition}),
    j => j.resource("cpu").fanIn(64).orderedBy("token"));

  p.output("counts", totals.output("countManifest"));
});

module.exports = workflow.compile(definition);
```

## Mapping for 10,000 documents

- one compiled map job;
- 10,000 map nodes lazily expanded from the input set;
- level 0: 157 reducers at fan-in 64;
- level 1: 3 reducers;
- root: 1 reducer;
- normal attempt count: 10,161.

## Determinism

Every document has a canonical item key. Partial manifests sort token keys. A partition's identity is a digest of sorted member refs, and merge arithmetic is associative. Completion order cannot change the root digest.

## Work-conserving execution

The expander maintains a bounded ready backlog. Each completed CPU map attempt releases capacity and immediately wakes dispatch. Reducers can start as soon as a complete partition exists rather than waiting for all 10,000 documents.

# Example 7 — Repository security scan and policy decision

## Intent

Scan a source snapshot with several independent tools, normalize findings, evaluate one policy, and sign the accepted report.

## JavaScript

```js
const workflow = require("workflow");
const security = require("security");

const definition = workflow.define("release-security-gate", p => {
  const source = p.input("source", {schema: "source-snapshot-ref/v1"});
  const policy = p.input("policy", {schema: "security-policy-ref/v1"});

  p.resource("scan", r => r.class("cpu.security-scan").maxInFlight(2));
  p.resource("control", r => r.class("control.local").maxInFlight(1));

  const requests = p.task("scan-matrix",
    security.tasks.expandScanMatrix({
      source,
      scanners: ["secrets/v2", "dependencies/v1", "static-analysis/v3", "licenses/v1"],
    }),
    j => j.resource("control"));

  const findings = p.map("scan", requests.output("requests"), request =>
    security.tasks.scan({source, request}),
    j => j.resource("scan").timeout("20m")
      .retry({maxAttempts: 2, classes: ["internal"]}));

  const report = p.reduce("merge-findings", findings, partition =>
    security.tasks.mergeFindings({partition}),
    j => j.resource("control").fanIn(32).orderedBy("findingKey"));

  const decision = p.task("evaluate-policy",
    security.tasks.evaluatePolicy({report: report.output("report"), policy}),
    j => j.after(report).resource("control"));

  const signed = p.task("sign-report",
    security.tasks.signReport({accepted: decision.output("acceptedReport")}),
    j => j.after(decision).resource("control").retry({maxAttempts: 1}));

  p.output("attestation", signed.output("attestation"));
});

module.exports = workflow.compile(definition);
```

## Execution and failure classes

- The matrix task emits compact scanner request descriptors; scanner binaries and rule databases are host capabilities.
- Scan nodes share two sandbox slots.
- A scanner crash can retry as `internal`; findings that violate policy are not a retry condition.
- Policy rejection returns `validation`/`SECURITY_POLICY_REJECTED`, preventing signing.
- The signing key never appears in plan, node, event, or task input. The signer runner addresses a named key through host configuration.

# Example 8 — Batch image classification (non-RAG ML)

## Intent

Preprocess images on CPU, run one model inference per item on a serial GPU, aggregate class distributions, and publish predictions.

## JavaScript

```js
const workflow = require("workflow");
const ml = require("ml");

const definition = workflow.define("image-classification-batch", p => {
  const images = p.inputSet("images", {schema: "image-ref-set/v1"});
  const model = p.input("model", {schema: "model-profile-ref/v1"});

  p.resource("preprocess", r => r.class("cpu.transform").maxInFlight(6));
  p.resource("inference", r => r.class("gpu.inference").maxInFlight(1));

  const tensors = p.map("preprocess", images, image =>
    ml.tasks.preprocessImage({image, width: 224, height: 224, normalize: "imagenet"}),
    j => j.resource("preprocess").timeout("2m"));

  const predictions = p.map("infer", tensors, tensor =>
    ml.tasks.classifyImage({tensor, model, topK: 5}),
    j => j.resource("inference").timeout("1m")
      .retry({maxAttempts: 2, classes: ["transport", "internal"]}));

  const summary = p.reduce("summarize", predictions, partition =>
    ml.tasks.aggregatePredictions({partition}),
    j => j.resource("preprocess").fanIn(256).orderedBy("imageKey"));

  p.output("predictions", predictions);
  p.output("summary", summary.output("summary"));
});

module.exports = workflow.compile(definition);
```

## Scheduling timeline

```text
CPU slots: preprocess I1 I2 I3 I4 I5 I6 ...
GPU slot:              infer I1 ─ infer I2 ─ infer I3 ...
CPU slots:                         more preprocessing + ready reducers
```

The GPU remains independently saturated while CPU preprocessing continues. A single global worker batch would risk leaving the GPU idle behind unrelated CPU tasks; resource-class dispatch does not.

## Persistence

Tensor bytes should be short-lived external artifacts with retention policy, not SQLite JSON. Predictions are validated compact values or artifact refs. Model identity includes immutable model digest/profile; endpoint credentials remain host-only.

# Example 9 — Multi-channel notification fan-out

## Intent

Render one notification, deliver it to a set of recipients/channels under separate rate limits, and aggregate durable receipts.

## JavaScript

```js
const workflow = require("workflow");
const notify = require("notify");

const definition = workflow.define("incident-notification", p => {
  const incident = p.input("incident", {schema: "incident-ref/v1"});
  const destinations = p.inputSet("destinations", {schema: "notification-destination-ref-set/v1"});

  p.resource("render", r => r.class("cpu.transform").maxInFlight(2));
  p.resource("email", r => r.class("notification.email").maxInFlight(2));
  p.resource("chat", r => r.class("notification.chat").maxInFlight(4));

  const message = p.task("render",
    notify.tasks.renderIncident({incident, template: "incident-standard/v2"}),
    j => j.resource("render"));

  const deliveries = p.map("deliver", destinations, destination =>
    notify.tasks.deliver({destination, message: message.output("message")}),
    j => j.after(message)
      .resourceBy("destination.channel", {
        email: "email",
        chat: "chat",
      })
      .timeout("30s")
      .retry({maxAttempts: 5, classes: ["transport", "rate-limit", "provider-5xx"]}));

  const receipt = p.reduce("receipt", deliveries, partition =>
    notify.tasks.reduceReceipts({partition, requireAll: true}),
    j => j.resource("render").fanIn(128).orderedBy("destinationKey"));

  p.output("receipt", receipt.output("receipt"));
});

module.exports = workflow.compile(definition);
```

## Design pressure: dynamic resource binding

`resourceBy` is not in the minimal DSL sketch. This example exposes a real need: one map job may route items to different resource classes based on a compact, compiler-approved discriminator.

Safe implementation options:

1. split destination sets by channel into separate map jobs during deterministic expansion;
2. allow a finite compiler-validated resource switch expression;
3. require the domain module to expose `emailDestinations` and `chatDestinations` ports.

Do not permit arbitrary JavaScript resource selection at dispatch time. The compiler must enumerate every possible effective class. Option 1 or 3 is simplest initially.

## Delivery semantics

Each destination item key becomes the idempotency key. A provider timeout may be ambiguous; attempt history records it, retry follows channel policy, and receipts state whether delivery was confirmed. Raw message bodies need not enter events.

# Example 10 — Sharded database backup and restore verification

## Intent

Snapshot a consistent database identity, dump table/range shards, build a root backup manifest, restore into an isolated verifier, and publish only a verified backup.

## JavaScript

```js
const workflow = require("workflow");
const database = require("database");

const definition = workflow.define("verified-database-backup", p => {
  const databaseRef = p.input("database", {schema: "database-handle-ref/v1"});
  const destination = p.input("destination", {schema: "artifact-store-ref/v1"});

  p.resource("read", r => r.class("db.source.read").maxInFlight(2));
  p.resource("storage", r => r.class("storage.object").maxInFlight(4));
  p.resource("verify", r => r.class("db.destination.write").maxInFlight(1));

  const snapshot = p.task("open-snapshot",
    database.tasks.openConsistentSnapshot({database: databaseRef}),
    j => j.resource("read").timeout("2m").retry({maxAttempts: 1}));

  const shards = p.task("plan-shards",
    database.tasks.planSnapshotShards({snapshot: snapshot.output("snapshot"), targetBytes: 256_000_000}),
    j => j.after(snapshot).resource("read"));

  const dumps = p.map("dump-shards", shards.output("shards"), shard =>
    database.tasks.dumpSnapshotShard({snapshot: snapshot.output("snapshot"), shard, destination}),
    j => j.resource("read").timeout("30m")
      .budget({artifactBytes: 1_000_000_000}));

  const manifest = p.reduce("backup-manifest", dumps, partition =>
    database.tasks.reduceBackupManifest({partition}),
    j => j.resource("storage").fanIn(64).orderedBy("shardKey"));

  const verified = p.task("restore-verify",
    database.tasks.restoreVerify({backup: manifest.output("manifest")}),
    j => j.after(manifest).resource("verify").timeout("2h").retry({maxAttempts: 1}));

  p.output("backup", verified.output("verifiedBackup"));
});

module.exports = workflow.compile(definition);
```

## Correctness constraints

- The snapshot handle is a compact opaque capability reference with expiry/ownership semantics; it is not a DSN.
- If the database cannot keep a snapshot valid across durable retries, shard dumping must use an immutable database snapshot service rather than a live transaction.
- Backup publication occurs only after root cardinality/digest validation and restore verification.
- A failed verification does not delete forensic artifacts automatically; retention follows explicit policy.

# Example 11 — Inventory reconciliation and idempotent repair

## Intent

Compare warehouse and storefront inventory snapshots, calculate discrepancies, apply bounded repairs, and produce an audit report.

## JavaScript

```js
const workflow = require("workflow");
const data = require("data");
const api = require("api");

const definition = workflow.define("inventory-reconciliation", p => {
  const warehouse = p.input("warehouse", {schema: "inventory-snapshot-ref/v1"});
  const storefront = p.input("storefront", {schema: "inventory-snapshot-ref/v1"});
  const repairPolicy = p.input("repairPolicy", {schema: "inventory-repair-policy-ref/v1"});

  p.resource("cpu", r => r.class("cpu.transform").maxInFlight(4));
  p.resource("store-api", r => r.class("http.partner-api").maxInFlight(2));

  const diff = p.task("diff",
    data.tasks.diffInventory({warehouse, storefront, policy: repairPolicy}),
    j => j.resource("cpu"));

  const repairs = p.map("repair", diff.output("repairs"), repair =>
    api.tasks.applyInventoryRepair({repair, idempotency: "repair-key"}),
    j => j.resource("store-api").timeout("30s")
      .budget({requests: 1})
      .retry({maxAttempts: 4, classes: ["transport", "rate-limit", "provider-5xx"]}));

  const audit = p.reduce("audit", repairs, partition =>
    data.tasks.reduceRepairAudit({partition, expected: diff.output("summary")}),
    j => j.resource("cpu").fanIn(128).orderedBy("sku"));

  p.output("audit", audit.output("report"));
});

module.exports = workflow.compile(definition);
```

## Job execution

A single `diff` task writes a compact repair-request set. Each repair is a separate node and can retry independently. The destination API call receives an idempotency key derived from the immutable repair key. The reducer verifies attempted/succeeded/failed cardinality against the diff summary before publishing the audit.

## Scheduling the workflow

A recurring schedule is not part of the pure definition. Trusted host configuration invokes `Ensure(plan, identity, bindings)` for each frozen pair of inventory snapshots. Schedule time may select inputs, but run identity derives from snapshot/policy/plan digests—not only the wall clock.

# Example 12 — Build, test, package, and sign

## Intent

Build a source snapshot for a platform matrix, run tests, package successful builds, and sign the release manifest.

## JavaScript

```js
const workflow = require("workflow");
const build = require("build");

const definition = workflow.define("cross-platform-release", p => {
  const source = p.input("source", {schema: "source-snapshot-ref/v1"});
  const targets = p.inputSet("targets", {schema: "build-target-ref-set/v1"});

  p.resource("build", r => r.class("cpu.transform").maxInFlight(4));
  p.resource("sign", r => r.class("control.local").maxInFlight(1));

  const binaries = p.map("build", targets, target =>
    build.tasks.compile({source, target, hermetic: true}),
    j => j.resource("build").timeout("30m"));

  const tests = p.map("test", binaries, binary =>
    build.tasks.testBinary({binary, suite: "release"}),
    j => j.resource("build").timeout("20m"));

  const packages = p.map("package", tests, tested =>
    build.tasks.packageBinary({
      binary: tested.output("binary"),
      testReport: tested.output("report"),
    }),
    j => j.resource("build"));

  const manifest = p.reduce("manifest", packages, partition =>
    build.tasks.reduceReleaseManifest({partition}),
    j => j.resource("build").fanIn(32).orderedBy("targetTriple"));

  const signed = p.task("sign",
    build.tasks.signRelease({manifest: manifest.output("manifest"), key: "release-primary"}),
    j => j.after(manifest).resource("sign").retry({maxAttempts: 1}));

  p.output("release", signed.output("signedManifest"));
});

module.exports = workflow.compile(definition);
```

## Execution boundary

The source is an immutable snapshot ref. Each build target creates one node. Sandboxed runners resolve toolchains from the host capability profile. Build commands, environment allowlists, and toolchain digests are task/catalog facts; scripts cannot inject arbitrary shell strings into a generic runner.

Signing uses the public key alias `release-primary`; secret key bytes stay in the signer service. A failing target prevents root manifest/signing unless a separate partial-release schema explicitly allows omissions.

# Example 13 — Human approval without holding a worker lease

## Intent

Prepare a deployment, wait durably for operator approval, then deploy and verify. Waiting must not occupy a worker goroutine, lease, or resource slot.

## JavaScript

```js
const workflow = require("workflow");
const build = require("build");
const ops = require("ops");

const definition = workflow.define("approved-deployment", p => {
  const release = p.input("release", {schema: "signed-release-ref/v1"});
  const environment = p.input("environment", {schema: "deployment-environment-ref/v1"});

  p.resource("control", r => r.class("control.local").maxInFlight(2));
  p.resource("approval", r => r.class("operator.approval").maxInFlight(100));

  const prepared = p.task("prepare",
    ops.tasks.prepareDeployment({release, environment}),
    j => j.resource("control"));

  const approval = p.gate("approval",
    ops.tasks.awaitApproval({
      subject: prepared.output("deploymentPlan"),
      policy: "production-two-person/v1",
      expiresAfter: "24h",
    }),
    g => g.after(prepared).resource("approval"));

  const deployed = p.task("deploy",
    ops.tasks.deploy({approvedPlan: approval.output("approvedPlan")}),
    j => j.after(approval).resource("control").timeout("30m").retry({maxAttempts: 1}));

  const verified = p.task("verify",
    ops.tasks.verifyDeployment({deployment: deployed.output("deployment")}),
    j => j.after(deployed).resource("control"));

  p.output("deployment", verified.output("verifiedDeployment"));
});

module.exports = workflow.compile(definition);
```

## Gate mapping

A gate is not a normal long-running task attempt:

1. dispatcher briefly leases a gate initializer if external registration is needed;
2. initializer records a correlation token hash and transitions the node to `waiting`;
3. it releases the lease and resource immediately;
4. authenticated operator events are appended separately;
5. when policy is satisfied, a transaction commits the approval artifact and marks the gate succeeded;
6. `deploy` becomes ready;
7. timeout/cancellation closes the gate without a stale callback being accepted.

## Design pressure

The minimal `PlanBuilder` sketch does not yet declare `p.gate`. This example justifies a first-class gate mode rather than implementing approval as a worker that sleeps for 24 hours. Gate event authentication and two-person policy belong to the `ops` host capability, not JavaScript.

# Example 14 — Probe matrix and SLO report

## Intent

Probe a finite matrix of endpoints and regions, evaluate SLO rules, and emit one status report and optional notification plan.

## JavaScript

```js
const workflow = require("workflow");
const ops = require("ops");
const notify = require("notify");

const definition = workflow.define("service-probe-matrix", p => {
  const probes = p.inputSet("probes", {schema: "service-probe-ref-set/v1"});
  const slo = p.input("slo", {schema: "slo-policy-ref/v1"});

  p.resource("network", r => r
    .class("http.public.egress")
    .maxInFlight(8)
    .rate({requestsPerMinute: 240, burst: 8}));
  p.resource("cpu", r => r.class("cpu.transform").maxInFlight(2));

  const observations = p.map("probe", probes, probe =>
    ops.tasks.probe({probe, samples: 3}),
    j => j.resource("network").timeout("20s")
      .retry({maxAttempts: 2, classes: ["transport"]}));

  const report = p.reduce("evaluate", observations, partition =>
    ops.tasks.evaluateSLO({partition, slo}),
    j => j.resource("cpu").fanIn(128).orderedBy("probeKey"));

  const notification = p.task("notification-plan",
    notify.tasks.planFromSLOReport({report: report.output("report")}),
    j => j.after(report).resource("cpu"));

  p.output("report", report.output("report"));
  p.output("notificationPlan", notification.output("plan"));
});

module.exports = workflow.compile(definition);
```

## Attempts versus samples

The task config requests three samples inside one bounded probe attempt. Those samples are not workflow retries. If the runner crashes, a second attempt may repeat all three. The output schema must distinguish sample observations from attempt metadata. If each sample needs independent lease/retry evidence, expand samples into map items instead.

# Example 15 — Mixed-resource document conversion

## Intent

Detect file types, route office documents, images, and plain text through appropriate converters, then bundle outputs with one manifest.

## JavaScript

```js
const workflow = require("workflow");
const files = require("files");

const definition = workflow.define("document-conversion-bundle", p => {
  const documents = p.inputSet("documents", {schema: "document-ref-set/v1"});

  p.resource("inspect", r => r.class("cpu.transform").maxInFlight(6));
  p.resource("convert", r => r.class("media.ffmpeg").maxInFlight(2));
  p.resource("storage", r => r.class("storage.object").maxInFlight(4));

  const inspected = p.map("inspect", documents, document =>
    files.tasks.inspect({document}),
    j => j.resource("inspect"));

  const requests = p.task("route",
    files.tasks.routeConversions({
      inspected,
      formats: {
        office: "pdf",
        image: "pdf",
        text: "pdf",
      },
    }),
    j => j.after(inspected).resource("inspect"));

  const converted = p.map("convert", requests.output("requests"), request =>
    files.tasks.convert({request}),
    j => j.resource("convert").timeout("10m"));

  const bundle = p.reduce("bundle", converted, partition =>
    files.tasks.reduceBundleManifest({partition}),
    j => j.resource("storage").fanIn(128).orderedBy("documentKey"));

  p.output("bundle", bundle.output("manifest"));
});

module.exports = workflow.compile(definition);
```

## Why routing is normalized data

The route task outputs a finite set of compact conversion request descriptors. The converter runner registry maps each request kind to an approved implementation. The script cannot put an arbitrary executable path or command line in a request. Unsupported file types fail routing before converter nodes are created.

# Deep transformation atlas: website snapshot example

The following expands Example 2 across every layer.

## Stage A — JavaScript authoring objects

During `workflow.define`, the Goja module owns:

```text
PlanBuilder(name=news-site-snapshot)
├── InputHandle(seed, schema=web-url-ref/v1)
├── ResourceBuilder(http, requestedClass=http.public.egress, max=4, rate=60/min)
├── ResourceBuilder(cpu, requestedClass=cpu.transform, max=4)
├── JobHandle(fetch-front-page)
├── JobHandle(extract-links)
├── SetHandle(fetch-articles[*])
├── SetHandle(parse-articles[*])
├── SetHandle(build-snapshot[partitions])
└── OutputHandle(snapshot)
```

Handles contain private runtime ownership identity. Calling a method with a handle from another plan fails before IR generation.

## Stage B — Normalized IR

The native module emits data equivalent to:

```json
{
  "schemaVersion": "scraper-workflow-ir/v3",
  "name": "news-site-snapshot",
  "inputs": [
    {"key": "seed", "schema": "web-url-ref/v1", "role": "seed-url"}
  ],
  "resources": [
    {
      "key": "http",
      "requestedClass": "http.public.egress",
      "maxInFlight": 4,
      "rate": {"requestsPerMinute": 60, "burst": 4}
    },
    {"key": "cpu", "requestedClass": "cpu.transform", "maxInFlight": 4}
  ],
  "jobs": [
    {
      "key": "fetch-front-page",
      "mode": "task",
      "task": {"kind": "web.fetch", "version": "v1"},
      "bindings": {"url": {"$ref": "input", "key": "seed"}},
      "resource": "http"
    },
    {
      "key": "extract-links",
      "mode": "task",
      "task": {"kind": "web.extract-links", "version": "v1"},
      "bindings": {"html": {"$ref": "job-output", "job": "fetch-front-page", "port": "html"}},
      "resource": "cpu"
    },
    {
      "key": "fetch-articles",
      "mode": "map",
      "source": {"$ref": "job-output", "job": "extract-links", "port": "urls"},
      "task": {"kind": "web.fetch", "version": "v1"},
      "bindings": {"url": {"$ref": "map-item"}},
      "resource": "http"
    },
    {
      "key": "parse-articles",
      "mode": "map",
      "source": {"$ref": "job-output-set", "job": "fetch-articles"},
      "task": {"kind": "web.parse-article", "version": "v1"},
      "bindings": {"html": {"$ref": "map-item-output", "port": "html"}},
      "resource": "cpu"
    },
    {
      "key": "build-snapshot",
      "mode": "reduce",
      "source": {"$ref": "job-output-set", "job": "parse-articles"},
      "task": {"kind": "data.reduce-manifest", "version": "v1"},
      "fanIn": 128,
      "orderedBy": "itemKey",
      "resource": "cpu"
    }
  ],
  "outputs": [
    {"name": "snapshot", "from": {"job": "build-snapshot", "port": "manifest"}}
  ]
}
```

Task descriptors also carry input/output schema and config digest; they are shortened above.

## Stage C — Validation

Go validates:

- all keys are unique and normalized;
- every reference points to an existing input/job/port;
- the graph is acyclic;
- `extract-links.html` expects the schema produced by `web.fetch.html`;
- the map source is a set with stable item keys;
- the reduction is associative/deterministic according to task catalog metadata;
- fan-in and requested concurrency are within absolute structural limits;
- inputs/config contain compact references and no forbidden fields;
- task kinds and versions exist in the selected catalog.

Validation errors happen before any run row is persisted and do not echo rejected source values.

## Stage D — Compilation

Suppose the host profile permits only three HTTP calls for this tenant. Compilation produces:

```json
{
  "schemaVersion": "scraper-workflow-plan/v3",
  "definitionDigest": "sha256:definition...",
  "compilerVersion": "workflowcompile/v3.0.0",
  "capabilityDigest": "sha256:workstation-profile...",
  "requestedPolicy": {
    "http": {"class": "http.public.egress", "maxInFlight": 4}
  },
  "effectivePolicy": {
    "http": {"class": "http.public.egress", "maxInFlight": 3}
  },
  "jobs": [
    {
      "key": "fetch-front-page",
      "taskImplementation": "web.fetch/v1@sha256:runner...",
      "resourceClass": "http.public.egress",
      "timeoutMs": 30000,
      "retry": {
        "maxAttempts": 3,
        "classes": ["transport", "rate-limit", "provider-5xx"]
      }
    }
  ],
  "digest": "sha256:plan..."
}
```

The actual plan contains all jobs. Compilation never embeds HTTP credentials or source HTML.

## Stage E — Submission and run identity

Trusted host code calls conceptually:

```go
run, err := submitService.Ensure(ctx, plan, identity, map[string]ValueRef{
    "seed": seedURLRef,
})
```

Identity includes plan digest and immutable input digest. If a run exists with the same identity, the service attaches. If the same requested run ID has a different identity, it returns a conflict.

The submit transaction creates:

- one run row;
- static `fetch-front-page` node;
- pending/static templates or expansion cursors for downstream jobs;
- `run.created` event.

It does not create 347 article nodes because links do not exist yet.

## Stage F — First lease and attempt

The dispatcher sees `fetch-front-page` ready and HTTP capacity available. One transaction:

1. reserves one HTTP in-flight grant and one rate token;
2. reserves one request from the budget if configured;
3. increments node attempt to 1;
4. inserts attempt row;
5. inserts lease with token hash and cancellation epoch;
6. marks node running;
7. appends `node.leased`.

The worker receives a lease grant containing compact refs and task identity.

## Stage G — Runner execution

The `web.fetch/v1` runner:

1. resolves `URLRef` through its allowed codec;
2. applies host egress policy and configured HTTP client;
3. makes one bounded request under context deadline;
4. validates status, maximum bytes, content type, final URL, and redirect policy;
5. writes response body to content-addressed storage;
6. returns an HTML `ArtifactRef` and redacted usage/status metadata.

It does not let JavaScript select arbitrary headers or persist response bodies in events.

## Stage H — Success and downstream readiness

One completion transaction checks lease/cancellation, commits the HTML ref, finishes the attempt, releases HTTP/budget reservation, removes the lease, marks success, and wakes `extract-links`.

`extract-links` resolves HTML under a CPU task lease and returns a deduplicated, sorted URL-ref set manifest.

## Stage I — Lazy map expansion

The map expander reads a bounded page of URL refs. For each URL it derives:

```text
node_key = "fetch-articles/" + canonicalItemKey(urlRef)
```

It atomically inserts missing nodes and advances the cursor/page digest. Repeating the page after a crash is idempotent.

## Stage J — Concurrent streaming through the graph

As each article fetch commits:

- its parse node becomes eligible;
- HTTP capacity is immediately reused by another fetch;
- CPU capacity can parse finished pages concurrently;
- no scheduler waits for all article fetches to finish.

## Stage K — Reduction and publication

After 128 parse outputs exist, a level-0 reducer can run. The final reducer verifies:

- expected versus actual item keys;
- no duplicates;
- record schemas;
- canonical ordering;
- child manifest digests and cardinality.

The declared run output points to the final root manifest. The projection reports succeeded only after that output commits.

# Execution semantics across all examples

## Static task jobs

A static task normally maps to one node per run. Inputs may bind workflow inputs and upstream outputs. It becomes ready only when required dependencies succeed.

```text
job "validate" → node (run-123, validate) → attempt 1 → success
```

## Map jobs

A map job is a template plus source-set reference. Expansion creates deterministic child nodes in pages. Each child binds `$item` to one compact ref.

```text
job "convert"
  ├─ node convert/doc-a
  ├─ node convert/doc-b
  └─ node convert/doc-c
```

## Reduction jobs

A reduction job is compiled into bounded partitions. Reducer outputs are themselves refs and may feed the next reduction level. Root ordering is canonical.

## Gates

A gate records durable waiting state and resumes from authenticated external signals. It must not hold a worker lease while waiting.

## Retries

Retries create new attempts for the same node. They do not create new logical node keys or duplicate downstream output identities.

```text
node fetch/page-17
  attempt 1 → transport timeout → retry at T+2s
  attempt 2 → HTTP 200 + valid artifact → success
```

Typed failure class and compiled policy decide retry. Error-string matching is prohibited.

## Timeouts and cancellation

A task timeout cancels runner context. Cancellation increments run epoch. Any completion carrying an old lease/epoch is rejected, so a stale worker cannot publish after cancellation.

## Resource admission

A worker may have free goroutine capacity yet be unable to lease a GPU or partner-API task. The authoritative store lease checks effective resource grants, rate tokens, capability tags, fairness, and budget atomically.

## Budgets

A node reserves estimated usage before execution and settles actual usage at completion. Hard budget exhaustion blocks or fails according to compiled policy; scripts cannot raise host ceilings.

## Artifact and result handling

- bytes go to approved content-addressed stores;
- node rows contain compact refs;
- outputs validate schema/digest/cardinality before commit;
- events contain redacted summaries;
- metrics use bounded labels;
- researchctl, if integrated, records plan/run/output identities rather than copying task data.

# How to decide task granularity

Make an operation its own durable node when it has one or more of these properties:

- independent retry value;
- separate external request or side effect;
- distinct resource class;
- useful progress/latency/cost evidence;
- independently validatable output;
- high failure cost or long duration;
- fan-out/fan-in boundary;
- cancellation should stop it separately.

Keep work inside one task when splitting would add only orchestration overhead and there is no meaningful independent retry/output boundary.

Bad split:

```text
open file → read first byte → read rest → close file
```

Useful split:

```text
enumerate immutable files → convert each independently → reduce manifest
```

# Patterns intentionally not expressed as arbitrary JavaScript

## Arbitrary loops that create operations

Do not write:

```js
for (const row of hugeDataset) {
  ctx.emit({id: row.id, input: {...wholePlan, row}});
}
```

Use a compact set reference plus `p.map`. Go controls lazy expansion, key derivation, row size, and durability.

## Runtime resource selection functions

Do not persist JavaScript functions that inspect live state. Use finite compiler-approved routing or separate jobs.

## Secret lookup

Do not call `env()`, read files, or place tokens in config. Use named host capabilities selected by public references.

## Long sleeps and polling

Do not hold a lease while waiting for approval, retry time, or schedule time. Persist gate/retry/schedule state and wake later.

## Unbounded recursive crawling

Do not allow a script to recursively emit unknown operations. Use a domain-owned frontier task with explicit depth/item/budget limits and deterministic frontier pages, or compile a bounded number of crawl rounds.

## Shell strings

Do not expose `exec(commandString)` as a generic task. Use registered task kinds with structured, validated config and sandbox/toolchain policy.

# DSL pressure-test findings from the cookbook

The examples validate the core `task`, `map`, `reduce`, resource, retry, timeout, budget, input, output, and descriptor model. They also reveal APIs that must be decided before v3 is frozen.

## 1. First-class gate API

Human approval and external callbacks require durable waiting without a lease. Add a `gate` job mode and builder after defining authentication, timeout, cancellation, and event contracts.

## 2. Finite resource routing

Notification/file-routing examples need item-dependent resources. Prefer explicit split ports/jobs first. If `resourceBy` is added, allow only finite compile-time mappings over registered discriminators.

## 3. Required versus optional dependencies

Most examples are fail-closed. Partial snapshots or best-effort notification receipts need an explicit schema and dependency policy. Do not infer optionality from caught JavaScript exceptions.

## 4. Multiple map outputs and typed ports

Task descriptors must declare named output ports so downstream map callbacks can bind `item.output("html")` or `item.output("metadata")` without exposing engine result maps.

## 5. Set manifests as first-class refs

Joins, routes, and reductions use set references heavily. The base contract needs stable item schema, canonical item key, count, pagination, and manifest digest.

## 6. Schedules remain host-side

A schedule chooses when to bind immutable inputs and submit/ensure a run. It should not alter definition semantics or use wall time as the sole run identity.

## 7. Expansion backpressure

Large maps need a target ready backlog so expansion does not create millions of ready rows. The dispatcher/expander should coordinate without changing deterministic child identity.

## 8. Domain task catalogs need more than a runner

Catalog entries should provide:

- config codec/schema;
- input and output port schemas;
- required capabilities;
- default failure translation;
- cost/resource estimation;
- determinism/idempotency metadata;
- help/examples;
- implementation digest/version.

# Implementation test matrix derived from the examples

| Capability | Example | Required test |
|---|---|---|
| Linear dependency | 1 | validate cannot lease before normalize output commits |
| Streaming map pipeline | 2 | parse A starts before fetch Z finishes |
| Rate and budget | 3 | partner API never exceeds tokens or request budget |
| Join/fail-closed quality | 4 | rejection prevents publish |
| Shared scarce resource | 5 | transcode/thumbnail fairness under two ffmpeg slots |
| Multi-level reduction | 6 | root count/digest stable across completion orders |
| Typed policy failure | 7 | rejected scan does not retry or sign |
| Independent CPU/GPU | 8 | GPU refills while CPU work remains active |
| Finite resource routing | 9 | every possible class is compiler-known |
| Snapshot/restore | 10 | invalid restore prevents authoritative backup output |
| Idempotent side effects | 11 | retry does not apply repair twice |
| Secret-backed signing | 12 | key material absent from DB/WAL/events/log capture |
| Gate waiting | 13 | no lease/resource held during 24-hour wait |
| Samples vs attempts | 14 | retry metadata cannot be confused with probe samples |
| Structured converter routing | 15 | arbitrary executable/command rejected |

Every example should eventually become:

1. a JavaScript fixture;
2. a normalized IR golden file;
3. a compiled-plan golden file for a fixed capability profile;
4. a store/dispatcher execution test with fake runners;
5. one or more negative/privacy tests.

# Suggested fixture layout

```text
pkg/gojamodules/workflow/testdata/v3/examples/
  01-linear-transform.js
  02-website-snapshot.js
  03-partner-api-sync.js
  04-etl-quality-gate.js
  05-media-transcode.js
  06-word-count-map-reduce.js
  07-security-policy.js
  08-image-classification.js
  09-notification-fanout.js
  10-verified-backup.js
  11-inventory-reconciliation.js
  12-cross-platform-release.js
  13-approved-deployment.js
  14-probe-matrix.js
  15-document-conversion.js
  golden/
    01-linear-transform.ir.json
    01-linear-transform.workstation-v3.plan.json
    ...
```

Fixture tests should run each script through an actual Goja runtime with `require("workflow")` and the relevant descriptor-only task modules. Hand-authored JSON that bypasses the module is insufficient for API parity testing.

# Author review checklist

Before accepting a workflow script:

- [ ] Every input is a compact typed value/ref.
- [ ] Every external call is represented by a registered bounded task.
- [ ] Every large fan-out uses a set ref and map expansion.
- [ ] Every large fan-in has a bounded reduction plan.
- [ ] Resource requests are symbolic and compiler-bindable.
- [ ] Retry classes are typed and bounded.
- [ ] Side effects have idempotency semantics.
- [ ] Output ordering/cardinality are explicit.
- [ ] Secrets and source bytes cannot enter node/event/report values.
- [ ] Waiting does not hold a lease.
- [ ] Publication happens only after validation.
- [ ] The script exports normalized/compiled intent, not a live callback.

# Operator execution checklist

When a plan runs:

- [ ] Definition, plan, capability, and input digests match the approved profile.
- [ ] Attach-versus-create identity is visible.
- [ ] Active counts by resource match configured capacity.
- [ ] Ready work does not remain idle while compatible capacity is free.
- [ ] Attempt history explains every retry/lease loss.
- [ ] Budget reserved/used/remaining is visible.
- [ ] Expansion and reduction progress is visible.
- [ ] Cancellation epoch prevents stale completion.
- [ ] Root outputs reopen and validate.
- [ ] SQLite/WAL/event/report privacy scans pass.

# Related documents

- [Durable dataflow workflow v3 and modern scripting architecture](../design-doc/01-durable-dataflow-workflow-v3-and-modern-scripting-architecture.md)
- [Investigation diary](01-investigation-diary.md)
- [Source catalogue and evidence map](02-source-catalogue-and-evidence-map.md)
- [`workflow-dsl-grammar-probe.json`](../scripts/output/workflow-dsl-grammar-probe.json)
