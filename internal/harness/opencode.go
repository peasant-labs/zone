// opencode.go implements the opencode stub harness.
package harness

import (
	"fmt"

	"github.com/peasant-labs/zone/internal/config"
)

// OpenCode is a stub harness for the opencode AI tool.
// Validate() always fails with a "not yet implemented" error.
// Cross-harness keys are rejected with specific errors before the stub error.
type OpenCode struct {
	BaseHarness
	config *config.HarnessConfig
}

func (o *OpenCode) Name() string                 { return "opencode" }
func (o *OpenCode) InstallCommands() []string     { return nil }
func (o *OpenCode) EntrypointCommand() string     { return "" }
func (o *OpenCode) RequiredEnvVars() []string     { return nil }
func (o *OpenCode) HomeConfigDir() string         { return "" }
func (o *OpenCode) DefaultAptPackages() []string  { return nil }
func (o *OpenCode) DefaultNpmPackages() []string  { return nil }
func (o *OpenCode) DefaultPipPackages() []string  { return nil }
func (o *OpenCode) NeedsNode() bool               { return false }
func (o *OpenCode) NeedsPython() bool             { return false }

// Validate checks for cross-harness keys first, then returns the stub error.
func (o *OpenCode) Validate() error {
	// Reject claude-code-specific keys
	if o.config.SkipPermissions != nil {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"opencode", "skip_permissions", "claude-code")
	}
	// Reject aider-specific keys
	if o.config.PythonVersion != "" {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"opencode", "python_version", "aider")
	}
	// Reject custom-specific keys
	if len(o.config.InstallCommands) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"opencode", "install_commands", "custom")
	}
	if o.config.EntrypointCommand != "" {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"opencode", "entrypoint_command", "custom")
	}
	if len(o.config.ConfigDirs) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"opencode", "config_dirs", "custom")
	}
	if len(o.config.RequiredEnv) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"opencode", "required_env", "custom")
	}
	if o.config.CustomHealthCheck != "" {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"opencode", "health_check", "custom")
	}
	if len(o.config.CustomAliases) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"opencode", "aliases", "custom")
	}
	if len(o.config.CustomShellRC) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"opencode", "shell_rc", "custom")
	}
	return fmt.Errorf(
		"the %q harness is not yet fully implemented; use harness = \"custom\" "+
			"with install_commands and entrypoint_command to configure it manually",
		o.Name(),
	)
}
