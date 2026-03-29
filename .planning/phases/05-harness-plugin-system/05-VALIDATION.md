---
phase: 5
slug: harness-plugin-system
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-29
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (testify v1.10.0) |
| **Config file** | none — `go test ./...` discovers all packages |
| **Quick run command** | `go test ./tests/ -run TestHarness -v` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./tests/ -run TestHarness -v`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 05-01-01 | 01 | 1 | HAR-01 | unit | `go test ./tests/ -run TestHarnessInterface -v` | ❌ W0 | ⬜ pending |
| 05-01-02 | 01 | 1 | HAR-02 | unit | `go test ./tests/ -run TestBaseHarnessDefaults -v` | ❌ W0 | ⬜ pending |
| 05-01-03 | 01 | 1 | HAR-03 | unit | `go test ./tests/ -run TestHarnessRegistry -v` | ❌ W0 | ⬜ pending |
| 05-02-01 | 02 | 1 | HAR-04 | unit | `go test ./tests/ -run TestClaudeCode -v` | ❌ W0 | ⬜ pending |
| 05-02-02 | 02 | 1 | HAR-04 | unit | `go test ./tests/ -run TestClaudeCodeInstallVersioned -v` | ❌ W0 | ⬜ pending |
| 05-02-03 | 02 | 1 | HAR-09 | unit | `go test ./tests/ -run TestSkipPermissionsDefault -v` | ❌ W0 | ⬜ pending |
| 05-02-04 | 02 | 1 | HAR-10 | unit | `go test ./tests/ -run TestPromptFlag -v` | ❌ W0 | ⬜ pending |
| 05-03-01 | 03 | 1 | HAR-05 | unit | `go test ./tests/ -run TestStubHarnessValidate -v` | ❌ W0 | ⬜ pending |
| 05-03-02 | 03 | 1 | HAR-06 | unit | `go test ./tests/ -run TestCustomHarness -v` | ❌ W0 | ⬜ pending |
| 05-03-03 | 03 | 1 | HAR-07 | unit | `go test ./tests/ -run TestCrossHarnessKeyValidation -v` | ❌ W0 | ⬜ pending |
| 05-04-01 | 04 | 2 | Bridge | unit | `go test ./tests/ -run TestBuildDockerfileData -v` | ❌ W0 | ⬜ pending |
| 05-04-02 | 04 | 2 | Bridge | unit | `go test ./tests/ -run TestBuildEntrypointData -v` | ❌ W0 | ⬜ pending |
| 05-04-03 | 04 | 2 | Bridge | unit | `go test ./tests/ -run TestBuildShellRCData -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `tests/harness_validate_test.go` — fill existing stub; covers HAR-05, HAR-06, HAR-07, HAR-09, HAR-10
- [ ] `tests/harness_registry_test.go` — HAR-01, HAR-02, HAR-03 (interface + registry)
- [ ] `tests/harness_claude_code_test.go` — HAR-04 (ClaudeCode full method coverage)
- [ ] `tests/harness_bridge_test.go` — BuildDockerfileData, BuildEntrypointData, BuildShellRCData integration

*Existing infrastructure covers test framework. Only test files need creation.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `zone launch` with claude-code starts container | HAR-04 SC1 | Requires Docker daemon | Full e2e in Phase 6 |
| Prompt flag passes through to harness | HAR-10 SC5 | Requires running container | Full e2e in Phase 6 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
