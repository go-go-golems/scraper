# Changelog

## 2026-04-07

- Initial workspace created


## 2026-04-07

Phase 0 complete: ErrorBoundary (8059130), ToastProvider (c5ae0c4), BreadcrumbNav (3a67968)

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/App.tsx — Wrapped AppShell with AppErrorBoundary + ToastProvider
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/common/AppErrorBoundary.tsx — New — React error boundary with friendly fallback
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/common/ToastProvider.tsx — New — Global toast context + stacked snackbars
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/layout/BreadcrumbNav.tsx — New — Route-derived breadcrumb navigation
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/pages/SubmitWorkflowPage.tsx — Replaced local Snackbar with useToast()
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/pages/WorkflowDetailPage.tsx — Added toast feedback to retry/cancel handlers


## 2026-04-07

Phase 2 partial: RuntimeEventTable built + wired (edc8d94, de5aed4, 2945cab). Bug discovered: useRuntimeEventFeed infinite loop in Storybook. Analysis doc written. Handover for Redux rewrite.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/features/runtime-events/runtimeEventFeed.ts — Root cause of infinite loop — 4 useState + 4 useEffect chain. Needs Redux rewrite.

