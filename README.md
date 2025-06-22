# env-sync

Securely sync .env files with Azure Key Vault using encrypted storage and shared team keys.

## Quick Start

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

## Team Setup

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

```bash
# Team lead initializes first
env-sync init --vault-url https://myteam-vault.vault.azure.net/ --secret-name myapp-dev-env --key-source env

# Push initial .env file
env-sync push

# Other team members initialize and pull
env-sync init --vault-url https://myteam-vault.vault.azure.net/ --secret-name myapp-dev-env --key-source env
env-sync pull
```

### Daily Workflow

```bash
# Pull latest changes
env-sync pull

# Make changes to .env file
# ...

# Push changes to team
env-sync push
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

## Commands

### Core Commands

-   `env-sync init` - Initialize project configuration
-   `env-sync push` - Upload encrypted .env to Azure Key Vault
-   `env-sync pull` - Download and decrypt .env from Azure Key Vault
-   `env-sync watch` - Monitor .env file and auto-sync changes

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

## Tilt Integration

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

## Security Best Practices

-   **Never commit encryption keys to version control**
-   Use secure channels for key distribution
-   Rotate keys periodically (quarterly recommended)
-   Use different keys for different environments
-   Add `.env-sync-key` to `.gitignore` if using file storage
-   Consider using password managers for team key storage

## Troubleshooting

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

## Configuration

Configuration is stored in `.env-sync.yaml`:

```yaml
vault_url: https://my-vault.vault.azure.net/
secret_name: myapp-dev-env
env_file: .env
sync_interval: 15m
key_source: env # env, file, or prompt
key_file: .env-sync-key # only if key_source is "file"
```

## Development

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

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run `make test` and `make lint`
6. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.
