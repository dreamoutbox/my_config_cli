package cmd

import (
	"path/filepath"
	"testing"

	"github.com/dreamoutbox/my_config_cli/internal/config"
)

func TestResolveAndPersistRepo(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// 1. Error when nothing provided
	_, err := resolveAndPersistRepo("")
	if err == nil {
		t.Fatal("expected error when no repo configured")
	}

	// 2. Pass flag -> persists to settings
	flagURL := "https://github.com/test/repo.git"
	resolved, err := resolveAndPersistRepo(flagURL)
	if err != nil {
		t.Fatalf("resolveAndPersistRepo: %v", err)
	}
	if resolved != flagURL {
		t.Fatalf("expected %q, got %q", flagURL, resolved)
	}

	// Verify settings file contains saved repo
	s, err := config.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if s.RepoURL != flagURL {
		t.Fatalf("expected saved URL %q, got %q", flagURL, s.RepoURL)
	}

	// 3. Fallback when flag is empty
	resolvedFallback, err := resolveAndPersistRepo("")
	if err != nil {
		t.Fatalf("fallback resolveAndPersistRepo: %v", err)
	}
	if resolvedFallback != flagURL {
		t.Fatalf("expected fallback %q, got %q", flagURL, resolvedFallback)
	}

	// 4. Override with new flag
	newURL := "https://github.com/test/repo2.git"
	resolvedNew, err := resolveAndPersistRepo(newURL)
	if err != nil {
		t.Fatalf("override resolveAndPersistRepo: %v", err)
	}
	if resolvedNew != newURL {
		t.Fatalf("expected %q, got %q", newURL, resolvedNew)
	}

	s2, err := config.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s2.RepoURL != newURL {
		t.Fatalf("expected updated saved URL %q, got %q", newURL, s2.RepoURL)
	}
	_ = filepath.Join(tmp, "noop")
}
