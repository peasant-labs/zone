---
phase: 10-network-sandboxing
plan: 03
subsystem: infra
tags: [docker, firewall, iptables, lifecycle, proxy]
requires:
  - phase: 10-network-sandboxing
    provides: Firewall primitives, rule generation, and platform fallbacks from plans 10-01 and 10-02
provides:
  - Docker manager lifecycle integration for firewall apply, refresh, and cleanup
  - Bridge interface discovery and proxy hostname extraction for runtime allowlisting
  - Stale firewall rule cleanup coverage tied to running zone container hashes
affects: [network-sandboxing, launch, stop, destroy, proxy]
tech-stack:
  added: []
  patterns: [Manager-owned firewall lifecycle state, post-start firewall application, proxy auto-allowlisting from effective proxy config]
key-files:
  created: []
  modified: [internal/docker/client_interface.go, internal/docker/network.go, internal/docker/launch.go, internal/docker/manager.go, internal/docker/manager_test.go]
key-decisions:
  - "Firewall setup runs after container start and stores refresh cancellation on Manager so stop/destroy can tear rules down before network removal."
  - "Proxy hostnames are extracted from the effective proxy configuration, including auto-detected host proxy env vars, before building whitelist rules."
patterns-established:
  - "Docker lifecycle features that depend on runtime network IDs should inspect the network after creation instead of guessing bridge names."
  - "Manager cleanup should cancel background goroutines before removing external resources they mutate."
requirements-completed: [NET-06, NET-02, NET-03, NET-04, NET-05, NET-09, NET-11]
duration: 11 min
completed: 2026-04-03
---

# Phase 10 Plan 03: Network Sandboxing Summary

**Docker manager firewall lifecycle integration with bridge inspection, stale-rule cleanup, and proxy-aware whitelist expansion.**

## Performance

- **Duration:** 11 min
- **Started:** 2026-04-03T04:29:00Z
- **Completed:** 2026-04-03T04:40:03Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Added bridge interface discovery and proxy hostname parsing helpers needed for real firewall attachment.
- Wired firewall apply, refresh cancellation, and cleanup into launch/stop manager lifecycle paths.
- Added stale-rule cleanup coverage that verifies only non-running container hashes are deleted.

## Task Commits

Each task was committed atomically:

1. **Task 1: Bridge interface name discovery and proxy hostname extraction** - `6a75677` (feat)
2. **Task 2: Integrate firewall lifecycle into Manager Launch, Stop, and Destroy with tests** - `82fa3c9` (feat)

**Plan metadata:** Pending

## Files Created/Modified
- `internal/docker/client_interface.go` - exposes `NetworkInspect` so manager lifecycle code can derive bridge interfaces from Docker.
- `internal/docker/network.go` - adds `BridgeInterfaceName` and `extractProxyHostnames` helpers.
- `internal/docker/launch.go` - applies firewall rules after container start, handles platform fallbacks, and launches stale-rule cleanup plus refresh.
- `internal/docker/manager.go` - stores firewall state on `Manager` and removes rules before tearing down the Docker network.
- `internal/docker/manager_test.go` - extends the Docker mock and verifies stale cleanup only deletes dead hashes.

## Decisions Made
- Stored firewall state and cancelation directly on `Manager` so lifecycle teardown stays aligned with existing stop/destroy orchestration.
- Resolved effective proxy settings before extracting proxy hostnames so auto-detected `HTTP_PROXY`/`HTTPS_PROXY` values are allowlisted in whitelist mode too.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added `NetworkInspect` to the Docker client interface**
- **Found during:** Task 1
- **Issue:** The plan expected `DockerClient.NetworkInspect`, but the interface in `client_interface.go` did not expose it yet, so the new bridge helper would not compile.
- **Fix:** Added `NetworkInspect` to the shared Docker client contract and used it in the new bridge interface helper.
- **Files modified:** `internal/docker/client_interface.go`, `internal/docker/network.go`
- **Verification:** `go build ./internal/docker/...`
- **Committed in:** `6a75677`

**2. [Rule 2 - Missing Critical] Allowlisted host-provided proxy endpoints in whitelist mode**
- **Found during:** Task 2
- **Issue:** Extracting proxy hostnames only from raw config fields would miss proxies inherited from host environment variables, leaving real proxy traffic blocked in whitelist mode.
- **Fix:** Resolved effective proxy values with `resolveProxy` before calling `extractProxyHostnames`, then appended those hosts to the runtime allow list.
- **Files modified:** `internal/docker/launch.go`
- **Verification:** `go test ./internal/docker/... -v -count=1`
- **Committed in:** `82fa3c9`

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 missing critical)
**Impact on plan:** Both fixes were required to make the planned firewall lifecycle compile and enforce proxy-aware whitelist behavior correctly.

## Issues Encountered
- `go test ./tests/... -v` and `go test ./... -count=1` still fail in unrelated dirty-worktree test `tests/cli_dx_test.go:TestUnknownKeysRemediationHintOnStderr` (expected exit code 2, got 1); left deferred per scope boundary.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 10 manager integration is complete; launch/stop/destroy now own the firewall lifecycle end to end.
- The remaining failing CLI DX test is unrelated to network sandboxing and should be resolved separately before treating the whole repository as clean.

## Self-Check: PASSED

---
*Phase: 10-network-sandboxing*
*Completed: 2026-04-03*
