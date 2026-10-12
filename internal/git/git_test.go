package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func configureGitUser(t *testing.T, dir string) {
	t.Helper()
	if _, err := GitRun(dir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("configure git email: %v", err)
	}
	if _, err := GitRun(dir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("configure git name: %v", err)
	}
}

func TestGitEngineLocal(t *testing.T) {
	tmp := t.TempDir()
	workDir := filepath.Join(tmp, "repo")

	// 1. EnsureRepo initializes repository
	if err := EnsureRepo(workDir, ""); err != nil {
		t.Fatalf("EnsureRepo: %v", err)
	}
	configureGitUser(t, workDir)

	// Check status when empty
	st, err := Status(workDir)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if strings.TrimSpace(st) != "" {
		t.Fatalf("expected empty status, got %q", st)
	}

	// 2. CommitAll when nothing is staged
	committed, err := CommitAll(workDir, "empty")
	if err != nil {
		t.Fatalf("CommitAll on clean tree: %v", err)
	}
	if committed {
		t.Fatal("expected no commit on clean tree")
	}

	// 3. Add file and commit
	testFile := filepath.Join(workDir, "hello.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0o600); err != nil {
		t.Fatal(err)
	}

	st, err = Status(workDir)
	if err != nil {
		t.Fatalf("Status with new file: %v", err)
	}
	if !strings.Contains(st, "hello.txt") {
		t.Fatalf("expected hello.txt in status, got %q", st)
	}

	committed, err = CommitAll(workDir, "initial commit")
	if err != nil {
		t.Fatalf("CommitAll: %v", err)
	}
	if !committed {
		t.Fatal("expected commit to be created")
	}

	// Tree should now be clean
	st, err = Status(workDir)
	if err != nil {
		t.Fatalf("Status after commit: %v", err)
	}
	if strings.TrimSpace(st) != "" {
		t.Fatalf("expected clean tree after commit, got %q", st)
	}
}

func TestGitPushAndPullWithLocalRemote(t *testing.T) {
	tmp := t.TempDir()
	bareRemote := filepath.Join(tmp, "remote.git")
	client1 := filepath.Join(tmp, "client1")
	client2 := filepath.Join(tmp, "client2")

	// Initialize bare remote repo
	if err := os.MkdirAll(bareRemote, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := GitRun(bareRemote, "init", "--bare", "-b", "main"); err != nil {
		// Fallback for older git that doesn't support -b
		if _, err := GitRun(bareRemote, "init", "--bare"); err != nil {
			t.Fatalf("init bare remote: %v", err)
		}
	}

	// Client 1 initializes and pushes to bare remote
	if err := EnsureRepo(client1, bareRemote); err != nil {
		t.Fatalf("EnsureRepo client1: %v", err)
	}
	configureGitUser(t, client1)
	if _, err := GitRun(client1, "branch", "-M", "main"); err != nil {
		t.Fatalf("rename branch to main: %v", err)
	}

	testFile := filepath.Join(client1, "sample.txt")
	if err := os.WriteFile(testFile, []byte("client1 content"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CommitAll(client1, "commit from client 1"); err != nil {
		t.Fatalf("CommitAll client1: %v", err)
	}
	if err := Push(client1); err != nil {
		t.Fatalf("Push client1: %v", err)
	}

	// Client 2 initializes against bare remote and pulls
	if err := EnsureRepo(client2, bareRemote); err != nil {
		t.Fatalf("EnsureRepo client2: %v", err)
	}
	configureGitUser(t, client2)
	if _, err := GitRun(client2, "branch", "-M", "main"); err != nil {
		t.Fatalf("rename branch to main client2: %v", err)
	}
	if err := Pull(client2); err != nil {
		t.Fatalf("Pull client2: %v", err)
	}

	pulledFile := filepath.Join(client2, "sample.txt")
	data, err := os.ReadFile(pulledFile)
	if err != nil {
		t.Fatalf("read pulled file: %v", err)
	}
	if string(data) != "client1 content" {
		t.Fatalf("expected 'client1 content', got %q", string(data))
	}
}
