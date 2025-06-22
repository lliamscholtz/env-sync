package crypto

import (
	"bytes"
	"testing"
)

func TestGenerateEncryptionKey(t *testing.T) {
	key, err := GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("failed to generate encryption key: %v", err)
	}

	if len(key) != KeySize {
		t.Errorf("expected key size %d, got %d", KeySize, len(key))
	}
}

func TestValidateEncryptionKey(t *testing.T) {
	t.Run("valid key", func(t *testing.T) {
		key, _ := GenerateEncryptionKey()
		if err := ValidateEncryptionKey(key); err != nil {
			t.Errorf("validation failed for a valid key: %v", err)
		}
	})

	t.Run("invalid key size", func(t *testing.T) {
		key := []byte("shortkey")
		if err := ValidateEncryptionKey(key); err == nil {
			t.Error("validation succeeded for an invalid key")
		}
	})
}

func TestEncryptDecrypt(t *testing.T) {
	key, err := GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	originalContent := []byte("this is a super secret message")

	encrypted, err := EncryptEnvContent(originalContent, key)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if encrypted == "" {
		t.Fatal("encrypted content is empty")
	}

	decrypted, err := DecryptEnvContent(encrypted, key)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if !bytes.Equal(originalContent, decrypted) {
		t.Errorf("decrypted content does not match original. got: %s, want: %s", decrypted, originalContent)
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1, _ := GenerateEncryptionKey()
	key2, _ := GenerateEncryptionKey()

	originalContent := []byte("another secret")
	encrypted, _ := EncryptEnvContent(originalContent, key1)

	_, err := DecryptEnvContent(encrypted, key2)
	if err == nil {
		t.Fatal("decryption succeeded with the wrong key, but it should have failed")
	}
}

func TestRotateKey(t *testing.T) {
	oldKey, _ := GenerateEncryptionKey()
	newKey, _ := GenerateEncryptionKey()

	originalContent := []byte("content to be re-encrypted")
	encryptedWithOldKey, _ := EncryptEnvContent(originalContent, oldKey)

	encryptedWithNewKey, err := RotateKey(oldKey, newKey, encryptedWithOldKey)
	if err != nil {
		t.Fatalf("key rotation failed: %v", err)
	}

	decryptedWithNewKey, err := DecryptEnvContent(encryptedWithNewKey, newKey)
	if err != nil {
		t.Fatalf("decryption failed with new key after rotation: %v", err)
	}

	if !bytes.Equal(originalContent, decryptedWithNewKey) {
		t.Error("rotated content does not match original")
	}

	// Make sure it cannot be decrypted with the old key
	_, err = DecryptEnvContent(encryptedWithNewKey, oldKey)
	if err == nil {
		t.Error("decryption succeeded with old key after rotation")
	}
}
