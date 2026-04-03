# Roadmap: Zone

## Overview

Zone is built in 10 focused phases ordered by the compiler-enforced dependency graph: config and cache are the foundation everything else reads; templates and harnesses build on the config types; the Docker lifecycle orchestrator consumes all of those; environment/auth forwarding layers over the lifecycle; CLI commands wire it all together; TUI adds the interactive layer; and network sandboxing ships last because it is Linux-only and requires a running container to test. Each phase delivers a coherent, independently verifiable capability before the next begins.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Project Scaffold** - Go module, package skeleton, CI pipeline, and GoReleaser config (completed 2026-03-27)
- [x] **Phase 2: Config Foundation** - Two-tier TOML config, strict decode, merge strategy, validation (completed 2026-03-27)
- [ ] **Phase 3: Cache & State** - .zone/ directory, config-hash tracking, file locking, build logs
- [x] **Phase 4: Template System** - Dockerfile/entrypoint/RC templates, go:embed, deterministic naming (completed 2026-03-29)
- [x] **Phase 5: Harness Plugin System** - Interface, registry, claude-code impl, custom harness, stubs (completed 2026-03-29)
- [x] **Phase 6: Docker Lifecycle Core** - Idempotent build/launch/stop/destroy, security hardening (completed 2026-03-29)
- [x] **Phase 7: Environment, Auth & Forwarding** - Env vars, SSH agent, auth copy, proxy, ports, hooks (completed 2026-03-30)
- [ ] **Phase 8: CLI Commands & DX** - All commands wired, exit codes, signals, JSON output, aliases
- [ ] **Phase 9: TUI Layer** - BubbleTea init wizard, build progress, status view, log viewer
- [ ] **Phase 10: Network Sandboxing** - Host-side iptables whitelist/blocklist, hostname glob, cleanup

## Phase Details

### Phase 1: Project Scaffold
**Goal**: Users can clone the repo, run `go build ./...` successfully, and CI passes on every commit
**Depends on**: Nothing (first phase)
**Requirements**: DX-10
**Success Criteria** (what must be TRUE):
  1. `go build ./...` succeeds from a clean checkout with no manual setup
  2. `go test ./...` runs (even with zero tests) and exits 0
  3. GoReleaser config exists and `goreleaser check` passes
  4. CI workflow runs on push and reports pass/fail
**Plans**: 2 plans
Plans:
- [ ] 01-01-PLAN.md — Go module init, Cobra CLI skeleton, all internal/pkg/test stubs
- [ ] 01-02-PLAN.md — GoReleaser, golangci-lint, Makefile, CI workflows

### Phase 2: Config Foundation
**Goal**: Users can write zone.toml and ~/.config/zone/config.toml and have them merged, validated, and surfaced clearly on errors
**Depends on**: Phase 1
**Requirements**: CFG-01, CFG-02, CFG-03, CFG-04, CFG-05, CFG-06, CFG-07, CFG-08, CFG-09, CFG-19
**Success Criteria** (what must be TRUE):
  1. A minimal `zone.toml` with `version = 1` and `harness = "claude-code"` parses without error
  2. Global config and per-repo config merge correctly: scalars overridden by repo, lists unioned
  3. An unknown config key produces an error message with a Levenshtein edit-distance suggestion
  4. A dangerous mount path (e.g., docker.sock) is blocked with a clear error including the resolved symlink
  5. `zone config` prints the merged config with each value annotated as global or repo source
**Plans**: 3 plans
Plans:
- [ ] 02-01-PLAN.md — Config type structs, TOML parsing, dependency install
- [ ] 02-02-PLAN.md — Two-tier merge algorithm, validation, Levenshtein, dangerous mounts, tests
- [ ] 02-03-PLAN.md — Wire zone config and zone validate Cobra commands

### Phase 3: Cache & State
**Goal**: Zone reliably tracks image/container/network IDs, detects config changes, and safely handles concurrent invocations
**Depends on**: Phase 2
**Requirements**: CAC-01, CAC-02, CAC-03, CAC-04, CAC-05, CAC-06
**Success Criteria** (what must be TRUE):
  1. `.zone/` directory is created on first launch and `.gitignore` is updated automatically
  2. Running two `zone` commands concurrently produces a clear lock-contention error (exit code 5) rather than corruption
  3. Changing zone.toml changes the computed config hash, triggering a rebuild on next launch
  4. Build logs are stored and readable via `zone logs --build`
**Plans**: 3 plans
Plans:
- [ ] 03-01-PLAN.md — Cache struct, directory management, atomic writes, SHA256 hash computation
- [ ] 03-02-PLAN.md — flock-based locking, gitignore management, build log storage, zone clean command
- [ ] 03-03-PLAN.md — Exit code 5 translation for ErrLockContention in main.go (gap closure)

### Phase 4: Template System
**Goal**: Zone generates correct, runnable Dockerfiles and entrypoints from embedded templates, with deterministic container naming
**Depends on**: Phase 2
**Requirements**: DOC-01, DOC-02, DOC-03, DOC-04, DOC-05, DOC-06, DOC-07, DOC-13, DOC-14, DOC-15, DOC-16
**Success Criteria** (what must be TRUE):
  1. Generated Dockerfile builds successfully with `docker build` and produces a non-root `zone` user matching host UID
  2. Generated entrypoint ends with `exec <harness-binary>` so PID 1 receives all signals correctly
  3. Container and network names are deterministic: same repo path always produces same name across machines
  4. Git safe.directory and user.name/email are set correctly inside the container
  5. Running as root UID (CI environment) skips user creation without error
**Plans**: 2 plans
Plans:
- [ ] 04-01-PLAN.md — Embed migration (FS to string vars), template file content from spec, hash.go fix, naming + security flags
- [ ] 04-02-PLAN.md — Render functions (Dockerfile/entrypoint/shellrc), platform detection, git identity, tests

### Phase 5: Harness Plugin System
**Goal**: Claude Code launches correctly inside a Zone container; custom harnesses work via config; unimplemented harnesses fail with clear messages
**Depends on**: Phase 4
**Requirements**: HAR-01, HAR-02, HAR-03, HAR-04, HAR-05, HAR-06, HAR-07, HAR-08, HAR-09, HAR-10
**Success Criteria** (what must be TRUE):
  1. `zone launch` with `harness = "claude-code"` starts Claude Code inside the container
  2. A custom harness defined with `install_commands` and `entrypoint_command` installs and runs correctly
  3. `zone launch` with `harness = "aider"` (or other stub) fails with a descriptive "not yet implemented" error
  4. An unknown harness-specific config key produces a per-harness validation error, not a generic parse error
  5. `zone launch -- -p "write tests"` passes the prompt flag through to the harness automatically
**Plans**: 3 plans
Plans:
- [ ] 05-01-PLAN.md — Harness interface, BaseHarness, registry, Get(), ClaudeCode full implementation
- [ ] 05-02-PLAN.md — Stub harnesses (opencode, gemini-cli, aider, codex-cli), custom harness, per-harness validation
- [ ] 05-03-PLAN.md — Bridge functions (BuildDockerfileData, BuildEntrypointData, BuildShellRCData)

### Phase 6: Docker Lifecycle Core
**Goal**: Users can launch, reattach, stop, restart, and destroy containers with full idempotency and security hardening
**Depends on**: Phase 3, Phase 5
**Requirements**: DOC-08, DOC-09, DOC-10, DOC-11, DOC-12, CFG-20, CLI-03, CLI-04, CLI-05, CLI-06, CLI-07, CLI-08, CLI-09, CLI-10, CLI-11, CLI-15, CLI-16
**Success Criteria** (what must be TRUE):
  1. Running `zone launch` twice reattaches to the running container rather than creating a duplicate
  2. Changing zone.toml and running `zone launch` warns the user about the config change and prompts restart
  3. Container runs with `no-new-privileges`, all capabilities dropped, and pids limit enforced
  4. `zone stop` removes the container and network; `zone destroy` additionally removes the image and cache
  5. `zone launch --harness claude-code` works in a directory with no zone.toml (zero-config quickstart)
  6. `zone launch --headless -p "task"` runs the agent detached and returns to the shell immediately
**Plans**: 4 plans
Plans:
- [x] 06-01-PLAN.md — Docker SDK install, client interface, Manager constructor, build pipeline, network, container creation
- [x] 06-02-PLAN.md — Launch state machine, config change detection, headless mode, zero-config quickstart
- [x] 06-03-PLAN.md — Stop, Destroy, RemoveImage cleanup methods
- [x] 06-04-PLAN.md — Wire all 8 Cobra commands (launch, join, exec, shell, build, stop, restart, destroy)

### Phase 7: Environment, Auth & Forwarding
**Goal**: Secrets, credentials, and runtime configuration reach the container correctly without being persisted in the image
**Depends on**: Phase 6
**Requirements**: CFG-10, CFG-11, CFG-12, CFG-13, CFG-14, CFG-15, CFG-16, CFG-17, CFG-18
**Success Criteria** (what must be TRUE):
  1. Environment variables matching glob patterns (e.g., `AWS_*`) are forwarded into the container
  2. A missing required env var causes `zone launch` to fail before Docker build starts, with a clear error listing which var is absent
  3. SSH agent forwarding mounts the host socket into the container; keys are never written to disk
  4. Auth config files are available read-write inside the container while the host copy is unchanged
  5. `pre_build` and `post_stop` hook commands execute at the correct lifecycle points
**Plans**: 3 plans
Plans:
- [x] 07-01-PLAN.md — Env collection, glob matching, .env parsing, required env validation (env.go)
- [x] 07-02-PLAN.md — Port binding parsing, proxy resolution, hook execution (ports.go, proxy.go, hooks.go)
- [x] 07-03-PLAN.md — Wire all helpers into Manager: buildMounts, createContainer, Launch, Stop, buildImage

### Phase 8: CLI Commands & DX
**Goal**: All 21 CLI commands work end-to-end with correct exit codes, signal handling, JSON output, and inline help
**Depends on**: Phase 7
**Requirements**: CLI-01, CLI-02, CLI-12, CLI-13, CLI-14, CLI-17, CLI-18, CLI-19, CLI-20, CLI-21, DX-01, DX-02, DX-03, DX-04, DX-05, DX-06, DX-07, DX-08, DX-09
**Success Criteria** (what must be TRUE):
  1. Every command has `--help` output with 2-4 usage examples
  2. `zone status --json` and `zone ls --json` produce valid, parseable JSON
  3. Pressing Ctrl+C during `zone launch` sends SIGINT to the harness process but leaves the container alive
  4. Every error message includes a remediation hint; exit codes map to error categories 0-6
  5. `zone validate` catches config errors without touching Docker or starting any containers
  6. Command aliases (launch/up, stop/down, ls/list, logs/log, status/st) all work identically
**Plans**: 4 plans
Plans:
- [x] 08-01-PLAN.md — Implement init, ls, logs, status commands + Manager.List/Logs/Status + DockerClient interface extension
- [x] 08-02-PLAN.md — Exit code taxonomy (0-6), cmd/errors.go remediation hints, signal.NotifyContext on all Docker commands
- [x] 08-03-PLAN.md — Help text with examples for all 15 commands, --port/-P flag, JSON output, aliases verification, DX integration tests
- [ ] 08-04-PLAN.md — Gap closure: codify deferred init boundary (D-02), fix fallback remediation hints, normalize help example counts, and run human DX runtime verification

### Phase 9: TUI Layer
**Goal**: Interactive users get a polished BubbleTea interface; non-TTY users and CI environments get clean plain-text output automatically
**Depends on**: Phase 8
**Requirements**: TUI-01, TUI-02, TUI-03, TUI-04, TUI-05, TUI-06, TUI-07
**Success Criteria** (what must be TRUE):
  1. `zone init` in a TTY launches the BubbleTea harness-selection wizard with a live config preview
  2. Docker build output streams through the BubbleTea progress view with no garbled terminal state
  3. Running `zone init` without `--harness` in a non-TTY environment produces a helpful error message
  4. `zone launch --plain` bypasses TUI even when running in a TTY
  5. Terminal state is fully restored after any TUI session, including forced exits and panics
**Plans**: 3 plans
Plans:
- [x] 09-01-PLAN.md — Install BubbleTea v2 deps, TTY helper, Init Wizard model, wire cmd/init.go
- [ ] 09-02-PLAN.md — Build Progress model with channel adapter, Status View model, wire cmd/launch.go and cmd/status.go
- [x] 09-03-PLAN.md — Log Viewer model with follow mode and search, wire cmd/logs.go

### Phase 10: Network Sandboxing
**Goal**: Containers running on Linux are network-isolated via host-side iptables rules that survive process crashes and clean up after themselves
**Depends on**: Phase 9
**Requirements**: NET-01, NET-02, NET-03, NET-04, NET-05, NET-06, NET-07, NET-08, NET-09, NET-10, NET-11, NET-12
**Success Criteria** (what must be TRUE):
  1. Whitelist mode blocks all outbound traffic except explicitly allowed hostnames from inside the container
  2. Blocklist mode allows all outbound traffic except explicitly denied hostnames
  3. On macOS, `zone launch` with network filtering configured warns the user and falls back to mode=none
  4. iptables rules are tagged with the container ID and cleaned up when the container stops or `zone clean` runs
  5. Hostname glob patterns (e.g., `*.anthropic.com`) match correctly in both whitelist and blocklist rules
  6. Stale rules from a previous crashed Zone process are detected and removed on the next startup
**Plans**: 3 plans
Plans:
- [x] 10-01-PLAN.md — Platform detection, DockerClient.Info(), hostname glob matcher, sentinel errors, tests
- [ ] 10-02-PLAN.md — Firewall rule generation (BuildRuleSet), Firewall Apply/Remove, iptables execution, rules cache, tests
- [ ] 10-03-PLAN.md — Manager integration: setupFirewall in Launch, cleanup in Stop/Destroy, refresh goroutine, stale rule cleanup, proxy auto-allowlisting

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Project Scaffold | 2/2 | Complete   | 2026-03-27 |
| 2. Config Foundation | 3/3 | Complete   | 2026-03-27 |
| 3. Cache & State | 2/3 | In Progress|  |
| 4. Template System | 2/2 | Complete   | 2026-03-29 |
| 5. Harness Plugin System | 3/3 | Complete   | 2026-03-29 |
| 6. Docker Lifecycle Core | 4/4 | Complete   | 2026-03-29 |
| 7. Environment, Auth & Forwarding | 3/3 | Complete   | 2026-03-30 |
| 8. CLI Commands & DX | 0/3 | Not started | - |
| 9. TUI Layer | 2/3 | In Progress|  |
| 10. Network Sandboxing | 0/3 | Not started | - |
