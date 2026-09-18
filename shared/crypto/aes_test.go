package crypto

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := []byte("12345678901234567890123456789012") // 32 bytes

	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple", "hello world"},
		{"with special chars", "token!@#$%^&*()"},
		{"empty", ""},
		{"long", "this is a very long token that should still work perfectly fine with AES-256-GCM encryption"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := Encrypt(tt.plaintext, key)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}

			if tt.plaintext == "" {
				if encrypted != "" {
					t.Errorf("expected empty string for empty input, got %q", encrypted)
				}
				return
			}

			// Check format
			if encrypted[:4] != "ENC[" || encrypted[len(encrypted)-1] != ']' {
				t.Errorf("encrypted format invalid: %s", encrypted)
			}

			// Decrypt
			decrypted, err := Decrypt(encrypted, key)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			// Compare
			if decrypted != tt.plaintext {
				t.Errorf("expected %q, got %q", tt.plaintext, decrypted)
			}
		})
	}
}

func TestDecryptPlaintext(t *testing.T) {
	key := []byte("12345678901234567890123456789012")

	// Should return as-is if not encrypted
	plaintext := "not-encrypted-token"
	result, err := Decrypt(plaintext, key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != plaintext {
		t.Errorf("expected %q, got %q", plaintext, result)
	}
}

func TestInvalidEncryptionKey(t *testing.T) {
	shortKey := []byte("short")

	_, err := Encrypt("test", shortKey)
	if err == nil {
		t.Error("expected error for short key")
	}

	_, err = Decrypt("ENC[test]", shortKey)
	if err == nil {
		t.Error("expected error for short key")
	}
}
