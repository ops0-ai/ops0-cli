#!/usr/bin/env bash
#
# ops0 CLI installer
# Usage: curl -fsSL https://raw.githubusercontent.com/ops0-ai/ops0-cli/main/install.sh | sh
#
# Detects OS/arch, downloads the appropriate release tarball from GitHub,
# and installs the `ops0` binary into /usr/local/bin (or $OPS0_INSTALL_DIR).
#
# We deliberately keep this dependency-free: no curl-piping into bash inside
# the script, no python, no homebrew assumptions. Just `curl`, `tar`, and
# `install`.

set -eu

REPO="ops0-ai/ops0-cli"
INSTALL_DIR="${OPS0_INSTALL_DIR:-/usr/local/bin}"

# Resolve OS / arch in goreleaser's naming convention.
detect_os() {
  case "$(uname -s)" in
    Linux)   echo "Linux" ;;
    Darwin)  echo "Darwin" ;;
    *)       echo "unsupported"; exit 1 ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "x86_64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)             echo "unsupported"; exit 1 ;;
  esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"

# Allow override for testing or pinning.
VERSION="${OPS0_VERSION:-}"
if [ -z "$VERSION" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name":' | head -n1 | cut -d'"' -f4)
fi
if [ -z "$VERSION" ]; then
  echo "Could not determine latest version. Set OPS0_VERSION to pin." >&2
  exit 1
fi

ARCHIVE="ops0_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

echo "Downloading ops0 ${VERSION} for ${OS}/${ARCH}..."
curl -fsSL -o "$TMP/ops0.tar.gz" "$URL"
tar -xzf "$TMP/ops0.tar.gz" -C "$TMP"

# Install with sudo if the dir isn't writable.
if [ -w "$INSTALL_DIR" ]; then
  install -m 0755 "$TMP/ops0" "$INSTALL_DIR/ops0"
else
  echo "Installing to $INSTALL_DIR (sudo required)..."
  sudo install -m 0755 "$TMP/ops0" "$INSTALL_DIR/ops0"
fi

echo "✓ ops0 installed to $INSTALL_DIR/ops0"
"$INSTALL_DIR/ops0" version
echo
echo "Next: run \`ops0 login\` and paste an API key from https://brew.ops0.ai/settings"
