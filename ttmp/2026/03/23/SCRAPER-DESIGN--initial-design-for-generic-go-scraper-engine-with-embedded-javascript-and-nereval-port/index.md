---
Title: Initial design for generic Go scraper engine with embedded JavaScript and nereval port
Ticket: SCRAPER-DESIGN
Status: active
Topics:
    - scraping
    - go
    - goja
    - javascript
    - nereval
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - local:scraper.md
Summary: Ticket collecting the initial architecture analysis, implementation guide, and research diary for building the new generic Go/goja scraper engine and porting the NEREVAL prototype.
LastUpdated: 2026-03-23T10:35:00-04:00
WhatFor: Provide a single workspace for the imported scraper sketch, the evidence-backed design guide, and the implementation diary for the initial scraper architecture work.
WhenToUse: Use when onboarding to the scraper project, reviewing the NEREVAL port plan, or starting implementation of the Go/goja engine in scraper/.
---

# Initial design for generic Go scraper engine with embedded JavaScript and nereval port

## Overview

This ticket turns the imported [scraper sketch](./sources/local/scraper.md) into an evidence-backed implementation guide for the `scraper/` repository. The central question is not only "what should the generic scraper engine look like?" but also "how does that design relate to the current NEREVAL prototype, and what should an engineer build first in `scraper/`?"

The main conclusion is that the imported architecture is a good fit for the current prototype. The NEREVAL JS code already demonstrates the need for durable ops, queue-keyed rate limiting, artifact-aware extraction, and resumable work. The new design guide makes those concerns explicit and maps them onto `go-go-goja` runtime primitives.

## Key Links

- **Primary design doc**: [design-doc/01-generic-go-scraper-engine-and-nereval-port-design-guide.md](./design-doc/01-generic-go-scraper-engine-and-nereval-port-design-guide.md)
- **Diary**: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)
- **Imported source**: [sources/local/scraper.md](./sources/local/scraper.md)
- **Earlier prototype repo**: `../2026-03-21--experiment-dom/`

## Status

Current status: **active**

Current deliverables in this ticket:

- imported `scraper.md` source,
- detailed design and implementation guide,
- investigation diary,
- ticket bookkeeping for follow-on implementation work.

## Topics

- scraping
- go
- goja
- javascript
- nereval
- architecture

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- `design-doc/` contains the main architecture and implementation guide.
- `reference/` contains the chronological diary and supporting references.
- `sources/` contains imported external or local source documents.
- `scripts/` is reserved for ticket-local helper code if later research needs it.
- `various/` is reserved for scratch notes.
- `archive/` is reserved for deprecated or superseded working material.
