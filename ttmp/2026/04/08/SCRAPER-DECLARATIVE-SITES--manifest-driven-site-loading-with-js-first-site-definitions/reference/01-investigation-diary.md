---
Title: Investigation diary
Ticket: SCRAPER-DECLARATIVE-SITES
Status: active
Topics:
    - scraper
    - architecture
    - backend
    - javascript
    - onboarding
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Records the reasoning behind the declarative-sites design ticket.
LastUpdated: 2026-04-08T09:20:00-04:00
WhatFor: Preserve the current-state observations and design rationale for manifest-driven site loading.
WhenToUse: Use when implementing or reviewing the declarative-sites design.
---

# Investigation diary

## Prompt summary

The request was to create a new ticket for declarative sites and write a detailed intern-facing analysis, design, and implementation guide. The underlying question was whether scraper still needs Go-defined sites, or whether declarative metadata plus JavaScript could carry most site definitions.

## Current-state observations

The current site seam is clearly Go-defined:

- [pkg/sites/registry/registry.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/registry/registry.go) defines the registration contract.
- [pkg/sites/defaults/defaults.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/defaults/defaults.go) explicitly registers built-in sites in Go.
- Site packages like [pkg/sites/hackernews/site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/site.go) and [pkg/sites/jsdemo/site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site.go) mostly exist to point at embedded filesystem roots and declare metadata such as queue policies.

At the same time, most actual site behavior is already file-oriented:

- scripts are JavaScript files
- verbs are JavaScript files
- migrations are SQL and sometimes JS files
- help/docs can live in files

That means the current model is only partially Go-native. Most runtime logic is already declarative-ish plus JS. The main Go-only part is the registration envelope.

## Working conclusion

The best direction is not “eliminate Go entirely.” The best direction is:

- ordinary sites should be manifest-driven and JS-first
- advanced sites can still opt into Go-native definitions when they need custom native modules or special runtime hooks

This preserves flexibility while reducing friction for the common case.

## Design emphasis

The guide focuses on:

- what belongs in a manifest
- what should stay out of the manifest
- how a manifest loader should build `registry.Definition`
- how embedded built-in sites and filesystem-loaded sites can coexist
- how to migrate safely without breaking current sites all at once

## Validation and publishing

After writing the ticket docs, I validated the workspace with:

```bash
docmgr doctor --ticket SCRAPER-DECLARATIVE-SITES --stale-after 30
```

The doctor report passed without findings beyond the success marker.

I then uploaded the full bundle to reMarkable with:

```bash
remarquee upload bundle ttmp/2026/04/08/SCRAPER-DECLARATIVE-SITES--manifest-driven-site-loading-with-js-first-site-definitions \
  --remote-dir /ai/2026/04/08/SCRAPER-DECLARATIVE-SITES \
  --name "Scraper declarative sites" \
  --non-interactive
```

I also verified the remote directory with:

```bash
remarquee cloud ls /ai/2026/04/08/SCRAPER-DECLARATIVE-SITES
```

The uploaded document appeared as `Scraper declarative sites`.

## 2026-04-08 implementation kickoff

Before touching code, I refined the high-level task list into a real execution backlog. The original ticket tasks were still design-shaped, but not yet granular enough for incremental implementation and commits. I inspected:

- [pkg/sites/registry/registry.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/registry/registry.go)
- [pkg/sites/defaults/defaults.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/defaults/defaults.go)
- [pkg/sites/jsdemo/site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site.go)
- [pkg/sites/hackernews/site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/site.go)
- [pkg/services/catalog/service.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/services/catalog/service.go)

That review confirmed a low-risk implementation direction:

- keep `registry.Definition` as the stable internal contract
- add a manifest package that maps declarative metadata into `registry.Definition`
- migrate built-in sites one at a time rather than changing registry consumers first

I then expanded the task list so the implementation could be done in slices:

- manifest structs and validation
- manifest loader and module mapping
- registry helper integration
- `js-demo` migration as the simple proof point
- `hackernews` migration as the queue-policy proof point
- mixed-registry validation

That gives a clean commit structure and keeps the diary tied to concrete slices rather than one large undifferentiated refactor.

## 2026-04-08 manifest package slice

I implemented the first code slice in a new package:

- [pkg/sites/manifest/manifest.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/manifest/manifest.go)
- [pkg/sites/manifest/modules.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/manifest/modules.go)
- [pkg/sites/manifest/validation.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/manifest/validation.go)
- [pkg/sites/manifest/manifest_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/manifest/manifest_test.go)

The point of this slice was to establish a strict declarative schema before touching registry wiring. I intentionally kept the first version small:

- one manifest type representing site metadata
- one bounded module identifier, `default-registry`
- validation for relative roots, file-name-only DB names, duplicate queue policies, and token-bucket rate limits

I did not add YAML decoding or `registry.Definition` mapping yet. That separation matters because it lets the validation contract settle before loader code starts depending on it.

I validated this slice with:

```bash
gofmt -w pkg/sites/manifest/*.go
go test ./pkg/sites/manifest -count=1
```

The tests passed cleanly on the first run.

## 2026-04-08 loader slice

The next slice added the actual load path:

- [pkg/sites/manifest/loader.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/manifest/loader.go)
- [pkg/sites/manifest/loader_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/manifest/loader_test.go)

I kept this logic inside `pkg/sites/manifest` instead of `pkg/sites/registry`. That choice avoids a package cycle and preserves a clean layering:

- `registry` stays a dumb container for `Definition`
- `manifest` becomes the translation layer from declarative YAML into `Definition`

This slice now supports three steps:

1. read `site.yaml` from any `fs.FS`
2. decode it with `KnownFields(true)` so typos fail fast
3. validate and map it into a normal `registry.Definition`

I also added `RegisterFS(...)` in the manifest package so call sites can register embedded manifest-backed sites without each package having to repeat the load-and-register sequence.

Validation for this slice was:

```bash
gofmt -w pkg/sites/manifest/*.go
go test ./pkg/sites/manifest -count=1
```

The loader tests cover:

- manifest-to-definition mapping
- queue-policy normalization
- strict YAML decoding for unknown keys
- registration through a real `registry.Registry`

## 2026-04-08 js-demo migration slice

The first real site migration was [pkg/sites/jsdemo/site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site.go). I added:

- [pkg/sites/jsdemo/site.yaml](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site.yaml)
- a manifest-backed `Definition()` implementation that loads once from the embedded filesystem
- a targeted test in [pkg/sites/jsdemo/site_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site_test.go)

I intentionally kept the external API unchanged:

- `Definition()` still returns `registry.Definition`
- `Register(...)` still delegates to `registry.Register(...)`

That preserved the CLI test seam in [pkg/cmd/site_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/site_test.go), where the test fetches `Definition()`, overrides queue policies, and registers the modified definition manually.

Because the manifest is embedded into the binary, load failures here would be programming errors, not operator/runtime errors. For that reason I used a one-time cached load behind `sync.Once` and let `Definition()` panic if the embedded manifest is invalid. That keeps the call sites simple and fails fast during development.

Validation for this slice was:

```bash
gofmt -w pkg/sites/jsdemo/site.go pkg/sites/jsdemo/site_test.go
go test ./pkg/sites/jsdemo ./pkg/cmd -run 'TestDefinitionLoadsEmbeddedManifest|TestJSDemo|TestJSDemoSubmitThenWorkerRunWithQueueRateLimit' -count=1
```

Those tests passed cleanly.

## 2026-04-08 hackernews and registry-follow-on slice

The second site migration was [pkg/sites/hackernews/site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/site.go). I added:

- [pkg/sites/hackernews/site.yaml](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/site.yaml)
- a manifest-backed `Definition()` implementation
- a manifest-load sanity test plus the existing queue-policy assertion in [pkg/sites/hackernews/site_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/site_test.go)

This slice mattered because Hacker News is the first built-in site with an actual declarative queue policy. Migrating it proved that manifest-backed sites can carry operational metadata, not just roots and module lists.

I also added a mixed-registry test in [pkg/sites/manifest/loader_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/manifest/loader_test.go). That test registers:

- one traditional Go `registry.Definition`
- one manifest-backed site via `RegisterFS(...)`

and verifies that both coexist in the same registry. That closed the main architecture risk in the design doc.

At this point I made two explicit implementation decisions:

- no separate `registry.RegisterDefinition(...)` helper is necessary, because `registry.Register(...)` is already the plain-definition path
- [pkg/sites/defaults/defaults.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/defaults/defaults.go) should stay as explicit top-level registration for now; the individual site packages can become manifest-backed internally without forcing a registry-bootstrap rewrite

Validation for this slice was:

```bash
gofmt -w pkg/sites/hackernews/site.go pkg/sites/hackernews/site_test.go pkg/sites/manifest/loader_test.go
go test ./pkg/sites/manifest ./pkg/sites/hackernews ./pkg/sites/... -count=1
```

That test run passed cleanly and confirmed that the still-Go-defined sites (`slashdot`, `nereval`) were unaffected.

## 2026-04-08 final validation and publishing pass

After the code slices were in place, I ran the full repository validation:

```bash
go test ./... -count=1
docmgr doctor --ticket SCRAPER-DECLARATIVE-SITES --stale-after 30
```

Both passed.

I then refreshed the reMarkable bundle. The first upload attempt was skipped because the document already existed:

```bash
remarquee upload bundle ttmp/2026/04/08/SCRAPER-DECLARATIVE-SITES--manifest-driven-site-loading-with-js-first-site-definitions \
  --remote-dir /ai/2026/04/08/SCRAPER-DECLARATIVE-SITES \
  --name "Scraper declarative sites" \
  --non-interactive
```

That returned a skip message, so I repeated the upload with `--force`:

```bash
remarquee upload bundle ttmp/2026/04/08/SCRAPER-DECLARATIVE-SITES--manifest-driven-site-loading-with-js-first-site-definitions \
  --remote-dir /ai/2026/04/08/SCRAPER-DECLARATIVE-SITES \
  --name "Scraper declarative sites" \
  --non-interactive \
  --force
```

That upload succeeded.

At this point the backend implementation portion of the ticket is complete. The remaining unchecked items are product follow-ons:

- expose manifest-origin metadata in the catalog API if we decide it is useful
- decide whether the frontend should display declarative vs Go-native provenance
- decide whether to add a “create a site without Go” onboarding tutorial

## 2026-04-08 follow-on approval

After the initial backend rollout, the next request was to do all three remaining follow-ons:

- expose manifest-origin metadata in the catalog API
- show declarative vs Go-native provenance in the frontend
- add a no-Go site authoring tutorial

I converted those into a second, more detailed task block in the ticket before touching code again. The intended execution order is:

1. backend provenance fields and catalog API responses
2. frontend provenance display
3. help/tutorial authoring

That order keeps the data model authoritative before the UI renders it.

## 2026-04-08 catalog provenance slice

The first follow-on slice extended the backend data model so provenance can travel through the catalog API. The touched files were:

- [pkg/sites/registry/registry.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/registry/registry.go)
- [pkg/sites/manifest/loader.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/manifest/loader.go)
- [pkg/services/catalog/service.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/services/catalog/service.go)
- [pkg/api/server/server_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/api/server/server_test.go)

I modeled provenance directly on `registry.Definition` with:

- `Origin`
- `ManifestPath`

The important design choice was to give `registry.Register(...)` a safe default. If a caller registers a definition without setting an origin, the registry now records it as `go`. That keeps older Go-defined sites backward compatible without forcing every existing site package to be edited immediately.

Manifest-backed sites set:

- `Origin = "manifest"`
- `ManifestPath = "site.yaml"` (or whatever manifest path was loaded)

The catalog service then exposes that metadata through both site summaries and site detail responses.

I tightened the HTTP tests so they assert:

- `GET /api/v1/sites` includes manifest provenance for `js-demo`
- `GET /api/v1/sites/js-demo/detail` includes `originKind` and `manifestPath`

Validation for this slice was:

```bash
gofmt -w pkg/sites/registry/registry.go pkg/sites/manifest/loader.go pkg/sites/manifest/loader_test.go pkg/services/catalog/service.go pkg/api/server/server_test.go
go test ./pkg/sites/manifest ./pkg/api/server -count=1
```

## 2026-04-08 frontend provenance slice

With the catalog payload updated, the frontend changes were straightforward. I updated:

- [web/src/api/types.ts](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/api/types.ts)
- [web/src/components/sites/SiteOriginChip.tsx](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/sites/SiteOriginChip.tsx)
- [web/src/components/sites/SiteCard.tsx](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/sites/SiteCard.tsx)
- [web/src/pages/SiteDetailPage.tsx](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/pages/SiteDetailPage.tsx)
- [web/src/stories/__fixtures__/factories.ts](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/stories/__fixtures__/factories.ts)

The UI decision was intentionally conservative:

- use one shared `SiteOriginChip`
- show it on both the site cards and the site detail header
- show the manifest path only on the detail page and card metadata, not everywhere

That gives operators immediate context without turning provenance into a noisy first-class workflow dimension.

I ran:

```bash
go test ./pkg/api/server -count=1
cd web && npm run build
```

The Go test passed. The frontend build still fails, but the failures are pre-existing unrelated TypeScript issues in Storybook and other UI files, not in the provenance slice. The errors include unused imports, broken stories, and older type mismatches in files such as `src/components/common/AlertBanner.stories.tsx`, `src/components/results/*`, and `src/test-utils/mockRuntimeEvents.ts`.

## 2026-04-08 no-Go tutorial slice

The last requested follow-on was documentation, but it needed to be real product documentation rather than just ticket notes. I added:

- [pkg/doc/tutorials/scraper-adding-a-declarative-site.md](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc/tutorials/scraper-adding-a-declarative-site.md)

and updated:

- [pkg/doc/tutorials/scraper-adding-a-site.md](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/doc/tutorials/scraper-adding-a-site.md)

The new tutorial explains the current preferred path for sites that do not need native Go extensions:

- create `site.yaml`
- keep behavior in JS
- use a tiny wrapper for embedded built-in sites
- keep registration explicit in `pkg/sites/defaults/defaults.go` for now
- validate through the existing scheduler/worker path

I validated both help pages through the actual embedded help system with:

```bash
go run ./cmd/scraper help scraper-adding-a-declarative-site
go run ./cmd/scraper help scraper-adding-a-site
```

Both rendered correctly.

## 2026-04-08 sites extraction: move YAML manifests out of Go packages

The next step was to physically move the four declarative sites (hackernews, jsdemo, slashdot, nereval) out of `pkg/sites/` into a top-level `sites/` directory. These packages no longer contain meaningful Go code — just `site.yaml` manifests and JavaScript/SQL files.

### What happened

1. Copied `pkg/sites/{hackernews,jsdemo,nereval,slashdot}` to `scraper/sites/` and deleted the Go site packages from `pkg/sites/`.
2. Updated `pkg/sites/defaults/defaults.go` to load sites from a `sites/` directory on disk using `sitemanifest.RegisterDir` instead of Go `//go:embed`.
3. Updated tests in `defaults_test.go`, `server_test.go`, and `service_test.go` to use explicit paths.

### What went wrong

This turned into a rabbit hole of CWD-relative path bugs:

- `defaults.NewRegistry()` looks for `sites/` relative to CWD. When `go test` runs, CWD is the package directory (e.g., `pkg/api/server/`), NOT the repo root.
- We spent multiple turns adjusting `filepath.Abs("../..")` vs `filepath.Abs("../../..")` etc., trying to get the right number of `..` levels for each test package.
- Moved `sites/` from `js-scraper/sites/` to `js-scraper/scraper/sites/` (it was originally placed one level above the repo root by mistake).
- The `TestServerSubmitThenWorkerAndInspectWorkflow` test used `sitesDir := t.TempDir()` for the SQLite storage dir but the **worker** got an empty site registry because `NewRootCommand()` → `defaults.NewRegistry()` → CWD lookup failed.

### Root cause diagnosis

The fundamental problem is **three overlapping ways to find manifests**:

1. `defaults.NewRegistry()` — CWD-dependent auto-discovery (`sites/` relative to CWD)
2. `--sites-manifest-dir` — explicit per-subcommand flag on `worker run` and `api serve`
3. Implicit embedded sites (now removed)

These overlap and conflict. The CWD-dependent path is the worst because it behaves differently in production vs tests vs development.

### The fix plan

Kill `defaults.NewRegistry()` CWD magic. Make manifest dir always explicit:

1. Remove `defaults.NewRegistry()` and `NewRegistryWithSitesDir()`
2. Add `--sites-manifest-dir` as a **persistent** root-level flag (not per-subcommand)
3. `NewRootCommand()` reads the persistent flag and loads from there
4. Tests pass explicit paths — no CWD gymnastics

This leaves two clean concepts:
- **Manifest dir** (site definitions) — always explicit
- **Sites dir** (SQLite databases) — already explicit via `--sites-dir`

### What we actually did

During the fix we discovered that the Go site packages (`pkg/sites/{hackernews,jsdemo,...}`) still need to exist because:
- `//go:embed` embeds scripts/verbs/migrations/fixtures into the binary
- Tests use `hackernews.Definition()`, `hackernews.ReadFixture()`, `jsdemo.Definition()`, etc.

So the extraction into a bare `sites/` directory was wrong — built-in sites must stay as Go packages for embedding. The `--sites-manifest-dir` flag is correctly for **external** sites only.

The final fix was:
1. Restored Go site packages from git
2. `defaults.NewRegistry()` explicitly registers built-in sites (no CWD dependency)
3. `defaults.LoadExternalSites(r, dir)` loads additional external sites
4. `--sites-manifest-dir` is a persistent root flag, read by `LoadSitesFromFlag()` helper
5. `NewRootCommandWithRegistry()` lets tests inject a pre-built registry
6. Removed `NewRegistryWithSitesDir()`, `Register()`, `DefaultSitesManifestPath()`
7. All `go test ./... -count=1` passing
