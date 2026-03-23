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

The `scraper` CLI is the operator entrypoint for a durable scraping engine. The engine is responsible for workflow state, leases, retries, artifacts, and execution scheduling. Site packages, such as NEREVAL, provide the site-specific behavior on top of those primitives.

The implementation is intentionally split into two layers:

- Go owns the durable runtime: workflow state, op persistence, worker coordination, HTTP execution, queue control, and process-level observability.
- JavaScript owns programmable site behavior: extraction logic, fan-out decisions, record projection logic, and site-specific migrations for each site database.

The first milestone keeps the JS contract narrow. Scripts do not fetch directly. Instead, they inspect dependency results, persist records or artifacts, and emit additional ops for the Go scheduler to execute.

Use the CLI help system as the top-level guide for operators and new contributors:

- `scraper help` for the command tree
- `scraper help scraper-architecture-overview` for this design summary
- ticket `SCRAPER-DESIGN` in `ttmp/` for the detailed implementation guide and diary
