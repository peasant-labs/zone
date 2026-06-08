// Tests for CodexCLI harness method return values and Validate().
package tests

import (
	"testing"

	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/internal/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getCodexCLI(t *testing.T, cfg *config.HarnessConfig) harness.Harness {
	t.Helper()
	if cfg == nil {
		cfg = &config.HarnessConfig{}
	}
	h, err := harness.Get("codex-cli", cfg)
	require.NoError(t, err)
	require.NotNil(t, h)
	return h
}

func TestCodexCLIInstall(t *testing.T) {
	h := getCodexCLI(t, nil)
	assert.Equal(t, []string{"npm install -g @openai/codex"}, h.InstallCommands())
}

func TestCodexCLIInstallVersioned(t *testing.T) {
	h := getCodexCLI(t, &config.HarnessConfig{Version: "0.134.0"})
	assert.Equal(t, []string{"npm install -g @openai/codex@0.134.0"}, h.InstallCommands())
}

func TestCodexCLIHealthCheck(t *testing.T) {
	h := getCodexCLI(t, nil)
	assert.Equal(t, "codex --version", h.HealthCheck())
}

func TestCodexCLIEntrypoint(t *testing.T) {
	h := getCodexCLI(t, nil)
	assert.Equal(t, "codex", h.EntrypointCommand())
}

func TestCodexCLIHomeConfigDir(t *testing.T) {
	h := getCodexCLI(t, nil)
	assert.Equal(t, "~/.codex", h.HomeConfigDir())
}

func TestCodexCLINeedsNode(t *testing.T) {
	h := getCodexCLI(t, nil)
	assert.True(t, h.NeedsNode())
}

func TestCodexCLIValidateAllowsSkipPermissions(t *testing.T) {
	h := getCodexCLI(t, &config.HarnessConfig{SkipPermissions: boolPtr(true)})
	require.NotNil(t, h)
}

func TestCodexCLIValidateRejectsPythonVersion(t *testing.T) {
	_, err := harness.Get("codex-cli", &config.HarnessConfig{PythonVersion: "3.12"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `does not support key "python_version"`)
	assert.Contains(t, err.Error(), `specific to "aider"`)
}

func TestCodexCLIValidateRejectsCustomKeys(t *testing.T) {
	_, err := harness.Get("codex-cli", &config.HarnessConfig{EntrypointCommand: "foo"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `does not support key "entrypoint_command"`)
	assert.Contains(t, err.Error(), `specific to "custom"`)
}
