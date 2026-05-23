---
Title: Tasks
Ticket: SCRAPER-JS-DEVENV
Status: active
---

# Tasks

## Phase 1: Syntax Highlighting
- [ ] Install prismjs dependency in web/package.json
- [ ] Create SyntaxHighlightedScriptViewer component
- [ ] Add syntax highlighting theme tokens to theme.ts
- [ ] Replace ScriptViewer internals with highlighted version
- [ ] Add Storybook stories for SyntaxHighlightedScriptViewer
- [ ] Verify all surfaces show highlighted code (script browser, op detail drawer)

## Phase 2: Script Reload from Disk
- [ ] Create ScriptViewerWithReload wrapper component
- [ ] Add reload button to SiteScriptBrowser
- [ ] Add reload button to OpDetailDrawer Script tab
- [ ] Add loading/success/error states for reload button

## Phase 3: Verb Source Inline View
- [ ] Create VerbSourceSection component
- [ ] Modify SiteVerbList to include source section in accordion
- [ ] Add "Run with Input" and "Copy Source" action buttons

## Phase 4: Backend Execution API
- [ ] Create scriptexec handler (POST /api/v1/sites/{site}/scripts/run)
- [ ] Create scriptexec service with sandboxed execution
- [ ] Add timeout enforcement
- [ ] Add request validation
- [ ] Add unit tests

## Phase 5: Execution Runner UI
- [ ] Create execution/ component directory
- [ ] Build ExecutionRunnerPanel, ExecutionScriptSelector, ExecutionInputEditor
- [ ] Build ExecutionActionBar, ExecutionResultPanel, ConsoleLogList
- [ ] Add RTK Query endpoint for script execution
- [ ] Wire "Run with Input" buttons
- [ ] Build LoadFromOpDialog
- [ ] Add Storybook stories

## Phase 6: Execution History
- [ ] Create history/ component directory
- [ ] Define ExecutionHistoryEntry TypeScript type
- [ ] Create useExecutionHistory hook (localStorage)
- [ ] Build ExecutionHistoryBrowser, Entry, Detail, FilterBar
- [ ] Wire "Re-run" buttons
- [ ] Add source snapshot capture

## Phase 7: REPL Console
- [ ] Create repl/ component directory
- [ ] Implement WebSocket connection management
- [ ] Build ReplConsoleWidget, ReplOutputArea, ReplEntry, ReplInputBar, ReplToolbar
- [ ] Implement backend WebSocket handler for REPL sessions
- [ ] Add multi-line input support
- [ ] Add session export
