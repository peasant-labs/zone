---
phase: 05-harness-plugin-system
plan: 03
subsystem: docker
tags: [go, bridge, harness, template, tdd, integration]

# Dependency graph
requires:
  - phase: 05-harness-plugin-system
    provides: Harness interface with 19 methods (05-01), all 6 harness implementations (05-02)
  - phase: 04-template-system
    provides: DockerfileData, EntrypointData, ShellRCData structs + render functions
  - phase: 02-config-foundation
    provides: MergedConfig with Zone, Workspace, Packages, Harness fields
provides:
  - BuildDockerfileData(h Harness, cfg *MergedConfig) DockerfileData
  - BuildEntrypointData(h Harness, cfg *MergedConfig) EntrypointData
  - BuildShellRCData(h Harness, cfg *MergedConfig) ShellRCData
  - mergeSlices: config-first package merging helper
  - configCopyCmd: copy-on-start shell command generator for Phase 7 volume strategy
affects: [06-container-lifecycle]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Bridge pattern: single integration point between harness system and template rendering
    - Copy-on-start strategy: Phase 7 mounts host dirs at <dir>.host; entrypoint copies to <dir>
    - Config-first merge: user-specified packages appear before harness defaults in merged lists

key-files:
  created:
    - internal/docker/harness_bridge.go
    - tests/harness_bridge_test.go
  modified: []

key-decisions:
  - "NodeVersion/PythonVersion come from cfg.Harness (MergedConfig), not from harness interface methods — harnesses express capability (NeedsNode bool), not versions"
  - "HostUID and MacOSUsername are NOT set by bridge — they are runtime values requiring os/user lookups; Phase 6 sets them after calling BuildDockerfileData"
  - "configCopyCmd uses .host suffix pattern — Phase 7 mounts host config dirs at <dir>.host; entrypoint copies at startup to <dir>"
  - "mergeSlices returns nil for both-empty case — preserves nil semantics for template rendering (no empty slice rendered)"
  - "InstallZsh derived from cfg.Zone.Shell == zsh — not a harness concern; shells are config-level"

patterns-established:
  - "Pattern: Bridge functions — call harness methods and populate template data structs; keeps harness-to-template logic centralized"
  - "Pattern: Config-first mergeSlices — cfg.Packages.Apt merged with h.DefaultAptPackages() in config-first order"

requirements-completed: [HAR-01, HAR-04, HAR-06, HAR-09]

# Metrics
duration: 1min
completed: 2026-03-29
---

# Phase 05 Plan 03: Harness Bridge Functions Summary

**Three bridge functions (BuildDockerfileData, BuildEntrypointData, BuildShellRCData) translating Harness interface methods + MergedConfig into template data structs; NodeVersion defaults to "22", PythonVersion to "3.12"; mergeSlices for config-first package merging; configCopyCmd for copy-on-start volume strategy; 12 integration tests green**

## Performance

- **Duration:** 1 min
- **Started:** 2026-03-29T19:27:48Z
- **Completed:** 2026-03-29T19:29:00Z
- **Tasks:** 1 (TDD: 2 commits — test RED + feat GREEN)
- **Files modified:** 2

## Accomplishments
- BuildDockerfileData: all DockerfileData fields populated from harness + MergedConfig; NodeVersion defaults "22", PythonVersion defaults "3.12"; package lists merged config-first; InstallZsh derived from Shell=="zsh"; HostUID/MacOSUsername intentionally left for Phase 6 (runtime values)
- BuildEntrypointData: HomeConfigDir + ExtraConfigDirs translated to ConfigCopyCommands via configCopyCmd; DetectGitIdentity() called for git forwarding; EntrypointCommand, Shell, MountPath populated
- BuildShellRCData: thin translation of HarnessName, Aliases, ShellRC, WelcomeMessage from harness methods
- mergeSlices: config-first ordering, nil-safe, returns nil for both-empty inputs
- configCopyCmd: generates copy-on-start shell command using Phase 7's .host suffix mount strategy
- 12 integration tests: cover all three Build* functions, NodeVersion override, PythonVersion default, package merging, InstallZsh, ConfigCopyCommands for 0/1/2 dirs, custom harness with Aliases/ShellRC

## Task Commits

Each TDD phase was committed atomically:

1. **Task 1 RED: failing tests** - `19e171a` (test)
2. **Task 1 GREEN: implementation** - `91c3600` (feat)

_Note: TDD tasks have two commits (test RED -> feat GREEN)_

## Files Created/Modified
- `/workspace/zone/internal/docker/harness_bridge.go` - BuildDockerfileData, BuildEntrypointData, BuildShellRCData, mergeSlices, configCopyCmd
- `/workspace/zone/tests/harness_bridge_test.go` - 12 integration tests covering all three Build* functions and edge cases

## Decisions Made
- NodeVersion and PythonVersion are NOT harness methods — they come from `MergedConfig.Harness.NodeVersion` / `.PythonVersion`. Harnesses only express capability via `NeedsNode() bool` and `NeedsPython() bool`. Version strings are user config concerns.
- HostUID and MacOSUsername are intentionally NOT set by the bridge. These require `os/user` lookups that depend on the host runtime environment. Phase 6 calls `HostUID()` and `MacOSUsername()` and sets these fields after `BuildDockerfileData` returns.
- configCopyCmd generates: `mkdir -p $(dirname <dir>) && cp -r <dir>.host <dir> 2>/dev/null || true`. Phase 7 mounts host config dirs at `<dir>.host`; the entrypoint copies them to `<dir>` at startup. The `|| true` prevents missing host dirs from aborting the entrypoint.
- mergeSlices returns nil (not empty slice) when both inputs are empty. This preserves nil semantics so template rendering can distinguish "no packages" from "empty packages list".

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All three Build* functions are ready for Phase 6 (container lifecycle: zone launch)
- Phase 6 callers: call BuildDockerfileData, set HostUID and MacOSUsername, then call RenderDockerfile
- Import graph is correct: internal/docker imports internal/harness (not reverse)
- configCopyCmd output format is established — Phase 7 must mount host dirs at `<dir>.host` to match

---
*Phase: 05-harness-plugin-system*
*Completed: 2026-03-29*

## Self-Check: PASSED
- FOUND: internal/docker/harness_bridge.go
- FOUND: tests/harness_bridge_test.go
- FOUND: .planning/phases/05-harness-plugin-system/05-03-SUMMARY.md
- FOUND commit: 19e171a (test RED)
- FOUND commit: 91c3600 (feat GREEN)
