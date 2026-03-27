# Phase 3: Cache & State - Research

**Researched:** 2026-03-27
**Domain:** Go file system management, flock-based locking, SHA256 hashing, atomic file writes, gitignore management
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Stale lock recovery:**
- Non-blocking flock attempt per spec — fail immediately if locked (no retry/timeout)
- Write PID to `.zone/.lock.pid` on lock acquire for diagnostics
- Lock contention error includes PID of holding process: "Another zone process (PID 12345) is operating on this repo."
- If PID in `.lock.pid` references a dead process (check `/proc/{pid}` on Linux, `kill -0` on macOS), auto-recover the lock with warning: "Recovered stale lock from dead process (PID 12345)."
- `zone clean` warns but proceeds even if lock is held: "Warning: another zone process (PID 12345) may be running. Cleaning anyway."
- Exit code 5 for lock contention (per spec)

**Cache invalidation cleanup:**
- Old Docker images are left in Docker's local store after rebuild — just update `.zone/image_id`
- No auto-pruning of old images; user can `docker image prune` or `zone destroy`
- If cached `image_id` references a pruned/deleted image, silently rebuild (detected via `ImageInspect`); log at verbose level only: "Cached image not found, rebuilding"
- All cache file writes (config.hash, image_id, container_id, network_id) use atomic write: write to `.zone/.tmp-{name}`, then `os.Rename`
- `zone destroy` requires `--yes`/`-y` flag to skip interactive confirmation; without it, print what will be removed and ask for confirmation

**Gitignore management:**
- If no `.gitignore` exists, create a minimal one with just `.zone/`
- Find git root via `git rev-parse --show-toplevel` and modify that `.gitignore` (correct for monorepos and subdirectory invocations)
- In monorepos/subdirectories, use relative path from git root: e.g., `subdir/.zone/` (not `**/.zone/` wildcard)
- Idempotent: exact string check — skip if entry already present; don't attempt to parse gitignore glob semantics

**Build log retention:**
- Last build log only (`last_build.log`), overwritten each build — matches spec
- Keep partial log on build failure — the partial output is the most useful debugging artifact
- Tee build output to both terminal and log file simultaneously (user sees progress live)
- Prepend brief metadata header to log: `# zone build | {timestamp} | config hash: {hash} | zone {version}`

### Claude's Discretion
- Internal cache directory management (ensure-dir helpers, file permission modes)
- Hash computation serialization details (deterministic JSON encoding of MergedConfig)
- Exact flock syscall usage (syscall.Flock vs golang.org/x/sys)
- Build log tee implementation (io.MultiWriter or similar)
- Error message wording for edge cases not covered above

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CAC-01 | `.zone/` directory stores config hash, Dockerfile, entrypoint, image/container/network IDs | `os.MkdirAll` + `os.WriteFile`/`os.ReadFile` patterns; `cache.go` directory management |
| CAC-02 | Cache hash includes merged config + templates + Zone version for automatic invalidation | `crypto/sha256` + `encoding/json` marshaling of `MergedConfig`; `fs.ReadFile(templates.FS, ...)` for template content; version passed as parameter |
| CAC-03 | File-based locking via flock for concurrent access protection | `syscall.Flock` with `LOCK_EX|LOCK_NB`; cross-platform (Linux + macOS); PID file tracking |
| CAC-04 | Lock contention produces error with exit code 5 | `ErrLockContention` sentinel; cmd layer maps to `os.Exit(5)` |
| CAC-05 | `zone init` and `zone launch` add `.zone/` to `.gitignore` | `git rev-parse --show-toplevel` via `os/exec`; exact string check; relative path from git root |
| CAC-06 | Build logs stored in `.zone/logs/last_build.log` | `io.MultiWriter` for tee; `os.MkdirAll` for logs subdir; metadata header prepend |
</phase_requirements>

---

## Summary

Phase 3 builds the `internal/cache/` package from three stub files into a fully functional cache layer. The domain is pure Go standard library: no new dependencies are needed. All required primitives — `syscall.Flock`, `crypto/sha256`, `encoding/json`, `os.Rename` (atomic write), `io.MultiWriter`, `os/exec` — are available in the current module without adding entries to `go.mod`.

The key design insight from the CONTEXT.md decisions is that the `Cache` struct becomes the single access point for the `.zone/` directory. All reads/writes go through it, and the lock is always held during the build/create/start sequence (but released before TTY attachment). The three files map cleanly to three tasks: `cache.go` (directory management + ID persistence + log storage), `hash.go` (SHA256 computation), and `lock.go` (flock + PID tracking).

One important integration detail: the spec shows `templates.DockerfileTmpl` as a string variable, but the actual `pkg/templates/templates.go` exports only `templates.FS` (an `embed.FS`). The hash function must use `fs.ReadFile(templates.FS, "Dockerfile.tmpl")` and `fs.ReadFile(templates.FS, "entrypoint.sh.tmpl")` instead of string vars. Similarly, the Zone version is a local var in `main.go` — it must be passed as a parameter to `ComputeHash()`, not imported from the cmd package (which would violate the import graph constraint).

**Primary recommendation:** Implement `internal/cache/` as three focused files with no new dependencies; pass `version string` explicitly to `ComputeHash`; use `syscall.Flock` directly (available on both Linux and macOS in Go stdlib); write all tests in `tests/hash_test.go` as integration tests using the compiled binary following the established pattern.

---

## Standard Stack

### Core (stdlib only — no new dependencies)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `crypto/sha256` | stdlib | Cache hash computation | Spec-specified; fast, deterministic |
| `encoding/json` | stdlib | Deterministic MergedConfig serialization for hash | Struct field order in JSON is deterministic for named structs |
| `syscall` | stdlib | `syscall.Flock`, `LOCK_EX`, `LOCK_NB`, `LOCK_UN`, `Kill` | Flock available on both Linux and macOS in Go stdlib |
| `os` | stdlib | File create/read/write/rename/stat/mkdir | All file operations |
| `io` | stdlib | `io.MultiWriter` for build log tee | Spec pattern for simultaneous write to terminal + file |
| `os/exec` | stdlib | `git rev-parse --show-toplevel` for gitignore management | No git library needed |
| `bufio` | stdlib | Line scanning when checking gitignore content | Idiomatic line-by-line file reading |
| `fmt` | stdlib | PID formatting, error messages | Standard |
| `time` | stdlib | Timestamp in build log header | Standard |
| `runtime` | stdlib | `runtime.GOOS` for cross-platform dead process detection | Distinguishes `/proc` vs `syscall.Kill` approach |
| `io/fs` | stdlib | `fs.ReadFile(templates.FS, name)` for reading embedded templates | embed.FS implements ReadFileFS |
| `strings` | stdlib | `strings.Contains` for gitignore exact-match check | Idiomatic |

**Verified:** `syscall.LOCK_EX = 2`, `syscall.LOCK_NB = 4`, `syscall.LOCK_UN = 8` — confirmed available on linux/arm64 (Go 1.25.5). `syscall.Flock` and `syscall.Kill` are both in the stdlib `syscall` package and available on Linux and macOS. No `golang.org/x/sys` dependency needed.

**Version verification:** `go.mod` is `go 1.25.5` with only `github.com/BurntSushi/toml`, `github.com/agnivade/levenshtein`, `github.com/spf13/cobra`, and `github.com/stretchr/testify` as direct deps. Phase 3 adds zero new entries to `go.mod`.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `syscall.Flock` | `golang.org/x/sys/unix.Flock` | `x/sys` is more portable but adds a dependency; `syscall.Flock` covers Linux + macOS which are the only v1 targets |
| `os.Rename` (atomic) | Write directly to target file | Direct write leaves a partial file on crash; `os.Rename` is atomic on same filesystem (Linux/macOS) |
| `encoding/json` for hash | `fmt.Sprintf` or TOML re-serialization | JSON struct marshaling is deterministic for named struct fields (field order fixed by reflection); simpler than custom serializer |

---

## Architecture Patterns

### Recommended Project Structure

The spec mandates this structure — no discretion here:

```
internal/cache/
├── cache.go    # Cache struct, directory init, ID file CRUD, log storage
├── hash.go     # ComputeHash(cfg, version) string
└── lock.go     # Lock struct with Acquire/Release/IsHeld
```

```
tests/
└── hash_test.go    # Unit tests for hash computation (stub exists, needs content)
```

### Pattern 1: Cache Struct with Directory Root

**What:** `Cache` holds the `.zone/` path and the open lock file handle. All operations are methods on `Cache`.

**When to use:** Always — single struct owns all state for `.zone/`.

```go
// Source: zone-spec.md section 7 + CONTEXT.md code_context

type Cache struct {
    dir      string    // absolute path to .zone/ directory
    lockFile *os.File  // held open for duration of lock
}

// New returns a Cache for the given repo directory.
// Does NOT acquire the lock or create the directory.
func New(repoDir string) *Cache {
    return &Cache{dir: filepath.Join(repoDir, ".zone")}
}

// EnsureDir creates the .zone/ and .zone/logs/ directories if they don't exist.
func (c *Cache) EnsureDir() error {
    if err := os.MkdirAll(c.dir, 0755); err != nil {
        return fmt.Errorf("create cache dir: %w", err)
    }
    if err := os.MkdirAll(filepath.Join(c.dir, "logs"), 0755); err != nil {
        return fmt.Errorf("create logs dir: %w", err)
    }
    return nil
}
```

### Pattern 2: Atomic File Write

**What:** Write to a `.tmp-{name}` file, then `os.Rename` to the target. Prevents partial reads.

**When to use:** Every cache file write — `config.hash`, `image_id`, `container_id`, `network_id`.

```go
// Source: CONTEXT.md locked decisions (atomic write requirement)

func (c *Cache) writeAtomic(name, content string) error {
    tmpPath := filepath.Join(c.dir, ".tmp-"+name)
    if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
        return fmt.Errorf("write tmp file: %w", err)
    }
    target := filepath.Join(c.dir, name)
    if err := os.Rename(tmpPath, target); err != nil {
        _ = os.Remove(tmpPath) // best-effort cleanup
        return fmt.Errorf("rename to %s: %w", name, err)
    }
    return nil
}

// Public helpers using writeAtomic:
func (c *Cache) SetImageID(id string) error    { return c.writeAtomic("image_id", id) }
func (c *Cache) SetContainerID(id string) error { return c.writeAtomic("container_id", id) }
func (c *Cache) SetNetworkID(id string) error  { return c.writeAtomic("network_id", id) }
func (c *Cache) SetConfigHash(h string) error  { return c.writeAtomic("config.hash", h) }

// Corresponding readers:
func (c *Cache) ImageID() (string, error)    { return c.readTrimmed("image_id") }
func (c *Cache) ContainerID() (string, error) { return c.readTrimmed("container_id") }
func (c *Cache) NetworkID() (string, error)  { return c.readTrimmed("network_id") }
func (c *Cache) ConfigHash() (string, error) { return c.readTrimmed("config.hash") }

func (c *Cache) readTrimmed(name string) (string, error) {
    data, err := os.ReadFile(filepath.Join(c.dir, name))
    if os.IsNotExist(err) {
        return "", nil // not set yet is not an error
    }
    if err != nil {
        return "", fmt.Errorf("read %s: %w", name, err)
    }
    return strings.TrimSpace(string(data)), nil
}
```

### Pattern 3: flock-Based Locking with PID File

**What:** Open/create `.zone/.lock`, call `syscall.Flock(LOCK_EX|LOCK_NB)`. Write PID to `.zone/.lock.pid`. On contention, read existing PID file to check for dead process before returning error.

**When to use:** Acquire before any `.zone/` read/write. Release before TTY attachment.

```go
// Source: zone-spec.md section 6.1 + CONTEXT.md locked decisions

type Lock struct {
    dir  string
    file *os.File
}

func NewLock(cacheDir string) *Lock {
    return &Lock{dir: cacheDir}
}

// Acquire attempts a non-blocking exclusive flock.
// Returns ErrLockContention (with PID) if already held by a live process.
// Auto-recovers stale locks from dead processes.
func (l *Lock) Acquire() error {
    lockPath := filepath.Join(l.dir, ".lock")
    f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
    if err != nil {
        return fmt.Errorf("open lock file: %w", err)
    }

    err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
    if err == nil {
        // Lock acquired — write our PID
        l.file = f
        pidPath := filepath.Join(l.dir, ".lock.pid")
        _ = os.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0644)
        return nil
    }

    f.Close()

    if err != syscall.EWOULDBLOCK {
        return fmt.Errorf("flock: %w", err)
    }

    // Lock is held — check if the holder is still alive
    pid := readPIDFile(filepath.Join(l.dir, ".lock.pid"))
    if pid > 0 && isDeadProcess(pid) {
        // Stale lock: clean up and retry once
        _ = os.Remove(filepath.Join(l.dir, ".lock"))
        _ = os.Remove(filepath.Join(l.dir, ".lock.pid"))
        fmt.Fprintf(os.Stderr, "Warning: Recovered stale lock from dead process (PID %d).\n", pid)
        return l.Acquire()
    }

    if pid > 0 {
        return fmt.Errorf("%w (PID %d)", ErrLockContention, pid)
    }
    return ErrLockContention
}

func (l *Lock) Release() {
    if l.file == nil {
        return
    }
    _ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
    l.file.Close()
    l.file = nil
    _ = os.Remove(filepath.Join(l.dir, ".lock.pid"))
}
```

### Pattern 4: Dead Process Detection (cross-platform)

**What:** Check if a PID refers to a running process. On Linux, check `/proc/{pid}`. On macOS, use `syscall.Kill(pid, 0)`.

**When to use:** When reading `.lock.pid` to decide whether to auto-recover a stale lock.

```go
// Source: CONTEXT.md locked decisions

func isDeadProcess(pid int) bool {
    if runtime.GOOS == "linux" {
        _, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
        return os.IsNotExist(err)
    }
    // macOS + other Unix: kill -0 checks existence without signaling
    err := syscall.Kill(pid, 0)
    return err == syscall.ESRCH // ESRCH = "no such process"
}
```

### Pattern 5: Cache Hash Computation

**What:** SHA256 over deterministic JSON of `MergedConfig` + embedded template contents + version string.

**Critical detail:** The spec shows `templates.DockerfileTmpl` and `templates.EntrypointTmpl` as named string vars, but the actual `pkg/templates/templates.go` only exports `templates.FS` (an `embed.FS`). Use `fs.ReadFile(templates.FS, "Dockerfile.tmpl")` instead.

**Critical detail 2:** The version is `var version = "dev"` in `main.go` — it cannot be imported by `internal/cache` (import graph: `internal/* -> cmd/*` is FORBIDDEN). Pass version as a function parameter.

```go
// Source: zone-spec.md section 6.2 + pkg/templates/templates.go inspection

import (
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "io/fs"

    "github.com/peasant-labs/zone/internal/config"
    "github.com/peasant-labs/zone/pkg/templates"
)

// ComputeHash returns the hex SHA256 of merged config JSON + Dockerfile template +
// entrypoint template + Zone binary version. Version is passed in (not imported)
// because main.go -> cmd -> internal would violate the import graph.
func ComputeHash(cfg *config.MergedConfig, version string) (string, error) {
    cfgJSON, err := json.Marshal(cfg)
    if err != nil {
        return "", fmt.Errorf("marshal config: %w", err)
    }

    dockerfileTmpl, err := fs.ReadFile(templates.FS, "Dockerfile.tmpl")
    if err != nil {
        return "", fmt.Errorf("read Dockerfile template: %w", err)
    }

    entrypointTmpl, err := fs.ReadFile(templates.FS, "entrypoint.sh.tmpl")
    if err != nil {
        return "", fmt.Errorf("read entrypoint template: %w", err)
    }

    h := sha256.New()
    h.Write(cfgJSON)
    h.Write(dockerfileTmpl)
    h.Write(entrypointTmpl)
    h.Write([]byte(version))

    return fmt.Sprintf("%x", h.Sum(nil)), nil
}
```

**JSON determinism note:** `encoding/json.Marshal` for named Go structs produces fields in the order they appear in the struct definition (fixed by reflection). The `MergedConfig` struct fields are stable, so the hash is deterministic across invocations for the same config. Maps within structs (if any) are NOT deterministic — inspect `MergedConfig` before adding any `map` fields.

**MergedConfig map fields:** `AnnotatedConfig` has `CustomAliases map[string]AnnotatedField[string]` but `MergedConfig` itself has no map fields — it uses slices throughout. Hashing `MergedConfig` (not `AnnotatedConfig`) is safe.

### Pattern 6: Build Log with Tee and Metadata Header

**What:** `io.MultiWriter` writes simultaneously to terminal stdout and the log file. Prepend a metadata header line.

```go
// Source: CONTEXT.md locked decisions + io stdlib

import (
    "fmt"
    "io"
    "os"
    "time"
)

// CreateBuildLog opens .zone/logs/last_build.log for writing (truncate if exists).
// Returns a writer that tees to both w (terminal) and the log file, plus a closer func.
func (c *Cache) CreateBuildLog(w io.Writer, configHash, version string) (io.Writer, func(), error) {
    logPath := filepath.Join(c.dir, "logs", "last_build.log")
    f, err := os.Create(logPath) // O_TRUNC
    if err != nil {
        return nil, nil, fmt.Errorf("create build log: %w", err)
    }

    header := fmt.Sprintf("# zone build | %s | config hash: %s | zone %s\n",
        time.Now().Format(time.RFC3339), configHash, version)
    _, _ = f.WriteString(header)

    tee := io.MultiWriter(w, f)
    closer := func() { _ = f.Close() }
    return tee, closer, nil
}
```

**Note:** Log file is NOT closed on build failure — keep partial log. The `closer` function is deferred but should always be called (even on error path) to flush buffered writes. On failure, the file remains with whatever content was written.

### Pattern 7: Gitignore Management

**What:** Run `git rev-parse --show-toplevel` to find git root. Compute relative path from git root to cwd. Check if `{rel}/.zone/` is already in `.gitignore`. If not, append it. If no `.gitignore` exists, create one.

```go
// Source: CONTEXT.md locked decisions

import (
    "bufio"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
)

// EnsureGitignore adds .zone/ to the git root's .gitignore.
// cwd is the directory zone is invoked from (where zone.toml lives).
// Idempotent: no-op if entry already present.
func EnsureGitignore(cwd string) error {
    // Find git root
    out, err := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel").Output()
    if err != nil {
        // Not in a git repo — silently skip
        return nil
    }
    gitRoot := strings.TrimSpace(string(out))

    // Compute relative path from git root to the .zone/ dir
    rel, err := filepath.Rel(gitRoot, cwd)
    if err != nil {
        return fmt.Errorf("relative path: %w", err)
    }
    var entry string
    if rel == "." {
        entry = ".zone/"
    } else {
        entry = rel + "/.zone/"
    }

    gitignorePath := filepath.Join(gitRoot, ".gitignore")

    // Check if entry already present (exact string match)
    if data, err := os.ReadFile(gitignorePath); err == nil {
        scanner := bufio.NewScanner(strings.NewReader(string(data)))
        for scanner.Scan() {
            if strings.TrimSpace(scanner.Text()) == entry {
                return nil // already present
            }
        }
    }

    // Append entry (or create minimal .gitignore)
    f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return fmt.Errorf("open .gitignore: %w", err)
    }
    defer f.Close()
    _, err = fmt.Fprintf(f, "\n%s\n", entry)
    return err
}
```

### Pattern 8: Sentinel Errors (lock.go)

```go
// Source: zone-spec.md section 8 (error handling) + CONTEXT.md

import "errors"

var ErrLockContention = errors.New("another zone process is operating on this repo")
```

**Note:** The spec defines `ErrLockContention` in `internal/docker/errors.go`. For Phase 3, define it in `internal/cache/lock.go` since the cache package owns locking. The docker package can import from cache (import graph: `internal/docker -> internal/cache` is OK).

### Anti-Patterns to Avoid

- **Direct file write to target:** Always use atomic write (`writeAtomic`) for cache files. Direct writes leave partial files if the process is killed.
- **Importing cmd from internal:** Version string must be passed as a parameter. `internal/cache` must not import `cmd` or `main`.
- **Using `templates.DockerfileTmpl`:** That string var does not exist in the codebase. Use `fs.ReadFile(templates.FS, "Dockerfile.tmpl")`.
- **Blocking flock:** Always use `LOCK_EX|LOCK_NB`. Never use `LOCK_EX` alone (would block indefinitely).
- **Releasing lock after TTY attach:** Lock must be released BEFORE TTY attachment so `zone join` can work concurrently.
- **Closing log file on build error:** Keep file open/partial — the partial log is the debugging artifact.
- **Glob-based gitignore entry:** Use exact relative path (`subdir/.zone/`), not `**/.zone/`.
- **Checking gitignore with glob semantics:** Use simple `strings.TrimSpace(line) == entry` comparison — don't parse gitignore patterns.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Exclusive file locking | Custom lock using lock files only | `syscall.Flock` | Flock is released automatically on process death; pure lockfile requires manual stale detection |
| SHA256 computation | Any custom hash function | `crypto/sha256` stdlib | Spec-specified; already in stdlib |
| JSON serialization for hash | Custom serializer | `encoding/json.Marshal` on named struct | Struct field order is deterministic; no custom code needed |
| Simultaneous stdout + file logging | Forked write loops | `io.MultiWriter` | stdlib one-liner; handles all error propagation |
| Atomic file writes | Writing directly to target | `os.WriteFile` to tmp + `os.Rename` | `os.Rename` is atomic on same filesystem; prevents partial reads |
| Git root discovery | Walking parent dirs | `git rev-parse --show-toplevel` via `os/exec` | Handles submodules, worktrees, edge cases that manual walking misses |

**Key insight:** Every problem in this phase has a stdlib solution. Adding `golang.org/x/sys` for flock would be unnecessary complexity — `syscall.Flock` covers Linux and macOS perfectly.

---

## Common Pitfalls

### Pitfall 1: Non-Atomic Rename Across Filesystems
**What goes wrong:** `os.Rename` fails with "invalid cross-device link" if `.zone/` is on a different filesystem than `/tmp`.
**Why it happens:** The CONTEXT.md specifies writing to `.zone/.tmp-{name}` (same directory as target), not to `/tmp`. This keeps source and destination on the same filesystem.
**How to avoid:** Always write temp files to the same directory as the target: `filepath.Join(c.dir, ".tmp-"+name)`.
**Warning signs:** Test on NFS-mounted home directories; `os.Rename` errors referencing "cross-device link".

### Pitfall 2: syscall.EWOULDBLOCK vs syscall.EAGAIN
**What goes wrong:** On some Linux kernels, `flock` returns `EAGAIN` instead of `EWOULDBLOCK` for lock contention when `LOCK_NB` is used.
**Why it happens:** POSIX says either errno may be returned; Linux returns `EWOULDBLOCK` but older kernels may vary.
**How to avoid:** Check for `err == syscall.EWOULDBLOCK || err == syscall.EAGAIN`.
**Warning signs:** Lock contention not detected on certain Linux distros.

### Pitfall 3: Lock File Held After Process Death Without Cleanup
**What goes wrong:** If Zone crashes after acquiring the flock but before writing the PID file, the PID file is empty or missing. Dead process detection reads an empty PID, returns `pid == 0`, falls through to generic `ErrLockContention`.
**Why it happens:** PID write is not atomic with flock acquire.
**How to avoid:** In `isDeadProcess`, treat `pid <= 0` (parse failure or zero) as "assume dead, auto-recover" — not as "assume live process". The flock itself is the true safety mechanism; the PID file is only for error messages.
**Warning signs:** Users see persistent lock contention after a crash.

### Pitfall 4: JSON Marshaling of MergedConfig with nil Pointer Fields
**What goes wrong:** `MergedConfig.Auth.MountHomeConfig` is `*bool`. JSON marshaling of `nil *bool` produces `null`, which changes the hash when the field is not set vs explicitly set to false.
**Why it happens:** `json.Marshal` represents `nil *bool` as `null` and `*bool(false)` as `false`.
**How to avoid:** The hash is computed on the effective merged config (after merging with defaults). The merge layer in Phase 2 ensures all `*bool` fields are populated with defaults before `MergedConfig` is constructed. Verify by checking `DefaultGlobalConfig()` sets `MountHomeConfig` and `ForwardSSHAgent`.
**Warning signs:** Config hash changes between invocations even when config is unchanged.

### Pitfall 5: EnsureGitignore in Non-Git Repos
**What goes wrong:** `git rev-parse --show-toplevel` exits non-zero when not in a git repo.
**Why it happens:** Zone can be used outside git repos (unlikely but possible).
**How to avoid:** If `exec.Command(...).Output()` returns a non-nil error, silently return `nil` — not in a git repo, skip gitignore management.
**Warning signs:** `zone launch` fails in non-git directories with confusing git error.

### Pitfall 6: Log File Not Flushed on Build Failure
**What goes wrong:** The `closer` function (wrapping `f.Close()`) is not called on the error path, leaving buffered content unwritten.
**Why it happens:** Defer vs manual close on error paths.
**How to avoid:** The caller should always `defer closer()` — the spec requires keeping partial log on failure. The `CreateBuildLog` pattern returns a `closer` func that callers must defer unconditionally.
**Warning signs:** Build log is empty or truncated on failed builds.

### Pitfall 7: Import Graph Violation for Version String
**What goes wrong:** `internal/cache/hash.go` imports `cmd` or `main` to get the version string.
**Why it happens:** Spec shows hash uses Zone version; version is in `main.go`.
**How to avoid:** Pass version as a parameter: `ComputeHash(cfg *config.MergedConfig, version string) (string, error)`. The cmd layer calls it with the version it already has from ldflags.
**Warning signs:** Build error "import cycle not allowed".

---

## Code Examples

### Complete lock.go structure

```go
// Source: zone-spec.md section 6.1 + CONTEXT.md

package cache

import (
    "errors"
    "fmt"
    "os"
    "runtime"
    "strconv"
    "strings"
    "syscall"
)

var ErrLockContention = errors.New("another zone process is operating on this repo")

type Lock struct {
    dir  string
    file *os.File
}

func NewLock(cacheDir string) *Lock { return &Lock{dir: cacheDir} }

func (l *Lock) Acquire() error { /* see Pattern 3 */ }
func (l *Lock) Release()       { /* see Pattern 3 */ }
func (l *Lock) IsHeld() bool   { return l.file != nil }

// helpers (unexported):
func readPIDFile(path string) int { ... }
func isDeadProcess(pid int) bool  { ... }  // see Pattern 4
```

### Wire in cmd/clean.go (Phase 3 scope for zone clean)

```go
// Source: CONTEXT.md decisions — zone clean warns but proceeds even if lock held

var cleanCmd = &cobra.Command{
    Use:   "clean",
    Short: "Remove the .zone/ cache directory",
    RunE: func(cmd *cobra.Command, args []string) error {
        cwd, _ := os.Getwd()
        c := cache.New(cwd)
        lock := cache.NewLock(c.Dir())

        // Check if lock is held; warn but proceed
        if pid := cache.ReadLockPID(c.Dir()); pid > 0 {
            fmt.Fprintf(os.Stderr,
                "Warning: another zone process (PID %d) may be running. Cleaning anyway.\n", pid)
        }

        return os.RemoveAll(c.Dir())
    },
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Shell script lockfiles (touch + rm) | `syscall.Flock` exclusive lock | Go stdlib since day 1 | Flock auto-released on process death; no stale cleanup needed in most cases |
| Writing to target file directly | Write to `.tmp-{name}` + `os.Rename` | Go 1.16+ (os.Rename always existed) | Atomic writes prevent partial reads on crash |
| `ioutil.WriteFile` / `ioutil.ReadFile` | `os.WriteFile` / `os.ReadFile` | Go 1.16 (ioutil deprecated) | Current project uses Go 1.25.5; always use `os.*` not `ioutil.*` |

**Deprecated/outdated:**
- `ioutil.WriteFile`, `ioutil.ReadFile`, `ioutil.TempFile`: Deprecated since Go 1.16. Use `os.WriteFile`, `os.ReadFile`, `os.CreateTemp`. The project's go.mod specifies `go 1.25.5` — never use `ioutil`.

---

## Open Questions

1. **Where to define ErrLockContention**
   - What we know: The spec defines it in `internal/docker/errors.go`; but `internal/docker -> internal/cache` is the import direction (docker imports cache, not reverse).
   - What's unclear: Should `ErrLockContention` live in `internal/cache` (owned by who defines it) or `internal/docker/errors.go` (per spec)?
   - Recommendation: Define in `internal/cache/lock.go`. The docker package imports it from there. The cmd layer maps it to exit code 5. This respects the import graph and keeps the sentinel with the package that generates it.

2. **`zone clean` scope in Phase 3**
   - What we know: `cmd/clean.go` stub exists; CONTEXT.md says `clean` warns but proceeds. But Phase 6 is the full lifecycle phase.
   - What's unclear: Does Phase 3 wire `zone clean` as a simple `os.RemoveAll(.zone/)` or leave it as stub?
   - Recommendation: Wire `zone clean` as a basic `os.RemoveAll(c.Dir())` with PID warning in Phase 3 — it only touches `.zone/` which is this phase's domain. Full `--all` flag (removes Docker image) deferred to Phase 6.

3. **Cache struct public API for downstream phases**
   - What we know: Phase 4 (templates) and Phase 6 (lifecycle) are the primary consumers.
   - What's unclear: Exact method signatures expected by Phase 6.
   - Recommendation: Export `GetImageID`, `SetImageID`, `GetContainerID`, `SetContainerID`, `GetNetworkID`, `SetNetworkID`, `GetConfigHash`, `SetConfigHash`, `EnsureDir`, `Dir() string`. Phase 6 will use all of these.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | `testing` stdlib + `testify` v1.10.0 |
| Config file | none — uses `go test ./...` |
| Quick run command | `go test ./tests/ -run TestHash -v` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CAC-01 | `.zone/` dir created; files read/write correctly | unit | `go test ./tests/ -run TestCache -v` | Wave 0 |
| CAC-02 | Hash changes when config changes; hash changes when version changes; hash stable for same inputs | unit | `go test ./tests/ -run TestHash -v` | ❌ Wave 0 (stub only) |
| CAC-03 | Lock acquired; second acquire returns error; lock released | unit | `go test ./tests/ -run TestLock -v` | Wave 0 |
| CAC-04 | Lock contention exits code 5 | integration | `go test ./tests/ -run TestLockContention -v` | Wave 0 |
| CAC-05 | `.gitignore` updated; idempotent; monorepo relative path | unit | `go test ./tests/ -run TestGitignore -v` | Wave 0 |
| CAC-06 | Build log created; metadata header present; tee works | unit | `go test ./tests/ -run TestBuildLog -v` | Wave 0 |

**Note:** The project's integration test pattern (from Phase 2) uses pre-built binary via `sync.Once` for CLI-level tests (`tests/config_cmd_test.go`). The cache unit tests (`tests/hash_test.go`) can use the cache package directly without the binary. Both patterns are appropriate.

### Sampling Rate
- **Per task commit:** `go test ./tests/ -run TestHash -v`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `tests/hash_test.go` — stub exists but has no test functions; needs `TestHashStability`, `TestHashChangesOnConfigChange`, `TestHashChangesOnVersion`
- [ ] `tests/cache_test.go` — new file; covers CAC-01 (dir management), CAC-03 (lock acquire/release), CAC-05 (gitignore), CAC-06 (build log)
- [ ] `tests/lock_contention_test.go` OR add to `cache_test.go` — integration test that spawns two processes to test exit code 5 (CAC-04)

---

## Sources

### Primary (HIGH confidence)
- zone-spec.md sections 6, 6.1, 6.2, 6.3, 3.7, 3.9, 8 — authoritative spec for all cache behavior
- `.planning/phases/03-cache-state/03-CONTEXT.md` — locked implementation decisions
- Go stdlib documentation (`go doc syscall`, `go doc os`, `go doc io`, `go doc crypto/sha256`) — verified at Go 1.25.5
- `/workspace/zone/internal/config/types.go` — `MergedConfig` struct (no map fields; safe for deterministic JSON hashing)
- `/workspace/zone/pkg/templates/templates.go` — confirms `templates.FS` (not named string vars)
- `/workspace/zone/go.mod` — confirms no `golang.org/x/sys`; no new deps needed

### Secondary (MEDIUM confidence)
- `tests/config_cmd_test.go` — established integration test pattern (binary build via `sync.Once`, `runZone` helper)

### Tertiary (LOW confidence)
- None

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — verified against Go 1.25.5 stdlib; no new dependencies; flock constants confirmed
- Architecture: HIGH — spec section 6 is prescriptive; CONTEXT.md decisions lock all key choices
- Pitfalls: HIGH — `EWOULDBLOCK/EAGAIN` and cross-device rename are well-known Go/Linux pitfalls; nil pointer JSON hash issue is deterministic from code inspection

**Research date:** 2026-03-27
**Valid until:** 2026-06-27 (stable domain — Go stdlib, flock semantics)
