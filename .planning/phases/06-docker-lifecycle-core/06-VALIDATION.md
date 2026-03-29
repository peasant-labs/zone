---
phase: 6
slug: docker-lifecycle-core
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-29
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing stdlib + testify v1.10.0 |
| **Config file** | none — `go test ./...` discovers all packages |
| **Quick run command** | `go test ./... -short -timeout 60s` |
| **Full suite command** | `go test ./... -timeout 120s` |
| **Estimated runtime** | ~15 seconds (unit), ~60 seconds (integration) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./... -short -timeout 60s`
- **After every plan wave:** Run `go test ./... -timeout 120s`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 06-01-01 | 01 | 1 | DOC-11 | unit (mock) | `go test ./internal/docker/ -run TestNewManager -v` | ❌ W0 | ⬜ pending |
| 06-01-02 | 01 | 1 | DOC-12 | unit (mock) | `go test ./internal/docker/ -run TestBuild -v` | ❌ W0 | ⬜ pending |
| 06-01-03 | 01 | 1 | DOC-09 | unit (mock) | `go test ./internal/docker/ -run TestLaunchStateMachine -v` | ❌ W0 | ⬜ pending |
| 06-01-04 | 01 | 1 | DOC-10 | unit (mock) | `go test ./internal/docker/ -run TestConfigHashDetection -v` | ❌ W0 | ⬜ pending |
| 06-01-05 | 01 | 1 | CFG-20 | unit (mock) | `go test ./internal/docker/ -run TestHomeVolume -v` | ❌ W0 | ⬜ pending |
| 06-02-01 | 02 | 1 | CLI-10 | unit (mock) | `go test ./internal/docker/ -run TestStopClearsCache -v` | ❌ W0 | ⬜ pending |
| 06-02-02 | 02 | 1 | CLI-16 | unit (mock) | `go test ./internal/docker/ -run TestDestroyVsStop -v` | ❌ W0 | ⬜ pending |
| 06-02-03 | 02 | 1 | DOC-08 | unit | `go test ./tests/ -run TestContainerLabels -v` | ✅ | ⬜ pending |
| 06-03-01 | 03 | 2 | CLI-03 | integration | `go test ./tests/ -run TestLaunchCommand -v` | ❌ W0 | ⬜ pending |
| 06-03-02 | 03 | 2 | CLI-04 | integration | `go test ./tests/ -run TestLaunchHeadless -v` | ❌ W0 | ⬜ pending |
| 06-03-03 | 03 | 2 | CLI-05 | integration | `go test ./tests/ -run TestZeroConfigLaunch -v` | ❌ W0 | ⬜ pending |
| 06-03-04 | 03 | 2 | CLI-06 thru CLI-11 | integration | `go test ./tests/ -run TestLifecycleCommands -v` | ❌ W0 | ⬜ pending |
| 06-03-05 | 03 | 2 | CLI-15 | integration | `go test ./tests/ -run TestCleanWithImage -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/docker/client_interface.go` — Mock-able Docker client interface for unit testing
- [ ] `internal/docker/manager_test.go` — Unit tests for DOC-09, DOC-10, DOC-11, DOC-12, CFG-20, CLI-10, CLI-16
- [ ] `tests/manager_test.go` — Integration tests (guarded with build tag or `testing.Short()`)
- [ ] Framework already installed: `github.com/stretchr/testify v1.10.0`

*Existing infrastructure covers test framework. Docker SDK dependency and mock interface need creation.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Interactive TTY attach (resize, signal forwarding) | CLI-03, CLI-06 | Requires live terminal interaction | Run `zone launch`, verify shell works, Ctrl+C detaches cleanly |
| Docker Desktop notification on macOS | DOC-11 | Platform-specific Docker Desktop behavior | Run on macOS, verify error message if Docker not started |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
