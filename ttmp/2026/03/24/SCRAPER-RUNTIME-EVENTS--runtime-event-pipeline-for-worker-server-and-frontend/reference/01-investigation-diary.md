---
Title: Investigation diary
Ticket: SCRAPER-RUNTIME-EVENTS
Status: active
Topics:
    - architecture
    - scraper
    - worker
    - server
    - http-api
    - scheduler
    - api
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: buf.gen.yaml
      Note: Records the final js+dts generation choice after the frontend build issue
    - Path: pkg/api/server/server.go
      Note: Shows that the server has no live worker event transport today
    - Path: pkg/cmd/worker.go
      Note: Shows that worker execution is a separate process today
    - Path: pkg/engine/scheduler/scheduler.go
      Note: Primary evidence for current event types and observer behavior
    - Path: pkg/runtimeevents/codec_test.go
      Note: Round-trip validation for binary and protojson event encoding
    - Path: pkg/sites/submitverbs/host.go
      Note: Shows the submission-side workflow creation path relevant to event emission
    - Path: proto/scraper/runtime/v1/events.proto
      Note: Primary implementation artifact for the first schema slice
    - Path: web/package.json
      Note: Shows the added protobuf runtime dependency in the frontend
ExternalSources: []
Summary: Chronological diary of the research and documentation work for the runtime-events ticket, including the current architecture evidence, the Watermill decision, the protobuf schema decision, and the first implementation slice.
LastUpdated: 2026-03-24T20:51:19-04:00
WhatFor: Record how the runtime-event architecture was inspected, which alternatives were compared, what failed during investigation, and why the ticket recommendation, Watermill decision, protobuf schema decision, and first implementation slice were chosen.
WhenToUse: Use when reviewing the reasoning behind the runtime-events ticket, resuming the work later, or validating that the proposed direction still matches the codebase.
---



# Investigation diary

## Goal

Create a dedicated ticket for runtime events, document the real worker/server seams in `scraper/`, compare the main transport/topology choices, and leave behind a diary that makes the reasoning easy to continue later.

## Context

The current scraper runtime already separates durable execution from HTTP inspection and submission. That is a good base for correctness, but it means live runtime signals are fragmented: scheduler events stay inside the worker, request logs stay inside the server, and the frontend only sees persisted snapshots via polling-oriented APIs.

## Step 1: Inspect the current event seams and prior ticket context

I started by checking whether scraper already had a reusable event boundary or whether this ticket would need a new architecture from scratch. The answer was mixed: the scheduler already emits useful operational events, but the main commands wire them nowhere, and the HTTP server is built as a separate process with no event transport at all.

That made the initial question much narrower and more concrete. The project does not need a brand new workflow model. It needs a way to take the existing scheduler and request signals, normalize them, and move them between the existing worker and server roles.

### Prompt Context

**User prompt (verbatim):** "Analyze scraper/ and create a new docmgr ticket:

We want to add events emitting to the worker and server so that we can collect log files and other realtime data and also show it in the frontend.

I want to use redis for the transfer (and so setup a docker composer for the seetup).

Maybe it is easier to merge the worker and the server through and use watermill for pubsub internally at first? I'm not sure. So before writing the implementation document, give me an overview of my choices and pros and cons. 

Keep a diary as you work."

**Assistant interpretation:** Inspect the existing scraper architecture, create a new ticketed design workspace, compare Redis-backed transport against temporary process merging, and write the recommendation before any implementation plan.

**Inferred user intent:** The user wants a low-regret event architecture that can support real-time frontend visibility and log collection without drifting into an unnecessary redesign.

### What I did

- inspected `pkg/cmd/worker.go`
- inspected `pkg/api/server/server.go`
- inspected `pkg/engine/scheduler/scheduler.go`
- inspected `pkg/services/submission/service.go`
- inspected `pkg/sites/submitverbs/host.go`
- inspected the earlier dashboard and HTTP API ticket documents for prior assumptions about event transport

Commands used:

```bash
sed -n '1,240p' scraper/pkg/cmd/worker.go
sed -n '1,320p' scraper/pkg/api/server/server.go
sed -n '1,620p' scraper/pkg/engine/scheduler/scheduler.go
sed -n '1,260p' scraper/pkg/services/submission/service.go
sed -n '1,260p' scraper/pkg/sites/submitverbs/host.go
rg -n "events|WebSocket|polling|ring buffer|ephemeral" scraper/ttmp/2026/03/23/SCRAPER-DASHBOARD--dashboard-ui-workflow-monitoring-engine-health-rate-limiting-operations -g '*.md'
```

### Why

- The worker/server topology already exists and should drive the recommendation.
- The earlier dashboard ticket already documented some event-related assumptions and open questions.
- The submission path matters because workflow creation is part of the event story, not just worker execution.

### What worked

- `pkg/engine/scheduler/scheduler.go` made the current event seam explicit through `Event`, `Observer`, and `emit(...)`.
- The dashboard docs clearly showed that polling was chosen earlier because the repo had no event bus.
- The code inspection confirmed that the current architecture intentionally separates worker execution from HTTP serving.

### What didn't work

- I initially tried to open a specific HTTP API design doc path that does not exist:

```text
sed: can't read scraper/ttmp/2026/03/23/SCRAPER-HTTP-API--http-api-for-durable-scraper-engine/design-doc/01-http-api-architecture-and-contract.md: No such file or directory
```

- That was useful anyway because it forced me to rely on the current code and the available diary/dashboard documents instead of assuming the missing design doc existed.

### What I learned

- Scraper already has scheduler-level event semantics, but they are currently local and ephemeral.
- Submission also creates workflow-related events conceptually, but the current host path uses `observer == nil`.
- The real gap is event transport and retention, not core workflow execution.

### What was tricky to build

- The tricky part of the analysis was separating transport choices from topology choices. Redis versus Watermill is not the whole decision. The more important question is whether scraper should keep its separate worker and server roles or collapse them into one process.

### What warrants a second pair of eyes

- Whether the current submission path should emit its own canonical events directly or route everything through a shared scheduler observer hook.
- Whether earlier dashboard assumptions about polling should remain as fallback behavior after streaming is added.

### What should be done in the future

- Define the canonical runtime event envelope before selecting a message framework.
- Decide early whether replay and retention are required for all events or only for a subset.

### Code review instructions

- Start in `pkg/engine/scheduler/scheduler.go` to understand the existing event model.
- Then read `pkg/cmd/worker.go` and `pkg/api/server/server.go` to see the process split.
- Finally compare the design recommendation in this ticket against the dashboard notes.

### Technical details

- The worker builds the scheduler with `observer == nil`.
- The submission host also builds a scheduler with `observer == nil` for workflow creation.
- The HTTP server exposes polling-friendly read APIs only.

## Step 2: Create the ticket and write the option matrix

Once the process seams were clear, I created a new ticket and wrote the design guide around concrete options instead of abstract preferences. The main goal was to answer the user’s uncertainty directly: Redis transport and merged server+worker mode are both viable, but they solve different problems and should not be treated as the same decision.

I ended up recommending a hybrid path. Keep the worker/server split as the default architecture, define a small transport abstraction, and use Redis as the main cross-process event bridge. If a simplified local mode is helpful, add it as an optional `server+worker` mode without forcing the whole system into that shape permanently.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Turn the inspection results into a ticketed options analysis with a concrete recommendation and future implementation plan.

**Inferred user intent:** The user wants the implementation document to start from a justified architecture choice rather than a vague brainstorm.

### What I did

- created ticket `SCRAPER-RUNTIME-EVENTS`
- added a design doc
- added this diary
- replaced placeholder ticket files with a real overview, task list, changelog, and recommendation
- documented four option families and a phased plan

Commands used:

```bash
docmgr ticket create-ticket --ticket SCRAPER-RUNTIME-EVENTS --title "Runtime event pipeline for worker, server, and frontend" --topics architecture,scraper,worker,server,http-api,scheduler,api
docmgr doc add --ticket SCRAPER-RUNTIME-EVENTS --doc-type design-doc --title "Event transport options and implementation plan for worker, server, and frontend"
docmgr doc add --ticket SCRAPER-RUNTIME-EVENTS --doc-type reference --title "Investigation diary"
```

### Why

- A ticketed document makes the recommendation durable and reviewable.
- The option matrix makes it easier to distinguish production architecture from local development shortcuts.
- The phased plan keeps Redis, SSE, and optional single-process mode from getting conflated.

### What worked

- The new ticket structure fit naturally with the earlier scraper tickets.
- The current code made the recommendation straightforward once the seams were explicit.
- The hybrid recommendation resolved the user’s uncertainty without pretending there is only one valid path.

### What didn't work

- N/A in this step. The ticket scaffold and document creation commands succeeded without adjustment.

### What I learned

- Watermill is not the first decision to make here. The event envelope and topology matter more than the message framework.
- Redis is most defensible when treated as the cross-process transport, not as a reason to redesign the worker/server split.
- A temporary merged mode is helpful only if it stays optional.

### What was tricky to build

- The sharp edge in the write-up was keeping the recommendation practical. A purely production-first answer would ignore local ergonomics, while a purely local-first answer would encourage a permanent merged process that the rest of the codebase does not naturally support.

### What warrants a second pair of eyes

- Whether Redis Streams are the right default versus a smaller bespoke transport.
- Whether large log payloads should be referenced indirectly instead of flowing inline through the event stream.

### What should be done in the future

- Turn the design recommendation into a detailed implementation guide only after the open questions around envelope shape and retention are answered.
- Add file-level design notes once the first event package and transport interfaces exist.

### Code review instructions

- Read the ticket index and design doc first.
- Compare the recommendation against the current command boundaries in the worker and API server.
- Validate the ticket workspace with `docmgr doctor --ticket SCRAPER-RUNTIME-EVENTS --stale-after 30`.

### Technical details

- At this point in the work, the recommended default was separate worker and server, Redis transport, and SSE to the frontend.
- At this point in the work, the recommended optional dev mode was in-memory transport plus local `server+worker` execution.
- At this point in the work, Watermill was still deferred pending a later decision.

## Step 3: Record the Watermill decision and tighten the plan

The next user message removed the last framework ambiguity: Watermill should be the standardized way we handle the event pipeline. That did not change the topology recommendation. It changed the implementation stance. Instead of treating Watermill as optional, the ticket now treats Watermill as the standard messaging layer while still keeping the worker/server split and Redis-backed cross-process delivery.

I kept one important architectural boundary in place while updating the docs. Scraper still needs its own `RuntimeEvent` model. Watermill should carry scraper events, not become the business-level event schema itself. That keeps the frontend contract and the log/artifact policy owned by scraper rather than leaked from the transport library.

### Prompt Context

**User prompt (verbatim):** "Let's use watermill as an easier way and standardized fashion of doing the event stuff."

**Assistant interpretation:** Accept Watermill as the chosen eventing framework and update the ticket so the implementation plan now assumes Watermill instead of a bespoke abstraction.

**Inferred user intent:** The user wants the design to stop hedging on the framework choice and move toward a more standardized implementation path.

### What I did

- updated the ticket index to mark Watermill as the chosen eventing layer
- updated the design doc to switch from "Watermill optional" to "Watermill standard"
- updated the task list to include Watermill-specific work items
- updated the changelog with the new decision
- checked official Watermill docs to verify the availability of Redis-backed and in-process backend patterns before writing that change down

### Why

- The user made the framework choice explicit.
- Once Watermill is chosen, the ticket should stop presenting it as an unresolved question.
- The remaining design question is now how thin the scraper wrapper should be, not whether a message framework is needed at all.

### What worked

- The earlier topology recommendation still held up. Watermill fits the separate worker/server topology just as well as it fits a local merged mode.
- The choice made the implementation plan clearer because the local and distributed backends can now share one messaging model.

### What didn't work

- N/A in this step. The change was a design clarification, not a debugging session.

### What I learned

- Watermill is a good standardization choice here only if scraper still owns the event envelope.
- The right follow-up question is now "how do we wrap Watermill cleanly," not "should we use Watermill at all."

### What was tricky to build

- The sharp edge was avoiding a full reversal of the earlier analysis. Choosing Watermill does not imply merging worker and server. Those are separate decisions, and the ticket needed to preserve that distinction.

### What warrants a second pair of eyes

- Whether the initial implementation should expose raw Watermill topics in config, or keep topic naming behind a scraper config layer.
- Whether log-heavy event flows will need a stricter split between event metadata and artifact/blob storage from the first version.

### What should be done in the future

- Add a concrete Watermill package and topic layout proposal to the implementation guide.
- Decide the initial message metadata conventions before code lands.

### Code review instructions

- Re-read the design doc executive summary and design decisions.
- Confirm that Watermill is now treated as standard infrastructure, while the worker/server split remains the default topology.
- Validate the updated ticket with `docmgr doctor --ticket SCRAPER-RUNTIME-EVENTS --stale-after 30`.

### Technical details

- Chosen framework: Watermill
- Preferred distributed mode: Watermill with Redis-backed transport
- Preferred local mode: Watermill in-process transport
- Preserved boundary: scraper-owned `RuntimeEvent` envelope

## Step 4: Switch the event contract to protobuf for Go and TypeScript generation

The next user instruction clarified that the shared event model should be generated for both Go and TypeScript, and that the protobuf schema exchange skill should shape the plan. That changes an important detail in the design: the event envelope should not just be a scraper-owned Go struct wrapped around Watermill. It should be a scraper-owned protobuf schema that generates both backend and frontend types from one source of truth.

That fits the architecture well. Watermill still handles messaging. Protobuf now owns the event contract. The HTTP/SSE boundary becomes `protojson`, and the frontend decodes with generated TS schema support instead of hand-maintained interfaces.

### Prompt Context

**User prompt (verbatim):** "use protobuf for go and ts side event generation, see the skill"

**Assistant interpretation:** Update the design so the runtime event contract is schema-first, generated for both Go and TS, and aligned with the protobuf exchange skill.

**Inferred user intent:** The user wants one durable event schema shared across backend and frontend rather than parallel hand-written types.

### What I did

- read the `protobuf-go-ts-schema-exchange` skill and its template and validation references
- checked whether the repo already had `proto/`, `buf.yaml`, `buf.gen.yaml`, or protobuf-related TS runtime dependencies
- confirmed that the repo currently has no proto or Buf setup
- updated the ticket index, design doc, tasks, and changelog to make protobuf the shared contract
- revised the design so Watermill carries generated proto-backed events and the web boundary uses `protojson`

### Why

- One schema is better than mirrored Go and TS types for an event contract that will evolve.
- The protobuf skill already gives the right workflow for Go and TS generation in this kind of setup.
- This reduces drift between backend emission and frontend decoding.

### What worked

- The protobuf skill fit the problem directly: schema-first, Buf-managed generation, `protojson` on the Go side, `fromJson` on the TS side.
- The repo has no existing proto setup, so the design can choose a clean layout instead of adapting around legacy codegen.

### What didn't work

- There was no existing proto or Buf scaffold to extend, so this step could only update the design and task plan, not integrate with existing generation machinery.

### What I learned

- `web/package.json` does not currently include protobuf runtime dependencies, so the frontend side of this plan is still entirely to be added.
- Because the skill warns about `int64` JSON strings and TS `bigint`, the event schema should prefer string identifiers and protobuf timestamp types where possible.
- `google.protobuf.Struct` is available as an escape hatch, but it should stay narrow so the event schema does not become shapeless.

### What was tricky to build

- The main sharp edge was preserving the earlier Watermill decision while changing the event contract. Watermill and protobuf solve different layers. The docs needed to make that separation explicit: Watermill is the bus, protobuf is the shared event schema.

### What warrants a second pair of eyes

- The exact proto package and `go_package` layout, because `paths=source_relative` plus repo structure can get awkward if chosen carelessly.
- Whether Watermill should carry protobuf binary payloads internally, or whether protojson should be used end to end for easier debugging.

### What should be done in the future

- Add a concrete `proto/` tree and Buf config to the implementation guide.
- Decide the initial `RuntimeEventV1` shape, including schema versioning and payload structure.
- Add the TS protobuf runtime dependency and generation output path under `web/`.

### Code review instructions

- Read the design doc executive summary, core design elements, and design decisions again.
- Confirm that protobuf now owns the shared event contract while Watermill remains the transport layer.
- Validate the updated ticket with `docmgr doctor --ticket SCRAPER-RUNTIME-EVENTS --stale-after 30`.

### Technical details

- Shared schema: protobuf
- Go/TS generation toolchain: Buf
- Go HTTP/SSE encoding: `protojson`
- TS decode path: `fromJson` from `@bufbuild/protobuf`

## Step 5: Implement the schema and codec foundation

Once the design was settled, I moved from planning into the first code slice. The goal for this step was not to wire the whole event bus. It was to land the shared contract and the minimum codec surface that later Watermill and HTTP work can build on cleanly. That meant adding the protobuf schema, Buf generation, generated Go/TS outputs, the frontend protobuf runtime dependency, and a small Go helper package with real round-trip tests.

This was also the first step where repo-specific constraints showed up. The initial TypeScript codegen target produced `.ts` files that the frontend’s `erasableSyntaxOnly` setting rejected. I had to adapt the Buf ES plugin target to `js+dts`, regenerate, and validate again. That is exactly the sort of detail that belongs in the diary because it will matter again when future proto files are added.

### Prompt Context

**User prompt (verbatim):** "ok, add detailed list of tasks to the ticket, then work on them one by one."

**Assistant interpretation:** Expand the ticket into a concrete execution plan and immediately start implementing the first task slice from that plan, while honoring the later instruction to commit at appropriate intervals and keep a detailed diary.

**Inferred user intent:** The user wants the runtime-events ticket to stop being a proposal and become an incremental implementation stream with real commits and reproducible checkpoints.

**Commit (code):** `448d450050ae6ea9e0880b44b6f3cf1a176d0db1` — "Add protobuf runtime event schema scaffold"

### What I did

- added `proto/scraper/runtime/v1/events.proto` with `RuntimeEventV1`, enums, timestamp, labels, and structured payload
- added `buf.yaml` and `buf.gen.yaml`
- generated Go artifacts into `gen/proto/scraper/runtime/v1`
- generated web artifacts into `web/src/pb/proto/scraper/runtime/v1`
- added `@bufbuild/protobuf` to `web/package.json`
- added `pkg/runtimeevents/codec.go` and `pkg/runtimeevents/codec_test.go`
- updated the ticket tasks to reflect the completed Phase 1 items

Commands used:

```bash
buf --version
go version
npm --version
buf generate
npm install @bufbuild/protobuf@^2
go get google.golang.org/protobuf@v1.36.11
go test ./pkg/runtimeevents ./gen/... -count=1
go test ./... -count=1
npm run build
git commit -m "Add protobuf runtime event schema scaffold"
```

### Why

- The rest of the event system should not be built on hand-written parallel Go and TS types.
- Watermill integration is easier once the event contract and codec layer already exist.
- A small codec package gives the later transport work one place to normalize schema versioning and JSON/binary handling.

### What worked

- `buf generate` produced working Go and web outputs from one schema.
- The generated Go package built cleanly once the protobuf runtime dependency was added.
- The `pkg/runtimeevents` codec package passed round-trip tests for both binary and `protojson`.
- `go test ./... -count=1` passed after the codec syntax fix.

### What didn't work

- The first generated web target (`target=ts`) did not fit this repo’s frontend TypeScript settings. The build failed with:

```text
src/pb/proto/scraper/runtime/v1/events_pb.ts(117,13): error TS1294: This syntax is not allowed when 'erasableSyntaxOnly' is enabled.
src/pb/proto/scraper/runtime/v1/events_pb.ts(163,13): error TS1294: This syntax is not allowed when 'erasableSyntaxOnly' is enabled.
src/pb/proto/scraper/runtime/v1/events_pb.ts(199,13): error TS1294: This syntax is not allowed when 'erasableSyntaxOnly' is enabled.
```

- I changed the Buf ES plugin target to `js+dts`, regenerated, and that specific codegen problem went away.
- A separate pre-existing frontend build failure remains:

```text
src/components/workflows/OpDetailDrawer.tsx(291,12): error TS2739: Type '{ source: string | null; loading: boolean; error: null; }' is missing the following properties from type 'ScriptTabProps': site, scriptPath
```

- I also hit a small Go syntax mistake in `pkg/runtimeevents/codec.go` around `protojson.UnmarshalOptions`, fixed it, and re-ran the tests.

### What I learned

- This frontend setup wants generated JS plus declarations, not raw TS output from the ES plugin.
- The schema and codec layer can land independently of Watermill and still provide immediate value.
- `protojson` plus generated TS decoding is still viable in this repo, but the exact codegen target matters.

### What was tricky to build

- The tricky part was separating a real problem introduced by the new work from an unrelated existing frontend failure. The generated `.ts` file was our issue and needed to be fixed. The `OpDetailDrawer.tsx` build error was already present and should not block the schema foundation slice from being committed.

### What warrants a second pair of eyes

- The initial `RuntimeEventV1` shape, especially whether the generic `payload` field is too flexible.
- The decision about protobuf binary versus `protojson` inside Watermill messages, which is still open.
- The generated Go package path under `gen/proto/...`, because that choice will become sticky once more proto files exist.

### What should be done in the future

- Add a TS decode example or test using `fromJson`.
- Decide the internal Watermill wire format.
- Start the next slice: topic conventions and a thin Watermill wrapper around the generated event types.

### Code review instructions

- Start with `proto/scraper/runtime/v1/events.proto`.
- Then read `pkg/runtimeevents/codec.go` and `pkg/runtimeevents/codec_test.go`.
- Verify generation outputs in `gen/proto/scraper/runtime/v1` and `web/src/pb/proto/scraper/runtime/v1`.
- Validate with `go test ./... -count=1`.
- Note that `npm run build` still fails, but the remaining error is in `src/components/workflows/OpDetailDrawer.tsx`, not in the generated protobuf artifacts.

### Technical details

- Protobuf package: `scraper.runtime.v1`
- Go package: `github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1`
- Web generation target: `js+dts`
- Repo-specific TS constraint: `erasableSyntaxOnly`

## Quick Reference

| Option | Best for | Main risk |
|--------|----------|-----------|
| SQLite + polling | Quick historical views | weak real-time behavior |
| Merged server+worker | Fast demos and local dev | changes operational model |
| Redis bridge | Production-like multi-process delivery | infra and retention complexity |
| Hybrid abstraction | Lowest-regret evolution | more up-front design discipline |

## Usage Examples

- Read the design guide first when deciding whether to implement Redis immediately or start with an in-memory backend.
- Read Step 1 when validating that the recommendation still matches the actual code.
- Read Step 2 when continuing the ticket and turning the recommendation into executable tasks.

## Related

- `SCRAPER-DASHBOARD` design docs for prior polling assumptions
- `SCRAPER-HTTP-API` diary for the existing server/process split rationale
