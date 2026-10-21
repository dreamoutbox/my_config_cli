package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupFakeGit puts a fake `git` first on PATH and returns the invocation log path.
func setupFakeGit(t *testing.T, mode string) string {
	t.Helper()

	script, err := os.ReadFile(filepath.Join("testdata", "fake_git.sh"))
	if err != nil {
		t.Fatalf("reading fake git: %v", err)
	}

	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "git"), script, 0o755); err != nil {
		t.Fatalf("writing fake git: %v", err)
	}

	logPath := filepath.Join(t.TempDir(), "calls.log")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_GIT_MODE", mode)
	t.Setenv("FAKE_GIT_LOG", logPath)
	t.Setenv("GITHUB_TOKEN", "scoped-token")
	return logPath
}

// setTerminal overrides terminal detection for the duration of a test.
func setTerminal(t *testing.T, interactive bool) {
	t.Helper()
	original := stdinIsTerminal
	stdinIsTerminal = func() bool { return interactive }
	t.Cleanup(func() { stdinIsTerminal = original })
}

func readCalls(t *testing.T, logPath string) []string {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading fake git log: %v", err)
	}
	parts := strings.Split(string(data), "call\n")
	calls := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			calls = append(calls, part)
		}
	}
	return calls
}

func TestRunWithAuthRetryRetriesWithoutAmbientToken(t *testing.T) {
	logPath := setupFakeGit(t, "auth")
	setTerminal(t, true)

	out, err := RunWithAuthRetry(t.TempDir(), "ls-remote", "--heads", "origin")
	if err != nil {
		t.Fatalf("RunWithAuthRetry: %v", err)
	}
	if !strings.Contains(out, "refs/heads/main") {
		t.Fatalf("expected refs output, got %q", out)
	}

	calls := readCalls(t, logPath)
	if len(calls) != 2 {
		t.Fatalf("expected 2 git invocations, got %d: %v", len(calls), calls)
	}
	if !strings.Contains(calls[0], "github_token=scoped-token") {
		t.Fatalf("first call should see the ambient token: %q", calls[0])
	}
	if strings.Contains(calls[1], "github_token=scoped-token") {
		t.Fatalf("retry must not inherit GITHUB_TOKEN: %q", calls[1])
	}
	if !strings.Contains(calls[1], "-c credential.helper=") {
		t.Fatalf("retry must reset credential helpers: %q", calls[1])
	}
	if !strings.Contains(calls[1], "terminal_prompt=1") {
		t.Fatalf("retry must enable terminal prompting: %q", calls[1])
	}
	if !strings.Contains(calls[1], "git_askpass=\n") {
		t.Fatalf("retry must clear GIT_ASKPASS: %q", calls[1])
	}
}

func TestRunWithAuthRetrySkipsWithoutTerminal(t *testing.T) {
	logPath := setupFakeGit(t, "auth")
	setTerminal(t, false)

	_, err := RunWithAuthRetry(t.TempDir(), "fetch", "origin")
	if err == nil {
		t.Fatal("expected error without a terminal")
	}
	if !strings.Contains(err.Error(), "no interactive terminal") {
		t.Fatalf("expected terminal hint, got %v", err)
	}

	if calls := readCalls(t, logPath); len(calls) != 1 {
		t.Fatalf("expected no retry without a terminal, got %d calls", len(calls))
	}
}

func TestRunWithAuthRetryDoesNotRetryOtherErrors(t *testing.T) {
	logPath := setupFakeGit(t, "generic")
	setTerminal(t, true)

	_, err := RunWithAuthRetry(t.TempDir(), "pull", "--rebase")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "some unrelated failure") {
		t.Fatalf("expected original error, got %v", err)
	}

	if calls := readCalls(t, logPath); len(calls) != 1 {
		t.Fatalf("expected exactly 1 call for non-auth error, got %d", len(calls))
	}
}

func TestRunWithAuthRetryReportsMissingRepository(t *testing.T) {
	logPath := setupFakeGit(t, "notfound")
	setTerminal(t, true)

	_, err := RunWithAuthRetry(t.TempDir(), "ls-remote", "origin")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "repository does not exist") {
		t.Fatalf("expected not-found message, got %v", err)
	}
	if !strings.Contains(err.Error(), "https://github.com/org/repo.git/") {
		t.Fatalf("expected masked URL in message, got %v", err)
	}

	if calls := readCalls(t, logPath); len(calls) != 1 {
		t.Fatalf("expected no retry for 404, got %d calls", len(calls))
	}
}
