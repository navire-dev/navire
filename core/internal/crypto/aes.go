package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

// Encrypt encrypts plaintext using AES-GCM and returns base64-encoded ciphertext with "ENC[...]" wrapper.
// The encryption key can be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256 respectively.
func Encrypt(plaintext string, key []byte) (string, error) {
	// Validate key length (AES supports 16, 24, or 32 bytes)
	keyLen := len(key)
	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		return "", fmt.Errorf("encryption key must be 16, 24, or 32 bytes for AES-128/192/256 (got %d)", keyLen)
	}

	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	return "ENC[" + encoded + "]", nil
}

// Decrypt decrypts ciphertext encrypted with Encrypt.
// If the input doesn't start with "ENC[", it's returned as-is (not encrypted).
func Decrypt(ciphertext string, key []byte) (string, error) {
	// Validate key length (AES supports 16, 24, or 32 bytes)
	keyLen := len(key)
	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		return "", fmt.Errorf("encryption key must be 16, 24, or 32 bytes for AES-128/192/256 (got %d)", keyLen)
	}

	// Not encrypted, return as-is
	if !strings.HasPrefix(ciphertext, "ENC[") {
		return ciphertext, nil
	}

	// Extract base64 data: ENC[base64data]
	encoded := strings.TrimSuffix(strings.TrimPrefix(ciphertext, "ENC["), "]")
	if encoded == "" {
		return "", fmt.Errorf("empty ciphertext")
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, encrypted := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}
