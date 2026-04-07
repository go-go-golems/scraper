---
Title: Phase 2 Implementation Diary
Ticket: SCRAPER-DASHBOARD
Status: active
Topics:
    - dashboard
    - react
    - scraper
    - frontend
    - material-ui
    - redux
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Step-by-step narrative of Phase 2 dashboard implementation"
LastUpdated: 2026-03-23T22:43:18.261981594-04:00
WhatFor: "Track what was done, what worked, what was tricky"
WhenToUse: "During implementation and for future review"
---

# Phase 2 Implementation Diary

Chronological record of implementing the 5 new dashboard features: artifact browser, rate limiter detail, op execution logs, retry/cancel, script source viewer.

---

## Entry 1: Starting Phase 2A — Artifact Browser Backend

**Date:** 2026-03-23
**Task:** 2A.1 — Backend: GetArtifacts store method + API endpoints

**Plan:** The `artifacts` table already stores blobs with metadata. `loadArtifacts()` exists as a private function in sqlite/store.go. I need to:
1. Add a public `GetArtifacts` method to the engineview service (not store interface — keep it simple like ListQueues)
2. Add two API endpoints: list artifacts (metadata only) and download single artifact (raw body)

**Starting now.**

**Result:** Completed all backend for all 5 features in one pass. Added 7 new endpoints in a single commit (`f40d3e2`).

**What worked well:**
- Kept all new methods in the engineview service (direct SQL) rather than extending the Store interface. This avoids changing the interface that the scheduler depends on.
- Used `length(body)` in the SQL to get artifact size without loading the blob into memory for the list endpoint.
- For retry, resetting status to `ready` (not `pending`) ensures the scheduler picks it up immediately on next poll — no need to go through dependency resolution.
- For cancel, used a transaction to atomically cancel ops + delete leases + update workflow.
- Script path traversal prevention: `filepath.Clean` + check for `..` before any FS access.

**What was tricky:**
- The `:cancel` and `:retry` URL suffixes conflict with Go's `{workflowID}` pattern matching (the colon gets consumed). Had to use a custom POST handler that parses the path manually, same pattern as the existing `:submit` handler.

---

## Entry 2: All Backend Complete — Starting Frontend

**Date:** 2026-03-23
**Task:** 2A.2-2E.4 — All frontend work

**Backend endpoints now available:**
- `GET /api/v1/workflows/{wfID}/ops/{opID}/artifacts` — artifact list
- `GET /api/v1/artifacts/{artifactID}` — artifact download
- `POST /api/v1/workflows/{wfID}/ops/{opID}:retry` — retry failed op
- `POST /api/v1/workflows/{wfID}:cancel` — cancel workflow
- `GET /api/v1/sites/{site}/scripts` — list scripts
- `GET /api/v1/sites/{site}/scripts/{path}` — read script source

**Plan:** Build all frontend components in parallel using agents:
- Agent 1: Artifact browser widgets (ArtifactList, ArtifactPreview) + RTK Query
- Agent 2: Rate limiter detail (TokenBucketGauge, QueueDetailPanel, QueuePolicySummary)
- Agent 3: Retry/Cancel (ConfirmDialog, RetryOpButton, CancelWorkflowButton) + Script viewer (ScriptViewer, ScriptTab)
- Then: Refactor OpDetailDrawer into tabbed layout and wire everything together

**Result:** Both agents completed successfully. 24 new files, all compiling cleanly. Committed as `e1a768b`.

---

## Entry 3: Tabbed Drawer Refactor

**Date:** 2026-03-23
**Task:** 2F — Refactor OpDetailDrawer into tabbed layout

Rewrote the OpDetailDrawer from a flat scroll layout to a tabbed layout with 6 tabs: Input, Deps, Result, Artifacts, Script, Logs. The Result tab now contains error, retry, and lease info (previously separate sections at the bottom). The Artifacts tab has list + inline preview. The Logs tab consumes execution-log kind artifacts. The Script tab uses ScriptTab.

Added RetryOpButton to the drawer header for failed ops. Widened drawer from 450px to 500px.

Updated all stories to use the new props (artifacts, artifactBodies, scriptSource). Added new stories: WithArtifacts, WithLogs.

Committed as `b73fb63`.

---

## Entry 4: Log Capture Backend

**Date:** 2026-03-23
**Task:** 2C.1 — Modify executor to capture ctx.log

Added `LogEntry` struct and `logEntries` slice to `executionState`. The `ctx.log()` function now appends to the buffer in addition to writing to zerolog. In `buildOpResult`, if there are log entries, they're serialized as JSON and written as an artifact with `kind: "execution-log"`.

This is the cleanest approach because:
- No new database table needed
- No new API endpoint needed (uses existing artifact endpoints)
- The dashboard's Logs tab already parses execution-log artifacts
- Logs are still visible in CLI stderr via zerolog

All tests pass. Committed as `05d9b02`.

---

## Summary

**4 commits in Phase 2:**
1. `f40d3e2` — Backend: 7 new API endpoints (artifacts, retry/cancel, scripts)
2. `e1a768b` — Frontend: 14 new widgets + 35 stories
3. `b73fb63` — Integration: Tabbed OpDetailDrawer wiring all features
4. `05d9b02` — Backend: ctx.log() capture as execution-log artifact

**What worked well:**
- Building all backend endpoints in one pass before touching frontend
- Parallel agent execution for frontend widgets (2 agents, ~2.5 min each)
- Storing logs as artifacts — zero new infrastructure, maximum reuse
- The tabbed drawer design keeps the drawer navigable even with 6 content areas

**What was tricky:**
- Go's ServeMux `:suffix` routing for `:retry` and `:cancel` — needed custom path parsing
- The `.gitignore` `logs` pattern matched our `components/logs/` directory — needed `git add -f`
- Artifact preview needs the body fetched separately from the list — the drawer needs both `artifacts` (metadata) and `artifactBodies` (content by ID) props

---

## Entry 5: Wiring + React Router

**Date:** 2026-03-23
**Tasks:** Wire all Phase 2 features into pages + proper URL navigation

**Problem:** All Phase 2 widgets existed as standalone components with stories but weren't connected to the live pages. The OpDetailDrawer wasn't receiving artifacts, scripts, or log data. The QueueMonitorPage had no expandable detail panels. WorkflowsPage clicking a row did nothing. Navigation was state-based (`useState` in App.tsx), so no URLs, no browser back/forward, no shareable links.

**What was done:**

1. **WorkflowDetailPage** — Now fetches artifacts (`useGetOpArtifactsQuery`), script source (`useGetScriptQuery`), and passes retry/cancel mutations to the drawer. Artifact bodies fetched on demand via `fetch()` when switching to Artifacts tab. CancelWorkflowButton added next to WorkflowHeader.

2. **QueueMonitorPage** — Added expandable QueueDetailPanel per queue with Collapse animation.

3. **React Router** — Replaced `useState` navigation with `BrowserRouter` + `Routes`. Routes: `/`, `/workflows`, `/workflows/:id`, `/queues`, `/submit`. AppShell tabs derive active state from `useLocation().pathname`. Removed `currentTab`/`setTab` from uiSlice (router owns navigation now). Updated AppShell stories with `MemoryRouter`.

**Commits:** `013a7bb` (wiring), `0a9646a` (React Router)

**What worked well:**
- React Router v6 `useParams` is clean — WorkflowDetailPage just reads `workflowId` from the URL
- `encodeURIComponent` on workflow IDs handles special characters in IDs
- The tab highlighting via `pathname.startsWith('/workflows')` correctly highlights the tab for both list and detail views

**What I'd do differently:**
- Should have used React Router from the start instead of the `useState` approach. The refactor was clean but it was unnecessary churn.

---

## Entry 6: Queue Policy Display + Sites Page

**Date:** 2026-03-23

**Problem 1: Queue policy not visible.** Discovered that no built-in site sets `QueuePolicies` in its `registry.Definition`. Every queue uses `DefaultQueuePolicy()` (MaxInFlight=1, no rate limit). The QueueStatusTable showed `-` for Rate/Burst/Tokens which looked broken rather than intentional.

**Fix:** Two-part:
- Backend: `EngineHandler.Queues` now enriches queue data with configured policy from the registry via `catalog.GetAllQueuePolicies()`. This merges the static config (from registry) with dynamic runtime data (from engine DB).
- Frontend: QueueStatusTable shows "1 (default)" in grey for MaxInFlight=1, and "none" instead of "-" for empty rate/burst/tokens. Clear that default is intentional.

**Problem 2: No way to browse sites, scripts, or verbs.** Script viewer was buried in the op drawer. Site info was invisible. Queue policies were defined in Go code with no UI exposure.

**Solution: Sites page (/sites, /sites/:name).** Three components:
- `SiteCard` — clickable card with name, DB file, verb/script counts, policy summary
- `SiteScriptBrowser` — split-pane with file list + ScriptViewer, fetches on demand via catalogApi.getScript
- `SiteVerbList` — accordion with verb name, description, field table

SiteDetailPage has three tabs: Overview (policies + stats), Verbs, Scripts. Added "Sites" tab to AppShell navigation.

**Key backend addition:** New `catalog.GetSiteDetail()` returns `SiteDetail` with verb count, script list, and queue policies. New `catalog.GetAllQueuePolicies()` returns all registered policies across all sites. New `GET /api/v1/sites/{site}/detail` endpoint.

**Commits:** `915107b` (backend), `efb0bdf` (frontend)

**What was tricky:** The `engineview.Service` only has the engine DB path — it doesn't have the registry. Rather than refactoring the service, I passed the `catalogService` to the `EngineHandler` and did the merge there. Handler-level composition is simpler than adding a cross-cutting concern to the service layer.

---

## Entry 7: Null Safety Fixes

**Date:** 2026-03-24
**Task:** Runtime errors from Go's nil-slice JSON serialization

Three components crashed at runtime because Go serializes `nil` slices as `null` in JSON, not `[]`. RTK Query returns `undefined` before the first fetch. Both cases need `?? []` guards.

**Fixes:**
- `OpDetailDrawer`: `artifacts` prop guarded with `?? []` before `.find()/.filter()`
- `SiteCard`: `site.queuePolicies` guarded with `?? []` before `.length` and `.map()`
- `SiteDetailPage`: same `?? []` pattern for `detail.queuePolicies`

**Lesson:** Every array from the Go API should be treated as nullable on the frontend. Could add a global RTK Query transform, but `?? []` at the usage site is simpler and explicit.

**Commits:** `abdbc3f`, `8911616`
