package config

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lliamscholtz/env-sync/internal/crypto"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	content := `
vault_url: "https://my-test-vault.vault.azure.net"
secret_name: "my-test-secret"
env_file: ".env.test"
sync_interval: "30m"
key_source: "file"
key_file: ".test-key"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".env-sync.yaml")
	err := os.WriteFile(configPath, []byte(content), 0644)
	assert.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	assert.Equal(t, "https://my-test-vault.vault.azure.net", cfg.VaultURL)
	assert.Equal(t, "my-test-secret", cfg.SecretName)
	assert.Equal(t, ".env.test", cfg.EnvFile)
	assert.Equal(t, 30*time.Minute, cfg.SyncInterval)
	assert.Equal(t, "file", cfg.KeySource)
	assert.Equal(t, ".test-key", cfg.KeyFile)
}

func TestConfigValidation(t *testing.T) {
	testCases := []struct {
		name      string
		config    *Config
		expectErr bool
	}{
		{"valid config", &Config{VaultURL: "a", SecretName: "b", KeySource: "env"}, false},
		{"missing vault url", &Config{SecretName: "b", KeySource: "env"}, true},
		{"missing secret name", &Config{VaultURL: "a", KeySource: "env"}, true},
		{"missing key source", &Config{VaultURL: "a", SecretName: "b"}, true},
		{"default env file", &Config{VaultURL: "a", SecretName: "b", KeySource: "env", EnvFile: ""}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetEncryptionKey(t *testing.T) {
	// Reset viper for a clean test run
	viper.Reset()

	key, _ := crypto.GenerateEncryptionKey()
	b64Key := base64.StdEncoding.EncodeToString(key)

	t.Run("from CLI flag", func(t *testing.T) {
		cfg := &Config{KeySource: "env"} // Source doesn't matter when CLI key is present
		retrievedKey, err := cfg.GetEncryptionKey(b64Key)
		assert.NoError(t, err)
		assert.Equal(t, key, retrievedKey)
	})

	t.Run("from env var", func(t *testing.T) {
		cfg := &Config{KeySource: "env"}
		t.Setenv("ENVSYNC_ENCRYPTION_KEY", b64Key)
		retrievedKey, err := cfg.GetEncryptionKey("")
		assert.NoError(t, err)
		assert.Equal(t, key, retrievedKey)
	})

	t.Run("from file", func(t *testing.T) {
		// Reset viper for a clean test run
		viper.Reset()

		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "test.key")
		// Create a proper base64-encoded key (32 bytes for AES-256)
		rawKey := make([]byte, 32)
		for i := range rawKey {
			rawKey[i] = byte(i)
		}
		b64Key := base64.StdEncoding.EncodeToString(rawKey)
		err := os.WriteFile(keyPath, []byte(b64Key), 0644)
		assert.NoError(t, err)

		cfg := &Config{KeySource: "file", KeyFile: keyPath}
		key, err := cfg.GetEncryptionKey("")
		assert.NoError(t, err)
		assert.Equal(t, rawKey, key)
	})

	t.Run("file source but no file path", func(t *testing.T) {
		cfg := &Config{KeySource: "file", KeyFile: ""}
		_, err := cfg.GetEncryptionKey("")
		assert.Error(t, err)
	})

	t.Run("env source but no env var", func(t *testing.T) {
		cfg := &Config{KeySource: "env"}
		// Ensure the env var is not set
		t.Setenv("ENVSYNC_ENCRYPTION_KEY", "")
		_, err := cfg.GetEncryptionKey("")
		assert.Error(t, err)
	})

	t.Run("from prompt", func(t *testing.T) {
		// This test is hard to automate without mocking stdin, so we'll just check the error case
		// Reset viper for a clean test run
		viper.Reset()
		cfg := &Config{KeySource: "prompt"}
		_, err := cfg.GetEncryptionKey("")
		assert.Error(t, err, "expected error when prompting in non-interactive test")
		assert.Contains(t, err.Error(), "cannot use prompt in non-interactive mode")
	})

	t.Run("invalid source", func(t *testing.T) {
		// Reset viper for a clean test run
		viper.Reset()
		cfg := &Config{
			KeySource: "invalid",
		}
		_, err := cfg.GetEncryptionKey("")
		assert.Error(t, err)
	})
}
