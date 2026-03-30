// ports.go parses port binding strings ("hostPort:containerPort") into Docker API types.
package docker

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/go-connections/nat"
)

// parsePortBindings converts a slice of "hostPort:containerPort" strings into
// nat.PortMap and nat.PortSet suitable for Docker container creation.
// Returns an error if any port string is malformed, out-of-range, or if
// any host port appears more than once (conflicting binding).
func parsePortBindings(ports []string) (nat.PortMap, nat.PortSet, error) {
	bindings := nat.PortMap{}
	exposed := nat.PortSet{}
	seenHostPorts := map[string]bool{}

	for _, p := range ports {
		parts := strings.SplitN(p, ":", 2)
		if len(parts) != 2 {
			return nil, nil, fmt.Errorf("invalid port entry %q: expected hostPort:containerPort", p)
		}
		hostPort, containerPort := parts[0], parts[1]

		if err := validatePort(hostPort); err != nil {
			return nil, nil, fmt.Errorf("invalid host port in %q: %w", p, err)
		}
		if err := validatePort(containerPort); err != nil {
			return nil, nil, fmt.Errorf("invalid container port in %q: %w", p, err)
		}

		if seenHostPorts[hostPort] {
			return nil, nil, fmt.Errorf("conflicting port binding: host port %s appears more than once", hostPort)
		}
		seenHostPorts[hostPort] = true

		natPort, err := nat.NewPort("tcp", containerPort)
		if err != nil {
			return nil, nil, fmt.Errorf("parse container port %q: %w", containerPort, err)
		}

		bindings[natPort] = []nat.PortBinding{{HostPort: hostPort}}
		exposed[natPort] = struct{}{}
	}

	return bindings, exposed, nil
}

// validatePort checks that s is a valid port number in range [1, 65535].
func validatePort(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("port %q is not a valid number", s)
	}
	if n < 1 || n > 65535 {
		return fmt.Errorf("port %d is out of range (1-65535)", n)
	}
	return nil
}
