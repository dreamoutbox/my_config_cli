#!/bin/sh
set -eu

# log prints a formatted info message.
log() {
    echo "==> $*"
}

# fail prints an error message and exits with code 1.
fail() {
    echo "ERROR: $*" >&2
    exit 1
}

# assert_file_exists checks that path exists.
assert_file_exists() {
    if [ ! -e "$1" ]; then
        fail "expected $1 to exist"
    fi
}

# assert_regular_file checks that path exists and is not a symlink.
assert_regular_file() {
    if [ ! -f "$1" ]; then
        fail "expected $1 to be a file"
    fi
    if [ -L "$1" ]; then
        fail "expected $1 to be a regular file, but found a symlink"
    fi
}

# assert_contains checks that file contains pattern.
assert_contains() {
    if ! grep -q "$2" "$1"; then
        fail "expected $1 to contain '$2'"
    fi
}

