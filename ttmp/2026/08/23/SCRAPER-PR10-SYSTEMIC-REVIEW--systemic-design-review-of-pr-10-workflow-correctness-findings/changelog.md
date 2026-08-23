# Changelog

## 2026-08-23

- Initial workspace created


## 2026-08-23

Captured all four PR 10 inline findings, traced them through runner/compiler/store/runtime and history, and proposed a four-phase single-owner invariant design with topology-matrix tests.

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/researchrunner/runner.go — Input custody and set admission findings
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3/compiler.go — Dependency graph and topology validation findings
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3sqlite/store.go — Readiness, output resolution, and terminal reconciliation findings


## 2026-08-23

Validated ticket structure, frontmatter, related files, Markdown fences, and whitespace; docmgr doctor reports all checks passed.


## 2026-08-23

Delivered the validated five-document review bundle to reMarkable at /ai/2026/08/23/SCRAPER-PR10-SYSTEMIC-REVIEW/SCRAPER PR10 Systemic Correctness Review.pdf.


## 2026-08-23

Accepted the systemic review for implementation, activated the ticket, and added focused custody, set-policy, dependency, lifecycle, validation, and documentation tasks.


## 2026-08-23

Step 5: staged verified scalar bytes directly into immutable artifact refs, added resolved-input byte ceilings, and separated ref submission from path staging (commit 5486f2e).

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/researchrunner/runner.go — Verified-byte custody and bounded reads
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3product/service.go — Immutable artifact-ref submission boundary


## 2026-08-23

Step 6: added explicit set-input maxItems contracts, compiler consumer/capacity checks, runner policy admission, and direct set pass-through outputs (commit 2dfdee1).

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3/compiler.go — Map/reduction compatibility validation
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3/types.go — Canonical ingress cardinality contract
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3sqlite/store.go — Direct set-input output resolution


## 2026-08-23

Step 7: derived readiness from value bindings, unified static/dynamic dependency lowering, validated typed cross-kind cycles, and rejected structurally invalid digest-valid plans (commit b0cdd1b).

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3/dependencies.go — Single dependency semantics owner
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3sqlite/expansion.go — Dynamic map lowering
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3sqlite/reduction.go — Dynamic reduction lowering
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3sqlite/store.go — Plan validation and static lowering

