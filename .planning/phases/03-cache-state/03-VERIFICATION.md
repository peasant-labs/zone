---
phase: 03-cache-state
verified: 2026-03-27T21:00:00Z
status: gaps_found
score: 7/9 must-haves verified
re_verification: true
previous_status: gaps_found
previous_score: 6/9
gaps_closed:
  - "Running two zone commands concurrently produces exit code 5 (CAC-04): errors.Is(err, cache.ErrLockContention) -> os.Exit(5) now in main.go"
gaps_remaining:
  - truth: ".zone/ entry is added to .gitignore at the git root with correct relative path"
    status: partial
    reason: "EnsureGitignore is fully implemented and tested. cmd/init.go and cmd/launch.go are stubs returning 'not implemented' — EnsureGitignore is never called from any user-invocable command. CAC-05 requires zone init and zone launch to call it. The wiring is deferred to Phase 6 (lifecycle) per ROADMAP.md architecture. REQUIREMENTS.md marks CAC-05 [x] complete, but the observable behavior ('gitignore updated automatically on first launch') cannot be triggered until Phase 6 wires the commands."
    artifacts:
      - path: "cmd/init.go"
        issue: "Still a stub returning 'not implemented' — does not call cache.EnsureGitignore"
      - path: "cmd/launch.go"
        issue: "Still a stub returning 'not implemented' — does not call cache.EnsureGitignore"
    missing:
      - "Wire cache.EnsureGitignore(cwd) into cmd/init.go RunE — Phase 6 scope"
      - "Wire cache.EnsureGitignore(cwd) into cmd/launch.go RunE — Phase 6 scope"
  - truth: "Build logs are stored and readable via zone logs --build"
    status: partial
    reason: "CreateBuildLog is implemented and tested — it writes to .zone/logs/last_build.log with a metadata header and tees to a writer. cmd/logs.go is a stub returning 'not implemented'. The storage mechanism is Phase 3 work; the reading surface (zone logs --build) is explicitly deferred to Phase 8 per CONTEXT.md. REQUIREMENTS.md marks CAC-06 [x] complete based on storage side only."
    artifacts:
      - path: "cmd/logs.go"
        issue: "Stub returning 'not implemented' — no --build flag, no read from .zone/logs/last_build.log"
    missing:
      - "Wire cmd/logs.go to read .zone/logs/last_build.log when --build flag is passed — Phase 8 scope"
regressions: []
human_verification:
  - test: "Concurrent zone invocations produce exit code 5"
    expected: "When zone launch holds the lock and a second zone launch is run against the same repo, the second process exits with code 5 and prints the PID of the holding process"
    why_human: "Requires two competing live processes with a real command that calls Lock.Acquire(). TestExitCodeLockContentionSentinel verifies the errors.Is detection works; main.go os.Exit(5) is in place. Full binary e2e is only exercisable once Phase 6 wires zone launch to call Lock.Acquire()."
---

# Phase 3: Cache & State Verification Report

**Phase Goal:** Zone reliably tracks image/container/network IDs, detects config changes, and safely handles concurrent invocations
**Verified:** 2026-03-27T21:00:00Z
**Status:** gaps_found
**Re-verification:** Yes — after gap closure plan 03-03

## Re-verification Summary

Previous status: `gaps_found` (6/9 verified, 2026-03-27T20:00:00Z)
Current status: `gaps_found` (7/9 verified)

**Gap closed (1 of 3):**
- Exit code 5 for ErrLockContention (CAC-04): `main.go` now imports `internal/cache` and checks `errors.Is(err, cache.ErrLockContention)` → `os.Exit(5)` before the generic `os.Exit(1)`. Two new tests confirm the sentinel detection.

**Gaps remaining (2 of 3):**
- CAC-05 wiring: `EnsureGitignore` orphaned — both command stubs unchanged (Phase 6 scope)
- CAC-06 surface: `zone logs --build` not wired — `cmd/logs.go` still a stub (Phase 8 scope)

**Regressions:** None — all 17 Phase 3 tests still pass, full test suite clean.

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `.zone/` directory is created with correct subdirectories when EnsureDir is called | VERIFIED | `TestCacheEnsureDir` passes; `EnsureDir()` in cache.go creates `.zone/` and `.zone/logs/` via `os.MkdirAll` |
| 2 | Image, container, network IDs and config hash are persisted atomically and readable | VERIFIED | `TestCacheAtomicWrite`, `TestCacheReadWrite`, `TestCacheReadMissing` pass; `writeAtomic` uses `.tmp-{name}` + `os.Rename`; `readTrimmed` returns `("", nil)` for missing files |
| 3 | Cache hash changes when config changes, when version changes, or when templates change | VERIFIED | `TestHashChangesOnConfigChange`, `TestHashChangesOnVersion` pass; `ComputeHash` feeds `json.Marshal(cfg)` + template bytes + version into `sha256.New()` |
| 4 | Cache hash is stable (same inputs produce same output across invocations) | VERIFIED | `TestHashStability` passes; SHA256 over deterministic JSON + embedded template bytes is stable |
| 5 | Running two zone commands concurrently produces a clear lock-contention error (exit code 5) rather than corruption | VERIFIED | `TestLockDouble` confirms `ErrLockContention` is returned with wrapping. `main.go` now has `errors.Is(err, cache.ErrLockContention)` → `os.Exit(5)` before `os.Exit(1)`. `TestExitCodeLockContentionSentinel` verifies the exact precondition for the exit code mapping. |
| 6 | Stale locks from dead processes are auto-recovered with a warning message | VERIFIED | `lock.go` checks `/proc/{pid}` (Linux) or `kill(pid, 0)` (macOS); stale lock cleanup with `fmt.Fprintf(os.Stderr, "Warning: Recovered stale lock...")` |
| 7 | `.zone/` entry is added to `.gitignore` at git root with correct relative path | PARTIAL | `TestGitignoreCreation` and `TestGitignoreIdempotent` pass. `EnsureGitignore` is fully correct. `cmd/init.go` and `cmd/launch.go` are stubs; `EnsureGitignore` is never called from any user-invocable command. Phase 6 wires this. |
| 8 | Build logs are stored in `.zone/logs/last_build.log` with a metadata header | VERIFIED | `TestBuildLogCreation` and `TestBuildLogHeader` pass; `CreateBuildLog` writes RFC3339 header with config hash and version, uses `io.MultiWriter` for tee |
| 9 | Build logs are readable via `zone logs --build` | PARTIAL | `CreateBuildLog` stores logs correctly. `cmd/logs.go` is a stub; no `--build` flag; no read path from `.zone/logs/last_build.log`. Deferred to Phase 8 per CONTEXT.md. |

**Score:** 7/9 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/cache/cache.go` | Cache struct, EnsureDir, atomic writes, ID CRUD, Dir(), EnsureGitignore, CreateBuildLog | VERIFIED | All exports present, fully implemented; `os.Rename` atomic pattern; `EnsureGitignore` and `CreateBuildLog` added in Plan 02 |
| `internal/cache/hash.go` | SHA256 hash from MergedConfig + templates + version | VERIFIED | `ComputeHash` uses `json.Marshal`, `fs.ReadFile(templates.FS, ...)`, `sha256.New()`, returns 64-char hex |
| `internal/cache/lock.go` | Lock struct, flock-based Acquire/Release, ErrLockContention, PID tracking, stale recovery | VERIFIED | All exports present; EWOULDBLOCK+EAGAIN cross-kernel check; cross-platform dead process detection via `/proc` (Linux) and `kill(0)` (macOS) |
| `main.go` | Exit code translation: errors.Is(err, cache.ErrLockContention) → os.Exit(5) | VERIFIED | Lines 18-19: `if errors.Is(err, cache.ErrLockContention) { os.Exit(5) }` before fallback `os.Exit(1)` on line 21 |
| `tests/cache_test.go` | 13 tests covering cache, lock, gitignore, build log, zone clean, exit code sentinel | VERIFIED | 13 tests present and passing (11 from Plans 01+02 + 2 new from Plan 03) |
| `tests/hash_test.go` | 4 hash tests for stability, config change, version change, empty check | VERIFIED | All 4 tests present and passing |
| `cmd/clean.go` | Wired zone clean command with PID warning | VERIFIED | Uses `cache.New(cwd)`, `cache.ReadLockPID`, `c.Clean()`; no "not implemented" |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/cache/hash.go` | `internal/config/types.go` | `json.Marshal(cfg *config.MergedConfig)` | WIRED | Line 18: `cfgJSON, err := json.Marshal(cfg)` |
| `internal/cache/hash.go` | `pkg/templates/templates.go` | `fs.ReadFile(templates.FS, ...)` | WIRED | Lines 23, 28: `fs.ReadFile(templates.FS, "Dockerfile.tmpl")` and `"entrypoint.sh.tmpl"` |
| `internal/cache/cache.go` | `.zone/` directory | `os.WriteFile`/`os.ReadFile` with atomic rename | WIRED | `writeAtomic`: `os.WriteFile(tmpPath)` + `os.Rename(tmpPath, target)` |
| `internal/cache/lock.go` | `syscall.Flock` | `LOCK_EX\|LOCK_NB` for non-blocking exclusive lock | WIRED | Line 42: `syscall.Flock(int(f.Fd()), syscall.LOCK_EX\|syscall.LOCK_NB)` |
| `internal/cache/lock.go` | `.zone/.lock.pid` | PID file written on acquire, read on contention | WIRED | Lines 47, 59: `os.WriteFile(pidPath, ...)` and `readPIDFile(...)` |
| `internal/cache/cache.go (EnsureGitignore)` | `git rev-parse --show-toplevel` | `os/exec.Command` for git root discovery | WIRED | Line 101: `exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel")` |
| `cmd/clean.go` | `internal/cache` | `cache.New(cwd)` + `c.Clean()` | WIRED | Lines 19, 27: `cache.New(cwd)` and `c.Clean()` |
| `main.go` | `internal/cache/lock.go` | `errors.Is(err, cache.ErrLockContention)` → `os.Exit(5)` | WIRED | Lines 18-19: `if errors.Is(err, cache.ErrLockContention) { os.Exit(5) }` — Gap 1 closed by Plan 03-03 |
| `EnsureGitignore` | `cmd/init.go` | Called during zone init | NOT WIRED | `cmd/init.go` is a stub — `EnsureGitignore` never called. Phase 6 scope. |
| `EnsureGitignore` | `cmd/launch.go` | Called during zone launch | NOT WIRED | `cmd/launch.go` is a stub — `EnsureGitignore` never called. Phase 6 scope. |
| `CreateBuildLog` | `cmd/logs.go` | Read via `zone logs --build` | NOT WIRED | `cmd/logs.go` is a stub. Phase 8 scope. |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| CAC-01 | 03-01-PLAN.md | `.zone/` directory stores config hash, Dockerfile, entrypoint, image/container/network IDs | SATISFIED | `Cache` struct with `SetImageID`, `SetContainerID`, `SetNetworkID`, `SetConfigHash`, `EnsureDir` — all implemented and tested. 4 tests pass. |
| CAC-02 | 03-01-PLAN.md | Cache hash includes merged config + templates + Zone version for automatic invalidation | SATISFIED | `ComputeHash` hashes `json.Marshal(cfg)` + `Dockerfile.tmpl` + `entrypoint.sh.tmpl` + version string. `TestHashChangesOnConfigChange` and `TestHashChangesOnVersion` verify invalidation. |
| CAC-03 | 03-02-PLAN.md | File-based locking via flock for concurrent access protection | SATISFIED | `Lock.Acquire()` uses `syscall.Flock(LOCK_EX\|LOCK_NB)`. `TestLockDouble` confirms contention is detected and `ErrLockContention` returned. |
| CAC-04 | 03-03-PLAN.md (gap closure) | Lock contention produces error with exit code 5 | SATISFIED | `main.go` checks `errors.Is(err, cache.ErrLockContention)` and calls `os.Exit(5)` before `os.Exit(1)`. `TestExitCodeLockContentionSentinel` verifies the detection logic. Gap closed by Plan 03-03. |
| CAC-05 | 03-02-PLAN.md | `zone init` and `zone launch` add `.zone/` to `.gitignore` | BLOCKED | `EnsureGitignore` is fully implemented and tested (2 tests pass). Neither `cmd/init.go` nor `cmd/launch.go` call it — both stubs. The function cannot be triggered by a user. Wiring is Phase 6 scope. REQUIREMENTS.md marks `[x]` based on the mechanism existing, not the full observable behavior. |
| CAC-06 | 03-02-PLAN.md | Build logs stored in `.zone/logs/last_build.log` | PARTIALLY SATISFIED | `CreateBuildLog` stores logs with RFC3339 metadata header — 2 tests pass. `cmd/logs.go` is a stub; `zone logs --build` is not wired. Storage side satisfies the letter of CAC-06; the reading surface is Phase 8 scope per CONTEXT.md. |

### Anti-Patterns Found

No blocker anti-patterns detected.

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `cmd/init.go` | 13 | `return fmt.Errorf("not implemented")` | Warning | Pre-existing stub; EnsureGitignore cannot be invoked via `zone init` — Phase 6 |
| `cmd/launch.go` | 13 | `return fmt.Errorf("not implemented")` | Warning | Pre-existing stub; EnsureGitignore cannot be invoked via `zone launch` — Phase 6 |
| `cmd/logs.go` | 13 | `return fmt.Errorf("not implemented")` | Warning | Pre-existing stub; build logs cannot be read via `zone logs --build` — Phase 8 |

Note: These stubs are pre-existing Phase 1 scaffolding. No Phase 3 code introduced new stubs or TODOs.

### Human Verification Required

#### 1. Concurrent lock contention exit code (end-to-end binary test)

**Test:** Build the zone binary. In a directory with a `.zone/` that has a live lock held (by a running zone launch), run a second `zone launch` command.
**Expected:** Second process exits with code 5 and prints a message including the PID of the holding process.
**Why human:** Requires two competing live processes with a real command that calls `Lock.Acquire()`. The sentinel test (`TestExitCodeLockContentionSentinel`) and the `main.go` exit code mapping are both verified. Full binary e2e becomes exercisable once Phase 6 wires `zone launch` to call `Lock.Acquire()`.

### Gaps Summary

Two gaps remain from the original three:

**Gap 1 — CLOSED (CAC-04):** Plan 03-03 added `errors.Is(err, cache.ErrLockContention)` → `os.Exit(5)` to `main.go`. Two sentinel tests confirm the logic. This was the only gap with a closure plan.

**Gap 2 — Open (CAC-05):** `EnsureGitignore` is orphaned. `cmd/init.go` and `cmd/launch.go` are Phase 1 stubs that will be implemented in Phase 6. The function is ready to be called. The observable behavior (".gitignore updated automatically") requires Phase 6.

**Gap 3 — Open (CAC-06):** `CreateBuildLog` (storage) is implemented and tested. `cmd/logs.go` reading surface deferred to Phase 8 per CONTEXT.md explicit design decision. CAC-06 as written ("Build logs stored in .zone/logs/last_build.log") is satisfied by the storage mechanism alone — but ROADMAP Success Criterion 4 ("readable via zone logs --build") is not yet achievable.

**Root cause:** Gaps 2 and 3 are architectural deferrals. Phase 3 delivers the cache layer mechanisms (EnsureGitignore function, CreateBuildLog method). The command wiring (Phase 6 for init/launch, Phase 8 for logs) is correctly out of Phase 3 scope. No additional gap-closure plans are needed for Phase 3 — these will be closed naturally when Phase 6 and Phase 8 implement their respective commands.

---

_Verified: 2026-03-27T21:00:00Z_
_Verifier: Claude (gsd-verifier)_
_Re-verification: Yes (previous: 2026-03-27T20:00:00Z, status: gaps_found, score: 6/9)_
