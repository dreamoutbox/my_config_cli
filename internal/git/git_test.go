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

func TestRemoteHasHeads(t *testing.T) {
	tmp := t.TempDir()
	bareRemote := filepath.Join(tmp, "remote.git")
	local := filepath.Join(tmp, "local")

	if err := os.MkdirAll(bareRemote, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := GitRun(bareRemote, "init", "--bare"); err != nil {
		t.Fatalf("init bare remote: %v", err)
	}

	if err := EnsureRepo(local, bareRemote); err != nil {
		t.Fatalf("EnsureRepo: %v", err)
	}
	configureGitUser(t, local)

	hasHeads, err := RemoteHasHeads(local)
	if err != nil {
		t.Fatalf("RemoteHasHeads on empty remote: %v", err)
	}
	if hasHeads {
		t.Fatal("expected empty remote to have no heads")
	}

	// Commit and push a file
	if err := os.WriteFile(filepath.Join(local, "file.txt"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CommitAll(local, "first commit"); err != nil {
		t.Fatalf("CommitAll: %v", err)
	}
	if err := Push(local); err != nil {
		t.Fatalf("Push: %v", err)
	}

	hasHeads, err = RemoteHasHeads(local)
	if err != nil {
		t.Fatalf("RemoteHasHeads after push: %v", err)
	}
	if !hasHeads {
		t.Fatal("expected remote to have heads after push")
	}
}

func TestPushForce(t *testing.T) {
	tmp := t.TempDir()
	bareRemote := filepath.Join(tmp, "remote.git")
	client1 := filepath.Join(tmp, "client1")
	client2 := filepath.Join(tmp, "client2")

	if err := os.MkdirAll(bareRemote, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := GitRun(bareRemote, "init", "--bare"); err != nil {
		t.Fatal(err)
	}

	// Client 1 pushes commit A
	if err := EnsureRepo(client1, bareRemote); err != nil {
		t.Fatal(err)
	}
	configureGitUser(t, client1)
	if err := os.WriteFile(filepath.Join(client1, "c1.txt"), []byte("from c1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CommitAll(client1, "c1 commit"); err != nil {
		t.Fatal(err)
	}
	if err := Push(client1); err != nil {
		t.Fatal(err)
	}

	// Client 2 initializes independently and tries normal push (fails without pull)
	if err := EnsureRepo(client2, bareRemote); err != nil {
		t.Fatal(err)
	}
	configureGitUser(t, client2)
	if err := os.WriteFile(filepath.Join(client2, "c2.txt"), []byte("from c2"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CommitAll(client2, "c2 commit"); err != nil {
		t.Fatal(err)
	}

	// Normal push should fail
	if err := Push(client2); err == nil {
		t.Fatal("expected normal push to fail due to remote divergence")
	}

	// Force push should succeed
	if err := PushWithOptions(client2, PushOptions{Force: true}); err != nil {
		t.Fatalf("expected force push to succeed, got: %v", err)
	}
}

func TestPullRebaseAndConflictAbort(t *testing.T) {
	tmp := t.TempDir()
	bareRemote := filepath.Join(tmp, "remote.git")
	client1 := filepath.Join(tmp, "client1")
	client2 := filepath.Join(tmp, "client2")

	if err := os.MkdirAll(bareRemote, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := GitRun(bareRemote, "init", "--bare"); err != nil {
		t.Fatal(err)
	}

	// Client 1 pushes file with content "foo"
	if err := EnsureRepo(client1, bareRemote); err != nil {
		t.Fatal(err)
	}
	configureGitUser(t, client1)
	if err := os.WriteFile(filepath.Join(client1, "conflict.txt"), []byte("remote foo"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CommitAll(client1, "remote commit"); err != nil {
		t.Fatal(err)
	}
	if err := Push(client1); err != nil {
		t.Fatal(err)
	}

	// Client 2 pushes file with conflicting content "bar"
	if err := EnsureRepo(client2, bareRemote); err != nil {
		t.Fatal(err)
	}
	configureGitUser(t, client2)
	if err := os.WriteFile(filepath.Join(client2, "conflict.txt"), []byte("local bar"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CommitAll(client2, "local commit"); err != nil {
		t.Fatal(err)
	}

	// Pull should fail due to conflict and abort rebase automatically
	err := Pull(client2)
	if err == nil {
		t.Fatal("expected Pull to fail on merge conflict")
	}
	if IsRebaseInProgress(client2) {
		t.Fatal("expected rebase in progress to be aborted")
	}
}
