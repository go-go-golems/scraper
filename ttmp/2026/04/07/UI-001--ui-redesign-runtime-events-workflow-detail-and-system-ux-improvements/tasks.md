# Tasks

Detailed implementation tasks for UI-001. Each task is atomic and testable.
Phases are ordered by dependency — do not skip ahead.

**Legend:**
- 🔴 = blocking / other tasks depend on this
- 🟡 = important but not blocking
- 🟠 = nice to have
- 📝 = no code, docs/design only
- ⬜ = not started

---

## Phase 0: Foundation (Day 1–2)

_Infrastructure that all later phases depend on._

### 0.1 Error Boundary 🔴

- [ ] ⬜ Create `web/src/components/common/AppErrorBoundary.tsx`
  - Use React class component with `componentDidCatch` / `getDerivedStateFromError`
  - Fallback UI: MUI Card with "Something went wrong" heading
  - Show error.message always; show error.stack only in `import.meta.env.DEV`
  - "Try Again" button calls `resetErrorBoundary()` (reset state to re-render children)
  - Accept `children` prop, no other config needed
- [ ] ⬜ Create `web/src/components/common/AppErrorBoundary.stories.tsx`
  - Story "Error": renders a child that throws, verify fallback card appears
  - Story "Healthy": renders normal children, no error
- [ ] ⬜ Wire into `App.tsx`
  - Wrap `<AppShell>{children}</AppShell>` with `<AppErrorBoundary>`
  - Verify: introduce a deliberate crash on Overview page, confirm fallback renders
- [ ] ⬜ Manual test
  - Visit each tab in the nav, verify normal operation
  - Force an error (e.g., throw in EngineOverviewPage), verify boundary catches it

### 0.2 Toast Notification System 🔴

- [ ] ⬜ Create `web/src/components/common/ToastContext.tsx`
  - React context exposing `showToast(message: string, severity: 'success' | 'error' | 'info' | 'warning')`
  - Internal queue to prevent stacking more than 3 toasts at once
  - Auto-dismiss after 4 seconds (configurable per call)
- [ ] ⬜ Create `web/src/components/common/ToastProvider.tsx`
  - Wraps children + renders MUI `<Snackbar>` anchored bottom-right
  - Uses MUI `<Alert>` inside Snackbar for severity coloring
  - `useToast()` hook for consumers
- [ ] ⬜ Create `web/src/components/common/ToastProvider.stories.tsx`
  - Story "Success": green toast "Workflow submitted"
  - Story "Error": red toast "Failed to cancel workflow"
  - Story "Stacked": fire 3 toasts in sequence, verify queue behavior
- [ ] ⬜ Wire `ToastProvider` into `App.tsx` above `AppErrorBoundary`
- [ ] ⬜ Integrate into `SubmitWorkflowPage.tsx`
  - After `submitMutation(...).unwrap()` succeeds → `showToast('Workflow submitted', 'success')`
  - After mutation fails → `showToast(error.message, 'error')`
- [ ] ⬜ Integrate into `CancelWorkflowButton.tsx`
  - After cancel succeeds → `showToast('Workflow canceled', 'info')`
  - After cancel fails → `showToast(error.message, 'error')`
- [ ] ⬜ Integrate into `RetryOpButton.tsx`
  - After retry succeeds → `showToast('Op retry initiated', 'success')`
  - After retry fails → `showToast(error.message, 'error')`

### 0.3 Breadcrumb Navigation 🟡

- [ ] ⬜ Create `web/src/components/layout/BreadcrumbNav.tsx`
  - Use `useLocation()` + `useNavigate()` to derive crumbs from current path
  - Route → crumb mapping:
    - `/` → `[Overview]`
    - `/workflows` → `[Workflows]`
    - `/workflows/:id` → `[Workflows, {workflow.name}]` (name from RTK query cache or route state)
    - `/events` → `[Events]`
    - `/queues` → `[Queues]`
    - `/sites` → `[Sites]`
    - `/sites/:name` → `[Sites, {name}]`
    - `/submit` → `[Submit]`
  - Last crumb is plain text (current page), all others are MUI `<Link>` clickable
  - Use MUI `<Breadcrumbs>` component with `>` separator
  - If there is only 1 crumb, render nothing (don't show breadcrumb for top-level pages)
- [ ] ⬜ Create `web/src/components/layout/BreadcrumbNav.stories.tsx`
  - Story "TopLevel": single crumb "Overview" → renders nothing
  - Story "WorkflowsList": single crumb "Workflows" → renders nothing
  - Story "WorkflowDetail": crumbs "Workflows > scrape-hackernews"
  - Story "SiteDetail": crumbs "Sites > hackernews"
- [ ] ⬜ Add `<BreadcrumbNav />` to `AppShell.tsx`
  - Insert between AppBar Toolbar and the `<Box sx={{ flexGrow: 1, p: 3 }}>` content area
  - Give it left padding `px: 3` to align with content, and `py: 0.5`
- [ ] ⬜ Update `WorkflowDetailPage.tsx` to pass workflow name via route state
  - When navigating from `WorkflowTable` click, pass `state: { workflowName: item.workflow.Name }`
  - BreadcrumbNav reads `location.state?.workflowName` as fallback for crumb label

---

## Phase 1: Shared Components (Day 3–5)

_Reusable components that all page redesigns need._

### 1.1 MultiSelectChipFilter 🔴

- [ ] ⬜ Create `web/src/components/common/MultiSelectChipFilter.tsx`
  - Props: `label: string`, `options: { value: string, label: string, color?: ChipProps['color'] }[]`, `selected: string[]`, `onChange: (selected: string[]) => void`
  - Use MUI `<Autocomplete multiple>` with `size="small"`
  - `renderTags`: render each selected value as a `<Chip>` with the option's color, deletable
  - `renderInput`: MUI `<TextField>` with the label
  - `filterOptions`: default MUI fuzzy filter is fine
  - When `selected` is empty array → shows "All" as placeholder text (not as a chip)
  - When user removes all chips → calls `onChange([])` (which means "show all" to parent)
- [ ] ⬜ Create `web/src/components/common/MultiSelectChipFilter.stories.tsx`
  - Story "Empty": severity options, none selected, shows "All" placeholder
  - Story "MultipleSelected": WARN + ERROR selected as colored chips
  - Story "AllSelected": all 4 severity options selected
  - Story "WithCustomColors": each option has explicit color mapping
  - Story "Disabled": `disabled={true}`, verify chips are non-interactive
- [ ] ⬜ Export from `web/src/components/common/index.ts` barrel (create if needed)

### 1.2 TimeRangeSelector 🔴

- [ ] ⬜ Install date dependencies: `pnpm add dayjs @mui/x-date-pickers`
  - Verify `dayjs` is the adapter for `@mui/x-date-pickers` (LocalizationProvider)
- [ ] ⬜ Create `web/src/components/common/TimeRangeSelector.tsx`
  - Props: `value: TimeRange`, `onChange: (value: TimeRange) => void`, `options?: string[]`
  - `TimeRange` type:
    ```
    type TimeRange =
      | { mode: 'live' }
      | { mode: 'relative'; range: string }   // e.g. '1h', '6h', '24h', '7d'
      | { mode: 'absolute'; from: string; to: string }  // ISO strings
    ```
  - Default options: `['live', '1h', '6h', '24h', '7d', 'custom']`
  - Render a row of MUI `<Chip>` components (like a button group)
  - Active chip gets `variant="filled"`, others get `variant="outlined"`
  - "Live" chip shows a small green dot prefix: `● Live`
  - "Custom" chip: when active, reveal a MUI `<DateRangePicker>` below the chips
  - Relative ranges: when selected, call `onChange({ mode: 'relative', range })`
  - Wire `<LocalizationProvider dateAdapter={AdapterDayjs}>` at the component level
- [ ] ⬜ Create `web/src/components/common/TimeRangeSelector.stories.tsx`
  - Story "LiveMode": "Live" chip filled, others outlined
  - Story "Relative6h": "Last 6h" chip filled
  - Story "Relative24h": "Last 24h" chip filled
  - Story "CustomMode": "Custom" chip filled, date range picker visible, pre-filled today→now
  - Story "MinimalOptions": pass `options={['live', '1h', '24h']}`, verify only those chips render

### 1.3 AlertBanner 🟡

- [ ] ⬜ Create `web/src/components/common/AlertBanner.tsx`
  - Props: `severity: 'error' | 'warning' | 'info'`, `message: string`, `action?: { label: string, onClick: () => void }`, `dismissible?: boolean` (default true), `autoDismissMs?: number` (default null, set 30000 for 'info')
  - Use MUI `<Alert>` with `severity`, `variant="filled"` for error, `variant="standard"` for warning/info
  - If `action` provided, render `<Button size="small" color="inherit">` inside `<Alert action={...}>`
  - If `dismissible`, render close icon that sets internal `visible` state to false
  - If `autoDismissMs` is set, start a timer on mount that auto-dismisses
  - If `visible === false`, render `null`
- [ ] ⬜ Create `web/src/components/common/AlertBanner.stories.tsx`
  - Story "ErrorAlert": "7 ops failed in the last hour" with [View Failed Ops] button
  - Story "WarningAlert": "Queue site:hn:http at 95% capacity" with [View Queue] button
  - Story "InfoAlert": "Engine restarted successfully", auto-dismiss after 5s (for story demo)
  - Story "NonDismissible": `dismissible={false}`, no close icon
  - Story "NoAction": just message, no action button, but dismissible

---

## Phase 2: RuntimeEventTable (Day 5–8)

_The core event table that replaces RuntimeEventList in all 3 locations._

### 2.1 Create RuntimeEventTable Component 🔴

- [ ] ⬜ Create `web/src/components/workflows/RuntimeEventTable.tsx`
  - Props: `events: RuntimeEventV1[]`, `loading: boolean`, `dense?: boolean` (default true), `expandable?: boolean` (default true), `showFilters?: boolean` (default false), `showPagination?: boolean` (default false), `onWorkflowClick?: (id: string) => void`, `onOpClick?: (id: string) => void`, `emptyMessage?: string`
  - State: `expandedRowId: string | null`, `sortField: 'timestamp' | 'severity' | 'source' | 'kind'`, `sortDir: 'asc' | 'desc'`, `page: number`, `rowsPerPage: number`
  - Table columns (MUI `<Table size="small">`):
    - **Time** (100px): `event.occurredAt` → `toLocaleTimeString()`, sortable
    - **Severity** (80px): colored dot + label, sortable by enum value (DEBUG < INFO < WARN < ERROR)
    - **Source** (100px): text label from `RuntimeEventSource[event.source]`, sortable
    - **Kind** (140px): normalized label via `normalizeEnumLabel`, sortable
    - **Message** (flex): `event.message`, truncate to 80 chars with ellipsis
  - Expandable rows: clicking a row toggles `expandedRowId`
  - Expanded detail row (`RuntimeEventDetailRow`):
    - Full message text (no truncation)
    - Op ID: if `event.opId` exists, render as clickable link (`onOpClick`)
    - Workflow ID: if `event.workflowId` exists, render as clickable link (`onWorkflowClick`)
    - Site, Worker ID, Queue, Request ID, Artifact ID as inline captions
    - Full payload rendered via `<JsonViewer>` component
  - Sorting: client-side sort the `events` array by `sortField` + `sortDir`
  - Column headers: render with sort indicator arrow, `onClick` toggles sort
  - If `showFilters`: render toolbar above table with `<MultiSelectChipFilter>` for severity and source
  - If `showPagination`: render MUI `<TablePagination>` at bottom
  - Empty state: centered message with icon
  - Loading state: 5 skeleton rows
- [ ] ⬜ Create `web/src/components/workflows/SeverityDotIndicator.tsx`
  - Tiny helper: renders a 10px colored circle + label text
  - Color map: DEBUG=grey, INFO=blue, WARN=orange, ERROR=red
  - Uses `<Box>` with `borderRadius: '50%'` + `<Typography variant="caption">`
- [ ] ⬜ Create `web/src/components/workflows/RuntimeEventTable.stories.tsx`
  - Story "Empty": 0 events, shows empty message
  - Story "WithEvents": 20 mock events, mixed severity/source, dense mode
  - Story "ExpandedRow": 20 events, 3rd row expanded showing full detail
  - Story "WithFilters": `showFilters={true}`, pre-select ERROR+WARN in severity
  - Story "Loading": `loading={true}`, shows skeleton rows
  - Story "StreamLive": renders with a "live" connection indicator
  - Story "StreamError": renders with an "error" connection state

### 2.2 Update runtimeEventFeed Hook 🔴

- [ ] ⬜ Edit `web/src/api/runtimeEventsApi.ts`
  - Add to `RuntimeEventsParams` type: `since?: string`, `until?: string`, `offset?: number`
  - Update `buildRuntimeEventQuery` to include `since`, `until`, `offset` in URL params
  - Update `transformResponse` to extract `total` count: `response.total ?? response.events.length`
  - Change return type to `{ events: RuntimeEventJson[], total: number }`
  - Update `useGetRecentRuntimeEventsQuery` accordingly
- [ ] ⬜ Edit `web/src/features/runtime-events/runtimeEventFeed.ts`
  - Add `pause()` / `resume()` to return value:
    - `pause()`: close the EventSource, set `connectionState` to `'paused'`, do NOT clear events
    - `resume()`: re-open the EventSource with same params, set state to `'connecting'`
  - Add `since`/`until` support:
    - Accept `timeRange?: TimeRange` in `UseRuntimeEventFeedOptions`
    - When mode is `'relative'`, compute `since` as `now - range` and pass to `serverFilters`
    - When mode is `'absolute'`, pass `since`/`until` directly
    - When mode is `'live'`, do NOT pass `since`/`until` (current behavior)
  - Add `totalCount` to return value (from API response)
- [ ] ⬜ Update `web/src/features/runtime-events/runtimeEventFeed.test.ts`
  - Add test: `pause()` closes EventSource without clearing events
  - Add test: `resume()` reopens EventSource
  - Add test: time range 'relative 1h' sets `since` to ~1 hour ago
  - Add test: time range 'live' does not set `since`/`until`

### 2.3 Replace RuntimeEventList in RuntimeEventsPage 🔴

- [ ] ⬜ Edit `web/src/pages/RuntimeEventsPage.tsx`
  - Remove old `<TextField select>` for severity and source
  - Remove old `<TextField>` for workflowId, opId, site, workerId (keep these but move into a collapsible "Advanced" section)
  - Add `<TimeRangeSelector>` at top, bound to `timeRange` state
  - Add `<MultiSelectChipFilter label="Severity">` and `<MultiSelectChipFilter label="Source">`
  - Replace `<RuntimeEventList>` with `<RuntimeEventTable showFilters showPagination>`
  - Add Pause/Resume button next to Clear button
  - Update `useRuntimeEventFeed` call to pass `timeRange`
  - Stream status chips remain (connection state, event count, last event timestamp)
- [ ] ⬜ Verify: visit `/events`, confirm table renders, filters work, time range works

### 2.4 Replace RuntimeEventList in WorkflowDetailPage 🔴

- [ ] ⬜ Edit `web/src/pages/WorkflowDetailPage.tsx`
  - Move runtime events from standalone `<Card>` into a tab (see Phase 3 for full tab layout)
  - For now: just replace `<RuntimeEventList>` with `<RuntimeEventTable showFilters={false}>`
  - Pass `serverFilters={{ workflowId, limit: 50 }}` to keep it workflow-scoped
  - Keep the `useRuntimeEventFeed` call, just swap the rendering component
- [ ] ⬜ Verify: visit `/workflows/:id`, confirm events section shows table instead of list

### 2.5 Replace RuntimeEventList in OpDetailDrawer 🔴

- [ ] ⬜ Edit `web/src/components/workflows/OpDetailDrawer.tsx`
  - In the "Runtime" tab section (around the `<RuntimeEventList>` usage):
    - Replace with `<RuntimeEventTable events={opRuntimeEvents} loading={...} dense />`
    - Remove the connection state chips from inside the tab (they're noisy in a drawer)
    - Keep the stream connection logic (useRuntimeEventFeed stays)
- [ ] ⬜ Verify: open OpDetailDrawer, click Runtime tab, confirm table renders in 500px drawer

### 2.6 Backend API Coordination 📝

- [ ] ⬜ Create backend ticket: "Add `since`, `until`, `offset` params to `GET /api/v1/runtime-events`"
  - Response should include `{ events: [...], total: number }`
  - SSE stream should support `since` param for backfill
- [ ] ⬜ If backend is not ready yet, implement client-side time filtering as interim:
  - After fetching events, filter by `runtimeEventOccurredAtMillis` against `since`/`until`
  - Add a code comment `// TODO: move to server-side when backend supports it`
  - Pagination will also be client-side until backend supports `offset`

---

## Phase 3: Workflow Detail Page Redesign (Day 8–11)

_Transform from vertical scroll into a tabbed, dense layout._

### 3.1 Create WorkflowSummaryCard Component 🔴

- [ ] ⬜ Create `web/src/components/workflows/WorkflowSummaryCard.tsx`
  - Props: `workflow: WorkflowDetail`, `stats: WorkflowStats`, `onCancel: () => void`, `cancelLoading: boolean`
  - Layout (3 rows):
    - Row 1: workflow name (`<Typography variant="h5">`), site chip, status chip, duration, copy-ID icon button, cancel button
    - Row 2: started-at, submitted-by (if available), other metadata as key-value pairs
    - Row 3: `<WorkflowProgressBar stats={stats} />` with fraction "12/20 ops complete"
  - Duration: compute `now - workflow.createdAt` for running, or `completedAt - createdAt` for finished
  - Copy-ID: use `navigator.clipboard.writeText(workflowId)` + `showToast('Copied', 'info')`
  - Replaces: inline usage of `<WorkflowHeader>` + `<WorkflowProgressBar>` + `<CancelWorkflowButton>`
- [ ] ⬜ Create `web/src/components/workflows/WorkflowSummaryCard.stories.tsx`
  - Story "Running": name, site, running status, ticking duration, 60% progress
  - Story "Succeeded": full green progress bar, finalized duration
  - Story "Failed": red status, 40% progress, error message shown
  - Story "Canceled": yellow status, partial progress
- [ ] ⬜ Wire into `WorkflowDetailPage.tsx` (replace the old header + progress bar)

### 3.2 Rewrite WorkflowDetailPage with Tabs 🔴

- [ ] ⬜ Edit `web/src/pages/WorkflowDetailPage.tsx` — major restructure
  - Replace the vertical `<Box>` stack of Cards with:
    1. Breadcrumb (already added in Phase 0)
    2. `<WorkflowSummaryCard>` (from 3.1 above)
    3. MUI `<Tabs>` with 4 tabs:
       - **Operations (N)** — see 3.3
       - **Runtime Events (N)** — see 3.4
       - **Artifacts (N)** — see 3.5
       - **JSON** — raw `<JsonViewer data={workflow} />`
  - Tab state: `const [activeTab, setActiveTab] = useState<'ops' | 'events' | 'artifacts' | 'json'>('ops')`
  - Each tab renders only when active (avoid fetching artifacts when on ops tab)
  - Keep `<OpDetailDrawer>` at the bottom (it's a drawer, always mounted but conditionally open)
  - Remove the old standalone `<Card>` for runtime events (moved into tab)
  - Remove the old standalone `<Card>` for ops table (moved into tab)
- [ ] ⬜ Verify: visit `/workflows/:id`, confirm tabs render, switching works, drawer still opens

### 3.3 Upgrade OpTable with Duration, Sort, Filter 🔴

- [ ] ⬜ Edit `web/src/components/workflows/OpTable.tsx`
  - Add new props: `statusFilter?: string`, `searchText?: string`
  - Add **Duration** column (after Retry, before Created):
    - Running ops: `Date.now() - createdAt` → format as "Xm Ys" with a pulsing green dot
    - Succeeded/Failed ops: if `completedAt` exists, show duration; otherwise "—"
    - Pending ops: "—"
    - Need to check if `WorkflowOp` type has timing fields; if not, add to `web/src/api/types.ts`
  - Make column headers sortable:
    - Clickable `<TableCell>` with sort arrow indicator
    - Local state: `sortBy: string`, `sortDir: 'asc' | 'desc'`
    - Sort the ops array client-side before rendering
  - Add filter bar above table (inside the "Operations" tab):
    - Status filter: MUI `<Select size="small">` with options All/Pending/Ready/Running/Succeeded/Failed/Canceled
    - Search: MUI `<TextField size="small" placeholder="Search by op ID...">`
    - Client-side filtering: filter ops by status + ID substring match
  - Update the stories file:
    - Story "WithDuration": show running (2m 14s), succeeded (342ms), failed (1m 02s), pending (—)
    - Story "FilteredByFailed": only 3 failed ops visible
    - Story "SortedByCreatedDesc": default sort
- [ ] ⬜ Verify: click column headers, type in search, change status filter — all work

### 3.4 Runtime Events Tab in Workflow Detail 🟡

- [ ] ⬜ Edit the "Runtime Events" tab content in `WorkflowDetailPage.tsx`
  - Render `<RuntimeEventTable showFilters showPagination={false} />` (reused from Phase 2)
  - Pre-filtered to `serverFilters={{ workflowId, limit: 200 }}`
  - Add severity + source `<MultiSelectChipFilter>` at top of the tab
  - Add a `<TimeRangeSelector>` for time range filtering within this workflow
  - Show event count in tab label: `"Runtime Events (${events.length})"`
- [ ] ⬜ Verify: switch to Events tab, apply filters, confirm table updates

### 3.5 Create WorkflowArtifactTable 🟡

- [ ] ⬜ Create `web/src/components/workflows/WorkflowArtifactTable.tsx`
  - Props: `workflowId: string`, `ops: WorkflowOp[]`
  - For each op that has completed, fetch artifacts via `useGetOpArtifactsQuery`
    - If there's a batch endpoint `GET /api/v1/workflows/:id/artifacts`, use that instead (check API)
    - If not, use `Promise.all` on individual op artifact queries, or RTK Query batch
  - Columns: **Name** (artifact.name), **Op** (truncated op ID, clickable → selects that op in drawer), **Content Type** (truncated), **Size** (if available), **Actions** (2 icon buttons)
  - Actions:
    - Download icon: `<a href="/api/v1/artifacts/${id}" download>`
    - Preview icon: toggles inline `<ArtifactPreview>` below the row
  - Empty state: "No artifacts produced yet"
- [ ] ⬜ Create `web/src/components/workflows/WorkflowArtifactTable.stories.tsx`
  - Story "WithArtifacts": 5 artifacts from 3 different ops
  - Story "Empty": no artifacts
  - Story "Loading": skeleton rows
- [ ] ⬜ Wire into the "Artifacts" tab in `WorkflowDetailPage.tsx`

---

## Phase 4: Overview & Queue Pages (Day 11–14)

_Make overview interactive, fix queue page._

### 4.1 Make StatCard Clickable 🔴

- [ ] ⬜ Edit `web/src/components/overview/StatCard.tsx`
  - Add optional props: `onClick?: () => void`, `href?: string`
  - Wrap `<Card>` in `<CardActionArea>` if `onClick` or `href` is provided
  - Add hover elevation effect (`sx={{ '&:hover': { elevation: 2 } }}` when clickable)
  - Add visual hint: a small arrow icon in the top-right corner when clickable
  - If `href` provided, render as `<Link>` wrapping the Card
  - Keep existing loading skeleton behavior unchanged
- [ ] ⬜ Edit `web/src/components/overview/StatCard.stories.tsx`
  - Add story "Clickable": card with `onClick={() => alert('clicked')}`
  - Add story "Link": card with `href="/workflows"`
  - Verify existing stories still pass (loading, withBreakdown)
- [ ] ⬜ Edit `web/src/components/overview/StatCardRow.tsx`
  - Pass navigation props to each StatCard:
    - Workflows card → `onClick={() => navigate('/workflows')}`
    - Ops card → `onClick={() => navigate('/workflows')}` (best we have for ops)
    - Queues card → `onClick={() => navigate('/queues')}`
- [ ] ⬜ Edit `web/src/components/overview/OpStatusBreakdown.tsx`
  - Add optional prop: `onSegmentClick?: (status: string) => void`
  - Each status bar/segment is wrapped in a clickable `<Box>` with `cursor: 'pointer'`
  - Hover effect: slightly brighter color
  - When clicked, calls `onSegmentClick(statusLabel)`
- [ ] ⬜ Edit `web/src/components/overview/OpStatusBreakdown.stories.tsx`
  - Add story "Interactive": click "failed" segment → shows alert
- [ ] ⬜ Edit `web/src/pages/EngineOverviewPage.tsx`
  - Wire `OpStatusBreakdown.onSegmentClick`:
    - "failed" → `navigate('/workflows?status=failed')`
    - "running" → `navigate('/workflows?status=running')`
    - others → `navigate('/workflows')`
  - Verify query params are picked up by `WorkflowFilters` on the Workflows page
  - If not: edit `WorkflowsPage.tsx` to read `status` from URL search params on mount

### 4.2 Add AlertBanner to Overview Page 🟡

- [ ] ⬜ Edit `web/src/pages/EngineOverviewPage.tsx`
  - Import `AlertBanner` from Phase 1
  - Read `status.OpCounts.failed` from the engine status query result
  - If `failed > 0`, render:
    ```
    <AlertBanner
      severity="error"
      message={`${failed} ops failed`}
      action={{ label: 'View Failed', onClick: () => navigate('/workflows?status=failed') }}
    />
    ```
  - Place the AlertBanner between `StatCardRow` and the `Grid` with breakdowns
  - If `failed === 0`, render nothing (no banner)
- [ ] ⬜ Verify: with failed ops present, banner shows; with none, no banner

### 4.3 Add RecentActivityFeed to Overview 🟡

- [ ] ⬜ Create `web/src/components/overview/RecentActivityFeed.tsx`
  - Props: none (self-contained, manages its own data)
  - Uses `useRuntimeEventFeed({ serverFilters: { limit: 5 }, stream: true })`
  - Renders a `<Card>` with title "Recent Activity"
  - Content: compact list of 5 events, one line each:
    - Format: `{timestamp}  {severityDot}  {message}` (message truncated to 60 chars)
  - Footer: `<Link component="button" onClick={() => navigate('/events')}>View All Events →</Link>`
  - Loading state: 3 skeleton lines
  - Error state: hidden (don't break the overview page if SSE fails)
- [ ] ⬜ Create `web/src/components/overview/RecentActivityFeed.stories.tsx`
  - Story "WithEvents": 5 mock events with recent timestamps
  - Story "Empty": no events, "No recent activity" message
  - Story "Loading": skeleton lines
- [ ] ⬜ Edit `web/src/pages/EngineOverviewPage.tsx`
  - Add `<RecentActivityFeed />` below the Grid, spanning full width
- [ ] ⬜ Verify: visit `/`, recent activity feed shows latest events, "View All Events" navigates correctly

### 4.4 Fix Queue Monitor Page 🔴

- [ ] ⬜ Edit `web/src/pages/QueueMonitorPage.tsx` — inline row expansion
  - Remove the separate `Typography` click targets and `Collapse` sections
  - Edit `QueueStatusTable.tsx` to use MUI collapsible table rows pattern:
    - Each data row gets an expand/collapse toggle icon button in the first column
    - Below each data row, a hidden `<TableRow>` with `<Collapse>`:
      - Contains `<QueueDetailPanel>` inline
    - Clicking the toggle icon flips `expandedQueue` state
    - Only one queue expanded at a time (or allow multiple — default to one)
  - Remove the old loop that rendered expand targets below the table
- [ ] ⬜ Edit `web/src/components/queues/QueueStatusTable.tsx`
  - Add expand/collapse column as first column (narrow, 40px)
  - Each row: `<IconButton size="small">{expanded ? <KeyboardArrowUp /> : <KeyboardArrowDown />}</IconButton>`
  - Expanded row: `<TableRow><TableCell colSpan={8}><Collapse in={expanded}><QueueDetailPanel /></Collapse></TableCell></TableRow>`
- [ ] ⬜ Edit `web/src/components/queues/QueueStatusTable.stories.tsx`
  - Add story "WithExpandedRow": 2nd row expanded, showing detail panel
  - Add story "NoExpansion": `expandable={false}` (if we make it optional)
- [ ] ⬜ Add throughput time range selector
  - Edit `QueueMonitorPage.tsx`: add chip group `[15m] [1h] [6h] [24h]` above `<ThroughputChart>`
  - Local state: `timeRange`, passed to chart (even though data is still placeholder)
- [ ] ⬜ Add placeholder data notice
  - Above the `<ThroughputChart>`, add:
    ```
    <Alert severity="info" sx={{ mb: 2 }}>
      Throughput data is currently simulated. Awaiting backend metrics endpoint.
    </Alert>
  - Add TODO comment: `// TODO: replace placeholderThroughput with real API data when GET /api/v1/queues/:key/metrics exists`
- [ ] ⬜ Create backend ticket: "Add `GET /api/v1/queues/:key/metrics?range=15m` endpoint for throughput data"

---

## Phase 5: Sites & Submit Pages (Day 14–16)

_Fix API anti-patterns and add feedback loops._

### 5.1 Fix Sites List N+1 API Problem 🔴

- [ ] ⬜ Edit `web/src/api/catalogApi.ts`
  - Check if `useListSitesQuery` already returns enough data for cards
  - If the list endpoint only returns `{ name }[]`, check if there's an option to include summary data
  - If not, consider: `useGetSiteDetailQuery` with a batch approach, or accept N+1 for now
  - **Preferred approach:** Create a new RTK Query endpoint `useListSiteSummariesQuery` that returns `{ name, verbCount, scriptCount, hasSubmitVerbs, databaseFileName }[]` (requires backend support)
  - **Fallback approach:** Use the existing list endpoint and make cards show less data (no queue policies on the card — move to detail page only)
- [ ] ⬜ Edit `web/src/pages/SitesListPage.tsx`
  - Remove `SiteCardWithDetail` wrapper component that calls `useGetSiteDetailQuery` per card
  - Render `<SiteCard site={site} onClick={...} />` directly using list data only
  - If cards need more data, use `useListSiteSummariesQuery` (from step above)
- [ ] ⬜ Edit `web/src/components/sites/SiteCard.tsx`
  - Ensure it works with summary data only (no queue policies needed on the card)
  - Show: site name, verb count, script count, has-submit badge
  - Remove any dependency on full site detail for the card view
- [ ] ⬜ Verify: visit `/sites`, no N+1 network requests in DevTools Network tab

### 5.2 Make Site Cards Link to Submit 🟡

- [ ] ⬜ Edit `web/src/components/sites/SiteCard.tsx`
  - Add a "Submit Workflow" button at the bottom of each card
  - Only show if `site.hasSubmitVerbs === true`
  - Button navigates to `/submit?site=${site.name}`
- [ ] ⬜ Edit `web/src/pages/SubmitWorkflowPage.tsx`
  - Read `site` from URL search params on mount: `new URLSearchParams(location.search).get('site')`
  - If present, pre-fill the SitePicker with that value
- [ ] ⬜ Verify: click "Submit Workflow" on a site card → lands on submit page with site pre-selected

### 5.3 Add Submission Feedback 🔴

- [ ] ⬜ Edit `web/src/pages/SubmitWorkflowPage.tsx`
  - In the mutation success handler:
    ```
    const result = await submitMutation(params).unwrap()
    showToast(`Workflow submitted: ${result.workflowId}`, 'success')
    navigate(`/workflows/${result.workflowId}`)
    ```
  - This uses the ToastProvider from Phase 0 and auto-navigates to the new workflow
  - On mutation failure: `showToast(error.message || 'Submission failed', 'error')` (stay on page)
- [ ] ⬜ Edit `web/src/components/submit/RecentSubmissionsTable.tsx`
  - Add `pollingInterval: 5000` to the query that fetches recent submissions (if not already)
  - Show status updates in real-time as workflows progress
  - Add clickable workflow IDs that navigate to `/workflows/:id`
- [ ] ⬜ Verify: submit a workflow → toast appears → auto-navigates to workflow detail

### 5.4 Dynamic Site List in Workflow Filters 🟡

- [ ] ⬜ Edit `web/src/pages/WorkflowsPage.tsx`
  - Remove hardcoded `const sites = ['hackernews', 'slashdot', 'js-demo', 'nereval']`
  - Replace with: `const { data: siteData } = useListSitesQuery()` → extract `siteData.map(s => s.name)`
  - Pass the dynamic list to `<WorkflowFilters sites={dynamicSites} />`
- [ ] ⬜ Verify: add a new site to the catalog → it appears in the Workflows filter dropdown

---

## Phase 6: Polish (Day 16–18)

_Final quality improvements — no functional changes._

### 6.1 Dark Mode Toggle 🟠

- [ ] ⬜ Edit `web/src/App.tsx` (or theme file)
  - Create two MUI themes: `lightTheme` and `darkTheme`
  - Add state: `const [mode, setMode] = useState<'light' | 'dark'>('light')`
  - Use `useMediaQuery('(prefers-color-scheme: dark)')` to detect system preference as default
  - Wrap app in `<ThemeProvider theme={mode === 'dark' ? darkTheme : lightTheme}>`
  - Persist preference to `localStorage` key `'theme-mode'`
- [ ] ⬜ Edit `web/src/components/layout/AppShell.tsx`
  - Add `<IconButton>` at the far-right of the Toolbar (before Tabs):
    - Icon: `Brightness4` (moon) when in light mode, `Brightness7` (sun) when in dark mode
    - `onClick`: toggles `mode` between 'light' and 'dark'
    - `aria-label="Toggle dark mode"`
- [ ] ⬜ Verify: toggle works, persists across page reload, respects system preference

### 6.2 Keyboard Shortcuts 🟠

- [ ] ⬜ Create `web/src/hooks/useGlobalShortcuts.ts`
  - Listens for keydown events on `document`
  - Ignore shortcuts when focus is inside `<input>`, `<textarea>`, or `[contenteditable]`
  - Bindings:
    - `r` → `dispatch(api.util.invalidateTags([]))` or refetch current page query (via RTK Query `utils.invalidateTags`)
    - `e` → `navigate('/events')`
    - `w` → `navigate('/workflows')`
    - `/` → focus the search field if one exists on the current page
    - `Escape` → close any open drawer (dispatch to Redux, or emit a custom event)
  - Return empty cleanup function
- [ ] ⬜ Wire into `AppShell.tsx`: call `useGlobalShortcuts()` at the top of the component
- [ ] ⬜ Add a small "?" help tooltip somewhere (e.g., bottom-left corner or in AppBar)
  - Shows a popover with the keyboard shortcut list
- [ ] ⬜ Verify: press `e` from any page → navigates to Events; press `Escape` with drawer open → drawer closes

### 6.3 Accessibility Audit 🟡

- [ ] ⬜ Run `pnpm lint` or manual audit on all interactive elements
  - Every clickable `<TableRow>` needs `role="button"` + `tabIndex={0}` + `onKeyDown` for Enter/Space
  - Every `<IconButton>` needs `aria-label`
  - Color-only indicators: ensure StatusChip always has visible text label (it does currently — good)
  - Drawer open/close: manage focus — when drawer opens, focus first interactive element; when it closes, return focus to the trigger
  - Breadcrumbs: verify screen reader reads them correctly
- [ ] ⬜ Test with keyboard-only navigation
  - Tab through the entire RuntimeEventsPage → every filter and table row is reachable
  - Tab through WorkflowDetailPage tabs → arrow keys switch tabs
  - OpDetailDrawer: tab order starts at the drawer, stays trapped inside while open

### 6.4 Cleanup Dead Code 🟡

- [ ] ⬜ Delete `web/src/components/workflows/RuntimeEventList.tsx` — fully replaced by RuntimeEventTable
- [ ] ⬜ Delete `web/src/components/workflows/RuntimeEventList.stories.tsx` (if it exists)
- [ ] ⬜ Search entire codebase for `RuntimeEventList` imports — verify zero remaining references
- [ ] ⬜ Remove the old `placeholderThroughput` constant from `QueueMonitorPage.tsx` (replace with a TODO comment pointing to the backend ticket)
- [ ] ⬜ Remove any unused `import` statements introduced during the refactoring
- [ ] ⬜ Run `pnpm build` — confirm zero TypeScript errors
- [ ] ⬜ Run `pnpm test` — confirm all tests pass (update test fixtures if needed)
- [ ] ⬜ Run `pnpm storybook` — confirm all stories load without errors

### 6.5 Update Storybook Stories for Existing Components 🟠

- [ ] ⬜ Edit `web/src/components/overview/StatCard.stories.tsx` — add "Clickable" story
- [ ] ⬜ Edit `web/src/components/overview/OpStatusBreakdown.stories.tsx` — add "Interactive" story with `onSegmentClick`
- [ ] ⬜ Edit `web/src/components/overview/QueueHealthPreview.stories.tsx` — add "Clickable" story with `onRowClick`
- [ ] ⬜ Edit `web/src/components/queues/QueueStatusTable.stories.tsx` — add "ExpandedRow" story
- [ ] ⬜ Edit `web/src/components/workflows/WorkflowTable.stories.tsx` — add "Sortable" story
- [ ] ⬜ Verify all stories render correctly in Storybook

---

## Summary

| Phase | Tasks | Est. Days | Dependencies |
|-------|-------|-----------|-------------|
| 0 — Foundation | 3 groups (14 sub-tasks) | 1–2 | None |
| 1 — Shared Components | 3 components (10 sub-tasks) | 3 | Phase 0 |
| 2 — RuntimeEventTable | 6 groups (17 sub-tasks) | 3 | Phase 1 |
| 3 — Workflow Detail | 5 groups (13 sub-tasks) | 3 | Phase 2 |
| 4 — Overview & Queue | 4 groups (12 sub-tasks) | 3 | Phase 1, 2 |
| 5 — Sites & Submit | 4 groups (10 sub-tasks) | 2 | Phase 0 |
| 6 — Polish | 5 groups (14 sub-tasks) | 2 | All phases |
| **Total** | **30 groups (90 sub-tasks)** | **~18 days** | |
