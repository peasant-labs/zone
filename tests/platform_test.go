package tests

import (
	"runtime"
	"testing"

	"github.com/peasant-labs/zone/internal/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostUIDReturnsNonNegative(t *testing.T) {
	uid, err := docker.HostUID()
	require.NoError(t, err, "HostUID must not error in test environment")
	assert.GreaterOrEqual(t, uid, 0, "UID must be non-negative")
}

func TestMacOSUsernameOnCurrentPlatform(t *testing.T) {
	username := docker.MacOSUsername()
	if runtime.GOOS == "darwin" {
		assert.NotEmpty(t, username, "On macOS, MacOSUsername should return a value")
		// Verify sanitization: only [a-zA-Z0-9_.-] characters
		for _, c := range username {
			valid := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') || c == '_' || c == '.' || c == '-'
			assert.True(t, valid, "MacOSUsername must contain only [a-zA-Z0-9_.-], got char: %c", c)
		}
	} else {
		assert.Empty(t, username, "On non-darwin, MacOSUsername must return empty string")
	}
}

func TestDetectGitIdentityReturnsConsistently(t *testing.T) {
	// This test verifies the function runs without panic.
	// The actual result depends on the test environment's git config.
	name, email, forward := docker.DetectGitIdentity()
	if forward {
		assert.NotEmpty(t, name, "When forward=true, name must not be empty")
		assert.NotEmpty(t, email, "When forward=true, email must not be empty")
	} else {
		// When forward=false, name and email are empty strings
		assert.Empty(t, name)
		assert.Empty(t, email)
	}
}
