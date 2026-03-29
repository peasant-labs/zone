// gemini_cli.go implements the gemini-cli stub harness.
package harness

import (
	"fmt"

	"github.com/peasant-labs/zone/internal/config"
)

// GeminiCLI is a stub harness for the Google Gemini CLI AI tool.
// Validate() always fails with a "not yet implemented" error.
// Cross-harness keys are rejected with specific errors before the stub error.
type GeminiCLI struct {
	BaseHarness
	config *config.HarnessConfig
}

func (g *GeminiCLI) Name() string                 { return "gemini-cli" }
func (g *GeminiCLI) InstallCommands() []string     { return nil }
func (g *GeminiCLI) EntrypointCommand() string     { return "" }
func (g *GeminiCLI) RequiredEnvVars() []string     { return nil }
func (g *GeminiCLI) HomeConfigDir() string         { return "" }
func (g *GeminiCLI) DefaultAptPackages() []string  { return nil }
func (g *GeminiCLI) DefaultNpmPackages() []string  { return nil }
func (g *GeminiCLI) DefaultPipPackages() []string  { return nil }
func (g *GeminiCLI) NeedsNode() bool               { return false }
func (g *GeminiCLI) NeedsPython() bool             { return false }

// Validate checks for cross-harness keys first, then returns the stub error.
func (g *GeminiCLI) Validate() error {
	// Reject claude-code-specific keys
	if g.config.SkipPermissions != nil {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"gemini-cli", "skip_permissions", "claude-code")
	}
	// Reject aider-specific keys
	if g.config.PythonVersion != "" {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"gemini-cli", "python_version", "aider")
	}
	// Reject custom-specific keys
	if len(g.config.InstallCommands) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"gemini-cli", "install_commands", "custom")
	}
	if g.config.EntrypointCommand != "" {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"gemini-cli", "entrypoint_command", "custom")
	}
	if len(g.config.ConfigDirs) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"gemini-cli", "config_dirs", "custom")
	}
	if len(g.config.RequiredEnv) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"gemini-cli", "required_env", "custom")
	}
	if g.config.CustomHealthCheck != "" {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"gemini-cli", "health_check", "custom")
	}
	if len(g.config.CustomAliases) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"gemini-cli", "aliases", "custom")
	}
	if len(g.config.CustomShellRC) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"gemini-cli", "shell_rc", "custom")
	}
	return fmt.Errorf(
		"the %q harness is not yet fully implemented; use harness = \"custom\" "+
			"with install_commands and entrypoint_command to configure it manually",
		g.Name(),
	)
}
