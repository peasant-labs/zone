---
phase: 8
slug: cli-commands-dx
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-30
---

# Phase 8 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard Go test runner |
| **Quick run command** | `go test ./cmd/... ./internal/docker/... -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/... ./internal/docker/... -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 08-01-01 | 01 | 1 | DX-01, DX-02 | unit | `go test ./cmd/... -run TestExitCode` | ❌ W0 | ⬜ pending |
| 08-01-02 | 01 | 1 | DX-04, DX-05 | unit | `go test ./cmd/... -run TestSignal` | ❌ W0 | ⬜ pending |
| 08-02-01 | 02 | 1 | CLI-01, CLI-02 | unit | `go test ./cmd/... -run TestInit` | ❌ W0 | ⬜ pending |
| 08-02-02 | 02 | 1 | CLI-12 | unit | `go test ./internal/docker/... -run TestList` | ❌ W0 | ⬜ pending |
| 08-02-03 | 02 | 1 | CLI-13, CLI-14 | unit | `go test ./internal/docker/... -run TestLogs` | ❌ W0 | ⬜ pending |
| 08-02-04 | 02 | 1 | CLI-17 | unit | `go test ./internal/docker/... -run TestStatus` | ❌ W0 | ⬜ pending |
| 08-03-01 | 03 | 2 | DX-03 | unit | `go test ./cmd/... -run TestJSON` | ❌ W0 | ⬜ pending |
| 08-03-02 | 03 | 2 | DX-08, DX-09 | unit | `go test ./cmd/... -run TestAlias\|TestHelp` | ❌ W0 | ⬜ pending |
| 08-03-03 | 03 | 2 | CLI-20, CLI-21 | unit | `go test ./cmd/... -run TestGlobalFlags\|TestPort` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Test file stubs for new command tests

*Existing test infrastructure covers framework needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Ctrl+C sends SIGINT to harness | DX-04 | Requires running Docker + TTY | Launch zone, press Ctrl+C, verify container still alive via `docker ps` |
| --follow streams live logs | CLI-13 | Requires running container producing output | Run `zone logs -f` against live container |
| JSON output is valid | DX-03 | Parseable JSON check | Run `zone status --json \| jq .` and `zone ls --json \| jq .` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
