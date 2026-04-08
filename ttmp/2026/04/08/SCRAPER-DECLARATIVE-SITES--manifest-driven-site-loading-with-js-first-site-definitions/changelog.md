# Changelog

## 2026-04-08 (session 2) — site manifest loading cleanup

- Diagnosed the root cause of the "site not registered" test failures: `defaults.NewRegistry()` used CWD-dependent `sites/` directory lookup which broke in tests.
- Restored Go site packages (`pkg/sites/{hackernews,jsdemo,nereval,slashdot}`) from git — the `//go:embed` directives and `ReadFixture()` test helpers require them.
- Removed the `sites/` top-level directory — built-in sites are embedded via Go packages; external sites use `--sites-manifest-dir`.
- Simplified `defaults` package: `NewRegistry()` registers all built-in sites; `LoadExternalSites(r, dir)` loads external manifests into an existing registry.
- Moved `--sites-manifest-dir` to a persistent root-level flag on `scraper` command (not per-subcommand).
- Added `LoadSitesFromFlag()` shared helper used by `worker run` and `api serve` to load external sites from the root flag.
- Added `NewRootCommandWithRegistry()` for tests that need a pre-populated registry.
- Updated `server_test.go` to use `defaults.NewRegistry()` instead of CWD-dependent path resolution.
- Updated `submission/service_test.go` to use `defaults.NewRegistry()` instead of `NewRegistryWithSitesDir()`.
- Simplified `defaults_test.go` — no more CWD tricks, just calls `NewRegistry()`.
- Removed `NewRegistryWithSitesDir()`, `Register()`, `DefaultSitesManifestPath()`, `DefaultSitesManifestDir` from defaults package.
- All `go test ./... -count=1` passing.

## 2026-04-08

- Initial workspace created.
- Added the declarative-sites design and implementation guide.
- Recorded the investigation diary and implementation task list.
- Validated the ticket with `docmgr doctor`.
- Uploaded the ticket bundle to reMarkable at `/ai/2026/04/08/SCRAPER-DECLARATIVE-SITES` as `Scraper declarative sites`.
- Expanded the ticket into a concrete implementation backlog covering manifest modeling, loader helpers, built-in site migration, and validation slices.
- Added the first implementation slice under `pkg/sites/manifest/`: manifest structs, bounded module IDs, validation helpers, and focused tests.
- Added manifest loading helpers that decode `site.yaml`, validate it, map it into `registry.Definition`, and register manifest-backed sites through a shared helper.
- Migrated `pkg/sites/jsdemo` to load its site definition from embedded `site.yaml` while preserving the existing `Definition()` and `Register(...)` seams.
- Migrated `pkg/sites/hackernews` to embedded `site.yaml`, including its HTTP queue rate limit, and added a mixed Go+manifest registry test.
- Re-ran `go test ./... -count=1`, re-ran `docmgr doctor`, and refreshed the reMarkable bundle with `--force`.
- Approved the follow-on scope to expose provenance in the catalog API, show it in the frontend, and add a no-Go site authoring tutorial.
- Added catalog/API provenance metadata so sites now report whether they are manifest-backed or Go-native, including manifest path for declarative sites.
- Added frontend provenance badges and detail text on the sites list and site detail pages.
- Added a new `scraper-adding-a-declarative-site` help/tutorial page and linked the older Go-native site tutorial to it.
