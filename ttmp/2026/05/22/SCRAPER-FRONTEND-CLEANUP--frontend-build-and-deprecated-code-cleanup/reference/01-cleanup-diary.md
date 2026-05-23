---
Title: Cleanup Diary
Ticket: SCRAPER-FRONTEND-CLEANUP
Status: active
Topics:
    - frontend
    - cleanup
    - typescript
    - storybook
DocType: reference
Intent: implementation-diary
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological diary for frontend build cleanup implementation."
Created: 2026-05-22
---

# Diary

## Goal

This diary records the cleanup session for scraper frontend build health, deprecated Storybook/mock code removal, and TypeScript cleanup.

## Step 1: Baseline frontend cleanup analysis

I started a new docmgr ticket dedicated to frontend cleanup, captured the current build failure log, searched for the failing symbols, and wrote a cleanup guide that groups the failures into practical cleanup phases. The goal is not just to make TypeScript quiet, but to remove stale story/mock references and make generated protobuf/API types the source of truth.

The initial evidence shows that `pnpm build` fails in `tsc -b` before Vite starts. The errors are concentrated in unused imports, stale Storybook stories, broken story fixture imports, incomplete MUI props, a factory that can return invalid `ArtifactSummary` objects, and runtime-event mock enum names that no longer exist after protobuf generation.

### Prompt Context

**User prompt (verbatim):** "make a new session for cleaning things up. Analyze the situation and create a cleanup guide, we want to get rid of deprecated code, confusing code, etc...

Then clean it up, commit at appropriate intervals and keep a diary"

**Assistant interpretation:** Create a new docmgr cleanup ticket, analyze the frontend build/code-health failures, write a concrete cleanup guide, then implement the cleanup in focused commits while updating a diary.

**Inferred user intent:** The user wants a clean, maintainable frontend build and a reviewable implementation trail, not ad-hoc one-off fixes.

**Commit (code):** N/A — planning and analysis step only.

### What I did

- Created docmgr ticket `SCRAPER-FRONTEND-CLEANUP`.
- Captured the current frontend build failure log:

```bash
cd scraper/web && pnpm build
```

- Saved evidence files:
  - `sources/01-frontend-build.log`
  - `sources/02-web-src-files.txt`
  - `sources/03-build-error-symbol-search.txt`
- Wrote the cleanup guide:
  - `analysis/01-frontend-cleanup-guide.md`
- Expanded `tasks.md` into phased implementation tasks.

### Why

- The previous session proved the runtime-event migration code was not the direct cause of the frontend build failure.
- A dedicated cleanup ticket prevents these broader build-health fixes from being mixed into the sessionstream runtime-event migration ticket.
- The guide gives concrete file locations and cleanup sketches before editing code.

### What worked

- The build failure reproduced consistently.
- The failures are small and localized enough for incremental cleanup commits.

### What didn't work

- Baseline `pnpm build` failed as expected. Representative errors:

```text
src/components/common/CodeViewPanel.tsx(145,18): error TS2769: No overload matches this call.
src/stories/msw/handlers.ts(2,38): error TS2307: Cannot find module '../api/types' or its corresponding type declarations.
src/test-utils/mockRuntimeEvents.ts(35,28): error TS2339: Property 'OP_COMPLETED' does not exist on type 'typeof RuntimeEventKind'.
```

### What I learned

- Most failures are stale story/test support code rather than product runtime code.
- `RuntimeEventKind` must be aligned with generated protobuf enum names: `WORKFLOW_CREATED`, `OP_LEASED`, `OP_SUCCEEDED`, `OP_FAILED`, and related current names.

### What was tricky to build

- The failures are mixed across production components and story/test helpers, so the cleanup needs to avoid casually changing product behavior while still deleting deprecated support code. I separated the work into low-risk TypeScript cleanup first, then fixture/mock cleanup.

### What warrants a second pair of eyes

- Whether story files should be included in the production build TypeScript project. For now, the cleanup assumes they are intentionally included and must compile.
- Whether runtime-event mocks should continue living in `test-utils` now that the runtime-event transport moved to sessionstream.

### What should be done in the future

- If stories remain part of `tsc -b`, add a lightweight rule/check that story helpers cannot import nonexistent API paths.

### Code review instructions

- Start with `analysis/01-frontend-cleanup-guide.md` to understand the planned cleanup surface.
- Compare the baseline build log in `sources/01-frontend-build.log` against final validation output.

### Technical details

Primary validation commands for this ticket:

```bash
cd scraper/web && pnpm build
cd scraper/web && pnpm test:unit -- --runInBand
```
