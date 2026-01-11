#!/usr/bin/env bash
# Installs the latest release of witr from GitHub
# Repo: https://github.com/pranshuparmar/witr

set -euo pipefail

REPO="pranshuparmar/witr"

# Standard configurable install prefix (override to avoid sudo):
#   INSTALL_PREFIX="$HOME/.local" ./install.sh
INSTALL_PREFIX="${INSTALL_PREFIX:=/usr/local}"

INSTALL_PATH="$INSTALL_PREFIX/bin/witr"
MAN_PATH="$INSTALL_PREFIX/share/man/man1/witr.1"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux)
        OS=linux
        ;;
    darwin)
        OS=darwin
        ;;
    freebsd)
        OS=freebsd
        ;;
    *)
        echo "Unsupported OS: $OS" >&2
        exit 1
        ;;
esac


# Detect Architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)
        ARCH=amd64
        ;;
    aarch64|arm64)
        ARCH=arm64
        ;;
    *)
        echo "Unsupported architecture: $ARCH" >&2
        exit 1
        ;;
esac

# Ensure required tools exist
for cmd in curl install; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Missing required command: $cmd"
    exit 1
  fi
done

# Get latest release tag from GitHub API
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d '"' -f4)
if [[ -z "$LATEST" ]]; then
    echo "Could not determine latest release tag." >&2
    exit 1
fi

# Construct download URL
URL="https://github.com/$REPO/releases/download/$LATEST/witr-$OS-$ARCH"
TMP=$(mktemp)
MANURL="https://github.com/$REPO/releases/download/$LATEST/witr.1"
MAN_TMP=$(mktemp)

# Cleanup on exit
cleanup() {
    rm -f "${TMP:-}" "${MAN_TMP:-}"
}
trap cleanup EXIT

# Download release
curl -fL "$URL" -o "$TMP"
curl -fL "$MANURL" -o "$MAN_TMP"

INSTALL_BIN_DIR=$(dirname "$INSTALL_PATH")
INSTALL_MAN_DIR=$(dirname "$MAN_PATH")

# Decide whether we need sudo (based on whether we can write to the target dirs)
need_sudo=0
if ! mkdir -p "$INSTALL_BIN_DIR" 2>/dev/null; then need_sudo=1; fi
if ! mkdir -p "$INSTALL_MAN_DIR" 2>/dev/null; then need_sudo=1; fi
if [[ "$need_sudo" == "0" ]]; then
    [[ -w "$INSTALL_BIN_DIR" ]] || need_sudo=1
    [[ -w "$INSTALL_MAN_DIR" ]] || need_sudo=1
fi

SUDO=()
if [[ "$need_sudo" == "1" ]]; then
    # checking for sudo because alpine using doas and people like me started to use run0
    if command -v sudo >/dev/null 2>&1; then
        # echo "sudo is available"
        SUDO=(sudo)
    elif command -v doas >/dev/null 2>&1; then
        # echo "doas is available"
        SUDO=(doas)
    elif command -v run0 >/dev/null 2>&1; then
        # echo "run0 is available"
        SUDO=(run0)
    fi

fi

# Install
${SUDO[@]+"${SUDO[@]}"} install -m 755 "$TMP" "$INSTALL_PATH"

# Install man page
${SUDO[@]+"${SUDO[@]}"} mkdir -p "$INSTALL_MAN_DIR"
${SUDO[@]+"${SUDO[@]}"} install -m 644 "$MAN_TMP" "$MAN_PATH"
echo "witr installed successfully to $INSTALL_PATH (version: $LATEST, os: $OS, arch: $ARCH)"
echo "Man page installed to $MAN_PATH"
