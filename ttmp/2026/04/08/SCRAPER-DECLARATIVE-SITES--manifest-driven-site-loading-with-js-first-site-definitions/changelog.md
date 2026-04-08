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
