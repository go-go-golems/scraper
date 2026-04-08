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

- [ ] Expose manifest-origin metadata in the catalog API if useful.
- [ ] Decide whether the frontend should show whether a site is declarative or Go-native.
- [ ] Decide whether operator docs should include a “create a site without Go” tutorial.

## Validation

- [x] Run `docmgr doctor --ticket SCRAPER-DECLARATIVE-SITES --stale-after 30`.
- [x] Upload the bundled ticket docs to reMarkable.
- [x] Run focused manifest package tests after each slice.
- [x] Run `go test ./pkg/sites/... -count=1` after site migrations.
- [ ] Run `go test ./... -count=1` after the full declarative-site slice is complete.
- [ ] Re-run `docmgr doctor --ticket SCRAPER-DECLARATIVE-SITES --stale-after 30` after implementation updates.
- [ ] Re-upload the updated ticket bundle to reMarkable after implementation progress is recorded.
