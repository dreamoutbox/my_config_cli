# myconfig

CLI tool to backup, restore, and synchronize AI tool configurations across environments via Git and symlinks.

## Features

- **Centralized storage**: Stores all shared configurations in `~/.my_config_cli/config`.
- **Git synchronization**: Push and pull configuration repositories using system `git`.
- **Automatic symlinking**: Automatically symlinks configs and skills to supported AI tools (Gemini CLI, Pi agent, OpenCode, Claude).
- **JSON configuration merge**: Merges MCP server definitions into composite configs like `opencode.json` without overwriting unrelated settings.
- **Health monitoring**: Inspect repository status and symlink health at any time.

## Installation & Build

Requires Go 1.24+.

```bash
make build
# or install directly into $GOPATH/bin
make install
```

## Usage

### 1. Push / Backup

Upload configurations from `~/.my_config_cli/config` to your git repository:

```bash
myconfig push --repo git@github.com:username/ai-configs.git
```

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
myconfig pull --repo git@github.com:username/ai-configs.git
# Or using the saved repository URL:
myconfig pull
# Aliases:
myconfig load
myconfig restore
```

### 3. Link (Manual)

Apply all symlinks and JSON merges without pulling from git:

```bash
myconfig link
```

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

## Configuration & Mappings

On initial execution, `myconfig` generates `~/.my_config_cli/mapping.json` using embedded defaults. You can customize the source and target paths as needed.

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
  - `~/.my_config_cli/config/.config/opencode/opencode.json` -> `~/.config/opencode/opencode.json`
  - `~/.my_config_cli/config/AGENTS.md` -> `~/.config/opencode/AGENTS.md`
  - `~/.my_config_cli/config/skills` -> `~/.config/opencode/skills`
  - `~/.my_config_cli/config/mcp_config.json` -> merged into `~/.config/opencode/opencode.json` (`mcpServers`)
- **Claude**:
  - `~/.my_config_cli/config/.claude/config.json` -> `~/.claude/config.json`
  - `~/.my_config_cli/config/AGENTS.md` -> `~/.claude/CLAUDE.md`
  - `~/.my_config_cli/config/mcp_config.json` -> `~/.claude/mcp.json`
  - `~/.my_config_cli/config/skills` -> `~/.claude/skills`
