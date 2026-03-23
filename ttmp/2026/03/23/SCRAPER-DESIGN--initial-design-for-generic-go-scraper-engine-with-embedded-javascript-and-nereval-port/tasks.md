# Tasks

## Complete

- [x] Create the `SCRAPER-DESIGN` ticket in `scraper/ttmp`
- [x] Import `/tmp/scraper.md` into the ticket sources
- [x] Read the imported architecture sketch carefully
- [x] Review the earlier NEREVAL ticket documents in `2026-03-21--experiment-dom/ttmp`
- [x] Review the current `2026-03-21--experiment-dom/nereval/` prototype code
- [x] Review the relevant `go-go-goja/` runtime and module primitives
- [x] Write a detailed intern-oriented analysis, design, and implementation guide
- [x] Write the investigation diary
- [x] Relate the key evidence files to the ticket and primary design doc with `docmgr doc relate`
- [x] Run `docmgr doctor --ticket SCRAPER-DESIGN --stale-after 30` and fix vocabulary and metadata warnings
- [x] Dry-run the reMarkable bundle upload
- [x] Upload the bundle to reMarkable and verify the remote listing

## Planned Build Phases

### Phase 1. Repository bootstrap and CLI foundation

- [x] Create the root Go module and baseline repository layout in `scraper/`
- [x] Add `cmd/scraper/main.go` as the production CLI entrypoint
- [x] Add a root Cobra/Glazed command builder under `pkg/cmd/root.go`
- [x] Initialize Glazed logging on the root command with the standard logging section
- [x] Initialize the embedded Glazed help system with `go:embed`, section loading, and root help registration
- [x] Add at least one high-quality help entry explaining the scraper architecture and command layout
- [x] Add a `version` or equivalent low-risk smoke-test command so the CLI can be exercised early
- [x] Verify `go test ./...` and a simple CLI invocation both work
- [x] Record the bootstrap work in the implementation diary
- [x] Commit the CLI/bootstrap milestone

### Phase 2. Engine package skeleton

- [x] Create package boundaries for engine config, store, scheduler, runners, and site registry
- [x] Define the durable engine data model for ops, results, dependencies, leases, artifacts, and workflow runs
- [x] Add initial Go types for `OpSpec`, `OpResult`, retry policy, queue keys, and workflow metadata
- [x] Define store interfaces for enqueueing, leasing, completing, failing, and querying ops
- [x] Define runner interfaces for Go-backed op kinds and JS-backed op kinds
- [x] Define the site registration contract so sites can contribute scripts, migrations, help text, and commands
- [x] Add package-level tests for the pure type and interface contracts that can be validated without SQLite
- [x] Record the engine skeleton work in the implementation diary
- [x] Commit the engine skeleton milestone

### Phase 3. Engine database and migrations

- [x] Create the engine SQLite schema for workflows, ops, dependencies, results, artifacts, and leases
- [x] Add ordered SQL migrations for the engine database under `pkg/engine/migrations`
- [x] Add migration startup logic and engine schema version checks
- [x] Implement the first SQLite-backed engine store
- [x] Add tests covering migration application, fresh database bootstrap, and upgrade from an older schema version
- [x] Add CLI/admin visibility for checking engine DB health if needed
- [x] Record the engine DB and migration work in the implementation diary
- [x] Commit the engine DB milestone

### Phase 4. Site registry and per-site databases

- [ ] Create the site registry that resolves a site name to scripts, runtime modules, and migration sources
- [ ] Give each site its own SQLite database separate from the engine database
- [ ] Define the site database lifecycle and location rules
- [ ] Support ordered SQL site migrations
- [ ] Support ordered JS site migrations executed against the site DB through a narrow migration runtime
- [ ] Define the migration execution order when both `.sql` and `.js` migrations exist
- [ ] Add tests for site DB bootstrap, mixed SQL/JS migrations, and rerun/idempotency behavior
- [ ] Add CLI plumbing for explicit site migration execution
- [ ] Record the site DB and migration work in the implementation diary
- [ ] Commit the site DB milestone

### Phase 5. JavaScript runtime integration

- [ ] Build the go-go-goja runtime factory for scraper ops
- [ ] Register scraper-specific native modules for op context, dependency reads, records, artifacts, and site DB access
- [ ] Define the milestone-one JS API without `ctx.fetch()`
- [ ] Support loading site scripts from the filesystem or embedded sources
- [ ] Add structured conversion between Go values and JS result envelopes
- [ ] Add tests for script execution, emitted ops, record writes, artifact writes, and controlled runtime teardown
- [ ] Record the JS runtime integration work in the implementation diary
- [ ] Commit the JS runtime milestone

### Phase 6. Scheduler and worker loop

- [ ] Implement workflow creation and initial op enqueueing
- [ ] Implement dependency-aware op readiness checks
- [ ] Implement leasing, retries, backoff, and terminal failure handling
- [ ] Implement queue-key based concurrency and rate-domain control
- [ ] Implement durable result persistence including `emittedIDs`
- [ ] Add worker loop logging, metrics hooks, and structured progress events
- [ ] Add tests for fan-out, dependency completion, retry behavior, and resume semantics
- [ ] Record the scheduler work in the implementation diary
- [ ] Commit the scheduler milestone

### Phase 7. Generic HTTP scraper primitives

- [ ] Implement the first Go-backed HTTP scrape op runner
- [ ] Persist response body, metadata, and request diagnostics as artifacts/results
- [ ] Support request templates, headers, method selection, form payloads, and retry classification
- [ ] Add queue keys appropriate for shared HTTP rate domains
- [ ] Add fixture-driven tests for HTML fetch success, retryable errors, and non-retryable failures
- [ ] Record the HTTP primitive work in the implementation diary
- [ ] Commit the HTTP primitive milestone

### Phase 8. NEREVAL site port

- [ ] Create the NEREVAL site package and register it with the engine
- [ ] Port NEREVAL list-page fetch behavior into the new HTTP op model
- [ ] Port ASP.NET pagination and viewstate handling into explicit op/data flow
- [ ] Port list extraction into JS analysis scripts
- [ ] Port detail-page fetch and detail extraction into site scripts
- [ ] Move NEREVAL read models into the NEREVAL site database with migrations
- [ ] Define the first end-to-end NEREVAL workflow graph in the new engine
- [ ] Add fixture and integration tests comparing the new outputs with the prototype behavior
- [ ] Record the NEREVAL port work in the implementation diary
- [ ] Commit the NEREVAL port milestone

### Phase 9. CLI workflows and operator ergonomics

- [ ] Add user-facing commands to initialize state, run workflows, inspect workflows, inspect ops, and migrate sites
- [ ] Add Glazed help entries for workflow concepts, site registration, migrations, and NEREVAL usage
- [ ] Add consistent CLI logging and operator-friendly progress output
- [ ] Add smoke-test examples for the main commands
- [ ] Record the CLI workflow work in the implementation diary
- [ ] Commit the operator CLI milestone

### Phase 10. Validation and handoff

- [ ] Run full `go test ./...` validation
- [ ] Run a local end-to-end NEREVAL scrape against fixtures or a constrained live target
- [ ] Update the design doc if implementation diverges from the current architecture notes
- [ ] Update the changelog and diary with the final implementation sequence
- [ ] Run `docmgr doctor --ticket SCRAPER-DESIGN --stale-after 30`
- [ ] Upload an updated implementation bundle to reMarkable
