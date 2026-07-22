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

