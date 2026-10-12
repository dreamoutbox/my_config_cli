help me write golang cli to backup and restore my ai config store to github.

use spf13/cobra.

I want to have central config directory. store at `~/.my_config_cli/config`.
and symlink to various ai tools.

## gemini cli

~/.my_config_cli/config/.gemini/antigravity-cli/settings.json -> ~/.gemini/antigravity-cli/settings.json
~/.my_config_cli/config/.gemini/config/config.json -> ~/.gemini/config/config.json

~/.my_config_cli/config/AGENTS.md -> .gemini/GEMINI.md
~/.my_config_cli/config/mcp_config.json -> ~/.gemini/config/mcp_config.json
~/.my_config_cli/config/mcp_config.json -> ~/.gemini/antigravity/mcp_config.json
~/.my_config_cli/config/skills/ (dir) -> ~/.gemini/config/skills

## pi coding agent

~/.my_config_cli/config/.pi/agent/settings.json -> ~/.pi/agent/settings.json

~/.my_config_cli/config/AGENTS.md -> ~/.pi/agent/SYSTEM.md
~/.my_config_cli/config/mcp_config.json -> ~/.pi/agent/mcp.json
~/.my_config_cli/config/skills/ -> ~/.pi/agent/skills/

## opencode

~/.my_config_cli/config/.config/opencode/opencode.json -> ~/.config/opencode/opencode.json

~/.my_config_cli/config/AGENTS.md -> ~/.config/opencode/AGENTS.md
~/.my_config_cli/config/skills/ -> ~/.config/opencode/skills/

* for opencode. it store mcp config inside the `~/.config/opencode/opencode.json` make this cli replace mcp config value in config file.

## claude code

~/.my_config_cli/config/.claude/config.json -> ~/.claude/config.json

~/.my_config_cli/config/AGENTS.md -> ~/.claude/CLAUDE.md
~/.my_config_cli/config/mcp_config.json -> ~/.claude/mcp.json
~/.my_config_cli/config/skills/ -> ~/.claude/skills/

## NOTES:

make ~/.my_config_cli/mapping.json to these store the symlink relationships. create default file if not exists (embedded the default mapping.json of this repository).

when upload to git with command `myconfig push --repo <repo_url>` make golang use git cli to upload all file under `~/.my_config_cli/config` to git repo.

if user did not specify `--repo <repo_url>`. fallback and read from `~/.my_config_cli/settings.json` `repo_url` field. if empty then error and tell user need to specify `--repo <repo_url>`. remember `--repo` by store in config. both push and restore command.

alias ``myconfig save` `myconfig backup` to `myconfig push`

alias ``myconfig load` `myconfig restore` to `myconfig pull`
