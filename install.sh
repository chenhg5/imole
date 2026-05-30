#!/bin/bash
# imole - Installer script for manual installs

set -euo pipefail

# Colors
if [[ -n "${NO_COLOR:-}" ]]; then
    GREEN='' BLUE='' YELLOW='' RED='' NC=''
else
    GREEN='\033[0;32m'
    BLUE='\033[0;34m'
    YELLOW='\033[1;33m'
    RED='\033[0;31m'
    NC='\033[0m'
fi

ICON_OK="✓"
ICON_ERR="☻"

log_info() { echo -e "${BLUE}${ICON_OK}${NC} $1"; }
log_success() { echo -e "${GREEN}${ICON_OK}${NC} $1"; }
log_error() { echo -e "${RED}${ICON_ERR}${NC} $1"; }
log_warn() { echo -e "${YELLOW}WARNING:${NC} $1"; }

INSTALL_DIR="/usr/local/bin"
BINARY_NAME="imole"
REPO="chenhg5/imole"

usage() {
    cat << EOF
imole installer

Usage: curl -fsSL https://raw.githubusercontent.com/chenhg5/imole/main/install.sh | bash [OPTIONS]

Options:
  -v, --version TAG Install specific version (e.g., v0.1.0)
  -p, --prefix DIR  Install to custom directory (default: /usr/local/bin)
  -h, --help        Show this help

Examples:
  curl -fsSL https://raw.githubusercontent.com/chenhg5/imole/main/install.sh | bash
  curl -fsSL https://raw.githubusercontent.com/chenhg5/imole/main/install.sh | bash -s v0.1.0

EOF
}

# Parse args
VERSION=""
PREFIX=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -p|--prefix)
            PREFIX="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log_warn "Unknown option: $1 (ignored)"
            shift
            ;;
    esac
done

[[ -n "$PREFIX" ]] && INSTALL_DIR="$PREFIX"

# Check requirements
check_requirements() {
    if [[ "$OSTYPE" != "darwin"* ]]; then
        log_error "imole is designed for macOS only"
        exit 1
    fi

    if command -v brew > /dev/null 2>&1 && brew list imole > /dev/null 2>&1; then
        log_warn "imole is installed via Homebrew"
        echo "Please use: brew upgrade imole"
        exit 1
    fi

    if [[ ! -d "$(dirname "$INSTALL_DIR")" ]]; then
        log_error "Directory $(dirname "$INSTALL_DIR") does not exist"
        exit 1
    fi
}

# Need sudo?
needs_sudo() {
    if [[ -e "$INSTALL_DIR/imole" && ! -w "$INSTALL_DIR" ]]; then
        return 0
    fi
    [[ ! -w "$(dirname "$INSTALL_DIR")" ]]
}

maybe_sudo() {
    if needs_sudo; then
        sudo "$@"
    else
        "$@"
    fi
}

# Get latest release tag — try API first, fallback to git ls-remote
get_latest_tag() {
    # Try GitHub API
    local tag
    tag=$(curl -fsSL --connect-timeout 5 --max-time 10 \
        "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null |
        sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -1) || true

    if [[ -n "$tag" ]]; then
        echo "$tag"
        return 0
    fi

    # Fallback: git ls-remote
    if command -v git > /dev/null 2>&1; then
        tag=$(git ls-remote --tags --refs "https://github.com/${REPO}.git" 2>/dev/null |
            awk -F/ '{print $NF}' |
            grep -E '^v[0-9]' |
            sort -V |
            tail -1) || true
        if [[ -n "$tag" ]]; then
            echo "$tag"
            return 0
        fi
    fi

    return 1
}

# Detect arch
get_arch() {
    local arch
    arch="$(uname -m)"
    if [[ "$arch" == "arm64" ]]; then
        echo "darwin-arm64"
    else
        echo "darwin-amd64"
    fi
}

# Download release binary
download_binary() {
    local tag="$1"
    local arch="$2"
    local url="https://github.com/${REPO}/releases/download/${tag}/${BINARY_NAME}-${arch}"
    local tmp_file="/tmp/${BINARY_NAME}-${arch}"

    log_info "Downloading ${BINARY_NAME} ${tag} for ${arch}..."

    if ! curl -fsSL --connect-timeout 10 --max-time 120 -o "$tmp_file" "$url"; then
        log_error "Failed to download ${BINARY_NAME}"
        rm -f "$tmp_file"
        return 1
    fi

    maybe_sudo cp "$tmp_file" "$INSTALL_DIR/${BINARY_NAME}"
    maybe_sudo chmod +x "$INSTALL_DIR/${BINARY_NAME}"
    rm -f "$tmp_file"

    log_success "Installed ${BINARY_NAME} ${tag} to $INSTALL_DIR"
    return 0
}

# Verify installation
verify() {
    if [[ -x "$INSTALL_DIR/${BINARY_NAME}" ]]; then
        local ver
        ver=$("$INSTALL_DIR/${BINARY_NAME}" --version 2>/dev/null || echo "unknown")
        log_success "${BINARY_NAME} ${ver} installed successfully"
    else
        log_error "Installation verification failed"
        exit 1
    fi
}

# Main
main() {
    check_requirements

    local tag
    if [[ -n "$VERSION" ]]; then
        tag="$VERSION"
    else
        log_info "Detecting latest version..."
        if ! tag="$(get_latest_tag)"; then
            log_error "Failed to detect latest version. Try: $0 -v v0.1.0"
            exit 1
        fi
    fi

    if ! download_binary "$tag" "$(get_arch)"; then
        log_error "Installation failed. Download the binary manually from:"
        echo "  https://github.com/${REPO}/releases/tag/${tag}"
        exit 1
    fi

    verify

    # PATH hint
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo ""
        log_warn "$INSTALL_DIR is not in your PATH"
        echo "Add to ~/.zshrc: export PATH=\"$INSTALL_DIR:\$PATH\""
    fi

    echo ""
    echo "Usage: imole --help"
    echo "        imole scan"
    echo "        imole backup --to /path/to/backup"
}

main