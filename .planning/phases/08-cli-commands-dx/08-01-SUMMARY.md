---
phase: 08-cli-commands-dx
plan: 01
subsystem: cli
tags: [cobra, docker-sdk, cli, json-output]
requires:
  - phase: 01-foundation
    provides: command wiring and stub structure for init/ls/logs/status
  - phase: 03-cache-state
    provides: .zone cache and build log file handling
  - phase: 07-env-auth-proxy-hooks
    provides: manager lifecycle patterns and merged config loading
provides:
  - init command implementation with harness selection, template generation, and --set overrides
  - global container listing via Docker labels with table/json/quiet output
  - logs and status command implementations backed by Manager.Logs and Manager.Status
  - Docker client interface extensions for ContainerList and ContainerLogs
affects: [phase-09-tui-polish, cli-ux, integration-tests]
tech-stack:
  added: []
  patterns: [cmd->config/cache->docker.Manager flow, Docker label-based discovery, dual plain/json CLI output]
key-files:
  created: [tests/cli_commands_test.go]
  modified:
    - cmd/init.go
    - cmd/ls.go
    - cmd/logs.go
    - cmd/status.go
    - internal/docker/client_interface.go
    - internal/docker/errors.go
    - internal/docker/manager.go
    - internal/docker/manager_test.go
key-decisions:
  - "Implemented zone ls without zone.toml dependency by adding docker.ListContainers(client) helper and delegating Manager.List to it"
  - "Kept logs --build independent from Docker by reading .zone/logs/last_build.log directly"
patterns-established:
  - "Command implementations return concrete operational errors instead of stub sentinels"
  - "Commands that support automation expose structured --json output alongside human-readable plain output"
requirements-completed: [CLI-01, CLI-02, CLI-12, CLI-13, CLI-14, CLI-17]
duration: 6m
completed: 2026-03-30
---

# Phase 8 Plan 1: CLI Commands DX Summary

**Init/ls/logs/status now execute real Docker- and cache-backed flows with plain/JSON output instead of returning stub errors.**

## Performance

- **Duration:** 6m
- **Started:** 2026-03-30T22:33:57Z
- **Completed:** 2026-03-30T22:40:13Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- Replaced `zone init` stub with functional scaffolding: harness requirement, existing-config guard, detection hints, template generation, and `--set` overrides.
- Extended Docker abstractions with `ContainerList` and `ContainerLogs`, then implemented `Manager.List`, `Manager.Logs`, and `Manager.Status`.
- Replaced `zone ls`, `zone logs`, and `zone status` stubs with full command bodies and added integration tests confirming no stub behavior remains.

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend DockerClient interface + add Manager.List/Logs/Status + ErrNetworkUnsupported + init command** - `0c17d4f` (feat)
2. **Task 2: Implement ls, logs, status command bodies + integration tests** - `c0a2581` (feat)

## Files Created/Modified
- `cmd/init.go` - Full `zone init` implementation with harness flag, template generation, and `--set` overrides.
- `internal/docker/client_interface.go` - Added `ContainerList` and `ContainerLogs` to DockerClient contract.
- `internal/docker/errors.go` - Added `ErrNetworkUnsupported` sentinel.
- `internal/docker/manager.go` - Added container listing/log/status methods and global `ListContainers` helper.
- `internal/docker/manager_test.go` - Extended mock client to satisfy newly added interface methods.
- `cmd/ls.go` - Implemented container list output modes (`table`, `--json`, `--quiet`, `--running`).
- `cmd/logs.go` - Implemented build-log file view and live container log streaming options.
- `cmd/status.go` - Implemented plain and JSON container inspection output.
- `tests/cli_commands_test.go` - Added integration coverage for init behavior and non-stub ls/logs/status behavior.

## Decisions Made
- Added a package-level `docker.ListContainers` helper so `zone ls` can work without repo config while still reusing Manager data mapping.
- Kept `logs --build` as a direct cache-file read path to avoid unnecessary Docker dependency for historical build output.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Commit identity missing in executor environment**
- **Found during:** Task 1 commit step
- **Issue:** Git commit failed with "Author identity unknown" due missing configured `user.name`/`user.email`.
- **Fix:** Used per-command `GIT_AUTHOR_*`/`GIT_COMMITTER_*` environment variables to create commits without mutating git config.
- **Files modified:** None (execution environment only)
- **Verification:** Task commits `0c17d4f` and `c0a2581` created successfully.
- **Committed in:** N/A (process-level fix)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** No scope creep; workaround only affected commit execution mechanics.

## Issues Encountered
- Git in this executor lacked commit identity defaults. Resolved non-invasively via command-scoped author/committer environment variables.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All previously stubbed remaining CLI commands now have concrete implementations.
- Ready for Phase 08 follow-up plans and Phase 09 TUI work, which can now rely on complete CLI command coverage.

## Self-Check: PASSED

- FOUND: .planning/phases/08-cli-commands-dx/08-01-SUMMARY.md
- FOUND: 0c17d4f
- FOUND: c0a2581
