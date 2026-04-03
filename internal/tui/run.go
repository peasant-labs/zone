package tui

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"
)

// RunTUI wraps tea.NewProgram(model).Run() with deferred panic recovery
// that restores terminal state before re-panicking. BubbleTea v2 catches
// panics inside the program loop by default, but this outer wrapper handles
// panics that occur outside the tea.Program scope (e.g., in model
// constructors or post-Run processing).
//
// All cmd/ integration points MUST use RunTUI instead of calling
// tea.NewProgram(model).Run() directly.
func RunTUI(model tea.Model, opts ...tea.ProgramOption) (tea.Model, error) {
	// Save terminal state before entering TUI
	fd := int(os.Stdin.Fd())
	oldState, stateErr := term.GetState(fd)

	defer func() {
		if r := recover(); r != nil {
			// Restore terminal state before re-panicking
			if stateErr == nil && oldState != nil {
				_ = term.Restore(fd, oldState)
			}
			panic(fmt.Sprintf("tui panic (terminal restored): %v", r))
		}
	}()

	p := tea.NewProgram(model, opts...)
	return p.Run()
}
