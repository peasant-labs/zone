---
phase: 03-cache-state
plan: 03
subsystem: cache
tags: [go, error-handling, exit-codes, lock-contention, testing]

# Dependency graph
requires:
  - phase: 03-cache-state-02
    provides: ErrLockContention sentinel in internal/cache/lock.go; flock-based Lock.Acquire() returning wrapped error

provides:
  - Exit code 5 translation for ErrLockContention in main.go (errors.Is check before generic os.Exit(1))
  - TestExitCodeLockContentionSentinel: verifies errors.Is detects wrapped ErrLockContention
  - TestExitCodeGenericError: confirms generic errors do not match ErrLockContention

affects: [06-launch, phase-integration-tests]

# Tech tracking
tech-stack:
  added: []
  patterns: [errors.Is chain traversal for exit code mapping in main.go]

key-files:
  created: []
  modified:
    - main.go
    - tests/cache_test.go

key-decisions:
  - "errors.Is in main.go traverses wrapped error chain — ErrLockContention wrapped via %w in Acquire() is correctly detected"
  - "Exit code 5 check placed before generic os.Exit(1) — ordering is critical for correct mapping"
  - "Full binary e2e test deferred to Phase 6 when zone launch calls Lock.Acquire(); sentinel test covers the mapping logic now"

patterns-established:
  - "Exit code mapping pattern: errors.Is check for specific sentinel before generic fallback in main.go"

requirements-completed: [CAC-04]

# Metrics
duration: 3min
completed: 2026-03-27
---

# Phase 03 Plan 03: Exit Code 5 for Lock Contention Summary

**Exit code 5 wired in main.go via errors.Is(err, cache.ErrLockContention) before fallback os.Exit(1), with two sentinel tests confirming the mapping logic**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-27T19:30:00Z
- **Completed:** 2026-03-27T19:33:00Z
- **Tasks:** 1
- **Files modified:** 2

## Accomplishments

- main.go imports `errors` and `internal/cache`, checks `errors.Is(err, cache.ErrLockContention)` and calls `os.Exit(5)` before the generic `os.Exit(1)` fallback
- `TestExitCodeLockContentionSentinel` confirms that errors returned by `Lock.Acquire()` under contention satisfy `errors.Is(err, cache.ErrLockContention)` — the exact precondition for exit code 5
- `TestExitCodeGenericError` confirms that unrelated errors do not match `ErrLockContention`, preventing false positives
- All 17 tests pass (15 existing + 2 new); no regressions

## Task Commits

Each task was committed atomically:

1. **Task 1: Add exit code 5 translation for ErrLockContention in main.go and test** - `397542a` (feat)

**Plan metadata:** (pending docs commit)

## Files Created/Modified

- `/workspace/zone/main.go` - Added `errors` and `internal/cache` imports; inserted `errors.Is(err, cache.ErrLockContention)` check producing `os.Exit(5)` before generic `os.Exit(1)`
- `/workspace/zone/tests/cache_test.go` - Added `TestExitCodeLockContentionSentinel` and `TestExitCodeGenericError` functions

## Decisions Made

- `errors.Is` correctly traverses the wrapping chain from `Lock.Acquire()` (`fmt.Errorf("%w (PID %d)", ErrLockContention, pid)`) — no custom `Unwrap` needed
- Full binary integration test (running the compiled binary and asserting exit code 5) deferred to Phase 6 when `zone launch` actually calls `Lock.Acquire()`; the sentinel test covers the logic correctness now
- Exit code 5 check is ordered before `os.Exit(1)` — this ordering is the safety-critical part of the implementation

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Self-Check: PASSED

All files present and commit 397542a verified.

## Next Phase Readiness

- CAC-04 requirement fulfilled: lock contention now produces exit code 5
- Phase 6 (zone launch) can call `Lock.Acquire()` and the exit code mapping is already in place
- Full binary e2e test for exit code 5 is ready to be added once Phase 6 wires the lock into launch command

---
*Phase: 03-cache-state*
*Completed: 2026-03-27*
