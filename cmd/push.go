package cmd

import (
	"fmt"
	"time"

	"github.com/dreamoutbox/my_config_cli/internal/config"
	"github.com/dreamoutbox/my_config_cli/internal/git"
	"github.com/dreamoutbox/my_config_cli/internal/importer"
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

var (
	pushAutoRebaseFlag bool
	pushForceFlag      bool
)

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

	m, err := config.LoadMapping()
	if err != nil {
		return fmt.Errorf("loading mapping: %w", err)
	}

	adoptResults := importer.AdoptMissing(m.Entries)
	var adoptedCount int
	for _, res := range adoptResults {
		if res.Error != nil {
			return fmt.Errorf("adopting %s: %w", res.Entry.Target, res.Error)
		}
		if res.Action == importer.ActionAdopted {
			adoptedCount++
			fmt.Fprintf(cmd.OutOrStdout(), "Adopted %s -> %s\n", res.Entry.Target, res.Entry.Source)
		}
	}
	if adoptedCount > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Adopted %d configuration file(s) into config store\n", adoptedCount)
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

	if pushForceFlag {
		if err := git.PushWithOptions(storeDir, git.PushOptions{Force: true}); err != nil {
			return err
		}
	} else {
		if pushAutoRebaseFlag {
			hasRemoteHeads, err := git.RemoteHasHeads(storeDir)
			if err != nil {
				return fmt.Errorf("checking remote status: %w", err)
			}
			if hasRemoteHeads {
				fmt.Fprintln(cmd.OutOrStdout(), "Pulling remote changes (auto-rebase)...")
				if err := git.Pull(storeDir); err != nil {
					return fmt.Errorf("auto-rebase failed: %w (rebase aborted; resolve manually or use --force to overwrite)", err)
				}
			}
		}

		if err := git.Push(storeDir); err != nil {
			return err
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Successfully pushed configuration to %s\n", git.MaskURL(repoURL))
	return nil
}

func init() {
	pushCmd.Flags().BoolVar(&pushAutoRebaseFlag, "auto-rebase", true, "Automatically pull and rebase remote changes before pushing")
	pushCmd.Flags().BoolVarP(&pushForceFlag, "force", "f", false, "Force push to remote repository (overwrite remote)")
	rootCmd.AddCommand(pushCmd)
}
