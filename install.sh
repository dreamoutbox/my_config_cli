#!/bin/sh
set -eu

BIN_DIR="${HOME}/.local/bin"
TARGET="${BIN_DIR}/myconfig"
REPO="dreamoutbox/my_config_cli"

mkdir -p "$BIN_DIR"

# If executing inside the source repository with Go available, build from local source
if [ -f "go.mod" ] && grep -q "my_config_cli" go.mod 2>/dev/null && command -v go >/dev/null 2>&1; then
  echo "Building and installing myconfig from local source into ${BIN_DIR}..."
  VERSION=$(cat VERSION 2>/dev/null || echo "0.1.0")
  go build -ldflags "-X github.com/dreamoutbox/my_config_cli/cmd.Version=${VERSION}" -o "$TARGET" .
else
  # Running via curl / standalone installer: download prebuilt binary
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m)

  case "$ARCH" in
    x86_64|amd64)
      ARCH="amd64"
      ;;
    aarch64|arm64)
      ARCH="arm64"
      ;;
    *)
      echo "Error: Unsupported CPU architecture '$ARCH'." >&2
      exit 1
      ;;
  esac

  case "$OS" in
    linux|darwin)
      ;;
    *)
      echo "Error: Unsupported operating system '$OS'." >&2
      exit 1
      ;;
  esac

  BINARY_NAME="myconfig-${OS}-${ARCH}"
  DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_NAME}"

  echo "Downloading ${BINARY_NAME} from ${DOWNLOAD_URL}..."
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$DOWNLOAD_URL" -o "$TARGET"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$TARGET" "$DOWNLOAD_URL"
  else
    echo "Error: curl or wget is required to download prebuilt binary." >&2
    exit 1
  fi
fi

chmod +x "$TARGET"
echo "myconfig installed successfully to ${TARGET}."

# Check if ~/.local/bin is present in PATH
case ":${PATH}:" in
  *":${BIN_DIR}:"*)
    ;;
  *)
    echo ""
    echo "Notice: ${BIN_DIR} is not in your current PATH."
    echo "Add the following line to your shell configuration (e.g. ~/.bashrc, ~/.zshrc):"
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    ;;
esac
