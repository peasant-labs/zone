# Project Research Summary

**Project:** Zone — Go CLI Docker workspace manager for LLM coding agents
**Domain:** Developer tooling / container lifecycle management
**Researched:** 2026-03-26
**Confidence:** HIGH

## Executive Summary

Zone is a Go CLI tool that provisions and manages Docker-based workspaces for LLM coding agents (Claude Code, Aider, OpenCode, etc.). The core value proposition is zero-friction, single-command workspace launch with hardened security defaults — an LLM agent inside a Zone container cannot exfiltrate data, escalate privileges, or corrupt the host environment. Research across the competitor landscape (Docker Sandboxes, claude-code-sandbox, devcontainers, OpenShell) confirms Zone's differentiating angle: host-side network enforcement, harness plugin architecture, and config-hash-based idempotency are absent from every competing tool.

The recommended approach is a layered Go architecture built strictly around a dependency graph enforced by the compiler: config and cache packages form the foundation, harness and network packages build on top, docker/manager sits above them as the central orchestrator, and cmd/ files are deliberately thin wiring with no business logic. The BubbleTea TUI stack (v2) should be treated as an optional layer that is bypassed entirely in non-TTY environments — not as a core dependency. The entire MVP (v1) can and should ship before adding TUI, network sandboxing, or GoReleaser distribution.

The dominant risks are correctness risks, not architectural uncertainty. Thirteen concrete pitfalls are documented with verified prevention strategies: terminal raw mode must be restored on every exit path, Docker SDK response bodies must be fully drained and closed, PID 1 signal propagation requires `exec` in entrypoints, and iptables rules must be tagged and audited on every startup to survive process crashes. These are all avoidable with deliberate test coverage designed specifically to exercise the crash/cancellation paths, not just happy-path flows.

## Key Findings

### Recommended Stack

The stack is fully specified and verified against package release pages. Go 1.23+ with Cobra v1.10.2 handles the CLI layer; BubbleTea v2.0.0 + Lip Gloss v2.0.0 + Bubbles v2.0.0 (all released together Feb 23, 2026) handle the TUI layer; github.com/docker/docker/client v28.5.2 handles the Docker SDK layer; BurntSushi/toml v1.5.0 handles strict config parsing. All three Charm v2 libraries must be upgraded together — they cannot mix with v1. The new vanity import path is `charm.land/*` not `github.com/charmbracelet/*`.

**Core technologies:**
- Go 1.23+ with Cobra v1.10.2 — CLI framework; de facto standard (used by kubectl, docker CLI, gh)
- charm.land/bubbletea/v2@v2.0.0 — TUI framework; Elm architecture, "Cursed Renderer" (10x faster in v2)
- charm.land/lipgloss/v2@v2.0.0 + charm.land/bubbles/v2@v2.0.0 — TUI styling and components; must match bubbletea major version
- github.com/docker/docker/client@v28.5.2 — official Docker SDK; used by docker CLI itself
- github.com/BurntSushi/toml@v1.5.0 — strict TOML decoding with `DecodeStrict()` for unknown-key rejection
- github.com/coreos/go-iptables@v0.8.0 — host-side iptables wrapper; Linux-only, build-tag guarded
- github.com/gofrs/flock@v0.13.0 — advisory file locking for .zone/ cache directory
- github.com/agnivade/levenshtein@v1.2.0 — edit-distance for "did you mean X?" config key suggestions
- GoReleaser v2.14.3 — binary distribution; builds for linux/darwin × amd64/arm64; generates Homebrew tap

**Critical version notes:** Do not start on BubbleTea v1 (`github.com/charmbracelet/bubbletea`) — v2 is stable and the v1 import path is obsolete. Do not use Viper — BurntSushi/toml strict decode is the right primitive for Zone's two-file config. Defer nftables support to v2.

### Expected Features

**Must have (table stakes — v1 launch):**
- `zone launch` with idempotent lifecycle (build, reattach if running, rebuild on config change)
- Project-only filesystem mount + persistent named home volume
- Two-tier TOML config (global + per-repo) with strict validation and edit-distance suggestions
- Claude Code harness (fully implemented) + custom harness via `install_commands`/`entrypoint_command`
- Environment variable forwarding with pre-launch validation of required vars
- SSH agent forwarding via socket mount (keys never written to container)
- Auth config copy-on-start to writable container path (not read-only mount)
- Container security hardening: `cap-drop=ALL`, `no-new-privileges`, scoped sudo, pids limit
- Container lifecycle commands: stop, restart, destroy, ls, logs, status, clean
- `.zone/` cache with config hash, image ID, container ID, file lock
- Deterministic naming from repo path + Docker labels
- Structured exit codes (0–6) + signal handling (SIGINT, SIGTERM)
- `--json` output on status, ls, config, logs
- `--plain` flag + auto-detect non-TTY fallback

**Should have (v1.x, add after validation):**
- BubbleTea TUI: init wizard, build progress, status view, log viewer
- Host-side iptables network sandboxing (Linux only)
- Lifecycle hooks: `pre_build`, `post_stop`
- Port forwarding and resource limits (memory, CPU, pids) in config
- Proxy support with auto-detection
- GoReleaser binary distribution (Homebrew tap + prebuilt binaries)

**Defer (v2+):**
- macOS network filtering (DNS proxy sidecar — significant infrastructure)
- Additional fully-implemented harnesses beyond claude-code (Aider, OpenCode, Codex, Gemini CLI)
- `zone migrate` config schema upgrade command
- Multi-machine or remote workspace support

### Architecture Approach

Zone follows a strict layered architecture where the compiler enforces the dependency graph. `cmd/` files are thin Cobra wiring only — no business logic. `internal/docker/manager.go` is the central orchestrator and the only package that depends on config, cache, harness, network, and templates together. TUI models in `internal/tui/` are pure BubbleTea — they receive plain data types from the cmd layer and never touch Docker. `pkg/templates/` uses `//go:embed` for compile-time template bundling — no runtime file paths, no CWD dependency.

**Major components:**
1. `cmd/` — Cobra command definitions; flag parsing, TTY detection, signal setup, exit code mapping
2. `internal/config/` — TOML strict decode, two-tier merge, semantic validation, edit-distance suggestions
3. `internal/cache/` — `.zone/` directory state (config hash, image ID, container ID, flock)
4. `internal/harness/` — Harness interface, registry, BaseHarness, claude-code impl, custom harness
5. `internal/docker/` — Docker SDK calls; central orchestrator; Dockerfile/entrypoint template rendering
6. `internal/network/` — Host-side iptables rule generation and cleanup (Linux-only, build-tagged)
7. `internal/tui/` — BubbleTea models for init wizard, build progress, status, log viewer
8. `pkg/templates/` — Embedded Dockerfile, entrypoint, bashrc templates

**Key pattern — SDK for lifecycle, CLI exec for interactive TTY:** Use the Docker SDK for build/create/start/stop/inspect/list. Use `os/exec docker exec -it` for interactive terminal attachment. The SDK's `ContainerExecAttach` hijacked connection handles SIGWINCH and signal forwarding poorly — shell out for interactive sessions.

### Critical Pitfalls

1. **Terminal raw mode not restored on crash** — Defer `terminal.Restore(oldState, os.Stdin.Fd())` as the very first statement in any attach/exec function; wrap goroutines in `recover()`. Test by killing the container mid-session.

2. **Docker SDK response bodies left unclosed** — Always `defer func() { io.Copy(io.Discard, resp.Body); resp.Body.Close() }()` immediately after any SDK call returning a body. For `HijackedResponse`, call both `.Close()` AND `.CloseWrite()`. Add `goleak` tests.

3. **PID 1 does not propagate signals** — Every generated entrypoint must end with `exec harness-binary "$@"` (not bare `harness-binary`), replacing the shell with the harness as PID 1. Alternatively, use `tini`/`dumb-init` as ENTRYPOINT prefix.

4. **iptables rules survive process crash** — Tag every rule with `--comment "zone-<container-id>-<rule-hash>"`. On startup, audit DOCKER-USER chain and remove stale Zone-tagged rules for containers that no longer exist. Always use `-C` (check) before `-A` (append). Only write to the DOCKER-USER chain — never INPUT, never FORWARD directly.

5. **Stopped container owns its name** — Before `ContainerCreate`, always call `ContainerInspect`. If the container exists and is stopped, call `ContainerStart`. If config hash changed, `ContainerRemove` then `ContainerCreate`. Never surface the "name already in use" error raw.

6. **Stale file lock after SIGKILL** — Use `flock.TryLock()` (non-blocking). On failure, check if the lock holder PID is alive with `proc.Signal(0)`. If dead, remove stale lock and proceed. Include `--force` override.

7. **iptables rules in wrong chain** — All egress rules go in DOCKER-USER chain (FORWARD table), not INPUT. INPUT has zero effect on container traffic which is NAT'd through the bridge.

8. **Auth config credentials in container writable layer** — Copy auth files to tmpfs mounts inside the container, never to the regular filesystem. Credentials in the writable layer survive `docker export`.

## Implications for Roadmap

Based on the component build order from ARCHITECTURE.md and the feature dependency graph from FEATURES.md, a 5-phase structure is recommended:

### Phase 1: Foundation (Config, Cache, Template System)
**Rationale:** All other packages depend on `MergedConfig` from `internal/config/`. Cache and templates have no upstream dependencies. Building these first means every subsequent phase has a stable foundation and the config hash (which gates image rebuilds) is correct from day one.
**Delivers:** Two-tier TOML config with strict decode + edit-distance suggestions; `.zone/` cache with flock; embedded template system; deterministic container naming.
**Addresses features:** Two-tier config, strict validation, config hash, .zone/ cache, deterministic naming, `zone validate` command
**Avoids pitfalls:** TOML merge semantic validation (run validation on merged struct, not per-file); stale file lock (implement TryLock + staleness check here); go:embed CWD sensitivity (test distributed binary from /tmp)
**Research flag:** STANDARD — well-documented Go patterns; BurntSushi/toml strict decode is straightforward

### Phase 2: Docker Manager + Container Lifecycle
**Rationale:** Docker manager (`internal/docker/`) is the central orchestrator but depends on config and cache. This phase delivers the core value proposition: idempotent `zone launch`. No TUI yet — plain text output is sufficient for validation.
**Delivers:** Full idempotent launch lifecycle (build if needed, reattach if running, rebuild on hash change); stop/restart/destroy/ls/status/logs/clean commands; container security hardening; SSH agent forwarding; env var forwarding; auth config copy-on-start; persistent home volume; structured exit codes; `--json` output; `--plain` flag + non-TTY auto-detect.
**Uses:** docker/docker/client SDK, gofrs/flock, pkg/templates, go-iptables (stub on macOS)
**Avoids pitfalls:** Terminal raw mode (defer restore before spawning IO goroutines); SDK response body leaks (drain + close on every SDK call with bodies); PID 1 signal propagation (exec in entrypoint template); stopped container name collision (always ContainerInspect before ContainerCreate); Docker socket path on macOS (probe multiple paths after FromEnv fails); blocking iptables exec (always use context with timeout)
**Research flag:** STANDARD for most patterns; NEEDS RESEARCH for interactive TTY attach edge cases (SIGWINCH handling, signal forwarding via `docker exec -it` vs SDK)

### Phase 3: Harness Plugin System
**Rationale:** Harness interface can be developed in parallel with Phase 2 once `config/types.go` defines `HarnessConfig`. Claude Code harness is the primary user. Custom harness via config is the escape hatch for all other agents. Typed `HarnessConfig` (union struct) not `map[string]interface{}` is required for validation — this must be decided before any harness code is written.
**Delivers:** Harness interface + registry; BaseHarness; claude-code fully implemented; custom harness via `install_commands`/`entrypoint_command`; per-harness field validation; harness-aware Dockerfile generation
**Avoids pitfalls:** Template injection via user-supplied harness name (sanitize all user values before templating); typed HarnessConfig prevents silent misconfiguration; auth config leakage (copy to tmpfs, not regular filesystem)
**Research flag:** STANDARD — spec defines the interface clearly; patterns are well-established

### Phase 4: TUI Layer (BubbleTea)
**Rationale:** TUI is a P2 feature per FEATURES.md — add after the core lifecycle is validated. Building TUI after the domain layer ensures TUI models receive plain data types, never Docker types, making the plain-text fallback a first-class path rather than an afterthought.
**Delivers:** Init wizard (harness selection + config preview); build progress (streaming Docker output); status view (live container state); log viewer (follow mode)
**Uses:** charm.land/bubbletea/v2, charm.land/lipgloss/v2, charm.land/bubbles/v2
**Avoids pitfalls:** BubbleTea panic corrupts terminal (wrap all Cmds in recover(); defer terminal.Restore in main()); TTY detection failure (check both stdout and stdin; respect CI=true, NO_COLOR, ZONE_PLAIN=1); BubbleTea + Docker streaming (drain background goroutines before quit; use done channel)
**Research flag:** NEEDS RESEARCH — BubbleTea v2 is newly stable (Feb 2026); Cobra + BubbleTea v2 integration patterns need verification; the "Cursed Renderer" API may differ from v1 documentation

### Phase 5: Network Sandboxing + Distribution
**Rationale:** Network sandboxing is high complexity and Linux-only. Deferring it allows the core product to be validated first. GoReleaser distribution and lifecycle hooks are P2 features that add polish but are not required for the core value proposition.
**Delivers:** Host-side iptables network sandboxing (DOCKER-USER chain, whitelist/blocklist modes); lifecycle hooks (pre_build, post_stop); port forwarding and resource limits; proxy auto-detection; GoReleaser binary distribution (Homebrew tap + prebuilt binaries for linux/darwin × amd64/arm64)
**Avoids pitfalls:** iptables in wrong chain (DOCKER-USER only, never INPUT); orphaned rules after crash (tag + audit on startup; zone clean wipes all Zone-tagged rules); iptables rule deduplication (-C before -A); macOS stub (coreos/go-iptables guarded by build tag `//go:build linux`)
**Research flag:** NEEDS RESEARCH — Docker's iptables interaction with nftables backends on recent Debian/Ubuntu needs testing; Docker Engine 29+ nftables experimental mode may affect rule placement

### Phase Ordering Rationale

- Config must precede everything — it's the only shared data type (`MergedConfig`) consumed by all other packages
- Cache must precede docker/manager — the manager needs the config hash to determine if a rebuild is needed
- Templates and harness can develop in parallel with cache once `config/types.go` is stable
- Docker manager is the longest phase — it delivers the entire MVP minus TUI and network sandboxing
- TUI is explicitly deferred to after lifecycle validation per FEATURES.md's v1/v1.x split
- Network sandboxing is deferred last because it is Linux-only, high complexity, and requires a running container to test
- The architecture's enforced import graph (`internal/* → cmd/*` is FORBIDDEN) makes this phase order a hard constraint, not just a preference

### Research Flags

Phases needing deeper research during planning:
- **Phase 4 (TUI):** BubbleTea v2.0.0 released Feb 23, 2026 — only one month old. Cobra integration patterns, the "Cursed Renderer" API differences from v1, and SIGWINCH handling in v2 should be verified before implementation. The vanity import path `charm.land/*` may have proxy/GOPROXY implications.
- **Phase 5 (Network Sandboxing):** Docker + nftables interaction on Linux distros where iptables is nftables-backed (recent Debian, Ubuntu, Fedora) needs integration testing. The DOCKER-USER chain behavior under Docker Engine 29+ nftables experimental mode is not fully documented.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Foundation):** BurntSushi/toml strict decode, gofrs/flock, and go:embed are well-documented with high-confidence sources.
- **Phase 2 (Docker Manager):** Docker SDK patterns are documented; the `os/exec docker exec -it` interactive attach pattern is the established workaround for SDK TTY limitations.
- **Phase 3 (Harness):** Interface + registry pattern is a Go standard; the spec defines the interface contract precisely.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All versions verified against pkg.go.dev and official release pages; BubbleTea v2 sourced from official Charm blog post |
| Features | HIGH | Cross-verified against 6+ competitor tools; table stakes vs differentiators distinction is clear |
| Architecture | HIGH | Spec provides authoritative project structure; import graph constraints are explicit; patterns verified against official SDK docs |
| Pitfalls | HIGH | 13 pitfalls with sources; each has a verified reproduction path and prevention strategy; several backed by filed GitHub issues and CVEs |

**Overall confidence:** HIGH

### Gaps to Address

- **BubbleTea v2 Cobra integration:** The v2 "Cursed Renderer" uses ncurses internally — behavior in `zone exec` and other passthrough contexts is undocumented. Verify before Phase 4 implementation.
- **nftables / iptables coexistence:** On systems where `iptables` is symlinked to `iptables-nft`, coreos/go-iptables still works but rule visibility in `iptables -L` may differ from `nft list ruleset`. Needs integration test before Phase 5.
- **macOS SSH_AUTH_SOCK domain socket:** macOS domain sockets cannot be bind-mounted into containers. SSH agent forwarding on macOS requires either `socat` bridging or an explicit user warning. The spec defers this but Zone should surface a clear error rather than silently failing.
- **Rootless Docker support:** Zone's iptables approach requires root or `CAP_NET_ADMIN` on the host — incompatible with rootless Docker mode. The spec does not address this. Surface a clear error when `docker info` shows rootless mode.

## Sources

### Primary (HIGH confidence)
- charm.land/blog/v2/ — Official Charm announcement of BubbleTea/Lip Gloss/Bubbles v2.0.0 stable (Feb 23, 2026)
- pkg.go.dev/github.com/docker/docker/client — Docker SDK v28.5.2, verified Nov 5, 2025
- goreleaser.com/blog/goreleaser-v2.14/ — GoReleaser v2.14.3, Mar 9, 2026
- github.com/spf13/cobra/releases — Cobra v1.10.2, Dec 2025
- docs.docker.com/engine/network/packet-filtering-firewalls/ — DOCKER-USER chain behavior
- Zone spec v4.0 (/workspace/zone/zone-spec.md) — project structure, import graph, command definitions

### Secondary (MEDIUM confidence)
- github.com/charmbracelet/bubbletea/releases/tag/v2.0.0 — BubbleTea v2.0.0 release notes
- pkg.go.dev/github.com/coreos/go-iptables/iptables — v0.8.0, Aug 2024 (may not be latest)
- elewis.dev/charming-cobras-with-bubbletea-part-1 — Cobra + BubbleTea TTY handoff pattern
- petermalmgren.com/signal-handling-docker/ — PID 1 signal propagation in Docker
- ntk148v.github.io/posts/docker-iptables/ — DOCKER-USER chain correctness
- leg100.github.io/en/posts/building-bubbletea-programs/ — BubbleTea Cmd panic recovery
- addshore.com/2021/01/go-docker-sdk-raw-terminal-ctrlc-handling/ — Docker SDK raw terminal handling

### Tertiary (for awareness)
- CVE-2025-64329 — containerd CRI attach goroutine leak (validates SDK body-drain requirement)
- github.com/moby/moby/issues/42029 — iptables forwarding rules not cleaned on container removal

---
*Research completed: 2026-03-26*
*Ready for roadmap: yes*
