// hash.go computes the full cache hash from config, templates, and zone version.
package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"

	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/pkg/templates"
)

// ComputeHash returns the hex SHA256 of merged config JSON + Dockerfile template +
// entrypoint template + Zone binary version. Version is passed in (not imported)
// because main.go -> cmd -> internal would violate the import graph.
func ComputeHash(cfg *config.MergedConfig, version string) (string, error) {
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}

	dockerfileTmpl, err := fs.ReadFile(templates.FS, "Dockerfile.tmpl")
	if err != nil {
		return "", fmt.Errorf("read Dockerfile template: %w", err)
	}

	entrypointTmpl, err := fs.ReadFile(templates.FS, "entrypoint.sh.tmpl")
	if err != nil {
		return "", fmt.Errorf("read entrypoint template: %w", err)
	}

	h := sha256.New()
	h.Write(cfgJSON)
	h.Write(dockerfileTmpl)
	h.Write(entrypointTmpl)
	h.Write([]byte(version))

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
