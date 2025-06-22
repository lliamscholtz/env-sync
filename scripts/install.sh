#!/bin/bash

set -e

# env-sync installer script
# Downloads the latest release and installs it

REPO="lliamscholtz/env-sync"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="env-sync"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Installing env-sync...${NC}"

# Detect OS and architecture
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    
    case $os in
        linux) os="linux" ;;
        darwin) os="darwin" ;;
        *) echo -e "${RED}❌ Unsupported OS: $os${NC}"; exit 1 ;;
    esac
    
    case $arch in
        x86_64) arch="x86_64" ;;
        arm64|aarch64) arch="arm64" ;;
        *) echo -e "${RED}❌ Unsupported architecture: $arch${NC}"; exit 1 ;;
    esac
    
    echo "${os}_${arch}"
}

# Get the latest release version
get_latest_version() {
    curl -s "https://api.github.com/repos/$REPO/releases/latest" | \
        grep '"tag_name":' | \
        sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install binary
install_binary() {
    local platform=$(detect_platform)
    local version=$(get_latest_version)
    
    if [ -z "$version" ]; then
        echo -e "${RED}❌ Failed to get latest version${NC}"
        exit 1
    fi
    
    echo -e "${BLUE}📦 Downloading env-sync $version for $platform...${NC}"
    
    local download_url="https://github.com/$REPO/releases/download/$version/env-sync_${platform}.tar.gz"
    local temp_dir=$(mktemp -d)
    local archive_file="$temp_dir/env-sync.tar.gz"
    
    # Download
    if ! curl -L -o "$archive_file" "$download_url"; then
        echo -e "${RED}❌ Failed to download from $download_url${NC}"
        exit 1
    fi
    
    # Extract
    tar -xzf "$archive_file" -C "$temp_dir"
    
    # Install
    if [ -w "$INSTALL_DIR" ]; then
        mv "$temp_dir/$BINARY_NAME" "$INSTALL_DIR/"
    else
        echo -e "${YELLOW}🔐 Installing to system directory (requires sudo)...${NC}"
        sudo mv "$temp_dir/$BINARY_NAME" "$INSTALL_DIR/"
    fi
    
    # Set permissions
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    
    # Cleanup
    rm -rf "$temp_dir"
    
    echo -e "${GREEN}✅ env-sync installed successfully!${NC}"
}

# Check if already installed
check_existing() {
    if command -v env-sync >/dev/null 2>&1; then
        local current_version=$(env-sync --version 2>/dev/null || echo "unknown")
        echo -e "${YELLOW}⚠️ env-sync is already installed ($current_version)${NC}"
        read -p "Do you want to overwrite it? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo -e "${BLUE}Installation cancelled.${NC}"
            exit 0
        fi
    fi
}

# Install dependencies
install_deps() {
    echo -e "${BLUE}🔧 Installing dependencies...${NC}"
    if ! env-sync install-deps; then
        echo -e "${YELLOW}⚠️ Dependency installation failed. You can install them manually later.${NC}"
    fi
}

# Main installation flow
main() {
    echo -e "${BLUE}=== env-sync Installer ===${NC}"
    
    # Check for existing installation
    check_existing
    
    # Install binary
    install_binary
    
    # Verify installation
    if ! command -v env-sync >/dev/null 2>&1; then
        echo -e "${RED}❌ Installation failed - binary not found in PATH${NC}"
        echo -e "${YELLOW}💡 You may need to restart your terminal or add $INSTALL_DIR to your PATH${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}🎉 Installation complete!${NC}"
    echo
    echo -e "${BLUE}Next steps:${NC}"
    echo -e "  1. Run ${GREEN}env-sync install-deps${NC} to install dependencies"
    echo -e "  2. Run ${GREEN}env-sync doctor${NC} to verify your setup"
    echo -e "  3. Run ${GREEN}env-sync init${NC} to initialize your project"
    echo
    echo -e "${BLUE}Documentation: https://github.com/$REPO${NC}"
}

main "$@" 