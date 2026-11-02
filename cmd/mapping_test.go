package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dreamoutbox/my_config_cli/internal/config"
)

func TestMappingPathCmd(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	var buf bytes.Buffer
	mappingCmd.SetOut(&buf)
	mappingCmd.SetErr(&buf)

	mappingPathDefaultOnly = false
	mappingPathCustomOnly = false

	if err := runMappingPath(mappingCmd); err != nil {
		t.Fatalf("runMappingPath: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Default mapping:") || !strings.Contains(out, "Custom mapping:") {
		t.Fatalf("expected both default and custom paths in output, got: %s", out)
	}

	buf.Reset()
	mappingPathDefaultOnly = true
	if err := runMappingPath(mappingCmd); err != nil {
		t.Fatalf("runMappingPath --default: %v", err)
	}
	wantDef := filepath.Join(tmp, ".my_config_cli", "mapping.json")
	if strings.TrimSpace(buf.String()) != wantDef {
		t.Fatalf("got %q, want %q", strings.TrimSpace(buf.String()), wantDef)
	}

	buf.Reset()
	mappingPathDefaultOnly = false
	mappingPathCustomOnly = true
	if err := runMappingPath(mappingCmd); err != nil {
		t.Fatalf("runMappingPath --custom: %v", err)
	}
	wantCust := filepath.Join(tmp, ".my_config_cli", "config", "custom_mapping.json")
	if strings.TrimSpace(buf.String()) != wantCust {
		t.Fatalf("got %q, want %q", strings.TrimSpace(buf.String()), wantCust)
	}

	mappingPathCustomOnly = false
}

func TestMappingAddCmdFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Create dummy target file
	vimrc := filepath.Join(tmp, ".vimrc")
	if err := os.WriteFile(vimrc, []byte("syntax on\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	mappingAddCmd.SetOut(&buf)
	mappingAddCmd.SetErr(&buf)
	mappingAddTarget = ""
	mappingAddSource = ""
	mappingAddKind = ""

	if err := runMappingAdd(mappingAddCmd, []string{"~/.vimrc"}); err != nil {
		t.Fatalf("runMappingAdd: %v", err)
	}

	// Verify content was copied to store
	storeVimrc := filepath.Join(tmp, ".my_config_cli", "config", ".vimrc")
	data, err := os.ReadFile(storeVimrc)
	if err != nil {
		t.Fatalf("store file missing: %v", err)
	}
	if string(data) != "syntax on\n" {
		t.Fatalf("got content %q, want 'syntax on\\n'", string(data))
	}

	// Verify custom mapping was saved
	custom, err := config.LoadCustomMapping()
	if err != nil {
		t.Fatalf("LoadCustomMapping: %v", err)
	}
	if len(custom.Entries) != 1 {
		t.Fatalf("expected 1 custom entry, got %d", len(custom.Entries))
	}
	entry := custom.Entries[0]
	if entry.Target != "~/.vimrc" || entry.Source != "~/.my_config_cli/config/.vimrc" || entry.SourceKind != config.SourceKindFile {
		t.Fatalf("unexpected entry: %+v", entry)
	}
}

func TestMappingAddCmdDirectory(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Create dummy target directory
	fishDir := filepath.Join(tmp, ".config", "fish")
	if err := os.MkdirAll(fishDir, 0o755); err != nil {
		t.Fatal(err)
	}
	confFile := filepath.Join(fishDir, "config.fish")
	if err := os.WriteFile(confFile, []byte("set -g fish_greeting\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	mappingAddCmd.SetOut(&buf)
	mappingAddCmd.SetErr(&buf)
	mappingAddTarget = ""
	mappingAddSource = ""
	mappingAddKind = ""

	if err := runMappingAdd(mappingAddCmd, []string{"~/.config/fish"}); err != nil {
		t.Fatalf("runMappingAdd: %v", err)
	}

	// Verify directory was copied to store
	storeFish := filepath.Join(tmp, ".my_config_cli", "config", ".config", "fish", "config.fish")
	data, err := os.ReadFile(storeFish)
	if err != nil {
		t.Fatalf("store fish file missing: %v", err)
	}
	if string(data) != "set -g fish_greeting\n" {
		t.Fatalf("got content %q", string(data))
	}

	// Verify custom mapping was saved with kind dir
	custom, err := config.LoadCustomMapping()
	if err != nil {
		t.Fatalf("LoadCustomMapping: %v", err)
	}
	if len(custom.Entries) != 1 {
		t.Fatalf("expected 1 custom entry, got %d", len(custom.Entries))
	}
	entry := custom.Entries[0]
	if entry.Target != "~/.config/fish" || entry.SourceKind != config.SourceKindDir {
		t.Fatalf("unexpected entry: %+v", entry)
	}
}

func TestMappingAddCmdFlagsAndMissingTarget(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	var buf bytes.Buffer
	mappingAddCmd.SetOut(&buf)
	mappingAddCmd.SetErr(&buf)

	// Missing target error
	mappingAddTarget = ""
	mappingAddSource = ""
	mappingAddKind = ""
	err := runMappingAdd(mappingAddCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "target path is required") {
		t.Fatalf("expected missing target error, got: %v", err)
	}

	// Explicit flags
	mappingAddTarget = "~/.tmux.conf"
	mappingAddSource = "~/.my_config_cli/config/custom_tmux.conf"
	mappingAddKind = "file"

	if err := runMappingAdd(mappingAddCmd, nil); err != nil {
		t.Fatalf("runMappingAdd with flags: %v", err)
	}

	custom, err := config.LoadCustomMapping()
	if err != nil {
		t.Fatalf("LoadCustomMapping: %v", err)
	}
	if len(custom.Entries) != 1 {
		t.Fatalf("expected 1 custom entry, got %d", len(custom.Entries))
	}
	entry := custom.Entries[0]
	if entry.Target != "~/.tmux.conf" || entry.Source != "~/.my_config_cli/config/custom_tmux.conf" || entry.SourceKind != config.SourceKindFile {
		t.Fatalf("unexpected entry: %+v", entry)
	}

	// Clean up flags
	mappingAddTarget = ""
	mappingAddSource = ""
	mappingAddKind = ""
}

func TestMappingGetCmd(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	var buf bytes.Buffer
	mappingGetCmd.SetOut(&buf)
	mappingGetCmd.SetErr(&buf)

	// 1. When empty
	if err := runMappingGet(mappingGetCmd); err != nil {
		t.Fatalf("runMappingGet on empty: %v", err)
	}
	if !strings.Contains(buf.String(), "No custom mappings configured") {
		t.Fatalf("expected empty message, got: %s", buf.String())
	}

	// 2. With entries
	sample := config.Mapping{
		Entries: []config.Entry{
			{Source: "~/.my_config_cli/config/.vimrc", Target: "~/.vimrc", Type: config.EntryTypeSymlink, SourceKind: config.SourceKindFile},
			{Source: "~/.my_config_cli/config/.config/fish", Target: "~/.config/fish", Type: config.EntryTypeSymlink, SourceKind: config.SourceKindDir},
		},
	}
	if err := config.SaveCustomMapping(sample); err != nil {
		t.Fatal(err)
	}

	buf.Reset()
	if err := runMappingGet(mappingGetCmd); err != nil {
		t.Fatalf("runMappingGet with entries: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "~/.vimrc") || !strings.Contains(out, "~/.config/fish") {
		t.Fatalf("expected entries in output, got: %s", out)
	}
	if !strings.Contains(out, "[file]") || !strings.Contains(out, "[dir ]") {
		t.Fatalf("expected kinds in output, got: %s", out)
	}
}

func TestMappingDeleteCmd(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	sample := config.Mapping{
		Entries: []config.Entry{
			{Source: "~/.my_config_cli/config/.vimrc", Target: "~/.vimrc", Type: config.EntryTypeSymlink, SourceKind: config.SourceKindFile},
			{Source: "~/.my_config_cli/config/.tmux.conf", Target: "~/.tmux.conf", Type: config.EntryTypeSymlink, SourceKind: config.SourceKindFile},
		},
	}
	if err := config.SaveCustomMapping(sample); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	mappingDeleteCmd.SetOut(&buf)
	mappingDeleteCmd.SetErr(&buf)
	mappingDeleteTarget = ""

	// 1. Missing target
	err := runMappingDelete(mappingDeleteCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "target path is required") {
		t.Fatalf("expected missing target error, got: %v", err)
	}

	// 2. Target not found
	err = runMappingDelete(mappingDeleteCmd, []string{"~/.nonexistent"})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got: %v", err)
	}

	// 3. Successful delete with positional argument
	buf.Reset()
	if err := runMappingDelete(mappingDeleteCmd, []string{"~/.vimrc"}); err != nil {
		t.Fatalf("runMappingDelete: %v", err)
	}
	if !strings.Contains(buf.String(), "Deleted custom mapping: ~/.vimrc") {
		t.Fatalf("unexpected delete output: %s", buf.String())
	}

	custom, err := config.LoadCustomMapping()
	if err != nil {
		t.Fatal(err)
	}
	if len(custom.Entries) != 1 || custom.Entries[0].Target != "~/.tmux.conf" {
		t.Fatalf("expected only ~/.tmux.conf to remain, got: %+v", custom.Entries)
	}

	// 4. Successful delete with --target flag
	buf.Reset()
	mappingDeleteTarget = "~/.tmux.conf"
	if err := runMappingDelete(mappingDeleteCmd, nil); err != nil {
		t.Fatalf("runMappingDelete with flag: %v", err)
	}
	mappingDeleteTarget = ""

	custom, err = config.LoadCustomMapping()
	if err != nil {
		t.Fatal(err)
	}
	if len(custom.Entries) != 0 {
		t.Fatalf("expected 0 entries after deleting all, got: %d", len(custom.Entries))
	}
}
