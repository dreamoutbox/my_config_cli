# Implementation Plan: Pull by Specific Commit

Add `--from-commit` flag to `myconfig pull` (and aliases `load`, `restore`) allowing users to check out and apply configurations from a specific Git commit hash, tag, or ref.

---

### Phase 1: Git Foundation - Remote Fetch and Ref Checkout [STATUS: DONE]
- Goal: Implement git fetch, checkout, and detached-HEAD recovery helpers in `internal/git`.
- Tasks:
  - [x] Add `Fetch(dir string) error` in `internal/git/git.go` executing `git fetch origin` to retrieve remote objects and tags without altering the working directory.
  - [x] Add `Checkout(dir, ref string) error` in `internal/git/git.go` executing `git checkout <ref>` to switch the store repository to the target commit, tag, or branch.
  - [x] Update `Pull(dir string) error` in `internal/git/git.go` to handle detached HEAD state by checking out the tracking/default branch before rebasing from `origin HEAD`.
  - [x] Add unit tests in `internal/git/git_test.go` covering `Fetch`, `Checkout` with commit hashes, detached HEAD state, and returning to the tracking branch with local bare repositories.
- Done when: `go test ./internal/git/...` passes and `./lint.sh` passes.

---

### Phase 2: CLI Integration - `--from-commit` Flag in `cmd/pull.go` [STATUS: DONE]
- Goal: Add `--from-commit` flag to `myconfig pull` and invoke fetch + checkout flow prior to applying symlinks and merges.
- Tasks:
  - [x] Declare `pullFromCommitFlag string` in `cmd/pull.go` and register `--from-commit` flag on `pullCmd`.
  - [x] Update `runPull` in `cmd/pull.go` to branch logic based on `pullFromCommitFlag`:
    - When `--from-commit` is non-empty: run `git.Fetch(storeDir)` followed by `git.Checkout(storeDir, pullFromCommitFlag)`, then display confirmation with the target commit.
    - When `--from-commit` is empty: run existing `git.Pull(storeDir)` workflow.
    - If checkout fails, abort immediately without applying links or merges.
  - [x] Retain downstream `runLink(cmd)` call so symlinks and merges are applied from the checked-out commit.
  - [x] Create `cmd/pull_test.go` verifying `pull` defaults to latest, `pull --from-commit <hash>` checks out and applies historic commit configurations, and invalid commits return an error without modifying links.
- Done when: `go test ./cmd/...` passes, `./lint.sh` passes, and `./build.sh` produces `./myconfig` with `--from-commit` flag registered on `pull`.

---

### Phase 3: Documentation and Verification [STATUS: TODO]
- Goal: Update documentation in `AGENTS.md` and `README.md` and verify end-to-end functionality.
- Tasks:
  - [ ] Update `AGENTS.md` to document the `--from-commit` flag lifecycle and detached HEAD handling during pull operations.
  - [ ] Update `README.md` usage section with examples for `myconfig pull --from-commit <hash>`.
  - [ ] Run full test suite (`./test.sh`), linter (`./lint.sh`), and binary build (`./build.sh`).
- Done when: `./test.sh`, `./lint.sh`, and `./build.sh` exit 0, and all documentation reflects the new flag.

---

## Unknowns, risks, assumptions

- **Assumption (Ref Specification):** `--from-commit` accepts any valid git refspec or object identifier (short SHA, full SHA, tag name, or branch).
- **Assumption (Remote Fetching):** `myconfig pull --from-commit <ref>` will run `git fetch origin` first so remote commits not yet present locally are retrieved. If the ref cannot be found, git checkout errors and `myconfig` aborts before touching symlinks.
- **Assumption (Re-attaching on Regular Pull):** Running a subsequent `myconfig pull` without `--from-commit` will switch from detached HEAD back to the repository's tracking/default branch before pulling latest changes.
- **Risk (Dirty Working Tree):** If uncommitted changes exist in `~/.my_config_cli/config/`, `git checkout` will fail to avoid overwriting files. The git error message will be surfaced directly to the user.
