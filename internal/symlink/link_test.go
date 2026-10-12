package symlink

import (
	"encoding/json"
	"os"
	"path/filepath"
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
