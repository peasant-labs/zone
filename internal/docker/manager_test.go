// manager_test.go tests Manager construction, build pipeline, network helpers,
// and resource parsers using a mock DockerClient (no live Docker daemon required).
package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/errdefs"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/config"
)

// mockClient is a configurable mock implementation of DockerClient.
// Fields are set before each test to control return values.
type mockClient struct {
	pingErr          error
	pingResp         types.Ping
	networkCreateID  string
	networkCreateErr error
	networkRemoveErr error
	containerCreateResp container.CreateResponse
	containerCreateErr  error
	imageBuildResp   types.ImageBuildResponse
	imageBuildErr    error
	imageInspectResp types.ImageInspect
	imageInspectErr  error
}

func (m *mockClient) Ping(ctx context.Context) (types.Ping, error) {
	return m.pingResp, m.pingErr
}

func (m *mockClient) ImageBuild(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	return m.imageBuildResp, m.imageBuildErr
}

func (m *mockClient) ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
	return m.imageInspectResp, nil, m.imageInspectErr
}

func (m *mockClient) ImageRemove(ctx context.Context, imageID string, options image.RemoveOptions) ([]image.DeleteResponse, error) {
	return nil, nil
}

func (m *mockClient) ContainerCreate(ctx context.Context, cfg *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error) {
	return m.containerCreateResp, m.containerCreateErr
}

func (m *mockClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	return nil
}

func (m *mockClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	return nil
}

func (m *mockClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	return nil
}

func (m *mockClient) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	return container.InspectResponse{}, nil
}

func (m *mockClient) ContainerUnpause(ctx context.Context, containerID string) error {
	return nil
}

func (m *mockClient) NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
	return network.CreateResponse{ID: m.networkCreateID}, m.networkCreateErr
}

func (m *mockClient) NetworkRemove(ctx context.Context, networkID string) error {
	return m.networkRemoveErr
}

func (m *mockClient) VolumeCreate(ctx context.Context, options volume.CreateOptions) (volume.Volume, error) {
	return volume.Volume{}, nil
}

func (m *mockClient) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	return nil
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
