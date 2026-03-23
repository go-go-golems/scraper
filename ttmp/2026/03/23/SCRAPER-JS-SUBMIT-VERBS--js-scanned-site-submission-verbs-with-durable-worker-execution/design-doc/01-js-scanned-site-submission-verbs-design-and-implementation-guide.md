---
Title: JS-scanned site submission verbs design and implementation guide
Ticket: SCRAPER-JS-SUBMIT-VERBS
Status: active
Topics:
    - scraper
    - javascript
    - cli
    - glazed
    - jsverbs
    - worker
DocType: design-doc
Intent: ""
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-goja/pkg/jsverbs/command.go
      Note: Glazed command compilation logic reused for site JS verb registration
    - Path: ../../../../../../../go-go-goja/pkg/jsverbs/runtime.go
      Note: Existing plain-function invocation path that cannot be used directly for scraper submission verbs
    - Path: ../../../../../../../go-go-goja/pkg/jsverbs/scan.go
      Note: Scanner used to discover __verb__ metadata from site JS files
    - Path: pkg/engine/scheduler/scheduler.go
      Note: Durable worker loop and scheduler behavior that the new worker command will wrap
    - Path: pkg/js/runtime/executor.go
      Note: Current execution-time JS op runtime that must remain separate from the new submission runtime
    - Path: pkg/sites/submitverbs/register.go
      Note: Scraper-side JS verb registration layer mounted under `site <site> run`
    - Path: pkg/sites/submitverbs/runtime.go
      Note: Scraper-specific submission runtime for scanned verbs
    - Path: pkg/sites/jsdemo/verbs/seed.js
      Note: Example submission-time JS verb that seeds a durable workflow
    - Path: pkg/sites/jsdemo/workflow.go
      Note: Legacy Go workflow builders retained as a behavioral reference
ExternalSources: []
Summary: Design guide for moving site command discovery into JS while keeping workflow execution durable and worker-driven.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# JS-scanned site submission verbs design and implementation guide

## Executive summary

This ticket adds a new submission path to the scraper engine. Today, site-facing operator commands such as `scraper site js-demo run seed` are handwritten in Go. They prepare the engine DB and site DB, build a workflow in Go, and then either run the scheduler locally or return a result. The user wants a different model: the command surface should be described in JavaScript with `__verb__` metadata, while Go remains the host that sets up databases, migrations, and durable workflow submission.

The target architecture has two different runtime roles:

- submission-time JS verbs, which are scanned from site JavaScript files and invoked once by the CLI to submit initial workflow state;
- execution-time op scripts, which are executed later by a worker process through the existing durable scheduler and runner registry.

This separation is important. A submission verb is not the same thing as an op body. The submission verb is a thin workflow seeder. It takes CLI values, creates a workflow record, inserts initial ops, and exits. The worker process then polls the engine DB, leases ready ops, and runs the real JS and HTTP runners. This lets operators flexibly submit different jobs through JS-defined commands without turning the CLI into an in-process worker loop.

## Problem statement

The current codebase already has the pieces needed for durable execution, but they are not assembled in the way the user wants.

Current state:

- `pkg/sites/jsdemo/cli.go` and the other site `cli.go` files defined Go-native commands before this ticket.
- `pkg/sites/jsdemo/workflow.go` and similar workflow builders build `CreateWorkflowParams` in Go.
- `pkg/js/runtime/executor.go` executes JS op bodies with the scraper `ctx` contract.
- `pkg/engine/scheduler/scheduler.go` can poll the DB and execute ready work.
- `go-go-goja/pkg/jsverbs` can scan JavaScript files for `__verb__`, `__section__`, and `doc` metadata and compile Glazed commands.

Missing pieces:

- there is no worker-oriented top-level command that simply polls the DB and executes queued work as a long-running process;
- `jsverbs` currently invokes plain JS functions directly and knows nothing about scraper engine DBs, site DBs, migrations, or workflow submission;
- site definitions have no separate filesystem root for submission verbs;
- there was no submission runtime contract for JS functions that needed to submit initial ops into a durable workflow.

The result is a split that is close, but not yet aligned with the intended model.

## Desired architecture

The desired flow is:

```text
Go CLI startup
  -> load site registry
  -> scan site verb JS files for __verb__
  -> build Glazed/Cobra commands from discovered metadata

Go CLI invocation of one discovered verb
  -> open engine DB
  -> migrate/open site DB
  -> create submission runtime
  -> invoke exactly one JS submission function
  -> JS submission function creates workflow + initial ops
  -> CLI prints workflow/result and exits

Separate Go worker process
  -> poll engine DB
  -> lease ready ops
  -> execute js/http runners
  -> persist results, artifacts, child ops
```

### ASCII diagram

```text
+-----------------------+      +-------------------+      +----------------------+
| scraper site ... verb | ---> | submit JS runtime | ---> | engine DB workflows  |
| discovered from JS    |      | create workflow   |      | and initial ops      |
+-----------------------+      +-------------------+      +----------------------+
                                                                      |
                                                                      v
+-----------------------+      +-------------------+      +----------------------+
| scraper worker run    | ---> | scheduler.RunOnce | ---> | js/http op execution |
| long-running poller   |      | lease ready ops   |      | result persistence   |
+-----------------------+      +-------------------+      +----------------------+
```

## Important conceptual split

The intern reading this ticket needs to keep one distinction straight from the beginning.

### Submission verb JS

Submission verbs are CLI entrypoints. They should:

- expose `__verb__` metadata for command discovery and help;
- receive CLI-bound values;
- create a workflow record and initial ops;
- optionally choose a “target” op to watch or report;
- return a structured submission result.

Submission verbs should not:

- behave like op bodies;
- read dependency results;
- assume they are running under a lease;
- perform long-running scrape work inline.

### Execution-time op JS

Op JS is the existing runtime used by the worker. Those scripts:

- execute with scraper `ctx`;
- read dependencies with `ctx.dep(...)`;
- emit child ops with `ctx.emit(...)`;
- write artifacts and records;
- run under scheduler control.

This ticket is about adding the first category without breaking the second.

## Current evidence in the codebase

The new implementation should be grounded in the current files:

- `scraper/pkg/sites/submitverbs/register.go`
  - current JS verb registration layer mounted under `site <site> run`
- `scraper/pkg/sites/submitverbs/runtime.go`
  - current scraper-specific submission runtime
- `scraper/pkg/sites/jsdemo/workflow.go`
  - legacy Go workflow builders for seed/item/summary
- `scraper/pkg/sites/jsdemo/scripts/seed.js`
  - current real op body that emits child ops
- `scraper/pkg/sites/jsdemo/verbs/seed.js`
  - current submission-time JS verb for workflow seeding
- `scraper/pkg/sites/registry/registry.go`
  - current site definition contract
- `scraper/pkg/engine/scheduler/scheduler.go`
  - worker loop primitive and lease/execution logic
- `scraper/pkg/js/runtime/executor.go`
  - real JS op execution runtime
- `go-go-goja/pkg/jsverbs/scan.go`
  - JS metadata scanner
- `go-go-goja/pkg/jsverbs/command.go`
  - Glazed command compilation
- `go-go-goja/pkg/jsverbs/runtime.go`
  - current plain-function invocation path that must be adapted or bypassed

## Proposed solution

The cleanest implementation is to add a scraper-specific JS verb host on top of `go-go-goja/pkg/jsverbs`.

### High-level design

1. Extend site definitions with an optional verbs filesystem:
   - `VerbFS`
   - `VerbRoot`

2. Add a scraper-side host package:
   - `pkg/sites/submitverbs`

3. That host package:
   - scans site `verbs/` trees with `jsverbs.ScanFS(...)`;
   - compiles commands from the discovered metadata;
   - wraps each command with Go execution logic;
   - injects a submission runtime into JS;
   - records workflows/initial ops in the durable engine DB.

4. Add a worker command:
   - `scraper worker run`

5. Keep current op runtime unchanged except for shared helpers if needed.

### Why not use `jsverbs` invocation unchanged?

`go-go-goja/pkg/jsverbs/runtime.go` currently assumes a plain JS function call. It builds a default runtime and calls the scanned function with positional/object arguments derived from Glazed fields. That is useful for generic JS verbs, but it is not enough for scraper submission verbs because the function also needs access to:

- engine DB handles;
- site DB handles;
- site metadata;
- workflow creation helpers;
- op emission helpers;
- maybe a `--wait` mode later.

So the correct approach is:

- reuse `jsverbs` for scanning and Glazed command shape;
- do not reuse the default `jsverbs` invocation path for scraper submission verbs;
- instead, provide a scraper-specific invocation layer that uses the scanned metadata but a different runtime contract.

## Proposed runtime contract for submission verbs

The submission runtime should be intentionally small.

### Proposed JS signature

```js
__verb__("seed", {
  short: "Submit a js-demo workflow",
  fields: {
    count: { type: "int", default: 4 },
    multiplier: { type: "int", default: 3 },
    prefix: { type: "string", default: "demo" }
  }
});

async function seed(ctx) {
  const values = ctx.values || {};
  const workflowID = String(ctx.workflow.id);
  const seedID = workflowID + ":seed";

  ctx.setWorkflowName("js-demo seed workflow");
  ctx.emit({
    id: seedID,
    kind: "js",
    queue: "site:js-demo:js",
    metadata: { script: "seed.js" },
    input: {
      runID: workflowID,
      count: values.count,
      multiplier: values.multiplier,
      prefix: values.prefix
    }
  });

  ctx.setTargetOpID(seedID + ":summary");
  return { data: { workflowID, targetOpID: seedID + ":summary" } };
}
```

### Proposed `ctx` shape

The new submission `ctx` should include:

- `ctx.site`
- `ctx.workflow`
- `ctx.values`
- `ctx.sections`
- `ctx.command`
- `ctx.now`
- `ctx.log(...)`
- `ctx.emit({...})`
- `ctx.setTargetOpID(opID)`
- `ctx.setWorkflowName(name)`
- `ctx.setWorkflowMetadata({...})`
- `require("site-db")` and `require("scraper-db")` if needed, though the v1 `js-demo` verbs do not rely on direct DB queries

The workflow itself should be auto-created by Go before the JS verb runs. The main path is:

- Go creates the workflow shell with CLI-derived input
- JS emits the initial ops
- JS optionally selects a target op and adjusts workflow metadata
- Go persists the workflow plus emitted ops durably

### Durable envelope returned to Go

The runtime should give Go a normalized result such as:

```json
{
  "workflowID": "js-demo-seed-123",
  "targetOpID": "js-demo-seed-123:seed:summary",
  "submittedOpIDs": [
    "js-demo-seed-123:seed"
  ],
  "data": {
    "count": 4,
    "prefix": "demo"
  }
}
```

Go should validate that:

- a workflow exists;
- at least one op was submitted;
- all emitted ops reference the created workflow.

## Proposed Go host architecture

### New package: `pkg/sites/submitverbs`

Responsibilities:

- scan a site’s `verbs/` filesystem;
- register discovered commands on the site subtree;
- wrap invocation in scraper-specific DB/migration/submission setup;
- optionally watch workflow completion later.

Suggested layout:

```text
pkg/sites/submitverbs/
  host.go
  runtime.go
  register.go
```

### Pseudocode for host registration

```go
func RegisterSiteVerbCommands(siteCmd *cobra.Command, def registry.Definition, sites *registry.Registry) error {
    if def.VerbsFS == nil {
        return nil
    }

    verbsRegistry, err := jsverbs.ScanFS(def.VerbsFS, def.VerbsRoot)
    if err != nil {
        return err
    }

    commands, err := verbsRegistry.Commands()
    if err != nil {
        return err
    }

    for _, c := range commands {
        wrapped, err := WrapAsScraperSubmissionCommand(def, verbsRegistry, c)
        if err != nil {
            return err
        }
        siteCmd.AddCommand(wrapped)
    }

    return nil
}
```

### Pseudocode for command execution

```go
func runSubmissionVerb(cmd *cobra.Command, def Definition, verb *jsverbs.VerbSpec, parsed *values.Values) error {
    ctx := cmd.Context()

    engineStore := openEngineStore(...)
    siteDB := migrateAndOpenSiteDB(...)

    runtime := newSubmissionRuntime(SubmissionRuntimeConfig{
        Site:        def,
        EngineStore: engineStore,
        SiteDB:      siteDB,
        Values:      parsed,
        Verb:        verb,
    })

    result, err := runtime.Invoke(ctx)
    if err != nil {
        return err
    }

    printSubmissionSummary(cmd, result)
    return nil
}
```

## Worker command design

The worker process should be explicit and boring.

Suggested command:

```text
scraper worker run --engine-db state/engine.db --poll-interval 250ms --worker-id local-worker
```

Responsibilities:

- open engine DB;
- build the default site registry;
- register runners;
- open site DBs on demand through the existing provider hook;
- loop forever with scheduler polling until context cancellation.

Pseudocode:

```go
func runWorker(ctx context.Context, opts WorkerOptions) error {
    registry := defaults.NewRegistry()
    store := sqlitestore.Open(ctx, opts.EngineDB)
    runners := buildDefaultRunners(registry)

    s := scheduler.New(store, runners, scheduler.Config{
        PollInterval: opts.PollInterval,
        MaxWorkers:   opts.MaxWorkers,
        ...
    }, opts.WorkerID, nil)

    s.SetSiteDBProvider(func(ctx context.Context, site model.SiteName) (QueryExecer, error) {
        return openSiteDBForSite(...)
    })

    for {
        if _, err := s.RunOnce(ctx); err != nil {
            log.Error().Err(err).Msg("worker cycle failed")
        }
        waitOrExit(ctx, opts.PollInterval)
    }
}
```

## Site layout changes

Each site should gain a separate `verbs/` tree.

Example:

```text
pkg/sites/jsdemo/
  site.go
  scripts/
    seed.js
    build_item.js
    summarize.js
  verbs/
    seed.js
    item.js
    summary.js
  migrations/
```

Why separate trees:

- `scripts/` are executed by workers as op bodies;
- `verbs/` are scanned by the CLI as submission-time entrypoints;
- both may share helpers later through `scripts/lib/` or a site-level shared helper tree;
- the role boundary stays clear.

## How `js-demo` should be used to prove the model

`js-demo` is the best first validation site because it avoids live HTTP and pagination noise.

Target demo:

1. Start worker:

```bash
scraper worker run --engine-db /tmp/engine.db --sites-dir /tmp/sites
```

2. Submit workflow from JS-scanned verb:

```bash
scraper site js-demo seed --engine-db /tmp/engine.db --sites-dir /tmp/sites --count 3 --multiplier 4 --prefix smoke
```

3. Inspect state:

```bash
scraper engine status --engine-db /tmp/engine.db
scraper engine workflow show <workflow-id>
```

4. Confirm completion in site DB and engine results.

This proves:

- JS command discovery works;
- Go host setup works;
- the command only submits initial work;
- the worker executes the emitted follow-on ops;
- the durable pipeline remains intact.

## API references and file references

The intern implementing this ticket should keep the following files open while working:

- `scraper/pkg/sites/registry/registry.go`
- `scraper/pkg/cmd/root.go`
- `scraper/pkg/cmd/site.go`
- `scraper/pkg/engine/scheduler/scheduler.go`
- `scraper/pkg/engine/runner/js.go`
- `scraper/pkg/js/runtime/executor.go`
- `scraper/pkg/sites/jsdemo/site.go`
- `scraper/pkg/sites/jsdemo/workflow.go`
- `scraper/pkg/sites/jsdemo/scripts/seed.js`
- `go-go-goja/pkg/jsverbs/scan.go`
- `go-go-goja/pkg/jsverbs/command.go`
- `go-go-goja/pkg/jsverbs/runtime.go`

## Phased implementation plan

### Phase 1. Ticket and design

- create this ticket
- write the guide
- write initial task breakdown

### Phase 2. Worker command

- add `scraper worker run`
- reuse existing scheduler logic
- add tests for a minimal worker loop

### Phase 3. Site definition and verb scanning

- extend site definitions with `VerbsFS` and `VerbsRoot`
- add `verbs/` to `js-demo`
- scan JS verbs during CLI construction

### Phase 4. Scraper submission runtime

- implement scraper-specific command wrapping
- auto-create the workflow in Go before invoking the JS verb
- implement `emit`, `setTargetOpID`, and workflow metadata helpers
- define normalized submission result

### Phase 5. Replace `js-demo` handwritten commands

- move `js-demo` command metadata into JS verb files
- remove or deprecate the current Go-only `run seed|item|summary` wrappers
- test CLI help and submission behavior

### Phase 6. End-to-end validation

- start worker
- submit `js-demo` workflow from JS verb
- confirm tasks execute
- use status admin commands to verify the durable path

## Testing strategy

Tests should exist at three levels.

### Unit tests

- JS verb scanning from `fs.FS`
- submission result normalization
- invalid workflow/op validation

### Command tests

- `scraper site js-demo seed --help`
- `scraper site js-demo item --help`
- `scraper site js-demo summary --help`
- command execution that submits but does not inline-run all work

### End-to-end tests

- start worker in background
- submit job from JS verb
- poll engine DB until workflow succeeds
- verify site DB records and result rows

## Risks and failure modes

- confusing submission JS and op JS contracts
- allowing submission verbs to mutate too much DB state directly
- trying to reuse generic `jsverbs` invocation without scraper host hooks
- leaving current handwritten site commands in place too long and creating duplicate command trees
- creating site DB lifecycle races between worker and submitter if open/migrate rules are inconsistent

## Alternatives considered

### Keep all site commands in Go

Rejected because it preserves the duplication the user wants to eliminate.

### Run the whole scheduler inline inside the CLI command

Rejected as the default because it conflates submission and execution. It is acceptable as a dev/test mode later, but not as the primary architecture.

### Reuse op `ctx` for submission verbs

Rejected because submission verbs are not leased ops and should not pretend to be.

## Open questions

- Whether submission verbs should live in the same file as op bodies later, or always in a separate `verbs/` tree
- Whether `--wait` should be added in the same ticket or deferred until the worker path is proven
- Whether `jsverbs` itself should grow a formal host-invocation hook later, or whether scraper should keep a private adapter layer

## Recommendation

Implement this in `js-demo` first and only once the pattern is stable consider porting Hacker News and Slashdot. `js-demo` is small, pure-JS, already proven, and ideal for validating the architecture without live-site noise.
