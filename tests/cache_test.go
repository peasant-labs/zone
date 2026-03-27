// Tests for cache directory management and atomic ID persistence.
package tests

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCacheEnsureDir verifies that New + EnsureDir creates .zone/ and .zone/logs/ directories.
func TestCacheEnsureDir(t *testing.T) {
	base := t.TempDir()
	c := cache.New(base)
	err := c.EnsureDir()
	require.NoError(t, err, "EnsureDir should not return an error")

	// .zone/ must exist
	info, err := os.Stat(c.Dir())
	require.NoError(t, err, ".zone/ directory must exist")
	assert.True(t, info.IsDir(), ".zone/ must be a directory")

	// .zone/logs/ must exist
	logsInfo, err := os.Stat(filepath.Join(c.Dir(), "logs"))
	require.NoError(t, err, ".zone/logs/ directory must exist")
	assert.True(t, logsInfo.IsDir(), ".zone/logs/ must be a directory")
}

// TestCacheAtomicWrite verifies that SetImageID writes atomically and no .tmp- file remains.
func TestCacheAtomicWrite(t *testing.T) {
	base := t.TempDir()
	c := cache.New(base)
	require.NoError(t, c.EnsureDir())

	err := c.SetImageID("sha256:abc123")
	require.NoError(t, err, "SetImageID should not error")

	id, err := c.ImageID()
	require.NoError(t, err, "ImageID should not error")
	assert.Equal(t, "sha256:abc123", id, "ImageID should return the written value")

	// No leftover .tmp- file should exist
	_, statErr := os.Stat(filepath.Join(c.Dir(), ".tmp-image_id"))
	assert.True(t, os.IsNotExist(statErr), ".tmp-image_id must not exist after write")
}

// TestCacheReadWrite verifies round-trip correctness for all four ID types.
func TestCacheReadWrite(t *testing.T) {
	base := t.TempDir()
	c := cache.New(base)
	require.NoError(t, c.EnsureDir())

	// ConfigHash round-trip
	require.NoError(t, c.SetConfigHash("deadbeef"))
	h, err := c.ConfigHash()
	require.NoError(t, err)
	assert.Equal(t, "deadbeef", h, "ConfigHash should return the written value")

	// ContainerID round-trip
	require.NoError(t, c.SetContainerID("cid123"))
	cid, err := c.ContainerID()
	require.NoError(t, err)
	assert.Equal(t, "cid123", cid, "ContainerID should return the written value")

	// NetworkID round-trip
	require.NoError(t, c.SetNetworkID("nid456"))
	nid, err := c.NetworkID()
	require.NoError(t, err)
	assert.Equal(t, "nid456", nid, "NetworkID should return the written value")
}

// TestCacheReadMissing verifies that reading a missing key returns ("", nil) — not an error.
func TestCacheReadMissing(t *testing.T) {
	base := t.TempDir()
	c := cache.New(base)
	require.NoError(t, c.EnsureDir())

	id, err := c.ImageID()
	require.NoError(t, err, "ImageID on fresh cache must not error")
	assert.Equal(t, "", id, "ImageID on fresh cache must return empty string")
}

// ---------------------------------------------------------------------------
// Lock tests
// ---------------------------------------------------------------------------

// TestLockAcquireRelease verifies basic acquire/release semantics and PID file creation.
func TestLockAcquireRelease(t *testing.T) {
	dir := t.TempDir()
	zoneDir := filepath.Join(dir, ".zone")
	require.NoError(t, os.MkdirAll(zoneDir, 0755))

	l := cache.NewLock(zoneDir)
	require.NoError(t, l.Acquire())
	assert.True(t, l.IsHeld(), "lock should be held after Acquire")

	// PID file should contain current process PID
	pidData, err := os.ReadFile(filepath.Join(zoneDir, ".lock.pid"))
	require.NoError(t, err, ".lock.pid must exist after Acquire")
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	require.NoError(t, err, ".lock.pid must contain a valid integer")
	assert.Equal(t, os.Getpid(), pid, ".lock.pid should contain current process PID")

	l.Release()
	assert.False(t, l.IsHeld(), "lock should not be held after Release")
}

// TestLockDouble verifies that a second Acquire on the same dir returns ErrLockContention.
func TestLockDouble(t *testing.T) {
	dir := t.TempDir()
	zoneDir := filepath.Join(dir, ".zone")
	require.NoError(t, os.MkdirAll(zoneDir, 0755))

	l1 := cache.NewLock(zoneDir)
	require.NoError(t, l1.Acquire())
	defer l1.Release()

	l2 := cache.NewLock(zoneDir)
	err := l2.Acquire()
	require.Error(t, err, "second Acquire should return an error")
	assert.True(t, errors.Is(err, cache.ErrLockContention), "error should wrap ErrLockContention")
}

// ---------------------------------------------------------------------------
// Gitignore tests
// ---------------------------------------------------------------------------

// TestGitignoreCreation verifies that EnsureGitignore adds .zone/ to the git root .gitignore.
func TestGitignoreCreation(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, exec.Command("git", "init", dir).Run())

	require.NoError(t, cache.EnsureGitignore(dir))

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err, ".gitignore must exist after EnsureGitignore")
	assert.Contains(t, string(data), ".zone/", ".gitignore should contain .zone/ entry")
}

// TestGitignoreIdempotent verifies that calling EnsureGitignore twice adds the entry exactly once.
func TestGitignoreIdempotent(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, exec.Command("git", "init", dir).Run())

	require.NoError(t, cache.EnsureGitignore(dir))
	require.NoError(t, cache.EnsureGitignore(dir))

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err, ".gitignore must exist")

	content := string(data)
	lines := strings.Split(content, "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == ".zone/" {
			count++
		}
	}
	assert.Equal(t, 1, count, ".zone/ should appear exactly once in .gitignore")
}

// ---------------------------------------------------------------------------
// Build log tests
// ---------------------------------------------------------------------------

// TestBuildLogCreation verifies that CreateBuildLog writes header + body to the log file
// and tees output to the provided writer.
func TestBuildLogCreation(t *testing.T) {
	c := cache.New(t.TempDir())
	require.NoError(t, c.EnsureDir())

	var buf bytes.Buffer
	tee, closer, err := c.CreateBuildLog(&buf, "abc123", "1.0.0")
	require.NoError(t, err)

	_, _ = fmt.Fprint(tee, "build output here")
	closer()

	logData, err := os.ReadFile(filepath.Join(c.Dir(), "logs", "last_build.log"))
	require.NoError(t, err, "last_build.log must exist after CreateBuildLog + closer")

	content := string(logData)
	assert.Contains(t, content, "# zone build |", "log should contain header prefix")
	assert.Contains(t, content, "config hash: abc123", "log should contain config hash")
	assert.Contains(t, content, "zone 1.0.0", "log should contain version")
	assert.Contains(t, content, "build output here", "log should contain written body")

	// tee must also forward output to the original writer
	assert.Contains(t, buf.String(), "build output here", "tee must forward output to original writer")
}

// TestBuildLogHeader verifies the metadata header format in the build log.
func TestBuildLogHeader(t *testing.T) {
	c := cache.New(t.TempDir())
	require.NoError(t, c.EnsureDir())

	var buf bytes.Buffer
	_, closer, err := c.CreateBuildLog(&buf, "deadbeef", "2.0.0")
	require.NoError(t, err)
	closer()

	logData, err := os.ReadFile(filepath.Join(c.Dir(), "logs", "last_build.log"))
	require.NoError(t, err)

	headerLine := strings.SplitN(string(logData), "\n", 2)[0]
	assert.True(t, strings.HasPrefix(headerLine, "# zone build |"), "header must start with '# zone build |'")
	assert.Contains(t, headerLine, "config hash: deadbeef", "header must contain config hash value")
	assert.Contains(t, headerLine, "zone 2.0.0", "header must contain version")
	// RFC3339 timestamp should be present (contains T and + or Z)
	assert.True(t, strings.Contains(headerLine, "T") && (strings.Contains(headerLine, "Z") || strings.Contains(headerLine, "+")),
		"header must contain RFC3339 timestamp")
}

// ---------------------------------------------------------------------------
// Integration test: zone clean command
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Exit code sentinel tests
// ---------------------------------------------------------------------------

// TestExitCodeLockContentionSentinel verifies that errors returned by Lock.Acquire()
// on contention are detectable via errors.Is(err, cache.ErrLockContention).
// This is the precondition for main.go's exit code 5 mapping.
// Full binary e2e test becomes possible once Phase 6 wires zone launch to call Lock.Acquire().
func TestExitCodeLockContentionSentinel(t *testing.T) {
	dir := t.TempDir()
	zoneDir := filepath.Join(dir, ".zone")
	require.NoError(t, os.MkdirAll(zoneDir, 0755))

	l1 := cache.NewLock(zoneDir)
	require.NoError(t, l1.Acquire())
	defer l1.Release()

	l2 := cache.NewLock(zoneDir)
	err := l2.Acquire()
	require.Error(t, err)

	// This is the exact check main.go uses to decide exit code 5
	assert.True(t, errors.Is(err, cache.ErrLockContention),
		"errors.Is(err, cache.ErrLockContention) must be true for main.go exit code 5 mapping")
}

// TestExitCodeGenericError verifies that a generic error does NOT match ErrLockContention.
func TestExitCodeGenericError(t *testing.T) {
	genericErr := fmt.Errorf("something went wrong")
	assert.False(t, errors.Is(genericErr, cache.ErrLockContention),
		"generic errors must NOT match ErrLockContention")
}

// ---------------------------------------------------------------------------
// Integration test: zone clean command
// ---------------------------------------------------------------------------

// TestCleanCommand verifies that `zone clean` removes the .zone/ directory.
func TestCleanCommand(t *testing.T) {
	dir := t.TempDir()
	// Create a .zone/ dir with a dummy file
	zoneDir := filepath.Join(dir, ".zone")
	require.NoError(t, os.MkdirAll(zoneDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(zoneDir, "test.txt"), []byte("data"), 0644))

	stdout, stderr, exitCode := runZone(t, dir, os.Environ(), "clean")
	require.Equal(t, 0, exitCode, "zone clean should exit 0; stderr: %s", stderr)

	_, err := os.Stat(zoneDir)
	assert.True(t, os.IsNotExist(err), ".zone/ directory must not exist after zone clean")
	assert.Contains(t, stdout, "Removed .zone/ cache directory", "stdout should confirm removal")
}
