#!/usr/bin/env bash

# piMan Installer Script
# Auto-detects OS/Architecture, downloads latest release, installs binary, and configures systemd.

set -e

# Colored Output Helpers
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${CYAN}"
echo "          _ __  ___"
echo "   ___ (_)  |/  /___ _ ___"
echo "  / _ \/ /|_/ // _ \`/ // /"
echo " / .__/_//_/ /_/\_,_/\_,_/"
echo "/_/         piMan Installer"
echo -e "${NC}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# 1. Platform Check
if [ "$OS" != "linux" ]; then
    echo -e "${YELLOW}Warning: piMan is optimized for Linux & Raspberry Pi clusters.${NC}"
    if [ "$OS" = "darwin" ]; then
        echo -e "For macOS, please build from source using:"
        echo -e "  ${CYAN}go install github.com/sameerchandra/piman@latest${NC}"
    else
        echo -e "${RED}Unsupported Operating System: ${OS}${NC}"
    fi
    exit 1
fi

# Determine architecture suffix
if [ "$ARCH" = "x86_64" ]; then
    ARCH_NAME="amd64"
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    ARCH_NAME="arm64"
elif [[ "$ARCH" =~ ^armv7 ]]; then
    ARCH_NAME="armv7"
else
    echo -e "${RED}Unsupported Linux architecture: ${ARCH}${NC}"
    exit 1
fi

# 2. Get latest version tag from GitHub API
echo -e "Fetching latest release tag from GitHub..."
LATEST_TAG=$(curl -s "https://api.github.com/repos/pimanager/piman/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$LATEST_TAG" ]; then
    echo -e "${YELLOW}Could not resolve latest release tag. Defaulting to v1.0.0...${NC}"
    LATEST_TAG="v1.0.0"
fi

# 3. Download release tarball
TEMP_DIR=$(mktemp -d)
TAR_FILE="$TEMP_DIR/piman.tar.gz"
DOWNLOAD_URL="https://github.com/pimanager/piman/releases/download/${LATEST_TAG}/piman-linux-${ARCH_NAME}.tar.gz"

echo -e "Downloading piMan ${CYAN}${LATEST_TAG}${NC} for ${CYAN}linux-${ARCH_NAME}${NC}..."
if ! curl -fsSL "$DOWNLOAD_URL" -o "$TAR_FILE"; then
    echo -e "${RED}Failed to download release asset from: ${DOWNLOAD_URL}${NC}"
    exit 1
fi

# 4. Extract package
echo -e "Extracting binary..."
tar -xzf "$TAR_FILE" -C "$TEMP_DIR"

# 5. Install to Path
INSTALL_DIR="/usr/local/bin"
if [ -w "$INSTALL_DIR" ]; then
    mv "$TEMP_DIR/piman" "$INSTALL_DIR/piman"
else
    echo -e "Write permissions to ${INSTALL_DIR} denied. Installing with ${YELLOW}sudo${NC}..."
    if ! sudo mv "$TEMP_DIR/piman" "$INSTALL_DIR/piman"; then
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
        mv "$TEMP_DIR/piman" "$INSTALL_DIR/piman"
        echo -e "Installed to local user bin: ${CYAN}${INSTALL_DIR}/piman${NC}"
        echo -e "Please ensure ${INSTALL_DIR} is present in your PATH."
    fi
fi
chmod +x "$INSTALL_DIR/piman"

# Get original logged-in user context
REAL_USER=$(logname 2>/dev/null || echo $USER)
REAL_HOME=$(eval echo "~$REAL_USER")

# 6. Initialize local directory config
echo -e "Running initialization..."
if [ "$USER" = "root" ] && [ "$REAL_USER" != "root" ]; then
    sudo -u "$REAL_USER" "$INSTALL_DIR/piman" init
else
    "$INSTALL_DIR/piman" init
fi

# 7. Interactive systemd Service Setup
if [ -d "/run/systemd/system" ] || [ -x "$(command -v systemctl)" ]; then
    echo ""
    read -p "Do you want to configure the piMan web server to run as a systemd service? [y/N] " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "Generating systemd service file..."
        SERVICE_CONTENT="[Unit]
Description=piMan Management Web Server
After=network.target

[Service]
Type=simple
User=$REAL_USER
Environment=HOME=$REAL_HOME
ExecStart=$INSTALL_DIR/piman serve --port 8080
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target"

        if [ -w "/etc/systemd/system" ]; then
            echo "$SERVICE_CONTENT" > /etc/systemd/system/piman.service
        else
            echo "$SERVICE_CONTENT" | sudo tee /etc/systemd/system/piman.service > /dev/null
        fi

        echo -e "Reloading systemd, enabling, and starting ${CYAN}piman.service${NC}..."
        sudo systemctl daemon-reload
        sudo systemctl enable piman.service
        sudo systemctl start piman.service

        echo -e "${GREEN}piMan service successfully started!${NC}"
        echo -e "Access the server at: ${CYAN}http://localhost:8080${NC}"
        echo -e "Check service logs with: ${CYAN}journalctl -u piman -f${NC}"
    fi
fi

# Clean up temp
rm -rf "$TEMP_DIR"
echo -e "\n${GREEN}Installation completed successfully!${NC}"
