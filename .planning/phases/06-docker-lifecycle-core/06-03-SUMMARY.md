---
phase: 06-docker-lifecycle-core
plan: 03
subsystem: docker
tags: [docker, container, lifecycle, stop, destroy, cleanup, tdd, unit-tests]

# Dependency graph
requires:
  - phase: 06-docker-lifecycle-core
    plan: 01
    provides: Manager struct, DockerClient interface, mock client infrastructure
  - phase: 06-docker-lifecycle-core
    plan: 02
    provides: removeNetwork, cache integration patterns
  - phase: 03-cache-state
    provides: Cache.ContainerID/SetContainerID/ImageID/SetImageID/Clean
provides:
  - Manager.Stop() with 10s timeout, ContainerStop + ContainerRemove + removeNetwork + cache clear
  - Manager.Destroy() with full teardown (Stop + ImageRemove + VolumeRemove + cache.Clean)
  - Manager.RemoveImage() standalone image removal for zone clean --image
  - errdefs.IsNotFound swallowing throughout all teardown paths
affects: [06-04-cobra-wiring]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "errdefs.IsNotFound() swallows expected not-found errors on Stop/Destroy — missing resource = desired end state"
    - "Stop clears container_id + network_id; retains image_id and config.hash for reuse"
    - "Destroy calls Stop first, then removes image/volume/cache — linear cleanup chain"
    - "homeVolumeName(m.repoDir) produces deterministic zone-home-<hash> for VolumeRemove"
    - "TDD RED/GREEN: failing tests committed before implementation"

key-files:
  created: []
  modified:
    - internal/docker/manager.go
    - internal/docker/manager_test.go

key-decisions:
  - "Stop retains image_id in cache — allows zone launch after stop to skip rebuild"
  - "Destroy calls Stop as first step — avoids code duplication for container/network cleanup"
  - "VolumeRemove NotFound swallowed in Destroy — volume may never have been created (persist_home=false)"
  - "RemoveImage is standalone (not part of Stop) — maps to zone clean --image, orthogonal to stop lifecycle"

requirements-completed: [CLI-10, CLI-15, CLI-16]

# Metrics
duration: 2min
completed: 2026-03-29
---

# Phase 6 Plan 03: Stop/Destroy/RemoveImage Lifecycle Methods Summary

**Stop/Destroy/RemoveImage teardown methods on Manager with full not-found error swallowing, home volume cleanup, and cache clearing — 8 new TDD unit tests all passing (31 total in docker package)**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-29T22:53:16Z
- **Completed:** 2026-03-29T22:55:11Z
- **Tasks:** 1 (TDD: RED commit + GREEN commit)
- **Files modified:** 2

## Accomplishments
- `Manager.Stop()`: graceful ContainerStop (10s), force ContainerRemove, removeNetwork, clears container_id + network_id — retains image_id for subsequent launch reuse
- `Manager.Destroy()`: calls Stop, then ImageRemove (PruneChildren=true), VolumeRemove(homeVolumeName), cache.Clean() — full teardown to pristine state
- `Manager.RemoveImage()`: standalone removal of cached image for `zone clean --image` use case
- All errdefs.IsNotFound errors swallowed consistently throughout — missing resource = desired end state
- Stop on empty container_id is a safe no-op (called by Destroy with no container)
- Extend mockClient with `imageRemovedIDs` and `volumeRemovedIDs` tracking for precise assertions
- 8 new unit tests: RunningContainer, NoContainer, ContainerNotFound, Destroy_Full, Destroy_NoContainer, DestroyVsStop_VolumeRetention (subtests), RemoveImage, RemoveImage_NoImage
- 31 total tests in docker package, all passing

## Task Commits

Each step committed atomically:

1. **TDD RED: failing tests for Stop/Destroy/RemoveImage** - `c651c62` (test)
2. **TDD GREEN: Stop, Destroy, RemoveImage implementation** - `f5eb6bb` (feat)

## Files Created/Modified
- `internal/docker/manager.go` - Added Stop(), Destroy(), RemoveImage() methods + image/errdefs imports
- `internal/docker/manager_test.go` - 8 new tests + imageRemovedIDs/volumeRemovedIDs tracking in mockClient

## Decisions Made
- Stop retains `image_id` in cache so `zone launch` after `zone stop` skips rebuilding the image
- Destroy calls Stop first to reuse container+network cleanup logic (DRY)
- VolumeRemove NotFound is swallowed — volume may not exist when `persist_home=false` or was never set up
- RemoveImage is orthogonal to Stop — it maps specifically to `zone clean --image`, not the stop lifecycle

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None.

## Next Phase Readiness
- `Manager.Stop()` and `Manager.Destroy()` are the primary entry points for Plan 04 (Cobra wiring of `cmd/stop.go` and `cmd/destroy.go`)
- `Manager.RemoveImage()` is the entry point for `zone clean --image` in Plan 04
- All three methods follow the same pattern as Manager.Launch() for consistency

---
*Phase: 06-docker-lifecycle-core*
*Completed: 2026-03-29*

## Self-Check: PASSED

- internal/docker/manager.go: FOUND
- internal/docker/manager_test.go: FOUND
- .planning/phases/06-docker-lifecycle-core/06-03-SUMMARY.md: FOUND
- Commit c651c62: FOUND
- Commit f5eb6bb: FOUND
