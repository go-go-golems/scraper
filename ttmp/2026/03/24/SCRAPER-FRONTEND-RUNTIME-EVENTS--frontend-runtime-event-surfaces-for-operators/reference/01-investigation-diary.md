---
Title: Investigation diary
Ticket: SCRAPER-FRONTEND-RUNTIME-EVENTS
Status: active
Topics:
    - scraper
    - frontend
    - react
    - api
    - events
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: web/src/pages/WorkflowDetailPage.tsx
      Note: First concrete frontend runtime event surface inspected during the investigation
    - Path: pkg/api/server/server.go
      Note: Backend route and SSE bootstrap used to validate the frontend assumptions
ExternalSources: []
Summary: Chronological research log for the frontend runtime event follow-up ticket.
LastUpdated: 2026-04-07T11:35:50-04:00
WhatFor: Preserve the reasoning, commands, evidence, and writing decisions used to produce the frontend runtime event implementation guide.
WhenToUse: Use when continuing this ticket or reviewing why the guide recommends its current phased frontend plan.
---

# Investigation diary

## Step 7: Audit RTK Query invalidation and fix cross-slice cache staleness

The next user report was broader than runtime events. After submission, the queue widget did not update immediately. That pointed to a cache-consistency problem rather than a runtime-event problem, so I paused feature work and made a dedicated pass over every RTK Query slice in `web/src/api/`.

### Prompt Context

**User prompt (verbatim):** "Then, make a big pass on all the rtk-query stuff and make a full matrix in the ticket that examines which queries get invalidated by which other queries: when i submit a  new op, the queue widget for example desnt update."

**Assistant interpretation:** Audit all RTK Query query and mutation endpoints, identify stale-cache paths such as submit not refreshing queue state, fix the code, and document the entire invalidation matrix in the ticket.

**Inferred user intent:** Turn cache invalidation from an ad hoc accident into an explicit frontend design rule so operator pages update immediately and predictably after workflow mutations.

### What I changed

- In `web/src/api/submissionApi.ts`:
  - added `onQueryStarted(...)` for `submitWorkflow`
  - after a successful submit, invalidated:
    - `engineApi` `EngineStatus`
    - `queueApi` `QueueStatus`
    - `runtimeEventsApi` `RuntimeEvents`
    - `workflowApi` `WorkflowList:LIST`, `Workflow:{workflowId}`, and `WorkflowOps:{workflowId}`
- In `web/src/api/workflowApi.ts`:
  - expanded `tagTypes` to include `OpResult` and `OpArtifacts`
  - made `listWorkflows(...)` provide both `WorkflowList:LIST` and per-row `Workflow:{workflowId}` tags
  - made `getOpResult(...)` provide `OpResult:{workflowId}:{opId}`
  - made `getOpArtifacts(...)` provide `OpArtifacts:{workflowId}:{opId}`
  - updated `retryOp(...)` to invalidate:
    - workflow list
    - workflow summary
    - workflow ops
    - op result
    - op artifacts
  - added cross-slice invalidation after retry into engine, queue, and runtime-event queries
  - updated `cancelWorkflow(...)` to invalidate workflow summary and workflow ops more precisely
  - added cross-slice invalidation after cancel into engine, queue, and runtime-event queries
- In the ticket docs:
  - added `design-doc/02-rtk-query-invalidation-matrix-and-cache-consistency-guide.md`
  - updated `index.md` and `tasks.md` so the RTK Query pass is tracked explicitly

### Why

- Scraper uses multiple RTK Query `createApi(...)` slices. Tags are slice-local.
- That means invalidating a `Workflow` tag inside `workflowApi` has no effect on `queueApi` or `engineApi`.
- Before this patch, submit/retry/cancel mutated backend state that affected:
  - workflows,
  - queue readiness,
  - engine aggregate counters,
  - recent runtime event history,
  but only some same-slice queries were being invalidated.
- Polling masked the issue eventually, but it did not provide immediate UI coherence.

### What worked

- `npm run test:unit` passed after the API-slice changes.
- `npm run build` passed after the API-slice changes.
- The resulting invalidation rules are now simple to explain:
  - same-slice refresh uses `invalidatesTags`
  - cross-slice refresh uses `otherApi.util.invalidateTags(...)` inside `onQueryStarted(...)` after `queryFulfilled`

### What didn't work

- There were no pre-existing RTK Query behavior tests around invalidation, so this pass still relies on static reasoning plus build/test validation rather than dedicated store-level integration tests.

### What I learned

- The most important frontend cache rule in this repo is not “remember to add tags.” It is “remember that tags stop at the API-slice boundary.”
- Once the slice boundary is made explicit, the stale queue widget is easy to explain and easy to fix.

### What was tricky to build

- The subtle part was separating same-slice invalidation from cross-slice invalidation.
- It would be easy to overfit on workflow pages and forget that:
  - queue pages,
  - overview widgets,
  - runtime-event history queries,
  all depend on the same backend mutation even though they live in different API slices.
- Another subtle point was detail-level drawer data. Retrying an op can make old result and artifact queries stale unless those endpoints provide their own tags.

### What warrants a second pair of eyes

- Whether scraper should eventually consolidate more of these slices, or keep the current multi-slice structure and just treat cross-slice invalidation as normal practice.
- Whether we should add dedicated RTK Query integration tests with a test store and mocked base queries now that the invalidation graph has become more important.

### What should be done in the future

- Add integration tests for submit/retry/cancel invalidation behavior.
- Decide whether runtime-event history should stay purely invalidation-driven plus SSE, or whether some pages should prefetch on navigation as well.
- Revisit polling intervals once immediate invalidation is reliable, especially on overview and queue pages.

### Code review instructions

- Start with:
  - `web/src/api/submissionApi.ts`
  - `web/src/api/workflowApi.ts`
  - `ttmp/.../design-doc/02-rtk-query-invalidation-matrix-and-cache-consistency-guide.md`
- Validate by:
  - running `npm run test:unit`
  - running `npm run build`
  - submitting a workflow and confirming the queue and overview widgets refresh without waiting for the next polling tick

### Technical details

- The root bug was that `submissionApi` had no invalidation behavior at all even though submission mutates data consumed by `workflowApi`, `queueApi`, `engineApi`, and `runtimeEventsApi`.
- `listWorkflows(...)` now provides both a list sentinel and per-workflow tags so row-level workflow invalidation can reach list consumers.
- `getOpResult(...)` and `getOpArtifacts(...)` now provide tags so retry can invalidate drawer-local detail queries cleanly.

## Step 6: Add a clear button to the global runtime-event console

After the idle event filtering change, the user wanted one more operator affordance on `/events`: a way to clear the current browser-side event buffer without reloading the whole page. That is a UI behavior, not a backend data mutation, so the right implementation point is the shared frontend feed hook.

### Prompt Context

**User prompt (verbatim):** "Add a clear button to the events view."

**Assistant interpretation:** Add a clear action to the global `/events` page that wipes the current displayed event list while preserving the ability to receive new live events.

**Inferred user intent:** Make the operator console usable as a live scratchpad during manual testing instead of forcing a full page refresh to visually reset the stream.

### What I changed

- In `web/src/features/runtime-events/runtimeEventFeed.ts`:
  - added a `clearEvents()` callback that empties the local merged event buffer and resets `lastEventAt`
  - exposed that callback from `useRuntimeEventFeed(...)`
- In `web/src/pages/RuntimeEventsPage.tsx`:
  - added an outlined `Clear` button beside the stream and event-count chips
  - wired the button to `clearEvents()`
  - disabled the button when there are no currently displayed events

### Why

- The runtime-event console is acting partly as an operator dashboard and partly as a debugging tail view.
- In that kind of screen, “clear current view and keep listening” is materially more useful than “reload the entire page and rebuild all state.”
- Keeping the behavior local to the hook avoids inventing any server-side “clear history” semantics, which would have been the wrong abstraction for this request.

### What worked

- `npm run test:unit` passed.
- `npm run build` passed.
- The implementation is small and localized because the shared feed hook already owns the merged in-browser event state.

### What didn't work

- Nothing notable in this slice; the existing helper tests were sufficient for quick regression coverage, and the production build remained green.

### What I learned

- The shared feed abstraction continues to pay off. Small runtime-event UX behaviors like “clear current buffer” can be added once in the hook and reused by additional runtime-event surfaces later if needed.

### What warrants a second pair of eyes

- Whether future pages should share this same clear behavior or whether some surfaces, such as workflow detail pages, should remain history-oriented and omit it.

### What should be done in the future

- If we add submit-page live progress or queue/overview runtime widgets, consider whether each surface should expose:
  - no clear action,
  - a local clear action,
  - or a “pause / resume stream” control alongside clear.

### Code review instructions

- Start with:
  - `web/src/features/runtime-events/runtimeEventFeed.ts`
  - `web/src/pages/RuntimeEventsPage.tsx`
- Validate by:
  - running `npm run test:unit`
  - running `npm run build`
  - opening `/events`
  - confirming `Clear` empties the visible list and that newly arriving SSE events repopulate it

### Technical details

- `clearEvents()` mutates only local React state inside the hook. It does not touch RTK Query cache, backend history, Redis, or SSE subscriptions.
- Because the history merge effect only runs when the query result changes, clearing the local list does not immediately repopulate old history from the current cached query value. New stream events still append normally after the clear.

## Step 5: Drop idle poll events from the published runtime-event stream

The user reviewed the new `/events` console and made a sensible product call: even if empty worker polls are useful for local debugging, they do not belong in the operator-facing runtime event stream. At the frontend level they are pure noise because a healthy but idle worker can generate them continuously.

### Prompt Context

**User prompt (verbatim):** "don't send a no leasable queues event to the rfontend either, that just pollutes the log. Or do you think we need that event?"

**Assistant interpretation:** Remove scheduler idle events from the published runtime-event pipeline unless there is a strong operational reason to keep them.

**Inferred user intent:** Keep the runtime-event UI focused on meaningful workflow and worker state changes instead of per-poll chatter.

### What I changed

- In `pkg/runtimeevents/scheduler.go`:
  - changed `FromSchedulerEvent(...)` to return `nil, nil` for `scheduler.EventIdle`
  - removed idle-specific kind and severity mapping because idle is no longer published
- In `pkg/runtimeevents/scheduler_observer.go`:
  - added a `nil` guard so dropped scheduler events are skipped cleanly before publish
- In `pkg/runtimeevents/scheduler_test.go`:
  - changed the idle test to assert that idle scheduler events are dropped rather than translated into `RuntimeEventV1`

### Why

- A worker idle poll is not a durable business event. It says only that one scheduler cycle found no leaseable work.
- At frontend timescales this creates a misleading, high-volume stream that hides real state transitions such as submit, lease, retry, failure, and success.
- If the product eventually needs liveness or heartbeat semantics, that should be implemented as an explicit low-frequency heartbeat signal, not as one event per empty poll cycle.

### What worked

- `go test ./pkg/runtimeevents ./pkg/engine/scheduler -count=1` passed after the change.
- The publish policy is now aligned with the user-visible logging policy introduced earlier:
  - idle remains available locally as trace logs
  - idle no longer reaches Redis, SSE, or the frontend event list

### What didn't work

- There was no pre-existing observer-level test for dropped scheduler events, so this slice relies on the mapper test plus the observer `nil` guard.

### What I learned

- The right boundary for this decision is the runtime-event mapping layer, not the UI. Filtering it out earlier avoids needless transport, storage, replay, and browser work.

### What warrants a second pair of eyes

- Whether the proto enum should keep the `WORKER_IDLE` value for future compatibility even though the current system no longer emits it. I left the schema unchanged because removing enum values has broader contract implications than simply ceasing to publish them.

### What should be done in the future

- If worker liveness becomes an operator requirement, add a dedicated heartbeat or worker-status channel with coarse sampling and explicit retention rules.

### Code review instructions

- Start with:
  - `pkg/runtimeevents/scheduler.go`
  - `pkg/runtimeevents/scheduler_observer.go`
  - `pkg/runtimeevents/scheduler_test.go`
- Validate by:
  - running `go test ./pkg/runtimeevents ./pkg/engine/scheduler -count=1`
  - confirming the frontend no longer receives `no leaseable queues` runtime events

### Technical details

- Before this patch, `scheduler.EventIdle` mapped to `RUNTIME_EVENT_KIND_WORKER_IDLE` with debug severity and was published through the same Watermill pipeline as real workflow events.
- After this patch, idle events terminate at `FromSchedulerEvent(...)` and are intentionally not represented as `RuntimeEventV1` messages.

## Step 1: Create the frontend follow-up ticket and map the current event surfaces

The backend runtime event ticket closed the transport and server work, but the frontend conversation immediately exposed the next question: what should the UI add now, and in what order? Instead of answering that ad hoc in chat, I created a new ticket so the next wave of frontend work has its own scoped design record. The intent of this ticket is not to write code immediately. The intent is to give a new engineer a precise, low-ambiguity guide.

### Prompt Context

**User prompt (verbatim):** "ok, Create a new ticket and create aa detailed design / implementation document.

Create a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new docmgr ticket for the frontend runtime event follow-up, inspect the current system carefully, then write a long-form design and implementation guide detailed enough for a new engineer to pick up the work safely.

**Inferred user intent:** The user wants a durable handoff artifact, not just a short recommendation list.

### What I did

- created ticket `SCRAPER-FRONTEND-RUNTIME-EVENTS`
- added:
  - a primary design doc
  - an investigation diary
- inspected the current route tree, navigation, workflow detail page, runtime event API client, store wiring, op drawer, submit page, overview page, queue page, backend event handlers, runtime event hub, protobuf schema, and backend integration tests
- wrote a detailed guide that explains:
  - how runtime events currently flow from backend to frontend
  - what surfaces already exist
  - what is missing
  - what to build next
  - which files to edit in each phase
  - which tests to add

Commands used:

```bash
docmgr status --summary-only
docmgr ticket create-ticket --ticket SCRAPER-FRONTEND-RUNTIME-EVENTS --title "Frontend runtime event surfaces for operators" --topics scraper,frontend,react,api,events
docmgr doc add --ticket SCRAPER-FRONTEND-RUNTIME-EVENTS --doc-type design-doc --title "Frontend runtime event surfaces architecture and intern implementation guide"
docmgr doc add --ticket SCRAPER-FRONTEND-RUNTIME-EVENTS --doc-type reference --title "Investigation diary"

nl -ba web/src/App.tsx | sed -n '1,240p'
nl -ba web/src/components/layout/AppShell.tsx | sed -n '1,260p'
nl -ba web/src/pages/WorkflowDetailPage.tsx | sed -n '1,260p'
nl -ba web/src/api/runtimeEventsApi.ts | sed -n '1,220p'
nl -ba web/src/components/workflows/RuntimeEventList.tsx | sed -n '1,260p'
nl -ba web/src/components/workflows/OpDetailDrawer.tsx | sed -n '1,340p'
nl -ba web/src/pages/SubmitWorkflowPage.tsx | sed -n '1,260p'
nl -ba web/src/pages/EngineOverviewPage.tsx | sed -n '1,260p'
nl -ba web/src/pages/QueueMonitorPage.tsx | sed -n '1,260p'
nl -ba pkg/api/server/server.go | sed -n '1,320p'
nl -ba pkg/api/handlers/runtime_events.go | sed -n '1,260p'
nl -ba pkg/runtimeevents/hub.go | sed -n '1,260p'
nl -ba pkg/runtimeevents/backend.go | sed -n '1,300p'
nl -ba pkg/runtimeevents/runner.go | sed -n '1,320p'
nl -ba proto/scraper/runtime/v1/events.proto | sed -n '1,260p'
nl -ba pkg/api/server/server_test.go | sed -n '200,420p'
```

### Why

- The backend runtime event work deserves a frontend-specific follow-up plan instead of continued scope creep in the backend ticket.
- A new engineer needs more than a bullet list. They need:
  - terminology,
  - architecture context,
  - file anchors,
  - implementation sequence,
  - warnings about likely mistakes.

### What worked

- The repository already has enough evidence to write a concrete guide without guessing.
- The current workflow detail page provides a good anchor example because it already exercises:
  - RTK Query history fetches,
  - protobuf JSON decode,
  - EventSource live updates,
  - local dedupe and merge logic.
- The backend integration test provides strong evidence that the API contract is real and stable enough for frontend expansion.

### What didn't work

- `docmgr doctor --ticket SCRAPER-FRONTEND-RUNTIME-EVENTS` currently panics inside the local docmgr installation instead of returning findings.

Observed failure:

```text
panic: runtime error: invalid memory address or nil pointer dereference
github.com/go-go-golems/docmgr/pkg/commands.(*DoctorCommand).RunIntoGlazeProcessor
```

- As a fallback, I validated frontmatter directly on the index, design doc, and diary with:
  - `docmgr validate frontmatter --doc <path> --suggest-fixes`

- There was no failure in the actual architecture investigation. The main product limitation remains that many frontend runtime event concerns are still implicit rather than formalized in shared abstractions.

### Follow-up after docmgr was fixed

- The user later indicated that `docmgr` had been fixed, so I reran the ticket doctor instead of relying on the earlier fallback path.
- The panic was gone, but the doctor still reported unknown topic vocabulary values on the ticket index: `events` and `react`.
- I checked the active vocabulary file with:

```bash
rg -n "slug: (frontend|react|events|backend|websocket|chat)" ttmp/vocabulary.yaml
```

- That confirmed that `frontend` existed but `react` and `events` were missing from the current vocabulary file.
- I added the missing topic entries explicitly:

```bash
docmgr vocab add --category topics --slug react --description "React application architecture and component implementation"
docmgr vocab add --category topics --slug events --description "Runtime events, event streams, and event-driven operator workflows"
```

- I reran the doctor after the vocabulary update:

```bash
docmgr doctor --ticket SCRAPER-FRONTEND-RUNTIME-EVENTS --stale-after 30
```

- This time it passed cleanly:

```text
## Doctor Report (1 findings)

### SCRAPER-FRONTEND-RUNTIME-EVENTS

- ✅ All checks passed
```

- The practical conclusion is that the ticket content itself was already fine. The remaining validation problem was only missing shared vocabulary state.

### What I learned

- The next most valuable frontend addition is not another workflow-local tweak. It is a reusable event feed abstraction plus a global operator console.
- The current code already contains the right seams for a phased implementation:
  - `runtimeEventsApi.ts` for fetches,
  - `WorkflowDetailPage.tsx` for live-stream reference behavior,
  - `OpDetailDrawer.tsx` for op-scoped UI,
  - `SubmitWorkflowPage.tsx` for post-submit live status,
  - overview and queue pages for event-derived widgets.

### What was tricky to build

- The tricky part was writing for an intern rather than for the current maintainers. That meant repeatedly stopping to explain:
  - what a runtime event is,
  - what the server hub is,
  - why the UI currently mixes polling and streaming,
  - where local component state is preferable to Redux.

### What warrants a second pair of eyes

- Whether the team wants the global runtime event console exposed as a top-level tab immediately.
- Whether debug events should be visible by default in operator views.
- Whether the preferred folder layout should use a new `features/runtime-events/` area or stay flatter under existing `api/`, `components/`, and `pages/` directories.

### What should be done in the future

- Turn the design phases into actual implementation work.
- Start by extracting reusable runtime event stream logic out of `WorkflowDetailPage`.
- Then add the global `/events` page before widening context-specific panels.

## Step 2: Implement Phase 1 shared feed logic and the global `/events` page

After the ticket and design work were in place, the next user request was simply to proceed. I treated that as authorization to start the first implementation slice from the ticket: extract the runtime-event streaming logic out of the workflow detail page, make it reusable, add a global operator page, test it, then update the ticket diary and checklist so the docs stayed synchronized with the code.

### What I changed

- Added a new shared frontend runtime-event module in:
  - `web/src/features/runtime-events/runtimeEventFeed.ts`
- Added pure helper tests in:
  - `web/src/features/runtime-events/runtimeEventFeed.test.ts`
- Added a lightweight unit-test config in:
  - `web/vitest.unit.config.ts`
- Refactored:
  - `web/src/pages/WorkflowDetailPage.tsx`
- Added the new global page:
  - `web/src/pages/RuntimeEventsPage.tsx`
- Wired navigation and routing in:
  - `web/src/App.tsx`
  - `web/src/components/layout/AppShell.tsx`
- Expanded the shared renderer in:
  - `web/src/components/workflows/RuntimeEventList.tsx`
- Exported the runtime event query param type from:
  - `web/src/api/runtimeEventsApi.ts`

### Why this implementation order made sense

The workflow detail page already had working history fetch plus SSE code, but it was trapped inside one page component. The global `/events` page needed exactly the same primitives:

- build query parameters,
- fetch recent history,
- open an SSE stream,
- decode protobuf JSON,
- merge and dedupe by event ID,
- sort by descending event time,
- expose a connection state the page can render.

That meant the right first move was not another page-specific hack. The right move was to extract those responsibilities into one shared hook and helper module, then point both pages at it.

### Commands and verification

Commands used during implementation:

```bash
nl -ba web/src/pages/WorkflowDetailPage.tsx | sed -n '1,260p'
nl -ba web/src/api/runtimeEventsApi.ts | sed -n '1,260p'
nl -ba web/src/App.tsx | sed -n '1,260p'
nl -ba web/src/components/layout/AppShell.tsx | sed -n '1,260p'
nl -ba web/src/components/workflows/RuntimeEventList.tsx | sed -n '1,320p'
nl -ba pkg/api/handlers/runtime_events.go | sed -n '1,260p'
cat web/package.json
npm run test:unit
npm run build
```

Verification results:

- `npm run test:unit` passed with 4 helper tests
- `npm run build` passed

### What worked

- The backend API surface was already sufficient for the first operator console. No server changes were needed.
- Pulling the logic into `runtimeEventFeed.ts` simplified `WorkflowDetailPage.tsx` immediately and made the new page straightforward to build.
- The global page could support:
  - server-backed filters for `workflowId`, `opId`, `site`, and `workerId`
  - client-side filters for `severity` and `source`
  - connection state and last-event indicators
  - workflow click-through navigation from the event list

### What failed or needed adjustment

- The first test implementation used plain objects for protobuf timestamps. The generated TypeScript types require actual protobuf message instances.
- I fixed that by creating timestamps with `create(TimestampSchema, ...)` instead of raw object literals.
- I also briefly called `docmgr validate frontmatter` with a path that accidentally duplicated the `ttmp/` segment, which failed with a path-not-found error. The corrected absolute-path invocation worked.

### What I learned

- The shared hook boundary was the correct abstraction level for this phase. It was large enough to remove duplication, but still small enough that the page components remain easy to read.
- The global `/events` page is already useful even before op-scoped tabs or dashboard widgets exist, because it creates an operator entry point that does not depend on first navigating into a workflow.
- The repo did not already have a simple unit-test entry point for pure frontend helpers, so adding `web/vitest.unit.config.ts` was worth doing early.

### What remains after this slice

- `OpDetailDrawer` still needs an op-scoped runtime-event tab.
- `SubmitWorkflowPage` still needs post-submit live progress.
- overview and queue pages still need event-derived widgets.
- reconnect/stale-state UX still needs hardening beyond the initial `connecting/live/error/closed` state model.

## Step 3: Add the op-scoped runtime tab and richer event payload rendering

The next requested step was to keep going, committing at appropriate intervals and keeping the diary current. I treated that as authorization to continue with the next concrete slice in the ticket: use the shared runtime-event feed inside the op drawer, then make the list renderer more informative for actual runtime-event payloads emitted by the backend.

### What I inspected before changing code

I first verified what the drawer looked like today and what payload keys actually exist in emitted runtime events. That avoided building UI around guessed fields.

Files inspected:

- `web/src/components/workflows/OpDetailDrawer.tsx`
- `web/src/components/workflows/RuntimeEventList.tsx`
- `proto/scraper/runtime/v1/events.proto`
- `pkg/runtimeevents/scheduler.go`
- `pkg/runtimeevents/runner.go`
- `pkg/services/submission/service.go`
- `pkg/api/server/server.go`

The key payload facts were:

- scheduler events can emit:
  - `attempt`
  - `workflowStatus`
  - `errorCode`
  - `errorMessage`
  - `retryable`
- runner events can emit:
  - `runnerKind`
  - `durationMillis`
  - `emittedCount`
  - `artifactCount`
  - `recordWriteCount`
  - `artifactSummaries`
  - `error`
  - `errorCode`
  - `retryable`
- request events can emit:
  - `method`
  - `path`
  - `statusCode`
  - `durationMillis`
- submission events can emit:
  - `verb`
  - `submittedCount`
  - `commandPath`
  - `siteDbPath`

### What I changed

- Added a `Runtime` tab to `web/src/components/workflows/OpDetailDrawer.tsx`
- Hooked that tab up to `useRuntimeEventFeed(...)` with:
  - `workflowId`
  - `opId`
  - `limit: 40`
- Scoped live streaming so the drawer only opens the SSE stream while:
  - the drawer is open, and
  - the `Runtime` tab is active
- Added simple stream-state and event-count chips inside the drawer tab
- Expanded `web/src/components/workflows/RuntimeEventList.tsx` so each event can now surface:
  - normalized event kind
  - queue
  - request ID
  - artifact ID
  - retry attempt
  - error code
  - retryability
  - runner kind
  - emitted/record counts
  - HTTP method/path/status
  - workflow status
  - command path and site DB path
- Reset drawer-local UI state when switching to another op:
  - selected artifact
  - active tab

### Commands and verification

Commands used during this slice:

```bash
nl -ba web/src/components/workflows/OpDetailDrawer.tsx | sed -n '1,360p'
nl -ba web/src/components/workflows/RuntimeEventList.tsx | sed -n '1,260p'
nl -ba proto/scraper/runtime/v1/events.proto | sed -n '1,260p'
nl -ba pkg/runtimeevents/scheduler.go | sed -n '1,320p'
nl -ba pkg/runtimeevents/runner.go | sed -n '1,260p'
nl -ba pkg/services/submission/service.go | sed -n '1,180p'
nl -ba pkg/api/server/server.go | sed -n '130,220p'
npm run test:unit
npm run build
```

Verification results:

- `npm run test:unit` passed
- `npm run build` passed

### What failed or needed adjustment

- The first compile pass failed because the drawer code used `spec` before TypeScript was satisfied that it was non-null.
- That happened because I needed optional access for the hook setup before the early return, but later wanted a non-optional `spec`.
- I fixed that by separating:
  - `selectedSpec = op?.op` for hook inputs
  - `const spec = selectedSpec` after the early return guard
- I also missed the `Stack` import in the first patch, which TypeScript caught immediately.

### What worked

- The shared feed abstraction was reusable without any changes to the backend.
- The op drawer became a natural second consumer because it already owns the rest of the "inspect this op" experience.
- Richer payload rendering improved both the new drawer tab and the existing global event console at the same time, because both reuse `RuntimeEventList.tsx`.

### What remains after this slice

- The remaining unchecked item in Phase 3 is a product decision about whether `DEBUG` events should be shown by default in workflow-local and op-local contexts.
- `SubmitWorkflowPage` still needs the post-submit live progress surface.
- overview and queue pages still need event-derived widgets.
- component-level tests and stream-hook tests are still missing.

## Step 4: Fix runtime-event UI warnings and reduce idle worker log noise

This step came from direct manual testing rather than from the original phase checklist. The user reported three concrete problems while exercising the new runtime-event screens: invalid nested MUI typography markup on `/events`, Redux serializability warnings caused by protobuf `bigint` timestamp fields, and worker idle logs flooding the terminal when there was no leaseable work. The goal of this step was not to change architecture again. It was to remove those rough edges without regressing the shared runtime-event design.

The key design correction was on the frontend data boundary. RTK Query had been caching decoded protobuf `RuntimeEventV1` messages. That looked convenient, but the generated timestamp shape contains `bigint` seconds, which Redux Toolkit correctly warns about as non-serializable. The right fix was to keep raw JSON in the RTK cache and decode protobuf messages only in local hook state inside `useRuntimeEventFeed`. That preserves the shared hook design while keeping Redux state serializable.

### Prompt Context

**User prompt (verbatim):** "When going to /events:  content.js:1:838260
In HTML, <p> cannot be a descendant of <p>.
This will cause a hydration error.

  ...
    <ul className=\"MuiList-ro...\">
      <Box>
        <Styled(div) as=\"div\" ref={null} className=\"MuiBox-root\" theme={{...}} sx={{}}>
          <Insertion>
          <div className=\"MuiBox-roo...\">
            <ListItem alignItems=\"flex-start\" sx={{px:0,py:1.25}}>
              <MuiListItem-root as=\"li\" ref={function} ownerState={{...}} className=\"MuiListIte...\" sx={{px:0,py:1.25}}>
                <Insertion>
                <li className=\"MuiListIte...\" ref={function}>
                  <ListItemText primary={<ForwardRef(Grid)>} secondary={<ForwardRef(Grid)>}>
                    <MuiListItemText-root className=\"MuiListIte...\" ref={null} ownerState={{primary:true, ...}}>
                      <Insertion>
                      <div className=\"MuiListIte...\">
                        <Typography>
                        <Typography variant=\"body2\" color=\"textSecondary\" className=\"MuiListIte...\" ref={null} ...>
                          <MuiTypography-root as=\"p\" ref={null} className=\"MuiTypogra...\" ...>
                            <Insertion>
>                           <p
>                             className=\"MuiTypography-root MuiTypography-body2 MuiListItemText-secondary css-1hxk12u-...\"
>                             style={{}}
>                           >
                              ...
                                <div className=\"MuiStack-r...\">
                                  <Typography variant=\"body2\" color=\"text.primary\">
                                    <MuiTypography-root as=\"p\" ref={null} className=\"MuiTypogra...\" sx={{color:\"tex...\"}} ...>
                                      <Insertion>
>                                     <p
>                                       className=\"MuiTypography-root MuiTypography-body2 css-20tgvo-MuiTypography-root\"
>                                       style={{}}
>                                     >
                                  ...
            ...
      ...
 Stack: 
    p unknown:0
<anonymous code>:1:145535
<p> cannot contain a nested <p>.
See this log for the ancestor stack trace. Stack: 
    p unknown:0
<anonymous code>:1:145535
In HTML, <div> cannot be a descendant of <p>.
This will cause a hydration error.

  ...
    <ul className=\"MuiList-ro...\">
      <Box>
        <Styled(div) as=\"div\" ref={null} className=\"MuiBox-root\" theme={{...}} sx={{}}>
          <Insertion>
          <div className=\"MuiBox-roo...\">
            <ListItem alignItems=\"flex-start\" sx={{px:0,py:1.25}}>
              <MuiListItem-root as=\"li\" ref={function} ownerState={{...}} className=\"MuiListIte...\" sx={{px:0,py:1.25}}>
                <Insertion>
                <li className=\"MuiListIte...\" ref={function}>
                  <ListItemText primary={<ForwardRef(Grid)>} secondary={<ForwardRef(Grid)>}>
                    <MuiListItemText-root className=\"MuiListIte...\" ref={null} ownerState={{primary:true, ...}}>
                      <Insertion>
                      <div className=\"MuiListIte...\">
                        <Typography>
                        <Typography variant=\"body2\" color=\"textSecondary\" className=\"MuiListIte...\" ref={null} ...>
                          <MuiTypography-root as=\"p\" ref={null} className=\"MuiTypogra...\" ...>
                            <Insertion>
>                           <p
>                             className=\"MuiTypography-root MuiTypography-body2 MuiListItemText-secondary css-1hxk12u-...\"
>                             style={{}}
>                           >
                              ...
                                <div className=\"MuiStack-r...\">
                                  <Typography>
                                  <Grid direction=\"row\" spacing={1} flexWrap=\"wrap\" useFlexGap={true}>
                                    <MuiStack-root as=\"div\" ownerState={{...}} ref={null} className=\"MuiStack-root\" ...>
                                      <Insertion>
>                                     <div className=\"MuiStack-root css-pex525-MuiStack-root\">
            ...
      ...
 Stack: 
    div unknown:0
<anonymous code>:1:145535
<p> cannot contain a nested <div>.
See this log for the ancestor stack trace. Stack: 
    p unknown:0
<anonymous code>:1:145535
A non-serializable value was detected in an action, in the path: `payload.0.occurredAt.seconds`. Value: 1775574401n 
Take a look at the logic that dispatched this action:  
Object { type: \"runtimeEventsApi/executeQuery/fulfilled\", payload: (100) […], meta: {…} }
 

(See https://redux.js.org/faq/actions#why-should-type-be-a-string-or-at-least-serializable-why-should-my-action-types-be-constants) 
(To allow non-serializable values see: https://redux-toolkit.js.org/usage/usage-guide#working-with-non-serializable-data) <anonymous code>:1:145535
A non-serializable value was detected in the state, in the path: `runtimeEventsApi.queries.getRecentRuntimeEvents({\"limit\":100}).data.0.occurredAt.seconds`. Value: 1775574401n 
Take a look at the reducer(s) handling this action type: runtimeEventsApi/executeQuery/fulfilled.
(See https://redux.js.org/faq/organizing-state#can-i-put-functions-promises-or-other-non-serializable-items-in-my-store-state)

not sure the SSE works.

It's on http://localhost:5174/events 


Make this logging.Trace 2026-04-07T11:08:51.243324908-04:00 INF no leaseable queues attempt=0 event=idle op_id= queue= site= workflow_id= workflow_status= 

So we don't flood the terminal.

Also get this on /workflows: A non-serializable value was detected in the state, in the path: `runtimeEventsApi.queries.getRecentRuntimeEvents({\"limit\":100}).data.0.occurredAt.seconds`. Value: 1775574401n 
Take a look at the reducer(s) handling this action type: workflowApi/executeQuery/fulfilled.
(See https://redux.js.org/faq/organizing-state#can-i-put-functions-promises-or-other-non-serializable-items-in-my-store-state) <anonymous code>:1:145535
A non-serializable value was detected in the state, in the path: `runtimeEventsApi.queries.getRecentRuntimeEvents({\"limit\":100}).data.0.occurredAt.seconds`. Value: 1775574401n 
Take a look at the reducer(s) handling this action type: workflowApi/executeQuery/pending.
(See https://redux.js.org/faq/organizing-state#can-i-put-functions-promises-or-other-non-serializable-items-in-my-store-state) <anonymous code>:1:145535
A non-serializable value was detected in the state, in the path: `runtimeEventsApi.queries.getRecentRuntimeEvents({\"limit\":100}).data.0.occurredAt.seconds`. Value: 1775574401n 
Take a look at the reducer(s) handling this action type: workflowApi/executeQuery/fulfilled.
(See https://redux.js.org/faq/organizing-state#can-i-put-functions-promises-or-other-non-serializable-items-in-my-store-state) (maybe that's the SSE ?)"

**Assistant interpretation:** Fix the concrete frontend runtime-event regressions the user found during manual testing, verify whether SSE is actually live, and reduce terminal noise from idle worker polling.

**Inferred user intent:** Make the runtime-event screens stable enough for real interactive use instead of leaving them in a technically-working-but-annoying state.

### What I changed

- In `web/src/api/runtimeEventsApi.ts`:
  - stopped transforming API responses directly into decoded protobuf messages inside RTK Query
  - kept raw JSON values in the cache instead
- In `web/src/features/runtime-events/runtimeEventFeed.ts`:
  - moved history decode into the hook layer with a local `decodeRuntimeEvents(...)` helper
  - kept SSE decode local there as well, so protobuf messages stay out of Redux state
- In `web/src/components/workflows/RuntimeEventList.tsx`:
  - enabled `disableTypography` on `ListItemText`
  - wrapped the secondary area in a plain `Box` so MUI no longer renders nested `<p>` or `<div>` inside a paragraph
- In `pkg/engine/scheduler/scheduler.go`:
  - changed `EventIdle` logging from `Info` to `Trace`

### Why

- The Redux warnings were real and stemmed from a design mistake: protobuf messages with `bigint` timestamp fields do not belong in a serializable RTK Query cache.
- The MUI warnings were also real and came from putting stacks and typography blocks into `ListItemText` secondary content without disabling the default typography wrapper.
- The idle worker log spam made normal manual testing noisy and harder to read even though the worker behavior itself was correct.

### What worked

- The browser verification after the patch showed that the running app was clean:
  - `http://localhost:5174/events` loaded with no warnings or errors in the console
  - `http://localhost:5174/workflows` loaded with no warnings or errors in the console
  - `/events` showed `Stream: live` and an updating event count, which confirmed the SSE path was active
- `go test ./... -count=1` passed after the scheduler logging change

### What didn't work

- The first attempt to use the tool shell for repeated `npm run test:unit` and `npm run build` validation behaved strangely and did not stream output back reliably even though the local app and Go test suite were fine.
- Because of that tool-output quirk, I verified the frontend fix primarily through direct browser loading plus console inspection against the running local Vite server at `http://localhost:5174`.
- I also tripped a shell quirk when trying to capture exit codes with a variable named `status` under `zsh`, because `status` is readonly there.

### What I learned

- The cleanest place to draw the frontend protobuf boundary is inside the shared runtime-event hook, not inside RTK Query cache transforms.
- The `/events` screen was not actually failing because SSE was broken. The noisy console warnings were obscuring the real state of the page.
- For MUI `ListItemText`, `disableTypography` is the safer default once the content becomes layout-like rather than plain text.

### What was tricky to build

- The subtle part was distinguishing a transport problem from a state-shape problem. The user reasonably suspected SSE might be broken because the console was full of warnings and the runtime-event state was participating in warnings on unrelated pages such as `/workflows`.
- The actual cause was that Redux Toolkit checks the entire store for serializability on every action. Once the runtime-event cache contained a protobuf `bigint`, unrelated actions started surfacing the same warning.
- The other subtlety was that MUI’s `ListItemText` secondary prop looks like an arbitrary React node slot, but by default it still wraps that content in typography markup. Once that secondary content contains `Stack` or nested `Typography`, invalid HTML follows unless you disable the wrapper.

### What warrants a second pair of eyes

- Whether the frontend should eventually normalize runtime events into an explicit app-level UI model instead of repeatedly decoding protobuf messages in local hooks.
- Whether the global `/events` page should default to hiding `DEBUG` events now that idle logs are trace-level on the backend.
- Whether the runtime-event list has grown enough that virtualization or pagination should be considered soon.

### What should be done in the future

- Add component-level tests around `RuntimeEventList` markup so this typography-wrapper regression does not reappear.
- Add stream-hook tests that verify history replay plus SSE merge behavior with mocked `EventSource`.
- Continue with the next planned UX slice: post-submit live progress on `SubmitWorkflowPage`.

### Code review instructions

- Start with:
  - `web/src/api/runtimeEventsApi.ts`
  - `web/src/features/runtime-events/runtimeEventFeed.ts`
  - `web/src/components/workflows/RuntimeEventList.tsx`
  - `pkg/engine/scheduler/scheduler.go`
- Validate by:
  - running `go test ./... -count=1`
  - loading `http://localhost:5174/events`
  - loading `http://localhost:5174/workflows`
  - confirming the browser console is free of the earlier nested markup and non-serializable warnings
  - confirming `/events` shows `Stream: live`

### Technical details

- The root Redux warning came from storing decoded `RuntimeEventV1` messages in `runtimeEventsApi.queries...data`, where `occurredAt.seconds` is a `bigint`.
- The fix keeps `JsonValue[]` in RTK Query cache and only calls `decodeRuntimeEvent(...)` inside `useRuntimeEventFeed`.
- The root DOM warning came from `ListItemText` secondary rendering an implicit typography wrapper around layout content that already contained `Typography` and `Stack`.

### Code review instructions

- Read the design doc first.
- Cross-check each "current state" claim against the referenced files.
- Confirm that the proposed phases do not require backend contract changes.
- Validate the ticket with `docmgr doctor --ticket SCRAPER-FRONTEND-RUNTIME-EVENTS --stale-after 30`.

### Technical details

- Existing frontend runtime event route count: one workflow-local timeline only
- Existing backend runtime event delivery modes: history endpoint plus SSE
- Proposed first implementation phase: reusable stream and merge abstraction

## Goal

Provide a continuation-friendly record of how the frontend runtime event design guide was produced.

## Context

This diary complements the main design document. The design doc is the artifact a new engineer should follow. The diary is the evidence trail that explains how the analysis was assembled and what file-backed observations drove the recommendations.

## Quick Reference

- Ticket: `SCRAPER-FRONTEND-RUNTIME-EVENTS`
- Primary doc:
  `design-doc/01-frontend-runtime-event-surfaces-architecture-and-intern-implementation-guide.md`
- Validation command:
  `docmgr doctor --ticket SCRAPER-FRONTEND-RUNTIME-EVENTS --stale-after 30`
- Validation fallback:
  `docmgr validate frontmatter --doc <path> --suggest-fixes`

## Usage Examples

- Use this diary when continuing the ticket and wanting the original research sequence.
- Use the design doc when implementing the feature itself.

## Related

- `SCRAPER-RUNTIME-EVENTS` for the backend transport and initial frontend workflow timeline

## 2026-04-07 continuation: local runtime repair and UI regression fixes

### Prompt context

The user continued manual testing against the local runtime-event stack and reported three classes of problems at once:

1. the worker terminal was flooded by `queue_rate_limited` messages after the Hacker News HTTP queue was moved to 1 rps,
2. the `/events` page showed no live data,
3. the frontend again emitted Redux non-serializable warnings for `occurredAt.seconds` and visually labeled `workflow_updated` as `WARN`.

The user then reported a specific workflow URL that appeared to be “in progress” without anything happening.

### Assistant interpretation

This was not one bug. It was a compound failure made of:

- expected scheduler behavior with bad log-level ergonomics,
- a local-process/runtime mismatch,
- and a frontend state-shape regression.

The correct response was to separate those problems instead of trying to solve them all with one speculative patch.

### Investigation sequence

I first inspected the runtime-event frontend and backend paths that were relevant to the symptoms:

- `web/src/api/runtimeEventsApi.ts`
- `web/src/pages/RuntimeEventsPage.tsx`
- `web/src/pages/WorkflowDetailPage.tsx`
- `web/src/components/workflows/OpDetailDrawer.tsx`
- `web/src/components/workflows/SeverityDotIndicator.tsx`
- `pkg/runtimeevents/scheduler.go`
- `pkg/api/handlers/runtime_events.go`
- `pkg/runtimeevents/backend.go`
- `pkg/engine/scheduler/scheduler.go`

Then I inspected live API state and local process state using:

```bash
curl -sS 'http://127.0.0.1:8080/api/v1/workflows/<workflow-id>'
curl -sS 'http://127.0.0.1:8080/api/v1/workflows/<workflow-id>/ops'
curl -sS 'http://127.0.0.1:8080/api/v1/engine/status'
tmux capture-pane -pt scraper-worker
ps -ef | rg 'scraper api serve|scraper worker run'
```

### What I found

#### 1. Queue-rate-limit logging was technically correct but operationally noisy

The scheduler emits `EventQueueRateLimited` when a queue has ready work but cannot lease due to queue policy. After the Hacker News fetch queue was moved to 1 rps, this became expected behavior and therefore terminal spam.

The scheduler log switch in `pkg/engine/scheduler/scheduler.go` still routed that event through the default `info` branch.

#### 2. The `/events` page was empty because the runtime-event backend had been off in one running stack

At one point the live API and worker were both running without `--events-backend redis`, so the runtime-event API returned:

```json
{"events":[]}
```

That was not a frontend rendering bug. It was a process-launch configuration issue.

#### 3. The “workflow is stuck” state was caused by API and worker looking at different SQLite databases

This was the most important finding.

The live process list showed an API started with quoted flag values:

```text
--engine-db "./tmp-scraper/engine.db"
--sites-dir "./tmp-scraper/sites"
```

while the worker was using:

```text
--engine-db ./tmp-scraper/engine.db
--sites-dir ./tmp-scraper/sites
```

Those are not the same paths once embedded quotes are passed through a shell boundary. The quoted values created a literal directory tree named:

```text
\"./tmp-scraper
```

I confirmed that both trees existed:

- normal runtime DB: `./tmp-scraper/engine.db`
- accidental quoted runtime DB: `\"./tmp-scraper/engine.db\"`

That explained the observed behavior exactly:

- the API showed a workflow with one `ready` op and zero active leases,
- the worker kept polling a different database and therefore never leased that op.

#### 4. The Redux warning regression had reappeared because `runtimeEventsApi` had drifted back

The current worktree version of `web/src/api/runtimeEventsApi.ts` was again using:

- `transformResponse` to decode JSON directly into protobuf messages,
- `updateCachedData` to patch decoded protobuf messages into RTK Query state.

That put protobuf `bigint` timestamp fields such as `occurredAt.seconds` back inside Redux state and actions, which recreated the exact non-serializable warnings seen earlier in the day.

#### 5. `workflow_updated` showing as `WARN` was a frontend enum-label bug

The backend mapping in `pkg/runtimeevents/scheduler.go` was still correct:

- `EventWorkflowUpdated` -> `RUNTIME_EVENT_SEVERITY_INFO`

The UI bug came from `web/src/components/workflows/SeverityDotIndicator.tsx`, where the numeric enum mapping was off by one:

```text
0 -> DEBUG
1 -> INFO
2 -> WARN
3 -> ERROR
```

But the generated enum values are actually:

```text
1 -> DEBUG
2 -> INFO
3 -> WARN
4 -> ERROR
```

So real `INFO` events were being painted as `WARN`.

### What I changed

#### Log noise reduction

In `pkg/engine/scheduler/scheduler.go`:

- changed `EventQueueRateLimited` logging from the default `info` branch to the trace-only branch shared with `EventIdle`

This preserves the event and the metrics signal while removing normal operator-terminal spam.

#### Hacker News rate limit

In `pkg/sites/hackernews/site.go`:

- added an explicit queue policy for `site:hackernews:http`
- set `MaxInFlight: 1`
- set token-bucket rate limit to `RatePerSecond: 1`, `Burst: 1`

In `pkg/sites/hackernews/site_test.go`:

- added a focused definition test that locks this policy in

#### Frontend serialization fix

In `web/src/api/runtimeEventsApi.ts`:

- changed the RTK Query endpoint to cache raw `JsonValue[]`
- removed protobuf decode from `transformResponse`
- changed SSE cache patching to keep raw JSON rather than decoded messages

In the component consumers:

- `web/src/pages/RuntimeEventsPage.tsx`
- `web/src/pages/WorkflowDetailPage.tsx`
- `web/src/components/workflows/OpDetailDrawer.tsx`

I moved decode to the component edge using `useMemo`, so Redux state stays serializable while the UI still works with decoded `RuntimeEventV1` values.

#### Severity label fix

In `web/src/components/workflows/SeverityDotIndicator.tsx`:

- corrected the numeric enum mapping so `INFO` is rendered as `INFO` and `WARN` is rendered as `WARN`

### Environment repair work

After confirming the API/worker DB split, I repaired the local runtime directly:

```bash
kill <old worker pids>
pkill -f 'scraper api serve'
rm -rf '\".'
docker compose up -d redis
tmux new-session -d -s scraper-api 'cd ... && go run ./cmd/scraper api serve --address 0.0.0.0:8080 --engine-db ./tmp-scraper/engine.db --sites-dir ./tmp-scraper/sites --events-backend redis'
tmux new-session -d -s scraper-worker 'cd ... && go run ./cmd/scraper worker run --engine-db ./tmp-scraper/engine.db --sites-dir ./tmp-scraper/sites --poll-interval 50ms --metrics-address 0.0.0.0:9091 --events-backend redis --http-proxy http://...'
```

This did two things:

- removed the accidental quoted-path runtime tree,
- ensured both processes used the same engine DB and site DB directory.

### What happened to the previously stuck workflow

The workflow the user linked,

`hackernews-extract-frontpage-1775586346661331397`

was stored in the accidental quoted-path database tree. Once that tree was removed, that workflow no longer existed in the repaired runtime.

This was the correct cleanup choice because the quoted tree was a launch bug, not a legitimate environment we wanted to preserve.

### Verification after repair

I resubmitted a fresh Hacker News extract workflow into the repaired stack:

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v1/sites/hackernews/verbs/extract-frontpage:submit \
  -H 'Content-Type: application/json' \
  -d '{"values":{"base-url":"https://news.ycombinator.com/","max-pages":5}}'
```

New workflow ID:

`hackernews-extract-frontpage-1775586649974859668`

I then verified:

- API health:
  - `curl http://127.0.0.1:8080/healthz` -> `{"ok":true}`
- worker metrics:
  - `scraper_workers_up{worker_id="scraper-worker"} 1`
- worker activity:
  - tmux pane showed `op_leased` and `op_succeeded`
- workflow completion:
  - final workflow status was `succeeded`
  - stats were `Total: 10`, `Succeeded: 10`
- runtime events:
  - `GET /api/v1/runtime-events?workflowId=<new-id>&limit=10` returned scheduler and runner events
  - `WORKFLOW_UPDATED` was present with `severity: INFO`

### Validation performed

Commands run:

```bash
go test ./pkg/engine/scheduler -count=1
go test ./pkg/sites/hackernews -count=1
go test ./pkg/runtimeevents -count=1
cd web && npm run test:unit
```

Results:

- the focused Go tests passed,
- the runtime-event helper/unit tests passed,
- the app-wide frontend build remained blocked by pre-existing Storybook/type issues outside this fix set,
- the repaired runtime stack produced successful live Hacker News runs and runtime-event history.

### Commits made during this continuation

- `4e0cdbe` `Rate limit Hacker News fetch queue`
- `4aa6c53` `Silence queue rate-limit info logs`
- `9c9a7ca` `Fix runtime event UI serialization and severity labels`

### What should happen next

- Keep using unquoted runtime paths when launching API and worker from `tmux`.
- If local launch automation is scripted, make sure it never embeds literal quotes into `--engine-db` and `--sites-dir` flag values.
- If we want truly durable event history rather than recent in-memory plus live stream, that needs separate design work because the current API serves from the in-process hub, not durable storage.
