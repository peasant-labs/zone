// validate.go validates config values including dangerous mount detection,
// Levenshtein key suggestions, and multi-error collection.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agnivade/levenshtein"
)

// ---------------------------------------------------------------------------
// Error types
// ---------------------------------------------------------------------------

// ValidationError represents a single validation issue.
type ValidationError struct {
	Category string // "unknown_key", "dangerous_mount", "type_error", "warning"
	Message  string
	Field    string // dotted field path (e.g., "workspace.extra_mounts")
}

// ValidationErrors is a slice of ValidationError implementing the error interface.
type ValidationErrors []ValidationError

// Error returns a human-readable description of all validation errors.
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "zone.toml has %d issues:\n", len(ve))
	// Group by category and print in order
	groups := map[string][]ValidationError{}
	for _, e := range ve {
		groups[e.Category] = append(groups[e.Category], e)
	}
	for _, cat := range []string{"unknown_key", "dangerous_mount", "type_error", "warning"} {
		if errs, ok := groups[cat]; ok {
			for _, e := range errs {
				fmt.Fprintf(&sb, "  - [%s] %s\n", e.Category, e.Message)
			}
		}
	}
	sb.WriteString("\nRun zone validate after fixing.")
	return sb.String()
}

// HasErrors returns true if there are any non-warning issues.
func (ve ValidationErrors) HasErrors() bool {
	for _, e := range ve {
		if e.Category != "warning" {
			return true
		}
	}
	return false
}

// Warnings returns only warning-level issues.
func (ve ValidationErrors) Warnings() []ValidationError {
	var w []ValidationError
	for _, e := range ve {
		if e.Category == "warning" {
			w = append(w, e)
		}
	}
	return w
}

// DangerousMountError describes a blocked mount path.
type DangerousMountError struct {
	Path     string
	Resolved string
	Chain    string // display chain: "~/docker.sock -> /var/run/docker.sock"
	Reason   string
}

// Error implements the error interface.
func (e *DangerousMountError) Error() string {
	if e.Chain != "" {
		return fmt.Sprintf("blocked mount: %s (%s) — %s", e.Path, e.Chain, e.Reason)
	}
	return fmt.Sprintf("blocked mount: %s (resolves to %s) — %s", e.Path, e.Resolved, e.Reason)
}

// ---------------------------------------------------------------------------
// Dangerous mount blocklist
// ---------------------------------------------------------------------------

type mountRule struct {
	pattern string
	reason  string
}

// dangerousMountBlocklist contains paths that are blocked from mounting.
// Patterns starting with "/." are expanded relative to $HOME.
var dangerousMountBlocklist = []mountRule{
	{pattern: "/var/run/docker.sock", reason: "Docker socket mount allows container escape"},
	{pattern: "/run/docker.sock", reason: "Docker socket mount allows container escape"},
	{pattern: "/var/run/podman/", reason: "Container runtime socket mount"},
	{pattern: "/run/podman/", reason: "Container runtime socket mount"},
	{pattern: "/var/run/containerd/", reason: "Container runtime socket mount"},
	{pattern: "/run/containerd/", reason: "Container runtime socket mount"},
	{pattern: "/proc", reason: "Kernel interface mount"},
	{pattern: "/sys", reason: "Kernel interface mount"},
	{pattern: "/dev", reason: "Device mount"},
	{pattern: "/.ssh", reason: "SSH keys exposure (use forward_ssh_agent instead)"},
	{pattern: "/etc/shadow", reason: "Host credentials file"},
	{pattern: "/etc/passwd", reason: "Host credentials file"},
	{pattern: "/etc", reason: "Host system config mount"},
	{pattern: "/.kube", reason: "Kubernetes credentials exposure"},
	{pattern: "/.aws", reason: "AWS credentials exposure"},
	{pattern: "/.gcp", reason: "GCP credentials exposure"},
	{pattern: "/.azure", reason: "Azure credentials exposure"},
	{pattern: "/.docker", reason: "Docker credentials exposure"},
	{pattern: "/.gnupg", reason: "GPG key exposure"},
	{pattern: "/boot", reason: "Kernel boot files"},
	{pattern: "/lib/modules", reason: "Kernel modules"},
}

// isBlockedMount checks if a resolved path matches any blocklist entry.
// Returns the reason string and true if blocked, empty string and false otherwise.
func isBlockedMount(resolved string) (string, bool) {
	// Special case: root mount
	if resolved == "/" {
		return "Host root mount", true
	}

	home, _ := os.UserHomeDir()

	for _, rule := range dangerousMountBlocklist {
		pattern := rule.pattern
		// Expand home-relative patterns (starting with "/.")
		if strings.HasPrefix(pattern, "/.") && home != "" {
			// e.g., "/.ssh" -> "/home/user/.ssh"
			expanded := filepath.Join(home, pattern[1:]) // strip leading slash before joining
			// Check both the literal ~/.ssh and /root/.ssh etc.
			if resolved == expanded || strings.HasPrefix(resolved, expanded+"/") {
				return rule.reason, true
			}
			// Also check the raw pattern in case home resolution differs
		}

		// Check absolute paths
		if resolved == pattern || strings.HasPrefix(resolved, pattern+"/") {
			return rule.reason, true
		}
		// For patterns without trailing slash that are directories (e.g., "/dev"),
		// also match exact prefix + "/"
	}

	return "", false
}

// resolveSymlinkTarget follows symlinks manually even if the final target doesn't
// exist. Returns the final path after following all symlinks (up to 10 hops).
func resolveSymlinkTarget(path string) string {
	current := path
	for i := 0; i < 10; i++ {
		target, err := os.Readlink(current)
		if err != nil {
			// Not a symlink or error reading it — return current
			return current
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(current), target)
		}
		current = target
	}
	return current
}

// buildSymlinkChain walks a path following symlinks and returns a chain string
// like "a -> b -> c". Returns empty string if path is not a symlink.
func buildSymlinkChain(path string) string {
	var chain []string
	chain = append(chain, path)
	current := path
	for i := 0; i < 10; i++ { // prevent infinite loops
		target, err := os.Readlink(current)
		if err != nil {
			break // not a symlink or error
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(current), target)
		}
		chain = append(chain, target)
		current = target
	}
	if len(chain) <= 1 {
		return ""
	}
	return strings.Join(chain, " -> ")
}

// ---------------------------------------------------------------------------
// Mount permission normalisation
// ---------------------------------------------------------------------------

// NormalizeMountPermission ensures a mount spec has an explicit permission suffix.
// Format: "/host:/container" or "/host:/container:ro|rw"
// Missing permission defaults to ":ro".
func NormalizeMountPermission(mountSpec string) (string, error) {
	parts := strings.Split(mountSpec, ":")
	switch len(parts) {
	case 2: // "/host:/container" — no permission specified, default to ro
		return mountSpec + ":ro", nil
	case 3: // "/host:/container:ro|rw"
		perm := parts[2]
		if perm != "ro" && perm != "rw" {
			return "", fmt.Errorf("invalid mount permission %q in %q (must be 'ro' or 'rw')", perm, mountSpec)
		}
		return mountSpec, nil
	default:
		return "", fmt.Errorf("invalid mount format %q (expected /host:/container[:ro|rw])", mountSpec)
	}
}

// ---------------------------------------------------------------------------
// Levenshtein key suggestion
// ---------------------------------------------------------------------------

// allKnownKeys is the complete list of valid config keys in dotted-path form.
var allKnownKeys = []string{
	"version", "harness",
	"zone.harness", "zone.base_image", "zone.shell",
	"auth.mount_home_config", "auth.forward_env", "auth.forward_ssh_agent", "auth.env_file",
	"workspace.mount_path", "workspace.extra_mounts", "workspace.ports", "workspace.persist_home",
	"packages.apt", "packages.pip", "packages.npm",
	"resources.memory", "resources.cpus", "resources.pids_limit",
	"network.mode", "network.allow", "network.deny", "network.default_allow", "network.default_deny",
	"network.http_proxy", "network.https_proxy", "network.no_proxy",
	"hooks.pre_build", "hooks.post_stop",
	"harness.version", "harness.skip_permissions", "harness.node_version",
	"harness.python_version", "harness.extra_args",
	"harness.install_commands", "harness.entrypoint_command", "harness.config_dirs",
	"harness.required_env", "harness.health_check", "harness.aliases", "harness.shell_rc",
}

// SuggestKey finds the closest known key to the unknown key within distance 3.
// Returns the best matching fully-qualified key and true, or empty string and false.
//
// Supports two matching strategies:
//  1. Full key comparison: "zone.baes_image" vs "zone.base_image"
//  2. Bare-name match: unknown key without section prefix matched against bare names
//  3. Section-aware bare-name: dotted unknown matched with bare name from same section
func SuggestKey(unknown string) (string, bool) {
	best := ""
	bestDist := 4 // threshold: only suggest if distance <= 3

	// First pass: check against fully-qualified keys
	for _, known := range allKnownKeys {
		d := levenshtein.ComputeDistance(unknown, known)
		if d < bestDist {
			bestDist = d
			best = known
		}
	}

	// Second pass: bare-name match (per pitfall 7 — user may omit section prefix)
	// If unknown has no dot, also check against bare key names within sections.
	if !strings.Contains(unknown, ".") {
		for _, known := range allKnownKeys {
			parts := strings.SplitN(known, ".", 2)
			if len(parts) == 2 {
				bare := parts[1]
				d := levenshtein.ComputeDistance(unknown, bare)
				if d < bestDist {
					bestDist = d
					best = known // return fully-qualified name
				}
			}
		}
	}

	// Third pass: section-aware bare comparison — for dotted unknowns where the
	// section name is valid, compare the bare part against bare parts of keys in
	// the same section. Uses lenient threshold (8) to catch truncations like
	// "harness.skip_perms" -> "harness.skip_permissions".
	// A section-aware match overrides the first/second pass only when no closer
	// match was found.
	if strings.Contains(unknown, ".") && best == "" {
		unknownParts := strings.SplitN(unknown, ".", 2)
		unknownSection := unknownParts[0]
		unknownBare := unknownParts[1]
		lenientThreshold := 9 // allow longer edit distances for same-section bare matches
		sectionBest := ""
		sectionBestDist := lenientThreshold
		for _, known := range allKnownKeys {
			knownParts := strings.SplitN(known, ".", 2)
			if len(knownParts) == 2 && knownParts[0] == unknownSection {
				knownBare := knownParts[1]
				d := levenshtein.ComputeDistance(unknownBare, knownBare)
				if d < sectionBestDist {
					sectionBestDist = d
					sectionBest = known
				}
			}
		}
		if sectionBest != "" {
			best = sectionBest
		}
	}

	return best, best != ""
}

// FormatSuggestion formats a dotted key as a section-aware display string.
// "harness.skip_permissions" -> "[harness] skip_permissions"
func FormatSuggestion(key string) string {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) == 2 {
		return fmt.Sprintf("[%s] %s", parts[0], parts[1])
	}
	return key
}

// ---------------------------------------------------------------------------
// Main Validate function
// ---------------------------------------------------------------------------

// Validate performs all config validations on a MergedConfig and returns
// collected ValidationErrors. All issues are gathered in one pass.
func Validate(cfg *MergedConfig) ValidationErrors {
	var errs ValidationErrors

	// 1. Mount validation: check each extra_mount
	for _, mount := range cfg.Workspace.ExtraMounts {
		// Normalize mount permission
		normalized, err := NormalizeMountPermission(mount)
		if err != nil {
			errs = append(errs, ValidationError{
				Category: "type_error",
				Message:  err.Error(),
				Field:    "workspace.extra_mounts",
			})
			continue
		}

		// Extract host path (first component)
		hostPath := strings.Split(normalized, ":")[0]

		// Resolve symlinks: EvalSymlinks resolves the full chain but requires all
		// components to exist. If the final target doesn't exist (e.g., a symlink
		// pointing to /var/run/docker.sock which may not exist on this host), we
		// fall back to reading the symlink chain manually.
		chain := buildSymlinkChain(hostPath)
		resolved, resolveErr := filepath.EvalSymlinks(hostPath)
		if resolveErr != nil {
			if os.IsNotExist(resolveErr) {
				// Path or its target doesn't exist — check the raw symlink target
				// by walking the chain ourselves.
				resolved = resolveSymlinkTarget(hostPath)
			} else {
				errs = append(errs, ValidationError{
					Category: "warning",
					Message:  fmt.Sprintf("cannot resolve mount path %s: %v", hostPath, resolveErr),
					Field:    "workspace.extra_mounts",
				})
				continue
			}
		}

		if reason, blocked := isBlockedMount(resolved); blocked {
			errs = append(errs, ValidationError{
				Category: "dangerous_mount",
				Message: (&DangerousMountError{
					Path:     hostPath,
					Resolved: resolved,
					Chain:    chain,
					Reason:   reason,
				}).Error(),
				Field: "workspace.extra_mounts",
			})
		}
	}

	// 2. Base image tag warning
	if cfg.Zone.BaseImage != "" && !strings.Contains(cfg.Zone.BaseImage, ":") {
		errs = append(errs, ValidationError{
			Category: "warning",
			Message:  fmt.Sprintf("base_image %q has no tag — consider pinning (e.g., %q)", cfg.Zone.BaseImage, cfg.Zone.BaseImage+":24.04"),
			Field:    "zone.base_image",
		})
	}

	// 3. Network mode "none" with non-empty allow list
	if cfg.Network.Mode == "none" && len(cfg.Network.Allow) > 0 {
		errs = append(errs, ValidationError{
			Category: "warning",
			Message:  `network.mode is "none" but allow list is non-empty — allow list will be ignored`,
			Field:    "network.mode",
		})
	}

	// 4. Conflicting port entries
	portsSeen := make(map[string]bool)
	for _, p := range cfg.Workspace.Ports {
		hostPort := strings.Split(p, ":")[0]
		if portsSeen[hostPort] {
			errs = append(errs, ValidationError{
				Category: "type_error",
				Message:  fmt.Sprintf("conflicting port mapping: host port %s used more than once", hostPort),
				Field:    "workspace.ports",
			})
		}
		portsSeen[hostPort] = true
	}

	return errs
}

// ValidateUnknownKeys converts a list of unknown keys from TOML parsing into
// ValidationErrors with Levenshtein suggestions where available.
func ValidateUnknownKeys(keys []string, file string) ValidationErrors {
	var errs ValidationErrors
	for _, key := range keys {
		msg := fmt.Sprintf("unknown key %q in %s", key, file)
		if suggestion, ok := SuggestKey(key); ok {
			msg += fmt.Sprintf(". Did you mean %s?", FormatSuggestion(suggestion))
		}
		errs = append(errs, ValidationError{
			Category: "unknown_key",
			Message:  msg,
			Field:    key,
		})
	}
	return errs
}
