#!/bin/bash
# install.sh - ops0 installation script

set -e

REPO="ops0-ai/ops0-cli"  # Replace with your GitHub username
BINARY_NAME="ops0"
INSTALL_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check if running as root for installation
check_permissions() {
    if [[ $EUID -eq 0 ]]; then
        INSTALL_DIR="/usr/local/bin"
        SUDO=""
    else
        # Check if we can write to /usr/local/bin
        if [[ -w "/usr/local/bin" ]]; then
            SUDO=""
        else
            SUDO="sudo"
            print_warning "Will need sudo access to install to $INSTALL_DIR"
        fi
    fi
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case $ARCH in
        x86_64) ARCH="x86_64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        i386|i686) ARCH="i386" ;;
        armv7l) ARCH="armv7" ;;
        *) 
            print_error "Unsupported architecture: $ARCH"
            exit 1 
            ;;
    esac

    case $OS in
        darwin) OS="Darwin" ;;
        linux) OS="Linux" ;;
        mingw*|msys*|cygwin*) OS="Windows" ;;
        *) 
            print_error "Unsupported OS: $OS"
            exit 1 
            ;;
    esac

    print_status "Detected platform: $OS/$ARCH"
}

# Get latest release from GitHub API
get_latest_release() {
    print_status "Fetching latest release information..."
    
    LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

    if [ -z "$LATEST_RELEASE" ]; then
        print_error "Failed to get latest release information"
        print_error "Please check if the repository exists: https://github.com/$REPO"
        exit 1
    fi

    print_status "Latest release: $LATEST_RELEASE"
}

# Download and install binary
install_binary() {
    # Construct download URL
    if [ "$OS" = "Windows" ]; then
        ARCHIVE_NAME="${BINARY_NAME}_${OS}_${ARCH}.zip"
    else
        ARCHIVE_NAME="${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
    fi
    
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_RELEASE/$ARCHIVE_NAME"
    
    print_status "Downloading $BINARY_NAME $LATEST_RELEASE..."
    print_status "URL: $DOWNLOAD_URL"

    # Create temporary directory
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"

    # Download archive
    if ! curl -L "$DOWNLOAD_URL" -o "$ARCHIVE_NAME"; then
        print_error "Failed to download $ARCHIVE_NAME"
        print_error "Please check if the release exists: $DOWNLOAD_URL"
        exit 1
    fi

    # Extract archive
    print_status "Extracting archive..."
    if [ "$OS" = "Windows" ]; then
        unzip -q "$ARCHIVE_NAME"
    else
        tar -xzf "$ARCHIVE_NAME"
    fi

    # Install binary
    if [ "$OS" = "Windows" ]; then
        BINARY_NAME="${BINARY_NAME}.exe"
        INSTALL_DIR="/usr/bin"  # For Git Bash/WSL
    fi

    print_status "Installing $BINARY_NAME to $INSTALL_DIR..."
    
    if ! $SUDO mv "$BINARY_NAME" "$INSTALL_DIR/"; then
        print_error "Failed to install $BINARY_NAME to $INSTALL_DIR"
        print_error "Please check permissions or install manually"
        exit 1
    fi

    # Make executable (not needed on Windows)
    if [ "$OS" != "Windows" ]; then
        $SUDO chmod +x "$INSTALL_DIR/$BINARY_NAME"
    fi

    # Cleanup
    cd /
    rm -rf "$TMP_DIR"
}

# Verify installation
verify_installation() {
    print_status "Verifying installation..."
    
    if command -v $BINARY_NAME >/dev/null 2>&1; then
        VERSION_OUTPUT=$($BINARY_NAME --version 2>/dev/null || echo "version info not available")
        print_success "$BINARY_NAME installed successfully!"
        print_success "Location: $(which $BINARY_NAME)"
        print_success "Version: $VERSION_OUTPUT"
    else
        print_error "Installation verification failed"
        print_error "$BINARY_NAME not found in PATH"
        exit 1
    fi
}

# Main installation flow
main() {
    echo "🚀 ops0 Installation Script"
    echo "============================"
    
    check_permissions
    detect_platform
    get_latest_release
    install_binary
    verify_installation
    
    echo ""
    print_success "Installation complete! 🎉"
    echo ""
    echo "Get started with:"
    echo "  $BINARY_NAME -m \"i want to plan my iac code\""
    echo ""
    echo "For more examples and documentation:"
    echo "  https://github.com/$REPO"
    echo ""
}

# Run main function
main "$@"