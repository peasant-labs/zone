# Phase 7: Environment, Auth & Forwarding - Research

**Researched:** 2026-03-30
**Domain:** Docker container runtime configuration — env forwarding, SSH agent sockets, auth config mounts, proxy injection, port mapping, lifecycle hooks
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Env var forwarding (CFG-10)**
- D-01: New `internal/docker/env.go` file handles all env collection, glob matching, and validation
- D-02: Glob matching uses `filepath.Match` semantics — iterate `os.Environ()`, match each key against patterns in `auth.forward_env`
- D-03: Matched env vars injected as `container.Config.Env` in createContainer() — Docker API `Env` field takes `[]string{"KEY=VALUE"}` format
- D-04: Non-required forward_env vars not set in host produce a warning to stderr (not an error) per spec §4.5

**Pre-launch env validation (CFG-11)**
- D-05: Validation runs after lock acquisition, before `buildIfNeeded()` in Launch()
- D-06: Check harness `RequiredEnvVars()` against host environment — missing required vars produce immediate error per spec §4.6: `"Error: Required environment variable X is not set. The Y harness needs this variable. Set it and re-run zone launch."`
- D-07: Validation also checks env vars from `harness.required_env` (custom harness field) — same error format
- D-08: If `auth.env_file` is set, vars in the .env file satisfy required env checks (parse .env file before validation)

**SSH agent forwarding (CFG-12)**
- D-09: When `forward_ssh_agent = true`: mount host `SSH_AUTH_SOCK` as bind mount into container, set `SSH_AUTH_SOCK` env var to the mount target path
- D-10: On macOS: warn to stderr and proceed without mounting
- D-11: On Linux when `SSH_AUTH_SOCK` is unset or socket file doesn't exist: warn to stderr and proceed without mounting
- D-12: Mount is read-only

**Auth config mount (CFG-13)**
- D-13: When `mount_home_config = true` (default): mount each harness ConfigDirs() dir from host to `<dir>.host` inside container (read-only bind mount)
- D-14: Entrypoint already has ConfigCopyCommands from Phase 5 harness bridge — those commands copy `<dir>.host` → `<dir>` at container start
- D-15: If host config dir doesn't exist: skip the mount with a debug log (not an error)
- D-16: Auth mounts added in buildMounts() alongside workspace and home volume mounts

**.env file support (CFG-14)**
- D-17: When `auth.env_file` is set and non-empty: resolve path relative to repo root, validate file exists, pass as Docker `--env-file` equivalent via HostConfig
- D-18: If env_file path doesn't exist: error (not warning)

**Proxy support (CFG-15)**
- D-19: Config values take precedence; if unset, auto-detect from host env vars
- D-20: Resolved proxy values passed as (1) `--build-arg` during Docker build, (2) container env vars for runtime use
- D-21: When whitelist network mode is active (Phase 10), proxy server hostname auto-added to allow list — stubbed here, implemented in Phase 10

**Port forwarding (CFG-16)**
- D-22: Parse `workspace.ports` entries as `"hostPort:containerPort"` format
- D-23: Map to Docker API `PortBindings` in HostConfig and `ExposedPorts` in container Config
- D-24: Conflicting port entries (same host port) produce an error at validation time
- D-25: Ad-hoc `--port/-P` CLI flag deferred to Phase 8

**Resource limits (CFG-17)**
- D-26: Memory and CPU limits already implemented in Phase 6 — no additional work needed
- D-27: PidsLimit already implemented in Phase 6 — no additional work needed

**Hooks (CFG-18)**
- D-28: `pre_build` commands execute on the HOST before Docker build starts — run via `os/exec` with inherited environment
- D-29: pre_build failure aborts launch with error
- D-30: `post_stop` commands execute on the HOST after container stops — run via `os/exec` with inherited environment
- D-31: post_stop failure warns to stderr but does not fail the stop operation
- D-32: Hook commands run sequentially in order defined in zone.toml; each command inherits the repo directory as working dir

### Claude's Discretion
- Internal helper function organization within env.go
- Exact SSH_AUTH_SOCK mount path inside container (e.g., `/run/ssh-agent.sock` or similar)
- .env file parsing approach (Docker-compatible format)
- Port parsing error message wording
- Hook execution timeout (if any)
- Test strategy (unit tests with mocked Docker client vs integration tests)

### Deferred Ideas (OUT OF SCOPE)
- Ad-hoc `--port/-P` CLI flag — Phase 8 (CLI commands phase)
- Proxy hostname auto-add to whitelist — Phase 10 (network sandboxing)
- Resource limit validation (invalid memory/CPU strings) — already handled by parseMemoryBytes/parseNanoCPUs in Phase 6
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CFG-10 | Environment variable forwarding supports glob patterns (e.g., `AWS_*`) | `filepath.Match`, `os.Environ()`, Docker `container.Config.Env []string{"KEY=VALUE"}` |
| CFG-11 | Pre-launch validation checks required env vars are set before Docker build starts | Insert validation in `Launch()` after lock, before `buildIfNeeded()`; harness `RequiredEnvVars()` and custom `required_env` both checked |
| CFG-12 | SSH agent forwarding mounts socket when `forward_ssh_agent = true` | `mount.TypeBind` with `SSH_AUTH_SOCK` socket; `runtime.GOOS` for macOS detection; read-only socket mount |
| CFG-13 | Auth config uses copy-on-start strategy (writable copy in container, host preserved) | `buildMounts()` adds read-only bind mounts to `<dir>.host`; Phase 5 `configCopyCmd()` already handles the copy entrypoint commands |
| CFG-14 | `.env` file support via `auth.env_file` config key | Parse .env file (bufio scanner, skip comments/blank lines), load into map; used for required-var satisfaction and passed via `container.Config.Env` |
| CFG-15 | Proxy support (http_proxy, https_proxy, no_proxy) with host auto-detection | Docker `ImageBuildOptions.BuildArgs map[string]*string`; container env vars; `os.Getenv` for host auto-detection |
| CFG-16 | Port forwarding from config (`ports = ["3000:3000"]`) | `nat.Port`, `nat.PortMap`, `nat.PortSet` from `github.com/docker/go-connections/nat`; `HostConfig.PortBindings`, `container.Config.ExposedPorts` |
| CFG-17 | Resource limits from config (memory, cpus, pids_limit) | Already complete in Phase 6 — no implementation needed |
| CFG-18 | Hooks support (pre_build, post_stop shell commands) | `exec.Command` with `Dir` set to repoDir; `cmd.Run()` sequential; inherited env via `cmd.Env = nil` (inherits parent env) |
</phase_requirements>

## Summary

Phase 7 extends the Docker Manager from Phase 6 by adding runtime configuration injection. All work stays in the `internal/docker` package. The new `env.go` file encapsulates env collection, glob matching, .env file parsing, and pre-launch validation. Existing `buildMounts()` and `createContainer()` methods gain SSH socket mounts, auth config dir mounts, and port bindings. The `buildImage()` method gains proxy build-args. The `Launch()` and `Stop()` methods gain hook execution.

Every decision in this phase has already been locked down via the CONTEXT.md discussion. The implementation is straightforward extension of established Phase 6 patterns: `fmt.Errorf("context: %w", err)` error wrapping, `*bool` pointer fields with nil-as-default, `mount.TypeBind` for sockets and config dirs, and `os/exec.Command` for hook execution.

The only external API that needs careful study is `github.com/docker/go-connections/nat` for port binding — confirmed to be already an indirect dependency via `github.com/docker/docker v28.5.2`. `nat.PortMap` (type `map[nat.Port][]nat.PortBinding`) maps to `HostConfig.PortBindings`; `nat.PortSet` (type `map[nat.Port]struct{}`) maps to `container.Config.ExposedPorts`.

**Primary recommendation:** Create `env.go` first (pure logic, fully unit-testable without Docker), then extend `buildMounts()`, then `createContainer()`, then hook execution in `launch.go` and `manager.go`.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `path/filepath` | stdlib | `filepath.Match` for glob matching | Spec §4.6 mandates this exact semantics |
| `os` | stdlib | `os.Environ()`, `os.Getenv()`, `os.Stat()` for env/file detection | No alternative |
| `os/exec` | stdlib | Hook command execution | Established project pattern (see `manager.go:attachInteractive`) |
| `bufio` | stdlib | .env file line scanning | Established project pattern (see `build.go:streamBuildOutput`) |
| `runtime` | stdlib | `runtime.GOOS` for macOS detection | Established pattern in `platform.go` |
| `strings` | stdlib | `strings.SplitN` for port/env parsing | Established project pattern |
| `github.com/docker/go-connections/nat` | v0.6.0 | Port binding types (`nat.Port`, `nat.PortMap`, `nat.PortSet`) | Required by Docker SDK HostConfig/Config fields |

### Docker SDK Types (already imported)
| Field | Location | Type |
|-------|----------|------|
| `container.Config.Env` | `github.com/docker/docker/api/types/container` | `[]string` |
| `container.Config.ExposedPorts` | same | `nat.PortSet` = `map[nat.Port]struct{}` |
| `container.HostConfig.PortBindings` | same | `nat.PortMap` = `map[nat.Port][]nat.PortBinding` |
| `mount.TypeBind` | `github.com/docker/docker/api/types/mount` | Already imported in `manager.go` |
| `types.ImageBuildOptions.BuildArgs` | `github.com/docker/docker/api/types` | `map[string]*string` (pointer to distinguish "" from unset) |

**No new go.mod dependencies required.** `github.com/docker/go-connections v0.6.0` is already an indirect dependency — just needs a direct import.

**Installation:**
```bash
# No new packages — nat is already in go.sum as indirect dep
# Import path: "github.com/docker/go-connections/nat"
```

## Architecture Patterns

### Recommended Project Structure
```
internal/docker/
├── env.go          # NEW: env collection, glob match, .env parse, required-var validation
├── manager.go      # EXTEND: buildMounts() gains SSH + auth config mounts
├── manager.go      # EXTEND: createContainer() gains Env, PortBindings, ExposedPorts
├── launch.go       # EXTEND: Launch() gains pre-launch validation + pre_build hooks
├── build.go        # EXTEND: buildImage() gains proxy BuildArgs
├── manager.go      # EXTEND: Stop() gains post_stop hooks
└── manager_test.go # EXTEND: new test functions for all above behaviors
```

### Pattern 1: env.go — Env Collection and Glob Matching

**What:** Single file that owns all env-var logic for container injection
**When to use:** Called from `createContainer()` and `Launch()`

```go
// CollectForwardedEnv returns a []string{"KEY=VALUE"} slice for container.Config.Env.
// Iterates os.Environ(), matches each key against patterns using filepath.Match.
// If a pattern does not match any host var, warns to stderr (non-fatal per spec §4.5).
func CollectForwardedEnv(patterns []string) []string {
    // ...
}

// ValidateRequiredEnv checks that all required vars are present in the combined
// env (host + .env file). Returns the first missing var as an error.
func ValidateRequiredEnv(required []string, envFile string) error {
    // ...
}

// ParseEnvFile reads a .env file (KEY=VALUE, skip # comments and blank lines).
// Returns map[string]string for use in validation and env injection.
func ParseEnvFile(path string) (map[string]string, error) {
    // ...
}
```

### Pattern 2: SSH Agent Socket Mount

**What:** Bind mount `SSH_AUTH_SOCK` socket into container at `/run/ssh-agent.sock`
**When to use:** When `auth.forward_ssh_agent = true` AND not macOS AND socket exists

```go
// In buildMounts():
if m.config.Auth.ForwardSSHAgent != nil && *m.config.Auth.ForwardSSHAgent {
    if runtime.GOOS == "darwin" {
        fmt.Fprintf(os.Stderr, "Warning: SSH agent forwarding is not available on macOS "+
            "(domain sockets cannot be bind-mounted). "+
            "SSH operations inside the container will not have agent access.\n")
    } else {
        sock := os.Getenv("SSH_AUTH_SOCK")
        if sock == "" {
            fmt.Fprintf(os.Stderr, "Warning: SSH_AUTH_SOCK is not set or socket not found. "+
                "SSH agent forwarding skipped.\n")
        } else if _, err := os.Stat(sock); err == nil {
            mounts = append(mounts, mount.Mount{
                Type:     mount.TypeBind,
                Source:   sock,
                Target:   "/run/ssh-agent.sock",
                ReadOnly: true,
            })
            // SSH_AUTH_SOCK env var added to container via CollectForwardedEnv extension
        }
    }
}
```

### Pattern 3: Auth Config Dir Mounts

**What:** Bind mount host config dirs (e.g., `~/.claude`) to `<dir>.host` in container
**When to use:** When `mount_home_config = true` (default), in `buildMounts()`

```go
// mount_home_config defaults to true when nil (same nil-as-default pattern as persist_home)
mountHomeConfig := m.config.Auth.MountHomeConfig == nil || *m.config.Auth.MountHomeConfig
if mountHomeConfig {
    h, _ := harness.Get(m.config.Zone.Harness, &m.config.Harness)
    for _, dir := range configDirsForHarness(h) {
        expanded := expandHome(dir)
        if _, err := os.Stat(expanded); os.IsNotExist(err) {
            // debug log — not an error per spec §4.10
            continue
        }
        mounts = append(mounts, mount.Mount{
            Type:     mount.TypeBind,
            Source:   expanded,
            Target:   expanded + ".host",
            ReadOnly: true,
        })
    }
}
```

The entrypoint already generates `configCopyCmd()` commands (Phase 5). No entrypoint changes needed.

### Pattern 4: Port Binding

**What:** Parse `"hostPort:containerPort"` strings into Docker API types
**When to use:** In `createContainer()` after parsing `m.config.Workspace.Ports`

```go
// In createContainer(), before cfg/hostCfg construction:
portBindings, exposedPorts, err := parsePortBindings(m.config.Workspace.Ports)
if err != nil {
    return "", fmt.Errorf("parse ports: %w", err)
}

// portBindings goes to hostCfg.PortBindings
// exposedPorts goes to cfg.ExposedPorts

func parsePortBindings(ports []string) (nat.PortMap, nat.PortSet, error) {
    bindings := nat.PortMap{}
    exposed := nat.PortSet{}
    seenHostPorts := map[string]bool{}

    for _, p := range ports {
        parts := strings.SplitN(p, ":", 2)
        if len(parts) != 2 {
            return nil, nil, fmt.Errorf("invalid port entry %q: expected hostPort:containerPort", p)
        }
        hostPort, containerPort := parts[0], parts[1]
        if err := validatePort(hostPort); err != nil {
            return nil, nil, fmt.Errorf("invalid host port in %q: %w", p, err)
        }
        if err := validatePort(containerPort); err != nil {
            return nil, nil, fmt.Errorf("invalid container port in %q: %w", p, err)
        }
        if seenHostPorts[hostPort] {
            return nil, nil, fmt.Errorf("conflicting port binding: host port %s appears more than once", hostPort)
        }
        seenHostPorts[hostPort] = true

        natPort, _ := nat.NewPort("tcp", containerPort)
        bindings[natPort] = []nat.PortBinding{{HostPort: hostPort}}
        exposed[natPort] = struct{}{}
    }
    return bindings, exposed, nil
}
```

### Pattern 5: Proxy Build-Args Injection

**What:** Pass proxy config as `--build-arg` during image build
**When to use:** In `buildImage()`, before `m.client.ImageBuild()` call

```go
func (m *Manager) resolveProxy() (httpProxy, httpsProxy, noProxy string) {
    // Config takes precedence over host env
    httpProxy = m.config.Network.HTTPProxy
    if httpProxy == "" {
        httpProxy = firstEnv("HTTP_PROXY", "http_proxy")
    }
    httpsProxy = m.config.Network.HTTPSProxy
    if httpsProxy == "" {
        httpsProxy = firstEnv("HTTPS_PROXY", "https_proxy")
    }
    noProxy = m.config.Network.NoProxy
    if noProxy == "" {
        noProxy = firstEnv("NO_PROXY", "no_proxy")
    }
    return
}

// In buildImage(), add to ImageBuildOptions:
httpProxy, httpsProxy, noProxy := m.resolveProxy()
buildArgs := map[string]*string{}
if httpProxy != "" {
    v := httpProxy
    buildArgs["HTTP_PROXY"] = &v
    buildArgs["http_proxy"] = &v
}
// ... same for https, no_proxy
```

Note: `BuildArgs` requires `*string` (pointer) to distinguish empty string from unset. Use a local variable and take its address.

### Pattern 6: Hook Execution

**What:** Run shell commands on the HOST via `os/exec` before build / after stop
**When to use:** `pre_build` in `Launch()` before `buildIfNeeded()`; `post_stop` in `Stop()` after container removal

```go
// runHooks executes a list of shell commands sequentially on the host.
// Each command runs with the repo directory as working dir and inherits parent env.
// failFast=true means first failure returns error; false means warn and continue.
func (m *Manager) runHooks(cmds []string, failFast bool) error {
    for _, cmd := range cmds {
        c := exec.Command("sh", "-c", cmd)
        c.Dir = m.repoDir
        // c.Env = nil inherits parent process environment (Go default)
        c.Stdout = os.Stdout
        c.Stderr = os.Stderr
        if err := c.Run(); err != nil {
            if failFast {
                return fmt.Errorf("pre_build hook failed: %w", err)
            }
            fmt.Fprintf(os.Stderr, "Warning: post_stop hook failed: %v\n", err)
        }
    }
    return nil
}
```

### Pattern 7: .env File Parsing

**What:** Parse Docker-compatible .env format (KEY=VALUE, skip comments and blank lines)
**When to use:** In `ValidateRequiredEnv()` and in `CollectForwardedEnv()` to supplement host env

Docker-compatible .env format rules:
- Lines beginning with `#` are comments — skip
- Empty lines — skip
- `KEY=VALUE` — add to map; VALUE may be empty
- `KEY` without `=` — skip (no value)
- No shell variable expansion, no quotes stripping required (Docker does not expand quotes)

```go
func ParseEnvFile(path string) (map[string]string, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read env file: %w", err)
    }
    result := map[string]string{}
    scanner := bufio.NewScanner(bytes.NewReader(data))
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        idx := strings.Index(line, "=")
        if idx < 0 {
            continue // skip KEY-only lines
        }
        key := line[:idx]
        val := line[idx+1:]
        result[key] = val
    }
    return result, scanner.Err()
}
```

### Anti-Patterns to Avoid

- **Mounting `~/.ssh` directly:** The spec explicitly blocks this as a dangerous mount (CFG-06). SSH agent socket forwarding is the correct pattern.
- **Writing env vars to Dockerfile:** Secrets baked into image layers persist in history. All secrets injection must be at runtime via `container.Config.Env`.
- **Passing `BuildArgs` with value pointer to loop variable:** In Go, taking `&v` of a range variable captures the final value. Always assign to a local variable before taking its address.
- **Expanding proxy env vars with both uppercase and lowercase blindly:** Check config first, then host env. Don't double-set if config already has a value.
- **Hardcoding SSH agent socket path without checking `SSH_AUTH_SOCK`:** On Linux, the path is not fixed (e.g., `/tmp/ssh-XXXXXX/agent.N`). Always read from `SSH_AUTH_SOCK`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Port string parsing | Custom string splitter | `nat.NewPort`, `nat.ParsePort` | Handles edge cases (ranges, proto suffixes) |
| Glob matching | Custom regex/wildcard | `filepath.Match` | Spec mandates this exact function |
| Memory/CPU parsing | Custom parser | `parseMemoryBytes`/`parseNanoCPUs` (Phase 6) | Already in `resources.go`, tested, handles all units |
| .env file parsing | Third-party library (godotenv) | stdlib `bufio.Scanner` | Docker format is simple, no dependency needed |

**Key insight:** The Docker SDK already provides `nat` types for port binding. Using anything else would create a type mismatch with `HostConfig.PortBindings`.

## Common Pitfalls

### Pitfall 1: filepath.Match requires exact segment matching
**What goes wrong:** `filepath.Match("AWS_*", "AWS_PROFILE")` returns `true`, but `filepath.Match("AWS_*", "AWS_PROFILE_FOO")` also returns `true`. This is correct behavior. But `filepath.Match("*", "AWS_PROFILE")` returns `true` matching everything. Plan carefully — forward_env patterns should be specific.
**Why it happens:** `*` in filepath.Match matches any sequence of non-Separator characters. Since env var names have no path separator, `*` matches any substring.
**How to avoid:** This is actually correct behavior per spec §4.6. Document that `AWS_*` matches ALL vars starting with `AWS_`.

### Pitfall 2: BuildArgs pointer loop variable
**What goes wrong:**
```go
for k, v := range proxyMap {
    buildArgs[k] = &v  // BUG: all pointers point to same loop variable
}
```
**Why it happens:** Go range loop reuses the same variable address.
**How to avoid:** Always copy to a local before taking address:
```go
val := v
buildArgs[k] = &val
```

### Pitfall 3: macOS SSH_AUTH_SOCK domain socket bind-mount
**What goes wrong:** macOS Docker Desktop runs containers inside a Linux VM. The host's `SSH_AUTH_SOCK` is a domain socket path that exists on macOS but not inside the VM's filesystem. Attempting to bind-mount it silently fails or produces an opaque error.
**Why it happens:** Docker Desktop on macOS does not forward arbitrary Unix domain sockets from the host macOS environment to the container.
**How to avoid:** Use `runtime.GOOS == "darwin"` check. Warn and skip — do NOT abort (D-10).

### Pitfall 4: tilde expansion in config dir paths
**What goes wrong:** `h.HomeConfigDir()` returns `"~/.claude"`. `os.Stat("~/.claude")` fails — Go's `os.Stat` does not expand `~`.
**Why it happens:** Shell tilde expansion is a shell feature, not a Go stdlib feature.
**How to avoid:** Use `os.UserHomeDir()` and `strings.TrimPrefix(dir, "~")` to expand the path. Or use `filepath.Join(home, dir[2:])`.

### Pitfall 5: env_file path resolution
**What goes wrong:** If `auth.env_file = "secrets.env"`, resolving relative to cwd gives the wrong path when zone is run from a subdirectory.
**Why it happens:** `os.Open("secrets.env")` uses the process working directory, which may not be the repo root.
**How to avoid:** Resolve relative to `m.repoDir` (not cwd): `filepath.Join(m.repoDir, m.config.Auth.EnvFile)`.

### Pitfall 6: nat.PortMap and ExposedPorts must stay in sync
**What goes wrong:** Setting `HostConfig.PortBindings` without setting `container.Config.ExposedPorts` causes Docker to silently ignore the binding.
**Why it happens:** Docker requires `ExposedPorts` in the container config to match the `PortBindings` in the host config.
**How to avoid:** In `parsePortBindings()`, return both `nat.PortMap` and `nat.PortSet`, and apply both.

### Pitfall 7: post_stop hooks and Stop() return value
**What goes wrong:** If post_stop hook fails and Stop() returns an error, the caller may re-run Stop thinking the container wasn't cleaned up, when in fact the container IS gone.
**Why it happens:** Stop's primary contract is "container stopped and removed". Hook failure is a secondary concern.
**How to avoid:** Per D-31, warn to stderr and return nil from Stop() when only the post_stop hook fails.

## Code Examples

Verified patterns from actual project source:

### Existing: container.Config.Env injection (confirmed from manager.go pattern)
```go
// Source: internal/docker/manager.go createContainer() — Config struct
cfg := &container.Config{
    Image:  imageID,
    Labels: ContainerLabels(m.repoDir, m.config.Zone.Harness),
    Env:    collectEnv(m.config),  // NEW in Phase 7
}
```

### Existing: mount.TypeBind (confirmed from manager.go buildMounts)
```go
// Source: internal/docker/manager.go buildMounts()
mounts = append(mounts, mount.Mount{
    Type:     mount.TypeBind,
    Source:   sock,        // SSH_AUTH_SOCK value
    Target:   "/run/ssh-agent.sock",
    ReadOnly: true,
})
```

### Existing: BuildArgs pattern (confirmed from build/build.go)
```go
// Source: github.com/docker/docker/api/types/build/build.go
// BuildArgs map[string]*string — pointer to distinguish "" from unset
httpVal := httpProxy
buildArgs := map[string]*string{
    "HTTP_PROXY":  &httpVal,
    "http_proxy":  &httpVal,
}
buildResp, err := m.client.ImageBuild(ctx, ctx2, types.ImageBuildOptions{
    Tags:       []string{containerName + ":latest"},
    Dockerfile: "Dockerfile",
    Remove:     true,
    NoCache:    noCache,
    BuildArgs:  buildArgs,  // NEW in Phase 7
})
```

### Existing: exec.Command with Dir (pattern from manager.go, adding Dir)
```go
// Source: internal/docker/manager.go attachInteractive — os/exec pattern
c := exec.Command("sh", "-c", hookCmd)
c.Dir = m.repoDir   // repo root as working dir
c.Stdout = os.Stdout
c.Stderr = os.Stderr
if err := c.Run(); err != nil {
    return fmt.Errorf("pre_build hook %q failed: %w", hookCmd, err)
}
```

### Existing: nat.Port usage (confirmed from go-connections v0.6.0)
```go
// Source: github.com/docker/go-connections/nat
import "github.com/docker/go-connections/nat"

natPort, err := nat.NewPort("tcp", "3000")
// natPort is nat.Port("3000/tcp")
bindings[natPort] = []nat.PortBinding{{HostPort: "3000"}}
exposed[natPort] = struct{}{}
```

### SSH_AUTH_SOCK env var injection pattern
```go
// The SSH_AUTH_SOCK env var must ALSO be injected as a container env var
// pointing to the mount target path — not the host socket path.
// This env var is added in createContainer() to container.Config.Env,
// alongside the other forwarded env vars.
const sshAgentTargetPath = "/run/ssh-agent.sock"
// ...
cfg.Env = append(cfg.Env, "SSH_AUTH_SOCK="+sshAgentTargetPath)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Passing `--env-file` as CLI arg | Docker SDK `container.Config.Env []string{"KEY=VALUE"}` | SDK-based approach from Phase 6 | All env injection goes through SDK, not CLI |
| `godotenv` third-party library | stdlib `bufio.Scanner` | Established project policy | No new dependencies for simple format |

**Deprecated/outdated:**
- `types.ImageBuildOptions` is deprecated in Docker SDK v28 in favor of `build.ImageBuildOptions`, but the project imports via the alias in `types_deprecated.go` which still works. The existing `build.go` already uses `types.ImageBuildOptions` — no change needed.

## Open Questions

1. **SSH agent target path inside container**
   - What we know: The context says `/run/ssh-agent.sock` or similar (Claude's discretion)
   - What's unclear: Whether `/run/` exists and is accessible by the `zone` user
   - Recommendation: Use `/tmp/ssh-agent.sock` as an alternative since `/tmp` is always writable; or use `/run/ssh-agent.sock` (typically world-writable on Linux). Either works — pick one consistently and document it.

2. **Hook command timeout**
   - What we know: Context marks this as Claude's discretion
   - What's unclear: Whether long-running pre_build hooks (e.g., model download) need a timeout
   - Recommendation: No timeout by default. pre_build can be legitimately long (downloading artifacts). User can ctrl+C if needed.

3. **env_file vars and container.Config.Env**
   - What we know: D-17 says pass as Docker `--env-file` equivalent via HostConfig
   - What's unclear: Docker SDK HostConfig doesn't have a direct `EnvFile` field — env files are CLI-only. The SDK equivalent is to parse the file and inject as `container.Config.Env` entries.
   - Recommendation: Parse the .env file in `CollectForwardedEnv()` and merge the resulting key=value pairs into the `container.Config.Env` slice. This is the correct SDK approach (there is no `HostConfig.EnvFile` field in the SDK).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing stdlib + testify v1.11.1 |
| Config file | none — `go test ./...` |
| Quick run command | `go test ./internal/docker/... -count=1 -run TestEnv` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CFG-10 | Glob matching collects correct env vars | unit | `go test ./internal/docker/... -run TestCollectForwardedEnv` | ❌ Wave 0 |
| CFG-10 | Non-matched forward_env var warns to stderr, not error | unit | `go test ./internal/docker/... -run TestCollectForwardedEnv_WarnMissing` | ❌ Wave 0 |
| CFG-11 | Missing required env var fails Launch() before build | unit | `go test ./internal/docker/... -run TestValidateRequiredEnv` | ❌ Wave 0 |
| CFG-11 | Required env from .env file satisfies validation | unit | `go test ./internal/docker/... -run TestValidateRequiredEnv_EnvFile` | ❌ Wave 0 |
| CFG-12 | SSH agent mount added when forward_ssh_agent=true on Linux | unit | `go test ./internal/docker/... -run TestBuildMounts_SSHAgent` | ❌ Wave 0 |
| CFG-12 | SSH agent mount skipped on macOS with warning | unit | `go test ./internal/docker/... -run TestBuildMounts_SSHAgent_macOS` | ❌ Wave 0 |
| CFG-13 | Auth config dir mount added when mount_home_config=true | unit | `go test ./internal/docker/... -run TestBuildMounts_AuthConfig` | ❌ Wave 0 |
| CFG-13 | Missing host config dir skipped without error | unit | `go test ./internal/docker/... -run TestBuildMounts_AuthConfig_MissingDir` | ❌ Wave 0 |
| CFG-14 | env_file missing returns error | unit | `go test ./internal/docker/... -run TestParseEnvFile_Missing` | ❌ Wave 0 |
| CFG-14 | env_file parsed correctly (KEY=VALUE, skip comments) | unit | `go test ./internal/docker/... -run TestParseEnvFile` | ❌ Wave 0 |
| CFG-15 | Proxy build-args injected from config | unit | `go test ./internal/docker/... -run TestResolveProxy` | ❌ Wave 0 |
| CFG-15 | Proxy auto-detected from host env when config empty | unit | `go test ./internal/docker/... -run TestResolveProxy_AutoDetect` | ❌ Wave 0 |
| CFG-16 | Port bindings parsed from "host:container" format | unit | `go test ./internal/docker/... -run TestParsePortBindings` | ❌ Wave 0 |
| CFG-16 | Conflicting host port returns error | unit | `go test ./internal/docker/... -run TestParsePortBindings_Conflict` | ❌ Wave 0 |
| CFG-18 | pre_build hooks run before buildIfNeeded | unit | `go test ./internal/docker/... -run TestLaunch_PreBuildHook` | ❌ Wave 0 |
| CFG-18 | pre_build failure aborts launch | unit | `go test ./internal/docker/... -run TestLaunch_PreBuildHook_Failure` | ❌ Wave 0 |
| CFG-18 | post_stop hooks run after container removal | unit | `go test ./internal/docker/... -run TestStop_PostStopHook` | ❌ Wave 0 |
| CFG-18 | post_stop failure warns but Stop returns nil | unit | `go test ./internal/docker/... -run TestStop_PostStopHook_Failure` | ❌ Wave 0 |
| CFG-17 | (no new tests — already covered in Phase 6) | n/a | n/a | ✅ existing |

### Sampling Rate
- **Per task commit:** `go test ./internal/docker/... -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/docker/env.go` — new file; all CFG-10, CFG-11, CFG-14 tests depend on it
- [ ] Test functions in `manager_test.go` for CFG-12, CFG-13, CFG-16, CFG-15, CFG-18
- [ ] No new test files needed — extend existing `manager_test.go` following established patterns

## Sources

### Primary (HIGH confidence)
- Direct code inspection: `internal/docker/manager.go` — `buildMounts()`, `createContainer()`, `Stop()` extension points
- Direct code inspection: `internal/docker/launch.go` — `Launch()` state machine; pre-launch validation insertion point at line 124
- Direct code inspection: `internal/docker/build.go` — `buildImage()` for proxy build-arg injection
- Direct code inspection: `internal/docker/harness_bridge.go` — `configCopyCmd()` (Phase 5 copy-on-start pattern confirmed)
- Direct code inspection: `internal/harness/claude_code.go` — `RequiredEnvVars()` returns `["ANTHROPIC_API_KEY"]`, `HomeConfigDir()` returns `"~/.claude"`
- Direct code inspection: `internal/harness/custom.go` — `RequiredEnvVars()` returns `c.config.RequiredEnv`
- Direct code inspection: `internal/config/types.go` — `AuthConfig`, `HooksConfig`, `WorkspaceConfig.Ports`, `NetworkConfig` all defined
- Direct SDK inspection: `/home/claude/go/pkg/mod/github.com/docker/docker@v28.5.2+incompatible/api/types/container/hostconfig.go` — `PortBindings nat.PortMap`
- Direct SDK inspection: `/home/claude/go/pkg/mod/github.com/docker/docker@v28.5.2+incompatible/api/types/container/config.go` — `ExposedPorts nat.PortSet`
- Direct SDK inspection: `/home/claude/go/pkg/mod/github.com/docker/docker@v28.5.2+incompatible/api/types/build/build.go` — `BuildArgs map[string]*string`
- Direct SDK inspection: `/home/claude/go/pkg/mod/github.com/docker/go-connections@v0.6.0/nat/nat.go` — `Port`, `PortMap`, `PortSet`, `PortBinding` types

### Secondary (MEDIUM confidence)
- `zone-spec.md` §4.5-4.11 (lines 434-539) — behavior specifications for all Phase 7 features
- `07-CONTEXT.md` — all implementation decisions pre-locked (D-01 through D-32)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — verified by reading go.mod, go-connections package directly, Docker SDK source
- Architecture: HIGH — extension points confirmed by reading Phase 6 source; no new architectural patterns needed
- Pitfalls: HIGH — identified from code structure (tilde expansion, pointer loop variable, nat.PortMap/ExposedPorts sync)
- API types: HIGH — read directly from module cache

**Research date:** 2026-03-30
**Valid until:** 2026-05-01 (Docker SDK and Go stdlib are stable; nat package is stable)
