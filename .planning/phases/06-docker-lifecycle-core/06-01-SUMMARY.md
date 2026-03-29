---
phase: 06-docker-lifecycle-core
plan: 01
subsystem: docker
tags: [docker-sdk, docker, container, manager, build, network, mock, unit-tests]

# Dependency graph
requires:
  - phase: 05-harness-plugin-system
    provides: Harness interface, Get() factory, harness_bridge.go bridge functions
  - phase: 04-template-system
    provides: RenderDockerfile/RenderEntrypoint/RenderShellRC template renderers
  - phase: 03-cache-state
    provides: cache.Cache, cache.ComputeHash, CreateBuildLog
  - phase: 02-config-foundation
    provides: MergedConfig, ResourcesConfig, WorkspaceConfig (PersistHome *bool)
provides:
  - Docker Manager foundation with mock-able DockerClient interface
  - NewManager constructor (fail-fast Ping verification)
  - Build() / buildImage() pipeline (tar context + JSON stream + cache)
  - createContainer() with security flags, labels, mounts, resource limits
  - createNetwork() / removeNetwork() bridge network helpers
  - parseMemoryBytes() / parseNanoCPUs() resource string parsers
  - 13 unit tests with mock client (no live Docker required)
affects: [06-02-launch-state-machine, 06-03-stop-destroy, 06-04-cobra-wiring]

# Tech tracking
tech-stack:
  added:
    - github.com/docker/docker v28.5.2+incompatible
    - github.com/docker/go-units v0.5.0
    - github.com/docker/go-connections v0.6.0
    - github.com/opencontainers/image-spec v1.1.1
    - github.com/containerd/errdefs v1.0.0
    - github.com/distribution/reference v0.6.0
    - go.opentelemetry.io/otel (transitive)
  patterns:
    - DockerClient interface wrapping SDK client for mock-based unit testing
    - newManagerWithClient() test constructor accepting injected mock
    - buildContext() tar archive with explicit file modes (entrypoint 0755)
    - streamBuildOutput() JSON line scanner with aux imageID capture
    - homeVolumeName() deterministic zone-home-<16hex> from repo path hash
    - PersistHome *bool nil-means-true pattern for default home volume
    - errdefs.IsNotFound() for swallowing expected network/container not-found errors

key-files:
  created:
    - internal/docker/client_interface.go
    - internal/docker/resources.go
    - internal/docker/build.go
    - internal/docker/manager_test.go
  modified:
    - internal/docker/errors.go (added ErrDockerNotRunning, ErrNoContainer)
    - internal/docker/manager.go (full Manager implementation)
    - internal/docker/network.go (createNetwork, removeNetwork)
    - go.mod (docker SDK + transitive deps)
    - go.sum

key-decisions:
  - "DockerClient interface enables mock-based unit testing without live Docker daemon"
  - "go mod tidy removes unused deps — create source files before tidying when adding new imports"
  - "errdefs package requires containerd/errdefs transitive dep (not auto-pulled by go mod tidy alone)"
  - "testify upgraded to v1.11.1 by go mod tidy (transitive dep resolution)"

patterns-established:
  - "Mock DockerClient: implement full interface in mockClient struct, configure per-test return values"
  - "Resource limits: parseMemoryBytes/parseNanoCPUs return 0 for empty/zero (Docker API no-limit semantics)"
  - "Home volume: PersistHome == nil treated as true (spec default), *false explicitly disables"

requirements-completed: [DOC-11, DOC-12, DOC-08, CFG-20]

# Metrics
duration: 5min
completed: 2026-03-29
---

# Phase 6 Plan 01: Docker Manager Foundation Summary

**Docker SDK v28.5.2 integrated with mock-able DockerClient interface, Manager constructor (Ping fail-fast), build pipeline (tar + JSON streaming), labeled bridge network helpers, container creation (security flags + resource limits + home volume), and 13 unit tests using mock client**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-29T22:39:58Z
- **Completed:** 2026-03-29T22:45:14Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- Docker SDK v28.5.2 installed with all transitive dependencies resolved
- DockerClient interface makes Manager fully testable without a live Docker daemon
- Build pipeline renders templates → tar context → ImageBuild → JSON stream → cache
- createContainer() applies full security hardening: no-new-privileges, CapDrop ALL, CapAdd [CHOWN DAC_OVERRIDE SETGID SETUID FOWNER], PidsLimit, Memory, NanoCPUs, IPv6 disabled via sysctl
- 13 unit tests pass covering all resource parsers, mount logic, build streaming, and network helpers

## Task Commits

Each task was committed atomically:

1. **Task 1: Docker SDK, client interface, sentinel errors, resource parsers** - `0bbb0cd` (feat)
2. **Task 2: Manager struct, build pipeline, network helpers, container creation, tests** - `34d2f9d` (feat)

**Plan metadata:** (docs commit — see below)

## Files Created/Modified
- `internal/docker/client_interface.go` - Mock-able DockerClient interface (15 methods)
- `internal/docker/errors.go` - Added ErrDockerNotRunning and ErrNoContainer sentinel errors
- `internal/docker/resources.go` - parseMemoryBytes and parseNanoCPUs resource parsers
- `internal/docker/manager.go` - Manager struct, NewManager, Build, createContainer, buildMounts, attachInteractive
- `internal/docker/build.go` - buildContext (tar), streamBuildOutput (JSON), buildImage orchestrator
- `internal/docker/network.go` - createNetwork, removeNetwork (errdefs.IsNotFound swallowed)
- `internal/docker/manager_test.go` - 13 unit tests with mockClient (no live Docker)
- `go.mod` - docker SDK + all transitive deps
- `go.sum` - updated checksums

## Decisions Made
- DockerClient interface wraps SDK for testability — newManagerWithClient() accepts mock in tests
- go mod tidy removes any dep without a Go source importer — write source files before tidying
- errdefs package (for IsNotFound) requires explicit `go get github.com/containerd/errdefs` (not auto-pulled)
- testify v1.11.1 pulled by go mod tidy resolution of transitive deps (from v1.10.0)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Transitive Docker SDK dependencies required explicit `go get`**
- **Found during:** Task 1 (Docker SDK installation)
- **Issue:** `go mod tidy` removes deps without importers; multiple transitive packages (containerd/errdefs, distribution/reference, otelhttp) needed explicit `go get` after creating source files
- **Fix:** Added `go get` calls for each missing transitive dep before running `go mod tidy`
- **Files modified:** go.mod, go.sum
- **Verification:** `go build ./...` exits 0
- **Committed in:** 0bbb0cd, 34d2f9d (staged across tasks)

---

**Total deviations:** 1 auto-fixed (Rule 3 — blocking)
**Impact on plan:** Standard Go module management behavior. No scope creep.

## Issues Encountered
- Docker SDK transitive deps are numerous (errdefs, distribution, otelhttp, moby packages) — required iterative `go get` calls since `go mod tidy` alone doesn't pull them without importers in source

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Manager foundation complete with all methods Plans 02-04 depend on
- NewManager, Build(), createContainer(), createNetwork(), removeNetwork() all implemented
- Mock-based test infrastructure established — Plans 02-04 tests can reuse mockClient
- Plans 02-04 can import internal/docker and call Manager methods directly

---
*Phase: 06-docker-lifecycle-core*
*Completed: 2026-03-29*

## Self-Check: PASSED

- internal/docker/client_interface.go: FOUND
- internal/docker/errors.go: FOUND
- internal/docker/resources.go: FOUND
- internal/docker/manager.go: FOUND
- internal/docker/build.go: FOUND
- internal/docker/network.go: FOUND
- internal/docker/manager_test.go: FOUND
- .planning/phases/06-docker-lifecycle-core/06-01-SUMMARY.md: FOUND
- Commit 0bbb0cd: FOUND
- Commit 34d2f9d: FOUND
