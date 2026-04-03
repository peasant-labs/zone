---
phase: 10
slug: network-sandboxing
status: draft
nyquist_compliant: true
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
| **Config file** | none -- standard Go test toolchain |
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
| 10-01-01 | 01 | 1 | NET-04,NET-07,NET-09,NET-11 | unit | `go build ./... && go test ./internal/docker/... -count=1` | existing (manager_test.go extended) | pending |
| 10-01-02 | 01 | 1 | NET-12 | unit | `go test ./internal/network/... ./tests/ -run "TestMatcher\|TestCompile\|TestMatch" -v` | W0: tests/matcher_test.go | pending |
| 10-01-03 | 01 | 1 | NET-09,NET-11 | unit | `go test ./tests/ -run "TestNetwork\|TestPlatform" -v` | W0: tests/network_platform_test.go | pending |
| 10-02-01 | 02 | 2 | NET-01,NET-02,NET-12 | unit | `go test ./internal/network/... -run "TestBuildRuleSet\|TestNormalizeMode\|TestRulesEqual" -v` | W0: internal/network/rules_test.go | pending |
| 10-02-02 | 02 | 2 | NET-05,NET-06,NET-10 | unit | `go test ./internal/network/... -run "TestFirewall\|TestRule\|TestClean" -v` | W0: internal/network/firewall_test.go | pending |
| 10-03-01 | 03 | 3 | NET-04,NET-09,NET-11 | unit | `go build ./internal/docker/...` | existing (network.go extended) | pending |
| 10-03-02 | 03 | 3 | NET-01,NET-02,NET-03,NET-05,NET-06 | integration | `go build ./... && go test ./internal/docker/... -run "TestStaleRule\|TestManager" -v -count=1` | W0: manager_test.go extended | pending |

*Status: pending -- green -- red -- flaky*

---

## Wave 0 Requirements

- [ ] `tests/matcher_test.go` -- stubs for NET-12 hostname glob matching (Plan 01, Task 2)
- [ ] `tests/network_platform_test.go` -- stubs for NET-09, NET-11 platform detection (Plan 01, Task 3)
- [ ] `internal/network/rules_test.go` -- stubs for NET-01, NET-02, NET-12 rule evaluation (Plan 02, Task 1)
- [ ] `internal/network/firewall_test.go` -- stubs for NET-05, NET-06, NET-10 iptables generation/refresh/cache (Plan 02, Task 2)
- [ ] `internal/docker/manager_test.go` -- TestStaleRuleCleanupOnLaunch for stale rule detection (Plan 03, Task 2)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| iptables rules actually block traffic | NET-01, NET-02 | Requires root/sudo + running container + network access | Launch container with whitelist mode, attempt `curl` to blocked host from inside, verify connection refused |
| Stale rules cleaned on startup | NET-05 | Requires crash simulation + root iptables access | Insert fake zone-tagged rules, launch zone, verify rules removed |
| macOS warning message | NET-09 | Requires macOS host | Run `zone launch` with mode=whitelist on macOS, verify warning printed |
| Rootless Docker fallback | NET-11 | Requires rootless Docker installation | Run zone with rootless Docker, verify warning + fallback to mode=none |
| End-to-end network sandboxing (NET-04, NET-07, NET-08) | NET-04, NET-07, NET-08 | Requires running container with real Docker bridge network, root for iptables, and network access to test IPv6 bypass prevention | Launch container with whitelist mode on Linux, verify bridge network created, rules applied per container, IPv6 disabled on bridge |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
