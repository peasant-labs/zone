// env_test.go tests environment variable collection, .env file parsing, and
// required env validation logic.
package docker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── CollectForwardedEnv ────────────────────────────────────────────────────

// TestCollectForwardedEnv_GlobPattern verifies that a wildcard pattern like
// "AWS_*" matches all env vars with the given prefix.
func TestCollectForwardedEnv_GlobPattern(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "AKID123")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "SECRET456")

	vars, warnings := CollectForwardedEnv([]string{"AWS_*"})

	assert.Empty(t, warnings, "no warnings expected when pattern matches")
	assert.Len(t, vars, 2)
	assert.Contains(t, vars, "AWS_ACCESS_KEY_ID=AKID123")
	assert.Contains(t, vars, "AWS_SECRET_ACCESS_KEY=SECRET456")
}

// TestCollectForwardedEnv_ExactMatch verifies that a pattern without wildcards
// matches an exact variable name.
func TestCollectForwardedEnv_ExactMatch(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")

	vars, warnings := CollectForwardedEnv([]string{"ANTHROPIC_API_KEY"})

	assert.Empty(t, warnings)
	assert.Equal(t, []string{"ANTHROPIC_API_KEY=sk-ant-test"}, vars)
}

// TestCollectForwardedEnv_NoMatch verifies that a pattern matching nothing
// returns an empty slice and a warning message.
func TestCollectForwardedEnv_NoMatch(t *testing.T) {
	vars, warnings := CollectForwardedEnv([]string{"NONEXISTENT_ZXQW_*"})

	assert.Empty(t, vars, "no vars expected when nothing matches")
	assert.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "NONEXISTENT_ZXQW_*")
	assert.Contains(t, warnings[0], "did not match")
}

// TestCollectForwardedEnv_MultiplePatterns verifies that multiple patterns
// each contribute matches to the result without duplicates.
func TestCollectForwardedEnv_MultiplePatterns(t *testing.T) {
	t.Setenv("FOO_A", "1")
	t.Setenv("FOO_B", "2")
	t.Setenv("BAR_X", "3")

	vars, warnings := CollectForwardedEnv([]string{"FOO_*", "BAR_X"})

	assert.Empty(t, warnings)
	assert.Len(t, vars, 3)
	assert.Contains(t, vars, "FOO_A=1")
	assert.Contains(t, vars, "FOO_B=2")
	assert.Contains(t, vars, "BAR_X=3")
}

// TestCollectForwardedEnv_Deduplication verifies that two patterns matching
// the same variable produce only one entry in the result.
func TestCollectForwardedEnv_Deduplication(t *testing.T) {
	t.Setenv("SHARED_VAR", "value")

	vars, warnings := CollectForwardedEnv([]string{"SHARED_VAR", "SHARED_*"})

	assert.Empty(t, warnings)
	assert.Equal(t, 1, len(vars), "duplicate key should appear only once")
}

// TestCollectForwardedEnv_EmptyPatterns verifies that an empty patterns slice
// returns empty results with no warnings.
func TestCollectForwardedEnv_EmptyPatterns(t *testing.T) {
	vars, warnings := CollectForwardedEnv([]string{})

	assert.Empty(t, vars)
	assert.Empty(t, warnings)
}

// TestCollectForwardedEnv_KeyValueFormat verifies the returned strings are in
// "KEY=VALUE" format.
func TestCollectForwardedEnv_KeyValueFormat(t *testing.T) {
	t.Setenv("ZONE_TEST_FMT", "hello=world")

	vars, _ := CollectForwardedEnv([]string{"ZONE_TEST_FMT"})

	require.Len(t, vars, 1)
	assert.Equal(t, "ZONE_TEST_FMT=hello=world", vars[0])
}

// ─── ParseEnvFile ──────────────────────────────────────────────────────────

// TestParseEnvFile_ValidFile verifies that standard KEY=VALUE lines are parsed.
func TestParseEnvFile_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte("FOO=bar\nBAZ=qux\n"), 0600))

	m, err := ParseEnvFile(path)

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"FOO": "bar", "BAZ": "qux"}, m)
}

// TestParseEnvFile_SkipsComments verifies that lines starting with # are ignored.
func TestParseEnvFile_SkipsComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte("# comment\nKEY=val\n"), 0600))

	m, err := ParseEnvFile(path)

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"KEY": "val"}, m)
}

// TestParseEnvFile_SkipsEmptyLines verifies that blank lines are ignored.
func TestParseEnvFile_SkipsEmptyLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte("\n\nKEY=val\n\n"), 0600))

	m, err := ParseEnvFile(path)

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"KEY": "val"}, m)
}

// TestParseEnvFile_SkipsNoEqualsLines verifies that lines without "=" are ignored.
func TestParseEnvFile_SkipsNoEqualsLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte("NOEQUALS\nKEY=val\n"), 0600))

	m, err := ParseEnvFile(path)

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"KEY": "val"}, m)
}

// TestParseEnvFile_EmptyValue verifies that KEY= with empty value is included.
func TestParseEnvFile_EmptyValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte("EMPTY=\n"), 0600))

	m, err := ParseEnvFile(path)

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"EMPTY": ""}, m)
}

// TestParseEnvFile_MissingFile verifies that a non-existent path returns an error.
func TestParseEnvFile_MissingFile(t *testing.T) {
	_, err := ParseEnvFile("/nonexistent/path/.env")

	assert.Error(t, err)
}

// TestParseEnvFile_ValueWithEquals verifies that values containing "=" keep the
// full value (only the first "=" is the separator).
func TestParseEnvFile_ValueWithEquals(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte("BASE64=abc=def==\n"), 0600))

	m, err := ParseEnvFile(path)

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"BASE64": "abc=def=="}, m)
}

// ─── ValidateRequiredEnv ──────────────────────────────────────────────────

// TestValidateRequiredEnv_AllPresent verifies nil return when all required vars
// are set in the host environment.
func TestValidateRequiredEnv_AllPresent(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")

	err := ValidateRequiredEnv([]string{"ANTHROPIC_API_KEY"}, "claude-code", "", t.TempDir())

	assert.NoError(t, err)
}

// TestValidateRequiredEnv_MissingVar verifies that a missing required var
// returns an error containing the var name and harness name.
func TestValidateRequiredEnv_MissingVar(t *testing.T) {
	// Ensure the var is NOT set
	os.Unsetenv("ZONE_MISSING_TEST_VAR_XYZ")

	err := ValidateRequiredEnv([]string{"ZONE_MISSING_TEST_VAR_XYZ"}, "my-harness", "", t.TempDir())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ZONE_MISSING_TEST_VAR_XYZ")
	assert.Contains(t, err.Error(), "my-harness")
}

// TestValidateRequiredEnv_SatisfiedByEnvFile verifies that a required var
// present in the .env file (but not host env) is accepted.
func TestValidateRequiredEnv_SatisfiedByEnvFile(t *testing.T) {
	os.Unsetenv("SECRET_FROM_FILE")
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(envFile, []byte("SECRET_FROM_FILE=myvalue\n"), 0600))

	err := ValidateRequiredEnv([]string{"SECRET_FROM_FILE"}, "custom", ".env", dir)

	assert.NoError(t, err)
}

// TestValidateRequiredEnv_EmptyRequired verifies that an empty required slice
// always returns nil.
func TestValidateRequiredEnv_EmptyRequired(t *testing.T) {
	err := ValidateRequiredEnv([]string{}, "some-harness", "", t.TempDir())

	assert.NoError(t, err)
}

// TestValidateRequiredEnv_EmptyEnvFilePath verifies that when no env_file is
// configured, only host env is checked.
func TestValidateRequiredEnv_EmptyEnvFilePath(t *testing.T) {
	t.Setenv("HOST_ONLY_VAR", "present")

	err := ValidateRequiredEnv([]string{"HOST_ONLY_VAR"}, "test-harness", "", t.TempDir())

	assert.NoError(t, err)
}

// TestValidateRequiredEnv_MissingEnvFile verifies that when env_file is set but
// the file does not exist, an error is returned.
func TestValidateRequiredEnv_MissingEnvFile(t *testing.T) {
	dir := t.TempDir()
	// .env file is not created — it's missing

	err := ValidateRequiredEnv([]string{"SOME_VAR"}, "custom", ".env", dir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse env file")
}
