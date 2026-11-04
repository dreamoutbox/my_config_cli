package cmd

import (
	"github.com/spf13/cobra"
)

// repoFlag is the persistent --repo flag shared across push/pull commands.
var repoFlag string

// Version is the current application version.
var Version = "0.3.0"

var rootCmd = &cobra.Command{
	Use:     "myconfig",
	Short:   "Backup and restore configs to/from a git repository",
	Version: Version,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&repoFlag, "repo", "", "Git remote URL (overrides stored setting)")
}
