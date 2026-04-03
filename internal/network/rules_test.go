package network

import (
	"fmt"
	"testing"

	"github.com/peasant-labs/zone/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockResolver(mapping map[string][]string) func(string) ([]string, error) {
	return func(host string) ([]string, error) {
		if ips, ok := mapping[host]; ok {
			return ips, nil
		}
		return nil, fmt.Errorf("unknown host: %s", host)
	}
}

func TestBuildRuleSetWhitelist(t *testing.T) {
	rules, err := BuildRuleSet(&config.NetworkConfig{
		Mode:  "whitelist",
		Allow: []string{"api.github.com"},
	}, mockResolver(map[string][]string{"api.github.com": {"1.2.3.4"}}))
	require.NoError(t, err)
	assert.Equal(t, "whitelist", rules.Mode)
	assert.Equal(t, "api.github.com", rules.AllowedIPs["1.2.3.4"])
	assert.Empty(t, rules.DeniedIPs)
}

func TestBuildRuleSetBlocklist(t *testing.T) {
	rules, err := BuildRuleSet(&config.NetworkConfig{
		Mode: "blocklist",
		Deny: []string{"evil.com"},
	}, mockResolver(map[string][]string{"evil.com": {"5.6.7.8"}}))
	require.NoError(t, err)
	assert.Equal(t, "blocklist", rules.Mode)
	assert.Equal(t, "evil.com", rules.DeniedIPs["5.6.7.8"])
	assert.Empty(t, rules.AllowedIPs)
}

func TestBuildRuleSetModeNone(t *testing.T) {
	called := false
	rules, err := BuildRuleSet(&config.NetworkConfig{Mode: "none"}, func(string) ([]string, error) {
		called = true
		return []string{"1.2.3.4"}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, "none", rules.Mode)
	assert.Empty(t, rules.AllowedIPs)
	assert.Empty(t, rules.DeniedIPs)
	assert.False(t, called)
}

func TestBuildRuleSetDenyFirst(t *testing.T) {
	rules, err := BuildRuleSet(&config.NetworkConfig{
		Mode:  "whitelist",
		Allow: []string{"good.example.com", "bad.example.com"},
		Deny:  []string{"bad.example.com"},
	}, mockResolver(map[string][]string{
		"good.example.com": {"10.0.0.1"},
		"bad.example.com":  {"10.0.0.2"},
	}))
	require.NoError(t, err)
	assert.Equal(t, "good.example.com", rules.AllowedIPs["10.0.0.1"])
	assert.NotContains(t, rules.AllowedIPs, "10.0.0.2")
	assert.Equal(t, "bad.example.com", rules.DeniedIPs["10.0.0.2"])
}

func TestBuildRuleSetEmptyModeDefaultsNone(t *testing.T) {
	rules, err := BuildRuleSet(&config.NetworkConfig{}, mockResolver(nil))
	require.NoError(t, err)
	assert.Equal(t, "none", rules.Mode)
}

func TestBuildRuleSetGlobInWhitelistAllowReturnsError(t *testing.T) {
	_, err := BuildRuleSet(&config.NetworkConfig{
		Mode:  "whitelist",
		Allow: []string{"*.anthropic.com"},
	}, mockResolver(nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "glob patterns in whitelist allow list are not supported")
}

func TestNormalizeMode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: "none"},
		{name: "whitelist", input: "whitelist", want: "whitelist"},
		{name: "uppercase whitelist", input: "WHITELIST", want: "whitelist"},
		{name: "blocklist", input: "Blocklist", want: "blocklist"},
		{name: "invalid", input: "invalid", want: "none"},
		{name: "trim spaces", input: "  whitelist  ", want: "whitelist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeMode(tt.input))
		})
	}
}

func TestRulesEqual(t *testing.T) {
	a := RuleSet{
		Mode:       "whitelist",
		AllowedIPs: map[string]string{"1.2.3.4": "api.github.com"},
		DeniedIPs:  map[string]string{"5.6.7.8": "evil.com"},
	}
	b := RuleSet{
		Mode:       "whitelist",
		AllowedIPs: map[string]string{"1.2.3.4": "different-host.example"},
		DeniedIPs:  map[string]string{"5.6.7.8": "different-deny.example"},
	}
	assert.True(t, RulesEqual(a, b))

	b.Mode = "blocklist"
	assert.False(t, RulesEqual(a, b))

	b = RuleSet{
		Mode:       "whitelist",
		AllowedIPs: map[string]string{"9.9.9.9": "api.github.com"},
		DeniedIPs:  map[string]string{"5.6.7.8": "evil.com"},
	}
	assert.False(t, RulesEqual(a, b))
}
