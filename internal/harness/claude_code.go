// claude_code.go implements the claude-code harness.
package harness

import (
	"fmt"

	"github.com/peasant-labs/zone/internal/config"
)

// ClaudeCode implements the Harness interface for the Anthropic Claude Code AI tool.
// It embeds BaseHarness for optional method defaults and overrides all required methods.
type ClaudeCode struct {
	BaseHarness
	config *config.HarnessConfig
}

// Name returns the harness identifier used in zone.toml.
func (c *ClaudeCode) Name() string { return "claude-code" }

// InstallCommands returns the npm install command, appending @version when set.
func (c *ClaudeCode) InstallCommands() []string {
	pkg := "@anthropic-ai/claude-code"
	if c.config.Version != "" {
		pkg += "@" + c.config.Version
	}
	return []string{"npm install -g " + pkg}
}

// HealthCheck returns the command used to verify claude-code is installed.
func (c *ClaudeCode) HealthCheck() string { return "claude --version" }

// EntrypointCommand returns the bare command name (extra args and prompt flag
// are appended by the entrypoint template at container runtime).
func (c *ClaudeCode) EntrypointCommand() string { return "claude" }

// PromptFlag returns the flag used to pass a prompt string to claude-code.
func (c *ClaudeCode) PromptFlag() string { return "-p" }

// RequiredEnvVars returns the environment variables that must be set before launch.
func (c *ClaudeCode) RequiredEnvVars() []string { return []string{"ANTHROPIC_API_KEY"} }

// HomeConfigDir returns the directory where claude-code stores its configuration.
func (c *ClaudeCode) HomeConfigDir() string { return "~/.claude" }

// NeedsNode returns true — claude-code is an npm package requiring Node.js.
func (c *ClaudeCode) NeedsNode() bool { return true }

// NeedsPython returns false — claude-code has no Python dependency.
func (c *ClaudeCode) NeedsPython() bool { return false }

// DefaultAptPackages returns nil — no extra apt packages beyond base image.
func (c *ClaudeCode) DefaultAptPackages() []string { return nil }

// DefaultNpmPackages returns nil — claude-code itself is installed via InstallCommands,
// not via the generic npm packages list.
func (c *ClaudeCode) DefaultNpmPackages() []string { return nil }

// DefaultPipPackages returns nil — no pip dependencies.
func (c *ClaudeCode) DefaultPipPackages() []string { return nil }

// Validate rejects HarnessConfig fields that belong to other harnesses.
// Common fields (version, extra_args, node_version, skip_permissions) are allowed.
func (c *ClaudeCode) Validate() error {
	if c.config.PythonVersion != "" {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"claude-code", "python_version", "aider")
	}
	if len(c.config.InstallCommands) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"claude-code", "install_commands", "custom")
	}
	if c.config.EntrypointCommand != "" {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"claude-code", "entrypoint_command", "custom")
	}
	if len(c.config.ConfigDirs) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"claude-code", "config_dirs", "custom")
	}
	if len(c.config.RequiredEnv) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"claude-code", "required_env", "custom")
	}
	if c.config.CustomHealthCheck != "" {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"claude-code", "health_check", "custom")
	}
	if len(c.config.CustomAliases) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"claude-code", "aliases", "custom")
	}
	if len(c.config.CustomShellRC) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"claude-code", "shell_rc", "custom")
	}
	return nil
}
