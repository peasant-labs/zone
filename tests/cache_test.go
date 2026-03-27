// Tests for cache directory management and atomic ID persistence.
package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCacheEnsureDir verifies that New + EnsureDir creates .zone/ and .zone/logs/ directories.
func TestCacheEnsureDir(t *testing.T) {
	base := t.TempDir()
	c := cache.New(base)
	err := c.EnsureDir()
	require.NoError(t, err, "EnsureDir should not return an error")

	// .zone/ must exist
	info, err := os.Stat(c.Dir())
	require.NoError(t, err, ".zone/ directory must exist")
	assert.True(t, info.IsDir(), ".zone/ must be a directory")

	// .zone/logs/ must exist
	logsInfo, err := os.Stat(filepath.Join(c.Dir(), "logs"))
	require.NoError(t, err, ".zone/logs/ directory must exist")
	assert.True(t, logsInfo.IsDir(), ".zone/logs/ must be a directory")
}

// TestCacheAtomicWrite verifies that SetImageID writes atomically and no .tmp- file remains.
func TestCacheAtomicWrite(t *testing.T) {
	base := t.TempDir()
	c := cache.New(base)
	require.NoError(t, c.EnsureDir())

	err := c.SetImageID("sha256:abc123")
	require.NoError(t, err, "SetImageID should not error")

	id, err := c.ImageID()
	require.NoError(t, err, "ImageID should not error")
	assert.Equal(t, "sha256:abc123", id, "ImageID should return the written value")

	// No leftover .tmp- file should exist
	_, statErr := os.Stat(filepath.Join(c.Dir(), ".tmp-image_id"))
	assert.True(t, os.IsNotExist(statErr), ".tmp-image_id must not exist after write")
}

// TestCacheReadWrite verifies round-trip correctness for all four ID types.
func TestCacheReadWrite(t *testing.T) {
	base := t.TempDir()
	c := cache.New(base)
	require.NoError(t, c.EnsureDir())

	// ConfigHash round-trip
	require.NoError(t, c.SetConfigHash("deadbeef"))
	h, err := c.ConfigHash()
	require.NoError(t, err)
	assert.Equal(t, "deadbeef", h, "ConfigHash should return the written value")

	// ContainerID round-trip
	require.NoError(t, c.SetContainerID("cid123"))
	cid, err := c.ContainerID()
	require.NoError(t, err)
	assert.Equal(t, "cid123", cid, "ContainerID should return the written value")

	// NetworkID round-trip
	require.NoError(t, c.SetNetworkID("nid456"))
	nid, err := c.NetworkID()
	require.NoError(t, err)
	assert.Equal(t, "nid456", nid, "NetworkID should return the written value")
}

// TestCacheReadMissing verifies that reading a missing key returns ("", nil) — not an error.
func TestCacheReadMissing(t *testing.T) {
	base := t.TempDir()
	c := cache.New(base)
	require.NoError(t, c.EnsureDir())

	id, err := c.ImageID()
	require.NoError(t, err, "ImageID on fresh cache must not error")
	assert.Equal(t, "", id, "ImageID on fresh cache must return empty string")
}
