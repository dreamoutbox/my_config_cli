#!/bin/sh
set -eu

DIR="$(cd "$(dirname "$0")" && pwd)"
. "$DIR/lib.sh"

log "Running push_adopts_gitconfig_bashrc test on Ubuntu 24.04..."

# 1. Prepare environment:
# Configure git global user (creates ~/.gitconfig on host)
git config --global user.name "Docker Tester"
git config --global user.email "docker-tester@example.com"

# Ensure ~/.bashrc has unique test marker
MARKER="# test-marker-docker-test"
echo "$MARKER" >> "$HOME/.bashrc"

# Verify initial state: real files, not symlinks
assert_regular_file "$HOME/.gitconfig"
assert_regular_file "$HOME/.bashrc"

# 2. Setup a local bare git remote
REMOTE_DIR="/tmp/test-remote.git"
rm -rf "$REMOTE_DIR"
git init --bare -b main "$REMOTE_DIR"

# 3. Fresh start: run myconfig push without prior import
log "Executing myconfig push..."
myconfig push --repo "$REMOTE_DIR"

# 4. Assertions on store directory (~/.my_config_cli/config)
STORE_DIR="$HOME/.my_config_cli/config"
log "Verifying store directory..."
assert_file_exists "$STORE_DIR/.gitconfig"
assert_file_exists "$STORE_DIR/.bashrc"
assert_contains "$STORE_DIR/.gitconfig" "Docker Tester"
assert_contains "$STORE_DIR/.bashrc" "$MARKER"

# 5. Assert host files remain regular files (NOT converted to symlinks by push)
log "Verifying host target files remain regular files..."
assert_regular_file "$HOME/.gitconfig"
assert_regular_file "$HOME/.bashrc"

# 6. Verify remote repo received the commit containing both files
CLONE_DIR="/tmp/test-clone"
rm -rf "$CLONE_DIR"
git clone "$REMOTE_DIR" "$CLONE_DIR"
assert_file_exists "$CLONE_DIR/.gitconfig"
assert_file_exists "$CLONE_DIR/.bashrc"
assert_contains "$CLONE_DIR/.gitconfig" "docker-tester@example.com"
assert_contains "$CLONE_DIR/.bashrc" "$MARKER"

log "push_adopts_gitconfig_bashrc test passed successfully!"

