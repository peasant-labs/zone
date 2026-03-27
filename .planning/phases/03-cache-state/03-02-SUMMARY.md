---
phase: 03-cache-state
plan: 02
subsystem: cache
tags: [flock, syscall, pid-tracking, gitignore, build-log, io.MultiWriter, exec.Command, zone-clean]

# Dependency graph
requires:
  - phase: 03-cache-state
    plan: 01
    provides: Cache struct with New, EnsureDir, Dir, Clean, atomic read/write methods
  - phase: 01-scaffold
    provides: cobra cmd structure with cleanCmd stub registered in root.go

provides:
  - Lock struct with flock-based Acquire/Release/IsHeld, ErrLockContention sentinel, ReadLockPID
  - Stale lock auto-recovery with stderr warning (dead process detection via /proc or kill -0)
  - EnsureGitignore standalone function with git root discovery and idempotent .gitignore management
  - CreateBuildLog method with io.MultiWriter tee and RFC3339 metadata header
  - Wired zone clean command (replaces "not implemented" stub)
  - 7 new tests covering lock contention, gitignore, build log, and zone clean integration

affects:
  - 04-template-system
  - 06-lifecycle

# Tech tracking
tech-stack:
  added: []
  patterns:
    - flock LOCK_EX|LOCK_NB for non-blocking exclusive lock with EWOULDBLOCK+EAGAIN cross-kernel check
    - PID file written on lock acquire, read on contention for diagnostic error messages
    - Stale lock detection via /proc/{pid} on Linux, kill(pid, 0)/ESRCH on macOS
    - EnsureGitignore uses git rev-parse --show-toplevel for reliable git root discovery
    - filepath.Rel(gitRoot, cwd) to compute correct relative .zone/ path for nested repos
    - io.MultiWriter for simultaneous terminal + log file output (CreateBuildLog)
    - Build log header format: "# zone build | {RFC3339} | config hash: {hash} | zone {version}"

key-files:
  created:
    - internal/cache/lock.go
  modified:
    - internal/cache/cache.go
    - cmd/clean.go
    - tests/cache_test.go

key-decisions:
  - "zone clean warns if lock PID found but proceeds unconditionally — CONTEXT.md decision, avoids deadlock if prior process crashed"
  - "EnsureGitignore is a standalone function (not Cache method) — operates on cwd, independent of .zone/ existence"
  - "Stale lock recovery treats pid<=0 (parse failure) as stale — avoids infinite lock if PID file was corrupted"
  - "Dead process check: /proc on Linux for zero-overhead stat, kill(0) on macOS for portability"

patterns-established:
  - "Lock.Acquire: open .lock, flock(LOCK_EX|LOCK_NB), write .lock.pid; on EWOULDBLOCK check if holder is dead before returning ErrLockContention"
  - "EnsureGitignore: git rev-parse --show-toplevel -> rel path -> line-scan .gitignore -> append if absent"
  - "CreateBuildLog: os.Create logPath, write header, return io.MultiWriter(w, f) + closer func"

requirements-completed: [CAC-03, CAC-04, CAC-05, CAC-06]

# Metrics
duration: 2min
completed: 2026-03-27
---

# Phase 3 Plan 02: Cache & State — Lock, Gitignore, Build Log, and zone clean Summary

**flock-based Lock with stale recovery + idempotent EnsureGitignore + io.MultiWriter build log + wired zone clean command, 15 total cache tests passing**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-27T19:22:28Z
- **Completed:** 2026-03-27T19:24:49Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments
- Lock struct with syscall.Flock LOCK_EX|LOCK_NB, PID file tracking, ErrLockContention sentinel, and auto-recovery of stale locks from dead processes (cross-platform: /proc on Linux, kill -0 on macOS)
- EnsureGitignore discovers git root via `git rev-parse --show-toplevel`, computes relative `.zone/` entry for nested repos, is idempotent via line-scan before append
- CreateBuildLog uses io.MultiWriter to simultaneously tee output to terminal writer and `.zone/logs/last_build.log` with RFC3339 metadata header
- zone clean command wired: reads lock PID for warning, calls c.Clean(), prints confirmation — no longer returns "not implemented"
- 7 new tests + 8 from Plan 01 = 15 total passing; full `go test ./...` green with no regressions

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement lock.go** - `59317ad` (feat)
2. **Task 2: EnsureGitignore + CreateBuildLog + wire zone clean** - `bb65b71` (feat)
3. **Task 3: Tests for lock, gitignore, build log, zone clean** - `7084e0d` (test)

**Plan metadata:** (docs commit follows)

## Files Created/Modified
- `internal/cache/lock.go` — Lock struct, NewLock, Acquire/Release/IsHeld, ErrLockContention, ReadLockPID, readPIDFile, isDeadProcess
- `internal/cache/cache.go` — Added EnsureGitignore (standalone), CreateBuildLog (method); added bufio/io/os/exec/time imports
- `cmd/clean.go` — Wired with cache.New(cwd), ReadLockPID PID warning, c.Clean() call
- `tests/cache_test.go` — Added TestLockAcquireRelease, TestLockDouble, TestGitignoreCreation, TestGitignoreIdempotent, TestBuildLogCreation, TestBuildLogHeader, TestCleanCommand

## Decisions Made
- **zone clean warns but proceeds if lock held** — CONTEXT.md decision. A crashed prior process would leave lock held forever if clean blocked; warning + proceed is the correct UX.
- **EnsureGitignore as standalone function** — It operates on the working directory (cwd), not on an existing .zone/ dir. Making it a Cache method would require EnsureDir first, which is wrong since gitignore management is independent of cache initialization.
- **Stale lock: pid<=0 treated as stale** — A corrupted or empty .lock.pid file should not permanently block zone. Auto-recovery with a warning is the safe default.
- **Dead process check via /proc on Linux** — Cheaper than kill(0) and works for processes owned by other users (kill(0) can fail with EPERM for foreign processes).

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered
None — all implementations compiled on first attempt and all tests passed immediately.

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- `internal/cache` package is now complete: atomic reads/writes, SHA256 hash, flock locking, gitignore management, build log storage
- Phase 4 (template system) can call `cache.EnsureGitignore(cwd)` after writing templates to .zone/
- Phase 6 (lifecycle) can use `cache.NewLock(c.Dir()).Acquire()` / `defer lock.Release()` to protect concurrent operations
- Phase 6 can call `c.CreateBuildLog(os.Stdout, hash, version)` for docker build output logging
- No blockers. `go test ./...` passes. `go build ./...` succeeds.

---
*Phase: 03-cache-state*
*Completed: 2026-03-27*

## Self-Check: PASSED

- internal/cache/lock.go: FOUND
- internal/cache/cache.go: FOUND
- cmd/clean.go: FOUND
- tests/cache_test.go: FOUND
- 03-02-SUMMARY.md: FOUND
- Commit 59317ad (lock.go): FOUND
- Commit bb65b71 (cache.go + clean.go): FOUND
- Commit 7084e0d (tests): FOUND
- All 15 Phase 3 tests pass: CONFIRMED (15 RUN, 15 PASS)
- go vet ./...: PASSED
- go build ./...: PASSED
