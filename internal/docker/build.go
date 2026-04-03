// build.go implements the Docker image build pipeline: tar context construction,
// JSON build output streaming, and the buildImage orchestrator.
package docker

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/harness"
)

// BuildLine carries a single line of Docker build output.
type BuildLine struct {
	Text string
}

// BuildResult carries the final build outcome.
type BuildResult struct {
	ImageID string
	Err     error
}

// BuildWithProgress starts the Docker image build in a goroutine.
// Build output lines are sent to the lines channel as they arrive.
// The final result (image ID or error) is sent to the result channel.
// Both channels are closed by this method when the build finishes.
//
// Usage: create channels, call BuildWithProgress, read lines until result arrives.
func (m *Manager) BuildWithProgress(ctx context.Context, noCache bool, lines chan<- BuildLine, result chan<- BuildResult) {
	go func() {
		defer close(lines)
		defer close(result)
		imageID, err := m.buildImageWithChannel(ctx, noCache, lines)
		result <- BuildResult{ImageID: imageID, Err: err}
	}()
}

// buildMessage represents a single JSON line from Docker's image build stream.
type buildMessage struct {
	Stream      string `json:"stream"`
	Error       string `json:"error"`
	ErrorDetail *struct {
		Message string `json:"message"`
	} `json:"errorDetail"`
	Aux *struct {
		ID string `json:"ID"`
	} `json:"aux"`
}

// buildContext creates a tar archive containing the Dockerfile, entrypoint script,
// and shell RC file. The archive is returned as an io.Reader for ImageBuild.
// entrypoint.sh is added with mode 0755 to ensure it is executable inside the container.
func buildContext(dockerfile, entrypoint, shellrc string) (io.Reader, error) {
	buf := &bytes.Buffer{}
	tw := tar.NewWriter(buf)

	files := []struct {
		name    string
		content string
		mode    int64
	}{
		{"Dockerfile", dockerfile, 0644},
		{"entrypoint.sh", entrypoint, 0755},
		{"zone-bashrc", shellrc, 0644},
	}

	for _, f := range files {
		hdr := &tar.Header{
			Name: f.name,
			Mode: f.mode,
			Size: int64(len(f.content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, fmt.Errorf("tar header %s: %w", f.name, err)
		}
		if _, err := tw.Write([]byte(f.content)); err != nil {
			return nil, fmt.Errorf("tar write %s: %w", f.name, err)
		}
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("tar close: %w", err)
	}
	return buf, nil
}

// streamBuildOutput reads Docker's JSON build response stream, writes plain-text
// progress to w, and captures the final image ID from the aux message.
// Returns an error if any JSON line contains an error field.
func streamBuildOutput(body io.ReadCloser, w io.Writer) (imageID string, err error) {
	defer body.Close()
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		var msg buildMessage
		if jsonErr := json.Unmarshal(scanner.Bytes(), &msg); jsonErr != nil {
			continue // skip malformed lines
		}
		if msg.Error != "" {
			return "", fmt.Errorf("docker build: %s", msg.Error)
		}
		if msg.Stream != "" {
			fmt.Fprint(w, msg.Stream)
		}
		if msg.Aux != nil && msg.Aux.ID != "" {
			imageID = msg.Aux.ID
		}
	}
	return imageID, scanner.Err()
}

// buildImage orchestrates the full Docker image build pipeline:
//  1. Resolve harness from config
//  2. Render Dockerfile, entrypoint.sh, zone-bashrc templates
//  3. Build tar context
//  4. Compute config hash (for cache invalidation)
//  5. Stream ImageBuild output to stderr + build log
//  6. Verify image exists via ImageInspect
//  7. Persist image ID and config hash in cache
func (m *Manager) buildImage(ctx context.Context, noCache bool) (string, error) {
	h, err := harness.Get(m.config.Zone.Harness, &m.config.Harness)
	if err != nil {
		return "", fmt.Errorf("get harness: %w", err)
	}

	// Build template data structs via bridge functions
	dfData := BuildDockerfileData(h, m.config)
	uid, err := HostUID()
	if err != nil {
		return "", fmt.Errorf("get host UID: %w", err)
	}
	dfData.HostUID = uid
	dfData.MacOSUsername = MacOSUsername()

	epData := BuildEntrypointData(h, m.config)
	rcData := BuildShellRCData(h, m.config)

	// Render templates
	dockerfile, err := RenderDockerfile(dfData, m.version)
	if err != nil {
		return "", fmt.Errorf("render Dockerfile: %w", err)
	}
	entrypoint, err := RenderEntrypoint(epData, m.version)
	if err != nil {
		return "", fmt.Errorf("render entrypoint: %w", err)
	}
	shellrc, err := RenderShellRC(rcData, m.version)
	if err != nil {
		return "", fmt.Errorf("render shellrc: %w", err)
	}

	// Build tar context
	ctx2, err := buildContext(dockerfile, entrypoint, shellrc)
	if err != nil {
		return "", fmt.Errorf("build context: %w", err)
	}

	// Compute config hash for cache invalidation
	hash, err := cache.ComputeHash(m.config, m.version)
	if err != nil {
		return "", fmt.Errorf("compute config hash: %w", err)
	}

	// Create build log; write to both stderr and the log file
	logWriter, closer, err := m.cache.CreateBuildLog(os.Stderr, hash, m.version)
	if err != nil {
		return "", fmt.Errorf("create build log: %w", err)
	}
	defer closer()

	// Proxy build-args (CFG-15)
	httpProxy, httpsProxy, noProxy := resolveProxy(&m.config.Network)
	buildArgs := proxyBuildArgs(httpProxy, httpsProxy, noProxy)

	containerName := ContainerName(m.repoDir)
	buildResp, err := m.client.ImageBuild(ctx, ctx2, types.ImageBuildOptions{
		Tags:       []string{containerName + ":latest"},
		Dockerfile: "Dockerfile",
		Remove:     true,
		NoCache:    noCache,
		BuildArgs:  buildArgs,
	})
	if err != nil {
		return "", fmt.Errorf("image build: %w", err)
	}

	imageID, err := streamBuildOutput(buildResp.Body, logWriter)
	if err != nil {
		m.showBuildError()
		return "", err
	}

	// Verify the image still exists (may have been pruned mid-build)
	if _, _, err := m.client.ImageInspectWithRaw(ctx, imageID); err != nil {
		return "", fmt.Errorf("image vanished after build: %w", err)
	}

	// Persist to cache
	if err := m.cache.SetImageID(imageID); err != nil {
		return "", fmt.Errorf("cache image ID: %w", err)
	}
	if err := m.cache.SetConfigHash(hash); err != nil {
		return "", fmt.Errorf("cache config hash: %w", err)
	}

	return imageID, nil
}

// buildImageWithChannel is like buildImage but also sends each build output
// line to the provided channel in addition to writing to the log file.
// Used by BuildWithProgress for streaming TUI display.
func (m *Manager) buildImageWithChannel(ctx context.Context, noCache bool, lines chan<- BuildLine) (string, error) {
	h, err := harness.Get(m.config.Zone.Harness, &m.config.Harness)
	if err != nil {
		return "", fmt.Errorf("get harness: %w", err)
	}

	dfData := BuildDockerfileData(h, m.config)
	uid, err := HostUID()
	if err != nil {
		return "", fmt.Errorf("get host UID: %w", err)
	}
	dfData.HostUID = uid
	dfData.MacOSUsername = MacOSUsername()

	epData := BuildEntrypointData(h, m.config)
	rcData := BuildShellRCData(h, m.config)

	dockerfile, err := RenderDockerfile(dfData, m.version)
	if err != nil {
		return "", fmt.Errorf("render Dockerfile: %w", err)
	}
	entrypoint, err := RenderEntrypoint(epData, m.version)
	if err != nil {
		return "", fmt.Errorf("render entrypoint: %w", err)
	}
	shellrc, err := RenderShellRC(rcData, m.version)
	if err != nil {
		return "", fmt.Errorf("render shellrc: %w", err)
	}

	ctx2, err := buildContext(dockerfile, entrypoint, shellrc)
	if err != nil {
		return "", fmt.Errorf("build context: %w", err)
	}

	hash, err := cache.ComputeHash(m.config, m.version)
	if err != nil {
		return "", fmt.Errorf("compute config hash: %w", err)
	}

	logWriter, closer, err := m.cache.CreateBuildLog(os.Stderr, hash, m.version)
	if err != nil {
		return "", fmt.Errorf("create build log: %w", err)
	}
	defer closer()

	httpProxy, httpsProxy, noProxy := resolveProxy(&m.config.Network)
	buildArgs := proxyBuildArgs(httpProxy, httpsProxy, noProxy)

	containerName := ContainerName(m.repoDir)
	buildResp, err := m.client.ImageBuild(ctx, ctx2, types.ImageBuildOptions{
		Tags:       []string{containerName + ":latest"},
		Dockerfile: "Dockerfile",
		Remove:     true,
		NoCache:    noCache,
		BuildArgs:  buildArgs,
	})
	if err != nil {
		return "", fmt.Errorf("image build: %w", err)
	}

	imageID, err := streamBuildOutputWithChannel(buildResp.Body, logWriter, lines)
	if err != nil {
		m.showBuildError()
		return "", err
	}

	if _, _, err := m.client.ImageInspectWithRaw(ctx, imageID); err != nil {
		return "", fmt.Errorf("image vanished after build: %w", err)
	}

	if err := m.cache.SetImageID(imageID); err != nil {
		return "", fmt.Errorf("cache image ID: %w", err)
	}
	if err := m.cache.SetConfigHash(hash); err != nil {
		return "", fmt.Errorf("cache config hash: %w", err)
	}

	return imageID, nil
}

// streamBuildOutputWithChannel reads Docker's JSON build response stream, writes
// plain-text progress to logW, sends each line to the lines channel for TUI display,
// and captures the final image ID from the aux message.
func streamBuildOutputWithChannel(body io.ReadCloser, logW io.Writer, lines chan<- BuildLine) (string, error) {
	defer body.Close()
	scanner := bufio.NewScanner(body)
	var imageID string
	for scanner.Scan() {
		var msg buildMessage
		if jsonErr := json.Unmarshal(scanner.Bytes(), &msg); jsonErr != nil {
			continue
		}
		if msg.Error != "" {
			return "", fmt.Errorf("docker build: %s", msg.Error)
		}
		if msg.Stream != "" {
			fmt.Fprint(logW, msg.Stream) // still write to log
			lines <- BuildLine{Text: msg.Stream}
		}
		if msg.Aux != nil && msg.Aux.ID != "" {
			imageID = msg.Aux.ID
		}
	}
	return imageID, scanner.Err()
}

// showBuildError prints the last 20 lines of the build log to stderr.
// Called when buildImage encounters a build error.
func (m *Manager) showBuildError() {
	logPath := m.cache.Dir() + "/logs/last_build.log"
	data, err := os.ReadFile(logPath)
	if err != nil {
		return
	}

	lines := bytes.Split(data, []byte("\n"))
	start := len(lines) - 20
	if start < 0 {
		start = 0
	}

	fmt.Fprintln(os.Stderr, "\n--- Build error (last 20 lines) ---")
	for _, line := range lines[start:] {
		fmt.Fprintf(os.Stderr, "%s\n", line)
	}
	fmt.Fprintf(os.Stderr, "Full build log: %s\n", logPath)
}
