// resources.go parses human-readable resource limit strings into Docker API integers.
package docker

import (
	"fmt"
	"strconv"
	"strings"

	units "github.com/docker/go-units"
)

// parseMemoryBytes converts a memory string (e.g. "512m", "2g") to bytes.
// Returns 0 for empty or "0" strings, which disables the memory limit in the
// Docker API.
func parseMemoryBytes(s string) (int64, error) {
	if s == "" || s == "0" {
		return 0, nil // 0 = no limit in Docker API
	}
	return units.RAMInBytes(s)
}

// parseNanoCPUs converts a CPU string (e.g. "0.5", "2") to nanocpus (1 CPU = 1e9 nanocpus).
// Returns 0 for empty or "0" strings, which disables the CPU limit in the Docker API.
func parseNanoCPUs(s string) (int64, error) {
	if s == "" || s == "0" {
		return 0, nil
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, fmt.Errorf("parse cpus %q: %w", s, err)
	}
	return int64(f * 1e9), nil
}
