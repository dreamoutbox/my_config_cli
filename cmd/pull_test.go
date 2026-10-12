package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dreamoutbox/my_config_cli/internal/config"
	"github.com/dreamoutbox/my_config_cli/internal/git"
)

func TestRunPullAndFromCommit(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Configure dedicated test mapping
	baseDir, err := config.BaseDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	testMapping := config.Mapping{
		Entries: []config.Entry{
			{
				Source: "~/.my_config_cli/config/AGENTS.md",
				Target: "~/.gemini/GEMINI.md",
				Type:   config.EntryTypeSymlink,
			},
		},
	}
	mapData, err := json.Marshal(testMapping)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "mapping.json"), mapData, 0o600); err != nil {
		t.Fatal(err)
	}

	bareRemote := filepath.Join(tmp, "remote.git")
	if err := os.MkdirAll(bareRemote, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := git.GitRun(bareRemote, "init", "--bare", "-b", "main"); err != nil {
		if _, err := git.GitRun(bareRemote, "init", "--bare"); err != nil {
			t.Fatalf("init bare remote: %v", err)
		}
	}

	// Seed bare remote with two commits modifying GEMINI.md
	seedDir := filepath.Join(tmp, "seed")
	if err := git.EnsureRepo(seedDir, bareRemote); err != nil {
		t.Fatal(err)
	}
	configureTestGitUser(t, seedDir)

	// Commit 1: V1
	if err := os.WriteFile(filepath.Join(seedDir, "mcp_config.json"), []byte("{\"mcpServers\":{}}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(seedDir, "AGENTS.md"), []byte("# Config V1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if committed, err := git.CommitAll(seedDir, "Commit V1"); err != nil || !committed {
		t.Fatalf("seed commit 1: committed=%v, err=%v", committed, err)
	}
	if err := git.Push(seedDir); err != nil {
		t.Fatalf("seed push 1: %v", err)
	}
	commit1, err := git.CurrentCommit(seedDir, false)
	if err != nil {
		t.Fatalf("get commit 1: %v", err)
	}

	// Commit 2: V2
	if err := os.WriteFile(filepath.Join(seedDir, "AGENTS.md"), []byte("# Config V2"), 0o600); err != nil {
		t.Fatal(err)
	}
	if committed, err := git.CommitAll(seedDir, "Commit V2"); err != nil || !committed {
		t.Fatalf("seed commit 2: committed=%v, err=%v", committed, err)
	}
	if err := git.Push(seedDir); err != nil {
		t.Fatalf("seed push 2: %v", err)
	}

	// 1. Initial normal pull pulls latest (V2)
	repoFlag = bareRemote
	pullFromCommitFlag = ""

	var out bytes.Buffer
	pullCmd.SetOut(&out)
	pullCmd.SetErr(&out)

	if err := runPull(pullCmd); err != nil {
		t.Fatalf("initial runPull: %v\nOutput: %s", err, out.String())
	}

	targetFile := filepath.Join(tmp, ".gemini", "GEMINI.md")
	data, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("reading linked file: %v", err)
	}
	if string(data) != "# Config V2" {
		t.Fatalf("expected V2 after normal pull, got %q", string(data))
	}

	// 2. Pull with --from-commit pointing to commit1
	out.Reset()
	pullFromCommitFlag = commit1

	if err := runPull(pullCmd); err != nil {
		t.Fatalf("runPull --from-commit: %v\nOutput: %s", err, out.String())
	}

	data, err = os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("reading linked file after --from-commit: %v", err)
	}
	if string(data) != "# Config V1" {
		t.Fatalf("expected V1 after pull --from-commit, got %q", string(data))
	}

	// 3. Pull with invalid --from-commit should fail
	out.Reset()
	pullFromCommitFlag = "non_existent_ref_99999"

	if err := runPull(pullCmd); err == nil {
		t.Fatal("expected runPull with invalid --from-commit to fail")
	}

	// 4. Return to latest using standard pull (pullFromCommitFlag = "")
	out.Reset()
	pullFromCommitFlag = ""

	if err := runPull(pullCmd); err != nil {
		t.Fatalf("restore latest runPull: %v\nOutput: %s", err, out.String())
	}

	data, err = os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("reading linked file after return pull: %v", err)
	}
	if string(data) != "# Config V2" {
		t.Fatalf("expected V2 after restoring latest, got %q", string(data))
	}

	storeDir, err := config.StoreDir()
	if err != nil {
		t.Fatal(err)
	}
	br, err := git.CurrentBranch(storeDir)
	if err != nil {
		t.Fatalf("get branch: %v", err)
	}
	if br == "" {
		t.Fatal("expected store repo to be reattached to branch after pull, got detached HEAD")
	}
}
