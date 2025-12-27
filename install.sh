#!/usr/bin/env bash
# Installs witr from GitHub
# Linux: downloads prebuilt binary
# macOS: builds from source (requires Go 1.21+)
# Repo: https://github.com/pranshuparmar/witr

set -euo pipefail

REPO="pranshuparmar/witr"
INSTALL_PATH="/usr/local/bin/witr"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
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

install_linux() {
    # Get latest release tag from GitHub API
    LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d '"' -f4)
    if [[ -z "$LATEST" ]]; then
        echo "Could not determine latest release tag." >&2
        exit 1
    fi

    URL="https://github.com/$REPO/releases/download/$LATEST/witr-linux-$ARCH"
    TMP=$(mktemp)
    MANURL="https://github.com/$REPO/releases/download/$LATEST/witr.1"
    MAN_TMP=$(mktemp)

    echo "Downloading witr $LATEST for linux-$ARCH..."
    curl -fL "$URL" -o "$TMP"
    curl -fL "$MANURL" -o "$MAN_TMP"

    # Install
    sudo install -m 755 "$TMP" "$INSTALL_PATH"
    rm -f "$TMP"
    sudo install -D -m 644 "$MAN_TMP" /usr/local/share/man/man1/witr.1
    rm -f "$MAN_TMP"

    echo "witr installed successfully to $INSTALL_PATH (version: $LATEST, arch: $ARCH)"
    echo "Man page installed to /usr/local/share/man/man1/witr.1"
}

install_macos() {
    # Check for Go
    if ! command -v go &> /dev/null; then
        echo "Error: Go is required to build witr on macOS." >&2
        echo "Install Go from https://go.dev/dl/ or via: brew install go" >&2
        exit 1
    fi

    GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
    GO_MAJOR=$(echo "$GO_VERSION" | cut -d. -f1)
    GO_MINOR=$(echo "$GO_VERSION" | cut -d. -f2)
    if [[ "$GO_MAJOR" -lt 1 ]] || [[ "$GO_MAJOR" -eq 1 && "$GO_MINOR" -lt 21 ]]; then
        echo "Error: Go 1.21+ required, found $GO_VERSION" >&2
        exit 1
    fi

    # Check if running from within the witr repo
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    if [[ -f "$SCRIPT_DIR/cmd/witr/main.go" ]]; then
        echo "Building from local repository..."
        cd "$SCRIPT_DIR"
    else
        # Clone and build
        TMP_DIR=$(mktemp -d)
        trap "rm -rf $TMP_DIR" EXIT

        echo "Cloning witr repository..."
        git clone --depth 1 "https://github.com/$REPO.git" "$TMP_DIR/witr"
        cd "$TMP_DIR/witr"
    fi

    # Get version info
    VERSION=$(git describe --tags --always 2>/dev/null || echo "dev")
    COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    BUILD_DATE=$(date +%Y-%m-%d)

    echo "Building witr $VERSION for darwin-$ARCH..."
    go build -ldflags "-X main.version=$VERSION -X main.commit=$COMMIT -X main.buildDate=$BUILD_DATE" -o witr ./cmd/witr

    # Install
    sudo install -m 755 witr "$INSTALL_PATH"

    # Install man page if exists
    if [[ -f witr.1 ]]; then
        sudo mkdir -p /usr/local/share/man/man1
        sudo install -m 644 witr.1 /usr/local/share/man/man1/witr.1
        echo "Man page installed to /usr/local/share/man/man1/witr.1"
    fi

    echo "witr installed successfully to $INSTALL_PATH (version: $VERSION, arch: $ARCH)"
}

# Main
case "$OS" in
    linux)
        install_linux
        ;;
    darwin)
        install_macos
        ;;
    *)
        echo "Unsupported OS: $OS" >&2
        echo "Supported: linux, darwin (macOS)" >&2
        exit 1
        ;;
esac
