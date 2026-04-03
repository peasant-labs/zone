# Phase 10: Network Sandboxing - Context

**Gathered:** 2026-04-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement host-side iptables-based network sandboxing for containers on Linux. Two modes: whitelist (deny all, allow specific hostnames) and blocklist (allow all, deny specific hostnames). Mode "none" is the default and applies no restrictions. Hostname rules support literal and glob matching via `filepath.Match`. DNS resolution runs on the host, translating hostnames to IPs for iptables rules. A background goroutine refreshes rules every 5 minutes by re-resolving hostnames. Rules are tagged with container-specific comments for identification and cleanup. Stale rules from crashed processes are detected and removed. macOS falls back to mode=none with a warning. Rootless Docker falls back to mode=none with a clear error. The `internal/network/` package (existing stubs) implements rule generation, matching, and iptables management. The `internal/docker/` package integrates firewall lifecycle into container launch/stop.

</domain>

<decisions>
## Implementation Decisions

### Platform Detection
- **D-01:** Implement `Platform` struct and `DetectPlatform()` per spec §12 — fields: OS, IsDockerDesktop, IsRootless, SupportsIPTables
- **D-02:** `SupportsIPTables` is true only when `runtime.GOOS == "linux" && !isRootless`
- **D-03:** Rootless Docker detected via `strings.Contains(securityOptions, "rootless")` from Docker Info API
- **D-04:** `DetectPlatform()` called once in Manager constructor, stored on Manager struct for use throughout lifecycle

### Sudo Behavior (NET-11)
- **D-05:** Test `sudo iptables -L -n` availability at launch time before attempting any firewall setup — fail fast if sudo unavailable
- **D-06:** If sudo unavailable or iptables not found: warn "Network filtering requires sudo and iptables. Falling back to unrestricted network access. Set [network] mode = \"none\" to suppress this warning." and fall back to mode=none
- **D-07:** Use `sudo iptables` only for firewall commands (rule insert/delete/flush) — never run the entire zone tool with sudo

### Whitelist Mode (NET-01, NET-08)
- **D-08:** Default policy: DROP all outbound from the container's bridge network interface
- **D-09:** Evaluation order: check deny list first (deny always wins), then check merged allow list (global default_allow + per-repo allow), then default DROP
- **D-10:** For each allowed hostname: resolve to IPs on the host, add ACCEPT rules for each resolved IP
- **D-11:** Always allow DNS resolution to Docker's embedded DNS (127.0.0.11:53) — container must be able to resolve hostnames internally even though iptables blocks the resolved IPs

### Blocklist Mode (NET-02)
- **D-12:** Default policy: ACCEPT all outbound (normal Docker networking)
- **D-13:** Evaluation order: check merged deny list (global default_deny + per-repo deny), add DROP rules for resolved IPs
- **D-14:** Blocklist mode does not use the allow list

### Mode "none" (NET-03)
- **D-15:** No iptables rules applied — container gets default Docker networking
- **D-16:** This is the default to minimize first-run friction — users opt into sandboxing

### Docker Bridge Network (NET-04)
- **D-17:** Each container already gets its own bridge network via `createNetwork()` in Phase 6 — reuse existing implementation
- **D-18:** Network created WITHOUT `--internal` flag (which would block all external traffic and make selective filtering impossible)
- **D-19:** IPv6 disabled on container network via sysctl (already implemented in Phase 6 security flags)

### iptables Rule Tagging (NET-05)
- **D-20:** All rules tagged with `-m comment --comment "zone-{container-hash}"` for identification
- **D-21:** Tag enables cleanup: `sudo iptables -S | grep "zone-{hash}"` to find rules for a specific container
- **D-22:** Tag enables stale detection: find all zone-* comments, cross-reference with running containers

### Periodic Refresh (NET-06)
- **D-23:** Background goroutine started when firewall rules are applied (mode != "none")
- **D-24:** Every 5 minutes: re-resolve all hostnames to IPs, diff against current rules, update changed rules
- **D-25:** Goroutine stopped when container stops (via context cancellation from zone stop) or zone process exits
- **D-26:** Use `context.WithCancel` tied to the container lifecycle — cancel propagates through signal handling chain

### IPv6 Bypass Prevention (NET-07)
- **D-27:** Already handled in Phase 6: `net.ipv6.conf.all.disable_ipv6=1` sysctl on container network
- **D-28:** No additional work needed — carried forward from Phase 6 security flags

### macOS Fallback (NET-09)
- **D-29:** When `Platform.OS == "darwin"` and network mode is whitelist or blocklist: warn "Network filtering is not available on macOS in this version. Container will have unrestricted network access. Set `network.mode = \"none\"` to suppress this warning."
- **D-30:** Fall back to mode=none — do not error, do not block launch

### Firewall Rules Cache (NET-10)
- **D-31:** Write generated rules to `.zone/firewall.rules` after every apply/refresh
- **D-32:** Format: human-readable iptables commands, one per line, with comments
- **D-33:** Purpose: inspectability — users can `cat .zone/firewall.rules` to see what's enforced

### Hostname Glob Matching (NET-12)
- **D-34:** Use `filepath.Match` semantics per spec §4.8 — `*` matches any subdomain segment
- **D-35:** Phase 1 constraint: only literal hostnames and simple globs (`*.domain.com`) supported
- **D-36:** Complex patterns rejected at config parse time with clear error
- **D-37:** Implement in `internal/network/matcher.go` — precompile patterns for efficient repeated matching

### Rule Cleanup Strategy
- **D-38:** On `zone stop`: remove all iptables rules tagged with this container's hash
- **D-39:** On `zone clean`/`zone destroy`: remove rules + remove `.zone/firewall.rules`
- **D-40:** On every `zone launch`: scan for stale zone-* rules (rules whose container hash doesn't match any running zone container), remove them
- **D-41:** Stale rule detection: `sudo iptables -S | grep "zone-"` → extract hashes → cross-reference with `docker ps --filter label=com.zone.managed=true`

### nftables Compatibility
- **D-42:** Use the `iptables` CLI as-is — modern distros provide `iptables-nft` compatibility layer that translates iptables commands to nftables rules
- **D-43:** At startup, test `sudo iptables -L -n` — if it succeeds, proceed regardless of whether the backend is legacy iptables or nftables
- **D-44:** If the test fails, treat as "iptables unavailable" and fall back to mode=none with warning

### Proxy Auto-Allowlisting
- **D-45:** Per spec §4.11: when whitelist mode is active, extract hostname from `http_proxy`/`https_proxy` config values (or auto-detected host values)
- **D-46:** Add extracted proxy hostnames to the runtime allow set before generating iptables rules
- **D-47:** This is a runtime addition only — don't modify the user's config, just the resolved allow set

### Rootless Docker Handling
- **D-48:** When `Platform.IsRootless == true` and network mode is whitelist or blocklist: warn "Network filtering is unavailable with rootless Docker (iptables requires root). Falling back to unrestricted network access."
- **D-49:** Fall back to mode=none — same pattern as macOS fallback

### Error Handling
- **D-50:** Exit code 4 for network errors — already mapped in `cmd/errors.go` from Phase 8
- **D-51:** Extend exit code 4 handling with specific sentinel errors: `ErrFirewallSetup`, `ErrSudoUnavailable`, `ErrIPTablesUnavailable`
- **D-52:** All network errors include remediation hints per DX-02 pattern

### Claude's Discretion
- iptables chain naming convention (custom chain vs FORWARD rules)
- Exact DNS resolution approach (net.LookupHost vs custom resolver)
- Goroutine synchronization patterns for refresh
- Test strategy for iptables functionality (mock exec, integration tests with sudo)
- Error message exact wording beyond spec examples
- Firewall.rules file format details

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Network implementation specification
- `zone-spec.md` §4.8 (lines 470-478) — Network rule syntax: literal hostnames and glob patterns, `filepath.Match` semantics
- `zone-spec.md` §4.9 (lines 491-518) — Complete network implementation design: host-side iptables, bridge network, DNS resolution, IPv6 disable, rule tagging, periodic refresh, macOS limitation, sudo requirement, known CDN limitation
- `zone-spec.md` §4.11 (lines 531-539) — Proxy support with auto-allowlisting in whitelist mode

### Platform detection
- `zone-spec.md` §12 (lines 1275-1298) — Platform struct and DetectPlatform() with SupportsIPTables, IsRootless detection

### Config types
- `zone-spec.md` §4.2 (lines 284-307) — Network config fields: mode, default_allow, default_deny, allow, deny
- `internal/config/types.go` lines 73-82 — NetworkConfig struct (already implemented)
- `internal/config/merge.go` — Network allow/deny merge strategy (already implemented)

### Project structure
- `zone-spec.md` §7 (lines 682-688) — File layout: `internal/network/firewall.go`, `rules.go`, `matcher.go`

### Existing code to integrate with
- `internal/docker/network.go` — createNetwork/removeNetwork (Phase 6, reuse)
- `internal/docker/platform.go` — Needs Platform struct + DetectPlatform() added
- `internal/docker/errors.go` — Sentinel errors (extend with firewall errors)
- `cmd/errors.go` lines 46-49 — Exit code 4 mapping already present (extend)
- `internal/network/` — Stub files ready for implementation: firewall.go, matcher.go, rules.go

### Error handling pattern
- `zone-spec.md` §3.3 (lines 87-97) — Exit code 4 for network errors
- `zone-spec.md` §3.12 (lines 218-243) — Actionable error message format

### Requirements
- `.planning/REQUIREMENTS.md` — NET-01 through NET-12

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/network/firewall.go` — Stub ready for iptables rule generation, apply, remove, refresh
- `internal/network/matcher.go` — Stub ready for hostname glob matching engine
- `internal/network/rules.go` — Stub ready for rule parsing from NetworkConfig
- `internal/docker/network.go` — `createNetwork()`/`removeNetwork()` already handle bridge network lifecycle
- `internal/docker/platform.go` — Needs `Platform` struct and `DetectPlatform()` function added
- `internal/docker/errors.go` — `ErrNetworkUnsupported` already defined; extend with firewall-specific errors
- `cmd/errors.go` — Exit code 4 mapping and remediation hint already present
- `internal/config/types.go` — `NetworkConfig` struct fully defined with all needed fields
- `internal/config/merge.go` — Network allow/deny merge logic already implemented
- `internal/config/validate.go` — Warning for `mode=none` with non-empty allow list already implemented

### Established Patterns
- Error wrapping: `fmt.Errorf("context: %w", err)` throughout
- Sentinel errors in `internal/docker/errors.go`
- Docker SDK calls with context propagation for cancellation
- Manager struct pattern: constructor validates dependencies, methods orchestrate lifecycle
- `os/exec` for external commands (used in `platform.go` for git, extend for `sudo iptables`)

### Integration Points
- `internal/docker/manager.go` — Manager.Launch() needs to call firewall setup after container start
- `internal/docker/manager.go` — Manager.Stop() needs to call firewall cleanup before network removal
- `internal/docker/launch.go` — Launch state machine needs platform check + firewall integration
- `cmd/launch.go` — May need to surface network warnings to user
- `.zone/firewall.rules` — Cache directory already managed by `internal/cache/`

</code_context>

<specifics>
## Specific Ideas

No specific requirements — all decisions auto-selected from recommended defaults following the spec's prescriptive network implementation design. The spec provides exact Platform struct, DetectPlatform() code, iptables approach, rule tagging convention, refresh interval, and fallback behavior.

</specifics>

<deferred>
## Deferred Ideas

- DNS proxy sidecar for cross-platform hostname-level filtering — Phase 2 per spec §4.9
- macOS network filtering via DNS proxy — Phase 2 (NET-V2-01, NET-V2-02)
- Advanced glob patterns (`**`) via gobwas/glob — Phase 2 (CFG-V2-02)
- Per-rule logging/audit trail — backlog
- Real-time rule change notifications in TUI status view — backlog

</deferred>

---

*Phase: 10-network-sandboxing*
*Context gathered: 2026-04-03*
