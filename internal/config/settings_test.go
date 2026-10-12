package config

import (
	"os"
	"testing"
)

func TestSettingsRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Should return zero value when no file exists.
	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings on missing file: %v", err)
	}
	if s.RepoURL != "" {
		t.Fatalf("expected empty RepoURL, got %q", s.RepoURL)
	}

	// Save and reload.
	want := Settings{RepoURL: "https://github.com/example/repo"}
	if err := SaveSettings(want); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	got, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings after save: %v", err)
	}
	if got.RepoURL != want.RepoURL {
		t.Fatalf("got %q, want %q", got.RepoURL, want.RepoURL)
	}

	// Verify the settings file exists.
	if _, err := os.Stat(tmp + "/.my_config_cli/settings.json"); err != nil {
		t.Fatalf("settings file missing: %v", err)
	}
}
