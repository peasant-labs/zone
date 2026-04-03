# Zone

## What This Is

Zone is a Go CLI tool that generates and manages Docker workspaces for LLM coding agents (Claude Code, OpenCode, Gemini CLI, Aider, Codex CLI, and custom harnesses). Users run `zone launch` in any repo to get a sandboxed Docker container preconfigured for their chosen AI coding tool, with zero manual Docker configuration.

## Core Value

Run `zone launch` in any repo and get a sandboxed Docker workspace for your LLM coding agent, with zero manual Docker configuration.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] CLI with 14 commands (init, launch, join, exec, shell, build, stop, restart, ls, logs, clean, destroy, status, config, validate) built with Cobra
- [ ] Two-tier TOML config (global ~/.config/zone/config.toml + per-repo zone.toml) with strict decoding and merge strategy
- [ ] Harness plugin architecture with interface, registry, and BaseHarness — claude-code fully implemented, others stubbed
- [ ] Custom harness support via install_commands, entrypoint_command, and config_dirs
- [ ] Docker image generation from Go text/templates (Dockerfile, entrypoint, shell RC) with go:embed
- [ ] Idempotent container lifecycle (build-if-needed, reattach-if-running, clean-restart on config change)
- [ ] .zone/ cache directory with config hash, image ID, container ID, file lock, build logs
- [ ] BubbleTea TUI for init wizard, build progress, status view, and log viewer
- [ ] Plain text fallback when not a TTY (--plain flag, auto-detected)
- [ ] Network sandboxing via host-side iptables (whitelist/blocklist modes, Linux-only in v1)
- [ ] Environment variable forwarding with glob support and pre-launch validation
- [ ] SSH agent forwarding (mount socket, not keys)
- [ ] Auth config mount via copy-on-start strategy (writable copy in container)
- [ ] Config validation: unknown key detection with edit-distance suggestions, dangerous mount blocking with symlink resolution
- [ ] Deterministic container/network naming from repo path, Docker labels for discovery
- [ ] Container security hardening (no-new-privileges, capability dropping, scoped sudo, pids limit)
- [ ] Structured exit codes (0-6) mapping to error categories
- [ ] Context and signal handling (graceful Ctrl+C, SIGTERM propagation)
- [ ] --json output for scriptability on status, ls, config, logs
- [ ] Port forwarding and resource limits (memory, CPU, pids) from config
- [ ] Persistent home volume (survives container recreation)
- [ ] Hooks (pre_build, post_stop shell commands)
- [ ] Proxy support (http_proxy, https_proxy, no_proxy) with auto-detection
- [ ] GoReleaser binary distribution

### Out of Scope

- DNS proxy sidecar for cross-platform network filtering — Phase 2 feature per spec
- macOS network filtering — requires DNS proxy sidecar, Linux-only in v1
- `zone migrate` command for config schema upgrades — future feature
- Mobile or GUI interfaces — CLI-only tool
- Telemetry or analytics — explicitly excluded by spec

## Context

- Project lives in /workspace/zone/
- Spec document: zone/zone-spec.md (v4.0) — the authoritative reference for all implementation details
- Go module path: zone (local)
- Distribution: go install, Homebrew tap, prebuilt binaries via GoReleaser
- Target platforms: Linux + macOS (network filtering Linux-only in v1)
- The spec includes complete Go code examples for key components (harness interface, Docker manager, naming, templates, error handling, signal handling)
- Project structure is fully specified in Section 7 of the spec

## Constraints

- **Tech stack**: Go with Cobra, BubbleTea, Lip Gloss, BurntSushi/toml, Docker SDK — all specified in spec
- **Security**: Host-side network enforcement only (no CAP_NET_ADMIN in container), scoped sudo, dangerous mount blocking
- **Compatibility**: Linux + macOS for core functionality; network filtering Linux-only in v1
- **Privacy**: No telemetry, no analytics, all operations local

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Host-side iptables for network filtering | In-container CAP_NET_ADMIN would let LLM agent disable firewall | — Pending |
| Copy-on-start for auth config (not read-only mount) | Harnesses need to write to their own config dirs | — Pending |
| Scoped sudo (package managers only) | Prevent LLM agent privilege escalation while allowing runtime installs | — Pending |
| Claude-code fully implemented, others stubbed | Focus v1 on one harness, custom harness covers the rest | — Pending |
| BurntSushi/toml strict decoding | Catch typos early with edit-distance suggestions | — Pending |
| Typed HarnessConfig struct (not map[string]interface{}) | Type safety, per-harness validation of allowed fields | — Pending |

---
*Last updated: 2026-04-03 after Phase 9 completion — BubbleTea TUI layer (init wizard, build progress, status view, log viewer) with TTY detection and --plain override*
