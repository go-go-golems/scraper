---
Title: Bootstrap config and early site command loading
Ticket: SCRAPER-DECLARATIVE-SITES
Status: active
DocType: design
Owners: []
Summary: Design for loading site manifest directories from config, env, and bootstrap CLI flags before building the Cobra command tree.
---

# Bootstrap config and early site command loading

## Problem statement

After moving all declarative sites into the repo-level `sites/` directory, scraper now has a real bootstrap dependency:

- dynamic site verbs such as `scraper site js-demo run seed` are created by `newSiteCommand(...)`
- `newSiteCommand(...)` iterates the loaded site registry at command construction time
- therefore the site registry must already be populated **before** `rootCmd.Execute()` runs

A normal Cobra flag is too late for this. The usual flow is:

1. build command tree
2. parse flags
3. run command

But scraper needs:

1. discover site manifest directories
2. load site manifests into the registry
3. build command tree using the populated registry
4. parse runtime flags and execute

## Why the current `--sites-manifest-dir` flag is insufficient by itself

A persistent Cobra flag is visible in help, but Cobra does not make its value available until after the full command tree already exists. That means a runtime-only `--sites-manifest-dir` flag cannot be the source of truth for building dynamic `site <name> run <verb>` commands.

This is the same architectural shape that sqleton solved for repository-loaded SQL commands:

- app-owned config is loaded first
- command sources are discovered next
- dynamic commands are injected into the root command
- only then does normal Cobra execution begin

Relevant sqleton references:

- `/home/manuel/code/wesen/corporate-headquarters/sqleton/cmd/sqleton/config.go`
- `/home/manuel/code/wesen/corporate-headquarters/sqleton/cmd/sqleton/main.go`
- `/home/manuel/code/wesen/corporate-headquarters/sqleton/ttmp/2026/04/02/SQLETON-02-VIPER-APP-CONFIG-CLEANUP--remove-viper-and-separate-sqleton-app-config-from-command-config/design/01-sqleton-viper-removal-and-app-config-cleanup-design.md`
- `/home/manuel/code/wesen/obsidian-vault/Projects/2026/04/02/PROJ - Sqleton SQL Command Cleanup - Technical Project Report.md`

## Design goals

1. Allow scraper to load site manifests from multiple directories.
2. Support three bootstrap inputs:
   - config file
   - environment variable
   - bootstrap CLI flags
3. Resolve those inputs **before** building dynamic site commands.
4. Keep runtime command flags separate from bootstrap/app config.
5. Preserve direct constructor seams for tests, e.g. `NewRootCommand(version, dirs...)`.

## Proposed bootstrap architecture

### 1. App-owned config loader

Add a small scraper-specific config loader, similar to sqleton's `collectRepositoryPaths(...)` pattern.

Suggested file:

- `pkg/cmd/app_config.go`

Suggested shape:

```go
type AppConfig struct {
    SitesManifestDirs []string `yaml:"sitesManifestDirs"`
}
```

Responsibilities:

- resolve the standard app config path with `glazed/pkg/config.ResolveAppConfigPath("scraper", "")`
- read YAML directly
- decode only scraper-owned app config
- merge environment-provided directories
- normalize, trim, dedupe, and expand env vars in paths

Suggested env var:

```text
SCRAPER_SITES_MANIFEST_DIRS
```

Use `filepath.SplitList(...)` so the value is portable across macOS/Linux.

### 2. Bootstrap flag pre-parser

Add a tiny pre-parser that runs on raw CLI args before building the full Cobra tree.

Suggested file:

- `pkg/cmd/bootstrap.go`

Suggested shape:

```go
type BootstrapOptions struct {
    SitesManifestDirs []string
}

func ParseBootstrapArgs(args []string) (BootstrapOptions, error)
```

Implementation notes:

- use a separate `pflag.FlagSet`
- parse only bootstrap flags, not runtime flags
- support repeated `--sites-manifest-dir /path/a --sites-manifest-dir /path/b`
- ignore unknown flags so regular Cobra parsing still owns command/runtime flags later

This parser exists only to discover the directories needed to build the command tree.

### 3. Merge order

Suggested merge order:

1. config file
2. environment variable
3. bootstrap CLI flags

This gives the most immediate, operator-controlled source the final say in ordering while still allowing config defaults.

Normalization rules:

- `strings.TrimSpace`
- drop empty strings
- `os.ExpandEnv`
- `filepath.Clean`
- stable de-duplication preserving first occurrence after merge order

### 4. Root command construction stays explicit

Keep:

```go
func NewRootCommand(version string, manifestDirs ...string) (*cobra.Command, error)
```

That constructor should assume bootstrap resolution has already happened.

This is important because:

- tests can call it directly with `testfixtures.SitesDir(t)`
- non-main integrations can build root commands with explicit directories
- command construction remains deterministic and independent of global process state

### 5. Add a bootstrap-aware constructor/helper for main

Add a helper such as:

```go
func NewRootCommandFromBootstrap(version string, args []string) (*cobra.Command, error)
```

or equivalently:

```go
func CollectSitesManifestDirs(args []string) ([]string, error)
```

Then `cmd/scraper/main.go` becomes:

```go
func main() {
    rootCmd, err := scrapercmd.NewRootCommandFromBootstrap(version, os.Args[1:])
    if err != nil {
        os.Exit(1)
    }
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

Internally that helper should:

1. parse bootstrap args
2. load scraper app config
3. load env var overrides
4. merge and normalize dirs
5. call `NewRootCommand(version, dirs...)`

## Why this is cleaner than late-loading in `RunE`

Late-loading in `RunE` works for subsystems that only need data during execution, such as a worker needing extra manifests after the command is already chosen.

It does **not** work for scraper's dynamic site verbs because the command discovery itself depends on the site registry contents.

So scraper should treat site-manifest directory discovery as bootstrap config, not as an ordinary runtime flag.

## Command/help behavior

The root command should still declare `--sites-manifest-dir` as a persistent flag so:

- help output documents it
- Cobra accepts it later without error
- users have one obvious CLI interface

But the command tree should be built from the **pre-parsed** value, not from a late `RunE` callback.

That means the same flag exists in two roles:

1. bootstrap pre-parse source
2. normal Cobra-declared flag for UX/help consistency

This is acceptable because the bootstrap parser reads raw args before Cobra, while Cobra still owns normal command parsing once the tree exists.

## Testing strategy

### Unit tests

Add focused tests for:

- app config YAML loading
- env var parsing via `filepath.SplitList`
- merge order and de-duplication
- bootstrap arg parsing for repeated `--sites-manifest-dir`

### Integration tests

Add tests proving:

- `NewRootCommandFromBootstrap(version, args)` builds site verbs for `js-demo`
- config-only bootstrap works
- env-only bootstrap works
- CLI flag bootstrap works
- mixed sources merge correctly

## Implementation plan

1. Add bootstrap design doc and task breakdown.
2. Add `pkg/cmd/app_config.go` and tests.
3. Add `pkg/cmd/bootstrap.go` and tests.
4. Add `NewRootCommandFromBootstrap(...)` helper.
5. Update `cmd/scraper/main.go` to use bootstrap-aware construction.
6. Remove late `LoadSitesFromFlag(...)` loading from `api`/`worker` if it becomes redundant.
7. Add end-to-end tests for dynamic site command availability after bootstrap resolution.
8. Re-run `go test ./... -count=1`.

## Review guidance

When reviewing this change, start here:

- `pkg/cmd/app_config.go`
- `pkg/cmd/bootstrap.go`
- `pkg/cmd/root.go`
- `cmd/scraper/main.go`
- `pkg/cmd/site_test.go`

Then verify:

```bash
go test ./pkg/cmd/... -count=1
go test ./... -count=1
```

Finally sanity-check the binary behavior manually:

```bash
go run ./cmd/scraper --sites-manifest-dir ./sites site js-demo run seed --help
```