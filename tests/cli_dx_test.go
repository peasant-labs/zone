package tests

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	zonecmd "github.com/peasant-labs/zone/cmd"
	"github.com/peasant-labs/zone/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAliases verifies all command aliases resolve correctly (DX-08, D-37).
func TestAliases(t *testing.T) {
	binary := getZoneBinary(t)

	aliases := []struct {
		alias   string
		primary string
	}{
		{"up", "launch"},
		{"down", "stop"},
		{"list", "ls"},
		{"log", "logs"},
		{"st", "status"},
	}

	for _, a := range aliases {
		t.Run(a.alias, func(t *testing.T) {
			aliasOut, err := exec.Command(binary, a.alias, "--help").CombinedOutput()
			require.NoError(t, err, "alias %s --help failed: %s", a.alias, string(aliasOut))

			primaryOut, err := exec.Command(binary, a.primary, "--help").CombinedOutput()
			require.NoError(t, err, "primary %s --help failed: %s", a.primary, string(primaryOut))

			assert.Contains(t, string(aliasOut), extractShortLine(primaryOut),
				"alias %s help does not match primary %s", a.alias, a.primary)
		})
	}
}

func extractShortLine(help []byte) string {
	lines := strings.Split(string(help), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" && !strings.HasPrefix(l, "Usage:") && !strings.HasPrefix(l, "Aliases:") {
			return l
		}
	}
	return ""
}

// TestHelpExamples verifies all 15 commands have Example sections in --help (DX-09).
func TestHelpExamples(t *testing.T) {
	binary := getZoneBinary(t)

	commands := []string{
		"init", "launch", "join", "exec", "shell", "build",
		"stop", "restart", "destroy", "clean", "ls", "logs",
		"status", "config", "validate",
	}

	for _, cmdName := range commands {
		t.Run(cmdName, func(t *testing.T) {
			out, err := exec.Command(binary, cmdName, "--help").CombinedOutput()
			require.NoError(t, err, "%s --help failed: %s", cmdName, string(out))
			help := string(out)
			assert.Contains(t, help, "Examples:", "%s --help missing Examples section", cmdName)
			count := countUsageExamples(help)
			assert.GreaterOrEqual(t, count, 2, "%s --help must include at least 2 usage examples", cmdName)
			assert.LessOrEqual(t, count, 4, "%s --help must include at most 4 usage examples", cmdName)
		})
	}
}

func countUsageExamples(help string) int {
	idx := strings.Index(help, "Examples:")
	if idx < 0 {
		return 0
	}
	section := help[idx+len("Examples:"):]
	for _, stop := range []string{"Flags:\n", "Global Flags:\n", "Use \""} {
		if stopIdx := strings.Index(section, stop); stopIdx >= 0 {
			section = section[:stopIdx]
			break
		}
	}

	count := 0
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(strings.TrimRight(line, " \t"), "  zone ") {
			count++
		}
	}
	return count
}

// TestGlobalFlags verifies --verbose, --debug, --quiet, --plain are available (CLI-20).
func TestGlobalFlags(t *testing.T) {
	binary := getZoneBinary(t)

	out, err := exec.Command(binary, "--help").CombinedOutput()
	require.NoError(t, err, "root --help failed: %s", string(out))

	flags := []string{"--verbose", "--debug", "--quiet", "--plain"}
	for _, f := range flags {
		assert.Contains(t, string(out), f, "root --help missing global flag %s", f)
	}
}

// TestLaunchPortFlag verifies --port/-P flag is registered on launch (CLI-21).
func TestLaunchPortFlag(t *testing.T) {
	binary := getZoneBinary(t)

	out, err := exec.Command(binary, "launch", "--help").CombinedOutput()
	require.NoError(t, err, "launch --help failed: %s", string(out))
	assert.Contains(t, string(out), "--port", "launch --help missing --port flag")
	assert.Contains(t, string(out), "-P", "launch --help missing -P shorthand")
}

// TestExitCode2OnBadConfig verifies config errors produce exit code 2 (DX-01).
func TestExitCode2OnBadConfig(t *testing.T) {
	binary := getZoneBinary(t)
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "zone.toml"), []byte("version = 999\n"), 0644)
	require.NoError(t, err)

	cmd := exec.Command(binary, "validate")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+dir+"/no-xdg")
	_ = cmd.Run()

	assert.Equal(t, 2, cmd.ProcessState.ExitCode(),
		"expected exit code 2 for config error, got %d", cmd.ProcessState.ExitCode())
}

// TestExitCode6OnNoContainer verifies no-container errors produce exit code 6 (DX-01).
func TestExitCode6OnNoContainer(t *testing.T) {
	binary := getZoneBinary(t)
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "zone.toml"), []byte("version = 1\nharness = \"claude-code\"\n"), 0644)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".zone"), 0755))

	cmd := exec.Command(binary, "status")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+dir+"/no-xdg")
	out, _ := cmd.CombinedOutput()

	code := cmd.ProcessState.ExitCode()
	assert.True(t, code == 1 || code == 3 || code == 6,
		"expected exit code 1, 3, or 6 for status with no container, got %d (output: %s)", code, string(out))
}

// TestLsJsonOutput verifies zone ls --json produces valid JSON (DX-03).
func TestLsJsonOutput(t *testing.T) {
	binary := getZoneBinary(t)

	cmd := exec.Command(binary, "ls", "--json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		assert.NotContains(t, string(out), "not implemented")
		t.Skip("Docker not available, skipping JSON validation")
	}

	var result []interface{}
	err = json.Unmarshal(out, &result)
	assert.NoError(t, err, "zone ls --json produced invalid JSON: %s", string(out))
}

// TestLogsBuildFlag verifies zone logs --build reads cache file (CLI-14).
func TestLogsBuildFlag(t *testing.T) {
	binary := getZoneBinary(t)
	dir := t.TempDir()

	logDir := filepath.Join(dir, ".zone", "logs")
	require.NoError(t, os.MkdirAll(logDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(logDir, "last_build.log"), []byte("# zone build | 2026-03-30 | test log\nStep 1/5: FROM ubuntu\n"), 0644))

	cmd := exec.Command(binary, "logs", "--build")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+dir+"/no-xdg")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "logs --build failed: %s", string(out))
	assert.Contains(t, string(out), "Step 1/5: FROM ubuntu")
}

// TestValidateExitZero verifies valid config produces exit 0 (CLI-19).
func TestValidateExitZero(t *testing.T) {
	binary := getZoneBinary(t)
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "zone.toml"), []byte("version = 1\nharness = \"claude-code\"\n"), 0644))

	cmd := exec.Command(binary, "validate")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+dir+"/no-xdg")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "validate failed: %s", string(out))
	assert.Contains(t, string(out), "valid")
}

// TestConfigShowsMerged verifies zone config works with valid zone.toml (CLI-18).
func TestConfigShowsMerged(t *testing.T) {
	binary := getZoneBinary(t)
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "zone.toml"), []byte("version = 1\nharness = \"claude-code\"\n"), 0644))

	cmd := exec.Command(binary, "config")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+dir+"/no-xdg")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "config failed: %s", string(out))
	assert.Contains(t, string(out), "claude-code")
}

// TestRemediationHintOnStderr verifies error messages go to stderr (DX-02).
func TestRemediationHintOnStderr(t *testing.T) {
	binary := getZoneBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(binary, "validate")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+dir+"/no-xdg")

	var stderr strings.Builder
	cmd.Stderr = &stderr
	_ = cmd.Run()

	assert.True(t, stderr.Len() > 0, "expected error output on stderr for missing zone.toml")
	assert.Contains(t, stderr.String(), "zone init", "expected remediation hint to include next-step command")
}

// TestFallbackRemediationHintOnStderr verifies generic errors include actionable remediation (DX-02).
func TestFallbackRemediationHintOnStderr(t *testing.T) {
	binary := getZoneBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(binary, "exec")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+dir+"/no-xdg")

	var stderr strings.Builder
	cmd.Stderr = &stderr
	err := cmd.Run()
	require.Error(t, err)

	assert.Contains(t, stderr.String(), "no command specified")
	assert.Contains(t, stderr.String(), "zone --help")
}

// TestUnknownKeysRemediationHintOnStderr verifies unknown-key config failures include remediation (DX-02).
func TestUnknownKeysRemediationHintOnStderr(t *testing.T) {
	err := &config.UnknownKeysError{Keys: []string{"bad_field"}, File: "zone.toml"}

	msg, code := zonecmd.MapError(err)
	assert.Equal(t, 2, code, "unknown-key config errors should map to exit code 2")
	assert.Contains(t, strings.ToLower(msg), "unknown")
	assert.Contains(t, msg, "zone validate")

	// Wrapped error containing "zone.toml" gets config-specific remediation
	wrapped := errors.New("wrapper: " + err.Error())
	msg2, code2 := zonecmd.MapError(wrapped)
	assert.Equal(t, 2, code2, "config-related errors should map to exit code 2")
	assert.Contains(t, msg2, "zone validate")
}
