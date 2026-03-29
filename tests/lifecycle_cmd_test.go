// Integration tests verifying that all 8 lifecycle commands are no longer stubs.
// After Plan 04, no lifecycle command should return "not implemented".
package tests

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCommandsNotStub verifies that every lifecycle command either:
//  1. Returns a real implementation error (Docker not running, no config, etc.)
//  2. Returns help text without any "not implemented" output
//
// Running without Docker or zone.toml will produce errors, but those errors
// must NOT be "not implemented" — that exact string is the stub sentinel.
func TestCommandsNotStub(t *testing.T) {
	binary := getZoneBinary(t)

	commands := []string{
		"launch",
		"join",
		"exec",
		"shell",
		"build",
		"stop",
		"restart",
		"destroy",
	}

	for _, cmdName := range commands {
		cmdName := cmdName // capture for parallel subtests
		t.Run(cmdName, func(t *testing.T) {
			// Verify --help works (command is registered and has proper metadata).
			helpOut, err := exec.Command(binary, cmdName, "--help").CombinedOutput()
			require.NoError(t, err, "help for %s failed: %s", cmdName, string(helpOut))

			// Run the actual command in a temp dir (will fail, but NOT with "not implemented").
			// Use a no-zone.toml temp dir so config load fails rather than Docker calls.
			tmpDir := t.TempDir()
			runCmd := exec.Command(binary, cmdName)
			runCmd.Dir = tmpDir
			// Provide a clean env pointing to a non-existent XDG dir to avoid
			// picking up a real global config.
			runCmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+tmpDir+"/no-xdg")
			result, _ := runCmd.CombinedOutput()
			output := string(result)

			assert.NotContains(t, output, "not implemented",
				"%s still returns 'not implemented': %s", cmdName, output)
		})
	}
}

// TestCleanImageFlag verifies that the --image flag is registered on the clean command.
func TestCleanImageFlag(t *testing.T) {
	binary := getZoneBinary(t)

	out, err := exec.Command(binary, "clean", "--help").CombinedOutput()
	require.NoError(t, err, "clean --help failed: %s", string(out))
	assert.Contains(t, string(out), "--image",
		"clean command missing --image flag")
}
