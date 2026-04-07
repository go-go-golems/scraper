# Tasks

## Bug Fixes

- [ ] **Fix split pane** — `ArtifactsPanel` uses `flexDirection: column`, making table and preview panel stack vertically. Change to `flexDirection: row`, give table `flex: 1, minWidth: 0`, give preview `flex: '0 0 45%'`
- [ ] **Wire Op column → OpDetailDrawer** — `ArtifactTable` Op cell renders as styled text but has no click handler. Add `onOpClick` prop to `ArtifactTable`, pass through `ArtifactsPanel`, wire to `setSelectedOpId` + `setDrawerOpen` in `WorkflowDetailPage`

## Backend

- [ ] Add `ListWorkflowResults` service method in `artifact_read_service.go` — reads `op_results` via `openStore()`, with filtering by op, kind, status, search + pagination
- [ ] Add `WorkflowResultsResponse` type in `api/types/types.go`
- [ ] Add `WorkflowResults` handler in `engine.go`
- [ ] Register `GET /api/v1/workflows/{workflowID}/results` in `routes_engine.go`
- [ ] Add Go tests for `ListWorkflowResults`

## Frontend

- [ ] Add `getWorkflowResults` RTK Query endpoint to `workflowApi.ts`
- [ ] Add `WorkflowResultSummary` type to `types.ts`
- [ ] Create `web/src/components/results/ResultsPanel.tsx`
- [ ] Create `web/src/components/results/ResultsTable.tsx`
- [ ] Create `web/src/components/results/ResultPreviewPanel.tsx`
- [ ] Create `web/src/components/results/ResultFilterBar.tsx`
- [ ] Create `web/src/components/results/ActiveResultFilterChips.tsx`
- [ ] Add Results tab to `WorkflowDetailPage` tab strip
- [ ] Wire ResultsTable `onOpClick` → drawer (same pattern as ArtifactTable)
- [ ] Add "Open in Drawer" button to `ResultPreviewPanel`

## Storybook + MSW

- [ ] Add MSW handler for `GET /api/v1/workflows/:id/results` in `handlers.ts`
- [ ] Stories for `ResultsPanel`, `ResultsTable`, `ResultPreviewPanel`

## Docs

- [ ] Update implementation diary (`reference/01-investigation-diary.md`)
- [ ] Run `docmgr doctor --ticket SCRAPER-RESULTS-BROWSER --stale-after 30`
