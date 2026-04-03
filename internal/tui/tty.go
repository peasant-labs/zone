package tui

import (
	"os"

	"golang.org/x/term"
)

// IsTTY returns true if stdin is a terminal AND the --plain flag is not set.
// All command integration points call this to decide TUI vs plain text.
func IsTTY(plainFlag bool) bool {
	if plainFlag {
		return false
	}
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// IsOutputTTY returns true if stdout is a terminal.
// Used by the log viewer to detect piped output (zone logs -f | grep error).
func IsOutputTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}
