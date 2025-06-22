package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

const (
	// KeySize is the expected size of the encryption key (32 bytes for AES-256).
	KeySize = 32
	// NonceSize is the size of the nonce (12 bytes for GCM).
	NonceSize = 12
)

// GenerateEncryptionKey creates a new 256-bit (32-byte) encryption key.
func GenerateEncryptionKey() ([]byte, error) {
	return GenerateRandomBytes(KeySize)
}

// GenerateRandomBytes generates a slice of random bytes of the specified length.
func GenerateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return bytes, nil
}

// DisplayKeyForSharing converts a key to a base64 encoded string for easy sharing.
func DisplayKeyForSharing(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}

// ValidateEncryptionKey checks if the key has the correct size.
func ValidateEncryptionKey(key []byte) error {
	if len(key) != KeySize {
		return fmt.Errorf("invalid key size: must be %d bytes", KeySize)
	}
	return nil
}

// EncryptEnvContent encrypts content using AES-256-GCM.
// The output is a base64 encoded string: nonce + ciphertext + tag
func EncryptEnvContent(content []byte, key []byte) (string, error) {
	if err := ValidateEncryptionKey(key); err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, content, nil)

	// Prepend nonce to the ciphertext
	encryptedData := append(nonce, ciphertext...)

	return base64.StdEncoding.EncodeToString(encryptedData), nil
}

// DecryptEnvContent decrypts a base64 encoded string using AES-256-GCM.
func DecryptEnvContent(encodedData string, key []byte) ([]byte, error) {
	if err := ValidateEncryptionKey(key); err != nil {
		return nil, err
	}

	encryptedData, err := base64.StdEncoding.DecodeString(encodedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 data: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedData) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return plaintext, nil
}

// RotateKey decrypts content with an old key and re-encrypts it with a new key.
func RotateKey(oldKey, newKey []byte, encryptedContent string) (string, error) {
	decryptedContent, err := DecryptEnvContent(encryptedContent, oldKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt with old key during rotation: %w", err)
	}

	newEncryptedContent, err := EncryptEnvContent(decryptedContent, newKey)
	if err != nil {
		return "", fmt.Errorf("failed to re-encrypt with new key during rotation: %w", err)
	}

	return newEncryptedContent, nil
}

// KeyToString formats the key as either base64 or hex.
func KeyToString(key []byte, format string) (string, error) {
	switch format {
	case "base64":
		return base64.StdEncoding.EncodeToString(key), nil
	case "hex":
		return hex.EncodeToString(key), nil
	default:
		return "", fmt.Errorf("unsupported key format: %s", format)
	}
}
