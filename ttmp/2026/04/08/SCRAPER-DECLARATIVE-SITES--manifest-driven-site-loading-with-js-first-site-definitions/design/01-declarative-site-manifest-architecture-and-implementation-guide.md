---
Title: Declarative site manifest architecture and implementation guide
Ticket: SCRAPER-DECLARATIVE-SITES
Status: active
Topics:
    - scraper
    - architecture
    - backend
    - javascript
    - api
    - onboarding
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/sites/registry/registry.go
      Note: Existing Go contract that the manifest loader would target.
    - Path: pkg/sites/defaults/defaults.go
      Note: Current built-in bootstrap path to be complemented or partially replaced.
    - Path: pkg/sites/hackernews/site.go
      Note: Example current site definition that is mostly metadata plus embedded FS roots.
    - Path: pkg/sites/jsdemo/site.go
      Note: Example of a site that mainly points at JS assets and standard modules.
    - Path: pkg/services/catalog/service.go
      Note: Existing site metadata API surface that should continue to work after manifest loading is added.
ExternalSources: []
Summary: Intern-facing design for moving scraper toward manifest-driven site registration while preserving Go-native extension points for advanced sites.
LastUpdated: 2026-04-08T09:20:00-04:00
WhatFor: Explain how to add declarative sites safely and incrementally.
WhenToUse: Use when implementing manifest loading, reviewing the site contract, or deciding how much Go should remain in site definitions.
---

# Declarative site manifest architecture and implementation guide

## 1. Problem statement

Today scraper requires a Go site definition for every built-in site. In practice, that Go definition often contains only metadata and pointers to embedded files. The actual site behavior is usually elsewhere:

- JavaScript op scripts
- JavaScript submit verbs
- SQL migrations
- sometimes JS migrations
- optional help content

That means a new site author still needs to touch Go even when the site’s logic is already almost entirely file-driven.

This is a poor fit for the way scraper has evolved. The system is increasingly JS-first. Verbs are JS. Execution scripts are JS. The operator/debugging story is centered around scripts, artifacts, and runtime events. Requiring a Go package mostly to say “my scripts live here” is friction without much value for ordinary sites.

The core design question is:

Can scraper load most sites from a declarative manifest plus a site directory of JS and migration files, while still allowing Go-native sites when a site truly needs custom native code?

The answer is yes, with an important boundary:

- declarative sites should cover the common case
- Go-defined sites should remain available for advanced native integrations

## 2. Current system architecture

### 2.1 The current registration seam

The current site contract is [registry.Definition](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/registry/registry.go). It includes:

- `Name`
- `DatabaseFileName`
- `ScriptsFS` and `ScriptsRoot`
- `VerbsFS` and `VerbsRoot`
- `Modules`
- SQL and JS migration FS/root pairs
- help FS/root
- `QueuePolicies`
- `RuntimeModuleRegistrars`
- optional `RegisterCLI`

This is then stored in a `Registry`, and built-in sites are currently assembled in [defaults.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/defaults/defaults.go).

### 2.2 What existing sites really do

If you inspect [pkg/sites/hackernews/site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/site.go), most of the file is:

- embed the site filesystem
- assign roots
- declare a queue policy

If you inspect [pkg/sites/jsdemo/site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site.go), the shape is similar:

- embed the site filesystem
- assign roots
- declare standard modules

This is important. These files are not implementing business logic. They are packaging metadata.

### 2.3 Catalog/API impact

The catalog layer in [pkg/services/catalog/service.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/services/catalog/service.go) exposes site information to the API and frontend:

- site summary
- site detail
- verbs
- scripts
- queue policies

That means any declarative-site work must preserve the shape of `registry.Definition`, or at least preserve what the catalog service expects to read from it.

## 3. What should become declarative

The following parts are good manifest candidates because they are metadata, not code:

- site name
- site DB filename
- roots for scripts, verbs, SQL migrations, JS migrations, and help docs
- queue policies
- list of standard native modules to expose
- whether the site is embedded, filesystem-loaded, or both
- optional display metadata such as title or description

The following parts should remain outside the manifest, at least initially:

- arbitrary Go runtime registrars
- arbitrary Go module constructors
- hand-written Cobra command logic
- custom code that mutates the registry in unusual ways

This split gives us a clean rule:

- manifests describe bounded configuration
- Go remains the escape hatch for unbounded behavior

## 4. Recommended architecture

### 4.1 High-level shape

Add a new package, likely:

```text
pkg/sites/manifest/
  model.go
  validate.go
  loader.go
```

The loader reads a site manifest and returns a normal [registry.Definition](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/registry/registry.go). That means the rest of scraper does not need to care whether a site came from:

- a Go package
- a manifest on disk
- an embedded manifest

Everything downstream still consumes `registry.Definition`.

### 4.2 Recommended manifest file

Use one obvious file name first:

```text
site.yaml
```

YAML is a reasonable first choice because:

- it is readable for humans
- queue policies and nested config are easy to express
- it is already a familiar format for operational metadata

JSON could also work, but YAML is the better authoring format here.

### 4.3 Proposed manifest schema

Pseudocode schema:

```yaml
name: hackernews
databaseFileName: hackernews.db

scripts:
  root: scripts

verbs:
  root: verbs

sqlMigrations:
  root: migrations

jsMigrations:
  root: js-migrations

help:
  root: help

modules:
  - defaultRegistryModules

queuePolicies:
  - queue: site:hackernews:http
    maxInFlight: 1
    rateLimit:
      kind: token-bucket
      ratePerSecond: 1
      burst: 1
```

Recommended initial rules:

- `name` is required
- `databaseFileName` is required
- roots are optional, but if present they must be relative and must exist
- `modules` must be selected from a bounded allowlist
- queue policy keys must parse as known queue keys
- rate-limit kinds must be validated strictly

### 4.4 Why not support arbitrary Go registrars in the manifest

Because that would not actually be declarative.

If the manifest can say “call this arbitrary Go registrar,” then the manifest becomes a thin shell around Go code again, and we lose the safety and portability benefits.

The manifest should only be allowed to request known, standard capabilities.

For example:

- `defaultRegistryModules`
- maybe later `httpHelpers`
- maybe later `htmlParsing`

But not:

- `call package X function Y`

## 5. Loader design

### 5.1 Loader responsibilities

The loader should do four things:

1. read and decode the manifest
2. validate it
3. bind it to an `fs.FS` root
4. return a populated `registry.Definition`

Pseudocode:

```text
func LoadDefinition(siteFS fs.FS, manifestPath string) (registry.Definition, error) {
  manifest := decode(siteFS, manifestPath)
  validate(manifest)

  return registry.Definition{
    Name: manifest.Name,
    DatabaseFileName: manifest.DatabaseFileName,
    ScriptsFS: siteFS,
    ScriptsRoot: manifest.Scripts.Root,
    VerbsFS: siteFS,
    VerbsRoot: manifest.Verbs.Root,
    SQLMigrationsFS: siteFS,
    SQLMigrationsRoot: manifest.SQLMigrations.Root,
    JSMigrationsFS: siteFS,
    JSMigrationsRoot: manifest.JSMigrations.Root,
    HelpFS: siteFS,
    HelpRoot: manifest.Help.Root,
    Modules: resolveStandardModules(manifest.Modules),
    QueuePolicies: convertQueuePolicies(manifest.QueuePolicies),
  }, nil
}
```

### 5.2 Validation rules

Important validation rules for a first version:

- site name must be non-empty and stable
- DB filename must not escape the sites dir
- roots must be relative, not absolute
- roots must not contain `..`
- duplicate queue keys are rejected
- unknown module names are rejected
- missing required directories are rejected

This is worth being strict about. A manifest loader should fail early and clearly.

## 6. Embedded versus filesystem-loaded sites

This system should support both.

### 6.1 Embedded built-ins

Built-in sites can still be compiled into the binary using `embed.FS`, but instead of a handwritten Go `Definition()` function, the built-in package can do something like:

```go
//go:embed site.yaml scripts/*.js verbs/*.js migrations/*.sql
var siteFS embed.FS

func Register(r *registry.Registry) error {
    def, err := manifest.LoadDefinition(siteFS, "site.yaml")
    if err != nil {
        return err
    }
    return r.Register(def)
}
```

This is a major improvement because the Go package becomes thin packaging code rather than a metadata authoring surface.

### 6.2 External site directories

Later, scraper could also load sites directly from directories on disk:

```text
sites/
  hackernews/
    site.yaml
    scripts/
    verbs/
    migrations/
```

That opens the door to:

- site packs outside the main binary
- faster site author iteration
- easier onboarding for contributors who mostly write JS

## 7. Module strategy

This is the most important boundary in the design.

The manifest should not choose arbitrary Go modules. It should choose from a bounded catalog of standard modules.

Example internal map:

```text
module ID -> Go resolver

defaultRegistryModules -> gggengine.DefaultRegistryModules()
```

This is how you keep declarative sites safe and predictable.

If a site needs a one-off native Go module, that site should remain a Go-defined site until the capability becomes standard enough to promote into the declarative allowlist.

## 8. Queue policy handling

Queue policies already exist in the current contract and should map naturally from a manifest.

Current engine behavior:

- queue policy is resolved by site + queue
- default policy is used when no explicit policy exists

That means declarative sites only need to supply a clean map:

```text
manifest queue policies -> registry.Definition.QueuePolicies
```

This is a good fit for declarative configuration because queue policies are pure metadata.

## 9. API implications

The good news is that the current catalog layer can stay mostly unchanged.

Why:

- [catalog.Service](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/services/catalog/service.go) reads from `registry.Definition`
- if the manifest loader produces normal `registry.Definition` values, the API layer does not need to care where the site came from

Possible future enhancement:

- add an optional `origin` field to the catalog view:
  - `go`
  - `manifest-embedded`
  - `manifest-filesystem`

That would help operators and site authors understand what kind of site they are looking at, but it should be optional.

## 10. Migration strategy

Do not migrate every site at once.

Recommended rollout:

1. add manifest model and loader
2. add tests for manifest validation and loading
3. convert one simple site first, ideally `js-demo`
4. keep the existing Go registration path working
5. convert more sites only after the loader is proven

This is the lowest-risk migration because it avoids a flag day.

### 10.1 Suggested first proof point

Best first migration candidate:

- [jsdemo/site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site.go)

Why:

- it already uses mostly standard modules
- it is simple
- it is already a good smoke-test site

`hackernews` is a good second candidate because it adds queue policy coverage.

## 11. Testing plan

### 11.1 Manifest tests

- valid manifest decodes successfully
- invalid YAML fails clearly
- unknown module ID fails clearly
- missing required fields fail
- invalid relative paths fail
- duplicate queue policies fail

### 11.2 Loader tests

- manifest roots map correctly into `registry.Definition`
- queue policies are normalized correctly
- embedded `fs.FS` loading works
- filesystem directory loading works if supported in the first version

### 11.3 Integration tests

- register a manifest-driven `js-demo` site
- list it through the catalog service
- load verbs and scripts successfully
- run an end-to-end workflow

## 12. Risks and mitigations

### Risk: manifest schema grows into a second programming language

Mitigation:

- keep the schema narrow
- do not add arbitrary expressions or callbacks
- promote capabilities into standard module IDs deliberately

### Risk: embedded and filesystem sites drift apart

Mitigation:

- use the same loader for both
- differ only in the `fs.FS` backing store

### Risk: Go-native sites become second-class

Mitigation:

- keep `registry.Register(def Definition)` exactly as a valid path
- treat Go-native registration as the advanced extension path, not as deprecated baggage

## 13. Recommended implementation order

1. Add manifest schema structs and validation.
2. Add a manifest loader that returns `registry.Definition`.
3. Add tests for decoding, validation, and definition construction.
4. Add a small built-in wrapper that loads one embedded manifest-driven site.
5. Convert `js-demo`.
6. Validate catalog, verbs, scripts, migrations, and workflow execution still work.
7. Convert one queue-policy site such as Hacker News.
8. Decide whether to add external directory discovery as a second ticket or in the same implementation pass.

## 14. Recommendation

Yes, scraper should move toward declarative sites.

But the right target is not “no Go ever.” The right target is:

- manifests for ordinary sites
- JavaScript for ordinary site behavior
- Go only when a site truly needs native extension hooks

That gives site authors a much simpler authoring model while preserving the power that the current Go registration path provides for advanced cases.
