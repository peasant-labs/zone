# Pitfalls Research

**Domain:** Go CLI / Docker workspace manager (Zone)
**Researched:** 2026-03-26
**Confidence:** HIGH

---

## Critical Pitfalls

### Pitfall 1: Terminal Raw Mode Not Restored on Panic or Error Path

**What goes wrong:**
When attaching to a container with TTY enabled, the host terminal must be put into raw mode. If the attach goroutine exits via a panic, a context cancellation, or an unexpected container stop — rather than a clean return — the terminal stays in raw mode. The user's shell is left in a broken state: no echo, no line buffering, arrow keys produce garbage. They must type `reset` blindly to recover.

**Why it happens:**
`terminal.MakeRaw()` returns an old-state value that must be passed to `terminal.Restore()`. Developers defer the restore in the happy path but forget to cover panic exits in spawned goroutines. BubbleTea does recover panics in its event loop but not in commands, compounding this.

**How to avoid:**
- Wrap the entire attach/exec session in a function that defers `terminal.Restore(oldState, os.Stdin.Fd())` as the very first statement.
- Use `recover()` in goroutines that handle stdin/stdout streaming to guarantee the restore fires.
- Test by deliberately killing the container mid-session and confirming the host terminal remains usable.

**Warning signs:**
- Test suite that only tests clean container exits.
- Attach code that does not have a deferred terminal restore before spawning IO goroutines.
- Manual `reset` appearing in developer notes or Slack history.

**Phase to address:** Container lifecycle phase (attach/exec/shell commands).

---

### Pitfall 2: Docker SDK Response Bodies Left Unclosed — Goroutine and Connection Leaks

**What goes wrong:**
`ImageBuild`, `ContainerLogs`, `ContainerAttach`, and `ImagePull` all return an `io.ReadCloser` (or `types.HijackedResponse`). If the caller fails to fully read AND close the body — even on error paths — the underlying HTTP/1.1 keep-alive connection is never returned to the pool. The goroutine draining the body blocks forever. Long-running CLI sessions accumulate leaked goroutines until the daemon connection pool is exhausted.

**Why it happens:**
The Docker SDK's response types look like structs, not HTTP connections. Developers `defer resp.Body.Close()` but forget to drain: closing without reading causes the transport to drop the connection instead of reusing it. On error returns the `defer` is never set up at all.

**How to avoid:**
- Always: `defer func() { io.Copy(io.Discard, resp.Body); resp.Body.Close() }()` immediately after any SDK call that returns a body.
- For `HijackedResponse`: call `.Close()` AND `.CloseWrite()` — missing `CloseWrite()` leaves the write side of the hijacked connection open.
- Add a `goleak` test that starts and cancels a build to detect leaks in CI.

**Warning signs:**
- `docker stats` showing daemon file-descriptor count climbing across CLI invocations.
- `runtime.NumGoroutine()` growing in integration tests.
- Any SDK call followed by `if err != nil { return }` before the `defer resp.Body.Close()`.

**Phase to address:** Foundation / Docker manager layer (first phase that wires SDK calls).

---

### Pitfall 3: PID 1 in Container Does Not Propagate Signals — Graceful Shutdown Broken

**What goes wrong:**
The generated container entrypoint script becomes PID 1. When `zone stop` sends SIGTERM (via `ContainerStop`), the kernel delivers it to PID 1. Shell scripts (bash/sh) do not forward signals to child processes by default — the LLM harness process never receives SIGTERM and gets SIGKILL after Docker's stop timeout. Any in-flight work is lost.

**Why it happens:**
Developers test `zone stop` only for fast-exiting containers. The signal gap only surfaces when the harness is mid-write or doing cleanup. Shell scripts as PID 1 are a known antipattern but are natural when templating entrypoints.

**How to avoid:**
- Use `exec` as the last line of every generated entrypoint script: `exec harness-binary "$@"` not `harness-binary "$@"`. This replaces the shell with the harness, giving it PID 1.
- Alternatively, add `tini` or `dumb-init` as the ENTRYPOINT prefix in generated Dockerfiles: `ENTRYPOINT ["/usr/bin/tini", "--", "/entrypoint.sh"]`.
- Verify by running `zone stop` with a harness that traps SIGTERM and writes to a file — confirm the file exists post-stop.

**Warning signs:**
- Entrypoint template that ends with `harness "$@"` (no `exec`).
- Generated Dockerfile has no `tini`/`dumb-init` and no `STOPSIGNAL` override.
- Container stop takes the full 10-second timeout every time.

**Phase to address:** Dockerfile template generation phase.

---

### Pitfall 4: iptables Rules Survive Process Crash — Orphaned Firewall State

**What goes wrong:**
Zone adds host-side iptables rules (DOCKER-USER chain) to enforce network sandboxing. If the Zone process crashes, is killed with SIGKILL, or the machine reboots mid-session, those rules remain in the kernel's netfilter tables. On next launch, Zone attempts to add the same rules again — either creating duplicates (breaking idempotency) or silently shadowing the old rules. Worse, rules for containers that no longer exist persist indefinitely, blocking traffic for unrelated containers that reuse the same IP.

**Why it happens:**
iptables rules are kernel state, not process state. There is no automatic cleanup when a process exits. Developers write "add rules on start, delete on stop" but the delete only runs on clean exits.

**How to avoid:**
- Name every rule with a comment tag derived from the container ID: `--comment "zone-<container-id>-<rule-hash>"`. On startup, audit the DOCKER-USER chain and remove any Zone-tagged rules whose container no longer exists.
- Use `iptables -C` (check) before `iptables -A` (append) to guard against duplicates.
- Implement a `zone clean` path that wipes all Zone-tagged rules unconditionally, and call it as part of `zone destroy` and `zone stop`.
- Store the set of installed rule identifiers in the `.zone/` cache alongside the file lock so they can be audited on next startup.

**Warning signs:**
- iptables management code with no startup audit step.
- Rules added using `iptables -A` without a prior `-C` check.
- No rule comment/tagging strategy in the implementation plan.
- `zone launch` not checking for stale rules from prior crashed sessions.

**Phase to address:** Network sandboxing phase (Linux-only, v1).

---

### Pitfall 5: iptables Modifications in Wrong Chain — Rules Bypassed via NAT

**What goes wrong:**
Docker uses NAT for container networking. Packets from containers traverse the FORWARD chain, not INPUT. Adding egress restriction rules to the INPUT chain has zero effect on outbound container traffic — the LLM agent can reach the internet unrestricted while the developer believes sandboxing is active.

**Why it happens:**
INPUT is the intuitive location for "block incoming/outgoing" rules. The Docker/iptables interaction is non-obvious: Docker's NAT masquerades container traffic through the bridge, bypassing INPUT entirely.

**How to avoid:**
- All Zone network rules must go in the DOCKER-USER chain (FORWARD table), which Docker specifically provides for user-managed rules and which runs before Docker's own FORWARD rules.
- Use `iptables -I DOCKER-USER 1 -s <container-ip> -d <blocked-range> -j DROP` for blocklist mode.
- Validate with `iptables -L DOCKER-USER -n -v` and a real egress test (curl from inside the container) before shipping.

**Warning signs:**
- Network filtering code targeting the INPUT chain.
- Whitelist/blocklist tests that only verify rule existence, not actual packet filtering.
- No integration test that runs curl/wget from inside a sandboxed container.

**Phase to address:** Network sandboxing phase.

---

### Pitfall 6: File Lock Not Released on Crash — Zone Directory Deadlock

**What goes wrong:**
Zone uses a file lock in `.zone/lock` to prevent concurrent `zone launch` invocations in the same repo. If the Zone process holding the lock is SIGKILL'd or the machine hard-reboots, the lock file persists. The next invocation of any Zone command blocks indefinitely on `flock()` or `TryLock()` returns false, leaving the user confused with a silent hang.

**Why it happens:**
Advisory file locks (flock) are released by the OS when the process exits — but only if the process exits cleanly or is killed. `flock`-based locks held via file descriptor are released on process death. However, if Zone uses a PID-file lock strategy (write PID, check on startup) rather than true flock, a stale PID file from a killed process blocks forever.

**How to avoid:**
- Use `github.com/gofrs/flock` with its `TryLock()` returning immediately, not a blocking `Lock()`. On failure, read the `.zone/lock` file for the owning PID and check if that PID is alive with `os.FindProcess` + `proc.Signal(0)`. If the process is gone, remove the stale lock and proceed.
- Include a `--force` flag on `zone launch` that removes stale locks after user confirmation.
- Test the stale-lock path in CI by acquiring a lock, killing the holder with SIGKILL, and verifying the next invocation recovers.

**Warning signs:**
- Lock acquisition code with no staleness check.
- Missing recovery path when lock holder PID is dead.
- `zone launch` hanging silently in manual testing after a `kill -9` of a prior session.

**Phase to address:** Foundation / cache directory phase.

---

### Pitfall 7: Stopped Container Owns Its Name — "Already In Use" on Relaunch

**What goes wrong:**
Zone uses deterministic container names derived from the repo path. When `zone stop` stops a container without removing it (by design, to preserve state), the stopped container still holds its name in the Docker daemon. A subsequent `zone launch` calling `ContainerCreate` with the same name gets: `Error response from daemon: Conflict. The container name "/zone-abc123" is already in use by container "def456"`. Without explicit handling, this surfaces as an opaque error to the user.

**Why it happens:**
Docker name uniqueness is global across all states — running and stopped. Developers test launch → stop → launch on a fresh system but forget the stopped-but-not-removed container is still registered.

**How to avoid:**
- The idempotent relaunch path must explicitly check container state via `ContainerInspect` before `ContainerCreate`. If the container exists and is stopped, call `ContainerStart` instead of `ContainerCreate`. If config has changed (hash mismatch), call `ContainerRemove` then `ContainerCreate`.
- Never surface the Docker "already in use" error raw — catch it by checking for `containertypes.ErrContainerAlreadyExists` or the error string and provide actionable guidance.

**Warning signs:**
- Container lifecycle code that calls `ContainerCreate` without first checking `ContainerInspect`.
- No integration test for the stop-then-relaunch path.
- User-reported "already in use" errors in early testing.

**Phase to address:** Container lifecycle phase (idempotent launch).

---

### Pitfall 8: BubbleTea Panics in Commands Leave Terminal in Raw Mode

**What goes wrong:**
BubbleTea recovers panics that occur in the event loop (the `Update` method), but panics in `tea.Cmd` goroutines propagate unhandled to the runtime. This crashes the process without running BubbleTea's terminal cleanup, leaving the host terminal in raw mode with no cursor. The user sees a panic trace and must type `reset` to recover.

**Why it happens:**
BubbleTea's model design encourages moving expensive work into Cmds. Developers put Docker API calls, file I/O, and network operations in Cmds without wrapping them in `recover()`. Any nil pointer dereference or unexpected error in those paths causes terminal corruption.

**How to avoid:**
- Wrap every `tea.Cmd` function body with a `defer func() { if r := recover(); r != nil { /* send error Msg */ } }()`.
- Add a top-level `defer terminal.Restore(...)` in `main()` before starting the BubbleTea program, as a belt-and-suspenders fallback.
- Validate all Docker API responses before returning them in Msgs — convert errors to typed Msg values rather than panicking.

**Warning signs:**
- Cmd functions that perform Docker SDK calls without `recover()` wrappers.
- Missing `defer terminal.Restore` in main before `p.Run()`.
- Any `.(*SomeType)` type assertion in a Cmd without a `, ok` check.

**Phase to address:** TUI / BubbleTea integration phase.

---

### Pitfall 9: BubbleTea TTY Detection Falls Through to TUI in Non-Interactive Contexts

**What goes wrong:**
Zone must detect whether it is running in a TTY to decide between BubbleTea TUI and plain-text output. Naive TTY detection using `os.Stdin.Fd()` and `term.IsTerminal()` returns false negatives in: CI environments with pseudo-TTYs, `docker exec` sessions, piped invocations, and `zone exec -- bash -c "..."` sub-shells. When detection fails, Zone either shows BubbleTea in a broken non-TTY environment or falls back to plain-text when the user expects a TUI.

**Why it happens:**
`term.IsTerminal()` checks stdin only. In `zone exec` and `zone logs`, the relevant TTY is stdout. In scripting use cases, stdout may be piped while stdin is a TTY. `--plain` flag exists but users don't know to pass it.

**How to avoid:**
- Check `term.IsTerminal(int(os.Stdout.Fd()))` for output-oriented commands and `term.IsTerminal(int(os.Stdin.Fd()))` for interactive commands.
- Auto-detect: if TERM is unset, CI=true, or NO_COLOR is set, force plain mode.
- Respect the `--plain` flag and `ZONE_PLAIN=1` environment variable for explicit override.
- Test TTY detection by running Zone in a subshell with `| cat` and confirming plain output.

**Warning signs:**
- TTY detection code that only checks stdin.
- No CI environment in test matrix where Zone is run non-interactively.
- BubbleTea rendering control characters appearing in CI log output.

**Phase to address:** TUI / BubbleTea integration phase (alongside foundation).

---

### Pitfall 10: Docker Socket Path Differs on macOS — Silent Connection Failure

**What goes wrong:**
Recent Docker Desktop on macOS moves the socket from `/var/run/docker.sock` to `$HOME/.docker/run/docker.sock`. Docker Desktop creates a symlink at `/var/run/docker.sock` only if the user opts in during installation. If the symlink is absent and DOCKER_HOST is unset, the Go SDK's `client.NewClientWithOpts(client.FromEnv)` silently falls back to the default path, which does not exist. The CLI fails with a confusing "connection refused" or "no such file or directory" error that users attribute to Zone rather than Docker configuration.

**Why it happens:**
Docker SDK's `FromEnv` respects DOCKER_HOST if set but falls back to a hardcoded default. macOS Docker Desktop's socket relocation is installation-option-dependent, making it non-deterministic.

**How to avoid:**
- After client creation, immediately call `cli.Ping(ctx)` to validate the connection. On failure, check common socket paths in order: `$DOCKER_HOST`, `/var/run/docker.sock`, `$HOME/.docker/run/docker.sock`, and report which path was tried and failed.
- Provide a `zone validate` subcommand that checks Docker connectivity as its first step.
- Include macOS Docker socket detection in the `zone doctor` / `zone validate` output.

**Warning signs:**
- No startup ping/connectivity check before any Docker SDK calls.
- Error messages that surface raw `dial unix /var/run/docker.sock: no such file or directory`.
- macOS excluded from test environments.

**Phase to address:** Foundation / Docker manager phase.

---

### Pitfall 11: TOML Strict Decode Misses Merged Config Key Conflicts

**What goes wrong:**
Zone uses two-tier TOML config (global `~/.config/zone/config.toml` + per-repo `zone.toml`) with a merge strategy. BurntSushi/toml's strict decode (`MetaData.Undecoded()`) reports unknown keys in a single file but cannot detect keys that are valid individually but semantically conflicting when merged (e.g., `network.mode = "whitelist"` in global but `network.allowed_hosts` absent in local). The merge code silently wins, and the user gets unexpected behavior.

**Why it happens:**
Strict decode validates against the Go struct schema, not against cross-file invariants. Merge logic that does field-by-field overlay does not validate the final merged state against semantic rules.

**How to avoid:**
- Parse each file independently with strict decode (catch unknown keys early).
- Apply the merge algorithm to produce a single merged `Config` struct.
- Run a semantic validation pass on the merged struct: check that mode/field combinations are coherent, required fields are present given the chosen mode, and mutually exclusive settings are not both set.
- Use `zone validate` to expose this final merged validation to users before `zone launch`.

**Warning signs:**
- Merge code that does field overlay without a subsequent validation step.
- No test for conflicting global/local config combinations.
- `zone validate` that only checks file syntax, not merged semantics.

**Phase to address:** Config parsing / validation phase.

---

### Pitfall 12: Auth Config Copy-on-Start Leaks Credentials Into Container Image

**What goes wrong:**
Zone's copy-on-start strategy copies host auth config files (e.g., `~/.config/claude/config.json`) into the container at start time as writable files. If these copied files end up in a Docker commit, an image push, or a debug export, credentials leak. Even without a push, the files persist in the container's writable layer and in any `docker export` or `docker commit` snapshot.

**Why it happens:**
Developers treat the container's writable layer as ephemeral but forget it persists until `ContainerRemove`. Auth config files are often large JSON blobs that look innocuous in a `docker inspect` but contain API keys.

**How to avoid:**
- Copy auth files into a tmpfs mount inside the container (`--tmpfs /home/user/.config/claude:rw,noexec,nosuid`) so they never touch the writable layer. If tmpfs is unavailable, copy to a path that is explicitly listed in `.dockerignore` equivalents and warn the user if a `docker commit` is attempted.
- Add a startup check: if the container already has an auth file from a previous copy that is older than the source, refresh it.
- Document clearly in `zone destroy` that auth files in the container are wiped.

**Warning signs:**
- Auth files copied to regular container filesystem paths (not tmpfs).
- No warning when user runs `docker commit` or `docker export` on a Zone container.
- Auth copy code without a corresponding cleanup path in `zone destroy`.

**Phase to address:** Auth config / harness integration phase.

---

### Pitfall 13: go:embed Template Path Fails When Binary Runs From Unexpected CWD

**What goes wrong:**
Zone uses `go:embed` to bundle Dockerfile and entrypoint templates. `embed.FS` paths are relative to the package directory at compile time — this is correct and robust. However, developers sometimes mix `embed.FS` with `os.Open` calls using paths relative to the working directory. When `zone launch` is invoked from a subdirectory or via a symlink, `os.Open("templates/Dockerfile.tmpl")` fails with "no such file or directory" while `embed.FS` access succeeds.

**Why it happens:**
It is tempting to fall back to `os.Open` for development (to hot-reload templates without recompiling). If this fallback path is accidentally left in or conditionally triggered, it breaks in any non-development invocation.

**How to avoid:**
- Use only `embed.FS` for templates in production code paths. No `os.Open` fallback.
- If a development hot-reload mode is desired, gate it behind a build tag (`//go:build dev`) that is never included in release builds.
- Test the distributed binary (not `go run`) against a repo in `/tmp` to catch CWD-sensitive paths.

**Warning signs:**
- Template loading code with `os.Open` alongside `//go:embed` directives.
- Template tests that only run from the repo root.
- `go run ./cmd/zone launch` working but installed binary failing.

**Phase to address:** Dockerfile template generation phase.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Raw `docker run` exec instead of SDK | Faster to implement shell commands | No structured error handling, no SDK integration, breaks JSON output mode | Never for Zone — SDK is required |
| Blocking `iptables` exec call without timeout | Simpler code | Hangs Zone process if iptables is slow (common under heavy load) | Never — always use `context` with timeout |
| Skipping startup Docker ping | Faster launch | Confusing failures deep in command execution | Never — validate early |
| Hardcoded `/var/run/docker.sock` | No socket detection logic | Fails on macOS Docker Desktop without symlink | Never — use SDK's `FromEnv` + fallback probing |
| Single-file config (no global/local split) | Simpler parse logic | No cross-repo defaults, users must reconfigure every repo | MVP only if scope-reduced, not for Zone v1 |
| `iptables -A` without `-C` guard | Fewer syscalls | Duplicate rules accumulate, network behavior breaks | Never for idempotent operations |
| Skip `exec` in entrypoint, use plain shell | Simpler template | Signal propagation broken, graceful shutdown fails | Never — critical for harness lifecycle |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Docker SDK `ImageBuild` | Not reading/closing response body | `defer io.Copy(io.Discard, resp.Body); defer resp.Body.Close()` immediately after call |
| Docker SDK `ContainerAttach` | Only closing `HijackedResponse`, not calling `CloseWrite()` | Call both `resp.Close()` and `resp.CloseWrite()` after attach session ends |
| Docker SDK `ContainerExecAttach` | Using `Detach: true` while expecting to read output | Only use `Detach: false` when you need to consume exec output |
| BubbleTea + Docker streaming | Sending messages to the BubbleTea program after `p.Quit()` is called | Drain background goroutines before or guard sends with a `done` channel |
| iptables + Docker daemon restart | Saving iptables state then restoring, wiping Docker's own chains | Only write to DOCKER-USER chain; never use `iptables-restore` to overwrite all chains |
| SSH agent socket on macOS | Mounting `$SSH_AUTH_SOCK` into container directly | macOS domain sockets cannot be bind-mounted; use `SSH_AUTH_SOCK` detection and warn user on macOS |
| TOML strict decode | Calling `DecodeFile` without checking `MetaData.Undecoded()` | Always check `Undecoded()` and return error listing unknown keys with edit-distance suggestions |
| Docker socket on macOS | Hardcoding `/var/run/docker.sock` | Probe `$DOCKER_HOST`, then `/var/run/docker.sock`, then `$HOME/.docker/run/docker.sock` |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Blocking on `ImageBuild` response without streaming | UI freezes during build, no progress feedback | Stream the response body through `jsonmessage.DisplayJSONMessagesStream` in a goroutine | Every build > 5 seconds |
| Calling `ContainerList` on every status poll | High daemon CPU for `zone ls` in watch mode | Cache container state, use `ContainerInspect` by ID for known containers | At > 5 concurrent zones |
| iptables rule count growth (no deduplication) | Slower packet processing, `iptables -L` output grows unbounded | Audit and clean Zone-tagged rules on startup; never add without checking | After ~100 launch/stop cycles without `destroy` |
| Synchronous file lock check with sleep poll | Zone commands that appear frozen | Use `flock` advisory lock with `TryLock()` and immediate failure + helpful message | Every invocation when another zone is active |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Mounting host Docker socket into container (`/var/run/docker.sock`) | LLM agent gets full Docker API access, can escape sandbox | Never mount the Docker socket; Zone manages Docker from the host process only |
| `CAP_NET_ADMIN` inside container for network filtering | LLM agent can disable its own firewall via `iptables -F` | Enforce network rules host-side only via DOCKER-USER chain; no `CAP_NET_ADMIN` in container |
| `--privileged` flag in generated `ContainerCreate` | All capabilities granted, seccomp disabled, full host access | Use explicit capability grants only; default to `--cap-drop=ALL` + specific adds |
| Auth files copied to container filesystem (non-tmpfs) | Credentials persist in writable layer, visible in `docker export` | Use tmpfs for auth directories; never let credentials touch the image layer |
| Scoped sudo without allowlist validation | LLM agent injects `;rm -rf /` into package manager args | Validate sudoers rule to permit only exact package manager binaries with no shell interpolation |
| Template injection via user-supplied harness name | Malicious harness name injects shell commands into generated Dockerfile | Sanitize all user-supplied values before templating; use `text/template` with explicit variable substitution, not `fmt.Sprintf` on raw input |
| Symlink escape in dangerous mount blocking | Config references `../../etc/passwd` via symlink chain | Use `filepath.EvalSymlinks` before checking mount paths against blocklist |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Surfacing raw Docker error messages | Users see "Error response from daemon: ..." with no Zone context | Wrap all Docker errors with `fmt.Errorf("zone: %w", ...)` and add human-readable context |
| Silent hang on file lock contention | User runs `zone launch` in two terminals; second one blocks with no output | Immediately print "Another zone session is active in this repo (PID: X)" and exit non-zero |
| No progress during image build | User waits silently for 30-120 seconds, assumes Zone crashed | Stream build output to BubbleTea progress view or plain-text in non-TTY mode |
| Confusing "name already in use" error on relaunch | User who ran `zone stop` cannot relaunch without manual `docker rm` | Handle idempotent relaunch transparently; never expose this Docker error |
| `--plain` flag buried in help | Users in CI struggle with BubbleTea control characters in logs | Auto-detect CI environment (CI=true, NO_COLOR, no TTY) and default to plain mode |
| Config validation errors after `zone launch` starts | Build fails partway through because config was never validated | Run full config validation (parse + merge + semantic check) before any Docker operation |

---

## "Looks Done But Isn't" Checklist

- [ ] **Idempotent launch:** Does `zone launch` handle stopped-but-existing container without error? Verify with stop → launch → launch sequence.
- [ ] **Terminal restore:** After killing a container mid-attach, is the host terminal still usable? Run `echo hello` immediately after to confirm.
- [ ] **Signal propagation:** Does `zone stop` gracefully terminate the harness (not just SIGKILL)? Check with a harness that writes a file on SIGTERM receipt.
- [ ] **iptables cleanup:** After `zone destroy`, are all Zone-tagged iptables rules removed? Run `iptables -L DOCKER-USER -n` to verify.
- [ ] **Stale lock recovery:** After `kill -9` on a running zone process, does the next `zone launch` recover? Verify without manual intervention.
- [ ] **Build response drain:** Under `go tool pprof` / `goleak`, does a cancelled build leave no goroutines blocked on response body reads?
- [ ] **macOS socket detection:** On a macOS machine without the Docker symlink, does Zone connect and give a clear error? Test with `DOCKER_HOST=""`.
- [ ] **Non-TTY mode:** Piping `zone ls | cat` produces clean JSON or plain text with no ANSI escape codes.
- [ ] **Config merge validation:** A global config with `network.mode = "whitelist"` and a local config with no `allowed_hosts` produces an error, not silent misconfiguration.
- [ ] **Auth file cleanup:** After `zone destroy`, auth files copied into the container are gone. Verify with `docker inspect` on a new container from the same image.

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Terminal left in raw mode | LOW | User runs `reset` or `stty sane`; no data loss |
| Goroutine/connection leak | MEDIUM | Restart Zone; if daemon-level, `docker restart` the Docker daemon; identify and fix the unclosed body |
| Orphaned iptables rules | MEDIUM | Run `zone clean` (must be implemented); fallback: manual `iptables -D DOCKER-USER` for each Zone-tagged rule |
| Stale file lock | LOW | Delete `.zone/lock` manually or use `zone launch --force`; no container state lost |
| Container name collision | LOW | Run `zone destroy` then `zone launch`; or `docker rm <name>` then relaunch |
| BubbleTea terminal corruption from panic | LOW | Run `reset`; investigate panic stack trace; no persistent state damage |
| iptables in wrong chain (egress not blocked) | HIGH | Rewrite network filtering layer; requires testing all egress paths; potential security incident if exploited |
| Auth files in container writable layer | HIGH | `zone destroy` to remove container; rotate compromised credentials; redesign to use tmpfs |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Terminal raw mode not restored | Container lifecycle (attach/exec) | Kill container mid-attach; confirm terminal usable |
| SDK response body leaks | Foundation / Docker manager | `goleak` test on build, pull, attach operations |
| PID 1 signal propagation | Dockerfile template generation | `zone stop` with SIGTERM-trapping harness |
| iptables rules survive crash | Network sandboxing phase | SIGKILL zone process; confirm rules cleaned on next start |
| iptables rules in wrong chain | Network sandboxing phase | Run curl from inside container against blocked host |
| File lock not released on crash | Foundation / cache directory | `kill -9` zone; confirm next invocation proceeds |
| Stopped container name collision | Container lifecycle (idempotent launch) | stop → launch sequence without destroy |
| BubbleTea panic corrupts terminal | TUI integration phase | Inject panic in a Cmd; confirm terminal recovery |
| BubbleTea TTY detection failure | TUI integration phase (+ foundation) | Run `zone ls \| cat`; check for escape codes |
| Docker socket path on macOS | Foundation / Docker manager | Test on macOS without `/var/run/docker.sock` symlink |
| TOML merge config validation | Config parsing / validation phase | Conflicting global + local config must error, not silently win |
| Auth config credential leakage | Auth / harness integration phase | `docker export` zone container; confirm no credentials |
| go:embed CWD sensitivity | Dockerfile template generation | Run installed binary from `/tmp` directory |

---

## Sources

- [PID 1 Signal Handling in Docker - Peter Malmgren](https://petermalmgren.com/signal-handling-docker/)
- [Go Docker SDK Raw Terminal Ctrl+C Handling - addshore](https://addshore.com/2021/01/go-docker-sdk-raw-terminal-ctrlc-handling/)
- [Tips for Building BubbleTea Programs](https://leg100.github.io/en/posts/building-bubbletea-programs/)
- [Docker and iptables - You may do it wrong (ntk148v)](https://ntk148v.github.io/posts/docker-iptables/)
- [BubbleTea Issue #860: Automatically open terminal when stdout is not a TTY](https://github.com/charmbracelet/bubbletea/issues/860)
- [BubbleTea Issue #234: Recover panics from within cmds goroutines](https://github.com/charmbracelet/bubbletea/issues/234)
- [CVE-2025-64329: containerd CRI Attach Goroutine Leak](https://windowsforum.com/threads/cve-2025-64329-patch-containerd-cri-attach-goroutine-leak-dos.392772/)
- [Docker orphaned iptables rules issue #3376](https://github.com/distribution/distribution/issues/3376)
- [iptables forwarding rules not cleaned up on container removal #42029](https://github.com/moby/moby/issues/42029)
- [Docker macOS socket path - apocas/docker-modem issue #156](https://github.com/apocas/docker-modem/issues/156)
- [Cross-platform file locking with Go - Chrono](https://www.chronohq.com/blog/cross-platform-file-locking-with-go)
- [go-iptables - coreos/go-iptables](https://github.com/coreos/go-iptables)
- [Docker Packet Filtering and Firewalls - Official Docs](https://docs.docker.com/engine/network/packet-filtering-firewalls/)
- [Building Docker Images in Go - Nearform](https://www.nearform.com/blog/building-docker-images-in-go/)
- [Hunting Zombie Processes in Go and Docker - Stormkit](https://www.stormkit.io/blog/hunting-zombie-processes-in-go-and-docker)

---
*Pitfalls research for: Go CLI Docker workspace manager (Zone)*
*Researched: 2026-03-26*
