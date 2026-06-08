// quickstart.go provides zero-config zone.toml generation for first-time users.
// When `zone launch --harness <name>` is run with no zone.toml present, this
// generates a minimal zone.toml with commented examples, and updates .gitignore.
package docker

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/scaffold"
)

// minimalZoneTomlTemplate is the template for the generated zone.toml file.
// It includes the required fields and commented-out optional sections so users
// can discover configuration options without reading the full documentation.
const minimalZoneTomlTemplate = `version = 1
harness = "%s"

# Uncomment to customize:
# [zone]
# base_image = "ubuntu:24.04"
# shell = "bash"
#
# [resources]
# memory = "4g"
# cpus = "2"
# pids_limit = 512
#
# [workspace]
# persist_home = true
`

// generateMinimalZoneToml returns a minimal zone.toml string for the given harness name.
func generateMinimalZoneToml(harness string) string {
	return fmt.Sprintf(minimalZoneTomlTemplate, harness)
}

// HandleZeroConfig generates a minimal zone.toml in the repo directory and
// ensures the .zone/ directory is listed in .gitignore.
// Called by the CLI when --harness is provided but no zone.toml exists.
func (m *Manager) HandleZeroConfig(harnessName string) error {
	return QuickstartWriteZoneToml(m.repoDir, harnessName)
}

// QuickstartWriteZoneToml writes a minimal zone.toml to repoDir for harnessName
// and ensures .zone/ is listed in .gitignore. Does not require a live Docker daemon.
func QuickstartWriteZoneToml(repoDir, harnessName string) error {
	tomlPath := filepath.Join(repoDir, "zone.toml")
	content := generateMinimalZoneToml(harnessName)
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write zone.toml: %w", err)
	}
	if err := scaffold.EnsureAgentSkill(repoDir); err != nil {
		return fmt.Errorf("create agent skill: %w", err)
	}
	if err := cache.EnsureGitignore(repoDir); err != nil {
		return fmt.Errorf("update gitignore: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Created zone.toml with harness %q\n", harnessName)
	return nil
}
