package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/dreamoutbox/my_config_cli/internal/config"
	"github.com/dreamoutbox/my_config_cli/internal/git"
)

func configureTestGitUser(t *testing.T, dir string) {
	t.Helper()
	if _, err := git.GitRun(dir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("configure git email: %v", err)
	}
	if _, err := git.GitRun(dir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("configure git name: %v", err)
	}
}

func TestPushWithAutoRebaseRemoteReadme(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	bareRemote := filepath.Join(tmp, "remote.git")
	if err := os.MkdirAll(bareRemote, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := git.GitRun(bareRemote, "init", "--bare", "-b", "main"); err != nil {
		if _, err := git.GitRun(bareRemote, "init", "--bare"); err != nil {
			t.Fatalf("init bare remote: %v", err)
		}
	}

	// Seed bare remote with a README commit (simulating GitHub repo creation with README)
	seedDir := filepath.Join(tmp, "seed")
	if err := git.EnsureRepo(seedDir, bareRemote); err != nil {
		t.Fatal(err)
	}
	configureTestGitUser(t, seedDir)
	_ = os.WriteFile(filepath.Join(seedDir, "README.md"), []byte("# Config Backup"), 0o600)
	if committed, err := git.CommitAll(seedDir, "Initial README commit"); err != nil || !committed {
		t.Fatalf("seed commit: committed=%v, err=%v", committed, err)
	}
	if err := git.Push(seedDir); err != nil {
		t.Fatalf("seed push: %v", err)
	}

	// Local user setup: populate local storeDir with config file
	storeDir, err := config.StoreDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(storeDir, "settings.json"), []byte("{\"theme\":\"dark\"}"), 0o600)

	// Ensure repo and set git user for commit
	if err := git.EnsureRepo(storeDir, bareRemote); err != nil {
		t.Fatal(err)
	}
	configureTestGitUser(t, storeDir)

	// Test 1: Push with auto-rebase (default) succeeds despite remote README
	repoFlag = bareRemote
	pushAutoRebaseFlag = true
	pushForceFlag = false

	var out bytes.Buffer
	pushCmd.SetOut(&out)
	pushCmd.SetErr(&out)

	if err := runPush(pushCmd); err != nil {
		t.Fatalf("expected runPush with auto-rebase to succeed, got: %v\nOutput: %s", err, out.String())
	}

	// Verify README was rebased into storeDir
	readmePath := filepath.Join(storeDir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Fatal("expected remote README.md to be present in local store after auto-rebase")
	}
}

func TestPushWithAutoRebaseDisabledFailsOnDivergence(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	bareRemote := filepath.Join(tmp, "remote.git")
	if err := os.MkdirAll(bareRemote, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := git.GitRun(bareRemote, "init", "--bare", "-b", "main"); err != nil {
		if _, err := git.GitRun(bareRemote, "init", "--bare"); err != nil {
			t.Fatal(err)
		}
	}

	// Seed bare remote
	seedDir := filepath.Join(tmp, "seed")
	if err := git.EnsureRepo(seedDir, bareRemote); err != nil {
		t.Fatal(err)
	}
	configureTestGitUser(t, seedDir)
	_ = os.WriteFile(filepath.Join(seedDir, "README.md"), []byte("# Config Backup"), 0o600)
	_, _ = git.CommitAll(seedDir, "Initial README commit")
	_ = git.Push(seedDir)

	// Local user setup
	storeDir, err := config.StoreDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(storeDir, "config.json"), []byte("{}"), 0o600)

	if err := git.EnsureRepo(storeDir, bareRemote); err != nil {
		t.Fatal(err)
	}
	configureTestGitUser(t, storeDir)

	// Run push with auto-rebase = false
	repoFlag = bareRemote
	pushAutoRebaseFlag = false
	pushForceFlag = false

	var out bytes.Buffer
	pushCmd.SetOut(&out)
	pushCmd.SetErr(&out)

	if err := runPush(pushCmd); err == nil {
		t.Fatal("expected runPush to fail when auto-rebase is disabled and remote has diverged")
	}
}

func TestPushWithForceSucceeds(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	bareRemote := filepath.Join(tmp, "remote.git")
	if err := os.MkdirAll(bareRemote, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := git.GitRun(bareRemote, "init", "--bare", "-b", "main"); err != nil {
		if _, err := git.GitRun(bareRemote, "init", "--bare"); err != nil {
			t.Fatal(err)
		}
	}

	// Seed bare remote
	seedDir := filepath.Join(tmp, "seed")
	if err := git.EnsureRepo(seedDir, bareRemote); err != nil {
		t.Fatal(err)
	}
	configureTestGitUser(t, seedDir)
	_ = os.WriteFile(filepath.Join(seedDir, "README.md"), []byte("# Old Readme"), 0o600)
	_, _ = git.CommitAll(seedDir, "Initial commit")
	_ = git.Push(seedDir)

	// Local user setup
	storeDir, err := config.StoreDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(storeDir, "override.txt"), []byte("new data"), 0o600)

	if err := git.EnsureRepo(storeDir, bareRemote); err != nil {
		t.Fatal(err)
	}
	configureTestGitUser(t, storeDir)

	// Run push with force = true
	repoFlag = bareRemote
	pushAutoRebaseFlag = false
	pushForceFlag = true

	var out bytes.Buffer
	pushCmd.SetOut(&out)
	pushCmd.SetErr(&out)

	if err := runPush(pushCmd); err != nil {
		t.Fatalf("expected runPush with force to succeed, got: %v", err)
	}
}

func TestPushToEmptyRemote(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	bareRemote := filepath.Join(tmp, "remote.git")
	if err := os.MkdirAll(bareRemote, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := git.GitRun(bareRemote, "init", "--bare"); err != nil {
		t.Fatal(err)
	}

	// Local user setup
	storeDir, err := config.StoreDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(storeDir, "first.txt"), []byte("hello"), 0o600)

	if err := git.EnsureRepo(storeDir, bareRemote); err != nil {
		t.Fatal(err)
	}
	configureTestGitUser(t, storeDir)

	// Run push with auto-rebase (default) on empty remote
	repoFlag = bareRemote
	pushAutoRebaseFlag = true
	pushForceFlag = false

	var out bytes.Buffer
	pushCmd.SetOut(&out)
	pushCmd.SetErr(&out)

	if err := runPush(pushCmd); err != nil {
		t.Fatalf("expected runPush on empty remote to succeed, got: %v", err)
	}
}
