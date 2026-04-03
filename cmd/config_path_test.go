package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadMergedFromDirUsesZoneTomlFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "zone.toml"), []byte("version = 1\nharness = \"claude-code\"\n"), 0644))

	merged, _, err := loadMergedFromDir(dir)
	require.NoError(t, err)
	require.Equal(t, "claude-code", merged.Zone.Harness)
}
