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
- `~/.my_config_cli/config/`: Central directory tracked by git containing the canonical configuration files and directories.

### Command Aliases and Operations
- `myconfig push`: Aliased as `myconfig save` and `myconfig backup`. Commits and pushes the central store to git.
- `myconfig pull`: Aliased as `myconfig load` and `myconfig restore`. Pulls remote git changes and applies symlinks.
- `myconfig import`: Reverse-syncs configurations from system targets into the central store directory (`~/.my_config_cli/config/`).
  - Supports `--tool <tool>` (defaults to `antigravity`).
  - Supports `--dry-run` to preview actions.
  - Safely skips existing symlinks that already point to the central store.
  - **No auto-push:** Does not run git commit or push; users must inspect status and manually execute `myconfig push`.

### Repository Resolution Precedence
1. Command line flag `--repo <url>` (if passed, automatically persisted to `settings.json`).
2. Stored `repo_url` in `~/.my_config_cli/settings.json`.
3. If neither is available, commands return an explicit error requiring `--repo`.

---

## 4. Mapping Types & Target Tools

Mappings support two distinct entry types:

1. `symlink`:
   - Target is replaced with a filesystem symbolic link pointing to the source file or directory.
   - Parent directories are created automatically (`0755`).
   - Existing symlinks or stale files/directories at the target are removed and replaced silently.

2. `json_merge`:
   - **Crucial for OpenCode:** OpenCode stores MCP server configurations inside `~/.config/opencode/opencode.json` rather than a standalone file.
   - `json_merge` parses `mcp_config.json` from the source, extracts or takes the `mcpServers` object, and updates only the `mcpServers` key in `opencode.json` without modifying other configuration fields.
   - Never replace `opencode.json` with a symlink.

Supported tools in default mappings:
- Gemini CLI (`.gemini/antigravity-cli/settings.json`, `config/config.json`, `GEMINI.md`, `mcp_config.json`, `skills/`)
- Pi Agent (`.pi/agent/settings.json`, `SYSTEM.md`, `mcp.json`, `skills/`)
- OpenCode (`.config/opencode/opencode.json`, `AGENTS.md`, `skills/`, MCP merge)
- Claude (`.claude/config.json`, `CLAUDE.md`, `mcp.json`, `skills/`)

---

## 5. Development & Testing Conventions

- **Build & Verify:**
  ```bash
  ./lint.sh    # Runs go vet ./...
  ./test.sh    # Runs all unit tests
  ./build.sh   # Compiles ./myconfig
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
- **Do Not Symlink Composite Configs:** OpenCode requires merging the `mcpServers` JSON key into its monolithic config file. Creating a symlink over `opencode.json` will break user settings.
- **Path Expansion:** When reading or checking paths from `mapping.json`, always route through `pathutil.Expand` before performing filesystem operations.
- **Embed Limitations:** In Go, `//go:embed` patterns cannot traverse parent directories (`..`). All embedded assets must reside within or under the declaring package directory (hence `assets/assets.go`).
