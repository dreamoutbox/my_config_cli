# myconfig

CLI tool to backup, restore, and synchronize AI tool configurations across environments via Git and symlinks.

## Features

- **Centralized storage**: Stores all shared configurations in `~/.my_config_cli/config`.
- **Git synchronization**: Push and pull configuration repositories using system `git`.
- **Private repository support**: If ambient credentials are rejected, retry once with an interactive username/password prompt.
- **Automatic symlinking**: Automatically symlinks configs and skills to supported AI tools (Gemini CLI, Pi agent, OpenCode, Claude), creating placeholders when a tool or source file is missing.
- **JSON configuration merge**: Merges MCP server definitions into composite configs like `opencode.json` without overwriting unrelated settings.
- **Health monitoring**: Inspect repository status and symlink health at any time.

## Installation

### Using curl

Install `myconfig` to `~/.local/bin`:

```bash
curl -fsSL https://raw.githubusercontent.com/dreamoutbox/my_config_cli/main/install.sh | sh
```

### From Source

Requires Go 1.24+.

```bash
./build.sh
# or install directly into ~/.local/bin
./install.sh
```

## Usage

### 1. Push / Backup

Upload configurations from `~/.my_config_cli/config` to your git repository:

```bash
myconfig push --repo git@github.com:username/my-config.git
```

On a fresh machine, `myconfig push` automatically adopts mapped files present on the system (such as `~/.gitconfig` and `~/.bashrc`) into the central config store if they are not already present. Existing system files remain regular files and are not replaced with symlinks.

The repository URL is saved to `~/.my_config_cli/settings.json`, so subsequent runs do not require `--repo`:

```bash
myconfig push
# Aliases:
myconfig save
myconfig backup
```

### 2. Pull / Restore

Pull configurations from git and immediately recreate all symlinks:

```bash
myconfig pull --repo git@github.com:username/my-config.git
# Or using the saved repository URL:
myconfig pull
# Pull configurations from a specific commit hash, tag, or ref:
myconfig pull --from-commit a1b2c3d
# Aliases:
myconfig load
myconfig restore
```

### 3. Link (Manual)

Apply all symlinks and JSON merges without pulling from git:

```bash
myconfig link
```

If a tool is not installed yet, or a source file is absent from the repository, `link` creates a placeholder in `~/.my_config_cli/config` (`{}` for JSON files, empty otherwise) and still creates the symlink. Placeholder entries are reported as `created`; run `myconfig push` to save them for other machines.

### 4. Status

Check git status of the configuration store and inspect the health of all symlink targets:

```bash
myconfig status
```

Status codes for mapping targets:
- `OK`: Target is an active symlink pointing to the correct source file/directory.
- `MISSING`: Target file or directory does not exist.
- `STALE`: Target is not a symlink, points to an invalid destination, or source file is missing.
- `MERGED`: Composite JSON target file exists with merged keys.

### 5. Custom Mappings

View mapping paths and register custom file or directory mappings:

```bash
# Print mapping file paths:
myconfig mapping
# or
myconfig mapping path

# List all custom mappings:
myconfig mapping get
# (aliases: list, ls)

# Add custom mappings (automatically copied to config store and tracked in git):
myconfig mapping add ~/.vimrc
myconfig mapping add ~/.config/fish
myconfig mapping add ~/.tmux.conf --kind file

# Delete a custom mapping:
myconfig mapping delete ~/.vimrc
# (aliases: remove, rm)
```

Custom mappings are saved to `~/.my_config_cli/config/custom_mapping.json` inside the git repository and merged alongside default mappings. Run `myconfig link` to apply symlinks and `myconfig push` to back them up to git.

## Private Repositories & Authentication

`myconfig pull` and `myconfig push` use the system `git` and whatever credentials it normally would (SSH keys, credential helpers, `GITHUB_TOKEN`/`GH_TOKEN`). If git rejects those credentials with a 403/access error, `myconfig` retries once with environment tokens and credential helpers disabled, so git can prompt for a username and password:

```
Access denied with default credentials; enter credentials for github.com
Username for 'https://github.com':
Password for 'https://alice@github.com':
```

- Credentials typed at the prompt are not stored by `myconfig`.
- The retry is skipped when there is no interactive terminal (for example in CI). Use a personal access token in `--repo` for those environments.
- A 404 or `Repository not found` is not retried. GitHub returns 404 for private repositories the current token cannot see, so check both the URL and the token's access.

**PAT URL warning:** a token embedded in `--repo` (for example `https://user:token@github.com/org/repo.git`) is written to `~/.my_config_cli/settings.json` and used as the git remote URL. `myconfig` masks it in its own output (`https://***@github.com/...`), but the token remains on disk. Prefer SSH or a credential helper.

## Configuration & Mappings

On initial execution, `myconfig` generates `~/.my_config_cli/mapping.json` using embedded defaults. You can customize the source and target paths as needed. Symlink entries can set `source_kind` to `file` (default) or `dir`; when the source is missing, `link` creates a placeholder of that kind. Existing mappings without the field default to `file` unless the source already exists as a directory.

### Default Mapping Layout

- **Gemini**:
  - `~/.my_config_cli/config/.gemini/antigravity-cli/settings.json` -> `~/.gemini/antigravity-cli/settings.json`
  - `~/.my_config_cli/config/.gemini/config/config.json` -> `~/.gemini/config/config.json`
  - `~/.my_config_cli/config/AGENTS.md` -> `~/.gemini/GEMINI.md`
  - `~/.my_config_cli/config/mcp_config.json` -> `~/.gemini/config/mcp_config.json`
  - `~/.my_config_cli/config/mcp_config.json` -> `~/.gemini/antigravity/mcp_config.json`
  - `~/.my_config_cli/config/skills` -> `~/.gemini/config/skills`
- **Pi Agent**:
  - `~/.my_config_cli/config/.pi/agent/settings.json` -> `~/.pi/agent/settings.json`
  - `~/.my_config_cli/config/AGENTS.md` -> `~/.pi/agent/SYSTEM.md`
  - `~/.my_config_cli/config/mcp_config.json` -> `~/.pi/agent/mcp.json`
  - `~/.my_config_cli/config/skills` -> `~/.pi/agent/skills`
- **OpenCode**:
  - `~/.my_config_cli/config/AGENTS.md` -> `~/.config/opencode/AGENTS.md`
  - `~/.my_config_cli/config/skills` -> `~/.config/opencode/skills`
  - `~/.my_config_cli/config/mcp_config.json` -> merged into `~/.config/opencode/opencode.json` (`mcpServers`)
- **Claude**:
  - `~/.my_config_cli/config/.claude/config.json` -> `~/.claude/config.json`
  - `~/.my_config_cli/config/AGENTS.md` -> `~/.claude/CLAUDE.md`
  - `~/.my_config_cli/config/mcp_config.json` -> `~/.claude/mcp.json`
  - `~/.my_config_cli/config/skills` -> `~/.claude/skills`
- **Shell & Git**:
  - `~/.my_config_cli/config/.bashrc` -> `~/.bashrc`
  - `~/.my_config_cli/config/.gitconfig` -> `~/.gitconfig`

## Development & Testing

```bash
./lint.sh        # Runs go vet ./...
./test.sh        # Runs all unit tests
./docker-test.sh # Runs integration tests inside Ubuntu 24.04 container
./build.sh       # Compiles ./myconfig
```
