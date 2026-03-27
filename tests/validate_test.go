// Tests for dangerous mount detection and symlink resolution.
package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/peasant-labs/zone/internal/config"
)

// makeRepoWithMounts creates a minimal MergedConfig with the given extra_mounts.
func makeRepoWithMounts(mounts []string) *config.MergedConfig {
	m := &config.MergedConfig{}
	m.Workspace.ExtraMounts = mounts
	return m
}

// TestUnknownKeySuggestion_Close: "baes_image" in [zone] -> suggests "base_image"
func TestUnknownKeySuggestion_Close(t *testing.T) {
	suggestion, found := config.SuggestKey("zone.baes_image")
	if !found {
		t.Fatal("expected suggestion for 'zone.baes_image', got none")
	}
	if !strings.Contains(suggestion, "base_image") {
		t.Errorf("suggestion %q should contain 'base_image'", suggestion)
	}
}

// TestUnknownKeySuggestion_Far: "xyzzy" -> no suggestion (distance > 3)
func TestUnknownKeySuggestion_Far(t *testing.T) {
	_, found := config.SuggestKey("xyzzy")
	if found {
		t.Error("expected no suggestion for 'xyzzy', got one")
	}
}

// TestUnknownKeySuggestion_BareKey: "skip_permissions" at top level -> suggests "[harness] skip_permissions"
func TestUnknownKeySuggestion_BareKey(t *testing.T) {
	suggestion, found := config.SuggestKey("skip_permissions")
	if !found {
		t.Fatal("expected suggestion for 'skip_permissions', got none")
	}
	formatted := config.FormatSuggestion(suggestion)
	if !strings.Contains(formatted, "harness") {
		t.Errorf("suggestion %q should reference 'harness'", formatted)
	}
	if !strings.Contains(formatted, "skip_permissions") {
		t.Errorf("suggestion %q should contain 'skip_permissions'", formatted)
	}
}

// TestUnknownKeySuggestion_SectionAware: "harness.skip_perms" -> suggests "[harness] skip_permissions"
func TestUnknownKeySuggestion_SectionAware(t *testing.T) {
	suggestion, found := config.SuggestKey("harness.skip_perms")
	if !found {
		t.Fatal("expected suggestion for 'harness.skip_perms', got none")
	}
	formatted := config.FormatSuggestion(suggestion)
	if !strings.Contains(formatted, "harness") {
		t.Errorf("suggestion %q should reference 'harness'", formatted)
	}
	if !strings.Contains(formatted, "skip_permissions") {
		t.Errorf("suggestion %q should contain 'skip_permissions'", formatted)
	}
}

// TestDangerousMount_DockerSocket: extra_mounts=["/var/run/docker.sock:/docker.sock"]
// -> DangerousMountError with reason about Docker socket
func TestDangerousMount_DockerSocket(t *testing.T) {
	cfg := makeRepoWithMounts([]string{"/var/run/docker.sock:/docker.sock"})
	errs := config.Validate(cfg)
	var found bool
	for _, e := range errs {
		if e.Category == "dangerous_mount" && strings.Contains(e.Message, "docker") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected dangerous_mount error for docker socket, got %v", errs)
	}
}

// TestDangerousMount_SSHDir: extra_mounts=["~/.ssh:/ssh"] -> blocked
func TestDangerousMount_SSHDir(t *testing.T) {
	home, _ := os.UserHomeDir()
	sshDir := filepath.Join(home, ".ssh")
	cfg := makeRepoWithMounts([]string{sshDir + ":/ssh"})
	errs := config.Validate(cfg)
	var found bool
	for _, e := range errs {
		if e.Category == "dangerous_mount" && strings.Contains(strings.ToLower(e.Message), "ssh") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected dangerous_mount error for ~/.ssh, got %v", errs)
	}
}

// TestDangerousMount_SymlinkResolution: symlink to /var/run/docker.sock -> blocked
func TestDangerousMount_SymlinkResolution(t *testing.T) {
	tmp := t.TempDir()
	link := filepath.Join(tmp, "sneaky.sock")
	if err := os.Symlink("/var/run/docker.sock", link); err != nil {
		t.Skip("cannot create symlink:", err)
	}
	cfg := makeRepoWithMounts([]string{link + ":/docker.sock"})
	errs := config.Validate(cfg)
	var found bool
	for _, e := range errs {
		if e.Category == "dangerous_mount" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected dangerous_mount error for symlink to docker socket, got %v", errs)
	}
}

// TestDangerousMount_NonExistentPath: non-existent path -> no error
func TestDangerousMount_NonExistentPath(t *testing.T) {
	cfg := makeRepoWithMounts([]string{"/nonexistent-path-xyz/abc:/target"})
	errs := config.Validate(cfg)
	for _, e := range errs {
		if e.Category == "dangerous_mount" {
			t.Errorf("unexpected dangerous_mount error for non-existent path: %s", e.Message)
		}
	}
}

// TestDangerousMount_AllCollected: two dangerous mounts -> both reported
func TestDangerousMount_AllCollected(t *testing.T) {
	cfg := makeRepoWithMounts([]string{
		"/var/run/docker.sock:/docker.sock",
		"/proc:/proc",
	})
	errs := config.Validate(cfg)
	var count int
	for _, e := range errs {
		if e.Category == "dangerous_mount" {
			count++
		}
	}
	if count < 2 {
		t.Errorf("expected 2 dangerous_mount errors, got %d: %v", count, errs)
	}
}

// TestMountReadOnly_NoSuffix: "/host:/container" -> normalized to "/host:/container:ro"
func TestMountReadOnly_NoSuffix(t *testing.T) {
	normalized, err := config.NormalizeMountPermission("/host:/container")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if normalized != "/host:/container:ro" {
		t.Errorf("normalized = %q, want %q", normalized, "/host:/container:ro")
	}
}

// TestMountReadOnly_ExplicitRO: "/host:/container:ro" -> unchanged
func TestMountReadOnly_ExplicitRO(t *testing.T) {
	normalized, err := config.NormalizeMountPermission("/host:/container:ro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if normalized != "/host:/container:ro" {
		t.Errorf("normalized = %q, want %q", normalized, "/host:/container:ro")
	}
}

// TestMountReadOnly_ExplicitRW: "/host:/container:rw" -> unchanged
func TestMountReadOnly_ExplicitRW(t *testing.T) {
	normalized, err := config.NormalizeMountPermission("/host:/container:rw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if normalized != "/host:/container:rw" {
		t.Errorf("normalized = %q, want %q", normalized, "/host:/container:rw")
	}
}

// TestMountReadOnly_InvalidPerm: "/host:/container:wx" -> validation error
func TestMountReadOnly_InvalidPerm(t *testing.T) {
	_, err := config.NormalizeMountPermission("/host:/container:wx")
	if err == nil {
		t.Error("expected error for invalid permission 'wx', got nil")
	}
}

// TestBaseImageWarning: base_image="ubuntu" (no tag) -> warning (not error)
func TestBaseImageWarning(t *testing.T) {
	cfg := &config.MergedConfig{}
	cfg.Zone.BaseImage = "ubuntu"
	errs := config.Validate(cfg)
	var found bool
	for _, e := range errs {
		if e.Category == "warning" && strings.Contains(e.Message, "base_image") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for base_image without tag, got %v", errs)
	}
	// Ensure it's a warning, not an error
	if errs.HasErrors() {
		t.Error("expected only warnings (no errors) for base_image without tag")
	}
}

// TestNetworkModeNoneWithAllow: network.mode="none" with non-empty allow list -> warning
func TestNetworkModeNoneWithAllow(t *testing.T) {
	cfg := &config.MergedConfig{}
	cfg.Network.Mode = "none"
	cfg.Network.Allow = []string{"github.com"}
	errs := config.Validate(cfg)
	var found bool
	for _, e := range errs {
		if e.Category == "warning" && strings.Contains(e.Message, "allow") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for network.mode=none with allow list, got %v", errs)
	}
}

// TestMultipleErrors: config with unknown key + dangerous mount -> both errors collected
func TestMultipleErrors(t *testing.T) {
	cfg := makeRepoWithMounts([]string{"/var/run/docker.sock:/docker.sock"})
	cfg.Zone.BaseImage = "ubuntu" // adds warning

	errs := config.Validate(cfg)
	if len(errs) < 2 {
		t.Errorf("expected at least 2 issues (dangerous_mount + warning), got %d: %v", len(errs), errs)
	}

	// ValidateUnknownKeys should also work
	keyErrs := config.ValidateUnknownKeys([]string{"baes_image", "xyzzy"}, "zone.toml")
	if len(keyErrs) != 2 {
		t.Errorf("expected 2 unknown key errors, got %d", len(keyErrs))
	}
	// "baes_image" has suggestion
	if !strings.Contains(keyErrs[0].Message, "Did you mean") {
		t.Errorf("expected suggestion in error for 'baes_image', got: %s", keyErrs[0].Message)
	}
}
