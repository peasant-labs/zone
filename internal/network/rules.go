// rules.go parses network allow/deny rules from configuration.
package network

import (
	"fmt"
	"net"
	"strings"

	"github.com/peasant-labs/zone/internal/config"
)

// RuleSet holds the resolved rule parameters for a single container firewall.
type RuleSet struct {
	Mode       string
	AllowedIPs map[string]string
	DeniedIPs  map[string]string
}

// BuildRuleSet generates a RuleSet from a merged NetworkConfig.
func BuildRuleSet(cfg *config.NetworkConfig, resolveFunc func(string) ([]string, error)) (RuleSet, error) {
	mode := "none"
	if cfg != nil {
		mode = normalizeMode(cfg.Mode)
	}
	if mode == "none" {
		return RuleSet{Mode: "none"}, nil
	}

	if resolveFunc == nil {
		resolveFunc = net.LookupHost
	}

	rs := RuleSet{
		Mode:       mode,
		AllowedIPs: make(map[string]string),
		DeniedIPs:  make(map[string]string),
	}

	denyPatterns, err := CompileAll(cfg.Deny)
	if err != nil {
		return rs, fmt.Errorf("compile deny patterns: %w", err)
	}

	if mode == "whitelist" {
		for _, pattern := range cfg.Allow {
			cp, err := Compile(pattern)
			if err != nil {
				return rs, fmt.Errorf("compile allow pattern %q: %w", pattern, err)
			}
			if cp.IsGlob() {
				return rs, fmt.Errorf("glob patterns in whitelist allow list are not supported in Phase 1: %q cannot be resolved to IP addresses for iptables rules. Use exact hostnames instead", pattern)
			}
			if MatchAny(cp.String(), denyPatterns) {
				continue
			}
			ips, err := resolveFunc(cp.String())
			if err != nil {
				continue
			}
			for _, ip := range ips {
				rs.AllowedIPs[ip] = cp.String()
			}
		}
	}

	if mode == "whitelist" || mode == "blocklist" {
		for _, pattern := range cfg.Deny {
			cp, err := Compile(pattern)
			if err != nil {
				return rs, fmt.Errorf("compile deny pattern %q: %w", pattern, err)
			}
			if cp.IsGlob() {
				continue
			}
			ips, err := resolveFunc(cp.String())
			if err != nil {
				continue
			}
			for _, ip := range ips {
				rs.DeniedIPs[ip] = cp.String()
			}
		}
	}

	return rs, nil
}

func normalizeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "whitelist", "blocklist":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return "none"
	}
}

// RulesEqual reports whether two rule sets produce the same effective IP sets.
func RulesEqual(a, b RuleSet) bool {
	if a.Mode != b.Mode {
		return false
	}
	if len(a.AllowedIPs) != len(b.AllowedIPs) || len(a.DeniedIPs) != len(b.DeniedIPs) {
		return false
	}
	for ip := range a.AllowedIPs {
		if _, ok := b.AllowedIPs[ip]; !ok {
			return false
		}
	}
	for ip := range a.DeniedIPs {
		if _, ok := b.DeniedIPs[ip]; !ok {
			return false
		}
	}
	return true
}
