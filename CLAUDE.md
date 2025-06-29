# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

env-sync is a Go CLI tool that securely syncs .env files with Azure Key Vault using AES-256 encryption. It features intelligent conflict detection, robust file watching with atomic write support, and comprehensive team collaboration capabilities through shared encryption keys.

## Development Commands

### Building and Testing
- `make build` - Build the binary to `bin/env-sync`
- `make test` - Run all Go tests (including conflict resolution tests)
- `make lint` - Run golangci-lint for code quality checks
- `make clean` - Remove build artifacts
- `make install` - Install binary via `go install`

### System Setup
- `make deps` - Install system dependencies (Azure CLI, Tilt)
- `make doctor` - Run health checks (builds first, then runs diagnostics)
- `make setup` - Complete setup (deps + build + doctor)

### Development Workflow
- `go test ./...` - Run tests in all packages
- `go test ./internal/sync/...` - Run conflict resolution tests specifically
- `go build -o bin/env-sync ./cmd/main.go` - Build manually
- `./bin/env-sync doctor` - Check system health after building

## Architecture

### Project Structure
- `cmd/` - Application entrypoint (main.go)
- `internal/` - Core application logic (not exposed externally)
  - `auth/` - Azure authentication (azure.go)
  - `config/` - Configuration management
  - `crypto/` - AES-256 encryption/decryption
  - `utils/` - Helper utilities
  - `vault/` - Azure Key Vault client
  - `watcher/` - File watching functionality
- `scripts/` - Installation scripts for dependencies
- `tilt/` - Tilt integration files
- `demo/` - Example configurations for multiple users

### Key Components
- **Configuration System**: Uses Viper for YAML config files (`.env-sync.yaml`, `.env-sync.dev.yaml`, etc.)
- **Encryption**: AES-256 encryption with team-shared keys (env var, file, or prompt-based)
- **Azure Integration**: Uses Azure SDK for Key Vault operations with proper authentication
- **Smart File Watching**: fsnotify-based monitoring with atomic write support and robust change detection
- **Multi-Environment Support**: Separate config files for different environments (dev/qa/prod)
- **Intelligent Conflict Detection**: Key-level conflict detection with diff display and multiple resolution strategies
- **Enhanced Push Process**: Automatic conflict detection before every push with user confirmation prompts
- **Debug Mode**: Comprehensive debug logging for troubleshooting file change detection issues

### Architecture Patterns
The codebase follows Clean Architecture principles:
- Domain models for configuration and encryption
- Repository pattern for Key Vault operations
- Service layer for business logic
- CLI handlers for user interface

## Development Guidelines

### Go Standards
- Follow the Cursor rules defined in `.cursor/rules/go-microservices.mdc`
- Use Clean Architecture with proper separation of concerns
- Apply interface-driven development with dependency injection
- Write table-driven tests with good coverage
- Handle errors explicitly with context wrapping

### Security Considerations
- Never commit encryption keys or sensitive data
- Use secure defaults for all cryptographic operations
- Validate all external inputs rigorously
- Implement proper error handling without information leakage

### Testing Strategy
- Unit tests for all core functionality
- Integration tests for Azure Key Vault operations
- Test configuration loading and validation
- Mock external dependencies cleanly

## Multi-Environment Support

The tool supports multiple configuration files for different environments:
- `.env-sync.yaml` - Default configuration
- `.env-sync.dev.yaml` - Development environment
- `.env-sync.qa.yaml` - QA environment  
- `.env-sync.prod.yaml` - Production environment

Each environment can have different vault URLs, secret names, encryption keys, and sync intervals.

## Common Operations

### File Watching Modes
- **Safe mode** (default): `env-sync watch` - Pulls every 15min + manual push with conflict detection
- **Auto-push mode**: `env-sync watch --confirm=false` - Automatic push with conflict detection
- **Debug mode**: `env-sync watch --debug` - Detailed logging for troubleshooting file events
- **Custom timing**: Configurable via `sync_interval` in config (default 15 minutes)

### Conflict Resolution Strategies
- **manual** (default): Interactive conflict resolution with user prompts
- **local**: Local changes always win (remote changes discarded)
- **remote**: Remote changes always win (local changes discarded)  
- **merge**: Create conflict markers for manual resolution
- **backup**: Create backups and prefer local changes

### Key Management
- Encryption keys can be sourced from environment variables, files, or prompts
- Support for key rotation with `rotate-key` command
- Different keys per environment for security isolation

### Conflict Detection & Resolution
- **Intelligent Detection**: Key-level conflict detection that ignores non-overlapping changes
- **Diff Display**: Shows exactly which keys conflict with local vs remote values
- **Push Integration**: Automatic conflict detection before every push operation
- **Multiple Strategies**: manual, local, remote, merge, backup resolution options
- **Backup Creation**: Automatic timestamped backups in `.env-sync-backups/` when using backup strategy

### User Confirmation System
- **Push Confirmation**: Interactive prompts before pushing: `🚀 Push changes to remote? [y/N]:`
- **Conflict Confirmation**: Special prompts when conflicts detected with diff display
- **Safety-first Defaults**: Confirmation required unless explicitly disabled with `--confirm=false`
- **Graceful Handling**: Clear feedback when pushes are declined by user

### File Change Detection
- **Atomic Write Support**: Handles all modern editors (VS Code, vim, nano, etc.)
- **Multiple Changes**: Detects every consecutive file change, not just the first one
- **Smart Timing**: Prevents pull-triggered pushes while catching legitimate changes
- **Health Checks**: Periodic verification that file watcher is still active
- **Debug Logging**: Comprehensive event logging with `--debug` flag for troubleshooting

### Health Diagnostics
The `doctor` command provides comprehensive system health checks for Azure CLI, Tilt, authentication, and configuration validation.