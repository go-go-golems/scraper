# Tasks

## Review And Documentation

- [x] Create the `SCRAPER-OP-DEBUGGER` ticket workspace.
- [x] Review current workflow, op, artifact, script, and runtime-event inspection seams.
- [x] Write a detailed design and implementation guide for workflow artifact browsing and per-op JS replay debugging.
- [x] Record the investigation diary.
- [ ] Validate and publish the ticket bundle.

## Architecture Decisions To Confirm

- [x] Confirm that artifact browsing and JS replay debugging should be built on top of the existing durable engine state rather than a separate ad hoc debug store.
- [x] Confirm that the first-class replay unit is an executed op, not an entire workflow.
- [x] Confirm that a replay/debug path must reconstruct the JS runtime context from persisted workflow, op, dependency-result, and artifact state.
- [x] Confirm that the first debugger backend should be deterministic and read-only by default.
- [x] Confirm that operator browsing and engineer replay should share the same backend data model where possible.

## Phase 1: Foundation APIs

- [ ] Add a real backend endpoint for op result retrieval.
- [ ] Add a workflow-level artifact listing endpoint.
- [ ] Add artifact metadata enrichment suitable for browsing, filtering, and grouping.
- [ ] Add an endpoint that returns a consolidated op debug bundle.
- [ ] Define stable JSON DTOs for artifact browsing and debug bundles.

## Phase 2: Debug Bundle Generation

- [ ] Define a `DebugBundle` model containing:
  - [ ] workflow snapshot
  - [ ] op snapshot
  - [ ] lease snapshot when present
  - [ ] dependency result exports
  - [ ] artifact summaries and selected artifact bodies
  - [ ] script source reference and source text
  - [ ] runtime-event excerpt
- [ ] Decide which artifacts should be embedded inline versus linked by ID.
- [ ] Add deterministic serialization for bundle export and local inspection.
- [ ] Add tests for debug-bundle generation from completed workflows.

## Phase 3: Replay Execution

- [ ] Add a Go service that can execute one JS script from a debug bundle without mutating engine state.
- [ ] Provide a replay mode that mirrors the existing JS `ctx` shape as closely as possible.
- [ ] Decide whether replay should support:
  - [ ] `ctx.log`
  - [ ] `ctx.dep`
  - [ ] `ctx.writeRecord`
  - [ ] `ctx.writeArtifact`
  - [ ] `ctx.emit`
- [ ] Make replay read-only by default and capture emitted outputs as a result envelope instead of writing them durably.
- [ ] Add tests comparing live execution context and replay context shape.

## Phase 4: CLI Debugger

- [ ] Add a `scraper debug op bundle` command to export a debug bundle.
- [ ] Add a `scraper debug op replay` command to run one JS script from a bundle or workflow/op reference.
- [ ] Add options to override selected inputs for rapid iteration.
- [ ] Add a human-readable and JSON output mode for replay results.
- [ ] Add copy/paste playbooks for common debugging flows.

## Phase 5: Web UI Artifact Browser

- [ ] Add a workflow-level artifact browser view.
- [ ] Add filtering by op, kind, content type, and name.
- [ ] Add quick preview modes for text, JSON, HTML, and execution logs.
- [ ] Add links from artifacts back to the owning op.
- [ ] Add “open debug bundle / replay” affordances in op detail views.

## Phase 6: Web UI Replay Surfaces

- [ ] Decide whether replay runs should be CLI-first, backend-first, or UI-triggered in the initial release.
- [ ] If UI-triggered, add a replay panel with:
  - [ ] selected script path
  - [ ] editable op input
  - [ ] dependency preview
  - [ ] replay output/result panel
  - [ ] replay log output
- [ ] Make replay results clearly non-durable unless explicitly promoted to a workflow submission.

## Phase 7: Safety, Testing, and UX Hardening

- [ ] Make sure replay mode cannot accidentally mutate engine state by default.
- [ ] Add tests for bundle generation on missing dependencies or missing artifacts.
- [ ] Add tests for replay behavior on JS errors and rejected promises.
- [ ] Add tests for artifact-browser API pagination and filtering.
- [ ] Add operator/developer documentation for the debugger workflow.

## Follow-On Tickets To Consider

- [ ] Separate ticket for the backend artifact/debug-bundle APIs.
- [ ] Separate ticket for the CLI replay tooling.
- [ ] Separate ticket for the frontend artifact browser and replay panel.
- [ ] Separate ticket for durable replay-run history if that becomes desirable.

## Validation And Publishing

- [ ] Run `docmgr doctor --ticket SCRAPER-OP-DEBUGGER --stale-after 30`.
- [ ] Upload the bundled ticket docs to reMarkable.
- [ ] Commit the doc slice.
