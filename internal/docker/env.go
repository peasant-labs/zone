// env.go encapsulates all environment variable logic for container injection:
// glob-based collection from the host, .env file parsing, and pre-launch
// required env var validation.
//
// This file has no Docker SDK dependency, making it fully unit-testable in isolation.
package docker

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CollectForwardedEnv returns env vars from the host environment whose keys
// match any of the supplied glob patterns. Patterns are matched using
// filepath.Match semantics (e.g., "AWS_*", "ANTHROPIC_API_KEY").
//
// Returns:
//   - envVars: []string in "KEY=VALUE" format, deduplicated across patterns
//   - warnings: one warning message per pattern that matched zero host vars
func CollectForwardedEnv(patterns []string) (envVars []string, warnings []string) {
	if len(patterns) == 0 {
		return nil, nil
	}

	// Build a map of all host env vars: key → "KEY=VALUE" entry.
	hostEnv := make(map[string]string)
	for _, entry := range os.Environ() {
		idx := strings.Index(entry, "=")
		if idx < 0 {
			continue
		}
		key := entry[:idx]
		hostEnv[key] = entry
	}

	// Track which keys we've already included (deduplication).
	seen := make(map[string]bool)
	// Track which patterns matched at least one var.
	patternMatched := make(map[string]bool)

	for _, pattern := range patterns {
		for key, entry := range hostEnv {
			matched, err := filepath.Match(pattern, key)
			if err != nil {
				// filepath.Match only returns an error for malformed patterns;
				// treat as no match and skip silently.
				continue
			}
			if matched {
				patternMatched[pattern] = true
				if !seen[key] {
					seen[key] = true
					envVars = append(envVars, entry)
				}
			}
		}
	}

	for _, pattern := range patterns {
		if !patternMatched[pattern] {
			warnings = append(warnings, fmt.Sprintf(
				"Warning: forward_env pattern %q did not match any host environment variables",
				pattern,
			))
		}
	}

	return envVars, warnings
}

// ParseEnvFile reads a Docker-compatible .env file and returns a map of
// key → value pairs. The format is one KEY=VALUE per line; lines starting
// with '#' and blank lines are ignored. If the first '=' appears at index i,
// key = line[:i] and value = line[i+1:], so values may themselves contain '='.
//
// Returns an error if the file cannot be read.
func ParseEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read env file %s: %w", path, err)
	}

	result := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := line[:idx]
		val := line[idx+1:]
		result[key] = val
	}
	return result, scanner.Err()
}

// ValidateRequiredEnv checks that every variable in required is available in
// either the host environment or the optional .env file. It is called before
// container launch so the user gets an early, actionable error rather than a
// silent container failure.
//
// Parameters:
//   - required:     list of env var names that must be present
//   - harnessName:  human-readable harness name used in the error message
//   - envFilePath:  path to the .env file, relative to repoDir (or absolute); empty means skip
//   - repoDir:      absolute path to the repo root (used to resolve relative envFilePath)
//
// Returns the first missing variable as a descriptive error, or nil if all are present.
func ValidateRequiredEnv(required []string, harnessName string, envFilePath string, repoDir string) error {
	if len(required) == 0 {
		return nil
	}

	// Build a combined set of available keys: start with host env.
	available := make(map[string]bool)
	for _, entry := range os.Environ() {
		idx := strings.Index(entry, "=")
		if idx < 0 {
			continue
		}
		available[entry[:idx]] = true
	}

	// Supplement with .env file entries if configured.
	if envFilePath != "" {
		resolved := envFilePath
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(repoDir, envFilePath)
		}
		fileVars, err := ParseEnvFile(resolved)
		if err != nil {
			return fmt.Errorf("parse env file: %w", err)
		}
		for k := range fileVars {
			available[k] = true
		}
	}

	for _, v := range required {
		if !available[v] {
			return fmt.Errorf(
				"required environment variable %s is not set. "+
					"The %s harness needs this variable. "+
					"Set it and re-run zone launch",
				v, harnessName,
			)
		}
	}
	return nil
}
