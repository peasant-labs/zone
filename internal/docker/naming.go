// naming.go implements deterministic container and network naming from repo paths.
package docker

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
)

var nameCleanRe = regexp.MustCompile(`[^a-zA-Z0-9_.-]`)

// ContainerName returns a deterministic container name derived from the repo's absolute path.
// Format: zone-<sanitized-repo-name>-<16-char-sha256-hash>
func ContainerName(repoPath string) string {
	absPath, _ := filepath.Abs(repoPath)
	hash := sha256.Sum256([]byte(absPath))
	shortHash := hex.EncodeToString(hash[:])[:16]
	repoName := filepath.Base(absPath)
	repoName = nameCleanRe.ReplaceAllString(repoName, "-")
	return fmt.Sprintf("zone-%s-%s", repoName, shortHash)
}

// NetworkName returns the deterministic network name: container name + "-net".
func NetworkName(repoPath string) string {
	return ContainerName(repoPath) + "-net"
}

// ContainerLabels returns Docker labels applied to every zone container for discovery.
func ContainerLabels(repoPath, harness string) map[string]string {
	return map[string]string{
		"com.zone.managed":   "true",
		"com.zone.repo-path": repoPath,
		"com.zone.harness":   harness,
	}
}
