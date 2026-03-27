// global.go reads and writes global config from ~/.config/zone/config.toml.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// GlobalConfigPath returns the path to the global config file, following the
// XDG Base Directory specification. It prefers $XDG_CONFIG_HOME/zone/config.toml
// and falls back to ~/.config/zone/config.toml.
//
// NOTE: os.UserConfigDir() is intentionally NOT used here because on macOS it
// returns ~/Library/Application Support, which violates XDG expectations for a
// CLI tool. Zone always uses the XDG path directly.
func GlobalConfigPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "zone", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("global config path: %w", err)
	}
	return filepath.Join(home, ".config", "zone", "config.toml"), nil
}

// LoadGlobal reads the global config file and returns the parsed config.
// If the file does not exist, built-in defaults are returned without error.
// This allows zone to function correctly on first run with no global config.
func LoadGlobal() (*GlobalConfig, error) {
	path, err := GlobalConfigPath()
	if err != nil {
		// If we cannot determine the path (e.g. no home dir), use defaults.
		return DefaultGlobalConfig(), nil
	}

	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		// Missing global config file is not an error — return defaults.
		return DefaultGlobalConfig(), nil
	}

	cfg := &GlobalConfig{}
	md, err := toml.DecodeFile(path, cfg)
	if err != nil {
		return nil, fmt.Errorf("parse global config %s: %w", path, err)
	}

	// Strict unknown-key detection (same as per-repo config).
	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, len(undecoded))
		for i, k := range undecoded {
			keys[i] = k.String()
		}
		return cfg, &UnknownKeysError{Keys: keys, File: path}
	}

	// Default version when missing.
	if cfg.Version == 0 {
		cfg.Version = 1
	}

	return cfg, nil
}
