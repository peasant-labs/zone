// ports_test.go tests port binding string parsing and validation.
package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParsePortBindings_SinglePort verifies "3000:3000" parses to nat.Port("3000/tcp") with HostPort "3000".
func TestParsePortBindings_SinglePort(t *testing.T) {
	pm, ps, err := parsePortBindings([]string{"3000:3000"})
	require.NoError(t, err)

	require.Len(t, pm, 1)
	require.Len(t, ps, 1)

	bindings, ok := pm["3000/tcp"]
	require.True(t, ok, "expected nat.Port 3000/tcp in PortMap")
	require.Len(t, bindings, 1)
	assert.Equal(t, "3000", bindings[0].HostPort)

	_, inSet := ps["3000/tcp"]
	assert.True(t, inSet, "expected nat.Port 3000/tcp in PortSet")
}

// TestParsePortBindings_DifferentPorts verifies "8080:80" parses to nat.Port("80/tcp") with HostPort "8080".
func TestParsePortBindings_DifferentPorts(t *testing.T) {
	pm, ps, err := parsePortBindings([]string{"8080:80"})
	require.NoError(t, err)

	require.Len(t, pm, 1)
	require.Len(t, ps, 1)

	bindings, ok := pm["80/tcp"]
	require.True(t, ok, "expected nat.Port 80/tcp in PortMap")
	require.Len(t, bindings, 1)
	assert.Equal(t, "8080", bindings[0].HostPort)

	_, inSet := ps["80/tcp"]
	assert.True(t, inSet, "expected nat.Port 80/tcp in PortSet")
}

// TestParsePortBindings_MultiplePorts verifies multiple valid ports return correct PortMap and PortSet with all entries.
func TestParsePortBindings_MultiplePorts(t *testing.T) {
	pm, ps, err := parsePortBindings([]string{"8080:80", "9090:9090", "443:443"})
	require.NoError(t, err)

	assert.Len(t, pm, 3)
	assert.Len(t, ps, 3)

	// Verify all entries present in PortMap
	_, ok80 := pm["80/tcp"]
	assert.True(t, ok80, "expected 80/tcp")
	_, ok9090 := pm["9090/tcp"]
	assert.True(t, ok9090, "expected 9090/tcp")
	_, ok443 := pm["443/tcp"]
	assert.True(t, ok443, "expected 443/tcp")

	// Verify all entries present in PortSet
	_, ps80 := ps["80/tcp"]
	assert.True(t, ps80)
	_, ps9090 := ps["9090/tcp"]
	assert.True(t, ps9090)
	_, ps443 := ps["443/tcp"]
	assert.True(t, ps443)
}

// TestParsePortBindings_Empty verifies empty ports list returns empty PortMap and PortSet with no error.
func TestParsePortBindings_Empty(t *testing.T) {
	pm, ps, err := parsePortBindings([]string{})
	require.NoError(t, err)
	assert.Len(t, pm, 0)
	assert.Len(t, ps, 0)
}

// TestParsePortBindings_InvalidFormat verifies "invalid" (no colon) returns error.
func TestParsePortBindings_InvalidFormat(t *testing.T) {
	_, _, err := parsePortBindings([]string{"invalid"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "expected hostPort:containerPort")
}

// TestParsePortBindings_PortZero verifies "0:3000" returns error (port 0 invalid).
func TestParsePortBindings_PortZero(t *testing.T) {
	_, _, err := parsePortBindings([]string{"0:3000"})
	require.Error(t, err)
}

// TestParsePortBindings_PortOutOfRange verifies "70000:3000" returns error (port > 65535).
func TestParsePortBindings_PortOutOfRange(t *testing.T) {
	_, _, err := parsePortBindings([]string{"70000:3000"})
	require.Error(t, err)
}

// TestParsePortBindings_NonNumericHostPort verifies "abc:3000" returns error (non-numeric host port).
func TestParsePortBindings_NonNumericHostPort(t *testing.T) {
	_, _, err := parsePortBindings([]string{"abc:3000"})
	require.Error(t, err)
}

// TestParsePortBindings_ConflictingHostPort verifies duplicate host port returns error containing "conflicting".
func TestParsePortBindings_ConflictingHostPort(t *testing.T) {
	_, _, err := parsePortBindings([]string{"3000:3000", "3000:8080"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "conflicting")
}

// TestParsePortBindings_PortMapPortSetSync verifies PortMap and PortSet contain the same nat.Port keys.
func TestParsePortBindings_PortMapPortSetSync(t *testing.T) {
	pm, ps, err := parsePortBindings([]string{"8080:80", "443:443"})
	require.NoError(t, err)

	// Keys in PortMap and PortSet must be identical
	assert.Len(t, pm, len(ps), "PortMap and PortSet should have same number of entries")
	for port := range pm {
		_, inSet := ps[port]
		assert.True(t, inSet, "port %s in PortMap but not in PortSet", port)
	}
	for port := range ps {
		_, inMap := pm[port]
		assert.True(t, inMap, "port %s in PortSet but not in PortMap", port)
	}
}

// TestValidatePort_Valid verifies valid port numbers pass validation.
func TestValidatePort_Valid(t *testing.T) {
	tests := []string{"1", "80", "443", "3000", "8080", "65535"}
	for _, p := range tests {
		t.Run(p, func(t *testing.T) {
			err := validatePort(p)
			assert.NoError(t, err)
		})
	}
}

// TestValidatePort_Invalid verifies invalid port values return errors.
func TestValidatePort_Invalid(t *testing.T) {
	tests := []struct {
		input   string
		wantMsg string
	}{
		{"0", "out of range"},
		{"65536", "out of range"},
		{"abc", "not a valid number"},
		{"-1", "out of range"},
		{"99999", "out of range"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			err := validatePort(tc.input)
			require.Error(t, err)
			assert.ErrorContains(t, err, tc.wantMsg)
		})
	}
}
