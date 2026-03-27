// merge.go implements the two-tier config merge strategy.
package config

import "fmt"

// Merge combines a GlobalConfig and a RepoConfig into a MergedConfig and
// AnnotatedConfig. The AnnotatedConfig tracks the source of each field value
// for display in `zone config`.
//
// Merge rules (from spec section 4.4):
//   - Scalars: repo overrides global if non-empty/non-zero
//   - Bool pointers: repo overrides global if non-nil
//   - packages.apt/pip/npm: union (deduplicated, global-first)
//   - auth.forward_env: union (deduplicated, global-first)
//   - network.allow: per-repo allow appended to global default_allow
//   - network.deny: per-repo deny appended to global default_deny
//   - harness.extra_args: append (repo after global)
//   - hooks.pre_build/post_stop: append
//   - workspace.extra_mounts, workspace.ports: repo replaces global
func Merge(global *GlobalConfig, repo *RepoConfig) (*MergedConfig, *AnnotatedConfig) {
	m := &MergedConfig{}
	a := &AnnotatedConfig{}

	// --- Version ---
	m.Version, a.Version.Value, a.Version.Source = mergeIntAnnotated(global.Version, repo.Version)

	// --- Zone ---
	m.Zone.Harness, a.Harness.Value, a.Harness.Source = mergeString(global.Zone.Harness, repo.Zone.Harness)
	// Prefer repo HarnessName (from sugar) over zone.harness
	if repo.HarnessName != "" {
		m.Zone.Harness = repo.HarnessName
		a.Harness.Value = repo.HarnessName
		a.Harness.Source = SourceRepo
	}

	m.Zone.BaseImage, a.BaseImage.Value, a.BaseImage.Source = mergeString(global.Zone.BaseImage, repo.Zone.BaseImage)
	m.Zone.Shell, a.Shell.Value, a.Shell.Source = mergeString(global.Zone.Shell, repo.Zone.Shell)

	// --- Auth ---
	{
		v, av, src := mergeBoolPtr(global.Auth.MountHomeConfig, repo.Auth.MountHomeConfig)
		m.Auth.MountHomeConfig = &v
		a.MountHomeConfig.Value = av
		a.MountHomeConfig.Source = src
	}
	{
		v, av, src := mergeBoolPtr(global.Auth.ForwardSSHAgent, repo.Auth.ForwardSSHAgent)
		m.Auth.ForwardSSHAgent = &v
		a.ForwardSSHAgent.Value = av
		a.ForwardSSHAgent.Source = src
	}
	m.Auth.EnvFile, a.EnvFile.Value, a.EnvFile.Source = mergeString(global.Auth.EnvFile, repo.Auth.EnvFile)

	// forward_env: union (deduplicated)
	m.Auth.ForwardEnv, a.ForwardEnv = mergeUnion(global.Auth.ForwardEnv, repo.Auth.ForwardEnv)

	// --- Workspace (no workspace section in GlobalConfig) ---
	m.Workspace.MountPath, a.MountPath.Value, a.MountPath.Source = mergeString("", repo.Workspace.MountPath)
	{
		v, av, src := mergeBoolPtr(nil, repo.Workspace.PersistHome)
		m.Workspace.PersistHome = &v
		a.PersistHome.Value = av
		a.PersistHome.Source = src
	}

	// extra_mounts, ports: replace (repo replaces global; global has none)
	m.Workspace.ExtraMounts, a.ExtraMounts = mergeReplace(nil, repo.Workspace.ExtraMounts)
	m.Workspace.Ports, a.Ports = mergeReplace(nil, repo.Workspace.Ports)

	// --- Packages ---
	m.Packages.Apt, a.AptPackages = mergeUnion(global.Packages.Apt, repo.Packages.Apt)
	m.Packages.Pip, a.PipPackages = mergeUnion(global.Packages.Pip, repo.Packages.Pip)
	m.Packages.Npm, a.NpmPackages = mergeUnion(global.Packages.Npm, repo.Packages.Npm)

	// --- Resources ---
	m.Resources.Memory, a.Memory.Value, a.Memory.Source = mergeString(global.Resources.Memory, repo.Resources.Memory)
	m.Resources.Cpus, a.Cpus.Value, a.Cpus.Source = mergeString(global.Resources.Cpus, repo.Resources.Cpus)
	m.Resources.PidsLimit, a.PidsLimit.Value, a.PidsLimit.Source = mergeIntAnnotated(global.Resources.PidsLimit, repo.Resources.PidsLimit)

	// --- Network ---
	m.Network.Mode, a.NetworkMode.Value, a.NetworkMode.Source = mergeString(global.Network.Mode, repo.Network.Mode)
	m.Network.HTTPProxy, a.HTTPProxy.Value, a.HTTPProxy.Source = mergeString(global.Network.HTTPProxy, repo.Network.HTTPProxy)
	m.Network.HTTPSProxy, a.HTTPSProxy.Value, a.HTTPSProxy.Source = mergeString(global.Network.HTTPSProxy, repo.Network.HTTPSProxy)
	m.Network.NoProxy, a.NoProxy.Value, a.NoProxy.Source = mergeString(global.Network.NoProxy, repo.Network.NoProxy)

	// allow: global default_allow + repo allow (append)
	m.Network.Allow, a.Allow = mergeAppend(global.Network.DefaultAllow, repo.Network.Allow)
	// deny: global default_deny + repo deny (append)
	m.Network.Deny, a.Deny = mergeAppend(global.Network.DefaultDeny, repo.Network.Deny)

	// --- Hooks (only in repo; global has none) ---
	m.Hooks.PreBuild, a.PreBuild = mergeAppend(nil, repo.Hooks.PreBuild)
	m.Hooks.PostStop, a.PostStop = mergeAppend(nil, repo.Hooks.PostStop)

	// --- Harness ---
	m.Harness.Version, a.HarnessVersion.Value, a.HarnessVersion.Source = mergeString(global.Harness.Version, repo.Harness.Version)
	{
		v, av, src := mergeBoolPtr(global.Harness.SkipPermissions, repo.Harness.SkipPermissions)
		m.Harness.SkipPermissions = &v
		a.SkipPermissions.Value = av
		a.SkipPermissions.Source = src
	}
	m.Harness.NodeVersion, a.NodeVersion.Value, a.NodeVersion.Source = mergeString(global.Harness.NodeVersion, repo.Harness.NodeVersion)
	m.Harness.PythonVersion, a.PythonVersion.Value, a.PythonVersion.Source = mergeString(global.Harness.PythonVersion, repo.Harness.PythonVersion)
	m.Harness.EntrypointCommand, a.EntrypointCommand.Value, a.EntrypointCommand.Source = mergeString(global.Harness.EntrypointCommand, repo.Harness.EntrypointCommand)
	m.Harness.CustomHealthCheck, a.CustomHealthCheck.Value, a.CustomHealthCheck.Source = mergeString(global.Harness.CustomHealthCheck, repo.Harness.CustomHealthCheck)

	// harness.extra_args: append
	m.Harness.ExtraArgs, a.ExtraArgs = mergeAppend(global.Harness.ExtraArgs, repo.Harness.ExtraArgs)
	m.Harness.InstallCommands, a.InstallCommands = mergeAppend(global.Harness.InstallCommands, repo.Harness.InstallCommands)
	m.Harness.ConfigDirs, a.ConfigDirs = mergeAppend(global.Harness.ConfigDirs, repo.Harness.ConfigDirs)
	m.Harness.RequiredEnv, a.RequiredEnv = mergeAppend(global.Harness.RequiredEnv, repo.Harness.RequiredEnv)
	m.Harness.CustomShellRC, a.CustomShellRC = mergeAppend(global.Harness.CustomShellRC, repo.Harness.CustomShellRC)

	// Merge aliases maps
	m.Harness.CustomAliases = mergeStringMaps(global.Harness.CustomAliases, repo.Harness.CustomAliases)
	a.CustomAliases = mergeStringMapsAnnotated(global.Harness.CustomAliases, repo.Harness.CustomAliases)

	return m, a
}

// LoadMerged is a convenience function that loads both the global and repo
// configs and merges them.
func LoadMerged(repoPath string) (*MergedConfig, *AnnotatedConfig, error) {
	global, err := LoadGlobal()
	if err != nil {
		return nil, nil, fmt.Errorf("global config: %w", err)
	}
	repo, err := LoadRepo(repoPath)
	if err != nil {
		return nil, nil, fmt.Errorf("repo config: %w", err)
	}
	merged, annotated := Merge(global, repo)
	return merged, annotated, nil
}

// ---------------------------------------------------------------------------
// Merge primitives
// ---------------------------------------------------------------------------

// mergeString: repo wins if non-empty, else global, else "" with SourceDefault.
func mergeString(global, repo string) (string, string, Source) {
	if repo != "" {
		return repo, repo, SourceRepo
	}
	if global != "" {
		return global, global, SourceGlobal
	}
	return "", "", SourceDefault
}

// mergeBoolPtr: repo wins if non-nil, else global, else false with SourceDefault.
func mergeBoolPtr(global, repo *bool) (bool, bool, Source) {
	if repo != nil {
		return *repo, *repo, SourceRepo
	}
	if global != nil {
		return *global, *global, SourceGlobal
	}
	return false, false, SourceDefault
}

// mergeIntAnnotated: repo wins if non-zero, else global, else 0 with SourceDefault.
func mergeIntAnnotated(global, repo int) (int, int, Source) {
	if repo != 0 {
		return repo, repo, SourceRepo
	}
	if global != 0 {
		return global, global, SourceGlobal
	}
	return 0, 0, SourceDefault
}

// mergeUnion: deduplicated union (global first, then repo additions).
func mergeUnion(global, repo []string) ([]string, []AnnotatedListItem) {
	seen := make(map[string]bool)
	var result []string
	var annotated []AnnotatedListItem
	for _, v := range global {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
			annotated = append(annotated, AnnotatedListItem{Value: v, Source: SourceGlobal})
		}
	}
	for _, v := range repo {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
			annotated = append(annotated, AnnotatedListItem{Value: v, Source: SourceRepo})
		}
	}
	return result, annotated
}

// mergeAppend: global + repo concatenated (no dedup).
func mergeAppend(global, repo []string) ([]string, []AnnotatedListItem) {
	var result []string
	var annotated []AnnotatedListItem
	for _, v := range global {
		result = append(result, v)
		annotated = append(annotated, AnnotatedListItem{Value: v, Source: SourceGlobal})
	}
	for _, v := range repo {
		result = append(result, v)
		annotated = append(annotated, AnnotatedListItem{Value: v, Source: SourceRepo})
	}
	return result, annotated
}

// mergeReplace: repo replaces global entirely if repo is non-empty.
func mergeReplace(global, repo []string) ([]string, []AnnotatedListItem) {
	source := SourceGlobal
	vals := global
	if len(repo) > 0 {
		source = SourceRepo
		vals = repo
	}
	var annotated []AnnotatedListItem
	for _, v := range vals {
		annotated = append(annotated, AnnotatedListItem{Value: v, Source: source})
	}
	return vals, annotated
}

// mergeStringMaps: repo values override global values for same keys.
func mergeStringMaps(global, repo map[string]string) map[string]string {
	if len(global) == 0 && len(repo) == 0 {
		return nil
	}
	result := make(map[string]string)
	for k, v := range global {
		result[k] = v
	}
	for k, v := range repo {
		result[k] = v
	}
	return result
}

// mergeStringMapsAnnotated: repo values override global values for same keys,
// with source tracking.
func mergeStringMapsAnnotated(global, repo map[string]string) map[string]AnnotatedField[string] {
	if len(global) == 0 && len(repo) == 0 {
		return nil
	}
	result := make(map[string]AnnotatedField[string])
	for k, v := range global {
		result[k] = AnnotatedField[string]{Value: v, Source: SourceGlobal}
	}
	for k, v := range repo {
		result[k] = AnnotatedField[string]{Value: v, Source: SourceRepo}
	}
	return result
}
