---
Title: Convert Hacker News and Slashdot to pure JS submit-verb sites
Ticket: SCRAPER-PURE-JS-SITES
Status: closed
Topics:
    - scraper
    - javascript
    - jsverbs
    - cli
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Ticket for removing the remaining site-specific Go submission wrappers from Hacker News and Slashdot so they follow the same JS submit-verb pattern as js-demo.
LastUpdated: 2026-03-23T21:45:00-04:00
WhatFor: Tracks the cleanup work needed to make the built-in HTTP exercise sites JS-first at submission time.
WhenToUse: Use when reviewing or extending the pure-JS site conversion workstream.
---

# Convert Hacker News and Slashdot to pure JS submit-verb sites

## Overview

This ticket removes the remaining bespoke Go submission path from the built-in Hacker News and Slashdot sites. After this change:

- site entrypoints are defined in JS under `verbs/*.js`
- the generic submit-verb host discovers and runs those commands
- workers still execute the durable op graph later
- only minimal declarative Go site definitions remain

## Key Links

- Design guide: [01-pure-js-site-conversion-plan-for-hacker-news-and-slashdot.md](./design-doc/01-pure-js-site-conversion-plan-for-hacker-news-and-slashdot.md)
- Diary: [01-investigation-diary.md](./reference/01-investigation-diary.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **closed**

## Topics

- scraper
- javascript
- jsverbs
- cli

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
