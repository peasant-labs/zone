// Package docker provides Docker container lifecycle management via the Docker SDK.
package docker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/config"
)

// Manager orchestrates Docker container lifecycle operations for a single repository.
// It holds a Docker client (interface for testability), config, cache, and metadata.
type Manager struct {
	client  DockerClient
	config  *config.MergedConfig
	cache   *cache.Cache
	repoDir string // absolute path to repo root
	version string // zone binary version for template rendering

	// attachFn is the function used to attach an interactive TTY session.
	// Defaults to attachInteractive; overridden in tests with a no-op.
	attachFn func(containerID string, cmd []string, asRoot bool) error
}

// NewManager creates a Manager, verifying Docker daemon connectivity via Ping().
// Returns ErrDockerNotRunning if the daemon is unreachable.
func NewManager(cfg *config.MergedConfig, c *cache.Cache, repoDir, version string) (*Manager, error) {
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client init: %w", err)
	}
	if _, err := cli.Ping(context.Background()); err != nil {
		cli.Close()
		return nil, fmt.Errorf("%w: %v", ErrDockerNotRunning, err)
	}
	absDir, _ := filepath.Abs(repoDir)
	m := &Manager{client: cli, config: cfg, cache: c, repoDir: absDir, version: version}
	m.attachFn = m.attachInteractive
	return m, nil
}

// newManagerWithClient creates a Manager with a pre-existing DockerClient.
// Used in unit tests to inject a mock client without requiring a live Docker daemon.
func newManagerWithClient(cli DockerClient, cfg *config.MergedConfig, c *cache.Cache, repoDir, version string) *Manager {
	absDir, _ := filepath.Abs(repoDir)
	m := &Manager{client: cli, config: cfg, cache: c, repoDir: absDir, version: version}
	m.attachFn = m.attachInteractive
	return m
}

// Build renders templates, creates a build context, and builds a Docker image.
// It is the public entry point for both `zone build` and the Launch state machine.
// Returns the image ID on success.
func (m *Manager) Build(ctx context.Context, noCache bool) (string, error) {
	if err := m.cache.EnsureDir(); err != nil {
		return "", err
	}
	return m.buildImage(ctx, noCache)
}

// createContainer creates a Docker container for the given imageID.
// It applies security flags, labels, mounts, and resource limits from config.
// Returns the container ID on success.
func (m *Manager) createContainer(ctx context.Context, imageID string) (string, error) {
	containerName := ContainerName(m.repoDir)
	networkName := NetworkName(m.repoDir)

	// Create the dedicated bridge network
	netID, err := m.createNetwork(ctx, networkName)
	if err != nil {
		return "", err
	}
	if err := m.cache.SetNetworkID(netID); err != nil {
		return "", fmt.Errorf("cache network ID: %w", err)
	}

	sec := ContainerSecurityFlags()
	pidsLimit := sec.DefaultPidsLimit
	if m.config.Resources.PidsLimit > 0 {
		pidsLimit = int64(m.config.Resources.PidsLimit)
	}

	memBytes, err := parseMemoryBytes(m.config.Resources.Memory)
	if err != nil {
		return "", fmt.Errorf("parse memory: %w", err)
	}
	nanoCPUs, err := parseNanoCPUs(m.config.Resources.Cpus)
	if err != nil {
		return "", fmt.Errorf("parse cpus: %w", err)
	}

	mounts := m.buildMounts()

	cfg := &container.Config{
		Image:  imageID,
		Labels: ContainerLabels(m.repoDir, m.config.Zone.Harness),
	}

	hostCfg := &container.HostConfig{
		SecurityOpt: sec.SecurityOpt,
		CapDrop:     strslice.StrSlice(sec.CapDrop),
		CapAdd:      strslice.StrSlice(sec.CapAdd),
		Resources: container.Resources{
			Memory:    memBytes,
			NanoCPUs:  nanoCPUs,
			PidsLimit: &pidsLimit,
		},
		Sysctls: map[string]string{
			"net.ipv6.conf.all.disable_ipv6": "1",
		},
		Mounts: mounts,
	}

	netCfg := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {},
		},
	}

	resp, err := m.client.ContainerCreate(ctx, cfg, hostCfg, netCfg, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}
	return resp.ID, nil
}

// buildMounts constructs the list of Docker mounts for the container.
// Always includes a bind mount for the workspace. Includes a home volume
// mount when persist_home is true (the default when nil).
func (m *Manager) buildMounts() []mount.Mount {
	mountPath := m.config.Workspace.MountPath
	if mountPath == "" {
		mountPath = "/workspace"
	}

	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: m.repoDir,
			Target: mountPath,
		},
	}

	// Home volume persistence: default is true when PersistHome is nil
	persistHome := m.config.Workspace.PersistHome == nil || *m.config.Workspace.PersistHome
	if persistHome {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeVolume,
			Source: homeVolumeName(m.repoDir),
			Target: "/home/zone",
		})
	}

	return mounts
}

// homeVolumeName returns the deterministic Docker volume name for a repo's home dir.
// Format: zone-home-<16-char-sha256-hash> derived from the absolute repo path.
func homeVolumeName(repoPath string) string {
	absPath, _ := filepath.Abs(repoPath)
	hash := sha256.Sum256([]byte(absPath))
	shortHash := hex.EncodeToString(hash[:])[:16]
	return fmt.Sprintf("zone-home-%s", shortHash)
}

// attachInteractive runs an interactive TTY session inside the container using
// `docker exec -it`. Uses os/exec because the SDK's hijacked connection API is
// unreliable for raw terminal I/O (locked decision).
func (m *Manager) attachInteractive(containerID string, cmd []string, asRoot bool) error {
	args := []string{"exec", "-it"}
	if asRoot {
		args = append(args, "-u", "root")
	}
	args = append(args, containerID)
	args = append(args, cmd...)
	c := exec.Command("docker", args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
