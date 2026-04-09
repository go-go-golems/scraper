---
Title: Bootstrap Config and Site Manifest Loading
Slug: scraper-bootstrap-config-and-site-manifest-loading
Short: "How scraper finds site manifest directories before building dynamic site commands."
Topics:
- scraper
- bootstrap
- config
- sites
- manifests
Commands:
- scraper
- site
- worker
- api
Flags:
- sites-manifest-dir
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

Scraper loads site manifests during a bootstrap phase that happens **before** the full Cobra command tree is built. This matters because dynamic site commands such as `site js-demo run seed` are discovered from the loaded site manifests. If scraper does not know where the site directories are yet, those commands do not exist.

## Why This Is A Bootstrap Concern

Normal Cobra flags are parsed after the command tree already exists. That is too late for scraper's site verbs, because the site command tree is created by scanning loaded manifests and JS verb files.

So scraper uses a two-phase startup model:

```text
raw CLI args
-> bootstrap manifest-dir discovery
-> load site manifests into registry
-> build Cobra command tree
-> normal Cobra parsing and execution
```

## Where Site Directories Can Come From

Scraper merges site manifest directories from three sources, in this order:

1. app config file
2. environment variable
3. bootstrap CLI flags

Later sources win only in ordering/append position; paths are normalized and de-duplicated.

## Config File

Default app config path is resolved through the standard Glazed app-config resolution for `scraper`.

Example `~/.scraper/config.yaml`:

```yaml
sitesManifestDirs:
  - /home/me/code/scraper-sites
  - /opt/shared-scraper-sites
```

## Environment Variable

Use:

```text
SCRAPER_SITES_MANIFEST_DIRS
```

The value is parsed with `filepath.SplitList(...)`, so on typical Unix systems it looks like:

```bash
export SCRAPER_SITES_MANIFEST_DIRS="/path/to/sites-a:/path/to/sites-b"
```

## CLI Flag

Use one or more repeated flags:

```bash
scraper \
  --sites-manifest-dir ./sites \
  --sites-manifest-dir ../extra-sites \
  site js-demo run seed --help
```

The bootstrap parser also accepts the `--sites-manifest-dir=/path` form.

## Common Commands

Run a site verb from the repo's default `sites/` directory:

```bash
go run ./cmd/scraper --sites-manifest-dir ./sites site js-demo run seed --help
```

Run the worker against the same manifest set:

```bash
go run ./cmd/scraper \
  --sites-manifest-dir ./sites \
  worker run \
  --sites-dir /tmp/scraper-sites \
  --engine-db /tmp/engine.db
```

Serve the API with the same manifest set:

```bash
go run ./cmd/scraper \
  --sites-manifest-dir ./sites \
  api serve \
  --sites-dir ./state/sites \
  --engine-db ./state/engine.db
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| `site js-demo run seed` is missing | scraper did not load the manifest directories during bootstrap | Pass `--sites-manifest-dir`, set `SCRAPER_SITES_MANIFEST_DIRS`, or configure `~/.scraper/config.yaml` |
| `--help` should work on a site verb but bootstrap fails first | The bootstrap parser should only extract manifest dirs | Verify you are on a build that includes the manual bootstrap scanner |
| Commands work in tests but not with `go run ./cmd/scraper` | Tests pass explicit manifest dirs while the real CLI does not | Add a bootstrap source for the real CLI invocation |
| Worker/API sees no sites | The root command was built without any manifest directories | Start those commands with the same bootstrap site-dir inputs as the `site` command |

## See Also

- `scraper help scraper-architecture-overview`
- `scraper help scraper-runtime-model`
- `scraper help scraper-adding-a-declarative-site`
- `scraper help scraper-new-developer-onboarding`
