---
Title: Goja runtime helper consolidation plan
Ticket: SCRAPER-CLEANUP-GOJA-RUNTIME
Status: active
Topics:
    - scraper
    - backend
    - architecture
    - cleanup
    - javascript
    - goja
    - jsverbs
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Detailed plan for extracting shared Goja runtime helpers while preserving separate executors."
LastUpdated: 2026-04-07T16:05:00-04:00
WhatFor: "Guide a low-risk consolidation of duplicated Goja helper code."
WhenToUse: "Use when implementing or reviewing the runtime helper extraction."
---

# Goja runtime helper consolidation plan

## Goal

Reduce duplicated runtime plumbing between the op executor and the submit executor without erasing the fact that they have different contracts and contexts.

## Candidate shared helper areas

- JSON/raw-message helpers
- primitive coercion helpers
- metadata/map helpers
- dependency parsing
- retry policy parsing
- compatible module/path helpers

## Target structure

```text
pkg/js/runtimecodec/
  json.go
  coercion.go
  metadata.go
  dependencies.go
  retry.go
  module_paths.go
```

## What stays local

- op runtime context construction
- submit runtime context construction
- op result shaping
- submit return interpretation

## Validation

```bash
go test ./pkg/js/runtime -count=1
go test ./pkg/sites/submitverbs -count=1
go test ./... -count=1
```
