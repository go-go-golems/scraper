---
Title: HTTP API architecture and implementation guide
Ticket: SCRAPER-HTTP-API
Status: active
Topics:
    - scraper
    - http-api
    - api
    - server
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/cmd/root.go
      Note: Current CLI bootstrap where an HTTP server command would be added
    - Path: pkg/cmd/site.go
      Note: Existing site command tree and dynamic submit-verb registration flow
    - Path: pkg/cmd/worker.go
      Note: Worker polling model that should remain the main execution process
    - Path: pkg/cmd/engine.go
      Note: Existing engine-status read path that can be lifted into HTTP handlers
    - Path: pkg/sites/submitverbs/host.go
      Note: Best current seam for HTTP workflow submission because it already prepares DBs and persists initial ops
    - Path: pkg/sites/submitverbs/runtime.go
      Note: Submission-time JS runtime and `ctx.emit(...)` contract used by submit verbs
    - Path: pkg/engine/scheduler/scheduler.go
      Note: Durable workflow creation and worker-side processing loop
    - Path: pkg/engine/store/store.go
      Note: Store contracts that future HTTP handlers should call through service helpers
    - Path: pkg/engine/store/sqlite/store.go
      Note: SQLite implementation details and likely source for read-side query helpers
ExternalSources: []
Summary: Detailed intern-oriented design guide for adding an HTTP API to the durable scraper engine, centered on JS submit-verb discovery for workflow submission, read-only engine inspection endpoints, and a clear separation between HTTP submission and worker polling.
LastUpdated: 2026-03-23T21:20:00-04:00
WhatFor: Explains how to expose the existing durable scraper system through an HTTP server without collapsing the current submit-verb and worker separation.
WhenToUse: Use when implementing or reviewing the scraper HTTP server, designing submission and inspection endpoints, or onboarding a new developer to the API surface and its underlying runtime boundaries.
---

# HTTP API architecture and implementation guide

## Executive Summary

The scraper system already has most of the architectural pieces needed for an HTTP API. It can:

- discover site-specific JS submit verbs
- prepare engine and site databases
- execute one submission-time JS function that writes a durable workflow and initial ops
- run a long-lived worker that polls the engine database and processes ready ops
- inspect engine state through operator-oriented CLI commands

What it does not have yet is an HTTP server that exposes those capabilities in a stable, remote-callable way.

The recommended first version is intentionally narrow. The HTTP API should not try to become a second scheduler. It should stay a thin Go host around the existing durable architecture:

1. Accept HTTP requests.
2. Resolve a site and JS submit verb.
3. Run the same preparation logic the CLI already uses.
4. Submit durable workflows and initial ops.
5. Return workflow IDs and useful metadata.
6. Expose a small read API for workflow and engine status.
7. Leave actual op execution to the existing worker process.

For a new intern, the most important mental model is:

- HTTP submit endpoint = "run one JS submit verb and persist the initial workflow"
- worker process = "poll the DB and execute queued work"
- JS op runtime = "run later, inside workers, to do actual scraping and fan-out"

The API should preserve that separation rather than hiding it.

## Problem Statement

### What the system can do today

The CLI already exposes the durable scraper engine in two important ways:

- `scraper site ...` commands prepare DBs, run site migrations, and invoke JS submit verbs through [host.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/submitverbs/host.go)
- `scraper worker run` starts the long-lived polling loop that executes ready ops through [worker.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/worker.go)

The architectural direction is already strong:

- submission is durable
- work is queued in SQLite
- workers are separate from submitters
- JS submit verbs are flexible enough to emit different workflow shapes

### What is missing

There is no HTTP API for:

- submitting workflows remotely
- listing available site submit verbs
- inspecting workflow state without shelling out to CLI commands
- integrating the scraper with other local tools and user interfaces

That gap matters because the system is now rich enough that a user should not need to spawn shell commands for every integration. A future web UI, local control panel, automation script, or external service will want a stable HTTP surface.

### Why not just wrap the CLI

A naive implementation could shell out to:

- `scraper site ...`
- `scraper engine status`
- `scraper worker run`

That would work as a demo, but it would be the wrong architecture because it would:

- duplicate parsing and formatting logic
- make error handling string-based
- couple the API to terminal-oriented text output
- make testing harder
- discourage extracting reusable services

The correct direction is to move the useful logic behind shared Go services, then let both the CLI and HTTP server call those services.

## Goals

### Primary goals

- Expose a minimal but real HTTP API for workflow submission and inspection.
- Reuse the existing JS submit-verb model rather than inventing a second submission DSL.
- Keep the worker as a separate long-running polling process.
- Keep the first API local-operator friendly and easy to test.
- Produce an implementation path that is understandable for a new intern.

### Non-goals for the first slice

- authentication and multi-user permissions
- public internet deployment
- websocket live streaming
- browser UI
- distributed job queues beyond the existing SQLite-backed engine
- replacing the CLI

## Current System Model

### Current control paths

The current system has three important paths.

#### Path 1: CLI submit path

`scraper site <site> <verb> ...`

This is assembled in [site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site.go), which dynamically registers per-site JS submit verbs through [`pkg/sites/submitverbs/register.go`](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/submitverbs/register.go).

The heavy lifting then happens inside [host.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/submitverbs/host.go):

- open engine DB
- open scraper DB handle
- migrate site DB
- open site DB
- build submission workflow shell
- execute one JS submit verb
- persist workflow and initial ops

That file is the best existing seam for HTTP submission.

#### Path 2: worker execution path

`scraper worker run`

This lives in [worker.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/worker.go). It:

- opens the engine store
- opens the scraper DB
- constructs the runner registry
- constructs a scheduler
- polls the engine DB for ready ops
- executes them

This process should remain the main execution engine after the HTTP API is added.

#### Path 3: engine inspection path

`scraper engine status`
`scraper engine migrations status`

These live in [engine.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/engine.go) and already expose a useful read-side view of the engine DB. They are not HTTP-ready yet, but they prove that the underlying inspection data is already valuable.

### Current JS runtime split

The system now has two different JS contexts.

#### Submit-verb runtime

Defined in [runtime.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/submitverbs/runtime.go).

This runtime is invoked once per command submission. It exposes a `ctx` shaped around:

- command metadata
- parsed argument values
- workflow metadata
- `ctx.emit(...)` for initial op submission
- workflow naming and targeting helpers

This runtime is not an op executor. It is a workflow builder.

#### Op runtime

Defined elsewhere in the scraper JS runtime and invoked later by workers. This is the runtime that executes queued ops like:

- `js`
- `http/fetch`

This runtime handles actual durable scraping and can emit follow-up child ops during execution.

The HTTP API design must preserve this distinction.

## Proposed Solution

### High-level architecture

Add a new Go-hosted HTTP server to the scraper CLI. Its job is to expose:

- submit-verb discovery
- workflow submission
- workflow and engine inspection

It should not itself become a worker in the first implementation.

The recommended command is:

```text
scraper api serve
```

### First-version API surface

The first version should expose five families of endpoints.

#### 1. Health and metadata

- `GET /healthz`
- `GET /api/v1/info`

These answer:

- is the server up
- what version/build is running
- what DB paths and mode are configured

#### 2. Site and verb discovery

- `GET /api/v1/sites`
- `GET /api/v1/sites/{site}`
- `GET /api/v1/sites/{site}/verbs`
- `GET /api/v1/sites/{site}/verbs/{verb}`

These are read-only and should expose the information already available from the JS submit-verb registry:

- site name
- verb name
- full command path
- help text
- parameter schema or sections
- source file

This is important because clients need a way to learn what they can submit.

#### 3. Workflow submission

- `POST /api/v1/sites/{site}/verbs/{verb}:submit`

Request body:

```json
{
  "workflowId": "optional-client-chosen-id",
  "values": {
    "base-url": "https://slashdot.org/",
    "max-pages": 2
  }
}
```

Response body:

```json
{
  "workflow": {
    "id": "slashdot-live-001",
    "site": "slashdot",
    "name": "slashdot seed workflow",
    "status": "pending"
  },
  "targetOpID": "slashdot-live-001:seed",
  "submittedCount": 1,
  "siteDBPath": "state/sites/slashdot.db",
  "verbData": {
    "baseURL": "https://slashdot.org/",
    "maxPages": 2
  }
}
```

This endpoint should call the same host logic as the CLI submit path.

#### 4. Workflow inspection

- `GET /api/v1/workflows/{workflowID}`
- `GET /api/v1/workflows/{workflowID}/ops`
- `GET /api/v1/workflows/{workflowID}/results`
- `GET /api/v1/workflows/{workflowID}/artifacts`

The first version can start smaller:

- `GET /api/v1/workflows/{workflowID}`
- `GET /api/v1/workflows/{workflowID}/ops`

The point is to make workflow state remotely inspectable without direct SQLite access.

#### 5. Engine status

- `GET /api/v1/engine/status`
- `GET /api/v1/engine/migrations`

These should mirror the data already produced by `scraper engine status` and `scraper engine migrations status`, but return JSON rather than terminal text.

### Request flow

The submit flow should look like this:

```text
HTTP client
    |
    v
POST /api/v1/sites/js-demo/verbs/seed:submit
    |
    v
Go HTTP handler
    |
    v
resolve site definition + JS verb spec
    |
    v
prepare engine DB + site DB + migrations
    |
    v
invoke submit-verb JS runtime once
    |
    v
create workflow + initial ops in engine DB
    |
    v
return workflow ID + metadata
```

Then later:

```text
worker process
    |
    v
poll engine DB
    |
    v
lease ready ops
    |
    v
run js/http ops
    |
    v
persist results and child ops
```

This is exactly the same durability model as the CLI path, just with HTTP in front.

## Design Decisions

### Decision 1: keep HTTP submission and worker execution separate

Do not make the HTTP server run the scheduler inline after every submit request.

Why:

- the worker already exists
- durable execution is the core architecture
- inline execution would blur submission and processing
- later multi-client and background execution stories become cleaner

A future `wait=true` query parameter may poll workflow status, but should still not become the main processing model.

### Decision 2: reuse JS submit verbs as the HTTP submission language

The existing submit-verb mechanism is already the correct "workflow plan" boundary. The HTTP API should discover and invoke those verbs rather than inventing:

- a separate YAML workflow format
- ad hoc JSON workflow specs
- hand-written site-specific HTTP handlers

This preserves one source of truth for submission behavior.

### Decision 3: extract service helpers instead of shelling out to CLI commands

The HTTP handlers should not execute shell commands. They should call Go helpers that are also usable by CLI code.

Likely new packages:

- `pkg/api/server`
- `pkg/api/handlers`
- `pkg/api/types`
- `pkg/services/submission`
- `pkg/services/engineview`

The names may change, but the layering principle matters:

- transport layer converts HTTP to typed requests
- service layer contains reusable business logic
- store/runtime layers remain unchanged

### Decision 4: use standard library HTTP first

For the first slice, `net/http` is enough. The API surface is small and predictable. Introducing `chi`, `gorilla/mux`, or another router is optional later, not necessary now.

That keeps the implementation:

- easier to read
- easier for an intern to debug
- consistent with the project’s current size

### Decision 5: JSON responses only

The HTTP API should return JSON and structured errors. Do not reuse human-readable CLI output strings.

Example error shape:

```json
{
  "error": {
    "code": "verb_not_found",
    "message": "submit verb not found",
    "details": {
      "site": "slashdot",
      "verb": "seed"
    }
  }
}
```

### Decision 6: treat the first version as local-operator API

The first implementation should assume local trusted use:

- bind default to `127.0.0.1`
- no auth in first slice
- conservative endpoints only

This keeps scope under control while leaving room for later auth work.

## Detailed API Shape

### Endpoint table

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/api/v1/info` | Server metadata and configured paths |
| `GET` | `/api/v1/sites` | List registered sites |
| `GET` | `/api/v1/sites/{site}` | Show site metadata |
| `GET` | `/api/v1/sites/{site}/verbs` | List JS submit verbs for a site |
| `GET` | `/api/v1/sites/{site}/verbs/{verb}` | Show one verb definition |
| `POST` | `/api/v1/sites/{site}/verbs/{verb}:submit` | Submit a workflow through one JS submit verb |
| `GET` | `/api/v1/workflows/{workflowID}` | Show workflow summary |
| `GET` | `/api/v1/workflows/{workflowID}/ops` | List ops for the workflow |
| `GET` | `/api/v1/engine/status` | Engine summary |
| `GET` | `/api/v1/engine/migrations` | Engine migration summary |

### Read-side JSON types

Suggested response types:

```text
ServerInfoResponse
SiteSummary
VerbSummary
VerbDetail
SubmitWorkflowRequest
SubmitWorkflowResponse
WorkflowSummary
WorkflowOpSummary
EngineStatusResponse
MigrationStatusResponse
APIError
```

Put these in one place, for example:

- [`pkg/api/types`](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/api)

That way CLI and HTTP tooling can share them later if needed.

### Submission request normalization

The submit endpoint should accept a plain JSON object under `values`. Those key-value pairs then need to be converted into the same parsed-value shape that the CLI submit host expects.

That means the HTTP API needs a small adapter:

```text
JSON body values
    |
    v
typed map[string]any
    |
    v
Glazed parsed values builder
    |
    v
submitverbs.Host.Submit(...)
```

This is one of the most important practical implementation details.

## Service-Layer Design

### Why add service helpers

Right now the submit and status logic is still very command-oriented. For an HTTP API we need reusable functions that are not tightly coupled to Cobra.

Recommended services:

#### Submission service

Responsibilities:

- resolve site definition
- resolve submit verb
- convert HTTP input into parsed values
- call submit host
- return typed result

Likely built on top of [host.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/submitverbs/host.go).

#### Catalog service

Responsibilities:

- list sites
- list submit verbs
- expose help/metadata for one verb

Likely built on the site registry and submit-verb registry.

#### Engine view service

Responsibilities:

- fetch engine status
- fetch migration status
- fetch workflow summary and op list

This will probably need some new store-side read helpers because current status logic is still mainly packaged for the CLI.

### Recommended package diagram

```text
pkg/api/server
    |
    +-- pkg/api/handlers
    |       |
    |       +-- pkg/api/types
    |
    +-- pkg/services/catalog
    +-- pkg/services/submission
    +-- pkg/services/engineview
            |
            +-- pkg/sites/submitverbs
            +-- pkg/sites/registry
            +-- pkg/engine/store/sqlite
            +-- pkg/engine/scheduler
```

The server and handlers should stay thin.

## Pseudocode

### Server bootstrap

```go
func newAPICommand(siteRegistry *siteregistry.Registry) *cobra.Command {
    opts := &apiOptions{}

    cmd := &cobra.Command{
        Use:   "api",
        Short: "Run the scraper HTTP API",
    }

    serveCmd := &cobra.Command{
        Use:   "serve",
        Short: "Serve the HTTP API",
        RunE: func(cmd *cobra.Command, args []string) error {
            services, err := api.NewServices(siteRegistry, api.Config{
                EngineDB: opts.EngineDB,
                SitesDir: opts.SitesDir,
            })
            if err != nil {
                return err
            }

            srv := api.NewServer(services, api.ServerConfig{
                Address: opts.Address,
            })
            return srv.ListenAndServe(cmd.Context())
        },
    }

    return cmd
}
```

### Submit handler

```go
func (h *Handlers) submitWorkflow(w http.ResponseWriter, r *http.Request) {
    site := readPathParam(r, "site")
    verb := readPathParam(r, "verb")

    var req types.SubmitWorkflowRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeAPIError(w, http.StatusBadRequest, "invalid_json", err)
        return
    }

    result, err := h.Submission.Submit(r.Context(), submission.Request{
        Site:       model.SiteName(site),
        Verb:       verb,
        WorkflowID: req.WorkflowID,
        Values:     req.Values,
    })
    if err != nil {
        writeSubmissionError(w, err)
        return
    }

    writeJSON(w, http.StatusCreated, types.SubmitWorkflowResponseFromResult(result))
}
```

### Worker remains separate

```go
func workerMainLoop(ctx context.Context, cfg Config) error {
    scheduler := buildScheduler(cfg)
    for {
        _, err := scheduler.RunOnce(ctx)
        if err != nil {
            log.Error().Err(err).Msg("scheduler cycle failed")
        }
        sleep(cfg.PollInterval)
    }
}
```

### Workflow lifecycle diagram

```text
Client
  |
  | POST /api/v1/sites/js-demo/verbs/seed:submit
  v
HTTP API server
  |
  | resolve site + verb
  | run site migrations if needed
  | invoke one submit-verb JS function
  | create workflow + initial ops
  v
Engine DB
  |
  | later polled by
  v
Worker process
  |
  | lease ready op
  | run js/http runner
  | persist result
  | emit child ops
  v
Engine DB + site DB
```

## Testing Strategy

### Layered tests

The implementation should be tested in layers.

#### Layer 1: catalog tests

Verify:

- sites are listed
- verbs are listed
- one verb detail includes help/schema/source info

This can be done without running workers.

#### Layer 2: submission handler tests

Use `httptest.NewServer` or `httptest.NewRecorder`.

Verify:

- successful submit returns `201`
- unknown site returns `404`
- unknown verb returns `404`
- invalid JSON returns `400`
- duplicate workflow ID behavior is consistent and documented

#### Layer 3: read-side handler tests

Verify:

- engine status returns JSON equivalent of the current CLI status data
- workflow summary reflects newly submitted workflows
- workflow ops listing changes after workers process work

#### Layer 4: end-to-end local test with `js-demo`

This is the key proof.

Test flow:

1. start API server against temporary DB paths
2. `POST` `js-demo` `seed` submit request
3. confirm workflow exists through HTTP
4. run `scraper worker run --max-cycles ...`
5. confirm workflow moved to succeeded state

This proves:

- submit-verb discovery works
- submission writes durable work
- worker processes that work
- HTTP inspection endpoints observe the results

## Alternatives Considered

### Alternative 1: shell out to CLI commands from HTTP handlers

Rejected because:

- fragile parsing
- poor error modeling
- test-hostile
- duplicates logic

### Alternative 2: let the HTTP submit endpoint also run the scheduler until completion

Rejected as the default because:

- it fights the durable worker architecture
- request lifetime becomes coupled to workflow lifetime
- it makes rate limiting and background processing harder to reason about

This may still be useful later as an explicit debug mode, but not as the primary contract.

### Alternative 3: invent a new JSON workflow-spec endpoint instead of reusing JS submit verbs

Rejected because:

- it duplicates submission logic
- it creates two sources of truth
- it makes site behavior harder to keep consistent between CLI and HTTP

### Alternative 4: expose raw store tables directly over HTTP

Rejected because:

- it leaks schema details too early
- it couples clients to storage implementation
- it bypasses useful service-layer normalization

## Implementation Plan

The phased implementation plan below is the recommended build order. For an intern, the most important order is:

1. understand the current CLI submit flow
2. extract reusable services
3. add thin HTTP handlers
4. prove end-to-end behavior with `js-demo`
5. only then expand read APIs

### Phase 1: ticket and design

- create this ticket
- write the design guide
- write a chronological diary
- capture concrete code seams

### Phase 2: reusable service extraction

- extract submit logic from CLI-oriented code into a reusable service
- extract site/verb catalog helpers
- extract engine status helpers from [engine.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/engine.go)

This phase is about reducing future handler complexity.

### Phase 3: server bootstrap

- add `pkg/cmd/api.go`
- add `scraper api serve`
- add config flags:
  - `--address`
  - `--engine-db`
  - `--sites-dir`
  - `--read-timeout`
  - `--write-timeout`

### Phase 4: metadata endpoints

- `GET /healthz`
- `GET /api/v1/info`
- `GET /api/v1/sites`
- `GET /api/v1/sites/{site}/verbs`
- `GET /api/v1/sites/{site}/verbs/{verb}`

### Phase 5: submission endpoint

- `POST /api/v1/sites/{site}/verbs/{verb}:submit`
- JSON-to-Glazed value adapter
- typed success and error responses

### Phase 6: workflow and engine inspection endpoints

- `GET /api/v1/engine/status`
- `GET /api/v1/engine/migrations`
- `GET /api/v1/workflows/{workflowID}`
- `GET /api/v1/workflows/{workflowID}/ops`

### Phase 7: end-to-end testing

- `js-demo` API submission test
- worker-run processing test
- CLI/admin status cross-check against API output where practical

### Phase 8: documentation and help

- add Glazed help topic for the HTTP API
- add usage examples for `curl`
- document the submit-versus-worker split clearly

### Intern checklist

- Read [host.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/submitverbs/host.go) first.
- Read [runtime.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/submitverbs/runtime.go) second.
- Read [worker.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/worker.go) third.
- Read [engine.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/engine.go) fourth.
- Implement service helpers before HTTP handlers.
- Use `httptest` heavily.
- Start with `js-demo` because it avoids live HTTP dependencies.

## Open Questions

### Should the API support waiting for workflow completion?

Probably yes later, but the first slice should keep submission and execution separate. A follow-up could add:

- `?wait=true&timeout=5s`

implemented as polling on workflow state, not inline scheduling.

### Should workflow inspection include results and artifacts immediately?

Maybe, but there is value in starting with just:

- workflow summary
- op list

and adding richer result/artifact views later.

### Should we add auth now?

No. The first version should assume trusted local use and bind to loopback. Auth can be a later ticket once the transport contract is stable.

### How much of Glazed schema should the HTTP catalog expose?

At minimum:

- field names
- types
- required/default values
- help text

The catalog can grow richer later, but it needs enough metadata for a simple future web form.

## References

- [scraper-runtime-model.md](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc/topics/scraper-runtime-model.md)
- [scraper-new-developer-onboarding.md](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc/tutorials/scraper-new-developer-onboarding.md)
- [01-generic-go-scraper-engine-and-nereval-port-design-guide.md](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/design-doc/01-generic-go-scraper-engine-and-nereval-port-design-guide.md)
- [host.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/submitverbs/host.go)
- [runtime.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/submitverbs/runtime.go)
- [worker.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/worker.go)
- [engine.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/engine.go)
