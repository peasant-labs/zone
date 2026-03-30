// launch.go implements the Launch state machine — the core of `zone launch`.
// It handles all container states: running, paused, exited, dead, created,
// restarting, stale (externally deleted), and fresh (no prior container).
// It also handles config change detection, headless mode, and lock management.
package docker

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/errdefs"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/harness"
)

// LaunchOpts configures the behaviour of Manager.Launch.
type LaunchOpts struct {
	// Headless prints the container ID to stdout and returns immediately,
	// without attaching a TTY. Useful for fire-and-forget agent workflows.
	Headless bool

	// Prompt is forwarded to the harness via its PromptFlag() when non-empty.
	Prompt string

	// Rebuild forces a fresh image build regardless of hash state.
	Rebuild bool

	// NoCache passes --no-cache to the Docker build.
	NoCache bool

	// HarnessArgs are appended verbatim to the harness entrypoint command.
	HarnessArgs []string

	// Ports are ad-hoc host:container port bindings from --port/-P.
	Ports []string
}

// Launch implements the full zone launch state machine:
//
//  1. Acquire exclusive lock on .zone/
//  2. Inspect any cached container ID and branch on its state
//  3. Build image if needed (hash mismatch or force rebuild)
//  4. Create and start a new container
//  5. Release lock (before TTY attach so zone join can connect concurrently)
//  6. Headless: print container ID and return; Interactive: exec -it attach
func (m *Manager) Launch(ctx context.Context, opts LaunchOpts) error {
	if len(opts.Ports) > 0 {
		m.config.Workspace.Ports = append(m.config.Workspace.Ports, opts.Ports...)
	}

	// Ensure cache directory exists before acquiring lock.
	if err := m.cache.EnsureDir(); err != nil {
		return err
	}

	// Step 1: acquire exclusive lock.
	lock := cache.NewLock(m.cache.Dir())
	if err := lock.Acquire(); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	// NOTE: lock.Release() is called EXPLICITLY before attach — do NOT defer here.

	// Step 1.5: Pre-launch env validation (CFG-11)
	h, harnessErr := harness.Get(m.config.Zone.Harness, &m.config.Harness)
	if harnessErr == nil {
		required := h.RequiredEnvVars()
		// Also include custom harness required_env if applicable
		if len(m.config.Harness.RequiredEnv) > 0 {
			required = append(required, m.config.Harness.RequiredEnv...)
		}
		if err := ValidateRequiredEnv(required, h.Name(), m.config.Auth.EnvFile, m.repoDir); err != nil {
			lock.Release()
			return err
		}
	}

	// Step 2: inspect any existing container.
	containerID, err := m.cache.ContainerID()
	if err != nil {
		lock.Release()
		return fmt.Errorf("read container ID from cache: %w", err)
	}

	if containerID != "" {
		info, err := m.inspectContainerState(ctx, containerID)
		if err != nil {
			lock.Release()
			return err
		}

		if info == nil {
			// Stale: container was deleted externally.
			if err := m.cleanStaleCache(ctx); err != nil {
				lock.Release()
				return err
			}
			// Fall through to build path below.
		} else {
			// Branch on container state.
			switch info.State.Status {
			case "running":
				return m.handleRunning(ctx, info, lock, opts)

			case "paused":
				if err := m.client.ContainerUnpause(ctx, containerID); err != nil {
					lock.Release()
					return fmt.Errorf("unpause container: %w", err)
				}
				lock.Release()
				return m.attachFn(containerID, m.harnessCmd(opts), false)

			case "exited", "dead":
				if info.State.OOMKilled {
					fmt.Fprintf(os.Stderr, "Warning: Container was killed due to memory limit. Increase resources.memory in zone.toml.\n")
				}
				if err := m.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
					lock.Release()
					return fmt.Errorf("remove exited container: %w", err)
				}
				_ = m.removeNetwork(ctx)
				_ = m.cache.SetContainerID("")
				_ = m.cache.SetNetworkID("")
				// Fall through to build path.

			case "created", "restarting":
				time.Sleep(2 * time.Second)
				timeout := 5
				_ = m.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
				if err := m.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
					lock.Release()
					return fmt.Errorf("remove stuck container: %w", err)
				}
				_ = m.removeNetwork(ctx)
				_ = m.cache.SetContainerID("")
				_ = m.cache.SetNetworkID("")
				// Fall through to build path.
			}
		}
	}

	// Step 2.5: Run pre_build hooks (CFG-18)
	if len(m.config.Hooks.PreBuild) > 0 {
		if err := runHooks(m.config.Hooks.PreBuild, m.repoDir, true, os.Stderr); err != nil {
			lock.Release()
			return fmt.Errorf("pre_build: %w", err)
		}
	}

	// Step 3: build image if needed.
	if err := m.buildIfNeeded(ctx, opts.Rebuild, opts.NoCache); err != nil {
		lock.Release()
		return err
	}

	// Step 4: create and start container.
	newContainerID, err := m.createAndStart(ctx)
	if err != nil {
		lock.Release()
		return err
	}

	// Step 5: release lock BEFORE attach so zone join can connect.
	lock.Release()

	// Step 6: headless or interactive.
	if opts.Headless {
		fmt.Println(newContainerID)
		return nil
	}
	return m.attachFn(newContainerID, m.harnessCmd(opts), false)
}

// inspectContainerState wraps ContainerInspect and translates a "not found" error
// into (nil, nil) so callers can detect stale container IDs without error propagation.
func (m *Manager) inspectContainerState(ctx context.Context, containerID string) (*container.InspectResponse, error) {
	info, err := m.client.ContainerInspect(ctx, containerID)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil, nil // stale cache entry
		}
		return nil, fmt.Errorf("inspect container: %w", err)
	}
	return &info, nil
}

// handleRunning handles the case where the cached container is already running.
// It compares the current config hash to the running container's hash.
// If they differ, it prints a warning but does NOT stop the container.
// The lock is released before attach.
func (m *Manager) handleRunning(ctx context.Context, info *container.InspectResponse, lock *cache.Lock, opts LaunchOpts) error {
	changed, err := m.checkConfigHash()
	if err != nil {
		lock.Release()
		return err
	}
	if changed {
		fmt.Fprintf(os.Stderr, "Config has changed since this container was started. Run 'zone restart --rebuild' to apply changes.\n")
	}

	containerID := info.ID
	lock.Release()

	if opts.Headless {
		fmt.Println(containerID)
		return nil
	}
	return m.attachFn(containerID, m.harnessCmd(opts), false)
}

// checkConfigHash computes the current config hash and compares it to the
// cached hash. Returns (true, nil) when the hashes differ (config is stale).
func (m *Manager) checkConfigHash() (changed bool, err error) {
	current, err := cache.ComputeHash(m.config, m.version)
	if err != nil {
		return false, fmt.Errorf("compute config hash: %w", err)
	}
	cached, err := m.cache.ConfigHash()
	if err != nil {
		return false, fmt.Errorf("read cached config hash: %w", err)
	}
	return current != cached, nil
}

// cleanStaleCache clears the container_id from cache, attempts to remove any
// orphaned network (swallowing errors), and clears the cached network_id.
func (m *Manager) cleanStaleCache(ctx context.Context) error {
	if err := m.cache.SetContainerID(""); err != nil {
		return fmt.Errorf("clear stale container ID: %w", err)
	}
	_ = m.removeNetwork(ctx) // best-effort: orphaned network may already be gone
	_ = m.cache.SetNetworkID("")
	return nil
}

// buildIfNeeded decides whether a Docker image build is required.
// It builds unconditionally when forceRebuild is true. Otherwise it computes
// the current config hash and skips the build when the hash matches the cached
// hash AND the image still exists in Docker.
func (m *Manager) buildIfNeeded(ctx context.Context, forceRebuild, noCache bool) error {
	if forceRebuild {
		_, err := m.buildImage(ctx, noCache)
		return err
	}

	current, err := cache.ComputeHash(m.config, m.version)
	if err != nil {
		return fmt.Errorf("compute config hash: %w", err)
	}

	cached, err := m.cache.ConfigHash()
	if err != nil {
		return fmt.Errorf("read cached config hash: %w", err)
	}

	if current == cached {
		// Hashes match — check the image still physically exists.
		imageID, err := m.cache.ImageID()
		if err == nil && imageID != "" {
			if _, _, err := m.client.ImageInspectWithRaw(ctx, imageID); err == nil {
				return nil // image exists, skip build
			}
		}
	}

	// Hash mismatch or image missing — build.
	_, err = m.buildImage(ctx, noCache)
	return err
}

// createAndStart creates a container from the cached image ID, starts it,
// persists the container ID to cache, and returns the container ID.
func (m *Manager) createAndStart(ctx context.Context) (string, error) {
	imageID, err := m.cache.ImageID()
	if err != nil {
		return "", fmt.Errorf("read cached image ID: %w", err)
	}

	containerID, err := m.createContainer(ctx, imageID)
	if err != nil {
		return "", err
	}

	if err := m.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("start container: %w", err)
	}

	if err := m.cache.SetContainerID(containerID); err != nil {
		return "", fmt.Errorf("cache container ID: %w", err)
	}

	return containerID, nil
}

// harnessCmd builds the full command slice for the harness entrypoint.
// It retrieves the harness, gets its EntrypointCommand, and appends any
// prompt flag and extra args from opts.
func (m *Manager) harnessCmd(opts LaunchOpts) []string {
	h, err := harness.Get(m.config.Zone.Harness, &m.config.Harness)
	if err != nil {
		// Fallback: return minimal shell command if harness lookup fails.
		return []string{m.config.Zone.Shell}
	}

	cmd := []string{h.EntrypointCommand()}

	if opts.Prompt != "" && h.PromptFlag() != "" {
		cmd = append(cmd, h.PromptFlag(), opts.Prompt)
	}

	cmd = append(cmd, opts.HarnessArgs...)
	return cmd
}
