# Tasks

## Analysis And Design

- [x] Create the ticket workspace.
- [x] Review the responsibilities inside `OpDetailDrawer.tsx`.
- [x] Confirm that `RuntimeEventTable` should become the single runtime-event presentation component.
- [x] Write the workflow UI cleanup design and target component layout.
- [x] Record the investigation diary.

## Drawer Decomposition

- [x] Split `OpDetailDrawer.tsx` into a shell plus tab-focused subcomponents.
- [x] Extract shared helpers into a small helper module.
- [x] Keep prop contracts stable during the first pass.
- [x] Keep tab behavior stable during the first pass.

## Runtime Event Consolidation

- [x] Migrate remaining `RuntimeEventList` usages to `RuntimeEventTable`.
- [x] Extract shared formatting helpers if needed.
- [x] Remove `RuntimeEventList.tsx`.
- [x] Remove duplicated formatting logic after migration.

## Validation

- [x] Run relevant frontend tests.
- [x] Run `npm run build` once the current build noise is addressed or isolated.
- [x] Run `docmgr doctor --ticket SCRAPER-CLEANUP-WORKFLOW-UI --stale-after 30`.
