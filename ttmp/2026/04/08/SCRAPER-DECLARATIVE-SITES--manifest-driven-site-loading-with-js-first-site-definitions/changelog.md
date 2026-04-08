# Changelog

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
