package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dreamoutbox/my_config_cli/internal/config"
	"github.com/dreamoutbox/my_config_cli/internal/git"
	"github.com/dreamoutbox/my_config_cli/internal/symlink"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show git repository state and symlink health",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus(cmd)
	},
}

// runStatus displays current git status of config store and health of configured symlinks.
func runStatus(cmd *cobra.Command) error {
	storeDir, err := config.StoreDir()
	if err != nil {
		return fmt.Errorf("resolving store directory: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "=== Git Status ===")
	gitDir := filepath.Join(storeDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		fmt.Fprintf(cmd.OutOrStdout(), "Store repository not initialized at %s\n", storeDir)
	} else {
		output, err := git.Status(storeDir)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error checking git status: %v\n", err)
		} else {
			trimmed := strings.TrimSpace(output)
			if trimmed == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "Working tree clean (no uncommitted changes)")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), trimmed)
			}
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), "\n=== Mapping Health ===")
	m, err := config.LoadMapping()
	if err != nil {
		return fmt.Errorf("loading mapping: %w", err)
	}

	for _, entry := range m.Entries {
		health, checkErr := symlink.CheckHealth(entry)
		if checkErr != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "  [%-7s] %s -> %s (error: %v)\n", "ERROR", entry.Source, entry.Target, checkErr)
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  [%-7s] %s -> %s\n", health, entry.Source, entry.Target)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
