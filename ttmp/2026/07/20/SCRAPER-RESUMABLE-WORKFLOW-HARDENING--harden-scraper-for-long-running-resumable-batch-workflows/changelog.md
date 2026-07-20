# Changelog

## 2026-07-20

- Initial workspace created


## 2026-07-20

Created the upstream long-running workflow hardening architecture guide and four executable v0.0.4 probes. Confirmed permanent descendant cancellation after retry, sequential in-process MaxWorkers behavior, stale-token completion acceptance, noncumulative heartbeat behavior, and unsafe mixed-precision RFC3339Nano SQLite TEXT comparisons. Defined engine-only phased fixes and explicit sessionstream/runtime-event boundaries.

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/ttmp/2026/07/20/SCRAPER-RESUMABLE-WORKFLOW-HARDENING--harden-scraper-for-long-running-resumable-batch-workflows/design-doc/01-long-running-resumable-workflow-hardening-architecture-and-implementation-guide.md — Primary intern implementation guide
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/ttmp/2026/07/20/SCRAPER-RESUMABLE-WORKFLOW-HARDENING--harden-scraper-for-long-running-resumable-batch-workflows/reference/01-investigation-diary.md — Chronological probe evidence
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/ttmp/2026/07/20/SCRAPER-RESUMABLE-WORKFLOW-HARDENING--harden-scraper-for-long-running-resumable-batch-workflows/scripts/03-probe-stale-lease-completion.go — Lease ownership evidence
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/ttmp/2026/07/20/SCRAPER-RESUMABLE-WORKFLOW-HARDENING--harden-scraper-for-long-running-resumable-batch-workflows/scripts/04-probe-rfc3339nano-text-ordering.go — Timestamp ordering evidence


## 2026-07-20

Validated the long-running workflow hardening guide and diary, then dry-ran and uploaded their reMarkable bundle to /ai/2026/07/20/SCRAPER-RESUMABLE-WORKFLOW-HARDENING/SCRAPER Resumable Workflow Hardening Guide.pdf. Implementation tasks remain open.

### Related Files

- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/ttmp/2026/07/20/SCRAPER-RESUMABLE-WORKFLOW-HARDENING--harden-scraper-for-long-running-resumable-batch-workflows/design-doc/01-long-running-resumable-workflow-hardening-architecture-and-implementation-guide.md — Validated and delivered primary guide
- /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper/ttmp/2026/07/20/SCRAPER-RESUMABLE-WORKFLOW-HARDENING--harden-scraper-for-long-running-resumable-batch-workflows/reference/01-investigation-diary.md — Probe and validation evidence

