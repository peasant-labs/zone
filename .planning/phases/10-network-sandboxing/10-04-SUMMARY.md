---
phase: 10-network-sandboxing
plan: "04"
subsystem: infra
tags: [iptables, firewall, docker, cleanup, network-sandboxing]

# Dependency graph
requires:
  - phase: 10-network-sandboxing plan 03
    provides: firewall lifecycle wired into docker manager (m.firewall, firewallCancel fields, Stop firewall block)

provides:
  - Exported RemoveRulesByHash standalone function for hash-based iptables rule removal
  - reconstructFirewallForCleanup method reconstructs Firewall from cache/naming in fresh processes
  - Manager.Stop removes iptables rules even when m.firewall is nil (fresh-process cleanup)
  - zone clean invokes firewall cleanup before deleting .zone/ on Linux
  - NET-05 durable tagged cleanup fully satisfied

affects: [10-network-sandboxing-05, 10-VALIDATION]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Fresh-process firewall reconstruction: derive container hash from ContainerName(repoDir), create Firewall with nil execFn (defaults to sudo iptables)"
    - "Best-effort cleanup: firewall errors in Stop and clean are Fprintf(Stderr) warnings, not fatal returns"
    - "Linux-only guard: runtime.GOOS == linux gates all iptables calls in cmd layer"

key-files:
  created: []
  modified:
    - internal/network/firewall.go
    - internal/network/firewall_test.go
    - internal/docker/manager.go
    - internal/docker/manager_test.go
    - cmd/clean.go

key-decisions:
  - "RemoveRulesByHash exported as thin wrapper over removeRulesForHash — preserves the unexported implementation, adds stable public API"
  - "reconstructFirewallForCleanup returns nil for mode=none or empty mode — avoids unnecessary sudo invocations for repos that never had firewall rules"
  - "containerHash derived from last 16 chars of ContainerName(repoDir) — matches the 16-char sha256 shortHash format used throughout naming.go"
  - "cmd/clean.go uses nil execFn for RemoveRulesByHash — falls back to DefaultExecFunc (sudo iptables), consistent with Firewall.Remove behavior"

patterns-established:
  - "Fresh-process cleanup pattern: check m.firewall != nil, else call reconstructFirewallForCleanup; use fw local variable to unify both paths"
  - "Best-effort firewall operations: all iptables calls outside Apply are warn-and-continue (never block the primary operation)"

requirements-completed: [NET-05]

# Metrics
duration: 12min
completed: 2026-04-03
---

# Phase 10 Plan 04: Durable Firewall Cleanup Summary

**Fresh-process iptables cleanup via reconstructFirewallForCleanup and exported RemoveRulesByHash — zone stop, zone destroy, and zone clean now remove tagged rules even without prior Launch in the same process**

## Performance

- **Duration:** 12 min
- **Started:** 2026-04-03T00:00:00Z
- **Completed:** 2026-04-03T00:12:00Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Exported `RemoveRulesByHash` as a stable public API for standalone iptables cleanup by container hash
- Added `reconstructFirewallForCleanup` to Manager, enabling Stop to clean up rules in fresh CLI processes where m.firewall is nil
- Wired firewall cleanup into `zone clean` before `.zone/` deletion, using best-effort Linux-only guard
- All new behavior covered by TDD tests (TestRemoveRulesByHash, TestStop_FreshProcessFirewallCleanup, TestStop_FreshProcessNoFirewallWhenModeNone)

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Failing tests for fresh-process cleanup** - `dc2d6c6` (test)
2. **Task 1 GREEN: Export RemoveRulesByHash and add reconstructFirewallForCleanup** - `eb59c65` (feat)
3. **Task 2: Add firewall cleanup to zone clean command** - `2477c4f` (feat)

_Note: TDD tasks have RED commit followed by GREEN implementation commit_

## Files Created/Modified
- `internal/network/firewall.go` - Added exported `RemoveRulesByHash` wrapper above `CleanStaleRules`
- `internal/network/firewall_test.go` - Added `TestRemoveRulesByHash` with matching and no-match subtests
- `internal/docker/manager.go` - Added `reconstructFirewallForCleanup` method; updated Stop to use `fw` variable pattern
- `internal/docker/manager_test.go` - Added `TestStop_FreshProcessFirewallCleanup` and `TestStop_FreshProcessNoFirewallWhenModeNone`
- `cmd/clean.go` - Added `runtime`, `network`, and `docker` imports; iptables cleanup block before `c.Clean()`

## Decisions Made
- `RemoveRulesByHash` exported as thin wrapper over `removeRulesForHash` — no duplication, just API surface
- `reconstructFirewallForCleanup` returns nil for mode="" or mode="none" — avoids sudo calls for repos without firewall config
- containerHash taken from last 16 chars of `ContainerName(repoDir)` to match the 16-char sha256 shortHash
- `cmd/clean.go` uses `nil` execFn → defaults to `sudo iptables` in `DefaultExecFunc`

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

Pre-existing test failures in `tests/cli_dx_test.go` (`TestExitCode6OnNoContainer` and `TestUnknownKeysRemediationHintOnStderr`) were confirmed to exist in the main branch before this plan's changes. They are out of scope for this plan.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Self-Check: PASSED

All files present and all commits verified.

## Next Phase Readiness
- NET-05 (tagged cleanup) is fully satisfied
- All cleanup paths (Stop, Destroy, clean) now remove iptables rules durably
- Phase 10-05 can proceed with glob enforcement in network config validation
- Pre-existing test failures in cli_dx_test.go need attention in a separate plan

---
*Phase: 10-network-sandboxing*
*Completed: 2026-04-03*
