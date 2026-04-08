---
Title: Investigation diary
Ticket: SCRAPER-DECLARATIVE-SITES
Status: active
Topics:
    - scraper
    - architecture
    - backend
    - javascript
    - onboarding
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Records the reasoning behind the declarative-sites design ticket.
LastUpdated: 2026-04-08T09:20:00-04:00
WhatFor: Preserve the current-state observations and design rationale for manifest-driven site loading.
WhenToUse: Use when implementing or reviewing the declarative-sites design.
---

# Investigation diary

## Prompt summary

The request was to create a new ticket for declarative sites and write a detailed intern-facing analysis, design, and implementation guide. The underlying question was whether scraper still needs Go-defined sites, or whether declarative metadata plus JavaScript could carry most site definitions.

## Current-state observations

The current site seam is clearly Go-defined:

- [pkg/sites/registry/registry.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/registry/registry.go) defines the registration contract.
- [pkg/sites/defaults/defaults.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/defaults/defaults.go) explicitly registers built-in sites in Go.
- Site packages like [pkg/sites/hackernews/site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/site.go) and [pkg/sites/jsdemo/site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site.go) mostly exist to point at embedded filesystem roots and declare metadata such as queue policies.

At the same time, most actual site behavior is already file-oriented:

- scripts are JavaScript files
- verbs are JavaScript files
- migrations are SQL and sometimes JS files
- help/docs can live in files

That means the current model is only partially Go-native. Most runtime logic is already declarative-ish plus JS. The main Go-only part is the registration envelope.

## Working conclusion

The best direction is not “eliminate Go entirely.” The best direction is:

- ordinary sites should be manifest-driven and JS-first
- advanced sites can still opt into Go-native definitions when they need custom native modules or special runtime hooks

This preserves flexibility while reducing friction for the common case.

## Design emphasis

The guide focuses on:

- what belongs in a manifest
- what should stay out of the manifest
- how a manifest loader should build `registry.Definition`
- how embedded built-in sites and filesystem-loaded sites can coexist
- how to migrate safely without breaking current sites all at once

## Validation and publishing

After writing the ticket docs, I validated the workspace with:

```bash
docmgr doctor --ticket SCRAPER-DECLARATIVE-SITES --stale-after 30
```

The doctor report passed without findings beyond the success marker.

I then uploaded the full bundle to reMarkable with:

```bash
remarquee upload bundle ttmp/2026/04/08/SCRAPER-DECLARATIVE-SITES--manifest-driven-site-loading-with-js-first-site-definitions \
  --remote-dir /ai/2026/04/08/SCRAPER-DECLARATIVE-SITES \
  --name "Scraper declarative sites" \
  --non-interactive
```

I also verified the remote directory with:

```bash
remarquee cloud ls /ai/2026/04/08/SCRAPER-DECLARATIVE-SITES
```

The uploaded document appeared as `Scraper declarative sites`.

## 2026-04-08 implementation kickoff

Before touching code, I refined the high-level task list into a real execution backlog. The original ticket tasks were still design-shaped, but not yet granular enough for incremental implementation and commits. I inspected:

- [pkg/sites/registry/registry.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/registry/registry.go)
- [pkg/sites/defaults/defaults.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/defaults/defaults.go)
- [pkg/sites/jsdemo/site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/jsdemo/site.go)
- [pkg/sites/hackernews/site.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/site.go)
- [pkg/services/catalog/service.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/services/catalog/service.go)

That review confirmed a low-risk implementation direction:

- keep `registry.Definition` as the stable internal contract
- add a manifest package that maps declarative metadata into `registry.Definition`
- migrate built-in sites one at a time rather than changing registry consumers first

I then expanded the task list so the implementation could be done in slices:

- manifest structs and validation
- manifest loader and module mapping
- registry helper integration
- `js-demo` migration as the simple proof point
- `hackernews` migration as the queue-policy proof point
- mixed-registry validation

That gives a clean commit structure and keeps the diary tied to concrete slices rather than one large undifferentiated refactor.

## 2026-04-08 manifest package slice

I implemented the first code slice in a new package:

- [pkg/sites/manifest/manifest.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/manifest/manifest.go)
- [pkg/sites/manifest/modules.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/manifest/modules.go)
- [pkg/sites/manifest/validation.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/manifest/validation.go)
- [pkg/sites/manifest/manifest_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/manifest/manifest_test.go)

The point of this slice was to establish a strict declarative schema before touching registry wiring. I intentionally kept the first version small:

- one manifest type representing site metadata
- one bounded module identifier, `default-registry`
- validation for relative roots, file-name-only DB names, duplicate queue policies, and token-bucket rate limits

I did not add YAML decoding or `registry.Definition` mapping yet. That separation matters because it lets the validation contract settle before loader code starts depending on it.

I validated this slice with:

```bash
gofmt -w pkg/sites/manifest/*.go
go test ./pkg/sites/manifest -count=1
```

The tests passed cleanly on the first run.
