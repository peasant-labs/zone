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

