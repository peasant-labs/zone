---
phase: 10-network-sandboxing
plan: 05
subsystem: infra
tags: [iptables, network, firewall, glob, hostname-matching, testing]
requires:
  - phase: 10-network-sandboxing
    provides: RuleSet generation, Firewall struct with Apply/Remove/StartRefresh from plan 10-02
provides:
  - BuildRuleSet storing glob patterns in DenyGlobs/AllowGlobs instead of erroring or skipping
  - Deny-before-allow glob evaluation via MatchAny in whitelist mode
  - Warning emission for unresolvable glob patterns (not hard errors)
  - Refresh loop glob evaluation via BuildRuleSet re-invocation with io.Discard suppression
  - RulesEqual glob-aware equality comparison
affects: [network-sandboxing, launch, stop]
tech-stack:
  added: []
  patterns: [warnWriter io.Writer package variable for test-overridable warning output, io.Discard for suppressing periodic-refresh warnings]
key-files:
  created: []
  modified:
    - internal/network/rules.go
    - internal/network/rules_test.go
    - internal/network/firewall.go
    - internal/network/firewall_test.go
key-decisions:
  - "Downgraded whitelist allow glob from hard error to warning + storage — the glob is stored in AllowGlobs for future refresh matching rather than blocking rule build."
  - "Deny globs in whitelist mode use MatchAny deny-before-allow evaluation at build time (no warning needed — filtering works via existing deny pattern compilation)."
  - "refreshOnce suppresses BuildRuleSet warnings with io.Discard to avoid spam during the 5-minute periodic refresh loop."
  - "warnWriter package variable (not passed as parameter) keeps the function signature stable while allowing test capture."
patterns-established:
  - "warnWriter io.Writer var pattern: use package-level writer overridable in tests, avoids adding parameters to BuildRuleSet."
  - "Glob storage in RuleSet enables refresh-time re-evaluation without re-parsing config each time."

requirements-completed: [NET-12]

duration: 12min
completed: 2026-04-03
---

# Phase 10 Plan 05: Glob Enforcement Summary

**BuildRuleSet now stores hostname glob patterns in DenyGlobs/AllowGlobs with warning emission instead of erroring or silently skipping, enabling end-to-end glob-based network rule enforcement in both whitelist and blocklist modes.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-04-03T05:00:00Z
- **Completed:** 2026-04-03T05:12:00Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Fixed whitelist allow glob handling: stores in `AllowGlobs` + warns instead of returning hard error.
- Fixed blocklist deny glob handling: stores in `DenyGlobs` + warns instead of silently skipping.
- Deny globs in whitelist mode filter allow entries via `MatchAny` at build time (deny-before-allow).
- `RulesEqual` updated to compare `DenyGlobs` and `AllowGlobs` for change detection.
- `refreshOnce` suppresses warning output via `io.Discard` to prevent stderr spam during periodic refresh.
- Added comprehensive tests: glob storage, warning emission, deny-before-allow filtering, refresh evaluation, glob equality.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add glob fields to RuleSet and fix BuildRuleSet** - `c480624` (test - RED), `ec3f04a` (feat - GREEN)
2. **Task 2: Wire glob patterns into firewall refresh evaluation** - `8e2e4ff` (feat + test - GREEN, behavior already satisfied by Task 1)

## Files Created/Modified
- `internal/network/rules.go` - Added `DenyGlobs`, `AllowGlobs`, `Warnings` to `RuleSet`; `warnWriter` var; rewrote allow/deny glob blocks; updated `RulesEqual`.
- `internal/network/rules_test.go` - Replaced error-expecting glob test with warning-checking tests; added 4 new tests for glob behaviors.
- `internal/network/firewall.go` - Added `io.Discard` warning suppression in `refreshOnce`.
- `internal/network/firewall_test.go` - Added `TestRefreshGlobDenyMatch`, `TestRefreshAllowGlobStored`, `containsArg` helper.

## Decisions Made
- Used `var warnWriter io.Writer = os.Stderr` pattern (not function parameter) to keep `BuildRuleSet` signature stable while enabling test capture.
- Deny globs in whitelist mode don't need an explicit warning because they work correctly at build time via `MatchAny` — only blocklist deny globs need the warning since they cannot create direct iptables DROP rules.
- `refreshOnce` suppresses warnings with `io.Discard` rather than checking a flag, keeping the suppression co-located with the refresh behavior.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- Pre-existing test failures in `tests/cli_dx_test.go` (`TestExitCode6OnNoContainer`, `TestUnknownKeysRemediationHintOnStderr`) remain deferred per `deferred-items.md`. These are unrelated to network sandboxing and were present before this plan.
- Task 2 tests compiled green immediately (no RED state) because Task 1's `BuildRuleSet` rewrite already satisfied the refresh behavior. The `refreshOnce` warning suppression was still added as a production improvement.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- NET-12 (hostname glob patterns) is now complete.
- Plans 10-03 and 10-04 (manager integration, cleanup) can use the full glob-aware RuleSet.
- The full test suite passes for all network and docker packages.

## Self-Check: PASSED
