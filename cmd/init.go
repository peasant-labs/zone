package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/peasant-labs/zone/internal/cache"
	"github.com/peasant-labs/zone/internal/tui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a zone.toml in the current directory",
	Long: `Scaffold a zone.toml configuration file in the current directory.

Detects existing harness indicators (.claude/, CLAUDE.md, .aider*)
and suggests the appropriate harness. Use --set to customize the
generated config without editing the file afterward.`,
	Example: `  zone init --harness claude-code
  zone init --harness aider --set resources.memory=8g
  zone init --harness claude-code --set resources.cpus=4`,
	RunE: func(cmd *cobra.Command, args []string) error {
		harnessName, _ := cmd.Flags().GetString("harness")

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}

		if harnessName == "" {
			plainFlag, _ := cmd.Root().PersistentFlags().GetBool("plain")
			if !tui.IsTTY(plainFlag) {
				return fmt.Errorf("no --harness specified and stdin is not a terminal.\n\n" +
					"  Use `zone init --harness <name>` for non-interactive init.\n" +
					"  Available: claude-code, opencode, aider, gemini-cli, codex-cli, custom")
			}

			// Build detection hints map for wizard
			detected := buildDetectionMap(cwd)

			// Launch BubbleTea init wizard via panic-safe RunTUI (D-27)
			wizard := tui.NewInitWizard(detected)
			finalModel, err := tui.RunTUI(wizard)
			if err != nil {
				return fmt.Errorf("init wizard: %w", err)
			}

			result := finalModel.(tui.InitWizard)
			if result.Cancelled {
				return fmt.Errorf("init cancelled")
			}
			if result.Err != nil {
				return result.Err
			}
			harnessName = result.SelectedHarness
		}

		tomlPath := filepath.Join(cwd, "zone.toml")
		if _, err := os.Stat(tomlPath); err == nil {
			return fmt.Errorf("zone.toml already exists. Use `zone config` to modify it")
		}

		detectHarnessHints(cmd, cwd)

		content := generateInitTemplate(harnessName)
		setFlags, _ := cmd.Flags().GetStringArray("set")
		for _, kv := range setFlags {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid --set format %q, expected key=value", kv)
			}
			content = applySetOverride(content, parts[0], parts[1])
		}

		if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write zone.toml: %w", err)
		}

		if err := cache.EnsureGitignore(cwd); err != nil {
			return fmt.Errorf("update gitignore: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Created zone.toml with harness %q\n", harnessName)
		return nil
	},
}

func init() {
	initCmd.Flags().String("harness", "", "Harness to configure (e.g., claude-code, aider)")
	initCmd.Flags().StringArray("set", nil, "Override config value (e.g., --set resources.memory=8g)")
}

// buildDetectionMap checks for harness indicator files and returns a map
// of harness name -> detected. Used by the TUI wizard to show "* detected" hints.
func buildDetectionMap(dir string) map[string]bool {
	detected := make(map[string]bool)

	hints := []struct {
		pattern string
		harness string
		isDir   bool
	}{
		{".claude", "claude-code", true},
		{"CLAUDE.md", "claude-code", false},
		{".opencode", "opencode", true},
	}

	for _, h := range hints {
		path := filepath.Join(dir, h.pattern)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if h.isDir && !info.IsDir() {
			continue
		}
		detected[h.harness] = true
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".aider") {
			detected["aider"] = true
			break
		}
	}

	return detected
}

func detectHarnessHints(cmd *cobra.Command, dir string) {
	hints := []struct {
		pattern string
		harness string
		isDir   bool
	}{
		{pattern: ".claude", harness: "claude-code", isDir: true},
		{pattern: "CLAUDE.md", harness: "claude-code", isDir: false},
		{pattern: ".opencode", harness: "opencode", isDir: true},
	}

	for _, h := range hints {
		path := filepath.Join(dir, h.pattern)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if h.isDir && !info.IsDir() {
			continue
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Detected: %s indicators (%s)\n", h.harness, h.pattern)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".aider") {
			fmt.Fprintf(cmd.ErrOrStderr(), "Detected: aider indicators (%s)\n", e.Name())
			break
		}
	}
}

func generateInitTemplate(harness string) string {
	return fmt.Sprintf(`version = 1
harness = %q

# ---- Zone configuration ----
# [zone]
# base_image = "ubuntu:24.04"
# shell = "bash"
# packages = ["git", "curl", "ripgrep"]
# extra_args = []

# ---- Resource limits ----
# [resources]
# memory = "4g"
# cpus = "2"
# pids_limit = 512

# ---- Workspace settings ----
# [workspace]
# persist_home = true
# extra_mounts = ["./data:/mnt/data:ro"]
# ports = ["3000:3000"]

# ---- Authentication & environment ----
# [auth]
# forward_env = ["AWS_*", "GITHUB_TOKEN"]
# required_env = []
# forward_ssh_agent = false
# mount_home_config = true
# env_file = ""

# ---- Network settings ----
# [network]
# mode = "none"
# allow = []
# deny = []

# ---- Hooks ----
# [hooks]
# pre_build = []
# post_stop = []

# ---- Harness-specific settings ----
# [harness]
# skip_permissions = false
# api_key_env = "ANTHROPIC_API_KEY"
# model = ""
`, harness)
}

func applySetOverride(content, key, value string) string {
	parts := strings.SplitN(key, ".", 2)
	field := parts[0]
	if len(parts) == 2 {
		field = parts[1]
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		prefix := "# " + field + " = "
		if strings.HasPrefix(trimmed, prefix) {
			lines[i] = field + " = " + formatTOMLValue(value)
			break
		}
	}

	return strings.Join(lines, "\n")
}

func formatTOMLValue(v string) string {
	if v == "true" || v == "false" {
		return v
	}
	if v == "" {
		return `""`
	}
	for _, c := range v {
		if (c < '0' || c > '9') && c != '.' {
			return fmt.Sprintf("%q", v)
		}
	}
	return v
}
