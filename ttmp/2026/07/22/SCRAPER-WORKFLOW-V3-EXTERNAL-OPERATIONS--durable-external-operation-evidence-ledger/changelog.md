# Changelog

## 2026-07-22

- Initial workspace created


## 2026-07-22

Created an evidence-backed intern implementation guide for a generic Workflow V3 external-operation ledger, including RAG and researchctl boundaries, schema/API pseudocode, decisions, phased tasks, and failure/privacy qualification

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/ttmp/2026/07/22/SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS--durable-external-operation-evidence-ledger/design-doc/01-durable-external-operation-evidence-ledger-design-and-implementation-guide.md — Primary architecture and implementation guide
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/ttmp/2026/07/22/SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS--durable-external-operation-evidence-ledger/reference/01-investigation-diary.md — Chronological investigation and design rationale


## 2026-07-22

Validated frontmatter and vocabulary, passed docmgr doctor, and uploaded the five-document external-operation design bundle to /ai/2026/07/22/SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/ttmp/2026/07/22/SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS--durable-external-operation-evidence-ledger/design-doc/01-durable-external-operation-evidence-ledger-design-and-implementation-guide.md — Validated and delivered primary guide
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/ttmp/2026/07/22/SCRAPER-WORKFLOW-V3-EXTERNAL-OPERATIONS--durable-external-operation-evidence-ledger/reference/01-investigation-diary.md — Recorded validation and upload evidence


## 2026-07-22

Commit 2cd6536 stores and publishes the external-operation ledger design ticket


## 2026-07-22

Step 3: defined validated privacy-safe external-operation descriptor spec completion ticket and recorder contracts (commit b637095)

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3/external_operation.go — Generic operation contract and validation
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3/external_operation_test.go — Contract privacy and validation tests


## 2026-07-22

Step 4: added additive external-operation SQLite tables with WAL/FULL/foreign-key startup verification and migration coverage (commit 1542075)

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3sqlite/schema.sql — Durable ledger schema
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3sqlite/store_test.go — Migration and PRAGMA regression coverage


## 2026-07-22

Step 5: implemented durable lease-fenced operation admission and ticket-fenced late immutable completion (commit e061769)

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3sqlite/external_operation.go — Begin/Finish operation evidence authority


## 2026-07-22

Step 6: scoped external-operation recorders to exact trusted host-module aliases (commit 80ef254)

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3runtime/engine.go — Lease-scoped recorder injection
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3runtime/modules.go — Host-only operation descriptor ownership


## 2026-07-22

Step 7: added coherent operation queries, bounded projections, and atomic canonical JSONL/manifest custody export (commit 27efa9e)

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3sqlite/external_operation_query.go — Primary export mechanism


## 2026-07-22

Step 8: reconciled operation allocations and actual/conservative counters with authoritative attempt budget settlement (commit b8857b1)

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/pkg/workflowv3sqlite/budget.go — Operation accounting reconciliation


## 2026-07-22

Step 9: integrated TTC generation and per-request embedding provider spans with the generic Workflow V3 ledger (RAG commits b728e0a, 3bde483, 0147ea2)

### Related Files

- /home/manuel/workspaces/2026-07-13/rag-eval-ttc/rag-evaluation-system/internal/workflowv3ttc/provider.go — Provider adapter


## 2026-07-22

Step 10: contained a fixture sweep timeout after provider instrumentation; reverted uncommitted sweep custody wiring and preserved exact evidence for performance triage


## 2026-07-22

Step 11: verified RAG fixture sweep exports 282 durable provider-operation records into per-cell JSONL/manifest custody


## 2026-07-22

Step 12: completed RAG per-cell operation custody and deterministic failed-cell reductions


## 2026-07-22

Step 13: added race-validated concurrent external-operation admission and completion regression


## 2026-07-22

Step 14: mapped the researchctl public import seam for RAG-owned operation custody export


## 2026-07-22

Step 15: added RAG-owned strict researchctl operation-custody run-export builder

