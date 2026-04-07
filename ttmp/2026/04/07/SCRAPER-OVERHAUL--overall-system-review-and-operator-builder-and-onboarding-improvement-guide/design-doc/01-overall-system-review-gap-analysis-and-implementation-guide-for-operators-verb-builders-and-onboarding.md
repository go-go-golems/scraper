---
Title: Overall system review, gap analysis, and implementation guide for operators, verb builders, and onboarding
Ticket: SCRAPER-OVERHAUL
Status: active
Topics:
    - scraper
    - architecture
    - onboarding
    - frontend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/api/server/server.go
      Note: Current HTTP API surface and the boundary for frontend-visible operator information
    - Path: pkg/engine/model/types.go
      Note: Canonical workflow, op, dependency, retry, queue-policy, and rate-limit types
    - Path: pkg/engine/scheduler/scheduler.go
      Note: Scheduler loop that resolves queue policy and emits rate-limit events
    - Path: pkg/engine/store/sqlite/store.go
      Note: |-
        Durable enforcement of dependency promotion and token-bucket rate limiting
        Durable dependency promotion and queue rate-limit enforcement
    - Path: pkg/js/runtime/executor.go
      Note: Worker-side JS runtime that emits child ops, dependencies, artifacts, and logs
    - Path: pkg/services/catalog/service.go
      Note: |-
        Catalog/service layer that already exposes site, verb, and queue-policy metadata to the API
        Existing catalog API seam for site
    - Path: pkg/sites/registry/registry.go
      Note: |-
        Site definition seam for queue policies, help content, and per-site capabilities
        Site definition seam for queue policies
    - Path: pkg/sites/submitverbs/runtime.go
      Note: |-
        Submit-verb JS runtime that converts emitted JS op specs into durable ops
        Submit verb emission path for op specs and dependsOn edges
    - Path: web/src/components/workflows/WorkflowTable.tsx
      Note: Workflow list UI that currently shows site names but does not link deeper into site context
    - Path: web/src/pages/QueueMonitorPage.tsx
      Note: Operator UI gap due to placeholder throughput data
    - Path: web/src/pages/SiteDetailPage.tsx
      Note: Existing frontend surface that already shows queue policy summaries and verb/script information
ExternalSources: []
Summary: Evidence-based overall review of scraper's current architecture, operator UX, verb-builder ergonomics, and onboarding path, with a detailed implementation guide addressing rate limits, dependencies, proxy visibility, site linking, and product help surfaces.
LastUpdated: 2026-04-07T12:02:00-04:00
WhatFor: Give a new engineer and project owner a detailed map of what the system currently does, what is missing, and how to improve it for long-running operations, verb debugging, and onboarding.
WhenToUse: Use before redesigning the operator UI, adding new site or queue metadata surfaces, revisiting rate-limit and dependency ergonomics, or expanding onboarding/help inside the product.
---


# Overall system review, gap analysis, and implementation guide for operators, verb builders, and onboarding

## Executive Summary

Scraper already has a coherent core architecture. The stable primitives are good: sites are registered declaratively, submit verbs seed workflows, workers execute durable ops, dependencies are persisted, queue policy is enforced in the engine store, and the API/frontend can already expose site metadata, workflow state, queue state, and live runtime events. The foundation is stronger than the current UI makes it look.

The central issue is not that the backend is missing basic concepts. The issue is that the operator-facing product surface does not yet expose those concepts clearly enough. The current system is legible to someone reading Go and JS source, but not yet legible to:

- an operator running many workflows over multiple days,
- a verb author trying to debug a workflow graph,
- a new teammate trying to understand what the system is and how to use it safely.

The specific questions raised in this review have straightforward answers:

- Do rate limits work? Yes. The token-bucket limiter is real and durable, enforced transactionally in SQLite. But the built-in site registry currently does not define non-default queue policies, so most shipped paths behave as if no rate limiting is configured.
- Can individual verbs or ops have their own rate limits? Not as a first-class verb-level feature. Policy is attached to `site + queue`, and ops choose their queue. A verb can effectively get its own pacing only by emitting ops into a dedicated queue that the site defines policy for.
- How are dependencies defined and created? Dependencies are explicit `dependsOn` edges on emitted op specs. Submit verbs and worker-side scripts both create them via `ctx.emit({ dependsOn: [...] })`, and the store persists them in `op_dependencies`.
- Is proxy support present? Yes, on the worker path. There is an explicit `--http-proxy` worker flag and HTTP runner support, plus an earlier proxy ticket documenting the rollout. But that setting is not surfaced in the API or frontend, so operators cannot see it in the web product.
- Can the workflows page show site information and link to site context? The site API and site pages already exist, but the workflows list currently shows the site only as a passive chip rather than as a navigation entry into site detail.
- Do help pages and tooltips exist? Yes, but mostly in the CLI and submit form fields. The repo has embedded help pages and field-level verb help, but the frontend does not yet use the embedded docs as a first-class help system.

The right next move is an overall product/architecture polish pass, not a rewrite. The system should keep its current runtime model and add a clearer operator information architecture:

1. site as the capability/config contract,
2. workflow as the execution graph,
3. queue as the pacing/control-plane unit,
4. runtime events as the operational timeline,
5. embedded docs as the explanation layer.

## Problem Statement

The codebase is currently in the common “strong backend, partial product” phase. Major backend capabilities exist, but many of the concepts are visible only indirectly. An engineer can infer the model by reading:

- `pkg/sites/registry/registry.go`
- `pkg/engine/model/types.go`
- `pkg/engine/scheduler/scheduler.go`
- `pkg/engine/store/sqlite/store.go`
- `pkg/sites/submitverbs/runtime.go`
- `pkg/js/runtime/executor.go`

but an operator or new engineer using the UI does not get the same clarity.

That shows up in three roles.

### Role 1: Operator running many jobs over days

This person needs to answer:

- Which workflows are active, blocked, retrying, or failing?
- Which queues are saturated or rate-limited?
- Which site is responsible for a workflow?
- What pacing and proxy assumptions is the worker running under?
- Where do I go when a workflow stalls?

Today, some of that data exists, but it is split awkwardly across:

- overview counts,
- queue monitor,
- workflows table,
- workflow detail,
- events page,
- site detail,
- CLI help pages.

### Role 2: Verb builder debugging site workflows

This person needs to answer:

- What exactly did my submit verb emit?
- Which queue did each emitted op land in?
- Which dependencies were created?
- Why is an op still pending?
- Which runtime event or artifact explains the failure?

Today the information exists, but the UI still lacks a graph-minded, script-minded debugging path.

### Role 3: New engineer onboarding

This person needs to answer:

- What is the difference between a submit verb, an op, a script, and a workflow?
- Where are site capabilities declared?
- How do rate limits and queues relate to site definitions and emitted ops?
- Which docs matter first?
- Which smoke tests prove the system works?

The CLI docs are already good, but the web UI currently does very little teaching.

## Terminology and Mental Model

The code uses several related words that need to be separated explicitly.

### Site

A site is the top-level package/registry entry that bundles:

- site name,
- site DB name,
- scripts,
- submit verbs,
- migrations,
- optional queue policies,
- optional help assets,
- runtime modules.

The canonical struct is `registry.Definition` in `pkg/sites/registry/registry.go:14-30`.

### Verb

A verb is a JavaScript-defined submission command discovered from a site’s `verbs/` tree. It is a CLI/API entrypoint, not a worker job. Its job is to describe the initial durable work graph.

### Workflow

A workflow is the durable run container. It has an ID, site, name, input, metadata, timestamps, and a lifecycle status. It groups a set of ops.

The canonical type is `model.WorkflowRun` in `pkg/engine/model/types.go:44-53`.

### Op

An op is one durable unit of execution. It has:

- runner kind,
- queue key,
- dedup key,
- retry policy,
- metadata,
- dependencies,
- optional parent op,
- input.

The canonical type is `model.OpSpec` in `pkg/engine/model/types.go:124-137`.

### Queue

A queue is the control-plane pacing key. It is not just a label. It is the key that the scheduler and store use for:

- max in-flight checks,
- token-bucket rate limiting,
- queue status reporting.

### Dependency

A dependency is an edge from one op to another. Required dependencies must succeed before the dependent op becomes ready. Optional dependencies may fail or cancel without blocking readiness.

The canonical type is `model.Dependency` in `pkg/engine/model/types.go:55-58`.

## Current Architecture

The current system can be summarized like this:

```text
site definition
  -> register scripts + verbs + queue policies + migrations
  -> submit verb emits initial OpSpecs
  -> engine store persists workflow + ops + op_dependencies
  -> worker scheduler refreshes pending/ready state
  -> store enforces queue policy transactionally
  -> runner executes js or http/fetch
  -> result persists data + artifacts + emitted child ops
  -> API exposes workflows, queues, sites, artifacts, runtime events
  -> frontend polls snapshots and streams runtime events
```

### The site registry is the capability seam

`registry.Definition` already has the right long-term fields:

- `HelpFS` / `HelpRoot`
- `QueuePolicies`
- `RuntimeModuleRegistrars`
- scripts and verbs roots

That is in `pkg/sites/registry/registry.go:14-30`.

The important operational detail is in `QueuePolicyProvider()` in `pkg/sites/registry/registry.go:75-92`: if a site definition does not provide a queue policy for a queue, the system falls back to `model.DefaultQueuePolicy()`.

### Built-in sites currently ship mostly default queue policy

The default registry registers four built-in sites in `pkg/sites/defaults/defaults.go:19-35`:

- hackernews
- slashdot
- js-demo
- nereval

The built-in site definitions inspected here:

- `pkg/sites/jsdemo/site.go:14-25`
- `pkg/sites/hackernews/site.go:14-23`
- `pkg/sites/slashdot/site.go:14-24`
- `pkg/sites/nereval/site.go:15-26`

set scripts, verbs, migrations, and sometimes modules, but do not define `QueuePolicies`.

So the answer to “do rate limits work?” is:

- yes in the engine,
- mostly no in shipped default site behavior,
- yes in tests and in any registry override that defines policy.

### Rate limiting is real and durable

The policy model is explicit in `pkg/engine/model/types.go:74-115`:

- `RateLimitPolicy`
- `QueuePolicy`
- `DefaultQueuePolicy()`
- `Normalize()`

The scheduler uses the site/queue policy when attempting leases in `pkg/engine/scheduler/scheduler.go:226-258`.

The actual race-safe enforcement happens in `pkg/engine/store/sqlite/store.go:407-472`:

1. load queue limiter state,
2. refill tokens,
3. block if tokens are below one,
4. select a ready op,
5. decrement tokens,
6. persist updated limiter state,
7. create the lease.

This is not an in-memory best-effort limiter. It is a durable store-enforced limiter.

The repository docs say the same thing directly in `pkg/doc/topics/scraper-queue-policies-and-rate-limiting.md:23-25` and `:54-68`.

The repository tests also prove the intended path. A command-path test injects explicit rate policy into `js-demo` and `hackernews` registry definitions in `pkg/cmd/site_test.go:223-301` and `:355-420`.

### Per-verb rate limits are indirect, not first-class

The queue key lives on each emitted op spec. The JS API reference documents this in `pkg/doc/topics/scraper-js-api-reference.md:150-177`, where `queue` is part of the `ctx.emit(...)` shape.

That means a verb can choose the queue of the ops it emits. For example:

- `pkg/sites/hackernews/verbs/seed.js:31-41`
- `pkg/sites/jsdemo/verbs/summary.js:37-66`

Because policy is attached to `site + queue`, a verb can effectively get separate pacing if:

1. it emits ops to a dedicated queue, and
2. the site definition provides policy for that queue.

So:

- per-queue policy: first-class and real,
- per-verb policy: not first-class,
- per-op policy: not first-class,
- per-verb/per-op pacing via dedicated queues: possible today.

This distinction should be made explicit in product docs and UI.

### Dependencies are explicit durable edges

Dependencies are defined in the op model at `pkg/engine/model/types.go:55-58` and attached to `OpSpec` at `:124-137`.

They are created in both JS runtimes:

- submit verb runtime in `pkg/sites/submitverbs/runtime.go:274-332` and `:560-583`
- worker JS runtime in `pkg/js/runtime/executor.go:327-384`

In both cases, JS emits:

```javascript
ctx.emit({
  ...,
  dependsOn: [{ opID: "other-op", required: true }]
})
```

The store persists those edges into `op_dependencies` in `pkg/engine/store/sqlite/store.go:985-997`.

Readiness is then derived in `RefreshRunnableOps()`:

- blocked required dependencies cancel downstream pending ops when upstream fails or cancels in `pkg/engine/store/sqlite/store.go:251-280`
- pending ops become ready only when dependency conditions are satisfied in `:282-307`
- ops without dependencies start as `ready`; ops with dependencies start as `pending` in `:1034-1039`

This is a strong model. The main weakness is not implementation. The weakness is visibility. The frontend does not yet show dependency graphs or “why pending?” explanations clearly.

### Proxy support exists in the worker, but is not visible in the product

The worker has an explicit `--http-proxy` flag in `pkg/cmd/worker.go:121-129`.

That value is passed into `config.HTTP.ProxyURL` and on to `runner.NewHTTPRunner(...)` in `pkg/cmd/worker.go:39-55` and `:85-87`.

The HTTP runner builds a cloned transport with `http.ProxyURL(...)` when `ProxyURL` is set in `pkg/engine/runner/http.go:25-48`.

There is also a prior dedicated ticket documenting the rollout:

- `SCRAPER-HTTP-PROXY`
- `ttmp/2026/03/24/SCRAPER-HTTP-PROXY--http-proxy-support-for-durable-scraper-worker/design/01-http-proxy-support-architecture-and-implementation-guide.md`

So the answer to “is there a proxy we used in the past in ticket history?” is yes. There is both:

- implemented worker-level proxy support in code, and
- a design-history ticket documenting it.

What is still missing is API/UI visibility:

- no API endpoint reports current worker proxy configuration,
- no workflow or queue page surfaces whether HTTP work is going through a proxy,
- no site or operator page explains proxy policy.

### The catalog API already exposes more metadata than the workflows page uses

The catalog service already exposes:

- site summaries,
- site detail,
- queue policies,
- verb summaries,
- field help and defaults.

That is all in `pkg/services/catalog/service.go:28-85` and `:223-287`, and the API handlers expose it in `pkg/api/handlers/catalog.go:44-101`.

The site detail page uses that reasonably well already:

- queue policy table,
- verb count,
- script count,
- verb and script tabs,

in `web/src/pages/SiteDetailPage.tsx:61-166`.

But the workflows table in `web/src/components/workflows/WorkflowTable.tsx:61-129` shows site only as a static chip, not as a link into the site context. That is a missed information-architecture connection.

### Help exists, but mostly in the CLI and field metadata

The repo already embeds help pages from `pkg/doc/topics/*` and `pkg/doc/tutorials/*` through `pkg/doc/doc.go:9-13` and wires them into the root command in `pkg/cmd/root.go:33-40`.

There is already strong help coverage for:

- architecture overview,
- runtime model,
- queue policies and rate limiting,
- JS API reference,
- onboarding tutorial,
- site authoring.

The frontend also already renders verb field help in `web/src/components/submit/VerbParameterForm.tsx:28-118`.

What is missing is a frontend help model that connects these pieces:

- no global help entry in the app shell,
- no contextual “what is a workflow / queue / site / op?” help,
- no surfacing of embedded docs through HTTP,
- no site-specific help even though the registry definition already reserves `HelpFS` and `HelpRoot`.

### Additional review findings

#### Queue monitor throughput is currently placeholder data

`web/src/pages/QueueMonitorPage.tsx:9-25` hardcodes random throughput series and uses them in `:80-83`.

That is acceptable during development, but not for an operator-facing control-plane screen. It introduces false confidence and should either be labeled as simulated or replaced.

#### Workflow list is not yet scaled for long-running operations

`engineview.ListWorkflows()` clamps list size to `<= 200` and defaults to `50` in `pkg/services/engineview/service.go:200-214`, while the workflows page currently fetches `limit: 50` and filters only by `site` and `status` in `web/src/pages/WorkflowsPage.tsx`.

For “1000s of requests over days,” the project will need:

- stronger filtering,
- paging UX,
- richer search fields,
- possibly retention/archive cues,
- a clearer distinction between active and historical workflows.

#### Site-specific help is structurally planned but unused

The registry has `HelpFS` / `HelpRoot` in `pkg/sites/registry/registry.go:26-27`, but `rg` over `pkg/sites` found no built-in site assigning those fields. So site-local help is a reserved seam, not a used feature.

## Answers To The Requested Questions

## 1. Do rate limits work?

Yes. They are implemented and durable.

Evidence:

- policy types: `pkg/engine/model/types.go:74-115`
- scheduler policy resolution: `pkg/engine/scheduler/scheduler.go:226-258`
- transactional enforcement and token persistence: `pkg/engine/store/sqlite/store.go:407-472`
- documentation and tests: `pkg/doc/topics/scraper-queue-policies-and-rate-limiting.md:23-25`, `:54-68`, and `pkg/cmd/site_test.go:223-301`, `:355-420`

Operational caveat:

- built-in sites currently do not define non-default queue policies in their shipped definitions
- therefore the default runtime experience mostly behaves as “max 1 in-flight, no explicit rate limit”

## 2. Can individual verbs or ops have their own rate limits?

Not directly.

Current truth:

- policy is attached to `site + queue`
- ops choose their queue
- verbs emit ops

Therefore:

- verbs can simulate dedicated pacing by emitting to dedicated queues
- ops can simulate special pacing by choosing those queues
- but there is no first-class “verb policy” or “op policy” declaration layer

Recommendation:

- keep policy at the queue level
- make queue choice much more visible in both authoring docs and UI
- if product language wants “verb rate limits,” implement that as a build-time helper that generates queue policy plus queue naming conventions, not as a second competing rate-limit system

## 3. How are dependencies defined and how are they created when verbs are submitted?

Dependencies are plain op-spec edges emitted by JavaScript.

Submit path:

- submit verb calls `ctx.emit({... dependsOn: [...] })`
- submit runtime converts the JS object to `model.OpSpec`
- store writes `op_dependencies`

Worker path:

- running JS op calls `ctx.emit({... dependsOn: [...] })`
- worker-side runtime converts the JS object to `model.OpSpec`
- persisted emitted child ops go through the same dependency table

The readiness algorithm is in the store, not in JS.

Recommendation:

- add a dependency graph view in the workflow detail page
- show “pending because waiting on X” explanations
- keep terminology consistent: use “verb” only for submit entrypoints and “op” for durable executable jobs

## 4. Display proxy settings?

Current state:

- worker CLI supports `--http-proxy`
- HTTP runner uses it
- prior proxy ticket exists
- API/frontend do not surface it

Recommendation:

- add proxy visibility to `/api/v1/info` or a new worker/runtime config endpoint
- show proxy mode in the overview or a dedicated “runtime configuration” panel
- include a clear distinction between:
  - direct,
  - env-driven,
  - explicit proxy URL

Do not expose secrets. Show mode and sanitized host, not credentials.

## 5. Display site information and link site information in workflows page?

Current state:

- workflows page shows site as a chip only
- site detail page is already useful and exists

Recommendation:

- make the site chip in the workflows table clickable
- add site iconography and site DB/queue policy summary in hover or expanded context
- add site column filters and site detail cross-links from workflow detail and queue pages

## 6. Which help pages and tooltips should be shown?

Current state:

- CLI has real help pages
- submit form shows verb field help
- general frontend pages are mostly self-explanatory only if the reader already understands the domain

Recommendation:

- expose embedded docs in the frontend through a lightweight docs API
- add concise contextual help blocks, not prose walls, on:
  - Overview: what counts mean
  - Workflows: workflow vs op
  - Queue Monitor: queue key, max in-flight, tokens, rate/sec
  - Submit: what a verb does and what happens after submission
  - Site detail: site package, verbs, scripts, queue policy
  - Workflow detail: target op, dependency semantics, runtime event meaning

## Role-Based Gap Analysis

## Operator Gaps

Needs:

- trustable queue pacing visibility,
- workflow/site navigation,
- runtime configuration visibility,
- real throughput/history rather than placeholder charts,
- long-running workload support.

Current gaps:

- placeholder throughput chart,
- no proxy display,
- weak cross-linking between workflows and sites,
- limited workflow search/filter model,
- no global explanation of queues or tokens in the UI.

## Verb Builder Gaps

Needs:

- graph visibility,
- emitted op inspection,
- dependency explanations,
- queue choice visibility,
- faster path from runtime event to script source and site context.

Current gaps:

- no dependency graph view,
- no explicit op graph on workflow detail,
- no “why pending?” UI,
- no direct linkage from op queue to site queue policy explanation,
- limited surfacing of emitted op structure after submission.

## Onboarding Gaps

Needs:

- consistent vocabulary,
- embedded docs reachable from the product,
- safe smoke-test paths,
- explanation of verbs vs ops vs scripts vs queues.

Current gaps:

- docs exist but are mostly CLI-only,
- site help seam exists but is unused,
- app shell has no help center,
- frontend teaches little about the system model.

## Recommended Product Direction

The system should be organized around three explicit frontend narratives.

## Narrative A: Operate the system

Primary pages:

- Overview
- Queues
- Workflows
- Events

Key additions:

- real throughput and rate-limit explanations,
- proxy/runtime config surface,
- workflow/site/queue cross-linking,
- active vs historical workflow segmentation.

## Narrative B: Build and debug verbs

Primary pages:

- Sites
- Site detail
- Workflow detail
- Op drawer
- Runtime events

Key additions:

- emitted-op and dependency graph visibility,
- queue choice visibility,
- better linkage from workflow -> site -> script/verb source,
- artifact and runtime-event debugging shortcuts.

## Narrative C: Learn the system

Primary pages:

- Help center
- Site detail
- Submit

Key additions:

- frontend docs endpoint backed by embedded help pages,
- contextual help cards,
- onboarding playbook surfaced in product.

## Proposed Solution

## 1. Clarify the model in both backend docs and frontend language

Introduce a consistent glossary everywhere:

- Site
- Verb
- Workflow
- Op
- Queue
- Dependency
- Artifact
- Runtime event

Use “verb” only for submit entrypoints. Use “op” for durable units of work.

## 2. Keep queue policy where it already belongs: site + queue

Do not introduce a second rate-limit model. Instead:

- expose queue selection more clearly in authoring docs and UI,
- provide conventions for dedicated per-verb queues,
- add optional site helpers that make queue naming and queue policy declaration simpler.

Example future helper pseudocode:

```text
site definition:
  queues:
    "site:hackernews:http":
      maxInFlight: 2
      rateLimit: 1 req/sec, burst 2
    "site:hackernews:frontpage-join":
      maxInFlight: 1

verb emits:
  queue = "site:hackernews:frontpage-join"
```

## 3. Make site metadata a first-class navigation object

The site detail page already exists. Use it more aggressively.

Implementation ideas:

- make site chips in workflow and queue tables clickable
- add “View site” links in workflow detail and op drawer
- show queue-policy snippets in workflow pages when an op queue is visible

## 4. Surface proxy/runtime configuration safely

Add a minimal runtime-config surface.

Possible API sketch:

```json
GET /api/v1/info
{
  "version": "...",
  "address": "...",
  "engineDB": "...",
  "sitesDir": "...",
  "http": {
    "proxyMode": "explicit",
    "proxyHost": "proxy.example.net:8080"
  }
}
```

or a new endpoint:

```json
GET /api/v1/runtime/config
```

The server should never expose proxy credentials.

## 5. Add dependency visibility to workflow debugging

The data is already present in `WorkflowOps`.

A good first implementation is not a fancy graph library. Start with:

- dependency badges on ops,
- a “blocked by” list for pending ops,
- a drawer panel showing:
  - parent op,
  - required dependencies,
  - optional dependencies,
  - current dependency statuses.

Pseudocode:

```text
for each workflow op:
  if status == pending and dependsOn not empty
    show badge: "waiting on N deps"
    show first few dependency ids/statuses

op drawer:
  load workflow ops
  join op.dependsOn against workflow op status map
  render required and optional dependency sections
```

## 6. Build a frontend help layer on top of existing docs

The embedded docs system is already good. Reuse it.

Implementation options:

- Option A: API endpoint that returns rendered help pages or structured metadata
- Option B: import a curated subset of help markdown into the frontend build

Recommended first slice:

- add a small docs API,
- expose:
  - architecture overview,
  - runtime model,
  - queue policy/rate limit,
  - onboarding tutorial,
- link them from the app shell and page-level help affordances.

## API References

Existing useful endpoints:

- `GET /api/v1/info` — server/runtime basics in `pkg/api/server/server.go:57`
- `GET /api/v1/sites` — site list in `:58`
- `GET /api/v1/sites/{site}/detail` — rich site detail in `:60`
- `GET /api/v1/sites/{site}/verbs` — verb metadata in `:61`
- `GET /api/v1/workflows` — workflow list in `:65`
- `GET /api/v1/workflows/{workflowID}` — workflow summary in `:66`
- `GET /api/v1/workflows/{workflowID}/ops` — op list in `:67`
- `GET /api/v1/queues` — queue status in `:89`
- `GET /api/v1/runtime-events` and `/stream` — event history/live stream in `:70-71`

Recommended additions:

- `GET /api/v1/help/{slug}` or `GET /api/v1/docs/{slug}`
- richer runtime configuration data in `/api/v1/info` or a new config endpoint

## Implementation Plan

## Phase 0: Documentation and vocabulary

- clarify glossary in docs and UI copy
- add frontend-facing explanation of queue = pacing domain
- document “per-verb rate limit via dedicated queue” as the current supported pattern

Files:

- `pkg/doc/topics/scraper-runtime-model.md`
- `pkg/doc/topics/scraper-js-api-reference.md`
- new frontend help ticket/docs as needed

## Phase 1: Operator information architecture

- make site chips in workflow/queue views clickable
- add site quick links from workflow detail
- add overview cards that link into queues/sites/events

Files:

- `web/src/components/workflows/WorkflowTable.tsx`
- `web/src/pages/WorkflowDetailPage.tsx`
- `web/src/components/queues/QueueStatusTable.tsx`

## Phase 2: Queue and rate-limit clarity

- replace or label placeholder throughput chart
- show token/rate-limit explanations inline
- expose “default policy” vs “explicit policy” more clearly
- add queue filter/search improvements

Files:

- `web/src/pages/QueueMonitorPage.tsx`
- `web/src/components/queues/QueueStatusTable.tsx`
- `web/src/components/queues/QueueDetailPanel.tsx`

## Phase 3: Dependency and builder debugging UX

- show dependency counts in op table
- add blocked-by detail in op drawer
- surface parent/emitted relationships
- link runtime events/artifacts/scripts more tightly

Files:

- `web/src/components/workflows/OpTable.tsx`
- `web/src/components/workflows/OpDetailDrawer.tsx`
- `web/src/pages/WorkflowDetailPage.tsx`

## Phase 4: Proxy and runtime configuration visibility

- expose proxy/runtime config through API
- show current proxy mode in operator UI
- sanitize any displayed values

Files:

- `pkg/api/handlers/catalog.go` or new handler
- `pkg/api/types`
- `web/src/api/...`
- overview/runtime-config frontend component

## Phase 5: Help and onboarding inside the product

- add frontend help center entry
- expose embedded docs through HTTP
- add page-level contextual help cards/tooltips

Files:

- `pkg/doc/doc.go`
- new API handler/service for docs
- `web/src/components/layout/AppShell.tsx`
- targeted page components

## Testing and Validation Strategy

### Backend and model validation

- `go test ./... -count=1`
- queue/rate-limit regression tests remain authoritative
- add handler tests for new docs/config endpoints if introduced

### Frontend validation

- `npm run test:unit`
- `npm run build`
- page-level tests for:
  - site link navigation from workflows
  - queue-policy/help explanations
  - dependency rendering
  - proxy config display

### Product validation by persona

Operator smoke test:

1. submit workflows across two sites
2. verify workflow list links into site context
3. verify queue page explains explicit vs default policy
4. verify runtime configuration/proxy mode is visible

Verb builder smoke test:

1. submit a workflow with dependencies
2. inspect dependency state in workflow detail
3. trace from runtime event to op to site/script context

Onboarding smoke test:

1. start in frontend help center
2. navigate to sites
3. submit `js-demo`
4. inspect workflow and queue pages
5. understand the core model without reading source first

## Risks and Alternatives

### Risk: overloading the frontend with too much documentation

Mitigation:

- keep page-level help concise,
- link to deeper docs instead of embedding entire manuals inline.

### Risk: inventing a second policy model for verbs

Mitigation:

- keep the queue as the only pacing domain,
- build convenience layers on top of queues, not beside them.

### Risk: exposing sensitive proxy data

Mitigation:

- expose proxy mode and sanitized host only,
- never return credentials.

### Alternative: leave docs in CLI only

Rejected because the current operator UI is already trying to be a real control plane. Once there is a web UI, the core model has to be understandable there too.

### Alternative: build graph visualization before simpler dependency explanations

Rejected for the first slice. A graph view may come later, but a “blocked by” explanation and dependency badges are cheaper and cover the immediate debugging need.

## Open Questions

- Should the frontend help system render markdown directly from the server, or should it consume structured help sections?
- Should proxy visibility be shown globally only, or also per workflow/op when the relevant runner kind is `http/fetch`?
- Should the first “per-verb rate limit” UX be documentation-only, or should the site catalog explicitly show which queues each verb commonly emits into?
- For very large workflow histories, should the product introduce workflow archive views or saved filters?

## Short Recommendation

Do not redesign the engine. The core runtime model is already sound. The next project should be a visibility-and-legibility overhaul that makes queues, sites, dependencies, runtime configuration, and help first-class operator concepts in the product.
