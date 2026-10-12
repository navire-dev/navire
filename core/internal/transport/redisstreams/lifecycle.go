package redisstreams

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/navire-dev/navire/core/internal/lifecycle"
	"github.com/redis/go-redis/v9"
)

const (
	LifecycleStream          = "navire:lifecycle"
	LifecycleRetention       = time.Hour
	LifecycleCleanupInterval = time.Hour
)

type LifecyclePublisher struct {
	Client *redis.Client
	Stream string
}

func (p LifecyclePublisher) Publish(ctx context.Context, event lifecycle.Event) (string, error) {
	if p.Client == nil {
		return "", fmt.Errorf("redis client is required")
	}
	if err := event.Validate(); err != nil {
		return "", err
	}
	stream := p.Stream
	if stream == "" {
		stream = LifecycleStream
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("encode lifecycle event: %w", err)
	}
	id, err := p.Client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]any{"payload": payload},
	}).Result()
	if err != nil {
		return "", fmt.Errorf("publish lifecycle event: %w", err)
	}
	return id, nil
}
