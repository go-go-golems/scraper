---
Title: 'Publish scraper: add bump-glazed, bump-go-go-golems, align with go-template, and prepare for release'
Ticket: SCRAPER-PUBLISH
Status: active
Topics:
    - release
    - go
    - glazed
    - scraper
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Prepare scraper for first real publish: align Makefile with go-template (add bump-go-go-golems, GOWORK=off), bump stale go-go-golems deps (glazed 1.2.14→1.3.6, go-go-goja 0.4.16→0.7.2), update CI workflows, add LICENSE, and verify build."
LastUpdated: 2026-06-01T16:35:41.686431428-04:00
WhatFor: ""
WhenToUse: ""
---

# Publish scraper: add bump-glazed, bump-go-go-golems, align with go-template, and prepare for release

## Overview

Scraper is tagged at `v0.0.1` but depends on severely outdated go-go-golems packages. This ticket tracks aligning the build/release infrastructure with the standard go-template, bumping dependencies, updating CI, and preparing for a proper publish.

**Current status**: Investigation complete, implementation not started.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- release
- go
- glazed
- scraper

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
