<!-- Source: https://parc.yolo.scapegoat.dev/note/research/kb/tribal/dsl-normalized-config-compiled-plan -->
<!-- Retrieved: 2026-07-21 -->

[Terminology and agent guide](https://parc.yolo.scapegoat.dev/AGENTS.md)

tags

aliasesDSL to compiled plan, normalize then compile, declarative pipeline, three-stage pipeline

statusactive

typeknowledge-base

## DSL → Normalized Config → Compiled Plan — How We Do It

≡ Summary

We never execute the user's input directly. Instead, we pass it through a three-stage pipeline: parse the declarative DSL into raw types, normalize into a validated and defaulted intermediate config, then compile into an executable plan. This separation means validation errors surface before execution, the runtime never sees half-specified inputs, and multiple execution modes (CLI, server, test) share the same planning logic. Two media projects converged on this independently: Screencast Studio and Almanach Studio.

## The pattern

Our media/recording systems follow a strict three-stage pipeline:

```
User DSL  →  Normalize  →  Compile  →  Execute
(raw)        (validated     (concrete    (runtime
              + defaulted)   plan)        action)
```

Each stage has a distinct job:

1. **DSL (raw input).** The user writes a declarative description of what they want: video sources, audio sources, destinations, codec preferences, capture settings. This is a human-facing format — terse, opinionated, with sensible defaults omitted. It is not executable.
2. **Normalized config (validated + defaulted).** The normalizer takes the raw DSL, fills defaults, canonicalizes names, checks template existence, resolves references, and accumulates warnings. The output is structurally complete — every field has a value, every reference resolves. If the DSL is invalid, the normalizer returns errors before any execution happens.
3. **Compiled plan (concrete execution recipe).** The compiler takes the normalized config and produces concrete jobs: exact ffmpeg command lines, exact GStreamer pipeline descriptions, exact output paths, session IDs, and resource requirements. The plan is executable — the runtime can hand it directly to the execution engine.

The key invariant: **the runtime never sees raw DSL.** It only sees compiled plans. This means the runtime can focus on execution concerns (process supervision, cancellation, telemetry) without re-implementing validation logic.

## Why we do it this way

**Validation before execution.** When the user writes a bad DSL (missing source, circular reference, unsupported codec), the normalizer catches it immediately. The alternative — passing bad input directly to ffmpeg or GStreamer — produces obscure error messages from the execution engine that the user can't map back to their DSL.

**Defaults don't leak into the DSL.** The DSL is terse because defaults live in the normalizer, not in the user's file. If we change the default codec from H.264 to H.265, we change the normalizer — not every DSL file the user ever wrote.

**Multiple execution modes share the same plan.** Screencast Studio has both a `record` CLI mode (one-shot) and a `serve` mode (persistent web server). Both compile the same DSL → config → plan pipeline. The plan is the shared abstraction; the execution mode is a wrapper.

**Warnings accumulate without blocking.** The normalizer can emit warnings ("codec not specified, using H.264 default") alongside the valid config. This lets the user know what decisions were made on their behalf without stopping the pipeline. The compiled plan carries these warnings forward to the runtime, which can surface them in the UI.

**Testing becomes compositional.** You can test the normalizer without starting ffmpeg. You can test the compiler without starting ffmpeg. You can test the runtime by providing a pre-compiled plan. Each stage is testable in isolation because it has a clear input type and output type.

Alternatives we considered:

- **Direct execution (shelling out).** Simpler to build, but every edge case surfaces as an obscure ffmpeg error. Hard to test. Hard to add preview or telemetry later.
- **Single-stage config.** Merging DSL + defaults + compilation into one step. Works for small projects, but loses the ability to give the user "your setup is structurally valid, but here are warnings" feedback.
- **Code generation.** Generating Go source from the DSL and compiling it. Overkill — the DSL describes recording plans, not programs.

## Where it lives

| Repo | Path | Use |
| --- | --- | --- |
| `2026-04-09--screencast-studio` | `pkg/dsl/types.go`, `pkg/dsl/normalize.go` | DSL types, normalization, plan compilation |
| `2026-04-09--screencast-studio` | `pkg/app/application.go` | Application facade: NormalizeDSL, CompileDSL, RecordPlan |
| `2026-04-09--screencast-studio` | `internal/web/session_manager.go` | Runtime: compile DSL into plan, start managed recording |
| `almanach` (extract workspace) | render service | GStreamer pipeline DSL → normalized config → compiled plan |

### Related PARC project reports

- [Screencast Studio](https://parc.yolo.scapegoat.dev/note/projects/2026/04/10/proj-screencast-studio-architecture-and-runtime-deep-dive) — canonical instance: ffmpeg-backed recording with DSL → EffectiveConfig → CompiledPlan
- [Screencast Studio - GStreamer Migration and Media Runtime Intern Guide](https://parc.yolo.scapegoat.dev/note/projects/2026/04/13/proj-screencast-studio-gstreamer-migration-and-media-runtime-intern-guide) — GStreamer migration: same three-stage pipeline, different execution engine

## Common mistakes

1. **Validating in the compiler instead of the normalizer.** The temptation is to add validation checks ("does this output path exist?") during compilation. But if the compiler validates, then a CLI `compile` command can fail with a runtime-style error instead of a clean "your DSL has these problems" message. Keep validation in the normalizer; keep the compiler as a pure function from config to plan.
2. **Letting defaults drift between normalizer and documentation.** If the normalizer defaults to 30fps and the documentation says 24fps, the user's expectation doesn't match reality. This is especially pernicious because the DSL deliberately omits defaulted fields, so the user can't see what the system chose for them. The fix: the normalizer emits warnings when it fills non-obvious defaults, and the compiled plan carries those warnings to the runtime.
3. **Compiling directly to command-line strings instead of structured plans.** Early prototypes may compile straight to `ffmpeg -i ...` strings. This works until you need to inspect the plan ("what codec did we choose?"), serialize it for logging, or feed it to a different execution engine (e.g., GStreamer instead of ffmpeg). Always compile to a structured plan type first, then let the execution layer turn the plan into command lines or API calls.
4. **Forgetting that the compiled plan is immutable once created.** The plan represents a snapshot of the user's intent at compile time. If the runtime mutates the plan (e.g., changing the output path after compilation), then the plan no longer represents what the user asked for. If you need runtime-specific values (session ID, timestamp), inject them during compilation — don't patch the plan later.
5. **Not carrying warnings through to the execution layer.** The normalizer accumulates warnings. The compiler puts them in the plan. But if the runtime doesn't surface them, the user never sees them. A warning like "audio source references a device that may not be available" is useless if it stays in the plan object and never reaches the web UI or CLI output.
6. **Adding execution policy to the DSL.** The DSL describes *what* to record, not *how* to manage the recording process. Questions like "should I stop recording after 30 minutes?" or "what should happen if the disk fills up?" are runtime policy, not DSL content. If you put policy in the DSL, every new runtime mode needs a new DSL field, and the DSL becomes a kitchen sink instead of a declarative description.

## Variations

- **ffmpeg execution** (Screencast Studio). The compiled plan contains `VideoJob` and `AudioMixJob` structs. The runtime turns each job into an ffmpeg subprocess with specific flags. The plan is engine-agnostic — the same plan could drive GStreamer or a different engine.
- **GStreamer execution** (Almanach Studio). The same three-stage pipeline, but the compiled plan produces a GStreamer pipeline description instead of ffmpeg command lines. The DSL and normalizer are structurally the same; only the compilation target changes.
- **Glazed command compilation** (go-go-goja jsverbs). The jsverbs scanner performs a similar pipeline: scan JS source for metadata (DSL) → validate and normalize into Go structures (normalized config) → compile into Glazed command definitions (compiled plan). The execution engine is Cobra+Glazed instead of ffmpeg/GStreamer. See [goja: Embedding a JavaScript Interpreter in Go — How We Do It](https://parc.yolo.scapegoat.dev/note/research/kb/tribal/goja-embedding-in-go) for the jsverbs variation.
- **Design system compilation** (DMETA). The DMETA compiler extends the three-stage pipeline to four layers: `Semantic IR → Interaction IR → Web MetaDesignSystem → React target`. Each layer has its own types, validation, and lowering pass. The Factory freezes module policy like a compiled plan, and each lowering pass is a compilation step. See [DMETA Design System Compiler Pipeline — How We Do It](https://parc.yolo.scapegoat.dev/note/research/kb/tribal/dmeta-design-system-compiler-pipeline) for the full architecture.
