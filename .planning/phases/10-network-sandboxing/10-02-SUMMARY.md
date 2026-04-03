---
phase: 10-network-sandboxing
plan: 02
subsystem: infra
tags: [iptables, network, firewall, dns, testing]
requires:
  - phase: 10-network-sandboxing
    provides: Platform detection, matcher compilation, and firewall error sentinels from plan 10-01
provides:
  - RuleSet generation from merged network config with deny-first whitelist behavior
  - Firewall apply/remove helpers for whitelist and blocklist iptables management
  - Periodic refresh diffing, stale-rule cleanup, and firewall.rules cache output
affects: [network-sandboxing, launch, stop, destroy]
tech-stack:
  added: []
  patterns: [Injected resolver and exec hooks for firewall testing, tagged FORWARD-chain iptables rules, cached human-readable firewall command output]
key-files:
  created: [internal/network/rules_test.go, internal/network/firewall_test.go]
  modified: [internal/network/rules.go, internal/network/firewall.go]
key-decisions:
  - "Whitelist default DROP uses the same zone-{hash} comment tag as other rules so cleanup and tagging stay uniform."
  - "Firewall refresh uses an injectable resolver plus a small refreshOnce helper to keep the 5-minute loop testable without sleeping in tests."
patterns-established:
  - "Network rule generation should normalize mode first and resolve hostnames through an injected resolver for deterministic tests."
  - "Firewall lifecycle code should isolate iptables execution behind ExecFunc and cache the rendered commands for inspection."
requirements-completed: [NET-01, NET-02, NET-03, NET-05, NET-06, NET-08, NET-10]
duration: 4 min
completed: 2026-04-03
---

# Phase 10 Plan 02: Network Sandboxing Summary

**Host-side iptables firewall generation with deny-first hostname resolution, tagged rule lifecycle management, and cached firewall command output.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-04-03T04:27:01Z
- **Completed:** 2026-04-03T04:31:32Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Implemented `BuildRuleSet` for whitelist, blocklist, and none modes with deny-first filtering and explicit whitelist-glob validation.
- Added `Firewall.Apply`, `Remove`, `CleanStaleRules`, and refresh logic around tagged `sudo iptables` calls and Docker DNS exceptions.
- Added focused tests covering rule generation, cache writing, stale cleanup, and refresh cancellation/reapply behavior.

## Task Commits

Each task was committed atomically:

1. **Task 1: Rule set generation from NetworkConfig** - `5889787` (test), `48a1361` (feat)
2. **Task 2: Firewall struct with Apply, Remove, StartRefresh, and iptables execution** - `3f732c6` (test), `b6dc346` (feat)

**Plan metadata:** Included in the final docs commit for summary/state/roadmap updates.

## Files Created/Modified
- `internal/network/rules.go` - builds resolved allow/deny IP sets from merged network config.
- `internal/network/rules_test.go` - locks whitelist, blocklist, deny-first, mode normalization, and equality behavior.
- `internal/network/firewall.go` - applies/removes tagged iptables rules, refreshes changed rule sets, and writes `firewall.rules`.
- `internal/network/firewall_test.go` - verifies whitelist/blocklist application, cleanup, stale detection, cache output, and refresh lifecycle.

## Decisions Made
- Reused a single `zone-{hash}` comment tag across all generated rules, including the whitelist default DROP, so removal and stale detection can treat every rule uniformly.
- Added an internal `refreshOnce` helper and injectable resolver hook so refresh behavior stays deterministic under unit test while the production loop still runs every five minutes.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- `go test ./... -count=1` still fails in unrelated dirty-worktree test `tests/cli_dx_test.go:TestUnknownKeysRemediationHintOnStderr` (expected exit code 2, got 1); this remains deferred per `.planning/phases/10-network-sandboxing/deferred-items.md`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Plan 10-03 can now wire these firewall helpers into manager launch/stop/destroy flows.
- The unrelated CLI DX regression is still outside this plan's scope and should stay isolated from network sandboxing integration work.

## Self-Check: PASSED
