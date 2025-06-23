# env-sync

🔄 Securely sync .env files with Azure Key Vault using encrypted storage and shared team keys.

## ✨ Features

-   🔐 **Secure Encryption**: AES-256 encryption for all .env data
-   🔑 **Team Key Sharing**: Share encryption keys securely across team members
-   📁 **File Watching**: Automatic sync with configurable push/pull modes
-   🏥 **Health Monitoring**: Built-in diagnostics and dependency management
-   🔄 **Key Rotation**: Easy encryption key rotation for security maintenance
-   🐳 **Tilt Integration**: Seamless integration with Tilt development workflows

## 🚀 Quick Start

### Automatic Installation (Recommended)

```bash
# Install env-sync and all dependencies
curl -fsSL https://raw.githubusercontent.com/lliamscholtz/env-sync/main/scripts/install.sh | bash

# Or with Go
go install github.com/lliamscholtz/env-sync@latest

# Install dependencies automatically
env-sync install-deps

# Verify installation
env-sync doctor

# Check version
env-sync --version
```

### Manual Installation

If automatic installation fails, install dependencies manually:

**Azure CLI:**

-   **Windows**: `winget install Microsoft.AzureCLI`
-   **macOS**: `brew install azure-cli`
-   **Linux**: `curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash`

**Tilt (Optional):**

-   **Windows**: `choco install tilt`
-   **macOS**: `brew install tilt`
-   **Linux**: `curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash`

### System Requirements

-   **Operating System**: Windows 10+, macOS 10.15+, or Linux
-   **Architecture**: x86_64 (arm64 support coming soon)
-   **Network**: Internet connection for dependency installation and Azure access
-   **Permissions**: Administrator/sudo access for dependency installation

## 👥 Team Setup

### Step 1: Generate Shared Encryption Key (Team Lead)

```bash
# Generate a new encryption key
env-sync generate-key

# Copy the displayed key for team distribution
```

### Step 2: Distribute Key to Team

Share the encryption key with team members through secure channels:

-   **Recommended**: Password manager shared vault (1Password, Bitwarden)
-   Secure messaging (encrypted Slack/Teams)
-   Encrypted email
-   In-person for highly sensitive projects

### Step 3: Team Member Setup

Each team member configures the key:

**Option A: Environment Variable (Recommended)**

```bash
# Add to ~/.bashrc, ~/.zshrc, or equivalent
export ENVSYNC_ENCRYPTION_KEY="<base64-key-from-team-lead>"
```

**Option B: Key File**

```bash
# Save key to file (add to .gitignore!)
echo "<base64-key-from-team-lead>" > .env-sync-key
echo ".env-sync-key" >> .gitignore
```

### Step 4: Initialize Project

**Single Environment Setup:**

```bash
# Team lead initializes first
env-sync init --vault-url https://myteam-vault.vault.azure.net/ --secret-name myapp-dev-env --key-source env

# Push initial .env file
env-sync push

# Other team members initialize and pull
env-sync init --vault-url https://myteam-vault.vault.azure.net/ --secret-name myapp-dev-env --key-source env
env-sync pull
```

**Multi-Environment Setup (Recommended for teams):**

```bash
# Team lead creates configurations for each environment
env-sync init --sync-file .env-sync.dev.yaml --vault-url https://myteam-vault.vault.azure.net/ --secret-name myapp-dev-env --key-source env
env-sync init --sync-file .env-sync.qa.yaml --vault-url https://myteam-vault.vault.azure.net/ --secret-name myapp-qa-env --key-source env
env-sync init --sync-file .env-sync.prod.yaml --vault-url https://myteam-vault.vault.azure.net/ --secret-name myapp-prod-env --key-source env

# Push initial .env files for each environment
env-sync push --sync-file .env-sync.dev.yaml
env-sync push --sync-file .env-sync.qa.yaml
env-sync push --sync-file .env-sync.prod.yaml

# Other team members pull the configurations they need
env-sync pull --sync-file .env-sync.dev.yaml    # Developers
env-sync pull --sync-file .env-sync.qa.yaml     # QA team
env-sync pull --sync-file .env-sync.prod.yaml   # DevOps/Production team
```

### Daily Workflow

**Single Environment:**

```bash
# Pull latest changes
env-sync pull

# Make changes to .env file
# ...

# Push changes to team
env-sync push

# Or use file watcher for automatic sync
env-sync watch           # Pull-only mode (safe default)
env-sync watch --push    # Full sync mode (pull + push on changes)
```

**Multi-Environment:**

```bash
# Work with specific environments
env-sync pull --sync-file .env-sync.dev.yaml     # Pull dev changes
env-sync push --sync-file .env-sync.dev.yaml     # Push dev changes

# Watch specific environment
env-sync watch --sync-file .env-sync.dev.yaml    # Watch dev environment
env-sync watch --sync-file .env-sync.qa.yaml --push  # Watch QA with push enabled

# Check status of different environments
env-sync status --sync-file .env-sync.dev.yaml
env-sync status --sync-file .env-sync.qa.yaml
env-sync status --sync-file .env-sync.prod.yaml
```

### Key Rotation (Security Maintenance)

```bash
# Generate new key (team lead)
env-sync generate-key

# Share new key with team through secure channels
# Each team member updates their key storage

# Rotate key and re-encrypt content
env-sync rotate-key --old-key-source env --new-key "<new-base64-key>"

# All team members update their stored key and pull
env-sync pull
```

## 📋 Commands

### Core Commands

-   `env-sync init` - Initialize project configuration
-   `env-sync push` - Upload encrypted .env to Azure Key Vault
-   `env-sync pull` - Download and decrypt .env from Azure Key Vault
-   `env-sync watch` - Monitor and sync .env file changes (pull-only by default)
-   `env-sync watch --push` - Full sync mode with push on file changes

### Multi-Configuration Support

Use `--sync-file` to work with multiple configuration files for different environments:

```bash
# Create separate configurations for different environments
env-sync init --sync-file .env-sync.dev.yaml --vault-url <vault> --secret-name dev-secrets --key-source env
env-sync init --sync-file .env-sync.qa.yaml --vault-url <vault> --secret-name qa-secrets --key-source env
env-sync init --sync-file .env-sync.prod.yaml --vault-url <vault> --secret-name prod-secrets --key-source env

# Use different configurations for operations
env-sync push --sync-file .env-sync.dev.yaml     # Push to dev environment
env-sync pull --sync-file .env-sync.qa.yaml      # Pull from QA environment
env-sync watch --sync-file .env-sync.prod.yaml   # Watch prod environment
env-sync status --sync-file .env-sync.dev.yaml   # Check dev status
```

**Benefits:**

-   🎯 **Environment Isolation**: Separate configs for dev/qa/prod
-   🔐 **Different Keys**: Use different encryption keys per environment
-   🏗️ **Team Workflows**: Different team members can work on different environments
-   📁 **Project Organization**: Keep environment-specific settings organized

### File Watcher Modes

The `watch` command supports two modes:

**Pull-Only Mode (Default - Recommended for most teams)**

```bash
env-sync watch
```

-   ⬇️ Automatically pulls changes from Azure Key Vault every 15 minutes
-   🛡️ Ignores local file changes (prevents accidental overwrites)
-   🔒 Safe for team environments where multiple people might edit

**Full Sync Mode (Advanced)**

```bash
env-sync watch --push
```

-   ⬇️ Pulls changes from Azure Key Vault every 15 minutes
-   ⬆️ Pushes local changes to Azure Key Vault when .env file is modified
-   ⚠️ Includes anti-cycle protection to prevent sync loops
-   👥 Best for single-person workflows or when you're the primary editor

### Key Management

-   `env-sync generate-key` - Generate new encryption key for team sharing
-   `env-sync rotate-key` - Rotate encryption key and re-encrypt content

### System Management

-   `env-sync doctor` - Check system health and dependencies
-   `env-sync doctor --check <component>` - Check specific component (azure-cli, tilt, auth, config)
-   `env-sync doctor --fix` - Automatically fix detected issues
-   `env-sync install-deps` - Install required dependencies
-   `env-sync install-deps --only <dep>` - Install specific dependency (azure-cli, tilt)
-   `env-sync auth` - Check Azure authentication status
-   `env-sync status` - Show sync status and configuration

## 🐳 Tilt Integration

Add to your `Tiltfile`:

```python
load('ext://env_sync', 'env_sync')

env_sync(
    vault_url='https://my-vault.vault.azure.net/',
    secret_name='myapp-dev-env',
    env_file='.env',
    sync_interval='15m',
    key_source='env'
)
```

## 🔒 Security Best Practices

-   **Never commit encryption keys to version control**
-   Use secure channels for key distribution
-   Rotate keys periodically (quarterly recommended)
-   Use different keys for different environments
-   Add `.env-sync-key` to `.gitignore` if using file storage
-   Consider using password managers for team key storage
-   Use pull-only mode (`env-sync watch`) in team environments to prevent conflicts

## 🔧 Troubleshooting

### Common Issues

1. **"Azure CLI not found"**

    ```bash
    # Install only Azure CLI
    env-sync install-deps --only azure-cli

    # Install only Tilt
    env-sync install-deps --only tilt

    # Install all dependencies without prompts
    env-sync install-deps --yes
    ```

2. **"Permission denied during installation"**

    - **Windows**: Run PowerShell as Administrator
    - **macOS/Linux**: Use `sudo` if prompted

3. **"Package manager not found"**

    - **Windows**: Install Chocolatey or Windows Package Manager
    - **macOS**: Install Homebrew
    - **Linux**: Ensure apt, yum, or dnf is available

4. **"Cannot connect to Azure"**

    ```bash
    az login
    env-sync auth --check
    ```

5. **File Watcher Issues**

    ```bash
    # If watcher seems stuck or not responding
    env-sync pull  # Manual pull to verify connectivity

    # Check file permissions
    ls -la .env

    # Restart with verbose output
    env-sync watch --push  # or just 'env-sync watch'
    ```

### System Diagnostics

```bash
# Run comprehensive system check
env-sync doctor

# Check specific components
env-sync doctor --check azure-cli
env-sync doctor --check tilt
env-sync doctor --check auth

# Automatic problem resolution
env-sync doctor --fix
```

## ⚙️ Configuration

### Single Configuration File

Configuration is stored in `.env-sync.yaml`:

```yaml
vault_url: https://my-vault.vault.azure.net/
secret_name: myapp-dev-env
env_file: .env
sync_interval: 15m
key_source: env # env, file, or prompt
key_file: .env-sync-key # only if key_source is "file"
```

### Multiple Configuration Files

For multi-environment setups, create separate configuration files:

**`.env-sync.dev.yaml`:**

```yaml
vault_url: https://my-vault.vault.azure.net/
secret_name: myapp-dev-env
env_file: .env.dev
sync_interval: 15m
key_source: env
```

**`.env-sync.qa.yaml`:**

```yaml
vault_url: https://my-vault.vault.azure.net/
secret_name: myapp-qa-env
env_file: .env.qa
sync_interval: 30m
key_source: file
key_file: .env-sync-qa-key
```

**`.env-sync.prod.yaml`:**

```yaml
vault_url: https://my-vault.vault.azure.net/
secret_name: myapp-prod-env
env_file: .env.prod
sync_interval: 60m
key_source: file
key_file: .env-sync-prod-key
```

### Configuration Priority

When using `--sync-file`, it takes precedence over `--config`:

```bash
# Uses .env-sync.dev.yaml (--sync-file takes priority)
env-sync push --config .env-sync.yaml --sync-file .env-sync.dev.yaml

# Uses .env-sync.yaml (fallback to --config)
env-sync push --config .env-sync.yaml

# Uses default .env-sync.yaml (no flags specified)
env-sync push
```

## 🛠️ Development

If you want to build from source or contribute to `env-sync`, you'll need Go 1.21+ installed.

**Build the binary:**

```bash
make build
```

This will create the `env-sync` binary in the `bin/` directory.

**Run tests:**

```bash
make test
```

**Run linter:**

```bash
make lint
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run `make test` and `make lint`
6. Submit a pull request

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.
