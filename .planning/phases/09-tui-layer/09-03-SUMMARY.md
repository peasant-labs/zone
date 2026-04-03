---
phase: 09-tui-layer
plan: 03
subsystem: tui
tags: [bubbletea-v2, bubbles-v2, viewport, log-viewer, follow-mode, search, tty-gating]

requires:
  - phase: 09-tui-layer
    plan: 01
    provides: RunTUI panic-safe wrapper, IsTTY/IsOutputTTY helpers, BubbleTea v2 dep stack

provides:
  - BubbleTea alt-screen log viewer with follow mode and search
  - internal/tui/log_viewer.go: LogViewer model with viewport, follow channel relay, substring search
  - cmd/logs.go: TUI-gated logs command with TTY+stdout pipe detection

affects:
  - All zone logs invocations in TTY environments

tech-stack:
  added: []
  patterns:
    - Dual TTY guard (IsTTY AND IsOutputTTY) before TUI launch — prevents garbling piped output (Pitfall 3 / D-20)
    - Channel relay pattern: goroutine streams new log lines, waitForLogLine() bridges channel to tea.Cmd
    - viewport.SetHighlights() with byte-offset ranges for search match highlighting
    - AltScreen = true on tea.View for full-terminal log viewer

key-files:
  created: []
  modified:
    - internal/tui/log_viewer.go
    - cmd/logs.go

key-decisions:
  - "Dual TTY guard (IsTTY + IsOutputTTY) required so `zone logs -f | grep error` streams plain text (D-20 / Pitfall 3)"
  - "viewport.SetHighlights() used for search highlighting instead of manual string replacement — leverages built-in byte-range highlight API"
  - "Follow mode goroutine uses io.Pipe() to bridge mgr.Logs streaming into bufio.Scanner for line-by-line channel relay"
  - "AltScreen = true on tea.View for log viewer (alt-screen appropriate for pager, per D-16 / D-28)"

metrics:
  duration: 8min
  completed: 2026-04-03
  tasks: 2
  files: 2
---

# Phase 09 Plan 03: Log Viewer TUI Summary

**BubbleTea alt-screen log viewer with follow mode, substring search, and dual TTY gating wired into cmd/logs.go**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-03T03:10:00Z
- **Completed:** 2026-04-03T03:18:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Implemented `LogViewer` BubbleTea v2 model in `internal/tui/log_viewer.go` with:
  - Alt-screen viewport (`v.AltScreen = true`) for full-terminal pager experience
  - Follow mode with channel relay: `waitForLogLine()` bridges a `<-chan string` into the tea message loop
  - Case-insensitive substring search via `/` key with `strings.Contains` matching
  - `viewport.SetHighlights()` byte-range API for highlighting all search matches in viewport
  - Hotkeys: q quit, f toggle follow, up/down scroll, g/G top/bottom, n/N navigate matches
  - Status bar showing follow status, search query, match count
  - Search input prompt at bottom while typing
- Wired log viewer into `cmd/logs.go` with correct dual TTY gate:
  - `tui.IsTTY(plainFlag) && tui.IsOutputTTY()` required for TUI path (D-20 / Pitfall 3)
  - `--json`, `--plain`, piped stdout, non-TTY stdin all bypass TUI to `mgr.Logs()` plain text
  - Follow mode: goroutine uses `io.Pipe()` + `bufio.Scanner` to relay new log lines via `followCh`
  - Build log (`--build`): shows file content in TUI viewer when in TTY, plain text otherwise
  - All TUI launches use `tui.RunTUI()` wrapper for panic safety (D-27)

## Task Commits

1. **Task 1: Log viewer model with alt-screen viewport, follow mode, and search** - `a1b2a5d` (feat)
2. **Task 2: Wire log viewer into cmd/logs.go with TTY/pipe/--json/--plain gating** - `d31cddb` (feat)

## Files Created/Modified

- `internal/tui/log_viewer.go` - LogViewer BubbleTea model: viewport, follow channel, search, AltScreen
- `cmd/logs.go` - TUI-gated logs command: dual TTY check, follow goroutine, build log TUI path

## Decisions Made

- **Dual TTY guard required for pipe safety:** `zone logs -f | grep error` must output plain text — stdout pipe detection via `tui.IsOutputTTY()` is critical (D-20 / Pitfall 3).
- **viewport.SetHighlights() for search:** The bubbles/v2 viewport has a built-in highlight API that takes byte-offset ranges (`[][]int`). Used this instead of manually replacing strings in content, which preserves ANSI codes and is more correct.
- **AltScreen = true for log viewer:** Unlike the init wizard (which uses inline rendering per D-28), the log viewer is a full-screen pager that benefits from the alternate screen buffer — clears on exit, no scroll history contamination.
- **Follow goroutine cleanup:** `cancel()` is called explicitly after `RunTUI` returns (or errors) to ensure the follow goroutine terminates promptly.

## Deviations from Plan

None — plan executed exactly as written.

The plan specified `viewport.SetYOffset()` for jumping to search matches and `HighlightNext()`/`HighlightPrevious()` for n/N navigation — both are available in the bubbles/v2 viewport API and were used as specified.

## Self-Check: PASSED

Files verified:
- FOUND: internal/tui/log_viewer.go
- FOUND: cmd/logs.go
- FOUND: .planning/phases/09-tui-layer/09-03-SUMMARY.md

Commits verified:
- FOUND: a1b2a5d (Task 1: log viewer model)
- FOUND: d31cddb (Task 2: wire cmd/logs.go)

---
*Phase: 09-tui-layer*
*Completed: 2026-04-03*
