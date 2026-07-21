---
Title: Reproducible JavaScript task bundles and worker registries
Ticket: SCRAPER-WORKFLOW-V3
Status: active
Topics:
    - architecture
    - scheduler
    - goja
    - javascript
    - scraper
    - workflows
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/xgoja/sourcegraph/graph.go
      Note: Reusable source discovery and literal import resolution foundation
    - Path: repo://pkg/engine/runner/runner.go
      Note: Current string-keyed mutable runner registry that v3 must version seal and advertise
    - Path: repo://pkg/js/runtime/executor.go
      Note: Current per-operation Goja runtime and script loading baseline
    - Path: repo://pkg/sites/manifest/loader.go
      Note: Strict site manifest and script-root loading baseline
    - Path: repo://pkg/workflow/executor.go
      Note: Existing Go executor-to-runner adapter demonstrating language-independent engine contracts
    - Path: repo://ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/scripts/05-js-task-bundle-registration-probe.mjs
      Note: Executable registration sealing and exact matching probe
ExternalSources: []
Summary: Design and first implementation for domain-authored JavaScript task bundles that workers verify, seal, advertise, execute in fresh lease-scoped runtimes, and pin by immutable implementation digest and module profile.
LastUpdated: 2026-07-21T22:30:00Z
WhatFor: Replace hypothetical built-in domain task modules with a robust extension system where developers can ship custom JavaScript task descriptors and implementations reproducibly across workers.
WhenToUse: Read before implementing workflow task catalogs, JavaScript task execution, worker capability advertisement, task bundle packaging, dynamic registry reload, or custom domain tasks.
---


# Reproducible JavaScript task bundles and worker registries

## Executive answer

Yes. Domain developers should be able to provide custom JavaScript tasks, and workers should be able to populate their task registries from those JavaScript packages. That capability is essential if workflow v3 is to remain generic rather than accumulating every `data.tasks.*`, `web.tasks.*`, or organization-specific operation in scraper's Go tree.

The robust design is **not** an unrestricted global side effect where any `require()` call mutates the process-wide runner registry. Instead, use explicit, immutable **JavaScript task bundles**:

1. a developer authors a bundle manifest, task catalog, schemas, and execution modules;
2. a build command resolves imports, bundles dependencies, validates descriptors/tests, and emits a content-addressed artifact;
3. each worker is configured with an immutable bundle lock or signed bundle references;
4. worker startup fetches and verifies each exact artifact, evaluates its catalog in a registration-only Goja runtime, constructs a candidate registry, runs self-tests, and then atomically seals/activates that registry generation;
5. the worker advertises task keys **and exact implementation digests**;
6. the workflow compiler binds each task descriptor to an approved implementation digest;
7. the dispatcher leases a node only to a worker advertising that exact implementation and required capabilities;
8. each attempt executes the entrypoint in a fresh lease-scoped Goja runtime with only allowlisted modules and narrow `workflow/task` services;
9. host wrappers validate inputs and outputs before persistence;
10. rolling upgrades add a new immutable version/digest instead of replacing code beneath an active run.

The developer experience can still feel like “load this JS and register these tasks”:

```bash
scraper task-bundle build ./customer-tasks/task-bundle.yaml
scraper worker --task-bundle sha256:4c7f...
```

The worker internally loads `catalog.js` and registers its task entries, but registration happens in an explicit boot/reload transaction. Ordinary workflow authoring and task execution cannot mutate the registry.

## Why this is needed

The cookbook intentionally uses hypothetical domain modules:

```js
const data = require("data");

data.tasks.normalizeCustomers({source});
data.tasks.joinDatasets({left, right, on: ["customerId"]});
```

If every such task must be implemented and compiled into scraper as Go, scraper becomes a monolith and domain iteration requires a core release. If each workflow instead embeds arbitrary processing callbacks, scraper loses schema validation, resource planning, reproducibility, retry semantics, worker matching, and security boundaries.

Task bundles preserve both goals:

- scraper remains generic durable infrastructure;
- domain developers can add rich behavior in JavaScript;
- plans remain data-only and immutable;
- workers execute exact, verifiable implementations.

## Current-state evidence and gaps

Scraper already has pieces that make the design feasible, but not the complete contract.

### Current runner registry

`pkg/engine/runner/runner.go` defines a `Runner` with `Kind() string` and a registry keyed only by that string. `Register` rejects duplicate kinds and `Get` retrieves one runner. This is a useful base abstraction but it lacks:

- task version;
- implementation/bundle digest;
- input/output schemas;
- capability/resource requirements;
- registry generation or sealing;
- worker capability advertisement;
- rolling coexistence of old and new implementations.

The workflow-native registry in `pkg/workflow/executor.go` adapts Go `Executor` values into the same runner registry. It demonstrates that multiple implementation languages can share one engine-facing runner contract.

### Current JavaScript execution

`pkg/js/runtime/executor.go` already creates a Goja runtime per operation, installs configured modules, loads one script by path, calls its exported function, awaits a promise, and converts the result. Per-operation runtime creation is a good isolation default for v3.

The current operation chooses the script through mutable operation metadata and site-relative paths. That is not sufficient for v3 because a durable plan does not pin the script content digest, dependency graph, schemas, task ABI, or implementation identity.

### Current site packaging

Site manifests declare script/verb roots and a small module list (`pkg/sites/manifest/manifest.go`, `loader.go`). The manifest loader uses strict YAML fields, and xgoja's source graph already discovers JavaScript/TypeScript files, resolves literal imports, rejects dynamic non-literal imports, and records source origins.

Those mechanisms can inform task-bundle construction. A task bundle needs stronger content identity, dependency locking, task schemas, trust policy, and worker advertisement than a current site script root.

### xgoja provider boundary

xgoja/v2 providers package **Go-native runtime modules** selected into generated hosts. A JavaScript task bundle should not generate a new Go provider for every domain task. Instead:

- scraper ships generic native modules such as `workflow/task-bundle` and `workflow/task` through an xgoja provider;
- domain bundles are runtime artifacts consumed by those generic modules;
- a bundle may include a generated authoring-side CommonJS module and TypeScript declarations;
- worker host configuration selects the narrow native capabilities the bundle is allowed to import.

## Design goals

### Functional goals

- Domain developers can add task kinds without changing scraper core.
- One bundle can provide many related tasks.
- The same bundle can provide authoring descriptor factories, execution entrypoints, schemas, help, and tests.
- Every worker can load an exact bundle set reproducibly.
- Workers advertise only successfully verified/loaded implementations.
- Plans pin exact implementation identities.
- Old and new versions can coexist during rolling upgrades and run resumption.
- JavaScript tasks and Go tasks share the same plan/node/attempt contracts.

### Robustness goals

- Registration is atomic and fail-closed.
- Task execution cannot mutate the registry.
- A failed bundle does not partially register tasks.
- A bundle cannot replace an existing `kind@version` with different bytes.
- Node input/output schema validation is enforced by Go wrappers.
- One attempt gets one lease-scoped runtime and capability set.
- Restarting a worker reproduces the same registry digest from the same lock file.
- Missing implementation is visible as a capability/admission problem, not a random “module not found” during execution.

### Security goals

- Safe authoring runtimes do not receive task execution authority.
- Task runtimes receive only declared, host-approved native modules/services.
- Bundle code cannot choose arbitrary filesystem paths, network clients, database handles, or store locators.
- Secrets remain in host services and never enter bundle manifest, plan, node input, event, or report.
- Resource, timeout, output-size, artifact, and budget ceilings are enforced outside JavaScript.
- Bundle provenance and signer/trust decisions are auditable.

### Non-goals

- Treating Goja as a secure sandbox for hostile code in the same process.
- Supporting runtime `npm install` or network dependency resolution.
- Allowing a task to register more tasks while executing.
- Replacing exact implementation pinning with “latest” resolution.
- Guaranteeing exactly-once external side effects without domain idempotency.
- Persisting JavaScript closures in workflow IR or node rows.

## Critical security statement: Goja is not an isolation boundary

Goja gives language/runtime isolation, not a complete hostile-code security sandbox. A custom task can consume CPU/memory, exploit a dangerous native module, or trigger bugs in the host process. Therefore:

- bundles from the same trusted engineering domain may execute in-process with strict capabilities and deadlines;
- mutually untrusted, third-party, or user-uploaded bundles must execute in a separate worker process/container/VM with OS-level CPU, memory, filesystem, network, and syscall controls;
- module allowlists and Goja interruption are defense in depth, not a substitute for process isolation.

The worker advertisement should include an isolation class such as `trusted-inprocess` or `sandboxed-subprocess`. The compiler/policy can reject a bundle whose trust class requires stronger isolation than the available worker pool.

## Core representation

### Task key

```go
type TaskKey struct {
    Kind    string `json:"kind"`    // namespaced: acme.customer.normalize
    Version string `json:"version"` // immutable API version: v1
}
```

Kind names are globally namespaced. Versions describe the task contract. A given `kind@version` must have immutable schemas and semantics.

### Implementation identity

```go
type ImplementationID struct {
    Language       string `json:"language"`       // javascript | go
    BundleDigest   string `json:"bundleDigest"`   // sha256:...
    Entrypoint     string `json:"entrypoint"`     // ./tasks/normalize.js#run
    EntrypointHash string `json:"entrypointHash"` // optional direct content hash
    ABIVersion     string `json:"abiVersion"`     // scraper-js-task/v1
}
```

The bundle digest covers the complete executable artifact and canonical manifest. The compiled plan binds `TaskKey` to `ImplementationID`.

### Task catalog entry

```go
type CatalogEntry struct {
    Key                   TaskKey
    Implementation        ImplementationID
    InputSchema           SchemaRef
    Outputs               []OutputPort
    ConfigSchema          SchemaRef
    RequiredCapabilities  []CapabilityRequest
    AllowedModules        []string
    DefaultResourceClass  string
    TimeoutCeiling        time.Duration
    OutputByteCeiling     int64
    Semantics             TaskSemantics
    FailureCodes          []FailureCodeDescriptor
}

type TaskSemantics struct {
    Determinism string // deterministic | externally-dependent
    Idempotency string // pure | keyed-side-effect | non-idempotent
    SideEffects []string
}
```

The host validates catalog entries. JavaScript cannot assert a capability the worker does not provide or increase a ceiling above policy.

### Task bundle reference

```go
type TaskBundleRef struct {
    SchemaVersion  string `json:"schemaVersion"`  // scraper-task-bundle-ref/v1
    Name           string `json:"name"`
    Version        string `json:"version"`
    Digest         string `json:"digest"`
    Locator        string `json:"locator"`
    ManifestDigest string `json:"manifestDigest"`
    SignatureRef   string `json:"signatureRef,omitempty"`
}
```

A lock file contains references, not mutable directories:

```yaml
schemaVersion: scraper-task-bundle-lock/v1
bundles:
  - name: acme-customer-tasks
    version: 1.4.2
    digest: sha256:4c7f0d...
    manifestDigest: sha256:933e11...
    locator: registry://internal/task-bundles/sha256:4c7f0d...
    signatureRef: sigstore://...
```

## Developer-facing bundle layout

```text
customer-tasks/
  task-bundle.yaml
  catalog.js
  authoring.js
  schemas/
    customer-export-ref.v1.json
    normalized-customer-dataset-ref.v1.json
    normalize-input.v1.json
  tasks/
    normalize-customers.js
    validate-customers.js
    join-customer-orders.js
  tests/
    normalize-customers.test.js
    fixtures/
      small-export.ref.json
  package.json
  pnpm-lock.yaml
```

The lock file is consumed at build time. Runtime workers never run package-manager resolution.

## Bundle manifest

```yaml
schemaVersion: scraper-task-bundle-source/v1
name: acme-customer-tasks
version: 1.4.2
namespace: acme.customer
abiVersion: scraper-js-task/v1
catalog: ./catalog.js
authoringModule:
  name: acme-customer
  entrypoint: ./authoring.js
sources:
  include:
    - catalog.js
    - authoring.js
    - tasks/**/*.js
    - schemas/**/*.json
  exclude:
    - tests/**
imports:
  runtimeModules:
    - workflow/task
    - data/records
permissions:
  isolation: trusted-inprocess
  artifacts:
    readSchemas:
      - customer-export-ref/v1
      - order-dataset-ref/v1
    writeSchemas:
      - normalized-customer-dataset-ref/v1
  network: none
  databases: none
selfTests:
  - ./tests/normalize-customers.test.js
```

Unknown fields fail. `permissions` is a request; build/worker policy may narrow or reject it.

## Explicit catalog registration API

The catalog is JavaScript, but registration is explicit and data-oriented:

```js
const taskBundle = require("workflow/task-bundle");

module.exports = taskBundle.define({
  name: "acme-customer-tasks",
  version: "1.4.2",
  namespace: "acme.customer",
  abiVersion: "scraper-js-task/v1",
}, bundle => {
  bundle.task({
    kind: "acme.customer.normalize",
    version: "v1",
    entrypoint: "./tasks/normalize-customers.js#run",
    inputSchema: "normalize-customer-input/v1",
    outputs: {
      dataset: "normalized-customer-dataset-ref/v1",
      report: "data-quality-report-ref/v1",
    },
    configSchema: "normalize-customer-config/v1",
    defaultResource: "cpu.transform",
    timeoutCeiling: "10m",
    outputByteCeiling: 64_000,
    semantics: {
      determinism: "deterministic",
      idempotency: "pure",
      sideEffects: ["artifact-write"],
    },
    capabilities: [
      {name: "artifacts", mode: "read-write"},
    ],
    modules: ["workflow/task", "data/records"],
  });

  bundle.task({
    kind: "acme.customer.validate",
    version: "v1",
    entrypoint: "./tasks/validate-customers.js#run",
    inputSchema: "validate-customer-input/v1",
    outputs: {
      acceptedDataset: "validated-customer-dataset-ref/v1",
    },
    defaultResource: "cpu.transform",
    timeoutCeiling: "5m",
    semantics: {
      determinism: "deterministic",
      idempotency: "pure",
      sideEffects: ["artifact-read"],
    },
    capabilities: [{name: "artifacts", mode: "read"}],
    modules: ["workflow/task", "data/records"],
  });
});
```

`taskBundle.define` executes only in registration/build runtimes. It invokes the callback immediately and returns a frozen catalog object. It does not mutate a package-global Go registry.

### Why entrypoint strings are preferable to stored function references

The catalog names `./tasks/normalize-customers.js#run` rather than embedding an execution closure:

- import/source graph can be resolved statically;
- the bundle builder can hash and validate the module;
- catalog evaluation does not retain a live Goja runtime;
- workers can create a fresh runtime per attempt;
- authoring metadata can be read without loading execution code;
- stack traces and provenance have stable module names.

The entrypoint string is safe only because it is resolved inside the immutable bundle, not against an arbitrary host filesystem path.

## Task implementation API

```js
const task = require("workflow/task");
const records = require("data/records");

exports.run = task.implementation(async ctx => {
  const input = ctx.input();
  const config = ctx.config();

  const source = await ctx.artifacts.openJSONLines(input.source, {
    schema: "customer-export-ref/v1",
    maxBytes: config.maxInputBytes,
  });

  const writer = await ctx.outputs.createJSONLines("dataset", {
    schema: "normalized-customer-dataset-ref/v1",
  });

  let seen = 0;
  let accepted = 0;
  for await (const raw of source) {
    ctx.checkpoint();
    seen += 1;
    const normalized = records.normalize(raw, {
      fields: config.fields,
      unknown: "reject",
    });
    await writer.write(normalized);
    accepted += 1;

    if (seen % 1000 === 0) {
      ctx.progress({completed: seen, unit: "records"});
    }
  }

  const dataset = await writer.commit();
  const report = await ctx.outputs.putJSON("report", {
    schema: "data-quality-report-ref/v1",
    value: {seen, accepted, rejected: seen - accepted},
  });

  return task.success({dataset, report});
});
```

The implementation receives a lease-scoped object. It does not receive the engine store, workflow registry, raw lease token, credentials, or an operation-emission API.

### Task context surface

Recommended capabilities:

```ts
interface TaskContext<I, C, O> {
  input(): Readonly<I>;
  config(): Readonly<C>;
  identity(): Readonly<{runId: string; nodeKey: string; attempt: number}>;
  checkpoint(): void; // throws on cancellation/deadline/lease loss
  progress(update: ProgressUpdate): void;
  artifacts: LeaseScopedArtifacts;
  outputs: ValidatingOutputWriter<O>;
  log: RedactingLogger;
  clock?: DeterministicTaskClock;
}
```

`clock` is omitted by default for deterministic tasks. Randomness, if needed, is an explicit seeded capability. Direct `Date.now`, ambient environment, filesystem, process, and network modules should not be available unless policy explicitly provides a wrapper.

### Typed failures

```js
const task = require("workflow/task");

exports.run = task.implementation(async ctx => {
  try {
    return await doWork(ctx);
  } catch (error) {
    if (isMalformedDomainInput(error)) {
      throw task.failure({
        class: "validation",
        code: "ACME_CUSTOMER_INVALID_RECORD",
        retryable: false,
        message: "customer export failed schema validation",
      });
    }
    throw error;
  }
});
```

The host converts uncategorized exceptions to a redacted `internal` failure. Stack traces may enter secured worker logs but not durable event/error payloads.

## Authoring-side descriptor module

Workflow authors should not hand-write raw task descriptors. The bundle can publish an authoring module generated from the catalog plus optional ergonomic factories:

```js
// authoring.js — evaluated in the safe authoring runtime
const descriptors = require("workflow/task-descriptors");

exports.tasks = {
  normalizeCustomers(options) {
    return descriptors.task({
      kind: "acme.customer.normalize",
      version: "v1",
      config: {
        source: options.source,
        fields: options.fields || "standard/v1",
        maxInputBytes: options.maxInputBytes || 1_000_000_000,
      },
    });
  },

  validateCustomers(options) {
    return descriptors.task({
      kind: "acme.customer.validate",
      version: "v1",
      config: {dataset: options.dataset},
    });
  },
};
```

A workflow then uses:

```js
const workflow = require("workflow");
const customer = require("acme-customer");

const definition = workflow.define("customer-import", p => {
  const source = p.input("source", {schema: "customer-export-ref/v1"});
  p.resource("cpu", r => r.class("cpu.transform").maxInFlight(4));

  const normalized = p.task("normalize",
    customer.tasks.normalizeCustomers({source}),
    j => j.resource("cpu"));

  const validated = p.task("validate",
    customer.tasks.validateCustomers({
      dataset: normalized.output("dataset"),
    }),
    j => j.after(normalized).resource("cpu"));

  p.output("dataset", validated.output("acceptedDataset"));
});

module.exports = workflow.compile(definition);
```

The authoring module is data-only. Requiring `acme-customer` does not make its execution entrypoints callable and does not grant artifact access.

## Bundle build pipeline

```text
source tree + lock file + bundle manifest
                 │
                 ▼
1. strict manifest decode
2. discover files and resolve literal imports
3. reject dynamic/unknown imports
4. bundle/vendor JavaScript and TypeScript
5. evaluate catalog in registration-only Goja runtime
6. validate task keys, schemas, entrypoints, modules, capabilities, semantics
7. compile/check every entrypoint
8. run bundle self-tests in a test capability host
9. generate authoring descriptor module + TypeScript declarations + help
10. canonicalize manifest/catalog
11. produce SBOM/provenance
12. archive deterministic file tree
13. compute content and manifest digests
14. sign/attest according to publisher policy
                 │
                 ▼
immutable TaskBundle artifact + TaskBundleRef
```

Proposed command:

```bash
scraper task-bundle build ./task-bundle.yaml \
  --out ./dist/acme-customer-tasks.bundle \
  --emit-lock ./dist/task-bundles.lock.yaml
```

### Deterministic bundle contents

```text
bundle/
  manifest.json             canonical built manifest
  catalog.json              normalized task catalog, no JS functions
  execution.cjs             deterministic execution bundle
  authoring.cjs             safe descriptor-only module
  index.d.ts                generated declarations
  schemas/...               canonical schemas
  help/...                  generated/curated help
  sbom.spdx.json
  provenance.json
  self-test-report.json
```

Source maps may be included in a separate secured debugging artifact to avoid leaking source into ordinary worker reports.

### Dependency policy

- Resolve dependencies at build time only.
- Commit and verify the package-manager lock.
- Bundle allowed pure-JS dependencies into `execution.cjs` or include them by digest.
- Never run `npm install`/`pnpm install` on worker startup.
- Reject native Node addons; Goja cannot execute them and they compromise portability.
- Reject dynamic `require(variable)` unless a finite mapping is statically declared and included.
- Record tool versions and source graph digest in provenance.

## Worker configuration

```yaml
schemaVersion: scraper-worker/v3
workerId: worker-cpu-01
isolationClass: trusted-inprocess
resources:
  cpu.transform: 6
bundleLock: /etc/scraper/task-bundles.lock.yaml
trust:
  allowedPublishers:
    - keyless://acme-engineering/task-bundles
  requireSignature: true
  allowUnsignedLocalDevelopment: false
runtime:
  allowedNativeModules:
    - workflow/task
    - data/records
limits:
  taskTimeoutMax: 1h
  outputBytesMax: 1048576
  progressEventsPerMinute: 12
```

Local development may use a disk bundle with an explicitly different trust/isolation profile. Production plans must not accidentally bind to a local-development implementation digest.

## Worker boot and registration algorithm

```go
func BuildRegistry(ctx context.Context, lock BundleLock, policy Policy) (*RegistryGeneration, error) {
    candidate := NewMutableCandidate()

    for _, ref := range lock.Bundles {
        artifact := fetchByDigest(ref)
        verifyDigest(artifact, ref.Digest)
        verifySignatureAndPublisher(artifact, policy)
        manifest := strictDecodeManifest(artifact)
        verifyABI(manifest.ABIVersion)
        verifyIsolation(manifest, policy)
        verifySourceAndCatalogDigests(artifact, manifest)

        catalog := evaluateCatalogInRegistrationRuntime(artifact, policy.RegistrationModules)
        validateCatalogAgainstManifest(catalog, manifest)
        verifyEntrypointsAndImports(artifact, catalog, policy)
        runSelfTests(ctx, artifact, catalog, policy.SelfTestHost)

        for _, entry := range catalog.Tasks {
            candidate.Register(entry, NewJSRunnerFactory(artifact, entry))
        }
    }

    candidate.ValidateNoConflicts()
    generation := candidate.Seal()
    generation.Digest = CanonicalDigest(generation.CapabilityManifest())
    return generation, nil
}
```

Only after the entire candidate succeeds does the worker become ready and advertise the generation. One invalid bundle prevents that configured generation from activating; it does not leave half of its tasks registered.

### “Loading JS registers tasks” semantics

The ergonomic statement is true at one controlled point:

```text
worker bootstrap/reload
   → load immutable bundle artifact
   → evaluate catalog.js in registration VM
   → register normalized entries in candidate registry
   → seal and activate
```

It is false during ordinary module evaluation in workflow authoring or task execution. Those phases cannot access the candidate/active registry mutation API.

## Immutable registry generations

```go
type RegistryGeneration struct {
    ID             uint64
    Digest         string
    CreatedAt      time.Time
    Implementations map[TaskKey]Implementation
    BundleRefs     []TaskBundleRef
}
```

Once active, a generation is immutable. Attempts acquire a generation reference before runtime creation and release it after completion. A reload builds a new generation beside the old one, then atomically switches new leases to the new advertised set. Old generations remain until active attempts drain and any plans pinned to them are no longer assigned to that worker.

Do not mutate a map used by running attempts.

## Worker capability advertisement

```json
{
  "schemaVersion": "scraper-worker-capabilities/v3",
  "workerId": "worker-cpu-01",
  "registryDigest": "sha256:registry...",
  "isolationClass": "trusted-inprocess",
  "resources": {"cpu.transform": 6},
  "tasks": [
    {
      "kind": "acme.customer.normalize",
      "version": "v1",
      "implementation": {
        "language": "javascript",
        "bundleDigest": "sha256:4c7f0d...",
        "entrypoint": "./tasks/normalize-customers.js#run",
        "abiVersion": "scraper-js-task/v1"
      },
      "capabilityDigest": "sha256:taskcap..."
    }
  ],
  "expiresAt": "2026-07-21T23:00:00Z"
}
```

Advertisements are heartbeated with bounded TTL. The scheduler uses store-backed worker/resource capability state. A stale advertisement is not eligible for leasing.

## Compiler binding

The workflow authoring descriptor says:

```json
{
  "kind": "acme.customer.normalize",
  "version": "v1",
  "config": {"source": {"$ref": "input", "key": "source"}}
}
```

The compiler selects an approved catalog entry and records:

```json
{
  "task": {"kind": "acme.customer.normalize", "version": "v1"},
  "implementation": {
    "language": "javascript",
    "bundleDigest": "sha256:4c7f0d...",
    "entrypoint": "./tasks/normalize-customers.js#run",
    "abiVersion": "scraper-js-task/v1"
  },
  "schemas": {
    "input": "normalize-customer-input/v1",
    "outputs": {"dataset": "normalized-customer-dataset-ref/v1"}
  },
  "effectiveCapabilities": ["artifacts:read-write"],
  "effectiveResource": "cpu.transform"
}
```

The implementation digest is part of the compiled plan digest. A worker with the same task key but a different bundle digest is not an equivalent executor for that plan.

### Catalog source for compilation

Compilation should use a signed/approved catalog snapshot, not a race-prone live worker list. The catalog snapshot says which implementations policy permits. Fleet advertisements answer whether capacity currently exists. Thus:

- **catalog**: valid implementation choices and schemas;
- **worker advertisement**: currently available exact implementations/resources;
- **compiled plan**: chosen implementation;
- **dispatcher**: match plan requirement to live worker.

A plan may compile while workers are temporarily offline, but submission/operator policy can require minimum available capacity.

## Lease and execution flow

```text
compiled node requires acme.customer.normalize/v1
bundle digest sha256:4c7f...
resource cpu.transform
              │
              ▼
dispatch query finds worker advertising exact requirement
              │
              ▼
transaction reserves resource/budget and creates lease + attempt
              │
              ▼
worker resolves active registry generation by implementation ID
              │
              ▼
runner factory creates fresh Goja runtime
  - immutable bundle loader
  - workflow/task module bound to lease
  - catalog-approved pure modules
  - no authoring/submission/operator module
              │
              ▼
require exact entrypoint and call exported run
              │
              ▼
input schema validated before call
output writers enforce port schema/size
promise awaited under context deadline
              │
              ▼
worker returns refs + typed usage/failure
              │
              ▼
store validates lease/cancel epoch and commits attempt/node/output
```

## Runtime construction and caching

Use a fresh `goja.Runtime` per attempt. Do not share global JavaScript state across attempts. For performance, workers may cache immutable parsed/compiled Goja `Program` objects and immutable bundle bytes keyed by bundle digest if goja's APIs permit safe reuse, but each attempt receives:

- a fresh runtime/global object;
- a fresh CommonJS module cache;
- a fresh lease-scoped task service;
- context cancellation/interrupt wiring;
- per-attempt logs/progress limits.

A runtime pool is acceptable only if it proves complete state reset; fresh construction is the safer first implementation.

## Input and output enforcement

The JavaScript implementation is not trusted to self-validate.

Before invocation, Go:

1. decodes compact input through the catalog schema;
2. validates all references and config;
3. checks effective capability/resource policy;
4. constructs lease-scoped services.

During/after execution, Go:

1. validates each output port name;
2. validates reference schema, digest, size, and provenance;
3. enforces output count/byte limits;
4. rejects undeclared outputs;
5. redacts/normalizes failures and usage;
6. persists only after all required outputs validate.

A task cannot return arbitrary bytes in an operation result as an escape hatch.

## Native module and capability model

### Always-generic native modules

Scraper provides through its xgoja provider:

- `workflow/task-bundle` — registration/build phase only;
- `workflow/task-descriptors` — safe authoring phase;
- `workflow/task` — execution phase, lease-scoped;
- selected pure utility modules such as deterministic codecs.

### Bundle-requested native modules

A bundle declares modules by stable alias. Worker policy maps aliases to exact xgoja provider modules and host services. Both catalog and worker capability digest record the selection.

For example, `data/records` can be a pure bundle-local module. Trusted
first-party execution bundles may also request current go-go-goja host modules
through policy-bound aliases:

- `fetch:partner` or `fetch:public` selects a configured client with bounded
  endpoints, redirects, response sizes, timeouts, and host-owned credentials;
- `db:source` or `db:destination` selects a Go-preconfigured database handle
  with JavaScript `configure()` disabled;
- `fs:input`, `fs:workspace`, and `fs:output` select read-only or attempt-local
  mounts;
- `exec:media` or `exec:build` selects an allowlisted command profile and must
  run in process/container isolation;
- `crypto`, `path`, `yaml`, and `time` use the current pure or bounded utility
  contracts.

The suffix is a configured module instance, not a free-form authority string.
Bundle catalogs enumerate the aliases, compiled jobs retain them, and workers
advertise the exact selected module-profile digest. Authoring and registration
runtimes receive none of these execution authorities.

### Runtime phase matrix

| Module | Build/registration | Workflow authoring | Task attempt |
|---|---:|---:|---:|
| `workflow/task-bundle` | yes | no | no |
| `workflow/task-descriptors` | optional | yes | no |
| generated `acme-customer` authoring module | no | yes | no |
| `workflow` | no | yes | no |
| `workflow/task` | no | no | yes |
| task execution entrypoint | compile/check only | no | yes |
| `fetch:*`, `db:*`, `fs:*` | no | no | explicit trusted bundle grant |
| `exec:*` | no | no | explicit isolated worker grant |
| `crypto`, `path`, `yaml`, `time` | no | no | declared task profile only |
| `workflow/submit` | no | trusted control only | no |
| `workflow/operator` | no | admin only | no |

This phase split prevents execution code from registering tasks or authoring code from reading artifacts.

## Updating bundles and rolling workers

### Immutability rule

Never publish different bytes under the same bundle digest. Never silently change implementation for an existing compiled plan.

Preferred versioning:

- contract-breaking schema/semantics: new task version (`v2`);
- bug fix preserving contract: new bundle/package version and implementation digest; compiler policy decides whether new plans bind it;
- active old plans remain pinned to old digest.

### Rolling sequence

1. publish and approve new bundle digest;
2. add digest to worker lock/config;
3. workers build/verify candidate registry generation;
4. workers advertise old and new implementation digests if both are retained;
5. compiler begins binding new plans to new digest;
6. old runs continue on workers advertising old digest;
7. after old plans/runs drain or migrate explicitly, remove old digest;
8. workers reload and stop advertising it.

If no worker can execute an old pinned implementation, the run becomes `capability-blocked`; it must not substitute new code automatically.

## Reload behavior

A file watcher is useful for local development but not the production identity contract.

### Development

```bash
scraper worker --dev-task-bundle ./customer-tasks --watch
```

Every change rebuilds a new ephemeral digest and registry generation. The UI clearly marks runs non-production/local. Active attempts finish on their acquired generation.

### Production

Reload is triggered by a new immutable lock/config generation. Worker fetches and validates complete candidate state before atomic activation. Failed reload leaves the prior generation active and emits a redacted operator event.

## Side effects and idempotency

JavaScript flexibility does not create exactly-once delivery. For tasks that call external systems:

- catalog declares `idempotency: keyed-side-effect`;
- engine supplies stable idempotency key derived from run/node logical identity, not attempt number;
- domain client passes it where supported;
- attempts record ambiguous timeout outcomes;
- retry policy is conservative when remote completion is unknown;
- output schema can represent `confirmed`, `unknown`, or `reconciled` state.

A bundle declaring `non-idempotent` side effects may be rejected from automatic retry or require operator policy.

## Reproducibility chain

A completed attempt should be attributable to:

```text
workflow definition digest
  + compiled plan digest
  + task key/version
  + bundle digest
  + entrypoint
  + ABI version
  + catalog/schema digests
  + selected native module/provider digests
  + worker registry digest
  + immutable input refs
  + attempt/lease facts
  = reproducible execution identity
```

This does not mean external network responses are deterministic. It means the exact code, declared contract, host capabilities, and inputs are knowable.

## Error handling and quarantine

### Bundle build failure

Build fails before publication. No digest is approved.

### Worker verification failure

Worker does not advertise the bundle. It records a safe operator diagnostic such as `TASK_BUNDLE_SIGNATURE_INVALID` or `TASK_BUNDLE_ABI_UNSUPPORTED`. Prior active registry remains if this was a reload.

### Catalog conflict

Two configured bundles cannot register the same `TaskKey` with different implementations in one registry generation unless an explicit host policy selects one. Default is fail-closed.

### Runtime construction failure

If a worker advertised an implementation but cannot create its runtime, the attempt fails `internal/TASK_RUNTIME_CONSTRUCTION`. Repeated failures should quarantine that implementation/worker generation and withdraw advertisement rather than burn all task retries.

### Output validation failure

The attempt fails `validation/TASK_OUTPUT_SCHEMA`. No output/artifact reference is committed as authoritative. Retry follows explicit policy; deterministic output-schema bugs normally should not retry repeatedly.

## Observability

Worker-level projections:

- active registry generation/digest;
- configured, loaded, rejected, and quarantined bundle counts;
- task implementations by language/version/digest (operator API, not high-cardinality Prometheus labels);
- bundle load/self-test duration;
- runtime construction failures;
- capability-blocked nodes;
- active JS runtimes and execution duration;
- cancellation interrupt latency;
- output validation failures.

Attempt records include bundle/entrypoint identity. Prometheus labels should use bounded task kind/version and language, not full digest or node key.

## Suggested Go package boundaries

```text
pkg/workflowtask/
  key.go                 TaskKey and descriptor contracts
  catalog.go             normalized catalog and validation
  implementation.go      ImplementationID and semantics

pkg/taskbundle/
  manifest.go            strict source/built manifest
  build.go               deterministic build pipeline
  artifact.go            archive and digest handling
  trust.go               signatures/publisher policy
  lock.go                bundle lock

pkg/taskbundle/goja/
  catalog_loader.go       registration-only Goja evaluation
  authoring_module.go     generated descriptor module loader
  runner_factory.go      execution runtime factory
  task_context.go        lease-scoped workflow/task adapter

pkg/workerregistry/
  candidate.go            mutable boot/reload candidate
  generation.go           immutable active registry
  advertisement.go        worker capability manifest
  reload.go               atomic generation swap/drain

pkg/engine/v3/dispatcher/
  capability_match.go     exact TaskKey/Implementation/resource matching
```

Keep bundle models/build/trust logic independent of goja where possible. Goja adapters decode catalog values and invoke task entrypoints; they do not own task semantics.

## Store additions

Logical tables or equivalent store contracts:

```sql
CREATE TABLE worker_generations (
  worker_id TEXT NOT NULL,
  generation_digest TEXT NOT NULL,
  isolation_class TEXT NOT NULL,
  capabilities_json TEXT NOT NULL,
  advertised_at_us INTEGER NOT NULL,
  expires_at_us INTEGER NOT NULL,
  PRIMARY KEY(worker_id, generation_digest)
);

CREATE TABLE worker_task_implementations (
  worker_id TEXT NOT NULL,
  generation_digest TEXT NOT NULL,
  task_kind TEXT NOT NULL,
  task_version TEXT NOT NULL,
  bundle_digest TEXT NOT NULL,
  entrypoint TEXT NOT NULL,
  abi_version TEXT NOT NULL,
  capability_digest TEXT NOT NULL,
  PRIMARY KEY(worker_id, generation_digest, task_kind, task_version, bundle_digest, entrypoint)
);
```

Node/plan requirements include exact implementation fields. Lease eligibility joins or indexes against unexpired worker capabilities and resource capacity. For a single-process SQLite worker, the same contracts still apply and improve restart evidence.

## Reproducible registration probe

The ticket includes a source-backed probe:

```bash
node ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/scripts/05-js-task-bundle-registration-probe.mjs \
  > ttmp/2026/07/21/SCRAPER-WORKFLOW-V3--durable-dataflow-workflow-engine-and-modern-goja-dsl/scripts/output/js-task-bundle-registration-probe.json
```

Its fixture catalog explicitly registers two tasks from JavaScript, verifies each bundle-local entrypoint export, seals a worker registry generation, and tests exact scheduler matching. Current probe identities are:

- bundle digest: `sha256:b349df43ce6f637dde813d2f611dff9fcef43063840709333c7d513c7688eb28`;
- registry digest: `sha256:cc2cff85c297bd3f90b3d8ac87ab45ad0534b298caef24a299dacba5f957aedd`;
- exact requirement accepted: true;
- wrong bundle, task version, and entrypoint rejected: true.

The probe demonstrates grammar, deterministic file hashing, explicit registration, entrypoint verification, registry sealing, and exact matching. It is not the production Go/Goja implementation; production still needs canonical Go types, trust verification, registration-only native modules, worker advertisements, and store-backed leasing.

## Decision records

### ADR-JS-1 — Explicit bundle loading, not ambient global registration

- **Status:** proposed
- **Context:** developers want loading JavaScript to register tasks.
- **Options:** side effects from arbitrary `require`; explicit Go registration only; explicit bundle catalog evaluated by worker loader.
- **Decision:** worker boot/reload explicitly evaluates one immutable `catalog.js` and registers normalized entries into a candidate registry.
- **Rationale:** preserves the ergonomic model while making phase, identity, errors, and atomicity visible.
- **Consequences:** developers need a bundle manifest/catalog; ordinary task code cannot self-register.

### ADR-JS-2 — Entrypoint references, not retained Goja functions

- **Status:** proposed
- **Decision:** catalog entries name immutable bundle-local module exports.
- **Rationale:** enables static import resolution, digesting, fresh runtimes, provenance, and registry sealing.
- **Consequences:** runtime requires module loading per attempt; immutable compiled-program caches can optimize later.

### ADR-JS-3 — Exact implementation digest in compiled plan

- **Status:** proposed
- **Decision:** task kind/version alone is insufficient; plan pins bundle digest and entrypoint.
- **Rationale:** resumes and rolling deploys must not execute changed code silently.
- **Consequences:** workers may need to retain multiple bundle generations; unavailable old code blocks rather than substitutes.

### ADR-JS-4 — Fresh runtime per attempt

- **Status:** proposed
- **Decision:** instantiate a new Goja runtime/module cache/task context for each attempt.
- **Rationale:** avoids cross-attempt mutable global state and simplifies cancellation/cleanup.
- **Consequences:** runtime construction cost; benchmark and cache immutable programs rather than pooling mutable runtimes first.

### ADR-JS-5 — Catalog registration and execution use separate capabilities

- **Status:** proposed
- **Decision:** `workflow/task-bundle`, authoring descriptors, and `workflow/task` are available in distinct runtime phases.
- **Rationale:** least privilege and prevention of execution-time registry mutation.
- **Consequences:** generated/embedded hosts need explicit phase-specific runtime plans.

### ADR-JS-6 — Host validation wraps JavaScript

- **Status:** proposed
- **Decision:** Go validates input/config/output/reference/limits regardless of JS validation.
- **Rationale:** custom flexibility must not reopen arbitrary durable payloads or schema bypasses.
- **Consequences:** schemas and output ports are mandatory catalog data.

### ADR-JS-7 — Untrusted bundles require process isolation

- **Status:** proposed
- **Decision:** do not claim in-process Goja is a hostile-code sandbox.
- **Rationale:** module restrictions do not guarantee CPU/memory/native-host isolation.
- **Consequences:** production may need a sandbox worker launcher/container pool and isolation-aware scheduling.

## Alternatives considered

### Compile every custom task into the worker binary

Reliable but too slow for domain iteration and creates a core monolith. Keep Go-native tasks for privileged/high-performance operations; allow JS bundles for domain extensions.

### Put the JavaScript function inside the workflow plan

Rejected. It duplicates code across plans/nodes, obscures dependency identity, complicates signing, and encourages execution of unverified closures.

### Load a mutable directory by path on every attempt

Rejected for production. Files can change beneath active runs and workers may disagree. Development watch mode may do this only by rebuilding a new explicit ephemeral digest.

### Register tasks through module top-level side effects

Rejected. It is order-dependent, hard to make atomic, easy to invoke in the wrong runtime, and incompatible with sealed registry generations.

### Key registry only by task kind

Rejected. Current registry does this, but v3 requires contract version and implementation digest to resume reproducibly.

### Share one Goja runtime across all attempts

Rejected as the initial model. Mutable globals/module caches can leak state and one runaway task can affect unrelated attempts.

### Give custom tasks unrestricted ambient host modules

Rejected. Current go-go-goja host module APIs are available only through exact
profile-selected aliases, preconfigured handles, attempt mounts, bounded
network policy, and allowlisted subprocess profiles. These controls make
trusted first-party JavaScript useful without pretending Goja is a hostile-code
sandbox. Hostile or third-party code still requires process isolation.

## Phased implementation plan

### Phase 1 — Contracts and static fixtures

- Add `TaskKey`, `ImplementationID`, catalog, schema-port, semantics, and bundle-ref types.
- Upgrade v3 runner registry key from string kind to exact implementation identity.
- Write catalog/manifest strict decoders and canonical digest golden tests.
- Create one small fixture bundle with a pure normalize task.

### Phase 2 — Registration module and bundle builder

- Implement `workflow/task-bundle` as a `modules.NativeModule` for registration runtimes.
- Build source graph with literal import validation.
- Normalize catalog to JSON and validate entrypoints/schemas.
- Produce deterministic archive, provenance, DTS, and digest.
- Add xgoja provider packaging for generic modules.

### Phase 3 — Worker registry generations

- Implement candidate/seal/activate registry flow.
- Load exact bundle locks, verify digest/trust/ABI, run self-tests.
- Advertise registry/task capability manifests with TTL.
- Add atomic reload and old-generation drain.

### Phase 4 — Lease-scoped JS execution

- Implement `workflow/task` context and JS runner factory.
- Use fresh runtime per attempt with immutable bundle loader/module allowlist.
- Wire context cancellation and promise waiting.
- Validate all inputs/outputs in Go wrappers.
- Translate failures and enforce progress/output/timeout limits.

### Phase 5 — Compiler and dispatcher binding

- Bind descriptors to approved exact implementations.
- Record implementation identity in plan/node/attempt.
- Match leases to unexpired worker advertisements and resources.
- Surface capability-blocked nodes and prevent substitution.

### Phase 6 — Trust, sandboxing, and rolling operations

- Add signatures/publisher policy, SBOM, provenance verification.
- Add subprocess/container worker class for untrusted bundles.
- Add operator bundle list/inspect/quarantine/reload commands.
- Test rolling old/new bundle coexistence and long-run resumption.

### Phase 7 — Replace cookbook placeholders

- Implement example domain bundles outside scraper core.
- Replace hypothetical `data.tasks.*` with imported bundle authoring modules.
- Convert cookbook scripts to integration fixtures and compiled-plan goldens.

## Testing strategy

### Bundle builder

- same source/lock/toolchain produces byte-identical bundle digest;
- map/file order and timestamps do not affect archive identity;
- unknown fields, duplicate keys, missing entrypoints, dynamic imports, undeclared modules, invalid schemas fail;
- self-test failure prevents publication;
- generated authoring module/DTS/catalog have parity.

### Registration

- one bundle with multiple tasks registers atomically;
- conflict leaves registry unchanged;
- malformed bundle never partially appears;
- same lock recreates same registry digest after restart;
- execution runtime cannot import registration module;
- authoring runtime cannot import task execution module.

### Worker/dispatcher

- node leases only to exact bundle digest/entrypoint/ABI;
- same kind/version with wrong digest is not eligible;
- expired advertisement is not eligible;
- missing old implementation produces capability-blocked state;
- registry reload does not interrupt attempts using old generation;
- old generation drains before cleanup.

### Runtime

- fresh attempts cannot observe globals from prior attempts;
- cancellation interrupts JavaScript and rejects stale completion;
- promise resolution/rejection maps correctly;
- undeclared output port/schema/size fails before persistence;
- raw stack/source/secret canaries do not enter events/results;
- task cannot import filesystem/process/network/registry modules unless allowed;
- progress/log rate limits hold;
- runtime and artifact handles close on every error path.

### Reproducibility and privacy

- attempt records contain exact bundle/entrypoint/registry identities;
- bundle code is not copied into every node row;
- node input remains compact;
- credentials and host-service objects cannot serialize into plan/node/event;
- SQLite/WAL scans find no source-data or secret canaries;
- plan resume after worker restart uses the same implementation digest.

### Isolation tests

- trusted in-process task exceeding deadline is interrupted;
- memory/CPU hostile fixtures run only in sandboxed subprocess worker tests;
- sandbox worker cannot access undeclared network/filesystem paths;
- killing sandbox process results in lease loss/retry evidence without corrupting worker registry.

## Acceptance criteria

### Developer experience

- [ ] A domain repository can define several JS tasks in one bundle without modifying scraper Go code.
- [ ] One build command emits an immutable bundle, catalog, authoring module, DTS, help, provenance, and digest.
- [ ] A workflow can `require()` the generated descriptor-only authoring module.
- [ ] Worker configuration can load the same bundle by immutable reference.

### Registration and reproducibility

- [ ] Catalog loading is explicit, atomic, and fail-closed.
- [ ] Registry is keyed by task kind, contract version, and exact implementation identity.
- [ ] Same lock reproduces the same registry digest on every worker.
- [ ] Plans pin bundle digest, entrypoint, and ABI.
- [ ] A worker never substitutes a different implementation for a pinned plan.
- [ ] Old/new implementations coexist during rolling upgrades.

### Execution

- [ ] Every attempt uses a fresh lease-scoped runtime or an equivalently proven isolation mechanism.
- [ ] Only catalog/host-approved modules and services are available.
- [ ] Inputs and outputs are validated by Go outside JavaScript.
- [ ] Cancellation, lease loss, timeout, output size, log, progress, and budget limits are enforced.
- [ ] Task execution cannot mutate the registry or emit raw operations.

### Security

- [ ] Bundle digest/signature/publisher/ABI/imports are verified before advertisement.
- [ ] Safe authoring, registration, and execution phases expose distinct module sets.
- [ ] Secret/source canaries are absent from plan/node/event/report persistence.
- [ ] Untrusted bundles use OS/process isolation; docs do not claim Goja is a hostile-code sandbox.

### Operations

- [ ] Worker capability advertisements include registry and exact task implementation digests.
- [ ] Dispatcher matches exact implementation plus resource/capability/isolation class.
- [ ] Failed reload keeps the prior registry active.
- [ ] Capability-blocked nodes are visible and actionable.
- [ ] Bundle inspection/quarantine/drain tooling is documented and tested.

## Implemented first bundle slice

The linear-transform fixture now exercises the design with one shared source of
truth under `pkg/testfixtures/workflowv3linear`:

```js
const workflow = require("workflow");
const tasks = require("cookbook-linear-transform-tasks");
```

The fixture contains the descriptor mapping, real `execution/tasks.cjs`,
manifest-derived task specs, source-derived bundle digest, and authoring script.
`pkg/workflowv3` seals those specs into an immutable registry generation.
`pkg/workflowv3runtime` resolves only the complete pinned identity and builds a
fresh runtime with only `workflow/task` and the declared `fs:input` alias.

Implemented evidence includes:

- a changed source byte produces a different bundle and registry digest;
- wrong bundle digest, entrypoint, or ABI does not resolve or lease;
- bundle manifests and files are returned as defensive copies;
- authoring and execution consume the same catalog;
- task outputs are schema-validated by Go before fenced persistence;
- typed JavaScript failures survive async Promise rejection;
- source/private canaries remain outside SQLite main/WAL/SHM;
- a restart between tasks resumes and reopens the same root output.

## Implemented trusted HTTP and database profiles

Slices 3–5 extend the same bundle and registry contracts without introducing an
ambient module registry:

- `RegistryBuilder.AdvertiseModules` explicitly adds exact aliases during the
  worker boot transaction;
- the sealed generation digest covers implementation identities and sorted
  aliases;
- `ResolveNode` requires exact implementation, modules, resource class, and
  retry policy;
- `TaskModuleRegistry` constructs one policy-selected module instance for each
  fresh lease runtime;
- engine startup fails when sealed aliases and configured module factories do
  not match exactly.

The public HTTP bundle requests only `fs:input` and `fetch:public`. The host
injects allowed origins, timeout, maximum response bytes, disabled credential
sources, redirect checks, and `http.Client`; none of those policy values or
headers enter the manifest or plan. Typed JavaScript failures expose stable
codes while the host persists only redacted messages.

The database bundle requests only `fs:input` and `db:sync`. The host injects a
Go-preconfigured `QueryExecer`; go-go-goja's `configure()` path is disabled.
The trusted bundle uses a target transaction and stable run/node operation key
to couple writes with an idempotency marker. A test commits 500 rows, simulates
a crash before workflow completion, restarts the workflow store, and proves the
second attempt creates no second operation or audit row.

Canonical bundle specs now include resource class and bounded fixed retry
policy. This changes the bundle/catalog/plan digest intentionally and is
covered by regenerated direct-Go/JavaScript goldens. Resource/policy mismatch
is an admission error, not a runtime fallback.

Hot reload, signatures/provenance verification, old/new generation draining,
and untrusted subprocess isolation remain later architecture slices. They do not
alter the exact identity or fresh-runtime contracts exercised here.

## Intern checklist

Before implementing custom JS task support:

- [ ] Do not extend the current string-only runner registry without adding version/digest identity.
- [ ] Do not let ordinary `require()` mutate global process state.
- [ ] Freeze manifest/catalog/ABI schemas and golden digests.
- [ ] Separate registration, authoring, and execution runtime module sets.
- [ ] Make bundle build deterministic before adding hot reload.
- [ ] Validate one pure task end to end before adding network/database capabilities.
- [ ] Add exact worker capability matching before rolling versions.
- [ ] Prove fresh runtime state and cancellation cleanup.
- [ ] Add process isolation before accepting untrusted publishers.
- [ ] Keep code bytes and source data out of workflow rows.

## References

- [Primary workflow-v3 architecture](01-durable-dataflow-workflow-v3-and-modern-scripting-architecture.md)
- [JavaScript cookbook and execution atlas](../reference/03-workflow-v3-javascript-cookbook-and-execution-atlas.md)
- `pkg/engine/runner/runner.go`
- `pkg/workflow/executor.go`
- `pkg/js/runtime/executor.go`
- `pkg/sites/manifest/manifest.go`
- `pkg/sites/manifest/loader.go`
- `go-go-goja/pkg/xgoja/providerapi/module.go`
- `go-go-goja/pkg/xgoja/sourcegraph/graph.go`
