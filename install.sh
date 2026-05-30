#!/bin/bash
# imole - Installer script (macOS / Linux / Windows Git-Bash)
# Downloads pre-built binary and installs to INSTALL_DIR.

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

REPO="chenhg5/imole"
BINARY_NAME="imole"
INSTALL_DIR=""

# Detect OS
detect_os() {
    case "$OSTYPE" in
        darwin*)  echo "darwin" ;;
        linux*)   echo "linux" ;;
        msys*|cygwin*|win32*) echo "windows" ;;
        *)        echo "unknown" ;;
    esac
}

# Detect arch
detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        arm64|aarch64)  echo "arm64" ;;
        x86_64|amd64)   echo "amd64" ;;
        *)              echo "amd64" ;;
    esac
}

# Set default INSTALL_DIR based on OS
default_install_dir() {
    local os="$1"
    case "$os" in
        darwin|linux)
            echo "/usr/local/bin"
            ;;
        windows)
            if [[ -n "${LOCALAPPDATA:-}" ]]; then
                echo "$LOCALAPPDATA/imole/bin"
            else
                echo "$HOME/AppData/Local/imole/bin"
            fi
            ;;
        *)
            echo "/usr/local/bin"
            ;;
    esac
}

# Check if sudo is needed
needs_sudo() {
    [[ ! -w "$INSTALL_DIR" ]] && [[ ! -w "$(dirname "$INSTALL_DIR")" ]]
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
    local tag
    tag=$(curl -fsSL --connect-timeout 5 --max-time 10 \
        "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null |
        sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -1) || true

    if [[ -n "$tag" ]]; then
        echo "$tag"
        return 0
    fi

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

# Download and install
download_and_install() {
    local tag="$1"
    local os="$2"
    local arch="$3"
    local asset_name="${BINARY_NAME}-${os}-${arch}"
    [[ "$os" == "windows" ]] && asset_name="${asset_name}.exe"

    local tmp
    tmp="$(mktemp -d)"

    # Try source tarball first
    local url="https://github.com/${REPO}/archive/refs/tags/${tag}.tar.gz"
    log_info "Fetching ${BINARY_NAME} ${tag} for ${os}/${arch}..."

    if curl -fsSL --connect-timeout 10 --max-time 120 -o "$tmp/source.tar.gz" "$url" 2>/dev/null; then
        if tar -xzf "$tmp/source.tar.gz" -C "$tmp" 2>/dev/null; then
            local extracted_dir
            extracted_dir=$(find "$tmp" -mindepth 1 -maxdepth 1 -type d | head -n 1)

            if [[ -n "$extracted_dir" && -f "$extracted_dir/bin/${asset_name}" ]]; then
                mkdir -p "$INSTALL_DIR" 2>/dev/null || true
                if [[ "$os" == "windows" ]]; then
                    cp "$extracted_dir/bin/${asset_name}" "$INSTALL_DIR/${BINARY_NAME}.exe"
                else
                    verify_sudo
                    maybe_sudo cp "$extracted_dir/bin/${asset_name}" "$INSTALL_DIR/${BINARY_NAME}.new"
                    maybe_sudo chmod +x "$INSTALL_DIR/${BINARY_NAME}.new"
                    maybe_sudo mv -f "$INSTALL_DIR/${BINARY_NAME}.new" "$INSTALL_DIR/${BINARY_NAME}"
                fi
                rm -rf "$tmp"
                log_success "Installed ${BINARY_NAME} ${tag} to $INSTALL_DIR"
                return 0
            fi
        fi
    fi

    # Fallback: release asset
    local asset_url="https://github.com/${REPO}/releases/download/${tag}/${asset_name}"
    log_info "Trying release asset..."
    if curl -fSL --connect-timeout 10 --max-time 120 -o "$tmp/${asset_name}" "$asset_url" 2>/dev/null; then
        mkdir -p "$INSTALL_DIR" 2>/dev/null || true
        if [[ "$os" == "windows" ]]; then
            cp "$tmp/${asset_name}" "$INSTALL_DIR/${BINARY_NAME}.exe"
        else
            verify_sudo
            maybe_sudo cp "$tmp/${asset_name}" "$INSTALL_DIR/${BINARY_NAME}.new"
            maybe_sudo chmod +x "$INSTALL_DIR/${BINARY_NAME}.new"
            maybe_sudo mv -f "$INSTALL_DIR/${BINARY_NAME}.new" "$INSTALL_DIR/${BINARY_NAME}"
        fi
        rm -rf "$tmp"
        log_success "Installed ${BINARY_NAME} ${tag} to $INSTALL_DIR"
        return 0
    fi

    # Fallback: go build
    if command -v go > /dev/null 2>&1; then
        log_warn "Download failed, building from source..."
        if git clone --depth=1 "https://github.com/${REPO}.git" "$tmp/src" 2>/dev/null; then
            local out_name="${BINARY_NAME}"
            [[ "$os" == "windows" ]] && out_name="${BINARY_NAME}.exe"
            if (cd "$tmp/src" && go build -ldflags="-s -w" -o "$out_name" ./cmd/imole) 2>/dev/null; then
                mkdir -p "$INSTALL_DIR" 2>/dev/null || true
                if [[ "$os" == "windows" ]]; then
                    cp "$tmp/src/$out_name" "$INSTALL_DIR/${BINARY_NAME}.exe"
                else
                    verify_sudo
                    maybe_sudo cp "$tmp/src/$out_name" "$INSTALL_DIR/${BINARY_NAME}.new"
                    maybe_sudo chmod +x "$INSTALL_DIR/${BINARY_NAME}.new"
                    maybe_sudo mv -f "$INSTALL_DIR/${BINARY_NAME}.new" "$INSTALL_DIR/${BINARY_NAME}"
                fi
                rm -rf "$tmp"
                log_success "Built and installed ${BINARY_NAME} to $INSTALL_DIR"
                return 0
            fi
        fi
    fi

    rm -rf "$tmp"
    log_error "Failed to install ${BINARY_NAME}"
    echo ""
    echo "Manual install: https://github.com/${REPO}/releases/tag/${tag}"
    return 1
}

# Verify installation
verify() {
    local binary="$INSTALL_DIR/${BINARY_NAME}"
    [[ "$(detect_os)" == "windows" ]] && binary="${binary}.exe"

    if [[ -x "$binary" ]]; then
        local ver
        ver=$("$binary" --version 2>/dev/null || echo "unknown")
        log_success "${BINARY_NAME} ${ver} installed successfully"
    else
        log_error "Installation verification failed"
        exit 1
    fi
}

# PATH hint
path_hint() {
    local os="$1"
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo ""
        log_warn "$INSTALL_DIR is not in your PATH"
        case "$os" in
            darwin)
                echo "  Add to ~/.zshrc: export PATH=\"$INSTALL_DIR:\$PATH\""
                ;;
            linux)
                echo "  Add to ~/.bashrc: export PATH=\"$INSTALL_DIR:\$PATH\""
                ;;
            windows)
                echo "  Add to System Environment Variables or run:"
                echo "  setx PATH \"%PATH%;$INSTALL_DIR\""
                ;;
        esac
    fi
}

# Main
main() {
    local os
    os="$(detect_os)"

    if [[ "$os" == "unknown" ]]; then
        log_error "Unsupported OS: $OSTYPE"
        echo "Supported: macOS (darwin), Linux, Windows (Git-Bash/WSL)"
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
                echo "  --prefix DIR   Install to DIR"
                echo "  VERSION        Version tag (e.g., v0.1.0)"
                echo ""
                echo "Supported: macOS, Linux, Windows (Git-Bash/WSL)"
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

    [[ -z "$INSTALL_DIR" ]] && INSTALL_DIR="$(default_install_dir "$os")"

    if [[ -z "$tag" ]]; then
        log_info "Detecting latest version..."
        if ! tag="$(get_latest_tag)"; then
            log_error "Failed to detect latest version"
            echo "Try: $0 v0.1.0"
            exit 1
        fi
    fi

    local arch
    arch="$(detect_arch)"

    download_and_install "$tag" "$os" "$arch"
    verify
    path_hint "$os"

    echo ""
    echo "Usage: ${BINARY_NAME} --help"
    echo "       ${BINARY_NAME} scan"
    echo "       ${BINARY_NAME} backup --to /path/to/backup"
}

main "$@"