# Phase 8: CLI Commands & DX - Research

**Researched:** 2026-03-30
**Domain:** Go CLI (Cobra), Docker SDK v28, signal handling, JSON output, exit codes
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**init command (CLI-01, CLI-02)**
- D-01: `zone init --harness <name>` scaffolds zone.toml non-interactively: write commented template file, add `.zone/` to `.gitignore`, print confirmation message
- D-02: `zone init` without `--harness` in Phase 8: error with "Interactive mode requires a terminal. Use `--harness <name>` for non-interactive init."
- D-03: `--set key=value` flag for dotted-path config overrides (e.g., `--set resources.memory=8g`) — applies overrides to the generated zone.toml
- D-04: Detect existing zone.toml: error with "zone.toml already exists. Use `zone config` to modify it."
- D-05: Detect harness hints: check for `.claude/` directory, `CLAUDE.md`, `.aider*`, etc. — print "Detected: <harness> indicators" hint
- D-06: Generated zone.toml includes commented-out sections showing all available fields per spec §13

**ls command (CLI-12)**
- D-07: `Manager.List(ctx)` method queries Docker API with label filter `com.zone.managed=true`, returns `[]ContainerInfo` structs
- D-08: `cmd/ls.go` formats as plain text table: NAME, HARNESS, STATUS, UPTIME, REPO columns
- D-09: `--json` flag outputs JSON array to stdout (bypass table formatting)
- D-10: `--running` flag filters to running containers only
- D-11: `--quiet/-q` flag prints only container names (one per line) for scripting
- D-12: ls does NOT require zone.toml — it's a global discovery command using Docker client directly

**logs command (CLI-13, CLI-14)**
- D-13: Plain text output via Docker SDK `ContainerLogs(ctx, id, types.ContainerLogsOptions{...})`
- D-14: `--follow/-f` flag: set `Follow: true` in options, stream to stdout via `io.Copy`
- D-15: `--build` flag: read `.zone/logs/last_build.log` from cache and print to stdout (no Docker needed)
- D-16: `--tail N` flag: set `Tail: "N"` in options (Docker SDK handles this)
- D-17: When stdout is piped, output plain text (no TUI chrome) — this is the Phase 8 default behavior
- D-18: Requires running container (unless `--build`); error with exit code 6 if no container

**status command (CLI-17)**
- D-19: One-shot container inspection via Docker SDK `ContainerInspect` — print formatted plain text
- D-20: Display: Repo, Harness, Container name, Status (with uptime), Image ID, Network mode, Port mappings, Resource limits, Mounts
- D-21: `--json` flag: output raw container inspect JSON to stdout
- D-22: Requires running or stopped container; error with exit code 6 if no container

**Structured exit codes (DX-01)**
- D-23: Extend main.go with full sentinel error → exit code mapping per spec §3.3
- D-24: Use `errors.Is()` chain in main.go — order matters: check specific errors before generic fallback
- D-25: Add sentinel errors to `internal/docker/errors.go` if missing: `ErrNoContainer` (exists), `ErrNetworkUnsupported` (missing)

**Signal handling (DX-04, DX-05)**
- D-26: Add `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)` to all commands that call Docker Manager: launch, join, exec, shell, build, stop, restart, destroy, clean, logs, status
- D-27: Pass the context through to Manager methods — all Docker SDK calls already take context from Phase 6
- D-28: Ctrl+C during `zone launch` sends SIGINT to harness process inside container; container stays alive — handled by the `docker exec -it` subprocess naturally

**Remediation hints (DX-02)**
- D-29: Create `cmd/errors.go` with `mapError(err error) (string, int)` function that returns remediation message + exit code
- D-30: Remediation hints added at cmd layer — internal packages return clean sentinel errors, cmd layer wraps with user-facing messages
- D-31: Use fmt.Fprintf to stderr for remediation hints (separate from command output to stdout)
- D-32: Follow spec §3.12 format: "Error: <message>\n\n  <platform-specific hints>\n\n<context line>"

**JSON output (DX-03)**
- D-33: `--json` on `zone status`: output container inspection as JSON object
- D-34: `--json` on `zone ls`: output as JSON array of container info objects
- D-35: `--json` on `zone logs`: output log lines as JSON array with timestamp fields
- D-36: `zone config --json` already implemented in Phase 2 — no changes needed

**Command aliases (DX-08)**
- D-37: Aliases already defined in Phase 1 stubs: launch/up, stop/down, ls/list, logs/log, status/st — verify all are present and working

**Help text with examples (DX-09)**
- D-38: All 15 commands get `Long` field with 2-4 usage examples per spec
- D-39: Example format: indented, showing common use cases for each command
- D-40: `zone launch` examples: basic, headless, zero-config, with prompt, with harness args

**Ad-hoc port flag (CLI-21 partial, deferred from Phase 7)**
- D-41: Add `--port/-P` flag to `zone launch` command — repeatable string flag
- D-42: Parse same `host:container` format as config ports, merge with config ports
- D-43: Pass merged ports list to Manager.Launch opts

**Harness exit behavior (DX-06)**
- D-44: When harness process exits (user types /exit): the `docker exec -it` subprocess returns, launch command returns exit code 0
- D-45: Container stops after harness exit — this is the entrypoint `exec` behavior from Phase 4

**Stop cleanup (DX-07)**
- D-46: Already implemented in Phase 6 Manager.Stop() — verify: stop container, remove container, remove network, clear IDs from cache

### Claude's Discretion
- ContainerInfo struct field layout
- Exact table column widths/formatting in ls
- Harness hint detection file patterns
- Help text example wording
- Remediation hint exact wording beyond spec examples
- Error message ordering in main.go

### Deferred Ideas (OUT OF SCOPE)
- TUI BubbleTea views (init wizard, status live view, logs viewer, build progress) → Phase 9
- `--edit` flag on `zone config` (opens $EDITOR) → Phase 9 or backlog
- `--schema` flag on `zone config` → backlog
- `--from-devcontainer` migration on `zone init` → v2 backlog
- Network-related exit code 4 detailed handling → Phase 10
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CLI-01 | User can run `zone init` to scaffold zone.toml with interactive harness selection | D-01 through D-06: non-interactive path with --harness flag; interactive deferred to Phase 9 |
| CLI-02 | User can run `zone init --harness <name>` to scaffold non-interactively | D-01: QuickstartWriteZoneToml extended for full commented template |
| CLI-12 | User can run `zone ls` to list all zone containers across all repos | D-07 through D-12: Manager.List() + DockerClient.ContainerList() + filters |
| CLI-13 | User can run `zone logs` to view harness output with `--follow` | D-13 through D-17: ContainerLogs API + stdcopy demultiplexing |
| CLI-14 | User can run `zone logs --build` to view the last Docker build log | D-15: cache.ReadBuildLog pattern |
| CLI-17 | User can run `zone status` to see container state, harness, uptime, ports, resources | D-19 through D-22: ContainerInspect + plain text formatter |
| CLI-18 | User can use `zone config` to show effective merged config | Already complete in Phase 2 |
| CLI-19 | User can use `zone validate` to check zone.toml without launching | Already complete in Phase 2 |
| CLI-20 | User can use global flags --verbose, --debug, --quiet, --plain on any command | Global flags already in root.go; Phase 8 adds signal context to all Docker commands |
| CLI-21 | User can forward arguments to the harness via `--` separator | D-41 through D-43: --port/-P flag to launch; -- args already supported |
| DX-01 | Structured exit codes 0-6 | D-23 through D-25: errors.Is() chain in main.go |
| DX-02 | All error messages include remediation hints | D-29 through D-32: cmd/errors.go mapError() |
| DX-03 | `--json` flag on status, ls, config, logs for machine-readable output | D-33 through D-36: JSON paths for each command |
| DX-04 | Signal handling: Ctrl+C sends SIGINT to harness, container stays alive | D-26 through D-28: signal.NotifyContext in all Docker-calling commands |
| DX-05 | Context propagation: all Docker SDK calls take context for graceful cancellation | D-26 through D-27: context passed through to Manager methods |
| DX-06 | Harness process exit causes container stop; zone launch returns exit code 0 | D-44 through D-45: natural behavior of `docker exec -it` subprocess |
| DX-07 | `zone stop` cleanup: stop container, remove container, remove network, clear IDs | D-46: already implemented in Manager.Stop() — verification only |
| DX-08 | Command aliases: launch/up, stop/down, ls/list, logs/log, status/st | D-37: aliases already set in Phase 1 stubs — verification only |
| DX-09 | Help text with 2-4 usage examples per command | D-38 through D-40: Long + Example fields in all 15 Cobra commands |
</phase_requirements>

---

## Summary

Phase 8 completes the CLI surface of the `zone` tool. Seven commands are stub (`init`, `ls`, `logs`, `status` are zero-implementation; `launch`, `join`/`exec`/`shell`/`build`/`stop`/`restart`/`destroy`/`clean` are wired but missing signal context). The phase adds four new command implementations, completes the exit code taxonomy, adds `signal.NotifyContext` to all Docker-calling commands, adds `--json` output paths to `ls`/`status`/`logs`, adds remediation hints to all errors, and adds help text with usage examples to all 15 commands.

The codebase is mature — every pattern this phase needs already exists in the repo. `cmd/config.go` provides the definitive JSON output pattern. `cmd/validate.go` provides the exit code 2 pattern. `cmd/launch.go` provides the Manager construction + flag reading pattern. `cmd/stop.go` provides the `mgr.Stop(cmd.Context())` pattern. The Docker SDK's `ContainerList` and `ContainerLogs` APIs are fully available in the pinned `docker/docker v28.5.2` dependency and need only be added to the `DockerClient` interface and Manager.

The test infrastructure uses `sync.Once` binary builds with `runZone()` helper and `setupDir()` fixture. All integration tests run against the compiled binary without Docker. The Phase 8 integration tests follow the same pattern.

**Primary recommendation:** Work plan-by-plan: (1) the four new command bodies (`init`, `ls`, `logs`, `status`) + Docker interface extension, (2) exit code taxonomy + `cmd/errors.go`, (3) signal context + `--port` flag + help text/aliases sweep across all 15 commands.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/spf13/cobra` | v1.10.2 (pinned) | Cobra CLI framework — command routing, flags, aliases | Already used; Phase 1 pinned decision |
| `github.com/docker/docker` | v28.5.2+incompatible | Docker SDK — ContainerList, ContainerLogs, ContainerInspect | Already in go.mod |
| `encoding/json` | stdlib | JSON marshaling for --json output | Pattern from cmd/config.go |
| `os/signal` | stdlib | signal.NotifyContext for graceful Ctrl+C | Standard Go pattern per spec §9 |
| `syscall` | stdlib | SIGTERM constant for NotifyContext | Standard Go pattern |
| `text/tabwriter` | stdlib | Aligned table output for `zone ls` | Zero dependencies, spec table format |
| `github.com/docker/docker/pkg/stdcopy` | v28.5.2 (bundled) | Demultiplex Docker log streams (stdout+stderr mux) | Required when container has no TTY |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/docker/docker/api/types/filters` | v28.5.2 (bundled) | `filters.NewArgs()` for label-based container discovery | `zone ls` ContainerList with label filter |
| `github.com/docker/docker/api/types/container` | v28.5.2 (bundled) | `container.LogsOptions`, `container.ListOptions`, `container.Summary` | logs, ls, status commands |
| `io` | stdlib | `io.Copy` for log streaming | `zone logs --follow` |
| `fmt` | stdlib | Formatted output with `fmt.Fprintf(cmd.ErrOrStderr(), ...)` | Remediation hints to stderr |
| `strconv` | stdlib | Formatting tail count, port numbers | ls/logs flag parsing |
| `time` | stdlib | `time.Since()` for uptime calculation in status/ls | Uptime formatting from container Start time |

**Installation:** No new dependencies needed — all required packages are already in go.mod.

**Version verification (confirmed 2026-03-30):**
- `docker/docker v28.5.2+incompatible` — confirmed in go.mod; ContainerList/ContainerLogs verified in SDK source
- `cobra v1.10.2` — pinned decision from Phase 1

---

## Architecture Patterns

### Recommended Project Structure for Phase 8 additions

```
cmd/
├── errors.go          # NEW: mapError(err) (string, int) — remediation hints + exit code
├── init.go            # IMPLEMENT: harness detection, zone.toml generation
├── ls.go              # IMPLEMENT: Manager.List() + table/JSON output
├── logs.go            # IMPLEMENT: ContainerLogs + --follow/--build/--tail
├── status.go          # IMPLEMENT: ContainerInspect + plain text/JSON output
├── launch.go          # UPDATE: add --port/-P flag + signal.NotifyContext
├── join.go            # UPDATE: add signal.NotifyContext
├── exec.go            # UPDATE: add signal.NotifyContext
├── shell.go           # UPDATE: add signal.NotifyContext
├── build.go           # UPDATE: add signal.NotifyContext
├── stop.go            # UPDATE: add signal.NotifyContext
├── restart.go         # UPDATE: add signal.NotifyContext
├── destroy.go         # UPDATE: add signal.NotifyContext
├── clean.go           # UPDATE: add signal.NotifyContext
├── validate.go        # UPDATE: add Long/Examples help text only
├── config.go          # UPDATE: add Long/Examples help text only
└── root.go            # no change

internal/docker/
├── client_interface.go  # UPDATE: add ContainerList, ContainerLogs to interface
├── manager.go           # UPDATE: add Manager.List(), Manager.Logs(), Manager.Status()
└── errors.go            # UPDATE: add ErrNetworkUnsupported sentinel

main.go                  # UPDATE: extend exit code mapping to full 0-6 taxonomy
```

### Pattern 1: Signal Context (DX-04, DX-05)
**What:** Every command that calls Docker Manager creates a cancellable context tied to OS signals.
**When to use:** All commands that call `docker.NewManager` or Manager methods.
**Example:**
```go
// Source: zone-spec.md §9 (lines 783-792)
RunE: func(cmd *cobra.Command, args []string) error {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()

    // ... config load ...
    mgr, err := docker.NewManager(cfg, c, cwd, version)
    if err != nil {
        return err
    }

    return mgr.Launch(ctx, opts)
},
```

**Critical note for `launch`:** The `docker exec -it` subprocess in `attachInteractive` is spawned via `os/exec.Command`. When the user presses Ctrl+C, the terminal sends SIGINT to the entire process group, which includes the subprocess. This naturally sends SIGINT to the harness process inside the container. The `signal.NotifyContext` cancels the Go context but the `exec.Command.Run()` call will return when the subprocess exits — so context cancellation does not kill the subprocess directly. This matches the spec §3.8 requirement that "container continues running."

### Pattern 2: Manager.List() Implementation
**What:** Query Docker for all containers with `com.zone.managed=true` label.
**When to use:** `zone ls` command.
**Example:**
```go
// Source: Docker SDK v28.5.2 ContainerList
import (
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/filters"
)

// ContainerInfo holds the data returned by Manager.List().
type ContainerInfo struct {
    Name      string
    Harness   string
    Status    string
    State     string   // "running", "exited", etc.
    StartedAt time.Time
    RepoPath  string
    ID        string
}

func (m *Manager) List(ctx context.Context) ([]ContainerInfo, error) {
    f := filters.NewArgs()
    f.Add("label", "com.zone.managed=true")
    containers, err := m.client.ContainerList(ctx, container.ListOptions{
        All:     true, // include stopped containers
        Filters: f,
    })
    if err != nil {
        return nil, fmt.Errorf("list containers: %w", err)
    }
    // map container.Summary fields to ContainerInfo
    result := make([]ContainerInfo, 0, len(containers))
    for _, c := range containers {
        info := ContainerInfo{
            Name:     strings.TrimPrefix(c.Names[0], "/"),
            Harness:  c.Labels["com.zone.harness"],
            Status:   c.Status, // e.g., "Up 2 hours"
            State:    string(c.State), // "running", "exited"
            RepoPath: c.Labels["com.zone.repo-path"],
            ID:       c.ID,
        }
        // Parse StartedAt from c.Created (unix timestamp)
        info.StartedAt = time.Unix(c.Created, 0)
        result = append(result, info)
    }
    return result, nil
}
```

**Important:** The `DockerClient` interface in `client_interface.go` must be extended to add `ContainerList` and `ContainerLogs`. The mock in `manager_test.go` must be updated to add stub implementations for these new methods.

### Pattern 3: ContainerLogs Streaming
**What:** Stream Docker container logs to stdout, with optional follow/tail.
**When to use:** `zone logs` command.
**Example:**
```go
// Source: Docker SDK v28.5.2 client/container_logs.go
import (
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/pkg/stdcopy"
)

func (m *Manager) Logs(ctx context.Context, opts LogsOpts) error {
    containerID, err := m.cache.ContainerID()
    if err != nil || containerID == "" {
        return ErrNoContainer
    }

    logOpts := container.LogsOptions{
        ShowStdout: true,
        ShowStderr: true,
        Follow:     opts.Follow,
        Tail:       opts.Tail, // "all" or "N"
        Timestamps: false,
    }

    rc, err := m.client.ContainerLogs(ctx, containerID, logOpts)
    if err != nil {
        return fmt.Errorf("get logs: %w", err)
    }
    defer rc.Close()

    // Containers created WITHOUT a TTY use multiplexed stdout/stderr streams.
    // StdCopy demultiplexes them. Zone containers do NOT use TTY mode in
    // container.Config (no Tty: true), so stdcopy is correct here.
    _, err = stdcopy.StdCopy(os.Stdout, os.Stderr, rc)
    return err
}
```

**Critical note on TTY vs non-TTY logs:** Docker's `ContainerLogs` returns either a raw stream (if container config has `Tty: true`) or a multiplexed stream with an 8-byte header per frame (if `Tty: false`). Zone containers are created with `Tty: false` (the `container.Config` in `createContainer` does not set `Tty: true`), so `stdcopy.StdCopy` is always correct. Using plain `io.Copy` would produce garbled output with the 8-byte frame headers visible.

### Pattern 4: JSON Output for ls/status
**What:** JSON output follows the `cmd/config.go` pattern — marshal and print to `cmd.OutOrStdout()`.
**When to use:** Any command with `--json` flag.
**Example:**
```go
// Source: cmd/config.go renderJSON pattern
if jsonFlag {
    b, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        return fmt.Errorf("marshal JSON: %w", err)
    }
    fmt.Fprintln(cmd.OutOrStdout(), string(b))
    return nil
}
```

For `zone logs --json`, the output is a JSON array of `{"timestamp": "...", "line": "..."}` objects. This requires reading logs with `Timestamps: true` in `LogsOptions`, then parsing each line.

### Pattern 5: Exit Code Mapping in main.go
**What:** `errors.Is()` chain maps sentinel errors to exit codes before falling back to generic 1.
**When to use:** In `main()` after `cmd.Execute()` returns an error.
**Example:**
```go
// Source: zone-spec.md §3.3, existing main.go
if err := cmd.Execute(); err != nil {
    switch {
    case errors.Is(err, config.ErrNoConfig),
         errors.Is(err, config.ErrSchemaVersion):
        os.Exit(2)
    case errors.Is(err, docker.ErrDockerNotRunning):
        os.Exit(3)
    case errors.Is(err, docker.ErrNetworkUnsupported):
        os.Exit(4)
    case errors.Is(err, cache.ErrLockContention):
        os.Exit(5)
    case errors.Is(err, docker.ErrNoContainer):
        os.Exit(6)
    default:
        // Check UnknownKeysError (config error category)
        var uke *config.UnknownKeysError
        if errors.As(err, &uke) {
            os.Exit(2)
        }
        os.Exit(1)
    }
}
```

**Critical ordering:** `errors.Is()` short-circuits on first match. Specific errors must appear before generic categories. `ErrNoContainer` must be checked before a generic "any error" fallback. The existing `cache.ErrLockContention` check is currently at position 1 — it must be incorporated into the new chain.

### Pattern 6: Remediation Hints (cmd/errors.go)
**What:** `mapError` returns a user-facing message + exit code from a sentinel error.
**When to use:** In every command's `RunE` on error return, before returning to Cobra.
**Example:**
```go
// Source: spec §3.12
func mapError(err error) (msg string, exitCode int) {
    switch {
    case errors.Is(err, docker.ErrDockerNotRunning):
        return "Error: Docker daemon is not running.\n\n  macOS:  Open Docker Desktop, or run `open -a Docker`\n  Linux:  Run `sudo systemctl start docker`\n\nZone requires Docker to create sandboxed workspaces.", 3
    case errors.Is(err, docker.ErrNoContainer):
        return "Error: No running zone container for this repo.\n\n  Run `zone launch` to start one, then `zone join` in another terminal.", 6
    case errors.Is(err, config.ErrNoConfig):
        return "Error: No zone.toml found.\n\n  Run `zone init --harness <name>` to create one.", 2
    // ... etc
    default:
        return "Error: " + err.Error(), 1
    }
}
```

**Integration with Cobra:** Cobra prints `cmd.PrintErrln()` for errors returned from `RunE`. To avoid double-printing, `RunE` should call `fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", msg)` and then return a _sentinel_ error (not a descriptive error) so Cobra's default error printing is suppressed. Alternatively, use `cmd.SilenceErrors()` on the root command. The cleanest approach given the existing codebase: set `rootCmd.SilenceErrors = true` and `rootCmd.SilenceUsage = true` in root.go, then print remediation from `RunE` before returning.

### Pattern 7: ls Table Formatting
**What:** Plain text aligned table using `text/tabwriter`.
**When to use:** `zone ls` default (non-JSON) output.
**Example:**
```go
// Source: text/tabwriter stdlib + spec §13 ls format
import "text/tabwriter"

w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
fmt.Fprintln(w, "NAME\tHARNESS\tSTATUS\tUPTIME\tREPO")
for _, c := range containers {
    uptime := "-"
    if c.State == "running" {
        uptime = formatDuration(time.Since(c.StartedAt))
    }
    fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
        c.Name, c.Harness, formatState(c.State), uptime, c.RepoPath)
}
w.Flush()
```

### Pattern 8: init Command — zone.toml Scaffolding
**What:** `zone init --harness <name>` generates a fully-commented zone.toml and adds `.zone/` to `.gitignore`.
**When to use:** `zone init` command with `--harness` flag.

The existing `QuickstartWriteZoneToml` in `internal/docker/quickstart.go` generates a minimal template. Phase 8 needs a fuller template with all commented-out sections (per D-06 and spec §13). The `zone init` command should use a new `FullInitZoneToml()` function or extend `QuickstartWriteZoneToml` with a `fullTemplate` variant.

**Harness detection** (D-05): Check for these files/dirs in cwd before writing:
- `.claude/` directory → claude-code
- `CLAUDE.md` → claude-code
- `.aider*` (any file starting with `.aider`) → aider
- No standard detection pattern for opencode/gemini-cli/codex-cli exists in codebase

**`--set key=value` parsing** (D-03): Parse dotted paths (e.g., `resources.memory`) and inject into the generated TOML string. Since the template is string-based, the simplest approach is string replacement of commented lines. Alternatively, generate a `MergedConfig` struct, apply overrides via reflection, then serialize — but this is complex. The simpler approach: after generating the template, apply `--set` overrides as literal TOML line inserts.

### Anti-Patterns to Avoid

- **Direct `io.Copy` for non-TTY container logs:** Zone containers don't have `Tty: true`. Raw `io.Copy` will produce garbled output with 8-byte frame headers. Always use `stdcopy.StdCopy`.
- **Calling `os.Exit()` from `RunE`:** Cobra handles exit codes via error propagation. Call `os.Exit()` only in `main()`. The exception is `validate.go` which already uses `os.Exit(2)` — this is an existing pattern but ideally handled via main.go.
- **Not closing ContainerLogs ReadCloser:** The `ContainerLogs` response body must be closed. Always `defer rc.Close()`.
- **Standalone Docker client for ls:** `zone ls` does not require zone.toml but DOES require a Docker daemon. Use `dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())` directly (same as `NewManager` does), or add a `docker.NewDiscoveryClient()` helper. Do NOT construct a full Manager — that requires config.
- **Missing `--json` output on stderr:** Error messages go to stderr even when `--json` is active. Use `fmt.Fprintf(cmd.ErrOrStderr(), ...)` for all errors; never write errors to stdout.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Log stream demultiplexing | Custom byte parser for 8-byte frames | `github.com/docker/docker/pkg/stdcopy.StdCopy` | Docker's frame format has edge cases; SDK handles it correctly |
| Container label filtering | Manual iteration over all containers | `filters.NewArgs()` + `container.ListOptions.Filters` | Docker filters server-side, more efficient |
| Aligned table output | Manual spacing/padding with fmt | `text/tabwriter` | Auto-aligns columns regardless of data width |
| Signal context cancellation | Manual `os/signal.Notify()` + goroutine | `signal.NotifyContext()` (Go 1.16+) | Single function, context-integrated, idiomatic |
| Duration formatting | Custom hours/minutes/seconds logic | Simple helper using `time.Since()` + `time.Duration` methods | Standard approach; tabwriter handles alignment |

**Key insight:** Every "hard" problem in this phase (log demux, container discovery, signal handling) has a first-party solution in the Docker SDK or Go stdlib. Building custom solutions introduces exactly the bugs these libraries solve.

---

## Common Pitfalls

### Pitfall 1: ContainerLogs TTY Mux Confusion
**What goes wrong:** Calling `io.Copy(os.Stdout, rc)` on ContainerLogs output produces garbled output with non-printable bytes at the start of each log line.
**Why it happens:** Docker multiplexes stdout+stderr into a single stream with 8-byte headers when the container has `Tty: false`. Zone containers always have `Tty: false`.
**How to avoid:** Always use `stdcopy.StdCopy(stdout, stderr, rc)` for zone container logs.
**Warning signs:** Log output starts with non-printable characters like `\x01\x00\x00\x00`.

### Pitfall 2: DockerClient Interface Missing Methods
**What goes wrong:** Adding `ContainerList` and `ContainerLogs` calls to Manager but forgetting to add them to `DockerClient` interface in `client_interface.go` and to the mock in `manager_test.go`.
**Why it happens:** Go compiles fine if unused, but tests fail when the mock doesn't implement the interface.
**How to avoid:** Update `client_interface.go` first, then update the `mockClient` in `manager_test.go` to add stub methods, then implement.
**Warning signs:** Compile error "mockClient does not implement DockerClient" in test files.

### Pitfall 3: ls Requires Docker But Not zone.toml
**What goes wrong:** `zone ls` calls `config.LoadMerged()` (which requires zone.toml) before creating the Docker client for discovery.
**Why it happens:** Copying the pattern from `cmd/stop.go` without reading D-12.
**How to avoid:** `zone ls` must create a Docker client directly (or via a lightweight helper) without any config load. Use `dockerclient.NewClientWithOpts(...)` and `Ping()` directly.
**Warning signs:** `zone ls` fails with "no zone.toml found" when run outside a zone project.

### Pitfall 4: Errors.Is Ordering in main.go
**What goes wrong:** Generic `os.Exit(1)` fires before specific exit codes because the `errors.Is` chain has wrong order.
**Why it happens:** `switch` with `errors.Is` falls through to `default` if specific errors appear after the default case, or if wrapped errors don't match due to wrong wrapping.
**How to avoid:** Specific sentinels first, generic fallback last. Test with `errors.Wrap` chains — `errors.Is` traverses `%w` chains correctly.
**Warning signs:** `zone launch` exits 1 instead of 3 when Docker is not running.

### Pitfall 5: Signal Context Cancellation vs Container Lifetime
**What goes wrong:** Adding `signal.NotifyContext` causes `zone launch` to kill the container when Ctrl+C is pressed.
**Why it happens:** The context is passed to `mgr.Launch(ctx, opts)`, and if the context cancel propagates into `attachFn`, the `exec.Command` for `docker exec -it` is killed.
**How to avoid:** The `attachInteractive` function uses `exec.Command("docker", ...)` — this is separate from the Go context. The `os/exec.Command.Run()` does not cancel based on context unless you use `exec.CommandContext`. Since `attachInteractive` uses `exec.Command` (not `exec.CommandContext`), Ctrl+C goes to the subprocess naturally through the terminal's process group, not through the context. The context cancel only affects Docker SDK calls that happen BEFORE the attach. This is correct behavior.
**Warning signs:** Container is killed instead of staying alive when user presses Ctrl+C.

### Pitfall 6: --set Dotted Path Parsing for zone init
**What goes wrong:** Implementing dotted-path TOML override as a fully generic reflection-based system.
**Why it happens:** Over-engineering a feature that only needs to handle a limited set of paths.
**How to avoid:** For Phase 8, implement as targeted string replacement in the template. The generated template has commented-out lines like `# memory = "4g"` — `--set resources.memory=8g` uncomments and replaces the value. A map of `section.key → template_line_pattern` is sufficient.
**Warning signs:** Spending more than 30 minutes on the --set implementation.

---

## Code Examples

### Adding Methods to DockerClient Interface
```go
// Source: internal/docker/client_interface.go — extend with these two methods
ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error)
ContainerLogs(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error)
```

### Uptime Formatting Helper
```go
// Source: time stdlib — simple helper for ls/status
func formatUptime(t time.Time) string {
    d := time.Since(t)
    h := int(d.Hours())
    m := int(d.Minutes()) % 60
    if h > 0 {
        return fmt.Sprintf("%dh %dm", h, m)
    }
    if m > 0 {
        return fmt.Sprintf("%dm", m)
    }
    return fmt.Sprintf("%ds", int(d.Seconds()))
}
```

### signal.NotifyContext import pattern
```go
// Source: spec §9 — imports needed for signal context
import (
    "context"
    "os"
    "os/signal"
    "syscall"
)

ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()
```

### Cobra Long + Examples fields
```go
// Source: spec §3.11 — Cobra uses "Example" field (not "Examples")
var launchCmd = &cobra.Command{
    Use:     "launch",
    Aliases: []string{"up"},
    Short:   "Build (if needed) and attach to the container",
    Long:    "Build (if needed) and attach to a sandboxed Docker container for this repo.",
    Example: `  # Launch interactively (default)
  zone launch

  # Zero-config quickstart
  zone launch --harness claude-code

  # Launch and pass a prompt to the harness
  zone launch -p "refactor the auth module"

  # Fire and forget (background with task)
  zone launch --headless -p "fix the failing tests"`,
    RunE: ...,
}
```

**Important:** The Cobra field is `Example` (singular), not `Examples`. This is a common mistake.

### Harness Detection for zone init
```go
// Source: spec §13 init wizard detection hints
func detectHarnessHints(cwd string) []string {
    var hints []string
    if _, err := os.Stat(filepath.Join(cwd, ".claude")); err == nil {
        hints = append(hints, ".claude/ directory")
    }
    if _, err := os.Stat(filepath.Join(cwd, "CLAUDE.md")); err == nil {
        hints = append(hints, "CLAUDE.md")
    }
    entries, _ := os.ReadDir(cwd)
    for _, e := range entries {
        if strings.HasPrefix(e.Name(), ".aider") {
            hints = append(hints, ".aider* file")
            break
        }
    }
    return hints
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `types.ContainerLogsOptions` | `container.LogsOptions` | Docker SDK v25+ | Type moved from top-level `types` package to `container` sub-package |
| `types.ContainerListOptions` | `container.ListOptions` | Docker SDK v25+ | Same move |
| `signal.Notify()` + goroutine | `signal.NotifyContext()` | Go 1.16 | Simpler, context-integrated |

**Deprecated/outdated:**
- `github.com/docker/docker/api/types.ContainerLogsOptions`: Use `github.com/docker/docker/api/types/container.LogsOptions` — the top-level types package consolidation happened in Docker SDK v25+. The project is on v28.5.2.
- `github.com/docker/docker/api/types.ContainerListOptions`: Use `github.com/docker/docker/api/types/container.ListOptions`.

---

## Open Questions

1. **`zone logs --json` line parsing with timestamps**
   - What we know: `container.LogsOptions{Timestamps: true}` prefixes each line with RFC3339 timestamp
   - What's unclear: The timestamp + log line separator character (space) and whether `stdcopy.StdCopy` preserves line boundaries correctly for JSON line-by-line parsing
   - Recommendation: For Phase 8, implement `--json` on logs by buffering all lines via `bufio.Scanner` on the demultiplexed stream. Each line becomes `{"timestamp": "...", "line": "..."}` in the JSON array.

2. **`--set` scope for zone init**
   - What we know: D-03 specifies dotted-path overrides applied to generated zone.toml
   - What's unclear: Whether `--set` needs to handle sections not in the template (e.g., `--set packages.apt=git,curl`)
   - Recommendation: Support only scalar string/int overrides for fields that appear in the template. Skip list overrides for Phase 8 (error with "list overrides not supported by --set, edit zone.toml directly").

3. **Manager.List vs standalone Docker client for ls**
   - What we know: D-12 says ls is a global discovery command not requiring zone.toml; D-07 says Manager.List(ctx)
   - What's unclear: If Manager.List() is a method on Manager, it requires config+cache construction, contradicting D-12
   - Recommendation: Add a package-level function `docker.ListContainers(ctx context.Context) ([]ContainerInfo, error)` that creates its own Docker client directly. This is consistent with D-12 and avoids the config dependency. The cmd/ls.go uses this function rather than constructing a Manager.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | `go test` + `github.com/stretchr/testify v1.11.1` |
| Config file | none (standard go test) |
| Quick run command | `go test ./tests/... -run TestInit -timeout 30s` |
| Full suite command | `go test ./... -timeout 120s` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CLI-01 | `zone init` without --harness errors with terminal message | integration | `go test ./tests/... -run TestInitNoHarnessError -timeout 30s` | ❌ Wave 0 |
| CLI-02 | `zone init --harness claude-code` creates zone.toml | integration | `go test ./tests/... -run TestInitHarnessFlag -timeout 30s` | ❌ Wave 0 |
| CLI-12 | `zone ls --help` works, no "not implemented" | integration | `go test ./tests/... -run TestLsNotStub -timeout 30s` | ❌ Wave 0 |
| CLI-13 | `zone logs --help` works, no "not implemented" | integration | `go test ./tests/... -run TestLogsNotStub -timeout 30s` | ❌ Wave 0 |
| CLI-14 | `zone logs --build` without .zone/ errors with exit 6 | integration | `go test ./tests/... -run TestLogsBuildNoCache -timeout 30s` | ❌ Wave 0 |
| CLI-17 | `zone status --help` works, no "not implemented" | integration | `go test ./tests/... -run TestStatusNotStub -timeout 30s` | ❌ Wave 0 |
| DX-01 | `zone validate` on bad config exits 2 | integration | `go test ./tests/... -run TestExitCodes -timeout 30s` | ❌ Wave 0 (extends existing) |
| DX-01 | Docker not running exits 3 | integration | manual-only — requires no Docker daemon | manual-only |
| DX-01 | No container exits 6 (join without launch) | integration | `go test ./tests/... -run TestExitCode6 -timeout 30s` | ❌ Wave 0 |
| DX-02 | Error messages include remediation hints | integration | `go test ./tests/... -run TestRemediationHints -timeout 30s` | ❌ Wave 0 |
| DX-03 | `zone ls --json` produces parseable JSON | integration (mock Docker) | `go test ./tests/... -run TestLsJSON -timeout 30s` | ❌ Wave 0 |
| DX-03 | `zone status --json` produces parseable JSON | integration (mock Docker) | `go test ./tests/... -run TestStatusJSON -timeout 30s` | ❌ Wave 0 |
| DX-08 | All aliases work identically to primary commands | integration | `go test ./tests/... -run TestAliases -timeout 30s` | ❌ Wave 0 |
| DX-09 | All 15 commands have --help with examples | integration | `go test ./tests/... -run TestHelpExamples -timeout 30s` | ❌ Wave 0 |
| DX-04 | Ctrl+C leaves container alive | manual-only | n/a — requires live container + signal delivery | manual-only |
| DX-05 | Context propagation | unit | `go test ./internal/docker/... -run TestContextPropagation -timeout 30s` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./tests/... -timeout 60s`
- **Per wave merge:** `go test ./... -timeout 120s`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `tests/cli_dx_test.go` — covers CLI-01, CLI-02, CLI-12, CLI-13, CLI-14, CLI-17, DX-01, DX-02, DX-03, DX-08, DX-09 using existing `getZoneBinary()` + `runZone()` + `setupDir()` helpers from `tests/config_cmd_test.go`
- [ ] Mock Docker client stubs for `ContainerList` and `ContainerLogs` in `internal/docker/manager_test.go`

---

## Sources

### Primary (HIGH confidence)
- Docker SDK v28.5.2 source — `/home/claude/go/pkg/mod/github.com/docker/docker@v28.5.2+incompatible/client/container_list.go`, `container_logs.go` — ContainerList/ContainerLogs API signatures verified
- Docker SDK v28.5.2 types — `api/types/container/options.go` — LogsOptions, ListOptions struct fields verified
- Docker SDK v28.5.2 pkg/stdcopy — demultiplexing API verified
- Docker SDK v28.5.2 api/types/filters — filters.NewArgs(), Add() API verified
- Go stdlib docs — `os/signal.NotifyContext` (Go 1.16+), `text/tabwriter` standard library
- `zone-spec.md` §3 (commands), §9 (signal handling), §13 (TUI layouts/table format) — authoritative spec
- Existing codebase (all files read above) — patterns from cmd/config.go, cmd/validate.go, cmd/launch.go, internal/docker/client_interface.go, main.go

### Secondary (MEDIUM confidence)
- Docker SDK `api/types/container/container.go` `Summary` struct — field layout for ls ContainerInfo mapping

### Tertiary (LOW confidence)
- None — all claims verified against source code or spec

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — verified in go.mod and SDK source
- Architecture: HIGH — patterns verified against existing codebase
- Pitfalls: HIGH — TTY mux confirmed in Docker SDK source comments; others from spec and code inspection

**Research date:** 2026-03-30
**Valid until:** 2026-06-01 (Docker SDK v28.x is stable; Go stdlib is stable)
