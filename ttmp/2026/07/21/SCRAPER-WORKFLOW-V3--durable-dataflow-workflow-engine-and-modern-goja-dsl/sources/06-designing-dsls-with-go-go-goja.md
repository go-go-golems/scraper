<!-- Source: https://parc.yolo.scapegoat.dev/note/projects/2026/06/22/article-designing-dsls-with-go-go-goja-go-backed-javascript-apis -->
<!-- Retrieved: 2026-07-21 -->

[Terminology and agent guide](https://parc.yolo.scapegoat.dev/AGENTS.md)

modifiedJul 20, 2026

tags

aliasesDesigning DSLs with go-go-goja, Go-backed JavaScript DSLs, go-go-goja DSL design guide

createdJun 22, 2026

repo/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja

source/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/doc/34-designing-dsls-with-go-go-goja.md

statusactive

typearticle

A Goja DSL is a JavaScript authoring surface whose important semantics live in Go. JavaScript gives the user a compact language for composing work; Go owns the durable model, validation rules, host resources, runtime lifecycle, and the final values that cross back into the host. This division is the reason to build a DSL with `go-go-goja` instead of writing a pure JavaScript helper library.

This guide explains how to design those DSLs. It is not a short recipe. A good DSL is an API, a set of invariants, a runtime contract, a documentation surface, and a maintenance promise. The examples here come from several concrete systems: `goja-text` template and Markdown builders, document-level Markdown helpers, generated protobuf builders, xgoja provider modules, and Widget IR DSL work. The lesson across all of them is consistent: start with ownership and invariants, then choose the smallest JavaScript shape that lets users express the work clearly.

## The central design question

Before writing a module loader, decide what the JavaScript author is allowed to own. This is the design question that determines the rest of the API.

In a `go-go-goja` DSL, JavaScript should usually own **composition**. It decides which sections go into a report, which query recipe to run, which page widgets appear together, which callback to attach, or which preset to select. Go should usually own **domain state**. It stores builder internals, validates illegal combinations, normalizes data, owns files and network resources, controls runtime shutdown, and produces final typed objects or serializable data.

The distinction matters because JavaScript objects are easy to create and hard to govern. A plain object like this is convenient:

```javascript
const doc = markdown.document(source, {
  frontmatter: { format: "yaml", repair: true, optional: true },
  blocks: [
    { name: "context-window", fence: "context-window", json: true, strip: true }
  ]
});
```

The object is readable at first glance. But it also moves the domain model into a loose structure that Go can only inspect after the fact. Misspelled keys, contradictory options, missing required fields, and invalid nested shapes all become runtime decoding problems. Worse, every caller can construct slightly different shapes, and the API begins to depend on examples rather than on an explicit object model.

A Go-backed fluent API is more verbose, but it gives Go a place to hold state and enforce invariants while the user is still composing the workflow:

```javascript
const doc = markdown.document(source)
  .Frontmatter()
    .YAML()
    .Repair()
    .Optional()
    .End()
  .Blocks()
    .Block("context-window")
      .FromFence("context-window")
      .JSON().Repair().Optional().End()
      .StripFromBody()
      .End()
    .End()
  .Build();
```

The difference is not syntax preference. The second form has named intermediate objects: `DocumentBuilder`, `FrontmatterBuilder`, `BlockSetBuilder`, `BlockRuleBuilder`, and `JSONBlockBuilder`. Each object has a small responsibility and a narrow set of legal transitions. The builder can reject duplicate block names, unsupported formats, impossible parse policies, and missing required blocks. JavaScript still reads like a workflow, but Go owns the shape of that workflow.

The rule is not "always build fluent APIs." The rule is: **when intermediate state has invariants, keep that state in Go**. If the capability is a stateless conversion such as `yaml.parse(text)`, a flat function is better. If the capability is a stable query contract, a data recipe may be better. Fluent builders are appropriate when they encode a real construction process.

## What counts as a DSL in go-go-goja

A DSL in this ecosystem is any JavaScript API that lets scripts express domain work in terms that are smaller or clearer than the underlying Go implementation. Some DSLs are tiny. Others describe entire UI pages. The size is less important than the presence of a domain vocabulary.

Consider these examples:

```javascript
const yaml = require("yaml");
const value = yaml.parse(source);
```

This is a module, but not much of a DSL. The JavaScript vocabulary is almost identical to the operation. There is no substantial authoring language.

```javascript
const template = require("template");
const set = template.text()
  .Name("report")
  .Funcs("glazed", "sprig")
  .MissingKey("error")
  .Parse(source);

const result = set.Render(data);
```

This is a small DSL. The user chooses rendering mode, helper sets, strictness policy, and parse source. Go owns the parsed template set and rendering semantics.

```javascript
const md = require("markdown");
const out = md.builder()
  .Title("Sprint report")
  .Paragraph("Generated from structured runtime data.")
  .Table()
    .Columns("Name", "Status", "Owner")
    .Row("Parser", "done", "Ada")
    .Row("Builder", "planned", "Intern")
    .End()
  .Heading(2, "Next steps")
  .Checklist([
    { text: "Add service tests", checked: true },
    { text: "Publish help page", checked: false }
  ])
  .RenderString();
```

This is a document-generation DSL. It hides Markdown spacing rules, table pipe escaping, code fence selection, and validation behind a Go-owned document tree.

```javascript
const ui = require("ui.dsl");
const data = require("data.dsl");

exports.page = ui.page("sessions", {
  root: ui.stack([
    ui.panel({ title: "Sessions" }, [
      data.dataTable({ rows, columns })
    ])
  ])
});
```

This is an IR-authoring DSL. The output is not a Go object meant for direct method calls. It is a serializable Widget IR tree that a renderer, validator, or host application can consume. Go still owns helper registration, schema validation, and provider wiring; JavaScript owns page composition.

These examples should prevent one common misunderstanding: a DSL is not defined by chaining. Chaining is only one possible surface. A DSL is defined by the presence of a domain vocabulary and an ownership boundary.

## The responsibility split

The most durable rule is:

> Go owns lifecycle, domain state, validation, host resources, and typed boundaries. JavaScript owns policy, composition, recipes, and presentation assembly.

This rule came out of real applications. In `minitrace-viz`, JavaScript was excellent for composing course pages and view recipes. It was less appropriate for long-lived server mechanics such as upload body limits, filesystem cleanup, cache lifecycle, HTTP listener ownership, and shutdown. In `goja-text`, JavaScript was excellent for saying "make this template strict and render it with these helpers." It would have been a poor place to store the parsed template set or the Markdown document tree.

Use Go for:

- Runtime lifecycle, shutdown, closers, and cancellation.
- Host-sensitive operations such as filesystem, network, database, and process execution.
- Parsing, normalization, cache ownership, and storage.
- Validation rules that define what the domain permits.
- Typed objects that the Go host must recover without JSON round trips.
- Error shaping that includes domain context.
- Security policy and capability enforcement.

Use JavaScript for:

- Selecting options and composing workflows.
- Turning user or application policy into calls on Go-backed objects.
- Building page or report structures from runtime data.
- Grouping rows, choosing views, and expressing recipes.
- Providing callbacks at carefully reviewed synchronous boundaries.
- Application-level glue where flexibility matters more than static typing.

The boundary is not absolute. JavaScript can pass data objects into Go, and Go can return plain JSON-shaped maps. The important point is that **the canonical domain model should not accidentally become a pile of JavaScript maps**. If the model has invariants, name it in Go.

## Choosing the right API shape

A common mistake is to start from a syntax the author likes. The better starting point is to classify the capability. The following table is a practical decision guide.

| Capability shape | Use this JavaScript surface | Go owns | Example |
| --- | --- | --- | --- |
| Stateless conversion | Flat function | Parsing and error conversion | `yaml.parse(text)` |
| Small namespace with modes | Namespaced functions | Mode-specific implementation | `sanitize.yaml.sanitize(text)` |
| Reusable parsed object | Builder that returns a parsed set | Parse config and reusable compiled state | `template.text().Parse(src)` |
| Programmatic construction | Fluent builder over a Go document/tree | Blocks, validation, serialization | `markdown.builder().Table().End()` |
| Source parsing with policies | Nested builder plus built result views | Extraction rules and typed accessors | `markdown.document(source).Frontmatter().Build()` |
| Stable data access | Query recipes and view helpers | Schema, SQL, row contracts | `mt.queries.turnBlockRows(opts)` |
| Host needs typed payloads | Generated protobuf builders | Concrete protobuf messages | `pb.Task.builder().title("...").build()` |
| UI/page authoring | IR helpers and recipes | Schema, registry, validation | `ui.panel(...)`, `course.slide(...)` |

This table is a guide, not a law. The point is to keep the API proportional to the state it represents.

### Flat functions

Use flat functions when the operation is stateless and the parameters are already the domain. A YAML parser does not need a builder if all it does is parse one string with a small fixed option set. A path joiner does not need a domain object.

```
exports.Set("parse", func(input string) (any, error) {
    return ParseYAML(input)
})
```

Flat functions are easy to document and test. Their weakness is that options grow badly. Once a function has five optional behaviors, nested data, validation modes, callbacks, and reusable internal state, the flat shape begins to collapse under its own convenience.

### Go-backed result objects

Use result objects when the operation produces evidence or reusable data, not just a scalar. `RenderResult` is better than a bare string when the caller may need the template name, mode, byte count, or diagnostics. `ExtractionCandidate` is better than a string when source positions, wrapper kind, confidence, and raw text matter.

A result object should be boring. It should not contain surprising behavior. It exists to make outputs explicit and inspectable.

```
type RenderResult struct {
    Text         string \`json:"text"\`
    TemplateName string \`json:"templateName"\`
    Mode         Mode   \`json:"mode"\`
    Bytes        int    \`json:"bytes"\`
}
```

The design question is whether to expose fields, methods, or both. For immutable evidence objects, exported fields are acceptable. For higher-level views that may normalize, cache, or validate on demand, methods are safer.

### Fluent builders

Use a fluent builder when the user is constructing something with state. The object being constructed might be a template configuration, a Markdown document, a database query, a report preset, or a protobuf message. The builder exists because the final value is not meaningful until several choices have been made.

A good builder has four properties:

1. It stores intermediate state in Go.
2. It exposes methods that correspond to domain choices.
3. It has an explicit completion step such as `Build()`, `Parse()`, `Render()`, or `End()`.
4. It can explain invalid state with domain-specific errors.

The builder should not be a thin wrapper around a map. This is the anti-pattern:

```
type Builder struct {
    options map[string]any
}

func (b *Builder) Set(name string, value any) *Builder {
    b.options[name] = value
    return b
}
```

That design hides the map behind methods but keeps the same weakness. A real builder names the fields and transitions it cares about.

```
type TemplateBuilder struct {
    cfg TemplateConfig
    errors []string
}

func (b *TemplateBuilder) MissingKey(policy string) *TemplateBuilder {
    switch policy {
    case "default", "zero", "error", "invalid":
        b.cfg.MissingKey = policy
    default:
        b.errors = append(b.errors, fmt.Sprintf("unknown missing-key policy %q", policy))
    }
    return b
}
```

The method does not simply store a string. It interprets a domain option and records a validation error at the point where the user made the mistake.

### Data contracts and query recipes

Do not use builders when the real reusable asset is a stable data contract. This was one of the strongest lessons from larger xgoja applications. If a capability can be expressed as normalized rows, a query recipe, and a small view helper, that may be better than a deep fluent object tree.

Compare these two approaches:

```javascript
const report = mt.archiveFile(path)
  .report()
  .preset("full")
  .includeTools(true)
  .includeFiles(true)
  .build();
```
```javascript
const rows = db.query(mt.queries.turnBlockRows({ sessionId }).sql);
const frames = mt.views.groupTurnFrames(rows);
```

The builder reads nicely, but it can become an opaque wrapper around many hidden decisions. The query recipe exposes a stable contract: rows with named fields. JavaScript can group, filter, and present those rows. Go still owns the schema and query construction. This is often the better shape when the data will be consumed by many views.

The rule is: **use builders for constructing domain objects; use data contracts for transporting facts**.

### Generated schema builders

Generated builders are appropriate when a schema already exists and the host needs concrete typed objects. Protobuf builders are the clearest example. JavaScript gets a fluent construction surface:

```javascript
const pb = require("examples.xgoja.protobuf.v1");

const task = pb.Task.builder()
  .id("task-1")
  .title("Ship protobuf builders")
  .addTags("xgoja")
  .putLabels("component", "provider")
  .priority(pb.TaskPriority.TASK_PRIORITY_HIGH)
  .build();
```

The resulting value is not merely JSON. Go can recover a concrete protobuf message from the Goja value. This is valuable when JavaScript authors need a friendly authoring layer but Go handlers, providers, or hosts must receive typed payloads.

The tradeoff is that generated APIs can be large. They are best when the schema is the real source of truth. If the schema is unstable or the domain still needs design exploration, a hand-written builder may be easier to evolve.

### IR DSLs and recipe helpers

Widget IR DSLs and similar authoring systems have a different final value. They usually produce JSON-compatible trees rather than opaque Go objects. The DSL helpers are authoring conveniences; the wire contract is the important artifact.

A useful division is:

```
JavaScript DSL helper -> Widget IR JSON -> renderer / validator / host
```

Recipes should expand to ordinary IR. They should not create special wire types unless the renderer truly needs them.

```javascript
const page = course.courseStudio({
  title: "Workshop",
  main: ui.stack([
    data.dataTable({ rows, columns })
  ])
});
```

The renderer should not need to know that `courseStudio` was used. It should receive normal widgets. This keeps recipes as authoring macros rather than protocol extensions.

## The design-first workflow

The most expensive DSL mistakes happen before the first line of implementation. A fluent API is easy to add and hard to remove. A provider alias becomes part of buildspecs. A callback contract becomes part of runtime ownership. A poorly scoped options object appears in examples and then spreads into application code.

Design the DSL first. The design document does not need to be bureaucratic, but it should answer the questions that code will otherwise answer accidentally.

A good DSL design document includes:

- The problem the DSL solves.
- The existing lower-level APIs it composes.
- The JavaScript user story with realistic examples.
- The Go-owned domain model.
- The API shape decision and alternatives considered.
- The validation model.
- The data conversion rules.
- The runtime ownership rules for callbacks or async work.
- The xgoja provider and buildspec wiring, if applicable.
- The TypeScript, help, examples, and tests required before the API is considered shipped.

The document should show the target JavaScript before the Go implementation. This makes API review possible. In the document-builder work, the key prompt was to write the design guide before implementing. That was the correct sequence because the central question was API shape, not code mechanics.

## Designing the Go-owned model

Start with the Go model before the JavaScript methods. The model should be meaningful without Goja. If the service layer cannot be tested without a runtime, the module boundary is probably too tangled.

For a Markdown builder, the model is a document tree:

```
type Document struct {
    Blocks []Block
}

type Block interface {
    blockKind() string
}

type HeadingBlock struct {
    Level int
    Text  []Inline
}

type TableBlock struct {
    Columns []TableColumn
    Rows    [][]InlineCell
}
```

For a template module, the model is a builder config plus a parsed template set:

```
type TemplateConfig struct {
    Mode       Mode
    Name       string
    FuncSets   []string
    MissingKey string
    LeftDelim  string
    RightDelim string
}

type TemplateSet struct {
    Mode Mode
    Name string
    text *texttemplate.Template
    html *htmltemplate.Template
}
```

For a document parser, the model is a parsing policy plus a built view:

```
type DocumentBuilder struct {
    source      string
    frontmatter FrontmatterPolicy
    blocks      []BlockRule
    errors      []string
}

type ParsedDocument struct {
    body        string
    ast         *MarkdownNode
    frontmatter *FrontmatterView
    blocks      []DocumentBlock
}
```

The JavaScript methods should be thin projections onto this model. If the methods are designed first, the Go model tends to become a storage mechanism for API quirks. If the model is designed first, the API can be judged by whether it lets users build the model clearly.

## Builder lifecycle

Every builder needs a lifecycle. Without a lifecycle, users cannot tell whether they are still configuring, whether validation has run, or whether the returned value is safe to use.

The common lifecycle is:

```
create builder -> configure -> validate -> build/render/parse -> use result
```

In JavaScript:

```javascript
const validation = template.text()
  .Funcs("sprig", "glazed")
  .MissingKey("error")
  .Validate();

const set = template.text()
  .Funcs("sprig", "glazed")
  .MissingKey("error")
  .Parse(source);
```

`Validate()` should be available when invalid configuration can be detected before expensive work. `Build()` should either return a finished Go-backed value or raise/return a detailed error. `Render()` can be a combined build-and-render operation only when that is the natural completion step.

Nested builders need one more concept: returning to the parent.

```javascript
const output = md.builder()
  .Title("Report")
  .Table()
    .Columns("Name", "Status")
    .Row("Parser", "done")
    .End()
  .Paragraph("End of report.")
  .RenderString();
```

`Table()` returns a `TableBuilder`. `End()` returns the parent `MarkdownBuilder`. This pattern is readable, but it introduces lifecycle bugs. The implementation must define what happens when:

- `End()` is called twice.
- A row is added after `End()`.
- The parent renders while a child builder is unfinished.
- A child builder is abandoned.
- A child builder has validation errors.

There are two acceptable strategies. One is strict: double `End()` or use-after-end records an error and `Render()` fails. The other is idempotent: `End()` after the first call returns the parent and does nothing. Choose deliberately and test it. Do not leave it to pointer behavior.

A simple strict sketch:

```
type TableBuilder struct {
    parent *MarkdownBuilder
    table  *TableBlock
    ended  bool
}

func (t *TableBuilder) Row(values ...any) *TableBuilder {
    if t.ended {
        t.parent.addError("table: Row called after End")
        return t
    }
    if len(values) != len(t.table.Columns) {
        t.parent.addError(fmt.Sprintf("table: row has %d cells, expected %d", len(values), len(t.table.Columns)))
        return t
    }
    t.table.Rows = append(t.table.Rows, normalizeCells(values))
    return t
}

func (t *TableBuilder) End() *MarkdownBuilder {
    if t.ended {
        t.parent.addError("table: End called more than once")
        return t.parent
    }
    t.ended = true
    return t.parent
}
```

The important design feature is not the exact behavior. It is that the behavior is specified, implemented, and tested.

## Escape hatches

Every useful DSL eventually needs an escape hatch. Template APIs need custom functions. Markdown builders need raw Markdown. Widget DSLs need raw component nodes or generic node constructors. Query builders need raw SQL fragments in controlled places.

Escape hatches should be explicit and searchable.

```javascript
md.builder()
  .Paragraph("ordinary strings are escaped")
  .Raw("<custom markdown fragment>")
  .RenderString();
```

The name `Raw` matters. It tells the reader and reviewer that this call bypasses normal protection. The same applies to Widget IR:

```javascript
ui.rawWidget({ type: "ExperimentalPanel", props: { ... } })
```

or template helpers:

```javascript
template.html()
  .JSFunc("trustedHTML", trustedHTMLHelper) // reviewed separately
```

Do not hide escape hatches behind generic names such as `Value`, `Any`, or `Custom` unless the docs are very clear. A reviewer should be able to search for risky boundaries.

## Validation and errors

A DSL should fail with domain context. The error should explain what the user did wrong in the language of the DSL, not in the language of reflection or JSON decoding.

Poor error:

```
cannot convert undefined to string
```

Better error:

```
markdown.table: row 3 has 2 cells, expected 4 columns
```

Poor error:

```
invalid character 'i' looking for beginning of object key string
```

Better error:

```
document.block("context-window"): JSON parse failed after repair: invalid character 'i' looking for beginning of object key string
```

There are two layers of validation:

1. **Configuration validation** checks the builder state before doing work. Examples: unknown function set, invalid heading level, duplicate block name, unsupported missing-key policy.
2. **Execution validation** checks the source or data while parsing/rendering. Examples: template parse error, missing required frontmatter, JSON repair failure, table row mismatch discovered during render.

A builder can accumulate configuration errors and report them together:

```
type ValidationResult struct {
    OK     bool     \`json:"ok"\`
    Errors []string \`json:"errors"\`
}
```

This is often better than failing on the first method call. Fluent APIs are easier to use when users can build a shape and then ask what is wrong with it. But do not let execution continue after validation has failed.

For JavaScript-facing errors, prefer Go functions returning `(T, error)` where goja can convert the error into a thrown exception. For result-style APIs, return a structured result when partial success is meaningful. Choose one convention per operation and document it.

## Data conversion rules

Data conversion is where many Goja DSLs become confusing. Goja can export JavaScript values into Go maps and slices, and it can expose Go structs and methods back into JavaScript. That convenience is useful, but the API should not leave conversion semantics implicit.

Document these rules for every nontrivial module:

- Whether JavaScript objects are accepted as input.
- Whether Go-backed objects are accepted as input.
- Whether returned values are Go-backed objects or plain JSON-shaped values.
- How `undefined`, `null`, dates, arrays, maps, and functions are handled.
- Whether property names are case-sensitive and whether Go exported fields appear as PascalCase.
- Whether dynamic values are copied, normalized, or retained.

For template rendering, a practical rule set is:

1. JavaScript objects and arrays are accepted as render data.
2. Go-backed values are accepted directly when goja can export them.
3. Object property names are preserved.
4. Template selectors follow Go template rules.
5. Missing keys default to a strict policy unless the builder changes it.

For protobuf builders, the rules are stricter:

1. Ordinary nested protobuf messages must be built with generated builders.
2. Plain objects may be accepted for JSON-shaped well-known types such as `Struct`.
3. Built messages carry hidden Go protobuf values recoverable by Go.
4. The host should not parse JSON to recover the message.

For Widget IR helpers, the rules are different again:

1. Outputs should remain JSON-compatible.
2. Actions should be data specs, not functions, unless the host explicitly supports callbacks.
3. Slots should accept renderable values or child nodes according to schema.
4. Recipes expand to normal IR nodes.

The point is not to force one conversion policy. The point is to make the policy part of the API design.

## Runtime ownership and callbacks

Goja runtimes are single-threaded from JavaScript's point of view. Native modules must not call JavaScript functions, touch `goja.Value`, settle promises, or mutate JS objects from arbitrary goroutines. This is not an implementation detail. It is part of the DSL contract whenever callbacks or async work appear.

A synchronous callback can be safe when it is called while already executing on the runtime owner. A template `JSFunc` is an example if template execution itself happens synchronously on the owner:

```javascript
const set = template.text()
  .JSFunc("shout", (s) => String(s).toUpperCase())
  .Parse("Hello {{ shout .Name }}");
```

The Go wrapper must translate Go template arguments into Goja values, call the JS function on the owner, translate exceptions into Go errors, and return normal Go values to the template engine.

A sketch:

```
func wrapJSFunc(vm *goja.Runtime, fn goja.Callable) func(args ...any) (any, error) {
    return func(args ...any) (any, error) {
        jsArgs := make([]goja.Value, 0, len(args))
        for _, arg := range args {
            jsArgs = append(jsArgs, vm.ToValue(arg))
        }
        ret, err := fn(goja.Undefined(), jsArgs...)
        if err != nil {
            return nil, err
        }
        return ret.Export(), nil
    }
}
```

This sketch is only safe if it runs on the owner. If a background goroutine may call the function, use `runtimebridge.RuntimeServices` and the runtime owner helpers described in `async-patterns`.

Promise-based APIs need even more discipline:

```
exports.Set("work", func(input string) goja.Value {
    promise, resolve, reject := vm.NewPromise()
    services := mustRuntimeServices(vm)
    callCtx := runtimebridge.CurrentOwnerContext(vm)

    go func() {
        result, err := blockingGoWork(callCtx, input)
        _ = services.PostWithCustomContext(callCtx, "module.work.settle", func(context.Context, *goja.Runtime) {
            if err != nil {
                _ = reject(vm.NewGoError(err))
                return
            }
            _ = resolve(vm.ToValue(result))
        })
    }()

    return vm.ToValue(promise)
})
```

The background goroutine does Go work. The owner callback settles the promise. This separation prevents races and preserves runtime shutdown semantics.

If a DSL stores callbacks for later use, the design document must answer:

- Who owns the callback lifetime?
- How is it released?
- Which context cancels pending work?
- Can the callback be invoked after runtime shutdown begins?
- Are calls serialized through the runtime owner?
- What happens if the callback throws?

If the design cannot answer these questions, defer callbacks.

## NativeModule versus xgoja provider modules

`go-go-goja` has a simple native module contract:

```
type NativeModule interface {
    Name() string
    Doc() string
    Loader(*goja.Runtime, *goja.Object)
}
```

This is enough for simple modules and for packages that register themselves through the default module registry. A minimal module looks like this:

```
type module struct{}

func (module) Name() string { return "markdown" }
func (module) Doc() string  { return "Markdown parsing and rendering" }

func (module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    _ = exports.Set("builder", func() *MarkdownBuilder {
        return NewMarkdownBuilder()
    })
}

func init() {
    modules.Register(module{})
}
```

An xgoja provider module is richer. Use it when module setup depends on selected aliases, static config, host services, generated assets, runtime closers, or provider capabilities.

```
func Register(registry *providerapi.ProviderRegistry) error {
    return registry.Package("my-provider", providerapi.Module{
        Name:        "my.dsl",
        DefaultAs:   "my.dsl",
        Description: "My domain DSL",
        TypeScript:  myTypeScriptModule(),
        NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
            service := serviceFromConfigAndHost(ctx.Config, ctx.Host)
            return func(vm *goja.Runtime, moduleObj *goja.Object) {
                exports := moduleObj.Get("exports").(*goja.Object)
                installMyDSL(vm, exports, service)
            }, nil
        },
    })
}
```

The distinction is important:

| Need | Prefer |
| --- | --- |
| Module has no host config and can auto-register | `NativeModule` |
| Module must be selected under aliases in `xgoja.yaml` | xgoja provider module |
| Module needs embedded assets, database handles, stores, auth services, or host config | xgoja provider module |
| Module ships generated TypeScript through xgoja tooling | xgoja provider module or `TypeScriptDeclarer` |
| Module is a core reusable primitive loaded by `goja-repl` | `NativeModule` may be enough |

Provider authors must keep public command/config values separate from internal module setup config. Public Glazed sections are user-facing. Internal xgoja config is what `NewModuleFactory` consumes. Host services are for Go objects that cannot be represented as JSON config.

## TypeScript declarations are part of the design

A Goja DSL without TypeScript declarations is harder for humans and agents to use. Declarations are not merely editor polish. They force the API author to name the public contract.

A builder API should declare its lifecycle:

```ts
declare module "template" {
  export function text(): TemplateBuilder;
  export function html(): TemplateBuilder;

  export interface TemplateBuilder {
    Name(name: string): TemplateBuilder;
    Funcs(...names: string[]): TemplateBuilder;
    MissingKey(policy: "default" | "zero" | "error" | "invalid"): TemplateBuilder;
    Delims(left: string, right: string): TemplateBuilder;
    Validate(): ValidationResult;
    BuildConfig(): TemplateConfig;
    Parse(source: string): TemplateSet;
  }

  export interface TemplateSet {
    Render(data: unknown): RenderResult;
    RenderTemplate(name: string, data: unknown): RenderResult;
    Templates(): TemplateInfo[];
  }
}
```

The declaration reveals design problems. If the types are full of `any`, the API may be too loose. If the builder has dozens of unrelated methods, the module may need smaller namespaces. If nested builders are hard to name, the underlying hierarchy may be unclear.

For generated DSLs, declaration generation should be part of the build or smoke-test path. For hand-written DSLs, keep declarations next to the module implementation and test that xgoja can emit them when selected.

## Documentation and examples are part of the API

A DSL needs examples at three levels.

First, it needs a minimal example that proves the entrypoint:

```javascript
const md = require("markdown");
console.log(md.builder().Title("Hello").RenderString());
```

Second, it needs a realistic example that shows why the DSL exists:

```javascript
const report = md.builder()
  .Title(data.title)
  .Paragraph(data.summary)
  .Table()
    .Columns("Metric", "Value")
    .Rows(data.metrics.map(m => [m.name, m.value]))
    .End()
  .RenderString();
```

Third, it needs an edge-case example that teaches the dangerous boundary:

```javascript
md.builder()
  .Paragraph("escaped user text")
  .Raw("<!-- raw Markdown escape hatch; review before using -->")
  .RenderString();
```

Good help docs explain why the API exists, what it owns, how it fails, and what to use instead when it is the wrong abstraction. They should include troubleshooting tables because many users find docs through errors.

## Large DSLs: registries, manifests, and codegen

Small DSLs can be hand-written. Large DSLs need structure that prevents drift. Widget IR work is the clearest example. The same vocabulary appeared in TypeScript types, React renderer switches, Go helper maps, schema lists, docs, stories, and tests. That is manageable for a small set of widgets and painful as the set grows.

The better architecture separates three artifacts:

1. **Wire contract** — the serializable node, props, action, and slot shapes.
2. **Authoring DSL** — Goja helpers and recipes that produce the wire contract.
3. **Renderer adapters** — handwritten code that adapts wire props to actual UI components.

Do not generate everything. Generate contracts, registries, helper tables, docs, and coverage checks. Keep semantic adaptation code handwritten.

A colocated manifest is a good source of metadata:

```yaml
type: DataTable
module: data
helper: dataTable
props: DataTableProps
adapter:
  path: ./DataTable.widget.tsx
  export: dataTableWidget
reactComponent: DataTable
children: false
slots:
  - emptyMessage
  - columns.header
actions:
  - onRowSelect
status: stable
docs: Tabular data display using serializable cell specs.
```

The manifest should live near the component and adapter. Humans own local metadata; generators own aggregate indexes.

Generated outputs might include:

```
src/widgets/generated/default-registry.generated.ts
src/widgets/generated/data-registry.generated.ts
pkg/widgetdsl/generated_components.go
pkg/widgetschema/generated_components.go
pkg/xgoja/providers/widgetsite/doc/generated-widget-reference.md
```

The validation command should fail if:

- Two widgets declare the same type.
- Two helpers collide in one DSL module.
- The adapter file or export is missing.
- A stable widget lacks docs or stories.
- Generated outputs are stale.
- A DSL helper appears without a schema entry.
- A schema entry appears without a renderer adapter.

Large DSLs should also split by domain. A broad module such as `rag.dsl` becomes convenient at first and vague later. Prefer explicit modules such as `ui.dsl`, `data.dsl`, `context_window.dsl`, and `course.dsl` when the vocabulary has separate layers. A clean break may be better than compatibility aliases if the project can tolerate it. Compatibility wrappers are not free; they preserve old concepts in the new design.

## Recipes are not wire types

Recipes are authoring macros. They should make common compositions shorter, but they should usually emit ordinary domain objects or IR nodes.

```javascript
const page = course.courseStudio({
  title: "Workshop",
  sections,
  main: transcript.transcriptWorkspacePanel(model)
});
```

The output should be the same kind of `WidgetNode` tree that direct calls would produce. The renderer should not care whether a node came from a recipe, a direct helper, or generated JSON.

This keeps recipes easy to add and remove. It also prevents authoring convenience from leaking into the wire protocol.

## Testing strategy

A DSL needs tests at every boundary it crosses.

### Service-layer tests

Test the Go domain model without Goja first. These tests are fast and isolate the invariant logic.

```
func TestMarkdownBuilderTableEscapesPipes(t *testing.T) {
    out, err := NewMarkdownBuilder().
        Table().Columns("Name", "Note").Row("Ada", "uses | pipe").End().
        RenderString()
    if err != nil {
        t.Fatal(err)
    }
    if !strings.Contains(out, \`uses \| pipe\`) {
        t.Fatalf("expected escaped pipe, got %s", out)
    }
}
```

### Runtime integration tests

Test what JavaScript sees. Reflection, `Export()`, method names, thrown errors, and callback behavior can differ from pure Go assumptions.

```
func TestRequireMarkdownBuilder(t *testing.T) {
    rt := newTestRuntimeWithModule("markdown")
    value, err := rt.VM.RunString(\`
      const md = require("markdown");
      md.builder().Title("Report").RenderString();
    \`)
    if err != nil {
        t.Fatal(err)
    }
    if !strings.Contains(value.String(), "# Report") {
        t.Fatalf("unexpected output: %s", value.String())
    }
}
```

Runtime tests should cover:

- The module can be required under the selected name.
- Fluent calls return the expected Go-backed objects.
- PascalCase methods are visible.
- Plain JS data is accepted where promised.
- Invalid inputs throw or return structured errors as documented.
- Callbacks execute on the owner and propagate exceptions.

### Provider and generated-runtime tests

If the module is selected through xgoja, test that path too:

```bash
xgoja doctor -f xgoja.yaml
xgoja list-modules -f xgoja.yaml
xgoja gen-dts -f xgoja.yaml --out js/types/xgoja-modules.d.ts --strict
xgoja build -f xgoja.yaml
./dist/app eval 'const dsl = require("my.dsl"); dsl.smoke()'
```

A module that works in a unit test but is missing from the provider registry is not shipped. A provider that registers a module but lacks TypeScript or help docs is incomplete for serious use.

### Documentation tests and examples

Examples should be runnable. If a help page shows `markdown.builder().Table()`, there should be a smoke test or jsverb that exercises that shape. Documentation drift is API drift.

## A worked design: template DSL

The template DSL begins with a problem: JavaScript scripts need Go `text/template` and `html/template` rendering, including Glazed and Sprig helper functions, without dropping into string concatenation or reimplementing templating in JS.

A flat API would be easy:

```javascript
template.renderText(source, data, { funcs: ["sprig"], missingKey: "error" });
```

The flat API is useful as convenience sugar, but it is not the right core. A template has reusable parsed state, mode-specific escaping behavior, helper function sets, delimiter policy, missing-key policy, named templates, and render metadata. Those concepts deserve Go objects.

The better API is:

```javascript
const set = template.text()
  .Name("report")
  .Funcs("glazed", "sprig")
  .MissingKey("error")
  .Parse(source);

const result = set.Render(data);
```

Design decisions:

- `text()` and `html()` are separate construction paths because escaping semantics differ.
- `TemplateBuilder` owns configuration and validation.
- `TemplateSet` owns parsed text/html template state.
- `RenderResult` returns text plus metadata.
- JavaScript template functions are a later phase because callbacks cross runtime ownership and synchronous execution boundaries.
- File loading is not added casually because it intersects with host filesystem policy.

This is the shape to copy when a DSL wraps a Go engine with reusable parsed state.

## A worked design: Markdown builder DSL

The Markdown builder begins with a different problem: JavaScript can already create Markdown with strings or templates, but that becomes brittle for programmatic reports. Tables require pipe escaping and row validation. Code fences require fence selection. Lists and paragraphs require spacing rules. These are not application concerns.

The model is a Go-owned document tree. JavaScript appends typed blocks:

```javascript
const doc = markdown.builder()
  .Title("Sprint report")
  .Paragraph("Generated from runtime data.")
  .Table()
    .Columns("Name", "Status", "Owner")
    .Row("Parser", "done", "Ada")
    .End()
  .RenderString();
```

Design decisions:

- The builder lives under `require("markdown")` because it creates Markdown and can reuse existing parse/render/validate utilities.
- Ordinary strings are escaped text by default.
- `Raw()` is an explicit escape hatch.
- Tables are first-class child builders because they have their own invariants.
- `RenderHTML()` is convenience; Markdown output is primary.

This is the shape to copy when the user is constructing a structured text artifact.

## A worked design: document parsing DSL

The document builder begins with application duplication. Slide and handout loaders repeatedly split frontmatter, repaired YAML, extracted headings, searched for structured blocks, repaired JSON, stripped those blocks, and rendered Markdown. The lower-level modules already existed: `markdown`, `extract`, and `sanitize`. The missing piece was a document-level policy object.

The API is nested because the policy is nested:

```javascript
const doc = markdown.document(source)
  .Frontmatter()
    .YAML()
    .Repair()
    .Optional()
    .End()
  .Blocks()
    .Block("context-window")
      .FromXMLTag("context-window")
      .FromFence("context-window")
      .JSON().Repair().Optional().End()
      .StripFromBody()
      .End()
    .End()
  .Build();

const title = doc.Frontmatter().String("title", doc.FirstHeading(baseName));
const html = doc.RenderHTML();
const snapshot = doc.Block("context-window").JSONValue();
```

Design decisions:

- The builder composes lower-level primitives but does not replace them.
- The built document exposes methods, not mutable fields.
- The first implementation slice can omit field schemas while still providing typed accessors.
- Field-schema builders can be added later if required.
- Refactoring the real application validates whether the abstraction actually removes duplication.

This is the shape to copy when application code repeats parser policy across files.

## A worked design: Widget IR DSL

Widget IR DSLs are larger and require stronger drift control. The output is a serializable tree consumed by React renderers and host routes. The DSL is an authoring surface, not the renderer itself.

The design splits ownership:

```
colocated manifest + schema
        |
        v
generated helper tables, docs, registries, checks
        |
        v
handwritten adapters render React components
        |
        v
generic WidgetRenderer dispatches through registry
```

Design decisions:

- Adapters live near the React components they adapt.
- Registry assembly is generated or manifest-driven.
- Codegen produces contracts and assembly, not semantic render logic.
- DSL modules split by domain layer.
- Old broad modules can be removed when a clean break is allowed.
- Recipes output ordinary IR.

This is the shape to copy when a DSL vocabulary grows large enough that manual maps, renderer switches, docs, and tests begin to drift.

## Anti-patterns

### The options-map trap

An options map is attractive because it is quick to write and easy to extend. It becomes a trap when it holds domain state that deserves names and validation.

```javascript
// Convenient, but weak once the shape grows.
markdown.document(source, {
  frontmatter: true,
  repair: true,
  blocks: [{ name: "context-window", json: true }]
});
```

Prefer named builders when the options interact.

### The builder-sprawl trap

A builder is not automatically better. If the data is naturally rows, messages, or plain facts, expose a stable contract instead of inventing a chain for every view.

```javascript
// May hide too much data policy in a builder.
archive.report().preset("full").includeTools().includeFiles().build();

// Often better for reusable analysis.
const rows = db.query(queries.turns({ sessionId }).sql);
const view = views.groupTurns(rows);
```

### The JavaScript-backend trap

xgoja can assemble a rich JavaScript backend. That does not mean long-lived backend mechanics should live in JavaScript. Use Go for server lifecycle, limits, persistence, cleanup, and observability. Use JavaScript for policy and composition.

### The invisible-runtime trap

Any callback or async API that ignores runtime ownership is wrong even if it works in small tests. Goja values and callbacks must be touched on the runtime owner. Background goroutines do Go work and schedule owner-thread settlement.

### The undocumented-surface trap

If there are no TypeScript declarations, help docs, examples, and runtime tests, the DSL is not finished. It may be implemented, but it is not teachable or stable.

### The compatibility-wrapper trap

Compatibility wrappers preserve old concepts. Sometimes that is necessary. Sometimes it blocks the new design from becoming clear. If a project explicitly allows a clean break, prefer deleting old broad modules over wrapping them indefinitely.

## Design checklist

Use this checklist before implementing a new DSL.

- The design states what JavaScript owns and what Go owns.
- The API shape is chosen deliberately: flat function, result object, builder, data contract, generated schema builder, or IR helper.
- The Go domain model is named and testable without Goja.
- The JavaScript examples show realistic usage, not only trivial calls.
- The validation model distinguishes configuration errors from execution errors.
- The data conversion rules are documented.
- Escape hatches are explicit and searchable.
- Callback and async boundaries use runtime-owner semantics or are deferred.
- TypeScript declarations are part of the planned work.
- Help docs, examples, and smoke tests are part of the definition of done.

## Implementation checklist

Use this checklist while implementing.

- Write service-layer Go tests first.
- Implement Go domain structs and validation without importing Goja where possible.
- Implement builders/results as Go-backed objects.
- Keep the module `Loader` small; it should wire exports, not hold domain logic.
- Add Goja runtime tests for `require()` and reflected method behavior.
- Test JavaScript object input if the API accepts it.
- Test exported Go-backed result values as JavaScript sees them.
- Add TypeScript declarations and run declaration generation if xgoja is involved.
- Wire provider registration and buildspec selection when required.
- Add help pages and runnable examples.
- Add generated-runtime or CLI smoke tests.
- Record known limitations and deferred phases.

## Troubleshooting

| Problem | Likely cause | Design correction |
| --- | --- | --- |
| The API accepts a large nested options object and errors are vague. | Domain state lives in JavaScript maps. | Introduce Go-backed builders or typed config objects with `Validate()`. |
| The builder chain is long and hard to explain. | The DSL may combine multiple domains or too many phases. | Split namespaces, child builders, or modules by domain responsibility. |
| Runtime tests pass in Go but generated xgoja builds cannot `require()` the module. | Provider registration or buildspec selection is missing. | Add provider module registration and select it under `runtime.modules` / buildspec modules. |
| JavaScript callbacks panic or race under load. | Callback is invoked outside the runtime owner. | Use `runtimebridge.RuntimeServices` and owner scheduling helpers. |
| Users cannot tell what methods exist. | TypeScript declarations and examples are missing. | Add `TypeScriptDeclarer` or provider `TypeScript` metadata and generate `.d.ts`. |
| Generated docs mention helpers that do not exist. | Manual maps and docs drifted. | Add manifest/codegen checks or runtime smoke tests. |
| A large DSL imports every domain everywhere. | One broad module became the default bucket. | Split DSL modules by domain and provide explicit registry composition. |
| The DSL preserves old names that no longer fit. | Compatibility wrappers are shaping the new API. | Decide whether a clean break is allowed; if yes, remove aliases and update callers. |

## See also

- `goja-repl help creating-modules` explains the basic `NativeModule` contract and module registration path.
- `goja-repl help async-patterns` explains runtime ownership, Promise settlement, callbacks, and shutdown semantics.
- `goja-repl help typescript-declaration-generator` explains declaration generation for JavaScript-facing APIs.
- `goja-repl help protobuf-builders-user-guide` explains generated protobuf builders and typed message recovery.
- `xgoja help provider-runtime-config-and-host-services` explains provider config, host-service contribution, and module setup timing.
- `xgoja help tutorial-protobuf-builder-provider` shows how to expose generated builders through an xgoja provider module.

## Closing perspective

A DSL is successful when users can express domain work in JavaScript without inheriting the accidental complexity of the Go implementation, and when Go can still enforce the invariants that make the domain safe. That is the balance to preserve. If JavaScript owns too much, the DSL becomes a collection of unvalidated maps and callbacks. If Go owns too much, the DSL becomes a rigid wrapper that cannot express application policy. The right boundary lets each side do what it is good at: Go preserves the model; JavaScript composes it.
