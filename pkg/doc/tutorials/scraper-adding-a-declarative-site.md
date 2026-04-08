---
Title: Adding a Declarative Site
Slug: scraper-adding-a-declarative-site
Short: "Step-by-step guide for adding a scraper site with site.yaml, JavaScript verbs and scripts, and no site-specific Go code."
Topics:
- scraper
- tutorial
- sites
- javascript
- manifests
- onboarding
Commands:
- site
- worker
Flags:
- sites-dir
- engine-db
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

If your site does not need custom native Go modules or special runtime hooks, the preferred path is now declarative: define the site with a `site.yaml` manifest, keep the scraping behavior in JavaScript, and let the existing engine handle execution, retries, queue policies, and persistence.

This tutorial shows the no-Go path for adding a site.

## When To Use This Path

Use the declarative path when:

- your site can be described with roots for scripts, verbs, migrations, and optional help
- queue policies are pure metadata
- the site only needs the standard runtime modules already supported by the manifest loader

Do not use this path yet when:

- the site requires a one-off Go-native module
- the site needs custom runtime registrars
- the site needs custom CLI wiring that cannot be shared

For those cases, use `scraper help scraper-adding-a-site` and keep the site Go-defined.

## Step 1 — Create The Site Directory

Create a directory under `pkg/sites/<site>/` with:

- `site.yaml`
- `scripts/`
- `verbs/`
- `migrations/`
- optional `fixtures/`

You still need a tiny `site.go` wrapper today for embedded built-in sites, but that wrapper should only embed the files and load the manifest. The site behavior itself stays in YAML, SQL, and JavaScript.

## Step 2 — Write `site.yaml`

The manifest is the declarative envelope for the site.

Minimal example:

```yaml
name: example
databaseFileName: example.db
scriptsRoot: scripts
verbsRoot: verbs
sqlMigrationsRoot: migrations
modules:
  - default-registry
```

Optional queue policy example:

```yaml
queuePolicies:
  - queue: site:example:http
    maxInFlight: 2
    rateLimit:
      ratePerSecond: 1
      burst: 2
```

Current manifest fields are validated strictly. Typos and unknown keys fail fast during load.

## Step 3 — Add The Small Wrapper

For embedded built-in sites, keep `site.go` minimal:

```go
//go:embed site.yaml scripts/*.js verbs/*.js migrations/*.sql
var siteFS embed.FS

func Definition() siteregistry.Definition {
    def, err := sitemanifest.LoadDefinition(siteFS, "")
    if err != nil {
        panic(err)
    }
    return def
}
```

In practice, the current built-in sites cache this load behind `sync.Once` so repeated calls stay cheap.

Reference examples:

- `pkg/sites/jsdemo/site.go`
- `pkg/sites/hackernews/site.go`

## Step 4 — Write The Submit Verbs

The entrypoint still lives in `verbs/` and uses the normal submit-verb model.

Typical responsibilities:

- read values from `ctx.values`
- create the initial durable ops
- optionally set a target op

The verb should not perform the crawl directly. It seeds the workflow graph.

## Step 5 — Write The Durable Scripts

Put the durable execution logic in `scripts/`.

Use the existing JS runtime APIs:

- `ctx.input`
- `ctx.dep(...)`
- `ctx.emit(...)`
- `ctx.writeRecord(...)`
- `ctx.writeArtifact(...)`

See:

- `scraper help scraper-js-api-reference`
- `pkg/sites/jsdemo/scripts/`
- `pkg/sites/hackernews/scripts/`

## Step 6 — Add Migrations

If the site needs queryable projections, add numbered SQL files in `migrations/`.

Examples:

- `pkg/sites/jsdemo/migrations/001_init.sql`
- `pkg/sites/hackernews/migrations/001_init.sql`

Keep the first migration small and focused on the output your workflow actually writes.

## Step 7 — Register The Site

Today, built-in sites are still registered explicitly in:

- `pkg/sites/defaults/defaults.go`

The registry bootstrap remains explicit for now, even when individual sites are manifest-backed internally.

## Step 8 — Add At Least One End-To-End Test

Do not stop at unit tests for a parser or one helper function. Add a command-path or scheduler-path test that proves:

1. the submit verb emits work
2. the worker can execute the work
3. the site DB receives the expected projection or artifacts

Good examples:

- `pkg/sites/jsdemo/site_test.go`
- `pkg/cmd/site_test.go`

## Step 9 — Validate

Before committing:

```bash
gofmt -w pkg/sites/<site>
go test ./pkg/sites/... -count=1
go test ./... -count=1
```

If you added or changed help pages:

```bash
go run ./cmd/scraper help scraper-adding-a-declarative-site
```

## Practical Advice

- Start with the smallest possible workflow graph.
- Add queue policies only where the site actually needs protection.
- Keep the first manifest small and boring.
- If you feel pressure to stuff custom runtime logic into the manifest, that is a signal the site may still need a Go-native wrapper.

## See Also

- `scraper help scraper-adding-a-site`
- `scraper help scraper-runtime-model`
- `scraper help scraper-js-api-reference`
- `scraper help scraper-architecture-overview`
