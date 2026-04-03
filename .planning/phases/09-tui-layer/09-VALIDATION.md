---
phase: 9
slug: tui-layer
status: draft
nyquist_compliant: false
wave_0_complete: false
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
| **Quick run command** | `go test ./internal/tui/ -v -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/tui/ -v -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 09-01-01 | 01 | 1 | TUI-05 | unit | `go test ./internal/tui/ -run TestIsTTY -v` | ❌ W0 | ⬜ pending |
| 09-01-02 | 01 | 1 | TUI-01 | unit | `go test ./internal/tui/ -run TestInitWizard -v` | ❌ W0 | ⬜ pending |
| 09-01-03 | 01 | 1 | TUI-02 | unit | `go test ./internal/tui/ -run TestBuildProgress -v` | ❌ W0 | ⬜ pending |
| 09-01-04 | 01 | 1 | TUI-03 | unit | `go test ./internal/tui/ -run TestStatusView -v` | ❌ W0 | ⬜ pending |
| 09-01-05 | 01 | 1 | TUI-04 | unit | `go test ./internal/tui/ -run TestLogViewer -v` | ❌ W0 | ⬜ pending |
| 09-02-01 | 02 | 2 | TUI-06 | integration | `go test ./tests/ -run TestPlainFlag -v` | ❌ W0 | ⬜ pending |
| 09-02-02 | 02 | 2 | TUI-07 | integration | `go test ./tests/ -run TestInitNoHarnessNonTTY -v` | existing | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/tui/tty.go` — IsTTY() and IsOutputTTY() helpers (TUI-05, TUI-06)
- [ ] `internal/tui/init_wizard_test.go` — stubs for TUI-01
- [ ] `internal/tui/build_progress_test.go` — stubs for TUI-02
- [ ] `internal/tui/status_view_test.go` — stubs for TUI-03
- [ ] `internal/tui/log_viewer_test.go` — stubs for TUI-04
- [ ] `tests/tui_integration_test.go` — covers TUI-06, TUI-07 (non-TTY path)
- [ ] `internal/docker/build_progress.go` — `BuildWithProgress()` method (needed by TUI-02)
- [ ] Framework install: `go get charm.land/bubbletea/v2@v2.0.2 charm.land/bubbles/v2@v2.1.0 charm.land/lipgloss/v2@v2.0.2 golang.org/x/term@v0.41.0`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Init wizard renders correctly in TTY | TUI-01 | Requires real terminal to verify visual rendering | Run `zone init` in a TTY, verify list navigation and config preview |
| Build progress scrolling | TUI-02 | Requires real Docker build output stream | Run `zone launch --rebuild` and verify progress viewport |
| Terminal state restored after panic | TUI-05 | Requires simulating process crash | Force-kill zone during TUI, verify terminal is not garbled |
| Status view hotkeys (r restart, s stop) | TUI-03 | Requires running container + real terminal | Run `zone status`, press r/s, verify container state changes |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
