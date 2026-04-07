# Changelog

## 2026-04-07

- Initial workspace created
- Reviewed the current engine view, API handler, server route, and frontend workflow API seams for artifacts and op results.
- Added a focused implementation guide for workflow artifact browsing, artifact downloading, and op-result retrieval.
- Added screen designs and a backend-first task breakdown.
- Added backend support for workflow-level artifact listing and real op-result retrieval.
- Added service tests and server endpoint tests for the new artifact/result routes.
- Updated the existing frontend op-result query to consume the new `{ result: ... }` response envelope.
- Added browser-oriented artifact list enrichment with preview hints plus workflow artifact filtering and pagination parameters.

## 2026-04-07

Step 1 (frontend): Wire getWorkflowArtifacts RTK Query endpoint in workflowApi.ts + WorkflowArtifactListResponse type in types.ts (commit 7834370)

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/api/workflowApi.ts — Added getWorkflowArtifacts endpoint

