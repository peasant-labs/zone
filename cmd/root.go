package cmd

import (
	"github.com/spf13/cobra"
)

// version is the zone binary version string, set from main via SetVersion.
// Defaults to "dev" when not injected via ldflags.
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "zone",
	Short: "Sandboxed Docker workspaces for LLM coding agents",
	Long:  "Zone generates and manages isolated Docker workspaces for LLM coding agents. Run zone launch in any repo to get a sandboxed container with zero manual Docker configuration.",
}

// SetVersion sets the version string on the root command, injected from main via ldflags.
func SetVersion(v, commit, date string) {
	version = v
	rootCmd.Version = v + " (" + commit + ") built " + date
}

// Execute runs the root command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(
		initCmd,
		launchCmd,
		joinCmd,
		execCmd,
		shellCmd,
		buildCmd,
		stopCmd,
		restartCmd,
		lsCmd,
		logsCmd,
		cleanCmd,
		destroyCmd,
		statusCmd,
		configCmd,
		validateCmd,
	)

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Increase output verbosity")
	rootCmd.PersistentFlags().Bool("debug", false, "Maximum verbosity including raw API responses")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress all non-essential output")
	rootCmd.PersistentFlags().Bool("plain", false, "Disable TUI and use plain text output")

	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
}
