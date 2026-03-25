# Changelog

## 2026-03-24

- Initial workspace created
- Re-ran `docmgr doctor` after the upstream fix, restored missing `react` and `events` topic vocabulary, and confirmed the ticket now passes validation cleanly
- Investigated the current frontend runtime event path across:
  - `web/src/pages/WorkflowDetailPage.tsx`
  - `web/src/api/runtimeEventsApi.ts`
  - `web/src/components/workflows/RuntimeEventList.tsx`
  - `web/src/components/workflows/OpDetailDrawer.tsx`
  - `web/src/pages/SubmitWorkflowPage.tsx`
  - `web/src/pages/EngineOverviewPage.tsx`
  - `web/src/pages/QueueMonitorPage.tsx`
- Cross-checked the frontend assumptions against the backend runtime event contract in:
  - `pkg/api/server/server.go`
  - `pkg/api/handlers/runtime_events.go`
  - `pkg/runtimeevents/hub.go`
  - `proto/scraper/runtime/v1/events.proto`
- Wrote a detailed design and implementation guide for a new engineer that covers:
  - current-state analysis,
  - gap analysis,
  - proposed frontend architecture,
  - API references,
  - pseudocode,
  - phased file-by-file implementation plan,
  - testing strategy,
  - risks and alternatives
