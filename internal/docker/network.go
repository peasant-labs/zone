// network.go manages Docker network creation and destruction.
package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/api/types/network"
)

// createNetwork creates a labeled bridge network for the container.
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
		return "", fmt.Errorf("create network %s: %w", networkName, err)
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
