package registration

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	"github.com/navire-dev/navire/shared/dispatchauth"
	"github.com/redis/go-redis/v9"
)

const (
	SecretHashKey      = "navire:dispatchers:auth"
	SecretHashField    = "registration_secret_hash"
	runtimeKeyPrefix   = "navire:dispatchers:runtime:"
	nonceKeyPrefix     = "navire:dispatchers:registration:nonce:"
	registrationWindow = 5 * time.Minute
	registrationLease  = 30 * time.Second
)

type (
	Request  = dispatchauth.RegistrationRequest
	Response = dispatchauth.RegistrationResponse
)

type Service struct {
	Redis  *redis.Client
	Secret []byte
	Now    func() time.Time
}

func (s Service) Register(ctx context.Context, request Request) (Response, error) {
	if s.Redis == nil {
		return Response{}, fmt.Errorf("registration Redis dependency is not configured")
	}
	if len(s.Secret) == 0 {
		return Response{}, fmt.Errorf("registration secret is not configured")
	}
	if request.DispatcherID == "" || request.Nonce == "" || request.Signature == "" {
		return Response{}, fmt.Errorf("dispatcher_id, nonce and signature are required")
	}
	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	if delta := now.Sub(time.Unix(request.Timestamp, 0)); delta > registrationWindow || delta < -registrationWindow {
		return Response{}, fmt.Errorf("registration timestamp is outside the allowed window")
	}
	expected := dispatchauth.Sign(s.Secret, request.DispatcherID, request.Timestamp, request.Nonce)
	if !dispatchauth.EqualSignature(expected, request.Signature) {
		return Response{}, fmt.Errorf("invalid registration signature")
	}

	nonceKey := nonceKeyPrefix + dispatchauth.Hash(request.Nonce)
	accepted, err := s.Redis.SetNX(ctx, nonceKey, request.DispatcherID, registrationWindow).Result()
	if err != nil {
		return Response{}, fmt.Errorf("store registration nonce: %w", err)
	}
	if !accepted {
		return Response{}, fmt.Errorf("registration nonce was already used")
	}

	token, err := dispatchauth.NewToken()
	if err != nil {
		return Response{}, err
	}
	if err := s.storeRuntime(ctx, request.DispatcherID, token, now); err != nil {
		return Response{}, err
	}
	return Response{
		Status:       "registered",
		DispatcherID: request.DispatcherID,
		Token:        token,
		LeaseSeconds: int(registrationLease / time.Second),
	}, nil
}

func (s Service) EnsureSecretHash(ctx context.Context) error {
	if s.Redis == nil {
		return fmt.Errorf("registration Redis dependency is not configured")
	}
	if len(s.Secret) == 0 {
		return fmt.Errorf("registration secret is not configured")
	}
	if err := s.Redis.HSet(ctx, SecretHashKey, SecretHashField, dispatchauth.Hash(string(s.Secret))).Err(); err != nil {
		return fmt.Errorf("store dispatcher registration secret hash: %w", err)
	}
	return nil
}

func (s Service) AuthenticateDispatcher(ctx context.Context, dispatcherID, token string) bool {
	if s.Redis == nil || dispatcherID == "" || token == "" {
		return false
	}
	stored, err := s.Redis.HGet(ctx, runtimeKeyPrefix+dispatcherID, "token_hash").Result()
	if err != nil || stored == "" {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(stored), []byte(dispatchauth.Hash(token))) != 1 {
		return false
	}
	_ = s.Redis.HSet(ctx, runtimeKeyPrefix+dispatcherID, "last_seen_at", time.Now().UTC().Format(time.RFC3339Nano)).Err()
	_ = s.Redis.Expire(ctx, runtimeKeyPrefix+dispatcherID, registrationLease).Err()
	return true
}

func (s Service) storeRuntime(ctx context.Context, dispatcherID, token string, now time.Time) error {
	key := runtimeKeyPrefix + dispatcherID
	values := map[string]any{
		"token_hash":       dispatchauth.Hash(token),
		"registered_at":    now.UTC().Format(time.RFC3339Nano),
		"last_seen_at":     now.UTC().Format(time.RFC3339Nano),
		"lease_expires_at": now.Add(registrationLease).UTC().Format(time.RFC3339Nano),
	}
	if err := s.Redis.HSet(ctx, key, values).Err(); err != nil {
		return fmt.Errorf("store dispatcher registration: %w", err)
	}
	if err := s.Redis.Expire(ctx, key, registrationLease).Err(); err != nil {
		return fmt.Errorf("set dispatcher registration lease: %w", err)
	}
	return nil
}

func ValidateDispatcherID(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("dispatcher_id is required")
	}
	if len(value) > 128 {
		return fmt.Errorf("dispatcher_id must be at most 128 characters")
	}
	return nil
}
