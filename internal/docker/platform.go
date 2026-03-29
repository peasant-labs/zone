// platform.go detects host platform capabilities (macOS, Linux, rootless Docker).
package docker

import (
	"fmt"
	"os/exec"
	"os/user"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// HostUID returns the current user's numeric UID.
// On Linux/macOS, this is the standard POSIX UID.
// Returns 0 for root users (common in CI environments).
func HostUID() (int, error) {
	u, err := user.Current()
	if err != nil {
		return 0, fmt.Errorf("get current user: %w", err)
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return 0, fmt.Errorf("parse UID %q: %w", u.Uid, err)
	}
	return uid, nil
}

var macUsernameCleanRe = regexp.MustCompile(`[^a-zA-Z0-9_.-]`)

// MacOSUsername returns the sanitized macOS username for symlink compatibility.
// Returns empty string on non-darwin platforms or if detection fails.
// Per user decision: sanitize to [a-zA-Z0-9_.-] only, no length checks.
func MacOSUsername() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	u, err := user.Current()
	if err != nil {
		return "" // silently skip on failure per user decision
	}
	return macUsernameCleanRe.ReplaceAllString(u.Username, "")
}

// DetectGitIdentity returns the host's git user.name and user.email.
// Per locked decision: if EITHER value is empty, returns forward=false.
// Partial config (only name or only email) is treated as missing.
func DetectGitIdentity() (name, email string, forward bool) {
	name = runGitConfig("user.name")
	email = runGitConfig("user.email")
	if name == "" || email == "" {
		return "", "", false
	}
	return name, email, true
}

func runGitConfig(key string) string {
	out, err := exec.Command("git", "config", "--global", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
