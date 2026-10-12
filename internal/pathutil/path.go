// Package pathutil provides filesystem path helpers.
package pathutil

import (
	"os"
	"path/filepath"
	"strings"
)

// Expand expands a leading tilde (~) to the current user's home directory.
func Expand(p string) (string, error) {
	if p == "~" {
		return os.UserHomeDir()
	}
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, p[2:]), nil
	}
	return p, nil
}
