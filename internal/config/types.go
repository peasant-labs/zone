// Package config provides TOML configuration parsing, merging, and validation for zone.
package config

// Source identifies where a config value originated.
type Source string

const (
	// SourceDefault indicates the value came from built-in defaults.
	SourceDefault Source = "global (default)"
	// SourceGlobal indicates the value came from the global config file.
	SourceGlobal Source = "global"
	// SourceRepo indicates the value came from the per-repo zone.toml.
	SourceRepo Source = "repo: zone.toml"
)

// AnnotatedField wraps a config value with its source for display purposes.
type AnnotatedField[T any] struct {
	Value  T
	Source Source
}

// AnnotatedListItem tracks provenance of individual list elements.
type AnnotatedListItem struct {
	Value  string
	Source Source
}

// AnnotatedFieldJSON is used for JSON serialisation of annotated fields.
type AnnotatedFieldJSON struct {
	Value  any    `json:"value"`
	Source Source `json:"source"`
}

// ZoneConfig holds the [zone] section of a config file.
type ZoneConfig struct {
	Harness   string `toml:"harness"`
	BaseImage string `toml:"base_image"`
	Shell     string `toml:"shell"`
}

// AuthConfig holds the [auth] section of a config file.
type AuthConfig struct {
	MountHomeConfig *bool    `toml:"mount_home_config"`
	ForwardEnv      []string `toml:"forward_env"`
	ForwardSSHAgent *bool    `toml:"forward_ssh_agent"`
	EnvFile         string   `toml:"env_file"`
}

// WorkspaceConfig holds the [workspace] section of a per-repo config file.
type WorkspaceConfig struct {
	MountPath   string   `toml:"mount_path"`
	ExtraMounts []string `toml:"extra_mounts"`
	Ports       []string `toml:"ports"`
	PersistHome *bool    `toml:"persist_home"`
}

// PackagesConfig holds the [packages] section of a config file.
type PackagesConfig struct {
	Apt []string `toml:"apt"`
	Pip []string `toml:"pip"`
	Npm []string `toml:"npm"`
}

// ResourcesConfig holds the [resources] section of a config file.
type ResourcesConfig struct {
	Memory    string `toml:"memory"`
	Cpus      string `toml:"cpus"`
	PidsLimit int    `toml:"pids_limit"`
}

// NetworkConfig holds the [network] section of a config file.
// Per-repo uses allow/deny; global uses default_allow/default_deny.
type NetworkConfig struct {
	Mode         string   `toml:"mode"`
	Allow        []string `toml:"allow"`
	Deny         []string `toml:"deny"`
	DefaultAllow []string `toml:"default_allow"`
	DefaultDeny  []string `toml:"default_deny"`
	HTTPProxy    string   `toml:"http_proxy"`
	HTTPSProxy   string   `toml:"https_proxy"`
	NoProxy      string   `toml:"no_proxy"`
}

// HooksConfig holds the [hooks] section of a per-repo config file.
type HooksConfig struct {
	PreBuild []string `toml:"pre_build"`
	PostStop []string `toml:"post_stop"`
}

// RepoConfig represents a parsed per-repo zone.toml.
// The [harness] TOML table maps to the Harness field (HarnessConfig).
// The top-level `harness = "..."` sugar string is handled separately in LoadRepo
// and stored in HarnessName after normalisation.
type RepoConfig struct {
	Version    int             `toml:"version"`
	HarnessName string         `toml:"-"` // populated from sugar or [zone].harness after parse
	Zone       ZoneConfig      `toml:"zone"`
	Auth       AuthConfig      `toml:"auth"`
	Workspace  WorkspaceConfig `toml:"workspace"`
	Packages   PackagesConfig  `toml:"packages"`
	Resources  ResourcesConfig `toml:"resources"`
	Network    NetworkConfig   `toml:"network"`
	Hooks      HooksConfig     `toml:"hooks"`
	Harness    HarnessConfig   `toml:"harness"` // [harness] table
}

// GlobalConfig represents a parsed global ~/.config/zone/config.toml.
type GlobalConfig struct {
	Version   int             `toml:"version"`
	Zone      ZoneConfig      `toml:"zone"`
	Auth      AuthConfig      `toml:"auth"`
	Packages  PackagesConfig  `toml:"packages"`
	Resources ResourcesConfig `toml:"resources"`
	Network   NetworkConfig   `toml:"network"`
	Harness   HarnessConfig   `toml:"harness"`
}

// MergedConfig is the result of merging global defaults with per-repo overrides.
type MergedConfig struct {
	Version   int
	Zone      ZoneConfig
	Auth      AuthConfig
	Workspace WorkspaceConfig
	Packages  PackagesConfig
	Resources ResourcesConfig
	Network   NetworkConfig
	Hooks     HooksConfig
	Harness   HarnessConfig
}

// AnnotatedConfig wraps the merged config values with source tracking for
// the `zone config` display command.
type AnnotatedConfig struct {
	Version   AnnotatedField[int]
	Harness   AnnotatedField[string]
	BaseImage AnnotatedField[string]
	Shell     AnnotatedField[string]
	// Auth
	MountHomeConfig AnnotatedField[bool]
	ForwardEnv      []AnnotatedListItem
	ForwardSSHAgent AnnotatedField[bool]
	EnvFile         AnnotatedField[string]
	// Workspace
	MountPath   AnnotatedField[string]
	ExtraMounts []AnnotatedListItem
	Ports       []AnnotatedListItem
	PersistHome AnnotatedField[bool]
	// Packages
	AptPackages []AnnotatedListItem
	PipPackages []AnnotatedListItem
	NpmPackages []AnnotatedListItem
	// Resources
	Memory    AnnotatedField[string]
	Cpus      AnnotatedField[string]
	PidsLimit AnnotatedField[int]
	// Network
	NetworkMode AnnotatedField[string]
	Allow       []AnnotatedListItem
	Deny        []AnnotatedListItem
	HTTPProxy   AnnotatedField[string]
	HTTPSProxy  AnnotatedField[string]
	NoProxy     AnnotatedField[string]
	// Hooks
	PreBuild []AnnotatedListItem
	PostStop []AnnotatedListItem
	// Harness
	HarnessVersion    AnnotatedField[string]
	SkipPermissions   AnnotatedField[bool]
	NodeVersion       AnnotatedField[string]
	PythonVersion     AnnotatedField[string]
	ExtraArgs         []AnnotatedListItem
	InstallCommands   []AnnotatedListItem
	EntrypointCommand AnnotatedField[string]
	ConfigDirs        []AnnotatedListItem
	RequiredEnv       []AnnotatedListItem
	CustomHealthCheck AnnotatedField[string]
	CustomAliases     map[string]AnnotatedField[string]
	CustomShellRC     []AnnotatedListItem
}

// DefaultRepoConfig returns a RepoConfig populated with sane defaults.
func DefaultRepoConfig() *RepoConfig {
	pidsLimit := 512
	mountPath := "/workspace"
	return &RepoConfig{
		Version: 1,
		Workspace: WorkspaceConfig{
			MountPath: mountPath,
		},
		Resources: ResourcesConfig{
			PidsLimit: pidsLimit,
		},
	}
}

// DefaultGlobalConfig returns a GlobalConfig populated with spec-defined defaults
// (section 4.1: base_image="ubuntu:24.04", shell="bash",
// apt=["git","curl","wget"], pids_limit=512, network.mode="none").
func DefaultGlobalConfig() *GlobalConfig {
	pidsLimit := 512
	mountHomeConfig := true
	forwardSSHAgent := false
	return &GlobalConfig{
		Version: 1,
		Zone: ZoneConfig{
			BaseImage: "ubuntu:24.04",
			Shell:     "bash",
		},
		Auth: AuthConfig{
			MountHomeConfig: &mountHomeConfig,
			ForwardSSHAgent: &forwardSSHAgent,
		},
		Packages: PackagesConfig{
			Apt: []string{"git", "curl", "wget"},
		},
		Resources: ResourcesConfig{
			PidsLimit: pidsLimit,
		},
		Network: NetworkConfig{
			Mode: "none",
		},
	}
}
