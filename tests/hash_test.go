// Tests for cache hash computation.
package tests

import (
	"testing"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHashStability verifies that ComputeHash returns the same result for identical inputs.
func TestHashStability(t *testing.T) {
	cfg := config.MergedConfig{
		Version: 1,
		Zone: config.ZoneConfig{
			Harness:   "claude-code",
			BaseImage: "ubuntu:24.04",
			Shell:     "bash",
		},
	}

	h1, err := cache.ComputeHash(&cfg, "1.0.0")
	require.NoError(t, err, "first ComputeHash must not error")

	h2, err := cache.ComputeHash(&cfg, "1.0.0")
	require.NoError(t, err, "second ComputeHash must not error")

	assert.Equal(t, h1, h2, "ComputeHash must return identical hashes for identical inputs")
}

// TestHashChangesOnConfigChange verifies that a config change produces a different hash.
func TestHashChangesOnConfigChange(t *testing.T) {
	cfg1 := config.MergedConfig{
		Version: 1,
		Zone: config.ZoneConfig{
			Harness:   "claude-code",
			BaseImage: "ubuntu:24.04",
			Shell:     "bash",
		},
	}
	cfg2 := config.MergedConfig{
		Version: 1,
		Zone: config.ZoneConfig{
			Harness:   "claude-code",
			BaseImage: "ubuntu:22.04",
			Shell:     "bash",
		},
	}

	h1, err := cache.ComputeHash(&cfg1, "1.0.0")
	require.NoError(t, err)

	h2, err := cache.ComputeHash(&cfg2, "1.0.0")
	require.NoError(t, err)

	assert.NotEqual(t, h1, h2, "Different configs must produce different hashes")
}

// TestHashChangesOnVersion verifies that a version change produces a different hash.
func TestHashChangesOnVersion(t *testing.T) {
	cfg := config.MergedConfig{
		Version: 1,
		Zone: config.ZoneConfig{
			Harness:   "claude-code",
			BaseImage: "ubuntu:24.04",
			Shell:     "bash",
		},
	}

	h1, err := cache.ComputeHash(&cfg, "1.0.0")
	require.NoError(t, err)

	h2, err := cache.ComputeHash(&cfg, "1.0.1")
	require.NoError(t, err)

	assert.NotEqual(t, h1, h2, "Different versions must produce different hashes")
}

// TestHashNotEmpty verifies that ComputeHash returns a 64-character hex SHA256 string.
func TestHashNotEmpty(t *testing.T) {
	cfg := config.MergedConfig{
		Version: 1,
		Zone: config.ZoneConfig{
			BaseImage: "ubuntu:24.04",
		},
	}

	h, err := cache.ComputeHash(&cfg, "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, 64, len(h), "ComputeHash must return a 64-character hex SHA256 string")
}
