package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureAgentSkillCreatesHarnessNeutralSkill(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, EnsureAgentSkill(dir))

	data, err := os.ReadFile(filepath.Join(dir, AgentSkillsDir, AgentZoneSkillFile))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "# "+AgentZoneSkillTitle)
	assert.Contains(t, content, "Use this skill when a workspace needs")
	assert.Contains(t, content, "`zone.toml`")
	assert.Contains(t, content, "`[packages].apt`")
	assert.Contains(t, content, "`zone validate`")
	assert.NotContains(t, content, "claude")
	assert.NotContains(t, content, "codex")
}

func TestEnsureAgentSkillDoesNotOverwriteExistingFile(t *testing.T) {
	dir := t.TempDir()
	skillPath := filepath.Join(dir, AgentSkillsDir, AgentZoneSkillFile)
	require.NoError(t, os.MkdirAll(filepath.Dir(skillPath), 0755))
	require.NoError(t, os.WriteFile(skillPath, []byte("custom skill\n"), 0644))

	require.NoError(t, EnsureAgentSkill(dir))

	data, err := os.ReadFile(skillPath)
	require.NoError(t, err)
	assert.Equal(t, "custom skill\n", string(data))
}
