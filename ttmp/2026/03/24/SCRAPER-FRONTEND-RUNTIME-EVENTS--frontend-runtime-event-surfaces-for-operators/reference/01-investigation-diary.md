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
LastUpdated: 2026-03-24T23:19:54-04:00
WhatFor: Preserve the reasoning, commands, evidence, and writing decisions used to produce the frontend runtime event implementation guide.
WhenToUse: Use when continuing this ticket or reviewing why the guide recommends its current phased frontend plan.
---

# Investigation diary

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
