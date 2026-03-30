---
phase: 07-environment-auth-forwarding
plan: 02
subsystem: docker
tags: [go, docker, nat, ports, proxy, hooks, exec]

# Dependency graph
requires:
  - phase: 06-docker-lifecycle-core
    provides: internal/docker package structure, manager.go patterns, error wrapping conventions
  - phase: 02-config-foundation
    provides: config.NetworkConfig, config.WorkspaceConfig, config.HooksConfig types
provides:
  - "parsePortBindings: hostPort:containerPort strings to nat.PortMap+nat.PortSet"
  - "validatePort: port range and numeric validation"
  - "resolveProxy: config-first then host env precedence"
  - "proxyBuildArgs: map[string]*string with unique pointers for Docker ImageBuild"
  - "proxyEnvVars: KEY=value slice for container env injection"
  - "firstEnv: first-non-empty env var lookup across key variants"
  - "runHooks: sh -c execution with fail-fast (pre_build) and warn-only (post_stop) modes"
affects:
  - 07-03 (wiring plan: consumes all three helpers)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pointer aliasing prevention: each *string value gets its own local variable before taking address"
    - "failFast vs warn-only hook pattern: same signature, bool parameter controls error vs warning"
    - "Config-over-env precedence: check config field first, fall back to firstEnv() helper"

key-files:
  created:
    - internal/docker/ports.go
    - internal/docker/ports_test.go
    - internal/docker/proxy.go
    - internal/docker/proxy_test.go
    - internal/docker/hooks.go
    - internal/docker/hooks_test.go
  modified: []

key-decisions:
  - "proxyBuildArgs uses separate local variables for each *string to prevent pointer aliasing bug where all pointers point to the same underlying value"
  - "runHooks takes io.Writer for stderr to enable test capture without os.Pipe() overhead"
  - "validatePort and parsePortBindings are unexported — consumed only within docker package by Plan 03 wiring"
  - "resolveProxy, proxyBuildArgs, proxyEnvVars, runHooks all unexported — same pattern as parseMemoryBytes in resources.go"

patterns-established:
  - "Proxy both-case pattern: HTTP_PROXY and http_proxy always emitted together — consistent with Docker conventions"
  - "Hook warn-only: fmt.Fprintf to stderr rather than log.Printf — avoids log prefix in terminal output"

requirements-completed:
  - CFG-15
  - CFG-16
  - CFG-17
  - CFG-18

# Metrics
duration: 2min
completed: 2026-03-30
---

# Phase 07 Plan 02: Port Parsing, Proxy Resolution, and Hook Execution Summary

**Three pure-logic helpers for Docker port binding (nat.PortMap/PortSet), proxy config-vs-env resolution with build-arg formatting, and lifecycle hook execution with fail-fast/warn-only modes**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-30T00:27:55Z
- **Completed:** 2026-03-30T00:29:55Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Port binding parser produces nat.PortMap and nat.PortSet from "hostPort:containerPort" strings with full validation (range, numeric, conflict detection)
- Proxy resolver centralizes config-takes-precedence semantics; build-args and env-var formatters emit both uppercase and lowercase variants
- Hook executor runs sh -c commands in repoDir with fail-fast (pre_build) or warn-only (post_stop) behavior
- All three helpers are independently testable with no Docker daemon required

## Task Commits

Each task was committed atomically:

1. **Task 1: ports.go with parsePortBindings and validatePort** - `8d81be8` (feat)
2. **Task 2: proxy.go and hooks.go with proxy resolution and hook execution** - `612bdc6` (feat)

## Files Created/Modified
- `internal/docker/ports.go` - parsePortBindings (nat.PortMap + nat.PortSet), validatePort (range check)
- `internal/docker/ports_test.go` - 12 test functions covering all port edge cases
- `internal/docker/proxy.go` - resolveProxy, firstEnv, proxyBuildArgs, proxyEnvVars
- `internal/docker/proxy_test.go` - 9 test functions: config precedence, env fallback, pointer aliasing, formatting
- `internal/docker/hooks.go` - runHooks with fail-fast and warn-only modes
- `internal/docker/hooks_test.go` - 7 test functions: fail-fast stop, warn-only continue, working dir, env inheritance

## Decisions Made
- proxyBuildArgs uses separate local variables (`v1 := val; v2 := val`) for each *string to prevent the pointer aliasing bug where all map values point to the same address
- runHooks takes `io.Writer` for stderr parameter to enable clean test capture with `bytes.Buffer`
- All helpers are unexported — consistent with parseMemoryBytes/parseNanoCPUs pattern in resources.go; consumed only within the docker package by Plan 03 wiring

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- parsePortBindings, resolveProxy, proxyBuildArgs, proxyEnvVars, and runHooks are all ready for Plan 03 wiring into createContainer and buildImage
- nat.PortMap and nat.PortSet types ready to be passed to container.HostConfig.PortBindings and container.Config.ExposedPorts

---
*Phase: 07-environment-auth-forwarding*
*Completed: 2026-03-30*
