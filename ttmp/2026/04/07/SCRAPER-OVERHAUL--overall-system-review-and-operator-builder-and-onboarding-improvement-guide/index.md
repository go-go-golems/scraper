---
Title: Overall system review and operator, builder, and onboarding improvement guide
Ticket: SCRAPER-OVERHAUL
Status: closed
Topics:
    - scraper
    - architecture
    - onboarding
    - frontend
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Ticket index for the broad scraper overhaul review covering operator UX, verb-builder debugging, onboarding, queue/rate-limit clarity, dependency visibility, proxy visibility, and help surfaces.
LastUpdated: 2026-04-07T12:05:00-04:00
WhatFor: Track the overall codebase review and the resulting product, architecture, and onboarding improvements needed to make scraper easier to operate, debug, and learn.
WhenToUse: Use this ticket when planning or reviewing broad usability and architecture improvements touching rate limits, dependencies, proxy visibility, workflow/site navigation, and help surfaces.
---

# Overall system review and operator, builder, and onboarding improvement guide

## Overview

This ticket captures a broad review of scraper as a product rather than as a collection of subsystems. The main question is not whether the engine works. The main question is whether operators, verb builders, and new teammates can understand what the system is doing without reading the source first.

The review focuses on:

- queue policy and rate-limit behavior,
- dependency creation and visibility,
- proxy support and operational visibility,
- site metadata and cross-linking in the UI,
- embedded help, onboarding, and contextual explanation,
- overall operator ergonomics for long-running workloads.

The primary outcome is a detailed, intern-facing design and implementation guide that explains the current architecture, answers the review questions with evidence, and proposes a phased improvement plan.

## Key Links

- Main design doc: [design-doc/01-overall-system-review-gap-analysis-and-implementation-guide-for-operators-verb-builders-and-onboarding.md](./design-doc/01-overall-system-review-gap-analysis-and-implementation-guide-for-operators-verb-builders-and-onboarding.md)
- Investigation diary: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **closed**

Current review status:

- architecture evidence gathered across engine, sites, API, and frontend,
- design guide drafted with answers to the requested questions,
- implementation work intentionally deferred and split into narrower follow-on tickets.

## Topics

- scraper
- architecture
- onboarding
- frontend

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- `design-doc/` - main review and implementation guide
- `reference/` - chronological investigation notes and supporting context
- `tasks.md` - phased follow-up work derived from the review
- `changelog.md` - document history and major findings
