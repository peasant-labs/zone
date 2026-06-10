// resources.go parses human-readable resource limit strings into Docker API integers.
package docker

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
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

// parseGPURequests converts a gpus string into Docker DeviceRequests for GPU
// passthrough, mirroring a subset of `docker run --gpus`. The default driver is
// nvidia (the gpu capability is requested so the runtime injects the devices).
// Accepted forms:
//   - "" / "0" / "none"      → no passthrough (returns nil)
//   - "all"                  → all GPUs
//   - "2"                    → first 2 GPUs (count)
//   - "device=0,1" / "0,1"   → specific GPU device IDs
//
// Requires the NVIDIA Container Toolkit (nvidia runtime) on the host.
func parseGPURequests(s string) ([]container.DeviceRequest, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" || s == "none" {
		return nil, nil
	}

	req := container.DeviceRequest{
		Capabilities: [][]string{{"gpu"}},
	}

	switch {
	case s == "all":
		req.Count = -1
	case strings.HasPrefix(s, "device="):
		req.DeviceIDs = splitTrim(strings.TrimPrefix(s, "device="))
		if len(req.DeviceIDs) == 0 {
			return nil, fmt.Errorf("parse gpus %q: no device IDs specified", s)
		}
	default:
		if n, err := strconv.Atoi(s); err == nil {
			if n <= 0 {
				return nil, fmt.Errorf("parse gpus %q: count must be positive", s)
			}
			req.Count = n
		} else {
			req.DeviceIDs = splitTrim(s)
		}
	}

	return []container.DeviceRequest{req}, nil
}

// splitTrim splits a comma-separated list and trims whitespace, dropping empties.
func splitTrim(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
