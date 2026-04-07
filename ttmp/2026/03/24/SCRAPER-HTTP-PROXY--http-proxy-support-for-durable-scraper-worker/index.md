---
Title: HTTP proxy support for durable scraper worker
Ticket: SCRAPER-HTTP-PROXY
Status: closed
Topics:
    - scraper
    - http
    - proxy
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/engine/config/config.go
      Note: Current HTTP config shape and future home of explicit proxy configuration
    - Path: pkg/engine/runner/http.go
      Note: Worker-side HTTP request execution and transport construction
    - Path: pkg/cmd/worker.go
      Note: CLI surface for exposing global worker proxy configuration
ExternalSources: []
Summary: Ticket workspace for adding explicit, testable HTTP proxy support to the durable scraper worker while preserving current environment-based fallback behavior.
LastUpdated: 2026-03-24T18:53:54.42159881-04:00
WhatFor: Tracks the design and implementation of first-class proxy support for worker-side http/fetch operations.
WhenToUse: Use when implementing, reviewing, or extending proxy behavior in the worker and HTTP runner.
---

# HTTP proxy support for durable scraper worker

## Overview

This ticket covers explicit HTTP proxy support for worker-side `http/fetch` operations.

The important architecture decision is that scraper already has implicit environment-based proxy behavior through Go’s default transport, but it does not yet have first-class scraper-owned proxy configuration. This ticket closes that gap by adding:

- explicit worker-level proxy config
- explicit CLI flag wiring
- explicit HTTP runner transport construction
- tests that prove proxied routing works

## Key Links

- Design guide: [design/01-http-proxy-support-architecture-and-implementation-guide.md](./design/01-http-proxy-support-architecture-and-implementation-guide.md)
- Diary: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **closed**

Current ticket state:

- design guide written
- tasks expanded
- implementation landed in code
- initial reMarkable upload done
- refreshed ticket sync pending

## Topics

- scraper
- http
- proxy

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
