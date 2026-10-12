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

	// 1. Test Missing Tool flag
	importToolFlag = ""
	importDryRunFlag = false
	var outBuf bytes.Buffer
	importCmd.SetOut(&outBuf)
	importCmd.SetErr(&outBuf)

	err := runImport(importCmd)
	if err == nil || !strings.Contains(err.Error(), "--tool is required") {
		t.Fatalf("expected error requiring --tool, got: %v", err)
	}

	// 2. Test Dry Run with "antigravity"
	outBuf.Reset()
	importToolFlag = "antigravity"
	importDryRunFlag = true

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

	// 3. Test Real Import with alias "gemini"
	outBuf.Reset()
	importToolFlag = "gemini"
	importDryRunFlag = false

	if err := runImport(importCmd); err != nil {
		t.Fatalf("real runImport with gemini alias: %v", err)
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

	// 4. Test subsequent import with "antigravity"
	outBuf.Reset()
	importToolFlag = "antigravity"
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
