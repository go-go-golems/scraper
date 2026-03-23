---
Title: Scraper Architecture Overview
Slug: scraper-architecture-overview
Short: "Overview of the scraper engine shape, CLI foundation, and the split between engine runtime and site code."
Topics:
  - scraper
  - architecture
  - go
  - goja
  - javascript
  - cli
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

# Scraper Architecture Overview

The `scraper` CLI is the operator entrypoint for a durable scraping engine. The engine is responsible for workflow state, leases, retries, artifacts, and execution scheduling. Site packages, such as Hacker News, Slashdot, `js-demo`, and later NEREVAL, provide the site-specific behavior on top of those primitives.

The implementation is intentionally split into two layers:

- Go owns the durable runtime: workflow state, op persistence, worker coordination, HTTP execution, queue control, and process-level observability.
- JavaScript owns programmable site behavior: extraction logic, fan-out decisions, record projection logic, and site-specific migrations for each site database.

The first milestone keeps the JS contract narrow. Scripts do not fetch directly. Instead, they inspect dependency results, persist records or artifacts, and emit additional ops for the Go scheduler to execute.

When JavaScript needs database access, the runtime should expose preconfigured module names such as `require("scraper-db")` and `require("site-db")`. JS should not be responsible for discovering or opening those SQLite files itself.

In the current codebase, JS ops are loaded from the site script filesystem declared in the site registry, and the `js` runner expects op metadata to include a `script` entry naming the module to execute.

The current HTTP runner uses kind `http/fetch` and expects op input to carry a `request` object plus optional `persistBody` and `artifactName` fields. Request URLs, headers, form fields, and body text may be rendered from the op/workflow input template context before the request is sent.

The current scheduler layer recovers expired leases back to `ready`, promotes dependency-satisfied ops automatically, cancels pending ops blocked by required failed dependencies, and treats each `site + queue` pair as a single active rate domain.

Use the CLI help system as the top-level guide for operators and new contributors:

- `scraper help` for the command tree
- `scraper help scraper-architecture-overview` for this design summary
- `scraper engine status` for a quick engine DB and runtime-state smoke check
- `scraper engine migrations status` for applied-vs-known migration visibility
- `scraper site migrate <site>` to initialize or upgrade a site-specific database
- built-in exercise sites currently include `hackernews` and `slashdot` for `js -> http/fetch -> js -> site-db` validation, plus `js-demo` for a pure-JS `js -> js -> site-db` path without any HTTP dependency
- `scraper site js-demo run <seed|item|summary>` to exercise different pure-JS entrypoints locally
- ticket `SCRAPER-DESIGN` in `ttmp/` for the detailed implementation guide and diary
