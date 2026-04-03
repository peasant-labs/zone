// harness_bridge.go translates Harness interface method calls and MergedConfig into
// the DockerfileData, EntrypointData, and ShellRCData structs used by the render pipeline.
// This is the single integration point between the harness system and template rendering.
//
// Phase 6 callers use these bridge functions instead of populating template data structs
// directly, keeping all harness-to-template translation logic centralized here.
package docker

import (
	"github.com/peasant-labs/zone/internal/config"
	"github.com/peasant-labs/zone/internal/harness"
)

const defaultMountPath = "/workspace"

func resolveMountPath(mountPath string) string {
	if mountPath == "" {
		return defaultMountPath
	}
	return mountPath
}

// BuildDockerfileData translates a Harness and MergedConfig into a DockerfileData
// struct ready for RenderDockerfile(). The caller (Phase 6) must still set HostUID
// and MacOSUsername after calling this function, as those are runtime values that
// require os/user lookups not appropriate here.
func BuildDockerfileData(h harness.Harness, cfg *config.MergedConfig) DockerfileData {
	nodeVer := cfg.Harness.NodeVersion
	if nodeVer == "" {
		nodeVer = "22"
	}
	pythonVer := cfg.Harness.PythonVersion
	if pythonVer == "" {
		pythonVer = "3.12"
	}
	mountPath := resolveMountPath(cfg.Workspace.MountPath)
	return DockerfileData{
		BaseImage:              cfg.Zone.BaseImage,
		AptPackages:            mergeSlices(cfg.Packages.Apt, h.DefaultAptPackages()),
		NeedsNode:              h.NeedsNode(),
		NodeVersion:            nodeVer,
		NeedsPython:            h.NeedsPython(),
		PythonVersion:          pythonVer,
		NpmPackages:            mergeSlices(cfg.Packages.Npm, h.DefaultNpmPackages()),
		PipPackages:            mergeSlices(cfg.Packages.Pip, h.DefaultPipPackages()),
		HarnessInstallCommands: h.InstallCommands(),
		HealthCheck:            h.HealthCheck(),
		InstallZsh:             cfg.Zone.Shell == "zsh",
		Shell:                  cfg.Zone.Shell,
		PostInstallCommands:    h.PostInstallCommands(),
		MountPath:              mountPath,
		// HostUID and MacOSUsername are set by the Phase 6 caller (runtime values)
	}
}

// BuildEntrypointData translates a Harness and MergedConfig into an EntrypointData
// struct ready for RenderEntrypoint(). Calls DetectGitIdentity() to populate the
// git forwarding fields.
func BuildEntrypointData(h harness.Harness, cfg *config.MergedConfig) EntrypointData {
	var copyCmds []string
	if homeDir := h.HomeConfigDir(); homeDir != "" {
		copyCmds = append(copyCmds, configCopyCmd(homeDir))
	}
	for _, d := range h.ExtraConfigDirs() {
		copyCmds = append(copyCmds, configCopyCmd(d))
	}

	name, email, forward := DetectGitIdentity()
	mountPath := resolveMountPath(cfg.Workspace.MountPath)

	return EntrypointData{
		MountPath:          mountPath,
		ForwardGitConfig:   forward,
		GitUserName:        name,
		GitUserEmail:       email,
		ConfigCopyCommands: copyCmds,
		Shell:              cfg.Zone.Shell,
		EntrypointCommand:  h.EntrypointCommand(),
	}
}

// BuildShellRCData translates a Harness and MergedConfig into a ShellRCData struct
// ready for RenderShellRC().
func BuildShellRCData(h harness.Harness, cfg *config.MergedConfig) ShellRCData {
	mountPath := resolveMountPath(cfg.Workspace.MountPath)
	return ShellRCData{
		HarnessName:    h.Name(),
		MountPath:      mountPath,
		Aliases:        h.Aliases(),
		ShellRC:        h.ShellRC(),
		WelcomeMessage: h.WelcomeMessage(),
	}
}

// mergeSlices appends b to a copy of a. Returns nil if both slices are empty/nil.
// This avoids mutating the input slices and preserves config-first ordering
// (user-specified packages appear before harness defaults).
func mergeSlices(a, b []string) []string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	result := make([]string, 0, len(a)+len(b))
	result = append(result, a...)
	result = append(result, b...)
	return result
}

// configCopyCmd generates a shell command to copy a config directory into the
// container home at startup. The entrypoint template iterates ConfigCopyCommands
// and executes each one.
//
// Design: Phase 7 mounts host config dirs at <dir>.host inside the container.
// The entrypoint copies them to <dir> at startup (copy-on-start strategy).
// Using "|| true" ensures missing host dirs don't abort the entrypoint.
func configCopyCmd(dir string) string {
	return "mkdir -p $(dirname " + dir + ") && cp -r " + dir + ".host " + dir + " 2>/dev/null || true"
}
