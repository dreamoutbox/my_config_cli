package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dreamoutbox/my_config_cli/assets"
)

const (
	mappingFile       = "mapping.json"
	customMappingFile = "custom_mapping.json"
)

// DefaultMappingPath returns the path to the user's default mapping file (~/.my_config_cli/mapping.json).
//
// Example usage:
//   path, err := config.DefaultMappingPath()
//   if err != nil { ... }
func DefaultMappingPath() (string, error) {
	dir, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, mappingFile), nil
}

// CustomMappingPath returns the path to the git-tracked custom mapping file (~/.my_config_cli/config/custom_mapping.json).
//
// Example usage:
//   path, err := config.CustomMappingPath()
//   if err != nil { ... }
func CustomMappingPath() (string, error) {
	dir, err := StoreDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, customMappingFile), nil
}

// LoadCustomMapping reads ~/.my_config_cli/config/custom_mapping.json.
// If the file does not exist, it returns an empty Mapping and nil error.
//
// Example usage:
//   custom, err := config.LoadCustomMapping()
//   if err != nil { ... }
func LoadCustomMapping() (Mapping, error) {
	path, err := CustomMappingPath()
	if err != nil {
		return Mapping{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Mapping{}, nil
	}
	if err != nil {
		return Mapping{}, err
	}
	var m Mapping
	if err := json.Unmarshal(data, &m); err != nil {
		return Mapping{}, fmt.Errorf("parsing custom mapping %s: %w", path, err)
	}
	return m, nil
}

// SaveCustomMapping writes the given Mapping to ~/.my_config_cli/config/custom_mapping.json.
//
// Arguments:
//   - m: Mapping object containing custom entries to serialize and save.
//
// Returns:
//   - error if directory creation, marshaling, or file writing fails.
//
// Example usage:
//   err := config.SaveCustomMapping(customMapping)
//   if err != nil { ... }
func SaveCustomMapping(m Mapping) error {
	path, err := CustomMappingPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return saveMapping(path, m)
}

// EntryType classifies how a mapping entry should be applied.
type EntryType string

const (
	// EntryTypeSymlink creates a filesystem symlink from source to target.
	EntryTypeSymlink EntryType = "symlink"
	// EntryTypeJSONMerge reads source as JSON and merges a single key into the target JSON file.
	EntryTypeJSONMerge EntryType = "json_merge"
)

// SourceKind describes the filesystem shape of a symlink source, used when the
// source is missing and a placeholder must be created.
type SourceKind string

const (
	SourceKindFile SourceKind = "file"
	SourceKindDir  SourceKind = "dir"
)

// Entry describes a single source→target relationship.
type Entry struct {
	Source     string     `json:"source"`
	Target     string     `json:"target"`
	Type       EntryType  `json:"type"`
	JSONKey    string     `json:"json_key,omitempty"`    // only for json_merge
	SourceKind SourceKind `json:"source_kind,omitempty"` // only for symlink
}

// Mapping holds all configured source→target entries.
type Mapping struct {
	Entries []Entry `json:"entries"`
}

// LoadMapping reads default mappings (~/.my_config_cli/mapping.json) and merges
// optional custom mappings from the git store (~/.my_config_cli/config/custom_mapping.json).
// If default mapping.json does not exist, it writes and returns the embedded default.
// It backfills any newly introduced embedded default entries into an existing mapping,
// merges entries from custom_mapping.json (overriding defaults by Target),
// drops symlink entries that collide with a json_merge target, and persists default changes if needed.
//
// Example usage:
//   mapping, err := config.LoadMapping()
//   if err != nil {
//       log.Fatalf("failed to load mapping: %v", err)
//   }
func LoadMapping() (Mapping, error) {
	path, err := DefaultMappingPath()
	if err != nil {
		return Mapping{}, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// First run: seed from embedded default.
		dir := filepath.Dir(path)
		if mkErr := os.MkdirAll(dir, 0o700); mkErr != nil {
			return Mapping{}, mkErr
		}
		if writeErr := os.WriteFile(path, assets.DefaultMappingJSON, 0o600); writeErr != nil {
			return Mapping{}, writeErr
		}
		data = assets.DefaultMappingJSON
	} else if err != nil {
		return Mapping{}, err
	}

	var m Mapping
	if err := json.Unmarshal(data, &m); err != nil {
		return Mapping{}, err
	}

	var changed bool
	backfilled, err := backfillDefaultEntries(&m)
	if err != nil {
		return Mapping{}, fmt.Errorf("backfilling default entries: %w", err)
	}
	if backfilled {
		changed = true
	}

	if dropConflictingSymlinks(&m) {
		changed = true
	}

	if changed {
		if err := saveMapping(path, m); err != nil {
			return Mapping{}, fmt.Errorf("persisting migrated mapping: %w", err)
		}
	}

	custom, err := LoadCustomMapping()
	if err != nil {
		return Mapping{}, fmt.Errorf("loading custom mapping: %w", err)
	}
	if len(custom.Entries) > 0 {
		m = mergeMappings(m, custom)
		dropConflictingSymlinks(&m)
	}

	return m, nil
}

// mergeMappings combines default mapping entries with custom mapping entries.
// Any entry in custom whose Target matches an entry in base overrides the base entry.
//
// Arguments:
//   - base: default Mapping struct.
//   - custom: custom Mapping struct to merge on top.
//
// Returns:
//   - merged Mapping containing unique target mappings.
//
// Example usage:
//   combined := mergeMappings(defaultMapping, customMapping)
func mergeMappings(base, custom Mapping) Mapping {
	customTargets := make(map[string]bool, len(custom.Entries))
	for _, entry := range custom.Entries {
		customTargets[entry.Target] = true
	}

	merged := make([]Entry, 0, len(base.Entries)+len(custom.Entries))
	for _, entry := range base.Entries {
		if !customTargets[entry.Target] {
			merged = append(merged, entry)
		}
	}
	merged = append(merged, custom.Entries...)
	return Mapping{Entries: merged}
}

// backfillDefaultEntries appends any entries from the embedded default mapping whose
// Target is not already present in m. This ensures existing users receive newly introduced
// default mappings (e.g. .bashrc, .gitconfig) without overwriting their existing configurations.
//
// Arguments:
//   - m: pointer to the Mapping to be inspected and updated in place.
//
// Returns:
//   - bool indicating whether any new entries were added.
//   - error if the embedded default mapping cannot be parsed.
//
// Example usage:
//   var m Mapping
//   added, err := backfillDefaultEntries(&m)
//   if err != nil { ... }
func backfillDefaultEntries(m *Mapping) (bool, error) {
	var def Mapping
	if err := json.Unmarshal(assets.DefaultMappingJSON, &def); err != nil {
		return false, fmt.Errorf("parsing embedded default mapping: %w", err)
	}

	existingTargets := make(map[string]bool, len(m.Entries))
	for _, entry := range m.Entries {
		existingTargets[entry.Target] = true
	}

	var changed bool
	for _, defEntry := range def.Entries {
		if !existingTargets[defEntry.Target] {
			m.Entries = append(m.Entries, defEntry)
			existingTargets[defEntry.Target] = true
			changed = true
		}
	}

	return changed, nil
}

// dropConflictingSymlinks removes symlink entries whose target is also managed
// by a json_merge entry, warning about each removal.
//
// Arguments:
//   - m: pointer to the Mapping to inspect and filter in place.
//
// Returns:
//   - true if any conflicting entries were removed, false otherwise.
//
// Example usage:
//   var m Mapping
//   if dropConflictingSymlinks(&m) {
//       saveMapping(path, m)
//   }
func dropConflictingSymlinks(m *Mapping) bool {
	mergeTargets := make(map[string]bool)
	for _, entry := range m.Entries {
		if entry.Type == EntryTypeJSONMerge {
			mergeTargets[entry.Target] = true
		}
	}

	kept := m.Entries[:0]
	changed := false
	for _, entry := range m.Entries {
		if entry.Type == EntryTypeSymlink && mergeTargets[entry.Target] {
			fmt.Fprintf(os.Stderr, "warning: ignoring symlink entry for %s (target is managed by json_merge)\n", entry.Target)
			changed = true
			continue
		}
		kept = append(kept, entry)
	}
	m.Entries = kept
	return changed
}

// saveMapping writes the mapping back to disk, preserving all user entries.
//
// Arguments:
//   - path: filesystem path where mapping JSON will be written.
//   - m: Mapping object to serialize and save.
//
// Returns:
//   - error if marshaling or file writing fails.
//
// Example usage:
//   err := saveMapping("/path/to/mapping.json", m)
func saveMapping(path string, m Mapping) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
