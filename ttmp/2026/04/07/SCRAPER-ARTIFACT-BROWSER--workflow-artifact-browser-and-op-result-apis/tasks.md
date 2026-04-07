# Tasks

## Review And Design

- [x] Create the `SCRAPER-ARTIFACT-BROWSER` ticket workspace.
- [x] Review current workflow, op, result, and artifact inspection seams across API, service, and UI layers.
- [x] Write the focused implementation guide for artifact browsing, download, and op-result retrieval.
- [x] Add screen designs for the workflow artifact browser.
- [x] Record the investigation diary.

## Scope Decisions

- [x] Keep this ticket limited to artifact browsing, downloading, and op-result retrieval.
- [x] Defer JS replay and debug execution to a separate ticket.
- [x] Treat workflow-level artifact browsing as the primary new backend surface.
- [x] Reuse existing durable engine state rather than adding a second artifact store.

## Phase 1: Backend Foundation

- [x] Add a workflow-level artifact listing service method.
- [x] Add `GET /api/v1/workflows/{workflowID}/artifacts`.
- [x] Add a real service-backed op-result retrieval method.
- [x] Add `GET /api/v1/workflows/{workflowID}/ops/{opID}/result`.
- [x] Define stable response DTOs for workflow artifact lists and op results.
- [x] Add tests for the new service methods.
- [x] Add handler/server route tests for the new endpoints.

## Phase 2: Artifact Browser Data Shape

- [x] Enrich artifact summaries for browser use:
  - [x] owning op id
  - [x] workflow id
  - [x] name
  - [x] kind
  - [x] content type
  - [x] size
  - [x] created at
  - [x] previewability hints
- [x] Decide whether filtering is server-side, client-side, or hybrid for v1.
- [x] Add pagination and filtering support if the browser needs it immediately.

## Phase 3: UI Integration Preparation

- [ ] Add frontend API queries for workflow-level artifacts.
- [ ] Keep artifact download by ID as the canonical body retrieval path.
- [ ] Prepare the workflow page for a browser tab or drawer panel.
- [ ] Add links from artifact rows back to the owning op.

## Phase 4: Validation And Docs

- [ ] Add curl playbooks for the new endpoints.
- [ ] Document how artifact browsing differs from future replay/debug work.
- [ ] Update the diary and changelog as implementation proceeds.
- [ ] Run `docmgr doctor --ticket SCRAPER-ARTIFACT-BROWSER --stale-after 30`.

## Follow-On Ticket

- [ ] Create the separate JS replay/debugger implementation ticket after the artifact browser backend is stable.
