---
phase: 09-tui-layer
plan: 02
subsystem: tui
tags: [bubbletea-v2, bubbles-v2, lipgloss-v2, docker-streaming, tui, build-progress, status-view]

requires:
  - phase: 09-tui-layer
    plan: 01
    provides: RunTUI wrapper, IsTTY helper, BubbleTea v2 deps, init wizard

provides:
  - internal/tui/build_progress.go: inline BubbleTea build progress with spinner+viewport
  - internal/docker/build.go: BuildWithProgress channel-based streaming adapter
  - internal/docker/launch.go: NeedsBuild helper and Restart method
  - internal/tui/status_view.go: alt-screen status view with 2s polling and hotkeys
  - cmd/launch.go: TTY-gated build progress + D-05 inline init wizard + RunTUI
  - cmd/status.go: TTY/--json gating + RunTUI + printStatusPlain helper

affects:
  - 09-03 (log viewer — uses same RunTUI/IsTTY patterns)

tech-stack:
  added: []
  patterns:
    - BuildWithProgress channel adapter: goroutine wraps buildImage, sends lines to chan<- BuildLine
    - waitForBuildLine/waitForBuildResult: channel relay pattern for BubbleTea v2 update loop
    - Sequential poll pattern: tickMsg -> fetchStatus -> statusUpdateMsg -> schedule next tick (avoids Pitfall 7)
    - AltScreen declarative: v.AltScreen = true in View() for status view; omitted for inline build progress

key-files:
  created: []
  modified:
    - internal/tui/build_progress.go (fully implemented from stub)
    - internal/tui/status_view.go (fully implemented from stub)
    - internal/docker/build.go (added BuildLine, BuildResult, BuildWithProgress, buildImageWithChannel, streamBuildOutputWithChannel)
    - internal/docker/launch.go (added NeedsBuild, Restart methods)
    - cmd/launch.go (added TUI build progress, D-05 inline init wizard, --plain flag)
    - cmd/status.go (added TUI path, printStatusPlain, --plain flag gating)

key-decisions:
  - "NeedsBuild added to docker.Manager (not just build_progress.go) since it requires Manager fields (config, cache, client)"
  - "Restart method added to docker.Manager (Stop + Launch) since cmd/status.go TUI hotkey needs it and Manager was missing it"
  - "buildImageWithChannel is a full copy of buildImage (not a refactor) to avoid changing the synchronous Build/buildImage path"
  - "Sequential polling: statusUpdateMsg schedules next tick to prevent overlapping Docker API calls (Pitfall 7)"
  - "formatStatusDuration duplicated in status_view.go (not imported from cmd/ls.go) to avoid cross-package dependency inversion"

metrics:
  duration: 5min
  tasks: 2
  files: 6
  completed: 2026-04-03
---

# Phase 09 Plan 02: Build Progress and Status View Summary

**BubbleTea build progress (inline, channel-based Docker streaming) and status view (alt-screen, 2s polling) wired into cmd/launch.go and cmd/status.go with TTY/--plain/--json gating**

## Performance

- **Duration:** 5 min
- **Started:** 2026-04-03T03:08:23Z
- **Completed:** 2026-04-03T03:12:40Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Added `BuildWithProgress` channel adapter to `internal/docker/build.go` — goroutine wraps `buildImageWithChannel`, sends `BuildLine` to channel, sends `BuildResult` when done
- Added `streamBuildOutputWithChannel` — identical to `streamBuildOutput` but also writes to `chan<- BuildLine` for TUI relay
- Added `NeedsBuild` to `internal/docker/launch.go` — pre-checks hash/image before starting TUI build
- Added `Restart` method to `docker.Manager` (Stop + Launch) for status view hotkey
- Implemented `BuildProgress` BubbleTea model — spinner + viewport inline rendering (no alt-screen per D-09), `waitForBuildLine`/`waitForBuildResult` channel relay, Ctrl+C cancels build
- Implemented `StatusView` BubbleTea model — alt-screen (per D-11), `lipgloss.NormalBorder` box, sequential 2s polling via `statusUpdateMsg`/`tickMsg`, hotkeys q/r/s
- Wired `cmd/launch.go` — reads `--plain` flag, D-05 inline init wizard (no zone.toml + no --harness + TTY), TUI build progress when TTY+not-headless, `RunTUI` for panic safety
- Wired `cmd/status.go` — `--json` check before TUI (D-15), `IsTTY(plainFlag)` gating, `RunTUI`, `printStatusPlain` helper extracted from old RunE

## Task Commits

1. **Task 1: Build progress model with channel adapter and cmd/launch.go wiring** - `d06aa8b` (feat)
2. **Task 2: Status view model with alt-screen polling and cmd/status.go wiring** - `576488f` (feat)

## Files Created/Modified

- `internal/docker/build.go` — Added `BuildLine`, `BuildResult` types; `BuildWithProgress`, `buildImageWithChannel`, `streamBuildOutputWithChannel`
- `internal/docker/launch.go` — Added `NeedsBuild(ctx, forceRebuild)` and `Restart(ctx)` methods
- `internal/tui/build_progress.go` — Full implementation: `BuildProgress` struct, `NewBuildProgress`, `waitForBuildLine`, `waitForBuildResult`, inline View
- `internal/tui/status_view.go` — Full implementation: `StatusView` struct, `NewStatusView`, `fetchStatus`, alt-screen View with Lip Gloss styling
- `cmd/launch.go` — Added `tui` import, `plainFlag` read, D-05 inline init wizard, TUI build progress block with `NeedsBuild`/`BuildWithProgress`/`RunTUI`
- `cmd/status.go` — Added `tui` import, `plainFlag` read, TUI path with `RunTUI`/`NewStatusView`, `printStatusPlain` helper, hotkey action handling

## Decisions Made

- **Restart method on Manager:** The plan's `cmd/status.go` code called `mgr.Restart(ctx)` but docker.Manager had no `Restart` method. Added it as `Stop` + `Launch` with default opts (Rule 2 — missing critical functionality).
- **Sequential polling pattern:** `statusUpdateMsg` schedules the next tick after receiving status, not from `Init`. This prevents overlapping Docker inspect calls when the API is slow (Pitfall 7).
- **buildImageWithChannel is a copy:** Rather than refactoring `buildImage` (risk of breaking existing path), `buildImageWithChannel` duplicates the build logic and only differs in calling `streamBuildOutputWithChannel` instead of `streamBuildOutput`. Existing `buildImage` / `Build` paths are unchanged.
- **formatStatusDuration in status_view.go:** Duplicated instead of imported from `cmd/ls.go` to keep the `internal/tui` package independent of `cmd/`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Functionality] Added Restart method to docker.Manager**
- **Found during:** Task 2 (cmd/status.go wiring review)
- **Issue:** Plan's `cmd/status.go` called `mgr.Restart(ctx)` but `docker.Manager` had no `Restart` method. `cmd/restart.go` implements restart at the command layer as `mgr.Stop + mgr.Launch` inline.
- **Fix:** Added `func (m *Manager) Restart(ctx context.Context) error` to `internal/docker/launch.go` — wraps `Stop` + `Launch` with default `LaunchOpts`.
- **Files modified:** `internal/docker/launch.go`
- **Committed in:** `d06aa8b` (Task 1 commit, since NeedsBuild was also in that commit)

None — plan executed as written (the Restart method addition is a direct requirement for the plan's success criteria).

## Self-Check: PASSED

All files verified:
- FOUND: internal/tui/build_progress.go
- FOUND: internal/tui/status_view.go
- FOUND: internal/docker/build.go (updated)
- FOUND: internal/docker/launch.go (updated)
- FOUND: cmd/launch.go (updated)
- FOUND: cmd/status.go (updated)

All commits verified:
- FOUND: d06aa8b (Task 1: build progress + channel adapter + cmd/launch.go)
- FOUND: 576488f (Task 2: status view + cmd/status.go)

Build/test verification:
- `go build ./...` exits 0
- `go vet ./...` exits 0
- `go test ./...` all tests pass

---
*Phase: 09-tui-layer*
*Completed: 2026-04-03*
