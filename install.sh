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

# 2. Dependency Check (Docker or Nginx)
echo -e "Checking router prerequisites..."
HAS_DOCKER=0
HAS_NGINX=0
if command -v docker >/dev/null 2>&1; then
    HAS_DOCKER=1
fi
if command -v nginx >/dev/null 2>&1; then
    HAS_NGINX=1
fi

if [ $HAS_DOCKER -eq 0 ] && [ $HAS_NGINX -eq 0 ]; then
    echo -e "${YELLOW}Warning: Neither Docker nor Nginx was found on this system.${NC}"
    echo -e "piMan requires at least Docker or Nginx to manage router operations."
    
    # Check if apt-get is available to install nginx
    if command -v apt-get >/dev/null 2>&1; then
        read -p "Would you like to install Nginx now? [Y/n] " -n 1 -r REPLY_INSTALL_NGINX
        echo ""
        if [[ $REPLY_INSTALL_NGINX =~ ^[Yy]$ ]] || [ -z "$REPLY_INSTALL_NGINX" ]; then
            echo -e "Installing Nginx..."
            if [ "$USER" = "root" ]; then
                apt-get update && apt-get install -y nginx
            else
                sudo apt-get update && sudo apt-get install -y nginx
            fi
            HAS_NGINX=1
        fi
    fi
    
    if [ $HAS_NGINX -eq 0 ]; then
        echo -e "${RED}Error: Cannot proceed without Docker or Nginx installed.${NC}"
        exit 1
    fi
fi

# 3. Port Configuration & Availability Check
DEFAULT_PORT=8080
PORT=$DEFAULT_PORT

while true; do
    read -p "Enter port for piMan service [default: $DEFAULT_PORT]: " USER_PORT
    if [ -z "$USER_PORT" ]; then
        PORT=$DEFAULT_PORT
    else
        PORT=$USER_PORT
    fi
    
    # Check if port is a valid number
    if ! [[ "$PORT" =~ ^[0-9]+$ ]] || [ "$PORT" -lt 1 ] || [ "$PORT" -gt 65535 ]; then
        echo -e "${RED}Error: Port must be a valid number between 1 and 65535.${NC}"
        continue
    fi
    
    # Check port availability using ss, netstat, or lsof
    PORT_IN_USE=0
    if command -v ss >/dev/null 2>&1; then
        if ss -tuln | grep -q -E ":$PORT\b"; then
            PORT_IN_USE=1
        fi
    elif command -v netstat >/dev/null 2>&1; then
        if netstat -tuln | grep -q -E ":$PORT\b"; then
            PORT_IN_USE=1
        fi
    elif command -v lsof >/dev/null 2>&1; then
        if lsof -i :$PORT >/dev/null 2>&1; then
            PORT_IN_USE=1
        fi
    fi
    
    if [ $PORT_IN_USE -eq 1 ]; then
        echo -e "${RED}Error: Port $PORT is already in use. Please select a different port.${NC}"
    else
        echo -e "${GREEN}Port $PORT is available.${NC}"
        break
    fi
done

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
ExecStart=$INSTALL_DIR/piman serve --port $PORT
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
        echo -e "Access the server at: ${CYAN}http://localhost:$PORT${NC}"
        echo -e "Check service logs with: ${CYAN}journalctl -u piman -f${NC}"
    fi
fi

# 8. Interactive Nginx Reverse Proxy Setup
if command -v nginx >/dev/null 2>&1; then
    echo ""
    read -p "Do you want to configure Nginx to expose piMan at http://piman.local? [Y/n] " -n 1 -r REPLY_NGINX
    echo ""
    if [[ $REPLY_NGINX =~ ^[Yy]$ ]] || [ -z "$REPLY_NGINX" ]; then
        NGINX_CONF_DIR="/etc/nginx/conf.d"
        NGINX_CONF_FILE="$NGINX_CONF_DIR/piman.conf"
        
        echo -e "Generating Nginx configuration file..."
        NGINX_CONTENT="server {
    listen 80;
    server_name piman.local;

    location / {
        proxy_pass http://127.0.0.1:$PORT;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        
        # WebSockets support
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection \"upgrade\";
    }
}"
        
        # Ensure Nginx conf directory exists
        if [ ! -d "$NGINX_CONF_DIR" ]; then
            if [ -w "/etc/nginx" ]; then
                mkdir -p "$NGINX_CONF_DIR"
            else
                sudo mkdir -p "$NGINX_CONF_DIR"
            fi
        fi

        # Write the conf file
        if [ -w "$NGINX_CONF_DIR" ]; then
            echo "$NGINX_CONTENT" > "$NGINX_CONF_FILE"
        else
            echo "$NGINX_CONTENT" | sudo tee "$NGINX_CONF_FILE" > /dev/null
        fi
        
        # Validate Nginx configuration
        echo -e "Validating Nginx configuration..."
        if sudo nginx -t; then
            echo -e "Reloading Nginx service..."
            if command -v systemctl >/dev/null 2>&1; then
                sudo systemctl reload nginx || sudo systemctl restart nginx
            else
                sudo service nginx reload || sudo service nginx restart
            fi
            echo -e "${GREEN}Nginx configured successfully! piMan is now exposed at: http://piman.local${NC}"
        else
            echo -e "${RED}Error: Nginx configuration validation failed. Removing piman.conf...${NC}"
            if [ -w "$NGINX_CONF_FILE" ]; then
                rm "$NGINX_CONF_FILE"
            else
                sudo rm "$NGINX_CONF_FILE"
            fi
        fi
    fi
fi

# Clean up temp
rm -rf "$TEMP_DIR"
echo -e "\n${GREEN}Installation completed successfully!${NC}"
