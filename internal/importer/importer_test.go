package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dreamoutbox/my_config_cli/internal/config"
)

func TestDetectToolAndFilter(t *testing.T) {
	entries := []config.Entry{
		{Target: "~/.gemini/GEMINI.md"},
		{Target: "~/.gemini/config/skills"},
		{Target: "~/.claude/CLAUDE.md"},
		{Target: "~/.pi/agent/SYSTEM.md"},
		{Target: "~/.bashrc"},
		{Target: "~/.gitconfig"},
		{Target: "~/.other/unknown.conf"},
	}

	if DetectTool(entries[0]) != ToolAntigravity {
		t.Fatalf("expected ToolAntigravity, got %v", DetectTool(entries[0]))
	}
	if DetectTool(entries[2]) != ToolClaude {
		t.Fatalf("expected ToolClaude, got %v", DetectTool(entries[2]))
	}
	if DetectTool(entries[3]) != ToolPi {
		t.Fatalf("expected ToolPi, got %v", DetectTool(entries[3]))
	}
	if DetectTool(entries[4]) != ToolBash {
		t.Fatalf("expected ToolBash, got %v", DetectTool(entries[4]))
	}
	if DetectTool(entries[5]) != ToolGit {
		t.Fatalf("expected ToolGit, got %v", DetectTool(entries[5]))
	}
	if DetectTool(entries[6]) != Tool("unknown") {
		t.Fatalf("expected unknown, got %v", DetectTool(entries[6]))
	}

	filteredGit := FilterEntries(entries, Tool("git"))
	if len(filteredGit) != 1 || filteredGit[0].Target != "~/.gitconfig" {
		t.Fatalf("expected 1 git entry, got: %+v", filteredGit)
	}

	filteredBashrc := FilterEntries(entries, Tool("bashrc"))
	if len(filteredBashrc) != 1 || filteredBashrc[0].Target != "~/.bashrc" {
		t.Fatalf("expected 1 bashrc entry, got: %+v", filteredBashrc)
	}

	filteredAntigravity := FilterEntries(entries, ToolAntigravity)
	if len(filteredAntigravity) != 2 {
		t.Fatalf("expected 2 antigravity entries, got %d", len(filteredAntigravity))
	}

	filteredGemini := FilterEntries(entries, ToolGemini)
	if len(filteredGemini) != 2 {
		t.Fatalf("expected 2 gemini entries, got %d", len(filteredGemini))
	}

	all := FilterEntries(entries, ToolAll)
	if len(all) != len(entries) {
		t.Fatalf("expected %d entries for ToolAll, got %d", len(entries), len(all))
	}
}

func TestImportEntryFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	targetFile := filepath.Join(tmp, "gemini_config.json")
	if err := os.WriteFile(targetFile, []byte(`{"key":"value"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	storeFile := filepath.Join(tmp, "store", "config.json")
	entry := config.Entry{
		Source: storeFile,
		Target: targetFile,
		Type:   config.EntryTypeSymlink,
	}

	// 1. Dry run: should not create file on disk
	resDry := ImportEntry(entry, true)
	if resDry.Error != nil {
		t.Fatalf("dry run error: %v", resDry.Error)
	}
	if resDry.Action != ActionCopied {
		t.Fatalf("expected ActionCopied, got %v", resDry.Action)
	}
	if _, err := os.Stat(storeFile); !os.IsNotExist(err) {
		t.Fatalf("storeFile should not exist after dry run")
	}

	// 2. Real import: should copy file
	res := ImportEntry(entry, false)
	if res.Error != nil {
		t.Fatalf("real import error: %v", res.Error)
	}
	if res.Action != ActionCopied {
		t.Fatalf("expected ActionCopied, got %v", res.Action)
	}

	data, err := os.ReadFile(storeFile)
	if err != nil {
		t.Fatalf("reading store file: %v", err)
	}
	if string(data) != `{"key":"value"}` {
		t.Fatalf("expected {\"key\":\"value\"}, got %s", string(data))
	}

	// 3. Import when target does not exist
	missingEntry := config.Entry{
		Source: filepath.Join(tmp, "store", "missing.json"),
		Target: filepath.Join(tmp, "nonexistent.json"),
		Type:   config.EntryTypeSymlink,
	}
	resMissing := ImportEntry(missingEntry, false)
	if resMissing.Action != ActionSkipped {
		t.Fatalf("expected ActionSkipped for missing target, got %v", resMissing.Action)
	}
}

func TestImportEntryDirectory(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	targetDir := filepath.Join(tmp, "skills")
	subDir := filepath.Join(targetDir, "sample")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sampleFile := filepath.Join(subDir, "SKILL.md")
	if err := os.WriteFile(sampleFile, []byte("# Skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	storeDir := filepath.Join(tmp, "store", "skills")
	entry := config.Entry{
		Source: storeDir,
		Target: targetDir,
		Type:   config.EntryTypeSymlink,
	}

	res := ImportEntry(entry, false)
	if res.Error != nil {
		t.Fatalf("directory import error: %v", res.Error)
	}
	if res.Action != ActionCopied {
		t.Fatalf("expected ActionCopied, got %v", res.Action)
	}

	copiedFile := filepath.Join(storeDir, "sample", "SKILL.md")
	content, err := os.ReadFile(copiedFile)
	if err != nil {
		t.Fatalf("reading copied file in directory: %v", err)
	}
	if string(content) != "# Skill" {
		t.Fatalf("expected '# Skill', got %q", string(content))
	}
}

func TestImportEntryAlreadyLinked(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	storeFile := filepath.Join(tmp, "store", "AGENTS.md")
	if err := os.MkdirAll(filepath.Dir(storeFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(storeFile, []byte("# Agents"), 0o600); err != nil {
		t.Fatal(err)
	}

	targetLink := filepath.Join(tmp, "GEMINI.md")
	if err := os.Symlink(storeFile, targetLink); err != nil {
		t.Fatal(err)
	}

	entry := config.Entry{
		Source: storeFile,
		Target: targetLink,
		Type:   config.EntryTypeSymlink,
	}

	res := ImportEntry(entry, false)
	if res.Error != nil {
		t.Fatalf("unexpected error: %v", res.Error)
	}
	if res.Action != ActionAlreadyLinked {
		t.Fatalf("expected ActionAlreadyLinked, got %v", res.Action)
	}
}

func TestImportEntryJSONMerge(t *testing.T) {
	entry := config.Entry{
		Source: "~/.my_config_cli/config/mcp_config.json",
		Target: "~/.config/opencode/opencode.json",
		Type:   config.EntryTypeJSONMerge,
	}

	res := ImportEntry(entry, false)
	if res.Action != ActionSkipped {
		t.Fatalf("expected ActionSkipped for json_merge, got %v", res.Action)
	}
}
