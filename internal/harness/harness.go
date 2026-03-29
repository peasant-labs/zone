// Package harness defines the harness interface, registry, and base implementation.
package harness

import (
	"fmt"
	"sort"

	"github.com/peasant-labs/zone/internal/config"
)

// Harness defines the full plugin contract for an AI coding agent runtime.
// Each harness corresponds to one AI tool (claude-code, aider, opencode, etc.)
// and provides all the configuration data needed to build and run its Docker container.
//
// The 19 required methods cover identity, installation, runtime, dependencies,
// shell configuration, and lifecycle hooks. NodeVersion and PythonVersion are NOT
// interface methods — those come from MergedConfig.Harness (see RESEARCH.md anti-patterns).
type Harness interface {
	// Identity
	Name() string
	Version() string

	// Installation
	InstallCommands() []string
	PostInstallCommands() []string

	// Runtime
	HealthCheck() string
	EntrypointCommand() string
	PromptFlag() string
	RequiredEnvVars() []string
	HomeConfigDir() string
	ExtraConfigDirs() []string

	// Dependencies
	DefaultAptPackages() []string
	DefaultNpmPackages() []string
	DefaultPipPackages() []string
	NeedsNode() bool
	NeedsPython() bool

	// Shell
	ShellRC() []string
	Aliases() map[string]string
	WelcomeMessage() string

	// Lifecycle
	Validate() error
}

// BaseHarness is an empty struct that provides no-op default implementations for
// the 9 optional Harness methods. Concrete harnesses embed BaseHarness and override
// only the methods they need to customise.
type BaseHarness struct{}

func (b BaseHarness) Version() string              { return "" }
func (b BaseHarness) PostInstallCommands() []string { return nil }
func (b BaseHarness) HealthCheck() string           { return "" }
func (b BaseHarness) PromptFlag() string            { return "" }
func (b BaseHarness) ExtraConfigDirs() []string     { return nil }
func (b BaseHarness) ShellRC() []string             { return nil }
func (b BaseHarness) Aliases() map[string]string    { return nil }
func (b BaseHarness) WelcomeMessage() string        { return "" }
func (b BaseHarness) Validate() error               { return nil }

// registry maps harness names to factory functions.
// All 6 registered names are available via Get().
var registry = map[string]func(*config.HarnessConfig) Harness{
	"claude-code": func(c *config.HarnessConfig) Harness { return &ClaudeCode{config: c} },
	"opencode":    func(c *config.HarnessConfig) Harness { return &OpenCode{config: c} },
	"gemini-cli":  func(c *config.HarnessConfig) Harness { return &GeminiCLI{config: c} },
	"aider":       func(c *config.HarnessConfig) Harness { return &Aider{config: c} },
	"codex-cli":   func(c *config.HarnessConfig) Harness { return &CodexCLI{config: c} },
	"custom":      func(c *config.HarnessConfig) Harness { return &Custom{config: c} },
}

// Get constructs and validates a harness by name.
// Returns (nil, error) if name is unknown or if Validate() fails.
func Get(name string, cfg *config.HarnessConfig) (Harness, error) {
	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown harness %q, available: %v", name, availableNames())
	}
	h := factory(cfg)
	if err := h.Validate(); err != nil {
		return nil, fmt.Errorf("harness %q config: %w", name, err)
	}
	return h, nil
}

// availableNames returns sorted registry keys for use in error messages.
func availableNames() []string {
	names := make([]string, 0, len(registry))
	for k := range registry {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// ---------------------------------------------------------------------------
// Placeholder types -- replaced by Plan 02
// These minimal implementations satisfy the Harness interface so Plan 01
// compiles independently. Plan 02 will replace each with a full implementation.
// ---------------------------------------------------------------------------

// OpenCode is a placeholder harness. TODO(plan-02): replace with full implementation.
type OpenCode struct {
	BaseHarness
	config *config.HarnessConfig
}

func (o *OpenCode) Name() string              { return "opencode" }
func (o *OpenCode) InstallCommands() []string  { return nil }
func (o *OpenCode) EntrypointCommand() string  { return "" }
func (o *OpenCode) RequiredEnvVars() []string  { return nil }
func (o *OpenCode) HomeConfigDir() string      { return "" }
func (o *OpenCode) DefaultAptPackages() []string { return nil }
func (o *OpenCode) DefaultNpmPackages() []string { return nil }
func (o *OpenCode) DefaultPipPackages() []string { return nil }
func (o *OpenCode) NeedsNode() bool            { return false }
func (o *OpenCode) NeedsPython() bool          { return false }

func (o *OpenCode) Validate() error {
	return fmt.Errorf(
		"the %q harness is not yet fully implemented; use harness = \"custom\" with install_commands and entrypoint_command to configure it manually",
		o.Name(),
	)
}

// GeminiCLI is a placeholder harness. TODO(plan-02): replace with full implementation.
type GeminiCLI struct {
	BaseHarness
	config *config.HarnessConfig
}

func (g *GeminiCLI) Name() string              { return "gemini-cli" }
func (g *GeminiCLI) InstallCommands() []string  { return nil }
func (g *GeminiCLI) EntrypointCommand() string  { return "" }
func (g *GeminiCLI) RequiredEnvVars() []string  { return nil }
func (g *GeminiCLI) HomeConfigDir() string      { return "" }
func (g *GeminiCLI) DefaultAptPackages() []string { return nil }
func (g *GeminiCLI) DefaultNpmPackages() []string { return nil }
func (g *GeminiCLI) DefaultPipPackages() []string { return nil }
func (g *GeminiCLI) NeedsNode() bool            { return false }
func (g *GeminiCLI) NeedsPython() bool          { return false }

func (g *GeminiCLI) Validate() error {
	return fmt.Errorf(
		"the %q harness is not yet fully implemented; use harness = \"custom\" with install_commands and entrypoint_command to configure it manually",
		g.Name(),
	)
}

// Aider is a placeholder harness. TODO(plan-02): replace with full implementation.
type Aider struct {
	BaseHarness
	config *config.HarnessConfig
}

func (a *Aider) Name() string              { return "aider" }
func (a *Aider) InstallCommands() []string  { return nil }
func (a *Aider) EntrypointCommand() string  { return "" }
func (a *Aider) RequiredEnvVars() []string  { return nil }
func (a *Aider) HomeConfigDir() string      { return "" }
func (a *Aider) DefaultAptPackages() []string { return nil }
func (a *Aider) DefaultNpmPackages() []string { return nil }
func (a *Aider) DefaultPipPackages() []string { return nil }
func (a *Aider) NeedsNode() bool            { return false }
func (a *Aider) NeedsPython() bool          { return false }

func (a *Aider) Validate() error {
	return fmt.Errorf(
		"the %q harness is not yet fully implemented; use harness = \"custom\" with install_commands and entrypoint_command to configure it manually",
		a.Name(),
	)
}

// CodexCLI is a placeholder harness. TODO(plan-02): replace with full implementation.
type CodexCLI struct {
	BaseHarness
	config *config.HarnessConfig
}

func (c *CodexCLI) Name() string              { return "codex-cli" }
func (c *CodexCLI) InstallCommands() []string  { return nil }
func (c *CodexCLI) EntrypointCommand() string  { return "" }
func (c *CodexCLI) RequiredEnvVars() []string  { return nil }
func (c *CodexCLI) HomeConfigDir() string      { return "" }
func (c *CodexCLI) DefaultAptPackages() []string { return nil }
func (c *CodexCLI) DefaultNpmPackages() []string { return nil }
func (c *CodexCLI) DefaultPipPackages() []string { return nil }
func (c *CodexCLI) NeedsNode() bool            { return false }
func (c *CodexCLI) NeedsPython() bool          { return false }

func (c *CodexCLI) Validate() error {
	return fmt.Errorf(
		"the %q harness is not yet fully implemented; use harness = \"custom\" with install_commands and entrypoint_command to configure it manually",
		c.Name(),
	)
}

// Custom is a placeholder harness. TODO(plan-02): replace with full implementation.
type Custom struct {
	BaseHarness
	config *config.HarnessConfig
}

func (c *Custom) Name() string              { return "custom" }
func (c *Custom) InstallCommands() []string  { return c.config.InstallCommands }
func (c *Custom) EntrypointCommand() string  { return c.config.EntrypointCommand }
func (c *Custom) RequiredEnvVars() []string  { return c.config.RequiredEnv }
func (c *Custom) HomeConfigDir() string      { return "" }
func (c *Custom) DefaultAptPackages() []string { return nil }
func (c *Custom) DefaultNpmPackages() []string { return nil }
func (c *Custom) DefaultPipPackages() []string { return nil }
func (c *Custom) NeedsNode() bool            { return false }
func (c *Custom) NeedsPython() bool          { return false }

func (c *Custom) Validate() error {
	if c.config.EntrypointCommand == "" {
		return fmt.Errorf("custom harness requires %q in [harness] config", "entrypoint_command")
	}
	return nil
}
