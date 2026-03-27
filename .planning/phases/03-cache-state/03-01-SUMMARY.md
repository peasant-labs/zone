---
phase: 03-cache-state
plan: 01
subsystem: cache
tags: [sha256, atomic-write, os.Rename, embed.FS, crypto/sha256, encoding/json]

# Dependency graph
requires:
  - phase: 02-config-foundation
    provides: MergedConfig struct and HarnessConfig with CustomAliases map[string]string
  - phase: 01-scaffold
    provides: pkg/templates with embedded FS (Dockerfile.tmpl, entrypoint.sh.tmpl)

provides:
  - Cache struct with New, EnsureDir, Dir, Clean, and 8 ID getter/setter methods
  - Atomic write pattern via .tmp- prefix + os.Rename
  - ComputeHash(cfg *config.MergedConfig, version string) returning 64-char hex SHA256
  - .zone/ and .zone/logs/ directory management

affects:
  - 04-template-system
  - 06-lifecycle

# Tech tracking
tech-stack:
  added: []
  patterns:
    - atomic-write via .tmp-{name} + os.Rename for crash-safe file persistence
    - readTrimmed returns ("", nil) for missing files — not-found is not an error
    - ComputeHash takes version as string parameter (not imported) to avoid import graph violation
    - SHA256 over json.Marshal(cfg) + template bytes + version string for deterministic cache invalidation
    - fs.ReadFile(templates.FS, name) for reading embedded templates (not string vars)

key-files:
  created:
    - internal/cache/cache.go
    - internal/cache/hash.go
    - tests/cache_test.go
  modified:
    - tests/hash_test.go

key-decisions:
  - "ComputeHash takes version as string param to avoid main.go import graph violation"
  - "readTrimmed returns (\"\", nil) for missing files — not-found is not an error"
  - "Hash includes only Dockerfile.tmpl + entrypoint.sh.tmpl (not zone-bashrc.tmpl) per spec"
  - "json.Marshal on MergedConfig struct produces deterministic field order; map[string]string keys sorted alphabetically"

patterns-established:
  - "Cache atomic write: writeAtomic writes .tmp-{name}, then os.Rename to target"
  - "Cache read: readTrimmed uses os.IsNotExist to distinguish missing vs error"
  - "Hash computation: json.Marshal(cfg) + embedded template bytes + version string fed to sha256.New()"

requirements-completed: [CAC-01, CAC-02]

# Metrics
duration: 8min
completed: 2026-03-27
---

# Phase 3 Plan 01: Cache & State — Cache and Hash Implementation Summary

**Cache struct with atomic .tmp-+os.Rename writes and SHA256 hash over json.Marshal(MergedConfig)+templates+version**

## Performance

- **Duration:** 8 min
- **Started:** 2026-03-27T19:18:10Z
- **Completed:** 2026-03-27T19:26:00Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments
- Cache struct with New, EnsureDir, Dir, Clean, and 8 getter/setter methods for image/container/network/config IDs
- Atomic write via .tmp- prefix + os.Rename pattern for crash-safe ID persistence
- readTrimmed returns ("", nil) for missing files — not an error (enables fresh-cache queries without init overhead)
- ComputeHash returns 64-char hex SHA256 over merged config JSON + Dockerfile + entrypoint templates + version
- 8 tests across cache_test.go and hash_test.go all pass; no new dependencies added to go.mod

## Task Commits

Each task was committed atomically:

1. **Task 1: Write test scaffolds for cache and hash** - `f310f6b` (test — RED state)
2. **Task 2: Implement cache.go** - `48556e1` (feat — GREEN for cache tests)
3. **Task 3: Implement hash.go** - `dcf1757` (feat — GREEN for hash tests)

**Plan metadata:** (docs commit follows)

_Note: TDD tasks — test commit establishes RED, implementation commits achieve GREEN._

## Files Created/Modified
- `internal/cache/cache.go` — Cache struct, EnsureDir, atomic writeAtomic/readTrimmed, 8 ID getter/setter methods, Clean
- `internal/cache/hash.go` — ComputeHash(cfg *MergedConfig, version string) (string, error) via SHA256
- `tests/cache_test.go` — TestCacheEnsureDir, TestCacheAtomicWrite, TestCacheReadWrite, TestCacheReadMissing
- `tests/hash_test.go` — TestHashStability, TestHashChangesOnConfigChange, TestHashChangesOnVersion, TestHashNotEmpty

## Decisions Made
- **ComputeHash takes version as string param** — main.go's `var version = "dev"` cannot be imported into internal/cache without violating the import graph (main -> cmd -> internal). Passing as parameter is the correct pattern.
- **readTrimmed returns ("", nil) for missing files** — not-found is a valid cache state (fresh repo), not an error. Callers check for empty string to detect uncached state.
- **Hash includes Dockerfile.tmpl + entrypoint.sh.tmpl only** — zone-bashrc.tmpl is excluded per spec; only templates that affect the built image participate in the invalidation hash.
- **fs.ReadFile(templates.FS, name)** — templates.FS is an embed.FS, not string vars; ReadFile is the correct interface.

## Deviations from Plan

None — plan executed exactly as written. Task 2 and Task 3 were implemented together because the `tests` package (single build unit) required ComputeHash to exist before the cache_test.go build could proceed. Both implementations followed the plan's specified code verbatim.

## Issues Encountered
- Task 2 verification required Task 3 implementation first: the `tests/` package is a single Go build unit, so undefined `cache.ComputeHash` caused `go test ./tests/ -run TestCache*` to fail to compile. Implemented hash.go immediately (no deviation — this was already Task 3) and verified both test groups together.

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- `internal/cache` package is complete and ready for consumers
- Phase 4 (template system) can import `cache.New`, `cache.Cache`, and use Dir() to locate .zone/ for Dockerfile/entrypoint output
- Phase 6 (lifecycle) can import all getter/setter methods for image/container/network ID tracking
- No blockers. go build ./... passes with no new dependencies.

---
*Phase: 03-cache-state*
*Completed: 2026-03-27*

## Self-Check: PASSED

- internal/cache/cache.go: FOUND
- internal/cache/hash.go: FOUND
- tests/cache_test.go: FOUND
- tests/hash_test.go: FOUND
- 03-01-SUMMARY.md: FOUND
- Commit f310f6b (test scaffolds): FOUND
- Commit 48556e1 (cache.go): FOUND
- Commit dcf1757 (hash.go): FOUND
- All 8 tests pass: CONFIRMED
- go vet ./internal/cache/...: PASSED
- go build ./...: PASSED
