---
Title: "Artifact Browser — Frontend UI Design"
Ticket: SCRAPER-ARTIFACT-BROWSER
Status: active
Topics:
    - frontend
    - artifacts
    - workflows
    - http-api
    - implementation
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/pages/WorkflowDetailPage.tsx"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/api/workflowApi.ts"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/artifacts/ArtifactList.tsx"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/artifacts/ArtifactPreview.tsx"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/common/MultiSelectChipFilter.tsx"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/api/server/routes_engine.go"
Summary: "Full frontend UI design for the workflow artifact browser: two-panel layout, filter bar, preview panel, YAML component DSL, API mapping, and implementation order."
LastUpdated: 2026-04-07T20:45:00-04:00
WhatFor: "Build the Artifacts tab on the workflow detail page without ambiguity."
WhenToUse: "Use when implementing the Phase 2 frontend work for SCRAPER-ARTIFACT-BROWSER."
ImplementationStatus:
    backend: done
    frontend: not_started
    frontendQuery: not_started
    stories: not_started
---

# Artifact Browser — Frontend UI Design

## Overview

This document describes the full UI design for the workflow artifact browser, Phase 2 of SCRAPER-ARTIFACT-BROWSER.

The backend is **already implemented** (`pkg/api`, `pkg/services/engineview`). The frontend piece is what remains.

**Implementation status**

| Layer | Status |
|---|---|
| Backend service (`ListWorkflowArtifacts`, `GetOpResult`) | ✅ Done |
| Backend handler + route | ✅ Done |
| Server tests | ✅ Done |
| Frontend RTK Query `getWorkflowArtifacts` | ❌ Not started |
| Frontend `ArtifactsTab` component | ❌ Not started |
| Integration into `WorkflowDetailPage` | ❌ Not started |
| Storybook stories | ❌ Not started |

## Layout decision: two-panel split

The artifact browser uses a **master/detail split pane** within the Artifacts tab — not a drawer, not a modal. Rationale:

- `OpDetailDrawer` already occupies the right edge when you drill into an op. Stacking a second drawer would add unwanted depth.
- A split pane lets the table and preview coexist, so users can scan the full artifact list while a preview is open.
- The preview panel is collapsible, giving power users full-width table access.

```
┌─ Workflow: hackernews-extract-frontpage-... ─────────────────────────────────┐
│ Status: succeeded · Site: hackernews · Ops: 12/14                          │
├─ Overview │ Ops │ Runtime │ Artifacts │ Site Info ──────────────────────────┤
│                                                                         [▤]│  ← preview toggle
├──────────────────────────────────┬────────────────────────────────────────┤
│ FilterBar                        │ ArtifactPreviewPanel                  │
│ [Op ▼] [Kind ▼] [Type ▼]        │ Header (name, op, kind, size)         │
│ [search........................]  │ ActionBar: [Open Raw] [↓] [→ Op]     │
│ ─────────────────────────────── │ [× Close]                             │
│ Active: [frontpage-extract ×]    │ ─────────────────────────────────────│
│                                  │ Preview content                        │
│ ArtifactTable                    │ (ArtifactPreview for HTML/JSON/text)  │
│ ┌─────────────────────────────┐  │ (img tag for images)                  │
│ │ ◈ index.html           ▶    │  │ (fallback for binary)                 │
│ │ ◆ summary.json             │  │                                        │
│ │ 📄 debug.log                │  │                                        │
│ │ 🖼 screenshot.png            │  │                                        │
│ └─────────────────────────────┘  │                                        │
│ Showing 1–5 of 12    [←] 1/3 [→] │                                        │
└──────────────────────────────────┴────────────────────────────────────────┘
```

## Screen 1: Full artifact browser view

```
╔══════════════════════════════════════════════════════════════════════════════════════════════════╗
║ Workflow: hackernews-extract-frontpage-1775586649974859668                                         ║
║ Status: succeeded    Site: hackernews    Ops: 12/14     Last run: 2 hours ago                   ║
╠══════════════════════════════════════════════════════════════════════════════════════════════════╣
║  Overview  │   Ops   │  Runtime  │  Artifacts  │  Site Info                                       ║
╠══════════════════════════════════════════════════════════════════════════════════════════════════╣
║                                                                                                   ║
║  [▤ Preview panel]   Op: [All Ops                    ▼]   Kind: [All           ▼]             ║
║  Type: [All types                ▼]   search: [search by name...                        ] 🔍   ║
║  ────────────────────────────────────────────────────────────────────────────────────────────  ║
║  Active filters:  [frontpage-extract ×]  [json-output ×]                                        ║
║  Clear all                                                                                       ║
║                                                                                                   ║
║  ┌──────────────────────────────────────────────────────────────────────────────────────────┐    ║
║  │ ◈ Name                  │ Op                      │ Kind          │ Size    │ Actions │    ║
║  ├────────────────────────┼─────────────────────────┼───────────────┼─────────┼─────────┤    ║
║  │ ▸ index.html           │ ...:frontpage-fetch     │ http-body     │ 48 KB   │ View ↓  │    ║
║  │ ◆ summary.json         │ ...:frontpage-extract   │ json-output   │  2 KB   │ View ↓  │    ║
║  │ 📄 execution-log.json  │ ...:frontpage-extract   │ exec-log      │ 12 KB   │ View ↓  │    ║
║  │ ◆ listing-page-2.json │ ...:frontpage-extract   │ json-output   │  4 KB   │ View ↓  │    ║
║  │ ▸ detail-12345.html    │ ...:item-detail-fetch  │ http-body     │ 31 KB   │ View ↓  │    ║
║  │ 🖼 screenshot.png       │ ...:item-detail-fetch  │ screenshot    │  1.0 MB │ View ↓  │    ║
║  └──────────────────────────────────────────────────────────────────────────────────────────┘    ║
║                                                                                                   ║
║  Showing 1–5 of 12 artifacts                                     [←]  Page 1 of 3  [→]        ║
║                                                                                                   ║
╠══════════════════════════════════════════════════════════════════════════════════════════════════╣
║                                                                                                   ║
║  Artifact: summary.json                                                  [Open Raw] [↓] [×]      ║
║  Op: ...:frontpage-extract  [→ Op detail]                                                   ║
║  Kind: json-output    Content-Type: application/json    Size: 2 KB    Created: 2h ago         ║
║  ───────────────────────────────────────────────────────────────────────────────────────────  ║
║                                                                                                   ║
║  ┌──────────────────────────────────────────────────────────────────────────────────────────┐    ║
║  │ {                                                                                       │    ║
║  │   "stories": 30,                                                                        │    ║
║  │   "nextPage": "/news?p=2",                                                             │    ║
║  │   "scrapedAt": "2026-04-07T14:32:05Z",                                                  │    ║
║  │   "site": "hackernews"                                                                  │    ║
║  │ }                                                                                       │    ║
║  └──────────────────────────────────────────────────────────────────────────────────────────┘    ║
║                                                                                                   ║
╚══════════════════════════════════════════════════════════════════════════════════════════════════╝
```

## Screen 2: Preview panel — HTML artifact

```
╔════════════════════════════════════════════════════════════════════════════════╗
║ Artifact: index.html                                                    [Open Raw] [↓] [×] ║
║ Op: ...:frontpage-fetch  [→ Op detail]                                               ║
║ Kind: http-body    Content-Type: text/html    Size: 48 KB    Created: 2h ago        ║
╠════════════════════════════════════════════════════════════════════════════════╣
║                                                                                ║
║  <!DOCTYPE html>                                                               ║
║  <html lang="en">                                                              ║
║  <head>                                                                        ║
║    <title>Hacker News</title>                                                  ║
║    <link rel="stylesheet" href="news.css?...">                                  ║
║  </head>                                                                       ║
║  <body>                                                                        ║
║    <table class="itemlist">                                                    ║
║      <tr class="athing" id="12345">                                           ║
║        <td class="title">                                                      ║
║          <a href="https://example.com">Show HN: A cool project</a>             ║
║        </td>                                                                   ║
║      </tr>                                                                     ║
║      <tr><td class="subtext">142 points by user1 · 3 hours ago</td></tr>     ║
║      ...                                                                      ║
║    </table>                                                                    ║
║  </body>                                                                       ║
║  </html>                                                                       ║
║                                                                                ║
╚════════════════════════════════════════════════════════════════════════════════╝
```

## Screen 3: Preview panel — image artifact

```
╔════════════════════════════════════════════════════════════════════════════════╗
║ Artifact: screenshot.png                                               [Open Raw] [↓] [×] ║
║ Op: ...:item-detail-fetch  [→ Op detail]                                            ║
║ Kind: screenshot    Content-Type: image/png    Size: 1.0 MB    Created: 2h ago      ║
╠════════════════════════════════════════════════════════════════════════════════╣
║                                                                                ║
║      ┌─────────────────────────────────────────────────────────────────┐      ║
║      │                                                                 │      ║
║      │              [  rendered <img> tag — full width ]              │      ║
║      │                                                                 │      ║
║      └─────────────────────────────────────────────────────────────────┘      ║
║                                                                                ║
║                           [ Download full image ]                               ║
╚════════════════════════════════════════════════════════════════════════════════╝
```

## Screen 4: Preview panel — binary/unknown artifact

```
╔════════════════════════════════════════════════════════════════════════════════╗
║ Artifact: response.bin                                               [Open Raw] [↓] [×] ║
║ Op: ...:api-call         [→ Op detail]                                               ║
║ Kind: raw    Content-Type: application/octet-stream    Size: 2.1 MB    Created: 2h  ║
╠════════════════════════════════════════════════════════════════════════════════╣
║                                                                                ║
║                         🗁  Binary file — cannot preview                       ║
║                                                                                ║
║                           Size: 2.1 MB                                          ║
║                           Type: application/octet-stream                        ║
║                                                                                ║
║                          [ Download file ]                                      ║
║                                                                                ║
╚════════════════════════════════════════════════════════════════════════════════╝
```

## Screen 5: Empty state — no artifacts

```
╔════════════════════════════════════════════════════════════════════════════════╗
║  [▤ Preview panel]   Op: [All Ops                    ▼]   Kind: [All           ▼]   ║
║  ─────────────────────────────────────────────────────────────────────────────────   ║
║                                                                                        ║
║                              🔍  No artifacts found                                  ║
║                                                                                        ║
║            This workflow has not produced any artifacts yet,                          ║
║            or the current filters match nothing.                                      ║
║                                                                                        ║
║                         [ Clear all filters ]                                        ║
║                                                                                        ║
╚════════════════════════════════════════════════════════════════════════════════════════╝
```

## Screen 6: Bridge to op drawer

The "→ Op detail" link in the preview panel switches to the Ops tab and opens `OpDetailDrawer` for that artifact's owning op. The Op detail drawer already has a reverse link ("See all artifacts from this op in the browser" in the `OpResultTab`), which filters the Artifacts tab to that op.

```
OpResultTab (in OpDetailDrawer)
────────────────────────────────────────────
...
Artifacts: 3    Emitted: 2

↗ See all artifacts from this op in the browser
   → switches to Artifacts tab, sets opId filter
```

## Component hierarchy (YAML DSL)

This DSL describes the component tree in shorthand. `→` means "renders child", `uses:` means "reads from API or state".

```yaml
# ─── Page / Route ───────────────────────────────────────────────────────────────
WorkflowDetailPage:
  # Existing. The ArtifactsTab is wired into the tab bar.
  # On tab switch to Artifacts, renders ArtifactsPanel.

  ArtifactsPanel:                          # ← NEW: two-panel split pane
    uses: [useGetWorkflowArtifactsQuery]
    state: [selectedArtifactId, previewVisible, filters]

    FilterBar:                             # ← NEW
      uses: [useGetWorkflowOpsQuery]       # to populate Op dropdown
      state: [filters]
      controls:
        - Select: opId filter   (from ops list, value = op.ID, label = op.op.Kind or short name)
        - Select: kind filter    (from enum/backend, value = artifact.kind)
        - Select: contentType filter
        - TextInput: search      (debounced, maps to ?search=)
      children:
        - ActiveFilterChips:     # ← NEW: shows dismissible active filters
            uses: [filters]
        - Button: Clear all

    ArtifactTable:                         # ← NEW (or refactor ArtifactList)
      uses: [useGetWorkflowArtifactsQuery, filters]
      state: [selectedArtifactId]
      children:
        - TableHead: [Name, Op, Kind, Size, Actions]
        - TableRow × N:
            cells:
              - NameCell:         icon + truncated name (tooltip on hover)
              - OpCell:           short op ID, clickable → op-detail link
              - KindCell:         badge/chip
              - SizeCell:         human-readable (12 KB)
              - ActionsCell:      [View] [↓ Download]
            onClick: → selectArtifact(id)
        - TablePagination:
            shows: "Showing 1–5 of {total}" + prev/next
            uses: [useGetWorkflowArtifactsQuery]

    ArtifactPreviewPanel:                  # ← NEW (right half, collapsible)
      state: [selectedArtifactId, previewVisible]
      children:
        - PreviewHeader:                  # artifact name + close button
        - PreviewMeta:                    # op link, kind, size, created
        - PreviewActions:                  # [Open Raw] [↓] [→ Op detail]
        - PreviewContent:                  # conditional on contentType
            when text/html or text/plain or application/json:
              → ArtifactPreview            # existing component
            when image/*:
              → img (src = /api/v1/artifacts/{id})
            when binary:
              → BinaryFallbackView        # icon + size + download CTA
        - PreviewBodyFetch:               # fetch /api/v1/artifacts/{id} on select
          uses: [selectedArtifactId]
```

## Existing components to reuse

| Component | File | Usage |
|---|---|---|
| `ArtifactPreview` | `components/artifacts/ArtifactPreview.tsx` | Renders HTML, JSON, plain-text previews. Already has stories. |
| `ArtifactList` | `components/artifacts/ArtifactList.tsx` | List rendering with selection. May need adaptation for table layout (currently List-based, not Table-based). |
| `MultiSelectChipFilter` | `components/common/MultiSelectChipFilter.tsx` | Already used for queue filters. Can be reused for the filter bar. |
| `JsonViewer` | `components/common/JsonViewer.tsx` | JSON rendering with collapse. Already used in `OpResultTab`. |

## New components to create

| Component | File | Notes |
|---|---|---|
| `ArtifactsPanel` | `components/artifacts/ArtifactsPanel.tsx` | Root: two-panel layout, owns filter state |
| `FilterBar` | `components/artifacts/FilterBar.tsx` | Op dropdown + Kind + Type + Search inputs |
| `ActiveFilterChips` | `components/artifacts/ActiveFilterChips.tsx` | Dismissible chips for active filters |
| `ArtifactTable` | `components/artifacts/ArtifactTable.tsx` | Full-width table, replaces `ArtifactList` list layout |
| `ArtifactPreviewPanel` | `components/artifacts/ArtifactPreviewPanel.tsx` | Right half preview with header + actions |
| `BinaryFallbackView` | `components/artifacts/BinaryFallbackView.tsx` | Icon + size + download for non-previewable artifacts |

## API mapping

All artifact browsing flows through these three backend endpoints:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Frontend data flow                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  WorkflowDetailPage                                              │
│    │                                                            │
│    ├── useGetWorkflowQuery(workflowId)                          │
│    │       → GET /api/v1/workflows/{workflowID}                  │
│    │                                                            │
│    ├── useGetWorkflowOpsQuery(workflowId)                       │
│    │       → GET /api/v1/workflows/{workflowID}/ops              │
│    │       Used to: populate Op filter dropdown                 │
│    │                                                            │
│    ├── useGetWorkflowArtifactsQuery({ workflowId, ...filters })│
│    │       → GET /api/v1/workflows/{workflowID}/artifacts        │
│    │           ?opId=...&kind=...&contentType=...&search=...    │
│    │           &limit=20&offset=0                               │
│    │       Returns: { workflowID, total, artifacts[] }           │
│    │                                                            │
│    └── ArtifactsPanel (new)                                     │
│          │                                                       │
│          ├── ArtifactTable (displays artifacts[])              │
│          │                                                       │
│          └── ArtifactPreviewPanel                               │
│                  │                                              │
│                  └── fetch /api/v1/artifacts/{id}                │
│                          → GET /api/v1/artifacts/{artifactID}   │
│                          Returns: raw body (text or binary)      │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Endpoint 1 — workflow artifact list

```
GET /api/v1/workflows/{workflowID}/artifacts
```

**Frontend caller**: `useGetWorkflowArtifactsQuery` (new RTK Query endpoint)

**Query params**:
| Param | Source | Notes |
|---|---|---|
| `opId` | `FilterBar` → op dropdown | Filter to single op |
| `kind` | `FilterBar` → kind dropdown | e.g. `http-response-body`, `json-output` |
| `contentType` | `FilterBar` → type dropdown | e.g. `text/html`, `application/json` |
| `search` | `FilterBar` → search input | Substring match on `artifact.name` |
| `limit` | `ArtifactTable` pagination | Default 20 |
| `offset` | `ArtifactTable` pagination | `offset = (page - 1) * limit` |

**Response**:
```json
{
  "workflowID": "wf-123",
  "total": 12,
  "artifacts": [
    {
      "id": "wf-123:frontpage-fetch:response-body",
      "opID": "wf-123:frontpage-fetch",
      "workflowID": "wf-123",
      "name": "frontpage.html",
      "kind": "http-response-body",
      "contentType": "text/html",
      "metadata": { "method": "GET" },
      "size": 48123,
      "createdAt": "2026-04-07T14:20:01Z",
      "previewable": true,
      "previewKind": "html"
    }
  ]
}
```

### Endpoint 2 — artifact body (existing)

```
GET /api/v1/artifacts/{artifactID}
```

**Frontend caller**: `ArtifactPreviewPanel` fetches on `selectedArtifactId` change

**Behavior**: Returns raw body. `content-type` header matches `artifact.contentType`.

### Endpoint 3 — op detail (for bridge link)

```
GET /api/v1/workflows/{workflowID}/ops/{opID}
```

Existing. Used by `OpDetailDrawer` when "→ Op detail" is clicked from the preview panel. The `WorkflowDetailPage` already fetches ops via `useGetWorkflowOpsQuery`.

## New RTK Query endpoint

```typescript
// workflowApi.ts

getWorkflowArtifacts: builder.query<
  ArtifactSummary[],
  {
    workflowId: string;
    opId?: string;
    kind?: string;
    contentType?: string;
    search?: string;
    limit?: number;
    offset?: number;
  }
>({
  query: ({ workflowId, opId, kind, contentType, search, limit = 20, offset = 0 }) => {
    const sp = new URLSearchParams();
    if (opId) sp.set('opId', opId);
    if (kind) sp.set('kind', kind);
    if (contentType) sp.set('contentType', contentType);
    if (search) sp.set('search', search);
    sp.set('limit', String(limit));
    sp.set('offset', String(offset));
    return `/workflows/${workflowId}/artifacts?${sp}`;
  },
  transformResponse: (response: { workflowID: string; total: number; artifacts: ArtifactSummary[] }) =>
    response.artifacts,
  providesTags: (_result, _error, { workflowId }) => [
    { type: 'WorkflowArtifacts', id: workflowId },
  ],
})
```

Export: `useGetWorkflowArtifactsQuery`.

Invalidate tag `{ type: 'WorkflowArtifacts', id: workflowId }` when future operations create or delete artifacts (e.g., replay).

## Implementation order

### Step 1 — Wire the query

1. Add `getWorkflowArtifacts` to `workflowApi.ts`.
2. Export `useGetWorkflowArtifactsQuery`.
3. Validate: fetch the endpoint from the browser devtools against a real running workflow.

### Step 2 — `ArtifactsPanel` skeleton

1. Create `web/src/components/artifacts/ArtifactsPanel.tsx`.
2. Render `FilterBar` + `ArtifactTable` side-by-side (no preview panel yet).
3. Wire the query with a hardcoded `limit=20, offset=0`.
4. Add loading skeleton rows (5 grey bars matching table shape).
5. Add empty state component.

### Step 3 — Filter bar

1. Create `FilterBar.tsx` using `MultiSelectChipFilter` as a base.
2. Populate op dropdown from `useGetWorkflowOpsQuery`.
3. Add client-side filter chips for active filters.
4. Wire `onFilterChange` → rebuild query params → `refetch()`.

### Step 4 — Pagination

1. Add "Showing 1–N of {total}" + prev/next to `ArtifactTable`.
2. Wire `page` state → `offset` param.
3. Disable prev/next at boundaries.

### Step 5 — Preview panel

1. Create `ArtifactPreviewPanel.tsx` (right half).
2. On row click: set `selectedArtifactId`, fetch `/api/v1/artifacts/{id}`.
3. Dispatch content types:
   - `text/html` / `text/plain` / `application/json` → `ArtifactPreview`
   - `image/*` → `<img>` tag
   - anything else → `BinaryFallbackView`
4. Add "Preview panel" toggle in the panel header (collapses the right half).
5. Implement "Open Raw" → `window.open('/api/v1/artifacts/{id}')`.
6. Implement "Download" → programmatic `<a>` download.
7. Implement "→ Op detail" → navigate to Ops tab + open drawer.

### Step 6 — Bridge: Op result tab → artifact browser

1. In `OpResultTab`, add link: `↗ See all artifacts from this op in the browser`.
2. On click: switch to Artifacts tab + set `opId` filter to the current op's ID.
3. Requires lifting `activeTab` state or using a URL param.

### Step 7 — Stories

1. `ArtifactsPanel.stories.tsx`: default, loading, empty, with preview open.
2. `ArtifactPreviewPanel.stories.tsx`: HTML, JSON, plain text, image, binary.
3. `FilterBar.stories.tsx`: with ops loaded, no ops, with active filters.
4. `ArtifactTable.stories.tsx`: single page, multi page, empty, loading skeleton.

## Non-goals (unchanged from backend doc)

- JS replay / debug execution
- Artifact deletion or mutation
- Multi-select artifact download (zip)
- Cross-workflow artifact search
