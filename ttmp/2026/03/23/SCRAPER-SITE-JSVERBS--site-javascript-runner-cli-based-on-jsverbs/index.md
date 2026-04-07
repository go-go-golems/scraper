---
Title: Site JavaScript runner CLI based on jsverbs
Ticket: SCRAPER-SITE-JSVERBS
Status: closed
Topics:
    - scraping
    - go
    - goja
    - javascript
    - glazed
    - cli
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Ticket for designing a site-aware JavaScript CLI runner in scraper that reuses go-go-goja jsverbs without collapsing it into the existing workflow op runtime.
LastUpdated: 2026-03-23T14:38:00-04:00
WhatFor: Organize the analysis and implementation guidance for adding CLI-testable site verbs to scraper.
WhenToUse: Use when implementing or reviewing the future `scraper site js` command tree and its relation to site scripts, libs, and runtime modules.
---

# Site JavaScript runner CLI based on jsverbs

## Overview

This ticket studies how `go-go-goja/pkg/jsverbs` can be applied inside `scraper` so site-specific JavaScript can be exercised from the CLI. The central design question is not “how do we run JS?” because `scraper` already runs JS. The real question is how to add an operator-facing, Glazed-native site command layer without confusing it with the existing durable workflow op runtime.

The primary conclusion is that `scraper` should keep two JavaScript entrypoint types:

- `scripts/` for scheduler-driven workflow ops,
- `verbs/` for CLI-driven jsverbs,
- with shared `lib/` helpers beneath both.

## Key Links

- **Primary analysis**: [design-doc/01-site-javascript-cli-runner-with-jsverbs-design-and-implementation-guide.md](./design-doc/01-site-javascript-cli-runner-with-jsverbs-design-and-implementation-guide.md)
- **Diary**: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)
- **Tasks**: [tasks.md](./tasks.md)
- **Changelog**: [changelog.md](./changelog.md)

## Status

Current status: **closed**

The design and diary are written. Validation and reMarkable delivery are the next ticket steps.

## Recommended Direction

- Keep scheduler op scripts and CLI verbs as separate runtime contracts.
- Use `jsverbs.ScanFS(...)` on an embedded site-owned `verbs/` tree.
- Mount generated commands under `scraper site js <site> ...`.
- Reuse shared `lib/` helpers and preconfigured DB modules.

## Structure

- `design-doc/` — primary implementation guide
- `reference/` — investigation diary
- `tasks.md` — phased checklist
- `changelog.md` — ticket history
