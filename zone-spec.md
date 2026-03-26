# Zone — Technical Specification v3.0

> **What this document is:** A complete spec for scaffolding `zone`, a Go CLI tool that generates and manages Docker workspaces for LLM coding harnesses (Claude Code, OpenCode, Gemini CLI, Aider, and custom harnesses). Hand this to Claude Code to scaffold the project.

---

## 1. Project Overview

**Zone** is a CLI tool (distributed as a Go binary) that:
1. Reads a per-repo `zone.toml` config file (with global defaults from `~/.config/zone/config.toml`)
2. Generates a Dockerfile + entrypoint tailored to the configured LLM harness
3. Builds, launches, and manages the Docker container
4. Caches build artifacts in `.zone/` for fast re-launches
5. Supports network whitelisting/blocklisting with glob patterns

**Distribution:** `go install`, Homebrew tap, or prebuilt binaries via GoReleaser.

**Core value proposition:** Run `zone launch` in any repo and get a sandboxed Docker workspace for your LLM coding agent, with zero manual Docker configuration.

---

## 2. Tech Stack

| Concern | Library |
|---------|---------|
| CLI framework / subcommands | [Cobra](https://github.com/spf13/cobra) |
| Interactive TUI (all interactive commands) | [BubbleTea](https://github.com/charmbracelet/bubbletea) |
| TUI styling | [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| TOML parsing | [BurntSushi/toml](https://github.com/BurntSushi/toml) (strict decoding with unknown key detection) |
| Docker SDK | [docker/docker/client](https://pkg.go.dev/github.com/docker/docker/client) |
| Dockerfile templating | `text/template` (stdlib) |
| Glob matching (network rules) | `filepath.Match` (stdlib) or [gobwas/glob](https://github.com/gobwas/glob) for `**` support |
| Hashing (cache) | `crypto/sha256` (stdlib) |
| Binary distribution | [GoReleaser](https://goreleaser.com/) |

---

## 3. CLI Surface

Built with **Cobra** for subcommand routing. **BubbleTea** for all interactive displays (init wizard, build progress, status, log tailing).

| Command | Description | Key Flags |
|---------|-------------|-----------|
| `zone init` | Scaffolds a `zone.toml` in the current directory. BubbleTea interactive harness selection with config preview. Adds `.zone/` to `.gitignore`. | `--harness <name>` to skip interactive prompt |
| `zone launch` | Builds (if needed) and runs the container. **Idempotent:** if a container is already running for this repo, attaches to it. | `--headless` runs detached. `--rebuild` forces fresh build. `--fresh` ignores all cache. `-- <args>` forwards to harness. |
| `zone join` | Attaches a new shell to an already-running zone container. For multi-terminal workflows. | None |
| `zone exec` | Runs a one-off command inside the running container without attaching a persistent shell. | `-- <cmd>` the command to run |
| `zone build` | Force-rebuilds the Docker image from current `zone.toml` without launching. | `--no-cache` passes `--no-cache` to Docker build |
| `zone stop` | Stops the running zone container for this repo. | None |
| `zone logs` | BubbleTea log viewer. Defaults to runtime (harness) output. | `--follow` tails live. `--build` shows last build log. `--tail N` last N lines. |
| `zone clean` | Removes `.zone/` cache and optionally Docker artifacts. | `--all` removes Docker image too. `--cache-only` just wipes `.zone/` |
| `zone status` | BubbleTea live view: container state, harness, image ID, uptime, network mode, port mappings. | `--json` for machine-readable output |
| `zone config` | Shows effective merged config (global + per-repo) with source annotations. | `--global` to show/edit global config only |

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Generic / unknown error |
| `2` | Config error (invalid `zone.toml`, missing required fields, schema mismatch) |
| `3` | Docker error (daemon not running, build failed, image not found) |
| `4` | Network error (firewall setup failed, host resolution failure) |
| `5` | Cache error (corrupted `.zone/`, lock contention) |

### Behavior: `zone launch` without `zone.toml`

**Error out.** Detect common existing configs and hint at migration:

```
Error: No zone.toml found in current directory.

  Detected: .devcontainer/devcontainer.json
  Hint: Run `zone init --from-devcontainer` to convert.

  Or run `zone init` for a fresh configuration.
```

Detection targets: `.devcontainer/devcontainer.json`, `Dockerfile`, `docker-compose.yml`. If none found, just say `Run zone init`.

### Behavior: `zone launch` idempotency

0. Acquire file lock on `.zone/.lock` (see Section 6). If lock fails, error with "another zone process is running."
1. Check `.zone/container_id`, if file exists AND that container is running, attach to it.
2. If container exists but is stopped, remove it and proceed to step 3.
3. Check `.zone/config.hash`, compare SHA256 of effective merged config against stored hash.
   - If match AND `.zone/image_id` references a valid image: run container from cached image.
   - If mismatch OR no cached image: regenerate Dockerfile, rebuild image, update cache.
4. Start container on its own isolated Docker network, write `container_id` to `.zone/`, attach TTY (unless `--headless`).
5. Release file lock.

---

## 4. Configuration: Two-Tier Config

### 4.1 Global Config: `~/.config/zone/config.toml`

Follows XDG Base Directory spec (`$XDG_CONFIG_HOME/zone/config.toml`, defaults to `~/.config/zone/`).

Created on first `zone init` if it doesn't exist, or manually via `zone config --global`.

```toml
# ~/.config/zone/config.toml — Global defaults
version = "1"

[zone]
base_image = "ubuntu:24.04"       # default base image for all repos

[auth]
mount_home_config = true          # auto-mount harness config dirs (read-only)
forward_env = []                  # env vars to always forward

[packages]
apt = ["git", "curl", "wget"]     # default apt packages for all zones
pip = []
npm = []

[network]
mode = "whitelist"                # "whitelist" | "blocklist" | "none"
default_allow = [                 # always allowed in whitelist mode
  "registry.npmjs.org",
  "pypi.org",
  "files.pythonhosted.org",
  "github.com",
  "*.github.com",
  "*.githubusercontent.com",
  "api.anthropic.com",
  "*.anthropic.com",
  "generativelanguage.googleapis.com",
  "*.npmjs.org",
]
default_deny = []                 # always blocked in blocklist mode

[harness]
# Global harness defaults (overridden per-repo)
```

### 4.2 Per-Repo Config: `./zone.toml`

```toml
# zone.toml — Per-repo configuration
version = "1"

[zone]
harness = "claude-code"           # required: "claude-code" | "opencode" | "gemini-cli" | "aider" | "custom"
base_image = "ubuntu:24.04"       # overrides global if set

[auth]
mount_home_config = true          # mounts harness config dir as read-only
forward_env = ["ANTHROPIC_API_KEY"]

[workspace]
mount_path = "/workspace"
extra_mounts = []                 # format: ["/host/path:/container/path:ro"]

[packages]
apt = []                          # merged: global defaults + per-repo (union)
pip = []
npm = []

[network]
mode = "whitelist"                # overrides global if set
allow = [                         # additional allows (added to global default_allow)
  "api.openai.com",
  "*.googleapis.com",
]
deny = []                         # additional denials

[harness]
# Harness-specific options
skip_permissions = false          # claude-code: --dangerously-skip-permissions (default OFF)
extra_args = []
```

### 4.3 Config Schema Versioning

Top-level `version = "1"` field is required. On parse:
- Missing version: warn and assume `"1"`
- Unrecognized version: error with "zone.toml version X is not supported by this version of zone. Run `zone self-update` or check https://github.com/jonathanung/zone"
- Future `zone migrate` command will auto-upgrade old configs

### 4.4 Config Merge Strategy

Per-repo values **override** global values for scalar fields. For list fields:
- `packages.apt/pip/npm`: **union** (global defaults + per-repo additions, deduplicated)
- `network.allow`: per-repo `allow` is **appended** to global `default_allow`
- `network.deny`: per-repo `deny` is **appended** to global `default_deny`
- `auth.forward_env`: **union**
- All other lists: per-repo **replaces** global

`zone config` output must show the merged result with source annotations:
```
[packages]
apt = ["git", "curl", "wget", "build-essential"]
#       ^^^^^^^^^^^^^^^^^^^^ (global)  ^^^^^^^^^^^^^^^^ (repo)
```

### 4.5 Config Validation

Use `BurntSushi/toml` strict decoding. Unknown keys produce an error:
```
Error: Unknown config keys in zone.toml: [zone.skip_perms]
Did you mean [harness].skip_permissions?
```

**Mount validation:** `extra_mounts` is scanned for dangerous paths. The following are blocked and produce an error:
- `/var/run/docker.sock` (Docker socket escape)
- `/var/run/podman/` (Podman socket)
- Anything under `/proc`, `/sys`, `/dev` (kernel interfaces)
- `~/.ssh` (SSH keys, unless explicitly overridden with `--allow-dangerous-mount`)

### 4.6 Network Rule Syntax

Rules are strings. Two formats:
- **Literal hostname:** `"api.anthropic.com"` — exact match
- **Glob pattern:** `"*.anthropic.com"` — standard glob, `*` matches any subdomain segment

Glob matching uses `filepath.Match` semantics for Phase 1. For more advanced globbing (e.g. `**` recursive), use `gobwas/glob` in Phase 2.

**Phase 1 constraint:** Only literal hostnames and simple globs (`*.domain.com`) are supported in network enforcement. Complex patterns are validated at config parse time and rejected with a clear error.

Evaluation order for whitelist mode:
1. Check deny list first (deny always wins)
2. Check allow list (global default_allow + per-repo allow)
3. Default: block

Evaluation order for blocklist mode:
1. Check deny list (global default_deny + per-repo deny)
2. Default: allow

Mode `"none"`: no network restrictions applied (container gets default Docker networking).

### 4.7 Network Implementation

**Phase 1 approach: Host-side Docker network isolation.** No `CAP_NET_ADMIN` required inside the container.

1. Each zone container gets its own isolated Docker bridge network: `zone-net-{container-hash}`
2. Network is created with `--internal` flag (no inter-container traffic)
3. For whitelist mode: Zone resolves allowed hostnames to IPs from the host, then configures iptables rules on the **host's Docker network bridge** (not inside the container)
4. DNS resolution inside the container uses Docker's embedded DNS (127.0.0.11), which only resolves allowed hosts
5. For "none" mode: container uses default Docker bridge network

The generated firewall rules are cached in `.zone/firewall.rules` for inspectability.

**Why not in-container iptables:** Granting `CAP_NET_ADMIN` to a container running an LLM agent with shell access is a significant privilege escalation risk. The agent could modify or disable the firewall rules. Host-side enforcement is the correct security boundary.

**Phase 2:** DNS proxy sidecar container for glob/regex matching at query time.

---

## 5. Harness-Specific Config Keys

### `[harness]` section keys by harness type

**claude-code:**
- `skip_permissions` (bool, **default: false**), adds `--dangerously-skip-permissions`
- `extra_args` (list[string]), forwarded to `claude` command
- `node_version` (string, default: "20"), Node.js major version

**⚠️ Security note on `skip_permissions`:** When `zone init` detects `harness = "claude-code"`, the wizard shows:
```
⚠️  skip_permissions is currently OFF (recommended).
Enabling it allows the AI to modify files without approval.
Only enable in sandboxed/disposable environments.

[s] Enable skip_permissions  [Enter] Keep OFF
```

**opencode:**
- `extra_args` (list[string])

**gemini-cli:**
- `extra_args` (list[string])
- `node_version` (string, default: "20")

**aider:**
- `extra_args` (list[string])
- `python_version` (string, default: "3.12")

**custom:**
- `install_commands` (list[string]), shell commands to install the harness
- `entrypoint_command` (string), the command to exec into
- `config_dir` (string, optional), host directory to mount for auth
- `required_env` (list[string]), env vars that must be set

---

## 6. `.zone/` Cache Directory

```
.zone/
├── .lock              # flock-based file lock for concurrent access protection
├── config.hash        # SHA256 hex digest of effective merged config
├── Dockerfile         # last generated Dockerfile (human-inspectable)
├── entrypoint.sh      # last generated entrypoint
├── firewall.rules     # last generated network firewall rules (host-side)
├── image_id           # Docker image ID (sha256:...)
├── container_id       # Docker container ID (for reattach)
├── network_id         # Docker network ID (for cleanup)
└── logs/
    └── last_build.log # stdout/stderr from last docker build
```

**Concurrent access protection:** Before any cache read/write, acquire an exclusive flock on `.zone/.lock`. If lock cannot be acquired (non-blocking attempt), error with exit code 5: "Another zone process is operating on this repo. Wait or run `zone clean` to reset."

**`zone init` must add `.zone/` to `.gitignore`** if a `.gitignore` exists and `.zone/` is not already in it.

---

## 7. Project Structure

```
zone/
├── go.mod
├── go.sum
├── main.go                       # entry point, version vars, Cobra root
├── Makefile                      # build, test, lint targets
├── .goreleaser.yml               # binary distribution config
├── README.md
├── cmd/
│   ├── root.go                   # Cobra root command + global flags + signal handling
│   ├── init.go                   # zone init (BubbleTea wizard)
│   ├── launch.go                 # zone launch (idempotent lifecycle)
│   ├── join.go                   # zone join (multi-terminal attach)
│   ├── exec.go                   # zone exec (one-off commands)
│   ├── build.go                  # zone build (standalone build)
│   ├── stop.go                   # zone stop
│   ├── logs.go                   # zone logs
│   ├── clean.go                  # zone clean
│   ├── status.go                 # zone status
│   └── config.go                 # zone config (show merged config)
├── internal/
│   ├── config/
│   │   ├── types.go              # Config struct definitions (shared types)
│   │   ├── config.go             # TOML parsing, strict decoding
│   │   ├── global.go             # Global config read/write (~/.config/zone/)
│   │   ├── merge.go              # Two-tier merge strategy
│   │   └── validate.go           # Config validation (dangerous mounts, unknown keys)
│   ├── cache/
│   │   ├── cache.go              # .zone/ directory management
│   │   ├── hash.go               # SHA256 config hashing
│   │   └── lock.go               # flock-based concurrent access protection
│   ├── docker/
│   │   ├── manager.go            # Build, run, attach, stop, status
│   │   ├── dockerfile.go         # Dockerfile generation (text/template)
│   │   ├── entrypoint.go         # Entrypoint script generation
│   │   ├── naming.go             # Deterministic container + network naming
│   │   ├── network.go            # Docker network create/destroy + host-side firewall
│   │   └── errors.go             # Sentinel errors (ErrImageNotFound, ErrContainerRunning, etc.)
│   ├── network/
│   │   ├── firewall.go           # Host-side iptables rule generation
│   │   ├── rules.go              # Rule parsing (literal + glob)
│   │   └── matcher.go            # Hostname glob matching engine (precompiled)
│   ├── harness/
│   │   ├── harness.go            # Harness interface + registry
│   │   ├── claude_code.go        # fully implemented
│   │   ├── opencode.go           # stub
│   │   ├── gemini_cli.go         # stub
│   │   ├── aider.go              # stub
│   │   └── custom.go             # user-defined harness from config
│   └── tui/
│       ├── init_wizard.go        # BubbleTea init wizard model
│       ├── build_progress.go     # BubbleTea build progress model
│       ├── status_view.go        # BubbleTea live status model
│       └── log_viewer.go         # BubbleTea log viewer model
├── pkg/
│   └── templates/
│       ├── templates.go          # //go:embed declarations
│       ├── Dockerfile.tmpl       # Go text/template Dockerfile
│       └── entrypoint.sh.tmpl    # Go text/template entrypoint
└── tests/
    ├── config_merge_test.go      # Config merge logic (write these FIRST)
    ├── validate_test.go          # Dangerous mount detection
    ├── naming_test.go            # Container name generation
    └── matcher_test.go           # Network rule matching
```

### Import graph (enforced):
```
cmd/* → internal/*       ✓
cmd/* → pkg/templates    ✓
internal/docker → internal/config   ✓
internal/docker → internal/cache    ✓
internal/docker → internal/network  ✓
internal/docker → pkg/templates     ✓
internal/* → cmd/*       ✗ (forbidden, breaks encapsulation)
```

---

## 8. Error Handling Convention

All exported functions return `error`. Errors are wrapped with operation context:

```go
func (m *Manager) Build(ctx context.Context, noCache bool) (string, error) {
    if err := m.generateDockerfile(); err != nil {
        return "", fmt.Errorf("dockerfile generation: %w", err)
    }
    // ...
}
```

Sentinel errors in `internal/docker/errors.go`:

```go
var (
    ErrNoConfig         = errors.New("no zone.toml found")
    ErrImageNotFound    = errors.New("cached image not found")
    ErrContainerRunning = errors.New("container already running")
    ErrLockContention   = errors.New("another zone process is operating on this repo")
    ErrDangerousMount   = errors.New("mount path is blocked for security")
    ErrDockerNotRunning = errors.New("docker daemon is not running")
)
```

Cmd layer maps sentinel errors to exit codes. Internal packages never call `os.Exit()`.

---

## 9. Context and Signal Handling

All Docker SDK calls and long-running operations take `context.Context` as first parameter. Cobra commands create context with signal handling for graceful Ctrl+C:

```go
func runLaunch(cmd *cobra.Command, args []string) error {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()

    mgr, err := docker.NewManager(cfg, cache)
    if err != nil { return err }
    defer mgr.Close()

    return mgr.Launch(ctx, launchOpts)
}
```

Docker build streaming reads from `io.ReadCloser` in a goroutine that respects `ctx.Done()`:

```go
func (m *Manager) Build(ctx context.Context, noCache bool) (<-chan BuildProgress, error) {
    resp, err := m.client.ImageBuild(ctx, tarball, buildOpts)
    if err != nil { return nil, err }

    ch := make(chan BuildProgress)
    go func() {
        defer close(ch)
        defer resp.Body.Close()
        decoder := json.NewDecoder(resp.Body)
        for {
            var msg BuildProgress
            if err := decoder.Decode(&msg); err != nil { return }
            select {
            case ch <- msg:
            case <-ctx.Done():
                return
            }
        }
    }()
    return ch, nil
}
```

---

## 10. Harness Plugin Architecture

### Interface

```go
package harness

type Harness interface {
    Name() string
    InstallCommands() []string
    EntrypointCommand() string
    RequiredEnvVars() []string
    HomeConfigDir() string          // e.g. "~/.claude", empty if N/A
    DefaultAptPackages() []string
    DefaultNpmPackages() []string
    DefaultPipPackages() []string
    NeedsNode() bool
    NeedsPython() bool
}
```

### Registry

```go
var registry = map[string]func(config map[string]interface{}) Harness{
    "claude-code": func(c map[string]interface{}) Harness { return &ClaudeCode{config: c} },
    "opencode":    func(c map[string]interface{}) Harness { return &OpenCode{config: c} },
    "gemini-cli":  func(c map[string]interface{}) Harness { return &GeminiCLI{config: c} },
    "aider":       func(c map[string]interface{}) Harness { return &Aider{config: c} },
    "custom":      func(c map[string]interface{}) Harness { return &Custom{config: c} },
}

func Get(name string, config map[string]interface{}) (Harness, error) {
    factory, ok := registry[name]
    if !ok {
        return nil, fmt.Errorf("unknown harness %q, available: %v", name, availableNames())
    }
    return factory(config), nil
}
```

All implementations use pointer receivers. Stubs return descriptive errors (not panics):

```go
func (o *OpenCode) InstallCommands() []string {
    // This will be caught at build time, not runtime crash
    return nil // stub
}
func (o *OpenCode) EntrypointCommand() string {
    return "" // stub, validated before use
}
```

Validation layer checks for empty `InstallCommands()` / `EntrypointCommand()` and returns:
```
Error: The "opencode" harness is not yet fully implemented.
Use harness = "custom" with install_commands and entrypoint_command to configure it manually.
```

---

## 11. Dockerfile Generation

Templates live in `pkg/templates/` and are embedded via `//go:embed`:

```go
// pkg/templates/templates.go
package templates

import _ "embed"

//go:embed Dockerfile.tmpl
var DockerfileTmpl string

//go:embed entrypoint.sh.tmpl
var EntrypointTmpl string
```

Imported by `internal/docker/dockerfile.go` as `import "zone/pkg/templates"`.

### Template: `Dockerfile.tmpl`

```dockerfile
FROM {{ .BaseImage }}

ARG HOST_UID={{ .HostUID }}
ARG DEBIAN_FRONTEND=noninteractive

# Base system packages
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    git \
    sudo \
{{- range .AptPackages }}
    {{ . }} \
{{- end }}
    && rm -rf /var/lib/apt/lists/*

{{- if .NeedsNode }}

# Node.js {{ .NodeVersion }}
RUN curl -fsSL https://deb.nodesource.com/setup_{{ .NodeVersion }}.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*
{{- end }}

{{- if .NpmPackages }}

# Global npm packages
RUN npm install -g {{ join .NpmPackages " " }}
{{- end }}

{{- if .PipPackages }}

# Pip packages
RUN pip install --break-system-packages {{ join .PipPackages " " }}
{{- end }}

{{- range .HarnessInstallCommands }}
RUN {{ . }}
{{- end }}

# Create non-root user
RUN useradd -m -s /bin/bash -u ${HOST_UID} zone \
    && echo "zone ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

{{- if .MacOSUsername }}

# macOS plugin symlink compatibility
RUN mkdir -p /Users/{{ .MacOSUsername }} \
    && ln -sf /home/zone /Users/{{ .MacOSUsername }}
{{- end }}

USER zone
WORKDIR {{ .MountPath }}

COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
```

### Template: `entrypoint.sh.tmpl`

```bash
#!/bin/bash
set -e

# Mark workspace as git-safe
git config --global --add safe.directory {{ .MountPath }}

# Execute harness command, forwarding any extra args
exec {{ .EntrypointCommand }} "$@"
```

Note: No firewall script in the container. Network filtering is host-side (see Section 4.7).

---

## 12. Docker Manager

### Key responsibilities of `internal/docker/manager.go`

Uses the **Go Docker SDK** (`github.com/docker/docker/client`). Client is initialized once in constructor:

```go
type Manager struct {
    client *client.Client
    config *config.MergedConfig
    cache  *cache.Cache
}

func NewManager(cfg *config.MergedConfig, c *cache.Cache) (*Manager, error) {
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        return nil, fmt.Errorf("docker daemon: %w", ErrDockerNotRunning)
    }
    return &Manager{client: cli, config: cfg, cache: c}, nil
}

func (m *Manager) Close() error { return m.client.Close() }

func (m *Manager) GenerateDockerfile(ctx context.Context) (string, error)
func (m *Manager) Build(ctx context.Context, noCache bool) (<-chan BuildProgress, error)
func (m *Manager) Launch(ctx context.Context, opts LaunchOpts) error
func (m *Manager) Join(ctx context.Context) error
func (m *Manager) Exec(ctx context.Context, cmd []string) error
func (m *Manager) Stop(ctx context.Context) error
func (m *Manager) Logs(ctx context.Context, opts LogOpts) error
func (m *Manager) Status(ctx context.Context) (*ContainerStatus, error)
```

### Container + Network Naming Convention

Deterministic from repo absolute path. 16-char hash, sanitized name:

```go
func ContainerName(repoPath string) string {
    absPath, _ := filepath.Abs(repoPath)
    hash := sha256.Sum256([]byte(absPath))
    shortHash := hex.EncodeToString(hash[:])[:16]
    repoName := filepath.Base(absPath)
    repoName = regexp.MustCompile(`[^a-zA-Z0-9_.-]`).ReplaceAllString(repoName, "-")
    return fmt.Sprintf("zone-%s-%s", repoName, shortHash)
}

func NetworkName(repoPath string) string {
    return ContainerName(repoPath) + "-net"
}
```

### Interactive TTY Attach

For interactive `zone launch`, `zone join`, and `zone exec`, use `os/exec` with `docker exec -it` rather than the Go Docker SDK's `ContainerExecAttach`. The SDK's hijacked connection API is notoriously difficult for proper raw terminal I/O (resize handling, signal forwarding). This is the same pragmatic choice Docker CLI itself makes.

```go
func (m *Manager) attachInteractive(containerID string, cmd []string) error {
    args := append([]string{"exec", "-it", containerID}, cmd...)
    c := exec.Command("docker", args...)
    c.Stdin = os.Stdin
    c.Stdout = os.Stdout
    c.Stderr = os.Stderr
    return c.Run()
}
```

Use the SDK for everything non-interactive: build, create, start, stop, inspect, remove, network create/destroy.

---

## 13. BubbleTea TUI Components

BubbleTea is used consistently across all interactive commands. This gives Zone a cohesive, polished feel. Every BubbleTea model lives in `internal/tui/` and is imported by the corresponding `cmd/` file.

### `zone init` — Init Wizard (`internal/tui/init_wizard.go`)

Interactive harness selector with config preview:

```
? Select your LLM harness:

  > Claude Code    . Anthropic's agentic coding tool
    OpenCode       . Open-source coding agent (stub)
    Gemini CLI     . Google's Gemini in the terminal (stub)
    Aider          . AI pair programming in the terminal (stub)
    Custom         . Define your own harness

  ↑/↓ navigate  •  enter select  •  q quit
```

After selection, show config preview before writing:

```
Claude Code Harness
├─ Base: ubuntu:24.04
├─ Packages: git, curl, nodejs (v20)
├─ Network: Whitelist mode (8 rules)
├─ Auth: ~/.claude mounted (read-only)
└─ skip_permissions: OFF

⚠️  skip_permissions is OFF (recommended).
    Enabling it allows the AI to modify files without approval.

[s] Enable skip_permissions  [c] Customize  [Enter] Confirm  [q] Cancel
```

### `zone launch` — Build Progress (`internal/tui/build_progress.go`)

When a rebuild is triggered, stream Docker build output through a BubbleTea viewport with spinner:

```
⠋ Building zone image...

Step 1/8 : FROM ubuntu:24.04
 ---> a1b2c3d4e5f6
Step 2/8 : RUN apt-get update...
 ---> Running in f6e5d4c3b2a1

  ↓ scroll  •  ctrl+c cancel
```

Subscribes to the `<-chan BuildProgress` from `Manager.Build()` via a `tea.Cmd` that reads from the channel. On build complete, transitions to TTY attach (or prints container ID if `--headless`).

### `zone status` — Live Status View (`internal/tui/status_view.go`)

```
╭─ Zone Status ─────────────────────────╮
│ Repo:      my-project                 │
│ Harness:   claude-code                │
│ Container: zone-my-project-a1b2c3d4   │
│ Status:    ● Running (2h 14m)         │
│ Image:     sha256:abcdef...           │
│ Network:   whitelist (12 rules)       │
│ Ports:     3000/tcp → 0.0.0.0:54321  │
│ Mounts:    /workspace (rw)            │
│            ~/.claude (ro)             │
╰───────────────────────────────────────╯
  q quit  •  r restart  •  s stop
```

Polls container status every 2 seconds via Docker SDK. `--json` flag bypasses BubbleTea entirely and prints raw JSON to stdout (for scripting).

### `zone logs` — Log Viewer (`internal/tui/log_viewer.go`)

BubbleTea viewport with auto-scroll and search:

```
[14:32:01] claude: Reading project files...
[14:32:03] claude: Found 42 source files
[14:32:05] claude: Analyzing codebase structure...
█

  ↑/↓ scroll  •  / search  •  f follow  •  q quit
```

`--follow` starts in follow mode (auto-scroll to bottom). Without `--follow`, starts paused at the end. `--build` loads from `.zone/logs/last_build.log` instead of live container logs.

### Integration pattern

All BubbleTea models follow the same Cobra integration:

```go
func runStatus(cmd *cobra.Command, args []string) error {
    if jsonFlag {
        // Bypass TUI, print raw JSON
        return printStatusJSON(ctx, mgr)
    }

    model := tui.NewStatusView(mgr)
    p := tea.NewProgram(model, tea.WithAltScreen())
    finalModel, err := p.Run()
    if err != nil {
        return fmt.Errorf("tui: %w", err)
    }
    result := finalModel.(tui.StatusView)
    return result.Err // nil if clean exit
}
```

---

## 14. Versioning via ldflags

```go
// main.go
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)

func main() {
    rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
    rootCmd.Execute()
}
```

`.goreleaser.yml`:
```yaml
builds:
  - ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}
```

---

## 15. Implementation Priority

### Phase 1 (MVP)
1. Project scaffolding: go.mod, Cobra commands, project structure, version injection
2. Config parsing: zone.toml + global config + merge logic + strict validation + dangerous mount blocking
3. `zone init` — BubbleTea wizard with config preview, scaffold zone.toml, add .zone/ to .gitignore, detect existing configs
4. `zone launch` — full idempotent lifecycle with flock (generate, build, run, reattach), BubbleTea build progress
5. `zone exec` — one-off command execution
6. `zone stop` — stop container + remove network
7. `zone status` — BubbleTea live status view (+ `--json` bypass)
8. `zone logs` — BubbleTea log viewer with follow/search (+ `--build` for build logs)
9. Claude Code harness — fully implemented, skip_permissions defaults OFF
10. `.zone/` caching — config hash, Dockerfile, image ID, container ID, flock
11. Network whitelist/blocklist — literal hostnames + simple globs, host-side enforcement, per-container isolated network
12. All BubbleTea TUI models: init wizard, build progress, status view, log viewer

### Phase 2
13. `zone join` — multi-terminal attach
14. `zone build` — standalone build command
15. `zone clean` — cache/artifact cleanup
16. `zone config` — show merged config with source annotations
17. Custom harness support
18. GoReleaser setup + Homebrew formula
19. `zone init --from-devcontainer` migration

### Phase 3
20. OpenCode harness
21. Gemini CLI harness
22. Aider harness
23. Glob network rules via DNS proxy sidecar
24. `zone migrate` for config schema upgrades
25. Homebrew tap

---

## 16. Key Design Principles

1. **Idempotent launch.** `zone launch` is the only command most users need. It handles everything.
2. **Inspectable cache.** `.zone/Dockerfile` is a real Dockerfile the user can read and debug. No magic.
3. **Harness as plugin.** Adding a new harness = adding one Go file. No core changes.
4. **Two-tier config.** Global defaults + per-repo overrides. Predictable merge semantics.
5. **No global mutable state.** Cache is per-repo in `.zone/`. Global config is read-only defaults.
6. **Docker SDK + exec fallback.** SDK for management, `docker exec -it` for interactive TTY.
7. **Fail loud.** Missing config, missing Docker, missing env vars, all produce clear error messages with exit codes.
8. **Secure by default.** Whitelist network mode. `skip_permissions` off. Host-side firewall. No `CAP_NET_ADMIN`. Dangerous mount blocking. Per-container network isolation.
9. **Config versioned.** Schema versioning from day one prevents future migration pain.

---

## 17. Notes for Claude Code

- Use `//go:embed` in `pkg/templates/templates.go`, not in `internal/docker/`. Embed paths are relative to the `.go` file.
- BubbleTea `tea.NewProgram().Run()` is blocking. Wrap in Cobra `RunE`, extract result from final model, check for cancellation. Use `tea.WithAltScreen()` for status and logs views. All TUI models follow the same integration pattern (see Section 13).
- Build progress TUI subscribes to `<-chan BuildProgress` via a `tea.Cmd` that reads from the channel. Status view polls every 2 seconds. Log viewer wraps Docker SDK's `ContainerLogs` stream.
- Docker SDK client is initialized once in `NewManager()`, reused across all operations, closed via `defer mgr.Close()`.
- All Docker SDK methods take `context.Context` as first param. Create context with `signal.NotifyContext` in Cobra commands.
- For interactive TTY attach, use `os/exec` with `docker exec -it`, not the Go SDK attach API.
- Config merge logic is the most subtle part. Write `config_merge_test.go` FIRST before implementing.
- File lock via `syscall.Flock()` on `.zone/.lock`. Non-blocking attempt, exit code 5 on contention.
- Container naming: 16-char hash, sanitize repo name to `[a-zA-Z0-9_.-]`.
- Network: create isolated Docker network per container (`--internal`). Apply firewall from host side. NO `CAP_NET_ADMIN` in container.
- Stubs should return empty values caught by validation, NOT panic.
- Test `zone launch` idempotency early, the container reattach + flock logic is the most critical path.
- Use `BurntSushi/toml` with strict decoding (`.Undecoded()` check) to catch typos in config keys.