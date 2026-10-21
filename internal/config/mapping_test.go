package config

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestDefaultMappingHasNoSymlinkMergeConflict(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	m, err := LoadMapping()
	if err != nil {
		t.Fatalf("LoadMapping: %v", err)
	}

	mergeTargets := make(map[string]bool)
	for _, entry := range m.Entries {
		if entry.Type == EntryTypeJSONMerge {
			mergeTargets[entry.Target] = true
		}
	}
	for _, entry := range m.Entries {
		if entry.Type == EntryTypeSymlink && mergeTargets[entry.Target] {
			t.Fatalf("default mapping symlinks %s, which is also a json_merge target", entry.Target)
		}
	}
}

func TestLoadMappingDropsConflictingSymlink(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	dir := filepath.Join(tmp, ".my_config_cli")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	opencodeTarget := "~/.config/opencode/opencode.json"
	customTarget := "~/.custom/AGENTS.md"
	stale := Mapping{Entries: []Entry{
		{Source: "~/store/opencode.json", Target: opencodeTarget, Type: EntryTypeSymlink},
		{Source: "~/store/mcp_config.json", Target: opencodeTarget, Type: EntryTypeJSONMerge, JSONKey: "mcpServers"},
		{Source: "~/store/AGENTS.md", Target: customTarget, Type: EntryTypeSymlink},
	}}
	data, err := json.MarshalIndent(stale, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	mappingPath := filepath.Join(dir, "mapping.json")
	if err := os.WriteFile(mappingPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	m, err := LoadMapping()
	if err != nil {
		t.Fatalf("LoadMapping: %v", err)
	}
	if len(m.Entries) != 2 {
		t.Fatalf("expected 2 entries after migration, got %d", len(m.Entries))
	}

	var hasJSONMerge, hasCustom bool
	for _, entry := range m.Entries {
		if entry.Type == EntryTypeSymlink && entry.Target == opencodeTarget {
			t.Fatal("conflicting symlink entry was not dropped")
		}
		if entry.Type == EntryTypeJSONMerge {
			hasJSONMerge = true
		}
		if entry.Type == EntryTypeSymlink && entry.Target == customTarget {
			hasCustom = true
		}
	}
	if !hasJSONMerge {
		t.Fatal("json_merge entry was removed")
	}
	if !hasCustom {
		t.Fatal("unrelated user entry was removed")
	}

	// Migration must be persisted so subsequent loads are already clean.
	persisted, err := os.ReadFile(mappingPath)
	if err != nil {
		t.Fatal(err)
	}
	var onDisk Mapping
	if err := json.Unmarshal(persisted, &onDisk); err != nil {
		t.Fatal(err)
	}
	if len(onDisk.Entries) != 2 {
		t.Fatalf("expected migrated mapping on disk to have 2 entries, got %d", len(onDisk.Entries))
	}

	again, err := LoadMapping()
	if err != nil {
		t.Fatalf("second LoadMapping: %v", err)
	}
	if len(again.Entries) != 2 {
		t.Fatalf("expected idempotent migration, got %d entries", len(again.Entries))
	}
}
