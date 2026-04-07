---
Title: Site JavaScript CLI runner with jsverbs design and implementation guide
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
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-goja/pkg/doc/08-jsverbs-example-overview.md
      Note: Primary jsverbs overview that frames the proposed CLI runner
    - Path: ../../../../../../../go-go-goja/pkg/jsverbs/binding.go
      Note: Binding modes and shared binding-plan logic shape how site verbs should receive CLI input
    - Path: ../../../../../../../go-go-goja/pkg/jsverbs/runtime.go
      Note: Runtime overlay and invocation path show how a host-composed jsverbs runner works
    - Path: ../../../../../../../go-go-goja/pkg/jsverbs/scan.go
      Note: ScanFS and registry construction are central to the proposed site-verb loader
    - Path: pkg/js/runtime/executor.go
      Note: Current durable site script runtime that should remain separate from jsverbs
    - Path: pkg/sites/hackernews/scripts/extract_frontpage.js
      Note: Concrete op-script example used to explain why direct jsverbs scanning would be the wrong default
    - Path: pkg/sites/registry/registry.go
      Note: Current site definition is the natural place to add verb-specific FS metadata
ExternalSources: []
Summary: Detailed design for adding a site-aware JavaScript CLI runner to scraper by reusing go-go-goja jsverbs alongside the existing scheduler-facing site script runtime.
LastUpdated: 2026-03-23T14:38:00-04:00
WhatFor: Explain how jsverbs works today, how it differs from the existing scraper site runtime, and how to add a site-aware CLI harness that lets operators test site logic from the command line.
WhenToUse: Use when implementing CLI-testable site JavaScript commands, extending the site registry, or deciding how scheduler-driven site scripts and jsverbs should coexist.
---


# Site JavaScript CLI runner with jsverbs design and implementation guide

## Executive Summary

The current `scraper` repository has a working scheduler-facing JavaScript runtime, but it does not yet have a good CLI harness for exercising site code in isolation. Site logic today is executed as workflow ops: a Go scheduler leases an op, picks a runner by kind, loads a site-scoped embedded script tree, injects `ctx`, and expects the script to return an `OpResult` envelope. That runtime is correct for durable workflow execution, but it is awkward for quick operator feedback because everything has to be wrapped as a workflow op.

`go-go-goja/pkg/jsverbs` already solves a different but adjacent problem: scan JavaScript from a directory or `fs.FS`, compile discovered functions into Glazed commands, and invoke them through goja with typed CLI bindings. The most important design conclusion from this ticket is that `scraper` should not try to force the current op scripts directly into `jsverbs`. The current op scripts export one `module.exports = function (ctx) { ... }` entrypoint and expect workflow metadata, dependencies, durable emit semantics, and op envelopes. `jsverbs` expects top-level functions plus static `__verb__` metadata. Those are different contracts.

The recommended architecture is therefore a split model:

- keep `scripts/` for scheduler-facing op scripts,
- add a new optional `verbs/` tree per site for CLI-facing jsverbs,
- keep `lib/` as the shared layer that both sides call into,
- expose preconfigured runtime modules to the verb runtime just like the op runtime already exposes `site-db` and `scraper-db`,
- mount the resulting commands under a dedicated CLI subtree such as `scraper site js <site> ...`.

That gives operators a fast harness for debugging parsers, projections, and small site-specific workflows from the CLI without collapsing the durable op runtime and the interactive verb runtime into one muddled abstraction.

## Problem Statement

The user wants a “site JS runner” so different parts of a site can be tested from the CLI. Today the repository does not provide that. The only production JavaScript execution path is the scheduler-oriented `js` op runner.

The resulting problems are practical:

- validating a site parser requires building a workflow,
- debugging small JavaScript helpers requires a workflow-shaped context,
- there is no user-facing CLI subtree for site-specific JS tools,
- the existing `RegisterCLI` field in the site registry exists as an extension point but is not currently used,
- the exercise sites now prove the engine path end to end, but they still do not give the operator a small, direct harness for “just run the extractor with this fixture.”

The new CLI runner therefore needs to satisfy four goals at once:

1. It must feel like a normal Glazed/Cobra CLI.
2. It must reuse site-owned embedded files rather than introduce an unrelated ad hoc loading model.
3. It must not break or distort the durable workflow runtime.
4. It must be understandable to a new engineer and testable with fixtures.

## Scope

This ticket is about design and implementation guidance for a site-aware CLI JavaScript runner. It is not a code implementation ticket.

In scope:

- studying `go-go-goja/pkg/jsverbs`,
- studying the current scraper site runtime,
- defining how a site-specific CLI runner should fit into the current architecture,
- defining CLI shape, file layout, runtime module shape, and implementation phases,
- explaining what should be tested and what should remain separate.

Out of scope:

- implementing the command,
- implementing NEREVAL,
- changing the durable workflow contracts,
- replacing the current scheduler-facing JS runtime.

## Current State

### Current jsverbs capabilities in go-go-goja

The example runner in `go-go-goja` scans JavaScript, builds commands, and mounts them under a normal Cobra root. The top-level flow is in [cmd/jsverbs-example/main.go](../../../../../../../go-go-goja/cmd/jsverbs-example/main.go):

- `jsverbs.ScanDir(dir)` scans JS sources before command registration.
- `registry.AddSharedSection(...)` can register host-owned shared CLI flag sections.
- `registry.Commands()` converts discovered verbs into Glazed commands.
- `cli.AddCommandsToRootCommand(...)` mounts them under Cobra.
- Glazed logging and help are initialized on the root command.

The scanning API is broader than the example command. `pkg/jsverbs/scan.go` exposes:

- `ScanDir(...)`
- `ScanFS(...)`
- `ScanSource(...)`
- `ScanSources(...)`

This matters directly for `scraper`, because sites already expose an embedded `fs.FS` for scripts. A future site CLI runner does not need to write temporary files to disk just to use `jsverbs`; it can scan an embedded tree with `ScanFS(...)`.

The current `Registry` model in [pkg/jsverbs/model.go](../../../../../../../go-go-goja/pkg/jsverbs/model.go) is also a good fit for host-side composition:

- it records files, discovered verbs, and diagnostics,
- it supports registry-level shared sections through `AddSharedSection(...)`,
- it resolves sections local-first and shared-second,
- it stores module contents so runtime execution does not have to re-read disk.

The binding contract is explicit in [pkg/jsverbs/binding.go](../../../../../../../go-go-goja/pkg/jsverbs/binding.go). A verb parameter can be populated in one of four modes:

- positional field value,
- named section object,
- `all`,
- `context`.

The `context` binding is especially relevant for `scraper`, because it already gives host-supplied metadata to the JavaScript function without pretending to be the durable `ctx` object from the workflow runtime.

Command compilation in [pkg/jsverbs/command.go](../../../../../../../go-go-goja/pkg/jsverbs/command.go) already solves several CLI problems we do not want to reimplement in `scraper`:

- field/section translation into Glazed schema,
- writer-style versus row-style output,
- default section handling,
- choice and list handling,
- shared section resolution.

Runtime invocation in [pkg/jsverbs/runtime.go](../../../../../../../go-go-goja/pkg/jsverbs/runtime.go) also matters:

- source is loaded from the registry with a loader,
- an overlay captures top-level functions into `__glazedVerbRegistry`,
- parsed Glazed values are mapped back into JS arguments,
- promises are awaited,
- relative `require()` continues to work.

### Current scraper site runtime

The current site registry in [pkg/sites/registry/registry.go](../../../../../../pkg/sites/registry/registry.go) already exposes a site-scoped JavaScript filesystem:

- `ScriptsFS`
- `ScriptsRoot`
- `RuntimeModuleRegistrars`
- `RegisterCLI`

The important detail is that `RegisterCLI` exists but is not consumed anywhere in the production root command today. The root command in [pkg/cmd/root.go](../../../../../../pkg/cmd/root.go) only adds:

- `engine`
- `site`
- `version`

The current `site` command in [pkg/cmd/site.go](../../../../../../pkg/cmd/site.go) only handles database migration. It does not expose site-specific runtime tools.

The existing scheduler-facing `js` runner in [pkg/engine/runner/js.go](../../../../../../pkg/engine/runner/js.go) is site-aware but op-oriented:

- it resolves the site definition,
- it builds an executor with the site script filesystem,
- it passes `ScraperDB` and `SiteDB`,
- it executes one script selected by op metadata.

The executor in [pkg/js/runtime/executor.go](../../../../../../pkg/js/runtime/executor.go) expects the following contract:

- an op metadata key `script`,
- `module.exports = function (ctx) { ... }` or default export,
- workflow/op/lease/input data marshaled into `ctx`,
- runtime helpers like:
  - `ctx.log(...)`
  - `ctx.dep(opID)`
  - `ctx.emit(...)`
  - `ctx.writeRecord(...)`
  - `ctx.writeArtifact(...)`
- durable result-envelope semantics.

The runtime also exposes preconfigured `site-db` and `scraper-db` modules through [pkg/js/runtime/databases.go](../../../../../../pkg/js/runtime/databases.go).

### Current site script shape

The exercise sites make the current mismatch concrete.

Hacker News `seed.js` in [pkg/sites/hackernews/scripts/seed.js](../../../../../../pkg/sites/hackernews/scripts/seed.js) is a scheduler op entrypoint:

```js
module.exports = function (ctx) {
  const fetchID = ctx.emit({ ... });
  const extractID = ctx.emit({ ...metadata: { script: "extract_frontpage.js" }... });
  return { data: { fetchID, extractID } };
};
```

Hacker News `extract_frontpage.js` in [pkg/sites/hackernews/scripts/extract_frontpage.js](../../../../../../pkg/sites/hackernews/scripts/extract_frontpage.js) also expects the durable op runtime:

- it reads `ctx.input.fetchedOpID`,
- it calls `ctx.dep(...)`,
- it reads dependency artifacts,
- it writes into `site-db`,
- it returns an `error` or `data` envelope.

Slashdot uses the same pattern in [pkg/sites/slashdot/scripts/extract_frontpage.js](../../../../../../pkg/sites/slashdot/scripts/extract_frontpage.js).

These scripts are not bad fits for CLI use because they are “JavaScript.” They are bad fits because they encode workflow semantics rather than command semantics.

## Architecture Mismatch

The key difference between the two systems is this:

```text
jsverbs
  JavaScript function
    -> scan metadata
    -> compile command
    -> parse CLI values
    -> invoke function(args...)

scraper op runtime
  workflow op
    -> select site + script by metadata.script
    -> build durable ctx
    -> invoke exported function(ctx)
    -> capture emitted ops / artifacts / records / errors
```

The current site scripts do not expose top-level verbs, and `jsverbs` does not know anything about workflow dependencies, durable emits, or op envelopes.

That creates three design options:

1. Scan the existing `scripts/` tree directly with `jsverbs`.
2. Teach `jsverbs` about the durable op-style `ctx` model.
3. Keep the two runtimes separate, but let them share libraries and site registration.

Option 3 is the recommended direction.

## Why scanning the current op scripts directly is the wrong default

At first glance, it is tempting to point `jsverbs.ScanFS(...)` at `ScriptsFS` and call it done. That would be structurally wrong for several reasons.

### Problem 1: discovery shape mismatch

`jsverbs` discovers top-level functions and top-level `__verb__` metadata. The current op scripts export one module function. They do not define the function tree that `jsverbs` expects to discover.

### Problem 2: accidental helper exposure

If we scan the full site `scripts/` tree, helper libraries under `lib/` may contain top-level functions that look like CLI verbs even though they are only library functions.

### Problem 3: workflow-only semantics

The current op scripts assume:

- workflow metadata,
- dependency lookup,
- emitted child ops,
- durable result envelopes.

That is the wrong mental model for a small operator verb such as:

- “parse this local fixture and print extracted rows,”
- “show normalized story projection from this HTML file,”
- “run the front-page parser against a saved artifact.”

### Problem 4: CLI ergonomics become poor

If the operator must provide synthetic `workflow`, `lease`, `op`, and dependency envelopes just to run a parser, the command surface is technically powerful but practically unusable.

## Recommended Solution

### Core recommendation

Add an optional `verbs/` tree to each site package and build a dedicated site-JS CLI harness on top of `go-go-goja/pkg/jsverbs`.

Keep three layers separate:

```text
site package
├── scripts/        durable workflow op entrypoints
├── verbs/          CLI entrypoints compiled through jsverbs
└── lib/            shared pure helpers used by both
```

This lets the system preserve both strengths:

- durable workflow execution stays durable and explicit,
- CLI exploration stays interactive and ergonomic.

### The role of each layer

#### `scripts/`

Use for:

- scheduler-driven op entrypoints,
- fan-out logic,
- dependency inspection,
- durable emits,
- result-envelope production.

Expected shape:

```js
module.exports = function (ctx) {
  // workflow/op semantics
}
```

#### `verbs/`

Use for:

- CLI-testable parsing commands,
- one-off site diagnostics,
- fixture loading and inspection,
- DB inspection or projection-oriented commands,
- narrow site-specific operator tools.

Expected shape:

```js
__section__("input", { ... });

function parseFrontpage(input, meta) {
  const frontpage = require("../lib/frontpage.js");
  return frontpage.extractStories(input.html, input.baseURL);
}

__verb__("parseFrontpage", {
  command: "parse-frontpage",
  sections: ["input"],
  fields: {
    input: { bind: "input" },
    meta: { bind: "context" }
  }
});
```

#### `lib/`

Use for:

- pure parsing functions,
- normalization helpers,
- shared projection helpers,
- common URL or HTML utilities.

This is the part that should be shared aggressively. The scheduler runtime and CLI runtime should share logic through libraries, not by pretending they are the same execution environment.

## Proposed Site Registry Changes

The current `Definition` already has `ScriptsFS`, `ScriptsRoot`, and `RegisterCLI`. The recommended addition is an explicit jsverbs surface instead of trying to overload `RegisterCLI` as a bag of arbitrary logic.

### Proposed Go shape

```go
type Definition struct {
    Name                    model.SiteName
    DatabaseFileName        string

    ScriptsFS               fs.FS
    ScriptsRoot             string

    VerbsFS                 fs.FS
    VerbsRoot               string
    VerbSharedSections      []*jsverbs.SectionSpec
    VerbRuntimeRegistrars   []gggengine.RuntimeModuleRegistrar

    SQLMigrationsFS         fs.FS
    SQLMigrationsRoot       string
    JSMigrationsFS          fs.FS
    JSMigrationsRoot        string

    HelpFS                  fs.FS
    HelpRoot                string
    RuntimeModuleRegistrars []gggengine.RuntimeModuleRegistrar
    RegisterCLI             func(root *cobra.Command) error
}
```

Notes:

- `Scripts*` remains for scheduler op scripts.
- `Verbs*` is new and optional.
- `VerbSharedSections` allows Go to register standard flags like `db`, `fixture`, or `output`.
- `VerbRuntimeRegistrars` allows the CLI verb runtime to expose extra modules without contaminating the op runtime.
- `RegisterCLI` can remain for future custom commands, but the main site-js runner should not depend on every site hand-writing Cobra wiring.

## Proposed CLI Shape

The most legible command tree is:

```text
scraper site js list-sites
scraper site js <site> list
scraper site js <site> <verb...>
```

Examples:

```bash
scraper site js list-sites
scraper site js hackernews list
scraper site js hackernews parse-frontpage --html-file fixtures/frontpage.html --base-url https://news.ycombinator.com/
scraper site js slashdot parse-frontpage --html-file fixtures/frontpage.html --base-url https://slashdot.org/
```

Why this shape is better than mounting everything at the root:

- it keeps site verbs grouped under the existing `site` namespace,
- it makes the site name explicit,
- it avoids polluting the root command tree with every per-site helper verb,
- it preserves room for future non-JS site commands.

## Proposed Go-side Architecture

### New package

Recommended package:

```text
pkg/sitejs/
  registry.go
  runtime.go
  command.go
```

Responsibilities:

- resolve a site definition,
- scan `VerbsFS` with `jsverbs.ScanFS(...)`,
- register host-owned shared sections,
- build a runtime with preconfigured modules,
- mount the generated commands under Cobra.

### High-level data flow

```text
site definition
    |
    v
resolve VerbsFS + VerbsRoot
    |
    v
jsverbs.ScanFS(...)
    |
    v
Add shared sections
    |
    v
attach scraper-specific runtime modules
    |
    v
registry.Commands()
    |
    v
mount under "scraper site js <site> ..."
```

### Pseudocode sketch

```go
func newSiteJSCommand(siteRegistry *siteregistry.Registry) *cobra.Command {
    root := &cobra.Command{
        Use:   "js",
        Short: "Run site-scoped JavaScript helper verbs",
    }

    root.AddCommand(&cobra.Command{
        Use: "list-sites",
        RunE: func(cmd *cobra.Command, args []string) error {
            for _, def := range siteRegistry.List() {
                if def.VerbsFS == nil || strings.TrimSpace(def.VerbsRoot) == "" {
                    continue
                }
                fmt.Fprintln(cmd.OutOrStdout(), def.Name)
            }
            return nil
        },
    })

    for _, def := range siteRegistry.List() {
        if def.VerbsFS == nil || strings.TrimSpace(def.VerbsRoot) == "" {
            continue
        }

        registry, err := buildSiteVerbRegistry(def)
        if err != nil {
            // fail early at startup
            return commandThatReports(err)
        }

        commands, err := registry.Commands()
        if err != nil {
            return commandThatReports(err)
        }

        siteCmd := &cobra.Command{Use: string(def.Name)}
        siteCmd.AddCommand(listVerbCommand(registry))
        cli.AddCommandsToRootCommand(siteCmd, commands, nil, ...)
        root.AddCommand(siteCmd)
    }

    return root
}
```

## Runtime Design For Site Verbs

The runtime for site verbs should not pretend to be the durable op runtime. It should be smaller and more explicit.

### What to reuse

Reuse these ideas from the current scraper runtime:

- site-scoped embedded FS,
- preconfigured `site-db`,
- preconfigured `scraper-db`,
- site-specific runtime module registrars,
- relative `require()` support.

Reuse these ideas from `jsverbs`:

- scanning and command discovery,
- Glazed schema generation,
- `bind: "context"`,
- shared sections,
- structured versus text output.

### What not to reuse directly

Do not inject the durable `ctx` object with:

- workflow,
- lease,
- dependency lookup,
- emit,
- artifact writes,
- record writes.

Those concepts make sense for workflow ops. They do not make sense as the default surface for CLI verbs.

### Recommended verb runtime modules

The CLI verb runtime should expose a small set of modules. Recommended first pass:

- `site-db`
- `scraper-db`
- `site-env`
- `fixtures`

#### `site-db`

Use the same preconfigured module pattern as the current runtime.

#### `scraper-db`

Optional in milestone one, but useful for reading workflow artifacts or stored results when debugging.

#### `site-env`

Recommended helper module:

```js
const env = require("site-env");

env.siteName();      // "hackernews"
env.rootDir();       // site verbs root
env.sitesDir();      // configured sites dir
env.dbPath();        // resolved site db path
env.now();           // RFC3339 string
```

This keeps host-provided path/config values out of ad hoc `context` object conventions when a module call is clearer.

#### `fixtures`

Recommended helper module:

```js
const fixtures = require("fixtures");

fixtures.readText("frontpage.html");
fixtures.readJSON("case-01.json");
```

That makes fixture-backed parsing commands extremely easy to write and removes repetitive path plumbing from every verb.

## Shared Section Recommendations

`jsverbs` shared sections are an important fit here. Register them from Go so the per-site verb files stay small and consistent.

Recommended shared sections:

### `site-db`

Fields:

- `sites-dir`
- `read-only`

Use when a verb needs DB access.

### `fixture-input`

Fields:

- `fixture`
- `html-file`
- `json-file`
- `base-url`

Use when a verb parses fixture content.

### `output`

Fields:

- `pretty`
- `limit`

Use when a verb prints diagnostics or slices of parsed rows.

## Example JavaScript Patterns

### Good verb wrapper around a shared helper

```js
__section__("input", {
  title: "Input",
  fields: {
    htmlFile: { type: "string", argument: true, help: "Path to HTML fixture" },
    baseURL: { type: "string", required: true }
  }
});

function parseFrontpage(input, meta) {
  const fs = require("fs");
  const parser = require("../lib/frontpage.js");
  const html = fs.readFileSync(input.htmlFile, "utf8");
  return parser.extractStories(html, input.baseURL);
}

__verb__("parseFrontpage", {
  command: "parse-frontpage",
  sections: ["input"],
  fields: {
    input: { bind: "input" },
    meta: { bind: "context" }
  }
});
```

### Bad pattern: directly reusing the durable op script as the verb

```js
// Avoid this.
module.exports = function (ctx) {
  const dep = ctx.dep(ctx.input.fetchedOpID);
  ...
}
```

This is wrong because the CLI command is now pretending to be a workflow op instead of being an operator tool.

## File Layout Recommendation

Recommended site layout:

```text
pkg/sites/hackernews/
  site.go
  migrations/
  fixtures/
  scripts/
    seed.js
    extract_frontpage.js
  verbs/
    parse_frontpage.js
    inspect_story_rows.js
  lib/
    frontpage.js
```

Alternative layout:

```text
pkg/sites/hackernews/
  scripts/
    ops/
    verbs/
    lib/
```

I recommend the first layout over burying verbs under `scripts/` because:

- it makes the runtime split obvious,
- it avoids accidental scanning of workflow op scripts,
- it is easier for a new intern to understand immediately.

## How This Fits The Current Exercise Sites

The Hacker News and Slashdot packages are good first adopters because they already have:

- embedded fixtures,
- shared parser helpers,
- working site DB schemas,
- stable extraction tests.

A first wave of verbs could be:

### Hacker News

- `parse-frontpage`
- `show-top-story-ids`
- `load-fixture-into-db`
- `list-stories`

### Slashdot

- `parse-frontpage`
- `show-source-links`
- `load-fixture-into-db`
- `list-stories`

These commands would let an operator validate three different layers independently:

1. pure parser output,
2. projection writing into the site DB,
3. DB contents after projection.

## Implementation Plan

### Phase 1: Add site-verb fields to the site registry

Files likely touched:

- `pkg/sites/registry/registry.go`
- `pkg/sites/defaults/defaults.go`
- site packages that want verbs

Implement:

- `VerbsFS`
- `VerbsRoot`
- optional shared-section/runtime registration fields for verbs

Validation:

- unit test that a site definition can register scripts and verbs independently.

### Phase 2: Build a site-verb registry loader

Files likely added:

- `pkg/sitejs/registry.go`

Implement:

- `buildSiteVerbRegistry(definition, options)`,
- `jsverbs.ScanFS(...)` against the site verbs tree,
- registration of scraper-owned shared sections,
- registration of scraper/site runtime modules.

Validation:

- package tests scanning a small embedded fixture FS.

### Phase 3: Build the CLI command tree

Files likely touched:

- `pkg/cmd/root.go`
- `pkg/cmd/site.go`
- new `pkg/cmd/site_js.go`

Implement:

- `scraper site js list-sites`
- `scraper site js <site> list`
- dynamic mounting of generated commands

Validation:

- root command test confirms built-in sites with verbs appear,
- `list` command prints discovered verb paths.

### Phase 4: Add scraper-specific runtime modules for site verbs

Files likely added:

- `pkg/sitejs/runtime.go`

Implement:

- preconfigured `site-db`,
- optional `scraper-db`,
- `site-env`,
- fixture helpers.

Validation:

- package tests for DB and fixture modules.

### Phase 5: Add verbs to Hacker News and Slashdot

Files likely added:

- `pkg/sites/hackernews/verbs/*.js`
- `pkg/sites/slashdot/verbs/*.js`

Implement:

- parser-focused verbs first,
- DB-inspection verbs second,
- keep all heavy logic in shared `lib/`.

Validation:

- fixture-backed CLI tests,
- help output checks,
- text-output and structured-output checks.

### Phase 6: Decide whether NEREVAL should adopt verbs

Do this after the exercise sites prove the pattern.

Candidate NEREVAL verbs:

- `parse-list-page`
- `parse-detail-page`
- `show-next-page-form`
- `project-detail-json`

## Testing Strategy

### Package-level tests

- scan an embedded FS with `ScanFS(...)`,
- ensure malformed metadata surfaces as a `ScanError`,
- ensure host shared sections register correctly.

### CLI tests

- `scraper site js list-sites`
- `scraper site js hackernews list`
- `scraper site js hackernews parse-frontpage --fixture frontpage.html ...`

### Runtime tests

- `site-db` is injected correctly,
- `fixtures.readText(...)` loads embedded fixtures,
- writer-style commands print plain text,
- structured verbs return rows.

### Integration tests

- compare CLI parser output with the existing workflow test fixtures,
- verify DB writes from CLI verbs produce the same rows as the scheduler-driven extractor path when they share the same helper functions.

## Design Decisions

### Decision 1: Keep scheduler op scripts and CLI verbs separate

Reason:

- different contracts,
- different ergonomics,
- less accidental coupling,
- easier onboarding.

### Decision 2: Share libraries, not entrypoints

Reason:

- parser logic should be reused,
- execution semantics should not be.

### Decision 3: Use `ScanFS(...)` instead of directory-only scanning

Reason:

- site packages already use embedded files,
- it preserves one source of truth,
- it avoids temporary-file hacks.

### Decision 4: Mount under `scraper site js`

Reason:

- natural operator discovery,
- no root command explosion,
- future room for site-specific non-JS commands.

## Alternatives Considered

### Alternative A: Directly scan `scripts/`

Rejected because:

- the current scripts are op entrypoints, not verbs,
- helper discovery becomes noisy,
- operators would have to fake workflow state.

### Alternative B: Extend `jsverbs` to understand the durable `ctx` contract

Rejected for milestone one because:

- it would blur two currently clean abstractions,
- it would make `jsverbs` scraper-specific,
- it would still produce poor CLI ergonomics for simple parser tasks.

### Alternative C: Skip jsverbs and hand-write Cobra commands per site

Rejected because:

- `go-go-goja` already provides the core plumbing,
- hand-written commands duplicate schema and runtime glue,
- it makes adding small site tools too expensive.

## Risks

### Risk: logic duplication between `scripts/` and `verbs/`

Mitigation:

- require both layers to call shared `lib/` helpers,
- review new site code for logic moving into entrypoints.

### Risk: too many tiny ad hoc verbs

Mitigation:

- reserve verbs for operator-facing tools and fixture debugging,
- keep naming disciplined,
- group shared CLI schema through sections.

### Risk: runtime-module drift between site verbs and op scripts

Mitigation:

- define one shared module-construction helper for DB injection and site env,
- keep the differences explicit in code and docs.

## Open Questions

- Should `scraper site js <site>` mount all verbs dynamically at process startup, or lazily build them when the site subcommand is entered?
- Should site verbs be able to write to the site DB by default, or should writes require an explicit `--write` flag?
- Should fixture access be via a module such as `require("fixtures")`, via shared sections, or both?
- Should the site registry keep `RegisterCLI` as a low-level escape hatch after the generic site-js runner exists, or should the new system cover almost all site CLI needs?

## References

### go-go-goja

- [go-go-goja/pkg/doc/08-jsverbs-example-overview.md](../../../../../../../go-go-goja/pkg/doc/08-jsverbs-example-overview.md)
- [go-go-goja/pkg/doc/09-jsverbs-example-fixture-format.md](../../../../../../../go-go-goja/pkg/doc/09-jsverbs-example-fixture-format.md)
- [go-go-goja/pkg/doc/10-jsverbs-example-developer-guide.md](../../../../../../../go-go-goja/pkg/doc/10-jsverbs-example-developer-guide.md)
- [go-go-goja/pkg/doc/11-jsverbs-example-reference.md](../../../../../../../go-go-goja/pkg/doc/11-jsverbs-example-reference.md)
- [go-go-goja/cmd/jsverbs-example/main.go](../../../../../../../go-go-goja/cmd/jsverbs-example/main.go)
- [go-go-goja/pkg/jsverbs/model.go](../../../../../../../go-go-goja/pkg/jsverbs/model.go)
- [go-go-goja/pkg/jsverbs/scan.go](../../../../../../../go-go-goja/pkg/jsverbs/scan.go)
- [go-go-goja/pkg/jsverbs/binding.go](../../../../../../../go-go-goja/pkg/jsverbs/binding.go)
- [go-go-goja/pkg/jsverbs/command.go](../../../../../../../go-go-goja/pkg/jsverbs/command.go)
- [go-go-goja/pkg/jsverbs/runtime.go](../../../../../../../go-go-goja/pkg/jsverbs/runtime.go)

### scraper

- [pkg/cmd/root.go](../../../../../../pkg/cmd/root.go)
- [pkg/cmd/site.go](../../../../../../pkg/cmd/site.go)
- [pkg/sites/registry/registry.go](../../../../../../pkg/sites/registry/registry.go)
- [pkg/engine/runner/js.go](../../../../../../pkg/engine/runner/js.go)
- [pkg/js/runtime/executor.go](../../../../../../pkg/js/runtime/executor.go)
- [pkg/js/runtime/databases.go](../../../../../../pkg/js/runtime/databases.go)
- [pkg/sites/hackernews/scripts/seed.js](../../../../../../pkg/sites/hackernews/scripts/seed.js)
- [pkg/sites/hackernews/scripts/extract_frontpage.js](../../../../../../pkg/sites/hackernews/scripts/extract_frontpage.js)
- [pkg/sites/slashdot/scripts/extract_frontpage.js](../../../../../../pkg/sites/slashdot/scripts/extract_frontpage.js)
