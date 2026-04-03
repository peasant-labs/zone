// custom.go implements the custom harness — a fully user-defined tool via HarnessConfig.
package harness

import (
	"fmt"

	"github.com/peasant-labs/zone/internal/config"
)

// Custom implements the Harness interface for user-defined tooling.
// All behavior is driven by HarnessConfig fields set in zone.toml [harness] section.
// entrypoint_command is required; all other fields are optional.
type Custom struct {
	BaseHarness
	config *config.HarnessConfig
}

func (c *Custom) Name() string                   { return "custom" }
func (c *Custom) InstallCommands() []string       { return c.config.InstallCommands }
func (c *Custom) EntrypointCommand() string       { return c.config.EntrypointCommand }
func (c *Custom) HealthCheck() string             { return c.config.CustomHealthCheck }
func (c *Custom) RequiredEnvVars() []string       { return c.config.RequiredEnv }
func (c *Custom) HomeConfigDir() string           { return "" }
func (c *Custom) ExtraConfigDirs() []string       { return c.config.ConfigDirs }
func (c *Custom) DefaultAptPackages() []string    { return nil }
func (c *Custom) DefaultNpmPackages() []string    { return nil }
func (c *Custom) DefaultPipPackages() []string    { return nil }
func (c *Custom) NeedsNode() bool                 { return false }
func (c *Custom) NeedsPython() bool               { return false }
func (c *Custom) ShellRC() []string               { return c.config.CustomShellRC }
func (c *Custom) Aliases() map[string]string      { return c.config.CustomAliases }

// Validate rejects claude-code-specific keys and requires entrypoint_command.
func (c *Custom) Validate() error {
	// Reject claude-code-specific keys (only when explicitly enabled)
	if c.config.SkipPermissions != nil && *c.config.SkipPermissions {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"custom", "skip_permissions", "claude-code")
	}
	// Required field check
	if c.config.EntrypointCommand == "" {
		return fmt.Errorf("custom harness requires %q in [harness] config", "entrypoint_command")
	}
	return nil
}
