# PLAN: myconfig CLI

> Binary: `myconfig` | Module: `github.com/dreamoutbox/my_config_cli`
> Config root: `~/.my_config_cli/` | Git: shell out to system `git`

---

## Decisions Log

| Decision | Choice |
|---|---|
| Binary name | `myconfig` |
| Config root | `~/.my_config_cli/` |
| Git integration | Shell out to system `git` |
| Symlink creation | On `pull` + standalone `myconfig link` |
| Symlink conflict | Overwrite silently (rm + re-create) |
| opencode MCP | Inject/replace only `mcpServers` key in existing opencode.json |
| Primary commands | `push` / `pull` (aliases: save/backup and load/restore respectively) |
| `status` command | Yes — shows dirty files + symlink health |
| Default mapping.json | Embedded in binary via `go:embed` |

---

## Phase 1: Project Scaffold [STATUS: DONE]

- Goal: Compilable, lintable skeleton with cobra wired up.
- Tasks:
  - [x] `go get github.com/spf13/cobra`
  - [x] Create `cmd/root.go` — root command, persistent flags (`--repo`)
  - [x] Create `main.go` — calls `cmd.Execute()`
  - [x] Add `Makefile` with `build`, `lint`, `test` targets
  - [x] Verify `go build ./...` succeeds and `myconfig --help` prints usage
- Done when: `go build -o myconfig .` succeeds; `./myconfig --help` lists subcommands scaffold.

---

## Phase 2: Config & Mapping Layer [STATUS: DONE]

- Goal: Load/save `settings.json` and `mapping.json` from `~/.my_config_cli/`.
- Tasks:
  - [x] Create `internal/config/settings.go` — `Settings` struct (`RepoURL string`); read/write `~/.my_config_cli/settings.json`
  - [x] Embed `assets/mapping.json` (default mappings from draft) using `//go:embed`
  - [x] Create `internal/config/mapping.go` — `Mapping` struct; read from `~/.my_config_cli/mapping.json`, fall back to embedded default if missing, write it on first run
  - [x] Unit-test: settings round-trip, mapping fallback to embedded default
- Done when: `go test ./internal/config/...` passes.

---

## Phase 3: Symlink Engine [STATUS: DONE]

- Goal: Create/repair all symlinks defined in `mapping.json`.
- Tasks:
  - [x] Create `internal/symlink/link.go` — for each mapping entry: expand `~`, create parent dirs, remove existing symlink/file, create new symlink
  - [x] Handle `mapping.json` entry types: plain file symlink and directory symlink
  - [x] Handle opencode special case: entries tagged `type: json_merge` — read target JSON, replace `mcpServers` key from source file, write back (no symlink for this entry)
  - [x] Unit-test: symlink creation, stale overwrite, json_merge injection
- Done when: `go test ./internal/symlink/...` passes; manual smoke test on one mapping entry works.

---

## Phase 4: `link` Command [STATUS: DONE]

- Goal: Expose symlink engine as `myconfig link`.
- Tasks:
  - [x] Create `cmd/link.go` — calls symlink engine on all mapping entries, prints a summary line per entry (`linked`, `replaced`, `merged`)
  - [x] Wire into root command
- Done when: `myconfig link` creates all symlinks and prints per-entry status.

---

## Phase 5: Git Engine [STATUS: DONE]

- Goal: Wrap system `git` for init, commit-all, push, and pull of `~/.my_config_cli/config/`.
- Tasks:
  - [x] Create `internal/git/git.go` — `Run(dir string, args ...string) error` helper (exec + stderr passthrough)
  - [x] `EnsureRepo(dir, repoURL string) error` — `git init` + `git remote add/set-url origin` if not already a repo
  - [x] `CommitAll(dir, message string) error` — `git add -A`, `git commit -m`
  - [x] `Push(dir string) error` — `git push -u origin HEAD`
  - [x] `Pull(dir string) error` — `git pull --rebase origin HEAD`
  - [x] Unit-test with a temp dir: init, commit, push (mocked), pull (mocked)
- Done when: `go test ./internal/git/...` passes.

---

## Phase 6: `push` Command [STATUS: DONE]

- Goal: Commit and push `~/.my_config_cli/config/` to the configured git remote.
- Tasks:
  - [x] Create `cmd/push.go` — `--repo` flag (overrides settings); resolve repo URL (flag -> settings -> error); call `EnsureRepo`, `CommitAll`, `Push`; persist repo URL back to settings
  - [x] Register aliases `save` and `backup` pointing to the same command
  - [x] Wire into root command
- Done when: `myconfig push --repo <url>` commits and pushes; second run without `--repo` succeeds using stored URL.

---

## Phase 7: `pull` Command [STATUS: DONE]

- Goal: Pull latest config from git, then auto-apply all symlinks.
- Tasks:
  - [x] Create `cmd/pull.go` — `--repo` flag; resolve repo URL same as push; call `EnsureRepo`, `Pull`; then call symlink engine (same as `link`)
  - [x] Register aliases `load` and `restore`
  - [x] Wire into root command
- Done when: `myconfig pull` fetches config and recreates all symlinks.

---

## Phase 8: `status` Command [STATUS: DONE]

- Goal: Show pending git changes and symlink health at a glance.
- Tasks:
  - [x] Create `cmd/status.go`
  - [x] Section 1 — Git status: run `git status --short` in config dir, display output
  - [x] Section 2 — Symlink health: for each mapping entry, check if target path exists and points to the right source; print `OK`, `MISSING`, `STALE`, or `MERGED` per entry
- Done when: `myconfig status` prints both sections correctly for healthy and broken states.

---

## Phase 9: Polish & Release [STATUS: DONE]

- Goal: Build artifact, README, and install instructions.
- Tasks:
  - [x] Write `README.md` covering: install, `myconfig link`, `myconfig push`, `myconfig pull`, `myconfig status`, mapping.json customisation
  - [x] Add Makefile targets (`build`, `lint`, `test`, `install`)
  - [x] Verify `go vet ./...` and `go build ./...` clean on Linux
- Done when: `go install ./...` installs `myconfig`; README covers all commands.

---

## Unknowns, Risks, Assumptions

| Item | Note |
|---|---|
| `mapping.json` schema | Need to finalise field names (`source`, `target`, `type`). Phase 2 defines this; later phases depend on it. |
| opencode `json_merge` key name | Draft says `mcpServers` — verify against actual opencode.json schema before Phase 3. |
| Git auth | Relies entirely on the user's system git credential helper; no auth logic in the CLI. |
| `go 1.27.1` in go.mod | `go:embed` available since 1.16 — no issue. Verify `go 1.27.1` is valid syntax (likely should be `1.24` or similar; fix in Phase 1). |
| Windows | Not targeted; tilde expansion and symlinks are Unix-only. |
