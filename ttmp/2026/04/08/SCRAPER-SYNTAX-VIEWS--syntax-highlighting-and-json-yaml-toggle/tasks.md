# Tasks

## Step 1: Install dependencies

- [ ] `npm install @uiw/react-codemirror @codemirror/lang-javascript @codemirror/lang-json js-yaml @types/js-yaml` in `web/`

## Step 2: Create CodeViewPanel component

- [ ] Create `web/src/components/common/CodeViewPanel.tsx` — CodeMirror-based viewer with JSON/YAML toggle + copy button
- [ ] Props: `data`, `label`, `defaultFormat` (`'yaml'` | `'json'`), `formats`, `maxHeight`
- [ ] Read-only CodeMirror, syntax highlighted, collapsible
- [ ] Copy button → `navigator.clipboard.writeText` + toast confirmation
- [ ] Stories: JSON mode, YAML mode, copy click

## Step 3: Wire CodeViewPanel into ResultPreviewPanel

- [ ] Replace inline `JsonViewer` usages in `ResultPreviewPanel` with `CodeViewPanel`
- [ ] Set `defaultFormat="yaml"`, `formats={['json', 'yaml']}`

## Step 4: Wire CodeViewPanel into ArtifactPreview

- [ ] Replace `JsonViewer` in `ArtifactPreview` for JSON content
- [ ] Keep `<pre>` for HTML content (separate concern)

## Step 5: Upgrade ScriptViewer to CodeMirror

- [ ] Replace raw `<pre>` in `ScriptViewer` with `@uiw/react-codemirror`
- [ ] Use `@codemirror/lang-javascript`, read-only, line numbers, keep filename header
- [ ] Style to match existing grey theme

## Step 6: Wire CodeViewPanel into OpResultTab

- [ ] Replace `JsonViewer` in `OpResultTab` with `CodeViewPanel`

## Step 7: Storybook

- [ ] `CodeViewPanel.stories.tsx`: JSON mode, YAML mode, with copy toast
- [ ] `ScriptViewer.stories.tsx`: JS syntax highlighted script
- [ ] Mark `JsonViewer` as deprecated in story file

## Docs

- [ ] Update implementation diary
