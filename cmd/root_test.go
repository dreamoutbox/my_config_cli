package cmd

import (
	"testing"
)

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Fatal("expected Version to be non-empty")
	}
	if rootCmd.Version != Version {
		t.Fatalf("expected rootCmd.Version %q, got %q", Version, rootCmd.Version)
	}
}
