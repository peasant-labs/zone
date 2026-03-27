---
phase: 02-config-foundation
plan: 03
subsystem: config
tags: [cobra, toml, json, annotated-output, validation, exit-codes]

# Dependency graph
requires:
  - phase: 02-config-foundation/02-01
    provides: typed config structs, LoadRepo, LoadGlobal, LoadMerged
  - phase: 02-config-foundation/02-02
    provides: Validate, ValidateUnknownKeys, ValidationErrors, NormalizeMountPermission

provides:
  - zone config command with --json and --global flags (annotated TOML + JSON output)
  - zone validate command with --allow-dangerous-mount flag (exit code 2 on errors)
  - Integration tests for CFG-07 (annotated TOML) and CFG-08 (JSON output)

affects: [phase-03, phase-04, phase-07, phase-08]

# Tech tracking
tech-stack:
  added: [github.com/stretchr/testify v1.10.0]
  patterns:
    - renderAnnotatedTOML writes to strings.Builder, returns string for capture
    - renderJSON uses map[string]any with AnnotatedFieldJSON leaves
    - Integration tests use binary subprocess via sync.Once build + exec.Command
    - cmd.OutOrStdout() for all stdout output enabling test capture
    - errors.As(err, &uke) pattern for UnknownKeysError alongside valid config

key-files:
  created:
    - cmd/config.go
    - tests/config_cmd_test.go
  modified:
    - cmd/validate.go
    - .planning/phases/02-config-foundation/02-VALIDATION.md
    - go.mod
    - go.sum

key-decisions:
  - "renderAnnotatedTOML emits comment block above lists (not inline) — inline TOML comments on array elements are invalid TOML per spec pitfall 4"
  - "zone validate loads global + repo separately to accumulate UnknownKeysError alongside valid partial config"
  - "Integration tests use pre-built binary via sync.Once pattern — avoids go run recompile cost per test"
  - "testify v1.10.0 added for assert/require in integration tests"

patterns-established:
  - "Comment block pattern: # key: source provides [vals]; source adds [vals] above list fields in annotated TOML"
  - "JSON output pattern: {value: X, source: Y} per leaf field via AnnotatedFieldJSON type"
  - "Exit code 2 pattern: os.Exit(2) for config errors per spec section 8 DX-01"
  - "cmd.OutOrStdout() pattern: all command output goes through Cobra's OutOrStdout for test capture"

requirements-completed: [CFG-07, CFG-08]

# Metrics
duration: 4min
completed: 2026-03-27
---

# Phase 2 Plan 03: Config CLI Commands Summary

**`zone config` with annotated TOML + JSON output modes and `zone validate` with grouped errors + exit code 2, both wired to internal/config package via Cobra commands**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-27T07:18:34Z
- **Completed:** 2026-03-27T07:22:32Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- `zone config` renders annotated TOML with `# source` comments per field; list fields get a comment block above showing provenance per source
- `zone config --json` renders JSON with `{value, source}` structure per field using AnnotatedFieldJSON
- `zone config --global` works without zone.toml, merging against empty repo to show global defaults
- `zone validate` collects all errors (unknown keys, dangerous mounts, type errors) in one pass, exits 2 on errors
- 6 integration tests prove CFG-07 and CFG-08 output correctness using pre-built binary subprocess

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire zone config command** - `971a538` (feat)
2. **Task 2: Wire zone validate command** - `b8e41df` (feat)
3. **Task 3: Integration tests CFG-07 + CFG-08** - `f405804` (test)

## Files Created/Modified

- `/workspace/zone/cmd/config.go` - Full config command: --json, --global, renderAnnotatedTOML, renderJSON
- `/workspace/zone/cmd/validate.go` - Full validate command: --allow-dangerous-mount, exit code 2, grouped errors
- `/workspace/zone/tests/config_cmd_test.go` - 6 integration tests using pre-built binary
- `/workspace/zone/.planning/phases/02-config-foundation/02-VALIDATION.md` - nyquist_compliant: true, tasks marked green
- `/workspace/zone/go.mod` - Added github.com/stretchr/testify v1.10.0
- `/workspace/zone/go.sum` - Updated checksums

## Decisions Made

- **renderAnnotatedTOML list format:** Comment block ABOVE the array (not inline comments on array elements), because inline TOML comments on array elements are not valid TOML per spec pitfall 4
- **validate.go uses LoadGlobal + LoadRepo separately:** Needed to accumulate UnknownKeysError from both configs while still having a valid config for Validate()
- **Binary subprocess for integration tests:** Built once per test run via sync.Once, avoids `go run` recompile overhead; direct execution also tests exit codes correctly
- **testify added as dependency:** Provides assert/require helpers for cleaner test assertions

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added github.com/stretchr/testify dependency**
- **Found during:** Task 3 (integration tests)
- **Issue:** Tests imported testify/assert and testify/require which were not in go.mod
- **Fix:** Ran `go get github.com/stretchr/testify@v1.10.0` and `go get github.com/stretchr/testify/assert@v1.10.0`
- **Files modified:** go.mod, go.sum
- **Verification:** Tests compile and pass cleanly
- **Committed in:** f405804 (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking - missing dependency)
**Impact on plan:** Minimal — testify is a standard Go testing library, adding it enables cleaner test assertions.

## Issues Encountered

None - plan executed cleanly. All 37 tests pass including 6 new integration tests.

## Next Phase Readiness

- Phase 2 config foundation complete: config parsing, merge, validation, and CLI commands all wired
- `zone config` and `zone validate` are user-facing and functional
- Phase 3 (Cache) and Phase 4 (Template System) can now proceed — both depend on Phase 2's config foundation
- No blockers

---
*Phase: 02-config-foundation*
*Completed: 2026-03-27*
