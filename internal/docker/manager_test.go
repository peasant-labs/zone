// manager_test.go tests Manager construction, build pipeline, network helpers,
// and resource parsers using a mock DockerClient (no live Docker daemon required).
package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/errdefs"
	"github.com/docker/go-connections/nat"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/config"
	networkpkg "github.com/peasant-labs/zone/internal/network"
	"github.com/peasant-labs/zone/internal/scaffold"
)

// mockClient is a configurable mock implementation of DockerClient.
// Fields are set before each test to control return values.
type mockClient struct {
	pingErr             error
	pingResp            types.Ping
	infoResp            system.Info
	infoErr             error
	networkCreateID     string
	networkInspectResp  network.Inspect
	networkInspectErr   error
	networkCreateErr    error
	networkRemoveErr    error
	containerCreateResp container.CreateResponse
	containerCreateErr  error
	containerListResp   []container.Summary
	containerListErr    error
	imageBuildResp      types.ImageBuildResponse
	imageBuildErr       error
	imageInspectResp    types.ImageInspect
	imageInspectErr     error
	imageRemoveErr      error
	volumeRemoveErr     error

	// Launch state machine fields
	containerInspectResp container.InspectResponse
	containerInspectErr  error
	containerUnpauseErr  error
	containerRemoveErr   error
	containerStopErr     error

	// Track calls for assertion
	unpauseCalled    bool
	removeCalled     bool
	stopCalled       bool
	startCalled      bool
	startedIDs       []string
	imageRemovedIDs  []string
	volumeRemovedIDs []string

	// Capture container creation arguments for assertions
	lastContainerConfig *container.Config
	lastHostConfig      *container.HostConfig
	lastBuildOptions    types.ImageBuildOptions
}

func (m *mockClient) Ping(ctx context.Context) (types.Ping, error) {
	return m.pingResp, m.pingErr
}

func (m *mockClient) Info(ctx context.Context) (system.Info, error) {
	return m.infoResp, m.infoErr
}

func (m *mockClient) ImageBuild(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	m.lastBuildOptions = options
	return m.imageBuildResp, m.imageBuildErr
}

func (m *mockClient) ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
	return m.imageInspectResp, nil, m.imageInspectErr
}

func (m *mockClient) ImageRemove(ctx context.Context, imageID string, options image.RemoveOptions) ([]image.DeleteResponse, error) {
	m.imageRemovedIDs = append(m.imageRemovedIDs, imageID)
	return nil, m.imageRemoveErr
}

func (m *mockClient) ContainerCreate(ctx context.Context, cfg *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error) {
	m.lastContainerConfig = cfg
	m.lastHostConfig = hostConfig
	return m.containerCreateResp, m.containerCreateErr
}

func (m *mockClient) ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error) {
	return m.containerListResp, m.containerListErr
}

func (m *mockClient) ContainerLogs(ctx context.Context, ctr string, options container.LogsOptions) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (m *mockClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	m.startCalled = true
	m.startedIDs = append(m.startedIDs, containerID)
	return nil
}

func (m *mockClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	m.stopCalled = true
	return m.containerStopErr
}

func (m *mockClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	m.removeCalled = true
	return m.containerRemoveErr
}

func (m *mockClient) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	return m.containerInspectResp, m.containerInspectErr
}

func (m *mockClient) ContainerUnpause(ctx context.Context, containerID string) error {
	m.unpauseCalled = true
	return m.containerUnpauseErr
}

func (m *mockClient) NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
	return network.CreateResponse{ID: m.networkCreateID}, m.networkCreateErr
}

func (m *mockClient) NetworkInspect(ctx context.Context, networkID string, options network.InspectOptions) (network.Inspect, error) {
	return m.networkInspectResp, m.networkInspectErr
}

func (m *mockClient) NetworkRemove(ctx context.Context, networkID string) error {
	return m.networkRemoveErr
}

func (m *mockClient) VolumeCreate(ctx context.Context, options volume.CreateOptions) (volume.Volume, error) {
	return volume.Volume{}, nil
}

func (m *mockClient) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	m.volumeRemovedIDs = append(m.volumeRemovedIDs, volumeID)
	return m.volumeRemoveErr
}

func (m *mockClient) Close() error {
	return nil
}

// newTestManager creates a Manager with a mock client for testing.
func newTestManager(t *testing.T, mc *mockClient, cfg *config.MergedConfig) (*Manager, string) {
	t.Helper()
	tmpDir := t.TempDir()
	c := cache.New(tmpDir)
	m := newManagerWithClient(mc, cfg, c, tmpDir, "test-version")
	// Override attachFn with a no-op so tests don't try to exec docker
	m.attachFn = func(containerID string, cmd []string, asRoot bool) error { return nil }
	return m, tmpDir
}

// newDefaultConfig returns a MergedConfig with sane defaults for testing.
func newDefaultConfig() *config.MergedConfig {
	return &config.MergedConfig{
		Zone: config.ZoneConfig{
			Harness:   "claude-code",
			BaseImage: "ubuntu:24.04",
			Shell:     "bash",
		},
		Resources: config.ResourcesConfig{
			PidsLimit: 512,
		},
		Workspace: config.WorkspaceConfig{
			MountPath: "/workspace",
		},
	}
}

func boolPtr(b bool) *bool { return &b }

// TestNewManagerWithClient verifies that newManagerWithClient populates all Manager fields.
func TestNewManagerWithClient(t *testing.T) {
	mc := &mockClient{}
	cfg := newDefaultConfig()
	tmpDir := t.TempDir()
	c := cache.New(tmpDir)

	m := newManagerWithClient(mc, cfg, c, tmpDir, "v1.2.3")

	assert.NotNil(t, m)
	assert.Equal(t, mc, m.client)
	assert.Equal(t, cfg, m.config)
	assert.Equal(t, c, m.cache)
	assert.Equal(t, "v1.2.3", m.version)
	// repoDir should be absolute
	assert.True(t, filepath.IsAbs(m.repoDir))
}

// TestBuildContext verifies the tar archive structure and file modes.
func TestBuildContext(t *testing.T) {
	dockerfile := "FROM ubuntu:24.04\nRUN echo hello"
	entrypoint := "#!/bin/bash\nexec bash"
	shellrc := "# bashrc"

	reader, err := buildContext(dockerfile, entrypoint, shellrc)
	require.NoError(t, err)

	// Read all tar entries
	tr := tar.NewReader(reader)
	files := map[string]*tar.Header{}
	contents := map[string]string{}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		files[hdr.Name] = hdr
		data, err := io.ReadAll(tr)
		require.NoError(t, err)
		contents[hdr.Name] = string(data)
	}

	// Verify all three files are present
	require.Contains(t, files, "Dockerfile")
	require.Contains(t, files, "entrypoint.sh")
	require.Contains(t, files, "zone-bashrc")

	// Verify contents
	assert.Equal(t, dockerfile, contents["Dockerfile"])
	assert.Equal(t, entrypoint, contents["entrypoint.sh"])
	assert.Equal(t, shellrc, contents["zone-bashrc"])

	// Verify entrypoint.sh has executable mode
	assert.Equal(t, int64(0755), files["entrypoint.sh"].Mode)

	// Verify Dockerfile and zone-bashrc have 0644 mode
	assert.Equal(t, int64(0644), files["Dockerfile"].Mode)
	assert.Equal(t, int64(0644), files["zone-bashrc"].Mode)
}

// TestStreamBuildOutput_Success verifies that stream messages are written and imageID is captured.
func TestStreamBuildOutput_Success(t *testing.T) {
	lines := []string{
		`{"stream":"Step 1/3 : FROM ubuntu:24.04\n"}`,
		`{"stream":"Step 2/3 : RUN echo hello\n"}`,
		`{"aux":{"ID":"sha256:abc123def456"}}`,
	}
	body := io.NopCloser(strings.NewReader(strings.Join(lines, "\n")))

	var buf bytes.Buffer
	imageID, err := streamBuildOutput(body, &buf)

	require.NoError(t, err)
	assert.Equal(t, "sha256:abc123def456", imageID)
	assert.Contains(t, buf.String(), "Step 1/3")
	assert.Contains(t, buf.String(), "Step 2/3")
}

// TestStreamBuildOutput_Error verifies that a build error JSON line returns an error.
func TestStreamBuildOutput_Error(t *testing.T) {
	lines := []string{
		`{"stream":"Step 1/3 : FROM ubuntu:24.04\n"}`,
		`{"error":"The command '/bin/sh -c apt-get install -y badpkg' returned a non-zero code: 100","errorDetail":{"message":"exit code 100"}}`,
	}
	body := io.NopCloser(strings.NewReader(strings.Join(lines, "\n")))

	var buf bytes.Buffer
	imageID, err := streamBuildOutput(body, &buf)

	require.Error(t, err)
	assert.Empty(t, imageID)
	assert.Contains(t, err.Error(), "docker build")
}

// TestParseMemoryBytes verifies memory string parsing.
func TestParseMemoryBytes(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"", 0, false},
		{"0", 0, false},
		{"512m", 536870912, false},
		{"2g", 2147483648, false},
		{"1024k", 1048576, false},
		{"invalid", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseMemoryBytes(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, got)
			}
		})
	}
}

// TestParseNanoCPUs verifies CPU string parsing.
func TestParseNanoCPUs(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"", 0, false},
		{"0", 0, false},
		{"0.5", 500000000, false},
		{"2", 2000000000, false},
		{"1.5", 1500000000, false},
		{"notanumber", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseNanoCPUs(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, got)
			}
		})
	}
}

// TestHomeVolumeName verifies deterministic naming from repo path.
func TestHomeVolumeName(t *testing.T) {
	// Same path always produces the same name
	name1 := homeVolumeName("/home/user/project")
	name2 := homeVolumeName("/home/user/project")
	assert.Equal(t, name1, name2)

	// Different paths produce different names
	name3 := homeVolumeName("/home/user/other")
	assert.NotEqual(t, name1, name3)

	// Format: zone-home-<16hex>
	assert.True(t, strings.HasPrefix(name1, "zone-home-"))
	suffix := strings.TrimPrefix(name1, "zone-home-")
	assert.Len(t, suffix, 16, "short hash should be 16 hex chars")
}

// TestBuildMounts_PersistHomeDefault verifies that nil *bool (default) includes home volume mount.
func TestBuildMounts_PersistHomeDefault(t *testing.T) {
	mc := &mockClient{}
	cfg := newDefaultConfig()
	cfg.Workspace.PersistHome = nil // default = true
	// Disable MountHomeConfig to get a predictable mount count (2: workspace + home volume)
	disabled := false
	cfg.Auth.MountHomeConfig = &disabled

	m, repoDir := newTestManager(t, mc, cfg)

	mounts := m.buildMounts()

	require.Len(t, mounts, 2, "should have workspace bind mount + home volume mount")

	// Workspace bind mount
	assert.Equal(t, "/workspace", mounts[0].Target)

	// Home volume mount
	assert.Equal(t, "/home/zone", mounts[1].Target)

	// Verify volume name is deterministic
	expectedVolName := homeVolumeName(repoDir)
	assert.Equal(t, expectedVolName, mounts[1].Source)
}

// TestBuildMounts_PersistHomeFalse verifies that persist_home=false skips the home volume.
func TestBuildMounts_PersistHomeFalse(t *testing.T) {
	mc := &mockClient{}
	cfg := newDefaultConfig()
	f := false
	cfg.Workspace.PersistHome = &f
	// Disable MountHomeConfig to get a predictable mount count (1: workspace only)
	cfg.Auth.MountHomeConfig = &f

	m, _ := newTestManager(t, mc, cfg)

	mounts := m.buildMounts()

	require.Len(t, mounts, 1, "should only have workspace bind mount")
	assert.Equal(t, "/workspace", mounts[0].Target)
}

// TestCreateNetwork verifies that createNetwork calls the client and returns the network ID.
func TestCreateNetwork(t *testing.T) {
	mc := &mockClient{
		networkCreateID: "net-abc123",
	}
	cfg := newDefaultConfig()
	m, _ := newTestManager(t, mc, cfg)

	netID, err := m.createNetwork(context.Background(), "test-network")

	require.NoError(t, err)
	assert.Equal(t, "net-abc123", netID)
}

// TestCreateNetwork_AlreadyExists verifies that createNetwork reuses an existing network.
func TestCreateNetwork_AlreadyExists(t *testing.T) {
	mc := &mockClient{
		networkCreateErr: errors.New("network with name test-network already exists"),
		networkInspectResp: network.Inspect{
			ID: "existing-net-789",
		},
	}
	cfg := newDefaultConfig()
	m, _ := newTestManager(t, mc, cfg)

	netID, err := m.createNetwork(context.Background(), "test-network")

	require.NoError(t, err)
	assert.Equal(t, "existing-net-789", netID)
}

// TestRemoveNetwork_NotFound verifies that a "not found" error from NetworkRemove is swallowed.
func TestRemoveNetwork_NotFound(t *testing.T) {
	// Use a "not found" error from errdefs
	notFoundErr := errdefs.NotFound(errors.New("network not found"))

	mc := &mockClient{
		networkRemoveErr: notFoundErr,
	}
	cfg := newDefaultConfig()
	m, tmpDir := newTestManager(t, mc, cfg)

	// Write a network ID to cache so removeNetwork has something to remove
	require.NoError(t, os.MkdirAll(tmpDir+"/.zone", 0755))
	require.NoError(t, os.WriteFile(tmpDir+"/.zone/network_id", []byte("net-abc123"), 0644))
	// Re-create cache with the tmp dir that has the .zone subdir
	m.cache = cache.New(tmpDir)
	// Set the network ID via cache
	require.NoError(t, m.cache.EnsureDir())
	require.NoError(t, m.cache.SetNetworkID("net-abc123"))

	// removeNetwork should succeed even though NetworkRemove returns "not found"
	err := m.removeNetwork(context.Background())
	assert.NoError(t, err)
}

// TestRemoveNetwork_OtherError verifies that non-"not found" errors from NetworkRemove are propagated.
func TestRemoveNetwork_OtherError(t *testing.T) {
	otherErr := errors.New("permission denied")
	mc := &mockClient{
		networkRemoveErr: otherErr,
	}
	cfg := newDefaultConfig()
	m, _ := newTestManager(t, mc, cfg)

	require.NoError(t, m.cache.EnsureDir())
	require.NoError(t, m.cache.SetNetworkID("net-abc123"))

	err := m.removeNetwork(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "remove network")
}

func TestStaleRuleCleanupOnLaunch(t *testing.T) {
	mc := &mockClient{
		containerListResp: []container.Summary{
			{Names: []string{"/zone-project-abc1234567890def"}},
		},
	}
	m := newManagerWithClient(mc, newDefaultConfig(), cache.New(t.TempDir()), t.TempDir(), "test-version")

	runningHashes, err := m.listRunningZoneHashes(context.Background())
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"abc1234567890def": true}, runningHashes)

	var calls [][]string
	execFn := func(ctx context.Context, args ...string) ([]byte, error) {
		copied := append([]string(nil), args...)
		calls = append(calls, copied)
		if len(args) == 1 && args[0] == "-S" {
			return []byte("-A FORWARD -i br-x -d 1.2.3.4 -j ACCEPT -m comment --comment zone-abc1234567890def\n" +
				"-A FORWARD -i br-x -d 5.6.7.8 -j DROP -m comment --comment zone-deadbeefdeadbeef\n"), nil
		}
		return nil, nil
	}

	err = networkpkg.CleanStaleRules(context.Background(), execFn, runningHashes)
	require.NoError(t, err)

	var deleteCalls [][]string
	for _, call := range calls {
		if len(call) > 0 && call[0] == "-D" {
			deleteCalls = append(deleteCalls, call)
		}
	}
	require.Len(t, deleteCalls, 1)
	assert.Equal(t, []string{"-D", "FORWARD", "-i", "br-x", "-d", "5.6.7.8", "-j", "DROP", "-m", "comment", "--comment", "zone-deadbeefdeadbeef"}, deleteCalls[0])
}

// --- Launch State Machine Tests ---

// makeLaunchMock sets up a mockClient and Manager for Launch tests.
// The mock image build returns a synthetic image ID; ContainerCreate returns a synthetic container ID.
// Sets ANTHROPIC_API_KEY so the pre-launch env validation passes for the claude-code harness.
func makeLaunchMock(t *testing.T, status string, oomKilled bool) (*mockClient, *Manager, string) {
	t.Helper()

	// Satisfy claude-code required env var validation
	t.Setenv("ANTHROPIC_API_KEY", "test-key-for-launch-tests")

	buildJSON := `{"aux":{"ID":"sha256:testimage123"}}` + "\n"
	mc := &mockClient{
		imageBuildResp: types.ImageBuildResponse{
			Body: io.NopCloser(strings.NewReader(buildJSON)),
		},
		imageInspectResp:    types.ImageInspect{ID: "sha256:testimage123"},
		containerCreateResp: container.CreateResponse{ID: "container-abc"},
		networkCreateID:     "net-xyz",
		containerInspectResp: container.InspectResponse{
			ContainerJSONBase: &container.ContainerJSONBase{
				ID: "container-abc",
				State: &container.State{
					Status:    status,
					OOMKilled: oomKilled,
				},
			},
		},
	}

	cfg := newDefaultConfig()
	// Disable MountHomeConfig to avoid side-effects from real ~/.claude presence
	disabled := false
	cfg.Auth.MountHomeConfig = &disabled
	m, tmpDir := newTestManager(t, mc, cfg)

	// Prime cache so EnsureDir works for lock
	require.NoError(t, m.cache.EnsureDir())

	return mc, m, tmpDir
}

// TestLaunchStateMachine_Fresh verifies that a fresh launch (no container_id in cache)
// triggers a full build + create + start sequence.
func TestLaunchStateMachine_Fresh(t *testing.T) {
	mc, m, _ := makeLaunchMock(t, "", false)

	err := m.Launch(context.Background(), LaunchOpts{})
	require.NoError(t, err)

	// ContainerStart should have been called (build+create+start path)
	assert.True(t, mc.startCalled, "ContainerStart should be called for fresh launch")
	// ContainerRemove should NOT have been called (no existing container to clean up)
	assert.False(t, mc.removeCalled, "ContainerRemove should NOT be called for fresh launch")
}

// TestLaunchStateMachine_Running verifies that a running container triggers reattach, not rebuild.
func TestLaunchStateMachine_Running(t *testing.T) {
	mc, m, _ := makeLaunchMock(t, "running", false)

	// Prime cache with a container ID
	require.NoError(t, m.cache.SetContainerID("container-abc"))

	// Prime config hash to match so no "stale config" warning
	hash, err := computeTestHash(m)
	require.NoError(t, err)
	require.NoError(t, m.cache.SetConfigHash(hash))

	var attachedID string
	m.attachFn = func(containerID string, cmd []string, asRoot bool) error {
		attachedID = containerID
		return nil
	}

	err = m.Launch(context.Background(), LaunchOpts{})
	require.NoError(t, err)

	// Should have reattached to the existing container, not created a new one
	assert.Equal(t, "container-abc", attachedID)
	assert.False(t, mc.startCalled, "ContainerStart should NOT be called for running container")
}

// TestLaunchStateMachine_RunningStaleConfig verifies warning printed when running container
// has a different config hash than the current one.
func TestLaunchStateMachine_RunningStaleConfig(t *testing.T) {
	_, m, _ := makeLaunchMock(t, "running", false)

	require.NoError(t, m.cache.SetContainerID("container-abc"))
	// Write a deliberately different (stale) hash
	require.NoError(t, m.cache.SetConfigHash("stale-hash-000"))

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := m.Launch(context.Background(), LaunchOpts{})

	w.Close()
	os.Stderr = oldStderr
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Config has changed")
}

// TestLaunchStateMachine_Paused verifies that a paused container is unpaused then attached.
func TestLaunchStateMachine_Paused(t *testing.T) {
	mc, m, _ := makeLaunchMock(t, "paused", false)

	require.NoError(t, m.cache.SetContainerID("container-abc"))

	var attachedID string
	m.attachFn = func(containerID string, cmd []string, asRoot bool) error {
		attachedID = containerID
		return nil
	}

	err := m.Launch(context.Background(), LaunchOpts{})
	require.NoError(t, err)

	assert.True(t, mc.unpauseCalled, "ContainerUnpause should be called for paused container")
	assert.Equal(t, "container-abc", attachedID)
}

// TestLaunchStateMachine_Exited verifies that an exited container is removed then rebuilt.
func TestLaunchStateMachine_Exited(t *testing.T) {
	mc, m, _ := makeLaunchMock(t, "exited", false)

	require.NoError(t, m.cache.SetContainerID("container-abc"))

	err := m.Launch(context.Background(), LaunchOpts{})
	require.NoError(t, err)

	assert.True(t, mc.removeCalled, "ContainerRemove should be called for exited container")
	assert.True(t, mc.startCalled, "ContainerStart should be called after rebuild")
}

// TestLaunchStateMachine_ExitedOOM verifies OOM kill warning printed to stderr.
func TestLaunchStateMachine_ExitedOOM(t *testing.T) {
	_, m, _ := makeLaunchMock(t, "exited", true)

	require.NoError(t, m.cache.SetContainerID("container-abc"))

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := m.Launch(context.Background(), LaunchOpts{})

	w.Close()
	os.Stderr = oldStderr
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "memory limit")
}

// TestLaunchStateMachine_StaleID verifies that a "not found" inspect response causes
// the cache to be cleaned and a fresh build to proceed.
func TestLaunchStateMachine_StaleID(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key-stale")

	buildJSON := `{"aux":{"ID":"sha256:testimage123"}}` + "\n"
	mc := &mockClient{
		imageBuildResp: types.ImageBuildResponse{
			Body: io.NopCloser(strings.NewReader(buildJSON)),
		},
		imageInspectResp:    types.ImageInspect{ID: "sha256:testimage123"},
		containerCreateResp: container.CreateResponse{ID: "container-new"},
		networkCreateID:     "net-xyz",
		// ContainerInspect returns NotFound (stale ID)
		containerInspectErr: errdefs.NotFound(errors.New("no such container")),
	}

	cfg := newDefaultConfig()
	disabled := false
	cfg.Auth.MountHomeConfig = &disabled
	m, _ := newTestManager(t, mc, cfg)
	require.NoError(t, m.cache.EnsureDir())
	require.NoError(t, m.cache.SetContainerID("stale-container-id"))

	err := m.Launch(context.Background(), LaunchOpts{})
	require.NoError(t, err)

	assert.True(t, mc.startCalled, "ContainerStart should be called after stale cache clean")

	// Container ID should now be the new one
	storedID, _ := m.cache.ContainerID()
	assert.Equal(t, "container-new", storedID)
}

// TestLaunchHeadless verifies that headless mode prints the container ID to stdout
// and does NOT call attachFn.
func TestLaunchHeadless(t *testing.T) {
	_, m, _ := makeLaunchMock(t, "", false)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	attachCalled := false
	m.attachFn = func(containerID string, cmd []string, asRoot bool) error {
		attachCalled = true
		return nil
	}

	err := m.Launch(context.Background(), LaunchOpts{Headless: true})

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "container-abc")
	assert.False(t, attachCalled, "attachFn should NOT be called in headless mode")
}

// TestConfigHashDetection_AutoRebuild verifies that when there's no running container
// and the hash has changed, a silent rebuild is triggered.
func TestConfigHashDetection_AutoRebuild(t *testing.T) {
	mc, m, _ := makeLaunchMock(t, "", false)

	// Set a stale hash (no container ID — fresh launch path)
	require.NoError(t, m.cache.SetConfigHash("old-hash-doesnt-match-current"))

	err := m.Launch(context.Background(), LaunchOpts{})
	require.NoError(t, err)

	// Build should have run (ContainerStart was called)
	assert.True(t, mc.startCalled, "should have rebuilt and started container")
}

// --- Zero-config quickstart test ---

// TestGenerateMinimalZoneToml verifies the generated zone.toml content.
func TestGenerateMinimalZoneToml(t *testing.T) {
	result := generateMinimalZoneToml("claude-code")

	assert.Contains(t, result, "version = 1")
	assert.Contains(t, result, `harness = "claude-code"`)
	assert.Contains(t, result, "# Uncomment to customize:")
}

func TestQuickstartWriteZoneTomlCreatesAgentSkill(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, QuickstartWriteZoneToml(dir, "codex-cli"))

	data, err := os.ReadFile(filepath.Join(dir, scaffold.AgentSkillsDir, scaffold.AgentZoneSkillFile))
	require.NoError(t, err)
	assert.Contains(t, string(data), "Zone Workspace Dependencies")
	assert.Contains(t, string(data), "`zone.toml`")
}

func TestHarnessCmdCodexInteractive(t *testing.T) {
	cfg := newDefaultConfig()
	cfg.Zone.Harness = "codex-cli"
	m, _ := newTestManager(t, &mockClient{}, cfg)

	got := m.harnessCmd(LaunchOpts{})

	assert.Equal(t, []string{"codex"}, got)
}

func TestHarnessCmdCodexPromptUsesExec(t *testing.T) {
	cfg := newDefaultConfig()
	cfg.Zone.Harness = "codex-cli"
	m, _ := newTestManager(t, &mockClient{}, cfg)

	got := m.harnessCmd(LaunchOpts{Prompt: "fix the tests"})

	assert.Equal(t, []string{"codex", "exec", "fix the tests"}, got)
}

func TestHarnessCmdCodexDangerouslyBypassApprovalsAndSandbox(t *testing.T) {
	cfg := newDefaultConfig()
	cfg.Zone.Harness = "codex-cli"
	cfg.Harness.SkipPermissions = boolPtr(true)
	m, _ := newTestManager(t, &mockClient{}, cfg)

	got := m.harnessCmd(LaunchOpts{
		Prompt:      "fix the tests",
		HarnessArgs: []string{"--model", "gpt-5.3-codex"},
	})

	assert.Equal(t, []string{
		"codex",
		"exec",
		"--dangerously-bypass-approvals-and-sandbox",
		"--model",
		"gpt-5.3-codex",
		"fix the tests",
	}, got)
}

func TestHarnessCmdOpenCodeInteractive(t *testing.T) {
	cfg := newDefaultConfig()
	cfg.Zone.Harness = "opencode"
	m, _ := newTestManager(t, &mockClient{}, cfg)

	got := m.harnessCmd(LaunchOpts{})

	assert.Equal(t, []string{"opencode"}, got)
}

func TestHarnessCmdOpenCodePromptUsesPromptFlag(t *testing.T) {
	cfg := newDefaultConfig()
	cfg.Zone.Harness = "opencode"
	m, _ := newTestManager(t, &mockClient{}, cfg)

	got := m.harnessCmd(LaunchOpts{Prompt: "fix the tests"})

	assert.Equal(t, []string{"opencode", "--prompt", "fix the tests"}, got)
}

func TestHarnessCmdOpenCodeDangerouslySkipPermissions(t *testing.T) {
	cfg := newDefaultConfig()
	cfg.Zone.Harness = "opencode"
	cfg.Harness.SkipPermissions = boolPtr(true)
	m, _ := newTestManager(t, &mockClient{}, cfg)

	got := m.harnessCmd(LaunchOpts{
		Prompt:      "fix the tests",
		HarnessArgs: []string{"--model", "anthropic/claude-sonnet-4-20250514"},
	})

	assert.Equal(t, []string{
		"opencode",
		"--dangerously-skip-permissions",
		"--model",
		"anthropic/claude-sonnet-4-20250514",
		"--prompt",
		"fix the tests",
	}, got)
}

func TestHarnessCmdAppendsConfiguredExtraArgsBeforeCLIArgs(t *testing.T) {
	cfg := newDefaultConfig()
	cfg.Zone.Harness = "claude-code"
	cfg.Harness.ExtraArgs = []string{"--verbose"}
	m, _ := newTestManager(t, &mockClient{}, cfg)

	got := m.harnessCmd(LaunchOpts{
		Prompt:      "fix the tests",
		HarnessArgs: []string{"--model", "sonnet"},
	})

	assert.Equal(t, []string{
		"claude",
		"-p",
		"fix the tests",
		"--verbose",
		"--model",
		"sonnet",
	}, got)
}

// computeTestHash is a helper to get the current config hash for a manager.
func computeTestHash(m *Manager) (string, error) {
	return cache.ComputeHash(m.config, m.version)
}

// --- Stop / Destroy / RemoveImage Tests ---

// TestStop_RunningContainer verifies that Stop calls ContainerStop, ContainerRemove,
// removeNetwork, clears container_id and network_id, and retains image_id.
func TestStop_RunningContainer(t *testing.T) {
	mc := &mockClient{}
	cfg := newDefaultConfig()
	m, _ := newTestManager(t, mc, cfg)

	require.NoError(t, m.cache.EnsureDir())
	require.NoError(t, m.cache.SetContainerID("container-abc"))
	require.NoError(t, m.cache.SetNetworkID("net-xyz"))
	require.NoError(t, m.cache.SetImageID("sha256:imageabc"))

	err := m.Stop(context.Background())
	require.NoError(t, err)

	// Verify ContainerStop and ContainerRemove were called
	assert.True(t, mc.stopCalled, "ContainerStop should be called")
	assert.True(t, mc.removeCalled, "ContainerRemove should be called")

	// Verify container_id and network_id are cleared
	cid, _ := m.cache.ContainerID()
	assert.Empty(t, cid, "container_id should be cleared after Stop")
	nid, _ := m.cache.NetworkID()
	assert.Empty(t, nid, "network_id should be cleared after Stop")

	// Verify image_id is NOT cleared
	iid, _ := m.cache.ImageID()
	assert.Equal(t, "sha256:imageabc", iid, "image_id should be retained after Stop")
}

// TestStop_NoContainer verifies that Stop is a no-op when container_id is empty.
func TestStop_NoContainer(t *testing.T) {
	mc := &mockClient{}
	cfg := newDefaultConfig()
	m, _ := newTestManager(t, mc, cfg)

	require.NoError(t, m.cache.EnsureDir())
	// No container_id set

	err := m.Stop(context.Background())
	require.NoError(t, err)

	// Verify no Docker client calls were made
	assert.False(t, mc.stopCalled, "ContainerStop should NOT be called when no container")
	assert.False(t, mc.removeCalled, "ContainerRemove should NOT be called when no container")
}

// TestStop_ContainerNotFound verifies that a NotFound error from ContainerStop is swallowed,
// Stop still returns nil and clears the cache.
func TestStop_ContainerNotFound(t *testing.T) {
	mc := &mockClient{
		containerStopErr: errdefs.NotFound(errors.New("no such container")),
	}
	cfg := newDefaultConfig()
	m, _ := newTestManager(t, mc, cfg)

	require.NoError(t, m.cache.EnsureDir())
	require.NoError(t, m.cache.SetContainerID("container-ghost"))
	require.NoError(t, m.cache.SetNetworkID("net-xyz"))

	err := m.Stop(context.Background())
	require.NoError(t, err, "Stop should return nil when ContainerStop returns NotFound")

	// Cache should still be cleared
	cid, _ := m.cache.ContainerID()
	assert.Empty(t, cid, "container_id should be cleared even when container was not found")
}

// TestDestroy_Full verifies that Destroy calls Stop, ImageRemove, VolumeRemove, and cache.Clean().
func TestDestroy_Full(t *testing.T) {
	mc := &mockClient{}
	cfg := newDefaultConfig()
	m, _ := newTestManager(t, mc, cfg)

	require.NoError(t, m.cache.EnsureDir())
	require.NoError(t, m.cache.SetContainerID("container-abc"))
	require.NoError(t, m.cache.SetNetworkID("net-xyz"))
	require.NoError(t, m.cache.SetImageID("sha256:imageabc"))

	err := m.Destroy(context.Background())
	require.NoError(t, err)

	// Verify Stop was called (container removed)
	assert.True(t, mc.stopCalled, "ContainerStop should be called by Destroy")
	assert.True(t, mc.removeCalled, "ContainerRemove should be called by Destroy")

	// Verify ImageRemove was called with correct image ID
	require.Len(t, mc.imageRemovedIDs, 1, "ImageRemove should be called once")
	assert.Equal(t, "sha256:imageabc", mc.imageRemovedIDs[0])

	// Verify VolumeRemove was called with zone-home-<hash>
	require.Len(t, mc.volumeRemovedIDs, 1, "VolumeRemove should be called once")
	assert.True(t, strings.HasPrefix(mc.volumeRemovedIDs[0], "zone-home-"),
		"VolumeRemove should be called with zone-home-<hash> volume name")

	// Verify cache was cleaned (cache dir should not exist)
	_, err = os.Stat(m.cache.Dir())
	assert.True(t, os.IsNotExist(err), "cache directory should be removed after Destroy")
}

// TestDestroy_NoContainer verifies that Destroy still removes image, volume, and cache
// even when no container is running.
func TestDestroy_NoContainer(t *testing.T) {
	mc := &mockClient{}
	cfg := newDefaultConfig()
	m, _ := newTestManager(t, mc, cfg)

	require.NoError(t, m.cache.EnsureDir())
	// No container_id — but image_id is set
	require.NoError(t, m.cache.SetImageID("sha256:imageabc"))

	err := m.Destroy(context.Background())
	require.NoError(t, err)

	// ContainerStop should NOT be called (no container)
	assert.False(t, mc.stopCalled, "ContainerStop should NOT be called when no container")

	// ImageRemove should still be called
	require.Len(t, mc.imageRemovedIDs, 1, "ImageRemove should be called even without container")
	assert.Equal(t, "sha256:imageabc", mc.imageRemovedIDs[0])

	// VolumeRemove should still be called
	require.Len(t, mc.volumeRemovedIDs, 1, "VolumeRemove should be called even without container")

	// Cache dir removed
	_, err = os.Stat(m.cache.Dir())
	assert.True(t, os.IsNotExist(err), "cache directory should be removed after Destroy")
}

// TestDestroyVsStop_VolumeRetention verifies that Stop does NOT call VolumeRemove
// but Destroy DOES call VolumeRemove.
func TestDestroyVsStop_VolumeRetention(t *testing.T) {
	// Test Stop: volume NOT removed
	t.Run("Stop does not remove volume", func(t *testing.T) {
		mc := &mockClient{}
		cfg := newDefaultConfig()
		m, _ := newTestManager(t, mc, cfg)
		require.NoError(t, m.cache.EnsureDir())
		require.NoError(t, m.cache.SetContainerID("container-abc"))

		err := m.Stop(context.Background())
		require.NoError(t, err)

		assert.Empty(t, mc.volumeRemovedIDs, "Stop should NOT call VolumeRemove")
	})

	// Test Destroy: volume IS removed
	t.Run("Destroy removes volume", func(t *testing.T) {
		mc := &mockClient{}
		cfg := newDefaultConfig()
		m, _ := newTestManager(t, mc, cfg)
		require.NoError(t, m.cache.EnsureDir())
		require.NoError(t, m.cache.SetContainerID("container-abc"))

		err := m.Destroy(context.Background())
		require.NoError(t, err)

		assert.NotEmpty(t, mc.volumeRemovedIDs, "Destroy should call VolumeRemove")
	})
}

// TestRemoveImage verifies that RemoveImage calls ImageRemove with the cached image ID
// and clears the image_id from cache afterward.
func TestRemoveImage(t *testing.T) {
	mc := &mockClient{}
	cfg := newDefaultConfig()
	m, _ := newTestManager(t, mc, cfg)

	require.NoError(t, m.cache.EnsureDir())
	require.NoError(t, m.cache.SetImageID("sha256:removetest"))

	err := m.RemoveImage(context.Background())
	require.NoError(t, err)

	// Verify ImageRemove was called with the correct image ID
	require.Len(t, mc.imageRemovedIDs, 1, "ImageRemove should be called once")
	assert.Equal(t, "sha256:removetest", mc.imageRemovedIDs[0])

	// Verify image_id is cleared from cache
	iid, _ := m.cache.ImageID()
	assert.Empty(t, iid, "image_id should be cleared after RemoveImage")
}

// TestRemoveImage_NoImage verifies that RemoveImage is a no-op when image_id is empty.
func TestRemoveImage_NoImage(t *testing.T) {
	mc := &mockClient{}
	cfg := newDefaultConfig()
	m, _ := newTestManager(t, mc, cfg)

	require.NoError(t, m.cache.EnsureDir())
	// No image_id set

	err := m.RemoveImage(context.Background())
	require.NoError(t, err)

	assert.Empty(t, mc.imageRemovedIDs, "ImageRemove should NOT be called when no image_id")
}

// --- Phase 7 Integration Tests: buildMounts SSH + Auth Config ---

// TestBuildMounts_SSHAgent verifies that ForwardSSHAgent=true adds the SSH socket bind mount
// on Linux when SSH_AUTH_SOCK points to a real socket.
func TestBuildMounts_SSHAgent(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("SSH agent socket bind-mount is not supported on macOS")
	}

	// Create a real Unix socket to satisfy os.Stat + ModeSocket check
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "ssh-agent.sock")

	// Create an actual listener on the socket path so the file has ModeSocket type
	ln, err := createUnixSocket(sockPath)
	require.NoError(t, err, "failed to create test Unix socket")
	defer ln.Close()

	t.Setenv("SSH_AUTH_SOCK", sockPath)

	mc := &mockClient{}
	cfg := newDefaultConfig()
	fwdSSH := true
	cfg.Auth.ForwardSSHAgent = &fwdSSH

	m, _ := newTestManager(t, mc, cfg)
	mounts := m.buildMounts()

	var sshMount *mount.Mount
	for _, mt := range mounts {
		if mt.Target == "/tmp/ssh-agent.sock" {
			cp := mt
			sshMount = &cp
			break
		}
	}
	require.NotNil(t, sshMount, "expected SSH agent mount at /tmp/ssh-agent.sock")
	assert.Equal(t, sockPath, sshMount.Source)
	assert.True(t, sshMount.ReadOnly, "SSH agent mount should be read-only")
}

// TestBuildMounts_SSHAgent_NoSocket verifies that when SSH_AUTH_SOCK is unset,
// no SSH agent mount is added.
func TestBuildMounts_SSHAgent_NoSocket(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("SSH agent socket bind-mount is not supported on macOS")
	}

	// Ensure SSH_AUTH_SOCK is unset
	t.Setenv("SSH_AUTH_SOCK", "")

	mc := &mockClient{}
	cfg := newDefaultConfig()
	fwdSSH := true
	cfg.Auth.ForwardSSHAgent = &fwdSSH

	m, _ := newTestManager(t, mc, cfg)
	mounts := m.buildMounts()

	for _, mt := range mounts {
		assert.NotEqual(t, "/tmp/ssh-agent.sock", mt.Target, "should not mount SSH socket when SSH_AUTH_SOCK is unset")
	}
}

// TestBuildMounts_AuthConfig verifies that MountHomeConfig=true (default) adds
// auth config mounts with ".host" suffix for the claude-code harness.
func TestBuildMounts_AuthConfig(t *testing.T) {
	// The claude-code harness HomeConfigDir is "~/.claude".
	// We need to ensure the expanded path exists for the stat check.
	// Use a temp dir and override the HOME env var so expandHome returns our tmpDir.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create the ~/.claude directory inside our temp home
	claudeDir := filepath.Join(tmpDir, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	mc := &mockClient{}
	cfg := newDefaultConfig() // uses claude-code harness
	mountHome := true
	cfg.Auth.MountHomeConfig = &mountHome

	m, _ := newTestManager(t, mc, cfg)
	mounts := m.buildMounts()

	var authMount *mount.Mount
	for _, mt := range mounts {
		if strings.HasSuffix(mt.Target, ".host") {
			cp := mt
			authMount = &cp
			break
		}
	}
	require.NotNil(t, authMount, "expected auth config mount with .host suffix")
	assert.Equal(t, claudeDir, authMount.Source, "source should be expanded ~/.claude")
	assert.Equal(t, "/home/zone/.claude.host", authMount.Target, "target should be absolute container path")
	assert.True(t, authMount.ReadOnly, "auth config mount should be read-only")
}

// TestBuildMounts_AuthConfig_Disabled verifies that MountHomeConfig=false
// produces no auth config mounts.
func TestBuildMounts_AuthConfig_Disabled(t *testing.T) {
	mc := &mockClient{}
	cfg := newDefaultConfig()
	disabled := false
	cfg.Auth.MountHomeConfig = &disabled

	m, _ := newTestManager(t, mc, cfg)
	mounts := m.buildMounts()

	for _, mt := range mounts {
		assert.False(t, strings.HasSuffix(mt.Target, ".host"),
			"no auth config mounts expected when MountHomeConfig=false")
	}
}

// --- Phase 7 Integration Tests: createContainer env vars and ports ---

// makeLaunchMockForCreate sets up a mock suitable for createContainer tests.
// It provides a valid build response and an image inspect so the full flow succeeds.
func makeLaunchMockForCreate(t *testing.T) (*mockClient, *Manager) {
	t.Helper()
	buildJSON := `{"aux":{"ID":"sha256:testimage123"}}` + "\n"
	mc := &mockClient{
		imageBuildResp: types.ImageBuildResponse{
			Body: io.NopCloser(strings.NewReader(buildJSON)),
		},
		imageInspectResp:    types.ImageInspect{ID: "sha256:testimage123"},
		containerCreateResp: container.CreateResponse{ID: "container-abc"},
		networkCreateID:     "net-xyz",
	}
	cfg := newDefaultConfig()
	m, _ := newTestManager(t, mc, cfg)
	require.NoError(t, m.cache.EnsureDir())
	return mc, m
}

// TestCreateContainer_EnvVars verifies that ForwardEnv patterns result in matching
// host env vars appearing in container.Config.Env.
func TestCreateContainer_EnvVars(t *testing.T) {
	t.Setenv("TEST_ZONE_VAR", "hello-zone")

	mc, m := makeLaunchMockForCreate(t)
	m.config.Auth.ForwardEnv = []string{"TEST_ZONE_VAR"}

	// Trigger createContainer via a full launch flow (fresh container)
	require.NoError(t, m.cache.SetImageID("sha256:testimage123"))
	_, err := m.createContainer(context.Background(), "sha256:testimage123")
	require.NoError(t, err)

	require.NotNil(t, mc.lastContainerConfig)
	assert.Contains(t, mc.lastContainerConfig.Env, "TEST_ZONE_VAR=hello-zone",
		"forwarded env var should appear in container Config.Env")
}

// TestCreateContainer_Ports verifies that Workspace.Ports entries produce
// correct HostConfig.PortBindings and Config.ExposedPorts.
func TestCreateContainer_Ports(t *testing.T) {
	mc, m := makeLaunchMockForCreate(t)
	m.config.Workspace.Ports = []string{"3000:3000"}

	require.NoError(t, m.cache.SetImageID("sha256:testimage123"))
	_, err := m.createContainer(context.Background(), "sha256:testimage123")
	require.NoError(t, err)

	require.NotNil(t, mc.lastContainerConfig, "container config should be captured")
	require.NotNil(t, mc.lastHostConfig, "host config should be captured")

	// Verify ExposedPorts contains 3000/tcp
	port3000, _ := nat.NewPort("tcp", "3000")
	assert.Contains(t, mc.lastContainerConfig.ExposedPorts, port3000,
		"ExposedPorts should contain 3000/tcp")

	// Verify PortBindings maps 3000/tcp -> host port 3000
	bindings, ok := mc.lastHostConfig.PortBindings[port3000]
	require.True(t, ok, "PortBindings should contain 3000/tcp")
	require.Len(t, bindings, 1)
	assert.Equal(t, "3000", bindings[0].HostPort)
}

// TestCreateContainer_EnvFile verifies that Auth.EnvFile vars appear in Config.Env.
func TestCreateContainer_EnvFile(t *testing.T) {
	// Create a temp .env file
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	require.NoError(t, os.WriteFile(envPath, []byte("MY_SECRET=abc123\nOTHER_VAR=value\n"), 0644))

	mc, m := makeLaunchMockForCreate(t)
	m.config.Auth.EnvFile = envPath // absolute path

	require.NoError(t, m.cache.SetImageID("sha256:testimage123"))
	_, err := m.createContainer(context.Background(), "sha256:testimage123")
	require.NoError(t, err)

	require.NotNil(t, mc.lastContainerConfig)
	env := mc.lastContainerConfig.Env
	assert.Contains(t, env, "MY_SECRET=abc123", "env file var MY_SECRET should be in container env")
	assert.Contains(t, env, "OTHER_VAR=value", "env file var OTHER_VAR should be in container env")
}

// createUnixSocket creates a Unix domain socket at path and returns the net.Listener.
// Used in SSH agent tests to create a real socket file that satisfies os.ModeSocket check.
func createUnixSocket(path string) (interface{ Close() error }, error) {
	return net.Listen("unix", path)
}

// --- Phase 7 Task 2 Tests: Launch validation, hooks, buildImage proxy args ---

// makeLaunchMockWithAPIKey sets up a full Launch mock that satisfies the claude-code
// harness required env var check (ANTHROPIC_API_KEY must be set in host env).
func makeLaunchMockWithAPIKey(t *testing.T) (*mockClient, *Manager) {
	t.Helper()
	// Build a fresh imageBuildResp each time so the Body reader isn't exhausted
	buildJSON := `{"aux":{"ID":"sha256:testimage123"}}` + "\n"
	mc := &mockClient{
		imageBuildResp: types.ImageBuildResponse{
			Body: io.NopCloser(strings.NewReader(buildJSON)),
		},
		imageInspectResp:    types.ImageInspect{ID: "sha256:testimage123"},
		containerCreateResp: container.CreateResponse{ID: "container-abc"},
		networkCreateID:     "net-xyz",
	}
	cfg := newDefaultConfig()
	// Disable MountHomeConfig to avoid side-effects from real ~/.claude presence
	disabled := false
	cfg.Auth.MountHomeConfig = &disabled
	m, _ := newTestManager(t, mc, cfg)
	require.NoError(t, m.cache.EnsureDir())
	return mc, m
}

// TestLaunch_RequiredEnvValidation verifies that Launch fails with a descriptive error
// when ANTHROPIC_API_KEY is not set (claude-code harness).
func TestLaunch_RequiredEnvValidation(t *testing.T) {
	_, m := makeLaunchMockWithAPIKey(t)
	m.config.Zone.Harness = "custom"
	m.config.Harness.EntrypointCommand = "custom-agent"
	m.config.Harness.RequiredEnv = []string{"ANTHROPIC_API_KEY"}

	// Ensure the required key is completely absent (not just empty)
	t.Setenv("ANTHROPIC_API_KEY", "")
	require.NoError(t, os.Unsetenv("ANTHROPIC_API_KEY"))
	t.Cleanup(func() {
		// t.Setenv already registered cleanup to restore the prior value
	})

	err := m.Launch(context.Background(), LaunchOpts{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ANTHROPIC_API_KEY",
		"error should name the missing required env var")
}

// TestLaunch_RequiredEnvValidation_Satisfied verifies that Launch proceeds past validation
// when ANTHROPIC_API_KEY is set.
func TestLaunch_RequiredEnvValidation_Satisfied(t *testing.T) {
	_, m := makeLaunchMockWithAPIKey(t)
	m.config.Zone.Harness = "custom"
	m.config.Harness.EntrypointCommand = "custom-agent"
	m.config.Harness.RequiredEnv = []string{"ANTHROPIC_API_KEY"}

	t.Setenv("ANTHROPIC_API_KEY", "test-key-satisfied")

	// Launch should NOT fail at the validation step.
	// It may fail at a subsequent step (the mock doesn't return a real image ID from
	// aux), but the error must not contain "required environment variable".
	err := m.Launch(context.Background(), LaunchOpts{})
	if err != nil {
		assert.NotContains(t, err.Error(), "required environment variable",
			"error should not be a validation error when ANTHROPIC_API_KEY is set")
	}
}

// TestLaunch_PreBuildHook verifies that a successful pre_build hook does not block Launch.
func TestLaunch_PreBuildHook(t *testing.T) {
	_, m := makeLaunchMockWithAPIKey(t)
	t.Setenv("ANTHROPIC_API_KEY", "test-key-hooks")

	m.config.Hooks.PreBuild = []string{"echo prebuild-ran"}

	// Should not error due to the hook (echo always succeeds)
	err := m.Launch(context.Background(), LaunchOpts{})
	// If there is an error, it must not be from the pre_build hook.
	if err != nil {
		assert.NotContains(t, err.Error(), "pre_build",
			"Launch should not fail due to successful pre_build hook")
	}
}

// TestLaunch_PreBuildHook_Failure verifies that a failing pre_build hook aborts Launch.
func TestLaunch_PreBuildHook_Failure(t *testing.T) {
	_, m := makeLaunchMockWithAPIKey(t)
	t.Setenv("ANTHROPIC_API_KEY", "test-key-hook-fail")

	m.config.Hooks.PreBuild = []string{"false"} // "false" always exits non-zero

	err := m.Launch(context.Background(), LaunchOpts{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pre_build",
		"error should mention pre_build when hook fails")
}

// TestStop_PostStopHook verifies that a successful post_stop hook does not affect Stop() return.
func TestStop_PostStopHook(t *testing.T) {
	mc := &mockClient{}
	cfg := newDefaultConfig()
	m, _ := newTestManager(t, mc, cfg)

	require.NoError(t, m.cache.EnsureDir())
	require.NoError(t, m.cache.SetContainerID("container-abc"))
	require.NoError(t, m.cache.SetNetworkID("net-xyz"))

	m.config.Hooks.PostStop = []string{"echo poststop-ran"}

	err := m.Stop(context.Background())
	require.NoError(t, err, "Stop should return nil when post_stop hook succeeds")
}

// TestStop_PostStopHook_Failure verifies that a failing post_stop hook is swallowed —
// Stop() still returns nil (post_stop hooks are warn-only).
func TestStop_PostStopHook_Failure(t *testing.T) {
	mc := &mockClient{}
	cfg := newDefaultConfig()
	m, _ := newTestManager(t, mc, cfg)

	require.NoError(t, m.cache.EnsureDir())
	require.NoError(t, m.cache.SetContainerID("container-abc"))
	require.NoError(t, m.cache.SetNetworkID("net-xyz"))

	m.config.Hooks.PostStop = []string{"false"} // always fails

	err := m.Stop(context.Background())
	require.NoError(t, err, "Stop should return nil even when post_stop hook fails")
}

// TestBuildImage_ProxyBuildArgs verifies that a configured HTTP proxy is passed as
// a build-arg to ImageBuild.
func TestBuildImage_ProxyBuildArgs(t *testing.T) {
	buildJSON := `{"aux":{"ID":"sha256:testimage123"}}` + "\n"
	mc := &mockClient{
		imageBuildResp: types.ImageBuildResponse{
			Body: io.NopCloser(strings.NewReader(buildJSON)),
		},
		imageInspectResp: types.ImageInspect{ID: "sha256:testimage123"},
	}

	cfg := newDefaultConfig()
	disabled := false
	cfg.Auth.MountHomeConfig = &disabled
	cfg.Network.HTTPProxy = "http://proxy:8080"

	m, _ := newTestManager(t, mc, cfg)
	require.NoError(t, m.cache.EnsureDir())

	_, err := m.buildImage(context.Background(), false)
	require.NoError(t, err)

	args := mc.lastBuildOptions.BuildArgs
	require.NotNil(t, args, "BuildArgs should be populated when proxy is configured")

	val, ok := args["HTTP_PROXY"]
	require.True(t, ok, "HTTP_PROXY should be in BuildArgs")
	assert.Equal(t, "http://proxy:8080", *val)

	valLower, ok := args["http_proxy"]
	require.True(t, ok, "http_proxy (lowercase) should be in BuildArgs")
	assert.Equal(t, "http://proxy:8080", *valLower)
}

// TestStop_FreshProcessFirewallCleanup verifies that Stop removes firewall rules
// even when m.firewall is nil (simulating a fresh zone stop process).
func TestStop_FreshProcessFirewallCleanup(t *testing.T) {
	mc := &mockClient{}
	cfg := newDefaultConfig()
	cfg.Network.Mode = "whitelist"
	m, _ := newTestManager(t, mc, cfg)

	// Override platform to simulate Linux with iptables support
	m.platform = Platform{OS: "linux", SupportsIPTables: true}

	require.NoError(t, m.cache.EnsureDir())
	require.NoError(t, m.cache.SetContainerID("container-abc"))
	require.NoError(t, m.cache.SetNetworkID("net-xyz"))

	// m.firewall is nil -- simulating a fresh process
	assert.Nil(t, m.firewall)

	// reconstructFirewallForCleanup should produce a non-nil Firewall
	fw := m.reconstructFirewallForCleanup(context.Background())
	assert.NotNil(t, fw, "reconstructFirewallForCleanup should return Firewall when mode=whitelist and platform supports iptables")

	// Verify Stop completes without error (cleanup will fail silently
	// since mock doesn't have real iptables, but the path is exercised)
	err := m.Stop(context.Background())
	require.NoError(t, err)
}

// TestStop_FreshProcessNoFirewallWhenModeNone verifies that Stop does NOT
// attempt firewall cleanup when mode=none.
func TestStop_FreshProcessNoFirewallWhenModeNone(t *testing.T) {
	mc := &mockClient{}
	cfg := newDefaultConfig()
	cfg.Network.Mode = "none"
	m, _ := newTestManager(t, mc, cfg)

	m.platform = Platform{OS: "linux", SupportsIPTables: true}
	require.NoError(t, m.cache.EnsureDir())
	require.NoError(t, m.cache.SetContainerID("container-abc"))
	require.NoError(t, m.cache.SetNetworkID("net-xyz"))

	assert.Nil(t, m.firewall)
	fw := m.reconstructFirewallForCleanup(context.Background())
	assert.Nil(t, fw, "reconstructFirewallForCleanup should return nil when mode=none")

	err := m.Stop(context.Background())
	require.NoError(t, err)
}
