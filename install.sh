#!/bin/bash
# imole - Installer script
# Downloads source tarball (contains pre-built binary) and installs to INSTALL_DIR.

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
REPO="chenhg5/imole"
BINARY_NAME="imole"

# Check if sudo is needed
needs_sudo() {
    if [[ -e "$INSTALL_DIR" ]]; then
        [[ ! -w "$INSTALL_DIR" ]]
        return
    fi
    local parent_dir
    parent_dir="$(dirname "$INSTALL_DIR")"
    [[ ! -w "$parent_dir" ]]
}

maybe_sudo() {
    if needs_sudo; then
        sudo "$@"
    else
        "$@"
    fi
}

# Pre-verify sudo if needed
verify_sudo() {
    if needs_sudo; then
        log_info "Admin access required for $INSTALL_DIR"
        sudo -v
    fi
}

# Get latest release tag
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

# Download source tarball and extract binary
download_and_install() {
    local tag="$1"
    local arch="$2"
    local tmp
    tmp="$(mktemp -d)"

    # Try downloading the source tarball (contains pre-built binary)
    local url="https://github.com/${REPO}/archive/refs/tags/${tag}.tar.gz"
    log_info "Fetching ${BINARY_NAME} ${tag}..."

    if curl -fsSL --connect-timeout 10 --max-time 120 -o "$tmp/source.tar.gz" "$url" 2>/dev/null; then
        if tar -xzf "$tmp/source.tar.gz" -C "$tmp" 2>/dev/null; then
            local extracted_dir
            extracted_dir=$(find "$tmp" -mindepth 1 -maxdepth 1 -type d | head -n 1)

            if [[ -n "$extracted_dir" && -f "$extracted_dir/bin/${BINARY_NAME}-${arch}" ]]; then
                verify_sudo
                mkdir -p "$INSTALL_DIR" 2>/dev/null || true
                # Atomic install: copy to .new then move
                maybe_sudo cp "$extracted_dir/bin/${BINARY_NAME}-${arch}" "$INSTALL_DIR/${BINARY_NAME}.new"
                maybe_sudo chmod +x "$INSTALL_DIR/${BINARY_NAME}.new"
                maybe_sudo mv -f "$INSTALL_DIR/${BINARY_NAME}.new" "$INSTALL_DIR/${BINARY_NAME}"
                rm -rf "$tmp"
                log_success "Installed ${BINARY_NAME} ${tag} to $INSTALL_DIR"
                return 0
            fi
        fi
    fi

    # Fallback: try direct release asset download
    local asset_url="https://github.com/${REPO}/releases/download/${tag}/${BINARY_NAME}-${arch}"
    log_info "Trying release asset..."
    if curl -fSL --connect-timeout 10 --max-time 120 -o "$tmp/${BINARY_NAME}" "$asset_url" 2>/dev/null; then
        verify_sudo
        mkdir -p "$INSTALL_DIR" 2>/dev/null || true
        maybe_sudo cp "$tmp/${BINARY_NAME}" "$INSTALL_DIR/${BINARY_NAME}.new"
        maybe_sudo chmod +x "$INSTALL_DIR/${BINARY_NAME}.new"
        maybe_sudo mv -f "$INSTALL_DIR/${BINARY_NAME}.new" "$INSTALL_DIR/${BINARY_NAME}"
        rm -rf "$tmp"
        log_success "Installed ${BINARY_NAME} ${tag} to $INSTALL_DIR"
        return 0
    fi

    # Fallback: git clone and build
    if command -v go > /dev/null 2>&1; then
        log_warn "Download failed, building from source..."
        if git clone --depth=1 "https://github.com/${REPO}.git" "$tmp/src" 2>/dev/null; then
            if (cd "$tmp/src" && go build -ldflags="-s -w" -o "${BINARY_NAME}" ./cmd/imole) 2>/dev/null; then
                verify_sudo
                mkdir -p "$INSTALL_DIR" 2>/dev/null || true
                maybe_sudo cp "$tmp/src/${BINARY_NAME}" "$INSTALL_DIR/${BINARY_NAME}.new"
                maybe_sudo chmod +x "$INSTALL_DIR/${BINARY_NAME}.new"
                maybe_sudo mv -f "$INSTALL_DIR/${BINARY_NAME}.new" "$INSTALL_DIR/${BINARY_NAME}"
                rm -rf "$tmp"
                log_success "Built and installed ${BINARY_NAME} to $INSTALL_DIR"
                return 0
            fi
        fi
    fi

    rm -rf "$tmp"
    log_error "Failed to install ${BINARY_NAME}"
    echo ""
    echo "Manual install: download from https://github.com/${REPO}/releases/tag/${tag}"
    return 1
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
    if [[ "$OSTYPE" != "darwin"* ]]; then
        log_error "imole is designed for macOS only"
        exit 1
    fi

    # Parse args
    local tag=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --prefix)
                INSTALL_DIR="$2"
                shift 2
                ;;
            -h|--help)
                echo "Usage: $0 [--prefix DIR] [VERSION]"
                echo ""
                echo "Options:"
                echo "  --prefix DIR   Install to DIR (default: /usr/local/bin)"
                echo "  VERSION        Version tag (e.g., v0.1.0)"
                exit 0
                ;;
            v*)
                tag="$1"
                shift
                ;;
            *)
                log_warn "Unknown option: $1"
                shift
                ;;
        esac
    done

    if [[ -z "$tag" ]]; then
        log_info "Detecting latest version..."
        if ! tag="$(get_latest_tag)"; then
            log_error "Failed to detect latest version"
            echo "Try: $0 v0.1.0"
            exit 1
        fi
    fi

    local arch
    arch="$(get_arch)"

    download_and_install "$tag" "$arch"
    verify

    # PATH hint
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo ""
        log_warn "$INSTALL_DIR is not in your PATH"
        echo "Add to ~/.zshrc: export PATH=\"$INSTALL_DIR:\$PATH\""
    fi

    echo ""
    echo "Usage: imole --help"
    echo "       imole scan"
    echo "       imole backup --to /path/to/backup"
}

main "$@"