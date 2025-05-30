#!/bin/bash

set -e

# GoFlux CLI Installation Script
# Usage: curl -fsSL https://raw.githubusercontent.com/barisgit/goflux/main/scripts/install.sh | bash

REPO="barisgit/goflux"
BINARY_NAME="flux"
INSTALL_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    armv7l) ARCH="armv7" ;;
    *) 
        echo -e "${RED}❌ Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

case $OS in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    *)
        echo -e "${RED}❌ Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

echo -e "${BLUE}🚀 GoFlux CLI Installer${NC}"
echo "Detected: $OS/$ARCH"

# Get latest release version
echo -e "${YELLOW}📡 Fetching latest release...${NC}"
LATEST_VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo -e "${RED}❌ Failed to get latest version${NC}"
    exit 1
fi

echo "Latest version: $LATEST_VERSION"

# Construct download URL
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/flux_${LATEST_VERSION#v}_${OS}_${ARCH}.tar.gz"

if [ "$OS" = "windows" ]; then
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/flux_${LATEST_VERSION#v}_${OS}_${ARCH}.zip"
fi

echo -e "${YELLOW}📥 Downloading $BINARY_NAME $LATEST_VERSION...${NC}"
echo "URL: $DOWNLOAD_URL"

# Create temporary directory
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

# Download and extract
if command -v curl >/dev/null 2>&1; then
    curl -sL "$DOWNLOAD_URL" -o archive
elif command -v wget >/dev/null 2>&1; then
    wget -q "$DOWNLOAD_URL" -O archive
else
    echo -e "${RED}❌ curl or wget required${NC}"
    exit 1
fi

# Extract archive
if [ "$OS" = "windows" ]; then
    unzip -q archive
else
    tar -xzf archive
fi

# Make binary executable
chmod +x "$BINARY_NAME"

# Check if we can write to install directory
if [ -w "$INSTALL_DIR" ]; then
    echo -e "${YELLOW}📦 Installing $BINARY_NAME to $INSTALL_DIR...${NC}"
    mv "$BINARY_NAME" "$INSTALL_DIR/"
else
    echo -e "${YELLOW}📦 Installing $BINARY_NAME to $INSTALL_DIR (requires sudo)...${NC}"
    sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
fi

# Cleanup
cd /
rm -rf "$TMP_DIR"

# Verify installation
if command -v "$BINARY_NAME" >/dev/null 2>&1; then
    echo -e "${GREEN}✅ GoFlux CLI installed successfully!${NC}"
    echo ""
    echo -e "${BLUE}🎯 Quick Start:${NC}"
    echo "  flux new my-app"
    echo "  cd my-app"
    echo "  flux dev"
    echo ""
    echo -e "${BLUE}📖 Documentation:${NC}"
    echo "  https://github.com/$REPO"
    echo ""
    flux --version
else
    echo -e "${RED}❌ Installation failed. $BINARY_NAME not found in PATH.${NC}"
    echo "You may need to add $INSTALL_DIR to your PATH."
    exit 1
fi 