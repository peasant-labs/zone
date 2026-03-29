// codex_cli.go implements the codex-cli stub harness.
package harness

import (
	"fmt"

	"github.com/peasant-labs/zone/internal/config"
)

// CodexCLI is a stub harness for the OpenAI Codex CLI AI tool.
// Validate() always fails with a "not yet implemented" error.
// Cross-harness keys are rejected with specific errors before the stub error.
type CodexCLI struct {
	BaseHarness
	config *config.HarnessConfig
}

func (c *CodexCLI) Name() string                 { return "codex-cli" }
func (c *CodexCLI) InstallCommands() []string     { return nil }
func (c *CodexCLI) EntrypointCommand() string     { return "" }
func (c *CodexCLI) RequiredEnvVars() []string     { return nil }
func (c *CodexCLI) HomeConfigDir() string         { return "" }
func (c *CodexCLI) DefaultAptPackages() []string  { return nil }
func (c *CodexCLI) DefaultNpmPackages() []string  { return nil }
func (c *CodexCLI) DefaultPipPackages() []string  { return nil }
func (c *CodexCLI) NeedsNode() bool               { return false }
func (c *CodexCLI) NeedsPython() bool             { return false }

// Validate checks for cross-harness keys first, then returns the stub error.
func (c *CodexCLI) Validate() error {
	// Reject claude-code-specific keys
	if c.config.SkipPermissions != nil {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"codex-cli", "skip_permissions", "claude-code")
	}
	// Reject aider-specific keys
	if c.config.PythonVersion != "" {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"codex-cli", "python_version", "aider")
	}
	// Reject custom-specific keys
	if len(c.config.InstallCommands) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"codex-cli", "install_commands", "custom")
	}
	if c.config.EntrypointCommand != "" {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"codex-cli", "entrypoint_command", "custom")
	}
	if len(c.config.ConfigDirs) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"codex-cli", "config_dirs", "custom")
	}
	if len(c.config.RequiredEnv) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"codex-cli", "required_env", "custom")
	}
	if c.config.CustomHealthCheck != "" {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"codex-cli", "health_check", "custom")
	}
	if len(c.config.CustomAliases) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"codex-cli", "aliases", "custom")
	}
	if len(c.config.CustomShellRC) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"codex-cli", "shell_rc", "custom")
	}
	return fmt.Errorf(
		"the %q harness is not yet fully implemented; use harness = \"custom\" "+
			"with install_commands and entrypoint_command to configure it manually",
		c.Name(),
	)
}
