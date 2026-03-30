package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/peasant-labs/zone/internal/config"
	"github.com/spf13/cobra"
)

var (
	configJSON   bool
	configGlobal bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show effective merged configuration with source annotations",
	Long: `Show the effective merged configuration with source annotations.

Displays the result of merging global (~/.config/zone/config.toml) and
repo (zone.toml) configs. Each value is annotated with its source.
Use --json for machine-readable output.`,
	Example: `  zone config
  zone config --json
  zone config --global`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if configGlobal {
			return showGlobalConfig(cmd, configJSON)
		}
		return showMergedConfig(cmd, configJSON)
	},
}

func init() {
	configCmd.Flags().BoolVar(&configJSON, "json", false, "Output merged config as JSON with source metadata")
	configCmd.Flags().BoolVar(&configGlobal, "global", false, "Show global config (works without zone.toml)")
}

// showGlobalConfig renders global config (with defaults) to stdout.
func showGlobalConfig(cmd *cobra.Command, asJSON bool) error {
	global, err := config.LoadGlobal()
	if err != nil {
		// Even with unknown keys, we still have a config.
		var uke *config.UnknownKeysError
		if !errors.As(err, &uke) {
			return fmt.Errorf("loading global config: %w", err)
		}
	}

	// Merge global against an empty repo to produce an AnnotatedConfig.
	repo := config.DefaultRepoConfig()
	// Clear repo defaults so global is the sole source.
	repo.Version = 0
	repo.Workspace.MountPath = ""
	repo.Resources.PidsLimit = 0
	_, annotated := config.Merge(global, repo)

	if asJSON {
		output := renderJSON(annotated)
		fmt.Fprintln(cmd.OutOrStdout(), output)
		return nil
	}

	output := renderAnnotatedTOML(annotated)
	fmt.Fprint(cmd.OutOrStdout(), output)
	return nil
}

// showMergedConfig loads zone.toml from cwd and renders merged config.
func showMergedConfig(cmd *cobra.Command, asJSON bool) error {
	const repoPath = "zone.toml"
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		cmd.PrintErrln("No zone.toml found. Run `zone init` to create one, or `zone config --global` to view global defaults.")
		return config.ErrNoConfig
	}

	_, annotated, err := config.LoadMerged(repoPath)
	if err != nil {
		// Unknown keys are non-fatal for display purposes.
		var uke *config.UnknownKeysError
		if !errors.As(err, &uke) {
			return fmt.Errorf("loading config: %w", err)
		}
		// Re-load with lower-level calls to still get annotated config.
		global, globalErr := config.LoadGlobal()
		if globalErr != nil {
			var guke *config.UnknownKeysError
			if !errors.As(globalErr, &guke) {
				return fmt.Errorf("loading global config: %w", globalErr)
			}
		}
		repo, repoErr := config.LoadRepo(repoPath)
		var ruke *config.UnknownKeysError
		if repoErr != nil && !errors.As(repoErr, &ruke) {
			return fmt.Errorf("loading repo config: %w", repoErr)
		}
		_, annotated = config.Merge(global, repo)
	}

	if asJSON {
		output := renderJSON(annotated)
		fmt.Fprintln(cmd.OutOrStdout(), output)
		return nil
	}

	output := renderAnnotatedTOML(annotated)
	fmt.Fprint(cmd.OutOrStdout(), output)
	return nil
}

// renderAnnotatedTOML generates TOML output with inline source comments.
// Scalar fields get a trailing "# <source>" comment.
// List fields get a comment block above the array showing provenance.
func renderAnnotatedTOML(a *config.AnnotatedConfig) string {
	var sb strings.Builder

	// --- Top-level ---
	if a.Version.Value != 0 {
		fmt.Fprintf(&sb, "version = %d # %s\n", a.Version.Value, a.Version.Source)
	}

	// --- [zone] ---
	zoneLines := buildZoneSection(a)
	if len(zoneLines) > 0 {
		fmt.Fprintf(&sb, "\n[zone]\n")
		for _, l := range zoneLines {
			fmt.Fprintln(&sb, l)
		}
	}

	// --- [auth] ---
	authLines := buildAuthSection(a)
	if len(authLines) > 0 {
		fmt.Fprintf(&sb, "\n[auth]\n")
		for _, l := range authLines {
			fmt.Fprintln(&sb, l)
		}
	}

	// --- [workspace] ---
	wsLines := buildWorkspaceSection(a)
	if len(wsLines) > 0 {
		fmt.Fprintf(&sb, "\n[workspace]\n")
		for _, l := range wsLines {
			fmt.Fprintln(&sb, l)
		}
	}

	// --- [packages] ---
	pkgLines := buildPackagesSection(a)
	if len(pkgLines) > 0 {
		fmt.Fprintf(&sb, "\n[packages]\n")
		for _, l := range pkgLines {
			fmt.Fprintln(&sb, l)
		}
	}

	// --- [resources] ---
	resLines := buildResourcesSection(a)
	if len(resLines) > 0 {
		fmt.Fprintf(&sb, "\n[resources]\n")
		for _, l := range resLines {
			fmt.Fprintln(&sb, l)
		}
	}

	// --- [network] ---
	netLines := buildNetworkSection(a)
	if len(netLines) > 0 {
		fmt.Fprintf(&sb, "\n[network]\n")
		for _, l := range netLines {
			fmt.Fprintln(&sb, l)
		}
	}

	// --- [hooks] ---
	hooksLines := buildHooksSection(a)
	if len(hooksLines) > 0 {
		fmt.Fprintf(&sb, "\n[hooks]\n")
		for _, l := range hooksLines {
			fmt.Fprintln(&sb, l)
		}
	}

	// --- [harness] ---
	harnessLines := buildHarnessSection(a)
	if len(harnessLines) > 0 {
		fmt.Fprintf(&sb, "\n[harness]\n")
		for _, l := range harnessLines {
			fmt.Fprintln(&sb, l)
		}
	}

	return sb.String()
}

func buildZoneSection(a *config.AnnotatedConfig) []string {
	var lines []string
	if a.Harness.Value != "" {
		lines = append(lines, fmt.Sprintf("harness = %q # %s", a.Harness.Value, a.Harness.Source))
	}
	if a.BaseImage.Value != "" {
		lines = append(lines, fmt.Sprintf("base_image = %q # %s", a.BaseImage.Value, a.BaseImage.Source))
	}
	if a.Shell.Value != "" {
		lines = append(lines, fmt.Sprintf("shell = %q # %s", a.Shell.Value, a.Shell.Source))
	}
	return lines
}

func buildAuthSection(a *config.AnnotatedConfig) []string {
	var lines []string
	lines = append(lines, fmt.Sprintf("mount_home_config = %v # %s", a.MountHomeConfig.Value, a.MountHomeConfig.Source))
	lines = append(lines, fmt.Sprintf("forward_ssh_agent = %v # %s", a.ForwardSSHAgent.Value, a.ForwardSSHAgent.Source))
	if a.EnvFile.Value != "" {
		lines = append(lines, fmt.Sprintf("env_file = %q # %s", a.EnvFile.Value, a.EnvFile.Source))
	}
	if len(a.ForwardEnv) > 0 {
		lines = append(lines, buildListLines("forward_env", a.ForwardEnv)...)
	}
	return lines
}

func buildWorkspaceSection(a *config.AnnotatedConfig) []string {
	var lines []string
	if a.MountPath.Value != "" {
		lines = append(lines, fmt.Sprintf("mount_path = %q # %s", a.MountPath.Value, a.MountPath.Source))
	}
	lines = append(lines, fmt.Sprintf("persist_home = %v # %s", a.PersistHome.Value, a.PersistHome.Source))
	if len(a.ExtraMounts) > 0 {
		lines = append(lines, buildListLines("extra_mounts", a.ExtraMounts)...)
	}
	if len(a.Ports) > 0 {
		lines = append(lines, buildListLines("ports", a.Ports)...)
	}
	return lines
}

func buildPackagesSection(a *config.AnnotatedConfig) []string {
	var lines []string
	if len(a.AptPackages) > 0 {
		lines = append(lines, buildListLines("apt", a.AptPackages)...)
	}
	if len(a.PipPackages) > 0 {
		lines = append(lines, buildListLines("pip", a.PipPackages)...)
	}
	if len(a.NpmPackages) > 0 {
		lines = append(lines, buildListLines("npm", a.NpmPackages)...)
	}
	return lines
}

func buildResourcesSection(a *config.AnnotatedConfig) []string {
	var lines []string
	if a.Memory.Value != "" {
		lines = append(lines, fmt.Sprintf("memory = %q # %s", a.Memory.Value, a.Memory.Source))
	}
	if a.Cpus.Value != "" {
		lines = append(lines, fmt.Sprintf("cpus = %q # %s", a.Cpus.Value, a.Cpus.Source))
	}
	if a.PidsLimit.Value != 0 {
		lines = append(lines, fmt.Sprintf("pids_limit = %d # %s", a.PidsLimit.Value, a.PidsLimit.Source))
	}
	return lines
}

func buildNetworkSection(a *config.AnnotatedConfig) []string {
	var lines []string
	if a.NetworkMode.Value != "" {
		lines = append(lines, fmt.Sprintf("mode = %q # %s", a.NetworkMode.Value, a.NetworkMode.Source))
	}
	if a.HTTPProxy.Value != "" {
		lines = append(lines, fmt.Sprintf("http_proxy = %q # %s", a.HTTPProxy.Value, a.HTTPProxy.Source))
	}
	if a.HTTPSProxy.Value != "" {
		lines = append(lines, fmt.Sprintf("https_proxy = %q # %s", a.HTTPSProxy.Value, a.HTTPSProxy.Source))
	}
	if a.NoProxy.Value != "" {
		lines = append(lines, fmt.Sprintf("no_proxy = %q # %s", a.NoProxy.Value, a.NoProxy.Source))
	}
	if len(a.Allow) > 0 {
		lines = append(lines, buildListLines("allow", a.Allow)...)
	}
	if len(a.Deny) > 0 {
		lines = append(lines, buildListLines("deny", a.Deny)...)
	}
	return lines
}

func buildHooksSection(a *config.AnnotatedConfig) []string {
	var lines []string
	if len(a.PreBuild) > 0 {
		lines = append(lines, buildListLines("pre_build", a.PreBuild)...)
	}
	if len(a.PostStop) > 0 {
		lines = append(lines, buildListLines("post_stop", a.PostStop)...)
	}
	return lines
}

func buildHarnessSection(a *config.AnnotatedConfig) []string {
	var lines []string
	if a.HarnessVersion.Value != "" {
		lines = append(lines, fmt.Sprintf("version = %q # %s", a.HarnessVersion.Value, a.HarnessVersion.Source))
	}
	if a.SkipPermissions.Value {
		lines = append(lines, fmt.Sprintf("skip_permissions = %v # %s", a.SkipPermissions.Value, a.SkipPermissions.Source))
	}
	if a.NodeVersion.Value != "" {
		lines = append(lines, fmt.Sprintf("node_version = %q # %s", a.NodeVersion.Value, a.NodeVersion.Source))
	}
	if a.PythonVersion.Value != "" {
		lines = append(lines, fmt.Sprintf("python_version = %q # %s", a.PythonVersion.Value, a.PythonVersion.Source))
	}
	if a.EntrypointCommand.Value != "" {
		lines = append(lines, fmt.Sprintf("entrypoint_command = %q # %s", a.EntrypointCommand.Value, a.EntrypointCommand.Source))
	}
	if a.CustomHealthCheck.Value != "" {
		lines = append(lines, fmt.Sprintf("health_check = %q # %s", a.CustomHealthCheck.Value, a.CustomHealthCheck.Source))
	}
	if len(a.ExtraArgs) > 0 {
		lines = append(lines, buildListLines("extra_args", a.ExtraArgs)...)
	}
	if len(a.InstallCommands) > 0 {
		lines = append(lines, buildListLines("install_commands", a.InstallCommands)...)
	}
	if len(a.ConfigDirs) > 0 {
		lines = append(lines, buildListLines("config_dirs", a.ConfigDirs)...)
	}
	if len(a.RequiredEnv) > 0 {
		lines = append(lines, buildListLines("required_env", a.RequiredEnv)...)
	}
	if len(a.CustomShellRC) > 0 {
		lines = append(lines, buildListLines("shell_rc", a.CustomShellRC)...)
	}
	if len(a.CustomAliases) > 0 {
		lines = append(lines, buildAliasesLines(a.CustomAliases)...)
	}
	return lines
}

// buildListLines builds the annotated TOML representation for a list field.
// Per spec: inline comments on array elements are invalid TOML, so we emit
// a comment block ABOVE the array showing provenance by source.
func buildListLines(key string, items []config.AnnotatedListItem) []string {
	if len(items) == 0 {
		return nil
	}

	// Group items by source to produce a readable comment.
	sourceGroups := make(map[config.Source][]string)
	var sourceOrder []config.Source
	seen := make(map[config.Source]bool)
	for _, item := range items {
		if !seen[item.Source] {
			seen[item.Source] = true
			sourceOrder = append(sourceOrder, item.Source)
		}
		sourceGroups[item.Source] = append(sourceGroups[item.Source], item.Value)
	}

	var commentParts []string
	for _, src := range sourceOrder {
		vals := sourceGroups[src]
		quoted := make([]string, len(vals))
		for i, v := range vals {
			quoted[i] = fmt.Sprintf("%q", v)
		}
		commentParts = append(commentParts, fmt.Sprintf("%s provides [%s]", src, strings.Join(quoted, ", ")))
	}

	// Build the merged array.
	allVals := make([]string, len(items))
	for i, item := range items {
		allVals[i] = fmt.Sprintf("%q", item.Value)
	}

	comment := fmt.Sprintf("# %s: %s", key, strings.Join(commentParts, "; "))
	array := fmt.Sprintf("%s = [%s]", key, strings.Join(allVals, ", "))
	return []string{comment, array}
}

// buildAliasesLines builds TOML lines for the aliases map in [harness].
func buildAliasesLines(aliases map[string]config.AnnotatedField[string]) []string {
	if len(aliases) == 0 {
		return nil
	}
	var lines []string
	// Sort keys for deterministic output.
	keys := make([]string, 0, len(aliases))
	for k := range aliases {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines = append(lines, "[harness.aliases]")
	for _, k := range keys {
		f := aliases[k]
		lines = append(lines, fmt.Sprintf("%s = %q # %s", k, f.Value, f.Source))
	}
	return lines
}

// renderJSON produces JSON output where each leaf field is {"value": X, "source": "..."}.
func renderJSON(a *config.AnnotatedConfig) string {
	data := map[string]any{
		"version": config.AnnotatedFieldJSON{Value: a.Version.Value, Source: a.Version.Source},
		"zone": map[string]any{
			"harness":    config.AnnotatedFieldJSON{Value: a.Harness.Value, Source: a.Harness.Source},
			"base_image": config.AnnotatedFieldJSON{Value: a.BaseImage.Value, Source: a.BaseImage.Source},
			"shell":      config.AnnotatedFieldJSON{Value: a.Shell.Value, Source: a.Shell.Source},
		},
		"auth": map[string]any{
			"mount_home_config": config.AnnotatedFieldJSON{Value: a.MountHomeConfig.Value, Source: a.MountHomeConfig.Source},
			"forward_ssh_agent": config.AnnotatedFieldJSON{Value: a.ForwardSSHAgent.Value, Source: a.ForwardSSHAgent.Source},
			"env_file":          config.AnnotatedFieldJSON{Value: a.EnvFile.Value, Source: a.EnvFile.Source},
			"forward_env":       renderListItemsJSON(a.ForwardEnv),
		},
		"workspace": map[string]any{
			"mount_path":   config.AnnotatedFieldJSON{Value: a.MountPath.Value, Source: a.MountPath.Source},
			"persist_home": config.AnnotatedFieldJSON{Value: a.PersistHome.Value, Source: a.PersistHome.Source},
			"extra_mounts": renderListItemsJSON(a.ExtraMounts),
			"ports":        renderListItemsJSON(a.Ports),
		},
		"packages": map[string]any{
			"apt": renderListItemsJSON(a.AptPackages),
			"pip": renderListItemsJSON(a.PipPackages),
			"npm": renderListItemsJSON(a.NpmPackages),
		},
		"resources": map[string]any{
			"memory":     config.AnnotatedFieldJSON{Value: a.Memory.Value, Source: a.Memory.Source},
			"cpus":       config.AnnotatedFieldJSON{Value: a.Cpus.Value, Source: a.Cpus.Source},
			"pids_limit": config.AnnotatedFieldJSON{Value: a.PidsLimit.Value, Source: a.PidsLimit.Source},
		},
		"network": map[string]any{
			"mode":        config.AnnotatedFieldJSON{Value: a.NetworkMode.Value, Source: a.NetworkMode.Source},
			"http_proxy":  config.AnnotatedFieldJSON{Value: a.HTTPProxy.Value, Source: a.HTTPProxy.Source},
			"https_proxy": config.AnnotatedFieldJSON{Value: a.HTTPSProxy.Value, Source: a.HTTPSProxy.Source},
			"no_proxy":    config.AnnotatedFieldJSON{Value: a.NoProxy.Value, Source: a.NoProxy.Source},
			"allow":       renderListItemsJSON(a.Allow),
			"deny":        renderListItemsJSON(a.Deny),
		},
		"hooks": map[string]any{
			"pre_build": renderListItemsJSON(a.PreBuild),
			"post_stop": renderListItemsJSON(a.PostStop),
		},
		"harness": map[string]any{
			"version":            config.AnnotatedFieldJSON{Value: a.HarnessVersion.Value, Source: a.HarnessVersion.Source},
			"skip_permissions":   config.AnnotatedFieldJSON{Value: a.SkipPermissions.Value, Source: a.SkipPermissions.Source},
			"node_version":       config.AnnotatedFieldJSON{Value: a.NodeVersion.Value, Source: a.NodeVersion.Source},
			"python_version":     config.AnnotatedFieldJSON{Value: a.PythonVersion.Value, Source: a.PythonVersion.Source},
			"entrypoint_command": config.AnnotatedFieldJSON{Value: a.EntrypointCommand.Value, Source: a.EntrypointCommand.Source},
			"health_check":       config.AnnotatedFieldJSON{Value: a.CustomHealthCheck.Value, Source: a.CustomHealthCheck.Source},
			"extra_args":         renderListItemsJSON(a.ExtraArgs),
			"install_commands":   renderListItemsJSON(a.InstallCommands),
			"config_dirs":        renderListItemsJSON(a.ConfigDirs),
			"required_env":       renderListItemsJSON(a.RequiredEnv),
			"shell_rc":           renderListItemsJSON(a.CustomShellRC),
			"aliases":            renderAliasesJSON(a.CustomAliases),
		},
	}

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	return string(b)
}

// renderListItemsJSON converts annotated list items to JSON-serialisable form.
// Each item gets {"value": "...", "source": "..."}.
func renderListItemsJSON(items []config.AnnotatedListItem) []config.AnnotatedFieldJSON {
	if len(items) == 0 {
		return []config.AnnotatedFieldJSON{}
	}
	result := make([]config.AnnotatedFieldJSON, len(items))
	for i, item := range items {
		result[i] = config.AnnotatedFieldJSON{Value: item.Value, Source: item.Source}
	}
	return result
}

// renderAliasesJSON converts the aliases map to JSON-serialisable form.
func renderAliasesJSON(aliases map[string]config.AnnotatedField[string]) map[string]config.AnnotatedFieldJSON {
	if len(aliases) == 0 {
		return map[string]config.AnnotatedFieldJSON{}
	}
	result := make(map[string]config.AnnotatedFieldJSON, len(aliases))
	for k, v := range aliases {
		result[k] = config.AnnotatedFieldJSON{Value: v.Value, Source: v.Source}
	}
	return result
}
