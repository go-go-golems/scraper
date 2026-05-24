---
Title: 'JS Development Environment for Scraper: syntax-highlighted verb viewer, script reload, execution replay, and REPL console'
Ticket: SCRAPER-JS-DEVENV
Status: active
Topics:
    - scraper
    - frontend
    - ui-design
    - javascript
    - developer-experience
    - debugger
    - developer-tools
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - "SCRAPER-OP-DEBUGGER:Related ticket for workflow artifact browsing and per-op JS replay debugging"
Summary: "Design and implementation guide for a JS developer environment inside the scraper web UI: syntax highlighting, script reload, execution runner with ctx input, execution history browser, and REPL console."
LastUpdated: 2026-04-07T19:00:00-04:00
WhatFor: "Central reference for all JS dev environment UI design decisions and implementation guidance."
WhenToUse: "Use when implementing or reviewing any JS dev environment feature in the scraper UI."
---

# JS Development Environment for Scraper

## Overview

This ticket designs and guides the implementation of a first-class JavaScript
development environment inside the scraper web application. The goal is to give
site authors and operators a cohesive, browser-based workspace for writing,
testing, debugging, and iterating on scraper JS scripts.

The design covers five major features:

1. **Syntax-highlighted verb viewer** — replace plain-text code with token-level JS highlighting.
2. **Script reload from disk** — one-click reload to see file changes immediately.
3. **Execution runner with ctx input** — run scripts with custom `ctx` input from the UI.
4. **Execution history browser** — view, compare, and re-run past executions.
5. **REPL console widget** — interactive JS console for the scraper runtime.

## Key Links

- **Main design guide**: [design-doc/01-js-dev-environment-ui-design-and-implementation-guide.md](./design-doc/01-js-dev-environment-ui-design-and-implementation-guide.md)
- **Investigation diary**: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)
- **Related ticket**: SCRAPER-OP-DEBUGGER (workflow artifact browsing and per-op JS replay debugging)

## Status

Current status: **active**

Design document is complete. No implementation has started yet.

## Topics

- scraper
- frontend
- ui-design
- javascript
- developer-experience
- debugger
- developer-tools

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design-doc/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
