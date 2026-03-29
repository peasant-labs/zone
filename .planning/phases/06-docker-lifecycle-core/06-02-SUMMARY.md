---
phase: 06-docker-lifecycle-core
plan: 02
subsystem: docker
tags: [docker, container, state-machine, launch, headless, zero-config, tdd, unit-tests]

# Dependency graph
requires:
  - phase: 06-docker-lifecycle-core
    plan: 01
    provides: Manager struct, DockerClient interface, buildImage, createContainer, attachInteractive, mock client infrastructure
  - phase: 03-cache-state
    provides: Cache.ContainerID/SetContainerID/ConfigHash/SetConfigHash, ComputeHash, Lock.Acquire/Release
  - phase: 05-harness-plugin-system
    provides: harness.Get(), Harness.EntrypointCommand(), Harness.PromptFlag()
provides:
  - Manager.Launch() with full 6-state state machine (running/paused/exited/dead/stale/fresh)
  - LaunchOpts struct (Headless/Prompt/Rebuild/NoCache/HarnessArgs)
  - inspectContainerState() with errdefs.IsNotFound stale detection
  - handleRunning() with config hash comparison and warning
  - checkConfigHash() comparing current vs cached hash
  - cleanStaleCache() for externally-deleted container cleanup
  - buildIfNeeded() hash-aware conditional build
  - createAndStart() container creation + start + cache persistence
  - harnessCmd() entrypoint command builder
  - generateMinimalZoneToml() zero-config template generator
  - handleZeroConfig() zone.toml writer + gitignore updater
  - attachFn hook on Manager for unit test injection
affects: [06-03-stop-destroy, 06-04-cobra-wiring]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "attachFn func field on Manager: default=attachInteractive, overridden in tests to no-op"
    - "TDD RED/GREEN: failing tests committed before implementation"
    - "errdefs.IsNotFound() translates 'not found' Docker errors to (nil, nil) for stale container detection"
    - "Lock released before TTY attach: allows zone join from another terminal"
    - "buildIfNeeded() uses two-condition skip: hash match AND image still exists"
    - "fmt.Fprintln(os.Stderr) for warnings; fmt.Println for headless container ID to stdout"

key-files:
  created:
    - internal/docker/launch.go
    - internal/docker/quickstart.go
  modified:
    - internal/docker/manager.go (added attachFn field, updated NewManager + newManagerWithClient)
    - internal/docker/manager_test.go (added 10 new tests + mock call tracking)

key-decisions:
  - "attachFn field on Manager enables test injection without build tags or interface wrapping"
  - "Lock NOT deferred — explicitly released before attachInteractive so zone join can connect"
  - "TestLaunchStateMachine_Fresh: verify removeCalled=false (not inspect call tracking) to avoid brittle mock state assertions"
  - "handleRunning receives full InspectResponse (not just containerID) to avoid second ContainerInspect call"

requirements-completed: [DOC-09, DOC-10, CLI-03, CLI-04, CLI-05]

# Metrics
duration: 4min
completed: 2026-03-29
---

# Phase 6 Plan 02: Launch State Machine Summary

**Launch state machine with 6-state container handling (running/paused/exited/dead/stale/fresh), config hash drift detection with user warning, headless fire-and-forget mode printing container ID to stdout, and zero-config quickstart generating zone.toml from --harness flag — 10 TDD unit tests all passing**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-29T22:47:45Z
- **Completed:** 2026-03-29T22:51:15Z
- **Tasks:** 1 (TDD: RED commit + GREEN commit)
- **Files modified:** 4

## Accomplishments
- Full `Manager.Launch()` state machine handling all 6 container states idempotently
- Config hash comparison on running container — warns on drift, does NOT auto-restart
- Lock acquired at start, explicitly released before TTY attach so `zone join` can connect
- Headless mode: prints container ID to stdout, returns without TTY attach (exit code 0)
- `buildIfNeeded()` skips rebuild when hash matches AND image still exists in Docker
- `generateMinimalZoneToml()` produces well-structured zone.toml with commented options section
- `attachFn` field on Manager enables test isolation without exec-time Docker dependency
- 23 tests total (13 from Plan 01 + 10 new state machine tests), all pass

## Task Commits

Each step committed atomically:

1. **TDD RED: failing tests for all behaviors** - `56c1521` (test)
2. **TDD GREEN: full implementation** - `39125a6` (feat)

## Files Created/Modified
- `internal/docker/launch.go` - LaunchOpts, Launch(), inspectContainerState(), handleRunning(), checkConfigHash(), cleanStaleCache(), buildIfNeeded(), createAndStart(), harnessCmd()
- `internal/docker/quickstart.go` - generateMinimalZoneToml(), handleZeroConfig()
- `internal/docker/manager.go` - Added attachFn field; updated NewManager + newManagerWithClient to initialize it
- `internal/docker/manager_test.go` - 10 new tests + mock call tracking fields (unpauseCalled, removeCalled, stopCalled, startCalled)

## Decisions Made
- `attachFn` field on Manager is the simplest test injection approach — no build tags, no interface wrapping, no global var
- Lock release is explicit (NOT deferred) before `attachInteractive` so concurrent `zone join` sessions can acquire the lock
- `handleRunning` receives full `*container.InspectResponse` to avoid a second `ContainerInspect` call
- `TestLaunchStateMachine_Fresh` asserts `removeCalled=false` rather than trying to assert inspect wasn't called (mock state is set regardless)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Mock `containerInspectResp` missing `ID` field**
- **Found during:** TDD GREEN phase (test failed)
- **Issue:** `makeLaunchMock` set `ContainerJSONBase.State` but not `ContainerJSONBase.ID`; `handleRunning` used `info.ID` which was empty string
- **Fix:** Added `ID: "container-abc"` to `ContainerJSONBase` in `makeLaunchMock`
- **Files modified:** internal/docker/manager_test.go
- **Commit:** 39125a6

**2. [Rule 1 - Bug] `TestLaunchStateMachine_Fresh` assertion was incorrect**
- **Found during:** TDD GREEN phase (test failed)
- **Issue:** Test compared `mc.containerInspectResp` to empty struct but the mock always had a response configured (regardless of whether it was called)
- **Fix:** Changed assertion from "inspect response unchanged" to "removeCalled=false" — tests the same semantic (no existing container cleanup) without the brittle struct comparison
- **Files modified:** internal/docker/manager_test.go
- **Commit:** 39125a6

---

**Total deviations:** 2 auto-fixed (Rule 1 — bugs in test setup)
**Impact on plan:** Minor test fixture corrections only; no scope creep.

## Issues Encountered
None — Docker SDK + cache integration straightforward given Plan 01 infrastructure.

## User Setup Required
None.

## Next Phase Readiness
- `Manager.Launch()` is the primary entry point for Plan 04 (Cobra wiring of `cmd/launch.go`)
- `LaunchOpts` struct is the parameter type cmd/launch.go will populate from CLI flags
- `handleZeroConfig()` is ready for Plan 04 zero-config path in launch command
- Plan 03 (stop/destroy) can use the same `removeNetwork` + cache cleanup patterns established here

---
*Phase: 06-docker-lifecycle-core*
*Completed: 2026-03-29*

## Self-Check: PASSED

- internal/docker/launch.go: FOUND
- internal/docker/quickstart.go: FOUND
- internal/docker/manager.go: FOUND
- internal/docker/manager_test.go: FOUND
- .planning/phases/06-docker-lifecycle-core/06-02-SUMMARY.md: FOUND
- Commit 56c1521: FOUND
- Commit 39125a6: FOUND
