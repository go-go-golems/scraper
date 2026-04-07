---
Title: "Workflow Op Results Browser — Split Pane Fix + Results Tab"
Ticket: SCRAPER-RESULTS-BROWSER
Status: active
Topics:
    - frontend
    - results
    - artifacts
    - workflows
    - backend
DocType: ticket
Intent: long-term
Owners: []
RelatedFiles:
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/artifacts/ArtifactsPanel.tsx"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/artifacts/ArtifactTable.tsx"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/pages/WorkflowDetailPage.tsx"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/services/engineview/artifact_read_service.go"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/api/handlers/engine.go"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/api/server/routes_engine.go"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/api/workflowApi.ts"
Summary: "Fix the Artifacts tab split pane layout, wire the Op column to open OpDetailDrawer, and add a new Results tab for browsing parsed op data (JS extract results, HTTP response data) across the workflow."
LastUpdated: 2026-04-07T23:00:00-04:00
WhatFor: "Enable efficient cross-op result browsing, especially for JS extract ops that have no artifacts but produce important parsed data."
WhenToUse: "Use when implementing or iterating on the workflow detail Results tab and split pane fixes."
---

# SCRAPER-RESULTS-BROWSER: Workflow Op Results Browser

## Goal

Three things:

1. **Fix the Artifacts tab split pane** — the table and preview panel are currently stacked vertically instead of side-by-side
2. **Wire Op column → OpDetailDrawer** — clicking an op in the artifact table opens the drawer (same as the Ops tab does)
3. **Add a Results tab** — browse parsed result data across all ops (JS extract output, HTTP response data), viewable in a preview panel

The Results tab serves ops that have `recordCount > 0` but `artifactCount = 0` (e.g. JS extract ops that return parsed JSON without writing any files).

---

## Bug 1: Split pane is stacked, not side-by-side

### Current state

`ArtifactsPanel` uses a column flex container, so the table and preview panel stack vertically:

```
ArtifactsPanel
└── <Box sx={{ display: 'flex', flexDirection: 'column' }}>   ← STACKED
    ├── FilterBar
    ├── ActiveFilterChips
    ├── ArtifactTable               ← top half
    └── ArtifactPreviewPanel        ← bottom half (below, not beside)
```

The preview panel is also inside a tall `height: 500` box with no split.

### Expected (from design doc)

```
ArtifactsPanel
└── <Box sx={{ display: 'flex', flexDirection: 'row' }}>     ← SIDE-BY-SIDE
    ├── ArtifactTable               ← left: flex: 1, minWidth: 0
    └── ArtifactPreviewPanel        ← right: fixed width, collapsible
```

### Fix

- `ArtifactsPanel`: change `flexDirection: 'column'` → `flexDirection: 'row'`; wrap table in `Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}`; give preview panel a fixed `width` (or `flex: '0 0 45%'`); use `height: '100%'` or `minHeight` on the outer container

---

## Bug 2: Op column in artifact table does not open the drawer

### Current state

`ArtifactTable` renders the Op cell as a styled `<Typography>`:

```tsx
<TableCell>
  <Tooltip title={artifact.opID}>
    <Typography variant="caption" sx={{ fontFamily: 'monospace', color: 'primary.main' }}>
      {opNameMap[artifact.opID] ?? artifact.opID}
    </Typography>
  </Tooltip>
</TableCell>
```

Clicking it does nothing. The Ops tab opens `OpDetailDrawer` via `handleSelectOp` in `OpTable`, but `ArtifactTable` has no equivalent callback.

### Expected

Clicking the Op cell in the artifact table opens `OpDetailDrawer` for that op — identical to what the Ops tab does.

### Fix

- `ArtifactTable`: add `onOpClick?: (opId: string) => void` prop; wrap the Op `<TableCell>` in a `Link` or `ButtonBase` that calls it
- `ArtifactsPanel`: pass `onOpClick` up to `WorkflowDetailPage`
- `WorkflowDetailPage`: wire `onOpClick` → `setSelectedOpId(id)` + `setDrawerOpen(true)`

### UX note

The design doc originally called for a "→ Op detail" link in the preview panel header. That was removed since the Op column in the table already handles the navigation. Make sure we don't add back a redundant link.

---

## Feature: Results Tab

### Why

A JS extract op returns parsed data (stories, items, records) but has no artifact. The Artifacts tab correctly shows nothing for that op — but users need to see those results.

```
┌─ Ops ─────────────────────────────────────────────────────────────────────────┐
│ Op                    │ Kind       │ Status      │ Records  │ Artifacts     │
├───────────────────────┼────────────┼─────────────┼──────────┼──────────────┤
│ frontpage-fetch       │ http       │ succeeded   │ 0       │ 1 (html)      │ ← has artifact
│ extract               │ js         │ succeeded   │ 30      │ 0            │ ← NO artifact
│ item-detail-fetch     │ http       │ succeeded   │ 0       │ 2 (html)      │ ← has artifact
│ item-detail-extract  │ js         │ succeeded   │ 10      │ 0            │ ← NO artifact
└───────────────────────┴────────────┴─────────────┴──────────┴──────────────┘
                                  ↑ users need to see these records
```

### Design: Results tab layout

```
╔═══════════════════════════════════════════════════════════════════════════════════════╗
║ Workflow: hackernews-extract-frontpage-1775586649974859668                              ║
║ Status: succeeded    Site: hackernews    Ops: 14/14     Last run: 2 hours ago          ║
╠═══════════════════════════════════════════════════════════════════════════════════════╣
║  Overview  │   Ops   │  Runtime  │  Results  │  Artifacts  │  Site Info              ║
╠═══════════════════════════════════════════════════════════════════════════════════════╣
║                                                                                       ║
║  [▤ Preview panel]   Op: [All Ops                    ▼]   Kind: [All           ▼]   ║
║  Status: [All               ▼]   search: [search by name...              ] 🔍      ║
║  ──────────────────────────────────────────────────────────────────────────────────── ║
║  Active filters:  [js ×]  [succeeded ×]                                           ║
║  Clear all                                                                          ║
║                                                                                       ║
║  ┌────────────────────────────────────────────────────────────────────────────────┐   ║
║  │ Op                 │ Kind    │ Status      │ Records │ Errors │ Artifacts    │   ║
║  ├───────────────────┼─────────┼─────────────┼─────────┼────────┼──────────────┤   ║
║  │ ▸ frontpage-fetch  │ http    │ succeeded   │   —     │   0    │  1           │   ║
║  │ ◆ extract         │ js      │ succeeded   │  30     │   0    │  0  ← key   │   ║
║  │ ▸ item-detail-fetch│ http    │ succeeded   │   —     │   0    │  2           │   ║
║  │ ◆ item-extract    │ js      │ succeeded   │  10     │   0    │  0  ← key   │   ║
║  │ ▸ seed-script      │ js      │ failed      │   —     │   1    │  0           │   ║
║  └────────────────────────────────────────────────────────────────────────────────┘   ║
║                                                                                       ║
║  Showing 1–10 of 14 results                                       [←]  Page 1  [→]     ║
║                                                                                       ║
╠═══════════════════════════════════════════════════════════════════════════════════════╣
║                                                                                       ║
║  Op: extract          Kind: js         Status: succeeded                             ║
║  Records: 30    Artifacts: 0    Completed: 2h ago                                     ║
║  ──────────────────────────────────────────────────────────────────────────────────── ║
║                                                                                       ║
║  ┌────────────────────────────────────────────────────────────────────────────────┐   ║
║  │ {                                                                               │   ║
║  │   "stories": [                                                                  │   ║
║  │     { "id": 12345, "url": "https://...", "title": "Show HN: A cool project" },  │   ║
║  │     { "id": 12346, "url": "https://...", "title": "Another story" },           │   ║
║  │     ...                                                                         │   ║
║  │   ],                                                                            │   ║
║  │   "nextPage": "/news?p=2"                                                        │   ║
║  │ }                                                                               │   ║
║  └────────────────────────────────────────────────────────────────────────────────┘   ║
║                                                                                       ║
║                                [OPEN IN DRAWER →]                                     ║
║                                                                                       ║
╚═══════════════════════════════════════════════════════════════════════════════════════╝
```

### Key differences from Artifacts tab

| Aspect | Artifacts tab | Results tab |
|---|---|---|
| Source data | `artifacts` SQLite table | `op_results` SQLite table (same store) |
| Content | Files, HTML, JSON, images | Parsed JSON records, data payloads |
| Preview | `ArtifactPreview` (HTML/JSON/text) | `OpResultPreview` (JSON/text, same renderer) |
| Filter | Op, Kind, Content-Type, search | Op, Kind, Status, search |
| Preview action | Open Raw, Download | Open Raw, **Open in drawer** |
| Op column | Opens `OpDetailDrawer` | Opens `OpDetailDrawer` (same) |

### Data model

Each result summary row comes from the `op_results` table:

```sql
results(op_id, workflow_id, data_json, records_json, emitted_json, error_json, completed_at)
```

The new backend endpoint reads this with pagination + filtering:

```json
{
  "workflowID": "wf-123",
  "total": 14,
  "results": [
    {
      "opID": "wf-123:extract",
      "kind": "js",
      "status": "succeeded",
      "recordCount": 30,
      "artifactCount": 0,
      "dataSize": 2048,
      "error": null,
      "completedAt": "2026-04-07T14:32:05Z"
    }
  ]
}
```

### Empty state

```
╔════════════════════════════════════════════════════════════════════════════╗
║                         No results found                                  ║
║                                                                             ║
║   This workflow has no op results yet,                                     ║
║   or the current filters match nothing.                                     ║
║                                                                             ║
║                          [ Clear all filters ]                             ║
║                                                                             ║
╚════════════════════════════════════════════════════════════════════════════╝
```

---

## Bridge interactions

```
┌── Ops tab ──────────────────────────────────────────────────────────────────┐
│ OpTable → click op → OpDetailDrawer opens                                   │
│            ├── Result tab → shows single op's result (existing)             │
│            └── Artifacts tab → shows artifacts from this op (existing)     │
└───────────────────────────────────────────────────────────────────────────────┘

┌── Artifacts tab ────────────────────────────────────────────────────────────┐
│ ArtifactTable → click Op cell → OpDetailDrawer opens (new, this ticket)    │
│ ArtifactPreviewPanel → (no drawer navigation needed)                       │
└─────────────────────────────────────────────────────────────────────────────┘

┌── Results tab ──────────────────────────────────────────────────────────────┐
│ ResultsTable → click Op cell → OpDetailDrawer opens (same as Artifacts)    │
│ ResultPreviewPanel → [Open in Drawer] → OpDetailDrawer + Result tab         │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Implementation order

### Step 1: Fix split pane (Bug 1)

1. `ArtifactsPanel`: change outer flex from `column` → `row`
2. Give `ArtifactTable` wrapper `flex: 1, minWidth: 0`
3. Give `ArtifactPreviewPanel` wrapper `flex: '0 0 45%'` (or fixed `width`)
4. Ensure outer container has `height` / `minHeight` so the row layout works
5. Collapse the preview placeholder box (the "click to preview" empty state should not take vertical space)

### Step 2: Wire Op column → drawer (Bug 2)

1. `ArtifactTable`: add `onOpClick?: (opId: string) => void` prop
2. Render the Op cell as a clickable `TableCell` (button or link style)
3. `ArtifactsPanel`: add `onOpClick` to props, pass to `ArtifactTable`
4. `WorkflowDetailPage`: add `onOpClick` to `ArtifactsPanel`, wire to `setSelectedOpId` + `setDrawerOpen(true)`

### Step 3: Backend — ListWorkflowResults (new)

1. Add `ListWorkflowResultsOptions` struct in `engineview`
2. Add `ListWorkflowResults` service method in `artifact_read_service.go` (reads `op_results` table via `openStore`)
3. Add `WorkflowResultsResponse` type in `api/types/types.go`
4. Add `WorkflowResults` handler in `engine.go`
5. Register `GET /api/v1/workflows/{workflowID}/results` in `routes_engine.go`
6. Add Go tests

### Step 4: RTK Query endpoint

1. Add `getWorkflowResults` query to `workflowApi.ts`
2. Add `WorkflowResultSummary` type to `types.ts`
3. Export `useGetWorkflowResultsQuery`

### Step 5: ResultsPanel component

1. Create `web/src/components/results/ResultsPanel.tsx` (mirrors `ArtifactsPanel` structure)
2. Create `web/src/components/results/ResultsTable.tsx` (mirrors `ArtifactTable` structure)
3. Create `web/src/components/results/ResultPreviewPanel.tsx` (renders JSON/text, no download needed)
4. Create `web/src/components/results/ResultFilterBar.tsx` (Op + Kind + Status + Search, mirrors `FilterBar`)
5. Create `web/src/components/results/ActiveResultFilterChips.tsx` (mirrors `ActiveFilterChips`)

### Step 6: Wire Results tab into WorkflowDetailPage

1. Add `'results'` to `activeTab` state type
2. Add `ResultsTab` to tab strip
3. Render `ResultsPanel` when `activeTab === 'results'`

### Step 7: Bridge from Results tab to OpDetailDrawer

1. `ResultsTable`: add `onOpClick` (same pattern as `ArtifactTable`)
2. `ResultsPanel`: pass `onOpClick` through
3. `ResultPreviewPanel`: add "Open in Drawer" button → calls `onOpClick(opID)`
4. `WorkflowDetailPage`: wire both to open drawer

### Step 8: Storybook + MSW

1. Add MSW handlers for `/api/v1/workflows/:id/results` in `handlers.ts`
2. Add stories for `ResultsPanel`, `ResultsTable`, `ResultPreviewPanel`

---

## Non-goals (deferred)

- Cross-workflow result search
- Result mutation or deletion
- JS replay from the Results tab
- Multi-select result export
