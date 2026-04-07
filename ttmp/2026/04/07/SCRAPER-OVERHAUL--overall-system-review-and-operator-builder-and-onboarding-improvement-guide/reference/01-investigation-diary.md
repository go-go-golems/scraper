---
Title: Investigation diary
Ticket: SCRAPER-OVERHAUL
Status: active
Topics:
    - scraper
    - architecture
    - onboarding
    - frontend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Chronological investigation notes for the scraper overhaul review, including evidence gathered, mistaken turns, major conclusions, and recommended follow-on tickets.
LastUpdated: 2026-04-07T12:05:00-04:00
WhatFor: Preserve the chronological investigation record behind the overhaul review so a future engineer can see what was checked, what evidence was used, and what still needs deeper implementation work.
WhenToUse: Use when reviewing how the conclusions in the design doc were reached, reproducing the review, or extending the overhaul work into implementation tickets.
---

# Investigation diary

## 2026-04-07

### Prompt context

The request was to create a new ticket for a general overhaul and improvement pass over scraper, then study the codebase and produce a detailed analysis/design/implementation guide. The review needed to answer specific questions about rate limits, per-verb limits, dependency creation, proxy support/history, site information in workflows, and help/tooltips. It also needed to consider three user roles:

- operator running many workflows over days,
- verb builder debugging workflows and scripts,
- new teammate onboarding into the system.

### Interpretation

This was not an implementation-first task. The right output was a serious architecture and product review that could stand on its own as an intern handoff. That meant grounding every answer in code and existing docs rather than producing generic product advice.

### What I looked at

I started from the architecture seams most likely to answer the review questions:

- `pkg/sites/registry/registry.go`
- `pkg/engine/model/types.go`
- `pkg/engine/scheduler/scheduler.go`
- `pkg/engine/store/sqlite/store.go`
- `pkg/sites/submitverbs/runtime.go`
- `pkg/js/runtime/executor.go`
- `pkg/services/catalog/service.go`
- `pkg/api/server/server.go`
- `web/src/pages/WorkflowsPage.tsx`
- `web/src/components/workflows/WorkflowTable.tsx`
- `web/src/pages/SiteDetailPage.tsx`
- `web/src/pages/QueueMonitorPage.tsx`
- `web/src/components/submit/VerbParameterForm.tsx`

I also searched the existing ticket/doc history for proxy-related work and checked the built-in site definitions to see whether queue policies were actually being used in shipped paths.

### What worked

- The engine/store model was straightforward to validate. Rate limiting and dependency handling are explicit and live in predictable places.
- The catalog and site-detail surfaces already contain useful metadata, which made it easier to identify UI gaps as visibility problems instead of backend capability problems.
- The doc history already contained a prior proxy ticket, which answered the “did we do this before?” question directly.

### What did not work

- My first quick search path was wrong. I initially looked for a top-level `sites/` tree, but this repo keeps site code under `pkg/sites/`. That was a cheap mistake, but worth recording because it is the kind of thing a new reviewer will trip over.
- The built-in site definitions make the product look less capable than the engine really is, because they mostly omit explicit queue policies. That created an initial false impression that rate limiting might not exist.
- `docmgr validate frontmatter --doc ttmp/...` resolved to `ttmp/ttmp/...` in this repo layout. Using absolute document paths worked immediately, so I switched to that instead of digging into the CLI path-resolution behavior during the review ticket.

### Key findings

- Rate limits do work. The token-bucket limiter is durable and enforced transactionally in SQLite.
- Built-in sites generally do not ship non-default queue policies, so the default demo/operator experience does not strongly show rate limiting.
- Rate limits are attached to `site + queue`, not to verbs or individual ops as first-class concepts.
- Verbs and worker scripts create dependency edges explicitly through `dependsOn`, and those edges are persisted in `op_dependencies`.
- Worker-level HTTP proxy support exists and has prior design history, but it is invisible in the current product UI.
- Site metadata and queue policy summaries are already exposed in the catalog/site detail path, but workflows do not link into them well.
- Help content exists in the CLI and submit forms, but not yet as a coherent frontend help model.
- The queue monitor still contains placeholder throughput data, which is risky for operator trust.

### Why the design guide is structured the way it is

I organized the guide around system roles rather than only around packages. That made it easier to answer the user’s actual concern: “how understandable is this system to the people using it?” The codebase already has good primitives, so the most useful document is one that maps those primitives to operator, builder, and onboarding journeys.

### What warrants a second pair of eyes

- Whether the product should keep “per-queue only” pacing as the sole mental model, or whether a higher-level authoring convenience for per-verb queue conventions is worthwhile.
- Whether embedded docs should be served through an HTTP docs API or curated/imported into the frontend build.
- Whether proxy visibility should be global only or also attached to HTTP/fetch-heavy workflow and op views.
- Whether the queue monitor should be improved immediately or temporarily downgraded until it can display trustworthy data.

### Recommended next tickets

- Workflow and queue cross-linking plus site metadata visibility.
- Dependency visibility and “why pending?” debugging UX.
- Runtime configuration/proxy visibility in the API and frontend.
- Frontend help center and contextual explanations using existing embedded docs.
- Long-history operator UX for large workflow volumes.

### Reproduction notes

Useful commands for re-running the review:

```bash
cd /home/manuel/workspaces/2026-03-23/js-scraper/scraper
rg -n "QueuePolicies|HelpFS|HelpRoot|DefaultQueuePolicy|RateLimitPolicy|dependsOn|http-proxy" pkg web
sed -n '1,220p' pkg/sites/registry/registry.go
sed -n '1,220p' pkg/engine/model/types.go
sed -n '220,520p' pkg/engine/store/sqlite/store.go
sed -n '1,220p' pkg/services/catalog/service.go
docmgr doctor --ticket SCRAPER-OVERHAUL --stale-after 30
remarquee cloud ls '/ai/2026/04/07/SCRAPER-OVERHAUL/' --long --non-interactive
```

### Related

- Main design guide: [../design-doc/01-overall-system-review-gap-analysis-and-implementation-guide-for-operators-verb-builders-and-onboarding.md](../design-doc/01-overall-system-review-gap-analysis-and-implementation-guide-for-operators-verb-builders-and-onboarding.md)
