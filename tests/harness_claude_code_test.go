// Tests for ClaudeCode harness method return values and Validate().
package tests

import (
	"testing"

	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/internal/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getClaudeCode(t *testing.T, cfg *config.HarnessConfig) harness.Harness {
	t.Helper()
	if cfg == nil {
		cfg = &config.HarnessConfig{}
	}
	h, err := harness.Get("claude-code", cfg)
	require.NoError(t, err)
	require.NotNil(t, h)
	return h
}

// TestClaudeCodeInstall verifies default (no version) install commands.
func TestClaudeCodeInstall(t *testing.T) {
	h := getClaudeCode(t, &config.HarnessConfig{})
	cmds := h.InstallCommands()
	require.Len(t, cmds, 1)
	assert.Equal(t, "npm install -g @anthropic-ai/claude-code", cmds[0])
}

// TestClaudeCodeInstallVersioned verifies versioned install appends @version.
func TestClaudeCodeInstallVersioned(t *testing.T) {
	h := getClaudeCode(t, &config.HarnessConfig{Version: "1.0.26"})
	cmds := h.InstallCommands()
	require.Len(t, cmds, 1)
	assert.Equal(t, "npm install -g @anthropic-ai/claude-code@1.0.26", cmds[0])
}

// TestClaudeCodeHealthCheck verifies health check command.
func TestClaudeCodeHealthCheck(t *testing.T) {
	h := getClaudeCode(t, nil)
	assert.Equal(t, "claude --version", h.HealthCheck())
}

// TestClaudeCodeEntrypoint verifies entrypoint command.
func TestClaudeCodeEntrypoint(t *testing.T) {
	h := getClaudeCode(t, nil)
	assert.Equal(t, "claude", h.EntrypointCommand())
}

// TestClaudeCodePromptFlag verifies prompt flag.
func TestClaudeCodePromptFlag(t *testing.T) {
	h := getClaudeCode(t, nil)
	assert.Equal(t, "-p", h.PromptFlag())
}

// TestClaudeCodeRequiredEnvVars verifies required env vars list.
func TestClaudeCodeRequiredEnvVars(t *testing.T) {
	h := getClaudeCode(t, nil)
	vars := h.RequiredEnvVars()
	require.Len(t, vars, 1)
	assert.Equal(t, "ANTHROPIC_API_KEY", vars[0])
}

// TestClaudeCodeHomeConfigDir verifies home config directory.
func TestClaudeCodeHomeConfigDir(t *testing.T) {
	h := getClaudeCode(t, nil)
	assert.Equal(t, "~/.claude", h.HomeConfigDir())
}

// TestClaudeCodeNeedsNode verifies NeedsNode returns true.
func TestClaudeCodeNeedsNode(t *testing.T) {
	h := getClaudeCode(t, nil)
	assert.True(t, h.NeedsNode())
}

// TestClaudeCodeNeedsPython verifies NeedsPython returns false.
func TestClaudeCodeNeedsPython(t *testing.T) {
	h := getClaudeCode(t, nil)
	assert.False(t, h.NeedsPython())
}

// TestClaudeCodeDefaultPackages verifies all default package lists are nil.
func TestClaudeCodeDefaultPackages(t *testing.T) {
	h := getClaudeCode(t, nil)
	assert.Nil(t, h.DefaultAptPackages())
	assert.Nil(t, h.DefaultNpmPackages())
	assert.Nil(t, h.DefaultPipPackages())
}

// TestClaudeCodeValidateClean verifies zero-value config passes validation.
func TestClaudeCodeValidateClean(t *testing.T) {
	h, err := harness.Get("claude-code", &config.HarnessConfig{})
	require.NoError(t, err)
	require.NotNil(t, h)
}

// TestClaudeCodeValidatePythonVersion verifies python_version is rejected.
func TestClaudeCodeValidatePythonVersion(t *testing.T) {
	_, err := harness.Get("claude-code", &config.HarnessConfig{PythonVersion: "3.12"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `does not support key "python_version"`)
	assert.Contains(t, err.Error(), `specific to "aider"`)
}

// TestClaudeCodeValidateInstallCommands verifies install_commands is rejected.
func TestClaudeCodeValidateInstallCommands(t *testing.T) {
	_, err := harness.Get("claude-code", &config.HarnessConfig{InstallCommands: []string{"apt install foo"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `does not support key "install_commands"`)
	assert.Contains(t, err.Error(), `specific to "custom"`)
}

// TestClaudeCodeValidateEntrypointCommand verifies entrypoint_command is rejected.
func TestClaudeCodeValidateEntrypointCommand(t *testing.T) {
	_, err := harness.Get("claude-code", &config.HarnessConfig{EntrypointCommand: "foo"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `does not support key "entrypoint_command"`)
	assert.Contains(t, err.Error(), `specific to "custom"`)
}

// TestSkipPermissionsDefault verifies nil SkipPermissions (false) is accepted.
func TestSkipPermissionsDefault(t *testing.T) {
	h, err := harness.Get("claude-code", &config.HarnessConfig{SkipPermissions: nil})
	require.NoError(t, err)
	require.NotNil(t, h)
}
