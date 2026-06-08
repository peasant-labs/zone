// codex_cli.go implements the codex-cli harness.
package harness

import (
	"fmt"

	"github.com/peasant-labs/zone/internal/config"
)

const codexDangerouslyBypassFlag = "--dangerously-bypass-approvals-and-sandbox"

// CodexCLI implements the Harness interface for the OpenAI Codex CLI AI tool.
type CodexCLI struct {
	BaseHarness
	config *config.HarnessConfig
}

func (c *CodexCLI) Name() string { return "codex-cli" }

func (c *CodexCLI) InstallCommands() []string {
	pkg := "@openai/codex"
	if c.config.Version != "" {
		pkg += "@" + c.config.Version
	}
	return []string{"npm install -g " + pkg}
}

func (c *CodexCLI) HealthCheck() string       { return "codex --version" }
func (c *CodexCLI) EntrypointCommand() string { return "codex" }
func (c *CodexCLI) RequiredEnvVars() []string { return nil }
func (c *CodexCLI) HomeConfigDir() string     { return "~/.codex" }
func (c *CodexCLI) DefaultAptPackages() []string {
	return nil
}
func (c *CodexCLI) DefaultNpmPackages() []string { return nil }
func (c *CodexCLI) DefaultPipPackages() []string { return nil }
func (c *CodexCLI) NeedsNode() bool              { return true }
func (c *CodexCLI) NeedsPython() bool            { return false }

// RuntimeCommand uses `codex` for interactive sessions and `codex exec` for
// prompt-driven noninteractive runs.
func (c *CodexCLI) RuntimeCommand(prompt string, args []string) []string {
	cmd := []string{"codex"}
	if prompt != "" {
		cmd = append(cmd, "exec")
	}
	if c.config.SkipPermissions != nil && *c.config.SkipPermissions {
		cmd = append(cmd, codexDangerouslyBypassFlag)
	}
	cmd = append(cmd, args...)
	if prompt != "" {
		cmd = append(cmd, prompt)
	}
	return cmd
}

// Validate rejects HarnessConfig fields that belong to other harnesses.
// Common fields (version, extra_args, node_version, skip_permissions) are allowed.
func (c *CodexCLI) Validate() error {
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
	return nil
}
