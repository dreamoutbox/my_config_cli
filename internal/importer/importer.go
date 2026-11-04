// Package importer handles reverse synchronization of configuration files
// from system target locations into the central config repository.
package importer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dreamoutbox/my_config_cli/internal/config"
	"github.com/dreamoutbox/my_config_cli/internal/pathutil"
)

// Action classifies the result of importing an entry.
type Action string

const (
	ActionCopied        Action = "copied"
	ActionSkipped       Action = "skipped"
	ActionAlreadyLinked Action = "already_linked"
	ActionAdopted       Action = "adopted"
)

// Tool identifies supported configuration sources.
type Tool string

const (
	ToolAntigravity Tool = "antigravity"
	ToolGemini      Tool = "gemini"
	ToolGit         Tool = "git"
	ToolBash        Tool = "bash"
	ToolClaude      Tool = "claude"
	ToolPi          Tool = "pi"
	ToolOpenCode    Tool = "opencode"
	ToolAll         Tool = "all"
)

// NormalizeTool maps tool aliases to canonical tool identifiers.
//
// Arguments:
//   - tool: user-provided tool identifier or alias.
//
// Returns:
//   - canonical Tool enum value.
//
// Example usage:
//
//	canonical := importer.NormalizeTool("bashrc") // returns ToolBash
func NormalizeTool(tool Tool) Tool {
	switch strings.ToLower(string(tool)) {
	case "antigravity", "gemini":
		return ToolAntigravity
	case "bash", "bashrc":
		return ToolBash
	case "git", "gitconfig":
		return ToolGit
	case "claude":
		return ToolClaude
	case "pi":
		return ToolPi
	case "opencode":
		return ToolOpenCode
	default:
		return Tool(strings.ToLower(string(tool)))
	}
}

// Result captures the outcome of an individual entry import.
type Result struct {
	Entry  config.Entry
	Action Action
	Reason string
	Error  error
}

// DetectTool categorizes a mapping entry by target application or configuration.
//
// Arguments:
//   - entry: config.Entry to inspect based on its Target path.
//
// Returns:
//   - Tool enum identifying the corresponding application or subsystem.
//
// Example usage:
//
//	tool := importer.DetectTool(entry)
func DetectTool(entry config.Entry) Tool {
	target := entry.Target
	if strings.HasPrefix(target, "~/.gemini/") {
		return ToolAntigravity
	}
	if strings.HasPrefix(target, "~/.claude/") {
		return ToolClaude
	}
	if strings.HasPrefix(target, "~/.pi/") {
		return ToolPi
	}
	if strings.HasPrefix(target, "~/.config/opencode/") {
		return ToolOpenCode
	}
	if target == "~/.bashrc" || strings.HasSuffix(target, "/.bashrc") {
		return ToolBash
	}
	if target == "~/.gitconfig" || strings.HasSuffix(target, "/.gitconfig") {
		return ToolGit
	}
	return Tool("unknown")
}

// FilterEntries selects entries matching the requested tool.
// ToolAll returns all entries unmodified.
//
// Arguments:
//   - entries: slice of config.Entry to filter.
//   - tool: Tool filter criterion.
//
// Returns:
//   - slice of config.Entry matching the specified tool.
//
// Example usage:
//
//	gitEntries := importer.FilterEntries(allEntries, importer.ToolGit)
func FilterEntries(entries []config.Entry, tool Tool) []config.Entry {
	normalized := NormalizeTool(tool)
	if normalized == ToolAll || normalized == "" {
		return entries
	}

	var filtered []config.Entry
	for _, entry := range entries {
		if DetectTool(entry) == normalized {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// ImportEntry reverse-syncs a single mapping entry from target to source.
func ImportEntry(entry config.Entry, dryRun bool) Result {
	res := Result{Entry: entry}

	if entry.Type == config.EntryTypeJSONMerge {
		res.Action = ActionSkipped
		res.Reason = "json_merge import not supported"
		return res
	}

	targetPath, err := pathutil.Expand(entry.Target)
	if err != nil {
		res.Error = fmt.Errorf("expanding target path %s: %w", entry.Target, err)
		return res
	}

	sourcePath, err := pathutil.Expand(entry.Source)
	if err != nil {
		res.Error = fmt.Errorf("expanding source path %s: %w", entry.Source, err)
		return res
	}

	targetLstat, err := os.Lstat(targetPath)
	if os.IsNotExist(err) {
		res.Action = ActionSkipped
		res.Reason = "target does not exist"
		return res
	} else if err != nil {
		res.Error = fmt.Errorf("inspecting target %s: %w", targetPath, err)
		return res
	}

	if targetLstat.Mode()&os.ModeSymlink != 0 {
		linkDest, err := os.Readlink(targetPath)
		if err != nil {
			res.Error = fmt.Errorf("reading symlink %s: %w", targetPath, err)
			return res
		}
		if linkDest == sourcePath {
			res.Action = ActionAlreadyLinked
			res.Reason = "target already points to store"
			return res
		}
	}

	if err := copyPath(targetPath, sourcePath, dryRun); err != nil {
		res.Error = fmt.Errorf("copying: %w", err)
		return res
	}

	res.Action = ActionCopied
	return res
}

// AdoptMissing inspects mapping entries and copies target configurations into the store
// if and only if the store copy is missing and the target exists on the host system.
// Existing store files are never overwritten, and existing targets are left as regular files
// (they are not replaced with symlinks).
//
// Arguments:
//   - entries: slice of config.Entry to inspect for adoption.
//
// Returns:
//   - slice of Result detailing the action taken for each entry.
//
// Example usage:
//
//	results := importer.AdoptMissing(mapping.Entries)
func AdoptMissing(entries []config.Entry) []Result {
	results := make([]Result, len(entries))
	for i, entry := range entries {
		results[i] = adoptEntry(entry)
	}
	return results
}

// adoptEntry inspects a single mapping entry and copies target into source
// when the source does not exist in the store.
func adoptEntry(entry config.Entry) Result {
	res := Result{Entry: entry}

	if entry.Type != config.EntryTypeSymlink {
		res.Action = ActionSkipped
		res.Reason = fmt.Sprintf("%s entry type not eligible for adoption", entry.Type)
		return res
	}

	targetPath, err := pathutil.Expand(entry.Target)
	if err != nil {
		res.Error = fmt.Errorf("expanding target path %s: %w", entry.Target, err)
		return res
	}

	sourcePath, err := pathutil.Expand(entry.Source)
	if err != nil {
		res.Error = fmt.Errorf("expanding source path %s: %w", entry.Source, err)
		return res
	}

	// Inspect target on host filesystem.
	targetLstat, err := os.Lstat(targetPath)
	if os.IsNotExist(err) {
		res.Action = ActionSkipped
		res.Reason = "target does not exist"
		return res
	} else if err != nil {
		res.Error = fmt.Errorf("inspecting target %s: %w", targetPath, err)
		return res
	}

	if targetLstat.Mode()&os.ModeSymlink != 0 {
		linkDest, err := os.Readlink(targetPath)
		if err != nil {
			res.Error = fmt.Errorf("reading symlink %s: %w", targetPath, err)
			return res
		}
		if linkDest == sourcePath {
			res.Action = ActionAlreadyLinked
			res.Reason = "target already points to store"
			return res
		}
	}

	// Never overwrite existing store files/directories during push.
	if _, err := os.Lstat(sourcePath); err == nil {
		res.Action = ActionSkipped
		res.Reason = "source already exists in store"
		return res
	} else if !os.IsNotExist(err) {
		res.Error = fmt.Errorf("inspecting source %s: %w", sourcePath, err)
		return res
	}

	// Verify target is accessible and not a broken symlink.
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		res.Action = ActionSkipped
		res.Reason = "target is broken symlink"
		return res
	} else if err != nil {
		res.Error = fmt.Errorf("resolving target %s: %w", targetPath, err)
		return res
	}

	if err := copyPath(targetPath, sourcePath, false); err != nil {
		res.Error = fmt.Errorf("adopting %s to %s: %w", targetPath, sourcePath, err)
		return res
	}

	res.Action = ActionAdopted
	return res
}

// ImportAll executes reverse import for all provided entries.
func ImportAll(entries []config.Entry, dryRun bool) []Result {
	results := make([]Result, len(entries))
	for i, entry := range entries {
		results[i] = ImportEntry(entry, dryRun)
	}
	return results
}

// copyPath copies a regular file or directory hierarchy from src to dst.
//
// Arguments:
//   - src: path to source file or directory.
//   - dst: path to destination file or directory.
//   - dryRun: if true, preview only without modifying filesystem.
//
// Returns:
//   - error if inspection or copying fails.
//
// Example usage:
//
//	err := copyPath(targetPath, sourcePath, false)
func copyPath(src, dst string, dryRun bool) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst, dryRun)
	}
	return copyFile(src, dst, dryRun)
}

// copyFile copies a regular file from src to dst, preserving permissions.
func copyFile(src, dst string, dryRun bool) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("source %s is not a regular file", src)
	}

	if dryRun {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	// Remove existing destination link or file to prevent writing into stale references.
	if _, err := os.Lstat(dst); err == nil {
		if err := os.Remove(dst); err != nil {
			return err
		}
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

// copyDir recursively copies a directory hierarchy from src to dst.
func copyDir(src, dst string, dryRun bool) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("source %s is not a directory", src)
	}

	if dryRun {
		return nil
	}

	if dstLstat, err := os.Lstat(dst); err == nil {
		if dstLstat.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(dst); err != nil {
				return err
			}
		}
	}

	if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, d := range entries {
		srcChild := filepath.Join(src, d.Name())
		dstChild := filepath.Join(dst, d.Name())

		if d.IsDir() {
			if err := copyDir(srcChild, dstChild, dryRun); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcChild, dstChild, dryRun); err != nil {
				return err
			}
		}
	}

	return nil
}
