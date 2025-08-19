#!/bin/bash

# RepoSentry systemd installation script

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="reposentry"
SERVICE_NAME="reposentry"
USER_NAME="reposentry"
GROUP_NAME="reposentry"
CONFIG_DIR="/etc/reposentry"
DATA_DIR="/var/lib/reposentry"
LOG_DIR="/var/log/reposentry"
BINARY_PATH="/usr/local/bin/${BINARY_NAME}"

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

check_binary() {
    if [[ ! -f "bin/${BINARY_NAME}-linux" ]]; then
        log_error "Binary bin/${BINARY_NAME}-linux not found. Run 'make build-linux' first."
        exit 1
    fi
}

create_user() {
    if ! id -u "${USER_NAME}" >/dev/null 2>&1; then
        log_info "Creating user ${USER_NAME}..."
        useradd --system --shell /bin/false --home "${DATA_DIR}" --create-home "${USER_NAME}"
    else
        log_info "User ${USER_NAME} already exists"
    fi
}

create_directories() {
    log_info "Creating directories..."
    
    # Config directory
    mkdir -p "${CONFIG_DIR}"
    chown root:root "${CONFIG_DIR}"
    chmod 755 "${CONFIG_DIR}"
    
    # Data directory
    mkdir -p "${DATA_DIR}"
    chown "${USER_NAME}:${GROUP_NAME}" "${DATA_DIR}"
    chmod 750 "${DATA_DIR}"
    
    # Log directory
    mkdir -p "${LOG_DIR}"
    chown "${USER_NAME}:${GROUP_NAME}" "${LOG_DIR}"
    chmod 750 "${LOG_DIR}"
}

install_binary() {
    log_info "Installing binary..."
    cp "bin/${BINARY_NAME}-linux" "${BINARY_PATH}"
    chown root:root "${BINARY_PATH}"
    chmod 755 "${BINARY_PATH}"
}

install_config() {
    log_info "Installing configuration..."
    
    # Copy example config if no config exists
    if [[ ! -f "${CONFIG_DIR}/config.yaml" ]]; then
        cp configs/example.yaml "${CONFIG_DIR}/config.yaml"
        chown root:root "${CONFIG_DIR}/config.yaml"
        chmod 640 "${CONFIG_DIR}/config.yaml"
        log_warn "Default configuration installed at ${CONFIG_DIR}/config.yaml"
        log_warn "Please edit this file with your repository settings"
    else
        log_info "Configuration file already exists at ${CONFIG_DIR}/config.yaml"
    fi
    
    # Create environment file if it doesn't exist
    if [[ ! -f "${CONFIG_DIR}/environment" ]]; then
        cat > "${CONFIG_DIR}/environment" << 'EOF'
# Environment variables for RepoSentry
# Add your tokens here:
# GITHUB_TOKEN=your_github_token_here
# GITLAB_TOKEN=your_gitlab_token_here
EOF
        chown root:root "${CONFIG_DIR}/environment"
        chmod 600 "${CONFIG_DIR}/environment"
        log_warn "Environment file created at ${CONFIG_DIR}/environment"
        log_warn "Please add your API tokens to this file"
    fi
}

install_service() {
    log_info "Installing systemd service..."
    cp deployments/systemd/reposentry.service /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable "${SERVICE_NAME}"
}

main() {
    log_info "Installing RepoSentry systemd service..."
    
    check_root
    check_binary
    create_user
    create_directories
    install_binary
    install_config
    install_service
    
    log_info "Installation completed successfully!"
    echo
    log_info "Next steps:"
    echo "  1. Edit configuration: ${CONFIG_DIR}/config.yaml"
    echo "  2. Add API tokens: ${CONFIG_DIR}/environment"
    echo "  3. Start service: sudo systemctl start ${SERVICE_NAME}"
    echo "  4. Check status: sudo systemctl status ${SERVICE_NAME}"
    echo "  5. View logs: sudo journalctl -u ${SERVICE_NAME} -f"
}

# Handle command line arguments
case "${1:-install}" in
    install)
        main
        ;;
    uninstall)
        log_info "Uninstalling RepoSentry..."
        systemctl stop "${SERVICE_NAME}" 2>/dev/null || true
        systemctl disable "${SERVICE_NAME}" 2>/dev/null || true
        rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
        rm -f "${BINARY_PATH}"
        systemctl daemon-reload
        log_info "Service uninstalled. Data and config preserved."
        log_info "To remove all data: sudo rm -rf ${CONFIG_DIR} ${DATA_DIR} ${LOG_DIR}"
        log_info "To remove user: sudo userdel ${USER_NAME}"
        ;;
    *)
        echo "Usage: $0 [install|uninstall]"
        exit 1
        ;;
esac
