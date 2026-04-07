# Tasks

## Phase 0-4: COMPLETE
## Phase 2A-2F: COMPLETE
## React Router navigation: COMPLETE

## Remaining fixes

- [x] Workflow detail navigation (done via React Router)
- [x] Wire CancelWorkflowButton into WorkflowHeader (done)
- [x] Wire artifact/script/log fetching in WorkflowDetailPage (done)
- [ ] Go embed wiring: serve built SPA from `scraper api serve`
- [ ] Queues page: show configured policy (MaxInFlight=1 default) clearly, not just "-"
- [ ] Submit verb log capture (2C.2 — submitverbs/runtime.go)

## Phase 3: Sites Page

- [ ] 3.1 Backend: Add site detail endpoint returning definition fields, scripts list, verbs list, queue policies
- [ ] 3.2 SiteCard widget: name, DB filename, script count, verb count, queue policy summary
- [ ] 3.3 SitesListPage (/sites): grid of SiteCards for all registered sites
- [ ] 3.4 SiteDetailPage (/sites/:name): tabbed view with:
  - Overview tab: site name, DB filename, registered queue policies
  - Verbs tab: list of verbs with fields/help (reuse VerbParameterForm in read-only mode)
  - Scripts tab: file tree + ScriptViewer (standalone script browser)
- [ ] 3.5 Add "Sites" tab to AppShell navigation
- [ ] 3.6 Stories for all new widgets
- [ ] 3.7 Link from Overview page and Op drawer (site chip -> /sites/:name)
