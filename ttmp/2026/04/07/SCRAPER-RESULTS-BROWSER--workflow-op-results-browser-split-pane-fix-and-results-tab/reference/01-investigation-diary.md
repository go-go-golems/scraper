---
Title: "SCRAPER-RESULTS-BROWSER Implementation Diary"
Ticket: SCRAPER-RESULTS-BROWSER
Status: active
Topics:
    - implementation
    - results
    - split-pane
    - frontend
    - backend
DocType: reference
Intent: long-term
LastUpdated: 2026-04-07T23:45:00-04:00
---

# Implementation Diary — SCRAPER-RESULTS-BROWSER

## Session summary

Implemented all items in the ticket in one session. Three bugs/features addressed:

1. Split pane fix (Bug 1)
2. Op column → drawer wiring (Bug 2)
3. Results tab (new feature)

---

## Commits

### `348a7b0` — fix(ArtifactsPanel): change to row flex layout — split pane side-by-side

**What**: Changed `ArtifactsPanel` outer flex from `column` → `row`, giving the left panel `flex: 1, minWidth: 0` and the right preview `flex: '0 0 45%'`. Removed the fixed-height preview placeholder — empty state now uses `flex: 1` to fill the space.

**Why it was broken**: The design doc specified side-by-side but the implementation used `flexDirection: 'column'`, stacking everything vertically. The preview panel was below the table rather than beside it.

### `308d3cb` — fix(ArtifactTable): add onOpClick prop — Op cell opens OpDetailDrawer

**What**: Added `onOpClick?: (opId: string) => void` prop to `ArtifactTable`. When provided, the Op cell renders as a `<Link component="button">` that calls `onOpClick(artifact.opID)` with `e.stopPropagation()` to avoid selecting the row. `ArtifactsPanel` passes it through, `WorkflowDetailPage` wires it to `setSelectedOpId + setDrawerOpen(true)`.

**Why it was broken**: `ArtifactTable` rendered the Op cell as a plain styled `<Typography>` with no click handler, despite the design showing it as a navigable link. The table row click only selects the artifact, not the op.

### `4f2d694` — feat(engineview): add ListWorkflowResults + GET /workflows/{id}/results

**What**: Backend — `ListWorkflowResults` service method in `artifact_read_service.go` + `WorkflowResultsResponse` DTO + handler + route registration.

**Data flow**: Reads `results` table (via `openStore`) joined with `ops` table for kind/status metadata. `recordCount` is derived from `len(records_json)`. `artifactCount` is a correlated subquery against the `artifacts` table.

**Key SQL**:
```sql
SELECT results.op_id, ops.kind, ops.status,
       length(results.data_json),
       results.records_json,
       results.error_json,
       results.completed_at,
       (SELECT COUNT(1) FROM artifacts
        WHERE artifacts.op_id = results.op_id
          AND artifacts.workflow_id = results.workflow_id) AS artifact_count
FROM results
JOIN ops ON results.op_id = ops.id AND results.workflow_id = ops.workflow_id
WHERE <filters>
ORDER BY results.completed_at DESC, results.op_id
LIMIT ? OFFSET ?
```

### `ae967cd` — feat(workflowApi): add getWorkflowResults RTK Query endpoint + types

**What**: Added `WorkflowResultSummary` + `WorkflowResultsResponse` types to `types.ts`. Added `getWorkflowResults` query to `workflowApi.ts`, exported `useGetWorkflowResultsQuery`.

### `f51c23a` — feat(SCRAPER-RESULTS-BROWSER): add Results tab — ResultsPanel + results components

**What**: All frontend components for the Results tab:

| File | Component |
|---|---|
| `results/ResultFilterBar.tsx` | Op/Kind/Status/Search filter bar |
| `results/ActiveResultFilterChips.tsx` | Dismissible active filter chips |
| `results/ResultsTable.tsx` | Table: Op/Kind/Status/Records/Artifacts/Error/Actions |
| `results/ResultPreviewPanel.tsx` | Fetches full op result via `/workflows/{id}/ops/{opId}/result`, renders via `JsonViewer` |
| `results/ResultsPanel.tsx` | Two-panel split pane mirroring `ArtifactsPanel` |

**Key design decisions**:
- `ResultsTable` Op cell + "Open in Drawer" button both call `onOpClick` → opens `OpDetailDrawer`
- `ResultPreviewPanel` fetches the full `OpResult` body on selection, renders `body.Data` and `body.Records` via `JsonViewer`
- Tab order: Ops | Results | Artifacts (Results is the new middle tab)
- `workflows/ops/result` is used for the preview body — same endpoint the drawer uses

---

## What's still open

- [ ] Storybook stories for Results components + MSW handlers for `/workflows/{id}/results`
- [ ] Add MSW handlers for the results endpoint in `handlers.ts`
- [ ] Update `SCRAPER-ARTIFACT-BROWSER` tasks to mark remaining items (BinaryFallbackView, curl playbooks, docmgr doctor)
- [ ] Close `SCRAPER-ARTIFACT-BROWSER` ticket
