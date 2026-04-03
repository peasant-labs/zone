// rules.go parses network allow/deny rules from configuration.
package network

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/peasant-labs/zone/internal/config"
)

// warnWriter is the destination for BuildRuleSet warnings.
// Overridden in tests to capture output.
var warnWriter io.Writer = os.Stderr

// RuleSet holds the resolved rule parameters for a single container firewall.
type RuleSet struct {
	Mode       string
	AllowedIPs map[string]string // ip -> hostname for ACCEPT rules
	DeniedIPs  map[string]string // ip -> hostname for DROP rules
	DenyGlobs  []CompiledPattern // deny glob patterns for refresh-time evaluation
	AllowGlobs []CompiledPattern // allow glob patterns for refresh-time evaluation
	Warnings   []string          // warnings about unresolvable globs
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
				// Glob patterns cannot be DNS-resolved to IP addresses for iptables rules.
				// Store for refresh-time evaluation and warn the user.
				rs.AllowGlobs = append(rs.AllowGlobs, cp)
				w := fmt.Sprintf("Warning: allow glob %q cannot be DNS-resolved to IP addresses for iptables rules. It will be used for pattern matching during rule refresh.", pattern)
				rs.Warnings = append(rs.Warnings, w)
				continue
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
				// Deny globs cannot be DNS-resolved directly.
				// In whitelist mode: they already work via MatchAny deny-before-allow evaluation above.
				// In blocklist mode: store for refresh-time evaluation and warn.
				rs.DenyGlobs = append(rs.DenyGlobs, cp)
				if mode == "blocklist" {
					w := fmt.Sprintf("Warning: deny glob %q cannot be DNS-resolved to IP addresses for direct iptables rules. It will be used for pattern matching during rule refresh.", pattern)
					rs.Warnings = append(rs.Warnings, w)
				}
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

	for _, w := range rs.Warnings {
		fmt.Fprintln(warnWriter, w)
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
	if len(a.DenyGlobs) != len(b.DenyGlobs) || len(a.AllowGlobs) != len(b.AllowGlobs) {
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
	for i, g := range a.DenyGlobs {
		if g.String() != b.DenyGlobs[i].String() {
			return false
		}
	}
	for i, g := range a.AllowGlobs {
		if g.String() != b.AllowGlobs[i].String() {
			return false
		}
	}
	return true
}
