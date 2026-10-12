// Package symlink manages symlinks and JSON file merges for mapping entries.
package symlink

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dreamoutbox/my_config_cli/internal/config"
	"github.com/dreamoutbox/my_config_cli/internal/pathutil"
)

// Action indicates the outcome of applying a mapping entry.
type Action string

const (
	ActionLinked   Action = "linked"
	ActionReplaced Action = "replaced"
	ActionMerged   Action = "merged"
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
		act, err := applySymlink(entry.Source, entry.Target)
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

// applySymlink removes any existing target and creates a symlink from source to target.
func applySymlink(sourcePattern, targetPattern string) (Action, error) {
	src, err := pathutil.Expand(sourcePattern)
	if err != nil {
		return "", err
	}
	dst, err := pathutil.Expand(targetPattern)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}

	action := ActionLinked
	if _, err := os.Lstat(dst); err == nil {
		action = ActionReplaced
		if err := os.RemoveAll(dst); err != nil {
			return "", err
		}
	}

	if err := os.Symlink(src, dst); err != nil {
		return "", err
	}

	return action, nil
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
		return fmt.Errorf("reading source JSON: %w", err)
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

	return os.WriteFile(dst, append(out, '\n'), 0o600)
}
