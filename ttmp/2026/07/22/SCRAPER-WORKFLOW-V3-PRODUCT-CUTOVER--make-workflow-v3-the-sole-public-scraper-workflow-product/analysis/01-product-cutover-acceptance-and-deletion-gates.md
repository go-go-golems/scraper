---
Title: Workflow V3 product cutover acceptance and deletion gates
Ticket: SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER
Status: active
Topics:
    - scraper
    - workflow-v3
    - durability
    - cleanup
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/cmd/root.go
      Note: Canonical V3 worker and explicit legacy worker boundary
    - Path: repo://pkg/workflowv3product/application.go
      Note: New product surface contains no legacy engine import
    - Path: repo://ttmp/2026/07/22/SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER--make-workflow-v3-the-sole-public-scraper-workflow-product/scripts/01-check-product-cutover.sh
      Note: Executable local and cross-repository guard
ExternalSources: []
Summary: Evidence-backed boundary between the completed Workflow V3 product surface and legacy packages that remain blocked by named downstream migrations.
LastUpdated: 2026-07-23T11:30:00-04:00
WhatFor: Prevent both new dependencies on the old engine and premature deletion while current site and RAG consumers remain.
WhenToUse: Run before changing public worker commands or deleting any legacy Scraper engine package.
---


# Workflow V3 product cutover acceptance and deletion gates

## Result

Workflow V3 is now the primary generic workflow product. The main binary owns pure JavaScript validation/compilation, durable submission, local execution, an independently restartable worker, list/show/follow/cancel operations, task-package inspection, and an authorized operator HTTP projection. The new surface has no import of `pkg/engine`, exact `pkg/workflow`, `pkg/services`, `pkg/sites`, or `pkg/js/runtime`.

The remaining site/API/engine system is intentionally not deleted in this phase. This is evidence-based, not an indefinite compatibility promise. There are 52 local legacy-importing files across production and tests, and 15 downstream RAG-evaluation files still importing legacy Scraper packages. Removing those packages now would violate preserved supported behavior and the accepted convergence order.

## Public command boundary

| Surface | Result | Evidence |
|---|---|---|
| generic authoring | Workflow V3 only | `scraper workflow validate/explain/compile` |
| generic submission/execution | Workflow V3 only | `scraper workflow submit/run` |
| canonical worker name | Workflow V3 only | `scraper worker run` exposes `--workflow-db`, `--artifact-root`, task packages, and capacities |
| generic inspection/control | Workflow V3 only | `workflow runs list/show/follow/cancel` and `/api/v3/workflow/*` |
| task extension | Workflow V3 package registry | `scraper task-packages list`, `pkg/taskpackages/cookbooklinear` |
| old site worker | explicitly legacy | `scraper legacy worker run` |
| old site/API/engine commands | retained for current consumers | `scraper site`, `scraper api`, and `scraper engine` |

There is no adapter from a Workflow V3 plan to the old scheduler and no adapter presenting old workflows as V3 runs.

## Deletion inventory

| Candidate | Current callers | Replacement gate | Disposition now |
|---|---|---|---|
| `pkg/engine` scheduler/store/runner/model | old CLI, API, metrics, runtime events, sites, services, exact RAG imports | `RAG-V2-EXECUTION-CUTOVER` plus retained site acceptance | retain; prohibit new V3 imports |
| exact `pkg/workflow` | old preparation workflow and RAG worker/smoke imports | `RAG-V2-WORKFLOW-LOWERING` and `RAG-V2-EXECUTION-CUTOVER` | retain; V3 product does not import it |
| `pkg/services/engineview` and submission | old HTTP API and RAG status/artifact handlers | V3 API consumers and RAG execution cutover | retain |
| `pkg/sites` and `pkg/js/runtime` | direct site commands, old worker, site migrations | explicit retained-site port/delete decisions | retain |
| old worker at public `worker run` | public name conflict | Workflow V3 restart and CLI acceptance | removed from canonical name; available only under `legacy` |
| old API/frontend model | current devctl and web application | later API/frontend consumer cutover | retain; V3 operator API is separate and stable |

## Guard command

Run:

```bash
RAG_EVAL_REPO=/home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system \
  ttmp/2026/07/22/SCRAPER-WORKFLOW-V3-PRODUCT-CUTOVER--*/scripts/01-check-product-cutover.sh
```

The guard:

1. rejects imports of superseded packages from the Workflow V3 product, task packages, new command file, and examples;
2. runs focused product and command acceptance tests;
3. proves the canonical worker uses V3 flags and the legacy worker remains explicitly namespaced;
4. refuses silent deletion while local production callers remain;
5. optionally verifies downstream RAG legacy imports so any changed gate forces this inventory to be reviewed.

Observed output:

```text
ok github.com/go-go-golems/scraper/pkg/workflowv3product
ok github.com/go-go-golems/scraper/pkg/cmd
downstream legacy callers: 15
Workflow V3 product cutover guard passed; local legacy callers: 52
```

## Cross-repository acceptance

The RAG repository was tested against the workspace Scraper implementation:

```bash
cd /home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system
go test ./internal/workflow ./internal/preparationworkflow -count=1
```

Both packages passed. This proves the worker command cut and new packages did not break current RAG library consumers. It does not claim that RAG has migrated to V3; the 15-import inventory proves the opposite and keeps deletion blocked.

## Conditions for the later destructive cut

Delete a retained cluster only when all of the following are evidenced in the owning successor ticket:

1. every production caller is ported or explicitly deleted;
2. the equivalent V3 fixture passes restart, retry, cancellation, artifact, and observation acceptance;
3. direct and transitive import scans are empty;
4. old commands, help, devctl services, API routes, frontend clients, migrations, and generated types are removed together;
5. full Scraper and downstream tests pass after deletion;
6. this inventory and guard are updated in the same focused commit.

“Unused by the new product” is not sufficient evidence for deletion while existing supported consumers remain.
