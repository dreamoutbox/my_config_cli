#!/bin/sh
set -eu

IMAGE_NAME="myconfig-docker-test"

echo "==> Building Docker test image ($IMAGE_NAME)..."
docker build -f docker/Dockerfile -t "$IMAGE_NAME" .

echo "==> Running Docker tests..."
docker run --rm "$IMAGE_NAME" "$@"

