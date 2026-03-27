package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var launchCmd = &cobra.Command{
	Use:     "launch",
	Aliases: []string{"up"},
	Short:   "Build (if needed) and attach to the container",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("not implemented")
	},
}
