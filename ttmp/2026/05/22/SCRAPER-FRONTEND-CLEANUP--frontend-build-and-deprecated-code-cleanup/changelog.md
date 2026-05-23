# Changelog

## 2026-05-22

- Initial workspace created


## 2026-05-22

Created cleanup ticket, captured baseline frontend build failures, and wrote the cleanup guide.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/05/22/SCRAPER-FRONTEND-CLEANUP--frontend-build-and-deprecated-code-cleanup/analysis/01-frontend-cleanup-guide.md — Cleanup guide


## 2026-05-22

Phase 1: removed stale unused frontend symbols, fixed story API drift, fixed CodeViewPanel ToggleButton typing, and confirmed pnpm build passes.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/common/AlertBanner.stories.tsx — Removed unsupported onDismiss story prop
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/common/CodeViewPanel.tsx — ToggleButton value/onChange cleanup
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/stories/msw/handlers.ts — Fixed stale story fixture imports and unused handler params

