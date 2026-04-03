---
phase: 08-cli-commands-dx
verified: 2026-03-30T23:09:01Z
status: gaps_found
score: 18/21 must-haves verified
gaps:
  - truth: "User can run `zone init` with interactive harness selection (CLI-01)"
    status: failed
    reason: "`zone init` without --harness always errors instead of presenting interactive selection."
    artifacts:
      - path: "cmd/init.go"
        issue: "RunE returns error when harness flag is empty (non-interactive only)."
    missing:
      - "Implement interactive harness selection path for TTY sessions in `zone init`"
      - "Keep current non-TTY guidance error behavior for automation contexts"
  - truth: "Every command has --help output with 2-4 usage examples (DX-09 success criterion)"
    status: failed
    reason: "Example counts are out of contract: `launch` has 5 examples; `join` and `shell` have 1 each."
    artifacts:
      - path: "cmd/launch.go"
        issue: "Example block contains 5 usage lines (>4)."
      - path: "cmd/join.go"
        issue: "Example block contains 1 usage line (<2)."
      - path: "cmd/shell.go"
        issue: "Example block contains 1 usage line (<2)."
    missing:
      - "Normalize each command's `Example` section to 2-4 usage examples"
  - truth: "Every error message includes remediation hints (DX-02)"
    status: partial
    reason: "Known sentinel errors include remediation text, but fallback/default paths return raw `Error: <err>` without actionable guidance."
    artifacts:
      - path: "cmd/errors.go"
        issue: "Default and UnknownKeys branches return plain error text with no explicit remediation hint."
    missing:
      - "Add remediation guidance for fallback error paths (generic and unknown-keys branches)"
human_verification:
  - test: "Ctrl+C while running `zone launch` with an active harness"
    expected: "SIGINT reaches harness process; CLI exits gracefully; container remains alive"
    why_human: "Requires interactive terminal signal behavior and live Docker container/harness process"
  - test: "Harness process exits naturally during `zone launch`"
    expected: "Container lifecycle and `zone launch` exit code match DX-06 contract"
    why_human: "Needs end-to-end runtime observation of process/container behavior not provable by static grep"
---

# Phase 8: CLI Commands & DX Verification Report

**Phase Goal:** All 21 CLI commands work end-to-end with correct exit codes, signal handling, JSON output, and inline help.
**Verified:** 2026-03-30T23:09:01Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | zone init --harness claude-code creates zone.toml with harness = claude-code | ✓ VERIFIED | `cmd/init.go` writes `zone.toml`; `tests/cli_commands_test.go::TestInitCreatesZoneToml` |
| 2 | zone init without --harness prints error about interactive mode | ✓ VERIFIED | `cmd/init.go:26-28`; `TestInitNoHarness` |
| 3 | zone init in directory with existing zone.toml prints error | ✓ VERIFIED | `cmd/init.go:35-38`; `TestInitExistingZoneToml` |
| 4 | zone ls outputs table with NAME, HARNESS, STATUS, UPTIME, REPO columns | ✓ VERIFIED | `cmd/ls.go:81` header + tabwriter rows |
| 5 | zone logs --build prints build log from .zone/logs/last_build.log | ✓ VERIFIED | `cmd/logs.go:36-45`; `TestLogsBuildFlag` |
| 6 | zone status outputs container info (or error when no container) | ✓ VERIFIED | `cmd/status.go` renders info; `manager.Status` returns `ErrNoContainer` |
| 7 | Config errors produce exit code 2 | ✓ VERIFIED | `cmd/errors.go` config branches => `2`; `main.go` uses mapped exit code |
| 8 | Docker errors produce exit code 3 | ✓ VERIFIED | `cmd/errors.go` Docker sentinel => `3` |
| 9 | Lock contention produces exit code 5 | ✓ VERIFIED | `cmd/errors.go` lock contention => `5` |
| 10 | No-container errors produce exit code 6 | ✓ VERIFIED | `cmd/errors.go` no-container => `6` |
| 11 | Every error message includes remediation hint on stderr | ✗ FAILED | Fallback branches in `cmd/errors.go` return plain `Error: <err>` |
| 12 | Ctrl+C during any Docker command cancels the context gracefully | ✓ VERIFIED | `signal.NotifyContext` added across Docker-calling commands; ctx passed into manager/docker calls |
| 13 | Harness process exit causes launch to return exit code 0 | ? UNCERTAIN | No deterministic automated proof in current tests; needs runtime behavior check |
| 14 | Every command has --help output with 2-4 usage examples | ✗ FAILED | `cmd/launch.go` has 5; `cmd/join.go` and `cmd/shell.go` have 1 each |
| 15 | zone ls --json produces valid JSON array | ✓ VERIFIED | `cmd/ls.go:67` `json.MarshalIndent(containers)`; spot-check test passes |
| 16 | zone status --json produces valid JSON object | ✓ VERIFIED | `cmd/status.go:58` `json.MarshalIndent(info)` |
| 17 | zone logs --json produces JSON with timestamp fields | ✓ VERIFIED | `internal/docker/manager.go` JSON `LogEntry{timestamp,stream,line}` in `Logs()` |
| 18 | Command aliases work: launch/up, stop/down, ls/list, logs/log, status/st | ✓ VERIFIED | Aliases declared in command definitions; `TestAliases` passes |
| 19 | zone launch --port 3000:3000 forwards the port | ✓ VERIFIED | `cmd/launch.go` reads `--port`; `LaunchOpts.Ports`; `internal/docker/launch.go` merges into workspace ports |
| 20 | zone validate catches config errors without Docker | ✓ VERIFIED | `cmd/validate.go` imports config only (no docker manager init) |
| 21 | zone config shows merged config with source annotations | ✓ VERIFIED | `cmd/config.go` annotated TOML/JSON rendering includes `source` metadata |

**Score:** 18/21 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `cmd/init.go` | init command behavior | ✓ VERIFIED | Exists, substantive logic, invoked via root command |
| `cmd/ls.go` | ls command with table/json | ✓ VERIFIED | Exists + wired; uses `docker.ListContainers` helper (not `mgr.List`) |
| `cmd/logs.go` | logs command with build/follow/json | ✓ VERIFIED | Exists + wired to `mgr.Logs` |
| `cmd/status.go` | status command plain/json | ✓ VERIFIED | Exists + wired to `mgr.Status` |
| `internal/docker/client_interface.go` | Docker interface includes list/logs | ✓ VERIFIED | `ContainerList`, `ContainerLogs` present |
| `internal/docker/manager.go` | List/Logs/Status implementations | ✓ VERIFIED | Methods implemented with Docker SDK calls |
| `cmd/errors.go` | centralized error mapping | ✓ VERIFIED | Exit taxonomy 0-6 implemented |
| `main.go` | mapped exit path | ✓ VERIFIED | Single `MapError` + `os.Exit(exitCode)` path |
| `cmd/root.go` | Cobra silence flags | ✓ VERIFIED | `SilenceErrors` + `SilenceUsage` set |
| `cmd/launch.go` | launch flags/help/port threading | ✓ VERIFIED | `--port/-P` + `LaunchOpts.Ports` wiring |
| `tests/cli_dx_test.go` | DX integration tests | ✓ VERIFIED | Alias/help/JSON/exit behavior tests present |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `cmd/ls.go` | `internal/docker/manager.go` | container list call | ✓ WIRED | Uses `docker.ListContainers(ctx, cli)` helper implemented in manager package |
| `cmd/logs.go` | `internal/docker/manager.go` | `mgr.Logs(ctx, ...)` | ✓ WIRED | Call and opts propagation present |
| `cmd/status.go` | `internal/docker/manager.go` | `mgr.Status(ctx)` | ✓ WIRED | Call + render path present |
| `main.go` | `cmd/errors.go` (+ docker/config sentinels) | `cmd.MapError(err)` | ✓ WIRED | Sentinel-specific `errors.Is` checks live in `cmd/errors.go` |
| `cmd/launch.go` | `os/signal` | `signal.NotifyContext` | ✓ WIRED | Context cancellation setup in RunE |
| `cmd/launch.go` | `internal/docker/launch.go` | `LaunchOpts.Ports` | ✓ WIRED | flag -> opts -> manager launch merge |
| `cmd/ls.go` | `encoding/json` | `json.MarshalIndent` | ✓ WIRED | JSON branch implemented |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| `cmd/ls.go` | `containers` | `docker.ListContainers` -> Docker `ContainerList` | Yes | ✓ FLOWING |
| `cmd/status.go` | `info` | `mgr.Status` -> Docker `ContainerInspect` | Yes | ✓ FLOWING |
| `cmd/logs.go` + `manager.Logs` | log entries | Docker `ContainerLogs` stream -> parsed JSON entries | Yes | ✓ FLOWING |
| `cmd/config.go` | merged annotated config | `config.LoadMerged` / `config.Merge` | Yes | ✓ FLOWING |
| `cmd/launch.go` | `ports` | CLI flag `--port` -> `LaunchOpts.Ports` -> `m.config.Workspace.Ports` | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Help/examples/aliases/global flags/JSON + exit-code paths | `go test ./tests -run 'TestHelpExamples|TestAliases|TestGlobalFlags|TestExitCode2OnBadConfig|TestLogsBuildFlag|TestValidateExitZero|TestConfigShowsMerged|TestLsJsonOutput|TestLaunchPortFlag' -count=1` | `ok github.com/peasant-labs/zone/tests` | ✓ PASS |
| Launch/join/shell help surfaces examples | `go run . launch --help && go run . join --help && go run . shell --help` | Help rendered; counts show 5/1/1 examples respectively | ✗ FAIL (DX-09 contract) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| CLI-01 | 08-01 | `zone init` with interactive harness selection | ✗ BLOCKED | `cmd/init.go` errors when `--harness` absent (no interactive flow) |
| CLI-02 | 08-01 | `zone init --harness <name>` non-interactive scaffold | ✓ SATISFIED | `cmd/init.go` + `TestInitCreatesZoneToml` |
| CLI-12 | 08-01 | `zone ls` lists zone containers | ✓ SATISFIED | `cmd/ls.go` table/json + Docker list helper |
| CLI-13 | 08-01 | `zone logs` with follow support | ✓ SATISFIED | `cmd/logs.go` `--follow` + `mgr.Logs` |
| CLI-14 | 08-01 | `zone logs --build` build log view | ✓ SATISFIED | file read path + `TestLogsBuildFlag` |
| CLI-17 | 08-01 | `zone status` container state/details | ✓ SATISFIED | `cmd/status.go` + `mgr.Status` inspect output |
| DX-01 | 08-02 | exit code taxonomy 0-6 | ✓ SATISFIED | `cmd/errors.go` mapping + single exit in `main.go` |
| DX-02 | 08-02 | remediation hints on all errors | ✗ BLOCKED | fallback branches lack remediation guidance |
| DX-04 | 08-02 | Ctrl+C sends SIGINT to harness; container stays alive | ? NEEDS HUMAN | signal contexts present; runtime SIGINT/container behavior needs live verification |
| DX-05 | 08-02 | context propagation for Docker calls | ✓ SATISFIED | Docker client interface uses context; command ctx propagation present |
| DX-06 | 08-02 | harness exit stops container; `zone launch` exits 0 | ? NEEDS HUMAN | no explicit deterministic automated assertion in current tests |
| DX-07 | 08-02 | `zone stop` cleanup behavior | ✓ SATISFIED | `manager.Stop`: stop/remove container, remove network, clear cache IDs |
| CLI-18 | 08-03 | `zone config` merged config with source annotations | ✓ SATISFIED | annotated TOML/JSON render paths |
| CLI-19 | 08-03 | `zone validate` validates config without launch | ✓ SATISFIED | config-only validation flow |
| CLI-20 | 08-03 | global flags on any command | ✓ SATISFIED | persistent flags on root + inherited help output |
| CLI-21 | 08-03 | harness arg forwarding via `--` | ✓ SATISFIED | `launch` passes `args` -> `LaunchOpts.HarnessArgs` -> `harnessCmd` append |
| DX-03 | 08-03 | `--json` on status/ls/config/logs | ✓ SATISFIED | flags and JSON branches implemented |
| DX-08 | 08-03 | command aliases | ✓ SATISFIED | alias declarations + `TestAliases` |
| DX-09 | 08-03 | help text with 2-4 usage examples per command | ✗ BLOCKED | launch/join/shell example counts violate contract |

**Orphaned requirements (Phase 8 mapping vs plan frontmatter):** None. All Phase 8 requirement IDs in `REQUIREMENTS.md` are represented in phase plan frontmatter.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `tests/cli_dx_test.go` | 129 | Accepts exit code `1 \| 3 \| 6` for no-container case | ⚠️ Warning | Weakens strict DX-01 no-container exit-code guarantee in regression tests |
| `tests/cli_dx_test.go` | 141 | Docker-dependent JSON test skips when Docker unavailable | ℹ️ Info | Reduces runtime assurance in non-Docker CI environments |

### Human Verification Required

### 1. Ctrl+C signal behavior during launch

**Test:** Start `zone launch` with a running harness process, then press Ctrl+C.
**Expected:** Harness receives SIGINT; CLI exits gracefully; container stays alive.
**Why human:** Requires interactive terminal signal semantics and live container observation.

### 2. Harness-exit lifecycle behavior

**Test:** Run `zone launch` with a harness that exits naturally (success path).
**Expected:** Behavior matches DX-06 contract (container/exit-code semantics).
**Why human:** Requires full runtime lifecycle verification not inferable from static analysis.

### Gaps Summary

Phase 08 has substantial progress and most DX plumbing is present/wired, but the phase goal is not fully achieved. The highest-impact blockers are: (1) `CLI-01` interactive init is missing, (2) `DX-09` example-count contract (2-4 per command) is violated, and (3) `DX-02` is incomplete on fallback error paths without remediation hints. Additionally, DX-04 and DX-06 still need live runtime verification.

---

_Verified: 2026-03-30T23:09:01Z_
_Verifier: the agent (gsd-verifier)_
