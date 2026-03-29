// aider.go implements the aider stub harness.
package harness

import (
	"fmt"

	"github.com/peasant-labs/zone/internal/config"
)

// Aider is a stub harness for the aider AI coding tool.
// Validate() always fails with a "not yet implemented" error.
// Cross-harness keys are rejected with specific errors before the stub error.
// Note: python_version is aider-owned and NOT rejected here.
type Aider struct {
	BaseHarness
	config *config.HarnessConfig
}

func (a *Aider) Name() string                 { return "aider" }
func (a *Aider) InstallCommands() []string     { return nil }
func (a *Aider) EntrypointCommand() string     { return "" }
func (a *Aider) RequiredEnvVars() []string     { return nil }
func (a *Aider) HomeConfigDir() string         { return "" }
func (a *Aider) DefaultAptPackages() []string  { return nil }
func (a *Aider) DefaultNpmPackages() []string  { return nil }
func (a *Aider) DefaultPipPackages() []string  { return nil }
func (a *Aider) NeedsNode() bool               { return false }
func (a *Aider) NeedsPython() bool             { return false }

// Validate checks for cross-harness keys first, then returns the stub error.
// Aider owns python_version, so it does NOT reject that field.
func (a *Aider) Validate() error {
	// Reject claude-code-specific keys
	if a.config.SkipPermissions != nil {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"aider", "skip_permissions", "claude-code")
	}
	// Reject custom-specific keys
	if len(a.config.InstallCommands) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"aider", "install_commands", "custom")
	}
	if a.config.EntrypointCommand != "" {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"aider", "entrypoint_command", "custom")
	}
	if len(a.config.ConfigDirs) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"aider", "config_dirs", "custom")
	}
	if len(a.config.RequiredEnv) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"aider", "required_env", "custom")
	}
	if a.config.CustomHealthCheck != "" {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"aider", "health_check", "custom")
	}
	if len(a.config.CustomAliases) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"aider", "aliases", "custom")
	}
	if len(a.config.CustomShellRC) > 0 {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"aider", "shell_rc", "custom")
	}
	return fmt.Errorf(
		"the %q harness is not yet fully implemented; use harness = \"custom\" "+
			"with install_commands and entrypoint_command to configure it manually",
		a.Name(),
	)
}
