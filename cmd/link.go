package cmd

import (
	"fmt"

	"github.com/dreamoutbox/my_config_cli/internal/config"
	"github.com/dreamoutbox/my_config_cli/internal/symlink"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "Apply symlinks and config merges from mapping.json",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLink(cmd)
	},
}

// runLink loads mapping.json and applies all entries to the target system.
func runLink(cmd *cobra.Command) error {
	m, err := config.LoadMapping()
	if err != nil {
		return fmt.Errorf("loading mapping: %w", err)
	}

	results := symlink.ApplyAll(m)
	var failed, created int

	for _, res := range results {
		if res.Error != nil {
			failed++
			fmt.Fprintf(cmd.ErrOrStderr(), "FAIL [%s]: %v\n", res.Entry.Target, res.Error)
			continue
		}
		if res.Action == symlink.ActionCreated {
			created++
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%-8s %s -> %s\n", res.Action, res.Entry.Source, res.Entry.Target)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nSummary: %d applied, %d failed\n", len(results)-failed, failed)
	if created > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%d source placeholder(s) created in the store; run 'myconfig push' to save them.\n", created)
	}

	if failed > 0 {
		return fmt.Errorf("%d mapping entries failed to apply", failed)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(linkCmd)
}
