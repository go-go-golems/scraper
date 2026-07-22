---
Title: Slice 11 Process Isolation - Bounded Worker Protocol and OS Sandbox
Ticket: SCRAPER-WORKFLOW-V3
Status: active
Topics:
    - architecture
    - worker
    - scheduler
    - goja
    - workflows
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/workflowv3/types.go
      Note: Canonical isolation class and requested/effective policy identity
    - Path: repo://pkg/workflowv3/catalog.go
      Note: Host-authoritative task isolation maxima
    - Path: repo://pkg/workflowv3runtime/task_runner.go
      Note: Existing trusted in-process execution semantics reused inside workers
    - Path: repo://pkg/workflowv3runtime/isolation.go
      Note: Exact executable identity bounded launcher candidate validation and publication
    - Path: repo://pkg/workflowv3runtime/isolation_cgroup_linux.go
      Note: Delegated cgroup v2 memory process and aggregate CPU evidence
    - Path: repo://cmd/workflowv3-task-worker/main.go
      Note: Static least-authority worker executable
    - Path: repo://cmd/workflowv3-isolation-launcher/main.go
      Note: Static pre-exec cgroup join and Bubblewrap launcher
    - Path: repo://pkg/workflowv3runtime/engine.go
      Note: Parent-owned lease execution and fenced completion
ExternalSources: []
Summary: Implementation contract for exact isolation identity, a bounded worker protocol, Linux OS sandboxing, parent-owned leases, and validated candidate publication.
LastUpdated: 2026-07-22T01:10:00-04:00
WhatFor: Define Slice 11 before implementation so process separation is an enforceable security and durability boundary rather than a second execution engine.
WhenToUse: Read before adding isolation classes, worker subprocesses, broad modules, exec tools, resource limits, or untrusted Workflow V3 publishers.
---

# Slice 11 Process Isolation - Bounded Worker Protocol and OS Sandbox

## Executive summary

Slice 11 adds one host-compiled isolation decision to every executable task.
Trusted first-party tasks may remain `in-process.trusted`. Tasks selected as
`subprocess.restricted` execute in a fresh dedicated worker process inside an
OS sandbox. The parent retains the workflow lease, artifact authority, budget
settlement, output validation, and fenced completion. The child receives no
workflow database handle, inherited environment, host filesystem, ambient
network, or artifact-store credentials.

This is not a generic remote scheduler. It is a narrow attempt execution
transport under the existing Workflow V3 engine.

## Scope and non-goals

Included:

- canonical requested and effective isolation classes and policy digests;
- task-catalog host maxima and exact registry advertisement;
- one bounded versioned request and one bounded versioned result frame;
- a dedicated worker executable;
- fresh input/bundle staging and candidate output staging;
- Linux sandbox launch using Bubblewrap, no network namespace, empty
environment, read-only runtime/bundle/input mounts, writable attempt output,
process groups, delegated cgroup v2 controls, and rlimits;
- aggregate CPU-time, memory, process-count, wall-time, output-byte,
protocol-frame, file-count, and descriptor limits;
- parent cancellation and child-tree termination;
- typed child/protocol/limit failures and ordinary retry policy;
- parent validation/publication and existing lease-token/cancel-epoch fencing;
- projections and immutable attempt isolation identity.

Excluded from this slice:

- a Kubernetes scheduler, generic OCI image builder, remote workers, arbitrary
container networking, secret injection, or treating Goja as a hostile-code
sandbox;
- silently falling back to in-process execution when the configured sandbox is
unavailable;
- passing live database/network handles over the protocol.

A future `container.networked` profile may implement the same protocol, but an
unsupported class is rejected at compile/admission time rather than weakened.

## Canonical model

```go
type IsolationPolicy struct {
    Class            string `json:"class"`
    WallTimeMillis   int64  `json:"wallTimeMillis"`
    CPUTimeMillis    int64  `json:"cpuTimeMillis"`
    MemoryBytes      int64  `json:"memoryBytes"`
    MaxProcesses     int64  `json:"maxProcesses"`
    MaxOutputBytes   int64  `json:"maxOutputBytes"`
    MaxOutputFiles   int    `json:"maxOutputFiles"`
    MaxProtocolBytes int64  `json:"maxProtocolBytes"`
}

type PlanIsolation struct {
    Requested IsolationPolicy `json:"requested"`
    Effective IsolationPolicy `json:"effective"`
    PolicyDigest string `json:"policyDigest"`
    ExecutorDigest string `json:"executorDigest"`
}
```

Closed classes initially are:

- `in-process.trusted`;
- `subprocess.restricted`.

Task bundle catalog entries declare an isolation maximum/default. Worker boot
atomically advertises the SHA-256 identity of the exact static worker, static
cgroup launcher, Bubblewrap executable, protocol version, and fixed tool IDs and
bytes. JavaScript may request only a known class and safe integer bounds. Go
compiles effective policy by applying host limits; scripts cannot relax it.
Class, complete policy, policy digest, and executor digest enter plan, registry
generation, node, lease, attempt, projections, and worker request identity.
Dynamic map and reduction children inherit the exact compiled policy. Rolling
registry coexistence retains an executor set keyed by digest so generation A
can finish with A while B admits B; substitution is rejected.

`in-process.trusted` preserves Slices 1–10 defaults. Existing plans decode with
the trusted default, so migration is additive and deterministic.

## Protocol

One request frame is strict canonical JSON followed by newline:

```text
scraper-workflow-isolated-task-request/v1
  protocol version
  run/node/attempt/cancel epoch
  complete implementation identity and bundle digest
  effective isolation policy and digest
  relative bundle root and entrypoint
  declared input schemas plus relative staged paths
  declared output schemas and writable output root
  exact declared module aliases
```

One result frame is strict JSON followed by newline:

```text
scraper-workflow-isolated-task-result/v1
  echoed identity
  success { candidate output metadata, usage }
    or failure { closed class/code/retryable }
```

The child never returns artifact-store locators as authority. Candidate outputs
are relative files with schema, media type, size, and SHA-256. The parent opens
files beneath the attempt output directory without following symlinks,
recomputes size/digest, validates every expected port/schema and total limits,
publishes bytes through its own `ArtifactStore`, then calls ordinary fenced
completion.

Protocol rules:

- stdin/stdout each contain exactly one bounded frame;
- unknown fields, trailing data, duplicate outputs, absolute/traversal paths,
invalid UTF-8, wrong schema/version/identity, oversized frames, and malformed
JSON are rejected;
- stderr is bounded diagnostic output and never parsed as control state;
- no raw stderr or child exception text is persisted; stable codes are used;
- EOF, signal death, timeout, and protocol violation become typed failures.

## Sandbox contract

The Linux launcher resolves and pins an operator-configured worker executable
before admission. Bubblewrap creates:

- a new user, PID, IPC, UTS, and network namespace;
- a read-only root containing only required runtime files;
- read-only bundle and input mounts;
- one writable output mount and isolated `/tmp`;
- no home directory, workflow DB, artifact root, SSH agent, cloud metadata,
provider credentials, or inherited environment;
- `no_new_privs`, a process group, and bounded file descriptors;
- a pre-exec static launcher that joins a delegated cgroup before Bubblewrap can
  fork, enforcing aggregate `memory.max`/zero swap and `pids.max`;
- parent monitoring of cgroup aggregate CPU usage plus wall-time kill through
  `cgroup.kill`, with file-size and descriptor rlimits inside the worker.

Sandbox setup failure is configuration/infrastructure evidence and does not run
the task. It must never fall back. Tests detect platform capability explicitly;
production startup rejects a restricted profile when the launcher cannot prove
its controls.

A narrowly allowlisted isolated exec module may expose fixed tool IDs mapped by
the worker profile to absolute binaries and fixed argument schemas. JavaScript
never supplies a binary path, shell string, environment, working directory, or
redirection target. No `sh -c` surface exists.

## Parent execution and durability

1. Registry resolution and lease creation remain unchanged and atomic.
2. Parent resolves refs and stages immutable input bytes in a lease-local root.
3. Parent launches the exact effective isolation profile.
4. Heartbeat/cancellation remain parent responsibilities.
5. Child reports candidate files or typed failure.
6. Parent validates and publishes candidate bytes.
7. Existing `CompleteWithUsage` or `FailWithUsage` performs lease-token and
cancel-epoch fencing plus budget settlement.
8. Temporary staging is removed regardless of outcome.

No child opens SQLite. A child killed after producing bytes but before response
leaves only lease-local files. A parent crash may leave temporary files; restart
lease loss charges conservatively and retry starts a fresh sandbox. A stale
child cannot commit output because only the parent can call the store and must
still own the lease.

## Failure vocabulary

Stable failures include:

- `configuration/ISOLATION_PROFILE_UNAVAILABLE`;
- `identity/ISOLATION_POLICY_MISMATCH`;
- `protocol/ISOLATION_FRAME_INVALID`;
- `protocol/ISOLATION_FRAME_TOO_LARGE`;
- `resource/ISOLATION_WALL_TIME`;
- `resource/ISOLATION_MEMORY_LIMIT`;
- `resource/ISOLATION_PROCESS_LIMIT`;
- `resource/ISOLATION_OUTPUT_LIMIT`;
- `execution/ISOLATION_CHILD_EXIT`;
- `canceled/ISOLATION_CANCELED`.

Retryability is compiled policy, not inferred from text. Runtime construction
and sandbox setup failures remain separate from semantic retry debt where the
child never began domain execution.

## Security and privacy invariants

- Parent passes no process environment except a fixed safe locale/path policy.
- Request/result frames contain identities, schemas, safe relative paths,
bounds, usage, and digests—not source/provider payloads or secrets.
- Input and output bodies remain files outside SQLite and protocol frames.
- Child cannot see parent descriptors beyond stdin/stdout/stderr.
- Symlink/hardlink/device/FIFO candidate outputs are rejected.
- Worker executable identity and sandbox policy digest are advertised and
projected.
- Logs/events never persist argv containing secrets, raw child stderr, source
body, prompt, vector, provider body, or host paths.

## Migration and compatibility

Add node/attempt isolation columns and indexes through additive migration.
Existing rows become `in-process.trusted` with the canonical default policy
digest during read/migration. Exact plan JSON remains self-validating. Existing
bundle identities, retries, resources, budgets, gates, maps, reductions, and v2
behavior do not change.

## Test matrix

### Canonical/compiler

- direct Go/JavaScript equality and IR/plan/DTS goldens;
- unknown class, unsafe integer, zero/overflow bound, policy digest mismatch,
and requested relaxation rejected;
- task maximum clamps requested bounds deterministically;
- map/reduction inheritance and digest stability.

### Protocol and sandbox

- same pure task in-process and restricted produces equal output digest;
- strict wrong version/bundle/isolation/attempt identity rejection;
- malformed, trailing, oversized, duplicate, traversal, symlink, special-file,
and hash/size mismatch output rejection;
- fuzz frame decoder with a hard byte bound;
- environment/source/host filesystem/network denial canaries;
- wall-time, output-byte/file, process, memory, and descriptor bounds;
- fixed allowlisted tool succeeds while path/shell/undeclared tool fails.

### Durability/concurrency

- child kill becomes immutable attempt evidence and retries normally;
- parent cancellation kills child tree and stale candidate cannot publish;
- parent crash/lease loss followed by restart executes in a fresh sandbox;
- unrelated in-process and isolated resource classes refill independently;
- registry generation A/B and exact worker/profile identity remain pinned;
- budget actual/conservative/zero settlement remains correct;
- SQLite/WAL/events/projections contain no payload/environment canaries;
- full runtime/store/isolation race suites pass.

## Acceptance criteria

Slice 11 is complete only when restricted tasks demonstrably run outside the
scraper process under enforceable least-authority OS controls; identity and
limits are canonical and durable; the parent alone owns lease, artifacts,
budget, and completion; malformed/dead/canceled/over-limit children cannot
publish; existing trusted tasks and databases remain compatible; and all
focused, race, lint, type, build, migration, privacy, and repository validation
passes with real executable evidence.
