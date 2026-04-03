# Phase 10: Network Sandboxing - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-03
**Phase:** 10-network-sandboxing
**Areas discussed:** sudo behavior, refresh goroutine lifecycle, rule cleanup strategy, nftables compatibility, proxy auto-allowlisting
**Mode:** Auto (--auto flag)

---

## Sudo Behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Test upfront, warn and fallback | Test `sudo iptables -L -n` at launch, warn and fall back to mode=none if unavailable | ✓ |
| Prompt user interactively | Ask user for sudo password when needed | |
| Fail hard | Error and refuse to launch if sudo unavailable | |

**User's choice:** [auto] Test upfront, warn and fallback (recommended default)
**Notes:** Per spec §4.9 — "If sudo is unavailable, warn and fall back to mode=none." Non-blocking fallback preserves the core value of zero-friction launch.

---

## Refresh Goroutine Lifecycle

| Option | Description | Selected |
|--------|-------------|----------|
| Background goroutine tied to container | Start on firewall apply, stop on container stop via context cancellation | ✓ |
| Cron-like timer from zone process | Independent timer that checks all containers | |
| No refresh (static rules only) | Apply once at launch, no updates | |

**User's choice:** [auto] Background goroutine tied to container lifecycle (recommended default)
**Notes:** Per spec §4.9 — "Rules are periodically refreshed (every 5 minutes) by re-resolving hostnames in a background goroutine." Context cancellation from signal handling chain provides clean shutdown.

---

## Rule Cleanup Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Launch + stop + clean + destroy | Clean stale rules on every launch, remove own rules on stop/clean/destroy | ✓ |
| Stop only | Remove rules only when container stops | |
| Manual cleanup via zone clean | Only clean rules when user explicitly runs zone clean | |

**User's choice:** [auto] Launch + stop + clean + destroy (recommended default)
**Notes:** Comprehensive cleanup ensures stale rules from crashed processes never accumulate. Cross-referencing zone-tagged rules against running containers on every launch catches orphans.

---

## nftables Compatibility

| Option | Description | Selected |
|--------|-------------|----------|
| Use iptables CLI as-is | Rely on iptables-nft compatibility layer, test with `sudo iptables -L -n` at startup | ✓ |
| Detect and use nft directly | Detect nftables backend and issue nft commands | |
| Require legacy iptables | Only support legacy iptables, error on nftables | |

**User's choice:** [auto] Use iptables CLI as-is (recommended default)
**Notes:** Modern distros (Debian 10+, Ubuntu 20.04+, Fedora 32+) provide iptables-nft that translates iptables commands to nftables. Testing at startup confirms the compatibility layer works. No need to directly use nft CLI in Phase 1.

---

## Proxy Auto-Allowlisting

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-add proxy hostname to runtime allow set | Extract hostname from proxy URL, add to resolved allow set before generating rules | ✓ |
| Require user to add proxy to allow list | User must manually add proxy hostname to config | |
| Skip proxy entirely in whitelist mode | Proxy traffic blocked like everything else not in allow list | |

**User's choice:** [auto] Auto-add proxy hostname to runtime allow set (recommended default)
**Notes:** Per spec §4.11 — "When whitelist mode is active, the proxy server hostname is automatically added to the allow list." This is a runtime addition only — the user's config is not modified.

---

## Claude's Discretion

- iptables chain naming convention
- DNS resolution approach (net.LookupHost vs custom resolver)
- Goroutine synchronization patterns
- Test strategy for iptables functionality
- Error message exact wording
- Firewall.rules file format

## Deferred Ideas

- DNS proxy sidecar for cross-platform filtering → Phase 2
- macOS network filtering → Phase 2
- Advanced glob patterns (`**`) → Phase 2
- Per-rule logging/audit trail → backlog
- Real-time rule notifications in TUI → backlog
