---
Title: JS-scanned site submission verbs with durable worker execution
Ticket: SCRAPER-JS-SUBMIT-VERBS
Status: active
Topics:
    - scraper
    - javascript
    - cli
    - glazed
    - jsverbs
    - worker
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Implementation ticket for replacing handwritten site submission commands with JS-scanned Glazed verbs while keeping durable execution in a separate worker process."
LastUpdated: 2026-03-23T17:40:56.397859671-04:00
WhatFor: "Collects the design, tasks, diary, and implementation details for JS-defined site submission commands and the worker execution path they feed."
WhenToUse: "Use when implementing or reviewing JS-scanned site verbs, workflow submission helpers, and worker-driven execution."
---

# JS-scanned site submission verbs with durable worker execution

## Overview

This ticket implements a new command-discovery path for the scraper engine. Site-facing submission commands should be defined in JavaScript with `__verb__` metadata, discovered by Go at CLI startup, and wrapped by Go so they can prepare databases, run site migrations, and submit durable workflow state. A separate worker process should then poll the engine DB and execute the queued ops.

## Key Links

- [Design guide](./design-doc/01-js-scanned-site-submission-verbs-design-and-implementation-guide.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Status

Current status: **active**

This ticket is both a design and implementation ticket. The first validation target is `js-demo`, because it can prove the command-submission and worker-execution split without live HTTP dependencies.

## Topics

- scraper
- javascript
- cli
- glazed
- jsverbs
- worker

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
