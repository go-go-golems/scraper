# Changelog

## 2026-03-23

Created a new design ticket for a site-aware JavaScript CLI runner in `scraper`, studied `go-go-goja` jsverbs, compared it with the current scheduler-facing site runtime, and wrote a detailed implementation guide recommending separate site `verbs/` and workflow `scripts/` entrypoints over direct scanning of the existing op scripts.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/pkg/doc/08-jsverbs-example-overview.md — Primary jsverbs overview the ticket was anchored on
- /home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/pkg/jsverbs/scan.go — Scanner API and `ScanFS(...)` behavior used to ground the proposed site verb loader
- /home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/pkg/jsverbs/binding.go — Binding modes and shared binding-plan logic used in the design
- /home/manuel/workspaces/2026-03-23/js-scraper/go-go-goja/pkg/jsverbs/runtime.go — Runtime overlay and invocation path used in the design
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/registry/registry.go — Current site registration seams and unused `RegisterCLI` escape hatch
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/js/runtime/executor.go — Current scheduler-facing JS runtime contract that should remain separate from site verbs
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/hackernews/scripts/extract_frontpage.js — Concrete example of why current op scripts should not be scanned directly as jsverbs
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-SITE-JSVERBS--site-javascript-runner-cli-based-on-jsverbs/design-doc/01-site-javascript-cli-runner-with-jsverbs-design-and-implementation-guide.md — Primary design deliverable
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/03/23/SCRAPER-SITE-JSVERBS--site-javascript-runner-cli-based-on-jsverbs/reference/01-investigation-diary.md — Chronological research diary
