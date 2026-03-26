# Feature Research

**Domain:** Docker workspace manager CLI for LLM coding agents
**Researched:** 2026-03-26
**Confidence:** HIGH (cross-verified across multiple tools: Docker Sandboxes, devcontainers, claude-code-sandbox, OpenShell, sandbox-agent, DevPod)

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Single-command workspace launch | The entire value prop is zero-friction startup; anything requiring multi-step manual Docker setup defeats the purpose | LOW | `zone launch` is the core UX; idempotent — reattach if running, restart on config change |
| Filesystem isolation (project-only mount) | Every sandbox tool mounts only the project dir; full host access negates the security purpose | LOW | Docker handles this; the design decision is what to exclude (SSH keys, ~/.aws, etc.) |
| Container lifecycle management (start/stop/restart/destroy) | Users need to control container state; CLI must match what they expect from Docker UX | MEDIUM | Commands: launch, stop, restart, destroy, ls — with idempotent build-if-needed logic |
| Persistent home volume | Without this, agent-installed packages and config disappear on every container recreation, breaking long-running sessions | LOW | Named Docker volume scoped to repo path; survives container recreation |
| Environment variable forwarding | API keys (ANTHROPIC_API_KEY etc.) must reach the container; users will not tolerate manual re-entry per session | LOW | Glob support + pre-launch validation that required vars are set |
| Auth/credential forwarding to agent | Claude Code, Aider, etc. need their auth config to authenticate; missing this = unusable | MEDIUM | Copy-on-start strategy (writable copy); read-only mount breaks harnesses that write to config dirs |
| Git identity injection | Commits made inside container must be attributed to the user; missing this causes confusion in PR workflows | LOW | Inject git user.name, user.email from host config |
| SSH agent forwarding | Git operations over SSH (push/pull) require SSH access; copying keys is a security anti-pattern | LOW | Mount SSH_AUTH_SOCK socket, not key files — keys never written to container |
| Container status and listing | Users run multiple workspaces; need to see what's running and where | LOW | `zone ls`, `zone status` with container name, image, repo, uptime |
| Log access | Debugging agent output and build failures requires log visibility | LOW | `zone logs` streaming; separate build logs in .zone/ cache |
| Config file driven setup | Zero-manual-Docker-config means all settings in a declarative file, not CLI flags | MEDIUM | TOML format; two-tier (global + per-repo) merge strategy |
| Clean environment (standardized shell) | Custom user dotfiles/aliases cause agent misinterpretation; agents need predictable shell state | LOW | Zone controls RC files via templates; no host dotfiles leaked in |
| Build output visibility | Docker image builds are slow; users abandon tools that show a blank screen during builds | LOW | Progress display with log streaming; spinner + step indicators minimum |

### Differentiators (Competitive Advantage)

Features that set the product apart. Not required, but valued.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Harness plugin architecture | Supports any LLM coding agent without forking; custom harnesses via config | HIGH | Interface + registry + BaseHarness; claude-code fully implemented, others stubbed; `install_commands` + `entrypoint_command` for custom tools |
| Host-side iptables network sandboxing | Agent cannot exfiltrate data or download malware even if compromised; in-container CAP_NET_ADMIN would let the agent disable its own firewall | HIGH | Whitelist/blocklist modes; Linux-only in v1; DNS sidecar for macOS deferred to Phase 2. Claude Code's bubblewrap/seatbelt approach is agent-owned; Zone's approach is host-owned and agent-proof |
| Config hash-based change detection | Automatic container rebuild on config change without user managing Docker manually; eliminates stale container issues | MEDIUM | SHA of merged config written to .zone/ cache; diff triggers clean rebuild |
| BubbleTea TUI with plain-text fallback | Interactive init wizard and build progress in TTY; graceful degradation to plain output in CI/scripts | HIGH | Auto-detect TTY; --plain flag; non-TTY defaults to plain. Daytona reports this as a major user request. Most tools lack this entirely |
| Strict TOML config validation with edit-distance suggestions | Typos in config are caught immediately with actionable "did you mean X?" messages, not silent silently wrong behavior | MEDIUM | Unknown key detection + Levenshtein distance suggestions; dangerous mount blocking with symlink resolution |
| Idempotent container lifecycle | `zone launch` always does the right thing: build if no image, reattach if running, rebuild on config change — no user decision tree | MEDIUM | Eliminates the most common Docker workflow friction: "is my container stale?" |
| Deterministic naming from repo path | Multiple repos, no naming collisions, no user-managed naming scheme; containers and networks discoverable by label | LOW | Hash or slug from absolute repo path; Docker labels for discovery |
| Container security hardening | Scoped sudo (package managers only), capability dropping, no-new-privileges, pids limit — LLM agent cannot escalate to root | MEDIUM | Prevents agent from disabling its own restrictions; most sandbox tools skip this |
| --json output mode | Scriptable status/ls/config/logs output enables CI integration, monitoring dashboards, external tooling | LOW | Structured exit codes (0-6) + JSON flag on key commands |
| Pre-launch env validation | Tells user exactly which required API keys are missing before wasting time starting a container | LOW | Validates globs resolve to set vars; reports missing keys with name |
| Hooks (pre_build, post_stop) | Enables project-specific automation without coupling to harness implementation | LOW | Shell commands in config; runs at defined lifecycle points |
| Proxy support with auto-detection | Respects corporate network proxies automatically; without this, enterprise users cannot function behind proxies | LOW | http_proxy/https_proxy/no_proxy with env auto-detection |
| Port forwarding and resource limits | Agents running web servers, databases need port access; resource limits prevent runaway processes from consuming host | LOW | Memory, CPU, pids limits in config; port mapping declarative |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| In-container network filtering (CAP_NET_ADMIN) | Simpler to implement — just run a proxy inside the container | The LLM agent running inside can disable its own firewall, defeating the security model entirely | Host-side iptables rules enforced outside the container; agent cannot touch them |
| Copying SSH private keys into the container | Enables git push/pull without complex socket forwarding | Keys written to container filesystem are accessible to the agent and persist in image layer history even after deletion | SSH agent forwarding via socket mount; keys never leave host memory |
| Read-only mount of auth config directories | Seems safe — agent can read but not corrupt credentials | Harnesses (Claude Code, Aider) write to their own config dirs during operation; read-only mounts cause runtime failures | Copy-on-start: writable copy inside container, original unchanged on host |
| Telemetry and analytics | Helps developers understand usage patterns, improve the tool | LLM coding agent workflows involve sensitive code, proprietary projects, API keys; users will not trust a tool that phones home | No telemetry, ever. Trust is the product |
| GUI or desktop interface | Lower barrier to entry, visual container management | The target user is a developer running agents in a terminal; a GUI adds a dependency (display server, framework) and breaks headless/CI use | Excellent TUI + --plain fallback covers all use cases including headless |
| Docker socket mount by default | Enables Docker-in-Docker for agent testing | Gives the agent full Docker daemon access — equivalent to root on the host; collapses the security boundary | Explicit opt-in only, documented as high-risk; default config excludes it |
| `map[string]interface{}` harness config | Flexible — any key works without code changes | No compile-time validation; typos in agent config silently pass; runtime errors far from the misconfiguration site | Typed HarnessConfig struct per harness; validation at config parse time |
| Automatic schema migration (`zone migrate`) | Users want zero-friction config upgrades | Schema migration is complex to implement correctly, easy to corrupt user configs, and rarely needed for a tool with stable config | Clear changelog + version field in config; manual migration with good error messages |
| Server-side component / daemon | Enables multi-machine management, central state | Adds operational burden (install, upgrade, secure, monitor a daemon); breaks the "runs only on your computer" trust model | Client-only; all state in .zone/ local to each repo + Docker labels for discovery |

## Feature Dependencies

```
Container Launch (zone launch)
    └──requires──> Image Build
                       └──requires──> Harness Plugin (dockerfile template)
                                          └──requires──> Harness Registry
    └──requires──> Config Loading (zone.toml + global config.toml)
                       └──requires──> Config Merge Strategy
                       └──requires──> Config Validation (unknown keys, dangerous mounts)
    └──requires──> .zone/ Cache (config hash, image ID, container ID)

Network Sandboxing (iptables)
    └──requires──> Container Launch (needs container network name)
    └──requires──> Linux host (iptables not available on macOS)
    └──conflicts──> macOS (requires DNS sidecar, deferred to v2)

SSH Agent Forwarding
    └──requires──> SSH_AUTH_SOCK set on host
    └──enhances──> Git operations inside container

Auth Config Forwarding
    └──requires──> Container Launch
    └──enhances──> Harness Plugin (knows which dirs to copy)

TUI (BubbleTea)
    └──requires──> TTY detection
    └──conflicts──> CI environments (auto-fallback to --plain)
    └──enhances──> Build progress, init wizard, status view

--json output
    └──enhances──> status, ls, config, logs commands
    └──requires──> Structured exit codes

Hooks (pre_build, post_stop)
    └──requires──> Container lifecycle events
    └──enhances──> Custom project automation

Config Hash Change Detection
    └──requires──> .zone/ cache directory
    └──enhances──> Idempotent lifecycle (triggers rebuild)

Port Forwarding + Resource Limits
    └──requires──> Container Launch (applied at container create time)
```

### Dependency Notes

- **Network sandboxing requires container launch:** iptables rules are keyed to Docker network name, which is determined at container creation by deterministic naming from repo path.
- **Auth config forwarding requires harness knowledge:** BaseHarness defines which config dirs to copy; without the harness plugin system, this is hardcoded per-tool.
- **TUI conflicts with CI/non-TTY environments:** auto-detection is critical; missing TTY check causes CI pipelines to hang or output garbage escape codes.
- **Config validation requires typed structs:** edit-distance suggestions on unknown keys only work if the set of valid keys is known at compile time; `map[string]interface{}` configs make this impossible.
- **Persistent home volume requires deterministic naming:** volume name derived from repo path; changing the naming scheme orphans existing volumes.

## MVP Definition

### Launch With (v1)

Minimum viable product — what's needed to validate the concept.

- [ ] `zone launch` with idempotent lifecycle (build, reattach, rebuild on change) — core value prop
- [ ] Project-only filesystem mount + persistent home volume — basic security + usability
- [ ] Two-tier TOML config (global + per-repo) with strict validation and edit-distance suggestions — zero silent misconfig
- [ ] Harness plugin for claude-code (fully implemented) + custom harness via config — covers primary user + escape hatch
- [ ] Environment variable forwarding with pre-launch validation — API keys must reach the agent
- [ ] SSH agent forwarding + auth config copy-on-start — git operations + agent authentication
- [ ] Container security hardening (no-new-privileges, cap drop, scoped sudo, pids limit) — non-negotiable for agent workloads
- [ ] Container lifecycle commands (stop, restart, ls, logs, destroy, status) — basic ops
- [ ] .zone/ cache with config hash, image ID, container ID, file lock — enables idempotent behavior
- [ ] Plain-text fallback when no TTY (--plain flag, auto-detect) — CI usability
- [ ] --json output on status, ls, config, logs — scriptability
- [ ] Deterministic naming from repo path + Docker labels — multi-workspace management
- [ ] Structured exit codes (0-6) + signal handling — scriptability and graceful shutdown

### Add After Validation (v1.x)

Features to add once core is working.

- [ ] BubbleTea TUI (init wizard, build progress, status view, log viewer) — trigger: user feedback that plain output is insufficient for onboarding
- [ ] Network sandboxing via host-side iptables (Linux) — trigger: user reports of agent exfiltration attempts or security audit requirement
- [ ] Hooks (pre_build, post_stop) — trigger: users building project-specific automation around zone
- [ ] Port forwarding and resource limits in config — trigger: agent workflows requiring web servers or hitting resource exhaustion
- [ ] Proxy support with auto-detection — trigger: first enterprise/corporate network user report
- [ ] GoReleaser binary distribution (Homebrew tap, prebuilt binaries) — trigger: users complaining about `go install` friction

### Future Consideration (v2+)

Features to defer until product-market fit is established.

- [ ] macOS network filtering (DNS proxy sidecar) — requires significant infrastructure; Linux-only is acceptable for v1 given Linux dominance in server/CI environments
- [ ] Additional fully-implemented harnesses (OpenCode, Aider, Codex, Gemini CLI) — v1 custom harness covers these; full implementation adds validation and tested defaults
- [ ] `zone migrate` command for config schema upgrades — only needed when schema changes break existing configs
- [ ] Multi-machine or remote workspace support — scope creep into DevPod territory; different product

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| `zone launch` idempotent lifecycle | HIGH | MEDIUM | P1 |
| Two-tier config + validation | HIGH | MEDIUM | P1 |
| Harness plugin (claude-code) | HIGH | HIGH | P1 |
| Env var forwarding + validation | HIGH | LOW | P1 |
| Auth config copy-on-start | HIGH | LOW | P1 |
| SSH agent forwarding | HIGH | LOW | P1 |
| Container security hardening | HIGH | MEDIUM | P1 |
| Container lifecycle commands | HIGH | LOW | P1 |
| .zone/ cache + config hash | HIGH | MEDIUM | P1 |
| Deterministic naming + labels | MEDIUM | LOW | P1 |
| Structured exit codes + signals | MEDIUM | LOW | P1 |
| --json output mode | MEDIUM | LOW | P1 |
| Plain-text / --plain fallback | MEDIUM | LOW | P1 |
| BubbleTea TUI | MEDIUM | HIGH | P2 |
| Host-side iptables network filtering | HIGH | HIGH | P2 |
| Hooks (pre_build, post_stop) | MEDIUM | LOW | P2 |
| Port forwarding + resource limits | MEDIUM | LOW | P2 |
| Proxy support | MEDIUM | LOW | P2 |
| GoReleaser distribution | MEDIUM | MEDIUM | P2 |
| macOS network filtering | MEDIUM | HIGH | P3 |
| Additional harnesses (non-custom) | LOW | HIGH | P3 |
| `zone migrate` schema command | LOW | HIGH | P3 |

**Priority key:**
- P1: Must have for launch
- P2: Should have, add when possible
- P3: Nice to have, future consideration

## Competitor Feature Analysis

| Feature | Docker Sandboxes | claude-code-sandbox | devcontainers/DevPod | OpenShell | Zone (planned) |
|---------|-----------------|--------------------|--------------------|-----------|----------------|
| Single-command launch | yes (sandbox run) | yes (claude-sandbox) | yes (devcontainer up) | yes | yes (zone launch) |
| Harness plugin system | no (hardcoded agents) | no (Claude only) | no (IDE-coupled) | no (hardcoded 4 agents) | yes (interface + registry) |
| Custom agent support | no | no | no | no | yes (install_commands + entrypoint) |
| Host-side network filtering | no (container-level) | no | no | yes (Landlock+seccomp) | yes (iptables, Linux v1) |
| Two-tier config (global + repo) | no | partial (config.json) | yes (devcontainer.json) | no | yes |
| Strict config validation | no | no | no | no | yes (edit-distance suggestions) |
| Config hash rebuild detection | no | no | no | no | yes |
| Persistent home volume | yes | no | no | no | yes |
| SSH agent forwarding | no (SSH isolated) | no | yes | no | yes (socket mount) |
| Auth config copy-on-start | no | partial (read-only) | no | no | yes |
| Container security hardening | partial (user namespace) | no | no | yes (seccomp+Landlock) | yes (cap drop, no-new-priv, pids) |
| TUI with plain fallback | no | web UI only | no | no | yes (BubbleTea + --plain) |
| --json scriptable output | no | no | partial | no | yes |
| Structured exit codes | no | no | no | no | yes |
| GoReleaser binary distribution | n/a (Docker product) | npm package | npm package | not packaged | yes |

## Sources

- Docker Sandboxes documentation and Claude Code sandbox guide: https://docs.docker.com/ai/sandboxes/agents/claude-code/
- Anthropic Claude Code sandboxing engineering post: https://www.anthropic.com/engineering/claude-code-sandboxing
- claude-code-sandbox (textcortex, archived): https://github.com/textcortex/claude-code-sandbox
- sandbox-agent (rivet-dev): https://github.com/rivet-dev/sandbox-agent
- OpenShell multi-agent sandbox: https://openshelldocs.com/sandboxed-execution-multiple-coding-agents-codex-opencode
- Docker Sandbox blog post (Hartley Brody, Jan 2026): https://blog.hartleybrody.com/docker-sandbox/
- Pere Villega Incus-based sandbox (Mar 2026): https://perevillega.com/posts/2026-03-03-ai-sandbox-coding-agents/
- INNOQ dev sandbox blog (Dec 2025): https://www.innoq.com/en/blog/2025/12/dev-sandbox/
- DevPod documentation: https://devpod.sh/docs/what-is-devpod
- devcontainers specification: https://containers.dev/implementors/spec/
- Collabnix Docker Sandboxes guide: https://collabnix.com/how-to-run-ai-coding-agents-safely-with-docker-sandboxes-a-definitive-guide-for-claude-and-gemini-users/
- Daytona CLI JSON output issue (TTY detection reference): https://github.com/daytonaio/daytona/issues/3494
- Docker iptables documentation: https://docs.docker.com/engine/network/packet-filtering-firewalls/

---
*Feature research for: Docker workspace manager CLI for LLM coding agents (Zone)*
*Researched: 2026-03-26*
