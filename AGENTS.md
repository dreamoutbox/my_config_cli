# AGENTS.md

Context, conventions, architecture, and cautions for AI coding agents working on `myconfig`.

---

## 1. Project Overview

`myconfig` is a Go CLI application that centralizes and synchronizes AI development tool configurations via a Git repository and local filesystem symlinks.

- **Binary Name:** `myconfig`
- **Module:** `github.com/dreamoutbox/my_config_cli`
- **Go Version:** 1.24+
- **CLI Framework:** `github.com/spf13/cobra`
- **Git Integration:** Wraps system `git` via `os/exec` (no external git library dependencies)

---

## 2. Architecture & Directory Layout

```
.
├── assets/
│   ├── assets.go          # Exposes DefaultMappingJSON via go:embed
│   └── mapping.json       # Default tool mapping definitions
├── cmd/
│   ├── root.go            # Root command and persistent flags (--repo)
│   ├── repo.go            # Repository URL resolution and settings persistence
│   ├── mapping.go         # 'myconfig mapping' (path, add) commands
│   ├── link.go            # 'myconfig link' command
│   ├── import.go          # 'myconfig import' command (reverse sync)
│   ├── push.go            # 'myconfig push' (aliases: save, backup)
│   ├── pull.go            # 'myconfig pull' (aliases: load, restore)
│   └── status.go          # 'myconfig status' (git state + mapping health)
├── internal/
│   ├── config/            # Settings (settings.json) and Mapping (mapping.json) loaders
│   ├── git/               # System git CLI wrappers (init, remote, commit, push, pull, status)
│   ├── importer/          # Reverse-mapping importer engine and copy utilities
│   ├── pathutil/          # Tilde (~) path expansion helper
│   └── symlink/           # Symlink creation, health check, and json_merge engine
├── docker/
│   ├── tests/             # Docker integration test scripts
│   └── Dockerfile         # Ubuntu 24.04 test environment
├── docker-test.sh         # Runs integration tests in Ubuntu 24 container
├── build.sh               # Compiles ./myconfig
├── lint.sh                # Runs go vet ./...
├── test.sh                # Runs all unit tests
├── install.sh             # Installs to ~/.local/bin
├── release.sh             # Bumps version and tags release commit
├── build-release.sh       # Compiles release binaries for multiple platforms
├── VERSION                # Semantic version string
└── main.go                # Application entrypoint
```

---

## 3. Data Storage & Lifecycle

All CLI configuration and sync stores live under the user's home directory:
- `~/.my_config_cli/settings.json`: Stores user settings such as `repo_url`.
- `~/.my_config_cli/mapping.json`: Stores mapping relationships. Seeded from embedded `assets/mapping.json` on first run if missing.
- `~/.my_config_cli/config/custom_mapping.json`: Git-tracked custom mappings loaded and merged on top of `mapping.json`.
- `~/.my_config_cli/config/`: Central directory tracked by git containing the canonical configuration files and directories.

### Command Aliases and Operations
- `myconfig mapping`: Manage and inspect mappings. Default prints paths for `mapping.json` and `custom_mapping.json`.
  - `myconfig mapping path`: Supports `--default` and `--custom` flags to output individual paths.
  - `myconfig mapping get` (aliases: `list`, `ls`): Lists all custom entries defined in `custom_mapping.json`.
  - `myconfig mapping add <target>`: Adds a custom file or directory entry to `custom_mapping.json`. Automatically infers `file` or `dir`, copies existing target files into the config store if absent, and derives store paths under `~/.my_config_cli/config/`.
  - `myconfig mapping delete <target>` (aliases: `remove`, `rm`): Deletes a custom mapping entry from `custom_mapping.json`.
- `myconfig push`: Aliased as `myconfig save` and `myconfig backup`. Commits and pushes the central store to git.
  - Auto-adoption: On a fresh machine where configuration files (like `~/.gitconfig` and `~/.bashrc`) exist on the host but not in the central store, `myconfig push` automatically adopts them into the store before committing and pushing. Host target files remain regular files and are not replaced with symlinks.
  - Supports `--auto-rebase` (default `true`): pulls and rebases remote changes (e.g. remote README or commits from another device) before pushing. Aborts cleanly if conflicts arise.
  - Supports `--force` / `-f`: force pushes to remote repository to overwrite remote changes if needed.
- `myconfig pull`: Aliased as `myconfig load` and `myconfig restore`. Pulls remote git changes and applies symlinks.
  - Supports `--from-commit <hash/ref>`: fetches remote objects and checks out a specific commit hash, tag, or branch instead of latest HEAD before applying symlinks.
  - Subsequent regular `myconfig pull` calls safely re-attach detached HEAD back to the repository's tracking/default branch before rebasing latest changes.
- `myconfig import`: Reverse-syncs configurations from system targets into the central store directory (`~/.my_config_cli/config/`).
  - Requires `--tool <tool>` (supported tools: `git`, `bashrc` / `bash`, `antigravity` / `gemini`, `claude`, `pi`, `opencode`, `all`).
  - Supports `--dry-run` to preview actions.
  - Safely skips existing symlinks that already point to the central store.
  - **No auto-push:** Does not run git commit or push; users must inspect status and manually execute `myconfig push`.

### Authentication Fallback
- Network commands (`pull`, `fetch`, `push`, `ls-remote`) run normally first. If git reports an access/403 error, they retry once interactively with `GH_TOKEN`/`GITHUB_TOKEN` removed, `GIT_ASKPASS`/`SSH_ASKPASS` cleared, and `-c credential.helper=` set (empty resets the helper list). The user is prompted by git for username/password.
- Credentials are never persisted by myconfig. A user's own credential helper may still offer to store them after the retry; that is outside myconfig.
- The retry is skipped when stdin is not a TTY (CI). The error then includes a hint to use a personal access token in `--repo`.
- 404 / `Repository not found` is not treated as an auth error and is never retried. GitHub also returns 404 for private repositories the current token cannot see, so the error mentions both wrong URL and insufficient access.
- All printed URLs and git error messages pass through `git.MaskURL`, which replaces `https://user:token@host` userinfo with `***@`.

### Repository Resolution Precedence
1. Command line flag `--repo <url>` (if passed, automatically persisted to `settings.json`).
2. Stored `repo_url` in `~/.my_config_cli/settings.json`.
3. If neither is available, commands return an explicit error requiring `--repo`.

> A PAT embedded in `--repo` is saved to `settings.json` and configured as the git remote URL. Prefer SSH or a credential helper; myconfig masks the token in output but does not remove it from disk.

---

## 4. Mapping Types & Target Tools

Mappings support two distinct entry types:

1. `symlink`:
   - Target is replaced with a filesystem symbolic link pointing to the source file or directory.
   - Parent directories are created automatically (`0755`).
   - Existing symlinks or stale files/directories at the target are removed and replaced silently.
   - If the source is missing, a placeholder is created in the store using the entry's `source_kind` (`file` or `dir`). JSON placeholders contain `{}`; other files are empty. The result is reported as `created`, and the user should run `myconfig push` to save the placeholders.
   - Mappings without `source_kind` (older user mappings) default to `file`; an existing source directory is left untouched.

2. `json_merge`:
   - **Crucial for OpenCode:** OpenCode stores MCP server configurations inside `~/.config/opencode/opencode.json` rather than a standalone file.
   - `json_merge` parses `mcp_config.json` from the source, extracts or takes the `mcpServers` object, and updates only the `mcpServers` key in `opencode.json` without modifying other configuration fields.
   - Writes are atomic (temp file + rename). If the target is a symlink, it is removed first so a real file is written and a dangling link is never followed.
   - Never replace `opencode.json` with a symlink. `config.LoadMapping` drops (and warns about) any `symlink` entry whose target is also a `json_merge` target, persisting the cleaned mapping once. This migrates users seeded before the conflict was removed from `assets/mapping.json`.

Supported tools in default mappings:
- Gemini CLI (`.gemini/antigravity-cli/settings.json`, `config/config.json`, `GEMINI.md`, `mcp_config.json`, `skills/`)
- Pi Agent (`.pi/agent/settings.json`, `SYSTEM.md`, `mcp.json`, `skills/`)
- OpenCode (`.config/opencode/opencode.json`, `AGENTS.md`, `skills/`, MCP merge)
- Claude (`.claude/config.json`, `CLAUDE.md`, `mcp.json`, `skills/`)
- Shell & Git (`~/.bashrc`, `~/.gitconfig`)

---

## 5. Development & Testing Conventions

- **Build & Verify:**
  ```bash
  ./lint.sh        # Runs go vet ./...
  ./test.sh        # Runs all unit tests
  ./docker-test.sh # Runs Ubuntu 24 integration tests in Docker
  ./build.sh       # Compiles ./myconfig
  ```
- **Tests:** Always maintain full unit test coverage for new components.
  - `internal/git/git_test.go` uses local temp bare repositories to verify push/pull mechanics without network access.
  - `internal/config/settings_test.go` and `internal/symlink/link_test.go` isolate `HOME` using `t.TempDir()` and `t.Setenv("HOME", tmp)`.
- **Enums:** Use typed constants for closed variant sets (e.g. `EntryType`, `Action`, `HealthStatus`), avoiding loose string literals.
- **Portability:** Never hardcode machine-specific paths (e.g. `/home/<user>/`). Always use `~` in config/mapping patterns and expand dynamically via `pathutil.Expand`.
- **Git Identity:** Unit tests creating git commits must explicitly configure local repo `user.name` and `user.email` to avoid host environment dependencies.

---

## 6. Cautions & Pitfalls

- **System Git Required:** The CLI relies on the system `git` binary and the host system's credentials (SSH keys / credential helpers). Do not replace this with a Go git library unless explicitly requested.
- **Auth Retry Scope:** Only `pull`, `fetch`, `push`, and `ls-remote` go through `RunWithAuthRetry`. Never use it for local commands (commit, add, checkout, status): a matching stderr pattern would trigger a credential prompt for a local failure.
- **Do Not Symlink Composite Configs:** OpenCode requires merging the `mcpServers` JSON key into its monolithic config file. Creating a symlink over `opencode.json` will break user settings; conflicting entries are dropped at load time.
- **Placeholders Are Real Files:** Missing sources are created in the store, not skipped. `link`/`pull` report them as `created`; remind users to run `myconfig push` so other machines receive the placeholders.
- **Path Expansion:** When reading or checking paths from `mapping.json`, always route through `pathutil.Expand` before performing filesystem operations.
- **Embed Limitations:** In Go, `//go:embed` patterns cannot traverse parent directories (`..`). All embedded assets must reside within or under the declaring package directory (hence `assets/assets.go`).
