# Zone -- Technical Specification v4.0

> **What this document is:** A complete spec for scaffolding `zone`, a Go CLI tool that generates and manages Docker workspaces for LLM coding harnesses (Claude Code, OpenCode, Gemini CLI, Aider, Codex CLI, and custom harnesses). Hand this to Claude Code to scaffold the project.

---

## 1. Project Overview

**Zone** is a CLI tool (distributed as a Go binary) that:
1. Reads a per-repo `zone.toml` config file (with global defaults from `~/.config/zone/config.toml`)
2. Generates a Dockerfile + entrypoint tailored to the configured LLM harness
3. Builds, launches, and manages the Docker container
4. Caches build artifacts in `.zone/` for fast re-launches
5. Supports network whitelisting/blocklisting for sandboxed execution

**Distribution:** `go install`, Homebrew tap, or prebuilt binaries via GoReleaser.

**Core value proposition:** Run `zone launch` in any repo and get a sandboxed Docker workspace for your LLM coding agent, with zero manual Docker configuration.

**Privacy:** Zone does not collect telemetry or analytics data. All operations are local.

### Why Zone over a shell script?

| Capability | Shell script (e.g. claudocker) | Zone |
|------------|-------------------------------|------|
| Zero-config launch | Manual Dockerfile | Auto-generated from harness selection |
| Multi-harness support | Hardcoded tools | Any LLM agent via plugin interface |
| Network sandboxing | None | Whitelist/blocklist per container |
| Config reuse across repos | Copy script | Global defaults + per-repo overrides |
| Idempotent reattach | Manual tmux attach | Automatic container lifecycle |
| Cross-platform | Linux/macOS (bash) | Go binary, Linux + macOS |
| Port forwarding, resource limits | Manual docker flags | Declarative config |

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

### 3.1 Global Flags (root command)

| Flag | Description |
|------|-------------|
| `--verbose` / `-v` | Increases output verbosity. Shows Docker SDK calls, config merge decisions, cache hit/miss. |
| `--debug` | Maximum verbosity. Includes raw Docker API responses, template rendering output, full error stack traces. |
| `--quiet` / `-q` | Suppress all non-essential output. Only errors go to stderr. |
| `--plain` | Disable TUI, use plain text output. Auto-enabled when stdout is not a TTY (see Section 3.5). |

These are handled in `cmd/root.go` and set a global log level used by all internal packages.

### 3.2 Commands

| Command | Aliases | Description | Key Flags |
|---------|---------|-------------|-----------|
| `zone init` | | Scaffolds a `zone.toml` in the current directory. BubbleTea interactive harness selection with config preview. Adds `.zone/` to `.gitignore`. | `--harness <name>` to skip interactive prompt. `--set key=value` to override config values (dotted path). |
| `zone launch` | `zone up` | Builds (if needed) and runs the container. **Idempotent:** if a container is already running for this repo, attaches to it. If no `zone.toml` exists, triggers the init wizard inline (see Section 3.6). | `--prompt` / `-p` passes a prompt to the harness agent. `--headless` runs detached. `--rebuild` forces fresh build. `--no-cache` passes `--no-cache` to Docker build. `--port` / `-P` ad-hoc port mapping (e.g. `-P 3000:3000`). `-- <args>` forwards to harness. |
| `zone join` | | Attaches a new interactive shell (`/bin/bash`) to an already-running zone container. Does NOT re-run the harness entrypoint. For multi-terminal workflows alongside the running agent. | `--root` shell in as root. |
| `zone exec` | | Runs a one-off command inside the running container without attaching a persistent shell. Allocates TTY when stdin is a terminal. | `-- <cmd>` the command to run. `-T` / `--no-tty` force disable TTY (for piping output). |
| `zone shell` | | Opens an interactive shell in the container even if no harness is running. If container is not running, starts it with a temporary entrypoint. For debugging. | `--root` shell in as root. |
| `zone build` | | Force-rebuilds the Docker image from current `zone.toml` without launching. | `--no-cache` passes `--no-cache` to Docker build. |
| `zone stop` | `zone down` | Stops and removes the running zone container + network for this repo. Clears `container_id` and `network_id` from cache. | `--timeout N` seconds before SIGKILL (default: 10). `--force` / `-f` sends SIGKILL immediately. |
| `zone restart` | | Stops the running container and relaunches. Equivalent to `zone stop && zone launch`. | `--rebuild` force image rebuild. `-- <args>` forwarded to harness. |
| `zone ls` | `zone list` | Lists all zone containers across all repos (running + stopped). | `--running` show only running. `--json` machine-readable output. `--quiet` / `-q` print only container names. |
| `zone logs` | `zone log` | BubbleTea log viewer. Defaults to runtime (harness) output. | `--follow` / `-f` tails live. `--build` shows last build log. `--tail N` last N lines. |
| `zone clean` | | Removes `.zone/` cache and optionally Docker artifacts. Refuses to act if container is running (use `zone stop` first or `zone destroy`). | `--all` removes Docker image too. `--cache-only` just wipes `.zone/`. |
| `zone destroy` | | Full teardown: stops container, removes image, network, and `.zone/` cache. Does NOT remove `zone.toml`. | `--yes` / `-y` skip confirmation. `--include-config` also removes `zone.toml`. |
| `zone status` | `zone st` | BubbleTea live view: container state, harness, image ID, uptime, network mode, port mappings, resource limits. | `--json` for machine-readable output. |
| `zone config` | | Shows effective merged config (global + per-repo) with source annotations. | `--global` to show/edit global config only. `--edit` open in `$EDITOR`. `--json` machine-readable. `--schema` dump full schema with descriptions. |
| `zone validate` | | Validates `zone.toml` without launching. Checks syntax, unknown keys, dangerous mounts, harness-specific config, env var availability. | Exit code 0 for valid, 2 for invalid. |

### 3.3 Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Generic / unknown error |
| `2` | Config error (invalid `zone.toml`, missing required fields, schema mismatch) |
| `3` | Docker error (daemon not running, build failed, image not found) |
| `4` | Network error (firewall setup failed, host resolution failure) |
| `5` | Cache error (corrupted `.zone/`, lock contention) |
| `6` | No running container (e.g., `zone exec` / `zone join` when nothing is launched) |

### 3.4 Argument Forwarding

Arguments after `--` are forwarded verbatim to the harness entrypoint. Zone flags must appear before `--`. Example:

```
zone launch --headless -- -p "fix the bug"
   ^zone flag^           ^harness args^
```

This convention matches `kubectl`, `npm run`, and `cargo run`.

The `--prompt` / `-p` flag is a convenience that translates to the harness-appropriate flag automatically (see Section 5).

### 3.5 TTY Detection

On startup, Zone checks `os.Stdin` with `term.IsTerminal()`:
- **TTY detected:** BubbleTea TUI enabled for interactive commands.
- **Non-TTY (CI, pipe, cron):** All TUI is disabled automatically. Commands that would use BubbleTea fall back to plain text output. `zone init` without `--harness` errors with "Interactive mode requires a terminal. Use `--harness <name>` for non-interactive init."
- **Piped stdout:** Commands like `zone logs` output plain text when piped (e.g., `zone logs -f | grep error` works correctly).

Override with `--plain` (force disable even in TTY).

### 3.6 Behavior: `zone launch` without `zone.toml`

If no `zone.toml` is found:
1. If `--harness <name>` is provided: generate a default `zone.toml` for that harness (with comments showing all available options) and proceed to build/launch. **This is the zero-config quickstart path:** `zone launch --harness claude-code`.
2. If no `--harness` and stdout is a TTY: launch the init wizard inline (same BubbleTea flow as `zone init`), write the config, then proceed to build/launch.
3. If no `--harness` and non-TTY: error with "No zone.toml found. Use `zone launch --harness <name>` for non-interactive setup."
4. Detected existing configs (`.devcontainer/devcontainer.json`, `Dockerfile`, `docker-compose.yml`) are shown as hints during the wizard, not as blocking errors.

`zone init` remains available for users who want to customize config without launching.

### 3.7 Behavior: `zone launch` idempotency

0. Acquire file lock on `.zone/.lock` (see Section 6). If lock fails, error with "another zone process is running."
1. Check `.zone/container_id`. If file exists, inspect container state:
   - **Running:** Check config hash against `.zone/config.hash`. If changed, warn: "Config has changed since this container was started. Run `zone restart --rebuild` to apply changes." Then attach to running container.
   - **Paused:** Unpause, then attach.
   - **Exited/Dead:** Inspect exit code and `OOMKilled` flag. If OOM, warn user. Remove container + network, proceed to step 2.
   - **Created/Restarting:** Wait briefly, then stop, remove, proceed to step 2.
   - **Container ID references non-existent container** (e.g., pruned, host rebooted): Clean up stale cache files (`container_id`, `network_id`), attempt to remove orphaned network, proceed to step 2.
2. Check `.zone/config.hash`, compare full cache hash (see Section 6.2) against stored hash.
   - If match AND `.zone/image_id` references a valid image (verified via `ImageInspect`): run container from cached image.
   - If mismatch OR no cached image OR image was pruned: regenerate Dockerfile, rebuild image, update cache.
3. Start container on its own Docker network, write `container_id` to `.zone/`, **release file lock**, then attach TTY (unless `--headless`).
4. On `--headless`: release lock, print container ID, exit.

**Important:** The file lock is released *before* TTY attachment. The lock protects the build/create/start sequence, not the interactive session. This allows `zone join` to work while `zone launch` is attached.

### 3.8 Behavior: Exiting an attached `zone launch` session

- **Ctrl+C:** Sends SIGINT to the harness process inside the container. The container continues running. The user is detached. (Container stays alive for `zone join` / `zone logs`.)
- **Harness process exits** (e.g., user types `/exit` in Claude Code): The container stops. `zone launch` returns exit code 0.
- **`zone stop`** from another terminal explicitly stops the container.

### 3.9 Behavior: `zone stop` cleanup

`zone stop` performs:
1. Stop the container (SIGTERM, wait `--timeout` seconds, then SIGKILL)
2. Remove the container
3. Remove the associated Docker network
4. Clear `container_id` and `network_id` from `.zone/` cache
5. Retain `image_id`, `config.hash`, and `Dockerfile` for fast re-launch

### 3.10 Common Workflow Patterns

```
# Interactive session (most common)
zone launch                              # build + attach, Ctrl+C to detach

# Zero-config quickstart (first time)
zone launch --harness claude-code        # one command, zero files, immediate sandbox

# Fire and forget
zone launch --headless -p "fix the tests"   # background agent with task
zone logs -f                                # check progress

# Multi-terminal
zone launch                              # terminal 1: agent running
zone join                                # terminal 2: shell alongside agent

# Quick one-off
zone exec -- npm test                    # run command in container

# Debug container
zone shell                              # interactive shell, no harness
zone shell --root                       # root shell for debugging
```

### 3.11 Help Text Convention

Every command includes an `Example` field in its Cobra definition showing 2-4 common usage patterns:

```
zone launch --help

Usage:
  zone launch [flags] [-- harness-args...]

Aliases:
  up

Examples:
  # Launch interactively (default)
  zone launch

  # Zero-config quickstart
  zone launch --harness claude-code

  # Launch and pass a prompt to the harness
  zone launch -p "refactor the auth module"

  # Fire and forget (background with task)
  zone launch --headless -p "fix the failing tests"

Flags:
  ...
```

### 3.12 Actionable Error Messages

All error messages include remediation hints:

```
Error: Docker daemon is not running.

  macOS:  Open Docker Desktop, or run `open -a Docker`
  Linux:  Run `sudo systemctl start docker`

Zone requires Docker to create sandboxed workspaces.
```

```
Error: No running zone container for this repo.

  Run `zone launch` to start one, then `zone join` in another terminal.
```

```
Error: Required environment variable ANTHROPIC_API_KEY is not set.

  The claude-code harness needs this variable.
  Set it in your shell and re-run zone launch.
```

### 3.13 Scriptability Convention

Any command that produces structured or tabular output supports `--json` for machine-readable output. When `--json` is active, BubbleTea TUI is bypassed, output goes to stdout as JSON, and stderr is used for progress/error messages only.

Commands with `--json` support: `zone status`, `zone ls`, `zone config`, `zone logs`.

---

## 4. Configuration: Two-Tier Config

### 4.1 Global Config: `~/.config/zone/config.toml`

Follows XDG Base Directory spec (`$XDG_CONFIG_HOME/zone/config.toml`, defaults to `~/.config/zone/`).

Created on first `zone init` if it doesn't exist, or manually via `zone config --global --edit`.

```toml
# ~/.config/zone/config.toml -- Global defaults
version = 1

[zone]
base_image = "ubuntu:24.04"       # default base image for all repos
shell = "bash"                    # "bash" | "zsh"

[auth]
mount_home_config = true          # auto-mount harness config dirs (copy-on-start, see Section 4.9)
forward_env = []                  # env vars to always forward (supports globs: "AWS_*")
forward_ssh_agent = false         # forward SSH_AUTH_SOCK for git operations

[packages]
apt = ["git", "curl", "wget"]     # default apt packages for all zones
pip = []
npm = []

[resources]
memory = ""                       # Docker memory limit (e.g., "8g"). Empty = no limit.
cpus = ""                         # Docker CPU limit (e.g., "4"). Empty = no limit.
pids_limit = 512                  # Max processes (fork bomb protection)
gpus = ""                         # GPU passthrough: "all", count ("2"), or "device=0,1". Empty = no GPU. Requires NVIDIA Container Toolkit.

[network]
mode = "none"                     # "whitelist" | "blocklist" | "none"
default_allow = [                 # always allowed in whitelist mode
  "registry.npmjs.org",
  "*.npmjs.org",
  "pypi.org",
  "files.pythonhosted.org",
  "github.com",
  "*.github.com",
  "api.github.com",
  "*.githubusercontent.com",
  "objects.githubusercontent.com",
  "api.anthropic.com",
  "*.anthropic.com",
  "api.openai.com",
  "generativelanguage.googleapis.com",
  "*.googleapis.com",
  "go.dev",
  "proxy.golang.org",
  "sum.golang.org",
  "crates.io",
  "static.crates.io",
  "rubygems.org",
]
default_deny = []                 # always blocked in blocklist mode

[harness]
# Global harness defaults (overridden per-repo)
```

### 4.2 Per-Repo Config: `./zone.toml`

**Minimal valid config (two lines):**

```toml
version = 1
harness = "claude-code"
```

The `harness` key is accepted at both the top level (sugar) and inside `[zone]`. Top-level `harness` is equivalent to `[zone] harness`.

**Full config with all options:**

```toml
# zone.toml -- Per-repo configuration
version = 1

[zone]
harness = "claude-code"           # required: "claude-code" | "opencode" | "gemini-cli" | "aider" | "codex-cli" | "custom"
base_image = "ubuntu:24.04"       # overrides global if set
shell = "bash"                    # "bash" | "zsh"

[auth]
mount_home_config = true          # copy harness config dir into container at start
forward_env = ["ANTHROPIC_API_KEY"]  # supports globs: "AWS_*"
forward_ssh_agent = false         # forward SSH_AUTH_SOCK (safer than mounting ~/.ssh)
env_file = ""                     # path to .env file (Docker --env-file)

[workspace]
mount_path = "/workspace"
extra_mounts = []                 # format: ["/host/path:/container/path:ro"]
ports = []                        # format: ["3000:3000", "8080:8080"] host:container
persist_home = true               # named Docker volume for /home/zone (survives container recreation)

[packages]
apt = []                          # merged: global defaults + per-repo (union)
pip = []
npm = []

[resources]
memory = ""                       # e.g., "8g" -- overrides global
cpus = ""                         # e.g., "4" -- overrides global
pids_limit = 512                  # max processes
gpus = ""                         # e.g., "all" / "2" / "device=0,1" -- overrides global

[network]
mode = "none"                     # overrides global if set
allow = [                         # additional allows (added to global default_allow)
  "api.openai.com",
]
deny = []                         # additional denials

[hooks]
pre_build = []                    # shell commands run on host before Docker build
post_stop = []                    # shell commands run on host after container stops

[harness]
# Harness-specific options (see Section 5)
version = ""                      # pin harness tool version (e.g., "1.0.26"). "" = latest.
skip_permissions = false          # claude-code only: --dangerously-skip-permissions (default OFF)
extra_args = []
```

### 4.3 Config Schema Versioning

Top-level `version = 1` field is required (integer, not string). On parse:
- Missing version: silently assume `1` (no warning at schema version 1; warn only when version 2+ exists)
- Unrecognized version: error with "zone.toml version X is not supported by this version of zone. Update zone or check https://github.com/jonathanung/zone"
- Future `zone migrate` command will auto-upgrade old configs (write new file, rename atomically)

### 4.4 Config Merge Strategy

Per-repo values **override** global values for scalar fields. For list fields:
- `packages.apt/pip/npm`: **union** (global defaults + per-repo additions, deduplicated)
- `network.allow`: per-repo `allow` is **appended** to global `default_allow`
- `network.deny`: per-repo `deny` is **appended** to global `default_deny`
- `auth.forward_env`: **union**
- `harness.extra_args`: **append** (per-repo appended to global, not replace)
- `hooks.pre_build/post_stop`: **append**
- `extra_mounts`, `ports`: per-repo **replaces** global

`zone config` output shows the merged result with source annotations:
```
[packages]
apt = ["git", "curl", "wget", "build-essential"]
#       ^^^^^^^^^^^^^^^^^^^^ (global)  ^^^^^^^^^^^^^^^^ (repo)
```

### 4.5 Config Validation

Use `BurntSushi/toml` strict decoding. Unknown keys produce an error with edit-distance suggestions (Levenshtein, max distance 3):
```
Error: Unknown config keys in zone.toml: [zone.skip_perms]
Did you mean [harness].skip_permissions?
```

**Mount validation:** `extra_mounts` paths are resolved via `filepath.EvalSymlinks()` before checking. The following are blocked and produce an error:
- `/var/run/docker.sock` (Docker socket escape)
- `/var/run/podman/`, `/var/run/containerd/` (container runtime sockets)
- Anything under `/proc`, `/sys`, `/dev` (kernel interfaces)
- `~/.ssh` (SSH keys -- use `forward_ssh_agent` instead)
- `/etc/shadow`, `/etc/passwd` (host credentials)
- `/` or `/etc` (host root)
- `~/.kube`, `~/.aws`, `~/.gcp`, `~/.azure`, `~/.docker`, `~/.gnupg` (cloud/registry credentials)
- `/boot`, `/lib/modules` (kernel)
- Any mount outside the current project directory produces a **warning** (not error)

Override with `--allow-dangerous-mount` for mounts that would be blocked.

**Extra mounts default to read-only.** Append `:rw` explicitly for read-write access:
```toml
extra_mounts = [
  "/data/models:/models:ro",       # read-only (default even without :ro)
  "/tmp/scratch:/scratch:rw",      # explicit read-write
]
```

**Harness-specific validation:** See Section 5 -- each harness validates its own config keys via the `Validate()` interface method. Errors are specific:
```
Error: harness "aider" does not support key "skip_permissions" (that key is specific to "claude-code").
```

**Additional validations:**
- `base_image` without a tag (e.g., `ubuntu` vs `ubuntu:24.04`): warn
- `forward_env` vars not set in host environment: warn (not error, may be set at launch time)
- `network.mode = "none"` with non-empty `allow` list: warn that allow list is ignored
- `network.allow/deny` patterns checked for syntactic validity
- Conflicting `ports` entries: error

### 4.6 Environment Variable Forwarding

`forward_env` supports both literal names and glob patterns:
```toml
[auth]
forward_env = ["ANTHROPIC_API_KEY", "AWS_*", "GOOGLE_*"]
```

Glob matching uses `filepath.Match` semantics (same as network rules).

**Pre-launch validation:** Before starting the container, Zone checks that every env var listed in the harness's `RequiredEnvVars()` is set on the host. Missing required vars produce an immediate, clear error *before* the Docker build starts:
```
Error: Required environment variable ANTHROPIC_API_KEY is not set.
The claude-code harness needs this variable. Set it and re-run zone launch.
```

**`.env` file support:** When `auth.env_file` is set, Zone passes it as Docker's `--env-file`. This avoids putting secret names in a committed `zone.toml`.

### 4.7 SSH Agent Forwarding

```toml
[auth]
forward_ssh_agent = true
```

When enabled, Zone mounts the host's `SSH_AUTH_SOCK` as a volume and sets the env var inside the container. Private keys never enter the container -- only the agent socket. This is the recommended approach for git push/pull inside the container.

`~/.ssh` is blocked as a dangerous mount. Use `forward_ssh_agent` instead.

### 4.8 Network Rule Syntax

Rules are strings. Two formats:
- **Literal hostname:** `"api.anthropic.com"` -- exact match
- **Glob pattern:** `"*.anthropic.com"` -- standard glob, `*` matches any subdomain segment

Glob matching uses `filepath.Match` semantics for Phase 1. For more advanced globbing (e.g. `**` recursive), use `gobwas/glob` in Phase 2.

**Phase 1 constraint:** Only literal hostnames and simple globs (`*.domain.com`) are supported in network enforcement. Complex patterns are validated at config parse time and rejected with a clear error.

Evaluation order for whitelist mode:
1. Check deny list first (deny always wins)
2. Check allow list (global default_allow + per-repo allow)
3. Default: block

Evaluation order for blocklist mode:
1. Check deny list (global default_deny + per-repo deny)
2. Default: allow

Mode `"none"`: no network restrictions applied (container gets default Docker networking). **This is the default for Phase 1** to minimize first-run friction. Users opt into sandboxing.

### 4.9 Network Implementation

**Important design notes:**

1. **Network restrictions apply at container runtime only, NOT during image build.** The Docker build uses the host's default networking so `apt-get update`, `npm install`, etc. always work.
2. **Docker's embedded DNS (127.0.0.11) does NOT filter queries.** It resolves all hostnames. Network enforcement is purely via iptables/firewall rules.

**Phase 1 approach (Linux only):** Host-side Docker network isolation via iptables.

1. Each zone container gets its own Docker bridge network: `zone-net-{container-hash}` (created **without** `--internal`, since `--internal` blocks all external traffic and makes selective iptables filtering impossible)
2. Zone resolves allowed hostnames to IPs from the host, then configures iptables rules on the host's Docker network bridge
3. Default policy: DROP all outbound from the container's network
4. ACCEPT rules added for each resolved IP from the allow list
5. IPv6 is disabled on the container network (`--sysctl net.ipv6.conf.all.disable_ipv6=1`) to prevent IPv6 bypass of iptables rules
6. iptables rules are tagged with comments (`-m comment --comment "zone-{hash}"`) for identification and cleanup
7. Rules are periodically refreshed (every 5 minutes) by re-resolving hostnames in a background goroutine

**macOS limitation:** macOS Docker Desktop runs containers inside a Linux VM. Host-side iptables is not directly accessible from macOS. For Phase 1, network whitelisting/blocklisting is **Linux-only**. On macOS, if `network.mode` is set to `whitelist` or `blocklist`, Zone warns: "Network filtering is not available on macOS in this version. Container will have unrestricted network access. Set `network.mode = \"none\"` to suppress this warning." Phase 2's DNS proxy sidecar resolves this cross-platform.

**Host-side iptables requires elevated privileges.** Zone uses `sudo iptables` only for the firewall rule commands, not the entire tool. If sudo is unavailable, warn and fall back to `mode = "none"`.

The generated firewall rules are cached in `.zone/firewall.rules` for inspectability.

**Why not in-container iptables:** Granting `CAP_NET_ADMIN` to a container running an LLM agent with shell access is a significant privilege escalation risk. The agent could modify or disable the firewall rules. Host-side enforcement is the correct security boundary.

**Known limitation:** IP-based enforcement can be bypassed when blocked services share CDN infrastructure with allowed services. Document this. Phase 2's DNS proxy sidecar provides DNS-level filtering.

**Phase 2:** DNS proxy sidecar container for hostname-level filtering at DNS query time. Cross-platform (works on both Linux and macOS).

### 4.10 Auth Config Mount Strategy (copy-on-start)

When `mount_home_config = true`, Zone does NOT mount the host config directory read-only into the container (which would prevent the harness from writing to its own config). Instead:

1. Mount the host's config dir (e.g., `~/.claude`) to a staging location: `/home/zone/.claude-host` (read-only)
2. In the entrypoint, copy contents to the actual config location: `cp -a /home/zone/.claude-host/. /home/zone/.claude/`
3. The harness gets a writable copy; the host's originals are preserved

If the host config directory doesn't exist, skip the mount with a warning (not an error).

### 4.11 Proxy Support

```toml
[network]
http_proxy = ""
https_proxy = ""
no_proxy = ""
```

When set, these are passed as `--build-arg` during Docker build and as environment variables during container runtime. If unset in config, Zone auto-detects `HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY` from the host environment and forwards them. When whitelist mode is active, the proxy server hostname is automatically added to the allow list.

---

## 5. Harness-Specific Config Keys

### `[harness]` section keys by harness type

**All harnesses share:**
- `version` (string, default: ""), pin harness tool version (e.g., `"1.0.26"`). Empty means latest.
- `extra_args` (list[string]), forwarded to harness command

**claude-code:**
- `skip_permissions` (bool, **default: false**), adds `--dangerously-skip-permissions`
- `node_version` (string, default: "22"), Node.js major version

**opencode:**
- `skip_permissions` (bool, **default: false**), adds `--dangerously-skip-permissions`
- `node_version` (string, default: "22")

**gemini-cli:**
- `node_version` (string, default: "22")

**aider:**
- `python_version` (string, default: "3.12")

**codex-cli:**
- `skip_permissions` (bool, **default: false**), adds `--dangerously-bypass-approvals-and-sandbox`
- `node_version` (string, default: "22")

**custom:**
- `install_commands` (list[string]), shell commands to install the harness
- `entrypoint_command` (string), the command to exec into
- `config_dirs` (list[string], optional), host directories to mount for auth
- `required_env` (list[string]), env vars that must be set
- `health_check` (string, optional), command to verify install (e.g., `"my-tool --version"`)
- `aliases` (map[string]string, optional), shell aliases to register
- `shell_rc` (list[string], optional), lines to add to .bashrc/.zshrc

**Security note on `skip_permissions`:** When `zone init` detects `harness = "claude-code"`, the wizard shows:
```
  skip_permissions is currently OFF (recommended).
  Enabling it allows the AI to modify files without approval.
  Only enable in sandboxed/disposable environments.

[s] Enable skip_permissions  [Enter] Keep OFF
```

### Harness-specific config validation

Each harness validates that only its supported keys are used. Setting `skip_permissions` on an `aider` harness produces:
```
Error: harness "aider" does not support key "skip_permissions" (that key is specific to "claude-code").
```

Values are type-checked at parse time. `python_version = 3.12` (float) is rejected with: "python_version must be a string (use `\"3.12\"` not `3.12`)."

---

## 6. `.zone/` Cache Directory

```
.zone/
+-- .lock              # flock-based file lock for concurrent access protection
+-- config.hash        # SHA256 hex digest of full cache key (see 6.2)
+-- Dockerfile         # last generated Dockerfile (human-inspectable)
+-- entrypoint.sh      # last generated entrypoint
+-- zone-bashrc        # last generated shell RC file
+-- firewall.rules     # last generated network firewall rules (host-side)
+-- image_id           # Docker image ID (sha256:...)
+-- container_id       # Docker container ID (for reattach)
+-- network_id         # Docker network ID (for cleanup)
+-- logs/
    +-- last_build.log # stdout/stderr from last docker build
```

### 6.1 Concurrent Access Protection

Before any cache read/write, acquire an exclusive flock on `.zone/.lock` via `syscall.Flock()`. Non-blocking attempt. If lock cannot be acquired, error with exit code 5: "Another zone process is operating on this repo. Wait or run `zone clean` to reset."

### 6.2 Cache Hash

The cache hash includes more than just the merged config to ensure upgrades and template changes trigger rebuilds:

```go
hash := sha256.New()
hash.Write([]byte(mergedConfigJSON))       // effective merged config
hash.Write([]byte(templates.DockerfileTmpl)) // Dockerfile template
hash.Write([]byte(templates.EntrypointTmpl)) // entrypoint template
hash.Write([]byte(version))                  // Zone binary version (from ldflags)
```

This ensures that upgrading Zone automatically invalidates stale cached images.

### 6.3 Gitignore

`zone init` and `zone launch` (when auto-creating config) must add `.zone/` to `.gitignore` if a `.gitignore` exists and `.zone/` is not already in it.

---

## 7. Project Structure

```
zone/
+-- go.mod
+-- go.sum
+-- main.go                       # entry point, version vars, Cobra root
+-- Makefile                      # build, test, lint targets
+-- .goreleaser.yml               # binary distribution config
+-- README.md
+-- cmd/
|   +-- root.go                   # Cobra root command + global flags + signal handling + TTY detection
|   +-- init.go                   # zone init (BubbleTea wizard)
|   +-- launch.go                 # zone launch (idempotent lifecycle)
|   +-- join.go                   # zone join (multi-terminal attach)
|   +-- exec.go                   # zone exec (one-off commands)
|   +-- shell.go                  # zone shell (debug shell)
|   +-- build.go                  # zone build (standalone build)
|   +-- stop.go                   # zone stop + container/network removal
|   +-- restart.go                # zone restart
|   +-- ls.go                     # zone ls (list all zones)
|   +-- logs.go                   # zone logs
|   +-- clean.go                  # zone clean
|   +-- destroy.go                # zone destroy (full teardown)
|   +-- status.go                 # zone status
|   +-- config.go                 # zone config (show merged config)
|   +-- validate.go               # zone validate
+-- internal/
|   +-- config/
|   |   +-- types.go              # Config struct definitions (shared types)
|   |   +-- harness_config.go     # Typed HarnessConfig struct (union of all harness fields)
|   |   +-- config.go             # TOML parsing, strict decoding
|   |   +-- global.go             # Global config read/write (~/.config/zone/)
|   |   +-- merge.go              # Two-tier merge strategy
|   |   +-- validate.go           # Config validation (dangerous mounts, unknown keys, env vars)
|   +-- cache/
|   |   +-- cache.go              # .zone/ directory management
|   |   +-- hash.go               # Full cache hash (config + templates + version)
|   |   +-- lock.go               # flock-based concurrent access protection
|   +-- docker/
|   |   +-- manager.go            # Build, run, attach, stop, status, list
|   |   +-- dockerfile.go         # Dockerfile generation (text/template)
|   |   +-- entrypoint.go         # Entrypoint script generation
|   |   +-- shellrc.go            # Shell RC file generation (aliases, prompt, welcome)
|   |   +-- naming.go             # Deterministic container + network naming + labels
|   |   +-- network.go            # Docker network create/destroy + host-side firewall
|   |   +-- platform.go           # Platform detection (macOS, Linux, rootless Docker)
|   |   +-- errors.go             # Sentinel errors
|   +-- network/
|   |   +-- firewall.go           # Host-side iptables rule generation + cleanup
|   |   +-- rules.go              # Rule parsing (literal + glob)
|   |   +-- matcher.go            # Hostname glob matching engine (precompiled)
|   +-- harness/
|   |   +-- harness.go            # Harness interface + registry + BaseHarness
|   |   +-- claude_code.go        # fully implemented
|   |   +-- opencode.go           # stub
|   |   +-- gemini_cli.go         # stub
|   |   +-- aider.go              # stub
|   |   +-- codex_cli.go          # stub
|   |   +-- custom.go             # user-defined harness from config
|   +-- tui/
|       +-- init_wizard.go        # BubbleTea init wizard model
|       +-- build_progress.go     # BubbleTea build progress model
|       +-- status_view.go        # BubbleTea live status model
|       +-- log_viewer.go         # BubbleTea log viewer model
+-- pkg/
|   +-- templates/
|       +-- templates.go          # //go:embed declarations
|       +-- Dockerfile.tmpl       # Go text/template Dockerfile
|       +-- entrypoint.sh.tmpl    # Go text/template entrypoint
|       +-- zone-bashrc.tmpl      # Go text/template shell RC
+-- tests/
    +-- config_merge_test.go      # Config merge logic (write these FIRST)
    +-- harness_validate_test.go  # Harness-specific config validation
    +-- validate_test.go          # Dangerous mount detection, symlink resolution
    +-- naming_test.go            # Container name generation
    +-- matcher_test.go           # Network rule matching
    +-- hash_test.go              # Cache hash includes templates + version
```

### Import graph (enforced):
```
cmd/* -> internal/*       OK
cmd/* -> pkg/templates    OK
internal/docker -> internal/config   OK
internal/docker -> internal/cache    OK
internal/docker -> internal/network  OK
internal/docker -> internal/harness  OK
internal/docker -> pkg/templates     OK
internal/* -> cmd/*       FORBIDDEN (breaks encapsulation)
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
    ErrNoContainer      = errors.New("no running container for this repo")
    ErrLockContention   = errors.New("another zone process is operating on this repo")
    ErrDangerousMount   = errors.New("mount path is blocked for security")
    ErrDockerNotRunning = errors.New("docker daemon is not running")
    ErrNetworkUnsupported = errors.New("network filtering not available on this platform")
)
```

Cmd layer maps sentinel errors to exit codes. Internal packages never call `os.Exit()`.

**Error wrapping in `NewManager`:** Use the actual error from the Docker SDK, not a generic sentinel:
```go
func NewManager(cfg *config.MergedConfig, c *cache.Cache) (*Manager, error) {
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        return nil, fmt.Errorf("docker client init: %w", err)
    }
    // Verify connectivity
    if _, err := cli.Ping(context.Background()); err != nil {
        return nil, fmt.Errorf("%w: %v", ErrDockerNotRunning, err)
    }
    return &Manager{client: cli, config: cfg, cache: c}, nil
}
```

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
    // --- Identity ---
    Name() string
    Version() string // "" = latest

    // --- Installation (Dockerfile generation) ---
    InstallCommands() []string       // run as root
    PostInstallCommands() []string   // run as zone user (after USER zone)
    HealthCheck() string             // final RUN verify, e.g. "claude --version"

    // --- Runtime ---
    EntrypointCommand() string
    PromptFlag() string              // e.g. "-p" for claude, "--message" for aider
    RequiredEnvVars() []string

    // --- Dependencies ---
    HomeConfigDir() string           // primary config dir, e.g. "~/.claude"
    ExtraConfigDirs() []string       // additional config dirs (for multi-tool custom harness)
    DefaultAptPackages() []string
    DefaultNpmPackages() []string
    DefaultPipPackages() []string
    NeedsNode() bool
    NeedsPython() bool

    // --- Shell experience ---
    ShellRC() []string               // lines appended to ~/.zone-bashrc
    Aliases() map[string]string      // e.g. {"cc": "claude --dangerously-skip-permissions"}
    WelcomeMessage() string          // shown on first interactive attach

    // --- Lifecycle ---
    Validate() error                 // harness-specific config validation
}
```

### BaseHarness (default implementations)

```go
type BaseHarness struct{}

func (b BaseHarness) Version() string              { return "" }
func (b BaseHarness) PostInstallCommands() []string { return nil }
func (b BaseHarness) HealthCheck() string           { return "" }
func (b BaseHarness) PromptFlag() string            { return "" }
func (b BaseHarness) ExtraConfigDirs() []string     { return nil }
func (b BaseHarness) ShellRC() []string             { return nil }
func (b BaseHarness) Aliases() map[string]string    { return nil }
func (b BaseHarness) WelcomeMessage() string        { return "" }
func (b BaseHarness) Validate() error               { return nil }
```

Each concrete harness embeds `BaseHarness` and overrides only what it needs. Adding a new harness = adding one Go file.

### Registry

```go
var registry = map[string]func(config *HarnessConfig) Harness{
    "claude-code": func(c *HarnessConfig) Harness { return &ClaudeCode{config: c} },
    "opencode":    func(c *HarnessConfig) Harness { return &OpenCode{config: c} },
    "gemini-cli":  func(c *HarnessConfig) Harness { return &GeminiCLI{config: c} },
    "aider":       func(c *HarnessConfig) Harness { return &Aider{config: c} },
    "codex-cli":   func(c *HarnessConfig) Harness { return &CodexCLI{config: c} },
    "custom":      func(c *HarnessConfig) Harness { return &Custom{config: c} },
}

func Get(name string, config *HarnessConfig) (Harness, error) {
    factory, ok := registry[name]
    if !ok {
        return nil, fmt.Errorf("unknown harness %q, available: %v", name, availableNames())
    }
    h := factory(config)
    if err := h.Validate(); err != nil {
        return nil, fmt.Errorf("harness %q config: %w", name, err)
    }
    return h, nil
}
```

### Typed HarnessConfig (replaces `map[string]interface{}`)

```go
// internal/config/harness_config.go
type HarnessConfig struct {
    // Common
    Version   string   `toml:"version"`
    ExtraArgs []string `toml:"extra_args"`

    // Claude Code
    SkipPermissions *bool  `toml:"skip_permissions"`
    NodeVersion     string `toml:"node_version"`

    // Aider
    PythonVersion string `toml:"python_version"`

    // Custom
    InstallCommands   []string          `toml:"install_commands"`
    EntrypointCommand string            `toml:"entrypoint_command"`
    ConfigDirs        []string          `toml:"config_dirs"`
    RequiredEnv       []string          `toml:"required_env"`
    CustomHealthCheck string            `toml:"health_check"`
    CustomAliases     map[string]string `toml:"aliases"`
    CustomShellRC     []string          `toml:"shell_rc"`
}
```

Each harness's `Validate()` method checks that only its relevant fields are set and rejects others with a clear error.

### Stub behavior

Stubs return descriptive errors via `Validate()`, not panics:
```go
func (o *OpenCode) Validate() error {
    return fmt.Errorf("the %q harness is not yet fully implemented; use harness = \"custom\" with install_commands and entrypoint_command to configure it manually", o.Name())
}
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

//go:embed zone-bashrc.tmpl
var ZoneBashrcTmpl string
```

Imported by `internal/docker/dockerfile.go` as `import "zone/pkg/templates"`.

### Template: `Dockerfile.tmpl`

```dockerfile
# syntax=docker/dockerfile:1
FROM {{ .BaseImage }}

ARG HOST_UID={{ .HostUID }}
ARG DEBIAN_FRONTEND=noninteractive

# Base system packages (rarely change -- cached layer)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    git \
    sudo \
    && rm -rf /var/lib/apt/lists/*

{{- if .AptPackages }}

# User-specified apt packages
RUN apt-get update && apt-get install -y --no-install-recommends \
{{- range .AptPackages }}
    {{ . }} \
{{- end }}
    && rm -rf /var/lib/apt/lists/*
{{- end }}

{{- if .NeedsNode }}

# Node.js {{ .NodeVersion }}
RUN curl -fsSL https://deb.nodesource.com/setup_{{ .NodeVersion }}.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*
{{- end }}

{{- if .NeedsPython }}

# Python {{ .PythonVersion }}
RUN apt-get update && apt-get install -y --no-install-recommends \
    python{{ .PythonVersion }} python3-pip \
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

{{- if .HarnessInstallCommands }}

# Harness installation
RUN {{ join .HarnessInstallCommands " && " }}
{{- end }}

{{- if .HealthCheck }}

# Verify harness installed correctly
RUN {{ .HealthCheck }}
{{- end }}

{{- if .InstallZsh }}

# Zsh
RUN apt-get update && apt-get install -y --no-install-recommends zsh \
    && rm -rf /var/lib/apt/lists/*
{{- end }}

# Create non-root user with scoped sudo
{{- if eq .HostUID 0 }}
# Running as root (CI environment) -- skip user creation
{{- else }}
RUN useradd -m -s /bin/{{ .Shell }} -u ${HOST_UID} zone \
    && echo "zone ALL=(ALL) NOPASSWD: /usr/bin/apt-get, /usr/bin/apt, /usr/bin/pip*, /usr/bin/npm, /usr/local/bin/npm" >> /etc/sudoers
{{- end }}

{{- if .MacOSUsername }}

# macOS plugin symlink compatibility
RUN mkdir -p /Users/{{ .MacOSUsername }} \
    && ln -sf /home/zone /Users/{{ .MacOSUsername }}
{{- end }}

{{- if ne .HostUID 0 }}
USER zone
{{- end }}

{{- if .PostInstallCommands }}

# Post-install (as zone user)
RUN {{ join .PostInstallCommands " && " }}
{{- end }}

WORKDIR {{ .MountPath }}

COPY entrypoint.sh /entrypoint.sh
COPY zone-bashrc /home/zone/.zone-bashrc

ENTRYPOINT ["/entrypoint.sh"]
```

**Security notes on the Dockerfile:**
- `sudo` is scoped to package management commands only (`apt-get`, `apt`, `pip`, `npm`), not `ALL`. This prevents the LLM agent from using sudo for privilege escalation while still allowing runtime package installs.
- `MacOSUsername` is sanitized to `[a-zA-Z0-9_.-]` before injection (same regex as container name sanitization).
- `HostUID = 0` (root, common in CI) skips user creation entirely to avoid `useradd` failure.
- Harness install commands are collapsed into a single `RUN` layer for cache efficiency.

### Template: `entrypoint.sh.tmpl`

```bash
#!/bin/bash
set -e

# Mark workspace as git-safe
git config --global --add safe.directory {{ .MountPath }}

{{- if .ForwardGitConfig }}

# Forward host git identity
git config --global user.name "{{ .GitUserName }}"
git config --global user.email "{{ .GitUserEmail }}"
{{- end }}

# Copy host auth config into writable location
{{- range .ConfigCopyCommands }}
{{ . }}
{{- end }}

# Source shell customizations
echo 'source ~/.zone-bashrc' >> ~/.bashrc
{{- if eq .Shell "zsh" }}
echo 'source ~/.zone-bashrc' >> ~/.zshrc
{{- end }}

# Execute harness command, forwarding any extra args
exec {{ .EntrypointCommand }} "$@"
```

### Template: `zone-bashrc.tmpl`

```bash
# Generated by zone -- do not edit
export ZONE_HARNESS="{{ .HarnessName }}"
export ZONE_WORKSPACE="{{ .MountPath }}"

# Zone-aware prompt
export PS1="[zone:{{ .HarnessName }}] \u:\w\$ "

# Aliases
{{- range $alias, $cmd := .Aliases }}
alias {{ $alias }}="{{ $cmd }}"
{{- end }}

# Harness-specific shell RC
{{- range .ShellRC }}
{{ . }}
{{- end }}

# Welcome message (shown once per session)
if [ -z "$ZONE_WELCOMED" ]; then
{{- if .WelcomeMessage }}
    echo "{{ .WelcomeMessage }}"
{{- else }}
    echo "Zone workspace: {{ .HarnessName }}"
    echo "Workspace: {{ .MountPath }}"
{{- end }}
    echo ""
    export ZONE_WELCOMED=1
fi
```

Note: No firewall script in the container. Network filtering is host-side (see Section 4.9).

---

## 12. Docker Manager

### Key responsibilities of `internal/docker/manager.go`

Uses the **Go Docker SDK** (`github.com/docker/docker/client`). Client is initialized once in constructor:

```go
type Manager struct {
    client   *client.Client
    config   *config.MergedConfig
    cache    *cache.Cache
    platform Platform  // detected platform info
}

func NewManager(cfg *config.MergedConfig, c *cache.Cache) (*Manager, error) {
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        return nil, fmt.Errorf("docker client init: %w", err)
    }
    // Verify connectivity
    if _, err := cli.Ping(context.Background()); err != nil {
        return nil, fmt.Errorf("%w: %v", ErrDockerNotRunning, err)
    }
    plat := DetectPlatform(cli)
    return &Manager{client: cli, config: cfg, cache: c, platform: plat}, nil
}

func (m *Manager) Close() error { return m.client.Close() }

func (m *Manager) GenerateDockerfile(ctx context.Context) (string, error)
func (m *Manager) Build(ctx context.Context, noCache bool) (<-chan BuildProgress, error)
func (m *Manager) Launch(ctx context.Context, opts LaunchOpts) error
func (m *Manager) Join(ctx context.Context, asRoot bool) error
func (m *Manager) Shell(ctx context.Context, asRoot bool) error
func (m *Manager) Exec(ctx context.Context, cmd []string, allocTTY bool) error
func (m *Manager) Stop(ctx context.Context, timeout int) error
func (m *Manager) Logs(ctx context.Context, opts LogOpts) error
func (m *Manager) Status(ctx context.Context) (*ContainerStatus, error)
func (m *Manager) ListAll(ctx context.Context) ([]ZoneContainer, error)
```

### Container + Network Naming Convention

Deterministic from repo absolute path. 16-char hash, sanitized name. Containers are labeled for discovery by `zone ls`:

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

// Labels applied to every zone container for discovery
func ContainerLabels(repoPath, harness string) map[string]string {
    return map[string]string{
        "com.zone.managed":   "true",
        "com.zone.repo-path": repoPath,
        "com.zone.harness":   harness,
    }
}
```

### Container Creation Security Flags

All containers are created with hardened security settings:

```go
hostConfig := &container.HostConfig{
    SecurityOpt: []string{"no-new-privileges"},
    CapDrop:     []string{"ALL"},
    CapAdd:      []string{"CHOWN", "DAC_OVERRIDE", "SETGID", "SETUID", "FOWNER"},
    Resources: container.Resources{
        Memory:    memoryLimit,   // from config, 0 = no limit
        NanoCPUs:  cpuLimit,     // from config, 0 = no limit
        PidsLimit: &pidsLimit,   // from config, default 512
    },
    Sysctls: map[string]string{
        "net.ipv6.conf.all.disable_ipv6": "1",  // prevent IPv6 bypass
    },
}
```

### Home Volume Persistence

When `persist_home = true` (default), a named Docker volume is created for `/home/zone`:

```go
volumeName := fmt.Sprintf("zone-home-%s", shortHash)
```

This volume survives container recreation, preserving harness state, npm cache, shell history, etc. It is cleaned up by `zone destroy` but not by `zone stop` or `zone clean`.

### Interactive TTY Attach

For interactive `zone launch`, `zone join`, `zone shell`, and `zone exec`, use `os/exec` with `docker exec -it` rather than the Go Docker SDK's `ContainerExecAttach`. The SDK's hijacked connection API is notoriously difficult for proper raw terminal I/O (resize handling, signal forwarding).

```go
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
    // Forward DOCKER_HOST if set, so CLI targets same daemon as SDK
    if dockerHost := os.Getenv("DOCKER_HOST"); dockerHost != "" {
        c.Env = append(os.Environ(), "DOCKER_HOST="+dockerHost)
    }
    return c.Run()
}
```

Use the SDK for everything non-interactive: build, create, start, stop, inspect, remove, network create/destroy.

### Platform Detection

```go
type Platform struct {
    OS            string // "linux", "darwin"
    IsDockerDesktop bool
    IsRootless      bool
    SupportsIPTables bool
}

func DetectPlatform(cli *client.Client) Platform {
    info, _ := cli.Info(context.Background())
    // Check for rootless Docker
    isRootless := strings.Contains(strings.Join(info.SecurityOptions, ","), "rootless")
    // macOS always uses Docker Desktop VM
    isMacOS := runtime.GOOS == "darwin"
    return Platform{
        OS:              runtime.GOOS,
        IsDockerDesktop: isMacOS || strings.Contains(info.OperatingSystem, "Docker Desktop"),
        IsRootless:      isRootless,
        SupportsIPTables: runtime.GOOS == "linux" && !isRootless,
    }
}
```

### BuildKit

All builds use BuildKit (`DOCKER_BUILDKIT=1`). The Dockerfile template includes `# syntax=docker/dockerfile:1` header.

---

## 13. BubbleTea TUI Components

BubbleTea is used consistently across all interactive commands when stdout is a TTY. Every BubbleTea model lives in `internal/tui/` and is imported by the corresponding `cmd/` file.

### `zone init` -- Init Wizard (`internal/tui/init_wizard.go`)

Interactive harness selector with auto-detection hints and config preview:

```
? Select your LLM harness:

  Detected: .claude/ directory, CLAUDE.md
  Suggested: Claude Code

  > Claude Code    . Anthropic's agentic coding tool  * detected
    OpenCode       . Open-source coding agent (stub)
    Gemini CLI     . Google's Gemini in the terminal (stub)
    Aider          . AI pair programming in the terminal (stub)
    Codex CLI      . OpenAI's coding agent (stub)
    Custom         . Define your own harness

  up/down navigate  .  enter select  .  q quit
```

After selection, show config preview before writing:

```
Claude Code Harness
+- Base: ubuntu:24.04
+- Packages: git, curl, nodejs (v22)
+- Network: None (unrestricted)
+- Auth: ~/.claude copied into container
+- skip_permissions: OFF
+- Prompt flag: -p

  skip_permissions is OFF (recommended).
  Enabling it allows the AI to modify files without approval.

[s] Enable skip_permissions  [n] Network sandboxing  [c] Customize  [Enter] Confirm  [q] Cancel
```

The generated `zone.toml` includes commented-out options showing all available fields for discoverability.

### `zone launch` -- Build Progress (`internal/tui/build_progress.go`)

When a rebuild is triggered, stream Docker build output through a BubbleTea viewport with spinner:

```
  Building zone image...

Step 1/8 : FROM ubuntu:24.04
 ---> a1b2c3d4e5f6
Step 2/8 : RUN apt-get update...
 ---> Running in f6e5d4c3b2a1

  down scroll  .  ctrl+c cancel
```

Subscribes to the `<-chan BuildProgress` from `Manager.Build()` via a `tea.Cmd` that reads from the channel. On build complete, transitions to TTY attach (or prints container ID if `--headless`).

### `zone status` -- Live Status View (`internal/tui/status_view.go`)

```
-- Zone Status --------------------------
| Repo:      my-project                  |
| Harness:   claude-code                 |
| Container: zone-my-project-a1b2c3d4    |
| Status:    . Running (2h 14m)          |
| Image:     sha256:abcdef...            |
| Network:   none (unrestricted)         |
| Ports:     3000/tcp -> 0.0.0.0:3000    |
| Resources: 8g mem, 4 cpus, 512 pids    |
| Mounts:    /workspace (rw)             |
|            ~/.claude (copy-on-start)   |
------------------------------------------
  q quit  .  r restart  .  s stop
```

Polls container status every 2 seconds via Docker SDK. `--json` flag bypasses BubbleTea entirely and prints raw JSON to stdout (for scripting).

### `zone logs` -- Log Viewer (`internal/tui/log_viewer.go`)

BubbleTea viewport with auto-scroll and search:

```
[14:32:01] claude: Reading project files...
[14:32:03] claude: Found 42 source files
[14:32:05] claude: Analyzing codebase structure...
|

  up/down scroll  .  / search  .  f follow  .  q quit
```

`--follow` starts in follow mode (auto-scroll to bottom). Without `--follow`, starts paused at the end. `--build` loads from `.zone/logs/last_build.log` instead of live container logs.

When stdout is piped, outputs plain text (no TUI chrome). `zone logs -f | grep error` works.

### `zone ls` -- Container List

```
NAME                          HARNESS       STATUS     UPTIME    REPO
zone-my-project-a1b2c3d4      claude-code   Running    2h 14m    /Users/alice/my-project
zone-api-server-e5f6a7b8      aider         Stopped    -         /Users/alice/api-server
```

Plain text table (no BubbleTea needed). Discovers containers via Docker label filter `com.zone.managed=true`.

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
1. Project scaffolding: go.mod, Cobra commands (including aliases), project structure, version injection, global flags (`--verbose`, `--debug`, `--quiet`, `--plain`), TTY detection
2. Config parsing: zone.toml + global config + merge logic + strict validation + dangerous mount blocking (with symlink resolution) + harness-specific validation via typed `HarnessConfig`
3. `zone init` -- BubbleTea wizard with harness auto-detection, config preview, generate commented zone.toml, add .zone/ to .gitignore, detect existing configs, `--harness` + `--set` for non-interactive
4. `zone launch` -- full idempotent lifecycle with flock (generate, build, run, reattach), auto-init when no zone.toml, `--prompt`/`-p` flag, `--headless`, `--port`, all container states handled, config change detection, lock released before TTY attach, env var pre-validation
5. `zone join` -- multi-terminal shell attach (plain bash/zsh, not harness re-run)
6. `zone exec` -- one-off command execution with TTY auto-detection
7. `zone shell` -- debug shell access (with `--root`)
8. `zone stop` -- stop + remove container + network, clear cache files
9. `zone restart` -- stop + relaunch
10. `zone build` -- standalone build command
11. `zone ls` -- list all zone containers via Docker labels
12. `zone status` -- BubbleTea live status view (+ `--json` bypass)
13. `zone logs` -- BubbleTea log viewer with follow/search, plain text when piped (+ `--build` for build logs)
14. `zone config` -- show merged config with source annotations (+ `--json`, `--edit`, `--schema`)
15. `zone validate` -- config validation without launching
16. `zone clean` / `zone destroy` -- cache and artifact cleanup
17. Claude Code harness -- fully implemented, skip_permissions defaults OFF, prompt flag, health check, shell aliases, welcome message
18. `.zone/` caching -- full cache hash (config + templates + version), Dockerfile, image ID, container ID, flock
19. Port forwarding -- config + ad-hoc flag
20. Resource limits -- memory, cpus, pids_limit with Docker HostConfig
21. Container security -- cap-drop ALL, no-new-privileges, scoped sudo, IPv6 disabled
22. Home volume persistence -- named Docker volume for /home/zone
23. Shell experience -- zone-bashrc template with prompt, aliases, welcome message
24. Copy-on-start auth config mounting
25. SSH agent forwarding
26. Platform detection (macOS, Linux, rootless Docker)
27. All BubbleTea TUI models: init wizard, build progress, status view, log viewer
28. Actionable error messages with platform-specific remediation hints

### Phase 2
29. Network whitelist/blocklist -- host-side iptables (Linux only), periodic refresh, cleanup
30. DNS proxy sidecar for cross-platform network filtering (macOS + Linux)
31. Custom harness support (install_commands, entrypoint, config_dirs, aliases, shell_rc)
32. GoReleaser setup + Homebrew formula
33. `zone init --from-devcontainer` migration
34. Rootless Docker detection and graceful degradation
35. Proxy support (http_proxy, https_proxy)
36. Container health checks (HEALTHCHECK in Dockerfile, harness health_check method)
37. Lifecycle hooks (pre_build, post_stop)
38. `zone config --global --edit` global config management

### Phase 3
39. OpenCode harness
40. Gemini CLI harness
41. Aider harness
42. Codex CLI harness
43. Advanced glob network rules via DNS proxy sidecar
44. `zone migrate` for config schema upgrades
45. Homebrew tap
46. Third-party harness definitions (`~/.config/zone/harnesses/*.toml`)
47. Zone profiles (`zone.gpu.toml`, `zone.minimal.toml`, `zone launch --profile gpu`)
48. GPU support (`[zone] gpus = "all"`)
49. WSL2 documentation and testing

---

## 16. Key Design Principles

1. **Zero-friction launch.** `zone launch --harness claude-code` is the zero-to-agent single command. No config file required.
2. **Idempotent launch.** `zone launch` handles everything: init, build, run, reattach. It is the only command most users need.
3. **Inspectable cache.** `.zone/Dockerfile` is a real Dockerfile the user can read and debug. No magic.
4. **Harness as plugin.** Adding a new harness = adding one Go file with `BaseHarness` embedding. No core changes.
5. **Two-tier config.** Global defaults + per-repo overrides. Predictable merge semantics. Commented generated config for discoverability.
6. **No global mutable state.** Cache is per-repo in `.zone/`. Global config is read-only defaults.
7. **Docker SDK + exec fallback.** SDK for management, `docker exec -it` for interactive TTY. Same daemon targeting via `DOCKER_HOST`.
8. **Fail loud with remediation.** Missing config, missing Docker, missing env vars all produce clear error messages with platform-specific hints and exit codes.
9. **Secure by default.** `skip_permissions` off. Cap-drop ALL. No-new-privileges. Scoped sudo. IPv6 disabled. Dangerous mount blocking with symlink resolution. Per-container network isolation.
10. **Config versioned.** Schema versioning from day one prevents future migration pain.
11. **TTY-aware.** BubbleTea TUI in interactive terminals, plain text in pipes/CI. Automatic detection with `--plain` override.
12. **Scriptable.** `--json` on all structured output. Exit codes for all error classes. Composable with pipes and scripts.
13. **Cross-platform.** Linux and macOS supported. Platform-specific features (network filtering) degrade gracefully with clear messaging.

---

## 17. Platform Support

| Platform | Status | Notes |
|----------|--------|-------|
| **Linux** | Fully supported | Native Docker, iptables for network filtering |
| **macOS** | Fully supported | Docker Desktop or Colima. Network filtering limited to Phase 2 DNS proxy. macOS username symlink for plugin compatibility. |
| **Windows/WSL2** | Supported (Phase 3) | Run Zone inside WSL2 with Docker Desktop's WSL2 backend. Not supported on native Windows. |
| **Rootless Docker** | Partial | Detected at startup. Network filtering unavailable (iptables requires root). Falls back to `mode = "none"` with warning. |
| **Podman** | Untested | May work via `podman-docker` compatibility layer. Not officially supported. |

---

## 18. Notes for Claude Code

- Use `//go:embed` in `pkg/templates/templates.go`, not in `internal/docker/`. Embed paths are relative to the `.go` file.
- BubbleTea `tea.NewProgram().Run()` is blocking. Wrap in Cobra `RunE`, extract result from final model, check for cancellation. Use `tea.WithAltScreen()` for status and logs views. All TUI models follow the same integration pattern (see Section 13). Check `term.IsTerminal()` before creating BubbleTea programs.
- Build progress TUI subscribes to `<-chan BuildProgress` via a `tea.Cmd` that reads from the channel. Status view polls every 2 seconds. Log viewer wraps Docker SDK's `ContainerLogs` stream.
- Docker SDK client is initialized once in `NewManager()`, reused across all operations, closed via `defer mgr.Close()`. Verify connectivity with `Ping()` before proceeding.
- All Docker SDK methods take `context.Context` as first param. Create context with `signal.NotifyContext` in Cobra commands.
- For interactive TTY attach, use `os/exec` with `docker exec -it`, not the Go SDK attach API. Forward `DOCKER_HOST` env var to the exec'd command.
- Config merge logic is the most subtle part. Write `config_merge_test.go` FIRST before implementing.
- File lock via `syscall.Flock()` on `.zone/.lock`. Non-blocking attempt, exit code 5 on contention. **Release lock before TTY attach** so `zone join` can work concurrently.
- Container naming: 16-char hash, sanitize repo name to `[a-zA-Z0-9_.-]`. Apply `com.zone.managed` label for `zone ls` discovery.
- Cache hash must include merged config + Dockerfile template + entrypoint template + zone version string. This ensures binary upgrades invalidate stale images.
- Network: Do NOT use `--internal` on Docker networks (it blocks all traffic). Use a regular bridge network + host-side iptables for selective filtering. iptables is Linux-only; macOS falls back to unrestricted.
- Container security: `--security-opt no-new-privileges`, `--cap-drop ALL`, `--cap-add CHOWN,DAC_OVERRIDE,SETGID,SETUID,FOWNER`, `--sysctl net.ipv6.conf.all.disable_ipv6=1`, scoped sudo.
- Stubs should return validation errors caught by `Validate()`, NOT panic.
- Test `zone launch` idempotency early -- the container reattach + flock logic is the most critical path. Test all container states (running, exited, dead, created, paused, non-existent).
- Use `BurntSushi/toml` with strict decoding (`.Undecoded()` check) to catch typos in config keys.
- `HarnessConfig` is a typed struct, not `map[string]interface{}`. Each harness validates only its own fields.
- Handle `HostUID = 0` (root/CI) by skipping `useradd` in the Dockerfile template.
- Set `DOCKER_BUILDKIT=1` for all builds. Include `# syntax=docker/dockerfile:1` in template header.
- Harness install commands are collapsed into a single `RUN` layer (joined with ` && `) for Docker cache efficiency.
