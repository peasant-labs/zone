# Phase 7: Environment, Auth & Forwarding - Context

**Gathered:** 2026-03-30
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement environment variable forwarding (glob-based matching, pre-launch validation), SSH agent socket forwarding, auth config copy-on-start mounting, .env file support, proxy auto-detection and forwarding, port forwarding from config, and lifecycle hooks (pre_build, post_stop). All of these inject runtime configuration into the container without persisting secrets in the Docker image. This phase extends the Docker Manager from Phase 6 — the createContainer(), buildMounts(), buildImage(), and Launch() methods gain new capabilities. The --port/-P ad-hoc CLI flag is deferred to Phase 8 (CLI flags).

</domain>

<decisions>
## Implementation Decisions

### Env var forwarding (CFG-10)
- **D-01:** New `internal/docker/env.go` file handles all env collection, glob matching, and validation — keeps manager.go focused on lifecycle orchestration
- **D-02:** Glob matching uses `filepath.Match` semantics per spec §4.6 — iterate `os.Environ()`, match each key against patterns in `auth.forward_env`
- **D-03:** Matched env vars injected as `container.Config.Env` in createContainer() — Docker API `Env` field takes `[]string{"KEY=VALUE"}` format
- **D-04:** Non-required forward_env vars not set in host environment produce a warning to stderr (not an error) per spec §4.5 — "may be set at launch time"

### Pre-launch env validation (CFG-11)
- **D-05:** Validation runs after lock acquisition, before `buildIfNeeded()` in Launch() — fail fast before any Docker build operations
- **D-06:** Check harness `RequiredEnvVars()` against host environment — missing required vars produce immediate error per spec §4.6: `"Error: Required environment variable X is not set. The Y harness needs this variable. Set it and re-run zone launch."`
- **D-07:** Validation also checks env vars from `harness.required_env` (custom harness field) — same error format
- **D-08:** If `auth.env_file` is set, vars in the .env file satisfy required env checks (parse .env file before validation)

### SSH agent forwarding (CFG-12)
- **D-09:** When `forward_ssh_agent = true`: mount host `SSH_AUTH_SOCK` as bind mount into container, set `SSH_AUTH_SOCK` env var to the mount target path
- **D-10:** On macOS: warn to stderr "SSH agent forwarding is not available on macOS (domain sockets cannot be bind-mounted). SSH operations inside the container will not have agent access." — proceed without mounting (do not abort launch)
- **D-11:** On Linux when `SSH_AUTH_SOCK` is unset or socket file doesn't exist: warn to stderr "SSH_AUTH_SOCK is not set or socket not found. SSH agent forwarding skipped." — proceed without mounting
- **D-12:** Mount is read-only (socket file only needs read access for the agent protocol)

### Auth config mount (CFG-13)
- **D-13:** When `mount_home_config = true` (default): mount each harness ConfigDirs() dir from host to `<dir>.host` inside container (read-only bind mount)
- **D-14:** Entrypoint already has ConfigCopyCommands from Phase 5 harness bridge — those commands copy `<dir>.host` → `<dir>` at container start, giving the harness a writable copy
- **D-15:** If host config dir doesn't exist: skip the mount with a debug log (not an error, per spec §4.10) — the corresponding copy command has `|| true` to handle missing source
- **D-16:** Auth mounts added in buildMounts() alongside workspace and home volume mounts

### .env file support (CFG-14)
- **D-17:** When `auth.env_file` is set and non-empty: resolve path relative to repo root, validate file exists, pass as Docker `--env-file` equivalent via HostConfig
- **D-18:** If env_file path doesn't exist: error (not warning) — user explicitly configured it, missing file is likely a mistake

### Proxy support (CFG-15)
- **D-19:** Config values (`network.http_proxy`, `network.https_proxy`, `network.no_proxy`) take precedence; if unset, auto-detect from host env vars `HTTP_PROXY`/`http_proxy`, `HTTPS_PROXY`/`https_proxy`, `NO_PROXY`/`no_proxy`
- **D-20:** Resolved proxy values passed as: (1) `--build-arg` during Docker build for package installation, (2) container env vars for runtime use
- **D-21:** When whitelist network mode is active (Phase 10), proxy server hostname auto-added to allow list — this integration point is stubbed here, implemented in Phase 10

### Port forwarding (CFG-16)
- **D-22:** Parse `workspace.ports` entries as `"hostPort:containerPort"` format (both must be valid port numbers 1-65535)
- **D-23:** Map to Docker API `PortBindings` in HostConfig and `ExposedPorts` in container Config
- **D-24:** Conflicting port entries (same host port) produce an error at validation time per spec §4.5
- **D-25:** Ad-hoc `--port/-P` CLI flag deferred to Phase 8 (CLI commands phase)

### Resource limits (CFG-17)
- **D-26:** Memory and CPU limits already implemented in Phase 6 createContainer() — no additional work needed
- **D-27:** PidsLimit already implemented in Phase 6 — no additional work needed

### Hooks (CFG-18)
- **D-28:** `pre_build` commands execute on the HOST (not in container) before Docker build starts — run via `os/exec` with inherited environment
- **D-29:** pre_build failure aborts launch with error — pre_build may prepare required artifacts (e.g., downloading models, generating configs)
- **D-30:** `post_stop` commands execute on the HOST after container stops — run via `os/exec` with inherited environment
- **D-31:** post_stop failure warns to stderr but does not fail the stop operation — container is already stopped, cleanup hooks are best-effort
- **D-32:** Hook commands run sequentially in order defined in zone.toml; each command inherits the repo directory as working dir

### Claude's Discretion
- Internal helper function organization within env.go
- Exact SSH_AUTH_SOCK mount path inside container (e.g., `/run/ssh-agent.sock` or similar)
- .env file parsing approach (Docker-compatible format)
- Port parsing error message wording
- Hook execution timeout (if any)
- Test strategy (unit tests with mocked Docker client vs integration tests)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Env forwarding and validation
- `zone-spec.md` §4.6 (lines 441-457) — forward_env glob matching, pre-launch validation, .env file support
- `zone-spec.md` §4.5 (lines 434-439) — Additional validations including forward_env warnings

### SSH agent forwarding
- `zone-spec.md` §4.7 (lines 459-468) — SSH agent socket mount, no key copying, ~/.ssh blocked

### Auth config mount
- `zone-spec.md` §4.10 (lines 520-528) — Copy-on-start strategy: mount to .host suffix, entrypoint copies

### Proxy support
- `zone-spec.md` §4.11 (lines 530-539) — Proxy env vars, build-arg + runtime injection, auto-detection

### Config fields
- `zone-spec.md` §4.1-4.2 (lines 254-373) — All config fields affecting this phase: auth, workspace.ports, resources, network proxy, hooks
- `internal/config/types.go` — AuthConfig, WorkspaceConfig, HooksConfig, NetworkConfig, ResourcesConfig structs

### Hooks
- `zone-spec.md` §4.2 (lines 364-366) — pre_build and post_stop hook definitions

### Existing code (Phase 6 extension points)
- `internal/docker/manager.go` — createContainer(), buildMounts(), Build() methods to extend
- `internal/docker/launch.go` — Launch() method where pre-launch validation and hooks integrate
- `internal/docker/build.go` — buildImage() where proxy build-args and pre_build hooks integrate
- `internal/docker/harness_bridge.go` — BuildDockerfileData/EntrypointData already populate ConfigCopyCommands
- `internal/docker/resources.go` — parseMemoryBytes/parseNanoCPUs (already complete)

### Harness integration
- `internal/harness/harness.go` — Harness interface with RequiredEnvVars(), ConfigDirs() methods
- `internal/harness/claude_code.go` — ClaudeCode.RequiredEnvVars() returns ["ANTHROPIC_API_KEY"]

### Requirements
- `.planning/REQUIREMENTS.md` — CFG-10 through CFG-18

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/docker/manager.go:createContainer()` — Currently handles security flags, resource limits, workspace + home mounts; needs env vars, port bindings, SSH mount, auth config mounts added
- `internal/docker/manager.go:buildMounts()` — Returns mount list; extend with SSH socket, auth config dir mounts, extra mounts
- `internal/docker/launch.go:Launch()` — Insert pre-launch validation after lock, pre_build hooks before buildIfNeeded
- `internal/docker/build.go:buildImage()` — Insert proxy build-args into Docker build options
- `internal/docker/harness_bridge.go:configCopyCmd()` — Already generates copy commands for harness config dirs (Phase 5)
- `internal/docker/resources.go` — Memory/CPU parsing complete, reusable pattern for port parsing
- `internal/config/types.go` — AuthConfig (ForwardEnv, ForwardSSHAgent, EnvFile), HooksConfig (PreBuild, PostStop), WorkspaceConfig (Ports), NetworkConfig (HTTPProxy, HTTPSProxy, NoProxy) all defined
- `internal/config/merge.go` — Merge logic already handles all Phase 7 config fields (forward_env union, forward_ssh_agent bool merge, hooks append, ports replace)

### Established Patterns
- Error wrapping: `fmt.Errorf("context: %w", err)` throughout
- Config access: `m.config.Auth.ForwardEnv`, `m.config.Hooks.PreBuild`, etc.
- Bool pointer: `*m.config.Auth.ForwardSSHAgent` with nil-is-default pattern
- Build-arg pattern in Docker SDK: `types.ImageBuildOptions{BuildArgs: map[string]*string{...}}`
- Env var pattern in Docker SDK: `container.Config{Env: []string{"KEY=VALUE"}}`

### Integration Points
- `createContainer()` → add Env, PortBindings, ExposedPorts, additional mounts
- `buildMounts()` → add SSH socket mount, auth config dir mounts
- `Launch()` → add pre-launch validation step, pre_build hook execution
- `buildImage()` → add proxy build-args
- `Stop()` → add post_stop hook execution after container removal
- New `env.go` → called from createContainer() to collect forwarded env vars

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. All decisions auto-selected from recommended defaults following the spec's prescriptive patterns. The spec provides exact behavior for env forwarding, SSH agent mounting, auth config copy-on-start, proxy injection, port mapping, and hook execution timing.

</specifics>

<deferred>
## Deferred Ideas

- Ad-hoc `--port/-P` CLI flag — Phase 8 (CLI commands phase)
- Proxy hostname auto-add to whitelist — Phase 10 (network sandboxing)
- Resource limit validation (invalid memory/CPU strings) — already handled by parseMemoryBytes/parseNanoCPUs in Phase 6

None beyond the above — discussion stayed within phase scope.

</deferred>

---

*Phase: 07-environment-auth-forwarding*
*Context gathered: 2026-03-30*
