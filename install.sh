#!/bin/sh
set -e

# lazy-cli installer
# Detects OS and architecture, downloads the correct binary from GitHub Releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/bulkinglb/lazy-cli/main/install.sh | sh
#   wget -qO- https://raw.githubusercontent.com/bulkinglb/lazy-cli/main/install.sh | sh

REPO="bulkinglb/lazy-cli"
BINARY_NAME="lazy-cli"
INSTALL_DIR="/usr/local/bin"

# Colors (if terminal supports it)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

info()  { printf "${BOLD}%s${NC}\n" "$1"; }
ok()    { printf "${GREEN}✓${NC} %s\n" "$1"; }
warn()  { printf "${YELLOW}!${NC} %s\n" "$1"; }
fail()  { printf "${RED}✗ %s${NC}\n" "$1"; exit 1; }

# --- Detect OS ---
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "macos" ;;
        *)       fail "Unsupported OS: $(uname -s). Only Linux and macOS are supported." ;;
    esac
}

# --- Detect Architecture ---
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64" ;;
        aarch64|arm64)  echo "arm64" ;;
        *)              fail "Unsupported architecture: $(uname -m). Only amd64 and arm64 are supported." ;;
    esac
}

# --- Get latest release tag from GitHub ---
get_latest_version() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//'
    else
        fail "Neither curl nor wget found. Please install one of them."
    fi
}

# --- Download file ---
download() {
    url="$1"
    output="$2"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" -o "$output"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$output" "$url"
    fi
}

# --- Main ---
main() {
    info "lazy-cli installer"
    echo ""

    # Detect platform
    OS=$(detect_os)
    ARCH=$(detect_arch)
    ok "Detected: ${OS}/${ARCH}"

    # Get latest version
    info "Fetching latest release..."
    VERSION=$(get_latest_version)
    if [ -z "$VERSION" ]; then
        fail "Could not determine latest version. Check https://github.com/${REPO}/releases"
    fi
    ok "Latest version: ${VERSION}"

    # Build download URL
    ASSET_NAME="${BINARY_NAME}-${OS}-${ARCH}"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET_NAME}"

    # Download to temp file
    TMP_DIR=$(mktemp -d)
    TMP_FILE="${TMP_DIR}/${BINARY_NAME}"
    info "Downloading ${ASSET_NAME}..."
    download "$DOWNLOAD_URL" "$TMP_FILE" || fail "Download failed. Check if the release exists: https://github.com/${REPO}/releases/tag/${VERSION}"
    ok "Downloaded"

    # Make executable
    chmod +x "$TMP_FILE"

    # Install
    # Try /usr/local/bin first (needs sudo on most systems)
    # Fall back to ~/.local/bin if no sudo access
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
        ok "Installed to ${INSTALL_DIR}/${BINARY_NAME}"
    elif command -v sudo >/dev/null 2>&1; then
        info "Installing to ${INSTALL_DIR} (requires sudo)..."
        sudo mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
        ok "Installed to ${INSTALL_DIR}/${BINARY_NAME}"
    else
        # Fallback: install to ~/.local/bin
        INSTALL_DIR="${HOME}/.local/bin"
        mkdir -p "$INSTALL_DIR"
        mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
        ok "Installed to ${INSTALL_DIR}/${BINARY_NAME}"

        # Check if ~/.local/bin is in PATH
        case ":${PATH}:" in
            *":${INSTALL_DIR}:"*) ;;
            *)
                warn "${INSTALL_DIR} is not in your PATH."
                echo "  Add this to your shell profile (~/.bashrc or ~/.zshrc):"
                echo ""
                echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
                echo ""
                ;;
        esac
    fi

    # Cleanup
    rm -rf "$TMP_DIR"

    # Verify
    echo ""
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        ok "Verified: $(${BINARY_NAME} version 2>/dev/null || echo "${BINARY_NAME} ready")"
    else
        ok "Installed. You may need to restart your shell or update your PATH."
    fi

    # Next steps
    echo ""
    info "Next steps:"
    echo "  1. Get a llama-server binary:  https://github.com/ggml-org/llama.cpp"
    echo "  2. Get a GGUF model file:      https://huggingface.co (search for GGUF)"
    echo "  3. Run setup:"
    echo ""
    echo "     lazy-cli setup --llama-server /path/to/llama-server --model /path/to/model.gguf"
    echo ""
    echo "  4. Start the CLI:"
    echo ""
    echo "     lazy-cli"
    echo ""
}

main
