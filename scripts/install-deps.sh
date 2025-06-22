#!/bin/bash

set -e

echo "🔧 Installing env-sync dependencies..."

# Detect OS and package manager
detect_os() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        if command -v apt-get >/dev/null 2>&1; then
            echo "linux-apt"
        elif command -v yum >/dev/null 2>&1; then
            echo "linux-yum"
        elif command -v dnf >/dev/null 2>&1; then
            echo "linux-dnf"
        else
            echo "linux-unknown"
        fi
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        echo "darwin"
    else
        echo "unknown"
    fi
}

# Install Azure CLI
install_azure_cli() {
    local os_type=$(detect_os)
    echo "📦 Installing Azure CLI for $os_type..."

    case $os_type in
        linux-apt)
            curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
            ;;
        linux-yum)
            sudo rpm --import https://packages.microsoft.com/keys/microsoft.asc
            echo -e '[azure-cli]\nname=Azure CLI\nbaseurl=https://packages.microsoft.com/yumrepos/azure-cli\nenabled=1\ngpgcheck=1\ngpgkey=https://packages.microsoft.com/keys/microsoft.asc' | sudo tee /etc/yum.repos.d/azure-cli.repo
            sudo yum install -y azure-cli
            ;;
        darwin)
            if command -v brew >/dev/null 2>&1; then
                brew install azure-cli
            else
                echo "❌ Homebrew not found. Please install Homebrew first: https://brew.sh"
                exit 1
            fi
            ;;
        *)
            echo "❌ Unsupported OS. Please install Azure CLI manually: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
            exit 1
            ;;
    esac
}

# Install Tilt (optional)
install_tilt() {
    local os_type=$(detect_os)
    echo "📦 Installing Tilt for $os_type..."

    case $os_type in
        linux-*)
            curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash
            ;;
        darwin)
            if command -v brew >/dev/null 2>&1; then
                brew install tilt
            else
                curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash
            fi
            ;;
        *)
            echo "❌ Unsupported OS for Tilt installation"
            exit 1
            ;;
    esac
}

# Main installation
main() {
    # Check if Azure CLI is installed
    if ! command -v az >/dev/null 2>&1; then
        install_azure_cli
    else
        echo "✅ Azure CLI already installed"
    fi

    # Check if Tilt is installed (optional)
    if ! command -v tilt >/dev/null 2>&1; then
        read -p "📦 Install Tilt for development workflow integration? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            install_tilt
        fi
    else
        echo "✅ Tilt already installed"
    fi

    echo "🎉 Dependency installation complete!"
    echo "💡 Next steps:"
    echo "   1. Run 'az login' to authenticate with Azure"
    echo "   2. Run 'env-sync doctor' to verify installation"
}

main "$@" 