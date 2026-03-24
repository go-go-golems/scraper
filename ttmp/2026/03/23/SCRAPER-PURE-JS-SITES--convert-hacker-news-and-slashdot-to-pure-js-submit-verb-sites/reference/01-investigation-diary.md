---
Title: Investigation diary
Ticket: SCRAPER-PURE-JS-SITES
Status: active
Topics:
    - scraper
    - javascript
    - jsverbs
    - cli
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/sites/hackernews/cli.go
      Note: Current Hacker News wrapper targeted for removal
    - Path: pkg/sites/slashdot/cli.go
      Note: Current Slashdot wrapper targeted for removal
    - Path: pkg/sites/jsdemo/verbs/seed.js
      Note: Existing pattern to copy
ExternalSources: []
Summary: Short diary for the pure-JS-site cleanup ticket, recording the current inconsistency, the chosen target state, and the remaining Go that still needs to stay for now.
LastUpdated: 2026-03-23T21:30:00-04:00
WhatFor: Records the reasoning behind converting Hacker News and Slashdot to JS submit verbs.
WhenToUse: Use when reviewing why site-specific Go submission code is being removed and what Go still remains intentionally.
---

# Investigation diary

## Goal

Create a focused cleanup ticket for converting Hacker News and Slashdot from bespoke Go submission wrappers to the same JS submit-verb pattern used by `js-demo`.

## Context

Current state:

- `js-demo` is already JS-first for submission
- `hackernews` and `slashdot` still have Go-side `cli.go` and `workflow.go`
- their actual scrape behavior is already mostly in JS scripts

That means the remaining inconsistency is mainly on the submit side, not the execution side.

## Quick Reference

Target cleanup:

- add `verbs/*.js` to HN and Slashdot
- set `VerbsFS` and `VerbsRoot`
- remove site-specific `RegisterCLI`
- delete site-specific `cli.go`
- delete site-specific `workflow.go`

Go that still remains for now:

- `site.go`
- embedded FS declarations
- site registry definition
- migrations wiring
- optional fixtures and modules

## Usage Examples

Desired end state examples:

```text
scraper site hackernews seed --base-url ... --max-pages ...
scraper site hackernews extract-frontpage --base-url ... --max-pages ...
scraper site slashdot seed --base-url ... --max-pages ...
scraper site slashdot extract-frontpage --base-url ... --max-pages ...
```

with those commands coming from JS submit verbs rather than bespoke Go registration.

## Outcome

Implemented in the codebase:

- Hacker News now exposes JS submit verbs under `pkg/sites/hackernews/verbs/`
- Slashdot now exposes JS submit verbs under `pkg/sites/slashdot/verbs/`
- both site definitions now set `VerbsFS` and `VerbsRoot`
- both site-specific `cli.go` and `workflow.go` files were removed
- the old `pkg/sites/cliutil/http_runner.go` helper was removed because nothing still used it

Validation run:

```bash
go test ./... -count=1
```

Result:

- all tests passed

## Related

- [01-pure-js-site-conversion-plan-for-hacker-news-and-slashdot.md](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-PURE-JS-SITES--convert-hacker-news-and-slashdot-to-pure-js-submit-verb-sites/design-doc/01-pure-js-site-conversion-plan-for-hacker-news-and-slashdot.md)
- [site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site.go)
