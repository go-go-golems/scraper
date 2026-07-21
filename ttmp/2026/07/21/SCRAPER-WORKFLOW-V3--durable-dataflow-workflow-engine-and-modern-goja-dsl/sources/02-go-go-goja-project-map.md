<!-- Source: https://parc.yolo.scapegoat.dev/note/research/kb/projects/go-go-goja -->
<!-- Retrieved: 2026-07-21 -->

[Terminology and agent guide](https://parc.yolo.scapegoat.dev/AGENTS.md)

modifiedJul 20, 2026

tags

aliasesgo-go-goja, go-go-goja MOC, xgoja, Goja host runtime

createdJul 15, 2026

repo/home/manuel/code/wesen/go-go-golems/go-go-goja

statusactive

typeknowledge-base

## go-go-goja — Go-Hosted JavaScript Runtimes and Generated Applications

`go-go-goja` is the Go-side runtime and tooling ecosystem for embedding JavaScript with goja. It combines explicit runtime ownership, native modules, fluent Go-backed objects, jsverbs/Glazed command generation, xgoja compile-time provider composition, HTTP serving, storage, async boundaries, and reusable application hosts. The central design problem is not simply “run JavaScript”; it is to make JavaScript capabilities composable while keeping Go in control of lifecycle, permissions, context, scheduling, and generated-binary composition.

≡ Summary

- **Runtime:** create and own goja runtimes with explicit context, session, thread, and async semantics.
- **Modules and DSLs:** expose typed Go capabilities as JavaScript modules and fluent builders.
- **xgoja:** compose providers and source graphs at build time into focused generated hosts rather than one ambient mega-runtime.

## Architecture map

```
flowchart TD
    APP[Go application] --> RUNTIME[Runtime creation and ownership]
    RUNTIME --> VM[goja VM]
    VM --> MODULES[Native modules and Go-backed objects]
    MODULES --> DSL[Fluent DSLs and jsverbs]
    DSL --> COMMANDS[Glazed commands / HTTP services / UI hosts]
    PROVIDERS[Compile-time providers] --> XGOJA[xgoja RuntimePlan]
    XGOJA --> HOST[Generated application binary]
    RUNTIME --> CONTEXT[Request, session, cancellation, async context]
    CONTEXT --> MODULES
```

The runtime is the boundary that makes the rest safe. A module should not invent its own lifecycle or reach into process-global state; it should receive the runtime/context capabilities its host intentionally provides. xgoja then makes the module set and application profile explicit at build time.

## Capability areas

### Runtime ownership and execution

- [go-go-goja Runtime System: Creation, Context, Scheduling, Bindings, and Modules](https://parc.yolo.scapegoat.dev/note/projects/2026/05/23/article-go-go-goja-runtime-system-creation-context-scheduling-and-modules) — runtime construction, scheduling, and module ownership.
- [go-go-goja Context Management: Runtime, Request, and Async Call Context](https://parc.yolo.scapegoat.dev/note/projects/2026/05/15/article-go-go-goja-context-management-runtime-request-and-async-call-context) — request context and asynchronous work.
- [goja Execution Model — Sessions, Thread Safety, and Async — How We Do It](https://parc.yolo.scapegoat.dev/note/research/kb/tribal/goja-execution-model) — sessions, thread safety, and async invariants.
- [go-go-goja REPL API - Profiles, IIFE Rewriting, and AST-Driven Session Semantics](https://parc.yolo.scapegoat.dev/note/projects/2026/04/03/proj-go-go-goja-repl-api-profiles-iife-rewriting-and-ast-driven-session-semantics) — interactive runtime semantics.
- [Goja REPL Hardening](https://parc.yolo.scapegoat.dev/note/projects/2026/04/08/proj-goja-repl-hardening) — hardening the interactive boundary.

### Native modules and Go-backed JavaScript

- [goja: Embedding a JavaScript Interpreter in Go — How We Do It](https://parc.yolo.scapegoat.dev/note/research/kb/tribal/goja-embedding-in-go) — baseline embedding pattern.
- [Designing DSLs with go-go-goja - Go-Backed JavaScript APIs](https://parc.yolo.scapegoat.dev/note/projects/2026/06/22/article-designing-dsls-with-go-go-goja-go-backed-javascript-apis) — fluent API design.
- [Goja Fluent-Builder DSLs: Designing Typed Composable Grammars in Go for JavaScript](https://parc.yolo.scapegoat.dev/note/projects/2026/07/05/article-goja-fluent-builder-dsls-designing-typed-composable-grammars-in-go-for-javascript) — typed composable builder grammar.
- [go-go-goja Protobuf Builders: Goja-Native Fluent Proto Construction](https://parc.yolo.scapegoat.dev/note/projects/2026/06/12/article-go-go-goja-protobuf-builders-goja-native-fluent-proto-construction) — structured native objects.
- [go-go-goja: Adding Transaction Support to the Goja DB Module](https://parc.yolo.scapegoat.dev/note/projects/2026/06/05/article-go-go-goja-adding-transaction-support-to-the-goja-db-module) — stateful module boundaries.
- [Report: Go-Go-Goja EventEmitter Implementation](https://parc.yolo.scapegoat.dev/note/projects/2026/04/26/article-report-go-go-goja-eventemitter-implementation) — event delivery.
- [Report: Go-Go-Goja fswatch Implementation](https://parc.yolo.scapegoat.dev/note/projects/2026/04/26/article-report-go-go-goja-fswatch-implementation) — host-side filesystem events.

### xgoja and generated hosts

- [xgoja: Compile-Time Goja Module Composition and jsverbs Mounting](https://parc.yolo.scapegoat.dev/note/projects/2026/05/22/article-xgoja-compile-time-goja-module-composition-and-jsverbs-mounting) — compile-time composition model.
- [xgoja: Generated Goja Applications, Provider Architecture, and Runtime Profiles](https://parc.yolo.scapegoat.dev/note/projects/2026/05/24/article-xgoja-generated-goja-applications-provider-architecture-and-runtime-profiles) — provider architecture and profiles.
- [New XGoja: Source Graphs, Provider Plans, and the V2 Runtime Compiler](https://parc.yolo.scapegoat.dev/note/projects/2026/06/12/article-new-xgoja-source-graphs-provider-plans-and-the-v2-runtime-compiler) — source graphs and provider plans.
- [xgoja v2 RuntimePlan Hard Cutover — A Technical Deep Dive](https://parc.yolo.scapegoat.dev/note/projects/2026/06/13/article-xgoja-v2-runtimeplan-hard-cutover-technical-deep-dive) — the hard-cut runtime-plan transition.
- [Playbook: Building go-go-goja xgoja Provider Packages](https://parc.yolo.scapegoat.dev/note/projects/2026/05/27/article-playbook-building-go-go-goja-xgoja-provider-packages) — provider package implementation workflow.
- [XGoja Provider Support - Third-Party Package Rollout](https://parc.yolo.scapegoat.dev/note/projects/2026/05/24/proj-xgoja-provider-support-third-party-package-rollout) — rolling providers into existing packages.

### Applications and integrations

- [go-go-goja jsverbs](https://parc.yolo.scapegoat.dev/note/projects/2026/03/16/proj-go-go-goja-jsverbs-javascript-to-glazed-commands) — JavaScript-defined structured CLI commands.
- [Building a Query Tool with xgoja: Jsverbs, Embedded Modules, and the Contracts That Are Not Written Down](https://parc.yolo.scapegoat.dev/note/projects/2026/06/03/article-xgoja-building-a-query-tool-with-jsverbs-and-embedded-modules) — a generated query application.
- [go-go-goja HTTP Serve Support for xgoja Generated Verbs](https://parc.yolo.scapegoat.dev/note/projects/2026/06/04/article-go-go-goja-http-serve-support-for-xgoja-generated-verbs) — HTTP serving from generated verbs.
- [xgoja: HTTP Serve, Hot Reload, and Runtime Service Architecture](https://parc.yolo.scapegoat.dev/note/projects/2026/06/08/article-xgoja-http-serve-hot-reload-and-runtime-service-architecture) — service lifecycle and reloads.
- [go-go-goja Express Auth: From Planned Routes to Generated Host Auth](https://parc.yolo.scapegoat.dev/note/projects/2026/06/14/article-go-go-goja-express-auth-from-planned-routes-to-generated-host-auth) — route and host authentication.
- [go-go-goja Programmatic Auth After Rate Limiting: Deep Dive](https://parc.yolo.scapegoat.dev/note/projects/2026/06/20/article-go-go-goja-programmatic-auth-after-rate-limiting-deep-dive) — auth ordering and route policy.
- [xgoja: Build Environments and Jsverb Command Design for Vector RAG Tools](https://parc.yolo.scapegoat.dev/note/projects/2026/06/06/xgoja-env/article-xgoja-build-environments-and-jsverb-command-design-for-vector-rag-tools-gpt-5-5-medium) — RAG tool host design.

## Recommended reading path

1. Start with the runtime-system report and the goja execution-model tribal entry.
2. Read the native-module and fluent-builder notes to understand the JavaScript API boundary.
3. Read xgoja provider and RuntimePlan reports to understand generated binaries.
4. Read one integration note such as jsverbs, HTTP serve, or Express Auth.
5. Use the application reports for concrete constraints and failure modes.

## Working rules

- Keep goja runtime ownership explicit; do not treat a runtime as a globally shareable object.
- Marshal work back to the owning runtime thread when callbacks or async results cross Go boundaries.
- Keep module capabilities narrow and host-provided.
- Use plain JSON-shaped values at stable boundaries when that improves interoperability.
- Prefer wrapper-first APIs over leaking internal Go objects directly.
- Compose providers at build time with xgoja rather than exposing every module to every binary.
- Treat generated hosts as products with their own configuration, help, assets, and release behavior.

## Related knowledge

- [goja-text — Go-Backed Text, Markdown, and Source-Preserving Pipelines](https://parc.yolo.scapegoat.dev/note/research/kb/projects/goja-text) and [goja-bleve — Native Vector and Hybrid Search for JavaScript RAG](https://parc.yolo.scapegoat.dev/note/research/kb/projects/goja-bleve) — sibling native modules built on the same host/runtime ecosystem.
- [researchctl — Experiment Management Tool and DSL](https://parc.yolo.scapegoat.dev/note/research/kb/projects/researchctl) — another Goja-backed DSL and generated-host use case.
- [Data-Only vs Host-Access Module Split — How We Do It](https://parc.yolo.scapegoat.dev/note/research/kb/tribal/data-only-vs-host-access-module-split) — capability separation pattern.
- [DSL → Normalized Config → Compiled Plan — How We Do It](https://parc.yolo.scapegoat.dev/note/research/kb/tribal/dsl-normalized-config-compiled-plan) — normalize and compile declarative input before execution.

## Repository map

Repository: `/home/manuel/code/wesen/go-go-golems/go-go-goja`

| Concern | Location |
| --- | --- |
| Runtime and context | runtime packages |
| Native modules | module/provider packages |
| xgoja composition | xgoja/provider packages |
| jsverbs and Glazed | jsverbs packages |
| Generated examples | `examples/` |
| Runtime and provider tests | package tests and integration fixtures |
