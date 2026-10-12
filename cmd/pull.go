package cmd

import (
	"fmt"

	"github.com/dreamoutbox/my_config_cli/internal/config"
	"github.com/dreamoutbox/my_config_cli/internal/git"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:     "pull",
	Aliases: []string{"load", "restore"},
	Short:   "Pull configuration store from git repository and apply symlinks",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPull(cmd)
	},
}

// runPull fetches latest configuration from git remote, then applies all mappings.
func runPull(cmd *cobra.Command) error {
	repoURL, err := resolveAndPersistRepo(repoFlag)
	if err != nil {
		return err
	}

	storeDir, err := config.StoreDir()
	if err != nil {
		return fmt.Errorf("resolving store directory: %w", err)
	}

	if err := git.EnsureRepo(storeDir, repoURL); err != nil {
		return fmt.Errorf("initializing git repository: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Pulling configuration from %s...\n", repoURL)
	if err := git.Pull(storeDir); err != nil {
		return fmt.Errorf("pulling from remote: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Applying configuration symlinks and merges...")
	return runLink(cmd)
}

func init() {
	rootCmd.AddCommand(pullCmd)
}
