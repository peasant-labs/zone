// opencode.go implements the opencode harness.
package harness

import (
	"fmt"

	"github.com/peasant-labs/zone/internal/config"
)

const opencodeDangerouslySkipPermissionsFlag = "--dangerously-skip-permissions"

// OpenCode implements the Harness interface for the opencode AI tool.
type OpenCode struct {
	BaseHarness
	config *config.HarnessConfig
}

func (o *OpenCode) Name() string { return "opencode" }

// InstallCommands downloads and installs opencode, then symlinks to PATH.
func (o *OpenCode) InstallCommands() []string {
	return []string{
		"curl -fsSL https://opencode.ai/install | bash",
		"ln -sf /root/.opencode/bin/opencode /usr/local/bin/opencode",
	}
}

func (o *OpenCode) HealthCheck() string          { return "opencode --version" }
func (o *OpenCode) EntrypointCommand() string    { return "opencode" }
func (o *OpenCode) RequiredEnvVars() []string    { return nil }
func (o *OpenCode) HomeConfigDir() string        { return "~/.opencode" }
func (o *OpenCode) NeedsNode() bool              { return false }
func (o *OpenCode) NeedsPython() bool            { return false }
func (o *OpenCode) DefaultAptPackages() []string { return nil }
func (o *OpenCode) DefaultNpmPackages() []string { return nil }
func (o *OpenCode) DefaultPipPackages() []string { return nil }

func (o *OpenCode) WelcomeMessage() string {
	return "Zone workspace: opencode"
}

// RuntimeCommand uses `opencode` for interactive sessions and `--prompt` for
// prompt-driven noninteractive runs.
func (o *OpenCode) RuntimeCommand(prompt string, args []string) []string {
	cmd := []string{"opencode"}
	if o.config.SkipPermissions != nil && *o.config.SkipPermissions {
		cmd = append(cmd, opencodeDangerouslySkipPermissionsFlag)
	}
	cmd = append(cmd, args...)
	if prompt != "" {
		cmd = append(cmd, "--prompt", prompt)
	}
	return cmd
}

// Validate rejects HarnessConfig fields that belong to other harnesses.
func (o *OpenCode) Validate() error {
	if o.config.PythonVersion != "" {
		return fmt.Errorf("harness %q does not support key %q (that key is specific to %q)",
			"opencode", "python_version", "aider")
	}
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
	return nil
}
