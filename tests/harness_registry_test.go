// Tests for Harness interface, BaseHarness defaults, and registry Get().
package tests

import (
	"strings"
	"testing"

	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/internal/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHarnessInterface verifies ClaudeCode satisfies the Harness interface at compile time.
func TestHarnessInterface(t *testing.T) {
	// compile-time assertion: var _ harness.Harness = (*harness.ClaudeCode)(nil)
	// We test this at runtime by calling Get and using the returned value as Harness.
	h, err := harness.Get("claude-code", &config.HarnessConfig{})
	require.NoError(t, err)
	require.NotNil(t, h)
	// h is a harness.Harness — if ClaudeCode doesn't satisfy the interface, this won't compile.
	var _ harness.Harness = h
}

// TestBaseHarnessDefaults verifies BaseHarness default method implementations.
func TestBaseHarnessDefaults(t *testing.T) {
	b := harness.BaseHarness{}
	assert.Equal(t, "", b.Version())
	assert.Equal(t, "", b.HealthCheck())
	assert.Equal(t, "", b.PromptFlag())
	assert.Nil(t, b.PostInstallCommands())
	assert.Nil(t, b.ExtraConfigDirs())
	assert.Nil(t, b.ShellRC())
	assert.Nil(t, b.Aliases())
	assert.Equal(t, "", b.WelcomeMessage())
	assert.NoError(t, b.Validate())
}

// TestHarnessRegistryGet verifies Get("claude-code") returns a valid harness with correct name.
func TestHarnessRegistryGet(t *testing.T) {
	h, err := harness.Get("claude-code", &config.HarnessConfig{})
	require.NoError(t, err)
	require.NotNil(t, h)
	assert.Equal(t, "claude-code", h.Name())
}

// TestHarnessRegistryUnknown verifies Get returns an error for unknown harness names.
func TestHarnessRegistryUnknown(t *testing.T) {
	h, err := harness.Get("nonexistent", &config.HarnessConfig{})
	assert.Nil(t, h)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "unknown harness"),
		"expected error to contain 'unknown harness', got: %s", err.Error())
	assert.True(t, strings.Contains(err.Error(), "available:"),
		"expected error to contain 'available:', got: %s", err.Error())
}

// TestRegistryAllNames verifies all 6 harness names are registered.
func TestRegistryAllNames(t *testing.T) {
	// For registrations that pass Validate(), verify Get() works.
	// For stubs that always fail Validate(), verify the error is about "not yet implemented"
	// (not "unknown harness") — they ARE registered.
	stubs := []string{"gemini-cli", "aider"}
	for _, name := range stubs {
		h, err := harness.Get(name, &config.HarnessConfig{})
		assert.Nil(t, h, "stub harness %q should return nil harness due to Validate() error", name)
		require.Error(t, err, "stub harness %q should return an error", name)
		assert.False(t, strings.Contains(err.Error(), "unknown harness"),
			"stub harness %q should be registered (not 'unknown'), got: %s", name, err.Error())
		assert.True(t, strings.Contains(err.Error(), "not yet fully implemented"),
			"stub harness %q error should mention 'not yet fully implemented', got: %s", name, err.Error())
	}

	// custom requires entrypoint_command to pass Validate(); with empty config it fails too
	h, err := harness.Get("custom", &config.HarnessConfig{})
	assert.Nil(t, h)
	require.Error(t, err)
	assert.False(t, strings.Contains(err.Error(), "unknown harness"),
		"'custom' should be registered, got: %s", err.Error())

	// claude-code with empty config should succeed
	h, err = harness.Get("claude-code", &config.HarnessConfig{})
	require.NoError(t, err)
	require.NotNil(t, h)

	h, err = harness.Get("codex-cli", &config.HarnessConfig{})
	require.NoError(t, err)
	require.NotNil(t, h)
}
