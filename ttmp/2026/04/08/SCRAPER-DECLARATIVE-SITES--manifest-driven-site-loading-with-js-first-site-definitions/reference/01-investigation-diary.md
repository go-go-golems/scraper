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

### The result

All 4 sites are now pure filesystem manifests in `sites/` at the repo root:
- `sites/hackernews/` — site.yaml + JS scripts + SQL migrations + HTML fixtures
- `sites/jsdemo/` — site.yaml + JS scripts + SQL migrations
- `sites/nereval/` — site.yaml + JS scripts + SQL migrations + HTML fixtures
- `sites/slashdot/` — site.yaml + JS scripts + SQL migrations + HTML fixtures

No Go site packages remain. The binary ships with zero embedded sites. All sites are loaded at runtime from `sites/` via `NewRootCommand(version, manifestDirs...)`.

New helpers:
- `pkg/testfixtures/helpers.go` — `SitesDir(t)` and `ReadFixture(t, site, name)` using `runtime.Caller(0)` for reliable path resolution
- `defaults.NewRegistryFromDirs(dirs ...string)` — loads sites from one or more manifest directories

The `--sites-manifest-dir` persistent flag on the root command still allows loading additional sites at runtime (for production use where `main()` calls `NewRootCommand()` with no dirs).

## Step 1: Design bootstrap config and early site command loading

After making all sites external, one architectural gap remained: the real `main()` path still called `NewRootCommand(version)` with no manifest dirs. That meant the registry was empty at construction time, but scraper's dynamic site verbs are built when `newSiteCommand(...)` iterates the loaded registry. The user explicitly wanted to support multiple site directories from flags, env, and config, so I stopped and wrote down the bootstrap pattern before changing code.

The key insight came from sqleton: repository-backed commands are not loaded inside `RunE`; they are discovered during an app-owned bootstrap phase before Cobra execution begins. Scraper now has the same shape. Site manifests are not ordinary runtime data — they are command-definition sources — so they must be resolved before the command tree exists.

### Prompt Context

**User prompt (verbatim):** "i do want to put it at a variety of sites directories based on need, so I think we might need to parse the flag / env / config before building the command tree. We have similar setups in ~/code/wesen/corporate-headquarters/sqleton I think with a document on how to do it (or maybe \"/home/manuel/code/wesen/obsidian-vault/Projects/2026/04/02/PROJ - Sqleton SQL Command Cleanup - Technical Project Report.md\") 

Do you find info on how we can load the commands as verbs after parsing the fconfig stuff"

**Assistant interpretation:** The user wants scraper to adopt a sqleton-style bootstrap flow where app config, environment variables, and selected CLI flags are resolved before dynamic site verbs are constructed.

**Inferred user intent:** Make site-directory selection flexible and reliable in real CLI usage, without depending on CWD tricks or late-loading that occurs after the Cobra command tree is already frozen.

### What I did
- Inspected sqleton's startup/config flow in:
  - `/home/manuel/code/wesen/corporate-headquarters/sqleton/cmd/sqleton/main.go`
  - `/home/manuel/code/wesen/corporate-headquarters/sqleton/cmd/sqleton/config.go`
  - `/home/manuel/code/wesen/corporate-headquarters/sqleton/ttmp/2026/04/02/SQLETON-02-VIPER-APP-CONFIG-CLEANUP--remove-viper-and-separate-sqleton-app-config-from-command-config/design/01-sqleton-viper-removal-and-app-config-cleanup-design.md`
  - `/home/manuel/code/wesen/obsidian-vault/Projects/2026/04/02/PROJ - Sqleton SQL Command Cleanup - Technical Project Report.md`
- Wrote a dedicated implementation/design doc:
  - `/home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/04/08/SCRAPER-DECLARATIVE-SITES--manifest-driven-site-loading-with-js-first-site-definitions/design/02-bootstrap-config-and-early-site-command-loading.md`
- Expanded the ticket task list with a concrete Phase 3 implementation plan.

### Why
- Scraper now depends on manifest directories to build dynamic `site <name> run <verb>` commands.
- A normal Cobra runtime flag is too late because command discovery happens before `Execute()`.
- The sqleton pattern provides a proven architecture for separating app-owned bootstrap config from runtime command parsing.

### What worked
- The sqleton references were directly relevant.
- The architecture maps closely: sqleton repository directories and scraper site-manifest directories are both command-definition sources.
- The right model became clear quickly: bootstrap config first, command tree second, Cobra execution third.

### What didn't work
- The previously-added late `LoadSitesFromFlag(...)` model was sufficient only for execution-time consumers. It is not sufficient for site verb discovery because those commands must exist before Cobra parsing begins.
- No code failure occurred in this planning step, but the architectural mismatch was already visible in `cmd/scraper/main.go` and `pkg/cmd/site.go`.

### What I learned
- The right distinction is not “flag vs config”; it is **bootstrap config vs runtime config**.
- Site manifests are command-definition inputs, not normal runtime data.
- Sqleton's app-owned config loader plus early repository loading is the right precedent for scraper.

### What was tricky to build
- The tricky part is not reading YAML or env vars. The tricky part is timing: Cobra wants the command tree fully built before parsing, but scraper wants parsed site manifests to define parts of that tree.
- That means the implementation must include a very small pre-parser for bootstrap flags, not a normal `RunE`-time flag read.

### What warrants a second pair of eyes
- Merge order for config/env/bootstrap CLI dirs.
- Whether `--sites-manifest-dir` should remain declared as a normal persistent Cobra flag purely for UX/help, even though bootstrap parsing happens before Cobra.
- Whether we want a config-path override in the first cut or only the default app config path.

### What should be done in the future
- Implement `pkg/cmd/app_config.go`.
- Implement bootstrap arg pre-parsing.
- Add `NewRootCommandFromBootstrap(...)`.
- Update `main()` and add focused tests.

### Code review instructions
- Start with the new design doc in `design/02-bootstrap-config-and-early-site-command-loading.md`.
- Compare the proposed flow with sqleton's `collectRepositoryPaths(...)` and `initAllCommands(...)` pattern.
- After implementation lands, validate with `go test ./pkg/cmd/... -count=1` and `go test ./... -count=1`.

### Technical details
- Proposed env var: `SCRAPER_SITES_MANIFEST_DIRS`
- Proposed config shape:

```yaml
sitesManifestDirs:
  - /path/to/sites-a
  - /path/to/sites-b
```

- Proposed bootstrap merge order:
  1. config file
  2. environment variable
  3. bootstrap CLI flags

## Step 2: Add scraper-owned app config loading

I implemented the first code slice for the bootstrap architecture by adding an app-owned config loader in `pkg/cmd/app_config.go`. This mirrors the sqleton pattern directly: application startup config is decoded by scraper itself instead of being inferred from generic parser middleware or late Cobra callbacks. The goal of this slice was to make config-file and environment-driven site-directory discovery deterministic before introducing any bootstrap CLI pre-parser.

I kept the scope intentionally small. This slice does not build commands yet and it does not parse raw CLI args. It only establishes the authoritative config/env merge path for site manifest directories so the later bootstrap parser can feed one more input source into the same normalized pipeline.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the first part of the bootstrap pattern by giving scraper an app-owned config loader, similar to sqleton's repository config loader.

**Inferred user intent:** Separate app/bootstrap config from runtime command parsing so dynamic site commands can eventually be built from a pre-resolved set of manifest directories.

**Commit (code):** `57ae66e` — "Add scraper app-owned manifest-dir config loader"

### What I did
- Added `/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/app_config.go`.
- Added `/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/app_config_test.go`.
- Implemented:
  - `AppConfig`
  - `loadAppConfig(appName string)`
  - `loadAppConfigFromPath(configPath string)`
  - `collectSitesManifestDirs(appName string, bootstrapDirs []string)`
  - `sitesManifestDirsFromEnv()`
  - `normalizeManifestDirs(...)`
- Used `glazed/pkg/config.ResolveAppConfigPath` to locate the standard app config.
- Used `SCRAPER_SITES_MANIFEST_DIRS` as the environment override.
- Added tests for:
  - empty config path
  - YAML config decoding
  - env parsing via `filepath.SplitList`
  - config + env + bootstrap merge order
- Ran:

```bash
go test ./pkg/cmd/... -run 'TestLoadAppConfigFromPath|TestSitesManifestDirsFromEnv|TestCollectSitesManifestDirsMergesConfigEnvAndBootstrap' -count=1
```

### Why
- The bootstrap architecture needs one central place to resolve manifest directories from app-owned sources.
- If config/env normalization is spread across `main.go`, `root.go`, and tests, the later flag pre-parser would become harder to reason about.
- This slice establishes one reusable merge function that later code can call before building the Cobra tree.

### What worked
- The sqleton pattern translated cleanly.
- `glazed/pkg/config.ResolveAppConfigPath` is enough for the first cut; no extra config-path override was needed.
- The config/env merge tests were straightforward and gave a stable contract for later bootstrap work.

### What didn't work
- My first test run failed because I left `t.Parallel()` in tests that also used `t.Setenv(...)`.
- Exact failure:

```text
panic: testing: test using t.Setenv, t.Chdir, or cryptotest.SetGlobalRandom can not use t.Parallel
```

- Command that failed:

```bash
go test ./pkg/cmd/... -count=1
```

- I fixed this by removing `t.Parallel()` from the env-mutating tests in `pkg/cmd/app_config_test.go`.

### What I learned
- The normalization behavior is worth centralizing early because the later bootstrap parser should not need to care about trimming, env expansion, or de-duplication.
- `filepath.SplitList(...)` is the right portable choice for the env var because it keeps macOS/Linux path-list behavior aligned.

### What was tricky to build
- The subtle part was deciding whether config/env merging should be implemented in the future bootstrap parser or in an app-config module. Keeping it in `app_config.go` is cleaner because the bootstrap parser only contributes one source of directories; it should not own app config semantics.
- The test isolation issue with `t.Setenv(...)` and `t.Parallel()` was also easy to miss because the tests are tiny but still mutate process-global state.

### What warrants a second pair of eyes
- Whether path normalization should remain `filepath.Clean(os.ExpandEnv(...))` only, or eventually also support rebasing relative config-file paths against the config directory.
- Whether the env var name should stay plural (`SCRAPER_SITES_MANIFEST_DIRS`) or whether we also want a single-dir compatibility alias.

### What should be done in the future
- Add bootstrap CLI pre-parsing for repeated `--sites-manifest-dir` values.
- Add a bootstrap-aware root command constructor.
- Update `main()` to call that constructor before building the command tree.

### Code review instructions
- Start with `pkg/cmd/app_config.go`.
- Then read `pkg/cmd/app_config_test.go` to confirm the merge contract.
- Re-run:

```bash
go test ./pkg/cmd/... -run 'TestLoadAppConfigFromPath|TestSitesManifestDirsFromEnv|TestCollectSitesManifestDirsMergesConfigEnvAndBootstrap' -count=1
```

### Technical details
- Config YAML shape:

```yaml
sitesManifestDirs:
  - /path/to/sites-a
  - /path/to/sites-b
```

- Environment shape:

```text
SCRAPER_SITES_MANIFEST_DIRS=/path/a:/path/b
```

- Merge order implemented in this slice:
  1. config file
  2. env var
  3. bootstrap dirs (passed in by future code)

## Step 3: Add bootstrap pre-parsing and bootstrap-aware root construction

With the app-config layer in place, I implemented the actual bootstrap path that scraper's real binary needs. The important behavior change is that `cmd/scraper/main.go` no longer constructs the root command directly from `NewRootCommand(version)`. It now resolves manifest directories first — from config, environment, and bootstrap CLI flags — and only then builds the Cobra tree. That finally aligns scraper with the sqleton pattern that inspired this work.

This slice also let me remove the late-loading `LoadSitesFromFlag(...)` behavior from `api` and `worker`. Once the registry is built before command construction, late loading becomes redundant and conceptually misleading. The persistent flag remains declared on root for help/UX consistency, but the actual registry population now happens during bootstrap.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the pre-parse/bootstrap phase so scraper can discover site manifest directories before dynamic site verbs are built.

**Inferred user intent:** Make the real CLI behave like the test harness: site verbs should exist because manifests were resolved before the Cobra tree was constructed, not because later code happened to mutate the registry.

**Commit (code):** `10ee143` — "Bootstrap site manifest dirs before building scraper command tree"

### What I did
- Added `/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/bootstrap.go`.
- Added `/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/bootstrap_test.go`.
- Implemented:
  - `BootstrapOptions`
  - `ParseBootstrapArgs(args []string)`
  - `CollectSitesManifestDirs(appName string, args []string)`
  - `NewRootCommandFromBootstrap(version, args []string)`
- Updated `/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/root.go`:
  - root still declares `--sites-manifest-dir`
  - flag type changed to `StringSlice`
  - root construction now receives manifest dirs explicitly and bakes them into the pre-built registry
- Updated `/home/manuel/workspaces/2026-03-23/js-scraper/scraper/cmd/scraper/main.go` to call `NewRootCommandFromBootstrap(version, os.Args[1:])`
- Removed late `LoadSitesFromFlag(...)` calls from:
  - `/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/api.go`
  - `/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/worker.go`
- Added tests proving:
  - repeated bootstrap flags are parsed
  - unknown runtime flags do not break bootstrap parsing
  - `NewRootCommandFromBootstrap(...)` builds `site js-demo run seed`
  - config + env + bootstrap flag sources merge correctly
- Ran:

```bash
go test ./pkg/cmd/... -count=1
go test ./... -count=1
```

### Why
- Dynamic site verbs are part of the command-definition phase, not the execution phase.
- If `main()` waits until after normal Cobra parsing, `site js-demo run seed` may not exist yet.
- Pre-parsing a tiny bootstrap flag set solves this without forcing the whole app into a custom parser.

### What worked
- `pflag.FlagSet` with `ParseErrorsWhitelist.UnknownFlags = true` worked well for a minimal bootstrap parser.
- Keeping `NewRootCommand(version, dirs...)` as the deterministic constructor preserved the clean test seam.
- Removing late loading from `api`/`worker` simplified the mental model: one registry, built once, before the tree exists.

### What didn't work
- The first compile/test pass broke `pkg/cmd/site_test.go` because I changed `newRootCommand(...)` to take an extra `[]string` parameter.
- Exact build error:

```text
pkg/cmd/site_test.go:33:49: not enough arguments in call to newRootCommand
	have (string, *registry.Registry)
	want (string, *registry.Registry, []string)
```

- Command that failed:

```bash
go test ./pkg/cmd/... -count=1
```

- I fixed this by making `newRootCommand(...)` variadic again:

```go
func newRootCommand(version string, siteRegistry *siteregistry.Registry, manifestDirs ...string)
```

That restored test compatibility while still letting root construction seed the persistent flag with bootstrap-resolved dirs.

### What I learned
- The smallest viable bootstrap parser is enough. We do not need to pre-parse the whole CLI, only the app-owned flags that define command sources.
- Keeping the persistent Cobra flag even after bootstrap parsing is useful for help and operator discoverability, as long as the code comments make it clear that the real work happened earlier.

### What was tricky to build
- The trickiest part was deciding whether to keep late-loading in `worker`/`api` as a fallback. It looked harmless, but it created two competing registry-population models: bootstrap-time and run-time. That is exactly the ambiguity we were trying to remove.
- Another subtle point was making pre-parsing tolerate the rest of the real CLI args. A normal `pflag` parse would fail on unknown flags like `--engine-db` or `--count`; using `UnknownFlags` was necessary so bootstrap parsing could coexist with the later full Cobra parse.

### What warrants a second pair of eyes
- Whether we want to keep the root persistent flag purely as declarative/help UX, or whether that dual-role flag could confuse future maintainers.
- Whether bootstrap parsing should eventually also support an explicit config-path override, not just the default app config path.

### What should be done in the future
- Consider a small comment in `main.go` explaining the bootstrap-before-Cobra requirement.
- Decide whether a config-path override belongs in the first public CLI contract or in a later follow-up.

### Code review instructions
- Start with:
  - `pkg/cmd/bootstrap.go`
  - `pkg/cmd/bootstrap_test.go`
  - `pkg/cmd/root.go`
  - `cmd/scraper/main.go`
- Then verify that `pkg/cmd/api.go` and `pkg/cmd/worker.go` no longer mutate the registry at `RunE` time.
- Validate with:

```bash
go test ./pkg/cmd/... -count=1
go test ./... -count=1
```

### Technical details
- Bootstrap parser flag shape:

```text
--sites-manifest-dir /path/a --sites-manifest-dir /path/b
```

- Root persistent flag type is now `StringSlice` so Cobra accepts repeated values after the tree already exists.
- Effective bootstrap pipeline is now:

```text
raw args
-> ParseBootstrapArgs
-> load app config
-> load env dirs
-> merge + normalize
-> NewRootCommand(version, dirs...)
-> build dynamic site verbs
-> normal Cobra Execute()
```

## Step 4: Replace `pflag` bootstrap parsing with a manual scanner

After the main bootstrap slice was committed, I ran a real binary path check instead of relying only on unit tests. That exposed a subtle but important bug: `go run ./cmd/scraper --sites-manifest-dir ./sites site js-demo run seed --help` failed before the Cobra tree executed because the bootstrap `pflag.FlagSet` treated `--help` as its own help request. In other words, the bootstrap parser was still too command-like; it needed to be a dumb extractor that only notices site-manifest flags and ignores everything else.

I replaced the `pflag`-based parser with a tiny manual scanner that only recognizes:

- `--sites-manifest-dir /path`
- `--sites-manifest-dir=/path`

Everything else is ignored during bootstrap. That made the pre-parser robust against `--help`, command-local flags like `--count`, and arbitrary future Cobra flags.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Harden the bootstrap parsing so it behaves correctly on real CLI invocations, not just synthetic test args.

**Inferred user intent:** Ensure the early-loading architecture works in the actual binary without the bootstrap parser accidentally taking over normal Cobra behavior.

**Commit (code):** `7e0b43a` — "Make scraper bootstrap parser ignore help and non-bootstrap flags"

### What I did
- Replaced the `pflag.FlagSet` implementation in `pkg/cmd/bootstrap.go` with a manual scanner over raw args.
- Kept support for:
  - repeated `--sites-manifest-dir VALUE`
  - `--sites-manifest-dir=VALUE`
- Updated `pkg/cmd/bootstrap_test.go` to assert that `--help` and unrelated command flags are ignored by bootstrap parsing.
- Re-ran:

```bash
go test ./pkg/cmd/... -run 'TestParseBootstrapArgs|TestNewRootCommandFromBootstrap|TestCollectSitesManifestDirsMergesConfigEnvAndBootstrapFlag' -count=1
go test ./... -count=1
```

- Manually validated:

```bash
go run ./cmd/scraper --sites-manifest-dir ./sites site js-demo run seed --help
```

That command now renders the expected help output for the dynamic `seed` verb.

### Why
- Bootstrap parsing should extract one app-owned concern only: manifest dirs.
- It should not have any semantics for help, unknown flags, or later command-local arguments.
- A manual scanner is simpler and more robust than fighting `pflag` into acting like a partial parser.

### What worked
- The manual scanner eliminated the `--help` interception bug completely.
- The real `go run` path now behaves the same way as the tests.
- Supporting both spaced and `=` flag forms preserved reasonable CLI ergonomics.

### What didn't work
- The first bootstrap implementation used `pflag.ParseErrorsWhitelist.UnknownFlags = true`, which looked promising but still failed on `--help`.
- Exact observed failure from a manual check:

```text
newroot err: pflag: help requested
exit status 2
```

- Command that exposed it:

```bash
go run /tmp/scraper_check.go
```

where the helper invoked:

```go
scrapercmd.NewRootCommandFromBootstrap("dev", []string{"--sites-manifest-dir", "./sites", "site", "js-demo", "run", "seed", "--help"})
```

### What I learned
- A bootstrap parser that only needs one or two flags is usually better implemented as a tiny scanner than as a partial general-purpose flag parser.
- The real binary path is worth checking even when unit tests are strong, because CLI help flags are a classic edge case.

### What was tricky to build
- The tricky part was not parsing the site-dir flags themselves; it was avoiding accidental ownership of the rest of the command line.
- `pflag` is designed to be a real parser, which is the wrong abstraction here. Bootstrap needs extraction, not parsing.

### What warrants a second pair of eyes
- Whether we should document explicitly that bootstrap parsing recognizes only long-form `--sites-manifest-dir` today, not a short alias.
- Whether future bootstrap concerns (for example a config-path override) should still use the same manual scanner or justify a slightly richer structure.

### What should be done in the future
- N/A

### Code review instructions
- Review `pkg/cmd/bootstrap.go` with special attention to the switch cases for spaced and `=` forms.
- Confirm `pkg/cmd/bootstrap_test.go` covers `--help` and unrelated flags.
- Validate manually with:

```bash
go run ./cmd/scraper --sites-manifest-dir ./sites site js-demo run seed --help
```

### Technical details
- Recognized bootstrap forms:

```text
--sites-manifest-dir /path
--sites-manifest-dir=/path
```

- Ignored during bootstrap:
  - `--help`
  - `-h`
  - subcommand names
  - runtime flags such as `--engine-db`, `--count`, `--poll-interval`
  - positional args
