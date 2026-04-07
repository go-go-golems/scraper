---
Title: RTK Query invalidation matrix and cache consistency guide
Ticket: SCRAPER-FRONTEND-RUNTIME-EVENTS
Status: active
Topics:
    - scraper
    - frontend
    - react
    - api
    - events
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: web/src/api/submissionApi.ts
      Note: Submit mutation that now triggers cross-slice invalidation after successful workflow submission
    - Path: web/src/api/workflowApi.ts
      Note: Workflow queries and retry/cancel mutations with the main workflow-local invalidation logic
    - Path: web/src/api/queueApi.ts
      Note: Queue widget query that was previously left stale after submission
    - Path: web/src/api/engineApi.ts
      Note: Overview status query that must move with workflow mutations
    - Path: web/src/api/runtimeEventsApi.ts
      Note: Runtime event history query that participates in recent-history consistency
    - Path: web/src/pages/SubmitWorkflowPage.tsx
      Note: User-facing submit flow that exposed the missing invalidation behavior
    - Path: web/src/pages/QueueMonitorPage.tsx
      Note: Queue screen that relied only on polling before this pass
    - Path: web/src/pages/EngineOverviewPage.tsx
      Note: Overview page with engine and queue snapshot queries
    - Path: web/src/pages/WorkflowsPage.tsx
      Note: Workflow list page that should reflect new and updated workflows immediately
    - Path: web/src/pages/WorkflowDetailPage.tsx
      Note: Workflow detail page that consumes workflow, ops, op result, artifacts, and runtime events
ExternalSources: []
Summary: Detailed explanation of scraper's RTK Query slice topology, why tags were not propagating across slices, and the endpoint-by-endpoint invalidation matrix used to keep queue, engine, workflow, and runtime-event views coherent.
LastUpdated: 2026-04-07T11:28:44-04:00
WhatFor: Provide a durable reference for how scraper's RTK Query cache should behave after submit, retry, and cancel mutations, including a full query and mutation invalidation matrix.
WhenToUse: Use when adding a new RTK Query endpoint, changing mutation side effects, debugging stale frontend widgets, or reviewing cache-consistency behavior across pages.
---

# RTK Query invalidation matrix and cache consistency guide

## Executive Summary

The frontend had a real cache-consistency bug: after submitting a workflow, the queue and overview widgets could stay stale until their next polling tick. The immediate reason was visible in the API slice layout under `web/src/api/`: the application uses multiple independent RTK Query `createApi(...)` slices, but the mutations only invalidated tags inside their own slice. RTK Query tags are slice-local. Invalidating a `Workflow` tag in `workflowApi` does nothing to `queueApi` or `engineApi`.

That means the stale queue widget was not a rendering problem and not a backend problem. It was a cache-topology problem.

This document has two goals:

1. explain the frontend cache topology clearly enough that a new engineer does not repeat the same mistake,
2. provide a full matrix that shows which queries are provided by which slice and which mutations must invalidate them.

## The Key Rule

RTK Query tags do not cross API slices.

In scraper, these are separate caches:

- `catalogApi`
- `queueApi`
- `engineApi`
- `workflowApi`
- `submissionApi`
- `runtimeEventsApi`

Each one has:

- its own reducer path,
- its own middleware,
- its own tag namespace,
- its own invalidation boundary.

So this does **not** work across slices:

```ts
// Inside submissionApi
invalidatesTags: ['WorkflowList']
```

That would only matter if `submissionApi` itself had queries providing `WorkflowList`. It does not. The query that actually uses `WorkflowList` lives in `workflowApi`.

The practical consequence is:

- for same-slice updates, use normal `invalidatesTags`
- for cross-slice updates, dispatch `otherApi.util.invalidateTags(...)` after successful mutation completion

## Why The Queue Widget Was Stale

The user-observed bug was: submit a new workflow, and the queue widget did not update immediately.

That happened because:

- `SubmitWorkflowPage` calls `useSubmitWorkflowMutation()` from [submissionApi.ts](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/api/submissionApi.ts)
- queue widgets use `useListQueuesQuery()` from [queueApi.ts](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/api/queueApi.ts)
- `submissionApi` previously had no invalidation logic at all
- `queueApi` was only refreshed by polling on pages like [QueueMonitorPage.tsx](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/pages/QueueMonitorPage.tsx) and [EngineOverviewPage.tsx](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/pages/EngineOverviewPage.tsx)

So the system eventually became correct, but not immediately consistent.

## Current Slice Topology

The Redux store is assembled in [store/index.ts](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/store/index.ts). Each API slice is mounted independently. That is the architectural fact that drives the entire invalidation matrix.

```text
Redux Store
├── catalogApi
├── engineApi
├── queueApi
├── workflowApi
├── submissionApi
└── runtimeEventsApi
```

### Consequence

If a mutation changes backend state that affects multiple screens, the mutation must invalidate every affected slice explicitly.

## Query Matrix

This table is the inventory of frontend queries, what they fetch, and which tags they provide.

| Slice | Query endpoint | Consumer surfaces | Provided tags | Notes |
| --- | --- | --- | --- | --- |
| `catalogApi` | `listSites()` | submit page, site pages | `Sites` | Read-only catalog data |
| `catalogApi` | `listVerbs(site)` | submit page | `Verbs:{site}` | Read-only catalog data |
| `catalogApi` | `listScripts(site)` | site pages | `Scripts:{site}` | Read-only catalog data |
| `catalogApi` | `getSiteDetail(site)` | site detail page | `Sites:{site}` | Read-only catalog data |
| `catalogApi` | `getScript(site,path)` | workflow drawer/site detail | `Scripts:{site:path}` | Read-only catalog data |
| `engineApi` | `getEngineStatus()` | overview page | `EngineStatus` | Snapshot of workflow/op/result counts |
| `queueApi` | `listQueues()` | queue monitor, overview queue preview | `QueueStatus` | Snapshot of queue readiness/in-flight state |
| `workflowApi` | `listWorkflows(params)` | workflows page | `WorkflowList:LIST`, `Workflow:{workflowId}` for each listed row | Row-level workflow tags allow item invalidation to reach list rows |
| `workflowApi` | `getWorkflow(workflowId)` | workflow detail page | `Workflow:{workflowId}` | Workflow header and summary stats |
| `workflowApi` | `getWorkflowOps(workflowId)` | workflow detail page | `WorkflowOps:{workflowId}` | Op table for a workflow |
| `workflowApi` | `getOpResult(workflowId,opId)` | op drawer | `OpResult:{workflowId}:{opId}` | Needed for retry-driven detail refresh |
| `workflowApi` | `getOpArtifacts(wfId,opId)` | op drawer | `OpArtifacts:{workflowId}:{opId}` | Needed for retry-driven artifact refresh |
| `runtimeEventsApi` | `getRecentRuntimeEvents(filters)` | workflow timeline, op runtime tab, global `/events` history replay | `RuntimeEvents` | Generic recent-history cache; live stream is separate SSE |

## Mutation Matrix

This table is the important one. It shows which mutations change backend state and which frontend query families must be invalidated as a result.

| Mutation | Backend effect | Same-slice invalidation | Cross-slice invalidation | Why |
| --- | --- | --- | --- | --- |
| `submissionApi.submitWorkflow` | creates or extends a workflow and queues runnable ops | `workflowApi`: `WorkflowList:LIST`, `Workflow:{workflowId}`, `WorkflowOps:{workflowId}` | `engineApi`: `EngineStatus`; `queueApi`: `QueueStatus`; `runtimeEventsApi`: `RuntimeEvents` | Submission changes workflow lists, workflow detail, queue readiness, engine counters, and recent runtime history |
| `workflowApi.retryOp` | changes one op from terminal/failed state back into a retry path | `workflowApi`: `WorkflowList:LIST`, `Workflow:{wfId}`, `WorkflowOps:{wfId}`, `OpResult:{wfId}:{opId}`, `OpArtifacts:{wfId}:{opId}` | `engineApi`: `EngineStatus`; `queueApi`: `QueueStatus`; `runtimeEventsApi`: `RuntimeEvents` | Retry changes op state, workflow state, queue readiness, aggregate counts, and event history |
| `workflowApi.cancelWorkflow` | cancels a workflow and may affect its runnable ops | `workflowApi`: `WorkflowList:LIST`, `Workflow:{wfId}`, `WorkflowOps:{wfId}` | `engineApi`: `EngineStatus`; `queueApi`: `QueueStatus`; `runtimeEventsApi`: `RuntimeEvents` | Cancellation affects workflow summaries, op table, queue availability, aggregate counters, and recent runtime history |

## Before And After

### Before this pass

- `submissionApi.submitWorkflow` had no invalidation behavior
- `workflowApi.retryOp` invalidated only generic `WorkflowOps`
- `workflowApi.cancelWorkflow` invalidated only generic `Workflow` and `WorkflowList`
- `getOpResult` and `getOpArtifacts` had no tags, so retrying an op could leave drawer detail state stale
- queue and overview screens relied on polling as the only refresh path

### After this pass

- submit explicitly invalidates workflow, queue, engine, and runtime-event caches after success
- retry invalidates workflow detail, op result, op artifacts, queue, engine, and runtime events
- cancel invalidates workflow detail plus queue, engine, and runtime events
- workflow list rows now provide per-workflow tags as well as the list sentinel tag

## The Implementation Pattern

### Same-slice invalidation

Use normal `invalidatesTags` when the query and mutation are in the same API slice.

Example:

```ts
retryOp: builder.mutation<void, { wfId: string; opId: string }>({
  query: ({ wfId, opId }) => ({
    url: `/workflows/${wfId}/ops/${opId}:retry`,
    method: 'POST',
  }),
  invalidatesTags: (_result, _error, { wfId, opId }) => [
    { type: 'WorkflowList', id: 'LIST' },
    { type: 'Workflow', id: wfId },
    { type: 'WorkflowOps', id: wfId },
    { type: 'OpResult', id: `${wfId}:${opId}` },
    { type: 'OpArtifacts', id: `${wfId}:${opId}` },
  ],
})
```

### Cross-slice invalidation

Use `onQueryStarted` and dispatch `otherApi.util.invalidateTags(...)` after `queryFulfilled`.

```ts
async onQueryStarted(_arg, { dispatch, queryFulfilled }) {
  try {
    await queryFulfilled;
    dispatch(engineApi.util.invalidateTags(['EngineStatus']));
    dispatch(queueApi.util.invalidateTags(['QueueStatus']));
    dispatch(runtimeEventsApi.util.invalidateTags(['RuntimeEvents']));
  } catch {
    // do nothing on failure
  }
}
```

This pattern matters because it preserves a clean rule:

- no invalidation on failed mutation
- immediate refetch on successful mutation
- polling stays as a backstop, not as the only coherence mechanism

## Page-Level Impact Matrix

This is the user-facing view of the same cache logic.

| Page or widget | Queries used | Must refresh after submit | Must refresh after retry | Must refresh after cancel |
| --- | --- | --- | --- | --- |
| [SubmitWorkflowPage.tsx](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/pages/SubmitWorkflowPage.tsx) | mutation only | yes, via snackbar/recent submissions state | no | no |
| [WorkflowsPage.tsx](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/pages/WorkflowsPage.tsx) | `listWorkflows` | yes | yes | yes |
| [WorkflowDetailPage.tsx](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/pages/WorkflowDetailPage.tsx) | `getWorkflow`, `getWorkflowOps`, `getOpResult`, `getOpArtifacts`, runtime event feed | yes when viewing the submitted workflow | yes | yes |
| [QueueMonitorPage.tsx](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/pages/QueueMonitorPage.tsx) | `listQueues` | yes | yes | yes |
| [EngineOverviewPage.tsx](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/pages/EngineOverviewPage.tsx) | `getEngineStatus`, `listQueues` | yes | yes | yes |
| Global `/events` page | `getRecentRuntimeEvents` + SSE | yes | yes | yes |

## Recommended Rules For Future Endpoints

When adding a new query or mutation, use this checklist.

### For a new query

- Which slice should own it?
- Which UI screens consume it?
- Is the data operational state, catalog state, or recent-event state?
- Which tag should it provide?
- Does it need a list sentinel tag, per-entity tags, or both?

### For a new mutation

- Which backend entities does it change?
- Which pages show those entities?
- Are those queries in the same API slice as the mutation?
- If not, which `otherApi.util.invalidateTags(...)` calls must run after success?
- Does the mutation affect runtime event history or queue/engine aggregates?

## Anti-Patterns To Avoid

### 1. Relying on polling instead of invalidation

Polling is a safety net. It is not an excuse for missing cache invalidation.

If a mutation is user-initiated and the affected screen is open, the UI should usually update immediately.

### 2. Using generic tags only

Generic tags like `WorkflowList` are useful, but they are not enough for detailed invalidation. Lists should usually provide:

- one list sentinel tag
- per-entity tags for each row when possible

That lets mutations target one workflow without assuming the entire list is the only consumer.

### 3. Forgetting detail queries

If a mutation changes op state, it may also change:

- result visibility
- artifact availability
- workflow summary counts
- queue pressure

The drawer-level queries are easy to forget unless they are included in the matrix explicitly.

## Validation Checklist

After changing mutation invalidation behavior:

1. run `npm run test:unit`
2. run `npm run build`
3. submit a workflow and confirm:
   - the workflows page updates without waiting for the next polling tick
   - the queue page updates without waiting for the next polling tick
   - the overview page updates without waiting for the next polling tick
4. retry an op and confirm:
   - workflow status and op table change immediately
   - op drawer result/artifact queries refresh
5. cancel a workflow and confirm:
   - workflow summary changes immediately
   - queue and overview aggregates move immediately

## Short Recommendation

Keep the current multi-slice structure, but treat cross-slice invalidation as a first-class requirement. In scraper, workflow mutations are operational events. They affect more than one page. The code should say that explicitly.
