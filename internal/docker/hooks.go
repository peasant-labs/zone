// hooks.go executes lifecycle hook commands (pre_build, post_stop) on the host.
package docker

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// runHooks executes a list of shell commands sequentially in repoDir.
// If failFast is true, execution stops and an error is returned on the first failure.
// If failFast is false, all commands run and failures emit a warning to stderr without
// returning an error (warn-only mode, used for post_stop hooks).
// Commands inherit the parent process environment (cmd.Env is nil).
func runHooks(cmds []string, repoDir string, failFast bool, stderr io.Writer) error {
	for _, cmd := range cmds {
		c := exec.Command("sh", "-c", cmd)
		c.Dir = repoDir
		c.Stdout = os.Stdout
		c.Stderr = stderr

		if err := c.Run(); err != nil {
			if failFast {
				return fmt.Errorf("hook %q failed: %w", cmd, err)
			}
			fmt.Fprintf(stderr, "Warning: hook %q failed: %v\n", cmd, err)
		}
	}
	return nil
}
