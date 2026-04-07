---
Title: 'Documentation Review: pkg/doc Embedded Help Pages'
Ticket: DOC-REVIEW
Status: active
Topics:
    - documentation
    - scraper
    - onboarding
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/doc/topics/scraper-architecture-overview.md
      Note: Primary review target - architecture overview
    - Path: pkg/doc/topics/scraper-queue-policies-and-rate-limiting.md
      Note: Primary review target - queue policies
    - Path: pkg/doc/topics/scraper-runtime-model.md
      Note: Primary review target - runtime model
    - Path: pkg/doc/tutorials/scraper-adding-a-site.md
      Note: Primary review target - adding a site tutorial
    - Path: pkg/doc/tutorials/scraper-new-developer-onboarding.md
      Note: Primary review target - onboarding tutorial
    - Path: pkg/js/runtime/executor.go
      Note: Source of truth for execution-time JS ctx API
    - Path: pkg/sites/submitverbs/runtime.go
      Note: Source of truth for submission-time JS ctx API
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-23T20:44:40.495251623-04:00
WhatFor: ""
WhenToUse: ""
---


# Documentation Review: pkg/doc Embedded Help Pages

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- documentation
- scraper
- onboarding
- architecture

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
