package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/peasant-labs/zone/internal/scaffold"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCreatesZoneToml(t *testing.T) {
	binary := getZoneBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(binary, "init", "--harness", "claude-code")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+dir+"/no-xdg")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "init failed: %s", string(out))
	assert.Contains(t, string(out), "Created zone.toml")

	data, err := os.ReadFile(filepath.Join(dir, "zone.toml"))
	require.NoError(t, err, "zone.toml not created")
	assert.Contains(t, string(data), `harness = "claude-code"`)
	assert.Contains(t, string(data), "version = 1")

	skillData, err := os.ReadFile(filepath.Join(dir, scaffold.AgentSkillsDir, scaffold.AgentZoneSkillFile))
	require.NoError(t, err, "agent skill not created")
	assert.Contains(t, string(skillData), "Zone Workspace Dependencies")
	assert.Contains(t, string(skillData), "`zone.toml`")
}

func TestInitExistingZoneToml(t *testing.T) {
	binary := getZoneBinary(t)
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "zone.toml"), []byte("version = 1\nharness = \"claude-code\"\n"), 0644))

	cmd := exec.Command(binary, "init", "--harness", "claude-code")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+dir+"/no-xdg")
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "zone.toml already exists")
}

func TestInitNoHarness(t *testing.T) {
	binary := getZoneBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(binary, "init")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+dir+"/no-xdg")
	cmd.Stdin = strings.NewReader("") // non-TTY stdin

	var stdout strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	require.Error(t, err)
	assert.Contains(t, stderr.String(), "no --harness specified")
	assert.Contains(t, stderr.String(), "--harness <name>")
}

func TestInitSetFlag(t *testing.T) {
	binary := getZoneBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(binary, "init", "--harness", "claude-code", "--set", "memory=8g")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+dir+"/no-xdg")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "init --set failed: %s", string(out))

	data, err := os.ReadFile(filepath.Join(dir, "zone.toml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `memory = "8g"`)
}

func TestLsNotStub(t *testing.T) {
	binary := getZoneBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(binary, "ls")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+dir+"/no-xdg")
	out, _ := cmd.CombinedOutput()
	assert.NotContains(t, string(out), "not implemented")
}

func TestLogsNotStub(t *testing.T) {
	binary := getZoneBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(binary, "logs")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+dir+"/no-xdg")
	out, _ := cmd.CombinedOutput()
	assert.NotContains(t, string(out), "not implemented")
}

func TestLogsBuildNoLog(t *testing.T) {
	binary := getZoneBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(binary, "logs", "--build")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+dir+"/no-xdg")
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "no build log found")
}

func TestStatusNotStub(t *testing.T) {
	binary := getZoneBinary(t)
	dir := t.TempDir()

	cmd := exec.Command(binary, "status")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+dir+"/no-xdg")
	out, _ := cmd.CombinedOutput()
	assert.NotContains(t, string(out), "not implemented")
}

func TestLsHelpShowsFlags(t *testing.T) {
	binary := getZoneBinary(t)
	out, err := exec.Command(binary, "ls", "--help").CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(out), "--json")
	assert.Contains(t, string(out), "--running")
	assert.Contains(t, string(out), "--quiet")
}

func TestLogsHelpShowsFlags(t *testing.T) {
	binary := getZoneBinary(t)
	out, err := exec.Command(binary, "logs", "--help").CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(out), "--follow")
	assert.Contains(t, string(out), "--build")
	assert.Contains(t, string(out), "--tail")
}
