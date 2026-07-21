<!-- Source: https://parc.yolo.scapegoat.dev/note/research/kb/projects/widget-dsl -->
<!-- Retrieved: 2026-07-21 -->

[Terminology and agent guide](https://parc.yolo.scapegoat.dev/AGENTS.md)

modifiedJul 20, 2026

tags

aliaseswidget DSL, Widget DSL MOC, Widget IR, server-driven widget DSL

createdJul 15, 2026

repo/home/manuel/code/wesen/go-go-golems/go-go-os-frontend

statusactive

typeknowledge-base

## Widget DSL — Intent-Level UI Authoring, IR, and React Targets

The Widget DSL work defines a layered way to author UI intent in JavaScript or Go, normalize it into a widget intermediate representation, and render that representation through React or another target. The system grew from RAG evaluation and generated-host experiments into a broader design pattern: application authors describe semantic widgets and interaction intent, while the renderer owns layout, styling, state wiring, and target-specific details.

≡ Summary

- **Intent layer:** authors express semantic UI structures, not CSS coordinates or renderer internals.
- **IR layer:** typed widget instances, slots, actions, and data contracts provide a stable boundary.
- **Target layer:** React, Storybook, generated hosts, and application presets consume the same structured representation.

## Architecture

```
flowchart LR
    AUTHOR[JavaScript / Go DSL] --> NORMALIZE[Normalize and validate]
    NORMALIZE --> IR[Widget IR]
    IR --> PRESETS[Recipes, presets, design-system policy]
    PRESETS --> REACT[React renderer]
    PRESETS --> STORY[Storybook / visual review]
    IR --> HOST[xgoja or server-driven host]
    DATA[Application data] --> IR
```

The important boundary is semantic versus presentational. A widget DSL should say “render a result card with evidence, actions, and an expandable detail slot,” not “place a 320px div at x=40.” The IR makes the intent inspectable, serializable, testable, and portable across applications.

## Capability areas

### Foundations and IR

- [Building a Goja UI DSL from Scratch: Widget IR to xgoja](https://parc.yolo.scapegoat.dev/note/projects/2026/06/05/article-building-a-goja-ui-dsl-from-scratch-widget-ir-to-xgoja) — initial DSL and IR boundary.
- [Widget IR: Building a Data-First React Rendering Pipeline for RAG Evaluation](https://parc.yolo.scapegoat.dev/note/projects/2026/06/07/article-widget-ir-building-a-data-first-react-rendering-pipeline-for-rag-evaluation) — data-first rendering.
- [Goja Fluent-Builder DSLs: Designing Typed Composable Grammars in Go for JavaScript](https://parc.yolo.scapegoat.dev/note/projects/2026/07/05/article-goja-fluent-builder-dsls-designing-typed-composable-grammars-in-go-for-javascript) — typed builder grammar.
- [Designing DSLs with go-go-goja - Go-Backed JavaScript APIs](https://parc.yolo.scapegoat.dev/note/projects/2026/06/22/article-designing-dsls-with-go-go-goja-go-backed-javascript-apis) — Go-backed JavaScript DSLs.

### Recipes, versions, and migration

- [From Boilerplate to Recipes: Building Higher-Level Widgets on Top of Widget IR — gpt5.5 - thinking medium](https://parc.yolo.scapegoat.dev/note/projects/2026/06/06/rag-eval-dsl/article-from-boilerplate-to-recipes-building-higher-level-widgets-on-top-of-widget-ir-gpt5-5-thinking-medium) — higher-level recipes.
- [From Boilerplate to Recipes: Building Higher-Level Widgets on Top of Widget IR](https://parc.yolo.scapegoat.dev/note/projects/2026/06/06/rag-eval-dsl/article-semantic-recipes-on-top-of-widget-ir) — semantic recipe design.
- [Widget DSL Grammar: Designing an Intent-Level UI Authoring Layer for a Widget IR System](https://parc.yolo.scapegoat.dev/note/projects/2026/07/05/article-widget-dsl-grammar-designing-an-intent-level-ui-authoring-layer-for-a-widget-ir-system) — grammar design.
- [Widget DSL V2 Cutover: Typed Fluent Builders for Server-Driven Widget IR](https://parc.yolo.scapegoat.dev/note/projects/2026/07/05/article-widget-dsl-v2-cutover-typed-fluent-builders-for-server-driven-widget-ir) — typed-builder migration.
- [Widget DSL v3: From Split Modules to a Real Host Migration](https://parc.yolo.scapegoat.dev/note/projects/2026/07/08/article-widget-dsl-v3-from-split-modules-to-a-real-host-migration) — host migration.

### Applications and product surfaces

- [CRM Widget Kit: Engine, Contract, and Preset over a Widget IR](https://parc.yolo.scapegoat.dev/note/projects/2026/07/07/proj-crm-widget-kit-engine-contract-and-preset-over-a-widget-ir) — reusable engine/contract/preset architecture.
- [Doodle Scheduling Site: SQLite and the rag Widget DSL on xgoja](https://parc.yolo.scapegoat.dev/note/projects/2026/07/07/proj-doodle-scheduling-site-sqlite-and-the-rag-widget-dsl-on-xgoja) — generated application.
- [Doodle on xgoja and Widget DSL v3: A SQLite Scheduling Site Deep Dive](https://parc.yolo.scapegoat.dev/note/projects/2026/07/09/article-doodle-on-xgoja-and-widget-dsl-v3-a-sqlite-scheduling-site-deep-dive) — integrated runtime.
- [Doodle Project Report: From xgoja JavaScript to Rendered Widget UI](https://parc.yolo.scapegoat.dev/note/projects/2026/07/09/article-doodle-project-report-from-xgoja-javascript-to-rendered-widget-ui) — end-to-end rendering.
- [WidgetRenderer Standalone Site: Goja-Authored, React-Rendered UI](https://parc.yolo.scapegoat.dev/note/projects/2026/06/04/article-widgetrenderer-standalone-site-goja-authored-react-rendered-ui) — standalone renderer.
- [RAG React Design System: From Prototype Dashboard to Structured Design System](https://parc.yolo.scapegoat.dev/note/projects/2026/06/02/article-rag-react-design-system-from-prototype-dashboard-to-structured-design-system) — design-system application.

### Design-system and IR neighbors

- [DMETA Design System Factory: From Semantic Archetypes to Validated IR](https://parc.yolo.scapegoat.dev/note/projects/2026/05/19/article-dmeta-design-system-factory-from-semantic-archetypes-to-validated-ir) — semantic design-system IR.
- [DMETA as a Design System Compiler: Layered IRs, Interaction Representations, and MetaDesignSystems](https://parc.yolo.scapegoat.dev/note/projects/2026/05/24/article-dmeta-as-a-design-system-compiler-layered-irs-and-metadesignsystems) — compiler pipeline.
- [Presentation-Based User Interfaces: AITR-794, CLIM, and the DMETA Implementation Model](https://parc.yolo.scapegoat.dev/note/projects/2026/05/20/article-presentation-based-user-interfaces-aitr-794-and-dmeta-implementation-guide) — presentation-based UI model.
- [DMETA Design System Compiler Pipeline — How We Do It](https://parc.yolo.scapegoat.dev/note/research/kb/tribal/dmeta-design-system-compiler-pipeline) — reusable compiler pattern.
- [Typed Widget Instance Streaming for Chat Overlays — How We Do It](https://parc.yolo.scapegoat.dev/note/research/kb/tribal/typed-widget-instance-streaming-for-chat-overlays) — typed streaming widget instances.

## Working rules

- Keep author intent separate from renderer implementation.
- Normalize and validate before rendering.
- Prefer typed widget instances and explicit slots over arbitrary renderer escape hatches.
- Make actions and data contracts part of the IR, not hidden inside React callbacks.
- Keep styling and design-system policy in the target layer or preset layer.
- Treat DSL versions and host migrations as explicit compatibility events.
- Test semantics, serialized IR, and visual output separately.

## Related project maps

- [RAG Evaluation System — Corpus, Retrieval, and Workflow Evaluation](https://parc.yolo.scapegoat.dev/note/research/kb/projects/rag-evaluation-system) — major origin and consumer of the widget work.
- [go-go-goja — Go-Hosted JavaScript Runtimes and Generated Applications](https://parc.yolo.scapegoat.dev/note/research/kb/projects/go-go-goja) — JavaScript runtime and xgoja host.
- [Geppetto — Go LLM Runtime, Engines, Profiles, and Sessions](https://parc.yolo.scapegoat.dev/note/research/kb/projects/geppetto) and [Pinocchio — CLI Chat Applications, TUI, RPC, and Session Hosts](https://parc.yolo.scapegoat.dev/note/research/kb/projects/pinocchio) — streamed chat/application consumers.
- [Glazed — Structured Go CLI Applications and Help Systems](https://parc.yolo.scapegoat.dev/note/research/kb/projects/glazed) — structured command/help surfaces using similar schema-first ideas.

## Repository map

Primary repositories: `/home/manuel/code/wesen/go-go-os-frontend`, `/home/manuel/code/wesen/go-go-golems/dmeta`, and related xgoja/widget workspaces.

| Concern | Location |
| --- | --- |
| Widget packages | frontend `packages/` and widget-kit packages |
| DSL and builders | JavaScript/Go DSL packages |
| IR contracts | widget model/type packages |
| React targets | renderer and application packages |
| Stories and review | Storybook and visual-diff fixtures |
