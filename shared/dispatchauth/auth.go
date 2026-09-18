package dispatchauth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
)

type RegistrationRequest struct {
	DispatcherID string `json:"dispatcher_id"`
	Timestamp    int64  `json:"timestamp"`
	Nonce        string `json:"nonce"`
	Signature    string `json:"signature"`
}

type RegistrationResponse struct {
	Status       string `json:"status"`
	DispatcherID string `json:"dispatcher_id"`
	Token        string `json:"registration_token"`
	LeaseSeconds int    `json:"lease_seconds"`
}

// Sign creates the proof used by a Dispatcher during registration.
func Sign(secret []byte, dispatcherID string, timestamp int64, nonce string) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = fmt.Fprintf(mac, "%s\n%d\n%s", dispatcherID, timestamp, nonce)
	return hex.EncodeToString(mac.Sum(nil))
}

func EqualSignature(expected, actual string) bool {
	expectedBytes, err := hex.DecodeString(expected)
	if err != nil {
		return false
	}
	actualBytes, err := hex.DecodeString(actual)
	if err != nil {
		return false
	}
	return hmac.Equal(expectedBytes, actualBytes)
}

func Hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func NewToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate registration token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func NewNonce() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate registration nonce: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func Timestamp(value int64) string {
	return strconv.FormatInt(value, 10)
}
