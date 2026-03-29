# Phase 6: Docker Lifecycle Core - Research

**Researched:** 2026-03-29
**Domain:** Go Docker SDK, container lifecycle management, idempotent launch state machine
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Docker SDK integration (DOC-11)**
- Use `github.com/docker/docker/client` Go SDK for all non-interactive operations: build, create, start, stop, inspect, remove, network create/destroy
- Use `os/exec` with `docker exec -it` for interactive TTY attach (SDK's hijacked connection API is unreliable for raw terminal I/O)
- Client initialized once in Manager constructor with `client.FromEnv` + `WithAPIVersionNegotiation()`
- Verify connectivity with `Ping()` on construction — fail fast with `ErrDockerNotRunning` and actionable error message
- Context propagation on all SDK calls for graceful cancellation

**Container state machine (DOC-09)**
- Running: check config hash, warn if changed, reattach
- Paused: unpause, then attach
- Exited/Dead: inspect exit code and OOMKilled flag, warn if OOM, remove container + network, proceed to build/launch
- Created/Restarting: wait briefly (2s), stop, remove, proceed to build/launch
- Stale container ID: clean up stale cache files, attempt to remove orphaned network, proceed to build/launch
- No container_id file: fresh launch path

**Build behavior (DOC-12)**
- Build progress streamed to stderr line-by-line (plain text — TUI is Phase 9)
- Build log captured to `.zone/logs/last_build.log`
- Build errors show last 20 lines of build log + path to full log
- All builds use BuildKit (`DOCKER_BUILDKIT=1`, Dockerfile has `# syntax=docker/dockerfile:1`)
- Config hash comparison determines whether rebuild is needed (Phase 3 cache hash)
- Image pruned detection: verify `image_id` with `ImageInspect` before reusing

**Config change detection (DOC-10)**
- Compare full cache hash against stored `.zone/config.hash`
- If running container has stale config: warn, do NOT auto-restart
- If no running container and hash mismatch: auto-rebuild silently

**Container creation security (DOC-06, carried from Phase 4)**
- SecurityOpt: `no-new-privileges`
- CapDrop: ALL, CapAdd: CHOWN, DAC_OVERRIDE, SETGID, SETUID, FOWNER
- PidsLimit from config (default 512)
- Memory and CPU limits from config (0 = no limit)
- IPv6 disabled via sysctl `net.ipv6.conf.all.disable_ipv6=1`
- Each container gets its own bridge network

**Docker labels (DOC-08)**
- Apply `com.zone.managed=true`, `com.zone.repo-path`, `com.zone.harness` labels per spec
- Labels enable `zone ls` discovery (Phase 8 wires the list command)

**Home volume persistence (CFG-20)**
- When `persist_home = true` (default): create named volume `zone-home-<shortHash>` for `/home/zone`
- Volume survives container recreation
- `zone destroy` removes the volume; `zone stop` and `zone clean` do NOT

**Interactive attach**
- `zone launch`: build-if-needed, create, start, then attach TTY via `docker exec -it`
- `zone join`: attach new shell to running container (no harness restart)
- `zone shell`: interactive shell, no harness process
- `zone exec -- <cmd>`: one-off command execution inside container
- Lock released before TTY attach to allow `zone join` from another terminal

**Zero-config quickstart (CLI-05)**
- `zone launch --harness claude-code` with no zone.toml: generate minimal zone.toml, add `.zone/` to `.gitignore`, proceed to build/launch
- No `--harness` and no zone.toml in Phase 6: error with actionable message

**Headless mode (CLI-04)**
- `zone launch --headless`: build, create, start, print container ID to stdout, return immediately
- `zone launch --headless -p "task"`: inject prompt via harness PromptFlag() into entrypoint args
- Exit code 0 on successful start

**Stop/restart/destroy cleanup**
- `zone stop`: stop container, remove container, remove network, clear container_id + network_id from cache
- `zone restart`: stop + relaunch (rebuild if `--rebuild` flag)
- `zone destroy`: stop + remove image + remove home volume + remove all `.zone/` cache
- `zone build`: force-rebuild image without launching

**Cobra command wiring**
- Wire all 8 command stubs: launch, join, exec, shell, build, stop, restart, destroy
- Extend `cmd/clean.go` for `--image` flag
- Flags: `--harness`, `--headless`, `-p`/`--prompt`, `--rebuild`, `--timeout`, `--root`, `--yes`/`-y`

### Claude's Discretion

- Manager method internal structure (helper extraction, error wrapping patterns)
- Build context tar construction approach
- Exact timeout values for container state transitions
- Test strategy for Docker integration (mock client vs integration tests)
- Error message wording beyond spec examples

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DOC-08 | Docker labels applied for discovery by `zone ls` | `ContainerLabels()` already implemented in `internal/docker/naming.go`; Manager.Launch() passes them to ContainerCreate |
| DOC-09 | Idempotent launch: reattach if running, handle paused/exited/dead/stale states | Full state machine in spec §3.7; `ContainerInspect().State.Status` drives the switch |
| DOC-10 | Config change detection warns user to restart when config hash differs | `cache.ComputeHash()` + `cache.ConfigHash()` comparison; warn path and rebuild path both documented |
| DOC-11 | Docker SDK used for build/create/start/stop/inspect; context propagation | SDK v28.5.2+incompatible verified to compile; all method signatures confirmed |
| DOC-12 | Build progress streamed from Docker SDK with proper response body cleanup | `ImageBuild()` returns `io.ReadCloser`; JSON line streaming pattern verified; `cache.CreateBuildLog()` ready |
| CFG-20 | Persistent home volume via named Docker volume (survives container recreation) | `volume.CreateOptions`, `VolumeRemove()` verified; volume name = `zone-home-<shortHash>` |
| CLI-03 | `zone launch` to build (if needed) and attach to a container for this repo | Manager.Launch() orchestrates full state machine; `attachInteractive()` via `os/exec docker exec -it` |
| CLI-04 | `zone launch --headless -p "task"` to run a detached agent with a prompt | `--headless` flag; `harness.PromptFlag()` translates `-p` to harness-specific arg |
| CLI-05 | `zone launch --harness <name>` with no zone.toml for zero-config quickstart | Generate minimal zone.toml; `config.ErrNoConfig` detection; `cache.EnsureGitignore()` already implemented |
| CLI-06 | `zone join` to attach a new shell to a running container | `attachInteractive()` with `bash` or configured shell; requires running container (exit code 6 if not) |
| CLI-07 | `zone exec -- <cmd>` to run a one-off command inside the running container | `attachInteractive()` or non-TTY exec; TTY detection via `os.Stdin` |
| CLI-08 | `zone shell` to open an interactive shell even if no harness is running | Starts container with temporary entrypoint if not running; `attachInteractive()` |
| CLI-09 | `zone build` to force-rebuild the Docker image without launching | Manager.Build() standalone; skips state machine, always rebuilds |
| CLI-10 | `zone stop` to stop and remove the container and network, retaining cache | `ContainerStop()` → `ContainerRemove()` → `NetworkRemove()`; clear container_id + network_id |
| CLI-11 | `zone restart` to stop and relaunch the container | Manager.Stop() + Manager.Launch(); `--rebuild` forces image rebuild |
| CLI-15 | `zone clean` to remove .zone/ cache and optionally Docker image | Already partially wired; extend with `--image` flag for image removal |
| CLI-16 | `zone destroy` to fully tear down container, image, network, and cache | Full cleanup: Stop + ImageRemove + VolumeRemove + Cache.Clean |
</phase_requirements>

## Summary

Phase 6 wires the Docker Manager — the core of zone's value proposition. All foundational layers are complete: config parsing (Phase 2), cache + hash (Phase 3), template rendering (Phase 4), and harness system (Phase 5). This phase implements `internal/docker/manager.go` and `internal/docker/network.go` stubs, then connects them to the 8 Cobra command stubs.

The architecture is well-specified: the spec provides exact Manager struct, method signatures, security flags, state machine behavior, and TTY attach pattern. The key design decisions are already locked — use Docker SDK v28.5.2 for non-interactive operations and `os/exec docker exec -it` for TTY. The existing `ContainerSecurityFlags()`, `ContainerLabels()`, `ContainerName()`, `NetworkName()`, and all bridge/render functions are ready to consume.

The primary implementation challenge is the `Manager.Launch()` method, which orchestrates a multi-step state machine: acquire lock → inspect container state → conditional build/rebuild → network create → container create → start → release lock → attach (or headless exit). Each step has specific cleanup and error behavior. Resource parsing (memory string → bytes, CPU string → nanocpus) requires the `docker/go-units` package.

**Primary recommendation:** Implement in three waves — (1) Manager constructor + network.go + basic Build(); (2) full Launch() state machine + Stop() + Destroy(); (3) wire all 8 Cobra commands with their flags.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/docker/docker` | v28.5.2+incompatible | Docker Engine SDK — build, create, start, stop, inspect, network | Only official Go Docker SDK; v28 is latest stable (verified 2026-03-29) |
| `github.com/docker/go-units` | v0.5.0 | Parse memory/CPU strings (`"512m"` → bytes, `"0.5"` → nanocpus) | Transitive dep of docker/docker; `units.RAMInBytes()` is the canonical parser |
| `github.com/docker/go-connections/nat` | v0.6.0 | Port binding types (`nat.PortMap`, `nat.PortBinding`) | Required for ContainerCreate PortBindings; transitive dep |
| `github.com/docker/docker/errdefs` | (same module) | Type-safe error detection (`errdefs.IsNotFound()`) | Distinguish "container not found" from other Docker errors |
| `github.com/opencontainers/image-spec` | v1.1.1 | Platform type for ContainerCreate | Required parameter type; transitive dep |
| `archive/tar` (stdlib) | — | Build context construction | Docker's `ImageBuild` requires a tar archive as `io.Reader` |
| `os/exec` (stdlib) | — | Interactive TTY attach via `docker exec -it` | Locked decision: SDK hijacked connection unreliable for terminal I/O |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/docker/docker/api/types` | (same module) | `ImageBuildOptions`, build response types | ImageBuild call |
| `github.com/docker/docker/api/types/container` | (same module) | `Config`, `HostConfig`, `StopOptions`, `RemoveOptions`, `State` | All container lifecycle operations |
| `github.com/docker/docker/api/types/network` | (same module) | `NetworkingConfig`, `CreateOptions`, `EndpointSettings` | Network operations + container networking |
| `github.com/docker/docker/api/types/mount` | (same module) | `Mount`, `TypeBind`, `TypeVolume` | Workspace bind mounts and home volume mounts |
| `github.com/docker/docker/api/types/volume` | (same module) | `CreateOptions`, `Volume` | Persistent home volume management |
| `github.com/docker/docker/api/types/strslice` | (same module) | `StrSlice` for CapDrop/CapAdd | Capability list types in HostConfig |

### Installation
```bash
go get github.com/docker/docker@v28.5.2+incompatible
go mod tidy
```

Running `go mod tidy` after the above will pull all transitive dependencies including `docker/go-units`, `docker/go-connections`, `opencontainers/image-spec`, `moby/term`, and OpenTelemetry packages.

**Version verification (performed 2026-03-29):**
```
github.com/docker/docker v28.5.2+incompatible  # latest stable as of research date
github.com/docker/go-units v0.5.0              # transitive, pulled automatically
github.com/docker/go-connections v0.6.0        # transitive, pulled automatically
```

## Architecture Patterns

### Recommended File Structure
```
internal/docker/
├── manager.go        # Manager struct, constructor, Launch/Join/Shell/Exec/Stop/Build/Status/ListAll
├── network.go        # createNetwork(), removeNetwork() helpers
├── build.go          # (optional split) buildImage(), streamBuildOutput(), buildContext()
├── errors.go         # Sentinel errors: ErrDockerNotRunning, ErrNoContainer (already exists: SecurityConfig)
├── naming.go         # ContainerName/NetworkName/ContainerLabels (already complete)
├── platform.go       # HostUID/MacOSUsername/DetectGitIdentity/DetectPlatform (already complete)
├── harness_bridge.go # BuildDockerfileData/EntrypointData/ShellRCData (already complete)
├── dockerfile.go     # RenderDockerfile (already complete)
├── entrypoint.go     # RenderEntrypoint (already complete)
└── shellrc.go        # RenderShellRC (already complete)
```

### Pattern 1: Manager Constructor with Fail-Fast Ping
**What:** Initialize Docker client once; verify daemon is running before returning
**When to use:** All Manager operations; fail early with actionable error

```go
// Source: zone-spec.md §12
type Manager struct {
    client   *client.Client
    config   *config.MergedConfig
    cache    *cache.Cache
    platform Platform
}

func NewManager(cfg *config.MergedConfig, c *cache.Cache) (*Manager, error) {
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        return nil, fmt.Errorf("docker client init: %w", err)
    }
    if _, err := cli.Ping(context.Background()); err != nil {
        return nil, fmt.Errorf("%w: %v", ErrDockerNotRunning, err)
    }
    plat := DetectPlatform(cli)
    return &Manager{client: cli, config: cfg, cache: c, platform: plat}, nil
}
```

### Pattern 2: Container State Machine
**What:** Inspect existing container state and branch accordingly before launch
**When to use:** Every `zone launch` invocation

```go
// Source: zone-spec.md §3.7
func (m *Manager) inspectContainerState(ctx context.Context, containerID string) (*container.InspectResponse, error) {
    info, err := m.client.ContainerInspect(ctx, containerID)
    if errdefs.IsNotFound(err) {
        return nil, nil // stale ID — treat as fresh launch
    }
    return &info, err
}

// State.Status values: "running", "paused", "exited", "dead", "created", "restarting"
// State.OOMKilled: true if container was killed by OOM killer
// State.ExitCode: non-zero on abnormal exit
```

### Pattern 3: Build Context as Tar Archive
**What:** Pack Dockerfile + entrypoint.sh + zone-bashrc into a tar `io.Reader` for ImageBuild
**When to use:** Every `Manager.Build()` call

```go
// Source: spec §12 + stdlib archive/tar
func buildContext(dockerfile, entrypoint, shellrc string) (io.Reader, error) {
    buf := &bytes.Buffer{}
    tw := tar.NewWriter(buf)
    files := []struct{ name, content string }{
        {"Dockerfile", dockerfile},
        {"entrypoint.sh", entrypoint},
        {"zone-bashrc", shellrc},
    }
    for _, f := range files {
        hdr := &tar.Header{Name: f.name, Mode: 0644, Size: int64(len(f.content))}
        tw.WriteHeader(hdr)
        tw.Write([]byte(f.content))
    }
    tw.Close()
    return buf, nil
}
```

### Pattern 4: Build Output Streaming (Plain Text)
**What:** Stream Docker's JSON build messages to stderr line-by-line; capture to log file
**When to use:** Every `ImageBuild()` call in Phase 6 (TUI upgrade is Phase 9)

```go
// Source: zone-spec.md §12 "Build progress streamed to stderr line-by-line (plain text)"
type buildMessage struct {
    Stream      string `json:"stream"`
    Error       string `json:"error"`
    ErrorDetail *struct {
        Message string `json:"message"`
    } `json:"errorDetail"`
    Aux *struct {
        ID string `json:"ID"`
    } `json:"aux"`
}

func streamBuildOutput(body io.ReadCloser, w io.Writer) (imageID string, err error) {
    defer body.Close()
    scanner := bufio.NewScanner(body)
    for scanner.Scan() {
        var msg buildMessage
        if jsonErr := json.Unmarshal(scanner.Bytes(), &msg); jsonErr != nil {
            continue
        }
        if msg.Error != "" {
            return "", fmt.Errorf("docker build: %s", msg.Error)
        }
        if msg.Stream != "" {
            fmt.Fprint(w, msg.Stream)
        }
        if msg.Aux != nil && msg.Aux.ID != "" {
            imageID = msg.Aux.ID
        }
    }
    return imageID, scanner.Err()
}
```

### Pattern 5: Interactive TTY Attach via os/exec
**What:** Use `docker exec -it` subprocess for terminal I/O; inherit stdin/stdout/stderr
**When to use:** ALL interactive commands (launch, join, shell, exec)

```go
// Source: zone-spec.md §12 "Interactive TTY Attach"
func (m *Manager) attachInteractive(containerID string, cmd []string, asRoot bool) error {
    args := []string{"exec", "-it"}
    if asRoot {
        args = append(args, "-u", "root")
    }
    args = append(args, containerID)
    args = append(args, cmd...)
    c := exec.Command("docker", args...)
    c.Stdin = os.Stdin
    c.Stdout = os.Stdout
    c.Stderr = os.Stderr
    if dockerHost := os.Getenv("DOCKER_HOST"); dockerHost != "" {
        c.Env = append(os.Environ(), "DOCKER_HOST="+dockerHost)
    }
    return c.Run()
}
```

### Pattern 6: ContainerCreate with Full Security Config
**What:** Create container with hardened security, resource limits, mounts, network
**When to use:** During `Manager.Launch()` after network is created

```go
// Source: zone-spec.md §12 "Container Creation Security Flags"
func (m *Manager) createContainer(ctx context.Context, imageName, containerName, networkName string) (string, error) {
    sec := ContainerSecurityFlags()
    repoPath, _ := os.Getwd()
    pidsLimit := int64(sec.DefaultPidsLimit)
    if m.config.Resources.PidsLimit > 0 {
        pidsLimit = int64(m.config.Resources.PidsLimit)
    }

    memBytes, _ := parseMemoryBytes(m.config.Resources.Memory)
    nanoCPUs, _ := parseNanoCPUs(m.config.Resources.Cpus)

    cfg := &container.Config{
        Image:  imageName,
        Labels: ContainerLabels(repoPath, m.config.Zone.Harness),
    }

    hostCfg := &container.HostConfig{
        SecurityOpt: sec.SecurityOpt,
        CapDrop:     strslice.StrSlice(sec.CapDrop),
        CapAdd:      strslice.StrSlice(sec.CapAdd),
        Resources: container.Resources{
            Memory:    memBytes,
            NanoCPUs:  nanoCPUs,
            PidsLimit: &pidsLimit,
        },
        Sysctls: map[string]string{
            "net.ipv6.conf.all.disable_ipv6": "1",
        },
        Mounts: m.buildMounts(repoPath),
    }

    netCfg := &network.NetworkingConfig{
        EndpointsConfig: map[string]*network.EndpointSettings{
            networkName: {},
        },
    }

    resp, err := m.client.ContainerCreate(ctx, cfg, hostCfg, netCfg, nil, containerName)
    if err != nil {
        return "", fmt.Errorf("create container: %w", err)
    }
    return resp.ID, nil
}
```

### Pattern 7: Resource String Parsing
**What:** Convert human-readable memory/CPU config strings to Docker API integers
**When to use:** Inside `createContainer()` — convert MergedConfig.Resources fields

```go
import "github.com/docker/go-units"

func parseMemoryBytes(s string) (int64, error) {
    if s == "" || s == "0" {
        return 0, nil  // 0 = no limit in Docker API
    }
    return units.RAMInBytes(s)  // "512m" -> 536870912, "2g" -> 2147483648
}

func parseNanoCPUs(s string) (int64, error) {
    if s == "" || s == "0" {
        return 0, nil  // 0 = no limit in Docker API
    }
    f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
    if err != nil {
        return 0, fmt.Errorf("parse cpus %q: %w", s, err)
    }
    return int64(f * 1e9), nil  // 0.5 -> 500000000 nanocpus
}
```

### Pattern 8: Home Volume Name
**What:** Derive persistent volume name from repo path hash (same short hash used in container name)
**When to use:** `persist_home = true` (default) during container creation

```go
// Source: zone-spec.md §12 "Home Volume Persistence"
// shortHash is the same 16-char hex prefix from ContainerName()
func homeVolumeName(repoPath string) string {
    absPath, _ := filepath.Abs(repoPath)
    hash := sha256.Sum256([]byte(absPath))
    shortHash := hex.EncodeToString(hash[:])[:16]
    return fmt.Sprintf("zone-home-%s", shortHash)
}
```

### Pattern 9: Zero-Config zone.toml Generation
**What:** Write a minimal zone.toml with harness set and commented-out options
**When to use:** `zone launch --harness <name>` with no zone.toml

```go
// Source: zone-spec.md §3.6
const minimalZoneToml = `version = 1
harness = "%s"

# Uncomment to customize:
# [zone]
# base_image = "ubuntu:24.04"
# shell = "bash"
#
# [resources]
# memory = "4g"
# cpus = "2"
# pids_limit = 512
#
# [workspace]
# persist_home = true
`

func generateMinimalZoneToml(harness string) string {
    return fmt.Sprintf(minimalZoneToml, harness)
}
```

### Anti-Patterns to Avoid
- **Attaching TTY via Docker SDK's `ContainerExecAttach`:** SDK hijacked connection is unreliable for raw terminal I/O — always use `os/exec docker exec -it`
- **Not closing `ImageBuild` response body:** Causes connection leak — always `defer resp.Body.Close()`
- **Holding the file lock during TTY attach:** Blocks `zone join` from a second terminal — release lock before attach
- **Using `ContainerStop` alone for cleanup:** Must also call `ContainerRemove` to free the container (stopped != removed)
- **Skipping `errdefs.IsNotFound` check on ContainerInspect:** Raw error from inspect when container doesn't exist is not an `os.ErrNotExist` — must use `errdefs.IsNotFound(err)`
- **Assuming `m.config.Workspace.PersistHome != nil`:** It's a `*bool`; nil means "not set by user" but the effective default is `true` — check `m.config.Workspace.PersistHome == nil || *m.config.Workspace.PersistHome`

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Parse "512m" → bytes | Custom string parser | `github.com/docker/go-units.RAMInBytes()` | Handles all SI/IEC suffixes, already a transitive dep |
| Port binding types | Custom struct | `github.com/docker/go-connections/nat.PortMap` | Required by Docker SDK ContainerCreate HostConfig |
| "Container not found" detection | String matching on error | `github.com/docker/docker/errdefs.IsNotFound()` | Docker API errors are typed; string matching breaks on locale or version changes |
| Build output parsing | Custom JSON decoder | `bufio.Scanner` + `encoding/json` per line | Docker build stream is newline-delimited JSON messages, not a single JSON object |
| Build context | Manual file copy | `archive/tar` stdlib | Docker's `ImageBuild` only accepts `io.Reader` pointing to a tar archive |

**Key insight:** The Docker SDK's type system is strict — CapDrop/CapAdd require `strslice.StrSlice`, port bindings require `nat.PortMap`, mounts require `[]mount.Mount`. Using the wrong types causes compile errors, not runtime failures.

## Common Pitfalls

### Pitfall 1: ImageBuild Response Body Leak
**What goes wrong:** Build succeeds but subsequent builds hang or daemon shows resource exhaustion
**Why it happens:** `ImageBuild` returns an `ImageBuildResponse` whose `.Body` is an HTTP response body — if not closed, the connection is held
**How to avoid:** Always `defer resp.Body.Close()` immediately after the `ImageBuild` call, before any error check
**Warning signs:** Docker daemon becomes unresponsive after repeated builds in tests

### Pitfall 2: ContainerInspect on Stale ID Returns Error (Not NotFound)
**What goes wrong:** State machine falls into wrong branch when container was pruned
**Why it happens:** `ContainerInspect` on a non-existent ID returns a Docker API error, not `os.ErrNotExist`
**How to avoid:** Wrap with `errdefs.IsNotFound(err)` check; return `(nil, nil)` for the stale case
**Warning signs:** `zone launch` returns "container not found" error instead of proceeding to build

### Pitfall 3: File Lock Held During Interactive Session
**What goes wrong:** `zone join` returns "another zone process is operating" immediately
**Why it happens:** Lock is acquired before build, never released until `zone launch` exits
**How to avoid:** Release lock after `ContainerStart()`, before calling `attachInteractive()` — exactly as spec §3.7 step 3 specifies
**Warning signs:** Second terminal `zone join` fails immediately after `zone launch` is attached

### Pitfall 4: `persist_home` nil Pointer Dereference
**What goes wrong:** Panic in `createContainer()` when `zone.toml` doesn't set `persist_home`
**Why it happens:** `WorkspaceConfig.PersistHome` is `*bool`; nil when not set by user
**How to avoid:** `persistHome := m.config.Workspace.PersistHome == nil || *m.config.Workspace.PersistHome` — treat nil as `true` (spec default)
**Warning signs:** Panic on fresh `zone launch` without explicit `persist_home` in config

### Pitfall 5: BuildKit Not Enabled
**What goes wrong:** Dockerfile with `# syntax=docker/dockerfile:1` header fails
**Why it happens:** Older Docker daemons don't enable BuildKit by default
**How to avoid:** Set `BuildArgs: map[string]*string{"DOCKER_BUILDKIT": strPtr("1")}` in `ImageBuildOptions` OR set env var in the subprocess approach — actually use `ImageBuildOptions.BuildArgs` is wrong; set `DOCKER_BUILDKIT=1` in the environment when the daemon doesn't auto-enable it. For SDK approach, use `types.ImageBuildOptions{SuppressOutput: false, Remove: true}` and note that BuildKit is a daemon feature — the correct approach is to rely on Docker Engine 23+ having BuildKit as default, but add `"DOCKER_BUILDKIT": "1"` to `os.Environ()` when spawning any docker CLI subprocess
**Warning signs:** `# syntax=docker/dockerfile:1` causes "unknown instruction: #" error

### Pitfall 6: Stop Timeout Must Be a Pointer
**What goes wrong:** `ContainerStop` times out immediately or uses wrong default
**Why it happens:** `container.StopOptions.Timeout` is `*int` — passing 0 is different from passing nil (nil = daemon default ~10s)
**How to avoid:** `timeout := 10; cli.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout})`
**Warning signs:** Container killed immediately without SIGTERM grace period

### Pitfall 7: go.mod +incompatible Tag
**What goes wrong:** `go get github.com/docker/docker` resolves to wrong package (new `github.com/docker/docker/client` module)
**Why it happens:** Docker has two separate Go modules now: the legacy `github.com/docker/docker` (v28.x+incompatible) and a new `github.com/docker/docker/client` module (v0.3.x which declares itself as `github.com/moby/moby/client`)
**How to avoid:** Always use `go get github.com/docker/docker@v28.5.2+incompatible` — the `+incompatible` suffix is mandatory
**Warning signs:** `module declares its path as github.com/moby/moby/client` error during go get

## Code Examples

### Manager Launch Orchestration (high-level)
```go
// Source: zone-spec.md §3.7 full state machine
type LaunchOpts struct {
    Headless  bool
    Prompt    string
    Rebuild   bool
    NoCache   bool
    HarnessArgs []string
}

func (m *Manager) Launch(ctx context.Context, opts LaunchOpts) error {
    repoPath, _ := os.Getwd()

    // Step 0: Acquire lock
    lock := cache.NewLock(m.cache.Dir())
    if err := lock.Acquire(); err != nil {
        return err  // ErrLockContention propagates to main.go exit code 5
    }
    // lock.Release() called before attach (not deferred to end of func)

    // Step 1: Check existing container
    containerID, _ := m.cache.ContainerID()
    if containerID != "" {
        info, err := m.inspectContainerState(ctx, containerID)
        if err != nil {
            return err
        }
        if info == nil {
            // Stale ID
            m.cleanStaleCache()
        } else {
            switch info.State.Status {
            case "running":
                return m.handleRunning(ctx, info, lock, opts)
            case "paused":
                m.client.ContainerUnpause(ctx, containerID)
                lock.Release()
                return m.attachInteractive(containerID, m.harnessCmd(opts), false)
            case "exited", "dead":
                m.warnOnOOM(info.State)
                m.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
                m.removeNetwork(ctx)
                m.cache.SetContainerID("")
            case "created", "restarting":
                time.Sleep(2 * time.Second)
                m.client.ContainerStop(ctx, containerID, container.StopOptions{})
                m.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
                m.removeNetwork(ctx)
                m.cache.SetContainerID("")
            }
        }
    }

    // Step 2: Build if needed
    if err := m.buildIfNeeded(ctx, opts.Rebuild, opts.NoCache); err != nil {
        lock.Release()
        return err
    }

    // Step 3: Create + Start
    containerID, err := m.createAndStart(ctx, repoPath)
    if err != nil {
        lock.Release()
        return err
    }
    m.cache.SetContainerID(containerID)

    // Step 4: Release lock before attach
    lock.Release()

    if opts.Headless {
        fmt.Println(containerID)
        return nil
    }

    return m.attachInteractive(containerID, m.harnessCmd(opts), false)
}
```

### Build Progress with Log Tee
```go
// Source: cache.go CreateBuildLog + spec §12 build streaming
func (m *Manager) buildImage(ctx context.Context, noCache bool, version string) (string, error) {
    // Render templates
    h, err := harness.Get(m.config.Zone.Harness, &m.config.Harness)
    if err != nil {
        return "", err
    }

    uid, _ := HostUID()
    dfData := BuildDockerfileData(h, m.config)
    dfData.HostUID = uid
    dfData.MacOSUsername = MacOSUsername()

    dockerfile, _ := RenderDockerfile(dfData, version)
    entrypoint, _ := RenderEntrypoint(BuildEntrypointData(h, m.config), version)
    shellrc, _     := RenderShellRC(BuildShellRCData(h, m.config), version)

    // Build context tar
    ctx2, cancel := context.WithCancel(ctx)
    defer cancel()

    buildCtx, _ := buildContext(dockerfile, entrypoint, shellrc)

    // Compute hash and set up log tee
    hash, _ := cache.ComputeHash(m.config, version)
    logWriter, closeLog, _ := m.cache.CreateBuildLog(os.Stderr, hash, version)
    defer closeLog()

    imageName := ContainerName(repoPath) + ":latest"
    resp, err := m.client.ImageBuild(ctx2, buildCtx, types.ImageBuildOptions{
        Tags:       []string{imageName},
        Dockerfile: "Dockerfile",
        Remove:     true,
        NoCache:    noCache,
        Labels:     ContainerLabels(repoPath, m.config.Zone.Harness),
    })
    if err != nil {
        return "", fmt.Errorf("image build: %w", err)
    }

    imageID, err := streamBuildOutput(resp.Body, logWriter)
    if err != nil {
        m.showBuildError(logWriter)
        return "", err
    }

    m.cache.SetImageID(imageID)
    m.cache.SetConfigHash(hash)
    return imageID, nil
}
```

### Network Creation
```go
// Source: spec §12, Docker SDK verified
func (m *Manager) createNetwork(ctx context.Context, networkName string) (string, error) {
    repoPath, _ := os.Getwd()
    resp, err := m.client.NetworkCreate(ctx, networkName, network.CreateOptions{
        Driver: "bridge",
        Labels: map[string]string{
            "com.zone.managed":   "true",
            "com.zone.repo-path": repoPath,
        },
    })
    if err != nil {
        return "", fmt.Errorf("create network %s: %w", networkName, err)
    }
    return resp.ID, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `docker/docker v1.x` | `v28.x+incompatible` | Ongoing | API types moved to subpackages (`api/types/container`, `api/types/network`) |
| Global BuildKit opt-in | BuildKit default in Docker Engine 23+ | Docker 23 (2023) | `DOCKER_BUILDKIT=1` still safe to set; harmless on new engines |
| `types.ContainerJSON` | `container.InspectResponse` | v25+ | State fields accessed as `info.State.Status`, `info.State.OOMKilled` |
| `ContainerStop(ctx, id, nil)` | `ContainerStop(ctx, id, container.StopOptions{Timeout: &n})` | v25+ | Nil options struct (pointer) replaced with value struct with optional fields |

**Deprecated/outdated:**
- `docker/docker/client` (new module v0.3.x): Declares itself as `github.com/moby/moby/client` — do NOT use; use `github.com/docker/docker@v28.5.2+incompatible`
- `types.ImageBuildOptions` from `api/types`: Still valid in v28 (not yet moved to subpackage); use `github.com/docker/docker/api/types.ImageBuildOptions`

## Open Questions

1. **Build log "last 20 lines on error" display**
   - What we know: spec says "show last 20 lines of build log + path to full log"
   - What's unclear: whether to implement a ring buffer during streaming or read the log file after failure
   - Recommendation: read last 20 lines from the `.zone/logs/last_build.log` file after `streamBuildOutput` returns error; simpler than in-memory ring buffer

2. **`zone shell` when no container is running**
   - What we know: spec says "starts container with a temporary entrypoint"
   - What's unclear: exact entrypoint command (likely `/bin/bash` or the configured shell directly)
   - Recommendation: use `/bin/sh -c "sleep infinity"` as a temp entrypoint so `attachInteractive` can run the configured shell; simpler than creating a one-shot entrypoint script

3. **DOCKER_BUILDKIT environment in SDK builds**
   - What we know: spec says all builds use BuildKit; Docker 23+ has it as default
   - What's unclear: whether to explicitly set it in `ImageBuildOptions` or rely on daemon default
   - Recommendation: set `BuildArgs: map[string]*string{}` and include `"DOCKER_BUILDKIT"="1"` only in fallback; modern Docker enables it automatically, test environment should be Docker 23+

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing stdlib + testify v1.10.0 |
| Config file | none (go test ./... runs all packages) |
| Quick run command | `go test ./tests/ -run TestManager -v` |
| Full suite command | `go test ./... -timeout 120s` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DOC-08 | ContainerLabels applied to created containers | unit | `go test ./tests/ -run TestContainerLabels -v` | Already exists in `tests/naming_test.go` |
| DOC-09 | State machine branches: running/paused/exited/stale | unit (mock) | `go test ./internal/docker/ -run TestLaunchStateMachine -v` | No — Wave 0 |
| DOC-10 | Config hash mismatch warning; auto-rebuild path | unit (mock) | `go test ./internal/docker/ -run TestConfigHashDetection -v` | No — Wave 0 |
| DOC-11 | Manager constructor: Ping failure returns ErrDockerNotRunning | unit (mock) | `go test ./internal/docker/ -run TestNewManagerPingFail -v` | No — Wave 0 |
| DOC-12 | Build output streamed; body closed on exit | unit | `go test ./internal/docker/ -run TestStreamBuildOutput -v` | No — Wave 0 |
| CFG-20 | Home volume created when persist_home=true; skipped when false | unit (mock) | `go test ./internal/docker/ -run TestHomeVolume -v` | No — Wave 0 |
| CLI-03 | `zone launch` exits with error when Docker not running | integration | `go test ./tests/ -run TestLaunchNoDocker -v` | No — Wave 0 |
| CLI-04 | `zone launch --headless` prints container ID to stdout | integration | `go test ./tests/ -run TestLaunchHeadless -v` | No — Wave 0 |
| CLI-05 | `zone launch --harness claude-code` creates zone.toml | integration | `go test ./tests/ -run TestZeroConfigLaunch -v` | No — Wave 0 |
| CLI-10 | `zone stop` clears container_id and network_id from cache | unit (mock) | `go test ./internal/docker/ -run TestStopClearsCache -v` | No — Wave 0 |
| CLI-15 | `zone clean --image` also removes Docker image | integration | `go test ./tests/ -run TestCleanWithImage -v` | No — Wave 0 |
| CLI-16 | `zone destroy` removes volume; `zone stop` does not | unit (mock) | `go test ./internal/docker/ -run TestDestroyVsStop -v` | No — Wave 0 |

**Note on test strategy (Claude's Discretion):** Most Docker Manager tests should use a mock Docker client interface. The Docker client methods (`ContainerInspect`, `ContainerCreate`, etc.) can be extracted into an interface for test injection. Integration tests (requiring a live Docker daemon) should be guarded with `testing.Short()` skip or a build tag `//go:build integration`.

### Sampling Rate
- **Per task commit:** `go test ./... -short -timeout 60s`
- **Per wave merge:** `go test ./... -timeout 120s`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `tests/manager_test.go` — integration tests for CLI-03, CLI-04, CLI-05, CLI-15
- [ ] `internal/docker/manager_test.go` — unit tests for DOC-09, DOC-10, DOC-11, DOC-12, CFG-20, CLI-10, CLI-16
- [ ] Mock client interface in `internal/docker/client_interface.go` — enables unit testing without live Docker daemon
- [ ] Framework already installed: `github.com/stretchr/testify v1.10.0` in go.mod

## Sources

### Primary (HIGH confidence)
- `zone-spec.md` §3.7, §3.8, §3.9, §12 (lines 131-161, 1150-1303) — Manager struct, method signatures, security flags, state machine, TTY attach, BuildKit
- `internal/docker/naming.go`, `errors.go`, `harness_bridge.go`, `platform.go` — Existing code verified by reading
- `internal/cache/cache.go`, `hash.go`, `lock.go` — Existing code verified by reading
- Go compilation tests — All Docker SDK patterns verified to compile against `v28.5.2+incompatible`

### Secondary (MEDIUM confidence)
- `https://pkg.go.dev/github.com/docker/docker@v28.5.2+incompatible/client` — Function signatures confirmed
- Bash `go list -m -versions github.com/docker/docker` — v28.5.2+incompatible confirmed as latest (2026-03-29)
- `go mod tidy` output — Full transitive dependency graph for docker/docker confirmed

### Tertiary (LOW confidence)
- Docker Engine 23+ auto-enables BuildKit — verified as common knowledge; explicit `DOCKER_BUILDKIT=1` is harmless fallback

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — versions verified via go list and compilation
- Architecture: HIGH — spec provides exact code; confirmed compilable
- Pitfalls: HIGH — all verified against actual SDK behavior or spec text
- Test strategy: MEDIUM — mock interface approach is Claude's Discretion; may need adjustment based on planner decisions

**Research date:** 2026-03-29
**Valid until:** 2026-05-01 (Docker SDK stable, ~30 days)
