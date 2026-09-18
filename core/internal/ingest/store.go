package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

var ErrEndpointNotFound = errors.New("endpoint not found")

// EndpointStore exposes the active endpoint projection without exposing its
// storage implementation to API features.
type EndpointStore interface {
	GetEndpoint(context.Context, string) (Endpoint, error)
}

// RedisEndpointStore reads the active endpoint projection from Redis.
type RedisEndpointStore struct {
	Client *redis.Client
}

func (s RedisEndpointStore) GetEndpoint(ctx context.Context, name string) (Endpoint, error) {
	if s.Client == nil {
		return Endpoint{}, fmt.Errorf("redis client is required")
	}
	raw, err := s.Client.HGet(ctx, EndpointsHashKey, name).Bytes()
	if errors.Is(err, redis.Nil) {
		return Endpoint{}, ErrEndpointNotFound
	}
	if err != nil {
		return Endpoint{}, err
	}
	var endpoint Endpoint
	if err := json.Unmarshal(raw, &endpoint); err != nil {
		return Endpoint{}, fmt.Errorf("decode endpoint %s: %w", name, err)
	}
	return endpoint, nil
}
