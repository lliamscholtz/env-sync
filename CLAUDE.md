# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

env-sync is a Go CLI tool that securely syncs .env files with Azure Key Vault using AES-256 encryption. It supports team collaboration through shared encryption keys and provides file watching capabilities for automatic synchronization.

## Development Commands

### Building and Testing
- `make build` - Build the binary to `bin/env-sync`
- `make test` - Run all Go tests
- `make lint` - Run golangci-lint for code quality checks
- `make clean` - Remove build artifacts
- `make install` - Install binary via `go install`

### System Setup
- `make deps` - Install system dependencies (Azure CLI, Tilt)
- `make doctor` - Run health checks (builds first, then runs diagnostics)
- `make setup` - Complete setup (deps + build + doctor)

### Development Workflow
- `go test ./...` - Run tests in all packages
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
- **File Watching**: fsnotify-based file monitoring with configurable push/pull modes
- **Multi-Environment Support**: Separate config files for different environments (dev/qa/prod)

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
- **Pull-only mode** (default): `env-sync watch` - Safe for team environments
- **Full sync mode**: `env-sync watch --push` - Pushes local changes, includes anti-cycle protection

### Key Management
- Encryption keys can be sourced from environment variables, files, or prompts
- Support for key rotation with `rotate-key` command
- Different keys per environment for security isolation

### Health Diagnostics
The `doctor` command provides comprehensive system health checks for Azure CLI, Tilt, authentication, and configuration validation.