---
Title: JS Dev Environment UI Design and Implementation Guide
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
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/engine/runner/js.go:JS runner that wires executor into the scheduler
    - Path: pkg/js/runtime/executor.go
      Note: Core JS execution engine that builds ctx and runs scripts
    - Path: pkg/js/runtime/executor.go:Core JS execution engine that builds ctx and runs scripts
    - Path: web/src/api/catalogApi.ts
      Note: RTK Query API for sites/verbs/scripts
    - Path: web/src/api/catalogApi.ts:RTK Query API for sites/verbs/scripts
    - Path: web/src/api/types.ts:Shared TypeScript domain types
    - Path: web/src/api/workflowApi.ts
      Note: RTK Query API for workflows/ops/results
    - Path: web/src/api/workflowApi.ts:RTK Query API for workflows/ops/results
    - Path: web/src/components/scripts/ScriptViewer.tsx
      Note: Current plain-text script viewer to be replaced with syntax highlighting
    - Path: web/src/components/scripts/ScriptViewer.tsx:Current plain-text script viewer
    - Path: web/src/components/sites/SiteScriptBrowser.tsx
      Note: Script file browser needing reload button
    - Path: web/src/components/sites/SiteScriptBrowser.tsx:Script file browser with source view
    - Path: web/src/components/sites/SiteVerbList.tsx
      Note: Verb accordion list needing inline source view
    - Path: web/src/components/sites/SiteVerbList.tsx:Verb accordion list on site detail page
    - Path: web/src/components/workflows/OpDetailDrawer.tsx
      Note: Op detail drawer with script tab needing reload
    - Path: web/src/components/workflows/OpDetailDrawer.tsx:Op detail drawer with tabs
ExternalSources:
    - SCRAPER-OP-DEBUGGER:Related ticket for workflow artifact browsing and per-op JS replay debugging
Summary: Detailed intern-facing design document for building a JS development environment inside the scraper web UI. Covers syntax-highlighted verb viewer, script reload from disk, execution replay with ctx input, execution history browser, and a REPL console widget. Focused on UI design with ASCII mockups, YAML widget hierarchy DSL, and pseudocode. No production code.
LastUpdated: 2026-04-07T19:00:00-04:00
WhatFor: Guide the implementation of a first-class JS developer environment surface for scraper site authors and debuggers.
WhenToUse: Use when implementing or reviewing any JS dev environment UI component, the execution history system, or the REPL console.
---


# JS Dev Environment UI Design and Implementation Guide

## Executive Summary

This document specifies the user interface design for a JavaScript development
environment embedded in the scraper web application. The goal is to give site
authors and operators a cohesive, browser-based workspace for writing, testing,
debugging, and iterating on scraper JS scripts — without leaving the UI or
resubmitting entire workflows.

The design covers five major features:

1. **Syntax-highlighted verb viewer** — replace the current plain-text code
   display in the verbs tab and script browser with proper token-level
   highlighting for JavaScript.

2. **Script reload from disk** — add a one-click "Reload from Disk" action that
   re-fetches the script source from the backend (which reads from the
   filesystem), so site authors can edit scripts in their editor and see changes
   reflected immediately.

3. **Execution runner with ctx input** — a dedicated panel where a user selects
   a verb or script, provides a `ctx`-compatible JSON input, and runs the script
   in a sandboxed backend execution. Shows output, logs, artifacts, and errors
   inline.

4. **Execution history browser** — a timeline/list view of past executions for
   a given script or verb, showing input, output, duration, status, and error.
   Each history entry can be expanded for full detail and used as the basis for
   a re-run with the same or modified input.

5. **REPL console widget** — a live interactive console where the user can type
   JS expressions or script fragments, see console output in real time, and
   interact with the scraper JS runtime. Think browser DevTools console, but
   for scraper scripts.

This document is focused on UI design, not backend implementation. ASCII
mockups, YAML widget hierarchies (a compressed TSX-like notation), and
pseudocode are used throughout. The target audience is a new intern who needs to
understand every part of the system.

## Table of Contents

1. [Problem Statement](#problem-statement)
2. [Current-State Architecture](#current-state-architecture)
3. [Gap Analysis](#gap-analysis)
4. [Design Principles](#design-principles)
5. [Feature 1: Syntax-Highlighted Verb Viewer](#feature-1-syntax-highlighted-verb-viewer)
6. [Feature 2: Script Reload from Disk](#feature-2-script-reload-from-disk)
7. [Feature 3: Execution Runner with Ctx Input](#feature-3-execution-runner-with-ctx-input)
8. [Feature 4: Execution History Browser](#feature-4-execution-history-browser)
9. [Feature 5: REPL Console Widget](#feature-5-repl-console-widget)
10. [Widget Hierarchy Reference (YAML DSL)](#widget-hierarchy-reference-yaml-dsl)
11. [Backend API Endpoints Needed](#backend-api-endpoints-needed)
12. [Phased Implementation Plan](#phased-implementation-plan)
13. [Testing Strategy](#testing-strategy)
14. [Risks and Open Questions](#risks-and-open-questions)
15. [References](#references)

---

## Problem Statement

The scraper web UI today has three views relevant to JavaScript development:

- **Site Detail → Verbs tab**: An accordion list of verb definitions showing
  name, parameters, and help text. No script source is shown here.

- **Site Detail → Scripts tab**: A file browser on the left, source viewer on
  the right. The source viewer (`ScriptViewer.tsx`) renders code as plain
  monospace text in a `<pre>` block — no syntax highlighting, no line
  decoration beyond line numbers.

- **Workflow Detail → Op Detail Drawer → Script tab**: Same `ScriptViewer`
  component, showing the script associated with a workflow op. Read-only, plain
  text.

These views share three critical limitations:

1. **No syntax highlighting.** JavaScript code is rendered as unstyled text. For
   any script longer than ~20 lines, the developer has to mentally parse the
   code to understand structure, string boundaries, comments, or keywords.

2. **No reload mechanism.** Once a script source is loaded, it is cached by RTK
   Query. If the developer edits the file on disk and saves, they must
   invalidate the cache manually (or hard-refresh the page) to see the updated
   source. There is no "reload" button.

3. **No execution or replay.** The UI is purely observational. The developer can
   see what a script *contains* and what it *produces* (via workflow results),
   but cannot *run* a script with custom input, view console output, or iterate
   on a script without going through the full workflow submission path.

A fourth and fifth limitation exist at a higher level:

4. **No execution history.** There is no way to see a list of "all the times I
   ran this script" with their inputs and outputs. Past executions are scattered
   across workflow runs and ops, making it hard to compare runs or reproduce a
   specific failure.

5. **No interactive console.** There is no REPL-like surface where the developer
   can type JS, see immediate feedback, explore the runtime context, or debug
   incrementally.

This design document addresses all five gaps, prioritized for developer and
debugger ergonomics.

---

## Current-State Architecture

### Frontend Stack

The scraper web UI is a React + TypeScript SPA using:

- **Vite** for build and dev server
- **MUI (Material UI)** for component library
- **RTK Query** (Redux Toolkit) for data fetching and caching
- **Storybook** for component development
- **Vitest** for unit tests

Key files and their responsibilities:

| File | Responsibility |
|------|---------------|
| `web/src/App.tsx` | Top-level routing and layout |
| `web/src/api/types.ts` | Shared TypeScript domain types matching Go models |
| `web/src/api/catalogApi.ts` | RTK Query API for sites, verbs, scripts |
| `web/src/api/workflowApi.ts` | RTK Query API for workflows, ops, results, artifacts |
| `web/src/pages/SiteDetailPage.tsx` | Site detail page with Overview/Verbs/Scripts tabs |
| `web/src/components/sites/SiteVerbList.tsx` | Accordion list of verb definitions |
| `web/src/components/sites/SiteScriptBrowser.tsx` | File browser + source viewer |
| `web/src/components/scripts/ScriptViewer.tsx` | Plain-text source code viewer |
| `web/src/components/scripts/ScriptTab.tsx` | Wrapper for ScriptViewer with loading/error states |
| `web/src/components/workflows/OpDetailDrawer.tsx` | Drawer with input/deps/result/artifacts/runtime/script/logs tabs |

### Backend Stack

The scraper backend is a Go application with:

- **Echo** HTTP framework for API routing
- **SQLite** for durable workflow/op/result/artifact storage
- **goja** (Go JS engine) for JavaScript execution
- **go-go-goja** for module system, runtime factory, native modules

Key backend files:

| File | Responsibility |
|------|---------------|
| `pkg/api/handlers/engine.go` | HTTP handlers for workflow/op/result/artifact APIs |
| `pkg/api/handlers/catalog.go` | HTTP handlers for site/verb/script catalog APIs |
| `pkg/services/engineview/service.go` | Business logic for querying workflow state |
| `pkg/services/catalog/service.go` | Business logic for site/verb/script catalog |
| `pkg/js/runtime/executor.go` | Core JS executor that builds `ctx`, requires script, runs it |
| `pkg/engine/runner/js.go` | Scheduler runner that wires the executor into the engine |

### The JS Execution Context (`ctx`)

When a JS op runs, the executor in `pkg/js/runtime/executor.go` builds a `ctx`
object that the script receives as its first argument. This is the contract
between the scraper runtime and user-written scripts.

The `ctx` object contains:

```javascript
{
  site:           string,         // e.g. "hackernews"
  now:            string,         // ISO 8601 timestamp
  workflow: {
    id:           string,
    site:         string,
    name:         string,
    status:       string,
    input:        any,            // workflow-level input JSON
    metadata:     Record<string,string>
  },
  op: {
    id:           string,
    workflowID:   string,
    site:         string,
    kind:         string,         // always "js" for JS ops
    queue:        string,
    dedupKey:     string,
    metadata:     Record<string,string>,
    input:        any             // op-level input JSON
  },
  lease: { ... },                // worker lease info
  // Methods the script can call:
  log:           (msg: string) => void,
  dep:           (opID: string) => any,     // resolve dependency result
  emit:          (opSpec) => void,           // emit child ops
  writeRecord:   (collection, key, data) => void,
  writeArtifact: (name, kind, contentType, data) => void,
}
```

This is the central data contract for all JS execution and replay. Any
"execution runner" or "REPL" feature must provide a way to supply this context
(or a simplified subset of it).

### Current Script Source Flow

```
Browser                   Backend                     Filesystem
  |                         |                             |
  | GET /api/v1/sites/X/scripts/Y                       |
  |------------------------>|                             |
  |                         | read file from embedded FS  |
  |                         |---------------------------->|
  |                         |<----------------------------|
  |   { source: "..." }     |                             |
  |<------------------------|                             |
  |                         |                             |
  | RTK Query caches result |                             |
  |                         |                             |
  | GET /api/v1/sites/X/scripts/Y  (cached, no reload)  |
  |-----(cache hit)-------->|                             |
```

The problem: RTK Query caches the script source response. There is no cache
invalidation mechanism exposed in the UI. The developer cannot force a re-read
from disk.

### Current Verb Display

The `SiteVerbList` component shows verbs as MUI Accordions. Each accordion
expands to show:
- Verb name (monospace subtitle)
- Short description
- A table of parameter sections and fields (name, type, default, help)

Critically, it does **not** show the verb's script source. The user must switch
to the Scripts tab, find the right file, and read it there. There is no direct
link between a verb name and the script that implements it.

---

## Gap Analysis

| Gap | Current Behavior | Desired Behavior | Impact if Unaddressed |
|-----|-----------------|-------------------|----------------------|
| No syntax highlighting | `ScriptViewer` renders code in a `<pre>` block with no token coloring | Token-level JS highlighting (keywords, strings, comments, numbers, identifiers, punctuation) using a lightweight highlighter | Developers struggle to read code >20 lines. Mental parsing slows debugging. |
| No reload from disk | RTK Query caches script source; no invalidation UI | One-click "Reload from Disk" button that invalidates the RTK Query cache and re-fetches | Developer must hard-refresh page after editing a script. Breaks edit-test loop. |
| No execution runner | UI is read-only; scripts can only run through workflow submission | Dedicated panel to select a script, provide `ctx` JSON, execute on backend, show result inline | Developer must submit a full workflow to test a single script change. Very slow iteration. |
| No execution history | Past executions are scattered across workflow ops | Chronological list of past runs per script/verb with input, output, status, duration | Cannot compare runs, reproduce failures, or see trends without manual workflow hunting. |
| No REPL console | No interactive JS surface anywhere in the UI | Live console with input prompt, real-time output, access to scraper runtime context | Developer cannot explore the runtime, test expressions, or debug incrementally. |

### Gap Severity and Priority

```
Priority 1 (blocks basic developer workflow):
  ┌─────────────────────────────────────────────┐
  │ Syntax highlighting                         │
  │ Script reload from disk                     │
  └─────────────────────────────────────────────┘

Priority 2 (enables iterative development):
  ┌─────────────────────────────────────────────┐
  │ Execution runner with ctx input              │
  └─────────────────────────────────────────────┘

Priority 3 (enables debugging and exploration):
  ┌─────────────────────────────────────────────┐
  │ Execution history browser                    │
  │ REPL console widget                          │
  └─────────────────────────────────────────────┘
```

The phased implementation plan follows this priority order.

---

## Design Principles

These principles govern every UI decision in this document. When in doubt,
refer back to these.

### P1: The edit-test loop must be fast

The primary user workflow is: edit script in editor → save → see updated source
→ run with input → inspect output → repeat. Every second of friction in this
loop reduces productivity.

Consequences:
- Reload must be one click, not a full page refresh.
- Execution must return results inline, not require navigating to a different page.
- Execution history must be visible alongside the script source.

### P2: The UI is a viewer and launcher, not an editor

We are not building an in-browser code editor (like Monaco or CodeMirror in
editable mode). The developer edits scripts in their preferred editor (VS Code,
vim, etc.). The scraper UI shows the source, lets them reload it, and lets them
execute it.

This simplifies the design enormously:
- No in-browser editing state management.
- No auto-save or conflict resolution.
- No collaborative editing.
- The "source of truth" is always the file on disk.

### P3: Context is king

JavaScript in scraper is not standalone. It runs with a `ctx` object that
provides workflow info, op info, dependency results, and write methods. The UI
must make this context visible and editable at every step:
- When viewing a past execution, show the exact `ctx` that was used.
- When running a new execution, let the user provide or modify `ctx` input.
- When showing the REPL, expose the current `ctx` state.

### P4: Read-only replay first, write-side later

The execution runner and history browser start as read-only observation tools.
The user can inspect and re-run, but the replay does not create durable workflow
ops or write to the engine database. This follows the safety model from the
SCRAPER-OP-DEBUGGER ticket.

### P5: Use MUI patterns consistently

The scraper UI uses MUI throughout. New components must follow the same
patterns: MUI components, sx prop styling, RTK Query for data, consistent
spacing and typography. No new CSS-in-JS libraries, no new component libraries
beyond what is already in `package.json`.

---

## Feature 1: Syntax-Highlighted Verb Viewer

### Overview

Replace the plain-text `<pre>` rendering in `ScriptViewer` with token-level
JavaScript syntax highlighting. This applies to three locations:
1. Site Detail → Scripts tab (the `SiteScriptBrowser` component)
2. Workflow Detail → Op Detail Drawer → Script tab (the `ScriptTab` component)
3. A new inline source view that will appear in the Verb accordion (see below)

### Where Syntax Highlighting Appears

```
Location                          Component          Currently
─────────────────────────────────────────────────────────────────
Site Detail > Scripts tab         ScriptViewer        Plain <pre>
Op Detail Drawer > Script tab     ScriptTab           Plain <pre>
Site Detail > Verbs tab           SiteVerbList         No source shown
Execution Runner > Source pane    (new)               N/A
Execution History > Source pane   (new)               N/A
REPL Console > Source preview     (new)               N/A
```

### Approach: Lightweight Token Highlighter

We do not need a full IDE-grade parser. We need enough to color:
- Keywords (`function`, `const`, `let`, `var`, `if`, `else`, `return`, `for`, `while`, `async`, `await`, `require`, `import`, `export`, `try`, `catch`, `throw`, `new`, `typeof`, `instanceof`, `class`, `extends`, `this`, `null`, `undefined`, `true`, `false`)
- Strings (single-quoted, double-quoted, template literals)
- Comments (single-line `//`, multi-line `/* */`)
- Numbers
- Punctuation and operators
- Identifiers (function names, variable names)

Two viable approaches:

**Option A: Use a library like Prism.js or Highlight.js**

Pros: battle-tested, handles edge cases, supports many languages, easy theming.
Cons: additional dependency, may include more than we need.

**Option B: Use a lightweight regex-based tokenizer**

Pros: zero dependencies, full control over token types, easy to customize for
scraper-specific patterns (like `ctx.dep()`, `ctx.emit()`, `ctx.log()`).
Cons: more code to write and test, may miss edge cases.

**Recommendation: Option A (Prism.js or Highlight.js)** for the MVP, with the
option to switch to a custom tokenizer later for scraper-specific highlighting
(e.g., highlighting `ctx` method calls differently from regular function calls).

Prism.js is preferred because:
- it produces a flat list of `<span>` elements with CSS classes,
- it works well with React,
- it has a small core (2KB gzipped for the JS language definition),
- it supports line-numbering and line-highlighting via plugins.

### Component Design: `SyntaxHighlightedScriptViewer`

This is a drop-in replacement for the current `ScriptViewer`. It accepts the
same props and renders highlighted code instead of plain text.

```
Props:
  source: string           // the raw JS source code
  filename: string         // for the caption and language detection
  highlightLines?: number[]  // optional: lines to visually emphasize (for debugging)
  maxLines?: number         // optional: virtualize rendering for very long files
```

### ASCII Mockup: Before vs After

**Before (current `ScriptViewer`):**

```
┌──────────────────────────────────────────────────────────┐
│ scripts/fetch-page.js                                     │
│ ┌────────────────────────────────────────────────────────┐│
│ │  1 │ const axios = require('axios');                   ││
│ │  2 │                                                   ││
│ │  3 │ async function run(ctx) {                         ││
│ │  4 │   const url = ctx.op.input.url;                   ││
│ │  5 │   ctx.log("Fetching: " + url);                    ││
│ │  6 │   const response = await axios.get(url);          ││
│ │  7 │   ctx.writeArtifact(                              ││
│ │  8 │     "page.html",                                  ││
│ │  9 │     "html",                                       ││
│ │ 10 │     "text/html",                                  ││
│ │ 11 │     response.data                                 ││
│ │ 12 │   );                                              ││
│ │ 13 │   return { title: response.data.title };          ││
│ │ 14 │ }                                                 ││
│ │ 15 │                                                   ││
│ │ 16 │ module.exports = { run };                         ││
│ └────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────┘
  All text is the same color. Hard to distinguish keywords,
  strings, and comments at a glance.
```

**After (new `SyntaxHighlightedScriptViewer`):**

```
┌──────────────────────────────────────────────────────────┐
│ scripts/fetch-page.js                                     │
│ ┌────────────────────────────────────────────────────────┐│
│ │  1 │ [kw]const[/] [id]axios[/] = [kw]require[/]([str]││
│ │    │'axios'[/]);                                       ││
│ │  2 │                                                   ││
│ │  3 │ [kw]async[/] [kw]function[/] [fn]run[/]([id]ctx[/││
│ │    │]) {                                               ││
│ │  4 │   [kw]const[/] [id]url[/] = [id]ctx[/].[id]op[/].││
│ │    │[id]input[/].[id]url[/];                            ││
│ │  5 │   [id]ctx[/].[fn]log[/]([str]"Fetching: "[/] +   ││
│ │    │[id]url[/]);                                       ││
│ │  6 │   [kw]const[/] [id]response[/] = [kw]await[/]    ││
│ │    │[id]axios[/].[fn]get[/]([id]url[/]);               ││
│ │  7 │   [id]ctx[/].[fn]writeArtifact[/](               ││
│ │  8 │     [str]"page.html"[/],                          ││
│ │  9 │     [str]"html"[/],                               ││
│ │ 10 │     [str]"text/html"[/],                           ││
│ │ 11 │     [id]response[/].[id]data[/]                   ││
│ │ 12 │   );                                              ││
│ │ 13 │   [kw]return[/] { [id]title[/]: [id]response[/].  ││
│ │    │[id]data[/].[id]title[/] };                         ││
│ │ 14 │ }                                                 ││
│ │ 15 │                                                   ││
│ │ 16 │ [id]module[/].[id]exports[/] = { [fn]run[/] };   ││
│ └────────────────────────────────────────────────────────┘│
│ Legend: [kw]=purple  [str]=green  [fn]=blue  [id]=white   │
│         [cmt]=gray   [num]=orange                          │
└──────────────────────────────────────────────────────────┘
```

In a real browser, these would be styled `<span>` elements with CSS classes
like `token keyword`, `token string`, `token function`, etc.

### YAML Widget Hierarchy

This uses a compressed YAML DSL to represent the React component tree.
Think of it as "TSX without the JSX" — each node has a `component`, optional
`props`, and `children`.

```yaml
# SyntaxHighlightedScriptViewer
component: Box
props: { sx: { position: relative } }
children:
  - component: Typography
    props: { variant: caption, color: text.secondary }
    text: "${filename}"
  - component: Box
    props:
      sx:
        maxHeight: 500
        overflow: auto
        backgroundColor: "#1e1e2e"  # dark theme
        borderRadius: 1
        border: 1px solid
        borderColor: divider
    children:
      - component: HighlightedCodeBlock
        props:
          source: "${source}"
          language: javascript
          showLineNumbers: true
          highlightLines: "${highlightLines}"
          theme: scraper-dark
        children: []  # renders internally with Prism spans
```

### Verb Source Inline View

In addition to replacing `ScriptViewer`, we want to show script source
*inside* the verb accordion on the Site Detail → Verbs tab. Currently, the
accordion shows parameter tables but no code. We add a collapsible "Source"
section at the bottom of each verb's accordion panel.

```
┌─▼ fetch-page ────────────────────────────────────────────┐
│ Fetches a URL and stores the HTML as an artifact.         │
│                                                           │
│ ┌─ Parameters ──────────────────────────────────────────┐ │
│ │ Name       Type     Default    Help                   │ │
│ │ url*       string   -          URL to fetch           │ │
│ │ timeout    number   30000      Request timeout in ms  │ │
│ └───────────────────────────────────────────────────────┘ │
│                                                           │
│ ┌─▼ Source (scripts/fetch-page.js) ────────────────────┐ │
│ │  1 │ const axios = require('axios');                 │ │
│ │  2 │ async function run(ctx) { ...                   │ │
│ │  ...                                                 │ │
│ │ 16 │ module.exports = { run };                       │ │
│ └───────────────────────────────────────────────────────┘ │
│                                                           │
│ [▶ Run with Input]  [📋 Copy Source]  [🔄 Reload]        │
└───────────────────────────────────────────────────────────┘
```

The "Source" section is collapsed by default. The user clicks to expand it. It
uses the same `SyntaxHighlightedScriptViewer` component. The action buttons at
the bottom are new — they are entry points to Features 3 and 4.

### Pseudocode: Rendering Pipeline

```
function SyntaxHighlightedScriptViewer({ source, filename, highlightLines }) {
  // 1. Tokenize the source using Prism
  const tokens = Prism.tokenize(source, Prism.languages.javascript)

  // 2. Flatten tokens into styled spans
  const spans = tokens.map((token, i) => {
    if (typeof token === 'string') {
      return <span key={i}>{token}</span>
    }
    return <span key={i} className={`token ${token.type}`}>{token.content}</span>
  })

  // 3. Split into lines for line numbering
  const lines = splitIntoLines(spans)

  // 4. Render with line numbers and optional highlighting
  return (
    <Box className="code-container">
      {lines.map((line, i) => (
        <Box className={highlightLines?.includes(i+1) ? 'line highlighted' : 'line'}>
          <span className="line-number">{i + 1}</span>
          <span className="line-content">{line}</span>
        </Box>
      ))}
    </Box>
  )
}
```

### Files to Modify

| File | Change |
|------|--------|
| `web/src/components/scripts/ScriptViewer.tsx` | Replace with `SyntaxHighlightedScriptViewer` or wrap internally |
| `web/src/components/scripts/ScriptTab.tsx` | Use new viewer, no API change |
| `web/src/components/sites/SiteVerbList.tsx` | Add inline "Source" section to each verb accordion |
| `web/src/components/sites/SiteScriptBrowser.tsx` | Use new viewer |
| `web/src/components/workflows/OpDetailDrawer.tsx` | Use new viewer in Script tab |
| `web/src/theme.ts` | Add syntax highlighting color tokens |
| `web/package.json` | Add `prismjs` dependency |

---

## Feature 2: Script Reload from Disk

### Overview

Add a "Reload from Disk" action button wherever script source is displayed. When
clicked, the button:

1. Invalidates the RTK Query cache entry for that script.
2. Re-fetches the script source from the backend.
3. Shows a brief loading state.
4. Displays the updated source.

This is the simplest feature in the design, but it is critical for the
edit-test loop (Principle P1).

### Where Reload Appears

Every surface that shows script source should have a reload button:

```
┌──────────────────────────────────────────────────────────┐
│ Location                              Reload Button?      │
├──────────────────────────────────────────────────────────┤
│ Site Detail > Scripts tab             Yes (toolbar)      │
│ Site Detail > Verbs > Source section  Yes (inline)       │
│ Op Detail Drawer > Script tab         Yes (toolbar)      │
│ Execution Runner > Source pane        Yes (toolbar)      │
│ Execution History > Source preview    No (read-only)     │
│ REPL Console                          No (uses live ctx) │
└──────────────────────────────────────────────────────────┘
```

Execution history entries are frozen snapshots of the source at the time of
execution, so they should NOT reload. The REPL console uses its own execution
context, so reload is handled differently there.

### UI Mockup: Reload Button in Script Browser

```
┌─ Scripts ─────────────────────────────────────────────────┐
│ ┌──────────┐ ┌─────────────────────────────────────────┐  │
│ │ scripts/ │ │ scripts/fetch-page.js     [🔄 Reload]   │  │
│ │          │ │ ─────────────────────────────────────    │  │
│ │ fetch-p..│ │  1 │ const axios = require('axios');    │  │
│ │ parse-h..│ │  2 │                                   │  │
│ │ extract..│ │  3 │ async function run(ctx) {          │  │
│ │          │ │  4 │   const url = ctx.op.input.url;    │  │
│ │ utils/   │ │ ...│ ...                               │  │
│ │  helper..│ │ 16 │ module.exports = { run };          │  │
│ └──────────┘ └─────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────┘
```

The reload button appears next to the filename in the script viewer header.
It is a small icon button with a refresh icon. On hover, a tooltip says
"Reload from Disk".

### Reload Button States

```
┌──────────────────────────────────────────────────────┐
│ State          Visual           Enabled   Duration   │
├──────────────────────────────────────────────────────┤
│ idle           🔄 (refresh icon)   Yes      -       │
│ loading        ⟳ (spinning)        No       ~200ms  │
│ success        ✅ (check, 1.5s)    No       1.5s    │
│ error          ⚠️ (error)          Yes      until   │
│                                    next click       │
└──────────────────────────────────────────────────────┘
```

### Implementation: RTK Query Cache Invalidation

The current `getScript` endpoint in `catalogApi.ts` uses RTK Query's automatic
caching. To force a re-fetch, we use the `initiate` function with
`forceRefetch: true`:

```typescript
// Pseudocode for the reload handler
function useReloadScript(site: string, path: string) {
  const dispatch = useDispatch()
  const [isReloading, setIsReloading] = useState(false)

  const reload = useCallback(async () => {
    setIsReloading(true)
    // Invalidate the specific cache entry
    dispatch(
      catalogApi.util.invalidateTags([{ type: 'Scripts', id: `${site}:${path}` }])
    )
    // The query will automatically re-fetch because the tag was invalidated
    // Add a small delay for the loading animation
    await new Promise(r => setTimeout(r, 300))
    setIsReloading(false)
  }, [dispatch, site, path])

  return { reload, isReloading }
}
```

Alternatively, the `refetch` function returned by the query hook can be called
directly:

```typescript
const { data, refetch, isFetching } = useGetScriptQuery({ site, path })
// ...
<button onClick={() => refetch()}>Reload</button>
```

This is simpler and preferred. The `refetch` function from RTK Query
automatically bypasses the cache and makes a fresh request.

### YAML Widget Hierarchy: Reload Button

```yaml
# ScriptViewerWithReload
component: Box
props: { sx: { display: flex, flexDirection: column, height: 100% } }
children:
  # Toolbar row
  - component: Box
    props:
      sx:
        display: flex
        justifyContent: space-between
        alignItems: center
        mb: 0.5
    children:
      - component: Typography
        props: { variant: caption, color: text.secondary }
        text: "${filename}"
      - component: IconButton
        props:
          onClick: "${handleReload}"
          disabled: "${isReloading}"
          size: small
          title: Reload from Disk
        children:
          - component: RefreshIcon
            props:
              className: "${isReloading ? 'spinning' : ''}"
              fontSize: small

  # Code display
  - component: SyntaxHighlightedScriptViewer
    props:
      source: "${source}"
      filename: "${filename}"
```

### Backend: No Changes Needed

The existing `GET /api/v1/sites/{site}/scripts/{path}` endpoint already reads
from the filesystem on every request. The backend does not cache script source
itself — RTK Query on the frontend does. So the reload is purely a frontend
concern.

---

## Feature 3: Execution Runner with Ctx Input

### Overview

The execution runner is the heart of the developer environment. It lets a user:

1. Select a script (either from the verb list or the script browser).
2. Provide a `ctx`-compatible JSON input.
3. Execute the script on the backend in a sandboxed, read-only replay mode.
4. See the result inline: output data, console logs, artifacts, and errors.

This turns the scraper UI from a passive observer into an active development
tool.

### Conceptual Model

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│   Developer selects script ──────┐                          │
│                                   │                          │
│   Developer provides ctx input ───┼──▶ ExecutionRunner      │
│                                   │         │               │
│   Developer clicks "Run" ─────────┘         │               │
│                                              ▼               │
│                                     Backend executes        │
│                                     script with ctx         │
│                                              │               │
│                                              ▼               │
│                                     Result envelope:        │
│                                     - data                  │
│                                     - logs                  │
│                                     - artifacts             │
│                                     - records               │
│                                     - emitted ops           │
│                                     - error (if any)        │
│                                     - duration              │
│                                              │               │
│                                              ▼               │
│                                     Result displayed        │
│                                     in the UI               │
│                                                             │
│   [Re-run with same input]  [Re-run with modified input]   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### The Execution Runner Page Layout

The execution runner is a dedicated panel, accessible from:

1. Site Detail → Verbs tab → "▶ Run with Input" button in the verb accordion.
2. Site Detail → Scripts tab → "▶ Run" button in the script browser toolbar.
3. Workflow Detail → Op Detail Drawer → "▶ Re-execute" button in the Script tab.

When activated, it opens as a **full-width panel below the site header** or as a
**modal dialog**, depending on the entry point. The recommended layout for the
initial implementation is a panel within the Site Detail page, replacing the
current tab content.

#### ASCII Mockup: Execution Runner Panel

```
┌─ Site: hackernews ─────────────────────────────────────────────────────────┐
│ Overview | Verbs | Scripts | ▶ Execution Runner                            │
│─────────────────────────────────────────────────────────────────────────────│
│                                                                             │
│ ┌─ Script ────────────────────────────────────────────────────────────────┐ │
│ │ Script: [hackernews/scripts/fetch-page.js ▼]    [🔄 Reload]  [View]   │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│ ┌─ Input (ctx) ────────────────────────┐ ┌─ Source Preview ──────────────┐ │
│ │ {                                    │ │  1│ const axios = require(...  │ │
│ │   "site": "hackernews",              │ │  3│ async function run(ctx) {  │ │
│ │   "now": "2026-04-07T19:00:00Z",     │ │  4│   const url = ctx.input... │ │
│ │   "workflow": {                      │ │ ...│ ...                       │ │
│ │     "id": "test-wf-001",             │ │ 16│ module.exports = { run };   │ │
│ │     "input": {}                      │ │                               │ │
│ │   },                                 │ │                               │ │
│ │   "op": {                            │ │                               │ │
│ │     "id": "test-op-001",             │ │                               │ │
│ │     "input": {                       │ │                               │ │
│ │       "url": "https://news.ycom..."  │ │                               │ │
│ │     }                                │ │                               │ │
│ │   }                                  │ │                               │ │
│ │ }                                    │ │                               │ │
│ │                                      │ │                               │ │
│ │ [Load from template ▼] [From op]     │ │                               │ │
│ └──────────────────────────────────────┘ └───────────────────────────────┘ │
│                                                                             │
│ ┌─ Actions ───────────────────────────────────────────────────────────────┐ │
│ │  [▶ Run Script]    [⏹ Cancel]    Duration: 0.42s    Status: ● Ready    │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│ ┌─ Result ────────────────────────────────────────────────────────────────┐ │
│ │ ┌─ Output ───────┐ ┌─ Console ──────┐ ┌─ Artifacts ──────────────────┐ │ │
│ │ │ {              │ │ [INFO] Fetching │ │ page.html (text/html, 42KB)  │ │ │
│ │ │   "title": "..."│ │ [INFO] Status:  │ │ [Preview] [Download]         │ │ │
│ │ │ }              │ │ [WARN] No cache │ │                              │ │ │
│ │ │                │ │                 │ │                              │ │ │
│ │ └────────────────┘ └─────────────────┘ └──────────────────────────────┘ │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Input Panel: Providing the `ctx`

The input panel is a JSON editor (read-only syntax-highlighted view with an
"Edit" toggle that switches to a textarea). The user can:

1. **Type JSON manually.** A simple textarea with JSON validation.
2. **Load from template.** A dropdown of pre-built templates:
   - "Minimal ctx" — site, now, empty workflow/op
   - "Full ctx" — all fields populated with example data
   - "From last run" — reuse the ctx from the most recent execution
3. **Load from a workflow op.** The user can pick a completed workflow op, and
   the system fills in the ctx from that op's actual execution data. This is
   the "replay with real data" workflow.

#### Input Editor States

```
┌─ State: Viewing ───────────────────────────────────────────┐
│ { "site": "hackernews", ... }              [✏️ Edit]       │
└────────────────────────────────────────────────────────────┘

┌─ State: Editing ───────────────────────────────────────────┐
│ ┌────────────────────────────────────────────────────────┐ │
│ │ {                                                      │ │
│ │   "site": "hackernews",                                │ │
│ │   "op": { "input": { "url": "https://..." } }         │ │
│ │ }                                                      │ │
│ └────────────────────────────────────────────────────────┘ │
│ [✓ Apply] [✗ Cancel] [🔧 Format JSON] [📋 Paste from...] │
└────────────────────────────────────────────────────────────┘

┌─ State: Error ─────────────────────────────────────────────┐
│ ⚠️ Invalid JSON: Unexpected token at line 3, column 12    │
│ ┌────────────────────────────────────────────────────────┐ │
│ │ { "site": "hackernews",                                │ │
│ │   "op": { "input": { "url": "https://..." }            │ │
│ │ }                                           ↑ error    │ │
│ └────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────┘
```

### Result Panel: Output, Logs, and Artifacts

The result panel uses a tabbed layout similar to the Op Detail Drawer but
simplified for execution results:

- **Output tab**: The `data` field from the execution result, rendered as JSON.
- **Console tab**: The log entries captured during execution (from `ctx.log()`
  calls). Each entry has a timestamp and message.
- **Artifacts tab**: List of artifacts written by the script (from
  `ctx.writeArtifact()` calls). Each artifact can be previewed or downloaded.
- **Records tab**: List of records written by the script (from
  `ctx.writeRecord()` calls).
- **Emitted tab**: List of child ops emitted by the script (from `ctx.emit()`
  calls).

### YAML Widget Hierarchy: Execution Runner

```yaml
# ExecutionRunnerPanel
component: Box
props: { sx: { display: flex, flexDirection: column, gap: 2 } }
children:
  # Script selector row
  - component: ExecutionScriptSelector
    props:
      site: "${siteName}"
      scripts: "${scripts}"
      selectedScript: "${selectedScript}"
      onScriptChange: "${handleScriptChange}"
    children: []

  # Main content: input + source side by side
  - component: Box
    props: { sx: { display: flex, gap: 2, minHeight: 400 } }
    children:
      # Left: ctx input editor
      - component: ExecutionInputEditor
        props:
          value: "${ctxJson}"
          onChange: "${setCtxJson}"
          validationError: "${jsonError}"
          templates: "${ctxTemplates}"
          onLoadFromOp: "${loadFromOp}"
        props.sx: { flex: 1, minWidth: 0 }
        children: []

      # Right: source preview (read-only)
      - component: Box
        props.sx: { flex: 1, minWidth: 0 }
        children:
          - component: SyntaxHighlightedScriptViewer
            props:
              source: "${scriptSource}"
              filename: "${selectedScript}"
              maxLines: 50

  # Action bar
  - component: ExecutionActionBar
    props:
      onRun: "${handleRun}"
      onCancel: "${handleCancel}"
      isRunning: "${isRunning}"
      duration: "${executionDuration}"
      status: "${executionStatus}"
    children: []

  # Result panel (shown after execution)
  - component: ConditionalRender
    condition: "${hasResult}"
    children:
      - component: ExecutionResultPanel
        props:
          result: "${executionResult}"
          logs: "${executionLogs}"
          artifacts: "${executionArtifacts}"
          records: "${executionRecords}"
          emittedOps: "${emittedOps}"
          error: "${executionError}"
        children: []
```

### YAML Widget Hierarchy: Execution Result Panel

```yaml
# ExecutionResultPanel
component: Card
children:
  - component: Tabs
    props: { value: "${resultTab}", onChange: "${setResultTab}" }
    children:
      - component: Tab
        props: { label: "Output", value: "output" }
      - component: Tab
        props:
          label:
            component: Badge
            props: { badgeContent: "${logs.length}", color: primary }
            children: [{ text: "Console" }]
          value: "console"
      - component: Tab
        props:
          label:
            component: Badge
            props: { badgeContent: "${artifacts.length}", color: primary }
            children: [{ text: "Artifacts" }]
          value: "artifacts"
      - component: Tab
        props: { label: "Records", value: "records" }
      - component: Tab
        props: { label: "Emitted", value: "emitted" }
      - component: Tab
        props:
          label:
            component: Box
            props: { color: error ? error.main : text.primary }
            children: [{ text: "Error" }]
          value: "error"
        condition: "${!!error}"

  - component: TabPanel
    props: { value: "${resultTab}", index: "output" }
    children:
      - component: JsonViewer
        props: { data: "${result.data}" }

  - component: TabPanel
    props: { value: "${resultTab}", index: "console" }
    children:
      - component: ConsoleLogList
        props: { entries: "${logs}" }

  - component: TabPanel
    props: { value: "${resultTab}", index: "artifacts" }
    children:
      - component: ArtifactList
        props: { artifacts: "${artifacts}" }
      - component: ArtifactPreview
        props: { content: "${selectedArtifactBody}" }
```

### The "Load from Op" Workflow

One of the most powerful features is loading ctx from an existing workflow op.
This lets the developer grab real execution data and replay it:

```
Developer clicks "From op"
      │
      ▼
Modal opens with workflow/op picker:
  ┌─────────────────────────────────────────────┐
  │ Load ctx from completed op                   │
  │                                              │
  │ Workflow: [wf-2026-04-07-001 ▼]             │
  │                                              │
  │ Ops:                                         │
  │ ○ fetch-page     ✓ succeeded   0.3s ago     │
  │ ● parse-html     ✓ succeeded   0.1s ago     │
  │ ○ extract-links  ✗ failed      just now     │
  │                                              │
  │ Selected: parse-html                         │
  │                                              │
  │ Include:                                     │
  │ ☑ workflow input    ☑ op input               │
  │ ☑ dependency results                        │
  │ ☐ artifacts (may be large)                  │
  │                                              │
  │                    [Load] [Cancel]            │
  └─────────────────────────────────────────────┘
      │
      ▼
System fetches the op's debug bundle (or constructs ctx from
workflow + op + dependency results), fills in the input editor,
and the developer can tweak and run.
```

This is the bridge between the SCRAPER-OP-DEBUGGER ticket (which provides the
`DebugBundle` and bundle API) and this ticket (which provides the UI for using
that bundle as execution input).

### Backend API Endpoints Needed

The execution runner requires one new backend endpoint:

```
POST /api/v1/sites/{site}/scripts/run
```

Request body:
```json
{
  "scriptPath": "scripts/fetch-page.js",
  "ctx": {
    "site": "hackernews",
    "now": "2026-04-07T19:00:00Z",
    "workflow": { "id": "manual-001", "site": "hackernews", "input": {} },
    "op": { "id": "manual-op-001", "input": { "url": "https://news.ycombinator.com" } }
  },
  "options": {
    "timeout": 30000,
    "captureArtifacts": true,
    "captureRecords": true
  }
}
```

Response:
```json
{
  "status": "succeeded",
  "duration": 420,
  "data": { "title": "Hacker News" },
  "logs": [
    { "timestamp": "2026-04-07T19:00:00.001Z", "message": "Fetching: https://..." },
    { "timestamp": "2026-04-07T19:00:00.420Z", "message": "Done" }
  ],
  "artifacts": [
    { "name": "page.html", "kind": "html", "contentType": "text/html", "size": 42000, "body": "..." }
  ],
  "records": [],
  "emittedOps": [],
  "error": null
}
```

Safety: This endpoint is **read-only**. It does not create durable ops, does not
write to the engine DB, and does not mutate site DBs. It runs the script in an
in-memory sandbox and returns the captured output.

### Execution Runner State Machine

```
                    ┌─────────┐
                    │  IDLE   │
                    └────┬────┘
                         │ user clicks "Run"
                         ▼
                    ┌─────────┐
              ┌────▶│ RUNNING │◀────┐
              │     └────┬────┘     │
              │          │          │
    timeout / │          │ done     │ user clicks
    cancel    │          ▼          │ "Run" again
              │     ┌─────────┐    │
              │     │ DONE    │────┘
              │     │(success)│
              │     └─────────┘
              │
              ▼
         ┌─────────┐
         │ DONE    │
         │(failed) │
         └─────────┘
```

State transitions:
- IDLE → RUNNING: user clicks "Run"
- RUNNING → DONE(success): backend returns success response
- RUNNING → DONE(failed): backend returns error or timeout
- DONE(*) → RUNNING: user clicks "Re-run"
- RUNNING → IDLE: user clicks "Cancel" (or timeout)

---

## Feature 4: Execution History Browser

### Overview

The execution history browser shows a chronological list of past executions for
a given script or verb. Each entry captures:
- When it ran (timestamp)
- What input was used (ctx JSON)
- What output was produced (data, logs, artifacts)
- How long it took (duration)
- Whether it succeeded or failed (status)
- If it failed, the error message

The history browser serves three user workflows:

1. **Compare runs.** The developer runs the same script multiple times with
   slightly different inputs. They want to compare outputs side-by-side.

2. **Reproduce failures.** A script failed yesterday. The developer wants to
   see the exact input that caused the failure and re-run it.

3. **Track progress.** The developer is iterating on a script and wants to see
   if their changes are making things better or worse over time.

### Where Execution History Lives

The history browser is accessed from:
1. The Execution Runner panel → "History" tab
2. Site Detail → Scripts tab → right-click or long-press on a script
3. Site Detail → Verbs tab → "📜 History" button in the verb accordion

It is rendered as a panel within the existing page layout, not as a separate
page.

### ASCII Mockup: Execution History List

```
┌─ Execution History: fetch-page.js ─────────────────────────────────────────┐
│                                                                             │
│ Filter: [All ▼] | [Last hour] [Last day] [Last week] | Search: [______]   │
│                                                                             │
│ ┌─ #12  ✅ succeeded   2026-04-07 18:42:03   Duration: 0.38s ────────────┐ │
│ │ Input: { url: "https://news.ycombinator.com" }                         │ │
│ │ Output: { title: "Hacker News", links: 30 }                            │ │
│ │ Artifacts: page.html (42KB)                                             │ │
│ │                                                          [Re-run] [View] │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│ ┌─ #11  ❌ failed      2026-04-07 18:35:17   Duration: 30.01s ────────────┐ │
│ │ Input: { url: "https://example.com/slow" }                             │ │
│ │ Error: TIMEOUT - Script execution exceeded 30s limit                   │ │
│ │                                                          [Re-run] [View] │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│ ┌─ #10  ✅ succeeded   2026-04-07 17:12:44   Duration: 0.41s ────────────┐ │
│ │ Input: { url: "https://slashdot.org" }                                  │ │
│ │ Output: { title: "Slashdot", links: 25 }                               │ │
│ │ Artifacts: page.html (38KB)                                             │ │
│ │                                                          [Re-run] [View] │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│ ┌─ #9   ✅ succeeded   2026-04-06 22:01:05   Duration: 0.35s ────────────┐ │
│ │ Input: { url: "https://news.ycombinator.com" }                         │ │
│ │ Output: { title: "Hacker News", links: 28 }                            │ │
│ │ Artifacts: page.html (41KB)                                             │ │
│ │                                                          [Re-run] [View] │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│ Load more...                                                                │
└─────────────────────────────────────────────────────────────────────────────┘
```

### ASCII Mockup: Execution History Detail (Expanded Entry)

When the user clicks "View" on a history entry, the detail view replaces the
list or opens alongside it:

```
┌─ Execution #12 Detail ─────────────────────────────────────────────────────┐
│                                                                             │
│ Status: ✅ succeeded        Duration: 0.38s     Ran at: 2026-04-07 18:42  │
│ Script: scripts/fetch-page.js (v1, 16 lines)                               │
│                                                                             │
│ ┌─ Tabs ─────────────────────────────────────────────────────────────────┐ │
│ │ Input │ Output │ Console │ Artifacts │ Records │ Source Snapshot        │ │
│ └────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│ ┌─ Output ───────────────────────────────────────────────────────────────┐ │
│ │ {                                                                      │ │
│ │   "title": "Hacker News",                                             │ │
│ │   "linkCount": 30,                                                     │ │
│ │   "firstHeading": "Show HN: ..."                                      │ │
│ │ }                                                                      │ │
│ └────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│ [▶ Re-run with this input]  [▶ Re-run with modified input]                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

The "Source Snapshot" tab is important. It shows the exact script source that
was executed during this run. This is a frozen snapshot — even if the developer
has modified the file on disk since then, the snapshot shows the version that
actually ran. This is critical for reproducibility.

### History Data Model

Each execution history entry has this shape:

```typescript
interface ExecutionHistoryEntry {
  id: string;
  scriptPath: string;
  site: string;
  status: 'succeeded' | 'failed' | 'cancelled' | 'timeout';
  ctx: {
    site: string;
    now: string;
    workflow: { id: string; site: string; input: any };
    op: { id: string; input: any };
  };
  result: {
    data: any;
    logs: Array<{ timestamp: string; message: string }>;
    artifacts: Array<{
      name: string;
      kind: string;
      contentType: string;
      size: number;
    }>;
    records: Array<{ collection: string; key: string; data: any }>;
    emittedOps: any[];
    error: { code: string; message: string } | null;
  };
  sourceSnapshot: string;   // the script source at the time of execution
  sourceHash: string;       // SHA-256 hash of the source (for dedup/comparison)
  duration: number;         // milliseconds
  startedAt: string;        // ISO 8601
  completedAt: string;      // ISO 8601
  triggeredBy: 'manual' | 'op-replay' | 'workflow';
  tags: string[];           // user-defined tags for organization
}
```

### Where History Is Stored

Two options:

**Option A: Browser localStorage**

Pros: no backend changes needed, instant access, works offline.
Cons: limited storage (~5-10MB), lost on browser data clear, not shared across
devices.

**Option B: Backend SQLite table**

Pros: durable, shared across devices, queryable.
Cons: requires backend API changes, more complex implementation.

**Recommendation: Start with Option A (localStorage) for the MVP, migrate to
Option B when the backend execution API is stable.**

The migration path is straightforward: the frontend already uses RTK Query.
When the backend history endpoint exists, the frontend switches from localStorage
reads to RTK Query reads. The UI components do not change.

### YAML Widget Hierarchy: Execution History

```yaml
# ExecutionHistoryBrowser
component: Box
props: { sx: { display: flex, flexDirection: column, gap: 1.5 } }
children:
  # Filter bar
  - component: ExecutionHistoryFilterBar
    props:
      statusFilter: "${statusFilter}"
      timeRange: "${timeRange}"
      searchQuery: "${searchQuery}"
      onStatusFilterChange: "${setStatusFilter}"
      onTimeRangeChange: "${setTimeRange}"
      onSearchChange: "${setSearchQuery}"
    children: []

  # History list
  - component: Box
    props.sx: { display: flex, flexDirection: column, gap: 1 }
    children:
      - component: ExecutionHistoryList
        props:
          entries: "${filteredEntries}"
          loading: "${isLoading}"
          selectedId: "${selectedEntryId}"
          onSelect: "${handleSelectEntry}"
        children: []

  # Detail panel (conditional, shown when entry is selected)
  - component: ConditionalRender
    condition: "${!!selectedEntry}"
    children:
      - component: ExecutionHistoryDetail
        props:
          entry: "${selectedEntry}"
          onRerun: "${handleRerun}"
          onRerunModified: "${handleRerunModified}"
          onClose: "${handleCloseDetail}"
        children:
          - component: Tabs
            props: { value: "${detailTab}" }
            children:
              - component: TabPanel
                props: { value: "input" }
                children:
                  - component: JsonViewer
                    props: { data: "${selectedEntry.ctx}" }
              - component: TabPanel
                props: { value: "output" }
                children:
                  - component: JsonViewer
                    props: { data: "${selectedEntry.result.data}" }
              - component: TabPanel
                props: { value: "console" }
                children:
                  - component: ConsoleLogList
                    props: { entries: "${selectedEntry.result.logs}" }
              - component: TabPanel
                props: { value: "artifacts" }
                children:
                  - component: ArtifactList
                    props: { artifacts: "${selectedEntry.result.artifacts}" }
              - component: TabPanel
                props: { value: "source" }
                children:
                  - component: SyntaxHighlightedScriptViewer
                    props:
                      source: "${selectedEntry.sourceSnapshot}"
                      filename: "${selectedEntry.scriptPath}"
```

### Re-run from History

When the user clicks "Re-run with this input":

1. The system navigates to the Execution Runner panel.
2. The ctx input is populated from the history entry.
3. The script selector is set to the history entry's script.
4. The user clicks "Run" to execute.

When the user clicks "Re-run with modified input":

1. Same as above, but the ctx input editor is opened in editing mode.
2. The user can modify any field before running.

This creates a tight loop between history and execution: inspect past →
reproduce → modify → run → new history entry.

---

## Feature 5: REPL Console Widget

### Overview

The REPL (Read-Eval-Print Loop) console is an interactive JavaScript console
embedded in the scraper web UI. It provides a browser-DevTools-like experience
for the scraper JS runtime.

The REPL lets the developer:
1. Type JS expressions and see immediate results.
2. Access the scraper `ctx` object (or a simplified version of it).
3. Require modules and call functions.
4. See `console.log()` output and errors inline.
5. Execute multi-line scripts (not just single expressions).

### Conceptual Model

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│   Developer types in the input bar                          │
│         │                                                   │
│         ▼                                                   │
│   Frontend sends expression to backend                      │
│         │                                                   │
│         ▼                                                   │
│   Backend evaluates in a persistent JS VM                   │
│   (or creates one from the current ctx config)              │
│         │                                                   │
│         ▼                                                   │
│   Backend returns:                                          │
│   - result value (serialized)                               │
│   - console output (captured)                               │
│   - error (if thrown)                                       │
│         │                                                   │
│         ▼                                                   │
│   Frontend displays result in the console output area       │
│                                                             │
│   The VM state persists between evaluations:                │
│   - variables defined in one eval are available in the next │
│   - required modules remain loaded                          │
│   - ctx object is mutable                                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### ASCII Mockup: REPL Console

```
┌─ JS Console — hackernews ──────────────────────────────────────────────────┐
│ ┌─ Console Output ────────────────────────────────────────────────────────┐ │
│ │                                                                          │ │
│ │ ▶ const axios = require('axios')                       [18:42:01.001]  │ │
│ │   undefined                                            [18:42:01.050]  │ │
│ │                                                                          │ │
│ │ ▶ const url = "https://news.ycombinator.com"           [18:42:05.200]  │ │
│ │   undefined                                            [18:42:05.201]  │ │
│ │                                                                          │ │
│ │ ▶ const response = await axios.get(url)                [18:42:10.100]  │ │
│ │   { status: 200, headers: {...}, data: "<html>..." }   [18:42:10.530]  │ │
│ │                                                                          │ │
│ │ ▶ response.data.substring(0, 200)                      [18:42:15.300]  │ │
│ │   "<html><head><title>Hacker News</title>..."          [18:42:15.301]  │ │
│ │                                                                          │ │
│ │ ▶ ctx.log("Hello from the REPL!")                      [18:42:20.000]  │ │
│ │   [LOG] Hello from the REPL!                           [18:42:20.001]  │ │
│ │   undefined                                            [18:42:20.002]  │ │
│ │                                                                          │ │
│ │ ▶ ctx.writeArtifact("test.txt", "text", "text/plain",  [18:42:25.000]  │ │
│ │   "Hello world")                                                        │ │
│ │   undefined                                            [18:42:25.001]  │ │
│ │                                                                          │ │
│ │ ▶ const x = 1 / 0                                     [18:42:30.000]  │ │
│ │   Infinity                                             [18:42:30.001]  │ │
│ │                                                                          │ │
│ │ ▶ throw new Error("test error")                        [18:42:35.000]  │ │
│ │   ❌ Error: test error                                 [18:42:35.001]  │ │
│ │      at <eval>:1:7                                                      │ │
│ │      at scraperReplEval (repl.go:42)                                    │ │
│ │                                                                          │ │
│ └──────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│ ┌─ Input Bar ──────────────────────────────────────────────────────────────┐ │
│ │ > const result = await ctx.dep("op-123")                   [▶ Run]     │ │
│ └──────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│ [🔄 Reset VM]  [📋 Clear]  [⏏ Export Session]  VM: ● Ready  Uptime: 3m  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### REPL Components

The REPL console has four visual areas:

1. **Console output** (top, scrollable): Shows the history of evaluations.
   Each entry is either an input (prefixed with `▶`) or an output (indented).
   Console log entries are shown with `[LOG]` prefix. Errors are shown with
   `❌` prefix and red color.

2. **Input bar** (bottom, fixed): A text input (or textarea for multi-line)
   where the developer types JS. Press Enter or click "Run" to evaluate.

3. **Toolbar** (bottom strip): Action buttons for reset, clear, and export.

4. **Status bar** (bottom right): VM status indicator (Ready, Evaluating,
   Error), uptime counter.

### REPL Entry Points

The REPL is accessible from:

1. Site Detail → Scripts tab → "🔧 REPL" button in the script browser toolbar.
   Opens the REPL with the site's module environment pre-loaded.
2. Site Detail → Verbs tab → "🔧 REPL" button in the verb accordion.
   Opens the REPL with the verb's script pre-loaded and `ctx` initialized.
3. Execution Runner → "🔧 Open REPL" button.
   Opens the REPL with the execution ctx already set up.

### REPL vs Execution Runner: When to Use Which

```
┌─────────────────────────────────────────────────────────────────┐
│                    REPL Console                                  │
│ Use for:                                                        │
│  - Exploring the runtime and modules                            │
│  - Quick expression evaluation                                  │
│  - Testing small code fragments                                 │
│  - Debugging ctx values interactively                           │
│  - Iterating on logic incrementally                             │
│                                                                  │
│ Characteristics:                                                 │
│  - Persistent VM state across evaluations                       │
│  - No ctx input configuration needed (uses current VM state)    │
│  - Results are ephemeral (unless exported)                      │
│  - Best for: exploration, learning, quick tests                 │
├─────────────────────────────────────────────────────────────────┤
│                  Execution Runner                                │
│ Use for:                                                        │
│  - Running a full script from start to finish                   │
│  - Testing with specific ctx input                              │
│  - Comparing runs with different inputs                         │
│  - Reproducing a specific execution from history                │
│                                                                  │
│ Characteristics:                                                 │
│  - Fresh VM for each run                                        │
│  - Full ctx input configuration                                 │
│  - Results are captured and stored in history                   │
│  - Best for: validation, regression testing, comparison         │
└─────────────────────────────────────────────────────────────────┘
```

### REPL Backend API

The REPL requires a persistent server-side session. Two approaches:

**Option A: WebSocket-based REPL**

The frontend opens a WebSocket connection to the backend. Each message is a JS
expression to evaluate. The backend responds with the result.

```
Frontend                          Backend
   │                                │
   │  WS /api/v1/repl/connect       │
   │─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ▶│
   │                                │
   │  { "type": "eval",             │
   │    "code": "1 + 1" }          │
   │───────────────────────────────▶│
   │                                │
   │  { "type": "result",           │
   │    "value": 2 }               │
   │◀───────────────────────────────│
   │                                │
   │  { "type": "eval",             │
   │    "code": "const x = 42" }   │
   │───────────────────────────────▶│
   │                                │
   │  { "type": "result",           │
   │    "value": undefined }       │
   │◀───────────────────────────────│
   │                                │
```

**Option B: HTTP polling with session tokens**

Each eval is a POST request with a session ID. The backend maintains a VM pool
keyed by session ID.

```
Frontend                          Backend
   │                                │
   │  POST /api/v1/repl/session     │
   │───────────────────────────────▶│
   │  { "sessionId": "abc-123" }   │
   │◀───────────────────────────────│
   │                                │
   │  POST /api/v1/repl/eval        │
   │───────────────────────────────▶│
   │  { "sessionId": "abc-123",    │
   │    "code": "1 + 1" }          │
   │◀───────────────────────────────│
   │  { "result": 2 }              │
   │                                │
```

**Recommendation: Option A (WebSocket)** for real-time feel, with Option B as a
fallback if WebSocket infrastructure is not available. The scraper backend
already uses Watermill for event streaming, so WebSocket support is likely
straightforward.

### REPL Session Lifecycle

```
┌──────────┐     connect      ┌──────────┐
│  CLOSED  │─────────────────▶│  READY   │
└──────────┘                  └────┬─────┘
                                   │
                          user types code + Enter
                                   │
                                   ▼
                              ┌──────────┐
                              │ EVALUATING│
                              └────┬─────┘
                                   │
                        ┌──────────┼──────────┐
                        │          │          │
                   success      error      timeout
                        │          │          │
                        ▼          ▼          ▼
                   ┌─────────┐ ┌─────────┐ ┌─────────┐
                   │  READY  │ │  ERROR  │ │ TIMEOUT │
                   └────┬────┘ └────┬────┘ └────┬────┘
                        │          │          │
                        └──────────┼──────────┘
                                   │
                          user types next code
                                   │
                                   ▼
                              ┌──────────┐
                              │ EVALUATING│
                              └──────────┘
                                   │
                          ... (loop) ...
                                   │
                          user disconnects / resets
                                   │
                                   ▼
                              ┌──────────┐
                              │  CLOSED  │
                              └──────────┘
```

### REPL Console Log Capture

The backend must intercept `console.log()`, `console.warn()`, `console.error()`,
and `ctx.log()` calls made during evaluation and stream them back to the
frontend. This is done by:

1. Overriding the `console` object in the JS VM before evaluation.
2. Overriding `ctx.log` to write to the same capture buffer.
3. Including captured logs in the WebSocket response.

```javascript
// Backend pseudocode for log capture
vm.Set("console", {
  log:   (...args) => captureLog("log", args),
  warn:  (...args) => captureLog("warn", args),
  error: (...args) => captureLog("error", args),
  info:  (...args) => captureLog("info", args),
})
```

### YAML Widget Hierarchy: REPL Console

```yaml
# ReplConsoleWidget
component: Box
props:
  sx:
    display: flex
    flexDirection: column
    height: 100%
    minHeight: 400
    backgroundColor: "#1e1e2e"
    borderRadius: 1
    border: 1px solid
    borderColor: divider
    overflow: hidden
children:
  # Header bar
  - component: Box
    props:
      sx:
        display: flex
        justifyContent: space-between
        alignItems: center
        px: 2
        py: 1
        borderBottom: 1px solid
        borderColor: divider
    children:
      - component: Typography
        props: { variant: subtitle2, color: text.secondary, fontFamily: monospace }
        text: "JS Console — ${siteName}"
      - component: ReplStatusBar
        props:
          vmStatus: "${vmStatus}"
          uptime: "${uptime}"
          sessionId: "${sessionId}"
        children: []

  # Console output area (scrollable)
  - component: ReplOutputArea
    props:
      entries: "${consoleEntries}"
      autoScroll: true
    props.sx: { flex: 1, overflow: auto, px: 2, py: 1 }
    children:
      # Each entry is rendered based on type
      - component: ReplEntry
        condition: "${entry.type === 'input'}"
        props:
          prefix: "▶"
          text: "${entry.code}"
          timestamp: "${entry.timestamp}"
        props.sx: { color: text.primary, fontFamily: monospace }

      - component: ReplEntry
        condition: "${entry.type === 'output'}"
        props:
          prefix: "  "
          text: "${entry.value}"
          timestamp: "${entry.timestamp}"
        props.sx: { color: success.main, fontFamily: monospace }

      - component: ReplEntry
        condition: "${entry.type === 'log'}"
        props:
          prefix: "[LOG]"
          text: "${entry.message}"
          timestamp: "${entry.timestamp}"
        props.sx: { color: info.main, fontFamily: monospace }

      - component: ReplEntry
        condition: "${entry.type === 'error'}"
        props:
          prefix: "❌"
          text: "${entry.message}"
          stack: "${entry.stack}"
          timestamp: "${entry.timestamp}"
        props.sx: { color: error.main, fontFamily: monospace }

  # Input bar (fixed at bottom)
  - component: ReplInputBar
    props:
      value: "${inputText}"
      onChange: "${setInputText}"
      onSubmit: "${handleSubmit}"
      disabled: "${vmStatus === 'evaluating'}"
      placeholder: "Type JavaScript and press Enter..."
      multiline: true
      maxRows: 8
    props.sx:
      borderTop: 1px solid
      borderColor: divider
      px: 2
      py: 1
    children:
      - component: IconButton
        props: { onClick: "${handleSubmit}", title: "Run (Enter)" }
        children:
          - component: PlayArrowIcon

  # Toolbar (bottom strip)
  - component: ReplToolbar
    props:
      onReset: "${handleResetVm}"
      onClear: "${handleClearOutput}"
      onExport: "${handleExportSession}"
      vmStatus: "${vmStatus}"
    props.sx:
      display: flex
      justifyContent: space-between
      px: 2
      py: 0.5
      borderTop: 1px solid
      borderColor: divider
```

### REPL Multi-Line Input

For multi-line code (functions, if-blocks, etc.), the input bar expands to a
textarea. The user presses Shift+Enter for a new line, Enter to submit.

```
┌─ Input Bar (single line) ────────────────────────────────┐
│ > const x = 1 + 2                           [▶ Run]     │
└──────────────────────────────────────────────────────────┘

┌─ Input Bar (multi-line, expanded) ───────────────────────┐
│ > async function fetchAndParse(ctx) {                    │
│     const url = ctx.op.input.url;                        │
│     const response = await axios.get(url);               │
│     return response.data;                                │
│   }                                         [▶ Run]      │
│                                                          │
│ Shift+Enter for new line · Enter to run                  │
└──────────────────────────────────────────────────────────┘
```

### REPL Session Export

The "Export Session" button downloads the entire REPL session as a JSON file or
markdown document:

```json
{
  "site": "hackernews",
  "sessionId": "abc-123",
  "startedAt": "2026-04-07T18:42:00Z",
  "entries": [
    { "type": "input", "code": "const axios = require('axios')", "timestamp": "..." },
    { "type": "output", "value": "undefined", "timestamp": "..." },
    { "type": "input", "code": "const response = await axios.get('https://...')", "timestamp": "..." },
    { "type": "output", "value": "{ status: 200, ... }", "timestamp": "..." }
  ]
}
```

---

## Widget Hierarchy Reference (YAML DSL)

This section provides a consolidated reference of all new components and their
YAML widget hierarchies. The YAML DSL is a compressed notation that maps
directly to React/TSX component trees.

### DSL Conventions

```yaml
# A widget node represents a React component.
# Required fields:
#   component: the React component name (PascalCase)
# Optional fields:
#   props: an object of component props (JSX props)
#   props.sx: MUI sx prop (inline styles)
#   condition: a boolean expression for conditional rendering
#   text: inline text content (for simple text nodes)
#   children: a list of child widget nodes
#
# Special syntax:
#   ${variable}: a runtime value (props, state, or derived)
#   ${handler}: a callback function
#   component: ConditionalRender  -- renders children only if condition is true
#   component: TabPanel           -- a tab content panel
```

### New Components Summary

| Component | Purpose | Location |
|-----------|---------|----------|
| `SyntaxHighlightedScriptViewer` | Syntax-highlighted JS source viewer | `components/scripts/SyntaxHighlightedScriptViewer.tsx` |
| `ScriptViewerWithReload` | Script viewer + reload button wrapper | `components/scripts/ScriptViewerWithReload.tsx` |
| `VerbSourceSection` | Inline source view inside verb accordion | `components/sites/VerbSourceSection.tsx` |
| `ExecutionRunnerPanel` | Full execution runner page/panel | `components/execution/ExecutionRunnerPanel.tsx` |
| `ExecutionScriptSelector` | Script picker dropdown for execution | `components/execution/ExecutionScriptSelector.tsx` |
| `ExecutionInputEditor` | JSON editor for ctx input | `components/execution/ExecutionInputEditor.tsx` |
| `ExecutionActionBar` | Run/Cancel/Status bar | `components/execution/ExecutionActionBar.tsx` |
| `ExecutionResultPanel` | Tabbed result display | `components/execution/ExecutionResultPanel.tsx` |
| `ConsoleLogList` | List of console log entries | `components/execution/ConsoleLogList.tsx` |
| `LoadFromOpDialog` | Modal for loading ctx from a workflow op | `components/execution/LoadFromOpDialog.tsx` |
| `ExecutionHistoryBrowser` | List of past executions | `components/history/ExecutionHistoryBrowser.tsx` |
| `ExecutionHistoryEntry` | Single history entry card | `components/history/ExecutionHistoryEntry.tsx` |
| `ExecutionHistoryDetail` | Expanded detail view of a history entry | `components/history/ExecutionHistoryDetail.tsx` |
| `ExecutionHistoryFilterBar` | Filter bar for history list | `components/history/ExecutionHistoryFilterBar.tsx` |
| `ReplConsoleWidget` | Full REPL console | `components/repl/ReplConsoleWidget.tsx` |
| `ReplOutputArea` | Scrollable output area | `components/repl/ReplOutputArea.tsx` |
| `ReplEntry` | Single REPL input/output/error entry | `components/repl/ReplEntry.tsx` |
| `ReplInputBar` | Input textarea + Run button | `components/repl/ReplInputBar.tsx` |
| `ReplToolbar` | Reset/Clear/Export actions | `components/repl/ReplToolbar.tsx` |
| `ReplStatusBar` | VM status indicator | `components/repl/ReplStatusBar.tsx` |

### Component Directory Structure

```
web/src/components/
├── scripts/
│   ├── ScriptViewer.tsx              (MODIFY: replace with syntax highlighting)
│   ├── SyntaxHighlightedScriptViewer.tsx  (NEW)
│   └── ScriptViewerWithReload.tsx    (NEW)
├── sites/
│   ├── SiteVerbList.tsx              (MODIFY: add VerbSourceSection)
│   ├── SiteScriptBrowser.tsx         (MODIFY: use ScriptViewerWithReload)
│   └── VerbSourceSection.tsx         (NEW)
├── execution/                        (NEW directory)
│   ├── ExecutionRunnerPanel.tsx
│   ├── ExecutionScriptSelector.tsx
│   ├── ExecutionInputEditor.tsx
│   ├── ExecutionActionBar.tsx
│   ├── ExecutionResultPanel.tsx
│   ├── ConsoleLogList.tsx
│   └── LoadFromOpDialog.tsx
├── history/                          (NEW directory)
│   ├── ExecutionHistoryBrowser.tsx
│   ├── ExecutionHistoryEntry.tsx
│   ├── ExecutionHistoryDetail.tsx
│   └── ExecutionHistoryFilterBar.tsx
├── repl/                             (NEW directory)
│   ├── ReplConsoleWidget.tsx
│   ├── ReplOutputArea.tsx
│   ├── ReplEntry.tsx
│   ├── ReplInputBar.tsx
│   ├── ReplToolbar.tsx
│   └── ReplStatusBar.tsx
└── workflows/
    └── OpDetailDrawer.tsx            (MODIFY: use ScriptViewerWithReload)
```

---

## Backend API Endpoints Needed

This section summarizes all new backend API endpoints required by the features
in this design. The frontend cannot be implemented without these endpoints.

### Endpoints Summary

| Method | Endpoint | Purpose | Feature |
|--------|----------|---------|---------|
| `POST` | `/api/v1/sites/{site}/scripts/run` | Execute a script with provided ctx | Execution Runner |
| `GET` | `/api/v1/sites/{site}/scripts/history` | List execution history | History Browser |
| `GET` | `/api/v1/sites/{site}/scripts/history/{id}` | Get single history entry detail | History Browser |
| `WS` | `/api/v1/repl/connect` | Open REPL WebSocket session | REPL Console |
| `POST` | `/api/v1/repl/eval` | Evaluate JS in REPL session (HTTP fallback) | REPL Console |

### Endpoint Details

#### `POST /api/v1/sites/{site}/scripts/run`

Execute a script with a provided ctx input. This is the core of the execution
runner.

**Request:**

```json
{
  "scriptPath": "scripts/fetch-page.js",
  "ctx": {
    "site": "hackernews",
    "now": "2026-04-07T19:00:00Z",
    "workflow": {
      "id": "manual-run-001",
      "site": "hackernews",
      "name": "Manual Run",
      "status": "running",
      "input": {},
      "metadata": {}
    },
    "op": {
      "id": "manual-op-001",
      "workflowID": "manual-run-001",
      "site": "hackernews",
      "kind": "js",
      "queue": "default",
      "dedupKey": "",
      "metadata": { "script": "scripts/fetch-page.js" },
      "input": {
        "url": "https://news.ycombinator.com"
      }
    }
  },
  "options": {
    "timeout": 30000,
    "captureArtifacts": true,
    "captureRecords": true,
    "captureEmittedOps": true
  }
}
```

**Response (success):**

```json
{
  "status": "succeeded",
  "duration": 420,
  "data": { "title": "Hacker News", "linkCount": 30 },
  "logs": [
    { "timestamp": "2026-04-07T19:00:00.001Z", "message": "Fetching: https://..." },
    { "timestamp": "2026-04-07T19:00:00.420Z", "message": "Done, got 42KB" }
  ],
  "artifacts": [
    {
      "name": "page.html",
      "kind": "html",
      "contentType": "text/html",
      "size": 42000,
      "body": "<html>...</html>"
    }
  ],
  "records": [],
  "emittedOps": [],
  "error": null,
  "sourceHash": "sha256:abc123..."
}
```

**Response (error):**

```json
{
  "status": "failed",
  "duration": 1500,
  "data": null,
  "logs": [
    { "timestamp": "...", "message": "Fetching: https://..." }
  ],
  "artifacts": [],
  "records": [],
  "emittedOps": [],
  "error": {
    "code": "SCRIPT_ERROR",
    "message": "TypeError: Cannot read property 'title' of undefined",
    "stack": "at run (scripts/fetch-page.js:13:38)\n...",
    "line": 13,
    "column": 38
  },
  "sourceHash": "sha256:abc123..."
}
```

**Safety rules (same as SCRAPER-OP-DEBUGGER):**
- Does NOT create durable ops in the engine DB.
- Does NOT write to site DBs.
- Does NOT create real artifacts in the artifact store.
- All output is in-memory and returned in the response.
- Timeout is enforced server-side.

#### `GET /api/v1/sites/{site}/scripts/history`

List execution history entries for a site. Supports filtering and pagination.

**Query parameters:**
- `scriptPath` (optional): filter by script path
- `status` (optional): filter by status (`succeeded`, `failed`, etc.)
- `limit` (optional, default 50): max entries to return
- `offset` (optional, default 0): pagination offset
- `from` (optional): ISO 8601 timestamp, only entries after this time
- `to` (optional): ISO 8601 timestamp, only entries before this time

**Response:**

```json
{
  "entries": [
    {
      "id": "exec-012",
      "scriptPath": "scripts/fetch-page.js",
      "site": "hackernews",
      "status": "succeeded",
      "duration": 420,
      "startedAt": "2026-04-07T18:42:03Z",
      "completedAt": "2026-04-07T18:42:03Z",
      "triggeredBy": "manual",
      "ctxSummary": { "url": "https://news.ycombinator.com" },
      "resultSummary": { "title": "Hacker News", "linkCount": 30 },
      "error": null
    }
  ],
  "total": 12
}
```

Note: this returns summaries, not full ctx/result data. Full data is available
via the detail endpoint.

#### `GET /api/v1/sites/{site}/scripts/history/{id}`

Get the full detail of a single execution history entry, including the complete
ctx input, result data, logs, artifacts, and source snapshot.

#### `WS /api/v1/repl/connect`

Open a WebSocket connection for a REPL session.

**Query parameters:**
- `site` (required): the site context for the REPL
- `moduleId` (optional): pre-load a specific module

**Message format (client → server):**

```json
{ "type": "eval", "code": "const x = 1 + 2", "id": "eval-001" }
```

**Message format (server → client):**

```json
{ "type": "result", "id": "eval-001", "value": 3, "duration": 1 }
```

```json
{ "type": "log", "level": "info", "message": "Hello", "timestamp": "..." }
```

```json
{ "type": "error", "id": "eval-001", "message": "...", "stack": "..." }
```

```json
{ "type": "status", "status": "ready" }
```

---

## Phased Implementation Plan

This plan is ordered by priority (following the gap analysis) and by dependency
(simpler features before complex ones).

### Phase 1: Syntax Highlighting (Priority 1, 2-3 days)

**Goal:** Replace plain-text code rendering with syntax highlighting everywhere
scripts are shown.

**Steps:**
1. Install `prismjs` as a dependency in `web/package.json`.
2. Create `SyntaxHighlightedScriptViewer` component.
3. Add syntax highlighting theme tokens to `web/src/theme.ts`.
4. Replace `ScriptViewer` internals with `SyntaxHighlightedScriptViewer`.
5. Verify all existing surfaces (script browser, op detail drawer) now show
   highlighted code.
6. Add Storybook stories for the new component.

**Files:**
- `web/package.json` — add prismjs
- `web/src/components/scripts/ScriptViewer.tsx` — modify
- `web/src/theme.ts` — add syntax color tokens
- `web/src/components/scripts/SyntaxHighlightedScriptViewer.stories.tsx` — new

**Validation:**
- Manual: open Site Detail → Scripts tab, verify highlighted JS.
- Manual: open Op Detail Drawer → Script tab, verify highlighted JS.
- Automated: Storybook snapshot tests for the new component.

### Phase 2: Script Reload from Disk (Priority 1, 0.5-1 day)

**Goal:** Add a "Reload from Disk" button wherever scripts are displayed.

**Steps:**
1. Create `ScriptViewerWithReload` wrapper component.
2. Use RTK Query's `refetch` function for the reload action.
3. Add reload button to: SiteScriptBrowser, ScriptTab (op detail drawer).
4. Add loading/success/error states for the reload button.

**Files:**
- `web/src/components/scripts/ScriptViewerWithReload.tsx` — new
- `web/src/components/sites/SiteScriptBrowser.tsx` — modify
- `web/src/components/workflows/OpDetailDrawer.tsx` — modify

**Validation:**
- Manual: edit a script file on disk, click reload, see updated source.

### Phase 3: Verb Source Inline View (Priority 1, 1 day)

**Goal:** Show script source inside the verb accordion on Site Detail → Verbs tab.

**Steps:**
1. Create `VerbSourceSection` component.
2. Determine the script path for each verb (from verb metadata or convention).
3. Fetch script source on demand when the section is expanded.
4. Add "▶ Run with Input" and "📋 Copy Source" action buttons.

**Files:**
- `web/src/components/sites/VerbSourceSection.tsx` — new
- `web/src/components/sites/SiteVerbList.tsx` — modify

**Validation:**
- Manual: open Site Detail → Verbs tab, expand a verb, see source.
- Verify that verbs without associated scripts show a graceful empty state.

### Phase 4: Backend Execution API (Priority 2, 3-5 days)

**Goal:** Implement the `POST /api/v1/sites/{site}/scripts/run` endpoint.

**Steps:**
1. Create a new handler in `pkg/api/handlers/` for the execution endpoint.
2. Create a new service in `pkg/services/scriptexec/` that:
   - Accepts a script path and ctx input.
   - Creates an in-memory sandbox (no DB writes).
   - Reuses the existing `pkg/js/runtime/executor.go` with replay adapters.
   - Captures logs, artifacts, records, emitted ops.
   - Returns a result envelope.
3. Add timeout enforcement (default 30s, configurable).
4. Add request validation (valid JSON ctx, valid script path).
5. Add unit tests.

**Files:**
- `pkg/api/handlers/scriptexec.go` — new
- `pkg/services/scriptexec/service.go` — new
- `pkg/services/scriptexec/service_test.go` — new

**Validation:**
- Unit tests for successful execution, error execution, timeout.
- Manual: `curl` the endpoint with a real script and ctx.

### Phase 5: Execution Runner UI (Priority 2, 3-5 days)

**Goal:** Build the frontend execution runner panel.

**Steps:**
1. Create the `execution/` component directory.
2. Build `ExecutionRunnerPanel`, `ExecutionScriptSelector`, `ExecutionInputEditor`,
   `ExecutionActionBar`, `ExecutionResultPanel`, `ConsoleLogList`.
3. Add RTK Query endpoint for `POST /sites/{site}/scripts/run`.
4. Wire the "Run with Input" buttons from verb accordion and script browser.
5. Add the "Load from Op" dialog.
6. Add Storybook stories.

**Files:**
- All files in `web/src/components/execution/` — new
- `web/src/api/executionApi.ts` — new (RTK Query API for execution)
- `web/src/pages/SiteDetailPage.tsx` — modify (add Execution Runner tab)

**Validation:**
- Manual: select a script, provide ctx, click Run, see result.
- Verify error display for invalid input.
- Verify console log rendering.

### Phase 6: Execution History (Priority 3, 3-4 days)

**Goal:** Build the execution history browser with localStorage persistence.

**Steps:**
1. Create the `history/` component directory.
2. Define the `ExecutionHistoryEntry` TypeScript type.
3. Create a `useExecutionHistory` hook that reads/writes localStorage.
4. Build `ExecutionHistoryBrowser`, `ExecutionHistoryEntry`, `ExecutionHistoryDetail`,
   `ExecutionHistoryFilterBar`.
5. Wire "Re-run" buttons to navigate to the execution runner with pre-filled input.
6. Add source snapshot capture (store the script source at execution time).

**Files:**
- All files in `web/src/components/history/` — new
- `web/src/hooks/useExecutionHistory.ts` — new

**Validation:**
- Manual: run a script, check history list, expand entry, verify source snapshot.
- Manual: re-run from history, verify input is pre-filled.

### Phase 7: REPL Console (Priority 3, 5-7 days)

**Goal:** Build the interactive REPL console.

**Steps:**
1. Create the `repl/` component directory.
2. Implement WebSocket connection management (connect, reconnect, disconnect).
3. Build `ReplConsoleWidget`, `ReplOutputArea`, `ReplEntry`, `ReplInputBar`,
   `ReplToolbar`, `ReplStatusBar`.
4. Implement the backend WebSocket handler for REPL sessions.
5. Add session state management (VM persistence across evaluations).
6. Add multi-line input support (Shift+Enter for new line).
7. Add session export functionality.

**Files:**
- All files in `web/src/components/repl/` — new
- `web/src/api/replApi.ts` — new (WebSocket management)
- `pkg/api/handlers/repl.go` — new (backend WebSocket handler)

**Validation:**
- Manual: open REPL, type expressions, see results.
- Manual: verify multi-line input works.
- Manual: verify console.log capture.
- Manual: verify error display with stack traces.
- Manual: verify session export.

---

## Testing Strategy

### Unit Tests (Vitest)

For every new component:

1. **Rendering tests**: does the component render without crashing?
2. **Props tests**: does it display the correct data for given props?
3. **Interaction tests**: do buttons fire the correct callbacks?
4. **Edge case tests**: empty data, loading state, error state.

Example test pseudocode:

```typescript
// SyntaxHighlightedScriptViewer.test.tsx
describe('SyntaxHighlightedScriptViewer', () => {
  it('renders JS source with syntax highlighting', () => {
    render(<SyntaxHighlightedScriptViewer source="const x = 1" filename="test.js" />)
    expect(screen.getByText('const')).toHaveClass('token keyword')
  })

  it('shows line numbers', () => {
    const source = 'line1\nline2\nline3'
    render(<SyntaxHighlightedScriptViewer source={source} filename="test.js" />)
    expect(screen.getByText('1')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
  })

  it('highlights specified lines', () => {
    render(<SyntaxHighlightedScriptViewer source="a\nb\nc" filename="test.js" highlightLines={[2]} />)
    const line2 = screen.getByText('b').closest('.line')
    expect(line2).toHaveClass('highlighted')
  })
})
```

### Integration Tests (RTK Query + MSW)

For API-dependent components:

1. **Mock Service Worker (MSW)** handlers for the new endpoints.
2. Test that the execution runner calls the correct endpoint.
3. Test that history entries are displayed correctly from mock data.

### Storybook Stories

Every new component should have a `.stories.tsx` file with:
- Default state
- Loading state
- Error state
- Edge cases (empty data, very long source, etc.)

### Manual Smoke Tests

Per phase:

| Phase | Smoke Test |
|-------|-----------|
| 1 | Open script browser → see highlighted JS code |
| 2 | Edit script on disk → click reload → see updated source |
| 3 | Open verbs tab → expand verb → see source section |
| 4 | `curl POST /api/v1/sites/hackernews/scripts/run` → see result |
| 5 | Open execution runner → select script → provide ctx → run → see result |
| 6 | Run script → check history → expand entry → re-run |
| 7 | Open REPL → type expression → see result → export session |

---

## Risks and Open Questions

### Risks

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Prism.js bundle size | Slightly larger JS bundle (~20KB gzipped for JS language) | Acceptable for a developer tool. Can tree-shake unused languages. |
| Backend execution security | Arbitrary JS execution on the server | Sandbox the execution with timeout, memory limit, no DB writes. Rate-limit the endpoint. |
| WebSocket connection stability | REPL drops on network issues | Add reconnection logic with exponential backoff. Show connection status. |
| localStorage size limits | History entries may exceed 5-10MB | Implement entry count limits (e.g., max 100 entries per site). Compress entries. |
| REPL VM state leakage | Variables from one user's session leaking to another | Each session gets its own isolated JS VM. No shared state. |

### Open Questions

1. **Should the REPL support async/await natively?** The scraper JS runtime
   supports Promises (via goja). The REPL should auto-wrap expressions in
   `async` context so `await` works at the top level.

2. **Should execution history be stored per-user or per-site?** If the UI has
   authentication, history should be per-user. Currently the UI is unauthenticated,
   so per-site (stored in localStorage) is the MVP.

3. **Should the execution runner support `ctx.dep()` resolution?** The ctx object
   has a `dep(opID)` method that resolves dependency results. In the execution
   runner, the user provides a flat ctx. Should they also be able to provide
   dependency results? If so, the input format needs a `dependencies` field.

4. **What is the maximum artifact size that should be inlined in the execution
   response?** HTML pages can be 100KB+. Should large artifacts be truncated or
   streamed? Recommendation: set a 1MB threshold. Artifacts larger than that
   return only metadata, not the body.

5. **Should the REPL have autocomplete?** This would require the backend to
   expose a completion API. This is a nice-to-have, not an MVP requirement.

6. **Should the execution runner be available as a standalone page or only as
   a tab within Site Detail?** Recommendation: start as a tab, extract to a
   standalone page if it proves useful.

---

## References

### Frontend Files

| File | Path | Description |
|------|------|-------------|
| App | `web/src/App.tsx` | Top-level routing |
| API Types | `web/src/api/types.ts` | Shared TypeScript domain types |
| Catalog API | `web/src/api/catalogApi.ts` | RTK Query for sites/verbs/scripts |
| Workflow API | `web/src/api/workflowApi.ts` | RTK Query for workflows/ops/results |
| ScriptViewer | `web/src/components/scripts/ScriptViewer.tsx` | Current plain-text viewer |
| ScriptTab | `web/src/components/scripts/ScriptTab.tsx` | Script tab wrapper |
| SiteVerbList | `web/src/components/sites/SiteVerbList.tsx` | Verb accordion list |
| SiteScriptBrowser | `web/src/components/sites/SiteScriptBrowser.tsx` | Script file browser |
| OpDetailDrawer | `web/src/components/workflows/OpDetailDrawer.tsx` | Op detail drawer |
| SiteDetailPage | `web/src/pages/SiteDetailPage.tsx` | Site detail page |
| Theme | `web/src/theme.ts` | MUI theme configuration |

### Backend Files

| File | Path | Description |
|------|------|-------------|
| Engine Handlers | `pkg/api/handlers/engine.go` | Workflow/op/result HTTP handlers |
| Catalog Handlers | `pkg/api/handlers/catalog.go` | Site/verb/script HTTP handlers |
| Engine View Service | `pkg/services/engineview/service.go` | Workflow query business logic |
| Catalog Service | `pkg/services/catalog/service.go` | Site/verb/script business logic |
| JS Executor | `pkg/js/runtime/executor.go` | Core JS execution engine |
| JS Runner | `pkg/engine/runner/js.go` | Scheduler runner for JS ops |
| Engine Model | `pkg/engine/model/types.go` | Workflow/Op/Result Go types |
| API Types | `pkg/api/types/types.go` | API request/response Go types |

### Related Tickets

| Ticket | Description |
|--------|-------------|
| SCRAPER-OP-DEBUGGER | Workflow artifact browsing and per-op JS replay debugging |
| SCRAPER-JS-DEVENV | This ticket |

### External References

- Prism.js documentation: https://prismjs.com/
- MUI component library: https://mui.com/
- RTK Query documentation: https://redux-toolkit.js.org/rtk-query/overview
- goja JS engine: https://github.com/dop251/goja
- go-go-goja module system: (internal, see `go-go-goja` workspace)
