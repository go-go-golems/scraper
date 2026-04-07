# Tasks

## Complete

- [x] Create a new ticket for the site JavaScript CLI runner investigation
- [x] Study `go-go-goja/pkg/doc/08-jsverbs-example-overview.md`
- [x] Study the current `pkg/jsverbs` scan, binding, command, and runtime implementation
- [x] Study the current scraper site registry and scheduler-facing JS runtime
- [x] Compare the current exercise-site scripts with the jsverbs model
- [x] Write a detailed intern-oriented design and implementation guide
- [x] Write the investigation diary
- [x] Relate key files to the design doc and diary
- [x] Run `docmgr doctor --ticket SCRAPER-SITE-JSVERBS --stale-after 30`
- [x] Dry-run the reMarkable bundle upload
- [x] Upload the bundle to reMarkable and verify the remote listing

## Follow-up Implementation Phases

### Phase 1. Extend the site registry for CLI verbs

- [ ] Add explicit `VerbsFS` and `VerbsRoot` fields to `pkg/sites/registry/registry.go`
- [ ] Add optional shared-section and runtime-module hooks for site verbs
- [ ] Keep the current scheduler-facing `ScriptsFS` and `ScriptsRoot` unchanged

### Phase 2. Build the site-js registry loader

- [ ] Add a package that scans site verb files with `jsverbs.ScanFS(...)`
- [ ] Register scraper-owned shared sections
- [ ] Register scraper/site runtime modules for verb execution
- [ ] Add tests for scan failures and shared-section registration

### Phase 3. Add the CLI subtree

- [ ] Add `scraper site js list-sites`
- [ ] Add `scraper site js <site> list`
- [ ] Dynamically mount generated site verbs under `scraper site js <site> ...`
- [ ] Add help coverage for the new command tree

### Phase 4. Add runtime helpers

- [ ] Reuse preconfigured `site-db` and optional `scraper-db`
- [ ] Add a `site-env` helper module
- [ ] Add a fixture-loading helper module
- [ ] Add tests for these modules

### Phase 5. Add first verbs to exercise sites

- [ ] Add Hacker News verbs for fixture parsing and DB inspection
- [ ] Add Slashdot verbs for fixture parsing and DB inspection
- [ ] Reuse shared `lib/` helpers instead of copying parsing logic
- [ ] Add CLI tests for structured and text output modes

### Phase 6. Revisit NEREVAL after the smaller sites prove the pattern

- [ ] Decide which NEREVAL parsing and projection helpers should become verbs
- [ ] Keep durable workflow op scripts and CLI verbs as separate entrypoints
