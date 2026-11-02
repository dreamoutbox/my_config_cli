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

// Collapse converts an absolute or relative path under the user's home directory
// to a portable tilde path (~/...). Paths outside home are returned clean and unchanged.
//
// Arguments:
//   - p: filesystem path to convert.
//
// Returns:
//   - portable string path starting with ~/ if within home, or original clean path.
//   - error if expanding or resolving home or absolute path fails.
//
// Example usage:
//   portable, err := pathutil.Collapse("/home/user/.bashrc")
//   // portable is "~/.bashrc"
func Collapse(p string) (string, error) {
	if strings.HasPrefix(p, "~/") || p == "~" {
		return filepath.Clean(p), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return p, err
	}

	abs := p
	if !filepath.IsAbs(abs) {
		cwd, err := os.Getwd()
		if err == nil {
			abs = filepath.Join(cwd, p)
		}
	}
	abs = filepath.Clean(abs)
	cleanHome := filepath.Clean(home)

	if abs == cleanHome {
		return "~", nil
	}
	if strings.HasPrefix(abs, cleanHome+string(filepath.Separator)) {
		rel, err := filepath.Rel(cleanHome, abs)
		if err != nil {
			return p, err
		}
		return "~/" + filepath.ToSlash(rel), nil
	}

	return filepath.Clean(p), nil
}
