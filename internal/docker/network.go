// network.go manages Docker network creation and destruction.
package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"

	"github.com/peasant-labs/zone/internal/config"
)

// createNetwork creates a labeled bridge network for the container.
// If the network already exists, it reuses it by inspecting and returning the existing ID.
// Returns the network ID on success.
func (m *Manager) createNetwork(ctx context.Context, networkName string) (string, error) {
	resp, err := m.client.NetworkCreate(ctx, networkName, network.CreateOptions{
		Driver: "bridge",
		Labels: map[string]string{
			"com.zone.managed":   "true",
			"com.zone.repo-path": m.repoDir,
		},
	})
	if err != nil {
		// If the network already exists, reuse it.
		info, inspectErr := m.client.NetworkInspect(ctx, networkName, network.InspectOptions{})
		if inspectErr != nil {
			return "", fmt.Errorf("create network %s: %w", networkName, err)
		}
		return info.ID, nil
	}
	return resp.ID, nil
}

// removeNetwork removes the network associated with this manager's repo.
// Swallows "not found" errors since the network may have already been removed.
// Clears the cached network ID on success.
func (m *Manager) removeNetwork(ctx context.Context) error {
	netID, err := m.cache.NetworkID()
	if err != nil || netID == "" {
		return nil // no network to remove
	}
	if err := m.client.NetworkRemove(ctx, netID); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("remove network: %w", err)
		}
	}
	return m.cache.SetNetworkID("")
}

// BridgeInterfaceName returns the Linux bridge interface name for a Docker network.
// It inspects the network to get the "com.docker.network.bridge.name" option.
// Falls back to "br-" + netID[:12] if the option is not set.
func (m *Manager) BridgeInterfaceName(ctx context.Context, netID string) string {
	info, err := m.client.NetworkInspect(ctx, netID, network.InspectOptions{})
	if err == nil {
		if name, ok := info.Options["com.docker.network.bridge.name"]; ok && name != "" {
			return name
		}
	}
	if len(netID) >= 12 {
		return "br-" + netID[:12]
	}
	return "br-" + netID
}

// extractProxyHostnames extracts hostnames from http_proxy and https_proxy config values.
// Used for auto-allowlisting in whitelist mode (D-45, D-46, D-47).
func extractProxyHostnames(netCfg *config.NetworkConfig) []string {
	var hostnames []string
	for _, proxyURL := range []string{netCfg.HTTPProxy, netCfg.HTTPSProxy} {
		if proxyURL == "" {
			continue
		}
		host := proxyURL
		if idx := strings.Index(host, "://"); idx >= 0 {
			host = host[idx+3:]
		}
		if idx := strings.Index(host, "/"); idx >= 0 {
			host = host[:idx]
		}
		if idx := strings.Index(host, "@"); idx >= 0 {
			host = host[idx+1:]
		}
		if idx := strings.LastIndex(host, ":"); idx >= 0 {
			host = host[:idx]
		}
		if host != "" {
			hostnames = append(hostnames, host)
		}
	}
	return hostnames
}
