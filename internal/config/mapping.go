package config

import (
	"encoding/json"
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
	return m, json.Unmarshal(data, &m)
}
