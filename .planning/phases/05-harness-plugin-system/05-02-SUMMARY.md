---
phase: 05-harness-plugin-system
plan: 02
subsystem: harness
tags: [go, stub, plugin, validation, tdd, cross-harness]

# Dependency graph
requires:
  - phase: 05-harness-plugin-system
    provides: Harness interface, BaseHarness, registry, Get(), placeholder stubs in harness.go
  - phase: 02-config-foundation
    provides: HarnessConfig typed struct (internal/config/harness_config.go)
provides:
  - OpenCode stub harness in own file with cross-harness key rejection
  - GeminiCLI stub harness in own file with cross-harness key rejection
  - Aider stub harness in own file (owns python_version, rejects skip_permissions and custom keys)
  - CodexCLI stub harness in own file with cross-harness key rejection
  - Custom harness reading all 7 config fields (InstallCommands, EntrypointCommand, HealthCheck, ConfigDirs, RequiredEnv, CustomAliases, CustomShellRC)
  - Custom.Validate() rejects skip_permissions (claude-code specific) and requires entrypoint_command
  - All stubs check cross-harness keys BEFORE returning stub error (priority ordering)
  - Placeholder type definitions removed from harness.go
affects: [05-03-harness-bridge, 06-container-lifecycle]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Cross-harness key validation pattern: check foreign keys before returning own error (priority ordering)
    - Custom harness pass-through pattern: config fields map directly to interface method returns
    - Stub-with-validation pattern: stubs still do meaningful key validation even though they fail ultimately

key-files:
  created:
    - internal/harness/opencode.go
    - internal/harness/gemini_cli.go
    - internal/harness/aider.go
    - internal/harness/codex_cli.go
    - internal/harness/custom.go
    - tests/harness_validate_test.go
  modified:
    - internal/harness/harness.go

key-decisions:
  - "Cross-harness validation order: foreign-key errors reported BEFORE stub 'not implemented' error — user sees specific error not generic stub message"
  - "Aider owns python_version and does NOT reject it; all other stubs reject it as aider-specific"
  - "Custom.Validate() checks skip_permissions BEFORE entrypoint_command — cross-harness keys are a config mistake, missing required field is a usage mistake"
  - "Placeholder types removed from harness.go; each harness now lives in its own file"

patterns-established:
  - "Pattern: Stub-with-validation — stubs validate cross-harness keys even though they always fail Validate()"
  - "Pattern: Custom pass-through — custom harness is a thin adapter mapping HarnessConfig fields to Harness interface"

requirements-completed: [HAR-05, HAR-06, HAR-07]

# Metrics
duration: 3min
completed: 2026-03-29
---

# Phase 05 Plan 02: Stub Harnesses and Custom Harness Summary

**Four stub harnesses (opencode, gemini-cli, aider, codex-cli) each in own file with cross-harness key rejection before "not yet implemented" error; custom harness reads all 7 HarnessConfig fields with required entrypoint_command validation; 19 tests green**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-29T19:22:29Z
- **Completed:** 2026-03-29T19:25:20Z
- **Tasks:** 1 (TDD: 2 commits — test RED + feat GREEN)
- **Files modified:** 7

## Accomplishments
- All four stub harnesses moved to individual files with cross-harness key rejection in priority order (foreign keys reported before stub error)
- Custom harness implements all 13 interface methods; 7 pass-through directly from HarnessConfig fields
- Custom.Validate() rejects skip_permissions (claude-code-specific) and requires entrypoint_command
- Placeholder type definitions removed from harness.go — clean separation of concerns
- 19 tests covering stub errors, custom field pass-through, and cross-harness key rejection all pass

## Task Commits

Each TDD phase was committed atomically:

1. **Task 1 RED: failing tests** - `8390b1c` (test)
2. **Task 1 GREEN: implementation** - `d0c49b8` (feat)

**Plan metadata:** (final docs commit — see below)

_Note: TDD tasks have two commits (test RED → feat GREEN)_

## Files Created/Modified
- `/workspace/zone/internal/harness/opencode.go` - OpenCode stub: 10 interface methods + Validate() with cross-harness check then stub error
- `/workspace/zone/internal/harness/gemini_cli.go` - GeminiCLI stub: identical pattern to opencode
- `/workspace/zone/internal/harness/aider.go` - Aider stub: rejects skip_permissions and custom keys; allows python_version (aider-owned)
- `/workspace/zone/internal/harness/codex_cli.go` - CodexCLI stub: identical pattern to opencode
- `/workspace/zone/internal/harness/custom.go` - Custom harness: HealthCheck/ExtraConfigDirs/ShellRC/Aliases read from config; Validate() checks skip_permissions then entrypoint_command
- `/workspace/zone/internal/harness/harness.go` - Removed all placeholder type definitions (130 lines removed)
- `/workspace/zone/tests/harness_validate_test.go` - 19 tests: TestStubHarnessValidate* x4, TestCustomHarness* x12, TestCrossHarnessKey* x3

## Decisions Made
- Cross-harness validation order: foreign-key errors are reported BEFORE the stub "not implemented" error. This gives users the precise actionable error (e.g., "python_version is specific to aider") rather than the generic stub message.
- Aider owns `python_version` and does NOT reject it — even though it's a stub that fails Validate(), the key itself is valid for aider.
- Custom harness checks `skip_permissions` before `entrypoint_command` — a cross-harness key is always a config mistake regardless of other fields.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All 6 harnesses (claude-code + 4 stubs + custom) are registered and validate properly
- Get() works for all harness types — Plan 03 (harness bridge: BuildDockerfileData/BuildEntrypointData/BuildShellRCData) can proceed
- Custom harness provides full config-driven behavior for user-defined tooling

---
*Phase: 05-harness-plugin-system*
*Completed: 2026-03-29*

## Self-Check: PASSED
- FOUND: internal/harness/opencode.go
- FOUND: internal/harness/gemini_cli.go
- FOUND: internal/harness/aider.go
- FOUND: internal/harness/codex_cli.go
- FOUND: internal/harness/custom.go
- FOUND: tests/harness_validate_test.go
- FOUND: .planning/phases/05-harness-plugin-system/05-02-SUMMARY.md
- FOUND commit: 8390b1c (test RED)
- FOUND commit: d0c49b8 (feat GREEN)
