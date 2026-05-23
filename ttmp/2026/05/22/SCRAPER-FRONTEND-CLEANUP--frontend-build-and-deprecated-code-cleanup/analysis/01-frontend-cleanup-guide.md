---
Title: Frontend Cleanup Guide
Ticket: SCRAPER-FRONTEND-CLEANUP
Status: active
Topics:
    - frontend
    - cleanup
    - typescript
    - storybook
DocType: analysis
Intent: implementation-guide
Owners: []
RelatedFiles:
    - Path: web/src/api/workflowApi.ts
      Note: Production API file with unused type import in baseline build
    - Path: web/src/components/common/CodeViewPanel.tsx
      Note: Production component with MUI ToggleButton typing issue
    - Path: web/src/stories/msw/handlers.ts
      Note: Story MSW fixtures with stale relative API type imports
    - Path: web/src/test-utils/mockRuntimeEvents.ts
      Note: Runtime event mock factory with deprecated protobuf enum names
ExternalSources: []
Summary: Cleanup guide for frontend TypeScript, Storybook, fixture, and deprecated mock build failures.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Frontend Cleanup Guide

## Goal

Get the scraper frontend back to a clean `pnpm build` state while removing deprecated, stale, or confusing code. The cleanup should be low-risk: prefer deleting dead imports, aligning stories/fixtures with current component APIs, and replacing stale mock enum names with generated protobuf enum names.

## Current failure surface

Baseline command:

```bash
cd scraper/web && pnpm build
```

Result: `tsc -b` fails before Vite starts. The captured log is in:

```text
ttmp/2026/05/22/SCRAPER-FRONTEND-CLEANUP--frontend-build-and-deprecated-code-cleanup/sources/01-frontend-build.log
```

The failures cluster into five cleanup categories:

1. unused imports/constants/locals;
2. stale Storybook stories that no longer match component props/types;
3. stale MSW/story imports after the frontend API module layout changed;
4. fixture data that no longer satisfies current API types;
5. stale runtime-event mock enum names after protobuf-generated enum names became the source of truth.

## Runtime and maintenance map

The affected code is mostly developer-support surface area rather than production runtime flow:

- `web/src/api/workflowApi.ts` is production RTK Query API code.
- `web/src/components/**` includes production components plus Storybook stories.
- `web/src/stories/**` contains story fixtures and MSW handlers.
- `web/src/test-utils/mockRuntimeEvents.ts` is shared mock data for stories/tests.
- `web/src/pb/proto/**` contains generated protobuf TypeScript bindings and should drive runtime event enum usage.

The cleanup should avoid changing API payload contracts unless TypeScript proves a fixture is stale. Deprecated story/mock code should be removed or updated rather than shimmed.

## Issues and cleanup sketches

### 1. Unused imports and constants obscure real dependencies

Problem: Strict TypeScript reports several imported symbols and constants that are not used. These add visual noise and make it harder to see which modules actually depend on which APIs.

Where to look:

- `web/src/api/workflowApi.ts` — unused `WorkflowResultSummary` type import.
- `web/src/components/artifacts/ActiveFilterChips.tsx` — unused `FIELD_LABELS` constant.
- `web/src/components/results/ActiveResultFilterChips.tsx` — unused `FIELD_LABELS` constant.
- `web/src/components/results/ResultFilterBar.tsx` — unused React and MUI imports.
- `web/src/components/results/ResultsPanel.tsx` — unused `WorkflowOp` type import.

Example:

```ts
const FIELD_LABELS: Record<keyof ArtifactFilters, string> = {
  opId: 'Op',
  kind: 'Kind',
  contentType: 'Type',
  search: 'Search',
};
```

Why it matters: Dead symbols are small individually, but they train maintainers to ignore TypeScript warnings and hide real coupling.

Cleanup sketch:

```text
For each noUnusedLocals error:
  if the symbol has no behavioral role -> delete it;
  if it was intended for UI labels -> either wire it into chip rendering or delete it.
Prefer deletion for this pass.
```

### 2. Story files drifted from component APIs

Problem: Several Storybook files reference old component props or Storybook helper shapes.

Where to look:

- `web/src/components/common/AlertBanner.stories.tsx` passes `onDismiss`, but `AlertBannerProps` has `dismissible` and internal dismiss state only.
- `web/src/components/common/AppErrorBoundary.stories.tsx` exports a story named `Error`, shadowing the global `Error` constructor used inside `ThrowingChild`.
- `web/src/components/workflows/RuntimeEventTable.stories.tsx` uses `canvas.getAllByRole`, but the inferred canvas type does not expose that method.
- `web/src/components/results/ResultsTable.stories.tsx` references `STORY_WORKFLOW_ID` without importing it.

Example:

```tsx
<AlertBanner
  severity="error"
  message="7 ops failed in the last hour"
  action={{ label: 'View Failed Ops', onClick: () => alert('navigating…') }}
  onDismiss={() => setShow(false)}
/>
```

Why it matters: Stories are documentation. When they compile against stale APIs, they mislead future frontend work and block production builds because the repo compiles stories with TypeScript.

Cleanup sketch:

```text
AlertBanner story:
  remove unsupported onDismiss;
  demonstrate built-in dismiss button through the existing dismissible prop.

AppErrorBoundary story:
  rename exported story from Error to ErrorState;
  keep ThrowingChild using global Error.

RuntimeEventTable story:
  import within/canvas helpers from Storybook testing utilities if needed,
  or remove play interaction if it is not needed for static docs.

ResultsTable story:
  import STORY_WORKFLOW_ID from the fixture module.
```

### 3. Component prop/type errors indicate incomplete UI wiring

Problem: A few production component errors indicate not just dead code, but incomplete or unsafe UI code.

Where to look:

- `web/src/components/common/CodeViewPanel.tsx` renders `ToggleButton` without required `value` and keeps unused format-change code.
- `web/src/components/results/ResultPreviewPanel.tsx` can pass `unknown` into JSX via `Tooltip title={result.error.Message}` because API error shape is only partially narrowed by MUI/React types in this context.

Example:

```tsx
<ToggleButton
  key={f}
  selected={format === f}
  onClick={() => handleToggleFormat(f)}
>
  {FORMAT_LABELS[f]}
</ToggleButton>
```

Why it matters: Missing required props can break accessibility/state semantics. Rendering `unknown` as a React node can produce runtime surprises when API payloads are not exactly strings.

Cleanup sketch:

```tsx
<ToggleButtonGroup size="small" value={format} exclusive onChange={handleFormatChange}>
  {formats.map((f) => (
    <ToggleButton key={f} value={f}>...</ToggleButton>
  ))}
</ToggleButtonGroup>
```

For unknown data before rendering:

```ts
function stringValue(value: unknown): string {
  return typeof value === 'string' ? value : JSON.stringify(value);
}
```

### 4. Story fixtures import types through deprecated relative paths

Problem: `web/src/stories/msw/handlers.ts` imports `../api/types`, but the actual API types live at `web/src/api/types.ts`. From `web/src/stories/msw`, the relative path must go up two directories.

Where to look:

```ts
import type { ArtifactSummary } from '../api/types';
import type { WorkflowOp } from '../api/types';
import type { WorkflowResultSummary } from '../api/types';
```

Why it matters: Broken fixture imports prevent the build and are a sign that story helpers drifted during directory moves.

Cleanup sketch:

```ts
import type {
  ArtifactSummary,
  WorkflowOp,
  WorkflowResultSummary,
} from '../../api/types';
```

Also replace inline `import('../api/types')` type references with direct imported types.

### 5. Required API fixture fields became optional in factories

Problem: `createArtifactSummary` accepts `Partial<ArtifactSummary>` and spreads it after base defaults. That means callers can pass `{ previewable: undefined }`, making the returned object incompatible with `ArtifactSummary.previewable: boolean`.

Where to look:

- `web/src/stories/__fixtures__/factories.ts`
- `web/src/api/types.ts` (`ArtifactSummary.previewable: boolean`)

Example:

```ts
return {
  id: 'art-001',
  ...overrides,
};
```

Why it matters: Factories should guarantee valid current API objects. Letting required fields become undefined spreads stale/partial data through stories and tests.

Cleanup sketch:

```ts
const artifact = { ...defaults, ...overrides };
return {
  ...artifact,
  previewable: artifact.previewable ?? false,
};
```

### 6. Runtime-event mock factories use deprecated enum names

Problem: `web/src/test-utils/mockRuntimeEvents.ts` still references enum names that do not exist in generated protobuf TypeScript bindings.

Where to look:

```ts
RuntimeEventKind.WORKFLOW_STARTED
RuntimeEventKind.OP_DISPATCHED
RuntimeEventKind.OP_COMPLETED
RuntimeEventKind.OP_ERROR
```

Current generated names from `web/src/pb/proto/scraper/runtime/v1/events_pb.d.ts` include:

```ts
RuntimeEventKind.WORKFLOW_CREATED
RuntimeEventKind.OP_LEASED
RuntimeEventKind.OP_SUCCEEDED
RuntimeEventKind.OP_FAILED
RuntimeEventKind.LOG_LINE
```

Why it matters: Generated protobuf bindings are the source of truth after the sessionstream migration. Mock code using old names makes tests/stories unreliable and suggests nonexistent backend events.

Cleanup sketch:

```ts
const KINDS = [
  RuntimeEventKind.WORKFLOW_CREATED,
  RuntimeEventKind.OP_LEASED,
  RuntimeEventKind.OP_SUCCEEDED,
  RuntimeEventKind.OP_FAILED,
  RuntimeEventKind.LOG_LINE,
];
```

## Implementation phases

### Phase 1: Low-risk compile cleanup

- Delete unused imports and constants.
- Fix component story API drift.
- Fix `CodeViewPanel` toggle value handling.
- Fix simple React/MUI type mismatches.

### Phase 2: Fixtures and deprecated mock cleanup

- Correct MSW fixture imports.
- Normalize factory returns to satisfy required fields.
- Replace stale runtime-event enum names with generated protobuf enum names.

### Phase 3: Validation and docs

- Run `pnpm build`.
- Run `pnpm test:unit -- --runInBand`.
- Update the diary with exact errors and commits.
- Run docmgr doctor.

## Review guidance

Review this cleanup as a build-health pass, not a behavior rewrite. Start with files that had TypeScript errors, then confirm the final build is green.

Primary validation commands:

```bash
cd scraper/web && pnpm build
cd scraper/web && pnpm test:unit -- --runInBand
```
