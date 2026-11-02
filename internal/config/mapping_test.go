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

	var hasJSONMerge, hasCustom bool
	for _, entry := range m.Entries {
		if entry.Type == EntryTypeSymlink && entry.Target == opencodeTarget {
			t.Fatal("conflicting symlink entry was not dropped")
		}
		if entry.Type == EntryTypeJSONMerge && entry.Target == opencodeTarget {
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
	if len(onDisk.Entries) != len(m.Entries) {
		t.Fatalf("expected migrated mapping on disk to have %d entries, got %d", len(m.Entries), len(onDisk.Entries))
	}

	again, err := LoadMapping()
	if err != nil {
		t.Fatalf("second LoadMapping: %v", err)
	}
	if len(again.Entries) != len(m.Entries) {
		t.Fatalf("expected idempotent migration, got %d entries (want %d)", len(again.Entries), len(m.Entries))
	}
}

func TestDefaultMappingContainsBashrcAndGitconfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	m, err := LoadMapping()
	if err != nil {
		t.Fatalf("LoadMapping: %v", err)
	}

	var hasBashrc, hasGitconfig bool
	for _, entry := range m.Entries {
		if entry.Target == "~/.bashrc" && entry.Type == EntryTypeSymlink && entry.SourceKind == SourceKindFile {
			hasBashrc = true
		}
		if entry.Target == "~/.gitconfig" && entry.Type == EntryTypeSymlink && entry.SourceKind == SourceKindFile {
			hasGitconfig = true
		}
	}
	if !hasBashrc {
		t.Fatal("default mapping missing ~/.bashrc")
	}
	if !hasGitconfig {
		t.Fatal("default mapping missing ~/.gitconfig")
	}
}

func TestLoadMappingBackfillsDefaults(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	dir := filepath.Join(tmp, ".my_config_cli")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	// Existing mapping with only a single custom entry
	customTarget := "~/.custom/file.txt"
	existing := Mapping{Entries: []Entry{
		{Source: "~/store/file.txt", Target: customTarget, Type: EntryTypeSymlink, SourceKind: SourceKindFile},
	}}
	data, err := json.MarshalIndent(existing, "", "  ")
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

	var hasCustom, hasBashrc, hasGitconfig bool
	for _, entry := range m.Entries {
		if entry.Target == customTarget {
			hasCustom = true
		}
		if entry.Target == "~/.bashrc" {
			hasBashrc = true
		}
		if entry.Target == "~/.gitconfig" {
			hasGitconfig = true
		}
	}

	if !hasCustom {
		t.Fatal("existing custom entry was lost during backfill")
	}
	if !hasBashrc {
		t.Fatal("~/.bashrc was not backfilled")
	}
	if !hasGitconfig {
		t.Fatal("~/.gitconfig was not backfilled")
	}
}

func TestMappingPathHelpers(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	defPath, err := DefaultMappingPath()
	if err != nil {
		t.Fatalf("DefaultMappingPath: %v", err)
	}
	wantDef := filepath.Join(tmp, ".my_config_cli", "mapping.json")
	if defPath != wantDef {
		t.Fatalf("DefaultMappingPath got %q, want %q", defPath, wantDef)
	}

	custPath, err := CustomMappingPath()
	if err != nil {
		t.Fatalf("CustomMappingPath: %v", err)
	}
	wantCust := filepath.Join(tmp, ".my_config_cli", "config", "custom_mapping.json")
	if custPath != wantCust {
		t.Fatalf("CustomMappingPath got %q, want %q", custPath, wantCust)
	}
}

func TestLoadCustomMappingMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	custom, err := LoadCustomMapping()
	if err != nil {
		t.Fatalf("LoadCustomMapping on missing file: %v", err)
	}
	if len(custom.Entries) != 0 {
		t.Fatalf("expected 0 entries for missing custom mapping, got %d", len(custom.Entries))
	}
}

func TestCustomMappingSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	sample := Mapping{
		Entries: []Entry{
			{Source: "~/.my_config_cli/config/.vimrc", Target: "~/.vimrc", Type: EntryTypeSymlink, SourceKind: SourceKindFile},
			{Source: "~/.my_config_cli/config/.config/nvim", Target: "~/.config/nvim", Type: EntryTypeSymlink, SourceKind: SourceKindDir},
		},
	}

	if err := SaveCustomMapping(sample); err != nil {
		t.Fatalf("SaveCustomMapping: %v", err)
	}

	loaded, err := LoadCustomMapping()
	if err != nil {
		t.Fatalf("LoadCustomMapping: %v", err)
	}
	if len(loaded.Entries) != 2 {
		t.Fatalf("expected 2 custom entries, got %d", len(loaded.Entries))
	}
	if loaded.Entries[0].Target != "~/.vimrc" || loaded.Entries[1].Target != "~/.config/nvim" {
		t.Fatalf("unexpected custom entries: %+v", loaded.Entries)
	}
}

func TestLoadMappingMergesCustomMappingAndOverrides(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	custom := Mapping{
		Entries: []Entry{
			// Override default .bashrc source
			{Source: "~/.my_config_cli/config/custom_bashrc", Target: "~/.bashrc", Type: EntryTypeSymlink, SourceKind: SourceKindFile},
			// Additional custom directory entry
			{Source: "~/.my_config_cli/config/.config/fish", Target: "~/.config/fish", Type: EntryTypeSymlink, SourceKind: SourceKindDir},
		},
	}
	if err := SaveCustomMapping(custom); err != nil {
		t.Fatalf("SaveCustomMapping: %v", err)
	}

	m, err := LoadMapping()
	if err != nil {
		t.Fatalf("LoadMapping: %v", err)
	}

	var foundBashrc Entry
	var foundFish bool
	for _, entry := range m.Entries {
		if entry.Target == "~/.bashrc" {
			foundBashrc = entry
		}
		if entry.Target == "~/.config/fish" {
			foundFish = true
		}
	}

	if foundBashrc.Source != "~/.my_config_cli/config/custom_bashrc" {
		t.Fatalf("expected custom .bashrc override %q, got %q", "~/.my_config_cli/config/custom_bashrc", foundBashrc.Source)
	}
	if !foundFish {
		t.Fatal("expected custom fish entry in merged mapping")
	}
}
