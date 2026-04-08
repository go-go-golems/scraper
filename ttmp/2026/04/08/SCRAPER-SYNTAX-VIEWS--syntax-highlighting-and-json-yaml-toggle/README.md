---
Title: "Syntax Highlighting — Script + JSON/YAML Views"
Ticket: SCRAPER-SYNTAX-VIEWS
Status: done
Topics:
    - frontend
    - ux
    - syntax-highlighting
    - codemirror
    - yaml
DocType: ticket
Intent: long-term
Owners: []
RelatedFiles:
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/common/JsonViewer.tsx"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/scripts/ScriptViewer.tsx"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/results/ResultPreviewPanel.tsx"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/artifacts/ArtifactPreview.tsx"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/artifacts/ArtifactPreviewPanel.tsx"
Summary: "Add syntax highlighting to script and data viewers: CodeMirror for the JS script view, and a JSON/YAML toggle with clipboard copy button for data views (op results, artifact bodies, etc.)."
LastUpdated: 2026-04-08T00:05:00-04:00
WhatFor: "Make the script viewer and data views readable and convenient to copy/paste from."
WhenToUse: "Use when iterating on the script viewer, JSON viewers, or data preview components."
---

# SCRAPER-SYNTAX-VIEWS: Syntax Highlighting + JSON/YAML Toggle

## Goal

Improve readability and copy/paste ergonomics across the script and data viewers:

1. **Script viewer**: CodeMirror with JavaScript syntax highlighting, line numbers, even if read-only for now
2. **Data viewers** (JSON views in result/artifact previews): JSON/YAML toggle (YAML default), syntax highlighting, clipboard copy button

## What's in scope

### 1. Script viewer — CodeMirror

**Current**: `ScriptViewer` renders raw `<pre><code>` with line numbers as styled spans. No syntax highlighting.

**Goal**: CodeMirror with JavaScript syntax highlighting, line numbers, read-only.

**Component**: `web/src/components/scripts/ScriptViewer.tsx`

**Implementation options**:
- `@uiw/react-codemirror` + `@codemirror/lang-javascript` — React wrapper, easy integration
- `@monaco-editor/react` — Monaco editor (VS Code engine), but heavier
- Plain `@codemirror/view` + `@codemirror/state` — full manual setup, most control

**Recommendation**: `@uiw/react-codemirror` — good balance of features, simple API, no heavy dependencies. `ScriptViewer` already has the right shape (filename header + scrollable code area).

**Note**: Script editing is explicitly deferred — this is read-only highlighting only.

---

### 2. Data viewers — JSON/YAML toggle + copy button

**Current**: `JsonViewer` + `ArtifactPreview` render `<pre>` blocks with raw `JSON.stringify` output. No toggle, no copy button, no YAML option.

**Goal**: Each data view gets:
- A header bar with: [label] [YAML | JSON toggle] [Copy button]
- Syntax-highlighted output (JSON mode or YAML mode)
- Clipboard copy on click

**Affected components**:
- `JsonViewer.tsx` — used in `ResultPreviewPanel`, `ArtifactPreview`, `OpResultTab`
- `ArtifactPreview.tsx` — inline JSON/HTML/text rendering
- `OpResultTab.tsx` — op result data display

**Implementation**:

```tsx
<CodeViewPanel
  label="Result"
  data={data}
  defaultFormat="yaml"        // YAML default as requested
  formats={['json', 'yaml']} // show both toggle buttons
  onCopy={() => toast('Copied')}
/>
```

`CodeViewPanel` handles:
1. Toggle between JSON and YAML output
2. Syntax highlighting via `@uiw/react-codemirror` (JSON/YAML language support)
3. Copy to clipboard via `navigator.clipboard.writeText`
4. Collapsible panel (reuse the existing collapse logic from `JsonViewer`)

**YAML library**: `js-yaml` — already likely present, or add it.

---

## Screen: Script viewer with CodeMirror

```
╔════════════════════════════════════════════════════════════════════╗
║ extract.js                                          [readonly]   ║
╠════════════════════════════════════════════════════════════════════╣
║  1 │ function extract({ page, record }) {                       ║
║  2 │   const items = page.querySelectorAll('.athing');           ║
║  3 │   return items.map(item => ({                              ║  ← JS syntax highlighted
║  4 │     url:    item.querySelector('a').href,                  ║
║  5 │     title:  item.querySelector('a').textContent,            ║
║  6 │   }));                                                    ║
║  7 │ }                                                         ║
╚════════════════════════════════════════════════════════════════════╝
```

## Screen: Result preview with YAML/JSON toggle

```
╔═══════════════════════════════════════════════════════════════════════════════╗
║ Op: extract          Kind: js         Status: succeeded              [×]   ║
║ Records: 30    Artifacts: 0    Completed: 2h ago                                  ║
╠═══════════════════════════════════════════════════════════════════════════════╣
║  [Result ▼]  [YAML │ JSON]                                [📋 Copy]         ║
╠═══════════════════════════════════════════════════════════════════════════════╣
║ stories:                                                                  YAML highlighted
║   - id: 12345                                                             ║
║     url: https://example.com/1                                            ║
║     title: Show HN: A cool project                                         ║
║   - id: 12346                                                             ║
║     url: https://example.com/2                                            ║
║     title: Another story                                                   ║
║ nextPage: /news?p=2                                                       ║
╚═══════════════════════════════════════════════════════════════════════════════╝
```

## Screen: JSON mode

```
╔═══════════════════════════════════════════════════════════════════════════════╗
║  [Result ▼]  [YAML │ JSON]                                [📋 Copy]         ║
╠═══════════════════════════════════════════════════════════════════════════════╣
║ {                                                                       JSON highlighted
║   "stories": [                                                           ║
║     { "id": 12345, "url": "https://example.com/1",                       ║
║       "title": "Show HN: A cool project" },                               ║
║     ...                                                                  ║
║   ],                                                                     ║
║   "nextPage": "/news?p=2"                                               ║
║ }                                                                       ║
╚═══════════════════════════════════════════════════════════════════════════════╝
```

---

## Implementation order

### Step 1: Install dependencies

```bash
cd web
npm install @uiw/react-codemirror @codemirror/lang-javascript @codemirror/lang-json js-yaml @types/js-yaml
```

Add to `preview.tsx` mock store if needed (unlikely for read-only).

### Step 2: Refactor JsonViewer → CodeViewPanel

1. Create `web/src/components/common/CodeViewPanel.tsx`
2. Accept `data`, `label`, `defaultFormat`, `formats` props
3. Use `@uiw/react-codemirror` in read-only mode with JSON/YAML language
4. Add toggle between `json` and `yaml` format
5. Add "Copy" button using `navigator.clipboard.writeText`
6. Preserve existing collapse/expand logic from `JsonViewer`

### Step 3: Wire CodeViewPanel into ResultPreviewPanel

1. Replace the inline `JsonViewer` usages in `ResultPreviewPanel` with `CodeViewPanel`
2. Set `defaultFormat="yaml"`, `formats={['json', 'yaml']}`

### Step 4: Wire CodeViewPanel into ArtifactPreview

1. `ArtifactPreview` currently uses `JsonViewer` directly
2. Replace with `CodeViewPanel` for JSON content; keep `<pre>` for HTML (syntax highlighting of HTML is a separate concern)

### Step 5: Upgrade ScriptViewer to CodeMirror

1. Replace the raw `<pre>` rendering in `ScriptViewer` with `@uiw/react-codemirror`
2. Use `@codemirror/lang-javascript` language extension
3. Keep read-only, keep line numbers, keep filename header
4. Style the editor to match the existing grey theme (`backgroundColor: '#f5f5f5'`)

### Step 6: Wire CodeViewPanel into OpResultTab

1. `OpResultTab` uses `JsonViewer` for the result data display
2. Replace with `CodeViewPanel` — same pattern as ResultPreviewPanel

### Step 7: Storybook stories

1. `CodeViewPanel.stories.tsx`: JSON mode, YAML mode, with copy toast
2. `ScriptViewer.stories.tsx`: JS syntax highlighted, multi-line script
3. Update `JsonViewer.stories.tsx` to mark as deprecated / delegate to `CodeViewPanel`

---

## Non-goals

- Script editing (read-only only for now)
- YAML input / YAML editing
- Live-editing JSON views
- Monaco Editor (heavier than needed for read-only)
