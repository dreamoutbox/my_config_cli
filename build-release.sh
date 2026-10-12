#!/bin/sh
set -eu

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
cd "$SCRIPT_DIR"

VERSION=$(cat VERSION 2>/dev/null || echo "dev")
VERSION="${VERSION#v}"

DIST_DIR="dist"
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

echo "Building release binaries for v${VERSION}..."

TARGETS="
linux/amd64/myconfig-linux-amd64
linux/arm64/myconfig-linux-arm64
darwin/amd64/myconfig-darwin-amd64
darwin/arm64/myconfig-darwin-arm64
windows/amd64/myconfig-windows-amd64.exe
"

for target in $TARGETS; do
  GOOS=$(echo "$target" | cut -d/ -f1)
  GOARCH=$(echo "$target" | cut -d/ -f2)
  OUTNAME=$(echo "$target" | cut -d/ -f3)

  echo "Compiling ${GOOS}/${GOARCH} -> ${DIST_DIR}/${OUTNAME}..."
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build \
    -trimpath \
    -ldflags "-s -w -X github.com/dreamoutbox/my_config_cli/cmd.Version=${VERSION}" \
    -o "${DIST_DIR}/${OUTNAME}" .

  # Package into archive
  ARCHIVE_BASE="myconfig-v${VERSION}-${GOOS}-${GOARCH}"
  TMP_ARCHIVE_DIR=$(mktemp -d)
  if [ "$GOOS" = "windows" ]; then
    cp "${DIST_DIR}/${OUTNAME}" "${TMP_ARCHIVE_DIR}/myconfig.exe"
    if command -v zip >/dev/null 2>&1; then
      (cd "$TMP_ARCHIVE_DIR" && zip "${SCRIPT_DIR}/${DIST_DIR}/${ARCHIVE_BASE}.zip" myconfig.exe >/dev/null)
    fi
  else
    cp "${DIST_DIR}/${OUTNAME}" "${TMP_ARCHIVE_DIR}/myconfig"
    tar -czf "${DIST_DIR}/${ARCHIVE_BASE}.tar.gz" -C "$TMP_ARCHIVE_DIR" myconfig
  fi
  rm -rf "$TMP_ARCHIVE_DIR"
done

# Generate SHA-256 checksums
echo "Generating SHA-256 checksums..."
cd "$DIST_DIR"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum * > checksums.txt
elif command -v shasum >/dev/null 2>&1; then
  shasum -a 256 * > checksums.txt
fi
cd "$SCRIPT_DIR"

echo "Release artifacts created in ${DIST_DIR}/:"
ls -lh "$DIST_DIR"
