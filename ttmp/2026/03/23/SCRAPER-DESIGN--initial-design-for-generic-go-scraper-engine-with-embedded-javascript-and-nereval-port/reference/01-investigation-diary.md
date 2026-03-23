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
    - Path: ttmp/2026/03/23/SCRAPER-DESIGN--initial-design-for-generic-go-scraper-engine-with-embedded-javascript-and-nereval-port/sources/local/scraper.md
      Note: Imported source investigated during the diary step
ExternalSources:
    - local:scraper.md
Summary: Chronological diary of the initial SCRAPER-DESIGN research and documentation pass.
LastUpdated: 2026-03-23T12:10:00-04:00
WhatFor: Record the research steps, commands, decisions, and rough edges encountered while producing the initial design guide.
WhenToUse: Use when reviewing how the initial design doc was produced or when continuing the implementation work later.
---


# Investigation diary

## Goal

Capture the exact research and documentation work used to create the initial design for the new Go/goja scraper engine and the NEREVAL port plan.

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

**Commit (code):** Pending at the time of writing this diary entry; the bootstrap milestone is prepared for commit after validation and ticket updates.

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
