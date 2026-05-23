---
Title: Workflow UI cleanup and runtime event consolidation plan
Ticket: SCRAPER-CLEANUP-WORKFLOW-UI
Status: active
Topics:
    - scraper
    - frontend
    - architecture
    - cleanup
    - react
    - events
    - workflows
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Detailed plan for decomposing OpDetailDrawer and consolidating runtime-event rendering."
LastUpdated: 2026-04-07T16:05:00-04:00
WhatFor: "Guide a low-risk cleanup of the workflow detail UI."
WhenToUse: "Use when implementing or reviewing the component split."
---

# Workflow UI cleanup and runtime event consolidation plan

## Target layout

```text
web/src/components/workflows/
  OpDetailDrawer.tsx
  op-detail/
    OpResultTab.tsx
    OpArtifactsTab.tsx
    OpRuntimeTab.tsx
    OpScriptTab.tsx
    OpLogsTab.tsx
    helpers.ts
  RuntimeEventTable.tsx
```

`RuntimeEventList.tsx` should be removed after migration.

## Principles

- move code before redesigning code
- keep visible behavior stable during the first pass
- avoid maintaining two runtime-event renderers without a strong product reason

## Implementation order

1. split the drawer tab bodies into separate components
2. migrate callers to `RuntimeEventTable`
3. remove `RuntimeEventList.tsx`

## Validation

```bash
npm run build
```
