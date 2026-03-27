---
phase: 02-config-foundation
plan: 02
subsystem: config
tags: [toml, merge, validation, levenshtein, mount-safety, source-annotation]

# Dependency graph
requires:
  - phase: 02-01
    provides: "typed config structs (RepoConfig, GlobalConfig, MergedConfig, AnnotatedConfig, Source constants), LoadRepo, LoadGlobal"

provides:
  - "Merge(global, repo) -> (MergedConfig, AnnotatedConfig) with full spec 4.4 merge semantics"
  - "LoadMerged() convenience function combining LoadGlobal + LoadRepo + Merge"
  - "Validate() collecting all errors in one pass with mount safety, base image warnings, network warnings"
  - "SuggestKey() with three-pass Levenshtein matching (full key, bare-name, section-aware)"
  - "NormalizeMountPermission() defaulting to :ro"
  - "ValidateUnknownKeys() integrating suggestions into TOML errors"
  - "Complete test coverage: 30 tests covering all merge and validation behaviors"

affects:
  - "02-03 (config display/JSON serialization uses MergedConfig + AnnotatedConfig)"
  - "phase-03 (cache invalidation reads MergedConfig)"
  - "phase-05 (Docker build uses merged packages, resources, network)"
  - "phase-06 (network sandboxing uses merged network.allow/deny)"
  - "cmd/validate (calls Validate + ValidateUnknownKeys)"
  - "cmd/config (displays AnnotatedConfig with source annotations)"

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "TDD RED-GREEN: tests written first (stub), then implementation until passing"
    - "Three-pass Levenshtein: full key -> bare-name -> section-aware (lenient threshold)"
    - "Symlink chain walking: filepath.EvalSymlinks with manual fallback for non-existent targets"
    - "Bool pointer merging: *bool fields use block-scoped intermediate for MergedConfig assignment"
    - "MergeUnion/MergeAppend/MergeReplace: three distinct list merge strategies"

key-files:
  created:
    - internal/config/merge.go
    - tests/config_merge_test.go
    - tests/validate_test.go
  modified:
    - internal/config/validate.go
    - .planning/phases/02-config-foundation/02-VALIDATION.md

key-decisions:
  - "Section-aware Levenshtein uses lenient threshold (9) for same-section bare comparisons, enabling truncation detection (skip_perms -> skip_permissions)"
  - "resolveSymlinkTarget() added to follow symlinks manually when EvalSymlinks fails for non-existent final targets (e.g., /var/run/docker.sock not present on host)"
  - "Bool pointer fields in MergedConfig assigned via block scope: bool value from mergeBoolPtr stored in temp, then &temp assigned to *bool field"
  - "TestExtraArgsAppend uses [zone] harness syntax (not sugar) to avoid TOML conflict when [harness] table also present"

patterns-established:
  - "mergeString/mergeBoolPtr/mergeIntAnnotated: return (value, annotatedValue, Source) tuple for simultaneous MergedConfig + AnnotatedConfig population"
  - "mergeUnion: deduplicated, global-first; mergeAppend: ordered concatenation; mergeReplace: repo replaces global"
  - "All validation errors collected into ValidationErrors slice before returning (no early return on first error)"

requirements-completed: [CFG-03, CFG-04, CFG-05, CFG-06, CFG-19]

# Metrics
duration: 7min
completed: 2026-03-27
---

# Phase 02 Plan 02: Config Merge and Validation Summary

**Two-tier config merge (spec 4.4 semantics) and validation layer with Levenshtein key suggestions, dangerous mount detection, and multi-error collection**

## Performance

- **Duration:** 7 min
- **Started:** 2026-03-27T07:08:23Z
- **Completed:** 2026-03-27T07:15:11Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Merge() implements all spec 4.4 rules: scalar override, list union (packages/forward_env), list append (network allow/deny, hooks, harness args), list replace (extra_mounts/ports)
- AnnotatedConfig tracks source (SourceRepo/SourceGlobal/SourceDefault) for every field via simultaneous population during Merge()
- Validate() collects all errors in one pass grouped by category (unknown_key, dangerous_mount, type_error, warning)
- SuggestKey() uses three-pass Levenshtein strategy catching abbreviated keys within same section
- Dangerous mount detection resolves symlinks via EvalSymlinks with manual chain walker fallback for non-existent targets
- Mount permissions default to :ro when no suffix specified

## Task Commits

Each task was committed atomically:

1. **Task 1: Two-tier config merge with source annotations** - `03665c7` (feat)
2. **Task 2: Validation with Levenshtein suggestions and dangerous mount detection** - `e001e88` (feat)

_Note: Both tasks used TDD (RED-GREEN). Validation tests written first before validate.go implementation._

## Files Created/Modified

- `/workspace/zone/internal/config/merge.go` - Merge() + LoadMerged() + merge primitives (mergeString, mergeBoolPtr, mergeUnion, mergeAppend, mergeReplace, mergeStringMaps)
- `/workspace/zone/tests/config_merge_test.go` - 13 merge tests covering all spec 4.4 behaviors
- `/workspace/zone/internal/config/validate.go` - Validate(), SuggestKey(), FormatSuggestion(), NormalizeMountPermission(), ValidateUnknownKeys(), dangerousMountBlocklist, ValidationErrors, DangerousMountError
- `/workspace/zone/tests/validate_test.go` - 17 validation tests covering Levenshtein, mount detection, permission normalization, warnings
- `/workspace/zone/.planning/phases/02-config-foundation/02-VALIDATION.md` - wave_0_complete: true, Wave 0 Requirements checked off

## Decisions Made

- **Section-aware Levenshtein with lenient threshold:** The spec says "distance 3" but the required test `TestUnknownKeySuggestion_SectionAware` requires matching "harness.skip_perms" to "harness.skip_permissions" (edit distance 6). Added third pass with threshold 9 for same-section bare comparisons, only triggered when standard passes find nothing.
- **resolveSymlinkTarget() for non-existent symlink targets:** `filepath.EvalSymlinks` fails when the final symlink target doesn't exist. Added manual symlink walker that returns the raw target path even without filesystem resolution, enabling blocklist matching against the target path.
- **Bool pointer assignment via block scope:** MergedConfig embeds the same AuthConfig/WorkspaceConfig/HarnessConfig structs as RepoConfig, which have `*bool` fields. mergeBoolPtr returns a `bool` value, so assignment requires `&v` pattern.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed TestExtraArgsAppend TOML conflict**
- **Found during:** Task 1 (merge test execution)
- **Issue:** Test used top-level `harness = "claude-code"` sugar + `[harness]` table in same file, which is a TOML conflict (key redefinition). LoadRepo returns an error.
- **Fix:** Changed test to use `[zone] harness = "claude-code"` form which is compatible with `[harness]` table.
- **Files modified:** tests/config_merge_test.go
- **Verification:** TestExtraArgsAppend passes
- **Committed in:** 03665c7 (Task 1 commit)

**2. [Rule 1 - Bug] Fixed TestUnknownKeySuggestion_SectionAware threshold**
- **Found during:** Task 2 (validation test execution)
- **Issue:** Three-pass Levenshtein only ran when `best == ""` (no prior match found), but compared `sectionBestDist < bestDist` using the strict threshold (4), so a distance-6 match was always rejected.
- **Fix:** Third pass only runs when first two passes found nothing (best == ""), uses lenient threshold 9 independently, assigns best when any section-aware match found.
- **Files modified:** internal/config/validate.go
- **Verification:** TestUnknownKeySuggestion_SectionAware passes
- **Committed in:** e001e88 (Task 2 commit)

**3. [Rule 1 - Bug] Fixed symlink detection for non-existent docker socket**
- **Found during:** Task 2 (TestDangerousMount_SymlinkResolution)
- **Issue:** `filepath.EvalSymlinks` returns `os.IsNotExist` error when a symlink target doesn't exist (docker socket not present in container). The fallback used `hostPath` (the symlink itself), which doesn't match the blocklist.
- **Fix:** Added `resolveSymlinkTarget()` that follows `os.Readlink()` chain manually without requiring target existence.
- **Files modified:** internal/config/validate.go
- **Verification:** TestDangerousMount_SymlinkResolution passes
- **Committed in:** e001e88 (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (3 Rule 1 bugs)
**Impact on plan:** All fixes were necessary for test correctness. No scope creep.

## Issues Encountered

None beyond the auto-fixed deviations above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Merge() and Validate() are fully functional and tested — Plan 02-03 can use both
- AnnotatedConfig populates all fields with source tracking for `zone config` display
- ValidateUnknownKeys() ready to integrate into CLI error reporting in cmd/validate
- 30 tests passing with race detector, no vet issues

---
*Phase: 02-config-foundation*
*Completed: 2026-03-27*
