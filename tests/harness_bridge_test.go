package tests

import (
	"testing"

	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/internal/docker"
	"github.com/peasant-labs/zone/internal/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testMergedConfig returns a MergedConfig with sensible test defaults.
func testMergedConfig() *config.MergedConfig {
	return &config.MergedConfig{
		Zone: config.ZoneConfig{
			BaseImage: "ubuntu:24.04",
			Shell:     "bash",
			Harness:   "claude-code",
		},
		Workspace: config.WorkspaceConfig{
			MountPath: "/workspace",
		},
		Packages: config.PackagesConfig{
			Apt: []string{"git", "curl"},
		},
	}
}

// getClaudeCodeHarness constructs a validated claude-code harness for tests.
func getClaudeCodeHarness(t *testing.T) harness.Harness {
	t.Helper()
	h, err := harness.Get("claude-code", &config.HarnessConfig{})
	require.NoError(t, err)
	return h
}

// TestBuildDockerfileDataClaudeCode verifies that BuildDockerfileData populates all
// DockerfileData fields correctly for a default claude-code harness.
func TestBuildDockerfileDataClaudeCode(t *testing.T) {
	h := getClaudeCodeHarness(t)
	cfg := testMergedConfig()

	data := docker.BuildDockerfileData(h, cfg)

	assert.True(t, data.NeedsNode, "claude-code requires Node")
	assert.Equal(t, "22", data.NodeVersion, "default NodeVersion should be 22")
	assert.Equal(t, []string{"npm install -g @anthropic-ai/claude-code"}, data.HarnessInstallCommands)
	assert.Equal(t, "claude --version", data.HealthCheck)
	assert.Equal(t, "ubuntu:24.04", data.BaseImage)
	assert.Contains(t, data.AptPackages, "git")
	assert.Equal(t, "bash", data.Shell)
	assert.Equal(t, "/workspace", data.MountPath)
	assert.False(t, data.NeedsPython, "claude-code does not need Python")
}

// TestBuildDockerfileDataNodeVersionOverride verifies that cfg.Harness.NodeVersion
// overrides the default "22" when set.
func TestBuildDockerfileDataNodeVersionOverride(t *testing.T) {
	h := getClaudeCodeHarness(t)
	cfg := testMergedConfig()
	cfg.Harness.NodeVersion = "20"

	data := docker.BuildDockerfileData(h, cfg)

	assert.Equal(t, "20", data.NodeVersion)
}

// TestBuildDockerfileDataPythonVersionDefault verifies that PythonVersion defaults to
// "3.12" even when the harness does not need Python.
func TestBuildDockerfileDataPythonVersionDefault(t *testing.T) {
	h := getClaudeCodeHarness(t)
	cfg := testMergedConfig()
	// cfg.Harness.PythonVersion is empty

	data := docker.BuildDockerfileData(h, cfg)

	assert.False(t, data.NeedsPython)
	assert.Equal(t, "3.12", data.PythonVersion, "PythonVersion should default to 3.12 for future use")
}

// TestBuildDockerfileDataPackageMerge verifies that cfg.Packages lists are merged
// with harness DefaultAptPackages/DefaultNpmPackages/DefaultPipPackages.
func TestBuildDockerfileDataPackageMerge(t *testing.T) {
	h := getClaudeCodeHarness(t)
	cfg := testMergedConfig()
	cfg.Packages.Apt = []string{"git", "curl"}
	cfg.Packages.Npm = []string{"typescript"}
	cfg.Packages.Pip = []string{"requests"}

	data := docker.BuildDockerfileData(h, cfg)

	// ClaudeCode DefaultAptPackages() returns nil, so only config packages
	assert.Equal(t, []string{"git", "curl"}, data.AptPackages)
	// ClaudeCode DefaultNpmPackages() returns nil, so only config packages
	assert.Equal(t, []string{"typescript"}, data.NpmPackages)
	// ClaudeCode DefaultPipPackages() returns nil, so only config packages
	assert.Equal(t, []string{"requests"}, data.PipPackages)
}

// TestBuildDockerfileDataPostInstallCommands verifies PostInstallCommands comes from
// the harness.PostInstallCommands() method.
func TestBuildDockerfileDataPostInstallCommands(t *testing.T) {
	// Use custom harness with post-install commands via install_commands
	// ClaudeCode.PostInstallCommands() returns nil (via BaseHarness default)
	h := getClaudeCodeHarness(t)
	cfg := testMergedConfig()

	data := docker.BuildDockerfileData(h, cfg)

	// ClaudeCode.PostInstallCommands() returns nil via BaseHarness
	assert.Nil(t, data.PostInstallCommands)
}

// TestBuildDockerfileDataInstallZsh verifies that InstallZsh=true when Shell is "zsh"
// and InstallZsh=false when Shell is "bash".
func TestBuildDockerfileDataInstallZsh(t *testing.T) {
	h := getClaudeCodeHarness(t)

	cfgZsh := testMergedConfig()
	cfgZsh.Zone.Shell = "zsh"
	dataZsh := docker.BuildDockerfileData(h, cfgZsh)
	assert.True(t, dataZsh.InstallZsh, "Shell=zsh should set InstallZsh=true")

	cfgBash := testMergedConfig()
	cfgBash.Zone.Shell = "bash"
	dataBash := docker.BuildDockerfileData(h, cfgBash)
	assert.False(t, dataBash.InstallZsh, "Shell=bash should set InstallZsh=false")
}

// TestBuildEntrypointDataClaudeCode verifies EntrypointCommand, Shell, and MountPath
// are populated correctly for claude-code.
func TestBuildEntrypointDataClaudeCode(t *testing.T) {
	h := getClaudeCodeHarness(t)
	cfg := testMergedConfig()

	data := docker.BuildEntrypointData(h, cfg)

	assert.Equal(t, "claude", data.EntrypointCommand)
	assert.Equal(t, "bash", data.Shell)
	assert.Equal(t, "/workspace", data.MountPath)
}

// TestBuildEntrypointDataConfigCopy verifies that HomeConfigDir generates one
// ConfigCopyCommand containing the config dir path.
func TestBuildEntrypointDataConfigCopy(t *testing.T) {
	h := getClaudeCodeHarness(t)
	cfg := testMergedConfig()

	data := docker.BuildEntrypointData(h, cfg)

	// ClaudeCode.HomeConfigDir() = "~/.claude", ExtraConfigDirs() = nil
	require.Len(t, data.ConfigCopyCommands, 1, "one command for ~/.claude")
	assert.Contains(t, data.ConfigCopyCommands[0], "~/.claude")
}

// TestBuildEntrypointDataCustomConfigDirs verifies that a custom harness with multiple
// ConfigDirs generates one ConfigCopyCommand per directory.
func TestBuildEntrypointDataCustomConfigDirs(t *testing.T) {
	h, err := harness.Get("custom", &config.HarnessConfig{
		EntrypointCommand: "mytool",
		ConfigDirs:        []string{"~/.mytool", "~/.other"},
	})
	require.NoError(t, err)
	cfg := testMergedConfig()

	data := docker.BuildEntrypointData(h, cfg)

	// Custom.HomeConfigDir() = "", ExtraConfigDirs() = ["~/.mytool","~/.other"]
	require.Len(t, data.ConfigCopyCommands, 2, "two commands for two ConfigDirs")
	assert.Contains(t, data.ConfigCopyCommands[0], "~/.mytool")
	assert.Contains(t, data.ConfigCopyCommands[1], "~/.other")
}

// TestBuildEntrypointDataNoConfigDir verifies that ConfigCopyCommands is empty when
// HomeConfigDir="": and ExtraConfigDirs=nil.
func TestBuildEntrypointDataNoConfigDir(t *testing.T) {
	h, err := harness.Get("custom", &config.HarnessConfig{
		EntrypointCommand: "mytool",
	})
	require.NoError(t, err)
	cfg := testMergedConfig()

	data := docker.BuildEntrypointData(h, cfg)

	// Custom.HomeConfigDir() = "", ExtraConfigDirs() = nil
	assert.Empty(t, data.ConfigCopyCommands)
}

// TestBuildShellRCDataClaudeCode verifies HarnessName, MountPath, Aliases, ShellRC,
// and WelcomeMessage are populated from the harness for claude-code.
func TestBuildShellRCDataClaudeCode(t *testing.T) {
	h := getClaudeCodeHarness(t)
	cfg := testMergedConfig()

	data := docker.BuildShellRCData(h, cfg)

	assert.Equal(t, "claude-code", data.HarnessName)
	assert.Equal(t, "/workspace", data.MountPath)
	assert.Nil(t, data.Aliases, "claude-code has no aliases")
	assert.Nil(t, data.ShellRC, "claude-code has no shell RC lines")
	assert.Equal(t, "", data.WelcomeMessage)
}

// TestBuildShellRCDataCustom verifies that custom harness Aliases and ShellRC are
// forwarded through the ShellRCData struct.
func TestBuildShellRCDataCustom(t *testing.T) {
	h, err := harness.Get("custom", &config.HarnessConfig{
		EntrypointCommand: "my-tool",
		CustomAliases:     map[string]string{"mt": "my-tool"},
		CustomShellRC:     []string{"export FOO=bar"},
	})
	require.NoError(t, err)
	cfg := testMergedConfig()

	data := docker.BuildShellRCData(h, cfg)

	assert.Equal(t, "custom", data.HarnessName)
	assert.Equal(t, map[string]string{"mt": "my-tool"}, data.Aliases)
	assert.Equal(t, []string{"export FOO=bar"}, data.ShellRC)
}
