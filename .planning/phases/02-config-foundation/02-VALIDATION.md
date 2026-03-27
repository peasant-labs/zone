---
phase: 2
slug: config-foundation
status: draft
nyquist_compliant: true
wave_0_complete: true
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
| 02-01-01 | 01 | 1 | CFG-01 | unit | `go test ./tests/ -run TestMinimalConfig -v` | ✅ | ✅ green |
| 02-01-02 | 01 | 1 | CFG-02 | unit | `go test ./tests/ -run TestGlobalConfigLoad -v` | ❌ W0 | ⬜ pending |
| 02-01-03 | 01 | 1 | CFG-03 | unit | `go test ./tests/ -run TestScalarOverride -v` | ✅ | ✅ green |
| 02-01-04 | 01 | 1 | CFG-04 | unit | `go test ./tests/ -run TestListMerge -v` | ✅ | ✅ green |
| 02-01-05 | 01 | 1 | CFG-09 | unit | `go test ./tests/ -run TestConfigVersion -v` | ✅ | ✅ green |
| 02-02-01 | 02 | 1 | CFG-05 | unit | `go test ./tests/ -run TestUnknownKeySuggestion -v` | ✅ | ✅ green |
| 02-02-02 | 02 | 1 | CFG-06 | unit | `go test ./tests/ -run TestDangerousMount -v` | ✅ | ✅ green |
| 02-02-03 | 02 | 1 | CFG-19 | unit | `go test ./tests/ -run TestMountReadOnly -v` | ✅ | ✅ green |
| 02-03-01 | 03 | 2 | CFG-07 | integration | `go test ./tests/ -run TestConfigAnnotatedOutput -v` | ✅ | ✅ green |
| 02-03-02 | 03 | 2 | CFG-08 | unit | `go test ./tests/ -run TestConfigJSON -v` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `tests/config_merge_test.go` — test stubs for CFG-01, CFG-02, CFG-03, CFG-04, CFG-07, CFG-08, CFG-09
- [x] `tests/validate_test.go` — test stubs for CFG-05, CFG-06, CFG-19
- [x] `go get github.com/BurntSushi/toml@v1.6.0` — add TOML parser dependency
- [x] `go get github.com/agnivade/levenshtein@v1.2.1` — add edit-distance dependency

*Existing stub files exist but contain only `package tests`.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 5s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** signed off 2026-03-27
