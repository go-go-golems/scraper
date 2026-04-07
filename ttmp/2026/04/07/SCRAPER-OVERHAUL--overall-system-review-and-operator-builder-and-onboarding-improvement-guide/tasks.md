# Tasks

## Review And Documentation

- [x] Create the `SCRAPER-OVERHAUL` ticket workspace and baseline docs.
- [x] Study the engine, site registry, API, and frontend surfaces relevant to rate limits, dependencies, proxying, and help.
- [x] Write a detailed intern-facing design and implementation guide answering the requested questions with code evidence.
- [x] Record the investigation process, dead ends, and review notes in the diary.
- [ ] Keep the ticket up to date if follow-on implementation tickets are split out.

## Architecture Findings To Convert Into Product Work

- [ ] Clarify the vocabulary in product copy: site, verb, workflow, op, queue, dependency, artifact, runtime event.
- [ ] Decide whether to keep the current “per-queue only” rate-limit model or add authoring conveniences for per-verb queue conventions.
- [ ] Decide how proxy/runtime configuration should be surfaced safely in the API and UI.
- [ ] Decide whether embedded docs should be exposed through a docs API or curated frontend imports.

## Candidate Implementation Phases

- [ ] Phase 1: Link site context into workflow and queue pages.
- [ ] Phase 2: Improve queue monitor trustworthiness and rate-limit visibility.
- [ ] Phase 3: Add dependency-focused debugging surfaces in workflow detail and op detail.
- [ ] Phase 4: Surface runtime configuration, including proxy mode.
- [ ] Phase 5: Add a help center and page-level contextual help/tooltips.
- [ ] Phase 6: Improve large-history workflow navigation for operators managing 1000s of jobs over days.

## Validation And Publishing

- [x] Run `docmgr doctor --ticket SCRAPER-OVERHAUL --stale-after 30`.
- [x] Upload the ticket bundle to reMarkable.
