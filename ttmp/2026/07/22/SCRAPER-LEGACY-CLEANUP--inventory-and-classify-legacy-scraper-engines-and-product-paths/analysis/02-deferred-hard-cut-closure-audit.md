---
Title: Scraper deferred hard-cut closure audit
Ticket: SCRAPER-LEGACY-CLEANUP
Status: complete
Topics: [scraper, workflow-v3, cleanup, architecture]
DocType: analysis
Intent: long-term
Summary: Evidence for deletion of the superseded site engine and sole ownership by Workflow V3.
LastUpdated: 2026-07-24
---

# Scraper deferred hard-cut closure audit

## Replacement gates

All named gates passed before deletion:

- Workflow V3 product commands compile, submit, execute, resume, inspect, follow, cancel, serve bounded read models, and enumerate exact task packages.
- Workflow observations replace old metric/event projection with one deterministic retry-aware contract.
- Researchctl executes Workflow V3 as the sole data plane while retaining immutable laboratory identity and evidence.
- RAG study and intake callers use Workflow V3 task packages, external-operation custody, and observations; repository-wide downstream search finds zero old Scraper imports.
- Bounded real-provider generation, embedding, and reranking acceptance passed.

## Atomic deletion

Deleted the complete old product cluster rather than retaining adapters:

- old model/store/scheduler/runners in `pkg/engine`;
- old convenience workflow executor in `pkg/workflow`;
- dynamic site manifests, shipped sites, JS operation runtime, and site commands;
- old API/types/services, queue/engine projections, manual operation retry, runtime events, Prometheus wrappers, and protobufs;
- old frontend and generated Storybook bundle;
- legacy dev stack/bootstrap and site flags;
- obsolete help and tutorials.

The root now exposes only `workflow`, Workflow V3 `worker`, `task-packages`, and `version`. The operator API is `/api/v3/workflow`; no `/api/v1` route or compatibility read model remains.

## Preserved behavior

Workflow V3 retains durable leases/renewal, append-only attempts, deterministic retries, cancellation fencing, budgets, approval gates, external operations, maps/reductions, database effects, process isolation, content-addressed artifacts, bounded projections, and privacy rules. Scraping is no longer hard-coded as a generic engine concept; future scraping behavior must be a versioned domain task package.

## Acceptance

- Full sequential Go suite passes, including real Bubblewrap/cgroup isolation.
- Built all four product/integration binaries.
- Built-binary tmux smoke passed submit -> restarted worker -> terminal show/follow and authenticated API cancellation.
- Sole-product guard reports zero local and zero RAG downstream legacy importers and proves old root commands are absent.
- Module tidy removed dependencies used only by the duplicate engine/API/frontend.
- GolangCI-Lint passes in repository module mode with zero issues.
- Generation and `git diff --check` pass.

## Regression guard

Reject any production import of `scraper/pkg/engine`, `scraper/pkg/workflow`, old service/site/runtime packages, `/api/v1` route, `sites-manifest-dir`, or old root command. Add supported domain semantics through exact Workflow V3 task packages, never through a second scheduler/store/API lifecycle.
