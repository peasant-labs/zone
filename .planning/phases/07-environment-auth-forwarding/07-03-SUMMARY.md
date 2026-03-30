---
phase: 07-environment-auth-forwarding
plan: "03"
subsystem: docker
tags: [integration, env, ssh, auth, ports, proxy, hooks, wiring]
dependency_graph:
  requires: ["07-01", "07-02"]
  provides: ["all-phase-7-features-wired"]
  affects: ["internal/docker/manager.go", "internal/docker/launch.go", "internal/docker/build.go"]
tech_stack:
  added: []
  patterns:
    - "validateEnvBeforeLock: pre-launch env validation after lock acquire, before inspect"
    - "hookReleaseOnError: lock.Release() before returning any hook or validation error"
    - "postStopWarnOnly: _ = runHooks(...) discards error for best-effort hooks"
    - "proxyBuildArgs: proxy vars passed as BuildArgs map[string]*string to ImageBuild"
key_files:
  created: []
  modified:
    - internal/docker/manager.go
    - internal/docker/launch.go
    - internal/docker/build.go
    - internal/docker/manager_test.go
decisions:
  - "MountHomeConfig default=true (nil pointer means enabled): auth config mounts active unless explicitly disabled"
  - "SSH agent: macOS prints warning and skips mount; Linux skips if SSH_AUTH_SOCK unset/not-socket"
  - "Existing Launch state machine tests updated to set ANTHROPIC_API_KEY via t.Setenv to satisfy validation"
  - "Existing buildMounts tests updated to disable MountHomeConfig for predictable mount counts"
metrics:
  duration: "5 min"
  completed: "2026-03-30"
  tasks_completed: 2
  files_modified: 4
---

# Phase 7 Plan 03: Wire Phase 7 Helpers into Manager Summary

One-liner: All Phase 7 helpers (env.go, ports.go, proxy.go, hooks.go) wired into Manager.createContainer(), buildMounts(), Launch(), Stop(), and buildImage() with 14 new integration tests.

## What Was Built

### Task 1: Extend buildMounts and createContainer

**buildMounts() additions:**

- SSH agent socket bind-mount (CFG-12): When `forward_ssh_agent=true`, on Linux with a valid `SSH_AUTH_SOCK` socket file, mounts the host socket as `/tmp/ssh-agent.sock` (read-only). On macOS, prints a warning and skips.
- Auth config dir mounts (CFG-13): When `mount_home_config=true` (default), collects harness `HomeConfigDir()` + `ExtraConfigDirs()`, expands `~/` prefix via `expandHome()`, and bind-mounts each existing dir at `<dir>.host` (read-only).
- Two new helpers: `collectConfigDirs(h harness.Harness) []string` and `expandHome(path string) string`.

**createContainer() additions:**

- CFG-10: `CollectForwardedEnv(m.config.Auth.ForwardEnv)` populates `container.Config.Env`
- CFG-14: `ParseEnvFile(envFilePath)` loads `.env` file vars into `container.Config.Env`
- CFG-15: `proxyEnvVars(...)` appends proxy env vars to `container.Config.Env`
- CFG-12: `SSH_AUTH_SOCK=/tmp/ssh-agent.sock` appended to env when socket was mounted
- CFG-16: `parsePortBindings(m.config.Workspace.Ports)` populates `HostConfig.PortBindings` and `Config.ExposedPorts`

**mockClient extensions:**
- `lastContainerConfig *container.Config` — captured in ContainerCreate
- `lastHostConfig *container.HostConfig` — captured in ContainerCreate
- `lastBuildOptions types.ImageBuildOptions` — captured in ImageBuild

### Task 2: Extend Launch, Stop, and buildImage

**Launch() additions (launch.go):**

- CFG-11: Pre-launch `ValidateRequiredEnv()` after lock acquire (Step 1.5), before container inspect. Calls `harness.Get()`, collects `RequiredEnvVars()` + `m.config.Harness.RequiredEnv`, calls `ValidateRequiredEnv()`. On failure: `lock.Release()` then return error.
- CFG-18: Pre-build hooks (Step 2.5), after container state branching, before `buildIfNeeded`. `runHooks(m.config.Hooks.PreBuild, ..., failFast=true)`. On failure: `lock.Release()` then return `fmt.Errorf("pre_build: %w", err)`.

**Stop() additions (manager.go):**

- CFG-18: Post-stop hooks after cache clearing, before `return nil`. `_ = runHooks(m.config.Hooks.PostStop, ..., failFast=false)` — best-effort, failures swallowed.

**buildImage() additions (build.go):**

- CFG-15: `resolveProxy(&m.config.Network)` + `proxyBuildArgs(...)` → `BuildArgs: buildArgs` in `types.ImageBuildOptions`.

## Tests Added

| Test | Covers |
|------|--------|
| TestBuildMounts_SSHAgent | SSH socket bind-mount on Linux |
| TestBuildMounts_SSHAgent_NoSocket | No SSH mount when SSH_AUTH_SOCK unset |
| TestBuildMounts_AuthConfig | Auth config dir mounted at .host suffix |
| TestBuildMounts_AuthConfig_Disabled | No auth mounts when disabled |
| TestCreateContainer_EnvVars | ForwardEnv pattern in Config.Env |
| TestCreateContainer_Ports | Port bindings in HostConfig and ExposedPorts |
| TestCreateContainer_EnvFile | .env file vars in Config.Env |
| TestLaunch_RequiredEnvValidation | Error when ANTHROPIC_API_KEY missing |
| TestLaunch_RequiredEnvValidation_Satisfied | No validation error when key present |
| TestLaunch_PreBuildHook | Successful pre_build hook passes |
| TestLaunch_PreBuildHook_Failure | Failing pre_build aborts with "pre_build" in error |
| TestStop_PostStopHook | Successful post_stop hook, Stop returns nil |
| TestStop_PostStopHook_Failure | Failing post_stop hook, Stop still returns nil |
| TestBuildImage_ProxyBuildArgs | HTTP_PROXY in BuildArgs when configured |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Existing buildMounts tests asserted exact mount counts**
- **Found during:** Task 1
- **Issue:** `TestBuildMounts_PersistHomeDefault` and `TestBuildMounts_PersistHomeFalse` asserted `Len(t, mounts, 2)` and `Len(t, mounts, 1)` respectively. After adding auth config mounts, the test environment's real `~/.claude` directory caused extra mounts, breaking these assertions.
- **Fix:** Added `cfg.Auth.MountHomeConfig = &disabled` to both tests to disable auth config mounts, making counts deterministic.
- **Files modified:** internal/docker/manager_test.go
- **Commit:** 290f029

**2. [Rule 1 - Bug] Existing Launch state machine tests failed env validation**
- **Found during:** Task 2
- **Issue:** All Launch tests using `makeLaunchMock` (8 tests) failed with "required environment variable ANTHROPIC_API_KEY is not set" after pre-launch validation was added.
- **Fix:** Added `t.Setenv("ANTHROPIC_API_KEY", "test-key-for-launch-tests")` to `makeLaunchMock` helper and `TestLaunchStateMachine_StaleID`. Also added `MountHomeConfig = &disabled` to prevent real `~/.claude` from affecting tests.
- **Files modified:** internal/docker/manager_test.go
- **Commit:** 290f029

**3. [Rule 1 - Bug] TestLaunch_RequiredEnvValidation used t.Setenv("", "") which doesn't unset**
- **Found during:** Task 2 test run
- **Issue:** `t.Setenv("ANTHROPIC_API_KEY", "")` sets the key to empty string in `os.Environ()`, but `ValidateRequiredEnv` marks any key that appears in `os.Environ()` as available (regardless of value). Test passed validation and produced wrong result.
- **Fix:** Changed to `t.Setenv("ANTHROPIC_API_KEY", "")` followed by `os.Unsetenv("ANTHROPIC_API_KEY")` to truly remove the key from the environment.
- **Files modified:** internal/docker/manager_test.go
- **Commit:** 290f029

## Self-Check: PASSED

- FOUND: internal/docker/manager.go (SSH mounts, auth config mounts, env vars, port bindings)
- FOUND: internal/docker/launch.go (pre-launch validation, pre_build hooks)
- FOUND: internal/docker/build.go (proxy build-args)
- FOUND: internal/docker/manager_test.go (14 new tests, all passing)
- FOUND: .planning/phases/07-environment-auth-forwarding/07-03-SUMMARY.md
- COMMIT 2825dc5: feat(07-03): wire SSH agent mounts, auth config mounts, env vars, and port bindings
- COMMIT 290f029: feat(07-03): wire pre-launch validation, pre_build/post_stop hooks, and proxy build-args
- Full test suite: `go test ./... -count=1` exits 0 (58 docker tests + integration tests pass)
