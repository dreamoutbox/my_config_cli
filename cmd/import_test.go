package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunImport(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Create dummy Gemini configuration
	geminiDir := filepath.Join(tmp, ".gemini")
	if err := os.MkdirAll(geminiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	geminiMD := filepath.Join(geminiDir, "GEMINI.md")
	if err := os.WriteFile(geminiMD, []byte("# My Gemini Config"), 0o600); err != nil {
		t.Fatal(err)
	}

	// 1. Test Dry Run
	importToolFlag = "antigravity"
	importDryRunFlag = true

	var outBuf bytes.Buffer
	importCmd.SetOut(&outBuf)
	importCmd.SetErr(&outBuf)

	if err := runImport(importCmd); err != nil {
		t.Fatalf("dry runImport: %v", err)
	}
	if !strings.Contains(outBuf.String(), "Dry run complete") {
		t.Fatalf("expected output to mention dry run completion, got: %s", outBuf.String())
	}

	destMD := filepath.Join(tmp, ".my_config_cli", "config", "AGENTS.md")
	if _, err := os.Stat(destMD); !os.IsNotExist(err) {
		t.Fatalf("destMD should not exist after dry run")
	}

	// 2. Test Real Import
	outBuf.Reset()
	importDryRunFlag = false

	if err := runImport(importCmd); err != nil {
		t.Fatalf("real runImport: %v", err)
	}

	destContent, err := os.ReadFile(destMD)
	if err != nil {
		t.Fatalf("reading imported file %s: %v", destMD, err)
	}
	if string(destContent) != "# My Gemini Config" {
		t.Fatalf("unexpected content in imported file: %s", string(destContent))
	}

	outStr := outBuf.String()
	if !strings.Contains(outStr, "myconfig push") {
		t.Fatalf("expected output to instruct user to run 'myconfig push', got: %s", outStr)
	}
	if !strings.Contains(outStr, "COPIED") {
		t.Fatalf("expected output to report COPIED, got: %s", outStr)
	}

	// 3. Re-running after linking should report already linked or copied
	outBuf.Reset()
	if err := runImport(importCmd); err != nil {
		t.Fatalf("subsequent runImport: %v", err)
	}
}

func TestRunImportUnknownTool(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	importToolFlag = "nonexistent"
	importDryRunFlag = false

	var outBuf bytes.Buffer
	importCmd.SetOut(&outBuf)
	importCmd.SetErr(&outBuf)

	if err := runImport(importCmd); err != nil {
		t.Fatalf("unexpected error for nonexistent tool: %v", err)
	}
	if !strings.Contains(outBuf.String(), "No entries found") {
		t.Fatalf("expected 'No entries found', got: %s", outBuf.String())
	}
}
