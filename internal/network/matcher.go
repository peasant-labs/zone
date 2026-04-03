// matcher.go implements hostname glob matching for network rules.
package network

import (
	"fmt"
	"path/filepath"
	"strings"
)

// CompiledPattern is a precompiled hostname pattern for efficient matching.
type CompiledPattern struct {
	raw    string
	isGlob bool
}

// Compile validates and compiles a hostname pattern.
// Supports literal hostnames ("api.anthropic.com") and simple globs ("*.anthropic.com").
// Rejects patterns containing path separators, double-stars, brackets, or braces.
func Compile(pattern string) (CompiledPattern, error) {
	if pattern == "" {
		return CompiledPattern{}, fmt.Errorf("empty pattern")
	}
	if strings.Contains(pattern, "/") || strings.Contains(pattern, "**") ||
		strings.ContainsAny(pattern, "[]{}") {
		return CompiledPattern{}, fmt.Errorf("unsupported pattern %q: only literal hostnames and *.domain.com globs are supported", pattern)
	}

	return CompiledPattern{raw: pattern, isGlob: strings.Contains(pattern, "*")}, nil
}

// Match returns true if hostname matches this compiled pattern.
// Literal patterns use exact string comparison. Glob patterns use filepath.Match.
func (p CompiledPattern) Match(hostname string) bool {
	if !p.isGlob {
		return p.raw == hostname
	}
	if strings.Count(p.raw, ".") != strings.Count(hostname, ".") {
		return false
	}

	matched, _ := filepath.Match(p.raw, hostname)
	return matched
}

// String returns the original pattern string.
func (p CompiledPattern) String() string {
	return p.raw
}

// IsGlob returns true if this pattern contains wildcards.
func (p CompiledPattern) IsGlob() bool {
	return p.isGlob
}

// CompileAll compiles a list of hostname patterns, returning an error on the first invalid pattern.
func CompileAll(patterns []string) ([]CompiledPattern, error) {
	compiled := make([]CompiledPattern, 0, len(patterns))
	for _, pattern := range patterns {
		compiledPattern, err := Compile(pattern)
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, compiledPattern)
	}
	return compiled, nil
}

// MatchAny returns true if hostname matches any of the compiled patterns.
func MatchAny(hostname string, patterns []CompiledPattern) bool {
	for _, pattern := range patterns {
		if pattern.Match(hostname) {
			return true
		}
	}
	return false
}
