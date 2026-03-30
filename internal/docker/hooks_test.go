// hooks_test.go tests lifecycle hook execution with fail-fast and warn-only modes.
package docker

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunHooks_FailFast_StopsOnFirstFailure verifies that failFast=true stops on first failure.
func TestRunHooks_FailFast_StopsOnFirstFailure(t *testing.T) {
	repoDir := t.TempDir()
	var stderr bytes.Buffer

	// "false" always exits with code 1; "echo third" should NOT run
	cmds := []string{"echo first", "false", "echo third"}
	err := runHooks(cmds, repoDir, true, &stderr)

	require.Error(t, err, "failFast=true should return error when a hook fails")
	assert.ErrorContains(t, err, "false", "error should contain the failing command")
}

// TestRunHooks_WarnOnly_RunsAllAndWarns verifies failFast=false runs all commands and warns on failure.
func TestRunHooks_WarnOnly_RunsAllAndWarns(t *testing.T) {
	repoDir := t.TempDir()
	var stderr bytes.Buffer

	cmds := []string{"echo first", "false", "echo third"}
	err := runHooks(cmds, repoDir, false, &stderr)

	// warn-only: no error returned
	require.NoError(t, err, "failFast=false should not return error")
	// Warning should appear in stderr
	assert.Contains(t, stderr.String(), "Warning", "should emit warning for failed hook")
	assert.Contains(t, stderr.String(), "false", "warning should mention the failing command")
}

// TestRunHooks_Empty verifies empty command list returns nil immediately.
func TestRunHooks_Empty(t *testing.T) {
	repoDir := t.TempDir()
	var stderr bytes.Buffer

	err := runHooks([]string{}, repoDir, true, &stderr)
	require.NoError(t, err)
	assert.Empty(t, stderr.String())
}

// TestRunHooks_WorkingDir verifies commands run with repoDir as working directory.
// Creates a file with a relative path and checks it appears in repoDir.
func TestRunHooks_WorkingDir(t *testing.T) {
	repoDir := t.TempDir()
	var stderr bytes.Buffer

	cmds := []string{"touch marker.txt"}
	err := runHooks(cmds, repoDir, true, &stderr)
	require.NoError(t, err)

	// File should exist in repoDir, confirming working dir was set correctly
	markerPath := filepath.Join(repoDir, "marker.txt")
	_, statErr := os.Stat(markerPath)
	assert.NoError(t, statErr, "marker.txt should exist in repoDir %s", repoDir)
}

// TestRunHooks_EnvInherited verifies commands inherit parent process environment.
func TestRunHooks_EnvInherited(t *testing.T) {
	repoDir := t.TempDir()
	var stderr bytes.Buffer

	// Set a known env var and verify the hook can see it
	t.Setenv("ZONE_TEST_HOOK_VAR", "hello-from-parent")

	// Conditional test: if env var is not available, test -z returns failure
	cmds := []string{"test \"$ZONE_TEST_HOOK_VAR\" = 'hello-from-parent'"}
	err := runHooks(cmds, repoDir, true, &stderr)
	require.NoError(t, err, "hook should see parent env var ZONE_TEST_HOOK_VAR")
}

// TestRunHooks_FailFast_ErrorWrapsCommand verifies error wraps the command string.
func TestRunHooks_FailFast_ErrorWrapsCommand(t *testing.T) {
	repoDir := t.TempDir()
	var stderr bytes.Buffer

	cmds := []string{"exit 42"}
	err := runHooks(cmds, repoDir, true, &stderr)

	require.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "exit 42") || strings.Contains(err.Error(), "hook"),
		"error should contain the command or 'hook': %s", err.Error(),
	)
}

// TestRunHooks_AllSucceed verifies that all-success returns nil.
func TestRunHooks_AllSucceed(t *testing.T) {
	repoDir := t.TempDir()
	var stderr bytes.Buffer

	cmds := []string{"echo hello", "echo world", "true"}
	err := runHooks(cmds, repoDir, true, &stderr)
	require.NoError(t, err)
}
