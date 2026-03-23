# Tasks

## Complete

- [x] Create the `SCRAPER-JS-SUBMIT-VERBS` ticket in `scraper/ttmp`
- [x] Write the primary design and implementation guide
- [x] Create the investigation diary

## Planned implementation phases

### Phase 1. Worker command foundation

- [x] Add `scraper worker run`
- [x] Add worker CLI flags for engine DB path, sites dir, worker ID, poll interval, and max workers
- [x] Reuse the default site registry and existing runner registry
- [x] Reuse the site DB provider hook so workers open site DBs on demand
- [x] Add command tests for worker help and non-destructive startup validation
- [x] Commit the worker-command milestone

### Phase 2. Site registry verb roots

- [x] Extend site definitions with `VerbsFS` and `VerbsRoot`
- [x] Add a `verbs/` tree to `js-demo`
- [x] Keep existing `scripts/` for execution-time op bodies
- [x] Wire default site registration so the new verb trees are visible at CLI startup
- [x] Commit the site-verb-registry milestone

### Phase 3. Scraper JS verb host

- [x] Add a scraper-specific host package for scanning site verbs and wrapping them as commands
- [x] Reuse `go-go-goja/pkg/jsverbs` for scanning and command-shape generation
- [x] Avoid the default plain-function invocation path in `pkg/jsverbs/runtime.go`
- [x] Define a submission runtime context with workflow creation and initial-op emission helpers
- [x] Normalize JS return values into a durable submission result envelope
- [x] Add tests for valid submission flows
- [x] Commit the JS-verb-host milestone

### Phase 4. js-demo migration

- [x] Add JS submission verb files for `js-demo` seed, item, and summary
- [x] Replace or bypass the handwritten Go `js-demo run ...` commands with JS-scanned commands
- [x] Ensure `--help` for discovered commands shows the expected fields and docs
- [x] Ensure the commands submit workflow state without running the whole pipeline inline
- [x] Commit the js-demo-submission-verbs milestone

### Phase 5. End-to-end worker validation

- [x] Start a worker process against a temporary engine DB
- [x] Submit a `js-demo` workflow through a JS-scanned command
- [x] Use engine status/admin commands to show queued work and eventual completion
- [x] Verify site DB records and durable results after the worker processes the queue
- [x] Add automated tests covering submission plus worker execution
- [x] Commit the end-to-end-validation milestone

### Phase 6. Ticket sync and delivery

- [x] Update the design guide if implementation diverges
- [x] Update the diary with each completed implementation step
- [x] Update changelog entries with commit references
- [x] Run `go test ./... -count=1`
- [x] Run `docmgr doctor --ticket SCRAPER-JS-SUBMIT-VERBS --stale-after 30`
- [x] Upload the ticket bundle to reMarkable
- [ ] Commit the ticket-sync milestone

## TODO

- [ ] Add tasks here
