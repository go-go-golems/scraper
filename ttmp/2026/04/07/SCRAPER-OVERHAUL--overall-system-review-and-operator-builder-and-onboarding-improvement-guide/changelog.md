# Changelog

## 2026-04-07

- Created the `SCRAPER-OVERHAUL` review ticket and scaffold documents.
- Reviewed the site registry, engine model, scheduler, SQLite store, submit-verb runtime, worker JS runtime, catalog API, and key frontend pages.
- Confirmed that rate limiting is real and durable in the engine, but built-in sites generally do not define non-default queue policies.
- Confirmed that dependencies are explicit `dependsOn` edges emitted by submit verbs and worker JS runtimes and persisted in `op_dependencies`.
- Confirmed that worker-level proxy support exists and is documented in the earlier `SCRAPER-HTTP-PROXY` ticket, but is not surfaced in the current API or frontend.
- Confirmed that the catalog API and site detail page already expose more site/queue metadata than the workflow pages currently use.
- Identified product-level gaps for operators, verb builders, and onboarding, including weak site/workflow cross-linking, limited dependency visibility, missing help surfaces, and a placeholder throughput chart on the queue monitor.
- Wrote the main design and implementation guide with recommendations and phased follow-up work.
- Added `onboarding` to the docmgr vocabulary so the ticket validates cleanly.
- Related the main design doc to the most relevant backend/frontend files.
- Ran `docmgr doctor --ticket SCRAPER-OVERHAUL --stale-after 30` successfully.
- Uploaded the ticket bundle to reMarkable at `/ai/2026/04/07/SCRAPER-OVERHAUL`.

- Ticket closed as an umbrella review on 2026-04-07 after the analysis was split into focused follow-on implementation tickets.
