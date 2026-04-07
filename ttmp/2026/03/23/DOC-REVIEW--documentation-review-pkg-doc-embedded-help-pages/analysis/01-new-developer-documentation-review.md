---
Title: New Developer Documentation Review
Ticket: DOC-REVIEW
Status: active
Topics:
    - documentation
    - scraper
    - onboarding
    - architecture
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Honest new-developer review of the five embedded help pages in pkg/doc, with concrete improvement recommendations"
LastUpdated: 2026-03-23T20:44:44.18921787-04:00
WhatFor: "Guide improvements to embedded documentation"
WhenToUse: "When planning doc improvements for the scraper project"
---

# New Developer Documentation Review

**Reviewer posture:** First day on the team. No prior context on the codebase. Reading `pkg/doc/` as my primary onboarding material before touching code.

## Documents Reviewed

| # | File | Type | Title |
|---|------|------|-------|
| 1 | `topics/scraper-architecture-overview.md` | GeneralTopic | Scraper Architecture Overview |
| 2 | `topics/scraper-runtime-model.md` | GeneralTopic | Scraper Runtime Model |
| 3 | `topics/scraper-queue-policies-and-rate-limiting.md` | GeneralTopic | Queue Policies and Rate Limiting |
| 4 | `tutorials/scraper-new-developer-onboarding.md` | Tutorial | New Developer Onboarding |
| 5 | `tutorials/scraper-adding-a-site.md` | Tutorial | Adding a New Site |

---

## Overall Assessment

**The documentation is genuinely good.** It is well above average for a project of this size and age. The writing is clear, the structure is logical, the troubleshooting tables are excellent, and the progressive complexity model (js-demo → hackernews → slashdot → nereval) is a smart pedagogical choice. A new developer reading these pages in order will build a correct mental model of the system.

That said, there are real gaps and inaccuracies that would trip up the careful reader. The rest of this review catalogs them honestly, grouped by severity.

---

## Critical Issues (would block or mislead a new developer)

### 1. ~~Two site-authoring patterns exist, but only one is documented~~ RESOLVED

**Status: Fixed.** Hackernews and slashdot were converted from Go-coded workflow builders (`cli.go` + `workflow.go` + `cliutil`) to JS submit verbs. All four built-in sites now follow the same submit-verb pattern with `verbs/` directories and `VerbsFS`/`VerbsRoot` in their definitions. The `cliutil` package is now empty. The documentation's single-pattern model is now accurate.

**Remaining note:** The `RegisterCLI` field still exists in `registry.Definition` for extensibility, but no built-in site uses it. The docs do not need to explain two competing patterns.

### 2. The `registry.Definition` struct has fields the docs don't mention

The "Adding a Site" tutorial says the important fields are: `Name`, `DatabaseFileName`, `ScriptsFS/Root`, `VerbsFS/Root`, `SQLMigrationsFS/Root`.

The actual struct also has:
- `Modules []gggengine.ModuleSpec` — nereval uses this to inject `DefaultRegistryModules()`
- `JSMigrationsFS` / `JSMigrationsRoot` — JS-based migrations (unused so far but in the API)
- `HelpFS` / `HelpRoot` — per-site help pages
- `QueuePolicies map[model.QueueKey]model.QueuePolicy` — site-level queue policy
- `RuntimeModuleRegistrars` — additional runtime modules
- `RegisterCLI func(*cobra.Command) error` — Go-coded CLI registration (unused by built-in sites after the conversion)

Of these, `Modules` is load-bearing (nereval depends on it). The others are extension points that a new contributor may need.

**Recommendation:** Mention these briefly in the adding-a-site tutorial as optional/advanced fields.

### 3. The JS API is almost completely undocumented

The runtime model page says scripts can use `ctx.input`, `ctx.dep(...)`, `ctx.emit(...)`, `require("site-db")`, `require("scraper-db")`. That's the entire API documentation.

The actual API is much richer:

**Execution-time `ctx` (scripts/):**
- `ctx.site` — current site name
- `ctx.now` — RFC3339Nano timestamp
- `ctx.workflow` — `{id, site, name, status, input, metadata}`
- `ctx.op` — `{id, workflowID, site, kind, queue, dedupKey, metadata}`
- `ctx.lease` — `{workerID, token, acquiredAt, expiresAt}`
- `ctx.input` — decoded op input
- `ctx.log(...args)` — structured logging
- `ctx.dep(opID)` — returns `{opID, data, records, artifacts, emittedIDs, completedAt, error}` or null
- `ctx.emit(spec)` — emits child op, returns ID
- `ctx.writeRecord(collection, key, data)` — writes to site DB records
- `ctx.writeArtifact(spec)` — writes artifact

**Submission-time `ctx` (verbs/):**
- `ctx.site`, `ctx.now`, `ctx.values`, `ctx.sections`, `ctx.command`, `ctx.workflow`
- `ctx.log(...args)`
- `ctx.emit(spec)`
- `ctx.setWorkflowName(name)`
- `ctx.setWorkflowMetadata(obj)`
- `ctx.setTargetOpID(opID)`

**OpSpec shape for `ctx.emit()`:**
- `{id, kind, queue, dedupKey, input, dependsOn: [{opID, required}], retry: {...}, metadata: {script: "..."}, workflowID, site, parentID}`

**Database modules:**
- `require("site-db").exec(sql, ...params)` — returns affected count
- `require("site-db").query(sql, ...params)` — returns array of row objects

A new developer writing their first site script would have to reverse-engineer all of this from example code and Go source. This is the single most impactful documentation addition possible.

**Recommendation:** Create a new reference page `scraper-js-api-reference.md` with full signatures, types, and examples for both contexts.

---

## Moderate Issues (would cause confusion or wasted time)

### 4. The onboarding tutorial should explain where CLI flags come from

All four built-in sites now use JS submit verbs. CLI flags like `--count`, `--max-pages`, `--base-url` are defined in `__verb__` metadata inside `verbs/*.js` files and automatically wired into Cobra by the submit-verb host. This is a key architectural insight that's never explained. A new developer who hits a flag error has no debugging path without understanding this mechanism.

**Recommendation:** Add a brief note explaining that `scraper site <site> run <verb>` flags are auto-generated from JS `__verb__` metadata.

### 5. ~~The `cliutil` package is invisible in documentation~~ RESOLVED

**Status: Fixed.** The `cliutil` package is now empty. Hackernews and slashdot were converted to JS submit verbs, eliminating the Go-coded CLI path. No documentation needed.

### 6. `pkg/sites/migrate/` is undocumented

The migration manager handles site DB creation and schema migration. The adding-a-site tutorial says "define migrations in `migrations/`" but never explains how they're discovered, applied, or what the manager does.

**Recommendation:** A paragraph or two in the adding-a-site tutorial about the migration lifecycle.

### 7. The `pkg/engine/config/` package is not mentioned

There's a config package with HTTP configuration (user agent, timeout). The runner and scheduler use it. Not mentioned anywhere.

**Recommendation:** Brief mention in the architecture overview.

### 8. The `pkg/engine/runner/runner.go` registry is not explained

The runner registry (`runner.NewRegistry()`, `runners.Register(...)`) is the mechanism that maps op kinds (`"js"`, `"http/fetch"`) to runner implementations. The docs mention runners but never explain how they're discovered or registered.

**Recommendation:** A paragraph in the runtime model page.

### 9. `pkg/js/runtime/promises.go` is not mentioned

The JS runtime supports async/Promise-based scripts (confirmed by `jsdemo/scripts/build_item.js` which uses `async function`). This is never mentioned in the docs.

**Recommendation:** Note in the JS API reference that scripts can be async.

---

## Minor Issues (polish, accuracy, completeness)

### 10. Absolute paths in markdown links

All file references use absolute paths like `[pkg/cmd/root.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/root.go)`. These break for anyone whose checkout is in a different directory. They also embed a workspace path that includes a date, which will be wrong in the future.

**Recommendation:** Use relative paths from the repo root, or just use the path as text without a link (the `scraper help` viewer probably can't follow filesystem links anyway).

### 11. Cross-references use `scraper help <slug>` syntax

The "See Also" sections say things like `[scraper-runtime-model](scraper help scraper-runtime-model)`. These aren't valid markdown links. They tell the reader to run a CLI command, which is fine, but the link syntax is misleading.

**Recommendation:** Either make them plain text ("Run `scraper help scraper-runtime-model`") or use a relative markdown link if the help system supports it.

### 12. The onboarding tutorial references ticket docs with absolute paths

Step 7 links to `ttmp/2026/03/23/SCRAPER-DESIGN--...` paths. These paths will become stale as the project evolves and tickets accumulate.

**Recommendation:** Consider either making the ticket references more generic ("search for SCRAPER-DESIGN in the ttmp directory") or noting the date context.

### 13. All five pages are marked `IsTopLevel: true` and `ShowPerDefault: true`

This means a new developer running `scraper help` sees all five pages at the top level. That's fine for now with five pages, but as docs grow, some curation will be needed. The onboarding tutorial is the natural entry point and should probably be the one page that shows by default, with the others linked from it.

**Recommendation:** Consider making only the onboarding tutorial `ShowPerDefault: true` and having it link to the rest.

### 14. No mention of the `engine config` concept

The `pkg/engine/config/` package defines `HTTP` config with `UserAgent` and `Timeout`. This affects how the HTTP runner behaves. Not documented.

### 15. ~~The architecture overview says "site packages provide submit verbs"~~ RESOLVED

All four built-in sites now provide submit verbs after the hackernews/slashdot conversion.

### 16. Missing: how to run a single site in live mode

All tutorials emphasize fixtures and smoke tests (rightly so for onboarding). But there's no doc page that explains how to actually run a site against a live website in production. The nereval section says "do not run it live as part of onboarding" but never says where to learn the live path.

**Recommendation:** Either a brief "Running Live" section in the architecture overview, or a note pointing to the intended production usage pattern.

### 17. Missing: error handling patterns in JS scripts

Scripts can return `{error: {code, message, retryable, details}}`. The retry policy can be set on ops. None of this is documented. A new developer writing a scraper for a flaky site would need to know this.

---

## What the Docs Get Right (keep these)

These are strengths worth preserving:

1. **The progressive complexity model** (js-demo → hackernews → slashdot → nereval) is excellent pedagogy. Each site proves one additional capability.

2. **The troubleshooting tables** at the end of every page are surprisingly useful. They capture real failure modes and avoid the "just restart" non-answers.

3. **The separation between conceptual pages and tutorials** is clean. The architecture and runtime model pages explain *why*, the tutorials explain *how*.

4. **The "What To Read In Code" sections** with ordered file lists are extremely valuable for a new developer. Reading code in the right order saves hours.

5. **The engine DB vs site DB distinction** is explained three times in slightly different ways across the pages, and that redundancy is helpful — it's the kind of thing that takes repetition to internalize.

6. **The clear constraint statements** like "a submit verb is not a worker" and "it does not crawl pages for minutes" prevent the most common architectural misunderstanding.

7. **Embedding docs in the binary** via `go:embed` and the help system is a great choice. Docs stay in sync with the code they describe.

---

## Recommended New Pages

After the hackernews/slashdot conversion, the two-pattern gap is resolved. The remaining high-value addition:

| Priority | Slug | Type | Content |
|----------|------|------|---------|
| **P0** | `scraper-js-api-reference` | Reference | Full ctx.* API for both submission-time and execution-time, OpSpec schema, database module API, return envelopes, async support |

## Recommended Edits to Existing Pages

| Page | Edit |
|------|------|
| `scraper-architecture-overview` | Mention the runner registry. Mention `pkg/engine/config/`. |
| `scraper-runtime-model` | Mention async script support. Link to the JS API reference. |
| `scraper-adding-a-site` | Mention the migration manager lifecycle. Briefly note advanced Definition fields (Modules, QueuePolicies). |
| `scraper-new-developer-onboarding` | Explain that CLI flags come from JS `__verb__` metadata. Update the hackernews step to use the new verb-based CLI (now `--base-url` and `--max-pages` instead of `--fixture`). |
| All pages | Convert absolute paths to repo-relative. Fix "See Also" link syntax. |

---

## Summary

The documentation is a solid foundation — well-structured, well-written, and clearly authored by someone who understands the system deeply. With the hackernews/slashdot conversion to JS submit verbs, the previous biggest gap (two competing site-authoring patterns) is now resolved. The remaining critical gap is the missing JS API reference, which forces new developers to read Go source code to write JavaScript. Fixing that and applying the minor edits above would make the docs genuinely excellent for onboarding.
