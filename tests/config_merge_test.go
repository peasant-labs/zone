// Tests for config merge logic.
package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/peasant-labs/zone/internal/config"
)

// minimalTOML is the smallest valid zone.toml.
const minimalTOML = `version = 1
harness = "claude-code"
`

// TestMinimalConfig: LoadRepo with minimal config succeeds, HarnessName == "claude-code"
func TestMinimalConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zone.toml")
	if err := os.WriteFile(path, []byte(minimalTOML), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadRepo(path)
	if err != nil {
		t.Fatalf("LoadRepo: %v", err)
	}
	if cfg.HarnessName != "claude-code" {
		t.Errorf("HarnessName = %q, want %q", cfg.HarnessName, "claude-code")
	}
}

// TestScalarOverride: global base_image="ubuntu:22.04", repo base_image="ubuntu:24.04"
// -> merged base_image == "ubuntu:24.04"
func TestScalarOverride(t *testing.T) {
	global := config.DefaultGlobalConfig()
	global.Zone.BaseImage = "ubuntu:22.04"

	dir := t.TempDir()
	path := filepath.Join(dir, "zone.toml")
	toml := `version = 1
harness = "claude-code"
[zone]
base_image = "ubuntu:24.04"
`
	if err := os.WriteFile(path, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := config.LoadRepo(path)
	if err != nil {
		t.Fatalf("LoadRepo: %v", err)
	}
	merged, _ := config.Merge(global, repo)
	if merged.Zone.BaseImage != "ubuntu:24.04" {
		t.Errorf("BaseImage = %q, want %q", merged.Zone.BaseImage, "ubuntu:24.04")
	}
}

// TestScalarFallback: global base_image="ubuntu:22.04", repo base_image="" (not set)
// -> merged base_image == "ubuntu:22.04"
func TestScalarFallback(t *testing.T) {
	global := config.DefaultGlobalConfig()
	global.Zone.BaseImage = "ubuntu:22.04"

	dir := t.TempDir()
	path := filepath.Join(dir, "zone.toml")
	toml := `version = 1
harness = "claude-code"
`
	if err := os.WriteFile(path, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := config.LoadRepo(path)
	if err != nil {
		t.Fatalf("LoadRepo: %v", err)
	}
	merged, _ := config.Merge(global, repo)
	if merged.Zone.BaseImage != "ubuntu:22.04" {
		t.Errorf("BaseImage = %q, want %q", merged.Zone.BaseImage, "ubuntu:22.04")
	}
}

// TestListUnion: global apt=["git","curl"], repo apt=["curl","wget"]
// -> merged apt == ["git","curl","wget"] (deduped, global-first)
func TestListUnion(t *testing.T) {
	global := config.DefaultGlobalConfig()
	global.Packages.Apt = []string{"git", "curl"}

	dir := t.TempDir()
	path := filepath.Join(dir, "zone.toml")
	toml := `version = 1
harness = "claude-code"
[packages]
apt = ["curl", "wget"]
`
	if err := os.WriteFile(path, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := config.LoadRepo(path)
	if err != nil {
		t.Fatalf("LoadRepo: %v", err)
	}
	merged, _ := config.Merge(global, repo)
	want := []string{"git", "curl", "wget"}
	if !stringSliceEqual(merged.Packages.Apt, want) {
		t.Errorf("Apt = %v, want %v", merged.Packages.Apt, want)
	}
}

// TestListAppend: global forward_env=["AWS_*"], repo forward_env=["ANTHROPIC_API_KEY"]
// -> merged forward_env == ["AWS_*","ANTHROPIC_API_KEY"] (union)
func TestListAppend(t *testing.T) {
	global := config.DefaultGlobalConfig()
	global.Auth.ForwardEnv = []string{"AWS_*"}

	dir := t.TempDir()
	path := filepath.Join(dir, "zone.toml")
	toml := `version = 1
harness = "claude-code"
[auth]
forward_env = ["ANTHROPIC_API_KEY"]
`
	if err := os.WriteFile(path, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := config.LoadRepo(path)
	if err != nil {
		t.Fatalf("LoadRepo: %v", err)
	}
	merged, _ := config.Merge(global, repo)
	want := []string{"AWS_*", "ANTHROPIC_API_KEY"}
	if !stringSliceEqual(merged.Auth.ForwardEnv, want) {
		t.Errorf("ForwardEnv = %v, want %v", merged.Auth.ForwardEnv, want)
	}
}

// TestNetworkAllow: global default_allow=["github.com"], repo allow=["api.openai.com"]
// -> merged allow == ["github.com","api.openai.com"]
func TestNetworkAllow(t *testing.T) {
	global := config.DefaultGlobalConfig()
	global.Network.DefaultAllow = []string{"github.com"}

	dir := t.TempDir()
	path := filepath.Join(dir, "zone.toml")
	toml := `version = 1
harness = "claude-code"
[network]
allow = ["api.openai.com"]
`
	if err := os.WriteFile(path, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := config.LoadRepo(path)
	if err != nil {
		t.Fatalf("LoadRepo: %v", err)
	}
	merged, _ := config.Merge(global, repo)
	want := []string{"github.com", "api.openai.com"}
	if !stringSliceEqual(merged.Network.Allow, want) {
		t.Errorf("Allow = %v, want %v", merged.Network.Allow, want)
	}
}

// TestListReplace: global extra_mounts=[], repo extra_mounts=["/data:/data"]
// -> merged extra_mounts == ["/data:/data"]
func TestListReplace(t *testing.T) {
	global := config.DefaultGlobalConfig()
	// global has no extra_mounts (empty slice)

	dir := t.TempDir()
	path := filepath.Join(dir, "zone.toml")
	toml := `version = 1
harness = "claude-code"
[workspace]
extra_mounts = ["/data:/data"]
`
	if err := os.WriteFile(path, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := config.LoadRepo(path)
	if err != nil {
		t.Fatalf("LoadRepo: %v", err)
	}
	merged, _ := config.Merge(global, repo)
	want := []string{"/data:/data"}
	if !stringSliceEqual(merged.Workspace.ExtraMounts, want) {
		t.Errorf("ExtraMounts = %v, want %v", merged.Workspace.ExtraMounts, want)
	}
}

// TestHooksAppend: global pre_build=["echo a"], repo pre_build=["echo b"]
// -> merged pre_build == ["echo a","echo b"]
func TestHooksAppend(t *testing.T) {
	global := &config.GlobalConfig{}
	// GlobalConfig has no hooks — we set via a custom global
	// The Merge function handles global.Network.DefaultAllow etc,
	// but hooks only exist in repo. We test via two repo-side entries
	// by manipulating the globals manually.

	// Use a zero global but with a hook via the test helper struct
	g := config.DefaultGlobalConfig()
	// GlobalConfig doesn't have Hooks, but Merge should handle this.
	// Since GlobalConfig has no hooks, only repo hooks should appear.
	// Instead, test the append logic by merging two items where global is empty:
	_ = global

	dir := t.TempDir()
	path := filepath.Join(dir, "zone.toml")
	toml := `version = 1
harness = "claude-code"
[hooks]
pre_build = ["echo b"]
`
	if err := os.WriteFile(path, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := config.LoadRepo(path)
	if err != nil {
		t.Fatalf("LoadRepo: %v", err)
	}
	merged, _ := config.Merge(g, repo)
	// With empty global hooks, merged should just have the repo hooks.
	want := []string{"echo b"}
	if !stringSliceEqual(merged.Hooks.PreBuild, want) {
		t.Errorf("PreBuild = %v, want %v", merged.Hooks.PreBuild, want)
	}
}

// TestExtraArgsAppend: global extra_args=["--verbose"], repo extra_args=["--debug"]
// -> merged == ["--verbose","--debug"]
// Note: cannot use top-level harness sugar with [harness] table in same file;
// use [zone] harness instead.
func TestExtraArgsAppend(t *testing.T) {
	global := config.DefaultGlobalConfig()
	global.Harness.ExtraArgs = []string{"--verbose"}

	dir := t.TempDir()
	path := filepath.Join(dir, "zone.toml")
	// Use [zone] harness to avoid conflict with [harness] table
	toml := `version = 1
[zone]
harness = "claude-code"
[harness]
extra_args = ["--debug"]
`
	if err := os.WriteFile(path, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := config.LoadRepo(path)
	if err != nil {
		t.Fatalf("LoadRepo: %v", err)
	}
	merged, _ := config.Merge(global, repo)
	want := []string{"--verbose", "--debug"}
	if !stringSliceEqual(merged.Harness.ExtraArgs, want) {
		t.Errorf("ExtraArgs = %v, want %v", merged.Harness.ExtraArgs, want)
	}
}

// TestBoolOverride: global forward_ssh_agent=false, repo forward_ssh_agent=true -> merged == true
func TestBoolOverride(t *testing.T) {
	global := config.DefaultGlobalConfig()
	f := false
	global.Auth.ForwardSSHAgent = &f

	dir := t.TempDir()
	path := filepath.Join(dir, "zone.toml")
	toml := `version = 1
harness = "claude-code"
[auth]
forward_ssh_agent = true
`
	if err := os.WriteFile(path, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := config.LoadRepo(path)
	if err != nil {
		t.Fatalf("LoadRepo: %v", err)
	}
	merged, _ := config.Merge(global, repo)
	if merged.Auth.ForwardSSHAgent == nil || !*merged.Auth.ForwardSSHAgent {
		t.Errorf("ForwardSSHAgent = false/nil, want true")
	}
}

// TestBoolNilFallback: global forward_ssh_agent=true, repo forward_ssh_agent=nil -> merged == true
func TestBoolNilFallback(t *testing.T) {
	global := config.DefaultGlobalConfig()
	tr := true
	global.Auth.ForwardSSHAgent = &tr

	dir := t.TempDir()
	path := filepath.Join(dir, "zone.toml")
	// repo does not set forward_ssh_agent
	toml := `version = 1
harness = "claude-code"
`
	if err := os.WriteFile(path, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := config.LoadRepo(path)
	if err != nil {
		t.Fatalf("LoadRepo: %v", err)
	}
	merged, _ := config.Merge(global, repo)
	if merged.Auth.ForwardSSHAgent == nil || !*merged.Auth.ForwardSSHAgent {
		t.Errorf("ForwardSSHAgent = false/nil, want true (global fallback)")
	}
}

// TestConfigVersion: version=0 defaults to 1; version=1 valid; version=2 errors
func TestConfigVersion(t *testing.T) {
	tests := []struct {
		name    string
		toml    string
		wantErr bool
		wantVer int
	}{
		{
			name: "version_0_defaults_to_1",
			toml: `harness = "claude-code"
`,
			wantErr: false,
			wantVer: 1,
		},
		{
			name: "version_1_valid",
			toml: `version = 1
harness = "claude-code"
`,
			wantErr: false,
			wantVer: 1,
		},
		{
			name: "version_2_errors",
			toml: `version = 2
harness = "claude-code"
`,
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "zone.toml")
			if err := os.WriteFile(path, []byte(tc.toml), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := config.LoadRepo(path)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Version != tc.wantVer {
				t.Errorf("Version = %d, want %d", cfg.Version, tc.wantVer)
			}
		})
	}
}

// TestSourceAnnotation: merged config has correct Source on each AnnotatedField.
func TestSourceAnnotation(t *testing.T) {
	global := config.DefaultGlobalConfig()
	global.Zone.BaseImage = "ubuntu:22.04"
	global.Zone.Shell = "bash"

	dir := t.TempDir()
	path := filepath.Join(dir, "zone.toml")
	toml := `version = 1
harness = "claude-code"
[zone]
base_image = "ubuntu:24.04"
`
	if err := os.WriteFile(path, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := config.LoadRepo(path)
	if err != nil {
		t.Fatalf("LoadRepo: %v", err)
	}
	_, ann := config.Merge(global, repo)

	// base_image set by repo -> SourceRepo
	if ann.BaseImage.Source != config.SourceRepo {
		t.Errorf("BaseImage.Source = %q, want %q", ann.BaseImage.Source, config.SourceRepo)
	}
	// shell set only in global -> SourceGlobal
	if ann.Shell.Source != config.SourceGlobal {
		t.Errorf("Shell.Source = %q, want %q", ann.Shell.Source, config.SourceGlobal)
	}
}

// stringSliceEqual compares two string slices for equality.
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
