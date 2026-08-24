<!-- Source: https://parc.yolo.scapegoat.dev/note/research/kb/projects/scraper -->
<!-- Retrieved: 2026-07-21 -->

[Terminology and agent guide](https://parc.yolo.scapegoat.dev/AGENTS.md)

modifiedJul 20, 2026

tags

aliasesscraper, Scraper MOC, scraper workflow runtime, durable extraction workflows

createdJul 15, 2026

repo/home/manuel/code/wesen/go-go-golems/scraper

statusactive

typeknowledge-base

## Scraper — Durable Workflows, Research Extraction, and Evidence Pipelines

The `scraper` work evolved from browser/LLM-assisted extraction into a reusable workflow runtime for research and document-processing jobs. It provides durable run state, workflow events, structured extraction contracts, provider/model policies, OCR pipelines, persistence, and inspection surfaces. The central design goal is to make long-running research jobs restartable and auditable rather than hiding them inside one fragile prompt or command.

≡ Summary

- **Workflow runtime:** jobs, steps, retries, events, persistence, and recovery are explicit.
- **Extraction contracts:** LLM/OCR outputs are validated against target schemas and provenance.
- **Research pipeline:** source acquisition, extraction, repair, indexing, and human review remain inspectable stages.

## Architecture

```
flowchart TD
    INPUT[Web pages, PDFs, books, research targets] --> WORKFLOW[Workflow definition]
    WORKFLOW --> RUN[Durable run state]
    RUN --> STEPS[Fetch / OCR / extract / validate / repair]
    STEPS --> EVENTS[Runtime events and logs]
    STEPS --> ARTIFACTS[Raw and derived artifacts]
    ARTIFACTS --> REVIEW[Human review and correction]
    REVIEW --> IMPORT[Database / RAG / report]
    RUN --> RESUME[Resume and retry]
```

The runtime is more important than any individual scraper. A workflow must expose its inputs, state transitions, target contracts, provider configuration, failure state, and artifacts. This is why the same architecture can support therapist research, Book OCR, RAG ingestion, and other extraction programs.

## Capability areas

### Workflow runtime and events

- [Scraper Workflow API: Building a Public Reusable Durable Workflow Runtime](https://parc.yolo.scapegoat.dev/note/projects/2026/05/25/article-scraper-workflow-api-building-a-public-reusable-durable-workflow-runtime) — public workflow API and durability model.
- [Scraper Runtime Events Session Report](https://parc.yolo.scapegoat.dev/note/projects/2026/03/25/proj-scraper-runtime-events-session-report) — runtime event design.
- [Sessionstream Runtime Events in Scraper](https://parc.yolo.scapegoat.dev/note/projects/2026/05/22/article-sessionstream-runtime-events-in-scraper) — sessionstream integration.
- [Devctl Trace Profiles for Pinocchio and CoinVault](https://parc.yolo.scapegoat.dev/note/projects/2026/05/07/article-devctl-trace-profiles-pinocchio-and-coinvault) — adjacent trace/profile tooling.
- [RAG Evaluation Pipeline Architecture — Getting Started](https://parc.yolo.scapegoat.dev/note/research/kb/on-ramp/rag-evaluation-pipeline-architecture) — downstream evaluation orientation.

### Research and extraction applications

- [Claude Agent SDK: Teaching an AI to Write Web Scrapers](https://parc.yolo.scapegoat.dev/note/projects/2026/03/22/proj-claude-agent-sdk-teaching-an-ai-to-write-web-scrapers) — original agent-assisted scraping direction.
- [Providence Therapist Search: End-to-End Research System and LLM Extraction Lab](https://parc.yolo.scapegoat.dev/note/projects/2026/05/15/article-providence-therapist-search-end-to-end-research-system-and-llm-extraction-lab) — research extraction application.
- [Providence Therapist Search: A Retro Monochrome Research Dashboard](https://parc.yolo.scapegoat.dev/note/projects/2026/05/14/article-providence-therapist-search-a-retro-monochrome-research-dashboard) — inspection UI.
- [Book OCR Quality Lab: Prompt Optimization, SQLite Log Filtering, and Experiment Provenance](https://parc.yolo.scapegoat.dev/note/projects/2026/05/24/article-book-ocr-quality-lab-baseline-runs-sqlite-log-filtering-and-experiment-provenance) — OCR measurement.
- [Building Book OCR on the Scraper Job System: Workflow Runtime Deep Dive](https://parc.yolo.scapegoat.dev/note/projects/2026/05/24/article-building-book-ocr-on-scraper-job-system-workflow-runtime-deep-dive) — OCR workflow integration.
- [Extracting Book OCR from Scraper: Workflow Runtime Boundaries and External OCR Pipelines](https://parc.yolo.scapegoat.dev/note/projects/2026/05/24/article-extracting-book-ocr-from-scraper-workflow-runtime-and-external-ocr-pipelines) — extraction decomposition.
- [Book OCR Project Report - Structured Workflow Runtime and Manual PDF Repair](https://parc.yolo.scapegoat.dev/note/projects/2026/05/26/article-book-ocr-project-report-structured-workflow-runtime-and-manual-pdf-repair) — production repair loop.
- [Structured Book OCR - Target Page Contracts Workflow Runtime and Production Hardening](https://parc.yolo.scapegoat.dev/note/projects/2026/05/26/article-structured-book-ocr-target-page-contracts-workflow-runtime-and-production-hardening) — target contracts and hardening.
- [Book OCR Productization: Plugin Seams, Profile Policy, and the Road to v0.1.0](https://parc.yolo.scapegoat.dev/note/projects/2026/07/03/article-book-ocr-productization-plugin-seams-profile-policy-and-the-road-to-v0-1-0) — productization boundaries.

### RAG and evidence consumers

- [RAG Evaluation System — Corpus, Retrieval, and Workflow Evaluation](https://parc.yolo.scapegoat.dev/note/research/kb/projects/rag-evaluation-system) — retrieval and relevance evaluation.
- [goja-text — Go-Backed Text, Markdown, and Source-Preserving Pipelines](https://parc.yolo.scapegoat.dev/note/research/kb/projects/goja-text) — source-preserving text/chunking.
- [goja-bleve — Native Vector and Hybrid Search for JavaScript RAG](https://parc.yolo.scapegoat.dev/note/research/kb/projects/goja-bleve) — search and vector indexing.
- [researchctl — Experiment Management Tool and DSL](https://parc.yolo.scapegoat.dev/note/research/kb/projects/researchctl) — explicit experiment/evidence workflow.
- [go-minitrace — Transcript Analysis and Evidence Workbench](https://parc.yolo.scapegoat.dev/note/research/kb/projects/go-minitrace) — transcript and run evidence analysis.

## Recommended reading path

1. Read the workflow API and runtime-events reports.
2. Read one extraction application, preferably Book OCR or therapist research.
3. Read the target-contract and manual-repair reports for correctness boundaries.
4. Follow RAG evaluation and source-preserving text links for downstream use.
5. Use the evidence MOCs when turning workflow runs into durable conclusions.

## Working rules

- Define workflow state and target contracts before adding provider calls.
- Make operations idempotent, resumable, and observable.
- Retain raw inputs and intermediate artifacts beside derived outputs.
- Separate extraction from validation and human correction.
- Record provider/model/profile identity with every run.
- Keep live LLM calls out of unit tests; use fixtures and replayable artifacts.
- Treat manual repair as an explicit workflow stage, not an invisible edit.

## Repository map

Repository: `/home/manuel/code/wesen/go-go-golems/scraper`

| Concern | Location |
| --- | --- |
| Workflow runtime | workflow/job packages |
| Browser and source acquisition | scraper adapters |
| LLM/OCR extraction | extraction/provider packages |
| Runtime events | event/trace packages |
| Persistence | database and artifact packages |
| CLI and service surfaces | command/server packages |
