# Tasks

## Planning

- [x] Create the `SCRAPER-PURE-JS-SITES` ticket
- [x] Confirm the current inconsistency between `js-demo` and the HN/Slashdot sites
- [x] Write the conversion plan and diary

## Implementation

- [x] Add `verbs/seed.js` and `verbs/extract_frontpage.js` to Hacker News
- [x] Add `verbs/seed.js` and `verbs/extract_frontpage.js` to Slashdot
- [x] Update Hacker News `site.go` to expose `VerbsFS` and `VerbsRoot`
- [x] Update Slashdot `site.go` to expose `VerbsFS` and `VerbsRoot`
- [x] Remove Hacker News `RegisterCLI`
- [x] Remove Slashdot `RegisterCLI`
- [x] Delete Hacker News `cli.go`
- [x] Delete Slashdot `cli.go`
- [x] Delete Hacker News `workflow.go`
- [x] Delete Slashdot `workflow.go`
- [x] Delete the now-unused site-specific HTTP runner helper in `pkg/sites/cliutil`

## Validation

- [x] Update and run site tests
- [x] Verify the generic JS submit-verb commands exist for both sites
- [ ] Decide whether any generic `--fixture` or local-run support still needs to be added afterward
