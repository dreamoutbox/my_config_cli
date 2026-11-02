package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dreamoutbox/my_config_cli/internal/config"
	"github.com/dreamoutbox/my_config_cli/internal/importer"
	"github.com/dreamoutbox/my_config_cli/internal/pathutil"
	"github.com/spf13/cobra"
)

var (
	mappingPathDefaultOnly bool
	mappingPathCustomOnly  bool

	mappingAddTarget string
	mappingAddSource string
	mappingAddKind   string

	mappingDeleteTarget string
)

var mappingCmd = &cobra.Command{
	Use:   "mapping",
	Short: "Manage and inspect configuration mappings",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMappingPath(cmd)
	},
}

var mappingPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print mapping file paths",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMappingPath(cmd)
	},
}

var mappingGetCmd = &cobra.Command{
	Use:     "get",
	Aliases: []string{"list", "ls"},
	Short:   "List all custom mappings",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMappingGet(cmd)
	},
}

var mappingDeleteCmd = &cobra.Command{
	Use:     "delete <target>",
	Aliases: []string{"remove", "rm"},
	Short:   "Delete a custom mapping",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMappingDelete(cmd, args)
	},
}

var mappingAddCmd = &cobra.Command{
	Use:   "add [target]",
	Short: "Add a custom file or directory mapping",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMappingAdd(cmd, args)
	},
}

// runMappingPath prints the filesystem paths for default and custom mapping files.
//
// Arguments:
//   - cmd: Cobra command context providing output writer and flags.
//
// Returns:
//   - error if resolving paths fails.
//
// Example usage:
//
//	err := runMappingPath(cmd)
func runMappingPath(cmd *cobra.Command) error {
	defPath, err := config.DefaultMappingPath()
	if err != nil {
		return fmt.Errorf("resolving default mapping path: %w", err)
	}
	customPath, err := config.CustomMappingPath()
	if err != nil {
		return fmt.Errorf("resolving custom mapping path: %w", err)
	}

	if mappingPathDefaultOnly {
		fmt.Fprintln(cmd.OutOrStdout(), defPath)
		return nil
	}
	if mappingPathCustomOnly {
		fmt.Fprintln(cmd.OutOrStdout(), customPath)
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Default mapping: %s\n", defPath)
	fmt.Fprintf(cmd.OutOrStdout(), "Custom mapping:  %s\n", customPath)
	return nil
}

// runMappingAdd registers a new custom mapping into custom_mapping.json and
// copies existing file/directory content to config store if not already present.
//
// Arguments:
//   - cmd: Cobra command context providing output writers and flags.
//   - args: command line arguments optionally containing positional target path.
//
// Returns:
//   - error if path resolution, source kind check, or saving fails.
//
// Example usage:
//
//	err := runMappingAdd(cmd, []string{"~/.vimrc"})
func runMappingAdd(cmd *cobra.Command, args []string) error {
	rawTarget := mappingAddTarget
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		rawTarget = args[0]
	}
	if strings.TrimSpace(rawTarget) == "" {
		return fmt.Errorf("target path is required (e.g. 'myconfig mapping add ~/.bashrc' or '--target ~/.bashrc')")
	}

	portableTarget, err := pathutil.Collapse(rawTarget)
	if err != nil {
		return fmt.Errorf("resolving target path %q: %w", rawTarget, err)
	}
	expandedTarget, err := pathutil.Expand(portableTarget)
	if err != nil {
		return fmt.Errorf("expanding target path %q: %w", portableTarget, err)
	}

	var sourcePattern string
	if strings.TrimSpace(mappingAddSource) != "" {
		collapsedSource, err := pathutil.Collapse(mappingAddSource)
		if err != nil {
			return fmt.Errorf("resolving source path %q: %w", mappingAddSource, err)
		}
		sourcePattern = collapsedSource
	} else {
		sourcePattern = deriveStoreSource(portableTarget)
	}

	expandedSource, err := pathutil.Expand(sourcePattern)
	if err != nil {
		return fmt.Errorf("expanding source path %q: %w", sourcePattern, err)
	}

	kind, err := resolveSourceKind(expandedTarget, mappingAddKind)
	if err != nil {
		return err
	}

	newEntry := config.Entry{
		Source:     sourcePattern,
		Target:     portableTarget,
		Type:       config.EntryTypeSymlink,
		SourceKind: kind,
	}

	// Copy existing content to central store if store source is currently absent.
	if _, err := os.Lstat(expandedSource); os.IsNotExist(err) {
		if _, err := os.Lstat(expandedTarget); err == nil {
			res := importer.ImportEntry(newEntry, false)
			if res.Error != nil {
				return fmt.Errorf("copying existing target to config store: %w", res.Error)
			}
			if res.Action == importer.ActionCopied {
				fmt.Fprintf(cmd.OutOrStdout(), "Copied existing target to config store: %s\n", sourcePattern)
			}
		}
	}

	custom, err := config.LoadCustomMapping()
	if err != nil {
		return fmt.Errorf("loading custom mapping: %w", err)
	}

	updated := upsertCustomEntry(&custom, newEntry)
	if err := config.SaveCustomMapping(custom); err != nil {
		return fmt.Errorf("saving custom mapping: %w", err)
	}

	if updated {
		fmt.Fprintf(cmd.OutOrStdout(), "Updated custom mapping: %s -> %s (kind: %s)\n", newEntry.Source, newEntry.Target, newEntry.SourceKind)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Added custom mapping: %s -> %s (kind: %s)\n", newEntry.Source, newEntry.Target, newEntry.SourceKind)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Run 'myconfig link' to apply symlinks, or 'myconfig push' to sync to git repository.")
	return nil
}

// deriveStoreSource calculates the canonical config store source path for a target.
// Home-relative targets (e.g. ~/.vimrc) map to ~/.my_config_cli/config/.vimrc.
//
// Arguments:
//   - target: portable target path pattern (e.g. "~/.bashrc").
//
// Returns:
//   - canonical source pattern in central store (e.g. "~/.my_config_cli/config/.bashrc").
//
// Example usage:
//
//	source := deriveStoreSource("~/.config/fish")
func deriveStoreSource(target string) string {
	clean := filepath.Clean(target)
	if strings.HasPrefix(clean, "~/") {
		rel := strings.TrimPrefix(clean, "~/")
		return "~/.my_config_cli/config/" + filepath.ToSlash(rel)
	}
	return "~/.my_config_cli/config/" + filepath.Base(clean)
}

// resolveSourceKind determines whether the mapping targets a file or directory.
//
// Arguments:
//   - target: expanded target path on disk.
//   - kindFlag: user-provided kind flag ("file", "dir", or "").
//
// Returns:
//   - config.SourceKind (config.SourceKindFile or config.SourceKindDir).
//   - error if user-provided flag is invalid.
//
// Example usage:
//
//	kind, err := resolveSourceKind("/home/user/.bashrc", "file")
func resolveSourceKind(target string, kindFlag string) (config.SourceKind, error) {
	switch strings.ToLower(strings.TrimSpace(kindFlag)) {
	case string(config.SourceKindDir):
		return config.SourceKindDir, nil
	case string(config.SourceKindFile):
		return config.SourceKindFile, nil
	case "":
		if info, err := os.Stat(target); err == nil && info.IsDir() {
			return config.SourceKindDir, nil
		}
		return config.SourceKindFile, nil
	default:
		return "", fmt.Errorf("invalid kind %q: must be %q or %q", kindFlag, config.SourceKindFile, config.SourceKindDir)
	}
}

// upsertCustomEntry updates an existing entry with the same target or appends a new one.
//
// Arguments:
//   - custom: pointer to Mapping to update.
//   - newEntry: Entry to append or update.
//
// Returns:
//   - true if an existing entry was replaced, false if a new entry was appended.
//
// Example usage:
//
//	updated := upsertCustomEntry(&custom, entry)
func upsertCustomEntry(custom *config.Mapping, newEntry config.Entry) bool {
	for i, entry := range custom.Entries {
		if entry.Target == newEntry.Target {
			custom.Entries[i] = newEntry
			return true
		}
	}
	custom.Entries = append(custom.Entries, newEntry)
	return false
}

// runMappingGet outputs all entries configured in custom_mapping.json.
//
// Arguments:
//   - cmd: Cobra command context providing output writer.
//
// Returns:
//   - error if loading custom mapping fails.
//
// Example usage:
//   err := runMappingGet(cmd)
func runMappingGet(cmd *cobra.Command) error {
	custom, err := config.LoadCustomMapping()
	if err != nil {
		return fmt.Errorf("loading custom mapping: %w", err)
	}

	if len(custom.Entries) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No custom mappings configured.")
		return nil
	}

	for _, entry := range custom.Entries {
		kind := entry.SourceKind
		if kind == "" {
			kind = config.SourceKindFile
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[%-4s] %s -> %s\n", kind, entry.Target, entry.Source)
	}
	return nil
}

// runMappingDelete removes a custom entry by target path from custom_mapping.json.
//
// Arguments:
//   - cmd: Cobra command context providing output writer and flags.
//   - args: command line arguments optionally containing positional target path.
//
// Returns:
//   - error if target is missing, not found in custom mapping, or saving fails.
//
// Example usage:
//   err := runMappingDelete(cmd, []string{"~/.vimrc"})
func runMappingDelete(cmd *cobra.Command, args []string) error {
	rawTarget := mappingDeleteTarget
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		rawTarget = args[0]
	}
	if strings.TrimSpace(rawTarget) == "" {
		return fmt.Errorf("target path is required (e.g. 'myconfig mapping delete ~/.bashrc' or '--target ~/.bashrc')")
	}

	portableTarget, err := pathutil.Collapse(rawTarget)
	if err != nil {
		return fmt.Errorf("resolving target path %q: %w", rawTarget, err)
	}
	expandedTarget, _ := pathutil.Expand(portableTarget)

	custom, err := config.LoadCustomMapping()
	if err != nil {
		return fmt.Errorf("loading custom mapping: %w", err)
	}

	foundIndex := -1
	for i, entry := range custom.Entries {
		entryExpanded, _ := pathutil.Expand(entry.Target)
		if entry.Target == portableTarget || (expandedTarget != "" && entryExpanded == expandedTarget) {
			foundIndex = i
			break
		}
	}

	if foundIndex == -1 {
		return fmt.Errorf("custom mapping for target %q not found", portableTarget)
	}

	removed := custom.Entries[foundIndex]
	custom.Entries = append(custom.Entries[:foundIndex], custom.Entries[foundIndex+1:]...)

	if err := config.SaveCustomMapping(custom); err != nil {
		return fmt.Errorf("saving custom mapping: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Deleted custom mapping: %s -> %s\n", removed.Target, removed.Source)
	fmt.Fprintln(cmd.OutOrStdout(), "Run 'myconfig push' to sync changes to remote repository.")
	return nil
}

func init() {
	mappingCmd.Flags().BoolVar(&mappingPathDefaultOnly, "default", false, "Print only the default mapping path")
	mappingCmd.Flags().BoolVar(&mappingPathCustomOnly, "custom", false, "Print only the custom mapping path")

	mappingPathCmd.Flags().BoolVar(&mappingPathDefaultOnly, "default", false, "Print only the default mapping path")
	mappingPathCmd.Flags().BoolVar(&mappingPathCustomOnly, "custom", false, "Print only the custom mapping path")

	mappingAddCmd.Flags().StringVar(&mappingAddTarget, "target", "", "Target destination path")
	mappingAddCmd.Flags().StringVar(&mappingAddSource, "source", "", "Source path inside the central store")
	mappingAddCmd.Flags().StringVar(&mappingAddKind, "kind", "", "Entry kind: file or dir (auto-detected if omitted)")

	mappingDeleteCmd.Flags().StringVar(&mappingDeleteTarget, "target", "", "Target destination path to remove")

	mappingCmd.AddCommand(mappingPathCmd)
	mappingCmd.AddCommand(mappingGetCmd)
	mappingCmd.AddCommand(mappingAddCmd)
	mappingCmd.AddCommand(mappingDeleteCmd)
	rootCmd.AddCommand(mappingCmd)
}
