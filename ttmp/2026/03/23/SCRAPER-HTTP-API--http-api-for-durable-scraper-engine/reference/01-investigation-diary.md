---
Title: Investigation diary
Ticket: SCRAPER-HTTP-API
Status: active
Topics:
    - scraper
    - http-api
    - api
    - server
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/sites/submitverbs/host.go
      Note: Main evidence for how workflow submission is currently prepared and persisted
    - Path: pkg/sites/submitverbs/runtime.go
      Note: JS submit-verb runtime contract referenced throughout the design
    - Path: pkg/cmd/worker.go
      Note: Worker polling model used to justify keeping execution separate from API submission
ExternalSources: []
Summary: Chronological diary for the HTTP API ticket, recording the current architecture evidence, the reasoning behind the proposed API boundaries, validation steps, and delivery details.
LastUpdated: 2026-03-23T21:20:00-04:00
WhatFor: Records how the HTTP API design was researched and what concrete code paths informed the final proposal.
WhenToUse: Use when reviewing the history of the HTTP API ticket, understanding why certain boundaries were proposed, or retracing the commands used to validate and deliver the ticket.
---

# Investigation diary

## Goal

Create a dedicated ticket for an HTTP API to the durable scraper system, write a detailed intern-oriented analysis and implementation guide, keep a diary of the work, validate the ticket with `docmgr`, and upload the resulting bundle to reMarkable.

## Context

The scraper system has reached a point where it already supports durable workflow submission, long-running workers, JS-defined site submit verbs, rate-limited queues, and operator inspection commands. The main missing operator interface is an HTTP server.

The architecture is already close to ready for this:

- submit verbs provide a flexible workflow-submission boundary
- workers already poll the engine DB independently
- the engine already has inspection commands

That meant this ticket could focus on surfacing and organizing existing capabilities rather than inventing a new architecture from scratch.

## Prompt Context

The user request that started this ticket was:

> Ok, create a new ticket to create an HTTP API to the system.
>
> Then create a detailed reate a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
>   references.
>   It should be very clear and detailed. Store in the ticket and the nupload to remarkable.
>
> Keep a diary as you work.

## Chronology

### Step 1: Create the ticket scaffold

I created a new ticket rooted under `scraper/ttmp`:

- ticket ID: `SCRAPER-HTTP-API`
- title: `HTTP API for durable scraper engine`

Then I added:

- one design doc
- one reference diary

The workspace gave the usual ticket files:

- `index.md`
- `tasks.md`
- `changelog.md`
- the design doc
- this diary

### Step 2: Inspect the current submission and worker seams

I inspected the existing Go code before writing the proposal. The most important files were:

1. [host.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/submitverbs/host.go)
2. [runtime.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/submitverbs/runtime.go)
3. [worker.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/worker.go)
4. [site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site.go)
5. [engine.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/engine.go)

Commands used:

```bash
nl -ba scraper/pkg/sites/submitverbs/host.go | sed -n '1,260p'
nl -ba scraper/pkg/sites/submitverbs/runtime.go | sed -n '1,260p'
nl -ba scraper/pkg/cmd/worker.go | sed -n '1,260p'
nl -ba scraper/pkg/cmd/site.go | sed -n '1,320p'
nl -ba scraper/pkg/cmd/engine.go | sed -n '1,260p'
```

Main findings:

- `submitverbs.Host.Submit(...)` is already the right submission seam for HTTP.
- The worker loop is clearly separate and should stay that way.
- The engine inspection commands already define a useful first read-side API.
- The HTTP API should wrap Go services, not shell out to CLI commands.

### Step 3: Establish the architectural boundary

The main question was whether the new HTTP server should:

- submit work only
- submit and also run work inline
- or act as a general replacement for the CLI

I chose the first option for the initial design:

- HTTP server submits work and exposes read APIs
- worker process continues polling and executing ops

This matches the current durable design and avoids turning every request into a long-lived scheduler loop.

### Step 4: Write the design guide

The design guide was written as a new-intern document, not just a short RFC. It includes:

- current architecture and file references
- endpoint proposal
- request and response shapes
- service-layer extraction plan
- pseudocode
- ASCII diagrams
- phased implementation steps
- testing plan centered on `js-demo`

### Step 5: Expand the ticket bookkeeping

I replaced the default placeholder task list with a detailed phased plan and updated:

- `index.md`
- `tasks.md`
- `changelog.md`

so the ticket reads like a real work package rather than a scaffold.

### Step 6: Extract reusable HTTP-facing services

The first implementation slice focused on removing Cobra-specific assumptions from the future API path. Instead of letting handlers reach directly into site registration, JS scanning, DB opening, and engine inspection, I added three small service packages and one shared submit-verb loader.

The key implementation choice was to keep the existing submit-verb host as the write-path anchor. The new submission service does not invent a second workflow seeding mechanism. It resolves the same JS verb metadata the CLI uses, reconstructs Glazed parsed values from JSON input, and then calls `submitverbs.Host.Submit(...)`. In parallel, the catalog service exposes site and verb metadata, and the engine-view service exposes engine status plus workflow and op read models.

#### Prompt Context

**User prompt (verbatim):** "Create detailed tasks for building the HTTP API in the ticket, and then work on them one by one, commit at appropriate intervals, keep a diary."

**Assistant interpretation:** Expand the ticket into a concrete implementation plan, then begin actual code work in phase-sized commits while recording progress.

**Inferred user intent:** The user wants the HTTP API delivered incrementally, with a clear paper trail and enough structure that another developer can understand each implementation step.

**Commit (code):** `7fc2f7d1f2678a748fce871bf56d290ca7270722` — "Add scraper HTTP API bootstrap and catalog services"

#### What I did

- Added `pkg/sites/submitverbs/catalog.go` to centralize JS verb scanning and command lookup.
- Added `pkg/services/catalog/service.go` for site and verb metadata.
- Added `pkg/services/submission/service.go` and `pkg/services/submission/values.go` to turn HTTP JSON into Glazed parsed values and submit durable workflows through the existing host.
- Added `pkg/services/engineview/service.go` to expose engine status and workflow/op read models without shelling out to CLI commands.
- Added `pkg/cmd/api.go`, `pkg/api/server/server.go`, and the first handler/response packages.
- Registered `scraper api serve` in the root command and added initial HTTP tests plus `scraper api serve --help` coverage.
- Validated with:

```bash
go test ./pkg/api/server ./pkg/cmd -count=1
go test ./... -count=1
```

#### Why

- The HTTP server needed stable service seams before adding more endpoints.
- Reusing the submit-verb host preserves one workflow-submission implementation instead of forking CLI and API behavior.
- Reusing the generated Glazed schema keeps HTTP validation aligned with CLI flags and JS `__verb__` metadata.

#### What worked

- `submitverbs.Host.Submit(...)` was already the right write-path seam.
- The catalog path was straightforward once site verb scanning was factored into one helper.
- `net/http` with a standard `ServeMux` was enough for the first route set.

#### What didn't work

- N/A in this step. The first compile and test pass succeeded once the service split was in place.

#### What I learned

- The most fragile part of the HTTP API is not routing; it is faithfully reconstructing Glazed parsed values from JSON while preserving defaults and type validation.
- A direct workflow-op read query is cleaner than widening the generic store interface immediately.

#### What was tricky to build

- The tricky part was preserving section/default semantics from JS verbs. A naive `map[string]any` would have let HTTP diverge from CLI behavior. The fix was to use the generated command schema, initialize defaults through Glazed, and then apply request values against the real field definitions.

#### What warrants a second pair of eyes

- The JSON-to-values adapter in `pkg/services/submission/values.go`, especially if future verbs add more complex field types.
- The workflow-op read model in `pkg/services/engineview/service.go`, because it is currently a purpose-built SQL query rather than a shared store contract.

#### What should be done in the future

- Add targeted unit tests for the catalog service and JSON-to-values conversion edge cases.
- Decide whether the engine-view read helpers should eventually become part of the formal store interface.

#### Code review instructions

- Start in `pkg/cmd/api.go`, then `pkg/api/server/server.go`.
- Follow the call chain into `pkg/services/submission/service.go` and `pkg/sites/submitverbs/host.go`.
- Validate with:

```bash
go test ./pkg/api/server ./pkg/cmd -count=1
go test ./... -count=1
```

#### Technical details

- New packages:
  - `pkg/services/catalog`
  - `pkg/services/submission`
  - `pkg/services/engineview`
  - `pkg/api/handlers`
  - `pkg/api/server`
  - `pkg/api/types`
- New command:

```text
scraper api serve
```

### Step 7: Prove the API with an end-to-end `js-demo` workflow

The second implementation slice turned the server from “catalog plus plumbing” into a real durable API. I added an end-to-end test that submits a `js-demo` workflow through HTTP, runs the separate worker command against the same DB, and then reads the finished workflow and engine status back through the API.

I also added the first embedded help page for the new HTTP surface so operators can discover the server, understand the submit-versus-worker split, and copy working `curl` examples directly from CLI help.

#### Prompt Context

**User prompt (verbatim):** (see Step 6)

**Assistant interpretation:** Continue the HTTP API work until it includes the critical durable workflow proof and supporting operator docs.

**Inferred user intent:** The user wants the HTTP API to be more than route scaffolding; it should be exercised against the real durable runtime and documented for humans.

**Commit (code):** `449ca2bceb183dcf9c4b15ed9f823fd89fd9e82e` — "Add HTTP API workflow integration coverage"

#### What I did

- Added an end-to-end HTTP integration test in `pkg/api/server/server_test.go`.
- Submitted `js-demo` through:

```text
POST /api/v1/sites/js-demo/verbs/seed:submit
```

- Used `scraper worker run` against the same engine DB and sites dir inside the test.
- Verified:
  - workflow submission succeeded
  - worker processed all 5 durable ops
  - `GET /api/v1/engine/status` matched the executed state
  - `GET /api/v1/workflows/{id}` returned a succeeded workflow
  - `GET /api/v1/workflows/{id}/ops` returned the durable op list
- Added the embedded help page `pkg/doc/topics/scraper-http-api.md`.
- Manually checked:

```bash
go run ./cmd/scraper api serve --help
```

#### Why

- The architectural claim of this ticket is that the HTTP API should submit work while the worker executes it later.
- `js-demo` is the smallest built-in site that exercises the real submit-verb runtime, durable op graph, site DB, and worker loop without needing live HTTP fetches.

#### What worked

- The submit endpoint and worker process cooperated correctly through the shared engine DB.
- The engine status endpoint reported the same state the CLI `engine status` command reports.
- The help page immediately made the new API discoverable from the CLI.

#### What didn't work

- The first version of the new test created an import cycle because it lived in the `server` package and also imported `pkg/cmd`, which imports `pkg/api/server`. The exact failure was:

```text
package github.com/go-go-golems/scraper/pkg/api/server
	imports github.com/go-go-golems/scraper/pkg/cmd from server_test.go
	imports github.com/go-go-golems/scraper/pkg/api/server from api.go: import cycle not allowed in test
```

- I fixed that by moving the test to the external package `server_test`.

#### What I learned

- The cleanest durable proof is “HTTP submit + separate worker + HTTP inspect,” not trying to execute the scheduler inside the server test.
- External-package tests are the right choice when API server tests need to drive root-command behavior.

#### What was tricky to build

- The sharp edge was making the test prove the real architecture without cheating. It would have been easy to call lower-level scheduler helpers directly, but that would have skipped the exact boundary the user cares about. Running the worker command against the same SQLite DB kept the test honest.

#### What warrants a second pair of eyes

- The custom `:submit` path parsing in `pkg/api/server/server.go`, because it is currently implemented with a small manual parser on top of `ServeMux`.
- The decision to defer result/artifact listing from the workflow endpoints for now.

#### What should be done in the future

- Add request-scoped logging fields or request IDs if the server grows beyond local operator use.
- Consider replacing the manual `:submit` path parsing with a cleaner router or a dedicated pattern helper if more non-trivial routes appear.

#### Code review instructions

- Start with `pkg/api/server/server_test.go`.
- Then read `pkg/api/server/server.go` and `pkg/doc/topics/scraper-http-api.md`.
- Validate with:

```bash
go test ./pkg/api/server ./pkg/cmd -count=1
go test ./... -count=1
go run ./cmd/scraper api serve --help
```

#### Technical details

- Key API routes now covered:

```text
GET  /healthz
GET  /api/v1/sites
GET  /api/v1/sites/{site}/verbs/{verb}
POST /api/v1/sites/{site}/verbs/{verb}:submit
GET  /api/v1/engine/status
GET  /api/v1/workflows/{workflowID}
GET  /api/v1/workflows/{workflowID}/ops
```

## Quick Reference

### Recommended first command

```text
scraper api serve
```

### Recommended first endpoints

```text
GET  /healthz
GET  /api/v1/info
GET  /api/v1/sites
GET  /api/v1/sites/{site}/verbs
GET  /api/v1/sites/{site}/verbs/{verb}
POST /api/v1/sites/{site}/verbs/{verb}:submit
GET  /api/v1/workflows/{workflowID}
GET  /api/v1/workflows/{workflowID}/ops
GET  /api/v1/engine/status
GET  /api/v1/engine/migrations
```

### Core design rule

```text
HTTP API submits workflows.
Worker process executes workflows.
```

### Best code seam to reuse

- [host.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/submitverbs/host.go)

## Usage Examples

### Example submit request

```bash
curl -X POST http://127.0.0.1:8080/api/v1/sites/js-demo/verbs/seed:submit \
  -H 'Content-Type: application/json' \
  -d '{
    "workflowId": "demo-http-001",
    "values": {
      "count": 3,
      "multiplier": 4,
      "prefix": "http"
    }
  }'
```

### Example follow-up workflow inspection

```bash
curl http://127.0.0.1:8080/api/v1/workflows/demo-http-001
curl http://127.0.0.1:8080/api/v1/workflows/demo-http-001/ops
```

### Example worker execution in a separate process

```bash
scraper worker run --engine-db state/engine.db --sites-dir state/sites
```

## Validation

Ticket validation command:

```bash
docmgr doctor --ticket SCRAPER-HTTP-API --stale-after 30
```

Result:

- `docmgr doctor --ticket SCRAPER-HTTP-API --stale-after 30` passed after adding the missing `api`, `http-api`, and `server` topic vocabulary entries

## Delivery

Planned reMarkable destination:

- `/ai/2026/03/23/SCRAPER-HTTP-API`

Completed delivery:

- dry run:

```bash
remarquee upload bundle --dry-run \
  /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-HTTP-API--http-api-for-durable-scraper-engine \
  --name "SCRAPER-HTTP-API HTTP API architecture guide" \
  --remote-dir /ai/2026/03/23/SCRAPER-HTTP-API \
  --toc-depth 2 \
  --non-interactive
```

- upload:

```bash
remarquee upload bundle \
  /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-HTTP-API--http-api-for-durable-scraper-engine \
  --name "SCRAPER-HTTP-API HTTP API architecture guide" \
  --remote-dir /ai/2026/03/23/SCRAPER-HTTP-API \
  --toc-depth 2 \
  --non-interactive
```

- verification:

```bash
remarquee cloud ls /ai/2026/03/23/SCRAPER-HTTP-API --long --non-interactive
```

Verified listing:

- `SCRAPER-HTTP-API HTTP API architecture guide`

## Related

- [01-http-api-architecture-and-implementation-guide.md](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-HTTP-API--http-api-for-durable-scraper-engine/design-doc/01-http-api-architecture-and-implementation-guide.md)
- [scraper-runtime-model.md](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc/topics/scraper-runtime-model.md)
- [scraper-new-developer-onboarding.md](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc/tutorials/scraper-new-developer-onboarding.md)
