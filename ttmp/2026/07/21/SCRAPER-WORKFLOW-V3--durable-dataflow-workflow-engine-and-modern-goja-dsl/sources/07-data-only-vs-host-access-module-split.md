<!-- Source: https://parc.yolo.scapegoat.dev/note/research/kb/tribal/data-only-vs-host-access-module-split -->
<!-- Retrieved: 2026-07-21 -->

[Terminology and agent guide](https://parc.yolo.scapegoat.dev/AGENTS.md)

modifiedJul 20, 2026

tags

aliasessafe defaults for runtimes, host-access module split, capability split, data-only modules

createdMay 11, 2026

statusactive

typeknowledge-base

## Data-Only vs Host-Access Module Split — How We Do It

Related foundation: [Host-Mediated Sandbox Principles](https://parc.yolo.scapegoat.dev/note/research/kb/fundamentals/host-mediated-sandbox-principles)

≡ Summary

In embedded or sandboxed JS runtimes, modules that only compute on data are safe defaults. Modules that touch the outside world are explicit opt-ins. We use this split to keep runtimes safe-by-default and capability-driven. Three projects converge on it: Node-like Primitives, Capsule Lab, and the generic goja embedding pattern.

## The pattern

We divide host-exposed runtime capabilities into two buckets:

1. **Data-only modules**
	- transform values
		- parse/format
		- crypto helpers
		- path/string utilities
		- timers or other inert coordination primitives
2. **Host-access modules**
	- filesystem
		- OS/process access
		- network
		- display/canvas
		- device I/O
		- anything that mutates or observes the outside world

The rule:

```
Data-only can be default-enabled.
Host-access must be explicitly installed or permitted.
```

## Why we do it this way

**Safe-by-default runtimes are easier to reason about.** A fresh goja runtime that can do math and string manipulation but cannot open files or spawn processes is a much safer baseline than Node's "everything is there" model.

**Capabilities become visible in host code.** In Node-like Primitives, you choose whether to install `fs`, `os`, `exec`, or only safe modules. In Capsule Lab, the capsule declares permissions and the host only exposes matching APIs.

**Permission models stay small.** If the default runtime surface already has powerful modules, then every script is implicitly privileged. Splitting modules by capability lets you start from zero and add only what the use case needs.

## Where it lives

| Repo | Path | Use |
| --- | --- | --- |
| `go-go-goja` | `engine/factory.go`, `modules/*` | builder chooses safe defaults vs host-access modules |
| `2026-04-02--capsule-lab` | kernel + host shell | permission-locked API surface for capsules |
| `goja-embedding` pattern | runtime setup | host installs only allowed side-effect APIs |

### Related PARC project reports

- [go-go-goja Node-like Primitives: Technical Deep Dive](https://parc.yolo.scapegoat.dev/note/projects/2026/04/25/proj-go-go-goja-node-like-primitives-technical-deep-dive) — explicit split between safe primitives and host-access modules
- [Capsule Lab: A Sandboxed JS Capsule Runtime in the Browser](https://parc.yolo.scapegoat.dev/note/projects/2026/04/02/proj-capsule-lab-a-sandboxed-js-capsule-runtime-in-the-browser) — permission declarations determine which host APIs exist
- [goja: Embedding a JavaScript Interpreter in Go — How We Do It](https://parc.yolo.scapegoat.dev/note/research/kb/tribal/goja-embedding-in-go) — fresh runtime + host-mediated side effects as the generic form

## Common mistakes

1. **Treating `fs` as a harmless default.** The moment filesystem access is default-installed, the runtime is no longer sandbox-like.
2. **Conflating timers with I/O.** Timers coordinate execution but don't inherently expose external state. Group them with safe primitives unless their callbacks imply hidden privileges.
3. **Adding one-off escape hatches.** A single convenience function like `readFileIfExists()` is still filesystem access.
4. **Installing modules before permissions are known.** In permission-driven hosts like Capsule Lab, capability exposure must happen after permission evaluation, not before.
5. **Thinking wrappers make a capability safe.** A thin wrapper around the filesystem is still filesystem access.
6. **Letting tests bypass the split.** If tests always initialize the runtime with every module, they stop proving the permission model.

## Variations

- **Builder-driven split** — Node-like Primitives uses runtime builders and explicit module registration.
- **Permission-driven split** — Capsule Lab installs APIs based on declared permissions.
- **Host-API split** — generic goja embedding exposes only the side-effect functions the host chooses to `Set`.
