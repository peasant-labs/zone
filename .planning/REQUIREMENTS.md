# Requirements: Zone

**Defined:** 2026-03-26
**Core Value:** Run `zone launch` in any repo and get a sandboxed Docker workspace for your LLM coding agent, with zero manual Docker configuration.

## v1 Requirements

### CLI Commands

- [ ] **CLI-01**: User can run `zone init` to scaffold a `zone.toml` in the current directory with interactive harness selection
- [ ] **CLI-02**: User can run `zone init --harness <name>` to scaffold non-interactively
- [ ] **CLI-03**: User can run `zone launch` to build (if needed) and attach to a container for this repo
- [ ] **CLI-04**: User can run `zone launch --headless -p "task"` to run a detached agent with a prompt
- [ ] **CLI-05**: User can run `zone launch --harness <name>` with no zone.toml for zero-config quickstart
- [ ] **CLI-06**: User can run `zone join` to attach a new shell to a running container without re-running the harness
- [ ] **CLI-07**: User can run `zone exec -- <cmd>` to run a one-off command inside the running container
- [ ] **CLI-08**: User can run `zone shell` to open an interactive shell even if no harness is running
- [ ] **CLI-09**: User can run `zone build` to force-rebuild the Docker image without launching
- [ ] **CLI-10**: User can run `zone stop` to stop and remove the container and network, retaining cache
- [ ] **CLI-11**: User can run `zone restart` to stop and relaunch the container
- [ ] **CLI-12**: User can run `zone ls` to list all zone containers across all repos
- [ ] **CLI-13**: User can run `zone logs` to view harness output, with `--follow` for live tailing
- [ ] **CLI-14**: User can run `zone logs --build` to view the last Docker build log
- [ ] **CLI-15**: User can run `zone clean` to remove .zone/ cache and optionally Docker image
- [ ] **CLI-16**: User can run `zone destroy` to fully tear down container, image, network, and cache
- [ ] **CLI-17**: User can run `zone status` to see container state, harness, uptime, ports, resources
- [ ] **CLI-18**: User can run `zone config` to show effective merged config with source annotations
- [ ] **CLI-19**: User can run `zone validate` to check zone.toml validity without launching
- [ ] **CLI-20**: User can use global flags `--verbose`, `--debug`, `--quiet`, `--plain` on any command
- [ ] **CLI-21**: User can forward arguments to the harness via `--` separator (e.g., `zone launch -- -p "fix"`)

### Configuration

- [x] **CFG-01**: User can create a minimal zone.toml with just `version = 1` and `harness = "claude-code"`
- [x] **CFG-02**: User can set global defaults in `~/.config/zone/config.toml` (XDG compliant)
- [x] **CFG-03**: Per-repo config overrides global for scalar fields
- [x] **CFG-04**: List fields merge correctly: packages union, network allow/deny append, extra_args append
- [x] **CFG-05**: Unknown config keys produce an error with edit-distance suggestions (Levenshtein)
- [x] **CFG-06**: Dangerous mount paths are blocked (docker.sock, /proc, /sys, ~/.ssh, cloud creds) with symlink resolution
- [x] **CFG-07**: `zone config` shows merged result with source annotations (global vs repo)
- [x] **CFG-08**: `zone config --json` outputs machine-readable merged config
- [x] **CFG-09**: Config schema version field (`version = 1`) is validated on parse
- [ ] **CFG-10**: Environment variable forwarding supports glob patterns (e.g., `AWS_*`)
- [ ] **CFG-11**: Pre-launch validation checks required env vars are set before Docker build starts
- [ ] **CFG-12**: SSH agent forwarding mounts socket when `forward_ssh_agent = true`
- [ ] **CFG-13**: Auth config uses copy-on-start strategy (writable copy in container, host preserved)
- [ ] **CFG-14**: `.env` file support via `auth.env_file` config key
- [ ] **CFG-15**: Proxy support (http_proxy, https_proxy, no_proxy) with host auto-detection
- [ ] **CFG-16**: Port forwarding from config (`ports = ["3000:3000"]`)
- [ ] **CFG-17**: Resource limits from config (memory, cpus, pids_limit)
- [ ] **CFG-18**: Hooks support (pre_build, post_stop shell commands)
- [x] **CFG-19**: Extra mounts default to read-only, require explicit `:rw` for write
- [ ] **CFG-20**: Persistent home volume via named Docker volume (survives container recreation)

### Harness System

- [ ] **HAR-01**: Harness interface defines identity, installation, runtime, dependencies, shell, lifecycle methods
- [ ] **HAR-02**: BaseHarness provides default implementations for optional methods
- [ ] **HAR-03**: Factory registry maps harness names to constructors
- [ ] **HAR-04**: `claude-code` harness is fully implemented with install, health check, env vars, config dir
- [ ] **HAR-05**: `opencode`, `gemini-cli`, `aider`, `codex-cli` harnesses return descriptive "not yet implemented" errors
- [ ] **HAR-06**: `custom` harness supports install_commands, entrypoint_command, config_dirs, required_env, health_check, aliases, shell_rc
- [ ] **HAR-07**: Each harness validates only its supported config keys; cross-harness keys produce specific errors
- [ ] **HAR-08**: HarnessConfig is a typed struct (not map[string]interface{})
- [ ] **HAR-09**: `skip_permissions` for claude-code defaults to false with security warning in init wizard
- [ ] **HAR-10**: `--prompt`/`-p` flag translates to harness-appropriate prompt flag automatically

### Docker Management

- [x] **DOC-01**: Dockerfile generated from Go text/template with go:embed
- [ ] **DOC-02**: Entrypoint script generated from template with `exec` for proper PID 1 signal handling
- [ ] **DOC-03**: Shell RC file generated from template with aliases, prompt, welcome message
- [ ] **DOC-04**: Non-root `zone` user created with UID matching host user
- [ ] **DOC-05**: Sudo scoped to package managers only (apt-get, pip, npm)
- [x] **DOC-06**: Container created with `no-new-privileges`, capability dropping, pids limit
- [x] **DOC-07**: Deterministic container naming from repo absolute path (hash-based)
- [ ] **DOC-08**: Docker labels applied for discovery by `zone ls`
- [ ] **DOC-09**: Idempotent launch: reattach if running, handle paused/exited/dead/stale states
- [ ] **DOC-10**: Config change detection warns user to restart when config hash differs
- [ ] **DOC-11**: Docker SDK used for build/create/start/stop/inspect; context propagation for graceful cancel
- [ ] **DOC-12**: Build progress streamed from Docker SDK with proper response body cleanup
- [ ] **DOC-13**: Git safe.directory configured in entrypoint for workspace mount
- [ ] **DOC-14**: Git user.name and user.email forwarded from host
- [ ] **DOC-15**: macOS username symlink compatibility in Dockerfile
- [ ] **DOC-16**: Root UID detection skips user creation (CI environments)

### Network Sandboxing

- [ ] **NET-01**: Whitelist mode: deny all outbound, allow specific hostnames via iptables
- [ ] **NET-02**: Blocklist mode: allow all outbound, deny specific hostnames
- [ ] **NET-03**: Mode "none" applies no network restrictions (default)
- [ ] **NET-04**: Each container gets its own Docker bridge network
- [ ] **NET-05**: Host-side iptables rules tagged with comments for identification and cleanup
- [ ] **NET-06**: Rules refreshed periodically (every 5 min) by re-resolving hostnames
- [ ] **NET-07**: IPv6 disabled on container network to prevent bypass
- [ ] **NET-08**: Deny list takes priority over allow list in whitelist mode
- [ ] **NET-09**: macOS warns that network filtering is unavailable and falls back to mode=none
- [ ] **NET-10**: Firewall rules cached in .zone/firewall.rules for inspectability
- [ ] **NET-11**: `sudo iptables` used only for firewall commands; fallback to none if sudo unavailable
- [ ] **NET-12**: Hostname glob matching for network rules (e.g., `*.anthropic.com`)

### TUI

- [ ] **TUI-01**: Init wizard with BubbleTea interactive harness selection and config preview
- [ ] **TUI-02**: Build progress display with Docker build log streaming
- [ ] **TUI-03**: Status view with live container state, uptime, ports, resources
- [ ] **TUI-04**: Log viewer with follow mode and build log option
- [ ] **TUI-05**: TTY auto-detection: BubbleTea when TTY, plain text when not
- [ ] **TUI-06**: `--plain` flag force-disables TUI even in TTY
- [ ] **TUI-07**: Non-TTY `zone init` without `--harness` errors with helpful message

### Cache & Build

- [x] **CAC-01**: .zone/ directory stores config hash, Dockerfile, entrypoint, image/container/network IDs
- [x] **CAC-02**: Cache hash includes merged config + templates + Zone version for automatic invalidation
- [x] **CAC-03**: File-based locking via flock for concurrent access protection
- [x] **CAC-04**: Lock contention produces error with exit code 5
- [x] **CAC-05**: `zone init` and `zone launch` add .zone/ to .gitignore
- [x] **CAC-06**: Build logs stored in .zone/logs/last_build.log

### Developer Experience

- [ ] **DX-01**: Structured exit codes: 0 success, 1 generic, 2 config, 3 docker, 4 network, 5 cache, 6 no container
- [ ] **DX-02**: All error messages include remediation hints
- [ ] **DX-03**: `--json` flag on status, ls, config, logs for machine-readable output
- [ ] **DX-04**: Signal handling: Ctrl+C sends SIGINT to harness, container stays alive
- [ ] **DX-05**: Context propagation: all Docker SDK calls take context for graceful cancellation
- [ ] **DX-06**: Harness process exit causes container stop; zone launch returns exit code 0
- [ ] **DX-07**: `zone stop` cleanup: stop container, remove container, remove network, clear IDs from cache
- [ ] **DX-08**: Command aliases: launch/up, stop/down, ls/list, logs/log, status/st
- [ ] **DX-09**: Help text with 2-4 usage examples per command
- [x] **DX-10**: GoReleaser configuration for binary distribution

## v2 Requirements

### Network

- **NET-V2-01**: DNS proxy sidecar for cross-platform hostname-level filtering
- **NET-V2-02**: macOS network filtering via DNS proxy sidecar

### Config

- **CFG-V2-01**: `zone migrate` command for config schema upgrades
- **CFG-V2-02**: Advanced glob patterns (`**`) via gobwas/glob

### Harnesses

- **HAR-V2-01**: Full implementation of opencode harness
- **HAR-V2-02**: Full implementation of gemini-cli harness
- **HAR-V2-03**: Full implementation of aider harness
- **HAR-V2-04**: Full implementation of codex-cli harness

## Out of Scope

| Feature | Reason |
|---------|--------|
| GUI / desktop interface | CLI-only tool; target users are terminal developers |
| Telemetry / analytics | Explicitly excluded by spec; trust is the product |
| In-container network filtering | Agent can disable own firewall; host-side enforcement is correct |
| Docker socket mount by default | Equivalent to root on host; security anti-pattern |
| Mobile interface | CLI tool distributed as Go binary |
| SSH key copying into container | Keys persist in image layers; use SSH agent forwarding instead |
| Docker-in-Docker | Security boundary collapse; explicit opt-in only via extra_mounts |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| CLI-01 | Phase 8 | Pending |
| CLI-02 | Phase 8 | Pending |
| CLI-03 | Phase 6 | Pending |
| CLI-04 | Phase 6 | Pending |
| CLI-05 | Phase 6 | Pending |
| CLI-06 | Phase 6 | Pending |
| CLI-07 | Phase 6 | Pending |
| CLI-08 | Phase 6 | Pending |
| CLI-09 | Phase 6 | Pending |
| CLI-10 | Phase 6 | Pending |
| CLI-11 | Phase 6 | Pending |
| CLI-12 | Phase 8 | Pending |
| CLI-13 | Phase 8 | Pending |
| CLI-14 | Phase 8 | Pending |
| CLI-15 | Phase 6 | Pending |
| CLI-16 | Phase 6 | Pending |
| CLI-17 | Phase 8 | Pending |
| CLI-18 | Phase 8 | Pending |
| CLI-19 | Phase 8 | Pending |
| CLI-20 | Phase 8 | Pending |
| CLI-21 | Phase 8 | Pending |
| CFG-01 | Phase 2 | Complete |
| CFG-02 | Phase 2 | Complete |
| CFG-03 | Phase 2 | Complete |
| CFG-04 | Phase 2 | Complete |
| CFG-05 | Phase 2 | Complete |
| CFG-06 | Phase 2 | Complete |
| CFG-07 | Phase 2 | Complete |
| CFG-08 | Phase 2 | Complete |
| CFG-09 | Phase 2 | Complete |
| CFG-10 | Phase 7 | Pending |
| CFG-11 | Phase 7 | Pending |
| CFG-12 | Phase 7 | Pending |
| CFG-13 | Phase 7 | Pending |
| CFG-14 | Phase 7 | Pending |
| CFG-15 | Phase 7 | Pending |
| CFG-16 | Phase 7 | Pending |
| CFG-17 | Phase 7 | Pending |
| CFG-18 | Phase 7 | Pending |
| CFG-19 | Phase 2 | Complete |
| CFG-20 | Phase 6 | Pending |
| HAR-01 | Phase 5 | Pending |
| HAR-02 | Phase 5 | Pending |
| HAR-03 | Phase 5 | Pending |
| HAR-04 | Phase 5 | Pending |
| HAR-05 | Phase 5 | Pending |
| HAR-06 | Phase 5 | Pending |
| HAR-07 | Phase 5 | Pending |
| HAR-08 | Phase 5 | Pending |
| HAR-09 | Phase 5 | Pending |
| HAR-10 | Phase 5 | Pending |
| DOC-01 | Phase 4 | Complete |
| DOC-02 | Phase 4 | Pending |
| DOC-03 | Phase 4 | Pending |
| DOC-04 | Phase 4 | Pending |
| DOC-05 | Phase 4 | Pending |
| DOC-06 | Phase 4 | Complete |
| DOC-07 | Phase 4 | Complete |
| DOC-08 | Phase 6 | Pending |
| DOC-09 | Phase 6 | Pending |
| DOC-10 | Phase 6 | Pending |
| DOC-11 | Phase 6 | Pending |
| DOC-12 | Phase 6 | Pending |
| DOC-13 | Phase 4 | Pending |
| DOC-14 | Phase 4 | Pending |
| DOC-15 | Phase 4 | Pending |
| DOC-16 | Phase 4 | Pending |
| NET-01 | Phase 10 | Pending |
| NET-02 | Phase 10 | Pending |
| NET-03 | Phase 10 | Pending |
| NET-04 | Phase 10 | Pending |
| NET-05 | Phase 10 | Pending |
| NET-06 | Phase 10 | Pending |
| NET-07 | Phase 10 | Pending |
| NET-08 | Phase 10 | Pending |
| NET-09 | Phase 10 | Pending |
| NET-10 | Phase 10 | Pending |
| NET-11 | Phase 10 | Pending |
| NET-12 | Phase 10 | Pending |
| TUI-01 | Phase 9 | Pending |
| TUI-02 | Phase 9 | Pending |
| TUI-03 | Phase 9 | Pending |
| TUI-04 | Phase 9 | Pending |
| TUI-05 | Phase 9 | Pending |
| TUI-06 | Phase 9 | Pending |
| TUI-07 | Phase 9 | Pending |
| CAC-01 | Phase 3 | Complete |
| CAC-02 | Phase 3 | Complete |
| CAC-03 | Phase 3 | Complete |
| CAC-04 | Phase 3 | Complete |
| CAC-05 | Phase 3 | Complete |
| CAC-06 | Phase 3 | Complete |
| DX-01 | Phase 8 | Pending |
| DX-02 | Phase 8 | Pending |
| DX-03 | Phase 8 | Pending |
| DX-04 | Phase 8 | Pending |
| DX-05 | Phase 8 | Pending |
| DX-06 | Phase 8 | Pending |
| DX-07 | Phase 8 | Pending |
| DX-08 | Phase 8 | Pending |
| DX-09 | Phase 8 | Pending |
| DX-10 | Phase 1 | Complete |

**Coverage:**
- v1 requirements: 102 total
- Mapped to phases: 102
- Unmapped: 0

---
*Requirements defined: 2026-03-26*
*Last updated: 2026-03-26 after roadmap creation — traceability populated*
