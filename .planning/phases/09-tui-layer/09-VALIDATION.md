---
phase: 9
slug: tui-layer
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-03
---

# Phase 9 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify v1.11.1 |
| **Config file** | none (standard `go test ./...`) |
| **Quick run command** | `go build ./... && go vet ./...` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go build ./... && go vet ./...`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

All tasks use build+grep automated verification. BubbleTea TUI models can be unit-tested by calling `model.Update(msg)` directly, but the primary verification during execution is compile+vet+grep to confirm correct exports, function signatures, and wiring patterns. TUI rendering correctness requires manual TTY verification (see Manual-Only Verifications below).

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | Status |
|---------|------|------|-------------|-----------|-------------------|--------|
| 09-01-01 | 01 | 1 | TUI-05 | build+grep | `go build ./... && grep "func IsTTY" internal/tui/tty.go && grep "func IsOutputTTY" internal/tui/tty.go` | pending |
| 09-01-02 | 01 | 1 | D-27 | build+grep | `go build ./... && grep "func RunTUI" internal/tui/run.go && grep "term.Restore" internal/tui/run.go && grep "recover()" internal/tui/run.go` | pending |
| 09-01-03 | 01 | 1 | TUI-01 | build+grep | `go build ./... && grep "func NewInitWizard" internal/tui/init_wizard.go && grep "tea.KeyPressMsg" internal/tui/init_wizard.go` | pending |
| 09-01-04 | 01 | 1 | TUI-06, TUI-07 | build+grep | `go build ./... && go vet ./cmd/... && grep "tui.RunTUI" cmd/init.go && grep "tui.IsTTY" cmd/init.go && ! grep "func isInteractive" cmd/init.go` | pending |
| 09-02-01 | 02 | 2 | TUI-02 | build+grep | `go build ./... && go vet ./... && grep "func.*BuildWithProgress" internal/docker/build.go && grep "func NewBuildProgress" internal/tui/build_progress.go && grep "func.*NeedsBuild" internal/docker/launch.go && grep "tui.RunTUI" cmd/launch.go` | pending |
| 09-02-02 | 02 | 2 | TUI-03 | build+grep | `go build ./... && go vet ./... && grep "func NewStatusView" internal/tui/status_view.go && grep "AltScreen = true" internal/tui/status_view.go && grep "tui.RunTUI" cmd/status.go` | pending |
| 09-03-01 | 03 | 2 | TUI-04 | build+grep | `go build ./... && grep "func NewLogViewer" internal/tui/log_viewer.go && grep "AltScreen = true" internal/tui/log_viewer.go && grep "searchMode" internal/tui/log_viewer.go` | pending |
| 09-03-02 | 03 | 2 | TUI-04 | build+grep | `go build ./... && go vet ./cmd/... && grep "tui.RunTUI" cmd/logs.go && grep "tui.IsOutputTTY" cmd/logs.go && grep "followCh" cmd/logs.go` | pending |

*Status: pending | green | red | flaky*

---

## Wave 0 Requirements

All Wave 0 dependencies are satisfied by Plan 01 Task 1 (dependency install + TTY helper) and Task 2 (RunTUI panic recovery wrapper):

- [x] Framework install: `go get charm.land/bubbletea/v2@v2.0.2 charm.land/bubbles/v2@v2.1.0 charm.land/lipgloss/v2@v2.0.2 golang.org/x/term@v0.41.0` (Plan 01 Task 1)
- [x] `internal/tui/tty.go` -- IsTTY() and IsOutputTTY() helpers (Plan 01 Task 1)
- [x] `internal/tui/run.go` -- RunTUI() panic-safe wrapper with term.GetState/term.Restore (Plan 01 Task 2)

No separate test stub files are required. Verification uses build+grep patterns that validate compilation success and correct function signatures/wiring without needing pre-existing test scaffolds.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Init wizard renders correctly in TTY | TUI-01 | Requires real terminal to verify visual rendering | Run `zone init` in a TTY, verify list navigation and config preview |
| Build progress scrolling | TUI-02 | Requires real Docker build output stream | Run `zone launch --rebuild` and verify progress viewport |
| Terminal state restored after panic | D-27 | Requires simulating process crash | Force-kill zone during TUI, verify terminal is not garbled |
| Status view hotkeys (r restart, s stop) | TUI-03 | Requires running container + real terminal | Run `zone status`, press r/s, verify container state changes |
| Log viewer follow mode + search | TUI-04 | Requires real terminal for interactive testing | Run `zone logs --follow`, type `/search`, verify highlighting |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify commands (build+grep pattern)
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all foundation dependencies (deps, TTY helper, RunTUI)
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved
