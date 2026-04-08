# Tasks

## Research And Design

- [x] Create the `SCRAPER-DECLARATIVE-SITES` ticket workspace.
- [x] Review the current Go site registration contract.
- [x] Review how built-in sites are currently assembled and exposed through the catalog/API.
- [x] Write a detailed intern-facing analysis, design, and implementation guide.
- [x] Record the investigation process in the diary.

## Architecture Decisions

- [x] Confirm the boundary between declarative sites and Go-native extension sites.
- [x] Define the initial manifest format and validation rules.
- [x] Define how manifests map onto `registry.Definition`.
- [x] Decide how declarative sites declare queue policies.
- [x] Decide how declarative sites request standard native modules.
- [x] Decide how declarative sites expose help/docs metadata.
- [x] Decide how built-in embedded sites and external filesystem sites coexist.

## Backend Implementation Plan

- [x] Add `pkg/sites/manifest/manifest.go` with the public manifest structs.
- [x] Add `pkg/sites/manifest/validation.go` with path, module, and queue-policy validation helpers.
- [x] Add `pkg/sites/manifest/manifest_test.go` covering valid manifests and validation failures.
- [x] Add `pkg/sites/manifest/loader.go` that loads `site.yaml` from an `fs.FS` and maps it into `registry.Definition`.
- [x] Add `pkg/sites/manifest/modules.go` with the bounded mapping from manifest module IDs to `gggengine.ModuleSpec`.
- [x] Add `pkg/sites/manifest/loader_test.go` covering loader behavior, queue-policy normalization, and unknown module IDs.
- [x] Decide that no separate `registry.RegisterDefinition(...)` helper is needed because `registry.Register(...)` already covers the plain-definition path.
- [x] Add a `registry.RegisterManifestFS(...)` or equivalent helper so embedded and filesystem sites share one loading path.
- [x] Keep direct Go `registry.Definition` registration untouched as the fallback/escape hatch.
- [x] Migrate `pkg/sites/jsdemo` to embed and load `site.yaml` instead of hand-assembling `registry.Definition`.
- [x] Add a `pkg/sites/jsdemo/site.yaml` manifest containing roots, default modules, and database filename.
- [x] Add `pkg/sites/jsdemo/site_test.go` coverage proving the manifest-backed registration still loads verbs/scripts and keeps the existing end-to-end behavior.
- [x] Migrate `pkg/sites/hackernews` to `site.yaml` as the queue-policy proof point.
- [x] Add `pkg/sites/hackernews/site.yaml` with the `site:hackernews:http` rate limit encoded declaratively.
- [x] Add `pkg/sites/hackernews/site_test.go` coverage proving the manifest-backed queue policy is preserved.
- [x] Decide that `pkg/sites/defaults/defaults.go` should stay as explicit site registration for now, while individual site packages can become manifest-backed internally.
- [x] Add an integration test for mixed registries containing both Go-defined sites and manifest-defined sites.

## API And Product Follow-On

- [x] Decide to expose manifest-origin metadata in the catalog API.
- [x] Add provenance fields to `registry.Definition` with a safe default for legacy Go-defined sites.
- [x] Populate manifest provenance in `pkg/sites/manifest/loader.go`.
- [x] Extend `pkg/services/catalog/service.go` site summary/detail DTOs with provenance metadata.
- [x] Add or update catalog/API tests proving `/api/v1/sites` and `/api/v1/sites/{site}/detail` return provenance.
- [x] Decide that the frontend should show whether a site is declarative or Go-native.
- [x] Add provenance fields to `web/src/api/types.ts` and story fixtures.
- [x] Add a shared site-origin badge/chip component in the frontend.
- [x] Show provenance on the sites list cards.
- [x] Show provenance and manifest path details on the site detail page.
- [x] Decide to add a no-Go site authoring tutorial.
- [x] Add a new help/tutorial page for creating a declarative site without Go.
- [x] Update the older “adding a site” tutorial to reference the new declarative path.
- [x] Validate the new help pages through the embedded help system if practical.

## Phase 1: Clean up site manifest loading (done)

- [x] Add `--sites-manifest-dir` as a persistent flag on the root command.
- [x] Simplify `defaults` package: `NewRegistry()` + `LoadExternalSites()`.
- [x] Add `NewRootCommandWithRegistry()` for tests.
- [x] All `go test ./... -count=1` passing.

## Phase 2: Make all sites external (done)

- [x] Copy site content to `sites/` at repo root (YAML, JS, SQL, fixtures).
- [x] Add `testfixtures` helper package for reading HTML fixtures from `sites/`.
- [x] Rewrite `defaults.NewRegistry()` → `NewRegistryFromDirs(dirs ...string)` (no Go site imports).
- [x] Update `NewRootCommand()` to accept `manifestDirs ...string` and load from them.
- [x] Delete Go site packages from `pkg/sites/`.
- [x] Update all tests: use `NewRootCommand(version, testfixtures.SitesDir(t))`.
- [x] Fix tests that used `jsdemo.Definition()` / `hackernews.Definition()` — load from manifest via `sitemanifest.LoadDefinition()`.
- [x] Fix tests that used `ReadFixture()` — use `testfixtures.ReadFixture()`.
- [x] Verify `go test ./... -count=1` passes (16 packages, all green).

## Phase 3: Bootstrap config and early command loading

- [ ] Add a detailed implementation/design document for bootstrap config and early site command loading.
- [ ] Add `pkg/cmd/app_config.go` with scraper-owned app config loading from config file + env.
- [ ] Add `pkg/cmd/app_config_test.go` for YAML loading, env parsing, and merge behavior.
- [ ] Add `pkg/cmd/bootstrap.go` with bootstrap flag pre-parsing for repeated `--sites-manifest-dir` values.
- [ ] Add `pkg/cmd/bootstrap_test.go` for bootstrap arg parsing and normalization.
- [ ] Add `NewRootCommandFromBootstrap(version, args)` that resolves dirs before building the command tree.
- [ ] Update `cmd/scraper/main.go` to use bootstrap-aware construction.
- [ ] Decide whether late `LoadSitesFromFlag(...)` in `api`/`worker` is still needed; remove it if redundant.
- [ ] Add an end-to-end test proving dynamic `site js-demo run seed` commands exist when manifests are provided only through bootstrap sources.
- [ ] Re-run `go test ./... -count=1`.

## Validation

- [x] Run `docmgr doctor --ticket SCRAPER-DECLARATIVE-SITES --stale-after 30`.
- [x] Upload the bundled ticket docs to reMarkable.
- [x] Run focused manifest package tests after each slice.
- [x] Run `go test ./pkg/sites/... -count=1` after site migrations.
- [x] Run `go test ./... -count=1` after the full declarative-site slice is complete.
- [x] Re-run `docmgr doctor --ticket SCRAPER-DECLARATIVE-SITES --stale-after 30` after implementation updates.
- [x] Re-upload the updated ticket bundle to reMarkable after implementation progress is recorded.
