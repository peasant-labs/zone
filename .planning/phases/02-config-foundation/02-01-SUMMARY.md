---
phase: 02-config-foundation
plan: 01
subsystem: config
tags: [toml, go, burtonsushi-toml, xdg, configuration]

# Dependency graph
requires:
  - phase: 01-project-scaffold
    provides: go.mod module path, cobra command stubs, project structure
provides:
  - Typed config structs (RepoConfig, GlobalConfig, MergedConfig, AnnotatedConfig)
  - HarnessConfig typed struct with all per-harness fields
  - LoadRepo() TOML parser with two-phase harness sugar handling
  - LoadGlobal() with XDG-compliant path and missing-file-as-defaults behavior
  - UnknownKeysError type for strict unknown-key detection
  - Source type and constants (SourceDefault, SourceGlobal, SourceRepo)
  - DefaultRepoConfig() and DefaultGlobalConfig() with spec-defined defaults
affects: [03-config-merge, 04-config-validation, 05-dockerfile-templating, 06-docker-lifecycle, 07-harness-plugins, 08-init-wizard]

# Tech tracking
tech-stack:
  added:
    - github.com/BurntSushi/toml v1.6.0 — TOML parsing with MetaData.Undecoded() for strict mode
    - github.com/agnivade/levenshtein v1.2.1 — edit-distance suggestions (used in Plan 03 validation)
  patterns:
    - Two-phase TOML decode: primary decode expects [harness] table; on type error, fallback to sugar struct with harness as string
    - HarnessName toml:"-" field: populated from sugar/[zone].harness after decode, never decoded directly from TOML
    - XDG path: manual $XDG_CONFIG_HOME check avoids os.UserConfigDir() macOS ~/Library/Application Support pitfall
    - Missing global config returns defaults (not error): zero-friction first-run behavior
    - *bool fields for nullable booleans: nil=not-set vs false=explicitly-disabled (enables merge semantics)
    - AnnotatedField[T] generic for source-tracked config display

key-files:
  created:
    - internal/config/types.go — All config structs, Source type, AnnotatedField/AnnotatedListItem generics, DefaultRepoConfig/DefaultGlobalConfig
    - internal/config/harness_config.go — HarnessConfig with typed fields for all harness types
    - internal/config/config.go — LoadRepo() with harness sugar handling, UnknownKeysError, sentinel errors, version validation
    - internal/config/global.go — GlobalConfigPath() XDG-compliant, LoadGlobal() with missing-file defaults
  modified:
    - go.mod — Added BurntSushi/toml and agnivade/levenshtein
    - go.sum — Updated checksums

key-decisions:
  - "Two-phase TOML decode for harness sugar: primary decode expects HarnessConfig struct; isHarnessTypeError() detects string/table conflict and triggers repoConfigSugar fallback"
  - "HarnessName toml:\"-\" pattern: avoids TOML key conflict between top-level harness string and [harness] table by using a non-decoded field populated post-parse"
  - "Explicit XDG path over os.UserConfigDir(): macOS returns ~/Library/Application Support which violates XDG expectations for a CLI tool"
  - "levenshtein added now, used in Plan 03: kept in go.mod as indirect to maintain explicit version pinning before validation is implemented"
  - "*bool for nullable booleans (SkipPermissions, MountHomeConfig, ForwardSSHAgent, PersistHome): distinguishes nil=not-set from false=explicitly-disabled for merge semantics"

patterns-established:
  - "Two-phase TOML decode: use primary struct; on harness type error, fall back to sugar struct, then normalise"
  - "Post-decode normalisation: HarnessName field resolved from sugar or [zone].harness after decode completes"
  - "XDG path: always use $XDG_CONFIG_HOME first, then ~/.config — never os.UserConfigDir()"
  - "Missing file = defaults: config loading functions return defaults (not errors) when optional files absent"

requirements-completed: [CFG-01, CFG-02, CFG-09]

# Metrics
duration: 4min
completed: 2026-03-27
---

# Phase 02 Plan 01: Config Types and TOML Parsing Summary

**Typed config structs and strict TOML parsing for zone.toml (per-repo) and ~/.config/zone/config.toml (global) using BurntSushi/toml with two-phase harness sugar decode**

## Performance

- **Duration:** ~4 min
- **Started:** 2026-03-27T07:00:53Z
- **Completed:** 2026-03-27T07:04:10Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- All config section structs defined with TOML tags matching zone-spec.md sections 4.1-4.2
- HarnessConfig is a fully typed struct (not map[string]interface{}) per spec section 5
- LoadRepo() handles both `harness = "claude-code"` sugar and `[harness]` table forms via two-phase decode
- LoadGlobal() returns DefaultGlobalConfig() when file is missing, enabling zero-friction first-run
- Unknown TOML keys detected via MetaData.Undecoded() and surfaced as UnknownKeysError
- Version field validated: missing defaults to 1, unsupported version returns ErrVersionMismatch
- Package compiles cleanly with zero vet warnings

## Task Commits

Each task was committed atomically:

1. **Task 1: Install dependencies and define config type structs** - `361f98c` (feat)
2. **Task 2: Implement TOML parsing for repo and global config** - `d6696e1` (feat)

**Plan metadata:** _(pending docs commit)_

## Files Created/Modified

- `/workspace/zone/internal/config/types.go` - All config structs, Source type, AnnotatedField[T] generic, DefaultRepoConfig/DefaultGlobalConfig
- `/workspace/zone/internal/config/harness_config.go` - HarnessConfig with *bool SkipPermissions and all per-harness typed fields
- `/workspace/zone/internal/config/config.go` - LoadRepo() with two-phase harness sugar handling, version validation, UnknownKeysError
- `/workspace/zone/internal/config/global.go` - GlobalConfigPath() XDG-compliant, LoadGlobal() with missing-file-as-defaults
- `/workspace/zone/go.mod` - Added BurntSushi/toml v1.6.0 and agnivade/levenshtein v1.2.1
- `/workspace/zone/go.sum` - Updated checksums

## Decisions Made

- **Two-phase TOML decode for harness sugar:** The `harness` key in TOML is ambiguous: `harness = "claude-code"` (string) and `[harness]` (table) conflict in the same file. Solution: primary decode into RepoConfig (where Harness is HarnessConfig struct); if that fails with a string/table type error, fall back to `repoConfigSugar` (where Harness is string), then copy fields over.
- **HarnessName toml:"-" pattern:** The `HarnessName` field uses `toml:"-"` so it is never decoded from TOML directly. It is populated post-decode from either the sugar string or `[zone].harness`. This cleanly separates the TOML decode concern from the harness name resolution concern.
- **Explicit XDG path over os.UserConfigDir():** macOS `os.UserConfigDir()` returns `~/Library/Application Support`, which is wrong for a cross-platform CLI tool expecting `~/.config`. Per research notes in 02-RESEARCH.md, the XDG path is constructed manually.
- **levenshtein added now, used in Plan 03:** The levenshtein library is already in go.mod as indirect to maintain explicit version pinning before config validation (Plan 03) implements the edit-distance suggestions for unknown keys.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `go mod tidy` removed `agnivade/levenshtein` as unused after initial install since the library is not yet imported in the config package. Re-added with `go get` to maintain the explicit version pin per plan requirements. It will become a direct dependency in Plan 03.

## Next Phase Readiness

- All config type structs available for Plan 02 (merge.go) and Plan 03 (validation)
- `LoadRepo()` and `LoadGlobal()` are the only missing dependencies for the merge layer
- `Merge()` function stub intentionally omitted from this plan (belongs in Plan 02 alongside merge semantics)
- levenshtein v1.2.1 already pinned for Plan 03 edit-distance suggestions on unknown keys

## Self-Check: PASSED

- internal/config/types.go: FOUND
- internal/config/harness_config.go: FOUND
- internal/config/config.go: FOUND
- internal/config/global.go: FOUND
- .planning/phases/02-config-foundation/02-01-SUMMARY.md: FOUND
- Commit 361f98c (Task 1): FOUND
- Commit d6696e1 (Task 2): FOUND

---
*Phase: 02-config-foundation*
*Completed: 2026-03-27*
