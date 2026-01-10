#!/bin/sh
set -e

# ==============================================================================
# Alab Installer
# https://github.com/hlop3z/astroladb
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/hlop3z/astroladb/main/install.sh | sh
#   wget -qO- https://raw.githubusercontent.com/hlop3z/astroladb/main/install.sh | sh
# ==============================================================================

REPO="hlop3z/astroladb"
BINARY_NAME="alab"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() { printf "${BLUE}[INFO]${NC} %s\n" "$1"; }
success() { printf "${GREEN}[OK]${NC} %s\n" "$1"; }
warn() { printf "${YELLOW}[WARN]${NC} %s\n" "$1"; }
error() { printf "${RED}[ERROR]${NC} %s\n" "$1" >&2; exit 1; }

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *)       error "Unsupported OS: $(uname -s)" ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        armv7l)        echo "arm" ;;
        i386|i686)     echo "386" ;;
        *)             error "Unsupported architecture: $(uname -m)" ;;
    esac
}

# Get latest release version from GitHub
get_latest_version() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | \
        grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/' || echo ""
}

# Download and install
install_alab() {
    OS=$(detect_os)
    ARCH=$(detect_arch)

    info "Detected OS: $OS, Architecture: $ARCH"

    # Get latest version
    VERSION=$(get_latest_version)

    if [ -z "$VERSION" ]; then
        warn "Could not fetch latest release. Building from source..."
        install_from_source
        return
    fi

    info "Latest version: $VERSION"

    # Construct download URL
    EXT=""
    if [ "$OS" = "windows" ]; then
        EXT=".exe"
    fi

    FILENAME="${BINARY_NAME}_${VERSION#v}_${OS}_${ARCH}${EXT}"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

    info "Downloading from: $DOWNLOAD_URL"

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    # Download binary
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$BINARY_NAME$EXT" 2>/dev/null; then
        warn "Pre-built binary not available. Building from source..."
        install_from_source
        return
    fi

    # Make executable
    chmod +x "$TMP_DIR/$BINARY_NAME$EXT"

    # Install
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_DIR/$BINARY_NAME$EXT" "$INSTALL_DIR/$BINARY_NAME$EXT"
    else
        info "Requesting sudo to install to $INSTALL_DIR"
        sudo mv "$TMP_DIR/$BINARY_NAME$EXT" "$INSTALL_DIR/$BINARY_NAME$EXT"
    fi

    success "Installed $BINARY_NAME to $INSTALL_DIR/$BINARY_NAME$EXT"
}

# Build and install from source
install_from_source() {
    info "Building from source..."

    # Check for Go
    if ! command -v go >/dev/null 2>&1; then
        error "Go is required to build from source. Install from https://go.dev/dl/"
    fi

    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    info "Found Go $GO_VERSION"

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    # Clone repository
    info "Cloning repository..."
    git clone --depth 1 "https://github.com/${REPO}.git" "$TMP_DIR/astroladb" 2>/dev/null || \
        error "Failed to clone repository"

    cd "$TMP_DIR/astroladb"

    # Build
    info "Building..."
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$BINARY_NAME" ./cmd/alab || \
        error "Build failed"

    # Install
    if [ -w "$INSTALL_DIR" ]; then
        mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    else
        info "Requesting sudo to install to $INSTALL_DIR"
        sudo mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    fi

    success "Built and installed $BINARY_NAME to $INSTALL_DIR/$BINARY_NAME"
}

# Main
main() {
    echo ""
    echo "  ╭───────────────────────────────────────╮"
    echo "  │         Alab Installer                │"
    echo "  │   Database Migration Tool             │"
    echo "  ╰───────────────────────────────────────╯"
    echo ""

    install_alab

    echo ""
    success "Installation complete!"
    echo ""
    echo "  Get started:"
    echo "    $ alab init          # Initialize a new project"
    echo "    $ alab --help        # Show all commands"
    echo ""
    echo "  Documentation: https://github.com/${REPO}"
    echo ""
}

main
