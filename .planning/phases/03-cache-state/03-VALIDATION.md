---
phase: 3
slug: cache-state
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-27
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `testing` stdlib + `testify` v1.10.0 |
| **Config file** | none — uses `go test ./...` |
| **Quick run command** | `go test ./tests/ -run "TestHash\|TestCache\|TestLock\|TestGitignore\|TestBuildLog" -v` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./tests/ -run "TestHash|TestCache|TestLock" -v`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 0 | CAC-01,02,03,04,05,06 | unit stubs | `go test ./tests/ -run TestHash -v` | ❌ W0 | ⬜ pending |
| 03-02-01 | 02 | 1 | CAC-01 | unit | `go test ./tests/ -run TestCache -v` | ❌ W0 | ⬜ pending |
| 03-02-02 | 02 | 1 | CAC-02 | unit | `go test ./tests/ -run TestHash -v` | ❌ W0 | ⬜ pending |
| 03-02-03 | 02 | 1 | CAC-03,04 | unit+integration | `go test ./tests/ -run "TestLock\|TestLockContention" -v` | ❌ W0 | ⬜ pending |
| 03-02-04 | 02 | 1 | CAC-05 | unit | `go test ./tests/ -run TestGitignore -v` | ❌ W0 | ⬜ pending |
| 03-02-05 | 02 | 1 | CAC-06 | unit | `go test ./tests/ -run TestBuildLog -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `tests/hash_test.go` — expand from stub: `TestHashStability`, `TestHashChangesOnConfigChange`, `TestHashChangesOnVersion`
- [ ] `tests/cache_test.go` — new file: `TestCacheEnsureDir`, `TestCacheAtomicWrite`, `TestCacheReadWrite`, `TestLockAcquireRelease`, `TestLockContention`, `TestGitignoreCreation`, `TestGitignoreIdempotent`, `TestBuildLogCreation`, `TestBuildLogHeader`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Concurrent `zone` commands produce exit code 5 | CAC-04 | Requires two processes contending on the same lock file | Run `zone launch &; zone launch` in same directory; second should exit 5 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
