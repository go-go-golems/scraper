---
Title: Investigation diary
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
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Records why Goja helper sharing is the real cleanup target, not executor merging."
LastUpdated: 2026-04-07T16:05:00-04:00
WhatFor: "Resume the cleanup plan with the executor-boundary decision intact."
WhenToUse: "Use when implementing or reviewing the runtime helper extraction."
---

# Investigation diary

## Main observation

There are two executors because there are two execution phases:

- submit-time JS execution
- durable op execution

The cleanup target is the duplicated helper layer, not the existence of two executors.

## Main recommendation

Keep two executors. Extract shared helper code for conversion, parsing, and module support where the behavior is truly the same.
