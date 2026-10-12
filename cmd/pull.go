package cmd

import (
	"fmt"

	"github.com/dreamoutbox/my_config_cli/internal/config"
	"github.com/dreamoutbox/my_config_cli/internal/git"
	"github.com/spf13/cobra"
)

var pullFromCommitFlag string

var pullCmd = &cobra.Command{
	Use:     "pull",
	Aliases: []string{"load", "restore"},
	Short:   "Pull configuration store from git repository and apply symlinks",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPull(cmd)
	},
}

// runPull fetches latest configuration or checks out a specific commit from git remote, then applies all mappings.
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

	if pullFromCommitFlag != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Fetching configuration from %s...\n", repoURL)
		if err := git.Fetch(storeDir); err != nil {
			return fmt.Errorf("fetching from remote: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Checking out commit %s...\n", pullFromCommitFlag)
		if err := git.Checkout(storeDir, pullFromCommitFlag); err != nil {
			return fmt.Errorf("checking out commit %s: %w", pullFromCommitFlag, err)
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Pulling configuration from %s...\n", repoURL)
		if err := git.Pull(storeDir); err != nil {
			return fmt.Errorf("pulling from remote: %w", err)
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Applying configuration symlinks and merges...")
	return runLink(cmd)
}

func init() {
	pullCmd.Flags().StringVar(&pullFromCommitFlag, "from-commit", "", "Checkout specific commit hash or ref instead of latest")
	rootCmd.AddCommand(pullCmd)
}
