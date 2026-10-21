// Package git provides thin wrappers around the system git CLI.
package git

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitRun executes a git subcommand inside the given working directory.
func GitRun(dir string, args ...string) (string, error) {
	return runGit(dir, nil, false, args...)
}

// runGit is the shared exec path for non-interactive and interactive commands.
// extraEnv replaces the inherited environment when non-nil; interactive wires
// stdin and stderr to the terminal so git can prompt for credentials.
func runGit(dir string, extraEnv []string, interactive bool, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if interactive {
		cmd.Stdin = os.Stdin
		cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)
	}
	if extraEnv != nil {
		cmd.Env = extraEnv
	}

	if err := cmd.Run(); err != nil {
		errOutput := strings.TrimSpace(stderr.String())
		if errOutput == "" {
			errOutput = strings.TrimSpace(stdout.String())
		}
		// Git may echo the remote URL in its errors; never leak embedded tokens.
		return stdout.String(), fmt.Errorf("git %s: %w: %s", MaskURL(strings.Join(args, " ")), err, MaskURL(errOutput))
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
	if _, err := RunWithAuthRetry(dir, args...); err != nil {
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
	out, err := RunWithAuthRetry(dir, "ls-remote", "--heads", "origin")
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

// Fetch downloads objects and refs from origin remote.
func Fetch(dir string) error {
	if _, err := RunWithAuthRetry(dir, "fetch", "origin"); err != nil {
		return fmt.Errorf("fetching from remote: %w", err)
	}
	return nil
}

// Checkout switches HEAD and working tree to the given commit, branch, or tag ref.
func Checkout(dir, ref string) error {
	if _, err := GitRun(dir, "checkout", ref); err != nil {
		return fmt.Errorf("checking out %s: %w", ref, err)
	}
	return nil
}

// CurrentBranch returns the currently checked-out branch name, or empty string if HEAD is detached.
func CurrentBranch(dir string) (string, error) {
	out, err := GitRun(dir, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// CurrentCommit returns the commit hash of HEAD. If short is true, returns the abbreviated hash.
func CurrentCommit(dir string, short bool) (string, error) {
	args := []string{"rev-parse"}
	if short {
		args = append(args, "--short")
	}
	args = append(args, "HEAD")
	out, err := GitRun(dir, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// DefaultBranch determines the remote default branch name, falling back to "main" or "master".
func DefaultBranch(dir string) string {
	out, err := GitRun(dir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if err == nil {
		ref := strings.TrimSpace(out)
		if strings.HasPrefix(ref, "origin/") {
			return strings.TrimPrefix(ref, "origin/")
		}
	}
	if _, err := GitRun(dir, "remote", "set-head", "origin", "-a"); err == nil {
		out, err := GitRun(dir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
		if err == nil {
			ref := strings.TrimSpace(out)
			if strings.HasPrefix(ref, "origin/") {
				return strings.TrimPrefix(ref, "origin/")
			}
		}
	}
	for _, branch := range []string{"main", "master"} {
		if _, err := GitRun(dir, "rev-parse", "--verify", "origin/"+branch); err == nil {
			return branch
		}
		if _, err := GitRun(dir, "rev-parse", "--verify", branch); err == nil {
			return branch
		}
	}
	return "main"
}

// Pull pulls latest changes using rebase from origin HEAD.
// If the working tree is in detached HEAD, it switches back to the default branch first.
// In case of a rebase failure or conflict, it aborts the rebase to keep the working tree clean.
func Pull(dir string) error {
	branch, err := CurrentBranch(dir)
	if err == nil && branch == "" {
		// Reattach to default branch if in detached HEAD state
		defBranch := DefaultBranch(dir)
		_ = Checkout(dir, defBranch)
	}

	if _, err := RunWithAuthRetry(dir, "pull", "--rebase", "origin", "HEAD"); err != nil {
		if _, err2 := RunWithAuthRetry(dir, "pull", "--rebase"); err2 != nil {
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
