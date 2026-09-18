package utils

import (
	"encoding/hex"
	"errors"
	"fmt"
)

func ValidateSecretsEncryptionKey(key string) ([]byte, error) {
	if key == "" {
		return nil, errors.New(
			"configuration error: NAVIRE_SECRETS_ENCRYPTION_KEY is required.\n" +
				"Generate one with:\n" +
				"export NAVIRE_SECRETS_ENCRYPTION_KEY=$(openssl rand -hex 32)",
		)
	}

	// Decode hex string to bytes
	decodedKey, err := hex.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf(
			"configuration error: NAVIRE_SECRETS_ENCRYPTION_KEY must be valid hex: %w\n"+
				"Generate one with:\n"+
				"export NAVIRE_SECRETS_ENCRYPTION_KEY=$(openssl rand -hex 32)",
			err,
		)
	}

	// Validate key length (accepted 32 bytes only)
	byteLen := len(decodedKey)

	if byteLen != 32 {
		return nil, fmt.Errorf(
			"configuration error: NAVIRE_SECRETS_ENCRYPTION_KEY must be 32 bytes for AES-256 (got %d bytes from %d hex chars)\n"+
				"Generate one with:\n"+
				"export NAVIRE_SECRETS_ENCRYPTION_KEY=$(openssl rand -hex 32)",
			byteLen,
			len(key),
		)
	}
	return decodedKey, nil
}
