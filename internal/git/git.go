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
		if _, err := GitRun(dir, "init", "-b", "main"); err != nil {
			if _, err := GitRun(dir, "init"); err != nil {
				return fmt.Errorf("initializing git repository: %w", err)
			}
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

// PushOptions defines parameters for pushing commits to remote.
type PushOptions struct {
	Force bool
}

// PushWithOptions pushes committed changes to origin HEAD with optional flags.
func PushWithOptions(dir string, opts PushOptions) error {
	args := []string{"push"}
	if opts.Force {
		args = append(args, "--force")
	}
	args = append(args, "-u", "origin", "HEAD")
	if _, err := GitRun(dir, args...); err != nil {
		return fmt.Errorf("pushing to remote: %w", err)
	}
	return nil
}

// Push pushes committed changes to origin HEAD.
func Push(dir string) error {
	return PushWithOptions(dir, PushOptions{})
}

// RemoteHasHeads returns true if the remote origin has any branches or heads.
func RemoteHasHeads(dir string) (bool, error) {
	out, err := GitRun(dir, "ls-remote", "--heads", "origin")
	if err != nil {
		return false, fmt.Errorf("checking remote heads: %w", err)
	}
	return strings.TrimSpace(out) != "", nil
}

// IsRebaseInProgress returns true if a git rebase is in progress.
func IsRebaseInProgress(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	_, errApply := os.Stat(filepath.Join(gitDir, "rebase-apply"))
	_, errMerge := os.Stat(filepath.Join(gitDir, "rebase-merge"))
	return errApply == nil || errMerge == nil
}

// Pull pulls latest changes using rebase from origin HEAD.
// In case of a rebase failure or conflict, it aborts the rebase to keep the working tree clean.
func Pull(dir string) error {
	if _, err := GitRun(dir, "pull", "--rebase", "origin", "HEAD"); err != nil {
		if _, err2 := GitRun(dir, "pull", "--rebase"); err2 != nil {
			if IsRebaseInProgress(dir) {
				_, _ = GitRun(dir, "rebase", "--abort")
			}
			return fmt.Errorf("pulling from remote: %w", err)
		}
	}
	return nil
}

// Status returns the short status of the working directory.
func Status(dir string) (string, error) {
	return GitRun(dir, "status", "--short")
}
