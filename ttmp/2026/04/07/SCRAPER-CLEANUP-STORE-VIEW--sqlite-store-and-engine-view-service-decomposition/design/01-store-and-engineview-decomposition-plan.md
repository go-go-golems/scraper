---
Title: Store and engine view decomposition plan
Ticket: SCRAPER-CLEANUP-STORE-VIEW
Status: active
Topics:
    - scraper
    - backend
    - architecture
    - cleanup
    - sqlite
    - api
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Detailed plan for splitting store.go and engineview/service.go into smaller files."
LastUpdated: 2026-04-07T16:05:00-04:00
WhatFor: "Guide a low-risk cleanup of the storage and read-model layers."
WhenToUse: "Use when implementing or reviewing the decomposition."
---

# Store and engine view decomposition plan

## Goal

Reduce maintenance drag in the storage and read-model layers by splitting large files into smaller responsibility-focused files while keeping behavior and package paths stable.

## Target structure

### SQLite store package

```text
pkg/engine/store/sqlite/
  store.go
  workflow_store.go
  op_store.go
  lease_store.go
  result_store.go
  queue_limiter.go
  sql_helpers.go
```

### Engine view package

```text
pkg/services/engineview/
  service.go
  workflow_read_service.go
  queue_read_service.go
  artifact_read_service.go
  workflow_mutation_service.go
  db_helpers.go
```

## Principles

- move code before redesigning code
- keep receiver types stable during the first pass
- keep API handlers and callers stable during the first pass
- keep tests green after each move

## Implementation order

1. split helper-heavy store sections first
2. split result/artifact logic next
3. split workflow/op reads next
4. split engineview helpers last

## Validation

```bash
go test ./pkg/engine/store/sqlite -count=1
go test ./pkg/services/engineview ./pkg/api/server -count=1
go test ./... -count=1
```
