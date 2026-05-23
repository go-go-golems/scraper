---
Title: Investigation Diary
Ticket: SCRAPER-JS-DEVENV
Status: active
Topics:
    - scraper
    - frontend
    - ui-design
    - javascript
    - developer-experience
    - debugger
    - developer-tools
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological investigation diary for the JS Dev Environment design ticket."
LastUpdated: 2026-04-07T19:00:00-04:00
WhatFor: "Record the investigation process, decisions, and evidence for the JS dev environment design."
WhenToUse: "Use to understand what was investigated, what decisions were made, and why."
---

# Investigation Diary

## Goal

Capture the investigation and design process for the JS developer environment
in the scraper web UI.

## Step 1: Current State Analysis

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to create a proper JS development environment for scraper. Read the SCRAPER-OP-DEBUGGER and get inspiration for: i want syntax highlighting in the verbs view, i want to be able to reload JS from disk, i want to have some way of running scripts with a certain ctx input, i want to view past executions of a script / verb and the input / output, and then use that to do the rerunning and such, i want to have some kind of REPL widget where I can see the script run console and maybe interact with it too."

**Assistant interpretation:** Design a comprehensive JS development environment for the scraper web UI, covering syntax highlighting, script reload, execution runner, execution history, and REPL console.

**Inferred user intent:** The user wants to transform the scraper UI from a passive observer into an active development workspace for JS site authors, enabling fast edit-test loops, debugging, and interactive exploration.

### What I did

1. Read the SCRAPER-OP-DEBUGGER design document to understand the existing debugger architecture and the DebugBundle model.
2. Explored the current frontend architecture: all components, API types, RTK Query endpoints.
3. Explored the current backend architecture: handlers, services, JS executor, engine model.
4. Identified the current ScriptViewer as plain-text `<pre>` with no highlighting.
5. Identified the SiteVerbList as parameter-only with no script source display.
6. Identified RTK Query caching as the root cause of the "no reload" problem.
7. Analyzed the JS execution context (`ctx`) contract in `pkg/js/runtime/executor.go`.
8. Created the docmgr ticket SCRAPER-JS-DEVENV.
9. Wrote the comprehensive design document in 5 chunks.

### Why

The existing SCRAPER-OP-DEBUGGER ticket focuses on workflow-level artifact browsing and per-op replay from durable state. This ticket complements it by focusing on the interactive development experience: writing scripts, testing them quickly, debugging inline, and exploring the runtime. Both tickets share the same backend execution model (sandboxed, read-only replay).

### What worked

- Reading the SCRAPER-OP-DEBUGGER document first gave excellent context on the existing architecture and safety model.
- Inspecting the actual component files (ScriptViewer, SiteVerbList, OpDetailDrawer) made the gap analysis concrete.
- Writing the design document in chunks avoided any single write failing.

### What didn't work

- N/A — no failures during investigation.

### What I learned

- The current ScriptViewer is only 45 lines of code — replacing it is straightforward.
- The backend already reads scripts from the filesystem on every request — reload is purely a frontend cache invalidation problem.
- The execution runner needs a new backend endpoint but can reuse the existing executor with sandboxed adapters.
- The REPL is the most complex feature because it requires persistent server-side state (WebSocket + JS VM).

### What was tricky to build

- Designing the "Load from Op" workflow that bridges the SCRAPER-OP-DEBUGGER DebugBundle with the execution runner's ctx input was the trickiest conceptual design. The solution is to use the DebugBundle as the source for the ctx template, mapping dependency results into the `ctx.dep()` resolver.

### What warrants a second pair of eyes

- The REPL WebSocket protocol design — ensure it handles edge cases (network interruption, long-running evaluations, VM memory leaks).
- The execution history storage strategy (localStorage vs backend) — the recommendation is to start with localStorage and migrate, but this needs validation.
- The ctx input editor UX — the "view mode / edit mode" toggle may be confusing.

### What should be done in the future

- Implement Phase 1 (syntax highlighting) and Phase 2 (reload) first — they are the highest impact with lowest risk.
- Validate the backend execution API design with a spike before building the full UI.
- Consider adding authentication/authorization to the execution endpoints.

### Code review instructions

- Review the design document at: `design-doc/01-js-dev-environment-ui-design-and-implementation-guide.md`
- Focus on: API endpoint design, state machine diagrams, and YAML widget hierarchies.
- Validate that the proposed component directory structure makes sense.

### Technical details

- Key files analyzed: `web/src/components/scripts/ScriptViewer.tsx`, `web/src/components/sites/SiteVerbList.tsx`, `web/src/components/workflows/OpDetailDrawer.tsx`, `pkg/js/runtime/executor.go`, `web/src/api/catalogApi.ts`
