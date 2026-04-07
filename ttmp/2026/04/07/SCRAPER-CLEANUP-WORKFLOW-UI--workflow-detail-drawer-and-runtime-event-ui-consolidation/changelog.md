# Changelog

## 2026-04-07

- Initial workspace created
- Added the workflow UI cleanup design and task plan, including removal of `RuntimeEventList`.

## 2026-04-07

Completed all tasks: decomposed OpDetailDrawer into 7 tab subcomponents + helpers (commit a607b83), removed unused RuntimeEventList.tsx (commit a0bfa4c), validated with tsc --noEmit and docmgr doctor.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/workflows/OpDetailDrawer.tsx — Refactored from monolith to shell + subcomponent imports (a607b83)
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/workflows/RuntimeEventList.tsx — Deleted as unused — RuntimeEventTable is the sole renderer (a0bfa4c)
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/workflows/op-detail/helpers.tsx — New shared helper module with KindIcon and connectionColor (a607b83)

