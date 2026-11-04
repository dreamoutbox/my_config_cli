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
	if _, err := git.GitRun(bareRemote, "init", "--bare", "-b", "main"); err != nil {
		if _, err := git.GitRun(bareRemote, "init", "--bare"); err != nil {
			t.Fatal(err)
		}
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

func TestPushAdoptsGitconfigAndBashrc(t *testing.T) {
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

	// Create real target files on user's system
	gitconfigTarget := filepath.Join(tmp, ".gitconfig")
	if err := os.WriteFile(gitconfigTarget, []byte("[user]\nname = Tester\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	bashrcTarget := filepath.Join(tmp, ".bashrc")
	if err := os.WriteFile(bashrcTarget, []byte("export FOO=bar\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Pre-configure global git user so commit succeeds in storeDir
	if _, err := git.GitRun(tmp, "config", "--global", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := git.GitRun(tmp, "config", "--global", "user.name", "Test User"); err != nil {
		t.Fatal(err)
	}

	repoFlag = bareRemote
	pushAutoRebaseFlag = true
	pushForceFlag = false

	var out bytes.Buffer
	pushCmd.SetOut(&out)
	pushCmd.SetErr(&out)

	if err := runPush(pushCmd); err != nil {
		t.Fatalf("expected runPush to adopt and succeed, got: %v\nOutput: %s", err, out.String())
	}

	storeDir, err := config.StoreDir()
	if err != nil {
		t.Fatal(err)
	}

	// Verify store files exist and have original content
	storeGitconfig := filepath.Join(storeDir, ".gitconfig")
	storeBashrc := filepath.Join(storeDir, ".bashrc")
	gcData, err := os.ReadFile(storeGitconfig)
	if err != nil {
		t.Fatalf("reading store gitconfig: %v", err)
	}
	if !bytes.Contains(gcData, []byte("Test User")) || !bytes.Contains(gcData, []byte("test@example.com")) {
		t.Fatalf("unexpected store gitconfig content: %q", string(gcData))
	}
	bashData, err := os.ReadFile(storeBashrc)
	if err != nil {
		t.Fatalf("reading store bashrc: %v", err)
	}
	if string(bashData) != "export FOO=bar\n" {
		t.Fatalf("unexpected store bashrc content: %q", string(bashData))
	}

	// Verify host target files remain regular files (NOT replaced with symlinks)
	gcLstat, err := os.Lstat(gitconfigTarget)
	if err != nil {
		t.Fatal(err)
	}
	if gcLstat.Mode()&os.ModeSymlink != 0 {
		t.Fatal("gitconfig target was replaced with a symlink; expected regular file")
	}
	bashLstat, err := os.Lstat(bashrcTarget)
	if err != nil {
		t.Fatal(err)
	}
	if bashLstat.Mode()&os.ModeSymlink != 0 {
		t.Fatal("bashrc target was replaced with a symlink; expected regular file")
	}

	// Verify cloned remote contains the adopted files
	cloneDir := filepath.Join(tmp, "clone")
	if _, err := git.GitRun(tmp, "clone", bareRemote, cloneDir); err != nil {
		t.Fatalf("cloning remote: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cloneDir, ".gitconfig")); err != nil {
		t.Fatalf("remote clone missing .gitconfig: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cloneDir, ".bashrc")); err != nil {
		t.Fatalf("remote clone missing .bashrc: %v", err)
	}
}
