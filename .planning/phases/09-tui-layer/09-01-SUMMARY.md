---
phase: 09-tui-layer
plan: 01
subsystem: tui
tags: [bubbletea-v2, bubbles-v2, lipgloss-v2, golang-x-term, tui, wizard, tty]

requires:
  - phase: 08-cli-commands-dx
    provides: cmd/init.go with stub non-TTY error, --plain flag on rootCmd

provides:
  - BubbleTea v2 dependency stack (bubbletea v2.0.2, bubbles v2.1.0, lipgloss v2.0.2, x/term v0.41.0)
  - internal/tui/tty.go: IsTTY(plainFlag) and IsOutputTTY() centralized TTY detection helpers
  - internal/tui/run.go: RunTUI() panic-safe wrapper with term.GetState/Restore (D-27)
  - internal/tui/init_wizard.go: full two-screen BubbleTea v2 wizard (selector + config preview)
  - cmd/init.go: TTY-gated init wizard with tui.IsTTY, tui.RunTUI, tui.NewInitWizard

affects:
  - 09-02 (build progress TUI — uses RunTUI wrapper, IsTTY pattern)
  - 09-03 (status view and log viewer TUI — uses RunTUI wrapper, IsTTY pattern)

tech-stack:
  added:
    - charm.land/bubbletea/v2 v2.0.2 (BubbleTea TUI framework v2)
    - charm.land/bubbles/v2 v2.1.0 (list, viewport, spinner components for BubbleTea v2)
    - charm.land/lipgloss/v2 v2.0.2 (terminal styling: colors, borders, layout)
    - golang.org/x/term v0.41.0 (TTY detection via IsTerminal, GetState/Restore)
  patterns:
    - RunTUI wrapper: all TUI programs must use RunTUI instead of direct tea.NewProgram().Run()
    - IsTTY(plainFlag) gate: check before any TUI launch in cmd/ layer
    - BubbleTea v2: View() returns tea.View struct, use tea.Quit (not tea.Quit()), tea.KeyPressMsg (not tea.KeyMsg)
    - Two-screen state machine: screenSelector -> screenPreview with exported result fields

key-files:
  created:
    - internal/tui/tty.go
    - internal/tui/run.go
    - internal/tui/init_wizard.go
  modified:
    - go.mod (added 4 direct deps)
    - go.sum
    - cmd/init.go (wired TUI wizard)
    - tests/cli_commands_test.go (updated TestInitNoHarness assertions)
    - tests/config_cmd_test.go (fixed hardcoded /workspace/zone path)

key-decisions:
  - "BubbleTea v2 (charm.land/bubbletea/v2 v2.0.2) confirmed stable — resolves D-30; production-validated in Charm's Crush AI agent"
  - "tea.Quit (no parens) is the Cmd in v2; tea.Quit() returns Msg — easy v1/v2 confusion"
  - "v2 window title set via v.WindowTitle on tea.View struct, not tea.SetWindowTitle() command"
  - "init_wizard.go uses inline rendering per D-28 (no v.AltScreen = true)"
  - "buildDetectionMap added to cmd/init.go; detectHarnessHints kept for non-TUI path stderr output"
  - "config_cmd_test.go hardcoded /workspace/zone path fixed to use go env GOMOD for worktree compat"

requirements-completed: [TUI-01, TUI-05, TUI-06, TUI-07]

duration: 7min
completed: 2026-04-03
---

# Phase 09 Plan 01: TUI Layer Foundation Summary

**BubbleTea v2 TUI foundation with init wizard (harness selector + config preview), panic-safe RunTUI wrapper, and centralized IsTTY() detection wired into cmd/init.go**

## Performance

- **Duration:** 7 min
- **Started:** 2026-04-03T02:56:00Z
- **Completed:** 2026-04-03T03:03:19Z
- **Tasks:** 4
- **Files modified:** 7

## Accomplishments

- Installed BubbleTea v2 dependency stack (bubbletea, bubbles, lipgloss, x/term) as direct deps in go.mod
- Created centralized `tui.IsTTY(plainFlag)` and `tui.IsOutputTTY()` replacing per-command os.Stdin.Stat() approach
- Created panic-safe `tui.RunTUI()` wrapper that saves/restores terminal state before re-panicking (implements D-27)
- Implemented full two-screen BubbleTea v2 init wizard: harness list selector with detection hints + config preview with hotkeys
- Wired wizard into cmd/init.go: `tui.IsTTY` gating, `tui.RunTUI(tui.NewInitWizard(detected))` launch, non-TTY error fallback

## Task Commits

1. **Task 1: Install BubbleTea v2 deps and create TTY detection helper** - `ace221e` (feat)
2. **Task 2: Panic-safe TUI runner with terminal state restoration** - `171c0f2` (feat)
3. **Task 3: Init Wizard BubbleTea model** - `0ec445a` (feat)
4. **Task 4: Wire Init Wizard into cmd/init.go** - `60f831b` (feat)

## Files Created/Modified

- `internal/tui/tty.go` - IsTTY(plainFlag bool) and IsOutputTTY() using golang.org/x/term
- `internal/tui/run.go` - RunTUI() panic-safe wrapper with term.GetState/Restore (D-27)
- `internal/tui/init_wizard.go` - Full two-screen wizard: screenSelector (bubbles list) + screenPreview (lipgloss box)
- `cmd/init.go` - TTY-gated init with tui.IsTTY, tui.RunTUI, tui.NewInitWizard, buildDetectionMap
- `go.mod` / `go.sum` - Added 4 direct BubbleTea v2 dependencies
- `tests/cli_commands_test.go` - Updated TestInitNoHarness for new non-TTY error message
- `tests/config_cmd_test.go` - Fixed hardcoded build path for worktree compatibility

## Decisions Made

- **BubbleTea v2 API differences confirmed:** `tea.Quit` (no parens) is the `Cmd` value; `tea.Quit()` (with parens) returns a `Msg`. Window title in v2 is set via `v.WindowTitle` on the `tea.View` struct, not via a Cmd. `tea.KeyPressMsg` replaces `tea.KeyMsg` for key handling.
- **Inline rendering for init wizard** per D-28: no `v.AltScreen = true` in View() return.
- **buildDetectionMap** added as a separate function in cmd/init.go to produce a map for the wizard, while keeping `detectHarnessHints` for the plain-text stderr detection output path.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed BubbleTea v2 API usage: tea.Quit, tea.SetWindowTitle**
- **Found during:** Task 3 (Init Wizard implementation)
- **Issue:** Plan code used `tea.Quit()` (returns Msg) and `tea.SetWindowTitle("...")` (removed in v2). Compilation failed.
- **Fix:** Changed to `tea.Quit` (the Cmd value) and `v.WindowTitle = "zone init"` on the View struct.
- **Files modified:** internal/tui/init_wizard.go
- **Verification:** `go build ./...` exits 0
- **Committed in:** `0ec445a` (Task 3 commit)

**2. [Rule 1 - Bug] Fixed hardcoded build path in config_cmd_test.go breaking worktree tests**
- **Found during:** Task 4 verification (`go test ./...`)
- **Issue:** `buildCmd.Dir = "/workspace/zone"` builds binary from main workspace, not worktree. TestInitNoHarness was testing old code.
- **Fix:** Use `go env GOMOD` to detect module root dynamically; update TestInitNoHarness assertions to match new non-TTY error message.
- **Files modified:** tests/config_cmd_test.go, tests/cli_commands_test.go
- **Verification:** `go test ./...` passes
- **Committed in:** `60f831b` (Task 4 commit)

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both fixes necessary for compilation and test correctness. No scope creep.

## Issues Encountered

- BubbleTea v2 module requires sub-package fetches (charm.land/bubbles/v2/list, /textinput) because go mod tidy removes indirect deps before code imports them. Required installing sub-packages explicitly before go mod tidy.

## Next Phase Readiness

- Foundation complete: RunTUI wrapper, IsTTY helper, and init wizard are all working
- Plans 09-02 (build progress) and 09-03 (status/logs viewers) can use the established RunTUI/IsTTY patterns
- No blockers

## Self-Check: PASSED

All files verified:
- FOUND: internal/tui/tty.go
- FOUND: internal/tui/run.go
- FOUND: internal/tui/init_wizard.go
- FOUND: .planning/phases/09-tui-layer/09-01-SUMMARY.md

All commits verified:
- FOUND: ace221e (Task 1: BubbleTea v2 deps + TTY helper)
- FOUND: 171c0f2 (Task 2: panic-safe RunTUI)
- FOUND: 0ec445a (Task 3: Init Wizard model)
- FOUND: 60f831b (Task 4: wire cmd/init.go)

---
*Phase: 09-tui-layer*
*Completed: 2026-04-03*
