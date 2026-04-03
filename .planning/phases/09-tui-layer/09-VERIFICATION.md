---
phase: 09-tui-layer
verified: 2026-04-03T04:00:00Z
status: human_needed
score: 14/14 must-haves verified
re_verification: false
gaps:
  - truth: "cmd/init.go does not contain func isInteractive() (old check removed per plan acceptance criteria)"
    status: resolved
    reason: "Fixed inline during execution — dead code block, isInteractive(), promptHarnessSelection(), and unused imports removed."
  - truth: "REQUIREMENTS.md checkboxes for TUI-02 and TUI-03 are marked complete"
    status: resolved
    reason: "Fixed inline during execution — TUI-02 and TUI-03 marked [x] in REQUIREMENTS.md."
human_verification:
  - test: "Run `zone init` in a real TTY without --harness"
    expected: "BubbleTea harness-selection list renders with arrow-key navigation; harnesses with indicator files show '* detected'; pressing Enter transitions to config preview screen with skip_permissions toggle hotkey; pressing Enter confirms and writes zone.toml"
    why_human: "TUI rendering, interactive key handling, and alt-screen lifecycle cannot be verified without an actual terminal session"
  - test: "Run `zone launch` in a TTY when a build is needed"
    expected: "Spinner + streaming build output renders inline (not alt-screen); each Docker build step appears line by line; pressing Ctrl+C cancels; after build completes the TUI clears and launch continues"
    why_human: "Requires Docker daemon and a real TTY to observe streaming behaviour"
  - test: "Run `zone status` in a TTY"
    expected: "Alt-screen box-drawn status display appears with container state, uptime, ports; live refresh every 2 seconds; pressing r/s triggers restart/stop after exit; pressing q exits cleanly"
    why_human: "Alt-screen modal UI and live polling require a real terminal and running container"
  - test: "Run `zone logs` in a TTY, then run `zone logs -f | grep something` in a shell"
    expected: "First: alt-screen viewer with / search, f follow toggle, up/down scroll. Second (piped): plain text flows through without ANSI garbling"
    why_human: "Requires a running container, real TTY, and shell pipeline to test the dual TTY guard"
  - test: "Run `zone init` and `zone logs` with --plain flag in a TTY"
    expected: "Both commands bypass TUI entirely and produce plain text output even though stdin/stdout are terminals"
    why_human: "Requires a real TTY to confirm --plain override works"
  - test: "Trigger a panic inside a TUI component and verify terminal is restored"
    expected: "Terminal returns to normal cooked mode after the panic; shell prompt is usable; panic message includes 'tui panic (terminal restored)'"
    why_human: "Requires injecting a panic and observing terminal state post-crash"
---

# Phase 09: TUI Layer Verification Report

**Phase Goal:** Interactive users get a polished BubbleTea interface; non-TTY users and CI environments get clean plain-text output automatically
**Verified:** 2026-04-03T04:00:00Z
**Status:** gaps_found — 2 gaps (1 dead code / plan criteria violation, 1 tracker sync)
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `zone init` in TTY launches BubbleTea harness-selection wizard | ? HUMAN | `tui.NewInitWizard` + `tui.RunTUI` wired in `cmd/init.go` lines 47-60; `tui.IsTTY(plainFlag)` gates correctly |
| 2 | Config preview shows after selection with hotkeys for toggle/confirm/cancel | ? HUMAN | `screenPreview` in `init_wizard.go` with `[s]` skipPermissions toggle, Enter confirm, q cancel — fully implemented |
| 3 | `zone init` without --harness in non-TTY errors with helpful message | VERIFIED | Test `TestInitNoHarness` passes; `cmd/init.go` line 38 returns correct error with `--harness <name>` guidance |
| 4 | `--plain` flag disables TUI and falls back to existing text prompt | VERIFIED | `tui.IsTTY(plainFlag)` in all four commands; `--plain` registered as persistent flag on `rootCmd` |
| 5 | Detected harness indicators show `* detected` hint in wizard list | ? HUMAN | `harnessItem.Description()` appends `"  * detected"` when `detected==true`; `buildDetectionMap` in `cmd/init.go` |
| 6 | Terminal state fully restored after TUI session including panics | ? HUMAN | `RunTUI` in `internal/tui/run.go`: `term.GetState` before, deferred `term.Restore` + re-panic in recover block |
| 7 | Docker build output streams line-by-line through BubbleTea spinner+viewport during `zone launch` | ? HUMAN | `BuildWithProgress`+`streamBuildOutputWithChannel` in `docker/build.go`; `NewBuildProgress`+`waitForBuildLine` channel relay in `build_progress.go`; wired in `cmd/launch.go` lines 126-143 |
| 8 | Build progress renders inline (not alt-screen) and terminal clean after build | VERIFIED | `build_progress.go` View() has no `v.AltScreen = true`; comment confirms `// Inline rendering: v.AltScreen is NOT set (build progress is transient, per D-09)` |
| 9 | `zone status` in TTY shows alt-screen box-drawn live status with 2-second polling | ? HUMAN | `status_view.go` has `v.AltScreen = true`, `tea.Tick(2*time.Second, ...)`, `lipgloss.NormalBorder()` box; wired in `cmd/status.go` |
| 10 | `zone status --json` bypasses TUI entirely and prints raw JSON | VERIFIED | `cmd/status.go` line 58: `if jsonMode \|\| !tui.IsTTY(plainFlag)` — json handled before TUI path |
| 11 | `zone status` hotkeys q/r/s work for quit/restart/stop | VERIFIED | `status_view.go` Update handles `"r"` → `m.Action = "restart"`, `"s"` → `m.Action = "stop"`, `"q"` → `tea.Quit`; `cmd/status.go` acts on `sv.Action` |
| 12 | `zone logs` in TTY shows alt-screen viewport with scrollable log content | ? HUMAN | `log_viewer.go` has `v.AltScreen = true`, `viewport.Model`; wired in `cmd/logs.go` with dual TTY guard |
| 13 | `zone logs --follow` starts in follow mode with auto-scroll to bottom | ? HUMAN | `NewLogViewer(..., true)` calls `vp.GotoBottom()`; `followCh` channel relay wired in `cmd/logs.go` follow block |
| 14 | `zone logs -f \| grep error` works without ANSI garbling | VERIFIED | `cmd/logs.go` line 94: `if jsonMode \|\| !tui.IsTTY(plainFlag) \|\| !tui.IsOutputTTY()` — piped stdout bypasses TUI |
| 15 | `zone logs --build` shows build log in TUI viewer | VERIFIED | `cmd/logs.go` lines 60-65: `--build` mode with `tui.IsTTY(plainFlag) && tui.IsOutputTTY()` guard, then `tui.RunTUI(tui.NewLogViewer(data, nil, false))` |
| 16 | `zone launch` without zone.toml and without --harness in TTY launches init wizard inline then proceeds to build | ? HUMAN | D-05 path wired at `cmd/launch.go` lines 57-83: `tui.IsTTY(plainFlag)` check, wizard launch, zone.toml write, then continues to build |
| 17 | Dead code removed: `isInteractive()` and `promptHarnessSelection()` gone from `cmd/init.go` | FAILED | Both functions still present at lines 252-292; unreachable block at lines 70-81 still references them |

**Score:** 14/17 truths verified (6 VERIFIED, 7 HUMAN, 2 FAILED→gaps)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/tui/tty.go` | IsTTY and IsOutputTTY helpers | VERIFIED | Exports `IsTTY(plainFlag bool) bool` and `IsOutputTTY() bool` using `golang.org/x/term.IsTerminal` |
| `internal/tui/run.go` | Panic-safe RunTUI wrapper | VERIFIED | `RunTUI(model tea.Model, opts ...tea.ProgramOption)` with `term.GetState`/deferred `term.Restore`+re-panic |
| `internal/tui/init_wizard.go` | Two-screen BubbleTea wizard | VERIFIED | `NewInitWizard`, `screenSelector`/`screenPreview`, `SelectedHarness`, `Confirmed`, `Cancelled`; `tea.KeyPressMsg`; `tea.NewView`; bubbles list and lipgloss |
| `internal/tui/build_progress.go` | Build progress with spinner+viewport | VERIFIED | `NewBuildProgress`, `BuildProgress.ImageID/BuildErr`, `waitForBuildLine/waitForBuildResult`, spinner, viewport; inline (no AltScreen) |
| `internal/docker/build.go` | `BuildWithProgress` channel adapter | VERIFIED | `BuildLine`, `BuildResult` types; `BuildWithProgress` goroutine wrapper; `streamBuildOutputWithChannel` |
| `internal/docker/launch.go` | `NeedsBuild` method | VERIFIED | `NeedsBuild(ctx, forceRebuild bool) bool` checks force-rebuild, hash, image existence; `Restart(ctx)` also added |
| `internal/tui/status_view.go` | Alt-screen status view with polling | VERIFIED | `NewStatusView`, `StatusView.Action/Err`, `tickMsg`, `statusUpdateMsg`, `tea.Tick(2*time.Second)`, `AltScreen = true`, lipgloss border box |
| `internal/tui/log_viewer.go` | Alt-screen log viewer with follow+search | VERIFIED | `NewLogViewer`, `LogViewer`, `followMode`, `searchMode`, `waitForLogLine`, `AltScreen = true`, `viewport.SetHighlights`, substring search |
| `cmd/init.go` | Wired TUI init with TTY gating | PARTIAL | TUI wizard correctly wired (lines 35-61); however dead code block (lines 70-81) and dead functions `isInteractive()`/`promptHarnessSelection()` remain |
| `cmd/launch.go` | TUI build progress wired | VERIFIED | `tui.IsTTY(plainFlag)`, `mgr.NeedsBuild`, `tui.RunTUI(tui.NewBuildProgress(...))`, D-05 inline init wizard |
| `cmd/status.go` | TUI status wired with JSON/plain gating | VERIFIED | `--json` bypasses TUI first, `tui.IsTTY(plainFlag)` gates TUI, `tui.RunTUI(tui.NewStatusView(...))`, `printStatusPlain` helper |
| `cmd/logs.go` | TUI log viewer wired with dual TTY gate | VERIFIED | `tui.IsTTY(plainFlag) && tui.IsOutputTTY()` dual guard, `--build` mode, follow goroutine, `tui.RunTUI(tui.NewLogViewer(...))` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `cmd/init.go` | `internal/tui/init_wizard.go` | `tui.NewInitWizard(detected)` | WIRED | Line 47 |
| `cmd/init.go` | `internal/tui/tty.go` | `tui.IsTTY(plainFlag)` | WIRED | Line 37 |
| `cmd/init.go` | `internal/tui/run.go` | `tui.RunTUI(wizard)` | WIRED | Line 48 |
| `cmd/launch.go` | `internal/tui/build_progress.go` | `tui.NewBuildProgress(linesCh, resultCh, cancel)` | WIRED | Line 132 |
| `cmd/launch.go` | `internal/tui/run.go` | `tui.RunTUI(model)` | WIRED | Line 133 |
| `cmd/launch.go` | `internal/tui/init_wizard.go` | D-05 inline wizard | WIRED | Lines 60-61 |
| `cmd/status.go` | `internal/tui/status_view.go` | `tui.NewStatusView(ctx, mgr, cfg, cwd)` | WIRED | Line 77 |
| `cmd/status.go` | `internal/tui/run.go` | `tui.RunTUI(model)` | WIRED | Line 78 |
| `cmd/logs.go` | `internal/tui/log_viewer.go` | `tui.NewLogViewer(...)` | WIRED | Lines 61, 128, 144 |
| `cmd/logs.go` | `internal/tui/tty.go` | `tui.IsTTY(plainFlag) && tui.IsOutputTTY()` | WIRED | Lines 60, 94 |
| `cmd/logs.go` | `internal/tui/run.go` | `tui.RunTUI(model)` | WIRED | Lines 62, 129, 145 |
| `internal/tui/build_progress.go` | `internal/docker/build.go` | `chan docker.BuildLine` / `chan docker.BuildResult` | WIRED | Channel types used in `waitForBuildLine`/`waitForBuildResult`; `BuildWithProgress` produces them |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| TUI-01 | 09-01 | Init wizard with BubbleTea interactive harness selection and config preview | SATISFIED | `init_wizard.go` two-screen model; `cmd/init.go` wired; `TestInitNoHarness` passes; REQUIREMENTS.md marked [x] |
| TUI-02 | 09-02 | Build progress display with Docker build log streaming | SATISFIED (code) / TRACKER STALE | `BuildWithProgress` + `build_progress.go` + `cmd/launch.go` wiring all verified; REQUIREMENTS.md still shows `[ ]` |
| TUI-03 | 09-02 | Status view with live container state, uptime, ports, resources | SATISFIED (code) / TRACKER STALE | `status_view.go` with alt-screen, 2s polling, hotkeys; `cmd/status.go` wiring verified; REQUIREMENTS.md still shows `[ ]` |
| TUI-04 | 09-03 | Log viewer with follow mode and build log option | SATISFIED | `log_viewer.go` follow+search+AltScreen; `cmd/logs.go` dual TTY guard; REQUIREMENTS.md marked [x] |
| TUI-05 | 09-01 | TTY auto-detection: BubbleTea when TTY, plain text when not | SATISFIED | `tui.IsTTY(plainFlag)` used in all 4 commands; REQUIREMENTS.md marked [x] |
| TUI-06 | 09-01 | `--plain` flag force-disables TUI even in TTY | SATISFIED | `--plain` persistent flag on rootCmd; `IsTTY(plainFlag)` returns false when true; REQUIREMENTS.md marked [x] |
| TUI-07 | 09-01 | Non-TTY `zone init` without `--harness` errors with helpful message | SATISFIED | `cmd/init.go` line 38 returns error with guidance; `TestInitNoHarness` asserts message; REQUIREMENTS.md marked [x] |

**Orphaned requirements:** None — all 7 TUI requirements claimed by plans 09-01, 09-02, 09-03.

### Anti-Patterns Found

| File | Lines | Pattern | Severity | Impact |
|------|-------|---------|----------|--------|
| `cmd/init.go` | 70-81 | Unreachable `if harnessName == ""` block — never entered because the preceding block at 35-61 always sets `harnessName` or returns | BLOCKER (plan criteria) | Does not break functionality; violates plan acceptance criterion `cmd/init.go does NOT contain func isInteractive()` |
| `cmd/init.go` | 252-258 | Dead function `isInteractive()` — unreachable, superseded by `tui.IsTTY` | BLOCKER (plan criteria) | Dead code retained; old OS-level TTY check kept alongside centralized helper |
| `cmd/init.go` | 261-292 | Dead function `promptHarnessSelection()` — unreachable, superseded by BubbleTea wizard | BLOCKER (plan criteria) | Dead code retained; plain-text selection prompt kept alongside TUI wizard |
| `cmd/init.go` | 4, 8 | `bufio` and `strconv` imports are only used by the dead `promptHarnessSelection()` function | WARNING | Retained solely because of dead code; will fail if dead functions are removed without removing imports |
| `.planning/REQUIREMENTS.md` | 105-106 | TUI-02 and TUI-03 checkboxes still `[ ]` despite full implementation | WARNING | Tracking is misleading; code is correct |

### Human Verification Required

#### 1. Init Wizard Interactive Flow

**Test:** In a real terminal, run `zone init` in a directory without zone.toml and without --harness. Navigate the BubbleTea list, observe `* detected` hints if applicable, press Enter to proceed to config preview, toggle skip_permissions with `[s]`, confirm with Enter.
**Expected:** Harness list renders with arrow navigation; detected harnesses show `* detected`; config preview shows lipgloss bordered box with harness details; Enter writes zone.toml; q or Ctrl+C cancels with "init cancelled" message.
**Why human:** BubbleTea interactive list rendering and multi-screen state machine require a real PTY.

#### 2. Build Progress Streaming

**Test:** In a real terminal with Docker running, run `zone launch` in a repo that needs a build (or `zone launch --rebuild`).
**Expected:** Spinner appears with "Building image..." header; Docker build steps stream line-by-line into the viewport; step count increments; on completion "Build complete (N steps)" shows; launch proceeds.
**Why human:** Requires Docker daemon, real TTY, and observable streaming render timing.

#### 3. Status View Alt-Screen with Polling

**Test:** With a running zone container, run `zone status` in a real terminal.
**Expected:** Alt-screen (normal terminal content hidden) shows lipgloss border box with container details; refreshes every ~2 seconds; pressing `r` exits and triggers restart; `s` exits and triggers stop; `q` exits cleanly restoring terminal.
**Why human:** Alt-screen lifecycle and live Docker polling require a running container and real TTY.

#### 4. Log Viewer with Pipe Safety

**Test 1:** Run `zone logs` in a real terminal with a running container. Use `/` to search, `f` to toggle follow, `n`/`N` to navigate matches, `g`/`G` to jump top/bottom.
**Test 2:** Run `zone logs -f | grep <something>` in a shell — verify plain text flows through without ANSI escape codes corrupting grep output.
**Expected:** Test 1: alt-screen pager with functional search highlighting and follow mode. Test 2: plain text output, grep works correctly.
**Why human:** Alt-screen pager UX and pipe behaviour require real terminal and shell.

#### 5. --plain Flag Override

**Test:** In a real TTY, run `zone init --plain` (without --harness) and `zone logs --plain`.
**Expected:** `zone init --plain` skips the TUI wizard and falls through to the non-TTY error path (since stdin IS a terminal but --plain overrides). `zone logs --plain` outputs plain text even though stdin/stdout are terminals.
**Why human:** Requires real TTY to confirm the `IsTTY(plainFlag)` false-return path is exercised.

#### 6. Panic-safe Terminal Restoration

**Test:** Inject a panic (e.g., via a debug build) inside a TUI component after `tea.NewProgram(model).Run()` begins.
**Expected:** Terminal returns to cooked mode (shell prompt usable, no raw-mode artifacts); panic message includes "tui panic (terminal restored):".
**Why human:** Cannot inject panics via grep; requires build instrumentation or deliberate panic trigger.

### Gaps Summary

Two gaps prevent a full "passed" status:

**Gap 1 — Dead code in cmd/init.go (code quality / plan criteria violation):**
The plan for 09-01 explicitly required `func isInteractive()` to be removed from `cmd/init.go`. It still exists at lines 252-258, alongside `promptHarnessSelection()` (lines 261-292) and an unreachable second harness-selection block (lines 70-81). These functions are not reachable in any execution path — after the BubbleTea wizard block at lines 35-61 either sets `harnessName` or returns an error, the second `if harnessName == ""` check at line 70 can never be true. The dead code does not affect runtime behaviour but violates the plan's acceptance criteria and leaves a maintenance hazard (two competing TTY-check strategies in the same file).

**Gap 2 — REQUIREMENTS.md tracker out of sync:**
TUI-02 and TUI-03 are fully implemented (build progress TUI in `cmd/launch.go` + `internal/tui/build_progress.go`; status view TUI in `cmd/status.go` + `internal/tui/status_view.go`), but REQUIREMENTS.md still marks them as incomplete with `[ ]`. This is a documentation/tracking inconsistency — the tracker table at lines 249-250 correctly says "Pending" but the implementation is complete.

Both gaps are quick fixes (dead code removal, checkbox update) that do not require new features.

---

_Verified: 2026-04-03T04:00:00Z_
_Verifier: Claude (gsd-verifier)_
