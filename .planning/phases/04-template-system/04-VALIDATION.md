---
phase: 4
slug: template-system
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-29
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `testing` stdlib + `github.com/stretchr/testify` v1.10.0 |
| **Config file** | none — `go test ./tests/...` |
| **Quick run command** | `go test ./tests/... -run TestContainerName -v` |
| **Full suite command** | `go test ./tests/... -v` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./tests/... -run TestContainerName -v`
- **After every plan wave:** Run `go test ./tests/... -v`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | DOC-01 | unit | `go test ./tests/... -run TestRenderDockerfile -v` | ❌ W0 | ⬜ pending |
| 04-01-02 | 01 | 1 | DOC-02 | unit | `go test ./tests/... -run TestRenderEntrypoint -v` | ❌ W0 | ⬜ pending |
| 04-01-03 | 01 | 1 | DOC-03 | unit | `go test ./tests/... -run TestRenderShellRC -v` | ❌ W0 | ⬜ pending |
| 04-01-04 | 01 | 1 | DOC-04 | unit | `go test ./tests/... -run TestDockerfileNonRootUser -v` | ❌ W0 | ⬜ pending |
| 04-01-05 | 01 | 1 | DOC-05 | unit | `go test ./tests/... -run TestDockerfileSudoScope -v` | ❌ W0 | ⬜ pending |
| 04-01-06 | 01 | 1 | DOC-06 | unit | `go test ./tests/... -run TestContainerSecurityFlags -v` | ❌ W0 | ⬜ pending |
| 04-01-07 | 01 | 1 | DOC-07 | unit | `go test ./tests/... -run TestContainerName -v` | ❌ W0 | ⬜ pending |
| 04-01-08 | 01 | 1 | DOC-13 | unit | `go test ./tests/... -run TestEntrypointGitSafeDir -v` | ❌ W0 | ⬜ pending |
| 04-01-09 | 01 | 1 | DOC-14 | unit | `go test ./tests/... -run TestGitIdentityDetection -v` | ❌ W0 | ⬜ pending |
| 04-01-10 | 01 | 1 | DOC-15 | unit | `go test ./tests/... -run TestMacOSUsername -v` | ❌ W0 | ⬜ pending |
| 04-01-11 | 01 | 1 | DOC-16 | unit | `go test ./tests/... -run TestDockerfileRootUID -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `tests/template_render_test.go` — stubs for DOC-01, DOC-02, DOC-03, DOC-04, DOC-05, DOC-13, DOC-16
- [ ] `tests/naming_test.go` — stubs for DOC-07 (file exists but is empty stub)
- [ ] `tests/git_config_test.go` — stubs for DOC-14
- [ ] `tests/platform_test.go` — stubs for DOC-15
- [ ] `tests/security_flags_test.go` — stubs for DOC-06

*Existing infrastructure covers framework — `go test ./tests/...` already works.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Dockerfile builds with `docker build` | DOC-01 SC1 | Requires Docker daemon | `docker build -f .zone/Dockerfile -t zone-test .` |
| Entrypoint PID 1 signal forwarding | DOC-02 SC2 | Requires running container | Run container, send SIGTERM, verify clean exit |

*All other phase behaviors have automated verification via unit tests.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
