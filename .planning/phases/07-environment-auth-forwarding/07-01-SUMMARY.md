---
phase: 07-environment-auth-forwarding
plan: 01
subsystem: docker
tags: [env, glob, filepath.Match, bufio, .env-file, forward_env, required_env]

requires:
  - phase: 06-docker-lifecycle-core
    provides: "internal/docker package, Manager, DockerClient interface"

provides:
  - "CollectForwardedEnv: glob-based env var collection from host using filepath.Match"
  - "ParseEnvFile: Docker-compatible .env file parser (KEY=VALUE, skips comments/blanks)"
  - "ValidateRequiredEnv: pre-launch check against combined host env + .env file"

affects:
  - 07-02-auth-config-mount
  - 07-03-container-create-wiring

tech-stack:
  added: []
  patterns:
    - "bufio.Scanner over bytes.NewReader for in-memory line parsing"
    - "filepath.Match for glob semantics (same as Docker's forward_env)"
    - "deduplication via map[string]bool keyed on env var name"
    - "combined host+file env check: os.Environ() supplemented by ParseEnvFile"

key-files:
  created:
    - internal/docker/env.go
    - internal/docker/env_test.go
  modified: []

key-decisions:
  - "filepath.Match chosen over path/filepath glob per spec section 4.6 — same semantics as Docker's own env forwarding"
  - "Warnings (not errors) for unmatched forward_env patterns — user intent may be pattern across optional vars"
  - "ValidateRequiredEnv resolves relative envFilePath via filepath.Join(repoDir, envFilePath) — consistent with how Manager already resolves config-relative paths"
  - "first-= split in ParseEnvFile preserves base64 and URL values that contain = signs"

patterns-established:
  - "TDD: failing tests committed first, implementation second, both as separate commits"
  - "Pure-logic helpers in docker package have no Docker SDK imports — fully unit-testable"

requirements-completed:
  - CFG-10
  - CFG-11
  - CFG-14

duration: 2min
completed: 2026-03-30
---

# Phase 07 Plan 01: Env Forwarding Logic Summary

**Glob-based env forwarding, .env file parsing, and pre-launch required-env validation via three pure functions in internal/docker/env.go**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-03-30T00:28:08Z
- **Completed:** 2026-03-30T00:29:48Z
- **Tasks:** 1 (TDD: 2 commits — RED test + GREEN impl)
- **Files modified:** 2

## Accomplishments

- CollectForwardedEnv collects host env vars via filepath.Match glob patterns with deduplication and per-pattern warnings for empty matches
- ParseEnvFile parses Docker-compatible .env files: KEY=VALUE lines, comment/blank skipping, first-= split for values containing =
- ValidateRequiredEnv checks combined host env + .env file against required list, returns spec-compliant error with var name and harness name
- 20 unit tests covering all 19 spec behaviors plus deduplication edge case

## Task Commits

Each task was committed atomically:

1. **TDD RED: failing tests** - `1b4eff3` (test)
2. **TDD GREEN: env.go implementation** - `eb5e9af` (feat)

## Files Created/Modified

- `internal/docker/env.go` - CollectForwardedEnv, ParseEnvFile, ValidateRequiredEnv; no Docker SDK imports
- `internal/docker/env_test.go` - 20 test functions covering all specified behaviors

## Decisions Made

- filepath.Match chosen for glob semantics per spec section 4.6 — same semantics Docker uses internally
- Warnings (not errors) for unmatched forward_env patterns to handle optional env vars gracefully
- first-= split in ParseEnvFile preserves base64 and URL values that contain = signs
- relative envFilePath resolved via filepath.Join(repoDir, path) — matches how Manager resolves other config-relative paths

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- env.go provides the three functions Plan 03 will wire into createContainer() and Launch()
- ParseEnvFile and CollectForwardedEnv are ready for auth config mount path resolution (Plan 02)
- All acceptance criteria met: filepath.Match, os.Environ(), bufio.NewScanner, strings.HasPrefix all present

---
*Phase: 07-environment-auth-forwarding*
*Completed: 2026-03-30*
