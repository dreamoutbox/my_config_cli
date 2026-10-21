# Implementation Plan: Private-Repo Auth Fallback and Missing-Tool Linking

Fix two problems found when running `myconfig pull` on a fresh machine:
1. Pull from a private repo fails with 403 because an env-provided token lacks access.
2. `link` fails (`FAIL [~/.config/opencode/opencode.json]: ... no such file or directory`) when a tool is not installed or a source file is absent from the repo.

---

## Root Cause Analysis

### Problem 1 (403)
- [`GitRun`](internal/git/git.go) runs git with the inherited environment and with stdout/stderr captured and **stdin detached**, so git can never prompt for credentials.
- An env token (e.g. Codespaces `GITHUB_TOKEN` via a credential helper) is used automatically. It is valid but scoped to another repo, so GitHub returns 403 instead of asking for new credentials.

### Problem 2 (opencode.json) - both suspicions checked against code
- **Confirmed suspicion 1 (writing through a symlink).** `assets/mapping.json` contains two entries with the same target `~/.config/opencode/opencode.json`:
  1. `symlink` -> source `~/.my_config_cli/config/.config/opencode/opencode.json`
  2. `json_merge` (mcpServers) from `mcp_config.json`
  Entry 1 creates a symlink. The source file does not exist in the remote repo, so the link is dangling. Entry 2 then runs `applyJSONMerge`: `os.ReadFile(dst)` fails silently (dangling), then `os.WriteFile(dst)` follows the dangling symlink and fails because the source parent dir `~/.my_config_cli/config/.config/opencode/` does not exist. The error message shows `dst` because that is the name passed to `open`.
- **Partly confirmed suspicion 2 (nested dir not created).** `applySymlink` creates the parent of the *target* only. Nothing creates the source file or the source's parent dir inside the store.
- The symlink entry for `opencode.json` also violates the AGENTS.md rule "never symlink opencode.json".
- Side effects seen in `status`: `STALE` for every symlink whose source is missing (settings.json for pi/claude/opencode); `MISSING` for the json_merge entry.
- Note: `~/.my_config_cli/mapping.json` is seeded once from the asset, so existing users keep the bad entry; a migration is needed.

---

### Phase 1: Auth Fallback for Git Network Operations [STATUS: DONE]
- Goal: On access-denied errors, retry once without ambient credentials and let the user type login/password (as if git was never configured).
- Tasks:
  - [x] In `internal/git`, add `error.go` with `IsAuthError(err error) bool` matching stderr substrings: `unable to access`, `returned error: 403`, `Authentication failed`, `could not read Username`, `Write access to repository not granted`. Use a typed enum/constants for the patterns.
  - [x] Add `GitRunInteractive(dir string, extraEnv []string, args ...string)` (shared by all callers, no duplicated exec code) that wires `os.Stdin/Stdout/Stderr` so git can prompt for username/password.
  - [x] Define the "clean credential" env/config used on retry: `GIT_TERMINAL_PROMPT=1`, `GIT_ASKPASS=` and `SSH_ASKPASS=` cleared, `GH_TOKEN`/`GITHUB_TOKEN` removed, and `-c credential.helper=` (empty, resets the helper list) so no stored/env token is used. Credentials are never persisted by myconfig.
  - [x] Add `RunWithAuthRetry(dir string, args ...string)`: run normally; if `IsAuthError`, print a short notice ("Access denied with default credentials; enter credentials for <host>"), then rerun interactively with the clean setup. Only the network commands use it: `pull`, `fetch`, `push`, `ls-remote`.
  - [x] Switch `Pull`, `Fetch`, `PushWithOptions`, `RemoteHasHeads` to the retry helper. Fix the double-wrapped message ("pulling from remote: pulling from remote:") while there.
  - [x] 404 / `Repository not found` is NOT an auth error: no retry; return a clear error "repository does not exist (or URL is wrong): <masked url>". Add `IsNotFoundError(err)` and a test.
  - [x] Skip retry when stdin is not a TTY (CI): return the original error with a hint to use a PAT in `--repo`.
  - [x] Add shared helper `MaskURL(raw string) string` (in `internal/git`) replacing embedded `user:token@` with `***@`. Use it in every printed URL (`Pulling/Fetching configuration from ...` in `cmd/pull.go`, `push`, `status`) and in wrapped git errors (git stderr may echo the URL). Tests: PAT URL, plain URL, ssh URL unchanged.
  - [x] Unit tests: `IsAuthError` table test; retry path using a fake `git` on `PATH` (script in `testdata/`) that fails with 403 on first call and succeeds when `GITHUB_TOKEN` is absent and credential helper is reset; non-auth errors do not retry.
- Done when: `./test.sh` and `./lint.sh` pass; manual run with a wrong `GITHUB_TOKEN` prompts for username/password and succeeds.

---

### Phase 2: Safe JSON Merge (never write through symlinks) [STATUS: DONE]
- Goal: `json_merge` works whether or not the tool is installed and never follows a dangling symlink.
- Tasks:
  - [x] Write a failing test in `internal/symlink/link_test.go` reproducing the bug: dangling symlink at target, then `applyJSONMerge` -> expect success.
  - [x] In `applyJSONMerge`, `os.Lstat(dst)` first; if `dst` is a symlink, remove it before reading/writing so a real file is written.
  - [x] Write atomically (temp file in same dir + rename) so a failed write cannot leave a half file.
  - [x] Make a missing/empty source (`mcp_config.json` absent) a clear error, not an opaque `open` failure.
  - [x] Preserve existing behavior: other keys of an existing real `opencode.json` untouched.
- Done when: new test passes and merge into a not-yet-existing `~/.config/opencode/opencode.json` creates the file with only `mcpServers`.

---

### Phase 3: Fix Mapping Conflict and Migrate Existing Users [STATUS: DONE]
- Goal: Remove the invalid `symlink` entry for `opencode.json` and repair stored mappings.
- Tasks:
  - [x] Delete the `symlink` entry for `~/.config/opencode/opencode.json` from `assets/mapping.json` (json_merge entry remains, as AGENTS.md requires).
  - [x] In `config.LoadMapping`, add validation: reject/skip a `symlink` entry whose target equals a `json_merge` target; emit a warning.
  - [x] Add a one-time migration in `internal/config` that drops such conflicting entries from an existing `~/.my_config_cli/mapping.json` (no overwrite of other user edits). Unit test with a temp `HOME`.
  - [x] `import` for opencode: confirm it does not rely on the removed symlink entry (check `internal/importer`).
- Done when: `myconfig pull` on a clean home produces no `FAIL`, and `status` no longer lists the opencode symlink entry.

---

### Phase 4: Always Materialize Sources and Target Dirs (tool not installed) [STATUS: DONE]
- Goal: Link everything even when the tool is not installed, and keep the store complete so `push` reflects reality.
- Tasks:
  - [x] In `applySymlink`, if `src` does not exist: create its parent dir (`0755`) and an empty placeholder (file) or directory, chosen by a new explicit `source_kind` field on `config.Entry` (typed enum `SourceKind`: `file`/`dir`). Set it for every entry in `assets/mapping.json` (`skills` -> `dir`, others -> `file`). Entries without the field (old user mappings) default to `file` unless the source is an existing dir.
  - [x] Seed placeholder JSON files with `{}` so tools do not choke on empty JSON; markdown/other files empty.
  - [x] Keep target-parent `MkdirAll` (already present) so `~/.config/opencode/`, `~/.pi/agent/` etc. are created without the tool installed.
  - [x] Add a distinct `ActionCreated` (enum) reported by `link` when a placeholder was created, so the user knows the store changed and `myconfig push` is needed.
  - [x] Make failures non-fatal per entry already works (`ApplyAll`); keep, but print a final summary line.
  - [x] Unit tests: missing source file, missing source dir, nested missing parents, idempotent second run, `CheckHealth` returns `OK` after link.
- Done when: running `link` in an empty `HOME` with an empty store yields `OK` for every entry in `status`, with the placeholders visible in `git status` of the store.

---

### Phase 5: Docs and Verification [STATUS: DONE]
- Goal: Document behavior and verify end to end.
- Tasks:
  - [x] Update `AGENTS.md` (auth fallback, placeholder creation, mapping migration, no symlink over `opencode.json`) and `README.md` (private repo usage, PAT URL warning).
  - [x] Reproduce both original scenarios in a scratch `HOME`: wrong env token -> credential prompt; fresh machine pull with no tools installed -> all entries `OK`.
  - [x] Run `./lint.sh`, `./test.sh`, `./build.sh`.
- Done when: all three scripts exit 0 and docs match behavior.

---

## Unknowns, risks, assumptions

- **Decision:** 403 / `unable to access` triggers the credential prompt. 404 errors out as "repository does not exist" with no retry (note: GitHub returns 404 for private repos the token cannot see; the error message should mention this).
- **Assumption:** Credentials typed by the user go to git's own prompt and are not stored by myconfig; a user's helper then may store them (we reset helpers for that call only, so nothing is saved).
- **Risk:** Resetting `credential.helper` also disables a legitimate stored login for the retry; acceptable since the retry only happens after default credentials were rejected.
- **Risk:** Placeholder files get committed on next `push` (empty `{}` files); this is the intended behavior per the request but may surprise users - mitigated by `ActionCreated` output.
- **Risk:** Placeholder `{}` for tools' settings could overwrite nothing (target is replaced by symlink; previous real target content is removed by `RemoveAll`, existing behavior). Consider backing up a pre-existing real target; out of scope unless requested.
- **Decision:** `source_kind` is an explicit mapping field (Phase 4).
