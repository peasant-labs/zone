# Phase 6: Docker Lifecycle Core - Context

**Gathered:** 2026-03-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement the Docker container lifecycle: build images from rendered templates, create/start containers with security hardening, attach TTY for interactive sessions, reattach to running containers, stop/restart/destroy with proper cleanup. Idempotent launch: detect container state and handle running/paused/exited/dead/stale scenarios. Zero-config quickstart via `--harness` flag. Headless mode for fire-and-forget agents. This phase wires up the Cobra command stubs (launch, join, exec, shell, build, stop, restart, clean, destroy) to the Docker Manager. TUI is NOT implemented here (Phase 9) — all output is plain text.

</domain>

<decisions>
## Implementation Decisions

### Docker SDK integration (DOC-11)
- Use `github.com/docker/docker/client` Go SDK for all non-interactive operations: build, create, start, stop, inspect, remove, network create/destroy
- Use `os/exec` with `docker exec -it` for interactive TTY attach (per spec §12 — SDK's hijacked connection API is unreliable for raw terminal I/O)
- Client initialized once in Manager constructor with `client.FromEnv` + `WithAPIVersionNegotiation()`
- Verify connectivity with `Ping()` on construction — fail fast with `ErrDockerNotRunning` and actionable error message
- Context propagation on all SDK calls for graceful cancellation

### Container state machine (DOC-09)
- Running: check config hash, warn if changed ("Config has changed since this container was started. Run `zone restart --rebuild` to apply changes."), then reattach
- Paused: unpause, then attach
- Exited/Dead: inspect exit code and OOMKilled flag, warn if OOM ("Container was killed due to memory limit. Increase `resources.memory` in zone.toml."), remove container + network, proceed to build/launch
- Created/Restarting: wait briefly (2s), then stop, remove, proceed to build/launch
- Stale container ID (container doesn't exist): clean up stale cache files (container_id, network_id), attempt to remove orphaned network, proceed to build/launch
- No container_id file: fresh launch path

### Build behavior (DOC-12)
- Build progress streamed to stderr line-by-line (plain text — TUI is Phase 9)
- Build log captured to `.zone/logs/last_build.log`
- Build errors show last 20 lines of build log + path to full log
- All builds use BuildKit (`DOCKER_BUILDKIT=1`, Dockerfile has `# syntax=docker/dockerfile:1`)
- Config hash comparison determines whether rebuild is needed (Phase 3 cache hash)
- Image pruned detection: verify `image_id` with `ImageInspect` before reusing

### Config change detection (DOC-10)
- Compare full cache hash (config + templates + version) against stored `.zone/config.hash`
- If running container has stale config: warn user with single-line message, do NOT auto-restart (user must run `zone restart --rebuild`)
- If no running container and hash mismatch: auto-rebuild silently (just regenerate + build)

### Container creation security (DOC-06, carried from Phase 4)
- SecurityOpt: `no-new-privileges`
- CapDrop: ALL, CapAdd: CHOWN, DAC_OVERRIDE, SETGID, SETUID, FOWNER
- PidsLimit from config (default 512)
- Memory and CPU limits from config (0 = no limit)
- IPv6 disabled via sysctl `net.ipv6.conf.all.disable_ipv6=1`
- Each container gets its own bridge network

### Docker labels (DOC-08)
- Apply `com.zone.managed=true`, `com.zone.repo-path`, `com.zone.harness` labels per spec
- Labels enable `zone ls` discovery (Phase 8 wires the list command)

### Home volume persistence (CFG-20)
- When `persist_home = true` (default): create named volume `zone-home-<shortHash>` for `/home/zone`
- Volume survives container recreation (stop + relaunch preserves npm cache, shell history, harness state)
- `zone destroy` removes the volume; `zone stop` and `zone clean` do NOT

### Interactive attach
- `zone launch`: build-if-needed, create, start, then attach TTY via `docker exec -it`
- `zone join`: attach new shell to running container (no harness restart)
- `zone shell`: interactive shell (`bash` or configured shell), no harness process
- `zone exec -- <cmd>`: one-off command execution inside container
- Lock released before TTY attach to allow `zone join` from another terminal

### Zero-config quickstart (CLI-05)
- `zone launch --harness claude-code` with no zone.toml: generate minimal zone.toml (`version = 1` + `harness = "claude-code"` + commented options), add `.zone/` to `.gitignore`, proceed to build/launch
- No `--harness` and no zone.toml in Phase 6: error with "No zone.toml found. Run `zone init --harness <name>` or `zone launch --harness <name>`" (TUI wizard deferred to Phase 9)
- Generated zone.toml includes commented-out sections showing available options for discoverability

### Headless mode (CLI-04)
- `zone launch --headless`: build, create, start, print container ID to stdout, return immediately
- `zone launch --headless -p "task"`: inject prompt via harness PromptFlag() into entrypoint args
- No TTY attach in headless mode
- Exit code 0 on successful start (not on agent completion)

### Stop/restart/destroy cleanup
- `zone stop`: stop container (SIGTERM → timeout → SIGKILL), remove container, remove network, clear container_id + network_id from cache, retain image_id + config.hash
- `zone restart`: stop + relaunch (rebuild if `--rebuild` flag)
- `zone destroy`: stop + remove image + remove home volume + remove all `.zone/` cache
- `zone build`: force-rebuild image without launching (useful for CI pre-warming)

### Cobra command wiring
- Wire `cmd/launch.go`, `cmd/join.go`, `cmd/exec.go`, `cmd/shell.go`, `cmd/build.go`, `cmd/stop.go`, `cmd/restart.go`, `cmd/destroy.go` — replace `"not implemented"` stubs with Manager calls
- `cmd/clean.go` already partially wired from Phase 3 — extend to handle image removal with `--image` flag
- Flags: `--harness`, `--headless`, `-p`/`--prompt`, `--rebuild`, `--timeout`, `--root`, `--yes`/`-y`

### Claude's Discretion
- Manager method internal structure (helper extraction, error wrapping patterns)
- Build context tar construction approach
- Exact timeout values for container state transitions
- Test strategy for Docker integration (mock client vs integration tests)
- Error message wording beyond spec examples

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Docker Manager architecture
- `zone-spec.md` §12 (lines 1150-1303) — Manager struct, constructor, method signatures, security flags, home volume, interactive TTY attach pattern, platform detection, BuildKit

### Container lifecycle behavior
- `zone-spec.md` §3.7 (lines 131-146) — Idempotent launch state machine (running/paused/exited/dead/stale), config hash check, lock release timing
- `zone-spec.md` §3.8 (lines 148-152) — Exit behavior: Ctrl+C sends SIGINT, harness exit stops container
- `zone-spec.md` §3.9 (lines 154-161) — Stop cleanup sequence
- `zone-spec.md` §3.6 (lines 121-129) — Zero-config quickstart paths (--harness, TTY, non-TTY)

### Command specifications
- `zone-spec.md` §3.1-3.2 (lines 52-85) — Full command table with flags, aliases, behavior
- `zone-spec.md` §3.4 (lines 99-108) — Argument forwarding convention (`--` separator)
- `zone-spec.md` §3.12 (lines 218-243) — Actionable error message format with remediation hints

### Security
- `zone-spec.md` §12 (lines 1219-1237) — Container security flags: no-new-privileges, CapDrop ALL, CapAdd list, IPv6 disable, resource limits

### Config types (consumed by Manager)
- `zone-spec.md` §4 (lines 254-458) — Config fields that affect container creation (resources, ports, mounts, persist_home)
- `internal/config/types.go` — MergedConfig, ResourcesConfig, WorkspaceConfig

### Existing code
- `internal/docker/naming.go` — ContainerName(), NetworkName(), ContainerLabels() (Phase 4)
- `internal/docker/errors.go` — ContainerSecurityFlags() (Phase 4)
- `internal/docker/harness_bridge.go` — BuildDockerfileData/EntrypointData/ShellRCData (Phase 5)
- `internal/docker/dockerfile.go` — RenderDockerfile() (Phase 4)
- `internal/docker/entrypoint.go` — RenderEntrypoint() (Phase 4)
- `internal/docker/shellrc.go` — RenderShellRC() (Phase 4)
- `internal/cache/cache.go` — Cache struct, directory management (Phase 3)
- `internal/cache/hash.go` — ComputeHash() (Phase 3)
- `internal/cache/lock.go` — flock-based locking (Phase 3)

### Requirements
- `.planning/REQUIREMENTS.md` — DOC-08 through DOC-12, CFG-20, CLI-03 through CLI-11, CLI-15, CLI-16

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/docker/naming.go` — ContainerName(), NetworkName(), ContainerLabels() ready to use
- `internal/docker/errors.go` — ContainerSecurityFlags() returns hardened security config struct
- `internal/docker/harness_bridge.go` — BuildDockerfileData/EntrypointData/ShellRCData bridge harness→template
- `internal/docker/dockerfile.go`, `entrypoint.go`, `shellrc.go` — RenderDockerfile/Entrypoint/ShellRC functions
- `internal/docker/platform.go` — HostUID(), MacOSUsername(), DetectGitIdentity()
- `internal/cache/` — Cache struct with ReadID/WriteID/ComputeHash/Lock.Acquire/Release
- `internal/harness/` — Full harness system with Get(), Validate(), all interface methods
- `cmd/*.go` — All 15 Cobra command stubs exist with `"not implemented"` RunE

### Established Patterns
- Error wrapping: `fmt.Errorf("context: %w", err)` throughout
- Stub-first: manager.go exists as package-only stub, ready to implement
- network.go exists as package-only stub for Docker network operations
- Config merge: MergedConfig has all fields needed for container creation (resources, workspace, auth)
- Cache hash: ComputeHash(version) already includes config + templates + version

### Integration Points
- `cmd/launch.go` → calls Manager.Launch() which orchestrates the full state machine
- `cmd/join.go`, `cmd/exec.go`, `cmd/shell.go` → call Manager.Join/Exec/Shell for interactive attach
- `cmd/stop.go`, `cmd/restart.go`, `cmd/destroy.go` → call Manager.Stop with varying cleanup levels
- `cmd/build.go` → calls Manager.Build for standalone image build
- Manager constructor → takes MergedConfig + Cache, creates Docker client
- Manager.Launch → uses harness.Get() + bridge functions + render functions + cache + Docker SDK

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. All decisions auto-selected from recommended defaults following the spec's prescriptive patterns. The spec provides exact Manager struct, method signatures, security flags, state machine behavior, and TTY attach pattern.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 06-docker-lifecycle-core*
*Context gathered: 2026-03-29*
