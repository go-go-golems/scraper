---
Title: Investigation diary
Ticket: SCRAPER-SITE-JSVERBS
Status: active
Topics:
    - scraping
    - go
    - goja
    - javascript
    - glazed
    - cli
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-goja/cmd/jsverbs-example/main.go
      Note: Concrete host-side composition example reviewed during the diary step
    - Path: ../../../../../../../go-go-goja/pkg/doc/10-jsverbs-example-developer-guide.md
      Note: Intern-oriented reference that influenced the shape of this ticket deliverable
    - Path: pkg/cmd/root.go
      Note: Current CLI root reviewed to confirm there is no site-js subtree yet
    - Path: pkg/cmd/site.go
      Note: Current site command only handles migration and is the likely future mount point for site js verbs
ExternalSources: []
Summary: Chronological diary for the design pass studying jsverbs and the current scraper site runtime to define a site-aware JavaScript CLI runner.
LastUpdated: 2026-03-23T14:38:00-04:00
WhatFor: Preserve the reasoning, evidence, and commands used to design a site JavaScript CLI harness for scraper.
WhenToUse: Use when implementing the site js runner later or reviewing why the design keeps scheduler op scripts and jsverbs separated.
---


# Investigation diary

## Goal

Capture the exact research work used to design a site-aware JavaScript CLI runner for `scraper` based on `go-go-goja/pkg/jsverbs`.

## Step 1: Study jsverbs and compare it with the current scraper site runtime

This step created a fresh ticket, then compared the `go-go-goja` jsverbs pipeline with the existing scheduler-oriented JavaScript runtime in `scraper`. The main conclusion is that the two systems are related but not interchangeable. `jsverbs` is a CLI command-construction pipeline built around top-level discovered functions and static metadata. The current scraper site runtime is an op executor built around `module.exports = function (ctx) { ... }` and durable workflow semantics.

That distinction turned out to be the most important design fact in the whole investigation. Once it was clear, the architecture recommendation followed naturally: keep the runtime contracts separate, add a new `verbs/` tree per site for CLI-facing jsverbs, and reuse `lib/` code underneath both sides.

### Prompt Context

**User prompt (verbatim):** "study go-go-goja and in particular go-go-goja/pkg/doc/08-jsverbs-example-overview.md and study how we can add a site js runner so that we can test the different parts of a sites from the CLI.

Create a new ticket and Create a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new documentation ticket, study the local jsverbs subsystem and the current scraper site runtime, then write an intern-oriented design and implementation guide for a site-aware JavaScript CLI runner, validate the ticket, and upload the bundle to reMarkable.

**Inferred user intent:** Add a practical operator and developer harness for site JavaScript so site logic can be exercised from the CLI without always building full workflows, but do it in a way that respects the existing architecture instead of creating a confusing second runtime.

**Commit (code):** N/A — documentation-only work, no code commit created

### What I did

- Created a new ticket:
  - `docmgr ticket create-ticket --ticket SCRAPER-SITE-JSVERBS --title "Site JavaScript runner CLI based on jsverbs" --topics scraping,go,goja,javascript,glazed,cli,architecture`
- Added the main design doc and diary doc:
  - `docmgr doc add --ticket SCRAPER-SITE-JSVERBS --doc-type design-doc --title "Site JavaScript CLI runner with jsverbs design and implementation guide"`
  - `docmgr doc add --ticket SCRAPER-SITE-JSVERBS --doc-type reference --title "Investigation diary"`
- Studied the primary jsverbs overview:
  - `go-go-goja/pkg/doc/08-jsverbs-example-overview.md`
- Studied the supporting jsverbs docs:
  - `go-go-goja/pkg/doc/09-jsverbs-example-fixture-format.md`
  - `go-go-goja/pkg/doc/10-jsverbs-example-developer-guide.md`
  - `go-go-goja/pkg/doc/11-jsverbs-example-reference.md`
- Studied the concrete jsverbs implementation:
  - `go-go-goja/cmd/jsverbs-example/main.go`
  - `go-go-goja/pkg/jsverbs/model.go`
  - `go-go-goja/pkg/jsverbs/scan.go`
  - `go-go-goja/pkg/jsverbs/binding.go`
  - `go-go-goja/pkg/jsverbs/command.go`
  - `go-go-goja/pkg/jsverbs/runtime.go`
- Studied the current scraper runtime and site seams:
  - `scraper/pkg/cmd/root.go`
  - `scraper/pkg/cmd/site.go`
  - `scraper/pkg/sites/registry/registry.go`
  - `scraper/pkg/engine/runner/js.go`
  - `scraper/pkg/js/runtime/executor.go`
  - `scraper/pkg/js/runtime/databases.go`
- Compared those abstractions with the current exercise-site scripts:
  - `scraper/pkg/sites/hackernews/scripts/seed.js`
  - `scraper/pkg/sites/hackernews/scripts/extract_frontpage.js`
  - `scraper/pkg/sites/slashdot/scripts/extract_frontpage.js`
- Wrote the design guide around:
  - current-state architecture,
  - mismatch analysis,
  - recommended split between `scripts/`, `verbs/`, and `lib/`,
  - CLI structure,
  - runtime module design,
  - phased implementation plan.

### Why

- The user explicitly pointed at `pkg/doc/08-jsverbs-example-overview.md`, which means the design should build from the actual local jsverbs implementation instead of inventing a new CLI abstraction.
- The scraper repo already has a working site runtime, so the right design question is not “how do we run JavaScript?” but “which parts of the existing runtime should a CLI harness reuse, and which parts should stay separate?”
- A new intern will need a file-by-file explanation, not a vague recommendation like “use jsverbs for site tools.”

### What worked

- The jsverbs docs and implementation were coherent enough to trace one full line from:
  - scan,
  - bind,
  - command build,
  - runtime invoke.
- The current scraper site runtime already exposes most of the host-side ingredients a CLI harness would want:
  - embedded site filesystems,
  - preconfigured DB modules,
  - site registry,
  - site-specific runtime module registration.
- The Hacker News and Slashdot exercise sites made the mismatch very obvious in a good way. Their op scripts are small and readable enough to show exactly why scanning them directly as verbs would be awkward.

### What didn't work

- The first `docmgr doc add --ticket SCRAPER-SITE-JSVERBS ...` attempt failed immediately after ticket creation with:
  - `Error: failed to find ticket directory: ticket not found: SCRAPER-SITE-JSVERBS`
- Resolution:
  - verified the ticket exists with `docmgr ticket list`,
  - re-ran the `doc add` commands successfully.
- One `rg` command used to scan site scripts failed because of a quoting mistake in zsh:
  - `zsh:1: unmatched "`
- Resolution:
  - ignored that failed convenience search and continued with direct file reads.

### What I learned

- `jsverbs` is best understood as a host-composed command pipeline, not a generic replacement for every goja runtime in the codebase.
- The best bridge between scraper ops and jsverbs is shared site libraries, not shared entrypoint semantics.
- `ScanFS(...)` is the key feature that makes the design practical for `scraper`, because the site packages already embed their JavaScript trees.
- The current site registry already hints at this direction because it has `RegisterCLI`, but the repo does not yet have a generic site-specific CLI composition layer.

### What was tricky to build

- The main conceptual trap was “we already have site JS files, so scan them with jsverbs.” That sounds elegant, but it would force CLI commands to inherit workflow concerns like dependencies, emits, and durable result envelopes. The design had to show clearly why that would be the wrong abstraction even though it looks DRY at first.
- A second tricky point was deciding how much of the current runtime should be shared. The answer is: share the embedded filesystems, preconfigured DB modules, and helper libraries; do not share the durable `ctx` contract.

### What warrants a second pair of eyes

- Whether the site registry should gain explicit `VerbsFS` and `VerbsRoot` fields, or whether a narrower convention-based approach would be enough for v1.
- Whether `RegisterCLI` should stay as a low-level escape hatch once the generic site-js runner exists.
- Whether site-verb DB writes should be allowed by default or require an explicit opt-in flag for safety.

### What should be done in the future

- Implement the design in `scraper` as a follow-up code ticket.
- Start with Hacker News and Slashdot as the first verb-enabled sites.
- Add parser verbs and DB-inspection verbs before touching NEREVAL.
- Keep the same shared `lib/` discipline so both the workflow runtime and CLI runtime call the same parsing code.

### Code review instructions

- Start with the jsverbs pipeline:
  - `go-go-goja/pkg/jsverbs/scan.go`
  - `go-go-goja/pkg/jsverbs/binding.go`
  - `go-go-goja/pkg/jsverbs/command.go`
  - `go-go-goja/pkg/jsverbs/runtime.go`
- Then compare the scraper runtime:
  - `scraper/pkg/sites/registry/registry.go`
  - `scraper/pkg/engine/runner/js.go`
  - `scraper/pkg/js/runtime/executor.go`
- Then read the site scripts:
  - `scraper/pkg/sites/hackernews/scripts/seed.js`
  - `scraper/pkg/sites/hackernews/scripts/extract_frontpage.js`
  - `scraper/pkg/sites/slashdot/scripts/extract_frontpage.js`
- Finally read the design guide and confirm the recommended split is consistent with those files.

### Technical details

- Main commands used during the research step:
  - `docmgr status --summary-only`
  - `docmgr ticket create-ticket --ticket SCRAPER-SITE-JSVERBS --title "Site JavaScript runner CLI based on jsverbs" --topics scraping,go,goja,javascript,glazed,cli,architecture`
  - `docmgr ticket list`
  - `docmgr doc add --ticket SCRAPER-SITE-JSVERBS --doc-type design-doc --title "Site JavaScript CLI runner with jsverbs design and implementation guide"`
  - `docmgr doc add --ticket SCRAPER-SITE-JSVERBS --doc-type reference --title "Investigation diary"`
  - `rg --files ...`
  - `rg -n ...`
  - `nl -ba <file> | sed -n '<range>'`
