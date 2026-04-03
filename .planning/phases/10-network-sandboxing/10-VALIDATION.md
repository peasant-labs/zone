---
phase: 10
slug: network-sandboxing
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-03
---

# Phase 10 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard Go test toolchain |
| **Quick run command** | `go test ./internal/network/... ./internal/docker/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/network/... ./internal/docker/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 10-01-01 | 01 | 1 | NET-12 | unit | `go test ./internal/network/ -run TestMatcher` | ❌ W0 | ⬜ pending |
| 10-01-02 | 01 | 1 | NET-01,NET-02 | unit | `go test ./internal/network/ -run TestRules` | ❌ W0 | ⬜ pending |
| 10-02-01 | 02 | 1 | NET-05,NET-10 | unit | `go test ./internal/network/ -run TestFirewall` | ❌ W0 | ⬜ pending |
| 10-02-02 | 02 | 1 | NET-06 | unit | `go test ./internal/network/ -run TestRefresh` | ❌ W0 | ⬜ pending |
| 10-03-01 | 03 | 2 | NET-09,NET-11 | unit | `go test ./internal/docker/ -run TestPlatform` | ❌ W0 | ⬜ pending |
| 10-03-02 | 03 | 2 | NET-01,NET-02,NET-03 | integration | `go test ./internal/docker/ -run TestFirewallIntegration` | ❌ W0 | ⬜ pending |
| 10-04-01 | 04 | 3 | NET-04,NET-07,NET-08 | integration | `go test ./tests/ -run TestNetwork` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/network/matcher_test.go` — stubs for NET-12 hostname glob matching
- [ ] `internal/network/rules_test.go` — stubs for NET-01, NET-02 rule evaluation
- [ ] `internal/network/firewall_test.go` — stubs for NET-05, NET-06, NET-10 iptables generation/refresh/cache
- [ ] `internal/docker/platform_test.go` — stubs for NET-09, NET-11 platform detection
- [ ] `tests/network_integration_test.go` — stubs for NET-04, NET-07, NET-08 end-to-end network sandboxing

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| iptables rules actually block traffic | NET-01, NET-02 | Requires root/sudo + running container + network access | Launch container with whitelist mode, attempt `curl` to blocked host from inside, verify connection refused |
| Stale rules cleaned on startup | NET-05 | Requires crash simulation + root iptables access | Insert fake zone-tagged rules, launch zone, verify rules removed |
| macOS warning message | NET-09 | Requires macOS host | Run `zone launch` with mode=whitelist on macOS, verify warning printed |
| Rootless Docker fallback | NET-11 | Requires rootless Docker installation | Run zone with rootless Docker, verify warning + fallback to mode=none |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
