---
phase: 06-docker-lifecycle-core
plan: 04
subsystem: docker
tags: [docker, cobra, cli, launch, stop, destroy, join, exec, shell, build, restart, clean, integration-tests]

# Dependency graph
requires:
  - phase: 06-docker-lifecycle-core
    plan: 01
    provides: Manager struct, DockerClient interface, attachInteractive, NewManager
  - phase: 06-docker-lifecycle-core
    plan: 02
    provides: Manager.Launch, LaunchOpts, handleZeroConfig/QuickstartWriteZoneToml
  - phase: 06-docker-lifecycle-core
    plan: 03
    provides: Manager.Stop, Manager.Destroy, Manager.RemoveImage
  - phase: 03-cache-state
    provides: Cache.New, Cache.ContainerID, Cache.EnsureDir, EnsureGitignore
  - phase: 02-config-foundation
    provides: config.LoadMerged, config.ErrNoConfig
provides:
  - All 8 Cobra lifecycle commands wired to Manager methods (launch, join, exec, shell, build, stop, restart, destroy)
  - cmd/clean extended with --image flag calling Manager.RemoveImage
  - cmd/root.go package-level var version for version propagation into Manager
  - Manager.Join(), Manager.Exec(), Manager.Shell() public methods
  - QuickstartWriteZoneToml standalone function for zero-config without live Docker
  - TestCommandsNotStub integration test: all 8 commands verified non-stub
affects: [07-volume-ssh, 08-e2e-integration]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "All commands follow: os.Getwd() → config.LoadMerged() → cache.New() → docker.NewManager() → method call"
    - "package-level var version in cmd/root.go — set by SetVersion, threaded into NewManager for template rendering"
    - "QuickstartWriteZoneToml is standalone (no Docker required) — zero-config path doesn't need Ping"
    - "cobra.ArbitraryArgs on exec command — args slice passed verbatim to Manager.Exec"
    - "Integration tests reuse getZoneBinary (sync.Once) from config_cmd_test.go — single binary build per test run"

key-files:
  created:
    - tests/lifecycle_cmd_test.go
  modified:
    - cmd/launch.go
    - cmd/build.go
    - cmd/stop.go
    - cmd/restart.go
    - cmd/destroy.go
    - cmd/join.go
    - cmd/exec.go
    - cmd/shell.go
    - cmd/clean.go
    - cmd/root.go
    - internal/docker/manager.go
    - internal/docker/quickstart.go

key-decisions:
  - "QuickstartWriteZoneToml extracted as standalone function — zero-config path (zone launch --harness) must not fail because Docker is not running yet"
  - "Manager.Join validates container state (running) before attach; Manager.Exec/Shell do not — exec is explicit, shell failure is acceptable"
  - "HandleZeroConfig exported on Manager (delegates to standalone) — preserves method form for any code that has a Manager instance"
  - "var version in cmd/root.go initialized to 'dev' — safe default when SetVersion not called (tests)"
  - "destroy confirmation reads from os.Stdin directly — cmd.InOrStdin() would require cobra wiring not needed for basic terminal usage"

requirements-completed: [CLI-06, CLI-07, CLI-08, CLI-09, CLI-11]

# Metrics
duration: 4min
completed: 2026-03-29
---

# Phase 6 Plan 04: Cobra Command Wiring Summary

**All 9 Cobra commands wired to Manager methods — 8 lifecycle commands no longer stub, clean extended with --image, integration test TestCommandsNotStub confirms zero "not implemented" responses**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-29T22:57:08Z
- **Completed:** 2026-03-29T23:01:10Z
- **Tasks:** 2
- **Files modified:** 12

## Accomplishments
- `cmd/launch.go`: full implementation with --harness (zero-config), --headless, -p/--prompt, --rebuild, --no-cache, -y/--yes, --root flags; zero-config path writes zone.toml without requiring live Docker
- `cmd/build.go`, `cmd/stop.go`, `cmd/restart.go`, `cmd/destroy.go`: wired to Manager with appropriate flags (--no-cache, --timeout, --rebuild, -y/--yes)
- `cmd/join.go`, `cmd/exec.go`, `cmd/shell.go`: wired to new Manager.Join/Exec/Shell methods with --root on exec
- `cmd/clean.go`: extended with --image flag that calls Manager.RemoveImage after cache clean
- `cmd/root.go`: package-level `var version = "dev"` + updated SetVersion to set it (enables version propagation into Manager.NewManager)
- `Manager.Join/Exec/Shell`: public methods added to internal/docker/manager.go — Join validates container is running, Exec/Shell read container ID from cache
- `QuickstartWriteZoneToml`: standalone function extracted from handleZeroConfig — allows zero-config without live Docker daemon
- `TestCommandsNotStub`: integration test verifies all 8 lifecycle commands return real errors (not "not implemented") in a no-config temp dir

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire launch, build, stop, restart, destroy, clean** - `1896d88` (feat)
2. **Task 2: Wire join, exec, shell; add Manager.Join/Exec/Shell; integration tests** - `e0caf8d` (feat)

**Plan metadata:** TBD (docs: complete plan)

## Files Created/Modified
- `cmd/launch.go` - Full launch implementation: flags, zero-config path, Manager.Launch
- `cmd/build.go` - Wired to Manager.Build with --no-cache
- `cmd/stop.go` - Wired to Manager.Stop with --timeout
- `cmd/restart.go` - Wired to Manager.Stop + Manager.Launch with --rebuild
- `cmd/destroy.go` - Wired to Manager.Destroy with -y/--yes confirmation
- `cmd/join.go` - Wired to Manager.Join
- `cmd/exec.go` - Wired to Manager.Exec with --root, ArbitraryArgs
- `cmd/shell.go` - Wired to Manager.Shell
- `cmd/clean.go` - Extended with --image flag + Manager.RemoveImage
- `cmd/root.go` - Added package-level var version, updated SetVersion
- `internal/docker/manager.go` - Added Join(), Exec(), Shell() methods
- `internal/docker/quickstart.go` - Exported HandleZeroConfig, added QuickstartWriteZoneToml standalone
- `tests/lifecycle_cmd_test.go` - Created: TestCommandsNotStub + TestCleanImageFlag

## Decisions Made
- QuickstartWriteZoneToml extracted as standalone function — zero-config path (zone launch --harness) must not fail because Docker is not running; writes zone.toml before connecting to Docker
- Manager.Join validates container state (running) before attaching; Manager.Exec and Manager.Shell only check container ID exists — exec failure is expected when container not running
- HandleZeroConfig exported on Manager and delegates to standalone — preserves method form for any code that has a Manager instance
- var version initialized to "dev" in cmd/root.go — safe default when SetVersion not called (test environments)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Exported handleZeroConfig and added QuickstartWriteZoneToml**
- **Found during:** Task 1 (launch.go implementation)
- **Issue:** Plan specified calling `handleZeroConfig` from cmd/launch.go but it was unexported; also, the zero-config path ran before Docker was available so calling NewManager (which pings Docker) would fail
- **Fix:** Exported `HandleZeroConfig` method on Manager and extracted `QuickstartWriteZoneToml` standalone function; cmd/launch.go calls the standalone function directly
- **Files modified:** internal/docker/quickstart.go, cmd/launch.go
- **Verification:** `go build ./...` passes; `zone launch --help` shows all flags
- **Committed in:** 1896d88 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (missing critical — correctness fix)
**Impact on plan:** Required for correct operation; zero-config path would fail with Docker not running error instead of writing zone.toml.

## Issues Encountered
None beyond the deviation above.

## User Setup Required
None.

## Next Phase Readiness
- All 8 lifecycle commands are now wired and functional — Phase 7 can add volume/SSH mounting without touching command stubs
- Manager.Join/Exec/Shell provide the attach interface needed for any TTY-based enhancements
- version variable is now threaded all the way from ldflags through SetVersion to Manager.NewManager

---
*Phase: 06-docker-lifecycle-core*
*Completed: 2026-03-29*

## Self-Check: PASSED

- cmd/launch.go: FOUND
- cmd/build.go: FOUND
- cmd/stop.go: FOUND
- cmd/restart.go: FOUND
- cmd/destroy.go: FOUND
- cmd/join.go: FOUND
- cmd/exec.go: FOUND
- cmd/shell.go: FOUND
- cmd/clean.go: FOUND
- cmd/root.go: FOUND
- internal/docker/manager.go: FOUND
- internal/docker/quickstart.go: FOUND
- tests/lifecycle_cmd_test.go: FOUND
- .planning/phases/06-docker-lifecycle-core/06-04-SUMMARY.md: FOUND
- Commit 1896d88: FOUND
- Commit e0caf8d: FOUND
