// Package docker provides Docker container lifecycle management via the Docker SDK.
package docker

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	dockernetwork "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/internal/harness"
	"github.com/peasant-labs/zone/internal/network"
)

// Manager orchestrates Docker container lifecycle operations for a single repository.
// It holds a Docker client (interface for testability), config, cache, and metadata.
type Manager struct {
	client   DockerClient
	config   *config.MergedConfig
	cache    *cache.Cache
	repoDir  string // absolute path to repo root
	version  string // zone binary version for template rendering
	platform Platform

	// attachFn is the function used to attach an interactive TTY session.
	// Defaults to attachInteractive; overridden in tests with a no-op.
	attachFn func(containerID string, cmd []string, asRoot bool) error

	// firewall manages iptables rules for the current container.
	// nil when mode=none or platform doesn't support iptables.
	firewall *network.Firewall

	// firewallCancel stops the background refresh goroutine.
	// nil when no refresh is running.
	firewallCancel context.CancelFunc
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
	platform := DetectPlatform(context.Background(), cli)
	absDir, _ := filepath.Abs(repoDir)
	m := &Manager{client: cli, config: cfg, cache: c, repoDir: absDir, version: version, platform: platform}
	m.attachFn = m.attachInteractive
	return m, nil
}

// newManagerWithClient creates a Manager with a pre-existing DockerClient.
// Used in unit tests to inject a mock client without requiring a live Docker daemon.
func newManagerWithClient(cli DockerClient, cfg *config.MergedConfig, c *cache.Cache, repoDir, version string) *Manager {
	absDir, _ := filepath.Abs(repoDir)
	m := &Manager{client: cli, config: cfg, cache: c, repoDir: absDir, version: version, platform: DetectPlatform(context.Background(), cli)}
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

	// Collect forwarded env vars (CFG-10)
	envVars, envWarnings := CollectForwardedEnv(m.config.Auth.ForwardEnv)
	for _, w := range envWarnings {
		fmt.Fprintln(os.Stderr, w)
	}

	// Load .env file vars (CFG-14)
	if m.config.Auth.EnvFile != "" {
		envFilePath := m.config.Auth.EnvFile
		if !filepath.IsAbs(envFilePath) {
			envFilePath = filepath.Join(m.repoDir, envFilePath)
		}
		envFileVars, err := ParseEnvFile(envFilePath)
		if err != nil {
			return "", fmt.Errorf("load env file: %w", err)
		}
		for k, v := range envFileVars {
			envVars = append(envVars, k+"="+v)
		}
	}

	// Add proxy env vars (CFG-15)
	httpProxy, httpsProxy, noProxy := resolveProxy(&m.config.Network)
	envVars = append(envVars, proxyEnvVars(httpProxy, httpsProxy, noProxy)...)

	// Add SSH_AUTH_SOCK env var if socket was mounted (CFG-12)
	if m.config.Auth.ForwardSSHAgent != nil && *m.config.Auth.ForwardSSHAgent && runtime.GOOS != "darwin" {
		sock := os.Getenv("SSH_AUTH_SOCK")
		if sock != "" {
			if fi, err := os.Stat(sock); err == nil && fi.Mode()&os.ModeSocket != 0 {
				envVars = append(envVars, "SSH_AUTH_SOCK=/tmp/ssh-agent.sock")
			}
		}
	}

	// Parse port bindings (CFG-16)
	portBindings, exposedPorts, err := parsePortBindings(m.config.Workspace.Ports)
	if err != nil {
		return "", fmt.Errorf("parse ports: %w", err)
	}

	cfg := &container.Config{
		Image:        imageID,
		Labels:       ContainerLabels(m.repoDir, m.config.Zone.Harness),
		Env:          envVars,
		ExposedPorts: exposedPorts,
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
		Mounts:       mounts,
		PortBindings: portBindings,
	}

	netCfg := &dockernetwork.NetworkingConfig{
		EndpointsConfig: map[string]*dockernetwork.EndpointSettings{
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

	// SSH agent forwarding (CFG-12)
	if m.config.Auth.ForwardSSHAgent != nil && *m.config.Auth.ForwardSSHAgent {
		if runtime.GOOS == "darwin" {
			fmt.Fprintf(os.Stderr, "Warning: SSH agent forwarding is not available on macOS (domain sockets cannot be bind-mounted). SSH operations inside the container will not have agent access.\n")
		} else {
			sock := os.Getenv("SSH_AUTH_SOCK")
			if sock == "" {
				fmt.Fprintf(os.Stderr, "Warning: SSH_AUTH_SOCK is not set or socket not found. SSH agent forwarding skipped.\n")
			} else if fi, err := os.Stat(sock); err == nil && fi.Mode()&os.ModeSocket != 0 {
				mounts = append(mounts, mount.Mount{
					Type:     mount.TypeBind,
					Source:   sock,
					Target:   "/tmp/ssh-agent.sock",
					ReadOnly: true,
				})
			} else {
				fmt.Fprintf(os.Stderr, "Warning: SSH_AUTH_SOCK is not set or socket not found. SSH agent forwarding skipped.\n")
			}
		}
	}

	// Auth config mounts (CFG-13) — copy-on-start strategy
	mountHomeConfig := m.config.Auth.MountHomeConfig == nil || *m.config.Auth.MountHomeConfig
	if mountHomeConfig {
		h, err := harness.Get(m.config.Zone.Harness, &m.config.Harness)
		if err == nil {
			configDirs := collectConfigDirs(h)
			for _, dir := range configDirs {
				expanded := expandHome(dir)
				if _, err := os.Stat(expanded); os.IsNotExist(err) {
					continue // skip missing dir
				}
				mounts = append(mounts, mount.Mount{
					Type:     mount.TypeBind,
					Source:   expanded,
					Target:   dir + ".host",
					ReadOnly: true,
				})
			}
		}
	}

	return mounts
}

// collectConfigDirs returns the HomeConfigDir plus ExtraConfigDirs for a harness,
// filtering out empty strings.
func collectConfigDirs(h harness.Harness) []string {
	var dirs []string
	if home := h.HomeConfigDir(); home != "" {
		dirs = append(dirs, home)
	}
	for _, d := range h.ExtraConfigDirs() {
		if d != "" {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// expandHome replaces a leading "~/" with the user's home directory.
// If os.UserHomeDir() returns an error, the path is returned unchanged.
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home + path[1:]
	}
	return path
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

// Stop stops and removes the container and its network, then clears container_id and
// network_id from cache. The image and config.hash are retained so a subsequent
// `zone launch` can reuse the existing image. Safe to call when no container exists.
func (m *Manager) Stop(ctx context.Context) error {
	containerID, err := m.cache.ContainerID()
	if err != nil {
		return fmt.Errorf("read container ID: %w", err)
	}
	if containerID == "" {
		return nil // no-op: already stopped
	}

	// Stop the container with a 10-second graceful timeout.
	timeout := 10
	if err := m.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("stop container: %w", err)
		}
		// Container already gone — continue to cleanup
	}

	// Force-remove the container (idempotent: swallow NotFound).
	if err := m.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("remove container: %w", err)
		}
	}

	// Cancel the refresh goroutine before removing rules (D-25)
	if m.firewallCancel != nil {
		m.firewallCancel()
		m.firewallCancel = nil
	}

	// Remove iptables rules before removing the network (D-38)
	// If m.firewall is nil (fresh process), reconstruct from cache/naming.
	fw := m.firewall
	if fw == nil {
		fw = m.reconstructFirewallForCleanup(ctx)
	}
	if fw != nil {
		if err := fw.Remove(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove firewall rules: %v\n", err)
		}
		_ = fw.RemoveRulesCache()
		m.firewall = nil
	}

	// Remove the dedicated bridge network.
	if err := m.removeNetwork(ctx); err != nil {
		return err
	}

	// Clear container and network IDs from cache; retain image_id and config.hash.
	if err := m.cache.SetContainerID(""); err != nil {
		return fmt.Errorf("clear container ID: %w", err)
	}
	if err := m.cache.SetNetworkID(""); err != nil {
		return fmt.Errorf("clear network ID: %w", err)
	}

	// Run post_stop hooks (CFG-18) — best-effort, warn on failure
	if len(m.config.Hooks.PostStop) > 0 {
		_ = runHooks(m.config.Hooks.PostStop, m.repoDir, false, os.Stderr)
	}

	return nil
}

// reconstructFirewallForCleanup creates a Firewall instance from cached/derived
// state for cleanup purposes. Used when m.firewall is nil (fresh process that
// didn't launch the container) but firewall rules may exist on the host.
//
// Returns nil if:
// - Network mode is "none" or empty (no firewall rules to clean)
// - Platform doesn't support iptables
// - repoDir is empty (can't derive container hash)
func (m *Manager) reconstructFirewallForCleanup(ctx context.Context) *network.Firewall {
	mode := m.config.Network.Mode
	if mode == "" || strings.ToLower(mode) == "none" {
		return nil
	}
	if !m.platform.SupportsIPTables {
		return nil
	}

	containerName := ContainerName(m.repoDir)
	if len(containerName) < 16 {
		return nil
	}
	containerHash := containerName[len(containerName)-16:]

	netID, _ := m.cache.NetworkID()
	bridgeIface := ""
	if netID != "" {
		bridgeIface = m.BridgeInterfaceName(ctx, netID)
	}

	return network.NewFirewall(containerHash, bridgeIface, m.cache.Dir(), nil)
}

// Destroy performs a full teardown: Stop, remove image, remove home volume, and
// wipe all .zone/ cache files. After Destroy, the repo is in a pristine state
// as if zone had never been run. Safe to call when no container exists.
func (m *Manager) Destroy(ctx context.Context) error {
	// Stop removes container + network and clears their cache IDs.
	if err := m.Stop(ctx); err != nil {
		return err
	}

	// Remove Docker image by cached ID (swallow NotFound — already pruned is fine).
	imageID, err := m.cache.ImageID()
	if err != nil {
		return fmt.Errorf("read image ID: %w", err)
	}
	if imageID != "" {
		if _, err := m.client.ImageRemove(ctx, imageID, image.RemoveOptions{Force: false, PruneChildren: true}); err != nil {
			if !errdefs.IsNotFound(err) {
				return fmt.Errorf("remove image: %w", err)
			}
		}
	}

	// Remove home volume (swallow NotFound — volume may never have been created).
	if err := m.client.VolumeRemove(ctx, homeVolumeName(m.repoDir), false); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("remove home volume: %w", err)
		}
	}

	// Wipe all .zone/ cache files.
	return m.cache.Clean()
}

// RemoveImage removes the cached Docker image and clears image_id from cache.
// Used by `zone clean --image`. Safe to call when no image is cached.
func (m *Manager) RemoveImage(ctx context.Context) error {
	imageID, err := m.cache.ImageID()
	if err != nil {
		return fmt.Errorf("read image ID: %w", err)
	}
	if imageID == "" {
		return nil // no-op: no image cached
	}

	if _, err := m.client.ImageRemove(ctx, imageID, image.RemoveOptions{Force: false, PruneChildren: true}); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("remove image: %w", err)
		}
	}

	return m.cache.SetImageID("")
}

// Join attaches a new interactive shell to the running container.
// Returns ErrNoContainer when no container is cached or the cached container
// is not in the "running" state.
func (m *Manager) Join(ctx context.Context) error {
	containerID, err := m.cache.ContainerID()
	if err != nil {
		return fmt.Errorf("read container ID: %w", err)
	}
	if containerID == "" {
		return ErrNoContainer
	}

	info, err := m.client.ContainerInspect(ctx, containerID)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return ErrNoContainer
		}
		return fmt.Errorf("inspect container: %w", err)
	}
	if info.State.Status != "running" {
		return fmt.Errorf("container is %s, not running. Run `zone launch` first", info.State.Status)
	}

	shell := m.config.Zone.Shell
	if shell == "" {
		shell = "bash"
	}
	return m.attachFn(containerID, []string{shell}, false)
}

// Exec runs a one-off command inside the running container.
// Returns ErrNoContainer when no container is cached.
// asRoot runs the command as root inside the container.
func (m *Manager) Exec(ctx context.Context, command []string, asRoot bool) error {
	containerID, err := m.cache.ContainerID()
	if err != nil {
		return fmt.Errorf("read container ID: %w", err)
	}
	if containerID == "" {
		return ErrNoContainer
	}
	return m.attachFn(containerID, command, asRoot)
}

// Shell opens an interactive shell inside the running container without running
// the harness entrypoint. Returns ErrNoContainer when no container is cached.
func (m *Manager) Shell(ctx context.Context) error {
	containerID, err := m.cache.ContainerID()
	if err != nil {
		return fmt.Errorf("read container ID: %w", err)
	}
	if containerID == "" {
		return ErrNoContainer
	}

	shell := m.config.Zone.Shell
	if shell == "" {
		shell = "bash"
	}
	return m.attachFn(containerID, []string{shell}, false)
}

// ContainerInfo holds metadata for a zone-managed container returned by List.
type ContainerInfo struct {
	Name      string    `json:"name"`
	Harness   string    `json:"harness"`
	Status    string    `json:"status"`
	State     string    `json:"state"`
	StartedAt time.Time `json:"started_at"`
	RepoPath  string    `json:"repo_path"`
	ID        string    `json:"id"`
}

// LogsOpts configures Manager.Logs behavior.
type LogsOpts struct {
	Follow bool
	Tail   string // "all" or a number string like "100"
	JSON   bool
}

// List queries Docker for all containers with the com.zone.managed=true label.
// Does not require a zone.toml — works globally across all repos.
func (m *Manager) List(ctx context.Context) ([]ContainerInfo, error) {
	return ListContainers(ctx, m.client)
}

// ListContainers queries Docker for all zone-managed containers without requiring
// a Manager instance. Used by `zone ls` which operates without a zone.toml.
func ListContainers(ctx context.Context, client DockerClient) ([]ContainerInfo, error) {
	f := filters.NewArgs()
	f.Add("label", "com.zone.managed=true")

	containers, err := client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: f,
	})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	result := make([]ContainerInfo, 0, len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		info := ContainerInfo{
			Name:      name,
			Harness:   c.Labels["com.zone.harness"],
			Status:    c.Status,
			State:     c.State,
			RepoPath:  c.Labels["com.zone.repo-path"],
			ID:        c.ID,
			StartedAt: time.Unix(c.Created, 0),
		}
		result = append(result, info)
	}

	return result, nil
}

// Logs streams container logs to the provided writer. Uses stdcopy.StdCopy for
// non-TTY containers (which is how zone creates all containers).
// When opts.JSON is true, outputs a JSON array of {timestamp, stream, line} objects.
func (m *Manager) Logs(ctx context.Context, w io.Writer, errW io.Writer, opts LogsOpts) error {
	containerID, err := m.cache.ContainerID()
	if err != nil || containerID == "" {
		return ErrNoContainer
	}

	logOpts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     opts.Follow,
		Tail:       opts.Tail,
		Timestamps: true,
	}

	rc, err := m.client.ContainerLogs(ctx, containerID, logOpts)
	if err != nil {
		return fmt.Errorf("get logs: %w", err)
	}
	defer rc.Close()

	if opts.JSON {
		var stdBuf, stderrBuf bytes.Buffer
		if _, copyErr := stdcopy.StdCopy(&stdBuf, &stderrBuf, rc); copyErr != nil {
			return copyErr
		}

		type LogEntry struct {
			Timestamp string `json:"timestamp"`
			Stream    string `json:"stream"`
			Line      string `json:"line"`
		}

		entries := make([]LogEntry, 0)
		for _, pair := range []struct {
			buf    *bytes.Buffer
			stream string
		}{
			{buf: &stdBuf, stream: "stdout"},
			{buf: &stderrBuf, stream: "stderr"},
		} {
			scanner := bufio.NewScanner(pair.buf)
			for scanner.Scan() {
				line := scanner.Text()
				ts, content, _ := strings.Cut(line, " ")
				entries = append(entries, LogEntry{Timestamp: ts, Stream: pair.stream, Line: content})
			}
			if err := scanner.Err(); err != nil {
				return err
			}
		}

		b, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal logs json: %w", err)
		}
		_, _ = fmt.Fprintln(w, string(b))
		return nil
	}

	_, err = stdcopy.StdCopy(w, errW, rc)
	return err
}

// Status returns the full container inspection result for the current repo's container.
func (m *Manager) Status(ctx context.Context) (*container.InspectResponse, error) {
	containerID, err := m.cache.ContainerID()
	if err != nil || containerID == "" {
		return nil, ErrNoContainer
	}

	info, err := m.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}

	return &info, nil
}
