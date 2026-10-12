package config

import (
	"os"
	"testing"
)

func TestMappingFallbackToEmbedded(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// No mapping.json exists yet — should seed from embedded default.
	m, err := LoadMapping()
	if err != nil {
		t.Fatalf("LoadMapping: %v", err)
	}
	if len(m.Entries) == 0 {
		t.Fatal("expected at least one entry from embedded default")
	}

	// File should now exist on disk.
	if _, err := os.Stat(tmp + "/.my_config_cli/mapping.json"); err != nil {
		t.Fatalf("mapping.json not written: %v", err)
	}

	// Second load reads from disk (same result).
	m2, err := LoadMapping()
	if err != nil {
		t.Fatalf("second LoadMapping: %v", err)
	}
	if len(m2.Entries) != len(m.Entries) {
		t.Fatalf("entry count changed: %d vs %d", len(m2.Entries), len(m.Entries))
	}
}
