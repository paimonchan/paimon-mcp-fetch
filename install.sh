#!/usr/bin/env bash
# Install paimon-mcp-fetch — Web content fetching MCP server.
#
# Run with:
#   curl -fsSL https://raw.githubusercontent.com/user/paimon-mcp-fetch/main/install.sh | sh
#
# Or with sudo for system-wide install:
#   curl -fsSL https://raw.githubusercontent.com/user/paimon-mcp-fetch/main/install.sh | sudo sh

set -e

REPO="user/paimon-mcp-fetch"
BINARY="paimon-mcp-fetch"

# Colors
info() { printf "\033[36m[install]\033[0m %s\n" "$*"; }
ok()   { printf "\033[32m[install]\033[0m %s\n" "$*"; }
warn() { printf "\033[33m[install]\033[0m %s\n" "$*"; }
error() { printf "\033[31m[install]\033[0m %s\n" "$*" >&2; exit 1; }

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
    linux)   GOOS="linux" ;;
    darwin)  GOOS="darwin" ;;
    *)       error "Unsupported OS: $OS" ;;
esac

case "$ARCH" in
    x86_64|amd64) GOARCH="amd64" ;;
    aarch64|arm64) GOARCH="arm64" ;;
    *)            error "Unsupported architecture: $ARCH" ;;
esac

SUFFIX="${GOOS}-${GOARCH}"
ASSET="${BINARY}-${SUFFIX}"

# Determine install directory
if [ -n "$INSTALL_DIR" ]; then
    INSTALL_DIR="$INSTALL_DIR"
elif [ -w /usr/local/bin ]; then
    INSTALL_DIR="/usr/local/bin"
elif [ -d "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
else
    INSTALL_DIR="$HOME/bin"
    mkdir -p "$INSTALL_DIR"
fi

INSTALL_PATH="${INSTALL_DIR}/${BINARY}"

info "Detected platform: $SUFFIX"
info "Install directory: $INSTALL_DIR"

# Check for required tools
if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
    error "curl or wget is required"
fi

# Get latest release
API_URL="https://api.github.com/repos/${REPO}/releases/latest"
info "Fetching latest release info..."

if command -v curl >/dev/null 2>&1; then
    RELEASE=$(curl -fsSL --max-time 30 "$API_URL")
else
    RELEASE=$(wget -qO- --timeout=30 "$API_URL")
fi

VERSION=$(echo "$RELEASE" | grep -o '"tag_name": *"[^"]*"' | sed 's/.*"\([^"]*\)".*/\1/')
info "Latest version: $VERSION"

# Find asset download URL
ASSET_URL=$(echo "$RELEASE" | grep -o '"browser_download_url": *"[^"]*'"${ASSET}"'"' | sed 's/.*"\([^"]*\)".*/\1/')
if [ -z "$ASSET_URL" ]; then
    error "Could not find asset '${ASSET}' in release ${VERSION}"
fi

# Download
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

info "Downloading ${ASSET}..."
if command -v curl >/dev/null 2>&1; then
    curl -fsSL --max-time 120 "$ASSET_URL" -o "$TMP_DIR/${BINARY}"
else
    wget --timeout=120 -O "$TMP_DIR/${BINARY}" "$ASSET_URL"
fi

# Verify checksum (optional)
CHECKSUM_URL=$(echo "$RELEASE" | grep -o '"browser_download_url": *"[^"]*'"${ASSET}"'.sha256"' | sed 's/.*"\([^"]*\)".*/\1/')
if [ -n "$CHECKSUM_URL" ]; then
    info "Verifying checksum..."
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL --max-time 30 "$CHECKSUM_URL" -o "$TMP_DIR/checksum.sha256"
    else
        wget --timeout=30 -O "$TMP_DIR/checksum.sha256" "$CHECKSUM_URL"
    fi
    EXPECTED=$(awk '{print $1}' "$TMP_DIR/checksum.sha256")
    ACTUAL=$(sha256sum "$TMP_DIR/${BINARY}" | awk '{print $1}')
    if [ "$EXPECTED" != "$ACTUAL" ]; then
        error "Checksum mismatch! Expected: $EXPECTED, Got: $ACTUAL"
    fi
    ok "Checksum verified."
fi

# Install
info "Installing to $INSTALL_PATH..."
chmod +x "$TMP_DIR/${BINARY}"
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_DIR/${BINARY}" "$INSTALL_PATH"
else
    info "Elevating with sudo for system install..."
    sudo mv "$TMP_DIR/${BINARY}" "$INSTALL_PATH"
fi

# Add to PATH if needed
if ! command -v "$BINARY" >/dev/null 2>&1; then
    SHELL_PROFILE=""
    if [ -n "$BASH_VERSION" ]; then
        SHELL_PROFILE="$HOME/.bashrc"
    elif [ -n "$ZSH_VERSION" ]; then
        SHELL_PROFILE="$HOME/.zshrc"
    fi

    if [ -n "$SHELL_PROFILE" ] && [ -f "$SHELL_PROFILE" ]; then
        warn "$INSTALL_DIR is not in your PATH. Adding to $SHELL_PROFILE..."
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_PROFILE"
        warn "Please run: source $SHELL_PROFILE"
    else
        warn "$INSTALL_DIR is not in your PATH. Please add it manually."
    fi
fi

ok "${BINARY} ${VERSION} installed successfully!"
info "Binary location: $(command -v "$BINARY" 2>/dev/null || echo "$INSTALL_PATH")"
info "Add this to your MCP client config:"
cat <<'EOF'

  {
    "mcpServers": {
      "fetch": {
        "command": "paimon-mcp-fetch"
      }
    }
  }

EOF
