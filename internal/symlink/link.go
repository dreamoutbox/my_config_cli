// Package symlink manages symlinks and JSON file merges for mapping entries.
package symlink

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dreamoutbox/my_config_cli/internal/config"
	"github.com/dreamoutbox/my_config_cli/internal/pathutil"
)

// Action indicates the outcome of applying a mapping entry.
type Action string

const (
	ActionLinked   Action = "linked"
	ActionReplaced Action = "replaced"
	ActionMerged   Action = "merged"
	// ActionCreated means a missing source placeholder was created in the store.
	ActionCreated Action = "created"
)

// HealthStatus indicates the current health of an entry on disk.
type HealthStatus string

const (
	HealthOK      HealthStatus = "OK"
	HealthMissing HealthStatus = "MISSING"
	HealthStale   HealthStatus = "STALE"
	HealthMerged  HealthStatus = "MERGED"
)

// Result captures the outcome of applying an individual entry.
type Result struct {
	Entry  config.Entry
	Action Action
	Error  error
}

// Apply applies a single mapping entry to the filesystem.
func Apply(entry config.Entry) Result {
	res := Result{Entry: entry}

	switch entry.Type {
	case config.EntryTypeSymlink:
		act, err := applySymlink(entry.Source, entry.Target, entry.SourceKind)
		res.Action = act
		res.Error = err
	case config.EntryTypeJSONMerge:
		key := entry.JSONKey
		if key == "" {
			key = "mcpServers"
		}
		err := applyJSONMerge(entry.Source, entry.Target, key)
		res.Action = ActionMerged
		res.Error = err
	default:
		res.Error = fmt.Errorf("unsupported entry type: %q", entry.Type)
	}

	return res
}

// ApplyAll applies all mapping entries in order.
func ApplyAll(mapping config.Mapping) []Result {
	results := make([]Result, len(mapping.Entries))
	for i, entry := range mapping.Entries {
		results[i] = Apply(entry)
	}
	return results
}

// CheckHealth inspects an entry on disk to verify its status.
func CheckHealth(entry config.Entry) (HealthStatus, error) {
	src, err := pathutil.Expand(entry.Source)
	if err != nil {
		return HealthMissing, err
	}
	dst, err := pathutil.Expand(entry.Target)
	if err != nil {
		return HealthMissing, err
	}

	if entry.Type == config.EntryTypeJSONMerge {
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			return HealthMissing, nil
		} else if err != nil {
			return HealthMissing, err
		}
		return HealthMerged, nil
	}

	dstInfo, err := os.Lstat(dst)
	if os.IsNotExist(err) {
		return HealthMissing, nil
	}
	if err != nil {
		return HealthMissing, err
	}

	if dstInfo.Mode()&os.ModeSymlink == 0 {
		return HealthStale, nil
	}

	target, err := os.Readlink(dst)
	if err != nil {
		return HealthStale, err
	}
	if target != src {
		return HealthStale, nil
	}

	if _, err := os.Stat(src); os.IsNotExist(err) {
		return HealthStale, nil
	}

	return HealthOK, nil
}

// applySymlink ensures the source exists (creating a placeholder when the tool
// or repository lacks it), then points the target at the source. kind selects
// the placeholder shape; existing sources are never modified. If kind is unspecified,
// it auto-detects dir if target or source is currently a directory on disk.
//
// Arguments:
//   - sourcePattern: path pattern for central config source.
//   - targetPattern: path pattern for target symlink.
//   - kind: SourceKind hint (file or dir).
//
// Returns:
//   - Action performed (ActionLinked, ActionReplaced, ActionCreated).
//   - error if operation fails.
//
// Example usage:
//   action, err := applySymlink("~/.my_config_cli/config/.bashrc", "~/.bashrc", config.SourceKindFile)
func applySymlink(sourcePattern, targetPattern string, kind config.SourceKind) (Action, error) {
	src, err := pathutil.Expand(sourcePattern)
	if err != nil {
		return "", err
	}
	dst, err := pathutil.Expand(targetPattern)
	if err != nil {
		return "", err
	}

	if kind == "" {
		if dstStat, err := os.Stat(dst); err == nil && dstStat.IsDir() {
			kind = config.SourceKindDir
		} else if srcStat, err := os.Stat(src); err == nil && srcStat.IsDir() {
			kind = config.SourceKindDir
		}
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}

	created, err := ensureSource(src, kind)
	if err != nil {
		return "", err
	}

	action := ActionLinked
	if created {
		action = ActionCreated
	}
	if _, err := os.Lstat(dst); err == nil {
		if !created {
			action = ActionReplaced
		}
		if err := os.RemoveAll(dst); err != nil {
			return "", err
		}
	}

	if err := os.Symlink(src, dst); err != nil {
		return "", err
	}

	return action, nil
}

// ensureSource creates a placeholder at src when it is missing.
// kind is the mapping's source_kind; an empty value (old mappings) defaults to
// a file, matching the previous behavior for missing sources. Returns true when
// a placeholder was created so the caller can report that a push is needed.
func ensureSource(src string, kind config.SourceKind) (bool, error) {
	if _, err := os.Stat(src); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}

	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		return false, err
	}

	switch kind {
	case config.SourceKindDir:
		return true, os.MkdirAll(src, 0o755)
	case config.SourceKindFile, "":
		content := []byte{}
		if strings.HasSuffix(strings.ToLower(src), ".json") {
			content = []byte("{}\n")
		}
		return true, os.WriteFile(src, content, 0o600)
	default:
		return false, fmt.Errorf("unsupported source_kind %q for %s", kind, src)
	}
}

// applyJSONMerge updates a specific key in target JSON with the contents from source.
func applyJSONMerge(sourcePattern, targetPattern, key string) error {
	src, err := pathutil.Expand(sourcePattern)
	if err != nil {
		return err
	}
	dst, err := pathutil.Expand(targetPattern)
	if err != nil {
		return err
	}

	srcData, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("source JSON file is missing: %s", src)
		}
		return fmt.Errorf("reading source JSON %s: %w", src, err)
	}
	if len(bytes.TrimSpace(srcData)) == 0 {
		return fmt.Errorf("source JSON file is empty: %s", src)
	}

	var parsedSrc any
	if err := json.Unmarshal(srcData, &parsedSrc); err != nil {
		return fmt.Errorf("parsing source JSON: %w", err)
	}

	replacement := parsedSrc
	if srcMap, ok := parsedSrc.(map[string]any); ok {
		if val, found := srcMap[key]; found {
			replacement = val
		}
	}

	// A symlink at the target would make writes follow it (possibly into a
	// missing path). Replace it with a real file; this also repairs links left
	// behind by older mappings.
	if info, lerr := os.Lstat(dst); lerr == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(dst); err != nil {
				return fmt.Errorf("removing symlink at %s: %w", dst, err)
			}
		}
	} else if !os.IsNotExist(lerr) {
		return fmt.Errorf("inspecting target %s: %w", dst, lerr)
	}

	targetMap := make(map[string]any)
	if dstData, err := os.ReadFile(dst); err == nil {
		if err := json.Unmarshal(dstData, &targetMap); err != nil {
			return fmt.Errorf("parsing target JSON: %w", err)
		}
	}

	targetMap[key] = replacement

	out, err := json.MarshalIndent(targetMap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling merged JSON: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	return writeFileAtomic(dst, append(out, '\n'), 0o600)
}

// writeFileAtomic writes data to a temp file in the target directory and
// renames it into place, so a failed write never leaves a partial file.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, path)
}
