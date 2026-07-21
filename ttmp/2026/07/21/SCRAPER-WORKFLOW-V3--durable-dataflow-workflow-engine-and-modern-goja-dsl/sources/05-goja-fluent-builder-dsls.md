<!-- Source: https://parc.yolo.scapegoat.dev/note/projects/2026/07/05/article-goja-fluent-builder-dsls-designing-typed-composable-grammars-in-go-for-javascript -->
<!-- Retrieved: 2026-07-21 -->

[Terminology and agent guide](https://parc.yolo.scapegoat.dev/AGENTS.md)

modifiedJul 20, 2026

tags

aliasesGoja Fluent Builder DSL Playbook, Goja DSL Patterns, Typed Goja Builders

createdJul 5, 2026

repo/home/manuel/workspaces/2026-07-03/improve-rag-evaluation-system

statusactive

typearticle

## Goja Fluent-Builder DSLs: Designing Typed Composable Grammars in Go for JavaScript

This article is a deep-dive technical analysis of how to design domain-specific languages (DSLs) that run inside the [goja](https://github.com/dop251/goja) JavaScript runtime, are implemented in Go, and use the fluent builder pattern to achieve strict runtime typechecking, validation, compile-time types from generated declarations, and a composable grammar extensible with lambdas. It is grounded in a survey of sixteen Goja DSLs built across the `go-go-golems` organization: `goja-bleve`, `goja-dbus`, `goja-text`, `goja-git`, `goja-github-actions`, `goja-treesitter`, `geppetto`, `discord-bot`, `researchctl`, `codesign`, the `widgetdsl` family in `rag-evaluation-system`, the `uidsl` and `express` modules in `go-go-goja`, the `minitracejs` module in `go-minitrace`, and the Go-native `glazed` and `go-emrichen` ancestors.

The target reader writes Go and JavaScript and wants to build a new Goja DSL that is type-safe, composable, and pleasant to author. The article does not prescribe a single library. It catalogs the implementation patterns that already exist, explains why each one works the way it does, and identifies which patterns are worth promoting into a shared playbook.

The base research behind this article lives in the docmgr ticket `GOJA-DSL-PLAYBOOK` at `rag-evaluation-system/ttmp/2026/07/05/GOJA-DSL-PLAYBOOK--goja-fluent-builder-dsl-playbook-base-research-and-resource-catalogue/`.

≡ Summary

- Goja DSLs fall into eight implementation patterns. Only three of them — the typed-reference fluent builder (goja-bleve), the lambda-configurator builder with fragment composition (codesign), and the immutable clone-on-each-step builder (geppetto) — realize the full goal of strict typing, validation, composable grammar, and compile-time declarations.
- The runtime typecheck substrate is a hidden, non-enumerable Go reference attached to each JavaScript object, paired with a generic `getTypedRef[T]` extractor that enforces types at the Go boundary.
- Compile-time types come from the `tsgen/spec` model in `go-go-goja`, declared per-module via the `modules.TypeScriptDeclarer` interface. Geppetto goes further and runs a DTS parity test that asserts the generated `.d.ts` matches the runtime export surface.
- The composable grammar is the lambda-configurator pattern — `project(name).goal(title, g => g.id(...).status(...))` — combined with `.use(fragment)` reusable builder lambdas. Codesign is the reference implementation.
- The `widgetdsl` `data.dsl` in `rag-evaluation-system` is the cautionary example: map-IR helpers that return `Record<string, any>`, validate by panicking, and emit open-ended TypeScript.

## Why this note exists

Over several projects, the same problem kept recurring. A Go application embeds goja to let users author behavior in JavaScript. The first instinct is to expose Go functions directly: `require("thing").doStuff(args)`. That works for a handful of calls, but as the surface grows, three pressures converge.

The first pressure is type safety. JavaScript callers pass arguments as plain objects. Go receives them as `goja.Value` and must decode them. Without a discipline, every function re-implements the same fragile `ExportTo` + type-assert + error path, and mistakes surface only at the call site that triggers them.

The second pressure is composition. Real configurations nest: a search request contains a query, the query contains field clauses, the clauses contain options. Exposing flat functions forces the caller to construct deeply nested object literals by hand, which is verbose and error-prone. A fluent builder lets the caller describe the structure step by step, with the Go side validating each step.

The third pressure is documentation. A DSL whose types are not declared in TypeScript cannot be checked by the editor, cannot be completed by IntelliSense, and cannot be verified against the runtime. The declarations drift from the implementation, and the drift is invisible until a caller breaks.

This article records the patterns that address all three pressures, drawn from working code rather than speculation. It exists so that the next Goja DSL starts from the best existing model instead of rediscovering it.

## When to use this pattern

Use a Go-implemented fluent-builder Goja DSL when:

- The configuration you are exposing is structured and nested, not a flat bag of options.
- The Go side owns resources (file handles, network connections, database handles, search indexes) that must not leak into JavaScript.
- Wrong arguments should fail at the Go boundary with a precise error, not silently produce a malformed object.
- You want TypeScript declarations that callers can rely on for completion and checking.
- The DSL should be extensible by callers through their own functions, not just through predefined options.

Do not use this pattern when a thin functional API is enough. `goja-git` exposes `repo.status()` and `repo.add({Paths})` with no chaining, and that is the right choice: the operations are flat, the options are shallow, and a builder would add ceremony without value. `goja-github-actions` polyfills `@actions/core` and `@actions/github` to match an existing JavaScript convention, and imposing a builder on top would break that convention. The builder pattern earns its complexity when structure and type safety matter more than brevity.

## The mental model: Go owns the handle, JavaScript describes intent

Every typed Goja DSL in the survey rests on the same division of responsibility. JavaScript describes intent. Go owns execution, resource lifecycle, marshaling, and validation. The boundary between them is a JavaScript object that carries a hidden reference to a Go struct.

This division is not stylistic. It exists because a `goja.Runtime` is not thread-safe in the ordinary sense. A single VM must be accessed by one goroutine at a time. The `go-go-goja` `engine` package formalizes this with a `RuntimeOwner` that serializes VM access by scheduling callbacks onto an event loop and associating each callback with a context. When a JavaScript callback fires — a lambda passed into a builder, a signal handler, a Promise continuation — it must execute on the runtime owner, not on an arbitrary goroutine. `goja-dbus` states this rule explicitly: all JavaScript callbacks, `goja.Value` creation, EventEmitter delivery, and Promise settlement happen on the runtime owner. Geppetto carries a `runtimeowner.RuntimeOwner` in its module `Options` for the same reason.

The consequence for builder design is that the Go struct behind a builder handle is owned by the runtime, not by the JavaScript caller. The caller never sees the struct directly. The caller sees a JavaScript object whose methods are Go functions, and those Go functions mutate or clone the hidden struct. The hidden reference is the contract that lets the Go side recover the typed struct when a later call hands the object back.

## The architecture of a Goja native module

A Goja native module is registered with the `goja_nodejs/require` subsystem. The module's `Loader` receives a `*goja.Runtime` and a module object, populates its `exports`, and returns. The `go-go-goja` `modules` package formalizes this with two interfaces:

```
type NativeModule interface {
    Name() string
    Doc() string
    Loader(vm *goja.Runtime, moduleObj *goja.Object)
}

type TypeScriptDeclarer interface {
    TypeScriptModule() *ts.Module
}
```

A module that implements both gets its TypeScript declaration wired into the generated `.d.ts` bundle. The `codesign` module is the cleanest example:

```
const ModuleName = "codesign"

type module struct{}

var _ modules.NativeModule = (*module)(nil)
var _ modules.TypeScriptDeclarer = (*module)(nil)

func (module) Name() string { return ModuleName }
func (module) Doc() string  { return "CPU/GPU codesign run builder, simulation, and artifact API" }

func (module) TypeScriptModule() *ts.Module {
    return &ts.Module{Name: ModuleName, RawDTS: []string{codesignDTS}}
}

func (module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    rt := &moduleRuntime{vm: vm, callbackDevices: map[string]goja.Value{}, ...}
    exports := moduleObj.Get("exports").(*goja.Object)
    rt.mustSet(exports, "runSpec", rt.runSpec)
    rt.mustSet(exports, "compareMetric", rt.compareMetric)
    // ...
}

func init() { modules.Register(&module{}) }
```

The `var _ Interface = (*T)(nil)` line is the standard Go interface-satisfaction assertion. It guarantees at compile time that the module implements both interfaces. The `init()` function registers the module so that hosts and xgoja-generated binaries can discover it.

The `mustSet` helper is a convention shared across the typed DSLs. It sets a property on a JavaScript object and panics with a Go error if the set fails, which should not happen for a well-formed export:

```
func (m *moduleRuntime) mustSet(o *goja.Object, key string, value any) {
    if err := o.Set(key, value); err != nil {
        panic(m.vm.NewGoError(fmt.Errorf("codesign: set export %s: %w", key, err)))
    }
}
```

This is the scaffolding. The patterns that follow differ in what the exported functions return and how they enforce types.

## Pattern A: the typed-reference fluent builder

Goja-bleve is the reference implementation for runtime typechecking. Its core mechanism is a hidden, non-enumerable property on each JavaScript object that points to a Go struct. The property is invisible to JavaScript enumeration, so callers cannot accidentally overwrite or inspect it, but the Go side can recover it on every subsequent call.

### The hidden reference and the ref kind

Every Go-facing handle wraps a struct that embeds a `refBase`:

```
type refBase struct {
    api    *moduleRuntime
    kind   refKind
    closed bool
}

type refKind string

const (
    refKindFieldBuilder  refKind = "fieldBuilder"
    refKindFieldMapping  refKind = "fieldMapping"
    refKindMapping       refKind = "mapping"
    refKindIndex         refKind = "index"
    refKindQuery         refKind = "query"
    refKindSearchRequest refKind = "searchRequest"
    refKindBatch         refKind = "batch"
    // ...
)

type fieldBuilderRef struct {
    refBase
    mapping *mapping.FieldMapping
}
```

The `kind` field tags what the handle represents. A field builder is not a field mapping, and the Go boundary uses the kind to reject misuse. The hidden property is attached when the wrapper is created:

```
const hiddenRefKey = "__bleve_ref"

func (m *moduleRuntime) newWrapper(ref any, kind refKind) *goja.Object {
    obj := m.vm.NewObject()
    m.mustSet(obj, "type", string(kind))
    m.attachRef(obj, ref)
    return obj
}
```

The `type` property is enumerable and gives JavaScript a readable hint about what the object is. The `__bleve_ref` property is non-enumerable and carries the Go pointer.

### The generic type extractor

The typecheck happens at the boundary, in a generic function:

```
func getTypedRef[T any](m *moduleRuntime, v goja.Value, expected string) (*T, error) {
    ref := m.getRef(v)
    if ref == nil {
        return nil, fmt.Errorf("bleve: expected %s wrapper, got value without Go reference", expected)
    }
    typed, ok := ref.(*T)
    if !ok {
        return nil, fmt.Errorf("bleve: expected %s wrapper, got %T", expected, ref)
    }
    return typed, nil
}
```

Go generics let the extractor return a concrete pointer type, so the call site gets a `*fieldBuilderRef` directly with no further assertion. When a function accepts a value that must be a field builder, it calls:

```
builder, err := getTypedRef[fieldBuilderRef](m, arg, "field builder")
if err != nil {
    return nil, err
}
```

The error message tells the caller exactly what was expected and what was received. This is the difference between a typed DSL and a map-IR DSL: the failure is immediate, specific, and recoverable.

### The fluent chain and the build terminal

A field builder is created by `bleve.field()`, which returns a wrapper around a fresh `fieldBuilderRef`. Each chain method mutates the wrapped struct and returns the same object, so the caller can chain:

```
func (m *moduleRuntime) fieldBuilder() *goja.Object {
    ref := &fieldBuilderRef{refBase: refBase{api: m, kind: refKindFieldBuilder}}
    obj := m.newWrapper(ref, refKindFieldBuilder)

    m.mustSet(obj, "text", func() *goja.Object {
        ref.mapping = bleve.NewTextFieldMapping()
        return obj
    })
    m.mustSet(obj, "keyword", func() *goja.Object {
        ref.mapping = bleve.NewKeywordFieldMapping()
        return obj
    })
    m.mustSet(obj, "store", func(enabled bool) *goja.Object {
        ref.mapping.Store = enabled
        return obj
    })
    // ...

    m.mustSet(obj, "build", func() (*goja.Object, error) {
        if ref.mapping == nil {
            return nil, fmt.Errorf("bleve: field type is required before build()")
        }
        built := &fieldMappingRef{refBase: refBase{api: m, kind: refKindFieldMapping}, mapping: ref.mapping}
        return m.newWrapper(built, refKindFieldMapping), nil
    })
    return obj
}
```

Two things matter here. First, the chain methods return the same `obj`, which is what makes `.text().store(true).build()` chain naturally in JavaScript. Second, `.build()` returns a **different** wrapper — one whose kind is `fieldMapping`, not `fieldBuilder`. The builder and the built artifact are distinct types. A caller who tries to call `.store(true)` on a built field mapping will fail the `getTypedRef` check, because the kind no longer matches.

The JavaScript caller experiences this as a fluent chain with a terminal:

```javascript
const bleve = require("bleve")

const text = bleve.field()
  .text()
  .store(true)
  .includeTermVectors(true)
  .build()

const keyword = bleve.field().keyword().store(true).build()
```

The `build()` terminal is where validation returns an error rather than panicking. If no field type was set, `build()` returns `("bleve: field type is required before build()")`. The caller handles it.

### Why the same-object mutation is acceptable

Goja-bleve mutates the same `fieldBuilderRef` on every chain call and returns the same JavaScript object. This is allocation-efficient — no new wrappers are created per step — but it means a builder is single-use and stateful. If a caller stores a half-built builder and reuses it across two `build()` calls, the second call reflects the accumulated state of both. For a search-mapping DSL where builders are constructed and immediately consumed, this is fine.

The same-object model breaks down when a builder might be shared or reused concurrently. That is the problem geppetto's variant solves.

## Pattern A′: the clone-on-each-step immutable builder

Geppetto reuses goja-bleve's hidden-reference substrate almost exactly — it names the key `__geppetto_ref` and uses the same `attachRef` and `mustSet` helpers — but it changes one thing: every chain method clones the reference and returns a **new** wrapper.

```
func (r *engineBuilderRef) cloneFor(api *moduleRuntime) *engineBuilderRef {
    if r == nil {
        return &engineBuilderRef{api: api}
    }
    var settings *inferenceSettingsRef
    if r.settings != nil {
        settings = r.settings.cloneFor(api)
    }
    return &engineBuilderRef{api: api, settings: settings}
}

func (m *moduleRuntime) newEngineBuilderObject(ref *engineBuilderRef) *goja.Object {
    ref.api = m
    o := m.vm.NewObject()
    m.attachRef(o, ref.cloneFor(m))

    m.mustSet(o, "inference", func(call goja.FunctionCall) goja.Value {
        settingsRef, err := m.requireInferenceSettingsRef(call.Arguments[0])
        if err != nil {
            panic(m.vm.NewGoError(err))
        }
        next := ref.cloneFor(m)
        next.settings = settingsRef.cloneFor(m)
        return m.newEngineBuilderObject(next)
    })

    m.mustSet(o, "build", func(goja.FunctionCall) goja.Value {
        if ref.settings == nil || ref.settings.settings == nil {
            panic(m.vm.NewGoError(fmt.Errorf("engine().build requires inference(settings) first")))
        }
        settings := cloneInferenceSettings(ref.settings.settings)
        eng, err := enginefactory.NewEngineFromSettings(settings)
        if err != nil {
            panic(m.vm.NewGoError(err))
        }
        return m.newEngineObject(&engineRef{Name: "inferenceSettings", Engine: eng, ...})
    })
    return o
}
```

The `inference(settings)` call does not mutate `ref`. It clones `ref` into `next`, sets `next.settings`, and returns a new wrapper around `next`. The original builder is unchanged. A caller can branch:

```javascript
const base = geppetto.engine().inference(sharedSettings)
const engA = base.build()
const engB = base.middleware(otherMiddleware).build()
```

Because each step is immutable, `base` remains valid after `engA` is built. This is safer for reuse and for the kind of programmatic composition where a DSL is driven by other code that may call the same builder from multiple paths.

The tradeoff is allocation. Every chain step allocates a new wrapper and a new ref clone. For a DSL with deep chains called in tight loops, this adds garbage. Goja-bleve's same-object mutation is the better choice there. The decision is not which is universally correct; it is which failure mode you prefer to accept. Immutability protects against aliasing bugs; mutation protects against allocation overhead.

### The DTS parity test

Geppetto's most important contribution is not the clone variant. It is the test that enforces compile-time-type correctness. The file `pkg/js/modules/geppetto/dts_parity_test.go` contains:

```
func TestGeneratedDTSMatchesRuntimeExportSurface(t *testing.T) {
    dtsPath := geppettoDTSPath(t)
    expected, err := parseDTSSurfaceFile(dtsPath)
    if err != nil {
        t.Fatalf("failed parsing generated d.ts (%s): %v", dtsPath, err)
    }

    rt := newJSRuntime(t, Options{})
    assertSameSet(
        t,
        "geppetto top-level exports",
        expected.TopLevel,
        runtimeObjectKeys(t, rt, \`require("geppetto")\`),
    )

    for _, namespace := range []string{"consts", "inferenceProfiles", "schema", "turnStores"} {
        want, ok := expected.Grouped[namespace]
        if !ok {
            t.Fatalf("generated d.ts does not contain export object for %q", namespace)
        }
        assertSameSet(
            t,
            fmt.Sprintf("geppetto.%s exports", namespace),
            want,
            runtimeObjectKeys(t, rt, fmt.Sprintf(\`require("geppetto").%s\`, namespace)),
        )
    }
}
```

The test loads the generated `geppetto.d.ts`, parses its exported names, instantiates the runtime, reads the actual `require("geppetto")` export keys, and asserts that the two sets match. If a developer adds a Go export but forgets to update the declaration, or adds a declaration but forgets to wire the export, the test fails.

This is the discipline that makes compile-time types trustworthy. Without it, a `.d.ts` is a claim that may or may not match reality. With it, the declaration is a tested contract. The playbook should require a parity test for every typed DSL.

## Pattern B: the plain Go builder struct

Go-minitrace's `minitracejs` module takes a different approach. It does not attach hidden references to JavaScript objects. Instead, it keeps a plain Go struct on the Go side and returns a fresh JavaScript object on every chain call that re-wraps the same struct pointer.

```
type SourceSetBuilder struct {
    sources []dbSource
    last    int
    errors  []string
}

func sourcesBuilderObject(vm *goja.Runtime, b *SourceSetBuilder) *goja.Object {
    obj := vm.NewObject()
    _ = obj.Set("File", func(path string) *goja.Object {
        b.AddFile(path)
        return sourcesBuilderObject(vm, b)
    })
    _ = obj.Set("Dir", func(path string) *goja.Object {
        b.sources = append(b.sources, dbSource{Kind: "dir", Path: path, Name: path})
        b.last = len(b.sources) - 1
        return sourcesBuilderObject(vm, b)
    })
    _ = obj.Set("Name", func(name string) *goja.Object {
        b.NameMostRecent(name)
        return sourcesBuilderObject(vm, b)
    })
    _ = obj.Set("Validate", func() ValidationResult { return b.Validate() })
    _ = obj.Set("Build", func() (*SourceSet, error) { return b.Build() })
    return obj
}
```

The JavaScript caller sees the same fluent experience:

```javascript
const m = require("minitrace")
const sources = m.sources()
  .File("./a.json")
  .Archive("./b.minitrace.json")
  .Dir("./sessions/")
  .Glob("./output/active/*/*.minitrace.json")

const check = sources.Validate()   // { valid: true, errors: [] }
const set = sources.Build()        // SourceSet or throws
```

The difference from Pattern A is that there is no `getTypedRef` boundary. The builder is a plain Go struct, and the JavaScript object is a thin view over it. Type safety comes from the Go function signatures themselves — `File(path string)` will reject a non-string argument at the goja conversion layer — not from an explicit kind check.

This pattern is simpler to implement and easier to unit-test in pure Go, because the builder is just a struct with methods. It loses the ability to distinguish a builder from a built artifact at the boundary. If a function accepts a `SourceSet`, it receives a plain map or struct, not a typed handle it can interrogate. The validation discipline carries the weight instead.

### The validate-then-build discipline

Go-minitrace introduces a validation contract that the typed-reference DSLs do not have in the same form: an explicit `Validate()` that returns a structured result, separate from `Build()`.

```
type ValidationResult struct {
    Valid  bool     \`json:"valid"\`
    Errors []string \`json:"errors,omitempty"\`
}

func (b *SourceSetBuilder) AddFile(path string) {
    path = strings.TrimSpace(path)
    if path == "" {
        b.errors = append(b.errors, "file path must not be empty")
        return
    }
    // ...
}
```

Errors are accumulated in the builder's `errors` slice during chaining, not thrown immediately. A caller can construct a complete source set, call `Validate()`, and receive every problem at once. This is materially better for authoring ergonomics than failing on the first error, because the caller can fix several issues per round-trip.

The `Build()` terminal returns `(value, error)` and refuses to produce a `SourceSet` if errors exist. The two-terminal shape — `Validate()` for inspection, `Build()` for materialization — is the validation discipline the playbook should adopt.

### Multiple terminals

The import builder in `minitracejs` shows that a terminal is not always called `build()`. An import pipeline has several distinct end states: detect the format, convert, preview, save diagnostics, and persist. Each is a terminal:

```javascript
m.importer()
  .File("./claude.json")
  .Into("./out")
  .SessionID("abc")
  .Strict(true)
  .Detect()        // (map, error)
  .Convert()       // (obj, error)
  .Preview()       // (map, error)
  .Diagnostics()   // []map
  .Save()          // (map, error)
```

The lesson is that a builder's terminal is whatever materializes the accumulated state into a result. The name should describe the result, not the act of building.

## Pattern C: the map-IR helper DSL (the cautionary example)

The `widgetdsl` family in `rag-evaluation-system` — `ui.dsl`, `data.dsl`, `context_window.dsl`, `course.dsl`, `cms.dsl` — takes a third approach. Helpers return plain `map[string]any`. There are no typed handles, no hidden references, no builders. The structure is the data.

```
func (r *runtime) installDataGrammar(exports *goja.Object) {
    setExport(exports, "f", r.fieldRoleObject())
    setExport(exports, "schema", r.schemaCtor)
    setExport(exports, "record", r.recordVerb)
    setExport(exports, "collection", r.collectionVerb)
    // ...
}

func (r *runtime) fieldRoleObject() *goja.Object {
    f := r.vm.NewObject()
    for _, role := range fieldRoles {
        role := role
        setExport(f, role, func(options ...goja.Value) map[string]any {
            out := map[string]any{"role": role}
            mergeOptions(out, exportOptions(options))
            return out
        })
    }
    return f
}
```

The JavaScript caller writes:

```javascript
const data = require("data.dsl")
const schema = data.schema({
  id:    data.f.key({ label: "ID" }),
  title: data.f.primary({ required: true, maxLength: 160 }),
})
data.record(values, { schema })
```

This is easy to author. The helpers are short, the maps are JSON-serializable, and the renderer needs no new code because everything compiles to plain Widget IR. The cost is paid in three places.

### Validation by panic

The grammar validates by panicking:

```
func (r *runtime) schemaCtor(call goja.FunctionCall) goja.Value {
    if len(call.Arguments) == 0 || !isPlainObject(call.Arguments[0]) {
        panic(r.vm.NewGoError(fmt.Errorf("data.dsl schema(fields) requires an object of f.* field specs")))
    }
    // ...
    for _, key := range obj.Keys() {
        spec := exportObject(obj.Get(key))
        if _, ok := spec["role"]; !ok {
            panic(r.vm.NewGoError(fmt.Errorf("data.dsl schema field %q must be built with f.<role>(...)", key)))
        }
    }
}
```

A panic in a goja binding becomes a thrown error in JavaScript. The caller can catch it, but the error is a single string with no structure. There is no accumulation: the first invalid field aborts the rest of the schema construction. Compare this to go-minitrace's `Validate()` returning `{valid, errors}`, which lets the caller see every problem.

### The magic tag and the untyped boundary

The schema is tagged with a magic string to distinguish it from a plain map:

```
return r.vm.ToValue(map[string]any{"__ragSchema": true, "fields": fields})
```

Later, `record` and `collection` check the tag by type-assertion:

```
if _, ok := schema["__ragSchema"]; !ok {
    panic(r.vm.NewGoError(fmt.Errorf("data.dsl record(values, {schema}) requires a schema built with data.schema")))
}
```

This works, but it is fragile. Any caller can set `__ragSchema: true` on a plain object and bypass the check. The Go boundary cannot distinguish a real schema from a forgery, because the handle carries no type information. A typed reference would make this impossible: `getTypedRef[schemaRef]` rejects anything that was not produced by `schema()`.

### Open-ended TypeScript

The most consequential cost is in the declarations. The `widgetdsl` TypeScript generator emits:

```typescript
export type Props = Record<string, any>;
export interface WidgetNode { kind: string; [key: string]: any; }
export interface FieldSpec { role: string; [key: string]: any; }
export function f(props?: Props): WidgetNode;
```

`Record<string, any>` means TypeScript cannot check any option passed to any helper. A typo in `maxLength` — `maxLengh: 160` — is accepted silently. The editor offers no completion, because any property is valid. This is the concrete failure that makes `data.dsl` "not really all that great": the compile-time-type layer does not exist, by design, because the IR is intentionally open-ended.

The widgetdsl design comment says "individual component props remain open-ended by design." That design choice is coherent for a renderer that must accept arbitrary React props, but it is the wrong choice for the grammar verbs. The grammar verbs have a closed set of roles, a closed set of cell kinds, and a closed set of actions. They could emit precise types. They do not, because the generator treats everything as `Props`.

## Pattern F: the lambda-configurator builder with fragment composition

Researchctl and codesign represent the strongest realization of the fluent-builder goal. They combine the typed-handle discipline with two ideas that the other DSLs do not have: lambda configurators and fragment composition.

### The lambda configurator

A lambda configurator is a JavaScript function that receives a fresh sub-builder and configures it. The top-level builder does not expose every option as a chain method. It exposes collection methods that take a title and a function:

```javascript
const { project } = require("researchctl")

module.exports = project("Example JS project")
  .goal("Choose a backend", g => g.id("GOAL-001").status("active").priority("P1"))
  .hypothesis("Simulation gives enough signal", h => h.id("H-001").status("open").confidence("unknown"))
  .experiment("Run simulation", e => e.id("EXP-001").status("planned").tests("H-001"))
  .toSpec()
```

The Go side applies the callback to a fresh sub-builder:

```
func (m *moduleRuntime) project(name string) *goja.Object {
    p := &spec.ResearchProjectSpec{SchemaVersion: spec.SupportedSchemaVersion, Kind: spec.KindResearchProject, Name: name}
    return m.projectBuilder(p)
}

// projectBuilder exposes .goal(title, build?), .hypothesis(claim, build?), etc.
// each collection method creates a fresh entity, applies build(entityBuilder) if given,
// and appends the entity to the project.
```

The sub-builder is itself a fluent same-object chain:

```
m.mustSet(o, "id", func(v string) *goja.Object { e.ID = spec.ID(v); return o })
m.mustSet(o, "status", func(v string) *goja.Object { e.Status = spec.Status(v); return o })
m.mustSet(o, "priority", func(v string) *goja.Object { e.Priority = spec.Priority(v); return o })
m.mustSet(o, "testedBy", func(v ...string) *goja.Object { e.TestedBy = append(e.TestedBy, ids(v)...); return o })
m.mustSet(o, "tag", func(v ...string) *goja.Object { e.Tags = append(e.Tags, v...); return o })
```

The pattern separates two concerns cleanly. The top-level builder owns the collection: what entities exist, in what order, with what titles. The sub-builder owns the entity's fields. The lambda is the bridge: it lets the caller describe the entity's fields in the scope where the entity is being added, without the caller having to hold a reference to the sub-builder across calls.

This is the answer to the ticket's requirement that the grammar be "extended with lambdas." The lambda is not a callback registered for later execution. It is a configuration function applied immediately to a fresh builder. The Go side controls what builder the lambda receives, so the lambda cannot escape its scope.

### Fragment composition

Codesign extends the pattern with `.use(fragment)`. A fragment is a reusable builder-configuration function typed as `FragmentFn<T>`, where `T` is the builder type. A caller writes a fragment once and applies it to any compatible builder:

```javascript
const codesign = require("codesign")

const reusableTopology = t => t
  .cpu("cpu0", { speed: 3.0, lanes: 4 })
  .accelerator("gpu0", { speed: 1.5 })

const run = codesign.runSpec("my-run")
  .experiment("EXP-001")
  .backend("cpu-sim")
  .topology(reusableTopology)
  .policy("min_finish_time")
  .use(commonMetrics)
  .validate()
  .toSpec()
```

The Go side applies a fragment through a helper that asserts the argument is a function and calls it with the builder:

```
func (m *moduleRuntime) applyBuilderCallback(b *goja.Object, cb goja.Value) error {
    fn, ok := goja.AssertFunction(cb)
    if !ok {
        return fmt.Errorf("expected a builder function, got %T", cb)
    }
    if _, err := fn(b, b); err != nil {
        return err
    }
    return nil
}

// in runSpecBuilder:
set("topology", func(cb goja.Value) (*goja.Object, error) {
    b := m.topologyBuilder()
    if err := m.applyBuilderCallback(b.obj, cb); err != nil {
        return nil, err
    }
    run.Topology = codesignspec.TopologySpec{Devices: append([]codesignspec.DeviceSpec(nil), b.devices...)}
    return obj, nil
})
```

`.use(fragment)` is the same mechanism applied to `self`: the fragment receives the run-spec builder itself, so it can apply a bundle of settings that span multiple sub-builders. This is composition: a fragment is a named, reusable unit of configuration that can be shared across runs.

### Lambdas that cross the Go/JS boundary at runtime

Not every lambda is a configurator applied once. Some are runtime callbacks that Go stores and invokes later. Codesign has three: `jsDevice` for custom device estimation, `policyCallback` for custom scheduling, and `callback` metrics for custom aggregation:

```javascript
codesign.runSpec("my-run")
  .policyCallback("my-policy", (task, candidateIds, state, scores) => "dev0")
  .topology(t => t.jsDevice("est", (phase, task, state, fallback) => estimate(task), { ... }))
  .metrics(m => m.callback("custom", (events) => ({ value: events.length, unit: "events" })))
```

The Go side validates each callback with `goja.AssertFunction` and stores it in a map keyed by an identifier:

```
set("policyCallback", func(id string, callback goja.Value) (*goja.Object, error) {
    if _, ok := goja.AssertFunction(callback); !ok {
        return nil, fmt.Errorf("policy callback must be a function")
    }
    m.callbackPolicies[id] = callback
    run.Policy = codesignspec.PolicySpec{ID: id, Type: callbackPolicyType, Config: codesignspec.JsonObject{"callbackId": id}}
    return obj, nil
})
```

When the simulator runs, it looks up the callback by identifier and invokes it on the runtime owner. This is where the runtime-ownership rule becomes load-bearing: a callback invoked from a simulator goroutine must be scheduled onto the VM's event loop, not called directly, or the VM races itself. Codesign's `callbackDevices` / `callbackPolicies` / `callbackMetrics` maps are the registry; the runtime owner is the scheduler.

### Precise TypeScript interfaces

Codesign's declarations are not open-ended. The `typescript.go` file emits named interfaces with typed methods:

```typescript
export type RunSpecLike = RunSpec | RunSpecBuilder | { toSpec(): RunSpec };

export interface RunSpecBuilder {
  experiment(id: string): this;
  backend(name: string): this;
  tag(...tags: string[]): this;
  meta(key: string, value: unknown): this;
  topology(fn: FragmentFn<TopologyBuilder>): this;
  workload(fn: FragmentFn<WorkloadBuilder>): this;
  policy(type: string, config?: JsonObject): this;
  policyCallback(id: string, callback: (task: unknown, candidateDeviceIds: string[], state: unknown, scores: Record<string, number>) => string | { deviceId: string }): this;
  metrics(fn: FragmentFn<MetricsBuilder>): this;
  use(fragment: FragmentFn<RunSpecBuilder>): this;
  validate(): ValidationResult;
  toSpec(): RunSpec;
  run(options?: RunOptions): RunResult;
}

export interface TopologyBuilder {
  device(id: string, type: string, config?: JsonObject): this;
  cpu(id: string, config?: { speed?: number; lanes?: number }): this;
  accelerator(id: string, config?: { speed?: number; bandwidthBytesPerNs?: number; setupNs?: number; lanes?: number }): this;
  jsDevice(id: string, callback: (phase: "estimate", task: unknown, state: unknown, fallback: unknown) => unknown, config?: JsonObject): this;
  use(fragment: FragmentFn<TopologyBuilder>): this;
}
```

`this` as the return type is what makes the chain checkable: TypeScript knows that `.topology(fn)` returns a `RunSpecBuilder`, so the next call must be a `RunSpecBuilder` method. `FragmentFn<T>` is a named type for the configurator, so a fragment written for `TopologyBuilder` cannot be passed where a `MetricsBuilder` fragment is expected. `RunSpecLike` is a union that accepts a builder, a plain spec, or anything with a `toSpec()` method, which is how a function can accept "anything that produces a run spec" without losing type information.

This is the compile-time-type target. The declarations narrow every option, every callback parameter, and every return type. A typo in `bandwidthBytesPerNs` is a compile error, not a silent no-op.

## Pattern G: Go-side typed builders via Proxy traps

Discord-bot's `require("ui")` module achieves typed builders through a different mechanism. Instead of attaching a hidden reference to a plain object, it surfaces Go-side builders through Goja's Proxy traps. A Proxy lets Go intercept property access, method calls, and construction on a JavaScript object, so the object's behavior is defined entirely by Go functions.

The `ui` module returns typed builders, not plain JavaScript objects. The documentation states the design rules directly:

- Wrong-parent calls fail loudly. `ui.message().field(...)` is an error, and the error tells the caller to use `ui.embed()`.
- Raw JavaScript objects are not accepted where builders are expected. A caller must pass a `ui.embed()` builder to `message.embed(...)`, not a plain object.
- The `.build()` terminal returns a typed Discord payload or the host's `normalizedResponse` fast path.

The caller experience is fluent:

```javascript
const ui = require("ui")

return ui.message()
  .content("Search results")
  .embed(ui.embed("Results").description("Found 3 items"))
  .row(ui.button("search:next", "Next", "primary"))
  .build()
```

The Proxy-trap approach has different tradeoffs than the hidden-key approach. A Proxy gives natural JavaScript ergonomics — the object behaves like a real object with methods — and the Go side can enforce structural rules that a plain object cannot express, like rejecting wrong-parent calls with a specific error. The cost is that Proxies are harder to introspect from Go: there is no hidden key to read back, because the Go state lives behind the trap handlers. The two mechanisms solve the same problem — giving Go a typed handle to recover — with different ergonomics.

Discord-bot pairs the `ui` builder DSL with a registration-style DSL for bot structure. `defineBot` takes a callback that receives a set of registration helpers:

```javascript
const { defineBot } = require("discord")

module.exports = defineBot(({ command, event, component, modal, autocomplete, configure }) => {
  configure({ name: "ping", description: "...", category: "examples" })
  command("ping", { description: "..." }, async (ctx) => { return { content: "pong" } })
  command("echo", { options: { text: { type: "string", required: true } } }, async (ctx) => { ... })
  event("ready", async (ctx) => { ... })
  component("ping:panel", async (ctx) => { ... })
  modal("feedback:submit", async (ctx) => { ... })
  autocomplete("search", "query", async (ctx) => { ... })
})
```

This is the registration pattern: the host calls the builder function once with a destructured set of helpers, and the builder body registers handlers by calling them. It is not a fluent chain. It is the right shape for a bot, where the structure is a flat list of handlers, not a nested configuration. The lesson is that a single project can combine patterns: a registration DSL for the flat structure, a fluent builder DSL for the nested payloads.

## The compile-time-type layer

The `tsgen/spec` package in `go-go-goja` is the substrate for all generated declarations. It models the TypeScript type system as Go structs:

```
type Module struct {
    Name        string
    Description string
    Functions   []Function
    RawDTS      []string
}

type Function struct {
    Name        string
    Description string
    Params      []Param
    Returns     TypeRef
}

type TypeRef struct {
    Kind  TypeKind   // string, number, boolean, any, unknown, void, never, named, array, union, object
    Name  string
    Item  *TypeRef
    Union []TypeRef
    Fields []Field
}
```

A module's `TypeScriptModule()` returns a `*spec.Module`. The render layer turns it into a `.d.ts`. Two authoring paths exist: a structured `Module` with `Function` / `Param` / `TypeRef`, or `RawDTS` strings for cases where hand-written declarations are clearer. Codesign uses `RawDTS` because its builder interfaces are easier to express as literal TypeScript than to assemble from `TypeRef` nodes. Geppetto generates its `.d.ts` from a `geppetto_codegen.yaml` via `cmd/gen-meta`.

The critical decision is not which authoring path to use. It is whether the declarations narrow types. A `TypeRef` of `Kind: TypeKindAny` is no better than `Record<string, any>`. A named `TypeRef` with `Fields` narrows. The playbook rule is: every builder and every terminal emits a named `TypeRef`, not `any`. Codesign's `RunSpecBuilder` interface is the reference. The widgetdsl `Props = Record<string, any>` is the anti-reference.

### The parity test as a contract

Generating a `.d.ts` is not enough. The declaration must match the runtime, or it is a lie. Geppetto's `TestGeneratedDTSMatchesRuntimeExportSurface` is the contract that makes the declaration trustworthy. It checks two things: that every name declared in the `.d.ts` exists at runtime, and that every name exported at runtime is declared. The first catches stale declarations; the second catches undocumented exports.

The test does not check types, only names. A name can be declared with the wrong type and the test will pass. A full parity test would also assert parameter and return types, which is feasible with the structured `Function` / `Param` / `TypeRef` model but not yet implemented in any surveyed DSL. This is an open opportunity for the playbook: extend the parity test to compare the structured `TypeRef` graph against the runtime function signatures, not just the names.

## Runtime ownership and lifecycle

Every DSL that holds a Go resource — a search index, a D-Bus connection, a database handle, a turn store — must define a lifecycle. The patterns in the survey converge on three rules.

The first rule is explicit close. Goja-dbus requires `await bus.close()`. Goja-bleve batches are single-use: once `.execute()` succeeds, later mutation or reset throws `"bleve: batch has already been executed"`. Geppetto turn stores carry a `closed` flag in their `refBase`. The common principle is that a resource handle has a terminal state, and operations after the terminal fail loudly rather than silently no-op.

The second rule is runtime ownership. A `goja.Runtime` is accessed by one goroutine at a time. The `go-go-goja` `engine` package provides a `RuntimeOwner` that serializes access. When Go invokes a JavaScript callback — a codesign policy callback, a dbus signal handler, a geppetto event emitter — it must schedule the invocation onto the runtime owner, not call `vm.RunString` or `fn(...)` directly from a worker goroutine. `goja-dbus` states this as an explicit invariant. Geppetto threads a `runtimeowner.RuntimeOwner` through its module `Options`. Codesign's callback registry depends on it: the simulator runs on its own goroutine, and callbacks must be marshalled back to the VM.

The third rule is context cancellation. The `engine` package distinguishes `WithStartupContext` (controls construction and initializers) from `WithLifetimeContext` (controls runtime-owned resources and cancellation after construction). A DSL that opens a connection in an initializer should tie the connection's context to the lifetime context, so that closing the runtime cancels the connection. Goja-dbus's `bus.close()` is the explicit path; the lifetime context is the implicit backstop.

## A comparison of the patterns

| Pattern | Type safety at boundary | Validation style | Compile-time types | Composition | Example |
| --- | --- | --- | --- | --- | --- |
| A: typed ref + same-object | `getTypedRef[T]` + `refKind` | terminal `(value, error)` | possible, not always emitted | chain only | goja-bleve |
| A′: typed ref + clone | `getTypedRef[T]` + `refKind` | mixed panic / error | DTS parity test | chain + branching | geppetto |
| B: plain Go builder struct | Go signatures only | `Validate()` + `Build()` | minimal descriptor | chain only | go-minitrace |
| C: map IR helpers | none | panic | `Record<string, any>` | nesting | widgetdsl `data.dsl` |
| F: lambda configurator + fragments | typed builders + `RunSpecLike` union | `.validate()` + `.toSpec()` | precise `interface` s | chain + `.use()` + lambdas | researchctl, codesign |
| G: Proxy traps | trap handlers enforce | `.build()` terminal | depends on declaration | chain | discord-bot `ui` |

The table makes the tradeoffs visible. Pattern C gives up everything for authoring ease. Pattern A maximizes boundary type safety but does not compose beyond chaining. Pattern F is the only one that combines type safety, validation, precise declarations, and lambda-based composition. Pattern G is an alternative typed-builder mechanism worth considering when Proxy ergonomics matter.

## Working rules

The following rules synthesize the patterns into guidance for a new DSL. They are not laws. Each is justified by a concrete failure in a surveyed DSL.

1. **Attach a hidden, non-enumerable Go reference to every handle.** Use a key like `__<module>_ref`. The Go side recovers the struct with a generic `getTypedRef[T]`. This is what makes the boundary typed. The widgetdsl magic-string `__ragSchema` tag is the cautionary example: it can be forged, because the handle carries no type.
2. **Tag every handle with a `refKind`.** A field builder and a field mapping are different types. The kind lets `getTypedRef` reject a built artifact where a builder is expected. Without it, a caller can pass a built mapping to a function expecting a builder, and the failure is delayed.
3. **Return the same object from chain methods, a new object from `.build()`.** Same-object return is what makes chaining natural. A different wrapper kind from the terminal is what makes the type transition enforceable.
4. **Prefer clone-on-each-step when a builder may be reused or branched.** Same-object mutation is allocation-efficient but single-use. If a DSL is driven programmatically and a builder might be built twice with different tails, clone per step. Geppetto is the model.
5. **Validate at terminals, return `(value, error)`, never panic.** Panicking gives the caller a single string with no structure and aborts on the first error. Returning `(value, error)` lets the caller handle failure. Accumulating errors in a slice and surfacing them via `Validate()` lets the caller see every problem at once. Go-minitrace is the model.
6. **Emit precise TypeScript interfaces, not `Record<string, any>`.** Every builder and terminal gets a named interface with typed methods and `this` return types. Every callback parameter gets a typed signature. Codesign's `RunSpecBuilder` is the reference. The widgetdsl `Props` is the anti-reference.
7. **Implement `modules.TypeScriptDeclarer` and run a DTS parity test.** The declaration is a contract. A test that asserts the declared names match the runtime exports keeps the contract honest. Geppetto's `TestGeneratedDTSMatchesRuntimeExportSurface` is the model. Extending it to compare types, not just names, is the open opportunity.
8. **Use lambda configurators for nested sub-builders.** `parent.child(title, c => c.id(...).status(...))` keeps the sub-builder scoped to the call that creates it. The Go side controls what builder the lambda receives. Researchctl and codesign are the models.
9. **Provide `.use(fragment)` for reusable configuration.** A `FragmentFn<T>` is a named function that configures a builder. Fragments are how callers build a library of reusable configurations without copy-paste. Codesign is the model.
10. **Validate runtime callbacks with `goja.AssertFunction` and invoke them on the runtime owner.** A callback stored for later invocation must be a function, and it must be scheduled onto the VM's event loop when Go invokes it, not called directly from a worker goroutine. Codesign's callback registry and goja-dbus's ownership rule are the models.
11. **Give every resource handle an explicit close and a terminal state.** A batch that has been executed throws on further mutation. A bus that has been closed refuses calls. The terminal state prevents use-after-close bugs from becoming silent no-ops.
12. **Thread a `runtimeowner.RuntimeOwner` through module `Options`.** The owner is the scheduler for all VM access. A DSL that fires callbacks from Go goroutines depends on it. Geppetto's `Options` struct is the model.

## Recommended implementation sequence

When building a new Goja fluent-builder DSL, work in this order. Each step has a concrete acceptance test.

1. **Define the Go domain structs.** Write the structs that hold the accumulated state — the equivalent of `fieldBuilderRef` or `RunSpec`. Add a `refBase` with a `refKind` to each. Acceptance: the structs compile and the kinds are unique.
2. **Implement `attachRef`, `getRef`, and `getTypedRef[T]`.** These are the boundary primitives. Copy them from goja-bleve or extract them into a shared `fluent` package. Acceptance: a round-trip test that attaches a ref, recovers it, and asserts the type.
3. **Implement the factory and chain methods.** Write the top-level factory (`field()`, `runSpec(name)`) and the chain methods. Decide same-object vs clone per step based on whether the builder will be reused. Acceptance: a JavaScript snippet chains and the final object has the expected `refKind`.
4. **Implement the terminal.** Add `.build()` or the domain-specific terminal. Validate accumulated state and return `(value, error)`. Acceptance: calling the terminal with missing required state returns a precise error.
5. **Add `Validate()` if the DSL has multi-error value.** Accumulate errors in a slice during chaining and surface them. Acceptance: a builder with three errors returns all three from `Validate()`.
6. **Implement the TypeScript declaration.** Implement `modules.TypeScriptDeclarer`. Emit named interfaces with `this` returns and typed callback params. Acceptance: a `.d.ts` is generated and a TypeScript file using the DSL type-checks without `any`.
7. **Add a DTS parity test.** Port geppetto's `TestGeneratedDTSMatchesRuntimeExportSurface`. Acceptance: adding an undocumented export or a stale declaration fails the test.
8. **Add lambda configurators for nested sub-builders.** Implement `applyBuilderCallback` and the collection methods that take `(title, fn)`. Acceptance: a sub-builder configured by a lambda produces the same struct as one configured by direct chain calls.
9. **Add `.use(fragment)` if reuse is a goal.** Type fragments as `FragmentFn<T>`. Acceptance: a fragment written for `TopologyBuilder` is rejected where a `MetricsBuilder` is expected.
10. **Add runtime callbacks if Go invokes JavaScript later.** Validate with `goja.AssertFunction`, store by identifier, invoke on the runtime owner. Acceptance: a callback fired from a worker goroutine executes on the VM without a race detector failure.

## Anti-patterns

- **Validating by panic.** A panic is a thrown error with a single string. It aborts on the first problem and gives the caller no structured way to collect failures. Replace with `(value, error)` terminals and an optional `Validate()` that accumulates.
- **Magic-string type tags.** Tagging a map with `__ragSchema: true` and checking it by key lookup is forgeable. Replace with a typed reference recovered by `getTypedRef`.
- **Open-ended TypeScript.** `Record<string, any>` and `[key: string]: any` disable checking for every option. Replace with named interfaces and typed fields, even at the cost of more declaration code.
- **Calling JavaScript callbacks directly from worker goroutines.** A `goja.Runtime` is not goroutine-safe. A callback invoked off the runtime owner races the VM. Always schedule onto the owner.
- **Mixing builder and artifact types.** If `.build()` returns the same wrapper kind as the builder, the boundary cannot distinguish a half-built from a built artifact. Always return a different `refKind` from the terminal.
- **Exposing Go resources as plain JavaScript objects.** A file handle or database connection placed directly on a JavaScript object can be closed from the wrong goroutine or leaked. Wrap it in a handle with a `refKind` and an explicit close.

## The remaining gap

The patterns now exist in working code. Codesign realizes all of the original goal: fluent builders, Go implementation, runtime typechecking, validation, compile-time types, composable grammar, and lambdas. Geppetto adds the DTS parity test. Goja-bleve provides the typed-reference substrate. Go-minitrace provides the validation discipline.

What does not exist is a shared library. Every DSL reimplements `attachRef`, `getTypedRef`, `mustSet`, and `applyBuilderCallback`. The substrate is copied across `goja-bleve`, `geppetto`, `researchctl`, and `codesign` with minor variations. The remaining work for the playbook is extraction: pull the typed-reference machinery into a `fluent` package in `go-go-goja`, pull the `FragmentFn` + `applyBuilderCallback` composition into a `fragments` package, and standardize the DTS parity test as a reusable test helper. Once that extraction exists, a new DSL starts from a typed substrate instead of reconstructing one.

## Related notes

- The base research catalogue and per-resource logbook are in docmgr ticket `GOJA-DSL-PLAYBOOK` at `rag-evaluation-system/ttmp/2026/07/05/GOJA-DSL-PLAYBOOK--goja-fluent-builder-dsl-playbook-base-research-and-resource-catalogue/`.
- [Widget DSL Grammar: Designing an Intent-Level UI Authoring Layer for a Widget IR System](https://parc.yolo.scapegoat.dev/note/projects/2026/07/05/article-widget-dsl-grammar-designing-an-intent-level-ui-authoring-layer-for-a-widget-ir-system) covers the `widgetdsl` grammar verbs in detail and is the cautionary example referenced in Pattern C.
- The `go-go-goja` `engine` package (`pkg/engine/`) defines the `RuntimeOwner` and the startup/lifetime context distinction that underpins the ownership rules.
- The `go-go-goja` `tsgen/spec` package (`pkg/tsgen/spec/types.go`) is the TypeScript declaration model every typed DSL should target.
