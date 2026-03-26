# Architecture Research

**Domain:** Go CLI tool for Docker workspace management (LLM coding agents)
**Researched:** 2026-03-26
**Confidence:** HIGH — spec provides authoritative structure; supplemented with ecosystem verification

---

## Standard Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         cmd/ (Cobra Layer)                           │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │  launch  │ │  init    │ │  status  │ │  logs    │ │  stop /  │  │
│  │  join    │ │  wizard  │ │  ls      │ │  build   │ │  destroy │  │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘  │
│       │ signal.NotifyContext, TTY detection, exit code mapping        │
├───────┴───────────────────────────────────────────────────────┬──────┤
│                      internal/tui/                            │      │
│  ┌─────────────┐ ┌───────────────┐ ┌──────────┐ ┌──────────┐ │      │
│  │ init_wizard │ │ build_progress│ │status_view│ │log_viewer│ │      │
│  │ (BubbleTea) │ │  (BubbleTea)  │ │(BubbleTea)│ │(BubbleTea)│ │      │
│  └─────────────┘ └───────────────┘ └──────────┘ └──────────┘ │      │
├───────────────────────────────────────────────────────────────┘      │
│                      internal/ (Domain Layer)                         │
│  ┌───────────────┐ ┌──────────────┐ ┌────────────┐ ┌─────────────┐  │
│  │  config/      │ │  harness/    │ │  cache/    │ │  network/   │  │
│  │  types.go     │ │  interface   │ │  hash.go   │ │  firewall   │  │
│  │  config.go    │ │  registry    │ │  lock.go   │ │  rules.go   │  │
│  │  merge.go     │ │  claude_code │ │  cache.go  │ │  matcher.go │  │
│  │  validate.go  │ │  custom      │ │            │ │             │  │
│  └───────┬───────┘ └──────┬───────┘ └─────┬──────┘ └──────┬──────┘  │
│          │                │               │               │           │
│  ┌───────┴───────────────────────────────────────────────┴──────────┐ │
│  │                       internal/docker/                            │ │
│  │  manager.go  dockerfile.go  entrypoint.go  network.go  naming.go │ │
│  └──────────────────────────────┬────────────────────────────────────┘ │
├─────────────────────────────────┼────────────────────────────────────┤
│                    pkg/templates/ (Embedded Templates)                │
│  ┌─────────────────┐ ┌───────────────────┐ ┌──────────────────────┐  │
│  │  Dockerfile.tmpl│ │ entrypoint.sh.tmpl│ │  zone-bashrc.tmpl    │  │
│  └─────────────────┘ └───────────────────┘ └──────────────────────┘  │
├─────────────────────────────────┼────────────────────────────────────┤
│                    External Boundaries                                │
│  ┌────────────────┐  ┌──────────────────┐  ┌───────────────────────┐ │
│  │  Docker Daemon │  │  Host iptables   │  │  ~/.config/zone/      │ │
│  │  (SDK + CLI)   │  │  (sudo iptables) │  │  .zone/ cache dir     │ │
│  └────────────────┘  └──────────────────┘  └───────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Boundary Rule |
|-----------|----------------|---------------|
| `cmd/*` | Cobra command definitions, flag parsing, TTY detection, signal handling, exit code mapping | Calls internal packages; never called by them |
| `internal/tui/` | BubbleTea models for init wizard, build progress, status, log tailing | Receives data from cmd layer; renders UI |
| `internal/config/` | TOML parsing (strict), two-tier merge, schema validation, dangerous mount blocking | Produces `MergedConfig`; consumed by docker and harness |
| `internal/harness/` | Harness interface, registry, BaseHarness, concrete implementations | Queried by docker for Dockerfile generation inputs |
| `internal/docker/` | Docker SDK calls (build, create, start, stop, inspect, list), template generation, naming | Central orchestrator; depends on config, cache, harness, network, templates |
| `internal/cache/` | `.zone/` directory read/write, flock-based locking, config hash computation | Used by docker and cmd; never calls out to Docker |
| `internal/network/` | Host-side iptables rule generation, DNS resolution, glob rule matching | Called by docker/network.go at container start/stop |
| `pkg/templates/` | `//go:embed` declarations for Dockerfile, entrypoint, bashrc templates | Pure data; imported by docker and cache (for hash) |

---

## Recommended Project Structure

The spec defines this structure exactly. It is the correct layout for this project:

```
zone/
├── main.go                        # entry: version ldflags, root cmd execution
├── go.mod / go.sum
├── Makefile
├── .goreleaser.yml
├── cmd/
│   ├── root.go                    # global flags, TTY detection, log level, signal setup
│   ├── init.go                    # zone init → tui.InitWizard
│   ├── launch.go                  # zone launch → docker.Manager.Launch (idempotent)
│   ├── join.go                    # zone join → docker.Manager.Join
│   ├── exec.go                    # zone exec → docker.Manager.Exec
│   ├── shell.go                   # zone shell → docker.Manager.Shell
│   ├── build.go                   # zone build → docker.Manager.Build
│   ├── stop.go                    # zone stop → docker.Manager.Stop
│   ├── restart.go                 # zone restart → stop + launch
│   ├── ls.go                      # zone ls → docker.Manager.ListAll
│   ├── logs.go                    # zone logs → docker.Manager.Logs + tui.LogViewer
│   ├── clean.go                   # zone clean → cache.Clean
│   ├── destroy.go                 # zone destroy → docker + cache teardown
│   ├── status.go                  # zone status → docker.Manager.Status + tui.StatusView
│   ├── config.go                  # zone config → config.Show (merged + annotated)
│   └── validate.go                # zone validate → config.Validate, harness.Validate
├── internal/
│   ├── config/
│   │   ├── types.go               # Config, GlobalConfig, MergedConfig structs
│   │   ├── harness_config.go      # Typed HarnessConfig (union of all harness fields)
│   │   ├── config.go              # TOML decode (strict), per-repo parsing
│   │   ├── global.go              # XDG global config read/write
│   │   ├── merge.go               # Two-tier merge (scalar override, list union)
│   │   └── validate.go            # Unknown keys, dangerous mounts, env var warnings
│   ├── cache/
│   │   ├── cache.go               # .zone/ read/write operations
│   │   ├── hash.go                # SHA256 of config + templates + version
│   │   └── lock.go                # syscall.Flock exclusive lock
│   ├── docker/
│   │   ├── manager.go             # Docker SDK client, all container operations
│   │   ├── dockerfile.go          # text/template rendering → Dockerfile string
│   │   ├── entrypoint.go          # text/template rendering → entrypoint.sh string
│   │   ├── shellrc.go             # text/template rendering → zone-bashrc string
│   │   ├── naming.go              # sha256-based container + network name derivation
│   │   ├── network.go             # Docker network create/destroy, iptables dispatch
│   │   ├── platform.go            # OS/rootless/iptables detection via Docker Info
│   │   └── errors.go              # Sentinel errors (ErrNoContainer, ErrDockerNotRunning...)
│   ├── network/
│   │   ├── firewall.go            # sudo iptables rule generation, cleanup, refresh goroutine
│   │   ├── rules.go               # Whitelist/blocklist rule set parsing
│   │   └── matcher.go             # Precompiled glob matcher (filepath.Match semantics)
│   ├── harness/
│   │   ├── harness.go             # Harness interface, BaseHarness, registry.Get()
│   │   ├── claude_code.go         # Fully implemented
│   │   ├── opencode.go            # Stub (Validate() returns not-implemented error)
│   │   ├── gemini_cli.go          # Stub
│   │   ├── aider.go               # Stub
│   │   ├── codex_cli.go           # Stub
│   │   └── custom.go              # config-driven custom harness
│   └── tui/
│       ├── init_wizard.go         # BubbleTea model: harness selection + config preview
│       ├── build_progress.go      # BubbleTea model: streaming build output
│       ├── status_view.go         # BubbleTea model: live container status
│       └── log_viewer.go          # BubbleTea model: log tailing with follow
├── pkg/
│   └── templates/
│       ├── templates.go           # //go:embed declarations
│       ├── Dockerfile.tmpl        # Go text/template
│       ├── entrypoint.sh.tmpl     # Go text/template
│       └── zone-bashrc.tmpl       # Go text/template
└── tests/
    ├── config_merge_test.go       # Merge strategy correctness (write first)
    ├── harness_validate_test.go   # Per-harness field rejection
    ├── validate_test.go           # Dangerous mount detection, symlink resolution
    ├── naming_test.go             # Deterministic container name generation
    ├── matcher_test.go            # Network glob rule matching
    └── hash_test.go               # Cache hash includes templates + version
```

### Structure Rationale

- **`cmd/`:** Each file = one Cobra command. Commands are thin: parse flags, construct context, call `internal/` packages, map sentinel errors to exit codes. No business logic here.
- **`internal/`:** Go compiler enforces that nothing outside this module imports these packages. Domain logic is fully encapsulated.
- **`internal/docker/`:** The central orchestrator package. It depends on config, cache, harness, network, and templates — but nothing in `internal/` depends on `docker/` (prevents cycles).
- **`pkg/templates/`:** Publicly importable (`pkg/` not `internal/`) so the cache hash computation in `internal/cache/hash.go` can include template contents. Also allows future external tooling to inspect templates.
- **`internal/tui/`:** Isolated from Docker logic. Receives data structs from cmd layer; cmd layer decides when to use TUI vs plain text based on TTY detection in `root.go`.
- **`internal/network/`:** Separated from `internal/docker/` because iptables logic is platform-specific and independently testable. Docker package calls network package for rule application.

---

## Enforced Import Graph

This is the explicit constraint from the spec. Violations cause circular imports or break encapsulation:

```
cmd/* → internal/* (OK)
cmd/* → pkg/templates (OK)
internal/docker → internal/config (OK)
internal/docker → internal/cache (OK)
internal/docker → internal/network (OK)
internal/docker → internal/harness (OK)
internal/docker → pkg/templates (OK)
internal/cache → pkg/templates (OK, for hash)
internal/* → cmd/* (FORBIDDEN)
internal/config → internal/docker (FORBIDDEN, would be circular)
internal/harness → internal/docker (FORBIDDEN, would be circular)
```

Build this graph consciously. Violations are caught by the compiler, not at runtime.

---

## Architectural Patterns

### Pattern 1: Thin Command, Fat Manager

**What:** Cobra command files (`cmd/*.go`) contain only flag binding, context construction, and error-to-exit-code mapping. All orchestration logic lives in `internal/docker/manager.go`.

**When to use:** Always. This is the primary structural rule for this project.

**Trade-offs:** Commands become nearly identical boilerplate, but this is intentional. It makes the business logic testable without spinning up Cobra. The manager can be called from tests directly.

**Example:**
```go
// cmd/launch.go
func runLaunch(cmd *cobra.Command, args []string) error {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()

    cfg, err := config.LoadMerged(".")
    if err != nil { return err }

    c := cache.New(".")
    mgr, err := docker.NewManager(cfg, c)
    if err != nil { return err }
    defer mgr.Close()

    return mgr.Launch(ctx, launchOpts)
    // Exit code mapping happens in root.go's PersistentPostRun
}
```

### Pattern 2: Sentinel Errors + Exit Code Mapping

**What:** All `internal/` functions return typed `error` values. Sentinel errors in `internal/docker/errors.go` are detected in `cmd/root.go` using `errors.Is()` and mapped to structured exit codes (0–6).

**When to use:** Everywhere. No package in `internal/` ever calls `os.Exit()`.

**Trade-offs:** Requires a central error dispatch in the cmd layer. But it means the entire internal domain is independently testable and exit codes are consistently enforced.

**Example:**
```go
// cmd/root.go PersistentPostRun or main.go
func handleError(err error) int {
    switch {
    case errors.Is(err, docker.ErrDockerNotRunning): return 3
    case errors.Is(err, docker.ErrNoContainer):      return 6
    case errors.Is(err, cache.ErrLockContention):    return 5
    // ... etc
    default: return 1
    }
}
```

### Pattern 3: Harness Plugin Interface + Registry

**What:** All harnesses implement a common `Harness` interface. A registry maps string names to factory functions. New harnesses add one file; no existing code changes.

**When to use:** For the harness plugin system. The interface is the contract between config and Dockerfile generation.

**Trade-offs:** Typed `HarnessConfig` (union struct with all possible fields) is more verbose than `map[string]interface{}` but provides type safety and per-harness validation. Each harness's `Validate()` method rejects fields that don't belong to it.

**Example:**
```go
// Adding opencode later: add opencode.go, add one line to registry map.
// Nothing else changes.
```

### Pattern 4: Channel-Based Build Progress Streaming

**What:** `docker.Manager.Build()` returns a `<-chan BuildProgress` channel, not a final result. The caller (cmd layer or TUI model) drains the channel to display progress. The BubbleTea model receives updates as `tea.Cmd` messages.

**When to use:** All long-running Docker SDK operations (image build). Not needed for quick operations (start, stop, inspect).

**Trade-offs:** Slightly more complex than blocking calls, but enables real-time progress display and clean cancellation via `ctx.Done()`.

### Pattern 5: SDK for Lifecycle, CLI exec for Interactive TTY

**What:** Use `github.com/docker/docker/client` (SDK) for build, create, start, stop, inspect, list, and network operations. Use `os/exec` with `docker exec -it` for interactive terminal attachment (launch, join, shell, exec).

**When to use:** Always. The SDK's `ContainerExecAttach` + hijacked connection API handles terminal resize events and signal forwarding poorly. Delegating to the Docker CLI binary solves this reliably.

**Trade-offs:** Requires Docker CLI binary to be on PATH for interactive commands. Acceptable since Zone already requires Docker to be installed.

### Pattern 6: Cobra + BubbleTea Handoff via TTY Gate

**What:** `cmd/root.go` detects TTY status on startup using `term.IsTerminal(os.Stdin.Fd())`. A global bool `globalPlainMode` is set. Every command that would start a BubbleTea program checks this flag first and falls back to plain text output.

**When to use:** All interactive commands (init, build, status, logs).

**Trade-offs:** Requires every TUI model to have a "plain text path." But this is necessary for CI, piped output, and `--json` scriptability. The `--plain` flag provides a manual override.

**Example flow:**
```
zone build
  → cmd/build.go checks globalPlainMode
  → if false: starts tui.BuildProgress(buildChan)
  → if true:  reads buildChan directly, writes lines to stdout
```

---

## Data Flow

### Primary Flow: `zone launch`

```
User: zone launch --harness claude-code
    ↓
cmd/root.go: TTY detection → globalPlainMode=false
cmd/root.go: signal.NotifyContext(SIGINT, SIGTERM)
    ↓
cmd/launch.go:
    config.LoadMerged(".") → reads zone.toml + ~/.config/zone/config.toml → MergedConfig
    cache.New(".")         → opens .zone/ directory, acquires flock
    docker.NewManager(cfg, cache) → SDK client.NewClientWithOpts + Ping
    mgr.Launch(ctx, opts)
    ↓
docker/manager.go Launch():
    1. inspect cache → container_id exists? → SDK ContainerInspect
       a. running + hash unchanged → attach (skip build)
       b. running + hash changed  → warn, attach anyway
       c. exited/dead             → remove + proceed
       d. stale container ID      → clean cache + proceed
    2. check config.hash vs cache.hash
       a. match + valid image_id  → run from cached image
       b. mismatch or no image    → regenerate + rebuild
    3. if rebuild needed:
       harness.Get(cfg.Harness, cfg.HarnessConfig) → Harness interface
       dockerfile.Generate(harness, cfg) → text/template → string → .zone/Dockerfile
       entrypoint.Generate(harness, cfg) → string → .zone/entrypoint.sh
       shellrc.Generate(harness, cfg)    → string → .zone/zone-bashrc
       SDK ImageBuild → <-chan BuildProgress → tui.BuildProgress or plain stdout
       cache.WriteImageID(imageID)
    4. if network.mode != "none":
       network.Create() → SDK NetworkCreate (bridge, no --internal)
       network/firewall.ApplyRules(cfg.NetworkRules, networkBridge)
           → resolve hostnames → sudo iptables rules → .zone/firewall.rules
    5. SDK ContainerCreate (hostConfig: no-new-privs, cap drop ALL, pids limit)
       cache.WriteContainerID(containerID)
       SDK ContainerStart
       cache.ReleaseLock()           ← lock released BEFORE TTY attach
    6. if --headless: return (print container ID)
       else: attachInteractive() → os/exec docker exec -it
    ↓
User sees harness prompt inside container
```

### Config Data Flow

```
~/.config/zone/config.toml   ./zone.toml
         ↓                        ↓
    config.LoadGlobal()    config.LoadRepo()
         ↓                        ↓
         └──── config.Merge() ────┘
                     ↓
              MergedConfig
          (scalar: repo wins,
           lists: union/append)
                     ↓
    ┌────────────────┬────────────────┐
    ↓                ↓                ↓
harness.Get()  docker.Manager   cache.Hash()
(validates     (builds, runs)   (includes templates
 config keys)                    + zone version)
```

### TUI Data Flow (BubbleTea)

```
cmd layer constructs data channel / initial state
    ↓
tea.NewProgram(model)
    ↓
model.Init() → returns initial tea.Cmd (e.g., waitForBuildMsg)
    ↓
model.Update(msg) → receives BuildProgress messages from channel
                  → returns updated model + next tea.Cmd
    ↓
model.View() → renders current state via Lip Gloss styles
    ↓
[if --plain or non-TTY: skip tea.NewProgram, cmd layer reads channel directly]
```

### Cache State Machine

```
.zone/ state transitions:

[empty]
  → zone init / zone launch: write Dockerfile, entrypoint, hash
      ↓
[templates only, no image]
  → docker build succeeds: write image_id
      ↓
[image cached]
  → container started: write container_id, network_id
      ↓
[container running]
  → zone stop: clear container_id, network_id (image_id retained)
      ↓
[image cached]  ← fast re-launch skips build
  → zone clean --all: remove image_id, Dockerfile
      ↓
[empty]
```

---

## Component Build Order

This ordering minimizes blocked work and ensures each phase builds on stable foundations:

| Order | Component | Depends On | Rationale |
|-------|-----------|------------|-----------|
| 1 | `internal/config/` | nothing | All other packages need MergedConfig. Build and test first. |
| 2 | `internal/cache/` | config (for hash) | Docker manager needs cache. Lock and hash logic is independently testable. |
| 3 | `pkg/templates/` | nothing | Pure template files + embed. Required by cache/hash and docker/dockerfile. |
| 4 | `internal/harness/` | config | Harness interface and claude-code impl. Docker needs harness for Dockerfile generation. |
| 5 | `internal/network/` | config | Firewall/matcher logic. Needed by docker/network.go. Testable without Docker. |
| 6 | `internal/docker/` | config, cache, harness, network, templates | Core orchestrator. Build after all dependencies are stable. |
| 7 | `internal/tui/` | (data from cmd layer) | BubbleTea models. Can develop in parallel with docker once data types are known. |
| 8 | `cmd/*` | all internal packages | Command wiring. Done last; validates integration. |

**Parallelizable:** Steps 3, 4, and 5 can be developed in parallel once step 1 (`config/types.go`) is complete, since they only depend on config types.

---

## Integration Points

### External Service Boundaries

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| Docker Daemon | SDK `client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())` | Ping on init to detect "daemon not running" early |
| Docker CLI binary | `os/exec docker exec -it` for interactive TTY attach | SDK's `ContainerExecAttach` is insufficient for raw terminal I/O |
| Host iptables | `os/exec sudo iptables` via `internal/network/firewall.go` | Linux-only; guarded by `platform.SupportsIPTables` |
| Global config (~/.config/zone/) | XDG spec path; direct file I/O | Created on first `zone init` |
| Repo cache (.zone/) | Direct file I/O + `syscall.Flock` | Gitignored; one per repo |

### Internal Package Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| `cmd` ↔ `docker/manager` | Direct function calls; manager is injected with config + cache | cmd never imports docker types beyond Manager |
| `cmd` ↔ `tui` | cmd constructs model with data, calls `tea.NewProgram(model).Run()` | TUI models are pure BubbleTea; cmd handles plain text fallback |
| `docker` ↔ `harness` | `harness.Get()` returns Harness interface; docker calls interface methods | No concrete harness types cross the boundary |
| `docker` ↔ `network` | `network.firewall.ApplyRules()` called from `docker/network.go` | Firewall goroutine (5 min refresh) owned by docker layer |
| `docker` ↔ `templates` | Import `pkg/templates` for string constants; pass to `text/template.Execute()` | Templates are embedded at compile time |
| `cache` ↔ `templates` | `pkg/templates` string values included in SHA256 hash | Ensures template changes invalidate cached images |
| `config` ↔ `harness` | `config.HarnessConfig` (typed union struct) passed to `harness.Get()` | Harness validates only its own fields in `Validate()` |

---

## Anti-Patterns

### Anti-Pattern 1: Business Logic in cmd/ Files

**What people do:** Put container lifecycle decisions, config merge logic, or hash comparisons directly in `cmd/launch.go`.

**Why it's wrong:** Makes the orchestration logic impossible to unit test without Cobra scaffolding. Creates coupling between CLI argument parsing and domain behavior.

**Do this instead:** `cmd/launch.go` creates context and calls `mgr.Launch(ctx, opts)`. All idempotency logic lives in `docker/manager.go`.

### Anti-Pattern 2: Using SDK for Interactive TTY Attachment

**What people do:** Use `client.ContainerExecAttach()` + `types.HijackedResponse` for interactive sessions.

**Why it's wrong:** Terminal resize events (SIGWINCH), signal forwarding, and raw mode restoration are extremely difficult to handle correctly through the SDK's hijacked connection. Real-world tools that got this wrong produce broken TTYs and hung processes.

**Do this instead:** Use `os/exec` with `docker exec -it` for all interactive commands. The Docker CLI binary handles TTY correctly. Use the SDK for everything non-interactive.

### Anti-Pattern 3: Flat Package Structure

**What people do:** Put all Go files in the root or a single `internal/` directory as the project grows.

**Why it's wrong:** Config, cache, harness, network, and Docker logic all have different dependency directions. Mixing them creates import cycles and makes the dependency graph opaque.

**Do this instead:** Follow the enforced import graph from the spec. Separate packages for each concern. The compiler will catch violations.

### Anti-Pattern 4: Sharing TUI State with Domain Logic

**What people do:** Pass BubbleTea model types into docker manager functions, or have manager functions return lipgloss-styled strings.

**Why it's wrong:** TUI rendering must be bypassed in plain-text mode (CI, pipes, `--json`). Coupling rendering to domain logic makes plain-text fallback a rewrite.

**Do this instead:** Domain functions return plain data types (`ContainerStatus`, `BuildProgress`, etc.). cmd layer decides whether to feed these to a BubbleTea model or print them as plain text.

### Anti-Pattern 5: In-Container Network Enforcement

**What people do:** Add `CAP_NET_ADMIN` to the container and run iptables inside it to filter outbound traffic.

**Why it's wrong:** An LLM agent with shell access can disable or modify in-container firewall rules. The enforcement boundary is the security guarantee.

**Do this instead:** Host-side iptables only. Container never gets `CAP_NET_ADMIN`. The `CapDrop: ALL` + minimal `CapAdd` list is the correct security posture.

---

## Scaling Considerations

Zone is a local CLI tool. "Scaling" means: handling more repos, more harnesses, more commands without architectural debt.

| Concern | Current (v1) | Future |
|---------|--------------|--------|
| New harness | Add one file to `internal/harness/`, one line in registry | No other changes needed |
| New command | Add one file to `cmd/`, register in `root.go` | Manager gets new method if needed |
| macOS network filtering | Warn + skip (Linux-only) | Phase 2: DNS proxy sidecar container in a new `internal/proxy/` package |
| Advanced glob patterns | `filepath.Match` (no `**`) | Phase 2: swap to `gobwas/glob` in `internal/network/matcher.go` only |
| Config schema v2 | `version = 1` only | `zone migrate` command + version dispatch in `config/config.go` |

---

## Sources

- Zone spec v4.0 (`/workspace/zone/zone-spec.md`) — Section 7 (project structure), Section 12 (Docker Manager), Section 10 (harness interface), Section 9 (signal handling) — HIGH confidence
- [Cobra documentation](https://cobra.dev/) — command structure, flag patterns — HIGH confidence
- [BubbleTea package docs](https://pkg.go.dev/github.com/charmbracelet/bubbletea) — Elm architecture, TTY detection — HIGH confidence
- [Docker Go SDK docs](https://pkg.go.dev/github.com/docker/docker/client) — client initialization, API methods — HIGH confidence
- [Inngest: Interactive CLIs with BubbleTea](https://www.inngest.com/blog/interactive-clis-with-bubbletea) — Cobra + BubbleTea integration pattern — MEDIUM confidence
- [Charming Cobras with BubbleTea](https://elewis.dev/charming-cobras-with-bubbletea-part-1) — TTY detection + BubbleTea handoff — MEDIUM confidence
- [Docker iptables documentation](https://docs.docker.com/engine/network/firewall-iptables/) — DOCKER-USER chain, host-side rules — HIGH confidence
- [Go project structure patterns](https://www.glukhov.org/post/2025/12/go-project-structure/) — `cmd/` + `internal/` conventions — MEDIUM confidence

---

*Architecture research for: Zone — Go CLI Docker workspace manager for LLM coding agents*
*Researched: 2026-03-26*
