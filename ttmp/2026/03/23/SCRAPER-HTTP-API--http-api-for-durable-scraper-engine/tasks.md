# Tasks

## Analysis and planning

- [x] Create the dedicated `SCRAPER-HTTP-API` ticket workspace
- [x] Inspect the current CLI submit path, worker path, and engine inspection path
- [x] Identify the best service seams for HTTP submission and read APIs
- [x] Write a detailed analysis, design, and implementation guide for a new intern
- [x] Record a chronological investigation diary
- [x] Upload the ticket bundle to reMarkable

## Detailed implementation plan

### Phase 1. Service extraction

- [x] Extract workflow-submission logic into a reusable service layer that is not Cobra-specific
  - [x] Add a submission service package under `pkg/services/submission`
  - [x] Keep [host.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/submitverbs/host.go) as the initial implementation anchor
  - [x] Expose a typed request object with `site`, `verb`, `workflowID`, `engineDB`, `sitesDir`, and arbitrary input values
  - [x] Expose a typed response object that mirrors the existing submit result envelope
- [x] Add a small service for listing sites and JS submit verbs
  - [x] Add a catalog service package under `pkg/services/catalog`
  - [x] Scan `VerbsFS`/`VerbsRoot` through `jsverbs.ScanFS`
  - [x] Return stable metadata for site and verb listing endpoints
- [x] Add a small service for engine and workflow read models
  - [x] Add an engine-view service package under `pkg/services/engineview`
  - [x] Reuse `sqlitestore.Inspect` for engine-level summary endpoints
  - [x] Add workflow and op read helpers for API use
- [x] Keep the new services usable from both CLI and HTTP handlers

### Phase 2. HTTP server bootstrap

- [x] Add a new `scraper api` command
- [x] Add `scraper api serve`
- [x] Add flags for:
  - [x] `--address`
  - [x] `--engine-db`
  - [x] `--sites-dir`
  - [x] `--read-timeout`
  - [x] `--write-timeout`
- [x] Add an `api` package tree:
  - [x] `pkg/api/types`
  - [x] `pkg/api/server`
  - [x] `pkg/api/handlers`
- [x] Default bind address to loopback only
- [x] Use `net/http` for the first version unless a stronger routing need appears

### Phase 3. Catalog endpoints

- [x] Implement `GET /healthz`
- [x] Implement `GET /api/v1/info`
- [x] Implement `GET /api/v1/sites`
- [x] Implement `GET /api/v1/sites/{site}`
- [x] Implement `GET /api/v1/sites/{site}/verbs`
- [x] Implement `GET /api/v1/sites/{site}/verbs/{verb}`
- [x] Expose enough JS/Glazed metadata for future form generation:
  - [x] command name
  - [x] full path
  - [x] help text
  - [x] source file
  - [x] fields / sections / defaults
  - [x] output mode if useful
  - [x] parent path segments if useful

### Phase 4. Submission endpoint

- [x] Implement `POST /api/v1/sites/{site}/verbs/{verb}:submit`
- [x] Define typed request and response structs
- [x] Add JSON-to-Glazed parsed-values conversion
  - [x] Support strings, booleans, integers, floats, and simple string lists
  - [x] Reject clearly incompatible payloads with `400`
- [x] Return structured JSON errors instead of CLI text
- [x] Preserve explicit workflow ID support
- [x] Preserve useful submit response data:
  - [x] workflow summary
  - [x] target op ID
  - [x] site DB path
  - [x] submitted op count
  - [x] verb result payload

### Phase 5. Inspection endpoints

- [x] Implement `GET /api/v1/engine/status`
- [x] Implement `GET /api/v1/engine/migrations`
- [x] Implement `GET /api/v1/workflows/{workflowID}`
- [x] Implement `GET /api/v1/workflows/{workflowID}/ops`
- [x] Decide whether results and artifacts should be included now or later
- [x] Add store helpers for workflow summaries if existing read helpers are not sufficient
  - [x] Add a direct op-list query for one workflow
  - [x] Include op dependencies and retry state in the API response if cheap to expose
  - [ ] Keep result and artifact listing deferred unless the implementation is trivial

### Phase 6. Errors, logging, and observability

- [x] Standardize API error envelopes
- [x] Add request logging
- [ ] Add handler-level context fields for:
  - [ ] site
  - [ ] verb
  - [ ] workflow ID
- [x] Decide whether to include request IDs in the first slice

### Phase 7. Testing

- [ ] Add unit tests for catalog services
- [ ] Add unit tests for JSON-to-values conversion
- [x] Add handler tests with `httptest`
- [x] Add end-to-end tests with `js-demo`
- [x] Prove:
  - [x] HTTP submit creates durable workflow state
  - [x] separate `worker run` processes the submitted workflow
  - [x] workflow inspection endpoints reflect the resulting state
- [x] Cross-check API engine status against existing `engine status` semantics where practical
  - [x] Add a root command test for `scraper api serve --help`
  - [x] Add a submission test for `POST /api/v1/sites/js-demo/verbs/seed:submit`
  - [x] Add a worker-follow-up test that uses the same engine DB after submission
  - [x] Add a workflow-ops endpoint test after worker execution

### Phase 8. Documentation and help

- [x] Add embedded help pages under [pkg/doc](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc) for the HTTP API
- [x] Add `curl` examples for submission and inspection
- [x] Add onboarding notes that explain:
  - [x] submit verb runtime versus op runtime
  - [x] API server versus worker process
  - [x] why the API does not execute workflows inline by default
- [ ] Update the main design docs if the final implementation shape diverges from this ticket
