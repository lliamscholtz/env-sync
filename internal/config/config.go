package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/lliamscholtz/env-sync/internal/crypto"
	"github.com/lliamscholtz/env-sync/internal/utils"
	"github.com/spf13/viper"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

// Config holds the application's configuration.
type Config struct {
	VaultURL         string        `yaml:"vault_url" mapstructure:"vault_url"`
	SecretName       string        `yaml:"secret_name" mapstructure:"secret_name"`
	EnvFile          string        `yaml:"env_file" mapstructure:"env_file"`
	SyncInterval     time.Duration `yaml:"sync_interval" mapstructure:"sync_interval"`
	KeySource        string        `yaml:"key_source" mapstructure:"key_source"` // "env", "file", "prompt"
	KeyFile          string        `yaml:"key_file" mapstructure:"key_file"`   // Path to key file if key_source is "file"
	ConflictStrategy string        `yaml:"conflict_strategy" mapstructure:"conflict_strategy"` // "manual", "local", "remote", "merge", "backup"
	AutoBackup       bool          `yaml:"auto_backup" mapstructure:"auto_backup"` // Enable automatic backups on conflicts
}

// LoadConfig loads the configuration from the given file path.
// It uses a new viper instance to avoid global state issues.
func LoadConfig(path string) (*Config, error) {
	v := viper.New()
	if path != "" {
		v.SetConfigFile(path)
		v.SetConfigType("yaml") // Explicitly set the config type
	} else {
		// Default behavior if no path is provided
		v.SetConfigName(".env-sync")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found is okay, we'll use defaults or command-line flags.
			utils.PrintDebug("📂 Config file not found, using defaults.")
		} else {
			// Other errors are fatal.
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set defaults for any zero values
	if cfg.SyncInterval == 0 {
		cfg.SyncInterval = 15 * time.Minute
	}
	if cfg.EnvFile == "" {
		cfg.EnvFile = ".env"
	}
	if cfg.ConflictStrategy == "" {
		cfg.ConflictStrategy = "manual" // Safe default
	}

	// Validate the final configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the configuration values are valid.
func (c *Config) Validate() error {
	if c.VaultURL == "" {
		return fmt.Errorf("vault_url is required")
	}
	if c.SecretName == "" {
		return fmt.Errorf("secret_name is required")
	}
	if c.EnvFile == "" {
		c.EnvFile = ".env" // Default value
	}
	if c.KeySource == "" {
		return fmt.Errorf("key_source is required (env, file, or prompt)")
	}
	return nil
}

// WriteToFile saves the configuration to a YAML file.
func (c *Config) WriteToFile(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetEncryptionKey loads the encryption key based on the configured source.
func (c *Config) GetEncryptionKey(cliKey string) ([]byte, error) {
	// Priority order: CLI flag -> Env Var -> Key File -> Prompt
	if cliKey != "" {
		return base64.StdEncoding.DecodeString(cliKey)
	}

	switch c.KeySource {
	case "env":
		key := os.Getenv("ENVSYNC_ENCRYPTION_KEY")
		if key == "" {
			return nil, fmt.Errorf("key_source is 'env', but ENVSYNC_ENCRYPTION_KEY environment variable is not set")
		}
		return base64.StdEncoding.DecodeString(key)
	case "file":
		if c.KeyFile == "" {
			return nil, fmt.Errorf("key_source is 'file', but key_file is not specified in config")
		}
		keyData, err := os.ReadFile(c.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file '%s': %w", c.KeyFile, err)
		}
		return base64.StdEncoding.DecodeString(string(keyData))
	case "prompt":
		// Check if we're in a non-interactive environment (for tests)
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return nil, fmt.Errorf("cannot use prompt in non-interactive mode")
		}
		fmt.Print("Enter encryption key (base64): ")
		keyInput, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return nil, fmt.Errorf("failed to read key from prompt: %w", err)
		}
		return base64.StdEncoding.DecodeString(string(keyInput))
	default:
		return nil, fmt.Errorf("invalid key source: '%s'. Must be one of: env, file, prompt", c.KeySource)
	}
}

// LoadAndValidateKey is a helper to load the key and validate it.
func (c *Config) LoadAndValidateKey(cliKey string) ([]byte, error) {
	key, err := c.GetEncryptionKey(cliKey)
	if err != nil {
		return nil, err
	}
	if err := crypto.ValidateEncryptionKey(key); err != nil {
		return nil, err
	}
	return key, nil
}
