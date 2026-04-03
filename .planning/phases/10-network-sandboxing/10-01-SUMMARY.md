---
phase: 10-network-sandboxing
plan: 01
subsystem: infra
tags: [docker, iptables, network, glob, testing]
requires:
  - phase: 09-tui-layer
    provides: Docker lifecycle, CLI error mapping, and test scaffolding used by network sandboxing foundations
provides:
  - Platform detection for Linux, macOS, Docker Desktop, and rootless Docker
  - Hostname glob compilation and matching for network allow/deny rules
  - Firewall sentinel errors with exit code 4 remediation mappings
  - Regression tests for matcher behavior and network platform fallbacks
affects: [network-sandboxing, launch, firewall, error-handling]
tech-stack:
  added: []
  patterns: [Docker client Info-based platform detection, filepath.Match hostname globbing, sentinel error mapping through cmd.MapError]
key-files:
  created: [tests/network_platform_test.go]
  modified: [internal/docker/client_interface.go, internal/docker/platform.go, internal/docker/errors.go, internal/docker/manager.go, internal/docker/manager_test.go, cmd/errors.go, internal/network/matcher.go, tests/matcher_test.go]
key-decisions:
  - "Platform detection runs once in Manager construction and stores a Platform snapshot for later firewall lifecycle decisions."
  - "Hostname rules accept only literals and simple single-label globs, while filepath.Match handles actual wildcard matching."
  - "Firewall setup, sudo, and iptables failures all map to exit code 4 with actionable remediation hints."
patterns-established:
  - "Platform capability checks should use DockerClient.Info rather than ad hoc OS probing."
  - "Network matcher validation rejects unsupported glob syntax up front before runtime rule generation."
requirements-completed: [NET-04, NET-07, NET-09, NET-11, NET-12]
duration: 2 min
completed: 2026-04-03
---

# Phase 10 Plan 01: Network Sandboxing Summary

**Platform-aware network sandboxing foundations with Docker Info detection, strict hostname glob matching, and firewall error propagation.**

## Performance

- **Duration:** 2 min
- **Started:** 2026-04-03T04:19:49Z
- **Completed:** 2026-04-03T04:21:54Z
- **Tasks:** 3
- **Files modified:** 9

## Accomplishments
- Extended the Docker client abstraction and manager state with a reusable `Platform` snapshot for later firewall integration.
- Implemented literal/simple-glob hostname matching with validation that rejects unsupported patterns before rule generation.
- Added firewall-specific sentinel errors, exit-code mappings, and regression coverage for matcher and platform fallback behavior.

## Task Commits

Each task was committed atomically:

1. **Task 1: Platform detection, DockerClient.Info(), sentinel errors, Manager.platform field** - `8458209` (feat)
2. **Task 2: Hostname glob matcher with tests** - `5721028` (test), `215aae4` (feat)
3. **Task 3: Platform detection and error mapping tests** - `a0d3a9b` (test)

**Plan metadata:** Pending

## Files Created/Modified
- `internal/docker/client_interface.go` - adds `Info(ctx)` to the mockable Docker client contract.
- `internal/docker/platform.go` - defines `Platform`, `DetectPlatform`, and iptables availability probing.
- `internal/docker/errors.go` - adds firewall/sudo/iptables sentinel errors.
- `internal/docker/manager.go` - stores detected platform on `Manager` during construction.
- `internal/docker/manager_test.go` - adds Docker Info mocking and stabilizes launch env validation coverage.
- `cmd/errors.go` - maps new firewall sentinels to exit code 4 remediation messages.
- `internal/network/matcher.go` - implements hostname pattern compilation and matching.
- `tests/matcher_test.go` - locks matcher compile/match behavior with TDD coverage.
- `tests/network_platform_test.go` - verifies platform fallback invariants and firewall error mappings.

## Decisions Made
- Used `DockerClient.Info()` for rootless and Docker Desktop detection so network capability checks stay tied to the daemon, not host heuristics.
- Restricted wildcard support to simple hostname globs and enforced same-label matching so `*.anthropic.com` matches `api.anthropic.com` but not `sub.api.anthropic.com`.
- Kept firewall setup failures in the existing network exit-code bucket so CLI remediation remains consistent.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Stabilized launch env validation tests against harness drift**
- **Found during:** Task 1 verification
- **Issue:** `go test ./internal/docker/...` failed because unrelated dirty harness changes made `claude-code` stop requiring `ANTHROPIC_API_KEY`, invalidating an older launch validation assumption.
- **Fix:** Updated `internal/docker/manager_test.go` so the required-env validation path uses a custom harness with explicit `required_env`, preserving the docker package's intended validation coverage without touching unrelated harness files.
- **Files modified:** `internal/docker/manager_test.go`
- **Verification:** `go test ./internal/docker/... -count=1`
- **Committed in:** `8458209`

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Kept task-scoped verification reliable without expanding scope into unrelated dirty files.

## Issues Encountered
- Task 3's TDD RED step passed immediately because Task 1 had already implemented the underlying platform/error behavior; the task completed as regression-test coverage rather than a failing-first cycle.
- `go test ./... -count=1` still fails in unrelated dirty-worktree test `tests/cli_dx_test.go:TestUnknownKeysRemediationHintOnStderr`; logged in `deferred-items.md` per scope boundary.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 10 now has the platform, matcher, and error-handling primitives needed for firewall rule generation in Plan 10-02.
- Deferred unrelated CLI DX regression should be resolved separately before relying on full-suite cleanliness outside this plan.

## Self-Check: PASSED

---
*Phase: 10-network-sandboxing*
*Completed: 2026-04-03*
