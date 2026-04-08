# Tasks

## Step 1: Install dependencies

- [x] `npm install @uiw/react-codemirror @codemirror/lang-javascript @codemirror/lang-json js-yaml @types/js-yaml` in `web/`
- [x] `npm install @codemirror/lang-yaml`

## Step 2: Create CodeViewPanel component

- [x] Create `web/src/components/common/CodeViewPanel.tsx` — CodeMirror-based viewer with JSON/YAML toggle + copy button
- [x] Props: `data`, `label`, `defaultFormat` (`'yaml'` | `'json'`), `formats`, `maxHeight`
- [x] Read-only CodeMirror, syntax highlighted, collapsible
- [x] Copy button → `navigator.clipboard.writeText` + "Copied" button state feedback
- [x] Stories: JSON mode, YAML mode, with copy click

## Step 3: Wire CodeViewPanel into ResultPreviewPanel

- [x] Replace inline `JsonViewer` usages in `ResultPreviewPanel` with `CodeViewPanel`
- [x] Set `defaultFormat="yaml"`, `formats={['json', 'yaml']}`

## Step 4: Wire CodeViewPanel into ArtifactPreview

- [x] Replace `JsonViewer` in `ArtifactPreview` for JSON content
- [ ] Keep `<pre>` for HTML content (separate concern, deferred)

## Step 5: Upgrade ScriptViewer to CodeMirror

- [x] Replace raw `<pre>` in `ScriptViewer` with `@uiw/react-codemirror`
- [x] Use `@codemirror/lang-javascript`, read-only, line numbers, keep filename header
- [x] Style to match existing grey theme

## Step 6: Wire CodeViewPanel into OpResultTab

- [x] Replace `JsonViewer` in `OpResultTab` with `CodeViewPanel`

## Step 7: Storybook

- [x] `CodeViewPanel.stories.tsx`: JSON mode, YAML mode, with copy click
- [x] `ScriptViewer.stories.tsx`: JS syntax highlighted script
- [ ] Mark `JsonViewer` as deprecated in story file (deferred)

## Docs

- [x] Update implementation diary
