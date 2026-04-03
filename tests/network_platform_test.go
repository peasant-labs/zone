package tests

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	cmdpkg "github.com/peasant-labs/zone/cmd"
	"github.com/peasant-labs/zone/internal/docker"
)

func TestPlatformDetection(t *testing.T) {
	mac := docker.Platform{OS: "darwin", SupportsIPTables: false}
	assert.Equal(t, "darwin", mac.OS)
	assert.False(t, mac.SupportsIPTables)

	rootless := docker.Platform{OS: runtime.GOOS, IsRootless: true, SupportsIPTables: false}
	assert.True(t, rootless.IsRootless)
	assert.False(t, rootless.SupportsIPTables)

	linux := docker.Platform{OS: "linux", IsRootless: false, SupportsIPTables: true}
	assert.Equal(t, "linux", linux.OS)
	assert.False(t, linux.IsRootless)
	assert.True(t, linux.SupportsIPTables)
}

func TestNetworkMacOSFallback(t *testing.T) {
	msg, code := cmdpkg.MapError(docker.ErrFirewallSetup)
	assert.Equal(t, 4, code)
	assert.Contains(t, msg, "firewall")

	msg, code = cmdpkg.MapError(docker.ErrSudoUnavailable)
	assert.Equal(t, 4, code)
	assert.Contains(t, msg, "sudo")

	msg, code = cmdpkg.MapError(docker.ErrIPTablesUnavailable)
	assert.Equal(t, 4, code)
	assert.Contains(t, msg, "iptables")

	_, code = cmdpkg.MapError(docker.ErrNetworkUnsupported)
	assert.Equal(t, 4, code)
}
