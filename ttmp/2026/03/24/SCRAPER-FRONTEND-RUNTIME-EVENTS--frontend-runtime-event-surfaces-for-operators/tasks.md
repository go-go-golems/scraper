# Tasks

## Phase 0: Research and Documentation

- [x] Create the ticket workspace
- [x] Inspect the existing runtime event backend and frontend seams
- [x] Write a detailed intern-facing design and implementation guide
- [x] Record the investigation diary

## Phase 1: Shared Frontend Runtime Event Infrastructure

- [x] Extract stream lifecycle and merge logic out of `WorkflowDetailPage`
- [x] Add a shared runtime event streaming hook
- [x] Add explicit connection-state handling for SSE consumers
- [x] Add shared client-side filtering and dedupe helpers
- [x] Expand runtime event rendering into reusable shared components

## Phase 2: Global Operator Event Console

- [x] Add a top-level `/events` page
- [x] Add navigation entry for the event console
- [x] Add source, severity, workflow, site, and worker filters
- [x] Add connection-state badge and last-event indicators
- [x] Add workflow click-through navigation

## Phase 3: Workflow and Op Context Views

- [x] Refactor `WorkflowDetailPage` to consume the shared runtime event feed abstractions
- [x] Add an op-scoped runtime event tab in `OpDetailDrawer`
- [x] Add richer payload rendering for retry, error, and artifact-summary payloads
- [ ] Decide default visibility rules for `DEBUG` events in workflow-local views

## Phase 4: Submit Flow Live Progress

- [ ] Add a post-submit live progress panel on `SubmitWorkflowPage`
- [ ] Stream runtime events for the newly submitted workflow
- [ ] Add a clear transition into the workflow detail page
- [ ] Decide whether recent submissions should link directly into live status panels

## Phase 5: Dashboard and Queue Widgets

- [ ] Add overview widgets for recent failures, retries, and active workers
- [ ] Add queue widgets for recent rate-limit and queue-local failure activity
- [ ] Keep snapshot polling and live event widgets complementary rather than competing
- [ ] Replace or augment placeholder queue throughput visuals with event-derived signals where appropriate

## Phase 6: Testing and UX Hardening

- [x] Add pure tests for event merge, sort, and filter helpers
- [ ] Add component tests for shared runtime event components
- [ ] Add stream-hook tests with mocked `EventSource`
- [ ] Add page-level tests for history-plus-stream merge behavior
- [ ] Review reconnect behavior and stale-connection UX
