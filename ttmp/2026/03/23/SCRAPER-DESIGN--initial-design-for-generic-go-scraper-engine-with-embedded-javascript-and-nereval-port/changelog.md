# Changelog

## 2026-03-23

- Initial workspace created
- Imported `/tmp/scraper.md` into the ticket sources as the primary design input
- Added the main design guide describing how the imported op/result architecture maps onto the current NEREVAL prototype and how to implement the Go/goja port in `scraper/`
- Added the investigation diary capturing the research path, commands, and design decisions

## 2026-03-23

Added the initial design guide and diary mapping the imported scraper architecture onto the current NEREVAL prototype and the local go-go-goja runtime.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/design-doc/01-generic-go-scraper-engine-and-nereval-port-design-guide.md — Primary design deliverable for the ticket
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/reference/01-investigation-diary.md — Chronological research log for the ticket


## 2026-03-23

Validated the ticket with docmgr doctor, seeded the local vocabulary, and uploaded the final document bundle to reMarkable at /ai/2026/03/23/SCRAPER-DESIGN.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/.docmgrignore — Ignored the raw imported source from doc validation so doctor only checks authored docs
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/changelog.md — Ticket changelog records validation and delivery
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/vocabulary.yaml — Seeded vocabulary needed for this repo's first ticket


## 2026-03-23

Revised the design so engine state lives in the engine DB while each site owns its own DB and can apply ordered SQL and JS migrations; prepared a v2 bundle for reMarkable delivery.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/design-doc/01-generic-go-scraper-engine-and-nereval-port-design-guide.md — Updated storage and migration architecture
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/reference/01-investigation-diary.md — Recorded the v2 architecture revision


## 2026-03-23

Expanded the ticket backlog into phased implementation work, bootstrapped the real `scraper` Go module and Glazed CLI, added embedded help docs, clarified `Lease` and `RetryPolicy` in the design guide, and added a repo-local `go.work` file so the new module builds in the local mono-workspace.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/go.work — Added a repo-local workspace linking `scraper`, `../glazed`, and `../go-go-goja`
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/go.mod — Created the first real module definition for the scraper repo
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/cmd/scraper/main.go — Added the CLI entrypoint
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/root.go — Added the Glazed root command with logging and help wiring
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc/topics/scraper-architecture-overview.md — Added the first embedded help entry
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/tasks.md — Replaced the one-shot checklist with phased build tasks
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/reference/01-investigation-diary.md — Recorded the bootstrap implementation step


## 2026-03-23

Added the phase-2 engine contracts: durable workflow/op/result types, store interfaces, runner interfaces, scheduler/config validation, and a site registry contract with package-level tests.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/model/types.go — Defined the durable engine data model
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/store.go — Added the first store interfaces
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/runner/runner.go — Added runner contracts and duplicate-safe runner registration
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/scheduler/scheduler.go — Added scheduler config validation
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/registry/registry.go — Added the site registration contract
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/reference/01-investigation-diary.md — Recorded the contract milestone
