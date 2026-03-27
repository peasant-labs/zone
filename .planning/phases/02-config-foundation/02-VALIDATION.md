---
phase: 2
slug: config-foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-27
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard testing (`go test`) |
| **Config file** | None needed — uses Go test flags |
| **Quick run command** | `go test ./tests/ -run TestConfig -v` |
| **Full suite command** | `go test ./tests/ -v -race` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./tests/ -run TestConfig -v`
- **After every plan wave:** Run `go test ./tests/ -v -race`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | CFG-01 | unit | `go test ./tests/ -run TestMinimalConfig -v` | ❌ W0 | ⬜ pending |
| 02-01-02 | 01 | 1 | CFG-02 | unit | `go test ./tests/ -run TestGlobalConfigLoad -v` | ❌ W0 | ⬜ pending |
| 02-01-03 | 01 | 1 | CFG-03 | unit | `go test ./tests/ -run TestScalarOverride -v` | ❌ W0 | ⬜ pending |
| 02-01-04 | 01 | 1 | CFG-04 | unit | `go test ./tests/ -run TestListMerge -v` | ❌ W0 | ⬜ pending |
| 02-01-05 | 01 | 1 | CFG-09 | unit | `go test ./tests/ -run TestConfigVersion -v` | ❌ W0 | ⬜ pending |
| 02-02-01 | 02 | 1 | CFG-05 | unit | `go test ./tests/ -run TestUnknownKeySuggestion -v` | ❌ W0 | ⬜ pending |
| 02-02-02 | 02 | 1 | CFG-06 | unit | `go test ./tests/ -run TestDangerousMount -v` | ❌ W0 | ⬜ pending |
| 02-02-03 | 02 | 1 | CFG-19 | unit | `go test ./tests/ -run TestMountReadOnly -v` | ❌ W0 | ⬜ pending |
| 02-03-01 | 03 | 2 | CFG-07 | integration | `go test ./tests/ -run TestConfigAnnotatedOutput -v` | ❌ W0 | ⬜ pending |
| 02-03-02 | 03 | 2 | CFG-08 | unit | `go test ./tests/ -run TestConfigJSON -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `tests/config_merge_test.go` — test stubs for CFG-01, CFG-02, CFG-03, CFG-04, CFG-07, CFG-08, CFG-09
- [ ] `tests/validate_test.go` — test stubs for CFG-05, CFG-06, CFG-19
- [ ] `go get github.com/BurntSushi/toml@v1.6.0` — add TOML parser dependency
- [ ] `go get github.com/agnivade/levenshtein@v1.2.1` — add edit-distance dependency

*Existing stub files exist but contain only `package tests`.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
