package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dreamoutbox/my_config_cli/assets"
)

const mappingFile = "mapping.json"

// EntryType classifies how a mapping entry should be applied.
type EntryType string

const (
	// EntryTypeSymlink creates a filesystem symlink from source to target.
	EntryTypeSymlink EntryType = "symlink"
	// EntryTypeJSONMerge reads source as JSON and merges a single key into the target JSON file.
	EntryTypeJSONMerge EntryType = "json_merge"
)

// Entry describes a single source→target relationship.
type Entry struct {
	Source  string    `json:"source"`
	Target  string    `json:"target"`
	Type    EntryType `json:"type"`
	JSONKey string    `json:"json_key,omitempty"` // only for json_merge
}

// Mapping holds all configured source→target entries.
type Mapping struct {
	Entries []Entry `json:"entries"`
}

// LoadMapping reads ~/.my_config_cli/mapping.json.
// If the file does not exist, it writes and returns the embedded default.
// Symlink entries that collide with a json_merge target are dropped and the
// cleaned mapping is persisted once, migrating users seeded before the fix.
func LoadMapping() (Mapping, error) {
	dir, err := BaseDir()
	if err != nil {
		return Mapping{}, err
	}
	path := filepath.Join(dir, mappingFile)

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// First run: seed from embedded default.
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

	if dropConflictingSymlinks(&m) {
		if err := saveMapping(path, m); err != nil {
			return Mapping{}, fmt.Errorf("persisting migrated mapping: %w", err)
		}
	}

	return m, nil
}

// dropConflictingSymlinks removes symlink entries whose target is also managed
// by a json_merge entry, warning about each removal. Returns true if changed.
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
func saveMapping(path string, m Mapping) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
