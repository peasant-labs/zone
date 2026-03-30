# Phase 7: Environment, Auth & Forwarding - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-30
**Phase:** 07-environment-auth-forwarding
**Areas discussed:** Env forwarding, Pre-launch validation, SSH agent forwarding, Auth config mounting, Hook execution, Port forwarding, Proxy support
**Mode:** Auto (all recommended defaults selected)

---

## Env Forwarding

| Option | Description | Selected |
|--------|-------------|----------|
| New `internal/docker/env.go` | Separate file for env collection, glob matching, validation | ✓ |
| Extend manager.go | Add env logic inline in manager.go | |

**User's choice:** [auto] New `internal/docker/env.go` (recommended default — separation of concerns)
**Notes:** Glob matching uses `filepath.Match` per spec §4.6. Non-required vars warn, don't error.

---

## Pre-launch Validation

| Option | Description | Selected |
|--------|-------------|----------|
| After lock, before build | Fail fast before any Docker operations | ✓ |
| Before lock acquisition | Validate even before getting exclusive access | |
| After build, before create | Allow build to succeed, validate at container creation | |

**User's choice:** [auto] After lock, before build (recommended default — fail fast)
**Notes:** Check harness RequiredEnvVars() + custom harness required_env. .env file vars count as set.

---

## SSH Agent Forwarding

| Option | Description | Selected |
|--------|-------------|----------|
| Warning + skip on macOS | Warn that domain sockets can't be mounted, proceed without SSH | ✓ |
| Error + abort on macOS | Block launch when SSH forwarding requested on macOS | |

**User's choice:** [auto] Warning + skip (recommended default — don't block launch)
**Notes:** Also warn+skip when SSH_AUTH_SOCK unset or socket missing on Linux.

---

## Auth Config Mounting

| Option | Description | Selected |
|--------|-------------|----------|
| Harness ConfigDirs() + mount_home_config | Mount harness-specific config dirs to .host suffix | ✓ |
| Generic home dir mount | Mount entire ~/.config read-only | |

**User's choice:** [auto] Harness ConfigDirs() (recommended default — spec §4.10 copy-on-start)
**Notes:** Phase 5 already has ConfigCopyCommands in entrypoint. Missing host dirs skipped with debug log.

---

## Hook Execution

| Option | Description | Selected |
|--------|-------------|----------|
| pre_build: abort on failure | Return error, stop launch | ✓ |
| pre_build: warn on failure | Log warning, continue launch | |

**User's choice:** [auto] Abort on failure (recommended default — pre_build may prepare required artifacts)

| Option | Description | Selected |
|--------|-------------|----------|
| post_stop: warn on failure | Log to stderr, don't fail stop | ✓ |
| post_stop: error on failure | Return error from Stop() | |

**User's choice:** [auto] Warn on failure (recommended default — container already stopped, best-effort)
**Notes:** Commands run on host via os/exec, sequentially, in repo dir.

---

## Port Forwarding

| Option | Description | Selected |
|--------|-------------|----------|
| Config-based ports only | Parse workspace.ports, defer --port flag to Phase 8 | ✓ |
| Config + CLI flag | Implement both config and --port/-P flag | |

**User's choice:** [auto] Config-based only (recommended default — CLI flags are Phase 8)
**Notes:** Parse "host:container" format, validate 1-65535, map to Docker PortBindings.

---

## Proxy Support

| Option | Description | Selected |
|--------|-------------|----------|
| Config priority + host fallback | Use config values if set, auto-detect from host env otherwise | ✓ |
| Config only | Only use explicitly configured proxy values | |
| Host only | Only auto-detect from host environment | |

**User's choice:** [auto] Config priority + host fallback (recommended default — spec §4.11)
**Notes:** Pass as build-args during build + container env vars at runtime. Check both UPPER and lower case variants.

---

## Claude's Discretion

- Internal helper function organization within env.go
- SSH_AUTH_SOCK mount path inside container
- .env file parsing approach
- Port parsing error messages
- Hook execution timeout
- Test strategy

## Deferred Ideas

- Ad-hoc --port/-P CLI flag → Phase 8
- Proxy hostname auto-add to whitelist → Phase 10
