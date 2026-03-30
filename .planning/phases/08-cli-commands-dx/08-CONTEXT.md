# Phase 8: CLI Commands & DX - Context

**Gathered:** 2026-03-30
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire the remaining 4 CLI commands (`init`, `ls`, `logs`, `status`), implement structured exit codes (0-6), add `signal.NotifyContext` to all Docker-calling commands, add `--json` support to `ls`/`status`/`logs`, add remediation hints to all error messages, add help text with usage examples to all 15 commands, add command aliases, and add the `--port/-P` ad-hoc flag deferred from Phase 7. TUI (BubbleTea) is NOT in scope — all output is plain text. The TUI layer comes in Phase 9 and will replace the plain-text output paths with BubbleTea views.

</domain>

<decisions>
## Implementation Decisions

### init command (CLI-01, CLI-02)
- **D-01:** `zone init --harness <name>` scaffolds zone.toml non-interactively: write commented template file, add `.zone/` to `.gitignore`, print confirmation message
- **D-02:** `zone init` without `--harness` in Phase 8: error with "Interactive mode requires a terminal. Use `--harness <name>` for non-interactive init." — TUI wizard deferred to Phase 9
- **D-03:** `--set key=value` flag for dotted-path config overrides (e.g., `--set resources.memory=8g`) — applies overrides to the generated zone.toml
- **D-04:** Detect existing zone.toml: error with "zone.toml already exists. Use `zone config` to modify it."
- **D-05:** Detect harness hints: check for `.claude/` directory, `CLAUDE.md`, `.aider*`, etc. in the repo — print "Detected: <harness> indicators" hint in output
- **D-06:** Generated zone.toml includes commented-out sections showing all available fields per spec §13

### ls command (CLI-12)
- **D-07:** `Manager.List(ctx)` method queries Docker API with label filter `com.zone.managed=true`, returns `[]ContainerInfo` structs
- **D-08:** `cmd/ls.go` formats as plain text table: NAME, HARNESS, STATUS, UPTIME, REPO columns
- **D-09:** `--json` flag outputs JSON array to stdout (bypass table formatting)
- **D-10:** `--running` flag filters to running containers only
- **D-11:** `--quiet/-q` flag prints only container names (one per line) for scripting
- **D-12:** ls does NOT require zone.toml — it's a global discovery command using Docker client directly

### logs command (CLI-13, CLI-14)
- **D-13:** Plain text output via Docker SDK `ContainerLogs(ctx, id, types.ContainerLogsOptions{...})`
- **D-14:** `--follow/-f` flag: set `Follow: true` in options, stream to stdout via `io.Copy`
- **D-15:** `--build` flag: read `.zone/logs/last_build.log` from cache and print to stdout (no Docker needed)
- **D-16:** `--tail N` flag: set `Tail: "N"` in options (Docker SDK handles this)
- **D-17:** When stdout is piped, output plain text (no TUI chrome) — this is the Phase 8 default behavior
- **D-18:** Requires running container (unless `--build`); error with exit code 6 if no container

### status command (CLI-17)
- **D-19:** One-shot container inspection via Docker SDK `ContainerInspect` — print formatted plain text
- **D-20:** Display: Repo, Harness, Container name, Status (with uptime), Image ID, Network mode, Port mappings, Resource limits, Mounts
- **D-21:** `--json` flag: output raw container inspect JSON to stdout
- **D-22:** Requires running or stopped container; error with exit code 6 if no container

### Structured exit codes (DX-01)
- **D-23:** Extend main.go with full sentinel error → exit code mapping per spec §3.3:
  - 0: success
  - 1: generic/unknown
  - 2: config error (`config.ErrNoConfig`, `config.UnknownKeysError`, schema errors)
  - 3: Docker error (`docker.ErrDockerNotRunning`, build failures, image not found)
  - 4: Network error (`docker.ErrNetworkUnsupported`, firewall failures)
  - 5: Cache error (`cache.ErrLockContention`, corrupted cache)
  - 6: No container (`docker.ErrNoContainer` for join/exec/shell/logs/status)
- **D-24:** Use `errors.Is()` chain in main.go — order matters: check specific errors before generic fallback
- **D-25:** Add sentinel errors to `internal/docker/errors.go` if missing: `ErrNoContainer`, `ErrNetworkUnsupported`

### Signal handling (DX-04, DX-05)
- **D-26:** Add `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)` to all commands that call Docker Manager: launch, join, exec, shell, build, stop, restart, destroy, clean, logs, status
- **D-27:** Pass the context through to Manager methods — all Docker SDK calls already take context from Phase 6
- **D-28:** Ctrl+C during `zone launch` sends SIGINT to harness process inside container; container stays alive per spec §3.8 — this is handled by the `docker exec -it` subprocess naturally

### Remediation hints (DX-02)
- **D-29:** Create `cmd/errors.go` with `mapError(err error) (string, int)` function that returns remediation message + exit code
- **D-30:** Remediation hints added at cmd layer — internal packages return clean sentinel errors, cmd layer wraps with user-facing messages
- **D-31:** Use fmt.Fprintf to stderr for remediation hints (separate from command output to stdout)
- **D-32:** Follow spec §3.12 format: "Error: <message>\n\n  <platform-specific hints>\n\n<context line>"

### JSON output (DX-03)
- **D-33:** `--json` on `zone status`: output container inspection as JSON object
- **D-34:** `--json` on `zone ls`: output as JSON array of container info objects
- **D-35:** `--json` on `zone logs`: output log lines as JSON array with timestamp fields
- **D-36:** `zone config --json` already implemented in Phase 2 — no changes needed

### Command aliases (DX-08)
- **D-37:** Aliases already defined in Phase 1 stubs: launch/up, stop/down, ls/list, logs/log, status/st — verify all are present and working

### Help text with examples (DX-09)
- **D-38:** All 15 commands get `Long` field with 2-4 usage examples per spec
- **D-39:** Example format: indented, showing common use cases for each command
- **D-40:** `zone launch` examples: basic, headless, zero-config, with prompt, with harness args

### Ad-hoc port flag (CLI-21 partial, deferred from Phase 7)
- **D-41:** Add `--port/-P` flag to `zone launch` command — repeatable string flag
- **D-42:** Parse same `host:container` format as config ports, merge with config ports
- **D-43:** Pass merged ports list to Manager.Launch opts

### Harness exit behavior (DX-06)
- **D-44:** When harness process exits (user types /exit): the `docker exec -it` subprocess returns, launch command returns exit code 0 per spec
- **D-45:** Container stops after harness exit — this is the entrypoint `exec` behavior from Phase 4

### Stop cleanup (DX-07)
- **D-46:** Already implemented in Phase 6 Manager.Stop() — verify: stop container, remove container, remove network, clear IDs from cache

### Claude's Discretion
- ContainerInfo struct field layout
- Exact table column widths/formatting in ls
- Harness hint detection file patterns
- Help text example wording
- Remediation hint exact wording beyond spec examples
- Error message ordering in main.go

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Command specifications
- `zone-spec.md` §3.1-3.2 (lines 52-85) — Full command table with flags, aliases, behavior
- `zone-spec.md` §3.3 (lines 87-97) — Exit codes 0-6
- `zone-spec.md` §3.4 (lines 99-108) — Argument forwarding convention
- `zone-spec.md` §3.5 (lines 112-117) — TTY detection
- `zone-spec.md` §3.12 (lines 218-243) — Actionable error messages with remediation hints
- `zone-spec.md` §3.13 (lines 244-248) — `--json` scriptability convention

### TUI layouts (reference only — Phase 9 implements BubbleTea)
- `zone-spec.md` §13 (lines 1310-1411) — init wizard, status view, log viewer, ls table format

### Signal handling
- `zone-spec.md` §9 (lines 778-818) — Context propagation, `signal.NotifyContext`, build streaming

### Config for init scaffolding
- `zone-spec.md` §4.2 (lines 265-373) — Full zone.toml field reference for commented template

### Existing wired commands (Phase 6)
- `cmd/launch.go` — Launch with --harness, --headless, -p, --rebuild, --no-cache
- `cmd/join.go`, `cmd/exec.go`, `cmd/shell.go` — Interactive attach
- `cmd/stop.go`, `cmd/restart.go`, `cmd/destroy.go` — Lifecycle cleanup
- `cmd/build.go` — Standalone build
- `cmd/clean.go` — Cache cleanup
- `cmd/config.go` — Full config display with --json (Phase 2)
- `cmd/validate.go` — Config validation with exit code 2 (Phase 2)

### Docker Manager (Phase 6-7 extension points)
- `internal/docker/manager.go` — Manager struct, createContainer(), buildMounts(), Stop()
- `internal/docker/launch.go` — Launch() state machine
- `internal/docker/errors.go` — Sentinel errors

### main.go
- `main.go` — Current exit code mapping (1, 2, 5 only)

### Requirements
- `.planning/REQUIREMENTS.md` — CLI-01, CLI-02, CLI-12-14, CLI-17-21, DX-01-09

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/root.go` — Global flags already defined: --verbose, --debug, --quiet, --plain (CLI-20 partially done)
- `cmd/launch.go` — Complete launch wiring with flags pattern to follow
- `cmd/config.go` — Full --json implementation pattern to follow for ls/status/logs
- `cmd/validate.go` — Exit code 2 pattern to follow for other error categories
- `internal/docker/naming.go` — ContainerLabels() returns labels for `zone ls` discovery
- `internal/docker/manager.go` — Manager with Docker client, already has all lifecycle methods
- `internal/cache/cache.go` — Cache.ReadID("last_build.log") accessible for `zone logs --build`
- `internal/docker/quickstart.go` — QuickstartWriteZoneToml() already generates minimal zone.toml (used by launch --harness zero-config)
- `cmd/ls.go`, `cmd/logs.go`, `cmd/status.go`, `cmd/init.go` — Stub commands with aliases already set

### Established Patterns
- Cobra command structure: var xxxCmd + init() for flags
- Manager construction: config.LoadMerged → cache.New → docker.NewManager
- Error wrapping: `fmt.Errorf("context: %w", err)` for sentinel error chain
- JSON output: `json.MarshalIndent` + `fmt.Fprintln(cmd.OutOrStdout(), ...)` pattern from config.go

### Integration Points
- `cmd/init.go` → calls QuickstartWriteZoneToml (extend) + harness detection
- `cmd/ls.go` → calls new Manager.List() or standalone Docker client query
- `cmd/logs.go` → calls Docker SDK ContainerLogs or reads cache file
- `cmd/status.go` → calls Docker SDK ContainerInspect
- `main.go` → extend exit code mapping with errors.Is chain
- All Docker-calling commands → add signal.NotifyContext

</code_context>

<specifics>
## Specific Ideas

None beyond spec — all decisions auto-selected from recommended defaults.

</specifics>

<deferred>
## Deferred Ideas

- TUI BubbleTea views (init wizard, status live view, logs viewer, build progress) → Phase 9
- `--edit` flag on `zone config` (opens $EDITOR) → Phase 9 or backlog
- `--schema` flag on `zone config` → backlog
- `--from-devcontainer` migration on `zone init` → v2 backlog
- Network-related exit code 4 detailed handling → Phase 10

</deferred>

---

*Phase: 08-cli-commands-dx*
*Context gathered: 2026-03-30*
