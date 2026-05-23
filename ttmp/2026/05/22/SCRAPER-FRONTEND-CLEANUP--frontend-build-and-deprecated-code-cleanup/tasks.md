---
Title: Tasks
Ticket: SCRAPER-FRONTEND-CLEANUP
Status: active
Created: 2026-05-22
---

# Tasks

## Phase 0: Analyze and plan

- [x] 1. Capture the current `pnpm build` failure log and symbol inventory.
- [x] 2. Write the frontend cleanup guide with concrete locations, rationale, and cleanup sketches.
- [x] 3. Write the initial cleanup diary entry.
- [x] 4. Commit the planning artifacts.

## Phase 1: Low-risk TypeScript and story cleanup

- [x] 5. Remove unused imports/constants/locals reported by `tsc`.
- [x] 6. Update stale component stories to match current component APIs.
- [x] 7. Fix MUI/React type errors that do not change product behavior.
- [x] 8. Validate Phase 1 with `pnpm build` or record remaining failures.
- [x] 9. Commit Phase 1.

## Phase 2: Fixtures, mocks, and deprecated API cleanup

- [x] 10. Fix story fixture imports and remove deprecated relative API references.
- [x] 11. Align runtime-event mock factories with generated protobuf enum names.
- [x] 12. Ensure story artifact/result fixtures satisfy current API types without optional required fields.
- [x] 13. Validate Phase 2 with `pnpm build` and targeted frontend tests.
- [x] 14. Commit Phase 2.

## Phase 3: Final validation and handoff

- [x] 15. Run `pnpm test:unit -- --runInBand`.
- [x] 16. Run `pnpm build`.
- [x] 17. Update cleanup guide and diary with final results, failures, and review instructions.
- [x] 18. Run `docmgr doctor --ticket SCRAPER-FRONTEND-CLEANUP --stale-after 30`.
- [x] 19. Commit final docs.
