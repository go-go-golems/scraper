---
Title: Manifest-driven site loading with JS-first site definitions
Ticket: SCRAPER-DECLARATIVE-SITES
Status: done
Topics:
    - scraper
    - architecture
    - backend
    - javascript
    - api
    - onboarding
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/sites/registry/registry.go
      Note: Current Go registration contract for sites.
    - Path: pkg/sites/defaults/defaults.go
      Note: Current built-in registry bootstrap path that hardcodes Go site registration.
    - Path: pkg/services/catalog/service.go
      Note: Current site metadata surface consumed by the API and frontend.
ExternalSources: []
Summary: Design ticket for replacing most Go-defined site registration with manifest-driven, declarative site loading while retaining Go as an escape hatch for advanced native integrations.
LastUpdated: 2026-04-08T09:20:00-04:00
WhatFor: Explain how scraper can load ordinary sites from manifests plus JS assets instead of requiring a Go site definition for every site.
WhenToUse: Use when planning or implementing declarative site manifests, manifest loaders, JS-first site packaging, or migration away from Go-only site registration.
---

# Manifest-driven site loading with JS-first site definitions

## Overview

This ticket studies how to move scraper from a Go-defined site registration model toward a declarative, manifest-driven model where most ordinary sites can be described by files plus JavaScript rather than by handwritten Go packages.

Today, sites are registered in Go through [registry.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/registry/registry.go) and then assembled in [defaults.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/defaults/defaults.go). That works, but it means even simple site metadata changes still require editing Go code, rebuilding, and keeping a site package around mostly to point at filesystem roots.

The main recommendation in this ticket is:

- keep Go registration as an advanced escape hatch
- add a manifest schema for ordinary sites
- load scripts, verbs, migrations, help, and queue policies from the manifest
- keep JavaScript as the main behavior layer
- restrict declarative sites to bounded, known-native capabilities at first

This is an intern-facing design ticket. It explains the current state, the tradeoffs, the proposed manifest schema, the loader architecture, the migration plan, the testing strategy, and the rollout path.

## Key Links

- Main design doc: [design/01-declarative-site-manifest-architecture-and-implementation-guide.md](./design/01-declarative-site-manifest-architecture-and-implementation-guide.md)
- Investigation diary: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **done**

This ticket is complete. The declarative-sites migration, bootstrap loading, tests, and docs are now in place.

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Structure

- `design/` - main architecture and implementation guide
- `reference/` - investigation history and working notes
- `tasks.md` - phased backlog for implementation
- `changelog.md` - key ticket updates
