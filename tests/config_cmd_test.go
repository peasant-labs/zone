// Integration tests for zone config and zone validate commands (CFG-07, CFG-08).
package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Binary build helper
// ---------------------------------------------------------------------------

var (
	cmdBuildOnce   sync.Once
	cmdZoneBinary  string
	cmdBuildErr    error
	cmdBinaryTmpDir string
)

func getZoneBinary(t *testing.T) string {
	t.Helper()
	cmdBuildOnce.Do(func() {
		var err error
		cmdBinaryTmpDir, err = os.MkdirTemp("", "zone-cmd-test-bin")
		if err != nil {
			cmdBuildErr = fmt.Errorf("create temp dir: %w", err)
			return
		}
		cmdZoneBinary = filepath.Join(cmdBinaryTmpDir, "zone")
		buildCmd := exec.Command("go", "build", "-o", cmdZoneBinary, ".")
		buildCmd.Dir = "/workspace/zone"
		out, buildErr := buildCmd.CombinedOutput()
		if buildErr != nil {
			cmdBuildErr = fmt.Errorf("build failed: %w\n%s", buildErr, out)
		}
	})
	if cmdBuildErr != nil {
		t.Fatalf("zone binary build failed: %v", cmdBuildErr)
	}
	return cmdZoneBinary
}

// runZone executes the zone binary from the given directory.
// env is the full environment to pass (use os.Environ() and append overrides).
func runZone(t *testing.T, dir string, env []string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	binary := getZoneBinary(t)
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	cmd.Env = env
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	exitCode = 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Logf("run error (not exit error): %v", err)
		exitCode = -1
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// setupDir creates a temp dir, optionally writes zone.toml and global config.
// Returns the tmpDir path and an env slice with XDG_CONFIG_HOME set.
func setupDir(t *testing.T, repoTOML string, globalTOML string) (string, []string) {
	t.Helper()
	tmpDir := t.TempDir()

	if repoTOML != "" {
		err := os.WriteFile(filepath.Join(tmpDir, "zone.toml"), []byte(repoTOML), 0644)
		require.NoError(t, err, "write zone.toml")
	}

	// Build environment based on os.Environ().
	env := os.Environ()

	if globalTOML != "" {
		globalDir := filepath.Join(tmpDir, "xdg", "zone")
		err := os.MkdirAll(globalDir, 0755)
		require.NoError(t, err, "create global config dir")
		err = os.WriteFile(filepath.Join(globalDir, "config.toml"), []byte(globalTOML), 0644)
		require.NoError(t, err, "write global config.toml")
		// Override XDG_CONFIG_HOME in the env slice.
		xdgPath := filepath.Join(tmpDir, "xdg")
		env = overrideEnv(env, "XDG_CONFIG_HOME", xdgPath)
	} else {
		// Point to an empty dir so no global config is found.
		env = overrideEnv(env, "XDG_CONFIG_HOME", filepath.Join(tmpDir, "no-xdg"))
	}

	return tmpDir, env
}

// overrideEnv returns a copy of env with the given key set to value.
func overrideEnv(env []string, key, value string) []string {
	prefix := key + "="
	var out []string
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			out = append(out, e)
		}
	}
	return append(out, prefix+value)
}

// ---------------------------------------------------------------------------
// TestConfigAnnotatedOutput (CFG-07): verify annotated TOML output format
// ---------------------------------------------------------------------------

func TestConfigAnnotatedOutput(t *testing.T) {
	repoTOML := `version = 1
harness = "claude-code"

[zone]
base_image = "ubuntu:24.04"
`
	globalTOML := `version = 1

[zone]
shell = "bash"
`
	dir, env := setupDir(t, repoTOML, globalTOML)

	stdout, stderr, exitCode := runZone(t, dir, env, "config")
	require.Equal(t, 0, exitCode, "config command should exit 0; stderr: %s", stderr)

	// version line with source comment
	assert.Contains(t, stdout, "version = 1", "should contain version field")
	assert.Contains(t, stdout, "# repo: zone.toml", "should contain repo source comment")

	// [zone] section header
	assert.Contains(t, stdout, "[zone]", "should contain [zone] section header")

	// harness annotated
	assert.Contains(t, stdout, `harness = "claude-code"`, "should contain harness value")

	// base_image annotated
	assert.Contains(t, stdout, `base_image = "ubuntu:24.04"`, "should contain base_image value")

	// shell from global
	assert.Contains(t, stdout, `shell = "bash"`, "should contain shell value")
	assert.Contains(t, stdout, "# global", "should contain global source annotation")
}

// ---------------------------------------------------------------------------
// TestConfigAnnotatedOutput_ListMerge: verify list merge comment block
// ---------------------------------------------------------------------------

func TestConfigAnnotatedOutput_ListMerge(t *testing.T) {
	repoTOML := `version = 1
harness = "claude-code"

[packages]
apt = ["wget"]
`
	globalTOML := `version = 1

[packages]
apt = ["git", "curl"]
`
	dir, env := setupDir(t, repoTOML, globalTOML)

	stdout, _, exitCode := runZone(t, dir, env, "config")
	require.Equal(t, 0, exitCode, "config command should exit 0")

	// [packages] section
	assert.Contains(t, stdout, "[packages]", "should contain [packages] section")

	// All three packages in the apt array
	assert.Contains(t, stdout, `"git"`, "should contain git from global")
	assert.Contains(t, stdout, `"curl"`, "should contain curl from global")
	assert.Contains(t, stdout, `"wget"`, "should contain wget from repo")

	// Comment block above apt showing provenance
	assert.Contains(t, stdout, "# apt:", "should contain apt comment showing provenance")
	assert.Contains(t, stdout, "global", "comment should mention global source")
	assert.Contains(t, stdout, "repo", "comment should mention repo source")
}

// ---------------------------------------------------------------------------
// TestConfigAnnotatedOutput_GlobalOnly: zone config --global without zone.toml
// ---------------------------------------------------------------------------

func TestConfigAnnotatedOutput_GlobalOnly(t *testing.T) {
	// No zone.toml, global config with shell default
	globalTOML := `version = 1

[zone]
shell = "bash"
base_image = "ubuntu:24.04"
`
	dir, env := setupDir(t, "", globalTOML)

	stdout, stderr, exitCode := runZone(t, dir, env, "config", "--global")
	require.Equal(t, 0, exitCode, "config --global should exit 0 without zone.toml; stderr: %s", stderr)

	// Should show default or global values with source annotations
	assert.True(t, len(stdout) > 0, "stdout should not be empty")
	assert.Contains(t, stdout, "[zone]", "should contain [zone] section")
}

// ---------------------------------------------------------------------------
// TestConfigAnnotatedOutput_NoZoneToml: zone config without zone.toml gives error
// ---------------------------------------------------------------------------

func TestConfigAnnotatedOutput_NoZoneToml(t *testing.T) {
	dir, env := setupDir(t, "", "")

	_, stderr, exitCode := runZone(t, dir, env, "config")
	assert.NotEqual(t, 0, exitCode, "config without zone.toml should exit non-zero")
	assert.Contains(t, stderr, "No zone.toml found", "should mention missing zone.toml")
	assert.Contains(t, stderr, "zone init", "should mention zone init command")
}

// ---------------------------------------------------------------------------
// TestConfigJSON (CFG-08): verify JSON output structure
// ---------------------------------------------------------------------------

func TestConfigJSON(t *testing.T) {
	repoTOML := `version = 1
harness = "claude-code"

[zone]
base_image = "ubuntu:24.04"
`
	globalTOML := `version = 1

[zone]
shell = "bash"
`
	dir, env := setupDir(t, repoTOML, globalTOML)

	stdout, stderr, exitCode := runZone(t, dir, env, "config", "--json")
	require.Equal(t, 0, exitCode, "config --json should exit 0; stderr: %s", stderr)

	// Parse stdout as JSON
	var result map[string]any
	err := json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err, "stdout should be valid JSON; got: %s", stdout)

	// version field: {"value": 1, "source": "repo: zone.toml"}
	versionField, ok := result["version"].(map[string]any)
	require.True(t, ok, "version should be an object with value/source")
	assert.Equal(t, float64(1), versionField["value"], "version.value should be 1")
	assert.Equal(t, "repo: zone.toml", versionField["source"], "version.source should be repo: zone.toml")

	// zone section
	zoneSection, ok := result["zone"].(map[string]any)
	require.True(t, ok, "zone should be a section object")

	// zone.shell from global
	shellField, ok := zoneSection["shell"].(map[string]any)
	require.True(t, ok, "zone.shell should be an object with value/source")
	assert.Equal(t, "bash", shellField["value"], "zone.shell.value should be bash")
	assert.Equal(t, "global", shellField["source"], "zone.shell.source should be global")

	// zone.base_image from repo
	baseImageField, ok := zoneSection["base_image"].(map[string]any)
	require.True(t, ok, "zone.base_image should be an object with value/source")
	assert.Equal(t, "ubuntu:24.04", baseImageField["value"], "zone.base_image.value should be ubuntu:24.04")
	assert.Equal(t, "repo: zone.toml", baseImageField["source"], "zone.base_image.source should be repo: zone.toml")
}

// ---------------------------------------------------------------------------
// TestConfigJSON_Structure: zone config --json --global produces valid JSON
// ---------------------------------------------------------------------------

func TestConfigJSON_Structure(t *testing.T) {
	dir, env := setupDir(t, "", "")

	stdout, stderr, exitCode := runZone(t, dir, env, "config", "--json", "--global")
	require.Equal(t, 0, exitCode, "config --json --global should exit 0; stderr: %s", stderr)

	// Validate JSON parses cleanly
	var result map[string]any
	err := json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err, "stdout should be valid JSON; got: %s", stdout)

	// Top-level keys should include main sections
	assert.Contains(t, result, "version", "should have version key")
	assert.Contains(t, result, "zone", "should have zone key")
	assert.Contains(t, result, "packages", "should have packages key")
	assert.Contains(t, result, "resources", "should have resources key")
	assert.Contains(t, result, "network", "should have network key")
}
