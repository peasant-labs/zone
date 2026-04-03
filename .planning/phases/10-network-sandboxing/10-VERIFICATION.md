---
phase: 10-network-sandboxing
verified: 2026-04-03T04:47:09Z
status: gaps_found
score: 4/6 must-haves verified
gaps:
  - truth: "iptables rules are tagged with the container ID and cleaned up when the container stops or zone clean runs"
    status: failed
    reason: "Rules are tagged, but cleanup is only attempted through the in-memory Manager.firewall pointer populated during the current launch process. Fresh stop/destroy/clean commands do not reconstruct firewall state, so host iptables rules can be left behind."
    artifacts:
      - path: "internal/docker/manager.go"
        issue: "Stop only removes rules when m.firewall != nil; NewManager/newManagerWithClient never hydrate firewall state from cache."
      - path: "cmd/clean.go"
        issue: "zone clean removes .zone/ only and never invokes firewall cleanup."
      - path: "cmd/stop.go"
        issue: "zone stop creates a fresh Manager, so firewall cleanup path is skipped unless the same process that launched the container is still alive."
    missing:
      - "Reconstruct firewall cleanup from cached network/container identity so Stop/Destroy work in fresh processes."
      - "Invoke firewall cleanup from zone clean or explicitly route clean through a teardown path that removes tagged iptables rules first."
      - "Add regression tests for fresh-process stop/destroy/clean firewall cleanup."
  - truth: "Hostname glob patterns (e.g., *.anthropic.com) match correctly in both whitelist and blocklist rules"
    status: failed
    reason: "The matcher exists, but generated firewall rules do not enforce globs end-to-end: whitelist allow globs are rejected and blocklist deny globs are skipped entirely."
    artifacts:
      - path: "internal/network/rules.go"
        issue: "BuildRuleSet errors on whitelist allow globs and ignores deny globs with `if cp.IsGlob() { continue }`, so blocklist glob rules never produce iptables entries."
      - path: "internal/network/matcher.go"
        issue: "Pattern matching logic is implemented, but only partially wired into rule generation."
    missing:
      - "Implement a real glob-to-enforced-rule path for whitelist/blocklist flows, or narrow the accepted config contract and success criteria."
      - "Add tests covering blocklist glob enforcement and end-to-end rule generation from glob inputs."
---

# Phase 10: Network Sandboxing Verification Report

**Phase Goal:** Containers running on Linux are network-isolated via host-side iptables rules that survive process crashes and clean up after themselves
**Verified:** 2026-04-03T04:47:09Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Whitelist mode blocks all outbound traffic except explicitly allowed hostnames from inside the container | ✓ VERIFIED | `internal/network/rules.go` builds whitelist IP sets, `internal/network/firewall.go` inserts allow rules plus default `DROP`, and `go test ./internal/network/... -run 'TestBuildRuleSet|TestFirewall|TestCleanStaleRules' -count=1` passed. |
| 2 | Blocklist mode allows all outbound traffic except explicitly denied hostnames | ✓ VERIFIED | `internal/network/rules.go` populates `DeniedIPs` for blocklist mode and `internal/network/firewall.go` emits only `DROP` rules in blocklist mode; focused tests passed. |
| 3 | On macOS, `zone launch` with network filtering configured warns the user and falls back to mode=none | ✓ VERIFIED | `internal/docker/launch.go:353-355` prints the macOS warning and returns without firewall setup; `internal/docker/platform.go` reports `SupportsIPTables=false` on darwin. |
| 4 | iptables rules are tagged with the container ID and cleaned up when the container stops or `zone clean` runs | ✗ FAILED | Tagging exists, but cleanup is not durable across processes: `internal/docker/manager.go:367-372` only removes rules when `m.firewall != nil`, while `cmd/stop.go`, `cmd/destroy.go`, and `cmd/clean.go` create fresh managers or skip firewall cleanup entirely. |
| 5 | Hostname glob patterns (e.g., `*.anthropic.com`) match correctly in both whitelist and blocklist rules | ✗ FAILED | `internal/network/matcher.go` matches globs, but `internal/network/rules.go:50-52` rejects whitelist allow globs and `internal/network/rules.go:72-74` skips deny globs entirely, so glob rules are not enforced end-to-end. |
| 6 | Stale rules from a previous crashed Zone process are detected and removed on the next startup | ✓ VERIFIED | `internal/docker/launch.go:391-392` calls `listRunningZoneHashes` + `network.CleanStaleRules`, and `TestStaleRuleCleanupOnLaunch` / `TestCleanStaleRules` passed. |

**Score:** 4/6 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `internal/network/matcher.go` | Hostname glob matcher | ⚠️ PARTIAL | Substantive and tested, but only partially wired into rule generation. |
| `internal/network/rules.go` | Rule generation from network config | ⚠️ HOLLOW | Works for literal hostnames, but glob handling is incomplete for real firewall enforcement. |
| `internal/network/firewall.go` | iptables apply/remove/cache/refresh | ✓ VERIFIED | Applies tagged rules, removes by tag, writes `firewall.rules`, refreshes every 5 minutes. |
| `internal/docker/launch.go` | Launch-time firewall integration and stale cleanup | ✓ VERIFIED | Applies firewall after container start, adds proxy hosts, cleans stale rules on filtered launches. |
| `internal/docker/manager.go` | Stop/destroy cleanup | ⚠️ HOLLOW | Cleanup depends on in-memory `m.firewall` state that is absent in fresh CLI processes. |
| `cmd/clean.go` | User-facing cleanup path | ✗ MISSING LINK | Removes `.zone/` only; no firewall cleanup path exists. |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `internal/docker/launch.go` | `internal/network/firewall.go` | `network.NewFirewall(...).Apply(...)` after `ContainerStart` | ✓ WIRED | `setupFirewall()` runs after `createAndStart()` and applies the generated rules. |
| `internal/docker/launch.go` | `internal/network/firewall.go` | `StartRefresh` | ✓ WIRED | `fw.StartRefresh(refreshCtx, &netCfg)` is called after apply. |
| `internal/docker/launch.go` | `internal/network/firewall.go` | stale cleanup via `network.CleanStaleRules` | ✓ WIRED | Launch derives running hashes then calls stale cleanup before applying new rules. |
| `internal/docker/manager.go` | `internal/network/firewall.go` | `Stop`/`Destroy` cleanup | ✗ NOT_WIRED | `Stop()` only removes rules through `m.firewall`, which is never restored in a new manager created by CLI commands. |
| `cmd/clean.go` | firewall cleanup | cleanup before cache deletion | ✗ NOT_WIRED | `zone clean` deletes cache directly and never touches iptables rules. |
| `internal/network/rules.go` | `internal/network/matcher.go` | glob matching in generated rules | ⚠️ PARTIAL | Matcher is used for deny-first filtering, but globs do not become enforced blocklist/whitelist iptables rules. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| `internal/docker/launch.go` | `netCfg` / `rs` | `m.config.Network` → `resolveProxy` / `extractProxyHostnames` → `network.BuildRuleSet` | Yes | ✓ FLOWING |
| `internal/docker/manager.go` | `m.firewall` | Assigned only in `launch.go` (`m.firewall = fw`) | No persisted source for fresh `stop` / `destroy` processes | ✗ DISCONNECTED |
| `internal/network/rules.go` | `cfg.Deny` glob entries | `CompileAll` / `Compile`, then `if cp.IsGlob() { continue }` | No, blocklist glob inputs are discarded | ⚠️ STATIC |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Focused network sandbox tests | `go test ./internal/network/... -run 'TestBuildRuleSet|TestFirewall|TestCleanStaleRules' -count=1` | `ok github.com/peasant-labs/zone/internal/network 0.059s` | ✓ PASS |
| Focused docker firewall lifecycle tests | `go test ./internal/docker/... -run 'TestStaleRuleCleanupOnLaunch|TestStop_RunningContainer|TestDestroy_Full' -count=1` | `ok github.com/peasant-labs/zone/internal/docker 0.006s` | ✓ PASS |
| Full phase-related automated suite | `go test ./internal/network/... ./internal/docker/... ./tests/...` | all packages passed | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| `NET-01` | 10-02, 10-03 | Whitelist mode: deny all outbound, allow specific hostnames via iptables | ✓ SATISFIED | Whitelist `RuleSet` + default `DROP` in `firewall.go`; tests passed. |
| `NET-02` | 10-02, 10-03 | Blocklist mode: allow all outbound, deny specific hostnames | ✓ SATISFIED | Blocklist emits `DROP` rules only; tests passed. |
| `NET-03` | 10-02, 10-03 | Mode `none` applies no network restrictions | ✓ SATISFIED | `BuildRuleSet` returns `Mode: "none"`; `Firewall.Apply` becomes no-op. |
| `NET-04` | 10-01, 10-03 | Each container gets its own Docker bridge network | ✓ SATISFIED | `createContainer()` calls `createNetwork()` and `BridgeInterfaceName()` inspects the resulting network. |
| `NET-05` | 10-02, 10-03 | Tagged iptables rules for identification and cleanup | ✗ BLOCKED | Tagging works, but fresh-process `stop`/`destroy`/`clean` do not reliably remove rules. |
| `NET-06` | 10-02, 10-03 | Rules refreshed periodically every 5 min | ✓ SATISFIED | `refreshInterval = 5 * time.Minute`, `StartRefresh()` exists, tests cover refresh and cancellation. |
| `NET-07` | 10-01 | IPv6 disabled on container network | ✓ SATISFIED | `internal/docker/manager.go:186-188` sets `net.ipv6.conf.all.disable_ipv6=1`. |
| `NET-08` | 10-02 | Deny list takes priority over allow list in whitelist mode | ✓ SATISFIED | `BuildRuleSet()` compiles deny patterns first and skips matching allow literals; tests passed. |
| `NET-09` | 10-01, 10-03 | macOS warning + fallback to none | ✓ SATISFIED | `setupFirewall()` prints warning and returns nil on darwin. |
| `NET-10` | 10-02 | Firewall rules cached in `.zone/firewall.rules` | ✓ SATISFIED | `writeRulesCache()` writes cache; tests assert file contents. |
| `NET-11` | 10-01, 10-03 | `sudo iptables` only for firewall commands; fallback if unavailable | ✓ SATISFIED | `DefaultExecFunc()` shells out to `sudo iptables`; `CheckIPTablesAvailable()` warns/falls back. |
| `NET-12` | 10-01, 10-02 | Hostname glob matching for network rules | ✗ BLOCKED | Matcher compiles/matches globs, but `BuildRuleSet()` does not enforce globs in blocklist rules and rejects whitelist allow globs. |

All requirement IDs declared in phase 10 plan frontmatter were accounted for. No orphaned Phase 10 requirement IDs were found in `REQUIREMENTS.md`.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `internal/docker/manager.go` | 367 | Cleanup gated on in-memory `m.firewall != nil` | 🛑 Blocker | Fresh `zone stop` / `zone destroy` processes can skip host firewall cleanup entirely. |
| `cmd/clean.go` | 38 | Cache-only clean path | 🛑 Blocker | `zone clean` removes `.zone/` but leaves tagged iptables rules behind. |
| `internal/network/rules.go` | 72 | `if cp.IsGlob() { continue }` in deny rule generation | 🛑 Blocker | Blocklist glob patterns never produce firewall rules. |

### Human Verification Required

### 1. Linux end-to-end whitelist enforcement

**Test:** Launch a Linux container with `network.mode = "whitelist"`, allow one hostname, then attempt `curl` to both allowed and blocked hosts from inside the container.
**Expected:** Allowed host succeeds; blocked host fails.
**Why human:** Requires live Docker, sudo-capable iptables, and real network access.

### 2. Platform-specific fallback UX

**Test:** Run `zone launch` with filtering configured on macOS and on a rootless Docker setup.
**Expected:** Warning is shown and launch proceeds with unrestricted networking.
**Why human:** Requires host-specific environments not available in this verifier session.

### Gaps Summary

Phase 10 is substantively implemented, and most of the Linux firewall pipeline exists: platform detection, matcher primitives, rule generation, iptables application, stale-rule cleanup, refresh, and cache writing all exist and pass automated tests. However, the goal is not fully achieved because cleanup is not durable across real CLI lifecycles. `zone stop`, `zone destroy`, and `zone clean` run in fresh processes, but firewall teardown only happens when an in-memory `m.firewall` pointer is present from the current launch process. In addition, hostname glob support is only implemented at the matcher layer, not in end-to-end firewall rule enforcement.

These two gaps block full goal achievement: host-side rules do not reliably clean up after themselves across commands, and glob-based rules do not work as the roadmap contract requires.

---

_Verified: 2026-04-03T04:47:09Z_
_Verifier: the agent (gsd-verifier)_
