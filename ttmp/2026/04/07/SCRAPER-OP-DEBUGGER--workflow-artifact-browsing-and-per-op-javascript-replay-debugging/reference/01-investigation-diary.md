---
Title: Investigation diary
Ticket: SCRAPER-OP-DEBUGGER
Status: active
Topics:
    - scraper
    - architecture
    - frontend
    - backend
    - http-api
    - javascript
    - jsverbs
    - onboarding
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Records the code-backed reasoning behind the artifact browser and per-op JS replay debugger design."
LastUpdated: 2026-04-07T15:12:00-04:00
WhatFor: "Continue the debugger ticket with the original evidence trail intact."
WhenToUse: "Use when resuming the ticket or verifying why the design recommends bundle-based op replay."
---

# Investigation diary

## Goal

Capture the current debugging seams in the scraper codebase and explain why the recommended design is:

- better workflow artifact browsing,
- real op-result APIs,
- and a debug-bundle-driven per-op JS replay path.

## Context

The user wants two related capabilities:

1. quickly browse artifacts for a workflow,
2. exercise individual JavaScript files with real intermediate data so they can debug site behavior without rerunning everything blindly.

Those are not separate systems. In scraper, a JS op’s “intermediate data” is spread across:

- workflow input,
- op input,
- dependency results,
- artifacts,
- script source,
- runtime events,
- and sometimes lease/timing context.

Any viable replay/debug system has to gather those pieces into one reproducible unit.

## Quick Reference

### Current useful seams

- Workflow and op inspection already exist:
  - `GET /api/v1/workflows/{workflowID}`
  - `GET /api/v1/workflows/{workflowID}/ops`
- Per-op artifact listing and artifact body download already exist:
  - `GET /api/v1/workflows/{workflowID}/ops/{opID}/artifacts`
  - `GET /api/v1/artifacts/{artifactID}`
- Script source browsing already exists:
  - `GET /api/v1/sites/{site}/scripts/{path...}`
- The JS runtime already constructs a rich execution context in `pkg/js/runtime/executor.go`
- The JS runtime already resolves dependency results through `ctx.dep(...)`

### Gaps found

- There is no confirmed backend handler for `GET /api/v1/workflows/{workflowID}/ops/{opID}/result`, even though the frontend has a `getOpResult` query.
- Artifact browsing is op-local, not workflow-global.
- There is no “debug bundle” endpoint or export command.
- There is no replay path that runs one JS file against persisted intermediate data in a deterministic, read-only way.

### Most important architectural insight

The debugger unit should be an executed op, not an entire workflow.

That keeps the replay surface small, deterministic, and useful.

## Usage Examples

### Example future debugging flow

1. Open a workflow page.
2. Browse all artifacts for the workflow.
3. Pick one op with incorrect output.
4. Request a debug bundle for that op.
5. Replay the exact script with the exact dependency results and selected artifact bodies.
6. Tweak input locally or through the UI replay panel.
7. Compare live run output and replay output.

### Commands and files inspected during this investigation

```bash
sed -n '1,260p' pkg/api/handlers/engine.go
sed -n '1,520p' pkg/services/engineview/service.go
sed -n '1,340p' pkg/engine/runner/js.go
sed -n '1,540p' pkg/js/runtime/executor.go
sed -n '1,320p' web/src/components/workflows/OpDetailDrawer.tsx
sed -n '1,320p' web/src/api/workflowApi.ts
sed -n '1,220p' web/src/components/scripts/ScriptTab.tsx
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8080/api/v1/workflows/<wf>/ops/<op>/result
```

### Example op debug bundle shape

```json
{
  "workflow": {
    "id": "wf-123",
    "site": "hackernews",
    "input": { "base-url": "https://news.ycombinator.com/" }
  },
  "op": {
    "id": "wf-123:frontpage-extract",
    "kind": "js",
    "queue": "site:hackernews:js",
    "input": {
      "baseURL": "https://news.ycombinator.com/",
      "fetchedOpID": "wf-123:frontpage-fetch",
      "maxPages": 5,
      "pageNumber": 1
    },
    "metadata": {
      "script": "extract_frontpage.js"
    }
  },
  "dependencies": [
    {
      "opID": "wf-123:frontpage-fetch",
      "result": {
        "data": null,
        "artifacts": [
          {
            "id": "wf-123:frontpage-fetch:response-body",
            "kind": "http-response-body",
            "bodyText": "<html>...</html>"
          }
        ]
      }
    }
  ],
  "script": {
    "path": "extract_frontpage.js",
    "source": "module.exports = async function(ctx) { ... }"
  }
}
```

## Findings

### 1. Artifact browsing is already partially present

`pkg/api/handlers/engine.go` and `pkg/services/engineview/service.go` already support:

- listing artifacts for one op,
- downloading one artifact body by ID.

The current UI in `web/src/components/workflows/OpDetailDrawer.tsx` uses this to:

- preview execution logs,
- preview text and JSON artifacts,
- show the script source for script-backed ops.

So the debugger project does not start from zero. It starts from an op-centric detail drawer that already exposes some of the necessary ingredients.

### 2. The JS runtime context is already structured enough for replay

`pkg/js/runtime/executor.go` builds the JS `ctx` object from:

- `workflow`
- `op`
- `lease`
- `input`
- `log(...)`
- `dep(...)`
- `emit(...)`
- `writeRecord(...)`
- `writeArtifact(...)`

This is a major advantage. It means replay does not require inventing a second execution model. It requires feeding the existing model from persisted data instead of live scheduler state.

### 3. Dependency replay is already conceptually available

`ctx.dep(...)` already calls `Dependencies.Result(...)`, and `exportOpResult(...)` already serializes dependency results into a JS-friendly structure.

That means the replay system can reuse the current dependency export semantics rather than inventing a new dependency format.

### 4. There is an API inconsistency around op results

The frontend defines `getOpResult` in `web/src/api/workflowApi.ts` against:

`/workflows/{workflowId}/ops/{opId}/result`

But I could not find a matching backend handler or service method, and a direct GET returned `405`.

This is important because op results are central to any debugger or bundle export path. The replay/debugger work should normalize this by adding an explicit supported endpoint rather than depending on a missing or partial route.

### 5. Workflow-global artifact browsing is still missing

Artifacts are currently exposed as:

- list artifacts for one op,
- download one artifact.

That is enough for the current drawer UX but awkward for operator browsing across large workflows. A workflow-global artifact index would make the system much easier to inspect before adding replay itself.

## Related

- [../design/01-workflow-artifact-browser-and-js-op-replay-debugger-guide.md](../design/01-workflow-artifact-browser-and-js-op-replay-debugger-guide.md)
- `pkg/api/handlers/engine.go`
- `pkg/services/engineview/service.go`
- `pkg/engine/runner/js.go`
- `pkg/js/runtime/executor.go`
- `web/src/components/workflows/OpDetailDrawer.tsx`
- `web/src/api/workflowApi.ts`

## Validation and publication

After drafting the ticket documents, I ran:

```bash
docmgr doctor --ticket SCRAPER-OP-DEBUGGER --stale-after 30
```

The ticket passed without findings.

I then prepared the bundle for offline review by uploading the ticket workspace to reMarkable. That publish step is part of the intended workflow for these design tickets because it gives the user an annotated long-form reference outside the repo while preserving the repo-local markdown as the source of truth.
