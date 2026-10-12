package cmd

import (
	"fmt"
	"time"

	"github.com/dreamoutbox/my_config_cli/internal/config"
	"github.com/dreamoutbox/my_config_cli/internal/git"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:     "push",
	Aliases: []string{"save", "backup"},
	Short:   "Commit and push configuration store to git repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPush(cmd)
	},
}

// runPush commits all files in the config store directory and pushes them to the git remote.
func runPush(cmd *cobra.Command) error {
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

	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	committed, err := git.CommitAll(storeDir, fmt.Sprintf("myconfig backup: %s", timestamp))
	if err != nil {
		return fmt.Errorf("committing changes: %w", err)
	}

	if committed {
		fmt.Fprintln(cmd.OutOrStdout(), "Created commit for updated configuration files")
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "No local changes detected in config store")
	}

	if err := git.Push(storeDir); err != nil {
		return fmt.Errorf("pushing to remote: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Successfully pushed configuration to %s\n", repoURL)
	return nil
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
