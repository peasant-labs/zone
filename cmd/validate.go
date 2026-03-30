package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/peasant-labs/zone/internal/config"
	"github.com/spf13/cobra"
)

var allowDangerousMount bool

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Check zone.toml validity without starting anything",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Check for zone.toml.
		repoPath := "zone.toml"
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			cmd.PrintErrln("No zone.toml found. Run `zone init` to create one, or `zone config --global` to view global defaults.")
			return config.ErrNoConfig
		}

		var allErrors config.ValidationErrors

		// 2. Load global config, collect any unknown-key errors.
		global, globalErr := config.LoadGlobal()
		if globalErr != nil {
			var guke *config.UnknownKeysError
			if errors.As(globalErr, &guke) {
				allErrors = append(allErrors, config.ValidateUnknownKeys(guke.Keys, guke.File)...)
			} else {
				cmd.PrintErrln(fmt.Sprintf("Error: %v", globalErr))
				return globalErr
			}
		}

		// 3. Load repo config, collect any unknown-key errors.
		repo, repoErr := config.LoadRepo(repoPath)
		var ruke *config.UnknownKeysError
		if repoErr != nil {
			if errors.As(repoErr, &ruke) {
				allErrors = append(allErrors, config.ValidateUnknownKeys(ruke.Keys, ruke.File)...)
			} else {
				// Fatal parse error.
				cmd.PrintErrln(fmt.Sprintf("Error: %v", repoErr))
				return repoErr
			}
		}

		// 4. Merge and validate the combined config.
		merged, _ := config.Merge(global, repo)
		if merged != nil {
			validationErrs := config.Validate(merged)

			// Filter dangerous_mount errors if --allow-dangerous-mount is set.
			if allowDangerousMount {
				var filtered config.ValidationErrors
				for _, e := range validationErrs {
					if e.Category != "dangerous_mount" {
						filtered = append(filtered, e)
					}
				}
				validationErrs = filtered
			}

			allErrors = append(allErrors, validationErrs...)
		}

		// 5. Report results.
		if len(allErrors) == 0 {
			cmd.Println("zone.toml is valid.")
			return nil // exit 0
		}

		// Print all errors grouped by category.
		cmd.PrintErrln(allErrors.Error())

		if allErrors.HasErrors() {
			return fmt.Errorf("validation: %w", config.ErrNoConfig)
		}

		// Warnings only — exit 0 but print them.
		return nil
	},
}

func init() {
	validateCmd.Flags().BoolVar(&allowDangerousMount, "allow-dangerous-mount", false, "Allow mounts that would normally be blocked for security")
}
