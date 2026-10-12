// Package config handles loading and persisting CLI settings and symlink mappings.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/dreamoutbox/my_config_cli/internal/pathutil"
)

const (
	settingsFile = "settings.json"
	defaultRoot  = "~/.my_config_cli"
	storeSubdir  = "config"
)

// BaseDir returns the root directory path for the CLI (~/.my_config_cli).
func BaseDir() (string, error) {
	return pathutil.Expand(defaultRoot)
}

// StoreDir returns the central config repository directory (~/.my_config_cli/config).
func StoreDir() (string, error) {
	base, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, storeSubdir), nil
}

// Settings holds persistent CLI configuration.
type Settings struct {
	RepoURL string `json:"repo_url"`
}

// LoadSettings reads settings from ~/.my_config_cli/settings.json.
// Returns zero-value Settings if the file does not exist.
func LoadSettings() (Settings, error) {
	dir, err := BaseDir()
	if err != nil {
		return Settings{}, err
	}
	path := filepath.Join(dir, settingsFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Settings{}, nil
	}
	if err != nil {
		return Settings{}, err
	}
	var s Settings
	return s, json.Unmarshal(data, &s)
}

// SaveSettings writes settings to ~/.my_config_cli/settings.json.
func SaveSettings(s Settings) error {
	dir, err := BaseDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, settingsFile), data, 0o600)
}
