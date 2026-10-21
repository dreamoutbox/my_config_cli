package symlink

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dreamoutbox/my_config_cli/internal/config"
)

func TestApplySymlink(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	srcFile := filepath.Join(tmp, "src_file.txt")
	if err := os.WriteFile(srcFile, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}

	targetFile := filepath.Join(tmp, "target_link.txt")
	entry := config.Entry{
		Source: srcFile,
		Target: targetFile,
		Type:   config.EntryTypeSymlink,
	}

	// 1. Initial link
	res := Apply(entry)
	if res.Error != nil {
		t.Fatalf("Apply initial: %v", res.Error)
	}
	if res.Action != ActionLinked {
		t.Fatalf("expected ActionLinked, got %v", res.Action)
	}

	health, err := CheckHealth(entry)
	if err != nil {
		t.Fatalf("CheckHealth: %v", err)
	}
	if health != HealthOK {
		t.Fatalf("expected HealthOK, got %v", health)
	}

	// 2. Overwrite existing symlink
	res = Apply(entry)
	if res.Error != nil {
		t.Fatalf("Apply overwrite: %v", res.Error)
	}
	if res.Action != ActionReplaced {
		t.Fatalf("expected ActionReplaced, got %v", res.Action)
	}

	// 3. Health when source is deleted
	if err := os.Remove(srcFile); err != nil {
		t.Fatal(err)
	}
	health, err = CheckHealth(entry)
	if err != nil {
		t.Fatalf("CheckHealth after delete: %v", err)
	}
	if health != HealthStale {
		t.Fatalf("expected HealthStale, got %v", health)
	}
}

func TestApplyJSONMerge(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	srcFile := filepath.Join(tmp, "mcp_config.json")
	srcContent := map[string]any{
		"mcpServers": map[string]any{
			"test-server": map[string]any{"command": "node"},
		},
	}
	srcBytes, _ := json.Marshal(srcContent)
	if err := os.WriteFile(srcFile, srcBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	targetFile := filepath.Join(tmp, "opencode.json")
	targetInitial := map[string]any{
		"theme": "dark",
	}
	targetBytes, _ := json.Marshal(targetInitial)
	if err := os.WriteFile(targetFile, targetBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	entry := config.Entry{
		Source:  srcFile,
		Target:  targetFile,
		Type:    config.EntryTypeJSONMerge,
		JSONKey: "mcpServers",
	}

	res := Apply(entry)
	if res.Error != nil {
		t.Fatalf("Apply JSONMerge: %v", res.Error)
	}
	if res.Action != ActionMerged {
		t.Fatalf("expected ActionMerged, got %v", res.Action)
	}

	// Verify target file has both "theme" and "mcpServers"
	mergedBytes, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	var mergedMap map[string]any
	if err := json.Unmarshal(mergedBytes, &mergedMap); err != nil {
		t.Fatal(err)
	}

	if mergedMap["theme"] != "dark" {
		t.Fatalf("theme key overwritten, got %v", mergedMap["theme"])
	}
	if _, ok := mergedMap["mcpServers"]; !ok {
		t.Fatal("mcpServers key missing from target")
	}

	health, err := CheckHealth(entry)
	if err != nil {
		t.Fatalf("CheckHealth JSONMerge: %v", err)
	}
	if health != HealthMerged {
		t.Fatalf("expected HealthMerged, got %v", health)
	}
}

func TestApplyJSONMergeReplacesDanglingSymlink(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	srcFile := filepath.Join(tmp, "mcp_config.json")
	if err := os.WriteFile(srcFile, []byte(`{"mcpServers":{"srv":{"command":"node"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	targetFile := filepath.Join(tmp, ".config", "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(targetFile), 0o755); err != nil {
		t.Fatal(err)
	}
	// Older mappings symlinked the target to a source that is absent from the repo.
	danglingSource := filepath.Join(tmp, "store", ".config", "opencode", "opencode.json")
	if err := os.Symlink(danglingSource, targetFile); err != nil {
		t.Fatal(err)
	}

	entry := config.Entry{
		Source:  srcFile,
		Target:  targetFile,
		Type:    config.EntryTypeJSONMerge,
		JSONKey: "mcpServers",
	}

	res := Apply(entry)
	if res.Error != nil {
		t.Fatalf("Apply JSONMerge over dangling symlink: %v", res.Error)
	}

	info, err := os.Lstat(targetFile)
	if err != nil {
		t.Fatalf("target missing after merge: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("target should be a real file after merge")
	}
	if _, err := os.Lstat(danglingSource); !os.IsNotExist(err) {
		t.Fatalf("merge wrote through the symlink to %s", danglingSource)
	}

	data, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	var merged map[string]any
	if err := json.Unmarshal(data, &merged); err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 {
		t.Fatalf("expected only mcpServers in new target, got %v", merged)
	}
	if _, ok := merged["mcpServers"]; !ok {
		t.Fatalf("mcpServers missing from merged target: %v", merged)
	}
}

func TestApplyJSONMergeSourceErrors(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	targetFile := filepath.Join(tmp, "opencode.json")

	missing := config.Entry{
		Source:  filepath.Join(tmp, "missing.json"),
		Target:  targetFile,
		Type:    config.EntryTypeJSONMerge,
		JSONKey: "mcpServers",
	}
	if res := Apply(missing); res.Error == nil || !strings.Contains(res.Error.Error(), "source JSON file is missing") {
		t.Fatalf("expected missing-source error, got %v", res.Error)
	}

	emptySrc := filepath.Join(tmp, "empty.json")
	if err := os.WriteFile(emptySrc, []byte("  \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	empty := missing
	empty.Source = emptySrc
	if res := Apply(empty); res.Error == nil || !strings.Contains(res.Error.Error(), "source JSON file is empty") {
		t.Fatalf("expected empty-source error, got %v", res.Error)
	}
}
