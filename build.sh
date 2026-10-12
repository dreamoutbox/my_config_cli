#!/bin/sh
set -eu

VERSION=$(cat VERSION 2>/dev/null || echo "0.1.0")
go build -ldflags "-X github.com/dreamoutbox/my_config_cli/cmd.Version=${VERSION}" -o myconfig .
