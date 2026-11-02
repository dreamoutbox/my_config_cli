# Plan: TODO Implementation (`PLAN-todo.md`)

Implementation plan for items in `TODO.md`:
1. Save `~/.bashrc` and `~/.gitconfig` in default mappings.
2. Support a git-tracked custom mapping file (`~/.my_config_cli/config/custom_mapping.json`) alongside the default `~/.my_config_cli/mapping.json`.
3. Add `myconfig mapping` (with `path` and `add` subcommands) to show mapping file paths and register custom file or directory mappings.

---

### Phase 1: Default `.bashrc` and `.gitconfig` Mappings & Embedded Default Migration [STATUS: DONE]
- Goal: Add `~/.bashrc` and `~/.gitconfig` to the embedded `assets/mapping.json` and ensure existing installations receive newly added embedded default entries automatically.
- Tasks:
  - [x] Add symlink entries for `~/.my_config_cli/config/.bashrc` -> `~/.bashrc` (`source_kind: "file"`) and `~/.my_config_cli/config/.gitconfig` -> `~/.gitconfig` (`source_kind: "file"`) to `assets/mapping.json`.
  - [x] Update `config.LoadMapping` in `internal/config/mapping.go` to backfill any missing embedded default entries (by target path) into an existing `~/.my_config_cli/mapping.json` and persist once so existing users get `.bashrc` and `.gitconfig` without losing customizations.
  - [x] Add unit tests in `internal/config/mapping_test.go` verifying `.bashrc` and `.gitconfig` are present in defaults and backfilled into pre-existing `mapping.json` files.
- Done when: `./lint.sh` and `./test.sh` pass, and `LoadMapping()` returns `.bashrc` and `.gitconfig` entries on both fresh and existing setups.

### Phase 2: Git-Tracked Custom Mapping (`custom_mapping.json`) [STATUS: DONE]
- Goal: Load and merge optional custom mappings from `~/.my_config_cli/config/custom_mapping.json` so custom file and directory mappings are versioned and synced via `myconfig push` and `myconfig pull`.
- Tasks:
  - [x] Add path helpers `DefaultMappingPath()` (`~/.my_config_cli/mapping.json`) and `CustomMappingPath()` (`~/.my_config_cli/config/custom_mapping.json`) in `internal/config/mapping.go`.
  - [x] Implement `LoadCustomMapping()` and `SaveCustomMapping(Mapping)` in `internal/config/mapping.go` (returning empty `Mapping` when `custom_mapping.json` does not exist yet).
  - [x] Update `LoadMapping()` to merge entries from `CustomMappingPath()` after default `mapping.json` entries (deduplicating/overriding by `Target` if a custom entry targets the same path, and running `dropConflictingSymlinks` across the combined set).
  - [x] Support auto-detecting `source_kind` (`file` vs `dir`) when applying or importing custom entries if `source_kind` is omitted and the target or source already exists on disk.
  - [x] Add unit tests in `internal/config/mapping_test.go` and `internal/symlink/link_test.go` covering custom file and directory entries, missing `custom_mapping.json`, and target override behavior.
- Done when: `./lint.sh` and `./test.sh` pass, and `link`, `status`, `pull`, and `import --tool all` operate seamlessly across both `mapping.json` and `custom_mapping.json`.

### Phase 3: `myconfig mapping` CLI Command (`path` and `add`) [STATUS: TODO]
- Goal: Provide CLI commands to display mapping paths and add custom file or directory mappings into the git-tracked `custom_mapping.json`.
- Tasks:
  - [ ] Create `cmd/mapping.go` with `myconfig mapping` (default action prints both default and custom mapping paths) and subcommand `myconfig mapping path`.
  - [ ] Add `myconfig mapping add <target>` (or `--target` / `--source` / `--kind file|dir`) that:
    - Infers `source_kind` (`file` or `dir`) from the existing filesystem path (or uses `--kind` when the path does not exist yet).
    - Derives the store source path under `~/.my_config_cli/config/` (preserving home-relative path structure, e.g. `~/.vimrc` -> `~/.my_config_cli/config/.vimrc`, `~/.config/nvim` -> `~/.my_config_cli/config/.config/nvim`) unless `--source` is explicitly provided.
    - If the target currently exists as a regular file or directory and the store source does not exist yet, copies the target into the store so data is preserved before linking.
    - Appends or updates the entry in `~/.my_config_cli/config/custom_mapping.json`.
  - [ ] Add unit tests for `cmd/mapping.go` and update `README.md`, `AGENTS.md`, and `TODO.md` to document the new mappings and commands.
- Done when: `./lint.sh`, `./test.sh`, and `./build.sh` succeed, `myconfig mapping` outputs the mapping file paths, and `myconfig mapping add` supports both files and directories.

---

## Unknowns, risks, assumptions
- **Assumption (Store path derivation in `mapping add`):** When adding a target inside `~` (e.g. `~/.bashrc` or `~/.config/fish`), its store path defaults to `~/.my_config_cli/config/<relative-to-home>`. If a target is outside `~`, `--source` must be specified (or it defaults to `~/.my_config_cli/config/<basename>`).
- **Assumption (Custom mapping persistence):** `custom_mapping.json` is only created on disk when the user creates it or runs `myconfig mapping add`, keeping fresh repos clean if no custom mappings are used.
- **Risk (Existing user `mapping.json`):** Backfilling new default entries into `~/.my_config_cli/mapping.json` matches by `Target` so any user-modified `Source` for an existing target is preserved untouched.

