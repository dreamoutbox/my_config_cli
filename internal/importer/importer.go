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
)

// Tool identifies supported configuration sources.
type Tool string

const (
	ToolAntigravity Tool = "antigravity"
	ToolAll         Tool = "all"
)

// Result captures the outcome of an individual entry import.
type Result struct {
	Entry  config.Entry
	Action Action
	Reason string
	Error  error
}

// DetectTool categorizes a mapping entry by target application.
func DetectTool(entry config.Entry) Tool {
	if strings.HasPrefix(entry.Target, "~/.gemini/") {
		return ToolAntigravity
	}
	return Tool("unknown")
}

// FilterEntries selects entries matching the requested tool.
// ToolAll returns all entries unmodified.
func FilterEntries(entries []config.Entry, tool Tool) []config.Entry {
	if tool == ToolAll || tool == "" {
		return entries
	}

	var filtered []config.Entry
	for _, entry := range entries {
		if DetectTool(entry) == tool {
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

	targetStat, err := os.Stat(targetPath)
	if err != nil {
		res.Error = fmt.Errorf("resolving target %s: %w", targetPath, err)
		return res
	}

	if targetStat.IsDir() {
		if err := copyDir(targetPath, sourcePath, dryRun); err != nil {
			res.Error = fmt.Errorf("copying directory: %w", err)
			return res
		}
	} else {
		if err := copyFile(targetPath, sourcePath, dryRun); err != nil {
			res.Error = fmt.Errorf("copying file: %w", err)
			return res
		}
	}

	res.Action = ActionCopied
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
