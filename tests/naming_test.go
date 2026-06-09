// Tests for deterministic container name generation.
package tests

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/peasant-labs/zone/internal/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerNameDeterministic(t *testing.T) {
	name1 := docker.ContainerName("/home/user/my-project")
	name2 := docker.ContainerName("/home/user/my-project")
	assert.Equal(t, name1, name2, "Same path must produce same container name")
}

func TestContainerNameFormat(t *testing.T) {
	name := docker.ContainerName("/home/user/my-project")
	assert.True(t, strings.HasPrefix(name, "zone-my-project-"), "Name must start with zone-<repo>-")
	parts := strings.SplitN(name, "-", 3)
	// zone + repo-name + hash (at least 3 parts)
	require.GreaterOrEqual(t, len(parts), 3)
}

func TestContainerNameDifferentPaths(t *testing.T) {
	name1 := docker.ContainerName("/home/user/project-a")
	name2 := docker.ContainerName("/home/user/project-b")
	assert.NotEqual(t, name1, name2, "Different paths must produce different names")
}

func TestContainerNameSanitization(t *testing.T) {
	name := docker.ContainerName("/home/user/my project!@#")
	// Special chars replaced with -
	assert.NotContains(t, name, " ")
	assert.NotContains(t, name, "!")
	assert.NotContains(t, name, "@")
	assert.NotContains(t, name, "#")
}

func TestContainerNameHashLength(t *testing.T) {
	name := docker.ContainerName("/home/user/myrepo")
	// Extract hash: everything after last dash that's 16 chars of hex
	parts := strings.Split(name, "-")
	hash := parts[len(parts)-1]
	assert.Equal(t, 16, len(hash), "Hash portion must be 16 characters")
}

func TestNetworkNameSuffix(t *testing.T) {
	containerName := docker.ContainerName("/home/user/myrepo")
	networkName := docker.NetworkName("/home/user/myrepo")
	assert.Equal(t, containerName+"-net", networkName, "Network name must be container name + -net")
}

func TestContainerLabels(t *testing.T) {
	labels := docker.ContainerLabels("/home/user/myrepo", "claude-code", "hash123", "sha256:image123")
	assert.Equal(t, "true", labels["com.zone.managed"])
	assert.Equal(t, "/home/user/myrepo", labels["com.zone.repo-path"])
	assert.Equal(t, "claude-code", labels["com.zone.harness"])
	assert.Equal(t, "hash123", labels["com.zone.config-hash"])
	assert.Equal(t, "sha256:image123", labels["com.zone.image-id"])
	assert.Equal(t, 5, len(labels), "Must have exactly 5 labels")
}

func TestContainerNameUsesAbsPath(t *testing.T) {
	// When given a relative path, ContainerName calls filepath.Abs
	// So the name should match what we get from the absolute version
	abs, _ := filepath.Abs(".")
	nameRel := docker.ContainerName(".")
	nameAbs := docker.ContainerName(abs)
	assert.Equal(t, nameRel, nameAbs, "Relative and absolute path must produce same name")
}

func TestContainerSecurityFlags(t *testing.T) {
	flags := docker.ContainerSecurityFlags()
	assert.Equal(t, []string{"no-new-privileges"}, flags.SecurityOpt)
	assert.Equal(t, []string{"ALL"}, flags.CapDrop)
	assert.Contains(t, flags.CapAdd, "CHOWN")
	assert.Contains(t, flags.CapAdd, "DAC_OVERRIDE")
	assert.Contains(t, flags.CapAdd, "SETGID")
	assert.Contains(t, flags.CapAdd, "SETUID")
	assert.Contains(t, flags.CapAdd, "FOWNER")
	assert.Equal(t, 5, len(flags.CapAdd), "Must have exactly 5 added capabilities")
	assert.Equal(t, int64(512), flags.DefaultPidsLimit)
}
