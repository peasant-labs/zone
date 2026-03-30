---
phase: 7
slug: environment-auth-forwarding
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-30
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard Go test runner |
| **Quick run command** | `go test ./internal/docker/... -run TestEnv -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/docker/... -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 07-01-01 | 01 | 1 | CFG-10 | unit | `go test ./internal/docker/... -run TestCollectEnv` | ❌ W0 | ⬜ pending |
| 07-01-02 | 01 | 1 | CFG-11 | unit | `go test ./internal/docker/... -run TestValidateRequired` | ❌ W0 | ⬜ pending |
| 07-01-03 | 01 | 1 | CFG-14 | unit | `go test ./internal/docker/... -run TestParseEnvFile` | ❌ W0 | ⬜ pending |
| 07-02-01 | 02 | 1 | CFG-12 | unit | `go test ./internal/docker/... -run TestSSH` | ❌ W0 | ⬜ pending |
| 07-02-02 | 02 | 1 | CFG-13 | unit | `go test ./internal/docker/... -run TestAuthConfig` | ❌ W0 | ⬜ pending |
| 07-02-03 | 02 | 1 | CFG-16 | unit | `go test ./internal/docker/... -run TestPort` | ❌ W0 | ⬜ pending |
| 07-02-04 | 02 | 1 | CFG-15 | unit | `go test ./internal/docker/... -run TestProxy` | ❌ W0 | ⬜ pending |
| 07-02-05 | 02 | 1 | CFG-17 | unit | N/A (already done Phase 6) | ✅ | ✅ green |
| 07-03-01 | 03 | 2 | CFG-18 | unit | `go test ./internal/docker/... -run TestHook` | ❌ W0 | ⬜ pending |
| 07-03-02 | 03 | 2 | CFG-10,11 | integration | `go test ./internal/docker/... -run TestLaunchWithEnv` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/docker/env_test.go` — stubs for CFG-10, CFG-11, CFG-14 tests
- [ ] `internal/docker/manager_test.go` — extend existing with SSH, auth, port, proxy tests

*Existing test infrastructure covers the framework needs — just need new test files.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| SSH socket forwarding on Linux | CFG-12 | Requires running SSH agent + Docker daemon | Start ssh-agent, set SSH_AUTH_SOCK, run zone launch, verify socket inside container |
| Auth config copy-on-start | CFG-13 | Requires running container with harness config | Create ~/.claude/ with test file, run zone launch, verify writable copy inside container |
| macOS SSH warning | CFG-12 | Requires macOS environment | Run on macOS with forward_ssh_agent=true, verify warning message |

*All other behaviors have automated unit test verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
