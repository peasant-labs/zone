// config.go handles TOML parsing with strict decoding.
package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

// Sentinel errors for repo config loading.
var (
	// ErrNoConfig is returned when no zone.toml is found in the expected location.
	ErrNoConfig = errors.New("no zone.toml found")
	// ErrVersionMismatch is returned when the config version is not supported.
	ErrVersionMismatch = errors.New("unsupported config version")
)

// UnknownKeysError is returned when the TOML file contains keys not defined
// in the config structs. This catches typos and obsolete keys.
type UnknownKeysError struct {
	Keys []string
	File string
}

func (e *UnknownKeysError) Error() string {
	return fmt.Sprintf("unknown config keys in %s: %v", e.File, e.Keys)
}

// repoConfigSugar is used for a second-pass decode when the primary decode fails
// because the user wrote `harness = "claude-code"` (a string) at the top level
// instead of using the [harness] table. These two TOML constructs are mutually
// exclusive in a single file.
type repoConfigSugar struct {
	Version   int             `toml:"version"`
	Harness   string          `toml:"harness"` // top-level sugar: harness = "claude-code"
	Zone      ZoneConfig      `toml:"zone"`
	Auth      AuthConfig      `toml:"auth"`
	Workspace WorkspaceConfig `toml:"workspace"`
	Packages  PackagesConfig  `toml:"packages"`
	Resources ResourcesConfig `toml:"resources"`
	Network   NetworkConfig   `toml:"network"`
	Hooks     HooksConfig     `toml:"hooks"`
}

// LoadRepo parses a zone.toml file at path and returns the typed config.
//
// It handles both the full form ([harness] table) and the sugar form
// (top-level `harness = "..."` string). Version validation is applied after
// decode. Unknown TOML keys are detected via MetaData.Undecoded() and returned
// as an *UnknownKeysError alongside a partially-filled config.
func LoadRepo(path string) (*RepoConfig, error) {
	cfg := &RepoConfig{}
	md, err := toml.DecodeFile(path, cfg)
	if err != nil {
		// Check if this is a type conflict on the "harness" key.
		// When the user writes `harness = "claude-code"` at top level, BurntSushi/toml
		// will fail because the struct expects [harness] to be a table (HarnessConfig).
		if isHarnessTypeError(err) {
			sugar := &repoConfigSugar{}
			md2, err2 := toml.DecodeFile(path, sugar)
			if err2 != nil {
				return nil, fmt.Errorf("parse %s: %w", path, err2)
			}
			// Copy sugar fields into the primary config struct.
			cfg.Version = sugar.Version
			cfg.HarnessName = sugar.Harness
			cfg.Zone = sugar.Zone
			cfg.Auth = sugar.Auth
			cfg.Workspace = sugar.Workspace
			cfg.Packages = sugar.Packages
			cfg.Resources = sugar.Resources
			cfg.Network = sugar.Network
			cfg.Hooks = sugar.Hooks
			// Use md2 for undecoded key detection.
			md = md2
		} else {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
	}

	// Normalise harness name: resolve sugar vs [zone].harness.
	if err := normaliseHarnessName(cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	// Version validation per CFG-09.
	if err := validateVersion(cfg.Version, path); err != nil {
		return nil, err
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}

	// Strict unknown-key detection.
	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, len(undecoded))
		for i, k := range undecoded {
			keys[i] = k.String()
		}
		return cfg, &UnknownKeysError{Keys: keys, File: path}
	}

	return cfg, nil
}

// isHarnessTypeError returns true when the TOML decode error is caused by the
// "harness" key being a string in the file but a struct in the Go type.
func isHarnessTypeError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// BurntSushi/toml reports type conflicts in messages containing both the key
	// name and a type description.
	return (strings.Contains(msg, "harness") &&
		(strings.Contains(msg, "string") || strings.Contains(msg, "type")))
}

// normaliseHarnessName resolves the harness name from the sugar field or
// [zone].harness, ensuring consistency.
func normaliseHarnessName(cfg *RepoConfig) error {
	sugar := cfg.HarnessName
	zoneHarness := cfg.Zone.Harness
	switch {
	case sugar != "" && zoneHarness != "" && sugar != zoneHarness:
		return fmt.Errorf("harness specified both at top level (%q) and in [zone] (%q) — use one or the other", sugar, zoneHarness)
	case sugar == "" && zoneHarness != "":
		cfg.HarnessName = zoneHarness
	case sugar != "" && zoneHarness == "":
		// HarnessName already set; mirror into Zone.Harness for convenience.
		cfg.Zone.Harness = sugar
	}
	return nil
}

// validateVersion checks the version field value, defaulting 0 to 1 and
// rejecting unsupported version numbers per spec section 4.3.
// It mutates the cfg.Version field when it defaults to 1.
func validateVersion(version int, path string) error {
	switch {
	case version == 0:
		// Missing version silently defaults to 1 (mutation done at call site).
		return nil
	case version == 1:
		return nil
	default:
		return fmt.Errorf("%w: zone.toml version %d is not supported by this version of zone. Update zone or check https://github.com/jonathanung/zone", ErrVersionMismatch, version)
	}
}
