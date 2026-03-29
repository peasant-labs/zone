---
phase: 05-harness-plugin-system
plan: 01
subsystem: harness
tags: [go, interface, registry, plugin, claude-code, tdd]

# Dependency graph
requires:
  - phase: 02-config-foundation
    provides: HarnessConfig typed struct (internal/config/harness_config.go)
provides:
  - Harness interface with 19 methods (identity, install, runtime, deps, shell, lifecycle)
  - BaseHarness struct with 9 no-op defaults via embedding
  - Factory registry mapping 6 harness names to constructors
  - Get(name, cfg) constructs and validates harnesses; returns sorted "available:" on unknown
  - ClaudeCode fully implemented with all spec-prescribed return values
  - ClaudeCode.Validate() rejects 8 cross-harness keys with specific error messages
  - Placeholder stubs for OpenCode, GeminiCLI, Aider, CodexCLI, Custom (Plan 02 replaces)
affects: [05-02-other-harnesses, 05-03-harness-bridge, 06-container-lifecycle]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - BaseHarness embedding pattern for optional method defaults
    - Factory registry map with Get() constructor+validate flow
    - Cross-harness key validation with specific "specific to X" error messages

key-files:
  created:
    - internal/harness/harness.go
    - internal/harness/claude_code.go
    - tests/harness_registry_test.go
    - tests/harness_claude_code_test.go
  modified: []

key-decisions:
  - "NodeVersion/PythonVersion are NOT Harness interface methods — they come from MergedConfig.Harness (RESEARCH.md anti-patterns)"
  - "Placeholder stubs in harness.go for Plan 02 types keep this plan compilable independently"
  - "Get() wraps Validate() error: harness 'X' config: <original error> — clean separation of construction vs validation errors"
  - "HAR-08 (HarnessConfig typed struct) confirmed pre-satisfied by Phase 02"

patterns-established:
  - "Pattern: BaseHarness embedding — embed in concrete structs, override only what differs"
  - "Pattern: Registry+Get — all access via Get(); never construct concrete types directly from outside package"
  - "Pattern: Validate-on-construction — Get() always calls Validate() before returning"

requirements-completed: [HAR-01, HAR-02, HAR-03, HAR-04, HAR-08, HAR-09, HAR-10]

# Metrics
duration: 2min
completed: 2026-03-29
---

# Phase 05 Plan 01: Harness Interface, Registry, and ClaudeCode Summary

**Harness plugin system foundation: 19-method Go interface with BaseHarness embedding, 6-harness factory registry with Get()+validate, and fully implemented ClaudeCode with versioned npm install, -p prompt flag, and 8 cross-harness key rejections**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-29T19:18:12Z
- **Completed:** 2026-03-29T19:20:18Z
- **Tasks:** 1 (TDD: 2 commits — test + feat)
- **Files modified:** 4

## Accomplishments
- Harness interface with exactly 19 methods as specified (NodeVersion excluded per RESEARCH.md anti-patterns)
- BaseHarness provides 9 no-op defaults; Go embedding lets concrete structs override only what they need
- Factory registry maps all 6 harness names; Get() constructs, validates, and returns sorted available names on error
- ClaudeCode fully implements all interface methods with spec-prescribed values; Validate() checks 8 cross-harness keys
- Placeholder structs for OpenCode, GeminiCLI, Aider, CodexCLI, Custom in harness.go to allow independent compilation; Plan 02 replaces these

## Task Commits

Each TDD phase was committed atomically:

1. **Task 1 RED: failing tests** - `6f95693` (test)
2. **Task 1 GREEN: implementation** - `4a639eb` (feat)

**Plan metadata:** (final docs commit — see below)

_Note: TDD tasks have two commits (test RED → feat GREEN)_

## Files Created/Modified
- `/workspace/zone/internal/harness/harness.go` - Harness interface, BaseHarness, registry, Get(), availableNames(), placeholder stubs
- `/workspace/zone/internal/harness/claude_code.go` - ClaudeCode struct: all 19 methods + cross-harness Validate()
- `/workspace/zone/tests/harness_registry_test.go` - TestHarnessInterface, TestBaseHarnessDefaults, TestHarnessRegistryGet/Unknown, TestRegistryAllNames
- `/workspace/zone/tests/harness_claude_code_test.go` - 15 ClaudeCode tests covering all methods and Validate() branches

## Decisions Made
- NodeVersion/PythonVersion are NOT Harness interface methods. The spec interface has `NeedsNode() bool` and `NeedsPython() bool`. Version strings come from `MergedConfig.Harness.NodeVersion` and are handled by the bridge function in Plan 03.
- Placeholder stubs (OpenCode, GeminiCLI, Aider, CodexCLI, Custom) included in harness.go so this plan compiles independently. Plan 02 will replace them.
- HAR-08 (HarnessConfig is a typed struct) confirmed pre-satisfied by Phase 02's internal/config/harness_config.go.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Harness interface and ClaudeCode are ready for Plan 02 (other harnesses: opencode, gemini-cli, aider, codex-cli, custom)
- Plan 03 (harness bridge: BuildDockerfileData/BuildEntrypointData/BuildShellRCData) can consume the Harness interface now
- Placeholder stubs are annotated with `// TODO(plan-02)` comments for easy discovery

---
*Phase: 05-harness-plugin-system*
*Completed: 2026-03-29*

## Self-Check: PASSED
- FOUND: internal/harness/harness.go
- FOUND: internal/harness/claude_code.go
- FOUND: tests/harness_registry_test.go
- FOUND: tests/harness_claude_code_test.go
- FOUND: .planning/phases/05-harness-plugin-system/05-01-SUMMARY.md
- FOUND commit: 6f95693 (test RED)
- FOUND commit: 4a639eb (feat GREEN)
