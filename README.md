# env-sync

🔄 Securely sync .env files with Azure Key Vault using encrypted storage and shared team keys.

## ✨ Features

-   🔐 **Secure Encryption**: AES-256 encryption for all .env data
-   🔑 **Team Key Sharing**: Share encryption keys securely across team members
-   📁 **Smart File Watching**: Automatic sync with atomic write support and robust change detection
-   ⚡ **Conflict Detection**: Intelligent conflict resolution with diff display and multiple strategies
-   🛡️ **Safe Defaults**: Confirmation prompts and manual conflict resolution by default
-   🏥 **Health Monitoring**: Built-in diagnostics and dependency management
-   🔄 **Key Rotation**: Easy encryption key rotation for security maintenance
-   🐳 **Tilt Integration**: Seamless integration with Tilt development workflows
-   🐛 **Debug Mode**: Comprehensive debug logging for troubleshooting file changes

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

# Push changes to team (now includes conflict detection)
env-sync push

# Or use file watcher for automatic sync
env-sync watch                    # Safe mode: pulls + manual push confirmation
env-sync watch --confirm=false   # Auto-push without confirmation  
env-sync watch --debug          # Enable debug logging for troubleshooting
```

**Multi-Environment:**

```bash
# Work with specific environments
env-sync pull --sync-file .env-sync.dev.yaml     # Pull dev changes
env-sync push --sync-file .env-sync.dev.yaml     # Push dev changes (with conflict detection)

# Watch specific environment with different conflict strategies
env-sync watch --sync-file .env-sync.dev.yaml    # Manual conflict resolution (safe)
env-sync watch --sync-file .env-sync.qa.yaml     # Automatic backups on conflicts
env-sync watch --sync-file .env-sync.prod.yaml --confirm=false  # Auto-push for CI/CD

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

### File Watcher Features

The `watch` command includes intelligent conflict detection and robust file change monitoring:

**Smart Conflict Detection**

```bash
env-sync watch                    # Manual conflict resolution with diff display
env-sync watch --confirm=false   # Auto-push with conflict detection
```

-   🔍 **Conflict Detection**: Automatically detects when multiple users modify the same keys
-   📊 **Diff Display**: Shows exactly which keys conflict with local vs remote values
-   🛡️ **Safe Defaults**: Always asks for confirmation before overwriting remote changes
-   ⚡ **Real Conflicts Only**: Ignores non-overlapping changes (different keys)

**Robust File Monitoring**

```bash
env-sync watch --debug          # Enable debug logging
```

-   📁 **Atomic Write Support**: Handles all editor types (VS Code, vim, nano, etc.)
-   🔄 **Multiple Changes**: Detects every file change, not just the first one
-   🕐 **Smart Timing**: Prevents pull-triggered pushes while catching real changes
-   🎯 **Reliable Detection**: Automatically re-establishes file watching if needed

**Conflict Resolution Strategies**

Configure automatic conflict resolution in your `.env-sync.yaml`:

```yaml
conflict_strategy: "manual"    # Ask user (safe default)
conflict_strategy: "backup"    # Create backups and use local
conflict_strategy: "local"     # Always use local changes
conflict_strategy: "remote"    # Always use remote changes  
conflict_strategy: "merge"     # Create merge conflict file
```

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

### ⚠️ Critical Security Warnings

-   **NEVER use `--key` parameter in production** - Keys are visible in process lists
-   **Never commit encryption keys to version control** 
-   **Never store keys in world-readable files** - Use `chmod 600` for key files

### 🛡️ Secure Key Management

-   **Environment Variables**: Use `ENVSYNC_ENCRYPTION_KEY` for CI/CD systems
-   **Key Files**: Store with `chmod 600` permissions and add to `.gitignore`
-   **Interactive Prompt**: Most secure option - keys never stored on disk
-   **Different Keys**: Use separate encryption keys per environment (dev/staging/prod)
-   **Key Rotation**: Rotate keys quarterly using `env-sync rotate-key`

### 🏢 Production Security

-   **Azure RBAC**: Use principle of least privilege for Key Vault access
-   **Network Security**: Configure private endpoints and firewall rules for Key Vault
-   **Audit Logging**: Enable Key Vault access logging and monitoring
-   **File Permissions**: Secure config files with `chmod 600 .env-sync*.yaml`

### 📋 Quick Security Checklist

- [ ] Keys stored securely (not in CLI args or world-readable files)
- [ ] Different keys for each environment
- [ ] Key Vault access restricted and monitored
- [ ] Configuration files have proper permissions
- [ ] Regular key rotation schedule established

**📖 For comprehensive security guidance, see [SECURITY.md](SECURITY.md)**

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
    # Enable debug mode to see detailed file events
    env-sync watch --debug

    # If watcher seems stuck or not responding
    env-sync pull  # Manual pull to verify connectivity

    # Check file permissions
    ls -la .env

    # Test with different editors if changes aren't detected
    echo "TEST_KEY=new_value" >> .env  # Direct file write test
    ```

6. **Conflict Detection Issues**

    ```bash
    # Test conflict detection manually
    env-sync push  # Will show conflicts if any exist

    # Check current conflict strategy
    cat .env-sync.yaml | grep conflict_strategy

    # Change conflict strategy for testing
    env-sync init  # Recreate config with manual strategy
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
conflict_strategy: manual # manual, local, remote, merge, backup
auto_backup: false # enable automatic backups on conflicts
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
conflict_strategy: manual    # Safe for development
auto_backup: true           # Keep backup history
```

**`.env-sync.qa.yaml`:**

```yaml
vault_url: https://my-vault.vault.azure.net/
secret_name: myapp-qa-env
env_file: .env.qa
sync_interval: 30m
key_source: file
key_file: .env-sync-qa-key
conflict_strategy: backup   # Automatic with backups
auto_backup: true
```

**`.env-sync.prod.yaml`:**

```yaml
vault_url: https://my-vault.vault.azure.net/
secret_name: myapp-prod-env
env_file: .env.prod
sync_interval: 60m
key_source: file
key_file: .env-sync-prod-key
conflict_strategy: manual   # Extra caution for production
auto_backup: true
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
