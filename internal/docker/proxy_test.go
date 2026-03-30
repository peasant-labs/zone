// proxy_test.go tests proxy resolution from config and host environment variables.
package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/peasant-labs/zone/internal/config"
)

// makeNetworkConfig is a helper to build a NetworkConfig for tests.
func makeNetworkConfig(http, https, no string) *config.NetworkConfig {
	return &config.NetworkConfig{
		HTTPProxy:  http,
		HTTPSProxy: https,
		NoProxy:    no,
	}
}

// TestResolveProxy_ConfigTakesPrecedence verifies config values take precedence over host env.
func TestResolveProxy_ConfigTakesPrecedence(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://env-proxy:3128")
	t.Setenv("HTTPS_PROXY", "https://env-proxy:3129")
	t.Setenv("NO_PROXY", "env.local")

	cfg := makeNetworkConfig("http://cfg-proxy:8080", "https://cfg-proxy:8443", "cfg.local")

	httpP, httpsP, noP := resolveProxy(cfg)

	assert.Equal(t, "http://cfg-proxy:8080", httpP, "config http_proxy should win over env")
	assert.Equal(t, "https://cfg-proxy:8443", httpsP, "config https_proxy should win over env")
	assert.Equal(t, "cfg.local", noP, "config no_proxy should win over env")
}

// TestResolveProxy_AutoDetectFromEnv verifies that when config is empty, host env is used.
func TestResolveProxy_AutoDetectFromEnv(t *testing.T) {
	// Clear any pre-existing proxy env vars
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("http_proxy", "http://lowercase-proxy:3128")
	t.Setenv("HTTPS_PROXY", "https://upper-proxy:3129")
	t.Setenv("https_proxy", "")
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "skip.local")

	cfg := makeNetworkConfig("", "", "")

	httpP, httpsP, noP := resolveProxy(cfg)

	assert.Equal(t, "http://lowercase-proxy:3128", httpP, "should fall back to lowercase http_proxy")
	assert.Equal(t, "https://upper-proxy:3129", httpsP, "should fall back to uppercase HTTPS_PROXY")
	assert.Equal(t, "skip.local", noP, "should fall back to lowercase no_proxy")
}

// TestProxyBuildArgs_BothUpperAndLower verifies proxyBuildArgs returns both upper and lowercase keys.
func TestProxyBuildArgs_BothUpperAndLower(t *testing.T) {
	args := proxyBuildArgs("http://proxy:3128", "https://proxy:3129", "local")

	require.Contains(t, args, "HTTP_PROXY")
	require.Contains(t, args, "http_proxy")
	require.Contains(t, args, "HTTPS_PROXY")
	require.Contains(t, args, "https_proxy")
	require.Contains(t, args, "NO_PROXY")
	require.Contains(t, args, "no_proxy")

	assert.Equal(t, "http://proxy:3128", *args["HTTP_PROXY"])
	assert.Equal(t, "http://proxy:3128", *args["http_proxy"])
	assert.Equal(t, "https://proxy:3129", *args["HTTPS_PROXY"])
	assert.Equal(t, "https://proxy:3129", *args["https_proxy"])
	assert.Equal(t, "local", *args["NO_PROXY"])
	assert.Equal(t, "local", *args["no_proxy"])
}

// TestProxyBuildArgs_NoPointerAliasing verifies that pointer values are unique addresses.
func TestProxyBuildArgs_NoPointerAliasing(t *testing.T) {
	args := proxyBuildArgs("http://proxy:3128", "https://proxy:3129", "local")

	// Each key should have a unique pointer (not aliased)
	httpUpper := args["HTTP_PROXY"]
	httpLower := args["http_proxy"]
	httpsUpper := args["HTTPS_PROXY"]
	httpsLower := args["https_proxy"]
	noUpper := args["NO_PROXY"]
	noLower := args["no_proxy"]

	require.NotNil(t, httpUpper)
	require.NotNil(t, httpLower)

	// Pointers should be distinct addresses (not aliased)
	assert.NotSame(t, httpUpper, httpLower, "HTTP_PROXY and http_proxy should be distinct pointers")
	assert.NotSame(t, httpsUpper, httpsLower, "HTTPS_PROXY and https_proxy should be distinct pointers")
	assert.NotSame(t, noUpper, noLower, "NO_PROXY and no_proxy should be distinct pointers")
}

// TestProxyEnvVars_Format verifies proxyEnvVars returns KEY=value formatted strings.
func TestProxyEnvVars_Format(t *testing.T) {
	envs := proxyEnvVars("http://proxy:3128", "https://proxy:3129", "local")

	assert.Contains(t, envs, "HTTP_PROXY=http://proxy:3128")
	assert.Contains(t, envs, "http_proxy=http://proxy:3128")
	assert.Contains(t, envs, "HTTPS_PROXY=https://proxy:3129")
	assert.Contains(t, envs, "https_proxy=https://proxy:3129")
	assert.Contains(t, envs, "NO_PROXY=local")
	assert.Contains(t, envs, "no_proxy=local")
}

// TestProxyBuildArgs_AllEmpty verifies all-empty produces empty map.
func TestProxyBuildArgs_AllEmpty(t *testing.T) {
	args := proxyBuildArgs("", "", "")
	assert.Empty(t, args)
}

// TestProxyEnvVars_AllEmpty verifies all-empty produces empty slice.
func TestProxyEnvVars_AllEmpty(t *testing.T) {
	envs := proxyEnvVars("", "", "")
	assert.Empty(t, envs)
}

// TestProxyEnvVars_OnlyHTTP verifies only http set returns only http entries.
func TestProxyEnvVars_OnlyHTTP(t *testing.T) {
	envs := proxyEnvVars("http://proxy:3128", "", "")

	assert.Contains(t, envs, "HTTP_PROXY=http://proxy:3128")
	assert.Contains(t, envs, "http_proxy=http://proxy:3128")
	// https and no_proxy should be absent
	for _, e := range envs {
		assert.NotContains(t, e, "HTTPS_PROXY=")
		assert.NotContains(t, e, "https_proxy=")
		assert.NotContains(t, e, "NO_PROXY=")
		assert.NotContains(t, e, "no_proxy=")
	}
}

// TestFirstEnv verifies firstEnv returns the first non-empty value.
func TestFirstEnv(t *testing.T) {
	t.Setenv("FIRST_KEY", "")
	t.Setenv("SECOND_KEY", "found-it")
	t.Setenv("THIRD_KEY", "not-this-one")

	result := firstEnv("FIRST_KEY", "SECOND_KEY", "THIRD_KEY")
	assert.Equal(t, "found-it", result)
}

// TestFirstEnv_AllEmpty verifies firstEnv returns empty string when all keys are unset.
func TestFirstEnv_AllEmpty(t *testing.T) {
	t.Setenv("MISSING_KEY_A", "")
	t.Setenv("MISSING_KEY_B", "")

	result := firstEnv("MISSING_KEY_A", "MISSING_KEY_B")
	assert.Equal(t, "", result)
}
