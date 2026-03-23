---
Title: Investigation diary
Ticket: SCRAPER-DESIGN
Status: active
Topics:
    - scraping
    - go
    - goja
    - javascript
    - nereval
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../2026-03-21--experiment-dom/nereval/worker.js
      Note: Primary runtime behavior examined during the research pass
    - Path: ../../../../../../../go-go-goja/engine/factory.go
      Note: Primary go-go-goja runtime composition file reviewed during the research pass
    - Path: pkg/cmd/site_test.go
      Note: Built-in site migration smoke test used in the exercise-site milestone
    - Path: pkg/sites/hackernews/site_test.go
      Note: End-to-end Hacker News workflow test used in Step 12
    - Path: pkg/sites/jsdemo/cli.go
      Note: Step 13 adds the runnable CLI demo flow
    - Path: pkg/sites/jsdemo/site.go
      Note: Step 13 adds the new built-in pure-JS exercise site
    - Path: pkg/sites/jsdemo/site_test.go
      Note: Step 13 validates the pure-JS workflow end to end
    - Path: pkg/sites/slashdot/site_test.go
      Note: End-to-end Slashdot workflow test used in Step 12
    - Path: ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/sources/local/scraper.md
      Note: Imported source investigated during the diary step
ExternalSources:
    - local:scraper.md
Summary: Chronological diary of the SCRAPER-DESIGN research and implementation work, including the exercise-site milestone before the NEREVAL port.
LastUpdated: 2026-03-23T14:24:00-04:00
WhatFor: Record the research steps, commands, decisions, and rough edges encountered while producing the initial design guide.
WhenToUse: Use when reviewing how the initial design doc was produced or when continuing the implementation work later.
---




# Investigation diary

## Goal

Capture the exact research and implementation work used to create the new Go/goja scraper engine, validate it on smaller exercise sites, and prepare for the NEREVAL port.

## Step 1: Create the ticket, import the sketch, and connect it to the existing NEREVAL evidence

This step created the new `SCRAPER-DESIGN` ticket in `scraper/ttmp`, imported the user-provided `/tmp/scraper.md`, and then traced the current implementation knowledge back to the earlier NEREVAL prototype and the local `go-go-goja` runtime. The main output of the step is the detailed design guide in `design-doc/01-generic-go-scraper-engine-and-nereval-port-design-guide.md`.

The practical aim was not only to summarize the imported note. The aim was to make the imported note useful by grounding it in the current `nereval/` implementation and by translating it into a concrete file-level build plan for `scraper/`.

### Prompt Context

**User prompt (verbatim):** "# AGENTS.md instructions for /home/manuel/workspaces/2026-03-23/js-scraper

<INSTRUCTIONS>
## JavaScript REPL (Node)
- Use `js_repl` for Node-backed JavaScript with top-level await in a persistent kernel.
- `js_repl` is a freeform/custom tool. Direct `js_repl` calls must send raw JavaScript tool input (optionally with first-line `// codex-js-repl: timeout_ms=15000`). Do not wrap code in JSON (for example `{\"code\":\"...\"}`), quotes, or markdown code fences.
- Helpers: `codex.cwd`, `codex.homeDir`, `codex.tmpDir`, `codex.tool(name, args?)`, and `codex.emitImage(imageLike)`.
- `codex.tool` executes a normal tool call and resolves to the raw tool output object. Use it for shell and non-shell tools alike. Nested tool outputs stay inside JavaScript unless you emit them explicitly.
- `codex.emitImage(...)` adds one image to the outer `js_repl` function output each time you call it, so you can call it multiple times to emit multiple images. It accepts a data URL, a single `input_image` item, an object like `{ bytes, mimeType }`, or a raw tool response object with exactly one image and no text. It rejects mixed text-and-image content.
- `codex.tool(...)` and `codex.emitImage(...)` keep stable helper identities across cells. Saved references and persisted objects can reuse them in later cells, but async callbacks that fire after a cell finishes still fail because no exec is active.
- Request full-resolution image processing with `detail: \"original\"` only when the `view_image` tool schema includes a `detail` argument. The same availability applies to `codex.emitImage(...)`: if `view_image.detail` is present, you may also pass `detail: \"original\"` there. Use this when high-fidelity image perception or precise localization is needed, especially for CUA agents.
- Example of sharing an in-memory Playwright screenshot: `await codex.emitImage({ bytes: await page.screenshot({ type: \"jpeg\", quality: 85 }), mimeType: \"image/jpeg\", detail: \"original\" })`.
- Example of sharing a local image tool result: `await codex.emitImage(codex.tool(\"view_image\", { path: \"/absolute/path\", detail: \"original\" }))`.
- When encoding an image to send with `codex.emitImage(...)` or `view_image`, prefer JPEG at about 85 quality when lossy compression is acceptable; use PNG when transparency or lossless detail matters. Smaller uploads are faster and less likely to hit size limits.
- Top-level bindings persist across cells. If a cell throws, prior bindings remain available and bindings that finished initializing before the throw often remain usable in later cells. For code you plan to reuse across cells, prefer declaring or assigning it in direct top-level statements before operations that might throw. If you hit `SyntaxError: Identifier 'x' has already been declared`, first reuse the existing binding, reassign a previously declared `let`, or pick a new descriptive name. Use `{ ... }` only for a short temporary block when you specifically need local scratch names; do not wrap an entire cell in block scope if you want those names reusable later. Reset the kernel with `js_repl_reset` only when you need a clean state.
- Top-level static import declarations (for example `import x from \"./file.js\"`) are currently unsupported in `js_repl`; use dynamic imports with `await import(\"pkg\")`, `await import(\"./file.js\")`, or `await import(\"/abs/path/file.mjs\")` instead. Imported local files must be ESM `.js`/`.mjs` files and run in the same REPL VM context. Bare package imports always resolve from REPL-global search roots (`CODEX_JS_REPL_NODE_MODULE_DIRS`, then cwd), not relative to the imported file location. Local files may statically import only other local relative/absolute/`file://` `.js`/`.mjs` files; package and builtin imports from local files must stay dynamic. `import.meta.resolve()` returns importable strings such as `file://...`, bare package names, and `node:...` specifiers. Local file modules reload between execs, while top-level bindings persist until `js_repl_reset`.
- Avoid direct access to `process.stdout` / `process.stderr` / `process.stdin`; it can corrupt the JSON line protocol. Use `console.log`, `codex.tool(...)`, and `codex.emitImage(...)`.
</INSTRUCTIONS><environment_context>
  <cwd>/home/manuel/workspaces/2026-03-23/js-scraper</cwd>
  <shell>zsh</shell>
  <current_date>2026-03-23</current_date>
  <timezone>America/New_York</timezone>
</environment_context>
Create a new docmgr ticket for the initial design of our scraper in scraper/. Import the document /tmp/scraper.md by doing docmgr import file --file /tmp/scraper.md --ticket ... and then read the NEREVEAL-APP ticket in 2026-03-21--experiment-dom/ which was an earlier prototype for a scraper that I now want to port to go and make more generic and flexible by using go-go-goja/ and the goja js engine to write a lot of the scraping itself.

Read the imported scraper.md carefully, and then write an analysis document of how it fits with the current nereval scraper and how we could implement our nereval scraper with the new primitives.

Create a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new `docmgr` ticket inside `scraper/`, import `/tmp/scraper.md`, read the earlier NEREVAL prototype docs and code, then write a detailed intern-oriented design and implementation guide for the future Go/goja engine and NEREVAL port, update the ticket artifacts, validate them, and upload the bundle to reMarkable.

**Inferred user intent:** Establish a design-first foundation for the `scraper/` repository so implementation can start with a clear architecture, explicit file plan, and grounded understanding of how the old NEREVAL prototype maps into the new generic engine.

**Commit (code):** N/A — documentation-only work, no git commit created

### What I did

- Created the ticket:
  - `docmgr ticket create-ticket --ticket SCRAPER-DESIGN --title "Initial design for generic Go scraper engine with embedded JavaScript and nereval port" --topics scraping,go,goja,javascript,nereval,architecture`
- Added the primary design doc and diary doc:
  - `docmgr doc add --ticket SCRAPER-DESIGN --doc-type design-doc --title "Generic Go scraper engine and nereval port design guide"`
  - `docmgr doc add --ticket SCRAPER-DESIGN --doc-type reference --title "Investigation diary"`
- Imported the user-provided source exactly as requested:
  - `docmgr import file --ticket SCRAPER-DESIGN --file /tmp/scraper.md`
- Read the imported `scraper.md` in detail, focusing on:
  - the architecture clarification sections,
  - the persisted `OpSpec` and `OpResult` sketches,
  - the Go `Engine` / `Store` / `Runner` API sketch,
  - the JS `ctx` API and example emitted-op flows.
- Read the earlier NEREVAL docs and the current prototype code:
  - `2026-03-21--experiment-dom/ttmp/.../NEREVAL-APP/...`
  - `2026-03-21--experiment-dom/ttmp/.../NEREVAL-QUEUE/...`
  - `2026-03-21--experiment-dom/ttmp/.../NEREVAL-REVIEW/...`
  - `2026-03-21--experiment-dom/nereval/fetch.js`
  - `2026-03-21--experiment-dom/nereval/extract.js`
  - `2026-03-21--experiment-dom/nereval/db.js`
  - `2026-03-21--experiment-dom/nereval/worker.js`
  - `2026-03-21--experiment-dom/nereval/run.js`
  - `2026-03-21--experiment-dom/nereval/REPORT.md`
- Read the local `go-go-goja` runtime and module infrastructure:
  - `go-go-goja/engine/factory.go`
  - `go-go-goja/engine/runtime.go`
  - `go-go-goja/engine/runtime_modules.go`
  - `go-go-goja/engine/module_roots.go`
  - `go-go-goja/modules/common.go`
  - `go-go-goja/modules/database/database.go`
- Inspected the current `scraper/` repository layout to confirm it is still scaffold-oriented and therefore needs a design-first build plan.
- Wrote the primary design guide and updated ticket index/tasks/changelog around it.

### Why

- The imported architecture sketch is useful, but by itself it does not tell an engineer where the current implementation knowledge lives or how to start building.
- The NEREVAL prototype contains real constraints that the new engine must preserve, especially ASP.NET form-state sequencing, durable work/resume behavior, and query-friendly projections.
- `go-go-goja` already contains most of the runtime composition patterns needed for the new engine, so the design should anchor on that instead of inventing an entirely new runtime model.

### What worked

- The `docmgr` workflow worked cleanly for ticket creation, doc creation, and source import.
- The imported `scraper.md` contained enough architecture and API detail to serve as a direct reference for the op/result model and the JS runtime contract.
- The current NEREVAL code is structured enough that its core concerns are easy to map to the new engine:
  - acquisition,
  - extraction,
  - storage,
  - queueing,
  - orchestration.
- The local `go-go-goja` repository already exposes:
  - runtime builder/factory composition,
  - runtime-scoped module registration,
  - event-loop lifecycle,
  - script-relative `require()` roots,
  - the existing native module pattern.

### What didn't work

- I initially searched for `go-go-goja` at the wrong path and hit:
  - `rg: /home/manuel/workspaces/go-go-goja: IO error for operation on /home/manuel/workspaces/go-go-goja: No such file or directory (os error 2)`
- That was a path assumption mistake. The correct repository path in this workspace is `./go-go-goja`.
- I also attempted `git -C /home/manuel/workspaces/2026-03-23/js-scraper status ...` while verifying ticket state and got:
  - `fatal: not a git repository (or any of the parent directories): .git`
- That was harmless; the outer workspace is not a git root even though `scraper/` itself contains a `.git`.

### What I learned

- The imported architecture is best understood as a formalization of the prototype, not as a separate competing design.
- The NEREVAL prototype already proves the need for:
  - durable op state,
  - queue-keyed rate control,
  - explicit dependency flow,
  - artifact retention,
  - projection-writing separated from work leasing.
- `go-go-goja` is already opinionated in the right direction: explicit composition, runtime-scoped modules, and clean lifecycle management.
- The biggest conceptual simplification for the port is to replace the prototype's special-case `detail_queue` and `viewstates` tables with general op/result/input semantics, while keeping normalized NEREVAL projections as derived read models.

### What was tricky to build

- The hardest part was deciding where to draw the line between "generic engine" and "NEREVAL-specific behavior."
- The prototype currently mixes those concerns in `db.js` and `worker.js`, so the document needed to separate them without losing the pragmatic lessons from the working code.
- A second tricky point was the JS fetch boundary. The imported sketch allows async JS and optional fetching, but if that is interpreted too loosely it destroys scheduler control. The design doc therefore makes a strict recommendation that `ctx.fetch()` remains scheduler-backed.

### What warrants a second pair of eyes

- The recommendation to turn NEREVAL viewstate into ordinary op input rather than a dedicated cache table is structurally cleaner, but it should be reviewed against any future desire to skip into page ranges without replaying the chain.
- The decision to keep one SQLite file for both engine state and NEREVAL projections in milestone one is pragmatic, but worth revisiting if write/read pressure grows.
- The proposed first-class op envelope adds `WorkflowID`, `ParentID`, and `Site` on top of the imported sketch; those additions are sensible, but the exact minimum durable contract should still be reviewed before implementation starts.

### What should be done in the future

- Implement the actual Go module in `scraper/` using the phased plan from the design doc.
- Add a dedicated runtime-scoped `records` or `scraperdb` module instead of relying on the generic `modules/database` module for primary engine behavior.
- Create HTML fixtures from the prototype immediately when implementation starts.
- Decide whether browser-backed scrape ops are explicitly deferred or included as empty extension points in milestone one.

### Code review instructions

- Start with the new design doc:
  - `scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/design-doc/01-generic-go-scraper-engine-and-nereval-port-design-guide.md`
- Then compare the evidence files in this order:
  - `2026-03-21--experiment-dom/nereval/fetch.js`
  - `2026-03-21--experiment-dom/nereval/extract.js`
  - `2026-03-21--experiment-dom/nereval/db.js`
  - `2026-03-21--experiment-dom/nereval/worker.js`
  - `go-go-goja/engine/factory.go`
  - `go-go-goja/engine/runtime_modules.go`
- Validate that the design preserves the prototype's actual constraints:
  - ASP.NET form-state chaining,
  - queue/rate semantics,
  - durable resume behavior,
  - normalized projections.

### Technical details

- Ticket created at:
  - `scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port`
- Imported source path:
  - `scraper/ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/sources/local/scraper.md`
- Key commands run:
  - `docmgr status --summary-only`
  - `docmgr import file --help`
  - `docmgr doc list --ticket SCRAPER-DESIGN`
  - `rg -n "NEREVAL|NEREVEAL|nereval|nereveal" 2026-03-21--experiment-dom -S`
  - `nl -ba <file> | sed -n '<range>'`

## Step 2: Revise the design around site-owned databases and migrations

After the first draft, the design was tightened so each site gets its own database and owns its own schema evolution. The engine keeps its own generic DB for workflow state, results, and artifacts, while a site like NEREVAL gets its own `nereval.db` plus ordered SQL and JS migrations. This changed the earlier recommendation that one SQLite file was sufficient for the first version.

The important architectural effect is that site-owned record layouts no longer need to be represented as top-level engine schema. Instead, the engine provides stable generic storage and migration hooks, and the site decides how much structure it wants in its own DB.

### Prompt Context

**User prompt (verbatim):** "actually, give each site its own db, and let it either do its own migrations, but also just execute sql migrations on that site specific db.

COntinue updating your design document accordingly and upload as a v2 to the remarkable tablet"

**Assistant interpretation:** Revise the design doc so engine state and site state live in separate databases, support both SQL and JS migrations for each site DB, and then publish a new v2 bundle to reMarkable.

**Inferred user intent:** Make schema ownership explicitly application-level so sites can evolve their own record layouts without routing everything through engine-level schema changes.

**Commit (code):** N/A — documentation-only work, no git commit created

### What I did
- Updated the main design doc to split storage into:
  - engine DB,
  - one DB per site.
- Added a dedicated section describing site-owned SQL and JS migrations.
- Updated the proposed file layout to include:
  - `pkg/engine/migrations/*.sql`
  - `sites/nereval/migrations/*.sql`
  - `sites/nereval/migrations/*.js`
  - `pkg/sites/nereval/site_db.go`
  - `pkg/sites/nereval/projectors.go`
- Updated the phased implementation plan so site DB lifecycle and migrations are an explicit milestone before the generic HTTP runner.
- Updated testing guidance to include SQL and JS migration tests for the site DB.

### Why
- This keeps engine correctness schema separate from application/read-model schema.
- It gives sites real control over their own layouts and indexes.
- It makes SQL-only schema changes easy while still allowing JS-based logical backfills when needed.

### What worked
- The existing design structure adapted cleanly to the two-database model.
- The migration story becomes more coherent with a dedicated site migration runtime instead of treating all schema as engine-owned.

### What didn't work
- N/A

### What I learned
- The earlier “single SQLite file is pragmatic” guidance was too weak for the ownership model the user wants.
- The right split is not “generic records or app schema”; it is “generic engine schema plus optional app-owned schema in a separate DB.”

### What was tricky to build
- The main sharp edge was deciding where JS migrations should run. The resolution is to run them in a dedicated migration runtime bound to the site DB, not inside the ordinary workflow runtime.

### What warrants a second pair of eyes
- Whether projectors should always write directly to the site DB during op completion, or whether some sites should prefer derived projection ops.
- Whether site migrations should run automatically only at workflow startup or also via a standalone CLI command.

### What should be done in the future
- Add a dedicated `scraper site migrate <site>` command if manual migration control becomes useful.
- Define a stable migration-history table format for site DBs before implementing the runner.

### Code review instructions
- Read the updated storage and migration sections in:
  - `design-doc/01-generic-go-scraper-engine-and-nereval-port-design-guide.md`
- Focus on:
  - the store layer split,
  - the site migration contract,
  - the revised phased implementation plan,
  - the updated risks/open questions.

### Technical details
- The revised design introduces:
  - engine DB migrations managed in Go,
  - site DB migrations managed by ordered SQL and JS files,
  - a dedicated migration runtime with a narrow DB-only API.

## Step 3: Bootstrap the actual scraper CLI and convert the ticket into a phased execution backlog

This step moved the work from design-only into implementation. The immediate goal was to give `scraper/` a real Go module, a Glazed-based CLI entrypoint, embedded help docs, and a small smoke-test command. In parallel, the ticket task list was expanded from “research complete” into an actual phased build plan so the implementation work can be tracked against concrete milestones.

The result is a working `scraper` binary skeleton with Glazed logging and help wired correctly, plus a first-pass test and help-topic smoke check. I also had to add a repo-local `scraper/go.work` file so the new module participates in the same local workspace as `../glazed` and `../go-go-goja` without depending on an outer workspace file.

### Prompt Context

**User prompt (verbatim):** "Cool, add a detailed list of tasks to the ticket on how to build this, using the glazed framework for CLI (don't forget to properly initialize the help system and logging, see skills), and decompose the task into a set of phases. 

Then work on the tasks one by one committing at appropriate intervals and keeping a detailed diary as you work."

**Assistant interpretation:** Convert the ticket into a detailed phased backlog, then start implementing the repository in that order, beginning with Glazed CLI/bootstrap work, and keep an implementation diary plus focused commits along the way.

**Inferred user intent:** Turn the design ticket into an execution guide and begin actual repo construction immediately instead of leaving the work at documentation level.

**Commit (code):** `7876934` — Bootstrap scraper CLI and design ticket

### What I did

- Read the Glazed authoring and help-entry guidance to anchor the bootstrap on real project patterns instead of a guessed setup.
- Read example root command and doc-loading implementations from existing Go repos that use the same libraries.
- Expanded `tasks.md` into a phased implementation plan covering:
  - repository bootstrap,
  - engine package skeleton,
  - engine DB and migrations,
  - site registry and per-site DBs,
  - JS runtime integration,
  - scheduler and worker loop,
  - HTTP primitives,
  - NEREVAL port,
  - operator CLI,
  - final validation and handoff.
- Added the first actual Go code in `scraper/`:
  - `go.mod`
  - `cmd/scraper/main.go`
  - `pkg/cmd/root.go`
  - `pkg/cmd/version.go`
  - `pkg/doc/doc.go`
  - `pkg/doc/topics/scraper-architecture-overview.md`
  - `pkg/cmd/root_test.go`
- Wired the root command with:
  - `logging.InitLoggerFromCobra` in `PersistentPreRunE`
  - `logging.AddLoggingSectionToRootCommand`
  - `help.NewHelpSystem()`
  - `doc.AddDocToHelpSystem(...)`
  - `help_cmd.SetupCobraRootCommand(...)`
- Added a simple `version` subcommand so the CLI can be exercised without having to implement the workflow engine first.
- Ran:
  - `gofmt -w ...`
  - `go mod tidy`
  - `go test ./...`
  - `go run ./cmd/scraper version`
  - `go run ./cmd/scraper help scraper-architecture-overview`
- Fixed a workspace integration issue by adding a repo-local `go.work` file that includes:
  - `.`
  - `../glazed`
  - `../go-go-goja`
- Updated the design doc with a short glossary section clarifying `Lease` and `RetryPolicy`.
- Marked the completed phase 1 bootstrap tasks in `tasks.md`.

### Why

- The repo had CI and release scaffolding, but no actual module or binary entrypoint.
- The user explicitly asked for Glazed-based CLI initialization with help and logging done correctly, so the first code needed to establish that convention early.
- A small smoke-test command lets us validate the CLI wiring before the scheduler, DB, and runtime pieces exist.
- The `go.work` change was necessary so local development treats `scraper`, `glazed`, and `go-go-goja` as one workspace.

### What worked

- The Glazed integration pattern from other repos transferred cleanly:
  - `PersistentPreRunE` for logger init,
  - a logging section on the root command,
  - embedded help topics loaded through `helpSystem.LoadSectionsFromFS`,
  - root help registration through `help_cmd.SetupCobraRootCommand`.
- `go test ./...` passed after the workspace was fixed.
- The CLI smoke tests worked:
  - `go run ./cmd/scraper version` returned `dev`
  - `go run ./cmd/scraper help scraper-architecture-overview` rendered the embedded help page correctly.
- The phased ticket checklist now gives the implementation an explicit order, which was missing from the earlier doc-only state.

### What didn't work

- The first `go test ./...` attempt failed with:
  - `pattern ./...: directory prefix . does not contain modules listed in go.work or their selected dependencies`
- Cause:
  - the active workspace did not include the new `scraper` module
- Resolution:
  - add a repo-local `scraper/go.work`
  - rerun the test and smoke commands

### What I learned

- The repo was even closer to a template shell than the earlier design pass suggested. The first meaningful milestone really is a workspace/bootstrap milestone, not yet an engine milestone.
- A repo-local `go.work` is the safer choice here because it makes the `scraper` repo self-contained instead of relying on an outer workspace file.
- The design doc benefits from a short glossary. `Lease` and `RetryPolicy` are obvious if you build schedulers often, but not obvious enough for the intended intern audience.

### What was tricky to build

- The main practical question was how much to build before the first commit. I kept the milestone intentionally narrow: just enough to establish the repository conventions and verify that the CLI and embedded help work.
- I also had to avoid over-committing to engine package structure before phase 2, because phase 1 is about operator-facing entrypoints and repo shape, not scheduler semantics yet.

### What warrants a second pair of eyes

- The module path currently assumes `github.com/go-go-golems/scraper`, which is consistent with the repo scaffolding but should still be confirmed against the eventual remote repository name.
- The help topic structure is intentionally minimal; once real subcommands exist, the help tree should likely grow into command-specific guides and architecture references.

### What should be done in the future

- Commit this bootstrap milestone.
- Start phase 2 by defining the core engine package boundaries and durable types before implementing any SQLite schema.
- Add more explicit CLI groups once there are real workflow, site, and migration commands to hang under them.

### Code review instructions

- Review the new CLI files first:
  - `scraper/cmd/scraper/main.go`
  - `scraper/pkg/cmd/root.go`
  - `scraper/pkg/cmd/version.go`
  - `scraper/pkg/doc/doc.go`
  - `scraper/pkg/doc/topics/scraper-architecture-overview.md`
  - `scraper/pkg/cmd/root_test.go`
- Confirm that the root command follows the intended conventions:
  - logger initialized in `PersistentPreRunE`
  - logging flags added on the root command
  - embedded help sections loaded and registered
  - smoke-test command present and tested
- Then review the ticket tracking updates:
  - `tasks.md`
  - the glossary addition in the design doc
  - this diary entry

### Technical details

- Concrete commands used during the bootstrap:
  - `rg -n "help_cmd\\.SetupCobraRootCommand|AddLoggingSectionToRootCommand|InitLoggerFromCobra|LoadSectionsFromFS|BuildCobraCommandFromCommand" /home/manuel/code/wesen -g'*.go'`
  - `sed -n '1,220p' /home/manuel/code/wesen/go-go-golems/devctl/cmd/devctl/main.go`
  - `sed -n '1,220p' /home/manuel/code/wesen/go-go-golems/devctl/pkg/doc/doc.go`
  - `sed -n '1,220p' /home/manuel/code/wesen/go-go-golems/devctl/pkg/doc/topics/devctl-user-guide.md`
  - `gofmt -w ...`
  - `go mod tidy`
  - `go test ./...`
  - `go run ./cmd/scraper version`
  - `go run ./cmd/scraper help scraper-architecture-overview`

## Step 4: Define the engine contracts before implementing storage

This step established the package boundaries and durable type contracts for the engine before any SQLite implementation was written. The goal was to lock down the shape of workflows, ops, leases, retries, results, runners, stores, scheduler configuration, and site registration so the next implementation steps can build against stable interfaces instead of inventing them ad hoc while writing the database layer.

The output is a phase-2 skeleton under `pkg/engine/` and `pkg/sites/registry/` with lightweight validation tests. No database or runtime logic was added yet; this milestone is intentionally about contract clarity.

### Prompt Context

**User prompt (verbatim):** "Cool, add a detailed list of tasks to the ticket on how to build this, using the glazed framework for CLI (don't forget to properly initialize the help system and logging, see skills), and decompose the task into a set of phases. 

Then work on the tasks one by one committing at appropriate intervals and keeping a detailed diary as you work."

**Assistant interpretation:** Continue from the bootstrap milestone into the next planned phase, implementing the engine package skeleton and documenting the work in the ticket diary.

**Inferred user intent:** Move steadily through the phased plan instead of stopping after the CLI bootstrap.

**Commit (code):** `163d52a` — Add engine contracts and site registry

### What I did

- Added package boundaries for the early engine surface:
  - `pkg/engine/config`
  - `pkg/engine/model`
  - `pkg/engine/store`
  - `pkg/engine/runner`
  - `pkg/engine/scheduler`
  - `pkg/sites/registry`
- Defined the core durable model types in `pkg/engine/model/types.go`:
  - workflow IDs and site names
  - workflow status
  - op status
  - dependency edges
  - retry policy and retry state
  - lease data
  - op spec
  - op result
  - record writes
  - artifact writes
  - op error
- Added a first-pass config model in `pkg/engine/config/config.go` for:
  - engine DB path
  - sites directory
  - worker identity and worker timings
  - shared HTTP timeout and user-agent settings
- Added store interfaces in `pkg/engine/store/store.go` for:
  - workflow creation and status updates
  - op enqueueing and lookup
  - op leasing and lease heartbeats
  - completion and failure recording
  - result lookup for dependency reads
- Added runner interfaces and a runner registry in `pkg/engine/runner/runner.go`:
  - `Runner`
  - `RunContext`
  - `DependencyResolver`
  - `SiteDatabase`
  - duplicate-safe runner registration
- Added scheduler config validation in `pkg/engine/scheduler/scheduler.go`.
- Added a site definition and registry in `pkg/sites/registry/registry.go` so a site can contribute:
  - script roots
  - SQL migration roots
  - JS migration roots
  - help roots
  - CLI registration hooks
- Added package-level tests covering:
  - op spec linkage and retry metadata
  - `EmittedIDs`
  - config validation
  - scheduler validation
  - runner duplicate registration and kind ordering
  - site registry validation and ordering
- Ran `go test ./...` successfully after the phase 2 additions.

### Why

- The database layer should not invent the contract it is meant to persist.
- The goja runtime layer should not define the data model implicitly through ad hoc marshaling code.
- Sites need a formal registration contract early because migrations, scripts, help entries, and CLI commands all depend on it.

### What worked

- The package boundaries were small enough to define without dragging in storage or runtime implementation prematurely.
- The data model maps cleanly to the ticket design:
  - `Lease`
  - `RetryPolicy`
  - dependency edges
  - `OpSpec`
  - `OpResult`
  - `EmittedIDs`
- The runner registry and site registry are already useful pure-Go seams with straightforward tests.

### What didn't work

- N/A

### What I learned

- The design document translated into code cleanly once the package split was kept disciplined.
- The site registry is a useful early abstraction because it prevents migrations, scripts, and docs from turning into separate ad hoc configuration mechanisms later.
- Even without SQLite, tests on these contracts already catch useful mistakes such as duplicate registration and invalid scheduler/config values.

### What was tricky to build

- The main design tension was keeping the model explicit without overfitting to one site. I kept the store and runner interfaces narrow enough that both the NEREVAL port and future sites can still shape the implementation details later.
- The second tension was whether to place registries under `pkg/engine/` or under `pkg/sites/`. I chose `pkg/sites/registry` because the registration contract is site-facing rather than a scheduler concern.

### What warrants a second pair of eyes

- Whether `OpStatusReady` belongs as a first-class persisted status or should remain a query-time derived state after dependencies are satisfied.
- Whether `SiteDatabase` in the runner contract should remain a minimal SQL executor or become a richer projection interface later.

### What should be done in the future

- Implement engine SQLite migrations and the first concrete store in phase 3.
- Decide how dependency readiness is persisted versus computed when the scheduler is implemented.
- Add serialization tests once the SQLite store starts writing and reading these types.

### Code review instructions

- Review the contracts in this order:
  - `pkg/engine/model/types.go`
  - `pkg/engine/store/store.go`
  - `pkg/engine/runner/runner.go`
  - `pkg/sites/registry/registry.go`
  - `pkg/engine/config/config.go`
  - `pkg/engine/scheduler/scheduler.go`
- Then review the tests:
  - `pkg/engine/model/types_test.go`
  - `pkg/engine/config/config_test.go`
  - `pkg/engine/runner/runner_test.go`
  - `pkg/engine/scheduler/scheduler_test.go`
  - `pkg/sites/registry/registry_test.go`

### Technical details

- Concrete commands used during the contract phase:
  - `gofmt -w ...`
  - `go test ./...`

## Step 5: Implement engine SQLite migrations and the first concrete store

This step turned the abstract engine contracts into a working persistence layer. The goal was to get a real engine database in place with ordered SQL migrations, schema version tracking, and a first SQLite-backed implementation of the store interfaces. I kept the scope focused on engine state only: workflows, ops, dependencies, leases, results, and artifacts. Site-owned databases and site migrations remain in a later phase.

The resulting `pkg/engine/store/sqlite` package can open or create the engine DB, apply embedded SQL migrations, create workflows, enqueue ops, lease ready ops, heartbeat leases, record completions and failures, persist artifacts, and read results back for dependency consumers.

### Prompt Context

**User prompt (verbatim):** "Cool, add a detailed list of tasks to the ticket on how to build this, using the glazed framework for CLI (don't forget to properly initialize the help system and logging, see skills), and decompose the task into a set of phases. 

Then work on the tasks one by one committing at appropriate intervals and keeping a detailed diary as you work."

**Assistant interpretation:** Continue through the next planned milestone by implementing the engine DB schema and store layer, and record the results in the diary and task tracker.

**Inferred user intent:** Keep converting the design into working code, one stable layer at a time.

**Commit (code):** `d70fb56` — Add engine SQLite store and migrations

### What I did

- Added ordered SQL migrations under `pkg/engine/store/sqlite/migrations/`:
  - `001_engine_core.sql`
  - `002_engine_runtime.sql`
- Added embedded migration loading and application logic in `pkg/engine/store/sqlite/migrations.go`.
- Added schema version tracking through the `schema_migrations` table and a `CurrentVersion` helper.
- Implemented a first concrete SQLite store in `pkg/engine/store/sqlite/store.go`.
- Implemented workflow operations:
  - create workflow plus initial ops
  - fetch workflow
  - update workflow status
- Implemented op operations:
  - enqueue ops
  - fetch op by ID
  - lease the next ready op for a site and queue
  - heartbeat a lease
  - complete an op
  - fail an op
- Implemented result and artifact persistence:
  - durable `results` table entries
  - separate `artifacts` table entries
  - reload artifacts when reading a result back
- Added emitted-op normalization during completion so child ops inherit:
  - workflow ID
  - site
  - parent op ID
  when those fields are omitted by the caller.
- Added tests covering:
  - migration application on a fresh DB
  - upgrade from version 1 to version 2
  - workflow creation, leasing, completion, and result round-trip
- Ran:
  - `go mod tidy`
  - `go test ./...`

### Why

- Phase 2 defined the contract, but without a real store there was still no durable engine.
- Ordered SQL migrations are the cleanest way to make the engine schema explicit and reviewable.
- A first working store gives later phases a stable place to connect the scheduler and JS runtime.

### What worked

- The two-migration split made the upgrade test straightforward:
  - migration 1 creates the core workflow/op tables
  - migration 2 adds dependencies, leases, results, and artifacts
- The contract from phase 2 mapped cleanly into SQLite tables and store methods.
- The round-trip test now proves a minimal durable flow:
  - create workflow
  - lease an op
  - complete it
  - read the result back

### What didn't work

- The first pass at result persistence tried to infer `workflow_id` from emitted child ops, which is wrong when an op completes without emitting anything.
- Resolution:
  - query the parent op context during completion
  - normalize emitted child ops against that parent context
  - persist results and artifacts using the parent workflow ID

### What I learned

- The engine really benefits from keeping artifacts in a dedicated table even in the first version. It keeps result envelopes smaller while preserving durable raw outputs.
- `EmittedIDs` is useful even with persisted child ops because it gives a durable fan-out summary directly on the result row.
- The site-scoped queue and lease model already feels natural in the concrete SQLite implementation.

### What was tricky to build

- The main tricky part was choosing where to normalize emitted child ops. Doing it in `CompleteOp` is pragmatic because the parent op context is available there and the scheduler has not been implemented yet.
- The second tricky part was keeping the migration logic simple without painting the future site migration story into a corner. I therefore kept engine migrations embedded and engine-local, with no attempt yet to reuse that code for per-site DBs.

### What warrants a second pair of eyes

- Whether `GetResult` should eventually return artifacts lazily instead of loading them eagerly.
- Whether `FailOp` should also persist a dedicated failure/result row for terminal failures instead of only updating retry state and status.
- Whether `OpStatusPending` and `OpStatusReady` remain the right split once dependency readiness is computed by the real scheduler.

### What should be done in the future

- Add CLI visibility for engine DB health or migration status if the operator workflow needs it.
- Implement the actual scheduler loop against this store.
- Add more store tests once retries, dependency completion, and multi-op fan-out are wired through a real scheduler.

### Code review instructions

- Review the migration files first:
  - `pkg/engine/store/sqlite/migrations/001_engine_core.sql`
  - `pkg/engine/store/sqlite/migrations/002_engine_runtime.sql`
- Then review:
  - `pkg/engine/store/sqlite/migrations.go`
  - `pkg/engine/store/sqlite/store.go`
  - `pkg/engine/store/sqlite/store_test.go`
- Pay special attention to:
  - schema version handling
  - lease acquisition conditions
  - completion-time result and artifact persistence
  - emitted-op normalization

### Technical details

- Concrete commands used during the engine DB phase:
  - `gofmt -w ...`
  - `go mod tidy`
  - `go test ./...`

## Step 6: Add engine visibility commands for smoke tests and debugging

This step added the first operator-facing engine inspection commands. The goal was pragmatic: make it easy to check whether the engine DB exists, whether migrations are current, and what the current workflow/op/lease/result counts look like without opening SQLite manually. That is useful both for user-facing smoke tests and for my own debugging while the scheduler and site runtime are still being built.

The resulting commands are:

- `scraper engine status`
- `scraper engine migrations status`

Both commands accept `--engine-db <path>`. They are intentionally non-destructive when the DB file does not exist; in that case they report the missing DB and the latest known migration version instead of creating a new engine DB.

### Prompt Context

**User prompt (verbatim):** "no, make the visibility commands, that's useful for me and for you to debug things as they go along and we do smoke tests"

**Assistant interpretation:** Implement the deferred engine visibility/admin commands now, not later, and make them useful for both operator workflows and development debugging.

**Inferred user intent:** Prioritize observability and inspectability early so future smoke tests and debugging sessions do not depend on opening SQLite directly.

**Commit (code):** `995aa29` — Add engine visibility commands

### What I did

- Added an inspection layer in `pkg/engine/store/sqlite/status.go`.
- Added inspection data types:
  - `MigrationStatus`
  - `EngineStatus`
- Added helpers to inspect:
  - DB existence
  - schema initialization
  - current schema version
  - latest known migration
  - applied vs pending migrations
  - workflow count
  - op counts by status
  - active and expired lease counts
  - result count
  - artifact count
- Added CLI commands in `pkg/cmd/engine.go`:
  - `scraper engine status`
  - `scraper engine migrations status`
- Registered the new engine command tree on the root command.
- Added tests for:
  - inspecting a missing DB
  - inspecting a populated DB
  - CLI output for `engine status`
  - CLI output for `engine migrations status`
- Added a brief note in the embedded help topic so the commands are discoverable from `scraper help`.
- Ran:
  - `go test ./...`
  - `go run ./cmd/scraper engine status --engine-db /tmp/does-not-exist-engine.db`

### Why

- The user explicitly asked for these commands because they are useful during incremental build-out and smoke testing.
- The engine DB is now real, so basic observability is more valuable than waiting for the full scheduler and site runtime.
- A non-destructive inspection path is a better operator tool than implicitly creating the DB on read-only status commands.

### What worked

- The inspector layer fits naturally next to the SQLite store implementation.
- A single inspection pass can populate both commands:
  - summary status
  - migration status
- The missing-DB behavior is useful and clear. The smoke test output now shows:
  - path
  - missing/existing state
  - initialization state
  - current version or `n/a`
  - latest known migration

### What didn't work

- N/A

### What I learned

- Even minimal admin commands immediately improve the development loop because they provide a stable, human-readable “is the engine sane?” check.
- The migration listing is especially useful because it avoids ambiguity between:
  - missing DB
  - existing but uninitialized DB
  - initialized but out-of-date DB

### What was tricky to build

- The main design choice was to keep inspection non-destructive. Reusing `Open()` would have auto-migrated or created the DB, which is wrong for a pure status command. The solution was to implement a separate inspection path that opens the DB only if the file already exists and checks `sqlite_master` before assuming the schema is initialized.

### What warrants a second pair of eyes

- Whether future admin commands should move under a broader `scraper inspect ...` tree instead of staying under `scraper engine ...`.
- Whether result and artifact counts should eventually become per-workflow or per-site filtered summaries once more CLI surfaces exist.

### What should be done in the future

- Add workflow- and op-level visibility commands once the scheduler starts producing richer runtime state.
- Consider a machine-readable output mode later if these commands become part of scripted smoke tests.

### Code review instructions

- Review:
  - `pkg/engine/store/sqlite/status.go`
  - `pkg/engine/store/sqlite/status_test.go`
  - `pkg/cmd/engine.go`
  - `pkg/cmd/engine_test.go`
- Then run:
  - `scraper engine status --engine-db state/engine.db`
  - `scraper engine migrations status --engine-db state/engine.db`

### Technical details

- Concrete commands used during the visibility phase:
  - `go test ./...`
  - `go run ./cmd/scraper engine status --engine-db /tmp/does-not-exist-engine.db`

## Step 7: Implement per-site DBs with mixed SQL and JS migrations

This step completed the first site-owned storage layer. The goal was to make site state fully separate from engine state by giving each site its own SQLite DB and a migration system that can execute both SQL files and JavaScript files against that site DB. I also added `scraper site migrate <site>` so site DBs can be initialized and upgraded explicitly during development and smoke tests.

The implementation uses one combined migration stream per site. SQL and JS migration files share a single numeric version sequence. The runner orders migrations by that numeric prefix across both file types and rejects duplicate version numbers, so a site can express a migration history such as:

- `001_init.sql`
- `002_seed.js`
- `003_indexes.sql`

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue working through the next unfinished phase after the engine visibility commands.

**Inferred user intent:** Keep executing the phased backlog in order instead of pausing after engine-only work.

**Commit (code):** `6b38cbb` — Add site DB migrations and CLI

### What I did

- Extended the site definition contract in `pkg/sites/registry/registry.go` with:
  - `DatabaseFileName`
  - `RuntimeModuleRegistrars`
- Added a new site migration manager in `pkg/sites/migrate/manager.go`.
- Implemented site DB path rules:
  - DBs live under a caller-provided sites directory
  - default filename is `<site>.db`
  - a site can override that filename explicitly
- Implemented site-local migration history using `schema_migrations` in the site DB.
- Implemented combined migration loading across:
  - `SQLMigrationsFS`
  - `JSMigrationsFS`
- Implemented deterministic ordering by numeric prefix across SQL and JS migration files.
- Rejected duplicate migration versions across SQL and JS sources.
- Implemented SQL migration application in a transaction.
- Implemented JS migration application in a transaction using a dedicated go-go-goja runtime.
- Exposed a narrow JS migration API object:
  - `exec(sql, ...args)`
  - `query(sql, ...args)`
  - `hasTable(name)`
  - `hasColumn(table, column)`
  - `log(...)`
- Supported async JS migrations by awaiting a returned promise if the migration exports an `async function`.
- Added tests for:
  - mixed SQL and JS migrations
  - relative `require()` inside JS migrations
  - rerun/idempotency
  - duplicate-version rejection
- Added the CLI command:
  - `scraper site migrate <site> --sites-dir <dir>`
- Added CLI tests for:
  - successful site migration
  - unknown site failure

### Why

- The design explicitly called for each site to own its own DB and migrations.
- SQL-only migrations are not sufficient for all future site evolution steps; some updates need conditional logic or backfills.
- An explicit CLI entrypoint is important for development and smoke testing because it lets us validate site DB behavior independently from the scheduler.

### What worked

- The combined SQL/JS ordering model is simple and easy to explain.
- The go-go-goja runtime is already a good fit for the migration runner even before the full JS op runtime exists.
- Relative `require()` in JS migrations works once the entry migration is loaded as a relative module.
- The migration rerun behavior is clean: already-applied versions are skipped and the second run applies zero migrations.

### What didn't work

- The first JS migration attempt failed with:
  - `require migration 002_seed.js: Invalid module`
- Cause:
  - the entry migration was required as a bare module name instead of a relative module path
- Resolution:
  - require the entrypoint as `./<migration-file>`
  - keep the loader tolerant of normalized relative module paths

### What I learned

- Site migrations are a good place to start using go-go-goja because the API surface is intentionally narrow and easy to control.
- The explicit site command is valuable immediately because it establishes the operator flow and makes tests more realistic, even before any production site package is registered.

### What was tricky to build

- The main tricky part was keeping JS migration execution transactional. The solution was to bind the JS API directly to the active `*sql.Tx`, so both SQL and JS migrations participate in the same commit/rollback boundary.
- The second tricky part was keeping the migration runtime narrow. I deliberately passed a purpose-built API object into the exported migration function instead of enabling broad default modules.

### What warrants a second pair of eyes

- Whether the site migration API should eventually expose a more structured row API than `query()` returning generic maps.
- Whether future site migration commands should also support a dry-run or status mode under the `site` command tree.

### What should be done in the future

- Register a real site package so `scraper site migrate <site>` is useful outside of tests.
- Reuse the go-go-goja runtime integration patterns from this migration runner when phase 5 introduces JS-backed op runners.
- Add site status or inspection commands once a real site DB exists in the repo.

### Code review instructions

- Review the site DB and migration code in this order:
  - `pkg/sites/registry/registry.go`
  - `pkg/sites/migrate/manager.go`
  - `pkg/sites/migrate/manager_test.go`
  - `pkg/cmd/site.go`
  - `pkg/cmd/site_test.go`
- Then compare the ticket updates:
  - `tasks.md`
  - the storage/migration notes in the design doc
  - this diary entry

### Technical details

- Concrete commands used during the site DB phase:
  - `go mod tidy`
  - `go test ./pkg/sites/migrate ./pkg/cmd`
  - `go test ./...`

## Step 8: Expose preconfigured `scraper-db` and `site-db` modules to JS

This step tightened the JS database story so JavaScript does not need to discover or open SQLite files itself. Instead of expecting JS to call `require("database").configure(...)` with a path, Go now preconfigures named modules and injects them into the runtime:

- `require("scraper-db")`
- `require("site-db")`

The work landed in two layers. First, `go-go-goja`'s `database` module was refactored so it can be instantiated as a named, preconfigured module backed by an existing `*sql.DB` or `*sql.Tx`. Second, the `scraper` repo added a runtime registrar that exposes those preconfigured modules to JS runtimes, and the site migration runtime now uses it so migrations can access `site-db` directly.

### Prompt Context

**User prompt (verbatim):** "expose a preconfigured site db and scraper db to js (instead of leaving it to the js to open the proper file). You can probably add an option to the db functionality in go-go-goja to pass in a \"preconfigured\" version"

**Assistant interpretation:** Refactor the go-go-goja database module so Go can construct named preconfigured DB modules, then wire that capability into scraper runtimes so JS gets stable site/scraper DB handles without owning file-path discovery.

**Inferred user intent:** Keep DB-path ownership in Go while still making DB access ergonomic for JS, especially for site code and smoke-test/debug workflows.

**Commit (code):** `go-go-goja` commit `3722177` added reusable preconfigured database modules, and `scraper` commit `2ad649a` wired runtime exposure for `scraper-db` and `site-db`.

### What I did

- Refactored `go-go-goja/modules/database` so the module can be constructed with options:
  - custom module name
  - preconfigured DB handle
  - optional close function
  - configure enabled/disabled
- Kept the default `require("database")` module behavior working for callers that still want to configure a DB explicitly.
- Added integration tests in `go-go-goja` for:
  - default configurable `database`
  - named preconfigured module use
  - rejection of `configure()` on preconfigured modules
- Added a scraper-side runtime registrar in `pkg/js/runtime/databases.go`.
- Made that registrar expose:
  - `scraper-db`
  - `site-db`
  when corresponding DB handles are provided by Go.
- Wired the site migration runtime to register `site-db` automatically, backed by the active transaction.
- Updated the site migration tests so one JS migration now uses:
  - `const siteDB = require("site-db")`
  instead of only the explicit migration helper API.
- Added a runtime-level scraper test proving both preconfigured modules can be required in the same JS runtime.

### Why

- The user was right that JS should not own DB-path discovery.
- Preconfigured modules keep DB lifecycle and file-location ownership in Go.
- Using module names like `site-db` and `scraper-db` gives JS a simple, stable contract that matches the architecture we already documented.

### What worked

- The refactored `database` module still supports the old configurable behavior.
- The new preconfigured path works cleanly with both `*sql.DB` and `*sql.Tx`.
- The site migration runtime can now expose `site-db` without broadening the migration API surface too much.
- Full test suites passed in both repos after the change.

### What didn't work

- The first draft of the `go-go-goja` integration test created an import cycle by testing the module from the same package while importing `engine`.
- Resolution:
  - move the test package to `databasemod_test`
- I also initially used the wrong callback signature for `runtime.Owner.Call` in two tests.
- Resolution:
  - switch the callbacks to use `*goja.Runtime`

### What I learned

- The `database` module needed an explicit ownership boundary: preconfigured should not imply that JS may close a Go-owned DB.
- The right model is:
  - Go injects the handle
  - JS uses it
  - Go still owns the file path and lifecycle

### What was tricky to build

- The sharp edge was ownership of `close()`. For preconfigured modules, letting JS close a Go-owned DB would be wrong, so the module now only closes a DB when Go explicitly provides a close function.
- The second tricky part was making the change reusable instead of only fixing the migration runtime. That is why I split the change into a reusable `go-go-goja` module option layer and a scraper-side runtime registrar.

### What warrants a second pair of eyes

- Whether the future JS op runtime should always expose both modules, or only expose `site-db` for site code and a more specialized API for engine reads.
- Whether `scraper-db` should stay a raw SQL module forever or later become a more opinionated engine-state API.

### What should be done in the future

- Reuse the new registrar in the phase-5 JS op runtime so site scripts get the same DB contract as migrations.
- Decide which JS runtimes should expose only `site-db` versus both `site-db` and `scraper-db`.

### Code review instructions

- Review the reusable database-module refactor in the sibling repo:
  - `go-go-goja/modules/database/database.go`
  - `go-go-goja/modules/database/database_test.go`
- Then review the scraper-side integration:
  - `scraper/pkg/js/runtime/databases.go`
  - `scraper/pkg/js/runtime/databases_test.go`
  - `scraper/pkg/sites/migrate/manager.go`
  - `scraper/pkg/sites/migrate/manager_test.go`

### Technical details

- Concrete commands used during the preconfigured DB phase:
  - `go test ./modules/database/...`
  - `go test ./pkg/js/runtime ./pkg/sites/migrate ./pkg/cmd`
  - `go test ./...`

## Step 9: Build the first executable JS op runtime

This step completed the first real JS execution slice for the scraper engine. Before this point we had site migration runtimes and preconfigured database modules, but not a general op runner that could load a site script, construct a stable `ctx` contract, and return a durable `OpResult`. The new code adds exactly that: a generic JS executor in `pkg/js/runtime` and a `js` runner adapter in `pkg/engine/runner`.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Move into phase 5 of the ticket backlog and implement the milestone-one JS runtime integration rather than stopping at the site DB milestone.

**Inferred user intent:** Turn the architectural design into working runtime code so future phases can run real site scripts through the engine instead of only through migration helpers.

**Commit (code):** `1bebe66` — Add JS runtime executor and runner

### What I did

- Added `pkg/js/runtime/executor.go` as the generic JS op executor.
- Defined the current metadata contract:
  - JS ops use runner kind `js`
  - the executed module path comes from op metadata key `script`
- Added a runtime builder that:
  - loads site scripts from the site's `ScriptsFS` and `ScriptsRoot`
  - injects preconfigured `scraper-db` and `site-db` modules
  - applies any site-provided runtime module registrars
- Defined the milestone-one `ctx` contract with:
  - `ctx.input`
  - `ctx.workflow`
  - `ctx.op`
  - `ctx.lease`
  - `ctx.dep(opID)`
  - `ctx.emit(spec)`
  - `ctx.writeRecord(collection, key, data)`
  - `ctx.writeArtifact(spec)`
  - `ctx.log(...)`
- Added JS-to-Go result-envelope marshalling:
  - plain return values become `OpResult.Data`
  - `{ data, error }` objects populate the durable result envelope explicitly
- Added emitted-op normalization so child ops inherit:
  - current workflow ID
  - current site
  - current op as `parent_id`
  unless JS overrides those fields
- Added `pkg/js/runtime/promises.go` so async JS ops can be awaited with the same explicit polling model already used in site migrations.
- Added `pkg/engine/runner/js.go` as the engine-facing `js` runner that resolves the site definition and delegates execution into the new runtime package.
- Added runtime and runner tests covering:
  - script execution
  - script-local relative `require()`
  - dependency reads
  - emitted ops
  - record writes
  - artifact writes
  - preconfigured `scraper-db` and `site-db` access
  - runtime closer execution on teardown

### Why

- The next scheduler and HTTP phases need a real programmable execution target, not only sketches in the design doc.
- The user had already pushed the architecture toward Go-owned DB handles and Go-owned path discovery, so this was the right moment to make the JS op runtime match that same ownership model.
- Building the executor before the scheduler keeps the interfaces honest: the scheduler can now target a real runner shape instead of a hypothetical one.

### What worked

- Relative `require("./lib/helper.js")` loading worked cleanly from `fstest.MapFS`, so the site script model already supports both embedded filesystems and normal directories.
- The preconfigured database modules integrated naturally into the executor once the runtime builder delegated to the existing `DatabaseRegistrar`.
- The result envelope shape maps well onto the current `model.OpResult`.
- Full repo tests passed after the new runner was added.

### What didn't work

- The first test draft imported `io/fs/fstest`, which is the wrong package path.
- Resolution:
  - switch to `testing/fstest`
- I also hit a test import-name collision between:
  - `github.com/dop251/goja_nodejs/require`
  - `github.com/stretchr/testify/require`
- Resolution:
  - alias the node require package as `noderequire`

### What I learned

- The executor wants two distinct layers:
  - runtime construction and JS marshalling in `pkg/js/runtime`
  - engine-facing site lookup in `pkg/engine/runner`
- That split is cleaner than putting site-registry lookups directly into the runtime package or pushing JS marshalling into the engine package.

### What was tricky to build

- The sharp edge here was not the Goja call itself. It was defining which defaults belong in emitted child ops. The current implementation makes emitted ops inherit the current workflow ID, site, and parent ID unless the script overrides them explicitly.
- The second subtle part was keeping the result envelope small while still allowing future extension. Right now the runtime only blesses the current fields we actually need: `data`, `error`, emitted ops, records, and artifacts.

### What warrants a second pair of eyes

- Whether the `script` metadata key should stay on `OpSpec.Metadata` or move into a runner-specific input envelope later.
- Whether the current `ctx.dep(opID)` lookup is sufficient for milestone one, or whether a later phase should add dependency aliases for ergonomics.
- Whether `ctx.writeRecord` should eventually accept structured keys instead of the current string-or-JSON-string fallback.

### What should be done in the future

- Wire the scheduler loop to lease and execute `js` ops through the new runner.
- Add the first Go-backed HTTP scrape runner so JS can consume fetch results via dependencies instead of doing transport itself.
- Decide whether to add helper sugar such as `ctx.ops.scrape(...)` only after the HTTP runner exists.

### Code review instructions

- Review the JS runtime core first:
  - `scraper/pkg/js/runtime/executor.go`
  - `scraper/pkg/js/runtime/promises.go`
  - `scraper/pkg/js/runtime/executor_test.go`
- Then review the engine adapter:
  - `scraper/pkg/engine/runner/runner.go`
  - `scraper/pkg/engine/runner/js.go`
  - `scraper/pkg/engine/runner/js_test.go`
- Finally, review the operator-facing ticket/doc sync:
  - `scraper/pkg/doc/topics/scraper-architecture-overview.md`
  - `scraper/ttmp/.../tasks.md`
  - `scraper/ttmp/.../design-doc/01-generic-go-scraper-engine-and-nereval-port-design-guide.md`

### Technical details

- Concrete commands used during the JS runtime phase:
  - `gofmt -w pkg/js/runtime/executor.go pkg/js/runtime/promises.go pkg/js/runtime/executor_test.go pkg/engine/runner/js.go pkg/engine/runner/js_test.go pkg/engine/runner/runner.go`
  - `go test ./pkg/js/runtime ./pkg/engine/runner`
  - `go test ./...`

## Step 10: Add the first real scheduler and worker loop

This step turned the engine from a durable store plus runnable ops into a real worker loop. Before this change, the codebase could persist workflows, lease one ready op, and run JS modules, but it did not yet have a scheduler that understood dependency promotion, retry timing, expired-lease recovery, queue-domain control, or workflow status transitions. The new scheduler package now closes that gap for milestone one.

### Prompt Context

**User prompt (verbatim):** "continue."

**Assistant interpretation:** Move from the phase-5 JS runner milestone into phase 6 and implement the scheduler/worker loop rather than stopping at isolated runner execution.

**Inferred user intent:** Keep building the engine phase by phase until the runtime and persistence layers are connected by a real durable execution loop.

**Commit (code):** `da71a46` — Add scheduler loop and retry handling

### What I did

- Replaced the scheduler stub in `pkg/engine/scheduler/scheduler.go` with a real scheduler implementation.
- Added scheduler support for:
  - workflow submission through `CreateWorkflow(...)`
  - `RunOnce(...)` for deterministic test cycles
  - `Run(...)` for polling execution loops
  - runner dispatch by op kind
  - durable success/failure handling through the store
  - workflow status recomputation
  - structured scheduler events and optional observer hooks
- Added queue-domain control by making the store expose distinct queue candidates and by ensuring only one active lease exists per `site + queue` pair.
- Extended `model.OpSpec` with durable `RetryState` so attempt count and next-attempt time survive across scheduling cycles.
- Extended the engine store contract with:
  - `RefreshRunnableOps(...)`
  - `ListQueueCandidates(...)`
  - `GetWorkflowStats(...)`
- Implemented those new store methods in the SQLite backend.
- Added runnable-op refresh semantics in SQLite:
  - recover expired leased `running` ops back to `ready`
  - delete expired lease rows
  - promote `pending` ops to `ready` when dependencies are satisfied
  - cancel `pending` ops whose required dependencies have failed or been canceled
- Updated SQLite failure handling so failures also persist a result envelope with `error_json`, which makes failure state visible through the normal result API.
- Added retry/backoff scheduling in the scheduler using:
  - `RetryPolicy`
  - `RetryState`
  - fixed, linear, and exponential backoff helpers
- Added integration-style scheduler tests covering:
  - fan-out from one op to a dependent child op
  - dependency completion before child execution
  - retry behavior with delayed requeue
  - expired-lease recovery / resume semantics
  - one-op-at-a-time queue-domain execution

### Why

- The user asked to continue phase by phase, and the next unimplemented phase after the JS runtime was the scheduler.
- The JS runner is only useful once the engine can actually claim work, run it durably, and recover from interruption.
- A scheduler implemented against the real SQLite store is also a better debugging surface for smoke tests than more abstract helper code.

### What worked

- The existing store schema already had most of the raw material needed for phase 6: op status, dependencies, leases, retry state, and results.
- Modeling queue domains as one active lease per `site + queue` pair fit the current architecture well and kept the first implementation simple.
- The scheduler tests were easiest to trust when they ran against the actual SQLite store instead of fake in-memory mocks.
- Full repo tests passed after the scheduler work landed.

### What didn't work

- There was a hidden resume gap in the earlier store implementation: an expired lease left an op in `running`, so it could not actually be reclaimed.
- Resolution:
  - add `RefreshRunnableOps(...)` to recover expired leased ops back to `ready`
- The previous failure path also did not persist failure details into the `results` table.
- Resolution:
  - update `FailOp(...)` to upsert a failure result row with `error_json`

### What I learned

- The scheduler became much simpler once the store knew how to answer "what is runnable now?" instead of forcing the worker loop to reconstruct dependency state in Go every cycle.
- Expired-lease recovery is not a secondary feature. It is a core correctness path, because without it the engine only looks durable until the first crash.

### What was tricky to build

- The subtle part was deciding where to put dependency semantics. I intentionally pushed status promotion/cancellation into the store layer rather than scattering it across scheduler branches. That keeps the scheduling loop smaller and makes the DB state transitions easier to test directly.
- The second tricky part was failure semantics for dependents. The current milestone cancels `pending` ops blocked by required failed dependencies. That is explicit and recoverable, but it is still a design point to revisit later if we want richer blocked/dead-letter states.

### What warrants a second pair of eyes

- Whether queue-domain concurrency should stay fixed at one active lease per `site + queue`, or whether a later phase should allow configurable per-queue parallelism.
- Whether blocked required-dependency ops should remain `canceled` or eventually gain a dedicated status such as `blocked`.
- Whether `CreateWorkflow(...)` belongs permanently in the scheduler package or should later move into a more explicit workflow service.

### What should be done in the future

- Add the HTTP fetch runner so the scheduler can drive real acquisition work, not only synthetic test runners.
- Add operator-facing workflow and op inspection commands in phase 9 so the new scheduler state becomes directly visible from the CLI.
- Revisit richer workflow status transitions once we have mixed success/failure outcomes from real NEREVAL workflows.

### Code review instructions

- Review the scheduler core first:
  - `scraper/pkg/engine/scheduler/scheduler.go`
  - `scraper/pkg/engine/scheduler/scheduler_test.go`
- Then review the store contract/state changes:
  - `scraper/pkg/engine/store/store.go`
  - `scraper/pkg/engine/store/sqlite/store.go`
  - `scraper/pkg/engine/model/types.go`
- Finally, review the doc/ticket sync:
  - `scraper/pkg/doc/topics/scraper-architecture-overview.md`
  - `scraper/ttmp/.../tasks.md`
  - `scraper/ttmp/.../design-doc/01-generic-go-scraper-engine-and-nereval-port-design-guide.md`

### Technical details

- Concrete commands used during the scheduler phase:
  - `gofmt -w pkg/engine/model/types.go pkg/engine/store/store.go pkg/engine/store/sqlite/store.go pkg/engine/scheduler/scheduler.go pkg/engine/scheduler/scheduler_test.go`
  - `go test ./pkg/engine/model ./pkg/engine/store/sqlite ./pkg/engine/scheduler`
  - `go test ./...`

## Step 11: Add the first generic HTTP fetch runner

This step completed phase 7 by adding the first Go-backed acquisition runner. Before this change, the engine could schedule and execute JS work, but there was still no real fetch primitive for those JS scripts to depend on. The new `http/fetch` runner closes that gap with a generic request envelope, templated request rendering, response metadata capture, optional body artifact persistence, and retry classification.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Move from the scheduler milestone into phase 7 and implement the first concrete HTTP acquisition runner.

**Inferred user intent:** Keep building the engine until it can perform real HTTP work that later NEREVAL scripts can consume as dependency results.

**Commit (code):** `72b7f32` — Add generic HTTP fetch runner

### What I did

- Added `pkg/engine/runner/http.go` implementing kind `http/fetch`.
- Defined the first fetch input envelope:
  - `request.method`
  - `request.url`
  - `request.headers`
  - `request.form`
  - `request.body`
  - top-level `persistBody`
  - top-level `artifactName`
- Added template rendering for:
  - URL strings
  - header values
  - form values
  - raw request body text
- Bound the template context to:
  - `.input` from op input JSON
  - `.workflow.input` from workflow input JSON
  - `.op` metadata such as ID, site, queue, and dedup key
- Added automatic request defaults for:
  - `GET` method when unspecified
  - configured `User-Agent` when unspecified
  - form `Content-Type` for rendered form payloads
- Added response result-envelope capture including:
  - rendered request method and URL
  - rendered request headers and body text
  - response status code and status string
  - final URL after redirects
  - response headers
  - response content type and body length
  - optional `bodyArtifactID`
- Added optional persisted body artifacts with:
  - deterministic artifact IDs
  - content type
  - request/response metadata
- Added retry classification:
  - transport errors are retryable
  - `429` is retryable
  - `5xx` is retryable
  - `4xx` such as `404` are non-retryable
- Added runner tests for:
  - successful HTML fetch with templated POST form data and artifact persistence
  - retryable `502` failure
  - non-retryable `404` failure

### Why

- The scheduler milestone made the engine capable of executing work, but phase 8 needs a real acquisition primitive for list/detail page fetches.
- The user already narrowed milestone one by removing `ctx.fetch()`, which means fetch has to exist as a Go-owned op runner instead of a JS helper.
- Testing the runner with `httptest` fixtures gives a stable base for the later NEREVAL port.

### What worked

- A simple `text/template`-based rendering layer was enough to cover the current request-shaping needs without inventing a custom DSL.
- Returning a structured request/response envelope fits naturally into the existing `OpResult.Data` field.
- Persisting the response body as a normal artifact aligns well with the result/artifact split we already committed to in the design docs.
- The retryable/non-retryable status split maps cleanly onto the scheduler’s existing retry flow.

### What didn't work

- The first success test assumed form fields would preserve insertion order in the encoded request body.
- Resolution:
  - update the test to expect the stable `url.Values.Encode()` ordering
- The first exact JSON assertion for response metadata was too strict because `httptest` adds headers such as `Date`.
- Resolution:
  - decode the result JSON and assert only the stable fields we actually care about

### What I learned

- The fetch runner does not need to know anything about DOM parsing to be useful. Its job is just to produce a stable transport artifact and metadata envelope.
- Capturing the actual sent headers in the result envelope is more useful than only reporting the explicit input headers, because it reflects defaulted values such as `User-Agent` and `Content-Type`.

### What was tricky to build

- The subtle part was drawing the boundary between request input and request diagnostics. I kept the op input small and expressive, then reported the fully rendered request separately in the result envelope so later debugging and fixture generation can see what was actually sent.
- The other small design choice was whether to introduce proxy support immediately. I intentionally left that out of this phase because it is not required for the current milestone and would complicate the runner surface prematurely.

### What warrants a second pair of eyes

- Whether the template context should eventually include dependency-derived values directly, or whether those should remain the responsibility of upstream JS ops that emit the fetch op.
- Whether the request envelope should later gain explicit query-parameter support instead of relying on templated URLs.
- Whether the fetch runner should later classify additional `4xx` statuses as retryable for specific sites.

### What should be done in the future

- Port NEREVAL list-page GET, next-page POST, and detail-page GET onto this runner in phase 8.
- Add richer request options only when a real site requires them, such as proxy selection or cookie jar behavior.
- Consider a small helper layer for emitting common fetch shapes from JS once the NEREVAL port makes those patterns obvious.

### Code review instructions

- Review the fetch runner first:
  - `scraper/pkg/engine/runner/http.go`
  - `scraper/pkg/engine/runner/http_test.go`
- Then review the operator-facing contract sync:
  - `scraper/pkg/doc/topics/scraper-architecture-overview.md`
  - `scraper/ttmp/.../tasks.md`
  - `scraper/ttmp/.../design-doc/01-generic-go-scraper-engine-and-nereval-port-design-guide.md`

### Technical details

- Concrete commands used during the HTTP phase:
  - `gofmt -w pkg/engine/runner/http.go pkg/engine/runner/http_test.go`
  - `go test ./pkg/engine/runner`
  - `go test ./...`

## Step 12: Add Hacker News and Slashdot as smaller exercise sites before NEREVAL

This step deliberately paused the NEREVAL port and inserted two smaller site packages first: Hacker News and Slashdot. The intent was to force the current engine shape through a complete real workflow on simpler HTML before committing to the more complex ASP.NET and detail-fan-out behavior in NEREVAL.

The resulting milestone is a much better proving ground for the system that now exists. Both sites run through the actual scheduler, the real `http/fetch` runner, the real JS runner, and a real site DB with migrations. That means the remaining NEREVAL work can focus on NEREVAL-specific behavior instead of still debugging the generic engine path.

### Prompt Context

**User prompt (verbatim):** "before we do nereval, actually make a scraper for hacker news and one for slashdot. that way we can exercise on something smaller."

**Assistant interpretation:** Pause the planned NEREVAL port and first add two simpler built-in sites, Hacker News and Slashdot, so the reusable runtime, scheduler, HTTP, JS, and site DB path can be exercised on smaller HTML targets.

**Inferred user intent:** De-risk the upcoming NEREVAL work by validating the generic scraper engine against simpler sites first, while also creating smoke-test targets that are easier to reason about during operator testing and future debugging.

**Commit (code):** `ceb4a98` — Add exercise scraper sites for Hacker News and Slashdot

### What I did

- Added a built-in site registry package:
  - `pkg/sites/defaults/defaults.go`
  - `pkg/sites/defaults/defaults_test.go`
- Switched the production root command to the built-in site registry in:
  - `pkg/cmd/root.go`
- Extended the scheduler so Go can inject pre-opened DB handles into runner execution:
  - `pkg/engine/scheduler/scheduler.go`
  - added `SetScraperDB(...)`
  - added `SetSiteDBProvider(...)`
- Added the Hacker News site package with:
  - `pkg/sites/hackernews/site.go`
  - `pkg/sites/hackernews/migrations/001_init.sql`
  - `pkg/sites/hackernews/scripts/seed.js`
  - `pkg/sites/hackernews/scripts/extract_frontpage.js`
  - `pkg/sites/hackernews/scripts/lib/frontpage.js`
  - `pkg/sites/hackernews/fixtures/frontpage.html`
  - `pkg/sites/hackernews/site_test.go`
- Added the Slashdot site package with:
  - `pkg/sites/slashdot/site.go`
  - `pkg/sites/slashdot/migrations/001_init.sql`
  - `pkg/sites/slashdot/scripts/seed.js`
  - `pkg/sites/slashdot/scripts/extract_frontpage.js`
  - `pkg/sites/slashdot/scripts/lib/frontpage.js`
  - `pkg/sites/slashdot/fixtures/frontpage.html`
  - `pkg/sites/slashdot/site_test.go`
- Added a built-in CLI smoke test in:
  - `pkg/cmd/site_test.go`
- Updated the embedded operator help entry in:
  - `pkg/doc/topics/scraper-architecture-overview.md`
- Ran:
  - `gofmt -w pkg/cmd/root.go pkg/cmd/site_test.go pkg/engine/scheduler/scheduler.go pkg/sites/defaults/defaults.go pkg/sites/defaults/defaults_test.go pkg/sites/hackernews/site.go pkg/sites/hackernews/site_test.go pkg/sites/slashdot/site.go pkg/sites/slashdot/site_test.go`
  - `go test ./pkg/sites/hackernews -run TestHackerNewsFrontpageWorkflow -v`
  - `go test ./pkg/sites/slashdot -run TestSlashdotFrontpageWorkflow -v`
  - `go test ./...`

### Why

- The engine already had enough moving parts that jumping straight into NEREVAL would have mixed engine bugs with site-specific bugs.
- Hacker News and Slashdot are useful exercise targets because:
  - each has a public front page,
  - each has simpler HTML than NEREVAL,
  - each still forces the engine through the same durable op flow,
  - each produces queryable site DB rows.
- Wiring the built-in site registry into the production CLI makes smoke testing easier for both implementation and operator workflows.

### What worked

- The current architecture was already strong enough to support real site packages without structural changes to the runner APIs.
- The `seed.js -> http/fetch -> extract_frontpage.js` pattern mapped naturally onto both sites.
- The scheduler-side site DB injection worked cleanly once it was threaded through `runner.RunContext`.
- The Slashdot path passed quickly once the first parser and site fixture were in place.
- The final `go test ./...` run passed after the Hacker News parser was tightened.

### What didn't work

- The first full run failed because both extractor scripts treated the dependency as failed too eagerly. The failing command was:
  - `go test ./...`
- The visible symptom in both sites was:
  - scheduler logs showed `error_code":"fetch_failed"`
  - the workflow transitioned to `failed` during the extract op
- Resolution:
  - narrow the JS guard from `if (dep.error)` to `if (dep.error && dep.error.code)`
- After that, Slashdot passed but Hacker News still failed with:
  - `Error: "[]" should have 2 item(s), but has 0`
  - command: `go test ./pkg/sites/hackernews -run TestHackerNewsFrontpageWorkflow -v`
- Resolution:
  - relax the Hacker News front-page regex so it tolerates whitespace in the paired subtext row and title link markup.

### What I learned

- The generic execution path is now real, not hypothetical:
  - JS can emit fetch ops,
  - fetch ops can persist body artifacts,
  - downstream JS can read those artifacts through dependency results,
  - site DB writes can happen against a preconfigured DB handle,
  - the scheduler can drive the whole chain to completion.
- Smaller fixture-backed sites are a very effective intermediate layer between unit tests and a full NEREVAL port.
- The exercise-site milestone also clarified that the site registry should be a first-class production concern, not only a testing seam.

### What was tricky to build

- The sharpest edge was the boundary between dependency result shape and JS ergonomics. The extractor scripts need to interpret dependency results defensively because absent or nullish fields can look different after JS export than they do in Go.
- The other tricky part was deciding how “real” the exercise sites should be. I intentionally kept them to one front-page workflow each, because the point is to validate the reusable engine path, not to build a large crawler for those sites.
- The scheduler DB injection had to stay narrow. The scheduler now resolves the site DB per op site and passes it down into the runner context, but it still does not let arbitrary site JS discover DB paths or open DBs itself.

### What warrants a second pair of eyes

- The regex-based HTML extraction is intentionally lightweight for the exercise sites, but it is also brittle by nature. Reviewers should treat these as proving fixtures, not as the final pattern for more complex sites.
- The built-in site registry is now wired into `NewRootCommand`, so any future production root-command changes should be reviewed against duplicate registration or startup errors.
- The scheduler DB injection path should be reviewed to make sure later multi-site or concurrent worker setups do not accidentally share the wrong site DB handle.

### What should be done in the future

- Keep Hacker News and Slashdot as permanent smoke-test sites while NEREVAL is being ported.
- Consider adding a minimal workflow-start command once the operator CLI phase begins so these sites can be run outside tests without custom harness code.
- Start the NEREVAL port next, but keep the same discipline:
  - fixture first,
  - smaller fan-out slices,
  - end-to-end assertions against the site DB.

### Code review instructions

- Start with the site registry and scheduler injection:
  - `scraper/pkg/sites/defaults/defaults.go`
  - `scraper/pkg/engine/scheduler/scheduler.go`
- Then review the concrete site packages:
  - `scraper/pkg/sites/hackernews/site.go`
  - `scraper/pkg/sites/hackernews/scripts/extract_frontpage.js`
  - `scraper/pkg/sites/hackernews/site_test.go`
  - `scraper/pkg/sites/slashdot/site.go`
  - `scraper/pkg/sites/slashdot/scripts/extract_frontpage.js`
  - `scraper/pkg/sites/slashdot/site_test.go`
- Validate with:
  - `go test ./pkg/sites/hackernews -run TestHackerNewsFrontpageWorkflow -v`
  - `go test ./pkg/sites/slashdot -run TestSlashdotFrontpageWorkflow -v`
  - `go test ./...`

### Technical details

- The Hacker News workflow shape is:
  - `seed.js`
  - emits `http/fetch`
  - `extract_frontpage.js` depends on the fetch op
  - extraction writes into `hackernews.db`
- The Slashdot workflow shape is the same, but the site DB schema differs to capture source-link and comment metadata.
- The built-in exercise sites are now available to the production CLI via `NewRootCommand(...)`, so commands such as `scraper site migrate hackernews` and `scraper site migrate slashdot` no longer require a custom registry in tests.

## Step 13: Add a pure-JS demo site and runnable CLI flow

This step added a built-in `js-demo` site so the engine can be exercised without touching the HTTP runner at all. The intent was to keep building confidence in the scheduler/runtime path before starting NEREVAL by proving that the same durable workflow machinery also works for a JS-only pipeline with fan-out, dependency joins, artifacts, records, and site-DB writes.

The resulting path is operator-friendly as well as test-friendly. `scraper site js-demo run` now migrates the site DB, opens the engine DB, seeds a deterministic workflow, runs scheduler cycles locally, and prints the final summary result. That makes it useful both as a smoke test during development and as a debugging harness when changing JS runtime behavior.

### Prompt Context

**User prompt (verbatim):** "make a js-demo site that showcases some the js possibilities, and allows us to do runs without even doing an http fetch"

**Assistant interpretation:** Add a built-in site dedicated to demonstrating the JS execution model and expose a command that runs it end to end without any HTTP ops.

**Inferred user intent:** Create a smaller, faster feedback loop for exercising the real scheduler and JS runtime before tackling the more complex NEREVAL port.

**Commit (code):** `b6cf91b` — `Add js demo site and runner command`

### What I did

- Added the new built-in site package:
  - [site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site.go)
  - [workflow.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/workflow.go)
  - [cli.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/cli.go)
- Added the embedded JS workflow:
  - [seed.js](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/scripts/seed.js)
  - [build_item.js](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/scripts/build_item.js)
  - [summarize.js](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/scripts/summarize.js)
  - [demo.js](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/scripts/lib/demo.js)
- Added the site schema in [001_init.sql](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/migrations/001_init.sql).
- Extended the site registry and JS runner so a site can opt into extra go-go-goja module specs via:
  - [registry.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/registry/registry.go)
  - [js.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/runner/js.go)
  - [executor.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/js/runtime/executor.go)
- Wired the new site into defaults and the `site` command tree through:
  - [defaults.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/defaults/defaults.go)
  - [site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site.go)
- Added tests:
  - [site_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site_test.go)
  - [site_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site_test.go)
- Ran:
  - `go test ./pkg/sites/jsdemo ./pkg/cmd -count=1`
  - `go test ./... -count=1`

### Why

- A pure-JS site removes the HTTP runner from the loop, which makes runtime and scheduler debugging faster and much more local.
- It demonstrates that the engine’s primitives are not tied to HTML scraping specifically; they are general durable workflow primitives that JS can use for orchestrated computation as well.
- The operator-facing CLI command gives a real end-to-end smoke test, not just a unit-test fixture.

### What worked

- The same `js` runner and scheduler path used by the HTTP-backed sites worked cleanly for a JS-only workflow.
- Deterministic op IDs made it easy for the CLI to find and print the final summary result.
- The demo site now exercises:
  - relative `require("./lib/...")`
  - `require("site-db")`
  - `ctx.emit(...)`
  - `ctx.dep(...)`
  - `ctx.writeRecord(...)`
  - `ctx.writeArtifact(...)`
  - async JS function exports

### What didn't work

- No runtime failures occurred after the first implementation pass, but I did have to be careful not to stage the unrelated `ttmp/` worktree dirt from the earlier `SCRAPER-SITE-JSVERBS` doc-only ticket while committing the code change.

### What I learned

- The scheduler/runtime split is generic enough that a site can behave like a tiny workflow lab, not just a web scraper.
- Site-level module injection is useful. The `js-demo` site opts into `go-go-goja` default modules without forcing every site script in the repository to get the same JS surface area.

### What was tricky to build

- The main design edge was how to expose extra JS capabilities to one site without broadly changing the contract for every other site. The resolution was to extend the site definition with optional static module specs and let the JS executor opt in per site. That keeps the demo expressive without silently broadening the runtime surface for production scrape sites.

### What warrants a second pair of eyes

- The new per-site module injection path in [executor.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/js/runtime/executor.go) is intentionally small, but it changes the site contract. It is worth reviewing whether future sites should use it sparingly or whether there should be a stricter allowlist policy.
- The `site` command tree now supports site-specific CLI registration. That mechanism is useful, but it should stay disciplined so the top-level CLI does not become inconsistent across sites.

### What should be done in the future

- Reuse `js-demo` as the basis for any future `site op run` or jsverbs-style script-runner experiments.
- Consider adding a second demo workflow that showcases `timer.sleep(...)` explicitly now that `go-go-goja` ships a real timer module.

### Code review instructions

- Start with [cli.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/cli.go) and [workflow.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/workflow.go) to understand how the operator path seeds and runs the workflow.
- Then read [seed.js](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/scripts/seed.js), [build_item.js](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/scripts/build_item.js), and [summarize.js](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/scripts/summarize.js) in order.
- Finish with [site_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site_test.go) and [site_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site_test.go) to validate the intended behavior.

### Technical details

- CLI entrypoint:
  - `scraper site js-demo run --workflow-id demo-1 --count 3 --multiplier 4 --prefix smoke`
- Example summary math used in tests:
  - base values for `count=3`, `multiplier=5` are `5`, `10`, `15`
  - squared values are `25`, `100`, `225`
  - totals are `30` and `350`

## Step 14: Generalize js-demo run into named entrypoints

This step turned the single hardcoded `scraper site js-demo run` command into a small entrypoint tree so the CLI can exercise individual parts of the JS pipeline directly. The new shape is `scraper site js-demo run <entrypoint>`, with `seed`, `item`, and `summary` as the first supported entrypoints.

That makes the demo materially more useful for debugging. `seed` still drives the whole fan-out workflow, `item` runs the single item-building op as its own workflow, and `summary` runs the join stage by constructing item dependencies explicitly. Together they let us test each script in its intended durable environment without having to wait for an HTTP-backed site.

### Prompt Context

**User prompt (verbatim):** "add tasks to the ticket, and then impolement this. use whatever names you think are best for the entry points"

**Assistant interpretation:** Update the design ticket to reflect multi-entrypoint site runs, then implement a CLI shape that exposes the js-demo stages individually.

**Inferred user intent:** Make `site ... run` feel like a real operator harness for individual site entrypoints rather than a single hardcoded demo command.

**Commit (code):** `60d279c` — `Add js demo run entrypoints`

### What I did

- Reworked [cli.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/cli.go) so `run` is now a parent command with:
  - `seed`
  - `item`
  - `summary`
- Split workflow construction in [workflow.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/workflow.go) into:
  - `BuildSeedWorkflow`
  - `BuildItemWorkflow`
  - `BuildSummaryWorkflow`
- Kept `BuildWorkflow` as a compatibility alias to the seed entrypoint.
- Added direct scheduler coverage for the single-item path in [site_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site_test.go).
- Expanded CLI coverage in [site_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site_test.go) to cover:
  - `run seed`
  - `run item`
  - `run summary`
- Ran:
  - `go test ./pkg/sites/jsdemo ./pkg/cmd -count=1`
  - `go test ./... -count=1`

### Why

- The user wanted `site ... run` to behave more like a site verb runner.
- Individual entrypoints are faster to debug than always running the whole fan-out workflow.
- Keeping the names aligned with the actual JS stages makes the operator surface clearer than inventing abstract labels.

### What worked

- `seed` maps naturally to the existing full workflow path.
- `item` is a clean single-op workflow and gives a tight feedback loop for `build_item.js`.
- `summary` exercises the dependency-join logic by creating item ops plus a dependent summary op without going through `seed.js`.

### What didn't work

- N/A

### What I learned

- The `js-demo` site is already a good proving ground for the broader site-runner idea because its stages are small and deterministic.
- Direct op-oriented entrypoints are easier to name and reason about when they stay close to the script names.

### What was tricky to build

- The only real design choice was whether to use abstract command names like `fanout` or literal stage names like `seed`. I chose `seed`, `item`, and `summary` because they map directly onto the scripts and make the CLI easier to correlate with the code.

### What warrants a second pair of eyes

- If we later generalize this pattern beyond `js-demo`, we should decide whether site entrypoints are always stage/script-shaped or whether some sites should expose higher-level operator names instead.

### What should be done in the future

- Reuse this `run <entrypoint>` shape when building the more generic site-op runner or jsverbs-backed CLI.

### Code review instructions

- Review [cli.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/cli.go) and [workflow.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/workflow.go) together.
- Then run:
  - `scraper site js-demo run seed --count 3`
  - `scraper site js-demo run item --index 2 --multiplier 4`
  - `scraper site js-demo run summary --count 3 --multiplier 5`

### Technical details

- The result-targeting model now differs by entrypoint:
  - `seed` returns the emitted summary op result
  - `item` returns the single `build_item.js` op result
  - `summary` returns the `summarize.js` op result after generating its dependencies
