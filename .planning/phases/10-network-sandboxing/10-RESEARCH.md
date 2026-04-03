# Phase 10: Network Sandboxing - Research

**Researched:** 2026-04-03
**Domain:** Linux iptables, Go os/exec, Docker bridge networking, hostname DNS resolution, goroutine lifecycle management
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Platform Detection**
- D-01: Implement `Platform` struct and `DetectPlatform()` per spec §12 — fields: OS, IsDockerDesktop, IsRootless, SupportsIPTables
- D-02: `SupportsIPTables` is true only when `runtime.GOOS == "linux" && !isRootless`
- D-03: Rootless Docker detected via `strings.Contains(securityOptions, "rootless")` from Docker Info API
- D-04: `DetectPlatform()` called once in Manager constructor, stored on Manager struct for use throughout lifecycle

**Sudo Behavior (NET-11)**
- D-05: Test `sudo iptables -L -n` availability at launch time before attempting any firewall setup — fail fast if sudo unavailable
- D-06: If sudo unavailable or iptables not found: warn "Network filtering requires sudo and iptables. Falling back to unrestricted network access. Set [network] mode = \"none\" to suppress this warning." and fall back to mode=none
- D-07: Use `sudo iptables` only for firewall commands (rule insert/delete/flush) — never run the entire zone tool with sudo

**Whitelist Mode (NET-01, NET-08)**
- D-08: Default policy: DROP all outbound from the container's bridge network interface
- D-09: Evaluation order: check deny list first (deny always wins), then check merged allow list, then default DROP
- D-10: For each allowed hostname: resolve to IPs on the host, add ACCEPT rules for each resolved IP
- D-11: Always allow DNS resolution to Docker's embedded DNS (127.0.0.11:53)

**Blocklist Mode (NET-02)**
- D-12: Default policy: ACCEPT all outbound (normal Docker networking)
- D-13: Evaluation order: check merged deny list, add DROP rules for resolved IPs
- D-14: Blocklist mode does not use the allow list

**Mode "none" (NET-03)**
- D-15: No iptables rules applied — container gets default Docker networking
- D-16: This is the default

**Docker Bridge Network (NET-04)**
- D-17: Reuse existing `createNetwork()` from Phase 6
- D-18: Network created WITHOUT `--internal` flag
- D-19: IPv6 disabled on container network via sysctl (already implemented in Phase 6)

**iptables Rule Tagging (NET-05)**
- D-20: All rules tagged with `-m comment --comment "zone-{container-hash}"`
- D-21: Tag enables cleanup via grep
- D-22: Tag enables stale detection by cross-referencing running containers

**Periodic Refresh (NET-06)**
- D-23: Background goroutine started when firewall rules are applied (mode != "none")
- D-24: Every 5 minutes: re-resolve all hostnames, diff against current rules, update changed rules
- D-25: Goroutine stopped when container stops (via context cancellation)
- D-26: Use `context.WithCancel` tied to the container lifecycle

**IPv6 Bypass Prevention (NET-07)**
- D-27: Already handled in Phase 6 via sysctl
- D-28: No additional work needed

**macOS Fallback (NET-09)**
- D-29: When `Platform.OS == "darwin"` and mode is whitelist/blocklist: warn and fall back to mode=none
- D-30: Fall back to mode=none — do not error, do not block launch

**Firewall Rules Cache (NET-10)**
- D-31: Write generated rules to `.zone/firewall.rules` after every apply/refresh
- D-32: Format: human-readable iptables commands, one per line, with comments
- D-33: Purpose: inspectability

**Hostname Glob Matching (NET-12)**
- D-34: Use `filepath.Match` semantics — `*` matches any subdomain segment
- D-35: Phase 1: only literal hostnames and simple globs (`*.domain.com`) supported
- D-36: Complex patterns rejected at config parse time with clear error
- D-37: Implement in `internal/network/matcher.go` — precompile patterns

**Rule Cleanup Strategy**
- D-38: On `zone stop`: remove all iptables rules tagged with this container's hash
- D-39: On `zone clean`/`zone destroy`: remove rules + remove `.zone/firewall.rules`
- D-40: On every `zone launch`: scan for stale zone-* rules, remove them
- D-41: Stale rule detection: `sudo iptables -S | grep "zone-"` → extract hashes → cross-reference with `docker ps --filter label=com.zone.managed=true`

**nftables Compatibility**
- D-42: Use `iptables` CLI as-is — modern distros provide `iptables-nft` compatibility layer
- D-43: Test `sudo iptables -L -n` at startup — if it succeeds, proceed
- D-44: If test fails, treat as "iptables unavailable" and fall back to mode=none with warning

**Proxy Auto-Allowlisting**
- D-45: Extract hostname from `http_proxy`/`https_proxy` when whitelist mode is active
- D-46: Add extracted proxy hostnames to the runtime allow set before generating iptables rules
- D-47: Runtime addition only — don't modify user's config

**Rootless Docker Handling**
- D-48: When `Platform.IsRootless == true` and mode is whitelist/blocklist: warn and fall back to mode=none
- D-49: Fall back to mode=none — same pattern as macOS fallback

**Error Handling**
- D-50: Exit code 4 for network errors — already mapped in `cmd/errors.go`
- D-51: Extend exit code 4 handling with specific sentinel errors: `ErrFirewallSetup`, `ErrSudoUnavailable`, `ErrIPTablesUnavailable`
- D-52: All network errors include remediation hints per DX-02 pattern

### Claude's Discretion

- iptables chain naming convention (custom chain vs FORWARD rules)
- Exact DNS resolution approach (net.LookupHost vs custom resolver)
- Goroutine synchronization patterns for refresh
- Test strategy for iptables functionality (mock exec, integration tests with sudo)
- Error message exact wording beyond spec examples
- Firewall.rules file format details

### Deferred Ideas (OUT OF SCOPE)

- DNS proxy sidecar for cross-platform hostname-level filtering — Phase 2 per spec §4.9
- macOS network filtering via DNS proxy — Phase 2 (NET-V2-01, NET-V2-02)
- Advanced glob patterns (`**`) via gobwas/glob — Phase 2 (CFG-V2-02)
- Per-rule logging/audit trail — backlog
- Real-time rule change notifications in TUI status view — backlog
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| NET-01 | Whitelist mode: deny all outbound, allow specific hostnames via iptables | D-08 through D-11 cover the DROP default + ACCEPT per resolved IP approach |
| NET-02 | Blocklist mode: allow all outbound, deny specific hostnames | D-12 through D-14 cover ACCEPT default + DROP per resolved IP |
| NET-03 | Mode "none" applies no network restrictions (default) | D-15, D-16 — no iptables rules, plain Docker networking |
| NET-04 | Each container gets its own Docker bridge network | D-17 through D-19 — existing createNetwork() is sufficient |
| NET-05 | Host-side iptables rules tagged with comments for identification and cleanup | D-20 through D-22 — `-m comment --comment "zone-{hash}"` pattern |
| NET-06 | Rules refreshed periodically (every 5 min) by re-resolving hostnames | D-23 through D-26 — background goroutine with context cancellation |
| NET-07 | IPv6 disabled on container network to prevent bypass | D-27, D-28 — already done via Sysctls in Phase 6 |
| NET-08 | Deny list takes priority over allow list in whitelist mode | D-09 — deny-first evaluation order in Firewall.Apply() |
| NET-09 | macOS warns that network filtering is unavailable and falls back to mode=none | D-29, D-30 — Platform.OS=="darwin" check at launch |
| NET-10 | Firewall rules cached in .zone/firewall.rules for inspectability | D-31 through D-33 — write human-readable iptables commands after each apply/refresh |
| NET-11 | sudo iptables used only for firewall commands; fallback to none if sudo unavailable | D-05 through D-07 — test with `sudo iptables -L -n` at startup |
| NET-12 | Hostname glob matching for network rules (e.g., `*.anthropic.com`) | D-34 through D-37 — filepath.Match, precompiled patterns, complex patterns rejected |
</phase_requirements>

---

## Summary

Phase 10 implements host-side Linux iptables network sandboxing for zone containers. The approach is well-specified: the spec provides the exact `Platform` struct, `DetectPlatform()` implementation, iptables rule tagging convention, DNS resolution strategy, and fallback behavior. Three stub files (`internal/network/firewall.go`, `rules.go`, `matcher.go`) are ready for implementation.

The integration points are clear: `Manager.Launch()` calls firewall setup after container start; `Manager.Stop()` calls firewall cleanup before network removal. The `Platform` struct is added to the Manager and checked at each relevant lifecycle point. All platform fallbacks (macOS, rootless Docker, sudo unavailable) produce user-friendly warnings and silently use mode=none rather than aborting.

The highest-risk area is the iptables/nftables interaction — verified through `sudo iptables -L -n` at startup per D-43. The second risk is goroutine lifecycle correctness in the periodic refresh goroutine: context cancellation must propagate cleanly through signal handling. Both have well-defined approaches in the context decisions.

**Primary recommendation:** Implement in three waves: (1) Platform detection + matcher, (2) firewall rule generation + apply/remove, (3) Manager integration + refresh goroutine + stale rule cleanup.

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `os/exec` | stdlib | Run `sudo iptables` commands | Already used in `internal/docker/platform.go` for git; same pattern |
| `net` (LookupHost) | stdlib | Resolve hostnames to IPs for rule generation | No external deps; synchronous resolution per D-10 |
| `path/filepath` (Match) | stdlib | Glob pattern matching per spec §4.8 | Spec mandates this exact function per D-34 |
| `context` | stdlib | Goroutine lifecycle tied to container stop | Already the project-wide cancellation pattern |
| `sync` | stdlib | Mutex for refresh goroutine state protection | Standard Go concurrency primitive |
| `strings` | stdlib | Tag parsing, rootless detection | Used throughout the project |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Docker SDK `client.Info()` | `github.com/docker/docker v28.5.2` | Get security options for rootless detection | One call in `DetectPlatform()`, already in go.mod |
| `github.com/docker/docker/api/types` | same | `types.Info` struct for `SecurityOptions` field | Same import as existing docker package |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `os/exec` for iptables | go-iptables library | go-iptables is cleaner API but an extra dependency; os/exec is already in use for git commands in the project, no new patterns |
| `net.LookupHost` | custom DNS resolver | LookupHost is simpler and sufficient for Phase 1; custom resolver adds complexity with no Phase 1 benefit |
| `filepath.Match` | gobwas/glob | filepath.Match is mandated by spec for Phase 1; gobwas/glob deferred to Phase 2 |

**Installation:** No new packages needed — all standard library or already in go.mod.

---

## Architecture Patterns

### Recommended Project Structure

```
internal/network/
├── firewall.go      # Firewall struct: Apply, Remove, RefreshOnce, StartRefresh
├── rules.go         # BuildRuleSet(): NetworkConfig → RuleSet (resolved IPs)
└── matcher.go       # MatchHostname(): precompiled glob patterns

internal/docker/
├── platform.go      # ADD: Platform struct + DetectPlatform() (new functions)
├── manager.go       # ADD: platform field; modify NewManager, Launch, Stop, Destroy
├── errors.go        # ADD: ErrFirewallSetup, ErrSudoUnavailable, ErrIPTablesUnavailable
└── network.go       # No changes needed (createNetwork/removeNetwork already correct)

cmd/errors.go        # ADD: mapError cases for new firewall sentinel errors
```

### Pattern 1: Platform Detection in Manager Constructor

**What:** `DetectPlatform()` called once in `NewManager()`, stored on `Manager.platform`.
**When to use:** Any lifecycle method that needs to branch on platform capabilities.

```go
// Source: zone-spec.md §12 (lines 1275-1298)
type Platform struct {
    OS              string
    IsDockerDesktop bool
    IsRootless      bool
    SupportsIPTables bool
}

func DetectPlatform(cli DockerClient) Platform {
    info, _ := cli.Info(context.Background())
    isRootless := strings.Contains(strings.Join(info.SecurityOptions, ","), "rootless")
    isMacOS := runtime.GOOS == "darwin"
    return Platform{
        OS:               runtime.GOOS,
        IsDockerDesktop:  isMacOS || strings.Contains(info.OperatingSystem, "Docker Desktop"),
        IsRootless:       isRootless,
        SupportsIPTables: runtime.GOOS == "linux" && !isRootless,
    }
}
```

Note: `DockerClient` interface needs `Info(ctx context.Context) (types.Info, error)` added. The `types.Info` type is `system.Info` in current Docker SDK — verify via `docker/docker/api/types/system`.

### Pattern 2: iptables Rule Lifecycle in Firewall

**What:** A `Firewall` struct encapsulates all iptables operations for one container.
**When to use:** Created per container launch when mode != "none".

```go
// internal/network/firewall.go
type Firewall struct {
    containerHash string   // used in comment tag: "zone-{hash}"
    bridgeIface   string   // Docker bridge interface name, e.g. "br-abc123"
    rules         RuleSet  // resolved IPs from last apply
    mu            sync.Mutex
}

// Apply sets up iptables rules. Called once after container start.
func (f *Firewall) Apply(ctx context.Context, rs RuleSet) error

// Remove tears down all iptables rules for this container's hash.
func (f *Firewall) Remove(ctx context.Context) error

// StartRefresh starts the 5-minute background refresh goroutine.
// Cancel ctx to stop it (tied to container lifecycle).
func (f *Firewall) StartRefresh(ctx context.Context, cfg *config.NetworkConfig)
```

### Pattern 3: iptables Comment Tagging

**What:** Every rule includes `-m comment --comment "zone-{hash}"` for identification.
**When to use:** All rule insertions.

```bash
# Whitelist: DROP default outbound from bridge interface
sudo iptables -I FORWARD -i br-abc123 -j DROP -m comment --comment "zone-abc123default"

# ACCEPT for allowed IP
sudo iptables -I FORWARD -i br-abc123 -d 1.2.3.4 -j ACCEPT -m comment --comment "zone-abc123"

# Always allow Docker embedded DNS
sudo iptables -I FORWARD -i br-abc123 -d 127.0.0.11 -p udp --dport 53 -j ACCEPT -m comment --comment "zone-abc123"
sudo iptables -I FORWARD -i br-abc123 -d 127.0.0.11 -p tcp --dport 53 -j ACCEPT -m comment --comment "zone-abc123"
```

**Chain choice (Claude's discretion):** Use the FORWARD chain with `-i {bridge}` interface matching rather than a custom chain. This is simpler (no chain creation/deletion), and the bridge interface name is unique per container. Rules are still cleanly identifiable by the comment tag.

### Pattern 4: Stale Rule Cleanup

**What:** On every `zone launch`, scan for dangling rules from previous crashed processes.
**When to use:** At the start of `Firewall.Apply()`.

```go
// Pseudocode for stale detection
out, _ := exec.CommandContext(ctx, "sudo", "iptables", "-S").Output()
hashes := extractZoneHashes(out)  // find all "zone-{hash}" comment patterns
running := listRunningZoneHashes(ctx, dockerClient)
for _, hash := range hashes {
    if !running[hash] {
        removeRulesForHash(ctx, hash)
    }
}
```

### Pattern 5: Background Refresh Goroutine

**What:** Re-resolves hostnames every 5 minutes and updates changed rules.
**When to use:** After initial `Firewall.Apply()` when mode != "none".

```go
func (f *Firewall) StartRefresh(ctx context.Context, cfg *config.NetworkConfig) {
    go func() {
        ticker := time.NewTicker(5 * time.Minute)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                f.mu.Lock()
                newRS, _ := BuildRuleSet(cfg)
                if !rulesEqual(f.rules, newRS) {
                    _ = f.Apply(ctx, newRS)  // re-apply with new IPs
                }
                f.mu.Unlock()
            }
        }
    }()
}
```

### Pattern 6: Matcher Precompilation

**What:** Parse and validate glob patterns once at config-load time; reuse for matching.
**When to use:** `internal/network/matcher.go` — called from `rules.go` during `BuildRuleSet()`.

```go
// internal/network/matcher.go
type CompiledPattern struct {
    raw     string
    isGlob  bool
}

// Compile validates the pattern and returns a CompiledPattern.
// Returns error for patterns that are neither literal nor simple glob (*.domain.com).
func Compile(pattern string) (CompiledPattern, error) {
    // Reject patterns with path separators, double stars, brackets, braces
    if strings.Contains(pattern, "/") || strings.Contains(pattern, "**") ||
       strings.ContainsAny(pattern, "[]{}") {
        return CompiledPattern{}, fmt.Errorf("unsupported pattern %q: only literal hostnames and *.domain.com globs are supported in Phase 1", pattern)
    }
    return CompiledPattern{raw: pattern, isGlob: strings.HasPrefix(pattern, "*.")}, nil
}

// Match returns true if hostname matches the compiled pattern.
// Uses filepath.Match for glob patterns.
func (p CompiledPattern) Match(hostname string) bool {
    if !p.isGlob {
        return p.raw == hostname
    }
    matched, _ := filepath.Match(p.raw, hostname)
    return matched
}
```

### Pattern 7: Docker Bridge Interface Discovery

**What:** To apply iptables rules on the correct interface, we need the bridge interface name.
**When to use:** Before `Firewall.Apply()`.

The Docker bridge interface name can be obtained from `NetworkInspect()` — already available in `DockerClient` interface. The `Options` field of the network inspect contains `"com.docker.network.bridge.name"`. Alternatively, the name is predictable: Docker names bridge interfaces `br-{network-id[:12]}`.

```go
// From NetworkInspect result:
info, _ := m.client.NetworkInspect(ctx, netID, network.InspectOptions{})
bridgeIface := info.Options["com.docker.network.bridge.name"]
// If not set, fall back to: "br-" + netID[:12]
```

### Anti-Patterns to Avoid

- **Custom iptables chain per container:** More complex lifecycle (chain creation/deletion/flush), harder to clean up stale entries. Use FORWARD with interface matching + comment tags instead.
- **Running zone as root for firewall:** The spec explicitly prohibits this (D-07). Always use `sudo iptables` for the specific firewall commands only.
- **In-container iptables (CAP_NET_ADMIN):** The spec explains this is a security anti-pattern (agent could disable its own firewall). Always host-side.
- **Blocking DNS in whitelist mode without Docker DNS exception:** Containers resolve hostnames via Docker's embedded DNS at 127.0.0.11. Always add the DNS exception rule (D-11) or the container cannot resolve any names.
- **IP-only tracking without periodic refresh:** CDN IPs change. The 5-minute refresh goroutine (D-23) handles this.
- **Forgetting to clean up on panic/crash:** That is what stale rule detection (D-40, D-41) is for — runs on every launch.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Hostname-to-IP resolution | Custom DNS client | `net.LookupHost` | Standard library; uses OS resolver; sufficient for Phase 1 |
| Glob matching | Custom wildcard matcher | `filepath.Match` | Spec mandates it; handles `*` correctly for subdomain matching |
| Process execution | Shell scripts calling iptables | `os/exec.CommandContext` | Context propagation, output capture, already used in project |
| Platform detection | Manual `/etc/os-release` parsing | Docker Info API SecurityOptions | Definitive rootless detection; already in DockerClient interface scope |

**Key insight:** This entire phase is about orchestrating existing OS primitives (iptables, DNS resolution) from Go. The challenge is lifecycle management and test coverage, not custom algorithms.

---

## Common Pitfalls

### Pitfall 1: Docker Info API Not in DockerClient Interface

**What goes wrong:** `DetectPlatform()` spec code calls `cli.Info(context.Background())` but the current `DockerClient` interface in `client_interface.go` does not include `Info()`. Build fails.

**Why it happens:** The interface was defined minimally for Phase 6 functionality. DetectPlatform() is new in Phase 10.

**How to avoid:** Add `Info(ctx context.Context) (system.Info, error)` to the `DockerClient` interface AND the `mockClient` in `manager_test.go` before implementing `DetectPlatform()`.

**Warning signs:** Compile error: `m.client.Info undefined (type DockerClient has no field or method Info)`.

### Pitfall 2: Bridge Interface Name Not Available Before NetworkInspect

**What goes wrong:** Rules applied with wrong interface name (e.g., `docker0` instead of `br-abc123`) silently pass the iptables call but filter wrong traffic.

**Why it happens:** The bridge interface name is generated by Docker from the network ID. It's not predictable purely from the network name.

**How to avoid:** After `createNetwork()`, call `NetworkInspect()` to retrieve the `com.docker.network.bridge.name` option. Store the bridge interface name in the cache or pass it to `Firewall.Apply()`. Note: the network inspect is already available via the `DockerClient` interface.

**Warning signs:** Network rules applied but container traffic not filtered (test by running `curl` inside container to blocked host).

### Pitfall 3: iptables Rule Order — DROP Must Come After ACCEPT Rules

**What goes wrong:** If the DROP default rule is inserted first (position 1 in FORWARD chain), subsequent ACCEPT rules won't match because iptables uses first-match-wins.

**Why it happens:** `iptables -I` (insert) inserts at position 1 by default. Insert ACCEPT rules after the DROP rule and they'll be checked first in practice only if inserted at position 1 (pushing DROP down).

**How to avoid:** Insert rules with explicit position or use `iptables -I FORWARD 1` for ACCEPT rules and then `iptables -A FORWARD` (append) for the DROP default. Better yet: use a position-aware insertion strategy. Insert ACCEPT rules at position 1 (they stack in LIFO), then append the DROP at end. This ensures: ACCEPT for DNS → ACCEPT for allowed IPs → DROP default.

**Warning signs:** All traffic blocked even for allowed hostnames; `sudo iptables -L FORWARD -n` shows DROP rule before ACCEPT rules.

### Pitfall 4: Goroutine Leak on Container Stop

**What goes wrong:** `StartRefresh()` goroutine continues running after container stops, causing a goroutine leak. On repeated launch/stop cycles, goroutines accumulate.

**Why it happens:** `ctx` passed to `StartRefresh()` must be the container-lifecycle context, not the process context. If the wrong context is passed (e.g., `context.Background()`), the goroutine never stops.

**How to avoid:** The context passed to `StartRefresh()` must be cancelled when `Manager.Stop()` is called. Use `context.WithCancel` derived from the launch context. Store the `cancel` function and call it in `Stop()`.

**Warning signs:** `pprof` shows goroutine count growing with each launch/stop cycle.

### Pitfall 5: Stale Rules When Zone Process Crashes Mid-Apply

**What goes wrong:** If the zone process crashes between inserting some ACCEPT rules and the DROP default rule, partial rules remain. On next launch, stale detection finds them and removes them — but only if the partial rules have the `zone-{hash}` comment tag.

**Why it happens:** iptables operations are not atomic. Each `sudo iptables` call is a separate process.

**How to avoid:** Always tag EVERY rule with the comment during insertion (D-20). Even partial rules will then be found by stale detection on next launch. Order of operations: insert ACCEPT rules first (tagged), then insert DROP (tagged). Stale detection runs before any new rules are applied.

**Warning signs:** After crash, running `sudo iptables -S | grep zone-` shows leftover rules with no corresponding running container.

### Pitfall 6: IPv4 Only — Container May Still Use IPv6

**What goes wrong:** iptables only applies to IPv4. If a container accesses an IPv6 address, the rules are bypassed.

**Why it happens:** iptables ≠ ip6tables.

**How to avoid:** IPv6 is already disabled via `Sysctls["net.ipv6.conf.all.disable_ipv6"] = "1"` in `createContainer()` (Phase 6, confirmed in `internal/docker/manager.go` line 181-183). This is D-27. No additional work.

**Warning signs:** Research only — this is already handled.

### Pitfall 7: nftables vs iptables on Modern Linux

**What goes wrong:** On Ubuntu 20.04+, Fedora, etc., `iptables` is actually `iptables-nft` (a shim). Rules are stored in nftables. The `sudo iptables -S` output format may differ slightly.

**Why it happens:** Linux distributions migrated from legacy iptables to nftables over 2020-2022.

**How to avoid:** Per D-42 through D-44, test `sudo iptables -L -n` at startup. If it succeeds, proceed regardless of backend. The `iptables-nft` shim translates commands transparently. The comment tag (`-m comment`) works in both. The only practical difference is that rules may appear in `nft list ruleset` rather than `iptables -S` — but using `iptables -S` for stale detection still works through the shim.

**Warning signs:** (None in normal operation — the shim is transparent.)

---

## Code Examples

Verified patterns from project codebase and spec:

### Platform Detection (per spec §12)

```go
// Source: zone-spec.md §12 lines 1276-1298
// File: internal/docker/platform.go (additions)

import (
    "context"
    "runtime"
    "strings"
    "github.com/docker/docker/api/types/system"
)

type Platform struct {
    OS              string
    IsDockerDesktop bool
    IsRootless      bool
    SupportsIPTables bool
}

func DetectPlatform(ctx context.Context, cli DockerClient) Platform {
    info, _ := cli.Info(ctx)
    secOpts := strings.Join(info.SecurityOptions, ",")
    isRootless := strings.Contains(secOpts, "rootless")
    isMacOS := runtime.GOOS == "darwin"
    return Platform{
        OS:               runtime.GOOS,
        IsDockerDesktop:  isMacOS || strings.Contains(info.OperatingSystem, "Docker Desktop"),
        IsRootless:       isRootless,
        SupportsIPTables: runtime.GOOS == "linux" && !isRootless,
    }
}
```

### Sudo iptables Availability Check (D-05)

```go
// Source: decisions D-05, D-06, D-07

func CheckIPTablesAvailable(ctx context.Context) error {
    cmd := exec.CommandContext(ctx, "sudo", "iptables", "-L", "-n")
    if err := cmd.Run(); err != nil {
        return ErrIPTablesUnavailable
    }
    return nil
}
```

### Hostname Matching with filepath.Match (D-34)

```go
// Source: zone-spec.md §4.8 lines 470-478

import "path/filepath"

// MatchHostname returns true if hostname matches pattern.
// Pattern may be literal ("api.anthropic.com") or glob ("*.anthropic.com").
func MatchHostname(pattern, hostname string) bool {
    matched, err := filepath.Match(pattern, hostname)
    return err == nil && matched
}
```

### iptables Rule Application (approach from spec §4.9)

```go
// Source: zone-spec.md §4.9 lines 498-516

// runIPTables executes: sudo iptables [args...]
func runIPTables(ctx context.Context, args ...string) error {
    fullArgs := append([]string{"iptables"}, args...)
    cmd := exec.CommandContext(ctx, "sudo", fullArgs...)
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("iptables %v: %w: %s", args, err, out)
    }
    return nil
}

// Example whitelist rule: allow resolved IP
func allowIP(ctx context.Context, bridgeIface, ip, tag string) error {
    return runIPTables(ctx,
        "-I", "FORWARD",
        "-i", bridgeIface,
        "-d", ip,
        "-j", "ACCEPT",
        "-m", "comment", "--comment", tag,
    )
}

// Example: DROP default for whitelist mode (appended after ACCEPT rules)
func dropDefault(ctx context.Context, bridgeIface, tag string) error {
    return runIPTables(ctx,
        "-A", "FORWARD",
        "-i", bridgeIface,
        "-j", "DROP",
        "-m", "comment", "--comment", tag+"-default",
    )
}
```

### Stale Rule Detection (D-40, D-41)

```go
// Source: decisions D-41

func extractZoneHashesFromIPTables(ctx context.Context) (map[string]bool, error) {
    cmd := exec.CommandContext(ctx, "sudo", "iptables", "-S")
    out, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    hashes := make(map[string]bool)
    // Match: --comment "zone-{hash}"
    re := regexp.MustCompile(`--comment "zone-([a-f0-9]+)`)
    for _, m := range re.FindAllSubmatch(out, -1) {
        hashes[string(m[1])] = true
    }
    return hashes, nil
}
```

### Firewall Rules Cache Write (D-31, D-32)

```go
// Source: decisions D-31 to D-33

func (f *Firewall) writeRulesCache(cacheDir string, rules []string) error {
    content := "# zone firewall rules\n# Generated: " + time.Now().Format(time.RFC3339) + "\n\n"
    content += strings.Join(rules, "\n") + "\n"
    path := filepath.Join(cacheDir, "firewall.rules")
    return os.WriteFile(path, []byte(content), 0644)
}
```

### Manager Integration Points (launch.go)

```go
// Source: internal/docker/launch.go — createAndStart() needs extension

// After container starts successfully, apply firewall rules if mode != "none":
// 1. Check platform.SupportsIPTables (warn + skip if false)
// 2. Check sudo availability (warn + skip if unavailable)
// 3. Clean stale rules
// 4. Build rule set from config
// 5. Apply rules
// 6. Write firewall.rules cache
// 7. Start refresh goroutine (with cancel func stored for Stop())

// In Manager.Stop(), before removeNetwork():
// cancel the refresh goroutine context
// call firewall.Remove(ctx)
// delete .zone/firewall.rules
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Legacy iptables (iptables-legacy) | iptables-nft shim on modern distros | Ubuntu 20.10+, Fedora 33+ (2020-2021) | `iptables` CLI still works; backend is nftables |
| Manual chain management | FORWARD chain + interface filter | N/A — project choice | Simpler lifecycle; comment tags handle cleanup |
| CAP_NET_ADMIN in container | Host-side iptables with sudo | N/A — spec design | Security boundary preserved; agent cannot disable rules |

**Deprecated/outdated:**
- `--net=none` Docker flag: Would block all networking including DNS. Not used here; we use bridge + iptables.
- Docker `--link` networking: Legacy feature; project already uses bridge networks per Phase 6.

---

## Open Questions

1. **DockerClient interface needs `Info()` method**
   - What we know: `DetectPlatform()` per spec calls `cli.Info(context.Background())`. Current `DockerClient` interface has no `Info()` method. The `mockClient` in `manager_test.go` also lacks it.
   - What's unclear: The Docker SDK's `Info()` return type — whether it's `types.Info` or `system.Info` (the types package was reorganized in Docker SDK v25+).
   - Recommendation: Add `Info(ctx context.Context) (system.Info, error)` to interface (using `github.com/docker/docker/api/types/system`) and add a stub to `mockClient` before implementing `DetectPlatform()`.

2. **Bridge interface name retrieval timing**
   - What we know: After `createNetwork()`, we have the network ID. `NetworkInspect()` returns `Options["com.docker.network.bridge.name"]`.
   - What's unclear: Whether this option is always populated (vs. falling back to `br-{id[:12]}`).
   - Recommendation: Implement with primary path using `NetworkInspect()` options; fallback to `"br-" + netID[:12]` if the option is empty. Log a warning in the fallback case.

3. **Goroutine cancel function storage**
   - What we know: The refresh goroutine needs a `context.WithCancel` derived from the container lifecycle. `Manager.Stop()` must call the cancel function.
   - What's unclear: Where to store the cancel function — as a Manager field, or in a separate per-container state struct.
   - Recommendation: Add a `firewallCancel context.CancelFunc` field to `Manager`. Set it in `createAndStart()` after `Firewall.Apply()`. Call it in `Stop()` before `Firewall.Remove()`. Set to nil after cancellation.

4. **Test strategy for iptables (sudo unavailable in CI)**
   - What we know: Tests run in a Docker container without sudo. `sudo iptables` will fail.
   - What's unclear: Best approach to test firewall logic without live iptables.
   - Recommendation: Use a function-level abstraction for `runIPTables` — inject an `execFn func(ctx context.Context, args ...string) error` into the `Firewall` struct. Default to the real `sudo iptables` call. In tests, inject a mock that records calls and returns nil. This lets unit tests verify argument construction without requiring sudo. Integration tests with actual iptables are manually-only (tagged with `//go:build integration`).

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing + testify v1.11.1 |
| Config file | none (Go native) |
| Quick run command | `go test ./internal/network/... ./tests/ -run TestNet -v` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| NET-01 | Whitelist mode generates DROP default + ACCEPT per-IP rules | unit | `go test ./internal/network/... -run TestFirewallWhitelist` | ❌ Wave 0 |
| NET-02 | Blocklist mode generates DROP per-IP rules, no default DROP | unit | `go test ./internal/network/... -run TestFirewallBlocklist` | ❌ Wave 0 |
| NET-03 | mode=none produces no iptables calls | unit | `go test ./internal/network/... -run TestFirewallModeNone` | ❌ Wave 0 |
| NET-04 | Container gets its own bridge network | unit | `go test ./internal/docker/... -run TestCreateNetwork` | ❌ Wave 0 (existing network.go has no tests) |
| NET-05 | Rules are tagged with zone-{hash} comment | unit | `go test ./internal/network/... -run TestRuleTagging` | ❌ Wave 0 |
| NET-06 | Refresh goroutine re-resolves and updates rules | unit | `go test ./internal/network/... -run TestFirewallRefresh` | ❌ Wave 0 |
| NET-07 | IPv6 disabled via sysctl | unit (existing) | `go test ./internal/docker/... -run TestContainerSysctls` | existing via manager_test.go |
| NET-08 | Deny list evaluated before allow list | unit | `go test ./internal/network/... -run TestRuleEvalOrder` | ❌ Wave 0 |
| NET-09 | macOS falls back to mode=none with warning | unit | `go test ./tests/... -run TestNetworkMacOSFallback` | ❌ Wave 0 |
| NET-10 | firewall.rules written after apply | unit | `go test ./internal/network/... -run TestFirewallRulesCache` | ❌ Wave 0 |
| NET-11 | sudo unavailable → warn + fallback to none | unit | `go test ./internal/network/... -run TestSudoUnavailable` | ❌ Wave 0 |
| NET-12 | Glob patterns match correctly; complex patterns rejected | unit | `go test ./tests/... -run TestMatcher` | ❌ Wave 0 (tests/matcher_test.go exists but is empty) |

### Sampling Rate

- **Per task commit:** `go test ./internal/network/... ./tests/ -run TestNet -v`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `tests/matcher_test.go` — covers NET-12 (file exists but empty — add tests)
- [ ] `internal/network/firewall_test.go` — covers NET-01, NET-02, NET-03, NET-05, NET-06, NET-08, NET-10, NET-11 using injected mock exec function
- [ ] `tests/network_platform_test.go` — covers NET-09 (macOS fallback), rootless fallback, DetectPlatform behavior
- [ ] `internal/docker/client_interface.go` — add `Info(ctx context.Context) (system.Info, error)` to DockerClient interface (required before DetectPlatform can compile)

Framework install: none needed — `go test` already configured.

---

## Sources

### Primary (HIGH confidence)

- `/workspace/zone/zone-spec.md` §4.8 (lines 470-478) — Network rule syntax, filepath.Match mandate
- `/workspace/zone/zone-spec.md` §4.9 (lines 491-518) — Complete network implementation: iptables approach, bridge network, DNS, IPv6 disable, rule tagging, refresh, macOS, sudo, CDN limitation
- `/workspace/zone/zone-spec.md` §4.11 (lines 531-539) — Proxy auto-allowlisting
- `/workspace/zone/zone-spec.md` §12 (lines 1275-1298) — Platform struct + DetectPlatform() exact code
- `/workspace/zone/.planning/phases/10-network-sandboxing/10-CONTEXT.md` — All implementation decisions D-01 through D-52
- `/workspace/zone/internal/docker/manager.go` — Existing Manager pattern, createContainer, Stop, Destroy
- `/workspace/zone/internal/docker/platform.go` — Existing platform functions (HostUID, MacOSUsername, DetectGitIdentity)
- `/workspace/zone/internal/docker/errors.go` — Existing sentinel errors + ErrNetworkUnsupported placeholder
- `/workspace/zone/cmd/errors.go` — Existing mapError with exit code 4 already present
- `/workspace/zone/internal/config/types.go` — NetworkConfig struct (Mode, Allow, Deny, DefaultAllow, DefaultDeny)
- `/workspace/zone/internal/config/merge.go` — Network allow/deny merge: `mergeAppend(global.DefaultAllow, repo.Allow)`
- `/workspace/zone/internal/cache/cache.go` — writeAtomic pattern for firewall.rules cache file
- `/workspace/zone/internal/docker/naming.go` — ContainerName hash format (zone-{name}-{16-char-sha256})

### Secondary (MEDIUM confidence)

- Go stdlib `net.LookupHost` documentation — host resolver, returns `[]string` of IPs
- Go stdlib `path/filepath.Match` documentation — glob semantics, `*` does not match path separator
- Go stdlib `os/exec.CommandContext` — context-aware subprocess execution

### Tertiary (LOW confidence)

- Community pattern for iptables-nft compatibility (multiple Linux distribution changelogs, 2020-2022)

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all stdlib or already in go.mod; no new dependencies
- Architecture: HIGH — spec provides exact code for Platform struct and DetectPlatform(); decisions cover all design choices
- Pitfalls: HIGH — sourced from code review of existing Manager + spec + iptables knowledge verified against spec text
- Test strategy: MEDIUM — mock exec injection pattern is a Claude's discretion area, but it is idiomatic Go

**Research date:** 2026-04-03
**Valid until:** 2026-05-03 (stable domain — iptables API and Go stdlib do not change frequently)
