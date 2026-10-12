// Package git provides thin wrappers around the system git CLI.
package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitRun executes a git subcommand inside the given working directory.
func GitRun(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errOutput := strings.TrimSpace(stderr.String())
		if errOutput == "" {
			errOutput = strings.TrimSpace(stdout.String())
		}
		return stdout.String(), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, errOutput)
	}

	return stdout.String(), nil
}

// EnsureRepo initializes a git repository if one does not exist and configures the remote origin.
func EnsureRepo(dir, repoURL string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		if _, err := GitRun(dir, "init"); err != nil {
			return fmt.Errorf("initializing git repository: %w", err)
		}
	}

	if repoURL != "" {
		if _, err := GitRun(dir, "remote", "get-url", "origin"); err != nil {
			if _, err := GitRun(dir, "remote", "add", "origin", repoURL); err != nil {
				return fmt.Errorf("adding origin remote: %w", err)
			}
		} else {
			if _, err := GitRun(dir, "remote", "set-url", "origin", repoURL); err != nil {
				return fmt.Errorf("updating origin remote: %w", err)
			}
		}
	}

	return nil
}

// HasStagedChanges returns true if there are staged differences ready to commit.
func HasStagedChanges(dir string) (bool, error) {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = dir
	err := cmd.Run()
	if err == nil {
		return false, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return true, nil
	}
	return false, err
}

// CommitAll stages all changes and creates a commit if differences are present.
// Returns true if a commit was created, false if working tree was already clean.
func CommitAll(dir, message string) (bool, error) {
	if _, err := GitRun(dir, "add", "-A"); err != nil {
		return false, fmt.Errorf("staging changes: %w", err)
	}

	changed, err := HasStagedChanges(dir)
	if err != nil {
		return false, fmt.Errorf("inspecting staged changes: %w", err)
	}
	if !changed {
		return false, nil
	}

	if _, err := GitRun(dir, "commit", "-m", message); err != nil {
		return false, fmt.Errorf("creating commit: %w", err)
	}

	return true, nil
}

// Push pushes committed changes to origin HEAD.
func Push(dir string) error {
	if _, err := GitRun(dir, "push", "-u", "origin", "HEAD"); err != nil {
		return fmt.Errorf("pushing to remote: %w", err)
	}
	return nil
}

// Pull pulls latest changes using rebase from origin HEAD.
func Pull(dir string) error {
	if _, err := GitRun(dir, "pull", "--rebase", "origin", "HEAD"); err != nil {
		if _, err2 := GitRun(dir, "pull", "--rebase"); err2 != nil {
			return fmt.Errorf("pulling from remote: %w", err)
		}
	}
	return nil
}

// Status returns the short status of the working directory.
func Status(dir string) (string, error) {
	return GitRun(dir, "status", "--short")
}
