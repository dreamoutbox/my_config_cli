package cmd

import (
	"fmt"
	"strings"

	"github.com/dreamoutbox/my_config_cli/internal/config"
	"github.com/dreamoutbox/my_config_cli/internal/importer"
	"github.com/spf13/cobra"
)

var (
	importToolFlag   string
	importDryRunFlag bool
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import existing tool configurations into central config store",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runImport(cmd)
	},
}

// runImport reads mapping.json in reverse and copies configuration files into the config store.
func runImport(cmd *cobra.Command) error {
	if strings.TrimSpace(importToolFlag) == "" {
		return fmt.Errorf("--tool is required (e.g. --tool git, --tool bashrc, --tool antigravity, --tool all)")
	}

	m, err := config.LoadMapping()
	if err != nil {
		return fmt.Errorf("loading mapping: %w", err)
	}

	selectedTool := importer.Tool(importToolFlag)
	entries := importer.FilterEntries(m.Entries, selectedTool)
	if len(entries) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No entries found for tool %q\n", importToolFlag)
		return nil
	}

	results := importer.ImportAll(entries, importDryRunFlag)
	var failed int
	var copied int

	for _, res := range results {
		if res.Error != nil {
			failed++
			fmt.Fprintf(cmd.ErrOrStderr(), "FAIL [%s]: %v\n", res.Entry.Target, res.Error)
			continue
		}

		actionLabel := strings.ToUpper(string(res.Action))
		if res.Reason != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "%-16s %s -> %s (%s)\n", actionLabel, res.Entry.Target, res.Entry.Source, res.Reason)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "%-16s %s -> %s\n", actionLabel, res.Entry.Target, res.Entry.Source)
		}

		if res.Action == importer.ActionCopied {
			copied++
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d mapping entries failed to import", failed)
	}

	if importDryRunFlag {
		fmt.Fprintln(cmd.OutOrStdout(), "\nDry run complete. No files were modified.")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nImport complete (%d imported).\n", copied)
	fmt.Fprintln(cmd.OutOrStdout(), "Review changes with 'myconfig status'.")
	fmt.Fprintln(cmd.OutOrStdout(), "When ready, manually commit and push with 'myconfig push'.")
	return nil
}

func init() {
	importCmd.Flags().StringVar(&importToolFlag, "tool", "", "Tool configuration to import (e.g. git, bashrc, antigravity, gemini, all)")
	_ = importCmd.MarkFlagRequired("tool")
	importCmd.Flags().BoolVar(&importDryRunFlag, "dry-run", false, "Preview import actions without writing to disk")
	rootCmd.AddCommand(importCmd)
}
