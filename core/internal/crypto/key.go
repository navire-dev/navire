package crypto

import (
	"encoding/hex"
	"fmt"
)

func ParseKey(value, name string) ([]byte, error) {
	if value == "" {
		return nil, fmt.Errorf("%s is required", name)
	}
	key, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("%s must be valid hex: %w", name, err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("%s must contain a 32-byte AES-256 key", name)
	}
	return key, nil
}

// ParseSecretKey parses a high-entropy hexadecimal secret used for
// authentication. Unlike AES keys, HMAC secrets may be longer than 32 bytes.
func ParseSecretKey(value, name string) ([]byte, error) {
	if value == "" {
		return nil, fmt.Errorf("%s is required", name)
	}
	key, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("%s must be valid hex: %w", name, err)
	}
	if len(key) < 32 {
		return nil, fmt.Errorf("%s must contain at least a 32-byte secret", name)
	}
	return key, nil
}
