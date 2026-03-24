---
Title: Pure JS site conversion plan for Hacker News and Slashdot
Ticket: SCRAPER-PURE-JS-SITES
Status: active
Topics:
    - scraper
    - javascript
    - jsverbs
    - cli
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/sites/hackernews/site.go
      Note: Current Hacker News site definition still uses bespoke Go CLI registration
    - Path: pkg/sites/hackernews/cli.go
      Note: Site-specific Go CLI wrapper to remove
    - Path: pkg/sites/hackernews/workflow.go
      Note: Go workflow builders to remove after moving submission logic into JS verbs
    - Path: pkg/sites/slashdot/site.go
      Note: Current Slashdot site definition still uses bespoke Go CLI registration
    - Path: pkg/sites/slashdot/cli.go
      Note: Site-specific Go CLI wrapper to remove
    - Path: pkg/sites/slashdot/workflow.go
      Note: Go workflow builders to remove after moving submission logic into JS verbs
    - Path: pkg/sites/jsdemo/site.go
      Note: Existing pure-JS submit-verb site to follow as the pattern
    - Path: pkg/sites/jsdemo/verbs/seed.js
      Note: Canonical example of a JS submit verb
    - Path: pkg/sites/submitverbs/host.go
      Note: Existing Go host that should continue to run submit verbs and prepare DBs
ExternalSources: []
Summary: Conversion plan for removing site-specific Go submit wrappers from Hacker News and Slashdot, replacing them with JS submit verbs like js-demo while keeping only the minimal site-definition Go that the current runtime still needs.
LastUpdated: 2026-03-23T21:30:00-04:00
WhatFor: Tracks the cleanup needed to make Hacker News and Slashdot follow the same JS-first site pattern as js-demo.
WhenToUse: Use when implementing or reviewing the move from bespoke Go site entrypoints to JS submit verbs.
---

# Pure JS site conversion plan for Hacker News and Slashdot

## Executive Summary

Hacker News and Slashdot are already mostly JS sites at execution time. Their scrape logic lives in `scripts/*.js`, including the pagination fan-out. What still remains in Go is the submission side:

- site-specific CLI command registration
- Go workflow-builder helpers
- a site-specific local runner wrapper

This ticket proposes removing those site-specific Go submission layers and replacing them with the same JS submit-verb model that `js-demo` already uses.

The target state is:

- `verbs/*.js` defines site entrypoints like `seed` and `extract-frontpage`
- the generic submit-verb host in Go discovers and runs those verbs
- the worker still processes the resulting durable ops
- only minimal site-definition Go remains

## Problem Statement

Today the site model is inconsistent:

- `js-demo` is JS-first for submission
- `hackernews` and `slashdot` still rely on bespoke Go wrappers in [cli.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/cli.go) and [cli.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/slashdot/cli.go)

That inconsistency causes a few problems:

- there are two patterns to learn
- site behavior is split between JS and Go
- changing submit-time behavior on HN/Slashdot still requires Go edits
- the codebase looks less ready for a generalized JS submit-verb model than it really is

The goal of this ticket is to remove the site-specific Go submission code so there is one clear path: JS verbs define the site entrypoints.

## Proposed Solution

For each of Hacker News and Slashdot:

1. Add a `verbs/` tree.
2. Implement `seed.js` and `extract_frontpage.js` submit verbs there.
3. Move the workflow-building logic from `workflow.go` into those submit verbs.
4. Update `site.go` to expose:
   - `VerbsFS`
   - `VerbsRoot`
5. Delete:
   - `cli.go`
   - `workflow.go`
   - `RegisterCLI` from the site definition

The submit verbs should build the same initial durable op graphs that the current Go builders create.

### Remaining Go in the sites

Yes, some Go still needs to remain today.

The current runtime still needs site-level Go for:

- embedded file systems via `//go:embed`
- registration into the site registry
- site definition metadata:
  - site name
  - DB filename
  - scripts FS/root
  - verbs FS/root
  - migrations FS/root
  - optional modules and queue policies
- fixture helpers if we still want embedded fixture loading from Go tests

So the near-term target is not "zero Go files per site." It is:

- no site-specific Go submission logic
- minimal declarative Go definition only

That is the clean path toward more JS without trying to solve site discovery from disk in the same ticket.

## Design Decisions

### Keep the generic Go host

The Go submit-verb host in [host.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/submitverbs/host.go) should stay. It does useful system work:

- open engine DB
- open site DB
- run site migrations
- execute one submit verb
- persist the initial workflow

That is framework-level work, not site-specific work.

### Remove site-specific Go wrappers

The code in:

- [workflow.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/workflow.go)
- [workflow.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/slashdot/workflow.go)
- [cli.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/cli.go)
- [cli.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/slashdot/cli.go)

should go away because it is now duplicating a pattern that JS submit verbs already cover.

### Keep the op scripts unchanged where possible

The current `scripts/*.js` already own the real scrape behavior. The conversion should avoid unnecessary churn there. The main change is on the submission side, not the execution side.

## Alternatives Considered

### Keep the Go wrappers and add JS verbs alongside them

Rejected because it keeps two sources of truth and delays the cleanup.

### Remove all Go from site packages

Rejected for now because the current runtime still depends on Go site definitions and embedded file systems. That is a larger architectural change than this ticket needs.

### Move fixture/local-run behavior into each site again

Rejected because if we want smoke helpers, they should become generic host functionality rather than site-specific wrappers.

## Implementation Plan

### Phase 1. Add JS submit verbs

- Add `verbs/seed.js` for Hacker News
- Add `verbs/extract_frontpage.js` for Hacker News
- Add `verbs/seed.js` for Slashdot
- Add `verbs/extract_frontpage.js` for Slashdot
- Mirror current defaults:
  - base URL
  - max pages
  - workflow naming
  - target op ID selection

### Phase 2. Update site definitions

- Update [site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/site.go) to embed `verbs/*.js`
- Set `VerbsFS` and `VerbsRoot`
- Remove `RegisterCLI`
- Do the same for Slashdot

### Phase 3. Delete site-specific Go submission code

- Delete Hacker News `cli.go`
- Delete Hacker News `workflow.go`
- Delete Slashdot `cli.go`
- Delete Slashdot `workflow.go`

### Phase 4. Replace tests

- Update tests to call the generic submit-verb path instead of the deleted bespoke commands
- Keep fixture-based coverage if still useful
- Verify:
  - `scraper site hackernews seed ...`
  - `scraper site hackernews extract-frontpage ...`
  - `scraper site slashdot seed ...`
  - `scraper site slashdot extract-frontpage ...`

### Phase 5. Decide on smoke/local-run ergonomics

- If the generic submit-verb path is enough, stop there
- If we still want `--fixture` or local inline completion, add that as generic host functionality rather than reintroducing site-specific Go

## Open Questions

### Do we still need a generic local runner mode?

Maybe. The deleted Go wrappers currently provide fixture-backed smoke execution in one command. If that remains valuable, it should become generic submit-host behavior, not per-site custom code.

### Should the public command names stay exactly the same?

Probably yes. The internals can change while the operator-facing command names remain stable.

### Do we want to keep fixture helpers in Go?

Probably yes for tests, even after removing the Go CLI/workflow builders.

## References

- [site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site.go)
- [seed.js](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/verbs/seed.js)
- [cli.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/cli.go)
- [workflow.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/workflow.go)
- [cli.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/slashdot/cli.go)
- [workflow.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/slashdot/workflow.go)
- [host.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/submitverbs/host.go)
