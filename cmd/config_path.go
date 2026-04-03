package cmd

import (
	"path/filepath"

	"github.com/peasant-labs/zone/internal/config"
)

// loadMergedFromDir resolves zone.toml from a repo directory and loads merged config.
func loadMergedFromDir(dir string) (*config.MergedConfig, *config.AnnotatedConfig, error) {
	return config.LoadMerged(filepath.Join(dir, "zone.toml"))
}
