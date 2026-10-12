# Implementation Plan: Import Subcommand

Implement `myconfig import` to reverse-sync configuration files from target application directories (starting with Antigravity / Gemini) into the central `~/.my_config_cli/config/` repository without auto-pushing to Git.

---

### Phase 1: Core Importer Logic and Copy Utilities [STATUS: DONE]
- Goal: Implement the reverse-mapping import engine, file/directory copy helpers, and tool filtering in `internal/importer`.
- Tasks:
  - [x] Define typed enums for `ImportAction` (`ImportActionCopied`, `ImportActionSkipped`, `ImportActionAlreadyLinked`) and `Tool` (`ToolAntigravity`, `ToolAll`, etc.).
  - [x] Implement path resolution and recursive file/directory copier with permission preservation.
  - [x] Implement entry reverse evaluation: check target status, detect existing symlinks pointing to source, handle missing targets, and copy real files/directories to the source destination.
  - [x] Implement tool filtering logic to isolate Antigravity (`~/.gemini/...`) entries from other tool mappings.
  - [x] Write comprehensive unit tests in `internal/importer/import_test.go` covering file copies, directory trees, missing files, already-linked symlinks, and tool filtering.
- Done when: `go test ./internal/importer/...` passes and `./lint.sh` passes.

---

### Phase 2: CLI Command Integration [STATUS: DONE]
- Goal: Add `myconfig import` command in `cmd/import.go` wired to `rootCmd`.
- Tasks:
  - [x] Create `cmd/import.go` with Cobra command definition for `import`.
  - [x] Support `--tool` flag (defaulting to `antigravity`) to restrict imports to Antigravity mappings.
  - [x] Support `--dry-run` flag to preview copy operations without modifying files on disk.
  - [x] Display concise, formatted summary output for each entry (e.g. `COPIED target -> source`, `SKIP reason`).
  - [x] Print clear instruction at the end reminding the user to inspect changes and run `myconfig push` manually (no automatic git commits or pushes).
  - [x] Write unit tests in `cmd/import_test.go` testing command execution and flag handling.
- Done when: `./test.sh` and `./lint.sh` pass, and `./build.sh` produces a binary where `./myconfig import --help` works as expected.

---


### Phase 3: Documentation and Verification [STATUS: TODO]
- Goal: Update architecture documentation in `AGENTS.md` and verify end-to-end functionality.
- Tasks:
  - [ ] Update [AGENTS.md](AGENTS.md) with details on `myconfig import`, tool filtering, and reverse-mapping lifecycle.
  - [ ] Verify execution using temp environment fixtures replicating Antigravity directories.
  - [ ] Run full test suite (`./test.sh`), linter (`./lint.sh`), and compiler (`./build.sh`).
- Done when: `./test.sh`, `./lint.sh`, and `./build.sh` exit 0, and all documentation is up to date.

---

## Unknowns, risks, assumptions

- **Assumption (Antigravity Scope):** Antigravity mapping entries are identified by target paths targeting `~/.gemini/`. The `--tool` flag will default to `antigravity`.
- **Assumption (Symlink Safety):** If a target is already a symlink pointing to the config store (`source`), `import` will safely skip it and report `ALREADY_LINKED` to prevent circular copies or self-overwriting.
- **Assumption (Git Non-Interference):** `import` will only write to the filesystem under `~/.my_config_cli/config/`. It will not run git add, commit, or push. Manual push via `myconfig push` will be suggested in the CLI output.
