// harness_config.go defines the typed HarnessConfig struct for per-harness configuration.
package config

// HarnessConfig holds the [harness] section of a config file.
// Fields are shared across all harnesses or specific to a single harness type.
// Nil pointer fields (e.g. SkipPermissions) mean "not set", enabling merge to
// distinguish between "set to false" and "not configured".
type HarnessConfig struct {
	// Common (all harnesses)
	Version   string   `toml:"version"`
	ExtraArgs []string `toml:"extra_args"`

	// Harness-specific dangerous permission bypass for supported harnesses.
	SkipPermissions *bool  `toml:"skip_permissions"`
	NodeVersion     string `toml:"node_version"`

	// Aider specific
	PythonVersion string `toml:"python_version"`

	// Custom harness
	InstallCommands   []string          `toml:"install_commands"`
	EntrypointCommand string            `toml:"entrypoint_command"`
	ConfigDirs        []string          `toml:"config_dirs"`
	RequiredEnv       []string          `toml:"required_env"`
	CustomHealthCheck string            `toml:"health_check"`
	CustomAliases     map[string]string `toml:"aliases"`
	CustomShellRC     []string          `toml:"shell_rc"`
}
